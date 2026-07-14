## Context

`RequestBodyHandle` 重构后，OpenAI Responses 到 raw chat 的 fallback 会传入 `nil` body，并依赖已绑定 handle 恢复原请求；公共 raw 发送管线抽取时遗漏了该恢复步骤。Grok raw multipart parser 则仍读取文件 bytes，但迁移后的上传结构只保存 `FileHeader`，导致 raw bytes 被丢弃。

## Goals / Non-Goals

**Goals:**
- 恢复 OpenAI raw fallback 从 bound handle 重放原始 Chat body。
- 让 Grok raw multipart 的 moderation 与 images edit 转换保留并使用上传 bytes。
- 保留标准 HTTP multipart `FileHeader` 路径和现有 failover 语义。

**Non-Goals:**
- 不修改 `RequestBodyHandle` 生命周期、handler coordinator、公开 API、配置或 schema。
- 不改变真实 HTTP response header 的生产读取逻辑。

## Decisions

- raw chat 入口在解析 model 前复用现有 `openAIRequestBodyBytes`；相比修改 fallback 调用方，该入口同时覆盖所有传空 body、依赖 bound handle 的调用。
- Grok raw parser把已读取 bytes保存在现有上传值中，data URL helper优先沿用 `FileHeader`，缺失时回退 raw bytes；相比构造伪 `FileHeader`，该方案不需要临时文件或新所有权。
- silent-refusal 仅修正测试 fixture 的 canonical header key，不修改生产代码。

## Risks / Trade-offs

- raw bytes 会在该兼容路径保留到单次请求转换完成；沿用当前 parser 已读取到内存的行为，不增加额外复制层级。
- 两种上传表示需要保持等价；现有 raw multipart 和标准 handler multipart 测试分别覆盖。
