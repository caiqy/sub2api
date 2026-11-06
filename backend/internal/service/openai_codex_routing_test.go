package service

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type routingPolicyDetector struct {
	policy CodexRestrictionPolicy
	cfg    *config.Config
}

func (d routingPolicyDetector) Detect(c *gin.Context, a *Account, _ CodexRestrictionPolicy, b []byte) CodexClientRestrictionDetectionResult {
	return NewOpenAICodexClientRestrictionDetector(d.cfg).Detect(c, a, d.policy, b)
}

func TestCodexRoutingConfiguration(t *testing.T) {
	for _, platform := range []string{PlatformOpenAI, PlatformAnthropic} {
		for _, typ := range []string{AccountTypeOAuth, AccountTypeAPIKey} {
			for _, parent := range []any{nil, false, true, "true"} {
				for _, child := range []any{nil, false, true, "true", 1} {
					a := &Account{Platform: platform, Type: typ, Extra: map[string]any{"codex_cli_only": parent, "codex_cli_only_route_only": child}}
					require.Equal(t, platform == PlatformOpenAI && typ == AccountTypeOAuth && parent == true && child == true, a.IsCodexCLIOnlyRouteOnlyEnabled())
				}
			}
		}
	}
}

func TestCodexRoutingInboundDetectorParity(t *testing.T) {
	for _, ua := range []string{"third-party", "codex_cli_rs/0.100.0", "codex_cli_rs/unknown"} {
		for _, app := range []bool{false, true} {
			for _, body := range []string{`{}`, `{"client_metadata":{"engine":true}}`} {
				for _, policy := range []CodexRestrictionPolicy{
					{},
					{MinCodexVersion: "0.101.0"},
					{MaxCodexVersion: "0.99.0"},
					{AllowAppServerClients: true},
					{Whitelist: []openai.AllowedClientEntry{{Originator: "third-party", UAContains: []string{ua}, SkipEngineFingerprint: true}}},
					{Blacklist: []openai.AllowedClientEntry{{UAContains: []string{ua}}}},
					{EngineFingerprintSignals: []openai.EngineFingerprintSignal{{Type: openai.FingerprintSignalBodyPath, Match: []string{"client_metadata.engine"}, Required: true}}, AllowAppServerClients: true},
				} {
					c, _ := gin.CreateTestContext(httptest.NewRecorder())
					c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
					c.Request.Header.Set("User-Agent", ua)
					c.Request.Header.Set("originator", "third-party")
					a := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: map[string]any{"codex_cli_only": true, "codex_cli_only_route_only": true, "codex_cli_only_allow_app_server": app}}
					d := routingPolicyDetector{policy: policy}
					s := &OpenAIGatewayService{codexDetector: d}
					expected := d.Detect(c, a, policy, []byte(body))
					s.PrepareCodexClientRouting(c, []byte(body))
					c.Request.Header.Set("User-Agent", "rewritten")
					require.Equal(t, expected.Matched, codexClientRouteAllowed(c.Request.Context(), a))
					require.Equal(t, expected, s.detectCodexClientRestriction(c, a, nil))
					child, cancel := context.WithCancel(context.Background())
					copied := ContextWithCodexClientRouting(child, c.Request.Context())
					cancel()
					require.Error(t, copied.Err())
				}
			}
		}
	}
}

func TestCodexRoutingForceCLI(t *testing.T) {
	cfg := &config.Config{}
	cfg.Gateway.ForceCodexCLI = true
	s := &OpenAIGatewayService{cfg: cfg}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	s.PrepareCodexClientRouting(c, nil)
	a := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: map[string]any{"codex_cli_only": true, "codex_cli_only_route_only": true}}
	require.True(t, codexClientRouteAllowed(c.Request.Context(), a))
}

func TestCodexRoutingPreviousResponse(t *testing.T) {
	for _, mode := range []string{"legacy", "default", "layered"} {
		t.Run(mode, func(t *testing.T) {
			a := Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Extra: map[string]any{"codex_cli_only": true, "codex_cli_only_route_only": true, "responses_websockets_v2_enabled": true}}
			b := a
			b.ID = 2
			b.Extra = nil
			s := &OpenAIGatewayService{accountRepo: stubOpenAIAccountRepo{accounts: []Account{a, b}}, cache: &stubGatewayCache{}, cfg: newOpenAIWSV2TestConfig()}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
			s.PrepareCodexClientRouting(c, []byte(`{}`))
			ctx := c.Request.Context()
			store := s.getOpenAIWSStateStore()
			require.NoError(t, store.BindResponseAccount(ctx, 0, "resp_filtered", 1, time.Hour))
			sel, err := s.SelectAccountByPreviousResponseID(ctx, nil, "resp_filtered", "gpt-5", nil, false)
			require.NoError(t, err)
			require.Nil(t, sel)
			if mode == "legacy" {
				return
			}
			var scheduler OpenAIAccountScheduler = newDefaultOpenAIAccountScheduler(s, nil)
			if mode == "layered" {
				scheduler = newLayeredOpenAIAccountScheduler(s, nil)
			}
			sel, _, err = scheduler.Select(ctx, OpenAIAccountScheduleRequest{Platform: PlatformOpenAI, PreviousResponseID: "resp_filtered", RequestedModel: "gpt-5"})
			require.NoError(t, err)
			require.NotNil(t, sel)
			require.Equal(t, int64(2), sel.Account.ID)
			if sel.ReleaseFunc != nil {
				sel.ReleaseFunc()
			}
		})
	}
}

func TestCodexRoutingSchedulers(t *testing.T) {
	for _, mode := range []string{"legacy", "default", "layered"} {
		t.Run(mode, func(t *testing.T) {
			a := Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1, Extra: map[string]any{"codex_cli_only": true, "codex_cli_only_route_only": true}}
			b := a
			b.ID = 2
			b.Extra = nil
			repo := stubOpenAIAccountRepo{accounts: []Account{a, b}}
			cache := &stubGatewayCache{sessionBindings: map[string]int64{"openai:sticky": 1}}
			s := &OpenAIGatewayService{accountRepo: repo, cache: cache}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
			s.PrepareCodexClientRouting(c, []byte(`{}`))
			ctx := c.Request.Context()
			selectAccount := func(excluded map[int64]struct{}) (*Account, error) {
				if mode == "legacy" {
					return s.SelectAccountForModelWithExclusions(ctx, nil, "sticky", "gpt-5", excluded)
				}
				var sched OpenAIAccountScheduler = newDefaultOpenAIAccountScheduler(s, nil)
				if mode == "layered" {
					sched = newLayeredOpenAIAccountScheduler(s, nil)
				}
				sel, _, err := sched.Select(ctx, OpenAIAccountScheduleRequest{Platform: PlatformOpenAI, SessionHash: "sticky", RequestedModel: "gpt-5", ExcludedIDs: excluded})
				if sel == nil {
					return nil, err
				}
				if sel.ReleaseFunc != nil {
					defer sel.ReleaseFunc()
				}
				return sel.Account, err
			}
			selected, err := selectAccount(nil)
			require.NoError(t, err)
			require.Equal(t, int64(2), selected.ID)
			require.NoError(t, s.BindStickySessionAfterProfitAdmission(ctx, nil, "sticky", selected.ID))
			require.Equal(t, int64(2), cache.sessionBindings["openai:sticky"])
			_, err = selectAccount(map[int64]struct{}{2: {}})
			require.Error(t, err)
			latest, vetoed, reason := s.ProfitControlVetoLatest(ctx, &a)
			require.True(t, vetoed)
			require.Equal(t, "codex_client_restricted", reason)
			require.Equal(t, int64(1), latest.ID)
			require.False(t, isOpenAIAccountRequestCompatible(ctx, s, &a, OpenAIAccountScheduleRequest{Platform: PlatformOpenAI}))
			require.Nil(t, s.recheckSelectedOpenAIAccountFromDB(ctx, &a, nil, PlatformOpenAI, "gpt-5", false, ""))
			parentID := int64(2)
			shadow := a
			shadow.ParentAccountID = &parentID
			require.False(t, isOpenAIAccountRequestCompatible(ctx, s, &shadow, OpenAIAccountScheduleRequest{Platform: PlatformOpenAI}))
			copied, copyErr := duplicateAccountExtra(a.Extra)
			require.NoError(t, copyErr)
			require.Equal(t, true, copied["codex_cli_only_route_only"])
			a.Extra["codex_cli_only_route_only"] = false
			require.True(t, codexClientRouteAllowed(ctx, &a))
			selected, err = selectAccount(map[int64]struct{}{2: {}})
			require.NoError(t, err)
			require.Equal(t, int64(1), selected.ID)
			require.False(t, s.detectCodexClientRestriction(c, selected, []byte(`{}`)).Matched)
			// Metadata calls retain their existing selection scope.
			a.Extra["codex_cli_only_route_only"] = true
			require.True(t, codexClientRouteAllowed(context.Background(), &a))
		})
	}
}
