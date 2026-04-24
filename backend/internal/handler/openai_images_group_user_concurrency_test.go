package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayHandler_ImagesAcquiresUserGroupSlotWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			Scheduling:         config.GatewaySchedulingConfig{LoadBatchEnabled: false},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	groupID := int64(1)
	group := &service.Group{
		ID:                     groupID,
		Platform:               service.PlatformOpenAI,
		Status:                 service.StatusActive,
		Hydrated:               true,
		UserConcurrencyEnabled: true,
		UserConcurrencyLimit:   1,
	}
	account := &service.Account{
		ID:          11,
		Name:        "openai-test-account",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{"api_key": "sk-test"},
	}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_images_group_slot_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"images upstream rejected payload"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	cache := &concurrencyCacheMock{
		acquireUserSlotFn: func(_ context.Context, _ int64, _ int, _ string) (bool, error) {
			return true, nil
		},
		acquireUserGroupSlotFn: func(_ context.Context, _ int64, _ int64, _ int, _ string) (bool, error) {
			return true, nil
		},
	}
	concurrencyService := service.NewConcurrencyService(cache)
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		nil,
		nil,
		nil,
		nil,
		openAIChatCompletionsGatewayCacheStub{},
		cfg,
		nil,
		concurrencyService,
		billingService,
		nil,
		billingCacheService,
		httpUpstream,
		deferredService,
		nil,
		nil,
		nil,
		nil,
	)
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, cfg)

	apiKey := &service.APIKey{
		ID:      101,
		UserID:  202,
		Status:  service.StatusActive,
		GroupID: &groupID,
		User:    &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1},
		Group:   group,
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/images/generations", h.Images)

	reqBody := `{"model":"gpt-image-2","prompt":"draw a cat"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, int32(1), atomic.LoadInt32(&cache.releaseUserGroupCalled), "images handler should acquire and release group-user slot")
}
