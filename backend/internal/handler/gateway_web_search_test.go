//go:build unit

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
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
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

type xSearchHTTPUpstream struct {
	service.HTTPUpstream

	bodies       [][]byte
	responseBody string
}

func (u *xSearchHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	u.bodies = append(u.bodies, body)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(u.responseBody)),
	}, nil
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
	router.Use(middleware.UsageDetailCapture())
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
				"model_mapping": map[string]any{"grok-4.6": "first-search-model"},
			},
		},
		{
			ID: 4203, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
			Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2,
			Credentials: map[string]any{
				"api_key": "second-key", "base_url": "https://api.x.ai/v1",
				"model_mapping": map[string]any{"grok-4.6": "second-search-model"},
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
	router.Use(middleware.UsageDetailCapture())
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
		require.NotNil(t, usageLog.DetailSnapshot)
		requireRequestPreviewSnapshot(t, usageLog.DetailSnapshot.RequestBody, `{"query":"latest news","max_results":5}`)
		require.Contains(t, usageLog.DetailSnapshot.ResponseBody, `"query":"latest news"`)
		require.Contains(t, usageLog.DetailSnapshot.ResponseBody, `"results"`)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for native search usage record")
	}
}

func TestGatewayHandler_XSearchRecordsSourcesUsageAndSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	searchPrice := 1000.0
	group := &service.Group{ID: 4301, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, SearchPricePer1k: &searchPrice}
	account := &service.Account{
		ID: 4302, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "x-search-key", "base_url": "https://api.x.ai/v1"},
	}
	cfg := &config.Config{
		RunMode:     config.RunModeSimple,
		Default:     config.DefaultConfig{RateMultiplier: 1},
		Gateway:     config.GatewayConfig{Scheduling: config.GatewaySchedulingConfig{LoadBatchEnabled: false}},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}
	accountRepo := &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account}}}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	cache := newInMemoryGatewayCache()
	concurrency := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCache := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCache.Stop)
	upstream := &xSearchHTTPUpstream{responseBody: `{"id":"resp_xs_1","status":"completed","output":[{"type":"x_search_call","action":{"sources":[{"url":"https://x.com/xai/status/1","title":"xAI update","snippet":"release note"}]}}]}`}
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
		ID: 4303, UserID: 4304, Status: service.StatusActive, GroupID: &group.ID,
		User:  &service.User{ID: 4304, Status: service.StatusActive, Concurrency: 1},
		Group: group,
	}
	router := gin.New()
	router.Use(middleware.UsageDetailCapture())
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: 1})
		c.Next()
	})
	router.POST("/x_search", h.XSearch)

	requestBody := `{"query":"latest xAI updates","max_results":2,"allowed_x_handles":["xai"],"from_date":"2026-08-01"}`
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x_search", strings.NewReader(requestBody)).WithContext(context.Background())
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, "https://x.com/xai/status/1", gjson.GetBytes(recorder.Body.Bytes(), "results.0.url").String())
	require.Len(t, upstream.bodies, 1)
	require.Equal(t, "x_search", gjson.GetBytes(upstream.bodies[0], "tools.0.type").String())
	require.Equal(t, "x_search_call.action.sources", gjson.GetBytes(upstream.bodies[0], "include.0").String())

	select {
	case usageLog := <-usageRepo.created:
		require.Equal(t, "grok-x-search", usageLog.Model)
		require.True(t, strings.HasPrefix(usageLog.RequestID, "x_search:"))
		require.InDelta(t, 1, usageLog.TotalCost, 1e-9, "SearchCount must contribute exactly one search-price unit")
		require.NotNil(t, usageLog.DetailSnapshot)
		requireRequestPreviewSnapshot(t, usageLog.DetailSnapshot.RequestBody, requestBody)
		require.Contains(t, gjson.Get(usageLog.DetailSnapshot.RequestBody, "preview").String(), `"allowed_x_handles":["xai"]`)
		require.Contains(t, usageLog.DetailSnapshot.ResponseBody, "https://x.com/xai/status/1")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for x_search usage record")
	}
}

func TestGatewayHandler_XSearchAuditRejectsBeforeDispatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{RunMode: config.RunModeSimple, Default: config.DefaultConfig{RateMultiplier: 1}}
	billingCache := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCache.Stop)
	engine := blockingHandlerPromptEngine()
	h := &GatewayHandler{
		billingCacheService:      billingCache,
		securityAuditCoordinator: securityaudit.NewCoordinator(nil, engine),
	}
	group := &service.Group{ID: 4311, Platform: service.PlatformGrok, Status: service.StatusActive}
	apiKey := &service.APIKey{
		ID: 4312, UserID: 4313, Status: service.StatusActive, GroupID: &group.ID,
		User:  &service.User{ID: 4313, Status: service.StatusActive, Concurrency: 1},
		Group: group,
	}
	router := gin.New()
	router.Use(middleware.UsageDetailCapture())
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: 1})
		c.Next()
	})
	router.POST("/x_search", h.XSearch)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x_search", strings.NewReader(`{"query":"blocked x search"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusForbidden, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), securityaudit.ErrorCodeBlocked)
	evaluated, _, requests := engine.snapshot()
	require.Equal(t, 1, evaluated)
	require.Len(t, requests, 1)
	require.Contains(t, string(requests[0].Body), "blocked x search")
}
