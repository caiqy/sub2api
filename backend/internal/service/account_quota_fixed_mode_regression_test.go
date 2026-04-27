package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsDailyQuotaPeriodExpired_FixedModePrefersFutureResetAt(t *testing.T) {
	now := time.Now().UTC()
	a := &Account{Extra: map[string]any{
		"quota_daily_reset_mode": "fixed",
		"quota_daily_reset_hour": float64(0),
		"quota_reset_timezone":   "Asia/Shanghai",
		"quota_daily_start":      now.Add(-72 * time.Hour).Format(time.RFC3339),
		"quota_daily_reset_at":   now.Add(10 * time.Hour).Format(time.RFC3339),
	}}

	require.False(t, a.IsDailyQuotaPeriodExpired())
}

func TestIsQuotaExceeded_FixedDailyQuotaUsesResetAtWindow(t *testing.T) {
	now := time.Now().UTC()
	a := &Account{Extra: map[string]any{
		"quota_daily_limit":      77.0,
		"quota_daily_used":       80.0,
		"quota_daily_reset_mode": "fixed",
		"quota_daily_reset_hour": float64(0),
		"quota_reset_timezone":   "Asia/Shanghai",
		"quota_daily_start":      now.Add(-72 * time.Hour).Format(time.RFC3339),
		"quota_daily_reset_at":   now.Add(10 * time.Hour).Format(time.RFC3339),
	}}

	require.True(t, a.IsQuotaExceeded())
}

func TestIsWeeklyQuotaPeriodExpired_FixedModePrefersFutureResetAt(t *testing.T) {
	now := time.Now().UTC()
	a := &Account{Extra: map[string]any{
		"quota_weekly_reset_mode": "fixed",
		"quota_weekly_reset_day":  float64(1),
		"quota_weekly_reset_hour": float64(0),
		"quota_reset_timezone":    "Asia/Shanghai",
		"quota_weekly_start":      now.Add(-21 * 24 * time.Hour).Format(time.RFC3339),
		"quota_weekly_reset_at":   now.Add(10 * time.Hour).Format(time.RFC3339),
	}}

	require.False(t, a.IsWeeklyQuotaPeriodExpired())
}

func TestIsQuotaExceeded_FixedWeeklyQuotaUsesResetAtWindow(t *testing.T) {
	now := time.Now().UTC()
	a := &Account{Extra: map[string]any{
		"quota_weekly_limit":      120.0,
		"quota_weekly_used":       121.0,
		"quota_weekly_reset_mode": "fixed",
		"quota_weekly_reset_day":  float64(1),
		"quota_weekly_reset_hour": float64(0),
		"quota_reset_timezone":    "Asia/Shanghai",
		"quota_weekly_start":      now.Add(-21 * 24 * time.Hour).Format(time.RFC3339),
		"quota_weekly_reset_at":   now.Add(10 * time.Hour).Format(time.RFC3339),
	}}

	require.True(t, a.IsQuotaExceeded())
}
