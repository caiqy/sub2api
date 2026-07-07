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
