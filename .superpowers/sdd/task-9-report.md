# Task 9 Report

## 缓存隔离

- `TestAdminUsageStatsRequestTypePriority` 在开始和 cleanup 时重置 30 秒 `usageStatsCache`，隔离包级 snapshot cache。

## RED/GREEN

- RED: 未重置包级 `usageStatsCache` 时，其他测试留下的缓存可能使该测试跳过仓库查询。
- GREEN: 测试前后重置缓存后，请求稳定到达测试仓库并断言 `request_type` 优先于 `stream`。

## 验证

- `go test -tags=unit ./internal/handler/admin -run '^TestAdminUsageStatsRequestTypePriority$' -count=3`
