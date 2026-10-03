package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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
