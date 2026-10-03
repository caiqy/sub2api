package handler

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// inflightReservationEstimator 估算单请求在途预留金额（USD）；false 表示无法定价。
type inflightReservationEstimator interface {
	EstimateInflightReservation(ctx context.Context, apiKey *service.APIKey, req service.InflightEstimateRequest) (float64, bool)
}

// requestMaxOutputTokens 从请求体中提取输出 token 上限（兼容 Anthropic / OpenAI Chat / Responses / Gemini）。
func requestMaxOutputTokens(body []byte) int {
	for _, path := range []string{"max_tokens", "max_completion_tokens", "max_output_tokens", "generationConfig.maxOutputTokens", "generation_config.max_output_tokens"} {
		if v := gjson.GetBytes(body, path); v.Exists() && v.Type == gjson.Number && v.Int() > 0 {
			return int(v.Int())
		}
	}
	return 0
}

// tokenInflightEstimate 文本类请求的估算输入。
func tokenInflightEstimate(model string, body []byte) service.InflightEstimateRequest {
	reasoningEffort := ""
	if effort := service.CanonicalRequestedReasoningEffort(body, model); effort != nil {
		reasoningEffort = *effort
	} else {
		for _, path := range []string{"reasoning.effort", "reasoning_effort", "output_config.effort"} {
			if strings.EqualFold(strings.TrimSpace(gjson.GetBytes(body, path).String()), "none") {
				reasoningEffort = "none"
				break
			}
		}
	}
	tier := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "service_tier").String()))
	switch tier {
	case "priority", "fast", "ultrafast", "flex":
	default:
		tier = "" // All other values use the base price; retain no unbounded request scalar.
	}
	return service.InflightEstimateRequest{
		Model:           strings.Clone(model),
		BodyBytes:       len(body),
		MaxTokens:       requestMaxOutputTokens(body),
		Kind:            service.InflightEstimateToken,
		ReasoningEffort: reasoningEffort,
		ServiceTier:     strings.Clone(tier),
	}
}

func inflightNoop() {}

const httpInflightEstimateKey = "http_inflight_estimate"
const httpInflightHolderKey = "http_inflight_holder"

// Re-estimate after account selection without retaining or re-reading the body.
// Source selection matches the usage billing baseline; an explicit public price
// still wins for composite routes, as in the per-turn WebSocket path.
func reserveInflightBalanceForAccount(c *gin.Context, billing *service.BillingCacheService, estimator inflightReservationEstimator, apiKey *service.APIKey, subscription *service.UserSubscription, mapping service.ChannelMappingResult, upstreamModel string) error {
	value, exists := c.Get(httpInflightEstimateKey)
	if !exists {
		return nil
	}
	req, ok := value.(service.InflightEstimateRequest)
	if !ok {
		return service.ErrBillingServiceUnavailable
	}
	requested := clientRequestedModel(c, req.Model)
	if apiKey.Group != nil && apiKey.Group.Platform == service.PlatformComposite && mapping.BillingModelSource == service.BillingModelSourceRequested {
		mapping.BillingModelSource = service.BillingModelSourceUpstream
	}
	req.Model = strings.Clone(openAIWSTurnBillingModel(nil, mapping, requested, upstreamModel))
	req.UpstreamModel = strings.Clone(upstreamModel)
	req.RequestedModel = strings.Clone(requested)
	_, err := reserveInflightBalance(c, billing, estimator, apiKey, subscription, req)
	if err != nil {
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalPolicyDenied)
		service.SetOpsUpstreamAttempted(c, false)
	}
	return err
}

// reserveInflightBalance 在 CheckBillingEligibility 之后为余额模式请求登记在途预留。
//
// 成功时把预留句柄挂到 c.Request 的 context 上：之后通过 submit*UsageRecordTask 提交的
// 计费任务会接管一个引用，直到余额缓存实际扣减后才释放。返回的 done 必须 defer 调用
// （停止续期并归还 handler 引用；没有提交计费任务时即刻释放）。
// 开关关闭、订阅模式、Redis 故障时返回 no-op（fail-open）；无法定价时默认 fail-open，
// 配置 fail_closed_on_unpriced=true 时返回 ErrInsufficientBalance。
func reserveInflightBalance(
	c *gin.Context,
	billing *service.BillingCacheService,
	estimator inflightReservationEstimator,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	req service.InflightEstimateRequest,
) (func(), error) {
	if c == nil || c.Request == nil {
		return inflightNoop, nil
	}
	if billing == nil || estimator == nil || apiKey == nil || apiKey.User == nil || !billing.InflightReservationEnabled() {
		return inflightNoop, nil
	}
	value, exists := c.Get(httpInflightHolderKey)
	var holder *atomic.Pointer[service.InflightReservation]
	if exists {
		var ok bool
		holder, ok = value.(*atomic.Pointer[service.InflightReservation])
		if !ok || holder == nil {
			return inflightNoop, service.ErrBillingServiceUnavailable
		}
	} else {
		holder = &atomic.Pointer[service.InflightReservation]{}
		if req.PricingAt.IsZero() {
			req.PricingAt = service.OpenAIPricingAtFromContext(c.Request.Context())
		}
		if req.PricingAt.IsZero() {
			req.PricingAt = service.GatewayTokenRequestPricingAtFromContext(c.Request.Context())
		}
		c.Set(httpInflightHolderKey, holder)
		c.Set(httpInflightEstimateKey, req)
	}
	parent := service.WithInflightReservationHolder(c.Request.Context(), holder)
	ctx, _, err := reserveInflightBalanceCtx(parent, billing, estimator, apiKey, subscription, req)
	if err != nil {
		return inflightNoop, err
	}
	holder.Store(service.InflightReservationFromContext(ctx))
	c.Request = c.Request.WithContext(service.WithInflightReservationHolder(ctx, holder))
	return func() { holder.Load().HandlerDone() }, nil
}

// reserveInflightBalanceCtx 同 reserveInflightBalance，但返回携带预留句柄的新 context
// （供 WebSocket 等自管 context 的路径）。
func reserveInflightBalanceCtx(
	ctx context.Context,
	billing *service.BillingCacheService,
	estimator inflightReservationEstimator,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	req service.InflightEstimateRequest,
) (context.Context, func(), error) {
	if billing == nil || estimator == nil || apiKey == nil || apiKey.User == nil || !billing.InflightReservationEnabled() {
		return ctx, inflightNoop, nil
	}
	if apiKey.Group != nil && apiKey.Group.IsSubscriptionType() && subscription != nil {
		return ctx, inflightNoop, nil
	}
	estimate, priced := estimator.EstimateInflightReservation(ctx, apiKey, req)
	if !priced && billing.InflightReservationFailClosedOnUnpriced() {
		return ctx, inflightNoop, service.ErrInsufficientBalance
	}
	if estimate <= 0 {
		return ctx, inflightNoop, nil
	}
	res, err := billing.ReserveInflight(ctx, apiKey.User, apiKey.Group, subscription, estimate)
	if err != nil {
		return ctx, inflightNoop, err
	}
	if res == nil {
		return ctx, inflightNoop, nil
	}
	return service.WithInflightReservation(ctx, res), res.HandlerDone, nil
}

// grokMediaInflightEstimate 媒体生成请求的估算输入；状态/内容查询返回空模型（不预留：
// 查询会为已生成的媒体计费，不能因余额预留而拦截用户取回已付费结果）。
func grokMediaInflightEstimate(endpoint service.GrokMediaEndpoint, model string, info service.GrokMediaRequestInfo, body []byte) service.InflightEstimateRequest {
	if !endpoint.IsGenerationRequest() {
		return service.InflightEstimateRequest{}
	}
	switch endpoint {
	case service.GrokMediaEndpointImagesGenerations, service.GrokMediaEndpointImagesEdits:
		return service.InflightEstimateRequest{Model: model, BodyBytes: len(body), Kind: service.InflightEstimateImage, Units: info.N}
	default:
		return service.InflightEstimateRequest{
			Model:                model,
			BodyBytes:            len(body),
			Kind:                 service.InflightEstimateVideo,
			Units:                1,
			VideoResolution:      info.Resolution,
			VideoDurationSeconds: info.DurationSeconds,
		}
	}
}

// grokVoiceSTTBytesPerSecond STT 时长粗估（~128kbps 压缩音频）。
const grokVoiceSTTBytesPerSecond = 16000

// grokVoiceInflightEstimate 语音 HTTP 接口估算：TTS 按输入字符数（百万字符），STT 按音频字节粗估时长（小时）。
// 其他接口（custom-voices）无音频计量，返回的估算为 0（不预留）。
func grokVoiceInflightEstimate(endpoint string, body []byte) service.InflightEstimateRequest {
	req := service.InflightEstimateRequest{Model: endpoint, Kind: service.InflightEstimateAudio, AudioMode: endpoint}
	switch endpoint {
	case "tts":
		req.AudioUnits = float64(len([]rune(extractGrokTTSInputText(body)))) / 1e6
	case "stt":
		req.AudioUnits = float64(len(body)) / grokVoiceSTTBytesPerSecond / 3600
	}
	return req
}
