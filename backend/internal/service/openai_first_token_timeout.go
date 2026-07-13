package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	openaiutil "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

var errOpenAIFirstTokenTimeout = errors.New("OpenAI first token timeout")

type openAIFirstTokenWatchdogContextKey struct{}

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
	return context.WithValue(timedCtx, openAIFirstTokenWatchdogContextKey{}, watchdog), watchdog
}

func firstTokenWatchdogFromContext(ctx context.Context) *openAIFirstTokenWatchdog {
	if ctx == nil {
		return nil
	}
	watchdog, _ := ctx.Value(openAIFirstTokenWatchdogContextKey{}).(*openAIFirstTokenWatchdog)
	return watchdog
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

func recordOpenAIFirstTokenTimeout(ctx context.Context, c *gin.Context, account *Account, timeoutErr *OpenAIFirstTokenTimeoutError) {
	if timeoutErr == nil {
		return
	}
	accountID := int64(0)
	accountName := ""
	platform := ""
	if account != nil {
		accountID = account.ID
		accountName = account.Name
		platform = account.Platform
	}
	detail := "class=" + string(timeoutErr.Class) +
		" timeout_ms=" + strconv.FormatInt(timeoutErr.Timeout.Milliseconds(), 10) +
		" elapsed_ms=" + strconv.FormatInt(timeoutErr.Elapsed.Milliseconds(), 10) +
		" transport=" + timeoutErr.Transport +
		" headers_received=" + strconv.FormatBool(timeoutErr.HeadersReceived) +
		" created_received=" + strconv.FormatBool(timeoutErr.CreatedReceived)
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:          platform,
		AccountID:         accountID,
		AccountName:       accountName,
		UpstreamRequestID: timeoutErr.UpstreamRequestID,
		Kind:              "first_token_timeout",
		Message:           timeoutErr.Error(),
		Detail:            detail,
	})
	logger.FromContext(ctx).Warn("gateway.openai_first_token_timeout",
		zap.Int64("account_id", accountID),
		zap.String("class", string(timeoutErr.Class)),
		zap.Duration("timeout", timeoutErr.Timeout),
		zap.Duration("elapsed", timeoutErr.Elapsed),
		zap.String("transport", timeoutErr.Transport),
		zap.Bool("headers_received", timeoutErr.HeadersReceived),
		zap.Bool("created_received", timeoutErr.CreatedReceived),
		zap.String("upstream_request_id", timeoutErr.UpstreamRequestID),
	)
}
