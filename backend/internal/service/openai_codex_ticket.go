package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const (
	openAICodexTicketExtraKeyPrefix = "codex_turn_ticket:"
	// openAICodexTicketEnabledExtraKey 是账号级打票开关：默认关闭，需逐账号显式开启；
	// 全局总开关关闭时该键一律失效。
	openAICodexTicketEnabledExtraKey = "codex_ticket_enabled"
	openAICodexAstraMinVersion       = "0.153.4"
	openAICodexTicketStatePrefix     = "gAAAAA"
	openAICodexTicketDefaultModel    = "gpt-6-astra"
	openAICodexTicketDefaultSolModel = "gpt-5.6-sol"
)

// 打票未命中原因：进入日志与账号状态，避免“一直打不到票”只能靠翻日志。
const (
	openAICodexTicketMissNoProxy   = "proxy_not_configured"
	openAICodexTicketMissNoToken   = "token_unavailable"
	openAICodexTicketMissTransport = "transport_error"
	openAICodexTicketMissHTTP      = "http_status"
	openAICodexTicketMissEmpty     = "empty_state"
	openAICodexTicketMissLength    = "length_mismatch"
	openAICodexTicketMissPrefix    = "prefix_mismatch"
)

// openAICodexTicketAttempt 是某 (账号, 模型) 最近一次打票尝试的只读摘要。
// 只保留在内存：探针默认每 6 秒一轮，落库会把写放大成灾难。
type openAICodexTicketAttempt struct {
	At         time.Time
	Reason     string
	HTTPStatus int
	Length     int
}

// openAICodexTicketAttempts 的键与门票一致（账号 ID + 模型）。
// 包级存储是因为管理端账号状态读取方没有网关服务引用，而该摘要本身是纯观测数据。
var openAICodexTicketAttempts sync.Map

func recordOpenAICodexTicketAttempt(accountID int64, model, reason string, status, length int) {
	if accountID <= 0 {
		return
	}
	openAICodexTicketAttempts.Store(openAICodexTicketKey(accountID, model), openAICodexTicketAttempt{
		At:         time.Now(),
		Reason:     reason,
		HTTPStatus: status,
		Length:     length,
	})
}

func openAICodexTicketLastAttempt(accountID int64, model string) (openAICodexTicketAttempt, bool) {
	raw, ok := openAICodexTicketAttempts.Load(openAICodexTicketKey(accountID, model))
	if !ok {
		return openAICodexTicketAttempt{}, false
	}
	attempt, ok := raw.(openAICodexTicketAttempt)
	return attempt, ok
}

// ErrOpenAICodexTicketUnavailable 表示该号该模型没有可用的 292 门票，
// 且 fail_closed 禁止裸打业务请求。
var ErrOpenAICodexTicketUnavailable = errors.New("codex turn-state ticket unavailable")

type openAICodexTicket struct {
	AccountID  int64     `json:"account_id"`
	Model      string    `json:"model"`
	State      string    `json:"state"`
	Length     int       `json:"length"`
	CapturedAt time.Time `json:"captured_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Attempts   int       `json:"attempts"`
}

func openAICodexTicketKey(accountID int64, model string) string {
	return fmt.Sprintf("%d\x00%s", accountID, strings.TrimSpace(model))
}

func openAICodexTicketExtraKey(model string) string {
	return openAICodexTicketExtraKeyPrefix + strings.TrimSpace(model)
}

func normalizeOpenAICodexTicketModel(model string) string {
	return strings.TrimSpace(model)
}

func extractOpenAICodexTicketModel(body []byte) string {
	return normalizeOpenAICodexTicketModel(gjson.GetBytes(body, "model").String())
}

// OpenAICodexTicketRuntimeSettings 是后台可热更新的打票运行参数。
// 未被后台覆盖的项回退到 yaml/env 基线。
type OpenAICodexTicketRuntimeSettings struct {
	TargetLength         int
	TTLSeconds           int
	RefreshBeforeSeconds int
	ProbeIntervalSeconds int
	MaxConcurrentProbes  int
}

// openAICodexTicketConfig 合并 yaml/env 基线与后台热更新值。
// 热更新项：总开关、打票代理、目标长度、TTL、提前刷新、探测周期、探针并发上限。
// ctx 会传给后台设置读取，因此调用方取消时必须一并取消这里的读取。
func (s *OpenAIGatewayService) openAICodexTicketConfig(ctx context.Context) config.OpenAICodexTicketConfig {
	cfg := config.OpenAICodexTicketConfig{}
	if s != nil && s.cfg != nil {
		cfg = s.cfg.Gateway.OpenAICodexTicket
	}
	if cfg.TargetLength <= 0 {
		cfg.TargetLength = 292
	}
	if cfg.RefreshBeforeSeconds < 0 || (cfg.RefreshBeforeSeconds == 0 && cfg.TTLSeconds <= 0) {
		cfg.RefreshBeforeSeconds = 600
	}
	if cfg.TTLSeconds <= 0 {
		cfg.TTLSeconds = 3600
	}
	if cfg.HarvestProbeIntervalSeconds <= 0 {
		cfg.HarvestProbeIntervalSeconds = 6
	}
	if cfg.HarvestAttemptTimeoutSeconds <= 0 {
		cfg.HarvestAttemptTimeoutSeconds = 25
	}
	if cfg.MaxConcurrentProbes <= 0 {
		cfg.MaxConcurrentProbes = 8
	}
	if s != nil && s.settingService != nil {
		cfg = s.settingService.ApplyOpenAICodexTicketOverrides(ctx, cfg)
	}
	if len(cfg.Models) == 0 {
		cfg.Models = []string{openAICodexTicketDefaultModel, openAICodexTicketDefaultSolModel}
	}
	return cfg
}

// isOpenAICodexTicketParticipant 判断账号是否参与打票：资格符合且账号级开关显式开启。
// 账号级开关默认关闭，缺失或非布尔真值一律视为关闭。
func isOpenAICodexTicketParticipant(account *Account) bool {
	if !isOpenAICodexTicketAccount(account) || account.Status != StatusActive || account.Extra == nil {
		return false
	}
	enabled, ok := account.Extra[openAICodexTicketEnabledExtraKey].(bool)
	return ok && enabled
}

// openAICodexTicketActiveFor 汇总「全局开 + 账号开 + 模型受管」三项条件。
func (s *OpenAIGatewayService) openAICodexTicketActiveFor(ctx context.Context, account *Account, model string) bool {
	return s.openAICodexTicketEnabled(ctx) &&
		isOpenAICodexTicketParticipant(account) &&
		s.openAICodexTicketGatedModel(ctx, model)
}

func (s *OpenAIGatewayService) openAICodexTicketGatedModel(ctx context.Context, model string) bool {
	model = normalizeOpenAICodexTicketModel(model)
	if model == "" || !s.openAICodexTicketEnabled(ctx) {
		return false
	}
	for _, item := range s.openAICodexTicketConfig(ctx).Models {
		if normalizeOpenAICodexTicketModel(item) == model {
			return true
		}
	}
	return false
}

// OpenAICodexTicketStatus 是给管理端看的门票摘要，不含 state blob。
type OpenAICodexTicketStatus struct {
	Model            string     `json:"model"`
	Length           int        `json:"length,omitempty"`
	Ready            bool       `json:"ready"`
	RemainingSeconds int64      `json:"remaining_seconds"`
	Blocked          bool       `json:"blocked"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	// 最近一次打票尝试的只读摘要，用于定位“一直打不到票”。
	LastAttemptAt  *time.Time `json:"last_attempt_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	LastHTTPStatus int        `json:"last_http_status,omitempty"`
	LastLength     int        `json:"last_length,omitempty"`
}

func OpenAICodexTicketStatuses(account *Account, cfg config.OpenAICodexTicketConfig, now time.Time) []OpenAICodexTicketStatus {
	if !cfg.Enabled || !isOpenAICodexTicketParticipant(account) {
		return nil
	}
	models, targetLen := cfg.Models, cfg.TargetLength
	if len(models) == 0 {
		models = []string{openAICodexTicketDefaultModel, openAICodexTicketDefaultSolModel}
	}
	if targetLen <= 0 {
		targetLen = 292
	}
	out := make([]OpenAICodexTicketStatus, 0, len(models))
	for _, model := range models {
		model = normalizeOpenAICodexTicketModel(model)
		if model == "" {
			continue
		}
		status := OpenAICodexTicketStatus{Model: model}
		ticket := parseOpenAICodexTicketFromAny(0, model, nil)
		if account != nil && account.Extra != nil {
			ticket = parseOpenAICodexTicketFromAny(account.ID, model, account.Extra[openAICodexTicketExtraKey(model)])
		}
		if ticket.valid(now, targetLen) {
			status.Ready = true
			status.Length = ticket.Length
			remaining := int64(ticket.ExpiresAt.Sub(now) / time.Second)
			if remaining < 0 {
				remaining = 0
			}
			status.RemainingSeconds = remaining
			exp := ticket.ExpiresAt
			status.ExpiresAt = &exp
		}
		if account != nil && account.ID > 0 {
			if attempt, ok := openAICodexTicketLastAttempt(account.ID, model); ok {
				at := attempt.At
				status.LastAttemptAt = &at
				status.LastError = attempt.Reason
				status.LastHTTPStatus = attempt.HTTPStatus
				status.LastLength = attempt.Length
			}
		}
		status.Blocked = cfg.FailClosed && !status.Ready
		out = append(out, status)
	}
	return out
}

func (s *OpenAIGatewayService) openAICodexTicketEnabled(ctx context.Context) bool {
	if s == nil {
		return false
	}
	return s.openAICodexTicketConfig(ctx).Enabled
}

func (s *OpenAIGatewayService) openAICodexTicketHarvestProxyURL(ctx context.Context) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(s.openAICodexTicketConfig(ctx).HarvestProxyURL)
}

func (t *openAICodexTicket) valid(now time.Time, targetLen int) bool {
	if t == nil {
		return false
	}
	state := strings.TrimSpace(t.State)
	if len(state) != targetLen || t.Length != targetLen || !strings.HasPrefix(state, openAICodexTicketStatePrefix) {
		return false
	}
	if t.ExpiresAt.IsZero() || !now.Before(t.ExpiresAt) {
		return false
	}
	return true
}

func (t *openAICodexTicket) needsRefresh(now time.Time, refreshBefore time.Duration) bool {
	if t == nil || t.ExpiresAt.IsZero() {
		return true
	}
	return !t.ExpiresAt.After(now.Add(refreshBefore))
}

func (s *OpenAIGatewayService) lookupOpenAICodexTicket(ctx context.Context, account *Account, model string) *openAICodexTicket {
	if s == nil || account == nil || account.ID <= 0 {
		return nil
	}
	model = normalizeOpenAICodexTicketModel(model)
	if model == "" {
		return nil
	}
	key := openAICodexTicketKey(account.ID, model)
	targetLen := 292
	if s != nil {
		targetLen = s.openAICodexTicketConfig(ctx).TargetLength
	}
	now := time.Now()
	var mem *openAICodexTicket
	if raw, ok := s.openaiCodexTickets.Load(key); ok {
		mem, _ = raw.(*openAICodexTicket)
	}
	var extra *openAICodexTicket
	if account.Extra != nil {
		extra = parseOpenAICodexTicketFromAny(account.ID, model, account.Extra[openAICodexTicketExtraKey(model)])
	}
	if extra.valid(now, targetLen) && (mem == nil || extra.CapturedAt.After(mem.CapturedAt)) {
		s.openaiCodexTickets.Store(key, extra)
		return extra
	}
	if mem.valid(now, targetLen) {
		return mem
	}
	if extra != nil {
		s.openaiCodexTickets.Store(key, extra)
		return extra
	}
	if mem != nil {
		s.openaiCodexTickets.Delete(key)
	}
	return nil
}

func parseOpenAICodexTicketFromAny(accountID int64, model string, raw any) *openAICodexTicket {
	if raw == nil {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var ticket openAICodexTicket
	if err := json.Unmarshal(b, &ticket); err != nil {
		return nil
	}
	ticket.AccountID = accountID
	if strings.TrimSpace(model) != "" {
		ticket.Model = model
	}
	ticket.State = strings.TrimSpace(ticket.State)
	if ticket.Length == 0 {
		ticket.Length = len(ticket.State)
	}
	if ticket.State == "" {
		return nil
	}
	return &ticket
}

func (s *OpenAIGatewayService) storeOpenAICodexTicket(ctx context.Context, account *Account, ticket *openAICodexTicket) {
	if s == nil || account == nil || ticket == nil || account.ID <= 0 {
		return
	}
	model := normalizeOpenAICodexTicketModel(ticket.Model)
	ticket.Model = model
	ticket.AccountID = account.ID
	s.openaiCodexTickets.Store(openAICodexTicketKey(account.ID, model), ticket)
	if s.accountRepo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{
		openAICodexTicketExtraKey(model): ticket,
	}); err != nil {
		logger.L().Warn("openai_codex_ticket persist failed",
			zap.Int64("account_id", account.ID),
			zap.String("model", model),
			zap.Error(err),
		)
	}
}

// applyOpenAICodexTicket 在出站请求上覆盖 x-codex-turn-state。
// 请求路径只注入已捕获的有效门票，不现场打票；无票则返回
// ErrOpenAICodexTicketUnavailable。打票由后台 harvester 完成。
// 门控条件：全局总开关开启 + 账号级开关显式开启 + 出站模型受管。
func (s *OpenAIGatewayService) applyOpenAICodexTicket(ctx context.Context, account *Account, model string, h http.Header) error {
	if s == nil || h == nil {
		return nil
	}
	model = normalizeOpenAICodexTicketModel(model)
	if model == "" || !s.openAICodexTicketActiveFor(ctx, account, model) {
		return nil
	}
	cfg := s.openAICodexTicketConfig(ctx)
	if strings.TrimSpace(cfg.HarvestProxyURL) == "" {
		s.recordOpenAICodexTicketMiss(account, model, openAICodexTicketMissNoProxy, 0, 0)
		if cfg.FailClosed {
			return ErrOpenAICodexTicketUnavailable
		}
		return nil
	}
	ticket := s.lookupOpenAICodexTicket(ctx, account, model)
	if ticket.valid(time.Now(), cfg.TargetLength) {
		h.Set(openAICodexTurnStateHeader, ticket.State)
		return nil
	}
	if !cfg.FailClosed {
		return nil
	}
	return ErrOpenAICodexTicketUnavailable
}

// openAICodexTicketOutboundModel 预测本请求真正出站的模型名，也就是
// applyOpenAICodexTicket 注入时读到的 body.model。
//
// 调度门控与注入必须按同一个模型名判定门票。普通请求下二者同源：Forward 的
// upstreamModel 与本函数都走 resolveOpenAIAccountUpstreamModelForRequest，且
// Forward 会把 body.model 改写成该值后才注入。但 /responses/compact 例外——
// Forward 会把出站模型进一步改写为 compact 映射或 gateway.openai_compact_model
// （默认非空），此时若门控仍按客户端原始模型判定，就会把「实际出站是非门控
// 模型、根本不需要票」的 compact 请求整片误拦成不可调度。
func (s *OpenAIGatewayService) openAICodexTicketOutboundModel(account *Account, requestedModel string, requireCompact bool) string {
	model := strings.TrimSpace(requestedModel)
	if account == nil || model == "" {
		return model
	}
	if !account.IsOpenAI() {
		return canonicalOpenAIAccountSchedulingModel(account, model)
	}
	_, upstreamModel := resolveOpenAIForwardMappedModels(account, model, requireCompact)
	if requireCompact {
		// 与 Forward 同序：compact 兜底模型优先于普通/compact 映射结果。
		if compactModel := strings.TrimSpace(s.resolveOpenAICompactFallbackModel(account, model)); compactModel != "" {
			upstreamModel = compactModel
		}
	}
	if upstreamModel = strings.TrimSpace(upstreamModel); upstreamModel != "" {
		return upstreamModel
	}
	return model
}

// outboundModel 必须是真正会发给上游的模型名（openAICodexTicketOutboundModel），
// 不是客户端原始模型：注入侧读的是出站 body.model，两侧口径必须一致。
// 账号级开关关闭的账号不参与门控：关闭开关即立刻恢复调度。
func (s *OpenAIGatewayService) openAICodexTicketBlocksAccount(ctx context.Context, account *Account, outboundModel string) bool {
	if s == nil || !s.openAICodexTicketEnabled(ctx) || !isOpenAICodexTicketParticipant(account) {
		return false
	}
	cfg := s.openAICodexTicketConfig(ctx)
	if !cfg.FailClosed {
		return false
	}
	model := normalizeOpenAICodexTicketModel(outboundModel)
	if !s.openAICodexTicketGatedModel(ctx, model) {
		return false
	}
	ticket := s.lookupOpenAICodexTicket(ctx, account, model)
	return !ticket.valid(time.Now(), cfg.TargetLength)
}

func (s *OpenAIGatewayService) fireOpenAICodexTicketProbe(ctx context.Context, account *Account, token, model, proxyURL string, attemptTimeout time.Duration) (state string, status int, err error) {
	attemptCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
	defer cancel()

	body := []byte(`{"model":` + jsonString(model) + `,"store":false,"stream":true,"instructions":"Reply with exactly: pong","input":[{"role":"user","content":[{"type":"input_text","text":"ping"}]}]}`)
	req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, chatgptCodexURL, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAIHarvest))
	req.Close = true
	req.Host = "chatgpt.com"
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Beta", "responses=experimental")
	req.Header.Set("session_id", uuid.NewString())
	if err := resolveAndSetOpenAIChatGPTAccountHeaders(attemptCtx, s.accountRepo, req.Header, account); err != nil {
		return "", 0, err
	}
	applyOpenAICodexTicketHarvestIdentity(req.Header, model)

	// Synthetic probes must use the dedicated no-reuse transport even when the
	// production account is bound to a plugin. This also avoids reading pluginManager
	// while handlers are still wiring it during gateway construction.
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return "", 0, err
	}
	if resp == nil {
		return "", 0, errors.New("nil upstream response")
	}
	// Only the response header is needed; no connection will be reused.
	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	return extractOpenAICodexTurnState(resp.Header), resp.StatusCode, nil
}

func jsonString(v string) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `""`
	}
	return string(b)
}

func applyOpenAICodexTicketHarvestIdentity(h http.Header, model string) {
	ensureCodexIdentityHeaders(h)
	enforceCodexIdentityHeaders(h)
	version := strings.TrimSpace(h.Get("version"))
	if needsOpenAICodexAstraVersion(model) && (version == "" || CompareVersions(version, openAICodexAstraMinVersion) < 0) {
		h.Set("version", openAICodexAstraMinVersion)
		h.Set("user-agent", buildCodexCLIUserAgent(openAICodexAstraMinVersion))
		h.Set("originator", openai.CodexDefaultOriginator)
	}
}

func needsOpenAICodexAstraVersion(model string) bool {
	m := strings.ToLower(normalizeOpenAICodexTicketModel(model))
	return strings.Contains(m, "gpt-6") || strings.Contains(m, "astra")
}

func (s *OpenAIGatewayService) StartOpenAICodexTicketHarvester() {
	if s == nil {
		return
	}
	s.openaiCodexTicketLifecycleMu.Lock()
	defer s.openaiCodexTicketLifecycleMu.Unlock()
	if s.openaiCodexTicketStopped || s.openaiCodexTicketDone != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.openaiCodexTicketCancel = cancel
	s.openaiCodexTicketDone = done
	go func() {
		defer close(done)
		s.openAICodexTicketHarvestLoop(ctx)
	}()
	logger.L().Info("openai_codex_ticket harvester started",
		zap.Int("ttl_seconds", s.openAICodexTicketConfig(context.Background()).TTLSeconds),
		zap.Int("target_length", s.openAICodexTicketConfig(context.Background()).TargetLength),
		zap.Strings("models", s.openAICodexTicketConfig(context.Background()).Models),
	)
}

func (s *OpenAIGatewayService) StopOpenAICodexTicketHarvester() {
	if s == nil {
		return
	}
	s.openaiCodexTicketLifecycleMu.Lock()
	s.openaiCodexTicketStopped = true
	cancel, done := s.openaiCodexTicketCancel, s.openaiCodexTicketDone
	s.openaiCodexTicketLifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (s *OpenAIGatewayService) openAICodexTicketHarvestLoop(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.refreshOpenAICodexTickets(ctx)
			timer.Reset(time.Duration(s.openAICodexTicketConfig(ctx).HarvestProbeIntervalSeconds) * time.Second)
		}
	}
}

// Each cycle attempts at most one concurrency-sized batch. Rotate through targets
// so persistent failures cannot starve later accounts or models.
func (s *OpenAIGatewayService) refreshOpenAICodexTickets(ctx context.Context) {
	if s == nil || s.accountRepo == nil || ctx.Err() != nil || !s.openAICodexTicketEnabled(ctx) {
		return
	}
	if !s.openaiCodexTicketRefreshMu.TryLock() {
		return
	}
	defer s.openaiCodexTicketRefreshMu.Unlock()
	accounts, err := s.accountRepo.ListByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		logger.L().Warn("openai_codex_ticket list accounts failed", zap.Error(err))
		return
	}
	cfg := s.openAICodexTicketConfig(ctx)
	now := time.Now()
	refreshBefore := time.Duration(cfg.RefreshBeforeSeconds) * time.Second
	var wg sync.WaitGroup
	probed := 0
	total := len(accounts) * len(cfg.Models)
	for scanned := 0; scanned < total && probed < cfg.MaxConcurrentProbes && ctx.Err() == nil; scanned++ {
		index := s.openaiCodexTicketCursor % total
		s.openaiCodexTicketCursor = (index + 1) % total
		account := accounts[index/len(cfg.Models)]
		if !isOpenAICodexTicketParticipant(&account) {
			continue
		}
		model := normalizeOpenAICodexTicketModel(cfg.Models[index%len(cfg.Models)])
		if model == "" {
			continue
		}
		if t := s.lookupOpenAICodexTicket(ctx, &account, model); t.valid(now, cfg.TargetLength) && !t.needsRefresh(now, refreshBefore) {
			continue
		}
		// Token/header helpers may update account metadata; each model owns its maps.
		account.Extra = maps.Clone(account.Extra)
		account.Credentials = maps.Clone(account.Credentials)
		probed++
		wg.Add(1)
		go func(acc Account, model string) {
			defer wg.Done()
			s.probeOnceOpenAICodexTicket(ctx, &acc, model)
		}(account, model)
	}
	wg.Wait()
	if probed > 0 {
		logger.L().Info("openai_codex_ticket probe cycle",
			zap.Int("probed", probed),
			zap.Int("max_concurrent", cfg.MaxConcurrentProbes),
		)
	}
}

// probeOnceOpenAICodexTicket 走打票代理打一发。命中合格 292（HTTP 200、长度==target、
// gAAAAA 前缀）就落库；否则记录未命中原因，交给下个周期重试。同一 key 并发去重，避免上一发还没
// 回来又叠一发。
func (s *OpenAIGatewayService) probeOnceOpenAICodexTicket(ctx context.Context, account *Account, model string) {
	if s == nil || !isOpenAICodexTicketParticipant(account) || ctx.Err() != nil {
		return
	}
	cfg := s.openAICodexTicketConfig(ctx)
	if !cfg.Enabled {
		return
	}
	proxyURL := s.openAICodexTicketHarvestProxyURL(ctx)
	if proxyURL == "" || s.httpUpstream == nil || ctx.Err() != nil {
		s.recordOpenAICodexTicketMiss(account, model, openAICodexTicketMissNoProxy, 0, 0)
		return
	}
	key := openAICodexTicketKey(account.ID, model)
	_, _, _ = s.openaiCodexTicketFlight.Do(key, func() (any, error) {
		// Keep existing probes counted across limit changes; never queue for capacity.
		s.openaiCodexTicketProbeMu.Lock()
		if s.openaiCodexTicketInFlight >= cfg.MaxConcurrentProbes {
			s.openaiCodexTicketProbeMu.Unlock()
			return nil, nil
		}
		s.openaiCodexTicketInFlight++
		s.openaiCodexTicketProbeMu.Unlock()
		defer func() {
			s.openaiCodexTicketProbeMu.Lock()
			s.openaiCodexTicketInFlight--
			s.openaiCodexTicketProbeMu.Unlock()
		}()
		token, _, err := s.GetAccessToken(ctx, account)
		if err != nil || strings.TrimSpace(token) == "" {
			s.recordOpenAICodexTicketMiss(account, model, openAICodexTicketMissNoToken, 0, 0)
			return nil, nil
		}
		state, status, perr := s.fireOpenAICodexTicketProbe(ctx, account, token, model, proxyURL, time.Duration(cfg.HarvestAttemptTimeoutSeconds)*time.Second)
		if perr != nil {
			s.recordOpenAICodexTicketMiss(account, model, openAICodexTicketMissTransport, 0, 0)
			return nil, nil
		}
		if reason := openAICodexTicketStateRejectReason(state, status, cfg.TargetLength); reason != "" {
			s.recordOpenAICodexTicketMiss(account, model, reason, status, len(state))
			return nil, nil
		}
		now := time.Now()
		ticket := &openAICodexTicket{
			AccountID:  account.ID,
			Model:      model,
			State:      state,
			Length:     len(state),
			CapturedAt: now,
			ExpiresAt:  now.Add(time.Duration(cfg.TTLSeconds) * time.Second),
			Attempts:   1,
		}
		s.storeOpenAICodexTicket(ctx, account, ticket)
		recordOpenAICodexTicketAttempt(account.ID, model, "", status, ticket.Length)
		logger.L().Info("openai_codex_ticket harvested",
			zap.Int64("account_id", account.ID), zap.String("model", model),
			zap.Int("length", ticket.Length), zap.String("mode", "continuous"))
		return nil, nil
	})
}

// openAICodexTicketStateRejectReason 归一回传入状态的门票判定结果，空串表示合格。
func openAICodexTicketStateRejectReason(state string, status, targetLength int) string {
	if status != http.StatusOK {
		return openAICodexTicketMissHTTP
	}
	if state == "" {
		return openAICodexTicketMissEmpty
	}
	if len(state) != targetLength {
		return openAICodexTicketMissLength
	}
	if !strings.HasPrefix(state, openAICodexTicketStatePrefix) {
		return openAICodexTicketMissPrefix
	}
	return ""
}

// recordOpenAICodexTicketMiss 统一记录未命中原因：既是日志，也是账号状态接口的数据来源。
func (s *OpenAIGatewayService) recordOpenAICodexTicketMiss(account *Account, model, reason string, status, length int) {
	if account == nil {
		return
	}
	recordOpenAICodexTicketAttempt(account.ID, model, reason, status, length)
	fields := []zap.Field{
		zap.Int64("account_id", account.ID),
		zap.String("model", model),
		zap.String("reason", reason),
		zap.Int("http", status),
		zap.Int("len", length),
	}
	// Raw token/transport errors may contain credentials; log only classified fields.
	logger.L().Info("openai_codex_ticket probe miss", fields...)
}

// IsOpenAICodexTicketExtraKey identifies server-managed ticket material.
func IsOpenAICodexTicketExtraKey(key string) bool {
	return strings.HasPrefix(key, openAICodexTicketExtraKeyPrefix)
}

// 打票运行参数边界：既挡住明显笔误，也避免把探测打成洪水。
// 这些边界由后台设置写入路径（setting_update.go）逐项执行。
const (
	openAICodexTicketMinTargetLength  = 32
	openAICodexTicketMaxTargetLength  = 4096
	openAICodexTicketMinTTLSeconds    = 60
	openAICodexTicketMaxTTLSeconds    = 86400
	openAICodexTicketMaxRefreshBefore = 43200
	openAICodexTicketMinProbeInterval = 1
	openAICodexTicketMaxProbeInterval = 3600
	openAICodexTicketMinConcurrency   = 1
	openAICodexTicketMaxConcurrency   = 256
)

// MergeOpenAICodexTicketExtra preserves only persisted tickets, never summaries or
// blobs supplied by an account edit. The repository repeats this under the row
// lock so a concurrent harvest cannot be overwritten by a stale admin snapshot.
func MergeOpenAICodexTicketExtra(extra, current map[string]any) map[string]any {
	result := maps.Clone(extra)
	for key := range result {
		if IsOpenAICodexTicketPrivateExtraKey(key) {
			delete(result, key)
		}
	}
	for key, value := range current {
		if IsOpenAICodexTicketExtraKey(key) {
			if result == nil {
				result = make(map[string]any)
			}
			result[key] = value
		}
	}
	return result
}

// ValidateOpenAICodexTicketHarvestProxyURL validates only syntax, without making
// a network request or including credentials in validation errors.
func ValidateOpenAICodexTicketHarvestProxyURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return errors.New("harvest proxy must be an HTTP(S) or SOCKS5(h) URL with a host and no path, query or fragment")
	}
	switch parsed.Scheme {
	case "http", "https", "socks5", "socks5h":
	default:
		return errors.New("harvest proxy scheme must be http, https, socks5 or socks5h")
	}
	if port := parsed.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return errors.New("harvest proxy port must be between 1 and 65535")
		}
	}
	return nil
}

// MaskProxyURL never returns a stored proxy password, even for invalid legacy data.
func MaskProxyURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || ValidateOpenAICodexTicketHarvestProxyURL(raw) != nil {
		return ""
	}
	parsed, _ := url.Parse(raw)
	if parsed.User != nil {
		if _, ok := parsed.User.Password(); ok {
			parsed.User = url.UserPassword(parsed.User.Username(), "***")
		}
	}
	return parsed.String()
}

// IsMaskedProxyURL recognizes the exact password placeholder emitted by the API.
func IsMaskedProxyURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User == nil {
		return false
	}
	password, ok := parsed.User.Password()
	return ok && password == "***"
}

// Credential shadows do not own tickets. Keep their existing forwarding policy
// instead of imposing a gate for a key the harvester never populates.
func isOpenAICodexTicketAccount(account *Account) bool {
	return account != nil && account.IsOpenAIOAuthLike() && !account.IsShadow()
}

// IsOpenAICodexTicketPrivateExtraKey also covers the retired account-level proxy
// override, whose credentials may remain in older account records.
func IsOpenAICodexTicketPrivateExtraKey(key string) bool {
	return IsOpenAICodexTicketExtraKey(key) || key == "codex_harvest_proxy_url"
}

// RedactOpenAICodexTicketExtra strips ephemeral ticket material from exports
// without changing the source account or unrelated backup fields.
func RedactOpenAICodexTicketExtra(extra map[string]any) map[string]any {
	redacted := maps.Clone(extra)
	for key := range redacted {
		if IsOpenAICodexTicketPrivateExtraKey(key) {
			delete(redacted, key)
		}
	}
	return redacted
}
