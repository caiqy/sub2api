//go:build unit

package repository

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSetSubscriptionCacheRejectsStaleSnapshot(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()
	fresh := &service.SubscriptionCacheData{
		Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour),
		DailyUsage: 0, Version: 20,
	}
	stale := &service.SubscriptionCacheData{
		Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(48 * time.Hour),
		DailyUsage: 10, Version: 10,
	}

	require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, fresh))
	require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, stale))
	stored, err := cache.GetSubscriptionCache(ctx, 1, 2)

	require.NoError(t, err)
	require.Equal(t, int64(20), stored.Version)
	require.Zero(t, stored.DailyUsage)
}

func TestSubscriptionCachePreservesPostResetUsageAcrossWriteOrder(t *testing.T) {
	for _, usageFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("usage_first_%t", usageFirst), func(t *testing.T) {
			cache, _ := newMiniRedisCache(t)
			ctx := context.Background()
			initial := &service.SubscriptionCacheData{
				Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), DailyUsage: 10, Version: 10,
			}
			stale := &service.SubscriptionCacheData{
				Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), DailyUsage: 10, Version: 10,
			}
			resetSnapshot := &service.SubscriptionCacheData{
				Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(23 * time.Hour), Version: 20,
			}
			fresh := &service.SubscriptionCacheData{
				Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(23 * time.Hour), DailyUsage: 1.5, Version: 30,
			}
			require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, initial))

			if usageFirst {
				require.NoError(t, cache.UpdateSubscriptionUsage(ctx, 1, 2, 1.5, 10))
				require.NoError(t, cache.InvalidateSubscriptionCacheVersioned(ctx, 1, 2, 20))
			} else {
				require.NoError(t, cache.InvalidateSubscriptionCacheVersioned(ctx, 1, 2, 20))
				require.NoError(t, cache.UpdateSubscriptionUsage(ctx, 1, 2, 1.5, 10))
			}
			require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, stale))
			require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, resetSnapshot))
			_, err := cache.GetSubscriptionCache(ctx, 1, 2)
			require.Error(t, err)
			require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, fresh))

			stored, err := cache.GetSubscriptionCache(ctx, 1, 2)
			require.NoError(t, err)
			require.Equal(t, 1.5, stored.DailyUsage)
			require.Equal(t, int64(30), stored.Version)
		})
	}
}

func TestUpdateSubscriptionUsage_OldDeltaSkippedAfterResetSnapshot(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()

	require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, &service.SubscriptionCacheData{
		Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), Version: 10,
	}))
	// Quota cycle advance: tombstone raised to the committed reset version.
	require.NoError(t, cache.InvalidateSubscriptionCacheVersioned(ctx, 1, 2, 20))
	// New cycle snapshot refilled from DB at a strictly newer row version.
	require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, &service.SubscriptionCacheData{
		Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), DailyUsage: 5, Version: 21,
	}))

	// Pre-reset delta (version 15) committed before the advance but delivered
	// after the snapshot refill must be skipped: 15 <= tombstone(20).
	require.NoError(t, cache.UpdateSubscriptionUsage(ctx, 1, 2, 3.0, 15))
	stored, err := cache.GetSubscriptionCache(ctx, 1, 2)
	require.NoError(t, err)
	require.Equal(t, 5.0, stored.DailyUsage, "pre-reset delta must not pollute the new cycle snapshot")
	require.Zero(t, stored.WeeklyUsage)
	require.Zero(t, stored.MonthlyUsage)

	// Post-reset delta (version 22) above both tombstone and snapshot applies.
	require.NoError(t, cache.UpdateSubscriptionUsage(ctx, 1, 2, 1.0, 22))
	stored, err = cache.GetSubscriptionCache(ctx, 1, 2)
	require.NoError(t, err)
	require.Equal(t, 6.0, stored.DailyUsage)
}

func TestRenewalTombstoneRejectsOldSnapshotAndDelta(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()

	require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, &service.SubscriptionCacheData{
		Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), DailyUsage: 10, Version: 10,
	}))
	// Expired renewal clears all usage in the DB and commits this row version.
	require.NoError(t, cache.InvalidateSubscriptionCacheVersioned(ctx, 1, 2, 20))
	require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, &service.SubscriptionCacheData{
		Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), DailyUsage: 10, Version: 10,
	}))
	require.NoError(t, cache.UpdateSubscriptionUsage(ctx, 1, 2, 3, 15))
	_, err := cache.GetSubscriptionCache(ctx, 1, 2)
	require.ErrorIs(t, err, redis.Nil, "old renewal snapshot and delta must not revive the tombstoned cache entry")

	require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, &service.SubscriptionCacheData{
		Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), Version: 21,
	}))
	require.NoError(t, cache.UpdateSubscriptionUsage(ctx, 1, 2, 3, 15))
	stored, err := cache.GetSubscriptionCache(ctx, 1, 2)
	require.NoError(t, err)
	require.Zero(t, stored.DailyUsage, "old renewal delta must not pollute the post-renewal snapshot")
}

func TestUpdateSubscriptionUsage_SameVersionDeltaNotReapplied(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()

	require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, &service.SubscriptionCacheData{
		Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), DailyUsage: 5, Version: 20,
	}))

	// The snapshot already includes this delta (deltaVersion == snapshotVersion).
	require.NoError(t, cache.UpdateSubscriptionUsage(ctx, 1, 2, 2.0, 20))
	stored, err := cache.GetSubscriptionCache(ctx, 1, 2)
	require.NoError(t, err)
	require.Equal(t, 5.0, stored.DailyUsage, "same-version delta must not be double counted")

	require.NoError(t, cache.UpdateSubscriptionUsage(ctx, 1, 2, 2.0, 21))
	stored, err = cache.GetSubscriptionCache(ctx, 1, 2)
	require.NoError(t, err)
	require.Equal(t, 7.0, stored.DailyUsage)
}

func TestUpdateSubscriptionUsage_OutOfOrderNewDeltasBothApply(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()

	require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, &service.SubscriptionCacheData{
		Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), Version: 10,
	}))

	// Both deltas are newer than the snapshot; arrival order must not matter.
	require.NoError(t, cache.UpdateSubscriptionUsage(ctx, 1, 2, 1.0, 12))
	require.NoError(t, cache.UpdateSubscriptionUsage(ctx, 1, 2, 2.0, 11))

	stored, err := cache.GetSubscriptionCache(ctx, 1, 2)
	require.NoError(t, err)
	require.Equal(t, 3.0, stored.DailyUsage, "both deltas newer than the snapshot must accumulate regardless of order")
	require.Equal(t, 3.0, stored.WeeklyUsage)
	require.Equal(t, 3.0, stored.MonthlyUsage)
	require.Equal(t, int64(10), stored.Version, "delta application must not rewrite the snapshot version")
}

func TestUpdateSubscriptionUsage_MissingKeyAfterTombstoneNoOp(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()

	require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, &service.SubscriptionCacheData{
		Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), Version: 10,
	}))
	require.NoError(t, cache.InvalidateSubscriptionCacheVersioned(ctx, 1, 2, 30))

	// Key is gone (tombstoned); a late delta must stay a no-op and must not
	// resurrect the hash.
	require.NoError(t, cache.UpdateSubscriptionUsage(ctx, 1, 2, 2.0, 31))
	_, err := cache.GetSubscriptionCache(ctx, 1, 2)
	require.ErrorIs(t, err, redis.Nil, "delta must not recreate a tombstoned snapshot")
}

func TestSubscriptionVersionScriptsCompareMicrosecondVersionsPrecisely(t *testing.T) {
	// These two valid UnixNano values are one microsecond apart but round to the
	// same IEEE-754 float64. Redis Lua must compare the decimal strings exactly.
	const before int64 = 9223372036854772230
	const after = before + 1000
	ctx := context.Background()

	t.Run("delta", func(t *testing.T) {
		cache, _ := newMiniRedisCache(t)
		require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, &service.SubscriptionCacheData{
			Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), DailyUsage: 1, Version: before,
		}))
		require.NoError(t, cache.UpdateSubscriptionUsage(ctx, 1, 2, 2, after))

		stored, err := cache.GetSubscriptionCache(ctx, 1, 2)
		require.NoError(t, err)
		require.Equal(t, 3.0, stored.DailyUsage)
	})

	t.Run("snapshot", func(t *testing.T) {
		cache, _ := newMiniRedisCache(t)
		require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, &service.SubscriptionCacheData{
			Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), DailyUsage: 1, Version: before,
		}))
		require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, &service.SubscriptionCacheData{
			Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), DailyUsage: 3, Version: after,
		}))

		stored, err := cache.GetSubscriptionCache(ctx, 1, 2)
		require.NoError(t, err)
		require.Equal(t, after, stored.Version)
		require.Equal(t, 3.0, stored.DailyUsage)
	})

	t.Run("tombstone", func(t *testing.T) {
		cache, _ := newMiniRedisCache(t)
		require.NoError(t, cache.SetSubscriptionCache(ctx, 1, 2, &service.SubscriptionCacheData{
			Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour), Version: before,
		}))
		require.NoError(t, cache.InvalidateSubscriptionCacheVersioned(ctx, 1, 2, after))

		_, err := cache.GetSubscriptionCache(ctx, 1, 2)
		require.ErrorIs(t, err, redis.Nil)
	})
}

func TestBillingBalanceKey(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		expected string
	}{
		{
			name:     "normal_user_id",
			userID:   123,
			expected: "billing:balance:123",
		},
		{
			name:     "zero_user_id",
			userID:   0,
			expected: "billing:balance:0",
		},
		{
			name:     "negative_user_id",
			userID:   -1,
			expected: "billing:balance:-1",
		},
		{
			name:     "max_int64",
			userID:   math.MaxInt64,
			expected: "billing:balance:9223372036854775807",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := billingBalanceKey(tc.userID)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestBillingSubKey(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		groupID  int64
		expected string
	}{
		{
			name:     "normal_ids",
			userID:   123,
			groupID:  456,
			expected: "billing:sub:123:456",
		},
		{
			name:     "zero_ids",
			userID:   0,
			groupID:  0,
			expected: "billing:sub:0:0",
		},
		{
			name:     "negative_ids",
			userID:   -1,
			groupID:  -2,
			expected: "billing:sub:-1:-2",
		},
		{
			name:     "max_int64_ids",
			userID:   math.MaxInt64,
			groupID:  math.MaxInt64,
			expected: "billing:sub:9223372036854775807:9223372036854775807",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := billingSubKey(tc.userID, tc.groupID)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestJitteredTTL(t *testing.T) {
	const (
		minTTL = 4*time.Minute + 30*time.Second // 270s = 5min - 30s
		maxTTL = 5*time.Minute + 30*time.Second // 330s = 5min + 30s
	)

	for i := 0; i < 200; i++ {
		ttl := jitteredTTL()
		require.GreaterOrEqual(t, ttl, minTTL, "jitteredTTL() 返回值低于下限: %v", ttl)
		require.LessOrEqual(t, ttl, maxTTL, "jitteredTTL() 返回值超过上限: %v", ttl)
	}
}

func TestJitteredTTL_HasVariation(t *testing.T) {
	// 多次调用应该产生不同的值（验证抖动存在）
	seen := make(map[time.Duration]struct{}, 50)
	for i := 0; i < 50; i++ {
		seen[jitteredTTL()] = struct{}{}
	}
	// 50 次调用中应该至少有 2 个不同的值
	require.Greater(t, len(seen), 1, "jitteredTTL() 应产生不同的 TTL 值")
}
