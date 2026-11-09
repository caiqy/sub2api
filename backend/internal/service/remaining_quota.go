package service

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RemainingQuotaSummary uses the same post-multiplier USD as wallet and subscription billing.
// Total excludes unlimited subscriptions; callers must display their count separately.
type RemainingQuotaSummary struct {
	ExcludedUserIDs        []int64 `json:"excluded_user_ids"`
	Balance                float64 `json:"balance"`
	Subscription           float64 `json:"subscription"`
	Total                  float64 `json:"total"`
	UnlimitedSubscriptions int     `json:"unlimited_subscriptions"`
}

type remainingQuotaReader interface {
	ReadRemainingQuota(ctx context.Context, excluded []int64, now time.Time) (float64, []UserSubscription, error)
}

func (s *adminServiceImpl) GetRemainingQuota(ctx context.Context, excluded []int64) (*RemainingQuotaSummary, error) {
	reader, ok := s.userRepo.(remainingQuotaReader)
	if !ok {
		return nil, fmt.Errorf("remaining quota reader unavailable")
	}
	now := time.Now()
	balance, subs, err := reader.ReadRemainingQuota(ctx, excluded, now)
	if err != nil {
		return nil, fmt.Errorf("read remaining quota: %w", err)
	}
	result := &RemainingQuotaSummary{Balance: balance}
	for i := range subs {
		amount, unlimited := remainingSubscriptionQuota(&subs[i], now)
		if unlimited {
			result.UnlimitedSubscriptions++
		} else {
			result.Subscription += amount
		}
	}
	result.Total = result.Balance + result.Subscription
	return result, nil
}

// Consume as early as possible, then advance to the next actual reset boundary.
// Taking the minimum of whole-term totals would overcount misaligned windows.
// Advancing quota early spends the remaining window duration from ExpiresAt:
// it brings capacity forward, rather than granting additional term capacity.
func remainingSubscriptionQuota(sub *UserSubscription, now time.Time) (float64, bool) {
	if sub == nil || sub.DeletedAt != nil || sub.Status != SubscriptionStatusActive || !sub.ExpiresAt.After(now) || sub.Group == nil {
		return 0, false
	}
	type window struct {
		limit, remaining float64
		next             time.Time
		period           time.Duration
	}
	windows := make([]window, 0, 3)
	add := func(limit *float64, usage float64, start *time.Time, period time.Duration, oneTime bool) {
		if limit == nil || *limit <= 0 {
			return
		}
		anchor := sub.StartsAt
		if anchor.IsZero() {
			anchor = now
		}
		if start != nil {
			anchor = *start
		}
		anchor = sub.windowResetAnchor(anchor)
		if !oneTime {
			if current, ok := sub.automaticWindowStartAt(&anchor, period, now); ok {
				anchor, usage = current, 0
			}
		}
		next := sub.ExpiresAt
		if !oneTime {
			next = anchor.Add(period)
		}
		windows = append(windows, window{*limit, math.Max(0, *limit-usage), next, period})
	}
	add(sub.Group.DailyLimitUSD, sub.DailyUsageUSD, sub.DailyWindowStart, 24*time.Hour, sub.HasOneTimeDailyQuota())
	add(sub.Group.WeeklyLimitUSD, sub.WeeklyUsageUSD, sub.WeeklyWindowStart, 7*24*time.Hour, false)
	add(sub.Group.MonthlyLimitUSD, sub.MonthlyUsageUSD, sub.MonthlyWindowStart, 30*24*time.Hour, false)
	if len(windows) == 0 {
		return 0, true
	}
	total := 0.0
	// ponytail: linear in reset boundaries; batch whole repeated cycles if very long terms become common.
	for {
		amount := math.Inf(1)
		next := sub.ExpiresAt
		for _, w := range windows {
			amount = math.Min(amount, w.remaining)
			if w.next.Before(next) {
				next = w.next
			}
		}
		total += amount
		if !next.Before(sub.ExpiresAt) {
			return total, false
		}
		for i := range windows {
			w := &windows[i]
			w.remaining -= amount
			if w.next.Equal(next) {
				w.remaining = w.limit
				w.next = w.next.Add(w.period)
			}
		}
	}
}
