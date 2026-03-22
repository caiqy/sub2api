//go:build unit

package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayHandler_GeminiV1BetaModels_UpstreamErrorStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitchesGemini: 1,
			Scheduling: config.GatewaySchedulingConfig{
				LoadBatchEnabled: false,
			},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	group := &service.Group{ID: 1, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID:          11,
		Name:        "gemini-test-account",
		Platform:    service.PlatformGemini,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{"api_key": "gemini-test-key", "base_url": "https://generativelanguage.googleapis.com"},
	}
	accountRepo := &stubAccountRepo{accounts: map[int64]*service.Account{account.ID: account}}
	groupRepo := &stubGroupRepo{group: group}
	usageLogRepo := &stubUsageLogRepo{}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"gemini_failed_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"message":"gemini upstream rejected payload"}}`)),
		},
	}

	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	concurrencyService := service.NewConcurrencyService(testutil.StubConcurrencyCache{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, cfg)
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
		nil,
		deferredService,
		nil,
		testutil.StubSessionLimitCache{},
		nil,
		nil,
		nil,
	)
	geminiCompatService := service.NewGeminiMessagesCompatService(
		accountRepo,
		groupRepo,
		testutil.StubGatewayCache{},
		nil,
		nil,
		nil,
		httpUpstream,
		nil,
		cfg,
	)
	h := NewGatewayHandler(gatewayService, geminiCompatService, nil, nil, concurrencyService, billingCacheService, nil, &service.APIKeyService{}, nil, nil, nil, cfg, nil)

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
	router.POST("/v1beta/models/*modelAction", h.GeminiV1BetaModels)

	reqBody := `{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.0-flash:generateContent", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.NotNil(t, usageLogRepo.lastLog)
	require.Equal(t, 0, usageLogRepo.lastLog.InputTokens)
	require.Equal(t, 0, usageLogRepo.lastLog.OutputTokens)
	require.Equal(t, 0.0, usageLogRepo.lastLog.TotalCost)
	require.Equal(t, 0.0, usageLogRepo.lastLog.ActualCost)
	require.NotNil(t, usageLogRepo.lastLog.DetailSnapshot)
	require.JSONEq(t, reqBody, usageLogRepo.lastLog.DetailSnapshot.RequestBody)
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.ResponseBody, "gemini upstream rejected payload")
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.UpstreamRequestHeaders, "X-Goog-Api-Key: gemini-test-key")
}

func TestGatewayHandler_GeminiV1BetaModels_FailoverExhaustedStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitchesGemini: 1,
			Scheduling: config.GatewaySchedulingConfig{
				LoadBatchEnabled: false,
			},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	group := &service.Group{ID: 1, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID:          11,
		Name:        "gemini-test-account",
		Platform:    service.PlatformGemini,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{"api_key": "gemini-test-key", "base_url": "https://generativelanguage.googleapis.com"},
	}
	accountRepo := &stubAccountRepo{accounts: map[int64]*service.Account{account.ID: account}}
	groupRepo := &stubGroupRepo{group: group}
	usageLogRepo := &stubUsageLogRepo{}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"gemini_failover_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"message":"gemini raw failover","status":"RESOURCE_EXHAUSTED","code":"RESOURCE_EXHAUSTED_RAW"}}`)),
		},
	}

	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	concurrencyService := service.NewConcurrencyService(testutil.StubConcurrencyCache{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, cfg)
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
		nil,
		deferredService,
		nil,
		testutil.StubSessionLimitCache{},
		nil,
		nil,
		nil,
	)
	geminiCompatService := service.NewGeminiMessagesCompatService(
		accountRepo,
		groupRepo,
		testutil.StubGatewayCache{},
		nil,
		nil,
		nil,
		httpUpstream,
		nil,
		cfg,
	)
	h := NewGatewayHandler(gatewayService, geminiCompatService, nil, nil, concurrencyService, billingCacheService, nil, &service.APIKeyService{}, nil, nil, nil, cfg, nil)
	h.maxAccountSwitchesGemini = 0

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
	router.POST("/v1beta/models/*modelAction", h.GeminiV1BetaModels)

	reqBody := `{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.0-flash:generateContent", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.NotNil(t, usageLogRepo.lastLog)
	require.NotNil(t, usageLogRepo.lastLog.DetailSnapshot)
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.ResponseBody, `"RESOURCE_EXHAUSTED_RAW"`)
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.ResponseBody, "gemini raw failover")
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.UpstreamRequestHeaders, "X-Goog-Api-Key: gemini-test-key")
}

func TestGatewayHandler_GeminiV1BetaModels_SelectionExhaustedAfterFailoverStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitchesGemini: 1,
			Scheduling: config.GatewaySchedulingConfig{
				LoadBatchEnabled: false,
			},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	group := &service.Group{ID: 1, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID:          11,
		Name:        "gemini-test-account",
		Platform:    service.PlatformGemini,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{"api_key": "gemini-test-key", "base_url": "https://generativelanguage.googleapis.com"},
	}
	accountRepo := &stubAccountRepo{accounts: map[int64]*service.Account{account.ID: account}}
	groupRepo := &stubGroupRepo{group: group}
	usageLogRepo := &stubUsageLogRepo{}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"gemini_selection_exhausted_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"message":"gemini raw failover","status":"RESOURCE_EXHAUSTED","code":"RESOURCE_EXHAUSTED_RAW"}}`)),
		},
	}

	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	concurrencyService := service.NewConcurrencyService(testutil.StubConcurrencyCache{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, cfg)
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
		nil,
		deferredService,
		nil,
		testutil.StubSessionLimitCache{},
		nil,
		nil,
		nil,
	)
	geminiCompatService := service.NewGeminiMessagesCompatService(
		accountRepo,
		groupRepo,
		testutil.StubGatewayCache{},
		nil,
		nil,
		nil,
		httpUpstream,
		nil,
		cfg,
	)
	h := NewGatewayHandler(gatewayService, geminiCompatService, nil, nil, concurrencyService, billingCacheService, nil, &service.APIKeyService{}, nil, nil, nil, cfg, nil)

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
	router.POST("/v1beta/models/*modelAction", h.GeminiV1BetaModels)

	reqBody := `{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.0-flash:generateContent", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.JSONEq(t, `{"error":{"code":429,"message":"Upstream rate limit exceeded, please retry later","status":"RESOURCE_EXHAUSTED"}}`, rec.Body.String())
	require.NotNil(t, usageLogRepo.lastLog)
	require.NotNil(t, usageLogRepo.lastLog.DetailSnapshot)
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.ResponseBody, `"RESOURCE_EXHAUSTED_RAW"`)
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.ResponseBody, "gemini raw failover")
	require.Contains(t, usageLogRepo.lastLog.DetailSnapshot.UpstreamRequestHeaders, "X-Goog-Api-Key: gemini-test-key")
}
