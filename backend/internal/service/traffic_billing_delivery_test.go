//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTrafficBillingUpstreamFailureRetainsModelBill(t *testing.T) {
	for _, protocol := range []string{"messages", "chat", "responses"} {
		t.Run(protocol, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/"+protocol, nil)
			upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: &streamReadCloser{payload: []byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"m\",\"model\":\"claude-sonnet-4\",\"usage\":{\"input_tokens\":9}}}\n\nevent: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hi\"}}\n\n"), err: io.ErrUnexpectedEOF}}}
			forward := newForwardPartialUsageServiceForTest(upstream)
			account := newAnthropicOAuthAccountForPartialUsageTest()
			var result *ForwardResult
			var err error
			switch protocol {
			case "messages":
				parsed, parseErr := ParseGatewayRequest(NewRequestBodyRef([]byte(`{"model":"claude-sonnet-4","stream":true,"messages":[{"role":"user","content":"hi"}]}`)), PlatformAnthropic)
				require.NoError(t, parseErr)
				result, err = forward.Forward(context.Background(), c, account, parsed)
			case "chat":
				result, err = forward.ForwardAsChatCompletions(context.Background(), c, account, []byte(`{"model":"claude-sonnet-4","stream":true,"messages":[{"role":"user","content":"hi"}]}`), nil)
			case "responses":
				handle, createErr := NewRequestBodyHandleFromBytes([]byte(`{"model":"claude-sonnet-4","stream":true,"input":"hi"}`), RequestBodyHandleOptions{})
				require.NoError(t, createErr)
				defer CleanupRequestBodyHandle(handle)
				result, err = forward.ForwardAsResponsesHandle(context.Background(), c, account, handle, nil)
			}
			require.Error(t, err)
			require.NotNil(t, result, "forward error: %v", err)
			require.False(t, result.ClientDisconnect)
			n := int64(12_000_000)
			ctx := WithTrafficBillingSnapshot(context.Background(), TrafficBillingSnapshot{Enabled: true, Consent: true, Successful: true, InputBytes: &n, ThresholdBytes: 10_000_000, PricePerMB: 0.01})
			logs := &openAIRecordUsageLogRepoStub{inserted: true}
			bills := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
			billing := newGatewayRecordUsageServiceWithBillingRepoForTest(logs, bills, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{})
			require.NoError(t, billing.RecordUsage(ctx, &RecordUsageInput{Result: result, APIKey: &APIKey{ID: 2, Group: &Group{RateMultiplier: 1}}, User: &User{ID: 1}, Account: account}))
			require.Zero(t, logs.lastLog.TrafficCost)
			require.Equal(t, "delivery_incomplete", logs.lastLog.TrafficBillingReason)
			require.Positive(t, logs.lastLog.InputCost)
			require.Positive(t, bills.lastCmd.BalanceCost)
		})
	}
}
