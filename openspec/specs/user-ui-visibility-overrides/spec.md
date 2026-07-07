# user-ui-visibility-overrides Specification

## Purpose

Define per-user UI visibility overrides for hiding purchase flows and user-visible custom menu entries without changing global settings.
## Requirements
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
