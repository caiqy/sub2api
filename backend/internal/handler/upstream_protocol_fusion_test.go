//go:build unit

package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestUpstreamProtocolFusionHandlerBodyCacheLifecycle(t *testing.T) {
	for _, ingress := range []struct{ path, body string }{
		{"/v1/responses", `{"model":"public-alias","input":"hello","stream":false}`},
		{"/v1/chat/completions", `{"model":"public-alias","messages":[{"role":"user","content":"hello"}],"stream":false}`},
		{"/v1/messages", `{"model":"public-alias","max_tokens":32,"messages":[{"role":"user","content":"hello"}],"stream":false}`},
	} {
		t.Run(ingress.path, func(t *testing.T) {
			spoolDir := t.TempDir()
			oldOptions := jsonRequestBodyHandleOptions
			jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: spoolDir}
			t.Cleanup(func() { jsonRequestBodyHandleOptions = oldOptions })
			env := newOpenAIResponsesRetentionTestEnv(t, nil, nil, nil, nil, nil, nil)
			env.upstream.err = errors.New("stop after capture")
			svc := env.handler.gatewayService
			sourceRequest := httptest.NewRequest(http.MethodPost, ingress.path, bytes.NewBufferString(ingress.body))
			coordinator, err := newJSONRequestBody(sourceRequest)
			require.NoError(t, err)
			t.Cleanup(coordinator.Cleanup)
			raw, err := coordinator.ReadRaw()
			require.NoError(t, err)
			calls := 0
			mappedBody := newOpenAIModelMappedBodyCache(raw, func(body []byte, model string) []byte {
				calls++
				return svc.ReplaceModelInBody(body, model)
			})
			require.NoError(t, coordinator.SetEffectiveBytes(mappedBody(true, "channel-alias")))
			effective := coordinator.Effective()
			for i, target := range []struct{ model, endpoint string }{
				{"minimax-m3", "https://anthropic.example/v1/messages"},
				{"gpt-5.6-luna", "https://responses.example/v1/responses"},
			} {
				account := &service.Account{ID: int64(900 + i), Platform: service.PlatformOpenCodeGo, Type: service.AccountTypeAPIKey, Concurrency: 1,
					Credentials: map[string]any{"api_key": "sk-test", "model_mapping": map[string]any{"channel-alias": target.model},
						"api_base_urls": map[string]any{service.APIProtocolAnthropic: "https://anthropic.example/v1", service.APIProtocolResponses: "https://responses.example/v1"}}}
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, ingress.path, nil)
				service.BindOpenAIRequestBodyHandle(c, effective)
				body := mappedBody(true, "channel-alias")
				switch ingress.path {
				case "/v1/responses":
					_, err = svc.Forward(context.Background(), c, account, body)
				case "/v1/chat/completions":
					_, err = svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "")
				case "/v1/messages":
					_, err = svc.ForwardAsAnthropic(context.Background(), c, account, body, "", "")
				}
				require.Error(t, err)
				require.Len(t, env.upstream.requests, i+1)
				request := env.upstream.requests[i]
				require.Equal(t, target.endpoint, request.URL.String())
				wireBody, err := io.ReadAll(request.Body)
				require.NoError(t, err)
				require.Equal(t, target.model, gjson.GetBytes(wireBody, "model").String())
				retained, err := effective.ReadAll()
				require.NoError(t, err)
				require.Equal(t, "channel-alias", gjson.GetBytes(retained, "model").String(), "one account's wire model must not poison another attempt's cached body")
			}
			require.Equal(t, 1, calls)
			retainedRaw, err := coordinator.ReadRaw()
			require.NoError(t, err)
			require.Equal(t, ingress.body, string(retainedRaw))
			coordinator.Cleanup()
			_, err = effective.ReadAll()
			require.Error(t, err)
			_, err = coordinator.ReadRaw()
			require.Error(t, err)
			entries, err := os.ReadDir(spoolDir)
			require.NoError(t, err)
			require.Empty(t, entries)
		})
	}
}
