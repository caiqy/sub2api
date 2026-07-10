package handler

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
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
