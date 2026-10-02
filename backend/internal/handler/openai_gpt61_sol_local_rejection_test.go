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
)

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
