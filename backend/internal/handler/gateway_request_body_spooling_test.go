package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
