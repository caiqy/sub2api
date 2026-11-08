package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

func TestGrokRealtimeTrafficCountsControlPayloadBeforeForwarding(t *testing.T) {
	control := []byte(` {"type":"session.update"} `)
	audio := []byte(`{"type":"input_audio_buffer.append","audio":"abc"}`)
	upstream := &grokVoiceRealtimeGuardTestConn{readExited: make(chan struct{})}
	svc := &OpenAIGatewayService{}
	type outcome struct {
		bytes int64
		audio bool
		err   error
	}
	results := make(chan outcome, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			results <- outcome{err: err}
			return
		}
		defer conn.CloseNow()
		var total int64
		snapshot := TrafficBillingSnapshot{Enabled: true, ThresholdBytes: int64(len(control)), PricePerMB: 0.01}
		observed, err := svc.ProxyGrokRealtimeConn(r.Context(), nil, conn, &GrokRealtimeUpstream{conn: upstream}, func(_ context.Context, payload []byte) error {
			total += int64(len(payload))
			snapshot.InputBytes = &total
			if snapshot.ConsentRequired() {
				return &TrafficBillingConsentError{Snapshot: snapshot}
			}
			return nil
		})
		results <- outcome{bytes: total, audio: observed, err: err}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	defer conn.CloseNow()
	require.NoError(t, conn.Write(ctx, coderws.MessageText, control))
	require.NoError(t, conn.Write(ctx, coderws.MessageText, audio))
	select {
	case got := <-results:
		var consentErr *TrafficBillingConsentError
		require.True(t, errors.As(got.err, &consentErr))
		require.Equal(t, int64(len(control)+len(audio)), got.bytes)
		require.Equal(t, 1, upstream.writes)
		require.False(t, got.audio, "rejected audio must not manufacture an audio model bill")
	case <-ctx.Done():
		t.Fatal("realtime relay failed to terminate")
	}
}
