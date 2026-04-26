package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func newLayeredTestService(accounts []Account) *OpenAIGatewayService {
	cfg := &config.Config{}
	cfg.Gateway.Sticky.OpenAI.Enabled = true
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyThreshold = 0.3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyValue = 100
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyMultiplier = 3.0
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyValue = 50
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds = 60
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds = 30
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeMaxFailures = 3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeTimeoutSeconds = 15
	cfg.Gateway.OpenAIWS.LBTopK = 7
	cfg.Gateway.OpenAIWS.StickySessionTTLSeconds = 3600
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600
	return &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: accounts},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
	}
}

// TestLayered_PriorityDeterminism verifies that the layered scheduler always
// picks the account with the lowest (best) priority when both are idle.
func TestLayered_PriorityDeterminism(t *testing.T) {
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 10},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 50},
	}
	svc := newLayeredTestService(accounts)
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })
	require.NotNil(t, scheduler)

	ctx := context.Background()
	req := OpenAIAccountScheduleRequest{RequestedModel: ""}

	for i := 0; i < 20; i++ {
		result, _, err := scheduler.Select(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Account)
		require.Equal(t, int64(1), result.Account.ID, "iteration %d: expected account 1 (priority=10)", i)
		if result.ReleaseFunc != nil {
			result.ReleaseFunc()
		}
	}
}

func TestLayered_RequiredImageCapabilityFiltersUnsupportedAccounts(t *testing.T) {
	accounts := []Account{
		{ID: 11, Platform: PlatformOpenAI, Type: AccountTypeUpstream, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0},
		{ID: 12, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5},
	}
	svc := newLayeredTestService(accounts)
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })
	require.NotNil(t, scheduler)

	result, _, err := scheduler.Select(context.Background(), OpenAIAccountScheduleRequest{
		RequiredImageCapability: OpenAIImagesCapabilityNative,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Account)
	require.Equal(t, int64(12), result.Account.ID)
	if result.ReleaseFunc != nil {
		result.ReleaseFunc()
	}
}

func TestLayeredScheduler_ConstructionDoesNotRehydrateTempUnschedulableAccounts(t *testing.T) {
	future := time.Now().Add(10 * time.Minute)
	reason, err := buildLayeredProbeTempUnschedReason("consecutive_failures", 3)
	require.NoError(t, err)
	repo := &startupRehydrateRepoStub{tempUnschedAccounts: []Account{{
		ID:                      301,
		Platform:                PlatformOpenAI,
		Type:                    AccountTypeAPIKey,
		Status:                  StatusActive,
		Schedulable:             true,
		TempUnschedulableUntil:  &future,
		TempUnschedulableReason: reason,
	}}}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	svc := &OpenAIGatewayService{accountRepo: repo, cfg: cfg}

	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	require.NotNil(t, scheduler)
	layered, ok := scheduler.(*layeredOpenAIAccountScheduler)
	require.True(t, ok, "scheduler should be layered")
	require.NotNil(t, layered.probe)
	_, exists := layered.probe.entries.Load(int64(301))
	require.False(t, exists, "scheduler construction must not implicitly rehydrate temp-unschedulable accounts")
	require.Equal(t, 0, repo.listCalls, "scheduler construction must not query startup recovery state")
}

func TestLayeredScheduler_StartOpenAIBackgroundRecoveryRehydratesTempUnschedulableAccounts(t *testing.T) {
	now := time.Now()
	future := now.Add(10 * time.Minute)
	cooldown := 60 * time.Second
	reason, err := buildLayeredProbeTempUnschedReason("consecutive_failures", 3)
	require.NoError(t, err)
	repo := &startupRehydrateRepoStub{tempUnschedAccounts: []Account{{
		ID:                      301,
		Platform:                PlatformOpenAI,
		Type:                    AccountTypeAPIKey,
		Status:                  StatusActive,
		Schedulable:             true,
		TempUnschedulableUntil:  &future,
		TempUnschedulableReason: reason,
	}}}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	svc := &OpenAIGatewayService{accountRepo: repo, cfg: cfg}

	scheduler := svc.getOpenAIAccountScheduler()
	svc.StartOpenAIBackgroundRecovery()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	require.NotNil(t, scheduler)
	layered, ok := scheduler.(*layeredOpenAIAccountScheduler)
	require.True(t, ok, "scheduler should be layered")
	value, exists := layered.probe.entries.Load(int64(301))
	require.True(t, exists, "explicit startup recovery should rehydrate temp-unschedulable accounts into probe entries")
	entry := value.(*openAIAccountProbeEntry)
	require.True(t, entry.dbFlagSet.Load(), "startup bootstrap entry should preserve db flag state")
	require.True(t, entry.startupRehydrated.Load(), "startup bootstrap entry should remember it was rehydrated from DB truth")
	require.False(t, entry.errorPenalized.Load(), "startup bootstrap entry should not set error penalty")
	require.False(t, entry.ttftPenalized.Load(), "startup bootstrap entry should not set ttft penalty")
	require.GreaterOrEqual(t, now.Sub(entry.penalizedAt), cooldown, "startup bootstrap entry should be immediately eligible for next tick")
	require.Equal(t, 1, repo.listCalls, "explicit startup recovery should query temp-unschedulable accounts exactly once")
}

func TestLayeredScheduler_StartOpenAIBackgroundRecoveryUsesTimeoutContext(t *testing.T) {
	repo := &startupRehydrateRepoStub{requireDeadline: true}
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	svc := &OpenAIGatewayService{accountRepo: repo, cfg: cfg}

	scheduler := svc.getOpenAIAccountScheduler()
	svc.StartOpenAIBackgroundRecovery()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	require.NotNil(t, scheduler)
	_, ok := scheduler.(*layeredOpenAIAccountScheduler)
	require.True(t, ok, "scheduler should be layered")
	require.Equal(t, 1, repo.listCalls, "explicit startup recovery should attempt temp-unschedulable rehydrate with timeout context")
	require.True(t, repo.sawDeadline, "explicit startup recovery should use a timeout context")
}

// TestLayered_TTFTPenaltyPushesToLowerPriority verifies that an account with
// high TTFT (exceeding minTTFT * TTFTPenaltyMultiplier) receives a priority
// penalty, causing the scheduler to prefer a lower-priority but faster account.
func TestLayered_TTFTPenaltyPushesToLowerPriority(t *testing.T) {
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 10},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 50},
	}

	cfg := &config.Config{}
	cfg.Gateway.Sticky.OpenAI.Enabled = true
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyThreshold = 0.3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyValue = 100
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyMultiplier = 3.0
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyValue = 50
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds = 60
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds = 30
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeMaxFailures = 3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeTimeoutSeconds = 15
	cfg.Gateway.OpenAIWS.LBTopK = 7
	cfg.Gateway.OpenAIWS.StickySessionTTLSeconds = 3600
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600

	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: accounts},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
	}
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })
	require.NotNil(t, scheduler)

	// Report multiple results to stabilize EWMA:
	//   Account 1: high TTFT (9000ms) → EWMA converges toward 9000
	//   Account 2: low  TTFT (1000ms) → EWMA converges toward 1000
	for i := 0; i < 10; i++ {
		scheduler.ReportResult(1, true, intPtr(9000))
		scheduler.ReportResult(2, true, intPtr(1000))
	}

	// After EWMA stabilization:
	//   minTTFT ≈ 1000, threshold = 1000 * 3.0 = 3000
	//   Account 1 TTFT ≈ 9000 > 3000 → penalty applied: effectivePriority = 10 + 50 = 60
	//   Account 2 TTFT ≈ 1000 < 3000 → no penalty:       effectivePriority = 50
	//   Account 2 (60 > 50) should be selected.

	ctx := context.Background()
	req := OpenAIAccountScheduleRequest{RequestedModel: ""}

	result, _, err := scheduler.Select(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Account)
	require.Equal(t, int64(2), result.Account.ID, "account 1 should be TTFT-penalized; account 2 (priority=50) should be selected")
	if result.ReleaseFunc != nil {
		result.ReleaseFunc()
	}
}

// TestLayered_ErrorPenaltyPushesToLowerPriority verifies that reporting
// consecutive failures raises the effective priority via the error penalty,
// causing the scheduler to prefer the other account.
func TestLayered_ErrorPenaltyPushesToLowerPriority(t *testing.T) {
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 10},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 50},
	}
	svc := newLayeredTestService(accounts)
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })
	require.NotNil(t, scheduler)

	// Report 5 consecutive failures on account 1.
	// EWMA alpha=0.2, starting from 0:
	//   after 1: 0.2
	//   after 2: 0.2 + 0.8*0.2 = 0.36
	//   after 3: 0.2 + 0.8*0.36 = 0.488
	//   after 4: 0.2 + 0.8*0.488 = 0.5904
	//   after 5: 0.2 + 0.8*0.5904 = 0.67232
	// errorRate ~0.67 > threshold 0.3  →  effectivePriority = 10 + 100 = 110 > 50
	for i := 0; i < 5; i++ {
		scheduler.ReportResult(1, false, nil)
	}

	ctx := context.Background()
	req := OpenAIAccountScheduleRequest{RequestedModel: ""}

	result, _, err := scheduler.Select(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Account)
	require.Equal(t, int64(2), result.Account.ID, "account 1 should be penalized; account 2 (priority=50) should be selected")
	if result.ReleaseFunc != nil {
		result.ReleaseFunc()
	}
}

// TestLayered_TTFTPenalty_UsesGroupLevelBaseline verifies that the TTFT penalty
// uses the group-level minimum TTFT (across all schedulable accounts in the same
// group), not a request-context-local minimum computed only from filtered candidates.
func TestLayered_TTFTPenalty_UsesGroupLevelBaseline(t *testing.T) {
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 10},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 50},
		{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 50},
	}
	svc := newLayeredTestService(accounts)
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	for i := 0; i < 10; i++ {
		slow := 9000
		fast := 1000
		normal := 2500
		scheduler.ReportResult(1, true, &slow)
		scheduler.ReportResult(2, true, &fast)
		scheduler.ReportResult(3, true, &normal)
	}

	excluded := map[int64]struct{}{2: {}}
	result, _, err := scheduler.Select(context.Background(), OpenAIAccountScheduleRequest{
		ExcludedIDs: excluded,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Account)
	require.Equal(t, int64(3), result.Account.ID)
	if result.ReleaseFunc != nil {
		result.ReleaseFunc()
	}
}

// TestLayered_TTFTPenalty_SharedEvaluatorUsesConsistentGroupBaseline verifies
// that the shared evaluator helpers compute and apply the same group-level TTFT
// baseline regardless of which account is being evaluated. This covers helper
// consistency only; it does not verify probe end-to-end recovery behavior.
func TestLayered_TTFTPenalty_SharedEvaluatorUsesConsistentGroupBaseline(t *testing.T) {
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 10},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 50},
	}
	svc := newLayeredTestService(accounts)
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	stats := newOpenAIAccountRuntimeStats()
	for i := 0; i < 5; i++ {
		slow := 9000
		fast := 1000
		stats.report(1, true, &slow)
		stats.report(2, true, &fast)
	}

	ls := newLayeredOpenAIAccountScheduler(svc, stats)
	t.Cleanup(func() { ls.Stop() })

	groupMinTTFT, hasGroupMin, err := ls.computeGroupMinTTFT(context.Background(), nil)
	require.NoError(t, err)
	eval1 := ls.evaluateRuntimePenalty(1, groupMinTTFT, hasGroupMin)
	eval2 := ls.evaluateRuntimePenalty(2, groupMinTTFT, hasGroupMin)

	require.True(t, eval1.TTFTPenalized)
	require.False(t, eval2.TTFTPenalized)
	require.Greater(t, eval1.GroupMinTTFT, 0.0)
	require.InDelta(t, eval1.GroupMinTTFT, eval2.GroupMinTTFT, 0.01)
}

func TestLayered_TTFTBaselineUsesOnlyRequestEligibleAccounts(t *testing.T) {
	groupID := int64(93001)
	repo := schedulerTestOpenAIAccountRepo{accounts: []Account{
		{ID: 21, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, Extra: map[string]any{}},
		{ID: 22, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, Extra: map[string]any{"privacy_mode": PrivacyModeTrainingOff}},
		{ID: 23, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 10, Extra: map[string]any{"privacy_mode": PrivacyModeTrainingOff}},
	}}
	svc := newLayeredTestService(repo.accounts)
	snapshotCfg := &config.Config{}
	snapshotCfg.Gateway.Scheduling.DbFallbackEnabled = true
	svc.accountRepo = repo
	svc.schedulerSnapshot = NewSchedulerSnapshotService(nil, nil, repo, schedulerTestGroupRepo{groups: map[int64]*Group{
		groupID: {ID: groupID, Name: "privacy-required", RequirePrivacySet: true},
	}}, snapshotCfg)
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })
	require.NotNil(t, scheduler)

	for i := 0; i < 5; i++ {
		fast := 1000
		normal := 4000
		fallback := 2000
		scheduler.ReportResult(21, true, &fast)
		scheduler.ReportResult(22, true, &normal)
		scheduler.ReportResult(23, true, &fallback)
	}

	result, _, err := scheduler.Select(context.Background(), OpenAIAccountScheduleRequest{
		GroupID:        &groupID,
		RequestedModel: "gpt-5.1",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Account)
	require.Equal(t, int64(22), result.Account.ID)
	if result.ReleaseFunc != nil {
		result.ReleaseFunc()
	}
}

// TestLayered_FallbackWhenHighPriorityFullyLoaded verifies that when the
// highest-priority account's concurrency slots are exhausted, the scheduler
// falls back to the next available account.
func TestLayered_FallbackWhenHighPriorityFullyLoaded(t *testing.T) {
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 10},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 50},
	}

	// Use a concurrency cache that reports account 1 at 100% load and blocks
	// acquisition, simulating a fully occupied account.
	cc := stubConcurrencyCache{
		loadMap: map[int64]*AccountLoadInfo{
			1: {AccountID: 1, LoadRate: 100, WaitingCount: 0},
			2: {AccountID: 2, LoadRate: 0, WaitingCount: 0},
		},
		acquireResults: map[int64]bool{
			1: false, // account 1 slot acquisition fails
			2: true,  // account 2 slot acquisition succeeds
		},
	}

	cfg := &config.Config{}
	cfg.Gateway.Sticky.OpenAI.Enabled = true
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyThreshold = 0.3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyValue = 100
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyMultiplier = 3.0
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyValue = 50
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds = 60
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds = 30
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeMaxFailures = 3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeTimeoutSeconds = 15
	cfg.Gateway.OpenAIWS.LBTopK = 7
	cfg.Gateway.OpenAIWS.StickySessionTTLSeconds = 3600
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600

	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: accounts},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(cc),
	}
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })
	require.NotNil(t, scheduler)

	ctx := context.Background()
	req := OpenAIAccountScheduleRequest{RequestedModel: ""}

	// Account 1 is at 100% load (filtered out by loadRate >= 100 check).
	// The scheduler should pick account 2.
	result, _, err := scheduler.Select(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Account)
	require.Equal(t, int64(2), result.Account.ID, "should fall back to account 2 when account 1 is fully loaded")
	if result.ReleaseFunc != nil {
		result.ReleaseFunc()
	}
}

func TestLayered_PreviousResponseStickyEnabled(t *testing.T) {
	ctx := context.Background()
	groupID := int64(9201)
	stickyAccount := Account{ID: 92011, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 9, Extra: map[string]any{"openai_apikey_responses_websockets_v2_enabled": true}}
	backupAccount := Account{ID: 92012, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, Extra: map[string]any{"openai_apikey_responses_websockets_v2_enabled": true}}
	cache := &stubGatewayCache{}
	stateStore := &openAIWSStateStoreSpy{responseAccounts: map[string]int64{"resp_layered_enabled": stickyAccount.ID}}
	cfg := &config.Config{}
	cfg.Gateway.Sticky.OpenAI.Enabled = true
	cfg.Gateway.Sticky.Gemini.Enabled = true
	cfg.Gateway.Sticky.Anthropic.Enabled = true
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.StickySessionTTLSeconds = 3600
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600

	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{stickyAccount, backupAccount}},
		cache:              cache,
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
		openaiWSStateStore: stateStore,
	}
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	selection, decision, err := scheduler.Select(ctx, OpenAIAccountScheduleRequest{GroupID: &groupID, PreviousResponseID: "resp_layered_enabled", SessionHash: "session_hash_layered_enabled", RequestedModel: "gpt-5.1"})
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, stickyAccount.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerPreviousResponse, decision.Layer)
	require.True(t, decision.StickyPreviousHit)
	require.Equal(t, 1, stateStore.getResponseAccountCalls["resp_layered_enabled"])
	require.Equal(t, 1, stateStore.bindResponseCalls["resp_layered_enabled"])
	require.Equal(t, stickyAccount.ID, cache.sessionBindings["openai:session_hash_layered_enabled"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestLayered_PreviousResponseStickyHonorsRequirePrivacySet(t *testing.T) {
	ctx := context.Background()
	groupID := int64(92011)
	stickyAccount := Account{ID: 920111, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 9, Extra: map[string]any{"openai_apikey_responses_websockets_v2_enabled": true}}
	backupAccount := Account{ID: 920112, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, Extra: map[string]any{"privacy_mode": PrivacyModeTrainingOff, "openai_apikey_responses_websockets_v2_enabled": true}}
	repo := schedulerTestOpenAIAccountRepo{accounts: []Account{stickyAccount, backupAccount}, setErrors: map[int64]string{}}
	cache := &schedulerTestGatewayCache{}
	stateStore := &openAIWSStateStoreSpy{responseAccounts: map[string]int64{"resp_layered_privacy_required": stickyAccount.ID}}
	snapshotCfg := &config.Config{}
	snapshotCfg.Gateway.Scheduling.DbFallbackEnabled = true
	cfg := &config.Config{}
	cfg.Gateway.Sticky.OpenAI.Enabled = true
	cfg.Gateway.Sticky.Gemini.Enabled = true
	cfg.Gateway.Sticky.Anthropic.Enabled = true
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.StickySessionTTLSeconds = 3600
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600

	svc := &OpenAIGatewayService{
		accountRepo:        repo,
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
		openaiWSStateStore: stateStore,
		schedulerSnapshot: NewSchedulerSnapshotService(&openAISnapshotCacheStub{
			snapshotAccounts: []*Account{&stickyAccount, &backupAccount},
			accountsByID: map[int64]*Account{
				stickyAccount.ID: &stickyAccount,
				backupAccount.ID: &backupAccount,
			},
		}, nil, repo, schedulerTestGroupRepo{groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "privacy-required", RequirePrivacySet: true},
		}}, snapshotCfg),
	}
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	selection, decision, err := scheduler.Select(ctx, OpenAIAccountScheduleRequest{GroupID: &groupID, PreviousResponseID: "resp_layered_privacy_required", SessionHash: "session_hash_layered_privacy_required", RequestedModel: "gpt-5.1"})
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, backupAccount.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Contains(t, repo.setErrors[stickyAccount.ID], "Privacy not set")
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestLayered_SessionStickyRecheckHonorsImageCapability(t *testing.T) {
	ctx := context.Background()
	groupID := int64(92012)
	stickySnapshotAccount := Account{ID: 920121, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 9, Extra: map[string]any{"openai_apikey_responses_websockets_v2_enabled": true}}
	stickyDBAccount := stickySnapshotAccount
	stickyDBAccount.Type = AccountTypeUpstream
	backupAccount := Account{ID: 920122, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	repo := schedulerTestOpenAIAccountRepo{accounts: []Account{stickyDBAccount, backupAccount}}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_layered_images": stickySnapshotAccount.ID}}
	snapshotCfg := &config.Config{}
	snapshotCfg.Gateway.Scheduling.DbFallbackEnabled = true
	cfg := &config.Config{}
	cfg.Gateway.Sticky.OpenAI.Enabled = true
	cfg.Gateway.Sticky.Gemini.Enabled = true
	cfg.Gateway.Sticky.Anthropic.Enabled = true
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"

	svc := &OpenAIGatewayService{
		accountRepo:        repo,
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
		schedulerSnapshot: NewSchedulerSnapshotService(&openAISnapshotCacheStub{
			snapshotAccounts: []*Account{&stickySnapshotAccount, &backupAccount},
			accountsByID: map[int64]*Account{
				stickySnapshotAccount.ID: &stickySnapshotAccount,
				backupAccount.ID:         &backupAccount,
			},
		}, nil, repo, nil, snapshotCfg),
	}

	selection, decision, err := svc.SelectAccountWithSchedulerForImages(ctx, &groupID, "session_hash_layered_images", "gpt-image-1", nil, OpenAIImagesCapabilityNative)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, backupAccount.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestLayered_FallbackWaitPlanRechecksPrivacyRequirementAgainstDB(t *testing.T) {
	ctx := context.Background()
	groupID := int64(92013)
	staleSnapshotAccount := Account{ID: 920131, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, Extra: map[string]any{"privacy_mode": PrivacyModeTrainingOff}}
	staleDBAccount := staleSnapshotAccount
	staleDBAccount.Extra = map[string]any{}
	backupAccount := Account{ID: 920132, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 5, Extra: map[string]any{"privacy_mode": PrivacyModeTrainingOff}}
	repo := schedulerTestOpenAIAccountRepo{accounts: []Account{staleDBAccount, backupAccount}, setErrors: map[int64]string{}}
	snapshotCfg := &config.Config{}
	snapshotCfg.Gateway.Scheduling.DbFallbackEnabled = true
	cfg := &config.Config{}
	cfg.Gateway.Sticky.OpenAI.Enabled = true
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	cfg.Gateway.Scheduling.FallbackWaitTimeout = 5 * time.Second
	cfg.Gateway.Scheduling.FallbackMaxWaiting = 3

	svc := &OpenAIGatewayService{
		accountRepo: repo,
		cfg:         cfg,
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{
			loadMap: map[int64]*AccountLoadInfo{
				staleSnapshotAccount.ID: {AccountID: staleSnapshotAccount.ID, LoadRate: 0},
				backupAccount.ID:        {AccountID: backupAccount.ID, LoadRate: 0},
			},
			acquireResults: map[int64]bool{
				staleSnapshotAccount.ID: false,
				backupAccount.ID:        false,
			},
		}),
		schedulerSnapshot: NewSchedulerSnapshotService(&openAISnapshotCacheStub{
			snapshotAccounts: []*Account{&staleSnapshotAccount, &backupAccount},
			accountsByID: map[int64]*Account{
				staleSnapshotAccount.ID: &staleSnapshotAccount,
				backupAccount.ID:        &backupAccount,
			},
		}, nil, repo, schedulerTestGroupRepo{groups: map[int64]*Group{
			groupID: {ID: groupID, Name: "privacy-required", RequirePrivacySet: true},
		}}, snapshotCfg),
	}
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	selection, decision, err := scheduler.Select(ctx, OpenAIAccountScheduleRequest{GroupID: &groupID, RequestedModel: "gpt-5.1"})
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.WaitPlan)
	require.Equal(t, backupAccount.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Contains(t, repo.setErrors[staleSnapshotAccount.ID], "Privacy not set")
}

func TestLayered_SessionStickyEnabled(t *testing.T) {
	ctx := context.Background()
	groupID := int64(9202)
	stickyAccount := Account{ID: 92021, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 9}
	backupAccount := Account{ID: 92022, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	cache := &stubGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_layered_sticky": stickyAccount.ID}}
	cfg := &config.Config{}
	cfg.Gateway.Sticky.OpenAI.Enabled = true
	cfg.Gateway.Sticky.Gemini.Enabled = true
	cfg.Gateway.Sticky.Anthropic.Enabled = true
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	cfg.Gateway.OpenAIWS.StickySessionTTLSeconds = 3600

	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{stickyAccount, backupAccount}},
		cache:              cache,
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
	}
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	selection, decision, err := scheduler.Select(ctx, OpenAIAccountScheduleRequest{GroupID: &groupID, SessionHash: "session_hash_layered_sticky", RequestedModel: "gpt-5.1"})
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, stickyAccount.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
	require.True(t, decision.StickySessionHit)
	require.Equal(t, 1, cache.getCalls["openai:session_hash_layered_sticky"])
	require.Equal(t, 1, cache.refreshCalls["openai:session_hash_layered_sticky"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestLayered_PreviousResponseStickyDisabledBypassesStickyLookupAndBind(t *testing.T) {
	ctx := context.Background()
	groupID := int64(9203)
	stickyAccount := Account{ID: 92031, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 9, Extra: map[string]any{"openai_apikey_responses_websockets_v2_enabled": true}}
	bestAccount := Account{ID: 92032, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, Extra: map[string]any{"openai_apikey_responses_websockets_v2_enabled": true}}
	cache := &stubGatewayCache{}
	stateStore := &openAIWSStateStoreSpy{responseAccounts: map[string]int64{"resp_layered_disabled": stickyAccount.ID}}
	cfg := &config.Config{}
	cfg.Gateway.Sticky.OpenAI.Enabled = false
	cfg.Gateway.Sticky.Gemini.Enabled = true
	cfg.Gateway.Sticky.Anthropic.Enabled = true
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"

	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{stickyAccount, bestAccount}},
		cache:              cache,
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
		openaiWSStateStore: stateStore,
	}
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	selection, decision, err := scheduler.Select(ctx, OpenAIAccountScheduleRequest{GroupID: &groupID, PreviousResponseID: "resp_layered_disabled", SessionHash: "session_hash_layered_disabled", RequestedModel: "gpt-5.1"})
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, bestAccount.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickyPreviousHit)
	require.Zero(t, stateStore.getResponseAccountCalls["resp_layered_disabled"])
	require.Zero(t, stateStore.bindResponseCalls["resp_layered_disabled"])
	require.Zero(t, cache.getCalls["openai:session_hash_layered_disabled"])
	require.Zero(t, cache.setCalls["openai:session_hash_layered_disabled"])
	require.Zero(t, cache.refreshCalls["openai:session_hash_layered_disabled"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestLayered_SessionStickyDisabledBypassesLookupBindAndRefresh(t *testing.T) {
	ctx := context.Background()
	groupID := int64(9204)
	stickyAccount := Account{ID: 92041, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 9}
	bestAccount := Account{ID: 92042, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	cache := &stubGatewayCache{sessionBindings: map[string]int64{"openai:session_hash_layered_disabled_only": stickyAccount.ID}}
	cfg := &config.Config{}
	cfg.Gateway.Sticky.OpenAI.Enabled = false
	cfg.Gateway.Sticky.Gemini.Enabled = true
	cfg.Gateway.Sticky.Anthropic.Enabled = true
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"

	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{stickyAccount, bestAccount}},
		cache:              cache,
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
	}
	scheduler := svc.getOpenAIAccountScheduler()
	t.Cleanup(func() { svc.StopOpenAIAccountScheduler() })

	selection, decision, err := scheduler.Select(ctx, OpenAIAccountScheduleRequest{GroupID: &groupID, SessionHash: "session_hash_layered_disabled_only", RequestedModel: "gpt-5.1"})
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, bestAccount.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickySessionHit)
	require.Zero(t, cache.getCalls["openai:session_hash_layered_disabled_only"])
	require.Zero(t, cache.setCalls["openai:session_hash_layered_disabled_only"])
	require.Zero(t, cache.refreshCalls["openai:session_hash_layered_disabled_only"])
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}
