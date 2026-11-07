package service

import (
	"context"
	"net/http/httptest"
	"testing"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyCodexRestrictionForwardBoundaries(t *testing.T) {
	for _, routeOnly := range []bool{false, true} {
		for _, endpoint := range []string{"responses", "compact", "chat", "messages", "embeddings", "images", "alpha", "input_tokens", "count_tokens", "ws"} {
			t.Run(endpoint+map[bool]string{false: "/parent", true: "/route"}[routeOnly], func(t *testing.T) {
				s := &OpenAIGatewayService{}
				a := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Extra: map[string]any{"codex_cli_only": true, "codex_cli_only_route_only": routeOnly, "openai_passthrough": true}, Credentials: map[string]any{"api_key": "test"}}
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				path := "/v1/responses"
				if endpoint == "compact" {
					path += "/compact"
				}
				c.Request = httptest.NewRequest("POST", path, nil)
				body := []byte(`{"model":"gpt-5","input":"test","messages":[{"role":"user","content":"test"}]}`)
				s.PrepareCodexClientRouting(c, body)
				// Rewriting outbound identity must not change the inbound verdict.
				c.Request.Header.Set("User-Agent", "codex_cli_rs/0.141.0")
				c.Request.Header.Set("X-Codex-Test", "true")
				var err error
				switch endpoint {
				case "responses", "compact":
					_, err = s.Forward(c.Request.Context(), c, a, body)
				case "chat":
					_, err = s.ForwardAsChatCompletions(c.Request.Context(), c, a, body, "", "")
				case "messages":
					_, err = s.ForwardAsAnthropic(c.Request.Context(), c, a, body, "", "")
				case "embeddings":
					_, err = s.ForwardEmbeddings(c.Request.Context(), c, a, body, "")
				case "images":
					_, err = s.ForwardImages(c.Request.Context(), c, a, body, &OpenAIImagesRequest{}, "")
				case "alpha":
					_, err = s.ForwardAlphaSearch(c.Request.Context(), c, a, body)
				case "input_tokens":
					err = s.ForwardResponsesInputTokens(c.Request.Context(), c, a, body)
				case "count_tokens":
					err = s.ForwardCountTokensAsAnthropic(c.Request.Context(), c, a, body, "")
				case "ws":
					err = s.ProxyResponsesWebSocketFromClient(c.Request.Context(), c, &coderws.Conn{}, a, "test", body, nil)
				}
				require.ErrorContains(t, err, "only", endpoint)
				if endpoint != "ws" {
					require.Equal(t, 403, rec.Code)
				}
			})
		}
	}
}

func TestAPIKeyCodexRestrictionDefaultAndScope(t *testing.T) {
	s := &OpenAIGatewayService{}
	for _, typ := range []string{AccountTypeAPIKey, AccountTypeOAuth, AccountTypeSetupToken} {
		for _, platform := range []string{PlatformOpenAI, PlatformAnthropic} {
			a := &Account{Platform: platform, Type: typ, Extra: map[string]any{"codex_cli_only_allow_app_server": true, "codex_cli_only_route_only": true}}
			require.False(t, a.IsCodexCLIOnlyEnabled())
			a.Extra["codex_cli_only"] = true
			require.Equal(t, platform == PlatformOpenAI && typ != AccountTypeSetupToken, a.IsCodexCLIOnlyEnabled())
			if typ != AccountTypeAPIKey || platform != PlatformOpenAI {
				require.NoError(t, s.enforceAPIKeyCodexClientRestriction(nil, a, nil))
			}
		}
	}
	a := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Extra: map[string]any{"codex_cli_only": true}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	s.PrepareCodexClientRouting(c, nil)
	require.True(t, codexClientRouteAllowed(context.Background(), a)) // child off retains ordinary scheduling
	require.False(t, s.detectCodexClientRestriction(c, a, nil).Matched)
}
