//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type autoAdvanceMockUserSubRepo struct {
	UserSubscriptionRepository
	sub *UserSubscription
}

func (m *autoAdvanceMockUserSubRepo) GetByID(ctx context.Context, id int64) (*UserSubscription, error) {
	if m.sub != nil && m.sub.ID == id {
		copied := *m.sub
		return &copied, nil
	}
	return nil, ErrSubscriptionNotFound
}

func (m *autoAdvanceMockUserSubRepo) GetByIDForUpdate(ctx context.Context, id int64) (*UserSubscription, error) {
	return m.GetByID(ctx, id)
}

func (m *autoAdvanceMockUserSubRepo) Update(ctx context.Context, sub *UserSubscription) error {
	m.sub = sub
	return nil
}

type autoAdvanceMockGroupRepo struct {
	GroupRepository
	group *Group
}

func (m *autoAdvanceMockGroupRepo) GetByID(ctx context.Context, id int64) (*Group, error) {
	if m.group != nil && m.group.ID == id {
		return m.group, nil
	}
	return nil, nil
}

func TestUpdateAutoQuotaAdvance_Permissions(t *testing.T) {
	ctx := context.Background()

	t.Run("group not allowed rejects enable", func(t *testing.T) {
		sub := &UserSubscription{
			ID:      1,
			UserID:  10,
			GroupID: 100,
			Status:  SubscriptionStatusActive,
			Group: &Group{
				ID:                100,
				AllowQuotaAdvance: true,
				AutoQuotaAdvance:  false, // 管理端未开启
			},
		}
		repo := &autoAdvanceMockUserSubRepo{sub: sub}
		svc := &SubscriptionService{
			userSubRepo: repo,
		}

		_, err := svc.UpdateAutoQuotaAdvance(ctx, 10, 1, true)
		require.ErrorIs(t, err, ErrAutoQuotaAdvanceNotAllowed)
	})

	t.Run("group allow_quota_advance false rejects enable", func(t *testing.T) {
		sub := &UserSubscription{
			ID:      1,
			UserID:  10,
			GroupID: 100,
			Status:  SubscriptionStatusActive,
			Group: &Group{
				ID:                100,
				AllowQuotaAdvance: false,
				AutoQuotaAdvance:  true,
			},
		}
		repo := &autoAdvanceMockUserSubRepo{sub: sub}
		svc := &SubscriptionService{
			userSubRepo: repo,
		}

		_, err := svc.UpdateAutoQuotaAdvance(ctx, 10, 1, true)
		require.ErrorIs(t, err, ErrAutoQuotaAdvanceNotAllowed)
	})

	t.Run("both allowed permits enable", func(t *testing.T) {
		sub := &UserSubscription{
			ID:      1,
			UserID:  10,
			GroupID: 100,
			Status:  SubscriptionStatusActive,
			Group: &Group{
				ID:                100,
				AllowQuotaAdvance: true,
				AutoQuotaAdvance:  true,
			},
		}
		repo := &autoAdvanceMockUserSubRepo{sub: sub}
		svc := &SubscriptionService{
			userSubRepo: repo,
		}

		updated, err := svc.UpdateAutoQuotaAdvance(ctx, 10, 1, true)
		require.NoError(t, err)
		require.True(t, updated.AutoQuotaAdvance)
		require.True(t, repo.sub.AutoQuotaAdvance)
	})

	t.Run("disable is always permitted", func(t *testing.T) {
		sub := &UserSubscription{
			ID:               1,
			UserID:           10,
			GroupID:          100,
			Status:           SubscriptionStatusActive,
			AutoQuotaAdvance: true,
			Group: &Group{
				ID:                100,
				AllowQuotaAdvance: false,
				AutoQuotaAdvance:  false,
			},
		}
		repo := &autoAdvanceMockUserSubRepo{sub: sub}
		svc := &SubscriptionService{
			userSubRepo: repo,
		}

		updated, err := svc.UpdateAutoQuotaAdvance(ctx, 10, 1, false)
		require.NoError(t, err)
		require.False(t, updated.AutoQuotaAdvance)
		require.False(t, repo.sub.AutoQuotaAdvance)
	})

	t.Run("other user cannot update", func(t *testing.T) {
		sub := &UserSubscription{
			ID:      1,
			UserID:  10,
			GroupID: 100,
			Status:  SubscriptionStatusActive,
			Group: &Group{
				ID:                100,
				AllowQuotaAdvance: true,
				AutoQuotaAdvance:  true,
			},
		}
		repo := &autoAdvanceMockUserSubRepo{sub: sub}
		svc := &SubscriptionService{
			userSubRepo: repo,
		}

		_, err := svc.UpdateAutoQuotaAdvance(ctx, 999, 1, true)
		require.ErrorIs(t, err, ErrSubscriptionNotFound)
	})
}

func TestTryAutoAdvanceQuotaCycle(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()

	t.Run("switches disabled returns nil", func(t *testing.T) {
		sub := exhaustedQuotaSubscription(now)
		sub.AutoQuotaAdvance = false
		group := sub.Group
		group.AllowQuotaAdvance = true
		group.AutoQuotaAdvance = true

		svc := &SubscriptionService{
			now: func() time.Time { return now },
		}

		res, err := svc.TryAutoAdvanceQuotaCycle(ctx, sub, group)
		require.NoError(t, err)
		require.Nil(t, res)
	})

	t.Run("group switch disabled returns nil", func(t *testing.T) {
		sub := exhaustedQuotaSubscription(now)
		sub.AutoQuotaAdvance = true
		group := sub.Group
		group.AllowQuotaAdvance = true
		group.AutoQuotaAdvance = false

		svc := &SubscriptionService{
			now: func() time.Time { return now },
		}

		res, err := svc.TryAutoAdvanceQuotaCycle(ctx, sub, group)
		require.NoError(t, err)
		require.Nil(t, res)
	})

	t.Run("multiple windows eligible returns ErrQuotaAdvanceMultipleWindows", func(t *testing.T) {
		sub := exhaustedQuotaSubscription(now)
		sub.AutoQuotaAdvance = true
		group := sub.Group
		group.AllowQuotaAdvance = true
		group.AutoQuotaAdvance = true
		// 日和周均达到门槛
		sub.DailyUsageUSD = *group.DailyLimitUSD - 0.5
		sub.WeeklyUsageUSD = *group.WeeklyLimitUSD - 0.5
		sub.MonthlyUsageUSD = 0

		svc := &SubscriptionService{
			now: func() time.Time { return now },
		}

		res, err := svc.TryAutoAdvanceQuotaCycle(ctx, sub, group)
		require.ErrorIs(t, err, ErrQuotaAdvanceMultipleWindows)
		require.Nil(t, res)
	})

	t.Run("single window eligible but calculation would expire subscription", func(t *testing.T) {
		client := newPaymentOrderLifecycleTestClient(t)
		repo := newTermLockingUserSubRepo()
		sub := exhaustedQuotaSubscription(now)
		sub.AutoQuotaAdvance = true
		group := sub.Group
		group.AllowQuotaAdvance = true
		group.AutoQuotaAdvance = true
		// 仅日窗口超限，需扣减 20 小时，但到期时间仅剩 10 小时
		sub.DailyUsageUSD = *group.DailyLimitUSD
		sub.WeeklyUsageUSD = 0
		sub.MonthlyUsageUSD = 0
		sub.ExpiresAt = now.Add(10 * time.Hour)
		repo.seed(sub)

		svc := &SubscriptionService{
			userSubRepo: repo,
			entClient:   client,
			now:         func() time.Time { return now },
		}

		res, err := svc.TryAutoAdvanceQuotaCycle(ctx, sub, group)
		require.Error(t, err)
		require.Equal(t, "QUOTA_ADVANCE_WOULD_EXPIRE", infraerrors.Reason(err))
		require.Nil(t, res)
	})

	t.Run("single window eligible and advance succeeds", func(t *testing.T) {
		client := newPaymentOrderLifecycleTestClient(t)
		repo := newTermLockingUserSubRepo()
		sub := exhaustedQuotaSubscription(now)
		sub.AutoQuotaAdvance = true
		group := sub.Group
		group.AllowQuotaAdvance = true
		group.AutoQuotaAdvance = true
		// 仅日窗口超限，需扣减 20 小时，到期时间 40 天
		sub.DailyUsageUSD = *group.DailyLimitUSD
		sub.WeeklyUsageUSD = 0
		sub.MonthlyUsageUSD = 0
		originalExpiry := sub.ExpiresAt
		repo.seed(sub)

		svc := &SubscriptionService{
			userSubRepo: repo,
			entClient:   client,
			now:         func() time.Time { return now },
		}

		res, err := svc.TryAutoAdvanceQuotaCycle(ctx, sub, group)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, float64(0), res.DailyUsageUSD)
		require.Equal(t, now, *res.DailyWindowStart)
		require.Equal(t, originalExpiry.Add(-20*time.Hour), res.ExpiresAt)
		require.True(t, res.AutoQuotaAdvance)
	})
}

// An unlocked read represents a request racing a renewal/meter/reset commit.
// The locked read must observe those committed changes before toggling.
type staleAutoAdvanceReadRepo struct {
	*termLockingUserSubRepo
	stale *UserSubscription
}

func (r *staleAutoAdvanceReadRepo) GetByID(ctx context.Context, id int64) (*UserSubscription, error) {
	if dbent.TxFromContext(ctx) == nil {
		copied := *r.stale
		return &copied, nil
	}
	return r.termLockingUserSubRepo.GetByID(ctx, id)
}

func TestUpdateAutoQuotaAdvance_PreservesConcurrentSubscriptionChanges(t *testing.T) {
	now := time.Now()
	stale := exhaustedQuotaSubscription(now)
	stale.Group.AutoQuotaAdvance = true
	current := *stale
	current.ExpiresAt = now.Add(60 * 24 * time.Hour)
	current.DailyUsageUSD = 3
	current.DailyWindowStart = &now
	base := newTermLockingUserSubRepo()
	base.seed(&current)
	repo := &staleAutoAdvanceReadRepo{termLockingUserSubRepo: base, stale: stale}
	svc := &SubscriptionService{userSubRepo: repo, entClient: newPaymentOrderLifecycleTestClient(t)}

	updated, err := svc.UpdateAutoQuotaAdvance(context.Background(), stale.UserID, stale.ID, true)

	require.NoError(t, err)
	require.Equal(t, current.ExpiresAt, updated.ExpiresAt)
	require.Equal(t, float64(3), updated.DailyUsageUSD)
	require.Equal(t, now, *updated.DailyWindowStart)
	stored, ok := base.lookupByID(stale.ID)
	require.True(t, ok)
	require.True(t, stored.AutoQuotaAdvance)
	require.Equal(t, current.ExpiresAt, stored.ExpiresAt)
	assertLockedReadDiscipline(t, base.snapshotEvents(), "GetByIDForUpdate", "GetByID")
}

func TestUpdateAutoQuotaAdvance_InvalidatesCachesAfterOuterCommit(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)
	repo := newTermLockingUserSubRepo()
	repo.updateVersion = 1000
	sub := exhaustedQuotaSubscription(time.Now())
	sub.AutoQuotaAdvance = true
	repo.seed(sub)
	cache := &redeemSubscriptionCacheStub{}
	l1 := &subscriptionCacheL1Tracker{}
	svc := &SubscriptionService{userSubRepo: repo, entClient: client,
		billingCacheService: &BillingCacheService{cache: cache}, subCacheL1: l1}
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	_, err = svc.UpdateAutoQuotaAdvance(dbent.NewTxContext(ctx, tx), sub.UserID, sub.ID, false)

	require.NoError(t, err)
	require.Zero(t, l1.dels.Load(), "cache invalidation must wait for commit")
	require.Zero(t, cache.versionedInvalidations.Load())
	require.NoError(t, tx.Commit())
	require.Equal(t, int32(1), l1.dels.Load())
	require.Equal(t, int32(1), cache.versionedInvalidations.Load())
	require.Equal(t, int32(1), cache.publications.Load())
}

func TestTryAutoAdvanceQuotaCycle_RevalidatesLockedState(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name   string
		mutate func(*UserSubscription)
	}{
		{"user disabled", func(s *UserSubscription) { s.AutoQuotaAdvance = false }},
		{"admin disabled auto", func(s *UserSubscription) { s.Group.AutoQuotaAdvance = false }},
		{"admin disabled manual", func(s *UserSubscription) { s.Group.AllowQuotaAdvance = false }},
		{"usage below limit but within manual grace", func(s *UserSubscription) { s.DailyUsageUSD = 9.5 }},
		{"window replaced and exhausted again", func(s *UserSubscription) { s.DailyWindowStart = &now }},
		{"window naturally expired", func(s *UserSubscription) { start := now.Add(-25 * time.Hour); s.DailyWindowStart = &start }},
		{"second window within manual grace", func(s *UserSubscription) { s.WeeklyUsageUSD = 69.5 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stale := exhaustedQuotaSubscription(now)
			stale.AutoQuotaAdvance, stale.Group.AutoQuotaAdvance = true, true
			stale.WeeklyUsageUSD, stale.MonthlyUsageUSD = 0, 0
			current := *stale
			group := *stale.Group
			current.Group = &group
			tt.mutate(&current)
			repo := newTermLockingUserSubRepo()
			repo.seed(&current)
			svc := &SubscriptionService{userSubRepo: repo, entClient: newPaymentOrderLifecycleTestClient(t), now: func() time.Time { return now }}

			_, _ = svc.TryAutoAdvanceQuotaCycle(context.Background(), stale, stale.Group)

			stored, ok := repo.lookupByID(stale.ID)
			require.True(t, ok)
			require.Equal(t, current.ExpiresAt, stored.ExpiresAt, "stale preflight must not deduct subscription time")
			require.Equal(t, current.DailyUsageUSD, stored.DailyUsageUSD)
			require.Equal(t, *current.DailyWindowStart, *stored.DailyWindowStart)
		})
	}
}

func TestTryAutoAdvanceQuotaCycle_StaleSmallLimitSnapshotDeductsOnlyOnce(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	for _, window := range []struct {
		name     string
		deducted time.Duration
	}{
		{"daily", 20 * time.Hour},
		{"weekly", 5 * 24 * time.Hour},
		{"monthly", 20 * 24 * time.Hour},
	} {
		t.Run(window.name, func(t *testing.T) {
			for _, limit := range []float64{0.5, 1} {
				sub := exhaustedQuotaSubscription(now)
				sub.AutoQuotaAdvance, sub.Group.AutoQuotaAdvance = true, true
				sub.DailyUsageUSD, sub.WeeklyUsageUSD, sub.MonthlyUsageUSD = 0, 0, 0
				switch window.name {
				case "daily":
					sub.Group.DailyLimitUSD, sub.DailyUsageUSD = &limit, limit
				case "weekly":
					sub.Group.WeeklyLimitUSD, sub.WeeklyUsageUSD = &limit, limit
				case "monthly":
					sub.Group.MonthlyLimitUSD, sub.MonthlyUsageUSD = &limit, limit
				}
				repo := newTermLockingUserSubRepo()
				repo.updateVersion = now.UnixNano()
				repo.seed(sub)
				svc := &SubscriptionService{userSubRepo: repo, entClient: newPaymentOrderLifecycleTestClient(t), now: func() time.Time { return now }}

				first, err := svc.TryAutoAdvanceQuotaCycle(context.Background(), sub, sub.Group)
				require.NoError(t, err)
				require.Equal(t, sub.ExpiresAt.Add(-window.deducted), first.ExpiresAt)
				second, err := svc.TryAutoAdvanceQuotaCycle(context.Background(), sub, sub.Group)
				require.NoError(t, err, "a competing request may reuse the refreshed reset snapshot")
				require.Equal(t, first.ExpiresAt, second.ExpiresAt)
				stored, ok := repo.lookupByID(sub.ID)
				require.True(t, ok)
				require.Equal(t, first.ExpiresAt, stored.ExpiresAt)
			}
		})
	}
}

func TestExtendSubscription_PreservesAutoQuotaAdvance(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)
	repo := newTermLockingUserSubRepo()

	sub := exhaustedQuotaSubscription(now)
	sub.AutoQuotaAdvance = true
	repo.seed(sub)

	svc := &SubscriptionService{
		userSubRepo: repo,
		entClient:   client,
		now:         func() time.Time { return now },
	}

	extended, err := svc.ExtendSubscription(ctx, sub.ID, 30)
	require.NoError(t, err)
	require.NotNil(t, extended)
	require.True(t, extended.AutoQuotaAdvance, "auto_quota_advance must be preserved after extending subscription")
}
