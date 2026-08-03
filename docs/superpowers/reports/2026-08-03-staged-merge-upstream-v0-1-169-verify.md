# v0.1.169 Staged Merge Verify Report（Comet 阶段 4）

- Change: `staged-merge-upstream-v0-1-169`
- Comet phase: verify（规模评估 full：18 tasks / 1 delta capability / 366 changed files）
- 语言: zh-CN
- 复核时间: 2026-08-03
- 复核对象 HEAD: `2e32649056658c7a1d116ef02824293f90306cba`（main / tag `v0.1.169.1`）
- 基线 source base: `e9a0e4aa53b5d9d5f5c84986cfadd8098dc8e4f3`

## Summary

| 维度 | 状态 |
| --- | --- |
| Completeness | 18/18 tasks；1 个 delta capability（2 requirements / 9 scenarios） |
| Correctness | 9/9 scenarios 覆盖；拓扑、版本、migration identity 全部复核一致 |
| Coherence | 遵循 design 决策；1 处 spec/design doc 漂移已按用户选择追加 Implementation Divergence 记录 |

## 1. Completeness

- tasks.md：18/18 全部 `[x]`，无未完成任务。
- delta spec：`upstream-release-sync` 的 2 个 requirement 均有对应实现证据：
  1. **按正式 release tag 分段集成**：三个 `--no-ff` merge 节点存在且第二父精确匹配固定 peeled SHA（见 §2）。
  2. **合并后验证本地关键能力**：build ledger（`2026-08-02-…-build.md`）记录各段聚焦测试、`make test`、`make build`、两轮 `make -C backend generate`、静态冲突检查；verify 阶段独立复核了拓扑与 migration identity。

## 2. Correctness

### 2.1 Merge 拓扑（对应场景：顺序合入多个 tag）

| Merge commit | 第一父 | 第二父 | 目标 peeled SHA | 匹配 |
| --- | --- | --- | --- | --- |
| `c7ae76df7`（v0.1.166） | `e9d2ce48e` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | `dc893dd0b8…` | ✓ |
| `de4264ba5`（v0.1.168） | `cd78fa1d5` | `99c8e4bf7564823bafbab369acab6539e734c1bb` | `99c8e4bf7…` | ✓ |
| `827369f76`（v0.1.169） | `e6b163fcb` | `26d894ef4f50645a4bf1030e378ac892f17d0223` | `26d894ef4…` | ✓ |

无虚构 `v0.1.167` 阶段；`v0.1.169` 之后无 `upstream/main` 提交合入。

### 2.2 版本与 Migration（对应场景：同号不同名 migration 兼容）

- `backend/cmd/server/VERSION` = `0.1.169.1`（三段闭合后一次更新）。
- 三个文件共存：`191_passkey_credentials.sql`（上游）、`191_subscription_quota_advance_receipts.sql`（本地）、`192_subscription_cache_invalidation_outbox.sql`（本地）。
- 本地 191/192 的 git blob identity 与 source base 完全一致：
  - `191_subscription_quota_advance_receipts.sql` = `c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6`（source base 与 HEAD 均相同）✓
  - `192_subscription_cache_invalidation_outbox.sql` = `502ecec1caf9f76e022c2e83acf3707190539301`（source base 与 HEAD 均相同）✓
  - 与 plan frontmatter 记录的 blob identity 一致，未重命名历史 migration。

### 2.3 发布场景（对应场景：从未合入默认分支的已验证分支发布测试版本）

用户授权发布后实际执行并通过：`main` 快进至验证 HEAD → annotated tag `v0.1.169.1` → Release workflow（run `30776732045`）success → 产出 `sub2api_0.1.169.1_linux_amd64.tar.gz`（sha256 `bb55b5c6…`）与 `checksums.txt` → GHCR `0.1.169.1` 与 `latest` 指向同一 digest（`sha256:7de3aad8…`）→ `sync-version-file` 验证 tag 为 default branch HEAD 祖先。

### 2.4 能力场景覆盖（9/9）

| 场景 | 证据 |
| --- | --- |
| 顺序合入多个 tag | §2.1 拓扑 ✓ |
| 从已验证但未归档的中间 release 继续扩展 | 历史证据保留在 git 历史与 build ledger ✓ |
| 某阶段首次出现本地能力回归 | 各段合并期间无未解决语义冲突；`git show --remerge-diff` 复核无残留 ✓ |
| 分段本机自动验证通过 | build ledger 记录各段门禁 exit 0 ✓ |
| 本机 Docker-backed integration 可用 | Docker 不可用 → 按 spec 记录未验证（见 §4） |
| 用户接受本机 integration 不可用风险 | 报告 §4 记录残余风险，未远程补跑 ✓ |
| 最终自动验证通过 | `make test` 225 files/1698 tests、`make build` 通过 ✓ |
| 本地关键能力专项 review | build verify 报告 14 行矩阵 `protected=11, manual=2, gap=0, unverified=1` ✓ |
| 新增上游能力与本地定制交互专项审查 | GHSA 路径 guard、代理断流 fail-open、Images 统一审计入口等专项复核 ✓ |
| Images 审计入口重复与关闭态大 payload 构造 | 8 项 Images 契约测试全部通过（build verify 报告 §Images Contracts）✓ |
| 同号不同名 migration 兼容 | §2.2 blob identity ✓（PostgreSQL 运行验证受 Docker 限制，见 §4） |
| 从未合入默认分支的已验证分支发布测试版本 | §2.3 实际执行通过 ✓ |

## 3. Coherence

- 实现遵循 openspec design.md 的 10 项决策：单 change 三段、merge/修复分离、阶段 0 固定基线、语义融合默认、先聚焦后 full 门禁、显式本机 integration 配置、migration 原名保留、生成物从源产生、安全修复按入口验收、版本三段闭合后更新。
- delta spec 与 design doc 漂移 1 处：「从未合入默认分支的已验证分支发布测试版本」场景未在 technical design doc 体现。按用户选择（追加漂移记录）已在 `docs/superpowers/specs/2026-08-02-staged-merge-upstream-v0-1-169-design.md` 追加 §12 Implementation Divergence，记录偏差与实际执行结果。该追加为 verify 允许产物，不改实现。

## 4. 残余风险（Docker/Testcontainers 不可用）

本机 `docker` 不可用（build 阶段 preflight `docker_command=unavailable`），以下 PostgreSQL/Testcontainers 契约保持 `unverified`：

- `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate`
- `TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade`
- `TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce`
- `TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails`
- `TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack`
- `TestSubscriptionCacheInvalidationMigration_RawRerunIsIdempotent`
- `TestUserRepoSuite` / `TestAPIKeyRepoSuite`（PostgreSQL lost-update 类别）
- 新库与升级库 PostgreSQL migration 路径

未使用远程服务器补跑，未将未执行 integration 记为通过。

## 5. 工作区与提交边界

- 当前未提交改动仅限 Comet 状态文件（`.comet.yaml`、`run-state.json`、`state-events.jsonl`、`trajectory.jsonl`）与 runtime selection（`.comet/current-change.json`），以及 verify 阶段允许产物（本报告 + design doc §12）。
- 存在无法归因的 untracked 路径 `memory/context/dmit-sub2api-memory-analysis.md`（不属于本 change）；归档提交将使用显式 pathspec，不包含该路径。
- 无实现、tasks、delta spec 或 Design Doc 之外的业务改动。

## 6. 结论

**PASS（带已记录边界）**：Completeness/Correctness/Coherence 全部通过，无 CRITICAL 或 IMPORTANT 问题；残余风险全部来自本机 Docker 不可用并已在 §4 明确记录。可进入 archive 阶段。
