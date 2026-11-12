//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	coderws "github.com/coder/websocket"
	"github.com/tidwall/gjson"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestInputContextProtocolRejections(t *testing.T) {
	for _, tc := range []struct {
		protocol string
		stream   bool
		status   int
		want     string
	}{
		{"messages", false, 400, `"type":"error"`}, {"chat", true, 400, `"code":"context_length_exceeded"`}, {"responses", false, 400, `"param":"input"`}, {"responses", true, 200, `event: response.failed`}, {"gemini", true, 400, `"status":"INVALID_ARGUMENT"`},
	} {
		t.Run(tc.protocol+tc.want, func(t *testing.T) {
			settings := service.NewSettingService(&contentModerationHandlerSettingRepo{values: map[string]string{service.SettingKeyInputContextLimit: `{"enabled":true,"group_limits":{"1":1}}`}}, nil)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest("POST", "/v1/"+tc.protocol, strings.NewReader(""))
			id := int64(1)
			body, _ := json.Marshal(map[string]any{"model": "gpt-4o", "input": "hello world too many tokens", "messages": []any{map[string]any{"content": "hello world too many tokens"}}, "contents": []any{map[string]any{"parts": []any{map[string]any{"text": "hello world too many tokens"}}}}, "stream": tc.stream})
			require.True(t, rejectInputContext(c, settings, &service.APIKey{GroupID: &id}, "gpt-4o", body, tc.protocol))
			require.Equal(t, tc.status, recorder.Code)
			require.Contains(t, recorder.Body.String(), tc.want)
			require.True(t, service.HasOpsClientBusinessLimited(c))
		})
	}
}

func TestInputContextWebSocketTurns(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		for _, blockedTurn := range []int{1, 2} {
			t.Run(fmt.Sprintf("passthrough=%t/turn=%d", passthrough, blockedTurn), func(t *testing.T) {
				settings := service.NewSettingService(&contentModerationHandlerSettingRepo{values: map[string]string{service.SettingKeyInputContextLimit: `{"enabled":true,"group_limits":{"2":10}}`}}, nil)
				cache := &concurrencyCacheMock{acquireUserSlotFn: func(context.Context, int64, int, string) (bool, error) { return true, nil }, acquireUserGroupSlotFn: func(context.Context, int64, int64, int, string) (bool, error) { return true, nil }, acquireAccountSlotFn: func(context.Context, int64, int, string) (bool, error) { return true, nil }}
				usage := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 4)}
				env := newOpenAIWSRegressionEnv(t, cache, openAIWSRegressionEnvOptions{Passthrough: passthrough, CaptureUpstreamMessages: true, UsageLogRepo: usage, InputContextSettings: settings})
				defer env.Close()
				env.apiKey.Group.AllowMessagesDispatch = true
				env.apiKey.Group.Platform = service.PlatformOpenAI
				env.apiKey.Group.Status = service.StatusActive
				env.apiKey.Group.Hydrated = true
				conn := env.dial(t)
				defer conn.CloseNow()
				if blockedTurn == 2 {
					env.writeMessage(t, conn, `{"type":"response.create","model":"gpt-5.1","input":"hi"}`)
					for {
						if gjson.GetBytes(env.readMessage(t, conn), "type").String() == "response.completed" {
							break
						}
					}
					select {
					case <-env.upstreamMessages:
					case <-time.After(5 * time.Second):
						t.Fatal("missing first upstream request")
					}
					select {
					case <-usage.created:
					case <-time.After(5 * time.Second):
						t.Fatal("missing first usage")
					}
				}
				payload := `{"type":"response.create","model":"gpt-5.1","input":"` + strings.Repeat("long input ", 100) + `"}`
				env.writeMessage(t, conn, payload)
				event := env.readMessage(t, conn)
				require.Equal(t, "response.failed", gjson.GetBytes(event, "type").String(), string(event))
				require.Equal(t, "context_length_exceeded", gjson.GetBytes(event, "response.error.code").String())
				require.True(t, gjson.GetBytes(event, "sequence_number").Exists())
				env.readCloseError(t, conn, coderws.StatusPolicyViolation)
				env.waitRequestDone(t)
				select {
				case msg := <-env.upstreamMessages:
					t.Fatalf("rejected request reached upstream: %s", msg)
				default:
				}
				select {
				case log := <-usage.created:
					t.Fatalf("rejected request billed: %+v", log)
				default:
				}
			})
		}
	}
}

func TestInputContextHTTPHandlers(t *testing.T) {
	for _, kind := range []string{"messages", "responses", "chat", "gemini", "openai-messages", "openai-responses", "openai-chat"} {
		t.Run(kind, func(t *testing.T) {
			settings := service.NewSettingService(&contentModerationHandlerSettingRepo{values: map[string]string{service.SettingKeyInputContextLimit: `{"enabled":true,"group_limits":{"2":1}}`}}, nil)
			group := &service.Group{ID: 2, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
			gateway := newTerminalGatewayMessagesEnv(t, group, nil)
			gateway.handler.settingService = settings
			body := `{"model":"gpt-5.1","messages":[{"role":"user","content":"hello world too many input tokens"}],"input":"hello world too many input tokens","contents":[{"role":"user","parts":[{"text":"hello world too many input tokens"}]}]}`
			path := "/v1/messages"
			var route string
			var fn gin.HandlerFunc
			var router *gin.Engine
			switch kind {
			case "messages":
				fn = gateway.handler.Messages
			case "responses":
				fn = gateway.handler.Responses
				path = "/v1/responses"
			case "chat":
				fn = gateway.handler.ChatCompletions
				path = "/v1/chat/completions"
			case "gemini":
				group.Platform = service.PlatformGemini
				fn = gateway.handler.GeminiV1BetaModels
				route = "/v1beta/models/*modelAction"
				path = "/v1beta/models/gemini-2.5-pro:generateContent"
			default:
				env := newOpenAIWSRegressionEnv(t, &concurrencyCacheMock{}, openAIWSRegressionEnvOptions{SkipUpstream: true, InputContextSettings: settings})
				defer env.Close()
				env.apiKey.Group.AllowMessagesDispatch = true
				env.apiKey.Group.Platform = service.PlatformOpenAI
				env.apiKey.Group.Status = service.StatusActive
				env.apiKey.Group.Hydrated = true
				switch kind {
				case "openai-messages":
					fn = env.handler.Messages
				case "openai-responses":
					fn = env.handler.Responses
					path = "/v1/responses"
				case "openai-chat":
					fn = env.handler.ChatCompletions
					path = "/v1/chat/completions"
				}
				router = gin.New()
				router.Use(func(c *gin.Context) {
					c.Set(string(middleware.ContextKeyAPIKey), env.apiKey)
					c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1, Concurrency: 1})
					c.Next()
				})
				router.POST(path, fn)
			}
			if router == nil {
				if route == "" {
					route = path
				}
				router = gateway.routerFor(route, fn)
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest("POST", path, strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)
			require.Equal(t, 400, recorder.Code, recorder.Body.String())
			require.Contains(t, recorder.Body.String(), "token")
			select {
			case log := <-gateway.usageRepo.created:
				t.Fatalf("rejected input billed %+v", log)
			default:
			}
		})
	}
}

func TestInputContextFallbackTargetRechecked(t *testing.T) {
	initialID, intermediateID, finalID := int64(50), int64(51), int64(52)
	initial := &service.Group{ID: initialID, Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true, FallbackGroupIDOnInvalidRequest: &intermediateID}
	intermediate := &service.Group{ID: intermediateID, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true, ClaudeCodeOnly: true, FallbackGroupID: &finalID}
	final := &service.Group{ID: finalID, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 150, Name: "initial", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, AccountGroups: []service.AccountGroup{{GroupID: initialID}}, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}
	upstream := &openAIRetryTrackingHTTPUpstreamStub{responses: []*http.Response{{StatusCode: 400, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Prompt is too long"}}`))}}}
	env := newTerminalGatewayMessagesEnvWithGatewayCacheAndGroups(t, initial, map[int64]*service.Group{initialID: initial, intermediateID: intermediate, finalID: final}, upstream, openAIChatCompletionsConcurrencyCacheStub{}, &geminiStickyGatewayCacheStub{}, account)
	env.handler.settingService = service.NewSettingService(&contentModerationHandlerSettingRepo{values: map[string]string{service.SettingKeyInputContextLimit: `{"enabled":true,"group_limits":{"50":1000,"51":0,"52":1}}`}}, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(`{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"hello world more input"}]}`))
	req.Header.Set("Content-Type", "application/json")
	env.router().ServeHTTP(rec, req)
	require.Equal(t, 400, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "gateway estimated input limit")
	require.Equal(t, []int64{150}, upstream.accountIDs)
	select {
	case log := <-env.usageRepo.created:
		t.Fatalf("policy rejection recorded generated usage %+v", log)
	default:
	}
}
func TestInputContextOpsSuppressesResidualUpstream(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	service.MarkOpsClientBusinessLimited(c, "input_context_limit_exceeded")
	id := int64(7)
	status := 502
	message := "old upstream failure"
	entry := &service.OpsInsertErrorLogInput{AccountID: &id, UpstreamModel: "old", UpstreamEndpoint: "old", UpstreamStatusCode: &status, UpstreamErrorMessage: &message}
	suppressOpsUpstreamAttributionForLocalModelConfiguration(c, entry)
	require.Nil(t, entry.AccountID)
	require.Nil(t, entry.UpstreamStatusCode)
	require.Nil(t, entry.UpstreamErrorMessage)
	require.Empty(t, entry.UpstreamModel)
}
