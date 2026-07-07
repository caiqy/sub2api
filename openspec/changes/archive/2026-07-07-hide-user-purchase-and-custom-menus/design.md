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
