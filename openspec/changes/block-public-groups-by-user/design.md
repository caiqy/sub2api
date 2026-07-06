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
