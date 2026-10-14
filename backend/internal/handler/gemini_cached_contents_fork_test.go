//go:build unit

package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type cachedForkQuotaRepo struct {
	service.UserPlatformQuotaRepository
}

type cachedForkQuotaCache struct {
	cachedBillingReservationCache
	entry *service.UserPlatformQuotaCacheEntry
}

func (c *cachedForkQuotaCache) GetUserPlatformQuotaCache(context.Context, int64, string) (*service.UserPlatformQuotaCacheEntry, bool, error) {
	return c.entry, true, nil
}

func TestGeminiCachedContentsPatch_PlatformHeadroomRejectionReleasesHolds(t *testing.T) {
	env := newCachedBillingEnv(t, 1000, 900)
	env.seed(time.Now().Add(time.Hour))
	limit := 110.0
	now := time.Now()
	day, week := timezone.StartOfDay(now), timezone.StartOfWeek(now)
	cache := &cachedForkQuotaCache{cachedBillingReservationCache: cachedBillingReservationCache{balance: 1000},
		entry: &service.UserPlatformQuotaCacheEntry{SchemaVersion: service.UserPlatformQuotaCacheSchemaV1,
			DailyLimitUSD: &limit, DailyWindowStart: &day, WeeklyWindowStart: &week, MonthlyWindowStart: &now}}
	cfg := &config.Config{}
	cfg.Billing.InflightReservation.Enabled = true
	bcs := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, cachedForkQuotaRepo{})
	t.Cleanup(bcs.Stop)
	env.h.billingCacheService = bcs
	second := *env.repo.records["abc"]
	second.ID, second.PublicID, second.UpstreamName = 101, "def", "cachedContents/up2"
	env.repo.records["def"] = &second
	env.upstream.beforePost = func(n int) {
		if n == 1 {
			rec := env.patchTTL("def", "86400s")
			require.Equal(t, http.StatusTooManyRequests, rec.Code, rec.Body.String())
			require.Len(t, cache.reserved["platform:3:gemini"], 1, "rejected PATCH releases its own hold")
		}
	}
	rec := env.patch()
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, 1, env.upstream.methodCalls(http.MethodPatch))
	env.waitForUsage(t, 1)
	for _, holds := range cache.reserved {
		require.Empty(t, holds, "successful mandatory billing and rejected requests both release")
	}
	// The request's resolved target also governs fixed-charge reservations.
	env.serve(http.MethodPatch, "/", `{}`, nil, func(c *gin.Context) {
		c.Request = c.Request.WithContext(service.WithResolvedTargetPlatform(c.Request.Context(), service.PlatformOpenAI))
		done, err := env.h.reserveGeminiCachedContentCharge(c, env.apiKey, nil, 1)
		require.NoError(t, err)
		require.Len(t, cache.reserved["platform:3:openai"], 1)
		done()
	})
}

func TestGeminiCachedContentsCreate_PlatformAmountRejectionBillsInputOnly(t *testing.T) {
	env := newCachedBillingEnv(t, 1000, 900)
	env.upstream.createBody = cachedBillingCreateBody(time.Now().Add(24 * time.Hour))
	limit := 10.0
	now := time.Now()
	day, week := timezone.StartOfDay(now), timezone.StartOfWeek(now)
	cache := &cachedForkQuotaCache{cachedBillingReservationCache: cachedBillingReservationCache{balance: 1000},
		entry: &service.UserPlatformQuotaCacheEntry{SchemaVersion: service.UserPlatformQuotaCacheSchemaV1,
			DailyLimitUSD: &limit, DailyWindowStart: &day, WeeklyWindowStart: &week, MonthlyWindowStart: &now}}
	cfg := &config.Config{}
	cfg.Billing.InflightReservation.Enabled = true
	bcs := service.NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, cachedForkQuotaRepo{})
	t.Cleanup(bcs.Stop)
	env.h.billingCacheService = bcs
	rec := env.create()
	require.Equal(t, http.StatusTooManyRequests, rec.Code, rec.Body.String())
	require.Equal(t, 1, env.upstream.methodCalls(http.MethodDelete))
	require.Empty(t, env.repo.records)
	require.InDelta(t, cachedBillingTokens*cachedBillingInputPrice, env.waitForUsage(t, 1)[0], 1e-9)
	for _, holds := range cache.reserved {
		require.Empty(t, holds)
	}
}

func TestGeminiCachedContentsCreate_GzipBodySpoolCleanup(t *testing.T) {
	env := newCachedBillingEnv(t, 500, 900)
	env.upstream.createBody = cachedBillingCreateBody(time.Now().Add(24 * time.Hour))
	dir := t.TempDir()
	old := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: dir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write([]byte(`{"model":"models/gemini-2.5-pro","contents":[{"parts":[{"text":"doc"}]}]}`))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	rec := env.serve(http.MethodPost, "/v1beta/cachedContents", compressed.String(), nil, func(c *gin.Context) {
		c.Request.Header.Set("Content-Encoding", "gzip")
		env.h.GeminiCachedContentsCreate(c)
		require.Equal(t, http.NoBody, c.Request.Body, "request body no longer retains compressed input")
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries, "temporary request spool is removed")
}

func TestGeminiCachedContentNativeBinding_UsesRewrittenHandleWithoutSticky(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	old := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: dir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })
	group := &service.Group{ID: 99, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	bound := &service.Account{ID: 99, Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
		AccountGroups: []service.AccountGroup{{GroupID: group.ID}}, Credentials: map[string]any{"api_key": "bound-key"}}
	sticky := *bound
	sticky.ID, sticky.Credentials = 100, map[string]any{"api_key": "sticky-key"}
	cliSession := strings.Repeat("a", 64)
	cache := &geminiStickyGatewayCacheStub{sessionBindings: map[string]int64{"gemini:" + cliSession: sticky.ID}}
	upstream := newGeminiBlockingBodyUpstream(t, nil)
	env := newTerminalGatewayMessagesEnvWithGatewayCache(t, group, upstream, openAIChatCompletionsConcurrencyCacheStub{}, cache, bound, &sticky)
	env.handler.cfg.Gateway.Sticky.Gemini.Enabled = true
	repo := &geminiCachedContentHandlerRepoStub{}
	record := seedGeminiCachedContent(repo, "abc", env.apiKey.ID, time.Now().Add(time.Hour))
	record.AccountID, record.GroupID, record.Model = bound.ID, group.ID, "gemini-2.5-flash"
	record.UpstreamModel = "models/gemini-2.5-flash"
	env.handler.geminiCachedContentService = service.NewGeminiCachedContentService(repo, env.handler.geminiCompatService, env.handler.cfg)
	body := `{"cachedContent":"cachedContents/abc","contents":[{"role":"user","parts":[{"text":"/.gemini/tmp/` + cliSession + `"}]}]}`
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", strings.NewReader(body))
		env.routerFor("/v1beta/models/*modelAction", env.handler.GeminiV1BetaModels).ServeHTTP(rec, req)
		close(done)
	}()
	var req *http.Request
	select {
	case req = <-upstream.started:
	case <-time.After(5 * time.Second):
		t.Fatal("bound native request did not reach upstream")
	}
	reopened, err := req.GetBody()
	require.NoError(t, err)
	forwarded, err := io.ReadAll(reopened)
	require.NoError(t, err)
	require.NoError(t, reopened.Close())
	require.Contains(t, string(forwarded), `"cachedContent":"cachedContents/upstreamabc"`)
	require.Equal(t, "bound-key", req.Header.Get("X-Goog-Api-Key"))
	require.Empty(t, cache.getCalls, "cached references skip CLI and digest sticky lookup")
	upstream.Release()
	requireGeminiHandlerDone(t, done)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Empty(t, cache.setCalls)
	require.Empty(t, cache.refreshCalls)
	select {
	case usage := <-env.usageRepo.created:
		require.Equal(t, bound.ID, usage.AccountID, "usage is attributed to the bound account")
	case <-time.After(time.Second):
		t.Fatal("bound cache usage was not recorded")
	}
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries)
}
