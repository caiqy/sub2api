package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type billingCacheWorkerStub struct {
	balanceUpdates      int64
	subscriptionUpdates int64
}

func (b *billingCacheWorkerStub) GetUserBalance(ctx context.Context, userID int64) (float64, error) {
	return 0, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetUserBalance(ctx context.Context, userID int64, balance float64) error {
	atomic.AddInt64(&b.balanceUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) DeductUserBalance(ctx context.Context, userID int64, amount float64) error {
	atomic.AddInt64(&b.balanceUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) InvalidateUserBalance(ctx context.Context, userID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetSubscriptionCache(ctx context.Context, userID, groupID int64) (*SubscriptionCacheData, error) {
	return nil, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetSubscriptionCache(ctx context.Context, userID, groupID int64, data *SubscriptionCacheData) error {
	atomic.AddInt64(&b.subscriptionUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) UpdateSubscriptionUsage(ctx context.Context, userID, groupID int64, cost float64, version int64) error {
	atomic.AddInt64(&b.subscriptionUpdates, 1)
	return nil
}

// versionRecordingBillingCache records the usage-delta version the worker
// passes to the Redis repository.
type versionRecordingBillingCache struct {
	billingCacheWorkerStub
	usageVersion atomic.Int64
}

type deleteRecordingBillingCache struct {
	billingCacheWorkerStub
	deleteCalls int
}

func (c *deleteRecordingBillingCache) InvalidateSubscriptionCache(context.Context, int64, int64) error {
	c.deleteCalls++
	return nil
}

func (v *versionRecordingBillingCache) UpdateSubscriptionUsage(ctx context.Context, userID, groupID int64, cost float64, version int64) error {
	v.usageVersion.Store(version)
	return nil
}

func (b *billingCacheWorkerStub) InvalidateSubscriptionCache(ctx context.Context, userID, groupID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetAPIKeyRateLimit(ctx context.Context, keyID int64) (*APIKeyRateLimitCacheData, error) {
	return nil, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetAPIKeyRateLimit(ctx context.Context, keyID int64, data *APIKeyRateLimitCacheData) error {
	return nil
}

func (b *billingCacheWorkerStub) UpdateAPIKeyRateLimitUsage(ctx context.Context, keyID int64, cost float64) error {
	return nil
}

func (b *billingCacheWorkerStub) InvalidateAPIKeyRateLimit(ctx context.Context, keyID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetUserPlatformQuotaCache(ctx context.Context, userID int64, platform string) (*UserPlatformQuotaCacheEntry, bool, error) {
	return nil, false, nil
}

func (b *billingCacheWorkerStub) SetUserPlatformQuotaCache(ctx context.Context, userID int64, platform string, entry *UserPlatformQuotaCacheEntry, ttl time.Duration) error {
	return nil
}

func (b *billingCacheWorkerStub) DeleteUserPlatformQuotaCache(ctx context.Context, userID int64, platform string) error {
	return nil
}

func (b *billingCacheWorkerStub) IncrUserPlatformQuotaUsageCache(ctx context.Context, userID int64, platform string, cost float64, ttl time.Duration, markDirty bool) error {
	return nil
}

func (b *billingCacheWorkerStub) PopDirtyUserPlatformQuotaKeys(ctx context.Context, n int) ([]UserPlatformQuotaKey, error) {
	return nil, nil
}

func (b *billingCacheWorkerStub) ReaddDirtyUserPlatformQuotaKeys(ctx context.Context, keys []UserPlatformQuotaKey) error {
	return nil
}

func (b *billingCacheWorkerStub) BatchGetUserPlatformQuotaCache(ctx context.Context, keys []UserPlatformQuotaKey) ([]*UserPlatformQuotaCacheEntry, error) {
	return nil, nil
}

func TestBillingCacheServiceQueueHighLoad(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(svc.Stop)

	start := time.Now()
	for i := 0; i < cacheWriteBufferSize*2; i++ {
		svc.QueueDeductBalance(1, 1)
	}
	require.Less(t, time.Since(start), 2*time.Second)

	svc.QueueUpdateSubscriptionUsage(1, 2, 1.5, 1)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.balanceUpdates) > 0
	}, 2*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.subscriptionUpdates) > 0
	}, 2*time.Second, 10*time.Millisecond)
}

func TestBillingCacheServiceEnqueueAfterStopReturnsFalse(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	svc.Stop()

	enqueued := svc.enqueueCacheWrite(cacheWriteTask{
		kind:   cacheWriteDeductBalance,
		userID: 1,
		amount: 1,
	})
	require.False(t, enqueued)
}

func TestBillingCacheService_QueueUpdateSubscriptionUsagePropagatesDBVersion(t *testing.T) {
	cache := &versionRecordingBillingCache{}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(svc.Stop)

	svc.QueueUpdateSubscriptionUsage(1, 2, 1.5, 1234567890123456789)

	require.Eventually(t, func() bool {
		return cache.usageVersion.Load() == 1234567890123456789
	}, 2*time.Second, 10*time.Millisecond, "DB result version must reach the Redis repository through the queue worker")
}

func TestFinalizePostUsageBilling_ForwardsSubscriptionUsageVersion(t *testing.T) {
	cache := &versionRecordingBillingCache{}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(svc.Stop)

	groupID := int64(20)
	finalizePostUsageBilling(context.Background(), &postUsageBillingParams{
		Cost:               &CostBreakdown{ActualCost: 2.5},
		User:               &User{ID: 10},
		APIKey:             &APIKey{GroupID: &groupID},
		Account:            &Account{},
		IsSubscriptionBill: true,
	}, &billingDeps{
		billingCacheService: svc,
	}, &UsageBillingApplyResult{SubscriptionUsageVersion: 424242})

	require.Eventually(t, func() bool {
		return cache.usageVersion.Load() == 424242
	}, 2*time.Second, 10*time.Millisecond, "finalize must pass the DB result version into the queue")
}

func TestInvalidateSubscriptionVersioned_RejectsUnversionedCache(t *testing.T) {
	cache := &deleteRecordingBillingCache{}
	svc := &BillingCacheService{cache: cache}

	err := svc.InvalidateSubscriptionVersioned(context.Background(), 1, 2, 3)

	require.Error(t, err)
	require.Zero(t, cache.deleteCalls, "versioned invalidation must not degrade to an unversioned delete")
}
