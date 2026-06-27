package service

import "time"

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
	if s.IsExpired() {
		return 0
	}
	return int(time.Until(s.ExpiresAt).Hours() / 24)
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
	return fixLegacyMidnightAnchor(s.StartsAt, s.DailyWindowStart, 24*time.Hour)
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
	return fixLegacyMidnightAnchor(s.StartsAt, s.WeeklyWindowStart, 7*24*time.Hour)
}

func (s *UserSubscription) effectiveMonthlyWindowStart() *time.Time {
	return fixLegacyMidnightAnchor(s.StartsAt, s.MonthlyWindowStart, 30*24*time.Hour)
}

func isMidnight(t time.Time) bool {
	return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
}

func fixLegacyMidnightAnchor(startsAt time.Time, windowStart *time.Time, period time.Duration) *time.Time {
	if windowStart == nil || startsAt.IsZero() {
		return windowStart
	}
	if !isMidnight(*windowStart) {
		return windowStart
	}
	// StartsAt=00:00 是合法锚点，不纠偏。
	if isMidnight(startsAt) {
		return windowStart
	}
	since := windowStart.Sub(startsAt)
	if since < 0 {
		return &startsAt
	}
	if since < period {
		return &startsAt
	}
	periods := int(since / period)
	aligned := startsAt.Add(time.Duration(periods) * period)
	if aligned.Equal(*windowStart) {
		return windowStart
	}
	return &aligned
}

func (s *UserSubscription) NeedsWeeklyResetAt(now time.Time) bool {
	ws := s.effectiveWeeklyWindowStart()
	if ws == nil {
		return false
	}
	return !now.Before(ws.Add(7 * 24 * time.Hour))
}

func (s *UserSubscription) NeedsMonthlyReset() bool {
	return s.NeedsMonthlyResetAt(time.Now())
}

func (s *UserSubscription) NeedsMonthlyResetAt(now time.Time) bool {
	ws := s.effectiveMonthlyWindowStart()
	if ws == nil {
		return false
	}
	return !now.Before(ws.Add(30 * 24 * time.Hour))
}

func (s *UserSubscription) NextWeeklyWindowStart(now time.Time) *time.Time {
	ws := s.effectiveWeeklyWindowStart()
	if ws == nil || now.Before(ws.Add(7*24*time.Hour)) {
		return nil
	}
	periods := now.Sub(*ws) / (7 * 24 * time.Hour)
	t := ws.Add(periods * 7 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) NextMonthlyWindowStart(now time.Time) *time.Time {
	ws := s.effectiveMonthlyWindowStart()
	if ws == nil || now.Before(ws.Add(30*24*time.Hour)) {
		return nil
	}
	periods := now.Sub(*ws) / (30 * 24 * time.Hour)
	t := ws.Add(periods * 30 * 24 * time.Hour)
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
	t := ws.Add(7 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) MonthlyResetTime() *time.Time {
	ws := s.effectiveMonthlyWindowStart()
	if ws == nil {
		return nil
	}
	t := ws.Add(30 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) FutureWeeklyResetTime(now time.Time) *time.Time {
	ws := s.effectiveWeeklyWindowStart()
	if ws == nil || now.Before(*ws) {
		return nil
	}
	periods := now.Sub(*ws) / (7 * 24 * time.Hour)
	t := ws.Add((periods + 1) * 7 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) FutureMonthlyResetTime(now time.Time) *time.Time {
	ws := s.effectiveMonthlyWindowStart()
	if ws == nil || now.Before(*ws) {
		return nil
	}
	periods := now.Sub(*ws) / (30 * 24 * time.Hour)
	t := ws.Add((periods + 1) * 30 * 24 * time.Hour)
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
	if now.Before(ws.Add(7 * 24 * time.Hour)) {
		return ws
	}
	periods := now.Sub(*ws) / (7 * 24 * time.Hour)
	t := ws.Add(periods * 7 * 24 * time.Hour)
	return &t
}

// CurrentMonthlyWindowStart returns the latest aligned window start boundary <= now.
func (s *UserSubscription) CurrentMonthlyWindowStart(now time.Time) *time.Time {
	ws := s.effectiveMonthlyWindowStart()
	if ws == nil {
		return nil
	}
	if now.Before(ws.Add(30 * 24 * time.Hour)) {
		return ws
	}
	periods := now.Sub(*ws) / (30 * 24 * time.Hour)
	t := ws.Add(periods * 30 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) CheckDailyLimit(group *Group, additionalCost float64) bool {
	if !group.HasDailyLimit() {
		return true
	}
	// 使用 < 而非 <=，与缓存路径 (billing_cache_service) 的 >= 判定语义保持一致：
	// usage >= limit → 拒绝，usage < limit → 放行
	return s.DailyUsageUSD+additionalCost < *group.DailyLimitUSD
}

func (s *UserSubscription) CheckWeeklyLimit(group *Group, additionalCost float64) bool {
	if !group.HasWeeklyLimit() {
		return true
	}
	return s.WeeklyUsageUSD+additionalCost < *group.WeeklyLimitUSD
}

func (s *UserSubscription) CheckMonthlyLimit(group *Group, additionalCost float64) bool {
	if !group.HasMonthlyLimit() {
		return true
	}
	return s.MonthlyUsageUSD+additionalCost < *group.MonthlyLimitUSD
}

func (s *UserSubscription) CheckAllLimits(group *Group, additionalCost float64) (daily, weekly, monthly bool) {
	daily = s.CheckDailyLimit(group, additionalCost)
	weekly = s.CheckWeeklyLimit(group, additionalCost)
	monthly = s.CheckMonthlyLimit(group, additionalCost)
	return
}
