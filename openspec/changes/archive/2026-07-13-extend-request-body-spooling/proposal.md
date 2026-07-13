## Why

现有 request body 驻留优化只在 OpenAI/Grok `/responses` 入口启用 10MB spool，其余协议仍会在 handler 入口把完整大 body 读入内存，并在上游等待、重试或 failover 期间形成长生命周期引用。需要将同一套文件化请求体和有界观测契约扩展到所有剩余入口，避免不同协议在小内存部署中继续出现不一致的内存放大。

## What Changes

- 引入共享的入站 request body coordinator，统一处理 identity/压缩 JSON、10MB spool、5MB preview、hash、重复打开和 cleanup。
- 将 spool 覆盖扩展到 Anthropic 分组 `/v1/responses` 兼容转换、Anthropic `/v1/messages`、OpenAI `/v1/chat/completions`、OpenAI Embeddings、OpenAI/Grok Images 与 Videos、Gemini `/v1beta/models/*`。
- 压缩 JSON 按解压后的有效 body 大小判断 spool；multipart 按原始请求体大小处理，并复用标准库临时文件能力承载上传内容。
- 所有入口在成功、失败、取消、retry 和 failover 后清理临时文件；spool I/O 失败统一返回 503，不回退到长期持有完整大 body。
- 保持现有 API、内容审计、模型映射、计费、usage 指纹、上游请求、流式终止和错误透传语义不变。

## Capabilities

### New Capabilities

### Modified Capabilities

- `request-body-retention-control`: 将大请求文件化和可重放要求从 `/responses` 扩展到所有支持大 body 的网关协议及媒体入口，并明确压缩、multipart 和生命周期语义。

## Impact

- 影响 Anthropic/OpenAI/Grok/Gemini handler、协议转换服务、上游 request builder、usage detail 与 ops capture。
- 复用现有 `RequestBodyHandle` 和 Go 标准库，不新增第三方依赖、配置项或数据库 schema。
- 需要跨协议成功、失败、取消、压缩、multipart、retry/failover 和临时文件清理回归测试。
