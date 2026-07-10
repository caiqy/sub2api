package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestRequestBodyCoordinator_JSON(t *testing.T) {
	const threshold = 10 << 20
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions.SpoolThresholdBytes = threshold
	jsonRequestBodyHandleOptions.PreviewLimitBytes = 5 << 20
	jsonRequestBodyHandleOptions.TempDir = t.TempDir()
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	for _, encoding := range []string{"identity", "gzip"} {
		for _, size := range []int{threshold - 1, threshold, threshold + 1} {
			t.Run(encoding+"/"+strconv.Itoa(size), func(t *testing.T) {
				body := requestBodyCoordinatorJSON(size)
				req := httptest.NewRequest(http.MethodPost, "/", requestBodyCoordinatorEncodedBody(t, body, encoding))
				if encoding != "identity" {
					req.Header.Set("Content-Encoding", encoding)
				}

				coordinator, err := newJSONRequestBody(req)
				if err != nil {
					t.Fatalf("newJSONRequestBody: %v", err)
				}
				t.Cleanup(coordinator.Cleanup)

				raw := coordinator.raw
				if raw.Size() != int64(size) {
					t.Fatalf("raw size = %d, want %d", raw.Size(), size)
				}
				hash := sha256.Sum256(body)
				if raw.Hash() != hex.EncodeToString(hash[:]) {
					t.Fatalf("raw hash = %q, want %q", raw.Hash(), hex.EncodeToString(hash[:]))
				}
				if coordinator.Effective() != raw {
					t.Fatal("effective handle does not reuse raw handle")
				}
				if req.Header.Get("Content-Encoding") != "" || req.ContentLength != int64(size) {
					t.Fatalf("request metadata = (%q, %d), want empty encoding and length %d", req.Header.Get("Content-Encoding"), req.ContentLength, size)
				}
				if size > 5<<20 && !strings.Contains(raw.PreviewString(), "omitted") {
					t.Fatal("preview was not truncated at 5MB")
				}

				entries, err := os.ReadDir(jsonRequestBodyHandleOptions.TempDir)
				if err != nil {
					t.Fatalf("read temp dir: %v", err)
				}
				if got, want := len(entries) > 0, size > threshold; got != want {
					t.Fatalf("spooled = %t, want %t", got, want)
				}

				for i := 0; i < 2; i++ {
					r, err := raw.Open()
					if err != nil {
						t.Fatalf("open %d: %v", i, err)
					}
					got, err := io.ReadAll(r)
					_ = r.Close()
					if err != nil || !bytes.Equal(got, body) {
						t.Fatalf("open %d returned unexpected body: %v", i, err)
					}
				}
			})
		}
	}
}

func TestRequestBodyCoordinator_Spool(t *testing.T) {
	const threshold = 10 << 20
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions.SpoolThresholdBytes = threshold
	jsonRequestBodyHandleOptions.TempDir = filepath.Join(t.TempDir(), "missing")
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(requestBodyCoordinatorJSON(threshold+1)))
	_, err := newJSONRequestBody(req)
	if !errors.Is(err, service.ErrRequestBodySpool) {
		t.Fatalf("newJSONRequestBody error = %v, want ErrRequestBodySpool", err)
	}

	spoolRequest := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(requestBodyCoordinatorJSON(threshold+1)))
	spoolRecorder := httptest.NewRecorder()
	requestBodyCoordinatorErrorHandler(spoolRecorder, spoolRequest)
	if spoolRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("spool response status = %d, want %d", spoolRecorder.Code, http.StatusServiceUnavailable)
	}

	jsonRequestBodyHandleOptions.TempDir = t.TempDir()
	limitRequest := httptest.NewRequest(http.MethodPost, "/", requestBodyCoordinatorGzip(t, requestBodyCoordinatorJSON((64<<20)+1)))
	limitRequest.Header.Set("Content-Encoding", "gzip")
	limitRecorder := httptest.NewRecorder()
	requestBodyCoordinatorErrorHandler(limitRecorder, limitRequest)
	if limitRecorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("limit response status = %d, want %d", limitRecorder.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestRequestBodyCoordinator_JSONEffective(t *testing.T) {
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions.TempDir = t.TempDir()
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	body := []byte(`{"data":"raw"}`)
	coordinator, err := newJSONRequestBody(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body)))
	if err != nil {
		t.Fatalf("newJSONRequestBody: %v", err)
	}
	t.Cleanup(coordinator.Cleanup)

	if err := coordinator.SetEffectiveBytes(append([]byte(nil), body...)); err != nil {
		t.Fatalf("SetEffectiveBytes: %v", err)
	}
	if coordinator.Effective() != coordinator.raw {
		t.Fatal("matching effective bytes did not reuse raw")
	}
	if err := coordinator.SetEffectiveReader(strings.NewReader(`{"data":"new"}`)); err != nil {
		t.Fatalf("SetEffectiveReader: %v", err)
	}
	if coordinator.Effective() == coordinator.raw {
		t.Fatal("different effective reader reused raw")
	}
	if err := coordinator.SetEffectiveReader(bytes.NewReader(body)); err != nil {
		t.Fatalf("SetEffectiveReader restore: %v", err)
	}
	if coordinator.Effective() != coordinator.raw {
		t.Fatal("matching effective reader did not reuse raw")
	}
}

func TestRequestBodyCoordinator_Cleanup(t *testing.T) {
	t.Run("raw and effective are cleaned once", func(t *testing.T) {
		coordinator, dir := newSpoolingRequestBodyCoordinator(t)
		coordinator.Cleanup()
		coordinator.Cleanup()
		requestBodyCoordinatorRequireEmptyDir(t, dir)
	})

	t.Run("replaced effective is cleaned", func(t *testing.T) {
		coordinator, dir := newSpoolingRequestBodyCoordinator(t)
		if err := coordinator.SetEffectiveBytes([]byte(strings.Repeat("effective", 64))); err != nil {
			t.Fatalf("SetEffectiveBytes: %v", err)
		}
		coordinator.Cleanup()
		requestBodyCoordinatorRequireEmptyDir(t, dir)
	})

	t.Run("each replaced effective is cleaned", func(t *testing.T) {
		coordinator, dir := newSpoolingRequestBodyCoordinator(t)
		for _, body := range [][]byte{[]byte(strings.Repeat("first", 64)), []byte(strings.Repeat("second", 64))} {
			if err := coordinator.SetEffectiveBytes(body); err != nil {
				t.Fatalf("SetEffectiveBytes: %v", err)
			}
		}
		coordinator.Cleanup()
		requestBodyCoordinatorRequireEmptyDir(t, dir)
	})

	t.Run("multipart form files are removed", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "body.txt")
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		if _, err := part.Write([]byte(strings.Repeat("file", 64))); err != nil {
			t.Fatalf("write form file: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		if err := req.ParseMultipartForm(1); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		coordinator := &requestBodyCoordinator{form: req.MultipartForm}
		coordinator.Cleanup()
		coordinator.Cleanup()
		if _, err := req.MultipartForm.File["file"][0].Open(); err == nil {
			t.Fatal("multipart temporary file remains after cleanup")
		}
	})
}

func TestRequestBodyCoordinator_CleanupGinTerminationPaths(t *testing.T) {
	for _, tt := range []struct {
		name       string
		status     int
		cancel     bool
		panicValue any
	}{
		{name: "success", status: http.StatusNoContent},
		{name: "business rejection", status: http.StatusBadRequest},
		{name: "cancellation", status: http.StatusRequestTimeout, cancel: true},
		{name: "panic recovery", status: http.StatusInternalServerError, panicValue: "boom"},
		{name: "upstream 4xx", status: http.StatusBadGateway},
		{name: "upstream 5xx", status: http.StatusServiceUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			oldOptions := jsonRequestBodyHandleOptions
			jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, PreviewLimitBytes: 64, TempDir: dir, FilePrefix: "sub2api-test-"}
			t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

			router := gin.New()
			router.Use(gin.Recovery())
			router.POST("/", func(c *gin.Context) {
				coordinator, err := newJSONRequestBody(c.Request)
				if err != nil {
					t.Fatalf("newJSONRequestBody: %v", err)
				}
				defer coordinator.Cleanup()
				if tt.cancel {
					<-c.Request.Context().Done()
				}
				if tt.panicValue != nil {
					panic(tt.panicValue)
				}
				c.Status(tt.status)
			})

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(strings.Repeat("request", 64))))
			if tt.cancel {
				ctx, cancel := context.WithCancel(req.Context())
				cancel()
				req = req.WithContext(ctx)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != tt.status {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.status)
			}
			requestBodyCoordinatorRequireEmptyDir(t, dir)
		})
	}
}

func newSpoolingRequestBodyCoordinator(t *testing.T) (*requestBodyCoordinator, string) {
	t.Helper()
	dir := t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, PreviewLimitBytes: 64, TempDir: dir, FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
	coordinator, err := newJSONRequestBody(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(strings.Repeat("raw", 64)))))
	if err != nil {
		t.Fatalf("newJSONRequestBody: %v", err)
	}
	return coordinator, dir
}

func requestBodyCoordinatorRequireEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("spool directory contains %d entries after cleanup", len(entries))
	}
}

func requestBodyCoordinatorErrorHandler(w http.ResponseWriter, req *http.Request) {
	coordinator, err := newJSONRequestBody(req)
	if err == nil {
		coordinator.Cleanup()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if status, ok := requestBodyReadErrorStatus(err); ok {
		w.WriteHeader(status)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
}

func requestBodyCoordinatorJSON(size int) []byte {
	const prefix = `{"data":"`
	const suffix = `"}`
	return []byte(prefix + strings.Repeat("a", size-len(prefix)-len(suffix)) + suffix)
}

func requestBodyCoordinatorEncodedBody(t *testing.T, body []byte, encoding string) io.Reader {
	t.Helper()
	if encoding == "gzip" {
		return requestBodyCoordinatorGzip(t, body)
	}
	return bytes.NewReader(body)
}

func requestBodyCoordinatorGzip(t *testing.T, body []byte) io.Reader {
	t.Helper()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(body); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return bytes.NewReader(compressed.Bytes())
}
