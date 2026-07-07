## Why

上游 `Wei-Shaw/sub2api` 已发布 `v0.1.146`，本地当前最高定制版本停留在 `v0.1.143.4`。需要按既有上游合并惯例吸收 release 更新，同时保留本地网关、调度、隐私和前端定制行为。

## What Changes

- 从当前干净的 `main` 切出临时合并分支，目标优先使用 upstream release tag `v0.1.146`。
- 合并上游更新时优先保留上游修复和本地定制；遇到真实语义冲突时暂停让用户确认。
- 合并后执行后端测试、前端类型检查/构建，并专项 review 本地关键能力是否被上游语义覆盖。
- 本次不直接推送远端、不直接合回主分支、不新增业务 capability 或无关重构。

## Capabilities

### New Capabilities
- `upstream-release-sync`: 约束一次上游 release 合并从目标确认、隔离分支、冲突处理到验证和语义 review 的完整流程。

### Modified Capabilities
无。

## Impact

- Git：新增临时合并分支，合并 upstream `v0.1.146`，后续是否合回 `main` 另行确认。
- 后端：可能触及网关、账号调度、sticky、fallback、privacy、runtime setting 等上游改动区域。
- 前端：可能触及上游 UI、类型、构建配置或用户侧入口变更。
- 验证：至少覆盖 `go test ./...`、前端 typecheck/build，以及本地关键能力语义 review。
