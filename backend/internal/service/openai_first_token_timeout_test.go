package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	openaiutil "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

func TestOpenAIFirstTokenTimeoutWatchdogTimesOutWithDiagnostics(t *testing.T) {
	ctx, watchdog := newOpenAIFirstTokenWatchdog(
		context.Background(),
		openaiutil.FirstTokenClassText,
		20*time.Millisecond,
		"sse",
	)
	require.NotNil(t, watchdog)
	watchdog.MarkHeaders("req_123")
	watchdog.Observe([]byte(`{"type":"response.created"}`))

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("watchdog 未在 deadline 后取消 context")
	}

	require.ErrorIs(t, context.Cause(ctx), errOpenAIFirstTokenTimeout)
	timeoutErr := watchdog.TimeoutError()
	require.NotNil(t, timeoutErr)
	require.Equal(t, openaiutil.FirstTokenClassText, timeoutErr.Class)
	require.Equal(t, 20*time.Millisecond, timeoutErr.Timeout)
	require.Equal(t, "sse", timeoutErr.Transport)
	require.True(t, timeoutErr.HeadersReceived)
	require.True(t, timeoutErr.CreatedReceived)
	require.Equal(t, "req_123", timeoutErr.UpstreamRequestID)
	require.GreaterOrEqual(t, timeoutErr.Elapsed, 20*time.Millisecond)
	require.False(t, watchdog.Observe([]byte(`{"type":"response.output_text.delta","delta":"late"}`)))
	require.False(t, watchdog.MarkHeaders("req_late"))
	require.False(t, watchdog.Stop())

	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(timeoutErr, &failoverErr))
}

func TestOpenAIFirstTokenTimeoutWatchdogStopsOnBusinessEvent(t *testing.T) {
	ctx, watchdog := newOpenAIFirstTokenWatchdog(
		context.Background(),
		openaiutil.FirstTokenClassText,
		20*time.Millisecond,
		"sse",
	)
	watchdog.Observe([]byte(`{"type":"response.output_text.delta","delta":"x"}`))

	select {
	case <-ctx.Done():
		t.Fatalf("业务事件后不应超时: %v", context.Cause(ctx))
	case <-time.After(40 * time.Millisecond):
	}
	require.Nil(t, watchdog.TimeoutError())
}

func TestWithOpenAIFirstTokenTimeoutZeroDisablesWatchdog(t *testing.T) {
	svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{}}}
	parent := context.Background()

	ctx, watchdog := svc.withOpenAIFirstTokenTimeout(parent, []byte(`{"stream":true}`), "sse")

	require.Equal(t, parent, ctx)
	require.Nil(t, watchdog)
}

func TestWithOpenAIFirstTokenTimeoutSelectsImageDuration(t *testing.T) {
	svc := &OpenAIGatewayService{cfg: &config.Config{Gateway: config.GatewayConfig{
		OpenAITextFirstTokenTimeout:  30,
		OpenAIImageFirstTokenTimeout: 600,
	}}}

	_, watchdog := svc.withOpenAIFirstTokenTimeout(
		context.Background(),
		[]byte(`{"tool_choice":{"type":"image_generation"}}`),
		"websocket",
	)
	t.Cleanup(func() { watchdog.Stop() })

	require.Equal(t, openaiutil.FirstTokenClassImage, watchdog.class)
	require.Equal(t, 600*time.Second, watchdog.timeout)
}
