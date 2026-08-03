package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIGatewayHandler_ResponsesPassesPreviewSnapshotAndStableHash(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldPreviewLimit := openAIResponsesRequestBodyPreviewLimitBytes
	openAIResponsesRequestBodyPreviewLimitBytes = 32
	t.Cleanup(func() { openAIResponsesRequestBodyPreviewLimitBytes = oldPreviewLimit })

	cfg := &config.Config{
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			Scheduling:         config.GatewaySchedulingConfig{LoadBatchEnabled: false},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	groupID := int64(1)
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account := &service.Account{
		ID:          11,
		Name:        "openai-test-account",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{"api_key": "sk-test"},
	}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{}
	billingRepo := &openAIResponsesRequestBodyRetentionBillingRepoStub{}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_test_123"}},
			Body: io.NopCloser(strings.NewReader(
				`{"id":"resp_1","object":"response","status":"completed","model":"gpt-5","output":[],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2,"input_tokens_details":{"cached_tokens":0}}}`,
			)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(&openAIResponsesRequestBodyRetentionBillingCacheStub{}, nil, nil, nil, nil, nil, cfg, nil)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		billingRepo,
		nil,
		nil,
		nil,
		openAIChatCompletionsGatewayCacheStub{},
		cfg,
		nil,
		concurrencyService,
		billingService,
		nil,
		billingCacheService,
		httpUpstream,
		deferredService,
		nil,
		nil,
		nil,
		nil,
	)
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, nil, nil, cfg, nil)

	apiKey := &service.APIKey{
		ID:      101,
		UserID:  202,
		Status:  service.StatusActive,
		GroupID: &groupID,
		User:    &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1},
		Group:   group,
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/responses", h.Responses)

	reqBody := `{"model":"gpt-5","stream":false,"input":"` + strings.Repeat("x", 80) + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, usageRepo.lastLog)
	require.NotNil(t, usageRepo.lastLog.DetailSnapshot)
	requestPreview := usageRepo.lastLog.DetailSnapshot.RequestBody
	require.NotEqual(t, reqBody, requestPreview)
	require.True(t, gjson.Valid(requestPreview), requestPreview)
	require.Equal(t, requestBodyPreviewSnapshotKind, gjson.Get(requestPreview, "kind").String())
	require.Equal(t, "[inline binary payload omitted]", gjson.Get(requestPreview, "preview").String())
	require.True(t, gjson.Get(requestPreview, "truncated").Bool())
	require.Equal(t, int64(len(reqBody)), gjson.Get(requestPreview, "size").Int())

	require.NotNil(t, billingRepo.lastCmd)
	require.Equal(t, service.HashUsageRequestPayload([]byte(reqBody)), billingRepo.lastCmd.RequestPayloadHash)
}

func TestOpenAIRequestBodyPreviewSnapshotSpoolFailureStillReturnsWrapper(t *testing.T) {
	oldPreviewLimit := openAIResponsesRequestBodyPreviewLimitBytes
	openAIResponsesRequestBodyPreviewLimitBytes = 32
	t.Cleanup(func() { openAIResponsesRequestBodyPreviewLimitBytes = oldPreviewLimit })

	missingTempDir := filepath.Join(t.TempDir(), "missing")
	t.Setenv("TMPDIR", missingTempDir)
	t.Setenv("TMP", missingTempDir)
	t.Setenv("TEMP", missingTempDir)
	body := []byte(strings.Repeat("x", int(service.DefaultRequestBodySpoolThresholdBytes)+1))

	snapshot := openAIRequestBodyPreviewSnapshot(body)

	require.True(t, gjson.Valid(snapshot), snapshot)
	require.Equal(t, requestBodyPreviewSnapshotKind, gjson.Get(snapshot, "kind").String())
	require.Equal(t, "[inline binary payload omitted]", gjson.Get(snapshot, "preview").String())
	require.True(t, gjson.Get(snapshot, "truncated").Bool())
	require.Equal(t, int64(len(body)), gjson.Get(snapshot, "size").Int())
}

type openAIResponsesRequestBodyRetentionBillingRepoStub struct {
	service.UsageBillingRepository

	lastCmd *service.UsageBillingCommand
}

func (s *openAIResponsesRequestBodyRetentionBillingRepoStub) Apply(ctx context.Context, cmd *service.UsageBillingCommand) (*service.UsageBillingApplyResult, error) {
	s.lastCmd = cmd
	return &service.UsageBillingApplyResult{Applied: true}, nil
}

type openAIResponsesRequestBodyRetentionBillingCacheStub struct {
	service.BillingCache
}

func (openAIResponsesRequestBodyRetentionBillingCacheStub) GetUserBalance(context.Context, int64) (float64, error) {
	return 100, nil
}
