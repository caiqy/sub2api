package handler

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type openAIImagesHandlerAccountRepo struct {
	service.AccountRepository
	accounts []service.Account
}

type openAIImagesSpoolSwitchRepo struct {
	*openAIRetryAccountRepoStub
	onList func()
}

func (r openAIImagesSpoolSwitchRepo) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]service.Account, error) {
	r.onList()
	return r.openAIRetryAccountRepoStub.ListSchedulableByGroupIDAndPlatform(ctx, groupID, platform)
}

func (r openAIImagesSpoolSwitchRepo) ListSchedulableByPlatform(ctx context.Context, platform string) ([]service.Account, error) {
	r.onList()
	return r.openAIRetryAccountRepoStub.ListSchedulableByPlatform(ctx, platform)
}

func (r openAIImagesHandlerAccountRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			account := r.accounts[i]
			return &account, nil
		}
	}
	return nil, service.ErrNoAvailableAccounts
}

func (r openAIImagesHandlerAccountRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r openAIImagesHandlerAccountRepo) ListSchedulableByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r openAIImagesHandlerAccountRepo) ListSchedulableUngroupedByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r openAIImagesHandlerAccountRepo) accountsForPlatform(platform string) []service.Account {
	out := make([]service.Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if account.Platform == platform {
			out = append(out, account)
		}
	}
	return out
}

type openAIImagesHandlerHTTPUpstream struct {
	service.HTTPUpstream
	mu           sync.Mutex
	accountIDs   []int64
	contentTypes []string
	resp         *http.Response
}

func (u *openAIImagesHandlerHTTPUpstream) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.mu.Lock()
	u.accountIDs = append(u.accountIDs, accountID)
	u.contentTypes = append(u.contentTypes, req.Header.Get("Content-Type"))
	u.mu.Unlock()
	return u.resp, nil
}

func (u *openAIImagesHandlerHTTPUpstream) calls() []int64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]int64(nil), u.accountIDs...)
}

func (u *openAIImagesHandlerHTTPUpstream) contentType() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.contentTypes) == 0 {
		return ""
	}
	return u.contentTypes[len(u.contentTypes)-1]
}

type openAIImagesSpoolUpstream struct {
	service.HTTPUpstream
	body    []byte
	started chan struct{}
	release chan struct{}
}

type openAIImagesReplayUpstream struct {
	service.HTTPUpstream
	mu           sync.Mutex
	bodies       [][]byte
	contentTypes []string
	lengths      []int64
	accountIDs   []int64
}

func (u *openAIImagesReplayUpstream) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	getBody, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	reopened, err := io.ReadAll(getBody)
	_ = getBody.Close()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(body, reopened) {
		return nil, io.ErrUnexpectedEOF
	}
	u.mu.Lock()
	u.bodies = append(u.bodies, body)
	u.contentTypes = append(u.contentTypes, req.Header.Get("Content-Type"))
	u.lengths = append(u.lengths, req.ContentLength)
	u.accountIDs = append(u.accountIDs, accountID)
	attempt := len(u.bodies)
	u.mu.Unlock()
	if attempt == 1 {
		status := http.StatusInternalServerError
		if accountID == 920 {
			status = http.StatusTooManyRequests
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"retry"}}`))}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"created":1,"data":[{"b64_json":"aGVsbG8="}]}`))}, nil
}

func (u *openAIImagesReplayUpstream) assert(t *testing.T, wantAccounts []int64, wantModels []string, wantSameBody bool) {
	t.Helper()
	u.mu.Lock()
	defer u.mu.Unlock()
	require.Equal(t, wantAccounts, u.accountIDs)
	require.Len(t, u.bodies, len(wantModels))
	for i, body := range u.bodies {
		require.Equal(t, int64(len(body)), u.lengths[i])
		mediaType, params, err := mime.ParseMediaType(u.contentTypes[i])
		require.NoError(t, err)
		require.Equal(t, "multipart/form-data", mediaType)
		form, err := multipart.NewReader(bytes.NewReader(body), params["boundary"]).ReadForm(0)
		require.NoError(t, err)
		require.Equal(t, wantModels[i], form.Value["model"][0])
		require.Len(t, form.File["image"], 1)
		require.NoError(t, form.RemoveAll())
	}
	if wantSameBody {
		require.Equal(t, u.bodies[0], u.bodies[1])
		require.Equal(t, u.contentTypes[0], u.contentTypes[1])
	}
}

func (u *openAIImagesSpoolUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	var err error
	u.body, err = io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	close(u.started)
	<-u.release
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"created":1,"data":[{"b64_json":"aGVsbG8="}]}`)),
	}, nil
}

func TestOpenAIImages_InlineSpoolKeepsRawBodyAndOmitsSnapshots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rawDir := t.TempDir()
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, PreviewLimitBytes: 64, TempDir: rawDir, FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	body := []byte(`{"model":"gpt-image-1","prompt":"inline-secret-` + strings.Repeat("x", 12<<20) + `","images":[{"image_url":"data:image/png;base64,aW5saW5lLXNlY3JldA=="}]}`)
	upstream := &openAIImagesSpoolUpstream{started: make(chan struct{}), release: make(chan struct{})}
	group := &service.Group{ID: 901, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	account := &service.Account{ID: 902, Name: "images", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: []*service.Account{account}}, upstream)

	var requestContext *gin.Context
	router := env.router("/v1/images/generations", func(c *gin.Context) {
		requestContext = c
		env.handler.Images(c)
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	done := make(chan struct{})
	go func() { router.ServeHTTP(recorder, req); close(done) }()

	select {
	case <-upstream.started:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for images upstream")
	}
	require.Equal(t, body, upstream.body)
	require.NotEmpty(t, readTestDir(t, rawDir), "raw body must spool while upstream is blocked")
	detail := middleware2.GetUsageDetailSnapshot(requestContext)
	ops, _ := requestContext.Get(service.OpsUpstreamRequestBodyKey)
	for _, snapshot := range []string{detail.RequestBody, detail.UpstreamRequestBody, ops.(string)} {
		require.NotContains(t, snapshot, "inline-secret-")
		require.NotContains(t, snapshot, "aW5saW5lLXNlY3JldA==")
	}

	close(upstream.release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for images handler")
	}
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Empty(t, readTestDir(t, rawDir))
}

func TestOpenAIGatewayHandlerImages_MultipartEffectiveSpoolFailureReturns503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir(), FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "draw"))
	image, err := writer.CreateFormFile("image", "source.png")
	require.NoError(t, err)
	_, err = image.Write([]byte("image"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	group := &service.Group{ID: 910, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	account := &service.Account{ID: 911, Name: "api-key", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}}
	upstream := &openAIImagesHandlerHTTPUpstream{}
	repo := openAIImagesSpoolSwitchRepo{
		openAIRetryAccountRepoStub: &openAIRetryAccountRepoStub{accounts: []*service.Account{account}},
		onList:                     func() { jsonRequestBodyHandleOptions.TempDir = filepath.Join(t.TempDir(), "missing") },
	}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, repo, upstream)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	env.router("/v1/images/edits", env.handler.Images).ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	require.Empty(t, upstream.calls(), "effective spool failure must not send an upstream request")
}

func TestOpenAIGatewayHandlerImages_OAuthMultipartSkipsEffectiveSpool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir(), FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "draw"))
	image, err := writer.CreateFormFile("image", "source.png")
	require.NoError(t, err)
	_, err = image.Write([]byte("image"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	group := &service.Group{ID: 912, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	account := &service.Account{ID: 913, Name: "oauth", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"access_token": "token"}}
	upstream := &openAIImagesHandlerHTTPUpstream{resp: &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"upstream"}}`))}}
	repo := openAIImagesSpoolSwitchRepo{
		openAIRetryAccountRepoStub: &openAIRetryAccountRepoStub{accounts: []*service.Account{account}},
		onList:                     func() { jsonRequestBodyHandleOptions.TempDir = filepath.Join(t.TempDir(), "missing") },
	}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, repo, upstream)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	env.router("/v1/images/edits", env.handler.Images).ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.Equal(t, []int64{account.ID}, upstream.calls())
	require.Equal(t, "application/json", upstream.contentType())
}

func TestOpenAIGatewayHandlerImages_MultipartReplayUsesMappedEffectiveBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name         string
		accounts     []*service.Account
		wantAccounts []int64
		wantModels   []string
		wantSameBody bool
	}{
		{
			name:         "same account retry",
			accounts:     []*service.Account{{ID: 920, Name: "pool", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "sk", "pool_mode": true, "pool_mode_retry_count": 1, "model_mapping": map[string]any{"gpt-image-2": "gpt-image-mapped"}}}},
			wantAccounts: []int64{920, 920}, wantModels: []string{"gpt-image-mapped", "gpt-image-mapped"}, wantSameBody: true,
		},
		{
			name: "cross account same model",
			accounts: []*service.Account{
				{ID: 921, Name: "first", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk", "model_mapping": map[string]any{"gpt-image-2": "gpt-image-mapped"}}},
				{ID: 922, Name: "second", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{"api_key": "sk", "model_mapping": map[string]any{"gpt-image-2": "gpt-image-mapped"}}},
			},
			wantAccounts: []int64{921, 922}, wantModels: []string{"gpt-image-mapped", "gpt-image-mapped"}, wantSameBody: true,
		},
		{
			name: "cross account different model",
			accounts: []*service.Account{
				{ID: 923, Name: "first", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk", "model_mapping": map[string]any{"gpt-image-2": "gpt-image-one"}}},
				{ID: 924, Name: "second", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{"api_key": "sk", "model_mapping": map[string]any{"gpt-image-2": "gpt-image-two"}}},
			},
			wantAccounts: []int64{923, 924}, wantModels: []string{"gpt-image-one", "gpt-image-two"}, wantSameBody: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("model", "gpt-image-2"))
			require.NoError(t, writer.WriteField("prompt", "draw"))
			image, err := writer.CreateFormFile("image", "source.png")
			require.NoError(t, err)
			_, err = image.Write([]byte("image"))
			require.NoError(t, err)
			require.NoError(t, writer.Close())

			upstream := &openAIImagesReplayUpstream{}
			group := &service.Group{ID: 919, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
			env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &openAIRetryAccountRepoStub{accounts: tt.accounts}, upstream)
			env.handler.maxAccountSwitches = 1
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
			req.Header.Set("Content-Type", writer.FormDataContentType())
			env.router("/v1/images/edits", env.handler.Images).ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			upstream.assert(t, tt.wantAccounts, tt.wantModels, tt.wantSameBody)
		})
	}
}

func TestOpenAIGatewayHandlerImages_OAuthBadRequestPassesThroughUpstreamImageError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(3131)
	accounts := []service.Account{{
		ID:          1,
		Name:        "image-account-1",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 0,
		Priority:    0,
		Credentials: map[string]any{"access_token": "token-1"},
	}}
	upstream := &openAIImagesHandlerHTTPUpstream{resp: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"req_img_bad_size"},
		},
		Body: io.NopCloser(bytes.NewBufferString(`{"error":{"message":"Invalid value for 'size': expected one of 1024x1024, 1536x1024.","type":"invalid_request_error","param":"size","code":"unknown_parameter"}}`)),
	}}
	accountRepo := openAIImagesHandlerAccountRepo{accounts: accounts}
	cfg := &config.Config{RunMode: config.RunModeSimple}
	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		cfg,
		nil,
		nil,
		nil,
		nil,
		nil,
		upstream,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	billingService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingService.Stop)
	h := NewOpenAIGatewayHandler(
		gatewayService,
		service.NewConcurrencyService(nil),
		billingService,
		service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, cfg),
		nil,
		nil,
		nil,
		nil,
		cfg,
	)

	body := []byte(`{"model":"gpt-image-2","prompt":"draw a cat","size":"bad-size"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		ID:      100,
		GroupID: &groupID,
		Group: &service.Group{
			ID:                   groupID,
			AllowImageGeneration: true,
		},
		User: &service.User{ID: 101},
	})
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 101, Concurrency: 0})

	h.Images(c)

	require.Equal(t, []int64{1}, upstream.calls())
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "req_img_bad_size", rec.Header().Get("x-request-id"))
	require.Equal(t, "invalid_request_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
	require.Equal(t, "unknown_parameter", gjson.GetBytes(rec.Body.Bytes(), "error.code").String())
	require.Equal(t, "size", gjson.GetBytes(rec.Body.Bytes(), "error.param").String())
	require.Contains(t, gjson.GetBytes(rec.Body.Bytes(), "error.message").String(), "Invalid value for 'size'")
}

func TestOpenAIGatewayHandlerImages_DisabledGroupRejectsBeforeScheduling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"model":"gpt-image-2","prompt":"draw","size":"1024x1024"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	groupID := int64(111)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		ID:      222,
		GroupID: &groupID,
		Group: &service.Group{
			ID:                   groupID,
			AllowImageGeneration: false,
		},
		User: &service.User{ID: 333},
	})
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 333, Concurrency: 1})

	h := &OpenAIGatewayHandler{
		gatewayService:      &service.OpenAIGatewayService{},
		billingCacheService: &service.BillingCacheService{},
		apiKeyService:       &service.APIKeyService{},
		concurrencyHelper:   &ConcurrencyHelper{concurrencyService: &service.ConcurrencyService{}},
	}

	h.Images(c)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, "permission_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
	require.Contains(t, rec.Body.String(), service.ImageGenerationPermissionMessage())
}
