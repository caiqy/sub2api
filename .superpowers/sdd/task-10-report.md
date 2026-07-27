# Task 10 实施报告：v0.1.160 能力矩阵与证据封闭

## 状态

`DONE_WITH_CONCERNS`

TDD N/A：本任务只封闭已有证据，未修改行为代码，未伪造 RED，也未重跑 Task 9 heavy gates。

## 提交

- 原单文件 ledger 提交为 `f5a45b541188f29ef1bc259227ccf4d09170aaf0 docs: close v0.1.160 stage gate`，边界仅为 `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`。
- 本轮 review-fix 是独立的 `docs: fix v0.1.160 stage evidence` 提交，边界仅为该 ledger 与本报告；不 amend `f5a45b541`。

## 证据

- merge：`e04cb1aa2c2554a04bec55f9b4393d3efd2eb693`；第二父：`8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62`。
- tag diff：133 paths；能力交集：12 个精确路径，映射 14 个能力结论，`protected=13`、`manual=1`、`gap=0`。timeout removal 为交集外的已审批人工核验。
- Task 9：`3de7191e3` 新增 staged migration upgrade 覆盖；`a719a8e6c` 恢复 moderation 委派并删除死 WS closure；`31b132689` 补 PromptAdminService Wire 绑定；`516e998b` 记录 full gate；`fd909c5be` 加强 migration 快照和测试 fixture；`0186949e0` 记录 review-fix 证据。本地 full gate、两轮 generate、静态 gate、远程 integration 均为 GREEN。
- migration：远程 staged-upgrade PASS 且无 SKIP；mutation RED 发生于修复后，只证明断言敏感性。
- worker panic：测试 fixture 的 nil 嵌入 mock 根因已修；本机和 archive `-count=3` 均无 panic。
- 远程日志：`C:/Users/caiqy/AppData/Local/Temp/sub2api-task-9-d70a45257d0f46c0911f6e17747ab6aa-integration.log`；remote 目录和 local tar 均已清理。
- VERSION：`backend/cmd/server/VERSION` 为 `0.1.159.6`。

## 风险与顾虑

- 环境型 skip 已分类，目标 migration 不在 skip 中。
- 既有 Browserslist、Vite chunk、Ryuk advisory 仍在；Windows `user-mapped section` 历史根因未定。
- Task 11：证据已补齐，等待独立复审。

## Review Fix 第 1 轮

- Finding 1：统一 ledger 顶部、中部和 Task 10 的 `v0.1.160` 状态；Task 11 统一为“证据已补齐，等待独立复审”。
- Finding 2：14 项矩阵将 scheduler 与 fallback/WaitPlan 拆为两个 `protected` 行；image moderation/security audit 改为补充证据、不计入统计，表格自然得到 `protected=13`、`manual=1`、`gap=0`。
- Finding 3：补齐 `0186949e0`，逐项声明六个 Task 9 提交的角色与边界；mutation 明确为删除上游 `172_composite_model_routes.sql`，不混同本地 `172_video_per_second_billing_metadata.sql`。
- Finding 4：真实执行 `go -C backend mod verify`，exit `0`，结果 `all modules verified`；真实执行 `pnpm --dir frontend install --frozen-lockfile`，exit `0`，结果 `Lockfile is up to date, resolution step is skipped`、`Already up to date`。随后 `git diff --exit-code -- backend/go.mod backend/go.sum frontend/package.json frontend/pnpm-lock.yaml` exit `0`；依赖命令未改动 lockfile 或其他 tracked 文件。
- 本轮未重跑 full gate；未改源码、依赖或 lockfile，未 push、tag、release 或 deploy。
- 剩余顾虑：既有环境型 skip、Browserslist/Vite/Ryuk advisory 及 Windows `user-mapped section` 历史根因仍未消除；独立复审尚未完成。
