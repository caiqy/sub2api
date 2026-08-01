//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// pinObserveOnlyCoordinator 固定 ObserveOnly=true 的默认 coordinator：
// 若 handler 未在 coordinator 之前拒绝缺键请求，请求仍会被执行（观察期不拦截），测试即失败。
func pinObserveOnlyCoordinator(t *testing.T) {
	t.Helper()
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(userStoreUnavailableRepoStub{}, service.DefaultIdempotencyConfig()))
	t.Cleanup(func() { service.SetDefaultIdempotencyCoordinator(nil) })
}

func TestSubscriptionAdvanceQuotaCycle_MissingIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pinObserveOnlyCoordinator(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/7/advance-quota-cycle", strings.NewReader(`{"daily":true}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "7"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 10})

	// nil service：若 handler 误调用 service 会 panic，测试通过即证明 service 未执行
	(&SubscriptionHandler{}).AdvanceQuotaCycle(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "IDEMPOTENCY_KEY_REQUIRED")
}

func TestSubscriptionAdvanceQuotaCycle_XIdempotencyKeyIsNotAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pinObserveOnlyCoordinator(t)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/7/advance-quota-cycle", strings.NewReader(`{"daily":true}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Idempotency-Key", "some-key")
	c.Params = gin.Params{{Key: "id", Value: "7"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 10})

	// 仅 canonical Idempotency-Key 有效；X-Idempotency-Key 不应通过校验
	(&SubscriptionHandler{}).AdvanceQuotaCycle(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "IDEMPOTENCY_KEY_REQUIRED")
}

func TestSubscriptionAdvanceQuotaCycle_RequiresWindowSelection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/7/advance-quota-cycle", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "7"}}
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 10})

	(&SubscriptionHandler{}).AdvanceQuotaCycle(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "At least one quota window")
}
