## Why

当前分支的后端全量测试会在 Windows 文件句柄并发清理和 OpenAI first-output keepalive 时序上随机失败，同时 `local-test-gates` 主规范因 Purpose 过短无法通过 strict 校验，阻塞分支合并。

## What Changes

- 让 request body spool cleanup 在仍有活动 reader 时延迟物理删除，同时立即阻止新的读取。
- 修正 keepalive ticker 与 downstream 空闲计时的初始化顺序，确保首个 keepalive 不会被误跳过。
- 为 Server-Timing rows 生命周期测试保留足够的 driver 与 app 时间比例，避免系统负载放大短 sleep 后产生误报。
- 补足 `local-test-gates` Purpose，并保留现有严格回归断言。
- 不改变 API、数据库结构、网关协议或调度策略。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `local-test-gates`: 明确 cleanup 与活动 spool reader 并发时的跨平台生命周期语义。

## Impact

影响 `backend/internal/service/request_body_handle.go`、`backend/internal/service/openai_gateway_response_handling.go`、相关 service/repository 测试及 `openspec/specs/local-test-gates/spec.md`。无需迁移、配置变更或新增依赖。
