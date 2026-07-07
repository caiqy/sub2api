# Comet Design Handoff

- Change: hide-user-purchase-and-custom-menus
- Phase: design
- Mode: compact
- Context hash: 2bc089f3eb4b13f5f1e26caa42ef9d3aadbbadafa863f70e3813ab924680fd93

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/hide-user-purchase-and-custom-menus/proposal.md

- Source: openspec/changes/hide-user-purchase-and-custom-menus/proposal.md
- Lines: 1-26
- SHA256: 009ddb4c7df91e923d520b8b0b2dab132c13aa8f0296f01b84540cde52f1fd34

```md
## Why

管理员已经可以按用户禁用公开分组，且底层 `user_resource_overrides` 表预留了复用空间。现在需要把同一类用户级 deny 能力用于界面入口：对指定用户隐藏购买页和自定义菜单，避免全局设置影响所有用户。

## What Changes

- 增加管理员按用户配置隐藏购买页和自定义菜单的能力。
- 用户被隐藏购买页后，侧边栏不显示购买入口，直达 `/purchase` 也不可继续使用充值或订阅购买。
- 用户被隐藏某个自定义菜单后，侧边栏、页面标题解析和直达自定义页面都不再暴露该菜单。
- 复用 `user_resource_overrides` 存储 deny 关系，不新增表，不引入通用策略引擎。
- 其他用户、管理员视角、全局公开设置保持现有行为。

## Capabilities

### New Capabilities
- `user-ui-visibility-overrides`: 管理员可以为单个用户隐藏购买页和指定自定义菜单，用户侧入口和直达访问都遵守该配置。

### Modified Capabilities
- `user-public-group-blocklist`: 继续复用同一张用户资源覆盖表；本变更不修改公开分组禁用的需求语义。

## Impact

- 后端：用户资源覆盖 repository/service、管理员用户更新接口、用户 DTO、公开设置或登录态用户配置、支付 checkout/order 入口保护。
- 前端：管理员用户配置 UI、用户类型、侧边栏过滤、自定义页面访问、购买页路由/入口处理。
- 数据库：不新增表；复用现有 `user_resource_overrides.resource_type/resource_id/effect`。
- 测试：覆盖 repository round trip、管理员保存、用户侧过滤、购买页直达保护和自定义菜单直达保护。
```

## openspec/changes/hide-user-purchase-and-custom-menus/design.md

- Source: openspec/changes/hide-user-purchase-and-custom-menus/design.md
- Lines: 1-44
- SHA256: 701504137bb67324f62af922d84ee7125e91321f87bc8ede970a5b0001b3a5bb

```md
## Context

现有公开分组禁用已经引入 `user_resource_overrides`，当前只使用 `resource_type='group'`、`effect='deny'`。购买页入口来自前端侧边栏和 `/purchase` 路由，支付后端通过 `checkout-info` 和创建订单接口提供购买能力。自定义菜单来自全局公开设置 `custom_menu_items`，前端侧边栏、标题解析和 `CustomPageView` 都读取同一份公开设置。

本次需求是用户级可见性控制，不是全局开关。管理员需要对单个用户隐藏整个购买页和指定自定义菜单；用户不应通过直达 URL 绕过隐藏配置。

## Goals / Non-Goals

**Goals:**
- 复用 `user_resource_overrides` 保存用户级 UI deny 配置。
- 管理员可读取并保存用户隐藏购买页、隐藏自定义菜单配置。
- 用户侧菜单、标题和页面访问遵守隐藏配置。
- 后端购买接口阻止被隐藏购买页的用户继续创建充值或订阅订单。

**Non-Goals:**
- 不新增数据库表。
- 不实现通用策略引擎或任意资源权限系统。
- 不改变全局 `payment_enabled`、`custom_menu_items` 配置语义。
- 不影响管理员菜单和其他用户。

## Decisions

1. 继续使用 `user_resource_overrides`，新增 `resource_type='ui'`，`effect='deny'`。
   - `resource_id=1` 表示购买页。
   - 自定义菜单使用菜单 `id` 经稳定哈希映射为正 `int64`，避免新增字符串资源 ID 列。
   - 替代方案：新增 JSON 字段或新表保存字符串菜单 ID。放弃原因是本次只需要 deny 列表，现有表足够。

2. 在 service.User 上新增 `HiddenUIResources`，并提供最小 helper 判断购买页和自定义菜单是否隐藏。
   - repository 负责读写 `resource_type='ui'` 的 deny 记录。
   - 管理员 UpdateUser 继续作为保存入口，避免新增管理 API。

3. 用户侧配置随登录态用户返回，而不是修改公开 settings 接口。
   - 公共 settings 仍可匿名读取，不能包含用户级过滤结果。
   - 前端在已登录用户态按 `authStore.user.hidden_*` 过滤侧边栏、自定义页面和标题。

4. 后端支付接口做强制校验。
   - 被隐藏购买页的用户访问 `checkout-info` 或创建订单时返回现有错误响应。
   - 前端隐藏只改善体验，不能作为唯一保护。

## Risks / Trade-offs

- 自定义菜单 ID 哈希存在理论碰撞 → 使用 64-bit FNV-1a 并把逻辑集中在 helper；若未来需要强一致审计，再迁移到字符串资源表。
- 前端 public settings 仍包含全量菜单 → 后端直达保护阻止页面展示；如未来要求网络响应也完全不含菜单，需要新增登录态 settings endpoint。
- 购买页被隐藏会同时影响充值和订阅 → 这是当前确认范围；后续如需拆分，可新增独立资源 ID。
```

## openspec/changes/hide-user-purchase-and-custom-menus/tasks.md

- Source: openspec/changes/hide-user-purchase-and-custom-menus/tasks.md
- Lines: 1-23
- SHA256: 4bae09503a4e138859dddff20134689f7a80bb9ff36ef5d7e3d7b2499d9adf65

```md
## 1. 后端用户级 UI 覆盖

- [ ] 1.1 扩展 `service.User`、用户 DTO 和管理员更新输入，表示隐藏购买页与隐藏自定义菜单 ID。
- [ ] 1.2 复用 `user_resource_overrides` 增加 UI deny 的读写方法，并覆盖 repository round trip 测试。
- [ ] 1.3 在管理员用户更新流程中读取、校验、保存 UI deny 配置。

## 2. 后端访问保护

- [ ] 2.1 在登录态用户返回中携带当前用户的隐藏 UI 配置。
- [ ] 2.2 在支付 checkout 和创建订单入口拒绝购买页被隐藏的用户。
- [ ] 2.3 增加后端测试覆盖隐藏购买页和隐藏自定义菜单的关键场景。

## 3. 前端用户体验

- [ ] 3.1 扩展前端用户类型和 auth store，持久化隐藏 UI 配置。
- [ ] 3.2 在侧边栏、标题解析和自定义页面中过滤被隐藏的购买页与自定义菜单。
- [ ] 3.3 在管理员用户配置 UI 中支持保存购买页和自定义菜单隐藏项。

## 4. 验证

- [ ] 4.1 运行相关 Go 测试，覆盖 repository、admin service、payment handler。
- [ ] 4.2 运行相关前端单测，覆盖 sidebar/custom page/admin user UI。
- [ ] 4.3 更新任务状态并准备进入 Comet verify。
```

## openspec/changes/hide-user-purchase-and-custom-menus/specs/user-ui-visibility-overrides/spec.md

- Source: openspec/changes/hide-user-purchase-and-custom-menus/specs/user-ui-visibility-overrides/spec.md
- Lines: 1-53
- SHA256: 74d14eddc8263a02341e44e3c8163d550d72dc53a57774185eb5997978ddf06b

```md
## ADDED Requirements

### Requirement: 管理员可以按用户隐藏购买页
The system SHALL allow an administrator to hide the purchase page for an individual user without changing global payment settings.

#### Scenario: 管理员保存隐藏购买页配置
- **WHEN** 管理员为某个用户启用购买页隐藏并保存用户配置
- **THEN** 系统记录该用户的购买页 deny 配置，并且其他用户的购买页可见性不受影响

#### Scenario: 管理员取消隐藏购买页配置
- **WHEN** 管理员取消某个用户的购买页隐藏并保存用户配置
- **THEN** 系统删除该用户的购买页 deny 配置

### Requirement: 被隐藏购买页不能被用户访问或使用
The system SHALL prevent a user with purchase page hidden from seeing or using purchase flows.

#### Scenario: 用户侧不展示购买入口
- **WHEN** 用户存在购买页 deny 配置并打开用户侧导航
- **THEN** 系统不展示购买页入口

#### Scenario: 用户直达购买页
- **WHEN** 用户存在购买页 deny 配置并访问 `/purchase`
- **THEN** 系统阻止该用户继续查看购买页

#### Scenario: 用户绕过前端创建购买订单
- **WHEN** 用户存在购买页 deny 配置并调用购买订单创建接口
- **THEN** 系统拒绝创建充值或订阅订单

### Requirement: 管理员可以按用户隐藏自定义菜单
The system SHALL allow an administrator to hide selected user-visible custom menu items for an individual user without changing global custom menu settings.

#### Scenario: 管理员保存隐藏自定义菜单配置
- **WHEN** 管理员为某个用户选择一个或多个自定义菜单并保存为隐藏
- **THEN** 系统记录该用户与这些自定义菜单的 deny 配置

#### Scenario: 其他用户仍可看到全局自定义菜单
- **WHEN** 其他用户没有对应自定义菜单 deny 配置
- **THEN** 系统继续按全局 custom menu settings 展示这些菜单

### Requirement: 被隐藏自定义菜单不能被用户访问
The system SHALL prevent a user from seeing or opening custom menu pages hidden for that user.

#### Scenario: 用户侧不展示被隐藏自定义菜单
- **WHEN** 用户存在某个自定义菜单 deny 配置并打开用户侧导航
- **THEN** 系统不展示该自定义菜单入口

#### Scenario: 用户直达被隐藏自定义页面
- **WHEN** 用户存在某个自定义菜单 deny 配置并访问该自定义菜单页面
- **THEN** 系统不展示该自定义页面内容

#### Scenario: 未隐藏的自定义菜单保持可用
- **WHEN** 用户没有某个自定义菜单 deny 配置
- **THEN** 系统继续按全局 custom menu settings 展示并打开该自定义菜单
```

