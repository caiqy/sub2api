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
