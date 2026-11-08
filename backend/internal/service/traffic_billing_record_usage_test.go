//go:build unit

package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTrafficBillingSimpleModeBothPipelines(t *testing.T) {
	for _, openai := range []bool{false, true} {
		for _, windows := range []bool{false, true} {
			n := int64(12_000_000)
			ctx := WithTrafficBillingSnapshot(context.Background(), TrafficBillingSnapshot{Enabled: true, Consent: true, Successful: true, InputBytes: &n, ThresholdBytes: 10_000_000, PricePerMB: 0.01})
			logs := &openAIRecordUsageLogRepoStub{inserted: true}
			bills := &openAIRecordUsageBillingRepoStub{}
			users := &openAIRecordUsageUserRepoStub{}
			subs := &openAIRecordUsageSubRepoStub{}
			key := &APIKey{ID: 2, Quota: 100, RateLimit5h: 30, Group: &Group{RateMultiplier: 0}}
			user := &User{ID: 1}
			account := &Account{ID: 3, Type: AccountTypeAPIKey}
			var err error
			if openai {
				svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(t, logs, bills, users, subs, nil)
				svc.cfg.RunMode = config.RunModeSimple
				svc.cfg.SimpleModeKeyRateLimitEnabled = windows
				err = svc.RecordUsage(ctx, &OpenAIRecordUsageInput{Result: &OpenAIForwardResult{RequestID: "simple-traffic", Model: "gpt-5.1"}, APIKey: key, User: user, Account: account})
			} else {
				svc := newGatewayRecordUsageServiceWithBillingRepoForTest(logs, bills, users, subs)
				svc.cfg.RunMode = config.RunModeSimple
				svc.cfg.SimpleModeKeyRateLimitEnabled = windows
				err = svc.RecordUsage(ctx, &RecordUsageInput{Result: &ForwardResult{RequestID: "simple-traffic", Model: "claude-sonnet-4"}, APIKey: key, User: user, Account: account})
			}
			require.NoError(t, err)
			require.InDelta(t, 0.24, logs.lastLog.ActualCost, 1e-12)
			require.Zero(t, users.deductCalls)
			require.Zero(t, subs.incrementCalls)
			if windows {
				require.Equal(t, 1, bills.calls)
				require.InDelta(t, 0.24, bills.lastCmd.APIKeyRateLimitCost, 1e-12)
				require.Zero(t, bills.lastCmd.BalanceCost)
				require.Zero(t, bills.lastCmd.SubscriptionCost)
				require.Zero(t, bills.lastCmd.APIKeyQuotaCost)
				require.Zero(t, bills.lastCmd.AccountQuotaCost)
			} else {
				require.Zero(t, bills.calls)
			}
		}
	}
}

func TestTrafficBillingBothUsagePipelines(t *testing.T) {
	for _, pipeline := range []string{"gateway", "openai"} {
		for _, disconnected := range []bool{false, true} {
			t.Run(pipeline+map[bool]string{false: "/success", true: "/disconnect"}[disconnected], func(t *testing.T) {
				n := int64(12_000_000)
				ctx := WithTrafficBillingSnapshot(context.Background(), TrafficBillingSnapshot{Enabled: true, Consent: true, Successful: true, InputBytes: &n, ThresholdBytes: 10_000_000, PricePerMB: 0.01})
				logs := &openAIRecordUsageLogRepoStub{inserted: true}
				bills := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
				userRepo := &openAIRecordUsageUserRepoStub{}
				subRepo := &openAIRecordUsageSubRepoStub{}
				key := &APIKey{ID: 2, Quota: 100, Group: &Group{SubscriptionType: SubscriptionTypeSubscription, RateMultiplier: 0}}
				user := &User{ID: 1}
				account := &Account{ID: 3, Type: AccountTypeAPIKey}
				sub := &UserSubscription{ID: 4}
				var err error
				if pipeline == "gateway" {
					svc := newGatewayRecordUsageServiceWithBillingRepoForTest(logs, bills, userRepo, subRepo)
					err = svc.RecordUsage(ctx, &RecordUsageInput{Result: &ForwardResult{RequestID: "traffic", Model: "claude-sonnet-4", ClientDisconnect: disconnected}, APIKey: key, User: user, Account: account, Subscription: sub})
				} else {
					svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(t, logs, bills, userRepo, subRepo, nil)
					err = svc.RecordUsage(ctx, &OpenAIRecordUsageInput{Result: &OpenAIForwardResult{RequestID: "traffic", Model: "gpt-5.1", ClientDisconnect: disconnected}, APIKey: key, User: user, Account: account, Subscription: sub})
				}
				require.NoError(t, err)
				require.NotNil(t, logs.lastLog)
				require.NotNil(t, bills.lastCmd)
				want := 0.24
				if disconnected {
					want = 0
				}
				require.InDelta(t, want, logs.lastLog.TrafficCost, 1e-10)
				require.InDelta(t, want, logs.lastLog.TotalCost, 1e-10)
				require.InDelta(t, want, bills.lastCmd.SubscriptionCost, 1e-10)
				require.Zero(t, bills.lastCmd.BalanceCost)
				require.Zero(t, bills.lastCmd.AccountQuotaCost)
			})
		}
	}
}
