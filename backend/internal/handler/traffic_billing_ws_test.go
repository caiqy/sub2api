//go:build unit

package handler

import (
	"context"
	"errors"
	"github.com/coder/websocket/wsjson"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

func TestTrafficBillingWSConsentErrorUsesProtocolPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		writeTrafficBillingWSError(r.Context(), conn, service.TrafficBillingSnapshot{ThresholdBytes: 100, PricePerMB: 0.01})
		_ = conn.Close(coderws.StatusPolicyViolation, "traffic_billing_consent_required")
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	defer conn.CloseNow()
	var event struct {
		Type  string `json:"type"`
		Error struct {
			Code      string  `json:"code"`
			Threshold int64   `json:"threshold_bytes"`
			Price     float64 `json:"price_per_mb"`
		} `json:"error"`
	}
	require.NoError(t, wsjson.Read(ctx, conn, &event))
	require.Equal(t, "error", event.Type)
	require.Equal(t, "traffic_billing_consent_required", event.Error.Code)
	require.Equal(t, int64(100), event.Error.Threshold)
	require.Equal(t, 0.01, event.Error.Price)
}

func TestResponsesWSTrafficRequiresSuccessfulCompleteDelivery(t *testing.T) {
	for _, event := range []string{"response.completed", "response.done", "response.failed", "response.incomplete", "response.cancelled", "response.canceled", "error"} {
		t.Run(event, func(t *testing.T) {
			result := &service.OpenAIForwardResult{OpenAIWSMode: true, UpstreamTerminalEvent: event, ClientDeliveryComplete: true}
			require.Equal(t, event == "response.completed" || event == "response.done", responsesWSTrafficSuccessful(result, nil))
			result.ClientDeliveryComplete = false
			require.False(t, responsesWSTrafficSuccessful(result, nil), "drain cannot qualify even for upstream completed")
			result.ClientDeliveryComplete = true
			result.ImageCount = 1
			require.False(t, responsesWSTrafficSuccessful(result, errors.New("partial image failure")))
		})
	}
	require.False(t, responsesWSTrafficSuccessful(nil, nil))
}

func TestGrokRealtimeTrafficRequiresNormalCloseAndAudioBill(t *testing.T) {
	result := &service.OpenAIForwardResult{AudioUsage: &service.AudioUsage{Mode: "realtime", DurationOrUnits: 1}}
	for _, code := range []coderws.StatusCode{1000, 1001, 1005, 1006, 1008, 1011} {
		require.Equal(t, code == 1000, grokRealtimeTrafficSuccessful(result, coderws.CloseError{Code: code}))
	}
	require.False(t, grokRealtimeTrafficSuccessful(result, nil))
	require.False(t, grokRealtimeTrafficSuccessful(result, errors.New("relay failed")))
	require.False(t, grokRealtimeTrafficSuccessful(&service.OpenAIForwardResult{}, coderws.CloseError{Code: 1000}))
	require.False(t, grokRealtimeTrafficSuccessful(nil, coderws.CloseError{Code: 1000}))
}
