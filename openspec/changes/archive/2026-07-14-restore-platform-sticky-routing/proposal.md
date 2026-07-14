## Why

上游合并后的调度拆分遗漏了平台级 Sticky 开关在部分读写与模型路由路径中的守卫，且 OpenAI 账号 hydration 失败路径重复释放调度槽位。两者会导致关闭 Sticky 后仍绑定会话，以及并发计数被错误递减。

## What Changes

- 恢复 OpenAI、Gemini、Anthropic/Antigravity 平台各自的 Sticky 开关守卫，并保留关闭时的结构化 bypass 日志。
- 在没有 `ConcurrencyService` 时，模型路由仍从路由账号集合选择，避免回退为全量账号旧顺序。
- 删除 OpenAI wrapper 在 hydration 失败时对已由通用选择结果处理的重复 release。

## Capabilities

### New Capabilities

- `platform-sticky-boundaries`: 平台 Sticky 开关在 HTTP、WebSocket、Ingress 和 compat 调用链中的统一状态边界。

### Modified Capabilities

- 无。

## Impact

- `backend/internal/service/gateway_service.go`
- `backend/internal/service/gateway_scheduling.go`
- `backend/internal/service/openai_gateway_scheduling.go`
- 现有 gateway Sticky、模型路由和 hydration 失败测试。
