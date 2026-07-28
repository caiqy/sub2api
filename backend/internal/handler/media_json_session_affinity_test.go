package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type mediaJSONSessionCache struct {
	service.GatewayCache
	mu       sync.Mutex
	sessions []string
}

func (c *mediaJSONSessionCache) GetSessionAccountID(_ context.Context, _ int64, session string) (int64, error) {
	c.mu.Lock()
	c.sessions = append(c.sessions, session)
	c.mu.Unlock()
	return 0, nil
}

func (c *mediaJSONSessionCache) SetSessionAccountID(context.Context, int64, string, int64, time.Duration) error {
	return nil
}

func (c *mediaJSONSessionCache) RefreshSessionTTL(context.Context, int64, string, time.Duration) error {
	return nil
}

func (c *mediaJSONSessionCache) DeleteSessionAccountID(context.Context, int64, string) error {
	return nil
}

func (c *mediaJSONSessionCache) all() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.sessions...)
}

type mediaJSONSessionUpstream struct {
	service.HTTPUpstream
	mu          sync.Mutex
	calls       int
	retryStatus int
}

func (u *mediaJSONSessionUpstream) Do(*http.Request, string, int64, int) (*http.Response, error) {
	u.mu.Lock()
	u.calls++
	call := u.calls
	u.mu.Unlock()
	status := http.StatusOK
	if call == 1 {
		status = u.retryStatus
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(`{"created":1,"data":[{"b64_json":"aGVsbG8="}]}`)),
	}, nil
}

func runMediaJSONSessionAffinity(t *testing.T, route, kind, body string, headers map[string]string) []string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	platform := service.PlatformOpenAI
	cache := &mediaJSONSessionCache{}
	upstream := &mediaJSONSessionUpstream{retryStatus: http.StatusTooManyRequests}
	group := &service.Group{ID: 771, Platform: platform, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	accounts := []*service.Account{
		{ID: 772, Name: "first", Platform: platform, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Priority: 1, Credentials: map[string]any{"api_key": "key", "pool_mode": true}},
	}
	if kind != "openai-images" {
		platform = service.PlatformGrok
		group.Platform = platform
		parentID := int64(774)
		accounts[0].Platform = platform
		accounts[0].Type = service.AccountTypeOAuth
		accounts[0].ParentAccountID = &parentID
		accounts[0].Credentials["base_url"] = "https://api.x.ai/v1"
		accounts = append(accounts, &service.Account{ID: 773, Name: "second", Platform: platform, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Priority: 2, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}})
		accounts = append(accounts, &service.Account{ID: parentID, Name: "parent", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Credentials: map[string]any{"access_token": "token"}})
		upstream.retryStatus = http.StatusInternalServerError
	}
	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			OpenAIWS: config.GatewayOpenAIWSConfig{
				Enabled:                  true,
				APIKeyEnabled:            true,
				ResponsesWebsocketsV2:    true,
				HTTPBridgeEnabled:        true,
				HTTPBridgeThresholdBytes: 1,
			},
			Sticky: config.GatewayStickyConfig{OpenAI: config.GatewayStickyPlatformConfig{Enabled: true}},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}
	billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	concurrency := service.NewConcurrencyService(nil)
	var accountRepo service.AccountRepository = &openAIRetryAccountRepoStub{accounts: accounts}
	if kind != "openai-images" {
		accountRepo = &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: accounts}}
	}
	gateway := service.NewOpenAIGatewayService(
		accountRepo, nil, nil, nil, nil, nil, cache, cfg,
		nil, concurrency, nil, nil, billing, upstream, nil, nil, service.NewGrokTokenProvider(accountRepo, nil), nil, nil,
	)
	h := NewOpenAIGatewayHandler(gateway, concurrency, billing, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)
	h.grokMediaEligibilityProber = &grokMediaEligibilityProberStub{eligible: true, reason: "eligible"}
	h.maxAccountSwitches = 1

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{ID: 775, UserID: 776, GroupID: &group.ID, Group: group, User: &service.User{ID: 776, Status: service.StatusActive}})
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 776})
	})
	switch kind {
	case "openai-images":
		router.POST(route, h.Images)
	case "grok-images":
		router.POST(route, h.GrokImages)
	case "grok-videos":
		router.POST(route, h.GrokVideoGeneration)
	default:
		t.Fatalf("unknown media handler %q", kind)
	}
	req := httptest.NewRequest(http.MethodPost, route, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	return cache.all()
}

func TestMediaJSONHandlers_PreserveContentDerivedSchedulerAffinity(t *testing.T) {
	for _, tt := range []struct {
		name  string
		route string
		kind  string
		model string
	}{
		{"openai images", "/v1/images/generations", "openai-images", "gpt-image-2"},
		{"grok images", "/v1/images/generations", "grok-images", "grok-imagine"},
		{"grok videos", "/v1/videos/generations", "grok-videos", "grok-imagine-video-1.5"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			invoke := func(prompt string, headers map[string]string, promptCacheKey string) []string {
				body := `{"model":"` + tt.model + `","prompt":"` + prompt + `"}`
				if promptCacheKey != "" {
					body = `{"model":"` + tt.model + `","prompt":"` + prompt + `","prompt_cache_key":"` + promptCacheKey + `"}`
				}
				return runMediaJSONSessionAffinity(t, tt.route, tt.kind, body, headers)
			}

			first := invoke("draw a lighthouse", nil, "")
			require.GreaterOrEqual(t, len(first), 2)
			require.NotEmpty(t, first[0])
			for _, session := range first[1:] {
				require.Equal(t, first[0], session, "retry must retain the scheduler session")
			}

			second := invoke("draw a library", nil, "")
			require.GreaterOrEqual(t, len(second), 2)
			require.NotEmpty(t, second[0])
			require.NotEqual(t, first[0], second[0], "different prompts need distinct scheduler sessions")

			explicitHeader := invoke("draw a forest", map[string]string{"session_id": "media-explicit"}, "")
			explicitHeaderOtherPrompt := invoke("draw a desert", map[string]string{"session_id": "media-explicit"}, "")
			require.Equal(t, explicitHeader[0], explicitHeaderOtherPrompt[0], "session_id must override prompt content")

			explicitBody := invoke("draw a river", nil, "media-cache-key")
			explicitBodyOtherPrompt := invoke("draw a mountain", nil, "media-cache-key")
			require.Equal(t, explicitBody[0], explicitBodyOtherPrompt[0], "prompt_cache_key must override prompt content")
		})
	}
}
