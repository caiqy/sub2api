# Brainstorm Summary

- Change: block-public-groups-by-user
- Date: 2026-07-06

## 确认的技术方案

已确认：需求修正为“用户侧看不见被禁用分组”，不是显示为 disabled。采用可复用的用户资源可见性覆盖表 `user_resource_overrides`，字段包含 `user_id`、`resource_type`、`resource_id`、`effect`、`created_at`，当前 change 只实现 `resource_type='group'` 且 `effect='deny'` 的公开标准分组禁用。后续充值菜单和自定义菜单可复用同一张表，以 `resource_type='menu'` 或更细的 `resource_type='custom_menu_item'` 表达。

后端在 `User` service model 上增加 `BlockedGroups []int64` 作为 group deny 的投影，Repository 在用户列表、详情、API Key auth 查询路径中加载该投影。`CanBindGroup` 扩展为先判断公开标准分组是否被 block，再保持公开分组默认允许、专属分组查 `AllowedGroups` 的现有语义。

管理端复用现有用户更新接口，新增 `blocked_groups` payload。前端复用 `UserAllowedGroupsModal`，将公开标准分组从固定勾选改为可切换；未勾选公开分组写入 `blocked_groups`，专属分组仍写入 `allowed_groups`。用户侧列表不展示 blocked public groups：`GetAvailableGroups` 统一过滤后，API Key 创建/编辑分组选择和可用渠道页面都会自然隐藏该分组。

已有 API Key 立即失效通过 auth cache 路径实现：`APIKeyAuthUserSnapshot` 增加 blocked groups，`GetByKeyForAuth` 加载禁用关系；管理员更新用户 blocked groups 后失效该用户 API Key auth cache。请求鉴权继续复用当前 group-not-allowed 错误。

## 关键取舍与风险

- 采用新覆盖表而不是复用 `user_allowed_groups`，避免 allow/deny 混在一个关系里。
- 表结构对后续菜单隐藏保留最小复用点，但本 change 只实现 group deny，避免把菜单功能提前做进来。
- 不新增策略层；当前规则仍是一个直接判断，放在 `CanBindGroup` 附近最小。
- 风险：auth cache 陈旧会延迟生效；缓解方式是 blocked groups 改变时失效该用户缓存。
- 风险：管理员 payload 可能包含专属/订阅分组；后端应校验并拒绝或过滤非公开标准分组。

## 测试策略

- 后端单测：`CanBindGroup` 覆盖公开默认允许、公开被禁用拒绝、专属 allow-list 不变。
- 后端 API Key service/middleware 测试：创建/更新 API Key 到 blocked public group 返回 `GROUP_NOT_ALLOWED`；已有 API Key 请求 blocked public group 返回 forbidden。
- Repository/DTO 测试：用户读写 `blocked_groups`，auth snapshot 包含 blocked groups。
- 前端测试：公开分组取消勾选后提交 `blocked_groups`，专属分组仍提交 `allowed_groups`。

## Spec Patch

无。
