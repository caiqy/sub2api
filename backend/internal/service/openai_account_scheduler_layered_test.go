package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func newLayeredTestService(accounts []Account) *OpenAIGatewayService {
	cfg := &config.Config{}
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

// TestLayered_TTFTPenaltyPushesToLowerPriority verifies that an account with
// high TTFT (exceeding minTTFT * TTFTPenaltyMultiplier) receives a priority
// penalty, causing the scheduler to prefer a lower-priority but faster account.
func TestLayered_TTFTPenaltyPushesToLowerPriority(t *testing.T) {
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 10},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 50},
	}

	cfg := &config.Config{}
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
