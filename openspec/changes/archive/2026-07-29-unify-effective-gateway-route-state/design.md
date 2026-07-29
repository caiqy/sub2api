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
