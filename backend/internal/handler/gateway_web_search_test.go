//go:build unit

package handler

import (
	"context"
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

type grokWebSearchGroupRepo struct {
	service.GroupRepository
	group *service.Group
}

func (r grokWebSearchGroupRepo) GetByID(context.Context, int64) (*service.Group, error) {
	return r.group, nil
}

func (r grokWebSearchGroupRepo) GetByIDLite(context.Context, int64) (*service.Group, error) {
	return r.group, nil
}

func TestGatewayHandler_WebSearchReusesStandardSessionStickyAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 4211, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true}
	accounts := []*service.Account{
		{ID: 4212, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "first-key", "base_url": "https://api.x.ai/v1"}},
		{ID: 4213, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{"api_key": "second-key", "base_url": "https://api.x.ai/v1"}},
	}
	cfg := &config.Config{
		RunMode:     config.RunModeSimple,
		Default:     config.DefaultConfig{RateMultiplier: 1},
		Gateway:     config.GatewayConfig{Scheduling: config.GatewaySchedulingConfig{LoadBatchEnabled: false}},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}
	accountRepo := &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: accounts}}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	cache := newInMemoryGatewayCache()
	concurrency := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCache := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCache.Stop)
	upstream := &grokMediaRequestRecorder{}
	gateway := service.NewGatewayService(
		accountRepo, grokWebSearchGroupRepo{group: group}, usageRepo, nil, nil, nil, nil, cache, cfg, nil,
		concurrency, service.NewBillingService(cfg, nil), nil, billingCache, nil, upstream,
		service.NewDeferredService(accountRepo, nil, 0), nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil,
	)
	openAIGateway := service.NewOpenAIGatewayService(
		accountRepo, usageRepo, nil, nil, nil, nil, cache, cfg, nil, concurrency,
		service.NewBillingService(cfg, nil), nil, billingCache, upstream,
		service.NewDeferredService(accountRepo, nil, 0), nil, service.NewGrokTokenProvider(accountRepo, nil), nil, nil,
	)
	h := &GatewayHandler{
		gatewayService:       gateway,
		openAIGatewayService: openAIGateway,
		billingCacheService:  billingCache,
		apiKeyService:        &service.APIKeyService{},
		concurrencyHelper:    NewConcurrencyHelper(concurrency, SSEPingFormatNone, 0),
		cfg:                  cfg,
	}
	apiKey := &service.APIKey{
		ID: 4214, UserID: 4215, Status: service.StatusActive, GroupID: &group.ID,
		User:  &service.User{ID: 4215, Status: service.StatusActive, Concurrency: 1},
		Group: group,
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: 1})
		c.Next()
	})
	router.POST("/web_search", h.WebSearch)

	request := func() {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/web_search", strings.NewReader(`{"query":"sticky search"}`)).WithContext(context.Background())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("session_id", "search-sticky-session")
		router.ServeHTTP(recorder, req)
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		select {
		case <-usageRepo.created:
		case <-time.After(time.Second):
			t.Fatal("web search usage was not recorded")
		}
	}

	request()
	accounts[0].Priority, accounts[1].Priority = 2, 1
	request()

	upstream.mu.Lock()
	require.Equal(t, []int64{4212, 4212}, upstream.accountIDs)
	upstream.mu.Unlock()
	require.NotEmpty(t, cache.sessionKeys())
	require.NotContains(t, cache.sessionKeys(), "")
}

func TestGatewayHandler_WebSearchFailoverRecordsFinalMappedUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 4201, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true}
	accounts := []*service.Account{
		{
			ID: 4202, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
			Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
			Credentials: map[string]any{
				"api_key": "first-key", "base_url": "https://api.x.ai/v1",
				"model_mapping": map[string]any{"grok-4.5": "first-search-model"},
			},
		},
		{
			ID: 4203, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
			Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2,
			Credentials: map[string]any{
				"api_key": "second-key", "base_url": "https://api.x.ai/v1",
				"model_mapping": map[string]any{"grok-4.5": "second-search-model"},
			},
		},
	}
	cfg := &config.Config{
		RunMode:     config.RunModeSimple,
		Default:     config.DefaultConfig{RateMultiplier: 1},
		Gateway:     config.GatewayConfig{Scheduling: config.GatewaySchedulingConfig{LoadBatchEnabled: false}},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}
	accountRepo := &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: accounts}}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	cache := openAIChatCompletionsGatewayCacheStub{}
	concurrency := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCache := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCache.Stop)
	upstream := &grokMediaRequestRecorder{statuses: []int{http.StatusInternalServerError, http.StatusOK}}
	gateway := service.NewGatewayService(
		accountRepo, grokWebSearchGroupRepo{group: group}, usageRepo, nil, nil, nil, nil, cache, cfg, nil,
		concurrency, service.NewBillingService(cfg, nil), nil, billingCache, nil, upstream,
		service.NewDeferredService(accountRepo, nil, 0), nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil,
	)
	h := &GatewayHandler{
		gatewayService:      gateway,
		billingCacheService: billingCache,
		apiKeyService:       &service.APIKeyService{},
		concurrencyHelper:   NewConcurrencyHelper(concurrency, SSEPingFormatNone, 0),
		cfg:                 cfg,
	}
	apiKey := &service.APIKey{
		ID: 4204, UserID: 4205, Status: service.StatusActive, GroupID: &group.ID,
		User:  &service.User{ID: 4205, Status: service.StatusActive, Concurrency: 1},
		Group: group,
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: 1})
		c.Next()
	})
	router.POST("/web_search", h.WebSearch)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/web_search", strings.NewReader(`{"query":"latest news"}`)).WithContext(context.Background())
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	upstream.mu.Lock()
	require.Equal(t, []int64{4202, 4203}, upstream.accountIDs)
	require.Contains(t, string(upstream.bodies[0]), `"model":"first-search-model"`)
	require.Contains(t, string(upstream.bodies[1]), `"model":"second-search-model"`)
	upstream.mu.Unlock()
	select {
	case usageLog := <-usageRepo.created:
		require.Equal(t, "grok-web-search", usageLog.Model)
		require.NotNil(t, usageLog.UpstreamModel)
		require.Equal(t, "second-search-model", *usageLog.UpstreamModel)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for native search usage record")
	}
}
