## Why

上游 `Wei-Shaw/sub2api` 已发布 `v0.1.151`，本地 `main` 当前基于 `v0.1.146.4` 并包含独立定制与大输入内存优化。需要吸收正式 release 更新，同时避免上游变更静默覆盖本地关键行为。

## What Changes

- 从干净且已同步的本地 `main` 创建隔离分支，合并 upstream release tag `v0.1.151`，不包含该 tag 之后的 `upstream/main` 提交。
- 冲突处理优先同时保留上游修复与本地定制；遇到不可共存的业务语义时暂停并请求用户决策。
- 复核 `VERSION`、生成文件、配置和 migration 等易受合并影响的运行与发布元数据。
- 执行后端全量测试、前端单测、类型检查与构建，并专项审查 scheduler、sticky、privacy、image capability、runtime setting 热更新、网关透传和大输入内存优化。
- 本 change 不执行发布、部署或无关重构。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `upstream-release-sync`: 将大输入请求体保留、落盘与内存释放语义加入上游合并后的本地关键能力专项审查范围。

## Impact

- Git：基于当前 `main` 创建隔离合并分支并合入 upstream `v0.1.151`。
- 后端：可能涉及网关协议转换、账号调度、sticky/fallback、内容审计、请求体生命周期、配置和生成代码。
- 前端：可能涉及上游页面、组件、类型、依赖或构建配置更新。
- 验证：覆盖 `go test ./...`、`pnpm test:run`、`pnpm typecheck`、`pnpm build` 及本地关键能力语义审查。
