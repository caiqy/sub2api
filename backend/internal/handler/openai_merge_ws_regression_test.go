package handler

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIMergeWSConfiguredLegacyTokenFloorEveryTurn(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		t.Run(map[bool]string{false: "ctx_pool", true: "passthrough"}[passthrough], func(t *testing.T) {
			env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{Passthrough: passthrough, CaptureUpstreamMessages: true, OAuthAccount: true, OAuthCredentials: map[string]any{"access_token": "test-oauth-token"}})
			defer env.Close()
			env.handler.cfg.Gateway.MinOutputTokens = 256
			env.apiKey.Group.Platform = service.PlatformOpenAI
			if passthrough {
				env.account.Extra["openai_oauth_responses_websockets_v2_mode"] = service.OpenAIWSIngressModePassthrough
			}
			// Keep the OAuth endpoint and authentication path, redirecting only
			// its transport to the local WS fixture.
			previousClient := http.DefaultClient
			transport := previousClient.Transport
			if transport == nil {
				transport = http.DefaultTransport
			}
			client := *previousClient
			client.Transport = openAIWSOAuthHandshakeTransportFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host != "chatgpt.com" {
					return transport.RoundTrip(req)
				}
				local := req.Clone(req.Context())
				u := *req.URL
				u.Scheme, u.Host = "http", strings.TrimPrefix(env.upstream.URL, "http://")
				local.URL, local.Host = &u, ""
				return transport.RoundTrip(local)
			})
			http.DefaultClient = &client
			t.Cleanup(func() { http.DefaultClient = previousClient })
			conn := env.dial(t)
			defer func() { _ = conn.CloseNow() }()
			for _, tokens := range []string{`,"max_tokens":16`, `,"max_tokens":16`, `,"max_tokens":0`, ""} {
				env.writeMessage(t, conn, `{"type":"response.create","model":"gpt-4.1","messages":[{"role":"user","content":"hello"}]`+tokens+`}`)
				payload := <-env.upstreamMessages
				cap := gjson.GetBytes(payload, "max_output_tokens")
				if tokens == `,"max_tokens":16` {
					require.EqualValues(t, 256, cap.Int(), string(payload))
				} else {
					require.False(t, cap.Exists(), string(payload))
				}
				require.False(t, gjson.GetBytes(payload, "messages").Exists())
				if passthrough {
					readOpenAIWSPassthroughCompletedTurn(t, env, conn)
				} else {
					require.Equal(t, "response.completed", gjson.GetBytes(env.readMessage(t, conn), "type").String())
				}
			}
			require.NoError(t, conn.Close(coderws.StatusNormalClosure, "done"))
			env.waitRequestDone(t)
		})
	}
}

func TestOpenAIMergeWSRejectsOAuthAmbiguityEveryTurn(t *testing.T) {
	for _, first := range []bool{false, true} {
		t.Run(map[bool]string{false: "later-turn", true: "first-turn"}[first], func(t *testing.T) {
			resolver := service.NewCompositeRouteResolver(nil)
			resolver.SetModelOwnershipResolver(func(_ context.Context, _ int64, model string) (service.CompositeModelOwnership, error) {
				if model == "gpt-4.1" {
					return service.CompositeModelOwnership{Ambiguous: true, OpenAIGatewayOAuthClaimed: true}, nil
				}
				return service.CompositeModelOwnership{}, nil
			})
			tc := openAIResponsesWSUsageLogCase{firstPayload: `{"type":"response.create","model":"gpt-4.1"}`, compositeResolver: resolver, group: compositeWSGroup(), firstFrameCloseExpected: first, secondTurnCloseExpected: !first, closeReason: "multiple provider platforms"}
			if !first {
				tc.firstPayload = `{"type":"response.create","model":"gpt-5.4"}`
				tc.secondPayload = `{"type":"response.create","model":"gpt-4.1"}`
			}
			runOpenAIResponsesWebSocketUsageLogCase(t, tc)
		})
	}
}
