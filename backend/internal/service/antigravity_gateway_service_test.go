package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// antigravityFailingWriter 模拟客户端断开连接的 gin.ResponseWriter
type antigravityFailingWriter struct {
	gin.ResponseWriter
	failAfter int // 允许成功写入的次数，之后所有写入返回错误
	writes    int
}

func (w *antigravityFailingWriter) Write(p []byte) (int, error) {
	if w.writes >= w.failAfter {
		return 0, errors.New("write failed: client disconnected")
	}
	w.writes++
	return w.ResponseWriter.Write(p)
}

// newAntigravityTestService 创建用于流式测试的 AntigravityGatewayService
func newAntigravityTestService(cfg *config.Config) *AntigravityGatewayService {
	return &AntigravityGatewayService{
		settingService: &SettingService{cfg: cfg},
	}
}

func TestAntigravityGatewayService_ClaudeCanonicalReadsUseDirectHandle(t *testing.T) {
	source, err := os.ReadFile("antigravity_gateway_claude.go")
	require.NoError(t, err)
	require.NotContains(t, string(source), "readAntigravityClaudeCanonicalBody")
	require.Equal(t, 3, strings.Count(string(source), "bodyHandle.ReadAll()"))
}

func TestAntigravityUpstreamErrorBodyReadLimit_RespectsDiagnosticLimit(t *testing.T) {
	svc := newAntigravityTestService(&config.Config{Gateway: config.GatewayConfig{
		LogUpstreamErrorBody:         true,
		LogUpstreamErrorBodyMaxBytes: int(gatewayUpstreamErrorBodyReadLimit) + 1024,
	}})

	require.Equal(t, int64(svc.settingService.cfg.Gateway.LogUpstreamErrorBodyMaxBytes), svc.upstreamErrorBodyReadLimit())
}

func TestForwardAsResponsesHandle_UsesReplayableTransformedHandle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gemini-3.1-pro-high","input":"Reply exactly: ok"}`)
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })

	upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{antigravityCompatSuccessResponse()}}
	svc := newAntigravityCompatService(config.GatewayConfig{MaxLineSize: defaultMaxLineSize}, upstream)
	c, _ := newAntigravityCompatContext(http.MethodPost, "/v1/responses", body)
	parsed, err := ParseGatewayRequest(NewRequestBodyRefFromHandle(handle), "responses")
	require.NoError(t, err)
	require.NotNil(t, parsed.Body.Handle())

	result, err := svc.ForwardAsResponsesHandle(context.Background(), c, newAntigravityCompatAccount(AccountTypeOAuth), handle, parsed)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requestBodies, 1)
	require.Contains(t, string(upstream.requestBodies[0]), "Reply exactly: ok")
}

func TestAntigravityCompatSpoolErrorPreservesSentinel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gemini-3.1-pro-high","input":"hello"}`)
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })
	spoolErr := fmt.Errorf("open transformed payload: %w", ErrRequestBodySpool)
	svc := newAntigravityCompatService(config.GatewayConfig{MaxLineSize: defaultMaxLineSize}, &httpUpstreamStub{err: spoolErr})
	c, recorder := newAntigravityCompatContext(http.MethodPost, "/v1/responses", body)

	result, err := svc.ForwardAsResponsesHandle(context.Background(), c, newAntigravityCompatAccount(AccountTypeOAuth), handle, nil)

	require.Nil(t, result)
	require.ErrorIs(t, err, ErrRequestBodySpool)
	require.Empty(t, recorder.Body.String())
}

func TestAntigravityGatewayService_ForwardGeminiTransportSpoolErrorReturnsWithoutRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"` + strings.Repeat("x", int(DefaultRequestBodySpoolThresholdBytes)+1) + `"}]}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", bytes.NewReader(body))
	responseBody := &closeTrackingReadCloser{Reader: strings.NewReader(`{"error":"partial"}`)}
	upstream := &transportSpoolCloseUpstream{
		resp: &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{}, Body: responseBody},
		err:  fmt.Errorf("read native payload: %w", ErrRequestBodySpool),
	}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}
	account := &Account{
		ID: 103, Name: "antigravity-native-spool", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Status: StatusActive, Concurrency: 1,
		Credentials: map[string]any{"access_token": "token", "project_id": "project", "model_mapping": map[string]any{"gemini-2.5-flash": "gemini-2.5-flash"}},
	}

	result, err := svc.ForwardGemini(context.Background(), c, account, "gemini-2.5-flash", "generateContent", false, body, false)

	require.Nil(t, result)
	require.ErrorIs(t, err, ErrRequestBodySpool)
	require.Equal(t, 1, upstream.callCount)
	requireRequestBodyClosed(t, upstream.request)
	require.True(t, responseBody.closed)
	require.Empty(t, recorder.Body.String())
}

func TestHandleSmartRetry_TransportSpoolErrorStopsModelCapacityRetries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	spoolErr := fmt.Errorf("read smart retry payload: %w", ErrRequestBodySpool)
	responseBody := &closeTrackingReadCloser{Reader: strings.NewReader(`{"error":"partial"}`)}
	upstream := &transportSpoolCloseUpstream{
		resp: &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{}, Body: responseBody},
		err:  spoolErr,
	}
	account := &Account{ID: 104, Name: "smart-spool", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Concurrency: 1}
	payloadHandle, err := NewRequestBodyHandleFromBytes([]byte(`{"request":"`+strings.Repeat("x", int(DefaultRequestBodySpoolThresholdBytes)+1)+`"}`), RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(payloadHandle) })
	respBody := []byte(`{
		"error": {
			"code": 503,
			"status": "UNAVAILABLE",
			"details": [
				{"@type":"type.googleapis.com/google.rpc.ErrorInfo","metadata":{"model":"wave8-smart-spool-model"},"reason":"MODEL_CAPACITY_EXHAUSTED"}
			]
		}
	}`)
	result := (&AntigravityGatewayService{}).handleSmartRetry(antigravityRetryLoopParams{
		ctx: context.Background(), prefix: "[test]", account: account, accessToken: "token", action: "generateContent",
		payloadHandle: payloadHandle, c: c, httpUpstream: upstream,
	}, &http.Response{StatusCode: http.StatusServiceUnavailable, Header: http.Header{}}, respBody, "https://ag.test", 0, []string{"https://ag.test"})

	require.Equal(t, smartRetryActionBreakWithResp, result.action)
	require.ErrorIs(t, result.err, ErrRequestBodySpool)
	require.Nil(t, result.resp)
	require.Nil(t, result.switchError)
	require.Equal(t, 1, upstream.callCount)
	requireRequestBodyClosed(t, upstream.request)
	require.True(t, responseBody.closed)
	require.Empty(t, recorder.Body.String())
}

func TestHandleSingleAccountRetryInPlace_TransportSpoolErrorStopsRetries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	spoolErr := fmt.Errorf("read single-account retry payload: %w", ErrRequestBodySpool)
	responseBody := &closeTrackingReadCloser{Reader: strings.NewReader(`{"error":"partial"}`)}
	upstream := &transportSpoolCloseUpstream{
		resp: &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{}, Body: responseBody},
		err:  spoolErr,
	}
	account := &Account{ID: 105, Name: "single-spool", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Concurrency: 1}
	payloadHandle, err := NewRequestBodyHandleFromBytes([]byte(`{"request":"`+strings.Repeat("x", int(DefaultRequestBodySpoolThresholdBytes)+1)+`"}`), RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(payloadHandle) })
	result := (&AntigravityGatewayService{}).handleSingleAccountRetryInPlace(antigravityRetryLoopParams{
		ctx: WithSingleAccountRetry(context.Background(), true, false), prefix: "[test]", account: account,
		accessToken: "token", action: "generateContent", payloadHandle: payloadHandle, c: c, httpUpstream: upstream,
	}, &http.Response{StatusCode: http.StatusServiceUnavailable, Header: http.Header{}}, []byte(`{"error":"unavailable"}`), "https://ag.test", 0, "wave8-single-spool-model")

	require.Equal(t, smartRetryActionBreakWithResp, result.action)
	require.ErrorIs(t, result.err, ErrRequestBodySpool)
	require.Nil(t, result.resp)
	require.Nil(t, result.switchError)
	require.Equal(t, 1, upstream.callCount)
	requireRequestBodyClosed(t, upstream.request)
	require.True(t, responseBody.closed)
	require.Empty(t, recorder.Body.String())
}

func TestAntigravityRetryLoop_TransportErrorCleansRequestBodySpool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	ctx, cancel := context.WithCancel(context.Background())
	responseBody := &closeTrackingReadCloser{Reader: strings.NewReader(`{"error":"preflight"}`)}
	upstream := &transportSpoolCloseUpstream{
		resp:   &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{}, Body: responseBody},
		err:    errors.New("preflight rejected request"),
		onCall: cancel,
	}
	payloadHandle, spoolDir := newAntigravityRetrySpoolHandle(t)
	result, err := (&AntigravityGatewayService{}).antigravityRetryLoop(antigravityRetryLoopParams{
		ctx: ctx, prefix: "[test]", account: &Account{ID: 109, Name: "main-preflight", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Concurrency: 1},
		accessToken: "token", action: "generateContent", payloadHandle: payloadHandle, c: c, httpUpstream: upstream,
	})

	require.Nil(t, result)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, upstream.callCount)
	require.True(t, responseBody.closed)
	requireAntigravityRetrySpoolReleased(t, upstream.request, payloadHandle, spoolDir)
}

func TestHandleSmartRetry_TransportErrorCleansRequestBodySpool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	ctx, cancel := context.WithCancel(context.Background())
	responseBody := &closeTrackingReadCloser{Reader: strings.NewReader(`{"error":"preflight"}`)}
	upstream := &transportSpoolCloseUpstream{
		resp:   &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{}, Body: responseBody},
		err:    errors.New("preflight rejected request"),
		onCall: cancel,
	}
	payloadHandle, spoolDir := newAntigravityRetrySpoolHandle(t)
	respBody := []byte(`{"error":{"code":503,"status":"UNAVAILABLE","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","metadata":{"model":"cleanup-smart-model"},"reason":"MODEL_CAPACITY_EXHAUSTED"}]}}`)
	result := (&AntigravityGatewayService{}).handleSmartRetry(antigravityRetryLoopParams{
		ctx: ctx, prefix: "[test]", account: &Account{ID: 110, Name: "smart-preflight", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Concurrency: 1},
		accessToken: "token", action: "generateContent", payloadHandle: payloadHandle, c: c, httpUpstream: upstream,
	}, &http.Response{StatusCode: http.StatusServiceUnavailable, Header: http.Header{}}, respBody, "https://ag.test", 0, []string{"https://ag.test"})

	require.Equal(t, smartRetryActionBreakWithResp, result.action)
	require.ErrorIs(t, result.err, context.Canceled)
	require.Equal(t, 1, upstream.callCount)
	require.True(t, responseBody.closed)
	requireAntigravityRetrySpoolReleased(t, upstream.request, payloadHandle, spoolDir)
}

func TestHandleSingleAccountRetryInPlace_TransportErrorCleansRequestBodySpool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	ctx, cancel := context.WithCancel(WithSingleAccountRetry(context.Background(), true, false))
	responseBody := &closeTrackingReadCloser{Reader: strings.NewReader(`{"error":"preflight"}`)}
	upstream := &transportSpoolCloseUpstream{
		resp:   &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{}, Body: responseBody},
		err:    errors.New("preflight rejected request"),
		onCall: cancel,
	}
	payloadHandle, spoolDir := newAntigravityRetrySpoolHandle(t)
	result := (&AntigravityGatewayService{}).handleSingleAccountRetryInPlace(antigravityRetryLoopParams{
		ctx: ctx, prefix: "[test]", account: &Account{ID: 111, Name: "single-preflight", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Concurrency: 1},
		accessToken: "token", action: "generateContent", payloadHandle: payloadHandle, c: c, httpUpstream: upstream,
	}, &http.Response{StatusCode: http.StatusServiceUnavailable, Header: http.Header{}}, []byte(`{"error":"unavailable"}`), "https://ag.test", 0, "cleanup-single-model")

	require.Equal(t, smartRetryActionBreakWithResp, result.action)
	require.ErrorIs(t, result.err, context.Canceled)
	require.Equal(t, 1, upstream.callCount)
	require.True(t, responseBody.closed)
	requireAntigravityRetrySpoolReleased(t, upstream.request, payloadHandle, spoolDir)
}

func newAntigravityRetrySpoolHandle(t *testing.T) (*RequestBodyHandle, string) {
	t.Helper()
	spoolDir := t.TempDir()
	handle, err := NewRequestBodyHandleFromBytes([]byte(`{"request":"payload"}`), RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: spoolDir})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })
	entries, err := os.ReadDir(spoolDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	return handle, spoolDir
}

func requireAntigravityRetrySpoolReleased(t *testing.T, req *http.Request, handle *RequestBodyHandle, spoolDir string) {
	t.Helper()
	requireRequestBodyClosed(t, req)
	CleanupRequestBodyHandle(handle)
	entries, err := os.ReadDir(spoolDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestAttemptCreditsOveragesRetry_BuildSpoolErrorReturnsUnchanged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	spoolErr := fmt.Errorf("build credits retry request: %w", ErrRequestBodySpool)
	oldBuilder := newAntigravityCreditsPayloadRequest
	newAntigravityCreditsPayloadRequest = func(*antigravityRetryLoopParams, string) (*http.Request, error) {
		return nil, spoolErr
	}
	t.Cleanup(func() { newAntigravityCreditsPayloadRequest = oldBuilder })
	payloadHandle, err := NewRequestBodyHandleFromBytes([]byte(`{"request":{}}`), RequestBodyHandleOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(payloadHandle) })

	account := &Account{ID: 108, Name: "credits-build-spool", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Concurrency: 1}
	result := (&AntigravityGatewayService{}).attemptCreditsOveragesRetry(antigravityRetryLoopParams{
		ctx: context.Background(), prefix: "[test]", account: account, accessToken: "token", action: "generateContent",
		payloadHandle: payloadHandle, c: c,
	}, "https://ag.test", "gemini-wave9", 0, http.StatusTooManyRequests, nil)

	require.True(t, result.err == spoolErr, "spool sentinel must be returned without wrapping")
	require.False(t, HasOpsUpstreamAttempted(c))
	require.Empty(t, recorder.Body.String())
}

func TestStripSignatureSensitiveBlocksFromClaudeRequest(t *testing.T) {
	req := &antigravity.ClaudeRequest{
		Model: "claude-sonnet-4-5",
		Thinking: &antigravity.ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: 1024,
		},
		Messages: []antigravity.ClaudeMessage{
			{
				Role: "assistant",
				Content: json.RawMessage(`[
					{"type":"thinking","thinking":"secret plan","signature":""},
					{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}}
				]`),
			},
			{
				Role: "user",
				Content: json.RawMessage(`[
					{"type":"tool_result","tool_use_id":"t1","content":"ok","is_error":false},
					{"type":"redacted_thinking","data":"..."}
				]`),
			},
		},
	}

	changed, err := stripSignatureSensitiveBlocksFromClaudeRequest(req)
	require.NoError(t, err)
	require.True(t, changed)
	require.Nil(t, req.Thinking)

	require.Len(t, req.Messages, 2)

	var blocks0 []map[string]any
	require.NoError(t, json.Unmarshal(req.Messages[0].Content, &blocks0))
	require.Len(t, blocks0, 2)
	require.Equal(t, "text", blocks0[0]["type"])
	require.Equal(t, "secret plan", blocks0[0]["text"])
	require.Equal(t, "text", blocks0[1]["type"])

	var blocks1 []map[string]any
	require.NoError(t, json.Unmarshal(req.Messages[1].Content, &blocks1))
	require.Len(t, blocks1, 1)
	require.Equal(t, "text", blocks1[0]["type"])
	require.NotEmpty(t, blocks1[0]["text"])
}

func TestStripThinkingFromClaudeRequest_DoesNotDowngradeTools(t *testing.T) {
	req := &antigravity.ClaudeRequest{
		Model: "claude-sonnet-4-5",
		Thinking: &antigravity.ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: 1024,
		},
		Messages: []antigravity.ClaudeMessage{
			{
				Role:    "assistant",
				Content: json.RawMessage(`[{"type":"thinking","thinking":"secret plan"},{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}}]`),
			},
		},
	}

	changed, err := stripThinkingFromClaudeRequest(req)
	require.NoError(t, err)
	require.True(t, changed)
	require.Nil(t, req.Thinking)

	var blocks []map[string]any
	require.NoError(t, json.Unmarshal(req.Messages[0].Content, &blocks))
	require.Len(t, blocks, 2)
	require.Equal(t, "text", blocks[0]["type"])
	require.Equal(t, "secret plan", blocks[0]["text"])
	require.Equal(t, "tool_use", blocks[1]["type"])
}

func TestIsPromptTooLongError(t *testing.T) {
	require.True(t, isPromptTooLongError([]byte(`{"error":{"message":"Prompt is too long"}}`)))
	require.True(t, isPromptTooLongError([]byte(`{"message":"Prompt is too long"}`)))
	require.False(t, isPromptTooLongError([]byte(`{"error":{"message":"other"}}`)))
}

type httpUpstreamStub struct {
	resp *http.Response
	err  error
}

type closeTrackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *closeTrackingReadCloser) Close() error {
	r.closed = true
	return nil
}

type transportSpoolCloseUpstream struct {
	HTTPUpstream
	request   *http.Request
	resp      *http.Response
	err       error
	onCall    func()
	callCount int
}

func (s *transportSpoolCloseUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	s.request = req
	s.callCount++
	if s.onCall != nil {
		s.onCall()
	}
	return s.resp, s.err
}

func (s *httpUpstreamStub) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return s.resp, s.err
}

func (s *httpUpstreamStub) DoWithTLS(_ *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return s.resp, s.err
}

type queuedHTTPUpstreamStub struct {
	responses      []*http.Response
	errors         []error
	requestBodies  [][]byte
	requestHeaders []http.Header
	callCount      int
	onCall         func(*http.Request, *queuedHTTPUpstreamStub)
}

func (s *queuedHTTPUpstreamStub) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	if req != nil {
		s.requestHeaders = append(s.requestHeaders, req.Header.Clone())
	} else {
		s.requestHeaders = append(s.requestHeaders, nil)
	}
	if req != nil && req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		s.requestBodies = append(s.requestBodies, body)
		req.Body = io.NopCloser(bytes.NewReader(body))
	} else {
		s.requestBodies = append(s.requestBodies, nil)
	}

	idx := s.callCount
	s.callCount++
	if s.onCall != nil {
		s.onCall(req, s)
	}

	var resp *http.Response
	if idx < len(s.responses) {
		resp = s.responses[idx]
	}
	var err error
	if idx < len(s.errors) {
		err = s.errors[idx]
	}
	if resp == nil && err == nil {
		return nil, errors.New("unexpected upstream call")
	}
	return resp, err
}

func (s *queuedHTTPUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, concurrency)
}

type claudeSignatureRetryHTTPUpstream struct {
	HTTPUpstream
	responses        []*http.Response
	requestBodies    [][]byte
	requestBodyPaths []string
	onCall           func(int)
}

type claudeSignatureRetryDiscardUpstream struct {
	HTTPUpstream
	responses []*http.Response
	calls     int
}

func (s *claudeSignatureRetryDiscardUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	if _, err := io.Copy(io.Discard, req.Body); err != nil {
		return nil, err
	}
	s.calls++
	if s.calls > len(s.responses) {
		return nil, errors.New("unexpected Claude signature retry upstream call")
	}
	return s.responses[s.calls-1], nil
}

func (s *claudeSignatureRetryHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	path := ""
	if reader, ok := req.Body.(requestBodySpoolReadCloser); ok {
		if file, ok := reader.ReadCloser.(*os.File); ok {
			path = file.Name()
		}
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	s.requestBodies = append(s.requestBodies, body)
	s.requestBodyPaths = append(s.requestBodyPaths, path)
	call := len(s.requestBodies)
	if s.onCall != nil {
		s.onCall(call)
	}
	if call > len(s.responses) {
		return nil, errors.New("unexpected Claude signature retry upstream call")
	}
	return s.responses[call-1], nil
}

type recordingInternal500CounterCache struct {
	incrementCalls []int64
	resetCalls     []int64
}

func (c *recordingInternal500CounterCache) IncrementInternal500Count(_ context.Context, accountID int64) (int64, error) {
	c.incrementCalls = append(c.incrementCalls, accountID)
	return int64(len(c.incrementCalls)), nil
}

func (c *recordingInternal500CounterCache) ResetInternal500Count(_ context.Context, accountID int64) error {
	c.resetCalls = append(c.resetCalls, accountID)
	return nil
}

type antigravitySettingRepoStub struct{}

type antigravitySettingRepoValuesStub struct {
	values map[string]string
}

func (s *antigravitySettingRepoValuesStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *antigravitySettingRepoValuesStub) GetValue(ctx context.Context, key string) (string, error) {
	if s.values == nil {
		return "", ErrSettingNotFound
	}
	v, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return v, nil
}

func (s *antigravitySettingRepoValuesStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *antigravitySettingRepoValuesStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *antigravitySettingRepoValuesStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *antigravitySettingRepoValuesStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *antigravitySettingRepoValuesStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

type antigravityUsageUpstreamRequestCollectorStub struct {
	headers string
	body    string
}

func (s *antigravityUsageUpstreamRequestCollectorStub) SetUsageUpstreamRequest(headers, body string) {
	s.headers = headers
	s.body = unwrapRequestBodyPreviewForTest(body)
}

func (s *antigravitySettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *antigravitySettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	return "", ErrSettingNotFound
}

func (s *antigravitySettingRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *antigravitySettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *antigravitySettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *antigravitySettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *antigravitySettingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestResolveAntigravityProjectID(t *testing.T) {
	tests := []struct {
		name    string
		account *Account
		want    string
		wantErr bool
	}{
		{
			name: "uses onboard project_id first",
			account: &Account{Credentials: map[string]any{
				"project_id": " onboard-project ",
				antigravityProjectIDFallbackCredentialKey: " configured-project ",
			}},
			want: "onboard-project",
		},
		{
			name: "uses configured credentials fallback",
			account: &Account{Credentials: map[string]any{
				antigravityProjectIDFallbackCredentialKey: " configured-project ",
			}},
			want: "configured-project",
		},
		{
			name: "uses configured extra fallback",
			account: &Account{Extra: map[string]any{
				antigravityProjectIDFallbackCredentialKey: " extra-project ",
			}},
			want: "extra-project",
		},
		{
			name:    "missing project",
			account: &Account{Credentials: map[string]any{}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveAntigravityProjectID(tc.account)
			if tc.wantErr {
				require.ErrorIs(t, err, errAntigravityProjectIDRequired)
				require.Empty(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestAntigravityGatewayService_ForwardGemini_UsesConfiguredProjectFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]any{{"text": "hello"}}},
		},
	})
	require.NoError(t, err)
	c.Request = httptest.NewRequest(http.MethodPost, "/antigravity/v1beta/models/gemini-2.5-flash:streamGenerateContent", bytes.NewReader(body))

	upstreamBody := []byte("data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":1,\"candidatesTokenCount\":1}}}\n\n")
	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(bytes.NewReader(upstreamBody)),
			},
		},
	}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}

	account := &Account{
		ID:          101,
		Name:        "acc-configured-project",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			antigravityProjectIDFallbackCredentialKey: "configured-project",
			"model_mapping": map[string]any{
				"gemini-2.5-flash": "gemini-2.5-flash",
			},
		},
	}

	result, err := svc.ForwardGemini(context.Background(), c, account, "gemini-2.5-flash", "streamGenerateContent", true, body, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requestBodies, 1)
	require.Equal(t, RequestBodyPreviewString(upstream.requestBodies[0]), requireOpsPreviewString(t, c, "configured-project"))

	var wrapped map[string]any
	require.NoError(t, json.Unmarshal(upstream.requestBodies[0], &wrapped))
	require.Equal(t, "configured-project", wrapped["project"])
}

func TestAntigravityGatewayService_ForwardGemini_MissingProjectReturnsLocalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]any{{"text": "hello"}}},
		},
	})
	require.NoError(t, err)
	c.Request = httptest.NewRequest(http.MethodPost, "/antigravity/v1beta/models/gemini-2.5-flash:streamGenerateContent", bytes.NewReader(body))

	upstream := &queuedHTTPUpstreamStub{}
	internal500Cache := &recordingInternal500CounterCache{}
	svc := &AntigravityGatewayService{
		tokenProvider:    &AntigravityTokenProvider{},
		httpUpstream:     upstream,
		internal500Cache: internal500Cache,
	}

	account := &Account{
		ID:          102,
		Name:        "acc-missing-project",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"model_mapping": map[string]any{
				"gemini-2.5-flash": "gemini-2.5-flash",
			},
		},
	}

	result, err := svc.ForwardGemini(context.Background(), c, account, "gemini-2.5-flash", "streamGenerateContent", true, body, false)
	require.Nil(t, result)
	require.ErrorIs(t, err, errAntigravityProjectIDRequired)
	require.Equal(t, http.StatusBadRequest, writer.Code)
	require.Empty(t, upstream.requestBodies)
	require.Empty(t, internal500Cache.incrementCalls)
	require.Contains(t, writer.Body.String(), "project_id")
	require.NotContains(t, writer.Body.String(), `"project":""`)
}

func TestAntigravityGatewayService_Forward_PromptTooLong(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"model": "claude-opus-4-6",
		"messages": []map[string]any{
			{"role": "user", "content": "hi"},
		},
		"max_tokens": 1,
		"stream":     false,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	c.Request = req

	respBody := []byte(`{"error":{"message":"Prompt is too long"}}`)
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"X-Request-Id": []string{"req-1"}},
		Body:       io.NopCloser(bytes.NewReader(respBody)),
	}

	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   &httpUpstreamStub{resp: resp},
	}

	account := &Account{
		ID:          1,
		Name:        "acc-1",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"project_id":   "proj",
		},
	}

	result, err := svc.Forward(context.Background(), c, account, body, false)
	require.Nil(t, result)

	var promptErr *PromptTooLongError
	require.ErrorAs(t, err, &promptErr)
	require.Equal(t, http.StatusBadRequest, promptErr.StatusCode)
	require.Equal(t, "req-1", promptErr.RequestID)
	require.NotEmpty(t, promptErr.Body)

	raw, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := raw.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 1)
	require.Equal(t, "prompt_too_long", events[0].Kind)
}

// TestAntigravityGatewayService_Forward_ModelRateLimitTriggersFailover
// 验证：当账号存在模型限流且剩余时间 >= antigravityRateLimitThreshold 时，
// Forward 方法应返回 UpstreamFailoverError，触发 Handler 切换账号
func TestAntigravityGatewayService_Forward_ModelRateLimitTriggersFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"model": "claude-opus-4-6",
		"messages": []map[string]any{
			{"role": "user", "content": "hi"},
		},
		"max_tokens": 1,
		"stream":     false,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	c.Request = req

	// 不需要真正调用上游，因为预检查会直接返回切换信号
	svc := &AntigravityGatewayService{
		tokenProvider: &AntigravityTokenProvider{},
		httpUpstream:  &httpUpstreamStub{resp: nil, err: nil},
	}

	// 设置模型限流：剩余时间 30 秒（> antigravityRateLimitThreshold 7s）
	futureResetAt := time.Now().Add(30 * time.Second).Format(time.RFC3339)
	account := &Account{
		ID:          1,
		Name:        "acc-rate-limited",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"project_id":   "proj",
		},
		Extra: map[string]any{
			modelRateLimitsKey: map[string]any{
				"claude-opus-4-6-thinking": map[string]any{
					"rate_limit_reset_at": futureResetAt,
				},
			},
		},
	}

	result, err := svc.Forward(context.Background(), c, account, body, false)
	require.Nil(t, result, "Forward should not return result when model rate limited")
	require.NotNil(t, err, "Forward should return error")

	// 核心验证：错误应该是 UpstreamFailoverError，而不是普通 502 错误
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr, "error should be UpstreamFailoverError to trigger account switch")
	require.Equal(t, http.StatusServiceUnavailable, failoverErr.StatusCode)
	// 非粘性会话请求，ForceCacheBilling 应为 false
	require.False(t, failoverErr.ForceCacheBilling, "ForceCacheBilling should be false for non-sticky session")
}

// TestAntigravityGatewayService_ForwardGemini_ModelRateLimitTriggersFailover
// 验证：ForwardGemini 方法同样能正确将 AntigravityAccountSwitchError 转换为 UpstreamFailoverError
func TestAntigravityGatewayService_ForwardGemini_ModelRateLimitTriggersFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]any{{"text": "hi"}}},
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", bytes.NewReader(body))
	c.Request = req

	// 不需要真正调用上游，因为预检查会直接返回切换信号
	svc := &AntigravityGatewayService{
		tokenProvider: &AntigravityTokenProvider{},
		httpUpstream:  &httpUpstreamStub{resp: nil, err: nil},
	}

	// 设置模型限流：剩余时间 30 秒（> antigravityRateLimitThreshold 7s）
	futureResetAt := time.Now().Add(30 * time.Second).Format(time.RFC3339)
	account := &Account{
		ID:          2,
		Name:        "acc-gemini-rate-limited",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"project_id":   "proj",
		},
		Extra: map[string]any{
			modelRateLimitsKey: map[string]any{
				"gemini-2.5-flash": map[string]any{
					"rate_limit_reset_at": futureResetAt,
				},
			},
		},
	}

	result, err := svc.ForwardGemini(context.Background(), c, account, "gemini-2.5-flash", "generateContent", false, body, false)
	require.Nil(t, result, "ForwardGemini should not return result when model rate limited")
	require.NotNil(t, err, "ForwardGemini should return error")

	// 核心验证：错误应该是 UpstreamFailoverError，而不是普通 502 错误
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr, "error should be UpstreamFailoverError to trigger account switch")
	require.Equal(t, http.StatusServiceUnavailable, failoverErr.StatusCode)
	// 非粘性会话请求，ForceCacheBilling 应为 false
	require.False(t, failoverErr.ForceCacheBilling, "ForceCacheBilling should be false for non-sticky session")
}

func TestAntigravityGatewayService_LocalTokenErrorsAreNotFailoverErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	account := &Account{
		ID:          20,
		Name:        "missing-local-token",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{"project_id": "project"},
	}
	upstream := &queuedHTTPUpstreamStub{}
	svc := &AntigravityGatewayService{tokenProvider: &AntigravityTokenProvider{}, httpUpstream: upstream}

	t.Run("messages", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		body := []byte(`{"model":"claude-opus-4-6","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))

		result, err := svc.Forward(context.Background(), c, account, body, false)

		require.Nil(t, result)
		require.Error(t, err)
		var failoverErr *UpstreamFailoverError
		require.NotErrorAs(t, err, &failoverErr)
		require.False(t, HasOpsUpstreamAttempted(c))
		require.Zero(t, upstream.callCount)
	})

	t.Run("gemini", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		body := []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", bytes.NewReader(body))

		result, err := svc.ForwardGemini(context.Background(), c, account, "gemini-2.5-flash", "generateContent", false, body, false)

		require.Nil(t, result)
		require.Error(t, err)
		var failoverErr *UpstreamFailoverError
		require.NotErrorAs(t, err, &failoverErr)
		require.False(t, HasOpsUpstreamAttempted(c))
		require.Zero(t, upstream.callCount)
	})
}

// TestAntigravityGatewayService_Forward_StickySessionForceCacheBilling
// 验证：粘性会话切换时，UpstreamFailoverError.ForceCacheBilling 应为 true
func TestAntigravityGatewayService_Forward_StickySessionForceCacheBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"model":    "claude-opus-4-6",
		"messages": []map[string]string{{"role": "user", "content": "hello"}},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	c.Request = req

	svc := &AntigravityGatewayService{
		tokenProvider: &AntigravityTokenProvider{},
		httpUpstream:  &httpUpstreamStub{resp: nil, err: nil},
	}

	// 设置模型限流：剩余时间 30 秒（> antigravityRateLimitThreshold 7s）
	futureResetAt := time.Now().Add(30 * time.Second).Format(time.RFC3339)
	account := &Account{
		ID:          3,
		Name:        "acc-sticky-rate-limited",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"project_id":   "proj",
		},
		Extra: map[string]any{
			modelRateLimitsKey: map[string]any{
				"claude-opus-4-6-thinking": map[string]any{
					"rate_limit_reset_at": futureResetAt,
				},
			},
		},
	}

	// 传入 isStickySession = true
	result, err := svc.Forward(context.Background(), c, account, body, true)
	require.Nil(t, result, "Forward should not return result when model rate limited")
	require.NotNil(t, err, "Forward should return error")

	// 核心验证：粘性会话切换时，ForceCacheBilling 应为 true
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr, "error should be UpstreamFailoverError to trigger account switch")
	require.Equal(t, http.StatusServiceUnavailable, failoverErr.StatusCode)
	require.True(t, failoverErr.ForceCacheBilling, "ForceCacheBilling should be true for sticky session switch")
}

// TestAntigravityGatewayService_ForwardGemini_StickySessionForceCacheBilling verifies
// that ForwardGemini sets ForceCacheBilling=true for sticky session switch.
func TestAntigravityGatewayService_ForwardGemini_StickySessionForceCacheBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]any{{"text": "hi"}}},
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", bytes.NewReader(body))
	c.Request = req

	svc := &AntigravityGatewayService{
		tokenProvider: &AntigravityTokenProvider{},
		httpUpstream:  &httpUpstreamStub{resp: nil, err: nil},
	}

	// 设置模型限流：剩余时间 30 秒（> antigravityRateLimitThreshold 7s）
	futureResetAt := time.Now().Add(30 * time.Second).Format(time.RFC3339)
	account := &Account{
		ID:          4,
		Name:        "acc-gemini-sticky-rate-limited",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"project_id":   "proj",
		},
		Extra: map[string]any{
			modelRateLimitsKey: map[string]any{
				"gemini-2.5-flash": map[string]any{
					"rate_limit_reset_at": futureResetAt,
				},
			},
		},
	}

	// 传入 isStickySession = true
	result, err := svc.ForwardGemini(context.Background(), c, account, "gemini-2.5-flash", "generateContent", false, body, true)
	require.Nil(t, result, "ForwardGemini should not return result when model rate limited")
	require.NotNil(t, err, "ForwardGemini should return error")

	// 核心验证：粘性会话切换时，ForceCacheBilling 应为 true
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr, "error should be UpstreamFailoverError to trigger account switch")
	require.Equal(t, http.StatusServiceUnavailable, failoverErr.StatusCode)
	require.True(t, failoverErr.ForceCacheBilling, "ForceCacheBilling should be true for sticky session switch")
}

func TestAntigravityGatewayService_ForwardGemini_ClearsStickySessionOnGeminiRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]any{{"text": "hi"}}},
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-3-flash-preview:generateContent", bytes.NewReader(body))
	c.Request = req

	respBody := []byte(`{
		"error": {
			"status": "RESOURCE_EXHAUSTED",
			"details": [
				{"@type": "type.googleapis.com/google.rpc.ErrorInfo", "metadata": {"model": "gemini-3-flash"}, "reason": "RATE_LIMIT_EXCEEDED"},
				{"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "15s"}
			]
		}
	}`)
	upstream := &httpUpstreamStub{resp: &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader(respBody)),
	}}
	repo := &stubAntigravityAccountRepo{}
	cache := &stubSmartRetryCache{}
	svc := &AntigravityGatewayService{
		tokenProvider: &AntigravityTokenProvider{},
		httpUpstream:  upstream,
		accountRepo:   repo,
		cache:         cache,
	}

	account := &Account{
		ID:          44,
		Name:        "acc-gemini-runtime-rate-limited",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"expires_at":   time.Now().Add(time.Hour).Format(time.RFC3339),
			"project_id":   "proj",
		},
		Extra: map[string]any{
			"mixed_scheduling": true,
		},
	}

	result, err := svc.ForwardGemini(
		context.Background(),
		c,
		account,
		"gemini-3-flash-preview",
		"generateContent",
		false,
		body,
		true,
		WithForwardGeminiSession(77, "gemini:sticky-runtime"),
	)

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusServiceUnavailable, failoverErr.StatusCode)
	require.Len(t, repo.modelRateLimitCalls, 2)
	require.Equal(t, "gemini-3-flash", repo.modelRateLimitCalls[0].modelKey)
	require.Equal(t, antigravityGeminiModelRateLimitKey, repo.modelRateLimitCalls[1].modelKey)
	require.Len(t, cache.deleteCalls, 1)
	require.Equal(t, int64(77), cache.deleteCalls[0].groupID)
	require.Equal(t, "gemini:sticky-runtime", cache.deleteCalls[0].sessionHash)
}

// TestAntigravityGatewayService_Forward_BillsWithMappedModel
// 验证：Antigravity Claude 转发返回的计费模型使用映射后的模型
func TestAntigravityGatewayService_Forward_BillsWithMappedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"model": "claude-sonnet-4-5",
		"messages": []map[string]any{
			{"role": "user", "content": "hello"},
		},
		"max_tokens": 16,
		"stream":     true,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	c.Request = req

	upstreamBody := []byte("data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":8,\"candidatesTokenCount\":3}}}\n\n")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"req-bill-1"}},
		Body:       io.NopCloser(bytes.NewReader(upstreamBody)),
	}

	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   &httpUpstreamStub{resp: resp},
	}

	const mappedModel = "gemini-3-pro-high"
	account := &Account{
		ID:          5,
		Name:        "acc-forward-billing",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"project_id":   "proj",
			"model_mapping": map[string]any{
				"claude-sonnet-4-5": mappedModel,
			},
		},
	}

	result, err := svc.Forward(context.Background(), c, account, body, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "claude-sonnet-4-5", result.Model)
	require.Equal(t, mappedModel, result.UpstreamModel)
}

// TestAntigravityGatewayService_ForwardGemini_BillsWithMappedModel
// 验证：Antigravity Gemini 转发返回的计费模型使用映射后的模型
func TestAntigravityGatewayService_ForwardGemini_BillsWithMappedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]any{{"text": "hello"}}},
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", bytes.NewReader(body))
	c.Request = req

	upstreamBody := []byte("data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":8,\"candidatesTokenCount\":3}}}\n\n")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"req-bill-2"}},
		Body:       io.NopCloser(bytes.NewReader(upstreamBody)),
	}

	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   &httpUpstreamStub{resp: resp},
	}

	const mappedModel = "gemini-3-pro-high"
	account := &Account{
		ID:          6,
		Name:        "acc-gemini-billing",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"project_id":   "proj",
			"model_mapping": map[string]any{
				"gemini-2.5-flash": mappedModel,
			},
		},
	}

	result, err := svc.ForwardGemini(context.Background(), c, account, "gemini-2.5-flash", "generateContent", true, body, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "gemini-2.5-flash", result.Model)
	require.Equal(t, mappedModel, result.UpstreamModel)
}

func TestAntigravityGatewayService_Forward_PassthroughFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"model": "claude-sonnet-4-5",
		"messages": []map[string]any{
			{"role": "user", "content": "hello"},
		},
		"max_tokens": 16,
		"stream":     true,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	c.Request = req

	upstreamBody := []byte("data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":8,\"candidatesTokenCount\":3}}}\n\n")
	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Request-Id": []string{"req-pass-1"}},
			Body:       io.NopCloser(bytes.NewReader(upstreamBody)),
		}},
	}

	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		httpUpstream:   upstream,
	}

	account := &Account{
		ID:          9,
		Name:        "acc-antigravity-apikey-pass",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "api-key-token",
			"model_mapping": map[string]any{
				"claude-sonnet-4-5": "gemini-3-pro-high",
			},
		},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-123"},
			},
		},
	}

	result, err := svc.Forward(context.Background(), c, account, body, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requestBodies, 1)
	require.Equal(t, "user-123", gjson.GetBytes(upstream.requestBodies[0], "metadata.user_id").String())
}

func TestAntigravityGatewayService_Forward_PassthroughFieldsHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"model": "claude-sonnet-4-5",
		"messages": []map[string]any{
			{"role": "user", "content": "hello"},
		},
		"max_tokens": 16,
		"stream":     true,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("X-Trace-Id", "trace-123")
	c.Request = req

	upstreamBody := []byte("data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":8,\"candidatesTokenCount\":3}}}\n\n")
	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Request-Id": []string{"req-pass-header-1"}},
			Body:       io.NopCloser(bytes.NewReader(upstreamBody)),
		}},
	}

	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		httpUpstream:   upstream,
	}

	account := &Account{
		ID:          91,
		Name:        "acc-antigravity-apikey-pass-header",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "api-key-token",
			"model_mapping": map[string]any{
				"claude-sonnet-4-5": "gemini-3-pro-high",
			},
		},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "header", Mode: "inject", Key: "X-Account-Tag", Value: "prod"},
				{Target: "header", Mode: "forward", Key: "X-Trace-Id"},
			},
		},
	}

	result, err := svc.Forward(context.Background(), c, account, body, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requestHeaders, 1)
	require.Equal(t, "prod", upstream.requestHeaders[0].Get("X-Account-Tag"))
	require.Equal(t, "trace-123", upstream.requestHeaders[0].Get("X-Trace-Id"))
}

func TestAntigravityGatewayService_ForwardGemini_PassthroughFieldsStructureConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logSink, restore := captureStructuredLog(t)
	defer restore()

	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"metadata": "string",
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]any{{"text": "hello"}}},
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", bytes.NewReader(body))
	c.Request = req

	svc := &AntigravityGatewayService{}
	account := &Account{
		ID:       10,
		Name:     "acc-antigravity-apikey-conflict",
		Platform: PlatformAntigravity,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "inject", Key: "metadata.user_id", Value: "user-123"},
			},
		},
	}

	result, err := svc.ForwardGemini(context.Background(), c, account, "gemini-2.5-flash", "generateContent", false, body, false)
	require.Nil(t, result)
	require.EqualError(t, err, "invalid_request_error: passthrough body path conflicts with non-object node: metadata")
	require.Equal(t, http.StatusBadRequest, writer.Code)
	require.True(t, logSink.ContainsMessage("passthrough body path conflicts with non-object node"))
	require.True(t, logSink.ContainsFieldValue("account_id", "10"))
	require.True(t, logSink.ContainsFieldValue("conflict_node", "metadata"))
}

func TestAntigravityGatewayService_ForwardGemini_FallbackReportsActualUpstreamModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-primary:generateContent", bytes.NewReader(body))

	const (
		originalModel = "gemini-primary"
		mappedModel   = "gemini-primary-upstream"
		fallbackModel = "gemini-fallback-upstream"
	)
	upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{
		{
			StatusCode: http.StatusNotFound,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"code":404,"message":"model not found"}}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(
				`data: {"response":{"modelVersion":"gemini-fallback-upstream","candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":8,"candidatesTokenCount":3}}}` + "\n\n",
			)),
		},
	}}
	settings := &antigravitySettingRepoValuesStub{values: map[string]string{
		SettingKeyEnableModelFallback:      "true",
		SettingKeyFallbackModelAntigravity: fallbackModel,
	}}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(settings, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}
	account := &Account{
		ID:          9,
		Name:        "acc-gemini-fallback",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"project_id":   "proj",
			"model_mapping": map[string]any{
				originalModel: mappedModel,
			},
		},
	}

	result, err := svc.ForwardGemini(context.Background(), c, account, originalModel, "generateContent", true, body, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, originalModel, result.Model)
	require.Equal(t, fallbackModel, result.UpstreamModel)
	require.Equal(t, fallbackModel, result.UpstreamResponseModel)
	require.False(t, result.UpstreamResponseModelConflict)
	mismatch := upstreamModelMismatch(result.UpstreamModel, result.UpstreamResponseModel)
	require.NotNil(t, mismatch)
	require.False(t, *mismatch)
	require.Len(t, upstream.requestBodies, 2)
	require.Contains(t, string(upstream.requestBodies[0]), `"model":"`+mappedModel+`"`)
	require.Contains(t, string(upstream.requestBodies[1]), `"model":"`+fallbackModel+`"`)
}

func TestAntigravityGatewayService_ForwardGemini_RetriesCorruptedThoughtSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]any{{"text": "hello"}}},
			{"role": "model", "parts": []map[string]any{{"text": "thinking", "thought": true, "thoughtSignature": "sig_bad_1"}}},
			{"role": "model", "parts": []map[string]any{{"functionCall": map[string]any{"name": "toolA", "args": map[string]any{"x": 1}}, "thoughtSignature": "sig_bad_2"}}},
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/antigravity/v1beta/models/gemini-3.1-pro-preview:streamGenerateContent", bytes.NewReader(body))
	c.Request = req

	firstRespBody := []byte(`{"response":{"error":{"code":400,"message":"Corrupted thought signature.","status":"INVALID_ARGUMENT"}}}`)
	secondRespBody := []byte("data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":8,\"candidatesTokenCount\":3}}}\n\n")

	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"X-Request-Id": []string{"req-sig-1"},
				},
				Body: io.NopCloser(bytes.NewReader(firstRespBody)),
			},
			{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"text/event-stream"},
					"X-Request-Id": []string{"req-sig-2"},
				},
				Body: io.NopCloser(bytes.NewReader(secondRespBody)),
			},
		},
	}

	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}

	const originalModel = "gemini-3.1-pro-preview"
	const mappedModel = "gemini-3.1-pro-high"
	account := &Account{
		ID:          7,
		Name:        "acc-gemini-signature",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"project_id":   "proj",
			"model_mapping": map[string]any{
				originalModel: mappedModel,
			},
		},
	}

	result, err := svc.ForwardGemini(context.Background(), c, account, originalModel, "streamGenerateContent", true, body, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, originalModel, result.Model)
	require.Equal(t, mappedModel, result.UpstreamModel)
	require.Len(t, upstream.requestBodies, 2, "signature error should trigger exactly one retry")

	firstReq := string(upstream.requestBodies[0])
	secondReq := string(upstream.requestBodies[1])
	require.Contains(t, firstReq, `"thoughtSignature":"sig_bad_1"`)
	require.Contains(t, firstReq, `"thoughtSignature":"sig_bad_2"`)
	require.Contains(t, secondReq, `"thoughtSignature":"skip_thought_signature_validator"`)
	require.NotContains(t, secondReq, `"thoughtSignature":"sig_bad_1"`)
	require.NotContains(t, secondReq, `"thoughtSignature":"sig_bad_2"`)

	raw, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := raw.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.NotEmpty(t, events)
	require.Equal(t, "signature_error", events[0].Kind)
}

func TestAntigravityGatewayService_ForwardGemini_SignatureRetryBuilderSpoolErrorReturnsSentinel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)
	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]},{"role":"model","parts":[{"text":"thinking","thought":true,"thoughtSignature":"bad-signature"}]}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/antigravity/v1beta/models/gemini-wave11:generateContent", bytes.NewReader(body))

	upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"response":{"error":{"code":400,"message":"Corrupted thought signature.","status":"INVALID_ARGUMENT"}}}`)),
	}}}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}
	account := &Account{
		ID: 110, Name: "signature-builder-spool", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Status: StatusActive, Concurrency: 1,
		Credentials: map[string]any{"access_token": "token", "project_id": "project", "model_mapping": map[string]any{"gemini-wave11": "gemini-wave11-mapped"}},
	}

	spoolErr := fmt.Errorf("build signature retry request: %w", ErrRequestBodySpool)
	oldBuilder := newAntigravitySignatureRetryPayloadRequest
	builderCalls := 0
	writtenAtBuilder := true
	newAntigravitySignatureRetryPayloadRequest = func(*antigravityRetryLoopParams, string) (*http.Request, error) {
		builderCalls++
		writtenAtBuilder = c.Writer.Written()
		return nil, spoolErr
	}
	t.Cleanup(func() { newAntigravitySignatureRetryPayloadRequest = oldBuilder })

	result, err := svc.ForwardGemini(context.Background(), c, account, "gemini-wave11", "generateContent", false, body, false)

	require.Nil(t, result)
	require.True(t, err == spoolErr, "spool sentinel must be returned without wrapping")
	require.Equal(t, 1, builderCalls, "signature retry must reach the real builder boundary exactly once")
	require.Equal(t, 1, upstream.callCount, "initial request must succeed before the signature retry builder fails")
	require.False(t, writtenAtBuilder, "service must not commit the response before returning the sentinel")
	require.False(t, c.Writer.Written())
}

func TestAntigravityRetryLoop_NilPayloadRequestReturnsError(t *testing.T) {
	svc := &AntigravityGatewayService{}
	result, err := svc.antigravityRetryLoop(antigravityRetryLoopParams{
		ctx:     context.Background(),
		account: &Account{},
		requestBuilder: func(*antigravityRetryLoopParams, string) (*http.Request, error) {
			return nil, nil
		},
	})

	require.Nil(t, result)
	require.EqualError(t, err, "antigravity request builder returned nil request")
}

func TestAntigravityGatewayService_ForwardGemini_SignatureRetryPropagatesFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]any{{"text": "hello"}}},
			{"role": "model", "parts": []map[string]any{{"text": "thinking", "thought": true, "thoughtSignature": "sig_bad_1"}}},
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/antigravity/v1beta/models/gemini-3.1-pro-preview:streamGenerateContent", bytes.NewReader(body))
	c.Request = req

	firstRespBody := []byte(`{"response":{"error":{"code":400,"message":"Corrupted thought signature.","status":"INVALID_ARGUMENT"}}}`)

	const originalModel = "gemini-3.1-pro-preview"
	const mappedModel = "gemini-3.1-pro-high"
	account := &Account{
		ID:          8,
		Name:        "acc-gemini-signature-failover",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"project_id":   "proj",
			"model_mapping": map[string]any{
				originalModel: mappedModel,
			},
		},
	}

	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"X-Request-Id": []string{"req-sig-failover-1"},
				},
				Body: io.NopCloser(bytes.NewReader(firstRespBody)),
			},
		},
		onCall: func(_ *http.Request, stub *queuedHTTPUpstreamStub) {
			if stub.callCount != 1 {
				return
			}
			futureResetAt := time.Now().Add(30 * time.Second).Format(time.RFC3339)
			account.Extra = map[string]any{
				modelRateLimitsKey: map[string]any{
					mappedModel: map[string]any{
						"rate_limit_reset_at": futureResetAt,
					},
				},
			}
		},
	}

	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}

	result, err := svc.ForwardGemini(context.Background(), c, account, originalModel, "streamGenerateContent", true, body, true)
	require.Nil(t, result)

	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr, "signature retry should propagate failover instead of falling back to the original 400")
	require.Equal(t, http.StatusServiceUnavailable, failoverErr.StatusCode)
	require.True(t, failoverErr.ForceCacheBilling)
	require.Len(t, upstream.requestBodies, 1, "retry should stop at preflight failover and not issue a second upstream request")

	raw, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok)
	events, ok := raw.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	require.Len(t, events, 2)
	require.Equal(t, "signature_error", events[0].Kind)
	require.Equal(t, "failover", events[1].Kind)
}

func TestAntigravityGatewayService_ForwardGemini_ModelFallbackUpdatesUsageSnapshotToFallbackRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]any{{"text": "hello"}}},
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/antigravity/v1beta/models/gemini-2.5-flash:streamGenerateContent", bytes.NewReader(body))
	c.Request = req

	collector := &antigravityUsageUpstreamRequestCollectorStub{}
	c.Set(UsageDetailCaptureContextKey, collector)

	firstRespBody := []byte(`{"error":{"message":"model not found"}}`)
	secondRespBody := []byte("data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":8,\"candidatesTokenCount\":3}}}\n\n")

	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{
			{
				StatusCode: http.StatusNotFound,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"X-Request-Id": []string{"req-fallback-1"},
				},
				Body: io.NopCloser(bytes.NewReader(firstRespBody)),
			},
			{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"text/event-stream"},
					"X-Request-Id": []string{"req-fallback-2"},
				},
				Body: io.NopCloser(bytes.NewReader(secondRespBody)),
			},
		},
	}

	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoValuesStub{values: map[string]string{
			SettingKeyEnableModelFallback:      "true",
			SettingKeyFallbackModelAntigravity: "gemini-2.5-pro",
		}}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider: &AntigravityTokenProvider{},
		httpUpstream:  upstream,
	}

	const originalModel = "gemini-2.5-flash"
	const mappedModel = "gemini-2.5-flash-mapped"
	const fallbackModel = "gemini-2.5-pro"
	account := &Account{
		ID:          11,
		Name:        "acc-gemini-fallback-snapshot",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token": "token",
			"project_id":   "proj",
			"model_mapping": map[string]any{
				originalModel: mappedModel,
			},
		},
	}

	result, err := svc.ForwardGemini(context.Background(), c, account, originalModel, "streamGenerateContent", true, body, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requestBodies, 2)
	require.Len(t, upstream.requestHeaders, 2)
	require.Contains(t, string(upstream.requestBodies[0]), mappedModel)
	require.Contains(t, string(upstream.requestBodies[1]), fallbackModel)
	require.Equal(t, string(upstream.requestBodies[1]), collector.body)
	require.Equal(t, ":method: POST\n:url: https://cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse\n"+FormatUsageDetailHeadersText(upstream.requestHeaders[1]), collector.headers)
	if len(upstream.requestBodies[0]) > 0 {
		require.NotEqual(t, string(upstream.requestBodies[0]), collector.body)
	}
}

func TestAntigravityGatewayService_ForwardGemini_ModelFallbackBuildSpoolErrorReturnsSentinel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)
	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/antigravity/v1beta/models/gemini-wave8:streamGenerateContent", bytes.NewReader(body))

	upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{{
		StatusCode: http.StatusNotFound,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"model not found"}}`)),
	}}}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoValuesStub{values: map[string]string{
			SettingKeyEnableModelFallback:      "true",
			SettingKeyFallbackModelAntigravity: "gemini-wave8-fallback",
		}}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider: &AntigravityTokenProvider{},
		httpUpstream:  upstream,
	}
	account := &Account{
		ID: 106, Name: "fallback-build-spool", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Status: StatusActive, Concurrency: 1,
		Credentials: map[string]any{"access_token": "token", "project_id": "project", "model_mapping": map[string]any{"gemini-wave8": "gemini-wave8-mapped"}},
	}
	oldBuilder := newAntigravityFallbackPayloadRequest
	builderCalls := 0
	writtenAtBuilder := true
	newAntigravityFallbackPayloadRequest = func(*antigravityRetryLoopParams, string) (*http.Request, error) {
		builderCalls++
		writtenAtBuilder = c.Writer.Written()
		return nil, fmt.Errorf("build fallback request: %w", ErrRequestBodySpool)
	}
	t.Cleanup(func() { newAntigravityFallbackPayloadRequest = oldBuilder })

	result, err := svc.ForwardGemini(context.Background(), c, account, "gemini-wave8", "streamGenerateContent", true, body, false)

	require.Nil(t, result)
	require.ErrorIs(t, err, ErrRequestBodySpool)
	require.Equal(t, 1, builderCalls)
	require.False(t, writtenAtBuilder, "service must not commit the response before returning the sentinel")
	require.False(t, c.Writer.Written())
	require.Equal(t, 1, upstream.callCount)
	require.Empty(t, writer.Body.String())
}

func TestPassthroughFieldsV2AntigravityForwardGemini_BodyInjectAndMapDoNotChain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"metadata": map[string]any{"client_trace": "trace-123"},
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]any{{"text": "hello"}}},
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", bytes.NewReader(body))
	c.Request = req

	upstreamBody := []byte("data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":8,\"candidatesTokenCount\":3}}}\n\n")
	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(bytes.NewReader(upstreamBody)),
		}},
	}

	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}

	account := &Account{
		ID:          13,
		Name:        "acc-antigravity-v2-map",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "api-key-token",
		},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "inject", Key: "requestMetadata.user_id", Value: "user-123"},
				{Target: "body", Mode: "map", Key: "requestMetadata.copied_user_id", SourceKey: "requestMetadata.user_id"},
				{Target: "body", Mode: "map", Key: "requestMetadata.client_trace", SourceKey: "metadata.client_trace"},
			},
		},
	}

	result, err := svc.ForwardGemini(context.Background(), c, account, "gemini-2.5-flash", "generateContent", true, body, false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requestBodies, 1)
	require.Equal(t, "user-123", gjson.GetBytes(upstream.requestBodies[0], "request.requestMetadata.user_id").String())
	require.False(t, gjson.GetBytes(upstream.requestBodies[0], "request.requestMetadata.copied_user_id").Exists())
	require.Equal(t, "trace-123", gjson.GetBytes(upstream.requestBodies[0], "request.requestMetadata.client_trace").String())
}

func TestAntigravityGatewayService_ForwardUpstream_CapturesUsageSnapshotBeforeSend(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body, err := json.Marshal(map[string]any{
		"model": "claude-sonnet-4-5",
		"messages": []map[string]any{
			{"role": "user", "content": "hello"},
		},
		"max_tokens": 16,
		"stream":     false,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "tools-2024-04-04")
	c.Request = req

	collector := &antigravityUsageUpstreamRequestCollectorStub{}
	c.Set(UsageDetailCaptureContextKey, collector)

	responseBody := []byte(`{"id":"msg_1","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":2}}`)
	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(responseBody)),
		}},
	}

	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		httpUpstream:   upstream,
	}

	account := &Account{
		ID:          12,
		Name:        "acc-forward-upstream-snapshot",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: map[string]any{
			"base_url": "https://example.com",
			"api_key":  "api-key-token",
		},
	}

	upstream.onCall = func(_ *http.Request, _ *queuedHTTPUpstreamStub) {
		require.True(t, HasOpsUpstreamAttempted(c))
	}
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{})
	require.NoError(t, err)
	defer CleanupRequestBodyHandle(handle)
	result, err := svc.ForwardUpstream(context.Background(), c, account, handle)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requestBodies, 1)
	require.Len(t, upstream.requestHeaders, 1)
	require.Equal(t, string(body), collector.body)
	require.Equal(t, ":method: POST\n:url: https://example.com/v1/messages\n"+FormatUsageDetailHeadersText(upstream.requestHeaders[0]), collector.headers)
	require.Contains(t, collector.headers, "Authorization: Bearer api-key-token")
	require.Contains(t, collector.headers, "X-Api-Key: api-key-token")
	require.Contains(t, collector.headers, "Anthropic-Version: 2023-06-01")
	require.Contains(t, collector.headers, "Anthropic-Beta: tools-2024-04-04")
}

func TestAntigravityGatewayService_ForwardUpstreamStripsContextManagementFromWireBodyWithoutBeta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"context_management":{"edits":[{"type":"clear_tool_uses_20250919"}]}}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"msg_1","usage":{"input_tokens":1,"output_tokens":1}}`)),
	}}}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		httpUpstream:   upstream,
	}
	account := &Account{ID: 76, Name: "upstream-sanitize", Platform: PlatformAntigravity, Type: AccountTypeUpstream, Concurrency: 1, Credentials: map[string]any{"base_url": "https://example.com", "api_key": "token"}}
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })

	result, err := svc.ForwardUpstream(context.Background(), c, account, handle)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requestBodies, 1)
	require.False(t, gjson.GetBytes(upstream.requestBodies[0], "context_management").Exists(), string(upstream.requestBodies[0]))
}

func TestAntigravityGatewayService_ForwardUpstreamTransportErrorClosesResponseBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	responseBody := &closeTrackingReadCloser{Reader: strings.NewReader(`{"error":"partial"}`)}
	transportErr := errors.New("upstream transport failed")
	upstream := &transportSpoolCloseUpstream{
		resp: &http.Response{StatusCode: http.StatusBadGateway, Header: http.Header{}, Body: responseBody},
		err:  transportErr,
	}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		httpUpstream:   upstream,
	}
	account := &Account{ID: 75, Name: "upstream-error", Platform: PlatformAntigravity, Type: AccountTypeUpstream, Concurrency: 1, Credentials: map[string]any{"base_url": "https://example.com", "api_key": "token"}}
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })

	result, err := svc.ForwardUpstream(context.Background(), c, account, handle)

	require.Nil(t, result)
	require.ErrorIs(t, err, transportErr)
	require.True(t, responseBody.closed)
	require.Empty(t, recorder.Body.String())
}

func TestAntigravityGatewayService_ForwardUpstreamKeepsLargeBodyFileBackedWhileBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)

	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"messages":[{"role":"user","content":"` + strings.Repeat("x", 10<<20) + `"}]}`)
	handle, err := NewRequestBodyHandleFromReader(bytes.NewReader(body), RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })

	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	upstream := &blockingUpstreamHandleHTTPStub{started: make(chan *http.Request, 1), release: make(chan struct{})}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		httpUpstream:   upstream,
	}
	account := &Account{ID: 77, Name: "upstream", Platform: PlatformAntigravity, Type: AccountTypeUpstream, Concurrency: 1, Credentials: map[string]any{"base_url": "https://example.com", "api_key": "token"}}

	done := make(chan error, 1)
	go func() {
		_, err := svc.ForwardHandle(context.Background(), c, account, handle, false)
		done <- err
	}()
	req := <-upstream.started
	if _, ok := req.Body.(requestBodySpoolReadCloser); !ok {
		t.Fatalf("upstream request body = %T, want requestBodySpoolReadCloser", req.Body)
	}
	reopened, err := req.GetBody()
	require.NoError(t, err)
	reopenedBody, err := io.ReadAll(reopened)
	require.NoError(t, err)
	require.NoError(t, reopened.Close())
	require.Equal(t, body, reopenedBody)
	require.Equal(t, int64(len(body)), req.ContentLength)
	close(upstream.release)
	require.NoError(t, <-done)
}

func TestAntigravityGatewayService_ClaudeForwardHandleSignatureRetryReparsesFileBackedCanonical(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)
	filler := strings.Repeat("x", int(DefaultRequestBodySpoolThresholdBytes)+1)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":4096,"stream":false,"thinking":{"type":"enabled","budget_tokens":1024},"messages":[{"role":"user","content":"start"},{"role":"assistant","content":[{"type":"thinking","thinking":"secret plan","signature":"bad-signature"},{"type":"tool_use","id":"tool-1","name":"lookup","input":{"query":"wave13"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool-1","content":"result"},{"type":"text","text":"` + filler + `"}]}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	canonicalHandle, err := NewRequestBodyHandleFromReader(bytes.NewReader(body), RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(canonicalHandle) })
	require.NotEmpty(t, canonicalHandle.spoolPath)

	signatureError := func() *http.Response {
		return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Corrupted thought signature."}}`))}
	}
	upstream := &claudeSignatureRetryHTTPUpstream{responses: []*http.Response{
		signatureError(),
		signatureError(),
		{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":1,\"candidatesTokenCount\":1}}}\n\n"))},
	}}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}
	account := &Account{ID: 112, Name: "claude-signature", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Concurrency: 1, Credentials: map[string]any{
		"access_token": "token", "project_id": "project", "model_mapping": map[string]any{"claude-sonnet-4-5": "gemini-3-pro-high"},
	}}

	result, err := svc.ForwardHandle(context.Background(), c, account, canonicalHandle, false)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requestBodies, 3)
	require.Contains(t, string(upstream.requestBodies[0]), `"thought":true`)
	require.Contains(t, string(upstream.requestBodies[0]), `"functionCall"`)
	require.NotContains(t, string(upstream.requestBodies[1]), `"thought":true`)
	require.Contains(t, string(upstream.requestBodies[1]), `"functionCall"`, "thinking-only retry must preserve tools")
	require.NotContains(t, string(upstream.requestBodies[2]), `"functionCall"`)
	require.NotContains(t, string(upstream.requestBodies[2]), `"functionResponse"`)
	require.Contains(t, string(upstream.requestBodies[2]), `(tool_use)`)
	require.Len(t, upstream.requestBodyPaths, 3)
	for _, path := range upstream.requestBodyPaths {
		require.NotEmpty(t, path, "large outbound payload must remain file-backed")
		require.NotEqual(t, canonicalHandle.spoolPath, path)
	}
	require.NotEqual(t, upstream.requestBodyPaths[0], upstream.requestBodyPaths[1])
	require.NotEqual(t, upstream.requestBodyPaths[1], upstream.requestBodyPaths[2])
	require.Equal(t, HashUsageRequestPayload(body), canonicalHandle.Hash())
	canonicalBody, err := canonicalHandle.ReadAll()
	require.NoError(t, err)
	require.Equal(t, body, canonicalBody)
}

func TestAntigravityGatewayService_ClaudeForwardHandleSignatureRetryCanonicalReadAllocation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":4096,"stream":false,"thinking":{"type":"enabled","budget_tokens":1024},"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"secret","signature":"bad"},{"type":"tool_use","id":"tool-1","name":"lookup","input":{}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool-1","content":"ok"}]}],"ignored":"` + strings.Repeat("x", 89<<20/10) + `"}`)
	canonicalHandle, err := NewRequestBodyHandleFromReader(bytes.NewReader(body), RequestBodyHandleOptions{SpoolThresholdBytes: 1, PreviewLimitBytes: 64, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(canonicalHandle) })

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	materialized, err := canonicalHandle.ReadAll()
	require.NoError(t, err)
	require.Equal(t, int(canonicalHandle.Size()), len(materialized))
	runtime.ReadMemStats(&after)
	oneReadAlloc := after.TotalAlloc - before.TotalAlloc
	runtime.GC()

	signatureError := func() *http.Response {
		return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Corrupted thought signature."}}`))}
	}
	upstream := &claudeSignatureRetryDiscardUpstream{responses: []*http.Response{
		signatureError(),
		signatureError(),
		{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":1,\"candidatesTokenCount\":1}}}\n\n"))},
	}}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}
	account := &Account{ID: 115, Name: "claude-signature-allocation", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Concurrency: 1, Credentials: map[string]any{
		"access_token": "token", "project_id": "project", "model_mapping": map[string]any{"claude-sonnet-4-5": "gemini-3-pro-high"},
	}}

	runtime.ReadMemStats(&before)
	result, err := svc.ForwardHandle(context.Background(), c, account, canonicalHandle, false)
	runtime.ReadMemStats(&after)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 3, upstream.calls)
	forwardAlloc := after.TotalAlloc - before.TotalAlloc
	t.Logf("one_read_alloc=%d signature_retry_alloc=%d", oneReadAlloc, forwardAlloc)
	require.Less(t, forwardAlloc, 3*oneReadAlloc+(16<<20), "initial send and two signature retries must each materialize canonical once")
}

func TestAntigravityGatewayService_ClaudeForwardHandleSignatureRetryPreservesFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":4096,"thinking":{"type":"enabled","budget_tokens":1024},"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"secret","signature":"bad"}]}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	handle, err := NewRequestBodyHandleFromReader(bytes.NewReader(body), RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })
	upstream := &claudeSignatureRetryHTTPUpstream{responses: []*http.Response{
		{StatusCode: http.StatusBadRequest, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Corrupted thought signature."}}`))},
		{StatusCode: http.StatusBadRequest, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"invalid project resource name"}}`))},
	}}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}
	account := &Account{ID: 113, Name: "claude-signature-failover", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Concurrency: 1, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}

	result, err := svc.ForwardHandle(context.Background(), c, account, handle, false)

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusBadRequest, failoverErr.StatusCode)
	require.True(t, failoverErr.RetryableOnSameAccount)
	require.Len(t, upstream.requestBodies, 2)
	require.False(t, c.Writer.Written())
}

func TestAntigravityGatewayService_ClaudeForwardHandleSignatureRetryPreservesSpoolSentinel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":4096,"thinking":{"type":"enabled","budget_tokens":1024},"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"secret","signature":"bad"}]}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	handle, err := NewRequestBodyHandleFromReader(bytes.NewReader(body), RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(func() { CleanupRequestBodyHandle(handle) })
	upstream := &claudeSignatureRetryHTTPUpstream{responses: []*http.Response{{
		StatusCode: http.StatusBadRequest, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"Corrupted thought signature."}}`)),
	}}}
	upstream.onCall = func(call int) {
		if call == 1 {
			require.NoError(t, os.Remove(handle.spoolPath))
		}
	}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}
	account := &Account{ID: 114, Name: "claude-signature-spool", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Concurrency: 1, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}

	result, err := svc.ForwardHandle(context.Background(), c, account, handle, false)

	require.Nil(t, result)
	require.ErrorIs(t, err, ErrRequestBodySpool)
	require.Len(t, upstream.requestBodies, 1)
	require.False(t, c.Writer.Written())
}

func TestAntigravityGatewayService_ForwardEnabledThinkingRetriesInvalidBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"thinking":{"type":"enabled","budget_tokens":1024},"messages":[{"role":"user","content":"hello"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))

	upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{
		{StatusCode: http.StatusBadRequest, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"thinking budget_tokens must be >= 1024"}}`))},
		{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"response":{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}}`))},
	}}
	svc := &AntigravityGatewayService{
		settingService: NewSettingService(&antigravitySettingRepoStub{}, &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}
	account := &Account{ID: 78, Name: "budget", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Concurrency: 1, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}

	_, _ = svc.Forward(context.Background(), c, account, body, false)
	require.Len(t, upstream.requestBodies, 2)
	require.Equal(t, int64(BudgetRectifyBudgetTokens), gjson.GetBytes(upstream.requestBodies[1], "request.generationConfig.thinkingConfig.thinkingBudget").Int())
}

func TestAntigravityGatewayService_ForwardBudgetRectifierClearsProvidedStickySession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)
	body := []byte(`{"model":"claude-sonnet-4-5","max_tokens":16,"thinking":{"type":"enabled","budget_tokens":1024},"messages":[{"role":"user","content":"hello"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))

	upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{
		{StatusCode: http.StatusBadRequest, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"thinking budget_tokens must be >= 1024"}}`))},
		{StatusCode: http.StatusTooManyRequests, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"status":"RESOURCE_EXHAUSTED","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","metadata":{"model":"gemini-3-flash"},"reason":"RATE_LIMIT_EXCEEDED"},{"@type":"type.googleapis.com/google.rpc.RetryInfo","retryDelay":"15s"}]}}`))},
	}}
	cfg := &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}}
	cfg.Gateway.Sticky.Anthropic.Enabled = true
	cache := &stubSmartRetryCache{}
	svc := &AntigravityGatewayService{
		accountRepo:    &stubAntigravityAccountRepo{},
		cache:          cache,
		settingService: NewSettingService(&antigravitySettingRepoStub{}, cfg),
		tokenProvider:  &AntigravityTokenProvider{},
		httpUpstream:   upstream,
	}
	account := &Account{ID: 79, Name: "budget-cleanup", Platform: PlatformAntigravity, Type: AccountTypeOAuth, Concurrency: 1, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}

	_, _ = svc.Forward(context.Background(), c, account, body, false, WithForwardGeminiSession(79, "gemini:budget-rectifier"))

	require.Len(t, upstream.requestBodies, 2)
	require.Len(t, cache.deleteCalls, 1)
	require.Equal(t, int64(79), cache.deleteCalls[0].groupID)
	require.Equal(t, "gemini:budget-rectifier", cache.deleteCalls[0].sessionHash)
}

type blockingUpstreamHandleHTTPStub struct {
	HTTPUpstream
	started chan *http.Request
	release chan struct{}
}

func (u *blockingUpstreamHandleHTTPStub) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.started <- req
	<-u.release
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"msg_1","usage":{"input_tokens":1,"output_tokens":1}}`))}, nil
}

func TestAntigravityCreditsRetryCapturesEachAttemptSnapshot(t *testing.T) {
	oldBaseURLs := append([]string(nil), antigravity.BaseURLs...)
	oldAvailability := antigravity.DefaultURLAvailability
	defer func() {
		antigravity.BaseURLs = oldBaseURLs
		antigravity.DefaultURLAvailability = oldAvailability
	}()
	antigravity.BaseURLs = []string{"https://ag-credits.test"}
	antigravity.DefaultURLAvailability = antigravity.NewURLAvailability(time.Minute)

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	collector := &antigravityUsageUpstreamRequestCollectorStub{}
	c.Set(UsageDetailCaptureContextKey, collector)
	upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{
		{StatusCode: http.StatusTooManyRequests, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"status":"RESOURCE_EXHAUSTED","message":"QUOTA_EXHAUSTED"}}`))},
		{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))},
	}}
	var usageBodies, opsBodies []string
	var attempts []bool
	upstream.onCall = func(_ *http.Request, _ *queuedHTTPUpstreamStub) {
		usageBodies = append(usageBodies, collector.body)
		value, _ := c.Get(OpsUpstreamRequestBodyKey)
		opsBody, ok := value.(string)
		require.True(t, ok)
		opsBodies = append(opsBodies, opsBody)
		attempts = append(attempts, HasOpsUpstreamAttempted(c))
		SetOpsUpstreamAttempted(c, false)
	}
	account := &Account{ID: 903, Name: "credits-snapshots", Type: AccountTypeOAuth, Platform: PlatformAntigravity, Concurrency: 1, Extra: map[string]any{"allow_overages": true}}
	body := []byte(`{"model":"claude-sonnet-4-5","request":{"contents":[{"parts":[{"text":"hello"}]}]}}`)

	resp, err := (&AntigravityGatewayService{}).antigravityRetryLoop(antigravityRetryLoopParams{
		ctx: context.Background(), c: c, prefix: "[test]", account: account, accessToken: "token", action: "generateContent", body: body,
		httpUpstream: upstream, requestedModel: "claude-sonnet-4-5",
		handleError: func(context.Context, string, *Account, int, http.Header, []byte, string, int64, string, bool) *handleModelRateLimitResult {
			return nil
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, []bool{true, true}, attempts)
	require.Len(t, usageBodies, 2)
	require.Len(t, opsBodies, 2)
	require.NotContains(t, usageBodies[0], "enabledCreditTypes")
	require.NotContains(t, opsBodies[0], "enabledCreditTypes")
	require.Contains(t, usageBodies[1], "enabledCreditTypes")
	require.Contains(t, opsBodies[1], "enabledCreditTypes")
}

// TestStreamUpstreamResponse_UsageAndFirstToken
// 验证：usage 字段可被累积/覆盖更新，并且能记录首 token 时间
func TestStreamUpstreamResponse_UsageAndFirstToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: pr}

	go func() {
		defer func() { _ = pw.Close() }()
		fmt.Fprintln(pw, `data: {"usage":{"input_tokens":1,"output_tokens":2,"cache_read_input_tokens":3,"cache_creation_input_tokens":4}}`)
		fmt.Fprintln(pw, `data: {"usage":{"output_tokens":5}}`)
	}()

	start := time.Now().Add(-10 * time.Millisecond)
	result := svc.streamUpstreamResponse(c, resp, start)
	_ = pr.Close()

	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.Equal(t, 1, result.usage.InputTokens)
	// 第二次事件覆盖 output_tokens
	require.Equal(t, 5, result.usage.OutputTokens)
	require.Equal(t, 3, result.usage.CacheReadInputTokens)
	require.Equal(t, 4, result.usage.CacheCreationInputTokens)
	require.NotNil(t, result.firstTokenMs)

	// 确保有透传输出
	require.Contains(t, rec.Body.String(), "data:")
}

// --- 流式 happy path 测试 ---

// TestStreamUpstreamResponse_NormalComplete
// 验证：正常流式转发完成时，数据正确透传、usage 正确收集、clientDisconnect=false
func TestStreamUpstreamResponse_NormalComplete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Body: pr, Header: http.Header{}}

	go func() {
		defer func() { _ = pw.Close() }()
		fmt.Fprintln(pw, `event: message_start`)
		fmt.Fprintln(pw, `data: {"type":"message_start","message":{"usage":{"input_tokens":10}}}`)
		fmt.Fprintln(pw, "")
		fmt.Fprintln(pw, `event: content_block_delta`)
		fmt.Fprintln(pw, `data: {"type":"content_block_delta","delta":{"text":"hello"}}`)
		fmt.Fprintln(pw, "")
		fmt.Fprintln(pw, `event: message_delta`)
		fmt.Fprintln(pw, `data: {"type":"message_delta","usage":{"output_tokens":5}}`)
		fmt.Fprintln(pw, "")
	}()

	result := svc.streamUpstreamResponse(c, resp, time.Now())
	_ = pr.Close()

	require.NotNil(t, result)
	require.False(t, result.clientDisconnect, "normal completion should not set clientDisconnect")
	require.NotNil(t, result.usage)
	require.Equal(t, 5, result.usage.OutputTokens, "should collect output_tokens from message_delta")
	require.NotNil(t, result.firstTokenMs, "should record first token time")

	// 验证数据被透传到客户端
	body := rec.Body.String()
	require.Contains(t, body, "event: message_start")
	require.Contains(t, body, "content_block_delta")
	require.Contains(t, body, "message_delta")
}

// TestHandleGeminiStreamingResponse_NormalComplete
// 验证：正常 Gemini 流式转发，数据正确透传、usage 正确收集
func TestHandleGeminiStreamingResponse_NormalComplete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Body: pr, Header: http.Header{}}

	go func() {
		defer func() { _ = pw.Close() }()
		// 第一个 chunk（部分内容）
		fmt.Fprintln(pw, `data: {"candidates":[{"content":{"parts":[{"text":"Hello"}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":3}}`)
		fmt.Fprintln(pw, "")
		// 第二个 chunk（最终内容+完整 usage）
		fmt.Fprintln(pw, `data: {"candidates":[{"content":{"parts":[{"text":" world"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":8,"cachedContentTokenCount":2}}`)
		fmt.Fprintln(pw, "")
	}()

	result, err := svc.handleGeminiStreamingResponse(c, resp, time.Now())
	_ = pr.Close()

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.clientDisconnect, "normal completion should not set clientDisconnect")
	require.NotNil(t, result.usage)
	// Gemini usage: promptTokenCount=10, candidatesTokenCount=8, cachedContentTokenCount=2
	// → InputTokens=10-2=8, OutputTokens=8, CacheReadInputTokens=2
	require.Equal(t, 8, result.usage.InputTokens)
	require.Equal(t, 8, result.usage.OutputTokens)
	require.Equal(t, 2, result.usage.CacheReadInputTokens)
	require.NotNil(t, result.firstTokenMs, "should record first token time")

	// 验证数据被透传到客户端
	body := rec.Body.String()
	require.Contains(t, body, "Hello")
	require.Contains(t, body, "world")
	// 不应包含错误事件
	require.NotContains(t, body, "event: error")
}

// TestHandleClaudeStreamingResponse_NormalComplete
// 验证：正常 Claude 流式转发（Gemini→Claude 转换），数据正确转换并输出
func TestHandleClaudeStreamingResponse_NormalComplete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Body: pr, Header: http.Header{}}

	go func() {
		defer func() { _ = pw.Close() }()
		// v1internal 包装格式：Gemini 数据嵌套在 "response" 字段下
		// ProcessLine 先尝试反序列化为 V1InternalResponse，裸格式会导致 Response.UsageMetadata 为空
		fmt.Fprintln(pw, `data: {"response":{"candidates":[{"content":{"parts":[{"text":"Hi there"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":3}}}`)
		fmt.Fprintln(pw, "")
	}()

	result, err := svc.handleClaudeStreamingResponse(c, resp, time.Now(), "claude-sonnet-4-5")
	_ = pr.Close()

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.clientDisconnect, "normal completion should not set clientDisconnect")
	require.NotNil(t, result.usage)
	// Gemini→Claude 转换的 usage：promptTokenCount=5→InputTokens=5, candidatesTokenCount=3→OutputTokens=3
	require.Equal(t, 5, result.usage.InputTokens)
	require.Equal(t, 3, result.usage.OutputTokens)
	require.NotNil(t, result.firstTokenMs, "should record first token time")

	// 验证输出是 Claude SSE 格式（processor 会转换）
	body := rec.Body.String()
	require.Contains(t, body, "event: message_start", "should contain Claude message_start event")
	require.Contains(t, body, "event: message_stop", "should contain Claude message_stop event")
	// 不应包含错误事件
	require.NotContains(t, body, "event: error")
}

// TestHandleGeminiStreamingResponse_ThoughtsTokenCount
// 验证：Gemini 流式转发时 thoughtsTokenCount 被计入 OutputTokens
func TestHandleGeminiStreamingResponse_ThoughtsTokenCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Body: pr, Header: http.Header{}}

	go func() {
		defer func() { _ = pw.Close() }()
		fmt.Fprintln(pw, `data: {"candidates":[{"content":{"parts":[{"text":"Hello"}]}}],"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":20,"thoughtsTokenCount":50}}`)
		fmt.Fprintln(pw, "")
		fmt.Fprintln(pw, `data: {"candidates":[{"content":{"parts":[{"text":" world"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":30,"thoughtsTokenCount":80,"cachedContentTokenCount":10}}`)
		fmt.Fprintln(pw, "")
	}()

	result, err := svc.handleGeminiStreamingResponse(c, resp, time.Now())
	_ = pr.Close()

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	// promptTokenCount=100, cachedContentTokenCount=10 → InputTokens=90
	require.Equal(t, 90, result.usage.InputTokens)
	// candidatesTokenCount=30 + thoughtsTokenCount=80 → OutputTokens=110
	require.Equal(t, 110, result.usage.OutputTokens)
	require.Equal(t, 10, result.usage.CacheReadInputTokens)
}

// TestHandleClaudeStreamingResponse_ThoughtsTokenCount
// 验证：Gemini→Claude 流式转换时 thoughtsTokenCount 被计入 OutputTokens
func TestHandleClaudeStreamingResponse_ThoughtsTokenCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Body: pr, Header: http.Header{}}

	go func() {
		defer func() { _ = pw.Close() }()
		fmt.Fprintln(pw, `data: {"response":{"candidates":[{"content":{"parts":[{"text":"Hi"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":50,"candidatesTokenCount":10,"thoughtsTokenCount":25}}}`)
		fmt.Fprintln(pw, "")
	}()

	result, err := svc.handleClaudeStreamingResponse(c, resp, time.Now(), "gemini-2.5-pro")
	_ = pr.Close()

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	// promptTokenCount=50 → InputTokens=50
	require.Equal(t, 50, result.usage.InputTokens)
	// candidatesTokenCount=10 + thoughtsTokenCount=25 → OutputTokens=35
	require.Equal(t, 35, result.usage.OutputTokens)
}

// --- 流式客户端断开检测测试 ---

// TestStreamUpstreamResponse_ClientDisconnectDrainsUsage
// 验证：客户端写入失败后，streamUpstreamResponse 继续读取上游以收集 usage
func TestStreamUpstreamResponse_ClientDisconnectDrainsUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Writer = &antigravityFailingWriter{ResponseWriter: c.Writer, failAfter: 0}

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Body: pr, Header: http.Header{}}

	go func() {
		defer func() { _ = pw.Close() }()
		fmt.Fprintln(pw, `event: message_start`)
		fmt.Fprintln(pw, `data: {"type":"message_start","message":{"usage":{"input_tokens":10}}}`)
		fmt.Fprintln(pw, "")
		fmt.Fprintln(pw, `event: message_delta`)
		fmt.Fprintln(pw, `data: {"type":"message_delta","usage":{"output_tokens":20}}`)
		fmt.Fprintln(pw, "")
	}()

	result := svc.streamUpstreamResponse(c, resp, time.Now())
	_ = pr.Close()

	require.NotNil(t, result)
	require.True(t, result.clientDisconnect)
	require.NotNil(t, result.usage)
	require.Equal(t, 20, result.usage.OutputTokens)
}

// TestStreamUpstreamResponse_ContextCanceled
// 验证：context 取消时返回 usage 且标记 clientDisconnect
func TestStreamUpstreamResponse_ContextCanceled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)

	resp := &http.Response{StatusCode: http.StatusOK, Body: cancelReadCloser{}, Header: http.Header{}}

	result := svc.streamUpstreamResponse(c, resp, time.Now())

	require.NotNil(t, result)
	require.True(t, result.clientDisconnect)
	require.NotContains(t, rec.Body.String(), "event: error")
}

// TestStreamUpstreamResponse_Timeout
// 验证：上游超时时返回已收集的 usage
func TestStreamUpstreamResponse_Timeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{StreamDataIntervalTimeout: 1, MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Body: pr, Header: http.Header{}}

	result := svc.streamUpstreamResponse(c, resp, time.Now())
	_ = pw.Close()
	_ = pr.Close()

	require.NotNil(t, result)
	require.False(t, result.clientDisconnect)
}

// TestStreamUpstreamResponse_TimeoutAfterClientDisconnect
// 验证：客户端断开后上游超时，返回 usage 并标记 clientDisconnect
func TestStreamUpstreamResponse_TimeoutAfterClientDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{StreamDataIntervalTimeout: 1, MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Writer = &antigravityFailingWriter{ResponseWriter: c.Writer, failAfter: 0}

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Body: pr, Header: http.Header{}}

	go func() {
		fmt.Fprintln(pw, `data: {"type":"message_start","message":{"usage":{"input_tokens":5}}}`)
		fmt.Fprintln(pw, "")
		// 不关闭 pw → 等待超时
	}()

	result := svc.streamUpstreamResponse(c, resp, time.Now())
	_ = pw.Close()
	_ = pr.Close()

	require.NotNil(t, result)
	require.True(t, result.clientDisconnect)
}

// TestHandleGeminiStreamingResponse_ClientDisconnect
// 验证：Gemini 流式转发中客户端断开后继续 drain 上游
func TestHandleGeminiStreamingResponse_ClientDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Writer = &antigravityFailingWriter{ResponseWriter: c.Writer, failAfter: 0}

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Body: pr, Header: http.Header{}}

	go func() {
		defer func() { _ = pw.Close() }()
		fmt.Fprintln(pw, `data: {"candidates":[{"content":{"parts":[{"text":"hi"}]}}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":10}}`)
		fmt.Fprintln(pw, "")
	}()

	result, err := svc.handleGeminiStreamingResponse(c, resp, time.Now())
	_ = pr.Close()

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.clientDisconnect)
	require.NotContains(t, rec.Body.String(), "write_failed")
}

// TestHandleGeminiStreamingResponse_ContextCanceled
// 验证：context 取消时不注入错误事件
func TestHandleGeminiStreamingResponse_ContextCanceled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)

	resp := &http.Response{StatusCode: http.StatusOK, Body: cancelReadCloser{}, Header: http.Header{}}

	result, err := svc.handleGeminiStreamingResponse(c, resp, time.Now())

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.clientDisconnect)
	require.NotContains(t, rec.Body.String(), "event: error")
}

// TestHandleClaudeStreamingResponse_ClientDisconnect
// 验证：Claude 流式转发中客户端断开后继续 drain 上游
func TestHandleClaudeStreamingResponse_ClientDisconnect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Writer = &antigravityFailingWriter{ResponseWriter: c.Writer, failAfter: 0}

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Body: pr, Header: http.Header{}}

	go func() {
		defer func() { _ = pw.Close() }()
		// v1internal 包装格式
		fmt.Fprintln(pw, `data: {"response":{"candidates":[{"content":{"parts":[{"text":"hello"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":8,"candidatesTokenCount":15}}}`)
		fmt.Fprintln(pw, "")
	}()

	result, err := svc.handleClaudeStreamingResponse(c, resp, time.Now(), "claude-sonnet-4-5")
	_ = pr.Close()

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.clientDisconnect)
}

// TestHandleClaudeStreamingResponse_EmptyStream
// 验证：上游只返回无法解析的 SSE 行时，触发 UpstreamFailoverError 而不是向客户端发出残缺流
func TestHandleClaudeStreamingResponse_EmptyStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Body: pr, Header: http.Header{}}

	go func() {
		defer func() { _ = pw.Close() }()
		// 所有行均为无法 JSON 解析的内容，ProcessLine 全部返回 nil
		fmt.Fprintln(pw, "data: not-valid-json")
		fmt.Fprintln(pw, "")
		fmt.Fprintln(pw, "data: also-invalid")
		fmt.Fprintln(pw, "")
	}()

	_, err := svc.handleClaudeStreamingResponse(c, resp, time.Now(), "claude-sonnet-4-5")
	_ = pr.Close()

	// 应当返回 UpstreamFailoverError 而非 nil，以便上层触发 failover
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.True(t, failoverErr.RetryableOnSameAccount)

	// 客户端不应收到任何 SSE 事件（既无 message_start 也无 message_stop）
	body := rec.Body.String()
	require.NotContains(t, body, "event: message_start")
	require.NotContains(t, body, "event: message_stop")
	require.NotContains(t, body, "event: message_delta")
}

// TestHandleClaudeStreamingResponse_ContextCanceled
// 验证：context 取消时不注入错误事件
func TestHandleClaudeStreamingResponse_ContextCanceled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newAntigravityTestService(&config.Config{
		Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)

	resp := &http.Response{StatusCode: http.StatusOK, Body: cancelReadCloser{}, Header: http.Header{}}

	result, err := svc.handleClaudeStreamingResponse(c, resp, time.Now(), "claude-sonnet-4-5")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.clientDisconnect)
	require.NotContains(t, rec.Body.String(), "event: error")
}

// TestExtractSSEUsage 验证 extractSSEUsage 从 SSE data 行正确提取 usage
func TestExtractSSEUsage(t *testing.T) {
	svc := &AntigravityGatewayService{}
	tests := []struct {
		name     string
		line     string
		expected ClaudeUsage
	}{
		{
			name:     "message_delta with output_tokens",
			line:     `data: {"type":"message_delta","usage":{"output_tokens":42}}`,
			expected: ClaudeUsage{OutputTokens: 42},
		},
		{
			name:     "non-data line ignored",
			line:     `event: message_start`,
			expected: ClaudeUsage{},
		},
		{
			name:     "top-level usage with all fields",
			line:     `data: {"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":5,"cache_creation_input_tokens":3}}`,
			expected: ClaudeUsage{InputTokens: 10, OutputTokens: 20, CacheReadInputTokens: 5, CacheCreationInputTokens: 3},
		},
		{
			// Anthropic message_start 把 usage 嵌套在 message.usage 下，
			// 必须从这里提取输入侧字段（含 cache_read/cache_creation_input_tokens）。
			name:     "message_start nested usage with input/cache tokens",
			line:     `data: {"type":"message_start","message":{"id":"msg_01","usage":{"input_tokens":35576,"cache_creation_input_tokens":0,"cache_read_input_tokens":12000,"output_tokens":1}}}`,
			expected: ClaudeUsage{InputTokens: 35576, OutputTokens: 1, CacheReadInputTokens: 12000},
		},
		{
			// message_start.message.usage.cache_creation 内的 5m/1h 明细也要解析。
			name:     "message_start nested usage with cache_creation breakdown",
			line:     `data: {"type":"message_start","message":{"usage":{"input_tokens":100,"cache_creation":{"ephemeral_5m_input_tokens":30,"ephemeral_1h_input_tokens":70}}}}`,
			expected: ClaudeUsage{InputTokens: 100, CacheCreation5mTokens: 30, CacheCreation1hTokens: 70},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := &ClaudeUsage{}
			svc.extractSSEUsage(tt.line, usage)
			require.Equal(t, tt.expected, *usage)
		})
	}
}

// TestExtractSSEUsage_StreamingSequence 复现 issue #2332：完整的 Anthropic streaming
// 序列（message_start → message_delta）必须把两类事件中的 usage 字段都汇入同一份累计值，
// 否则透传账号产出的 usage_logs 会出现 input_tokens=0、仅有 output_tokens 的"残缺"记录。
func TestExtractSSEUsage_StreamingSequence(t *testing.T) {
	svc := &AntigravityGatewayService{}
	usage := &ClaudeUsage{}

	// 1) message_start：携带完整输入侧 usage（input_tokens + cache_read）
	svc.extractSSEUsage(
		`data: {"type":"message_start","message":{"id":"msg_01","type":"message","role":"assistant","content":[],"model":"claude-opus-4-6","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":35576,"cache_creation_input_tokens":0,"cache_read_input_tokens":12000,"output_tokens":1}}}`,
		usage,
	)
	// 2) message_delta：流结束时只带 output_tokens（无 input_tokens 字段）
	svc.extractSSEUsage(
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":816}}`,
		usage,
	)

	require.Equal(t, 35576, usage.InputTokens, "message_start 的 input_tokens 必须被记录，否则记账会缺失输入侧 token (#2332)")
	require.Equal(t, 12000, usage.CacheReadInputTokens, "message_start 的 cache_read_input_tokens 必须被记录")
	require.Equal(t, 816, usage.OutputTokens, "message_delta 的最终 output_tokens 必须被记录")
}

// TestAntigravityClientWriter 验证 antigravityClientWriter 的断开检测
func TestAntigravityClientWriter(t *testing.T) {
	t.Run("normal write succeeds", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		flusher, _ := c.Writer.(http.Flusher)
		cw := newAntigravityClientWriter(c.Writer, flusher, "test")

		ok := cw.Write([]byte("hello"))
		require.True(t, ok)
		require.False(t, cw.Disconnected())
		require.Contains(t, rec.Body.String(), "hello")
	})

	t.Run("write failure marks disconnected", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		fw := &antigravityFailingWriter{ResponseWriter: c.Writer, failAfter: 0}
		flusher, _ := c.Writer.(http.Flusher)
		cw := newAntigravityClientWriter(fw, flusher, "test")

		ok := cw.Write([]byte("hello"))
		require.False(t, ok)
		require.True(t, cw.Disconnected())
	})

	t.Run("subsequent writes are no-op", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		fw := &antigravityFailingWriter{ResponseWriter: c.Writer, failAfter: 0}
		flusher, _ := c.Writer.(http.Flusher)
		cw := newAntigravityClientWriter(fw, flusher, "test")

		cw.Write([]byte("first"))
		ok := cw.Fprintf("second %d", 2)
		require.False(t, ok)
		require.True(t, cw.Disconnected())
	})
}

// TestUnwrapV1InternalResponse 测试 unwrapV1InternalResponse 的各种输入场景
func TestUnwrapV1InternalResponse(t *testing.T) {
	svc := &AntigravityGatewayService{}

	// 构造 >50KB 的大型 JSON
	largePadding := strings.Repeat("x", 50*1024)
	largeInput := []byte(fmt.Sprintf(`{"response":{"id":"big","pad":"%s"}}`, largePadding))
	largeExpected := fmt.Sprintf(`{"id":"big","pad":"%s"}`, largePadding)

	tests := []struct {
		name     string
		input    []byte
		expected string
		wantErr  bool
	}{
		{
			name:     "正常 response 包装",
			input:    []byte(`{"response":{"id":"123","content":"hello"}}`),
			expected: `{"id":"123","content":"hello"}`,
		},
		{
			name:     "无 response 透传",
			input:    []byte(`{"id":"456"}`),
			expected: `{"id":"456"}`,
		},
		{
			name:     "空 JSON",
			input:    []byte(`{}`),
			expected: `{}`,
		},
		{
			name:     "response 为 null",
			input:    []byte(`{"response":null}`),
			expected: `null`,
		},
		{
			name:     "response 为基础类型 string",
			input:    []byte(`{"response":"hello"}`),
			expected: `"hello"`,
		},
		{
			name:     "非法 JSON",
			input:    []byte(`not json`),
			expected: `not json`,
		},
		{
			name:     "嵌套 response 只解一层",
			input:    []byte(`{"response":{"response":{"inner":true}}}`),
			expected: `{"response":{"inner":true}}`,
		},
		{
			name:     "大型 JSON >50KB",
			input:    largeInput,
			expected: largeExpected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.unwrapV1InternalResponse(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, strings.TrimSpace(string(got)))
		})
	}
}

// --- unwrapV1InternalResponse benchmark 对照组 ---

// unwrapV1InternalResponseOld 旧实现：Unmarshal+Marshal 双重开销（仅用于 benchmark 对照）
func unwrapV1InternalResponseOld(body []byte) ([]byte, error) {
	var outer map[string]any
	if err := json.Unmarshal(body, &outer); err != nil {
		return nil, err
	}
	if resp, ok := outer["response"]; ok {
		return json.Marshal(resp)
	}
	return body, nil
}

func BenchmarkUnwrapV1Internal_Old_Small(b *testing.B) {
	body := []byte(`{"response":{"candidates":[{"content":{"parts":[{"text":"hello world"}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5}}}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = unwrapV1InternalResponseOld(body)
	}
}

func BenchmarkUnwrapV1Internal_New_Small(b *testing.B) {
	body := []byte(`{"response":{"candidates":[{"content":{"parts":[{"text":"hello world"}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5}}}`)
	svc := &AntigravityGatewayService{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.unwrapV1InternalResponse(body)
	}
}

func BenchmarkUnwrapV1Internal_Old_Large(b *testing.B) {
	body := generateLargeUnwrapJSON(10 * 1024) // ~10KB
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = unwrapV1InternalResponseOld(body)
	}
}

func BenchmarkUnwrapV1Internal_New_Large(b *testing.B) {
	body := generateLargeUnwrapJSON(10 * 1024) // ~10KB
	svc := &AntigravityGatewayService{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.unwrapV1InternalResponse(body)
	}
}

// generateLargeUnwrapJSON 生成指定最小大小的包含 response 包装的 JSON
func generateLargeUnwrapJSON(minSize int) []byte {
	parts := make([]map[string]string, 0)
	current := 0
	for current < minSize {
		text := fmt.Sprintf("这是第 %d 段内容，用于填充 JSON 到目标大小。", len(parts)+1)
		parts = append(parts, map[string]string{"text": text})
		current += len(text) + 20 // 估算 JSON 编码开销
	}
	inner := map[string]any{
		"candidates": []map[string]any{
			{"content": map[string]any{"parts": parts}},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     100,
			"candidatesTokenCount": 50,
		},
	}
	outer := map[string]any{"response": inner}
	b, _ := json.Marshal(outer)
	return b
}
