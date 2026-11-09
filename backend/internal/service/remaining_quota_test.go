package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRemainingSubscriptionQuota(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	ten, twenty, zero := 10.0, 20.0, 0.0
	tests := []struct {
		name      string
		sub       UserSubscription
		want      float64
		unlimited bool
	}{
		{"daily expiry boundary excludes reset", UserSubscription{StartsAt: now.Add(-day), ExpiresAt: now.Add(2 * day), DailyWindowStart: &now, DailyUsageUSD: 3, Group: &Group{DailyLimitUSD: &ten}}, 17, false},
		{"stale windows reset without mutation", UserSubscription{StartsAt: now.Add(-3 * day), ExpiresAt: now.Add(day), DailyWindowStart: quotaTime(now.Add(-2 * day)), DailyUsageUSD: 9, Group: &Group{DailyLimitUSD: &ten}}, 10, false},
		{"legacy midnight uses actual opening time", UserSubscription{StartsAt: now.Add(-23 * time.Hour), ExpiresAt: now.Add(13 * time.Hour), DailyWindowStart: quotaTime(startOfDay(now.Add(-23 * time.Hour))), DailyUsageUSD: 10, Group: &Group{DailyLimitUSD: &ten}}, 10, false},
		{"advance flags do not duplicate future quota", UserSubscription{StartsAt: now.Add(-day), ExpiresAt: now.Add(2 * day), DailyWindowStart: &now, DailyUsageUSD: 3, AutoQuotaAdvance: true, Group: &Group{DailyLimitUSD: &ten, AllowQuotaAdvance: true, AutoQuotaAdvance: true}}, 17, false},
		{"one day grant never resets", UserSubscription{StartsAt: now.Add(-12 * time.Hour), ExpiresAt: now.Add(12 * time.Hour), DailyWindowStart: quotaTime(now.Add(-2 * day)), DailyUsageUSD: 3, Group: &Group{DailyLimitUSD: &ten}}, 7, false},
		{"misaligned limits cannot sum independent budgets", UserSubscription{StartsAt: now.Add(-8 * day), ExpiresAt: now.Add(2 * day), DailyWindowStart: &now, WeeklyWindowStart: quotaTime(now.Add(-6 * day)), DailyUsageUSD: 10, WeeklyUsageUSD: 0, Group: &Group{DailyLimitUSD: &ten, WeeklyLimitUSD: &twenty}}, 10, false},
		{"unactivated windows anchored to subscription", UserSubscription{StartsAt: now.Add(-2 * day), ExpiresAt: now.Add(2 * day), Group: &Group{DailyLimitUSD: &ten}}, 20, false},
		{"monthly resets before expiry", UserSubscription{StartsAt: now.Add(-day), ExpiresAt: now.Add(60 * day), MonthlyWindowStart: &now, MonthlyUsageUSD: 5, Group: &Group{MonthlyLimitUSD: &twenty}}, 35, false},
		{"zero and missing limits are unlimited", UserSubscription{StartsAt: now, ExpiresAt: now.Add(day), Group: &Group{DailyLimitUSD: &zero, RateMultiplier: 2}}, 0, true},
		{"overdrawn window", UserSubscription{StartsAt: now.Add(-day), ExpiresAt: now.Add(time.Hour), DailyWindowStart: &now, DailyUsageUSD: 12, Group: &Group{DailyLimitUSD: &ten}}, 0, false},
		{"missing group", UserSubscription{ExpiresAt: now.Add(day)}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.sub.Status = SubscriptionStatusActive
			before := tt.sub
			got, unlimited := remainingSubscriptionQuota(&tt.sub, now)
			require.InDelta(t, tt.want, got, 1e-9)
			require.Equal(t, tt.unlimited, unlimited)
			require.Equal(t, before, tt.sub)
		})
	}
	sub := tests[0].sub
	sub.Status = SubscriptionStatusRevoked
	got, unlimited := remainingSubscriptionQuota(&sub, now)
	require.Zero(t, got)
	require.False(t, unlimited)
	sub.Status = SubscriptionStatusActive
	sub.ExpiresAt = now
	got, unlimited = remainingSubscriptionQuota(&sub, now)
	require.Zero(t, got)
	require.False(t, unlimited)
}
func quotaTime(t time.Time) *time.Time { return &t }

type remainingQuotaTestRepo struct {
	UserRepository
	excluded []int64
}

func (r *remainingQuotaTestRepo) ReadRemainingQuota(_ context.Context, excluded []int64, now time.Time) (float64, []UserSubscription, error) {
	r.excluded = excluded
	limit := 10.0
	return -2, []UserSubscription{{Status: SubscriptionStatusActive, StartsAt: now, ExpiresAt: now.Add(time.Hour), Group: &Group{DailyLimitUSD: &limit, RateMultiplier: 3}}, {Status: SubscriptionStatusActive, ExpiresAt: now.Add(time.Hour), Group: &Group{}}}, nil
}
func TestGetRemainingQuota(t *testing.T) {
	repo := &remainingQuotaTestRepo{}
	result, err := (&adminServiceImpl{userRepo: repo}).GetRemainingQuota(context.Background(), []int64{2, 3})
	require.NoError(t, err)
	require.Equal(t, []int64{2, 3}, repo.excluded)
	require.Equal(t, &RemainingQuotaSummary{Balance: -2, Subscription: 10, Total: 8, UnlimitedSubscriptions: 1}, result)
}
