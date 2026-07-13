# Brainstorm Summary

- Change: add-openai-first-token-timeouts
- Date: 2026-07-13

## 确认的技术方案

- 仅 `tool_choice.type == "image_generation"` 使用 600 秒生图档；其他 Responses 流式请求使用 30 秒生文档。
- `response.created` 与 `response.in_progress` 为前导事件；首个业务事件停止 watchdog，终态停止 watchdog 但不伪造首 Token。
- HTTP SSE 通过可取消的上游 context 同时覆盖响应头前和响应头后等待，超时返回 HTTP 504 `first_token_timeout`。
- WebSocket 按 turn 计时，超时发送 `response.cancel` 并 drain；确认终态后复用连接，否则废弃连接。
- 超时是请求级不可 failover 错误，不重试、不换号、不封禁或惩罚账号。

## 关键取舍与风险

- 不按工具列表或提示词猜测生图，避免 Codex 常驻图片工具导致文本请求绕过短超时。
- 图片前置事件没有官方时限保证，因此明确生图采用可运行时调整的 600 秒默认值，并记录阶段耗时。
- 本地取消不保证上游停止计费；可观测性不得将 timeout 表述为零消耗。
- WebSocket 只有确认当前 response 终态后才复用，优先避免迟到事件污染后续 turn。

## 测试策略

- 表驱动测试覆盖请求分类、前导/业务/终态事件分类和 0/负数配置边界。
- HTTP 测试覆盖响应头前后超时、业务事件停止计时、并发竞争和账号调度隔离。
- 两套 WebSocket 测试覆盖 cancel/drain 成功复用、清理失败废弃、多 turn 继续和一次性错误。
- 管理端与运行时设置测试覆盖默认值、旧 JSON 兼容、保存及新请求生效。

## Spec Patch

无。当前 delta spec 已覆盖已确认的验收场景和边界条件。
