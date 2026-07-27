# 子代理进度

- 当前任务：29 项中的第 17 项（OpenSpec 5.1）
- 当前阶段：`done`
- 状态：Task 17 已通过用户授权的预算外 1/1 修复与全新 Sol 最终复审；OpenSpec 5.1 可闭合，随后进入 Task 18
- 简报：`.superpowers/sdd/task-17-brief.md`
- 报告：`.superpowers/sdd/task-17-report.md`
- 审查差异：初始 `.superpowers/sdd/review-b7b7bba69..ed8115134.diff`；Round 1 `.superpowers/sdd/review-ed8115134..f974fb941.diff`；Round 2 `.superpowers/sdd/review-f974fb941..0b265313e.diff`；预算外 `.superpowers/sdd/review-0b265313e..73a8cccf9.diff`
- 基线 SHA：`075abc07399d6154130d2a2695fb24c785acd69c`
- 任务起点 SHA：`b7b7bba69`
- 实现提交：`02abe1574`、`ed8115134`、`c411927ec`、`f974fb941`、`73d25ba10`、`0b265313e`、`0e69a1b2c`、`4ba0c9f23`、`4e1c83ad2`、`73a8cccf9`
- 最后审查 SHA：`73a8cccf9`
- 已完成任务数：17
- 审查模式：`thorough`
- 审查修复轮次：2/2 后用户授权预算外 1/1；全新 Sol reviewer `ses_05b7cc12dffev47d9cJ8EyBhba` 最终 PASS，P0-P3 无 finding
- RED/GREEN：预算外三项 RED 分别证明通知提前返回、moderation 阻塞计费、Ops worker 未在 Ent 前 drain；修复后协调者 fresh 复跑三项精确回归、`-tags=unit` 的 service/handler/cmd-server 三包及 Wire 零 diff均 PASS；unmerged/marker/diff-check/VERSION/protected boundary PASS；race/full gate留 Task 18
- 风险信号：v0.1.163 merge、OpenAI reasoning policy、scheduler quota metadata/LastUsedAt、优雅关停 Cleanup、billing、axios、migration 185
- 未解决反馈：无 Task 17 blocker。余额/账号配额通知已纳入 usage task；Ops worker 在 usage 后、Ent 前 drain；cyber mandatory billing 已前置。残余 full gate/race/integration 属于 Task 18

## 最近完成

- Task 8：完成（`d3e0c596e..3a520c407`）；reviewer B 安全/业务语义 PASS（`ses_05f7550b1ffeC0IO93bKyePzVu`），reviewer A 结构复审 PASS（`ses_05f71bdbeffe1Y0Gn9WnLacxLX`），均为 Sol
- Task 9：完成（`d130c6754..0186949e0`）；第 1 轮修复复审 PASS（`ses_05edb6f30ffeKnMwGN2ZHqlhGg`，Sol）；mutation RED 明确记录为修复后敏感性验证，不冒充实现前 RED
- Task 10：完成（`3a3d8f46e..8c389c7be`）；第 1 轮证据修复复审 PASS（`ses_05ebda040ffenKzRUa4dPBaa4Z`，Sol），`protected=13`、`manual=1`、`gap=0`
- Task 11：完成（`3fc60752a..2fce42855`）；第 1 轮冲突修复复审 PASS（`ses_05e81d823ffe1nVy2t46bqPWUk`，Sol），26 个冲突与 6 个 reviewer finding 均闭合
- Task 12：完成（`6ebe135c1..81aa202ba`）；thorough review PASS（`ses_05e2803fbffeeoKJpSwmoE2aGm`，Sol），created-only ownership 与 v0.1.161 全门禁闭合
- Task 13：完成（`0595aa671..0cb71a654`）；第 1 轮证据修复复审 PASS（`ses_05e0fc9a4ffeY2B2Gm76sn2hZW`，Sol），22 个精确交集 `gap=0`
- Task 14：完成（`940c5cfcf..2d06e4939`）；第 1 轮 cleanup 复审 PASS（`ses_05def9c28ffesAqtmxvSmTT6Rn`，Sol），14 个冲突与 3 个注释化残留闭合
- Task 15：完成（`98fa814d2..521e000e7`）；第 1 轮 transport-error 修复复审 PASS（`ses_05db12056ffeod45HWlwcTm8YS`，Sol），4 项初始回归、WS bridge Ops/摘除/脱敏与更新后 full gate 均闭合
- Task 15 补充：Task 16 review 发现共享 Backup S3 uploader 失效缺口；`8b2c969dc..9ba81a5b1` 第 2 轮修复与 full gate 已由同一 Sol reviewer PASS
- Task 16：完成（`11889c61c..68480bbc8`）；第 2 轮证据修复复审 PASS（`ses_05d65883dffefGri6MnJvSQCOO`，Sol），190 raw/41 selected 口径、四行 `gap=0` 与共享 S3 runtime 修复证据闭合
- Task 17：完成（`b7b7bba69..73a8cccf9`）；2 轮常规修复后用户授权预算外 1/1，三项后台生命周期缺口闭合；全新 Sol reviewer `ses_05b7cc12dffev47d9cJ8EyBhba` PASS

## 约束

- 保留用户所有的未跟踪 `paseo.json`。
- 不得 push、tag、release、deploy 或 merge 到 `main`。
- 远程工作必须使用 `ssh-skill`；不得调用原生 SSH 或 SCP。
- 使用 OpenCode 角色路由：实施者使用 `general`，审查者使用 `reviewer`。
- 将每个完整 Git 修订范围作为一个 PowerShell 参数引用。
- 用户在任务 4 阻塞后批准 Linux 原生等价 integration 门禁：重建 `backend/.test-tmp`、设置 `TMPDIR`/`TMP`/`TEMP`，运行 `CI=true GOFLAGS='-v' go test -tags=integration ./...`；远端不再要求 Make 或 PowerShell。
- canonical OpenSpec、Design Doc 与 plan 已同步。Comet design handoff 是 design 阶段快照；`phase: build` 下官方刷新命令拒绝执行，因此不得手改其生成内容或 hash。
