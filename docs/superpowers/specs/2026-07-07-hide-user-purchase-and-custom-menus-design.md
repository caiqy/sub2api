---
comet_change: hide-user-purchase-and-custom-menus
role: technical-design
canonical_spec: openspec
---

# 用户购买页与自定义菜单隐藏设计

## 背景

公开分组禁用已经使用 `user_resource_overrides` 表表达用户级 deny。本次继续复用这张表，把同类能力扩展到 UI 入口：管理员可以对指定用户隐藏整个购买页和指定自定义菜单，且用户不能通过直达 URL 或直接调用购买接口绕过。

全局 `payment_enabled` 和 `custom_menu_items` 仍是站点级设置，不承载用户级过滤。用户级过滤随登录态用户数据下发，前端按当前用户过滤展示，后端对购买接口做强制校验。

## 技术方案

新增 UI deny 语义，不新增表：

- `resource_type='ui'`
- `effect='deny'`
- `resource_id=1` 表示购买页
- 自定义菜单使用其字符串 `id` 的稳定 64-bit 哈希映射为正 `int64`

后端在 `service.User` 增加隐藏 UI 配置字段，并提供 helper 判断：购买页是否隐藏、自定义菜单 ID 是否隐藏。repository 复用 `UserResourceOverride` 增加 UI deny 读写方法，管理员 `UpdateUser` 接收并保存这些字段。普通用户 DTO 返回当前用户的隐藏 UI 配置，管理员 DTO 同时返回可编辑配置。

购买页被隐藏时，后端支付入口必须拒绝：`checkout-info` 和创建订单都不能继续提供充值或订阅购买能力。前端路由和侧边栏也隐藏购买入口并阻止直达 `/purchase`。

自定义菜单被隐藏时，前端从侧边栏、标题解析和 `CustomPageView` 中过滤该菜单。因为自定义菜单内容来自 public settings，直达页面仍由登录态过滤决定，不新增登录态 settings endpoint。

## 关键取舍

选择复用 `user_resource_overrides`，而不是新增 `user_ui_overrides` 表。原因是当前只需要 deny 列表，现有表已有 user/resource/effect 三元组，新增表会带来迁移、仓储和测试成本。

自定义菜单字符串 ID 用哈希映射到 `resource_id`。这避免改表结构，但存在理论碰撞。实现时把哈希逻辑集中到 helper，并用固定测试保护映射稳定性；若未来需要审计可读性或碰撞兜底，再迁移到字符串资源表。

购买页隐藏覆盖充值和订阅购买。这是当前确认范围；后续若要只隐藏余额充值，可新增独立 UI resource ID。

## 测试策略

- repository：UI deny 写入、读取、清空、自定义菜单 ID 映射稳定。
- admin service/handler：管理员保存隐藏购买页和隐藏自定义菜单，返回用户 DTO 正确。
- payment handler：隐藏购买页用户访问 checkout-info 或创建订单被拒绝。
- frontend：用户侧 sidebar 不显示隐藏购买入口/自定义菜单，`CustomPageView` 不展示被隐藏菜单，管理员用户配置能提交隐藏项。

## 不做事项

- 不新增数据库表。
- 不做通用策略引擎。
- 不改变全局公开设置语义。
- 不影响管理员菜单和未配置隐藏项的其他用户。
