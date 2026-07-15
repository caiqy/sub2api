## Context

`2e3e92457b` 的冲突解决保留了 daily/weekly/monthly 三次条件更新，覆盖了 repository 已有的 `ResetUsageWindows` 原子更新。多个窗口同时重置时，中途失败会留下部分提交。

## Goals / Non-Goals

**Goals:**
- 恢复管理员所选 quota 窗口的单次原子更新。
- 仅在更新成功后失效缓存并返回刷新后的订阅。

**Non-Goals:**
- 不改变自动窗口重置的 CAS 和锚定周期逻辑。
- 不修改 repository 接口或数据库实现。

## Decisions

- `AdminResetQuota` 直接调用现有 `ResetUsageWindows`，删除顺序 reset 与部分成功标志。
- 原子调用失败时不失效缓存，因为数据库状态未变化。

## Risks / Trade-offs

- 管理员并发重置不再逐窗口使用旧 window start 作为 CAS 条件；该方法原有原子 repository 契约即采用按 ID 单次更新。
