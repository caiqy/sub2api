package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

func TestProbe_SendProbeRequest_OAuthUsesCodexResponsesEndpoint(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{
		resp: &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: {\"id\":\"resp_123\",\"type\":\"response.created\"}\n\n"))},
	}
	probe := &openAIAccountProbe{
		service: &OpenAIGatewayService{httpUpstream: upstream},
	}
	account := &Account{
		ID:          1,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "acct-123",
		},
	}

	result := probe.sendProbeRequest(context.Background(), account, "gpt-4o-mini", GatewayOpenAIWSSchedulerLayeredConfig{ProbeTimeoutSeconds: 1})
	require.NoError(t, result.err)

	require.NotNil(t, upstream.lastReq)
	require.Equal(t, chatgptCodexURL, upstream.lastReq.URL.String())
	require.Equal(t, "chatgpt.com", upstream.lastReq.Host)
	require.Equal(t, "text/event-stream", upstream.lastReq.Header.Get("accept"))
	require.Equal(t, "responses=experimental", upstream.lastReq.Header.Get("OpenAI-Beta"))
	require.Equal(t, "codex_cli_rs", upstream.lastReq.Header.Get("originator"))
	require.Equal(t, codexCLIUserAgent, upstream.lastReq.Header.Get("User-Agent"))
	require.Equal(t, "Bearer oauth-token", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "acct-123", upstream.lastReq.Header.Get("chatgpt-account-id"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(upstream.lastBody, &body))
	require.Equal(t, "gpt-4o-mini", body["model"])
	require.Equal(t, true, body["stream"])
	require.Contains(t, body, "input")
	require.Equal(t, float64(probeMaxTokens), body["max_output_tokens"])
	require.NotContains(t, body, "max_tokens")
	require.NotContains(t, body, "instructions")
	require.NotContains(t, string(upstream.lastBody), "messages")
}

func TestProbe_SendProbeRequest_OAuthUsesCustomUserAgentWhenConfigured(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{
		resp: &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: {\"id\":\"resp_123\",\"type\":\"response.created\"}\n\n"))},
	}
	probe := &openAIAccountProbe{
		service: &OpenAIGatewayService{httpUpstream: upstream},
	}
	account := &Account{
		ID:          11,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "acct-123",
			"user_agent":         "custom-probe-ua/1.0",
		},
	}

	result := probe.sendProbeRequest(context.Background(), account, "gpt-4o-mini", GatewayOpenAIWSSchedulerLayeredConfig{ProbeTimeoutSeconds: 1})
	require.NoError(t, result.err)
	require.Equal(t, "custom-probe-ua/1.0", upstream.lastReq.Header.Get("User-Agent"))
}

func TestProbe_SendProbeRequest_APIKeyUsesResponsesURLBuilder(t *testing.T) {
	upstream := &openAIHTTPUpstreamRecorder{
		resp: &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: {\"id\":\"resp_123\",\"type\":\"response.created\"}\n\n"))},
	}
	probe := &openAIAccountProbe{
		service: &OpenAIGatewayService{httpUpstream: upstream},
	}
	account := &Account{
		ID:          2,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://example.com/v1",
		},
	}

	result := probe.sendProbeRequest(context.Background(), account, "gpt-4o-mini", GatewayOpenAIWSSchedulerLayeredConfig{ProbeTimeoutSeconds: 1})
	require.NoError(t, result.err)

	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://example.com/v1/responses", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer sk-test", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "application/json", upstream.lastReq.Header.Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(upstream.lastBody, &body))
	require.Equal(t, "gpt-4o-mini", body["model"])
	require.Equal(t, true, body["stream"])
	require.Contains(t, body, "input")
	require.Equal(t, float64(probeMaxTokens), body["max_output_tokens"])
	require.NotContains(t, body, "max_tokens")
	require.NotContains(t, body, "instructions")
	require.NotContains(t, string(upstream.lastBody), "messages")
}

func TestProbe_SendProbeRequest_UsesFirstValidSSEEventAsTTFT(t *testing.T) {
	fixture := newAPIKeyProbeFixtureWithUpstream(&contextAwareBlockingProbeUpstream{bodyFactory: func(ctx context.Context) io.ReadCloser {
		return &contextAwareBlockingProbeBody{
			ctx: ctx,
			chunks: []string{
				": keep-alive\n\n",
				"\n",
				"data: {\"id\":\"resp_123\",\"type\":\"response.created\"}\n\n",
			},
		}
	}})
	fixture.lcfg.ProbeTimeoutSeconds = 1
	start := time.Now()

	result := fixture.probe.sendProbeRequest(context.Background(), fixture.account, "gpt-4o-mini", fixture.lcfg)
	elapsed := time.Since(start)

	require.NoError(t, result.err)
	require.GreaterOrEqual(t, result.ttftMs, 0)
	require.Less(t, elapsed, 700*time.Millisecond, "should return as soon as the first valid SSE event arrives, without waiting for stream completion or timeout")
}

func TestProbe_SendProbeRequest_AcceptsMultiLineDataEvent(t *testing.T) {
	fixture := newAPIKeyProbeFixture(io.NopCloser(strings.NewReader("data: {\"id\":\"resp_123\",\n" +
		"data: \"type\":\"response.created\"}\n\n")))

	result := fixture.probe.sendProbeRequest(context.Background(), fixture.account, "gpt-4o-mini", fixture.lcfg)

	require.NoError(t, result.err)
	require.GreaterOrEqual(t, result.ttftMs, 0)
}

func TestProbe_SendProbeRequest_IgnoresNonJSONDataBeforeValidEvent(t *testing.T) {
	fixture := newAPIKeyProbeFixture(io.NopCloser(strings.NewReader("data: not-json\n\n" +
		"data: {\"id\":\"resp_123\",\"type\":\"response.created\"}\n\n")))

	result := fixture.probe.sendProbeRequest(context.Background(), fixture.account, "gpt-4o-mini", fixture.lcfg)

	require.NoError(t, result.err)
	require.GreaterOrEqual(t, result.ttftMs, 0)
}

func TestProbe_SendProbeRequest_AcceptsFinalValidDataWithoutTrailingNewline(t *testing.T) {
	fixture := newAPIKeyProbeFixture(io.NopCloser(strings.NewReader("data: {\"id\":\"resp_123\",\"type\":\"response.created\"}")))

	result := fixture.probe.sendProbeRequest(context.Background(), fixture.account, "gpt-4o-mini", fixture.lcfg)

	require.NoError(t, result.err)
	require.GreaterOrEqual(t, result.ttftMs, 0)
}

func TestProbe_SendProbeRequest_FailsOnErrorEvent(t *testing.T) {
	fixture := newAPIKeyProbeFixture(io.NopCloser(strings.NewReader("data: {\"error\":{\"message\":\"boom\"}}\n\n")))

	result := fixture.probe.sendProbeRequest(context.Background(), fixture.account, "gpt-4o-mini", fixture.lcfg)

	require.Error(t, result.err)
	require.ErrorContains(t, result.err, "boom")
}

func TestProbe_SendProbeRequest_FailsOnExplicitSSEErrorEventType(t *testing.T) {
	fixture := newAPIKeyProbeFixture(io.NopCloser(strings.NewReader("event: error\n" +
		"data: {\"message\":\"boom\"}\n\n")))

	result := fixture.probe.sendProbeRequest(context.Background(), fixture.account, "gpt-4o-mini", fixture.lcfg)

	require.Error(t, result.err)
	require.ErrorContains(t, result.err, "boom")
}

func TestProbe_SendProbeRequest_FailsOnExplicitSSEErrorEventWithoutData(t *testing.T) {
	fixture := newAPIKeyProbeFixture(io.NopCloser(strings.NewReader("event: error\n\n")))

	result := fixture.probe.sendProbeRequest(context.Background(), fixture.account, "gpt-4o-mini", fixture.lcfg)

	require.Error(t, result.err)
	require.ErrorContains(t, result.err, "error event")
}

func TestProbe_SendProbeRequest_FailsOnDoneBeforeValidEvent(t *testing.T) {
	fixture := newAPIKeyProbeFixture(io.NopCloser(strings.NewReader("data: [DONE]\n\n")))

	result := fixture.probe.sendProbeRequest(context.Background(), fixture.account, "gpt-4o-mini", fixture.lcfg)

	require.Error(t, result.err)
	require.ErrorContains(t, result.err, "before valid event")
	require.ErrorContains(t, result.err, "[DONE]")
}

func TestProbe_SendProbeRequest_FailsWhenStreamEndsBeforeValidEvent(t *testing.T) {
	fixture := newAPIKeyProbeFixture(io.NopCloser(strings.NewReader(": keep-alive\n\n\n")))

	result := fixture.probe.sendProbeRequest(context.Background(), fixture.account, "gpt-4o-mini", fixture.lcfg)

	require.Error(t, result.err)
	require.ErrorContains(t, result.err, "before valid event")
}

func TestProbe_SendProbeRequest_TimesOutWithoutValidEvent(t *testing.T) {
	fixture := newAPIKeyProbeFixtureWithUpstream(&contextAwareBlockingProbeUpstream{})
	fixture.lcfg.ProbeTimeoutSeconds = 1

	result := fixture.probe.sendProbeRequest(context.Background(), fixture.account, "gpt-4o-mini", fixture.lcfg)

	require.Error(t, result.err)
	require.ErrorContains(t, result.err, "context deadline exceeded")
}

func TestProbe_ResolveProbeModel_KeepsExistingSelectionRules(t *testing.T) {
	probe := &openAIAccountProbe{}
	mappedAccount := &Account{
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"*":             "gpt-4.1-mini",
				"gpt-5.1-codex": "gpt-4.1-mini",
				"gpt-4o-mini":   "gpt-4o-mini-upstream",
			},
		},
	}
	fallbackAccount := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{}}

	require.Equal(t, "gpt-4o-mini", probe.resolveProbeModel(mappedAccount))
	require.Equal(t, "gpt-4o-mini", probe.resolveProbeModel(fallbackAccount))
}

type apiKeyProbeFixture struct {
	probe   *openAIAccountProbe
	account *Account
	lcfg    GatewayOpenAIWSSchedulerLayeredConfig
}

func newAPIKeyProbeFixture(body io.ReadCloser) apiKeyProbeFixture {
	return newAPIKeyProbeFixtureWithUpstream(&openAIHTTPUpstreamRecorder{
		resp: &http.Response{StatusCode: http.StatusOK, Body: body},
	})
}

func newAPIKeyProbeFixtureWithUpstream(upstream HTTPUpstream) apiKeyProbeFixture {
	return apiKeyProbeFixture{
		probe: &openAIAccountProbe{service: &OpenAIGatewayService{httpUpstream: upstream}},
		account: &Account{
			ID:          3,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Concurrency: 1,
			Credentials: map[string]any{
				"api_key":  "sk-test",
				"base_url": "https://example.com/v1",
			},
		},
		lcfg: GatewayOpenAIWSSchedulerLayeredConfig{ProbeTimeoutSeconds: 2},
	}
}

type contextAwareBlockingProbeBody struct {
	ctx    context.Context
	chunks []string
	index  int
}

func (b *contextAwareBlockingProbeBody) Read(p []byte) (int, error) {
	if b.index < len(b.chunks) {
		chunk := b.chunks[b.index]
		b.index++
		copyLen := copy(p, []byte(chunk))
		return copyLen, nil
	}
	if b.ctx == nil {
		return 0, io.EOF
	}
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (b *contextAwareBlockingProbeBody) Close() error {
	return nil
}

type contextAwareBlockingProbeUpstream struct {
	bodyFactory func(context.Context) io.ReadCloser
}

func (u *contextAwareBlockingProbeUpstream) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	body := io.ReadCloser(&contextAwareBlockingProbeBody{ctx: req.Context()})
	if u.bodyFactory != nil {
		body = u.bodyFactory(req.Context())
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       body,
	}, nil
}

func (u *contextAwareBlockingProbeUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

type failingClearTempUnschedulableRepo struct{ stubOpenAIAccountRepo }

type probeGroupAwareRepo struct{ stubOpenAIAccountRepo }

type failingGroupLookupProbeRepo struct{ stubOpenAIAccountRepo }

type panicExplainabilityRepo struct{ stubOpenAIAccountRepo }

func newProbeGroupAwareRepo(accounts []Account) probeGroupAwareRepo {
	return probeGroupAwareRepo{stubOpenAIAccountRepo{accounts: accounts}}
}

func (f failingClearTempUnschedulableRepo) ClearTempUnschedulable(ctx context.Context, id int64) error {
	return errors.New("clear failed")
}

func (r failingGroupLookupProbeRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	return nil, errors.New("group lookup failed")
}

func (r panicExplainabilityRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	panic("unexpected explainability repo call")
}

func (r panicExplainabilityRepo) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	panic("unexpected explainability repo call")
}

func (r probeGroupAwareRepo) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform == platform && acc.IsSchedulable() && len(acc.AccountGroups) == 0 {
			result = append(result, acc)
		}
	}
	return result, nil
}

func (r probeGroupAwareRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform != platform || !acc.IsSchedulable() {
			continue
		}
		for _, ag := range acc.AccountGroups {
			if ag.GroupID == groupID {
				result = append(result, acc)
				break
			}
		}
	}
	return result, nil
}

func TestProbe_MarkPenalized_RegistersAccount(t *testing.T) {
	probe := &openAIAccountProbe{
		stats:  newOpenAIAccountRuntimeStats(),
		stopCh: make(chan struct{}),
	}
	defer probe.stop()
	probe.markPenalized(42, nil, true, false)
	_, ok := probe.entries.Load(int64(42))
	require.True(t, ok)
}

func TestProbe_MarkPenalized_IsIdempotent(t *testing.T) {
	probe := &openAIAccountProbe{
		stats:  newOpenAIAccountRuntimeStats(),
		stopCh: make(chan struct{}),
	}
	defer probe.stop()

	probe.markPenalized(42, nil, true, false)
	val1, ok1 := probe.entries.Load(int64(42))
	require.True(t, ok1)

	probe.markPenalized(42, nil, true, false)
	val2, ok2 := probe.entries.Load(int64(42))
	require.True(t, ok2)

	// LoadOrStore returns the existing entry on second call, so pointers must match.
	require.Same(t, val1.(*openAIAccountProbeEntry), val2.(*openAIAccountProbeEntry))
}

func TestProbe_MarkPenalized_OverwritesReasonFlagsToCurrentEvaluation(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{})}
	defer probe.stop()

	probe.markPenalized(42, nil, true, true)
	value, ok := probe.entries.Load(int64(42))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)
	require.True(t, entry.errorPenalized.Load())
	require.True(t, entry.ttftPenalized.Load())

	probe.markPenalized(42, nil, true, false)
	require.True(t, entry.errorPenalized.Load())
	require.False(t, entry.ttftPenalized.Load(), "reason flags should reflect current evaluation, not accumulate stale true values")
}

func TestProbe_ClearPenaltyReasons_RemovesEntryWhenNoReasonsRemain(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{})}
	defer probe.stop()

	probe.markPenalized(7, nil, true, true)
	probe.clearPenaltyReasons(7)
	_, ok := probe.entries.Load(int64(7))
	require.False(t, ok)
}

func TestProbe_ClearPenaltyReasons_DoesNotRemoveEntryWhenProbing(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{})}
	defer probe.stop()

	probe.markPenalized(9, nil, true, true)
	value, ok := probe.entries.Load(int64(9))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)
	entry.probing.Store(true)

	probe.clearPenaltyReasons(9)
	_, ok = probe.entries.Load(int64(9))
	require.True(t, ok, "entry must remain while probing is in-flight")
	require.False(t, entry.errorPenalized.Load())
	require.False(t, entry.ttftPenalized.Load())
}

func TestProbe_ClearPenaltyReasons_DoesNotRemoveEntryWhenDBFlagSet(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{})}
	defer probe.stop()

	probe.markPenalized(10, nil, true, true)
	value, ok := probe.entries.Load(int64(10))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)
	entry.dbFlagSet.Store(true)

	probe.clearPenaltyReasons(10)
	_, ok = probe.entries.Load(int64(10))
	require.True(t, ok, "entry must remain while db flag is set")
	require.False(t, entry.errorPenalized.Load())
	require.False(t, entry.ttftPenalized.Load())
}

func TestProbe_FinalizePenaltyState_KeepsEntryWhenTTFTReasonRemains(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{}), ctx: context.Background()}
	defer probe.stop()

	probe.markPenalized(1, nil, true, true)
	value, ok := probe.entries.Load(int64(1))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)

	entry.errorPenalized.Store(false)
	entry.ttftPenalized.Store(true)
	probe.finalizePenaltyState(1, entry)

	_, ok = probe.entries.Load(int64(1))
	require.True(t, ok, "entry must remain while TTFT reason is still active")
}

func TestProbe_FinalizePenaltyState_RemovesEntryWhenBothReasonsClear(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{}), ctx: context.Background()}
	defer probe.stop()

	probe.markPenalized(1, nil, true, true)
	value, ok := probe.entries.Load(int64(1))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)

	entry.errorPenalized.Store(false)
	entry.ttftPenalized.Store(false)
	probe.finalizePenaltyState(1, entry)

	_, ok = probe.entries.Load(int64(1))
	require.False(t, ok, "entry should be removed only after both reasons clear")
}

func TestProbe_ReevaluatePenaltyReasons_UsesSharedGroupBaseline(t *testing.T) {
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 10},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 50},
	}
	svc := newLayeredTestService(accounts)
	defer svc.StopOpenAIAccountScheduler()

	stats := newOpenAIAccountRuntimeStats()
	probe := newOpenAIAccountProbe(svc, stats)
	defer probe.stop()

	for i := 0; i < 5; i++ {
		slow := 9000
		fast := 1000
		stats.report(1, true, &slow)
		stats.report(2, true, &fast)
	}

	eval, err := probe.reevaluatePenaltyReasons(context.Background(), 1, nil)
	require.NoError(t, err)
	require.True(t, eval.TTFTPenalized)
	require.False(t, eval.ErrorPenalized)
	require.Greater(t, eval.GroupMinTTFT, 0.0)
}

func TestProbe_ReevaluatePenaltyReasons_UsesAccountGroupID(t *testing.T) {
	groupID := int64(100)
	accounts := []Account{
		{
			ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Schedulable: true, Concurrency: 3, Priority: 10,
			AccountGroups: []AccountGroup{{GroupID: groupID}},
		},
		{
			ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Schedulable: true, Concurrency: 3, Priority: 50,
			AccountGroups: []AccountGroup{{GroupID: groupID}},
		},
		{
			ID: 3, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Schedulable: true, Concurrency: 3, Priority: 50,
			AccountGroups: []AccountGroup{{GroupID: 200}},
		},
	}
	repo := newProbeGroupAwareRepo(accounts)
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyThreshold = 0.3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyValue = 100
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyMultiplier = 3.0
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyValue = 50
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds = 60
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds = 30
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeMaxFailures = 3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeTimeoutSeconds = 15

	svc := &OpenAIGatewayService{accountRepo: repo, cfg: cfg}
	stats := newOpenAIAccountRuntimeStats()
	probe := newOpenAIAccountProbe(svc, stats)
	defer probe.stop()

	for i := 0; i < 5; i++ {
		slow := 9000
		fastSameGroup := 1000
		fastOtherGroup := 500
		stats.report(1, true, &slow)
		stats.report(2, true, &fastSameGroup)
		stats.report(3, true, &fastOtherGroup)
	}

	eval, err := probe.reevaluatePenaltyReasons(context.Background(), 1, probeAccountGroupID(&accounts[0]))
	require.NoError(t, err)
	require.True(t, eval.TTFTPenalized, "account 1 should compare against same-group min TTFT")
	require.InDelta(t, 1000.0, eval.GroupMinTTFT, 0.01, "group min TTFT must come from same group, not other groups")
}

func TestProbe_ReevaluatePenaltyReasons_UsesStoredEntryGroupID(t *testing.T) {
	groupA := int64(100)
	groupB := int64(200)

	accounts := []Account{
		{
			ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Schedulable: true, Concurrency: 3, Priority: 10,
			AccountGroups: []AccountGroup{{GroupID: groupA}, {GroupID: groupB}},
		},
		{
			ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Schedulable: true, Concurrency: 3, Priority: 50,
			AccountGroups: []AccountGroup{{GroupID: groupA}},
		},
		{
			ID: 3, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Schedulable: true, Concurrency: 3, Priority: 50,
			AccountGroups: []AccountGroup{{GroupID: groupB}},
		},
	}
	repo := newProbeGroupAwareRepo(accounts)
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.SchedulerMode = "layered"
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyThreshold = 0.3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ErrorPenaltyValue = 100
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyMultiplier = 3.0
	cfg.Gateway.OpenAIWS.SchedulerLayered.TTFTPenaltyValue = 50
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeCooldownSeconds = 60
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeIntervalSeconds = 30
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeMaxFailures = 3
	cfg.Gateway.OpenAIWS.SchedulerLayered.ProbeTimeoutSeconds = 15

	svc := &OpenAIGatewayService{accountRepo: repo, cfg: cfg}
	stats := newOpenAIAccountRuntimeStats()
	probe := newOpenAIAccountProbe(svc, stats)
	defer probe.stop()

	for i := 0; i < 5; i++ {
		slow := 9000
		fastA := 1000
		fastB := 500
		stats.report(1, true, &slow)
		stats.report(2, true, &fastA)
		stats.report(3, true, &fastB)
	}

	probe.markPenalized(1, &groupA, false, true)
	value, _ := probe.entries.Load(int64(1))
	entry := value.(*openAIAccountProbeEntry)

	eval, err := probe.reevaluatePenaltyReasons(context.Background(), 1, probeEntryGroupID(entry))
	require.NoError(t, err)
	require.True(t, eval.TTFTPenalized)
	require.InDelta(t, 1000.0, eval.GroupMinTTFT, 0.01, "must use stored group A baseline, not group B")
}

func TestProbe_ReevaluatePenaltyReasons_ReturnsErrorWhenGroupBaselineQueryFails(t *testing.T) {
	groupID := int64(100)
	svc := &OpenAIGatewayService{
		accountRepo: failingGroupLookupProbeRepo{},
		cfg:         &config.Config{},
	}
	stats := newOpenAIAccountRuntimeStats()
	probe := newOpenAIAccountProbe(svc, stats)
	defer probe.stop()

	_, err := probe.reevaluatePenaltyReasons(context.Background(), 1, &groupID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "group lookup failed")
}

func TestProbe_RecoverAccount_ResetsEWMA(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	probe := &openAIAccountProbe{
		stats:  stats,
		stopCh: make(chan struct{}),
	}
	defer probe.stop()

	// Report 5 failures → errorRate > 0.3
	for i := 0; i < 5; i++ {
		stats.report(1, false, nil)
	}
	errorRate, _, _ := stats.snapshot(1)
	require.Greater(t, errorRate, 0.3, "errorRate should exceed 0.3 after 5 failures")

	// Register entry in probe list
	probe.markPenalized(1, nil, true, false)
	val, ok := probe.entries.Load(int64(1))
	require.True(t, ok)
	entry := val.(*openAIAccountProbeEntry)

	// Recover
	probe.recoverAccount(1, entry)

	// After recovery: errorRate == 0, TTFT unchanged (not reset)
	errorRate, _, _ = stats.snapshot(1)
	require.Equal(t, 0.0, errorRate, "errorRate should be reset to 0")

	// Entry should be removed from probe list
	_, ok = probe.entries.Load(int64(1))
	require.False(t, ok, "entry should be removed from probe list after recovery")
}

func TestProbe_RecoverAccount_KeepsEntryWhenClearTempUnschedulableFails(t *testing.T) {
	probe := &openAIAccountProbe{
		stats:  newOpenAIAccountRuntimeStats(),
		stopCh: make(chan struct{}),
		ctx:    context.Background(),
		service: &OpenAIGatewayService{
			accountRepo: failingClearTempUnschedulableRepo{},
		},
	}
	defer probe.stop()

	probe.markPenalized(123, nil, true, false)
	value, ok := probe.entries.Load(int64(123))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)
	entry.dbFlagSet.Store(true)

	probe.recoverAccount(123, entry)

	_, ok = probe.entries.Load(int64(123))
	require.True(t, ok, "entry must remain so future probes can retry DB cleanup")
	require.True(t, entry.dbFlagSet.Load(), "dbFlagSet should remain true after failed clear")
}

func TestProbe_ManualRecovery_ClearsReasonsButPreservesTTFT(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	ttft := 1500
	stats.report(1, true, &ttft)

	probe := &openAIAccountProbe{stats: stats, stopCh: make(chan struct{}), ctx: context.Background()}
	defer probe.stop()
	probe.markPenalized(1, nil, true, true)

	value, ok := probe.entries.Load(int64(1))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)
	entry.dbFlagSet.Store(true)

	probe.applyManualRecovery(1, entry)

	errRate, ttftAfter, hasTTFT := stats.snapshot(1)
	require.Equal(t, 0.0, errRate)
	require.True(t, hasTTFT)
	require.InDelta(t, 1500.0, ttftAfter, 0.01)
	_, ok = probe.entries.Load(int64(1))
	require.False(t, ok)
}

func TestProbe_ManualRecovery_MarksEntryToIgnoreStaleProbeResults(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{}), ctx: context.Background()}
	defer probe.stop()
	probe.markPenalized(1, nil, true, true)

	value, ok := probe.entries.Load(int64(1))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)

	probe.applyManualRecovery(1, entry)
	require.True(t, entry.ignoreResults.Load(), "manual recovery should mark stale in-flight probe results to be ignored")
}

func TestProbe_ExplainabilityFields_IncludeRuntimeMetrics(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()
	ttft := 1200
	stats.report(1, true, &ttft)
	probe := &openAIAccountProbe{stats: stats, stopCh: make(chan struct{}), ctx: context.Background(), service: &OpenAIGatewayService{accountRepo: panicExplainabilityRepo{}}}
	defer probe.stop()
	probe.markPenalized(1, nil, true, false)

	value, ok := probe.entries.Load(int64(1))
	require.True(t, ok)
	entry := value.(*openAIAccountProbeEntry)
	entry.lastProbeTTFTMs.Store(1200)

	var fields []any
	require.NotPanics(t, func() {
		fields = probe.explainabilityFields(1, entry, 0)
	})
	joined := fmt.Sprint(fields...)
	require.Contains(t, joined, "error_rate")
	require.Contains(t, joined, "ttft")
	require.Contains(t, joined, "group_min_ttft")
	require.Contains(t, joined, "last_probe_ttft_ms")
}

func TestProbe_SuccessPath_LeavesEntryWhenTTFTStillPenalized(t *testing.T) {
	probe := &openAIAccountProbe{stats: newOpenAIAccountRuntimeStats(), stopCh: make(chan struct{}), ctx: context.Background()}
	defer probe.stop()
	probe.markPenalized(1, nil, true, true)
	value, _ := probe.entries.Load(int64(1))
	entry := value.(*openAIAccountProbeEntry)

	entry.errorPenalized.Store(false)
	entry.ttftPenalized.Store(true)
	probe.finalizePenaltyState(1, entry)

	_, ok := probe.entries.Load(int64(1))
	require.True(t, ok)
}

func TestProbe_ResetAccount_PreservesTTFT(t *testing.T) {
	stats := newOpenAIAccountRuntimeStats()

	// Report a TTFT so hasTTFT becomes true
	ttftVal := 500
	stats.report(1, true, &ttftVal)
	_, ttft, hasTTFT := stats.snapshot(1)
	require.True(t, hasTTFT)
	require.Greater(t, ttft, 0.0)

	// Reset — should only clear errorRate, not TTFT
	stats.resetAccount(1)

	errRate, ttftAfter, hasTTFTAfter := stats.snapshot(1)
	require.Equal(t, 0.0, errRate, "errorRate should be 0 after reset")
	require.True(t, hasTTFTAfter, "TTFT should be preserved after reset")
	require.InDelta(t, ttft, ttftAfter, 0.01, "TTFT value should be unchanged")
}

func TestProbe_Stop_PreventsNewRegistrations(t *testing.T) {
	probe := &openAIAccountProbe{
		stats:  newOpenAIAccountRuntimeStats(),
		stopCh: make(chan struct{}),
	}
	probe.stop()

	probe.markPenalized(99, nil, true, false)

	_, ok := probe.entries.Load(int64(99))
	require.False(t, ok, "markPenalized should be no-op after stop()")
}

func TestProbe_StopCancelsRootContextAndPreventsNewWork(t *testing.T) {
	probe := newOpenAIAccountProbe(nil, newOpenAIAccountRuntimeStats())
	require.NotNil(t, probe)

	// stop() 的生命周期契约：
	// 1) 会取消 probe 根 context；
	// 2) 会等待 loop 与所有已注册 worker 退出；
	// 3) 一旦 stop 开始，就不应再接受新的已注册工作。
	// 这里用一个已注册的 in-flight worker 来验证 stop 会等待其观察到取消。
	probe.wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer probe.wg.Done()
		<-probe.ctx.Done()
	}()

	probe.stop()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("probe.stop() did not wait for in-flight work to observe cancellation")
	}

	require.True(t, probe.stopped.Load())
	select {
	case <-probe.ctx.Done():
	default:
		t.Fatal("probe root context should be canceled after stop")
	}

	select {
	case <-probe.stopCh:
	default:
		t.Fatal("probe stopCh should be closed after stop")
	}
}
