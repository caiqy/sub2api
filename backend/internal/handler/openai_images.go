package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Images handles OpenAI Images API requests.
// POST /v1/images/generations
// POST /v1/images/edits
func (h *OpenAIGatewayHandler) Images(c *gin.Context) {
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
		"handler.openai_gateway.images",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	if isMultipartImagesContentType(c.GetHeader("Content-Type")) {
		setOpsRequestContext(c, "", false, nil)
	} else {
		setOpsRequestContext(c, "", false, body)
	}

	parsed, err := h.gatewayService.ParseOpenAIImagesRequest(c, body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	reqLog = reqLog.With(
		zap.String("model", parsed.Model),
		zap.Bool("stream", parsed.Stream),
		zap.Bool("multipart", parsed.Multipart),
		zap.String("capability", string(parsed.RequiredCapability)),
	)

	if parsed.Multipart {
		setOpsRequestContext(c, parsed.Model, parsed.Stream, nil)
	} else {
		setOpsRequestContext(c, parsed.Model, parsed.Stream, body)
	}
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(parsed.Stream, false)))

	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, parsed.Model)

	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}

	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, parsed.Stream, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if apiKey.GroupID != nil && apiKey.Group != nil {
		groupUserReleaseFunc, groupAcquired := h.acquireUserGroupSlot(c, subject.UserID, *apiKey.GroupID, apiKey.Group, parsed.Stream, &streamStarted, reqLog)
		if !groupAcquired {
			return
		}
		if groupUserReleaseFunc != nil {
			defer groupUserReleaseFunc()
		}
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		reqLog.Info("openai.images.billing_eligibility_check_failed", zap.Error(err))
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.handleStreamingAwareError(c, status, code, message, streamStarted)
		return
	}

	sessionHash := h.gatewayService.GenerateExplicitSessionHash(c, body)

	maxAccountSwitches := h.maxAccountSwitches
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError
	var lastFailedAccount *service.Account
	var lastFailedDuration time.Duration

	for {
		reqLog.Debug("openai.images.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
		selection, scheduleDecision, err := h.gatewayService.SelectAccountWithSchedulerForImages(
			c.Request.Context(),
			apiKey.GroupID,
			sessionHash,
			parsed.Model,
			failedAccountIDs,
			parsed.RequiredCapability,
		)
		if err != nil {
			reqLog.Warn("openai.images.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			if len(failedAccountIDs) == 0 {
				h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available compatible accounts", streamStarted)
				return
			}
			if lastFailoverErr != nil {
				h.handleFailoverExhausted(c, lastFailoverErr, streamStarted)
				h.submitOpenAIImagesFailoverFailedUsageLog(c, apiKey, lastFailedAccount, parsed, lastFailoverErr, lastFailedDuration)
			} else {
				h.handleFailoverExhaustedSimple(c, 502, streamStarted)
			}
			return
		}
		if selection == nil || selection.Account == nil {
			h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available compatible accounts", streamStarted)
			return
		}

		reqLog.Debug("openai.images.account_schedule_decision",
			zap.String("layer", scheduleDecision.Layer),
			zap.Bool("sticky_session_hit", scheduleDecision.StickySessionHit),
			zap.Int("candidate_count", scheduleDecision.CandidateCount),
			zap.Int("top_k", scheduleDecision.TopK),
			zap.Int64("latency_ms", scheduleDecision.LatencyMs),
			zap.Float64("load_skew", scheduleDecision.LoadSkew),
		)

		account := selection.Account
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		reqLog.Debug("openai.images.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
		setOpsSelectedAccount(c, account.ID, account.Platform)

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, parsed.Stream, &streamStarted, reqLog)
		if !acquired {
			return
		}

		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		setOpenAIFailedUsageExactUpstreamModel(c, resolveOpenAIFailedUsageExactUpstreamModel(account, parsed.Model, channelMapping.MappedModel))
		result, err := h.gatewayService.ForwardImages(c.Request.Context(), c, account, body, parsed, channelMapping.MappedModel)
		forwardDuration := time.Since(forwardStart)
		forwardDurationMs := forwardDuration.Milliseconds()
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		upstreamLatencyMs, _ := getContextInt64(c, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, responseLatencyMs)
		if err == nil && result != nil && result.FirstTokenMs != nil {
			service.SetOpsLatencyMs(c, service.OpsTimeToFirstTokenMsKey, int64(*result.FirstTokenMs))
		}
		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				if failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						reqLog.Warn("openai.images.pool_mode_same_account_retry",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
							zap.Int("retry_limit", retryLimit),
							zap.Int("retry_count", sameAccountRetryCount[account.ID]),
						)
						select {
						case <-c.Request.Context().Done():
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
					h.handleFailoverExhausted(c, failoverErr, streamStarted)
					h.submitOpenAIImagesFailoverFailedUsageLog(c, apiKey, account, parsed, failoverErr, forwardDuration)
					return
				}
				switchCount++
				reqLog.Warn("openai.images.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
			h.submitOpenAIImagesFailedUsageLog(c, apiKey, account, parsed, err, forwardDuration)
			fields := []zap.Field{
				zap.Int64("account_id", account.ID),
				zap.Bool("fallback_error_response_written", wroteFallback),
				zap.Error(err),
			}
			if shouldLogOpenAIForwardFailureAsWarn(c, wroteFallback) {
				reqLog.Warn("openai.images.forward_failed", fields...)
				return
			}
			reqLog.Error("openai.images.forward_failed", fields...)
			return
		}

		if result != nil {
			if account.Type == service.AccountTypeOAuth {
				h.gatewayService.UpdateCodexUsageSnapshotFromHeaders(c.Request.Context(), account.ID, result.ResponseHeaders)
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
		} else {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
		}

		userAgent := c.GetHeader("User-Agent")
		clientIP := ip.GetClientIP(c)
		requestPayloadHash := service.HashUsageRequestPayload(body)
		detailSnapshot := buildOpenAIImagesDetailSnapshot(c, parsed)
		inboundEndpoint := GetInboundEndpoint(c)
		upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
		if parsed.Multipart {
			requestPayloadHash = service.HashUsageRequestPayload([]byte(parsed.StickySessionSeed()))
		}

		h.submitUsageRecordTask(func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
				Result:             result,
				APIKey:             apiKey,
				User:               apiKey.User,
				Account:            account,
				Subscription:       subscription,
				DetailSnapshot:     detailSnapshot,
				InboundEndpoint:    inboundEndpoint,
				UpstreamEndpoint:   upstreamEndpoint,
				UserAgent:          userAgent,
				IPAddress:          clientIP,
				RequestPayloadHash: requestPayloadHash,
				APIKeyService:      h.apiKeyService,
				ChannelUsageFields: channelMapping.ToUsageFields(parsed.Model, result.UpstreamModel),
			}); err != nil {
				logger.L().With(
					zap.String("component", "handler.openai_gateway.images"),
					zap.Int64("user_id", subject.UserID),
					zap.Int64("api_key_id", apiKey.ID),
					zap.Any("group_id", apiKey.GroupID),
					zap.String("model", parsed.Model),
					zap.Int64("account_id", account.ID),
				).Error("openai.images.record_usage_failed", zap.Error(err))
			}
		})

		reqLog.Debug("openai.images.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", switchCount),
		)
		return
	}
}

func isMultipartImagesContentType(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "multipart/form-data")
}

func buildOpenAIImagesDetailSnapshot(c *gin.Context, parsed *service.OpenAIImagesRequest) *middleware2.UsageDetailSnapshot {
	snapshot := middleware2.BuildUsageDetailSnapshot(c)
	if snapshot == nil || parsed == nil || !parsed.Multipart {
		return snapshot
	}

	requestBody, err := json.Marshal(struct {
		Model          string `json:"model"`
		Prompt         string `json:"prompt"`
		Size           string `json:"size"`
		Quality        string `json:"quality"`
		Background     string `json:"background"`
		OutputFormat   string `json:"output_format"`
		Moderation     string `json:"moderation"`
		N              int    `json:"n"`
		HadSourceImage bool   `json:"had_source_image"`
		HadMask        bool   `json:"had_mask"`
	}{
		Model:          parsed.Model,
		Prompt:         parsed.Prompt,
		Size:           parsed.Size,
		Quality:        parsed.Quality,
		Background:     parsed.Background,
		OutputFormat:   parsed.OutputFormat,
		Moderation:     parsed.Moderation,
		N:              parsed.N,
		HadSourceImage: len(parsed.Uploads) > 0,
		HadMask:        parsed.HasMask,
	})
	if err != nil {
		return snapshot
	}

	snapshot.RequestBody = string(requestBody)
	return snapshot
}

func (h *OpenAIGatewayHandler) submitOpenAIImagesFailedUsageLog(c *gin.Context, apiKey *service.APIKey, account *service.Account, parsed *service.OpenAIImagesRequest, err error, duration time.Duration) {
	var upstreamErr service.OpenAIImageUpstreamError
	if errors.As(err, &upstreamErr) && upstreamErr != nil {
		h.submitOpenAIImagesFailedUsageLogWithResponse(
			c,
			apiKey,
			account,
			parsed,
			upstreamErr.OpenAIImageUpstreamStatusCode(),
			upstreamErr.OpenAIImageUpstreamResponseHeaders(),
			upstreamErr.OpenAIImageUpstreamResponseBody(),
			duration,
		)
		return
	}
	h.submitOpenAIImagesFailedUsageLogWithResponse(c, apiKey, account, parsed, 0, nil, nil, duration)
}

func (h *OpenAIGatewayHandler) submitOpenAIImagesFailoverFailedUsageLog(c *gin.Context, apiKey *service.APIKey, account *service.Account, parsed *service.OpenAIImagesRequest, failoverErr *service.UpstreamFailoverError, duration time.Duration) {
	if failoverErr == nil {
		h.submitOpenAIImagesFailedUsageLogWithResponse(c, apiKey, account, parsed, 0, nil, nil, duration)
		return
	}
	h.submitOpenAIImagesFailedUsageLogWithResponse(c, apiKey, account, parsed, failoverErr.StatusCode, failoverErr.ResponseHeaders, failoverErr.ResponseBody, duration)
}

func (h *OpenAIGatewayHandler) submitOpenAIImagesFailedUsageLogWithResponse(c *gin.Context, apiKey *service.APIKey, account *service.Account, parsed *service.OpenAIImagesRequest, upstreamStatusCode int, responseHeaders http.Header, responseBody []byte, duration time.Duration) {
	if c == nil || apiKey == nil || apiKey.User == nil || account == nil || parsed == nil {
		return
	}
	if responseHeaders != nil || responseBody != nil {
		headersText := service.FormatUsageDetailResponseHeadersText(upstreamStatusCode, responseHeaders)
		service.SetUsageResponseSnapshot(c, headersText, string(responseBody))
		service.SetUsageUpstreamResponse(c, upstreamStatusCode, responseHeaders, string(responseBody))
	}
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	detailSnapshot := buildOpenAIImagesDetailSnapshot(c, parsed)
	inboundEndpoint := GetInboundEndpoint(c)
	upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
	upstreamModel := resolveOpenAIFailedUsageUpstreamModel(c, account, parsed.Model)
	h.submitUsageRecordTask(func(ctx context.Context) {
		service.WriteFailedUsageLogBestEffort(ctx, h.gatewayService.UsageLogRepository(), &service.FailedUsageLogInput{
			APIKey:           apiKey,
			User:             apiKey.User,
			Account:          account,
			Model:            parsed.Model,
			UpstreamModel:    upstreamModel,
			Stream:           parsed.Stream,
			InboundEndpoint:  inboundEndpoint,
			UpstreamEndpoint: upstreamEndpoint,
			UserAgent:        userAgent,
			IPAddress:        clientIP,
			DetailSnapshot:   detailSnapshot,
			Duration:         duration,
		}, "handler.openai_gateway.images")
	})
}
