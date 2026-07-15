package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func TestGatewayHandler_MessagesCompressedRequestBodySpoolsUntilBlockedUpstreamCompletes(t *testing.T) {
	testGatewayRequestBodySpoolLifecycle(t, false, http.StatusBadRequest, http.StatusBadRequest)
}

func TestGatewayHandler_ResponsesCompressedRequestBodySpoolsEffectiveBodyUntilBlockedUpstreamCompletes(t *testing.T) {
	testGatewayRequestBodySpoolLifecycle(t, true, http.StatusBadRequest, http.StatusBadRequest)
}

func TestGatewayHandler_MessagesAndResponsesUpstream5xxPreserveErrorContract(t *testing.T) {
	t.Run("messages", func(t *testing.T) {
		testGatewayRequestBodySpoolLifecycle(t, false, http.StatusInternalServerError, http.StatusBadGateway)
	})
	t.Run("responses", func(t *testing.T) {
		testGatewayRequestBodySpoolLifecycle(t, true, http.StatusInternalServerError, http.StatusInternalServerError)
	})
}

func TestGatewayHandler_MessagesAndResponsesCanceledRequestsCleanSpools(t *testing.T) {
	for _, tt := range []struct {
		name  string
		route string
		body  string
	}{
		{"messages", "/v1/messages", `{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"` + strings.Repeat("x", 12<<20) + `"}]}`},
		{"responses", "/v1/responses", `{"model":"claude-opus-4-6","input":"` + strings.Repeat("x", 12<<20) + `"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			rawDir, effectiveDir := t.TempDir(), t.TempDir()
			oldOptions := jsonRequestBodyHandleOptions
			jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, TempDir: rawDir}
			t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
			t.Setenv("TMPDIR", effectiveDir)
			t.Setenv("TMP", effectiveDir)
			t.Setenv("TEMP", effectiveDir)

			ctx, cancel := context.WithCancel(context.Background())
			group := &service.Group{ID: 46, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
			account := &service.Account{ID: 146, Name: "anthropic-canceled", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "token"}}
			env := newTerminalGatewayMessagesEnv(t, group, cancelingGatewayRequestBodyUpstream{cancelingTerminalHTTPUpstream{cancel: cancel}}, account)
			req := httptest.NewRequest(http.MethodPost, tt.route, strings.NewReader(tt.body)).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			router := env.routerFor(tt.route, env.handler.Messages)
			if tt.route == "/v1/responses" {
				router = env.routerFor(tt.route, env.handler.Responses)
			}
			router.ServeHTTP(httptest.NewRecorder(), req)

			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Fatalf("context error = %v, want canceled", ctx.Err())
			}
			assertGatewaySpoolDirsEmpty(t, rawDir, effectiveDir)
		})
	}
}

func TestGatewayHandler_MessagesAndResponsesReplayLargeBodiesAcrossFailover(t *testing.T) {
	for _, tt := range []struct {
		name      string
		route     string
		gzip      bool
		responses bool
	}{
		{"messages-identity", "/v1/messages", false, false},
		{"messages-gzip", "/v1/messages", true, false},
		{"responses-identity", "/v1/responses", false, true},
		{"responses-gzip", "/v1/responses", true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			rawDir, effectiveDir := t.TempDir(), t.TempDir()
			oldOptions := jsonRequestBodyHandleOptions
			jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, PreviewLimitBytes: 64, TempDir: rawDir}
			t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
			t.Setenv("TMPDIR", effectiveDir)
			t.Setenv("TMP", effectiveDir)
			t.Setenv("TEMP", effectiveDir)

			body := []byte(`{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"` + strings.Repeat("x", 12<<20) + `"}]}`)
			expected := body
			if tt.responses {
				body = []byte(`{"model":"claude-opus-4-6","input":"` + strings.Repeat("x", 12<<20) + `","stream":true}`)
				expected = []byte(`{"model":"claude-opus-4-6","max_tokens":8192,"messages":[{"role":"user","content":"` + strings.Repeat("x", 12<<20) + `"}],"stream":true}`)
			}
			requestBody := body
			if tt.gzip {
				var compressed bytes.Buffer
				zipper := gzip.NewWriter(&compressed)
				if _, err := zipper.Write(body); err != nil {
					t.Fatalf("gzip body: %v", err)
				}
				if err := zipper.Close(); err != nil {
					t.Fatalf("close gzip: %v", err)
				}
				requestBody = compressed.Bytes()
			}

			upstream := &gatewayReplaySpoolUpstream{started: make(chan struct{}), release: make(chan struct{}), snapshots: make(chan gatewayReplaySnapshots, 1), responses: tt.responses}
			group := &service.Group{ID: 47, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
			accounts := []*service.Account{
				{ID: 147, Name: "first", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "first"}},
				{ID: 148, Name: "second", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{"api_key": "second"}},
			}
			env := newTerminalGatewayMessagesEnv(t, group, upstream, accounts...)
			var requestContext *gin.Context
			router := env.routerFor(tt.route, func(c *gin.Context) {
				requestContext = c
				if tt.responses {
					env.handler.Responses(c)
					return
				}
				env.handler.Messages(c)
			})
			upstream.snapshot = func() gatewayReplaySnapshots {
				detail := middleware.GetUsageDetailSnapshot(requestContext)
				ops, _ := requestContext.Get(service.OpsUpstreamRequestBodyKey)
				opsUpstream, ok := ops.(string)
				if !ok {
					t.Fatalf("ops upstream body type = %T, want string", ops)
				}
				return gatewayReplaySnapshots{usageRequest: detail.RequestBody, usageUpstream: detail.UpstreamRequestBody, opsUpstream: opsUpstream}
			}
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tt.route, bytes.NewReader(requestBody))
			req.Header.Set("Content-Type", "application/json")
			if tt.gzip {
				req.Header.Set("Content-Encoding", "gzip")
			}
			done := make(chan struct{})
			go func() { router.ServeHTTP(recorder, req); close(done) }()
			waitGatewayReplaySignal(t, upstream.started, "second upstream attempt")
			snapshot := waitGatewayReplaySnapshot(t, upstream.snapshots)
			assertBoundedBodySnapshot(t, "usage request", snapshot.usageRequest, int64(len(body)))
			assertBoundedBodySnapshot(t, "usage upstream", snapshot.usageUpstream, int64(len(expected)))
			assertBoundedBodySnapshot(t, "ops upstream", snapshot.opsUpstream, int64(len(expected)))
			assertGatewaySpoolDirsPresent(t, rawDir, effectiveDir)
			close(upstream.release)
			waitGatewayReplaySignal(t, done, "handler completion")
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			if tt.responses && !strings.Contains(recorder.Body.String(), "response.completed") {
				t.Fatalf("Responses stream did not emit response.completed: %s", recorder.Body.String())
			}
			upstream.assert(t, expected, accounts[0].ID, accounts[1].ID)
			assertGatewaySpoolDirsEmpty(t, rawDir, effectiveDir)
		})
	}
}

func TestGatewayHandler_RequestBodySpoolOpenFailureMapsTo503(t *testing.T) {
	old := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })

	coordinator, err := newJSONRequestBody(httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader([]byte(`{"model":"claude-test"}`))))
	if err != nil {
		t.Fatalf("newJSONRequestBody: %v", err)
	}
	defer coordinator.Cleanup()
	entries, err := os.ReadDir(jsonRequestBodyHandleOptions.TempDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("find spool: entries=%d err=%v", len(entries), err)
	}
	if err := os.Remove(filepath.Join(jsonRequestBodyHandleOptions.TempDir, entries[0].Name())); err != nil {
		t.Fatalf("remove spool: %v", err)
	}
	_, err = coordinator.ReadRaw()
	if !errors.Is(err, service.ErrRequestBodySpool) {
		t.Fatalf("ReadRaw error = %v, want ErrRequestBodySpool", err)
	}
	if status, ok := requestBodyReadErrorStatus(err); !ok || status != http.StatusServiceUnavailable {
		t.Fatalf("status = (%d, %t), want (503, true)", status, ok)
	}
}

func TestGatewayHandler_ResponsesForwardBodyErrorUsesRequestBodyStatus(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	(&GatewayHandler{}).writeResponsesForwardRequestBodyError(c, fmt.Errorf("forward responses: %w", service.ErrRequestBodySpool))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
}

func TestGatewayHandler_ResponsesSpoolTransportFailureReturns503WithoutUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 48, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 149, Name: "anthropic-spool", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "token"}}
	upstream := &responsesSpoolTransportUpstream{}
	env := newTerminalGatewayMessagesEnv(t, group, upstream, account)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"claude-opus-4-6","input":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	env.routerFor("/v1/responses", env.handler.Responses).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body = %s", recorder.Code, recorder.Body.String())
	}
	if got := gjson.Get(recorder.Body.String(), "error.code").String(); got != "server_error" {
		t.Fatalf("error.code = %q, want server_error", got)
	}
	if upstream.calls != 1 {
		t.Fatalf("upstream calls = %d, want 1", upstream.calls)
	}
	if env.usageRepo.lastLog != nil {
		t.Fatalf("spool transport failure recorded usage: %#v", env.usageRepo.lastLog)
	}
}

func TestGatewayHandler_ResponsesForwardBodyErrorDoesNotAppendAfterResponseCommit(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Writer.WriteHeader(http.StatusOK)
	if _, err := c.Writer.WriteString("data: existing\n\n"); err != nil {
		t.Fatalf("write committed response: %v", err)
	}

	if !(&GatewayHandler{}).writeResponsesForwardRequestBodyError(c, fmt.Errorf("forward responses: %w", service.ErrRequestBodySpool)) {
		t.Fatal("spool error was not handled")
	}
	if got, want := recorder.Body.String(), "data: existing\n\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestGatewayHandler_ResponsesLargeBodyWaitsWithReplayHandle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawDir, effectiveDir := t.TempDir(), t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, PreviewLimitBytes: 64, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
	t.Setenv("TMPDIR", effectiveDir)
	t.Setenv("TMP", effectiveDir)
	t.Setenv("TEMP", effectiveDir)

	cache := &blockingResponsesUserSlotCache{waiting: make(chan struct{}), release: make(chan struct{})}
	group := &service.Group{ID: 49, Platform: service.PlatformAnthropic, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 150, Name: "anthropic-wait", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "token"}}
	env := newTerminalGatewayMessagesEnvWithConcurrencyCache(t, group, &openAIStatusSpoolUpstream{status: http.StatusBadRequest}, cache, account)

	done := make(chan struct{})
	go func() {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"claude-opus-4-6","input":"`+strings.Repeat("x", 12<<20)+`"}`))
		req.Header.Set("Content-Type", "application/json")
		env.routerFor("/v1/responses", env.handler.Responses).ServeHTTP(recorder, req)
		close(done)
	}()

	waitGatewayReplaySignal(t, cache.waiting, "user slot wait")
	entries, err := os.ReadDir(rawDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("raw spool missing during user wait: entries=%d err=%v", len(entries), err)
	}
	entries, err = os.ReadDir(effectiveDir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("derived spool exists before forwarding: entries=%d err=%v", len(entries), err)
	}
	close(cache.release)
	waitGatewayReplaySignal(t, done, "handler completion")
	assertGatewaySpoolDirsEmpty(t, rawDir, effectiveDir)
}

func TestGatewayHandler_MessagesContextKeepsHandleInsteadOfAttemptBytes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 44, Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 144, Name: "antigravity", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}
	upstream := &openAIChatCompletionsHTTPUpstreamStub{response: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_context_body"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"Prompt is too long"}}`)),
	}}
	env := newTerminalGatewayMessagesEnv(t, group, upstream, account)

	var values map[string]any
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), env.apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: env.apiKey.UserID, Concurrency: env.apiKey.User.Concurrency})
		c.Next()
		values = make(map[string]any, len(c.Keys))
		for key, value := range c.Keys {
			values[key] = value
		}
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/messages", env.handler.Messages)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	for key, value := range values {
		if body, ok := value.([]byte); ok && len(body) > 0 {
			t.Fatalf("Gin context retained body bytes at %q", key)
		}
	}
	parsed, ok := values["parsed_request"].(*service.ParsedRequest)
	if !ok || parsed.Body.Handle() == nil {
		t.Fatal("Gin context did not retain a handle-backed parsed request")
	}
}

func TestGatewayHandler_MessagesCleansDerivedAttemptHandleAfterForwardPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	old := jsonRequestBodyHandleOptions
	rawDir := t.TempDir()
	effectiveDir := t.TempDir()
	t.Setenv("TMPDIR", effectiveDir)
	t.Setenv("TMP", effectiveDir)
	t.Setenv("TEMP", effectiveDir)
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })

	group := &service.Group{ID: 45, Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 145, Name: "antigravity", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}
	env := newTerminalGatewayMessagesEnv(t, group, panicGatewayRequestBodyUpstream{}, account)

	func() {
		defer func() { _ = recover() }()
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"`+strings.Repeat("x", 10<<20)+`"}]}`))
		env.router().ServeHTTP(recorder, req)
	}()
	for _, dir := range []string{rawDir, effectiveDir} {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("spool remains after Forward panic in %s: entries=%d err=%v", dir, len(entries), err)
		}
	}
}

type panicGatewayRequestBodyUpstream struct{ service.HTTPUpstream }

type responsesSpoolTransportUpstream struct {
	service.HTTPUpstream
	calls int
}

type blockingResponsesUserSlotCache struct {
	openAIChatCompletionsConcurrencyCacheStub
	waiting chan struct{}
	release chan struct{}
	calls   int
}

func (c *blockingResponsesUserSlotCache) AcquireUserSlot(ctx context.Context, _ int64, _ int, _ string) (bool, error) {
	c.calls++
	if c.calls == 1 {
		return false, nil
	}
	close(c.waiting)
	select {
	case <-c.release:
		return true, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func (u *responsesSpoolTransportUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	u.calls++
	return nil, fmt.Errorf("read request body: %w", service.ErrRequestBodySpool)
}

type cancelingGatewayRequestBodyUpstream struct{ cancelingTerminalHTTPUpstream }

func (u cancelingGatewayRequestBodyUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

func (panicGatewayRequestBodyUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	// The upstream owns the active request reader; this test isolates handler-owned attempt cleanup.
	_ = req.Body.Close()
	panic("forward panic")
}

func testGatewayRequestBodySpoolLifecycle(t *testing.T, mapModel bool, upstreamStatus, wantStatus int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	old := jsonRequestBodyHandleOptions
	rawDir := t.TempDir()
	effectiveDir := t.TempDir()
	t.Setenv("TMPDIR", effectiveDir)
	t.Setenv("TMP", effectiveDir)
	t.Setenv("TEMP", effectiveDir)
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, PreviewLimitBytes: 64, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })

	body := []byte(`{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"` + strings.Repeat("x", 12<<20) + `"}]}`)
	if mapModel {
		body = []byte(`{"model":"claude-opus-4-6","input":"` + strings.Repeat("x", 12<<20) + `"}`)
	}
	var compressed bytes.Buffer
	zipper := gzip.NewWriter(&compressed)
	if _, err := zipper.Write(body); err != nil {
		t.Fatalf("gzip body: %v", err)
	}
	if err := zipper.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	upstream := &blockingGatewayRequestBodyUpstream{started: make(chan []byte, 1), release: make(chan struct{}), status: upstreamStatus}
	group := &service.Group{ID: 44, Platform: service.PlatformAntigravity, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 144, Name: "antigravity", Platform: service.PlatformAntigravity, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}
	if mapModel {
		group.Platform = service.PlatformAnthropic
		account.Platform = service.PlatformAnthropic
		account.Type = service.AccountTypeAPIKey
		account.Credentials = map[string]any{"api_key": "token"}
	}
	env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
	router := env.routerFor("/v1/responses", env.handler.Responses)
	if !mapModel {
		router = env.router()
	}

	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		path := "/v1/messages"
		if mapModel {
			path = "/v1/responses"
		}
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(compressed.Bytes()))
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)
		close(done)
	}()
	effective := <-upstream.started
	entries, err := os.ReadDir(rawDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("spool missing while upstream blocks: entries=%d err=%v", len(entries), err)
	}
	effectiveEntries, err := os.ReadDir(effectiveDir)
	if err != nil || len(effectiveEntries) == 0 {
		t.Fatalf("effective spool missing while upstream blocks: entries=%d err=%v", len(effectiveEntries), err)
	}
	if mapModel {
		expected := []byte(`{"model":"claude-opus-4-6","max_tokens":8192,"messages":[{"role":"user","content":"` + strings.Repeat("x", 12<<20) + `"}],"stream":true}`)
		if !bytes.Equal(effective, expected) {
			t.Fatalf("Responses upstream body differs: got prefix=%q suffix=%q, want prefix=%q suffix=%q", effective[:smallestInt(256, len(effective))], effective[largestInt(0, len(effective)-256):], expected[:smallestInt(256, len(expected))], expected[largestInt(0, len(expected)-256):])
		}
		hash := sha256.Sum256(effective)
		if got := hex.EncodeToString(hash[:]); got != "4094f747ce09b3fb6fddf060651d3ce9c5484a5dfe1df7c124d7942bd860c7cc" {
			t.Fatalf("Responses upstream SHA-256 = %s", got)
		}
	} else {
		normalized, ok := normalizeAntigravityRequestID(effective)
		if !ok {
			t.Fatalf("Messages Gemini body did not contain an agent requestId: %q", effective[:smallestInt(256, len(effective))])
		}
		hash := sha256.Sum256(normalized)
		if got := hex.EncodeToString(hash[:]); got != "35a43779927f8d133eae6efa2b31aa2e500612aa79145dad42fd5a1cb4989e4f" {
			t.Fatalf("Messages normalized Gemini SHA-256 = %s", got)
		}
		if got := gjson.GetBytes(effective, "model").String(); got != "claude-opus-4-6-thinking" {
			t.Fatalf("Messages Gemini model = %q", got)
		}
	}
	close(upstream.release)
	<-done
	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d", recorder.Code, wantStatus)
	}
	if env.usageRepo.lastLog == nil || env.usageRepo.lastLog.DetailSnapshot == nil {
		t.Fatal("blocked request did not submit its usage detail")
	}
	assertBoundedBodySnapshot(t, "request", env.usageRepo.lastLog.DetailSnapshot.RequestBody, int64(len(body)))
	assertBoundedBodySnapshot(t, "upstream", env.usageRepo.lastLog.DetailSnapshot.UpstreamRequestBody, int64(len(effective)))
	entries, err = os.ReadDir(rawDir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("spool remains after request: entries=%d err=%v", len(entries), err)
	}
	effectiveEntries, err = os.ReadDir(effectiveDir)
	if err != nil || len(effectiveEntries) != 0 {
		t.Fatalf("effective spool remains after request: entries=%d err=%v", len(effectiveEntries), err)
	}
}

func assertBoundedBodySnapshot(t *testing.T, name, snapshot string, wantSize int64) {
	t.Helper()
	if !gjson.Valid(snapshot) {
		t.Fatalf("%s snapshot is not JSON: %q", name, snapshot)
	}
	if got := gjson.Get(snapshot, "kind").String(); got != "request_body_preview" {
		t.Fatalf("%s snapshot kind = %q", name, got)
	}
	if got := gjson.Get(snapshot, "size").Int(); got != wantSize {
		t.Fatalf("%s snapshot size = %d, want %d", name, got, wantSize)
	}
	if !gjson.Get(snapshot, "truncated").Bool() {
		t.Fatalf("%s snapshot was not truncated", name)
	}
	if len(snapshot) >= 10<<20 {
		t.Fatalf("%s snapshot retained the complete large body: %d bytes", name, len(snapshot))
	}
}

func smallestInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func largestInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func normalizeAntigravityRequestID(body []byte) ([]byte, bool) {
	const prefix = `"requestId":"agent-`
	start := bytes.Index(body, []byte(prefix))
	if start < 0 {
		return nil, false
	}
	start += len(prefix)
	end := bytes.IndexByte(body[start:], '"')
	if end != 36 {
		return nil, false
	}
	normalized := append([]byte(nil), body...)
	copy(normalized[start:start+end], "00000000-0000-0000-0000-000000000000")
	return normalized, true
}

type blockingGatewayRequestBodyUpstream struct {
	service.HTTPUpstream
	started chan []byte
	release chan struct{}
	status  int
	once    sync.Once
}

func (u *blockingGatewayRequestBodyUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	defer func() { _ = req.Body.Close() }()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	u.once.Do(func() {
		u.started <- body
		<-u.release
	})
	return &http.Response{StatusCode: u.status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Prompt is too long"}}`))}, nil
}

func (u *blockingGatewayRequestBodyUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

type gatewayReplaySnapshots struct {
	usageRequest  string
	usageUpstream string
	opsUpstream   string
}

type gatewayReplaySpoolUpstream struct {
	service.HTTPUpstream
	mu        sync.Mutex
	bodies    [][]byte
	hashes    []string
	lengths   []int64
	accounts  []int64
	started   chan struct{}
	release   chan struct{}
	snapshot  func() gatewayReplaySnapshots
	snapshots chan gatewayReplaySnapshots
	responses bool
}

func (u *gatewayReplaySpoolUpstream) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	if err := req.Body.Close(); err != nil {
		return nil, err
	}
	getBody, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	reopened, err := io.ReadAll(getBody)
	if closeErr := getBody.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(body, reopened) {
		return nil, io.ErrUnexpectedEOF
	}
	sum := sha256.Sum256(body)
	u.mu.Lock()
	u.bodies = append(u.bodies, body)
	u.hashes = append(u.hashes, hex.EncodeToString(sum[:]))
	u.lengths = append(u.lengths, req.ContentLength)
	u.accounts = append(u.accounts, accountID)
	attempt := len(u.bodies)
	u.mu.Unlock()
	if attempt == 1 {
		return &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"retry"}}`))}, nil
	}
	if u.snapshot != nil {
		select {
		case u.snapshots <- u.snapshot():
		case <-time.After(5 * time.Second):
			return nil, errors.New("timed out publishing gateway replay snapshots")
		}
	}
	close(u.started)
	select {
	case <-u.release:
	case <-time.After(5 * time.Second):
		return nil, errors.New("timed out waiting for gateway replay test release")
	}
	if u.responses {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(gatewayReplayResponsesSSE))}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"msg_123","type":"message","role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))}, nil
}

func (u *gatewayReplaySpoolUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

func (u *gatewayReplaySpoolUpstream) assert(t *testing.T, want []byte, firstAccount, secondAccount int64) {
	t.Helper()
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.bodies) != 2 {
		t.Fatalf("upstream attempts = %d, want 2", len(u.bodies))
	}
	if u.accounts[0] != firstAccount || u.accounts[1] != secondAccount {
		t.Fatalf("upstream accounts = %v, want [%d %d]", u.accounts, firstAccount, secondAccount)
	}
	wantHash := sha256.Sum256(want)
	wantHashString := hex.EncodeToString(wantHash[:])
	for i, body := range u.bodies {
		if !bytes.Equal(body, want) {
			t.Fatalf("attempt %d body differs: got hash=%s want hash=%s", i+1, u.hashes[i], wantHashString)
		}
		if u.lengths[i] != int64(len(want)) {
			t.Fatalf("attempt %d content length = %d, want %d", i+1, u.lengths[i], len(want))
		}
		if u.hashes[i] != wantHashString {
			t.Fatalf("attempt %d body hash = %s, want %s", i+1, u.hashes[i], wantHashString)
		}
	}
}

const gatewayReplayResponsesSSE = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_123\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-opus-4-6\",\"stop_reason\":null,\"stop_sequence\":null,\"usage\":{\"input_tokens\":1,\"output_tokens\":0}}}\n\nevent: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":1}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

func waitGatewayReplaySignal(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func waitGatewayReplaySnapshot(t *testing.T, snapshots <-chan gatewayReplaySnapshots) gatewayReplaySnapshots {
	t.Helper()
	select {
	case snapshot := <-snapshots:
		return snapshot
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for gateway replay snapshots")
		return gatewayReplaySnapshots{}
	}
}

func assertGatewaySpoolDirsPresent(t *testing.T, dirs ...string) {
	t.Helper()
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) == 0 {
			t.Fatalf("spool missing while upstream blocks in %s: entries=%d err=%v", dir, len(entries), err)
		}
	}
}

func assertGatewaySpoolDirsEmpty(t *testing.T, dirs ...string) {
	t.Helper()
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("spool remains in %s: entries=%d err=%v", dir, len(entries), err)
		}
	}
}
