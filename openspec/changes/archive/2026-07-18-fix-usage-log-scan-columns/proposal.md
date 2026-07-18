## Why

管理后台使用记录列表查询稳定返回 HTTP 500。根因是 `usageLogSelectColumns` 比 `scanUsageLog` 少了 `image_input_tokens` 和 `image_input_cost` 两列，导致数据库返回列数与 `Scan` 目标数不一致。

## What Changes

- 补齐使用记录查询缺失的两个图片输入计费列，并保持列顺序与扫描目标一致。
- 增加查询列与扫描目标契约的最小回归测试，防止后续字段只更新一侧。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

无。本次仅修复已有使用记录查询的实现缺陷，不改变需求或验收场景。

## Impact

- 影响 `backend/internal/repository/usage_log_repo_query.go` 的使用记录查询。
- 影响 repository 单元测试；不改变 API、数据库 schema 或依赖。
