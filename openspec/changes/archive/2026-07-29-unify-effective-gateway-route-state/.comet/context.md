# Comet Design Handoff

- Change: unify-effective-gateway-route-state
- Phase: design
- Mode: compact
- Context hash: 09c97e47905349c5775af6be349cd4ac720c7621a6a6ea511beb65c7024ba5bb

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/unify-effective-gateway-route-state/proposal.md

- Source: openspec/changes/unify-effective-gateway-route-state/proposal.md
- Lines: 1-28
- SHA256: a9b18d2f6c33e0efcbde08cf51a8b0d5249d5026d350690fa8056ef8fe8098b9

```md
## Why

上游 `v0.1.164` 引入 composite routing 后，最终 group、subscription、协议分发、模型映射和 WebSocket 会话账号仍由不同路径分别推进，已造成鉴权/计费状态漂移、模型回退和 provider affinity 缺口。原 staged merge 的审查预算已耗尽，需要用独立 follow-up change 收敛共同根因并提供新的验证证据。

## What Changes

- 把现有 handler `effectiveGatewayRoute` 提升为 service 层 request snapshot，让最终 group、request-owned API key、显式计费来源、subscription、精确 endpoint、平台和模型阶段由同一解析结果承载。
- 在 endpoint 协议分发、并发、调度和计费前原子应用有效路由状态；运行时 fallback 使用同一 resolver，失败时不得留下半更新状态。
- 固定 `client -> composite route -> channel mapping -> account mapping` 顺序，无映射时保持上一阶段模型。
- 让 HTTP bridge 的后续 `response.create` 在 route rewrite 后补齐 account mapping，并以现有跨 provider 拒绝路径作回归保护。
- 增加对应聚焦回归测试，并以本地 `make test`、`make build`、strict OpenSpec validation 和 fresh review 作为归档门禁。

## Capabilities

### New Capabilities

- `effective-gateway-routing`: 定义 composite/fallback 请求的最终鉴权、计费、协议分发、模型阶段与 WebSocket provider affinity 必须共享同一有效路由状态。

### Modified Capabilities

无。

## Impact

- 后端 handler：`GatewayHandler`、`OpenAIGatewayHandler`、Messages/count_tokens、prompt-too-long fallback 与 Ops context。
- 后端 service：现有 effective group/composite route/channel mapping 解析、billing eligibility 以及 OpenAI WebSocket passthrough/HTTP bridge 的请求改写。
- 测试：扩展现有 handler、billing 和 WebSocket 聚焦测试；不新增依赖。
- 兼容边界：不修改公开 API、数据库、migration、配置、前端或无关协议；不 push、tag、release 或 deploy。

```

## openspec/changes/unify-effective-gateway-route-state/design.md

- Source: openspec/changes/unify-effective-gateway-route-state/design.md
- Lines: 1-67
- SHA256: b24bdfd76bdb6442c40a7361c2273a15a94c487011b7764033e630a480b61e34

```md
## Context

`staged-merge-upstream-v0-1-165` 在合入上游 `v0.1.164` 后仍有四个同源缺口：最终 group 未统一 middleware context、协议分发和 Ops；prompt-too-long 二级 fallback 可能用空 subscription 退化为余额模式；HTTP bridge 后续帧缺少 account mapping/provider affinity；无 channel mapping 时 public alias 可能覆盖 concrete route model。

当前 `GatewayHandler` 已有 `effectiveGatewayRoute`，`OpenAIGatewayHandler` 另有 `resolveEffectiveOpenAIGatewayRoute`，两者只推进部分字段。现有 `CompositeRouteDecision`、`ChannelMappingResult`、request body handle、API key clone 和 billing service 足以实现修复，不需要新增依赖或持久化结构。

## Goals / Non-Goals

**Goals:**

- 让最终 group、request-owned API key、subscription、显式计费来源、endpoint、平台和模型阶段由同一个 request-scoped 结果承载。
- 让初始解析与 runtime fallback 先完整校验，再原子应用到 Gin/request/Ops/body 状态。
- 固定 `client -> composite route -> channel -> account` 的模型阶段与 WebSocket provider affinity。
- 以聚焦 TDD、本地全门禁和 fresh review 关闭原 `3 Important + 1 Minor`。

**Non-Goals:**

- 不修改公开 API、数据库、migration、配置或前端。
- 不重构无关协议或重复修复 staged merge 已关闭的 finding。
- 不新增通用路由框架、interface、factory 或缓存。
- 不运行远程 Docker integration，不执行 push、tag、release 或 deploy。

## Decisions

### 提升现有有效路由类型为跨层 snapshot

将 handler package 已有的 `effectiveGatewayRoute` 提升为 service package 的 `EffectiveGatewayRoute`，并让 route-local middleware、`GatewayHandler` 与 `OpenAIGatewayHandler` 共用 resolver。结果至少承载 effective API key/group/group ID、subscription、billing source、endpoint、client/routing/upstream model、platform、`CompositeRouteDecision` 和 `ChannelMappingResult`。

API key 使用 request-owned clone，不原地修改认证缓存中的共享对象。备选方案是新增平行 `EffectiveGatewayRoute` 或继续逐路径补丁；前者制造重复类型，后者保留状态漂移根因，均不采用。

### 分离纯解析与原子应用

实际 endpoint protocol switch 位于 `server/routes/gateway.go` 且早于 handler。扩展现有 `compositeTargetPlatformMiddleware`，使 resolver 在受限 body capture 和客户端识别后、protocol switch/并发/计费前运行。它先解析最终组、精确 endpoint、composite route 和最终组 billing snapshot，再返回完整结果；handler 在该 snapshot 上补充 channel/account mapping。失败时不修改 Gin context、request context、Ops 或 body handle。

单一 `Apply` helper 在成功后同步 middleware 维护的 API key/subscription 键、request context group/route decision/effective snapshot、Ops 可见的 API key 和 body。runtime fallback 调用相同 resolver/Apply，不复制状态更新步骤。不采用全局新 middleware，因为它会扩大到不需要 body/client 信息的协议。

### 显式表达计费来源

有效路由使用 `simple-skip`、`balance`、`subscription` 三态 billing source。`subscription` 状态要求 subscription 非空；最终订阅组无法取得有效订阅时直接失败，不能用空指针切换到余额检查。SimpleMode 保留既有跳过语义。

### 固定模型阶段

`clientModel` 保留客户端原始别名；composite route 后得到 concrete model；channel mapping 仅在 `Mapped=true` 时覆盖，形成 `routingModel`；账号绑定后才用 account mapping 形成 `upstreamModel`。无映射是 identity，usage 从有效路由读取阶段值，不从改写后的 body 反推。

### HTTP bridge 后续帧补齐账号绑定

普通 WebSocket 的后续帧已按 effective route、channel mapping、provider affinity、account mapping 的顺序处理，HTTP bridge 只缺 route rewrite 后的 account mapping。本 change 在 bridge later-turn 分支复用现有 `applyOpenAIWSAccountModelMapping`；保留 handler callback 中已存在的跨 provider 拒绝，不扩大到独立的 passthrough 首帧生命周期。

### 跨 change 关闭证据

follow-up 的 fresh reviewer 逐项核对原四项 finding，且不得出现新的 Critical/Important。通过后归档本 change，把 review 证据写回原 progress/canonical ledger并勾选 `6.1`，不重开原 change 已耗尽的 review 预算，再恢复原 Task 21。

## Risks / Trade-offs

- [共享 resolver 触及多个 handler 入口] → 只收敛现有两套有效路由逻辑，并为 composite、fallback 和非 composite identity 各留聚焦测试。
- [同步 Gin/request/Ops/body 时遗漏字段] → 只允许单一 `Apply` helper 写入这些状态，resolver 保持无副作用。
- [subscription snapshot 读取增加一次存储访问] → 仅在最终 group 与认证 group 不同且需要订阅时复用现有 repository/cache 路径，不新增缓存。
- [后续 WebSocket 帧拒绝 provider 切换可能暴露既有客户端依赖] → 这是保持已绑定账号连接正确性的必要失败，沿用现有 WebSocket 错误映射并在写入上游前完成。
- [统一模型阶段改变错误 usage 字段] → 通过 client/routing/upstream 三阶段断言固定预期，保留无映射 identity 行为。

## Migration Plan

不需要数据或配置迁移。实施在当前 staged merge 分支完成；聚焦测试、`make test`、`make build`、strict OpenSpec validation 和 fresh review 均通过后归档。若验证失败，保持原 change 阻断并修复本 change；若需要回退，revert 本 change 的聚焦提交，不改写已有 merge 历史。

## Open Questions

无。方案、范围、验证边界和自动恢复流程均已确认。

```

## openspec/changes/unify-effective-gateway-route-state/tasks.md

- Source: openspec/changes/unify-effective-gateway-route-state/tasks.md
- Lines: 1-30
- SHA256: 84aab042340942cedf36d4ad215afc505be7cd0577e0caedf84724db8b074fa2

```md
## 1. 有效路由核心状态

- [ ] 1.1 为最终 group、request-owned API key、显式 billing source、最终组 subscription 与 composite route model 编写 service 层 RED 测试
- [ ] 1.2 实现 `EffectiveGatewayRoute`、共享 resolver、context helpers 和模型阶段 identity 规则
- [ ] 1.3 为订阅分组缺少 subscription 时禁止退化为余额模式编写 RED 测试并收紧 `CheckBillingEligibility`
- [ ] 1.4 将共享 resolver 接入 Wire，重新生成并确认生成结果稳定

## 2. 协议分发与 Handler 消费

- [ ] 2.1 为 ClaudeCodeOnly fallback 后的最终平台分发、Gin API key/subscription、request group 与失败原子性编写 route-local RED 测试
- [ ] 2.2 扩展现有 composite route middleware，在 endpoint protocol switch 前 resolve/validate/apply 有效路由
- [ ] 2.3 让 `GatewayHandler`、`OpenAIGatewayHandler` 和 count_tokens 消费共享 snapshot，并移除重复的局部 effective group 推导
- [ ] 2.4 增加非 composite identity、Messages/Responses/count_tokens 最终组一致性回归

## 3. Runtime fallback

- [ ] 3.1 为 prompt-too-long 经过中间 ClaudeCodeOnly 组到最终订阅组编写 RED 测试，断言最终 subscription、无余额退化和无失败侧效应
- [ ] 3.2 使用共享 resolver/Apply 替换 prompt-too-long 的中间组校验、局部 clone 和 `subscription=nil` 分支

## 4. 模型审计与 WebSocket bridge

- [ ] 4.1 为无 channel mapping 的 public alias 到 concrete route model 编写 RED 测试，并修复 client/routing/upstream 三阶段审计
- [ ] 4.2 为 HTTP bridge later-turn route rewrite 后的 account mapping 编写 RED 测试，并复用现有账号映射 helper 完成最小修复
- [ ] 4.3 重跑普通 WebSocket later-turn account mapping、跨 provider 拒绝和 composite pricing/模型长度保护测试

## 5. 验证与关闭证据

- [ ] 5.1 运行全部聚焦测试、`git diff --check` 与 strict OpenSpec validation
- [ ] 5.2 连续运行两次 `make -C backend generate` 并确认第二次无 diff，再运行本地 `make test` 与 `make build`
- [ ] 5.3 由 fresh reviewer 逐项核对原 `3 Important + 1 Minor`，确认无新的 Critical/Important，并记录供归档和恢复原 Task 21 使用的证据

```

## openspec/changes/unify-effective-gateway-route-state/specs/effective-gateway-routing/spec.md

- Source: openspec/changes/unify-effective-gateway-route-state/specs/effective-gateway-routing/spec.md
- Lines: 1-94
- SHA256: e3ede9fe71ea9335d0a4df6b6da3e2f0185dbbcd0c4698d9a30b8e64df77e20b

[TRUNCATED]

```md
## ADDED Requirements

### Requirement: 最终有效路由是请求策略的单一权威状态

系统 MUST 在 composite routing 或 client-specific fallback 后，以最终有效路由统一决定 group、API key、subscription、计费来源、目标平台、endpoint 协议分发、调度输入和 Ops 上下文。系统 MUST 使用 request-owned API key clone，且不得原地修改共享认证对象。

#### Scenario: Client fallback 切换到最终分组

- **WHEN** API Key 原分组因客户端类型规则 fallback 到另一个可用分组
- **THEN** 系统 MUST 在 endpoint 协议分发、并发、调度和计费前把 Gin API key、request group、subscription、目标平台和 Ops 同步为最终分组
- **AND** 后续阶段 MUST 不再从原分组重新推导策略

#### Scenario: 最终分组不允许当前用户访问

- **WHEN** fallback 或 composite routing 解析出的最终分组不允许当前用户访问
- **THEN** 系统 MUST 在协议分发和账号调度前拒绝请求
- **AND** 系统 MUST 不应用部分有效路由状态

#### Scenario: 非 composite 请求保持 identity

- **WHEN** 请求无需 client fallback 且所属分组不是 composite
- **THEN** 系统 MUST 保持既有 group、平台、endpoint 和模型行为

### Requirement: 有效路由显式决定计费来源

系统 MUST 明确区分 SimpleMode 跳过、余额和订阅三种计费来源。订阅来源 MUST 包含最终分组的有效 subscription；系统不得用空 subscription 把订阅请求隐式转换为余额请求。

#### Scenario: 最终分组使用订阅计费

- **WHEN** 有效路由的最终分组是订阅分组
- **THEN** 系统 MUST 为该最终分组加载并校验有效 subscription
- **AND** billing MUST 使用订阅限额且不执行余额或 user-platform quota 检查

#### Scenario: 最终订阅不可用

- **WHEN** 有效路由的最终订阅分组不存在有效 subscription 或订阅限额不可用
- **THEN** 系统 MUST 按现有订阅错误契约拒绝请求
- **AND** 系统 MUST 不调度账号、不计费且不回退到余额模式

#### Scenario: SimpleMode 保持跳过计费

- **WHEN** 系统运行于 SimpleMode
- **THEN** 有效路由 MUST 保持现有跳过计费语义，且不得要求加载 subscription

### Requirement: Runtime fallback 原子替换有效路由

系统 MUST 让 prompt-too-long 等 runtime fallback 调用与初始请求相同的有效路由 resolver，并在候选结果完整通过授权、subscription、余额、模型和平台校验后原子应用。

#### Scenario: Prompt-too-long 切换到有效候选组

- **WHEN** 上游返回 prompt-too-long 且配置的二级 fallback 最终解析到可用分组
- **THEN** 系统 MUST 使用最终候选组重新建立 API key、subscription、平台、模型和调度状态后再重试

#### Scenario: Runtime fallback 校验失败

- **WHEN** runtime fallback 的最终候选组未通过任一必要校验
- **THEN** 系统 MUST 终止请求并保留应用前状态
- **AND** 系统 MUST 不使用中间组继续调度、写入上游或计费

### Requirement: 模型映射阶段保持确定顺序

系统 MUST 按 client model、composite route、可选 channel mapping、可选 account mapping 的顺序解析模型，并 MUST 分别保留 client、routing 和 upstream 模型供请求改写与 usage 审计使用。

#### Scenario: 无 channel mapping 时保留 concrete route model

- **WHEN** public client alias 经过 composite route 得到 concrete model，且 channel mapping 未命中
- **THEN** `routingModel` MUST 保持该 concrete model
- **AND** 系统 MUST 不使用 public client alias 覆盖它

#### Scenario: Channel 与 account mapping 同时命中

- **WHEN** channel mapping 和已选账号的 account mapping 都命中
- **THEN** 系统 MUST 先生成 channel-mapped `routingModel`，再以它执行 account mapping 得到 `upstreamModel`
- **AND** usage MUST 记录正确的 client、routing 和 upstream 阶段

#### Scenario: 所有映射均未命中

- **WHEN** composite route、channel mapping 和 account mapping 均未改变模型
- **THEN** 三个模型阶段 MUST 保持 identity，且请求体不得发生无意义改写


```

Full source: openspec/changes/unify-effective-gateway-route-state/specs/effective-gateway-routing/spec.md
