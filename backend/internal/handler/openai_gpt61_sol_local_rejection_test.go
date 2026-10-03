package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGPT61SolFinalPassthroughHTTPRejectionDoesNotPenalizeAccount(t *testing.T) {
	for _, route := range []string{"chat", "messages", "responses"} {
		for _, passthrough := range []bool{false, true} {
			for _, tc := range []struct {
				model   string
				effort  string
				blocked bool
			}{
				{"gpt-6.1-sol", "none", true},
				{"gpt-6.1-sol", "minimal", true},
				{"gpt-6.1-sol", "high", false},
				{"gpt-6-sol", "none", false},
			} {
				name := route + "/normal/" + tc.model + "/" + tc.effort
				if passthrough {
					name = route + "/passthrough/" + tc.model + "/" + tc.effort
				}
				t.Run(name, func(t *testing.T) {
					group := &service.Group{ID: 12, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
					account := &service.Account{
						ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
						Status: service.StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{12},
						Credentials: map[string]any{"api_key": "local-fixture-only", "model_mapping": map[string]any{"public": "gpt-6-sol"}},
						Extra: map[string]any{
							"openai_passthrough":         passthrough,
							"use_responses_api":          true,
							"passthrough_fields_enabled": true,
							"passthrough_field_rules": []service.PassthroughFieldRule{
								{Target: "body", Mode: "inject", Key: "model", Value: tc.model},
								{Target: "body", Mode: "inject", Key: "reasoning.effort", Value: tc.effort},
							},
						},
					}
					upstream := &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("local fake must not be reached for final policy refusal")}
					if !tc.blocked {
						upstream.err = nil
						response := `data: {"type":"response.completed","response":{"id":"resp_policy","object":"response","status":"completed","model":"MODEL","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}}` + "\n\n"
						upstream.response = &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(strings.ReplaceAll(response, "MODEL", tc.model)))}
					}
					env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)
					env.handler.cfg.Gateway.OpenAIWS.Enabled = false
					env.handler.cfg.Gateway.OpenAIScheduler.StickyEscapeEnabled = true
					env.handler.cfg.Gateway.OpenAIScheduler.StickyEscapeErrorRate = 0.1
					path, body, forward := "/v1/chat/completions", `{"model":"public","messages":[{"role":"user","content":"hi"}],"reasoning_effort":"high"}`, env.handler.ChatCompletions
					switch route {
					case "messages":
						path, body, forward = "/v1/messages", `{"model":"public","max_tokens":1,"messages":[{"role":"user","content":"hi"}],"thinking":{"type":"adaptive"}}`, env.handler.Messages
					case "responses":
						path, body, forward = "/v1/responses", `{"model":"public","input":"hi","reasoning":{"effort":"high"}}`, env.handler.Responses
					}
					var limited, attempted bool
					var upstreamEvents any
					observe := func(c *gin.Context) {
						forward(c)
						limited = service.HasOpsClientBusinessLimited(c)
						attempted = service.HasOpsUpstreamAttempted(c)
						upstreamEvents, _ = c.Get(service.OpsUpstreamErrorsKey)
					}
					recorder := httptest.NewRecorder()
					request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
					request.Header.Set("Content-Type", "application/json")
					env.router(path, observe).ServeHTTP(recorder, request)

					if !tc.blocked {
						assert.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
						assert.True(t, gjson.Valid(recorder.Body.String()))
						assert.Equal(t, tc.model, gjson.GetBytes(upstream.requestBody, "model").String())
						assert.Equal(t, tc.effort, gjson.GetBytes(upstream.requestBody, "reasoning.effort").String())
						assert.False(t, limited)
						assert.True(t, attempted)
						assert.Empty(t, upstreamEvents)
						return
					}
					assert.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
					assert.True(t, gjson.Valid(recorder.Body.String()), "local refusal must contain one JSON response")
					assert.Contains(t, recorder.Body.String(), "gpt-6.1-sol")
					if route == "messages" {
						assert.Equal(t, "error", gjson.Get(recorder.Body.String(), "type").String())
					}
					assert.Empty(t, upstream.requestBody, "final passthrough validation must precede outbound IO")
					assert.True(t, limited, "final local policy refusal must carry the business marker")
					assert.False(t, attempted)
					assert.Empty(t, upstreamEvents, "local policy refusal must not record a transport error")
					assert.Zero(t, env.handler.gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount, "local refusal must not penalize the selected account")
				})
			}
		}
	}
}

func newGPT61SolFinalOverrideEnv(t *testing.T, accountType string) (*terminalUsageOpenAIEnv, *openAIChatCompletionsHTTPUpstreamStub, *service.Account) {
	t.Helper()
	group := &service.Group{ID: 12, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true, AllowImageGeneration: true}
	account := &service.Account{
		ID: 1, Platform: service.PlatformOpenAI, Type: accountType, Status: service.StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{12},
		Credentials: map[string]any{"api_key": "local-fixture-only", "access_token": "local-fixture-only", "model_mapping": map[string]any{"public": "gpt-6-sol", "gpt-image-2": "gpt-image-2"}},
		Extra: map[string]any{"passthrough_fields_enabled": true, "passthrough_field_rules": []service.PassthroughFieldRule{
			{Target: "body", Mode: "inject", Key: "model", Value: "gpt-6.1-sol"},
			{Target: "body", Mode: "inject", Key: "reasoning.effort", Value: "none"},
		}},
	}
	upstream := &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("local fake must not be reached for final policy refusal")}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)
	env.handler.cfg.Gateway.OpenAIWS.Enabled = false
	env.handler.cfg.Gateway.OpenAIWS.SchedulerMode = "weighted"
	env.handler.cfg.Gateway.OpenAIScheduler.StickyEscapeEnabled = true
	env.handler.cfg.Gateway.OpenAIScheduler.StickyEscapeErrorRate = 0.1
	return env, upstream, account
}

func TestGPT61SolFinalOverrideAfterConcurrencyPing(t *testing.T) {
	for _, route := range []string{"chat", "messages", "responses"} {
		t.Run(route, func(t *testing.T) {
			env, upstream, _ := newGPT61SolFinalOverrideEnv(t, service.AccountTypeAPIKey)
			cache := &blockingResponsesUserSlotCache{waiting: make(chan struct{}), release: make(chan struct{})}
			env.handler.concurrencyHelper = NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatComment, time.Millisecond)
			path, body, forward := "/v1/chat/completions", `{"model":"public","stream":true,"messages":[{"role":"user","content":"hi"}],"reasoning_effort":"high"}`, env.handler.ChatCompletions
			switch route {
			case "messages":
				path, body, forward = "/v1/messages", `{"model":"public","max_tokens":1,"stream":true,"messages":[{"role":"user","content":"hi"}],"thinking":{"type":"adaptive"}}`, env.handler.Messages
			case "responses":
				path, body, forward = "/v1/responses", `{"model":"public","stream":true,"input":"hi","reasoning":{"effort":"high"}}`, env.handler.Responses
			}
			var limited, attempted bool
			var upstreamEvents any
			router := env.router(path, func(c *gin.Context) {
				forward(c)
				limited, attempted = service.HasOpsClientBusinessLimited(c), service.HasOpsUpstreamAttempted(c)
				upstreamEvents, _ = c.Get(service.OpsUpstreamErrorsKey)
			})
			recorder := httptest.NewRecorder()
			done := make(chan struct{})
			go func() {
				request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(recorder, request)
				close(done)
			}()
			waitGatewayReplaySignal(t, cache.waiting, "Sol user concurrency wait")
			time.Sleep(20 * time.Millisecond)
			close(cache.release)
			waitGatewayReplaySignal(t, done, "Sol local refusal after heartbeat")
			assert.Equal(t, http.StatusOK, recorder.Code, "heartbeat must already have committed the response")
			assert.Contains(t, recorder.Body.String(), "gpt-6.1-sol")
			for _, line := range strings.Split(recorder.Body.String(), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					assert.True(t, strings.HasPrefix(line, ":") || strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "data:"), "invalid SSE line: %s", line)
				}
			}
			switch route {
			case "responses":
				assert.Equal(t, 1, strings.Count(recorder.Body.String(), "event: response.failed\n"))
			case "messages":
				assert.Contains(t, recorder.Body.String(), `"type":"error"`)
			default:
				assert.Contains(t, recorder.Body.String(), `data: {"error":`)
			}
			assert.Empty(t, upstream.requestBody)
			assert.True(t, limited)
			assert.False(t, attempted)
			assert.Empty(t, upstreamEvents)
			assert.Zero(t, env.handler.gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount)
		})
	}
}

func TestGPT61SolFinalOverrideAuxiliaryHTTPRejectionDoesNotPenalizeAccount(t *testing.T) {
	for _, route := range []string{"search", "images"} {
		t.Run(route, func(t *testing.T) {
			accountType := service.AccountTypeAPIKey
			if route == "images" {
				accountType = service.AccountTypeOAuth
			}
			env, upstream, _ := newGPT61SolFinalOverrideEnv(t, accountType)
			path, body, forward := "/v1/alpha/search", `{"model":"public","commands":{}}`, env.handler.AlphaSearch
			if route == "images" {
				path, body, forward = "/v1/images/generations", `{"model":"gpt-image-2","prompt":"hi"}`, env.handler.Images
			}
			var limited, attempted bool
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			env.router(path, func(c *gin.Context) {
				forward(c)
				limited, attempted = service.HasOpsClientBusinessLimited(c), service.HasOpsUpstreamAttempted(c)
			}).ServeHTTP(recorder, request)
			assert.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
			assert.True(t, gjson.Valid(recorder.Body.String()))
			assert.Contains(t, recorder.Body.String(), "gpt-6.1-sol")
			assert.Empty(t, upstream.requestBody)
			assert.True(t, limited)
			assert.False(t, attempted)
			assert.Zero(t, env.handler.gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount)
		})
	}
}

func TestGPT61SolFinalOverrideWebSocketHTTPBridgeUsesPolicyClose(t *testing.T) {
	env, upstream, account := newGPT61SolFinalOverrideEnv(t, service.AccountTypeAPIKey)
	env.handler.cfg.Gateway.OpenAIWS.Enabled = true
	env.handler.cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	env.handler.cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	env.handler.cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
	account.Extra["openai_apikey_responses_websockets_v2_mode"] = service.OpenAIWSIngressModeHTTPBridge
	path := "/v1/responses"
	router := env.router(path, env.handler.Responses)
	done := make(chan struct{})
	router.GET(path, func(c *gin.Context) { defer close(done); env.handler.ResponsesWebSocket(c) })
	server := httptest.NewServer(router)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+path, nil)
	require.NoError(t, err)
	defer func() { _ = conn.CloseNow() }()
	require.NoError(t, conn.Write(ctx, coderws.MessageText, []byte(`{"type":"response.create","model":"public","input":"hi","reasoning":{"effort":"high"}}`)))
	_, _, err = conn.Read(ctx)
	var closeErr coderws.CloseError
	require.ErrorAs(t, err, &closeErr)
	assert.Equal(t, coderws.StatusPolicyViolation, closeErr.Code)
	assert.Contains(t, closeErr.Reason, "gpt-6.1-sol")
	waitGatewayReplaySignal(t, done, "Sol WebSocket policy refusal")
	assert.Empty(t, upstream.requestBody)
	assert.Zero(t, env.handler.gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount)
}

func TestGPT61SolEarlyIntentRefusalAfterConcurrencyPingUsesSSEFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 12, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{12},
		Credentials: map[string]any{"api_key": "local-fixture-only", "base_url": "https://api.openai.com", "model_mapping": map[string]any{"public": "gpt-6.1-sol"}},
	}
	upstream := &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("local fake must not be reached for policy refusal")}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)
	env.handler.cfg.Gateway.OpenAIWS.Enabled = false
	cache := &blockingResponsesUserSlotCache{waiting: make(chan struct{}), release: make(chan struct{})}
	env.handler.concurrencyHelper = NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatComment, time.Millisecond)
	path := "/v1/responses"
	body := `{"model":"public","stream":true,"input":"hi","reasoning":{"effort":"none"}}`
	var limited, attempted bool
	router := env.router(path, func(c *gin.Context) {
		env.handler.Responses(c)
		limited, attempted = service.HasOpsClientBusinessLimited(c), service.HasOpsUpstreamAttempted(c)
	})
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		close(done)
	}()
	waitGatewayReplaySignal(t, cache.waiting, "early guard user concurrency wait")
	time.Sleep(20 * time.Millisecond)
	close(cache.release)
	waitGatewayReplaySignal(t, done, "early guard local refusal after heartbeat")

	response := recorder.Body.String()
	assert.Equal(t, http.StatusOK, recorder.Code, "concurrency heartbeat must have committed the response")
	assert.Contains(t, response, "gpt-6.1-sol", "client must see the local policy reason")
	assert.NotContains(t, response, "Upstream request failed", "no upstream attempt happened, so the generic fallback is misleading")
	assert.Equal(t, 1, strings.Count(response, "event: response.failed\n"), "exactly one terminal event")
	assert.Equal(t, 1, strings.Count(response, "data: {"), "terminal event must be the only payload frame")
	for _, line := range strings.Split(response, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			assert.True(t, strings.HasPrefix(line, ":") || strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "data:"), "invalid SSE line: %s", line)
		}
	}
	assert.Empty(t, upstream.requestBody)
	assert.True(t, limited)
	assert.False(t, attempted)
	assert.Zero(t, env.handler.gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount)
}

func TestGPT61SolFinalMappedHTTPRejectionDoesNotPenalizeAccount(t *testing.T) {
	for _, route := range []string{"chat", "messages", "responses"} {
		for _, upstreamAuthFailure := range []bool{false, true} {
			name := route + "/local-model-policy"
			if upstreamAuthFailure {
				name = route + "/real-upstream-auth-failure"
			}
			t.Run(name, func(t *testing.T) {
				group := &service.Group{ID: 12, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
				account := &service.Account{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{12}, Credentials: map[string]any{"api_key": "local-fixture-only", "base_url": "https://api.openai.com", "model_mapping": map[string]any{"public": "gpt-6.1-sol"}}}
				upstream := &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("local fake must not be reached for policy refusal")}
				if upstreamAuthFailure {
					upstream.err = nil
					upstream.response = &http.Response{StatusCode: http.StatusUnauthorized, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_api_key","message":"local fake authentication failure"}}`))}
				}
				env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)
				env.handler.cfg.Gateway.OpenAIWS.APIKeyEnabled = false
				env.handler.cfg.Gateway.OpenAIWS.SchedulerMode = "weighted"
				env.handler.cfg.Gateway.OpenAIScheduler.StickyEscapeEnabled = true
				env.handler.cfg.Gateway.OpenAIScheduler.StickyEscapeErrorRate = 0.1
				path, body, forward := "/v1/chat/completions", `{"model":"public","messages":[{"role":"user","content":"hi"}],"reasoning_effort":"none"}`, env.handler.ChatCompletions
				switch route {
				case "messages":
					path, body, forward = "/v1/messages", `{"model":"public","max_tokens":1,"messages":[{"role":"user","content":"hi"}],"thinking":{"type":"disabled"}}`, env.handler.Messages
				case "responses":
					path, body, forward = "/v1/responses", `{"model":"public","input":"hi","reasoning":{"effort":"none"}}`, env.handler.Responses
				}
				if upstreamAuthFailure {
					body = strings.ReplaceAll(body, `"none"`, `"high"`)
					body = strings.ReplaceAll(body, `"disabled"`, `"adaptive"`)
				}
				var limited bool
				observe := func(c *gin.Context) { forward(c); limited = service.HasOpsClientBusinessLimited(c) }
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				env.router(path, observe).ServeHTTP(recorder, request)
				stats := env.handler.gatewayService.SnapshotOpenAIAccountSchedulerMetrics()
				if upstreamAuthFailure {
					assert.NotEmpty(t, upstream.requestBody, "fake upstream must observe the accepted request")
					assert.False(t, limited)
					assert.EqualValues(t, 1, stats.RuntimeStatsAccountCount, "actual auth failure must still reach account runtime health")
				} else {
					assert.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
					assert.Contains(t, recorder.Body.String(), "gpt-6.1-sol")
					assert.Empty(t, upstream.requestBody, "final account mapping validation precedes outbound IO")
					assert.True(t, limited, "local mapped-model policy must carry the business marker")
					assert.Zero(t, stats.RuntimeStatsAccountCount, "local refusal must not create a failed account runtime sample")
				}
			})
		}
	}
}
