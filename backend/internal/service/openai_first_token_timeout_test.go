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

type openAIWSFirstTokenLeaseStub struct {
	writeErr error
	readErr  error
	events   [][]byte
	broken   bool
}

func (s *openAIWSFirstTokenLeaseStub) WriteJSONWithContextTimeout(context.Context, any, time.Duration) error {
	return s.writeErr
}

func (s *openAIWSFirstTokenLeaseStub) ReadMessageWithContextTimeout(context.Context, time.Duration) ([]byte, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}
	if len(s.events) == 0 {
		return nil, context.DeadlineExceeded
	}
	event := s.events[0]
	s.events = s.events[1:]
	return event, nil
}

func (s *openAIWSFirstTokenLeaseStub) MarkBroken() { s.broken = true }

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

func TestCancelAndDrainOpenAIWSFirstToken(t *testing.T) {
	t.Run("terminal event keeps connection reusable", func(t *testing.T) {
		lease := &openAIWSFirstTokenLeaseStub{events: [][]byte{[]byte(`{"type":"response.canceled","response":{"id":"resp_1"}}`)}}
		require.True(t, cancelAndDrainOpenAIWSFirstToken(context.Background(), lease, "resp_1", time.Second, time.Second))
		require.False(t, lease.broken)
	})

	t.Run("terminal event for another response breaks connection", func(t *testing.T) {
		lease := &openAIWSFirstTokenLeaseStub{events: [][]byte{[]byte(`{"type":"response.canceled","response":{"id":"resp_other"}}`)}}
		require.False(t, cancelAndDrainOpenAIWSFirstToken(context.Background(), lease, "resp_1", time.Second, time.Millisecond))
		require.True(t, lease.broken)
	})

	t.Run("cancel write failure marks connection broken", func(t *testing.T) {
		lease := &openAIWSFirstTokenLeaseStub{writeErr: errors.New("write failed")}
		require.False(t, cancelAndDrainOpenAIWSFirstToken(context.Background(), lease, "resp_1", time.Second, time.Millisecond))
		require.True(t, lease.broken)
	})

	t.Run("drain failure marks connection broken", func(t *testing.T) {
		lease := &openAIWSFirstTokenLeaseStub{readErr: context.DeadlineExceeded}
		require.False(t, cancelAndDrainOpenAIWSFirstToken(context.Background(), lease, "resp_1", time.Second, time.Millisecond))
		require.True(t, lease.broken)
	})
}
