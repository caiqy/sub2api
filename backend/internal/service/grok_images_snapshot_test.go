package service

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type usageUpstreamSnapshotCollector struct {
	headers string
	body    string
}

func TestGrokMediaRequestInfo_ReleaseText(t *testing.T) {
	request := GrokMediaRequestInfo{
		Prompt:         "large prompt",
		InputImageURLs: []string{"https://example.com/image.png"},
		MaskImageURL:   "https://example.com/mask.png",
		Uploads:        []OpenAIImagesUpload{{FileName: "image.png", Data: []byte("image")}},
		MaskUpload:     &OpenAIImagesUpload{FileName: "mask.png", Data: []byte("mask")},
	}

	request.ReleaseText()

	require.Empty(t, request.Prompt)
	require.Nil(t, request.InputImageURLs)
	require.Empty(t, request.MaskImageURL)
	require.Nil(t, request.Uploads[0].Data)
	require.Nil(t, request.MaskUpload.Data)
}

type grokPreviewAccountRepo struct {
	AccountRepository
	parent *Account
}

func (r grokPreviewAccountRepo) GetByID(context.Context, int64) (*Account, error) {
	return r.parent, nil
}

func (c *usageUpstreamSnapshotCollector) SetUsageUpstreamRequest(headers, body string) {
	c.headers = headers
	c.body = unwrapRequestBodyPreviewForTest(body)
}

func TestBuildGrokResponsesRequestStoresUsageAndOpsUpstreamPreview(t *testing.T) {
	t.Setenv(xai.EnvAllowUnsafeURLOverrides, "true")
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	collector := &usageUpstreamSnapshotCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)

	body := []byte(`{"model":"grok-4.3","input":"hello"}`)
	account := &Account{Platform: PlatformGrok, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "https://xai.test/v1/"}}

	_, err := buildGrokResponsesRequest(context.Background(), c, account, body, "access-token", "", nil)

	require.NoError(t, err)
	require.Contains(t, collector.headers, ":method: POST")
	require.Equal(t, RequestBodyPreviewString(body), collector.body)
	require.Equal(t, RequestBodyPreviewString(body), requireOpsPreviewString(t, c, "hello"))
}

func TestForwardOpenAIImagesAPIKeyStoresBoundedUsageAndOpsUpstreamPreview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prompt := strings.Repeat("x", int(defaultRequestBodyPreviewLimitBytes)+1024)
	body := []byte(`{"model":"gpt-image-2","prompt":"` + prompt + `"}`)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	collector := &usageUpstreamSnapshotCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)

	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad image"}}`)),
	}}}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	account := &Account{
		ID:          9,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
	}

	_, _ = svc.ForwardImages(context.Background(), c, account, body, parsed, "")

	require.NotEmpty(t, collector.body)
	require.LessOrEqual(t, len(collector.body), int(defaultRequestBodyPreviewLimitBytes))
	require.Equal(t, requestBodyPreviewOmittedMarker, collector.body)
	require.Equal(t, collector.body, requireOpsPreviewString(t, c, requestBodyPreviewOmittedMarker))
}

func TestForwardOpenAIImagesAPIKeyOmitsMultipartBinaryFromUsageAndOpsPreview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "replace background"))
	imagePart, err := writer.CreateFormFile("image", "source.png")
	require.NoError(t, err)
	_, err = imagePart.Write([]byte("raw-source-image-bytes"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, c.Request.ParseMultipartForm(0))
	t.Cleanup(func() { _ = c.Request.MultipartForm.RemoveAll() })
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, c.Request.ParseMultipartForm(0))
	t.Cleanup(func() { _ = c.Request.MultipartForm.RemoveAll() })
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	collector := &usageUpstreamSnapshotCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)

	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad image"}}`)),
	}}}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body.Bytes())
	require.NoError(t, err)
	account := &Account{
		ID:          9,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
	}

	_, _ = svc.ForwardImages(context.Background(), c, account, body.Bytes(), parsed, "")

	require.Equal(t, "[multipart body omitted]", collector.body)
	require.Equal(t, "[multipart body omitted]", requireOpsPreviewString(t, c, "omitted"))
}

func TestGrokMediaJSONStoresFinalOutboundPreview(t *testing.T) {
	t.Setenv(xai.EnvAllowUnsafeURLOverrides, "true")
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-imagine","prompt":"draw a preview"}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	collector := &usageUpstreamSnapshotCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
	}}
	parentID := int64(100)
	svc := &OpenAIGatewayService{
		cfg:               &config.Config{},
		httpUpstream:      upstream,
		accountRepo:       grokPreviewAccountRepo{parent: &Account{ID: parentID, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"access_token": "xai-key"}}},
		grokTokenProvider: NewGrokTokenProvider(nil, nil),
	}
	account := &Account{
		ID: 10, Platform: PlatformGrok, Type: AccountTypeOAuth, Concurrency: 1, ParentAccountID: &parentID,
		Credentials: map[string]any{"base_url": "https://xai.test/v1", "access_token": "xai-key", "refresh_token": "xai-refresh", "expires_at": time.Now().Add(time.Hour)},
	}

	_, err := svc.ForwardGrokMedia(context.Background(), c, account, GrokMediaEndpointImagesGenerations, "", body, "application/json")
	require.NoError(t, err)
	require.Equal(t, "grok-imagine-image-quality", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, upstream.lastBody, upstream.replays[0])
	require.Equal(t, RequestBodyPreviewString(upstream.lastBody), collector.body)
	require.Equal(t, collector.body, requireOpsPreviewString(t, c, "grok-imagine-image-quality"))
}

func TestGrokMediaMultipartStoresOmittedPreview(t *testing.T) {
	t.Setenv(xai.EnvAllowUnsafeURLOverrides, "true")
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "grok-imagine-edit"))
	require.NoError(t, writer.WriteField("prompt", "edit preview"))
	part, err := writer.CreateFormFile("image", "private.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("private-image-bytes"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, c.Request.ParseMultipartForm(0))
	t.Cleanup(func() { _ = c.Request.MultipartForm.RemoveAll() })
	collector := &usageUpstreamSnapshotCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
	}}
	parentID := int64(101)
	svc := &OpenAIGatewayService{
		cfg:               &config.Config{},
		httpUpstream:      upstream,
		accountRepo:       grokPreviewAccountRepo{parent: &Account{ID: parentID, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"access_token": "xai-key"}}},
		grokTokenProvider: NewGrokTokenProvider(nil, nil),
	}
	account := &Account{
		ID: 11, Platform: PlatformGrok, Type: AccountTypeOAuth, Concurrency: 1, ParentAccountID: &parentID,
		Credentials: map[string]any{"base_url": "https://xai.test/v1", "access_token": "xai-key", "refresh_token": "xai-refresh", "expires_at": time.Now().Add(time.Hour)},
	}

	_, err = svc.ForwardGrokMedia(context.Background(), c, account, GrokMediaEndpointImagesEdits, "", body.Bytes(), writer.FormDataContentType())
	require.NoError(t, err)
	require.Contains(t, string(upstream.lastBody), "data:application/octet-stream;base64,")
	require.Equal(t, upstream.lastBody, upstream.replays[0])
	require.Equal(t, "[multipart body omitted]", collector.body)
	require.NotContains(t, collector.body, "data:image/")
	require.NotContains(t, collector.body, "private-image-bytes")
	require.Equal(t, "[multipart body omitted]", requireOpsPreviewString(t, c, "omitted"))
}
