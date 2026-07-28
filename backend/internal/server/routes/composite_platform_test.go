package routes

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type compositeRouteRepoStub struct {
	routes []service.CompositeModelRoute
}

type effectiveRouteGroupRepoStub struct {
	service.GroupRepository
	groups map[int64]*service.Group
}

func (r effectiveRouteGroupRepoStub) GetByIDLite(_ context.Context, id int64) (*service.Group, error) {
	group := r.groups[id]
	if group == nil {
		return nil, service.ErrGroupNotFound
	}
	return group, nil
}

type effectiveRouteSubscriptionRepoStub struct {
	service.UserSubscriptionRepository
	subscription *service.UserSubscription
}

func (r effectiveRouteSubscriptionRepoStub) GetActiveByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*service.UserSubscription, error) {
	if r.subscription == nil || r.subscription.UserID != userID || r.subscription.GroupID != groupID {
		return nil, service.ErrSubscriptionNotFound
	}
	return r.subscription, nil
}

func effectiveRouteMiddlewareForTest(t *testing.T, resolver *service.EffectiveGatewayRouteResolver) gin.HandlerFunc {
	t.Helper()
	return compositeTargetPlatformMiddleware(resolver)
}

func newEffectiveRouteResolverForTest(cfg *config.Config, groups map[int64]*service.Group, subscription *service.UserSubscription) *service.EffectiveGatewayRouteResolver {
	apiKeys := service.NewAPIKeyService(
		nil,
		nil,
		effectiveRouteGroupRepoStub{groups: groups},
		effectiveRouteSubscriptionRepoStub{subscription: subscription},
		nil,
		nil,
		cfg,
	)
	return service.NewEffectiveGatewayRouteResolver(apiKeys, service.NewCompositeRouteResolver(nil), cfg)
}

func effectiveCompositeRouteResolverForTest(composite *service.CompositeRouteResolver) *service.EffectiveGatewayRouteResolver {
	if composite == nil {
		composite = service.NewCompositeRouteResolver(nil)
	}
	cfg := &config.Config{RunMode: config.RunModeSimple}
	return service.NewEffectiveGatewayRouteResolver(&service.APIKeyService{}, composite, cfg)
}

func TestCompositeTargetPlatformMiddlewareAppliesFallbackBeforeProtocolDispatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalID, finalID := int64(101), int64(102)
	originalGroup := &service.Group{ID: originalID, Platform: service.PlatformAnthropic, Status: service.StatusActive, ClaudeCodeOnly: true, FallbackGroupID: &finalID}
	finalGroup := &service.Group{ID: finalID, Platform: service.PlatformOpenAI, Status: service.StatusActive}
	apiKey := &service.APIKey{ID: 1, UserID: 2, GroupID: &originalID, Group: originalGroup, User: &service.User{ID: 2, Status: service.StatusActive}}
	cfg := &config.Config{RunMode: config.RunModeSimple}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyAPIKey), apiKey)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.Group, originalGroup))
		c.Next()
	})
	router.Use(effectiveRouteMiddlewareForTest(t, newEffectiveRouteResolverForTest(cfg, map[int64]*service.Group{finalID: finalGroup}, nil)))
	router.POST("/v1/messages", func(c *gin.Context) {
		effectiveKey, ok := servermiddleware.GetAPIKeyFromContext(c)
		require.True(t, ok)
		require.Equal(t, finalGroup.ID, *effectiveKey.GroupID)
		require.Same(t, finalGroup, effectiveKey.Group)

		contextGroup, ok := c.Request.Context().Value(ctxkey.Group).(*service.Group)
		require.True(t, ok)
		require.Equal(t, finalGroup.ID, contextGroup.ID)
		require.Equal(t, service.PlatformOpenAI, getGroupPlatform(c))

		route, ok := service.EffectiveGatewayRouteFromContext(c.Request.Context())
		require.True(t, ok)
		require.Equal(t, "gpt-5", route.ClientModel)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"gpt-5","messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestCompositeTargetPlatformMiddlewareLoadsFinalSubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalID, finalID := int64(111), int64(112)
	originalGroup := &service.Group{ID: originalID, Platform: service.PlatformAnthropic, Status: service.StatusActive, ClaudeCodeOnly: true, FallbackGroupID: &finalID}
	finalGroup := &service.Group{ID: finalID, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeSubscription}
	apiKey := &service.APIKey{ID: 3, UserID: 4, GroupID: &originalID, Group: originalGroup, User: &service.User{ID: 4, Status: service.StatusActive}}
	subscription := &service.UserSubscription{ID: 5, UserID: apiKey.UserID, GroupID: finalID, Status: service.SubscriptionStatusActive}
	cfg := &config.Config{}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyAPIKey), apiKey)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.Group, originalGroup))
		c.Next()
	})
	router.Use(effectiveRouteMiddlewareForTest(t, newEffectiveRouteResolverForTest(cfg, map[int64]*service.Group{finalID: finalGroup}, subscription)))
	router.POST("/v1/messages", func(c *gin.Context) {
		effectiveKey, ok := servermiddleware.GetAPIKeyFromContext(c)
		require.True(t, ok)
		require.Equal(t, finalGroup.ID, *effectiveKey.GroupID)
		require.Same(t, finalGroup, effectiveKey.Group)

		got, ok := servermiddleware.GetSubscriptionFromContext(c)
		require.True(t, ok)
		require.Equal(t, finalGroup.ID, got.GroupID)

		contextGroup, ok := c.Request.Context().Value(ctxkey.Group).(*service.Group)
		require.True(t, ok)
		require.Equal(t, finalGroup.ID, contextGroup.ID)
		require.Equal(t, service.PlatformOpenAI, getGroupPlatform(c))
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"gpt-5","messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestCompositeTargetPlatformMiddlewareDoesNotApplyFailedRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalID, finalID := int64(121), int64(122)
	originalGroup := &service.Group{ID: originalID, Platform: service.PlatformAnthropic, Status: service.StatusActive, ClaudeCodeOnly: true, FallbackGroupID: &finalID}
	finalGroup := &service.Group{ID: finalID, Platform: service.PlatformOpenAI, Status: service.StatusDisabled}
	apiKey := &service.APIKey{ID: 6, UserID: 7, GroupID: &originalID, Group: originalGroup, User: &service.User{ID: 7, Status: service.StatusActive}}
	var keyAfterRoute *service.APIKey
	reachedHandler := false

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyAPIKey), apiKey)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.Group, originalGroup))
		c.Next()
		keyAfterRoute, _ = servermiddleware.GetAPIKeyFromContext(c)
	})
	router.Use(effectiveRouteMiddlewareForTest(t, newEffectiveRouteResolverForTest(&config.Config{}, map[int64]*service.Group{finalID: finalGroup}, nil)))
	router.POST("/v1/messages", func(c *gin.Context) {
		reachedHandler = true
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"gpt-5","messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.False(t, reachedHandler)
	require.Same(t, apiKey, keyAfterRoute)
	require.Same(t, originalGroup, keyAfterRoute.Group)
}

func (s compositeRouteRepoStub) ListByGroup(ctx context.Context, groupID int64, includeDisabled bool) ([]service.CompositeModelRoute, error) {
	routes := make([]service.CompositeModelRoute, 0, len(s.routes))
	for _, route := range s.routes {
		if route.GroupID != groupID {
			continue
		}
		if !includeDisabled && !route.Enabled {
			continue
		}
		routes = append(routes, route)
	}
	return routes, nil
}

func (s compositeRouteRepoStub) Create(ctx context.Context, route *service.CompositeModelRoute) error {
	return nil
}

func (s compositeRouteRepoStub) Update(ctx context.Context, route *service.CompositeModelRoute) error {
	return nil
}

func (s compositeRouteRepoStub) Delete(ctx context.Context, id int64) error {
	return nil
}

func (s compositeRouteRepoStub) DeleteByGroup(ctx context.Context, groupID int64) error {
	return nil
}

func TestCompositeTargetPlatformMiddlewareResolvesModelAndRestoresBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.HandlerFunc(servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
		groupID := int64(1)
		c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
			GroupID: &groupID,
			Group:   &service.Group{ID: groupID, Platform: service.PlatformComposite, Status: service.StatusActive},
			User:    &service.User{ID: 1},
		})
		c.Next()
	})))
	router.Use(compositeTargetPlatformMiddleware(effectiveCompositeRouteResolverForTest(nil)))
	router.POST("/", func(c *gin.Context) {
		platform, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context())
		require.True(t, ok)
		require.Equal(t, service.PlatformOpenAI, platform)

		body, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.JSONEq(t, `{"model":"gpt-5"}`, string(body))
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"model":"gpt-5"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestCompositeTargetPlatformMiddlewareUsesExplicitRouteAndRewritesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	resolver := service.NewCompositeRouteResolver(compositeRouteRepoStub{
		routes: []service.CompositeModelRoute{
			{
				ID:             1,
				GroupID:        1,
				PublicModel:    "openrouter/gpt-5",
				MatchType:      service.CompositeRouteMatchExact,
				TargetPlatform: service.PlatformOpenAI,
				UpstreamModel:  "gpt-5",
				Endpoint:       service.CompositeRouteEndpointAny,
				Priority:       100,
				Enabled:        true,
			},
		},
	})
	router.Use(gin.HandlerFunc(servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
		groupID := int64(1)
		c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
			GroupID: &groupID,
			Group:   &service.Group{ID: groupID, Platform: service.PlatformComposite, Status: service.StatusActive},
			User:    &service.User{ID: 1},
		})
		c.Next()
	})))
	router.Use(servermiddleware.UsageDetailCapture())
	router.Use(compositeTargetPlatformMiddleware(effectiveCompositeRouteResolverForTest(resolver)))
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		platform, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context())
		require.True(t, ok)
		require.Equal(t, service.PlatformOpenAI, platform)

		upstreamModel, ok := service.ResolvedUpstreamModelFromContext(c.Request.Context())
		require.True(t, ok)
		require.Equal(t, "gpt-5", upstreamModel)

		body, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.JSONEq(t, `{"model":"gpt-5","messages":[]}`, string(body))
		service.SetUsageRequestBody(c, service.RequestBodyPreviewSnapshot(string(body), int64(len(body))))
		detail := servermiddleware.BuildUsageDetailSnapshot(c)
		require.NotNil(t, detail)
		require.Contains(t, detail.RequestBody, "openrouter/gpt-5")
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"openrouter/gpt-5","messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestCompositeTargetPlatformMiddlewareUsesExplicitRouteForMultipartImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	resolver := service.NewCompositeRouteResolver(compositeRouteRepoStub{
		routes: []service.CompositeModelRoute{
			{
				ID:             1,
				GroupID:        1,
				PublicModel:    "image-alias",
				MatchType:      service.CompositeRouteMatchExact,
				TargetPlatform: service.PlatformOpenAI,
				UpstreamModel:  "gpt-image-1",
				Endpoint:       service.CompositeRouteEndpointImages,
				Priority:       100,
				Enabled:        true,
			},
		},
	})
	router.Use(gin.HandlerFunc(servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
		groupID := int64(1)
		c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
			GroupID: &groupID,
			Group:   &service.Group{ID: groupID, Platform: service.PlatformComposite, Status: service.StatusActive},
			User:    &service.User{ID: 1},
		})
		c.Next()
	})))
	router.Use(servermiddleware.UsageDetailCapture())
	router.Use(compositeTargetPlatformMiddleware(effectiveCompositeRouteResolverForTest(resolver)))
	router.POST("/v1/images/edits", func(c *gin.Context) {
		platform, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context())
		require.True(t, ok)
		require.Equal(t, service.PlatformOpenAI, platform)

		upstreamModel, ok := service.ResolvedUpstreamModelFromContext(c.Request.Context())
		require.True(t, ok)
		require.Equal(t, "gpt-image-1", upstreamModel)

		publicModel, ok := service.RequestedPublicModelFromContext(c.Request.Context())
		require.True(t, ok)
		require.Equal(t, "image-alias", publicModel)

		body, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "image-alias")
		service.SetUsageRequestBody(c, `{"kind":"image_metadata"}`)
		detail := servermiddleware.BuildUsageDetailSnapshot(c)
		require.Equal(t, `{"kind":"image_metadata"}`, detail.RequestBody)
		c.Status(http.StatusNoContent)
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "image-alias"))
	require.NoError(t, writer.WriteField("prompt", "draw"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestCompositeTargetPlatformMiddlewareBoundsOriginalBodySnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.HandlerFunc(servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
		groupID := int64(1)
		c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{GroupID: &groupID, Group: &service.Group{ID: groupID, Platform: service.PlatformComposite, Status: service.StatusActive}, User: &service.User{ID: 1}})
		c.Next()
	})))
	router.Use(servermiddleware.UsageDetailCapture())
	router.Use(compositeTargetPlatformMiddleware(effectiveCompositeRouteResolverForTest(nil)))
	router.POST("/v1/responses", func(c *gin.Context) {
		detail := servermiddleware.BuildUsageDetailSnapshot(c)
		require.NotNil(t, detail)
		require.LessOrEqual(t, len(detail.RequestBody), (5<<20)+1024)
		c.Status(http.StatusNoContent)
	})

	body := `{"model":"gpt-5","input":"` + strings.Repeat("x", (5<<20)+1) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestCompositeTargetPlatformMiddlewareRejectsOversizedRuntimeRouteModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	legacyUpstream := strings.Repeat("u", 150)
	resolver := service.NewCompositeRouteResolver(compositeRouteRepoStub{routes: []service.CompositeModelRoute{{
		GroupID:        1,
		PublicModel:    "legacy/",
		MatchType:      service.CompositeRouteMatchPrefix,
		TargetPlatform: service.PlatformOpenAI,
		UpstreamModel:  legacyUpstream,
		Endpoint:       service.CompositeRouteEndpointResponses,
		Enabled:        true,
	}}})

	for _, model := range []string{
		"gpt-" + strings.Repeat("x", 101),
		"legacy/gpt-5",
	} {
		t.Run(model[:6], func(t *testing.T) {
			reachedHandler := false
			router := gin.New()
			router.Use(func(c *gin.Context) {
				groupID := int64(1)
				c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{GroupID: &groupID, Group: &service.Group{ID: groupID, Platform: service.PlatformComposite, Status: service.StatusActive}, User: &service.User{ID: 1}})
				c.Next()
			})
			router.Use(compositeTargetPlatformMiddleware(effectiveCompositeRouteResolverForTest(resolver)))
			router.POST("/v1/responses", func(c *gin.Context) {
				reachedHandler = true
				c.Status(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":`+strconv.Quote(model)+`}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusServiceUnavailable, w.Code)
			require.False(t, reachedHandler)
		})
	}
}

func TestCompositeGeminiTargetPlatformMiddlewareUsesPathRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	resolver := service.NewCompositeRouteResolver(compositeRouteRepoStub{
		routes: []service.CompositeModelRoute{
			{
				ID:             1,
				GroupID:        1,
				PublicModel:    "openrouter/gemini-pro",
				MatchType:      service.CompositeRouteMatchExact,
				TargetPlatform: service.PlatformGemini,
				UpstreamModel:  "gemini-2.5-pro",
				Endpoint:       service.CompositeRouteEndpointGemini,
				Priority:       100,
				Enabled:        true,
			},
		},
	})
	router.Use(gin.HandlerFunc(servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
		groupID := int64(1)
		c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
			GroupID: &groupID,
			Group:   &service.Group{ID: groupID, Platform: service.PlatformComposite},
		})
		c.Next()
	})))
	router.Use(compositeGeminiTargetPlatformMiddleware(resolver))
	router.POST("/v1beta/models/*modelAction", func(c *gin.Context) {
		platform, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context())
		require.True(t, ok)
		require.Equal(t, service.PlatformGemini, platform)

		upstreamModel, ok := service.ResolvedUpstreamModelFromContext(c.Request.Context())
		require.True(t, ok)
		require.Equal(t, "gemini-2.5-pro", upstreamModel)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/openrouter/gemini-pro:generateContent", strings.NewReader(`{"contents":[]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
}
