# Brainstorm Summary

- Change: unify-effective-gateway-route-state
- Date: 2026-07-28

## 确认的技术方案

- 将 handler 层现有 `effectiveGatewayRoute` 提升为 service 层 request-scoped `EffectiveGatewayRoute`，由共享 resolver 生成。
- 扩展现有 route-local `compositeTargetPlatformMiddleware`，在真实 endpoint protocol switch 前完成客户端识别、最终 group/订阅/composite route 解析与原子 Apply；不新增全局 middleware。
- `GatewayHandler`、`OpenAIGatewayHandler`、Messages/Responses/count_tokens 与 prompt-too-long fallback 消费同一 snapshot；fallback 只在最终候选完整通过校验后替换上下文和 body。
- billing source 显式区分 `simple-skip`、`balance`、`subscription`；订阅模式缺少 subscription 时 fail closed。
- 模型阶段固定为 client、composite route、可选 channel、可选 account；无 channel mapping 时保留 concrete routing model。
- HTTP bridge later-turn 在 route rewrite 后复用现有 account model mapping；保留现有 provider-affinity callback，不扩展到 passthrough 首帧。

## 关键取舍与风险

- 采用一个跨 routes/handler/service 的值对象，换取单一状态权威；不增加 interface、factory、持久化模型或新依赖。
- 只在最终 group 改变且目标为订阅组时读取现有 subscription repository；不预建缓存，出现可测瓶颈后再处理。
- route-local middleware 需要读取已受 body limit 保护的请求体，但仅覆盖 composite 或 ClaudeCodeOnly 路径，普通路径保持既有热路径。
- 共享 API key 使用 request-owned clone，并清除原 group 的 `UserGroupRPMOverride` snapshot，避免最终组复用旧策略。
- follow-up 不重复远程 Docker integration；完整远程门禁由归档后恢复的原 Task 21 执行。

## 测试策略

- service TDD：最终 group、subscription、billing source、API key/User clone、composite route、SimpleMode 和 identity。
- route/handler TDD：protocol switch 前的最终平台、Gin API key/subscription、request group、Ops 可见状态和失败原子性。
- fallback TDD：prompt-too-long 经过中间 ClaudeCodeOnly 组到最终订阅组，不走余额且失败无第二次调度。
- 审计 TDD：public alias、concrete route、account upstream 三阶段。
- WebSocket TDD：HTTP bridge 第二轮 route rewrite 后 account mapping；保留普通 WS 和跨 provider 拒绝回归。
- 最终门禁：聚焦测试、两次稳定 generate、`make test`、`make build`、`git diff --check`、strict OpenSpec validation 和 fresh review。

## Spec Patch

已在最终确认前回写 route-local protocol switch 与 HTTP bridge later-turn 的边界；当前无需进一步 Spec Patch。
