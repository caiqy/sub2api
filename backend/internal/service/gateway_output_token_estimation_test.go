package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// newGatewayServiceForEstimationTest creates a minimal GatewayService for output token estimation tests.
func newGatewayServiceForEstimationTest() *GatewayService {
	return &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				StreamDataIntervalTimeout: 0,
				MaxLineSize:               defaultMaxLineSize,
			},
		},
		rateLimitService: &RateLimitService{},
	}
}

// =====================================================================
// applyOutputTokenEstimation unit tests
// =====================================================================

func TestApplyOutputTokenEstimation_NilUsage(t *testing.T) {
	// nil usage should not panic
	applyOutputTokenEstimation(nil, "some text")
}

func TestApplyOutputTokenEstimation_AlreadyHasOutputTokens(t *testing.T) {
	usage := &ClaudeUsage{OutputTokens: 42}
	applyOutputTokenEstimation(usage, "hello world this is a test")
	require.Equal(t, 42, usage.OutputTokens, "should not overwrite existing non-zero output_tokens")
}

func TestApplyOutputTokenEstimation_EmptyText(t *testing.T) {
	usage := &ClaudeUsage{OutputTokens: 0}
	applyOutputTokenEstimation(usage, "")
	require.Equal(t, 0, usage.OutputTokens, "empty text should not set output_tokens")
}

func TestApplyOutputTokenEstimation_ZeroOutputWithText(t *testing.T) {
	usage := &ClaudeUsage{OutputTokens: 0, InputTokens: 100}
	applyOutputTokenEstimation(usage, "Hello world! This is a test of the token estimation fallback.")
	require.Greater(t, usage.OutputTokens, 0, "should estimate output_tokens when upstream reports 0")
	require.Equal(t, 100, usage.InputTokens, "should not modify input_tokens")
}

// =====================================================================
// extractContentTextFromResponseBody unit tests
// =====================================================================

func TestExtractContentTextFromResponseBody_TextBlock(t *testing.T) {
	body := []byte(`{"content":[{"type":"text","text":"Hello, world!"}],"usage":{"output_tokens":0}}`)
	got := extractContentTextFromResponseBody(body)
	require.Equal(t, "Hello, world!", got)
}

func TestExtractContentTextFromResponseBody_ToolUseBlock(t *testing.T) {
	body := []byte(`{"content":[{"type":"tool_use","id":"t1","name":"get_weather","input":{"city":"Beijing"}}],"usage":{"output_tokens":0}}`)
	got := extractContentTextFromResponseBody(body)
	require.Contains(t, got, "Beijing")
}

func TestExtractContentTextFromResponseBody_ThinkingBlock(t *testing.T) {
	body := []byte(`{"content":[{"type":"thinking","thinking":"Let me think about this..."}],"usage":{"output_tokens":0}}`)
	got := extractContentTextFromResponseBody(body)
	require.Equal(t, "Let me think about this...", got)
}

func TestExtractContentTextFromResponseBody_MixedBlocks(t *testing.T) {
	body := []byte(`{"content":[
		{"type":"thinking","thinking":"First I think..."},
		{"type":"text","text":"Here is the answer."},
		{"type":"tool_use","id":"t1","name":"calc","input":{"expr":"1+1"}}
	],"usage":{"output_tokens":0}}`)
	got := extractContentTextFromResponseBody(body)
	require.Contains(t, got, "First I think...")
	require.Contains(t, got, "Here is the answer.")
	require.Contains(t, got, "1+1")
}

func TestExtractContentTextFromResponseBody_NoContent(t *testing.T) {
	body := []byte(`{"usage":{"output_tokens":0}}`)
	got := extractContentTextFromResponseBody(body)
	require.Equal(t, "", got)
}

func TestExtractContentTextFromResponseBody_EmptyContent(t *testing.T) {
	body := []byte(`{"content":[],"usage":{"output_tokens":0}}`)
	got := extractContentTextFromResponseBody(body)
	require.Equal(t, "", got)
}

// =====================================================================
// Standard streaming path (handleStreamingResponse) — output_tokens fallback
// =====================================================================

func TestHandleStreamingResponse_OutputTokensFallback_ZeroUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newGatewayServiceForEstimationTest()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: pr}

	go func() {
		defer func() { _ = pw.Close() }()
		// message_start with input_tokens
		_, _ = pw.Write([]byte("data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":50}}}\n\n"))
		// content_block_delta with actual text output
		_, _ = pw.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello, world! This is a test response with some content.\"}}\n\n"))
		_, _ = pw.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" More text here to make the estimation more interesting.\"}}\n\n"))
		// message_delta with output_tokens=0 (the bug we're fixing)
		_, _ = pw.Write([]byte("data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":0}}\n\n"))
		_, _ = pw.Write([]byte("data: [DONE]\n\n"))
	}()

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
	_ = pr.Close()
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.Equal(t, 50, result.usage.InputTokens)
	require.Greater(t, result.usage.OutputTokens, 0, "output_tokens should be estimated when upstream reports 0")
}

func TestHandleStreamingResponse_OutputTokensFallback_NonZeroUpstreamNotOverwritten(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newGatewayServiceForEstimationTest()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: pr}

	go func() {
		defer func() { _ = pw.Close() }()
		_, _ = pw.Write([]byte("data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":50}}}\n\n"))
		_, _ = pw.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello world\"}}\n\n"))
		// message_delta with actual output_tokens from a well-behaved upstream
		_, _ = pw.Write([]byte("data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":99}}\n\n"))
		_, _ = pw.Write([]byte("data: [DONE]\n\n"))
	}()

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
	_ = pr.Close()
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.Equal(t, 99, result.usage.OutputTokens, "should preserve upstream output_tokens when non-zero")
}

func TestHandleStreamingResponse_OutputTokensFallback_ThinkingDelta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newGatewayServiceForEstimationTest()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: pr}

	go func() {
		defer func() { _ = pw.Close() }()
		_, _ = pw.Write([]byte("data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":10}}}\n\n"))
		// thinking delta
		_, _ = pw.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"Let me analyze this problem step by step.\"}}\n\n"))
		// text delta
		_, _ = pw.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":1,\"delta\":{\"type\":\"text_delta\",\"text\":\"The answer is 42.\"}}\n\n"))
		_, _ = pw.Write([]byte("data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":0}}\n\n"))
		_, _ = pw.Write([]byte("data: [DONE]\n\n"))
	}()

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
	_ = pr.Close()
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.Greater(t, result.usage.OutputTokens, 0, "should estimate from thinking+text content")
}

func TestHandleStreamingResponse_OutputTokensFallback_PartialJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newGatewayServiceForEstimationTest()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: pr}

	go func() {
		defer func() { _ = pw.Close() }()
		_, _ = pw.Write([]byte("data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":10}}}\n\n"))
		// input_json_delta (tool use streaming)
		_, _ = pw.Write([]byte("data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"query\\\": \\\"weather in Beijing\\\"\"}}\n\n"))
		_, _ = pw.Write([]byte("data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":0}}\n\n"))
		_, _ = pw.Write([]byte("data: [DONE]\n\n"))
	}()

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
	_ = pr.Close()
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.Greater(t, result.usage.OutputTokens, 0, "should estimate from partial_json content")
}

func TestHandleStreamingResponse_OutputTokensFallback_NoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newGatewayServiceForEstimationTest()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	pr, pw := io.Pipe()
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: pr}

	go func() {
		defer func() { _ = pw.Close() }()
		_, _ = pw.Write([]byte("data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":10}}}\n\n"))
		// No content_block_delta at all, output_tokens=0
		_, _ = pw.Write([]byte("data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":0}}\n\n"))
		_, _ = pw.Write([]byte("data: [DONE]\n\n"))
	}()

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
	_ = pr.Close()
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.Equal(t, 0, result.usage.OutputTokens, "should remain 0 when no content was streamed")
}

// =====================================================================
// Passthrough streaming path — output_tokens fallback
// =====================================================================

func TestHandleStreamingResponsePassthrough_OutputTokensFallback_ZeroUpstream(t *testing.T) {
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
			`data: {"type":"message_start","message":{"usage":{"input_tokens":30}}}`,
			"",
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"This is a test response from the upstream proxy."}}`,
			"",
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" And here is some more text to make it interesting."}}`,
			"",
			`data: {"type":"message_delta","usage":{"output_tokens":0}}`,
			"",
			"data: [DONE]",
			"",
		}, "\n"))),
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "claude-3-7-sonnet-20250219")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.Equal(t, 30, result.usage.InputTokens)
	require.Greater(t, result.usage.OutputTokens, 0, "should estimate output_tokens when upstream reports 0")
}

func TestHandleStreamingResponsePassthrough_OutputTokensFallback_NonZeroPreserved(t *testing.T) {
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
			`data: {"type":"message_start","message":{"usage":{"input_tokens":30}}}`,
			"",
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Some text"}}`,
			"",
			`data: {"type":"message_delta","usage":{"output_tokens":77}}`,
			"",
			"data: [DONE]",
			"",
		}, "\n"))),
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "claude-3-7-sonnet-20250219")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.Equal(t, 77, result.usage.OutputTokens, "should preserve upstream output_tokens when non-zero")
}

func TestHandleStreamingResponsePassthrough_OutputTokensFallback_ThinkingDelta(t *testing.T) {
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
			`data: {"type":"message_start","message":{"usage":{"input_tokens":10}}}`,
			"",
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"I need to analyze this carefully."}}`,
			"",
			`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"The answer is 42."}}`,
			"",
			`data: {"type":"message_delta","usage":{"output_tokens":0}}`,
			"",
			"data: [DONE]",
			"",
		}, "\n"))),
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "claude-3-7-sonnet-20250219")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.Greater(t, result.usage.OutputTokens, 0, "should estimate from thinking+text content")
}

// =====================================================================
// Non-streaming path (handleNonStreamingResponse) — output_tokens fallback
// =====================================================================

func TestHandleNonStreamingResponse_OutputTokensFallback_ZeroUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newGatewayServiceForEstimationTest()

	upstreamJSON := `{"id":"msg_1","type":"message","content":[{"type":"text","text":"Hello! This is a response from an upstream proxy that does not report output tokens."}],"usage":{"input_tokens":20,"output_tokens":0}}`

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(upstreamJSON)),
	}

	usage, err := svc.handleNonStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, "model", "model")
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 20, usage.InputTokens)
	require.Greater(t, usage.OutputTokens, 0, "should estimate output_tokens from content text")
}

func TestHandleNonStreamingResponse_OutputTokensFallback_NonZeroPreserved(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newGatewayServiceForEstimationTest()

	upstreamJSON := `{"id":"msg_1","type":"message","content":[{"type":"text","text":"Hello world."}],"usage":{"input_tokens":20,"output_tokens":55}}`

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(upstreamJSON)),
	}

	usage, err := svc.handleNonStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, "model", "model")
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 55, usage.OutputTokens, "should preserve upstream output_tokens when non-zero")
}

func TestHandleNonStreamingResponse_OutputTokensFallback_ToolUse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newGatewayServiceForEstimationTest()

	upstreamJSON := `{"id":"msg_1","type":"message","content":[{"type":"tool_use","id":"t1","name":"get_weather","input":{"city":"Beijing","units":"celsius"}}],"usage":{"input_tokens":15,"output_tokens":0}}`

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(upstreamJSON)),
	}

	usage, err := svc.handleNonStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, "model", "model")
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Greater(t, usage.OutputTokens, 0, "should estimate output_tokens from tool_use input")
}

func TestHandleNonStreamingResponse_OutputTokensFallback_EmptyContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newGatewayServiceForEstimationTest()

	upstreamJSON := `{"id":"msg_1","type":"message","content":[],"usage":{"input_tokens":20,"output_tokens":0}}`

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(upstreamJSON)),
	}

	usage, err := svc.handleNonStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, "model", "model")
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 0, usage.OutputTokens, "should remain 0 when content array is empty")
}

// =====================================================================
// Passthrough non-streaming path — output_tokens fallback
// =====================================================================

func TestHandleNonStreamingResponsePassthrough_OutputTokensFallback_ZeroUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamJSON := `{"id":"msg_1","type":"message","content":[{"type":"text","text":"Hello from passthrough non-streaming with zero output tokens!"}],"usage":{"input_tokens":15,"output_tokens":0}}`

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := &GatewayService{
		cfg:              &config.Config{},
		rateLimitService: &RateLimitService{},
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(upstreamJSON)),
	}

	usage, err := svc.handleNonStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 1})
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 15, usage.InputTokens)
	require.Greater(t, usage.OutputTokens, 0, "should estimate output_tokens when upstream reports 0")
}

func TestHandleNonStreamingResponsePassthrough_OutputTokensFallback_NonZeroPreserved(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamJSON := `{"id":"msg_1","type":"message","content":[{"type":"text","text":"some text"}],"usage":{"input_tokens":15,"output_tokens":33}}`

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := &GatewayService{
		cfg:              &config.Config{},
		rateLimitService: &RateLimitService{},
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(upstreamJSON)),
	}

	usage, err := svc.handleNonStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 1})
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Equal(t, 33, usage.OutputTokens, "should preserve upstream output_tokens when non-zero")
}
