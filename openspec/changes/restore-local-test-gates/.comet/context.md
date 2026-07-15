# Comet Design Handoff

- Change: restore-local-test-gates
- Phase: design
- Mode: compact
- Context hash: 09db4530a2cb3ebb5614dd663bfd422f54a275e0722737a3c17e2c507dacfa6e

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/restore-local-test-gates/proposal.md

- Source: openspec/changes/restore-local-test-gates/proposal.md
- Lines: 1-25
- SHA256: 46f741b53fcded3e66acca863bc8c0cb147fdfc0775acfac8b07181db1775d29

```md
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

```

## openspec/changes/restore-local-test-gates/design.md

- Source: openspec/changes/restore-local-test-gates/design.md
- Lines: 1-30
- SHA256: 574c371de120593614fef15c0858f3e3fa84dd9c5aedcebc5f17f1742e9e170b

```md
## Context

本地验证目前出现三种不一致：稳定的 handler unit 断言失败、server unit 测试 fixture 无法满足已扩展的 `UserRepository` 接口、以及 `golangci-lint` 的历史和近期问题。request body spool cleanup 只在并行全套中失败，单独连续运行通过，必须先确认是否存在共享临时文件或异步清理竞争。

## Goals / Non-Goals

**Goals:**
- 使后端默认测试、后端 unit 测试、后端 lint、前端 ESLint、TypeScript 与全量 Vitest 在本地稳定通过。
- 让测试断言遵循 HTTP header 大小写无关语义，并让 fixture 与生产接口契约同步。
- 对每个 lint 诊断修复根因，不降低 linter 覆盖或添加全局排除。

**Non-Goals:**
- 不安装 Docker，不运行或修改 integration/e2e。
- 不改变公开 API、数据库 schema、上游协议或产品能力。
- 不把偶发 spool cleanup 失败视为生产缺陷，除非最小复现能够证明清理路径错误。

## Decisions

- 按失败域而非文件数量分阶段：先修 unit 可执行性与稳定失败，再修 lint，最后从冷缓存做全量复验。这使每一阶段都有独立失败信号。
- HTTP header 测试通过 `http.Header.Get` 或大小写无关比较检查语义，不依赖序列化文本中的 canonicalization 格式。
- server 测试 stub 显式实现新增 repository 方法，返回测试所需的零值或受控值；不修改生产接口以迁就测试。
- spool cleanup 使用重复、并行与临时路径隔离的复现矩阵判断根因；只有稳定复现后才在实际资源所有权边界添加清理。
- lint 按诊断类别分组处理：依赖边界、资源/取消、无效赋值、静态分析与未使用符号；每组运行对应 package 或完整 linter 验证。

## Risks / Trade-offs

- [大范围 lint 修复可能改变资源生命周期] → 每个类别保留聚焦测试，并在合并前运行完整后端默认和 unit 套件。
- [header 测试改写可能掩盖真实头缺失] → 断言值和关键 content type，只有名称比较改为大小写无关。
- [spool 问题可能为测试竞争] → 不在无复现证据时修改生产清理逻辑，先固定测试隔离或生命周期等待条件。
- [本地全量门禁耗时较长] → 仅最终阶段从冷缓存运行一次；开发阶段使用目标 package 验证。

```

## openspec/changes/restore-local-test-gates/tasks.md

- Source: openspec/changes/restore-local-test-gates/tasks.md
- Lines: 1-20
- SHA256: 0dd96a2347be1df8e384a7cd2094ec48b8b4ca2bfe9c9fe05b74c55ab159a499

```md
## 1. 恢复稳定 unit 测试

- [ ] 1.1 将 Anthropic failed-usage 测试改为按 HTTP header 语义断言，并运行对应 handler 测试。
- [ ] 1.2 修复 Images failover 耗尽时的错误响应契约，并运行对应 handler 测试。
- [ ] 1.3 让 server 与 middleware 测试 fixture 实现当前 `UserRepository` 接口，并运行两个 package 的 unit 测试。

## 2. 确认 request body spool 生命周期

- [ ] 2.1 通过重复与并行运行复现或排除 OpenAI request body spool cleanup 失败。
- [ ] 2.2 仅在稳定复现确认后修复资源所有权边界，并运行相关 service 测试。

## 3. 修复静态检查

- [ ] 3.1 修复依赖边界、资源关闭和 context cancel 诊断，并运行受影响 package 测试与 lint。
- [ ] 3.2 修复无效赋值、静态分析和未使用符号诊断，不降低 lint 规则。

## 4. 固化并验证本地门禁

- [ ] 4.1 更新本地测试入口以覆盖后端默认测试、后端 unit、lint 和前端全量验证，不调用 integration/e2e。
- [ ] 4.2 从冷缓存运行全部本地门禁并记录结果。

```

## openspec/changes/restore-local-test-gates/specs/local-test-gates/spec.md

- Source: openspec/changes/restore-local-test-gates/specs/local-test-gates/spec.md
- Lines: 1-23
- SHA256: 40164170fa25973e39564b693ca361632c55bba369c8f43d7ffa783d3ebcce35

```md
## ADDED Requirements

### Requirement: 本地代码质量门禁
仓库 MUST 使以下本地验证命令在不依赖 Docker 的开发环境中通过：后端默认测试、后端 unit 测试、后端 `golangci-lint`、前端 ESLint、前端 TypeScript 检查和前端全量 Vitest。integration/e2e 不属于本地代码质量门禁。

#### Scenario: 完整本地验证
- **WHEN** 开发者在依赖已安装的本地工作区执行定义的后端与前端验证命令
- **THEN** 所有命令以退出码 0 结束，且后端 unit 测试不会因测试 fixture 接口缺口而编译失败

#### Scenario: 不使用 Docker 的验证环境
- **WHEN** 开发者的工作站未安装 Docker
- **THEN** 本地代码质量门禁仍可完整执行，且不会要求启动 integration/e2e

### Requirement: 测试协议断言与资源清理
测试 MUST 断言 HTTP 协议语义而非 header 序列化大小写格式；request body 临时 spool 文件的清理测试 MUST 在重复和并行执行时具有稳定结果。

#### Scenario: HTTP header canonicalization
- **WHEN** 上游请求 header 由 Go 的 `http.Header` canonicalize
- **THEN** 相关测试仍验证关键 header 值和 content type，而不因名称大小写变化失败

#### Scenario: 并行 spool 清理
- **WHEN** request body spool cleanup 测试与同 package 的其他测试并行执行
- **THEN** 测试能够区分自身创建的临时文件，并稳定验证正确的清理生命周期

```
