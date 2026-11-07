//go:build unit

package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type codexRoutingSlotCache struct {
	profitCountingConcurrencyCache
	account  *service.Account
	attempts atomic.Int64
	wait     bool
}

func (c *codexRoutingSlotCache) AcquireAccountSlot(context.Context, int64, int, string) (bool, error) {
	n := c.attempts.Add(1)
	if c.wait && n == 1 {
		return false, nil
	}
	c.account.Extra["codex_cli_only_route_only"] = true
	return true, nil
}

func TestCodexRoutingHTTPEntries(t *testing.T) {
	for _, typ := range []string{service.AccountTypeOAuth, service.AccountTypeAPIKey} {
		for _, route := range []string{"responses", "compact", "chat", "messages", "embeddings", "images", "alpha"} {
			t.Run(route, func(t *testing.T) {
				group := &service.Group{ID: 12, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true, AllowImageGeneration: true}
				a := &service.Account{ID: 1, Platform: service.PlatformOpenAI, Type: typ, Status: service.StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{12}, Credentials: map[string]any{"access_token": "local-test", "api_key": "sk-test"}, Extra: map[string]any{"codex_cli_only": true, "codex_cli_only_route_only": true}}
				upstream := &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("filtered account must not reach upstream")}
				env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{a}}, upstream)
				path, body, forward := "/v1/responses", `{"model":"gpt-5","input":"hi"}`, env.handler.Responses
				switch route {
				case "compact":
					path = "/v1/responses/compact"
				case "chat":
					path, body, forward = "/v1/chat/completions", `{"model":"gpt-5","messages":[{"role":"user","content":"hi"}]}`, env.handler.ChatCompletions
				case "messages":
					path, body, forward = "/v1/messages", `{"model":"gpt-5","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`, env.handler.Messages
				case "embeddings":
					path, body, forward = "/v1/embeddings", `{"model":"gpt-5","input":"hi"}`, env.handler.Embeddings
				case "images":
					path, body, forward = "/v1/images/generations", `{"model":"gpt-image-1","prompt":"hi"}`, env.handler.Images
				case "alpha":
					path, body, forward = "/v1/alpha/search", `{"model":"gpt-5","query":"hi"}`, env.handler.AlphaSearch
				}
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				env.router(path, forward).ServeHTTP(w, req)
				require.NotEqual(t, http.StatusForbidden, w.Code, w.Body.String())
				require.GreaterOrEqual(t, w.Code, 400, w.Body.String())
				require.Empty(t, upstream.requestBody)
				require.Zero(t, env.handler.gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount)
			})
		}
	}
}

type codexRoutingCountUpstream struct {
	openAIChatCompletionsHTTPUpstreamStub
	accountIDs []int64
}

func (u *codexRoutingCountUpstream) Do(req *http.Request, proxy string, id int64, concurrency int) (*http.Response, error) {
	u.accountIDs = append(u.accountIDs, id)
	return u.openAIChatCompletionsHTTPUpstreamStub.Do(req, proxy, id, concurrency)
}

func TestCodexRoutingTokenCountEntries(t *testing.T) {
	for _, typ := range []string{service.AccountTypeOAuth, service.AccountTypeAPIKey} {
		for _, messages := range []bool{false, true} {
			for _, mode := range []string{"all-filtered", "mixed", "child-disabled"} {
				t.Run(fmt.Sprintf("messages=%v/%s", messages, mode), func(t *testing.T) {
					group := &service.Group{ID: 12, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
					a := &service.Account{ID: 1, Platform: service.PlatformOpenAI, Type: typ, Status: service.StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{12}, Credentials: map[string]any{"access_token": "local-test", "api_key": "sk-test"}, Extra: map[string]any{"codex_cli_only": true, "codex_cli_only_route_only": mode != "child-disabled"}}
					accounts := []*service.Account{a}
					if mode == "mixed" {
						b := *a
						b.ID, b.Extra = 2, nil
						accounts = append(accounts, &b)
					}
					upstream := &codexRoutingCountUpstream{openAIChatCompletionsHTTPUpstreamStub: openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"input_tokens":42}`))}}}
					env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: accounts}, upstream)
					path, body, forward := "/v1/responses/input_tokens", `{"model":"gpt-5","input":"hi"}`, env.handler.ResponsesInputTokens
					if messages {
						path, body, forward = "/v1/messages/count_tokens", `{"model":"gpt-5","messages":[{"role":"user","content":"hi"}]}`, env.handler.CountTokens
					}
					w := httptest.NewRecorder()
					req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("User-Agent", "third-party")
					env.router(path, forward).ServeHTTP(w, req)
					if mode == "all-filtered" {
						require.GreaterOrEqual(t, w.Code, 400, w.Body.String())
						require.NotEqual(t, http.StatusForbidden, w.Code, w.Body.String())
						require.Empty(t, upstream.accountIDs)
					} else if mode == "child-disabled" && typ == service.AccountTypeAPIKey {
						require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
						require.Empty(t, upstream.accountIDs)
					} else {
						require.Equal(t, http.StatusOK, w.Code, w.Body.String())
						wantID := int64(1)
						if mode == "mixed" {
							wantID = 2
						}
						require.Equal(t, []int64{wantID}, upstream.accountIDs)
						require.Contains(t, w.Body.String(), `"input_tokens":42`)
					}
					require.Zero(t, env.handler.gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount)
				})
			}
		}
	}
}

func TestCodexRoutingWebSocketEntry(t *testing.T) {
	for _, typ := range []string{service.AccountTypeOAuth, service.AccountTypeAPIKey} {
		group := &service.Group{ID: 12, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
		a := &service.Account{ID: 1, Platform: service.PlatformOpenAI, Type: typ, Status: service.StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{12}, Credentials: map[string]any{"access_token": "local-test", "api_key": "sk-test"}, Extra: map[string]any{"codex_cli_only": true, "codex_cli_only_route_only": true, "responses_websockets_v2_enabled": true}}
		upstream := &openAIChatCompletionsHTTPUpstreamStub{err: errors.New("filtered account must not reach upstream")}
		env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{a}}, upstream)
		server := newOpenAIWSHandlerTestServerWithAPIKey(t, env.handler, middleware.AuthSubject{UserID: env.apiKey.UserID, Concurrency: 1}, env.apiKey)
		defer server.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/openai/v1/responses", nil)
		require.NoError(t, err)
		defer conn.CloseNow()
		require.NoError(t, conn.Write(ctx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5","input":"hi"}`)))
		_, payload, err := conn.Read(ctx)
		if err != nil {
			var closeErr coderws.CloseError
			require.ErrorAs(t, err, &closeErr)
			require.Contains(t, strings.ToLower(closeErr.Reason), "account")
		} else {
			require.Contains(t, string(payload), "account")
		}
		require.Empty(t, upstream.requestBody)
		require.Zero(t, env.handler.gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount)
	}
}

func TestCodexRoutingSlotRecheck(t *testing.T) {
	for _, typ := range []string{service.AccountTypeOAuth, service.AccountTypeAPIKey} {
		for _, mode := range []string{"acquired", "fast", "wait"} {
			t.Run(mode, func(t *testing.T) {
				gw := &service.OpenAIGatewayService{}
				a := profitSlotTestAccount(1, 1)
				a.Type = typ
				a.Extra = map[string]any{"codex_cli_only": true}
				cache := &codexRoutingSlotCache{account: a, wait: mode == "wait"}
				h := &OpenAIGatewayHandler{gatewayService: gw, concurrencyHelper: NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatClaude, 0)}
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
				gw.PrepareCodexClientRouting(c, []byte(`{}`))
				selection := &service.AccountSelectionResult{Account: a, WaitPlan: &service.AccountWaitPlan{AccountID: 1, MaxConcurrency: 2, Timeout: time.Second, MaxWaiting: 2}}
				if mode == "acquired" {
					selection.Acquired = true
					selection.ReleaseFunc = func() { cache.accountReleases.Add(1) }
					a.Extra["codex_cli_only_route_only"] = true
				}
				started := false
				release, result := h.acquireResponsesAccountSlot(c, nil, "", selection, false, &started, zap.NewNop())
				require.Equal(t, openAISlotAcquireProfitVetoed, result)
				require.Nil(t, release)
				require.Equal(t, int64(1), cache.accountReleases.Load())
				require.Zero(t, w.Body.Len())
				require.Zero(t, gw.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount)
			})
		}
	}
}
