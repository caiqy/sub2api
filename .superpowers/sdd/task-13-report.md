# Task 13 实施报告：v0.1.161 能力矩阵与证据封闭

## 结论与提交边界

- 结果：`DONE_WITH_CONCERNS`。起点为 `0595aa671daa90d46d3e030c84a6b096adc019af`。
- 第一份 docs 提交：`21e233bd422c90153ef7e1e6f9fcfa796547f96b docs: close v0.1.161 stage gate`，仅含 `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`。
- 本报告是第二份独立 docs 提交的唯一文件；未修改源码、测试、生成物、plan/OpenSpec/progress/selectors，未 push、tag、release、deploy 或合并 main。
- TDD N/A：本任务只整理已存在的 docs/evidence，未改行为代码，不伪造 RED；未重跑 Task 12 heavy gates。

## 矩阵与合入证据

- `git diff --name-only 'v0.1.160^{}..v0.1.161^{}'` 的 tag changed-files 为 257 项。与阶段 0 矩阵的实际交集为 22 个精确路径，ledger 已逐项给出测试、调用链或人工证据：`protected=21`、`manual=1`、`gap=0`。
- 交集覆盖模型冷却、advanced/layered scheduler、fallback/WaitPlan、DB recheck、Grok/platform sticky、step-up、spooling、prompt cache、passthrough、images、failed usage、settings、quota、billing probe、用户资源、i18n、Ent/Wire、migration 181-184 与 dependency gate。
- merge 是 `f2158292c7ff3de4caa7ec22f9b7148400948f08`，第二父及 `v0.1.161^{}` 为 `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c`。26 个原始 U 文件及逐项融合结论已写入 ledger；全部为融合结果，非选边。
- Task 11 的六项 finding 已由 `1b80f95c9` 修复，`a534148f3` 记录，`2fce42855` 修正状态；Sol review PASS：`ses_05e81d823ffe1nVy2t46bqPWUk`。

## Task 12 历史与门禁

- 时间线：`0775a6063` 初始兼容；`f1cde1b52` 是无效反向测试，不是 GREEN；`c3bfb765f` 恢复 created-only；`47a6c031e` 对齐 fixture；`07029cc45` 修复 Grok lint；`81aa202ba` 记录 docs；`0595aa671` 协调 checkoff。Sol thorough review PASS：`ses_05e2803fbffeeoKJpSwmoE2aGm`。
- created-only：只有同 ID `response.created` 可绑定空 active turn；外来 delta/terminal 不计 usage、不完成 turn、不释放 permit；正常 fixture 先发送同 ID created。`TestObserveUpstreamMessage_BindsOnlyResponseCreated` 在 `0775a6063` 行为上真实 RED，`c3bfb765f` 后 GREEN；`47a6c031e` 只补 fixture。
- 既有最终门禁：`make test` 201 files/1537 tests、`make build`、双轮 generate/diff、静态/VERSION/timeout/migration 和 remote integration 全部 PASS。remote 两个 migration 目标为 5.20s/4.75s PASS；16 个 skip 是已接受环境基线且不命中受影响能力；日志保留、archive 和远端目录已清理。
- `VERSION=0.1.159.6`。已审批移除的 `openai-first-token-timeout` 未恢复。

## 残余风险与下一阶段

- 风险：Windows Git `sh`/历史 generate 文件锁，以及既有前端 Browserslist、动态 import/chunk-size、router-link/jsdom/i18n advisory；均不改变本次 `gap=0` 结论。
- v0.1.161 已闭合。Task 14/v0.1.162 在独立 reviewer PASS 前保持封闭。
