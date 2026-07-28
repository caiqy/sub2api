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
	apiKeyService := service.NewAPIKeyService(nil, nil, groupRepo, nil, nil, nil, cfg)
	effectiveResolver := service.NewEffectiveGatewayRouteResolver(apiKeyService, service.NewCompositeRouteResolver(nil), cfg)
	return NewGatewayHandler(gatewayService, nil, nil, nil, nil, concurrencyService, billingCacheService, nil, apiKeyService, nil, nil, nil, nil, cfg, nil, effectiveResolver), apiKey
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

func TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalID, finalID := int64(601), int64(602)
	originalGroup := &service.Group{ID: originalID, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
	finalGroup := &service.Group{ID: finalID, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID: 603, Name: "effective-anthropic", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
		Credentials: map[string]any{"api_key": "test-key"},
	}
	upstream := &openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"type":"message","id":"msg_1","role":"assistant","content":[{"type":"text","text":"ok"}],"model":"claude-sonnet-4-6","usage":{"input_tokens":1,"output_tokens":1}}`)),
	}}
	env := newTerminalGatewayMessagesEnvWithGatewayCacheAndGroups(
		t,
		originalGroup,
		map[int64]*service.Group{originalID: originalGroup, finalID: finalGroup},
		upstream,
		openAIChatCompletionsConcurrencyCacheStub{},
		openAIChatCompletionsGatewayCacheStub{},
		account,
	)
	finalKey := *env.apiKey
	finalKey.GroupID = &finalID
	finalKey.Group = finalGroup
	route := service.EffectiveGatewayRoute{
		APIKey: env.apiKey, Group: finalGroup, GroupID: &finalID,
		Endpoint: service.CompositeRouteEndpointMessages, ClientModel: "claude-sonnet-4-6",
		RoutingModel: "claude-sonnet-4-6", UpstreamModel: "claude-sonnet-4-6", Platform: service.PlatformAnthropic,
		Channel: service.ChannelMappingResult{ChannelID: 999},
	}
	route.APIKey = &finalKey
	var finalRoute service.EffectiveGatewayRoute

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), env.apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: env.apiKey.UserID, Concurrency: env.apiKey.User.Concurrency})
		c.Request = c.Request.WithContext(service.WithEffectiveGatewayRoute(c.Request.Context(), route))
		c.Next()
		finalRoute, _ = service.EffectiveGatewayRouteFromContext(c.Request.Context())
	})
	router.POST("/v1/messages", env.handler.Messages)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-sonnet-4-6","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, env.accountRepo.groupIDs, finalID)
	require.NotContains(t, env.accountRepo.groupIDs, originalID)
	require.Equal(t, "claude-sonnet-4-6", finalRoute.Channel.MappedModel)
	require.Zero(t, finalRoute.Channel.ChannelID)
}
