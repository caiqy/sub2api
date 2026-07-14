---
change: restore-platform-sticky-routing
design-doc: docs/superpowers/specs/2026-07-14-platform-sticky-boundaries-design.md
base-ref: d2bb23709d928a6e258cc2aae4a2c5dc6cf69fea
archived-with: 2026-07-14-restore-platform-sticky-routing
---

# Platform Sticky Boundaries Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 OpenAI、Gemini、Anthropic/Antigravity 的 Sticky 开关在调度、HTTP、WebSocket、Ingress 和 compat 路径中一致地控制绑定状态。

**Architecture:** 各 service 保持现有配置依赖，不引入共享服务。Gateway 继续使用平台 helper；OpenAI WS 使用返回 `nil` 的受控 state-store accessor；Gemini compat 在解析最终平台后一次决定是否执行会话缓存逻辑。

**Tech Stack:** Go 1.26.5、Gin、Testify、现有 `GatewayCache`、`OpenAIWSStateStore`、`config.GatewayControlRuntime`。

## Global Constraints

- 不新增依赖、公开 API、数据库 schema 或配置字段。
- 缺失 service/config 时 Sticky 默认开启。
- OpenAI Sticky 关闭时 HTTP、WS V2、WS ingress 的 response-account、response-connection、turn state 和 session-connection 全部零读写。
- Gemini Messages compat 按最终解析平台遵守对应开关。
- 默认开启时保留既有状态读写、模型路由和 retry 语义。
- 不通过删除断言或放宽安全语义让测试通过。

---

### Task 1: 恢复调度、HTTP 与 Antigravity 的平台边界

**Files:**
- Modify: `backend/internal/service/gateway_service.go`
- Modify: `backend/internal/service/gateway_scheduling.go`
- Modify: `backend/internal/service/openai_gateway_scheduling.go`
- Modify: `backend/internal/service/openai_gateway_response_handling.go`
- Modify: `backend/internal/service/antigravity_gateway_retry.go`
- Test: `backend/internal/service/gateway_multiplatform_test.go`
- Test: `backend/internal/service/scheduler_snapshot_hydration_test.go`
- Test: `backend/internal/service/openai_gateway_service_test.go`
- Test: `backend/internal/service/antigravity_smart_retry_test.go`

**Interfaces:**
- Consumes: `config.GatewayControlRuntime()`、`GatewayCache`、`OpenAIWSStateStore`。
- Produces: `GatewayService.stickyEnabledForPlatform`、`getCachedSessionAccountIDForPlatform`、`bindStickySessionForPlatform` 与单一的 hydration release 所有权。

- [x] **Step 1: 固定现有回归为 RED**

```powershell
go test -tags=unit ./internal/service -run "^(TestGatewayService_GenericStickyHelpersRespectToggle|TestGatewayService_SelectAccountForModelWithPlatform_StickyDisabledBypassesStickyReadAndWrite|TestGatewayService_StickyDisabledBypassLogs|TestGatewayService_SelectAccountWithLoadAwareness|TestGatewayServiceNewSelectionResult_ReleasesAcquiredSlotWhenHydrationFails|TestOpenAINewAcquiredSelectionResult_ReleasesSlotWhenHydrationFails|TestOpenAIGatewayService_BindHTTPResponseAccountSkipsDisabledSticky|TestAntigravityGatewayServiceClearStickySessionSkipsDisabledSticky)$" -count=1 -v
```

Expected: 开关遗漏路径仍发生缓存或 response-id 绑定；OpenAI hydration 用例的 release 调用为两次。

- [x] **Step 2: 统一 Gateway 与 HTTP/Antigravity 的最小守卫**

在 `GatewayService` 使用运行时快照按平台返回开关状态；所有通用会话读取和绑定通过该 helper。`SelectAccountWithLoadAwareness` 在无并发服务时只允许 routing account 集合内的 Sticky 命中。`OpenAIGatewayService.newAcquiredSelectionResult` 仅委托 `newSelectionResult`。

```go
func (s *OpenAIGatewayService) newAcquiredSelectionResult(ctx context.Context, account *Account, release func()) (*AccountSelectionResult, error) {
	return s.newSelectionResult(ctx, account, true, release, nil)
}
```

在 HTTP response-id 绑定处检查 `openAIStickyEnabled()`；在 Antigravity 的模型限流和 smart-retry 清理处复用 `clearStickySession`，并由该函数检查 Anthropic 开关。

- [x] **Step 3: 验证 Task 1 变绿**

Run:

```powershell
go test -tags=unit ./internal/service -run "^(TestGatewayService_GenericStickyHelpersRespectToggle|TestGatewayService_SelectAccountForModelWithPlatform_StickyDisabledBypassesStickyReadAndWrite|TestGatewayService_StickyDisabledBypassLogs|TestGatewayService_SelectAccountWithLoadAwareness|TestGatewayServiceNewSelectionResult_ReleasesAcquiredSlotWhenHydrationFails|TestOpenAINewAcquiredSelectionResult_ReleasesSlotWhenHydrationFails|TestOpenAIGatewayService_BindHTTPResponseAccount(SkipsDisabledSticky)?|TestAntigravityGatewayServiceClearStickySessionSkipsDisabledSticky|TestHandleSmartRetry_(ShortDelay_StickySession_FailedRetry_ClearsSession|LongDelay_StickySession_ClearsSession))$" -count=1
```

Expected: PASS；关闭时没有状态写入，开启时 Antigravity 仍清理绑定。

- [x] **Step 4: 提交 Task 1**

```powershell
git add backend/internal/service/gateway_service.go backend/internal/service/gateway_scheduling.go backend/internal/service/openai_gateway_scheduling.go backend/internal/service/openai_gateway_response_handling.go backend/internal/service/antigravity_gateway_retry.go backend/internal/service/gateway_multiplatform_test.go backend/internal/service/scheduler_snapshot_hydration_test.go backend/internal/service/openai_gateway_service_test.go backend/internal/service/antigravity_smart_retry_test.go
git commit -m "fix: restore platform sticky routing semantics"
```

### Task 2: 统一 OpenAI WebSocket 状态边界

**Files:**
- Modify: `backend/internal/service/openai_ws_forwarder_v2.go`
- Modify: `backend/internal/service/openai_ws_forwarder_ingress.go`
- Test: `backend/internal/service/openai_account_scheduler_test.go`
- Test: `backend/internal/service/openai_ws_forwarder_success_test.go`
- Test: `backend/internal/service/openai_ws_forwarder_ingress_session_test.go`

**Interfaces:**
- Consumes: `OpenAIGatewayService.openAIStickyEnabled()`、`OpenAIWSStateStore`。
- Produces: 受控 WS state-store accessor；关闭时返回 `nil`，开启时返回现有 store。

- [x] **Step 1: 扩展 state-store spy 并为 WS V2 和 ingress 写失败用例**

在 `openAIWSStateStoreSpy` 增加 response connection、turn state 和 session connection 的 Get/Bind 调用计数；各方法继续保留原状态接口行为。现有成功 fixture 中将 `cfg.Gateway.Sticky.OpenAI.Enabled` 设为 `false`，预填 response、连接、turn state 和 session connection；转发后断言各计数为零。

```go
type openAIWSStateStoreSpy struct {
	bindResponseCalls     map[string]int
	bindResponseConnCalls map[string]int
	getResponseConnCalls  map[string]int
	bindTurnStateCalls    map[string]int
	getTurnStateCalls     map[string]int
	bindSessionConnCalls  map[string]int
	getSessionConnCalls   map[string]int
}

cfg.Gateway.Sticky.OpenAI.Enabled = false
result, err := svc.Forward(context.Background(), c, account, body)
require.NoError(t, err)
require.NotNil(t, result)
require.Zero(t, store.bindResponseCalls["resp_new_1"])
require.Zero(t, store.bindResponseConnCalls["resp_new_1"])
require.Zero(t, store.getResponseConnCalls["resp_prev_1"])
```

Run:

```powershell
go test -tags=unit ./internal/service -run "StickyDisabled.*(WSv2|Ingress)" -count=1 -v
```

Expected: FAIL，当前 WS 路径仍读写 state store。

- [x] **Step 2: 让 WS V2 与 ingress 使用受控 accessor**

新增私有 helper，并在两个路径的 state store 初始化处使用它：

```go
func (s *OpenAIGatewayService) openAIStickyStateStore() OpenAIWSStateStore {
	if !s.openAIStickyEnabled() {
		return nil
	}
	return s.getOpenAIWSStateStore()
}
```

保持现有 `stateStore != nil` 分支，使 response-account、response-connection、turn state 和 session-connection 的读取与写入一起 bypass。开关关闭时在入口记录一次结构化 bypass 日志，不在每个 state 操作处重复记录。

- [x] **Step 3: 验证 WS 关闭与默认开启语义**

Run:

```powershell
go test -tags=unit ./internal/service -run "^(TestOpenAIGatewayService_Forward_WSv2_SuccessAndBindSticky|TestOpenAIGatewayService_ProxyResponsesWebSocketFromClient_KeepLeaseAcrossTurns|TestOpenAIGatewayService_.*StickyDisabled.*(WSv2|Ingress))$" -count=1
```

Expected: PASS；默认开启仍保存 response/connection state，关闭时所有四类 state 零读写。

- [x] **Step 4: 提交 Task 2**

```powershell
git add backend/internal/service/openai_ws_forwarder_v2.go backend/internal/service/openai_ws_forwarder_ingress.go backend/internal/service/openai_ws_forwarder_success_test.go backend/internal/service/openai_ws_forwarder_ingress_session_test.go
git commit -m "fix: gate OpenAI websocket sticky state"
```

### Task 3: 让 Gemini compat 遵守最终平台开关

**Files:**
- Modify: `backend/internal/service/gemini_messages_compat_service.go`
- Test: `backend/internal/service/gemini_messages_compat_service_test.go`

**Interfaces:**
- Consumes: `resolvePlatformAndSchedulingMode(ctx, groupID)`、`config.GatewayControlRuntime()`、`GatewayCache`。
- Produces: `stickyEnabledForPlatform(platform string) bool` 私有判断和关闭时的无状态候选选择路径。

- [x] **Step 1: 写入 Gemini、Anthropic 与 Antigravity 的失败测试**

为每个平台创建有既有缓存绑定的可调度账号和关闭配置；调用 `SelectAccountForModelWithExclusions`，断言返回正常候选账号且 cache 的 Get、Set、Delete、Refresh 调用均为零。

```go
cfg.Gateway.Sticky.Gemini.Enabled = false
account, err := svc.SelectAccountForModelWithExclusions(ctx, groupID, "session", "gemini-2.5-flash", nil)
require.NoError(t, err)
require.NotNil(t, account)
require.Empty(t, cache.getCalls)
require.Empty(t, cache.setCalls)
```

Run:

```powershell
go test -tags=unit ./internal/service -run "TestGeminiMessagesCompatService.*StickyDisabled" -count=1 -v
```

Expected: FAIL，当前实现直接调用 `GatewayCache`。

- [x] **Step 2: 在解析平台后一次决定 compat 会话行为**

添加私有 helper，nil config 与未知平台默认 `true`，Gemini 使用 `StickyGeminiEnabled`，Anthropic/Antigravity 使用 `StickyAnthropicEnabled`。只在 enabled 时调用命中、清理、刷新和绑定代码；关闭时直接沿用已加载账号的正常选择结果。

```go
stickyEnabled := s.stickyEnabledForPlatform(platform)
if stickyEnabled {
	if account := s.tryStickySessionHit(ctx, groupID, sessionHash, cacheKey, requestedModel, excludedIDs, platform, useMixedScheduling); account != nil {
		return account, nil
	}
}
// existing account selection
if stickyEnabled && sessionHash != "" {
	_ = s.cache.SetSessionAccountID(ctx, derefGroupID(groupID), cacheKey, selected.ID, geminiStickySessionTTL)
}
```

- [x] **Step 3: 验证 compat 平台隔离与默认开启**

Run:

```powershell
go test -tags=unit ./internal/service -run "TestGeminiMessagesCompatService.*Sticky" -count=1
```

Expected: PASS；每个平台只受自身开关影响，开启时保留原缓存命中和绑定。

- [x] **Step 4: 提交 Task 3**

```powershell
git add backend/internal/service/gemini_messages_compat_service.go backend/internal/service/gemini_messages_compat_service_test.go
git commit -m "fix: honor platform sticky toggles in gemini compat"
```

### Task 4: 完整回归与 change 收敛

**Files:**
- Modify: `openspec/changes/restore-platform-sticky-routing/tasks.md`
- Test: `backend/internal/service/...`

**Interfaces:**
- Consumes: Tasks 1-3 的平台边界实现和 `platform-sticky-boundaries` delta spec。
- Produces: 可验证、可归档的完整 change。

- [x] **Step 1: 运行受影响服务验证**

```powershell
go test -tags=unit ./internal/service -run "(Sticky|SelectAccountWithLoadAwareness|HydrationFails|OpenAIWS|GeminiMessagesCompat|SmartRetry)" -count=1
go build ./...
git diff --check
```

Expected: 聚焦服务测试和构建 PASS，diff 无空白错误。

- [x] **Step 2: 记录全包已知失败边界**

```powershell
go test -tags=unit ./internal/service -count=1
```

Expected: 本 change 的 Sticky/WS/compat 用例全部通过；若仍失败，仅允许为后续 body replay 或测试夹具批次已记录的失败，不能新增本 change 相关失败。

- [x] **Step 3: 更新任务并提交验证记录**

```powershell
git add openspec/changes/restore-platform-sticky-routing/tasks.md
git commit -m "test: verify platform sticky boundaries"
```

### Task 5: 补齐 invalid fallback 的 mixed Antigravity 清理

- [x] **Step 1: 按 resolved group/session 传播 Antigravity retry 清理参数，并覆盖 Anthropic Sticky enabled/disabled 回归**
