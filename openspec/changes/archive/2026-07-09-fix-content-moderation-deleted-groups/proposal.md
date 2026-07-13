## Why

内容审计设置保存时，如果配置里残留了已删除的审计分组 ID，后端会返回“审计分组不存在”，导致管理员无法保存其他审计设置。

## What Changes

- 保存内容审计配置时，自动移除已删除的审计分组 ID。
- 保留真实仓储错误的失败行为，避免掩盖数据库或查询异常。
- 增加回归测试覆盖已删除分组残留场景。

## Capabilities

### New Capabilities

- `content-moderation-config`: 内容审计配置保存时处理审计范围分组 ID 的行为。

### Modified Capabilities

无。

## Impact

- 影响后端内容审计配置保存逻辑：`backend/internal/service/content_moderation.go`。
- 影响后端内容审计服务测试：`backend/internal/service/content_moderation_test.go`。
