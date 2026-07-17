# 分段合并上游 v0.1.159 验证记录

## 执行内容
- 固定 v0.1.157、v0.1.158、v0.1.159 的 annotated tag refs 和 peeled commits。
- 在尚未合并 v0.1.157 前运行扩展前完整基线。
- 未合并任何新 tag，未 push、release 或 deploy。

## 扩展起点与固定对象
- 首轮已验证提交：`315617bdec0e21fe8aeb119a986bde960c4864b3`
- 扩展实施分支：`feature/20260717/staged-merge-upstream-v0-1-159`
- 最终边界：`v0.1.159^{}` (`2a75d7d2387587d86ca3c5e5cd8ca96cf3d104c6`)
- 排除范围：`v0.1.159^{}` 之后的 `upstream/main`
- v0.1.157：tag object `a44e63f9fab426ec181bafcf4e4c1a002bbcb8e0`，peeled commit `a2779cd5f30d6d3904a9d59088aed09507678dfe`。
- v0.1.158：tag object `c6ece7d092843c19a2d14d1264669c6416969f6d`，peeled commit `26abd19a2812edba02bbef93c3e2a620141cc257`。
- v0.1.159：tag object `2a2b58263cdf20adf049f3ad8f9e23b4401698c9`，peeled commit `2a75d7d2387587d86ca3c5e5cd8ca96cf3d104c6`。

## 扩展基线
- `make test` 通过：Go 测试和 lint、前端 lint/typecheck、Vitest 181 个测试文件和 1405 个测试全部通过。
- `pnpm --dir frontend run build` 通过：970 个模块完成生产构建。
- 连续两次 `make -C backend generate` 均通过，之后的 Ent/Wire 定向 diff 均为空。
- `git diff --check` 通过。

## 能力矩阵增量
待后续任务在对应 tag 合并后填写。

## v0.1.157
未开始合并。

## v0.1.158
未开始合并。

## v0.1.159
未开始合并。

## 最终验证
- 扩展前完整基线全部退出码为 0。
- 本次提交只包含本报告；首轮 v0.1.156 验证报告保持只读。

## 完整命令与结果摘要
### 工作区、分支和首轮结果
```bash
git branch --show-current
git status --short
git rev-parse HEAD
git merge-base --is-ancestor 12f991dde8a58e183d4bd16a87ef6fd0df714757 HEAD
git merge-base --is-ancestor a2779cd5f30d6d3904a9d59088aed09507678dfe HEAD
```

- 分支为 `feature/20260717/staged-merge-upstream-v0-1-159`。
- HEAD 为 `df422122df428b130cb0f9b114a23bb71b59c9a8`。
- 工作区仅有忽略的 `.comet/current-change.json`。
- `v0.1.156^{}` 祖先检查退出码为 0；`v0.1.157^{}` 祖先检查退出码为非 0。

### 获取和固定 tag
```bash
git fetch upstream --tags --prune
git rev-parse v0.1.157 "v0.1.157^{}"
git rev-parse v0.1.158 "v0.1.158^{}"
git rev-parse v0.1.159 "v0.1.159^{}"
git rev-list --count "v0.1.156^{}..v0.1.157^{}"
git rev-list --count "v0.1.157^{}..v0.1.158^{}"
git rev-list --count "v0.1.158^{}..v0.1.159^{}"
git diff --name-only "v0.1.156^{}..v0.1.157^{}"
git diff --name-only "v0.1.157^{}..v0.1.158^{}"
git diff --name-only "v0.1.158^{}..v0.1.159^{}"
git log --oneline "v0.1.159^{}"..upstream/main
```

- `git fetch upstream --tags --prune` 成功。
- tag object/peeled SHA 与固定对象一致。
- 三个提交区间的提交数依次为 82、20、12；变更文件数依次为 331、105、30。
- `git log --oneline "v0.1.159^{}"..upstream/main` 已记录为排除范围，未纳入本次扩展边界。

### 扩展前完整基线
```bash
make test
pnpm --dir frontend run build
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git diff --check
```

- 初次 `make test` 在 Go 测试通过后受执行器 120 秒时限终止，未产生命令失败输出或退出结果；以 600 秒执行窗口原样重跑后退出码为 0。
- `make test` 的重跑结果：Go 测试和 `golangci-lint` 通过；前端 lint/typecheck 通过；Vitest 为 181 个测试文件、1405 个测试通过。
- `pnpm --dir frontend run build` 退出码为 0，完成 970 个模块构建。
- 两次 `make -C backend generate` 均退出码为 0；两次 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 均退出码为 0，无生成 diff。
- `git diff --check` 退出码为 0。

## 文件
- 创建：`docs/superpowers/reports/2026-07-17-staged-merge-upstream-v0-1-159-validation.md`
- 未修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`
- 未提交：`.comet/current-change.json`（忽略的本地选择文件）。

## 提交
- 提交命令：`git add -f docs/superpowers/reports/2026-07-17-staged-merge-upstream-v0-1-159-validation.md`，随后执行 `git commit -m "docs: record v0.1.159 extension baseline"`。
- 提交范围：仅本报告。

## 自审
- 固定对象、提交计数、变更文件计数和最终边界与任务 brief 一致。
- 未合并 v0.1.157、v0.1.158 或 v0.1.159；未 push、release 或 deploy。
- 首轮 v0.1.156 验证报告未修改；`.comet/current-change.json` 未加入提交。
- 已复核首轮报告无 diff；暂存区和提交后将复核仅包含本报告，工作区仅保留忽略的 `.comet/current-change.json`。

## 残余风险与未执行事项
- 本任务仅固定 refs 和建立扩展前基线；v0.1.157、v0.1.158、v0.1.159 的合并和能力验证均未执行。
- 构建保留现有 Browserslist 数据过期、动态导入与 chunk 大小告警；本次命令均成功，未将其作为本任务范围内的修复项。
