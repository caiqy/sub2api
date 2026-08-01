//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCalculateQuotaCycleAdvance_DeductsRemainingDailyTime(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	sub := exhaustedQuotaSubscription(now)
	originalExpiry := sub.ExpiresAt

	result, err := calculateQuotaCycleAdvance(sub, QuotaWindowSelection{Daily: true}, now)

	require.NoError(t, err)
	require.Equal(t, 20*time.Hour, result.DeductedDuration)
	require.Equal(t, originalExpiry.Add(-20*time.Hour), result.Subscription.ExpiresAt)
	require.Equal(t, float64(0), result.Subscription.DailyUsageUSD)
	require.Equal(t, now, *result.Subscription.DailyWindowStart)
	require.Equal(t, originalExpiry, sub.ExpiresAt, "calculation must not mutate the database snapshot")
}

func TestCalculateQuotaCycleAdvance_UsesLongestSelectedRemainingTime(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	sub := exhaustedQuotaSubscription(now)
	originalMonthlyStart := *sub.MonthlyWindowStart

	result, err := calculateQuotaCycleAdvance(sub, QuotaWindowSelection{Daily: true, Weekly: true}, now)

	require.NoError(t, err)
	require.Equal(t, 5*24*time.Hour, result.DeductedDuration)
	require.Equal(t, sub.ExpiresAt.Add(-5*24*time.Hour), result.Subscription.ExpiresAt)
	require.Equal(t, float64(0), result.Subscription.DailyUsageUSD)
	require.Equal(t, float64(0), result.Subscription.WeeklyUsageUSD)
	require.Equal(t, sub.MonthlyUsageUSD, result.Subscription.MonthlyUsageUSD)
	require.Equal(t, originalMonthlyStart, *result.Subscription.MonthlyWindowStart)
}

func TestCalculateQuotaCycleAdvance_RejectsInvalidState(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		mutate    func(*UserSubscription)
		selection QuotaWindowSelection
		wantErr   error
	}{
		{
			name:      "no window selected",
			selection: QuotaWindowSelection{},
			wantErr:   ErrQuotaAdvanceSelectionRequired,
		},
		{
			name: "selected window is not exhausted",
			mutate: func(sub *UserSubscription) {
				sub.DailyUsageUSD = 9.99
			},
			selection: QuotaWindowSelection{Daily: true},
			wantErr:   ErrQuotaAdvanceWindowNotExhausted,
		},
		{
			name: "daily window already reached its normal reset",
			mutate: func(sub *UserSubscription) {
				start := now.Add(-25 * time.Hour)
				sub.DailyWindowStart = &start
			},
			selection: QuotaWindowSelection{Daily: true},
			wantErr:   ErrQuotaAdvanceWindowNotExhausted,
		},
		{
			name: "one-time daily quota has no next cycle",
			mutate: func(sub *UserSubscription) {
				sub.StartsAt = now.Add(-time.Hour)
				sub.ExpiresAt = now.Add(23 * time.Hour)
			},
			selection: QuotaWindowSelection{Daily: true},
			wantErr:   ErrQuotaAdvanceOneTimeWindow,
		},
		{
			name: "subscription is not active",
			mutate: func(sub *UserSubscription) {
				sub.Status = SubscriptionStatusSuspended
			},
			selection: QuotaWindowSelection{Daily: true},
			wantErr:   ErrQuotaAdvanceUnavailable,
		},
		{
			name: "deduction would expire subscription",
			mutate: func(sub *UserSubscription) {
				sub.ExpiresAt = now.Add(10 * time.Hour)
			},
			selection: QuotaWindowSelection{Daily: true},
			wantErr:   ErrQuotaAdvanceWouldExpire,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := exhaustedQuotaSubscription(now)
			if tt.mutate != nil {
				tt.mutate(sub)
			}

			_, err := calculateQuotaCycleAdvance(sub, tt.selection, now)

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestAdvanceQuotaCycle_RejectsUnversionedCacheBeforeUpdate(t *testing.T) {
	client := newPaymentOrderLifecycleTestClient(t)
	repo := newTermLockingUserSubRepo()
	now := time.Now()
	sub := exhaustedQuotaSubscription(now)
	repo.seed(sub)
	svc := &SubscriptionService{
		userSubRepo:         repo,
		billingCacheService: &BillingCacheService{cache: &billingCacheWorkerStub{}},
		entClient:           client,
	}

	_, err := svc.AdvanceQuotaCycle(context.Background(), sub.UserID, sub.ID, QuotaWindowSelection{Daily: true})

	require.Error(t, err)
	require.Empty(t, repo.snapshotEvents(), "quota advance must reject an unversioned cache before reading or writing the subscription")
}

func exhaustedQuotaSubscription(now time.Time) *UserSubscription {
	dailyLimit := 10.0
	weeklyLimit := 70.0
	monthlyLimit := 300.0
	dailyStart := now.Add(-4 * time.Hour)
	weeklyStart := now.Add(-2 * 24 * time.Hour)
	monthlyStart := now.Add(-10 * 24 * time.Hour)
	return &UserSubscription{
		ID:                 1,
		UserID:             10,
		GroupID:            20,
		StartsAt:           now.Add(-10 * 24 * time.Hour),
		ExpiresAt:          now.Add(40 * 24 * time.Hour),
		Status:             SubscriptionStatusActive,
		DailyWindowStart:   &dailyStart,
		WeeklyWindowStart:  &weeklyStart,
		MonthlyWindowStart: &monthlyStart,
		DailyUsageUSD:      dailyLimit,
		WeeklyUsageUSD:     weeklyLimit,
		MonthlyUsageUSD:    monthlyLimit,
		Group: &Group{
			DailyLimitUSD:   &dailyLimit,
			WeeklyLimitUSD:  &weeklyLimit,
			MonthlyLimitUSD: &monthlyLimit,
		},
	}
}
