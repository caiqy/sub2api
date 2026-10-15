//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestAccountTestService_BackgroundRetainsAntigravityPrompt(t *testing.T) {
	for _, model := range []string{"claude-sonnet-4-6", "gemini-2.5-flash"} {
		t.Run(model, func(t *testing.T) {
			account := &Account{
				ID: 902, Platform: PlatformAntigravity, Type: AccountTypeOAuth,
				Status: StatusActive, Concurrency: 1,
				Credentials: map[string]any{
					"access_token": "test-token", "project_id": "test-project",
					"model_mapping": map[string]any{model: model},
				},
			}
			response := newJSONResponse(http.StatusOK, "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}]}}\n\n")
			svc, upstream := adaptiveCNAccountTestService(account, response)
			svc.antigravityGatewayService = &AntigravityGatewayService{
				tokenProvider: &AntigravityTokenProvider{}, httpUpstream: upstream,
			}
			result, err := svc.RunTestBackground(context.Background(), account.ID, model)
			require.NoError(t, err)
			require.Equal(t, "success", result.Status, result.ErrorMessage)
			require.Equal(t, ".", gjson.GetBytes(upstream.lastBody, "request.contents.0.parts.0.text").String())
		})
	}
}

func TestAccountTestService_TextPromptReachesUpstream(t *testing.T) {
	for _, prompt := range []string{"  请回复 \"收到\"\nsecond line  ", "", " \t\n\u3000"} {
		want := "请回复 \"收到\"\nsecond line"
		if prompt == "" || prompt == " \t\n\u3000" {
			want = "hi"
		}
		for _, route := range []string{
			"claude-apikey", "claude-oauth", "claude-bedrock", "claude-vertex",
			"openai-responses", "openai-oauth", "openai-chat", "openai-compact",
			"gemini", "grok-default", "grok-text", "cn-anthropic", "cn-adaptive",
			"model-anthropic", "model-responses", "antigravity-apikey", "antigravity-claude", "antigravity-gemini",
		} {
			t.Run(route+"/"+prompt, func(t *testing.T) {
				account := &Account{
					ID: 901, Platform: PlatformAnthropic, Type: AccountTypeAPIKey,
					Status: StatusActive, Concurrency: 1,
					Credentials: map[string]any{"api_key": "test-key", "base_url": "https://upstream.example"},
				}
				model, mode := "claude-sonnet-4-6", AccountTestModeDefault
				paths := []string{"messages.0.content.0.text"}
				responses := []*http.Response{adaptiveCNAnthropicTestResponse()}
				switch route {
				case "claude-oauth":
					account.Type = AccountTypeOAuth
					account.Credentials["access_token"] = "test-token"
				case "claude-bedrock":
					account.Type = AccountTypeBedrock
					account.Credentials["auth_mode"] = "apikey"
					responses = []*http.Response{newJSONResponse(http.StatusOK, `{"content":[{"text":"ok"}]}`)}
				case "claude-vertex":
					account.Type = AccountTypeServiceAccount
					account.Credentials["service_account_json"] = `{"client_email":"test@example.com","private_key":"unused-cached-token","project_id":"test-project"}`
				case "openai-responses", "openai-oauth", "openai-chat", "openai-compact":
					account.Platform = PlatformOpenAI
					model = "gpt-5.4"
					paths = []string{"input.0.content.0.text"}
					responses = []*http.Response{adaptiveCNResponsesTestResponse()}
					if route == "openai-oauth" {
						account.Type = AccountTypeOAuth
						account.Credentials["access_token"] = "test-token"
					}
					if route == "openai-chat" {
						account.Extra = map[string]any{"openai_responses_mode": "force_chat_completions"}
						paths = []string{"messages.0.content"}
						responses = []*http.Response{adaptiveCNChatTestResponse()}
					}
					if route == "openai-compact" {
						mode = AccountTestModeCompact
						paths = []string{"input.0.content"}
						responses = []*http.Response{newJSONResponse(http.StatusOK, compactProbeSSESuccessBody)}
					}
				case "gemini":
					account.Platform = PlatformGemini
					model = "gemini-2.5-flash"
					paths = []string{"contents.0.parts.0.text"}
					responses = []*http.Response{newJSONResponse(http.StatusOK, "data: [DONE]\n\n")}
				case "grok-default", "grok-text":
					account.Platform = PlatformGrok
					model = "grok-4.3"
					paths = []string{"input"}
					responses = []*http.Response{adaptiveCNResponsesTestResponse()}
					if route == "grok-text" {
						mode = AccountTestModeGrokText
					}
				case "cn-anthropic":
					account.Platform = PlatformZhipu
					account.Credentials["api_protocol"] = APIProtocolAnthropic
				case "cn-adaptive":
					account = adaptiveCNAccountTestAccount(account.ID, PlatformDeepseek)
					model = "deepseek-chat"
					paths = []string{"messages.0.content", "messages.0.content.0.text", "input.0.content.0.text"}
					responses = []*http.Response{adaptiveCNChatTestResponse(), adaptiveCNAnthropicTestResponse(), adaptiveCNResponsesTestResponse()}
				case "model-anthropic", "model-responses":
					account = openCodeGoTestAccount(account.ID)
					model = "minimax-m3"
					if route == "model-responses" {
						model = "grok-4.6"
						paths = []string{"input.0.content.0.text"}
						responses = []*http.Response{adaptiveCNResponsesTestResponse()}
					}
				case "antigravity-apikey":
					account.Platform = PlatformAntigravity
				case "antigravity-claude", "antigravity-gemini":
					account.Platform = PlatformAntigravity
					account.Type = AccountTypeOAuth
					account.Credentials["access_token"] = "test-token"
					account.Credentials["project_id"] = "test-project"
					if route == "antigravity-gemini" {
						model = "gemini-2.5-flash"
					}
					account.Credentials["model_mapping"] = map[string]any{model: model}
					paths = []string{"request.contents.0.parts.0.text"}
					responses = []*http.Response{newJSONResponse(http.StatusOK, "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]},\"finishReason\":\"STOP\"}]}}\n\n")}
				}

				svc, upstream := adaptiveCNAccountTestService(account, responses...)
				if route == "claude-vertex" {
					key, err := parseVertexServiceAccountKey(account)
					require.NoError(t, err)
					cache := newClaudeTokenCacheStub()
					cache.tokens[vertexServiceAccountCacheKey(account, key)] = "cached-token"
					svc.claudeTokenProvider = NewClaudeTokenProvider(nil, cache, nil)
				}
				if route == "antigravity-claude" || route == "antigravity-gemini" {
					svc.antigravityGatewayService = &AntigravityGatewayService{
						tokenProvider: &AntigravityTokenProvider{}, httpUpstream: upstream,
					}
				}
				c, recorder := newTestContext()
				require.NoError(t, svc.TestAccountConnection(c, account.ID, model, prompt, mode), recorder.Body.String())
				require.Len(t, upstream.bodies, len(paths))
				for i, path := range paths {
					require.Equal(t, want, gjson.GetBytes(upstream.bodies[i], path).String(), "request %d: %s", i, upstream.bodies[i])
				}
				if route == "openai-compact" {
					require.Equal(t, "compaction_trigger", gjson.GetBytes(upstream.lastBody, "input.1.type").String())
				}
			})
		}
	}
}
