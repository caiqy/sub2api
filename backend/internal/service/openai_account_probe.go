package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	probePermanentBlockDuration = 100 * 365 * 24 * time.Hour
	probeDefaultFallbackModel   = "gpt-4o-mini"
	probeMaxTokens              = 1
)

// openAIAccountProbeEntry 记录一个被惩罚账号的探活状态。
type openAIAccountProbeEntry struct {
	accountID       int64
	penalizedAt     time.Time
	stateMu         sync.Mutex
	consecutiveFail atomic.Int32
	dbFlagSet       atomic.Bool
	ignoreResults   atomic.Bool
	probing         atomic.Bool
	errorPenalized  atomic.Bool
	ttftPenalized   atomic.Bool
	groupIDValue    atomic.Int64
	groupIDSet      atomic.Bool
	lastProbeTTFTMs atomic.Int64
	lastProbeAtUnix atomic.Int64
}

// openAIAccountProbe 异步探活：对被分层调度器惩罚的账号发送轻量级请求，
// 判断其是否已恢复或需要被标记为临时不可调度。
type openAIAccountProbe struct {
	service    *OpenAIGatewayService
	stats      *openAIAccountRuntimeStats
	entries    sync.Map // key: int64(accountID), value: *openAIAccountProbeEntry
	dispatchMu sync.Mutex
	stopCh     chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	stopped    atomic.Bool
}

func newOpenAIAccountProbe(service *OpenAIGatewayService, stats *openAIAccountRuntimeStats) *openAIAccountProbe {
	ctx, cancel := context.WithCancel(context.Background())
	p := &openAIAccountProbe{
		service: service,
		stats:   stats,
		stopCh:  make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.loop()
	}()
	return p
}

// markPenalized 注册一个账号进入探活列表（幂等），并记录惩罚原因与分组上下文。
func (p *openAIAccountProbe) markPenalized(accountID int64, groupID *int64, errorPenalized bool, ttftPenalized bool) {
	if p == nil || accountID <= 0 || p.stopped.Load() {
		return
	}
	entryAny, _ := p.entries.LoadOrStore(accountID, &openAIAccountProbeEntry{
		accountID:   accountID,
		penalizedAt: time.Now(),
	})
	entry, _ := entryAny.(*openAIAccountProbeEntry)
	if entry == nil {
		return
	}
	entry.stateMu.Lock()
	defer entry.stateMu.Unlock()
	if groupID != nil && *groupID > 0 {
		entry.groupIDValue.Store(*groupID)
		entry.groupIDSet.Store(true)
	}
	entry.errorPenalized.Store(errorPenalized)
	entry.ttftPenalized.Store(ttftPenalized)
	if entry.penalizedAt.IsZero() {
		entry.penalizedAt = time.Now()
	}
}

// clearPenaltyReasons 清除 entry 的惩罚原因；仅在无 DB 标记且无探活进行中时移除 entry。
func (p *openAIAccountProbe) clearPenaltyReasons(accountID int64) {
	if p == nil || accountID <= 0 {
		return
	}
	value, ok := p.entries.Load(accountID)
	if !ok {
		return
	}
	entry, _ := value.(*openAIAccountProbeEntry)
	if entry == nil {
		p.entries.Delete(accountID)
		return
	}

	entry.stateMu.Lock()
	defer entry.stateMu.Unlock()

	entry.errorPenalized.Store(false)
	entry.ttftPenalized.Store(false)

	if entry.probing.Load() {
		return
	}
	if entry.dbFlagSet.Load() {
		return
	}
	p.entries.Delete(accountID)
}

// stop 停止探活 goroutine。
func (p *openAIAccountProbe) stop() {
	if p == nil {
		return
	}
	if p.stopped.CompareAndSwap(false, true) {
		p.dispatchMu.Lock()
		if p.cancel != nil {
			p.cancel()
		}
		close(p.stopCh)
		p.dispatchMu.Unlock()
		p.wg.Wait()
	}
}

// loop 定时执行探活逻辑。
func (p *openAIAccountProbe) loop() {
	if p == nil || p.service == nil {
		return
	}
	lcfg := p.service.openAIWSSchedulerLayeredConfig()
	interval := time.Duration(lcfg.ProbeIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.tick()
		}
	}
}

// tick 遍历所有 entry，按策略调度探活请求。
func (p *openAIAccountProbe) tick() {
	if p == nil || p.service == nil || p.stopped.Load() {
		return
	}
	lcfg := p.service.openAIWSSchedulerLayeredConfig()
	cooldown := time.Duration(lcfg.ProbeCooldownSeconds) * time.Second
	if cooldown <= 0 {
		cooldown = 60 * time.Second
	}
	now := time.Now()

	p.entries.Range(func(key, value any) bool {
		accountID, ok := key.(int64)
		if !ok {
			return true
		}
		entry, ok := value.(*openAIAccountProbeEntry)
		if !ok || entry == nil {
			p.entries.Delete(key)
			return true
		}

		// 检查账号是否还存在且为 OpenAI
		account, err := p.service.getSchedulableAccount(context.Background(), accountID)
		if err != nil || account == nil || !account.IsOpenAI() {
			p.entries.Delete(accountID)
			return true
		}
		// 如果账号已被管理员标记为不可调度，移除 entry
		if !account.Schedulable {
			p.entries.Delete(accountID)
			return true
		}
		// 如果 dbFlagSet 为 true 但 TempUnschedulableUntil 已被清除（管理员手动恢复），则恢复账号
		if entry.dbFlagSet.Load() {
			if account.TempUnschedulableUntil == nil || account.TempUnschedulableUntil.Before(now) {
				p.applyManualRecovery(accountID, entry)
				return true
			}
		}

		// 冷却期检查
		if now.Sub(entry.penalizedAt) < cooldown {
			return true
		}

		if p.stopped.Load() {
			return false
		}

		// single-flight：一个 entry 同一时间只有一个探活 goroutine
		if !entry.probing.CompareAndSwap(false, true) {
			return true
		}

		p.dispatchMu.Lock()
		defer p.dispatchMu.Unlock()
		if p.stopped.Load() {
			entry.probing.Store(false)
			return false
		}
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			p.probeAccount(account, entry, lcfg)
		}()
		return true
	})
}

// probeAccount 对单个账号执行一次探活请求。
func (p *openAIAccountProbe) probeAccount(account *Account, entry *openAIAccountProbeEntry, lcfg GatewayOpenAIWSSchedulerLayeredConfig) {
	defer entry.probing.Store(false)

	model := p.resolveProbeModel(account)
	result := p.sendProbeRequest(p.ctx, account, model, lcfg)
	entry.stateMu.Lock()
	defer entry.stateMu.Unlock()
	if entry != nil && entry.ignoreResults.Load() {
		return
	}

	if result.err == nil {
		ttft := result.ttftMs
		p.stats.report(account.ID, true, &ttft)
		p.updateProbeObservations(entry, ttft)

		groupID := probeEntryGroupID(entry)
		eval, evalErr := p.reevaluatePenaltyReasons(p.ctx, account.ID, groupID)
		if evalErr != nil {
			entry.consecutiveFail.Store(0)
			slog.Warn("probe: failed to reevaluate penalty reasons", "account_id", account.ID, "error", evalErr.Error())
			return
		}

		entry.errorPenalized.Store(eval.ErrorPenalized)
		entry.ttftPenalized.Store(eval.TTFTPenalized)
		entry.consecutiveFail.Store(0)
		groupMinTTFT := float64(0)
		if eval.HasGroupMin {
			groupMinTTFT = eval.GroupMinTTFT
		}
		slog.Info("probe succeeded", p.explainabilityFields(account.ID, entry, groupMinTTFT)...)
		p.finalizePenaltyState(account.ID, entry)
		return
	}

	// 探活失败
	fails := entry.consecutiveFail.Add(1)
	fields := p.explainabilityFields(account.ID, entry, 0)
	fields = append(fields, "consecutive_fail", fails, "error", result.err.Error())
	slog.Debug("probe failed", fields...)

	if int(fails) >= lcfg.ProbeMaxFailures {
		p.setTempUnschedulable(account.ID, entry)
	}
}

func (p *openAIAccountProbe) updateProbeObservations(entry *openAIAccountProbeEntry, ttft int) {
	if entry == nil {
		return
	}
	entry.lastProbeTTFTMs.Store(int64(ttft))
	entry.lastProbeAtUnix.Store(time.Now().Unix())
}

func probeAccountGroupID(account *Account) *int64 {
	if account == nil {
		return nil
	}
	if len(account.AccountGroups) > 0 {
		for _, ag := range account.AccountGroups {
			if ag.GroupID > 0 {
				gid := ag.GroupID
				return &gid
			}
		}
	}
	if len(account.GroupIDs) > 0 {
		for _, gid := range account.GroupIDs {
			if gid > 0 {
				groupID := gid
				return &groupID
			}
		}
	}
	if len(account.Groups) > 0 {
		for _, grp := range account.Groups {
			if grp != nil && grp.ID > 0 {
				gid := grp.ID
				return &gid
			}
		}
	}
	return nil
}

func probeEntryGroupID(entry *openAIAccountProbeEntry) *int64 {
	if entry == nil || !entry.groupIDSet.Load() {
		return nil
	}
	gid := entry.groupIDValue.Load()
	if gid <= 0 {
		return nil
	}
	groupID := gid
	return &groupID
}

func (p *openAIAccountProbe) reevaluatePenaltyReasons(ctx context.Context, accountID int64, groupID *int64) (layeredPenaltyEvaluation, error) {
	if p == nil || p.service == nil {
		return layeredPenaltyEvaluation{}, fmt.Errorf("nil probe service")
	}
	ls := &layeredOpenAIAccountScheduler{
		service: p.service,
		stats:   p.stats,
	}
	groupMinTTFT, hasGroupMin, err := ls.computeGroupMinTTFT(ctx, groupID)
	if err != nil {
		return layeredPenaltyEvaluation{}, err
	}
	return ls.evaluateRuntimePenalty(accountID, groupMinTTFT, hasGroupMin), nil
}

func (p *openAIAccountProbe) explainabilityFields(accountID int64, entry *openAIAccountProbeEntry, groupMinTTFT float64) []any {
	fields := []any{"account_id", accountID}
	if entry != nil {
		fields = append(fields,
			"error_penalized", entry.errorPenalized.Load(),
			"ttft_penalized", entry.ttftPenalized.Load(),
			"last_probe_ttft_ms", entry.lastProbeTTFTMs.Load(),
		)
	} else {
		fields = append(fields,
			"error_penalized", false,
			"ttft_penalized", false,
			"last_probe_ttft_ms", int64(0),
		)
	}

	if p != nil && p.stats != nil {
		errorRate, ttft, hasTTFT := p.stats.snapshot(accountID)
		fields = append(fields, "error_rate", errorRate)
		if hasTTFT {
			fields = append(fields, "ttft", ttft)
		} else {
			fields = append(fields, "ttft", float64(0))
		}
	} else {
		fields = append(fields, "error_rate", float64(0), "ttft", float64(0))
	}
	fields = append(fields, "group_min_ttft", groupMinTTFT)
	return fields
}

func (p *openAIAccountProbe) finalizePenaltyState(accountID int64, entry *openAIAccountProbeEntry) {
	if entry == nil {
		p.entries.Delete(accountID)
		return
	}
	if entry.errorPenalized.Load() || entry.ttftPenalized.Load() {
		return
	}
	p.recoverAccount(accountID, entry)
}

// resolveProbeModel 为探活请求选择模型。
// 优先使用 model_mapping 中第一个非通配符模型，回退到 gpt-4o-mini。
func (p *openAIAccountProbe) resolveProbeModel(account *Account) string {
	if account == nil {
		return probeDefaultFallbackModel
	}
	mapping := account.GetModelMapping()
	if len(mapping) > 0 {
		// 排序以保证确定性
		keys := make([]string, 0, len(mapping))
		for k := range mapping {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if k != "*" && strings.TrimSpace(k) != "" {
				return k
			}
		}
	}
	return probeDefaultFallbackModel
}

// probeResult 包含探活请求的结果。
type probeResult struct {
	err    error
	ttftMs int // 记录真实流式首个有效事件时间（TTFT）。
}

func buildOpenAIProbeResponsesURL(account *Account) string {
	if account != nil && account.IsOAuth() {
		return chatgptCodexURL
	}
	baseURL := ""
	if account != nil {
		baseURL = account.GetOpenAIBaseURL()
	}
	return buildOpenAIResponsesURL(baseURL)
}

func createOpenAIProbePayload(model string, isOAuth bool) ([]byte, error) {
	payload := map[string]any{
		"model": model,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": "hi",
					},
				},
			},
		},
		"stream":            true,
		"max_output_tokens": probeMaxTokens,
	}
	if isOAuth {
		payload["store"] = false
	}
	return json.Marshal(payload)
}

func parseOpenAIProbeEventPayload(eventType string, payload string) (bool, string, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return false, "", nil
	}
	if payload == "[DONE]" {
		return false, "", fmt.Errorf("probe stream received [DONE] before valid event")
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		if eventType == "error" {
			return false, "", fmt.Errorf("probe stream error event")
		}
		return false, "", nil
	}

	if eventType == "error" {
		if msg := extractOpenAIProbeErrorMessage(data); msg != "" {
			return false, "", fmt.Errorf("probe stream error event: %s", msg)
		}
		return false, "", fmt.Errorf("probe stream error event")
	}

	if errEvent := extractOpenAIProbeErrorMessage(data); errEvent != "" {
		return false, "", fmt.Errorf("probe stream error event: %s", errEvent)
	}
	if eventType, _ := data["type"].(string); eventType == "error" {
		return false, "", fmt.Errorf("probe stream error event")
	}
	if _, hasError := data["error"]; hasError {
		return false, "", fmt.Errorf("probe stream error event")
	}
	return true, payload, nil
}

func extractOpenAIProbeErrorMessage(data map[string]any) string {
	if data == nil {
		return ""
	}
	if msg, ok := data["message"].(string); ok {
		return strings.TrimSpace(msg)
	}
	if errData, ok := data["error"].(map[string]any); ok {
		if msg, ok := errData["message"].(string); ok {
			return strings.TrimSpace(msg)
		}
	}
	return ""
}

func readOpenAIProbeResponseStream(ctx context.Context, body io.Reader, start time.Time) (int, error) {
	reader := bufio.NewReader(body)
	var eventDataLines []string
	eventType := ""

	flushEvent := func() (int, bool, error) {
		if len(eventDataLines) == 0 {
			if eventType == "error" {
				eventType = ""
				return 0, false, fmt.Errorf("probe stream error event")
			}
			eventType = ""
			return 0, false, nil
		}
		payload := strings.Join(eventDataLines, "\n")
		eventDataLines = nil
		currentEventType := eventType
		eventType = ""
		valid, _, err := parseOpenAIProbeEventPayload(currentEventType, payload)
		if err != nil {
			return 0, false, err
		}
		if valid {
			return int(time.Since(start).Milliseconds()), true, nil
		}
		return 0, false, nil
	}

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine == "" {
				ttftMs, ok, flushErr := flushEvent()
				if flushErr != nil {
					return 0, flushErr
				}
				if ok {
					return ttftMs, nil
				}
			} else if sseDataPrefix.MatchString(trimmedLine) {
				eventDataLines = append(eventDataLines, sseDataPrefix.ReplaceAllString(trimmedLine, ""))
			} else if strings.HasPrefix(trimmedLine, "event:") {
				eventType = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "event:"))
			}
		}

		if err != nil {
			if err == io.EOF {
				ttftMs, ok, flushErr := flushEvent()
				if flushErr != nil {
					return 0, flushErr
				}
				if ok {
					return ttftMs, nil
				}
				return 0, fmt.Errorf("probe stream ended before valid event: %w", err)
			}
			if ctx != nil && ctx.Err() != nil {
				return 0, fmt.Errorf("probe stream read error before valid event: %w", ctx.Err())
			}
			return 0, fmt.Errorf("probe stream read error before valid event: %w", err)
		}
	}
}

// sendProbeRequest 发送轻量级探活请求，并记录真实流式首个有效事件时间。
func (p *openAIAccountProbe) sendProbeRequest(ctx context.Context, account *Account, model string, lcfg GatewayOpenAIWSSchedulerLayeredConfig) probeResult {
	if p.service == nil || account == nil {
		return probeResult{err: fmt.Errorf("nil service or account")}
	}

	timeout := time.Duration(lcfg.ProbeTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	token, _, err := p.service.GetAccessToken(ctx, account)
	if err != nil {
		return probeResult{err: fmt.Errorf("get access token: %w", err)}
	}

	isOAuth := account.IsOAuth()
	reqURL := buildOpenAIProbeResponsesURL(account)
	bodyBytes, err := createOpenAIProbePayload(model, isOAuth)
	if err != nil {
		return probeResult{err: fmt.Errorf("marshal body: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return probeResult{err: fmt.Errorf("new request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if isOAuth {
		req.Host = "chatgpt.com"
		req.Header.Set("accept", "text/event-stream")
		req.Header.Set("OpenAI-Beta", "responses=experimental")
		req.Header.Set("originator", "codex_cli_rs")
		customUA := account.GetOpenAIUserAgent()
		if customUA != "" {
			req.Header.Set("User-Agent", customUA)
		} else {
			req.Header.Set("User-Agent", codexCLIUserAgent)
		}
		if chatgptAccountID := account.GetChatGPTAccountID(); chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
	}

	proxyURL := ""
	if account.Proxy != nil && account.Proxy.IsActive() {
		proxyURL = account.Proxy.URL()
	}

	start := time.Now()
	resp, err := p.service.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return probeResult{err: fmt.Errorf("do request: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return probeResult{err: fmt.Errorf("upstream status %d", resp.StatusCode)}
	}

	ttftMs, err := readOpenAIProbeResponseStream(ctx, resp.Body, start)
	if err != nil {
		return probeResult{err: err}
	}
	return probeResult{ttftMs: ttftMs}
}

// recoverAccount 恢复被惩罚的账号：重置错误 EWMA、保留 TTFT EWMA，清除 DB 标记，并从 entry 列表移除。
func (p *openAIAccountProbe) recoverAccount(accountID int64, entry *openAIAccountProbeEntry) {
	lastProbeTTFTMs := int64(0)
	if entry != nil {
		lastProbeTTFTMs = entry.lastProbeTTFTMs.Load()
	}
	if p.stats != nil {
		p.stats.resetAccount(accountID)
	}
	if entry != nil && entry.dbFlagSet.Load() && p.service != nil && p.service.accountRepo != nil {
		ctx := p.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		if err := p.service.accountRepo.ClearTempUnschedulable(ctx, accountID); err != nil {
			slog.Warn("probe: failed to clear temp unschedulable",
				"account_id", accountID,
				"error_penalized", entry.errorPenalized.Load(),
				"ttft_penalized", entry.ttftPenalized.Load(),
				"last_probe_ttft_ms", lastProbeTTFTMs,
				"error", err.Error(),
			)
			return
		}
		entry.dbFlagSet.Store(false)
	}
	p.entries.Delete(accountID)
	if entry != nil {
		entry.errorPenalized.Store(false)
		entry.ttftPenalized.Store(false)
		entry.lastProbeTTFTMs.Store(lastProbeTTFTMs)
	}
	slog.Info("probe: account recovered", p.explainabilityFields(accountID, entry, 0)...)
}

// applyManualRecovery 执行管理员手动恢复：完整清除内存中的惩罚原因与分组上下文，
// 但仅重置错误 EWMA，故意保留 TTFT EWMA，让调度在人工干预后仍保有延迟证据。
func (p *openAIAccountProbe) applyManualRecovery(accountID int64, entry *openAIAccountProbeEntry) {
	prevError := false
	prevTTFT := false
	lastProbeTTFTMs := int64(0)
	if entry != nil {
		entry.stateMu.Lock()
		defer entry.stateMu.Unlock()
		entry.ignoreResults.Store(true)
		prevError = entry.errorPenalized.Load()
		prevTTFT = entry.ttftPenalized.Load()
		lastProbeTTFTMs = entry.lastProbeTTFTMs.Load()
		entry.errorPenalized.Store(false)
		entry.ttftPenalized.Store(false)
		entry.groupIDSet.Store(false)
		entry.groupIDValue.Store(0)
	}
	if p.service != nil && p.service.accountRepo != nil {
		ctx := p.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		if err := p.service.accountRepo.ClearTempUnschedulable(ctx, accountID); err != nil {
			slog.Warn("probe: manual recovery failed to clear temp unschedulable",
				"account_id", accountID,
				"prev_error_penalized", prevError,
				"prev_ttft_penalized", prevTTFT,
				"error_penalized", false,
				"ttft_penalized", false,
				"last_probe_ttft_ms", lastProbeTTFTMs,
				"error", err.Error(),
			)
		}
	}
	if p.stats != nil {
		p.stats.resetAccount(accountID)
	}
	p.entries.Delete(accountID)
	fields := p.explainabilityFields(accountID, entry, 0)
	fields = append(fields,
		"prev_error_penalized", prevError,
		"prev_ttft_penalized", prevTTFT,
	)
	slog.Info("probe: manual recovery applied", fields...)
}

// setTempUnschedulable 将账号标记为临时不可调度（100 年，等待探活或管理员恢复）。
func (p *openAIAccountProbe) setTempUnschedulable(accountID int64, entry *openAIAccountProbeEntry) {
	if p.service == nil || p.service.accountRepo == nil {
		return
	}
	until := time.Now().Add(probePermanentBlockDuration)
	reason := "layered scheduler probe: consecutive failures exceeded threshold"
	ctx := p.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := p.service.accountRepo.SetTempUnschedulable(ctx, accountID, until, reason); err != nil {
		slog.Warn("probe: failed to set temp unschedulable",
			"account_id", accountID,
			"error_penalized", entry.errorPenalized.Load(),
			"ttft_penalized", entry.ttftPenalized.Load(),
			"last_probe_ttft_ms", entry.lastProbeTTFTMs.Load(),
			"error", err.Error(),
		)
		return
	}
	entry.dbFlagSet.Store(true)
	slog.Warn("probe: account marked temp unschedulable", p.explainabilityFields(accountID, entry, 0)...)
}
