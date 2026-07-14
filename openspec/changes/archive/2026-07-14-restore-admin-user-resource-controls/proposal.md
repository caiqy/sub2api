## Why

上游 `v0.1.151` 合并及 `admin_service.go` 拆分后，历史定制的用户屏蔽公开分组、隐藏购买页和隐藏自定义菜单逻辑未被迁移到 `admin_user.go`。管理端更新接口仍接受这些字段，但当前实现静默忽略，导致权限配置无法保存且鉴权缓存不会失效。

## What Changes

- 恢复 `UpdateUser` 对 blocked groups 的合法性校验、事务内持久化和返回值更新。
- 恢复隐藏购买页与自定义菜单资源的事务内持久化和返回值解析。
- 当上述权限资源发生变化时失效用户 auth cache。
- 保留当前 API、数据库结构和默认行为，不引入新 capability。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

无。此次变更仅恢复已有需求的实现，不修改需求契约。

## Impact

- 后端服务：`backend/internal/service/admin_user.go`
- 回归测试：`backend/internal/service/admin_service_blocked_groups_test.go`
- 复用现有 `UserRepository.SetBlockedGroups`、`SetHiddenUIResources` 和 Ent transaction，不新增依赖或迁移。
