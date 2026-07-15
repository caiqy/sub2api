package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GrokImages handles xAI image generation/editing through Grok groups.
func (h *OpenAIGatewayHandler) GrokImages(c *gin.Context) {
	endpoint := service.GrokMediaEndpointImagesGenerations
	if strings.Contains(c.Request.URL.Path, "/images/edits") {
		endpoint = service.GrokMediaEndpointImagesEdits
	}
	h.handleGrokMedia(c, endpoint, "")
}

// GrokVideoGeneration handles xAI video generation through Grok groups.
func (h *OpenAIGatewayHandler) GrokVideoGeneration(c *gin.Context) {
	h.handleGrokMedia(c, service.GrokMediaEndpointVideosGenerations, "")
}

// GrokVideoStatus handles xAI video status retrieval through Grok groups.
func (h *OpenAIGatewayHandler) GrokVideoStatus(c *gin.Context) {
	h.handleGrokMedia(c, service.GrokMediaEndpointVideoStatus, c.Param("request_id"))
}

func (h *OpenAIGatewayHandler) handleGrokMedia(c *gin.Context, endpoint service.GrokMediaEndpoint, requestID string) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)

	requestStart := time.Now()
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}

	reqLog := requestLogger(
		c,
		"handler.openai_gateway.grok_media",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
		zap.String("endpoint", string(endpoint)),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	var coordinator *requestBodyCoordinator
	var body []byte
	var requestInfo service.GrokMediaRequestInfo
	if endpoint.RequiresRequestBody() {
		var err error
		if isMultipartImagesContentType(c.GetHeader("Content-Type")) {
			coordinator, err = newMultipartRequestBody(c.Request, 0)
		} else {
			coordinator, err = newJSONRequestBody(c.Request)
		}
		if err != nil {
			if errors.Is(err, service.ErrRequestBodySpool) {
				h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
				return
			}
			if maxErr, ok := extractMaxBytesError(err); ok {
				h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
				return
			}
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
			return
		}
		defer coordinator.Cleanup()
		if coordinator.form != nil {
			c.Request.MultipartForm = coordinator.form
			requestInfo = service.ParseGrokMediaMultipartForm(coordinator.form)
		} else {
			body, err = coordinator.ReadRaw()
			if err != nil {
				if errors.Is(err, service.ErrRequestBodySpool) {
					h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
					return
				}
				h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
				return
			}
			if len(body) == 0 {
				h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
				return
			}
			requestInfo = service.ParseGrokMediaRequest(c.GetHeader("Content-Type"), body)
		}
	}

	contentType := c.GetHeader("Content-Type")
	if endpoint.RequiresRequestBody() {
		preview := grokMediaRequestBodyPreviewWithSize(contentType, body, requestInfo, coordinator.Effective().Size())
		service.SetUsageRequestBody(c, preview)
		service.SetOpsUpstreamRequestBodyPreview(c, preview, coordinator.Effective().Size())
	}
	if coordinator != nil && coordinator.form != nil && endpoint == service.GrokMediaEndpointImagesEdits {
		forwardBody, forwardContentType, prepareErr := service.PrepareGrokMediaFormForwardBody(requestInfo)
		if prepareErr == nil {
			prepareErr = coordinator.SetEffectiveBytes(forwardBody)
		}
		if prepareErr != nil {
			if errors.Is(prepareErr, service.ErrRequestBodySpool) {
				h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
				return
			}
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to prepare request body")
			return
		}
		// The effective handle owns the bytes; drop the temporary body for GC.
		forwardBody = nil
		contentType = forwardContentType
		body = nil
	}
	requestModel := requestInfo.Model
	if endpoint.IsGenerationRequest() && strings.TrimSpace(requestModel) == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	if endpoint == service.GrokMediaEndpointVideoStatus && strings.TrimSpace(requestID) == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "request_id is required")
		return
	}

	reqLog = reqLog.With(zap.String("model", requestModel))
	setOpsRequestContext(c, requestModel, false)
	setOpsEndpointContext(c, "", int16(service.RequestTypeSync))
	var sessionHash, requestPayloadHash string

	if endpoint.IsGenerationRequest() {
		if !service.GroupAllowsImageGeneration(apiKey.Group) {
			h.errorResponse(c, http.StatusForbidden, "permission_error", service.ImageGenerationPermissionMessage())
			return
		}
		if moderationBody := requestInfo.ModerationBody(); len(moderationBody) > 0 {
			decision := h.checkContentModeration(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIImages, requestModel, moderationBody)
			if decision != nil && decision.Blocked {
				h.errorResponse(c, contentModerationStatus(decision), contentModerationErrorCode(decision), decision.Message)
				return
			}
		}
		fallbackSessionSeed := ""
		if coordinator != nil && coordinator.form != nil {
			fallbackSessionSeed = coordinator.Effective().Hash()
		}
		sessionSeed := body
		if len(sessionSeed) == 0 && strings.TrimSpace(requestID) != "" {
			sessionSeed = []byte(requestID)
		}
		sessionHash = h.gatewayService.GenerateSessionHash(c, sessionSeed)
		if coordinator != nil && coordinator.form != nil {
			sessionHash = h.gatewayService.GenerateSessionHashWithFallback(c, sessionSeed, fallbackSessionSeed)
		}
		if coordinator != nil {
			service.BindOpenAIRequestBodyHandle(c, coordinator.Effective())
			requestPayloadHash = coordinator.Effective().Hash()
			coordinator.ReleaseMultipartValues()
		}
		body = nil
		requestInfo.ReleaseText()
		if endpoint == service.GrokMediaEndpointVideoStatus {
			sessionHash = service.GrokMediaVideoRequestSessionHash(requestID)
		}
		imageReleaseFunc, acquired := h.acquireImageGenerationSlot(c, streamStarted)
		if !acquired {
			return
		}
		if imageReleaseFunc != nil {
			defer imageReleaseFunc()
		}
	}

	if !endpoint.IsGenerationRequest() {
		if strings.TrimSpace(requestID) != "" {
			sessionHash = service.GrokMediaVideoRequestSessionHash(requestID)
			requestPayloadHash = service.HashUsageRequestPayload([]byte(requestID))
		}
		requestInfo.ReleaseText()
	}

	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}

	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		reqLog.Info("grok_media.billing_eligibility_check_failed", zap.Error(err))
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.errorResponse(c, status, code, message)
		return
	}

	requestCtx := c.Request.Context()
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError
	var lastFailedAccount *service.Account
	var lastFailedDuration time.Duration
	switchCount := 0
	maxAccountSwitches := h.maxAccountSwitches
	if maxAccountSwitches <= 0 {
		maxAccountSwitches = 3
	}
	routingStart := time.Now()

	for {
		selection, scheduleDecision, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
			requestCtx,
			apiKey.GroupID,
			"",
			sessionHash,
			requestModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportHTTPSSE,
			"",
			false,
			false,
			service.PlatformGrok,
		)
		if err != nil {
			reqLog.Warn("grok_media.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			if len(failedAccountIDs) == 0 {
				cls := classifyNoAccountErrorFromGin(c, h.gatewayService, apiKey, requestModel, requestModel, service.PlatformGrok)
				if !cls.ModelNotFound {
					markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
				}
				h.errorResponse(c, cls.Status, cls.ErrType, cls.Message)
				return
			}
			if lastFailoverErr != nil {
				h.submitFailoverFailedUsageLog(c, apiKey, lastFailedAccount, requestModel, false, lastFailoverErr, lastFailedDuration, nil, "handler.openai_gateway.grok_media")
				h.handleFailoverExhausted(c, lastFailoverErr, false)
			} else {
				h.errorResponse(c, http.StatusBadGateway, "api_error", "Upstream request failed")
			}
			return
		}
		if selection == nil || selection.Account == nil {
			cls := classifyNoAccountErrorFromGin(c, h.gatewayService, apiKey, requestModel, requestModel, service.PlatformGrok)
			if !cls.ModelNotFound {
				markOpsRoutingCapacityLimited(c)
			}
			if lastFailoverErr != nil {
				h.submitFailoverFailedUsageLog(c, apiKey, lastFailedAccount, requestModel, false, lastFailoverErr, lastFailedDuration, nil, "handler.openai_gateway.grok_media")
				h.handleFailoverExhausted(c, lastFailoverErr, false)
			} else {
				h.errorResponse(c, cls.Status, cls.ErrType, cls.Message)
			}
			return
		}

		reqLog.Debug("grok_media.account_schedule_decision",
			zap.String("layer", scheduleDecision.Layer),
			zap.Bool("sticky_session_hit", scheduleDecision.StickySessionHit),
			zap.Int("candidate_count", scheduleDecision.CandidateCount),
			zap.Int("top_k", scheduleDecision.TopK),
			zap.Int64("latency_ms", scheduleDecision.LatencyMs),
			zap.Float64("load_skew", scheduleDecision.LoadSkew),
		)

		account := selection.Account
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		setOpsSelectedAccount(c, account.ID, account.Platform)

		accountReleaseFunc, accountAcquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, false, &streamStarted, reqLog)
		if !accountAcquired {
			return
		}

		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		writerSizeBeforeForward := c.Writer.Size()
		service.SetOpsUpstreamAttempted(c, false)
		result, err := func() (*service.OpenAIForwardResult, error) {
			defer func() {
				if accountReleaseFunc != nil {
					accountReleaseFunc()
				}
			}()
			return h.gatewayService.ForwardGrokMedia(requestCtx, c, account, endpoint, requestID, nil, contentType)
		}()

		forwardDuration := time.Since(forwardStart)
		forwardDurationMs := forwardDuration.Milliseconds()
		upstreamLatencyMs, _ := getContextInt64(c, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, responseLatencyMs)

		if err != nil {
			if errors.Is(err, service.ErrRequestBodySpool) {
				h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
				return
			}
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				if c.Writer.Size() != writerSizeBeforeForward {
					if requestCtx.Err() == nil {
						h.submitFailoverFailedUsageLog(c, apiKey, account, requestModel, false, failoverErr, forwardDuration, nil, "handler.openai_gateway.grok_media")
					}
					h.handleFailoverExhausted(c, failoverErr, true)
					return
				}
				if failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						reqLog.Warn("grok_media.pool_mode_same_account_retry",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
							zap.Int("retry_limit", retryLimit),
							zap.Int("retry_count", sameAccountRetryCount[account.ID]),
						)
						select {
						case <-requestCtx.Done():
							return
						case <-time.After(sameAccountRetryDelay):
						}
						continue
					}
				}
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				lastFailedAccount = account
				lastFailedDuration = forwardDuration
				if switchCount >= maxAccountSwitches {
					h.submitFailoverFailedUsageLog(c, apiKey, account, requestModel, false, failoverErr, forwardDuration, nil, "handler.openai_gateway.grok_media")
					h.handleFailoverExhausted(c, failoverErr, false)
					return
				}
				switchCount++
				reqLog.Warn("grok_media.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			if requestCtx.Err() == nil && service.HasOpsUpstreamAttempted(c) && !service.HasOpsClientBusinessLimited(c) {
				h.submitFailedUsageLog(c, apiKey, account, requestModel, false, 0, nil, nil, forwardDuration, nil, "handler.openai_gateway.grok_media")
			}
			if c.Writer.Size() == writerSizeBeforeForward {
				h.errorResponse(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
			}
			reqLog.Warn("grok_media.forward_failed",
				zap.Int64("account_id", account.ID),
				zap.Error(err),
			)
			return
		}

		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
		if endpoint == service.GrokMediaEndpointVideosGenerations && strings.TrimSpace(result.ResponseID) != "" {
			if err := h.gatewayService.BindGrokMediaVideoRequestAccount(requestCtx, apiKey.GroupID, result.ResponseID, account.ID); err != nil {
				reqLog.Warn("grok_media.bind_video_request_account_failed",
					zap.Int64("account_id", account.ID),
					zap.String("request_id", result.ResponseID),
					zap.Error(err),
				)
			}
		}
		if shouldRecordGrokMediaUsage(endpoint, requestModel) {
			recordGrokMediaUsage(c, h, reqLog, apiKey, subject, subscription, account, result, requestModel, requestPayloadHash, requestID)
		}
		reqLog.Debug("grok_media.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", switchCount),
		)
		return
	}
}

func grokMediaRequestBodyPreview(contentType string, body []byte, requestInfo service.GrokMediaRequestInfo) string {
	return grokMediaRequestBodyPreviewWithSize(contentType, body, requestInfo, int64(len(body)))
}

func grokMediaRequestBodyPreviewWithSize(contentType string, body []byte, requestInfo service.GrokMediaRequestInfo, size int64) string {
	requestPreview := service.RequestBodyPreviewString(body)
	if !isMultipartImagesContentType(contentType) && requestPreview != "[inline binary payload omitted]" {
		return service.RequestBodyPreviewSnapshot(requestPreview, size)
	}
	prompt := multipartMetadataPromptPreview(requestInfo.Prompt)
	if isMultipartImagesContentType(contentType) {
		prompt = ""
	}
	preview, err := json.Marshal(struct {
		Model          string `json:"model"`
		Prompt         string `json:"prompt"`
		Size           string `json:"size"`
		N              int    `json:"n"`
		HadSourceImage bool   `json:"had_source_image"`
		HadMask        bool   `json:"had_mask"`
	}{
		Model:          requestInfo.Model,
		Prompt:         prompt,
		Size:           requestInfo.Size,
		N:              requestInfo.N,
		HadSourceImage: len(requestInfo.Uploads) > 0 || len(requestInfo.InputImageURLs) > 0,
		HadMask:        requestInfo.MaskUpload != nil || strings.TrimSpace(requestInfo.MaskImageURL) != "",
	})
	if err != nil {
		return "[multipart body omitted]"
	}
	return service.RequestBodyPreviewSnapshot(string(preview), size, true)
}

func shouldRecordGrokMediaUsage(endpoint service.GrokMediaEndpoint, requestModel string) bool {
	return endpoint.IsGenerationRequest() && strings.TrimSpace(requestModel) != ""
}

func recordGrokMediaUsage(
	c *gin.Context,
	h *OpenAIGatewayHandler,
	reqLog *zap.Logger,
	apiKey *service.APIKey,
	subject middleware2.AuthSubject,
	subscription *service.UserSubscription,
	account *service.Account,
	result *service.OpenAIForwardResult,
	requestModel string,
	requestPayloadHash string,
	requestID string,
) {
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	if requestPayloadHash == "" && strings.TrimSpace(requestID) != "" {
		requestPayloadHash = service.HashUsageRequestPayload([]byte(requestID))
	}
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
	detailSnapshot := middleware2.BuildUsageDetailSnapshot(c)
	quotaPlatform := service.QuotaPlatform(c.Request.Context(), apiKey)
	channelUsageFields := service.ChannelUsageFields{
		OriginalModel:      requestModel,
		ChannelMappedModel: requestModel,
	}
	h.submitOpenAIUsageRecordTask(c.Request.Context(), result, func(ctx context.Context) {
		if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
			Result:             result,
			APIKey:             apiKey,
			User:               apiKey.User,
			Account:            account,
			Subscription:       subscription,
			InboundEndpoint:    inboundEndpoint,
			UpstreamEndpoint:   upstreamEndpoint,
			UserAgent:          userAgent,
			IPAddress:          clientIP,
			RequestPayloadHash: requestPayloadHash,
			APIKeyService:      h.apiKeyService,
			QuotaPlatform:      quotaPlatform,
			ChannelUsageFields: channelUsageFields,
			DetailSnapshot:     detailSnapshot,
		}); err != nil {
			logger.L().With(
				zap.String("component", "handler.openai_gateway.grok_media"),
				zap.Int64("user_id", subject.UserID),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Any("group_id", apiKey.GroupID),
				zap.String("model", requestModel),
				zap.Int64("account_id", account.ID),
			).Error("grok_media.record_usage_failed", zap.Error(err))
			reqLog.Debug("grok_media.record_usage_failed", zap.Error(err))
		}
	})
}
