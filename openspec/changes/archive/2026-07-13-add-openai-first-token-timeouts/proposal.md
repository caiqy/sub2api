## Why

OpenAI/Codex Responses API 偶发在返回首个业务事件前停滞数分钟，现有响应头超时无法覆盖“已返回响应头但 SSE/WebSocket 仍无有效输出”的场景。图片生成又可能正常等待 50–300 秒，因此需要按明确生图意图区分超时档位，避免用统一短超时误杀正常生图。

## What Changes

- 为 OpenAI Responses `stream=true` 请求新增首 Token 超时保护，同时覆盖 HTTP SSE 和 Responses WebSocket。
- 新增 Gateway 运行时配置：生文请求默认 30 秒、明确生图请求默认 600 秒；值为 `0` 时关闭对应超时。
- 仅当 `tool_choice.type == "image_generation"` 时使用生图档；仅在工具列表中携带 `image_generation` 仍按生文档处理。
- 将 `response.created` 和 `response.in_progress` 视为前导事件；首个有效业务事件（包括图片 `response.output_item.added`）结束首 Token 等待。
- 超时后直接失败，不重试、不换号、不临时封禁或惩罚账号；SSE 返回 HTTP 504，WebSocket 发送 `response.cancel` 并清理当前 response。
- 在管理端“网关服务”设置页暴露两个超时字段，并记录超时阶段、传输类型和上游请求标识等诊断信息。
- 不覆盖非流式请求，不增加图片生成总时长限制，不通过提示词关键词猜测生图意图。

## Capabilities

### New Capabilities

- `openai-first-token-timeout`: 定义 OpenAI Responses 流式请求的生文/生图首 Token 分类、超时、协议错误、WebSocket 取消清理、运行时配置和可观测性要求。

### Modified Capabilities

无。

## Impact

- 后端配置与运行时设置 API：`backend/internal/config`、`backend/internal/service/setting_gateway_runtime.go`、管理端 setting handler/DTO。
- OpenAI Responses 转发：passthrough SSE、转换 SSE、池化 WebSocket ingress、WebSocket V2 passthrough relay。
- 管理端前端：Gateway runtime API 类型、SettingsView 表单、中文和英文文案。
- 使用记录与 Ops：复用现有失败 usage 和上游错误记录，不新增数据库 schema。
- 依赖：仅使用 Go 标准库和现有 `gjson`/WebSocket 设施，不新增第三方依赖。
