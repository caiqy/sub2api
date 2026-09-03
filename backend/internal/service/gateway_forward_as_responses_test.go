//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestAdaptResponsesClientToolsForAnthropic_FlattensNamespace(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"claude-fable-5",
		"input":[{"type":"function_call","call_id":"call_1","namespace":"codex_app","name":"read_thread","arguments":"{}"}],
		"tools":[{"type":"namespace","name":"codex_app","tools":[{"type":"function","name":"read_thread","description":"Read a task","parameters":{"type":"object","properties":{}}}]}]
	}`)

	adapted, mapping, err := adaptResponsesClientToolsForAnthropic(body)
	require.NoError(t, err)
	require.Equal(t, apicompat.ResponsesNamespaceName{Namespace: "codex_app", Name: "read_thread"}, mapping.NamespaceTools["codex_app__read_thread"])

	var request map[string]any
	require.NoError(t, json.Unmarshal(adapted, &request))
	tools := request["tools"].([]any)
	require.Len(t, tools, 1)
	tool := tools[0].(map[string]any)
	require.Equal(t, "function", tool["type"])
	require.Equal(t, "codex_app__read_thread", tool["name"])

	input := request["input"].([]any)
	call := input[0].(map[string]any)
	require.Equal(t, "codex_app__read_thread", call["name"])
	require.NotContains(t, call, "namespace")
}

func TestAdaptResponsesClientToolsForAnthropic_LiftsAdditionalTools(t *testing.T) {
	body := []byte(`{
		"model":"claude-fable-5",
		"input":[
			{"type":"additional_tools","tools":[
				{"type":"custom","name":"exec","description":"Run a command"},
				{"type":"namespace","name":"codex_app","tools":[
					{"type":"function","name":"read_thread","parameters":{"type":"object"}}
				]}
			]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"inspect"}]}
		]
	}`)

	adapted, mapping, err := adaptResponsesClientToolsForAnthropic(body)
	require.NoError(t, err)
	require.True(t, mapping.CustomTools["exec"])
	require.Equal(t, apicompat.ResponsesNamespaceName{Namespace: "codex_app", Name: "read_thread"}, mapping.NamespaceTools["codex_app__read_thread"])

	var request map[string]any
	require.NoError(t, json.Unmarshal(adapted, &request))
	tools := request["tools"].([]any)
	require.Len(t, tools, 2)
	require.Equal(t, "function", tools[0].(map[string]any)["type"])
	require.Equal(t, "codex_app__read_thread", tools[1].(map[string]any)["name"])

	input := request["input"].([]any)
	require.Len(t, input, 1)
	require.Equal(t, "message", input[0].(map[string]any)["type"])
}

func namespaceToolAnthropicStream() string {
	return strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_namespace","type":"message","role":"assistant","content":[],"model":"claude-fable-5","stop_reason":"","usage":{"input_tokens":10}}}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_namespace","name":"codex_app__read_thread","input":{"thread_id":"123"}}}`,
		``,
		`event: content_block_stop`,
		`data: {"type":"content_block_stop","index":0}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":5}}`,
		``,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n")
}

func namespaceToolMapping() apicompat.ResponsesClientToolMapping {
	return apicompat.ResponsesClientToolMapping{NamespaceTools: map[string]apicompat.ResponsesNamespaceName{
		"codex_app__read_thread": {Namespace: "codex_app", Name: "read_thread"},
	}}
}

func TestHandleResponsesBufferedStreamingResponse_RestoresNamespaceTool(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	resp := &http.Response{Body: io.NopCloser(strings.NewReader(namespaceToolAnthropicStream()))}

	svc := &GatewayService{}
	_, err := svc.handleResponsesBufferedStreamingResponse(resp, c, "claude-fable-5", "claude-fable-5", nil, time.Now(), namespaceToolMapping())
	require.NoError(t, err)
	require.Contains(t, rec.Body.String(), `"type":"function_call"`)
	require.Contains(t, rec.Body.String(), `"name":"read_thread"`)
	require.Contains(t, rec.Body.String(), `"namespace":"codex_app"`)
	require.NotContains(t, rec.Body.String(), `"name":"codex_app__read_thread"`)
}

func TestHandleResponsesBufferedStreamingResponse_ToolArgumentsAreValidJSON(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	resp := &http.Response{Body: io.NopCloser(strings.NewReader(toolAnthropicSSEStream()))}

	_, err := (&GatewayService{}).handleResponsesBufferedStreamingResponse(resp, c, "claude-fable-5", "claude-fable-5", nil, time.Now(), apicompat.ResponsesClientToolMapping{})
	require.NoError(t, err)

	var body struct {
		Output []struct {
			Type      string `json:"type"`
			Arguments string `json:"arguments"`
		} `json:"output"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Output, 1)
	require.Equal(t, "function_call", body.Output[0].Type)
	require.JSONEq(t, `{"query":"status"}`, body.Output[0].Arguments)
}

func TestAppendRawJSON_EmptyObjectPlaceholder(t *testing.T) {
	t.Parallel()

	fragment := `{"query":"status"}`
	require.JSONEq(t, fragment, string(appendRawJSON(json.RawMessage("{ \n\t }"), fragment)))
	require.Equal(t, `{"existing":true}{"query":"status"}`, string(appendRawJSON(json.RawMessage(`{"existing":true}`), fragment)))
}

func TestHandleResponsesStreamingResponse_RestoresNamespaceTool(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	resp := &http.Response{Body: io.NopCloser(strings.NewReader(namespaceToolAnthropicStream()))}

	svc := &GatewayService{}
	_, err := svc.handleResponsesStreamingResponse(resp, c, "claude-fable-5", "claude-fable-5", nil, time.Now(), namespaceToolMapping())
	require.NoError(t, err)
	require.Contains(t, rec.Body.String(), `response.output_item.added`)
	require.Contains(t, rec.Body.String(), `"name":"read_thread"`)
	require.Contains(t, rec.Body.String(), `"namespace":"codex_app"`)
	require.NotContains(t, rec.Body.String(), `"name":"codex_app__read_thread"`)
}

func TestExtractResponsesReasoningEffortFromBody(t *testing.T) {
	t.Parallel()

	got := ExtractResponsesReasoningEffortFromBody([]byte(`{"model":"claude-sonnet-4.5","reasoning":{"effort":"HIGH"}}`))
	require.NotNil(t, got)
	require.Equal(t, "high", *got)

	maxGot := ExtractResponsesReasoningEffortFromBody([]byte(`{"model":"deepseek-v4-pro","reasoning":{"effort":"max"}}`))
	require.NotNil(t, maxGot)
	require.Equal(t, "max", *maxGot)

	mappedMax := ExtractResponsesReasoningEffortFromBody(
		[]byte(`{"model":"public-alias","reasoning":{"effort":"max"}}`),
		"provider/glm-5.2",
		"public-alias",
	)
	require.NotNil(t, mappedMax)
	require.Equal(t, "max", *mappedMax)

	legacyMax := ExtractResponsesReasoningEffortFromBody([]byte(`{"model":"gpt-5.5","reasoning":{"effort":"max"}}`))
	require.NotNil(t, legacyMax)
	require.Equal(t, "xhigh", *legacyMax)

	require.Nil(t, ExtractResponsesReasoningEffortFromBody([]byte(`{"model":"claude-sonnet-4.5"}`)))
}

func TestHandleResponsesBufferedStreamingResponse_PreservesMessageStartCacheUsage(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	resp := &http.Response{
		Header: http.Header{"x-request-id": []string{"rid_buffered"}},
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
	result, err := svc.handleResponsesBufferedStreamingResponse(resp, c, "claude-sonnet-4.5", "claude-sonnet-4.5", nil, time.Now(), apicompat.ResponsesClientToolMapping{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 12, result.Usage.InputTokens)
	require.Equal(t, 7, result.Usage.OutputTokens)
	require.Equal(t, 9, result.Usage.CacheReadInputTokens)
	require.Equal(t, 3, result.Usage.CacheCreationInputTokens)
	require.Contains(t, rec.Body.String(), `"cached_tokens":9`)
}

func TestHandleResponsesStreamingResponse_PreservesMessageStartCacheUsage(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	resp := &http.Response{
		Header: http.Header{"x-request-id": []string{"rid_stream"}},
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
	result, err := svc.handleResponsesStreamingResponse(resp, c, "claude-sonnet-4.5", "claude-sonnet-4.5", nil, time.Now(), apicompat.ResponsesClientToolMapping{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 20, result.Usage.InputTokens)
	require.Equal(t, 8, result.Usage.OutputTokens)
	require.Equal(t, 11, result.Usage.CacheReadInputTokens)
	require.Equal(t, 4, result.Usage.CacheCreationInputTokens)
	require.Contains(t, rec.Body.String(), `response.completed`)
}

// ---------------------------------------------------------------------------
// ForwardAsResponses passthrough field regression tests
// ---------------------------------------------------------------------------
// These tests verify that account passthrough field rules correctly read values
// from the original Responses API request body (sourceBody) rather than the
// converted Anthropic body. This is the regression that would reappear if the
// buildUpstreamRequestWithSourceBody call were reverted to buildUpstreamRequest.
// ---------------------------------------------------------------------------

// TestGatewayService_ForwardAsResponses_PassthroughBodyMapCopiesFromOriginalResponsesBody
// verifies that a body "map" rule reads from the original Responses API body
// (which contains "instructions" as a top-level field) rather than the converted
// Anthropic body (which uses "system" instead).
//
// The Responses body contains a top-level "instructions" field (an OpenAI Responses-specific field).
// After conversion to Anthropic format, this becomes "system" and "instructions" no longer exists.
// The test configures a map rule: source_key="instructions" → key="metadata.client_instructions".
// If the fix is correct, the upstream body will contain metadata.client_instructions with the value.
// If reverted, the map would try to read "instructions" from the Anthropic body where it doesn't
// exist, so the field would NOT be injected.
func TestGatewayService_ForwardAsResponses_PassthroughBodyMapCopiesFromOriginalResponsesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	collector := &openAIUsageUpstreamRequestCollector{}
	c.Set(UsageDetailCaptureContextKey, collector)

	// Original Responses API body with "instructions" (OpenAI Responses-specific field).
	responsesBody := []byte(`{
		"model": "claude-sonnet-4-20250514",
		"input": "hello",
		"instructions": "be concise"
	}`)

	parsed := &ParsedRequest{Body: NewRequestBodyRef(responsesBody), Model: "claude-sonnet-4-20250514", Stream: false}

	upstreamSSE := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_resp","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":"","usage":{"input_tokens":5}}}`,
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
				"x-request-id": []string{"rid-resp-passthrough"},
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
		ID:          801,
		Name:        "resp-passthrough-map-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "upstream-key-resp",
			"base_url": "https://api.anthropic.com",
		},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				// Map: copy "instructions" from original Responses body → "metadata.client_instructions" in upstream
				{Target: "body", Mode: "map", Key: "metadata.client_instructions", SourceKey: "instructions"},
				// Inject: fixed header
				{Target: "header", Mode: "inject", Key: "X-Resp-Tag", Value: "resp-passthrough"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.ForwardAsResponses(context.Background(), c, account, responsesBody, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastBody, "upstream request should have been sent")

	// Core assertion: "instructions" value read from the ORIGINAL Responses body.
	mappedValue := gjson.GetBytes(upstream.lastBody, "metadata.client_instructions").String()
	require.Equal(t, "be concise", mappedValue,
		"body map rule should read 'instructions' from the original Responses body, not the converted Anthropic body")

	// Header injection should also work.
	require.Equal(t, "resp-passthrough", getHeaderRaw(upstream.lastReq.Header, "X-Resp-Tag"),
		"header inject rule should apply in ForwardAsResponses path")

	// Auth header should be replaced.
	require.Equal(t, "upstream-key-resp", getHeaderRaw(upstream.lastReq.Header, "x-api-key"))
	require.JSONEq(t, string(upstream.lastBody), collector.body)
	require.Equal(t, "be concise", gjson.Get(collector.body, "metadata.client_instructions").String())
	require.JSONEq(t, collector.body, requireOpsPreviewString(t, c, "be concise"))
	require.True(t, HasOpsUpstreamAttempted(c))
}

// TestGatewayService_ForwardAsResponses_PassthroughDisabledLeavesRulesInactive
// verifies that passthrough rules are NOT applied when passthrough_fields_enabled=false.
func TestGatewayService_ForwardAsResponses_PassthroughDisabledLeavesRulesInactive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	responsesBody := []byte(`{
		"model": "claude-sonnet-4-20250514",
		"input": "hello",
		"instructions": "should-not-appear"
	}`)

	parsed := &ParsedRequest{Body: NewRequestBodyRef(responsesBody), Model: "claude-sonnet-4-20250514", Stream: false}

	upstreamSSE := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_rdis","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":"","usage":{"input_tokens":5}}}`,
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
				"x-request-id": []string{"rid-resp-disabled"},
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
		ID:          802,
		Name:        "resp-passthrough-disabled-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "upstream-key-resp-disabled",
			"base_url": "https://api.anthropic.com",
		},
		Extra: map[string]any{
			"passthrough_fields_enabled": false, // disabled
			"passthrough_field_rules": []PassthroughFieldRule{
				{Target: "body", Mode: "map", Key: "metadata.client_instructions", SourceKey: "instructions"},
				{Target: "header", Mode: "inject", Key: "X-Resp-Tag", Value: "should-not-appear"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.ForwardAsResponses(context.Background(), c, account, responsesBody, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastBody)

	// Body map rule should NOT have been applied.
	require.False(t, gjson.GetBytes(upstream.lastBody, "metadata.client_instructions").Exists(),
		"body map field should not be injected when passthrough is disabled")

	// Header inject rule should NOT have been applied.
	require.Empty(t, getHeaderRaw(upstream.lastReq.Header, "X-Resp-Tag"),
		"header inject should not apply when passthrough is disabled")
}

// TestGatewayService_ForwardAsResponses_PassthroughBodyForwardCopiesFromOriginalResponsesBody
// verifies that a body "forward" rule reads the value from the original Responses API
// body (sourceBody) rather than the converted Anthropic body.
//
// "forward" mode copies body[key] from sourceBody → targetBody at the same key.
// The original Responses body has "instructions"; the converted Anthropic body converts
// that into "system" and removes "instructions". If the fix is correct, "instructions"
// will appear in the upstream body. If reverted, it won't.
func TestGatewayService_ForwardAsResponses_PassthroughBodyForwardCopiesFromOriginalResponsesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	responsesBody := []byte(`{
		"model": "claude-sonnet-4-20250514",
		"input": "hello",
		"instructions": "be verbose"
	}`)

	parsed := &ParsedRequest{Body: NewRequestBodyRef(responsesBody), Model: "claude-sonnet-4-20250514", Stream: false}

	upstreamSSE := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_rfwd","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":"","usage":{"input_tokens":5}}}`,
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
				"x-request-id": []string{"rid-resp-body-forward"},
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
		ID:          803,
		Name:        "resp-passthrough-body-forward-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "upstream-key-resp-fwd",
			"base_url": "https://api.anthropic.com",
		},
		Extra: map[string]any{
			"passthrough_fields_enabled": true,
			"passthrough_field_rules": []PassthroughFieldRule{
				// Forward: copy "instructions" from original Responses body → same key in upstream body
				{Target: "body", Mode: "forward", Key: "instructions"},
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.ForwardAsResponses(context.Background(), c, account, responsesBody, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastBody)

	// Core assertion: "instructions" was read from the ORIGINAL Responses body via forward mode.
	forwardedValue := gjson.GetBytes(upstream.lastBody, "instructions").String()
	require.Equal(t, "be verbose", forwardedValue,
		"body forward rule should read 'instructions' from the original Responses body, not the converted Anthropic body")
}

// TestGatewayService_ForwardAsResponses_PassthroughHeaderForwardCopiesFromClientRequest
// verifies that a header "forward" rule copies a header from the client request to the
// upstream request in the ForwardAsResponses path.
func TestGatewayService_ForwardAsResponses_PassthroughHeaderForwardCopiesFromClientRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Request.Header.Set("X-Custom-Trace", "resp-trace-xyz")

	responsesBody := []byte(`{
		"model": "claude-sonnet-4-20250514",
		"input": "hello"
	}`)

	parsed := &ParsedRequest{Body: NewRequestBodyRef(responsesBody), Model: "claude-sonnet-4-20250514", Stream: false}

	upstreamSSE := strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"msg_rhdr","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":"","usage":{"input_tokens":5}}}`,
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
				"x-request-id": []string{"rid-resp-header-forward"},
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
		ID:          804,
		Name:        "resp-passthrough-header-forward-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "upstream-key-resp-hdr",
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

	result, err := svc.ForwardAsResponses(context.Background(), c, account, responsesBody, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Core assertion: header was forwarded from client request.
	require.Equal(t, "resp-trace-xyz", getHeaderRaw(upstream.lastReq.Header, "X-Custom-Trace"),
		"header forward rule should copy X-Custom-Trace from client request to upstream")
}

func TestParseAnthropicSSEField(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		line, field, want string
		ok                bool
	}{
		{"event: message_start", "event", "message_start", true},
		{"event:message_start", "event", "message_start", true},
		{"data: {\"type\":\"message_start\"}", "data", "{\"type\":\"message_start\"}", true},
		{"data:{\"type\":\"message_start\"}", "data", "{\"type\":\"message_start\"}", true},
		{"event:  message_delta", "event", "message_delta", true},
		{"event: message_start", "data", "", false},
		{"", "event", "", false},
		{"invalid line", "event", "", false},
	} {
		got, ok := parseAnthropicSSEField(tt.line, tt.field)
		require.Equal(t, tt.ok, ok, tt.line)
		require.Equal(t, tt.want, got, tt.line)
	}
}

func TestHandleResponsesBufferedStreamingResponse_CompactSSEFormat(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	resp := &http.Response{
		Header: http.Header{"x-request-id": []string{"rid_compact"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`event:message_start`,
			`data:{"type":"message_start","message":{"id":"msg_compact","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4.5","stop_reason":"","usage":{"input_tokens":10}}}`,
			``,
			`event:content_block_start`,
			`data:{"type":"content_block_start","index":0,"content_block":{"type":"text","text":"OK"}}`,
			``,
			`event:message_delta`,
			`data:{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}`,
			``,
		}, "\n"))),
	}

	svc := &GatewayService{}
	result, err := svc.handleResponsesBufferedStreamingResponse(resp, c, "claude-sonnet-4.5", "claude-sonnet-4.5", nil, time.Now(), apicompat.ResponsesClientToolMapping{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 10, result.Usage.InputTokens)
	require.Equal(t, 5, result.Usage.OutputTokens)
}

func TestHandleResponsesStreamingResponse_CompactSSEFormat(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	resp := &http.Response{
		Header: http.Header{"x-request-id": []string{"rid_compact_stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`event:message_start`,
			`data:{"type":"message_start","message":{"id":"msg_compact_stream","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4.5","stop_reason":"","usage":{"input_tokens":15}}}`,
			``,
			`event:content_block_start`,
			`data:{"type":"content_block_start","index":0,"content_block":{"type":"text","text":"OK"}}`,
			``,
			`event:message_delta`,
			`data:{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":6}}`,
			``,
			`event:message_stop`,
			`data:{"type":"message_stop"}`,
			``,
		}, "\n"))),
	}

	svc := &GatewayService{}
	result, err := svc.handleResponsesStreamingResponse(resp, c, "claude-sonnet-4.5", "claude-sonnet-4.5", nil, time.Now(), apicompat.ResponsesClientToolMapping{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 15, result.Usage.InputTokens)
	require.Equal(t, 6, result.Usage.OutputTokens)
	require.Contains(t, rec.Body.String(), `response.completed`)
}
