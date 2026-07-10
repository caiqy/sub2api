package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestGatewayHandler_MessagesRequestBodySpoolsUntilBlockedUpstreamCompletes(t *testing.T) {
	testGatewayRequestBodySpoolLifecycle(t, false)
}

func TestGatewayHandler_ResponsesRequestBodySpoolsEffectiveBodyUntilBlockedUpstreamCompletes(t *testing.T) {
	testGatewayRequestBodySpoolLifecycle(t, true)
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

func testGatewayRequestBodySpoolLifecycle(t *testing.T, mapModel bool) {
	t.Helper()
	old := jsonRequestBodyHandleOptions
	dir := t.TempDir()
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, PreviewLimitBytes: 64, TempDir: dir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })

	body := []byte(`{"model":"claude-original","messages":[{"role":"user","content":"` + strings.Repeat("x", 10<<20) + `"}]}`)
	upstreamStarted := make(chan *service.RequestBodyHandle, 1)
	upstreamRelease := make(chan struct{})
	router := gin.New()
	router.POST("/v1/messages", func(c *gin.Context) {
		coordinator, err := newJSONRequestBody(c.Request)
		if err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		defer coordinator.Cleanup()
		raw, err := coordinator.ReadRaw()
		if err != nil {
			c.Status(requestBodyStatus(err))
			return
		}
		if mapModel {
			raw = bytes.Replace(raw, []byte("claude-original"), []byte("claude-effective"), 1)
		}
		if err := coordinator.SetEffectiveBytes(raw); err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		upstreamStarted <- coordinator.Effective()
		<-upstreamRelease
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body)))
		close(done)
	}()
	handle := <-upstreamStarted
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("spool missing while upstream blocks: entries=%d err=%v", len(entries), err)
	}
	effective, err := handle.ReadAll()
	if err != nil {
		t.Fatalf("read effective: %v", err)
	}
	if mapModel && !bytes.Contains(effective, []byte("claude-effective")) {
		t.Fatal("responses effective body did not include mapped model")
	}
	hash := sha256.Sum256(effective)
	if handle.Hash() != hex.EncodeToString(hash[:]) {
		t.Fatal("effective hash does not match effective body")
	}
	close(upstreamRelease)
	<-done
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	entries, err = os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("spool remains after request: entries=%d err=%v", len(entries), err)
	}
}

func requestBodyStatus(err error) int {
	if status, ok := requestBodyReadErrorStatus(err); ok {
		return status
	}
	return http.StatusBadRequest
}
