---
change: hide-user-purchase-and-custom-menus
design-doc: docs/superpowers/specs/2026-07-07-hide-user-purchase-and-custom-menus-design.md
base-ref: b41f4fc166998198dfea8a704c01983fd5bb0884
---

# 隐藏用户购买页与自定义菜单实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 管理员可按用户隐藏整个购买页和指定自定义菜单；用户侧入口、直达页面和购买接口都遵守该配置。

**Architecture:** 复用 `user_resource_overrides`，新增 `resource_type='ui'` + `effect='deny'`。购买页使用固定 `resource_id=1`。自定义菜单持久化为字符串 ID 的稳定哈希；对外 API 仍使用字符串 ID，后端读取时用当前全局 `custom_menu_items` 把哈希映射回字符串列表。

**Constraints:** 不新增表；不做通用策略引擎；不改变全局 payment/custom menu 设置；不影响管理员菜单和其他用户。

## 1. 后端存储与用户 DTO

- [x] 1.1 先补 repository/service 测试：UI deny 可保存购买页和 custom menu 哈希，清空后为空；`CustomMenuResourceID("docs")` 稳定且为正数。
- [x] 1.2 在 `service.User` 增加 `HiddenPurchasePage bool`、`HiddenCustomMenuResourceIDs []int64`、`HiddenCustomMenuIDs []string`，并实现 `IsPurchasePageHidden`、`IsCustomMenuHidden(id string)`、`CustomMenuResourceID(id string)`。
- [x] 1.3 扩展 `UserRepository` 与 `user_repo.go`：读写 `resource_type='ui'` deny；加载用户时填充 purchase flag 和 custom menu hash IDs；用当前全局 custom menu 列表反解 admin/user DTO 需要的字符串 IDs。
- [x] 1.4 扩展管理员 `UpdateUser` input/request/DTO：支持 `hidden_purchase_page`、`hidden_custom_menu_ids`，保存时把字符串 ID 哈希后写入同一张表。
- [x] 1.5 运行并通过相关 Go 测试后提交：`feat: store user hidden ui resources`。

## 2. 后端访问保护

- [x] 2.1 补当前用户/profile DTO 测试，确认普通用户响应包含隐藏购买页和隐藏自定义菜单字符串 ID。
- [x] 2.2 在 `payment_handler.go` 的 `checkout-info` 和创建订单入口校验当前用户；若购买页隐藏，返回 forbidden，不调用后续支付逻辑。
- [x] 2.3 补 payment handler 测试：隐藏购买页用户访问 checkout 和创建订单都被拒绝。
- [x] 2.4 运行并通过相关 Go 测试后提交：`feat: reject hidden purchase access`。

## 3. 前端过滤与管理 UI

- [x] 3.1 扩展 `frontend/src/types/index.ts` 和 auth store 用户字段：`hidden_purchase_page`、`hidden_custom_menu_ids`。
- [x] 3.2 在 `AppSidebar.vue` 过滤普通用户的 `/purchase` 和被隐藏 custom menu；管理员菜单不受影响，并补 sidebar 测试。
- [x] 3.3 在 `router/index.ts` 阻止普通用户直达 `/purchase`；在标题解析和 `CustomPageView.vue` 过滤被隐藏 custom menu，防止直达加载内容。
- [x] 3.4 在管理员用户配置 UI 复用现有用户编辑/分组配置入口，保存隐藏购买页和隐藏 custom menu IDs；不新增独立策略页面。
- [x] 3.5 运行并通过相关前端测试/type-check 后提交：`feat: hide purchase and custom menus in ui`。

## 4. 验证与收尾

- [x] 4.1 运行后端相关测试：repository、admin service/handler、payment handler。
- [x] 4.2 运行前端相关测试：AppSidebar、CustomPage/Router 相关测试、type-check。
- [x] 4.3 手动 smoke：隐藏购买页用户无购买入口、`/purchase` 被拦截、支付接口 forbidden；隐藏 custom menu 不出现在侧边栏且直达不加载；其他用户不受影响。
- [x] 4.4 勾选 `openspec/changes/hide-user-purchase-and-custom-menus/tasks.md` 对应任务并提交：`docs: mark hidden ui menu tasks complete`。
