package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func waitForOpenAIFailedUsageLog(t *testing.T, repo *openAIChatCompletionsUsageLogRepoStub) *service.UsageLog {
	t.Helper()

	if repo.lastLog != nil {
		return repo.lastLog
	}

	select {
	case log := <-repo.created:
		repo.lastLog = log
		return log
	case <-time.After(2 * time.Second):
		return repo.lastLog
	}
}

func TestOpenAIGatewayHandler_SubmitFailedUsageLog_UsesMessagesFallbackModelAsUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	gatewayService := service.NewOpenAIGatewayService(nil, usageRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := NewOpenAIGatewayHandler(gatewayService, nil, nil, nil, nil, nil, nil)

	apiKey := &service.APIKey{ID: 101, UserID: 202, User: &service.User{ID: 202}}
	account := &service.Account{ID: 11, Platform: service.PlatformOpenAI, Credentials: map[string]any{"api_key": "sk-test"}}
	fallbackModel := "gpt-4.1-mini"
	reqModel := "claude-3-5-sonnet-20241022"

	router := gin.New()
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/messages", func(c *gin.Context) {
		c.Set("openai_messages_fallback_model", fallbackModel)
		h.submitFailedUsageLog(c, apiKey, account, reqModel, false, 0, nil, nil, 0, nil, "handler.openai_gateway.messages")
		c.Status(http.StatusBadRequest)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-3-5-sonnet-20241022","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, fallbackModel, *log.UpstreamModel)
}

func TestOpenAIGatewayHandler_SubmitFailoverFailedUsageLog_UsesChatCompletionsFallbackModelAsUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	gatewayService := service.NewOpenAIGatewayService(nil, usageRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := NewOpenAIGatewayHandler(gatewayService, nil, nil, nil, nil, nil, nil)

	apiKey := &service.APIKey{ID: 101, UserID: 202, User: &service.User{ID: 202}}
	account := &service.Account{ID: 11, Platform: service.PlatformOpenAI, Credentials: map[string]any{"api_key": "sk-test"}}
	fallbackModel := "gpt-4.1-mini"
	reqModel := "gpt-5.4"

	router := gin.New()
	router.Use(middleware.UsageDetailCapture())
	router.POST("/chat/completions", func(c *gin.Context) {
		c.Set("openai_chat_completions_fallback_model", fallbackModel)
		h.submitFailoverFailedUsageLog(c, apiKey, account, reqModel, false, &service.UpstreamFailoverError{}, 0, nil, "handler.openai_gateway.chat_completions")
		c.Status(http.StatusTooManyRequests)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chat/completions", strings.NewReader(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":false}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, fallbackModel, *log.UpstreamModel)
}

func TestOpenAIGatewayHandler_SubmitFailedUsageLog_PrefersExactUpstreamModelOverFallbackModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	gatewayService := service.NewOpenAIGatewayService(nil, usageRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := NewOpenAIGatewayHandler(gatewayService, nil, nil, nil, nil, nil, nil)

	apiKey := &service.APIKey{ID: 101, UserID: 202, User: &service.User{ID: 202}}
	account := &service.Account{
		ID:       11,
		Platform: service.PlatformOpenAI,
		Credentials: map[string]any{
			"api_key": "sk-test",
			"model_mapping": map[string]any{
				"gpt-4.1-mini": "re-mapped-by-account",
			},
		},
	}
	exactUpstreamModel := "exact-upstream-model"
	fallbackModel := "gpt-4.1-mini"
	reqModel := "claude-3-5-sonnet-20241022"

	router := gin.New()
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/messages", func(c *gin.Context) {
		c.Set("openai_messages_fallback_model", fallbackModel)
		c.Set("openai_failed_usage_upstream_model", exactUpstreamModel)
		h.submitFailedUsageLog(c, apiKey, account, reqModel, false, 0, nil, nil, 0, nil, "handler.openai_gateway.messages")
		c.Status(http.StatusBadRequest)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-3-5-sonnet-20241022","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, exactUpstreamModel, *log.UpstreamModel)
}

func TestOpenAIGatewayHandler_MessagesUpstreamErrorStillCreatesUsageLog(t *testing.T) {
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
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
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
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		delay: 5 * time.Millisecond,
		response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_failed_messages_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"messages upstream rejected payload"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, cfg)
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
	router.POST("/v1/messages", h.Messages)

	reqBody := `{"model":"claude-3-5-sonnet-20241022","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"output_config":{"effort":"high"}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created for non-failover errors")
	require.NotNil(t, log.DurationMs)
	require.Greater(t, *log.DurationMs, 0)
	require.NotNil(t, log.ReasoningEffort)
	require.Equal(t, "high", *log.ReasoningEffort)
	require.NotNil(t, log.DetailSnapshot)
	require.JSONEq(t, reqBody, log.DetailSnapshot.RequestBody)
	require.Contains(t, log.DetailSnapshot.ResponseBody, "messages upstream rejected payload")
	require.Contains(t, log.DetailSnapshot.UpstreamRequestHeaders, "Authorization: Bearer sk-test")
}

func TestOpenAIGatewayHandler_MessagesFailoverExhaustedStillCreatesUsageLog(t *testing.T) {
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
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
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
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_messages_failover_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"openai_messages_rate_limited_raw","message":"openai messages raw failover"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, cfg)
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
	h.maxAccountSwitches = 0

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
	router.POST("/v1/messages", h.Messages)

	reqBody := `{"model":"claude-3-5-sonnet-20241022","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created when failover is exhausted")
	require.NotNil(t, log.DetailSnapshot)
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "Content-Type: application/json")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_messages_failover_123")
	require.Contains(t, log.DetailSnapshot.ResponseBody, `"openai_messages_rate_limited_raw"`)
	require.Contains(t, log.DetailSnapshot.ResponseBody, "openai messages raw failover")
}

func TestOpenAIGatewayHandler_MessagesSelectionExhaustedAfterFailoverStillCreatesUsageLog(t *testing.T) {
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
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
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
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_messages_selection_exhausted_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"openai_messages_rate_limited_raw","message":"openai messages raw failover"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, cfg)
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
	router.POST("/v1/messages", h.Messages)

	reqBody := `{"model":"claude-3-5-sonnet-20241022","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.JSONEq(t, `{"type":"error","error":{"type":"rate_limit_error","message":"Upstream rate limit exceeded, please retry later"}}`, rec.Body.String())
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created when selection is exhausted after failover")
	require.NotNil(t, log.DetailSnapshot)
	require.Contains(t, log.DetailSnapshot.ResponseBody, `"openai_messages_rate_limited_raw"`)
	require.Contains(t, log.DetailSnapshot.ResponseBody, "openai messages raw failover")
	require.Contains(t, log.DetailSnapshot.UpstreamRequestHeaders, "Authorization: Bearer sk-test")
}

func TestOpenAIGatewayHandler_UpstreamErrorStillCreatesUsageLog(t *testing.T) {
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
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
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
		delay: 5 * time.Millisecond,
		response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_failed_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"upstream rejected payload"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, cfg)
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
	h.maxAccountSwitches = 0

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
	router.POST("/v1/responses", h.Responses)

	reqBody := `{"model":"gpt-5.4","reasoning":{"effort":"high"},"stream":false,"input":"hello"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, 0, usageRepo.lastLog.InputTokens)
	require.Equal(t, 0, usageRepo.lastLog.OutputTokens)
	require.Equal(t, 0.0, usageRepo.lastLog.TotalCost)
	require.Equal(t, 0.0, usageRepo.lastLog.ActualCost)
	require.NotNil(t, usageRepo.lastLog.DurationMs)
	require.Greater(t, *usageRepo.lastLog.DurationMs, 0)
	require.NotNil(t, usageRepo.lastLog.ReasoningEffort)
	require.Equal(t, "high", *usageRepo.lastLog.ReasoningEffort)
	require.NotNil(t, usageRepo.lastLog.DetailSnapshot)
	require.JSONEq(t, reqBody, usageRepo.lastLog.DetailSnapshot.RequestBody)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "upstream rejected payload")
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.UpstreamRequestHeaders, "Authorization: Bearer sk-test")
}

func TestOpenAIGatewayHandler_ChatCompletionsUpstreamErrorStillCreatesUsageLog(t *testing.T) {
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
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
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
				"X-Request-Id": []string{"req_failed_chat_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"chat upstream rejected payload"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, cfg)
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
	router.POST("/chat/completions", h.ChatCompletions)

	reqBody := `{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":false}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, 0, usageRepo.lastLog.InputTokens)
	require.Equal(t, 0, usageRepo.lastLog.OutputTokens)
	require.Equal(t, 0.0, usageRepo.lastLog.TotalCost)
	require.Equal(t, 0.0, usageRepo.lastLog.ActualCost)
	require.NotNil(t, usageRepo.lastLog.DetailSnapshot)
	require.JSONEq(t, reqBody, usageRepo.lastLog.DetailSnapshot.RequestBody)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "chat upstream rejected payload")
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.UpstreamRequestHeaders, "Authorization: Bearer sk-test")
}

type openAIFailoverAccountRepoStub struct {
	openAIRetryAccountRepoStub
}

func (s *openAIFailoverAccountRepoStub) SetError(ctx context.Context, id int64, errorMsg string) error {
	return nil
}

func TestOpenAIGatewayHandler_FailoverExhaustedStillCreatesUsageLog(t *testing.T) {
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
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
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
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_failover_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"openai_rate_limited_raw","message":"openai raw failover"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, cfg)
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
	router.POST("/v1/responses", h.Responses)

	reqBody := `{"model":"gpt-5.4","stream":false,"input":"hello"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.NotNil(t, usageRepo.lastLog)
	require.NotNil(t, usageRepo.lastLog.DetailSnapshot)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseHeaders, "Content-Type: application/json")
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_failover_123")
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, `"openai_rate_limited_raw"`)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "openai raw failover")
}

func TestOpenAIGatewayHandler_ChatCompletionsFailoverExhaustedStillCreatesUsageLog(t *testing.T) {
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
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
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
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_chat_failover_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"openai_chat_rate_limited_raw","message":"openai chat raw failover"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, cfg)
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
	h.maxAccountSwitches = 0

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
	router.POST("/chat/completions", h.ChatCompletions)

	reqBody := `{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":false}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.NotNil(t, usageRepo.lastLog)
	require.NotNil(t, usageRepo.lastLog.DetailSnapshot)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, `"openai_chat_rate_limited_raw"`)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "openai chat raw failover")
}

func TestOpenAIGatewayHandler_RetrySuccessDoesNotReuseFailoverErrorSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			Scheduling:         config.GatewaySchedulingConfig{LoadBatchEnabled: false},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
		RateLimit:   config.RateLimitConfig{OAuth401CooldownMinutes: 1},
	}

	groupID := int64(1)
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account1 := &service.Account{ID: 11, Name: "openai-account-1", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-test-1"}}
	account2 := &service.Account{ID: 12, Name: "openai-account-2", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{"api_key": "sk-test-2"}}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{}
	httpUpstream := &openAIRetryTrackingHTTPUpstreamStub{
		responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_failover_disable_1"}},
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"organization has been disabled"}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_success_after_failover"}},
				Body:       io.NopCloser(strings.NewReader(`{"id":"resp_success","object":"response","status":"completed","model":"gpt-5.4","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Hello from success"}]}],"usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}`)),
			},
		},
	}
	accountRepo := &openAIFailoverAccountRepoStub{openAIRetryAccountRepoStub{accounts: []*service.Account{account1, account2}}}
	rateLimitService := service.NewRateLimitService(accountRepo, nil, cfg, nil, nil)
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, cfg)
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
		rateLimitService,
		billingCacheService,
		httpUpstream,
		deferredService,
		nil,
		nil,
		nil,
		nil,
	)
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, cfg)

	apiKey := &service.APIKey{ID: 101, UserID: 202, Status: service.StatusActive, GroupID: &groupID, User: &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1}, Group: group}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/responses", h.Responses)

	reqBody := `{"model":"gpt-5.4","stream":false,"input":"hello"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, usageRepo.lastLog)
	require.NotNil(t, usageRepo.lastLog.DetailSnapshot)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "Hello from success")
	require.NotContains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "organization has been disabled")
}
