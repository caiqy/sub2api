package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newGeminiStickyRequestContext(t *testing.T, body string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1beta/models/gemini-2.5-pro:generateContent", nil)
	c.Request.Header.Set("User-Agent", "gemini-test-client")
	c.Request.Header.Set("X-Real-IP", "203.0.113.9")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 77, Concurrency: 1})
	_ = body
	return c
}

func newGeminiStickyRequestAPIKey() *service.APIKey {
	groupID := int64(1)
	return &service.APIKey{
		ID:      88,
		GroupID: &groupID,
		Group: &service.Group{
			ID:       groupID,
			Platform: service.PlatformGemini,
		},
	}
}

func TestGeminiStickyRequestPath_DisabledBypassesStickyPreparation(t *testing.T) {
	body := `{"contents":[{"role":"user","parts":[{"text":"hello sticky world"}]}]}`
	cache := &geminiStickyGatewayCacheStub{sessionBindings: map[string]int64{"gemini:any": 123}}
	digestStore := service.NewDigestSessionStore()
	digestStore.Save(1, "preloaded-prefix", "preloaded-chain", "preloaded-uuid", 456, "")
	h := newGeminiStickyToggleHandler(t, false, cache, digestStore)
	c := newGeminiStickyRequestContext(t, body)
	apiKey := newGeminiStickyRequestAPIKey()

	state := h.prepareGeminiStickySelectionFromRequest(c, apiKey, middleware.AuthSubject{UserID: 77, Concurrency: 1}, "gemini-2.5-pro", []byte(body))

	require.Empty(t, state.SelectionSessionKey)
	require.Zero(t, state.SessionBoundAccountID)
	require.False(t, state.UseDigestFallback)
	require.Empty(t, state.GeminiDigestChain)
	require.Empty(t, state.GeminiPrefixHash)
	require.Empty(t, cache.getCalls)
	require.Empty(t, cache.setCalls)
}

func TestGeminiStickyRequestPath_EnabledDigestFallbackWithoutInitialSessionKey(t *testing.T) {
	body := `{"contents":[{"role":"user","parts":[{"text":"hello sticky world"}]},{"role":"model","parts":[{"text":"response"}]}]}`
	cache := &geminiStickyGatewayCacheStub{}
	digestStore := service.NewDigestSessionStore()
	apiKey := newGeminiStickyRequestAPIKey()
	c := newGeminiStickyRequestContext(t, body)
	prefixHash := service.GenerateGeminiPrefixHash(77, apiKey.ID, "203.0.113.9", "gemini-test-client", service.PlatformGemini, "gemini-2.5-pro")
	var geminiReq antigravity.GeminiRequest
	require.NoError(t, json.Unmarshal([]byte(body), &geminiReq))
	digestChain := service.BuildGeminiDigestChain(&geminiReq)
	digestStore.Save(1, prefixHash, digestChain, "uuid-request-path", 789, "")
	h := newGeminiStickyToggleHandler(t, true, cache, digestStore)

	state := h.prepareGeminiStickySelectionFromRequest(c, apiKey, middleware.AuthSubject{UserID: 77, Concurrency: 1}, "gemini-2.5-pro", []byte(body))

	require.True(t, state.UseDigestFallback)
	require.Equal(t, int64(789), state.SessionBoundAccountID)
	require.Equal(t, digestChain, state.GeminiDigestChain)
	require.Equal(t, prefixHash, state.GeminiPrefixHash)
	require.NotEmpty(t, state.SelectionSessionKey)
	require.Equal(t, 1, cache.setCalls[state.SelectionSessionKey])
	require.Equal(t, int64(789), cache.sessionBindings[state.SelectionSessionKey])
}

func TestGeminiStickyMainFlow_DisabledBypassesStickyInteractions(t *testing.T) {
	cache := &geminiStickyGatewayCacheStub{sessionBindings: map[string]int64{"gemini:existing": 111}}
	digestStore := service.NewDigestSessionStore()
	digestStore.Save(1, "prefix-disabled", "chain-a-b", "uuid-disabled", 222, "")
	h := newGeminiStickyToggleHandler(t, false, cache, digestStore)

	state := h.prepareGeminiStickySelection(context.Background(), geminiStickySelectionInput{
		GroupID:           nil,
		SessionKey:        "gemini:existing",
		DigestGroupID:     1,
		GeminiPrefixHash:  "prefix-disabled",
		GeminiDigestChain: "chain-a-b-c",
	})

	require.Empty(t, state.SelectionSessionKey)
	require.Zero(t, state.SessionBoundAccountID)
	require.False(t, state.UseDigestFallback)
	require.Zero(t, cache.getCalls["gemini:existing"])
	require.Zero(t, cache.setCalls["gemini:existing"])
	uuid, accountID, matchedChain, found := digestStore.Find(1, "prefix-disabled", "chain-a-b-c")
	require.True(t, found)
	require.Equal(t, "uuid-disabled", uuid)
	require.Equal(t, int64(222), accountID)
	require.Equal(t, "chain-a-b", matchedChain)
}

func TestGeminiStickyMainFlow_EnabledDigestFallbackWorksWithoutInitialSessionKey(t *testing.T) {
	cache := &geminiStickyGatewayCacheStub{}
	digestStore := service.NewDigestSessionStore()
	digestStore.Save(1, "prefix-enabled", "chain-a-b", "uuid-enabled", 333, "")
	h := newGeminiStickyToggleHandler(t, true, cache, digestStore)

	state := h.prepareGeminiStickySelection(context.Background(), geminiStickySelectionInput{
		GroupID:           nil,
		SessionKey:        "",
		DigestGroupID:     1,
		GeminiPrefixHash:  "prefix-enabled",
		GeminiDigestChain: "chain-a-b-c",
	})

	require.True(t, state.UseDigestFallback)
	require.Equal(t, int64(333), state.SessionBoundAccountID)
	require.NotEmpty(t, state.SelectionSessionKey)
	require.Equal(t, service.GenerateGeminiDigestSessionKey("prefix-enabled", "uuid-enabled"), state.SelectionSessionKey)
	require.Equal(t, 1, cache.setCalls[state.SelectionSessionKey])
	require.Equal(t, int64(333), cache.sessionBindings[state.SelectionSessionKey])
}

func TestGeminiStickyMainFlow_EnabledNonStickyPathPreserved(t *testing.T) {
	h := newGeminiStickyToggleHandler(t, true, nil, nil)

	state := h.prepareGeminiStickySelection(context.Background(), geminiStickySelectionInput{
		GroupID:    nil,
		SessionKey: "gemini:plain-session",
	})

	require.Equal(t, "gemini:plain-session", state.SelectionSessionKey)
	require.False(t, state.UseDigestFallback)
	state.bindSelectedAccount = false
	require.NoError(t, h.finalizeGeminiStickySelection(context.Background(), state, 444))
}

type geminiStickyGatewayCacheStub struct {
	sessionBindings map[string]int64
	getCalls        map[string]int
	setCalls        map[string]int
	refreshCalls    map[string]int
}

func (c *geminiStickyGatewayCacheStub) GetSessionAccountID(_ context.Context, _ int64, sessionHash string) (int64, error) {
	if c.getCalls == nil {
		c.getCalls = make(map[string]int)
	}
	c.getCalls[sessionHash]++
	if accountID, ok := c.sessionBindings[sessionHash]; ok {
		return accountID, nil
	}
	return 0, errors.New("not found")
}

func (c *geminiStickyGatewayCacheStub) SetSessionAccountID(_ context.Context, _ int64, sessionHash string, accountID int64, _ time.Duration) error {
	if c.setCalls == nil {
		c.setCalls = make(map[string]int)
	}
	c.setCalls[sessionHash]++
	if c.sessionBindings == nil {
		c.sessionBindings = make(map[string]int64)
	}
	c.sessionBindings[sessionHash] = accountID
	return nil
}

func (c *geminiStickyGatewayCacheStub) RefreshSessionTTL(_ context.Context, _ int64, sessionHash string, _ time.Duration) error {
	if c.refreshCalls == nil {
		c.refreshCalls = make(map[string]int)
	}
	c.refreshCalls[sessionHash]++
	return nil
}

func (c *geminiStickyGatewayCacheStub) DeleteSessionAccountID(_ context.Context, _ int64, sessionHash string) error {
	delete(c.sessionBindings, sessionHash)
	return nil
}

func newGeminiStickyToggleHandler(t *testing.T, enabled bool, cache service.GatewayCache, digestStore *service.DigestSessionStore) *GatewayHandler {
	t.Helper()
	cfg := &config.Config{}
	cfg.Gateway.Sticky.Gemini.Enabled = enabled
	gatewayService := service.NewGatewayService(nil, nil, nil, nil, nil, nil, nil, cache, cfg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, digestStore, nil, nil, nil, nil, nil)
	return &GatewayHandler{gatewayService: gatewayService, cfg: cfg}
}

func TestGeminiStickyEnabled_GetCachedSessionAccountID(t *testing.T) {
	t.Run("enabled delegates sticky lookup", func(t *testing.T) {
		cache := &geminiStickyGatewayCacheStub{sessionBindings: map[string]int64{"gemini:session-enabled": 101}}
		h := newGeminiStickyToggleHandler(t, true, cache, nil)

		accountID := h.getGeminiCachedSessionAccountID(context.Background(), nil, "gemini:session-enabled")

		require.Equal(t, int64(101), accountID)
		require.Equal(t, 1, cache.getCalls["gemini:session-enabled"])
	})

	t.Run("disabled bypasses sticky lookup", func(t *testing.T) {
		cache := &geminiStickyGatewayCacheStub{sessionBindings: map[string]int64{"gemini:session-disabled": 202}}
		h := newGeminiStickyToggleHandler(t, false, cache, nil)

		accountID := h.getGeminiCachedSessionAccountID(context.Background(), nil, "gemini:session-disabled")

		require.Zero(t, accountID)
		require.Zero(t, cache.getCalls["gemini:session-disabled"])
	})
}

func TestGeminiStickyEnabled_BindStickySession(t *testing.T) {
	t.Run("enabled writes sticky bind", func(t *testing.T) {
		cache := &geminiStickyGatewayCacheStub{}
		h := newGeminiStickyToggleHandler(t, true, cache, nil)

		err := h.bindGeminiStickySession(context.Background(), nil, "gemini:bind-enabled", 303)

		require.NoError(t, err)
		require.Equal(t, 1, cache.setCalls["gemini:bind-enabled"])
		require.Equal(t, int64(303), cache.sessionBindings["gemini:bind-enabled"])
	})

	t.Run("disabled skips sticky bind", func(t *testing.T) {
		cache := &geminiStickyGatewayCacheStub{}
		h := newGeminiStickyToggleHandler(t, false, cache, nil)

		err := h.bindGeminiStickySession(context.Background(), nil, "gemini:bind-disabled", 404)

		require.NoError(t, err)
		require.Zero(t, cache.setCalls["gemini:bind-disabled"])
		require.Zero(t, cache.sessionBindings["gemini:bind-disabled"])
	})
}

func TestGatewayHandler_GeminiRouteStickyLookupUsesGeminiToggleNotAnthropicToggle(t *testing.T) {
	t.Run("gemini disabled bypasses lookup even when anthropic enabled", func(t *testing.T) {
		cache := &geminiStickyGatewayCacheStub{sessionBindings: map[string]int64{"gemini:gateway-route-disabled": 707}}
		cfg := &config.Config{}
		cfg.Gateway.Sticky.Gemini.Enabled = false
		cfg.Gateway.Sticky.Anthropic.Enabled = true
		gatewayService := service.NewGatewayService(nil, nil, nil, nil, nil, nil, nil, cache, cfg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		h := &GatewayHandler{gatewayService: gatewayService, cfg: cfg}

		accountID := h.getCachedSessionAccountIDForPlatform(context.Background(), service.PlatformGemini, nil, "gemini:gateway-route-disabled")

		require.Zero(t, accountID)
		require.Zero(t, cache.getCalls["gemini:gateway-route-disabled"])
	})

	t.Run("gemini enabled performs lookup even when anthropic disabled", func(t *testing.T) {
		cache := &geminiStickyGatewayCacheStub{sessionBindings: map[string]int64{"gemini:gateway-route-enabled": 808}}
		cfg := &config.Config{}
		cfg.Gateway.Sticky.Gemini.Enabled = true
		cfg.Gateway.Sticky.Anthropic.Enabled = false
		gatewayService := service.NewGatewayService(nil, nil, nil, nil, nil, nil, nil, cache, cfg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		h := &GatewayHandler{gatewayService: gatewayService, cfg: cfg}

		accountID := h.getCachedSessionAccountIDForPlatform(context.Background(), service.PlatformGemini, nil, "gemini:gateway-route-enabled")

		require.Equal(t, int64(808), accountID)
		require.Equal(t, 1, cache.getCalls["gemini:gateway-route-enabled"])
	})
}

func TestGatewayHandler_GeminiRouteStickyBindUsesGeminiToggleNotAnthropicToggle(t *testing.T) {
	t.Run("gemini disabled bypasses bind even when anthropic enabled", func(t *testing.T) {
		cache := &geminiStickyGatewayCacheStub{}
		cfg := &config.Config{}
		cfg.Gateway.Sticky.Gemini.Enabled = false
		cfg.Gateway.Sticky.Anthropic.Enabled = true
		gatewayService := service.NewGatewayService(nil, nil, nil, nil, nil, nil, nil, cache, cfg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		h := &GatewayHandler{gatewayService: gatewayService, cfg: cfg}

		err := h.bindStickySessionForPlatform(context.Background(), service.PlatformGemini, nil, "gemini:gateway-bind-disabled", 909)

		require.NoError(t, err)
		require.Zero(t, cache.setCalls["gemini:gateway-bind-disabled"])
		require.Zero(t, cache.sessionBindings["gemini:gateway-bind-disabled"])
	})

	t.Run("gemini enabled writes bind even when anthropic disabled", func(t *testing.T) {
		cache := &geminiStickyGatewayCacheStub{}
		cfg := &config.Config{}
		cfg.Gateway.Sticky.Gemini.Enabled = true
		cfg.Gateway.Sticky.Anthropic.Enabled = false
		gatewayService := service.NewGatewayService(nil, nil, nil, nil, nil, nil, nil, cache, cfg, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		h := &GatewayHandler{gatewayService: gatewayService, cfg: cfg}

		err := h.bindStickySessionForPlatform(context.Background(), service.PlatformGemini, nil, "gemini:gateway-bind-enabled", 1001)

		require.NoError(t, err)
		require.Equal(t, 1, cache.setCalls["gemini:gateway-bind-enabled"])
		require.Equal(t, int64(1001), cache.sessionBindings["gemini:gateway-bind-enabled"])
	})
}

func TestGeminiStickyEnabled_DigestSession(t *testing.T) {
	t.Run("enabled keeps digest fallback behavior", func(t *testing.T) {
		digestStore := service.NewDigestSessionStore()
		h := newGeminiStickyToggleHandler(t, true, nil, digestStore)

		err := h.saveGeminiDigestSession(context.Background(), 1, "prefix-enabled", "chain-a-b", "uuid-enabled", 505, "")
		require.NoError(t, err)

		uuid, accountID, matchedChain, found := h.findGeminiDigestSession(context.Background(), 1, "prefix-enabled", "chain-a-b-c")
		require.True(t, found)
		require.Equal(t, "uuid-enabled", uuid)
		require.Equal(t, int64(505), accountID)
		require.Equal(t, "chain-a-b", matchedChain)
	})

	t.Run("disabled bypasses digest fallback sticky store", func(t *testing.T) {
		digestStore := service.NewDigestSessionStore()
		h := newGeminiStickyToggleHandler(t, false, nil, digestStore)

		err := h.saveGeminiDigestSession(context.Background(), 1, "prefix-disabled", "chain-x-y", "uuid-disabled", 606, "")
		require.NoError(t, err)

		uuid, accountID, matchedChain, found := h.findGeminiDigestSession(context.Background(), 1, "prefix-disabled", "chain-x-y-z")
		require.False(t, found)
		require.Empty(t, uuid)
		require.Zero(t, accountID)
		require.Empty(t, matchedChain)
	})
}

func TestGeminiStickyEnabled_SessionKeyForSelection(t *testing.T) {
	enabledHandler := newGeminiStickyToggleHandler(t, true, nil, nil)
	disabledHandler := newGeminiStickyToggleHandler(t, false, nil, nil)

	require.Equal(t, "gemini:session-key", enabledHandler.geminiStickySessionKey("gemini:session-key"))
	require.Empty(t, disabledHandler.geminiStickySessionKey("gemini:session-key"))
}
