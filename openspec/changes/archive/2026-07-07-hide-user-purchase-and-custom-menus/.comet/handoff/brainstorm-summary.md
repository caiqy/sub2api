# Brainstorm Summary

- Change: hide-user-purchase-and-custom-menus
- Date: 2026-07-07

## 确认的技术方案

采用方案 A：复用 `user_resource_overrides`，新增 `resource_type='ui'`，`effect='deny'`。购买页使用固定资源 ID，自定义菜单 ID 用稳定哈希映射到 `resource_id`。登录态用户返回隐藏配置，前端过滤菜单、标题和页面，后端支付接口做强制校验。

## 关键取舍与风险

- 不新增表，避免过度设计；代价是自定义菜单字符串 ID 需要哈希映射到 int64。
- 哈希碰撞理论存在；可用集中 helper 和测试固定映射降低风险，未来需要审计时再迁移字符串资源表。
- public settings 仍包含全量 custom menu；实际用户 UI 和直达页面按登录态隐藏配置过滤。

## 测试策略

覆盖 repository UI deny round trip、管理员更新保存、用户 DTO 映射、支付接口拒绝、前端侧边栏/自定义页面过滤。

## Spec Patch

无。
