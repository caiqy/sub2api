//go:build unit

package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type redeemSubscriptionRepoStub struct {
	*subscriptionUserSubRepoStub
}

type postCommitReadFailureRedeemRepo struct {
	*redeemRejectRepo
}

func (r *postCommitReadFailureRedeemRepo) GetByID(context.Context, int64) (*RedeemCode, error) {
	return nil, errors.New("post-commit read failed")
}

func (r *redeemSubscriptionRepoStub) ExtendExpiry(_ context.Context, id int64, expiresAt time.Time) error {
	sub := r.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.ExpiresAt = expiresAt
	return nil
}

func (r *redeemSubscriptionRepoStub) UpdateStatus(_ context.Context, id int64, status string) error {
	sub := r.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.Status = status
	return nil
}

func (r *redeemSubscriptionRepoStub) UpdateNotes(_ context.Context, id int64, notes string) error {
	sub := r.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.Notes = notes
	return nil
}

func (r *redeemSubscriptionRepoStub) ActivateWindows(_ context.Context, id int64, start time.Time) error {
	sub := r.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.DailyWindowStart = &start
	sub.WeeklyWindowStart = &start
	sub.MonthlyWindowStart = &start
	return nil
}

func (r *redeemSubscriptionRepoStub) ResetDailyUsage(_ context.Context, id int64, _ *time.Time, start time.Time) error {
	sub := r.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.DailyUsageUSD = 0
	sub.DailyWindowStart = &start
	return nil
}

func (r *redeemSubscriptionRepoStub) ResetUsageWindows(_ context.Context, id int64, resetDaily, resetWeekly, resetMonthly bool, start time.Time) error {
	sub := r.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	if resetDaily {
		sub.DailyUsageUSD = 0
		sub.DailyWindowStart = &start
	}
	if resetWeekly {
		sub.WeeklyUsageUSD = 0
		sub.WeeklyWindowStart = &start
	}
	if resetMonthly {
		sub.MonthlyUsageUSD = 0
		sub.MonthlyWindowStart = &start
	}
	return nil
}

func (r *redeemSubscriptionRepoStub) IncrementUsage(_ context.Context, id int64, cost float64) error {
	sub := r.byID[id]
	if sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.DailyUsageUSD += cost
	sub.WeeklyUsageUSD += cost
	sub.MonthlyUsageUSD += cost
	return nil
}

type redeemSubscriptionCacheStub struct {
	billingCacheWorkerStub
	invalidations          atomic.Int32
	publications           atomic.Int32
	invalidateErr          error
	waitInvalidateDeadline bool
	publishCtxActive       atomic.Bool
	subscribed             chan struct{}
	canceled               chan struct{}
	subscribeCalls         atomic.Int32
	subscribeFailures      atomic.Int32
}

func (s *redeemSubscriptionCacheStub) InvalidateSubscriptionCache(ctx context.Context, _ int64, _ int64) error {
	s.invalidations.Add(1)
	if s.waitInvalidateDeadline {
		<-ctx.Done()
		return ctx.Err()
	}
	return s.invalidateErr
}

func (s *redeemSubscriptionCacheStub) PublishSubscriptionCacheInvalidation(ctx context.Context, _ string) error {
	s.publications.Add(1)
	s.publishCtxActive.Store(ctx.Err() == nil)
	return nil
}

func (s *redeemSubscriptionCacheStub) SubscribeSubscriptionCacheInvalidation(ctx context.Context, _ func(string)) error {
	s.subscribeCalls.Add(1)
	if s.subscribeFailures.Load() > 0 {
		s.subscribeFailures.Add(-1)
		return errors.New("initial subscribe failed")
	}
	if s.subscribed != nil {
		close(s.subscribed)
		go func() {
			<-ctx.Done()
			close(s.canceled)
		}()
	}
	return nil
}

func TestInvalidateSubscriptionCaches_PublishesWhenRedisDeleteFails(t *testing.T) {
	cache := &redeemSubscriptionCacheStub{invalidateErr: errors.New("redis delete failed")}
	svc := NewSubscriptionService(groupRepoNoop{}, userSubRepoNoop{}, &BillingCacheService{cache: cache}, nil, nil)

	err := svc.invalidateSubscriptionCaches(10, 20)

	require.Error(t, err)
	require.Equal(t, int32(1), cache.invalidations.Load())
	require.Equal(t, int32(1), cache.publications.Load(), "Redis 删除失败不能阻止跨实例 L1 失效发布")
}

func TestInvalidateSubscriptionCaches_PublishGetsIndependentTimeout(t *testing.T) {
	cache := &redeemSubscriptionCacheStub{waitInvalidateDeadline: true}
	svc := NewSubscriptionService(groupRepoNoop{}, userSubRepoNoop{}, &BillingCacheService{cache: cache}, nil, nil)

	err := svc.invalidateSubscriptionCaches(10, 20)

	require.Error(t, err)
	require.Equal(t, int32(1), cache.publications.Load())
	require.True(t, cache.publishCtxActive.Load(), "发布不能复用已被 Redis 删除耗尽的 context")
}

func TestSubscriptionServiceStop_CancelsCacheInvalidationSubscriber(t *testing.T) {
	cache := &redeemSubscriptionCacheStub{
		subscribed: make(chan struct{}),
		canceled:   make(chan struct{}),
	}
	svc := NewSubscriptionService(groupRepoNoop{}, userSubRepoNoop{}, &BillingCacheService{cache: cache}, nil, &config.Config{
		SubscriptionCache: config.SubscriptionCacheConfig{L1Size: 16, L1TTLSeconds: 60},
	})
	<-cache.subscribed

	svc.Stop()

	select {
	case <-cache.canceled:
	case <-time.After(time.Second):
		t.Fatal("SubscriptionService.Stop 未取消缓存失效订阅")
	}
}

func TestSubscriptionCacheInvalidationSubscriber_RetriesInitialFailure(t *testing.T) {
	cache := &redeemSubscriptionCacheStub{
		subscribed: make(chan struct{}),
		canceled:   make(chan struct{}),
	}
	cache.subscribeFailures.Store(1)
	svc := NewSubscriptionService(groupRepoNoop{}, userSubRepoNoop{}, &BillingCacheService{cache: cache}, nil, &config.Config{
		SubscriptionCache: config.SubscriptionCacheConfig{L1Size: 16, L1TTLSeconds: 60},
	})
	t.Cleanup(svc.Stop)

	select {
	case <-cache.subscribed:
	case <-time.After(2 * time.Second):
		t.Fatal("初次订阅失败后未重试")
	}
	require.Equal(t, int32(2), cache.subscribeCalls.Load())
}

func TestRevokeAndRestore_DoNotReportFailureAfterDatabaseMutation(t *testing.T) {
	t.Run("revoke", func(t *testing.T) {
		repo := &revokeCacheUserSubRepoStub{sub: &UserSubscription{
			ID: 1, UserID: 10, GroupID: 20, Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(time.Hour),
		}}
		cache := &redeemSubscriptionCacheStub{invalidateErr: errors.New("redis delete failed")}
		svc := NewSubscriptionService(groupRepoNoop{}, repo, &BillingCacheService{cache: cache}, nil, nil)

		err := svc.RevokeSubscription(context.Background(), 1)

		require.NoError(t, err)
		require.True(t, repo.deleted)
	})

	t.Run("restore", func(t *testing.T) {
		deletedAt := time.Now().Add(-time.Hour)
		repo := &restoreUserSubRepoStub{sub: &UserSubscription{
			ID: 1, UserID: 10, GroupID: 20, Status: SubscriptionStatusActive,
			ExpiresAt: time.Now().Add(time.Hour), DeletedAt: &deletedAt,
		}}
		cache := &redeemSubscriptionCacheStub{invalidateErr: errors.New("redis delete failed")}
		svc := NewSubscriptionService(groupRepoNoop{}, repo, &BillingCacheService{cache: cache}, nil, nil)

		restored, err := svc.RestoreSubscription(context.Background(), 1)

		require.NoError(t, err)
		require.NotNil(t, restored)
		require.Nil(t, restored.DeletedAt)
	})
}

func TestRedeem_DoesNotFailAfterCommitWhenRefreshFails(t *testing.T) {
	ctx := context.Background()
	baseRepo := &redeemRejectRepo{code: RedeemCode{
		ID: 1, Code: "BALANCE-CODE", Type: RedeemTypeBalance, Value: 10, Status: StatusUnused,
	}}
	redeemRepo := &postCommitReadFailureRedeemRepo{redeemRejectRepo: baseRepo}
	userRepo := &mockUserRepo{getByIDUser: &User{ID: 10}}
	userRepo.updateBalanceFn = func(context.Context, int64, float64) error { return nil }
	svc := NewRedeemService(redeemRepo, userRepo, nil, nil, nil, newPaymentOrderLifecycleTestClient(t), nil, nil)

	result, err := svc.Redeem(ctx, 10, baseRepo.code.Code)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, StatusUsed, result.Status)
	require.NotNil(t, result.UsedBy)
	require.Equal(t, int64(10), *result.UsedBy)
}

func TestAssignOrExtendSubscription_OuterTransactionInvalidatesAfterCommit(t *testing.T) {
	client := newPaymentOrderLifecycleTestClient(t)
	tx, err := client.Tx(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	txCtx := dbent.NewTxContext(context.Background(), tx)
	repo := &redeemSubscriptionRepoStub{subscriptionUserSubRepoStub: newSubscriptionUserSubRepoStub()}
	cache := &redeemSubscriptionCacheStub{}
	svc := NewSubscriptionService(
		&subscriptionGroupRepoStub{group: &Group{ID: 20, SubscriptionType: SubscriptionTypeSubscription}},
		repo,
		&BillingCacheService{cache: cache},
		nil,
		nil,
	)

	_, _, err = svc.AssignOrExtendSubscription(txCtx, &AssignSubscriptionInput{UserID: 10, GroupID: 20, ValidityDays: 3})
	require.NoError(t, err)
	require.Zero(t, cache.publications.Load(), "外层事务提交前不能失效缓存")

	require.NoError(t, tx.Commit())
	require.Equal(t, int32(1), cache.publications.Load(), "外层事务提交后必须失效缓存")
}

func TestPaymentSubscriptionAssignment_DoesNotFailAfterCommitOnCacheError(t *testing.T) {
	client := newPaymentOrderLifecycleTestClient(t)
	repo := &redeemSubscriptionRepoStub{subscriptionUserSubRepoStub: newSubscriptionUserSubRepoStub()}
	cache := &redeemSubscriptionCacheStub{invalidateErr: errors.New("redis delete failed")}
	subscriptionSvc := NewSubscriptionService(
		&subscriptionGroupRepoStub{group: &Group{ID: 20, SubscriptionType: SubscriptionTypeSubscription}},
		repo,
		&BillingCacheService{cache: cache},
		nil,
		nil,
	)
	paymentSvc := &PaymentService{entClient: client, subscriptionSvc: subscriptionSvc}

	err := paymentSvc.ensurePaymentSubscriptionAssigned(context.Background(), &dbent.PaymentOrder{ID: 987654, UserID: 10}, 20, 3)

	require.NoError(t, err)
	require.Equal(t, int32(1), cache.publications.Load())
}

func TestRedeemSubscription_InvalidatesAfterCommit(t *testing.T) {
	for _, validityDays := range []int{3, -1} {
		t.Run(map[bool]string{true: "extend", false: "reduce"}[validityDays > 0], func(t *testing.T) {
			ctx := context.Background()
			client := newPaymentOrderLifecycleTestClient(t)
			groupID := int64(20)
			userID := int64(10)
			redeemRepo := &redeemRejectRepo{code: RedeemCode{
				ID:           1,
				Code:         "SUB-CODE",
				Type:         RedeemTypeSubscription,
				Status:       StatusUnused,
				GroupID:      &groupID,
				ValidityDays: validityDays,
			}}
			userRepo := &mockUserRepo{getByIDUser: &User{ID: userID}}
			subRepo := &redeemSubscriptionRepoStub{subscriptionUserSubRepoStub: newSubscriptionUserSubRepoStub()}
			now := time.Now()
			subRepo.seed(&UserSubscription{
				ID:        30,
				UserID:    userID,
				GroupID:   groupID,
				StartsAt:  now.Add(-time.Hour),
				ExpiresAt: now.Add(10 * 24 * time.Hour),
				Status:    SubscriptionStatusActive,
			})
			cache := &redeemSubscriptionCacheStub{}
			billingCache := &BillingCacheService{cache: cache}
			subscriptionSvc := NewSubscriptionService(
				&subscriptionGroupRepoStub{group: &Group{ID: groupID, SubscriptionType: SubscriptionTypeSubscription}},
				subRepo,
				billingCache,
				nil,
				nil,
			)
			redeemSvc := NewRedeemService(redeemRepo, userRepo, subscriptionSvc, nil, billingCache, client, nil, nil)

			_, err := redeemSvc.Redeem(ctx, userID, redeemRepo.code.Code)

			require.NoError(t, err)
			require.Eventually(t, func() bool { return cache.invalidations.Load() > 0 }, time.Second, 10*time.Millisecond)
			require.Equal(t, int32(1), cache.publications.Load(), "订阅兑换提交后应发布一次跨实例 L1 失效")
		})
	}
}

func TestSubscriptionSemanticMutations_PublishCrossInstanceInvalidation(t *testing.T) {
	tests := map[string]func(*SubscriptionService, *UserSubscription) error{
		"assign or extend": func(svc *SubscriptionService, sub *UserSubscription) error {
			_, _, err := svc.AssignOrExtendSubscription(context.Background(), &AssignSubscriptionInput{
				UserID: sub.UserID, GroupID: sub.GroupID, ValidityDays: 1,
			})
			return err
		},
		"extend": func(svc *SubscriptionService, sub *UserSubscription) error {
			_, err := svc.ExtendSubscription(context.Background(), sub.ID, 1)
			return err
		},
		"activate windows": func(svc *SubscriptionService, sub *UserSubscription) error {
			sub.DailyWindowStart = nil
			sub.WeeklyWindowStart = nil
			sub.MonthlyWindowStart = nil
			return svc.CheckAndActivateWindow(context.Background(), sub)
		},
		"admin reset": func(svc *SubscriptionService, sub *UserSubscription) error {
			_, err := svc.AdminResetQuota(context.Background(), sub.ID, true, false, false)
			return err
		},
		"automatic reset": func(svc *SubscriptionService, sub *UserSubscription) error {
			stale := time.Now().Add(-25 * time.Hour)
			sub.DailyWindowStart = &stale
			sub.StartsAt = time.Now().Add(-48 * time.Hour)
			sub.ExpiresAt = time.Now().Add(48 * time.Hour)
			return svc.CheckAndResetWindows(context.Background(), sub)
		},
		"expire status": func(svc *SubscriptionService, sub *UserSubscription) error {
			sub.ExpiresAt = time.Now().Add(-time.Minute)
			return svc.ValidateSubscription(context.Background(), sub)
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			now := time.Now()
			sub := &UserSubscription{
				ID:        30,
				UserID:    10,
				GroupID:   20,
				StartsAt:  now.Add(-time.Hour),
				ExpiresAt: now.Add(10 * 24 * time.Hour),
				Status:    SubscriptionStatusActive,
			}
			repo := &redeemSubscriptionRepoStub{subscriptionUserSubRepoStub: newSubscriptionUserSubRepoStub()}
			repo.seed(sub)
			cache := &redeemSubscriptionCacheStub{}
			svc := NewSubscriptionService(
				&subscriptionGroupRepoStub{group: &Group{ID: sub.GroupID, SubscriptionType: SubscriptionTypeSubscription}},
				repo,
				&BillingCacheService{cache: cache},
				nil,
				nil,
			)

			err := mutate(svc, sub)

			if name == "expire status" {
				require.ErrorIs(t, err, ErrSubscriptionExpired)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, int32(1), cache.publications.Load(), "%s 应发布跨实例 L1 失效", name)
		})
	}
}

func TestRecordUsage_DoesNotPublishCrossInstanceL1Invalidation(t *testing.T) {
	repo := &redeemSubscriptionRepoStub{subscriptionUserSubRepoStub: newSubscriptionUserSubRepoStub()}
	repo.seed(&UserSubscription{ID: 30, UserID: 10, GroupID: 20})
	cache := &redeemSubscriptionCacheStub{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, &BillingCacheService{cache: cache}, nil, nil)

	err := svc.RecordUsage(context.Background(), 30, 1)

	require.NoError(t, err)
	require.Zero(t, cache.publications.Load(), "高频 usage 由 billing Redis 同步，不应广播 L1 失效")
}
