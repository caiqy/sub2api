package service

// Anthropic APIKey 直通（passthrough）转发路径及其流式/非流式响应与 usage 解析。

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tokencount"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/gin-gonic/gin"
)

type anthropicPassthroughForwardInput struct {
	SourceHandle      *RequestBodyHandle
	BodyHandle        *RequestBodyHandle
	OwnBodyHandle     bool
	SourceBody        []byte
	Body              []byte
	Parsed            *ParsedRequest
	SourceHandleOwned bool
	RequestModel      string
	OriginalModel     string
	RequestStream     bool
	StartTime         time.Time
}

func applyOutputTokenEstimation(usage *ClaudeUsage, accumulatedText string) {
	if usage == nil || usage.OutputTokens > 0 || accumulatedText == "" {
		return
	}
	if estimated := tokencount.CountTokens(accumulatedText); estimated > 0 {
		usage.OutputTokens = estimated
		logger.LegacyPrintf("service.gateway", "output_tokens fallback estimation applied: estimated=%d text_len=%d", estimated, len(accumulatedText))
	}
}

func (s *GatewayService) forwardAnthropicAPIKeyPassthrough(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	reqModel string,
	originalModel string,
	reqStream bool,
	startTime time.Time,
) (*ForwardResult, error) {
	handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{})
	if err != nil {
		return nil, fmt.Errorf("create anthropic passthrough body handle: %w", err)
	}
	defer CleanupRequestBodyHandle(handle)
	return s.forwardAnthropicAPIKeyPassthroughWithInput(ctx, c, account, anthropicPassthroughForwardInput{
		SourceHandle:  handle,
		BodyHandle:    handle,
		SourceBody:    body,
		Body:          body,
		RequestModel:  reqModel,
		OriginalModel: originalModel,
		RequestStream: reqStream,
		StartTime:     startTime,
	})
}

func (s *GatewayService) forwardAnthropicAPIKeyPassthroughWithInput(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	input anthropicPassthroughForwardInput,
) (*ForwardResult, error) {
	if input.SourceHandle == nil || input.BodyHandle == nil {
		if input.Body == nil {
			return nil, errors.New("anthropic passthrough requires source and effective body handles")
		}
		handle, err := NewRequestBodyHandleFromBytes(input.Body, RequestBodyHandleOptions{})
		if err != nil {
			return nil, fmt.Errorf("create anthropic passthrough body handle: %w", err)
		}
		defer CleanupRequestBodyHandle(handle)
		input.SourceHandle, input.BodyHandle = handle, handle
		input.SourceBody = input.Body
	}
	var lastWireHandle *RequestBodyHandle
	defer func() { CleanupRequestBodyHandle(lastWireHandle) }()
	ownedBodyHandle := input.BodyHandle
	if !input.OwnBodyHandle || ownedBodyHandle == input.SourceHandle {
		ownedBodyHandle = nil
	}
	defer func() { CleanupRequestBodyHandle(ownedBodyHandle) }()
	if input.Parsed != nil && input.SourceHandleOwned {
		input.Parsed.Body.Replace(nil)
		input.Parsed.Body = NewRequestBodyRefFromHandle(input.BodyHandle)
		defer func() {
			if currentHandle := input.Parsed.Body.Handle(); currentHandle != nil {
				if restoredBody, restoreErr := currentHandle.ReadAll(); restoreErr == nil {
					_ = input.Parsed.ReplaceBody(restoredBody)
				}
			}
		}()
	}
	sourceBody := input.SourceBody
	if sourceBody == nil {
		var err error
		sourceBody, err = input.SourceHandle.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("read anthropic passthrough source body: %w", err)
		}
	}
	body := input.Body
	if body == nil {
		if input.BodyHandle == input.SourceHandle {
			body = sourceBody
		} else {
			var err error
			body, err = input.BodyHandle.ReadAll()
			if err != nil {
				return nil, fmt.Errorf("read anthropic passthrough effective body: %w", err)
			}
		}
	}
	token, tokenType, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	if tokenType != "apikey" {
		return nil, fmt.Errorf("anthropic api key passthrough requires apikey token, got: %s", tokenType)
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	logger.LegacyPrintf("service.gateway", "[Anthropic 自动透传] 命中 API Key 透传分支: account=%d name=%s model=%s stream=%v",
		account.ID, account.Name, input.RequestModel, input.RequestStream)

	if c != nil {
		c.Set("anthropic_passthrough", true)
		SetOpsUpstreamRequestBodyPreview(c, input.BodyHandle.PreviewString(), input.BodyHandle.Size())
	}
	// Pre-filter: strip empty text blocks (including nested in tool_result) to prevent upstream 400.
	effectiveBody := StripEmptyTextBlocks(body)
	// Pre-filter: strip web-search history blocks the upstream cannot accept
	// (emulation-synthesized ones always; genuine ones additionally for
	// passback-required third-party upstreams such as GLM/Kimi/DeepSeek,
	// which reject server_tool_use with 400). input.RequestModel 已是映射后的模型 ID。
	effectiveBody = FilterWebSearchHistoryBlocks(effectiveBody, input.RequestModel)
	if !bytes.Equal(body, effectiveBody) {
		handle, err := NewRequestBodyHandleFromBytes(effectiveBody, RequestBodyHandleOptions{})
		if err != nil {
			return nil, fmt.Errorf("create filtered anthropic passthrough body handle: %w", err)
		}
		CleanupRequestBodyHandle(ownedBodyHandle)
		ownedBodyHandle = handle
		input.BodyHandle = handle
		if input.Parsed != nil && input.SourceHandleOwned {
			input.Parsed.Body = NewRequestBodyRefFromHandle(handle)
		}
	}
	firstSourceBody, firstBody := sourceBody, effectiveBody
	input.SourceBody, input.Body = nil, nil

	var resp *http.Response
	retryStart := time.Now()
	for attempt := 1; attempt <= maxRetryAttempts; attempt++ {
		attemptSourceBody, attemptBody := firstSourceBody, firstBody
		firstSourceBody, firstBody = nil, nil
		if attemptSourceBody == nil {
			attemptSourceBody, err = input.SourceHandle.ReadAll()
			if err != nil {
				return nil, fmt.Errorf("read anthropic passthrough source body: %w", err)
			}
		}
		if attemptBody == nil {
			if input.BodyHandle == input.SourceHandle {
				attemptBody = attemptSourceBody
			} else {
				attemptBody, err = input.BodyHandle.ReadAll()
				if err != nil {
					return nil, fmt.Errorf("read anthropic passthrough effective body: %w", err)
				}
			}
		}
		upstreamCtx, releaseUpstreamCtx := detachStreamUpstreamContext(ctx, input.RequestStream)
		upstreamReq, wireBody, err := s.buildUpstreamRequestAnthropicAPIKeyPassthrough(upstreamCtx, c, account, attemptSourceBody, attemptBody, token)
		releaseUpstreamCtx()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"type": "error", "error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
			return nil, err
		}
		wireHandle, err := NewRequestBodyHandleFromBytes(wireBody, RequestBodyHandleOptions{})
		if err != nil {
			return nil, fmt.Errorf("spool anthropic passthrough wire body: %w", err)
		}
		wireReader, err := wireHandle.Open()
		if err != nil {
			CleanupRequestBodyHandle(wireHandle)
			return nil, fmt.Errorf("open anthropic passthrough wire body: %w", err)
		}
		if upstreamReq.Body != nil {
			_ = upstreamReq.Body.Close()
		}
		upstreamReq.Body = wireReader
		upstreamReq.GetBody = wireHandle.Open
		upstreamReq.ContentLength = wireHandle.Size()
		SetUsageUpstreamRequest(c, upstreamReq, wireHandle.PreviewString())
		SetOpsUpstreamAttempted(c, true)
		CleanupRequestBodyHandle(lastWireHandle)
		lastWireHandle = wireHandle

		resp, err = s.httpUpstream.DoWithTLS(upstreamReq, proxyURL, account.ID, account.Concurrency, s.tlsFPProfileService.ResolveTLSProfile(account))
		if upstreamReq.Body != nil {
			_ = upstreamReq.Body.Close()
		}
		if err != nil {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			if !errors.Is(err, context.Canceled) {
				scheduleOllamaCloudUsageActivity(s.deferredService, account)
			}
			if errors.Is(err, ErrRequestBodySpool) {
				return nil, fmt.Errorf("send anthropic passthrough request: %w", err)
			}
			safeErr := sanitizeUpstreamErrorMessage(err.Error())
			setOpsUpstreamError(c, 0, safeErr, "")
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: 0,
				UpstreamURL:        safeUpstreamURL(upstreamReq.URL.String()),
				Passthrough:        true,
				Kind:               "request_error",
				Message:            safeErr,
			})
			c.JSON(http.StatusBadGateway, gin.H{
				"type": "error",
				"error": gin.H{
					"type":    "upstream_error",
					"message": "Upstream request failed",
				},
			})
			return nil, fmt.Errorf("upstream request failed: %s", safeErr)
		}

		// 透传分支禁止 400 请求体降级重试（该重试会改写请求体）
		if resp.StatusCode >= 400 && resp.StatusCode != 400 && s.shouldRetryUpstreamError(account, resp.StatusCode) {
			if attempt < maxRetryAttempts {
				elapsed := time.Since(retryStart)
				if elapsed >= maxRetryElapsed {
					break
				}

				delay := retryBackoffDelay(attempt)
				remaining := maxRetryElapsed - elapsed
				if delay > remaining {
					delay = remaining
				}
				if delay <= 0 {
					break
				}

				respBody, _ := s.readUpstreamErrorBody(resp)
				_ = resp.Body.Close()
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
					Platform:           account.Platform,
					AccountID:          account.ID,
					AccountName:        account.Name,
					UpstreamStatusCode: resp.StatusCode,
					UpstreamRequestID:  resp.Header.Get("x-request-id"),
					UpstreamURL:        safeUpstreamURL(upstreamReq.URL.String()),
					Passthrough:        true,
					Kind:               "retry",
					Message:            extractUpstreamErrorMessage(respBody),
					Detail: func() string {
						if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
							return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
						}
						return ""
					}(),
				})
				logger.LegacyPrintf("service.gateway", "Anthropic passthrough account %d: upstream error %d, retry %d/%d after %v (elapsed=%v/%v)",
					account.ID, resp.StatusCode, attempt, maxRetryAttempts, delay, elapsed, maxRetryElapsed)
				if err := sleepWithContext(ctx, delay); err != nil {
					return nil, err
				}
				continue
			}
			break
		}

		break
	}
	if resp == nil || resp.Body == nil {
		return nil, errors.New("upstream request failed: empty response")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 && s.shouldRetryUpstreamError(account, resp.StatusCode) {
		if s.shouldFailoverUpstreamError(resp.StatusCode) {
			respBody, _ := s.readUpstreamErrorBody(resp)
			_ = resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(respBody))

			logger.LegacyPrintf("service.gateway", "[Anthropic Passthrough] Upstream error (retry exhausted, failover): Account=%d(%s) Status=%d RequestID=%s Body=%s",
				account.ID, account.Name, resp.StatusCode, resp.Header.Get("x-request-id"), truncateString(string(respBody), 1000))

			s.handleRetryExhaustedSideEffects(ctx, resp, account)
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Passthrough:        true,
				Kind:               "retry_exhausted_failover",
				Message:            extractUpstreamErrorMessage(respBody),
				Detail: func() string {
					if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
						return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
					}
					return ""
				}(),
			})
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
			}
		}
		return s.handleRetryExhaustedError(ctx, resp, c, account)
	}

	if resp.StatusCode >= 400 && s.shouldFailoverUpstreamError(resp.StatusCode) {
		respBody, _ := s.readUpstreamErrorBody(resp)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))

		logger.LegacyPrintf("service.gateway", "[Anthropic Passthrough] Upstream error (failover): Account=%d(%s) Status=%d RequestID=%s Body=%s",
			account.ID, account.Name, resp.StatusCode, resp.Header.Get("x-request-id"), truncateString(string(respBody), 1000))

		s.handleFailoverSideEffects(ctx, resp, account, input.RequestModel)
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: resp.StatusCode,
			UpstreamRequestID:  resp.Header.Get("x-request-id"),
			Passthrough:        true,
			Kind:               "failover",
			Message:            extractUpstreamErrorMessage(respBody),
			Detail: func() string {
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
				}
				return ""
			}(),
		})
		return nil, &UpstreamFailoverError{
			StatusCode:             resp.StatusCode,
			ResponseBody:           respBody,
			RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
		}
	}

	if resp.StatusCode >= 400 {
		return s.handleErrorResponse(ctx, resp, c, account, input.RequestModel)
	}
	var acceptedWireBody []byte
	if lastWireHandle != nil {
		if input.Parsed != nil {
			input.Parsed.RequestPayloadHash = lastWireHandle.Hash()
		}
		// 先验证再改绑，避免校验失败后 ParsedRequest 丢失原有的 byte-backed body。
		acceptedWireBody, err = lastWireHandle.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("read accepted anthropic passthrough wire body: %w", err)
		}
		if input.Parsed != nil && input.SourceHandleOwned {
			input.Parsed.Body = NewRequestBodyRefFromHandle(lastWireHandle)
		}
		if input.RequestStream || !input.SourceHandleOwned {
			acceptedWireBody = nil
		}
	}

	var usage *ClaudeUsage
	var firstTokenMs *int
	var clientDisconnect bool
	if input.RequestStream {
		streamResult, err := s.handleStreamingResponseAnthropicAPIKeyPassthrough(ctx, resp, c, account, input.StartTime, input.RequestModel)
		if err != nil {
			// 流中断时保留已观测到的 usage 与错误一起返回，避免上游已计量的请求
			// 完全漏记漏计费（issue #5148）。
			if partial := partialStreamUsageResult(c, resp, streamResult, input.OriginalModel, input.RequestModel, input.StartTime, err); partial != nil {
				return partial, err
			}
			return nil, err
		}
		usage = streamResult.usage
		firstTokenMs = streamResult.firstTokenMs
		clientDisconnect = streamResult.clientDisconnect
	} else {
		usage, err = s.handleNonStreamingResponseAnthropicAPIKeyPassthrough(ctx, resp, c, account)
		if err != nil {
			return nil, err
		}
	}
	if usage == nil {
		usage = &ClaudeUsage{}
	}
	if input.Parsed != nil && input.SourceHandleOwned && lastWireHandle != nil {
		if acceptedWireBody == nil {
			acceptedWireBody, err = lastWireHandle.ReadAll()
			if err != nil {
				return nil, fmt.Errorf("read accepted anthropic passthrough wire body: %w", err)
			}
		}
		if err := input.Parsed.ReplaceBody(acceptedWireBody); err != nil {
			return nil, fmt.Errorf("rewrite anthropic passthrough request body: %w", err)
		}
	}

	return &ForwardResult{
		RequestID:                     resp.Header.Get("x-request-id"),
		Usage:                         *usage,
		Model:                         input.OriginalModel,
		UpstreamModel:                 input.RequestModel,
		UpstreamResponseModel:         observedUpstreamResponseModel(c),
		UpstreamResponseModelConflict: observedUpstreamResponseModelConflict(c),
		UpstreamResponseServiceTier:   observedUpstreamResponseServiceTier(c),
		Stream:                        input.RequestStream,
		Duration:                      time.Since(input.StartTime),
		FirstTokenMs:                  firstTokenMs,
		ClientDisconnect:              clientDisconnect,
	}, nil
}

func (s *GatewayService) buildUpstreamRequestAnthropicAPIKeyPassthrough(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	sourceBody []byte,
	body []byte,
	token string,
) (*http.Request, []byte, error) {
	body = stripDeferredToolCacheControl(body)
	targetURL := claudeAPIURL
	baseURL := account.GetBaseURL()
	if baseURL != "" {
		validatedURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, nil, err
		}
		targetURL = validatedURL + "/v1/messages?beta=true"
	}

	// 能力维度 body sanitize：透传路径上 anthropic-beta header 原样透传客户端值，
	// 依此决定是否保留 body 中的 context_management。避免“客户端 body 带字段但
	// header 忘记带 beta token”的客户端 bug 在透传场景下让上游 400。
	clientBeta := ""
	if c != nil && c.Request != nil {
		clientBeta = getHeaderRaw(c.Request.Header, "anthropic-beta")
	}
	// 账号覆写了 anthropic-beta 时，覆写值即最终上游值：净化以覆写值为准
	if beta, ok := account.HeaderOverrideValue("anthropic-beta"); ok {
		clientBeta = beta
	}
	if sanitized, changed := sanitizeAnthropicBodyForBetaTokens(body, clientBeta); changed {
		body = sanitized
	}

	inboundHeader := http.Header{}
	if c != nil && c.Request != nil {
		inboundHeader = c.Request.Header
	}
	outboundHeader := http.Header{}
	for key, values := range inboundHeader {
		if !allowedHeaders[strings.ToLower(strings.TrimSpace(key))] {
			continue
		}
		for _, value := range values {
			addHeaderRaw(outboundHeader, resolveWireCasing(key), value)
		}
	}
	body, err := applyAccountPassthroughFieldsWithContext(ctx, account, inboundHeader, sourceBody, body, outboundHeader)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}

	req.Header = outboundHeader
	// 覆盖入站鉴权残留，并注入上游认证
	req.Header.Del("authorization")
	req.Header.Del("x-api-key")
	req.Header.Del("x-goog-api-key")
	req.Header.Del("cookie")
	setAnthropicAPIKeyAuthHeader(req.Header, account, token)

	if getHeaderRaw(req.Header, "content-type") == "" {
		setHeaderRaw(req.Header, "content-type", "application/json")
	}
	if getHeaderRaw(req.Header, "anthropic-version") == "" {
		setHeaderRaw(req.Header, "anthropic-version", "2023-06-01")
	}

	// 账号级请求头覆写（最终生效，覆盖上面所有来源的同名头）
	account.ApplyHeaderOverrides(req.Header)

	return req, body, nil
}

func extractContentTextFromResponseBody(body []byte) string {
	contentArr := gjson.GetBytes(body, "content")
	if !contentArr.IsArray() {
		return ""
	}
	var builder strings.Builder
	contentArr.ForEach(func(_, item gjson.Result) bool {
		switch item.Get("type").String() {
		case "text":
			_, _ = builder.WriteString(item.Get("text").String())
		case "tool_use":
			_, _ = builder.WriteString(item.Get("input").Raw)
		case "thinking":
			_, _ = builder.WriteString(item.Get("thinking").String())
		}
		return true
	})
	return builder.String()
}

func (s *GatewayService) handleStreamingResponseAnthropicAPIKeyPassthrough(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
	startTime time.Time,
	model string,
) (*streamingResult, error) {
	observer := upstreamResponseModelObserverFromContext(c)
	if observer == nil {
		observer = beginUpstreamResponseModelObservation(c)
	}
	if s.rateLimitService != nil {
		s.rateLimitService.UpdateSessionWindow(ctx, account, resp.Header)
	}

	writeAnthropicPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "text/event-stream"
	}
	c.Header("Content-Type", contentType)
	if c.Writer.Header().Get("Cache-Control") == "" {
		c.Header("Cache-Control", "no-cache")
	}
	if c.Writer.Header().Get("Connection") == "" {
		c.Header("Connection", "keep-alive")
	}
	c.Header("X-Accel-Buffering", "no")
	if v := resp.Header.Get("x-request-id"); v != "" {
		c.Header("x-request-id", v)
	}

	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}

	usage := &ClaudeUsage{}
	var firstTokenMs *int
	clientDisconnected := false
	sawTerminalEvent := false
	var outputText strings.Builder
	defer func() { applyOutputTokenEstimation(usage, outputText.String()) }()

	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	scanner.Buffer(scanBuf[:0], maxLineSize)

	type scanEvent struct {
		line string
		err  error
	}
	events := make(chan scanEvent, 16)
	done := make(chan struct{})
	sendEvent := func(ev scanEvent) bool {
		select {
		case events <- ev:
			return true
		case <-done:
			return false
		}
	}
	var lastReadAt int64
	atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
	go func(scanBuf *sseScannerBuf64K) {
		defer putSSEScannerBuf64K(scanBuf)
		defer close(events)
		for scanner.Scan() {
			atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
			if !sendEvent(scanEvent{line: scanner.Text()}) {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			_ = sendEvent(scanEvent{err: err})
		}
	}(scanBuf)
	defer close(done)

	streamInterval := time.Duration(0)
	if s.cfg != nil && s.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		streamInterval = time.Duration(s.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
	}
	var intervalTicker *time.Ticker
	if streamInterval > 0 {
		intervalTicker = time.NewTicker(streamInterval)
		defer intervalTicker.Stop()
	}
	var intervalCh <-chan time.Time
	if intervalTicker != nil {
		intervalCh = intervalTicker.C
	}

	keepaliveInterval := time.Duration(0)
	if s.cfg != nil && s.cfg.Gateway.StreamKeepaliveInterval > 0 {
		keepaliveInterval = time.Duration(s.cfg.Gateway.StreamKeepaliveInterval) * time.Second
	}
	var keepaliveTimer *time.Timer
	if keepaliveInterval > 0 {
		keepaliveTimer = time.NewTimer(keepaliveInterval)
		defer keepaliveTimer.Stop()
	}
	var keepaliveCh <-chan time.Time
	if keepaliveTimer != nil {
		keepaliveCh = keepaliveTimer.C
	}
	lastDataAt := time.Now()
	resetKeepaliveTimer := func() {
		if keepaliveTimer == nil {
			return
		}
		if !keepaliveTimer.Stop() {
			select {
			case <-keepaliveTimer.C:
			default:
			}
		}
		keepaliveTimer.Reset(keepaliveInterval)
	}
	inPartialEvent := false

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				if !clientDisconnected {
					// 兜底补刷，确保最后一个未以空行结尾的事件也能及时送达客户端。
					flusher.Flush()
				}
				if !sawTerminalEvent {
					if clientDisconnected && streamInterval > 0 {
						lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
						if time.Since(lastRead) >= streamInterval {
							return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}, fmt.Errorf("stream usage incomplete after timeout")
						}
					}
					return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: clientDisconnected}, fmt.Errorf("stream usage incomplete: missing terminal event")
				}
				return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: clientDisconnected}, nil
			}
			if ev.err != nil {
				if sawTerminalEvent {
					return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: clientDisconnected}, nil
				}
				if clientDisconnected {
					return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}, fmt.Errorf("stream usage incomplete after disconnect: %w", ev.err)
				}
				if errors.Is(ev.err, context.Canceled) || errors.Is(ev.err, context.DeadlineExceeded) {
					return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}, fmt.Errorf("stream usage incomplete: %w", ev.err)
				}
				if errors.Is(ev.err, bufio.ErrTooLong) {
					logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] SSE line too long: account=%d max_size=%d error=%v", account.ID, maxLineSize, ev.err)
					return &streamingResult{usage: usage, firstTokenMs: firstTokenMs}, ev.err
				}
				return &streamingResult{usage: usage, firstTokenMs: firstTokenMs}, fmt.Errorf("stream read error: %w", ev.err)
			}

			line := ev.line
			if data, ok := extractAnthropicSSEDataLine(line); ok {
				trimmed := strings.TrimSpace(data)
				observer.ObserveAnthropic([]byte(trimmed))
				if anthropicStreamEventIsTerminal("", trimmed) {
					sawTerminalEvent = true
				}
				if firstTokenMs == nil && trimmed != "" && trimmed != "[DONE]" {
					ms := int(time.Since(startTime).Milliseconds())
					firstTokenMs = &ms
				}
				parseSSEUsagePassthrough(data, usage)
				_, _ = outputText.WriteString(gjson.Get(data, "delta.text").String())
				_, _ = outputText.WriteString(gjson.Get(data, "delta.thinking").String())
				_, _ = outputText.WriteString(gjson.Get(data, "delta.partial_json").String())
			} else {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "event:") && anthropicStreamEventIsTerminal(strings.TrimSpace(strings.TrimPrefix(trimmed, "event:")), "") {
					sawTerminalEvent = true
				}
			}

			if !clientDisconnected {
				restored := string(reverseToolNamesIfPresent(c, []byte(line)))
				if _, err := io.WriteString(w, restored); err != nil {
					clientDisconnected = true
					logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] Client disconnected during streaming, continue draining upstream for usage: account=%d", account.ID)
				} else if _, err := io.WriteString(w, "\n"); err != nil {
					clientDisconnected = true
					logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] Client disconnected during streaming, continue draining upstream for usage: account=%d", account.ID)
				} else if line == "" {
					// 按 SSE 事件边界刷出，减少每行 flush 带来的 syscall 开销。
					flusher.Flush()
					lastDataAt = time.Now()
					resetKeepaliveTimer()
					inPartialEvent = false
				} else {
					inPartialEvent = true
				}
			}

		case <-intervalCh:
			lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
			if time.Since(lastRead) < streamInterval {
				continue
			}
			if clientDisconnected {
				return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}, fmt.Errorf("stream usage incomplete after timeout")
			}
			logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] Stream data interval timeout: account=%d model=%s interval=%s", account.ID, model, streamInterval)
			if s.rateLimitService != nil {
				s.rateLimitService.HandleStreamTimeout(ctx, account, model)
			}
			return &streamingResult{usage: usage, firstTokenMs: firstTokenMs}, fmt.Errorf("stream data interval timeout")

		case <-keepaliveCh:
			if clientDisconnected {
				continue
			}
			if inPartialEvent {
				resetKeepaliveTimer()
				continue
			}
			if time.Since(lastDataAt) < keepaliveInterval {
				resetKeepaliveTimer()
				continue
			}
			if _, err := fmt.Fprint(w, "event: ping\ndata: {\"type\": \"ping\"}\n\n"); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] Client disconnected during keepalive ping, continue draining upstream for usage: account=%d", account.ID)
				continue
			}
			flusher.Flush()
			lastDataAt = time.Now()
			resetKeepaliveTimer()
		}
	}
}

func extractAnthropicSSEDataLine(line string) (string, bool) {
	if !strings.HasPrefix(line, "data:") {
		return "", false
	}
	start := len("data:")
	for start < len(line) {
		if line[start] != ' ' && line[start] != '\t' {
			break
		}
		start++
	}
	return line[start:], true
}

// parseSSEUsagePassthrough 从 Anthropic SSE data 行提取 usage（包级函数：
// Anthropic 平台 passthrough 与国产供应商原生 Anthropic 直通共用）。
func parseSSEUsagePassthrough(data string, usage *ClaudeUsage) {
	if usage == nil || data == "" || data == "[DONE]" {
		return
	}

	parsed := gjson.Parse(data)
	switch parsed.Get("type").String() {
	case "message_start":
		msgUsage := parsed.Get("message.usage")
		if msgUsage.Exists() {
			usage.InputTokens = int(msgUsage.Get("input_tokens").Int())
			usage.CacheCreationInputTokens = int(msgUsage.Get("cache_creation_input_tokens").Int())
			usage.CacheReadInputTokens = int(msgUsage.Get("cache_read_input_tokens").Int())

			// 保持与通用解析一致：message_start 允许覆盖 5m/1h 明细（包括 0）。
			cc5m := msgUsage.Get("cache_creation.ephemeral_5m_input_tokens")
			cc1h := msgUsage.Get("cache_creation.ephemeral_1h_input_tokens")
			if cc5m.Exists() || cc1h.Exists() {
				usage.CacheCreation5mTokens = int(cc5m.Int())
				usage.CacheCreation1hTokens = int(cc1h.Int())
			}
		}
	case "message_delta":
		deltaUsage := parsed.Get("usage")
		if deltaUsage.Exists() {
			if v := deltaUsage.Get("input_tokens").Int(); v > 0 {
				usage.InputTokens = int(v)
			}
			if v := deltaUsage.Get("output_tokens").Int(); v > 0 {
				usage.OutputTokens = int(v)
			}
			if v := deltaUsage.Get("cache_creation_input_tokens").Int(); v > 0 {
				usage.CacheCreationInputTokens = int(v)
			}
			if v := deltaUsage.Get("cache_read_input_tokens").Int(); v > 0 {
				usage.CacheReadInputTokens = int(v)
			}

			cc5m := deltaUsage.Get("cache_creation.ephemeral_5m_input_tokens")
			cc1h := deltaUsage.Get("cache_creation.ephemeral_1h_input_tokens")
			if cc5m.Exists() && cc5m.Int() > 0 {
				usage.CacheCreation5mTokens = int(cc5m.Int())
			}
			if cc1h.Exists() && cc1h.Int() > 0 {
				usage.CacheCreation1hTokens = int(cc1h.Int())
			}
		}
	}

	if usage.CacheReadInputTokens == 0 {
		if cached := parsed.Get("message.usage.cached_tokens").Int(); cached > 0 {
			usage.CacheReadInputTokens = int(cached)
		}
		if cached := parsed.Get("usage.cached_tokens").Int(); usage.CacheReadInputTokens == 0 && cached > 0 {
			usage.CacheReadInputTokens = int(cached)
		}
	}
	if usage.CacheCreationInputTokens == 0 {
		cc5m := parsed.Get("message.usage.cache_creation.ephemeral_5m_input_tokens").Int()
		cc1h := parsed.Get("message.usage.cache_creation.ephemeral_1h_input_tokens").Int()
		if cc5m == 0 && cc1h == 0 {
			cc5m = parsed.Get("usage.cache_creation.ephemeral_5m_input_tokens").Int()
			cc1h = parsed.Get("usage.cache_creation.ephemeral_1h_input_tokens").Int()
		}
		total := cc5m + cc1h
		if total > 0 {
			usage.CacheCreationInputTokens = int(total)
		}
	}

	// Kimi's Anthropic-compatible stream uses input_tokens with two meanings:
	// message_start reports total prompt input, while message_delta reports only
	// uncached input. prompt_tokens remains the total in both events. Normalize
	// to ClaudeUsage's mutually-exclusive buckets so downstream billing does not
	// subtract cache tokens from an already-uncached value.
	usageNode := parsed.Get("usage")
	if parsed.Get("type").String() == "message_start" {
		usageNode = parsed.Get("message.usage")
	}
	normalizeAnthropicCompatiblePromptUsage(usageNode, usage)
}

// normalizeAnthropicCompatiblePromptUsage converts provider-native OpenAI-style
// prompt/cache fields into Claude's mutually-exclusive usage buckets. Native
// Anthropic responses do not expose these aliases and are left alone.
func normalizeAnthropicCompatiblePromptUsage(usageNode gjson.Result, usage *ClaudeUsage) bool {
	if usage == nil || !usageNode.Exists() {
		return false
	}
	promptTokens := usageNode.Get("prompt_tokens")
	promptCacheHitTokens := usageNode.Get("prompt_cache_hit_tokens")
	promptCacheMissTokens := usageNode.Get("prompt_cache_miss_tokens")
	if (!promptTokens.Exists() || promptTokens.Int() <= 0) &&
		!promptCacheHitTokens.Exists() && !promptCacheMissTokens.Exists() {
		return false
	}

	cacheReadTokens := usage.CacheReadInputTokens
	if v := usageNode.Get("cache_read_input_tokens"); v.Exists() {
		cacheReadTokens = int(v.Int())
	}
	if cacheReadTokens == 0 {
		if v := usageNode.Get("cached_tokens"); v.Exists() {
			cacheReadTokens = int(v.Int())
		}
	}
	if cacheReadTokens == 0 {
		if v := usageNode.Get("prompt_tokens_details.cached_tokens"); v.Exists() {
			cacheReadTokens = int(v.Int())
		}
	}
	if cacheReadTokens == 0 && promptCacheHitTokens.Exists() {
		cacheReadTokens = max(int(promptCacheHitTokens.Int()), 0)
	}

	cacheCreationTokens := usage.CacheCreationInputTokens
	if v := usageNode.Get("cache_creation_input_tokens"); v.Exists() {
		cacheCreationTokens = int(v.Int())
	}
	if cacheCreationTokens == 0 {
		cc5m := usageNode.Get("cache_creation.ephemeral_5m_input_tokens").Int()
		cc1h := usageNode.Get("cache_creation.ephemeral_1h_input_tokens").Int()
		if cc5m > 0 || cc1h > 0 {
			cacheCreationTokens = int(cc5m + cc1h)
		}
	}

	if promptCacheMissTokens.Exists() {
		usage.InputTokens = max(int(promptCacheMissTokens.Int()), 0)
	} else {
		usage.InputTokens = max(int(promptTokens.Int())-cacheReadTokens-cacheCreationTokens, 0)
	}
	usage.CacheReadInputTokens = cacheReadTokens
	usage.CacheCreationInputTokens = cacheCreationTokens
	return true
}

func parseClaudeUsageFromResponseBody(body []byte) *ClaudeUsage {
	usage := &ClaudeUsage{}
	if len(body) == 0 {
		return usage
	}

	parsed := gjson.ParseBytes(body)
	usageNode := parsed.Get("usage")
	if !usageNode.Exists() {
		return usage
	}

	usage.InputTokens = int(usageNode.Get("input_tokens").Int())
	usage.OutputTokens = int(usageNode.Get("output_tokens").Int())
	usage.CacheCreationInputTokens = int(usageNode.Get("cache_creation_input_tokens").Int())
	usage.CacheReadInputTokens = int(usageNode.Get("cache_read_input_tokens").Int())

	cc5m := usageNode.Get("cache_creation.ephemeral_5m_input_tokens").Int()
	cc1h := usageNode.Get("cache_creation.ephemeral_1h_input_tokens").Int()
	if cc5m > 0 || cc1h > 0 {
		usage.CacheCreation5mTokens = int(cc5m)
		usage.CacheCreation1hTokens = int(cc1h)
	}
	if usage.CacheCreationInputTokens == 0 && (cc5m > 0 || cc1h > 0) {
		usage.CacheCreationInputTokens = int(cc5m + cc1h)
	}
	if usage.CacheReadInputTokens == 0 {
		if cached := usageNode.Get("cached_tokens").Int(); cached > 0 {
			usage.CacheReadInputTokens = int(cached)
		}
	}
	normalizeAnthropicCompatiblePromptUsage(usageNode, usage)
	return usage
}

// invalidNonStreamingJSONFailoverError 把"上游 2xx 返回非 JSON body"归一为
// failover 错误（包级函数：Anthropic 平台 passthrough 与国产供应商原生
// Anthropic 直通共用）。
func invalidNonStreamingJSONFailoverError(
	ctx context.Context,
	rateLimitService *RateLimitService,
	resp *http.Response,
	account *Account,
	body []byte,
	parseErr error,
	requestedModel ...string,
) error {
	const statusCode = http.StatusBadGateway

	accountID := int64(0)
	accountName := ""
	retryableOnSameAccount := false
	if account != nil {
		accountID = account.ID
		accountName = account.Name
		retryableOnSameAccount = account.IsPoolMode() && account.IsPoolModeRetryableStatus(statusCode)
	}

	logger.LegacyPrintf(
		"service.gateway",
		"Account %d(%s): upstream returned non-JSON 2xx response, attempting failover: status=%d request_id=%s error=%v",
		accountID,
		accountName,
		resp.StatusCode,
		resp.Header.Get("x-request-id"),
		parseErr,
	)

	if rateLimitService != nil && account != nil {
		if len(requestedModel) > 0 {
			rateLimitService.HandleUpstreamError(ctx, account, statusCode, resp.Header, body, requestedModel[0])
		} else {
			rateLimitService.HandleUpstreamError(ctx, account, statusCode, resp.Header, body)
		}
	}

	return &UpstreamFailoverError{
		StatusCode:             statusCode,
		ResponseBody:           body,
		ResponseHeaders:        resp.Header,
		RetryableOnSameAccount: retryableOnSameAccount,
	}
}

func (s *GatewayService) handleNonStreamingResponseAnthropicAPIKeyPassthrough(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
) (*ClaudeUsage, error) {
	if s.rateLimitService != nil {
		s.rateLimitService.UpdateSessionWindow(ctx, account, resp.Header)
	}

	body, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, anthropicTooLargeError)
	if err != nil {
		return nil, err
	}
	observer := upstreamResponseModelObserverFromContext(c)
	if observer == nil {
		observer = beginUpstreamResponseModelObservation(c)
	}
	observer.ObserveAnthropic(body)

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		var raw json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, invalidNonStreamingJSONFailoverError(ctx, s.rateLimitService, resp, account, body, err)
		}
	}

	usage := parseClaudeUsageFromResponseBody(body)
	applyOutputTokenEstimation(usage, extractContentTextFromResponseBody(body))
	if IsForceCacheBilling(ctx) && usage.InputTokens > 0 {
		body, err = classifyAnthropicResponseInputAsCacheRead(body, usage)
		if err != nil {
			return nil, err
		}
	}

	writeAnthropicPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/json"
	}
	body = reverseToolNamesIfPresent(c, body)
	c.Data(resp.StatusCode, contentType, body)
	return usage, nil
}

func classifyAnthropicResponseInputAsCacheRead(body []byte, usage *ClaudeUsage) ([]byte, error) {
	classified, err := sjson.SetBytes(body, "usage.input_tokens", 0)
	if err != nil {
		return nil, fmt.Errorf("classify forced cache billing input tokens: %w", err)
	}
	classified, err = sjson.SetBytes(classified, "usage.cache_read_input_tokens", usage.CacheReadInputTokens+usage.InputTokens)
	if err != nil {
		return nil, fmt.Errorf("classify forced cache billing cache read tokens: %w", err)
	}
	return classified, nil
}

func writeAnthropicPassthroughResponseHeaders(dst http.Header, src http.Header, filter *responseheaders.CompiledHeaderFilter) {
	if dst == nil || src == nil {
		return
	}
	if filter != nil {
		responseheaders.WriteFilteredHeaders(dst, src, filter)
		return
	}
	if v := strings.TrimSpace(src.Get("Content-Type")); v != "" {
		dst.Set("Content-Type", v)
	}
	if v := strings.TrimSpace(src.Get("x-request-id")); v != "" {
		dst.Set("x-request-id", v)
	}
}
