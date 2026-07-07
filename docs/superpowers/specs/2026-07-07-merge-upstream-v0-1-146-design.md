---
comet_change: merge-upstream-v0-1-146
role: technical-design
canonical_spec: openspec
---

# Merge Upstream v0.1.146 Design

## 背景

当前本地 `main` 干净，`origin/main` 与 `HEAD` 一致。本地最高定制 tag 为 `v0.1.143.4`，upstream `Wei-Shaw/sub2api` 已发布 `v0.1.146`。仓库惯例要求先确认真实上游目标，在隔离分支合并，完成自动验证后再专项复核本地关键能力。

## 目标

1. 以 upstream release tag `v0.1.146` 作为本次合并目标。
2. 从当前 `main` 创建临时合并分支，不直接改写主线。
3. 冲突处理优先保留 upstream 修复和本地定制；不可共存语义暂停让用户确认。
4. 修复边界限定为 `v0.1.146` 合并引入或暴露的问题；无关旧问题只记录。

## 非目标

1. 不直接合回 `main` 或推送远端。
2. 不新增业务 capability、public API 或数据库 schema。
3. 不借合并机会做无关重构。
4. 不修复与本次 upstream 合并无关的历史问题。

## 方案选择

采用方案 A：合并 upstream release tag `v0.1.146` 到临时分支。

放弃 `upstream/main`，因为它可能包含未发布分支状态，范围不如 release tag 清晰。放弃逐提交 cherry-pick/replay，因为跨 release 合并的人工成本高，也更容易遗漏上下文。

## 执行设计

### 1. 准备与隔离

build 阶段先获取 upstream refs/tags，确认 `v0.1.146` 可用，并再次确认 `main` 干净。随后从 `main` 创建临时分支，例如 `merge/upstream-v0.1.146`。

### 2. 合并策略

在临时分支合并 upstream `v0.1.146`。对 Git 冲突和可疑自动合并结果按三类处理：

1. 可共存：直接融合，保留 upstream 更新和本地定制。
2. 需要小修：只修复合并引入或暴露的编译、测试、类型或语义问题。
3. 不可共存：暂停并给出选项，让用户决定保留策略。

版本文件、生成文件、配置和 migration 类文件需要单独复核，避免看似小的差异影响发布或运行。

### 3. 验证策略

合并收敛后执行：

1. `go test ./...`
2. 前端 typecheck
3. 前端 build
4. 本地关键能力专项 review

专项 review 聚焦 scheduler、sticky、privacy、image capability、runtime setting 热更新和网关透传字段。自动测试通过不等于这些语义仍然成立，review 结论需要在收尾汇总中明确记录。

## 风险与缓解

1. upstream 与本地定制在同一行为路径上冲突。缓解：优先共存，不能共存时暂停确认。
2. 测试通过但本地语义被静默改写。缓解：执行本地关键能力专项 review。
3. 验证暴露无关旧问题。缓解：只记录旧问题，不并入当前 change，除非用户另行授权。

## 成功标准

1. upstream `v0.1.146` 已在临时分支完成合并，且无冲突残留。
2. 合并引入的问题已修复，旧问题没有混入当前范围。
3. 后端测试、前端 typecheck/build 已执行并记录结果。
4. 本地关键能力 review 已完成并记录结论。
5. 是否合回 `main`、是否推送远端由用户在收尾阶段确认。
