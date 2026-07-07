# Brainstorm Summary

- Change: merge-upstream-v0-1-146
- Date: 2026-07-07

## 确认的技术方案

- 采用方案 A：合并 upstream release tag `v0.1.146` 到临时分支。
- build 阶段先 fetch upstream refs/tags，确认 `v0.1.146` 可用。
- 从当前干净 `main` 创建临时分支，不直接改写 `main`。
- 冲突处理优先保留 upstream 修复和本地定制；不可共存语义暂停确认。
- 修复边界限定为 upstream `v0.1.146` 合并引入或暴露的问题；无关旧问题只记录，不并入当前 change。

## 关键取舍与风险

- 使用 release tag 而不是 `upstream/main`：避免范围漂移和未发布分支状态。
- 不逐提交 cherry-pick/replay：避免人工遗漏和过高成本。
- 风险：测试通过仍可能遗漏本地 scheduler、sticky、privacy、image capability、runtime setting 热更新、网关透传字段等语义回归。
- 缓解：保留专项 review；发现不可共存语义冲突时暂停确认。

## 测试策略

- 运行 `go test ./...`。
- 运行前端 typecheck 和 build。
- 专项 review scheduler、sticky、privacy、image capability、runtime setting 热更新、网关透传字段。

## Spec Patch

无。
