package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime"
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

func TestRequestBodyCoordinator_Multipart(t *testing.T) {
	const maxUpload = 20 << 20
	multipartDir := t.TempDir()
	t.Setenv("TMP", multipartDir)
	t.Setenv("TEMP", multipartDir)
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir(), FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	t.Run("uses form owned files", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.WriteField("model", "gpt-image-2"); err != nil {
			t.Fatal(err)
		}
		part, err := writer.CreateFormFile("image", "source.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte("image bytes")); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		coordinator, err := newMultipartRequestBody(req, 0)
		if err != nil {
			t.Fatalf("newMultipartRequestBody: %v", err)
		}
		t.Cleanup(coordinator.Cleanup)
		if coordinator.form == nil || len(coordinator.form.File["image"]) != 1 {
			t.Fatal("multipart file was not retained by the coordinator form")
		}
		if coordinator.Effective() != coordinator.raw {
			t.Fatal("multipart effective handle does not reuse raw handle")
		}
	})

	t.Run("rejects a part over 20MB", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("image", "oversize.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(bytes.Repeat([]byte("x"), maxUpload+1)); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		_, err = newMultipartRequestBody(req, 0)
		var maxErr *http.MaxBytesError
		if !errors.As(err, &maxErr) || maxErr.Limit != maxUpload {
			t.Fatalf("newMultipartRequestBody error = %v, want 20MB MaxBytesError", err)
		}
	})
}

func TestRequestBodyCoordinator_ReleaseMultipartValuesKeepsFiles(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("prompt", "release this text"); err != nil {
		t.Fatal(err)
	}
	file, err := writer.CreateFormFile("image", "source.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("image bytes")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	coordinator, err := newMultipartRequestBody(req, 0)
	if err != nil {
		t.Fatalf("newMultipartRequestBody: %v", err)
	}
	t.Cleanup(coordinator.Cleanup)

	coordinator.ReleaseMultipartValues()
	if len(coordinator.form.Value) != 0 {
		t.Fatalf("multipart values retained: %#v", coordinator.form.Value)
	}
	if len(coordinator.form.File["image"]) != 1 {
		t.Fatal("multipart files were released with values")
	}
}

func TestRequestBodyCoordinator_MultipartPipe(t *testing.T) {
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir(), FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", "gpt-image-2"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	coordinator, err := newMultipartRequestBody(req, 0)
	if err != nil {
		t.Fatalf("newMultipartRequestBody: %v", err)
	}
	t.Cleanup(coordinator.Cleanup)

	producerErr := errors.New("multipart producer failed")
	_, err = coordinator.SetEffectiveMultipart(func(writer *multipart.Writer) error {
		if err := writer.WriteField("model", "mapped-model"); err != nil {
			return err
		}
		return producerErr
	})
	if !errors.Is(err, producerErr) {
		t.Fatalf("pipe consumer error = %v, want producer error", err)
	}
	if coordinator.Effective() != coordinator.raw {
		t.Fatal("failed producer replaced the effective handle")
	}

	contentType, err := coordinator.SetEffectiveMultipart(func(writer *multipart.Writer) error {
		if err := writer.WriteField("model", "mapped-model"); err != nil {
			return err
		}
		part, err := writer.CreateFormFile("image", "source.png")
		if err != nil {
			return err
		}
		_, err = part.Write([]byte("image bytes"))
		return err
	})
	if err != nil {
		t.Fatalf("SetEffectiveMultipart: %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/form-data" || params["boundary"] == "" {
		t.Fatalf("content type = %q, err = %v", contentType, err)
	}
	effective := coordinator.Effective()
	if effective == coordinator.raw || effective.Size() == 0 {
		t.Fatal("successful producer did not create an effective multipart handle")
	}
	for retry := 0; retry < 2; retry++ {
		reader, err := effective.Open()
		if err != nil {
			t.Fatalf("effective open %d: %v", retry, err)
		}
		payload, readErr := io.ReadAll(reader)
		_ = reader.Close()
		if readErr != nil {
			t.Fatalf("effective read %d: %v", retry, readErr)
		}
		if int64(len(payload)) != effective.Size() {
			t.Fatalf("effective content length = %d, want %d", len(payload), effective.Size())
		}
		form, err := multipart.NewReader(bytes.NewReader(payload), params["boundary"]).ReadForm(0)
		if err != nil {
			t.Fatalf("effective multipart %d: %v", retry, err)
		}
		if form.Value["model"][0] != "mapped-model" || len(form.File["image"]) != 1 {
			t.Fatalf("effective multipart %d = %#v", retry, form)
		}
		_ = form.RemoveAll()
	}
}

func TestRequestBodyCoordinator_MultipartPipeRecoversProducerPanic(t *testing.T) {
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir(), FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	coordinator, err := newJSONRequestBody(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"raw":true}`)))
	if err != nil {
		t.Fatalf("newJSONRequestBody: %v", err)
	}
	t.Cleanup(coordinator.Cleanup)

	producerErr := errors.New("producer exploded")
	_, err = coordinator.SetEffectiveMultipart(func(*multipart.Writer) error {
		panic(producerErr)
	})
	if err == nil || !errors.Is(err, producerErr) || !strings.Contains(err.Error(), "producer exploded") {
		t.Fatalf("SetEffectiveMultipart error = %v, want recovered producer panic", err)
	}
	if coordinator.Effective() != coordinator.raw {
		t.Fatal("panicking producer replaced the effective handle")
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

func TestRequestBodyCoordinator_SetEffectiveBytesReusesRawBeforeSpooling(t *testing.T) {
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	body := []byte(`{"data":"raw"}`)
	coordinator, err := newJSONRequestBody(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body)))
	if err != nil {
		t.Fatalf("newJSONRequestBody: %v", err)
	}
	t.Cleanup(coordinator.Cleanup)

	jsonRequestBodyHandleOptions.TempDir = filepath.Join(t.TempDir(), "missing")
	if err := coordinator.SetEffectiveBytes(append([]byte(nil), body...)); err != nil {
		t.Fatalf("SetEffectiveBytes matching raw: %v", err)
	}
	if coordinator.Effective() != coordinator.raw {
		t.Fatal("matching effective bytes did not reuse raw")
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

func TestRequestBodyCoordinator_CleanupRemovesRawEffectiveAndMultipartTemps(t *testing.T) {
	rawDir, multipartDir := t.TempDir(), t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: rawDir, FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
	t.Setenv("TMP", multipartDir)
	t.Setenv("TEMP", multipartDir)

	coordinator, err := newJSONRequestBody(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"data":"raw"}`)))
	if err != nil {
		t.Fatalf("newJSONRequestBody: %v", err)
	}
	if err := coordinator.SetEffectiveBytes([]byte(`{"data":"effective"}`)); err != nil {
		t.Fatalf("SetEffectiveBytes: %v", err)
	}

	var formBody bytes.Buffer
	writer := multipart.NewWriter(&formBody)
	part, err := writer.CreateFormFile("image", "source.png")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("multipart-temp")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", &formBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(1); err != nil {
		t.Fatalf("ParseMultipartForm: %v", err)
	}
	coordinator.form = req.MultipartForm

	rawPaths, err := filepath.Glob(filepath.Join(rawDir, "*"))
	if err != nil || len(rawPaths) != 2 {
		t.Fatalf("raw/effective spool files = %v, err = %v", rawPaths, err)
	}
	multipartPaths, err := filepath.Glob(filepath.Join(multipartDir, "multipart-*"))
	if err != nil || len(multipartPaths) == 0 {
		t.Fatalf("multipart temp files = %v, err = %v", multipartPaths, err)
	}

	coordinator.Cleanup()
	rawPaths, _ = filepath.Glob(filepath.Join(rawDir, "*"))
	multipartPaths, _ = filepath.Glob(filepath.Join(multipartDir, "multipart-*"))
	if len(rawPaths) != 0 || len(multipartPaths) != 0 {
		t.Fatalf("remaining raw=%v multipart=%v", rawPaths, multipartPaths)
	}
}

func TestUniqueRequestBodyHandles_DeduplicatesPointers(t *testing.T) {
	raw := &service.RequestBodyHandle{}
	effective := &service.RequestBodyHandle{}

	handles := uniqueRequestBodyHandles(raw, raw, nil, effective, effective)

	if len(handles) != 2 || handles[0] != raw || handles[1] != effective {
		t.Fatalf("unique handles = %v, want raw and effective once each", handles)
	}
}

func TestRequestBodyCoordinator_CleanupGinTerminationPaths(t *testing.T) {
	for _, tt := range []struct {
		name string
		want int
	}{
		{name: "success", want: http.StatusNoContent},
		{name: "business rejection", want: http.StatusUnprocessableEntity},
		{name: "cancellation", want: http.StatusRequestTimeout},
		{name: "panic recovery", want: http.StatusInternalServerError},
		{name: "upstream 4xx", want: http.StatusNotFound},
		{name: "upstream 5xx", want: http.StatusServiceUnavailable},
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
				switch tt.name {
				case "success":
					c.Status(http.StatusNoContent)
				case "business rejection":
					if c.GetHeader("X-Request-Valid") != "true" {
						c.AbortWithStatus(http.StatusUnprocessableEntity)
						return
					}
					c.Status(http.StatusNoContent)
				case "cancellation":
					<-c.Request.Context().Done()
					c.Status(http.StatusRequestTimeout)
				case "panic recovery":
					panic("boom")
				case "upstream 4xx", "upstream 5xx":
					upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if tt.name == "upstream 4xx" {
							w.WriteHeader(http.StatusNotFound)
							return
						}
						w.WriteHeader(http.StatusServiceUnavailable)
					}))
					defer upstream.Close()
					response, err := upstream.Client().Get(upstream.URL)
					if err != nil {
						t.Fatalf("upstream request: %v", err)
					}
					defer func() { _ = response.Body.Close() }()
					c.Header("X-Upstream-Status", response.Status)
					c.Status(response.StatusCode)
				}
			})

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(strings.Repeat("request", 64))))
			if tt.name == "cancellation" {
				ctx, cancel := context.WithCancel(req.Context())
				cancel()
				req = req.WithContext(ctx)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != tt.want {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.want)
			}
			if strings.HasPrefix(tt.name, "upstream") && recorder.Header().Get("X-Upstream-Status") == "" {
				t.Fatal("upstream response was not observed")
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
