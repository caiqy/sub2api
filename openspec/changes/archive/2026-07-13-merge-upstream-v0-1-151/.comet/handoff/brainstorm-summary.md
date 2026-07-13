# Brainstorm Summary

- Change: merge-upstream-v0-1-151
- Date: 2026-07-13

## 已确认事实与约束

- 目标为 upstream 正式 release tag `v0.1.151`，不包含该 tag 之后 `upstream/main` 的 40 个提交。
- 基线为已同步到 `origin/main` 的本地 `main`，包含 `v0.1.146.4` 与大输入内存优化。
- 必须保留 scheduler、sticky/fallback、privacy、image capability、runtime setting 热更新、网关透传和大输入请求体生命周期等本地关键能力。
- 本 change 不负责发布或部署。

## 确认的技术方案

- 从同步后的 `main@46d92f1d7` 创建独立 merge 分支，固定 `v0.1.151^{}` 为 `deff3123...`，执行一次 `--no-ff` merge。
- 上游 merge commit 与后续本地兼容修复分开提交，便于审查和回退。
- 冲突按业务语义融合上游修复与本地定制；不可共存时列出行为差异和影响并暂停决策。
- 合并后按能力矩阵审查调度、网关、安全与能力、运行时热更新和大输入请求体生命周期。

## 关键取舍与风险

- 单次合并的 diff 较大，但比逐 tag 重复解冲突或 cherry-pick 遗漏依赖更可追踪。
- 文本无冲突不代表行为无回归，因此必须执行能力级审查。
- 上游与本地业务规则无法共存时禁止自动猜测，必须由用户决策。
- `VERSION`、依赖、生成代码、migration 和配置可能产生非显式语义冲突，需要单独复核。

## 测试策略

- 合并或修复后先运行受影响区域定向测试，最终运行后端 `go test ./... -count=1`。
- 前端最终运行 `pnpm test:run`、`pnpm typecheck` 和 `pnpm build`。
- 执行 `git diff --check`、冲突标记扫描、工作树与 merge ancestry 检查。
- 能力审查发现回归时先补失败测试，再实施最小修复。
- 验证报告记录冲突决策、能力矩阵、测试输出、警告和残余风险。

## Spec Patch

- 已在 open 阶段 delta spec 中加入前端单测和大输入请求体保留、落盘与释放语义审查要求；无需额外 patch。
