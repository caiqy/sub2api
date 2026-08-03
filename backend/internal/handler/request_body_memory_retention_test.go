package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type retentionBlockingTransport struct {
	service.HTTPUpstream
	started chan struct{}
	release chan struct{}
}

func (u *retentionBlockingTransport) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	if _, err := io.Copy(io.Discard, req.Body); err != nil {
		return nil, err
	}
	if err := req.Body.Close(); err != nil {
		return nil, err
	}
	close(u.started)
	<-u.release
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_retention","output":[],"usage":{"input_tokens":1,"output_tokens":1}}`)),
	}, nil
}

func retainedHeapAfterGC() uint64 {
	runtime.GC()
	runtime.GC()
	debug.FreeOSMemory()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.HeapAlloc
}

func TestRequestBodyMemoryRetentionWhileUpstreamBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spoolDir := t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{
		SpoolThresholdBytes: service.DefaultRequestBodySpoolThresholdBytes,
		PreviewLimitBytes:   service.DefaultRequestBodyPreviewLimitBytes,
		TempDir:             spoolDir,
	}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	heapAt2MB, previewAt2MB := measureBlockedRequestBodyHeap(t, 2<<20, spoolDir)
	heapAt89MB, previewAt89MB := measureBlockedRequestBodyHeap(t, 89<<20/10, spoolDir)
	growth := uint64(0)
	if heapAt89MB >= heapAt2MB {
		growth = heapAt89MB - heapAt2MB
	}
	t.Logf("heap_at_2mb=%d heap_at_8_9mb=%d retained_growth=%d preview_at_2mb=%d preview_at_8_9mb=%d", heapAt2MB, heapAt89MB, growth, previewAt2MB, previewAt89MB)

	require.Less(t, growth, uint64(3<<20), "retained heap must not scale with full request body size")
	require.LessOrEqual(t, previewAt2MB, int(service.DefaultRequestBodyPreviewLimitBytes))
	require.LessOrEqual(t, previewAt89MB, int(service.DefaultRequestBodyPreviewLimitBytes))
}

func measureBlockedRequestBodyHeap(t *testing.T, size int64, spoolDir string) (uint64, int) {
	t.Helper()
	upstream := &retentionBlockingTransport{started: make(chan struct{}), release: make(chan struct{})}
	done := make(chan struct{})
	release := cleanupMatrixBlockedHandler(t, upstream.release, done)
	group := &service.Group{ID: 1401, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 1401, Name: "retention", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-retention"}}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)
	env.handler.cfg.Gateway.OpenAIWS.Enabled = false
	var requestContext *gin.Context
	router := env.router("/v1/responses", func(c *gin.Context) {
		requestContext = c
		env.handler.Responses(c)
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", retentionJSONBody(size))
	req.Header.Set("Content-Type", "application/json")
	go func() {
		router.ServeHTTP(recorder, req)
		close(done)
	}()
	waitMatrixSignal(t, upstream.started, "retention upstream")

	detail := middleware.GetUsageDetailSnapshot(requestContext)
	require.NotNil(t, detail)
	var snapshot matrixRequestBodyPreview
	require.NoError(t, json.Unmarshal([]byte(detail.UpstreamRequestBody), &snapshot))
	require.True(t, snapshot.Truncated)
	heap := retainedHeapAfterGC()
	previewSize := len(snapshot.Preview)

	release()
	waitMatrixSignal(t, done, "retention handler completion")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assertMatrixTempFiles(t, spoolDir, "sub2api-request-body-", false)
	return heap, previewSize
}

func retentionJSONBody(size int64) io.Reader {
	const prefix = `{"model":"gpt-5","stream":false,"input":"`
	const suffix = `"}`
	return io.MultiReader(
		strings.NewReader(prefix),
		io.LimitReader(retentionPaddingReader{}, size-int64(len(prefix)+len(suffix))),
		strings.NewReader(suffix),
	)
}

type retentionPaddingReader struct{}

func (retentionPaddingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}
