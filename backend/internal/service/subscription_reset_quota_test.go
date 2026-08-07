//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

// resetQuotaUserSubRepoStub 支持 GetByID、ResetUsageWindows，
// 其余方法继承 userSubRepoNoop（panic）。
type resetQuotaUserSubRepoStub struct {
	userSubRepoNoop

	sub *UserSubscription

	resetVersion int64

	resetDailyCalled   bool
	resetWeeklyCalled  bool
	resetMonthlyCalled bool
	resetDailyErr      error
	resetWeeklyErr     error
	resetMonthlyErr    error
	windowStart        time.Time
}

func (r *resetQuotaUserSubRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *resetQuotaUserSubRepoStub) ResetUsageWindows(_ context.Context, _ int64, resetDaily, resetWeekly, resetMonthly bool, windowStart time.Time) error {
	r.resetDailyCalled = resetDaily
	r.resetWeeklyCalled = resetWeekly
	r.resetMonthlyCalled = resetMonthly
	r.windowStart = windowStart
	if resetDaily && r.resetDailyErr != nil {
		return r.resetDailyErr
	}
	if resetWeekly && r.resetWeeklyErr != nil {
		return r.resetWeeklyErr
	}
	if resetMonthly && r.resetMonthlyErr != nil {
		return r.resetMonthlyErr
	}
	if r.sub == nil {
		return nil
	}
	if resetDaily {
		r.sub.DailyUsageUSD = 0
		r.sub.DailyWindowStart = &windowStart
	}
	if resetWeekly {
		r.sub.WeeklyUsageUSD = 0
		r.sub.WeeklyWindowStart = &windowStart
	}
	if resetMonthly {
		r.sub.MonthlyUsageUSD = 0
		r.sub.MonthlyWindowStart = &windowStart
	}
	return nil
}

func (r *resetQuotaUserSubRepoStub) ResetUsageWindowsWithVersion(ctx context.Context, id int64, resetDaily, resetWeekly, resetMonthly bool, windowStart time.Time) (int64, error) {
	err := r.ResetUsageWindows(ctx, id, resetDaily, resetWeekly, resetMonthly, windowStart)
	return r.resetVersion, err
}

func (r *resetQuotaUserSubRepoStub) ResetDailyUsage(_ context.Context, _ int64, _ *time.Time, windowStart time.Time) error {
	r.resetDailyCalled = true
	if r.resetDailyErr == nil && r.sub != nil {
		r.sub.DailyUsageUSD = 0
		r.sub.DailyWindowStart = &windowStart
	}
	return r.resetDailyErr
}

func (r *resetQuotaUserSubRepoStub) ResetDailyUsageWithVersion(ctx context.Context, id int64, expectedWindowStart *time.Time, windowStart time.Time) (int64, error) {
	err := r.ResetDailyUsage(ctx, id, expectedWindowStart, windowStart)
	return r.resetVersion, err
}

func (r *resetQuotaUserSubRepoStub) ResetWeeklyUsage(_ context.Context, _ int64, _ *time.Time, _ time.Time) error {
	r.resetWeeklyCalled = true
	return r.resetWeeklyErr
}

func (r *resetQuotaUserSubRepoStub) ResetWeeklyUsageWithVersion(ctx context.Context, id int64, expectedWindowStart *time.Time, windowStart time.Time) (int64, error) {
	err := r.ResetWeeklyUsage(ctx, id, expectedWindowStart, windowStart)
	return r.resetVersion, err
}

func (r *resetQuotaUserSubRepoStub) ResetMonthlyUsage(_ context.Context, _ int64, _ *time.Time, _ time.Time) error {
	r.resetMonthlyCalled = true
	return r.resetMonthlyErr
}

func (r *resetQuotaUserSubRepoStub) ResetMonthlyUsageWithVersion(ctx context.Context, id int64, expectedWindowStart *time.Time, windowStart time.Time) (int64, error) {
	err := r.ResetMonthlyUsage(ctx, id, expectedWindowStart, windowStart)
	return r.resetVersion, err
}

type resetQuotaVersionedCache struct {
	billingCacheWorkerStub
	version int64
}

func (c *resetQuotaVersionedCache) InvalidateSubscriptionCacheVersioned(_ context.Context, _ int64, _ int64, version int64) error {
	c.version = version
	return nil
}

type unversionedResetQuotaRepo struct {
	userSubRepoNoop
	sub    *UserSubscription
	writes int
}

type dailyOnlyVersionedResetQuotaRepo struct {
	unversionedResetQuotaRepo
}

func (r *dailyOnlyVersionedResetQuotaRepo) ResetDailyUsageWithVersion(ctx context.Context, id int64, expectedWindowStart *time.Time, windowStart time.Time) (int64, error) {
	if err := r.ResetDailyUsage(ctx, id, expectedWindowStart, windowStart); err != nil {
		return 0, err
	}
	return 1, nil
}

func (r *unversionedResetQuotaRepo) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *unversionedResetQuotaRepo) ResetUsageWindows(context.Context, int64, bool, bool, bool, time.Time) error {
	r.writes++
	return nil
}

func (r *unversionedResetQuotaRepo) ResetDailyUsage(context.Context, int64, *time.Time, time.Time) error {
	r.writes++
	return nil
}

func newResetQuotaSvc(stub *resetQuotaUserSubRepoStub) *SubscriptionService {
	return NewSubscriptionService(groupRepoNoop{}, stub, nil, nil, nil)
}

func TestAdminResetQuota_ResetBoth(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{ID: 1, UserID: 10, GroupID: 20},
	}
	svc := newResetQuotaSvc(stub)
	resetAt := time.Date(2026, 7, 1, 10, 37, 42, 123, time.UTC)
	svc.now = func() time.Time { return resetAt }

	result, err := svc.AdminResetQuota(context.Background(), 1, true, true, false)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, stub.resetDailyCalled, "应调用 ResetDailyUsage")
	require.True(t, stub.resetWeeklyCalled, "应调用 ResetWeeklyUsage")
	require.False(t, stub.resetMonthlyCalled, "不应调用 ResetMonthlyUsage")
	require.Equal(t, resetAt, stub.windowStart)
	require.Equal(t, resetAt, *result.DailyWindowStart)
	require.Equal(t, resetAt, *result.WeeklyWindowStart)
}

func TestAdminResetQuota_ResetDailyOnly(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{ID: 2, UserID: 10, GroupID: 20},
	}
	svc := newResetQuotaSvc(stub)

	result, err := svc.AdminResetQuota(context.Background(), 2, true, false, false)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, stub.resetDailyCalled, "应调用 ResetDailyUsage")
	require.False(t, stub.resetWeeklyCalled, "不应调用 ResetWeeklyUsage")
	require.False(t, stub.resetMonthlyCalled, "不应调用 ResetMonthlyUsage")
}

func TestAdminResetQuota_ResetWeeklyOnly(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{ID: 3, UserID: 10, GroupID: 20},
	}
	svc := newResetQuotaSvc(stub)

	result, err := svc.AdminResetQuota(context.Background(), 3, false, true, false)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, stub.resetDailyCalled, "不应调用 ResetDailyUsage")
	require.True(t, stub.resetWeeklyCalled, "应调用 ResetWeeklyUsage")
	require.False(t, stub.resetMonthlyCalled, "不应调用 ResetMonthlyUsage")
}

func TestAdminResetQuota_BothFalseReturnsError(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{ID: 7, UserID: 10, GroupID: 20},
	}
	svc := newResetQuotaSvc(stub)

	_, err := svc.AdminResetQuota(context.Background(), 7, false, false, false)

	require.ErrorIs(t, err, ErrInvalidInput)
	require.False(t, stub.resetDailyCalled)
	require.False(t, stub.resetWeeklyCalled)
	require.False(t, stub.resetMonthlyCalled)
}

func TestAdminResetQuota_SubscriptionNotFound(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{sub: nil}
	svc := newResetQuotaSvc(stub)

	_, err := svc.AdminResetQuota(context.Background(), 999, true, true, true)

	require.ErrorIs(t, err, ErrSubscriptionNotFound)
	require.False(t, stub.resetDailyCalled)
	require.False(t, stub.resetWeeklyCalled)
	require.False(t, stub.resetMonthlyCalled)
}

func TestAdminResetQuota_ResetDailyUsageError(t *testing.T) {
	dbErr := errors.New("db error")
	stub := &resetQuotaUserSubRepoStub{
		sub:           &UserSubscription{ID: 4, UserID: 10, GroupID: 20},
		resetDailyErr: dbErr,
	}
	svc := newResetQuotaSvc(stub)

	_, err := svc.AdminResetQuota(context.Background(), 4, true, true, false)

	require.ErrorIs(t, err, dbErr)
	require.True(t, stub.resetDailyCalled)
	require.True(t, stub.resetWeeklyCalled, "原子重置应在一次调用中提交所选窗口")
}

func TestAdminResetQuota_ResetWeeklyUsageError(t *testing.T) {
	dbErr := errors.New("db error")
	stub := &resetQuotaUserSubRepoStub{
		sub:            &UserSubscription{ID: 5, UserID: 10, GroupID: 20},
		resetWeeklyErr: dbErr,
	}
	svc := newResetQuotaSvc(stub)

	_, err := svc.AdminResetQuota(context.Background(), 5, false, true, false)

	require.ErrorIs(t, err, dbErr)
	require.True(t, stub.resetWeeklyCalled)
}

func TestAdminResetQuota_ResetMonthlyOnly(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{ID: 8, UserID: 10, GroupID: 20},
	}
	svc := newResetQuotaSvc(stub)

	result, err := svc.AdminResetQuota(context.Background(), 8, false, false, true)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, stub.resetDailyCalled, "不应调用 ResetDailyUsage")
	require.False(t, stub.resetWeeklyCalled, "不应调用 ResetWeeklyUsage")
	require.True(t, stub.resetMonthlyCalled, "应调用 ResetMonthlyUsage")
}

func TestAdminResetQuota_BeforeStartsAtSameDayPreservesAutomaticBoundary(t *testing.T) {
	startsAt := time.Date(2026, 7, 1, 15, 0, 0, 0, time.UTC)
	resetAt := time.Date(2026, 7, 1, 10, 37, 42, 123, time.UTC)
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{
			ID:        10,
			UserID:    10,
			GroupID:   20,
			StartsAt:  startsAt,
			ExpiresAt: startsAt.Add(45 * 24 * time.Hour),
		},
	}
	svc := newResetQuotaSvc(stub)
	svc.now = func() time.Time { return resetAt }

	result, err := svc.AdminResetQuota(context.Background(), 10, false, false, true)

	require.NoError(t, err)
	require.Equal(t, resetAt, *result.MonthlyWindowStart)
	boundary, ok := result.automaticWindowStartAt(result.MonthlyWindowStart, 30*24*time.Hour, resetAt.Add(30*24*time.Hour))
	require.True(t, ok)
	require.Equal(t, resetAt.Add(30*24*time.Hour), boundary)
}

func TestAdminResetQuota_ResetMonthlyUsageError(t *testing.T) {
	dbErr := errors.New("db error")
	stub := &resetQuotaUserSubRepoStub{
		sub:             &UserSubscription{ID: 9, UserID: 10, GroupID: 20},
		resetMonthlyErr: dbErr,
	}
	svc := newResetQuotaSvc(stub)

	_, err := svc.AdminResetQuota(context.Background(), 9, false, false, true)

	require.ErrorIs(t, err, dbErr)
	require.True(t, stub.resetMonthlyCalled)
}

func TestAdminResetQuota_ReturnsRefreshedSub(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{
			ID:            6,
			UserID:        10,
			GroupID:       20,
			DailyUsageUSD: 99.9,
		},
	}

	svc := newResetQuotaSvc(stub)
	result, err := svc.AdminResetQuota(context.Background(), 6, true, false, false)

	require.NoError(t, err)
	// ResetUsageWindows stub 会将 sub.DailyUsageUSD 归零，
	// 服务应返回第二次 GetByID 的刷新值而非初始的 99.9
	require.Equal(t, float64(0), result.DailyUsageUSD, "返回的订阅应反映已归零的用量")
	require.True(t, stub.resetDailyCalled)
}

func TestAdminResetQuota_UsesCommittedResetVersionForCacheInvalidation(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{
		sub:          &UserSubscription{ID: 10, UserID: 10, GroupID: 20},
		resetVersion: 1234567890123456000,
	}
	cache := &resetQuotaVersionedCache{}
	svc := newResetQuotaSvc(stub)
	svc.billingCacheService = &BillingCacheService{cache: cache}

	_, err := svc.AdminResetQuota(context.Background(), 10, true, false, false)

	require.NoError(t, err)
	require.Equal(t, stub.resetVersion, cache.version, "manual reset must tombstone cached usage at its committed row version")
}

func TestAdminResetQuota_OuterTransactionInvalidatesAfterCommit(t *testing.T) {
	client := newPaymentOrderLifecycleTestClient(t)
	tx, err := client.Tx(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })
	stub := &resetQuotaUserSubRepoStub{
		sub:          &UserSubscription{ID: 12, UserID: 10, GroupID: 20},
		resetVersion: 1234567890123456000,
	}
	cache := &resetQuotaVersionedCache{}
	svc := newResetQuotaSvc(stub)
	svc.billingCacheService = &BillingCacheService{cache: cache}

	_, err = svc.AdminResetQuota(dbent.NewTxContext(context.Background(), tx), 12, true, false, false)
	require.NoError(t, err)
	require.Zero(t, cache.version, "outer transaction must retain reset cache state until commit")

	require.NoError(t, tx.Commit())
	require.Equal(t, stub.resetVersion, cache.version)
}

func TestCheckAndResetWindows_UsesCommittedResetVersionForCacheInvalidation(t *testing.T) {
	now := time.Now()
	windowStart := now.Add(-48 * time.Hour)
	stub := &resetQuotaUserSubRepoStub{
		sub: &UserSubscription{
			ID: 11, UserID: 10, GroupID: 20,
			StartsAt: now.Add(-72 * time.Hour), ExpiresAt: now.Add(72 * time.Hour),
			Status: SubscriptionStatusActive, DailyWindowStart: &windowStart,
		},
		resetVersion: 2234567890123456000,
	}
	cache := &resetQuotaVersionedCache{}
	svc := newResetQuotaSvc(stub)
	svc.billingCacheService = &BillingCacheService{cache: cache}

	err := svc.CheckAndResetWindows(context.Background(), stub.sub)

	require.NoError(t, err)
	require.Equal(t, stub.resetVersion, cache.version, "automatic reset must tombstone cached usage at its committed row version")
}

func TestAdminResetQuota_RejectsMissingVersionedResetCapability(t *testing.T) {
	repo := &unversionedResetQuotaRepo{sub: &UserSubscription{ID: 13, UserID: 10, GroupID: 20}}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

	_, err := svc.AdminResetQuota(context.Background(), 13, true, false, false)

	require.Error(t, err)
	require.Zero(t, repo.writes, "manual reset must fail before a versionless write")
}

func TestAdminResetQuota_RejectsUnversionedCacheBeforeReset(t *testing.T) {
	stub := &resetQuotaUserSubRepoStub{sub: &UserSubscription{ID: 14, UserID: 10, GroupID: 20}, resetVersion: 1}
	svc := newResetQuotaSvc(stub)
	svc.billingCacheService = &BillingCacheService{cache: &billingCacheWorkerStub{}}

	_, err := svc.AdminResetQuota(context.Background(), 14, true, false, false)

	require.Error(t, err)
	require.False(t, stub.resetDailyCalled, "manual reset must fail before writing when cache tombstones are unavailable")
}

func TestCheckAndResetWindows_RejectsMissingVersionedResetCapability(t *testing.T) {
	now := time.Now()
	windowStart := now.Add(-48 * time.Hour)
	repo := &unversionedResetQuotaRepo{sub: &UserSubscription{
		ID: 15, UserID: 10, GroupID: 20,
		StartsAt: now.Add(-72 * time.Hour), ExpiresAt: now.Add(72 * time.Hour),
		Status: SubscriptionStatusActive, DailyWindowStart: &windowStart,
	}}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

	err := svc.CheckAndResetWindows(context.Background(), repo.sub)

	require.Error(t, err)
	require.Zero(t, repo.writes, "automatic reset must fail before a versionless write")
}

func TestCheckAndResetWindows_RejectsUnversionedCacheBeforeReset(t *testing.T) {
	now := time.Now()
	windowStart := now.Add(-48 * time.Hour)
	stub := &resetQuotaUserSubRepoStub{sub: &UserSubscription{
		ID: 16, UserID: 10, GroupID: 20,
		StartsAt: now.Add(-72 * time.Hour), ExpiresAt: now.Add(72 * time.Hour),
		Status: SubscriptionStatusActive, DailyWindowStart: &windowStart,
	}, resetVersion: 1}
	svc := newResetQuotaSvc(stub)
	svc.billingCacheService = &BillingCacheService{cache: &billingCacheWorkerStub{}}

	err := svc.CheckAndResetWindows(context.Background(), stub.sub)

	require.Error(t, err)
	require.False(t, stub.resetDailyCalled, "automatic reset must fail before writing when cache tombstones are unavailable")
}

func TestCheckAndResetWindows_PreflightsAllExpiredWindowCapabilitiesBeforeWriting(t *testing.T) {
	now := time.Now()
	dailyStart := now.Add(-48 * time.Hour)
	weeklyStart := now.Add(-14 * 24 * time.Hour)
	repo := &dailyOnlyVersionedResetQuotaRepo{unversionedResetQuotaRepo: unversionedResetQuotaRepo{sub: &UserSubscription{
		ID: 17, UserID: 10, GroupID: 20,
		StartsAt: now.Add(-72 * time.Hour), ExpiresAt: now.Add(72 * time.Hour),
		Status: SubscriptionStatusActive, DailyWindowStart: &dailyStart, WeeklyWindowStart: &weeklyStart,
	}}}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)

	err := svc.CheckAndResetWindows(context.Background(), repo.sub)

	require.ErrorIs(t, err, ErrSubscriptionUsageVersioningUnavailable)
	require.Zero(t, repo.writes, "automatic reset must preflight all expired window capabilities before any write")
}
