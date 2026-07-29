# 子代理进度

- 当前任务：29 项中的第 20 项（OpenSpec 6.1）
- 当前阶段：`follow-up-closure-ready`
- 状态：Task 20 已完成关闭写回，Task 21 ready；已完成 20 项，最后审查 SHA `4778e32dc`
- 简报：`.superpowers/sdd/task-20-brief.md`
- 报告：`.superpowers/sdd/task-20-report.md`
- 最终复审：`.superpowers/sdd/task-20-final-review.md`
- 预算外修复简报：`.superpowers/sdd/task-20-review-extra.md`
- 审查差异：初始 `.superpowers/sdd/review-07167bbfa..6ebd068ff.diff`；Round 1 fix `.superpowers/sdd/review-6ebd068ff..96455c43b.diff`；Round 2 final `.superpowers/sdd/review-6ebd068ff..09db65607.diff`；extra final `.superpowers/sdd/review-6ebd068ff..babe29e00.diff`
- 基线 SHA：`075abc07399d6154130d2a2695fb24c785acd69c`
- 任务起点 SHA：`07167bbfa`
- 实现提交：`699459921`、`6ebd068ff`、`48e2d4a0b`、`88aeed4b0`、`96455c43b`、`a9292253f`、`09db65607`、`1e7b8af75`、`babe29e00`
- 最后审查 SHA：`babe29e00`
- 已完成任务数：20
- 审查模式：`thorough`
- 审查修复轮次：常规 2/2 与用户授权 extra 1/1 已作为历史记录保留；关闭写回不追加新修复轮次
- RED/GREEN：follow-up 范围 `babe29e00f18df9a0011d8464446654148d5eb53..4778e32dc879f682fd5774c1fb0c5a63867802c6` 的验证与归档已完成，审查无 Critical/Important
- 风险信号：历史记录保留 `composite routing`、同号 172 migrations、两个 186 migrations、生成物与依赖冲突的原始审查链；当前仅做关闭记账
- 未解决反馈：保留 3 个非阻断 Minor 作为历史事实，不再转回阻断

## Task 20 关闭写回

- 范围：`babe29e00f18df9a0011d8464446654148d5eb53..4778e32dc879f682fd5774c1fb0c5a63867802c6`
- 复审会话：Paseo agent `74b8c51b-f4ae-4a56-b3a2-d8bae2d640e7`，runtime `ses_055058ab6ffeZf8yxzq33JSWqw`，内部 reviewer `ses_0550342a5ffeiJAD61Nle4BPcg`
- 验证报告：`docs/superpowers/reports/2026-07-29-unify-effective-gateway-route-state-verify.md`
- 归档路径：`openspec/changes/archive/2026-07-29-unify-effective-gateway-route-state`
- 归档提交：`a948e3b5e`、`263147b6b`
- 用户基线例外：`make test` 的 lease-loss EOF 为用户明确接受且已在原始上游 `v0.1.165` 复现的基线例外，不记为通过
- 3 个历史 Minor：源码字符串护栏、两个 `count_tokens` stale-subscription 运行时覆盖缺口、Ollama 固定 `50ms` barrier

## Task 20 永久最终阻断

- Extra fixer：唯一 fresh Terra `37a75748-745f-43f6-bc4b-525beef3863b` / `ses_0592794eaffeUDPSRnfuuyDVrb`；提交 `1e7b8af75`（源码/测试）与 `babe29e00`（tracked ledger）。
- Extra reviewer：唯一 fresh Sol `965b8494-82b5-492d-bc7a-50a228b98295` / `ses_058f9bd7bffeTdybiz4KqLVNPF`，只读 Plan Mode；结论 `0 Critical`、`Evidence=FAIL`、`Spec=CHANGES_REQUESTED`、`Overall=TASK20_BLOCKED_FINAL`、`TASK21_KEEP_CLOSED`。
- Important 1 OPEN：effective group 仍未统一 middleware subscription、protocol dispatch、Gin context 中的 API key 与 Ops 状态；handler 内局部 effective route 不能使下游全调用链共享同一权威 group。
- Important 2 OPEN：prompt-too-long secondary fallback 在解析最终 fallback 前校验中间 group；billing 又以 `subscription=nil` 检查最终 subscription group，使其退化为余额模式。
- Important 3 CLOSED：composite explicit alias price 保持优先，非 composite unmapped `channel_mapped` 不再让 original model 抢占 concrete price。
- Important 4 OPEN：普通 Responses WebSocket later frame 已执行 account mapping，但 HTTP bridge later-frame 路径仍只有 route `RewriteRequest`，未闭合 account mapping、provider affinity 与 passthrough 一致性。
- Important 5 CLOSED：runtime resolver/direct detector/exact-prefix/legacy route 的 100 字符边界已在 billing/audit 前生效。
- Important 6 CLOSED：tracked ledger 已自包含原始 10+4、repair chain、65+1 范围、focused evidence、advisory 与 Task 21 boundary。
- Minor OPEN：无 channel mapping 时 public alias 仍可能覆盖 concrete 中间模型，三段审计的 requested/channel-mapped/upstream 语义未完全闭合。
- 保持 CLOSED：Ollama post-commit enrichment、Grok multipart、5 MiB/multipart snapshot、copy-source request ownership、unit-tag group-copy evidence；不得重做。
- 最终额度：常规 Round 2 `2/2` + extra `1/1` 均耗尽；不得再派 fixer/reviewer、不得请求额外预算、不得改源码或继续 Task 21。

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
- Task 18：完成（`2eb7ba771..2918eb63c`）；lint RED 最小修复、本地 full gate、remote integration/migration 与 SKIP=13 证据闭合；fresh Sol reviewer `ses_05b2d6856ffeslt1jo0WjqIoQj` PASS
- Task 19：完成（`a452e3fdd..59e373f07`）；171/171/170 集合口径、六行能力矩阵、clean archive axios 1.18.1/build 与 shutdown 审计边界闭合；fresh Sol reviewer `ses_05b126592ffefdFI5RspQWcfYc` 第 1 轮修复复审 PASS

## 约束

- 保留用户所有的未跟踪 `paseo.json`。
- 不得 push、tag、release、deploy 或 merge 到 `main`。
- 远程工作必须使用 `ssh-skill`；不得调用原生 SSH 或 SCP。
- 使用 OpenCode 角色路由：实施者使用 `general`，审查者使用 `reviewer`。
- 将每个完整 Git 修订范围作为一个 PowerShell 参数引用。
- 用户在任务 4 阻塞后批准 Linux 原生等价 integration 门禁：重建 `backend/.test-tmp`、设置 `TMPDIR`/`TMP`/`TEMP`，运行 `CI=true GOFLAGS='-v' go test -tags=integration ./...`；远端不再要求 Make 或 PowerShell。
- canonical OpenSpec、Design Doc 与 plan 已同步。Comet design handoff 是 design 阶段快照；`phase: build` 下官方刷新命令拒绝执行，因此不得手改其生成内容或 hash。
