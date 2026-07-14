# Comet Design Handoff

- Change: restore-platform-sticky-routing
- Phase: design
- Mode: compact
- Context hash: 33feae13e7bb66c3eca13d27ef30da3cb7ce846f67abb95322295a4324c41925

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/restore-platform-sticky-routing/proposal.md

- Source: openspec/changes/restore-platform-sticky-routing/proposal.md
- Lines: 1-26
- SHA256: 61cb2703b7693e06b31ca8f0e41f1233266a5fdb5c9ae04371878604f29ef704

```md
## Why

上游合并后的调度拆分遗漏了平台级 Sticky 开关在部分读写与模型路由路径中的守卫，且 OpenAI 账号 hydration 失败路径重复释放调度槽位。两者会导致关闭 Sticky 后仍绑定会话，以及并发计数被错误递减。

## What Changes

- 恢复 OpenAI、Gemini、Anthropic/Antigravity 平台各自的 Sticky 开关守卫，并保留关闭时的结构化 bypass 日志。
- 在没有 `ConcurrencyService` 时，模型路由仍从路由账号集合选择，避免回退为全量账号旧顺序。
- 删除 OpenAI wrapper 在 hydration 失败时对已由通用选择结果处理的重复 release。

## Capabilities

### New Capabilities

- `platform-sticky-boundaries`: 平台 Sticky 开关在 HTTP、WebSocket、Ingress 和 compat 调用链中的统一状态边界。

### Modified Capabilities

- 无。

## Impact

- `backend/internal/service/gateway_service.go`
- `backend/internal/service/gateway_scheduling.go`
- `backend/internal/service/openai_gateway_scheduling.go`
- 现有 gateway Sticky、模型路由和 hydration 失败测试。

```

## openspec/changes/restore-platform-sticky-routing/design.md

- Source: openspec/changes/restore-platform-sticky-routing/design.md
- Lines: 1-40
- SHA256: 20696439fc8d3b8a2594843846c7068b32512b4cf65f9bd5c1aadfd1fef46966

```md
## Context

当前调度管线已拆分 Sticky、负载感知和模型路由路径。历史平台级开关逻辑未完整迁入，导致关闭开关后部分路径仍读取或写入会话缓存；无 `ConcurrencyService` 的模型路由也可能绕过已筛选的路由账号集合。另一个独立问题是 OpenAI wrapper 与通用 `GatewayService.newSelectionResult` 同时拥有 hydration 失败时的 release 责任。

## Goals / Non-Goals

**Goals:**
- 平台 Sticky 开关统一决定 cache read、write、prefetch 和相关调度分支是否可用。
- 未配置并发服务时维持模型路由账号集合的选择语义。
- hydration 失败时每个已取得的槽位仅释放一次。

**Non-Goals:**
- 不调整 Sticky 默认值、配置结构、公开接口或数据库 schema。
- 不重构调度算法或新增缓存策略。

## Decisions

### 1. 在 GatewayService 集中平台开关判断

添加最小平台判断 helper：nil service/config 与未知平台保持默认启用；Gemini、OpenAI、Anthropic/Antigravity 分别读取各自配置。所有已拆分的 Sticky 缓存读取、绑定、prefetch 和负载感知入口复用该判断；关闭时记录既有结构化 bypass 日志。

### 2. 保持模型路由候选集合

当 `ConcurrencyService` 不可用时，从当前已筛选的 routing account 集合选择，而不是退回全量账号列表。该路径不引入新的调度策略。

### 3. 由通用选择结果唯一拥有 release

`GatewayService.newSelectionResult` 已在 hydration 失败时调用 release。OpenAI `newAcquiredSelectionResult` 只委托该方法，不再包装第二次 release。

## Risks / Trade-offs

平台判断遗漏入口会造成关闭 Sticky 后仍有缓存操作，因此由现有跨平台、bypass 日志和模型路由测试覆盖。删除重复 release 只改变错误路径，保留成功路径的既有计数语义。

## Migration Plan

无需迁移。代码发布后新请求立即采用恢复后的调度与释放语义。

## Open Questions

无。

```

## openspec/changes/restore-platform-sticky-routing/tasks.md

- Source: openspec/changes/restore-platform-sticky-routing/tasks.md
- Lines: 1-13
- SHA256: 575619b04a7224b4cd8097e9a35dc9294abe9f2f3305e333c66d8a02e7708b49

```md
## 1. 回归证据与平台 Sticky 修复

- [x] 1.1 运行既有平台 Sticky、模型路由和 hydration 失败聚焦用例，记录预期失败。
- [x] 1.2 在当前 GatewayService 调度路径恢复平台 Sticky 守卫、bypass 日志和无并发服务时的 routing account 选择。

## 2. 槽位释放修复与验证

- [x] 2.1 删除 OpenAI hydration wrapper 的重复 release，保留通用选择结果的单一所有权。
- [ ] 2.2 运行聚焦服务测试和受影响 package 测试，确认 Sticky 默认启用、平台隔离、模型路由与单次 release。

## 3. 提交

- [ ] 3.1 提交该独立 Hotfix 修复。

```

## openspec/changes/restore-platform-sticky-routing/specs/platform-sticky-boundaries/spec.md

- Source: openspec/changes/restore-platform-sticky-routing/specs/platform-sticky-boundaries/spec.md
- Lines: 1-27
- SHA256: 4f165c67451628db9a6f99de1bfbbe0aa78246b319f294b683adb229c38a1eda

```md
## ADDED Requirements

### Requirement: Platform Sticky State Boundaries

系统 MUST 依据请求最终平台的 Sticky 开关决定是否访问会话和响应绑定状态。OpenAI、Gemini 与 Anthropic/Antigravity 的开关彼此独立；缺失运行时配置时保持既有默认开启行为。

#### Scenario: OpenAI Sticky disabled bypasses HTTP and WebSocket state

- **WHEN** OpenAI Sticky 被关闭且请求经过 HTTP、WS V2 或 WS ingress 路径
- **THEN** 系统 MUST 不读取或写入 response-account、response-connection、turn state 和 session-connection
- **AND** 系统 MUST 继续完成不依赖这些状态的当前请求

#### Scenario: Compat selection honors the resolved platform toggle

- **WHEN** Gemini Messages compat 服务为请求解析出 Gemini、Anthropic 或 Antigravity 平台且该平台 Sticky 被关闭
- **THEN** 系统 MUST 不读取、写入、删除或刷新该请求的会话缓存绑定
- **AND** 系统 MUST 从正常候选账号选择路径继续处理请求

#### Scenario: Enabled Sticky preserves existing state behavior

- **WHEN** 请求平台的 Sticky 开关保持开启
- **THEN** 系统 MUST 保留既有会话、响应和连接状态的读写行为

#### Scenario: Anthropic Sticky disabled bypasses Antigravity cleanup

- **WHEN** Anthropic/Antigravity Sticky 被关闭且 Antigravity 重试或模型限流路径需要清理会话绑定
- **THEN** 系统 MUST 不删除会话缓存绑定

```
