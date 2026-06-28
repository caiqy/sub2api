package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Task 5: 验证 calculateProgress 纯函数行为正确 ---

func newTestSubscriptionService() *SubscriptionService {
	return &SubscriptionService{}
}

func ptrFloat64(v float64) *float64  { return &v }
func ptrTime(t time.Time) *time.Time { return &t }

func TestCalculateProgress_BasicFields(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()

	sub := &UserSubscription{
		ID:        100,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
	}
	group := &Group{
		Name: "Premium",
	}

	progress := svc.calculateProgress(sub, group)

	assert.Equal(t, int64(100), progress.ID)
	assert.Equal(t, "Premium", progress.GroupName)
	assert.Equal(t, sub.ExpiresAt, progress.ExpiresAt)
	assert.True(t, progress.ExpiresInDays == 29 || progress.ExpiresInDays == 30, "ExpiresInDays should be 29 or 30, got %d", progress.ExpiresInDays)
	assert.Nil(t, progress.Daily, "无日限额时 Daily 应为 nil")
	assert.Nil(t, progress.Weekly, "无周限额时 Weekly 应为 nil")
	assert.Nil(t, progress.Monthly, "无月限额时 Monthly 应为 nil")
}

func TestCalculateProgress_DailyUsage(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()
	dailyStart := now.Add(-12 * time.Hour)

	sub := &UserSubscription{
		ID:               1,
		ExpiresAt:        now.Add(10 * 24 * time.Hour),
		DailyUsageUSD:    3.0,
		DailyWindowStart: ptrTime(dailyStart),
	}
	group := &Group{
		Name:          "Pro",
		DailyLimitUSD: ptrFloat64(10.0),
	}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Daily, "有日限额和窗口时 Daily 不应为 nil")
	assert.Equal(t, 10.0, progress.Daily.LimitUSD)
	assert.Equal(t, 3.0, progress.Daily.UsedUSD)
	assert.Equal(t, 7.0, progress.Daily.RemainingUSD)
	assert.Equal(t, 30.0, progress.Daily.Percentage)
	assert.Equal(t, dailyStart, progress.Daily.WindowStart)
}

func TestCalculateProgress_DailyCardUsesExpiryAsDailyResetTime(t *testing.T) {
	svc := newTestSubscriptionService()
	startsAt := time.Now().Add(-12 * time.Hour)
	dailyStart := time.Date(startsAt.Year(), startsAt.Month(), startsAt.Day(), 0, 0, 0, 0, startsAt.Location())
	expiresAt := startsAt.Add(24 * time.Hour)

	sub := &UserSubscription{
		ID:               1,
		StartsAt:         startsAt,
		ExpiresAt:        expiresAt,
		DailyUsageUSD:    3.0,
		DailyWindowStart: ptrTime(dailyStart),
	}
	group := &Group{
		Name:          "Daily",
		DailyLimitUSD: ptrFloat64(10.0),
	}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Daily, "日卡有日限额和窗口时 Daily 不应为 nil")
	assert.Equal(t, expiresAt, progress.Daily.ResetsAt, "日卡的一次性日额度结束时间应为订阅过期时间")
}

func TestCalculateProgress_WeeklyUsage(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()
	weeklyStart := now.Add(-3 * 24 * time.Hour)

	sub := &UserSubscription{
		ID:                1,
		ExpiresAt:         now.Add(10 * 24 * time.Hour),
		WeeklyUsageUSD:    25.0,
		WeeklyWindowStart: ptrTime(weeklyStart),
	}
	group := &Group{
		Name:           "Pro",
		WeeklyLimitUSD: ptrFloat64(50.0),
	}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Weekly, "有周限额和窗口时 Weekly 不应为 nil")
	assert.Equal(t, 50.0, progress.Weekly.LimitUSD)
	assert.Equal(t, 25.0, progress.Weekly.UsedUSD)
	assert.Equal(t, 25.0, progress.Weekly.RemainingUSD)
	assert.Equal(t, 50.0, progress.Weekly.Percentage)
}

func TestCalculateProgress_MonthlyUsage(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()
	monthlyStart := now.Add(-15 * 24 * time.Hour)

	sub := &UserSubscription{
		ID:                 1,
		ExpiresAt:          now.Add(10 * 24 * time.Hour),
		MonthlyUsageUSD:    80.0,
		MonthlyWindowStart: ptrTime(monthlyStart),
	}
	group := &Group{
		Name:            "Enterprise",
		MonthlyLimitUSD: ptrFloat64(100.0),
	}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Monthly, "有月限额和窗口时 Monthly 不应为 nil")
	assert.Equal(t, 100.0, progress.Monthly.LimitUSD)
	assert.Equal(t, 80.0, progress.Monthly.UsedUSD)
	assert.Equal(t, 20.0, progress.Monthly.RemainingUSD)
	assert.Equal(t, 80.0, progress.Monthly.Percentage)
}

func TestCalculateProgress_OverLimit_ClampedTo100Percent(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()

	sub := &UserSubscription{
		ID:               1,
		ExpiresAt:        now.Add(10 * 24 * time.Hour),
		DailyUsageUSD:    15.0, // 超过限额
		DailyWindowStart: ptrTime(now.Add(-1 * time.Hour)),
	}
	group := &Group{
		Name:          "Pro",
		DailyLimitUSD: ptrFloat64(10.0),
	}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Daily)
	assert.Equal(t, 100.0, progress.Daily.Percentage, "超额使用应被截断为 100%")
	assert.Equal(t, 0.0, progress.Daily.RemainingUSD, "超额使用时剩余应为 0")
}

func TestCalculateProgress_NoWindowStart_NoProgress(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()

	// 有限额但无窗口起始时间（订阅未激活）
	sub := &UserSubscription{
		ID:             1,
		ExpiresAt:      now.Add(10 * 24 * time.Hour),
		DailyUsageUSD:  0,
		WeeklyUsageUSD: 0,
	}
	group := &Group{
		Name:           "Pro",
		DailyLimitUSD:  ptrFloat64(10.0),
		WeeklyLimitUSD: ptrFloat64(50.0),
	}

	progress := svc.calculateProgress(sub, group)

	assert.Nil(t, progress.Daily, "无 DailyWindowStart 时 Daily 应为 nil")
	assert.Nil(t, progress.Weekly, "无 WeeklyWindowStart 时 Weekly 应为 nil")
}

func TestCalculateProgress_AllLimits(t *testing.T) {
	svc := newTestSubscriptionService()
	now := time.Now()

	sub := &UserSubscription{
		ID:                 1,
		ExpiresAt:          now.Add(10 * 24 * time.Hour),
		DailyUsageUSD:      5.0,
		WeeklyUsageUSD:     20.0,
		MonthlyUsageUSD:    60.0,
		DailyWindowStart:   ptrTime(now.Add(-6 * time.Hour)),
		WeeklyWindowStart:  ptrTime(now.Add(-3 * 24 * time.Hour)),
		MonthlyWindowStart: ptrTime(now.Add(-15 * 24 * time.Hour)),
	}
	group := &Group{
		Name:            "Full",
		DailyLimitUSD:   ptrFloat64(10.0),
		WeeklyLimitUSD:  ptrFloat64(50.0),
		MonthlyLimitUSD: ptrFloat64(100.0),
	}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Daily)
	require.NotNil(t, progress.Weekly)
	require.NotNil(t, progress.Monthly)

	assert.Equal(t, 50.0, progress.Daily.Percentage)
	assert.Equal(t, 40.0, progress.Weekly.Percentage)
	assert.Equal(t, 60.0, progress.Monthly.Percentage)
}

func TestCalculateProgress_ExpiredSubscription(t *testing.T) {
	svc := newTestSubscriptionService()

	sub := &UserSubscription{
		ID:        1,
		ExpiresAt: time.Now().Add(-24 * time.Hour), // 已过期
	}
	group := &Group{Name: "Expired"}

	progress := svc.calculateProgress(sub, group)

	assert.Equal(t, 0, progress.ExpiresInDays, "过期订阅的剩余天数应为 0")
}

func TestCalculateProgress_LegacyMidnightProgressShowsEffectiveAnchor(t *testing.T) {
	svc := newTestSubscriptionService()
	loc := time.UTC
	startsAt := time.Date(2024, 6, 15, 10, 30, 0, 0, loc)
	weeklyMidnight := time.Date(2024, 6, 15, 0, 0, 0, 0, loc)
	monthlyMidnight := time.Date(2024, 6, 15, 0, 0, 0, 0, loc)

	sub := &UserSubscription{
		ID:                 1,
		StartsAt:           startsAt,
		ExpiresAt:          startsAt.Add(90 * 24 * time.Hour),
		WeeklyUsageUSD:     10.0,
		MonthlyUsageUSD:    20.0,
		WeeklyWindowStart:  ptrTime(weeklyMidnight),
		MonthlyWindowStart: ptrTime(monthlyMidnight),
	}
	group := &Group{
		Name:            "Legacy",
		WeeklyLimitUSD:  ptrFloat64(100.0),
		MonthlyLimitUSD: ptrFloat64(200.0),
	}

	progress := svc.calculateProgress(sub, group)

	// Legacy midnight: WindowStart 应以 effective anchor 为基准（子时刻非 00:00），
	// 且因为是 2024 年的陈旧窗口，WindowStart 已前移对齐到当前周期。
	// 验证 WindowStart 时刻保留 StartsAt 的时分秒（而非 midnight 的 00:00）。
	require.NotNil(t, progress.Weekly)
	assert.Equal(t, startsAt.Hour(), progress.Weekly.WindowStart.Hour(), "legacy 周 WindowStart 的小时应等于 StartsAt 的小时，而非 00")
	assert.Equal(t, startsAt.Minute(), progress.Weekly.WindowStart.Minute(), "legacy 周 WindowStart 的分钟应等于 StartsAt 的分钟，而非 00")

	require.NotNil(t, progress.Monthly)
	assert.Equal(t, startsAt.Hour(), progress.Monthly.WindowStart.Hour(), "legacy 月 WindowStart 的小时应等于 StartsAt 的小时，而非 00")
	assert.Equal(t, startsAt.Minute(), progress.Monthly.WindowStart.Minute(), "legacy 月 WindowStart 的分钟应等于 StartsAt 的分钟，而非 00")
}

func TestCalculateProgress_StaleWindowShowsNextFutureReset(t *testing.T) {
	svc := newTestSubscriptionService()
	loc := time.UTC
	startsAt := time.Date(2024, 1, 1, 12, 0, 0, 0, loc)
	windowStart := startsAt // 窗口锚点，已落后多个周期

	sub := &UserSubscription{
		ID:                 1,
		StartsAt:           startsAt,
		ExpiresAt:          startsAt.Add(365 * 24 * time.Hour),
		WeeklyUsageUSD:     10.0,
		MonthlyUsageUSD:    20.0,
		WeeklyWindowStart:  ptrTime(windowStart),
		MonthlyWindowStart: ptrTime(windowStart),
	}
	group := &Group{
		Name:            "Stale",
		WeeklyLimitUSD:  ptrFloat64(100.0),
		MonthlyLimitUSD: ptrFloat64(200.0),
	}

	progress := svc.calculateProgress(sub, group)

	// stale weekly: WindowStart 应为最新对齐后的边界，ResetsAt 为下一个未来边界
	require.NotNil(t, progress.Weekly)
	ws := startsAt
	periods := int(time.Since(ws) / (7 * 24 * time.Hour))
	expectedWinStart := ws.Add(time.Duration(periods) * 7 * 24 * time.Hour)
	expectedReset := ws.Add(time.Duration(periods+1) * 7 * 24 * time.Hour)
	assert.Equal(t, expectedWinStart, progress.Weekly.WindowStart,
		"stale weekly: WindowStart 应为最新有效边界（start+periods*7d）")
	assert.Equal(t, expectedReset, progress.Weekly.ResetsAt,
		"stale weekly: ResetsAt 应为下一个未来边界（start+(periods+1)*7d），而非第一个过期边界")
	assert.Equal(t, 0.0, progress.Weekly.UsedUSD,
		"stale weekly: WindowStart 已推进到新窗口，展示用量应与 reset 后状态一致")

	// stale monthly
	require.NotNil(t, progress.Monthly)
	ms := startsAt
	mPeriods := int(time.Since(ms) / (30 * 24 * time.Hour))
	expectedMWinStart := ms.Add(time.Duration(mPeriods) * 30 * 24 * time.Hour)
	expectedMReset := ms.Add(time.Duration(mPeriods+1) * 30 * 24 * time.Hour)
	assert.Equal(t, expectedMWinStart, progress.Monthly.WindowStart,
		"stale monthly: WindowStart 应为最新有效边界")
	assert.Equal(t, expectedMReset, progress.Monthly.ResetsAt,
		"stale monthly: ResetsAt 应为下一个未来边界")
	assert.Equal(t, 0.0, progress.Monthly.UsedUSD,
		"stale monthly: WindowStart 已推进到新窗口，展示用量应与 reset 后状态一致")

	// ResetsAt 应在未来
	assert.True(t, progress.Weekly.ResetsAt.After(time.Now()), "stale weekly ResetsAt 应在未来")
	assert.True(t, progress.Monthly.ResetsAt.After(time.Now()), "stale monthly ResetsAt 应在未来")
}

func TestCalculateProgress_StaleDailyWindowShowsCurrentBoundary(t *testing.T) {
	svc := newTestSubscriptionService()
	start := time.Now().UTC().Add(-75 * time.Hour)
	dailyLimit := 100.0
	sub := &UserSubscription{
		ID:               1,
		StartsAt:         start,
		ExpiresAt:        start.Add(10 * 24 * time.Hour),
		DailyUsageUSD:    10,
		DailyWindowStart: ptrTime(start),
	}
	group := &Group{Name: "StaleDaily", DailyLimitUSD: ptrFloat64(dailyLimit)}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Daily)
	require.True(t, progress.Daily.WindowStart.After(start), "stale daily: WindowStart 应追到当前有效边界")
	require.Equal(t, 0.0, progress.Daily.UsedUSD, "stale daily: WindowStart 已推进到新窗口，展示用量应与 reset 后状态一致")
	require.True(t, progress.Daily.ResetsAt.After(progress.Daily.WindowStart), "stale daily: ResetsAt 应晚于 WindowStart")
	assert.InDelta(t, 24*time.Hour.Seconds(), progress.Daily.ResetsAt.Sub(progress.Daily.WindowStart).Seconds(), 1)
}

func TestCalculateProgress_LegacyMidnightDailyProgressShowsEffectiveAnchor(t *testing.T) {
	svc := newTestSubscriptionService()
	startsAt := time.Now().UTC().Add(12 * time.Hour).Truncate(time.Minute)
	startsAt = time.Date(startsAt.Year(), startsAt.Month(), startsAt.Day(), 15, 30, 0, 0, time.UTC)
	if !startsAt.After(time.Now().UTC()) {
		startsAt = startsAt.Add(24 * time.Hour)
	}
	legacyMidnight := startOfDay(startsAt)
	dailyLimit := 100.0
	sub := &UserSubscription{
		ID:               1,
		StartsAt:         startsAt,
		ExpiresAt:        startsAt.Add(7 * 24 * time.Hour),
		DailyUsageUSD:    10,
		DailyWindowStart: ptrTime(legacyMidnight),
	}
	group := &Group{Name: "LegacyDaily", DailyLimitUSD: ptrFloat64(dailyLimit)}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Daily)
	require.Equal(t, startsAt, progress.Daily.WindowStart)
	require.Equal(t, startsAt.Add(24*time.Hour), progress.Daily.ResetsAt)
}

func TestCalculateProgress_AlignedWindowStaysStable(t *testing.T) {
	svc := newTestSubscriptionService()
	loc := time.UTC
	startsAt := time.Date(2024, 1, 1, 12, 0, 0, 0, loc)
	// 窗口已经对齐到 StartsAt+7d / StartsAt+30d
	weeklyWin := startsAt.Add(7 * 24 * time.Hour)
	monthlyWin := startsAt.Add(30 * 24 * time.Hour)

	sub := &UserSubscription{
		ID:                 1,
		StartsAt:           startsAt,
		ExpiresAt:          startsAt.Add(365 * 24 * time.Hour),
		WeeklyUsageUSD:     10.0,
		MonthlyUsageUSD:    20.0,
		WeeklyWindowStart:  ptrTime(weeklyWin),
		MonthlyWindowStart: ptrTime(monthlyWin),
	}
	group := &Group{
		Name:            "Aligned",
		WeeklyLimitUSD:  ptrFloat64(100.0),
		MonthlyLimitUSD: ptrFloat64(200.0),
	}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Weekly)
	// 窗口非 midnight，effective anchor 保留 12:00 时刻，且因 2024 年数据已陈旧，
	// CurrentWeeklyWindowStart 会将日期推进到当前周期，但锚点时刻（12:00）保持不变。
	assert.Equal(t, weeklyWin.Hour(), progress.Weekly.WindowStart.Hour(),
		"对齐窗口的锚点小时应保持稳定")
	assert.Equal(t, weeklyWin.Minute(), progress.Weekly.WindowStart.Minute(),
		"对齐窗口的锚点分钟应保持稳定")

	require.NotNil(t, progress.Monthly)
	assert.Equal(t, monthlyWin.Hour(), progress.Monthly.WindowStart.Hour(),
		"对齐窗口的锚点小时应保持稳定")
	assert.Equal(t, monthlyWin.Minute(), progress.Monthly.WindowStart.Minute(),
		"对齐窗口的锚点分钟应保持稳定")
}

func TestCalculateProgress_ResetsInSeconds_NotNegative(t *testing.T) {
	svc := newTestSubscriptionService()
	// 使用过去的窗口起始时间，使得重置时间已过
	pastStart := time.Now().Add(-48 * time.Hour)

	sub := &UserSubscription{
		ID:               1,
		ExpiresAt:        time.Now().Add(10 * 24 * time.Hour),
		DailyUsageUSD:    1.0,
		DailyWindowStart: ptrTime(pastStart),
	}
	group := &Group{
		Name:          "Test",
		DailyLimitUSD: ptrFloat64(10.0),
	}

	progress := svc.calculateProgress(sub, group)

	require.NotNil(t, progress.Daily)
	assert.GreaterOrEqual(t, progress.Daily.ResetsInSeconds, int64(0),
		"ResetsInSeconds 不应为负数")
}
