package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type grokMediaSpoolSwitchReadCloser struct {
	io.Reader
	onEOF func()
}

func (r *grokMediaSpoolSwitchReadCloser) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err == io.EOF && r.onEOF != nil {
		r.onEOF()
		r.onEOF = nil
	}
	return n, err
}

func (r *grokMediaSpoolSwitchReadCloser) Close() error { return nil }

type grokMediaRequestRecorder struct {
	service.HTTPUpstream
	mu           sync.Mutex
	bodies       [][]byte
	contentTypes []string
	hashes       []string
	accountIDs   []int64
	methods      []string
	paths        []string
	statuses     []int
	cancel       context.CancelFunc
}

type grokVideoClaimCache struct {
	openAIChatCompletionsGatewayCacheStub
	mu      sync.Mutex
	claimed map[string]bool
}

type grokVideoBillingErrorRepo struct{ service.UsageBillingRepository }

func (grokVideoBillingErrorRepo) Apply(context.Context, *service.UsageBillingCommand) (*service.UsageBillingApplyResult, error) {
	return nil, errors.New("billing write failed")
}

func (c *grokVideoClaimCache) ClaimGrokVideoBilled(_ context.Context, key string, _ time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.claimed[key] {
		return false, nil
	}
	c.claimed[key] = true
	return true, nil
}

func (c *grokVideoClaimCache) ReleaseGrokVideoBilled(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.claimed, key)
	return nil
}

func TestPrepareGrokVideoCompletionBilling_ClaimsOnceAcrossDuplicatePolls(t *testing.T) {
	cache := &grokVideoClaimCache{claimed: make(map[string]bool)}
	group := &service.Group{ID: 4041, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 4042, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "video-key", "base_url": "https://api.x.ai/v1"}}
	env := newTerminalUsageOpenAIEnvWithUpstreamAndGatewayCache(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account}}}, &grokMediaRequestRecorder{}, cache)
	status := &service.OpenAIForwardResult{ResponseID: "video-claim-1", Model: "grok-imagine-video", VideoCount: 1, VideoDurationSeconds: 8}
	subject := middleware.AuthSubject{UserID: env.apiKey.UserID}

	first := prepareGrokVideoCompletionBilling(context.Background(), env.handler, zap.NewNop(), env.apiKey, subject, status.ResponseID, status)
	require.NotNil(t, first)
	require.Equal(t, 1, first.VideoCount)
	require.Nil(t, prepareGrokVideoCompletionBilling(context.Background(), env.handler, zap.NewNop(), env.apiKey, subject, status.ResponseID, status))
}

func TestRecordGrokMediaUsage_ReleasesVideoClaimAfterBillingError(t *testing.T) {
	cache := &grokVideoClaimCache{claimed: make(map[string]bool)}
	group := &service.Group{ID: 4051, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{ID: 4052, Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "video-key", "base_url": "https://api.x.ai/v1"}}
	repo := &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account}}}
	cfg := &config.Config{Default: config.DefaultConfig{RateMultiplier: 1}}
	billingCache := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCache.Stop)
	concurrency := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	gateway := service.NewOpenAIGatewayService(
		repo, &openAIChatCompletionsUsageLogRepoStub{}, grokVideoBillingErrorRepo{}, nil, nil, nil,
		cache, cfg, nil, concurrency, service.NewBillingService(cfg, nil), nil, billingCache,
		&grokMediaRequestRecorder{}, service.NewDeferredService(repo, nil, 0), nil, service.NewGrokTokenProvider(repo, nil),
	)
	h := NewOpenAIGatewayHandler(gateway, concurrency, billingCache, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)
	apiKey := &service.APIKey{ID: 4053, UserID: 4054, Status: service.StatusActive, GroupID: &group.ID, Group: group, User: &service.User{ID: 4054, Status: service.StatusActive, Concurrency: 1}}
	status := &service.OpenAIForwardResult{ResponseID: "video-claim-retry", Model: "grok-imagine-video", VideoCount: 1, VideoDurationSeconds: 8}
	subject := middleware.AuthSubject{UserID: apiKey.UserID}
	prepared := prepareGrokVideoCompletionBilling(context.Background(), h, zap.NewNop(), apiKey, subject, status.ResponseID, status)
	require.NotNil(t, prepared)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/video-claim-retry", nil)
	recordGrokMediaUsage(c, h, zap.NewNop(), apiKey, subject, nil, account, prepared, "grok-imagine-video", "payload", status.ResponseID)

	reclaimed := prepareGrokVideoCompletionBilling(context.Background(), h, zap.NewNop(), apiKey, subject, status.ResponseID, status)
	require.NotNil(t, reclaimed, "billing errors must release the video claim for a later poll")
}

func TestGrokImagesCompositeMultipartUsesResolvedUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 970, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	account := &service.Account{
		ID:          971,
		Name:        "composite-grok-image",
		Platform:    service.PlatformGrok,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":       "key",
			"base_url":      "https://api.x.ai/v1",
			"model_mapping": map[string]any{"grok-imagine-image-quality": "grok-upstream"},
		},
	}
	upstream := &grokMediaRequestRecorder{}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account}}}, upstream)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "image-alias"))
	require.NoError(t, writer.WriteField("prompt", "draw"))
	image, err := writer.CreateFormFile("image", "source.png")
	require.NoError(t, err)
	_, err = image.Write([]byte("image"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	ctx := service.WithCompositeRouteDecision(context.Background(), service.CompositeRouteDecision{
		Matched:        true,
		GroupID:        group.ID,
		PublicModel:    "image-alias",
		TargetPlatform: service.PlatformGrok,
		UpstreamModel:  "grok-imagine-image-quality",
		Endpoint:       service.CompositeRouteEndpointImages,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes())).WithContext(ctx)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	env.router("/v1/images/edits", env.handler.GrokImages).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Len(t, upstream.bodies, 1)
	require.Equal(t, "grok-upstream", gjson.GetBytes(upstream.bodies[0], "model").String())
}

func (u *grokMediaRequestRecorder) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		_ = req.Body.Close()
	}
	sum := sha256.Sum256(body)
	u.mu.Lock()
	u.bodies = append(u.bodies, body)
	u.contentTypes = append(u.contentTypes, req.Header.Get("Content-Type"))
	u.hashes = append(u.hashes, hex.EncodeToString(sum[:]))
	u.accountIDs = append(u.accountIDs, accountID)
	u.methods = append(u.methods, req.Method)
	u.paths = append(u.paths, req.URL.Path)
	call := len(u.bodies)
	status := http.StatusOK
	if call <= len(u.statuses) {
		status = u.statuses[call-1]
	}
	u.mu.Unlock()
	if u.cancel != nil {
		u.cancel()
		return nil, errors.New("request canceled")
	}
	responseBody := `{"id":"req_123","status":"completed"}`
	if strings.Contains(req.URL.Path, "/images/") {
		responseBody = `{"data":[{"url":"https://images.test/result.png"}]}`
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(responseBody))}, nil
}

func (u *grokMediaRequestRecorder) assert(t *testing.T, wantAccounts []int64, wantMethod, wantPath, wantContentType string, wantBody []byte) {
	t.Helper()
	u.mu.Lock()
	defer u.mu.Unlock()
	require.Equal(t, wantAccounts, u.accountIDs)
	require.Len(t, u.bodies, len(wantAccounts))
	wantHash := sha256.Sum256(wantBody)
	wantHashString := hex.EncodeToString(wantHash[:])
	for i, body := range u.bodies {
		require.Equalf(t, wantBody, body, "attempt %d body", i+1)
		require.Equalf(t, wantHashString, u.hashes[i], "attempt %d body hash", i+1)
		require.Equalf(t, wantMethod, u.methods[i], "attempt %d method", i+1)
		require.Equalf(t, wantPath, u.paths[i], "attempt %d path", i+1)
		require.Equalf(t, wantContentType, u.contentTypes[i], "attempt %d content type", i+1)
	}
}

type grokMediaEligibilityProberStub struct {
	eligible bool
	reason   string
	err      error
	calls    int
}

type grokVideoOwnerBindingCache struct {
	service.GatewayCache
	ownerID int64
	calls   int
}

func (c *grokVideoOwnerBindingCache) GetSessionAccountID(context.Context, int64, string) (int64, error) {
	c.calls++
	if c.calls == 1 {
		return c.ownerID, nil
	}
	return 0, nil
}

func (*grokVideoOwnerBindingCache) SetSessionAccountID(context.Context, int64, string, int64, time.Duration) error {
	return nil
}

func (*grokVideoOwnerBindingCache) RefreshSessionTTL(context.Context, int64, string, time.Duration) error {
	return nil
}

func (*grokVideoOwnerBindingCache) DeleteSessionAccountID(context.Context, int64, string) error {
	return nil
}

func (s *grokMediaEligibilityProberStub) ProbeMediaEligibility(context.Context, int64) (bool, string, error) {
	s.calls++
	return s.eligible, s.reason, s.err
}

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
			name:     "video generation defers usage until status",
			endpoint: service.GrokMediaEndpointVideosGenerations,
			model:    "grok-imagine-video-1.5",
			want:     false,
		},
		{
			name:     "video status skips immediate helper (status path claims separately)",
			endpoint: service.GrokMediaEndpointVideoStatus,
			model:    "",
			want:     false,
		},
		{
			name:     "video content skips usage",
			endpoint: service.GrokMediaEndpointVideoContent,
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
			// Nil result must never bill.
			require.False(t, shouldRecordGrokMediaUsage(tt.endpoint, tt.model, nil))
			// Immediate helper only bills image generation (async video bills on status).
			result := &service.OpenAIForwardResult{ImageCount: 1, VideoCount: 0}
			if tt.endpoint.IsGenerationRequest() && !isGrokVideoCreateEndpoint(tt.endpoint) && strings.TrimSpace(tt.model) != "" {
				require.Equal(t, tt.want, shouldRecordGrokMediaUsage(tt.endpoint, tt.model, result))
			} else {
				require.False(t, shouldRecordGrokMediaUsage(tt.endpoint, tt.model, result))
			}
			// Zero billable units never bill even for generation + model.
			empty := &service.OpenAIForwardResult{}
			require.False(t, shouldRecordGrokMediaUsage(tt.endpoint, tt.model, empty))
		})
	}
}

func TestGrokVideoStatus_UsesNoRequestBodyHandle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	missingRawDir := filepath.Join(t.TempDir(), "missing")
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: missingRawDir, FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
	group := &service.Group{ID: 906, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	parentID := int64(908)
	account := &service.Account{ID: 907, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}
	parent := &service.Account{ID: parentID, Name: "parent", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Credentials: map[string]any{"access_token": "token"}}
	upstream := &openAIImagesHandlerHTTPUpstream{resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"req_123","status":"completed"}`))}}
	env := newTerminalUsageOpenAIEnvWithUpstreamAndGatewayCache(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account, parent}}}, upstream, &grokVideoOwnerBindingCache{ownerID: account.ID})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/videos/req_123", nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), env.apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: env.apiKey.UserID, Concurrency: env.apiKey.User.Concurrency})
	})
	router.GET("/videos/:request_id", env.handler.GrokVideoStatus)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, []int64{account.ID}, upstream.calls())
	require.NoDirExists(t, missingRawDir)
}

func TestGrokVideoStatus_RejectsSchedulerAccountOtherThanOwnerBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 910, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	parentID := int64(913)
	selected := &service.Account{ID: 911, Name: "selected", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Priority: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}
	owner := &service.Account{ID: 912, Name: "owner", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Priority: 2, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}
	parent := &service.Account{ID: parentID, Name: "parent", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Credentials: map[string]any{"access_token": "token"}}
	cache := &grokVideoOwnerBindingCache{ownerID: owner.ID}
	upstream := &grokMediaRequestRecorder{}
	cfg := &config.Config{RunMode: config.RunModeSimple, Default: config.DefaultConfig{RateMultiplier: 1}, Gateway: config.GatewayConfig{Scheduling: config.GatewaySchedulingConfig{LoadBatchEnabled: false}}, Concurrency: config.ConcurrencyConfig{PingInterval: 0}}
	billing := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billing.Stop)
	concurrency := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	repo := &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{selected, owner, parent}}}
	gateway := service.NewOpenAIGatewayService(repo, nil, nil, nil, nil, nil, cache, cfg, nil, concurrency, nil, nil, billing, upstream, nil, nil, service.NewGrokTokenProvider(repo, nil), nil, nil)
	h := NewOpenAIGatewayHandler(gateway, concurrency, billing, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{ID: 101, UserID: 202, Status: service.StatusActive, GroupID: &group.ID, Group: group, User: &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1}})
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 202, Concurrency: 1})
	})
	router.GET("/videos/:request_id", h.GrokVideoStatus)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/videos/req_123", nil))

	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	require.Empty(t, upstream.accountIDs)
}

func TestGrokMedia_GenerateEditVideoRejectUpstreamFailoverPreserveRequestSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	grokHandler := func(c *gin.Context) *OpenAIGatewayHandler {
		handler, ok := c.MustGet("handler").(*OpenAIGatewayHandler)
		require.True(t, ok)
		return handler
	}
	for _, tt := range []struct {
		name         string
		route        string
		handler      gin.HandlerFunc
		body         func(t *testing.T) ([]byte, string)
		accounts     func(parentID int64) []*service.Account
		statuses     []int
		reject       bool
		cancel       bool
		wantStatus   int
		wantAccounts []int64
		wantMethod   string
		wantPath     string
		wantType     string
		wantBody     []byte
	}{
		{
			name: "generate success", route: "/v1/images/generations", handler: func(c *gin.Context) {
				handler, ok := c.MustGet("handler").(*OpenAIGatewayHandler)
				require.True(t, ok)
				handler.GrokImages(c)
			},
			body: func(t *testing.T) ([]byte, string) {
				return []byte(`{"model":"grok-imagine","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`), "application/json"
			},
			accounts: func(parentID int64) []*service.Account {
				return []*service.Account{{ID: 1001, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}}
			}, wantStatus: http.StatusOK, wantAccounts: []int64{1001}, wantMethod: http.MethodPost, wantPath: "/v1/images/generations", wantType: "application/json", wantBody: []byte(`{"model":"grok-imagine-image-quality","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`),
		},
		{
			name: "edit success", route: "/v1/images/edits", handler: func(c *gin.Context) {
				handler, ok := c.MustGet("handler").(*OpenAIGatewayHandler)
				require.True(t, ok)
				handler.GrokImages(c)
			},
			body: func(t *testing.T) ([]byte, string) {
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				require.NoError(t, writer.WriteField("model", "grok-imagine"))
				file, err := writer.CreateFormFile("image", "source.png")
				require.NoError(t, err)
				_, err = file.Write([]byte("media-secret-file"))
				require.NoError(t, err)
				require.NoError(t, writer.Close())
				return body.Bytes(), writer.FormDataContentType()
			},
			accounts: func(parentID int64) []*service.Account {
				return []*service.Account{{ID: 1002, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}}
			}, wantStatus: http.StatusOK, wantAccounts: []int64{1002}, wantMethod: http.MethodPost, wantPath: "/v1/images/edits", wantType: "application/json", wantBody: []byte(`{"image":{"type":"image_url","url":"data:application/octet-stream;base64,bWVkaWEtc2VjcmV0LWZpbGU="},"model":"grok-imagine-image-quality"}`),
		},
		{
			name: "video create success", route: "/v1/videos/generations", handler: func(c *gin.Context) { grokHandler(c).GrokVideoGeneration(c) },
			body: func(t *testing.T) ([]byte, string) {
				return []byte(`{"model":"grok-imagine-video-1.5","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`), "application/json"
			},
			accounts: func(parentID int64) []*service.Account {
				return []*service.Account{{ID: 1003, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}}
			}, wantStatus: http.StatusOK, wantAccounts: []int64{1003}, wantMethod: http.MethodPost, wantPath: "/v1/videos/generations", wantType: "application/json", wantBody: []byte(`{"model":"grok-imagine-video-1.5","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`),
		},
		{
			name: "edit upstream 4xx", route: "/v1/images/edits", handler: func(c *gin.Context) { grokHandler(c).GrokImages(c) },
			body: func(t *testing.T) ([]byte, string) {
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				require.NoError(t, writer.WriteField("model", "grok-imagine"))
				file, err := writer.CreateFormFile("image", "source.png")
				require.NoError(t, err)
				_, err = file.Write([]byte("media-secret-file"))
				require.NoError(t, err)
				require.NoError(t, writer.Close())
				return body.Bytes(), writer.FormDataContentType()
			},
			accounts: func(parentID int64) []*service.Account {
				return []*service.Account{{ID: 10021, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}}
			}, statuses: []int{http.StatusBadRequest}, wantStatus: http.StatusBadRequest, wantAccounts: []int64{10021}, wantMethod: http.MethodPost, wantPath: "/v1/images/edits", wantType: "application/json", wantBody: []byte(`{"image":{"type":"image_url","url":"data:application/octet-stream;base64,bWVkaWEtc2VjcmV0LWZpbGU="},"model":"grok-imagine-image-quality"}`),
		},
		{
			name: "video create canceled", route: "/v1/videos/generations", handler: func(c *gin.Context) { grokHandler(c).GrokVideoGeneration(c) },
			body: func(t *testing.T) ([]byte, string) {
				return []byte(`{"model":"grok-imagine-video-1.5","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`), "application/json"
			},
			accounts: func(parentID int64) []*service.Account {
				return []*service.Account{{ID: 10031, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}}
			}, cancel: true, wantAccounts: []int64{10031}, wantMethod: http.MethodPost, wantPath: "/v1/videos/generations", wantType: "application/json", wantBody: []byte(`{"model":"grok-imagine-video-1.5","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`),
		},
		{
			name: "video status has no body", route: "/videos/:request_id", handler: func(c *gin.Context) { grokHandler(c).GrokVideoStatus(c) },
			body: func(t *testing.T) ([]byte, string) { return nil, "" },
			accounts: func(parentID int64) []*service.Account {
				return []*service.Account{{ID: 1004, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}}
			}, wantStatus: http.StatusOK, wantAccounts: []int64{1004}, wantMethod: http.MethodGet, wantPath: "/v1/videos/req_123", wantType: "", wantBody: nil,
		},
		{
			name: "permission reject", route: "/v1/images/generations", handler: func(c *gin.Context) { grokHandler(c).GrokImages(c) },
			body: func(t *testing.T) ([]byte, string) {
				return []byte(`{"model":"grok-imagine","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`), "application/json"
			},
			accounts: func(parentID int64) []*service.Account {
				return []*service.Account{{ID: 1005, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}}
			}, reject: true, wantStatus: http.StatusForbidden, wantMethod: http.MethodPost,
		},
		{
			name: "upstream 4xx", route: "/v1/images/generations", handler: func(c *gin.Context) { grokHandler(c).GrokImages(c) },
			body: func(t *testing.T) ([]byte, string) {
				return []byte(`{"model":"grok-imagine","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`), "application/json"
			},
			accounts: func(parentID int64) []*service.Account {
				return []*service.Account{{ID: 1006, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}}
			}, statuses: []int{http.StatusBadRequest}, wantStatus: http.StatusBadRequest, wantAccounts: []int64{1006}, wantMethod: http.MethodPost, wantPath: "/v1/images/generations", wantType: "application/json", wantBody: []byte(`{"model":"grok-imagine-image-quality","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`),
		},
		{
			name: "same account retry", route: "/v1/images/generations", handler: func(c *gin.Context) { grokHandler(c).GrokImages(c) },
			body: func(t *testing.T) ([]byte, string) {
				return []byte(`{"model":"grok-imagine","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`), "application/json"
			},
			accounts: func(parentID int64) []*service.Account {
				return []*service.Account{{ID: 10061, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"api_key": "grok-key", "base_url": "https://api.x.ai/v1", "pool_mode": true}}}
			}, statuses: []int{http.StatusTooManyRequests, http.StatusOK}, wantStatus: http.StatusOK, wantAccounts: []int64{10061, 10061}, wantMethod: http.MethodPost, wantPath: "/v1/images/generations", wantType: "application/json", wantBody: []byte(`{"model":"grok-imagine-image-quality","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`),
		},
		{
			name: "upstream 5xx fails over", route: "/v1/images/generations", handler: func(c *gin.Context) { grokHandler(c).GrokImages(c) },
			body: func(t *testing.T) ([]byte, string) {
				return []byte(`{"model":"grok-imagine","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`), "application/json"
			},
			accounts: func(parentID int64) []*service.Account {
				return []*service.Account{{ID: 1007, Name: "first", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}, {ID: 1008, Name: "second", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}}
			}, statuses: []int{http.StatusInternalServerError, http.StatusOK}, wantStatus: http.StatusOK, wantAccounts: []int64{1007, 1008}, wantMethod: http.MethodPost, wantPath: "/v1/images/generations", wantType: "application/json", wantBody: []byte(`{"model":"grok-imagine-image-quality","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`),
		},
		{
			name: "canceled request", route: "/v1/images/generations", handler: func(c *gin.Context) { grokHandler(c).GrokImages(c) },
			body: func(t *testing.T) ([]byte, string) {
				return []byte(`{"model":"grok-imagine","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`), "application/json"
			},
			accounts: func(parentID int64) []*service.Account {
				return []*service.Account{{ID: 1009, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}}
			}, cancel: true, wantAccounts: []int64{1009}, wantMethod: http.MethodPost, wantPath: "/v1/images/generations", wantType: "application/json", wantBody: []byte(`{"model":"grok-imagine-image-quality","prompt":"metadata","image_url":"data:image/png;base64,bWVkaWEtc2VjcmV0"}`),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rawDir, formDir := t.TempDir(), t.TempDir()
			oldOptions := jsonRequestBodyHandleOptions
			jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, PreviewLimitBytes: 64, TempDir: rawDir, FilePrefix: "sub2api-test-"}
			t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
			t.Setenv("TMP", formDir)
			t.Setenv("TEMP", formDir)

			parentID := int64(2000)
			accounts := tt.accounts(parentID)
			parent := &service.Account{ID: parentID, Name: "parent", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Credentials: map[string]any{"access_token": "token"}}
			recorder := &grokMediaRequestRecorder{statuses: tt.statuses}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancel {
				recorder.cancel = cancel
			}
			group := &service.Group{ID: 999, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: !tt.reject}
			cache := service.GatewayCache(openAIChatCompletionsGatewayCacheStub{})
			if tt.wantMethod == http.MethodGet {
				cache = &grokVideoOwnerBindingCache{ownerID: accounts[0].ID}
			}
			env := newTerminalUsageOpenAIEnvWithUpstreamAndGatewayCache(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: append(accounts, parent)}}, recorder, cache)
			body, contentType := tt.body(t)
			var requestContext *gin.Context
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("handler", env.handler)
				c.Set(string(middleware.ContextKeyAPIKey), env.apiKey)
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: env.apiKey.UserID, Concurrency: env.apiKey.User.Concurrency})
				requestContext = c
				c.Next()
			})
			router.Use(middleware.UsageDetailCapture())
			if tt.wantMethod == http.MethodGet {
				router.GET(tt.route, tt.handler)
			} else {
				router.POST(tt.route, tt.handler)
			}
			req := httptest.NewRequest(tt.wantMethod, strings.ReplaceAll(tt.route, ":request_id", "req_123"), bytes.NewReader(body)).WithContext(ctx)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if tt.cancel {
				require.ErrorIs(t, ctx.Err(), context.Canceled)
			} else {
				require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			}
			recorder.assert(t, tt.wantAccounts, tt.wantMethod, tt.wantPath, tt.wantType, tt.wantBody)
			if tt.name == "video status has no body" {
				require.Empty(t, recorder.bodies[0])
			}
			detail := middleware.GetUsageDetailSnapshot(requestContext)
			require.NotNil(t, detail)
			ops, ok := requestContext.Get(service.OpsUpstreamRequestBodyKey)
			require.True(t, ok)
			opsBody, ok := ops.(string)
			require.True(t, ok)
			for _, snapshot := range []string{detail.RequestBody, detail.UpstreamRequestBody, opsBody} {
				for _, sentinel := range []string{"media-secret", "bWVkaWEtc2VjcmV0", "media-secret-file"} {
					require.NotContains(t, snapshot, sentinel)
				}
			}
			require.Empty(t, readTestDir(t, rawDir))
			require.Empty(t, readTestDir(t, formDir))
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
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(upstream.release) }) }
	t.Cleanup(func() {
		release()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("timed out waiting for grok media handler cleanup")
		}
	})
	go func() { router.ServeHTTP(recorder, req); close(done) }()

	select {
	case <-upstream.started:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for grok media upstream")
	}
	require.Equal(t, body.Bytes(), upstream.body)
	require.NotEmpty(t, readTestDir(t, rawDir), "raw body must spool while upstream is blocked")
	detail := middleware.GetUsageDetailSnapshot(requestContext)
	ops, ok := requestContext.Get(service.OpsUpstreamRequestBodyKey)
	require.True(t, ok)
	opsBody, ok := ops.(string)
	require.True(t, ok)
	for _, snapshot := range []string{detail.RequestBody, detail.UpstreamRequestBody, opsBody} {
		require.NotContains(t, snapshot, "source-secret-")
		require.NotContains(t, snapshot, "mask-secret")
	}

	release()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for grok media handler")
	}
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Empty(t, readTestDir(t, rawDir))
}

func TestGrokMedia_SessionSeedReleasedBeforeBlockedUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	grokHandler := func(c *gin.Context) *OpenAIGatewayHandler {
		handler, ok := c.MustGet("handler").(*OpenAIGatewayHandler)
		require.True(t, ok)
		return handler
	}
	for _, tt := range []struct {
		name, route string
		handler     gin.HandlerFunc
		body        func() ([]byte, string)
	}{
		{"generate", "/v1/images/generations", func(c *gin.Context) { grokHandler(c).GrokImages(c) }, func() ([]byte, string) {
			return []byte(`{"model":"grok-imagine","prompt":"` + strings.Repeat("x", 20<<20) + `"}`), "application/json"
		}},
		{"edit", "/v1/images/edits", func(c *gin.Context) { grokHandler(c).GrokImages(c) }, func() ([]byte, string) {
			var b bytes.Buffer
			w := multipart.NewWriter(&b)
			require.NoError(t, w.WriteField("model", "grok-imagine"))
			require.NoError(t, w.WriteField("prompt", strings.Repeat("x", 20<<20)))
			require.NoError(t, w.WriteField("image_url", "https://example.com/source.png"))
			require.NoError(t, w.Close())
			return b.Bytes(), w.FormDataContentType()
		}},
		{"video", "/v1/videos/generations", func(c *gin.Context) { grokHandler(c).GrokVideoGeneration(c) }, func() ([]byte, string) {
			return []byte(`{"model":"grok-imagine-video-1.5","prompt":"` + strings.Repeat("x", 20<<20) + `"}`), "application/json"
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rawDir := t.TempDir()
			oldOptions := jsonRequestBodyHandleOptions
			jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, PreviewLimitBytes: 64, TempDir: rawDir, FilePrefix: "sub2api-test-"}
			t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
			runtime.GC()
			var before runtime.MemStats
			runtime.ReadMemStats(&before)
			body, contentType := tt.body()
			req := httptest.NewRequest(http.MethodPost, tt.route, &releaseAfterEOFBody{data: body})
			req.Header.Set("Content-Type", contentType)
			upstream := &openAIImagesHashingUpstream{started: make(chan struct{}), release: make(chan struct{})}
			group := &service.Group{ID: 960, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
			parentID := int64(962)
			account := &service.Account{ID: 961, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, ParentAccountID: &parentID, Credentials: map[string]any{"api_key": "key", "base_url": "https://api.x.ai/v1"}}
			parent := &service.Account{ID: parentID, Name: "parent", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Credentials: map[string]any{"access_token": "token"}}
			env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account, parent}}}, upstream)
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("handler", env.handler)
				c.Set(string(middleware.ContextKeyAPIKey), env.apiKey)
				c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: env.apiKey.UserID, Concurrency: env.apiKey.User.Concurrency})
			})
			router.POST(tt.route, tt.handler)
			recorder := httptest.NewRecorder()
			done := make(chan struct{})
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(upstream.release) }) }
			t.Cleanup(func() {
				release()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Error("timed out waiting for Grok handler cleanup")
				}
			})
			go func() { router.ServeHTTP(recorder, req); close(done) }()
			select {
			case <-upstream.started:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for Grok upstream")
			}
			require.Positive(t, upstream.size)
			runtime.GC()
			runtime.GC()
			var after runtime.MemStats
			runtime.ReadMemStats(&after)
			require.LessOrEqual(t, after.HeapAlloc, before.HeapAlloc+uint64(12<<20), "blocked Grok request retained 20MB text")
			release()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for Grok handler")
			}
			require.Empty(t, readTestDir(t, rawDir))
		})
	}
}

func TestGrokMedia_MultipartTextPartLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, size := range []int{10 << 20, 20 << 20, (20 << 20) + 1} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			rawDir, formDir := t.TempDir(), t.TempDir()
			oldOptions := jsonRequestBodyHandleOptions
			jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, PreviewLimitBytes: 64, TempDir: rawDir, FilePrefix: "sub2api-test-"}
			t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
			t.Setenv("TMP", formDir)
			t.Setenv("TEMP", formDir)

			const textStart, textMiddle = "multipart-text-secret-start-", "multipart-text-secret-middle-"
			text := []byte(textStart + strings.Repeat("x", size-len(textStart)-len(textMiddle)) + textMiddle)
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("model", "grok-imagine"))
			require.NoError(t, writer.WriteField("prompt", string(text)))
			require.NoError(t, writer.Close())

			upstream := &openAIImagesSpoolUpstream{started: make(chan struct{}), release: make(chan struct{})}
			group := &service.Group{ID: 932, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
			parentID := int64(934)
			account := &service.Account{ID: 933, Name: "grok-media", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}
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

			if size > 20<<20 {
				router.ServeHTTP(recorder, req)
				require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code, recorder.Body.String())
				select {
				case <-upstream.started:
					t.Fatal("oversized multipart text part reached upstream")
				default:
				}
			} else {
				done := make(chan struct{})
				go func() { router.ServeHTTP(recorder, req); close(done) }()
				select {
				case <-upstream.started:
				case <-time.After(5 * time.Second):
					t.Fatal("timed out waiting for grok media upstream")
				}
				requireMultipartTextPart(t, upstream.body, upstream.contentType, "prompt", text)
				require.Empty(t, requestContext.Request.MultipartForm.Value)
				detail := middleware.GetUsageDetailSnapshot(requestContext)
				ops, ok := requestContext.Get(service.OpsUpstreamRequestBodyKey)
				require.True(t, ok)
				opsBody, ok := ops.(string)
				require.True(t, ok)
				for _, snapshot := range []string{detail.RequestBody, detail.UpstreamRequestBody, opsBody} {
					require.NotContains(t, snapshot, textStart)
					require.NotContains(t, snapshot, textMiddle)
				}
				assertMatrixRequestBodySnapshot(t, "multipart text usage upstream snapshot", detail.UpstreamRequestBody, upstream.body, "")
				assertMatrixRequestBodySnapshot(t, "multipart text ops upstream snapshot", opsBody, upstream.body, "")
				close(upstream.release)
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Fatal("timed out waiting for grok media handler")
				}
				require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			}
			require.Empty(t, readTestDir(t, rawDir))
			require.Empty(t, readTestDir(t, formDir))
		})
	}
}

func TestGrokMedia_MultipartEditTextSourcesRebuildUpstreamJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name, sourceField, source, maskField, mask string
	}{
		{"legacy aliases", "image", "https://example.com/source.png", "mask", "data:image/png;base64,bWFzaw=="},
		{"url aliases", "image_url", "data:image/png;base64,c291cmNl", "mask_image_url", "https://example.com/mask.png"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("model", "grok-imagine"))
			require.NoError(t, writer.WriteField("prompt", "replace background"))
			require.NoError(t, writer.WriteField(tt.sourceField, tt.source))
			require.NoError(t, writer.WriteField(tt.maskField, tt.mask))
			require.NoError(t, writer.Close())

			upstream := &openAIImagesSpoolUpstream{started: make(chan struct{}), release: make(chan struct{})}
			group := &service.Group{ID: 913, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
			parentID := int64(915)
			account := &service.Account{ID: 914, Name: "grok-media", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}
			parent := &service.Account{ID: parentID, Name: "grok-parent", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Credentials: map[string]any{"access_token": "grok-token"}}
			env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account, parent}}}, upstream)

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
			req.Header.Set("Content-Type", writer.FormDataContentType())
			done := make(chan struct{})
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(upstream.release) }) }
			t.Cleanup(func() {
				release()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Error("timed out waiting for grok media handler cleanup")
				}
			})
			go func() { env.router("/v1/images/edits", env.handler.GrokImages).ServeHTTP(recorder, req); close(done) }()

			select {
			case <-upstream.started:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for grok media upstream")
			}
			require.Equal(t, tt.source, gjson.GetBytes(upstream.body, "image.url").String())
			require.Equal(t, tt.mask, gjson.GetBytes(upstream.body, "mask.url").String())

			release()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for grok media handler")
			}
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		})
	}
}

func TestGrokMedia_MultipartEffectiveSpoolFailureReturns503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldOptions := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, TempDir: t.TempDir(), FilePrefix: "sub2api-test-"}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "grok-imagine"))
	require.NoError(t, writer.WriteField("image_url", "https://example.com/source.png"))
	require.NoError(t, writer.Close())

	group := &service.Group{ID: 916, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	parentID := int64(918)
	account := &service.Account{ID: 917, Name: "grok", Platform: service.PlatformGrok, Type: service.AccountTypeOAuth, Status: service.StatusActive, Schedulable: true, Concurrency: 1, ParentAccountID: &parentID, Credentials: map[string]any{"base_url": "https://api.x.ai/v1"}}
	parent := &service.Account{ID: parentID, Name: "parent", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive, Credentials: map[string]any{"access_token": "token"}}
	upstream := &openAIImagesHandlerHTTPUpstream{}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account, parent}}}, upstream)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", nil)
	req.Body = &grokMediaSpoolSwitchReadCloser{
		Reader: bytes.NewReader(body.Bytes()),
		onEOF: func() {
			jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: filepath.Join(t.TempDir(), "missing"), FilePrefix: "sub2api-test-"}
		},
	}
	req.ContentLength = int64(body.Len())
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	env.router("/v1/images/edits", env.handler.GrokImages).ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	require.Empty(t, upstream.calls())
}

func TestGrokMedia_TransportSpoolFailureReturns503WithoutUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 919, Platform: service.PlatformGrok, Status: service.StatusActive, Hydrated: true, AllowImageGeneration: true}
	account := &service.Account{
		ID: 920, Name: "grok-spool", Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1,
		Credentials: map[string]any{"api_key": "key", "base_url": "https://api.x.ai/v1"},
	}
	upstream := &responsesSpoolTransportUpstream{}
	env := newTerminalUsageOpenAIEnvWithUpstream(t, group, &terminalUsageGrokAccountRepo{openAIRetryAccountRepoStub{accounts: []*service.Account{account}}}, upstream)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"grok-imagine-image","prompt":"draw"}`))
	req.Header.Set("Content-Type", "application/json")

	env.router("/v1/images/generations", env.handler.GrokImages).ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	require.Equal(t, "api_error", gjson.Get(rec.Body.String(), "error.type").String())
	require.Equal(t, "Failed to spool request body", gjson.Get(rec.Body.String(), "error.message").String())
	require.Equal(t, 1, upstream.calls)
	require.Nil(t, env.usageRepo.lastLog)
}

func TestGrokMediaRequiredCapability(t *testing.T) {
	tests := []struct {
		name     string
		endpoint service.GrokMediaEndpoint
		want     service.OpenAIEndpointCapability
	}{
		{name: "image generation", endpoint: service.GrokMediaEndpointImagesGenerations, want: service.OpenAIEndpointCapabilityGrokMediaGeneration},
		{name: "image edit", endpoint: service.GrokMediaEndpointImagesEdits, want: service.OpenAIEndpointCapabilityGrokMediaGeneration},
		{name: "video generation", endpoint: service.GrokMediaEndpointVideosGenerations, want: service.OpenAIEndpointCapabilityGrokMediaGeneration},
		{name: "video edit", endpoint: service.GrokMediaEndpointVideosEdits, want: service.OpenAIEndpointCapabilityGrokMediaGeneration},
		{name: "video extension", endpoint: service.GrokMediaEndpointVideosExtensions, want: service.OpenAIEndpointCapabilityGrokMediaGeneration},
		{name: "video status preserves lookup", endpoint: service.GrokMediaEndpointVideoStatus, want: ""},
		{name: "video content preserves lookup", endpoint: service.GrokMediaEndpointVideoContent, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, grokMediaRequiredCapability(tt.endpoint))
		})
	}
}

func TestGrokMediaScheduleModelUsesNormalizedMappedUpstream(t *testing.T) {
	account := &service.Account{
		Platform: service.PlatformGrok,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"grok-imagine-video-1.5": "wrong-raw-model",
				"grok-imagine-video":     "mapped-video-model",
			},
		},
	}

	require.Equal(t, "mapped-video-model", grokMediaScheduleModel(account, "grok-imagine-video", nil))
	require.Equal(t, "actual-upstream-model", grokMediaScheduleModel(account, "grok-imagine-video", &service.OpenAIForwardResult{
		UpstreamModel: "actual-upstream-model",
	}))
	require.Equal(t, "mapped-video-model", grokMediaScheduleModel(account, "grok-imagine-video", &service.OpenAIForwardResult{}))
	require.Equal(t, "grok-imagine-video", grokMediaScheduleModel(nil, " grok-imagine-video ", nil))
}

func TestEnsureGrokMediaAccountEligibility(t *testing.T) {
	t.Run("non oauth account does not probe", func(t *testing.T) {
		prober := &grokMediaEligibilityProberStub{}
		h := &OpenAIGatewayHandler{grokMediaEligibilityProber: prober}
		account := &service.Account{Platform: service.PlatformGrok, Type: service.AccountTypeAPIKey}

		eligible, reason, err := h.ensureGrokMediaAccountEligibility(context.Background(), account)

		require.NoError(t, err)
		require.True(t, eligible)
		require.Equal(t, "non_oauth", reason)
		require.Zero(t, prober.calls)
	})

	t.Run("unobserved oauth is probed before forwarding", func(t *testing.T) {
		prober := &grokMediaEligibilityProberStub{eligible: true, reason: "eligible"}
		h := &OpenAIGatewayHandler{grokMediaEligibilityProber: prober}
		account := &service.Account{ID: 7, Platform: service.PlatformGrok, Type: service.AccountTypeOAuth}

		eligible, reason, err := h.ensureGrokMediaAccountEligibility(context.Background(), account)

		require.NoError(t, err)
		require.True(t, eligible)
		require.Equal(t, "eligible", reason)
		require.Equal(t, 1, prober.calls)
	})

	t.Run("missing prober fails closed", func(t *testing.T) {
		h := &OpenAIGatewayHandler{}
		account := &service.Account{ID: 8, Platform: service.PlatformGrok, Type: service.AccountTypeOAuth}

		eligible, reason, err := h.ensureGrokMediaAccountEligibility(context.Background(), account)

		require.Error(t, err)
		require.False(t, eligible)
		require.Equal(t, "billing_probe_unavailable", reason)
	})

	t.Run("probe failure fails closed", func(t *testing.T) {
		probeErr := errors.New("probe failed")
		prober := &grokMediaEligibilityProberStub{reason: "billing_unobserved", err: probeErr}
		h := &OpenAIGatewayHandler{grokMediaEligibilityProber: prober}
		account := &service.Account{ID: 9, Platform: service.PlatformGrok, Type: service.AccountTypeOAuth}

		eligible, reason, err := h.ensureGrokMediaAccountEligibility(context.Background(), account)

		require.ErrorIs(t, err, probeErr)
		require.False(t, eligible)
		require.Equal(t, "billing_unobserved", reason)
	})
}
