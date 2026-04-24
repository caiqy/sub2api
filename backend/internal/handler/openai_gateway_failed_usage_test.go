package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
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
)

func waitForOpenAIFailedUsageLog(t *testing.T, repo *openAIChatCompletionsUsageLogRepoStub) *service.UsageLog {
	t.Helper()

	if repo.lastLog != nil {
		return repo.lastLog
	}

	select {
	case log := <-repo.created:
		repo.lastLog = log
		return log
	case <-time.After(2 * time.Second):
		return repo.lastLog
	}
}

func TestOpenAIGatewayHandler_SubmitFailedUsageLog_UsesMessagesFallbackModelAsUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	gatewayService := service.NewOpenAIGatewayService(nil, usageRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := NewOpenAIGatewayHandler(gatewayService, nil, nil, nil, nil, nil, nil)

	apiKey := &service.APIKey{ID: 101, UserID: 202, User: &service.User{ID: 202}}
	account := &service.Account{ID: 11, Platform: service.PlatformOpenAI, Credentials: map[string]any{"api_key": "sk-test"}}
	fallbackModel := "gpt-4.1-mini"
	reqModel := "claude-3-5-sonnet-20241022"

	router := gin.New()
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/messages", func(c *gin.Context) {
		c.Set("openai_messages_fallback_model", fallbackModel)
		h.submitFailedUsageLog(c, apiKey, account, reqModel, false, 0, nil, nil, 0, nil, "handler.openai_gateway.messages")
		c.Status(http.StatusBadRequest)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-3-5-sonnet-20241022","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, fallbackModel, *log.UpstreamModel)
}

func TestOpenAIGatewayHandler_SubmitFailoverFailedUsageLog_UsesChatCompletionsFallbackModelAsUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	gatewayService := service.NewOpenAIGatewayService(nil, usageRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := NewOpenAIGatewayHandler(gatewayService, nil, nil, nil, nil, nil, nil)

	apiKey := &service.APIKey{ID: 101, UserID: 202, User: &service.User{ID: 202}}
	account := &service.Account{ID: 11, Platform: service.PlatformOpenAI, Credentials: map[string]any{"api_key": "sk-test"}}
	fallbackModel := "gpt-4.1-mini"
	reqModel := "gpt-5.4"

	router := gin.New()
	router.Use(middleware.UsageDetailCapture())
	router.POST("/chat/completions", func(c *gin.Context) {
		c.Set("openai_chat_completions_fallback_model", fallbackModel)
		h.submitFailoverFailedUsageLog(c, apiKey, account, reqModel, false, &service.UpstreamFailoverError{}, 0, nil, "handler.openai_gateway.chat_completions")
		c.Status(http.StatusTooManyRequests)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chat/completions", strings.NewReader(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":false}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, fallbackModel, *log.UpstreamModel)
}

func TestOpenAIGatewayHandler_SubmitFailedUsageLog_PrefersExactUpstreamModelOverFallbackModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	gatewayService := service.NewOpenAIGatewayService(nil, usageRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	h := NewOpenAIGatewayHandler(gatewayService, nil, nil, nil, nil, nil, nil)

	apiKey := &service.APIKey{ID: 101, UserID: 202, User: &service.User{ID: 202}}
	account := &service.Account{
		ID:       11,
		Platform: service.PlatformOpenAI,
		Credentials: map[string]any{
			"api_key": "sk-test",
			"model_mapping": map[string]any{
				"gpt-4.1-mini": "re-mapped-by-account",
			},
		},
	}
	exactUpstreamModel := "exact-upstream-model"
	fallbackModel := "gpt-4.1-mini"
	reqModel := "claude-3-5-sonnet-20241022"

	router := gin.New()
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/messages", func(c *gin.Context) {
		c.Set("openai_messages_fallback_model", fallbackModel)
		c.Set("openai_failed_usage_upstream_model", exactUpstreamModel)
		h.submitFailedUsageLog(c, apiKey, account, reqModel, false, 0, nil, nil, 0, nil, "handler.openai_gateway.messages")
		c.Status(http.StatusBadRequest)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-3-5-sonnet-20241022","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, exactUpstreamModel, *log.UpstreamModel)
}

func TestOpenAIGatewayHandler_MessagesUpstreamErrorStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			Scheduling:         config.GatewaySchedulingConfig{LoadBatchEnabled: false},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	groupID := int64(1)
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
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
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		delay: 5 * time.Millisecond,
		response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_failed_messages_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"messages upstream rejected payload"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		nil,
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, cfg)

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
	router.POST("/v1/messages", h.Messages)

	reqBody := `{"model":"claude-3-5-sonnet-20241022","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"output_config":{"effort":"high"}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), `"type":"invalid_request_error"`)
	require.Contains(t, rec.Body.String(), "messages upstream rejected payload")
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created for non-failover errors")
	require.NotNil(t, log.DurationMs)
	require.Greater(t, *log.DurationMs, 0)
	require.NotNil(t, log.ReasoningEffort)
	require.Equal(t, "high", *log.ReasoningEffort)
	require.NotNil(t, log.DetailSnapshot)
	require.JSONEq(t, reqBody, log.DetailSnapshot.RequestBody)
	require.Contains(t, log.DetailSnapshot.ResponseBody, "messages upstream rejected payload")
	require.Contains(t, log.DetailSnapshot.UpstreamRequestHeaders, "Authorization: Bearer sk-test")
}

func TestOpenAIGatewayHandler_MessagesFailoverExhaustedStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			Scheduling:         config.GatewaySchedulingConfig{LoadBatchEnabled: false},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	groupID := int64(1)
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
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
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_messages_failover_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"openai_messages_rate_limited_raw","message":"openai messages raw failover"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		nil,
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, cfg)
	h.maxAccountSwitches = 0

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
	router.POST("/v1/messages", h.Messages)

	reqBody := `{"model":"claude-3-5-sonnet-20241022","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created when failover is exhausted")
	require.NotNil(t, log.DetailSnapshot)
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "Content-Type: application/json")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_messages_failover_123")
	require.Contains(t, log.DetailSnapshot.ResponseBody, `"openai_messages_rate_limited_raw"`)
	require.Contains(t, log.DetailSnapshot.ResponseBody, "openai messages raw failover")
}

func TestOpenAIGatewayHandler_ImagesForwardFailedUsageLogCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, usageRepo, cleanup := newOpenAIImagesHandlerTestRouter(t, "/v1/images/generations", &http.Response{
		StatusCode: http.StatusBadRequest,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"req_image_failed_123"},
		},
		Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"images upstream rejected payload"}}`)),
	})
	defer cleanup()
	usageRepo.created = make(chan *service.UsageLog, 1)

	reqBody := `{"model":"gpt-image-2","prompt":"draw a lantern","size":"1024x1024"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created for non-failover image errors")
	require.NotNil(t, log.DurationMs)
	require.NotNil(t, log.DetailSnapshot)
	require.JSONEq(t, reqBody, log.DetailSnapshot.RequestBody)
	require.Contains(t, log.DetailSnapshot.UpstreamRequestHeaders, ":method: POST")
	require.Contains(t, log.DetailSnapshot.UpstreamRequestHeaders, "/v1/images/generations")
	require.Contains(t, log.DetailSnapshot.UpstreamRequestHeaders, "Authorization: Bearer sk-test")
	require.JSONEq(t, reqBody, log.DetailSnapshot.UpstreamRequestBody)
	require.Contains(t, log.DetailSnapshot.ResponseBody, "images upstream rejected payload")
	require.NotNil(t, log.InboundEndpoint)
	require.Equal(t, "/v1/images/generations", *log.InboundEndpoint)
	require.NotNil(t, log.UpstreamEndpoint)
	require.Contains(t, *log.UpstreamEndpoint, "/v1/images/generations")
}

func TestOpenAIGatewayHandler_ImagesEditMultipartForwardFailedUsageLogUsesMetadataSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, usageRepo, cleanup := newOpenAIImagesHandlerTestRouter(t, "/v1/images/edits", &http.Response{
		StatusCode: http.StatusBadRequest,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"req_image_edit_failed_123"},
		},
		Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"images edit upstream rejected payload"}}`)),
	})
	defer cleanup()
	usageRepo.created = make(chan *service.UsageLog, 1)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "replace background"))
	require.NoError(t, writer.WriteField("size", "1536x1024"))
	require.NoError(t, writer.WriteField("quality", "high"))
	require.NoError(t, writer.WriteField("background", "transparent"))
	require.NoError(t, writer.WriteField("output_format", "webp"))
	require.NoError(t, writer.WriteField("moderation", "low"))
	require.NoError(t, writer.WriteField("n", "2"))
	imagePart, err := writer.CreateFormFile("image", "source.png")
	require.NoError(t, err)
	_, err = imagePart.Write([]byte("raw-source-image-bytes"))
	require.NoError(t, err)
	maskPart, err := writer.CreateFormFile("mask", "mask.png")
	require.NoError(t, err)
	_, err = maskPart.Write([]byte("raw-mask-bytes"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created for multipart edit errors")
	require.NotNil(t, log.DetailSnapshot)
	require.NotContains(t, log.DetailSnapshot.RequestBody, "raw-source-image-bytes")
	require.NotContains(t, log.DetailSnapshot.RequestBody, "raw-mask-bytes")
	require.Equal(t, "gpt-image-2", gjson.Get(log.DetailSnapshot.RequestBody, "model").String())
	require.Equal(t, "replace background", gjson.Get(log.DetailSnapshot.RequestBody, "prompt").String())
	require.Equal(t, "1536x1024", gjson.Get(log.DetailSnapshot.RequestBody, "size").String())
	require.Equal(t, "high", gjson.Get(log.DetailSnapshot.RequestBody, "quality").String())
	require.Equal(t, "transparent", gjson.Get(log.DetailSnapshot.RequestBody, "background").String())
	require.Equal(t, "webp", gjson.Get(log.DetailSnapshot.RequestBody, "output_format").String())
	require.Equal(t, "low", gjson.Get(log.DetailSnapshot.RequestBody, "moderation").String())
	require.Equal(t, int64(2), gjson.Get(log.DetailSnapshot.RequestBody, "n").Int())
	require.True(t, gjson.Get(log.DetailSnapshot.RequestBody, "had_source_image").Bool())
	require.True(t, gjson.Get(log.DetailSnapshot.RequestBody, "had_mask").Bool())
	require.Contains(t, log.DetailSnapshot.ResponseBody, "images edit upstream rejected payload")
}

func TestOpenAIGatewayHandler_ImagesOAuthForwardFailedUsagePreservesUpstreamSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
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
		Name:        "openai-oauth-image-account",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "acct_test",
		},
		Extra: map[string]any{
			"openai_device_id":  "device_test",
			"openai_session_id": "session_test",
		},
	}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_oauth_image_failed_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"message":"oauth images upstream rejected payload"}}`)),
		},
	}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		nil,
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, cfg)

	apiKey := &service.APIKey{
		ID:      101,
		UserID:  202,
		Status:  service.StatusActive,
		GroupID: &groupID,
		User:    &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1},
		Group:   group,
	}

	reqBody := `{"model":"gpt-image-2","prompt":"draw a lantern"}`

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/images/generations", h.Images)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.DetailSnapshot)
	require.JSONEq(t, reqBody, log.DetailSnapshot.RequestBody)
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, ":status: 400")
	require.NotContains(t, log.DetailSnapshot.ResponseHeaders, ":status: 502")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "Content-Type: application/json")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_oauth_image_failed_123")
	require.Contains(t, log.DetailSnapshot.ResponseBody, "oauth images upstream rejected payload")
	require.NotEmpty(t, strings.TrimSpace(log.DetailSnapshot.ResponseBody))
}

func TestOpenAIGatewayHandler_ImagesFailoverExhaustedFailedUsageLogCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, usageRepo, cleanup := newOpenAIImagesHandlerTestRouter(t, "/v1/images/generations", &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Request-Id": []string{"req_image_failover_123"},
		},
		Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","message":"images upstream overloaded"}}`)),
	})
	defer cleanup()
	usageRepo.created = make(chan *service.UsageLog, 1)

	reqBody := `{"model":"gpt-image-2","prompt":"draw a lantern","size":"1024x1024"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created when image failover is exhausted")
	require.NotNil(t, log.DurationMs)
	require.NotNil(t, log.DetailSnapshot)
	require.JSONEq(t, reqBody, log.DetailSnapshot.RequestBody)
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, ":status: 429")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "Content-Type: application/json")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_image_failover_123")
	require.Contains(t, log.DetailSnapshot.ResponseBody, "images upstream overloaded")
	require.NotNil(t, log.InboundEndpoint)
	require.Equal(t, "/v1/images/generations", *log.InboundEndpoint)
	require.NotNil(t, log.UpstreamEndpoint)
	require.Contains(t, *log.UpstreamEndpoint, "/v1/images/generations")
}

func TestOpenAIGatewayHandler_SubmitOpenAIImagesFailedUsageLog_UsesErrorSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
	}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	gatewayService := service.NewOpenAIGatewayService(
		nil,
		usageRepo,
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
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	h := &OpenAIGatewayHandler{gatewayService: gatewayService}

	groupID := int64(1)
	apiKey := &service.APIKey{
		ID:      101,
		UserID:  202,
		Status:  service.StatusActive,
		GroupID: &groupID,
		User:    &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1},
		Group:   &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true},
	}
	account := &service.Account{ID: 11, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive}
	parsed := &service.OpenAIImagesRequest{Endpoint: "/v1/images/generations", Model: "gpt-image-2", Prompt: "draw a lantern", N: 1}

	router := gin.New()
	router.Use(middleware.UsageDetailCapture())
	router.POST("/test", func(c *gin.Context) {
		h.submitOpenAIImagesFailedUsageLog(c, apiKey, account, parsed, fakeOpenAIImagesOAuthUpstreamError{
			statusCode: 418,
			responseHeaders: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_err_snapshot_123"},
			},
			responseBody: []byte(`{"error":{"message":"err-carried image snapshot"}}`),
		}, time.Second)
		c.Status(http.StatusTeapot)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(`{"model":"gpt-image-2","prompt":"draw a lantern"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log)
	require.NotNil(t, log.DetailSnapshot)
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, ":status: 418")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "Content-Type: application/json")
	require.Contains(t, log.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_err_snapshot_123")
	require.Contains(t, log.DetailSnapshot.ResponseBody, "err-carried image snapshot")
}

type fakeOpenAIImagesOAuthUpstreamError struct {
	statusCode      int
	responseHeaders http.Header
	responseBody    []byte
}

func (e fakeOpenAIImagesOAuthUpstreamError) Error() string {
	return "fake openai images oauth upstream error"
}

func (e fakeOpenAIImagesOAuthUpstreamError) OpenAIImageUpstreamStatusCode() int {
	return e.statusCode
}

func (e fakeOpenAIImagesOAuthUpstreamError) OpenAIImageUpstreamResponseHeaders() http.Header {
	return e.responseHeaders.Clone()
}

func (e fakeOpenAIImagesOAuthUpstreamError) OpenAIImageUpstreamResponseBody() []byte {
	return append([]byte(nil), e.responseBody...)
}

var oauthImagesFallbackRootsOnce sync.Once

func newOAuthImagesFailureProxy(t *testing.T, upstreamHeaders http.Header, upstreamBody []byte) (*service.Proxy, func()) {
	t.Helper()

	serverCert, rootCert := newChatGPTDotComTLSCert(t)
	installOAuthImagesFallbackRoots(t, rootCert)

	upstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/backend-api/sentinel/chat-requirements":
			for key, values := range upstreamHeaders {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(upstreamBody)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	upstream.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}}
	upstream.StartTLS()

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, "connect required", http.StatusBadGateway)
			return
		}

		targetConn, err := net.Dial("tcp", upstream.Listener.Addr().String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		hijacker, ok := w.(http.Hijacker)
		if !ok {
			targetConn.Close()
			http.Error(w, "hijack unsupported", http.StatusInternalServerError)
			return
		}
		clientConn, _, err := hijacker.Hijack()
		if err != nil {
			targetConn.Close()
			return
		}
		_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

		go func() {
			_, _ = io.Copy(targetConn, clientConn)
			_ = targetConn.Close()
			_ = clientConn.Close()
		}()
		go func() {
			_, _ = io.Copy(clientConn, targetConn)
			_ = clientConn.Close()
			_ = targetConn.Close()
		}()
	}))

	parsed, err := url.Parse(proxyServer.URL)
	require.NoError(t, err)
	port := 0
	if parsed.Port() != "" {
		port, err = strconv.Atoi(parsed.Port())
		require.NoError(t, err)
	}

	cleanup := func() {
		proxyServer.Close()
		upstream.Close()
	}

	return &service.Proxy{
		ID:       99,
		Protocol: parsed.Scheme,
		Host:     parsed.Hostname(),
		Port:     port,
		Status:   service.StatusActive,
	}, cleanup
}

func installOAuthImagesFallbackRoots(t *testing.T, rootCert *x509.Certificate) {
	t.Helper()
	t.Setenv("GODEBUG", "x509usefallbackroots=1")
	oauthImagesFallbackRootsOnce.Do(func() {
		pool := x509.NewCertPool()
		pool.AddCert(rootCert)
		x509.SetFallbackRoots(pool)
	})
}

func newChatGPTDotComTLSCert(t *testing.T) (tls.Certificate, *x509.Certificate) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "sub2api test root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	rootCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "chatgpt.com"},
		DNSNames:     []string{"chatgpt.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, rootCert, &leafKey.PublicKey, caKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(leafKey)})
	serverCert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	return serverCert, rootCert
}

func TestOpenAIGatewayHandler_MessagesSelectionExhaustedAfterFailoverStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			Scheduling:         config.GatewaySchedulingConfig{LoadBatchEnabled: false},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
	}

	groupID := int64(1)
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true, AllowMessagesDispatch: true}
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
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_messages_selection_exhausted_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"openai_messages_rate_limited_raw","message":"openai messages raw failover"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		nil,
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, cfg)

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
	router.POST("/v1/messages", h.Messages)

	reqBody := `{"model":"claude-3-5-sonnet-20241022","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.JSONEq(t, `{"type":"error","error":{"type":"rate_limit_error","message":"Upstream rate limit exceeded, please retry later"}}`, rec.Body.String())
	log := waitForOpenAIFailedUsageLog(t, usageRepo)
	require.NotNil(t, log, "failed usage log should be created when selection is exhausted after failover")
	require.NotNil(t, log.DetailSnapshot)
	require.Contains(t, log.DetailSnapshot.ResponseBody, `"openai_messages_rate_limited_raw"`)
	require.Contains(t, log.DetailSnapshot.ResponseBody, "openai messages raw failover")
	require.Contains(t, log.DetailSnapshot.UpstreamRequestHeaders, "Authorization: Bearer sk-test")
}

func TestOpenAIGatewayHandler_UpstreamErrorStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
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
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		delay: 5 * time.Millisecond,
		response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_failed_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"upstream rejected payload"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		nil,
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, cfg)
	h.maxAccountSwitches = 0

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

	reqBody := `{"model":"gpt-5.4","reasoning":{"effort":"high"},"stream":false,"input":"hello"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, 0, usageRepo.lastLog.InputTokens)
	require.Equal(t, 0, usageRepo.lastLog.OutputTokens)
	require.Equal(t, 0.0, usageRepo.lastLog.TotalCost)
	require.Equal(t, 0.0, usageRepo.lastLog.ActualCost)
	require.NotNil(t, usageRepo.lastLog.DurationMs)
	require.Greater(t, *usageRepo.lastLog.DurationMs, 0)
	require.NotNil(t, usageRepo.lastLog.ReasoningEffort)
	require.Equal(t, "high", *usageRepo.lastLog.ReasoningEffort)
	require.NotNil(t, usageRepo.lastLog.DetailSnapshot)
	require.JSONEq(t, reqBody, usageRepo.lastLog.DetailSnapshot.RequestBody)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "upstream rejected payload")
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.UpstreamRequestHeaders, "Authorization: Bearer sk-test")
}

func TestOpenAIGatewayHandler_ChatCompletionsUpstreamErrorStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
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
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusBadRequest,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_failed_chat_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"chat upstream rejected payload"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		nil,
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, cfg)

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
	router.POST("/chat/completions", h.ChatCompletions)

	reqBody := `{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":false}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, 0, usageRepo.lastLog.InputTokens)
	require.Equal(t, 0, usageRepo.lastLog.OutputTokens)
	require.Equal(t, 0.0, usageRepo.lastLog.TotalCost)
	require.Equal(t, 0.0, usageRepo.lastLog.ActualCost)
	require.NotNil(t, usageRepo.lastLog.DetailSnapshot)
	require.JSONEq(t, reqBody, usageRepo.lastLog.DetailSnapshot.RequestBody)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "chat upstream rejected payload")
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.UpstreamRequestHeaders, "Authorization: Bearer sk-test")
}

type openAIFailoverAccountRepoStub struct {
	openAIRetryAccountRepoStub
}

func (s *openAIFailoverAccountRepoStub) SetError(ctx context.Context, id int64, errorMsg string) error {
	return nil
}

func TestOpenAIGatewayHandler_FailoverExhaustedStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
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
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_failover_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"openai_rate_limited_raw","message":"openai raw failover"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		nil,
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, cfg)

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

	reqBody := `{"model":"gpt-5.4","stream":false,"input":"hello"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.NotNil(t, usageRepo.lastLog)
	require.NotNil(t, usageRepo.lastLog.DetailSnapshot)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseHeaders, "Content-Type: application/json")
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseHeaders, "X-Request-Id: req_failover_123")
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, `"openai_rate_limited_raw"`)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "openai raw failover")
}

func TestOpenAIGatewayHandler_ChatCompletionsFailoverExhaustedStillCreatesUsageLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
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
	httpUpstream := &openAIChatCompletionsHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Request-Id": []string{"req_chat_failover_123"},
			},
			Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","code":"openai_chat_rate_limited_raw","message":"openai chat raw failover"}}`)),
		},
	}
	accountRepo := &openAIChatCompletionsAccountRepoStub{account: account}
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		nil,
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
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, cfg)
	h.maxAccountSwitches = 0

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
	router.POST("/chat/completions", h.ChatCompletions)

	reqBody := `{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"stream":false}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.NotNil(t, usageRepo.lastLog)
	require.NotNil(t, usageRepo.lastLog.DetailSnapshot)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, `"openai_chat_rate_limited_raw"`)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "openai chat raw failover")
}

func TestOpenAIGatewayHandler_RetrySuccessDoesNotReuseFailoverErrorSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Default: config.DefaultConfig{RateMultiplier: 1},
		Gateway: config.GatewayConfig{
			MaxAccountSwitches: 1,
			Scheduling:         config.GatewaySchedulingConfig{LoadBatchEnabled: false},
		},
		Concurrency: config.ConcurrencyConfig{PingInterval: 0},
		RateLimit:   config.RateLimitConfig{OAuth401CooldownMinutes: 1},
	}

	groupID := int64(1)
	group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}
	account1 := &service.Account{ID: 11, Name: "openai-account-1", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1, Credentials: map[string]any{"api_key": "sk-test-1"}}
	account2 := &service.Account{ID: 12, Name: "openai-account-2", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2, Credentials: map[string]any{"api_key": "sk-test-2"}}
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{}
	httpUpstream := &openAIRetryTrackingHTTPUpstreamStub{
		responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_failover_disable_1"}},
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"organization has been disabled"}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_success_after_failover"}},
				Body:       io.NopCloser(strings.NewReader(`{"id":"resp_success","object":"response","status":"completed","model":"gpt-5.4","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Hello from success"}]}],"usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}`)),
			},
		},
	}
	accountRepo := &openAIFailoverAccountRepoStub{openAIRetryAccountRepoStub{accounts: []*service.Account{account1, account2}}}
	rateLimitService := service.NewRateLimitService(accountRepo, nil, cfg, nil, nil)
	concurrencyService := service.NewConcurrencyService(openAIChatCompletionsConcurrencyCacheStub{})
	billingCacheService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg)
	deferredService := service.NewDeferredService(accountRepo, nil, 0)
	billingService := service.NewBillingService(cfg, nil)
	t.Cleanup(func() { billingCacheService.Stop() })

	gatewayService := service.NewOpenAIGatewayService(
		accountRepo,
		usageRepo,
		nil,
		nil,
		nil,
		nil,
		openAIChatCompletionsGatewayCacheStub{},
		cfg,
		nil,
		concurrencyService,
		billingService,
		rateLimitService,
		billingCacheService,
		httpUpstream,
		deferredService,
		nil,
		nil,
		nil,
		nil,
	)
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, &service.APIKeyService{}, nil, nil, cfg)

	apiKey := &service.APIKey{ID: 101, UserID: 202, Status: service.StatusActive, GroupID: &groupID, User: &service.User{ID: 202, Status: service.StatusActive, Concurrency: 1}, Group: group}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID, Concurrency: apiKey.User.Concurrency})
		c.Next()
	})
	router.Use(middleware.UsageDetailCapture())
	router.POST("/v1/responses", h.Responses)

	reqBody := `{"model":"gpt-5.4","stream":false,"input":"hello"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, usageRepo.lastLog)
	require.NotNil(t, usageRepo.lastLog.DetailSnapshot)
	require.Contains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "Hello from success")
	require.NotContains(t, usageRepo.lastLog.DetailSnapshot.ResponseBody, "organization has been disabled")
}
