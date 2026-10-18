package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/websearch"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIMergeMinOutputTokensActualHTTPPaths(t *testing.T) {
	for _, path := range []string{"chat", "messages_handle", "legacy_http", "prepare_passthrough"} {
		t.Run(path, func(t *testing.T) {
			body := []byte(`{"model":"gpt-4.1","instructions":"test","max_tokens":16,"messages":[{"role":"user","content":"hello"}],"stream":false}`)
			handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
			require.NoError(t, err)
			defer CleanupRequestBodyHandle(handle)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			BindOpenAIRequestBodyHandle(c, handle)
			upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(`data: {"type":"response.completed","response":{"id":"resp_merge","object":"response","model":"gpt-4.1","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":1,"output_tokens":1}}}` + "\n\n"))}}
			cfg := structuredInputRetryTestConfig()
			cfg.Gateway.MinOutputTokens = 256
			svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream, toolCorrector: NewCodexToolCorrector()}
			account := structuredInputRetryTestAccount()
			var sent []byte
			switch path {
			case "chat":
				_, err = svc.ForwardAsChatCompletions(context.Background(), c, account, nil, "", "")
				sent = upstream.lastBody
			case "messages_handle":
				_, err = svc.forwardAsAnthropicHandle(context.Background(), c, account, handle, "", "")
				sent = upstream.lastBody
			case "legacy_http":
				_, err = svc.Forward(context.Background(), c, account, body)
				sent = upstream.lastBody
			case "prepare_passthrough":
				account.Type = AccountTypeOAuth
				account.Credentials["access_token"] = "test-oauth-token"
				_, err = svc.forwardOpenAIPassthrough(context.Background(), c, account, body, body, "gpt-4.1", false, nil, false, time.Now())
				sent = upstream.lastBody
			}
			require.NoError(t, err)
			require.EqualValues(t, 256, gjson.GetBytes(sent, "max_output_tokens").Int(), string(sent))
			require.FileExists(t, handle.spoolPath, "caller retains canonical spool ownership")
			original, readErr := handle.ReadAll()
			require.NoError(t, readErr)
			require.Equal(t, body, original)
		})
	}
}

func TestOpenAIMergeMinOutputTokensRawMessagesFallback(t *testing.T) {
	for _, field := range []string{`,"max_tokens":16`, `,"max_tokens":0`, ""} {
		t.Run(field, func(t *testing.T) {
			body := []byte(`{"model":"gpt-4.1","messages":[{"role":"user","content":"hello"}]` + field + `}`)
			handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
			require.NoError(t, err)
			defer CleanupRequestBodyHandle(handle)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
			upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"chat_merge","model":"gpt-4.1","choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))}}
			cfg := structuredInputRetryTestConfig()
			cfg.Gateway.MinOutputTokens = 256
			svc := &OpenAIGatewayService{cfg: cfg, httpUpstream: upstream}
			account := structuredInputRetryTestAccount()
			account.Extra[openai_compat.ExtraKeyResponsesMode] = string(openai_compat.ResponsesSupportModeForceChatCompletions)
			_, err = svc.forwardAsAnthropicHandle(context.Background(), c, account, handle, "", "")
			require.NoError(t, err)
			require.Equal(t, "/v1/chat/completions", upstream.lastReq.URL.Path)
			cap := gjson.GetBytes(upstream.lastBody, "max_completion_tokens")
			if field == `,"max_tokens":16` {
				require.EqualValues(t, 256, cap.Int())
			} else {
				require.False(t, cap.Exists(), "omitted/zero limits must not invent a cap")
			}
			require.FileExists(t, handle.spoolPath)
		})
	}
}

func TestOpenAIMergeEmulatedSearchSuccessKeepsZeroBillableUsage(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "json", true: "sse"}[stream], func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			search := &websearch.SearchResponse{Query: "test", Results: []websearch.SearchResult{{URL: "https://example.test", Title: "Result", Snippet: "A search response"}}}
			write := writeWebSearchNonStreamResponse
			if stream {
				write = writeWebSearchStreamResponse
			}
			result, err := write(c, "test", search, "gpt-4.1", time.Now())
			require.NoError(t, err)
			require.Equal(t, ClaudeUsage{}, result.Usage, "client token estimates must not become billable upstream usage")
			require.Zero(t, result.SearchCount)
			require.Contains(t, recorder.Body.String(), "web_search_tool_result")
		})
	}
}

func TestOpenAIMergeMessagesWebSearchUsesCanonicalHandle(t *testing.T) {
	previousManager := getWebSearchManager()
	previousConfig, _ := webSearchEmulationCache.Load().(*cachedWebSearchEmulationConfig)
	if previousConfig == nil {
		previousConfig = &cachedWebSearchEmulationConfig{}
	}
	SetWebSearchManager(websearch.NewManager([]websearch.ProviderConfig{{Type: "brave", APIKey: "unused"}}, nil))
	// Expire the cache so the shared settings service reads the complete
	// enabled provider configuration from this test's repository.
	webSearchEmulationCache.Store(&cachedWebSearchEmulationConfig{})
	t.Cleanup(func() { SetWebSearchManager(previousManager); webSearchEmulationCache.Store(previousConfig) })
	body := []byte(`{"model":"gpt-4.1","tools":[{"type":"web_search"}],"messages":[]}`)
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	require.NoError(t, err)
	defer CleanupRequestBodyHandle(handle)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Extra: map[string]any{featureKeyWebSearchEmulation: WebSearchModeEnabled}}
	settings := NewSettingService(&antigravitySettingRepoValuesStub{values: map[string]string{
		SettingKeyWebSearchEmulationConfig: `{"enabled":true,"providers":[{"type":"brave","api_key":"sk-test"}]}`,
	}}, &config.Config{})
	svc := &OpenAIGatewayService{cfg: &config.Config{}, settingService: settings}
	require.True(t, svc.settingService.IsWebSearchEmulationEnabled(context.Background()), "the repository must provide an enabled search provider")
	_, err = svc.forwardAsAnthropicHandle(context.Background(), c, account, handle, "", "")
	require.EqualError(t, err, "web search emulation: no query found in messages", "must intercept before protocol conversion/upstream")
	require.FileExists(t, handle.spoolPath)
	require.NoError(t, os.Remove(handle.spoolPath))
	_, err = svc.forwardAsAnthropicHandle(context.Background(), c, account, handle, "", "")
	require.ErrorIs(t, err, ErrRequestBodySpool)
}
