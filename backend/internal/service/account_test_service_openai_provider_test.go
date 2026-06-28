package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
)

type accountTestTokenCacheStub struct {
	token string
}

func (c *accountTestTokenCacheStub) GetAccessToken(context.Context, string) (string, error) {
	return c.token, nil
}

func (c *accountTestTokenCacheStub) SetAccessToken(context.Context, string, string, time.Duration) error {
	return nil
}

func (c *accountTestTokenCacheStub) DeleteAccessToken(context.Context, string) error {
	return nil
}

func (c *accountTestTokenCacheStub) AcquireRefreshLock(context.Context, string, time.Duration) (bool, error) {
	return true, nil
}

func (c *accountTestTokenCacheStub) ReleaseRefreshLock(context.Context, string) error {
	return nil
}

type accountTestHTTPUpstreamRecorder struct {
	request *http.Request
}

func (u *accountTestHTTPUpstreamRecorder) Do(*http.Request, string, int64, int) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected Do call")
}

func (u *accountTestHTTPUpstreamRecorder) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	u.request = req
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\"}\n\n")),
	}, nil
}

func newAccountTestGinContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", nil)
	return c
}

func TestAccountTestServiceOpenAIOAuthUsesTokenProvider(t *testing.T) {
	upstream := &accountTestHTTPUpstreamRecorder{}
	tokenProvider := NewOpenAITokenProvider(nil, &accountTestTokenCacheStub{token: "provider-token"}, nil)
	svc := &AccountTestService{
		httpUpstream:        upstream,
		openAITokenProvider: tokenProvider,
	}
	account := &Account{
		ID:          90,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"refresh_token": "refresh-token",
			"expires_at":    time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		},
	}

	err := svc.testOpenAIAccountConnection(newAccountTestGinContext(), account, "gpt-5.4", "", "")

	if err != nil {
		t.Fatalf("testOpenAIAccountConnection returned error: %v", err)
	}
	if upstream.request == nil {
		t.Fatalf("expected upstream request")
	}
	if got := upstream.request.Header.Get("Authorization"); got != "Bearer provider-token" {
		t.Fatalf("Authorization header = %q, want %q", got, "Bearer provider-token")
	}
}
