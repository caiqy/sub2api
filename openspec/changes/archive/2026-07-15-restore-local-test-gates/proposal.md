## Why

当前仓库的默认 Go 测试和前端全量测试能够通过，但 `-tags=unit` 全量测试存在稳定失败与测试 fixture 编译缺口，`golangci-lint` 也报告 47 项问题。本地门禁结果不一致会掩盖真实回归，无法作为变更完成依据。

## What Changes

- 修复稳定的 handler unit 失败、server 测试 fixture 接口缺口与 request body spool cleanup 的可重复性问题。
- 消除现有 `golangci-lint run ./...` 报告的问题，不放宽规则、不新增忽略项。
- 将本地全量验证定义为后端默认测试、后端 unit 测试、后端 lint、前端 ESLint、TypeScript 与全量 Vitest；integration/e2e 不纳入本地门禁。

## Capabilities

### New Capabilities

- `local-test-gates`: 本地代码质量门禁的可执行验证契约。

### Modified Capabilities

- 无。

## Impact

- 涉及后端 handler、server 测试 fixture、service request body 生命周期与 lint 报告覆盖的文件。
- 涉及根目录与前端的本地验证入口说明。
- 不修改公开 API、数据库 schema、依赖、Docker 环境或 integration/e2e 行为。
