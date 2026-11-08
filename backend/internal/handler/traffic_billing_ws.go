package handler

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type wsTrafficTurn struct {
	snapshot service.TrafficBillingSnapshot
	err      error
}

func responsesWSTrafficSuccessful(result *service.OpenAIForwardResult, turnErr error) bool {
	return turnErr == nil && result != nil && result.ClientDeliveryComplete && result.SucceededForScheduling()
}

func grokRealtimeTrafficSuccessful(result *service.OpenAIForwardResult, proxyErr error) bool {
	return result != nil && result.AudioUsage != nil && coderws.CloseStatus(proxyErr) == coderws.StatusNormalClosure
}

type wsTrafficAdmissionError struct{ err error }

func (e *wsTrafficAdmissionError) Error() string { return e.err.Error() }
func (e *wsTrafficAdmissionError) Unwrap() error { return e.err }

func writeTrafficBillingWSError(ctx context.Context, conn *coderws.Conn, snapshot service.TrafficBillingSnapshot) {
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_ = wsjson.Write(writeCtx, conn, map[string]any{"type": "error", "error": map[string]any{"type": "invalid_request_error", "code": "traffic_billing_consent_required", "message": snapshot.ConsentMessage(), "threshold_bytes": snapshot.ThresholdBytes, "price_per_mb": snapshot.PricePerMB}})
}
