# Comet Design Handoff

- Change: block-public-groups-by-user
- Phase: design
- Mode: compact
- Context hash: c437db1dbdbc1f151fd19e33e632b6f5c9c1718f65fb5e34c3b0502faed0694a

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/block-public-groups-by-user/proposal.md

- Source: openspec/changes/block-public-groups-by-user/proposal.md
- Lines: 1-26
- SHA256: c9c6af023ee16fcd0bf2bf5dcda2f49b99d6df9abdbc0032cacbe884593ff36c

```md
## Why

公开标准分组目前默认对所有用户可用。管理员需要按用户设置例外，让分组保持公开状态的同时，对指定用户禁用访问。

## What Changes

- 为标准公开分组新增按用户维度的可见性禁用规则，底层使用可复用的用户资源覆盖表。
- 在现有用户分组配置弹窗中支持管理员开关公开分组访问。
- 被禁用的公开分组在用户侧分组选择和可用渠道页面中不可见。
- 在用户创建/更新 API Key 以及已有 API Key 请求鉴权时执行禁用规则。
- 保持专属分组授权和订阅分组行为不变。

## Capabilities

### New Capabilities
- `user-public-group-blocklist`: 管理员可以为单个用户禁用部分公开标准分组，其他用户仍保持默认可访问。

### Modified Capabilities

## Impact

- 后端用户/分组访问模型、DTO、Repository 和鉴权缓存快照。
- API Key 创建/更新授权，以及 API Key 请求鉴权。
- 管理员用户更新 API payload。
- 前端管理员用户分组配置弹窗、用户类型和 i18n 文案。
- 新用户资源覆盖关系的数据表和迁移；本次仅实现 group deny，后续菜单隐藏可复用。
```
## openspec/changes/block-public-groups-by-user/design.md

- Source: openspec/changes/block-public-groups-by-user/design.md
- Lines: 1-57
- SHA256: eb61d2e7350778bc3d19c37e8937626b11f6d1374385d4a39184a5f735c743be

```md
## Context

用户当前通过 `AllowedGroups` 表达专属标准分组授权，底层由 `user_allowed_groups` 维护。公开标准分组是非专属分组，`User.CanBindGroup` 目前会无条件返回 true。管理员用户分组配置弹窗也与此一致：公开分组固定显示为已选中。

新行为是公开访问的例外规则，不替代公开分组模型。公开分组必须继续默认可用，只有用户存在显式禁用记录时才不可见、不可绑定、不可继续通过已有 Key 使用。

## Goals / Non-Goals

**Goals:**
- 为公开标准分组新增按用户维度的轻量 deny-list 模型，并让用户侧列表隐藏被禁用分组。
- 使用可复用的用户资源覆盖表，为后续充值菜单和自定义菜单隐藏保留同一存储模型。
- 在现有用户分组配置弹窗中展示并保存公开分组开关。
- 对后续 API Key 绑定和已有 API Key 请求鉴权都执行禁用规则。
- 管理员修改后失效相关鉴权缓存。

**Non-Goals:**
- 不把公开分组改成白名单模型。
- 不改变订阅分组权益校验。
- 不自动解绑已有 API Key。
- 不新增单独的分组维度用户管理页面。
- 不在本 change 实现充值菜单或自定义菜单隐藏逻辑。

## Decisions

1. 新增可复用的 `user_resource_overrides` 关系。
   - Rationale: `user_allowed_groups` 已经表示专属分组的显式允许。复用它表达 deny 语义会让查询和 DTO 更难读；专用 `user_blocked_groups` 虽更短，但无法复用到后续菜单隐藏。
   - 表字段保持最小：`user_id`、`resource_type`、`resource_id`、`effect`、`created_at`；本次只实现 `resource_type='group'` 和 `effect='deny'`。
   - Alternative considered: 专用 `user_blocked_groups`；拒绝原因是已确认后续充值菜单和自定义菜单也需要按用户隐藏，通用覆盖表能复用同一持久化模型。

2. 扩展 `User`，增加 `BlockedGroups []int64` 作为 group deny 投影，并让 `CanBindGroup` 支持公开分组禁用判断。
   - Rationale: 把判断保留在 `CanBindGroup` 附近，可以复用现有 API Key service 流程，避免新增策略层。
   - ponytail: 这里只是直接 allow/deny 判断；只有出现更多分组规则时再引入 policy object。

3. 在鉴权缓存快照中包含 blocked groups。
   - Rationale: 已有 API Key 在缓存失效后必须立即失败，同时避免每次请求查数据库。

4. 复用现有用户更新接口，新增 `blocked_groups` 字段。
   - Rationale: 管理员弹窗已经保存 `allowed_groups` 和 `group_rates`，增加一个字段是最小 API 面。

5. 通过 `GetAvailableGroups` 统一隐藏用户侧禁用分组。
   - Rationale: API Key 创建/编辑分组选择和可用渠道页面都依赖可用分组列表；在 service 层过滤后，用户侧自然看不见被禁用分组。

## Risks / Trade-offs

- API Key 鉴权缓存陈旧时，禁用规则可能要等 TTL 才生效 -> blocked groups 变化时失效该用户的 API Key 鉴权缓存。
- exclusive 或 subscription 分组出现 block 记录会造成语义混乱 -> UI 只提交公开标准分组 ID，后端校验并忽略或拒绝非公开标准分组 ID。
- 通用覆盖表可能被误用为完整 RBAC -> 本次只暴露 group deny 的 repository/service 方法，不实现通用策略引擎。
- 现有测试有多处 user repository stub -> 只更新受影响 stub，并补充绑定和鉴权行为的聚焦测试。

## Migration Plan

- 创建 `user_resource_overrides` 表，使用 `(user_id, resource_type, resource_id)` 唯一约束、`user_id/resource_type` 查询索引和 `created_at`。
- 回滚方式是删除该表；没有 group deny 记录时会恢复现有公开访问行为。

## Open Questions

- 无。用户已确认按用户禁用、用户侧不可见、已有绑定该公开分组的 API Key 应立即失败，并确认采用可复用资源覆盖表。
```

## openspec/changes/block-public-groups-by-user/tasks.md

- Source: openspec/changes/block-public-groups-by-user/tasks.md
- Lines: 1-24
- SHA256: bc021a50dcc84b4eca3ab98e7e3410fcb9b9d5738311610e55461a80ff82b8ea

```md
## 1. 数据模型与 Repository

- [ ] 1.1 为 `user_resource_overrides` 增加 Ent schema 和数据库迁移，本次仅实现 group deny。
- [ ] 1.2 扩展 service user model、DTO mapping 和 repository 读写路径，将 group deny 投影为 `blocked_groups`。
- [ ] 1.3 校验管理员更新，确保 `blocked_groups` 只应用于公开标准分组。

## 2. 授权执行

- [ ] 2.1 更新 `GetAvailableGroups`，在用户侧隐藏被禁用的公开分组。
- [ ] 2.2 更新 API Key 创建/更新分组授权，拒绝被禁用的公开分组。
- [ ] 2.3 在 API Key 鉴权缓存快照和请求鉴权检查中包含 blocked groups。
- [ ] 2.4 blocked groups 变化时失效受影响的鉴权缓存。

## 3. 管理端 UI

- [ ] 3.1 在前端管理员用户类型和 API payload 中增加 `blocked_groups`。
- [ ] 3.2 让管理员用户分组配置弹窗中的公开标准分组可切换；用户侧列表不展示被禁用分组。
- [ ] 3.3 更新 zh/en 文案，说明公开分组默认可用，除非被禁用。

## 4. 验证

- [ ] 4.1 增加后端聚焦测试，覆盖用户侧隐藏、绑定拦截和已有 API Key 鉴权。
- [ ] 4.2 新增或更新前端测试，覆盖公开分组开关 payload。
- [ ] 4.3 运行目标后端和前端检查。
```

## openspec/changes/block-public-groups-by-user/specs/user-public-group-blocklist/spec.md

- Source: openspec/changes/block-public-groups-by-user/specs/user-public-group-blocklist/spec.md
- Lines: 1-38
- SHA256: dd6b5e68c6349df2082410ffd00d1b0199fca9a412b249af55e0eb99b3bb0648

```md
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
```
