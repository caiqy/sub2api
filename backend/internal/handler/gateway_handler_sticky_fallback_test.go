package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stickyFallbackGroupRepo struct {
	service.GroupRepository
	groups map[int64]*service.Group
}

func (r *stickyFallbackGroupRepo) GetByID(_ context.Context, id int64) (*service.Group, error) {
	return r.groups[id], nil
}

func (r *stickyFallbackGroupRepo) GetByIDLite(ctx context.Context, id int64) (*service.Group, error) {
	return r.GetByID(ctx, id)
}

type stickyFallbackAccountRepo struct {
	service.AccountRepository
	groupIDs []int64
}

type stickyCompositeRouteRepo struct {
	routes []service.CompositeModelRoute
}

func (r stickyCompositeRouteRepo) ListByGroup(_ context.Context, groupID int64, includeDisabled bool) ([]service.CompositeModelRoute, error) {
	routes := make([]service.CompositeModelRoute, 0, len(r.routes))
	for _, route := range r.routes {
		if route.GroupID == groupID && (includeDisabled || route.Enabled) {
			routes = append(routes, route)
		}
	}
	return routes, nil
}

func (stickyCompositeRouteRepo) Create(context.Context, *service.CompositeModelRoute) error {
	return nil
}
func (stickyCompositeRouteRepo) Update(context.Context, *service.CompositeModelRoute) error {
	return nil
}
func (stickyCompositeRouteRepo) Delete(context.Context, int64) error        { return nil }
func (stickyCompositeRouteRepo) DeleteByGroup(context.Context, int64) error { return nil }

func (*stickyFallbackAccountRepo) GetByID(context.Context, int64) (*service.Account, error) {
	return nil, errors.New("account not found")
}

func (r *stickyFallbackAccountRepo) ListSchedulableByGroupIDAndPlatforms(_ context.Context, groupID int64, _ []string) ([]service.Account, error) {
	r.groupIDs = append(r.groupIDs, groupID)
	return nil, nil
}

func (*stickyFallbackAccountRepo) ListSchedulableByPlatform(context.Context, string) ([]service.Account, error) {
	return nil, nil
}

func (*stickyFallbackAccountRepo) ListModelAvailabilityCandidates(context.Context, *int64, []string, bool) ([]service.Account, error) {
	return nil, nil
}

func newStickyFallbackMessagesHandler(t *testing.T, cfg *config.Config, groups map[int64]*service.Group, cache service.GatewayCache, accountRepo *stickyFallbackAccountRepo) (*GatewayHandler, *service.APIKey) {
	t.Helper()
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCacheService.Stop)
	groupRepo := &stickyFallbackGroupRepo{groups: groups}
	gatewayService := service.NewGatewayService(accountRepo, groupRepo, nil, nil, nil, nil, nil, cache, cfg, nil, concurrencyService, service.NewBillingService(cfg, nil), nil, billingCacheService, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	originalGroup := groups[1]
	apiKey := &service.APIKey{ID: 101, UserID: 202, Status: service.StatusActive, GroupID: &originalGroup.ID, User: &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1}, Group: originalGroup}
	return NewGatewayHandler(gatewayService, nil, nil, nil, nil, concurrencyService, billingCacheService, nil, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil), apiKey
}

func TestGatewayHandlerMessages_ClaudeCodeFallbackUsesResolvedGeminiStickyBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalID, fallbackID := int64(1), int64(2)
	groups := map[int64]*service.Group{
		originalID: {ID: originalID, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true, ClaudeCodeOnly: true, FallbackGroupID: &fallbackID},
		fallbackID: {ID: fallbackID, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true},
	}

	tests := []struct {
		name              string
		geminiEnabled     bool
		anthropic         bool
		wantCacheActivity bool
	}{
		{name: "gemini disabled bypasses fallback cache", geminiEnabled: false, anthropic: true},
		{name: "gemini enabled reads fallback cache key", geminiEnabled: true, anthropic: false, wantCacheActivity: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{RunMode: config.RunModeSimple, Default: config.DefaultConfig{RateMultiplier: 1}, Gateway: config.GatewayConfig{Scheduling: config.GatewaySchedulingConfig{LoadBatchEnabled: false}}, Concurrency: config.ConcurrencyConfig{PingInterval: 0}}
			cfg.Gateway.Sticky.Gemini.Enabled = tt.geminiEnabled
			cfg.Gateway.Sticky.Anthropic.Enabled = tt.anthropic
			cache := &geminiStickyGatewayCacheStub{}
			if tt.wantCacheActivity {
				cache.defaultAccountID = 999
			}
			accountRepo := &stickyFallbackAccountRepo{}
			h, apiKey := newStickyFallbackMessagesHandler(t, cfg, groups, cache, accountRepo)
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(middleware.ContextKeyAPIKey), apiKey)
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
				c.Next()
			})
			router.POST("/v1/messages", h.Messages)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"gemini-2.5-flash","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusServiceUnavailable, rec.Code)
			if !tt.wantCacheActivity {
				require.Empty(t, cache.getGroupIDs)
			} else {
				require.NotEmpty(t, cache.getGroupIDs)
			}
			require.Empty(t, cache.setGroupIDs)
			require.Contains(t, accountRepo.groupIDs, fallbackID)
			for _, groupID := range cache.getGroupIDs {
				require.Equal(t, fallbackID, groupID)
			}
			for key := range cache.getCalls {
				require.True(t, strings.HasPrefix(key, "gemini:"))
			}
		})
	}
}

func TestGatewayHandlerMessages_ClaudeCodeFallbackSuccessBindsResolvedGeminiStickySession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalID, fallbackID := int64(1), int64(2)
	originalGroup := &service.Group{ID: originalID, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true, ClaudeCodeOnly: true, FallbackGroupID: &fallbackID}
	fallbackGroup := &service.Group{ID: fallbackID, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 303, Name: "fallback-gemini", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, AccountGroups: []service.AccountGroup{{GroupID: fallbackID}}, Credentials: map[string]any{"api_key": "test-key"}}

	for _, tt := range []struct {
		name          string
		geminiEnabled bool
		wantCacheIO   bool
	}{
		{name: "Gemini enabled binds fallback session", geminiEnabled: true, wantCacheIO: true},
		{name: "Gemini disabled bypasses cache despite Anthropic enabled"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cache := &geminiStickyGatewayCacheStub{}
			upstream := &openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]}}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`)),
			}}
			env := newTerminalGatewayMessagesEnvWithGatewayCacheAndGroups(t, fallbackGroup, map[int64]*service.Group{originalID: originalGroup, fallbackID: fallbackGroup}, upstream, openAIChatCompletionsConcurrencyCacheStub{}, cache, account)
			env.apiKey.GroupID = &originalID
			env.apiKey.Group = originalGroup
			env.handler.cfg.Gateway.Sticky.Gemini.Enabled = tt.geminiEnabled
			env.handler.cfg.Gateway.Sticky.Anthropic.Enabled = true

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"gemini-2.5-flash","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`))
			req.Header.Set("Content-Type", "application/json")
			env.router().ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			if !tt.wantCacheIO {
				require.Empty(t, cache.getCalls)
				require.Empty(t, cache.setCalls)
				require.Empty(t, cache.getGroupIDs)
				require.Empty(t, cache.setGroupIDs)
				return
			}

			require.Len(t, cache.setCalls, 1)
			require.Equal(t, []int64{fallbackID}, cache.setGroupIDs)
			for sessionKey, calls := range cache.setCalls {
				require.True(t, strings.HasPrefix(sessionKey, "gemini:"))
				require.Equal(t, 1, calls)
				require.Equal(t, account.ID, cache.sessionBindings[sessionKey])
			}
		})
	}
}

func TestGatewayHandlerMessages_ClaudeCodeFallbackMixedAntigravitySmartRetryClearsResolvedGeminiStickySession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalID, fallbackID := int64(1), int64(2)
	originalGroup := &service.Group{ID: originalID, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true, ClaudeCodeOnly: true, FallbackGroupID: &fallbackID}
	fallbackGroup := &service.Group{ID: fallbackID, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID: 303, Name: "fallback-antigravity", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
		AccountGroups: []service.AccountGroup{{GroupID: fallbackID}},
		Credentials:   map[string]any{"access_token": "token", "project_id": "project"},
		Extra:         map[string]any{"mixed_scheduling": true},
	}
	cache := &geminiStickyGatewayCacheStub{}
	upstream := &openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{
			"error": {
				"status": "RESOURCE_EXHAUSTED",
				"details": [
					{"@type": "type.googleapis.com/google.rpc.ErrorInfo", "metadata": {"model": "gemini-3-flash"}, "reason": "RATE_LIMIT_EXCEEDED"},
					{"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "15s"}
				]
			}
		}`)),
	}}
	env := newTerminalGatewayMessagesEnvWithGatewayCacheAndGroups(t, fallbackGroup, map[int64]*service.Group{originalID: originalGroup, fallbackID: fallbackGroup}, upstream, openAIChatCompletionsConcurrencyCacheStub{}, cache, account)
	env.apiKey.GroupID = &originalID
	env.apiKey.Group = originalGroup
	env.handler.cfg.Gateway.Sticky.Gemini.Enabled = true
	env.handler.cfg.Gateway.Sticky.Anthropic.Enabled = true
	env.handler.maxAccountSwitchesGemini = 0

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"gemini-3-flash-preview","max_tokens":16,"metadata":{"user_id":"{\"device_id\":\"sticky-device\",\"session_id\":\"sticky-fallback-runtime\"}"},"messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	env.router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Len(t, cache.deleteCalls, 1)
	require.Equal(t, fallbackID, cache.deleteCalls[0].groupID)
	require.Equal(t, "gemini:sticky-fallback-runtime", cache.deleteCalls[0].sessionKey)
}

func TestGatewayHandlerResolveStickyRoute_PreservesClaudeCodeRestrictionErrors(t *testing.T) {
	cfg := &config.Config{}
	groupTwoID, groupThreeID := int64(2), int64(3)
	groups := map[int64]*service.Group{
		1: {ID: 1, Platform: service.PlatformAnthropic, ClaudeCodeOnly: true},
		2: {ID: groupTwoID, Platform: service.PlatformGemini, ClaudeCodeOnly: true},
		3: {ID: groupThreeID, Platform: service.PlatformAnthropic, ClaudeCodeOnly: true},
	}
	groups[2].FallbackGroupID = &groupThreeID
	groups[3].FallbackGroupID = &groupTwoID
	accountRepo := &stickyFallbackAccountRepo{}
	h, apiKey := newStickyFallbackMessagesHandler(t, cfg, groups, &geminiStickyGatewayCacheStub{}, accountRepo)

	_, _, _, _, err := h.resolveStickyRoute(context.Background(), apiKey, "")
	require.ErrorIs(t, err, service.ErrClaudeCodeOnly)

	apiKey.GroupID = &groups[2].ID
	apiKey.Group = groups[2]
	_, _, _, _, err = h.resolveStickyRoute(context.Background(), apiKey, "")
	require.ErrorContains(t, err, "fallback group cycle detected")

	_, group, groupID, platform, err := h.resolveStickyRoute(context.WithValue(context.Background(), ctxkey.ForcePlatform, service.PlatformAntigravity), apiKey, "")
	require.NoError(t, err)
	require.Nil(t, group)
	require.Equal(t, groups[2].ID, *groupID)
	require.Equal(t, service.PlatformAntigravity, platform)
}

func TestGatewayHandlerResolveStickyRouteRecomputesCompositeFallbackDecision(t *testing.T) {
	originalID, fallbackID := int64(1), int64(2)
	groups := map[int64]*service.Group{
		originalID: {ID: originalID, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true, ClaudeCodeOnly: true, FallbackGroupID: &fallbackID},
		fallbackID: {ID: fallbackID, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true},
	}
	resolver := service.NewCompositeRouteResolver(stickyCompositeRouteRepo{routes: []service.CompositeModelRoute{{
		GroupID:        fallbackID,
		PublicModel:    "public-model",
		MatchType:      service.CompositeRouteMatchExact,
		TargetPlatform: service.PlatformAnthropic,
		UpstreamModel:  "claude-sonnet-4-6",
		Endpoint:       service.CompositeRouteEndpointAny,
		Enabled:        true,
	}}})
	cfg := &config.Config{}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCacheService.Stop)
	gatewayService := service.NewGatewayService(
		&stickyFallbackAccountRepo{}, &stickyFallbackGroupRepo{groups: groups}, nil, nil, nil, nil, nil,
		&geminiStickyGatewayCacheStub{}, cfg, nil, concurrencyService, service.NewBillingService(cfg, nil), nil,
		billingCacheService, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, resolver, nil, nil,
	)
	h := NewGatewayHandler(gatewayService, nil, nil, nil, nil, concurrencyService, billingCacheService, nil, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)
	apiKey := &service.APIKey{GroupID: &originalID, Group: groups[originalID]}
	ctx := service.WithCompositeRouteDecision(context.Background(), service.CompositeRouteDecision{
		Matched:        true,
		GroupID:        originalID,
		PublicModel:    "public-model",
		TargetPlatform: service.PlatformOpenAI,
		UpstreamModel:  "gpt-5",
	})

	resolvedCtx, group, groupID, platform, err := h.resolveStickyRoute(ctx, apiKey, "public-model")

	require.NoError(t, err)
	require.Equal(t, fallbackID, group.ID)
	require.Equal(t, fallbackID, *groupID)
	require.Equal(t, service.PlatformAnthropic, platform)
	decision, ok := service.CompositeRouteDecisionFromContext(resolvedCtx)
	require.True(t, ok)
	require.Equal(t, fallbackID, decision.GroupID)
	require.Equal(t, "claude-sonnet-4-6", decision.UpstreamModel)
}
