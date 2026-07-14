## Why

近期请求体 spool/重放重构遗漏了 OpenAI raw chat fallback 的 body 恢复，并让 Grok raw multipart 文件内容在解析后丢失，导致 failover、moderation 和 images edit 路径无法复用原始请求体。

## What Changes

- OpenAI raw chat fallback 在解析模型和发送请求前从已绑定的 `RequestBodyHandle` 恢复原始 body。
- Grok raw multipart parser 保留上传文件 bytes，并让 moderation 与 images edit JSON 转换支持该内存表示。
- 修正 silent-refusal 测试夹具的 HTTP header canonicalization，保留现有生产行为。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `request-body-retention-control`: 明确 OpenAI raw fallback 与 Grok raw multipart 转换必须重放完整原始 body。

## Impact

- `backend/internal/service/openai_gateway_chat_completions_raw.go`
- `backend/internal/service/openai_gateway_chat_completions_raw_test.go`
- `backend/internal/service/grok_media.go`
- `backend/internal/service/openai_images.go`
- `backend/internal/service/openai_images_responses.go`
- `backend/internal/service/openai_gateway_grok_test.go`
- 不修改公开 API、配置、数据库 schema 或依赖。
