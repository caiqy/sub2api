package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const codexRecapTestRequest = `{"model":"deepseek-v4.1-flash","instructions":"Keep the user's language.","input":"Write a brief catch-up.","tools":[],"tool_choice":"auto","stream":false,"reasoning":{"effort":"high"},"text":{"format":{"type":"json_schema","strict":true,"name":"codex_output_schema","schema":{"type":"object","properties":{"recap":{"type":"string","minLength":1,"maxLength":320}},"required":["recap"],"additionalProperties":false}}}}`

const codexTitleTestRequest = `{"model":"deepseek-v4.1-flash","instructions":"Generate a concise task title.","input":"Connect to the 185 server.","tools":[],"tool_choice":"auto","stream":false,"reasoning":{"effort":"xhigh"},"text":{"format":{"type":"json_schema","strict":true,"name":"codex_output_schema","schema":{"type":"object","properties":{"title":{"type":"string","minLength":1,"maxLength":36}},"required":["title"],"additionalProperties":false}}}}`

func codexRecapTestAccount() *Account {
	return &Account{ID: 368, Platform: PlatformDeepseek, Type: AccountTypeAPIKey, Credentials: map[string]any{
		"api_key": "test-key", "base_url": "https://opencode.ai/zen/go/v1/chat/completions", "api_protocol": APIProtocolChatCompletions,
	}}
}

func codexRecapTestResponse(content, finish string) string {
	raw, _ := json.Marshal(content)
	return fmt.Sprintf(`{"id":"chatcmpl_recap","object":"chat.completion","model":"deepseek-v4.1-flash","choices":[{"index":0,"message":{"role":"assistant","content":%s},"finish_reason":%q}],"usage":{"prompt_tokens":10,"completion_tokens":7,"total_tokens":17}}`, raw, finish)
}

func runCodexRecapTest(t *testing.T, body, response, userAgent string, account *Account) (*httptest.ResponseRecorder, *httpUpstreamRecorder, *OpenAIForwardResult, error) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	c.Request.Header.Set("User-Agent", userAgent)
	upstream := newOKChatCompletionsUpstream("rid_recap", response)
	svc := &OpenAIGatewayService{cfg: deepSeekChatFallbackTestConfig(), httpUpstream: upstream}
	result, err := svc.Forward(context.Background(), c, account, []byte(body))
	return rec, upstream, result, err
}

func TestForwardResponses_CodexRecap(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			body, err := sjson.Set(codexRecapTestRequest, "stream", stream)
			require.NoError(t, err)
			content := `{"recap":"已确认原因，下一步实现兼容。"}`
			rec, upstream, result, err := runCodexRecapTest(t, body, codexRecapTestResponse(content, "stop"), "codex_cli_rs/0.149.0", codexRecapTestAccount())
			require.NoError(t, err)
			require.Equal(t, "json_object", gjson.GetBytes(upstream.lastBody, "response_format.type").String())
			require.False(t, gjson.GetBytes(upstream.lastBody, "response_format.json_schema").Exists())
			require.False(t, gjson.GetBytes(upstream.lastBody, "stream").Bool())
			require.False(t, gjson.GetBytes(upstream.lastBody, "stream_options").Exists())
			require.Equal(t, "application/json", upstream.lastReq.Header.Get("Accept"))
			require.Contains(t, string(upstream.lastBody), "Keep the user's language.")
			require.Contains(t, string(upstream.lastBody), "recap")
			require.Contains(t, string(upstream.lastBody), "320")
			require.Len(t, upstream.requests, 1)
			require.Equal(t, 200, rec.Code)
			require.Equal(t, stream, result.Stream)
			require.Equal(t, 10, result.Usage.InputTokens)
			require.Equal(t, 7, result.Usage.OutputTokens)
			require.Equal(t, "json_schema", gjson.Get(body, "text.format.type").String())
			if !stream {
				require.Equal(t, content, gjson.Get(rec.Body.String(), "output.0.content.0.text").String())
				return
			}
			require.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
			var types []string
			var delta string
			for _, line := range strings.Split(rec.Body.String(), "\n") {
				if !strings.HasPrefix(line, "data: {") {
					continue
				}
				event := gjson.Parse(strings.TrimPrefix(line, "data: "))
				require.Equal(t, int64(len(types)), event.Get("sequence_number").Int())
				types = append(types, event.Get("type").String())
				if event.Get("type").String() == "response.output_text.delta" {
					delta += event.Get("delta").String()
				}
				if event.Get("type").String() == "response.completed" {
					require.Equal(t, content, event.Get("response.output.0.content.0.text").String())
					require.Equal(t, int64(7), event.Get("response.usage.output_tokens").Int())
				}
			}
			require.Equal(t, content, delta)
			require.Equal(t, "response.created", types[0])
			require.Equal(t, "response.completed", types[len(types)-1])
			require.Contains(t, types, "response.output_item.added")
			require.Contains(t, types, "response.output_text.done")
		})
	}
}

func TestForwardResponses_CodexTitle(t *testing.T) {
	content := `{"title":"连接 185 服务器"}`
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			body, err := sjson.Set(codexTitleTestRequest, "stream", stream)
			require.NoError(t, err)
			rec, upstream, result, err := runCodexRecapTest(t, body, codexRecapTestResponse(content, "stop"), "codex_cli_rs/0.149.0", codexRecapTestAccount())
			require.NoError(t, err)
			require.Equal(t, "json_object", gjson.GetBytes(upstream.lastBody, "response_format.type").String())
			require.Equal(t, stream, result.Stream)
			require.Contains(t, rec.Body.String(), "连接 185 服务器")
		})
	}
}

func TestForwardResponses_CodexTitleRejectsInvalidOutput(t *testing.T) {
	for name, content := range map[string]string{
		"wrong_field": `{"recap":"ok"}`,
		"too_long":    `{"title":"` + strings.Repeat("中", 37) + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec, _, result, err := runCodexRecapTest(t, codexTitleTestRequest, codexRecapTestResponse(content, "stop"), "codex_cli_rs/0.149.0", codexRecapTestAccount())
			require.Error(t, err)
			require.Equal(t, http.StatusBadGateway, rec.Code)
			require.NotNil(t, result, "失败时仍保留已消耗用量")
		})
	}
}

func TestForwardResponses_CodexRecapRejectsInvalidOutput(t *testing.T) {
	for name, content := range map[string]string{
		"plain_text": "not JSON", "extra_field": `{"recap":"ok","other":1}`,
		"duplicate_field": `{"recap":"bad","recap":"ok"}`, "missing": `{}`,
		"null": `{"recap":null}`, "number": `{"recap":1}`, "empty": `{"recap":""}`,
		"too_long": `{"recap":"` + strings.Repeat("中", 321) + `"}`,
		"trailing": `{"recap":"ok"} {}`, "array": `[{"recap":"ok"}]`,
		"truncated": `{"recap":"unfinished`,
	} {
		t.Run(name, func(t *testing.T) {
			body, err := sjson.Set(codexRecapTestRequest, "stream", true)
			require.NoError(t, err)
			rec, upstream, result, err := runCodexRecapTest(t, body, codexRecapTestResponse(content, "stop"), "codex_cli_rs/0.149.0", codexRecapTestAccount())
			require.Error(t, err)
			require.Equal(t, http.StatusBadGateway, rec.Code)
			require.Equal(t, "upstream_error", gjson.Get(rec.Body.String(), "error.type").String())
			require.NotContains(t, rec.Body.String(), "response.output_text")
			require.NotContains(t, rec.Body.String(), "response.completed")
			require.NotNil(t, result, "失败时仍保留已消耗用量")
			require.Equal(t, 7, result.Usage.OutputTokens)
			require.Len(t, upstream.requests, 1)
		})
	}
	for _, finish := range []string{"length", "content_filter", "tool_calls", ""} {
		t.Run("finish_"+finish, func(t *testing.T) {
			rec, _, _, err := runCodexRecapTest(t, codexRecapTestRequest, codexRecapTestResponse(`{"recap":"ok"}`, finish), "codex_cli_rs/0.149.0", codexRecapTestAccount())
			require.Error(t, err)
			require.Equal(t, http.StatusBadGateway, rec.Code)
		})
	}
}

func TestForwardResponses_CodexRecapScope(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		value      any
	}{
		{"other_model", "model", "deepseek-v4-pro"},
		{"other_schema", "text.format.name", "other_schema"},
		{"not_strict", "text.format.strict", false},
		{"different_limit", "text.format.schema.properties.recap.maxLength", 100},
		{"extra_constraint", "text.format.schema.properties.recap.pattern", "^ok$"},
		{"extra_property", "text.format.schema.properties.other", map[string]any{"type": "string"}},
		{"extra_root_constraint", "text.format.schema.maxProperties", 1},
		{"tools", "tools", []any{map[string]any{"type": "function", "name": "test", "parameters": map[string]any{"type": "object"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := sjson.Set(codexRecapTestRequest, tc.path, tc.value)
			require.NoError(t, err)
			_, upstream, _, err := runCodexRecapTest(t, body, codexRecapTestResponse("unvalidated", "stop"), "codex_cli_rs/0.149.0", codexRecapTestAccount())
			require.NoError(t, err)
			require.Equal(t, "json_schema", gjson.GetBytes(upstream.lastBody, "response_format.type").String())
		})
	}
	for _, target := range []string{"https://api.deepseek.com", "https://opencode.ai.example/zen/go/v1/chat/completions", "https://opencode.ai/zen/v1/chat/completions"} {
		t.Run(target, func(t *testing.T) {
			account := codexRecapTestAccount()
			account.Credentials["base_url"] = target
			_, upstream, _, err := runCodexRecapTest(t, codexRecapTestRequest, codexRecapTestResponse("unvalidated", "stop"), "codex_cli_rs/0.149.0", account)
			require.NoError(t, err)
			require.Equal(t, "json_schema", gjson.GetBytes(upstream.lastBody, "response_format.type").String())
		})
	}
	_, upstream, _, err := runCodexRecapTest(t, codexRecapTestRequest, codexRecapTestResponse("unvalidated", "stop"), "other-client/1.0", codexRecapTestAccount())
	require.NoError(t, err)
	require.True(t, bytes.Contains(upstream.lastBody, []byte(`"json_schema"`)))
}

func TestForwardResponses_CodexRecapDuplicateSchemaConstraint(t *testing.T) {
	body := strings.Replace(codexRecapTestRequest, `"maxLength":320`, `"maxLength":10,"maxLength":320`, 1)
	_, upstream, _, err := runCodexRecapTest(t, body, codexRecapTestResponse("unvalidated", "stop"), "codex_cli_rs/0.149.0", codexRecapTestAccount())
	require.NoError(t, err)
	require.Equal(t, "json_schema", gjson.GetBytes(upstream.lastBody, "response_format.type").String())
}

func TestForwardResponses_CodexRecapReadFailureCommitted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, upstreamBody := range []string{`{"choices":`, "data: [DONE]\n\n"} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		body, err := sjson.Set(codexRecapTestRequest, "stream", true)
		require.NoError(t, err)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
		c.Request.Header.Set("User-Agent", "codex_cli_rs/0.149.0")
		svc := &OpenAIGatewayService{cfg: deepSeekChatFallbackTestConfig(), httpUpstream: newOKChatCompletionsUpstream("rid_recap", upstreamBody)}
		_, err = svc.Forward(context.Background(), c, codexRecapTestAccount(), []byte(body))
		require.Error(t, err)
		require.Equal(t, http.StatusBadGateway, rec.Code)
		require.True(t, IsResponseCommitted(c), "避免外层在 JSON 错误后追加 SSE")
		require.True(t, json.Valid(rec.Body.Bytes()))
	}
}

func TestValidateCodexRecapResponseBoundaries(t *testing.T) {
	for _, length := range []int{1, 320} {
		content := `{"recap":"` + strings.Repeat("😀", length) + `"}`
		var response apicompat.ChatCompletionsResponse
		require.NoError(t, json.Unmarshal([]byte(codexRecapTestResponse(content, "stop")), &response))
		got, err := validateCodexStringOutputResponse(&response, &codexStringOutputSpecs[0])
		require.NoError(t, err)
		require.Equal(t, content, got)
	}
	var response apicompat.ChatCompletionsResponse
	raw := strings.Replace(codexRecapTestResponse(`{"recap":"ok"}`, "stop"), "ok", string([]byte{0xff}), 1)
	require.NoError(t, json.Unmarshal([]byte(raw), &response))
	_, err := validateCodexStringOutputResponse(&response, &codexStringOutputSpecs[0])
	require.Error(t, err, "在解码替换非法 UTF-8 之前拒绝响应")
}

func TestForwardResponses_CodexRecapOpenCodeGoMappedModel(t *testing.T) {
	account := codexRecapTestAccount()
	account.Platform = PlatformOpenCodeGo
	account.Credentials["base_url"] = "https://opencode.ai/zen/go"
	account.Credentials["api_protocol"] = APIProtocolAdaptive
	account.Credentials["model_mapping"] = map[string]any{"deepseek-flash": "deepseek-v4.1-flash"}
	body, err := sjson.Set(codexRecapTestRequest, "model", "deepseek-flash")
	require.NoError(t, err)
	body, err = sjson.Set(body, "stream", true)
	require.NoError(t, err)
	response, err := sjson.Set(codexRecapTestResponse(`{"recap":"ok"}`, "stop"), "choices.0.message.reasoning", "brief reasoning")
	require.NoError(t, err)
	rec, upstream, result, err := runCodexRecapTest(t, body, response, "codex_cli_rs/0.149.0", account)
	require.NoError(t, err)
	require.Equal(t, "https://opencode.ai/zen/go/v1/chat/completions", upstream.lastReq.URL.String())
	require.Equal(t, "json_object", gjson.GetBytes(upstream.lastBody, "response_format.type").String())
	require.Equal(t, "deepseek-v4.1-flash", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "deepseek-flash", result.Model)
	require.Contains(t, rec.Body.String(), `"model":"deepseek-flash"`)
	require.Contains(t, rec.Body.String(), `"delta":"brief reasoning"`)
	require.Contains(t, rec.Body.String(), "response.completed")
}
