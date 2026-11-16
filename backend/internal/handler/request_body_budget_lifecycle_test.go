package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func handlerBodyBudgetUsed(budget *httputil.BodyMemoryBudget, limit int64) int64 {
	ctx, release := httputil.WithRequestBodyMemoryBudget(context.Background(), budget)
	defer release()
	lo, hi := int64(0), limit
	for lo < hi {
		mid := lo + (hi-lo+1)/2
		if httputil.ReserveRequestBodyMemory(ctx, mid) == nil {
			httputil.ReleaseRequestBodyMemory(ctx, mid)
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return limit - lo
}

type bodyBudgetRetryUpstream struct {
	service.HTTPUpstream
	budget   *httputil.BodyMemoryBudget
	limit    int64
	cancel   context.CancelFunc
	accounts []int64
	used     []int64
	platform string
}

func (u *bodyBudgetRetryUpstream) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.accounts = append(u.accounts, accountID)
	u.used = append(u.used, handlerBodyBudgetUsed(u.budget, u.limit))
	_, err := io.Copy(io.Discard, req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, err
	}
	if u.cancel != nil {
		u.cancel()
		return nil, context.Canceled
	}
	if len(u.accounts) == 1 {
		if u.platform == service.PlatformAntigravity {
			return &http.Response{StatusCode: 429, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"code":429,"status":"RESOURCE_EXHAUSTED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"RATE_LIMIT_EXCEEDED","metadata":{"model":"claude-opus-4-6"}},{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"10s"}]}}`))}, nil
		}
		return &http.Response{StatusCode: 503, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"controlled retry"}}`))}, nil
	}
	if u.platform == service.PlatformGemini {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`))}, nil
	}
	if u.platform == service.PlatformAntigravity {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: {\"response\":{\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":1,\"candidatesTokenCount\":1}}}\n\n"))}, nil
	}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"msg_test","type":"message","role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))}, nil
}

func (u *bodyBudgetRetryUpstream) DoWithTLS(req *http.Request, proxy string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxy, accountID, concurrency)
}

func TestMessagesBodyBudgetLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name     string
		payload  int
		limit    int64
		cancel   bool
		warmup   bool
		platform string
	}{
		{"retry", 12 << 20, 256 << 20, false, false, service.PlatformAnthropic},
		{"production-budget", 80 << 20, 1 << 30, false, false, service.PlatformAnthropic},
		{"warmup-enabled", 12 << 20, 256 << 20, false, true, service.PlatformAnthropic},
		{"gemini", 12 << 20, 256 << 20, false, false, service.PlatformGemini},
		{"antigravity", 12 << 20, 256 << 20, false, false, service.PlatformAntigravity},
		{"cancel", 12 << 20, 256 << 20, true, false, service.PlatformAnthropic},
		{"real-overload", 12 << 20, 1 << 20, false, false, service.PlatformAnthropic},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			budget := httputil.NewBodyMemoryBudget(tc.limit)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			upstream := &bodyBudgetRetryUpstream{budget: budget, limit: tc.limit, platform: tc.platform}
			if tc.cancel {
				upstream.cancel = cancel
			}
			group := &service.Group{ID: 47, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
			accounts := []*service.Account{
				{ID: 147, Name: "first", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "test-first"}},
				{ID: 148, Name: "second", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{"api_key": "test-second"}},
			}
			group.Platform = tc.platform
			for _, account := range accounts {
				account.Platform = tc.platform
				if tc.platform == service.PlatformAntigravity {
					account.Type = service.AccountTypeOAuth
					account.Credentials = map[string]any{"access_token": "test-token", "project_id": "project"}
				}
			}
			if tc.warmup {
				for _, account := range accounts {
					account.Credentials["intercept_warmup_requests"] = true
				}
			}
			env := newTerminalGatewayMessagesEnv(t, group, upstream, accounts...)
			tmp := t.TempDir()
			t.Setenv("TEMP", tmp)
			t.Setenv("TMP", tmp)
			t.Setenv("TMPDIR", tmp)
			body := []byte(`{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"` + strings.Repeat("x", tc.payload) + `"}]}`)
			router := gin.New()
			router.Use(middleware.RequestBodyLimit(200<<20, budget))
			router.Use(func(c *gin.Context) {
				c.Set(string(middleware.ContextKeyAPIKey), env.apiKey)
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: env.apiKey.UserID, Concurrency: 1})
				c.Next()
			})
			router.Use(middleware.UsageDetailCapture())
			router.POST("/v1/messages", env.handler.Messages)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(body)).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec, req)
			t.Logf("body=%d budget=%d status=%d accounts=%v reserved_at_send=%v", len(body), tc.limit, rec.Code, upstream.accounts, upstream.used)
			require.Zero(t, handlerBodyBudgetUsed(budget, tc.limit), "request and nested leases must all be released")
			switch {
			case tc.name == "real-overload":
				require.Equal(t, 503, rec.Code)
				require.Equal(t, "1", rec.Header().Get("Retry-After"))
				require.Contains(t, rec.Body.String(), "Request body buffering capacity is exhausted; retry later")
				require.Empty(t, upstream.accounts)
			case tc.cancel:
				require.ErrorIs(t, ctx.Err(), context.Canceled)
				require.Len(t, upstream.accounts, 1)
			default:
				require.Equal(t, 200, rec.Code, rec.Body.String())
				require.Equal(t, []int64{147, 148}, upstream.accounts)
				require.Equal(t, upstream.used[0], upstream.used[1], "failover must not accumulate previous attempts' temporary copies")
				require.GreaterOrEqual(t, upstream.used[0], int64(len(body)), "the live forwarding copy must remain charged during Do")
			}
		})
	}
}
