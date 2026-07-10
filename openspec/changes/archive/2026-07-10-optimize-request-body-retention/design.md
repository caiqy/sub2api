## Context

当前 `/responses` 大请求的内存问题已经不只来自内容审计和 usage worker。完整 request body 会在 middleware、handler、upstream capture、ops context 和 async usage snapshot 中形成长生命周期副本，尤其在上游响应耗时 10 秒以上时，对小内存机器压力很大。

## Goals / Non-Goals

**Goals:**
- 所有 `/responses` 请求都减少 request/upstream body 的观测副本和长生命周期引用。
- `>10MB` 请求在等待上游期间，把完整 body 从 RAM 下沉到临时文件。
- 保持 `requestPayloadHash`、内容审计、上游转发和 failover 语义不变。

**Non-Goals:**
- 不做流式 JSON 解析。
- 不处理 response body / upstream response body 的完整捕获。
- 不扩展到 Anthropic/Gemini 等其他协议入口。

## Decisions

- 使用一个小型 `RequestBodyHandle` 统一承载 body 来源、preview、hash 和重复打开能力。
- request capture 和 upstream capture 只保留 preview，不再各自完整 `ReadAll` body。
- `spool threshold=10MB`，`preview limit=5MB`；两者分离。
- 大 body 文件化失败时返回 `503`，不静默回退到高内存路径。

## Risks / Trade-offs

- usage detail 不再保留完整超大 body，管理员排障信息减少，但换来可控内存与存储成本。
- 文件化路径引入磁盘依赖和清理逻辑，但比依赖 swap 更可控。
- 非透传路径的 `map[string]any` 仍会在同步处理窗口内存在；首版先解决长生命周期完整 body。
