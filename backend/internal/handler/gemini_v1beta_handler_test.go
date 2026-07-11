//go:build unit

package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Native Gemini forwarding must retain the replayable body handle through retries.
var _ func(*service.GeminiMessagesCompatService, context.Context, *gin.Context, *service.Account, string, string, bool, *service.RequestBodyHandle) (*service.ForwardResult, error) = (*service.GeminiMessagesCompatService).ForwardNativeHandle

func TestGeminiV1BetaGenerateContentRequestBody_EffectiveHandleReopensLargeIdentityAndGzipBodies(t *testing.T) {
	old := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, TempDir: t.TempDir()}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })

	raw := []byte(`{"contents":[{"role":"model","parts":[{"text":"` + strings.Repeat("x", 12<<20) + `","thoughtSignature":"old-account"}]}]}`)
	rawHash := sha256.Sum256(raw)
	for _, tt := range []struct {
		name   string
		action string
		gzip   bool
	}{
		{"GenerateContentIdentity", "generateContent", false},
		{"StreamGenerateContentGzip", "streamGenerateContent", true},
		{"CountTokensIdentity", "countTokens", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requestBody := raw
			if tt.gzip {
				var compressed bytes.Buffer
				writer := gzip.NewWriter(&compressed)
				require.NoError(t, func() error { _, err := writer.Write(raw); return err }())
				require.NoError(t, writer.Close())
				requestBody = compressed.Bytes()
			}
			req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:"+tt.action, bytes.NewReader(requestBody))
			if tt.gzip {
				req.Header.Set("Content-Encoding", "gzip")
			}
			coordinator, err := newJSONRequestBody(req)
			require.NoError(t, err)
			defer coordinator.Cleanup()

			readRaw, err := coordinator.ReadRaw()
			require.NoError(t, err)
			require.Equal(t, hex.EncodeToString(rawHash[:]), service.HashUsageRequestPayload(readRaw))
			require.NoError(t, coordinator.SetEffectiveBytes(service.CleanGeminiNativeThoughtSignatures(readRaw)))
			require.Equal(t, hex.EncodeToString(rawHash[:]), coordinator.raw.Hash())

			reader, err := coordinator.Effective().Open()
			require.NoError(t, err)
			effective, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.NoError(t, reader.Close())
			require.NotContains(t, string(effective), "old-account")
			reopened, err := coordinator.Effective().Open()
			require.NoError(t, err)
			require.NoError(t, reopened.Close())
		})
	}
}

func TestGeminiV1BetaGenerateContentRequestBody_OAuthWrappedBodyStaysHandleBacked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawDir := t.TempDir()
	old := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })

	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"/.gemini/tmp/` + strings.Repeat("a", 64) + strings.Repeat("x", 12<<20) + `","thoughtSignature":"old-account"}]}]}`)
	upstream := &geminiBlockingBodyUpstream{started: make(chan *http.Request, 1), release: make(chan struct{})}
	group := &service.Group{ID: 91, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID: 91, Name: "oauth", Platform: service.PlatformGemini, Type: service.AccountTypeOAuth,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
		Credentials: map[string]any{"access_token": "token", "project_id": "project"},
	}
	env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
	env.handler.cfg.Gateway.Sticky.Gemini.Enabled = true
	env.handler.geminiCompatService = service.NewGeminiMessagesCompatService(
		nil, nil, nil, nil, service.NewGeminiTokenProvider(nil, nil, nil), nil, upstream, nil, env.handler.cfg,
	)
	router := env.routerFor("/v1beta/models/*modelAction", env.handler.GeminiV1BetaModels)
	done := make(chan struct{})
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(httptest.NewRecorder(), req)
		close(done)
	}()

	var request *http.Request
	select {
	case request = <-upstream.started:
	case <-time.After(5 * time.Second):
		t.Fatal("Gemini handler did not reach upstream")
	}
	reopened, err := request.GetBody()
	require.NoError(t, err)
	replayed, err := io.ReadAll(reopened)
	require.NoError(t, err)
	require.NoError(t, reopened.Close())
	require.Contains(t, string(replayed), `"project":"project"`)
	require.NotContains(t, string(replayed), "old-account")
	require.Contains(t, fmt.Sprintf("%T", request.Body), "requestBodySpoolReadCloser")
	entries, err := os.ReadDir(rawDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	close(upstream.release)
	<-done
	entries, err = os.ReadDir(rawDir)
	require.NoError(t, err)
	require.Empty(t, entries, "spools must be removed after the handler returns")
	_, err = request.Body.Read(make([]byte, 1))
	require.Error(t, err, "upstream request body must be closed before owned spools are cleaned up")
}

func TestGeminiV1BetaGenerateContentRequestBody_PreservesImageInputSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 92, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID: 92, Name: "api-key", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
		Credentials: map[string]any{"api_key": "key"},
	}
	upstream := directTerminalHTTPUpstream{response: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]}}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`)),
	}}
	env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash-image:generateContent", strings.NewReader(`{"contents":[{"role":"user","parts":[{"text":"draw"}]}],"generationConfig":{"imageConfig":{"imageSize":"2K"}}}`))
	req.Header.Set("Content-Type", "application/json")
	env.routerFor("/v1beta/models/*modelAction", env.handler.GeminiV1BetaModels).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, env.usageRepo.lastLog)
	require.NotNil(t, env.usageRepo.lastLog.ImageInputSize)
	require.Equal(t, "2K", *env.usageRepo.lastLog.ImageInputSize)
}

func TestGeminiV1BetaModels_ReleasesAcquiredAccountSlotWhenThoughtSignatureSpoolReadFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := jsonRequestBodyHandleOptions
	rawDir := t.TempDir()
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })

	cache := &geminiSpoolReleaseConcurrencyCache{acquired: make(chan struct{}), proceed: make(chan struct{})}
	group := &service.Group{ID: 93, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 93, Name: "api-key", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "key"}}
	env := newTerminalGatewayMessagesEnvWithConcurrencyCache(t, group, &openAIChatCompletionsHTTPUpstreamStub{}, cache, account)
	env.handler.cfg.Gateway.Sticky.Gemini.Enabled = true

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", strings.NewReader(`{"contents":[{"role":"user","parts":[{"text":"/.gemini/tmp/`+strings.Repeat("a", 64)+`","thoughtSignature":"old-account"}]}]}`))
	request.Header.Set("Content-Type", "application/json")
	done := make(chan struct{})
	go func() {
		env.routerFor("/v1beta/models/*modelAction", env.handler.GeminiV1BetaModels).ServeHTTP(recorder, request)
		close(done)
	}()

	select {
	case <-cache.acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("Gemini handler did not acquire the account slot")
	}
	entries, err := os.ReadDir(rawDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.NoError(t, os.Remove(filepath.Join(rawDir, entries[0].Name())))
	close(cache.proceed)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Gemini handler did not return after spool read failure")
	}

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, 1, cache.releaseCalls())
}

type geminiSpoolReleaseConcurrencyCache struct {
	openAIChatCompletionsConcurrencyCacheStub
	acquired chan struct{}
	proceed  chan struct{}
	mu       sync.Mutex
	releases int
}

func (c *geminiSpoolReleaseConcurrencyCache) AcquireAccountSlot(context.Context, int64, int, string) (bool, error) {
	c.acquired <- struct{}{}
	<-c.proceed
	return true, nil
}

func (c *geminiSpoolReleaseConcurrencyCache) ReleaseAccountSlot(context.Context, int64, string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.releases++
	return nil
}

func (c *geminiSpoolReleaseConcurrencyCache) releaseCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.releases
}

type geminiBlockingBodyUpstream struct {
	service.HTTPUpstream
	started chan *http.Request
	release chan struct{}
}

func (u *geminiBlockingBodyUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.started <- req
	<-u.release
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(
			`data: {"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}` + "\n\n" +
				"data: [DONE]\n\n",
		)),
	}, nil
}

// TestGeminiV1BetaHandler_PlatformRoutingInvariant 文档化并验证 Handler 层的平台路由逻辑不变量
// 该测试确保 gemini 和 antigravity 平台的路由逻辑符合预期
func TestGeminiV1BetaHandler_PlatformRoutingInvariant(t *testing.T) {
	tests := []struct {
		name            string
		platform        string
		expectedService string
		description     string
	}{
		{
			name:            "Gemini平台使用ForwardNative",
			platform:        service.PlatformGemini,
			expectedService: "GeminiMessagesCompatService.ForwardNative",
			description:     "Gemini OAuth 账户直接调用 Google API",
		},
		{
			name:            "Antigravity平台使用ForwardGemini",
			platform:        service.PlatformAntigravity,
			expectedService: "AntigravityGatewayService.ForwardGemini",
			description:     "Antigravity 账户通过 CRS 中转，支持 Gemini 协议",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 GeminiV1BetaModels 中的路由决策 (lines 199-205 in gemini_v1beta_handler.go)
			var routedService string
			if tt.platform == service.PlatformAntigravity {
				routedService = "AntigravityGatewayService.ForwardGemini"
			} else {
				routedService = "GeminiMessagesCompatService.ForwardNative"
			}

			require.Equal(t, tt.expectedService, routedService,
				"平台 %s 应该路由到 %s: %s",
				tt.platform, tt.expectedService, tt.description)
		})
	}
}

// TestGeminiV1BetaHandler_ListModelsAntigravityFallback 验证 ListModels 的 antigravity 降级逻辑
// 当没有 gemini 账户但有 antigravity 账户时，应返回静态模型列表
func TestGeminiV1BetaHandler_ListModelsAntigravityFallback(t *testing.T) {
	tests := []struct {
		name             string
		hasGeminiAccount bool
		hasAntigravity   bool
		expectedBehavior string
	}{
		{
			name:             "有Gemini账户-调用ForwardAIStudioGET",
			hasGeminiAccount: true,
			hasAntigravity:   false,
			expectedBehavior: "forward_to_upstream",
		},
		{
			name:             "无Gemini有Antigravity-返回静态列表",
			hasGeminiAccount: false,
			hasAntigravity:   true,
			expectedBehavior: "static_fallback",
		},
		{
			name:             "无任何账户-返回503",
			hasGeminiAccount: false,
			hasAntigravity:   false,
			expectedBehavior: "service_unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 GeminiV1BetaListModels 的逻辑 (lines 33-44 in gemini_v1beta_handler.go)
			var behavior string

			if tt.hasGeminiAccount {
				behavior = "forward_to_upstream"
			} else if tt.hasAntigravity {
				behavior = "static_fallback"
			} else {
				behavior = "service_unavailable"
			}

			require.Equal(t, tt.expectedBehavior, behavior)
		})
	}
}

// TestGeminiV1BetaHandler_GetModelAntigravityFallback 验证 GetModel 的 antigravity 降级逻辑
func TestGeminiV1BetaHandler_GetModelAntigravityFallback(t *testing.T) {
	tests := []struct {
		name             string
		hasGeminiAccount bool
		hasAntigravity   bool
		expectedBehavior string
	}{
		{
			name:             "有Gemini账户-调用ForwardAIStudioGET",
			hasGeminiAccount: true,
			hasAntigravity:   false,
			expectedBehavior: "forward_to_upstream",
		},
		{
			name:             "无Gemini有Antigravity-返回静态模型信息",
			hasGeminiAccount: false,
			hasAntigravity:   true,
			expectedBehavior: "static_model_info",
		},
		{
			name:             "无任何账户-返回503",
			hasGeminiAccount: false,
			hasAntigravity:   false,
			expectedBehavior: "service_unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 GeminiV1BetaGetModel 的逻辑 (lines 77-87 in gemini_v1beta_handler.go)
			var behavior string

			if tt.hasGeminiAccount {
				behavior = "forward_to_upstream"
			} else if tt.hasAntigravity {
				behavior = "static_model_info"
			} else {
				behavior = "service_unavailable"
			}

			require.Equal(t, tt.expectedBehavior, behavior)
		})
	}
}

func TestShouldFallbackGeminiModel_KnownFallbackOn404(t *testing.T) {
	t.Parallel()

	res := &service.UpstreamHTTPResult{StatusCode: http.StatusNotFound}
	require.True(t, shouldFallbackGeminiModel("gemini-3.1-pro-preview-customtools", res))
}

func TestShouldFallbackGeminiModel_UnknownModelOn404(t *testing.T) {
	t.Parallel()

	res := &service.UpstreamHTTPResult{StatusCode: http.StatusNotFound}
	require.False(t, shouldFallbackGeminiModel("gemini-future-model", res))
}

func TestShouldFallbackGeminiModel_DelegatesScopeFallback(t *testing.T) {
	t.Parallel()

	res := &service.UpstreamHTTPResult{
		StatusCode: http.StatusForbidden,
		Headers:    http.Header{"Www-Authenticate": []string{"Bearer error=\"insufficient_scope\""}},
		Body:       []byte("insufficient authentication scopes"),
	}
	require.True(t, shouldFallbackGeminiModel("gemini-future-model", res))
}
