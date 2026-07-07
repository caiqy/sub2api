package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const layeredSchedulerStartupRehydrateTimeout = 2 * time.Second

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
	schedGroup := schedulerGroupForRequest(ctx, s.service, req.GroupID)
	start := time.Now()
	defer func() {
		decision.LatencyMs = time.Since(start).Milliseconds()
		s.metrics.recordSelect(decision)
	}()

	if s.service != nil && s.service.openAIStickyEnabled() {
		// Layer 1: previous_response_id
		previousResponseID := strings.TrimSpace(req.PreviousResponseID)
		if normalizeOpenAICompatiblePlatform(req.Platform) != PlatformOpenAI {
			previousResponseID = ""
		}
		if previousResponseID != "" && (!req.StickyWeighted || !req.PreviousResponseCanMove) {
			selection, err := s.service.selectAccountByPreviousResponseIDForCapability(
				ctx,
				req.GroupID,
				previousResponseID,
				req.RequestedModel,
				req.ExcludedIDs,
				req.RequiredCapability,
				req.RequireCompact,
			)
			if err != nil {
				return nil, decision, err
			}
			if selection != nil && selection.Account != nil {
				if !s.isAccountTransportCompatible(selection.Account, req.RequiredTransport) {
					s.deletePreviousResponseStickyForRequest(ctx, req)
					if selection.ReleaseFunc != nil {
						selection.ReleaseFunc()
					}
					selection = nil
				}
			}
			if selection != nil && selection.Account != nil {
				if !s.isAccountRequestCompatible(ctx, selection.Account, req) ||
					!accountSatisfiesPrivacyRequirement(selection.Account, schedGroup) {
					s.deletePreviousResponseStickyForRequest(ctx, req)
					recordPrivacyRequirementError(ctx, s.service, selection.Account, schedGroup)
					if selection.ReleaseFunc != nil {
						selection.ReleaseFunc()
					}
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
		if !req.StickyWeighted {
			selection, updatedReq, err := s.selectBySessionHash(ctx, req)
			req = updatedReq
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
			if req.SkipStickyBind {
				req.PreserveStickyBinding = true
			}
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
		if req.StickyWeighted {
			if req.StickyPreviousAccountID > 0 && selection.Account.ID == req.StickyPreviousAccountID {
				decision.StickyPreviousHit = true
			}
			if req.StickyAccountID > 0 && selection.Account.ID == req.StickyAccountID {
				decision.StickySessionHit = true
			}
		}
	}
	return selection, decision, nil
}

// selectBySessionHash handles layered scheduler session sticky selection.
// Request-scoped incompatibility falls back without deleting sticky; binding-level
// invalidation (for example missing account, transport/privacy/upstream channel
// restriction) clears it. DB recheck model/image/compact mismatches are treated
// as request-scoped fallback.
func (s *layeredOpenAIAccountScheduler) selectBySessionHash(
	ctx context.Context,
	req OpenAIAccountScheduleRequest,
) (*AccountSelectionResult, OpenAIAccountScheduleRequest, error) {
	sessionHash := strings.TrimSpace(req.SessionHash)
	schedGroup := schedulerGroupForRequest(ctx, s.service, req.GroupID)
	if sessionHash == "" || s == nil || s.service == nil || s.service.cache == nil {
		return nil, req, nil
	}

	accountID := req.StickyAccountID
	if accountID <= 0 {
		var err error
		accountID, err = s.service.getStickySessionAccountID(ctx, req.GroupID, sessionHash)
		if err != nil || accountID <= 0 {
			return nil, req, nil
		}
	}
	if accountID <= 0 {
		return nil, req, nil
	}
	if req.ExcludedIDs != nil {
		if _, excluded := req.ExcludedIDs[accountID]; excluded {
			return nil, req, nil
		}
	}

	account, err := s.service.getSchedulableAccount(ctx, accountID)
	if err != nil || account == nil {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, req, nil
	}
	if shouldClearStickySession(account, req.RequestedModel) || !account.IsOpenAI() || !account.IsSchedulable() {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, req, nil
	}
	if schedGroup != nil && schedGroup.RequirePrivacySet && !account.IsPrivacySet() {
		_ = s.service.accountRepo.SetError(ctx, account.ID,
			fmt.Sprintf("Privacy not set, required by group [%s]", schedGroup.Name))
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, req, nil
	}
	if !s.isAccountRequestCompatible(ctx, account, req) {
		if s.isAccountUpstreamRestrictedByChannel(ctx, account, req) {
			_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		} else {
			req.SkipStickyBind = true
		}
		return nil, req, nil
	}
	if !s.isAccountTransportCompatible(account, req.RequiredTransport) {
		_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		return nil, req, nil
	}
	var clearSticky bool
	account, clearSticky = s.recheckSessionStickyAccountFromDB(ctx, account, req, schedGroup)
	if account == nil {
		if clearSticky {
			_ = s.service.deleteStickySessionAccountID(ctx, req.GroupID, sessionHash)
		} else {
			req.SkipStickyBind = true
		}
		return nil, req, nil
	}
	if reason, errorRate, ttft, shouldEscape := s.shouldEscapeStickyAccount(accountID); shouldEscape {
		slog.Info("sticky_escape_triggered",
			"account_id", accountID,
			"reason", reason,
			"error_rate", errorRate,
			"ttft", ttft,
		)
		req.SkipStickyBind = true
		return nil, req, nil
	}

	result, acquireErr := s.service.tryAcquireAccountSlot(ctx, accountID, account.Concurrency)
	if acquireErr == nil && result != nil && result.Acquired {
		_ = s.service.refreshStickySessionTTL(ctx, req.GroupID, sessionHash, s.service.openAIWSSessionStickyTTL())
		return &AccountSelectionResult{
			Account:     account,
			Acquired:    true,
			ReleaseFunc: result.ReleaseFunc,
		}, req, nil
	}

	cfg := s.service.schedulingConfig()
	if s.shouldEscapeBusyStickyAccount(accountID, account.Concurrency, cfg) {
		if req.ExcludedIDs == nil {
			req.ExcludedIDs = make(map[int64]struct{})
		}
		req.ExcludedIDs[accountID] = struct{}{}
		req.SkipStickyBind = true
		return nil, req, nil
	}
	if s.service.concurrencyService != nil {
		return &AccountSelectionResult{
			Account: account,
			WaitPlan: &AccountWaitPlan{
				AccountID:      accountID,
				MaxConcurrency: account.Concurrency,
				Timeout:        cfg.StickySessionWaitTimeout,
				MaxWaiting:     cfg.StickySessionMaxWaiting,
			},
		}, req, nil
	}
	return nil, req, nil
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
	schedGroup := schedulerGroupForRequest(ctx, s.service, req.GroupID)
	accounts, err := s.service.listSchedulableAccounts(ctx, req.GroupID, req.Platform)
	if err != nil {
		return nil, 0, 0, err
	}
	if len(accounts) == 0 {
		return nil, 0, 0, noAvailableOpenAISelectionError(req.RequestedModel, false)
	}

	// 1. 过滤候选
	filtered := make([]*Account, 0, len(accounts))
	loadReq := make([]AccountWithConcurrency, 0, len(accounts))
	compactBlocked := false
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
		if schedGroup != nil && schedGroup.RequirePrivacySet && !account.IsPrivacySet() {
			s.service.BlockAccountScheduling(account, time.Time{}, "privacy_not_set")
			_ = s.service.accountRepo.SetError(ctx, account.ID,
				fmt.Sprintf("Privacy not set, required by group [%s]", schedGroup.Name))
			continue
		}
		if !s.isAccountRequestCompatible(ctx, account, req) {
			continue
		}
		if !s.isAccountTransportCompatible(account, req.RequiredTransport) {
			continue
		}
		if req.RequireCompact && openAICompactSupportTier(account) == 0 {
			compactBlocked = true
			continue
		}
		filtered = append(filtered, account)
		loadReq = append(loadReq, AccountWithConcurrency{
			ID:             account.ID,
			MaxConcurrency: account.EffectiveLoadFactor(),
		})
	}
	if len(filtered) == 0 {
		return nil, 0, 0, noAvailableOpenAISelectionError(req.RequestedModel, compactBlocked)
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
	groupMinTTFT, hasGroupMin, groupMinErr := s.computeGroupMinTTFT(ctx, req.GroupID)
	if groupMinErr != nil {
		groupMinTTFT = 0
		hasGroupMin = false
	}
	available := make([]accountWithLoad, 0, len(candidates))
	loadRateSum := 0.0
	loadRateSumSquares := 0.0

	for _, c := range candidates {
		eval := s.evaluateRuntimePenalty(c.account.ID, groupMinTTFT, hasGroupMin)
		acc := s.applyPenaltyToAccount(c.account, eval)

		s.applyProbeRegistration(c.account, eval.ErrorPenalized, eval.TTFTPenalized, req.GroupID)

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

		fresh := s.service.resolveFreshSchedulableOpenAIAccount(ctx, selected.account, req.RequestedModel, req.RequireCompact, req.RequiredCapability)
		if fresh == nil || !s.isAccountRequestCompatible(ctx, fresh, req) || !accountSatisfiesPrivacyRequirement(fresh, schedGroup) || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) {
			recordPrivacyRequirementError(ctx, s.service, fresh, schedGroup)
			available = removeFromAvailable(available, selected.account.ID)
			continue
		}
		fresh = s.service.recheckSelectedOpenAIAccountFromDB(ctx, fresh, req.RequestedModel, req.RequireCompact, req.RequiredCapability)
		if fresh == nil || !s.isAccountRequestCompatible(ctx, fresh, req) || !accountSatisfiesPrivacyRequirement(fresh, schedGroup) || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) {
			recordPrivacyRequirementError(ctx, s.service, fresh, schedGroup)
			available = removeFromAvailable(available, selected.account.ID)
			continue
		}

		result, acquireErr := s.service.tryAcquireAccountSlot(ctx, fresh.ID, fresh.Concurrency)
		if acquireErr != nil {
			return nil, len(candidates), loadSkew, acquireErr
		}
		if result != nil && result.Acquired {
			if req.SessionHash != "" && !req.SkipStickyBind {
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
		fresh := s.service.resolveFreshSchedulableOpenAIAccount(ctx, account, req.RequestedModel, req.RequireCompact, req.RequiredCapability)
		if fresh == nil || !s.isAccountRequestCompatible(ctx, fresh, req) || !accountSatisfiesPrivacyRequirement(fresh, schedGroup) || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) {
			recordPrivacyRequirementError(ctx, s.service, fresh, schedGroup)
			continue
		}
		fresh = s.service.recheckSelectedOpenAIAccountFromDB(ctx, fresh, req.RequestedModel, req.RequireCompact, req.RequiredCapability)
		if fresh == nil || !s.isAccountRequestCompatible(ctx, fresh, req) || !accountSatisfiesPrivacyRequirement(fresh, schedGroup) || !s.isAccountTransportCompatible(fresh, req.RequiredTransport) {
			recordPrivacyRequirementError(ctx, s.service, fresh, schedGroup)
			continue
		}
		fallbackAccounts = append(fallbackAccounts, fresh)
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

func (s *layeredOpenAIAccountScheduler) recheckSessionStickyAccountFromDB(
	ctx context.Context,
	account *Account,
	req OpenAIAccountScheduleRequest,
	schedGroup *Group,
) (*Account, bool) {
	if s == nil || s.service == nil || account == nil {
		return nil, true
	}
	if s.service.schedulerSnapshot == nil || s.service.accountRepo == nil {
		fresh := s.service.recheckSelectedOpenAIAccountFromDB(ctx, account, req.RequestedModel, req.RequireCompact, req.RequiredCapability)
		if fresh == nil {
			return nil, false
		}
		return s.classifySessionStickyAccount(ctx, fresh, req, schedGroup)
	}

	latest, err := s.service.accountRepo.GetByID(ctx, account.ID)
	if err != nil || latest == nil {
		return nil, true
	}
	return s.classifySessionStickyAccount(ctx, latest, req, schedGroup)
}

func (s *layeredOpenAIAccountScheduler) classifySessionStickyAccount(
	ctx context.Context,
	account *Account,
	req OpenAIAccountScheduleRequest,
	schedGroup *Group,
) (*Account, bool) {
	if account == nil {
		return nil, true
	}
	if shouldClearStickySession(account, req.RequestedModel) || !account.IsOpenAI() || !account.IsSchedulable() {
		return nil, true
	}
	if !accountSatisfiesPrivacyRequirement(account, schedGroup) {
		return nil, true
	}
	if !s.isAccountTransportCompatible(account, req.RequiredTransport) {
		return nil, true
	}
	if s.isAccountUpstreamRestrictedByChannel(ctx, account, req) {
		return nil, true
	}
	if req.RequestedModel != "" && !account.IsModelSupported(req.RequestedModel) {
		return nil, false
	}
	if !account.SupportsOpenAIEndpointCapability(req.RequiredCapability) {
		return nil, false
	}
	if !account.SupportsOpenAIImageCapability(req.RequiredImageCapability) {
		return nil, false
	}
	if req.RequireCompact && openAICompactSupportTier(account) == 0 {
		return nil, false
	}
	return account, false
}

func (s *layeredOpenAIAccountScheduler) shouldEscapeStickyAccount(accountID int64) (reason string, errorRate float64, ttft float64, shouldEscape bool) {
	if s == nil || s.service == nil {
		return "", 0, 0, false
	}
	delegate := &defaultOpenAIAccountScheduler{stats: s.stats}
	return delegate.shouldEscapeStickyAccount(accountID, s.service.openAIStickyEscapeConfig())
}

func (s *layeredOpenAIAccountScheduler) shouldEscapeBusyStickyAccount(accountID int64, maxConcurrency int, cfg config.GatewaySchedulingConfig) bool {
	if s == nil || s.service == nil || !s.service.openAIStickyEscapeConfig().enabled || s.service.concurrencyService == nil || accountID <= 0 {
		return false
	}
	maxWaiting := cfg.StickySessionMaxWaiting
	if maxWaiting <= 0 {
		return false
	}
	waiting, err := s.service.concurrencyService.GetAccountWaitingCount(context.Background(), accountID)
	if err != nil {
		return false
	}
	return waiting >= maxWaiting
}

func (s *layeredOpenAIAccountScheduler) isAccountRequestCompatible(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest) bool {
	var service *OpenAIGatewayService
	if s != nil {
		service = s.service
	}
	return isOpenAIAccountRequestCompatible(ctx, service, account, req)
}

func (s *layeredOpenAIAccountScheduler) isAccountUpstreamRestrictedByChannel(ctx context.Context, account *Account, req OpenAIAccountScheduleRequest) bool {
	var service *OpenAIGatewayService
	if s != nil {
		service = s.service
	}
	return isOpenAIAccountUpstreamRestrictedByChannel(ctx, service, account, req)
}

func (s *layeredOpenAIAccountScheduler) deletePreviousResponseStickyForRequest(ctx context.Context, req OpenAIAccountScheduleRequest) {
	if s == nil || s.service == nil || !s.service.openAIStickyEnabled() {
		return
	}
	previousResponseID := strings.TrimSpace(req.PreviousResponseID)
	if previousResponseID == "" {
		return
	}
	store := s.service.getOpenAIWSStateStore()
	if store == nil {
		return
	}
	_ = store.DeleteResponseAccount(ctx, derefGroupID(req.GroupID), previousResponseID)
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

// applyProbeRegistration routes runtime penalty evaluation results into probe registration.
// Accounts with probe disabled (openai_probe_enabled=false) are completely skipped —
// they won't be markPenalized and won't have their entries cleared — avoiding
// interference with state from other sources.
func (s *layeredOpenAIAccountScheduler) applyProbeRegistration(account *Account, errorPenalized, ttftPenalized bool, groupID *int64) {
	if s == nil || s.probe == nil || account == nil {
		return
	}
	if !account.IsOpenAIProbeEnabled() {
		return
	}
	if errorPenalized || ttftPenalized {
		s.probe.markPenalized(account.ID, groupID, errorPenalized, ttftPenalized)
		return
	}
	s.probe.clearPenaltyReasons(account.ID)
}
