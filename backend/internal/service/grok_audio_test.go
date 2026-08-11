package service

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBuildGrokVoiceURL_UsesAPIDefaultForCLIProxyBase(t *testing.T) {
	account := &Account{
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"base_url": xai.DefaultCLIBaseURL,
		},
	}
	url, err := buildGrokVoiceURL(account, nil, "tts")
	require.NoError(t, err)
	require.Equal(t, xai.DefaultBaseURL+"/tts", url)

	url, err = buildGrokVoiceURL(account, nil, "realtime")
	require.NoError(t, err)
	require.Equal(t, xai.DefaultBaseURL+"/realtime", url)
}

func TestBuildGrokVoiceURL_EmptyBaseFallsBackToAPI(t *testing.T) {
	account := &Account{
		Platform:    PlatformGrok,
		Type:        AccountTypeOAuth,
		Credentials: map[string]any{},
	}
	url, err := buildGrokVoiceURL(account, nil, "stt")
	require.NoError(t, err)
	require.Equal(t, xai.DefaultBaseURL+"/stt", url)
}

func TestBuildGrokVoiceURL_RequiresEndpoint(t *testing.T) {
	account := &Account{Platform: PlatformGrok, Type: AccountTypeOAuth}
	_, err := buildGrokVoiceURL(account, nil, "  ")
	require.Error(t, err)
}

func TestBuildGrokVoiceURL_EncodesCustomVoicePathSegments(t *testing.T) {
	account := &Account{Platform: PlatformGrok, Type: AccountTypeOAuth}
	got, err := buildGrokVoiceURL(account, nil, "custom-voices/nlbqfwie/audio")
	require.NoError(t, err)
	require.Equal(t, xai.DefaultBaseURL+"/custom-voices/nlbqfwie/audio", got)

	_, err = buildGrokVoiceURL(account, nil, "custom-voices/../audio")
	require.Error(t, err)
}

func TestForwardGrokVoice_RejectsNonGrok(t *testing.T) {
	svc := &OpenAIGatewayService{}
	_, err := svc.ForwardGrokVoice(context.Background(), nil, &Account{Platform: PlatformOpenAI}, "tts", []byte(`{}`), "application/json")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
}

func TestForwardGrokVoice_RejectsUnknownEndpoint(t *testing.T) {
	svc := &OpenAIGatewayService{}
	_, err := svc.ForwardGrokVoice(context.Background(), nil, &Account{Platform: PlatformGrok}, "unknown", []byte(`{}`), "application/json")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported")
}

type grokVoiceRealtimeTestConn struct{}

func (*grokVoiceRealtimeTestConn) WriteJSON(context.Context, any) error { return nil }
func (*grokVoiceRealtimeTestConn) ReadMessage(context.Context) ([]byte, error) {
	return nil, context.DeadlineExceeded
}
func (*grokVoiceRealtimeTestConn) Ping(context.Context) error { return nil }
func (*grokVoiceRealtimeTestConn) Close() error               { return nil }

type grokVoiceRealtimeTestDialer struct {
	lastURL string
}

func (d *grokVoiceRealtimeTestDialer) Dial(_ context.Context, wsURL string, _ http.Header, _ string) (openAIWSClientConn, int, http.Header, error) {
	d.lastURL = wsURL
	return &grokVoiceRealtimeTestConn{}, 0, nil, nil
}

type grokVoiceRealtimeProxyResult struct {
	model string
	err   error
}

type grokVoiceRealtimeGuardTestConn struct {
	writes int
}

func (c *grokVoiceRealtimeGuardTestConn) WriteJSON(context.Context, any) error {
	c.writes++
	return nil
}

func (*grokVoiceRealtimeGuardTestConn) ReadMessage(ctx context.Context) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (*grokVoiceRealtimeGuardTestConn) Ping(context.Context) error { return nil }
func (*grokVoiceRealtimeGuardTestConn) Close() error               { return nil }

type grokVoiceRealtimeGuardTestDialer struct {
	conn *grokVoiceRealtimeGuardTestConn
}

func (d *grokVoiceRealtimeGuardTestDialer) Dial(context.Context, string, http.Header, string) (openAIWSClientConn, int, http.Header, error) {
	return d.conn, 0, nil, nil
}

func TestProxyGrokRealtimeUsesMappedModelInUpstreamURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dialer := &grokVoiceRealtimeTestDialer{}
	svc := &OpenAIGatewayService{openaiWSPassthroughDialer: dialer}
	account := &Account{
		Platform: PlatformGrok,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":       "voice-key",
			"model_mapping": map[string]any{"grok-voice-latest": "first-realtime", "first-realtime": "second-realtime"},
		},
	}
	result := make(chan grokVoiceRealtimeProxyResult, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client, err := coderws.Accept(w, r, nil)
		if err != nil {
			result <- grokVoiceRealtimeProxyResult{err: err}
			return
		}
		defer func() { _ = client.CloseNow() }()
		model, proxyErr := svc.ProxyGrokRealtime(r.Context(), nil, client, account, "token", "grok-voice-latest", nil)
		result <- grokVoiceRealtimeProxyResult{model: model, err: proxyErr}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	defer func() { _ = client.CloseNow() }()
	_, _, _ = client.Read(ctx)
	returned := <-result

	require.Error(t, returned.err)
	require.Equal(t, "first-realtime", returned.model)
	require.Contains(t, dialer.lastURL, "model=first-realtime")
}

func TestEstimateGrokVoiceAudioUsage_STTPrefersAudioEvidenceOverElapsed(t *testing.T) {
	got := estimateGrokVoiceAudioUsage("stt", bytes.Repeat([]byte("a"), 160000), "audio/wav", nil, 500*time.Millisecond)
	require.InDelta(t, 10.0/3600.0, got.DurationOrUnits, 1e-9)
}

func TestProxyGrokRealtimeGuardBlocksBeforeUpstreamWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sentinel := errors.New("blocked realtime event")
	upstream := &grokVoiceRealtimeGuardTestConn{}
	svc := &OpenAIGatewayService{openaiWSPassthroughDialer: &grokVoiceRealtimeGuardTestDialer{conn: upstream}}
	account := &Account{Platform: PlatformGrok, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "voice-key"}}
	result := make(chan grokVoiceRealtimeProxyResult, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client, err := coderws.Accept(w, r, nil)
		if err != nil {
			result <- grokVoiceRealtimeProxyResult{err: err}
			return
		}
		defer func() { _ = client.CloseNow() }()
		model, proxyErr := svc.ProxyGrokRealtime(r.Context(), nil, client, account, "token", "grok-voice-latest", func([]byte) error { return sentinel })
		result <- grokVoiceRealtimeProxyResult{model: model, err: proxyErr}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	client, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	defer func() { _ = client.CloseNow() }()
	require.NoError(t, client.Write(ctx, coderws.MessageText, []byte(`{"type":"session.update","session":{"instructions":"blocked"}}`)))

	returned := <-result
	require.ErrorIs(t, returned.err, sentinel)
	require.Zero(t, upstream.writes)
}
