## Context

内容审计配置会保存 `group_ids`。当某个分组删除后，旧 ID 仍可能留在配置里；管理员再次保存任意审计设置时，`validateConfig` 会查这个旧 ID 并返回“审计分组不存在”。

## Goals / Non-Goals

**Goals:**

- 保存配置时清理已删除的审计分组 ID。
- 只吞掉 `ErrGroupNotFound`，其他查询错误继续返回。
- 用现有内容审计服务测试覆盖回归。

**Non-Goals:**

- 不修改 API 请求/响应结构。
- 不新增前端清理逻辑。
- 不调整分组删除流程。

## Decisions

- 在后端保存路径清理陈旧分组 ID；原因是配置持久化在后端，直接调用 API 也应可恢复。
- 保留 `ErrGroupNotFound` 以外错误；原因是数据库或仓储异常不能被误判为陈旧配置。

## Risks / Trade-offs

- 已删除分组会在保存时静默移除 -> 管理员不会再看到旧分组，但这是删除后的唯一可用状态。
