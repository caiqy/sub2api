package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type terminalUsageOpenAIEnv struct {
	handler   *OpenAIGatewayHandler
	apiKey    *service.APIKey
	usageRepo *openAIChatCompletionsUsageLogRepoStub
}

type terminalUsageGrokAccountRepo struct{ openAIRetryAccountRepoStub }

func (terminalUsageGrokAccountRepo) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}

// ponytail: fixture only needs successful Grok quota snapshot and rate-limit persistence.
func (terminalUsageGrokAccountRepo) UpdateExtra(context.Context, int64, map[string]any) error {
	return nil
}

func (terminalUsageGrokAccountRepo) SetRateLimited(context.Context, int64, time.Time) error {
	return nil
}

type partialWriteTransportHTTPUpstream struct {
	service.HTTPUpstream
	writePartial func()
	response     *http.Response
}

func (u *partialWriteTransportHTTPUpstream) Do(*http.Request, string, int64, int) (*http.Response, error) {
	u.writePartial()
	if u.response != nil {
		return u.response, nil
	}
	return nil, errors.New("transport failed after partial response")
}

func (u *partialWriteTransportHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

type terminalReadErrorBody struct {
	cancel context.CancelFunc
}

func (b terminalReadErrorBody) Read([]byte) (int, error) {
	if b.cancel != nil {
		b.cancel()
	}
	return 0, errors.New("upstream body read failed")
}

func (terminalReadErrorBody) Close() error { return nil }

type terminalPartialReadErrorBody struct {
	data []byte
	sent bool
}

func (b *terminalPartialReadErrorBody) Read(p []byte) (int, error) {
	if b.sent {
		return 0, errors.New("upstream body read failed after partial response")
	}
	b.sent = true
	return copy(p, b.data), nil
}

func (*terminalPartialReadErrorBody) Close() error { return nil }

type directTerminalHTTPUpstream struct {
	service.HTTPUpstream
	response *http.Response
}

func (u directTerminalHTTPUpstream) Do(*http.Request, string, int64, int) (*http.Response, error) {
	return u.response, nil
}

type markingTerminalHTTPUpstream struct {
	service.HTTPUpstream
	mark       func()
	accountIDs []int64
}

type firstTokenTimeoutHTTPUpstream struct {
	service.HTTPUpstream
	calls int
}

type firstTokenCreatedHTTPUpstream struct {
	service.HTTPUpstream
}

type firstTokenCreatedBody struct {
	ctx     context.Context
	emitted bool
}

type firstOutputFailoverCloseTrackingBody struct {
	io.ReadCloser
	closed chan struct{}
	once   sync.Once
}

func (b *firstOutputFailoverCloseTrackingBody) Close() error {
	b.once.Do(func() { close(b.closed) })
	return b.ReadCloser.Close()
}

type firstOutputFailoverHTTPUpstream struct {
	service.HTTPUpstream
	accountIDs      []int64
	firstWriterDone chan struct{}
}

func (u *firstOutputFailoverHTTPUpstream) Do(_ *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.accountIDs = append(u.accountIDs, accountID)
	if accountID != 1 {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(
				"data: {\"type\":\"response.output_text.delta\",\"delta\":\"replayed\"}\n\n" +
					"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_replayed\",\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"replayed\"}]}],\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n",
			)),
		}, nil
	}

	firstBody, firstWriter := io.Pipe()
	trackedBody := &firstOutputFailoverCloseTrackingBody{ReadCloser: firstBody, closed: make(chan struct{})}
	go func() {
		defer close(u.firstWriterDone)
		defer func() { _ = firstWriter.Close() }()
		_, _ = firstWriter.Write([]byte("data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_first\"}}\n\n"))
		<-trackedBody.closed
	}()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       trackedBody,
	}, nil
}

func (u *firstOutputFailoverHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

func (b *firstTokenCreatedBody) Read(p []byte) (int, error) {
	if !b.emitted {
		b.emitted = true
		return copy(p, "data: {\"type\":\"response.created\"}\n\n"), nil
	}
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (*firstTokenCreatedBody) Close() error { return nil }

func (u firstTokenCreatedHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       &firstTokenCreatedBody{ctx: req.Context()},
	}, nil
}

func (u firstTokenCreatedHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

func (u *firstTokenTimeoutHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.calls++
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-time.After(1500 * time.Millisecond):
		return nil, errors.New("test upstream fallback timeout")
	}
}

func (u *firstTokenTimeoutHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

func (u *markingTerminalHTTPUpstream) Do(_ *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.accountIDs = append(u.accountIDs, accountID)
	u.mark()
	return nil, errors.New("dial failed")
}

func newTerminalUsageOpenAIEnv(t *testing.T, group *service.Group, accountRepo service.AccountRepository, response *http.Response) *terminalUsageOpenAIEnv {
	return newTerminalUsageOpenAIEnvWithUpstream(t, group, accountRepo, &openAIChatCompletionsHTTPUpstreamStub{response: response})
}

func newTerminalUsageOpenAIEnvWithUpstream(t *testing.T, group *service.Group, accountRepo service.AccountRepository, upstream service.HTTPUpstream) *terminalUsageOpenAIEnv {
	t.Helper()
	cfg := &config.Config{
		RunMode:     config.RunModeSimple,
		Default:     config.DefaultConfig{RateMultiplier: 1},
		Gateway:     config.GatewayConfig{MaxAccountSwitches: 1, Scheduling: config.GatewaySchedulingConfig{LoadBatchEnabled: false}},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.HTTPBridgeEnabled = true
	cfg.Gateway.OpenAIWS.HTTPBridgeThresholdBytes = 1
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })
	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		nil,
		nil,
		nil,
		nil,
		openAIChatCompletionsGatewayCacheStub{},
		cfg,
		nil,
		concurrencyService,
		service.NewBillingService(cfg, nil),
		nil,
		billingCacheService,
		upstream,
		service.NewDeferredService(accountRepo, nil, 0),
		nil,
		nil,
		nil,
		nil,
	)
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg)
	h.maxAccountSwitches = 0
	return &terminalUsageOpenAIEnv{
		handler: h,
		apiKey: &service.APIKey{
			ID:      101,
			UserID:  202,
			Status:  service.StatusActive,
			GroupID: &group.ID,
			User:    &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1},
			Group:   group,
		},
		usageRepo: usageRepo,
	}
}

func (e *terminalUsageOpenAIEnv) router(route string, handler gin.HandlerFunc) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), e.apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: e.apiKey.UserID, Concurrency: e.apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST(route, handler)
	return router
}

func TestOpenAIGatewayHandler_FirstTokenTimeoutReturns504AndCreatesOneFailedUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 10, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID:          110,
		Name:        "first-token-timeout",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://example.com/v1"},
	}
	upstream := &firstTokenTimeoutHTTPUpstream{}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, upstream)
	env.handler.cfg.Gateway.OpenAIWS.Enabled = false
	env.handler.cfg.Gateway.OpenAITextFirstTokenTimeout = 1

	reqBody := `{"model":"gpt-5.4","input":"hello","stream":true}`
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	env.router("/v1/responses", env.handler.Responses).ServeHTTP(recorder, req)

	require.Equal(t, http.StatusGatewayTimeout, recorder.Code, recorder.Body.String())
	require.Equal(t, "first_token_timeout", gjson.Get(recorder.Body.String(), "error.type").String())
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
	require.Equal(t, 1, upstream.calls)
	select {
	case log := <-env.usageRepo.created:
		require.NotNil(t, log)
		require.NotNil(t, log.DetailSnapshot)
		require.Contains(t, log.DetailSnapshot.UpstreamResponseBody, `"usage_state":"unknown"`)
	case <-time.After(2 * time.Second):
		t.Fatal("首 Token 超时应提交失败 usage")
	}
	select {
	case duplicate := <-env.usageRepo.created:
		t.Fatalf("首 Token 超时不应重复提交失败 usage: %+v", duplicate)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestOpenAIGatewayHandler_FirstTokenTimeoutSuppressesPreOutputKeepalive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 11, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID: 111, Name: "first-token-keepalive", Platform: service.PlatformOpenAI,
		Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true,
		Concurrency: 1, Priority: 1,
		Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://example.com/v1"},
	}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, firstTokenCreatedHTTPUpstream{})
	env.handler.cfg.Gateway.OpenAIWS.Enabled = false
	env.handler.cfg.Gateway.OpenAITextFirstTokenTimeout = 2
	env.handler.cfg.Gateway.StreamKeepaliveInterval = 1

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.4","input":"hello","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	env.router("/v1/responses", env.handler.Responses).ServeHTTP(recorder, req)

	require.Equal(t, http.StatusGatewayTimeout, recorder.Code, recorder.Body.String())
	require.Equal(t, "first_token_timeout", gjson.Get(recorder.Body.String(), "error.type").String())
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
}

func TestOpenAIGatewayHandler_ResponsesFirstOutputTimeoutFailsOverAfterKeepalive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 12, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	accounts := []*service.Account{
		{ID: 1, Name: "first-output-timeout", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 0, Credentials: map[string]any{"api_key": "sk-first"}},
		{ID: 2, Name: "replayed", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-second"}},
	}
	upstream := &firstOutputFailoverHTTPUpstream{firstWriterDone: make(chan struct{})}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: accounts}, upstream)
	env.handler.cfg.Gateway.OpenAIWS.Enabled = false
	env.handler.cfg.Gateway.OpenAIFirstOutputTimeoutSeconds = 2
	env.handler.cfg.Gateway.StreamKeepaliveInterval = 1
	env.handler.maxAccountSwitches = 1

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.4","input":"hello","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	env.router("/v1/responses", env.handler.Responses).ServeHTTP(recorder, req)

	require.Equal(t, []int64{1, 2}, upstream.accountIDs)
	require.Contains(t, recorder.Body.String(), ":\n\n")
	require.NotContains(t, recorder.Body.String(), "resp_first")
	require.Contains(t, recorder.Body.String(), "resp_replayed")
	select {
	case <-upstream.firstWriterDone:
	case <-time.After(time.Second):
		t.Fatal("first upstream body did not close after first-output timeout")
	}
}

func TestOpenAIGatewayHandler_EmbeddingsFailoverExhaustedCreatesFailedUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 1, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 11, Name: "embeddings", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-test"}}
	env := newTerminalUsageOpenAIEnv(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_embeddings_429"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","message":"embeddings overloaded"}}`)),
	})

	reqBody := `{"model":"text-embedding-3-small","input":"hello"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	env.router("/v1/embeddings", env.handler.Embeddings).ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code, rec.Body.String())
	log := waitForOpenAIFailedUsageLog(t, env.usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.DetailSnapshot)
	requireRequestPreviewSnapshot(t, log.DetailSnapshot.RequestBody, reqBody)
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_embeddings_429")
	require.Contains(t, log.DetailSnapshot.ResponseBody, "embeddings overloaded")
}

func TestOpenAIGatewayHandler_GrokMediaFailoverExhaustedCreatesFailedUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 2, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	parentID := int64(120)
	account := &service.Account{ID: 12, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}
	parent := &service.Account{ID: parentID, Name: "grok-credential-parent", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Credentials: map[string]any{"access_token": "grok-token"}}
	env := newTerminalUsageOpenAIEnv(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account, parent}}}, &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_grok_429"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","message":"grok media overloaded"}}`)),
	})

	reqBody := `{"model":"grok-imagine","prompt":"draw a lighthouse"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	env.router("/v1/images/generations", env.handler.GrokImages).ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code, rec.Body.String())
	log := waitForOpenAIFailedUsageLog(t, env.usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.DetailSnapshot)
	requireRequestPreviewSnapshot(t, log.DetailSnapshot.RequestBody, reqBody)
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_grok_429")
	require.Contains(t, log.DetailSnapshot.ResponseBody, "grok media overloaded")
}

func TestOpenAIGatewayHandler_ChatCompletionsPartialFailoverCreatesFailedUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 3, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 13, Name: "chat", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-test"}}
	upstreamBody := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"Hello\"}\n\n" +
		"data: {\"type\":\"response.failed\",\"response\":{\"id\":\"resp_partial\",\"status\":\"failed\",\"error\":{\"type\":\"server_error\",\"code\":\"server_error\",\"message\":\"partial chat failover\"}}}\n\n"
	env := newTerminalUsageOpenAIEnv(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "X-Request-Id": []string{"req_chat_partial"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	})

	reqBody := `{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":true}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	env.router("/v1/chat/completions", env.handler.ChatCompletions).ServeHTTP(rec, req)

	require.Contains(t, rec.Body.String(), "Hello", "test must exercise the post-write failover branch")
	log := waitForOpenAIFailedUsageLog(t, env.usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.DetailSnapshot)
	requireRequestPreviewSnapshot(t, log.DetailSnapshot.RequestBody, reqBody)
	require.Contains(t, log.DetailSnapshot.ResponseBody, "partial chat failover")
}

func TestOpenAIGatewayHandler_ResponsesPartialFailoverCreatesExactlyOneFailedUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 31, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 131, Name: "responses", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-test"}}
	var requestContext *gin.Context
	upstream := &partialWriteTransportHTTPUpstream{}
	upstream.writePartial = func() {
		requestContext.Header("Content-Type", "text/event-stream")
		_, _ = requestContext.Writer.WriteString("data: {\"type\":\"response.output_text.delta\",\"delta\":\"Hello\"}\n\n")
		requestContext.Writer.Flush()
	}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, upstream)
	env.usageRepo.created = make(chan *service.UsageLog, 2)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		requestContext = c
		c.Set(string(middleware.ContextKeyAPIKey), env.apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: env.apiKey.UserID, Concurrency: env.apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/responses", env.handler.Responses)

	reqBody := `{"model":"gpt-5.4","input":"hello","stream":true}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Contains(t, rec.Body.String(), "Hello", "test must exercise the post-write failover branch")
	require.NotNil(t, env.usageRepo.lastLog)
	require.Len(t, env.usageRepo.created, 1)
	require.Contains(t, env.usageRepo.lastLog.DetailSnapshot.ResponseBody, "Upstream request failed")
}

func TestOpenAIGatewayHandler_NativeResponsesFailedIsNotDuplicated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 32, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 132, Name: "native-responses", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-test"}, Extra: map[string]any{"openai_passthrough": true, "use_responses_api": true}}
	upstreamBody := "event: response.failed\ndata: {\"type\":\"response.failed\",\"response\":{\"id\":\"resp_native_failed\",\"status\":\"failed\",\"error\":{\"code\":\"invalid_request\",\"message\":\"native upstream failure\"}}}\n"
	env := newTerminalUsageOpenAIEnv(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.4","input":"hello","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	env.router("/v1/responses", env.handler.Responses).ServeHTTP(rec, req)

	require.Equal(t, 1, strings.Count(rec.Body.String(), "event: response.failed\n"), rec.Body.String())
	require.True(t, strings.HasSuffix(rec.Body.String(), "\n\n"), rec.Body.String())
	require.NotNil(t, env.usageRepo.lastLog)
	require.Len(t, env.usageRepo.created, 1)
}

func TestOpenAIGatewayHandler_PassthroughHTTP400CreatesFailedUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 132, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 232, Name: "passthrough-400", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-test"}, Extra: map[string]any{"openai_passthrough": true}}
	upstreamBody := `{"error":{"type":"invalid_request_error","message":"passthrough rejected payload"}}`
	env := newTerminalUsageOpenAIEnv(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(upstreamBody))})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.4","instructions":"keep-this","input":"hello","stream":false}`))
	req.Header.Set("Content-Type", "application/json")
	env.router("/v1/responses", env.handler.Responses).ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "passthrough rejected payload")
	log := waitForOpenAIFailedUsageLog(t, env.usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.DetailSnapshot)
	require.Contains(t, log.DetailSnapshot.ResponseBody, "passthrough rejected payload")
}

func TestOpenAIGatewayHandler_NativeNonPassthroughResponsesFailedIsNotDuplicated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 33, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 133, Name: "native-responses", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-test"}, Extra: map[string]any{"use_responses_api": true}}
	upstreamBody := "event: response.failed\ndata: {\"type\":\"response.failed\",\"response\":{\"id\":\"resp_native_failed\",\"status\":\"failed\",\"error\":{\"code\":\"invalid_request\",\"message\":\"native upstream failure\"}}}\n\n"
	env := newTerminalUsageOpenAIEnv(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.4","input":"hello","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	env.router("/v1/responses", env.handler.Responses).ServeHTTP(rec, req)

	require.Equal(t, 1, strings.Count(rec.Body.String(), "event: response.failed\n"), rec.Body.String())
	require.NotNil(t, env.usageRepo.lastLog)
	require.Len(t, env.usageRepo.created, 1)
}

func TestOpenAIGatewayHandler_NativeNonPassthroughBufferedResponsesFailedIsNotDuplicated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 34, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 134, Name: "native-responses", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-test"}, Extra: map[string]any{"use_responses_api": true}}
	upstreamBody := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\n" +
		"event: response.failed\ndata: {\"type\":\"response.failed\",\"response\":{\"id\":\"resp_native_failed\",\"status\":\"failed\",\"error\":{\"code\":\"invalid_request\",\"message\":\"native upstream failure\"}}}\n"
	env := newTerminalUsageOpenAIEnv(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	})
	env.handler.cfg.Gateway.StreamKeepaliveInterval = 1

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.4","input":"hello","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	env.router("/v1/responses", env.handler.Responses).ServeHTTP(rec, req)

	require.Equal(t, 1, strings.Count(rec.Body.String(), "event: response.failed\n"), rec.Body.String())
	require.True(t, strings.HasSuffix(rec.Body.String(), "\n\n"), rec.Body.String())
	require.NotNil(t, env.usageRepo.lastLog)
	require.Len(t, env.usageRepo.created, 1)
}

func TestOpenAIGatewayHandler_OrdinaryErrorsRequireUpstreamAttempt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("chat local credential error does not create failed usage", func(t *testing.T) {
		group := &service.Group{ID: 400, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 1400, Name: "chat-local", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{}}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}]}`))
		req.Header.Set("Content-Type", "application/json")
		env.router("/v1/chat/completions", env.handler.ChatCompletions).ServeHTTP(rec, req)

		require.Nil(t, env.usageRepo.lastLog)
	})

	t.Run("chat transport error creates failed usage", func(t *testing.T) {
		group := &service.Group{ID: 401, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 1401, Name: "chat-transport", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
		repo := &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account}}}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, repo, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("dial failed")})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}]}`))
		req.Header.Set("Content-Type", "application/json")
		env.router("/v1/chat/completions", env.handler.ChatCompletions).ServeHTTP(rec, req)

		require.NotNil(t, env.usageRepo.lastLog)
	})

	t.Run("messages local credential error does not create failed usage", func(t *testing.T) {
		group := &service.Group{ID: 402, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
		account := &service.Account{ID: 1402, Name: "messages-local", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{}}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"gpt-5.4","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`))
		req.Header.Set("Content-Type", "application/json")
		env.router("/v1/messages", env.handler.Messages).ServeHTTP(rec, req)

		require.Nil(t, env.usageRepo.lastLog)
	})

	t.Run("messages transport error creates failed usage", func(t *testing.T) {
		group := &service.Group{ID: 403, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
		account := &service.Account{ID: 1403, Name: "messages-transport", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("dial failed")})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"gpt-5.4","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`))
		req.Header.Set("Content-Type", "application/json")
		env.router("/v1/messages", env.handler.Messages).ServeHTTP(rec, req)

		require.NotNil(t, env.usageRepo.lastLog)
	})

	t.Run("embeddings local credential error does not create failed usage", func(t *testing.T) {
		group := &service.Group{ID: 41, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 141, Name: "embeddings-local", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{}}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", strings.NewReader(`{"model":"text-embedding-3-small","input":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		env.router("/v1/embeddings", env.handler.Embeddings).ServeHTTP(rec, req)

		require.Nil(t, env.usageRepo.lastLog)
	})

	t.Run("embeddings transport error creates failed usage", func(t *testing.T) {
		group := &service.Group{ID: 42, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 142, Name: "embeddings-transport", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("dial failed")})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", strings.NewReader(`{"model":"text-embedding-3-small","input":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		env.router("/v1/embeddings", env.handler.Embeddings).ServeHTTP(rec, req)

		require.NotNil(t, env.usageRepo.lastLog)
	})

	t.Run("images local credential error does not create failed usage", func(t *testing.T) {
		group := &service.Group{ID: 43, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
		account := &service.Account{ID: 143, Name: "images-local", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{}}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"gpt-image-2","prompt":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		env.router("/v1/images/generations", env.handler.Images).ServeHTTP(rec, req)

		require.Nil(t, env.usageRepo.lastLog)
	})

	t.Run("images transport error creates failed usage", func(t *testing.T) {
		group := &service.Group{ID: 44, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
		account := &service.Account{ID: 144, Name: "images-transport", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("dial failed")})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"gpt-image-2","prompt":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		env.router("/v1/images/generations", env.handler.Images).ServeHTTP(rec, req)

		require.NotNil(t, env.usageRepo.lastLog)
	})

	t.Run("grok local credential error does not create failed usage", func(t *testing.T) {
		group := &service.Group{ID: 45, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
		account := &service.Account{ID: 145, Name: "grok-local", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIChatCompletionsAccountRepoStub{account: account}, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"grok-imagine","prompt":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		env.router("/v1/images/generations", env.handler.GrokImages).ServeHTTP(rec, req)

		require.Nil(t, env.usageRepo.lastLog)
	})

	t.Run("grok transport error creates failed usage", func(t *testing.T) {
		group := &service.Group{ID: 46, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
		parentID := int64(1460)
		account := &service.Account{ID: 146, Name: "grok-transport", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}
		parent := &service.Account{ID: parentID, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Credentials: map[string]any{"access_token": "grok-token"}}
		repo := &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account, parent}}}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, repo, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("dial failed")})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"grok-imagine","prompt":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		env.router("/v1/images/generations", env.handler.GrokImages).ServeHTTP(rec, req)

		require.NotNil(t, env.usageRepo.lastLog)
	})
}

func TestOpenAIGatewayHandler_UpstreamAttemptSignalResetsAcrossAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 404, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	accountA := &service.Account{ID: 1404, Name: "attempted", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-test"}}
	accountB := &service.Account{ID: 1405, Name: "local-error", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{}}
	var requestContext *gin.Context
	upstream := &markingTerminalHTTPUpstream{mark: func() { service.SetOpsUpstreamAttempted(requestContext, true) }}
	repo := &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{accountA, accountB}}}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, repo, upstream)
	env.handler.maxAccountSwitches = 1

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	env.router("/v1/chat/completions", func(c *gin.Context) {
		requestContext = c
		env.handler.ChatCompletions(c)
	}).ServeHTTP(rec, req)

	require.Equal(t, []int64{accountA.ID}, upstream.accountIDs)
	require.False(t, service.HasOpsUpstreamAttempted(requestContext))
	require.Nil(t, env.usageRepo.lastLog)
}

type terminalUsageGroupRepo struct {
	service.GroupRepository
	group  *service.Group
	groups map[int64]*service.Group
}

func (r terminalUsageGroupRepo) GetByID(_ context.Context, id int64) (*service.Group, error) {
	if group, ok := r.groups[id]; ok {
		return group, nil
	}
	if r.group != nil && r.group.ID == id {
		return r.group, nil
	}
	if r.group != nil && r.group.FallbackGroupIDOnInvalidRequest != nil && *r.group.FallbackGroupIDOnInvalidRequest == id {
		return &service.Group{ID: id, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}, nil
	}
	return nil, service.ErrGroupNotFound
}

func (r terminalUsageGroupRepo) GetByIDLite(ctx context.Context, id int64) (*service.Group, error) {
	return r.GetByID(ctx, id)
}

type terminalUsageSettingRepo struct{ service.SettingRepository }

func (terminalUsageSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	return nil, service.ErrSettingNotFound
}

func (terminalUsageSettingRepo) GetValue(context.Context, string) (string, error) {
	return "", service.ErrSettingNotFound
}

func (terminalUsageSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

type terminalGatewayMessagesEnv struct {
	handler     *GatewayHandler
	apiKey      *service.APIKey
	usageRepo   *openAIChatCompletionsUsageLogRepoStub
	accountRepo *terminalGatewayAccountRepo
}

type terminalGatewayAccountRepo struct {
	openAIRetryAccountRepoStub
	groupIDs []int64
}

func (*terminalGatewayAccountRepo) SetModelRateLimit(context.Context, int64, string, time.Time, ...string) error {
	return nil
}

func (r *terminalGatewayAccountRepo) ListSchedulableByGroupIDAndPlatforms(_ context.Context, groupID int64, platforms []string) ([]service.Account, error) {
	r.groupIDs = append(r.groupIDs, groupID)
	return r.listByGroupAndPlatforms(groupID, platforms), nil
}

func (r *terminalGatewayAccountRepo) ListSchedulableByPlatforms(_ context.Context, platforms []string) ([]service.Account, error) {
	return r.listByPlatforms(platforms), nil
}

func (r *terminalGatewayAccountRepo) listByPlatforms(platforms []string) []service.Account {
	allowed := make(map[string]struct{}, len(platforms))
	for _, platform := range platforms {
		allowed[platform] = struct{}{}
	}
	accounts := make([]service.Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if account != nil {
			if _, ok := allowed[account.Platform]; ok {
				accounts = append(accounts, *account)
			}
		}
	}
	return accounts
}

func (r *terminalGatewayAccountRepo) listByGroupAndPlatforms(groupID int64, platforms []string) []service.Account {
	accounts := r.listByPlatforms(platforms)
	filtered := make([]service.Account, 0, len(accounts))
	for _, account := range accounts {
		if len(account.AccountGroups) == 0 {
			filtered = append(filtered, account)
			continue
		}
		for _, accountGroup := range account.AccountGroups {
			if accountGroup.GroupID == groupID {
				filtered = append(filtered, account)
				break
			}
		}
	}
	return filtered
}

func newTerminalGatewayMessagesEnv(t *testing.T, group *service.Group, upstream service.HTTPUpstream, accounts ...*service.Account) *terminalGatewayMessagesEnv {
	return newTerminalGatewayMessagesEnvWithGatewayCache(t, group, upstream, openAIChatCompletionsConcurrencyCacheStub{}, openAIChatCompletionsGatewayCacheStub{}, accounts...)
}

func newTerminalGatewayMessagesEnvWithConcurrencyCache(t *testing.T, group *service.Group, upstream service.HTTPUpstream, concurrencyCache service.ConcurrencyCache, accounts ...*service.Account) *terminalGatewayMessagesEnv {
	return newTerminalGatewayMessagesEnvWithGatewayCache(t, group, upstream, concurrencyCache, openAIChatCompletionsGatewayCacheStub{}, accounts...)
}

func newTerminalGatewayMessagesEnvWithGatewayCache(t *testing.T, group *service.Group, upstream service.HTTPUpstream, concurrencyCache service.ConcurrencyCache, cache service.GatewayCache, accounts ...*service.Account) *terminalGatewayMessagesEnv {
	return newTerminalGatewayMessagesEnvWithGatewayCacheAndGroups(t, group, nil, upstream, concurrencyCache, cache, accounts...)
}

func newTerminalGatewayMessagesEnvWithGatewayCacheAndGroups(t *testing.T, group *service.Group, groups map[int64]*service.Group, upstream service.HTTPUpstream, concurrencyCache service.ConcurrencyCache, cache service.GatewayCache, accounts ...*service.Account) *terminalGatewayMessagesEnv {
	t.Helper()
	cfg := &config.Config{
		RunMode:     config.RunModeSimple,
		Default:     config.DefaultConfig{RateMultiplier: 1},
		Gateway:     config.GatewayConfig{MaxAccountSwitches: 1, Scheduling: config.GatewaySchedulingConfig{LoadBatchEnabled: false}},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}
	accountRepo := &terminalGatewayAccountRepo{openAIRetryAccountRepoStub: openAIRetryAccountRepoStub{accounts: accounts}}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 2)}
	concurrencyService := service.NewConcurrencyService(concurrencyCache)
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })
	settingService := service.NewSettingService(terminalUsageSettingRepo{}, cfg)
	groupRepo := terminalUsageGroupRepo{group: group, groups: groups}
	gatewayService := service.NewGatewayService(
		accountRepo,
		groupRepo,
		usageRepo,
		nil,
		nil,
		nil,
		nil,
		cache,
		cfg,
		nil,
		concurrencyService,
		service.NewBillingService(cfg, nil),
		nil,
		billingCacheService,
		nil,
		upstream,
		service.NewDeferredService(accountRepo, nil, 0),
		nil,
		nil,
		nil,
		nil,
		settingService,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	tokenProvider := service.NewAntigravityTokenProvider(accountRepo, nil, nil)
	antigravityService := service.NewAntigravityGatewayService(accountRepo, cache, nil, tokenProvider, nil, upstream, settingService, nil)
	geminiCompatService := service.NewGeminiMessagesCompatService(accountRepo, groupRepo, cache, nil, nil, nil, upstream, antigravityService, cfg)
	return &terminalGatewayMessagesEnv{
		handler:     NewGatewayHandler(gatewayService, geminiCompatService, antigravityService, nil, concurrencyService, billingCacheService, nil, &service.APIKeyService{}, nil, nil, nil, nil, cfg, settingService),
		apiKey:      &service.APIKey{ID: 101, UserID: 202, Status: service.StatusActive, GroupID: &group.ID, User: &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1}, Group: group},
		usageRepo:   usageRepo,
		accountRepo: accountRepo,
	}
}

func (e *terminalGatewayMessagesEnv) router() *gin.Engine {
	return e.routerFor("/v1/messages", e.handler.Messages)
}

func (e *terminalGatewayMessagesEnv) routerFor(route string, handler gin.HandlerFunc) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), e.apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: e.apiKey.UserID, Concurrency: e.apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST(route, handler)
	return router
}

type cancelingTerminalHTTPUpstream struct {
	service.HTTPUpstream
	cancel context.CancelFunc
}

type promptTooLongFallbackBillingCache struct {
	service.BillingCache
	calls int
}

func (c *promptTooLongFallbackBillingCache) GetUserBalance(context.Context, int64) (float64, error) {
	c.calls++
	if c.calls == 1 {
		return 100, nil
	}
	return 0, nil
}

func (u cancelingTerminalHTTPUpstream) Do(*http.Request, string, int64, int) (*http.Response, error) {
	u.cancel()
	return nil, context.Canceled
}

func TestGatewayHandler_MessagesPromptTooLongWithoutFallbackCreatesFailedUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 4, Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 14, Name: "antigravity", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}
	upstream := &openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_prompt_too_long"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"Prompt is too long"}}`)),
	}}
	env := newTerminalGatewayMessagesEnv(t, group, upstream, account)

	reqBody := `{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	env.router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, env.usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.DetailSnapshot)
	requireRequestPreviewSnapshot(t, log.DetailSnapshot.RequestBody, reqBody)
	require.Contains(t, log.DetailSnapshot.ResponseBody, "Prompt is too long")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_prompt_too_long")
}

func TestGatewayHandler_MessagesPromptTooLongFallbackBillingRejectionCreatesOriginalFailedUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fallbackGroupID := int64(41)
	group := &service.Group{ID: 40, Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true, FallbackGroupIDOnInvalidRequest: &fallbackGroupID}
	account := &service.Account{ID: 140, Name: "antigravity", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}
	upstream := &openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_prompt_too_long_fallback"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"Prompt is too long"}}`)),
	}}
	env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
	billingCache := &promptTooLongFallbackBillingCache{}
	billingCacheService := service.NewBillingCacheService(billingCache, nil, nil, nil, nil, nil, &config.Config{Default: config.DefaultConfig{RateMultiplier: 1}}, nil)
	t.Cleanup(func() { billingCacheService.Stop() })
	env.handler.billingCacheService = billingCacheService

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	env.router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, env.usageRepo)
	require.NotNil(t, log)
	require.Len(t, env.usageRepo.created, 1)
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_prompt_too_long_fallback")
	require.Contains(t, log.DetailSnapshot.ResponseBody, "Prompt is too long")
	require.NotContains(t, log.DetailSnapshot.ResponseBody, "insufficient balance")
}

func TestGatewayHandler_MessagesPromptTooLongFallbackResolvesClaudeCodeOnlyGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initialID, intermediateID, finalID := int64(50), int64(51), int64(52)
	initialGroup := &service.Group{ID: initialID, Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true, FallbackGroupIDOnInvalidRequest: &intermediateID}
	intermediateGroup := &service.Group{ID: intermediateID, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true, ClaudeCodeOnly: true, FallbackGroupID: &finalID}
	finalGroup := &service.Group{ID: finalID, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	initialAccount := &service.Account{ID: 150, Name: "initial-antigravity", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, AccountGroups: []service.AccountGroup{{GroupID: initialID}}, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}
	finalAccount := &service.Account{ID: 152, Name: "final-gemini", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, AccountGroups: []service.AccountGroup{{GroupID: finalID}}, Credentials: map[string]any{"api_key": "key"}}

	t.Run("non Claude Code skips intermediate group and binds final Gemini session", func(t *testing.T) {
		cache := &geminiStickyGatewayCacheStub{}
		upstream := &openAIRetryTrackingHTTPUpstreamStub{responses: []*http.Response{
			{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Prompt is too long"}}`))},
			{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]}}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`))},
		}}
		env := newTerminalGatewayMessagesEnvWithGatewayCacheAndGroups(t, initialGroup, map[int64]*service.Group{initialID: initialGroup, intermediateID: intermediateGroup, finalID: finalGroup}, upstream, openAIChatCompletionsConcurrencyCacheStub{}, cache, initialAccount, finalAccount)
		env.handler.cfg.Gateway.Sticky.Gemini.Enabled = true

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-opus-4-6","max_tokens":16,"metadata":{"user_id":"fallback-session"},"messages":[{"role":"user","content":"hello"}]}`))
		req.Header.Set("Content-Type", "application/json")
		env.router().ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, []int64{initialAccount.ID, finalAccount.ID}, upstream.accountIDs)
		require.Len(t, upstream.requests, 2)
		require.Equal(t, "/v1beta/models/claude-opus-4-6:generateContent", upstream.requests[1].URL.Path)
		fallbackBody, err := io.ReadAll(upstream.requests[1].Body)
		require.NoError(t, err)
		require.True(t, gjson.GetBytes(fallbackBody, "contents").Exists())
		require.False(t, gjson.GetBytes(fallbackBody, "messages").Exists())
		require.Equal(t, "message", gjson.Get(rec.Body.String(), "type").String())
		require.Contains(t, env.accountRepo.groupIDs, finalID)
		require.NotEmpty(t, cache.getGroupIDs)
		for _, groupID := range cache.getGroupIDs {
			require.Equal(t, finalID, groupID)
		}
		require.NotEmpty(t, cache.setGroupIDs)
		for _, groupID := range cache.setGroupIDs {
			require.Equal(t, finalID, groupID)
		}
		require.Len(t, cache.setCalls, 1)
		for sessionKey, accountID := range cache.sessionBindings {
			require.True(t, strings.HasPrefix(sessionKey, "gemini:"))
			require.Equal(t, finalAccount.ID, accountID)
		}
	})

	t.Run("Claude Code keeps intermediate group", func(t *testing.T) {
		upstream := &openAIRetryTrackingHTTPUpstreamStub{responses: []*http.Response{
			{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Prompt is too long"}}`))},
			{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"type":"message","id":"msg_1","role":"assistant","content":[{"type":"text","text":"ok"}],"model":"claude-opus-4-6","usage":{"input_tokens":1,"output_tokens":1}}`))},
		}}
		env := newTerminalGatewayMessagesEnvWithGatewayCacheAndGroups(t, initialGroup, map[int64]*service.Group{initialID: initialGroup, intermediateID: intermediateGroup, finalID: finalGroup}, upstream, openAIChatCompletionsConcurrencyCacheStub{}, &geminiStickyGatewayCacheStub{}, initialAccount, &service.Account{ID: 151, Name: "intermediate-anthropic", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, AccountGroups: []service.AccountGroup{{GroupID: intermediateID}}, Credentials: map[string]any{"api_key": "key"}})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-opus-4-6","max_tokens":16,"system":[{"text":"You are Claude Code, Anthropic's official CLI for Claude."}],"metadata":{"user_id":"user_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa_account__session_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},"messages":[{"role":"user","content":"hello"}]}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "claude-cli/2.1.81")
		req.Header.Set("X-App", "claude-code")
		req.Header.Set("anthropic-beta", "message-batches-2024-09-24")
		req.Header.Set("anthropic-version", "2023-06-01")
		env.router().ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, []int64{initialAccount.ID, 151}, upstream.accountIDs)
		require.Contains(t, env.accountRepo.groupIDs, intermediateID)
		require.NotContains(t, env.accountRepo.groupIDs, finalID)
	})
}

func TestGatewayHandler_MessagesPromptTooLongFallbackMixedAntigravity429ClearsResolvedGeminiStickySession(t *testing.T) {
	testPromptTooLongFallbackMixedAntigravityStickyCleanup(t, http.StatusTooManyRequests, `{
		"error": {
			"status": "RESOURCE_EXHAUSTED",
			"details": [
				{"@type": "type.googleapis.com/google.rpc.ErrorInfo", "metadata": {"model": "gemini-3-flash"}, "reason": "RATE_LIMIT_EXCEEDED"},
				{"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "15s"}
			]
		}
	}`)
}

func TestGatewayHandler_MessagesPromptTooLongFallbackMixedAntigravity503ClearsResolvedGeminiStickySession(t *testing.T) {
	testPromptTooLongFallbackMixedAntigravityStickyCleanup(t, http.StatusServiceUnavailable, `{
		"error": {
			"status": "RESOURCE_EXHAUSTED",
			"details": [
				{"@type": "type.googleapis.com/google.rpc.ErrorInfo", "metadata": {"model": "gemini-3-flash"}, "reason": "RATE_LIMIT_EXCEEDED"},
				{"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "15s"}
			]
		}
	}`)
}

func testPromptTooLongFallbackMixedAntigravityStickyCleanup(t *testing.T, retryStatus int, retryBody string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	initialID, intermediateID, finalID := int64(60), int64(61), int64(62)
	initialGroup := &service.Group{ID: initialID, Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true, FallbackGroupIDOnInvalidRequest: &intermediateID}
	intermediateGroup := &service.Group{ID: intermediateID, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true, ClaudeCodeOnly: true, FallbackGroupID: &finalID}
	finalGroup := &service.Group{ID: finalID, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	initialAccount := &service.Account{ID: 160, Name: "initial-antigravity", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, AccountGroups: []service.AccountGroup{{GroupID: initialID}}, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}
	finalAccount := &service.Account{ID: 162, Name: "mixed-antigravity", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, AccountGroups: []service.AccountGroup{{GroupID: finalID}}, Credentials: map[string]any{"access_token": "token", "project_id": "project"}, Extra: map[string]any{"mixed_scheduling": true}}

	for _, tt := range []struct {
		name             string
		anthropicSticky  bool
		wantSessionClear bool
	}{
		{name: "Anthropic Sticky enabled clears resolved Gemini session", anthropicSticky: true, wantSessionClear: true},
		{name: "Anthropic Sticky disabled skips cleanup"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cache := &geminiStickyGatewayCacheStub{}
			upstream := &openAIRetryTrackingHTTPUpstreamStub{responses: []*http.Response{
				{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Prompt is too long"}}`))},
				{StatusCode: retryStatus, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(retryBody))},
			}}
			env := newTerminalGatewayMessagesEnvWithGatewayCacheAndGroups(t, initialGroup, map[int64]*service.Group{initialID: initialGroup, intermediateID: intermediateGroup, finalID: finalGroup}, upstream, openAIChatCompletionsConcurrencyCacheStub{}, cache, initialAccount, finalAccount)
			env.handler.cfg.Gateway.Sticky.Gemini.Enabled = true
			env.handler.cfg.Gateway.Sticky.Anthropic.Enabled = tt.anthropicSticky
			env.handler.maxAccountSwitches = 0

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-opus-4-6","max_tokens":16,"metadata":{"user_id":"{\"device_id\":\"sticky-device\",\"session_id\":\"invalid-fallback-session\"}"},"messages":[{"role":"user","content":"hello"}]}`))
			req.Header.Set("Content-Type", "application/json")
			env.router().ServeHTTP(rec, req)

			require.Equal(t, http.StatusBadGateway, rec.Code)
			require.Equal(t, []int64{initialAccount.ID, finalAccount.ID}, upstream.accountIDs)
			if !tt.wantSessionClear {
				require.Empty(t, cache.deleteCalls)
				return
			}
			require.Equal(t, []geminiStickyDeleteCall{{groupID: finalID, sessionKey: "gemini:invalid-fallback-session"}}, cache.deleteCalls)
		})
	}
}

func TestGatewayHandler_MessagesLocalErrorDoesNotCreateFailedUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, platform := range []string{service.PlatformGemini, service.PlatformAntigravity} {
		t.Run(platform, func(t *testing.T) {
			group := &service.Group{ID: 50, Platform: platform, Status: service.StatusActive, Hydrated: true}
			account := &service.Account{ID: 150, Name: "local", Platform: platform, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{}}
			if platform == service.PlatformAntigravity {
				account.Type = service.AccountTypeOAuth
				account.Credentials = map[string]any{"project_id": "project"}
			}
			env := newTerminalGatewayMessagesEnv(t, group, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")}, account)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`))
			req.Header.Set("Content-Type", "application/json")
			env.router().ServeHTTP(rec, req)

			require.Nil(t, env.usageRepo.lastLog)
		})
	}
}

func TestGatewayHandler_MessagesCanceledTransportErrorDoesNotCreateFailedUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 51, Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 151, Name: "antigravity-canceled", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}
	ctx, cancel := context.WithCancel(context.Background())
	env := newTerminalGatewayMessagesEnv(t, group, cancelingTerminalHTTPUpstream{cancel: cancel}, account)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	env.router().ServeHTTP(rec, req)

	require.ErrorIs(t, ctx.Err(), context.Canceled)
	require.Nil(t, env.usageRepo.lastLog)
}

func TestGatewayHandler_MessagesTransportErrorCreatesFailedUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 52, Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 152, Name: "antigravity-transport", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}
	env := newTerminalGatewayMessagesEnv(t, group, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("dial failed")}, account)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	env.router().ServeHTTP(rec, req)

	require.NotNil(t, env.usageRepo.lastLog)
}

func TestGatewayHandler_CompatibilityChatOrdinaryErrorsRequireUpstreamAttempt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("local credential error does not create failed usage", func(t *testing.T) {
		group := &service.Group{ID: 60, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 160, Name: "compat-chat-local", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{}}
		env := newTerminalGatewayMessagesEnv(t, group, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")}, account)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"hello"}]}`))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1/chat/completions", env.handler.ChatCompletions).ServeHTTP(rec, req)

		require.Nil(t, env.usageRepo.lastLog)
	})

	t.Run("transport error creates failed usage", func(t *testing.T) {
		group := &service.Group{ID: 61, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 161, Name: "compat-chat-transport", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
		env := newTerminalGatewayMessagesEnv(t, group, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("dial failed")}, account)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"hello"}]}`))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1/chat/completions", env.handler.ChatCompletions).ServeHTTP(rec, req)

		require.NotNil(t, env.usageRepo.lastLog)
	})
}

func TestGatewayHandler_CompatibilityResponsesTerminalUsageSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	requestBody := `{"model":"claude-opus-4-6","input":"hello","stream":true}`

	t.Run("local credential error does not create failed usage", func(t *testing.T) {
		group := &service.Group{ID: 70, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 170, Name: "responses-local", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{}}
		env := newTerminalGatewayMessagesEnv(t, group, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")}, account)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1/responses", env.handler.Responses).ServeHTTP(rec, req)

		require.Nil(t, env.usageRepo.lastLog)
	})

	t.Run("transport error creates failed usage", func(t *testing.T) {
		group := &service.Group{ID: 71, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 171, Name: "responses-transport", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
		env := newTerminalGatewayMessagesEnv(t, group, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("dial failed")}, account)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1/responses", env.handler.Responses).ServeHTTP(rec, req)

		require.NotNil(t, env.usageRepo.lastLog)
	})

	t.Run("scanner error after partial stream appends one failed terminal and failed usage", func(t *testing.T) {
		group := &service.Group{ID: 711, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 1711, Name: "responses-scanner", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
		var requestContext *gin.Context
		upstream := &partialWriteTransportHTTPUpstream{response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       terminalReadErrorBody{},
		}}
		upstream.writePartial = func() {
			requestContext.Header("Content-Type", "text/event-stream")
			_, _ = requestContext.Writer.WriteString("event: response.created\ndata: {\"type\":\"response.created\"}\n\n")
			requestContext.Writer.Flush()
		}
		env := newTerminalGatewayMessagesEnv(t, group, upstream, account)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1/responses", func(c *gin.Context) {
			requestContext = c
			env.handler.Responses(c)
		}).ServeHTTP(rec, req)

		responseBody := rec.Body.String()
		require.Equal(t, 1, strings.Count(responseBody, "event: response.failed\n"), responseBody)
		require.NotContains(t, responseBody, "response.completed")
		require.NotNil(t, env.usageRepo.lastLog)
		require.Len(t, env.usageRepo.created, 1)
	})

	t.Run("partial typed failover creates exactly one failed usage with diagnostic headers", func(t *testing.T) {
		group := &service.Group{ID: 72, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 172, Name: "responses-partial", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
		var requestContext *gin.Context
		upstream := &partialWriteTransportHTTPUpstream{response: &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"compat_responses_partial"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"expired upstream credential"}}`)),
		}}
		upstream.writePartial = func() {
			requestContext.Header("Content-Type", "text/event-stream")
			_, _ = requestContext.Writer.WriteString("data: {\"type\":\"response.output_text.delta\",\"delta\":\"Hello\"}\n\n")
			requestContext.Writer.Flush()
		}
		env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
		router := env.routerFor("/v1/responses", func(c *gin.Context) {
			requestContext = c
			env.handler.Responses(c)
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)

		responseBody := rec.Body.String()
		require.Equal(t, http.StatusOK, rec.Code, responseBody)
		require.Contains(t, responseBody, "Hello")
		require.Equal(t, 1, strings.Count(responseBody, "event: response.failed\n"))
		require.Equal(t, 1, strings.Count(responseBody, `"type":"response.failed"`))
		failedData := strings.SplitN(strings.SplitN(responseBody, "event: response.failed\ndata: ", 2)[1], "\n\n", 2)[0]
		require.True(t, gjson.Valid(failedData), failedData)
		require.Equal(t, "response.failed", gjson.Get(failedData, "type").String())
		require.Equal(t, "failed", gjson.Get(failedData, "response.status").String())
		require.NotNil(t, env.usageRepo.lastLog)
		require.Len(t, env.usageRepo.created, 1)
		require.Contains(t, env.usageRepo.lastLog.DetailSnapshot.ResponseHeaders, "X-Request-Id: compat_responses_partial")
	})

	t.Run("selection exhaustion after failover creates one failed usage", func(t *testing.T) {
		group := &service.Group{ID: 73, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 173, Name: "responses-selection", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
		upstream := &openAIRetryTrackingHTTPUpstreamStub{responses: []*http.Response{{
			StatusCode: http.StatusUnauthorized,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"compat_responses_selection"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"expired upstream credential"}}`)),
		}}}
		env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
		env.handler.maxAccountSwitches = 1

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1/responses", env.handler.Responses).ServeHTTP(rec, req)

		require.Equal(t, []int64{account.ID}, upstream.accountIDs)
		require.NotNil(t, env.usageRepo.lastLog)
		require.Len(t, env.usageRepo.created, 1)
		require.Contains(t, env.usageRepo.lastLog.DetailSnapshot.ResponseHeaders, "X-Request-Id: compat_responses_selection")
	})

	t.Run("zero max switches stops after first failover", func(t *testing.T) {
		group := &service.Group{ID: 74, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 174, Name: "responses-max-switch", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
		upstream := &openAIRetryTrackingHTTPUpstreamStub{responses: []*http.Response{{StatusCode: http.StatusUnauthorized, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"expired"}}`))}}}
		env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
		env.handler.maxAccountSwitches = 0

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1/responses", env.handler.Responses).ServeHTTP(rec, req)

		require.Equal(t, []int64{account.ID}, upstream.accountIDs)
		require.Len(t, env.usageRepo.created, 1)
	})
}

func TestOpenAIGatewayHandler_WSHTTPBridgeOrdinaryErrorUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		account        *service.Account
		upstream       service.HTTPUpstream
		wantUsage      bool
		wantErrorEvent bool
		wantImageUsage bool
	}{
		{
			name: "transport error creates one failed usage",
			account: &service.Account{
				ID: 201, Name: "ws-transport", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
				Status: service.StatusActive, Schedulable: true, Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test"}, Extra: map[string]any{"responses_websockets_v2_enabled": true},
			},
			upstream:  &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("dial failed")},
			wantUsage: true,
		},
		{
			name: "non-failover HTTP error creates one failed usage",
			account: &service.Account{
				ID: 202, Name: "ws-http-400", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
				Status: service.StatusActive, Schedulable: true, Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test"}, Extra: map[string]any{"responses_websockets_v2_enabled": true},
			},
			upstream: &openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad bridge request"}}`)),
			}},
			wantUsage: true,
		},
		{
			name: "local token error creates no failed usage",
			account: &service.Account{
				ID: 203, Name: "ws-local", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
				Status: service.StatusActive, Schedulable: true, Concurrency: 1,
				Credentials: map[string]any{}, Extra: map[string]any{"responses_websockets_v2_enabled": true},
			},
			upstream: &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")},
		},
		{
			name: "typed error event creates one failed usage",
			account: &service.Account{
				ID: 204, Name: "ws-typed-error", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
				Status: service.StatusActive, Schedulable: true, Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test"}, Extra: map[string]any{"responses_websockets_v2_enabled": true},
			},
			upstream: &openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader("data: {\"type\":\"error\",\"error\":{\"message\":\"typed bridge error\"}}\n\n")),
			}},
			wantUsage:      true,
			wantErrorEvent: true,
		},
		{
			name: "image result followed by error records normal usage once",
			account: &service.Account{
				ID: 205, Name: "ws-image-error", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
				Status: service.StatusActive, Schedulable: true, Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test"}, Extra: map[string]any{"responses_websockets_v2_enabled": true},
			},
			upstream: &openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body: io.NopCloser(strings.NewReader(
					"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ig_1\",\"type\":\"image_generation_call\",\"result\":\"image-data\"}}\n\n" +
						"data: {\"type\":\"error\",\"error\":{\"message\":\"error after image\"}}\n\n",
				)),
			}},
			wantUsage:      true,
			wantErrorEvent: true,
			wantImageUsage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := &service.Group{ID: 200, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
			env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIChatCompletionsAccountRepoStub{account: tt.account}, tt.upstream)
			env.usageRepo.created = make(chan *service.UsageLog, 2)
			done := make(chan struct{})
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(middleware.ContextKeyAPIKey), env.apiKey)
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: env.apiKey.UserID, Concurrency: env.apiKey.User.Concurrency})
				c.Next()
			})
			router.Use(middleware.UsageDetailCapture())
			router.GET("/v1/responses", func(c *gin.Context) {
				defer close(done)
				env.handler.ResponsesWebSocket(c)
			})
			server := httptest.NewServer(router)
			defer server.Close()

			dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
			conn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(server.URL, "http")+"/v1/responses", nil)
			cancelDial()
			require.NoError(t, err)
			defer func() { _ = conn.CloseNow() }()

			writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
			err = conn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.4","stream":true,"input":"hello"}`))
			cancelWrite()
			require.NoError(t, err)

			readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
			_, message, err := conn.Read(readCtx)
			cancelRead()
			if tt.wantImageUsage {
				require.NoError(t, err)
				require.Equal(t, "response.output_item.done", gjson.GetBytes(message, "type").String())
				readCtx, cancelRead = context.WithTimeout(context.Background(), 3*time.Second)
				_, message, err = conn.Read(readCtx)
				cancelRead()
			}
			if tt.wantErrorEvent {
				require.NoError(t, err)
				require.Equal(t, "error", gjson.GetBytes(message, "type").String())
				readCtx, cancelRead = context.WithTimeout(context.Background(), 3*time.Second)
				_, _, err = conn.Read(readCtx)
				cancelRead()
			}
			var closeErr coderws.CloseError
			require.ErrorAs(t, err, &closeErr, "ordinary bridge failure should produce one close frame, not a client error event followed by close")
			require.Equal(t, coderws.StatusInternalError, closeErr.Code)
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("websocket handler did not return")
			}

			if tt.wantUsage {
				require.NotNil(t, env.usageRepo.lastLog)
				require.Len(t, env.usageRepo.created, 1)
				require.Equal(t, tt.account.ID, env.usageRepo.lastLog.AccountID)
				if tt.wantImageUsage {
					require.Equal(t, 1, env.usageRepo.lastLog.ImageCount)
				}
			} else {
				require.Nil(t, env.usageRepo.lastLog)
				require.Empty(t, env.usageRepo.created)
			}
		})
	}
}

func TestGatewayHandler_GeminiNativeOrdinaryErrorUsageSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	requestBody := `{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`
	route := "/v1beta/models/gemini-2.5-flash:generateContent"

	run := func(t *testing.T, groupID, accountID int64, account *service.Account, upstream service.HTTPUpstream, requestContext context.Context) (*terminalGatewayMessagesEnv, *httptest.ResponseRecorder) {
		t.Helper()
		group := &service.Group{ID: groupID, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
		account.ID = accountID
		if account.Platform == "" {
			account.Platform = service.PlatformGemini
		}
		account.Status = service.StatusActive
		account.Schedulable = true
		account.Concurrency = 1
		env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, route, strings.NewReader(requestBody)).WithContext(requestContext)
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1beta/models/*modelAction", env.handler.GeminiV1BetaModels).ServeHTTP(rec, req)
		return env, rec
	}

	t.Run("service-written local credential error is not duplicated", func(t *testing.T) {
		env, rec := run(t, 80, 180, &service.Account{Name: "gemini-local", Type: service.AccountTypeAPIKey, Credentials: map[string]any{}}, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")}, context.Background())
		require.Equal(t, http.StatusBadGateway, rec.Code)
		require.Contains(t, rec.Body.String(), "gemini api_key not configured")
		require.NotContains(t, rec.Body.String(), "Upstream request failed")
		require.Nil(t, env.usageRepo.lastLog)
	})

	t.Run("antigravity local token error gets generic non-2xx without failed usage", func(t *testing.T) {
		account := &service.Account{
			Name:        "antigravity-local-token",
			Platform:    service.PlatformAntigravity,
			Type:        service.AccountTypeOAuth,
			Credentials: map[string]any{"project_id": "project"},
			Extra:       map[string]any{"mixed_scheduling": true},
		}
		env, rec := run(t, 85, 185, account, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")}, context.Background())
		require.Equal(t, http.StatusBadGateway, rec.Code)
		require.JSONEq(t, `{"error":{"code":502,"message":"Upstream request failed","status":"INTERNAL"}}`, rec.Body.String())
		require.Nil(t, env.usageRepo.lastLog)
	})

	t.Run("local request build error does not create failed usage", func(t *testing.T) {
		env, rec := run(t, 81, 181, &service.Account{Name: "gemini-build", Type: service.AccountTypeAPIKey, Credentials: map[string]any{"api_key": "key", "base_url": "://invalid"}}, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")}, context.Background())
		require.Equal(t, http.StatusBadGateway, rec.Code)
		require.Contains(t, rec.Body.String(), "invalid base_url")
		require.NotContains(t, rec.Body.String(), "Upstream request failed")
		require.Nil(t, env.usageRepo.lastLog)
	})

	t.Run("canceled response read does not create failed usage", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		upstream := directTerminalHTTPUpstream{response: &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: terminalReadErrorBody{cancel: cancel}}}
		env, _ := run(t, 82, 182, &service.Account{Name: "gemini-cancel", Type: service.AccountTypeAPIKey, Credentials: map[string]any{"api_key": "key"}}, upstream, ctx)
		require.ErrorIs(t, ctx.Err(), context.Canceled)
		require.Nil(t, env.usageRepo.lastLog)
	})

	t.Run("transport response read error gets generic non-2xx and creates failed usage", func(t *testing.T) {
		upstream := directTerminalHTTPUpstream{response: &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: terminalReadErrorBody{}}}
		env, rec := run(t, 83, 183, &service.Account{Name: "gemini-transport", Type: service.AccountTypeAPIKey, Credentials: map[string]any{"api_key": "key"}}, upstream, context.Background())
		require.Equal(t, http.StatusBadGateway, rec.Code)
		require.JSONEq(t, `{"error":{"code":502,"message":"Upstream request failed","status":"INTERNAL"}}`, rec.Body.String())
		require.NotNil(t, env.usageRepo.lastLog)
	})

	t.Run("zero-byte stream read error replaces stale SSE content type", func(t *testing.T) {
		group := &service.Group{ID: 87, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 187, Name: "gemini-empty-stream", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "key"}}
		upstream := directTerminalHTTPUpstream{response: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: terminalReadErrorBody{}}}
		env := newTerminalGatewayMessagesEnv(t, group, upstream, account)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:streamGenerateContent", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1beta/models/*modelAction", env.handler.GeminiV1BetaModels).ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadGateway, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
		require.NotContains(t, rec.Header().Get("Content-Type"), "text/event-stream")
		require.JSONEq(t, `{"error":{"code":502,"message":"Upstream request failed","status":"INTERNAL"}}`, rec.Body.String())
		require.NotNil(t, env.usageRepo.lastLog)
	})

	t.Run("partial streaming response is not followed by generic error", func(t *testing.T) {
		group := &service.Group{ID: 86, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 186, Name: "gemini-partial", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "key"}}
		body := &terminalPartialReadErrorBody{data: []byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"partial\"}]}}]}\n\n")}
		upstream := directTerminalHTTPUpstream{response: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: body}}
		env := newTerminalGatewayMessagesEnv(t, group, upstream, account)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:streamGenerateContent", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1beta/models/*modelAction", env.handler.GeminiV1BetaModels).ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "text/event-stream")
		require.Contains(t, rec.Body.String(), "partial")
		require.NotContains(t, rec.Body.String(), "Upstream request failed")
		require.NotNil(t, env.usageRepo.lastLog)
	})
}

func TestGatewayHandler_GeminiNativeAttemptSignalResetsAcrossAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 84, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	accountA := &service.Account{ID: 184, Name: "gemini-attempted", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "key"}}
	accountB := &service.Account{ID: 185, Name: "gemini-local", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{}}
	upstream := &openAIRetryTrackingHTTPUpstreamStub{responses: []*http.Response{{
		StatusCode: http.StatusUnauthorized,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"expired"}}`)),
	}}}
	env := newTerminalGatewayMessagesEnv(t, group, upstream, accountA, accountB)
	env.handler.maxAccountSwitchesGemini = 1
	var requestContext *gin.Context
	router := env.routerFor("/v1beta/models/*modelAction", func(c *gin.Context) {
		requestContext = c
		env.handler.GeminiV1BetaModels(c)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", strings.NewReader(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, []int64{accountA.ID}, upstream.accountIDs)
	require.False(t, service.HasOpsUpstreamAttempted(requestContext))
	require.Nil(t, env.usageRepo.lastLog)
}

func TestGatewayHandler_CompatibilityChatGeminiBranchUsageSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	requestBody := `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hello"}]}`

	t.Run("local credential error does not create failed usage", func(t *testing.T) {
		group := &service.Group{ID: 90, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 190, Name: "gemini-chat-local", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{}}
		env := newTerminalGatewayMessagesEnv(t, group, &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("must not be called")}, account)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1/chat/completions", env.handler.ChatCompletions).ServeHTTP(rec, req)

		require.Nil(t, env.usageRepo.lastLog)
	})

	t.Run("transport response read error creates failed usage", func(t *testing.T) {
		group := &service.Group{ID: 91, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 191, Name: "gemini-chat-transport", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "key"}}
		upstream := directTerminalHTTPUpstream{response: &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: terminalReadErrorBody{}}}
		env := newTerminalGatewayMessagesEnv(t, group, upstream, account)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1/chat/completions", env.handler.ChatCompletions).ServeHTTP(rec, req)

		require.NotNil(t, env.usageRepo.lastLog)
	})
}
