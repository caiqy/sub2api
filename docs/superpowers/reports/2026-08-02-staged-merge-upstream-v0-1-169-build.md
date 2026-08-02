# Stage 0 基线身份账本

- 实施时间：2026-08-02T09:23:14+08:00
- Comet 确认的隔离位置：`D:/Caiqy/Projects/Github/sub2api`
- 绑定分支：`feature/20260802/staged-merge-upstream-v0-1-169`
- immutable source base：`e9a0e4aa53b5d9d5f5c84986cfadd8098dc8e4f3`
- execution base：`10ee678a49c389958315bfdb1466796dc715f2e5`

## Source-to-execution 规划路径

`git merge-base --is-ancestor` 已确认 execution base 是 immutable source base 的后代。tree diff 与 commit path 均仅包含以下 planning-only 路径：

- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet.yaml`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/artifacts.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/checkpoint.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/context.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/handoff/brainstorm-summary.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/handoff/spec-context.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/handoff/spec-context.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/run-state.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/skill-snapshots/9bd4ffab011ae18aef91dc0db336ffc12d454b513229178c15b8b75d50930ba1/package.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/skill-snapshots/9bd4ffab011ae18aef91dc0db336ffc12d454b513229178c15b8b75d50930ba1/sha256`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/state-events.jsonl`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/subagent-progress.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/trajectory.jsonl`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.openspec.yaml`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/design.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/proposal.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/specs/upstream-release-sync/spec.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/tasks.md`
- `docs/superpowers/plans/2026-08-02-staged-merge-upstream-v0-1-169.md`
- `docs/superpowers/specs/2026-08-02-staged-merge-upstream-v0-1-169-design.md`

## 已验证身份

- source `VERSION`：`0.1.165.4`
- execution `VERSION`：`0.1.165.4`
- source receipt migration blob：`c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6`
- source outbox migration blob：`502ecec1caf9f76e022c2e83acf3707190539301`
- runtime selection：存在，唯一允许的运行时未跟踪状态为 `?? .comet/current-change.json`
- 目标 tag：`v0.1.166`、`v0.1.168`、`v0.1.169`
- Windows build shell：`D:/scoop/shims/bash.exe` 存在且可由 `Get-Command` 解析。

## TDD 与边界

- RED：N/A（证据型 docs-only 任务）
- GREEN：N/A（证据型 docs-only 任务）
- 不创建生产代码或行为变更，因此不伪造失败测试。
- 禁止 push、tag、release、deploy、镜像、SSH/服务器/数据库/Redis/Nginx 操作。
