## 1. 后端用户级 UI 覆盖

- [x] 1.1 扩展 `service.User`、用户 DTO 和管理员更新输入，表示隐藏购买页与隐藏自定义菜单 ID。
- [x] 1.2 复用 `user_resource_overrides` 增加 UI deny 的读写方法，并覆盖 repository round trip 测试。
- [x] 1.3 在管理员用户更新流程中读取、校验、保存 UI deny 配置。

## 2. 后端访问保护

- [x] 2.1 在登录态用户返回中携带当前用户的隐藏 UI 配置。
- [x] 2.2 在支付 checkout 和创建订单入口拒绝购买页被隐藏的用户。
- [x] 2.3 增加后端测试覆盖隐藏购买页和隐藏自定义菜单的关键场景。

## 3. 前端用户体验

- [x] 3.1 扩展前端用户类型和 auth store，持久化隐藏 UI 配置。
- [x] 3.2 在侧边栏、标题解析和自定义页面中过滤被隐藏的购买页与自定义菜单。
- [x] 3.3 在管理员用户配置 UI 中支持保存购买页和自定义菜单隐藏项。

## 4. 验证

- [x] 4.1 运行相关 Go 测试，覆盖 repository、admin service、payment handler。
- [x] 4.2 运行相关前端单测，覆盖 sidebar/custom page/admin user UI。
- [x] 4.3 更新任务状态并准备进入 Comet verify。
