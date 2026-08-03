package service

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRequestBodyHandle_MemoryAndFileModes(t *testing.T) {
	t.Run("memory mode keeps bytes in RAM", func(t *testing.T) {
		body := []byte(`{"model":"gpt-5"}`)
		h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{
			SpoolThresholdBytes: 10 << 20,
			PreviewLimitBytes:   5 << 20,
			TempDir:             t.TempDir(),
			FilePrefix:          "sub2api-test-",
		})
		require.NoError(t, err)
		require.Equal(t, int64(len(`{"model":"gpt-5"}`)), h.Size())
		require.NotEmpty(t, h.Hash())
		require.Equal(t, `{"model":"gpt-5"}`, h.PreviewString())
		first, err := h.ReadAll()
		require.NoError(t, err)
		second, err := h.ReadAll()
		require.NoError(t, err)
		require.Equal(t, body, first)
		require.Equal(t, body, second)
	})

	t.Run("file mode reopens full body", func(t *testing.T) {
		body := []byte(strings.Repeat("x", 2048))
		h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{
			SpoolThresholdBytes: 1024,
			PreviewLimitBytes:   256,
			TempDir:             t.TempDir(),
			FilePrefix:          "sub2api-test-",
		})
		require.NoError(t, err)

		r1, err := h.Open()
		require.NoError(t, err)
		first, err := io.ReadAll(r1)
		require.NoError(t, err)
		require.NoError(t, r1.Close())

		r2, err := h.Open()
		require.NoError(t, err)
		second, err := io.ReadAll(r2)
		require.NoError(t, err)
		require.NoError(t, r2.Close())

		require.Equal(t, body, first)
		require.Equal(t, body, second)
		require.Contains(t, h.PreviewString(), "omitted")
	})
}

func TestRequestBodyHandle_CleanupRemovesSpoolFile(t *testing.T) {
	body := []byte(strings.Repeat("z", 2048))
	dir := t.TempDir()
	h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{
		SpoolThresholdBytes: 1024,
		PreviewLimitBytes:   128,
		TempDir:             dir,
		FilePrefix:          "sub2api-test-",
	})
	require.NoError(t, err)

	require.NoError(t, h.Cleanup())
	matches, err := filepath.Glob(filepath.Join(dir, "sub2api-test-*"))
	require.NoError(t, err)
	require.Empty(t, matches)
}

func TestRequestBodyHandle_CleanupDefersSpoolRemovalUntilOpenReaderCloses(t *testing.T) {
	body := []byte(strings.Repeat("z", 2048))
	h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{
		SpoolThresholdBytes: 1024,
		PreviewLimitBytes:   128,
		TempDir:             t.TempDir(),
		FilePrefix:          "sub2api-test-",
	})
	require.NoError(t, err)

	spoolPath := h.spoolPath
	r, err := h.Open()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = r.Close()
		_ = h.Cleanup()
	})

	require.NoError(t, h.Cleanup())
	require.FileExists(t, spoolPath)
	reopened, err := h.Open()
	require.ErrorContains(t, err, "cleaned up")
	require.Nil(t, reopened)

	got, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, body, got)
	require.NoError(t, r.Close())
	require.NoFileExists(t, spoolPath)
}

func TestRequestBodyHandle_OpenAfterFileCleanupReturnsError(t *testing.T) {
	h, err := NewRequestBodyHandleFromBytes([]byte(strings.Repeat("z", 2048)), RequestBodyHandleOptions{
		SpoolThresholdBytes: 1024,
		PreviewLimitBytes:   128,
		TempDir:             t.TempDir(),
		FilePrefix:          "sub2api-test-",
	})
	require.NoError(t, err)
	require.NoError(t, h.Cleanup())
	require.NoError(t, h.Cleanup())

	r, err := h.Open()
	require.Error(t, err)
	require.Nil(t, r)
}

func TestRequestBodyHandle_FileReadErrorsAreSpoolFailures(t *testing.T) {
	newHandle := func(t *testing.T) *RequestBodyHandle {
		t.Helper()
		h, err := NewRequestBodyHandleFromBytes([]byte(strings.Repeat("z", 2048)), RequestBodyHandleOptions{
			SpoolThresholdBytes: 1024,
			PreviewLimitBytes:   128,
			TempDir:             t.TempDir(),
			FilePrefix:          "sub2api-test-",
		})
		require.NoError(t, err)
		return h
	}

	t.Run("open", func(t *testing.T) {
		h := newHandle(t)
		require.NoError(t, os.Remove(h.spoolPath))

		r, err := h.Open()
		require.Nil(t, r)
		require.ErrorIs(t, err, ErrRequestBodySpool)
	})

	t.Run("read all", func(t *testing.T) {
		h := newHandle(t)
		require.NoError(t, os.Remove(h.spoolPath))
		require.NoError(t, os.Mkdir(h.spoolPath, 0o700))

		body, err := h.ReadAll()
		require.Nil(t, body)
		require.ErrorIs(t, err, ErrRequestBodySpool)
	})
}

func TestRequestBodyHandle_CleanupFailureKeepsHandleRetryable(t *testing.T) {
	h, err := NewRequestBodyHandleFromBytes([]byte(strings.Repeat("z", 2048)), RequestBodyHandleOptions{
		SpoolThresholdBytes: 1024,
		PreviewLimitBytes:   128,
		TempDir:             t.TempDir(),
		FilePrefix:          "sub2api-test-",
	})
	require.NoError(t, err)

	removeErr := errors.New("remove failed")
	require.ErrorIs(t, h.cleanup(func(string) error { return removeErr }), removeErr)
	require.False(t, h.cleaned)
	require.True(t, h.spoolActive)
	require.NotEmpty(t, h.spoolPath)
	require.NoError(t, h.Cleanup())
	require.True(t, h.cleaned)
}

func TestRequestBodyPreviewString_RedactsOnlyWholeDataURLStringValues(t *testing.T) {
	const payload = "c2VjcmV0"
	tests := []struct {
		name    string
		body    string
		omitted bool
	}{
		{name: "attachment", body: `{"attachment":"data:image/png;base64,` + payload + `"}`, omitted: true},
		{name: "text direct data URL", body: `{"text":" data:image/png;base64,` + payload + `"}`, omitted: true},
		{name: "text mentions data URL", body: `{"text":"paste data:image/png;base64,` + payload + ` here"}`},
		{name: "empty data URL", body: `{"attachment":"data:image/png;base64,"}`},
		{name: "truncated JSON", body: `{"text":"data:image/png;base64,` + payload, omitted: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preview := RequestBodyPreviewString([]byte(tt.body))
			if tt.omitted {
				require.Equal(t, requestBodyPreviewOmittedMarker, preview)
				require.NotContains(t, preview, payload)
				return
			}
			require.Equal(t, tt.body, preview)
		})
	}
}

func TestRetryRequestBodyHandleCleanupRetriesFailure(t *testing.T) {
	var calls int
	done := make(chan struct{})
	retryRequestBodyHandleCleanup(func() error {
		calls++
		if calls == 1 {
			return errors.New("remove failed")
		}
		close(done)
		return nil
	})

	select {
	case <-done:
		require.Equal(t, 2, calls)
	case <-time.After(time.Second):
		t.Fatal("cleanup retry did not run")
	}
}

func TestRequestBodyHandle_CleanupRetryGoroutineCompletes(t *testing.T) {
	h := &RequestBodyHandle{spoolPath: filepath.Join(t.TempDir(), "missing"), spoolActive: true}
	h.scheduleCleanupRetry()

	require.Eventually(t, func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		return h.cleaned && !h.retrying
	}, time.Second, 10*time.Millisecond)
}

func TestRequestBodyHandle_MemoryCleanupReleasesAndRejectsReuse(t *testing.T) {
	h, err := NewRequestBodyHandleFromBytes([]byte(`{"model":"gpt-5"}`), RequestBodyHandleOptions{
		SpoolThresholdBytes: 10 << 20,
		PreviewLimitBytes:   64,
		TempDir:             t.TempDir(),
		FilePrefix:          "sub2api-test-",
	})
	require.NoError(t, err)
	require.NotNil(t, h.memory)

	require.NoError(t, h.Cleanup())
	require.NoError(t, h.Cleanup())
	require.True(t, h.cleaned)
	require.Nil(t, h.memory)

	r, err := h.Open()
	require.ErrorContains(t, err, "cleaned up")
	require.Nil(t, r)
	read, err := h.ReadAll()
	require.ErrorContains(t, err, "cleaned up")
	require.Nil(t, read)
}

func TestRequestBodyHandle_NilReadAllReturnsError(t *testing.T) {
	var h *RequestBodyHandle

	body, err := h.ReadAll()

	require.Nil(t, body)
	require.ErrorContains(t, err, "request body handle is nil")
}

func TestRequestBodyHandle_ConcurrentOpenAndCleanupNeverSucceedsEmpty(t *testing.T) {
	for _, tt := range []struct {
		name      string
		threshold int64
	}{
		{name: "memory", threshold: 4096},
		{name: "file", threshold: 1024},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(strings.Repeat("c", 2048))
			h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{
				SpoolThresholdBytes: tt.threshold,
				PreviewLimitBytes:   128,
				TempDir:             t.TempDir(),
				FilePrefix:          "sub2api-test-",
			})
			require.NoError(t, err)

			start := make(chan struct{})
			errs := make(chan error, 64)
			var wg sync.WaitGroup
			for i := 0; i < 64; i++ {
				wg.Add(1)
				go func(cleanup bool) {
					defer wg.Done()
					<-start
					if cleanup {
						errs <- h.Cleanup()
						return
					}
					r, err := h.Open()
					if err != nil {
						errs <- nil
						return
					}
					got, readErr := io.ReadAll(r)
					closeErr := r.Close()
					if readErr != nil {
						errs <- readErr
						return
					}
					if closeErr != nil {
						errs <- closeErr
						return
					}
					if !bytes.Equal(got, body) {
						errs <- errors.New("Open succeeded without the complete body")
						return
					}
					errs <- nil
				}(i%2 == 0)
			}
			close(start)
			wg.Wait()
			close(errs)
			for err := range errs {
				require.NoError(t, err)
			}
			require.NoError(t, h.Cleanup())
			r, err := h.Open()
			require.ErrorContains(t, err, "cleaned up")
			require.Nil(t, r)
		})
	}
}

func TestRequestBodyHandle_ZeroOptionsUseSafeDefaults(t *testing.T) {
	body := []byte(`{"model":"gpt-5"}`)
	h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{})
	require.NoError(t, err)
	defer func() { _ = h.Cleanup() }()

	require.Equal(t, string(body), h.PreviewString())
	read, err := h.ReadAll()
	require.NoError(t, err)
	require.Equal(t, body, read)
}

func TestRequestBodyPreviewStringFromBytesUsesDefaultLimit(t *testing.T) {
	body := []byte(strings.Repeat("x", int(DefaultRequestBodyPreviewLimitBytes)+1))

	preview := RequestBodyPreviewString(body)

	require.Equal(t, requestBodyPreviewOmittedMarker, preview)
}

func TestRequestBodyPreviewsOmitInlineBinaryPayloads(t *testing.T) {
	tests := map[string]string{
		"responses data URL":  `{"input":[{"type":"input_image","image_url":"data:image/png;base64,c2VjcmV0LXJlc3BvbnNlcw=="}]}`,
		"grok b64_json":       `{"image":{"b64_json":"c2VjcmV0LWdyb2s="}}`,
		"gemini inlineData":   `{"contents":[{"parts":[{"inlineData":{"mimeType":"image/png","data":"c2VjcmV0LWdlbWluaQ=="}}]}]}`,
		"gemini inline_data":  `{"contents":[{"parts":[{"inline_data":{"mime_type":"image/png","data":"c2VjcmV0LWdlbWluaQ=="}}]}]}`,
		"anthropic base64":    `{"messages":[{"content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"c2VjcmV0LWFudGhyb3BpYw=="}}]}]}`,
		"escaped data URL":    `{"image\u005furl":"  data:image/png\u003bbase64\u002c  c2VjcmV0LWVzY2FwZWQ="}`,
		"percent data URL":    `{"mask":"  data:image/png,%89PNG%0D%0A"}`,
		"escaped Gemini keys": `{"inline\u0044ata":{"d\u0061ta":"c2VjcmV0LWdlbWluaQ=="}}`,
		"escaped whitespace":  `{"mask_image_url":"\t\ndata:image/png,%89PNG"}`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			preview := RequestBodyPreviewString([]byte(body))
			require.Contains(t, preview, "omitted")
			require.NotContains(t, preview, "c2VjcmV0")

			h, err := NewRequestBodyHandleFromBytes([]byte(body), RequestBodyHandleOptions{})
			require.NoError(t, err)
			defer func() { _ = h.Cleanup() }()
			require.Equal(t, preview, h.PreviewString())
		})
	}
}

func TestRequestBodyPreviewsKeepDataURLPrefixWithoutPayload(t *testing.T) {
	body := []byte(`{"input":"Documentation mentions data:image/png;base64, but contains no payload","empty":"data:image/png;base64,"}`)

	require.Equal(t, string(body), RequestBodyPreviewString(body))
}

func TestRequestBodyPreviewsKeepTextAndCrossObjectLookalikes(t *testing.T) {
	tests := map[string]string{
		"text contains payload-like data URL":   `{"text":"paste data:image/png;base64,c2VjcmV0 here"}`,
		"Gemini data is in sibling object":      `{"parts":[{"inlineData":{"mimeType":"image/png"}},{"data":"c2VjcmV0"}]}`,
		"Gemini data is nested":                 `{"inline_data":{"nested":{"data":"c2VjcmV0"}}}`,
		"Anthropic fields split across sources": `{"content":[{"source":{"type":"base64"}},{"source":{"data":"c2VjcmV0"}}]}`,
		"empty binary fields":                   `{"b64_json":"","image_url":"data:image/png;base64,","inlineData":{"data":""},"source":{"type":"base64","data":""}}`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, body, RequestBodyPreviewString([]byte(body)))
		})
	}
	require.Equal(t, requestBodyPreviewOmittedMarker, RequestBodyPreviewString([]byte(`{"image_description":"data:image/png;base64,c2VjcmV0","image_prompt":"data:image/png,%89PNG","masking_instructions":"data:image/png;base64,c2VjcmV0"}`)))
}

func TestRequestBodyPreviewOnlyScansBoundedPrefix(t *testing.T) {
	prefix := `{"text":"` + strings.Repeat("x", int(DefaultRequestBodyPreviewLimitBytes))
	body := []byte(prefix + `","image_url":"data:image/png;base64,c2VjcmV0"}`)

	preview := RequestBodyPreviewString(body)

	require.Equal(t, requestBodyPreviewOmittedMarker, preview)

	sensitivePrefix := []byte(`{"image_url":"data:image/png;base64,c2VjcmV0","text":"` + strings.Repeat("x", int(DefaultRequestBodyPreviewLimitBytes)) + `"}`)
	require.Contains(t, RequestBodyPreviewString(sensitivePrefix), "omitted")
}

func TestRequestBodyPreviewKeepsDeepOrdinaryJSONWhenScannerStops(t *testing.T) {
	body := strings.Repeat("[", 257) + "0" + strings.Repeat("]", 257)

	require.Equal(t, body, RequestBodyPreviewString([]byte(body)))
}

func TestRequestBodyPreviewFindsSensitivePayloadAtDepth257(t *testing.T) {
	body := strings.Repeat("[", 257) + `{"image_url":"data:image/png,%89PNG"}` + strings.Repeat("]", 257)

	require.Contains(t, RequestBodyPreviewString([]byte(body)), "omitted")
}

func TestRequestBodyPreviewFailsClosedForDataFirstSourceAcrossBound(t *testing.T) {
	body := []byte(`{"source":{"data":"c2VjcmV0` + strings.Repeat("x", int(DefaultRequestBodyPreviewLimitBytes)) + `","type":"base64"}}`)

	require.Contains(t, RequestBodyPreviewString(body), "omitted")
}

func TestRequestBodyPreviewMalformedInputReturnsQuickly(t *testing.T) {
	done := make(chan string, 1)
	go func() { done <- RequestBodyPreviewString([]byte(`[}`)) }()

	select {
	case preview := <-done:
		require.Equal(t, requestBodyPreviewOmittedMarker, preview)
	case <-time.After(time.Second):
		t.Fatal("malformed preview scan hung")
	}
}

func TestRequestBodyPreviewRejectsMultipleTopLevelJSONValues(t *testing.T) {
	for _, body := range []string{`{} {}`, `null null`} {
		require.Equal(t, requestBodyPreviewOmittedMarker, RequestBodyPreviewString([]byte(body)), body)
	}
}

func TestContainsNonEmptyDataURL(t *testing.T) {
	for _, tt := range []struct {
		raw  string
		want bool
	}{
		{raw: "data:image/png;base64,aGVsbG8=", want: true},
		{raw: "data:text/plain,hello%20world", want: true},
		{raw: "data:image/png;base64,  payload", want: true},
		{raw: "data:image/png;base64,"},
		{raw: "data:image/png;base64,   "},
		{raw: strings.Repeat("data:image/png;base64", 10000)},
	} {
		require.Equal(t, tt.want, containsNonEmptyDataURL(tt.raw), tt.raw)
	}

	source, err := os.ReadFile("request_body_handle.go")
	require.NoError(t, err)
	require.NotContains(t, string(source), "isNonEmptyDataURL(raw[i:])")
}

func TestRequestBodyPreviewUsesStandardTokenScannerContract(t *testing.T) {
	sourceBytes, err := os.ReadFile("request_body_handle.go")
	require.NoError(t, err)
	source := string(sourceBytes)

	require.Contains(t, source, ".Token()")
	require.NotContains(t, source, "regexp.MustCompile")
	require.NotContains(t, source, "hasFallbackSensitiveSignal")
	for _, forbidden := range []string{"nextDecodedJSONByte", "scanObject", "scanArray", "jsonStringEnd", "hexNibble"} {
		require.NotContains(t, source, forbidden)
	}
	parseStart := strings.Index(source, "func parseRequestBodyPreviewSnapshot")
	require.NotEqual(t, -1, parseStart)
	parseSource := source[parseStart:]
	require.NotContains(t, parseSource, "json.Unmarshal")
	require.Contains(t, parseSource, "gjson.")
}

func TestRequestBodyHandle_SpoolCreateFailure(t *testing.T) {
	body := []byte(strings.Repeat("x", 2048))
	h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{
		SpoolThresholdBytes: 1024,
		PreviewLimitBytes:   128,
		TempDir:             filepath.Join(t.TempDir(), "missing"),
		FilePrefix:          "sub2api-test-",
	})

	require.Error(t, err)
	require.ErrorIs(t, err, ErrRequestBodySpool)
	require.Nil(t, h)
}

type requestBodyClientErrorReader struct{ err error }

func (r requestBodyClientErrorReader) Read([]byte) (int, error) { return 0, r.err }

func TestRequestBodyHandle_ClientReadFailureIsNotSpoolFailure(t *testing.T) {
	clientErr := errors.New("client body read failed")
	h, err := NewRequestBodyHandleFromReader(requestBodyClientErrorReader{err: clientErr}, RequestBodyHandleOptions{})

	require.Nil(t, h)
	require.ErrorIs(t, err, clientErr)
	require.NotErrorIs(t, err, ErrRequestBodySpool)
}

func TestRequestBodyHandle_CleanupStaleRequestBodySpoolFiles(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "sub2api-test-stale")
	fresh := filepath.Join(dir, "sub2api-test-fresh")
	other := filepath.Join(dir, "other-stale")
	require.NoError(t, os.WriteFile(stale, []byte("stale"), 0o600))
	require.NoError(t, os.WriteFile(fresh, []byte("fresh"), 0o600))
	require.NoError(t, os.WriteFile(other, []byte("other"), 0o600))

	now := time.Now()
	require.NoError(t, os.Chtimes(stale, now.Add(-2*time.Hour), now.Add(-2*time.Hour)))
	require.NoError(t, os.Chtimes(other, now.Add(-2*time.Hour), now.Add(-2*time.Hour)))

	require.NoError(t, CleanupStaleRequestBodySpoolFiles(dir, "sub2api-test-", time.Hour, now))
	require.NoFileExists(t, stale)
	require.FileExists(t, fresh)
	require.FileExists(t, other)
}
