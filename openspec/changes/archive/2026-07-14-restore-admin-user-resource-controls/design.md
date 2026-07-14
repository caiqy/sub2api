## Context

历史实现把用户基础字段、blocked groups 和隐藏 UI 资源放在同一 Ent transaction 中更新，并在提交成功后失效 auth cache。`v0.1.151` 将管理员服务拆到 `admin_user.go` 时只迁移了基础字段和 allowed groups，输入 DTO、仓储能力及回归测试仍然保留，因此接口出现“返回成功但配置未生效”的静默回归。

## Goals / Non-Goals

**Goals:**

- 在当前 `admin_user.go` 结构中恢复 blocked groups 的校验与持久化。
- 恢复隐藏购买页和自定义菜单资源的持久化及响应解析。
- 保证多类用户资源更新要么全部提交，要么全部回滚。
- 仅在 transaction commit 成功后失效 auth cache。

**Non-Goals:**

- 不新增管理员 API、数据库表或资源类型。
- 不调整 blocked groups 和 hidden UI 的既有业务规则。
- 不重构 `UserRepository` 或通用 transaction 基础设施。

## Decisions

1. 复用历史提交 `0b3fa21ad` 与 `ff4243b34` 的业务语义，但移植到当前 `admin_user.go`，不恢复旧的单体 `admin_service.go`。
2. 只要请求包含 blocked groups 或 hidden UI 字段，就创建一个 Ent transaction；`userRepo.Update`、`SetBlockedGroups`、`SetHiddenUIResources` 共用 `dbent.NewTxContext`。
3. blocked groups 在写入前调用现有 `validateBlockedGroups`，继续只允许可公开绑定的标准分组。
4. 自定义菜单 ID 使用现有 `customMenuIDsToResourceIDs` 归一化；响应使用 `resolveHiddenCustomMenuIDsForUser` 恢复可读 ID。
5. auth cache invalidation 放在 commit 之后，并将 blocked groups、隐藏购买页和隐藏菜单资源纳入变更检测。

## Risks / Trade-offs

- [Risk] transaction 内任一资源写入失败会使整个用户更新失败 → 这是预期的原子语义，避免部分权限生效。
- [Risk] 自定义菜单设置已删除的 ID 无法反解为字符串 → 延续现有 `ResolveHiddenCustomMenuIDs` 行为，不在本次修复扩展存储模型。
- [Risk] 无 `entClient` 的单元测试无法验证真实 transaction → 保留当前降级路径，并由 repository integration test 覆盖事务仓储行为。
