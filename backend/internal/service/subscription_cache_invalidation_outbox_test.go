package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

type subscriptionInvalidationRepoStub struct {
	mu        sync.Mutex
	events    []SubscriptionCacheInvalidationEvent
	scheduled []int64
	deleted   []int64
	retried   []int64
}

func (r *subscriptionInvalidationRepoStub) Claim(_ context.Context, _ string, _ int, _ time.Duration) ([]SubscriptionCacheInvalidationEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]SubscriptionCacheInvalidationEvent(nil), r.events...), nil
}

func (r *subscriptionInvalidationRepoStub) ScheduleSecondPass(_ context.Context, id int64, _ string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scheduled = append(r.scheduled, id)
	return nil
}

func (r *subscriptionInvalidationRepoStub) DeleteClaimed(_ context.Context, id int64, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deleted = append(r.deleted, id)
	return nil
}

func (r *subscriptionInvalidationRepoStub) RetryClaimed(_ context.Context, id int64, _ string, _ time.Time, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retried = append(r.retried, id)
	return nil
}

type subscriptionInvalidationCacheStub struct {
	tombstoneErr         error
	publishErr           error
	waitTombstoneTimeout bool
	publishContextActive atomic.Bool
	tombstones           atomic.Int32
	publications         atomic.Int32
}

func (c *subscriptionInvalidationCacheStub) InvalidateSubscriptionVersioned(ctx context.Context, _, _, _ int64) error {
	c.tombstones.Add(1)
	if c.waitTombstoneTimeout {
		<-ctx.Done()
		return ctx.Err()
	}
	return c.tombstoneErr
}

func (c *subscriptionInvalidationCacheStub) PublishSubscriptionCacheInvalidation(ctx context.Context, _ string) error {
	c.publications.Add(1)
	c.publishContextActive.Store(ctx.Err() == nil)
	return c.publishErr
}

type subscriptionCacheL1Tracker struct {
	dels  atomic.Int32
	waits atomic.Int32
}

func (c *subscriptionCacheL1Tracker) Del(any)                                      { c.dels.Add(1) }
func (*subscriptionCacheL1Tracker) Get(any) (any, bool)                            { return nil, false }
func (*subscriptionCacheL1Tracker) SetWithTTL(any, any, int64, time.Duration) bool { return true }
func (c *subscriptionCacheL1Tracker) Wait()                                        { c.waits.Add(1) }

func TestSubscriptionCacheInvalidationWorker_RequiresTombstoneAndPublishBeforeAck(t *testing.T) {
	repo := &subscriptionInvalidationRepoStub{}
	cache := &subscriptionInvalidationCacheStub{
		tombstoneErr: errors.New("redis unavailable"),
		publishErr:   errors.New("publish unavailable"),
	}
	worker := NewSubscriptionCacheInvalidationWorker(repo, cache)
	event := SubscriptionCacheInvalidationEvent{ID: 7, UserID: 10, GroupID: 20, Version: 100, Stage: 1}

	worker.processEvent(context.Background(), event)

	require.Equal(t, []int64{7}, repo.retried)
	require.Empty(t, repo.deleted, "an event must stay durable until both delivery operations succeed")
	require.Equal(t, int32(1), cache.tombstones.Load())
	require.Equal(t, int32(1), cache.publications.Load(), "publish must still be attempted when tombstoning fails")

	cache.tombstoneErr = nil
	cache.publishErr = nil
	worker.processEvent(context.Background(), event)

	require.Equal(t, []int64{7}, repo.deleted)
}

func TestSubscriptionCacheInvalidationWorker_UsesSafetySecondPass(t *testing.T) {
	repo := &subscriptionInvalidationRepoStub{}
	cache := &subscriptionInvalidationCacheStub{}
	worker := NewSubscriptionCacheInvalidationWorker(repo, cache)

	worker.processEvent(context.Background(), SubscriptionCacheInvalidationEvent{ID: 8, UserID: 10, GroupID: 20, Version: 101})

	require.Equal(t, []int64{8}, repo.scheduled)
	require.Empty(t, repo.deleted)
	require.Equal(t, int32(1), cache.tombstones.Load())
	require.Equal(t, int32(1), cache.publications.Load())
}

func TestSubscriptionCacheInvalidationWorker_PublishGetsIndependentTimeout(t *testing.T) {
	repo := &subscriptionInvalidationRepoStub{}
	cache := &subscriptionInvalidationCacheStub{waitTombstoneTimeout: true}
	worker := NewSubscriptionCacheInvalidationWorker(repo, cache)

	worker.processEvent(context.Background(), SubscriptionCacheInvalidationEvent{ID: 9, UserID: 10, GroupID: 20, Version: 102, Stage: 1})

	require.True(t, cache.publishContextActive.Load(), "publish must not reuse the tombstone timeout")
	require.Equal(t, []int64{9}, repo.retried)
}

func TestSubscriptionCacheInvalidationFastPath_WaitsForOuterCommit(t *testing.T) {
	client := newPaymentOrderLifecycleTestClient(t)
	cache := &redeemSubscriptionCacheStub{}
	l1 := &subscriptionCacheL1Tracker{}
	svc := &SubscriptionService{
		billingCacheService: &BillingCacheService{cache: cache},
		subCacheL1:          l1,
	}
	sub := &UserSubscription{UserID: 10, GroupID: 20, UpdatedAt: time.Unix(0, 100)}

	tx, err := client.Tx(context.Background())
	require.NoError(t, err)
	svc.deferSubscriptionCacheInvalidation(dbent.NewTxContext(context.Background(), tx), sub)
	require.Zero(t, l1.dels.Load(), "an outer transaction must not mutate L1 before commit")
	require.Zero(t, cache.versionedInvalidations.Load())
	require.NoError(t, tx.Rollback())
	require.Zero(t, l1.dels.Load(), "a rolled back transaction must have no cache side effects")
	require.Zero(t, cache.publications.Load())

	tx, err = client.Tx(context.Background())
	require.NoError(t, err)
	svc.deferSubscriptionCacheInvalidation(dbent.NewTxContext(context.Background(), tx), sub)
	require.NoError(t, tx.Commit())
	require.Equal(t, int32(1), l1.dels.Load())
	require.Equal(t, int32(1), l1.waits.Load())
	require.Equal(t, int32(1), cache.versionedInvalidations.Load())
	require.Equal(t, int32(1), cache.publications.Load())
}

func TestSubscriptionCacheInvalidationFastPath_NilCacheStillClearsLocalL1(t *testing.T) {
	l1 := &subscriptionCacheL1Tracker{}
	svc := &SubscriptionService{subCacheL1: l1}

	svc.deferSubscriptionCacheInvalidation(context.Background(), &UserSubscription{UserID: 10, GroupID: 20, UpdatedAt: time.Unix(0, 100)})

	require.Equal(t, int32(1), l1.dels.Load())
	require.Equal(t, int32(1), l1.waits.Load())
}

func TestSubscriptionCacheInvalidationFastPath_UnknownVersionOnlyClearsLocalL1(t *testing.T) {
	cache := &redeemSubscriptionCacheStub{}
	l1 := &subscriptionCacheL1Tracker{}
	svc := &SubscriptionService{
		billingCacheService: &BillingCacheService{cache: cache},
		subCacheL1:          l1,
	}

	svc.deferSubscriptionCacheInvalidation(context.Background(), &UserSubscription{UserID: 10, GroupID: 20})

	require.Equal(t, int32(1), l1.dels.Load())
	require.Equal(t, int32(1), l1.waits.Load())
	require.Zero(t, cache.versionedInvalidations.Load(), "a non-authoritative version must not tombstone Redis")
	require.Zero(t, cache.publications.Load(), "the durable outbox must publish after an unknown-version mutation")
}

func TestAdvanceQuotaCycle_UsesVersionedPostCommitInvalidation(t *testing.T) {
	client := newPaymentOrderLifecycleTestClient(t)
	repo := newTermLockingUserSubRepo()
	repo.updateVersion = 1
	sub := exhaustedQuotaSubscription(time.Now())
	sub.WeeklyUsageUSD = 0
	sub.MonthlyUsageUSD = 0
	repo.seed(sub)
	cache := &redeemSubscriptionCacheStub{}
	svc := &SubscriptionService{
		userSubRepo:         repo,
		billingCacheService: &BillingCacheService{cache: cache},
		entClient:           client,
	}

	_, err := svc.AdvanceQuotaCycle(context.Background(), sub.UserID, sub.ID, QuotaWindowSelection{Daily: true})

	require.NoError(t, err)
	require.Equal(t, int32(1), cache.versionedInvalidations.Load())
	require.Equal(t, int32(1), cache.publications.Load())
}
