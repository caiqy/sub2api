//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestCalculateQuotaCycleAdvance_ResetsOnlySingleExhaustedWindow(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	for _, tt := range []struct {
		name      string
		selection QuotaWindowSelection
		deducted  time.Duration
	}{
		{name: "daily", selection: QuotaWindowSelection{Daily: true}, deducted: 20 * time.Hour},
		{name: "weekly", selection: QuotaWindowSelection{Weekly: true}, deducted: 5 * 24 * time.Hour},
		{name: "monthly", selection: QuotaWindowSelection{Monthly: true}, deducted: 20 * 24 * time.Hour},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sub := exhaustedQuotaSubscription(now)
			sub.DailyUsageUSD, sub.WeeklyUsageUSD, sub.MonthlyUsageUSD = 1, 1, 1
			if tt.selection.Daily {
				sub.DailyUsageUSD = *sub.Group.DailyLimitUSD - 1
			}
			if tt.selection.Weekly {
				sub.WeeklyUsageUSD = *sub.Group.WeeklyLimitUSD - 1
			}
			if tt.selection.Monthly {
				sub.MonthlyUsageUSD = *sub.Group.MonthlyLimitUSD - 1
			}
			originalExpiry := sub.ExpiresAt
			wantDailyUsage, wantWeeklyUsage, wantMonthlyUsage := sub.DailyUsageUSD, sub.WeeklyUsageUSD, sub.MonthlyUsageUSD
			wantDailyStart, wantWeeklyStart, wantMonthlyStart := *sub.DailyWindowStart, *sub.WeeklyWindowStart, *sub.MonthlyWindowStart
			if tt.selection.Daily {
				wantDailyUsage, wantDailyStart = 0, now
			}
			if tt.selection.Weekly {
				wantWeeklyUsage, wantWeeklyStart = 0, now
			}
			if tt.selection.Monthly {
				wantMonthlyUsage, wantMonthlyStart = 0, now
			}

			result, err := calculateQuotaCycleAdvance(sub, tt.selection, now)

			require.NoError(t, err)
			require.Equal(t, tt.deducted, result.DeductedDuration)
			require.Equal(t, originalExpiry.Add(-tt.deducted), result.Subscription.ExpiresAt)
			require.Equal(t, wantDailyUsage, result.Subscription.DailyUsageUSD)
			require.Equal(t, wantWeeklyUsage, result.Subscription.WeeklyUsageUSD)
			require.Equal(t, wantMonthlyUsage, result.Subscription.MonthlyUsageUSD)
			require.Equal(t, wantDailyStart, *result.Subscription.DailyWindowStart)
			require.Equal(t, wantWeeklyStart, *result.Subscription.WeeklyWindowStart)
			require.Equal(t, wantMonthlyStart, *result.Subscription.MonthlyWindowStart)
			require.Equal(t, originalExpiry, sub.ExpiresAt, "calculation must not mutate the database snapshot")
		})
	}
}

func TestCalculateQuotaCycleAdvance_RejectsMultipleExhaustedWindows(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	for _, selection := range []QuotaWindowSelection{
		{Daily: true, Weekly: true, Monthly: true},
		{Daily: true},
	} {
		sub := exhaustedQuotaSubscription(now)
		sub.DailyUsageUSD = *sub.Group.DailyLimitUSD - 1
		sub.WeeklyUsageUSD = *sub.Group.WeeklyLimitUSD - 1
		sub.MonthlyUsageUSD = *sub.Group.MonthlyLimitUSD - 1
		result, err := calculateQuotaCycleAdvance(sub, selection, now)

		require.Equal(t, "QUOTA_ADVANCE_MULTIPLE_WINDOWS", infraerrors.Reason(err))
		require.Nil(t, result)
		require.Equal(t, now.Add(40*24*time.Hour), sub.ExpiresAt)
		require.Equal(t, float64(9), sub.DailyUsageUSD)
		require.Equal(t, float64(69), sub.WeeklyUsageUSD)
		require.Equal(t, float64(299), sub.MonthlyUsageUSD)
	}
}

func TestCalculateQuotaCycleAdvance_AllowsZeroUsageForAtMostOneDollarLimit(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	for _, limit := range []float64{1, 0.5} {
		sub := exhaustedQuotaSubscription(now)
		sub.Group.MonthlyLimitUSD = &limit
		sub.DailyUsageUSD = 0
		sub.WeeklyUsageUSD = 0
		sub.MonthlyUsageUSD = 0

		result, err := calculateQuotaCycleAdvance(sub, QuotaWindowSelection{Monthly: true}, now)

		require.NoError(t, err)
		require.Zero(t, result.Subscription.MonthlyUsageUSD)
	}
}

func TestCalculateQuotaCycleAdvance_AllowsIndependentDecimalBoundary(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	limit := 1.1
	sub := exhaustedQuotaSubscription(now)
	sub.Group.DailyLimitUSD = &limit
	sub.DailyUsageUSD = 0.1
	sub.WeeklyUsageUSD = 0
	sub.MonthlyUsageUSD = 0

	result, err := calculateQuotaCycleAdvance(sub, QuotaWindowSelection{Daily: true}, now)

	require.NoError(t, err)
	require.Zero(t, result.Subscription.DailyUsageUSD)
}

func TestCalculateQuotaCycleAdvance_AllowsSharedLargeMagnitudeBoundary(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	limit := 1_000_000.0
	sub := exhaustedQuotaSubscription(now)
	sub.Group.DailyLimitUSD = &limit
	sub.DailyUsageUSD = 999_998.9999999994
	sub.WeeklyUsageUSD = 0
	sub.MonthlyUsageUSD = 0

	result, err := calculateQuotaCycleAdvance(sub, QuotaWindowSelection{Daily: true}, now)

	require.NoError(t, err)
	require.Zero(t, result.Subscription.DailyUsageUSD)
}

func TestAdvanceQuotaCycle_RejectsTwoExhaustedWindowsBeforeUpdate(t *testing.T) {
	now := time.Now()
	client := newPaymentOrderLifecycleTestClient(t)
	repo := newTermLockingUserSubRepo()
	sub := exhaustedQuotaSubscription(now)
	sub.MonthlyUsageUSD = 0
	originalDailyStart, originalWeeklyStart := *sub.DailyWindowStart, *sub.WeeklyWindowStart
	repo.seed(sub)
	svc := &SubscriptionService{userSubRepo: repo, entClient: client}

	result, err := svc.AdvanceQuotaCycle(context.Background(), sub.UserID, sub.ID, QuotaWindowSelection{Daily: true, Weekly: true})

	require.Nil(t, result)
	require.Equal(t, http.StatusConflict, infraerrors.Code(err))
	require.Equal(t, "QUOTA_ADVANCE_MULTIPLE_WINDOWS", infraerrors.Reason(err))
	stored, ok := repo.lookupByID(sub.ID)
	require.True(t, ok)
	require.Equal(t, sub.ExpiresAt, stored.ExpiresAt)
	require.Equal(t, sub.DailyUsageUSD, stored.DailyUsageUSD)
	require.Equal(t, sub.WeeklyUsageUSD, stored.WeeklyUsageUSD)
	require.Equal(t, originalDailyStart, *stored.DailyWindowStart)
	require.Equal(t, originalWeeklyStart, *stored.WeeklyWindowStart)
	events := repo.snapshotEvents()
	require.Len(t, events, 1)
	require.Equal(t, "GetByIDForUpdate", events[0].name)
	require.True(t, events[0].inTx)
}

func TestAdvanceQuotaCycle_UsesUpdatedObjectWithoutRefreshRead(t *testing.T) {
	client := newPaymentOrderLifecycleTestClient(t)
	repo := newTermLockingUserSubRepo()
	repo.updateVersion = 1000
	sub := exhaustedQuotaSubscription(time.Now())
	sub.WeeklyUsageUSD = 0
	sub.MonthlyUsageUSD = 0
	sub.User = &User{ID: sub.UserID}
	sub.Group.ID = sub.GroupID
	assignedBy := int64(30)
	sub.AssignedBy = &assignedBy
	sub.AssignedByUser = &User{ID: assignedBy}
	repo.seed(sub)
	svc := &SubscriptionService{userSubRepo: repo, entClient: client}

	result, err := svc.AdvanceQuotaCycle(context.Background(), sub.UserID, sub.ID, QuotaWindowSelection{Daily: true})

	require.NoError(t, err)
	require.Equal(t, time.Unix(0, 1000), result.Subscription.UpdatedAt)
	require.Equal(t, sub.UserID, result.Subscription.User.ID)
	require.Equal(t, sub.GroupID, result.Subscription.Group.ID)
	require.Equal(t, assignedBy, result.Subscription.AssignedByUser.ID)
	require.Equal(t, []termReadWriteEvent{
		{name: "GetByIDForUpdate", inTx: true},
		{name: "Update", inTx: true},
	}, repo.snapshotEvents())
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
				sub.DailyUsageUSD = 8.9999999999
				sub.WeeklyUsageUSD = 0
				sub.MonthlyUsageUSD = 0
			},
			selection: QuotaWindowSelection{Daily: true},
			wantErr:   ErrQuotaAdvanceStateChanged,
		},
		{
			name: "daily window already reached its normal reset",
			mutate: func(sub *UserSubscription) {
				start := now.Add(-25 * time.Hour)
				sub.DailyWindowStart = &start
				sub.WeeklyUsageUSD = 0
				sub.MonthlyUsageUSD = 0
			},
			selection: QuotaWindowSelection{Daily: true},
			wantErr:   ErrQuotaAdvanceStateChanged,
		},
		{
			name: "one-time daily quota has no next cycle",
			mutate: func(sub *UserSubscription) {
				sub.StartsAt = now.Add(-time.Hour)
				sub.ExpiresAt = now.Add(23 * time.Hour)
				sub.WeeklyUsageUSD = 0
				sub.MonthlyUsageUSD = 0
			},
			selection: QuotaWindowSelection{Daily: true},
			wantErr:   ErrQuotaAdvanceOneTimeWindow,
		},
		{
			name: "multiple eligible windows take precedence over one-time selection",
			mutate: func(sub *UserSubscription) {
				sub.StartsAt = now.Add(-time.Hour)
				sub.ExpiresAt = now.Add(23 * time.Hour)
			},
			selection: QuotaWindowSelection{Daily: true},
			wantErr:   ErrQuotaAdvanceMultipleWindows,
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
				sub.WeeklyUsageUSD = 0
				sub.MonthlyUsageUSD = 0
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
