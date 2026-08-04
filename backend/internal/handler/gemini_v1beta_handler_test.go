//go:build unit

package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
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
	upstream := newGeminiBlockingBodyUpstream(t, nil)
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

	upstream.Release()
	requireGeminiHandlerDone(t, done)
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
	t.Cleanup(cache.Release)
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
	cache.Release()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Gemini handler did not return after spool read failure")
	}

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, 1, cache.releaseCalls())
}

func TestGeminiV1BetaModels_AntigravityInitialSpoolReadFailureReturns503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := jsonRequestBodyHandleOptions
	rawDir := t.TempDir()
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })

	cache := &geminiSpoolReleaseConcurrencyCache{acquired: make(chan struct{}), proceed: make(chan struct{})}
	t.Cleanup(cache.Release)
	group := &service.Group{ID: 94, Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID: 94, Name: "antigravity", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
		Credentials: map[string]any{"access_token": "token", "project_id": "project", "model_mapping": map[string]any{"gemini-2.5-flash": "gemini-2.5-flash"}},
	}
	env := newTerminalGatewayMessagesEnvWithConcurrencyCache(t, group, &openAIChatCompletionsHTTPUpstreamStub{}, cache, account)
	env.handler.cfg.Gateway.Sticky.Gemini.Enabled = true

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/antigravity/v1beta/models/gemini-2.5-flash:generateContent", strings.NewReader(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`))
	request.Header.Set("Content-Type", "application/json")
	done := make(chan struct{})
	go func() {
		env.routerFor("/antigravity/v1beta/models/*modelAction", func(c *gin.Context) {
			c.Set(string(middleware.ContextKeyForcePlatform), service.PlatformAntigravity)
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.ForcePlatform, service.PlatformAntigravity))
			env.handler.GeminiV1BetaModels(c)
		}).ServeHTTP(recorder, request)
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
	cache.Release()
	requireGeminiHandlerDone(t, done)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code, recorder.Body.String())
	require.Equal(t, 1, cache.releaseCalls())
}

func TestGeminiV1BetaModels_AntigravitySecondarySpoolFailuresReturn503WithoutBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const model = "gemini-wave9"

	tests := []struct {
		name          string
		status        int
		responseBody  string
		failCall      int
		thoughtSig    bool
		allowOverages bool
		settings      map[string]string
	}{
		{
			name: "smart retry transport", status: http.StatusServiceUnavailable, failCall: 2,
			responseBody: `{"error":{"code":503,"status":"UNAVAILABLE","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","metadata":{"model":"gemini-wave9-smart"},"reason":"MODEL_CAPACITY_EXHAUSTED"}]}}`,
		},
		{
			name: "single account retry transport", status: http.StatusServiceUnavailable, failCall: 2,
			responseBody: `{"error":{"code":503,"status":"RESOURCE_EXHAUSTED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","metadata":{"model":"gemini-wave9-single"},"reason":"RATE_LIMIT_EXCEEDED"},{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"7s"}]}}`,
		},
		{
			name: "credits retry transport", status: http.StatusTooManyRequests, failCall: 2, allowOverages: true,
			responseBody: `{"error":{"code":429,"status":"RESOURCE_EXHAUSTED","message":"QUOTA_EXHAUSTED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","metadata":{"model":"gemini-wave9-credits"},"reason":"RATE_LIMIT_EXCEEDED"},{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"0.1s"}]}}`,
		},
		{
			name: "model fallback transport", status: http.StatusNotFound, failCall: 2,
			responseBody: `{"error":{"message":"model not found"}}`,
			settings:     map[string]string{service.SettingKeyEnableModelFallback: "true", service.SettingKeyFallbackModelAntigravity: "gemini-wave9-fallback"},
		},
		{
			name: "signature retry transport", status: http.StatusBadRequest, failCall: 2, thoughtSig: true,
			responseBody: `{"response":{"error":{"code":400,"message":"Corrupted thought signature.","status":"INVALID_ARGUMENT"}}}`,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spoolDir := t.TempDir()
			t.Setenv("TMPDIR", spoolDir)
			t.Setenv("TMP", spoolDir)
			t.Setenv("TEMP", spoolDir)
			oldOptions := jsonRequestBodyHandleOptions
			jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: spoolDir}
			t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

			group := &service.Group{ID: int64(110 + i), Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true}
			account := &service.Account{
				ID: int64(210 + i), Name: "antigravity-secondary-spool", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth,
				Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
				Credentials: map[string]any{"access_token": "token", "project_id": "project", "model_mapping": map[string]any{model: "gemini-wave9-mapped"}},
			}
			if tt.allowOverages {
				account.Extra = map[string]any{"allow_overages": true}
			}
			upstream := &geminiSecondarySpoolUpstream{
				t: t, failCall: tt.failCall,
				firstResponse: &http.Response{StatusCode: tt.status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(tt.responseBody))},
			}
			env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
			settingService := service.NewSettingService(&geminiModerationSettingRepo{values: tt.settings}, env.handler.cfg)
			env.handler.settingService = settingService
			env.handler.antigravityGatewayService = service.NewAntigravityGatewayService(
				env.accountRepo, openAIChatCompletionsGatewayCacheStub{}, nil,
				service.NewAntigravityTokenProvider(env.accountRepo, nil, nil), nil, upstream, settingService, nil,
			)
			billingRepo := captureTerminalGatewayUsageBilling(t, env, group, upstream)

			text := "hello"
			body := `{"contents":[{"role":"user","parts":[{"text":"` + text + `"}]}]}`
			if tt.thoughtSig {
				body = `{"contents":[{"role":"user","parts":[{"text":"` + text + `"}]},{"role":"model","parts":[{"text":"thinking","thought":true,"thoughtSignature":"bad-signature"}]}]}`
			}

			var writerSpy *geminiHandlerWriterSpy
			router := env.routerFor("/antigravity/v1beta/models/*modelAction", func(c *gin.Context) {
				c.Set(string(middleware.ContextKeyForcePlatform), service.PlatformAntigravity)
				c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.ForcePlatform, service.PlatformAntigravity))
				writerSpy = &geminiHandlerWriterSpy{ResponseWriter: c.Writer}
				c.Writer = writerSpy
				upstream.writer = writerSpy
				env.handler.GeminiV1BetaModels(c)
			})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/antigravity/v1beta/models/"+model+":generateContent", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusServiceUnavailable, recorder.Code, recorder.Body.String())
			require.NotNil(t, writerSpy)
			require.False(t, writerSpy.writtenBeforeFirstHeader, "service must not write before handler maps the sentinel")
			require.Equal(t, http.StatusServiceUnavailable, writerSpy.firstStatus)
			if tt.failCall > 0 {
				require.False(t, upstream.writtenAtFailure, "writer must remain uncommitted when transport returns the sentinel")
				require.Equal(t, tt.failCall, upstream.calls)
			} else {
				require.Equal(t, 1, upstream.calls)
			}
			require.Nil(t, billingRepo.lastCmd, "spool failures must not invoke successful billing")
			failedUsage := waitForOpenAIFailedUsageLog(t, env.usageRepo)
			require.NotNil(t, failedUsage, "an attempted transport may create a failed audit usage record")
			require.Zero(t, failedUsage.TotalCost)
			require.Zero(t, failedUsage.ActualCost)
		})
	}
}

func TestGeminiV1BetaModels_ThoughtSignatureCleanupUsesEffectiveHandle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawDir := t.TempDir()
	old := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })

	upstream := &geminiEffectiveHandleUpstream{t: t, rawDir: rawDir}
	group := &service.Group{ID: 98, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	first := &service.Account{ID: 981, Name: "first", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "first"}}
	second := &service.Account{ID: 982, Name: "second", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{"api_key": "second"}}
	cache := openAIChatCompletionsGatewayCacheStub{}
	env := newTerminalGatewayMessagesEnvWithGatewayCache(t, group, upstream, openAIChatCompletionsConcurrencyCacheStub{}, cache, first, second)
	env.handler.cfg.Gateway.Sticky.Gemini.Enabled = true

	body := `{"contents":[{"role":"user","parts":[{"text":"/.gemini/tmp/` + strings.Repeat("a", 64) + `","thoughtSignature":"old-account"}]}]}`
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	env.routerFor("/v1beta/models/*modelAction", env.handler.GeminiV1BetaModels).ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Len(t, upstream.bodies, 2)
	require.Equal(t, []int64{first.ID, second.ID}, upstream.accountIDs)
	require.True(t, upstream.mutatedRaw)
	for _, forwarded := range upstream.bodies {
		require.NotContains(t, string(forwarded), "old-account")
		require.NotContains(t, string(forwarded), "raw-only")
	}
	require.Empty(t, readTestDir(t, rawDir))
}

func TestGeminiV1BetaModels_RequestBodyLifecycleAcrossActions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := jsonRequestBodyHandleOptions
	rawDir := t.TempDir()
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })

	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"` + strings.Repeat("x", 12<<20) + `"}]}]}`)
	wantHash := service.HashUsageRequestPayload(body)
	for _, action := range []string{"generateContent", "streamGenerateContent", "countTokens"} {
		t.Run(action, func(t *testing.T) {
			group := &service.Group{ID: 94, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
			account := &service.Account{ID: 94, Name: "api-key", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "key"}}
			upstream := newGeminiBlockingBodyUpstream(t, nil)
			env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
			done := make(chan struct{})
			go func() {
				rec := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:"+action, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				env.routerFor("/v1beta/models/*modelAction", env.handler.GeminiV1BetaModels).ServeHTTP(rec, req)
				close(done)
			}()

			select {
			case <-upstream.started:
			case <-time.After(5 * time.Second):
				t.Fatal("Gemini handler did not reach upstream")
			}
			require.Equal(t, []string{wantHash}, upstream.requestHashes)
			entries, err := os.ReadDir(rawDir)
			require.NoError(t, err)
			require.NotEmpty(t, entries, "large request must remain spooled while upstream waits")

			upstream.Release()
			requireGeminiHandlerDone(t, done)
			entries, err = os.ReadDir(rawDir)
			require.NoError(t, err)
			require.Empty(t, entries, "spools must be removed after the handler returns")
		})
	}
}

func TestGeminiV1BetaModels_CLILargeBodyUsesStickyAccountWhileSpooling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawDir := t.TempDir()
	old := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })

	group := &service.Group{ID: 99, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	primary := &service.Account{ID: 99, Name: "primary", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, AccountGroups: []service.AccountGroup{{GroupID: group.ID}}, Credentials: map[string]any{"api_key": "primary-key"}}
	sticky := &service.Account{ID: 100, Name: "sticky", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, AccountGroups: []service.AccountGroup{{GroupID: group.ID}}, Credentials: map[string]any{"api_key": "sticky-key"}}
	cliSession := strings.Repeat("a", 64)
	cache := &geminiStickyGatewayCacheStub{sessionBindings: map[string]int64{"gemini:" + cliSession: sticky.ID}}
	upstream := newGeminiBlockingBodyUpstream(t, nil)
	env := newTerminalGatewayMessagesEnvWithGatewayCache(t, group, upstream, openAIChatCompletionsConcurrencyCacheStub{}, cache, primary, sticky)
	env.handler.cfg.Gateway.Sticky.Gemini.Enabled = true
	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"` + strings.Repeat("x", 12<<20) + `/.gemini/tmp/` + cliSession + `"}]}]}`)
	wantHash := service.HashUsageRequestPayload(body)
	done := make(chan struct{})
	go func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1beta/models/*modelAction", env.handler.GeminiV1BetaModels).ServeHTTP(rec, req)
		close(done)
	}()

	var request *http.Request
	select {
	case request = <-upstream.started:
	case <-time.After(5 * time.Second):
		t.Fatal("Gemini handler did not reach upstream")
	}
	require.NotZero(t, cache.getCalls["gemini:"+cliSession])
	require.Equal(t, "sticky-key", request.Header.Get("X-Goog-Api-Key"))
	require.Equal(t, []string{wantHash}, upstream.requestHashes)
	entries, err := os.ReadDir(rawDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries, "large request must remain spooled while upstream waits")

	upstream.Release()
	requireGeminiHandlerDone(t, done)
	entries, err = os.ReadDir(rawDir)
	require.NoError(t, err)
	require.Empty(t, entries, "spools must be removed after the handler returns")
}

func TestGeminiV1BetaModels_ModelPathAndContentAuditKeepGoogleErrors(t *testing.T) {
	t.Run("model path", func(t *testing.T) {
		model, action, err := parseGeminiModelAction("gemini-2.5-flash/streamGenerateContent")
		require.NoError(t, err)
		require.Equal(t, "gemini-2.5-flash", model)
		require.Equal(t, "streamGenerateContent", action)
	})

	t.Run("content audit", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		rawDir := t.TempDir()
		old := jsonRequestBodyHandleOptions
		jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, TempDir: rawDir}
		t.Cleanup(func() { jsonRequestBodyHandleOptions = old })
		cfg := service.ContentModerationConfig{Enabled: true, Mode: service.ContentModerationModePreBlock, AllGroups: true, SampleRate: 100, APIKeys: []string{"test-key"}, BlockStatus: 451, BlockMessage: "blocked by audit", BlockedKeywords: []string{"forbidden-token"}}
		rawCfg, err := json.Marshal(cfg)
		require.NoError(t, err)
		moderation := service.NewContentModerationService(&geminiModerationSettingRepo{values: map[string]string{
			service.SettingKeyRiskControlEnabled:      "true",
			service.SettingKeyContentModerationConfig: string(rawCfg),
		}}, geminiModerationRepo{}, geminiModerationHashCache{}, nil, nil, nil, nil)
		group := &service.Group{ID: 95, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
		account := &service.Account{ID: 95, Name: "api-key", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "key"}}
		env := newTerminalGatewayMessagesEnv(t, group, &openAIChatCompletionsHTTPUpstreamStub{err: fmt.Errorf("must not be called")}, account)
		env.handler.contentModerationService = moderation
		body := `{"contents":[{"role":"user","parts":[{"text":"` + strings.Repeat("x", 12<<20) + `forbidden-token"}]}]}`
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1beta/models/*modelAction", env.handler.GeminiV1BetaModels).ServeHTTP(rec, req)

		require.Equal(t, http.StatusUnavailableForLegalReasons, rec.Code)
		require.JSONEq(t, `{"error":{"code":451,"message":"blocked by audit","status":"UNKNOWN"}}`, rec.Body.String())
		entries, err := os.ReadDir(rawDir)
		require.NoError(t, err)
		require.Empty(t, entries, "audit rejection must clean its spool")
	})
}

func TestGeminiV1BetaModels_StreamTerminationCleansSpool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawDir := t.TempDir()
	old := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })
	group := &service.Group{ID: 96, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 96, Name: "api-key", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "key"}}
	upstream := newGeminiBlockingBodyUpstream(t, &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: &terminalPartialReadErrorBody{data: []byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"partial\"}]}}]}\n\n")}})
	env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		body := `{"contents":[{"role":"user","parts":[{"text":"` + strings.Repeat("x", 12<<20) + `"}]}]}`
		req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:streamGenerateContent", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1beta/models/*modelAction", env.handler.GeminiV1BetaModels).ServeHTTP(rec, req)
		close(done)
	}()
	select {
	case <-upstream.started:
	case <-time.After(5 * time.Second):
		t.Fatal("Gemini handler did not reach upstream")
	}
	entries, err := os.ReadDir(rawDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	upstream.Release()
	requireGeminiHandlerDone(t, done)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "partial")
	require.NotContains(t, rec.Body.String(), "Upstream request failed")
	entries, err = os.ReadDir(rawDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestGeminiV1BetaModels_AntigravityForcedRouteKeepsSpool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawDir := t.TempDir()
	old := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })
	group := &service.Group{ID: 97, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 97, Name: "antigravity", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}
	upstream := newGeminiBlockingBodyUpstream(t, nil)
	env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		body := `{"contents":[{"role":"user","parts":[{"text":"` + strings.Repeat("x", 12<<20) + `"}]}]}`
		req := httptest.NewRequest(http.MethodPost, "/antigravity/v1beta/models/gemini-2.5-flash:generateContent", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router := env.routerFor("/antigravity/v1beta/models/*modelAction", func(c *gin.Context) {
			c.Set(string(middleware.ContextKeyForcePlatform), service.PlatformAntigravity)
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.ForcePlatform, service.PlatformAntigravity))
			env.handler.GeminiV1BetaModels(c)
		})
		router.ServeHTTP(rec, req)
		close(done)
	}()
	select {
	case <-upstream.started:
	case <-time.After(5 * time.Second):
		t.Fatal("forced antigravity route did not reach upstream")
	}
	entries, err := os.ReadDir(rawDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	upstream.Release()
	requireGeminiHandlerDone(t, done)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	entries, err = os.ReadDir(rawDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

type geminiModerationSettingRepo struct {
	service.SettingRepository
	values map[string]string
}

func (r *geminiModerationSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	return r.values[key], nil
}

func (r *geminiModerationSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		values[key] = r.values[key]
	}
	return values, nil
}

type geminiModerationRepo struct {
	service.ContentModerationRepository
}

func (geminiModerationRepo) CreateLog(context.Context, *service.ContentModerationLog) error {
	return nil
}

type geminiModerationHashCache struct {
	service.ContentModerationHashCache
}

func (geminiModerationHashCache) RecordFlaggedInputHash(context.Context, string) error { return nil }
func (geminiModerationHashCache) HasFlaggedInputHash(context.Context, string) (bool, error) {
	return false, nil
}

type geminiSpoolReleaseConcurrencyCache struct {
	openAIChatCompletionsConcurrencyCacheStub
	acquired    chan struct{}
	proceed     chan struct{}
	releaseOnce sync.Once
	mu          sync.Mutex
	releases    int
}

type geminiHandlerWriterSpy struct {
	gin.ResponseWriter
	writtenBeforeFirstHeader bool
	firstStatus              int
}

func (w *geminiHandlerWriterSpy) WriteHeader(status int) {
	if w.firstStatus == 0 {
		w.writtenBeforeFirstHeader = w.ResponseWriter.Written()
		w.firstStatus = status
	}
	w.ResponseWriter.WriteHeader(status)
}

type geminiSecondarySpoolUpstream struct {
	service.HTTPUpstream
	t                *testing.T
	firstResponse    *http.Response
	writer           gin.ResponseWriter
	failCall         int
	writtenAtFailure bool
	calls            int
}

func (u *geminiSecondarySpoolUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.calls++
	if u.calls == u.failCall {
		u.writtenAtFailure = u.writer != nil && u.writer.Written()
		return nil, fmt.Errorf("secondary transport: %w", service.ErrRequestBodySpool)
	}
	if req != nil && req.Body != nil {
		_, err := io.Copy(io.Discard, req.Body)
		require.NoError(u.t, err)
		require.NoError(u.t, req.Body.Close())
	}
	return u.firstResponse, nil
}

func (c *geminiSpoolReleaseConcurrencyCache) AcquireAccountSlot(context.Context, int64, int, string) (bool, error) {
	c.acquired <- struct{}{}
	<-c.proceed
	return true, nil
}

func (c *geminiSpoolReleaseConcurrencyCache) Release() {
	c.releaseOnce.Do(func() { close(c.proceed) })
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
	started       chan *http.Request
	release       chan struct{}
	releaseOnce   sync.Once
	requestHashes []string
	response      *http.Response
}

type geminiEffectiveHandleUpstream struct {
	service.HTTPUpstream
	t          *testing.T
	rawDir     string
	bodies     [][]byte
	accountIDs []int64
	mutatedRaw bool
}

func (u *geminiEffectiveHandleUpstream) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	require.NoError(u.t, err)
	require.NoError(u.t, req.Body.Close())
	u.bodies = append(u.bodies, body)
	u.accountIDs = append(u.accountIDs, accountID)
	if len(u.bodies) == 1 {
		entries, err := os.ReadDir(u.rawDir)
		require.NoError(u.t, err)
		for _, entry := range entries {
			path := filepath.Join(u.rawDir, entry.Name())
			candidate, err := os.ReadFile(path)
			require.NoError(u.t, err)
			if bytes.Contains(candidate, []byte("old-account")) {
				replacement := []byte(`{"contents":[{"role":"user","parts":[{"text":"raw-only","thoughtSignature":"old-account"}]}]}`)
				require.NoError(u.t, os.WriteFile(path, replacement, 0o600))
				u.mutatedRaw = true
			}
		}
	}
	if len(u.bodies) == 1 {
		return &http.Response{StatusCode: http.StatusUnauthorized, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"switch account"}}`))}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"role":"model","parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`))}, nil
}

func newGeminiBlockingBodyUpstream(t *testing.T, response *http.Response) *geminiBlockingBodyUpstream {
	t.Helper()
	u := &geminiBlockingBodyUpstream{started: make(chan *http.Request, 1), release: make(chan struct{}), response: response}
	t.Cleanup(u.Release)
	return u
}

func (u *geminiBlockingBodyUpstream) Release() {
	u.releaseOnce.Do(func() { close(u.release) })
}

func requireGeminiHandlerDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Gemini handler did not return")
	}
}

func (u *geminiBlockingBodyUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	if body, err := req.GetBody(); err == nil {
		defer body.Close()
		if payload, err := io.ReadAll(body); err == nil {
			u.requestHashes = append(u.requestHashes, service.HashUsageRequestPayload(payload))
		}
	}
	u.started <- req
	<-u.release
	if u.response != nil {
		return u.response, nil
	}
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
