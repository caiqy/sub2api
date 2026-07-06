## 1. 数据模型与 Repository

- [x] 1.1 为 `user_resource_overrides` 增加 Ent schema 和数据库迁移，本次仅实现 group deny。
- [x] 1.2 扩展 service user model、DTO mapping 和 repository 读写路径，将 group deny 投影为 `blocked_groups`。
- [x] 1.3 校验管理员更新，确保 `blocked_groups` 只应用于公开标准分组。

## 2. 授权执行

- [x] 2.1 更新 `GetAvailableGroups`，在用户侧隐藏被禁用的公开分组。
- [x] 2.2 更新 API Key 创建/更新分组授权，拒绝被禁用的公开分组。
- [x] 2.3 在 API Key 鉴权缓存快照和请求鉴权检查中包含 blocked groups。
- [x] 2.4 blocked groups 变化时失效受影响的鉴权缓存。

## 3. 管理端 UI

- [x] 3.1 在前端管理员用户类型和 API payload 中增加 `blocked_groups`。
- [x] 3.2 让管理员用户分组配置弹窗中的公开标准分组可切换；用户侧列表不展示被禁用分组。
- [x] 3.3 更新 zh/en 文案，说明公开分组默认可用，除非被禁用。

## 4. 验证

- [x] 4.1 增加后端聚焦测试，覆盖用户侧隐藏、绑定拦截和已有 API Key 鉴权。
- [x] 4.2 新增或更新前端测试，覆盖公开分组开关 payload。
- [x] 4.3 运行目标后端和前端检查。
