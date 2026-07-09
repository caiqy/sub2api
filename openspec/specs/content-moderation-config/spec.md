# content-moderation-config Specification

## Purpose
TBD - created by archiving change fix-content-moderation-deleted-groups. Update Purpose after archive.
## Requirements
### Requirement: 保存时清理已删除审计分组
内容审计配置保存时，系统 MUST 移除不存在的审计分组 ID，并保留仍存在的审计分组 ID。

#### Scenario: 配置包含已删除分组
- **WHEN** 管理员保存内容审计配置，且配置中的指定审计分组包含已删除分组 ID
- **THEN** 系统保存配置成功，并从保存结果中移除已删除分组 ID

#### Scenario: 分组查询发生非不存在错误
- **WHEN** 管理员保存内容审计配置，且分组查询返回非不存在错误
- **THEN** 系统 MUST 返回错误且不保存配置
