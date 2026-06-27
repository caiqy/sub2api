package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/stretchr/testify/require"
)

type dailyResetTrackingUserSubRepo struct {
	userSubRepoNoop

	resetDailyCalled bool
}

func (r *dailyResetTrackingUserSubRepo) ResetDailyUsage(context.Context, int64, time.Time) error {
	r.resetDailyCalled = true
	return nil
}

func TestAssignOrExtendSubscription_ExpiredDailyCardStartsNewOneTimeQuota(t *testing.T) {
	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{ID: 1, SubscriptionType: SubscriptionTypeSubscription},
	}
	subRepo := newSubscriptionUserSubRepoStub()
	oldStart := time.Now().AddDate(0, 0, -3)
	oldWindowStart := startOfDay(oldStart)
	subRepo.seed(&UserSubscription{
		ID:                 100,
		UserID:             200,
		GroupID:            1,
		StartsAt:           oldStart,
		ExpiresAt:          oldStart.AddDate(0, 0, 1),
		Status:             SubscriptionStatusExpired,
		DailyWindowStart:   &oldWindowStart,
		WeeklyWindowStart:  &oldWindowStart,
		MonthlyWindowStart: &oldWindowStart,
		DailyUsageUSD:      10,
		WeeklyUsageUSD:     20,
		MonthlyUsageUSD:    30,
		Notes:              "old",
	})
	svc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)

	renewed, reused, err := svc.AssignOrExtendSubscription(context.Background(), &AssignSubscriptionInput{
		UserID:       200,
		GroupID:      1,
		ValidityDays: 1,
		Notes:        "new",
	})

	require.NoError(t, err)
	require.True(t, reused)
	require.True(t, renewed.HasOneTimeDailyQuota(), "过期后重新购买 1 日卡仍应被识别为一次性日额度")
	require.Equal(t, SubscriptionStatusActive, renewed.Status)
	require.True(t, renewed.StartsAt.After(oldStart), "重新购买过期订阅时应重置当前周期 StartsAt")
	require.False(t, renewed.ExpiresAt.After(renewed.StartsAt.AddDate(0, 0, 1)))
	require.NotNil(t, renewed.DailyWindowStart)
	require.Equal(t, renewed.StartsAt, *renewed.DailyWindowStart)
	require.Equal(t, 0.0, renewed.DailyUsageUSD)
	require.Equal(t, 0.0, renewed.WeeklyUsageUSD)
	require.Equal(t, 0.0, renewed.MonthlyUsageUSD)
	require.Equal(t, "old\nnew", renewed.Notes)
}

func TestUserSubscriptionNeedsDailyReset_DailyCardKeepsOneTimeQuota(t *testing.T) {
	start := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	dailyWindowStart := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	sub := &UserSubscription{
		StartsAt:         start,
		ExpiresAt:        start.Add(24 * time.Hour),
		DailyWindowStart: &dailyWindowStart,
		DailyUsageUSD:    10,
	}

	require.True(t, sub.HasOneTimeDailyQuota())
	require.False(t, sub.NeedsDailyResetAt(dailyWindowStart.Add(25*time.Hour)), "日卡应作为一次性配额，跨 0 点后不再刷新日额度")
}

func TestUserSubscriptionNeedsDailyReset_MultiDaySubscriptionStillRefreshes(t *testing.T) {
	start := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	dailyWindowStart := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	sub := &UserSubscription{
		StartsAt:         start,
		ExpiresAt:        start.AddDate(0, 0, 2),
		DailyWindowStart: &dailyWindowStart,
	}

	require.False(t, sub.HasOneTimeDailyQuota())
	require.False(t, sub.NeedsDailyResetAt(dailyWindowStart.Add(24*time.Hour)), "legacy midnight 不应让多日订阅提前到 0 点刷新")
	require.True(t, sub.NeedsDailyResetAt(start.Add(24*time.Hour)), "多日订阅应按 StartsAt 精确 24 小时窗口刷新")
}

func TestUserSubscriptionDailyResetTime_DailyCardReturnsExpiry(t *testing.T) {
	start := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	dailyWindowStart := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	expiresAt := start.Add(24 * time.Hour)
	sub := &UserSubscription{
		StartsAt:         start,
		ExpiresAt:        expiresAt,
		DailyWindowStart: &dailyWindowStart,
	}

	resetAt := sub.DailyResetTime()
	require.NotNil(t, resetAt)
	require.Equal(t, expiresAt, *resetAt, "日卡展示的日额度结束时间应为订阅过期时间")
}

func TestCheckAndResetWindows_DailyCardDoesNotResetDailyUsage(t *testing.T) {
	now := time.Now()
	startsAt := now.Add(-23 * time.Hour)
	dailyWindowStart := now.Add(-25 * time.Hour)
	repo := &dailyResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:               1,
		UserID:           10,
		GroupID:          20,
		StartsAt:         startsAt,
		ExpiresAt:        startsAt.Add(24 * time.Hour),
		DailyUsageUSD:    10,
		DailyWindowStart: &dailyWindowStart,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.False(t, repo.resetDailyCalled, "日卡作为一次性配额，过了 24 小时日窗口也不应重置 daily usage")
	require.Equal(t, 10.0, sub.DailyUsageUSD)
}

func TestCheckAndResetWindows_MultiDaySubscriptionStillResetsDailyUsage(t *testing.T) {
	now := time.Now()
	startsAt := now.Add(-48 * time.Hour)
	dailyWindowStart := now.Add(-25 * time.Hour)
	repo := &dailyResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:               1,
		UserID:           10,
		GroupID:          20,
		StartsAt:         startsAt,
		ExpiresAt:        startsAt.AddDate(0, 0, 2),
		DailyUsageUSD:    10,
		DailyWindowStart: &dailyWindowStart,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.True(t, repo.resetDailyCalled, "多日订阅仍应重置过期 daily window")
	require.Equal(t, 0.0, sub.DailyUsageUSD)
}

func TestSubscriptionWeeklyWindowAnchorsToExactStart(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sub := &UserSubscription{
		WeeklyWindowStart: &start,
	}

	weekDuration := 7 * 24 * time.Hour

	t.Run("before boundary", func(t *testing.T) {
		before := start.Add(weekDuration - time.Nanosecond)
		require.False(t, sub.NeedsWeeklyResetAt(before))
		require.Nil(t, sub.NextWeeklyWindowStart(before))
		require.Nil(t, sub.NextWeeklyWindowStart(start.Add(-time.Hour)))
	})

	t.Run("exact boundary", func(t *testing.T) {
		boundary := start.Add(weekDuration)
		require.True(t, sub.NeedsWeeklyResetAt(boundary))
		next := sub.NextWeeklyWindowStart(boundary)
		require.NotNil(t, next)
		require.Equal(t, boundary, *next)
	})

	t.Run("stale catch-up", func(t *testing.T) {
		now := start.Add(24 * 24 * time.Hour)
		require.True(t, sub.NeedsWeeklyResetAt(now))
		next := sub.NextWeeklyWindowStart(now)
		require.NotNil(t, next)
		require.Equal(t, start.Add(21*24*time.Hour), *next)
	})
}

func TestSubscriptionMonthlyWindowAnchorsToExactStart(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sub := &UserSubscription{
		MonthlyWindowStart: &start,
	}

	monthDuration := 30 * 24 * time.Hour

	t.Run("before boundary", func(t *testing.T) {
		before := start.Add(monthDuration - time.Nanosecond)
		require.False(t, sub.NeedsMonthlyResetAt(before))
		require.Nil(t, sub.NextMonthlyWindowStart(before))
		require.Nil(t, sub.NextMonthlyWindowStart(start.Add(-time.Hour)))
	})

	t.Run("exact boundary", func(t *testing.T) {
		boundary := start.Add(monthDuration)
		require.True(t, sub.NeedsMonthlyResetAt(boundary))
		next := sub.NextMonthlyWindowStart(boundary)
		require.NotNil(t, next)
		require.Equal(t, boundary, *next)
	})

	t.Run("stale catch-up", func(t *testing.T) {
		now := start.Add(75 * 24 * time.Hour)
		require.True(t, sub.NeedsMonthlyResetAt(now))
		next := sub.NextMonthlyWindowStart(now)
		require.NotNil(t, next)
		require.Equal(t, start.Add(60*24*time.Hour), *next)
	})
}

type activateWindowsTrackingUserSubRepo struct {
	userSubRepoNoop

	activateCalled bool
	activateStart  time.Time
}

func (r *activateWindowsTrackingUserSubRepo) ActivateWindows(_ context.Context, _ int64, start time.Time) error {
	r.activateCalled = true
	r.activateStart = start
	return nil
}

func TestCheckAndActivateWindowUsesExactStartsAt(t *testing.T) {
	startsAt := time.Date(2026, 6, 15, 14, 30, 17, 0, time.UTC)
	repo := &activateWindowsTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:       1,
		StartsAt: startsAt,
	}

	err := svc.CheckAndActivateWindow(context.Background(), sub)

	require.NoError(t, err)
	require.True(t, repo.activateCalled, "ActivateWindows should be called for unactivated window")
	require.Equal(t, startsAt, repo.activateStart, "should use exact StartsAt, not truncated to midnight")
}

func TestCheckAndActivateWindow_L1WaitAfterActivate(t *testing.T) {
	startsAt := time.Date(2026, 6, 15, 14, 30, 17, 0, time.UTC)
	repo := &activateWindowsTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	cache := &trackingSubCache{}
	svc.subCacheL1 = cache
	sub := &UserSubscription{ID: 1, UserID: 10, GroupID: 20, StartsAt: startsAt}

	err := svc.CheckAndActivateWindow(context.Background(), sub)

	require.NoError(t, err)
	require.True(t, repo.activateCalled)
	require.Equal(t, []string{subCacheKey(10, 20)}, cache.deletedKeys)
	require.Equal(t, []string{"del:" + subCacheKey(10, 20), "wait"}, cache.operations)
}

func TestRenewedExpiredSubscriptionUsesExactStartsAt(t *testing.T) {
	startsAt := time.Date(2026, 6, 15, 14, 30, 17, 0, time.UTC)
	renewed := renewedSubscriptionTerm(&UserSubscription{
		ID:                 100,
		UserID:             200,
		GroupID:            1,
		StartsAt:           startsAt.Add(-24 * time.Hour),
		ExpiresAt:          startsAt.Add(-1 * time.Hour),
		Status:             SubscriptionStatusExpired,
		DailyWindowStart:   nil,
		WeeklyWindowStart:  nil,
		MonthlyWindowStart: nil,
		DailyUsageUSD:      10,
		WeeklyUsageUSD:     20,
		MonthlyUsageUSD:    30,
		Notes:              "old",
	}, "test-note", startsAt, startsAt.AddDate(0, 0, 7))

	require.NotNil(t, renewed.DailyWindowStart)
	require.NotNil(t, renewed.WeeklyWindowStart)
	require.NotNil(t, renewed.MonthlyWindowStart)
	require.Equal(t, startsAt, *renewed.DailyWindowStart)
	require.Equal(t, startsAt, *renewed.WeeklyWindowStart)
	require.Equal(t, startsAt, *renewed.MonthlyWindowStart)
	require.Equal(t, 0.0, renewed.DailyUsageUSD)
	require.Equal(t, 0.0, renewed.WeeklyUsageUSD)
	require.Equal(t, 0.0, renewed.MonthlyUsageUSD)
}

type windowResetTrackingUserSubRepo struct {
	userSubRepoNoop

	resetDailyCalled   bool
	resetDailyStart    time.Time
	resetWeeklyCalled  bool
	resetWeeklyStart   time.Time
	resetWeeklyErr     error
	resetMonthlyCalled bool
	resetMonthlyStart  time.Time
	resetMonthlyErr    error
}

func (r *windowResetTrackingUserSubRepo) ResetDailyUsage(_ context.Context, _ int64, start time.Time) error {
	r.resetDailyCalled = true
	r.resetDailyStart = start
	return nil
}

func (r *windowResetTrackingUserSubRepo) ResetWeeklyUsage(_ context.Context, _ int64, start time.Time) error {
	r.resetWeeklyCalled = true
	r.resetWeeklyStart = start
	if r.resetWeeklyErr != nil {
		return r.resetWeeklyErr
	}
	return nil
}

func (r *windowResetTrackingUserSubRepo) ResetMonthlyUsage(_ context.Context, _ int64, start time.Time) error {
	r.resetMonthlyCalled = true
	r.resetMonthlyStart = start
	if r.resetMonthlyErr != nil {
		return r.resetMonthlyErr
	}
	return nil
}

func TestCheckAndResetWindows_InvalidatesCacheAfterPartialResetFailure(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	startsAt := now.Add(-75 * 24 * time.Hour)
	repo := &windowResetTrackingUserSubRepo{resetMonthlyErr: errors.New("monthly reset failed")}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: 1e4, MaxCost: 1e3, BufferItems: 64})
	require.NoError(t, err)
	svc.subCacheL1 = cache
	cacheKey := subCacheKey(10, 20)
	cache.Set(cacheKey, &UserSubscription{ID: 999}, 1)
	cache.Wait()
	time.Sleep(10 * time.Millisecond)
	if _, ok := cache.Get(cacheKey); !ok {
		t.Skip("ristretto admission skipped Set; cannot verify Wait semantics")
	}
	dailyWindowStart := startsAt
	monthlyWindowStart := startsAt
	sub := &UserSubscription{
		ID:                 1,
		UserID:             10,
		GroupID:            20,
		StartsAt:           startsAt,
		ExpiresAt:          now.Add(24 * time.Hour),
		DailyWindowStart:   &dailyWindowStart,
		MonthlyWindowStart: &monthlyWindowStart,
	}

	err = svc.CheckAndResetWindows(context.Background(), sub)

	require.Error(t, err)
	require.True(t, repo.resetDailyCalled)
	require.True(t, repo.resetMonthlyCalled)
	_, ok := cache.Get(cacheKey)
	require.False(t, ok, "部分 reset 成功后即使后续失败，也必须失效 L1 缓存")
}

func TestAdminResetQuota_InvalidatesCacheAfterPartialResetFailure(t *testing.T) {
	now := time.Now().UTC()
	repo := &windowResetTrackingUserSubRepo{resetWeeklyErr: errors.New("weekly reset failed")}
	repo.userSubRepoNoop = userSubRepoNoop{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: 10, MaxCost: 10, BufferItems: 64})
	require.NoError(t, err)
	svc.subCacheL1 = cache
	cacheKey := subCacheKey(10, 20)
	cache.Set(cacheKey, &UserSubscription{ID: 999}, 1)
	cache.Wait()
	repoWithSub := &adminResetPartialFailureRepo{windowResetTrackingUserSubRepo: repo, sub: &UserSubscription{
		ID:        1,
		UserID:    10,
		GroupID:   20,
		StartsAt:  now.Add(-24 * time.Hour),
		ExpiresAt: now.Add(24 * time.Hour),
	}}
	svc.userSubRepo = repoWithSub

	_, err = svc.AdminResetQuota(context.Background(), 1, true, true, false)

	require.Error(t, err)
	require.True(t, repo.resetDailyCalled)
	require.True(t, repo.resetWeeklyCalled)
	_, ok := cache.Get(cacheKey)
	require.False(t, ok, "AdminResetQuota 部分 reset 成功后即使后续失败，也必须失效 L1 缓存")
}

type adminResetPartialFailureRepo struct {
	*windowResetTrackingUserSubRepo
	sub *UserSubscription
}

func (r *adminResetPartialFailureRepo) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func TestSevenDayCardDoesNotReceiveSecondWeeklyQuotaBeforeExpiry(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	startsAt := now.Add(-6 * 24 * time.Hour)
	weeklyWindowStart := startsAt
	repo := &windowResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:                1,
		UserID:            10,
		GroupID:           20,
		StartsAt:          startsAt,
		ExpiresAt:         startsAt.Add(7 * 24 * time.Hour),
		WeeklyWindowStart: &weeklyWindowStart,
		WeeklyUsageUSD:    10,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.False(t, repo.resetWeeklyCalled, "7 日卡在过期前不应获得第二次周配额，周窗口还不满 7 天")
	require.Equal(t, 10.0, sub.WeeklyUsageUSD)
}

func TestFourteenDayCardReceivesExactlyOneWeeklyResetBeforeExpiry(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	startsAt := now.Add(-8 * 24 * time.Hour)
	weeklyWindowStart := startsAt
	repo := &windowResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:                1,
		UserID:            10,
		GroupID:           20,
		StartsAt:          startsAt,
		ExpiresAt:         startsAt.Add(14 * 24 * time.Hour),
		WeeklyWindowStart: &weeklyWindowStart,
		WeeklyUsageUSD:    10,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.True(t, repo.resetWeeklyCalled, "14 日卡应在第一个周窗口过期后获得一次重置")
	expectedNewStart := startsAt.Add(7 * 24 * time.Hour)
	require.Equal(t, expectedNewStart, repo.resetWeeklyStart, "新周窗口应从原窗口起始 +7 天开始")
	require.Equal(t, 0.0, sub.WeeklyUsageUSD)
}

func TestThirtyDayCardDoesNotReceiveSecondMonthlyQuotaBeforeExpiry(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	startsAt := now.Add(-28 * 24 * time.Hour)
	monthlyWindowStart := startsAt
	repo := &windowResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:                 1,
		UserID:             10,
		GroupID:            20,
		StartsAt:           startsAt,
		ExpiresAt:          startsAt.Add(30 * 24 * time.Hour),
		MonthlyWindowStart: &monthlyWindowStart,
		MonthlyUsageUSD:    10,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.False(t, repo.resetMonthlyCalled, "30 日卡在过期前不应获得第二次月配额，月窗口还不满 30 天")
	require.Equal(t, 10.0, sub.MonthlyUsageUSD)
}

func TestSixtyDayCardReceivesExactlyOneMonthlyResetBeforeExpiry(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	startsAt := now.Add(-31 * 24 * time.Hour)
	monthlyWindowStart := startsAt
	repo := &windowResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:                 1,
		UserID:             10,
		GroupID:            20,
		StartsAt:           startsAt,
		ExpiresAt:          startsAt.Add(60 * 24 * time.Hour),
		MonthlyWindowStart: &monthlyWindowStart,
		MonthlyUsageUSD:    10,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.True(t, repo.resetMonthlyCalled, "60 日卡应在第一个月窗口过期后获得一次重置")
	expectedNewStart := startsAt.Add(30 * 24 * time.Hour)
	require.Equal(t, expectedNewStart, repo.resetMonthlyStart, "新月窗口应从原窗口起始 +30 天开始")
	require.Equal(t, 0.0, sub.MonthlyUsageUSD)
}

func TestCheckAndResetWindows_StaleWeeklyUsesNextWindowStartNotStartOfDay(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	startsAt := now.Add(-24 * 24 * time.Hour)
	weeklyWindowStart := startsAt
	repo := &windowResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:                1,
		UserID:            10,
		GroupID:           20,
		StartsAt:          startsAt,
		ExpiresAt:         startsAt.AddDate(0, 0, 50),
		WeeklyWindowStart: &weeklyWindowStart,
		WeeklyUsageUSD:    10,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.True(t, repo.resetWeeklyCalled)

	expectedNextWindowStart := startsAt.Add(21 * 24 * time.Hour)

	require.Equal(t, expectedNextWindowStart, repo.resetWeeklyStart,
		"陈旧周窗口应使用 NextWeeklyWindowStart 返回值（start+21d），不是 startOfDay(now)")
	require.NotEqual(t, startOfDay(now), repo.resetWeeklyStart,
		"不应使用统一 startOfDay(time.Now()) 作为重置起点")
	require.NotEqual(t, startsAt.Add(7*24*time.Hour), repo.resetWeeklyStart,
		"不应只加一个周期，应跳到最近的过期边界（start+21d 而非 start+7d）")
	require.Equal(t, 0.0, sub.WeeklyUsageUSD)
}

func TestCheckAndResetWindows_NilSubscriptionNoop(t *testing.T) {
	svc := NewSubscriptionService(groupRepoNoop{}, userSubRepoNoop{}, nil, nil, nil)
	err := svc.CheckAndResetWindows(context.Background(), nil)
	require.NoError(t, err)
}

type activeRenewalUserSubRepoStub struct {
	userSubRepoNoop

	sub *UserSubscription

	extendExpiryCalled bool
	updateStatusCalled bool
	updateNotesCalled  bool
	updateCalled       bool
}

func (r *activeRenewalUserSubRepoStub) GetByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.UserID != userID || r.sub.GroupID != groupID {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *activeRenewalUserSubRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *activeRenewalUserSubRepoStub) ExtendExpiry(_ context.Context, _ int64, newExpiresAt time.Time) error {
	r.extendExpiryCalled = true
	if r.sub != nil {
		r.sub.ExpiresAt = newExpiresAt
	}
	return nil
}

func (r *activeRenewalUserSubRepoStub) UpdateStatus(_ context.Context, _ int64, status string) error {
	r.updateStatusCalled = true
	if r.sub != nil {
		r.sub.Status = status
	}
	return nil
}

func (r *activeRenewalUserSubRepoStub) UpdateNotes(_ context.Context, _ int64, notes string) error {
	r.updateNotesCalled = true
	if r.sub != nil {
		r.sub.Notes = notes
	}
	return nil
}

func (r *activeRenewalUserSubRepoStub) Update(_ context.Context, sub *UserSubscription) error {
	r.updateCalled = true
	if r.sub != nil {
		*r.sub = *sub
	}
	return nil
}

func TestActiveRenewalDoesNotRechargeQuotaWindows(t *testing.T) {
	now := time.Now().UTC()
	start := now.Add(-3 * 24 * time.Hour)
	oldExpiresAt := now.Add(4 * 24 * time.Hour)
	notes := "keep"

	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{ID: 1, SubscriptionType: SubscriptionTypeSubscription},
	}
	subRepo := &activeRenewalUserSubRepoStub{
		sub: &UserSubscription{
			ID:                 100,
			UserID:             200,
			GroupID:            1,
			StartsAt:           start,
			ExpiresAt:          oldExpiresAt,
			Status:             SubscriptionStatusActive,
			DailyWindowStart:   &start,
			WeeklyWindowStart:  &start,
			MonthlyWindowStart: &start,
			DailyUsageUSD:      5,
			WeeklyUsageUSD:     40,
			MonthlyUsageUSD:    80,
			Notes:              notes,
		},
	}
	svc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)

	renewed, reused, err := svc.AssignOrExtendSubscription(context.Background(), &AssignSubscriptionInput{
		UserID:       200,
		GroupID:      1,
		ValidityDays: 7,
		Notes:        "new notes ignored for unexpired renewal",
	})

	require.NoError(t, err)
	require.True(t, reused, "未过期续订应返回 reused=true")

	require.True(t, renewed.ExpiresAt.After(oldExpiresAt), "ExpiresAt 应被延长")
	require.Equal(t, oldExpiresAt.AddDate(0, 0, 7), renewed.ExpiresAt)

	require.NotNil(t, renewed.DailyWindowStart)
	require.True(t, renewed.DailyWindowStart.Equal(start), "DailyWindowStart 应保持原值")
	require.NotNil(t, renewed.WeeklyWindowStart)
	require.True(t, renewed.WeeklyWindowStart.Equal(start), "WeeklyWindowStart 应保持原值")
	require.NotNil(t, renewed.MonthlyWindowStart)
	require.True(t, renewed.MonthlyWindowStart.Equal(start), "MonthlyWindowStart 应保持原值")

	require.Equal(t, 5.0, renewed.DailyUsageUSD, "DailyUsageUSD 应保持原值")
	require.Equal(t, 40.0, renewed.WeeklyUsageUSD, "WeeklyUsageUSD 应保持原值")
	require.Equal(t, 80.0, renewed.MonthlyUsageUSD, "MonthlyUsageUSD 应保持原值")
	require.Equal(t, notes, renewed.Notes, "未过期续订只延长 ExpiresAt，不应追加 Notes")

	require.True(t, subRepo.extendExpiryCalled, "续期应调用 ExtendExpiry")
	require.False(t, subRepo.updateStatusCalled, "活跃订阅续期不应调用 UpdateStatus")
	require.False(t, subRepo.updateNotesCalled, "未过期续订不应调用 UpdateNotes")
}

func TestActiveRenewalDoesNotReactivateSuspendedSubscription(t *testing.T) {
	now := time.Now().UTC()
	start := now.Add(-3 * 24 * time.Hour)
	oldExpiresAt := now.Add(4 * 24 * time.Hour)

	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{ID: 1, SubscriptionType: SubscriptionTypeSubscription},
	}
	subRepo := &activeRenewalUserSubRepoStub{
		sub: &UserSubscription{
			ID:                 100,
			UserID:             200,
			GroupID:            1,
			StartsAt:           start,
			ExpiresAt:          oldExpiresAt,
			Status:             SubscriptionStatusSuspended,
			DailyWindowStart:   &start,
			WeeklyWindowStart:  &start,
			MonthlyWindowStart: &start,
			DailyUsageUSD:      5,
			WeeklyUsageUSD:     40,
			MonthlyUsageUSD:    80,
		},
	}
	svc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)

	renewed, reused, err := svc.AssignOrExtendSubscription(context.Background(), &AssignSubscriptionInput{
		UserID:       200,
		GroupID:      1,
		ValidityDays: 7,
	})

	require.NoError(t, err)
	require.True(t, reused)
	require.Equal(t, SubscriptionStatusSuspended, renewed.Status)
	require.Equal(t, oldExpiresAt.AddDate(0, 0, 7), renewed.ExpiresAt)
	require.Equal(t, 40.0, renewed.WeeklyUsageUSD)
	require.False(t, subRepo.updateStatusCalled, "未过期 suspended 续订只延长 ExpiresAt，不应恢复 active")
}

func TestValidateAndCheckLimits_DailyCardDoesNotAllowSecondQuotaAfterMidnight(t *testing.T) {
	start := time.Now().Add(-23 * time.Hour)
	dailyWindowStart := time.Now().Add(-25 * time.Hour)
	dailyLimit := 10.0
	sub := &UserSubscription{
		Status:           SubscriptionStatusActive,
		StartsAt:         start,
		ExpiresAt:        start.Add(24 * time.Hour),
		DailyWindowStart: &dailyWindowStart,
		DailyUsageUSD:    dailyLimit + 0.01,
	}
	group := &Group{
		SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD:    &dailyLimit,
	}
	svc := NewSubscriptionService(groupRepoNoop{}, userSubRepoNoop{}, nil, nil, nil)

	needsMaintenance, err := svc.ValidateAndCheckLimits(sub, group)

	require.False(t, needsMaintenance, "日卡跨过日窗口后不应触发 daily reset 维护")
	require.True(t, errors.Is(err, ErrDailyLimitExceeded))
	require.Equal(t, dailyLimit+0.01, sub.DailyUsageUSD, "热路径不应清零日卡已用额度")
}

func TestLegacyMidnightWeeklyWindowUsesStartsAtAnchor(t *testing.T) {
	startsAt := time.Date(2026, 6, 26, 15, 30, 0, 0, time.UTC)
	legacyMidnight := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	sub := &UserSubscription{
		StartsAt:          startsAt,
		WeeklyWindowStart: &legacyMidnight,
	}

	beforeReset := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	require.False(t, sub.NeedsWeeklyResetAt(beforeReset))
	require.Nil(t, sub.NextWeeklyWindowStart(beforeReset))

	atReset := time.Date(2026, 7, 3, 15, 30, 0, 0, time.UTC)
	require.True(t, sub.NeedsWeeklyResetAt(atReset))
	next := sub.NextWeeklyWindowStart(atReset)
	require.NotNil(t, next)
	require.Equal(t, atReset, *next)
}

func TestLegacyMidnightMonthlyWindowUsesStartsAtAnchor(t *testing.T) {
	startsAt := time.Date(2026, 6, 26, 15, 30, 0, 0, time.UTC)
	legacyMidnight := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	sub := &UserSubscription{
		StartsAt:           startsAt,
		MonthlyWindowStart: &legacyMidnight,
	}

	beforeReset := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	require.False(t, sub.NeedsMonthlyResetAt(beforeReset))
	require.Nil(t, sub.NextMonthlyWindowStart(beforeReset))

	atReset := time.Date(2026, 7, 26, 15, 30, 0, 0, time.UTC)
	require.True(t, sub.NeedsMonthlyResetAt(atReset))
	next := sub.NextMonthlyWindowStart(atReset)
	require.NotNil(t, next)
	require.Equal(t, atReset, *next)
}

func TestLegacyMidnightResetTimeUsesEffectiveAnchor(t *testing.T) {
	startsAt := time.Date(2026, 6, 26, 15, 30, 0, 0, time.UTC)
	legacyMidnight := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	sub := &UserSubscription{
		StartsAt:           startsAt,
		WeeklyWindowStart:  &legacyMidnight,
		MonthlyWindowStart: &legacyMidnight,
	}

	weeklyReset := sub.WeeklyResetTime()
	require.NotNil(t, weeklyReset)
	require.Equal(t, time.Date(2026, 7, 3, 15, 30, 0, 0, time.UTC), *weeklyReset,
		"legacy midnight window: WeeklyResetTime 应为下周 15:30（startsAt+7d）")

	monthlyReset := sub.MonthlyResetTime()
	require.NotNil(t, monthlyReset)
	require.Equal(t, time.Date(2026, 7, 26, 15, 30, 0, 0, time.UTC), *monthlyReset,
		"legacy midnight window: MonthlyResetTime 应为 30d 后 15:30（startsAt+30d）")
}

func TestThirtyDayCardLegacyMidnightDoesNotResetMonthlyBeforeExpiry(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	startsAt := now.Add(-29 * 24 * time.Hour)
	legacyMidnight := startOfDay(startsAt)
	repo := &windowResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:                 1,
		UserID:             10,
		GroupID:            20,
		StartsAt:           startsAt,
		ExpiresAt:          startsAt.Add(30 * 24 * time.Hour),
		MonthlyWindowStart: &legacyMidnight,
		MonthlyUsageUSD:    10,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.False(t, repo.resetMonthlyCalled, "30 日卡 legacy midnight window 在到期前 1 天不应重置月配额")
	require.Equal(t, 10.0, sub.MonthlyUsageUSD)
}

func TestSevenDayCardLegacyMidnightDoesNotResetBeforeExpiry(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	startsAt := now.Add(-6*24*time.Hour + time.Hour)
	legacyMidnight := startOfDay(startsAt)
	repo := &windowResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:                1,
		UserID:            10,
		GroupID:           20,
		StartsAt:          startsAt,
		ExpiresAt:         startsAt.Add(7 * 24 * time.Hour),
		WeeklyWindowStart: &legacyMidnight,
		WeeklyUsageUSD:    10,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.False(t, repo.resetWeeklyCalled, "7 日卡 legacy midnight window 在到期前 1 小时不应调用 ResetWeeklyUsage")
	require.Equal(t, 10.0, sub.WeeklyUsageUSD)
}

func TestFixLegacyMidnightAnchor_FirstPeriodMidnightReturnsStartsAt(t *testing.T) {
	startsAt := time.Date(2026, 6, 15, 14, 30, 0, 0, time.UTC)
	// 首个周期内的 legacy midnight 应回到 StartsAt，避免把首刷推迟到第二个周期。
	adminResetMidnight := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)

	weekly := fixLegacyMidnightAnchor(startsAt, &adminResetMidnight, 7*24*time.Hour)
	monthly := fixLegacyMidnightAnchor(startsAt, &adminResetMidnight, 30*24*time.Hour)

	require.NotNil(t, weekly)
	require.NotNil(t, monthly)
	require.Equal(t, startsAt, *weekly, "首个周周期内的 midnight 应纠偏回 StartsAt")
	require.Equal(t, startsAt, *monthly, "首个月周期内的 midnight 应纠偏回 StartsAt")
}

func TestFixLegacyMidnightAnchor_StartsAtMidnightNotCorrected(t *testing.T) {
	startsAt := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	legacyMidnight := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	weekly := fixLegacyMidnightAnchor(startsAt, &legacyMidnight, 7*24*time.Hour)
	monthly := fixLegacyMidnightAnchor(startsAt, &legacyMidnight, 30*24*time.Hour)

	require.NotNil(t, weekly)
	require.NotNil(t, monthly)
	// StartsAt=00:00 是合法锚点，不应被纠偏
	require.Equal(t, legacyMidnight, *weekly, "StartsAt=00:00 时 midnight windowStart 不应被纠偏")
	require.Equal(t, legacyMidnight, *monthly, "StartsAt=00:00 时 midnight windowStart 不应被纠偏")
}

func TestFutureWeeklyResetTime_StaleWindow(t *testing.T) {
	loc := time.UTC
	startsAt := time.Date(2026, 1, 1, 12, 0, 0, 0, loc)
	sub := &UserSubscription{
		StartsAt:          startsAt,
		WeeklyWindowStart: &startsAt,
	}

	// now 已过 3 个周期
	now := startsAt.Add(25 * 24 * time.Hour)
	next := sub.FutureWeeklyResetTime(now)
	require.NotNil(t, next)
	require.Equal(t, startsAt.Add(28*24*time.Hour), *next, "stale weekly: 应返回下一个未来 reset（start+4*7d）")

	// 刚好在边界上
	atBoundary := startsAt.Add(7 * 24 * time.Hour)
	nextAt := sub.FutureWeeklyResetTime(atBoundary)
	require.NotNil(t, nextAt)
	require.Equal(t, startsAt.Add(14*24*time.Hour), *nextAt, "在边界上应返回再下一个重置时间")
}

func TestFutureMonthlyResetTime_StaleWindow(t *testing.T) {
	loc := time.UTC
	startsAt := time.Date(2026, 1, 1, 12, 0, 0, 0, loc)
	sub := &UserSubscription{
		StartsAt:           startsAt,
		MonthlyWindowStart: &startsAt,
	}

	// now 已过 2 个周期
	now := startsAt.Add(65 * 24 * time.Hour)
	next := sub.FutureMonthlyResetTime(now)
	require.NotNil(t, next)
	require.Equal(t, startsAt.Add(90*24*time.Hour), *next, "stale monthly: 应返回下一个未来 reset（start+3*30d）")
}

func TestFixLegacyMidnightAnchor_AlignedWindowStaysStable(t *testing.T) {
	startsAt := time.Date(2026, 6, 15, 14, 30, 0, 0, time.UTC)
	// 窗口已对齐到 StartsAt+7d（一个周期后），非 midnight
	alignedWeekly := startsAt.Add(7 * 24 * time.Hour)
	alignedMonthly := startsAt.Add(30 * 24 * time.Hour)

	weekly := fixLegacyMidnightAnchor(startsAt, &alignedWeekly, 7*24*time.Hour)
	monthly := fixLegacyMidnightAnchor(startsAt, &alignedMonthly, 30*24*time.Hour)

	require.NotNil(t, weekly)
	require.NotNil(t, monthly)
	require.Equal(t, alignedWeekly, *weekly, "已对齐的非 midnight 窗口应保持原值")
	require.Equal(t, alignedMonthly, *monthly, "已对齐的非 midnight 窗口应保持原值")
}

func TestLegacyMidnightLaterPeriodDoesNotMoveWindowIntoFuture(t *testing.T) {
	startsAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	legacyMidnight := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	sub := &UserSubscription{
		StartsAt:          startsAt,
		WeeklyWindowStart: &legacyMidnight,
	}

	effective := sub.effectiveWeeklyWindowStart()

	require.NotNil(t, effective)
	require.False(t, effective.After(now), "later legacy midnight 纠偏不能把有效窗口推到未来")
	require.Equal(t, startsAt.Add(7*24*time.Hour), *effective, "later legacy midnight 应回落到最近已到达边界")
}

func TestExtendSubscription_ExpiredSubscriptionResetsQuotaWindows(t *testing.T) {
	now := time.Now().UTC()
	oldStartsAt := now.Add(-10 * 24 * time.Hour)
	oldWindowStart := oldStartsAt

	subRepo := &activeRenewalUserSubRepoStub{
		sub: &UserSubscription{
			ID:                 100,
			UserID:             200,
			GroupID:            1,
			StartsAt:           oldStartsAt,
			ExpiresAt:          oldStartsAt.AddDate(0, 0, 1),
			Status:             SubscriptionStatusExpired,
			DailyWindowStart:   &oldWindowStart,
			WeeklyWindowStart:  &oldWindowStart,
			MonthlyWindowStart: &oldWindowStart,
			DailyUsageUSD:      10,
			WeeklyUsageUSD:     20,
			MonthlyUsageUSD:    30,
			Notes:              "old",
		},
	}
	svc := NewSubscriptionService(groupRepoNoop{}, subRepo, nil, nil, nil)

	before := time.Now().UTC()
	renewed, err := svc.ExtendSubscription(context.Background(), 100, 7)
	after := time.Now().UTC()

	require.NoError(t, err)
	require.Equal(t, SubscriptionStatusActive, renewed.Status)
	require.True(t, renewed.StartsAt.After(before) || renewed.StartsAt.Equal(before))
	require.True(t, renewed.StartsAt.Before(after) || renewed.StartsAt.Equal(after))
	require.True(t, renewed.ExpiresAt.After(now))

	require.NotNil(t, renewed.DailyWindowStart)
	require.True(t, renewed.DailyWindowStart.Equal(renewed.StartsAt))
	require.NotNil(t, renewed.WeeklyWindowStart)
	require.True(t, renewed.WeeklyWindowStart.Equal(renewed.StartsAt))
	require.NotNil(t, renewed.MonthlyWindowStart)
	require.True(t, renewed.MonthlyWindowStart.Equal(renewed.StartsAt))

	require.Equal(t, 0.0, renewed.DailyUsageUSD)
	require.Equal(t, 0.0, renewed.WeeklyUsageUSD)
	require.Equal(t, 0.0, renewed.MonthlyUsageUSD)

	require.True(t, subRepo.updateCalled, "过期正向延长应调用 Update 而非 ExtendExpiry")
	require.False(t, subRepo.extendExpiryCalled, "过期正向延长不应调用 ExtendExpiry")
	require.False(t, subRepo.updateStatusCalled, "过期正向延长不应单独调用 UpdateStatus")
}

func TestExtendSubscription_ActiveDoesNotRechargeQuotaWindows(t *testing.T) {
	now := time.Now().UTC()
	start := now.Add(-3 * 24 * time.Hour)
	oldExpiresAt := now.Add(4 * 24 * time.Hour)
	windowStart := start

	subRepo := &activeRenewalUserSubRepoStub{
		sub: &UserSubscription{
			ID:                 100,
			UserID:             200,
			GroupID:            1,
			StartsAt:           start,
			ExpiresAt:          oldExpiresAt,
			Status:             SubscriptionStatusActive,
			DailyWindowStart:   &windowStart,
			WeeklyWindowStart:  &windowStart,
			MonthlyWindowStart: &windowStart,
			DailyUsageUSD:      5,
			WeeklyUsageUSD:     40,
			MonthlyUsageUSD:    80,
		},
	}
	svc := NewSubscriptionService(groupRepoNoop{}, subRepo, nil, nil, nil)

	renewed, err := svc.ExtendSubscription(context.Background(), 100, 7)

	require.NoError(t, err)

	require.True(t, renewed.ExpiresAt.After(oldExpiresAt), "ExpiresAt 应被延长")
	require.Equal(t, oldExpiresAt.AddDate(0, 0, 7), renewed.ExpiresAt)

	require.True(t, renewed.StartsAt.Equal(start), "StartsAt 应保持原值")

	require.NotNil(t, renewed.DailyWindowStart)
	require.True(t, renewed.DailyWindowStart.Equal(start), "DailyWindowStart 应保持原值")
	require.NotNil(t, renewed.WeeklyWindowStart)
	require.True(t, renewed.WeeklyWindowStart.Equal(start), "WeeklyWindowStart 应保持原值")
	require.NotNil(t, renewed.MonthlyWindowStart)
	require.True(t, renewed.MonthlyWindowStart.Equal(start), "MonthlyWindowStart 应保持原值")

	require.Equal(t, 5.0, renewed.DailyUsageUSD, "DailyUsageUSD 应保持原值")
	require.Equal(t, 40.0, renewed.WeeklyUsageUSD, "WeeklyUsageUSD 应保持原值")
	require.Equal(t, 80.0, renewed.MonthlyUsageUSD, "MonthlyUsageUSD 应保持原值")

	require.True(t, subRepo.extendExpiryCalled, "活跃订阅续期应调用 ExtendExpiry")
	require.False(t, subRepo.updateCalled, "活跃订阅续期不应调用 Update（不重置窗口）")
	require.False(t, subRepo.updateStatusCalled, "活跃订阅续期不应调用 UpdateStatus")
}

func TestExtendSubscription_UnexpiredDoesNotReactivateExpiredStatus(t *testing.T) {
	now := time.Now().UTC()
	start := now.Add(-3 * 24 * time.Hour)
	oldExpiresAt := now.Add(4 * 24 * time.Hour)
	windowStart := start

	subRepo := &activeRenewalUserSubRepoStub{
		sub: &UserSubscription{
			ID:                 100,
			UserID:             200,
			GroupID:            1,
			StartsAt:           start,
			ExpiresAt:          oldExpiresAt,
			Status:             SubscriptionStatusExpired,
			DailyWindowStart:   &windowStart,
			WeeklyWindowStart:  &windowStart,
			MonthlyWindowStart: &windowStart,
			DailyUsageUSD:      5,
			WeeklyUsageUSD:     40,
			MonthlyUsageUSD:    80,
		},
	}
	svc := NewSubscriptionService(groupRepoNoop{}, subRepo, nil, nil, nil)

	renewed, err := svc.ExtendSubscription(context.Background(), 100, 7)

	require.NoError(t, err)
	require.Equal(t, oldExpiresAt.AddDate(0, 0, 7), renewed.ExpiresAt)
	require.Equal(t, SubscriptionStatusExpired, renewed.Status)
	require.False(t, subRepo.updateStatusCalled, "未过期 ExtendSubscription 只延长 ExpiresAt，不应恢复 active")
}

func TestCheckAndResetWindows_StaleMonthlyUsesNextWindowStartNotStartOfDay(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	startsAt := now.Add(-75 * 24 * time.Hour)
	monthlyWindowStart := startsAt
	repo := &windowResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:                 1,
		UserID:             10,
		GroupID:            20,
		StartsAt:           startsAt,
		ExpiresAt:          startsAt.AddDate(0, 0, 90),
		MonthlyWindowStart: &monthlyWindowStart,
		MonthlyUsageUSD:    10,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.True(t, repo.resetMonthlyCalled, "月窗口落后 75 天应触发重置")

	expectedNextWindowStart := startsAt.Add(60 * 24 * time.Hour)

	require.Equal(t, expectedNextWindowStart, repo.resetMonthlyStart,
		"陈旧月窗口应使用 NextMonthlyWindowStart 返回值（start+60d）")
	require.NotEqual(t, startOfDay(now), repo.resetMonthlyStart,
		"不应使用统一 startOfDay(time.Now()) 作为重置起点")
	require.NotEqual(t, startsAt.Add(30*24*time.Hour), repo.resetMonthlyStart,
		"不应只加一个周期，应跳到最近的过期边界（start+60d 而非 start+30d）")
	require.Equal(t, 0.0, sub.MonthlyUsageUSD)
}

func TestCheckAndResetWindows_StaleDailyUsesLatestWindowStart(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	startsAt := now.Add(-72 * time.Hour) // 3 days ago
	dailyWindowStart := startsAt
	repo := &windowResetTrackingUserSubRepo{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	sub := &UserSubscription{
		ID:               1,
		UserID:           10,
		GroupID:          20,
		StartsAt:         startsAt,
		ExpiresAt:        startsAt.AddDate(0, 0, 30),
		DailyWindowStart: &dailyWindowStart,
		DailyUsageUSD:    10,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.True(t, repo.resetDailyCalled, "落后 3 天的每日窗口应触发重置")
	expectedNewStart := startsAt.Add(72 * time.Hour)
	require.Equal(t, expectedNewStart, repo.resetDailyStart,
		"stale daily: 应跳到最新 24h 边界（start+72h），而非 start+24h")
	require.NotEqual(t, startsAt.Add(24*time.Hour), repo.resetDailyStart,
		"不应只加一个 24h 周期")
	require.Equal(t, 0.0, sub.DailyUsageUSD)
}

func TestLegacyMidnightDailyWindowUsesStartsAtAnchor(t *testing.T) {
	startsAt := time.Date(2026, 6, 26, 15, 30, 0, 0, time.UTC)
	legacyMidnight := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	sub := &UserSubscription{
		StartsAt:         startsAt,
		ExpiresAt:        startsAt.Add(7 * 24 * time.Hour),
		DailyWindowStart: &legacyMidnight,
	}

	beforeReset := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	require.False(t, sub.NeedsDailyResetAt(beforeReset))
	require.Nil(t, sub.NextDailyWindowStart(beforeReset))

	atReset := time.Date(2026, 6, 27, 15, 30, 0, 0, time.UTC)
	require.True(t, sub.NeedsDailyResetAt(atReset))
	next := sub.NextDailyWindowStart(atReset)
	require.NotNil(t, next)
	require.Equal(t, atReset, *next)
}

func TestCreateSubscriptionAnchorsWindowsToStartsAt(t *testing.T) {
	groupRepo := &subscriptionGroupRepoStub{group: &Group{ID: 20, SubscriptionType: SubscriptionTypeSubscription}}
	subRepo := newSubscriptionUserSubRepoStub()
	svc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)

	sub, err := svc.AssignSubscription(context.Background(), &AssignSubscriptionInput{
		UserID:       10,
		GroupID:      20,
		ValidityDays: 7,
	})

	require.NoError(t, err)
	require.NotNil(t, sub.DailyWindowStart)
	require.NotNil(t, sub.WeeklyWindowStart)
	require.NotNil(t, sub.MonthlyWindowStart)
	require.Equal(t, sub.StartsAt, *sub.DailyWindowStart)
	require.Equal(t, sub.StartsAt, *sub.WeeklyWindowStart)
	require.Equal(t, sub.StartsAt, *sub.MonthlyWindowStart)
}

func TestFixLegacyMidnightAnchor_DelayedFirstUse(t *testing.T) {
	startsAt := time.Date(2026, 6, 26, 15, 30, 0, 0, time.UTC)
	// 旧版首次激活发生在 StartsAt 之后 2 天的 00:00
	windowStart := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)

	weekly := fixLegacyMidnightAnchor(startsAt, &windowStart, 7*24*time.Hour)
	monthly := fixLegacyMidnightAnchor(startsAt, &windowStart, 30*24*time.Hour)

	require.NotNil(t, weekly)
	require.NotNil(t, monthly)

	// 延迟首次使用但仍处于首个周期内的 legacy midnight 应纠偏回 StartsAt，
	// 避免把首个重置点推迟到第二个周期。
	expectedWeeklyAnchor := startsAt
	expectedMonthlyAnchor := startsAt

	require.Equal(t, expectedWeeklyAnchor, *weekly,
		"StartsAt=6/26 15:30, window=6/28 00:00: weekly 应纠偏回 StartsAt")
	require.Equal(t, expectedMonthlyAnchor, *monthly,
		"StartsAt=6/26 15:30, window=6/28 00:00: monthly 应纠偏回 StartsAt")

	// 验证纠偏后的锚点不改变原有时分秒
	require.Equal(t, startsAt.Hour(), weekly.Hour())
	require.Equal(t, startsAt.Minute(), weekly.Minute())
	require.Equal(t, startsAt.Hour(), monthly.Hour())
	require.Equal(t, startsAt.Minute(), monthly.Minute())
}

func TestFixLegacyMidnightAnchor_FirstPeriodMidnightReturnsStartsAt_Weekly(t *testing.T) {
	startsAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// 非对齐 midnight（不在 StartsAt+n*period 边界上）
	nonAligned := startsAt.Add(7*24*time.Hour - 12*time.Hour)

	weekly := fixLegacyMidnightAnchor(startsAt, &nonAligned, 7*24*time.Hour)

	require.NotNil(t, weekly)
	require.Equal(t, startsAt, *weekly,
		"首个周期内的非对齐 midnight 应纠偏回 startsAt，避免推迟首刷")
}

func TestCheckAndResetWindows_L1WaitAfterReset(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	startsAt := now.Add(-25 * time.Hour)
	dailyWindowStart := startsAt
	repo := &windowResetTrackingUserSubRepo{}

	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	cache := &trackingSubCache{}
	svc.subCacheL1 = cache

	sub := &UserSubscription{
		ID:               1,
		UserID:           10,
		GroupID:          20,
		StartsAt:         startsAt,
		ExpiresAt:        startsAt.AddDate(0, 0, 2),
		DailyWindowStart: &dailyWindowStart,
		DailyUsageUSD:    10,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)
	require.NoError(t, err)
	require.True(t, repo.resetDailyCalled)

	require.Equal(t, []string{subCacheKey(10, 20)}, cache.deletedKeys)
	require.Equal(t, 1, cache.waitCalls, "CheckAndResetWindows reset 后必须调用 waitSubCacheL1")
}

func TestAssignOrExtendSubscription_L1WaitAfterInvalidate(t *testing.T) {
	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{ID: 20, SubscriptionType: SubscriptionTypeSubscription},
	}
	subRepo := &activeRenewalUserSubRepoStub{
		sub: &UserSubscription{
			ID:        1,
			UserID:    10,
			GroupID:   20,
			StartsAt:  time.Now().Add(-24 * time.Hour),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Status:    SubscriptionStatusActive,
		},
	}
	svc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)
	cache := &trackingSubCache{}
	svc.subCacheL1 = cache

	_, _, err := svc.AssignOrExtendSubscription(context.Background(), &AssignSubscriptionInput{
		UserID:       10,
		GroupID:      20,
		ValidityDays: 7,
	})
	require.NoError(t, err)

	require.Equal(t, []string{subCacheKey(10, 20)}, cache.deletedKeys)
	require.Equal(t, 1, cache.waitCalls, "AssignOrExtendSubscription 既有订阅续期后必须调用 waitSubCacheL1")
}

func TestExtendSubscription_L1WaitAfterInvalidate(t *testing.T) {
	now := time.Now().UTC()
	oldExpiresAt := now.Add(4 * 24 * time.Hour)
	subRepo := &activeRenewalUserSubRepoStub{
		sub: &UserSubscription{
			ID:        1,
			UserID:    10,
			GroupID:   20,
			StartsAt:  now.Add(-24 * time.Hour),
			ExpiresAt: oldExpiresAt,
			Status:    SubscriptionStatusActive,
		},
	}
	svc := NewSubscriptionService(groupRepoNoop{}, subRepo, nil, nil, nil)
	cache := &trackingSubCache{}
	svc.subCacheL1 = cache

	_, err := svc.ExtendSubscription(context.Background(), 1, 7)

	require.NoError(t, err)
	require.Equal(t, []string{subCacheKey(10, 20)}, cache.deletedKeys)
	require.Equal(t, 1, cache.waitCalls, "ExtendSubscription 后必须调用 waitSubCacheL1")
}

func TestAssignSubscriptionCreate_L1WaitAfterInvalidate(t *testing.T) {
	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{ID: 20, SubscriptionType: SubscriptionTypeSubscription},
	}
	subRepo := newSubscriptionUserSubRepoStub()
	svc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)
	cache := &trackingSubCache{}
	svc.subCacheL1 = cache

	_, err := svc.AssignSubscription(context.Background(), &AssignSubscriptionInput{
		UserID:       10,
		GroupID:      20,
		ValidityDays: 7,
	})

	require.NoError(t, err)
	require.Equal(t, []string{subCacheKey(10, 20)}, cache.deletedKeys)
	require.Equal(t, 1, cache.waitCalls, "AssignSubscription 新建后必须调用 waitSubCacheL1")
}

func TestAssignOrExtendSubscriptionCreate_L1WaitAfterInvalidate(t *testing.T) {
	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{ID: 20, SubscriptionType: SubscriptionTypeSubscription},
	}
	subRepo := newSubscriptionUserSubRepoStub()
	svc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)
	cache := &trackingSubCache{}
	svc.subCacheL1 = cache

	_, reused, err := svc.AssignOrExtendSubscription(context.Background(), &AssignSubscriptionInput{
		UserID:       10,
		GroupID:      20,
		ValidityDays: 7,
	})

	require.NoError(t, err)
	require.False(t, reused)
	require.Equal(t, []string{subCacheKey(10, 20)}, cache.deletedKeys)
	require.Equal(t, 1, cache.waitCalls, "AssignOrExtendSubscription 新建后必须调用 waitSubCacheL1")
}

func TestAdminResetQuota_L1WaitAfterSuccess(t *testing.T) {
	now := time.Now().UTC()
	repo := &windowResetTrackingUserSubRepo{}
	repoWithSub := &adminResetPartialFailureRepo{windowResetTrackingUserSubRepo: repo, sub: &UserSubscription{
		ID:        1,
		UserID:    10,
		GroupID:   20,
		StartsAt:  now.Add(-24 * time.Hour),
		ExpiresAt: now.Add(24 * time.Hour),
	}}
	svc := NewSubscriptionService(groupRepoNoop{}, repoWithSub, nil, nil, nil)
	cache := &trackingSubCache{}
	svc.subCacheL1 = cache

	_, err := svc.AdminResetQuota(context.Background(), 1, true, true, true)

	require.NoError(t, err)
	require.True(t, repo.resetDailyCalled)
	require.True(t, repo.resetWeeklyCalled)
	require.True(t, repo.resetMonthlyCalled)
	require.Equal(t, []string{subCacheKey(10, 20)}, cache.deletedKeys)
	require.Equal(t, 1, cache.waitCalls, "AdminResetQuota 成功后必须调用 waitSubCacheL1")
}

func TestRevokeSubscription_L1WaitAfterDelete(t *testing.T) {
	repo := &revokeSubscriptionRepoStub{sub: &UserSubscription{ID: 1, UserID: 10, GroupID: 20}}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	cache := &trackingSubCache{}
	svc.subCacheL1 = cache

	err := svc.RevokeSubscription(context.Background(), 1)

	require.NoError(t, err)
	require.True(t, repo.deleteCalled)
	require.Equal(t, []string{subCacheKey(10, 20)}, cache.deletedKeys)
	require.Equal(t, 1, cache.waitCalls, "RevokeSubscription 删除后必须调用 waitSubCacheL1")
}

func TestRecordUsage_L1WaitAfterIncrement(t *testing.T) {
	repo := &recordUsageRepoStub{sub: &UserSubscription{ID: 1, UserID: 10, GroupID: 20}}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	cache := &trackingSubCache{}
	svc.subCacheL1 = cache

	err := svc.RecordUsage(context.Background(), 1, 1.25)

	require.NoError(t, err)
	require.Equal(t, 1.25, repo.incrementCost)
	require.Equal(t, []string{subCacheKey(10, 20)}, cache.deletedKeys)
	require.Equal(t, []string{"del:" + subCacheKey(10, 20), "wait"}, cache.operations)
}

func TestValidateSubscription_L1WaitAfterExpireStatusUpdate(t *testing.T) {
	repo := &validateSubscriptionRepoStub{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	cache := &trackingSubCache{}
	svc.subCacheL1 = cache
	sub := &UserSubscription{
		ID:        1,
		UserID:    10,
		GroupID:   20,
		Status:    SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(-time.Hour),
	}

	err := svc.ValidateSubscription(context.Background(), sub)

	require.ErrorIs(t, err, ErrSubscriptionExpired)
	require.Equal(t, SubscriptionStatusExpired, repo.status)
	require.Equal(t, []string{subCacheKey(10, 20)}, cache.deletedKeys)
	require.Equal(t, []string{"del:" + subCacheKey(10, 20), "wait"}, cache.operations)
}

type validateSubscriptionRepoStub struct {
	userSubRepoNoop
	status string
}

func (r *validateSubscriptionRepoStub) UpdateStatus(_ context.Context, _ int64, status string) error {
	r.status = status
	return nil
}

type recordUsageRepoStub struct {
	userSubRepoNoop
	sub           *UserSubscription
	incrementCost float64
}

func (r *recordUsageRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *recordUsageRepoStub) IncrementUsage(_ context.Context, id int64, costUSD float64) error {
	if r.sub == nil || r.sub.ID != id {
		return ErrSubscriptionNotFound
	}
	r.incrementCost = costUSD
	return nil
}

type revokeSubscriptionRepoStub struct {
	userSubRepoNoop
	sub          *UserSubscription
	deleteCalled bool
}

func (r *revokeSubscriptionRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *revokeSubscriptionRepoStub) Delete(_ context.Context, id int64) error {
	if r.sub == nil || r.sub.ID != id {
		return ErrSubscriptionNotFound
	}
	r.deleteCalled = true
	return nil
}

func seedSubscriptionCache(t *testing.T, svc *SubscriptionService, userID, groupID int64) (*ristretto.Cache, string) {
	t.Helper()
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: 1e4, MaxCost: 1e3, BufferItems: 64})
	require.NoError(t, err)
	svc.subCacheL1 = cache
	key := subCacheKey(userID, groupID)
	cache.Set(key, "old_value", 1)
	cache.Wait()
	time.Sleep(10 * time.Millisecond)
	if _, ok := cache.Get(key); !ok {
		t.Skip("ristretto admission skipped Set; cannot verify Wait semantics")
	}
	return cache, key
}

type trackingSubCache struct {
	deletedKeys []string
	waitCalls   int
	operations  []string
}

func (c *trackingSubCache) Del(key any) {
	keyString := key.(string)
	c.deletedKeys = append(c.deletedKeys, keyString)
	c.operations = append(c.operations, "del:"+keyString)
}

func (c *trackingSubCache) Wait() {
	c.waitCalls++
	c.operations = append(c.operations, "wait")
}

func (c *trackingSubCache) Get(any) (any, bool) {
	return nil, false
}

func (c *trackingSubCache) SetWithTTL(any, any, int64, time.Duration) bool {
	return true
}
