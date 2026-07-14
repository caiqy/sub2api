---
comet_change: restore-platform-sticky-routing
role: technical-design
canonical_spec: openspec
---

# 平台 Sticky 跨管线状态边界设计

## 背景

平台 Sticky 配置已分别提供 OpenAI、Gemini 与 Anthropic 开关，但调度、HTTP、WebSocket、Ingress 和 compat 服务各自直接访问缓存或状态存储。关闭开关后仍可能出现跨请求账号、连接或 turn state 绑定。

现有未提交改动已经恢复 Gateway 调度平台守卫、无并发服务的模型路由优先级、OpenAI hydration 单次 release、HTTP response-id 绑定和 Antigravity 清理守卫。本设计将这些局部修复扩展为完整跨管线边界。

## 目标与非目标

目标：

- 平台 Sticky 开关关闭时，相关状态路径不读取、不写入、不刷新或删除绑定。
- OpenAI WebSocket 的 response-account、response-connection、turn state 和 session-connection 同时受 OpenAI 开关控制。
- Gemini Messages compat 服务按最终平台遵守对应开关。
- 缺失配置时维持默认开启；开启时不改变现有状态、调度和重试行为。

非目标：

- 不修改配置结构、公开 API、数据库 schema、缓存实现或调度算法。
- 不改变 OpenAI WS transport 的非 Sticky 行为；关闭 Sticky 后不再提供跨请求连接复用是已确认的结果。

## 设计

### Gateway 调度

保留 `GatewayService` 的平台 helper，所有通用会话缓存读写和调度绑定均通过它执行。无 `ConcurrencyService` 时，先以模型路由集合约束 Sticky 账号；不在集合内时回退现有 legacy 选择，使模型路由优先于旧会话绑定。

### OpenAI HTTP 与 WebSocket

HTTP response-id 绑定在入口检查 `openAIStickyEnabled()`，关闭时记录一次结构化 bypass 日志并跳过写入。

WS V2 和 ingress 在每次请求的状态初始化处取得受控 state-store。关闭时该 accessor 返回 `nil`，后续既有的 nil guard 会统一跳过 response-account、response-connection、turn state 和 session-connection 的读写，不需为每个调用点引入独立分支。状态存储错误在开启时仍按现有 warning/降级语义处理；关闭时不会触发存储调用。

### Gemini Messages Compat

在 `resolvePlatformAndSchedulingMode` 后计算最终平台的 Sticky 状态。关闭时跳过 `tryStickySessionHit` 以及后续绑定；因此不会发生读取、删除、TTL 刷新或写入。账号仍使用现有候选选择路径，避免将开关关闭解释为无可用账号。

### Antigravity 清理

模型限流和 smart-retry 均通过 `clearStickySession` 删除会话。该函数读取 Anthropic 运行时开关；关闭时返回，开启或缺失配置时执行既有删除与错误日志。

## 风险与取舍

- OpenAI WS 关闭 Sticky 时不再复用跨请求连接与 turn state，符合确认的完整 bypass 语义。
- 状态存储还可能被未来功能用于非 Sticky 数据；本次只在已识别的请求路径使用受控 accessor，避免全局修改 store 的语义。
- compat 服务的最终平台可能为 Antigravity，统一使用 Anthropic 开关以匹配 Gateway 平台映射。

## 测试与验证

- HTTP response-id、WS V2、WS ingress 分别验证关闭时 state store 零读写，开启时保留原行为。
- Compat 在 Gemini、Anthropic、Antigravity 三个平台关闭时验证会话缓存零读写，并验证正常候选选择仍可用。
- 保留并运行现有平台隔离、模型路由、默认开启、Antigravity 清理和 hydration 单次 release 回归测试。
- 运行受影响服务 package、`go build ./...`；在后续 body replay 和测试夹具批次完成后运行完整 unit/integration 门禁。
