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
