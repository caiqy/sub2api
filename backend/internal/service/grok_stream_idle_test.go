//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResolveGrokStreamIdleTimeout(t *testing.T) {
	require.Equal(t, 90*time.Second, resolveGrokStreamIdleTimeout(90))
	require.Equal(t, defaultGrokStreamIdleTimeout, resolveGrokStreamIdleTimeout(0))
	require.Equal(t, defaultGrokStreamIdleTimeout, resolveGrokStreamIdleTimeout(-1))
}

func TestGrokStreamIdleFailoverError(t *testing.T) {
	account := &Account{ID: 1, Platform: PlatformGrok, Type: AccountTypeOAuth}
	err := grokStreamIdleFailoverError(account, 180*time.Second)
	require.NotNil(t, err)
	require.Equal(t, 502, err.StatusCode)
	require.True(t, err.SafeToFailoverAfterWrite)
	require.Contains(t, string(err.ResponseBody), "empty_upstream")
}

func TestGrokPoolStreamIdleDefersTemporaryUnscheduleUntilSameAccountRetriesExhaust(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	account := &Account{
		ID: 91003, Platform: PlatformGrok, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "pool-key", "pool_mode": true, "pool_mode_retry_count": 1},
	}
	svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{StreamDataIntervalTimeout: 1}}}
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: reader}

	_, err := svc.handleStreamingResponseWithReasoning(context.Background(), resp, c, account, time.Now(), "grok-4.5", "grok-4.5", "")

	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.True(t, failoverErr.RetryableOnSameAccount)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
}
