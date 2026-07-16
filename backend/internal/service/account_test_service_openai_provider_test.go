package service

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
)

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

func TestAccountTestServiceOpenAIOAuthUsesStoredAccessToken(t *testing.T) {
	upstream := &accountTestHTTPUpstreamRecorder{}
	svc := &AccountTestService{
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          90,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "stored-token",
		},
	}

	err := svc.testOpenAIAccountConnection(newAccountTestGinContext(), account, "gpt-5.4", "", "")

	if err != nil {
		t.Fatalf("testOpenAIAccountConnection returned error: %v", err)
	}
	if upstream.request == nil {
		t.Fatalf("expected upstream request")
	}
	if got := upstream.request.Header.Get("Authorization"); got != "Bearer stored-token" {
		t.Fatalf("Authorization header = %q, want %q", got, "Bearer stored-token")
	}
}
