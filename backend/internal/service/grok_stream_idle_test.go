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

type grokPoolStreamIdleMutationRepo struct {
	AccountRepository
	tempUnschedulableCalls int
	errorCalls             int
}

func (r *grokPoolStreamIdleMutationRepo) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	r.tempUnschedulableCalls++
	return nil
}

func (r *grokPoolStreamIdleMutationRepo) SetError(context.Context, int64, string) error {
	r.errorCalls++
	return nil
}

type grokPoolStreamIdleRuntimeBlocker struct{ blockCalls int }

func (b *grokPoolStreamIdleRuntimeBlocker) BlockAccountScheduling(*Account, time.Time, string) {
	b.blockCalls++
}

func (*grokPoolStreamIdleRuntimeBlocker) ClearAccountSchedulingBlock(int64) {}

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
	repo := &grokPoolStreamIdleMutationRepo{}
	runtimeBlocker := &grokPoolStreamIdleRuntimeBlocker{}
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitService.SetSettingService(NewSettingService(&openAIAdvancedSchedulerSettingRepoStub{values: map[string]string{
		SettingKeyStreamTimeoutSettings: `{"enabled":true,"action":"temp_unsched","temp_unsched_minutes":1,"threshold_count":1,"threshold_window_minutes":1}`,
	}}, &config.Config{}))
	rateLimitService.SetAccountRuntimeBlocker(runtimeBlocker)
	svc := &OpenAIGatewayService{
		cfg:              &config.Config{Gateway: config.GatewayConfig{StreamDataIntervalTimeout: 1}},
		rateLimitService: rateLimitService,
	}
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: reader}

	_, err := svc.handleStreamingResponseWithReasoning(context.Background(), resp, c, account, time.Now(), "grok-4.5", "grok-4.5", "")

	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.True(t, failoverErr.RetryableOnSameAccount)
	require.Zero(t, repo.tempUnschedulableCalls, "pool same-account retry must not persist timeout unscheduling")
	require.Zero(t, repo.errorCalls, "pool same-account retry must not persist timeout errors")
	require.Zero(t, runtimeBlocker.blockCalls, "pool same-account retry must not runtime-block the account")
	require.Nil(t, account.TempUnschedulableUntil)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
}
