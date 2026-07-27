# Task 13 实施报告：v0.1.161 能力矩阵与证据封闭

## 结论与提交边界

- 结果：`DONE_WITH_CONCERNS`。起点为 `0595aa671daa90d46d3e030c84a6b096adc019af`。
- 第一份 docs 提交：`21e233bd422c90153ef7e1e6f9fcfa796547f96b docs: close v0.1.161 stage gate`，仅含 `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`。
- 原报告提交：`fe55d3e72 docs: record task 13 implementation report`，仅含本文件。本轮 review-fix 仅暂存本文件与 build ledger；未修改源码、测试、生成物、plan/OpenSpec/progress/selectors，未 push、tag、release、deploy 或合并 main。
- TDD N/A：本任务只整理已存在的 docs/evidence，未改行为代码，不伪造 RED；未重跑 Task 12 heavy gates。

## 矩阵与合入证据

- `git diff --name-only 'v0.1.160^{}..v0.1.161^{}'` 的 tag changed-files 为 257 项。与阶段 0 矩阵的实际交集为 22 个精确路径，ledger 已逐项给出测试、调用链或人工证据：`protected=21`、`manual=1`、`gap=0`。
- 精确交集仅覆盖 ledger 22 路径表所列的 Ent/Wire、用户资源、sticky/session、step-up、spooling、prompt cache、passthrough、images、failed usage、DB fresh recheck、scheduler、settings、quota、billing、migration、i18n 与 dependency gate。模型冷却及 fallback/WaitPlan 是跨路径补充聚焦回归，不属于这 22 条路径或其 `protected=21`、`manual=1`、`gap=0` 计数。
- merge 是 `f2158292c7ff3de4caa7ec22f9b7148400948f08`，第二父及 `v0.1.161^{}` 为 `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c`。26 个原始 U 文件均逐项人工决议：`VERSION` 按中间版本策略只保留本地 `0.1.159.6`；其余项目的融合或采用策略逐条见 ledger 的 `Task 13` 章节。
- Task 11 的六项 finding 已由 `1b80f95c9` 修复，`a534148f3` 记录，`2fce42855` 修正状态；Sol review PASS：`ses_05e81d823ffe1nVy2t46bqPWUk`。

## Task 12 历史与门禁

- 时间线：`0775a6063` 初始兼容；`f1cde1b52` 是无效反向测试，不是 GREEN；`c3bfb765f` 恢复 created-only；`47a6c031e` 对齐 fixture；`07029cc45` 修复 Grok lint；`81aa202ba` 记录 docs；`0595aa671` 协调 checkoff。Sol thorough review PASS：`ses_05e2803fbffeeoKJpSwmoE2aGm`。
- created-only：只有同 ID `response.created` 可绑定空 active turn；外来 delta/terminal 不计 usage、不完成 turn、不释放 permit；正常 fixture 先发送同 ID created。`TestObserveUpstreamMessage_BindsOnlyResponseCreated` 在 `0775a6063` 行为上真实 RED，`c3bfb765f` 后 GREEN；`47a6c031e` 只补 fixture。
- 既有最终门禁：`make test` 201 files/1537 tests、`make build`、双轮 generate/diff、静态/VERSION/timeout/migration 和 remote integration 全部 PASS。remote 两个 migration 目标为 5.20s/4.75s PASS；16 个 skip 是已接受环境基线且不命中受影响能力；日志保留、archive 和远端目录已清理。
- `VERSION=0.1.159.6`。已审批移除的 `openai-first-token-timeout` 未恢复。

## Sol Review-Fix 第 1 轮

- findings：修正 3 个 Important（精确交集与跨路径补充回归混写、26 个冲突笼统声称双边融合、admin 测试包/函数错误）及 1 个 Minor（原始 changed-files 的行号锚点漂移）。精确清单以 `TASK6:v0.1.161:raw:begin/end` marker 为准，行号仅表示当前快照。
- 正确聚焦验证：`go -C backend test ./internal/handler/admin -run '^TestAdminUserList_ParsesAPIKeyGroupID$' -count=1`，exit `0`，`ok github.com/Wei-Shaw/sub2api/internal/handler/admin 2.067s`（PASS）。
- 提交边界：本轮只暂存 `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md` 与本文件；不改测试以迎合证据，不重跑 Task 12 heavy gates。

## 残余风险与下一阶段

- 风险：Windows Git `sh`/历史 generate 文件锁，以及既有前端 Browserslist、动态 import/chunk-size、router-link/jsdom/i18n advisory；均不改变本次 `gap=0` 结论。
- v0.1.161 的实现门禁已闭合，本轮 docs/evidence 修正待 Sol 复审 PASS。Task 14/v0.1.162 在该复审 PASS 前保持封闭。
