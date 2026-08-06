## Why

`main@b576f73a` 在仓库既定的 Go 1.26.5 与 golangci-lint v2.9 配置下存在 144 项 backend lint 问题，导致 `make test` 无法通过，并阻塞后续 upstream 分段合并。upstream 原始 `v0.1.169` 与 `v0.1.171` 在相同门禁下均为 0 issues，因此需要先清理 fork-local 债务，恢复可信的绿色基线。

## What Changes

- 在不修改 `backend/.golangci.yml`、CI workflow、Go 版本或 lint 命令的前提下，修复 uncapped 清单中的 140 项 `ineffassign`、3 项 `staticcheck` 和 1 项 `unused`。
- 重构请求体及其派生副本的局部生命周期，使 lint 不再报告无效 `nil` 赋值，同时保持 spool、retry、failover、审计、计费和上游转发语义不变。
- 修正 3 项 QF1003（1 个生产条件链、2 个测试条件链），并删除确认无调用方的私有方法。
- 使用现有请求体内存保留测试、相关 package 测试、uncapped lint 和仓库级 `make test` 证明修复完整且没有行为回退。
- 不包含 breaking change。

## Capabilities

### New Capabilities

无。本 change 仅恢复实现对现有 `local-test-gates` 与 `request-body-retention-control` 规范的符合性，不新增 spec-level capability。

### Modified Capabilities

无。现有需求保持不变，因此本 change 通过 `skip_specs: true` 明确不创建 delta spec。

## Impact

- 默认影响范围限定为当前 39 个 lint 命中文件；若实际回退暴露现有测试无法覆盖的行为缺口，必须先更新 design、baseline manifest 与 tasks，经范围审查后才可修改额外 backend Go 测试。
- 不改变 public API、数据库 schema、配置格式、依赖、前端、部署、版本号或 upstream 合并内容。
- 主要风险是机械删除 `nil` 赋值可能破坏大请求在上游等待期间的内存释放合同；实现必须以作用域/ownership 重构和现有保留测试证明语义，而不能通过 lint suppression 隐藏问题。
