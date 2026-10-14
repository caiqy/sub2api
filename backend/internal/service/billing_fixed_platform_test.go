//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

func TestReserveFixedCharge_UserPlatformPendingAndRelease(t *testing.T) {
	now := time.Now()
	day, week := timezone.StartOfDay(now), timezone.StartOfWeek(now)
	limit := 10.0
	cache := &fixedChargeCacheStub{balance: 100, quota: &UserPlatformQuotaCacheEntry{
		SchemaVersion: UserPlatformQuotaCacheSchemaV1, DailyLimitUSD: &limit, WeeklyLimitUSD: &limit, MonthlyLimitUSD: &limit,
		DailyWindowStart: &day, WeeklyWindowStart: &week, MonthlyWindowStart: &now,
		DailyUsageUSD: 1, WeeklyUsageUSD: 1, MonthlyUsageUSD: 1,
	}}
	cfg := &config.Config{}
	cfg.Billing.InflightReservation.Enabled = true
	cfg.Billing.InflightReservation.MaxReservationUSD = 1
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, &fakeQuotaRepo{})
	t.Cleanup(svc.Stop)
	user := &User{ID: 1}
	key := &APIKey{ID: 2, Group: &Group{Platform: PlatformGemini}}
	ctx := WithResolvedTargetPlatform(context.Background(), PlatformOpenAI)
	first, err := svc.ReserveFixedCharge(ctx, user, key, key.Group, nil, 6)
	require.NoError(t, err)
	require.Equal(t, 6.0, first.Amount(), "fixed charges are not capped")
	require.Len(t, cache.reserved["platform:1:openai"], 1, "use the effective platform")
	_, err = svc.ReserveFixedCharge(ctx, user, key, key.Group, nil, 4)
	require.ErrorIs(t, err, ErrUserPlatformDailyQuotaExhausted)
	require.Len(t, cache.reserved["platform:1:openai"], 1, "rejection removes its own hold")
	done := first.Acquire()
	first.HandlerDone()
	require.Len(t, cache.reserved["platform:1:openai"], 1, "billing task retains the hold")
	done()
	for _, holds := range cache.reserved {
		require.Empty(t, holds)
	}
	second, err := svc.ReserveFixedCharge(ctx, user, key, key.Group, nil, 9)
	require.NoError(t, err, "the exact remaining amount is allowed")
	second.HandlerDone()
	cache.quota.DailyUsageUSD = 0
	cache.quota.WeeklyUsageUSD = 9
	_, err = svc.ReserveFixedCharge(ctx, user, key, key.Group, nil, 2)
	require.ErrorIs(t, err, ErrUserPlatformWeeklyQuotaExhausted)
	cache.quota.WeeklyUsageUSD = 0
	cache.quota.MonthlyUsageUSD = 9
	_, err = svc.ReserveFixedCharge(ctx, user, key, key.Group, nil, 2)
	require.ErrorIs(t, err, ErrUserPlatformMonthlyQuotaExhausted)
	key.Group.SubscriptionType = SubscriptionTypeSubscription
	cache.sub = &SubscriptionCacheData{Status: SubscriptionStatusActive}
	subscription, err := svc.ReserveFixedCharge(ctx, user, key, key.Group, &UserSubscription{ID: 4}, 20)
	require.NoError(t, err, "subscription billing remains exempt from platform quota")
	subscription.HandlerDone()
}

func TestGeminiCachedContentUpstreamModelResourceRejectsMappedPaths(t *testing.T) {
	for _, kind := range []string{AccountTypeAPIKey, AccountTypeServiceAccount} {
		account := &Account{Type: kind, Credentials: map[string]any{"project_id": "p"}}
		for _, model := range []string{"gemini-pro/../other", "gemini-pro?key=x", "gemini-pro:generateContent", ""} {
			_, err := GeminiCachedContentUpstreamModelResource(account, model)
			require.Error(t, err, model)
		}
		_, err := GeminiCachedContentUpstreamModelResource(account, "gemini-2.5-pro")
		require.NoError(t, err)
	}
}
