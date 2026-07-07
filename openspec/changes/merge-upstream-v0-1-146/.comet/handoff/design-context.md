# Comet Design Handoff

- Change: merge-upstream-v0-1-146
- Phase: design
- Mode: compact
- Context hash: 3c4eb80ff314ab912ef6728ce23e025cd087e06cdb0670c617d8487d788ed3d1

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/merge-upstream-v0-1-146/proposal.md

- Source: openspec/changes/merge-upstream-v0-1-146/proposal.md
- Lines: 1-25
- SHA256: 7fd6659b83fa842b99f30fd60b8cf4f719ec6c28ca2039e52174e0e0337415c9

```md
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

```

## openspec/changes/merge-upstream-v0-1-146/design.md

- Source: openspec/changes/merge-upstream-v0-1-146/design.md
- Lines: 1-36
- SHA256: 55d9bbd91b4f8bfc04d43e8797a292a74869de8acd38c95768ac00512ac43c14

```md
## Context

当前本地 `main` 干净，`origin/main` 与本地 `HEAD` 一致；本地最高定制 tag 为 `v0.1.143.4`，upstream 远端已有 release tag `v0.1.146`。仓库已有上游合并惯例要求先确认真实目标，优先在临时分支合并，合并后不仅跑测试，还要复核本地定制能力是否被上游语义覆盖。

## Goals / Non-Goals

**Goals:**
- 以 upstream `v0.1.146` 作为本次合并目标。
- 从当前 `main` 创建隔离合并分支，避免直接污染主线。
- 冲突处理时保留上游修复和本地定制；无法共存时暂停让用户确认。
- 合并后执行后端、前端验证，并专项 review 本地关键能力。

**Non-Goals:**
- 不直接推送远端或合回 `main`。
- 不新增业务功能、public API 或数据库 schema。
- 不借合并机会做无关重构。

## Decisions

1. 使用 upstream release tag `v0.1.146`，而不是默认 `upstream/main`。
   - 原因：仓库惯例明确要求区分分支与 release tag；当前有效更新来自 upstream 新 release。
   - 替代方案：直接合并 `upstream/main`。放弃原因是它可能包含未发布分支状态，范围不如 release tag 清晰。

2. 在临时分支完成合并和验证。
   - 原因：上游更新可能跨后端、前端、配置和生成文件；隔离分支能保留回退空间。
   - 替代方案：直接在 `main` 合并。放弃原因是冲突或语义回归会影响主线。

3. 验证分两层执行：自动检查 + 语义 review。
   - 自动检查覆盖 `go test ./...`、前端 typecheck/build。
   - 语义 review 聚焦 scheduler、sticky、privacy、image capability、runtime setting 热更新、网关透传字段。

## Risks / Trade-offs

- upstream 与本地定制在同一行为路径上冲突 -> 先保留两边能共存的语义；不能共存时暂停让用户选择。
- 测试通过但本地关键能力被静默改写 -> 增加面向本地能力清单的专项 review。
- release tag 与 `upstream/main` 存在差异 -> 本次以 `v0.1.146` 为准；若用户需要主线分支另开变更处理。

```

## openspec/changes/merge-upstream-v0-1-146/tasks.md

- Source: openspec/changes/merge-upstream-v0-1-146/tasks.md
- Lines: 1-21
- SHA256: 7e668a9b234c55d64ea2b497dc77f9e32b545d4e32b8b39cf22e030225601cb9

```md
## 1. 合并准备

- [ ] 1.1 获取 upstream 分支和 tags，确认 `v0.1.146` 可用且当前 `main` 干净。
- [ ] 1.2 从当前 `main` 创建临时合并分支。

## 2. 执行上游合并

- [ ] 2.1 合并 upstream `v0.1.146` 到临时分支。
- [ ] 2.2 处理冲突，优先保留上游更新和本地定制；不可共存语义暂停确认。
- [ ] 2.3 复核版本、生成文件和配置差异，避免发布或运行元数据被误改。

## 3. 验证与语义 review

- [ ] 3.1 运行 `go test ./...` 并修复合并引入的问题。
- [ ] 3.2 运行前端 typecheck 和 build，修复合并引入的问题。
- [ ] 3.3 专项 review scheduler、sticky、privacy、image capability、runtime setting 热更新和网关透传字段。

## 4. 收尾决策

- [ ] 4.1 汇总合并结果、冲突处理、验证输出和本地关键能力 review 结论。
- [ ] 4.2 让用户确认是否合回 `main`、是否推送远端。

```

## openspec/changes/merge-upstream-v0-1-146/specs/upstream-release-sync/spec.md

- Source: openspec/changes/merge-upstream-v0-1-146/specs/upstream-release-sync/spec.md
- Lines: 1-37
- SHA256: ccdb34d8d5e30fffbf96761e4ce33828913cc7178b49c6913189139fa1a42a63

```md
## ADDED Requirements

### Requirement: 确认上游合并目标
维护流程 SHALL 在合并前确认本地当前版本、upstream 最新 release tag、以及目标分支或 tag 的选择理由。

#### Scenario: 选择 upstream release tag
- **WHEN** upstream 存在比本地当前定制版本更新的 release tag
- **THEN** 维护流程记录目标 tag，并说明为何不默认使用 `upstream/main`

### Requirement: 在隔离分支执行合并
维护流程 SHALL 从干净的本地主线创建临时分支执行上游合并，除非用户明确选择其他隔离方式。

#### Scenario: 创建临时合并分支
- **WHEN** 本地 `main` 干净且已确认目标 upstream tag
- **THEN** 维护流程在临时分支中执行合并，不直接改写 `main`

### Requirement: 保留上游更新和本地定制
维护流程 MUST 在冲突处理时优先保留上游修复和本地定制能力；若两者语义不能共存，必须暂停等待用户确认。

#### Scenario: 冲突能力可以共存
- **WHEN** upstream 更新和本地定制修改同一文件但行为可以同时成立
- **THEN** 合并结果同时保留上游更新和本地定制语义

#### Scenario: 冲突能力不能共存
- **WHEN** upstream 更新和本地定制在同一行为上存在不可共存语义
- **THEN** 维护流程停止自动处理并请求用户选择保留策略

### Requirement: 合并后验证本地关键能力
维护流程 SHALL 在合并后执行自动验证，并专项 review 本地关键能力是否仍然成立。

#### Scenario: 自动验证通过
- **WHEN** 合并完成且无冲突残留
- **THEN** 维护流程运行后端测试、前端类型检查和构建验证

#### Scenario: 本地关键能力专项 review
- **WHEN** 自动验证完成
- **THEN** 维护流程复核 scheduler、sticky、privacy、image capability、runtime setting 热更新和网关透传字段等本地关键能力

```
