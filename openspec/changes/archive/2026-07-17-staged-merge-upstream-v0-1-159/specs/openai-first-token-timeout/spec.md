## REMOVED Requirements

### Requirement: 流式请求按明确生图意图选择首 Token 超时
**Reason**: 用户在了解上游仅部分覆盖及行为差异后，明确选择完全采用上游 `v0.1.156` 超时实现，不再维护本地文本/生图分档。
**Migration**: 删除本地分类与分档逻辑；部署方改用上游 `openai_first_output_timeout_seconds` 和 `openai_high_effort_first_output_timeout_seconds`。

### Requirement: 首 Token 等待采用业务事件边界
**Reason**: 本地业务事件分类与 watchdog 生命周期由上游 native HTTP 首输出判定替代。
**Migration**: 删除本地首 Token 事件分类和与通用流间隔超时的专用协调逻辑，采用上游 `v0.1.156` 响应暂存及首语义输出边界。

### Requirement: 首 Token 超时可运行时配置
**Reason**: 本地运行时配置、API 和管理端 UI 不属于上游实现，用户已批准完整移除。
**Migration**: 删除 `openai_text_first_token_timeout` 与 `openai_image_first_token_timeout` 的配置、持久化、API、UI 和兼容逻辑；按部署配置使用上游字段，上游默认值为关闭。

### Requirement: HTTP SSE 超时直接返回不可重试错误
**Reason**: 用户选择采用上游首输出超时的错误与 failover 语义。
**Migration**: 删除本地 `first_token_timeout` 不可重试错误，采用上游 `first_output_timeout`、failover 和账号流超时处理。

### Requirement: WebSocket 超时取消并清理当前 response
**Reason**: 上游 `v0.1.156` 未提供等价的上游 WebSocket 首输出 watchdog，用户仍明确批准移除本地实现。
**Migration**: 删除 pooled WebSocket 和 V2 passthrough relay 的本地首输出计时、cancel/drain 与连接复用保护；仅保留上游客户端首消息超时及既有 WebSocket 读写超时。

### Requirement: 首 Token 超时不影响账号调度状态
**Reason**: 用户选择采用上游 HTTP 首输出超时的账号处理与 failover 行为。
**Migration**: 删除本地请求级不可 failover 特判，采用上游 `HandleStreamTimeout` 和 `UpstreamFailoverError` 语义。

### Requirement: 超时诊断信息可观测
**Reason**: 本地专用失败 usage、Ops 字段和结构化日志随本地首 Token 能力移除。
**Migration**: 采用上游 `first_output_timeout` 的日志与 Ops 事件；不保留 `gateway.openai_first_token_timeout` 兼容事件。
