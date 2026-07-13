package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	openaiutil "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type firstTokenBlockingUpstream struct{}

func (firstTokenBlockingUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-time.After(1500 * time.Millisecond):
		return nil, errors.New("test upstream fallback timeout")
	}
}

func (u firstTokenBlockingUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, concurrency)
}

type firstTokenScriptedBody struct {
	steps []firstTokenBodyStep
	index int
}

type firstTokenBodyStep struct {
	delay time.Duration
	data  string
}

func (b *firstTokenScriptedBody) Read(p []byte) (int, error) {
	if b.index >= len(b.steps) {
		return 0, io.EOF
	}
	step := b.steps[b.index]
	b.index++
	if step.delay > 0 {
		time.Sleep(step.delay)
	}
	return copy(p, step.data), nil
}

func (*firstTokenScriptedBody) Close() error { return nil }

type firstTokenContextBody struct {
	ctx     context.Context
	prefix  []byte
	emitted bool
}

func (b *firstTokenContextBody) Read(p []byte) (int, error) {
	if !b.emitted {
		b.emitted = true
		return copy(p, b.prefix), nil
	}
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (*firstTokenContextBody) Close() error { return nil }

func newFirstTokenSSEGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(nil))
	return c, recorder
}

func TestOpenAIFirstTokenTimeoutBeforeResponseHeaders(t *testing.T) {
	c, _ := newFirstTokenSSEGinContext()
	body := []byte(`{"model":"gpt-5","stream":true,"input":"hello"}`)
	svc := &OpenAIGatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}},
			Gateway:  config.GatewayConfig{OpenAITextFirstTokenTimeout: 1},
		},
		httpUpstream: firstTokenBlockingUpstream{},
	}
	account := &Account{
		ID:          1,
		Name:        "timeout-account",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "test-key", "base_url": "https://example.com/v1"},
	}

	_, err := svc.forwardOpenAIPassthrough(context.Background(), c, account, body, "gpt-5", nil, true, time.Now())

	var timeoutErr *OpenAIFirstTokenTimeoutError
	require.ErrorAs(t, err, &timeoutErr)
	require.Equal(t, openaiutil.FirstTokenClassText, timeoutErr.Class)
	require.False(t, timeoutErr.HeadersReceived)
}

func TestOpenAIFirstTokenTimeoutAfterCreatedEvent(t *testing.T) {
	c, _ := newFirstTokenSSEGinContext()
	ctx, watchdog := newOpenAIFirstTokenWatchdog(context.Background(), openaiutil.FirstTokenClassText, 20*time.Millisecond, "sse")
	defer watchdog.Stop()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"req_created"}},
		Body: &firstTokenContextBody{
			ctx:    ctx,
			prefix: []byte("data: {\"type\":\"response.created\"}\n\n"),
		},
	}

	_, err := (&OpenAIGatewayService{}).handleStreamingResponsePassthrough(ctx, resp, c, &Account{ID: 1}, time.Now(), "", "")

	var timeoutErr *OpenAIFirstTokenTimeoutError
	require.ErrorAs(t, err, &timeoutErr)
	require.True(t, timeoutErr.CreatedReceived)
}

func TestOpenAIFirstTokenImageOutputItemStopsWatchdog(t *testing.T) {
	c, _ := newFirstTokenSSEGinContext()
	ctx, watchdog := newOpenAIFirstTokenWatchdog(context.Background(), openaiutil.FirstTokenClassImage, 20*time.Millisecond, "sse")
	defer watchdog.Stop()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Request-Id": []string{"req_image"}},
		Body: &firstTokenScriptedBody{steps: []firstTokenBodyStep{
			{data: "data: {\"type\":\"response.output_item.added\",\"item\":{\"type\":\"image_generation_call\"}}\n\n"},
			{delay: 40 * time.Millisecond, data: "data: {\"type\":\"response.completed\"}\n\n"},
		}},
	}

	_, err := (&OpenAIGatewayService{}).handleStreamingResponsePassthrough(ctx, resp, c, &Account{ID: 1}, time.Now(), "", "")

	require.NoError(t, err)
	require.NoError(t, context.Cause(ctx))
}
