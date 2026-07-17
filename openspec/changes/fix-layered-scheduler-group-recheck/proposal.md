## Why

OpenAI 分层调度器在选中账号后的数据库二次校验中遗漏分组 ID，导致已绑定分组的账号被按“未分组账号”规则误判并全部排除，网关返回 `503 Service temporarily unavailable`，而直接账号测试仍可成功。

## What Changes

- 分层调度器调用二次校验时传递请求的分组 ID。
- 增加分组账号在分层调度路径下可被成功选中的回归测试。
- 不改变 API、数据库结构或调度策略。

## Capabilities

### New Capabilities

- `openai-account-scheduling`: 记录 OpenAI 分层调度器必须保留请求分组上下文的既有行为。

### Modified Capabilities

无。

## Impact

影响 `backend/internal/service/openai_account_scheduler_layered.go` 及其测试。无需迁移、配置变更或新增依赖。
