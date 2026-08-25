package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type anthropicHTTPUpstreamRecorder struct {
	lastReq        *http.Request
	lastBody       []byte
	lastTLSProfile *tlsfingerprint.Profile
	resp           *http.Response
	err            error
}

func TestGatewayService_AnthropicAPIKeyPassthroughRetriesFromBodyHandles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"claude-test","messages":[{"role":"user","content":"retry"}]}`)
	source, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create source handle: %v", err)
	}
	t.Cleanup(func() { CleanupRequestBodyHandle(source) })
	effective, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	if err != nil {
		t.Fatalf("create effective handle: %v", err)
	}
	t.Cleanup(func() { CleanupRequestBodyHandle(effective) })

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	upstream := &anthropicRetryHandleUpstream{statuses: []int{http.StatusInternalServerError, http.StatusOK}}
	svc := &GatewayService{
		cfg:                 &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream:        upstream,
		tlsFPProfileService: &TLSFingerprintProfileService{},
	}
	account := newAnthropicAPIKeyAccountForTest()
	account.Credentials["custom_error_codes_enabled"] = true
	account.Credentials["custom_error_codes"] = []any{float64(http.StatusBadRequest)}
	_, err = svc.forwardAnthropicAPIKeyPassthroughWithInput(context.Background(), c, account, anthropicPassthroughForwardInput{
		SourceHandle:  source,
		BodyHandle:    effective,
		RequestModel:  "claude-test",
		OriginalModel: "claude-test",
		StartTime:     time.Now(),
	})
	if err != nil {
		t.Fatalf("passthrough: %v", err)
	}
	if len(upstream.bodies) != 2 {
		t.Fatalf("attempts = %d, want 2", len(upstream.bodies))
	}
	for i, got := range upstream.bodies {
		if !bytes.Equal(got, body) {
			t.Fatalf("attempt %d body = %q, want %q", i+1, got, body)
		}
	}
}

func TestGatewayService_AnthropicPassthroughRetryRereadsHandleAfterForwardFirstAttemptBytes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spoolDir := t.TempDir()
	t.Setenv("TMPDIR", spoolDir)
	t.Setenv("TMP", spoolDir)
	t.Setenv("TEMP", spoolDir)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"` + strings.Repeat("x", 2<<20) + `"}]}`)
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-sonnet-4-5"}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	upstream := &anthropicRetryHandleUpstream{
		statuses:         []int{http.StatusInternalServerError, http.StatusOK},
		removeSpoolDir:   spoolDir,
		removeSpoolAfter: 1,
	}
	cfg := &config.Config{
		Gateway:  config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
		Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}},
	}
	svc := &GatewayService{
		cfg:                 cfg,
		httpUpstream:        upstream,
		rateLimitService:    &RateLimitService{},
		settingService:      NewSettingService(upstreamPreviewSettingRepo{}, cfg),
		tlsFPProfileService: &TLSFingerprintProfileService{},
	}
	account := newAnthropicAPIKeyAccountForTest()
	account.Credentials["custom_error_codes_enabled"] = true
	account.Credentials["custom_error_codes"] = []any{float64(http.StatusBadRequest)}

	result, err := svc.Forward(context.Background(), c, account, parsed)
	require.Nil(t, result)
	require.ErrorIs(t, err, ErrRequestBodySpool)
	require.Len(t, upstream.bodies, 1)
}

type anthropicRetryHandleUpstream struct {
	statuses         []int
	bodies           [][]byte
	removeSpoolDir   string
	removeSpoolAfter int
}

func (u *anthropicRetryHandleUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return nil, errors.New("Do must not be used")
}

func (u *anthropicRetryHandleUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	if req.GetBody == nil || req.ContentLength <= 0 {
		return nil, errors.New("request is not handle-backed")
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	u.bodies = append(u.bodies, body)
	if u.removeSpoolDir != "" && len(u.bodies) == u.removeSpoolAfter {
		entries, err := os.ReadDir(u.removeSpoolDir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if err := os.Remove(filepath.Join(u.removeSpoolDir, entry.Name())); err != nil {
				return nil, err
			}
		}
	}
	status := u.statuses[len(u.bodies)-1]
	responseBody := []byte(`{"id":"msg_test","model":"claude-test","usage":{"input_tokens":1,"output_tokens":1}}`)
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(responseBody))}, nil
}

type upstreamPreviewSettingRepo struct{ SettingRepository }

func (upstreamPreviewSettingRepo) GetValue(context.Context, string) (string, error) {
	return "", ErrSettingNotFound
}

func (upstreamPreviewSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func newAnthropicAPIKeyAccountForTest() *Account {
	return &Account{
		ID:          201,
		Name:        "anthropic-apikey-pass-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "upstream-anthropic-key",
			"base_url": "https://api.anthropic.com",
		},
		Extra: map[string]any{
			"anthropic_passthrough": true,
		},
		Status:      StatusActive,
		Schedulable: true,
	}
}

func TestGatewayCompatCallersCaptureFinalPreviewAttemptAndFailoverHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		path string
		body []byte
		run  func(*GatewayService, *gin.Context, *Account, []byte, *ParsedRequest) error
	}{
		{
			name: "chat_completions",
			path: "/v1/chat/completions",
			body: []byte(`{"model":"claude-sonnet-4-20250514","messages":[{"role":"user","content":"hello"}]}`),
			run: func(s *GatewayService, c *gin.Context, account *Account, body []byte, parsed *ParsedRequest) error {
				_, err := s.ForwardAsChatCompletions(context.Background(), c, account, body, parsed)
				return err
			},
		},
		{
			name: "responses",
			path: "/v1/responses",
			body: []byte(`{"model":"claude-sonnet-4-20250514","input":"hello"}`),
			run: func(s *GatewayService, c *gin.Context, account *Account, body []byte, parsed *ParsedRequest) error {
				_, err := s.ForwardAsResponses(context.Background(), c, account, body, parsed)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(tt.body))
			collector := &openAIUsageUpstreamRequestCollector{}
			c.Set(UsageDetailCaptureContextKey, collector)
			upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req-compat-final"}},
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"rate limited"}}`)),
			}}
			svc := &GatewayService{
				cfg:                 &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
				httpUpstream:        upstream,
				tlsFPProfileService: &TLSFingerprintProfileService{},
			}
			account := newAnthropicAPIKeyAccountForTest()
			account.Extra = map[string]any{
				"passthrough_fields_enabled": true,
				"passthrough_field_rules":    []PassthroughFieldRule{{Target: "body", Mode: "inject", Key: "metadata.final_preview", Value: tt.name}},
			}
			parsed := &ParsedRequest{Body: NewRequestBodyRef(tt.body), Model: "claude-sonnet-4-20250514"}

			err := tt.run(svc, c, account, tt.body, parsed)
			var failoverErr *UpstreamFailoverError
			require.ErrorAs(t, err, &failoverErr)
			require.Equal(t, "req-compat-final", failoverErr.ResponseHeaders.Get("X-Request-Id"))
			require.JSONEq(t, string(upstream.lastBody), collector.body)
			require.Equal(t, tt.name, gjson.Get(collector.body, "metadata.final_preview").String())
			require.JSONEq(t, collector.body, requireOpsPreviewString(t, c, tt.name))
			require.LessOrEqual(t, len(collector.body), int(DefaultRequestBodyPreviewLimitBytes))
			require.True(t, HasOpsUpstreamAttempted(c))
		})
	}
}

func TestExecuteBedrockUpstreamCapturesCurrentPreviewAndAttempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	collector := &openAIUsageUpstreamRequestCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)
	body := []byte(`{"anthropic_version":"bedrock-2023-05-31","messages":[{"role":"user","content":"hello bedrock"}],"max_tokens":16}`)
	upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}}
	svc := &GatewayService{httpUpstream: upstream}
	account := &Account{ID: 901, Name: "bedrock-apikey", Platform: PlatformAnthropic, Type: AccountTypeBedrock, Concurrency: 1, Credentials: map[string]any{"auth_mode": "apikey"}}
	bodyHandle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{})
	require.NoError(t, err)
	defer CleanupRequestBodyHandle(bodyHandle)

	resp, err := svc.executeBedrockUpstream(context.Background(), c, account, bodyHandle, "anthropic.claude-sonnet-4-5-v1:0", "us-east-1", false, nil, "bedrock-key", "")
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer func() { _ = resp.Body.Close() }()
	require.JSONEq(t, string(body), collector.body)
	require.JSONEq(t, collector.body, requireOpsPreviewString(t, c, "hello bedrock"))
	require.True(t, HasOpsUpstreamAttempted(c))
}

func TestHandleBedrockUpstreamErrors_Secondary400FailoverClonesResponseHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	repo := &countTokensRuntimeStateRepo{}
	svc := &GatewayService{rateLimitService: &RateLimitService{accountRepo: repo, cfg: &config.Config{}}}
	account := &Account{ID: 902, Name: "bedrock-disabled", Platform: PlatformAnthropic, Type: AccountTypeBedrock, Status: StatusActive}
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header: http.Header{
			"Content-Type":     []string{"application/json"},
			"X-Amzn-Requestid": []string{"bedrock-request-400"},
			"X-Amzn-Errortype": []string{"ValidationException"},
		},
		Body: io.NopCloser(strings.NewReader(`{"message":"organization has been disabled"}`)),
	}

	result, err := svc.handleBedrockUpstreamErrors(context.Background(), resp, c, account)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, "bedrock-request-400", failoverErr.ResponseHeaders.Get("x-amzn-requestid"))
	require.Equal(t, "ValidationException", failoverErr.ResponseHeaders.Get("x-amzn-errortype"))
	require.Equal(t, "application/json", failoverErr.ResponseHeaders.Get("content-type"))
	require.Equal(t, 1, repo.setErrorCalls)

	resp.Header.Set("X-Amzn-Requestid", "mutated")
	require.Equal(t, "bedrock-request-400", failoverErr.ResponseHeaders.Get("x-amzn-requestid"))
}

func (u *anthropicHTTPUpstreamRecorder) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	u.lastReq = req
	if req != nil && req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		u.lastBody = b
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(b))
	}
	if u.err != nil {
		return nil, u.err
	}
	return u.resp, nil
}

func (u *anthropicHTTPUpstreamRecorder) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	u.lastTLSProfile = profile
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

type streamReadCloser struct {
	payload []byte
	sent    bool
	err     error
}

func (r *streamReadCloser) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		n := copy(p, r.payload)
		return n, nil
	}
	if r.err != nil {
		return 0, r.err
	}
	return 0, io.EOF
}

func (r *streamReadCloser) Close() error { return nil }

type failWriteResponseWriter struct {
	gin.ResponseWriter
}

func (w *failWriteResponseWriter) Write(data []byte) (int, error) {
	return 0, errors.New("client disconnected")
}

func (w *failWriteResponseWriter) WriteString(_ string) (int, error) {
	return 0, errors.New("client disconnected")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_ForwardStreamPreservesBodyAndAuthReplacement(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Request.Header.Set("User-Agent", "claude-cli/1.0.0")
	c.Request.Header.Set("Authorization", "Bearer inbound-token")
	c.Request.Header.Set("X-Api-Key", "inbound-api-key")
	c.Request.Header.Set("X-Goog-Api-Key", "inbound-goog-key")
	c.Request.Header.Set("Cookie", "secret=1")
	c.Request.Header.Set("Anthropic-Beta", "interleaved-thinking-2025-05-14")
	c.Request.Header.Set("X-Trace-Id", "trace-123")
	c.Request.Header.Set("X-Test", "keep")

	body := []byte(`{"model":"claude-3-7-sonnet-20250219","stream":true,"system":[{"type":"text","text":"x-anthropic-billing-header keep"}],"metadata":{"client_trace":"trace-123"},"messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	parsed := &ParsedRequest{
		Body:   NewRequestBodyRef(body),
		Model:  "claude-3-7-sonnet-20250219",
		Stream: true,
	}

	upstreamSSE := strings.Join([]string{
		`data: {"type":"message_start","message":{"usage":{"input_tokens":9,"cached_tokens":7}}}`,
		"",
		`data: {"type":"message_delta","usage":{"output_tokens":3}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"x-request-id": []string{"rid-anthropic-pass"},
				"Set-Cookie":   []string{"secret=upstream"},
			},
			Body: io.NopCloser(strings.NewReader(upstreamSSE)),
		},
	}

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			MaxLineSize: defaultMaxLineSize,
		},
	}
	svc := &GatewayService{
		cfg:                  cfg,
		responseHeaderFilter: compileResponseHeaderFilter(cfg),
		httpUpstream:         upstream,
		rateLimitService:     &RateLimitService{},
		deferredService:      &DeferredService{},
		billingCacheService:  nil,
	}

	account := &Account{
		ID:          101,
		Name:        "anthropic-apikey-pass",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":       "upstream-anthropic-key",
			"base_url":      "https://api.anthropic.com",
			"model_mapping": map[string]any{"claude-3-7-sonnet-20250219": "claude-3-haiku-20240307"},
		},
		Extra: map[string]any{
			"anthropic_passthrough":      true,
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "header", Mode: "inject", Key: "X-Account-Tag", Value: "prod"},
				{Target: "header", Mode: "forward", Key: "X-Trace-Id"},
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
				{Target: "body", Mode: "forward", Key: "metadata.client_trace"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)

	require.Equal(t, "claude-3-haiku-20240307", gjson.GetBytes(upstream.lastBody, "model").String(), "透传模式应应用账号级模型映射")

	require.Equal(t, "upstream-anthropic-key", getHeaderRaw(upstream.lastReq.Header, "x-api-key"))
	require.Equal(t, "prod", getHeaderRaw(upstream.lastReq.Header, "X-Account-Tag"))
	require.Equal(t, "trace-123", getHeaderRaw(upstream.lastReq.Header, "X-Trace-Id"))
	require.Empty(t, getHeaderRaw(upstream.lastReq.Header, "authorization"))
	require.Empty(t, getHeaderRaw(upstream.lastReq.Header, "x-goog-api-key"))
	require.Empty(t, getHeaderRaw(upstream.lastReq.Header, "cookie"))
	require.Empty(t, getHeaderRaw(upstream.lastReq.Header, "X-Test"))
	require.Equal(t, "2023-06-01", getHeaderRaw(upstream.lastReq.Header, "anthropic-version"))
	require.Equal(t, "interleaved-thinking-2025-05-14", getHeaderRaw(upstream.lastReq.Header, "anthropic-beta"))
	require.Empty(t, getHeaderRaw(upstream.lastReq.Header, "x-stainless-lang"), "API Key 透传不应注入 OAuth 指纹头")
	require.Equal(t, "user-1", gjson.GetBytes(upstream.lastBody, "metadata.user_id").String())
	require.Equal(t, "trace-123", gjson.GetBytes(upstream.lastBody, "metadata.client_trace").String())

	require.Contains(t, rec.Body.String(), `"cached_tokens":7`)
	require.NotContains(t, rec.Body.String(), `"cache_read_input_tokens":7`, "透传输出不应被网关改写")
	require.Equal(t, 7, result.Usage.CacheReadInputTokens, "计费 usage 解析应保留 cached_tokens 兼容")
	require.Empty(t, rec.Header().Get("Set-Cookie"), "响应头应经过安全过滤")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_DisabledLeavesConfiguredFieldsInactive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Request.Header.Set("X-Trace-Id", "trace-123")

	body := []byte(`{"model":"claude-3-5-sonnet-latest","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-3-5-sonnet-latest"}
	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","usage":{"input_tokens":5,"output_tokens":3}}`)),
		},
	}
	svc := &GatewayService{cfg: &config.Config{}, httpUpstream: upstream, rateLimitService: &RateLimitService{}}
	account := &Account{
		ID:          103,
		Name:        "anthropic-apikey-pass-disabled",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "upstream-anthropic-key", "base_url": "https://api.anthropic.com"},
		Extra: map[string]any{
			"anthropic_passthrough":      true,
			"passthrough_fields_enabled": false,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "header", Mode: "inject", Key: "X-Account-Tag", Value: "prod"},
				{Target: "header", Mode: "forward", Key: "X-Trace-Id"},
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	_, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.Equal(t, body, upstream.lastBody)
	require.Empty(t, upstream.lastReq.Header.Get("X-Account-Tag"))
	require.Empty(t, upstream.lastReq.Header.Get("X-Trace-Id"))
}

func TestGatewayService_AnthropicFieldsApplyWithoutAnthropicPassthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Request.Header.Set("X-Trace-Id", "trace-123")

	body := []byte(`{"model":"claude-3-5-sonnet-latest","metadata":{"client_trace":"trace-123"},"messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-3-5-sonnet-latest"}
	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","usage":{"input_tokens":5,"output_tokens":3}}`)),
		},
	}
	svc := &GatewayService{cfg: &config.Config{}, httpUpstream: upstream, rateLimitService: &RateLimitService{}}
	account := &Account{
		ID:          105,
		Name:        "anthropic-apikey-decoupled",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "upstream-anthropic-key", "base_url": "https://api.anthropic.com"},
		Extra: map[string]any{
			"anthropic_passthrough":      false,
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "header", Mode: "inject", Key: "X-Account-Tag", Value: "prod"},
				{Target: "header", Mode: "forward", Key: "X-Trace-Id"},
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
				{Target: "body", Mode: "forward", Key: "metadata.client_trace"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	_, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "prod", upstream.lastReq.Header.Get("X-Account-Tag"))
	require.Equal(t, "trace-123", upstream.lastReq.Header.Get("X-Trace-Id"))
	require.Equal(t, "user-1", gjson.GetBytes(upstream.lastBody, "metadata.user_id").String())
	require.Equal(t, "trace-123", gjson.GetBytes(upstream.lastBody, "metadata.client_trace").String())
}

func TestGatewayService_AnthropicPassthroughBuildFailureKeepsOriginalOpsBodySize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	payload := strings.Repeat("x", int(DefaultRequestBodyPreviewLimitBytes)+1024)
	body := []byte(`{"model":"claude-3-5-sonnet-latest","image_url":"data:image/png;base64,c2VjcmV0","padding":"` + payload + `"}`)
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-3-5-sonnet-latest"}
	svc := &GatewayService{cfg: &config.Config{}, rateLimitService: &RateLimitService{}}
	account := &Account{
		ID:       106,
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "upstream-anthropic-key",
			"base_url": "://invalid",
		},
		Extra: map[string]any{"anthropic_passthrough": true},
	}

	_, err := svc.Forward(context.Background(), c, account, parsed)
	require.Error(t, err)
	raw, ok := c.Get(OpsUpstreamRequestBodyKey)
	require.True(t, ok)
	preview, ok := raw.(string)
	require.True(t, ok)
	snapshot, ok := parseRequestBodyPreviewSnapshot(preview)
	require.True(t, ok)
	require.Equal(t, int64(len(body)), snapshot.Size)
	require.True(t, snapshot.Truncated)
	require.Equal(t, requestBodyPreviewOmittedMarker, snapshot.Preview)
}

func TestPassthroughFieldsV2AnthropicAPIKeyPassthrough_BodyInjectAndMapDoNotChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	body := []byte(`{"model":"claude-3-5-sonnet-latest","metadata":{"client_trace":"trace-123"},"messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-3-5-sonnet-latest"}
	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","usage":{"input_tokens":5,"output_tokens":3}}`)),
		},
	}
	svc := &GatewayService{cfg: &config.Config{}, httpUpstream: upstream, rateLimitService: &RateLimitService{}}
	account := &Account{
		ID:          106,
		Name:        "anthropic-apikey-v2-map",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "upstream-anthropic-key", "base_url": "https://api.anthropic.com"},
		Extra: map[string]any{
			"anthropic_passthrough":      true,
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
				{Target: "body", Mode: "map", Key: "metadata.copied_user_id", SourceKey: "metadata.user_id"},
				{Target: "body", Mode: "map", Key: "metadata.client_trace_copy", SourceKey: "metadata.client_trace"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	_, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.Equal(t, "user-1", gjson.GetBytes(upstream.lastBody, "metadata.user_id").String())
	require.False(t, gjson.GetBytes(upstream.lastBody, "metadata.copied_user_id").Exists())
	require.Equal(t, "trace-123", gjson.GetBytes(upstream.lastBody, "metadata.client_trace_copy").String())
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StructureConflictReturnsInvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	body := []byte(`{"model":"claude-3-5-sonnet-latest","metadata":"string","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-3-5-sonnet-latest"}
	upstream := &anthropicHTTPUpstreamRecorder{}
	svc := &GatewayService{cfg: &config.Config{}, httpUpstream: upstream, rateLimitService: &RateLimitService{}}
	account := &Account{
		ID:          104,
		Name:        "anthropic-apikey-pass-conflict",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "upstream-anthropic-key", "base_url": "https://api.anthropic.com"},
		Extra: map[string]any{
			"anthropic_passthrough":      true,
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, parsed)
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), `"type":"invalid_request_error"`)
	require.Contains(t, rec.Body.String(), "passthrough body path conflicts with non-object node: metadata")
	require.Nil(t, upstream.lastReq)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_ForwardCountTokensPreservesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", nil)
	c.Request.Header.Set("Authorization", "Bearer inbound-token")
	c.Request.Header.Set("X-Api-Key", "inbound-api-key")
	c.Request.Header.Set("Cookie", "secret=1")
	collector := &openAIUsageUpstreamRequestCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)

	body := []byte(`{"model":"claude-3-5-sonnet-latest","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}],"thinking":{"type":"enabled"}}`)
	parsed := &ParsedRequest{
		Body:  NewRequestBodyRef(body),
		Model: "claude-3-5-sonnet-latest",
	}

	upstreamRespBody := `{"input_tokens":42}`
	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"x-request-id": []string{"rid-count"},
				"Set-Cookie":   []string{"secret=upstream"},
			},
			Body: io.NopCloser(strings.NewReader(upstreamRespBody)),
		},
	}

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			MaxLineSize: defaultMaxLineSize,
		},
	}
	svc := &GatewayService{
		cfg:                  cfg,
		responseHeaderFilter: compileResponseHeaderFilter(cfg),
		httpUpstream:         upstream,
		rateLimitService:     &RateLimitService{},
	}

	account := &Account{
		ID:          102,
		Name:        "anthropic-apikey-pass-count",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":       "upstream-anthropic-key",
			"base_url":      "https://api.anthropic.com",
			"model_mapping": map[string]any{"claude-3-5-sonnet-latest": "claude-3-opus-20240229"},
		},
		Extra: map[string]any{
			"anthropic_passthrough":      true,
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "header", Mode: "inject", Key: "X-Trace-Id", Value: "trace-ct-1"},
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-1"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	err := svc.ForwardCountTokens(context.Background(), c, account, parsed)
	require.NoError(t, err)

	require.Equal(t, "claude-3-opus-20240229", gjson.GetBytes(upstream.lastBody, "model").String(), "count_tokens 透传模式应应用账号级模型映射")
	require.Equal(t, "user-1", gjson.GetBytes(upstream.lastBody, "metadata.user_id").String())
	require.Equal(t, "upstream-anthropic-key", getHeaderRaw(upstream.lastReq.Header, "x-api-key"))
	require.Equal(t, "trace-ct-1", getHeaderRaw(upstream.lastReq.Header, "X-Trace-Id"))
	require.Empty(t, getHeaderRaw(upstream.lastReq.Header, "authorization"))
	require.Empty(t, getHeaderRaw(upstream.lastReq.Header, "cookie"))
	bodyPreview := requireOpsPreviewString(t, c, "user-1")
	bodyBytes := []byte(bodyPreview)
	require.Equal(t, "user-1", gjson.GetBytes(bodyBytes, "metadata.user_id").String())
	require.Contains(t, collector.body, "user-1")
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, upstreamRespBody, rec.Body.String())
	require.Empty(t, rec.Header().Get("Set-Cookie"))
}

func TestGatewayService_AnthropicAPIKeyPassthrough_BearerAuthScheme(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	collector := &openAIUsageUpstreamRequestCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)
	c.Request.Header.Set("Authorization", "Bearer inbound-token")
	c.Request.Header.Set("X-Api-Key", "inbound-api-key")
	c.Request.Header.Set("Cookie", "secret=1")

	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
		},
	}
	account := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "ollama-key",
			"base_url": "https://ollama.com",
		},
		Extra: map[string]any{
			"anthropic_passthrough":        true,
			"anthropic_apikey_auth_scheme": AnthropicAPIKeyAuthSchemeAuthorizationBearer,
		},
	}

	body := []byte(`{"model":"gpt-oss:20b","messages":[]}`)
	msgReq, _, err := svc.buildUpstreamRequestAnthropicAPIKeyPassthrough(
		context.Background(), c, account, body, body, "ollama-key",
	)
	require.NoError(t, err)
	require.Equal(t, "https://ollama.com/v1/messages?beta=true", msgReq.URL.String())
	wireBody, err := io.ReadAll(msgReq.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"model":"gpt-oss:20b","messages":[]}`, string(wireBody))
	require.Equal(t, "Bearer ollama-key", getHeaderRaw(msgReq.Header, "authorization"))
	require.Empty(t, getHeaderRaw(msgReq.Header, "x-api-key"))
	require.Empty(t, getHeaderRaw(msgReq.Header, "cookie"))

	countReq, _, err := svc.buildCountTokensRequestAnthropicAPIKeyPassthrough(
		context.Background(), c, account, []byte(`{"model":"gpt-oss:20b","messages":[]}`), "ollama-key",
	)
	require.NoError(t, err)
	require.Equal(t, "https://ollama.com/v1/messages/count_tokens?beta=true", countReq.URL.String())
	require.Equal(t, "Bearer ollama-key", getHeaderRaw(countReq.Header, "authorization"))
	require.Empty(t, getHeaderRaw(countReq.Header, "x-api-key"))
	require.Empty(t, getHeaderRaw(countReq.Header, "cookie"))
}

// TestGatewayService_AnthropicAPIKeyPassthrough_ModelMappingEdgeCases 覆盖透传模式下模型映射的各种边界情况
func TestGatewayService_AnthropicAPIKeyPassthrough_ModelMappingEdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		model         string
		modelMapping  map[string]any // nil = 不配置映射
		expectedModel string
		endpoint      string // "messages" or "count_tokens"
	}{
		{
			name:          "Forward: 无映射配置时不改写模型",
			model:         "claude-sonnet-4-20250514",
			modelMapping:  nil,
			expectedModel: "claude-sonnet-4-20250514",
			endpoint:      "messages",
		},
		{
			name:          "Forward: 空映射配置时不改写模型",
			model:         "claude-sonnet-4-20250514",
			modelMapping:  map[string]any{},
			expectedModel: "claude-sonnet-4-20250514",
			endpoint:      "messages",
		},
		{
			name:          "Forward: 模型不在映射表中时不改写",
			model:         "claude-sonnet-4-20250514",
			modelMapping:  map[string]any{"claude-3-haiku-20240307": "claude-3-opus-20240229"},
			expectedModel: "claude-sonnet-4-20250514",
			endpoint:      "messages",
		},
		{
			name:          "Forward: 精确匹配映射应改写模型",
			model:         "claude-sonnet-4-20250514",
			modelMapping:  map[string]any{"claude-sonnet-4-20250514": "claude-sonnet-4-5-20241022"},
			expectedModel: "claude-sonnet-4-5-20241022",
			endpoint:      "messages",
		},
		{
			name:          "Forward: 通配符映射应改写模型",
			model:         "claude-sonnet-4-20250514",
			modelMapping:  map[string]any{"claude-sonnet-4-*": "claude-sonnet-4-5-20241022"},
			expectedModel: "claude-sonnet-4-5-20241022",
			endpoint:      "messages",
		},
		{
			name:          "CountTokens: 无映射配置时不改写模型",
			model:         "claude-sonnet-4-20250514",
			modelMapping:  nil,
			expectedModel: "claude-sonnet-4-20250514",
			endpoint:      "count_tokens",
		},
		{
			name:          "CountTokens: 模型不在映射表中时不改写",
			model:         "claude-sonnet-4-20250514",
			modelMapping:  map[string]any{"claude-3-haiku-20240307": "claude-3-opus-20240229"},
			expectedModel: "claude-sonnet-4-20250514",
			endpoint:      "count_tokens",
		},
		{
			name:          "CountTokens: 精确匹配映射应改写模型",
			model:         "claude-sonnet-4-20250514",
			modelMapping:  map[string]any{"claude-sonnet-4-20250514": "claude-sonnet-4-5-20241022"},
			expectedModel: "claude-sonnet-4-5-20241022",
			endpoint:      "count_tokens",
		},
		{
			name:          "CountTokens: 通配符映射应改写模型",
			model:         "claude-sonnet-4-20250514",
			modelMapping:  map[string]any{"claude-sonnet-4-*": "claude-sonnet-4-5-20241022"},
			expectedModel: "claude-sonnet-4-5-20241022",
			endpoint:      "count_tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)

			body := []byte(`{"model":"` + tt.model + `","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
			parsed := &ParsedRequest{
				Body:  NewRequestBodyRef(body),
				Model: tt.model,
			}

			credentials := map[string]any{
				"api_key":  "upstream-key",
				"base_url": "https://api.anthropic.com",
			}
			if tt.modelMapping != nil {
				credentials["model_mapping"] = tt.modelMapping
			}

			account := &Account{
				ID:          300,
				Name:        "edge-case-test",
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Concurrency: 1,
				Credentials: credentials,
				Extra:       map[string]any{"anthropic_passthrough": true},
				Status:      StatusActive,
				Schedulable: true,
			}

			if tt.endpoint == "messages" {
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
				parsed.Stream = false

				upstreamJSON := `{"id":"msg_1","type":"message","usage":{"input_tokens":5,"output_tokens":3}}`
				upstream := &anthropicHTTPUpstreamRecorder{
					resp: &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body:       io.NopCloser(strings.NewReader(upstreamJSON)),
					},
				}
				svc := &GatewayService{
					cfg:              &config.Config{},
					httpUpstream:     upstream,
					rateLimitService: &RateLimitService{},
				}

				result, err := svc.Forward(context.Background(), c, account, parsed)
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, tt.expectedModel, gjson.GetBytes(upstream.lastBody, "model").String(),
					"Forward 上游请求体中的模型应为: %s", tt.expectedModel)
			} else {
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", nil)

				upstreamRespBody := `{"input_tokens":42}`
				upstream := &anthropicHTTPUpstreamRecorder{
					resp: &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"Content-Type": []string{"application/json"}},
						Body:       io.NopCloser(strings.NewReader(upstreamRespBody)),
					},
				}
				svc := &GatewayService{
					cfg:              &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}},
					httpUpstream:     upstream,
					rateLimitService: &RateLimitService{},
				}

				err := svc.ForwardCountTokens(context.Background(), c, account, parsed)
				require.NoError(t, err)
				require.Equal(t, tt.expectedModel, gjson.GetBytes(upstream.lastBody, "model").String(),
					"CountTokens 上游请求体中的模型应为: %s", tt.expectedModel)
			}
		})
	}
}

// TestGatewayService_AnthropicAPIKeyPassthrough_ModelMappingPreservesOtherFields
// 确保模型映射只替换 model 字段，不影响请求体中的其他字段
func TestGatewayService_AnthropicAPIKeyPassthrough_ModelMappingPreservesOtherFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", nil)

	// 包含复杂字段的请求体：system、thinking、messages
	body := []byte(`{"model":"claude-sonnet-4-20250514","system":[{"type":"text","text":"You are a helpful assistant."}],"messages":[{"role":"user","content":[{"type":"text","text":"hello world"}]}],"thinking":{"type":"enabled","budget_tokens":5000},"max_tokens":1024}`)
	parsed := &ParsedRequest{
		Body:  NewRequestBodyRef(body),
		Model: "claude-sonnet-4-20250514",
	}

	upstreamRespBody := `{"input_tokens":42}`
	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(upstreamRespBody)),
		},
	}

	svc := &GatewayService{
		cfg:              &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}},
		httpUpstream:     upstream,
		rateLimitService: &RateLimitService{},
	}

	account := &Account{
		ID:          301,
		Name:        "preserve-fields-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":       "upstream-key",
			"base_url":      "https://api.anthropic.com",
			"model_mapping": map[string]any{"claude-sonnet-4-20250514": "claude-sonnet-4-5-20241022"},
		},
		Extra:       map[string]any{"anthropic_passthrough": true},
		Status:      StatusActive,
		Schedulable: true,
	}

	err := svc.ForwardCountTokens(context.Background(), c, account, parsed)
	require.NoError(t, err)

	sentBody := upstream.lastBody
	require.Equal(t, "claude-sonnet-4-5-20241022", gjson.GetBytes(sentBody, "model").String(), "model 应被映射")
	require.Equal(t, "You are a helpful assistant.", gjson.GetBytes(sentBody, "system.0.text").String(), "system 字段不应被修改")
	require.Equal(t, "hello world", gjson.GetBytes(sentBody, "messages.0.content.0.text").String(), "messages 字段不应被修改")
	require.Equal(t, "enabled", gjson.GetBytes(sentBody, "thinking.type").String(), "thinking 字段不应被修改")
	require.Equal(t, int64(5000), gjson.GetBytes(sentBody, "thinking.budget_tokens").Int(), "thinking.budget_tokens 不应被修改")
	require.False(t, gjson.GetBytes(sentBody, "max_tokens").Exists(),
		"max_tokens 作为生成参数应被 count_tokens 过滤剥离")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_CountTokensFiltersGenerationFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", nil)

	body := []byte(`{"model":"claude-sonnet-4-20250514","system":[{"type":"text","text":"sys"}],"messages":[{"role":"user","content":"hello"}],"tools":[{"name":"tool","input_schema":{"type":"object"}}],"temperature":0.7,"top_p":0.9,"top_k":40,"stream":true,"stop_sequences":["END"],"max_tokens":1024,"thinking":{"type":"enabled","budget_tokens":5000}}`)
	parsed := &ParsedRequest{
		Body:  NewRequestBodyRef(body),
		Model: "claude-sonnet-4-20250514",
	}

	upstreamRespBody := `{"input_tokens":42}`
	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(upstreamRespBody)),
		},
	}

	svc := &GatewayService{
		cfg:              &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}},
		httpUpstream:     upstream,
		rateLimitService: &RateLimitService{},
	}

	account := &Account{
		ID:          302,
		Name:        "count-token-filter-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "upstream-key",
			"base_url": "https://api.anthropic.com",
		},
		Extra:       map[string]any{"anthropic_passthrough": true},
		Status:      StatusActive,
		Schedulable: true,
	}

	err := svc.ForwardCountTokens(context.Background(), c, account, parsed)
	require.NoError(t, err)

	sentBody := upstream.lastBody
	require.False(t, gjson.GetBytes(sentBody, "temperature").Exists())
	require.False(t, gjson.GetBytes(sentBody, "top_p").Exists())
	require.False(t, gjson.GetBytes(sentBody, "top_k").Exists())
	require.False(t, gjson.GetBytes(sentBody, "stream").Exists())
	require.False(t, gjson.GetBytes(sentBody, "stop_sequences").Exists())
	require.Equal(t, "claude-sonnet-4-20250514", gjson.GetBytes(sentBody, "model").String())
	require.Equal(t, "sys", gjson.GetBytes(sentBody, "system.0.text").String())
	require.Equal(t, "hello", gjson.GetBytes(sentBody, "messages.0.content").String())
	require.Equal(t, "tool", gjson.GetBytes(sentBody, "tools.0.name").String())
	require.False(t, gjson.GetBytes(sentBody, "max_tokens").Exists(),
		"count_tokens 请求不得携带生成参数 max_tokens")
	require.Equal(t, "enabled", gjson.GetBytes(sentBody, "thinking.type").String())
}

// TestGatewayService_AnthropicAPIKeyPassthrough_EmptyModelSkipsMapping
// 确保空模型名不会触发映射逻辑
func TestGatewayService_AnthropicAPIKeyPassthrough_EmptyModelSkipsMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", nil)

	body := []byte(`{"messages":[{"role":"user","content":"hello"}]}`)
	parsed := &ParsedRequest{
		Body:  NewRequestBodyRef(body),
		Model: "", // 空模型
	}

	upstreamRespBody := `{"input_tokens":10}`
	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(upstreamRespBody)),
		},
	}

	svc := &GatewayService{
		cfg:              &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}},
		httpUpstream:     upstream,
		rateLimitService: &RateLimitService{},
	}

	account := &Account{
		ID:          302,
		Name:        "empty-model-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":       "upstream-key",
			"base_url":      "https://api.anthropic.com",
			"model_mapping": map[string]any{"*": "claude-3-opus-20240229"},
		},
		Extra:       map[string]any{"anthropic_passthrough": true},
		Status:      StatusActive,
		Schedulable: true,
	}

	err := svc.ForwardCountTokens(context.Background(), c, account, parsed)
	require.NoError(t, err)
	// 空模型名时，body 应原样透传，不应触发映射
	require.Equal(t, body, upstream.lastBody, "空模型名时请求体不应被修改")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_CountTokens404PassthroughNotError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		statusCode      int
		respBody        string
		wantPassthrough bool
	}{
		{
			name:            "404 endpoint not found passes through as 404",
			statusCode:      http.StatusNotFound,
			respBody:        `{"error":{"message":"Not found: /v1/messages/count_tokens","type":"not_found_error"}}`,
			wantPassthrough: true,
		},
		{
			name:            "404 generic not found does not passthrough",
			statusCode:      http.StatusNotFound,
			respBody:        `{"error":{"message":"resource not found","type":"not_found_error"}}`,
			wantPassthrough: false,
		},
		{
			name:            "400 Invalid URL does not passthrough",
			statusCode:      http.StatusBadRequest,
			respBody:        `{"error":{"message":"Invalid URL (POST /v1/messages/count_tokens)","type":"invalid_request_error"}}`,
			wantPassthrough: false,
		},
		{
			name:            "400 model error does not passthrough",
			statusCode:      http.StatusBadRequest,
			respBody:        `{"error":{"message":"model not found: claude-unknown","type":"invalid_request_error"}}`,
			wantPassthrough: false,
		},
		{
			name:            "500 internal error does not passthrough",
			statusCode:      http.StatusInternalServerError,
			respBody:        `{"error":{"message":"internal error","type":"api_error"}}`,
			wantPassthrough: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", nil)

			body := []byte(`{"model":"claude-sonnet-4-5-20250929","messages":[{"role":"user","content":"hi"}]}`)
			parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-sonnet-4-5-20250929"}

			upstream := &anthropicHTTPUpstreamRecorder{
				resp: &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.respBody)),
				},
			}

			svc := &GatewayService{
				cfg: &config.Config{
					Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
				},
				httpUpstream:     upstream,
				rateLimitService: nil,
			}

			account := &Account{
				ID:          200,
				Name:        "proxy-acc",
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Concurrency: 1,
				Credentials: map[string]any{
					"api_key":  "sk-proxy",
					"base_url": "https://proxy.example.com",
				},
				Extra:       map[string]any{"anthropic_passthrough": true},
				Status:      StatusActive,
				Schedulable: true,
			}

			err := svc.ForwardCountTokens(context.Background(), c, account, parsed)

			if tt.wantPassthrough {
				// 返回 nil（不记录为错误），HTTP 状态码 404 + Anthropic 错误体
				require.NoError(t, err)
				require.Equal(t, http.StatusNotFound, rec.Code)
				var errResp map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
				require.Equal(t, "error", errResp["type"])
				errObj, ok := errResp["error"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, "not_found_error", errObj["type"])
			} else {
				require.Error(t, err)
				require.Equal(t, tt.statusCode, rec.Code)
			}
		})
	}
}

func TestGatewayService_AnthropicAPIKeyPassthrough_BuildRequestRejectsInvalidBaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{
					Enabled: false,
				},
			},
		},
	}
	account := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "k",
			"base_url": "://invalid-url",
		},
	}

	_, _, err := svc.buildUpstreamRequestAnthropicAPIKeyPassthrough(context.Background(), c, account, []byte(`{}`), []byte(`{}`), "k")
	require.Error(t, err)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StripsDeferredToolCacheControl(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	svc := &GatewayService{cfg: &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}}}
	account := &Account{Platform: PlatformAnthropic, Type: AccountTypeAPIKey}
	body := []byte(`{"tools":[{"name":"deferred","custom":{"defer_loading":true},"cache_control":{"type":"ephemeral"}},{"name":"top-level-deferred","defer_loading":true,"cache_control":{"type":"ephemeral"}},{"name":"ordinary","defer_loading":false,"cache_control":{"type":"ephemeral"}},{"name":"malformed","defer_loading":"true","cache_control":{"type":"ephemeral"}}]}`)

	_, wireBody, err := svc.buildUpstreamRequestAnthropicAPIKeyPassthrough(context.Background(), c, account, body, body, "k")
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(wireBody, "tools.0.cache_control").Exists())
	require.False(t, gjson.GetBytes(wireBody, "tools.1.cache_control").Exists())
	require.True(t, gjson.GetBytes(wireBody, "tools.2.cache_control").Exists())
	require.True(t, gjson.GetBytes(wireBody, "tools.3.cache_control").Exists())

	countReq, _, err := svc.buildCountTokensRequestAnthropicAPIKeyPassthrough(context.Background(), c, account, body, "k")
	require.NoError(t, err)
	countBody, err := io.ReadAll(countReq.Body)
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(countBody, "tools.0.cache_control").Exists())
	require.False(t, gjson.GetBytes(countBody, "tools.1.cache_control").Exists())
	require.True(t, gjson.GetBytes(countBody, "tools.2.cache_control").Exists())
	require.True(t, gjson.GetBytes(countBody, "tools.3.cache_control").Exists())
}

func TestGatewayService_AnthropicOAuth_NotAffectedByAPIKeyPassthroughToggle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
		},
	}
	account := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"anthropic_passthrough": true,
		},
	}

	require.False(t, account.IsAnthropicAPIKeyPassthroughEnabled())

	req, _, err := svc.buildUpstreamRequest(context.Background(), c, account, []byte(`{"model":"claude-3-7-sonnet-20250219"}`), "oauth-token", "oauth", "claude-3-7-sonnet-20250219", true, false)
	require.NoError(t, err)
	require.Equal(t, "Bearer oauth-token", getHeaderRaw(req.Header, "authorization"))
	require.Contains(t, getHeaderRaw(req.Header, "anthropic-beta"), claude.BetaOAuth, "OAuth 链路仍应按原逻辑补齐 oauth beta")
}

func TestGatewayService_AnthropicOAuthMimic_RewritesSystemWithBillingBlock(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                       string
		body                       string
		wantModel                  string
		wantOriginalSystem         string
		wantOriginalSystemCacheTTL string
		wantMetadataUserID         string
	}{
		{
			name:               "sonnet system array",
			body:               `{"model":"claude-3-5-sonnet-latest","system":[{"type":"text","text":"x-anthropic-billing-header keep"}],"messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`,
			wantModel:          "claude-3-5-sonnet-latest",
			wantOriginalSystem: "x-anthropic-billing-header keep",
		},
		{
			name:               "sonnet system string",
			body:               `{"model":"claude-3-5-sonnet-latest","system":"x-anthropic-billing-header keep","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`,
			wantModel:          "claude-3-5-sonnet-latest",
			wantOriginalSystem: "x-anthropic-billing-header keep",
		},
		{
			name:                       "haiku full mimicry",
			body:                       `{"model":"claude-haiku-4-5","metadata":{"user_id":"pi-session-metadata"},"system":[{"type":"text","text":"Pi project instructions","cache_control":{"type":"ephemeral","ttl":"1h"}}],"thinking":{"type":"enabled","budget_tokens":1024},"messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`,
			wantModel:                  "claude-haiku-4-5-20251001",
			wantOriginalSystem:         "Pi project instructions",
			wantOriginalSystemCacheTTL: "1h",
			wantMetadataUserID:         "pi-session-metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
			c.Request.Header.Set("User-Agent", "pi/0.51.0")
			c.Request.Header.Set("Anthropic-Beta", "client-only-beta")

			parsed, err := ParseGatewayRequest(NewRequestBodyRef([]byte(tt.body)), PlatformAnthropic)
			require.NoError(t, err)

			upstream := &anthropicHTTPUpstreamRecorder{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
						"x-request-id": []string{"rid-oauth-mimic"},
					},
					Body: io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-haiku-4-5-20251001","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":12,"output_tokens":7}}`)),
				},
			}

			cfg := &config.Config{
				Gateway: config.GatewayConfig{
					MaxLineSize: defaultMaxLineSize,
				},
			}
			svc := &GatewayService{
				cfg:                  cfg,
				responseHeaderFilter: compileResponseHeaderFilter(cfg),
				httpUpstream:         upstream,
				rateLimitService:     &RateLimitService{},
				deferredService:      &DeferredService{},
			}

			account := &Account{
				ID:          301,
				Name:        "anthropic-oauth-mimic",
				Platform:    PlatformAnthropic,
				Type:        AccountTypeOAuth,
				Concurrency: 1,
				Credentials: map[string]any{
					"access_token": "oauth-token",
				},
				Status:      StatusActive,
				Schedulable: true,
			}

			result, err := svc.Forward(context.Background(), c, account, parsed)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, upstream.lastReq)
			require.Equal(t, "Bearer oauth-token", getHeaderRaw(upstream.lastReq.Header, "authorization"))
			finalBeta := getHeaderRaw(upstream.lastReq.Header, "anthropic-beta")
			for _, beta := range claude.FullClaudeCodeMimicryBetas() {
				require.Truef(t, anthropicBetaTokensContains(finalBeta, beta), "missing mimic beta %s", beta)
			}
			require.False(t, anthropicBetaTokensContains(finalBeta, "client-only-beta"))
			for key, value := range claude.DefaultHeaders {
				require.Equal(t, value, getHeaderRaw(upstream.lastReq.Header, key), "mimic fingerprint header %s", key)
			}
			require.NotEmpty(t, getHeaderRaw(upstream.lastReq.Header, "x-client-request-id"))

			require.Equal(t, tt.wantModel, gjson.GetBytes(upstream.lastBody, "model").String())
			system := gjson.GetBytes(upstream.lastBody, "system")
			require.True(t, system.Exists())
			require.True(t, system.IsArray(), "system should be an array")
			arr := system.Array()
			require.Len(t, arr, 2, "system array should have billing block + cc prompt block")

			billingText := arr[0].Get("text").String()
			require.Contains(t, billingText, "x-anthropic-billing-header:")
			require.Contains(t, billingText, "cc_version="+claude.CLICurrentVersion+".")
			require.Contains(t, billingText, "cc_entrypoint=cli;")

			require.Equal(t, claudeCodeSystemPrompt, arr[1].Get("text").String())
			require.Equal(t, "ephemeral", arr[1].Get("cache_control.type").String())

			// 原始 system prompt 应迁移至 messages 中。
			messages := gjson.GetBytes(upstream.lastBody, "messages")
			require.True(t, messages.IsArray())
			firstMsg := messages.Array()[0]
			require.Equal(t, "user", firstMsg.Get("role").String())
			require.Contains(t, firstMsg.Get("content.0.text").String(), tt.wantOriginalSystem)
			if tt.wantOriginalSystemCacheTTL != "" {
				require.Equal(t, "ephemeral", firstMsg.Get("content.0.cache_control.type").String())
				require.Equal(t, tt.wantOriginalSystemCacheTTL, firstMsg.Get("content.0.cache_control.ttl").String())
			} else {
				require.False(t, firstMsg.Get("content.0.cache_control").Exists())
			}

			if tt.wantMetadataUserID != "" {
				require.Equal(t, tt.wantMetadataUserID, gjson.GetBytes(upstream.lastBody, "metadata.user_id").String())
				require.True(t, gjson.GetBytes(upstream.lastBody, "context_management").Exists())
			}
		})
	}
}

func TestGatewayService_AnthropicOAuthRealClaudeCodeHaiku_PreservesClientHeadersAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	metadataUserID := FormatMetadataUserID(
		"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		"550e8400-e29b-41d4-a716-446655440000",
		"123e4567-e89b-42d3-a456-426614174000",
		claude.CLICurrentVersion,
	)
	body := []byte(`{"model":"claude-haiku-4-5-20251001","metadata":{"user_id":` + strconvQuote(metadataUserID) + `},"system":[{"type":"text","text":"Client-owned Claude Code system","cache_control":{"type":"ephemeral"}}],"context_management":{"edits":[{"type":"clear_thinking_20251015","keep":"all"}]},"messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	parsed, err := ParseGatewayRequest(NewRequestBodyRef(body), PlatformAnthropic)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Request.Header.Set("User-Agent", "claude-cli/"+claude.CLICurrentVersion+" (external, cli)")
	c.Request.Header.Set("X-Stainless-Package-Version", "real-client-package")
	clientBeta := strings.Join([]string{
		claude.BetaClaudeCode,
		claude.BetaOAuth,
		claude.BetaInterleavedThinking,
		claude.BetaContextManagement,
	}, ",")
	c.Request.Header.Set("Anthropic-Beta", clientBeta)

	upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"msg_real_cc","type":"message","role":"assistant","model":"claude-haiku-4-5-20251001","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":12,"output_tokens":7}}`)),
	}}
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	svc := &GatewayService{
		cfg:                  cfg,
		responseHeaderFilter: compileResponseHeaderFilter(cfg),
		httpUpstream:         upstream,
		rateLimitService:     &RateLimitService{},
		deferredService:      &DeferredService{},
	}
	account := &Account{
		ID: 302, Name: "anthropic-real-cc", Platform: PlatformAnthropic, Type: AccountTypeOAuth, Concurrency: 1,
		Credentials: map[string]any{"access_token": "oauth-token"}, Status: StatusActive, Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, c.Request.Header.Get("User-Agent"), getHeaderRaw(upstream.lastReq.Header, "User-Agent"))
	require.Equal(t, "real-client-package", getHeaderRaw(upstream.lastReq.Header, "X-Stainless-Package-Version"))
	require.Equal(t, clientBeta, getHeaderRaw(upstream.lastReq.Header, "anthropic-beta"))
	require.Empty(t, getHeaderRaw(upstream.lastReq.Header, "x-client-request-id"), "真实 CC 不应被强制写入 mimic request id")
	require.Equal(t, gjson.GetBytes(body, "system").Raw, gjson.GetBytes(upstream.lastBody, "system").Raw)
	require.Equal(t, gjson.GetBytes(body, "messages").Raw, gjson.GetBytes(upstream.lastBody, "messages").Raw)
	require.Equal(t, metadataUserID, gjson.GetBytes(upstream.lastBody, "metadata.user_id").String())
	require.True(t, gjson.GetBytes(upstream.lastBody, "context_management").Exists())
	require.NotContains(t, string(upstream.lastBody), "x-anthropic-billing-header:")
}

func TestGatewayService_AnthropicOAuth_SystemPromptInjectionCanBeDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetGatewayForwardingSettingsCacheForTest(t)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	body := []byte(`{"model":"claude-3-5-sonnet-latest","system":"Original system prompt","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	parsed, err := ParseGatewayRequest(NewRequestBodyRef(body), PlatformAnthropic)
	require.NoError(t, err)

	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"x-request-id": []string{"rid-oauth-no-system-injection"},
			},
			Body: io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-3-5-sonnet-20241022","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":12,"output_tokens":7}}`)),
		},
	}

	cfg := &config.Config{
		Gateway: config.GatewayConfig{
			MaxLineSize: defaultMaxLineSize,
		},
	}
	settingService := NewSettingService(&gatewayTTLSettingRepo{data: map[string]string{
		SettingKeyEnableClaudeOAuthSystemPromptInjection: "false",
	}}, cfg)
	svc := &GatewayService{
		cfg:                  cfg,
		responseHeaderFilter: compileResponseHeaderFilter(cfg),
		httpUpstream:         upstream,
		rateLimitService:     &RateLimitService{},
		deferredService:      &DeferredService{},
		settingService:       settingService,
	}

	account := &Account{
		ID:          302,
		Name:        "anthropic-oauth-no-system-injection",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "oauth-token",
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)

	system := gjson.GetBytes(upstream.lastBody, "system")
	require.True(t, system.Exists())
	require.Equal(t, "Original system prompt", system.String())
	require.NotContains(t, string(upstream.lastBody), "x-anthropic-billing-header:")
	require.NotContains(t, string(upstream.lastBody), "[System Instructions]")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StreamingStillCollectsUsageAfterClientDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Use a canceled context recorder to simulate client disconnect behavior.
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				MaxLineSize: defaultMaxLineSize,
			},
		},
		rateLimitService: &RateLimitService{},
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"message_start","message":{"usage":{"input_tokens":11}}}`,
			"",
			`data: {"type":"message_delta","usage":{"output_tokens":5}}`,
			"",
			"data: [DONE]",
			"",
		}, "\n"))),
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "claude-3-7-sonnet-20250219")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.Equal(t, 11, result.usage.InputTokens)
	require.Equal(t, 5, result.usage.OutputTokens)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_MissingTerminalEventReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				MaxLineSize: defaultMaxLineSize,
			},
		},
		rateLimitService: &RateLimitService{},
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"message_start","message":{"usage":{"input_tokens":11}}}`,
			"",
			`data: {"type":"message_delta","usage":{"output_tokens":5}}`,
			"",
		}, "\n"))),
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "claude-3-7-sonnet-20250219")
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing terminal event")
	require.NotNil(t, result)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_ForwardDirect_NonStreamingSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	body := []byte(`{"model":"claude-3-5-sonnet-latest","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	upstreamJSON := `{"id":"msg_1","type":"message","usage":{"input_tokens":12,"output_tokens":7,"cache_creation":{"ephemeral_5m_input_tokens":2,"ephemeral_1h_input_tokens":3},"cached_tokens":4}}`
	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"x-request-id": []string{"rid-nonstream"},
			},
			Body: io.NopCloser(strings.NewReader(upstreamJSON)),
		},
	}
	svc := &GatewayService{
		cfg:              &config.Config{},
		httpUpstream:     upstream,
		rateLimitService: &RateLimitService{},
	}

	result, err := svc.forwardAnthropicAPIKeyPassthrough(context.Background(), c, newAnthropicAPIKeyAccountForTest(), body, "claude-3-5-sonnet-latest", "claude-3-5-sonnet-latest", false, time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 12, result.Usage.InputTokens)
	require.Equal(t, 7, result.Usage.OutputTokens)
	require.Equal(t, 5, result.Usage.CacheCreationInputTokens)
	require.Equal(t, 4, result.Usage.CacheReadInputTokens)
	require.Equal(t, upstreamJSON, rec.Body.String())
}

func TestGatewayService_AnthropicAPIKeyPassthrough_ForwardDirect_InvalidTokenType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	account := &Account{
		ID:       202,
		Name:     "anthropic-oauth",
		Platform: PlatformAnthropic,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "oauth-token",
		},
	}
	svc := &GatewayService{}

	result, err := svc.forwardAnthropicAPIKeyPassthrough(context.Background(), c, account, []byte(`{}`), "claude-3-5-sonnet-latest", "claude-3-5-sonnet-latest", false, time.Now())
	require.Nil(t, result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires apikey token")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_ForwardDirect_UpstreamRequestError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	upstream := &anthropicHTTPUpstreamRecorder{
		err: errors.New("dial tcp timeout"),
	}
	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
		},
		httpUpstream: upstream,
	}
	account := newAnthropicAPIKeyAccountForTest()

	result, err := svc.forwardAnthropicAPIKeyPassthrough(context.Background(), c, account, []byte(`{"model":"x"}`), "x", "x", false, time.Now())
	require.Nil(t, result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "upstream request failed")
	require.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_ForwardDirect_EmptyResponseBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"x-request-id": []string{"rid-empty-body"}},
			Body:       nil,
		},
	}
	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
		},
		httpUpstream: upstream,
	}

	result, err := svc.forwardAnthropicAPIKeyPassthrough(context.Background(), c, newAnthropicAPIKeyAccountForTest(), []byte(`{"model":"x"}`), "x", "x", false, time.Now())
	require.Nil(t, result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty response")
}

func TestExtractAnthropicSSEDataLine(t *testing.T) {
	t.Run("valid data line with spaces", func(t *testing.T) {
		data, ok := extractAnthropicSSEDataLine("data:   {\"type\":\"message_start\"}")
		require.True(t, ok)
		require.Equal(t, `{"type":"message_start"}`, data)
	})

	t.Run("non data line", func(t *testing.T) {
		data, ok := extractAnthropicSSEDataLine("event: message_start")
		require.False(t, ok)
		require.Empty(t, data)
	})
}

func TestGatewayService_ParseSSEUsagePassthrough_MessageStartFallbacks(t *testing.T) {
	usage := &ClaudeUsage{}
	data := `{"type":"message_start","message":{"usage":{"input_tokens":12,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"cached_tokens":9,"cache_creation":{"ephemeral_5m_input_tokens":3,"ephemeral_1h_input_tokens":4}}}}`

	parseSSEUsagePassthrough(data, usage)

	require.Equal(t, 12, usage.InputTokens)
	require.Equal(t, 9, usage.CacheReadInputTokens, "应兼容 cached_tokens 字段")
	require.Equal(t, 7, usage.CacheCreationInputTokens, "聚合字段为空时应从 5m/1h 明细回填")
	require.Equal(t, 3, usage.CacheCreation5mTokens)
	require.Equal(t, 4, usage.CacheCreation1hTokens)
}

func TestGatewayService_ParseSSEUsagePassthrough_MessageDeltaSelectiveOverwrite(t *testing.T) {
	usage := &ClaudeUsage{}
	start := `{"type":"message_start","message":{"usage":{"input_tokens":10,"cache_creation_input_tokens":463184,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":463184}}}}`
	parseSSEUsagePassthrough(start, usage)

	data := `{"type":"message_delta","usage":{"input_tokens":0,"output_tokens":5,"cache_creation_input_tokens":463184,"cache_read_input_tokens":0,"cached_tokens":11,"cache_creation":{"ephemeral_5m_input_tokens":463184,"ephemeral_1h_input_tokens":0}}}`

	parseSSEUsagePassthrough(data, usage)

	require.Equal(t, 10, usage.InputTokens, "message_delta 中 0 值不应覆盖已有 input_tokens")
	require.Equal(t, 5, usage.OutputTokens)
	require.Equal(t, 463184, usage.CacheCreationInputTokens)
	require.Equal(t, 11, usage.CacheReadInputTokens, "cache_read_input_tokens 为空时应回退到 cached_tokens")
	require.Equal(t, 463184, usage.CacheCreation5mTokens)
	require.Equal(t, 0, usage.CacheCreation1hTokens)
}

func TestGatewayService_ParseSSEUsagePassthrough_NoopCases(t *testing.T) {

	usage := &ClaudeUsage{InputTokens: 3}
	parseSSEUsagePassthrough("", usage)
	require.Equal(t, 3, usage.InputTokens)

	parseSSEUsagePassthrough("[DONE]", usage)
	require.Equal(t, 3, usage.InputTokens)

	parseSSEUsagePassthrough("not-json", usage)
	require.Equal(t, 3, usage.InputTokens)

	// nil usage 不应 panic
	parseSSEUsagePassthrough(`{"type":"message_start"}`, nil)
}

func TestGatewayService_ParseSSEUsagePassthrough_FallbackFromUsageNode(t *testing.T) {
	usage := &ClaudeUsage{}
	data := `{"type":"content_block_delta","usage":{"cached_tokens":6,"cache_creation":{"ephemeral_5m_input_tokens":2,"ephemeral_1h_input_tokens":1}}}`

	parseSSEUsagePassthrough(data, usage)

	require.Equal(t, 6, usage.CacheReadInputTokens)
	require.Equal(t, 3, usage.CacheCreationInputTokens)
}

func TestParseClaudeUsageFromResponseBody(t *testing.T) {
	t.Run("empty or missing usage", func(t *testing.T) {
		got := parseClaudeUsageFromResponseBody(nil)
		require.NotNil(t, got)
		require.Equal(t, 0, got.InputTokens)

		got = parseClaudeUsageFromResponseBody([]byte(`{"id":"x"}`))
		require.NotNil(t, got)
		require.Equal(t, 0, got.OutputTokens)
	})

	t.Run("parse all usage fields and fallback", func(t *testing.T) {
		body := []byte(`{"usage":{"input_tokens":21,"output_tokens":34,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"cached_tokens":13,"cache_creation":{"ephemeral_5m_input_tokens":5,"ephemeral_1h_input_tokens":8}}}`)
		got := parseClaudeUsageFromResponseBody(body)
		require.Equal(t, 21, got.InputTokens)
		require.Equal(t, 34, got.OutputTokens)
		require.Equal(t, 13, got.CacheReadInputTokens, "cache_read_input_tokens 为空时应回退 cached_tokens")
		require.Equal(t, 13, got.CacheCreationInputTokens, "聚合字段为空时应由 5m/1h 回填")
		require.Equal(t, 5, got.CacheCreation5mTokens)
		require.Equal(t, 8, got.CacheCreation1hTokens)
	})

	t.Run("keep explicit aggregate values", func(t *testing.T) {
		body := []byte(`{"usage":{"input_tokens":1,"output_tokens":2,"cache_creation_input_tokens":9,"cache_read_input_tokens":7,"cached_tokens":99,"cache_creation":{"ephemeral_5m_input_tokens":4,"ephemeral_1h_input_tokens":5}}}`)
		got := parseClaudeUsageFromResponseBody(body)
		require.Equal(t, 9, got.CacheCreationInputTokens, "已显式提供聚合字段时不应被明细覆盖")
		require.Equal(t, 7, got.CacheReadInputTokens, "已显式提供 cache_read_input_tokens 时不应回退 cached_tokens")
	})
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StreamingErrTooLong(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				MaxLineSize: 32,
			},
		},
	}

	// Scanner 初始缓冲为 64KB，构造更长单行触发 bufio.ErrTooLong。
	longLine := "data: " + strings.Repeat("x", 80*1024)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(longLine)),
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 2}, time.Now(), "claude-3-7-sonnet-20250219")
	require.Error(t, err)
	require.ErrorIs(t, err, bufio.ErrTooLong)
	require.NotNil(t, result)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StreamingDataIntervalTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				StreamDataIntervalTimeout: 1,
				MaxLineSize:               defaultMaxLineSize,
			},
		},
		rateLimitService: &RateLimitService{},
	}

	pr, pw := io.Pipe()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       pr,
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 5}, time.Now(), "claude-3-7-sonnet-20250219")
	_ = pw.Close()
	_ = pr.Close()

	require.Error(t, err)
	require.Contains(t, err.Error(), "stream data interval timeout")
	require.NotNil(t, result)
	require.False(t, result.clientDisconnect)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StreamingSendsKeepaliveDuringIdle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				StreamKeepaliveInterval: 1,
				MaxLineSize:             defaultMaxLineSize,
			},
		},
		rateLimitService: &RateLimitService{},
	}

	pr, pw := io.Pipe()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       pr,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(1200 * time.Millisecond)
		_, _ = pw.Write([]byte(strings.Join([]string{
			`data: {"type":"message_start","message":{"usage":{"input_tokens":3}}}`,
			"",
			`data: {"type":"message_delta","usage":{"output_tokens":2}}`,
			"",
			"data: [DONE]",
			"",
		}, "\n")))
		_ = pw.Close()
	}()

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 8}, time.Now(), "claude-3-7-sonnet-20250219")
	_ = pr.Close()
	<-done

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, rec.Body.String(), "event: ping\ndata: {\"type\": \"ping\"}\n\n")
	require.Contains(t, rec.Body.String(), "data: [DONE]")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StreamingKeepaliveDoesNotInterleavePartialEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				StreamKeepaliveInterval: 1,
				MaxLineSize:             defaultMaxLineSize,
			},
		},
		rateLimitService: &RateLimitService{},
	}

	pr, pw := io.Pipe()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       pr,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = pw.Write([]byte(`data: {"type":"message_start","message":{"usage":{"input_tokens":4}}}` + "\n"))
		time.Sleep(1200 * time.Millisecond)
		_, _ = pw.Write([]byte("\n"))
		_, _ = pw.Write([]byte("data: [DONE]\n\n"))
		_ = pw.Close()
	}()

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 9}, time.Now(), "claude-3-7-sonnet-20250219")
	_ = pr.Close()
	<-done

	require.NoError(t, err)
	require.NotNil(t, result)
	body := rec.Body.String()
	require.NotContains(t, body, `data: {"type":"message_start","message":{"usage":{"input_tokens":4}}}`+"\n"+"event: ping")
	require.NotContains(t, body, "event: ping")
	require.Contains(t, body, "data: [DONE]")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StreamingReadError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				MaxLineSize: defaultMaxLineSize,
			},
		},
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: &streamReadCloser{
			err: io.ErrUnexpectedEOF,
		},
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 6}, time.Now(), "claude-3-7-sonnet-20250219")
	require.Error(t, err)
	require.Contains(t, err.Error(), "stream read error")
	require.NotNil(t, result)
	require.False(t, result.clientDisconnect)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StreamingTimeoutAfterClientDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Writer = &failWriteResponseWriter{ResponseWriter: c.Writer}

	svc := &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				StreamDataIntervalTimeout: 1,
				MaxLineSize:               defaultMaxLineSize,
			},
		},
		rateLimitService: &RateLimitService{},
	}

	pr, pw := io.Pipe()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       pr,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = pw.Write([]byte(`data: {"type":"message_start","message":{"usage":{"input_tokens":9}}}` + "\n"))
		// 保持上游连接静默，触发数据间隔超时分支。
		time.Sleep(1500 * time.Millisecond)
		_ = pw.Close()
	}()

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 7}, time.Now(), "claude-3-7-sonnet-20250219")
	_ = pr.Close()
	<-done

	require.Error(t, err)
	require.Contains(t, err.Error(), "stream usage incomplete after timeout")
	require.NotNil(t, result)
	require.True(t, result.clientDisconnect)
	require.Equal(t, 9, result.usage.InputTokens)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StreamingContextCanceled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				MaxLineSize: defaultMaxLineSize,
			},
		},
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: &streamReadCloser{
			err: context.Canceled,
		},
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 3}, time.Now(), "claude-3-7-sonnet-20250219")
	require.Error(t, err)
	require.Contains(t, err.Error(), "stream usage incomplete")
	require.NotNil(t, result)
	require.True(t, result.clientDisconnect)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StreamingUpstreamReadErrorAfterClientDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Writer = &failWriteResponseWriter{ResponseWriter: c.Writer}

	svc := &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				MaxLineSize: defaultMaxLineSize,
			},
		},
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: &streamReadCloser{
			payload: []byte(`data: {"type":"message_start","message":{"usage":{"input_tokens":8}}}` + "\n\n"),
			err:     io.ErrUnexpectedEOF,
		},
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 4}, time.Now(), "claude-3-7-sonnet-20250219")
	require.Error(t, err)
	require.Contains(t, err.Error(), "stream usage incomplete after disconnect")
	require.NotNil(t, result)
	require.True(t, result.clientDisconnect)
	require.Equal(t, 8, result.usage.InputTokens)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_OpsBodyReflectsPassthroughInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	collector := &openAIUsageUpstreamRequestCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)

	inputBody := []byte(`{"model":"claude-3-5-sonnet-latest","messages":[{"role":"user","content":"hi"}]}`)
	parsed := &ParsedRequest{Body: NewRequestBodyRef(inputBody), Model: "claude-3-5-sonnet-latest"}

	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","usage":{"input_tokens":5,"output_tokens":3}}`)),
		},
	}
	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
		},
		httpUpstream:     upstream,
		rateLimitService: &RateLimitService{},
	}
	account := &Account{
		ID:          501,
		Name:        "anthropic-apikey-ops-body-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "upstream-anthropic-key",
			"base_url": "https://api.anthropic.com",
		},
		Extra: map[string]any{
			"anthropic_passthrough":      true,
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "inject", Key: "metadata.ops_tag", Value: "injected-value"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	_, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)

	bodyPreview := requireOpsPreviewString(t, c, "injected-value")
	bodyBytes := []byte(bodyPreview)
	require.Equal(t, "injected-value", gjson.GetBytes(bodyBytes, "metadata.ops_tag").String(),
		"ops upstream request body should reflect passthrough-injected fields")
	require.Contains(t, collector.body, "injected-value")
}

func TestGatewayService_RetryPreviewUsesFinalBuilderBody(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		responses []string
	}{
		{
			name: "two-stage signature retry",
			body: `{"model":"claude-sonnet-4-5","max_tokens":1024,"thinking":{"type":"enabled","budget_tokens":1024},"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"work","signature":"stale"},{"type":"tool_use","id":"toolu_1","name":"lookup","input":{}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}]}`,
			responses: []string{
				`{"type":"error","error":{"type":"invalid_request_error","message":"Invalid signature in thinking block"}}`,
				`{"type":"error","error":{"type":"invalid_request_error","message":"Invalid signature in tool_use block"}}`,
				`{"id":"msg_retry","type":"message","content":[],"usage":{"input_tokens":1,"output_tokens":1}}`,
			},
		},
		{
			name: "budget retry",
			body: `{"model":"claude-sonnet-4-5","max_tokens":512,"thinking":{"type":"enabled","budget_tokens":512},"messages":[{"role":"user","content":"hello"}]}`,
			responses: []string{
				`{"type":"error","error":{"type":"invalid_request_error","message":"thinking budget_tokens input should be greater than or equal to 1024"}}`,
				`{"id":"msg_budget","type":"message","content":[],"usage":{"input_tokens":1,"output_tokens":1}}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			body := []byte(tt.body)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
			collector := &openAIUsageUpstreamRequestCollector{}
			c.Set(UsageDetailCaptureContextKey, collector)

			responses := make([]*http.Response, 0, len(tt.responses))
			for i, responseBody := range tt.responses {
				status := http.StatusBadRequest
				if i == len(tt.responses)-1 {
					status = http.StatusOK
				}
				responses = append(responses, &http.Response{
					StatusCode: status,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(responseBody)),
				})
			}
			upstream := &queuedHTTPUpstreamStub{responses: responses}
			cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
			svc := &GatewayService{
				cfg:              cfg,
				httpUpstream:     upstream,
				rateLimitService: &RateLimitService{},
				settingService:   NewSettingService(upstreamPreviewSettingRepo{}, cfg),
			}
			account := &Account{
				ID:          502,
				Name:        "retry-preview",
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Concurrency: 1,
				Credentials: map[string]any{"api_key": "upstream-key"},
				Extra: map[string]any{
					"passthrough_fields_enabled": true,
					"passthrough_field_rules": []PassthroughFieldRule{
						{Target: "body", Mode: "inject", Key: "metadata.retry_preview_marker", Value: tt.name},
					},
				},
			}

			_, err := svc.Forward(context.Background(), c, account, &ParsedRequest{
				Body:  NewRequestBodyRef(body),
				Model: "claude-sonnet-4-5",
			})
			require.NoError(t, err)
			require.Len(t, upstream.requestBodies, len(tt.responses))
			finalBody := string(upstream.requestBodies[len(upstream.requestBodies)-1])
			require.Equal(t, tt.name, gjson.Get(finalBody, "metadata.retry_preview_marker").String())
			require.Equal(t, finalBody, collector.body)
			require.Equal(t, finalBody, requireOpsPreviewString(t, c, tt.name))
		})
	}
}

func TestGatewayService_ForwardKeepsByteBackedParsedBodyReadable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`)
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-sonnet-4-5"}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"msg_readable","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)),
	}}}
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	svc := &GatewayService{
		cfg:              cfg,
		httpUpstream:     upstream,
		rateLimitService: &RateLimitService{},
		settingService:   NewSettingService(upstreamPreviewSettingRepo{}, cfg),
	}
	account := &Account{ID: 505, Name: "parsed-body-owner", Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Concurrency: 1, Credentials: map[string]any{"api_key": "upstream-key"}}

	_, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	readableBody, err := parsed.Body.ReadAll()
	require.NoError(t, err)
	require.JSONEq(t, string(body), string(readableBody))
}

func TestGatewayService_ForwardErrorKeepsByteBackedParsedBodyReadable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`)
	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-sonnet-4-5"}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"type":"error","error":{"type":"invalid_request_error","message":"invalid request"}}`)),
	}}}
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	svc := &GatewayService{
		cfg:              cfg,
		httpUpstream:     upstream,
		rateLimitService: &RateLimitService{},
		settingService:   NewSettingService(upstreamPreviewSettingRepo{}, cfg),
	}
	account := &Account{ID: 507, Name: "parsed-body-error-owner", Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Concurrency: 1, Credentials: map[string]any{"api_key": "upstream-key"}}

	_, err := svc.Forward(context.Background(), c, account, parsed)
	require.Error(t, err)
	readableBody, err := parsed.Body.ReadAll()
	require.NoError(t, err)
	require.JSONEq(t, string(body), string(readableBody))
}

func TestGatewayService_AnthropicPassthroughAcceptedWireSpoolFailureKeepsByteBackedParsedBodyReadable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sourceDir, wireDir := t.TempDir(), t.TempDir()
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"` + strings.Repeat("x", 2<<20) + `"}]}`)
	source, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: sourceDir})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(source) })
	t.Setenv("TMPDIR", wireDir)
	t.Setenv("TMP", wireDir)
	t.Setenv("TEMP", wireDir)

	parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-sonnet-4-5"}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	upstream := &anthropicRetryHandleUpstream{
		statuses:         []int{http.StatusOK},
		removeSpoolDir:   wireDir,
		removeSpoolAfter: 1,
	}
	cfg := &config.Config{
		Gateway:  config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
		Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}},
	}
	svc := &GatewayService{
		cfg:                 cfg,
		httpUpstream:        upstream,
		rateLimitService:    &RateLimitService{},
		settingService:      NewSettingService(upstreamPreviewSettingRepo{}, cfg),
		tlsFPProfileService: &TLSFingerprintProfileService{},
	}

	result, err := svc.forwardAnthropicAPIKeyPassthroughWithInput(context.Background(), c, newAnthropicAPIKeyAccountForTest(), anthropicPassthroughForwardInput{
		SourceHandle:      source,
		BodyHandle:        source,
		SourceBody:        body,
		Body:              body,
		Parsed:            parsed,
		SourceHandleOwned: true,
		RequestModel:      "claude-sonnet-4-5",
		OriginalModel:     "claude-sonnet-4-5",
		StartTime:         time.Now(),
	})
	require.Nil(t, result)
	require.ErrorIs(t, err, ErrRequestBodySpool)
	require.Nil(t, parsed.Body.Handle())
	readableBody, readErr := parsed.Body.ReadAll()
	require.NoError(t, readErr)
	require.JSONEq(t, string(body), string(readableBody))
}

func TestGatewayService_ForwardKeepsBorrowedParsedBodyHandle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`)
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })
	parsed := &ParsedRequest{Body: NewRequestBodyRefFromHandle(handle), Model: "claude-sonnet-4-5"}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"msg_borrowed","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)),
	}}}
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	svc := &GatewayService{
		cfg:              cfg,
		httpUpstream:     upstream,
		rateLimitService: &RateLimitService{},
		settingService:   NewSettingService(upstreamPreviewSettingRepo{}, cfg),
	}
	account := &Account{ID: 508, Name: "parsed-body-borrower", Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Concurrency: 1, Credentials: map[string]any{"api_key": "upstream-key"}}

	_, err = svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.Same(t, handle, parsed.Body.Handle())
	readableBody, err := parsed.Body.ReadAll()
	require.NoError(t, err)
	require.JSONEq(t, string(body), string(readableBody))
}

func TestGatewayService_AnthropicPassthroughPreservesParsedBodyOwnership(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`)
	newService := func() *GatewayService {
		cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
		return &GatewayService{
			cfg:              cfg,
			httpUpstream:     discardGatewayForwardUpstream{},
			rateLimitService: &RateLimitService{},
			settingService:   NewSettingService(upstreamPreviewSettingRepo{}, cfg),
		}
	}
	newAccount := func() *Account {
		account := newAnthropicAPIKeyAccountForTest()
		account.Extra["passthrough_fields_enabled"] = true
		account.Extra["passthrough_field_rules"] = []PassthroughFieldRule{{Target: "body", Mode: "inject", Key: "metadata.accepted_wire", Value: "changed"}}
		return account
	}

	t.Run("byte-backed", func(t *testing.T) {
		parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-sonnet-4-5"}
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

		_, err := newService().Forward(context.Background(), c, newAccount(), parsed)
		require.NoError(t, err)
		require.Nil(t, parsed.Body.Handle())
		readableBody, err := parsed.Body.ReadAll()
		require.NoError(t, err)
		require.Equal(t, "changed", gjson.GetBytes(readableBody, "metadata.accepted_wire").String())
	})

	t.Run("borrowed-handle", func(t *testing.T) {
		handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
		require.NoError(t, err)
		t.Cleanup(func() { CleanupRequestBodyHandle(handle) })
		parsed := &ParsedRequest{Body: NewRequestBodyRefFromHandle(handle), Model: "claude-sonnet-4-5"}
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

		_, err = newService().Forward(context.Background(), c, newAccount(), parsed)
		require.NoError(t, err)
		require.Same(t, handle, parsed.Body.Handle())
		readableBody, err := parsed.Body.ReadAll()
		require.NoError(t, err)
		require.JSONEq(t, string(body), string(readableBody))
	})
}

func TestGatewayService_BuildUpstreamRequestSameHandleMaterializesOnce(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"` + strings.Repeat("x", 89<<20/10) + `"}]}`)
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, PreviewLimitBytes: 64, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	materialized, err := handle.ReadAll()
	require.NoError(t, err)
	require.Equal(t, int(handle.Size()), len(materialized))
	runtime.ReadMemStats(&after)
	oneReadAlloc := after.TotalAlloc - before.TotalAlloc
	runtime.GC()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	svc := &GatewayService{cfg: &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}}}
	account := &Account{ID: 506, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "upstream-key"}}
	runtime.ReadMemStats(&before)
	req, wireHandle, err := svc.buildUpstreamRequestWithHandles(context.Background(), c, account, handle, handle, nil, nil, "upstream-key", "apikey", "claude-sonnet-4-5", false, false)
	runtime.ReadMemStats(&after)
	require.NoError(t, err)
	if req != nil && req.Body != nil {
		require.NoError(t, req.Body.Close())
	}
	CleanupRequestBodyHandle(wireHandle)
	buildAlloc := after.TotalAlloc - before.TotalAlloc
	t.Logf("one_read_alloc=%d build_alloc=%d", oneReadAlloc, buildAlloc)
	require.Less(t, buildAlloc, oneReadAlloc+(8<<20), "source==canonical must not allocate a second full materialization")
}

func TestGatewayService_ForwardFirstAttemptReusesMaterializedBody(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"` + strings.Repeat("x", 89<<20/10) + `"}]}`)
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, PreviewLimitBytes: 64, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	materialized, err := handle.ReadAll()
	require.NoError(t, err)
	require.Equal(t, int(handle.Size()), len(materialized))
	runtime.ReadMemStats(&after)
	oneReadAlloc := after.TotalAlloc - before.TotalAlloc
	runtime.GC()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	svc := &GatewayService{
		cfg:              cfg,
		httpUpstream:     discardGatewayForwardUpstream{},
		rateLimitService: &RateLimitService{},
		settingService:   NewSettingService(upstreamPreviewSettingRepo{}, cfg),
	}
	account := &Account{ID: 509, Name: "first-attempt-materialization", Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Concurrency: 1, Credentials: map[string]any{"api_key": "upstream-key"}}
	parsed := &ParsedRequest{Body: NewRequestBodyRefFromHandle(handle), Model: "claude-sonnet-4-5"}

	runtime.ReadMemStats(&before)
	_, err = svc.Forward(context.Background(), c, account, parsed)
	runtime.ReadMemStats(&after)
	require.NoError(t, err)
	forwardAlloc := after.TotalAlloc - before.TotalAlloc
	t.Logf("one_read_alloc=%d forward_alloc=%d", oneReadAlloc, forwardAlloc)
	require.Less(t, forwardAlloc, 2*oneReadAlloc+(16<<20), "first Forward attempt must reuse its initial materialization")
}

func TestGatewayService_BedrockFirstAttemptReusesMaterializedBody(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"` + strings.Repeat("x", 89<<20/10) + `"}]}`)
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, PreviewLimitBytes: 64, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	materialized, err := handle.ReadAll()
	require.NoError(t, err)
	require.Equal(t, int(handle.Size()), len(materialized))
	runtime.ReadMemStats(&after)
	oneReadAlloc := after.TotalAlloc - before.TotalAlloc
	runtime.GC()

	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	svc := &GatewayService{cfg: cfg, httpUpstream: discardGatewayForwardUpstream{}, rateLimitService: &RateLimitService{}}
	account := &Account{
		ID: 510, Name: "bedrock-first-attempt-materialization", Platform: PlatformAnthropic, Type: AccountTypeBedrock, Concurrency: 1,
		Credentials: map[string]any{"auth_mode": "apikey", "api_key": "bedrock-key"},
	}
	byteRecorder := httptest.NewRecorder()
	byteContext, _ := gin.CreateTestContext(byteRecorder)
	byteContext.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	byteParsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-sonnet-4-5"}
	runtime.ReadMemStats(&before)
	_, err = svc.Forward(context.Background(), byteContext, account, byteParsed)
	runtime.ReadMemStats(&after)
	require.NoError(t, err)
	byteForwardAlloc := after.TotalAlloc - before.TotalAlloc
	runtime.GC()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	parsed := &ParsedRequest{Body: NewRequestBodyRefFromHandle(handle), Model: "claude-sonnet-4-5"}

	runtime.ReadMemStats(&before)
	_, err = svc.Forward(context.Background(), c, account, parsed)
	runtime.ReadMemStats(&after)
	require.NoError(t, err)
	forwardAlloc := after.TotalAlloc - before.TotalAlloc
	t.Logf("one_read_alloc=%d byte_forward_alloc=%d bedrock_forward_alloc=%d", oneReadAlloc, byteForwardAlloc, forwardAlloc)
	require.Less(t, forwardAlloc, byteForwardAlloc+oneReadAlloc+(8<<20), "Bedrock first attempt must add at most one handle materialization")
}

func TestGatewayService_BedrockBodyHandleReadFailurePreservesSpoolSentinel(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`)
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })
	require.NoError(t, os.Remove(handle.spoolPath))

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	account := &Account{Platform: PlatformAnthropic, Type: AccountTypeBedrock}
	_, err = (&GatewayService{}).Forward(context.Background(), c, account, &ParsedRequest{
		Body:  NewRequestBodyRefFromHandle(handle),
		Model: "claude-sonnet-4-5",
	})

	require.ErrorIs(t, err, ErrRequestBodySpool)
}

func TestGatewayService_AnthropicPassthroughFirstAttemptReusesMaterializedBody(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"` + strings.Repeat("x", 89<<20/10) + `"}]}`)
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, PreviewLimitBytes: 64, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	materialized, err := handle.ReadAll()
	require.NoError(t, err)
	require.Equal(t, int(handle.Size()), len(materialized))
	runtime.ReadMemStats(&after)
	oneReadAlloc := after.TotalAlloc - before.TotalAlloc
	runtime.GC()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	svc := &GatewayService{
		cfg:              cfg,
		httpUpstream:     discardGatewayForwardUpstream{},
		rateLimitService: &RateLimitService{},
		settingService:   NewSettingService(upstreamPreviewSettingRepo{}, cfg),
	}
	account := newAnthropicAPIKeyAccountForTest()
	parsed := &ParsedRequest{Body: NewRequestBodyRefFromHandle(handle), Model: "claude-sonnet-4-5"}

	runtime.ReadMemStats(&before)
	_, err = svc.Forward(context.Background(), c, account, parsed)
	runtime.ReadMemStats(&after)
	require.NoError(t, err)
	forwardAlloc := after.TotalAlloc - before.TotalAlloc
	t.Logf("one_read_alloc=%d passthrough_forward_alloc=%d", oneReadAlloc, forwardAlloc)
	require.Less(t, forwardAlloc, 2*oneReadAlloc+(16<<20), "passthrough first attempt must reuse Forward's initial materialization")
}

type discardGatewayForwardUpstream struct{ HTTPUpstream }

func (discardGatewayForwardUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	if _, err := io.Copy(io.Discard, req.Body); err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"msg_materialized","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)),
	}, nil
}

func TestGatewayService_SignatureRetryTransportErrorKeepsAttemptBodiesAligned(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":1024,"thinking":{"type":"enabled","budget_tokens":1024},"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"work","signature":"stale"}]},{"role":"user","content":"continue"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	collector := &openAIUsageUpstreamRequestCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)

	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"type":"error","error":{"type":"invalid_request_error","message":"Invalid signature in thinking block"}}`)),
		}, nil},
		errors: []error{nil, errors.New("retry transport failed")},
	}
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	svc := &GatewayService{
		cfg:              cfg,
		httpUpstream:     upstream,
		rateLimitService: &RateLimitService{},
		settingService:   NewSettingService(upstreamPreviewSettingRepo{}, cfg),
	}
	account := &Account{
		ID: 503, Name: "retry-transport-preview", Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Concurrency: 1,
		Credentials: map[string]any{"api_key": "upstream-key"},
	}

	_, err := svc.Forward(context.Background(), c, account, &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-sonnet-4-5"})

	require.Error(t, err)
	require.Len(t, upstream.requestBodies, 2)
	originalAttempt := string(upstream.requestBodies[0])
	retryAttempt := string(upstream.requestBodies[1])
	require.NotEqual(t, originalAttempt, retryAttempt)
	require.Equal(t, originalAttempt, collector.body)
	require.Equal(t, originalAttempt, requireOpsPreviewString(t, c, "claude-sonnet-4-5"))

	rawEvents, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	requestBodies := map[string]string{}
	for _, event := range events {
		requestBodies[event.Kind] = unwrapRequestBodyPreviewForTest(event.UpstreamRequestBody)
	}
	require.Equal(t, originalAttempt, requestBodies["signature_error"])
	require.Equal(t, retryAttempt, requestBodies["signature_retry_request_error"])
}

func TestGatewayService_BudgetRetryTransportErrorKeepsAttemptBodiesAligned(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":512,"thinking":{"type":"enabled","budget_tokens":512},"messages":[{"role":"user","content":"hello"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	collector := &openAIUsageUpstreamRequestCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)

	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"type":"error","error":{"type":"invalid_request_error","message":"thinking budget_tokens input should be greater than or equal to 1024"}}`)),
		}, nil},
		errors: []error{nil, errors.New("budget retry transport failed")},
	}
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	svc := &GatewayService{
		cfg:              cfg,
		httpUpstream:     upstream,
		rateLimitService: &RateLimitService{},
		settingService:   NewSettingService(upstreamPreviewSettingRepo{}, cfg),
	}
	account := &Account{ID: 504, Name: "budget-retry-preview", Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Concurrency: 1, Credentials: map[string]any{"api_key": "upstream-key"}}

	_, err := svc.Forward(context.Background(), c, account, &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-sonnet-4-5"})

	require.Error(t, err)
	require.Len(t, upstream.requestBodies, 2)
	originalAttempt := string(upstream.requestBodies[0])
	retryAttempt := string(upstream.requestBodies[1])
	require.NotEqual(t, originalAttempt, retryAttempt)
	require.Equal(t, originalAttempt, collector.body)
	require.Equal(t, originalAttempt, requireOpsPreviewString(t, c, "claude-sonnet-4-5"))

	rawEvents, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := rawEvents.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	requestBodies := map[string]string{}
	for _, event := range events {
		requestBodies[event.Kind] = unwrapRequestBodyPreviewForTest(event.UpstreamRequestBody)
	}
	require.Equal(t, originalAttempt, requestBodies["budget_constraint_error"])
	require.Equal(t, retryAttempt, requestBodies["budget_retry_request_error"])
}

func TestGatewayService_AnthropicAPIKeyPassthrough_TransportErrorRecordsOllamaActivity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deferred := NewDeferredService(nil, nil, time.Second)
	svc := &GatewayService{
		cfg:             &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream:    &anthropicHTTPUpstreamRecorder{err: errors.New("dial tcp timeout")},
		deferredService: deferred,
	}
	ollama := &Account{ID: 601, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Concurrency: 1, Credentials: map[string]any{"api_key": "k", "base_url": "https://ollama.com"}, Extra: map[string]any{"anthropic_passthrough": true}, Status: StatusActive, Schedulable: true}
	other := newAnthropicAPIKeyAccountForTest()
	other.ID = 602

	for _, account := range []*Account{ollama, other} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
		_, err := svc.forwardAnthropicAPIKeyPassthrough(context.Background(), c, account, []byte(`{"model":"x"}`), "x", "x", false, time.Now())
		require.Error(t, err)
	}
	_, ok := deferred.lastUsedUpdates.Load(int64(601))
	require.True(t, ok)
	_, ok = deferred.lastUsedUpdates.Load(int64(602))
	require.False(t, ok)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_ContextCanceledSkipsOllamaActivity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deferred := NewDeferredService(nil, nil, time.Second)
	svc := &GatewayService{
		cfg:             &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream:    &anthropicHTTPUpstreamRecorder{err: context.Canceled},
		deferredService: deferred,
	}
	account := &Account{ID: 603, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Concurrency: 1, Credentials: map[string]any{"api_key": "k", "base_url": "https://ollama.com"}, Extra: map[string]any{"anthropic_passthrough": true}, Status: StatusActive, Schedulable: true}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	_, err := svc.forwardAnthropicAPIKeyPassthrough(context.Background(), c, account, []byte(`{"model":"x"}`), "x", "x", false, time.Now())
	require.Error(t, err)
	_, ok := deferred.lastUsedUpdates.Load(int64(603))
	require.False(t, ok)
}

func TestGatewayService_AnthropicAPIKeyPassthrough_Non2xxRecordsOllamaActivity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deferred := NewDeferredService(nil, nil, time.Second)
	svc := &GatewayService{
		cfg: &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
		httpUpstream: &anthropicHTTPUpstreamRecorder{resp: &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"type":"error","error":{"type":"invalid_request_error","message":"bad"}}`)),
		}},
		deferredService:  deferred,
		rateLimitService: &RateLimitService{},
	}
	account := &Account{ID: 604, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Concurrency: 1, Credentials: map[string]any{"api_key": "k", "base_url": "https://ollama.com"}, Extra: map[string]any{"anthropic_passthrough": true}, Status: StatusActive, Schedulable: true}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	_, _ = svc.forwardAnthropicAPIKeyPassthrough(context.Background(), c, account, []byte(`{"model":"x"}`), "x", "x", false, time.Now())
	_, ok := deferred.lastUsedUpdates.Load(int64(604))
	require.True(t, ok)
}
