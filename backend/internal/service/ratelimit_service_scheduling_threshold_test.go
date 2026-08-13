//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestRateLimitService_ApplyAccountSchedulingThreshold_SetsTempUnschedulable(t *testing.T) {
	accountSchedulingThresholdsSF.Forget(SettingKeyAccountSchedulingThresholds)
	accountSchedulingThresholdsCache.Store(&cachedAccountSchedulingThresholds{})

	settingsRepo := newMockSettingRepo()
	settingsRepo.data[SettingKeyAccountSchedulingThresholds] = `{"openai":80}`

	accountRepo := &rateLimitAccountRepoStub{}
	rl := NewRateLimitService(accountRepo, nil, &config.Config{}, nil, nil)
	rl.SetSettingService(NewSettingService(settingsRepo, &config.Config{}))

	until := time.Now().UTC().Add(6 * time.Hour)
	account := &Account{
		ID:          1001,
		Platform:    PlatformOpenAI,
		Status:      StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"codex_7d_used_percent": 91.5,
			"codex_7d_reset_at":     until.Format(time.RFC3339),
		},
	}

	blocked := rl.ApplyAccountSchedulingThreshold(context.Background(), account)

	require.True(t, blocked)
	require.Equal(t, 1, accountRepo.tempCalls)
	require.NotNil(t, account.TempUnschedulableUntil)
	require.WithinDuration(t, until, *account.TempUnschedulableUntil, time.Second)
	require.True(t, IsAccountSchedulingThresholdReason(accountRepo.lastTempReason))

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(accountRepo.lastTempReason), &payload))
	require.Equal(t, PlatformOpenAI, payload["platform"])
	require.Equal(t, "7d", payload["window"])
	require.Equal(t, float64(80), payload["threshold_percent"])
	require.Equal(t, float64(91.5), payload["used_percent"])
	require.Contains(t, payload["error_message"], "91.5% used >= 80%")
}

func TestRateLimitService_ApplyAccountSchedulingThreshold_UsesAccountOverrideInReason(t *testing.T) {
	accountSchedulingThresholdsSF.Forget(SettingKeyAccountSchedulingThresholds)
	accountSchedulingThresholdsCache.Store(&cachedAccountSchedulingThresholds{})

	settingsRepo := newMockSettingRepo()
	settingsRepo.data[SettingKeyAccountSchedulingThresholds] = `{"openai":90}`

	accountRepo := &rateLimitAccountRepoStub{}
	rl := NewRateLimitService(accountRepo, nil, &config.Config{}, nil, nil)
	rl.SetSettingService(NewSettingService(settingsRepo, &config.Config{}))

	until := time.Now().UTC().Add(6 * time.Hour)
	account := &Account{
		ID:          1003,
		Platform:    PlatformOpenAI,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{
			"account_scheduling_threshold": 80,
		},
		Extra: map[string]any{
			"codex_7d_used_percent": 85.5,
			"codex_7d_reset_at":     until.Format(time.RFC3339),
		},
	}

	blocked := rl.ApplyAccountSchedulingThreshold(context.Background(), account)

	require.True(t, blocked)
	require.Equal(t, 1, accountRepo.tempCalls)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(accountRepo.lastTempReason), &payload))
	require.Equal(t, float64(80), payload["threshold_percent"])
	require.Equal(t, float64(85.5), payload["used_percent"])
	require.Contains(t, payload["error_message"], "85.5% used >= 80%")
}

func TestRateLimitService_ApplyAccountSchedulingThreshold_SkipsDuplicateTempUnschedulable(t *testing.T) {
	accountSchedulingThresholdsSF.Forget(SettingKeyAccountSchedulingThresholds)
	accountSchedulingThresholdsCache.Store(&cachedAccountSchedulingThresholds{})

	settingsRepo := newMockSettingRepo()
	settingsRepo.data[SettingKeyAccountSchedulingThresholds] = `{"openai":80}`

	accountRepo := &rateLimitAccountRepoStub{}
	rl := NewRateLimitService(accountRepo, nil, &config.Config{}, nil, nil)
	rl.SetSettingService(NewSettingService(settingsRepo, &config.Config{}))

	until := time.Now().UTC().Add(6 * time.Hour).Truncate(time.Second)
	existingReason := BuildDetailedAccountSchedulingThresholdReason(AccountSchedulingThresholdReasonInput{
		Platform:         PlatformOpenAI,
		Window:           "7d",
		ThresholdPercent: 80,
		UsedPercent:      91.5,
		Until:            until,
		Now:              until.Add(-time.Hour),
	})
	account := &Account{
		ID:                      1002,
		Platform:                PlatformOpenAI,
		Status:                  StatusActive,
		Schedulable:             true,
		TempUnschedulableUntil:  &until,
		TempUnschedulableReason: existingReason,
		Extra: map[string]any{
			"codex_7d_used_percent": 91.5,
			"codex_7d_reset_at":     until.Format(time.RFC3339),
		},
	}

	blocked := rl.ApplyAccountSchedulingThreshold(context.Background(), account)

	require.True(t, blocked)
	require.Equal(t, 0, accountRepo.tempCalls)
	require.Equal(t, existingReason, account.TempUnschedulableReason)
	require.NotNil(t, account.TempUnschedulableUntil)
	require.True(t, until.Equal(*account.TempUnschedulableUntil))
}

func TestRateLimitService_ApplyAccountSchedulingThreshold_UnsupportedPlatformDoesNotBlock(t *testing.T) {
	accountSchedulingThresholdsSF.Forget(SettingKeyAccountSchedulingThresholds)
	accountSchedulingThresholdsCache.Store(&cachedAccountSchedulingThresholds{})

	settingsRepo := newMockSettingRepo()
	settingsRepo.data[SettingKeyAccountSchedulingThresholds] = `{"openai":80}`

	accountRepo := &rateLimitAccountRepoStub{}
	rl := NewRateLimitService(accountRepo, nil, &config.Config{}, nil, nil)
	rl.SetSettingService(NewSettingService(settingsRepo, &config.Config{}))

	account := &Account{
		ID:          2002,
		Platform:    PlatformKiro,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{
			"account_scheduling_threshold": 1,
		},
		Extra: map[string]any{
			"kiro_sched_utilization": 99.0,
			"kiro_sched_reset_at":    time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
		},
	}

	blocked := rl.ApplyAccountSchedulingThreshold(context.Background(), account)

	require.False(t, blocked)
	require.Equal(t, 0, accountRepo.tempCalls)
	require.Nil(t, account.TempUnschedulableUntil)
	require.Empty(t, account.TempUnschedulableReason)
}

// --- review-fix E：阈值清空/提高到 100 后，只解除 account_scheduling_threshold 来源的暂停 ---

type thresholdReleaseAccountRepoStub struct {
	rateLimitAccountRepoStub
	paused                 []Account
	cleared                []int64
	conditionalClearCalls  []int64
	conditionalClearResult bool
	conditionalClearErr    error
	listErr                error
}

func (r *thresholdReleaseAccountRepoStub) ListTempUnschedulableByPlatform(_ context.Context, platform string, now time.Time) ([]Account, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []Account
	for _, a := range r.paused {
		if a.Platform == platform && a.TempUnschedulableUntil != nil && a.TempUnschedulableUntil.After(now) {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *thresholdReleaseAccountRepoStub) ClearTempUnschedulable(_ context.Context, id int64) error {
	r.cleared = append(r.cleared, id)
	return nil
}

func (r *thresholdReleaseAccountRepoStub) ClearTempUnschedulableIfSource(_ context.Context, id int64, source string) (bool, error) {
	if source != AccountSchedulingThresholdReasonSource {
		return false, nil
	}
	r.conditionalClearCalls = append(r.conditionalClearCalls, id)
	return r.conditionalClearResult, r.conditionalClearErr
}

func TestRateLimitService_ReleaseAccountSchedulingThresholdPauses_OnlyClearsThresholdSource(t *testing.T) {
	now := time.Now().UTC()
	until := now.Add(2 * time.Hour)
	thresholdReason := BuildDetailedAccountSchedulingThresholdReason(AccountSchedulingThresholdReasonInput{
		Platform: PlatformOpenAI, Window: "7d", ThresholdPercent: 90, UsedPercent: 95, Until: until, Now: now,
	})

	repo := &thresholdReleaseAccountRepoStub{
		conditionalClearResult: true,
		paused: []Account{
			{ID: 5001, Platform: PlatformOpenAI, TempUnschedulableUntil: &until, TempUnschedulableReason: thresholdReason},
			{ID: 5002, Platform: PlatformOpenAI, TempUnschedulableUntil: &until, TempUnschedulableReason: BuildTempUnschedReasonPayload("rate_limit", "rate limited")},
			{ID: 5003, Platform: PlatformAnthropic, TempUnschedulableUntil: &until, TempUnschedulableReason: thresholdReason},
			{ID: 5004, Platform: PlatformGrok, TempUnschedulableUntil: &until, TempUnschedulableReason: BuildTempUnschedReasonPayload("layered_probe", "probe failed")},
		},
	}
	blocker := &runtimeBlockRecorder{}
	cache := &tempUnschedCacheRecorder{}
	rl := NewRateLimitService(repo, nil, &config.Config{}, nil, cache)
	rl.SetAccountRuntimeBlocker(blocker)

	// 只把 openai 阈值提高到 100：只解除 openai 来源暂停，其余平台与其它来源不受影响。
	rl.ReleaseAccountSchedulingThresholdPauses(context.Background(), []string{PlatformOpenAI})

	require.ElementsMatch(t, []int64{5001}, repo.conditionalClearCalls, "只应条件解除 account_scheduling_threshold 来源的暂停")
	require.Empty(t, repo.cleared, "阈值解除不得回退到无条件 ClearTempUnschedulable")
	require.ElementsMatch(t, []int64{5001}, blocker.clearedIDs, "runtime block 必须同步解除")
	require.ElementsMatch(t, []int64{5001}, cache.deletedIDs, "条件清除成功后必须删除 Redis temp-unsched 状态")
}

func TestRateLimitService_ReleaseAccountSchedulingThresholdPauses_DoesNotClearNewerNonThresholdPause(t *testing.T) {
	now := time.Now().UTC()
	until := now.Add(time.Hour)
	thresholdReason := BuildDetailedAccountSchedulingThresholdReason(AccountSchedulingThresholdReasonInput{
		Platform: PlatformOpenAI, Window: "7d", ThresholdPercent: 90, UsedPercent: 95, Until: until, Now: now,
	})
	repo := &thresholdReleaseAccountRepoStub{
		conditionalClearResult: false,
		paused:                 []Account{{ID: 5006, Platform: PlatformOpenAI, TempUnschedulableUntil: &until, TempUnschedulableReason: thresholdReason}},
	}
	cache := &tempUnschedCacheRecorder{}
	blocker := &runtimeBlockRecorder{}
	rl := NewRateLimitService(repo, nil, &config.Config{}, nil, cache)
	rl.SetAccountRuntimeBlocker(blocker)

	rl.ReleaseAccountSchedulingThresholdPauses(context.Background(), []string{PlatformOpenAI})

	require.Equal(t, []int64{5006}, repo.conditionalClearCalls)
	require.Empty(t, blocker.clearedIDs, "source-conditional DB miss means a newer pause must retain its runtime block")
	require.Empty(t, cache.deletedIDs, "source-conditional DB miss must retain Redis temp-unsched state")
}

func TestRateLimitService_ReleaseAccountSchedulingThresholdPauses_DBFailureRetainsRuntimeAndRedis(t *testing.T) {
	now := time.Now().UTC()
	until := now.Add(time.Hour)
	thresholdReason := BuildDetailedAccountSchedulingThresholdReason(AccountSchedulingThresholdReasonInput{
		Platform: PlatformOpenAI, Window: "7d", ThresholdPercent: 90, UsedPercent: 95, Until: until, Now: now,
	})
	repo := &thresholdReleaseAccountRepoStub{
		conditionalClearErr: errors.New("db unavailable"),
		paused:              []Account{{ID: 5008, Platform: PlatformOpenAI, TempUnschedulableUntil: &until, TempUnschedulableReason: thresholdReason}},
	}
	cache := &tempUnschedCacheRecorder{}
	blocker := &runtimeBlockRecorder{}
	rl := NewRateLimitService(repo, nil, &config.Config{}, nil, cache)
	rl.SetAccountRuntimeBlocker(blocker)

	rl.ReleaseAccountSchedulingThresholdPauses(context.Background(), []string{PlatformOpenAI})

	require.Equal(t, []int64{5008}, repo.conditionalClearCalls)
	require.Empty(t, blocker.clearedIDs)
	require.Empty(t, cache.deletedIDs)
}

func TestRateLimitService_ReleaseAccountSchedulingThresholdPauses_ClearsRuntimeOnlyThresholdBlock(t *testing.T) {
	account := &Account{ID: 5007, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	gateway := &OpenAIGatewayService{}
	until := time.Now().Add(time.Hour)
	gateway.BlockAccountScheduling(account, until, AccountSchedulingThresholdReasonSource)
	repo := &thresholdReleaseAccountRepoStub{conditionalClearResult: true}
	rl := NewRateLimitService(repo, nil, &config.Config{}, nil, &tempUnschedCacheRecorder{})
	rl.SetAccountRuntimeBlocker(gateway)

	rl.ReleaseAccountSchedulingThresholdPauses(context.Background(), []string{PlatformOpenAI})

	require.False(t, gateway.isOpenAIAccountRuntimeBlocked(account), "runtime-only threshold block must release even when persistence previously failed")
}

func TestRateLimitService_ReleaseAccountSchedulingThresholdPauses_RetainsNewerNonThresholdRuntimeBlock(t *testing.T) {
	account := &Account{ID: 5009, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	gateway := &OpenAIGatewayService{}
	gateway.BlockAccountScheduling(account, time.Now().Add(time.Hour), AccountSchedulingThresholdReasonSource)
	gateway.BlockAccountScheduling(account, time.Now().Add(2*time.Hour), "rate_limit")
	rl := NewRateLimitService(&thresholdReleaseAccountRepoStub{}, nil, &config.Config{}, nil, nil)
	rl.SetAccountRuntimeBlocker(gateway)

	rl.ReleaseAccountSchedulingThresholdPauses(context.Background(), []string{PlatformOpenAI})

	require.True(t, gateway.isOpenAIAccountRuntimeBlocked(account), "a newer non-threshold runtime pause must survive threshold release")
}

func TestRateLimitService_ReleaseAccountSchedulingThresholdPauses_ExpiredPausesNotTouched(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	thresholdReason := BuildDetailedAccountSchedulingThresholdReason(AccountSchedulingThresholdReasonInput{
		Platform: PlatformOpenAI, Window: "7d", ThresholdPercent: 90, UsedPercent: 95, Until: now.Add(2 * time.Hour), Now: now,
	})

	repo := &thresholdReleaseAccountRepoStub{
		paused: []Account{
			{ID: 5005, Platform: PlatformOpenAI, TempUnschedulableUntil: &past, TempUnschedulableReason: thresholdReason},
		},
	}
	rl := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)

	rl.ReleaseAccountSchedulingThresholdPauses(context.Background(), []string{PlatformOpenAI})

	require.Empty(t, repo.cleared, "已过期的暂停不在 ListTempUnschedulableByPlatform 结果中，不应产生 Clear 调用")
}
