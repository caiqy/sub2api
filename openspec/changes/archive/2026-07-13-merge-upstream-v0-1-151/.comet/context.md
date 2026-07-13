# Comet Design Handoff

- Change: merge-upstream-v0-1-151
- Phase: design
- Mode: compact
- Context hash: 458b0ed8c70da26dfff42937e764019079fb0d6122c77f22ce1ce9be937effd2

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/merge-upstream-v0-1-151/proposal.md

- Source: openspec/changes/merge-upstream-v0-1-151/proposal.md
- Lines: 1-28
- SHA256: 56cd73a9e86bd96ceef25381f3ada885642987996589731c9df3f9fa7f29d3cc

```md
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

```

## openspec/changes/merge-upstream-v0-1-151/design.md

- Source: openspec/changes/merge-upstream-v0-1-151/design.md
- Lines: 1-41
- SHA256: b7a5a3a13d06bc3987e7564d3b1e7fd4d98ad262a4c62d930a4e811857be202d

```md
## Context

当前本地 `main` 已合入大输入内存优化并同步到 `origin/main`，版本为 `v0.1.146.4`。upstream 最新正式 release tag 为 `v0.1.151`；`upstream/main` 比该 tag 额外前进 40 个提交。本次需要吸收正式版更新，同时保护本地网关、调度、隐私和请求体生命周期定制。

## Goals / Non-Goals

**Goals:**
- 以 upstream `v0.1.151` 作为唯一上游合并目标。
- 在隔离分支完成合并、冲突协调、修复和验证。
- 同时保留可共存的上游修复与本地定制，不可共存时交由用户决策。
- 通过自动检查和能力级 review 证明本地关键行为未回退。

**Non-Goals:**
- 不合入 `v0.1.151` 之后的 `upstream/main` 提交。
- 不在本 change 内发布版本或部署服务器。
- 不新增与上游合并无关的业务能力、public API、schema 或重构。

## Decisions

1. 合并目标固定为 release tag `v0.1.151`，不使用 `upstream/main`。
   - 原因：release tag 范围稳定、可追踪，避免额外引入 40 个尚未发布提交。
   - 替代方案：直接合并 `upstream/main`。放弃原因是范围会随远端变化且包含未发布内容。

2. 从同步后的 `main` 创建独立 merge 分支。
   - 原因：本地与 upstream 分叉较大，隔离分支为冲突处理、专项修复和回退保留清晰边界。
   - 替代方案：直接在 `main` 合并。放弃原因是失败或半完成状态会污染主线。

3. 冲突按业务语义协调，不机械选择 ours 或 theirs。
   - 可共存时同时保留上游更新与本地能力；不可共存时暂停并列出影响供用户选择。
   - `VERSION`、配置、生成文件和 migration 单独复核，避免文本无冲突但运行语义错误。

4. 验证分为自动检查与能力级 review。
   - 自动检查运行后端全量测试、前端单测、typecheck 和 build。
   - 能力级 review 聚焦 scheduler、sticky/fallback、privacy、image capability、runtime setting 热更新、网关透传及大输入请求体保留、落盘和释放路径。

## Risks / Trade-offs

- 上游与本地在同一路径存在不可共存语义 → 停止自动决策并请求用户选择。
- 文本合并成功但本地行为被静默覆盖 → 使用关键能力清单逐项 review，并为发现的回归补失败测试后最小修复。
- 上游生成文件或依赖变化导致构建不一致 → 单独核对依赖、生成代码、migration 和构建产物。
- 全量验证耗时较长 → 保留完整验证，因为跨版本合并的影响面无法用少量定向测试可靠覆盖。

```

## openspec/changes/merge-upstream-v0-1-151/tasks.md

- Source: openspec/changes/merge-upstream-v0-1-151/tasks.md
- Lines: 1-27
- SHA256: e3217cd688f387380015cfdc90e87c549553d3e8372c5c15e0aae8a21c33cb7e

```md
## 1. 合并准备

- [ ] 1.1 确认本地 `main` 与 `origin/main` 同步、工作树干净，并记录 upstream `v0.1.151` 的目标提交。
- [ ] 1.2 从当前 `main` 创建独立 merge 分支，并记录合并前基线。

## 2. 执行上游合并

- [ ] 2.1 合并 upstream `v0.1.151`，记录冲突文件和上游主要变更范围。
- [ ] 2.2 按业务语义协调冲突，同时保留可共存的上游修复与本地定制；不可共存时暂停请求用户决策。
- [ ] 2.3 复核 `VERSION`、依赖、配置、生成文件和 migration，修复合并造成的不一致。

## 3. 本地关键能力审查

- [ ] 3.1 审查 scheduler、sticky/fallback、privacy、image capability、runtime setting 热更新和网关透传语义。
- [ ] 3.2 审查大输入请求体保留、磁盘落盘、重放、清理和内存释放语义。
- [ ] 3.3 为审查发现的行为回归先补失败测试，再实施最小修复。

## 4. 完整验证

- [ ] 4.1 在 `backend/` 运行 `go test ./... -count=1`。
- [ ] 4.2 在 `frontend/` 运行 `pnpm test:run`、`pnpm typecheck` 和 `pnpm build`。
- [ ] 4.3 检查工作树、冲突标记和最终 diff，汇总警告、已知限制与剩余风险。

## 5. 收尾

- [ ] 5.1 记录目标 tag、合并提交、冲突决策、修复内容、验证结果和专项审查结论。
- [ ] 5.2 由用户决定 merge 分支的合回、推送或保留方式；本 change 不执行发布或部署。

```

## openspec/changes/merge-upstream-v0-1-151/specs/upstream-release-sync/spec.md

- Source: openspec/changes/merge-upstream-v0-1-151/specs/upstream-release-sync/spec.md
- Lines: 1-12
- SHA256: 02b335b0c8e8b761cf105aef9316f3bfcf25d6758080a0f221e213ea8aa2c6e0

```md
## MODIFIED Requirements

### Requirement: 合并后验证本地关键能力
维护流程 SHALL 在合并后执行自动验证，并专项 review 本地关键能力是否仍然成立。

#### Scenario: 自动验证通过
- **WHEN** 合并完成且无冲突残留
- **THEN** 维护流程运行后端测试、前端单测、前端类型检查和构建验证

#### Scenario: 本地关键能力专项 review
- **WHEN** 自动验证完成
- **THEN** 维护流程复核 scheduler、sticky、privacy、image capability、runtime setting 热更新、网关透传字段和大输入请求体保留、落盘与释放语义等本地关键能力

```
