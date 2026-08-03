package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIGatewayHandler_ResponsesFinalHandleIncludesRequestLevelRewrites(t *testing.T) {
	group := &service.Group{MaxReasoningEffort: "high"}
	env := newOpenAIResponsesRetentionTestEnv(t, group, nil, nil, nil)
	route := service.EffectiveGatewayRoute{
		APIKey: env.apiKey, Group: env.apiKey.Group, GroupID: env.apiKey.GroupID,
		ClientModel: "client-model", RoutingModel: "routed-model", UpstreamModel: "routed-model", Platform: service.PlatformOpenAI,
	}
	env.route = &route
	rawBody := []byte(`{"model":"client-model","stream":false,"input":"hello","reasoning":{"effort":"xhigh"}}`)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(string(rawBody)))
	request.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Len(t, env.upstream.requests, 1)
	upstreamBody, err := io.ReadAll(env.upstream.requests[0].Body)
	require.NoError(t, err)
	require.Equal(t, "routed-model", gjson.GetBytes(upstreamBody, "model").String())
	require.Equal(t, "high", gjson.GetBytes(upstreamBody, "reasoning.effort").String())
	require.NotNil(t, env.billingRepo.lastCmd)
	require.Equal(t, service.HashUsageRequestPayload(upstreamBody), env.billingRepo.lastCmd.RequestPayloadHash, string(upstreamBody))
}

func TestOpenAIGatewayHandler_ResponsesSessionAndCyberKeysUseRawBody(t *testing.T) {
	cache := &openAIResponsesRetentionGatewayCache{}
	settings := service.NewSettingService(&openAIResponsesRetentionSettingRepo{}, &config.Config{})
	env := newOpenAIResponsesRetentionTestEnv(t, nil, cache, nil, settings)
	rawBody := []byte(`{"model":"client-model","stream":false,"store":true,"prompt_cache_key":"raw-session","input":"hello"}`)
	route := service.EffectiveGatewayRoute{
		APIKey: env.apiKey, Group: env.apiKey.Group, GroupID: env.apiKey.GroupID,
		ClientModel: "client-model", RoutingModel: "routed-model", UpstreamModel: "routed-model", Platform: service.PlatformOpenAI,
	}
	env.route = &route

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses/compact", strings.NewReader(string(rawBody)))
	request.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Len(t, env.upstream.requests, 1)
	normalizedBody, err := io.ReadAll(env.upstream.requests[0].Body)
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(normalizedBody, "prompt_cache_key").Exists())
	require.False(t, gjson.GetBytes(normalizedBody, "store").Exists())
	require.False(t, gjson.GetBytes(normalizedBody, "stream").Exists())
	require.NotNil(t, env.requestContext)
	wantSessionHash := env.handler.gatewayService.GenerateSessionHash(env.requestContext, rawBody)
	require.Equal(t, wantSessionHash, strings.TrimPrefix(cache.sessionHash, service.PlatformOpenAI+":"))
	require.Equal(t, service.CyberSessionBlockKey(env.apiKey.ID, env.requestContext, rawBody), cache.cyberKey)

	grokRecorder := httptest.NewRecorder()
	grokContext, _ := gin.CreateTestContext(grokRecorder)
	grokContext.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{}`))
	grokContext.Request.Header.Set("X-Grok-Conv-Id", "grok-conversation")
	grokContext.Request = grokContext.Request.WithContext(service.WithResolvedTargetPlatform(grokContext.Request.Context(), service.PlatformGrok))
	require.NotEmpty(t, env.handler.gatewayService.GenerateSessionHash(grokContext, []byte(`{}`)))
	require.Empty(t, service.CyberSessionBlockKey(env.apiKey.ID, grokContext, []byte(`{}`)), "Grok-only header is outside the cyber block key contract")
	require.Equal(t, service.CyberSessionBlockKey(env.apiKey.ID, grokContext, rawBody), service.CyberSessionBlockKey(env.apiKey.ID, env.requestContext, rawBody))
}

func TestOpenAIGatewayHandler_ResponsesReadFailureReleasesAccountSlot(t *testing.T) {
	spoolDir := t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: spoolDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	cache := &concurrencyCacheMock{}
	cache.acquireUserSlotFn = func(context.Context, int64, int, string) (bool, error) { return true, nil }
	cache.acquireUserGroupSlotFn = func(context.Context, int64, int64, int, string) (bool, error) { return true, nil }
	cache.acquireAccountSlotFn = func(context.Context, int64, int, string) (bool, error) {
		entries, err := os.ReadDir(spoolDir)
		require.NoError(t, err)
		require.NotEmpty(t, entries)
		for _, entry := range entries {
			require.NoError(t, os.Remove(filepath.Join(spoolDir, entry.Name())))
		}
		return true, nil
	}
	env := newOpenAIResponsesRetentionTestEnv(t, nil, nil, cache, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5","stream":false,"input":"hello"}`))
	request.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code, recorder.Body.String())
	require.Equal(t, int32(1), atomic.LoadInt32(&cache.releaseAccountCalled))
	require.Empty(t, env.upstream.requests)
}

type openAIResponsesRetentionTestEnv struct {
	router         *gin.Engine
	handler        *OpenAIGatewayHandler
	apiKey         *service.APIKey
	billingRepo    *openAIResponsesRequestBodyRetentionBillingRepoStub
	upstream       *openAIRetryTrackingHTTPUpstreamStub
	route          *service.EffectiveGatewayRoute
	requestContext *gin.Context
}

func newOpenAIResponsesRetentionTestEnv(
	t *testing.T,
	groupOverrides *service.Group,
	gatewayCache service.GatewayCache,
	concurrencyCache service.ConcurrencyCache,
	settingService *service.SettingService,
) *openAIResponsesRetentionTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			Scheduling:         config.GatewaySchedulingConfig{LoadBatchEnabled: false},
			Sticky:             config.GatewayStickyConfig{OpenAI: config.GatewayStickyPlatformConfig{Enabled: true}},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}
	groupID := int64(1)
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	if groupOverrides != nil {
		group.MaxReasoningEffort = groupOverrides.MaxReasoningEffort
		group.ReasoningEffortMappings = groupOverrides.ReasoningEffortMappings
	}
	account := &service.Account{
		ID: 11, Name: "openai-test-account", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"openai_passthrough": true},
	}
	if gatewayCache == nil {
		gatewayCache = openAIChatCompletionsGatewayCacheStub{}
	}
	if concurrencyCache == nil {
		concurrencyCache = openAIChatCompletionsConcurrencyCacheStub{}
	}
	billingRepo := &openAIResponsesRequestBodyRetentionBillingRepoStub{}
	upstream := &openAIRetryTrackingHTTPUpstreamStub{responses: []*http.Response{{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_1","object":"response","status":"completed","model":"gpt-5","output":[],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2,"input_tokens_details":{"cached_tokens":0}}}`)),
	}}}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(concurrencyCache)
	billingCacheService := service.NewBillingCacheService(&openAIResponsesRequestBodyRetentionBillingCacheStub{}, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCacheService.Stop)
	gatewayService := service.NewOpenAIGatewayService(
		accountRepo, nil, billingRepo, nil, nil, nil, gatewayCache, cfg, nil, concurrencyService,
		service.NewBillingService(cfg, nil), nil, billingCacheService, upstream, service.NewDeferredService(accountRepo, nil, 0), nil,
		settingService,
	)
	handler := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)
	apiKey := &service.APIKey{
		ID: 101, UserID: 202, Status: service.StatusActive, GroupID: &groupID,
		User: &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1}, Group: group,
	}
	env := &openAIResponsesRetentionTestEnv{handler: handler, apiKey: apiKey, billingRepo: billingRepo, upstream: upstream}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		env.requestContext = c
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		if env.route != nil {
			c.Request = c.Request.WithContext(service.WithEffectiveGatewayRoute(c.Request.Context(), *env.route))
		}
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/responses", handler.Responses)
	router.POST("/v1/responses/compact", handler.Responses)
	env.router = router
	return env
}

type openAIResponsesRetentionGatewayCache struct {
	openAIChatCompletionsGatewayCacheStub
	sessionHash string
	cyberKey    string
}

func (c *openAIResponsesRetentionGatewayCache) GetSessionAccountID(_ context.Context, _ int64, sessionHash string) (int64, error) {
	if c.sessionHash == "" {
		c.sessionHash = sessionHash
	}
	return 0, errors.New("not found")
}

func (c *openAIResponsesRetentionGatewayCache) SetSessionAccountID(_ context.Context, _ int64, sessionHash string, _ int64, _ time.Duration) error {
	return nil
}

func (c *openAIResponsesRetentionGatewayCache) SetCyberSessionBlocked(context.Context, string, time.Duration) error {
	return nil
}

func (c *openAIResponsesRetentionGatewayCache) IsCyberSessionBlocked(_ context.Context, key string) (bool, error) {
	c.cyberKey = key
	return false, nil
}

type openAIResponsesRetentionSettingRepo struct {
	service.SettingRepository
}

func (*openAIResponsesRetentionSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	switch key {
	case service.SettingKeyCyberSessionBlockEnabled:
		return "true", nil
	case service.SettingKeyCyberSessionBlockTTLSeconds:
		return "3600", nil
	default:
		return "", service.ErrSettingNotFound
	}
}

func (*openAIResponsesRetentionSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{
		service.SettingKeyCyberSessionBlockEnabled:    "true",
		service.SettingKeyCyberSessionBlockTTLSeconds: "3600",
	}, nil
}

func TestOpenAIGatewayHandler_ResponsesPassesPreviewSnapshotAndStableHash(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldPreviewLimit := openAIResponsesRequestBodyPreviewLimitBytes
	openAIResponsesRequestBodyPreviewLimitBytes = 32
	t.Cleanup(func() { openAIResponsesRequestBodyPreviewLimitBytes = oldPreviewLimit })

	cfg := &config.Config{
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
	billingRepo := &openAIResponsesRequestBodyRetentionBillingRepoStub{}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_test_123"}},
			Body: io.NopCloser(strings.NewReader(
				`{"id":"resp_1","object":"response","status":"completed","model":"gpt-5","output":[],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2,"input_tokens_details":{"cached_tokens":0}}}`,
			)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(&openAIResponsesRequestBodyRetentionBillingCacheStub{}, nil, nil, nil, nil, nil, cfg, nil)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		billingRepo,
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

	reqBody := `{"model":"gpt-5","stream":false,"input":"` + strings.Repeat("x", 80) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, usageRepo.lastLog)
	require.NotNil(t, usageRepo.lastLog.DetailSnapshot)
	requestPreview := usageRepo.lastLog.DetailSnapshot.RequestBody
	require.NotEqual(t, reqBody, requestPreview)
	require.True(t, gjson.Valid(requestPreview), requestPreview)
	require.Equal(t, requestBodyPreviewSnapshotKind, gjson.Get(requestPreview, "kind").String())
	require.Equal(t, "[inline binary payload omitted]", gjson.Get(requestPreview, "preview").String())
	require.True(t, gjson.Get(requestPreview, "truncated").Bool())
	require.Equal(t, int64(len(reqBody)), gjson.Get(requestPreview, "size").Int())

	require.NotNil(t, billingRepo.lastCmd)
	require.Equal(t, service.HashUsageRequestPayload([]byte(reqBody)), billingRepo.lastCmd.RequestPayloadHash)
}

func TestOpenAIRequestBodyPreviewSnapshotSpoolFailureStillReturnsWrapper(t *testing.T) {
	oldPreviewLimit := openAIResponsesRequestBodyPreviewLimitBytes
	openAIResponsesRequestBodyPreviewLimitBytes = 32
	t.Cleanup(func() { openAIResponsesRequestBodyPreviewLimitBytes = oldPreviewLimit })

	missingTempDir := filepath.Join(t.TempDir(), "missing")
	t.Setenv("TMPDIR", missingTempDir)
	t.Setenv("TMP", missingTempDir)
	t.Setenv("TEMP", missingTempDir)
	body := []byte(strings.Repeat("x", int(service.DefaultRequestBodySpoolThresholdBytes)+1))

	snapshot := openAIRequestBodyPreviewSnapshot(body)

	require.True(t, gjson.Valid(snapshot), snapshot)
	require.Equal(t, requestBodyPreviewSnapshotKind, gjson.Get(snapshot, "kind").String())
	require.Equal(t, "[inline binary payload omitted]", gjson.Get(snapshot, "preview").String())
	require.True(t, gjson.Get(snapshot, "truncated").Bool())
	require.Equal(t, int64(len(body)), gjson.Get(snapshot, "size").Int())
}

type openAIResponsesRequestBodyRetentionBillingRepoStub struct {
	service.UsageBillingRepository

	lastCmd *service.UsageBillingCommand
}

func (s *openAIResponsesRequestBodyRetentionBillingRepoStub) Apply(ctx context.Context, cmd *service.UsageBillingCommand) (*service.UsageBillingApplyResult, error) {
	s.lastCmd = cmd
	return &service.UsageBillingApplyResult{Applied: true}, nil
}

type openAIResponsesRequestBodyRetentionBillingCacheStub struct {
	service.BillingCache
}

func (openAIResponsesRequestBodyRetentionBillingCacheStub) GetUserBalance(context.Context, int64) (float64, error) {
	return 100, nil
}
