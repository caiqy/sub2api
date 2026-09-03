package service

import "time"

const subscriptionDayDuration = 24 * time.Hour

type UserSubscription struct {
	ID      int64
	UserID  int64
	GroupID int64

	StartsAt  time.Time
	ExpiresAt time.Time
	Status    string

	DailyWindowStart   *time.Time
	WeeklyWindowStart  *time.Time
	MonthlyWindowStart *time.Time

	DailyUsageUSD   float64
	WeeklyUsageUSD  float64
	MonthlyUsageUSD float64

	AssignedBy *int64
	AssignedAt time.Time
	Notes      string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	User           *User
	Group          *Group
	AssignedByUser *User
}

func (s *UserSubscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive && time.Now().Before(s.ExpiresAt)
}

func (s *UserSubscription) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

func (s *UserSubscription) DaysRemaining() int {
	return s.daysRemainingAt(time.Now())
}

func (s *UserSubscription) daysRemainingAt(now time.Time) int {
	remaining := s.ExpiresAt.Sub(now)
	if remaining <= 0 {
		return 0
	}

	days := int(remaining / subscriptionDayDuration)
	if remaining%subscriptionDayDuration != 0 {
		days++
	}
	return days
}

func (s *UserSubscription) IsWindowActivated() bool {
	return s.DailyWindowStart != nil || s.WeeklyWindowStart != nil || s.MonthlyWindowStart != nil
}

func (s *UserSubscription) HasOneTimeDailyQuota() bool {
	if s == nil || s.StartsAt.IsZero() || s.ExpiresAt.IsZero() {
		return false
	}
	return !s.ExpiresAt.After(s.StartsAt.AddDate(0, 0, 1))
}

func (s *UserSubscription) NeedsDailyReset() bool {
	return s.NeedsDailyResetAt(time.Now())
}

func (s *UserSubscription) effectiveDailyWindowStart() *time.Time {
	return s.DailyWindowStart
}

func (s *UserSubscription) NeedsDailyResetAt(now time.Time) bool {
	ws := s.effectiveDailyWindowStart()
	if ws == nil {
		return false
	}
	if s.HasOneTimeDailyQuota() {
		return false
	}
	return !now.Before(ws.Add(24 * time.Hour))
}

func (s *UserSubscription) NeedsWeeklyReset() bool {
	return s.NeedsWeeklyResetAt(time.Now())
}

func (s *UserSubscription) effectiveWeeklyWindowStart() *time.Time {
	return s.WeeklyWindowStart
}

func (s *UserSubscription) effectiveMonthlyWindowStart() *time.Time {
	return s.MonthlyWindowStart
}

func (s *UserSubscription) NeedsWeeklyResetAt(now time.Time) bool {
	ws := s.effectiveWeeklyWindowStart()
	if ws == nil {
		return false
	}
	return !now.Before(s.windowResetAnchor(*ws).Add(7 * 24 * time.Hour))
}

func (s *UserSubscription) NeedsMonthlyReset() bool {
	return s.NeedsMonthlyResetAt(time.Now())
}

func (s *UserSubscription) canAutomaticallyResetDailyAt(now time.Time) bool {
	_, ok := s.automaticWindowStartAt(s.DailyWindowStart, 24*time.Hour, now)
	return !s.HasOneTimeDailyQuota() && ok
}

func (s *UserSubscription) canAutomaticallyResetWeeklyAt(now time.Time) bool {
	_, ok := s.automaticWindowStartAt(s.WeeklyWindowStart, 7*24*time.Hour, now)
	return ok
}

func (s *UserSubscription) canAutomaticallyResetMonthlyAt(now time.Time) bool {
	_, ok := s.automaticWindowStartAt(s.MonthlyWindowStart, 30*24*time.Hour, now)
	return ok
}

// windowResetAnchor 返回周/月窗口实际推进所依据的锚点。
// 早期订阅把首个窗口初始化在开通日零点；只有这个初始值是无歧义的，之后出现的
// 零点锚点可能来自手动重置，必须保持权威。
// 自动推进（automaticWindowStartAt）与对外展示的重置时间（WeeklyResetTime/
// MonthlyResetTime）必须共用这一修正，否则仪表盘显示的重置时间会早于窗口实际
// 滚动的时间。
// 日窗口按日历日对齐（automaticDailyWindowStartAt），不走这里。
func (s *UserSubscription) windowResetAnchor(previous time.Time) time.Time {
	legacyAnchor := startOfDay(s.StartsAt)
	if legacyAnchor.Before(s.StartsAt) && previous.Equal(legacyAnchor) {
		return s.StartsAt
	}
	return previous
}

// automaticWindowStartAt 计算周/月窗口（期限对齐滚动窗口）的当前窗口起点。
// 窗口从锚点按整数个 period 步进，且不越过订阅到期时间，避免最后一个不完整
// 周期重复发放额度（issue #5051）。日窗口不走此函数，见 automaticDailyWindowStartAt。
func (s *UserSubscription) automaticWindowStartAt(previous *time.Time, period time.Duration, now time.Time) (time.Time, bool) {
	if previous == nil {
		return time.Time{}, false
	}

	anchor := s.windowResetAnchor(*previous)
	next := anchor.Add(period)
	if now.Before(next) || !next.Before(s.ExpiresAt) {
		return time.Time{}, false
	}

	periods := now.Sub(anchor) / period
	lastPeriodBeforeExpiry := (s.ExpiresAt.Sub(anchor) - 1) / period
	if periods > lastPeriodBeforeExpiry {
		periods = lastPeriodBeforeExpiry
	}
	return anchor.Add(periods * period), true
}

func (s *UserSubscription) NeedsMonthlyResetAt(now time.Time) bool {
	ws := s.effectiveMonthlyWindowStart()
	if ws == nil {
		return false
	}
	return !now.Before(s.windowResetAnchor(*ws).Add(30 * 24 * time.Hour))
}

func (s *UserSubscription) NextWeeklyWindowStart(now time.Time) *time.Time {
	ws := s.effectiveWeeklyWindowStart()
	if ws == nil {
		return nil
	}
	anchor := s.windowResetAnchor(*ws)
	if now.Before(anchor.Add(7 * 24 * time.Hour)) {
		return nil
	}
	periods := now.Sub(anchor) / (7 * 24 * time.Hour)
	t := anchor.Add(periods * 7 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) NextMonthlyWindowStart(now time.Time) *time.Time {
	ws := s.effectiveMonthlyWindowStart()
	if ws == nil {
		return nil
	}
	anchor := s.windowResetAnchor(*ws)
	if now.Before(anchor.Add(30 * 24 * time.Hour)) {
		return nil
	}
	periods := now.Sub(anchor) / (30 * 24 * time.Hour)
	t := anchor.Add(periods * 30 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) NextDailyWindowStart(now time.Time) *time.Time {
	ws := s.effectiveDailyWindowStart()
	if ws == nil {
		return nil
	}
	if s.HasOneTimeDailyQuota() {
		return nil
	}
	if now.Before(ws.Add(24 * time.Hour)) {
		return nil
	}
	periods := int(now.Sub(*ws) / (24 * time.Hour))
	t := ws.Add(time.Duration(periods) * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) CurrentDailyWindowStart(now time.Time) *time.Time {
	ws := s.effectiveDailyWindowStart()
	if ws == nil {
		return nil
	}
	if s.HasOneTimeDailyQuota() || now.Before(ws.Add(24*time.Hour)) {
		return ws
	}
	periods := int(now.Sub(*ws) / (24 * time.Hour))
	t := ws.Add(time.Duration(periods) * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) FutureDailyResetTime(now time.Time) *time.Time {
	if s.DailyWindowStart == nil {
		return nil
	}
	if s.HasOneTimeDailyQuota() {
		t := s.ExpiresAt
		return &t
	}
	current := s.CurrentDailyWindowStart(now)
	if current == nil {
		return nil
	}
	t := current.Add(24 * time.Hour)
	return &t
}

func (s *UserSubscription) DailyResetTime() *time.Time {
	ws := s.effectiveDailyWindowStart()
	if ws == nil {
		return nil
	}
	if s.HasOneTimeDailyQuota() {
		t := s.ExpiresAt
		return &t
	}
	t := ws.Add(24 * time.Hour)
	return &t
}

func (s *UserSubscription) WeeklyResetTime() *time.Time {
	ws := s.effectiveWeeklyWindowStart()
	if ws == nil {
		return nil
	}
	t := s.windowResetAnchor(*s.WeeklyWindowStart).Add(7 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) MonthlyResetTime() *time.Time {
	ws := s.effectiveMonthlyWindowStart()
	if ws == nil {
		return nil
	}
	t := s.windowResetAnchor(*s.MonthlyWindowStart).Add(30 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) FutureWeeklyResetTime(now time.Time) *time.Time {
	ws := s.effectiveWeeklyWindowStart()
	if ws == nil {
		return nil
	}
	anchor := s.windowResetAnchor(*ws)
	if now.Before(anchor) {
		return nil
	}
	periods := now.Sub(anchor) / (7 * 24 * time.Hour)
	t := anchor.Add((periods + 1) * 7 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) FutureMonthlyResetTime(now time.Time) *time.Time {
	ws := s.effectiveMonthlyWindowStart()
	if ws == nil {
		return nil
	}
	anchor := s.windowResetAnchor(*ws)
	if now.Before(anchor) {
		return nil
	}
	periods := now.Sub(anchor) / (30 * 24 * time.Hour)
	t := anchor.Add((periods + 1) * 30 * 24 * time.Hour)
	return &t
}

// CurrentWeeklyWindowStart returns the latest aligned window start boundary <= now.
// For a fresh window (now within first period), returns the effective anchor.
// For a stale window (now past one or more boundaries), returns the latest boundary.
func (s *UserSubscription) CurrentWeeklyWindowStart(now time.Time) *time.Time {
	ws := s.effectiveWeeklyWindowStart()
	if ws == nil {
		return nil
	}
	anchor := s.windowResetAnchor(*ws)
	if now.Before(anchor.Add(7 * 24 * time.Hour)) {
		return &anchor
	}
	periods := now.Sub(anchor) / (7 * 24 * time.Hour)
	t := anchor.Add(periods * 7 * 24 * time.Hour)
	return &t
}

// CurrentMonthlyWindowStart returns the latest aligned window start boundary <= now.
func (s *UserSubscription) CurrentMonthlyWindowStart(now time.Time) *time.Time {
	ws := s.effectiveMonthlyWindowStart()
	if ws == nil {
		return nil
	}
	anchor := s.windowResetAnchor(*ws)
	if now.Before(anchor.Add(30 * 24 * time.Hour)) {
		return &anchor
	}
	periods := now.Sub(anchor) / (30 * 24 * time.Hour)
	t := anchor.Add(periods * 30 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) CheckDailyLimit(group *Group, additionalCost float64) bool {
	if !group.HasDailyLimit() {
		return true
	}
	// Inclusive limit: a request that exactly reaches the configured limit is allowed.
	return s.DailyUsageUSD+additionalCost <= *group.DailyLimitUSD
}

func (s *UserSubscription) CheckWeeklyLimit(group *Group, additionalCost float64) bool {
	if !group.HasWeeklyLimit() {
		return true
	}
	return s.WeeklyUsageUSD+additionalCost <= *group.WeeklyLimitUSD
}

func (s *UserSubscription) CheckMonthlyLimit(group *Group, additionalCost float64) bool {
	if !group.HasMonthlyLimit() {
		return true
	}
	return s.MonthlyUsageUSD+additionalCost <= *group.MonthlyLimitUSD
}

func (s *UserSubscription) CheckAllLimits(group *Group, additionalCost float64) (daily, weekly, monthly bool) {
	daily = s.CheckDailyLimit(group, additionalCost)
	weekly = s.CheckWeeklyLimit(group, additionalCost)
	monthly = s.CheckMonthlyLimit(group, additionalCost)
	return
}
