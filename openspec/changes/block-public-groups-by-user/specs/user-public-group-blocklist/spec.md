## ADDED Requirements

### Requirement: 管理员可以按用户禁用公开分组访问
The system SHALL allow an administrator to disable selected active public standard groups for an individual user without changing the group's public status.

#### Scenario: 为单个用户禁用公开分组
- **WHEN** 管理员为某个用户禁用一个公开标准分组并保存用户分组配置
- **THEN** 系统记录该用户与分组的禁用关系，并且该分组对其他用户仍保持公开可见

#### Scenario: 用户侧不展示被禁用公开分组
- **WHEN** 用户查询可绑定分组或可用渠道
- **THEN** 系统不返回该用户已被禁用的公开标准分组

#### Scenario: 专属分组保持 allow-list 行为
- **WHEN** 管理员修改用户的专属分组选项
- **THEN** 系统 SHALL 继续使用显式 allowed group membership 判断专属分组权限

### Requirement: 被禁用的公开分组不能被绑定
The system SHALL reject attempts by a user to bind an API Key to a public standard group blocked for that user.

#### Scenario: 用户为被禁用公开分组创建 API Key
- **WHEN** 用户创建 API Key 时选择了对自己禁用的公开分组
- **THEN** 系统使用现有 group-not-allowed 错误拒绝请求

#### Scenario: 用户将 API Key 更新到被禁用公开分组
- **WHEN** 用户更新 API Key 并切换到对自己禁用的公开分组
- **THEN** 系统使用现有 group-not-allowed 错误拒绝请求

### Requirement: 已有 API Key 遵守公开分组禁用规则
The system SHALL deny requests from existing API Keys whose bound public standard group is blocked for the owning user.

#### Scenario: 已有 Key 使用后来被禁用的公开分组
- **WHEN** API Key 已绑定公开标准分组，并且该分组后来被管理员对 Key 所属用户禁用
- **THEN** 请求在上游调度前鉴权失败

#### Scenario: 其他用户保持默认公开访问
- **WHEN** 其他用户没有该公开标准分组的禁用记录
- **THEN** 该用户仍可默认绑定并使用该分组的 API Key
