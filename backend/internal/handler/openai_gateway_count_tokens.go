package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ponytail: package-local seam for deterministic spool-read failure tests; do not replace in parallel.
var parseOpenAICountTokensGatewayRequest = service.ParseGatewayRequest

// GrokCountTokens handles Anthropic-compatible count_tokens requests locally.
// The route middleware already authenticates the API key and resolves the
// group; this handler intentionally does not select an account or check billing.
func (h *OpenAIGatewayHandler) GrokCountTokens(c *gin.Context) {
	body, err := readLenientJSONRequestBodyWithPrealloc(c.Request, h.cfg)
	if err != nil {
		if errors.Is(err, service.ErrRequestBodySpool) {
			h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
			return
		}
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.anthropicErrorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	bodyRef := service.NewRequestBodyRef(body)
	parsedReq, err := service.ParseGatewayRequest(bodyRef, domain.PlatformAnthropic)
	if err != nil {
		logRequestBodyParseFailure(requestLogger(c, "handler.openai_gateway.grok_count_tokens"), body, err)
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	if parsedReq.Model == "" {
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	estimated, err := service.EstimateGrokCountTokens(parsedReq.Body.Bytes())
	if err != nil {
		requestLogger(c, "handler.openai_gateway.grok_count_tokens").Warn("grok_count_tokens.local_estimate_failed", zap.Error(err))
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}

	setOpsRequestContext(c, parsedReq.Model, false)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(false, false)))
	c.JSON(http.StatusOK, gin.H{"input_tokens": estimated})
}

// CountTokens handles Anthropic-compatible POST /v1/messages/count_tokens for OpenAI groups.
// It validates billing and routes to an OpenAI token-count bridge without taking concurrency slots
// or recording usage.
func (h *OpenAIGatewayHandler) CountTokens(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.anthropicErrorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.anthropicErrorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.count_tokens",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)

	body, err := readLenientJSONRequestBodyWithPrealloc(c.Request, h.cfg)
	if err != nil {
		if errors.Is(err, service.ErrRequestBodySpool) {
			h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
			return
		}
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.anthropicErrorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}
	coordinator := &requestBodyCoordinator{}
	if err := coordinator.SetEffectiveBytes(body); err != nil {
		h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
		return
	}
	defer coordinator.Cleanup()
	effectiveBody := coordinator.Effective()
	service.SetUsageRequestBody(c, openAIResponsesRequestBodyPreviewSnapshot(effectiveBody))

	bodyRef := service.NewRequestBodyRefFromHandle(effectiveBody)
	parsedReq, err := parseOpenAICountTokensGatewayRequest(bodyRef, domain.PlatformAnthropic)
	if err != nil {
		logRequestBodyParseFailure(reqLog, body, err)
		if errors.Is(err, service.ErrRequestBodySpool) {
			h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
			return
		}
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	if parsedReq.Body.Handle() == nil {
		if err := coordinator.SetEffectiveBytes(parsedReq.Body.Bytes()); err != nil {
			h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
			return
		}
		effectiveBody = coordinator.Effective()
		parsedReq, err = parseOpenAICountTokensGatewayRequest(service.NewRequestBodyRefFromHandle(effectiveBody), domain.PlatformAnthropic)
		if err != nil {
			if errors.Is(err, service.ErrRequestBodySpool) {
				h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
				return
			}
			h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
			return
		}
	}
	cloneGatewayParsedRequestScalars(parsedReq)
	if parsedReq.Model == "" {
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	reqModel := parsedReq.Model
	clientModel := clientRequestedModel(c, reqModel)
	if err := SetClaudeCodeClientContext(c, nil, parsedReq); err != nil {
		h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
		return
	}
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	route, hasRoute := service.EffectiveGatewayRouteFromContext(c.Request.Context())
	if !hasRoute {
		if h.effectiveRouteResolver == nil {
			h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable")
			return
		}
		route, err = h.effectiveRouteResolver.Resolve(c.Request.Context(), apiKey, subscription, nil, clientModel, service.CompositeRouteEndpointCountTokens)
		if err != nil {
			h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "No available accounts: "+err.Error())
			return
		}
		ctx := service.WithEffectiveGatewayRoute(c.Request.Context(), route)
		if route.Decision != nil {
			ctx = service.WithCompositeRouteDecision(service.WithoutCompositeRouteDecision(ctx), *route.Decision)
		}
		c.Request = c.Request.WithContext(ctx)
	}
	if route.APIKey != nil {
		apiKey = route.APIKey
	}
	subscription = route.Subscription
	if route.RoutingModel != "" && route.RoutingModel != reqModel {
		body, err = effectiveBody.ReadAll()
		if err != nil {
			h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
			return
		}
		body = h.gatewayService.ReplaceModelInBody(body, route.RoutingModel)
		if err := coordinator.SetEffectiveBytes(body); err != nil {
			h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
			return
		}
		effectiveBody = coordinator.Effective()
		parsedReq, err = parseOpenAICountTokensGatewayRequest(service.NewRequestBodyRefFromHandle(effectiveBody), domain.PlatformAnthropic)
		if err != nil {
			if errors.Is(err, service.ErrRequestBodySpool) {
				h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
				return
			}
			h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
			return
		}
		cloneGatewayParsedRequestScalars(parsedReq)
	}
	reqModel = parsedReq.Model
	ensureCompositeTargetPlatform(c, apiKey, reqModel)
	if !compositeTargetPlatformAllowed(c, apiKey, reqModel, service.PlatformOpenAI) {
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Model is not supported by this OpenAI-compatible endpoint for composite groups")
		return
	}
	if !allowOpenAICompatibleMessagesDispatch(c.Request.Context(), apiKey) {
		h.anthropicErrorResponse(c, http.StatusForbidden, "permission_error",
			"This group does not allow /v1/messages dispatch")
		return
	}
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}
	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", parsedReq.Stream))

	setOpsRequestContext(c, reqModel, false)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(false, false)))

	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, reqModel)
	route = route.WithChannelMapping(channelMapping)
	c.Request = c.Request.WithContext(service.WithEffectiveGatewayRoute(c.Request.Context(), route))
	if channelMapping.Mapped {
		body, err = effectiveBody.ReadAll()
		if err != nil {
			h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
			return
		}
		body = h.gatewayService.ReplaceModelInBody(body, channelMapping.MappedModel)
		if err := coordinator.SetEffectiveBytes(body); err != nil {
			h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
			return
		}
		effectiveBody = coordinator.Effective()
		parsedReq, err = parseOpenAICountTokensGatewayRequest(service.NewRequestBodyRefFromHandle(effectiveBody), domain.PlatformAnthropic)
		if err != nil {
			if errors.Is(err, service.ErrRequestBodySpool) {
				h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
				return
			}
			h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
			return
		}
		cloneGatewayParsedRequestScalars(parsedReq)
		reqModel = parsedReq.Model
	}
	routingModel := service.NormalizeOpenAICompatRequestedModel(reqModel)
	preferredMappedModel := resolveOpenAIMessagesDispatchMappedModel(apiKey, reqModel)
	body, err = effectiveBody.ReadAll()
	if err != nil {
		h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
		return
	}
	sessionHash := h.gatewayService.GenerateSessionHash(c, body)
	service.BindOpenAIRequestBodyHandle(c, effectiveBody)
	parsedReq = nil

	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		reqLog.Info("openai_count_tokens.billing_eligibility_check_failed", zap.Error(err))
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.anthropicErrorResponse(c, status, code, message)
		return
	}

	requestStart := time.Now()
	// count_tokens 不计费：显式豁免利润门，避免高倍率账号池被门排除后连
	// token 计数都返回 no available accounts。
	c.Request = c.Request.WithContext(service.WithOpenAIProfitControlSuppressed(c.Request.Context()))
	currentRoutingModel := routingModel
	if preferredMappedModel != "" {
		currentRoutingModel = preferredMappedModel
	}
	selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
		c.Request.Context(),
		apiKey.GroupID,
		"",
		sessionHash,
		currentRoutingModel,
		nil,
		service.OpenAIUpstreamTransportAny,
		service.OpenAIEndpointCapabilityChatCompletions,
		false,
		false,
		false,
		openAICompatibleRequestPlatform(c.Request.Context(), apiKey),
	)
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	if err != nil {
		requestPlatform := openAICompatibleRequestPlatform(c.Request.Context(), apiKey)
		reqLog.Warn("openai_count_tokens.account_select_failed", zap.Error(openAICompatibleSelectionErrorForLog(err, requestPlatform)))
		cls := classifyOpenAICompatibleNoAccountErrorFromGin(c, h.gatewayService, apiKey, currentRoutingModel, reqModel)
		if !cls.ModelNotFound {
			markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
		}
		h.anthropicErrorResponse(c, cls.Status, cls.ErrType, cls.Message)
		return
	}
	if selection == nil || selection.Account == nil {
		cls := classifyOpenAICompatibleNoAccountErrorFromGin(c, h.gatewayService, apiKey, currentRoutingModel, reqModel)
		if !cls.ModelNotFound {
			markOpsRoutingCapacityLimited(c)
		}
		h.anthropicErrorResponse(c, cls.Status, cls.ErrType, cls.Message)
		return
	}

	account := selection.Account
	setOpsSelectedAccount(c, account.ID, account.Platform)
	if selection.Acquired && selection.ReleaseFunc != nil {
		defer selection.ReleaseFunc()
	}
	defaultMappedModel := preferredMappedModel

	if err := h.gatewayService.ForwardCountTokensAsAnthropic(c.Request.Context(), c, account, nil, defaultMappedModel); err != nil {
		if errors.Is(err, service.ErrRequestBodySpool) {
			h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
			return
		}
		reqLog.Error("openai_count_tokens.forward_failed", zap.Int64("account_id", account.ID), zap.Error(err))
	}
}
