package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	openaiutil "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/tidwall/gjson"
)

var errOpenAIFirstTokenTimeout = errors.New("OpenAI first token timeout")

type OpenAIFirstTokenTimeoutError struct {
	Class             openaiutil.FirstTokenClass
	Timeout           time.Duration
	Elapsed           time.Duration
	Transport         string
	HeadersReceived   bool
	CreatedReceived   bool
	UpstreamRequestID string
}

func (e *OpenAIFirstTokenTimeoutError) Error() string {
	return fmt.Sprintf("OpenAI %s stream timed out before first output after %s", e.Class, e.Timeout)
}

type openAIFirstTokenWatchdog struct {
	mu                sync.Mutex
	cancel            context.CancelCauseFunc
	timer             *time.Timer
	started           time.Time
	class             openaiutil.FirstTokenClass
	timeout           time.Duration
	transport         string
	stopped           bool
	timedOut          bool
	elapsed           time.Duration
	headersReceived   bool
	createdReceived   bool
	upstreamRequestID string
}

func newOpenAIFirstTokenWatchdog(
	ctx context.Context,
	class openaiutil.FirstTokenClass,
	timeout time.Duration,
	transport string,
) (context.Context, *openAIFirstTokenWatchdog) {
	if timeout <= 0 {
		return ctx, nil
	}
	timedCtx, cancel := context.WithCancelCause(ctx)
	watchdog := &openAIFirstTokenWatchdog{
		cancel:    cancel,
		started:   time.Now(),
		class:     class,
		timeout:   timeout,
		transport: transport,
	}
	watchdog.timer = time.AfterFunc(timeout, func() {
		watchdog.mu.Lock()
		if watchdog.stopped {
			watchdog.mu.Unlock()
			return
		}
		watchdog.stopped = true
		watchdog.timedOut = true
		watchdog.elapsed = time.Since(watchdog.started)
		watchdog.mu.Unlock()
		cancel(errOpenAIFirstTokenTimeout)
	})
	return timedCtx, watchdog
}

func (s *OpenAIGatewayService) withOpenAIFirstTokenTimeout(
	ctx context.Context,
	payload []byte,
	transport string,
) (context.Context, *openAIFirstTokenWatchdog) {
	class := openaiutil.ResponsesFirstTokenClass(payload)
	seconds := 0
	if s != nil && s.cfg != nil {
		seconds = s.cfg.Gateway.OpenAITextFirstTokenTimeout
		if class == openaiutil.FirstTokenClassImage {
			seconds = s.cfg.Gateway.OpenAIImageFirstTokenTimeout
		}
	}
	return newOpenAIFirstTokenWatchdog(ctx, class, time.Duration(seconds)*time.Second, transport)
}

func (w *openAIFirstTokenWatchdog) MarkHeaders(requestID string) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.headersReceived = true
	w.upstreamRequestID = requestID
}

func (w *openAIFirstTokenWatchdog) Observe(payload []byte) {
	if w == nil {
		return
	}
	if gjson.GetBytes(payload, "type").String() == "response.created" {
		w.mu.Lock()
		w.createdReceived = true
		w.mu.Unlock()
	}
	if openaiutil.ResponsesEventEndsFirstTokenWait(payload) {
		w.Stop()
	}
}

func (w *openAIFirstTokenWatchdog) Stop() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return
	}
	w.stopped = true
	w.timer.Stop()
}

func (w *openAIFirstTokenWatchdog) TimeoutError() *OpenAIFirstTokenTimeoutError {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.timedOut {
		return nil
	}
	return &OpenAIFirstTokenTimeoutError{
		Class:             w.class,
		Timeout:           w.timeout,
		Elapsed:           w.elapsed,
		Transport:         w.transport,
		HeadersReceived:   w.headersReceived,
		CreatedReceived:   w.createdReceived,
		UpstreamRequestID: w.upstreamRequestID,
	}
}
