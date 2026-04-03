package service

import (
	"context"
	"errors"
	"strings"
	"time"
)

// layeredOpenAIAccountScheduler 分层调度器：使用确定性优先级过滤 + LRU 选择，
// 替代 defaultOpenAIAccountScheduler 的加权随机评分。
type layeredOpenAIAccountScheduler struct {
	service *OpenAIGatewayService
	metrics openAIAccountSchedulerMetrics
	stats   *openAIAccountRuntimeStats
	probe   *openAIAccountProbe
}

func newLayeredOpenAIAccountScheduler(service *OpenAIGatewayService, stats *openAIAccountRuntimeStats) *layeredOpenAIAccountScheduler {
	if stats == nil {
		stats = newOpenAIAccountRuntimeStats()
	}
	s := &layeredOpenAIAccountScheduler{service: service, stats: stats}
	s.probe = newOpenAIAccountProbe(service, stats)
	return s
}

// Select 按三层策略选择账号：
//  1. previous_response_id 粘连
//  2. session_hash 粘连
//  3. 分层过滤（核心算法）
func (s *layeredOpenAIAccountScheduler) Select(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	decision := OpenAIAccountScheduleDecision{}
	start := time.Now()
	defer func() {
		decision.LatencyMs = time.Since(start).Milliseconds()
		s.metrics.recordSelect(decision)
	}()

	if s.service != nil && s.service.openAIStickyEnabled() {
		// Layer 1: previous_response_id
		previousResponseID := strings.TrimSpace(req.PreviousResponseID)
		if previousResponseID != "" {
			selection, err := s.service.SelectAccountByPreviousResponseID(
				ctx,
				req.GroupID,
				previousResponseID,
				req.RequestedModel,
				req.ExcludedIDs,
			)
			if err != nil {
				return nil, decision, err
			}
			if selection != nil && selection.Account != nil {
				if !s.isAccountTransportCompatible(selection.Account, req.RequiredTransport) {
					selection = nil
				}
			}
			if selection != nil && selection.Account != nil {
				decision.Layer = openAIAccountScheduleLayerPreviousResponse
				decision.StickyPreviousHit = true
				decision.SelectedAccountID = selection.Account.ID
				decision.SelectedAccountType = selection.Account.Type
				if req.SessionHash != "" {
					_ = s.service.BindStickySession(ctx, req.GroupID, req.SessionHash, selection.Account.ID)
				}
				return selection, decision, nil
			}
		}

		// Layer 2: session_hash sticky
		selection, err := s.selectBySessionHash(ctx, req)
		if err != nil {
			return nil, decision, err
		}
		if selection != nil && selection.Account != nil {
			decision.Layer = openAIAccountScheduleLayerSessionSticky
			decision.StickySessionHit = true
			decision.SelectedAccountID = selection.Account.ID
			decision.SelectedAccountType = selection.Account.Type
			return selection, decision, nil
		}
	}

	// Layer 3: layered filter
	selection, candidateCount, loadSkew, err := s.selectByLayeredFilter(ctx, req)
	decision.Layer = openAIAccountScheduleLayerLoadBalance
	decision.CandidateCount = candidateCount
	decision.LoadSkew = loadSkew
	if err != nil {
		return nil, decision, err
	}
	if selection != nil && selection.Account != nil {
		decision.SelectedAccountID = selection.Account.ID
		decision.SelectedAccountType = selection.Account.Type
	}
	return selection, decision, nil
}

// selectBySessionHash 复用 defaultOpenAIAccountScheduler 的 session hash 粘连逻辑。
func (s *layeredOpenAIAccountScheduler) selectBySessionHash(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
) (*AccountSelectionResult, error) {
	sessionHash := strings.TrimSpace(req.SessionHash)
	if sessionHash == "" || s == nil || s.service == nil || s.service.cache == nil {
		return nil, nil
	}

	accountID := req.StickyAccountID
	if accountID <= 0 {
		var err error
		accountID, err = s.service.getStickySessionAccountID(ctx, req.GroupID, sessionHash)
		if err != nil || accountID <= 0 {
			return nil, nil
		}
	}
	if accountID <= 0 {
		return nil, nil
	}
	if req.ExcludedIDs != nil {
		if _, excluded := req.ExcludedIDs[accountID]; excluded {
			return nil, nil
		}
	}

	account, err := s.service.getSchedulableAccount(ctx, accountID)
	if err != nil || account == nil {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil
	}
	if shouldClearStickySession(account, req.RequestedModel) || !account.IsOpenAI() || !account.IsSchedulable() {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil
	}
	if req.RequestedModel != "" && !account.IsModelSupported(req.RequestedModel) {
		return nil, nil
	}
	if !s.isAccountTransportCompatible(account, req.RequiredTransport) {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil
	}
	account = s.service.recheckSelectedOpenAIAccountFromDB(ctx, account, req.RequestedModel)
	if account == nil {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, nil
	}

	result, acquireErr := s.service.tryAcquireAccountSlot(ctx, accountID, account.Concurrency)
	if acquireErr == nil && result.Acquired {
		_ = s.service.refreshStickySessionTTL(ctx, req.GroupID, sessionHash, s.service.openAIWSSessionStickyTTL())
		return &AccountSelectionResult{
			Account:     account,
			Acquired:    true,
			ReleaseFunc: result.ReleaseFunc,
		}, nil
	}

	cfg := s.service.schedulingConfig()
	if s.service.concurrencyService != nil {
		return &AccountSelectionResult{
			Account: account,
			WaitPlan: &AccountWaitPlan{
				AccountID:      accountID,
				MaxConcurrency: account.Concurrency,
				Timeout:        cfg.StickySessionWaitTimeout,
				MaxWaiting:     cfg.StickySessionMaxWaiting,
			},
		}, nil
	}
	return nil, nil
}

// selectByLayeredFilter 是分层调度器的核心算法：
//  1. 过滤候选（可调度、模型支持、传输协议兼容）
//  2. 批量加载 Redis 负载信息
//  3. 应用运行时惩罚（错误率 / TTFT）
//  4. 过滤 loadRate >= 100%
//  5. 循环：filterByMinPriority → filterByMinLoadRate → selectByLRU → tryAcquireSlot
//  6. 回退 WaitPlan
func (s *layeredOpenAIAccountScheduler) selectByLayeredFilter(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
) (*AccountSelectionResult, int, float64, error) {
	accounts, err := s.service.listSchedulableAccounts(ctx, req.GroupID)
	if err != nil {
		return nil, 0, 0, err
	}
	if len(accounts) == 0 {
		return nil, 0, 0, errors.New("no available OpenAI accounts")
	}

	// 1. 过滤候选
	filtered := make([]*Account, 0, len(accounts))
	loadReq := make([]AccountWithConcurrency, 0, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if req.ExcludedIDs != nil {
			if _, excluded := req.ExcludedIDs[account.ID]; excluded {
				continue
			}
		}
		if !account.IsSchedulable() || !account.IsOpenAI() {
			continue
		}
		if req.RequestedModel != "" && !account.IsModelSupported(req.RequestedModel) {
			continue
		}
		if !s.isAccountTransportCompatible(account, req.RequiredTransport) {
			continue
		}
		filtered = append(filtered, account)
		loadReq = append(loadReq, AccountWithConcurrency{
			ID:             account.ID,
			MaxConcurrency: account.EffectiveLoadFactor(),
		})
	}
	if len(filtered) == 0 {
		return nil, 0, 0, errors.New("no available OpenAI accounts")
	}

	// 2. 批量加载负载信息
	loadMap := map[int64]*AccountLoadInfo{}
	if s.service.concurrencyService != nil {
		if batchLoad, loadErr := s.service.concurrencyService.GetAccountsLoadBatch(ctx, loadReq); loadErr == nil {
			loadMap = batchLoad
		}
	}

	// 3. 构建候选列表并加载负载信息
	type candidateInfo struct {
		account  *Account
		loadInfo *AccountLoadInfo
	}
	candidates := make([]candidateInfo, 0, len(filtered))
	for _, account := range filtered {
		loadInfo := loadMap[account.ID]
		if loadInfo == nil {
			loadInfo = &AccountLoadInfo{AccountID: account.ID}
		}
		candidates = append(candidates, candidateInfo{
			account:  account,
			loadInfo: loadInfo,
		})
	}

	// 4. 应用运行时惩罚（使用 group-level 共享评估）并过滤满载候选
	groupMinTTFT, hasGroupMin, err := s.computeGroupMinTTFT(ctx, req.GroupID)
	if err != nil {
		hasGroupMin = false
		groupMinTTFT = 0
	}
	available := make([]accountWithLoad, 0, len(candidates))
	loadRateSum := 0.0
	loadRateSumSquares := 0.0

	for _, c := range candidates {
		eval := s.evaluateRuntimePenalty(c.account.ID, groupMinTTFT, hasGroupMin)
		acc := s.applyPenaltyToAccount(c.account, eval)

		if eval.ErrorPenalized || eval.TTFTPenalized {
			s.probe.markPenalized(c.account.ID, req.GroupID, eval.ErrorPenalized, eval.TTFTPenalized)
		} else {
			s.probe.clearPenaltyReasons(c.account.ID)
		}

		// 过滤 loadRate >= 100%
		if c.loadInfo.LoadRate >= 100 {
			continue
		}

		loadRate := float64(c.loadInfo.LoadRate)
		loadRateSum += loadRate
		loadRateSumSquares += loadRate * loadRate
		available = append(available, accountWithLoad{account: acc, loadInfo: c.loadInfo})
	}

	loadSkew := calcLoadSkewByMoments(loadRateSum, loadRateSumSquares, len(available))

	// 5. 循环选择
	for len(available) > 0 {
		step1 := filterByMinPriority(available)
		step2 := filterByMinLoadRate(step1)
		selected := selectByLRU(step2, false)
		if selected == nil {
			break
		}

		fresh := s.service.resolveFreshSchedulableOpenAIAccount(ctx, selected.account, req.RequestedModel)
		if fresh == nil || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) {
			available = removeFromAvailable(available, selected.account.ID)
			continue
		}
		fresh = s.service.recheckSelectedOpenAIAccountFromDB(ctx, fresh, req.RequestedModel)
		if fresh == nil || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) {
			available = removeFromAvailable(available, selected.account.ID)
			continue
		}

		result, acquireErr := s.service.tryAcquireAccountSlot(ctx, fresh.ID, fresh.Concurrency)
		if acquireErr != nil {
			return nil, len(candidates), loadSkew, acquireErr
		}
		if result != nil && result.Acquired {
			if req.SessionHash != "" {
				_ = s.service.BindStickySession(ctx, req.GroupID, req.SessionHash, fresh.ID)
			}
			return &AccountSelectionResult{
				Account:     fresh,
				Acquired:    true,
				ReleaseFunc: result.ReleaseFunc,
			}, len(candidates), loadSkew, nil
		}
		available = removeFromAvailable(available, selected.account.ID)
	}

	// 6. 回退 WaitPlan
	cfg := s.service.schedulingConfig()
	fallbackAccounts := make([]*Account, 0, len(filtered))
	for _, account := range filtered {
		fresh := s.service.resolveFreshSchedulableOpenAIAccount(ctx, account, req.RequestedModel)
		if fresh != nil && s.isAccountTransportCompatible(fresh, req.RequiredTransport) {
			fallbackAccounts = append(fallbackAccounts, fresh)
		}
	}
	sortAccountsByPriorityAndLastUsed(fallbackAccounts, false)
	for _, account := range fallbackAccounts {
		return &AccountSelectionResult{
			Account: account,
			WaitPlan: &AccountWaitPlan{
				AccountID:      account.ID,
				MaxConcurrency: account.Concurrency,
				Timeout:        cfg.FallbackWaitTimeout,
				MaxWaiting:     cfg.FallbackMaxWaiting,
			},
		}, len(candidates), loadSkew, nil
	}

	return nil, len(candidates), loadSkew, ErrNoAvailableAccounts
}

// layeredPenaltyEvaluation 封装一次运行时惩罚评估的结果。
// 调度器和探针共用同一评估逻辑，保证 TTFT 基线一致。
type layeredPenaltyEvaluation struct {
	ErrorPenalized bool
	TTFTPenalized  bool
	ErrorRate      float64
	TTFT           float64
	HasTTFT        bool
	GroupMinTTFT   float64
	HasGroupMin    bool
}

// computeGroupMinTTFT 计算 group-level 的最小 TTFT 基线，遍历该组所有可调度
// OpenAI 账号的运行时统计。调用者应在候选循环之前调用一次，避免重复查询。
func (s *layeredOpenAIAccountScheduler) computeGroupMinTTFT(ctx context.Context, groupID *int64) (float64, bool, error) {
	if s == nil || s.service == nil || s.stats == nil {
		return 0, false, nil
	}
	accounts, err := s.service.listSchedulableAccounts(ctx, groupID)
	if err != nil {
		return 0, false, err
	}
	var minTTFT float64
	var hasMin bool
	for i := range accounts {
		account := &accounts[i]
		if !account.IsSchedulable() || !account.IsOpenAI() {
			continue
		}
		_, ttft, hasTTFT := s.stats.snapshot(account.ID)
		if !hasTTFT || ttft <= 0 {
			continue
		}
		if !hasMin || ttft < minTTFT {
			minTTFT = ttft
			hasMin = true
		}
	}
	return minTTFT, hasMin, nil
}

// evaluateRuntimePenalty 基于预计算的 group-level 最小 TTFT 基线，
// 判断 accountID 是否需要被惩罚。不执行额外的数据库/缓存查询。
func (s *layeredOpenAIAccountScheduler) evaluateRuntimePenalty(accountID int64, groupMinTTFT float64, hasGroupMin bool) layeredPenaltyEvaluation {
	result := layeredPenaltyEvaluation{
		GroupMinTTFT: groupMinTTFT,
		HasGroupMin:  hasGroupMin,
	}
	if s == nil || s.stats == nil || accountID <= 0 {
		return result
	}
	result.ErrorRate, result.TTFT, result.HasTTFT = s.stats.snapshot(accountID)

	lcfg := s.service.openAIWSSchedulerLayeredConfig()
	result.ErrorPenalized = result.ErrorRate >= lcfg.ErrorPenaltyThreshold

	if result.HasTTFT && result.HasGroupMin && result.GroupMinTTFT > 0 {
		result.TTFTPenalized = result.TTFT >= result.GroupMinTTFT*lcfg.TTFTPenaltyMultiplier
	}
	return result
}

// applyPenaltyToAccount 根据评估结果对账号的 Priority 施加惩罚。
// 若有惩罚则返回浅拷贝（仅修改 Priority），否则返回原指针。
func (s *layeredOpenAIAccountScheduler) applyPenaltyToAccount(account *Account, eval layeredPenaltyEvaluation) *Account {
	if account == nil {
		return nil
	}
	if !eval.ErrorPenalized && !eval.TTFTPenalized {
		return account
	}
	// Shallow copy: only Priority is modified. Do NOT modify any pointer fields.
	copied := *account
	if eval.ErrorPenalized {
		copied.Priority += s.service.openAIWSSchedulerLayeredConfig().ErrorPenaltyValue
	}
	if eval.TTFTPenalized {
		copied.Priority += s.service.openAIWSSchedulerLayeredConfig().TTFTPenaltyValue
	}
	return &copied
}

// removeFromAvailable 从候选列表中移除指定 ID 的账号。
func removeFromAvailable(available []accountWithLoad, id int64) []accountWithLoad {
	result := make([]accountWithLoad, 0, len(available))
	for _, a := range available {
		if a.account.ID != id {
			result = append(result, a)
		}
	}
	return result
}

func (s *layeredOpenAIAccountScheduler) isAccountTransportCompatible(account *Account, requiredTransport OpenAIUpstreamTransport) bool {
	if requiredTransport == OpenAIUpstreamTransportAny || requiredTransport == OpenAIUpstreamTransportHTTPSSE {
		return true
	}
	if s == nil || s.service == nil || account == nil {
		return false
	}
	return s.service.getOpenAIWSProtocolResolver().Resolve(account).Transport == requiredTransport
}

func (s *layeredOpenAIAccountScheduler) ReportResult(accountID int64, success bool, firstTokenMs *int) {
	if s == nil || s.stats == nil {
		return
	}
	s.stats.report(accountID, success, firstTokenMs)
}

func (s *layeredOpenAIAccountScheduler) ReportSwitch() {
	if s == nil {
		return
	}
	s.metrics.recordSwitch()
}

func (s *layeredOpenAIAccountScheduler) SnapshotMetrics() OpenAIAccountSchedulerMetricsSnapshot {
	if s == nil {
		return OpenAIAccountSchedulerMetricsSnapshot{}
	}

	selectTotal := s.metrics.selectTotal.Load()
	prevHit := s.metrics.stickyPreviousHitTotal.Load()
	sessionHit := s.metrics.stickySessionHitTotal.Load()
	switchTotal := s.metrics.accountSwitchTotal.Load()
	latencyTotal := s.metrics.latencyMsTotal.Load()
	loadSkewTotal := s.metrics.loadSkewMilliTotal.Load()

	snapshot := OpenAIAccountSchedulerMetricsSnapshot{
		SelectTotal:              selectTotal,
		StickyPreviousHitTotal:   prevHit,
		StickySessionHitTotal:    sessionHit,
		LoadBalanceSelectTotal:   s.metrics.loadBalanceSelectTotal.Load(),
		AccountSwitchTotal:       switchTotal,
		SchedulerLatencyMsTotal:  latencyTotal,
		RuntimeStatsAccountCount: s.stats.size(),
	}
	if selectTotal > 0 {
		snapshot.SchedulerLatencyMsAvg = float64(latencyTotal) / float64(selectTotal)
		snapshot.StickyHitRatio = float64(prevHit+sessionHit) / float64(selectTotal)
		snapshot.AccountSwitchRate = float64(switchTotal) / float64(selectTotal)
		snapshot.LoadSkewAvg = float64(loadSkewTotal) / 1000 / float64(selectTotal)
	}
	return snapshot
}

// Stop 停止探活 goroutine。
func (s *layeredOpenAIAccountScheduler) Stop() {
	if s != nil && s.probe != nil {
		s.probe.stop()
	}
}
