package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestOpenAISchedulingCodexTicketCompact(t *testing.T) {
	for _, path := range []string{"legacy_simple", "legacy_batch", "legacy_load_error", "legacy_wait", "advanced_acquire", "advanced_wait", "layered_acquire", "layered_wait"} {
		for _, name := range []string{"ordinary", "compact", "ordinary_snapshot", "compact_snapshot"} {
			compact := name == "compact" || name == "compact_snapshot"
			t.Run(path+"/"+name, func(t *testing.T) {
				account := ticketTestAccount(41)
				account.Status = StatusActive
				account.Schedulable = true
				account.Concurrency = 1
				groupID := int64(91)
				account.GroupIDs = []int64{groupID}
				svc := ticketTestService(t, config.OpenAICodexTicketConfig{
					Enabled: true, FailClosed: true, Models: []string{"gpt-6-astra"},
				}, nil)
				svc.cfg.Gateway.OpenAICompactModel = "gpt-5.5"
				svc.cfg.Gateway.Scheduling.LoadBatchEnabled = path != "legacy_simple"
				svc.accountRepo = schedulerTestOpenAIAccountRepo{accounts: []Account{*account}}
				if name == "ordinary_snapshot" || name == "compact_snapshot" {
					snapshot := *account
					// The deferred capability check must allow DB recovery from a stale tier 0.
					if path != "layered_acquire" && path != "layered_wait" {
						snapshot.Extra = map[string]any{openAICodexTicketEnabledExtraKey: true, "openai_compact_supported": false}
					}
					svc.schedulerSnapshot = &SchedulerSnapshotService{
						cache:       &openAISnapshotCacheStub{snapshotAccounts: []*Account{&snapshot}, accountsByID: map[int64]*Account{account.ID: &snapshot}},
						accountRepo: svc.accountRepo, cfg: svc.cfg,
					}
				}
				wait := path == "legacy_wait" || path == "advanced_wait" || path == "layered_wait"
				cache := schedulerTestConcurrencyCache{acquireResults: map[int64]bool{account.ID: !wait}}
				if path == "legacy_load_error" {
					cache.loadBatchErr = errors.New("load unavailable")
				}
				svc.concurrencyService = NewConcurrencyService(cache)
				req := OpenAIAccountScheduleRequest{
					GroupID: &groupID, Platform: PlatformOpenAI, RequestedModel: "gpt-6-astra",
					RequireCompact: compact, RequiredTransport: OpenAIUpstreamTransportAny,
					RequiredCapability: OpenAIEndpointCapabilityResponses,
				}
				var selection *AccountSelectionResult
				var err error
				switch path {
				case "advanced_acquire", "advanced_wait":
					scheduler := newDefaultOpenAIAccountScheduler(svc, nil)
					selection, _, err = scheduler.Select(context.Background(), req)
				case "layered_acquire", "layered_wait":
					scheduler := newLayeredOpenAIAccountScheduler(svc, nil)
					t.Cleanup(scheduler.Stop)
					selection, _, err = scheduler.Select(context.Background(), req)
				default:
					selection, err = svc.selectAccountWithLoadAwareness(context.Background(), &groupID, PlatformOpenAI, "", req.RequestedModel, nil, compact, req.RequiredCapability, false)
				}
				if !compact {
					require.Error(t, err)
					require.Nil(t, selection, "ordinary managed requests without a ticket must remain blocked")
					return
				}
				require.NoError(t, err)
				require.NotNil(t, selection)
				require.Equal(t, account.ID, selection.Account.ID)
				require.Equal(t, !wait, selection.Acquired)
				if selection.ReleaseFunc != nil {
					selection.ReleaseFunc()
				}
				if wait {
					require.NotNil(t, selection.WaitPlan)
				}
			})
		}
	}
}
