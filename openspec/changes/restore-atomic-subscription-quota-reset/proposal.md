## Why

上游合并冲突将管理员 quota reset 从单次原子更新改成多个顺序更新，后续窗口失败时可能留下部分已重置的数据。

## What Changes

- `AdminResetQuota` 通过 repository 的 `ResetUsageWindows` 一次提交所选窗口。
- 原子更新成功后再失效订阅缓存；失败时不产生部分重置语义。
- 更新仍断言部分成功行为的相邻回归测试。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- 无。

## Impact

- 修改 `backend/internal/service/subscription_service.go` 与相邻 unit test。
- 不修改公开 API、repository 接口、数据库 schema、依赖或产品 spec。
