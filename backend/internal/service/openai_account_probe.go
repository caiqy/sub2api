package service

import (
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
	probing         atomic.Bool
	errorPenalized  atomic.Bool
	ttftPenalized   atomic.Bool
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

// markPenalized 注册一个账号进入探活列表（幂等），并记录惩罚原因。
func (p *openAIAccountProbe) markPenalized(accountID int64, errorPenalized bool, ttftPenalized bool) {
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
	if errorPenalized {
		entry.errorPenalized.Store(true)
	}
	if ttftPenalized {
		entry.ttftPenalized.Store(true)
	}
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
				p.recoverAccount(accountID, entry)
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

	if result.err == nil {
		// 探活成功：上报 TTFT 到 EWMA（而不是直接重置），让 EWMA 自然回落
		ttft := result.ttftMs
		p.stats.report(account.ID, true, &ttft)

		// 检查更新后的 EWMA 是否仍然触发惩罚
		errorRate, ttftEWMA, hasTTFT := p.stats.snapshot(account.ID)
		stillPenalized := errorRate >= lcfg.ErrorPenaltyThreshold
		if !stillPenalized && hasTTFT {
			// 需要知道 minTTFT 才能判断 TTFT 是否仍超标。
			// 但探针无法获取其他账号的 TTFT（那是调度时才计算的）。
			// 所以这里用一个保守策略：如果 TTFT EWMA 降到了惩罚前探针记录的基线以下，
			// 则认为恢复。否则继续探测。
			// 简化处理：探针成功 + errorRate 正常 → 恢复。让调度器在下次选中时重新评估 TTFT。
			_ = ttftEWMA
		}

		if !stillPenalized {
			entry.consecutiveFail.Store(0)
			p.recoverAccount(account.ID, entry)
		} else {
			// errorRate 仍然超阈值，不恢复，但重置连续失败计数
			entry.consecutiveFail.Store(0)
			slog.Debug("probe succeeded but account still penalized",
				"account_id", account.ID,
				"error_rate", errorRate,
			)
		}
		return
	}

	// 探活失败
	fails := entry.consecutiveFail.Add(1)
	slog.Debug("probe failed",
		"account_id", account.ID,
		"consecutive_fail", fails,
		"error", result.err.Error(),
	)

	if int(fails) >= lcfg.ProbeMaxFailures {
		p.setTempUnschedulable(account.ID, entry)
	}
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
	ttftMs int // 首 token 延迟（毫秒），仅在成功时有效
}

// sendProbeRequest 发送轻量级探活请求，测量首 token 延迟。
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

	baseURL := account.GetOpenAIBaseURL()
	reqURL := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"

	body := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
		"max_tokens": probeMaxTokens,
		"stream":     false,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return probeResult{err: fmt.Errorf("marshal body: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return probeResult{err: fmt.Errorf("new request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	proxyURL := ""
	if account.Proxy != nil && account.Proxy.IsActive() {
		proxyURL = account.Proxy.URL()
	}

	start := time.Now()
	resp, err := p.service.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	ttftMs := int(time.Since(start).Milliseconds())
	if err != nil {
		return probeResult{err: fmt.Errorf("do request: %w", err)}
	}
	defer resp.Body.Close()
	// 读取并丢弃 body
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return probeResult{err: fmt.Errorf("upstream status %d", resp.StatusCode)}
	}
	return probeResult{ttftMs: ttftMs}
}

// recoverAccount 恢复被惩罚的账号：重置 EWMA 统计，清除 DB 标记，从 entry 列表移除。
func (p *openAIAccountProbe) recoverAccount(accountID int64, entry *openAIAccountProbeEntry) {
	if p.stats != nil {
		p.stats.resetAccount(accountID)
	}
	if entry.dbFlagSet.Load() && p.service != nil && p.service.accountRepo != nil {
		ctx := p.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		if err := p.service.accountRepo.ClearTempUnschedulable(ctx, accountID); err != nil {
			slog.Warn("probe: failed to clear temp unschedulable",
				"account_id", accountID,
				"error", err.Error(),
			)
			return
		}
		entry.dbFlagSet.Store(false)
	}
	p.entries.Delete(accountID)
	slog.Info("probe: account recovered", "account_id", accountID)
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
			"error", err.Error(),
		)
		return
	}
	entry.dbFlagSet.Store(true)
	slog.Warn("probe: account marked temp unschedulable", "account_id", accountID)
}
