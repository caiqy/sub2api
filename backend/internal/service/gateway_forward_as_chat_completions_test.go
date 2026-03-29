//go:build unit

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
	"github.com/tidwall/gjson"
)

func TestExtractCCReasoningEffortFromBody(t *testing.T) {
	t.Parallel()

	t.Run("nested reasoning.effort", func(t *testing.T) {
		got := extractCCReasoningEffortFromBody([]byte(`{"reasoning":{"effort":"HIGH"}}`))
		require.NotNil(t, got)
		require.Equal(t, "high", *got)
	})

	t.Run("flat reasoning_effort", func(t *testing.T) {
		got := extractCCReasoningEffortFromBody([]byte(`{"reasoning_effort":"x-high"}`))
		require.NotNil(t, got)
		require.Equal(t, "xhigh", *got)
	})

	t.Run("missing effort", func(t *testing.T) {
		require.Nil(t, extractCCReasoningEffortFromBody([]byte(`{"model":"gpt-5"}`)))
	})
}

func TestHandleCCBufferedFromAnthropic_PreservesMessageStartCacheUsageAndReasoning(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	reasoningEffort := "high"
	resp := &http.Response{
		Header: http.Header{"x-request-id": []string{"rid_cc_buffered"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`event: message_start`,
			`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4.5","stop_reason":"","usage":{"input_tokens":12,"cache_read_input_tokens":9,"cache_creation_input_tokens":3}}}`,
			``,
			`event: content_block_start`,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":"hello"}}`,
			``,
			`event: message_delta`,
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":7}}`,
			``,
		}, "\n"))),
	}

	svc := &GatewayService{}
	result, err := svc.handleCCBufferedFromAnthropic(resp, c, "gpt-5", "claude-sonnet-4.5", &reasoningEffort, time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 12, result.Usage.InputTokens)
	require.Equal(t, 7, result.Usage.OutputTokens)
	require.Equal(t, 9, result.Usage.CacheReadInputTokens)
	require.Equal(t, 3, result.Usage.CacheCreationInputTokens)
	require.NotNil(t, result.ReasoningEffort)
	require.Equal(t, "high", *result.ReasoningEffort)
}

func TestHandleCCStreamingFromAnthropic_PreservesMessageStartCacheUsageAndReasoning(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	reasoningEffort := "medium"
	resp := &http.Response{
		Header: http.Header{"x-request-id": []string{"rid_cc_stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`event: message_start`,
			`data: {"type":"message_start","message":{"id":"msg_2","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4.5","stop_reason":"","usage":{"input_tokens":20,"cache_read_input_tokens":11,"cache_creation_input_tokens":4}}}`,
			``,
			`event: content_block_start`,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":"hello"}}`,
			``,
			`event: message_delta`,
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":8}}`,
			``,
			`event: message_stop`,
			`data: {"type":"message_stop"}`,
			``,
		}, "\n"))),
	}

	svc := &GatewayService{}
	result, err := svc.handleCCStreamingFromAnthropic(resp, c, "gpt-5", "claude-sonnet-4.5", &reasoningEffort, time.Now(), true)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 20, result.Usage.InputTokens)
	require.Equal(t, 8, result.Usage.OutputTokens)
	require.Equal(t, 11, result.Usage.CacheReadInputTokens)
	require.Equal(t, 4, result.Usage.CacheCreationInputTokens)
	require.NotNil(t, result.ReasoningEffort)
	require.Equal(t, "medium", *result.ReasoningEffort)
	require.Contains(t, rec.Body.String(), `[DONE]`)
}

// ---------------------------------------------------------------------------
// ForwardAsChatCompletions passthrough field regression tests
// ---------------------------------------------------------------------------
// These tests verify that account passthrough field rules correctly read values
// from the original Chat Completions request body (sourceBody) rather than the
// converted Anthropic body. This is the regression that would reappear if the
// buildUpstreamRequestWithSourceBody call were reverted to buildUpstreamRequest.
// ---------------------------------------------------------------------------

// TestGatewayService_ForwardAsChatCompletions_PassthroughBodyMapCopiesFromOriginalCCBody
// verifies that a body "map" rule reads from the original Chat Completions body
// (which contains "user" as the key for the sender) rather than the converted
// Anthropic body (which also has a "messages" array but with a different structure).
//
// The Chat Completions body contains a top-level "user" field (an OpenAI-specific field).
// The test configures a map rule: source_key="user" → key="metadata.end_user".
// If the fix is correct, the upstream Anthropic body will contain
// metadata.end_user="cc-user-42" (copied from the original CC body's "user" field).
// If reverted, the map would read from the Anthropic body where "user" does not exist
// at top level, so the field would NOT be injected.
func TestGatewayService_ForwardAsChatCompletions_PassthroughBodyMapCopiesFromOriginalCCBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	// Original Chat Completions body with an OpenAI-specific "user" field.
	ccBody := []byte(`{
		"model": "claude-sonnet-4-20250514",
		"messages": [{"role": "user", "content": "hello"}],
		"user": "cc-user-42"
	}`)

	parsed := &ParsedRequest{Body: ccBody, Model: "claude-sonnet-4-20250514", Stream: false}

	upstreamSSE := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_cc","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":"","usage":{"input_tokens":5}}}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":"hi"}}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}`,
		``,
	}, "\n")

	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"x-request-id": []string{"rid-cc-passthrough"},
			},
			Body: io.NopCloser(strings.NewReader(upstreamSSE)),
		},
	}

	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
			Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
		},
		httpUpstream:        upstream,
		rateLimitService:    &RateLimitService{},
		tlsFPProfileService: &TLSFingerprintProfileService{},
	}

	account := &Account{
		ID:          701,
		Name:        "cc-passthrough-map-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "upstream-key-cc",
			"base_url": "https://api.anthropic.com",
		},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				// Map: copy "user" from original CC body → "metadata.end_user" in upstream body
				{Target: "body", Mode: "map", Key: "metadata.end_user", SourceKey: "user"},
				// Inject: fixed header
				{Target: "header", Mode: "inject", Key: "X-CC-Tag", Value: "cc-passthrough"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, ccBody, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)

	// The upstream body is Anthropic format, but should have the mapped field injected.
	require.NotNil(t, upstream.lastBody, "upstream request should have been sent")

	// Core assertion: the "user" value was read from the ORIGINAL CC body, not the Anthropic body.
	mappedValue := gjson.GetBytes(upstream.lastBody, "metadata.end_user").String()
	require.Equal(t, "cc-user-42", mappedValue,
		"body map rule should read 'user' from the original Chat Completions body, not the converted Anthropic body")

	// Header injection should also work.
	require.Equal(t, "cc-passthrough", getHeaderRaw(upstream.lastReq.Header, "X-CC-Tag"),
		"header inject rule should apply in ForwardAsChatCompletions path")

	// Auth header should be replaced.
	require.Equal(t, "upstream-key-cc", getHeaderRaw(upstream.lastReq.Header, "x-api-key"))
}

// TestGatewayService_ForwardAsChatCompletions_PassthroughDisabledLeavesRulesInactive
// verifies that passthrough field rules are not applied when passthrough_fields_enabled=false.
func TestGatewayService_ForwardAsChatCompletions_PassthroughDisabledLeavesRulesInactive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	ccBody := []byte(`{
		"model": "claude-sonnet-4-20250514",
		"messages": [{"role": "user", "content": "hello"}],
		"user": "should-not-appear"
	}`)

	parsed := &ParsedRequest{Body: ccBody, Model: "claude-sonnet-4-20250514", Stream: false}

	upstreamSSE := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_dis","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":"","usage":{"input_tokens":5}}}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":"hi"}}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}`,
		``,
	}, "\n")

	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"x-request-id": []string{"rid-cc-disabled"},
			},
			Body: io.NopCloser(strings.NewReader(upstreamSSE)),
		},
	}

	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
			Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
		},
		httpUpstream:        upstream,
		rateLimitService:    &RateLimitService{},
		tlsFPProfileService: &TLSFingerprintProfileService{},
	}

	account := &Account{
		ID:          702,
		Name:        "cc-passthrough-disabled-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "upstream-key-cc-disabled",
			"base_url": "https://api.anthropic.com",
		},
		Extra: map[string]any{
			"passthrough_fields_enabled": false, // disabled
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "map", Key: "metadata.end_user", SourceKey: "user"},
				{Target: "header", Mode: "inject", Key: "X-CC-Tag", Value: "should-not-appear"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, ccBody, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastBody)

	// Body map rule should NOT have been applied.
	require.False(t, gjson.GetBytes(upstream.lastBody, "metadata.end_user").Exists(),
		"body map field should not be injected when passthrough is disabled")

	// Header inject rule should NOT have been applied.
	require.Empty(t, getHeaderRaw(upstream.lastReq.Header, "X-CC-Tag"),
		"header inject should not apply when passthrough is disabled")
}

// TestGatewayService_ForwardAsChatCompletions_PassthroughBodyForwardCopiesFromOriginalCCBody
// verifies that a body "forward" rule reads the value from the original Chat Completions
// body (sourceBody) rather than the converted Anthropic body.
//
// "forward" mode copies body[key] from sourceBody → targetBody at the same key.
// The original CC body has a top-level "user" field; the converted Anthropic body does not.
// If the fix is correct, the upstream Anthropic body will have "user" injected.
// If reverted, "user" would be looked up in the Anthropic body where it doesn't exist,
// so the field would NOT appear.
func TestGatewayService_ForwardAsChatCompletions_PassthroughBodyForwardCopiesFromOriginalCCBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	ccBody := []byte(`{
		"model": "claude-sonnet-4-20250514",
		"messages": [{"role": "user", "content": "hello"}],
		"user": "cc-forward-user-99"
	}`)

	parsed := &ParsedRequest{Body: ccBody, Model: "claude-sonnet-4-20250514", Stream: false}

	upstreamSSE := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_ccfwd","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":"","usage":{"input_tokens":5}}}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":"hi"}}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}`,
		``,
	}, "\n")

	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"x-request-id": []string{"rid-cc-body-forward"},
			},
			Body: io.NopCloser(strings.NewReader(upstreamSSE)),
		},
	}

	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
			Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
		},
		httpUpstream:        upstream,
		rateLimitService:    &RateLimitService{},
		tlsFPProfileService: &TLSFingerprintProfileService{},
	}

	account := &Account{
		ID:          703,
		Name:        "cc-passthrough-body-forward-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "upstream-key-cc-fwd",
			"base_url": "https://api.anthropic.com",
		},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				// Forward: copy "user" from original CC body → same key in upstream body
				{Target: "body", Mode: "forward", Key: "user"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, ccBody, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastBody)

	// Core assertion: "user" was read from the ORIGINAL CC body via forward mode.
	forwardedValue := gjson.GetBytes(upstream.lastBody, "user").String()
	require.Equal(t, "cc-forward-user-99", forwardedValue,
		"body forward rule should read 'user' from the original CC body, not the converted Anthropic body")
}

// TestGatewayService_ForwardAsChatCompletions_PassthroughHeaderForwardCopiesFromClientRequest
// verifies that a header "forward" rule copies a header from the client request to the
// upstream request in the ForwardAsChatCompletions path.
func TestGatewayService_ForwardAsChatCompletions_PassthroughHeaderForwardCopiesFromClientRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Request.Header.Set("X-Custom-Trace", "cc-trace-abc")

	ccBody := []byte(`{
		"model": "claude-sonnet-4-20250514",
		"messages": [{"role": "user", "content": "hello"}]
	}`)

	parsed := &ParsedRequest{Body: ccBody, Model: "claude-sonnet-4-20250514", Stream: false}

	upstreamSSE := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_cchdr","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":"","usage":{"input_tokens":5}}}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":"hi"}}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}`,
		``,
	}, "\n")

	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"x-request-id": []string{"rid-cc-header-forward"},
			},
			Body: io.NopCloser(strings.NewReader(upstreamSSE)),
		},
	}

	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
			Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
		},
		httpUpstream:        upstream,
		rateLimitService:    &RateLimitService{},
		tlsFPProfileService: &TLSFingerprintProfileService{},
	}

	account := &Account{
		ID:          704,
		Name:        "cc-passthrough-header-forward-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "upstream-key-cc-hdr",
			"base_url": "https://api.anthropic.com",
		},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				// Forward: copy "X-Custom-Trace" from client request header → upstream header
				{Target: "header", Mode: "forward", Key: "X-Custom-Trace"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, ccBody, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Core assertion: header was forwarded from client request.
	require.Equal(t, "cc-trace-abc", getHeaderRaw(upstream.lastReq.Header, "X-Custom-Trace"),
		"header forward rule should copy X-Custom-Trace from client request to upstream")
}
