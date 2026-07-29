---
change: unify-effective-gateway-route-state
design-doc: docs/superpowers/specs/2026-07-28-unify-effective-gateway-route-state-design.md
base-ref: babe29e00f18df9a0011d8464446654148d5eb53
archived-with: 2026-07-29-unify-effective-gateway-route-state
---

# 统一网关有效路由状态 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 以一个 request-scoped effective route snapshot 关闭 staged merge 遗留的 `3 Important + 1 Minor`，让最终 group、计费来源、协议分发、模型阶段和 HTTP bridge later-turn 保持一致。

**Architecture:** 在 service 层提升现有 `effectiveGatewayRoute` 为共享值对象和 resolver；现有 route-local composite middleware 在 endpoint switch 前 resolve/validate/apply，handler 与 runtime fallback 消费同一 snapshot。模型审计和 HTTP bridge 只补各自已确认的单点缺口，不引入新依赖或通用路由框架。

**Tech Stack:** Go 1.25、Gin、现有 service/repository、`testify/require`、Wire、OpenSpec 1.6.0。

## Global Constraints

- 保持 `backend/cmd/server/VERSION` 为 `0.1.159.6`。
- 不修改公开 API、数据库 schema、migration、配置、前端或无关协议。
- 不新增依赖、interface、factory、缓存或独立 WebSocket 生命周期抽象。
- 不恢复 `openai-first-token-timeout`；不重做已关闭的 pricing、100 字符模型边界、Ollama、Grok multipart、body snapshot、copy-source 或 ledger 修复。
- 所有行为变更先写 RED 测试；每个任务完成后运行其聚焦测试。
- follow-up 只运行本地门禁；远程 Docker integration 留给恢复后的原 Task 21。
- 每个任务通过实现与审查后允许使用显式 pathspec 创建本地 commit；不 amend，不 push、tag、release 或 deploy。

---

## File Map

- Create `backend/internal/service/effective_gateway_route.go`: effective route 值对象、billing source、resolver、context/model-stage helpers。
- Create `backend/internal/service/effective_gateway_route_test.go`: 最终 group、subscription、clone、composite 与 identity 单元测试。
- Modify `backend/internal/service/billing_cache_service.go`: 订阅组缺 subscription 时 fail closed。
- Modify `backend/internal/service/billing_cache_service_balance_test.go`: 禁止订阅请求退化为余额测试。
- Modify `backend/internal/service/wire.go`: 注册共享 resolver provider。
- Modify `backend/internal/server/middleware/api_key_auth.go`: 原子应用 effective API key/subscription/request context。
- Modify `backend/internal/server/routes/gateway.go`: 在 protocol switch 前解析 effective route。
- Modify `backend/internal/server/routes/composite_platform_test.go`: route-local 最终 group/context/失败原子性测试。
- Modify `backend/internal/server/routes/gateway_test.go`: 最终平台驱动 Messages/count_tokens dispatch 回归。
- Modify `backend/internal/server/router.go` and `backend/internal/server/http.go`: 透传 resolver 依赖。
- Modify `backend/internal/handler/gateway_handler.go`: 消费 snapshot，并让 prompt-too-long fallback 使用共享 resolver。
- Modify `backend/internal/handler/openai_gateway_handler.go`: 移除 OpenAI 专属 effective group 推导，消费 snapshot。
- Modify `backend/internal/handler/openai_gateway_count_tokens.go`: count_tokens 消费最终状态。
- Modify `backend/internal/handler/gateway_handler_sticky_fallback_test.go`, `backend/internal/handler/openai_gateway_handler_test.go`, `backend/internal/handler/terminal_failed_usage_test.go`: handler/fallback 回归。
- Modify `backend/internal/handler/content_moderation_helper.go` and `backend/internal/handler/composite_platform_test.go`: 修复无 channel mapping 的三阶段审计。
- Modify `backend/internal/service/openai_ws_forwarder_ingress.go` and `backend/internal/service/openai_ws_http_bridge_test.go`: bridge later-turn account mapping。
- Regenerate `backend/cmd/server/wire_gen.go`: 仅由仓库生成命令更新。

---

### Task 1: Effective Route Core And Billing Invariant

**Files:**
- Create: `backend/internal/service/effective_gateway_route.go`
- Create: `backend/internal/service/effective_gateway_route_test.go`
- Modify: `backend/internal/service/billing_cache_service.go:735`
- Modify: `backend/internal/service/billing_cache_service_balance_test.go`

**Interfaces:**
- Produces: `EffectiveGatewayRouteResolver.Resolve(ctx, apiKey, currentSubscription, startGroup, clientModel, endpoint) (EffectiveGatewayRoute, error)`。
- Produces: `WithEffectiveGatewayRoute`, `EffectiveGatewayRouteFromContext`, `WithChannelMapping`, `WithUpstreamModel`。
- Guarantees: `BillingSourceSubscription` implies `Subscription != nil`; changed group gets an API key/User clone and clears stale `UserGroupRPMOverride`。

- [x] **Step 1: Write resolver RED tests**

在新测试文件定义只实现热路径方法的 embedded stubs：

```go
type effectiveRouteGroupRepo struct {
	GroupRepository
	groups map[int64]*Group
}

func (r effectiveRouteGroupRepo) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	group := r.groups[id]
	if group == nil {
		return nil, ErrGroupNotFound
	}
	return group, nil
}

type effectiveRouteSubscriptionRepo struct {
	UserSubscriptionRepository
	subscriptions map[[2]int64]*UserSubscription
}

func (r effectiveRouteSubscriptionRepo) GetActiveByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	sub := r.subscriptions[[2]int64{userID, groupID}]
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	copy := *sub
	return &copy, nil
}
```

增加这些测试：

```go
func TestEffectiveGatewayRouteResolverUsesFinalSubscriptionGroup(t *testing.T)
func TestEffectiveGatewayRouteResolverRejectsUnavailableFinalSubscription(t *testing.T)
func TestEffectiveGatewayRouteResolverRejectsDisallowedFinalStandardGroup(t *testing.T)
func TestEffectiveGatewayRouteResolverSimpleModeSkipsSubscriptionLookup(t *testing.T)
func TestEffectiveGatewayRouteResolverPreservesNonCompositeIdentity(t *testing.T)
```

首个测试必须断言：原 ClaudeCodeOnly group fallback 到 final composite subscription group；`APIKey`、`APIKey.User` 与输入不是同一指针；原输入不变；`UserGroupRPMOverride` 被清空；最终 subscription/group/platform/decision/client/routing/upstream model 正确。

- [x] **Step 2: Run resolver tests and confirm RED**

Workdir: `backend`

```powershell
go test ./internal/service -run '^TestEffectiveGatewayRouteResolver' -count=1
```

Expected: FAIL，因为 `EffectiveGatewayRouteResolver` 尚不存在。

- [x] **Step 3: Implement the shared value and resolver**

实现以下稳定接口；billing source 使用字符串便于测试日志读取，但不进入公开 API：

```go
type EffectiveGatewayBillingSource string

const (
	EffectiveGatewayBillingSimpleSkip  EffectiveGatewayBillingSource = "simple-skip"
	EffectiveGatewayBillingBalance     EffectiveGatewayBillingSource = "balance"
	EffectiveGatewayBillingSubscription EffectiveGatewayBillingSource = "subscription"
)

type EffectiveGatewayRoute struct {
	APIKey        *APIKey
	Group         *Group
	GroupID       *int64
	Subscription  *UserSubscription
	BillingSource EffectiveGatewayBillingSource
	Endpoint      string
	ClientModel   string
	RoutingModel  string
	UpstreamModel string
	Platform      string
	Decision      *CompositeRouteDecision
	Channel       ChannelMappingResult
}

type EffectiveGatewayRouteResolver struct {
	apiKeyService     *APIKeyService
	compositeResolver *CompositeRouteResolver
	cfg               *config.Config
}

func NewEffectiveGatewayRouteResolver(apiKeys *APIKeyService, composite *CompositeRouteResolver, cfg *config.Config) *EffectiveGatewayRouteResolver

func (r *EffectiveGatewayRouteResolver) Resolve(
	ctx context.Context,
	apiKey *APIKey,
	currentSubscription *UserSubscription,
	startGroup *Group,
	clientModel string,
	endpoint string,
) (EffectiveGatewayRoute, error)
```

策略错误使用现有 `internal/pkg/errors.ApplicationError`，确保 route 与 handler 能用同一映射：

```go
var (
	ErrEffectiveGatewayGroupDeleted = infraerrors.Forbidden("GROUP_DELETED", "API Key 所属分组已删除")
	ErrEffectiveGatewayGroupDisabled = infraerrors.Forbidden("GROUP_DISABLED", "API Key 所属分组已停用")
	ErrEffectiveGatewayRouteUnavailable = infraerrors.ServiceUnavailable("NO_AVAILABLE_ACCOUNTS", "No available accounts")
)
```

`Resolve` 按此顺序执行：选取 `startGroup`；调用现有 `ResolveEffectiveGatewayGroup`；验证 final group active/allowed；克隆 API key 与 User；按 SimpleMode/balance/subscription 三态解析 billing；对 final composite group 使用现有 resolver；初始化 `RoutingModel` 与 `UpstreamModel`。final subscription 变化时直接复用 `APIKeyService.userSubRepo`，不新增 cache；只有测量到该冷路径成为瓶颈时再升级。repository/composite 原始错误用 `ApplicationError.WithCause` 包装，resolver 不返回无法稳定映射的裸错误。

模型 helper 必须保持 identity：

```go
func (r EffectiveGatewayRoute) WithChannelMapping(mapping ChannelMappingResult) EffectiveGatewayRoute {
	r.Channel = mapping
	if mapping.Mapped {
		r.RoutingModel = mapping.MappedModel
	}
	if r.RoutingModel == "" {
		r.RoutingModel = mapping.MappedModel
	}
	r.UpstreamModel = r.RoutingModel
	return r
}

func (r EffectiveGatewayRoute) WithUpstreamModel(model string) EffectiveGatewayRoute {
	if model != "" {
		r.UpstreamModel = model
	}
	return r
}

type effectiveGatewayRouteContextKey struct{}

func WithEffectiveGatewayRoute(ctx context.Context, route EffectiveGatewayRoute) context.Context {
	return context.WithValue(ctx, effectiveGatewayRouteContextKey{}, route)
}

func EffectiveGatewayRouteFromContext(ctx context.Context) (EffectiveGatewayRoute, bool) {
	route, ok := ctx.Value(effectiveGatewayRouteContextKey{}).(EffectiveGatewayRoute)
	return route, ok
}
```

- [x] **Step 4: Run resolver tests and confirm GREEN**

```powershell
go test ./internal/service -run '^TestEffectiveGatewayRouteResolver' -count=1
```

Expected: PASS。

- [x] **Step 5: Write missing-subscription billing RED test**

在 unit-tag 测试中增加：

```go
func TestCheckBillingEligibility_RejectsMissingSubscriptionInsteadOfUsingBalance(t *testing.T) {
	cache := &balanceEligibilityCacheStub{balance: 100}
	cfg := &config.Config{}
	svc := NewBillingCacheService(cache, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(svc.Stop)

	group := &Group{ID: 9, SubscriptionType: SubscriptionTypeSubscription}
	err := svc.CheckBillingEligibility(context.Background(), &User{ID: 1}, nil, group, nil, PlatformAnthropic)
	require.ErrorIs(t, err, ErrSubscriptionNotFound)
}
```

运行：

```powershell
go test -tags=unit ./internal/service -run '^TestCheckBillingEligibility_RejectsMissingSubscriptionInsteadOfUsingBalance$' -count=1
```

Expected: FAIL；当前实现会检查余额并返回 nil。

- [x] **Step 6: Fail closed for subscription groups and rerun tests**

在 SimpleMode 和 circuit breaker 判断之后、余额/订阅分支之前增加：

```go
isSubscriptionMode := group != nil && group.IsSubscriptionType()
if isSubscriptionMode && subscription == nil {
	return ErrSubscriptionNotFound
}
```

后续分支只使用 `isSubscriptionMode`，不再用 `subscription != nil` 推断计费模式。重跑 Step 5 命令，Expected: PASS。

---

### Task 2: Pre-dispatch Apply And Handler Consumption

**Files:**
- Modify: `backend/internal/server/middleware/api_key_auth.go`
- Modify: `backend/internal/server/routes/gateway.go`
- Modify: `backend/internal/server/routes/composite_platform_test.go`
- Modify: `backend/internal/server/routes/gateway_test.go`
- Modify: `backend/internal/server/router.go`
- Modify: `backend/internal/server/http.go`
- Modify: `backend/internal/handler/gateway_handler.go`
- Modify: `backend/internal/handler/openai_gateway_handler.go`
- Modify: `backend/internal/handler/openai_gateway_count_tokens.go`
- Modify: `backend/internal/handler/gateway_handler_sticky_fallback_test.go`
- Modify: `backend/internal/handler/openai_gateway_handler_test.go`
- Modify: `backend/internal/service/wire.go`
- Regenerate: `backend/cmd/server/wire_gen.go`

**Interfaces:**
- Consumes: Task 1 `EffectiveGatewayRouteResolver` and context helpers。
- Produces: `middleware.ApplyEffectiveGatewayRoute(c, route)` as the only Gin/request auth-state writer after APIKeyAuth。
- Produces: route-local middleware resolves before `getGroupPlatform` protocol switch。

- [x] **Step 1: Write route-local RED tests**

在 `server/routes/composite_platform_test.go` 增加：

```go
func TestCompositeTargetPlatformMiddlewareAppliesFallbackBeforeProtocolDispatch(t *testing.T)
func TestCompositeTargetPlatformMiddlewareLoadsFinalSubscription(t *testing.T)
func TestCompositeTargetPlatformMiddlewareDoesNotApplyFailedRoute(t *testing.T)
```

在 `server/routes/gateway_test.go` 增加 `TestGatewayRoutesMessagesDispatchesUsingEffectiveFallbackPlatform`；在 handler 测试增加 `TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot`、`TestOpenAIGatewayResponsesUsesEffectiveRouteSnapshot`、`TestOpenAIGatewayMessagesUsesEffectiveRouteSnapshot` 和 `TestOpenAIGatewayCountTokensUsesEffectiveRouteSnapshot`。

测试 handler 内至少断言：

```go
effectiveKey, ok := servermiddleware.GetAPIKeyFromContext(c)
require.True(t, ok)
require.Equal(t, finalGroup.ID, *effectiveKey.GroupID)
require.Same(t, finalGroup, effectiveKey.Group)

subscription, ok := servermiddleware.GetSubscriptionFromContext(c)
require.True(t, ok)
require.Equal(t, finalGroup.ID, subscription.GroupID)

contextGroup, ok := c.Request.Context().Value(ctxkey.Group).(*service.Group)
require.True(t, ok)
require.Equal(t, finalGroup.ID, contextGroup.ID)

// OpsErrorLoggerMiddleware 优先读取同一个正式 API key。
require.Equal(t, service.PlatformOpenAI, getGroupPlatform(c))
```

失败测试在 auth middleware 返回后读取 Gin key，断言仍指向 original group，且 endpoint handler 未执行。

- [x] **Step 2: Run route tests and confirm RED**

```powershell
go test ./internal/server/routes -run '^TestCompositeTargetPlatformMiddleware(AppliesFallbackBeforeProtocolDispatch|LoadsFinalSubscription|DoesNotApplyFailedRoute)$' -count=1
```

Expected: FAIL；当前 middleware 只处理 original composite group。

- [x] **Step 3: Add the single Apply helper**

在 `api_key_auth.go` 增加：

```go
func ApplyEffectiveGatewayRoute(c *gin.Context, route service.EffectiveGatewayRoute) {
	if c == nil || c.Request == nil || route.APIKey == nil {
		return
	}
	c.Set(string(ContextKeyAPIKey), route.APIKey)
	if route.Subscription == nil {
		c.Set(string(ContextKeySubscription), nil)
	} else {
		c.Set(string(ContextKeySubscription), route.Subscription)
	}

	ctx := service.WithoutCompositeRouteDecision(c.Request.Context())
	if route.Group != nil {
		ctx = context.WithValue(ctx, ctxkey.Group, route.Group)
	}
	if route.Decision != nil {
		ctx = service.WithCompositeRouteDecision(ctx, *route.Decision)
	}
	ctx = service.WithEffectiveGatewayRoute(ctx, route)
	c.Request = c.Request.WithContext(ctx)
}
```

不写第二个 handler-local Apply helper。

- [x] **Step 4: Resolve before the route switch**

把 `compositeTargetPlatformMiddleware` 改为接收 `*service.EffectiveGatewayRouteResolver`。仅当 original group 是 composite 或 `ClaudeCodeOnly` 时读取 body；GET 只做 UA 检测。顺序固定为：

```go
var body []byte
if c.Request != nil && c.Request.Method != http.MethodGet {
	var err error
	body, err = pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		status := http.StatusBadRequest
		message := "Failed to read request body"
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			status = http.StatusRequestEntityTooLarge
			message = "Request body is too large"
		}
		c.JSON(status, gin.H{"error": gin.H{"type": "invalid_request_error", "message": message}})
		c.Abort()
		return
	}
}
handler.SetClaudeCodeClientContext(c, body, nil)
clientModel := compositeRequestModelFromBody(c.GetHeader("Content-Type"), body)
subscription, _ := middleware.GetSubscriptionFromContext(c)
route, err := resolver.Resolve(
	c.Request.Context(), apiKey, subscription, nil,
	clientModel, compositeRouteEndpointForPath(c.Request.URL.Path),
)
if err != nil {
	middleware.AbortWithError(c, pkgerrors.Code(err), pkgerrors.Reason(err), pkgerrors.Message(err))
	c.Abort()
	return
}
middleware.ApplyEffectiveGatewayRoute(c, route)
if route.RoutingModel != "" && route.RoutingModel != clientModel && gjson.ValidBytes(body) {
	if rewritten, rewriteErr := sjson.SetBytes(body, "model", route.RoutingModel); rewriteErr == nil {
		body = rewritten
	}
}
if c.Request.Method != http.MethodGet {
	resetRequestBody(c, body)
}
```

沿用当前 body limit、multipart snapshot 和模型长度错误响应；不得重新实现 parser。

- [x] **Step 5: Make handlers consume the snapshot**

在两个 handler struct 末尾增加 `effectiveRouteResolver *service.EffectiveGatewayRouteResolver`，构造器末尾追加该参数。删除 `resolveEffectiveOpenAIGatewayRoute`；Responses、Messages 和 count_tokens 优先读取：

```go
route, ok := service.EffectiveGatewayRouteFromContext(c.Request.Context())
if ok {
	apiKey = route.APIKey
	clientModel = route.ClientModel
	reqModel = route.RoutingModel
}
```

channel mapping 后调用并写回 request context：

```go
route = route.WithChannelMapping(mapping)
c.Request = c.Request.WithContext(service.WithEffectiveGatewayRoute(c.Request.Context(), route))
```

无 route snapshot 的 direct/forced tests 使用注入的 resolver 构造 identity/final state，不恢复两套独立解析逻辑。删除 handler-local `effectiveGatewayRoute` 与最后一个调用点消失后的 `cloneAPIKeyWithGroup`。

- [x] **Step 6: Wire the resolver through server and handlers**

在 `service.ProviderSet` 加入：

```go
NewEffectiveGatewayRouteResolver,
```

向 `ProvideRouter`、`SetupRouter`、`registerRoutes`、`RegisterGatewayRoutes`、`NewGatewayHandler`、`NewOpenAIGatewayHandler` 透传 `*service.EffectiveGatewayRouteResolver`。更新所有测试构造器调用，未使用 resolver 的测试传 `nil`。

运行：

```powershell
make -C backend generate
$before = git diff --binary | git hash-object --stdin
make -C backend generate
$after = git diff --binary | git hash-object --stdin
if ($before -ne $after) { throw "backend generate is not stable" }
```

Expected: 第一次更新 `wire_gen.go`；第二次不再改变 tracked diff；不要手改生成文件。

- [x] **Step 7: Run route and handler GREEN tests**

Workdir: `backend`

```powershell
go test ./internal/server/routes -run '^(TestCompositeTargetPlatformMiddleware|TestGatewayRoutesMessagesDispatchesUsingEffectiveFallbackPlatform|TestGatewayRoutesComposite)' -count=1
go test ./internal/handler -run '^(TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot|TestOpenAIGatewayResponsesUsesEffectiveRouteSnapshot|TestOpenAIGatewayMessagesUsesEffectiveRouteSnapshot|TestOpenAIGatewayCountTokensUsesEffectiveRouteSnapshot)$' -count=1
```

Expected: PASS；新增 dispatch test 证明 `getGroupPlatform` 看到 final platform，identity tests 证明普通 group 不改变。

---

### Task 3: Atomic Prompt-too-long Fallback

**Files:**
- Modify: `backend/internal/handler/gateway_handler.go:1115`
- Modify: `backend/internal/handler/terminal_failed_usage_test.go:865`

**Interfaces:**
- Consumes: shared resolver and `middleware.ApplyEffectiveGatewayRoute`。
- Produces: prompt-too-long candidate uses final group subscription and only applies after eligibility succeeds。

- [x] **Step 1: Extend the existing fallback test to RED**

扩展 `TestGatewayHandler_MessagesPromptTooLongFallbackResolvesClaudeCodeOnlyGroup`，新增 final group 为 subscription 的 subtest：

```go
finalGroup.SubscriptionType = service.SubscriptionTypeSubscription
```

给 APIKeyService 的 subscription repo 返回 `UserSubscription{UserID: env.apiKey.UserID, GroupID: finalID, Status: service.StatusActive}`；billing cache 的 `GetUserBalance` 在 fallback 调用时返回 0，并让 subscription cache 返回未超限数据。断言：响应成功、第二次账号来自 `finalID`、余额读取次数没有增加、usage 使用 final subscription。

使用该 APIKeyService、测试 composite resolver 和现有 config 构造 `service.NewEffectiveGatewayRouteResolver(...)`，赋给 `env.handler.effectiveRouteResolver`；不能在测试中绕过 production resolver。

新增同级测试 `TestGatewayHandler_MessagesPromptTooLongFallbackRejectsMissingFinalSubscriptionAtomically`，断言 403、没有第二次账号调度、Gin API key/request group 仍是应用前状态。

- [x] **Step 2: Run fallback tests and confirm RED**

```powershell
go test ./internal/handler -run '^TestGatewayHandler_MessagesPromptTooLongFallback(ResolvesClaudeCodeOnlyGroup|RejectsMissingFinalSubscriptionAtomically)$' -count=1
```

Expected: FAIL；当前代码先校验 intermediate group，并以 `subscription=nil` 检查 final group。

- [x] **Step 3: Replace the local fallback state machine**

删除 lines 1141-1157 的 intermediate subscription/fallback-chain 限制、局部 clone，以及 lines 1186/1225 的 `nil` 计费/usage 赋值。改为：

```go
candidate, err := h.effectiveRouteResolver.Resolve(
	service.WithoutCompositeRouteDecision(c.Request.Context()),
	apiKey,
	currentSubscription,
	fallbackGroup,
	clientRequestModel,
	service.CompositeRouteEndpointMessages,
)
if err != nil {
	submitPromptTooLongFailedUsage()
	h.handleStreamingAwareError(
		c,
		pkgerrors.Code(err),
		pkgerrors.Reason(err),
		pkgerrors.Message(err),
		streamStarted,
	)
	return
}

mapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(
	c.Request.Context(), candidate.GroupID, candidate.RoutingModel,
)
candidate = candidate.WithChannelMapping(mapping)
if err := h.billingCacheService.CheckBillingEligibility(
	c.Request.Context(), candidate.APIKey.User, candidate.APIKey,
	candidate.Group, candidate.Subscription, candidate.Platform,
); err != nil {
	status, code, message, retryAfter := billingErrorDetails(err)
	if retryAfter > 0 {
		c.Header("Retry-After", strconv.Itoa(retryAfter))
	}
	submitPromptTooLongFailedUsage()
	h.handleStreamingAwareError(c, status, code, message, streamStarted)
	return
}

middleware2.ApplyEffectiveGatewayRoute(c, candidate)
currentAPIKey = candidate.APIKey
currentSubscription = candidate.Subscription
currentStickyGroupID = candidate.GroupID
platform = candidate.Platform
```

在 Apply 后才更新 coordinator effective bytes、并发 slot、sticky key 和调度变量。

- [x] **Step 4: Run fallback tests and existing sticky cleanup regressions**

```powershell
go test ./internal/handler -run '^TestGatewayHandler_MessagesPromptTooLongFallback' -count=1
```

Expected: PASS，包括现有 non-Claude/Claude、billing rejection、Gemini sticky cleanup 和 failed usage 测试。

---

### Task 4: Preserve Concrete Routing Model In Audit

**Files:**
- Modify: `backend/internal/handler/content_moderation_helper.go:44`
- Modify: `backend/internal/handler/composite_platform_test.go:80`

**Interfaces:**
- Consumes: `ChannelMappingResult.ToUsageFields` identity behavior。
- Produces: `OriginalModel=client alias`、`ChannelMappedModel=routing model`、`UpstreamModel=account/upstream model`。

- [x] **Step 1: Change the incorrect assertion to the required RED behavior**

将 `TestClientRequestedModelUsesCompositePublicModel` 中 usage 部分拆为：

```go
func TestClientRequestedUsageFieldsPreservesConcreteCompositeRouteWithoutChannelMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request = c.Request.WithContext(service.WithCompositeRouteDecision(
		c.Request.Context(),
		service.CompositeRouteDecision{
			Matched:        true,
			Source:         service.CompositeRouteSourceExplicit,
			PublicModel:    "public-alias",
			TargetPlatform: service.PlatformOpenAI,
			UpstreamModel:  "gpt-5",
		},
	))

	fields := clientRequestedUsageFields(
		c,
		service.ChannelMappingResult{MappedModel: "gpt-5"},
		"gpt-5",
		"gpt-5.2",
	)
	require.Equal(t, "public-alias", fields.OriginalModel)
	require.Equal(t, "gpt-5", fields.ChannelMappedModel)
	require.Equal(t, "public-alias→gpt-5→gpt-5.2", fields.ModelMappingChain)
}
```

- [x] **Step 2: Run the test and confirm RED**

```powershell
go test ./internal/handler -run '^TestClientRequestedUsageFieldsPreservesConcreteCompositeRouteWithoutChannelMapping$' -count=1
```

Expected: FAIL；当前 `ChannelMappedModel` 为 `public-alias`。

- [x] **Step 3: Delete the alias overwrite and rerun protections**

从 `clientRequestedUsageFields` 删除：

```go
if !mapping.Mapped {
	fields.ChannelMappedModel = requestedModel
}
```

运行：

```powershell
go test ./internal/handler -run '^(TestClientRequestedUsageFieldsPreservesCompositeModelTriplet|TestClientRequestedUsageFieldsPreservesConcreteCompositeRouteWithoutChannelMapping|TestOpenAIGatewayHandler_SubmitFailedUsageLog_PreservesCompositeModelTriplet)$' -count=1
```

Expected: PASS；pricing 测试证明没有重做已关闭的 concrete/explicit alias 定价优先级。

---

### Task 5: HTTP Bridge Later-turn Account Mapping

**Files:**
- Modify: `backend/internal/service/openai_ws_forwarder_ingress.go:471`
- Modify: `backend/internal/service/openai_ws_http_bridge_test.go:1057`

**Interfaces:**
- Consumes: existing `applyOpenAIWSRequestRewrite` and `applyOpenAIWSAccountModelMapping`。
- Keeps: `OpenAIWSRequestRewrite` and `OpenAIWSIngressHooks.RewriteRequest` signatures unchanged。

- [x] **Step 1: Write the two-turn bridge RED test**

将现有 `TestOpenAIWSHTTPBridgeKeepsContinuationFramesOnHTTPWithoutPreviousResponseID` 重命名为：

```go
func TestOpenAIWSHTTPBridgeKeepsContinuationAndAppliesLaterAccountMapping(t *testing.T)
```

账号加入：

```go
account.Credentials["model_mapping"] = map[string]any{
	"routed-model": "account-model",
}
```

先声明结果槽并传入 hooks：

```go
var secondResult *OpenAIForwardResult
hooks := &OpenAIWSIngressHooks{
	RewriteRequest: func(turn int, payload []byte, originalModel string) (OpenAIWSRequestRewrite, error) {
		if turn != 2 {
			return OpenAIWSRequestRewrite{Payload: payload, OriginalModel: originalModel}, nil
		}
		return OpenAIWSRequestRewrite{
			Payload:       ReplaceModelInBody(payload, "routed-model"),
			OriginalModel: "routed-model",
		}, nil
	},
	AfterTurn: func(turn int, result *OpenAIForwardResult, _ error) {
		if turn == 2 && result != nil {
			copy := *result
			secondResult = &copy
		}
	},
}
```

把 `ProxyResponsesWebSocketFromClient` 的最后一个参数从 `nil` 改为 `hooks`。保留原 continuation/replay 断言，并追加：

```go
require.Equal(t, "account-model", gjson.GetBytes(upstream.bodies[1], "model").String())
require.NotNil(t, secondResult)
require.Equal(t, "account-model", secondResult.UpstreamModel)
```

- [x] **Step 2: Run the bridge test and confirm RED**

```powershell
go test ./internal/service -run '^TestOpenAIWSHTTPBridgeKeepsContinuationAndAppliesLaterAccountMapping$' -count=1
```

Expected: FAIL；实际 body 仍为 `routed-model`。

- [x] **Step 3: Apply the existing account mapper after rewrite**

在 bridge `turn > 1` 分支中，紧跟 `applyOpenAIWSRequestRewrite`：

```go
rewritten = applyOpenAIWSAccountModelMapping(account, rewritten)
currentBridgePayload.payloadRaw = rewritten
currentBridgePayload.rawForHash = rewritten
currentBridgePayload.payloadBytes = len(rewritten)
currentBridgePayload.originalModel = originalModel
```

不修改 parser、relay、HTTP/SSE conversion、replay、`OpenAIWSRequestRewrite` 或 handler provider-affinity callback。

- [x] **Step 4: Run bridge and provider-affinity regressions**

```powershell
go test ./internal/service -run '^(TestOpenAIWSHTTPBridgeKeepsContinuationAndAppliesLaterAccountMapping|TestPassthroughLifecycle_AppliesAccountMappingAfterLaterRequestRewrite)$' -count=1
go test ./internal/handler -run '^(TestOpenAIResponsesWebSocketAppliesAccountMappingAfterLaterCompositeRoute|TestOpenAIResponsesWebSocketRejectsLaterCrossProviderCompositeRoute)$' -count=1
```

Expected: PASS；普通 WS 和跨 provider 行为保持原样。

---

### Task 6: Full Local Gate And Fresh Review

**Files:**
- Modify: `openspec/changes/unify-effective-gateway-route-state/tasks.md`
- Modify after successful review: `openspec/changes/unify-effective-gateway-route-state/.comet/subagent-progress.md`
- Do not modify yet: original staged merge progress/ledger; cross-change evidence is written only after follow-up archive。

**Interfaces:**
- Consumes: Tasks 1-5 completed implementation。
- Produces: local gate evidence and a fresh review verdict that explicitly closes original `3 Important + 1 Minor`。

- [x] **Step 1: Run all focused tests**

Workdir: `backend`

```powershell
go test ./internal/service -run '^(TestEffectiveGatewayRouteResolver|TestOpenAIWSHTTPBridgeKeepsContinuationAndAppliesLaterAccountMapping|TestPassthroughLifecycle_AppliesAccountMappingAfterLaterRequestRewrite|TestOpenAIGatewayServiceRecordUsage_CompositePublicAliasPricing)' -count=1
go test -tags=unit ./internal/service -run '^TestCheckBillingEligibility_RejectsMissingSubscriptionInsteadOfUsingBalance$' -count=1
go test ./internal/server/routes -run '^(TestCompositeTargetPlatformMiddleware|TestGatewayRoutesComposite|TestGatewayRoutesMessagesDispatchesUsingEffectiveFallbackPlatform)' -count=1
go test ./internal/handler -run '^(TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot|TestGatewayHandler_MessagesPromptTooLongFallback|TestClientRequestedUsageFields|TestOpenAIResponsesWebSocket|TestOpenAIGateway)' -count=1
```

Expected: all commands exit 0。

- [x] **Step 2: Verify generated code stability**

Workdir: repository root

```powershell
make -C backend generate
$before = git diff --binary | git hash-object --stdin
make -C backend generate
$after = git diff --binary | git hash-object --stdin
if ($before -ne $after) { throw "backend generate is not stable" }
git diff --check
```

Expected: second generate introduces no additional diff；`git diff --check` exits 0。

- [x] **Step 3: Run local full gates**

```powershell
make test
make build
openspec validate "unify-effective-gateway-route-state" --strict
```

Expected: all commands exit 0；不运行远程 Docker integration。

Actual: `make build`、strict OpenSpec 和独立前端完整门禁通过；`make test` 的唯一失败为 `TestPassthroughLifecycle_LeaseLossSendsRetryClose`。该失败已在未含本地改动的上游 `v0.1.165` tag 上以 `-count=20` 复现，用户明确要求作为基线例外跳过；不得把该命令记录为 exit 0。

- [x] **Step 4: Dispatch fresh review**

Reviewer scope must include the complete follow-up diff and these required verdict lines:

```text
Important 1: final group authority across middleware subscription, protocol dispatch, Gin API key, scheduler/billing and Ops
Important 2: prompt-too-long resolves final group before validation and never bills a subscription group with nil subscription
Important 4: HTTP bridge later frame performs account mapping after route rewrite and retains provider affinity
Minor: no channel mapping preserves concrete routing model in requested/channel-mapped/upstream audit stages
New Critical/Important findings: none
```

Any open Critical/Important keeps this change blocked. Record command evidence, reviewer session ID and verdict in follow-up progress; do not consume or reopen the old change's exhausted review budget。

- [x] **Step 5: Mark OpenSpec implementation tasks complete only from evidence**

逐项更新 `openspec/changes/unify-effective-gateway-route-state/tasks.md`。只有对应 RED/GREEN、full gate 和 reviewer evidence 已存在的 checkbox 才能从 `[ ]` 改为 `[x]`；不得批量预勾选。
