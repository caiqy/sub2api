package repository

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newInflightTestEnv(t *testing.T, enabled bool, ttlSeconds int) (*miniredis.Miniredis, *billingCache, *service.BillingCacheService) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := &billingCache{rdb: rdb}
	cfg := &config.Config{}
	cfg.Billing.InflightReservation = config.InflightReservationConfig{
		Enabled:          enabled,
		TTLSeconds:       ttlSeconds,
		DefaultMaxTokens: 8192,
	}
	svc := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(svc.Stop)
	return mr, cache, svc
}

func inflightCount(t *testing.T, cache *billingCache, userID int64) int64 {
	t.Helper()
	zkey, _ := billingInflightKeys(userID)
	n, err := cache.rdb.ZCard(context.Background(), zkey).Result()
	require.NoError(t, err)
	return n
}

func TestInflightReservation_ConcurrentAdmitsOnlyWhatBalanceCovers(t *testing.T) {
	_, cache, svc := newInflightTestEnv(t, true, 60)
	ctx := context.Background()
	user := &service.User{ID: 42}
	require.NoError(t, cache.SetUserBalance(ctx, user.ID, 1.0))

	const n = 20
	var admitted, rejected atomic.Int32
	releases := make(chan func(), n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			release, err := svc.ReserveInflightBalance(ctx, user, nil, nil, 0.3)
			if err != nil {
				if !errors.Is(err, service.ErrInsufficientBalance) {
					t.Errorf("unexpected error: %v", err)
				}
				rejected.Add(1)
				return
			}
			admitted.Add(1)
			releases <- release
		}()
	}
	close(start)
	wg.Wait()
	close(releases)

	// balance 1.0, estimate 0.3: first always admitted, then while 1.0 - inflight >= 0.3 → 3 total.
	require.Equal(t, int32(3), admitted.Load())
	require.Equal(t, int32(n-3), rejected.Load())
	require.Equal(t, int64(3), inflightCount(t, cache, user.ID))

	for release := range releases {
		release()
		release() // idempotent
	}
	require.Equal(t, int64(0), inflightCount(t, cache, user.ID))

	// After release, capacity is available again.
	release, err := svc.ReserveInflightBalance(ctx, user, nil, nil, 0.3)
	require.NoError(t, err)
	release()
}

func TestInflightReservation_FirstRequestAlwaysAdmitted(t *testing.T) {
	_, cache, svc := newInflightTestEnv(t, true, 60)
	ctx := context.Background()
	user := &service.User{ID: 7}
	require.NoError(t, cache.SetUserBalance(ctx, user.ID, 0.01))

	release, err := svc.ReserveInflightBalance(ctx, user, nil, nil, 5)
	require.NoError(t, err, "a lone request keeps legacy behavior even if estimate exceeds balance")
	_, err = svc.ReserveInflightBalance(ctx, user, nil, nil, 0.001)
	require.ErrorIs(t, err, service.ErrInsufficientBalance)
	release()
	release2, err := svc.ReserveInflightBalance(ctx, user, nil, nil, 0.001)
	require.NoError(t, err)
	release2()
}

func TestInflightReservation_TTLExpiryFreesLeakedReservation(t *testing.T) {
	_, cache, svc := newInflightTestEnv(t, true, 1)
	ctx := context.Background()
	user := &service.User{ID: 9}
	require.NoError(t, cache.SetUserBalance(ctx, user.ID, 1.0))

	_, err := svc.ReserveInflightBalance(ctx, user, nil, nil, 0.8) // leaked: never released
	require.NoError(t, err)
	_, err = svc.ReserveInflightBalance(ctx, user, nil, nil, 0.8)
	require.ErrorIs(t, err, service.ErrInsufficientBalance)

	time.Sleep(1100 * time.Millisecond)
	release, err := svc.ReserveInflightBalance(ctx, user, nil, nil, 0.8)
	require.NoError(t, err)
	require.Equal(t, int64(1), inflightCount(t, cache, user.ID))
	release()
}

func TestInflightReservation_DisabledKeepsLegacyBehavior(t *testing.T) {
	_, cache, svc := newInflightTestEnv(t, false, 60)
	ctx := context.Background()
	user := &service.User{ID: 11}
	require.NoError(t, cache.SetUserBalance(ctx, user.ID, 0.01))
	require.False(t, svc.InflightReservationEnabled())
	for i := 0; i < 5; i++ {
		_, err := svc.ReserveInflightBalance(ctx, user, nil, nil, 10)
		require.NoError(t, err)
	}
	require.Equal(t, int64(0), inflightCount(t, cache, user.ID))
}

func TestInflightReservation_SubscriptionUnaffected(t *testing.T) {
	_, cache, svc := newInflightTestEnv(t, true, 60)
	ctx := context.Background()
	user := &service.User{ID: 12}
	require.NoError(t, cache.SetUserBalance(ctx, user.ID, 0))
	group := &service.Group{ID: 1, SubscriptionType: service.SubscriptionTypeSubscription}
	sub := &service.UserSubscription{ID: 1}
	for i := 0; i < 5; i++ {
		_, err := svc.ReserveInflightBalance(ctx, user, group, sub, 10)
		require.NoError(t, err)
	}
	require.Equal(t, int64(0), inflightCount(t, cache, user.ID))
}

func TestInflightReservation_ZeroEstimateSkips(t *testing.T) {
	_, cache, svc := newInflightTestEnv(t, true, 60)
	ctx := context.Background()
	user := &service.User{ID: 13}
	require.NoError(t, cache.SetUserBalance(ctx, user.ID, 1))
	_, err := svc.ReserveInflightBalance(ctx, user, nil, nil, 0)
	require.NoError(t, err)
	require.Equal(t, int64(0), inflightCount(t, cache, user.ID))
}

// downReserveCache 余额读取正常，但预留走一个已关闭的 Redis。
type downReserveCache struct {
	service.BillingCache
	down *billingCache
}

func (d *downReserveCache) ReserveInflightBalance(ctx context.Context, userID int64, requestID string, amount, balance float64, ttl time.Duration) (bool, float64, error) {
	return d.down.ReserveInflightBalance(ctx, userID, requestID, amount, balance, ttl)
}

func (d *downReserveCache) ReleaseInflightBalance(ctx context.Context, userID int64, requestID string) error {
	return d.down.ReleaseInflightBalance(ctx, userID, requestID)
}

func TestInflightReservation_RedisDownFailsOpen(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	up := &billingCache{rdb: rdb}
	require.NoError(t, up.SetUserBalance(context.Background(), 5, 0.01))

	downMR := miniredis.RunT(t)
	downRDB := redis.NewClient(&redis.Options{Addr: downMR.Addr(), MaxRetries: -1, DialTimeout: 100 * time.Millisecond})
	t.Cleanup(func() { _ = downRDB.Close() })
	downMR.Close()

	cfg := &config.Config{}
	cfg.Billing.InflightReservation = config.InflightReservationConfig{Enabled: true, TTLSeconds: 60}
	svc := service.NewBillingCacheService(&downReserveCache{BillingCache: up, down: &billingCache{rdb: downRDB}}, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(svc.Stop)

	user := &service.User{ID: 5}
	for i := 0; i < 3; i++ {
		release, err := svc.ReserveInflightBalance(context.Background(), user, nil, nil, 100)
		require.NoError(t, err)
		release()
	}
}

func TestInflightReservation_ScriptErrorSurfaces(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	cache := &billingCache{rdb: rdb}
	mr.Close()
	_, _, err := cache.ReserveInflightBalance(context.Background(), 1, "r", 1, 1, time.Second)
	require.Error(t, err)
	require.False(t, errors.Is(err, context.Canceled))
	_ = rdb.Close()
}

// 回归：原实现在 handler 返回时即释放预留，而计费（RecordUsage → 余额缓存扣减）是异步的，
// 窗口内的顺序请求看到「在途=0 且余额未扣」全部放行（余额 $1、5×$0.9 → -4.40）。
// 现在预留由计费任务在余额缓存扣减之后才释放。
func TestInflightReservation_SequentialInBillingWindowAdmitsOnlyWhatBalanceCovers(t *testing.T) {
	_, cache, svc := newInflightTestEnv(t, true, 60)
	ctx := context.Background()
	user := &service.User{ID: 77}
	require.NoError(t, cache.SetUserBalance(ctx, user.ID, 1.0))

	const cost = 0.9
	var pendingBilling []func()
	admitted := 0
	for i := 0; i < 5; i++ {
		res, err := svc.ReserveInflight(ctx, user, nil, nil, cost)
		if err != nil {
			require.ErrorIs(t, err, service.ErrInsufficientBalance)
			continue
		}
		admitted++
		taskDone := res.Acquire() // handler submits the async billing task
		res.HandlerDone()         // handler returns before billing lands
		pendingBilling = append(pendingBilling, func() {
			require.NoError(t, cache.DeductUserBalance(ctx, user.ID, cost))
			taskDone()
		})
	}
	require.Equal(t, 1, admitted, "requests arriving before billing lands must not be admitted")

	for _, bill := range pendingBilling {
		bill()
	}
	bal, err := cache.GetUserBalance(ctx, user.ID)
	require.NoError(t, err)
	require.InDelta(t, 0.1, bal, 1e-9)
	require.Equal(t, int64(0), inflightCount(t, cache, user.ID))
}

func TestInflightReservation_RenewalKeepsStreamingReservationAlive(t *testing.T) {
	_, cache, svc := newInflightTestEnv(t, true, 1)
	ctx := context.Background()
	user := &service.User{ID: 78}
	require.NoError(t, cache.SetUserBalance(ctx, user.ID, 1.0))

	res, err := svc.ReserveInflight(ctx, user, nil, nil, 0.8)
	require.NoError(t, err)
	time.Sleep(2500 * time.Millisecond) // streaming well past the 1s TTL
	_, err = svc.ReserveInflight(ctx, user, nil, nil, 0.8)
	require.ErrorIs(t, err, service.ErrInsufficientBalance, "renewed reservation must still count")
	require.Equal(t, int64(1), inflightCount(t, cache, user.ID))

	res.HandlerDone()
	require.Equal(t, int64(0), inflightCount(t, cache, user.ID))

	ok, err := cache.RenewInflightBalance(ctx, user.ID, "missing", time.Second)
	require.NoError(t, err)
	require.False(t, ok, "renew must not resurrect a released reservation")
}

func TestInflightReservation_RenewDoesNotResurrectExpiredMember(t *testing.T) {
	_, cache, _ := newInflightTestEnv(t, true, 60)
	ctx := context.Background()
	userID := int64(79)
	zkey, hkey := billingInflightKeys(userID)
	// 已过期但尚未被惰性清理的成员（score 在过去）。
	require.NoError(t, cache.rdb.ZAdd(ctx, zkey, redis.Z{Score: float64(time.Now().UnixMilli() - 1000), Member: "stale"}).Err())
	require.NoError(t, cache.rdb.HSet(ctx, hkey, "stale", "0.5").Err())

	ok, err := cache.RenewInflightBalance(ctx, userID, "stale", time.Minute)
	require.NoError(t, err)
	require.False(t, ok, "renew must not resurrect an expired member")
	require.Equal(t, int64(0), inflightCount(t, cache, userID))
	exists, err := cache.rdb.HExists(ctx, hkey, "stale").Result()
	require.NoError(t, err)
	require.False(t, exists)
}

func TestInflightReservation_SessionGrowthUsesMaximumWithoutAccumulating(t *testing.T) {
	_, cache, svc := newInflightTestEnv(t, true, 60)
	ctx := context.Background()
	user := &service.User{ID: 80}
	require.NoError(t, cache.SetUserBalance(ctx, user.ID, 100))
	reservation, err := svc.ReserveInflight(ctx, user, nil, nil, 2)
	require.NoError(t, err)
	require.NotNil(t, reservation)
	t.Cleanup(reservation.HandlerDone)
	ctx = service.WithInflightReservation(ctx, reservation)
	for _, estimate := range []float64{6, 3, 6} {
		next, err := svc.ReserveInflight(ctx, user, nil, nil, estimate)
		require.NoError(t, err)
		require.NotNil(t, next)
		t.Cleanup(next.HandlerDone)
		require.Equal(t, int64(1), inflightCount(t, cache, user.ID), "accepted turns share one live reservation")
		_, hkey := billingInflightKeys(user.ID)
		amounts, err := cache.rdb.HVals(ctx, hkey).Result()
		require.NoError(t, err)
		require.Equal(t, []string{"6"}, amounts, "growth, cheaper turns and retries reserve the maximum, not their sum")
		require.Same(t, reservation, next)
		require.Equal(t, 6.0, next.Amount())
	}
}

func TestInflightReservation_ConcurrentSessionGrowthAndTaskCompletions(t *testing.T) {
	_, cache, svc := newInflightTestEnv(t, true, 60)
	ctx := context.Background()
	user := &service.User{ID: 82}
	require.NoError(t, cache.SetUserBalance(ctx, user.ID, 1000))
	reservation, err := svc.ReserveInflight(ctx, user, nil, nil, 1)
	require.NoError(t, err)
	t.Cleanup(reservation.HandlerDone)
	ctx = service.WithInflightReservation(ctx, reservation)
	var workers sync.WaitGroup
	errors := make(chan error, 32)
	done := make([]func(), 32)
	for i := range done {
		done[i] = reservation.Acquire()
		workers.Add(1)
		go func(estimate float64) {
			defer workers.Done()
			_, err := svc.ReserveInflight(ctx, user, nil, nil, estimate)
			errors <- err
		}(float64(i + 1))
	}
	workers.Wait()
	for range done {
		require.NoError(t, <-errors)
	}
	require.Equal(t, 32.0, reservation.Amount())
	require.Equal(t, int64(1), inflightCount(t, cache, user.ID))
	reservation.HandlerDone()
	for _, complete := range done {
		workers.Add(1)
		go func() { defer workers.Done(); complete(); complete() }()
	}
	workers.Wait()
	require.Zero(t, inflightCount(t, cache, user.ID))
}

func TestInflightReservation_ExpiredGenerationKeepsOldBillingTaskOffNewHandle(t *testing.T) {
	_, cache, svc := newInflightTestEnv(t, true, 60)
	ctx := context.Background()
	user := &service.User{ID: 81}
	require.NoError(t, cache.SetUserBalance(ctx, user.ID, 100))
	previous, err := svc.ReserveInflight(ctx, user, nil, nil, 2)
	require.NoError(t, err)
	t.Cleanup(previous.HandlerDone)
	oldTaskDone := previous.Acquire()
	t.Cleanup(oldTaskDone)
	zkey, _ := billingInflightKeys(user.ID)
	members, err := cache.rdb.ZRange(ctx, zkey, 0, -1).Result()
	require.NoError(t, err)
	require.Len(t, members, 1)
	require.NoError(t, cache.rdb.ZAdd(ctx, zkey, redis.Z{Member: members[0], Score: float64(time.Now().UnixMilli() - 1)}).Err())
	next, err := svc.ReserveInflight(service.WithInflightReservation(ctx, previous), user, nil, nil, 6)
	require.NoError(t, err)
	require.NotNil(t, next)
	t.Cleanup(next.HandlerDone)
	require.NotSame(t, previous, next, "expiry creates a new generation instead of reviving the old handle")
	newTaskDone := next.Acquire()
	t.Cleanup(newTaskDone)
	oldTaskDone()
	oldTaskDone()
	require.Equal(t, int64(1), inflightCount(t, cache, user.ID), "late old task release must not touch the new UUID")
	require.Equal(t, 6.0, next.Amount())
	next.HandlerDone()
	require.Equal(t, int64(1), inflightCount(t, cache, user.ID), "the new billing task still owns its hold")
	newTaskDone()
	newTaskDone()
	require.Zero(t, inflightCount(t, cache, user.ID))
}
