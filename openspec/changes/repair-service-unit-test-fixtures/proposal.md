## Why

请求体、定价与运行时设置重构后，4 个 service unit 测试仍使用旧 fixture 契约，导致完整 unit 门禁失败并掩盖真实回归信号。

## What Changes

- credits overages direct-call fixture 使用当前 `payloadHandle` 契约。
- pricing fixture 对齐 channel/group 平台并删除重复用例。
- settings repository stub 实现生产一致的 `GetValue` 缺失语义并恢复测试全局状态。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- 无。

## Impact

- 仅修改 `backend/internal/service` 下 3 个测试文件。
- 不修改生产代码、公开 API、配置、数据库 schema、依赖或产品 spec。
