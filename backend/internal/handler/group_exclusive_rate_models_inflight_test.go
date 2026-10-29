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
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type exclusiveInflightRateRepo struct {
	service.UserGroupRateRepository
}

func (*exclusiveInflightRateRepo) GetByUserAndGroup(context.Context, int64, int64) (*float64, error) {
	rate := .12
	return &rate, nil
}

type exclusiveGeminiUpstream struct {
	service.HTTPUpstream
	cache    *handlerInflightCache
	observed float64
}

func (u *exclusiveGeminiUpstream) Do(req *http.Request, proxy string, accountID int64, concurrency int) (*http.Response, error) {
	u.observed = u.cache.total()
	if u.HTTPUpstream != nil {
		return u.HTTPUpstream.Do(req, proxy, accountID, concurrency)
	}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":1}}`))}, nil
}

func TestExclusiveRateModelsGeminiMappedInflightEntries(t *testing.T) {
	for _, endpoint := range []string{"native", "cache"} {
		for _, selected := range []string{"client-A", "gemini-2.5-pro"} {
			t.Run(endpoint+"/"+selected, func(t *testing.T) {
				channels := service.NewChannelService(&openAIWSUsageHandlerChannelRepoStub{channels: []service.Channel{{ID: 1, Status: service.StatusActive, GroupIDs: []int64{42}, ModelMapping: map[string]map[string]string{service.PlatformGemini: {"client-A": "gemini-2.5-pro"}}, BillingModelSource: service.BillingModelSourceChannelMapped}}, groupPlatforms: map[int64]string{42: service.PlatformGemini}}, nil, nil, nil, nil)
				env := newCachedBillingEnvWithRates(t, 500, &exclusiveInflightRateRepo{}, channels, 900)
				env.group.RateMultiplier = .225
				env.group.ExclusiveRateModels = service.GroupExclusiveRateModels{Enabled: true, Models: []string{selected}}
				cache := newHandlerInflightCache(500)
				cfg := &config.Config{Billing: config.BillingConfig{InflightReservation: config.InflightReservationConfig{Enabled: true, TTLSeconds: 60, DefaultMaxTokens: 1000}}}
				billing := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
				t.Cleanup(billing.Stop)
				env.h.billingCacheService = billing
				body := `{"contents":[{"role":"user","parts":[{"text":"hi"}]}],"generationConfig":{"maxOutputTokens":1000}}`
				native := &exclusiveGeminiUpstream{cache: cache}
				env.h.geminiCompatService = service.NewGeminiMessagesCompatService(&cachedBillingAccountRepo{accounts: env.accounts}, nil, nil, nil, nil, nil, native, nil, cfg)
				var observed float64
				env.upstream.createBody = cachedBillingCreateBody(time.Now().Add(time.Hour))
				if endpoint == "cache" {
					native.HTTPUpstream = env.upstream
					env.h.geminiCachedContentService = service.NewGeminiCachedContentService(env.repo, env.h.geminiCompatService, cfg)
					body = `{"model":"models/client-A","ttl":"3600s","contents":[{"role":"user","parts":[{"text":"hi"}]}],"generationConfig":{"maxOutputTokens":1000}}`
				}
				estimate := tokenInflightEstimate("gemini-2.5-pro", []byte(body))
				estimate.RequestedModel = "client-A"
				want, priced := env.h.gatewayService.EstimateInflightReservation(context.Background(), env.apiKey, estimate)
				require.True(t, priced)
				require.Greater(t, want, 0.)
				var status int
				var response string
				if endpoint == "native" {
					rec := env.serve(http.MethodPost, "/v1beta/models/client-A:generateContent", body, gin.Params{{Key: "modelAction", Value: "/client-A:generateContent"}}, env.h.GeminiV1BetaModels)
					status, response = rec.Code, rec.Body.String()
					observed = native.observed
				} else {
					rec := env.serve(http.MethodPost, "/v1beta/cachedContents", body, nil, env.h.GeminiCachedContentsCreate)
					status, response = rec.Code, rec.Body.String()
					observed = native.observed
				}
				require.Equal(t, http.StatusOK, status, response)
				require.InDelta(t, want, observed, 1e-12, "real mapped entry must reserve using the client model")
				env.waitForUsage(t, 1)
				rate := .225
				if selected == "client-A" {
					rate = .12
				}
				require.Equal(t, rate, env.usage.first().RateMultiplier)
				require.Eventually(t, func() bool { return cache.count() == 0 }, time.Second, time.Millisecond)
			})
		}
	}
}

func TestExclusiveRateModelsHTTPFreezesOriginalAcrossAccountReestimates(t *testing.T) {
	for _, kind := range []service.InflightEstimateKind{service.InflightEstimateToken, service.InflightEstimateImage, service.InflightEstimateVideo} {
		t.Run(string(rune('0'+kind)), func(t *testing.T) {
			cache := newHandlerInflightCache(100)
			cfg := &config.Config{Billing: config.BillingConfig{InflightReservation: config.InflightReservationConfig{Enabled: true, TTLSeconds: 60}}}
			billing := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
			t.Cleanup(billing.Stop)
			key := &service.APIKey{User: &service.User{ID: 1}, Group: &service.Group{Platform: service.PlatformComposite}}
			c := newInflightTestGinContext()
			ctx, _ := service.WithGatewayTokenRequestPricing(c.Request.Context(), "client-A")
			if kind != service.InflightEstimateToken {
				ctx = context.WithValue(c.Request.Context(), ctxkey.RequestedPublicModel, "client-A")
			}
			c.Request = c.Request.WithContext(ctx)
			est := &countingEstimator{cost: 1, priced: true}
			done, err := reserveInflightBalance(c, billing, est, key, nil, service.InflightEstimateRequest{Model: "mapped-B", Kind: kind, Units: 1})
			require.NoError(t, err)
			defer done()
			require.Equal(t, "client-A", est.last.RequestedModel, "initial reservation must freeze the public/client ID")
			// Account retries must keep the frozen model even if the routing context changes.
			c.Request = c.Request.WithContext(context.WithValue(context.Background(), ctxkey.RequestedPublicModel, "changed-routing-model"))
			require.NoError(t, reserveInflightBalanceForAccount(c, billing, est, key, nil, service.ChannelMappingResult{}, "account-C"))
			require.Equal(t, "client-A", est.last.RequestedModel)
		})
	}
}

func TestExclusiveRateModelsCompositeMediaInflightEntries(t *testing.T) {
	for _, platform := range []string{service.PlatformOpenAI, service.PlatformGrok} {
		t.Run(platform, func(t *testing.T) {
			model := "gpt-image-1"
			if platform == service.PlatformGrok {
				model = "grok-imagine-image-quality"
			}
			price := 1.
			group := &service.Group{ID: 42, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true, RateMultiplier: .225, ImagePrice1K: &price, ImagePrice2K: &price, ImagePrice4K: &price, ExclusiveRateModels: service.GroupExclusiveRateModels{Enabled: true, Models: []string{"image-alias"}}}
			account := &service.Account{ID: 900, Platform: platform, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "key", "base_url": "https://api.x.ai/v1"}}
			cache := newHandlerInflightCache(100)
			upstream := &exclusiveGeminiUpstream{HTTPUpstream: &grokMediaRequestRecorder{}, cache: cache}
			repo := &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account}}}
			env := newTerminalUsageOpenAIEnvWithUpstreamAndGatewayCache(t, group, repo, upstream, openAIChatCompletionsGatewayCacheStub{}, &exclusiveInflightRateRepo{})
			env.handler.cfg.RunMode = config.RunModeStandard
			env.handler.cfg.Billing.InflightReservation = config.InflightReservationConfig{Enabled: true, TTLSeconds: 60}
			billing := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, env.handler.cfg, nil)
			t.Cleanup(billing.Stop)
			env.handler.billingCacheService = billing
			env.apiKey.User.Balance = 100
			ctx := service.WithCompositeRouteDecision(context.Background(), service.CompositeRouteDecision{Matched: true, GroupID: group.ID, PublicModel: "image-alias", TargetPlatform: platform, UpstreamModel: model, Endpoint: service.CompositeRouteEndpointImages})
			request := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"image-alias","prompt":"draw","n":1}`)).WithContext(ctx)
			request.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			forward := env.handler.Images
			if platform == service.PlatformGrok {
				forward = env.handler.GrokImages
			}
			env.router("/v1/images/generations", forward).ServeHTTP(rec, request)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.InDelta(t, .12, upstream.observed, 1e-12, "composite public alias selects the user rate, independent of the routed media model")
			require.Eventually(t, func() bool { return cache.count() == 0 }, time.Second, time.Millisecond)
		})
	}
}

func TestExclusiveRateModelsWSTurnPolicyRefreshInflight(t *testing.T) {
	for _, transport := range []string{service.OpenAIWSIngressModePassthrough, service.OpenAIWSIngressModeDedicated, "http-bridge"} {
		t.Run(transport, func(t *testing.T) {
			repo := newWSGroupPricingAPIKeyRepoStub(.225)
			zero, output := 0., .001
			repo.group.ModelPricing = []service.ChannelModelPricing{{Models: []string{"gpt-5.4"}, InputPrice: &zero, OutputPrice: &output}}
			keys := newWSGroupPricingAPIKeyService(repo)
			cache := newHandlerInflightCache(100)
			observed := make(chan float64, 2)
			mode := transport
			if transport == "http-bridge" {
				mode = service.OpenAIWSIngressModeDedicated
			}
			got := runOpenAIResponsesWebSocketUsageLogCase(t, openAIResponsesWSUsageLogCase{
				inflightCache: cache, ingressMode: mode, httpBridge: transport == "http-bridge", apiKeyService: keys, apiKeyCredential: wsGroupPricingKey, userGroupRateRepo: &exclusiveInflightRateRepo{},
				firstPayload:    `{"type":"response.create","model":"gpt-5.4","max_output_tokens":1000,"input":"hi"}`,
				secondPayload:   `{"type":"response.create","model":"gpt-5.4","max_output_tokens":1000,"input":"hi"}`,
				observeUpstream: func([]byte) { observed <- cache.total() },
				afterFirstUpstreamRequest: func(*service.ChannelService) error {
					return repo.adminUpdate(keys, func(_ *service.APIKey, g *service.Group) {
						g.ExclusiveRateModels = service.GroupExclusiveRateModels{Enabled: true, Models: []string{"other-model"}}
					})
				},
			})
			require.Len(t, got.logs, 2)
			require.Equal(t, []float64{.12, .225}, []float64{got.logs[0].RateMultiplier, got.logs[1].RateMultiplier})
			require.InDelta(t, .12, <-observed, 1e-12)
			require.InDelta(t, .225, <-observed, 1e-12, "next real WS turn must reserve with the same refreshed policy used by its final charge")
			require.Eventually(t, func() bool { return cache.count() == 0 }, time.Second, time.Millisecond)
		})
	}
}
