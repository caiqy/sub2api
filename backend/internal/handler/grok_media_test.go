package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldRecordGrokMediaUsage(t *testing.T) {
	tests := []struct {
		name     string
		endpoint service.GrokMediaEndpoint
		model    string
		want     bool
	}{
		{
			name:     "image generation records usage",
			endpoint: service.GrokMediaEndpointImagesGenerations,
			model:    "grok-imagine",
			want:     true,
		},
		{
			name:     "image edit records usage",
			endpoint: service.GrokMediaEndpointImagesEdits,
			model:    "grok-imagine-edit",
			want:     true,
		},
		{
			name:     "video generation records usage",
			endpoint: service.GrokMediaEndpointVideosGenerations,
			model:    "grok-imagine-video-1.5",
			want:     true,
		},
		{
			name:     "video status skips empty model usage",
			endpoint: service.GrokMediaEndpointVideoStatus,
			model:    "",
			want:     false,
		},
		{
			name:     "generation skips usage without model",
			endpoint: service.GrokMediaEndpointImagesGenerations,
			model:    " ",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, shouldRecordGrokMediaUsage(tt.endpoint, tt.model))
		})
	}
}

func TestGrokMedia_MultipartSpoolPreservesFilesAndOmitsSnapshots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawDir := t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, PreviewLimitBytes: 64, TempDir: rawDir, FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "grok-imagine"))
	require.NoError(t, writer.WriteField("prompt", "replace background"))
	image, err := writer.CreateFormFile("image", "source.png")
	require.NoError(t, err)
	_, err = image.Write([]byte("source-secret-" + strings.Repeat("x", 12<<20)))
	require.NoError(t, err)
	mask, err := writer.CreateFormFile("mask", "mask.png")
	require.NoError(t, err)
	_, err = mask.Write([]byte("mask-secret"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	upstream := &openAIImagesSpoolUpstream{started: make(chan struct{}), release: make(chan struct{})}
	group := &service.Group{ID: 903, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	parentID := int64(905)
	account := &service.Account{ID: 904, Name: "grok-media", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}
	parent := &service.Account{ID: parentID, Name: "grok-parent", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Credentials: map[string]any{"access_token": "grok-token"}}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account, parent}}}, upstream)

	var requestContext *gin.Context
	router := env.router("/v1/images/generations", func(c *gin.Context) {
		requestContext = c
		env.handler.GrokImages(c)
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	done := make(chan struct{})
	go func() { router.ServeHTTP(recorder, req); close(done) }()

	select {
	case <-upstream.started:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for grok media upstream")
	}
	require.Equal(t, body.Bytes(), upstream.body)
	require.NotEmpty(t, readTestDir(t, rawDir), "raw body must spool while upstream is blocked")
	detail := middleware.GetUsageDetailSnapshot(requestContext)
	ops, _ := requestContext.Get(service.OpsUpstreamRequestBodyKey)
	for _, snapshot := range []string{detail.RequestBody, detail.UpstreamRequestBody, ops.(string)} {
		require.NotContains(t, snapshot, "source-secret-")
		require.NotContains(t, snapshot, "mask-secret")
	}

	close(upstream.release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for grok media handler")
	}
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Empty(t, readTestDir(t, rawDir))
}
