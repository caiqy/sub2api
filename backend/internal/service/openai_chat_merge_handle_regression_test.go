package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChatMergeFallbackReplaysOriginalSpool(t *testing.T) {
	for _, bound := range []bool{false, true} {
		for _, unknown := range []bool{false, true} {
			t.Run(fmt.Sprintf("bound=%t/unknown=%t", bound, unknown), func(t *testing.T) {
				body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"` + strings.Repeat("hello ", 180000) + `"}],"vendor_option":"keep","stream":false}`)
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
				input := body
				var canonical *RequestBodyHandle
				if bound {
					var err error
					canonical, err = NewRequestBodyHandleFromBytes(body, openAIRequestBodyHandleOptions())
					require.NoError(t, err)
					defer CleanupRequestBodyHandle(canonical)
					require.NotEmpty(t, canonical.spoolPath)
					BindOpenAIRequestBodyHandle(c, canonical)
					input = nil
				}
				account := structuredInputRetryTestAccount()
				status := http.StatusBadRequest
				if unknown {
					delete(account.Extra, openai_compat.ExtraKeyResponsesSupported)
					status = http.StatusNotFound
				}
				initialBody := &openAIChatFallbackDelayReader{Reader: strings.NewReader(`{"error":{"code":"invalid_type","param":"input","message":"Expected a string, but got an array instead."}}`)}
				upstream := &httpUpstreamRecorder{responses: []*http.Response{
					{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(initialBody)},
					{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))},
				}}
				svc := &OpenAIGatewayService{cfg: structuredInputRetryTestConfig(), httpUpstream: upstream}
				result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, input, "", "")
				require.NoError(t, err)
				require.Len(t, upstream.requests, 2)
				require.Equal(t, "/v1/responses", upstream.requests[0].URL.Path)
				require.Equal(t, "/v1/chat/completions", upstream.requests[1].URL.Path)
				require.JSONEq(t, string(body), string(upstream.bodies[1]))
				require.Equal(t, upstream.bodies[1], upstream.replays[1])
				require.GreaterOrEqual(t, result.Duration, initialBody.elapsed)
				if canonical != nil {
					replayed, err := canonical.ReadAll()
					require.NoError(t, err)
					require.Equal(t, body, replayed)
					require.FileExists(t, canonical.spoolPath)
				}
			})
		}
	}
}

func TestChatMergeRawWithStartTimeReadsBoundHandle(t *testing.T) {
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	canonical, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1, TempDir: t.TempDir()})
	require.NoError(t, err)
	defer CleanupRequestBodyHandle(canonical)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	BindOpenAIRequestBodyHandle(c, canonical)
	upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"ok"}}]}`))}}
	svc := &OpenAIGatewayService{cfg: structuredInputRetryTestConfig(), httpUpstream: upstream}
	result, err := svc.forwardAsRawChatCompletionsWithStartTime(context.Background(), c, structuredInputRetryTestAccount(), nil, "", time.Now().Add(-time.Second))
	require.NoError(t, err)
	require.JSONEq(t, string(body), string(upstream.lastBody))
	require.GreaterOrEqual(t, result.Duration, time.Second)
	require.FileExists(t, canonical.spoolPath)
}
