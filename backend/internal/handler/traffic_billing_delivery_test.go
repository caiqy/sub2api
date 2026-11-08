//go:build unit

package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type trafficFailedHTTPWriter struct {
	*httptest.ResponseRecorder
	short bool
}

func (w *trafficFailedHTTPWriter) Write(p []byte) (int, error) {
	if w.short && len(p) > 0 {
		return len(p) - 1, nil
	}
	return 0, errors.New("simulated downstream write failure")
}

func TestTrafficBillingHTTPDeliveryThroughHandlers(t *testing.T) {
	for _, endpoint := range []string{"messages", "chat/completions", "responses"} {
		for _, mode := range []string{"partial", "write_failure", "short_write", "success"} {
			t.Run(endpoint+"/"+mode, func(t *testing.T) {
				group := &service.Group{ID: 9300, Hydrated: true, Platform: service.PlatformAnthropic, Status: service.StatusActive, RateMultiplier: 1}
				account := &service.Account{ID: 9301, Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Credentials: map[string]any{"api_key": "test-key"}}
				stream := mode == "partial"
				upstreamBody := `{"id":"m","type":"message","role":"assistant","model":"claude-sonnet-4.5","content":[{"type":"text","text":"hello"}],"stop_reason":"end_turn","usage":{"input_tokens":12,"output_tokens":7}}`
				if stream {
					upstreamBody = compatStreamUsagePartialStream
				} else if endpoint != "messages" {
					upstreamBody = compatStreamUsagePartialStream + compatStreamUsageCompletion
				}
				upstream := &compatStreamUsageUpstream{response: &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(upstreamBody))}}
				env := newTerminalGatewayMessagesEnv(t, group, upstream, account)
				fn := env.handler.Messages
				if endpoint == "chat/completions" {
					fn = env.handler.ChatCompletions
				}
				if endpoint == "responses" {
					fn = env.handler.Responses
				}
				router := env.routerFor("/v1/"+endpoint, fn)
				body := `{"model":"claude-sonnet-4.5","messages":[{"role":"user","content":"hi"}]}`
				if endpoint == "responses" {
					body = `{"model":"claude-sonnet-4.5","input":"hi"}`
				}
				if stream {
					body = strings.TrimSuffix(body, "}") + `,"stream":true}`
				}
				req := httptest.NewRequest("POST", "/v1/"+endpoint, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				n := int64(12_000_000)
				req = req.WithContext(service.WithTrafficBillingSnapshot(context.Background(), service.TrafficBillingSnapshot{Enabled: true, Consent: true, Successful: true, InputBytes: &n, ThresholdBytes: 10_000_000, PricePerMB: 0.01}))
				rec := httptest.NewRecorder()
				var writer http.ResponseWriter = rec
				if mode == "write_failure" || mode == "short_write" {
					writer = &trafficFailedHTTPWriter{ResponseRecorder: rec, short: mode == "short_write"}
				}
				router.ServeHTTP(writer, req)
				require.Equal(t, 1, upstream.calls, "response: %s", rec.Body.String())
				select {
				case usage := <-env.usageRepo.created:
					require.Positive(t, usage.InputCost, "model usage must survive delivery failures")
					if mode == "success" {
						require.InDelta(t, 0.24, usage.TrafficCost, 1e-12)
					} else {
						require.Zero(t, usage.TrafficCost)
						require.Equal(t, "delivery_incomplete", usage.TrafficBillingReason)
					}
				default:
					t.Fatalf("no model bill produced: %s", rec.Body.String())
				}
			})
		}
	}
}
