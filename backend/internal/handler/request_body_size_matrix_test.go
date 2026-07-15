package handler

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequestBodySizeMatrix(t *testing.T) {
	for _, tt := range []struct {
		name      string
		size      int
		gzip      bool
		multipart bool
	}{
		{"identity/5MB", 5 << 20, false, false},
		{"identity/10MB", 10 << 20, false, false},
		{"identity/12MB", 12 << 20, false, false},
		{"gzip/5MB", 5 << 20, true, false},
		{"gzip/10MB", 10 << 20, true, false},
		{"gzip/12MB", 12 << 20, true, false},
		{"multipart/5MB", 5 << 20, false, true},
		{"multipart/10MB", 10 << 20, false, true},
		{"multipart/12MB", 12 << 20, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.multipart {
				testRequestBodySizeMatrixMultipart(t, tt.size)
				return
			}
			testRequestBodySizeMatrixJSON(t, tt.size, tt.gzip)
		})
	}
}

func testRequestBodySizeMatrixJSON(t *testing.T, size int, compressed bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rawDir, effectiveDir := t.TempDir(), t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, PreviewLimitBytes: 64, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
	t.Setenv("TMPDIR", effectiveDir)
	t.Setenv("TMP", effectiveDir)
	t.Setenv("TEMP", effectiveDir)

	clientBody := matrixJSONBody(t, size)
	requestBody := clientBody
	if compressed {
		var encoded bytes.Buffer
		writer := gzip.NewWriter(&encoded)
		require.NoError(t, func() error { _, err := writer.Write(clientBody); return err }())
		require.NoError(t, writer.Close())
		requestBody = encoded.Bytes()
	}

	upstream := &matrixBlockedUpstream{started: make(chan struct{}), release: make(chan struct{})}
	done := make(chan struct{})
	release := cleanupMatrixBlockedHandler(t, upstream.release, done)
	group := &service.Group{ID: 1301, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 1301, Name: "matrix", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-matrix"}}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)
	var requestContext *gin.Context
	router := env.router("/v1/embeddings", func(c *gin.Context) {
		requestContext = c
		env.handler.Embeddings(c)
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	if compressed {
		req.Header.Set("Content-Encoding", "gzip")
	}
	go func() { router.ServeHTTP(recorder, req); close(done) }()
	waitMatrixSignal(t, upstream.started, "JSON upstream")

	clientHash := matrixHash(clientBody)
	upstreamBody, upstreamHash := upstream.snapshot()
	require.Equal(t, clientBody, upstreamBody)
	require.Equal(t, clientHash, upstreamHash)
	t.Logf("client_sha256=%s upstream_sha256=%s decoded_bytes=%d", clientHash, upstreamHash, len(clientBody))
	detail := middleware.GetUsageDetailSnapshot(requestContext)
	require.NotNil(t, detail)
	ops, ok := requestContext.Get(service.OpsUpstreamRequestBodyKey)
	require.True(t, ok)
	opsBody, ok := ops.(string)
	require.True(t, ok)
	assertMatrixRequestBodySnapshot(t, "usage request", detail.RequestBody, clientBody, "")
	assertMatrixRequestBodySnapshot(t, "usage upstream", detail.UpstreamRequestBody, upstreamBody, "")
	assertMatrixRequestBodySnapshot(t, "ops upstream", opsBody, upstreamBody, "")
	assertMatrixTempFiles(t, rawDir, "sub2api-request-body-", size > 10<<20)
	assertMatrixTempFiles(t, effectiveDir, "sub2api-request-body-", false)

	release()
	waitMatrixSignal(t, done, "JSON handler completion")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assertMatrixTempFiles(t, rawDir, "sub2api-request-body-", false)
	assertMatrixTempFiles(t, effectiveDir, "sub2api-request-body-", false)
}

func testRequestBodySizeMatrixMultipart(t *testing.T, size int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rawDir, formDir := t.TempDir(), t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, PreviewLimitBytes: 64, TempDir: rawDir}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
	t.Setenv("TMPDIR", formDir)
	t.Setenv("TMP", formDir)
	t.Setenv("TEMP", formDir)

	clientBody, contentType := matrixMultipartBody(t, size)
	upstream := &openAIImagesSpoolUpstream{started: make(chan struct{}), release: make(chan struct{})}
	done := make(chan struct{})
	release := cleanupMatrixBlockedHandler(t, upstream.release, done)
	group := &service.Group{ID: 1302, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	parentID := int64(1303)
	account := &service.Account{ID: 1302, Name: "matrix", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}
	parent := &service.Account{ID: parentID, Name: "matrix-parent", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Credentials: map[string]any{"access_token": "matrix"}}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account, parent}}}, upstream)
	var requestContext *gin.Context
	router := env.router("/v1/images/generations", func(c *gin.Context) {
		requestContext = c
		env.handler.GrokImages(c)
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(clientBody))
	req.Header.Set("Content-Type", contentType)
	go func() { router.ServeHTTP(recorder, req); close(done) }()
	waitMatrixSignal(t, upstream.started, "multipart upstream")

	clientHash := matrixHash(clientBody)
	upstreamHash := matrixHash(upstream.body)
	require.Equal(t, clientBody, upstream.body)
	require.Equal(t, clientHash, upstreamHash)
	t.Logf("client_sha256=%s upstream_sha256=%s multipart_bytes=%d", clientHash, upstreamHash, len(clientBody))
	detail := middleware.GetUsageDetailSnapshot(requestContext)
	require.NotNil(t, detail)
	ops, ok := requestContext.Get(service.OpsUpstreamRequestBodyKey)
	require.True(t, ok)
	opsBody, ok := ops.(string)
	require.True(t, ok)
	for _, snapshot := range []string{detail.RequestBody, detail.UpstreamRequestBody, opsBody} {
		assertMatrixRequestBodySnapshot(t, "multipart snapshot", snapshot, clientBody, "matrix-file-")
	}
	assertMatrixTempFiles(t, rawDir, "sub2api-request-body-", size > 10<<20)
	assertMatrixTempFiles(t, formDir, "multipart-", true)

	release()
	waitMatrixSignal(t, done, "multipart handler completion")
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	assertMatrixTempFiles(t, rawDir, "sub2api-request-body-", false)
	assertMatrixTempFiles(t, formDir, "multipart-", false)
}

func matrixJSONBody(t *testing.T, size int) []byte {
	t.Helper()
	const prefix = `{"model":"matrix-model","input":"`
	const suffix = `"}`
	require.Greater(t, size, len(prefix)+len(suffix))
	body := []byte(prefix + strings.Repeat("x", size-len(prefix)-len(suffix)) + suffix)
	require.Len(t, body, size)
	return body
}

func matrixMultipartBody(t *testing.T, size int) ([]byte, string) {
	t.Helper()
	const boundary = "sub2api-request-body-size-matrix"
	build := func(fileSize int) (*bytes.Buffer, string) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.SetBoundary(boundary))
		require.NoError(t, writer.WriteField("model", "grok-imagine"))
		require.NoError(t, writer.WriteField("prompt", "matrix prompt"))
		file, err := writer.CreateFormFile("image", "matrix.png")
		require.NoError(t, err)
		payload := bytes.Repeat([]byte("x"), fileSize)
		copy(payload, "matrix-file-")
		_, err = file.Write(payload)
		require.NoError(t, err)
		require.NoError(t, writer.Close())
		return &body, writer.FormDataContentType()
	}
	empty, _ := build(0)
	body, contentType := build(size - empty.Len())
	require.Len(t, body.Bytes(), size)
	return body.Bytes(), contentType
}

type matrixRequestBodyPreview struct {
	Kind      string `json:"kind"`
	Preview   string `json:"preview"`
	Truncated bool   `json:"truncated"`
	Size      int64  `json:"size"`
}

func assertMatrixRequestBodySnapshot(t *testing.T, name, raw string, body []byte, omitted string) {
	t.Helper()
	var snapshot matrixRequestBodyPreview
	require.NoErrorf(t, json.Unmarshal([]byte(raw), &snapshot), "%s must be a request body snapshot", name)
	require.Equalf(t, "request_body_preview", snapshot.Kind, "%s snapshot kind", name)
	require.Equalf(t, int64(len(body)), snapshot.Size, "%s snapshot size", name)
	require.Truef(t, snapshot.Truncated, "%s snapshot must be truncated", name)
	require.NotEmptyf(t, snapshot.Preview, "%s snapshot preview", name)
	require.LessOrEqualf(t, len(snapshot.Preview), int(openAIResponsesRequestBodyPreviewLimitBytes), "%s preview exceeds the production limit", name)
	if omitted != "" {
		require.NotContainsf(t, snapshot.Preview, omitted, "%s must omit multipart file content", name)
		require.NotContainsf(t, raw, omitted, "%s wrapper must omit multipart file content", name)
	}
}

func assertMatrixTempFiles(t *testing.T, dir, prefix string, want bool) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	if !want {
		require.Emptyf(t, entries, "expected %s to be empty", dir)
		return
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			return
		}
	}
	t.Fatalf("expected %s files in %s", prefix, dir)
}

func cleanupMatrixBlockedHandler(t *testing.T, releaseChan chan struct{}, done <-chan struct{}) func() {
	t.Helper()
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseChan) }) }
	t.Cleanup(func() {
		release()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("timed out waiting for blocked matrix handler cleanup")
		}
	})
	return release
}

type matrixBlockedUpstream struct {
	service.HTTPUpstream
	body    []byte
	hash    string
	started chan struct{}
	release chan struct{}
}

func (u *matrixBlockedUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	reopened, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reopened.Close() }()
	if replay, err := io.ReadAll(reopened); err != nil || !bytes.Equal(body, replay) {
		return nil, io.ErrUnexpectedEOF
	}
	u.body, u.hash = body, matrixHash(body)
	close(u.started)
	select {
	case <-u.release:
	case <-time.After(5 * time.Second):
		return nil, io.ErrClosedPipe
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"object":"list","data":[{"object":"embedding","embedding":[0.1],"index":0}],"model":"matrix-model","usage":{"prompt_tokens":1,"total_tokens":1}}`))}, nil
}

func (u *matrixBlockedUpstream) snapshot() ([]byte, string) {
	return append([]byte(nil), u.body...), u.hash
}

func matrixHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func waitMatrixSignal(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}
