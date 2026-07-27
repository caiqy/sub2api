package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

type blockingNotificationSettings struct {
	SettingRepository
	values      map[string]string
	smtpStarted chan struct{}
	releaseSMTP chan struct{}
}

func (r *blockingNotificationSettings) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	for _, key := range keys {
		if key != SettingKeySMTPHost {
			continue
		}
		select {
		case r.smtpStarted <- struct{}{}:
		default:
		}
		select {
		case <-r.releaseSMTP:
			return map[string]string{}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	values := make(map[string]string, len(keys))
	for _, key := range keys {
		values[key] = r.values[key]
	}
	return values, nil
}

func (r *blockingNotificationSettings) GetValue(_ context.Context, key string) (string, error) {
	return r.values[key], nil
}

func TestFinalizePostUsageBillingWaitsForNotificationDelivery(t *testing.T) {
	t.Run("balance", func(t *testing.T) {
		settings := &blockingNotificationSettings{
			values: map[string]string{
				SettingKeyBalanceLowNotifyEnabled:   "true",
				SettingKeyBalanceLowNotifyThreshold: "10",
				SettingKeySiteName:                  "test",
			},
			smtpStarted: make(chan struct{}, 1),
			releaseSMTP: make(chan struct{}),
		}
		p := &postUsageBillingParams{
			Cost:    &CostBreakdown{ActualCost: 15, TotalCost: 15},
			User:    &User{ID: 1, Balance: 20, BalanceNotifyEnabled: true, BalanceNotifyExtraEmails: []NotifyEmailEntry{{Email: "user@example.com", Verified: true}}},
			Account: &Account{ID: 1},
		}

		assertFinalizePostUsageBillingWaitsForNotification(t, settings, p, &UsageBillingApplyResult{})
	})

	t.Run("account quota", func(t *testing.T) {
		settings := &blockingNotificationSettings{
			values: map[string]string{
				SettingKeyAccountQuotaNotifyEnabled: "true",
				SettingKeyAccountQuotaNotifyEmails:  `[{"email":"ops@example.com","verified":true}]`,
				SettingKeySiteName:                  "test",
			},
			smtpStarted: make(chan struct{}, 1),
			releaseSMTP: make(chan struct{}),
		}
		p := &postUsageBillingParams{
			Cost:                  &CostBreakdown{TotalCost: 100},
			Account:               &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Extra: map[string]any{"quota_notify_daily_enabled": true, "quota_notify_daily_threshold": 100.0}},
			AccountRateMultiplier: 1,
		}
		result := &UsageBillingApplyResult{QuotaState: &AccountQuotaState{DailyUsed: 950, DailyLimit: 1000}}

		assertFinalizePostUsageBillingWaitsForNotification(t, settings, p, result)
	})
}

func assertFinalizePostUsageBillingWaitsForNotification(t *testing.T, settings *blockingNotificationSettings, p *postUsageBillingParams, result *UsageBillingApplyResult) {
	t.Helper()

	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(settings.releaseSMTP) }) }
	pool := NewUsageRecordWorkerPoolWithOptions(UsageRecordWorkerPoolOptions{
		WorkerCount:           1,
		QueueSize:             1,
		TaskTimeout:           time.Second,
		OverflowPolicy:        "drop",
		OverflowSamplePercent: 0,
		AutoScaleEnabled:      false,
	})
	t.Cleanup(func() {
		release()
		pool.Stop()
	})

	notify := NewBalanceNotifyService(NewEmailService(settings, nil), settings, nil)
	done := make(chan struct{})
	pool.Submit(func(ctx context.Context) {
		finalizePostUsageBilling(ctx, p, &billingDeps{
			deferredService:      &DeferredService{},
			balanceNotifyService: notify,
		}, result)
		close(done)
	})

	select {
	case <-settings.smtpStarted:
	case <-time.After(time.Second):
		t.Fatal("notification delivery did not start")
	}

	select {
	case <-done:
		t.Fatal("usage task returned before notification delivery completed")
	case <-time.After(50 * time.Millisecond):
	}

	release()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("usage task did not return after notification delivery completed")
	}
}
