//go:build unit

package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayHandler_MessagesForwardErrorStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			Scheduling: config.GatewaySchedulingConfig{
				LoadBatchEnabled: false,
			},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	group := &service.Group{ID: 1, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID:          11,
		Name:        "anthropic-test-account",
		Platform:    service.PlatformAnthropic,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{"api_key": "anthropic-test-key"},
	}
	accountRepo := &stubAccountRepo{accounts: map[int64]*service.Account{account.ID: account}}
	groupRepo := &stubGroupRepo{group: group}
	usageLogRepo := &stubUsageLogRepo{}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		delay: 5 * time.Millisecond,
		response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"gateway_failed_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"anthropic upstream rejected payload"}}`)),
		},
	}

	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	concurrencyService := service.NewConcurrencyService(testutil.StubConcurrencyCache{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewGatewayService(
		accountRepo,
		groupRepo,
		usageLogRepo,
		nil,
		nil,
		nil,
		nil,
		testutil.StubGatewayCache{},
		cfg,
		nil,
		concurrencyService,
		billingService,
		nil,
		billingCacheService,
		nil,
		httpUpstream,
		deferredService,
		nil,
		testutil.StubSessionLimitCache{},
		nil,
		nil,
		nil,
	)
	h := NewGatewayHandler(gatewayService, nil, nil, nil, concurrencyService, billingCacheService, nil, &service.APIKeyService{}, nil, nil, nil, cfg, nil)

	apiKey := &service.APIKey{
		ID:      101,
		UserID:  202,
		Status:  service.StatusActive,
		GroupID: &group.ID,
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
	router.POST("/v1/messages", h.Messages)

	reqBody := `{"model":"claude-3-5-sonnet-20241022","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"output_config":{"effort":"high"}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.NotNil(t, usageLogRepo.lastLog)
	require.Equal(t, 0, usageLogRepo.lastLog.InputTokens)
	require.Equal(t, 0, usageLogRepo.lastLog.OutputTokens)
	require.Equal(t, 0.0, usageLogRepo.lastLog.TotalCost)
	require.Equal(t, 0.0, usageLogRepo.lastLog.ActualCost)
	require.NotNil(t, usageLogRepo.lastLog.DurationMs)
	require.Greater(t, *usageLogRepo.lastLog.DurationMs, 0)
	require.NotNil(t, usageLogRepo.lastLog.ReasoningEffort)
	require.Equal(t, "high", *usageLogRepo.lastLog.ReasoningEffort)
	require.NotNil(t, usageLogRepo.lastLog.DetailSnapshot)
	require.JSONEq(t, reqBody, usageLogRepo.lastLog.DetailSnapshot.RequestBody)
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.ResponseBody, "anthropic upstream rejected payload")
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.UpstreamRequestHeaders, "X-Api-Key: anthropic-test-key")
}

func TestGatewayHandler_MessagesFailoverExhaustedStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			Scheduling: config.GatewaySchedulingConfig{
				LoadBatchEnabled: false,
			},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	group := &service.Group{ID: 1, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID:          11,
		Name:        "anthropic-test-account",
		Platform:    service.PlatformAnthropic,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{"api_key": "anthropic-test-key"},
	}
	accountRepo := &stubAccountRepo{accounts: map[int64]*service.Account{account.ID: account}}
	groupRepo := &stubGroupRepo{group: group}
	usageLogRepo := &stubUsageLogRepo{}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"gateway_failover_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"anthropic_rate_limited_raw","message":"anthropic raw failover"}}`)),
		},
	}

	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	concurrencyService := service.NewConcurrencyService(testutil.StubConcurrencyCache{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewGatewayService(
		accountRepo,
		groupRepo,
		usageLogRepo,
		nil,
		nil,
		nil,
		nil,
		testutil.StubGatewayCache{},
		cfg,
		nil,
		concurrencyService,
		billingService,
		nil,
		billingCacheService,
		nil,
		httpUpstream,
		deferredService,
		nil,
		testutil.StubSessionLimitCache{},
		nil,
		nil,
		nil,
	)
	h := NewGatewayHandler(gatewayService, nil, nil, nil, concurrencyService, billingCacheService, nil, &service.APIKeyService{}, nil, nil, nil, cfg, nil)
	h.maxAccountSwitches = 0

	apiKey := &service.APIKey{
		ID:      101,
		UserID:  202,
		Status:  service.StatusActive,
		GroupID: &group.ID,
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
	router.POST("/v1/messages", h.Messages)

	reqBody := `{"model":"claude-3-5-sonnet-20241022","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.NotNil(t, usageLogRepo.lastLog)
	require.NotNil(t, usageLogRepo.lastLog.DetailSnapshot)
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.ResponseHeaders, "Content-Type: application/json")
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.ResponseHeaders, "X-Request-Id: gateway_failover_123")
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.ResponseBody, `"anthropic_rate_limited_raw"`)
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.ResponseBody, "anthropic raw failover")
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.UpstreamRequestHeaders, "X-Api-Key: anthropic-test-key")
}

func TestGatewayHandler_MessagesSelectionExhaustedAfterFailoverStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			Scheduling: config.GatewaySchedulingConfig{
				LoadBatchEnabled: false,
			},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	group := &service.Group{ID: 1, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID:          11,
		Name:        "anthropic-test-account",
		Platform:    service.PlatformAnthropic,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{"api_key": "anthropic-test-key"},
	}
	accountRepo := &stubAccountRepo{accounts: map[int64]*service.Account{account.ID: account}}
	groupRepo := &stubGroupRepo{group: group}
	usageLogRepo := &stubUsageLogRepo{}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"gateway_selection_exhausted_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"anthropic_rate_limited_raw","message":"anthropic raw failover"}}`)),
		},
	}

	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	concurrencyService := service.NewConcurrencyService(testutil.StubConcurrencyCache{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewGatewayService(
		accountRepo,
		groupRepo,
		usageLogRepo,
		nil,
		nil,
		nil,
		nil,
		testutil.StubGatewayCache{},
		cfg,
		nil,
		concurrencyService,
		billingService,
		nil,
		billingCacheService,
		nil,
		httpUpstream,
		deferredService,
		nil,
		testutil.StubSessionLimitCache{},
		nil,
		nil,
		nil,
	)
	h := NewGatewayHandler(gatewayService, nil, nil, nil, concurrencyService, billingCacheService, nil, &service.APIKeyService{}, nil, nil, nil, cfg, nil)

	apiKey := &service.APIKey{
		ID:      101,
		UserID:  202,
		Status:  service.StatusActive,
		GroupID: &group.ID,
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
	router.POST("/v1/messages", h.Messages)

	reqBody := `{"model":"claude-3-5-sonnet-20241022","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.JSONEq(t, `{"type":"error","error":{"type":"rate_limit_error","message":"Upstream rate limit exceeded, please retry later"}}`, rec.Body.String())
	require.NotNil(t, usageLogRepo.lastLog)
	require.NotNil(t, usageLogRepo.lastLog.DetailSnapshot)
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.ResponseBody, `"anthropic_rate_limited_raw"`)
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.ResponseBody, "anthropic raw failover")
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.UpstreamRequestHeaders, "X-Api-Key: anthropic-test-key")
}

func TestGatewayHandler_MessagesStreamingPartialWriteFailureStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			Scheduling: config.GatewaySchedulingConfig{
				LoadBatchEnabled: false,
			},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	group := &service.Group{ID: 1, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID:          11,
		Name:        "anthropic-stream-test-account",
		Platform:    service.PlatformAnthropic,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{"api_key": "anthropic-test-key"},
	}
	accountRepo := &stubAccountRepo{accounts: map[int64]*service.Account{account.ID: account}}
	groupRepo := &stubGroupRepo{group: group}
	usageLogRepo := &stubUsageLogRepo{}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"X-Request-Id": []string{"gateway_stream_fail_123"},
			},
			Body: io.NopCloser(strings.NewReader(partialMessageStartSSE +
				"event: error\n" +
				"data: {\"type\":\"error\",\"error\":{\"type\":\"api_error\",\"message\":\"upstream stream failed after partial write\"}}\n\n")),
		},
	}

	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	concurrencyService := service.NewConcurrencyService(testutil.StubConcurrencyCache{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewGatewayService(
		accountRepo,
		groupRepo,
		usageLogRepo,
		nil,
		nil,
		nil,
		nil,
		testutil.StubGatewayCache{},
		cfg,
		nil,
		concurrencyService,
		billingService,
		nil,
		billingCacheService,
		nil,
		httpUpstream,
		deferredService,
		nil,
		testutil.StubSessionLimitCache{},
		nil,
		nil,
		nil,
	)
	h := NewGatewayHandler(gatewayService, nil, nil, nil, concurrencyService, billingCacheService, nil, &service.APIKeyService{}, nil, nil, nil, cfg, nil)

	apiKey := &service.APIKey{
		ID:      101,
		UserID:  202,
		Status:  service.StatusActive,
		GroupID: &group.ID,
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
	router.POST("/v1/messages", h.Messages)

	reqBody := `{"model":"claude-3-5-sonnet-20241022","max_tokens":16,"stream":true,"messages":[{"role":"user","content":"hello"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "event: message_start")
	require.Contains(t, rec.Body.String(), `data: {"type":"error"`)
	require.Contains(t, rec.Body.String(), `"type":"error"`)
	require.NotNil(t, usageLogRepo.lastLog)
	require.NotNil(t, usageLogRepo.lastLog.DetailSnapshot)
	require.JSONEq(t, reqBody, usageLogRepo.lastLog.DetailSnapshot.RequestBody)
}
