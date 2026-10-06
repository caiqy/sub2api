package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const validSystemOneHandlerBody = `{"model":"jev-latest","state":"sample","questions":{"q":{"type":"noul","instructions":"Evaluate"}}}`

func newSystemOneHandlerContext(body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/systemone", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, recorder
}

func TestSystemOnePreservesForkGatewayAdmissionAndUsage(t *testing.T) {
	for _, blocked := range []bool{true, false} {
		t.Run(map[bool]string{true: "group-concurrency-denied", false: "native-usage-detail"}[blocked], func(t *testing.T) {
			group := &service.Group{ID: 91, Platform: service.PlatformTypeSafe, Status: service.StatusActive, Hydrated: true, UserConcurrencyEnabled: true, UserConcurrencyLimit: 2}
			account := &service.Account{ID: 191, Platform: service.PlatformTypeSafe, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "native-key"}}
			groupCalls := 0
			cache := &gatewayUserGroupConcurrencyCacheMock{
				concurrencyCacheMock: &concurrencyCacheMock{
					acquireUserSlotFn:    func(context.Context, int64, int, string) (bool, error) { return true, nil },
					acquireAccountSlotFn: func(context.Context, int64, int, string) (bool, error) { return true, nil },
				},
				acquireUserGroupSlotFn: func(_ context.Context, userID, groupID int64, limit int, _ string) (bool, error) {
					groupCalls++
					require.Equal(t, int64(202), userID)
					require.Equal(t, group.ID, groupID)
					require.Equal(t, 2, limit)
					if blocked {
						return false, errors.New("group concurrency unavailable")
					}
					return true, nil
				},
			}
			upstream := &systemOneNativeHandlerUpstream{}
			env := newTerminalGatewayMessagesEnvWithConcurrencyCache(t, group, upstream, cache, account)
			c, recorder := newSystemOneHandlerContext(validSystemOneHandlerBody)
			env.routerFor("/v1/systemone", env.handler.SystemOne).ServeHTTP(recorder, c.Request)
			if blocked {
				require.Equal(t, http.StatusTooManyRequests, recorder.Code, recorder.Body.String())
				require.Zero(t, upstream.calls)
				return
			}
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			require.Equal(t, 1, upstream.calls)
			usage := <-env.usageRepo.created
			require.NotNil(t, usage.UpstreamModel)
			require.Equal(t, "jev-latest", *usage.UpstreamModel)
			require.NotNil(t, usage.DetailSnapshot)
			requestSnapshot := service.RequestBodyPreviewSnapshot(validSystemOneHandlerBody, int64(len(validSystemOneHandlerBody)))
			require.Equal(t, requestSnapshot, usage.DetailSnapshot.RequestBody)
			require.Equal(t, requestSnapshot, usage.DetailSnapshot.UpstreamRequestBody)
			require.Contains(t, usage.DetailSnapshot.ResponseBody, `"input_tokens":123`)
			require.Equal(t, 1, groupCalls)
			require.Equal(t, int32(1), cache.releaseUserGroupCalled)
		})
	}
}

type systemOneNativeHandlerUpstream struct {
	service.HTTPUpstream
	calls int
}

func (u *systemOneNativeHandlerUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.calls++
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"model":"jev-1.13.0","answers":{},"usage":{"input_tokens":123}}`))}, nil
}

func TestSystemOneRequiresAuthentication(t *testing.T) {
	c, recorder := newSystemOneHandlerContext(validSystemOneHandlerBody)
	(&GatewayHandler{}).SystemOne(c)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Contains(t, recorder.Body.String(), "authentication_error")
}

func TestSystemOneRejectsNonTypeSafeGroupBeforeScheduling(t *testing.T) {
	c, recorder := newSystemOneHandlerContext(validSystemOneHandlerBody)
	groupID := int64(3)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{ID: 4, UserID: 5, GroupID: &groupID, Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI}})
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 5, Concurrency: 1})

	(&GatewayHandler{cfg: &config.Config{Gateway: config.GatewayConfig{MaxBodySize: 1 << 20}}}).SystemOne(c)
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, recorder.Body.String(), "only available for TypeSafe")
}

func newTypeSafeGroupContext(t *testing.T, path, body, groupPlatform string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	c, recorder := newSystemOneHandlerContext(body)
	c.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	groupID := int64(9)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{ID: 4, UserID: 5, GroupID: &groupID, Group: &service.Group{ID: groupID, Platform: groupPlatform}})
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 5, Concurrency: 1})
	return c, recorder
}

func TestRejectSystemOneOnlyPlatform(t *testing.T) {
	write := func(c *gin.Context, status int, errType, message string) {
		c.JSON(status, gin.H{"type": errType, "message": message})
	}

	c, recorder := newTypeSafeGroupContext(t, "/v1/messages", `{}`, service.PlatformTypeSafe)
	require.True(t, rejectSystemOneOnlyPlatform(c, mustAPIKey(t, c), write))
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, recorder.Body.String(), systemOneOnlyPlatformMessage)

	c, _ = newTypeSafeGroupContext(t, "/v1/messages", `{}`, service.PlatformComposite)
	require.False(t, rejectSystemOneOnlyPlatform(c, mustAPIKey(t, c), write))
	c.Request = c.Request.WithContext(service.WithResolvedTargetPlatform(c.Request.Context(), service.PlatformTypeSafe))
	require.True(t, rejectSystemOneOnlyPlatform(c, mustAPIKey(t, c), write))

	c, _ = newTypeSafeGroupContext(t, "/v1/messages", `{}`, service.PlatformAnthropic)
	require.False(t, rejectSystemOneOnlyPlatform(c, mustAPIKey(t, c), write))

	c, _ = newTypeSafeGroupContext(t, "/antigravity/v1/messages", `{}`, service.PlatformTypeSafe)
	c.Set(string(middleware2.ContextKeyForcePlatform), service.PlatformAntigravity)
	require.False(t, rejectSystemOneOnlyPlatform(c, mustAPIKey(t, c), write))
}

func mustAPIKey(t *testing.T, c *gin.Context) *service.APIKey {
	t.Helper()
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	require.True(t, ok)
	return apiKey
}

func TestTypeSafeGroupsRejectNonSystemOneProtocolsBeforeScheduling(t *testing.T) {
	h := &GatewayHandler{
		cfg:            &config.Config{Gateway: config.GatewayConfig{MaxBodySize: 1 << 20}},
		gatewayService: &service.GatewayService{},
	}
	for _, tc := range []struct {
		name    string
		path    string
		body    string
		handler func(*gin.Context)
	}{
		{"messages", "/v1/messages", `{"model":"jev-latest","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`, h.Messages},
		{"count_tokens", "/v1/messages/count_tokens", `{"model":"jev-latest","messages":[{"role":"user","content":"hi"}]}`, h.CountTokens},
		{"chat_completions", "/v1/chat/completions", `{"model":"jev-latest","messages":[{"role":"user","content":"hi"}]}`, h.ChatCompletions},
		{"responses", "/v1/responses", `{"model":"jev-latest","input":"hi"}`, h.Responses},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, groupPlatform := range []string{service.PlatformTypeSafe, service.PlatformComposite} {
				c, recorder := newTypeSafeGroupContext(t, tc.path, tc.body, groupPlatform)
				if groupPlatform == service.PlatformComposite {
					c.Request = c.Request.WithContext(service.WithResolvedTargetPlatform(c.Request.Context(), service.PlatformTypeSafe))
				}
				tc.handler(c)
				require.Equal(t, http.StatusNotFound, recorder.Code, recorder.Body.String())
				require.Contains(t, recorder.Body.String(), systemOneOnlyPlatformMessage)
			}
		})
	}
}
