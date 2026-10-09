//go:build unit

package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestUpstreamProtocolFusionMappedModelAndBorrowedHandle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, ingress := range routingMatrixIngresses() {
		for _, target := range []struct{ model, endpoint string }{
			{"minimax-m3", "http://anthropic.example/v1/messages"},
			{"gpt-5.6-luna", "http://responses.example/v1/responses"},
		} {
			t.Run(ingress.name+"/"+target.model, func(t *testing.T) {
				tc := routingMatrixCase{platform: PlatformOpenCodeGo, accountType: AccountTypeAPIKey, addresses: "split", ingress: ingress, model: "public-alias"}
				account := tc.account()
				account.Credentials["model_mapping"] = map[string]any{"public-alias": target.model}
				body := tc.body()
				handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
				require.NoError(t, err)
				t.Cleanup(func() { CleanupRequestBodyHandle(handle) })
				c := adaptiveProtocolTestContext(ingress.path, body)
				BindOpenAIRequestBodyHandle(c, handle)
				upstream := &httpUpstreamRecorder{err: errors.New("stop after capture")}
				svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}

				require.Error(t, ingress.forward(svc, c, account, body))
				require.Len(t, upstream.requests, 1)
				require.Equal(t, target.endpoint, upstream.lastReq.URL.String())
				require.Equal(t, target.model, gjson.GetBytes(upstream.lastBody, "model").String())
				retained, err := handle.ReadAll()
				require.NoError(t, err, "forwarding must not release its caller's spool")
				require.Equal(t, body, retained, "account mapping must not mutate the canonical replay body")
			})
		}
	}
}

func TestUpstreamProtocolFusionNativeAnthropicCustomBase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, ingress := range routingMatrixIngresses() {
		for _, base := range []string{"http://anthropic.example/custom", "http://anthropic.example/custom/v1", "http://anthropic.example/custom/v1/messages"} {
			t.Run(fmt.Sprintf("%s/%s", ingress.name, base), func(t *testing.T) {
				tc := routingMatrixCase{platform: PlatformKimi, accountType: AccountTypeAPIKey, apiProtocol: APIProtocolAnthropic, ingress: ingress, model: "kimi-k2"}
				account := tc.account()
				account.Credentials["base_url"] = base
				upstream := &httpUpstreamRecorder{err: errors.New("stop after capture")}
				svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
				body := tc.body()

				require.Error(t, ingress.forward(svc, adaptiveProtocolTestContext(ingress.path, body), account, body))
				require.Len(t, upstream.requests, 1)
				require.Equal(t, "http://anthropic.example/custom/v1/messages", upstream.lastReq.URL.String())
				require.Equal(t, "sk-test", upstream.lastReq.Header.Get("X-Api-Key"))
				require.Equal(t, "2023-06-01", getHeaderRaw(upstream.lastReq.Header, "anthropic-version"))
			})
		}
	}
}
