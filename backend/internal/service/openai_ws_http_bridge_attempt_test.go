package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestProxyResponsesWebSocketFromClient_HTTPBridgeResetsAttemptBeforeFollowupLocalFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_first\"}}\n\n",
		)),
	}}
	cfg := &config.Config{Gateway: config.GatewayConfig{OpenAIWS: config.GatewayOpenAIWSConfig{
		Enabled: true, APIKeyEnabled: true, ResponsesWebsocketsV2: true, ModeRouterV2Enabled: true,
		HTTPBridgeEnabled: true, HTTPBridgeThresholdBytes: 1,
	}}}
	svc := &OpenAIGatewayService{
		cfg:          cfg,
		httpUpstream: upstream,
	}
	account := &Account{ID: 99, Name: "bridge", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Concurrency: 1, Credentials: map[string]any{"api_key": "sk-test"}, Extra: map[string]any{"openai_apikey_responses_websockets_v2_mode": OpenAIWSIngressModeHTTPBridge}}
	resetBeforeFailure := make(chan bool, 1)
	serverErr := make(chan error, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			serverErr <- err
			return
		}
		defer func() { _ = conn.CloseNow() }()
		_, first, err := conn.Read(r.Context())
		if err != nil {
			serverErr <- err
			return
		}
		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		ginCtx.Request = r.Clone(r.Context())
		hooks := &OpenAIWSIngressHooks{
			AfterTurn: func(turn int, _ *OpenAIForwardResult, _ error) {
				if turn == 1 {
					SetOpsUpstreamAttempted(ginCtx, true)
				}
			},
			BeforeTurn: func(turn int) error {
				if turn == 2 {
					resetBeforeFailure <- !HasOpsUpstreamAttempted(ginCtx)
					return errors.New("local followup build failure")
				}
				return nil
			},
		}
		serverErr <- svc.ProxyResponsesWebSocketFromClient(r.Context(), ginCtx, conn, account, "sk-test", first, hooks)
	}))
	defer server.Close()

	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	conn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	cancel()
	require.NoError(t, err)
	defer func() { _ = conn.CloseNow() }()
	writeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err = conn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5","stream":true,"input":"one"}`))
	cancel()
	require.NoError(t, err)
	readCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	_, _, err = conn.Read(readCtx)
	cancel()
	if err != nil {
		select {
		case serverFailure := <-serverErr:
			require.NoError(t, serverFailure)
		default:
			require.NoError(t, err)
		}
	}
	writeCtx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	err = conn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5","stream":true,"input":"two"}`))
	cancel()
	require.NoError(t, err)

	select {
	case reset := <-resetBeforeFailure:
		require.True(t, reset)
	case <-time.After(3 * time.Second):
		t.Fatal("followup turn did not reach local failure")
	}
	select {
	case err := <-serverErr:
		require.ErrorContains(t, err, "local followup build failure")
	case <-time.After(3 * time.Second):
		t.Fatal("bridge did not return after local failure")
	}
}
