package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
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
	h := NewOpenAIGatewayHandler(gatewayService, nil, nil, nil, nil, nil, nil, nil, nil, nil)

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
	h := NewOpenAIGatewayHandler(gatewayService, nil, nil, nil, nil, nil, nil, nil, nil, nil)

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
	h := NewOpenAIGatewayHandler(gatewayService, nil, nil, nil, nil, nil, nil, nil, nil, nil)

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

func TestOpenAIGatewayHandler_SubmitFailedUsageLog_PreservesCompositeModelTriplet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	gatewayService := service.NewOpenAIGatewayService(nil, usageRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := NewOpenAIGatewayHandler(gatewayService, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	apiKey := &service.APIKey{ID: 101, UserID: 202, User: &service.User{ID: 202}}
	account := &service.Account{ID: 11, Platform: service.PlatformOpenAI, Credentials: map[string]any{"api_key": "sk-test"}}

	router := gin.New()
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/responses", func(c *gin.Context) {
		c.Request = c.Request.WithContext(service.WithCompositeRouteDecision(c.Request.Context(), service.CompositeRouteDecision{
			Matched:        true,
			Source:         service.CompositeRouteSourceExplicit,
			PublicModel:    "public-alias",
			TargetPlatform: service.PlatformOpenAI,
			UpstreamModel:  "gpt-5",
		}))
		setOpenAIFailedUsageExactUpstreamModel(c, "gpt-5.2")
		h.submitFailedUsageLog(c, apiKey, account, "gpt-5", false, 0, nil, nil, 0, nil, "handler.openai_gateway.responses")
		c.Status(http.StatusBadGateway)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"public-alias"}`))
	router.ServeHTTP(rec, req)

	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.Equal(t, "public-alias", log.RequestedModel)
	require.Equal(t, "gpt-5", log.Model)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, "gpt-5.2", *log.UpstreamModel)
}

func TestOpenAIGatewayHandler_SubmitFailedUsageLogSnapshotsCompositeModelsBeforeQueue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	gatewayService := service.NewOpenAIGatewayService(nil, usageRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	pool := newUsageRecordTestPool(t)
	h := &OpenAIGatewayHandler{gatewayService: gatewayService, usageRecordWorkerPool: pool}

	block := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(block) }) }
	t.Cleanup(release)
	started := make(chan struct{})
	pool.Submit(func(context.Context) {
		close(started)
		<-block
	})
	<-started

	apiKey := &service.APIKey{ID: 101, UserID: 202, User: &service.User{ID: 202}}
	account := &service.Account{ID: 11, Platform: service.PlatformOpenAI}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Request = c.Request.WithContext(service.WithCompositeRouteDecision(c.Request.Context(), service.CompositeRouteDecision{
		Matched:        true,
		Source:         service.CompositeRouteSourceExplicit,
		PublicModel:    "public-alias",
		TargetPlatform: service.PlatformOpenAI,
		UpstreamModel:  "gpt-5",
	}))
	setOpenAIFailedUsageExactUpstreamModel(c, "gpt-5.2")

	h.submitFailedUsageLog(c, apiKey, account, "gpt-5", false, 0, nil, nil, 0, nil, "handler.openai_gateway.responses")

	c.Request = httptest.NewRequest(http.MethodPost, "/v1/other", nil)
	setOpenAIFailedUsageExactUpstreamModel(c, "reused-upstream-model")
	release()

	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.Equal(t, "public-alias", log.RequestedModel)
	require.Equal(t, "gpt-5", log.Model)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, "gpt-5.2", *log.UpstreamModel)
}

func TestSetOpenAIFailedUsageExactUpstreamModelUpdatesOpsContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	setOpenAIFailedUsageExactUpstreamModel(c, "gpt-5.2")

	require.Equal(t, "gpt-5.2", c.GetString(opsUpstreamModelKey))
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)

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
	require.Contains(t, rec.Body.String(), `"type":"invalid_request_error"`)
	require.Contains(t, rec.Body.String(), "messages upstream rejected payload")
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created for non-failover errors")
	require.NotNil(t, log.DurationMs)
	require.Greater(t, *log.DurationMs, 0)
	require.NotNil(t, log.ReasoningEffort)
	require.Equal(t, "high", *log.ReasoningEffort)
	require.NotNil(t, log.DetailSnapshot)
	requireRequestPreviewSnapshot(t, log.DetailSnapshot.RequestBody, reqBody)
	require.Contains(t, log.DetailSnapshot.ResponseBody, "messages upstream rejected payload")
	require.Contains(t, log.DetailSnapshot.UpstreamRequestHeaders, "Authorization: Bearer sk-test")
}

func TestOpenAIGatewayHandler_MessagesUsesEffectiveClaudeCodeFallbackGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initialID, fallbackID := int64(801), int64(802)
	initialGroup := &service.Group{ID: initialID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, ClaudeCodeOnly: true, FallbackGroupID: &fallbackID}
	fallbackGroup := &service.Group{ID: fallbackID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
	account := &service.Account{ID: 803, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	cfg := &config.Config{RunMode: config.RunModeStandard, Default: config.DefaultConfig{RateMultiplier: 1}, Concurrency: config.ConcurrencyConfig{PingInterval: 0}}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	user := &service.User{ID: 805, Status: service.StatusActive, Concurrency: 1, Balance: 1}
	billingCacheService := service.NewBillingCacheService(nil, &effectiveGroupUserRepoStub{user: user}, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCacheService.Stop)
	gatewayService := service.NewOpenAIGatewayService(
		accountRepo, usageRepo, nil, nil, nil, nil, openAIChatCompletionsGatewayCacheStub{}, cfg, nil,
		concurrencyService, service.NewBillingService(cfg, nil), nil, billingCacheService,
		&openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"upstream rejected"}}`))}},
		service.NewDeferredService(accountRepo, nil, 0), nil,
	)
	groupRepo := terminalUsageGroupRepo{groups: map[int64]*service.Group{initialID: initialGroup, fallbackID: fallbackGroup}}
	apiKeyService := service.NewAPIKeyService(nil, nil, groupRepo, nil, nil, nil, cfg)
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, apiKeyService, nil, nil, nil, nil, cfg, nil)
	apiKey := &service.APIKey{ID: 804, UserID: 805, GroupID: &initialID, User: user, Group: initialGroup}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		c.Next()
	})
	router.POST("/v1/messages", h.Messages)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"gpt-5","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, accountRepo.groupIDs, fallbackID)
	require.NotContains(t, accountRepo.groupIDs, initialID)
}

func TestOpenAIGatewayHandler_CountTokensUsesEffectiveClaudeCodeFallbackGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initialID, fallbackID := int64(811), int64(812)
	initialGroup := &service.Group{ID: initialID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, ClaudeCodeOnly: true, FallbackGroupID: &fallbackID}
	fallbackGroup := &service.Group{ID: fallbackID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
	account := &service.Account{ID: 813, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	cfg := &config.Config{RunMode: config.RunModeStandard, Default: config.DefaultConfig{RateMultiplier: 1}, Concurrency: config.ConcurrencyConfig{PingInterval: 0}}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	user := &service.User{ID: 815, Status: service.StatusActive, Concurrency: 1, Balance: 1}
	billingCacheService := service.NewBillingCacheService(nil, &effectiveGroupUserRepoStub{user: user}, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCacheService.Stop)
	gatewayService := service.NewOpenAIGatewayService(
		accountRepo, nil, nil, nil, nil, nil, openAIChatCompletionsGatewayCacheStub{}, cfg, nil,
		concurrencyService, service.NewBillingService(cfg, nil), nil, billingCacheService,
		&openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"input_tokens":3}`))}},
		service.NewDeferredService(accountRepo, nil, 0), nil,
	)
	groupRepo := terminalUsageGroupRepo{groups: map[int64]*service.Group{initialID: initialGroup, fallbackID: fallbackGroup}}
	apiKeyService := service.NewAPIKeyService(nil, nil, groupRepo, nil, nil, nil, cfg)
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, apiKeyService, nil, nil, nil, nil, cfg, nil)
	apiKey := &service.APIKey{ID: 814, UserID: 815, GroupID: &initialID, User: user, Group: initialGroup}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		c.Next()
	})
	router.POST("/v1/messages/count_tokens", h.CountTokens)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", strings.NewReader(`{"model":"gpt-5","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(3), gjson.Get(rec.Body.String(), "input_tokens").Int())
	require.Contains(t, accountRepo.groupIDs, fallbackID)
	require.NotContains(t, accountRepo.groupIDs, initialID)
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)
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

func TestOpenAIGatewayHandler_ImagesForwardFailedUsageLogCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, usageRepo, cleanup := newOpenAIImagesHandlerTestRouter(t, "/v1/images/generations", &http.Response{
		StatusCode: http.StatusBadRequest,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"req_image_failed_123"},
		},
		Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"images upstream rejected payload"}}`)),
	})
	defer cleanup()
	usageRepo.created = make(chan *service.UsageLog, 1)

	reqBody := `{"model":"gpt-image-2","prompt":"draw a lantern","size":"1024x1024"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.NotContains(t, rec.Body.String(), "event:", "non-stream OAuth images 4xx response must not append SSE after JSON fallback")
	var clientBody map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &clientBody), "client response must remain a single JSON object")
	require.Equal(t, "invalid_request_error", gjson.Get(rec.Body.String(), "error.type").String())
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created for non-failover image errors")
	require.NotNil(t, log.DurationMs)
	require.NotNil(t, log.DetailSnapshot)
	requireRequestPreviewSnapshot(t, log.DetailSnapshot.RequestBody, reqBody)
	require.Contains(t, log.DetailSnapshot.UpstreamRequestHeaders, ":method: POST")
	require.Contains(t, log.DetailSnapshot.UpstreamRequestHeaders, "/v1/images/generations")
	require.Contains(t, log.DetailSnapshot.UpstreamRequestHeaders, "Authorization: Bearer sk-test")
	require.JSONEq(t, reqBody, gjson.Get(log.DetailSnapshot.UpstreamRequestBody, "preview").String())
	require.Contains(t, log.DetailSnapshot.ResponseBody, "images upstream rejected payload")
	require.NotNil(t, log.InboundEndpoint)
	require.Equal(t, "/v1/images/generations", *log.InboundEndpoint)
	require.NotNil(t, log.UpstreamEndpoint)
	require.Contains(t, *log.UpstreamEndpoint, "/v1/images/generations")
}

func TestOpenAIGatewayHandler_ImagesEditMultipartForwardFailedUsageLogUsesMetadataSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, usageRepo, cleanup := newOpenAIImagesHandlerTestRouter(t, "/v1/images/edits", &http.Response{
		StatusCode: http.StatusBadRequest,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"req_image_edit_failed_123"},
		},
		Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"images edit upstream rejected payload"}}`)),
	})
	defer cleanup()
	usageRepo.created = make(chan *service.UsageLog, 1)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "replace background"))
	require.NoError(t, writer.WriteField("size", "1536x1024"))
	require.NoError(t, writer.WriteField("quality", "high"))
	require.NoError(t, writer.WriteField("background", "transparent"))
	require.NoError(t, writer.WriteField("output_format", "webp"))
	require.NoError(t, writer.WriteField("moderation", "low"))
	require.NoError(t, writer.WriteField("n", "2"))
	imagePart, err := writer.CreateFormFile("image", "source.png")
	require.NoError(t, err)
	_, err = imagePart.Write([]byte("raw-source-image-bytes"))
	require.NoError(t, err)
	maskPart, err := writer.CreateFormFile("mask", "mask.png")
	require.NoError(t, err)
	_, err = maskPart.Write([]byte("raw-mask-bytes"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var clientBody map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &clientBody), "client response must remain a single JSON object")
	require.Equal(t, "invalid_request_error", gjson.Get(rec.Body.String(), "error.type").String())
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created for multipart edit errors")
	require.NotNil(t, log.DetailSnapshot)
	require.NotContains(t, log.DetailSnapshot.RequestBody, "raw-source-image-bytes")
	require.NotContains(t, log.DetailSnapshot.RequestBody, "raw-mask-bytes")
	requestMetadata := gjson.Get(log.DetailSnapshot.RequestBody, "preview").String()
	require.Equal(t, "gpt-image-2", gjson.Get(requestMetadata, "model").String())
	require.Empty(t, gjson.Get(requestMetadata, "prompt").String())
	require.Equal(t, "1536x1024", gjson.Get(requestMetadata, "size").String())
	require.Equal(t, "high", gjson.Get(requestMetadata, "quality").String())
	require.Equal(t, "transparent", gjson.Get(requestMetadata, "background").String())
	require.Equal(t, "webp", gjson.Get(requestMetadata, "output_format").String())
	require.Equal(t, "low", gjson.Get(requestMetadata, "moderation").String())
	require.Equal(t, int64(2), gjson.Get(requestMetadata, "n").Int())
	require.True(t, gjson.Get(requestMetadata, "had_source_image").Bool())
	require.True(t, gjson.Get(requestMetadata, "had_mask").Bool())
	require.Contains(t, log.DetailSnapshot.ResponseBody, "images edit upstream rejected payload")
}

func TestOpenAIGatewayHandler_ImagesOAuthForwardFailedUsagePreservesUpstreamSnapshot(t *testing.T) {
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
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	account := &service.Account{
		ID:          11,
		Name:        "openai-oauth-image-account",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "acct_test",
		},
		Extra: map[string]any{
			"openai_device_id":  "device_test",
			"openai_session_id": "session_test",
		},
	}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_oauth_image_failed_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"message":"oauth images upstream rejected payload"}}`)),
		},
	}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)

	apiKey := &service.APIKey{
		ID:      101,
		UserID:  202,
		Status:  service.StatusActive,
		GroupID: &groupID,
		User:    &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1},
		Group:   group,
	}

	reqBody := `{"model":"gpt-image-2","prompt":"draw a lantern"}`

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/images/generations", h.Images)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.DetailSnapshot)
	requireRequestPreviewSnapshot(t, log.DetailSnapshot.RequestBody, reqBody)
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, ":status: 400")
	require.NotContains(t, log.DetailSnapshot.ResponseHeaders, ":status: 502")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "Content-Type: application/json")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_oauth_image_failed_123")
	require.Contains(t, log.DetailSnapshot.ResponseBody, "oauth images upstream rejected payload")
	require.NotEmpty(t, strings.TrimSpace(log.DetailSnapshot.ResponseBody))
}

func TestOpenAIGatewayHandler_ImagesOAuthForwardFailedUsageUsesOriginalUpstreamSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode:     config.RunModeSimple,
		Default:     config.DefaultConfig{RateMultiplier: 1},
		Gateway:     config.GatewayConfig{MaxAccountSwitches: 1, Scheduling: config.GatewaySchedulingConfig{LoadBatchEnabled: false}},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	groupID := int64(1)
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	account := &service.Account{
		ID:          11,
		Name:        "openai-oauth-image-account",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{"access_token": "oauth-token", "chatgpt_account_id": "acct_test"},
		Extra:       map[string]any{"openai_device_id": "device_test", "openai_session_id": "session_test"},
	}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header: http.Header{
			"Content-Type":        []string{"application/json"},
			"X-Request-Id":        []string{"req_oauth_image_original_snapshot"},
			"X-Upstream-Debug-Id": []string{"upstream-only-debug-id"},
		},
		Body: io.NopCloser(strings.NewReader(`{"error":{"message":"oauth images upstream rejected payload","type":"invalid_request_error","param":"size","code":"bad_size"},"upstream_only":"raw-marker"}`)),
	}}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)

	apiKey := &service.APIKey{ID: 101, UserID: 202, Status: service.StatusActive, GroupID: &groupID, User: &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1}, Group: group}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/images/generations", h.Images)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"gpt-image-2","prompt":"draw a lantern"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.DetailSnapshot)
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Upstream-Debug-Id: upstream-only-debug-id")
	require.Contains(t, log.DetailSnapshot.ResponseBody, `"upstream_only":"raw-marker"`)
}

func TestOpenAIGatewayHandler_ImagesFailoverExhaustedFailedUsageLogCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, usageRepo, cleanup := newOpenAIImagesHandlerTestRouter(t, "/v1/images/generations", &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"req_image_failover_123"},
		},
		Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","message":"images upstream overloaded"}}`)),
	})
	defer cleanup()
	usageRepo.created = make(chan *service.UsageLog, 1)

	reqBody := `{"model":"gpt-image-2","prompt":"draw a lantern","size":"1024x1024"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created when image failover is exhausted")
	require.NotNil(t, log.DurationMs)
	require.NotNil(t, log.DetailSnapshot)
	requireRequestPreviewSnapshot(t, log.DetailSnapshot.RequestBody, reqBody)
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, ":status: 429")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "Content-Type: application/json")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_image_failover_123")
	require.Contains(t, log.DetailSnapshot.ResponseBody, "images upstream overloaded")
	require.NotNil(t, log.InboundEndpoint)
	require.Equal(t, "/v1/images/generations", *log.InboundEndpoint)
	require.NotNil(t, log.UpstreamEndpoint)
	require.Contains(t, *log.UpstreamEndpoint, "/v1/images/generations")
}

func TestOpenAIGatewayHandler_SubmitOpenAIImagesFailedUsageLog_UsesErrorSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
	}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	gatewayService := service.NewOpenAIGatewayService(
		nil,
		usageRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		cfg,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	h := &OpenAIGatewayHandler{gatewayService: gatewayService}

	groupID := int64(1)
	apiKey := &service.APIKey{
		ID:      101,
		UserID:  202,
		Status:  service.StatusActive,
		GroupID: &groupID,
		User:    &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1},
		Group:   &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true},
	}
	account := &service.Account{ID: 11, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive}
	parsed := &service.OpenAIImagesRequest{Endpoint: "/v1/images/generations", Model: "gpt-image-2", Prompt: "draw a lantern", N: 1}

	router := gin.New()
	router.Use(middleware.UsageDetailCapture())
	router.POST("/test", func(c *gin.Context) {
		h.submitFailedUsageLog(c, apiKey, account, parsed.Model, parsed.Stream, http.StatusTeapot, http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"req_err_snapshot_123"},
		}, []byte(`{"error":{"message":"err-carried image snapshot"}}`), time.Second, nil, "handler.openai_gateway.images")
		c.Status(http.StatusTeapot)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"model":"gpt-image-2","prompt":"draw a lantern"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.DetailSnapshot)
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, ":status: 418")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "Content-Type: application/json")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_err_snapshot_123")
	require.Contains(t, log.DetailSnapshot.ResponseBody, "err-carried image snapshot")
}

func TestOpenAIGatewayHandler_OpenAIImagesFailedUsageLogSnapshotsRequestBeforeQueue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	gatewayService := service.NewOpenAIGatewayService(nil, usageRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	pool := newUsageRecordTestPool(t)
	h := &OpenAIGatewayHandler{gatewayService: gatewayService, usageRecordWorkerPool: pool}

	block := make(chan struct{})
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() { close(block) })
	}
	t.Cleanup(release)
	started := make(chan struct{})
	pool.Submit(func(context.Context) {
		close(started)
		<-block
	})
	<-started

	apiKey := &service.APIKey{ID: 101, UserID: 202, User: &service.User{ID: 202}}
	account := &service.Account{ID: 11, Platform: service.PlatformOpenAI}
	parsed := &service.OpenAIImagesRequest{Model: "original-model", Stream: true}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	c.Request.Header.Set("User-Agent", "original-user-agent")
	c.Set(service.OpenAIFailedUsageUpstreamModelKey, "original-upstream-model")

	h.submitOpenAIImagesFailedUsageLogWithResponse(c, apiKey, account, parsed, 0, nil, nil, time.Second)

	parsed.Model = "reused-model"
	parsed.Stream = false
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/other", nil)
	c.Request.Header.Set("User-Agent", "reused-user-agent")
	c.Set(service.OpenAIFailedUsageUpstreamModelKey, "reused-upstream-model")
	release()

	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.Equal(t, "original-model", log.Model)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, "original-upstream-model", *log.UpstreamModel)
	require.True(t, log.Stream)
	require.NotNil(t, log.InboundEndpoint)
	require.Equal(t, "/v1/images/generations", *log.InboundEndpoint)
	require.NotNil(t, log.UserAgent)
	require.Equal(t, "original-user-agent", *log.UserAgent)
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)

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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)
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

	require.Equal(t, http.StatusBadRequest, rec.Code)
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
	requireRequestPreviewSnapshot(t, usageRepo.lastLog.DetailSnapshot.RequestBody, reqBody)
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)

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
	requireRequestPreviewSnapshot(t, usageRepo.lastLog.DetailSnapshot.RequestBody, reqBody)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "chat upstream rejected payload")
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.UpstreamRequestHeaders, "Authorization: Bearer sk-test")
}

func TestOpenAIGatewayHandler_ChatCompletionsHashesBeforeChannelMapping(t *testing.T) {
	cache := &openAIResponsesRetentionGatewayCache{blocked: make(chan string, 1)}
	settings := service.NewSettingService(&openAIResponsesRetentionSettingRepo{}, &config.Config{})
	groupID := int64(1)
	channelService := service.NewChannelService(openAIFailedUsageChannelRepoStub{
		channel: service.Channel{
			ID: 21, Status: service.StatusActive, GroupIDs: []int64{groupID},
			ModelMapping: map[string]map[string]string{service.PlatformOpenAI: {"client-model": "mapped-model"}},
		},
		groupPlatforms: map[int64]string{groupID: service.PlatformOpenAI},
	}, nil, nil, nil)
	env := newOpenAIResponsesRetentionTestEnv(t, &service.Group{MaxReasoningEffort: "high"}, cache, nil, settings, channelService, nil)
	env.billingRepo.applied = make(chan *service.UsageBillingCommand, 1)
	env.upstream.responses = []*http.Response{{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","code":"cyber_policy","message":"blocked"}}`)),
	}}
	rawBody := []byte(`{"model":"client-model","reasoning_effort":"xhigh","prompt_cache_key":"chat-session","messages":[{"role":"user","content":"hello"}]}`)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(rawBody))
	request.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	require.Len(t, env.upstream.requests, 1)
	upstreamBody, err := io.ReadAll(env.upstream.requests[0].Body)
	require.NoError(t, err)
	var recordedUsageHash string
	select {
	case cmd := <-env.billingRepo.applied:
		recordedUsageHash = cmd.RequestPayloadHash
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for cyber usage billing")
	}
	var recordedCyberKey string
	select {
	case recordedCyberKey = <-cache.blocked:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for cyber session block")
	}
	apiKey := env.apiKey
	requestContext := env.requestContext
	policyBody, changed := service.ApplyOpenAIReasoningEffortPolicy(rawBody, "high", nil)
	require.True(t, changed)
	wantUsageHash := service.HashUsageRequestPayload(policyBody)
	wantCyberKey := service.CyberSessionBlockKey(apiKey.ID, requestContext, rawBody)

	require.Equal(t, "mapped-model", gjson.GetBytes(upstreamBody, "model").String())
	require.Equal(t, "high", gjson.GetBytes(upstreamBody, "reasoning.effort").String())
	require.Equal(t, wantUsageHash, recordedUsageHash)
	require.Equal(t, wantCyberKey, recordedCyberKey)
}

func TestOpenAIGatewayHandler_ResponsesCyberPolicyCreatesSingleUsageLog(t *testing.T) {
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
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 4)}
	var createdCount atomic.Int32
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"X-Request-Id": []string{"req_cyber_123"},
			},
			Body: io.NopCloser(strings.NewReader(strings.Join([]string{
				`data: {"type":"response.created","response":{"id":"resp_cyber_123"}}`,
				"",
				`data: {"type":"response.failed","response":{"id":"resp_cyber_123","error":{"code":"cyber_policy","message":"cyber upstream rejected payload"},"usage":{"input_tokens":17,"output_tokens":3}}}`,
				"",
			}, "\n"))),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)
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

	reqBody := `{"model":"gpt-5.4","stream":true,"input":"hello cyber"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	time.Sleep(200 * time.Millisecond)
	for {
		select {
		case <-usageRepo.created:
			createdCount.Add(1)
		default:
			require.Equal(t, http.StatusOK, rec.Code)
			require.NotNil(t, usageRepo.lastLog)
			require.Equal(t, int32(1), createdCount.Load(), "cyber request should create exactly one usage log")
			require.Equal(t, 17, usageRepo.lastLog.InputTokens)
			require.Equal(t, 3, usageRepo.lastLog.OutputTokens)
			return
		}
	}
}

type effectiveGroupUserRepoStub struct {
	service.UserRepository

	user *service.User
}

func (s *effectiveGroupUserRepoStub) GetByID(ctx context.Context, id int64) (*service.User, error) {
	if s.user != nil && s.user.ID == id {
		return s.user, nil
	}
	return nil, service.ErrUserNotFound
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)

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

func TestOpenAIGatewayHandler_ResponsesFailedUsageUsesFinalOutboundModel(t *testing.T) {
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
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "acct_test",
			"model_mapping": map[string]any{
				"gpt-client":  "gpt-channel",
				"gpt-channel": "gpt-5.4-high",
			},
		},
	}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_responses_attempt_effort_123"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"openai_rate_limited_raw","message":"openai raw failover"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })
	channelService := service.NewChannelService(openAIFailedUsageChannelRepoStub{
		channel: service.Channel{
			ID:       21,
			Status:   service.StatusActive,
			GroupIDs: []int64{groupID},
			ModelMapping: map[string]map[string]string{
				service.PlatformOpenAI: {"gpt-client": "gpt-channel"},
			},
		},
		groupPlatforms: map[int64]string{groupID: service.PlatformOpenAI},
	}, nil, nil, nil)

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
		channelService,
		nil,
	)
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)

	apiKey := &service.APIKey{ID: 101, UserID: 202, Status: service.StatusActive, GroupID: &groupID, User: &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1}, Group: group}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/responses", h.Responses)

	reqBody := `{"model":"gpt-client","stream":false,"input":"hello"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, "gpt-5.4", gjson.GetBytes(httpUpstream.requestBody, "model").String())
	require.Equal(t, gjson.GetBytes(httpUpstream.requestBody, "model").String(), *log.UpstreamModel)
}

func TestOpenAIGatewayHandler_GrokResponsesFailedUsageUsesFinalOutboundModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		RunMode:     config.RunModeSimple,
		Default:     config.DefaultConfig{RateMultiplier: 1},
		Gateway:     config.GatewayConfig{MaxAccountSwitches: 0, Scheduling: config.GatewaySchedulingConfig{LoadBatchEnabled: false}},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}
	groupID := int64(2)
	group := &service.Group{ID: groupID, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID: 12, Name: "grok-test-account", Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{
			"api_key":             "xai-test",
			"openai_capabilities": []any{"chat_completions"},
			"model_mapping":       map[string]any{"grok-client": "grok-account-routing", "grok-channel": "grok-4.3"},
		},
		Extra: map[string]any{"use_responses_api": true},
	}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 2)}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_grok_final_model"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","message":"grok overloaded"}}`)),
	}}
	accountRepo := &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account}}}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })
	channelService := service.NewChannelService(openAIFailedUsageChannelRepoStub{
		channel: service.Channel{
			ID: 22, Status: service.StatusActive, GroupIDs: []int64{groupID},
			ModelMapping: map[string]map[string]string{service.PlatformGrok: {"grok-client": "grok-channel"}},
		},
		groupPlatforms: map[int64]string{groupID: service.PlatformGrok},
	}, nil, nil, nil)
	gatewayService := service.NewOpenAIGatewayService(
		accountRepo, usageRepo, nil, nil, nil, nil, openAIChatCompletionsGatewayCacheStub{}, cfg, nil,
		concurrencyService, service.NewBillingService(cfg, nil), nil, billingCacheService, httpUpstream,
		service.NewDeferredService(accountRepo, nil, 0), nil, service.NewGrokTokenProvider(accountRepo, nil), channelService, nil,
	)
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)
	apiKey := &service.APIKey{ID: 101, UserID: 202, Status: service.StatusActive, GroupID: &groupID, User: &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1}, Group: group}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/responses", h.Responses)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"grok-client","stream":false,"input":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	require.GreaterOrEqual(t, recorder.Code, http.StatusBadRequest)
	require.Equal(t, "grok-4.3", gjson.GetBytes(httpUpstream.requestBody, "model").String())
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, gjson.GetBytes(httpUpstream.requestBody, "model").String(), *log.UpstreamModel)
	require.Len(t, usageRepo.created, 1, "failed attempt must create exactly one usage log")
}

func TestOpenAIGatewayHandler_Responses429FastStopCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 3,
			Scheduling:         config.GatewaySchedulingConfig{LoadBatchEnabled: false},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	groupID := int64(1)
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID:          11,
		Name:        "openai-oauth-test-account",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{"access_token": "oauth-token", "chatgpt_account_id": "acct_test"},
		Extra:       map[string]any{"openai_device_id": "device_test", "openai_session_id": "session_test"},
	}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_responses_fast_stop_429"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"fast_stop_429","message":"openai oauth fast stop"}}`)),
	}}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })
	channelService := service.NewChannelService(openAIFailedUsageChannelRepoStub{
		channel: service.Channel{
			ID:       21,
			Status:   service.StatusActive,
			GroupIDs: []int64{groupID},
			ModelMapping: map[string]map[string]string{
				service.PlatformOpenAI: {"gpt-5.4": "gpt-5.4-high"},
			},
		},
		groupPlatforms: map[int64]string{groupID: service.PlatformOpenAI},
	}, nil, nil, nil)

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
		channelService,
		nil,
	)
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)

	apiKey := &service.APIKey{ID: 101, UserID: 202, Status: service.StatusActive, GroupID: &groupID, User: &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1}, Group: group}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/responses", h.Responses)

	for range 20 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.4","stream":false,"input":"warm storm"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		gatewayService.ClearAccountSchedulingBlock(account.ID)
	}
	usageRepo.created = make(chan *service.UsageLog, 1)
	usageRepo.lastLog = nil

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.4","stream":false,"input":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created when OAuth 429 failover fast-stops")
	require.NotNil(t, log.ReasoningEffort)
	require.Equal(t, "high", *log.ReasoningEffort)
	require.NotNil(t, log.DetailSnapshot)
	require.Contains(t, log.DetailSnapshot.ResponseBody, "fast_stop_429")
}

type openAIFailedUsageChannelRepoStub struct {
	service.ChannelRepository

	channel        service.Channel
	groupPlatforms map[int64]string
}

func (s openAIFailedUsageChannelRepoStub) Create(context.Context, *service.Channel) error {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) GetByID(context.Context, int64) (*service.Channel, error) {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) Update(context.Context, *service.Channel) error {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) Delete(context.Context, int64) error { panic("unused") }
func (s openAIFailedUsageChannelRepoStub) List(context.Context, pagination.PaginationParams, string, string) ([]service.Channel, *pagination.PaginationResult, error) {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) ListAll(context.Context) ([]service.Channel, error) {
	return []service.Channel{s.channel}, nil
}
func (s openAIFailedUsageChannelRepoStub) ExistsByName(context.Context, string) (bool, error) {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) ExistsByNameExcluding(context.Context, string, int64) (bool, error) {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) GetGroupIDs(context.Context, int64) ([]int64, error) {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) SetGroupIDs(context.Context, int64, []int64) error {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) GetChannelIDByGroupID(context.Context, int64) (int64, error) {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) GetGroupsInOtherChannels(context.Context, int64, []int64) ([]int64, error) {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) GetGroupPlatforms(_ context.Context, groupIDs []int64) (map[int64]string, error) {
	out := make(map[int64]string, len(groupIDs))
	for _, id := range groupIDs {
		out[id] = s.groupPlatforms[id]
	}
	return out, nil
}
func (s openAIFailedUsageChannelRepoStub) ListModelPricing(context.Context, int64) ([]service.ChannelModelPricing, error) {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) CreateModelPricing(context.Context, *service.ChannelModelPricing) error {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) UpdateModelPricing(context.Context, *service.ChannelModelPricing) error {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) DeleteModelPricing(context.Context, int64) error {
	panic("unused")
}
func (s openAIFailedUsageChannelRepoStub) ReplaceModelPricing(context.Context, int64, []service.ChannelModelPricing) error {
	panic("unused")
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)
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
		rateLimitService,
		billingCacheService,
		httpUpstream,
		deferredService,
		nil,
		nil,
		nil,
		nil,
	)
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)

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
