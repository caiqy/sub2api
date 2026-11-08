package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestSelectAccountWithScheduler_QuotaPausePreservesStickyOnFallback(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	groupID := int64(93098)
	paused := newOpenAIImageSizeTestAccount(81992, 0)
	paused.Type = AccountTypeOAuth
	paused.GroupIDs = []int64{groupID}
	paused.Extra["auto_pause_5h_threshold"] = 0.9
	paused.Extra["codex_5h_used_percent"] = 95.0
	healthy := newOpenAIImageSizeTestAccount(81993, 1)
	healthy.GroupIDs = []int64{groupID}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:regression": paused.ID}}
	svc := newOpenAIImageSizeSchedulerTestService(openAIImageSizeSchedulerMode{advanced: true, loadBatch: true}, []Account{paused, healthy}, cache, schedulerTestConcurrencyCache{})
	selection, _, err := svc.SelectAccountWithScheduler(context.Background(), &groupID, "", "regression", "gpt-5.4", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	if selection.ReleaseFunc != nil {
		defer selection.ReleaseFunc()
	}
	require.Equal(t, healthy.ID, selection.Account.ID)
	require.Equal(t, paused.ID, cache.sessionBindings["openai:regression"], "temporary quota pause must not migrate the session")
}

// Exercise admission through the public entry, including stale snapshots and busy sticky slots.
func TestSelectAccountWithScheduler_StickyEligibilityRegression(t *testing.T) {
	for _, gate := range []string{"legacy_privacy", "advanced_quota"} {
		for _, state := range []string{"no_snapshot", "snapshot", "latest_db", "allowed", "privacy_not_required"} {
			if gate == "advanced_quota" && state == "privacy_not_required" {
				continue
			}
			for _, sticky := range []string{"none", "session", "previous_response"} {
				if gate == "legacy_privacy" && sticky == "previous_response" {
					continue
				}
				for _, busy := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/%s/busy=%v", gate, state, sticky, busy), func(t *testing.T) {
						resetOpenAIAdvancedSchedulerSettingCacheForTest()
						groupID := int64(93099)
						a := newOpenAIImageSizeTestAccount(81991, 0)
						a.GroupIDs = []int64{groupID}
						quota := gate == "advanced_quota"
						if quota {
							a.Type = AccountTypeOAuth
							a.Credentials = map[string]any{"access_token": "local-access", "refresh_token": "local-refresh", "account_id": "local-account"}
							a.Extra["auto_pause_5h_threshold"] = 0.9
							a.Extra["codex_5h_used_percent"] = 95.0
						}
						allowed := state == "allowed" || state == "privacy_not_required"
						if state == "allowed" {
							if quota {
								a.Extra["codex_5h_used_percent"] = 50.0
							} else {
								a.Extra["privacy_mode"] = "training_off"
							}
						}
						cached := a
						cached.Extra = make(map[string]any, len(a.Extra))
						for k, v := range a.Extra {
							cached.Extra[k] = v
						}
						if state == "latest_db" {
							if quota {
								cached.Extra["codex_5h_used_percent"] = 50.0
							} else {
								cached.Extra["privacy_mode"] = "training_off"
							}
						}
						cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:regression": a.ID}}
						acquired := []int64{}
						svc := newOpenAIImageSizeSchedulerTestService(openAIImageSizeSchedulerMode{advanced: quota, loadBatch: true}, []Account{a}, cache, schedulerTestConcurrencyCache{acquireResults: map[int64]bool{a.ID: !busy}, acquiredIDs: &acquired})
						setErrors := map[int64]string{}
						svc.accountRepo = schedulerTestOpenAIAccountRepo{accounts: []Account{a}, setErrors: setErrors}
						store := &openAIWSStateStoreSpy{responseAccounts: map[string]int64{"resp_regression": a.ID}}
						svc.openaiWSStateStore = store
						svc.cfg.Gateway.OpenAIWS.Enabled = true
						svc.cfg.Gateway.OpenAIWS.OAuthEnabled = true
						svc.cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
						a.Extra["openai_oauth_responses_websockets_v2_enabled"] = true
						cached.Extra["openai_oauth_responses_websockets_v2_enabled"] = true
						svc.cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600
						if state != "no_snapshot" {
							svc.schedulerSnapshot = &SchedulerSnapshotService{cache: &openAISnapshotCacheStub{snapshotAccounts: []*Account{&cached}, accountsByID: map[int64]*Account{a.ID: &cached}}}
						}
						ctx := context.WithValue(context.Background(), ctxkey.Group, &Group{ID: groupID, Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true, RequirePrivacySet: !quota && state != "privacy_not_required"})
						if quota {
							paused, _ := shouldAutoPauseOpenAIAccountByQuota(ctx, &a)
							require.Equal(t, !allowed, paused)
						}
						session, previous := "", ""
						if sticky == "session" {
							session = "regression"
						}
						if sticky == "previous_response" {
							previous = "resp_regression"
						}
						sel, decision, err := svc.SelectAccountWithScheduler(ctx, &groupID, previous, session, "gpt-5.4", nil, OpenAIUpstreamTransportAny, false)
						if !quota {
							require.Empty(t, setErrors, "group privacy rejection must not globally disable an account")
						}
						if sel != nil && sel.ReleaseFunc != nil {
							defer sel.ReleaseFunc()
						}
						if !allowed {
							require.Error(t, err)
							require.Nil(t, sel, "ineligible account must never acquire or produce a WaitPlan")
							if state != "latest_db" {
								require.Empty(t, acquired, "known ineligible accounts must not acquire a slot")
							}
							if quota && sticky == "session" {
								require.Equal(t, a.ID, cache.sessionBindings["openai:regression"], "transient quota pause must preserve binding")
							}
							return
						}
						require.NoError(t, err)
						require.NotNil(t, sel)
						require.Equal(t, a.ID, sel.Account.ID)
						require.Equal(t, !busy, sel.Acquired)
						if busy {
							require.NotNil(t, sel.WaitPlan)
							require.Equal(t, a.ID, sel.WaitPlan.AccountID)
						}
						if sticky == "session" {
							require.Equal(t, a.ID, cache.sessionBindings["openai:regression"])
							require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
						}
						if sticky == "previous_response" {
							require.Equal(t, openAIAccountScheduleLayerPreviousResponse, decision.Layer)
						}
					})
				}
			}
		}
	}
}
