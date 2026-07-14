## 1. 恢复 blocked groups 更新语义

- [x] 1.1 在 `admin_user.go` 恢复 blocked groups 合法性校验、用户快照更新和事务内 `SetBlockedGroups`。
- [x] 1.2 将 blocked groups 变化纳入 commit 后 auth cache 失效条件。

## 2. 恢复隐藏 UI 资源更新语义

- [x] 2.1 在同一 transaction 内持久化隐藏购买页和隐藏自定义菜单资源。
- [x] 2.2 恢复隐藏菜单 resource ID 归一化及更新响应中的可读菜单 ID。

## 3. 验证

- [x] 3.1 运行管理员用户资源控制聚焦 unit tests。
- [x] 3.2 运行用户资源仓储 integration tests 和 `go test ./internal/service`。
