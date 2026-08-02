---
change: staged-merge-upstream-v0-1-169
design-doc: docs/superpowers/specs/2026-08-02-staged-merge-upstream-v0-1-169-design.md
base-ref: e9a0e4aa53b5d9d5f5c84986cfadd8098dc8e4f3
---

# 分段合并上游 v0.1.169 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` to implement this plan task-by-task. Only the 18 top-level `Task N` lines are Plan checkboxes; controller owns their status.

**Goal:** 在 Comet 已确认的干净隔离位置将上游 `v0.1.166`、`v0.1.168`、`v0.1.169` 作为三个可审计的 merge 节点按顺序合入，保护本地能力，并将版本一次更新为 `0.1.169.1`。

**Architecture:** 每个正式 release tag 都以 `git merge --no-ff --no-commit v0.1.166`、`git merge --no-ff --no-commit v0.1.168` 或 `git merge --no-ff --no-commit v0.1.169` 开始，先在未提交 merge 状态完成阻塞式冲突、接口和高风险入口审查，再创建只含上游树与必要冲突融合的 merge commit。merge 后以 changed-files 对照能力矩阵执行行为审查、聚焦测试和完整本机门禁；若发现可复现回归，保留 RED、最小修复并创建独立兼容提交。所有阶段证据写入 build ledger，最终由 verify report 汇总。

**Tech Stack:** Git worktree/merge、Go、Ent、Wire、PostgreSQL/Testcontainers、pnpm/Vitest、PowerShell、OpenSpec、Comet。

**Baselines:** frontmatter `base-ref` `e9a0e4aa53b5d9d5f5c84986cfadd8098dc8e4f3` 是不可变的 source base，用于上游范围、migration blob identity 与最终 merge-range 审计。Comet setup 后由 Task 1 捕获的 `$executionBase` 可以是 source base 的后代；它仅确定本实施链的起点，不替代 source base。

## 全局约束

- 本计划完成后，由 Comet 联合决策创建并绑定实际隔离位置和分支；计划不预选、不创建 worktree 或分支。Task 1 必须验证 source base 是该已确认位置 HEAD 的祖先，并捕获 `$executionBase`；从 source base 到 execution base 仅允许当前 change 的 OpenSpec 目录前缀、Design Doc 与本计划的 planning 差异。Comet 编排层保证这些文档在该位置可达。
- Task 1 必须以 `git status --short --untracked-files=all` 捕获实际状态。实施位置唯一允许的非业务状态是实际出现的 `?? .comet/current-change.json` Comet runtime selection；任何其他 tracked、untracked 或 staged 用户路径一律 BLOCK，必须由 Comet 切换或重建干净隔离位置后从 Task 1 重新入口。runtime selection 不得进入任何提交。
- 只合并 `v0.1.166`、`v0.1.168`、`v0.1.169`，其 peeled SHA 依次必须是 `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8`、`99c8e4bf7564823bafbab369acab6539e734c1bb`、`26d894ef4f50645a4bf1030e378ac892f17d0223`；不得创建 `v0.1.167` 阶段，不得合入 `v0.1.169` 之后的 `upstream/main` 提交。
- 三段 merge 均从 `$executionBase` 之后的干净阶段 HEAD 依次执行 `git merge --no-ff --no-commit v0.1.166`、`git merge --no-ff --no-commit v0.1.168`、`git merge --no-ff --no-commit v0.1.169`。merge commit 只含上游树和完成 merge 所需的融合，第二父必须是目标 tag peeled SHA；回归修复必须使用独立兼容提交。Task 15/17 的 merge log 与 migration audit 仍从 immutable source base 开始。
- `backend/cmd/server/VERSION` 在三个中间阶段保持 `0.1.165.4`，三段均封闭后才一次改为 `0.1.169.1`；不产生 `0.1.166.1` 或 `0.1.168.1`。
- 所有暂存使用显式路径，禁止 `git add .`。schema/provider/manifest 源与其生成输出必须同一提交；不得手拼 `go.sum`、Ent/Wire 输出或前端 lockfile。
- 每段先跑命中的聚焦测试，再执行 `make test`、对应阶段的已验证 Windows `VERSION`/`SHELL` build 命令、两轮 `make -C backend generate`、生成 diff 检查、`git diff --check`、unmerged index 检查和 tracked conflict marker 扫描。
- 阶段 0 先用 `Test-Path 'D:/scoop/shims/bash.exe'` 和 `Get-Command 'D:/scoop/shims/bash.exe'` 验证已验证 bash；缺失是环境阻塞，不通过修改产品代码规避。阶段 0、v0.1.166、v0.1.168、v0.1.169 使用 `make "VERSION=0.1.165.4" "SHELL=D:/scoop/shims/bash.exe" build`；最终阶段使用 `make "VERSION=0.1.169.1" "SHELL=D:/scoop/shims/bash.exe" build`。`make test` 保持仓库默认命令。
- 每段都运行轻量 Docker 检查。preflight 失败可直接将受影响 integration 记为 `unverified`。preflight 成功后，必须保存完整 `go test -v` 日志和退出码；非零退出先按 systematic debugging 判定根因，断言/代码失败阻塞当前阶段，只有日志证明 Docker/Testcontainers 环境不可用时才可记为 `unverified`。PASS 必须由 `$pattern = '^--- PASS: ' + [regex]::Escape($target) + ' \('` 逐顶级测试匹配；exit 0、包级 `ok`、`no tests to run` 和 `--- SKIP:` 都不构成通过证据，也不连接远程服务器补跑。
- 迁移必须原名保留 `backend/migrations/191_passkey_credentials.sql`、`backend/migrations/191_subscription_quota_advance_receipts.sql`、`backend/migrations/192_subscription_cache_invalidation_outbox.sql`；按完整 filename 排序、记录和校验 checksum，禁止重命名历史 migration。immutable source base 的历史 blob identity 固定为 `191_subscription_quota_advance_receipts.sql=c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6` 和 `192_subscription_cache_invalidation_outbox.sql=502ecec1caf9f76e022c2e83acf3707190539301`，每个相关阶段和最终阶段都必须精确核对。
- Images 契约固定为：只经过统一 security-audit 入口；legacy moderation 每请求至多一次；线程安全 payload provider 至多冻结一次；审计关闭态不构造大 payload；legacy moderation 只有完成运行态和范围判定后才求值；文本在 `ReleaseText` 前保持可读。
- 本 change 不推送、不打 tag、不触发 GitHub Actions、不发版、不构建或发布镜像、不部署、不操作服务器、数据库、Redis 或 Nginx。生产 Nginx 临时盾保持不变。
- 每个阶段的 ledger 证据提交后，index 必须为空；`git status --short --untracked-files=all` 过滤实际存在的 `?? .comet/current-change.json` 后必须为空；有 `gap` 的能力矩阵不得进入下一阶段。
- 每次 merge 前都执行同一 clean gate：`git diff --cached --name-only` 必须无输出；`git status --short --untracked-files=all` 仅可输出实际存在的 `?? .comet/current-change.json`。每次 merge commit 前必须无 unmerged index、`git diff --cached --check` 通过，且 staged paths 不含 runtime selection、build/verify ledger、计划、Design Doc、OpenSpec proposal/design/delta spec/tasks 等规划产物。兼容/证据提交前都列出 staged paths，并验证它们是该提交命令中明确列出的 allowlist 子集。
- 所有 `git add backend/ent` 以 Task 1 和本阶段 merge 前 clean gate 均通过为前提：若任一 gate 发现 `backend/ent/` 有用户改动，立即 BLOCK 并由 Comet 重建干净隔离位置；只有该目录由当前 merge 或当前兼容修复的生成步骤产生变化时才可递归暂存。
- Comet SDD 分工固定：implementer 和 reviewer 均不得修改本计划。唯一合法的 Plan checkbox 修改是 controller 在 reviewer 通过后将当前顶层 Task 从 `[ ]` 改为 `[x]`，再同步唯一映射的 OpenSpec `tasks.md` 项。implementer 完成该 Task 的代码/ledger commit(s) 后，worktree/index 必须通过本计划的 clean 规则，随后必须由 thorough reviewer 审查。controller 创建的 checkoff commit 严格只暂存本计划、`tasks.md` 和 progress 三个路径；checkoff commit 不是 merge 或兼容提交，三个 upstream merge 前仍须重新通过 clean gate。
- 通用 progress lifecycle：若平台可在 agent 完成前向 controller 返回 live agent/task ID，派发成功后立即将 agent identity、role、attempt 与显式 model 写入 progress，并创建严格 progress-only commit。`subagent-progress.md` 已由 planning 提交纳入 tracked；仓库 docs ignore 规则仍要求显式 `-f` 暂存，安全性由每次 exact staged allowlist 保障。
- 本平台 `functions.task` 是对 controller 的原子工具调用：工具返回前 controller 没有可执行的中间回合，returned `task_id` 与完成结果一起返回。每次 implementer、reviewer 或 fix 派发前，controller 必须先以 progress-only commit 写入唯一 dispatch token、role、attempt、显式 model、brief/report 路径与 base HEAD；不得把“待派发 implementer”宣称为已有 agent identity。工具返回后的第一个动作必须写入 returned `task_id`、`DONE`/`DONE_WITH_CONCERNS`/`BLOCKED`/`NEEDS_CONTEXT`、commit/test/report evidence，并创建 progress-only commit；该结果 checkpoint 可以与下一角色“即将派发”状态合并。
- 恢复规则：若 progress 只有 dispatch intent 而没有 returned `task_id` 或结果，先检查指定 report、Git 提交和宿主工具结果/通知；确认已有结果才恢复对应流程，不能确认时标记 `BLOCKED`，禁止盲目重复派发。已有 `task_id` 且需要 fix round 时按平台支持方式 resume；Comet thorough review 的 round 计数不重置。任何 implementer/reviewer 启动前及每次 merge clean gate 前，最新 progress checkpoint 必须已提交，且 worktree 过滤实际 `?? .comet/current-change.json` 后为空。
- Controller 每次 progress checkpoint 使用以下严格 allowlist：

```powershell
$progressPath = 'docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/subagent-progress.md'
git add -f -- $progressPath
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @($progressPath) -DifferenceObject $staged) { throw 'controller progress allowlist mismatch' }
git.exe commit -m "docs: record SDD progress"
```
- Controller 每次 checkoff 都使用以下严格 allowlist；三条路径均应因本次 reviewer verdict、顶层 Task/OpenSpec 勾选和 progress 记录而改变：

```powershell
$checkoffAllow = @('docs/superpowers/plans/2026-08-02-staged-merge-upstream-v0-1-169.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/tasks.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/subagent-progress.md')
git add -f -- $checkoffAllow
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject $checkoffAllow -DifferenceObject $staged) { throw 'controller checkoff allowlist mismatch' }
git.exe commit -m "docs: check off reviewed task"
```

## 文件结构与职责

| 路径 | 实施时的责任 |
| --- | --- |
| `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md` | 创建；同时记录 immutable source base、`$executionBase`、tag manifest、每段 changed-files、冲突台账、能力矩阵、命令退出码、Docker 判定、失败/修复和阶段结论。 |
| `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-verify.md` | 创建；记录最终门禁、tag 拓扑、migration 结果、能力专项 review、残余风险和非目标确认。 |
| `docs/superpowers/plans/2026-08-02-staged-merge-upstream-v0-1-169.md` | 仅由 controller 在 reviewer 通过后勾选对应 Task 顶层 checkbox，随 docs-only checkoff commit 提交。 |
| `docs/openspec/changes/staged-merge-upstream-v0-1-169/tasks.md` | 仅由 controller 在 reviewer 通过后勾选唯一映射 OpenSpec 项，随 docs-only checkoff commit 提交。 |
| `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/subagent-progress.md` | planning 已纳入 tracked；仅由 controller 记录可恢复的 dispatch intent/result。live-ID 平台在成功派发后立即记录 identity；本平台在工具返回后立即记录 returned `task_id` 和结果。每次 checkpoint 以 progress-only commit 提交；review 通过后的 checkoff 再随三路径 docs-only commit 记录。 |
| `backend/cmd/server/VERSION` | 仅在 Task 15 修改为 `0.1.169.1`。 |
| `backend/internal/repository/migrations_subscription_quota_passkey_upgrade_integration_test.go` | 创建；验证先应用本地 `0.1.165.4` migration 集合再应用完整 `v0.1.169` 集合时，双方 `191_*` 和本地 `192_*` 的排序、执行、幂等和 checksum 均稳定。 |
| `backend/internal/handler/openai_images.go`、`backend/internal/handler/security_audit_helper.go`、`backend/internal/handler/openai_images_controls_test.go` | 阶段 0 增加 Images 直接保护测试；仅在该测试或最终专项审查出现 RED 时修改生产代码，保持统一审计、单次 moderation/freeze、延迟求值和文本生命周期契约。 |
| `backend/internal/service/upstream_path_guard.go`、`backend/internal/service/upstream_path_guard_test.go`、`backend/internal/service/gemini_upstream_url.go`、`backend/internal/service/gemini_upstream_url_test.go`、`backend/internal/server/routes/gateway_test.go` | `v0.1.169` 合并后审查和补测 GHSA 闭集路径护栏与 Gemini URL 构造入口。 |
| `backend/migrations/191_passkey_credentials.sql`、`backend/migrations/191_subscription_quota_advance_receipts.sql`、`backend/migrations/192_subscription_cache_invalidation_outbox.sql` | 合并后保留原始 filename 和内容；本地 191/192 由 immutable source base blob identity 和 migration runner/integration 双重验证。 |

## 关键接口和验证约定

| 接口/边界 | 约定 |
| --- | --- |
| `applyMigrationsFS(ctx context.Context, db *sql.DB, fsys fs.FS) error` | migration runner 以完整 filename 排序；测试先传入排除 `191_passkey_credentials.sql` 的基线 `fs.FS`，再传入完整 embedded migration `fs.FS`。测试注释必须说明该 filtered FS 只模拟升级路径，不能替代 Git blob identity 历史保护。 |
| `calculateQuotaCycleAdvance(sub *UserSubscription, selection QuotaWindowSelection, now time.Time) (*QuotaCycleAdvanceResult, error)` | 只允许一个已耗尽窗口；结果不能修改数据库快照；后续 `SubscriptionService.AdvanceQuotaCycle` 必须保持 receipt、事务锁、版本化 tombstone/outbox 和 post-commit invalidation。 |
| `sanitizedUpstreamPathSuffix(raw string) (string, bool)` | 空 suffix 合法；非空 suffix 单段最多 128 bytes、最多 8 段、每段仅 `[A-Za-z0-9_.-]` 且不得纯点；空 segment、编码后 traversal、反斜杠、控制字符、非 ASCII 与其他标点拒绝。 |
| `validateUpstreamPathSegment(kind, segment string) error` 和 `buildGeminiAIStudioModelActionURL(baseURL, model, action string, stream bool) (string, error)` | Gemini native/compat 在任何上游 URL 拼接和请求发出前拒绝非法模型；合法 action 只允许 `generateContent`、`streamGenerateContent`、`countTokens`。 |
| Docker integration 证据 | 从 `backend` 执行各任务给出的 `go test -tags=integration -v -count=1 -run` 命令；保存输出到临时文件并逐个匹配该任务列出的 `--- PASS:` 行。 |
| 本地 migration blob identity | immutable source base 与最终 `HEAD` 的 `backend/migrations/191_subscription_quota_advance_receipts.sql` 均必须为 `c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6`；immutable source base 与最终 `HEAD` 的 `backend/migrations/192_subscription_cache_invalidation_outbox.sql` 均必须为 `502ecec1caf9f76e022c2e83acf3707190539301`。 |

## Canonical 能力矩阵

Task 3 建立下表的每一行，并在每段以 `protected`、`manual`、`unverified` 或 `gap` 更新状态和证据。`unverified` 只能用于 Docker/Testcontainers 不可用导致的 integration 契约；任何 `gap` 阻塞下一阶段。

| 能力契约 | 入口/调用链和关键文件 | 基线或阶段聚焦证据 |
| --- | --- | --- |
| advanced/layered scheduler、DB recheck、WaitPlan fallback | `backend/internal/service/openai_account_scheduler_layered_test.go`、`backend/internal/handler/gateway_handler_sticky_fallback_test.go` | `TestLayered_GroupedAccountPassesDBFreshRecheck`、`TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount`、`TestLayered_FallbackWaitPlanRechecksPrivacyRequirementAgainstDB`、`TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot` |
| Grok/platform/session/previous-response sticky、privacy、image capability | 同一 scheduler 文件和 OpenAI WS 入口 | `TestLayered_SessionStickyPreservesGrokBinding`、`TestLayered_PreviousResponseStickyEnabled`、`TestLayered_SessionStickyRecheckHonorsImageCapability`、`TestLayered_PreviousResponseStickyHonorsRequirePrivacySet` |
| OpenAI HTTP/WS、Live、turn ownership、最终 outbound model、failed usage、prompt-cache reuse、透传字段 | `backend/internal/handler/openai_gateway_handler.go`、`backend/internal/service/openai_ws_forwarder.go`、`backend/internal/service/openai_ws_v2/passthrough_relay.go` | `TestOpenAIGatewayService_Forward_WSv2_ResponseDoneUsageParsed`、`TestRelay_OnTurnComplete_UsesCurrentResponseCreateModel`、`TestRelay_OnTurnComplete_PerTerminalEvent`、`TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot` |
| prompt/security audit | `backend/internal/handler/security_audit_helper.go`、`backend/internal/server/routes/prompt_audit_route_coverage_test.go` | `TestEveryGatewayPOSTRouteIsClassifiedForPromptAuditCoverage`、`TestResponsesWebSocketHasFirstAndSubsequentTurnPromptGates` |
| Images 精确审计与文本生命周期 | `backend/internal/handler/openai_images.go`、`backend/internal/handler/openai_images_controls_test.go` | `TestOpenAIImages_UnifiedAuditRunsLegacyOnce`、`TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease`、`TestOpenAIImages_SecurityAuditUsesFrozenPayloadBeforeRelease`、`TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream`、`TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream`、`TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload`、`TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope`、`TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce` |
| request-body replay/spooling/cleanup | `backend/internal/handler/openai_images_controls_test.go`、`backend/internal/service/openai_gateway_request_body.go` | `TestOpenAIImages_InlineSpoolKeepsRawBodyAndOmitsSnapshots`、`TestOpenAIGatewayHandlerImages_MultipartReplayUsesMappedEffectiveBody` |
| 异步图片任务、对象存储、图片输入计费、上游计费倍率 | `backend/internal/service/image_task.go`、`backend/internal/service/image_storage.go`、`backend/internal/service/gateway_record_usage_test.go` | `TestAsyncImagePromptGuardRunsBeforeTaskCreation`、`TestAsyncImageSuccessfulPrecheckIsNotRepeatedByDetachedExecution`、`TestGatewayServiceRecordUsage_EmptyImageSizeDefaultsBeforeBillingAndPersistence`、`TestGatewayServiceRecordUsage_PeakRateAffectsTokenModeImageOutputTokens` |
| settings 热更新和部分更新 | `backend/internal/handler/admin/setting_handler_update.go`、`backend/internal/service/setting_update.go` | `TestUpdateSettingsPartialPayloadKeepsUnsentKeys`、`TestUpdateSettingsFullPayloadStillClearsSentEmptyFields` |
| repository scoped updates、用户/API Key 更新、会话绑定与 step-up | `backend/internal/repository/user_repo.go`、`backend/internal/repository/api_key_repo.go`、Passkey/auth 路由 | `TestUserRepoSuite`、`TestUserRepoAPIKeyGroupFilterSuite`、Passkey handler/service 测试和 auth session revocation 测试 |
| subscription quota cycle reset | `backend/internal/service/subscription_service.go`、receipt/outbox repositories、前端 `frontend/src/utils/subscriptionQuota.ts` | `TestCalculateQuotaCycleAdvance_ResetsOnlySingleExhaustedWindow`、`TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce`、`TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails`、`TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack` |
| 用户资源控制、分组复制、批量限额 | user/group handler、repository、frontend group/account 页面 | 当前相关 handler/repository/Vitest 目标与 changed-files 审查；记录每条入口、测试和人工结论 |
| 前端本地能力 | 菜单、设置、用量、订阅、渠道展示、移动端以及上游触及页面 | `frontend/src/views/admin/__tests__/SettingsView.spec.ts`、`frontend/src/views/admin/__tests__/UsageView.spec.ts`、`frontend/src/utils/__tests__/subscriptionQuota.spec.ts`、`frontend/src/components/channels/__tests__/AvailableChannelsTable.spec.ts` |
| pricing、count_tokens、release fallback、部署安全 | pricing service/resources、gateway route、release/compose/Caddy 文件 | pricing/count_tokens/route 聚焦测试、资源 diff 审查；两个 `deploy/tests/*.sh` 仅在 v0.1.169 merge 后存在时由已验证 bash 执行 |
| Ent/Wire、Go/pnpm 依赖、migrations | `backend/ent/schema/`、`backend/internal/**/wire.go`、`backend/cmd/server/wire_gen.go`、`backend/go.mod`、`frontend/pnpm-lock.yaml`、`backend/migrations/` | 两轮 generate 无 diff、依赖工具生成结果、migration runner unit/integration、前后端 full gate |

### Task 1: 验证 Comet 已绑定的 source/execution 双基线隔离位置

- [x] Task 1: 验证 Comet 已绑定的 source/execution 双基线隔离位置

**映射 OpenSpec:** 1.1

**Files:**
- Create: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`
- Modify: 无应用源码修改

**Interfaces:**
- Consumes: Comet 已确认的实际隔离位置、绑定分支、immutable source base `e9a0e4aa53b5d9d5f5c84986cfadd8098dc8e4f3` 与当前 `VERSION=0.1.165.4`。
- Produces: ledger 的实际隔离位置/绑定分支、immutable source base、`$executionBase`、source-to-execution planning-only tree diff/commit path 清单，以及 runtime selection 存在或缺失的状态记录。

**Step 1: 验证已确认的 source/execution 双基线和绑定分支**

Run:
```powershell
git rev-parse --show-toplevel
git branch --show-current
$sourceBase = 'e9a0e4aa53b5d9d5f5c84986cfadd8098dc8e4f3'
git merge-base --is-ancestor $sourceBase HEAD
if ($LASTEXITCODE -ne 0) { throw "execution HEAD does not descend from immutable source base: $sourceBase" }
$executionBase = (git rev-parse HEAD).Trim()
$sourceVersion = (git show "${sourceBase}:backend/cmd/server/VERSION").Trim()
$executionVersion = (git show "${executionBase}:backend/cmd/server/VERSION").Trim()
if ($sourceVersion -ne '0.1.165.4' -or $executionVersion -ne '0.1.165.4') { throw "source/execution VERSION must both be 0.1.165.4: source=$sourceVersion execution=$executionVersion" }
$sourceReceiptBlob = (git rev-parse "${sourceBase}:backend/migrations/191_subscription_quota_advance_receipts.sql").Trim()
$sourceOutboxBlob = (git rev-parse "${sourceBase}:backend/migrations/192_subscription_cache_invalidation_outbox.sql").Trim()
if ($sourceReceiptBlob -ne 'c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6' -or $sourceOutboxBlob -ne '502ecec1caf9f76e022c2e83acf3707190539301') { throw "immutable source base migration blob identity mismatch" }
$baselineDiff = @(git diff --name-only "${sourceBase}..${executionBase}")
$unexpectedBaselineDiff = $baselineDiff | Where-Object { $_ -notlike 'docs/openspec/changes/staged-merge-upstream-v0-1-169/*' -and $_ -notin @('docs/superpowers/specs/2026-08-02-staged-merge-upstream-v0-1-169-design.md', 'docs/superpowers/plans/2026-08-02-staged-merge-upstream-v0-1-169.md') }
if ($unexpectedBaselineDiff) { throw "execution base contains non-planning changes: $($unexpectedBaselineDiff -join ', ')" }
$baselineCommitPaths = @(git log -m --format= --name-only "${sourceBase}..${executionBase}" | Where-Object { $_.Trim() } | Sort-Object -Unique)
$unexpectedBaselineCommitPaths = $baselineCommitPaths | Where-Object { $_ -notlike 'docs/openspec/changes/staged-merge-upstream-v0-1-169/*' -and $_ -notin @('docs/superpowers/specs/2026-08-02-staged-merge-upstream-v0-1-169-design.md', 'docs/superpowers/plans/2026-08-02-staged-merge-upstream-v0-1-169.md') }
if ($unexpectedBaselineCommitPaths) { throw "source-to-execution history touched non-planning paths: $($unexpectedBaselineCommitPaths -join ', ')" }
$executionBase
$baselineDiff
$baselineCommitPaths
```

Expected: 位置和分支与 Comet 联合决策/绑定记录一致；`git merge-base --is-ancestor` 成功，`$executionBase` 是 immutable source base 的后代，而不是被要求等于 source base；source 与 execution 的 VERSION 均为 `0.1.165.4`，source base 的两个 migration blob OID 精确匹配固定值。`$baselineDiff` 与 `$baselineCommitPaths` 都只能包含 `docs/openspec/changes/staged-merge-upstream-v0-1-169/` 前缀下的当前 change 路径，或两个精确的 Design Doc/Plan 路径；任何提交曾触及 `backend/`、`frontend/`、`deploy/`、`.github/`、`Makefile` 或其他路径，即使随后 revert 至净零差异，也必须阻塞。基线或绑定不一致时停止，不创建 branch/worktree，也不开始 merge。

**Step 2: 验证唯一允许的 runtime selection 状态**

Run:
```powershell
$selectionStatus = '?? .comet/current-change.json'
$index = @(git diff --cached --name-only)
if ($index) { throw "staged paths block isolation: $($index -join ', ')" }
$status = @(git status --short --untracked-files=all)
$unexpected = $status | Where-Object { $_ -ne $selectionStatus }
if ($unexpected) { throw "dirty isolation must be rebuilt by Comet: $($unexpected -join '; ')" }
$status
```

Expected: `$status` 只能为空或精确为 `?? .comet/current-change.json`；任何其它用户/运行时路径或任何 `$index` 路径都 BLOCK，并由 Comet 切换/重建干净隔离位置后重新 Task 1。ledger 仅记录 runtime selection 是否存在，不建立任意用户 dirty path 排除清单。

**Step 3: 验证 Windows build shell**

Run:
```powershell
$bash = 'D:/scoop/shims/bash.exe'
if (-not (Test-Path -LiteralPath $bash)) { throw "required bash is missing: $bash" }
Get-Command $bash -ErrorAction Stop
```

Expected: 两个检查均成功；缺失 bash 记为环境阻塞，不修改产品代码、Makefile 或测试来规避。

**Step 4: 初始化并严格提交 ledger 的基线段**

在 ledger 写入实施时间、Comet 确认的隔离位置、绑定分支、immutable source base、`$executionBase`、source-to-execution planning-only diff、source/execution 的 `VERSION=0.1.165.4`、runtime selection 存在或缺失、两个 immutable source base migration blob OID、三个目标 tag 和“禁止推送/发布/部署/服务器/Nginx 操作”的边界，然后运行：

```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'stage 0 baseline identity allowlist mismatch' }
git.exe commit -m "docs: record stage 0 baseline identity"
```

Expected: implementer 提交严格只含 build ledger，不勾选计划或 OpenSpec。thorough reviewer 通过后，由 controller 创建独立 checkoff commit，勾选顶层 Task 1 和 OpenSpec 1.1 并更新 progress。

### Task 2: 重新获取 refs 并固定 tag 范围

- [x] Task 2: 重新获取 refs 并固定 tag 范围

**映射 OpenSpec:** 1.2

**Files:**
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: 三个正式 tag 和 `upstream/main`。
- Produces: 被核验的 peeled SHA、严格祖先链和明确排除的 tag 后提交清单。

**Step 1: 更新本地 upstream refs**

Run:
```powershell
git fetch upstream --tags --prune
git rev-parse 'v0.1.166^{}'
git rev-parse 'v0.1.168^{}'
git rev-parse 'v0.1.169^{}'
```

Expected: 输出依次精确为 `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8`、`99c8e4bf7564823bafbab369acab6539e734c1bb`、`26d894ef4f50645a4bf1030e378ac892f17d0223`。

**Step 2: 验证祖先关系与 release 上界**

Run:
```powershell
git merge-base --is-ancestor v0.1.166 v0.1.168
git merge-base --is-ancestor v0.1.168 v0.1.169
git for-each-ref refs/tags --merged=upstream/main --format='%(refname:short)' --sort=-v:refname
git log --oneline v0.1.169..upstream/main
```

Expected: 两个 `merge-base` 命令退出 0；最新正式 `v0.1.*` tag 是 `v0.1.169`；最后一个命令的所有提交被记录为范围外而不合并。若最新正式 tag 更高，停止在此任务，更新 OpenSpec 范围后才继续。

**Step 3: 将 tag manifest 与排除提交写入 ledger**

记录每段 tag、peeled SHA、预期 commit/file 数 `62/142`、`36/170`、`38/72`，以及 `v0.1.169..upstream/main` 的完整 oneline 输出。

**Step 4: 将 refs 证据严格提交为 ledger evidence**

Run:
```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'upstream tag manifest allowlist mismatch' }
git.exe commit -m "docs: record upstream tag manifest"
```

Expected: implementer 提交严格只含 build ledger，不创建或推送 tag，也不勾选计划或 OpenSpec。reviewer 通过后 controller 单独 check off Task 2/OpenSpec 1.2。

### Task 3: 建立能力矩阵与冲突台账

- [x] Task 3: 建立能力矩阵与冲突台账

**映射 OpenSpec:** 1.3

**Files:**
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: 本计划的 Canonical 能力矩阵、三个 tag 的 `git diff --name-only` 结果。
- Produces: 每行含行为契约、入口/调用链、关键文件、受影响 tag、聚焦测试、人工审查点、状态和证据的矩阵；以及冲突台账。

**Step 1: 获取三个 release 区间的 changed-files**

Run:
```powershell
git diff --name-only v0.1.165..v0.1.166
git diff --name-only v0.1.166..v0.1.168
git diff --name-only v0.1.168..v0.1.169
```

Expected: 三份清单分别写入 ledger 对应阶段，作为后续行为审查的输入，不以“测试全绿”替代矩阵结论。

**Step 2: 写入完整 canonical 能力矩阵**

将本计划“Canonical 能力矩阵”的全部 14 行逐字落入 ledger，并为每行填入当前状态 `protected`、`manual`、`unverified` 或 `gap`。Images 行必须逐项列出统一入口、legacy 单次、单次冻结、关闭态零大 payload、运行态/范围后求值、`ReleaseText` 前可用六项契约。

**Step 3: 建立六分类冲突台账格式**

每个实际冲突记录文件名、分类、ours 行为、theirs 行为、融合结果和验证证据。分类只允许“上游修复”“本地定制”“接口/配置演进”“版本/依赖”“生成代码”“migration”。

**Step 4: 将保护面证据严格提交为 ledger evidence**

Run:
```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'stage 0 capability matrix allowlist mismatch' }
git.exe commit -m "docs: record stage 0 capability matrix"
```

Expected: ledger 含完整矩阵和空的、可追踪的冲突台账；implementer 提交严格只含 build ledger，reviewer 通过后 controller 单独 check off Task 3/OpenSpec 1.3。

### Task 4: 验证阶段 0 的本地保护测试与生成稳定性

- [x] Task 4: 验证阶段 0 的本地保护测试与生成稳定性

**映射 OpenSpec:** 1.4

**Files:**
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`
- Test: `backend/internal/service/openai_account_scheduler_layered_test.go`
- Test: `backend/internal/handler/openai_images_controls_test.go`
- Modify when a new assertion exposes RED: `backend/internal/handler/openai_images.go`
- Test: `backend/internal/service/subscription_advance_quota_test.go`
- Test: `backend/internal/service/subscription_reset_quota_test.go`
- Modify test: `backend/internal/service/subscription_cache_invalidation_outbox_test.go`
- Generated: `backend/cmd/server/wire_gen.go`
- Modify for lint only: `backend/internal/repository/user_subscription_repo.go`
- Test: `frontend/src/utils/__tests__/subscriptionQuota.spec.ts`
- Modify for Step 9 review repair only: `backend/internal/handler/openai_gateway_handler.go`, `backend/internal/handler/openai_images.go`, `backend/internal/handler/openai_images_controls_test.go`
- Do not modify for Step 9: shared `handlerPromptEngine`, constructors, interfaces, or any file outside the preceding Step 9 allowlists.

**Interfaces:**
- Consumes: Canonical matrix中的所有 `protected` 行，以及已记录的 default service test undefined-helper 编译失败、stale Wire 输出和 deferred rows-close errcheck diagnostic evidence。
- Produces: 基线通过/失败证据；允许仅修复 outbox test build tag、`wire_gen.go` stale output 与 `user_subscription_repo.go` 的 deferred rows-close lint defect，并在 Step 9 以 request-scoped Images payload hook 复现及修复首轮 P1。高风险行只能由命名测试覆盖，不接受口头推断；ledger evidence 必须逐命令记录或引用既有 Fix-2 证据。

**Step 1: 补齐 Images 精确生命周期的直接保护测试**

在 `backend/internal/handler/openai_images_controls_test.go` 新增以下三个测试，复用当前 Images handler fixture 和它的 payload-provider 计数桩：

| 测试名 | 设置 | 必须断言 |
| --- | --- | --- |
| `TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload` | prompt audit 与 legacy moderation 均关闭，provider 返回大 payload 并递增原子计数 | handler 可继续既定请求路径，payload provider 调用数为 0。 |
| `TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope` | legacy moderation 配置为运行态或范围不匹配，provider 返回大 payload 并递增原子计数 | 在运行态和范围拒绝前 provider 调用数为 0，legacy moderation 不执行。 |
| `TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce` | 统一 coordinator 和 legacy moderation 都需要同一 Images 审计 payload | 请求完成前 provider 调用数为 1，legacy moderation 调用数为 1，文本直到审计结束才释放。 |

Run from `backend`:
```powershell
go test -count=1 ./internal/handler -run '^(TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$'
```

Expected: 三个直接测试 PASS；若任一 RED，先保留失败输出，再只修改 `openai_images.go` 使 provider 惰性、线程安全且单次冻结，禁止复制 legacy moderation 调用。

**Step 2: 提交 Images 保护测试和任何最小修复**

Run:
```powershell
git add backend/internal/handler/openai_images_controls_test.go backend/internal/handler/openai_images.go
$staged = @(git diff --cached --name-only)
$staged
$imagesAllow = @('backend/internal/handler/openai_images_controls_test.go', 'backend/internal/handler/openai_images.go')
$unexpected = $staged | Where-Object { $_ -notin $imagesAllow }
if ($unexpected -or $staged.Count -eq 0) { throw "Images protection allowlist mismatch: $($unexpected -join ', ')" }
git.exe commit -m "test: protect images audit lifecycle"
```

Expected: 此提交严格只包含实际改变的 Images 直接保护测试和所列必要的最小修复文件；provider/handler 已满足六项精确契约。

**Step 3: 用 build-tag TDD 修复 outbox test 的 baseline 编译缺陷**

以已记录的 `go test ./internal/service` default build 下 undefined helper 编译失败为 RED。在 `backend/internal/service/subscription_cache_invalidation_outbox_test.go` 文件首加入以下 build tag 和空行，使 consumer 与四个 unit-only helper owners 一致：

```go
//go:build unit

```

先运行原 service 聚焦命令确认其它目标在无 tag 的 default build 下 PASS，再运行精确 unit 命令覆盖该文件全部七个测试：

```powershell
go test -count=1 ./internal/service -run '^(TestLayered_GroupedAccountPassesDBFreshRecheck|TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount|TestLayered_SessionStickyPreservesGrokBinding|TestLayered_SessionStickyRecheckHonorsImageCapability|TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot|TestCalculateQuotaCycleAdvance_ResetsOnlySingleExhaustedWindow|TestAdvanceQuotaCycle_RejectsTwoExhaustedWindowsBeforeUpdate|TestAdminResetQuota_UsesCommittedResetVersionForCacheInvalidation|TestCheckAndResetWindows_UsesCommittedResetVersionForCacheInvalidation)$'
go test -count=1 -tags unit ./internal/service -run '^(TestSubscriptionCacheInvalidationWorker_(RequiresTombstoneAndPublishBeforeAck|UsesSafetySecondPass|PublishGetsIndependentTimeout)|TestSubscriptionCacheInvalidationFastPath_(WaitsForOuterCommit|NilCacheStillClearsLocalL1|UnknownVersionOnlyClearsLocalL1)|TestAdvanceQuotaCycle_UsesVersionedPostCommitInvalidation)$'
git add backend/internal/service/subscription_cache_invalidation_outbox_test.go
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('backend/internal/service/subscription_cache_invalidation_outbox_test.go') -DifferenceObject $staged) { throw 'outbox unit-tag allowlist mismatch' }
git.exe commit -m "test: align outbox tests with unit tag"
```

Expected: default service 聚焦目标均 PASS，unit 命令精确覆盖并 PASS 七个 outbox 测试；提交严格只含该一个 test 文件。

**Step 4: 刷新并稳定 Wire 生成输出**

以下 helper 只在本 Step 的同一 PowerShell Run block 内有效。每次调用先验证 clean Ent/Wire baseline；保存含 `<runId>-attempt-<attempt>` 的完整 stdout/stderr、exit code 和失败后 diff paths。只有至少一条 stderr 以 generated path 加精确 `user-mapped section open` signature 匹配，且剔除该行和已知 make/exit-status 包装后的其余 stderr 不含 `error`、`fail`、`panic` 或 `fatal`，才恢复本次生成的 Ent/Wire paths、等待 2 秒并最多重试三次；其它错误、非 generated path、恢复失败或第三次失败均 BLOCK。此规则不识别锁持有者、不杀进程，也不改 antivirus、CodeGraph、`wire.go`、provider 或 module。

```powershell
$runGenerateWithRetry = {
    param([Parameter(Mandatory)][string]$runId)
    for ($attempt = 1; $attempt -le 3; $attempt++) {
        $baselinePaths = @(git diff --name-only)
        $nonGeneratedBaseline = $baselinePaths | Where-Object { $_ -notlike 'backend/ent/*' -and $_ -ne 'backend/cmd/server/wire_gen.go' }
        if ($nonGeneratedBaseline -or $baselinePaths) { throw "generate requires a clean Ent/Wire baseline: $($baselinePaths -join ', ')" }

        $log = Join-Path $env:TEMP "sub2api-stage0-$runId-attempt-$attempt.log"
        $stderrLog = Join-Path $env:TEMP "sub2api-stage0-$runId-attempt-$attempt.stderr.log"
        & make -C backend generate 1> $log 2> $stderrLog
        $exitCode = $LASTEXITCODE
        "exit_code=$exitCode" | Add-Content -LiteralPath $log
        $failedPaths = @(git diff --name-only)
        "diff_paths=$($failedPaths -join ',')" | Add-Content -LiteralPath $log
        $nonGeneratedPaths = $failedPaths | Where-Object { $_ -notlike 'backend/ent/*' -and $_ -ne 'backend/cmd/server/wire_gen.go' }
        if ($nonGeneratedPaths) { throw "generate changed non-generated paths: $($nonGeneratedPaths -join ', ')" }
        if ($exitCode -eq 0) { return }

        $stderrLines = @(Get-Content -LiteralPath $stderrLog)
        $mappedSectionPattern = '^(?=.*backend[\\/](?:ent[\\/].+|cmd[\\/]server[\\/]wire_gen\.go))(?=.*user-mapped section open).+$'
        $mappedSectionLines = $stderrLines | Where-Object { $_ -match $mappedSectionPattern }
        if (-not $mappedSectionLines) { throw "generate failed without a retriable generated-path signature; inspect $log and $stderrLog" }
        $makeWrapperPattern = '^(?:make(?:\[\d+\])?:.*|.*exit status \d+.*)$'
        $residualStderr = $stderrLines | Where-Object { $_ -notmatch $mappedSectionPattern -and $_ -notmatch $makeWrapperPattern }
        $additionalErrors = $residualStderr | Where-Object { $_ -match '(?i)\b(error|fail|panic|fatal)\b' }
        if ($additionalErrors) { throw "generate stderr contains non-retriable errors: $($additionalErrors -join '; ')" }
        git restore --source=HEAD --worktree -- backend/ent backend/cmd/server/wire_gen.go
        git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
        if ($LASTEXITCODE -ne 0) { throw "generated paths could not be recovered; inspect $log and $stderrLog" }
        if ($attempt -eq 3) { throw "generate failed after three attempts; inspect $log and $stderrLog" }
        Start-Sleep -Seconds 2
    }
}

& $runGenerateWithRetry 'wire-refresh'
$wireDiff = @(git diff --name-only)
$wireDiff
if (Compare-Object -ReferenceObject @('backend/cmd/server/wire_gen.go') -DifferenceObject $wireDiff) { throw 'Wire generation changed paths outside wire_gen.go' }
git diff --check -- backend/cmd/server/wire_gen.go
git add backend/cmd/server/wire_gen.go
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('backend/cmd/server/wire_gen.go') -DifferenceObject $staged) { throw 'Wire output allowlist mismatch' }
git.exe commit -m "chore: refresh Wire output"
& $runGenerateWithRetry 'wire-stability-1'
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
& $runGenerateWithRetry 'wire-stability-2'
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
```

Expected: Wire commit 严格只含 `backend/cmd/server/wire_gen.go` 且人工确认 diff 仅为已诊断的依赖顺序/变量名生成变更；每次 generate 均有完整日志、exit 和 diff-path 证据，成功后两轮 generate 均退出 0 且不产生 Ent/Wire diff。

**Step 5: 用 lint TDD 修复 deferred subscription rows close**

以已记录的 `golangci-lint run ./internal/repository` 在 `backend/internal/repository/user_subscription_repo.go:578` 对 `defer rows.Close()` 的 unchecked errcheck 为 RED。Run from `backend`，并先运行：

```powershell
$redLintLog = Join-Path $env:TEMP "sub2api-stage0-repository-lint-red-$([guid]::NewGuid().ToString('N')).log"
golangci-lint run ./internal/repository 2>&1 | Tee-Object -FilePath $redLintLog
$redLintExit = $LASTEXITCODE
if ($redLintExit -eq 0) { throw 'expected deferred rows.Close errcheck RED' }
$lintFindings = @(Get-Content -LiteralPath $redLintLog | Where-Object { $_ -match '^[^:]+:\d+:\d+:' })
$expectedLintPattern = '^(?=.*internal[\\/]repository[\\/]user_subscription_repo\.go:578:18)(?=.*errcheck)(?=.*rows\.Close).+$'
$expectedLintFindings = $lintFindings | Where-Object { $_ -match $expectedLintPattern }
$unexpectedLintFindings = $lintFindings | Where-Object { $_ -notmatch $expectedLintPattern }
if ($expectedLintFindings.Count -ne 1 -or $unexpectedLintFindings) { throw "unexpected repository lint findings: $($lintFindings -join '; ')" }
```

只将该行改为 `defer func() { _ = rows.Close() }()`，复用同包既有 pattern；不新增抽象，不改变行为。然后运行并提交：

Run the same command from `backend` and preserve its GREEN log:

```powershell
$greenLintLog = Join-Path $env:TEMP "sub2api-stage0-repository-lint-green-$([guid]::NewGuid().ToString('N')).log"
golangci-lint run ./internal/repository 2>&1 | Tee-Object -FilePath $greenLintLog
if ($LASTEXITCODE -ne 0) { throw "repository lint remains RED; inspect $greenLintLog" }
git add backend/internal/repository/user_subscription_repo.go
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('backend/internal/repository/user_subscription_repo.go') -DifferenceObject $staged) { throw 'deferred rows close lint allowlist mismatch' }
git.exe commit -m "fix: handle deferred subscription rows close"
```

Expected: RED 必须只含精确的 `user_subscription_repo.go:578:18` `errcheck`/`rows.Close` finding，任何其它 lint finding BLOCK；同命令 GREEN 必须 exit 0。提交严格只含该 repository 文件的一行 deferred-close lint 修复，无新抽象或行为变化。

**Step 6: 运行 scheduler、sticky、gateway、审计、Images 和 quota 聚焦测试**

Run from `backend`:
```powershell
go test -count=1 ./internal/service -run '^(TestLayered_GroupedAccountPassesDBFreshRecheck|TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount|TestLayered_SessionStickyPreservesGrokBinding|TestLayered_SessionStickyRecheckHonorsImageCapability|TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot|TestCalculateQuotaCycleAdvance_ResetsOnlySingleExhaustedWindow|TestAdvanceQuotaCycle_RejectsTwoExhaustedWindowsBeforeUpdate|TestAdminResetQuota_UsesCommittedResetVersionForCacheInvalidation|TestCheckAndResetWindows_UsesCommittedResetVersionForCacheInvalidation)$'
go test -count=1 ./internal/handler -run '^(TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot|TestOpenAIImages_UnifiedAuditRunsLegacyOnce|TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease|TestOpenAIImages_SecurityAuditUsesFrozenPayloadBeforeRelease|TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$'
go test -count=1 ./internal/server/routes -run '^(TestEveryGatewayPOSTRouteIsClassifiedForPromptAuditCoverage|TestResponsesWebSocketHasFirstAndSubsequentTurnPromptGates)$'
go test -count=1 ./internal/service -run '^(TestLayered_FallbackWaitPlanRechecksPrivacyRequirementAgainstDB|TestLayered_PreviousResponseStickyHonorsRequirePrivacySet)$'
go test -count=1 ./internal/handler -run '^(TestOpenAIResponsesWebSocket_ReacquireSlotsOnSecondTurnWithoutDoubleRelease|TestOpenAIResponsesWebSocket_SecondTurnGroupAcquireFailureRollsBackUserSlot|TestOpenAIResponsesWebSocket_SecondTurnAccountAcquireFailureRollsBackUserAndGroupSlots)$'
go test -count=1 ./internal/service -run '^(TestRunLiveControllerClosesExpiredSession|TestLiveSidebandNormalCloseEndsCall|TestOpenAIWSPassthroughTurnLifecycle_SerializesTerminalCommitAndNextTurn)$'
go test -count=1 ./internal/service -run '^(TestGetModelPricing_Gpt54UsesStaticFallbackWhenRemoteMissing|TestOpenAIGatewayService_ForwardCountTokensAsAnthropic_OAuthFallsBackWhenPlatformEndpointUnsupported|TestGatewayServiceNewSelectionResult_ReleasesAcquiredSlotWhenHydrationFails)$'
go test -count=1 -tags unit ./internal/handler ./internal/handler/admin ./internal/service -run '^(TestGetMyPlatformQuotas_EmptyReturns200WithEmptyArray|TestGetMyPlatformQuotas_D14_LazyZeroForExpiredWindow|TestGetMyPlatformQuotas_NilRepo_Returns200Empty|TestGetMyPlatformQuotas_NoAuth_Returns401|TestLazyZeroQuotaForResponse_UserViewStripsWindowStart|TestLazyZeroQuotaForResponse_AdminViewIncludesWindowStart|TestLazyZeroQuotaForResponse_ActiveWindowPreservesUsage|TestUserHandlerBatchUpdateLimitsAcceptsPartialAndZeroValues|TestUserHandlerBatchUpdateLimitsRejectsInvalidRequests|TestUserHandlerBatchUpdateLimitsAllUsesEveryListedUser|TestDuplicateGroupHandlerReturnsAdminDTOWithoutOperationMetadata|TestDuplicateGroupHandlerRejectsInvalidID|TestDuplicateGroupHandlerReplaysSameIdempotencyKey|TestDuplicateGroupHandlerRecoversAfterMarkSucceededFailure|TestDuplicateGroupCopiesConfigurationDeeplyAndResetsRuntimeState|TestDuplicateGroupRecoversSameOperationAndScopesByAdmin|TestDuplicateGroupAdvancesNameAndTruncatesUnicodeByRunes|TestDuplicateGroupAtomicCreateFailureReturnsNoCopy)$'
go test -count=1 ./internal/service -run '^TestUserCanBindGroupRejectsBlockedPublicGroup$'
```

Run from repository root:
```powershell
pnpm --dir frontend exec vitest run src/utils/__tests__/subscriptionQuota.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/UsageView.spec.ts src/components/channels/__tests__/AvailableChannelsTable.spec.ts
```

Expected: 每个命名测试 PASS。rows 1、2、3、13 的现有 non-Docker 部分和 row 11 只有在上述直接命令实际 PASS 后才可记为 `protected`；row 11 可由其 18 个 unit-tag 测试与 default group-bind 测试直接记为 `protected`，不只为 `manual`。row 13 的 deploy scripts 尚未合入，明确为非当前 gap，由 Task 13 在 v0.1.169 merge 后执行并关闭该阶段证据；Task 5 只负责 row 10 quota/outbox/migration integration，不承接 row 13。Task 4 的三项已诊断 baseline gate defect 仅按 Step 3/4/5 修复；首轮 review P1 仅按 Step 9 的同请求 hook TDD 修复，不能通过修改 shared `handlerPromptEngine` 或改变生产默认路径规避。其余失败先在 ledger 保存 RED，再在首次引入该回归的 release 段写最小复现断言和兼容修复，不能在基线任务中改变产品行为。matrix 只有当前直接测试 PASS 才可记为 `protected`，仅完成具体调用链审查才可记为 `manual`；Task 4 只有在 Step 9 的 review repair、逐命令 ledger evidence、fresh reviewer 2/2 和 clean gate 均完成后才能闭合，否则 BLOCK。

**Step 7: 重跑阶段 0 full gate 与两轮生成检查**

Run from repository root:
```powershell
make test
make "VERSION=0.1.165.4" "SHELL=D:/scoop/shims/bash.exe" build
$runGenerateWithRetry = {
    param([Parameter(Mandatory)][string]$runId)
    for ($attempt = 1; $attempt -le 3; $attempt++) {
        $baselinePaths = @(git diff --name-only)
        $nonGeneratedBaseline = $baselinePaths | Where-Object { $_ -notlike 'backend/ent/*' -and $_ -ne 'backend/cmd/server/wire_gen.go' }
        if ($nonGeneratedBaseline -or $baselinePaths) { throw "generate requires a clean Ent/Wire baseline: $($baselinePaths -join ', ')" }

        $log = Join-Path $env:TEMP "sub2api-stage0-$runId-attempt-$attempt.log"
        $stderrLog = Join-Path $env:TEMP "sub2api-stage0-$runId-attempt-$attempt.stderr.log"
        & make -C backend generate 1> $log 2> $stderrLog
        $exitCode = $LASTEXITCODE
        "exit_code=$exitCode" | Add-Content -LiteralPath $log
        $failedPaths = @(git diff --name-only)
        "diff_paths=$($failedPaths -join ',')" | Add-Content -LiteralPath $log
        $nonGeneratedPaths = $failedPaths | Where-Object { $_ -notlike 'backend/ent/*' -and $_ -ne 'backend/cmd/server/wire_gen.go' }
        if ($nonGeneratedPaths) { throw "generate changed non-generated paths: $($nonGeneratedPaths -join ', ')" }
        if ($exitCode -eq 0) { return }

        $stderrLines = @(Get-Content -LiteralPath $stderrLog)
        $mappedSectionPattern = '^(?=.*backend[\\/](?:ent[\\/].+|cmd[\\/]server[\\/]wire_gen\.go))(?=.*user-mapped section open).+$'
        $mappedSectionLines = $stderrLines | Where-Object { $_ -match $mappedSectionPattern }
        if (-not $mappedSectionLines) { throw "generate failed without a retriable generated-path signature; inspect $log and $stderrLog" }
        $makeWrapperPattern = '^(?:make(?:\[\d+\])?:.*|.*exit status \d+.*)$'
        $residualStderr = $stderrLines | Where-Object { $_ -notmatch $mappedSectionPattern -and $_ -notmatch $makeWrapperPattern }
        $additionalErrors = $residualStderr | Where-Object { $_ -match '(?i)\b(error|fail|panic|fatal)\b' }
        if ($additionalErrors) { throw "generate stderr contains non-retriable errors: $($additionalErrors -join '; ')" }
        git restore --source=HEAD --worktree -- backend/ent backend/cmd/server/wire_gen.go
        git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
        if ($LASTEXITCODE -ne 0) { throw "generated paths could not be recovered; inspect $log and $stderrLog" }
        if ($attempt -eq 3) { throw "generate failed after three attempts; inspect $log and $stderrLog" }
        Start-Sleep -Seconds 2
    }
}

& $runGenerateWithRetry 'full-stability-1'
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
& $runGenerateWithRetry 'full-stability-2'
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git.exe diff --check
git.exe diff --cached --check
git diff --name-only --diff-filter=U
$conflictPattern = '^(<<<<<<< .+|\|\|\|\|\|\|\| .+|=======|>>>>>>> .+)$'
git grep -n -I -E $conflictPattern -- ':!docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md'
$conflictExit = $LASTEXITCODE
if ($conflictExit -eq 0) { throw 'tracked conflict markers found' }
if ($conflictExit -ne 1) { throw "conflict marker scan failed: $conflictExit" }
git rev-parse HEAD:backend/migrations/191_subscription_quota_advance_receipts.sql
git rev-parse HEAD:backend/migrations/192_subscription_cache_invalidation_outbox.sql
```

Expected: 两次 generate 均不产生 `backend/ent` 或 `backend/cmd/server/wire_gen.go` diff；两个 whitespace 命令无输出；unmerged 命令无输出；conflict marker scan 的 no-match exit `1` 是 PASS，exit `0` 的真实 marker 或其它 grep 错误均 BLOCK；最后两个命令精确输出 `c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6` 和 `502ecec1caf9f76e022c2e83acf3707190539301`。

**Step 8: 追加 repaired stage 0 gate 结果并严格提交 ledger**

Run:
```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'repaired stage 0 gates allowlist mismatch' }
git.exe commit -m "docs: record repaired stage 0 gates"
```

Expected: 仅在 Step 7 的 `make test`、VERSION/SHELL build、两轮 generate、除 conflict scan 外的静态检查和 migration OID 命令全部退出 0，且 conflict scan 为其指定的 no-match exit `1` 后，ledger 才追加每个修复后命令的退出码和命名失败；ledger 必须逐一记录 `wire-refresh`、`wire-stability-1`、`wire-stability-2`、`full-stability-1`、`full-stability-2` 的 stdout/stderr 日志路径。不得改写既有 `docs: record stage 0 local gates` 失败证据提交。不存在未归属的生成物。Images、unit-tag、Wire、lint 与 repaired ledger 五类提交保持分离，本提交严格只含 build ledger；Step 8 不闭合 Task 4，Step 9 的 review-repair code commit 和 final evidence commit 仍必需；implementer 不勾选 Plan/OpenSpec。

**Step 9: 闭合首轮 review findings**

本 Step 只处理已确认的 Images P1 与证据补全。禁止修改 shared `handlerPromptEngine`、新增接口、constructor 参数或任何生产默认语义；禁止运行 `make -C backend generate`、Docker、remote、`git fetch`、merge、push、tag、release 或部署命令。任何预期 RED/GREEN、复测或 clean gate 未满足都 BLOCK，保留日志并停止，不以环境或编译故障替代语义结果。

1. **P1 RED: 将测试 hook 接到同一 `ServeHTTP` 请求，但暂不接入 Images callsite。** 在 `backend/internal/handler/openai_gateway_handler.go` 的 `OpenAIGatewayHandler` 中添加未导出字段，且只用于测试观测：

```go
// nil preserves the production parsed.ModerationBody provider.
imagesModerationBody func(*service.OpenAIImagesRequest) []byte
```

在 `backend/internal/handler/openai_images_controls_test.go` 的三个既有 Images lifecycle 测试中，各自创建独立 handler，并在发起 request 前把计数 provider 设为该实例的 `imagesModerationBody`；request 开始后不得再变更 hook 或 handler。hook 必须保存收到的 `*service.OpenAIImagesRequest` 指针、递增 provider count，并返回 `parsed.ModerationBody()`。每个 `ServeHTTP` 都在 goroutine 中运行，使用 `done` channel 和 bounded `select` 等待完成，timeout 即 `t.Fatal`；此子步不得改动 `openai_images.go:160` callsite。

在同一 `openai_images_controls_test.go` 定义 test-local `blockingImagesPromptEngine`，不复用或修改 shared `handlerPromptEngine`。它实现 `securityaudit.PromptEngine` 的 `EffectiveMode() securityaudit.Mode`、`Enqueue(context.Context, securityaudit.Request) error` 和 `Evaluate(context.Context, securityaudit.Request) (*securityaudit.PromptDecision, error)`：`EffectiveMode` 返回 `securityaudit.ModeBlocking`，而且只有 `Evaluate` 实际被调用时才关闭/发送 `started`、等待 `release`、再返回允许继续的 `securityaudit.PromptDecision`。RED 子阶段不得将第三个测试配置为该 blocking engine，也不得等待 `started`；它必须使用可正常完成的 non-blocking engine，使旧 callsite 仍走真实 `parsed.ModerationBody`，然后通过 bounded `done` 得到 provider `expected/want 1, actual/got 0`。这使 RED 只证明 hook 尚未接入，而不会等待未触发的 hook 或死锁。

Run from `backend` before changing `openai_images.go`:

```powershell
$imagesRedLog = Join-Path $env:TEMP "sub2api-stage0-images-hook-red-$([guid]::NewGuid().ToString('N')).log"
go test -count=1 ./internal/handler -run '^(TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$' 2>&1 | Tee-Object -FilePath $imagesRedLog
$imagesRedExit = $LASTEXITCODE
if ($imagesRedExit -eq 0) { throw 'expected Images hook RED before the callsite uses imagesModerationBody' }
$imagesRed = Get-Content -Raw -LiteralPath $imagesRedLog
if ($imagesRed -notmatch 'TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce' -or $imagesRed -notmatch '(?is)(?:expected|want).*1.*(?:actual|got).*0|(?:actual|got).*0.*(?:expected|want).*1') { throw "expected third-test provider-count assertion RED; inspect $imagesRedLog" }
if ($imagesRed -match '(?i)(build failed|undefined:|docker|testcontainers)') { throw "RED must be the unused-callsite assertion, not a compile or environment failure; inspect $imagesRedLog" }
```

Expected: 前两个测试保持其同请求 `0` 计数契约；第三个测试以 non-blocking engine 在 bounded handler completion 后因 hook 尚未被 Images callsite 使用而得到 `expected/want 1, actual/got 0` RED。RED 不等待 `started`，也不得死锁；编译错误、Docker/Testcontainers 或其他环境失败不构成该 RED。

2. **P1 GREEN: 在唯一 callsite 接入 hook，保持 production 默认路径。** 仅在 `backend/internal/handler/openai_images.go` 当前 line 160 前建立 provider：hook 非 nil 时 provider 调用 `h.imagesModerationBody(parsed)`；hook 为 nil 时保持当前 `parsed.ModerationBody` method value。将该 provider 传给现有 `checkSecurityAuditLazy` 调用，随后保持 `coordinator.ReleaseMultipartValues()` 和 `parsed.ReleaseText()` 的原顺序。形状必须等价于：

```go
moderationBody := parsed.ModerationBody
if h.imagesModerationBody != nil {
	moderationBody = func() []byte { return h.imagesModerationBody(parsed) }
}
if decision := h.checkSecurityAuditLazy(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIImages, requestModel, moderationBody); decision != nil && !decision.AllowNextStage {
	h.openAISecurityAuditError(c, decision)
	return
}
```

不得增加新接口或 constructor 参数。三个测试分别断言同一请求内 provider/legacy 为 `0/0/1` 所需的计数。GREEN 的第三个测试才配置 `blockingImagesPromptEngine`：以 goroutine 启动 `ServeHTTP` 并用 bounded `select` 等待 `started`；先断言 provider count 为 1，再读取 captured parsed 指针并断言 `Prompt` 尚未清空。用 `sync.Once` 包装 `release` 函数，在创建 engine 后立即 `t.Cleanup(release)`，再主动调用 `release()`；以 bounded `select` 等待 `done`，最后断言 `Prompt` 已清空、provider count 为 1、legacy moderation count 为 1。任何 `started` 或 `done` timeout 都必须 `t.Fatal`，而 `t.Cleanup(release)` 仍保证解除阻塞。每个测试继续使用独立 handler，hook 必须在 request 前设置且 request 开始后不再 mutate。

Run from `backend`:

```powershell
go test -count=1 ./internal/handler -run '^(TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$'
go test -count=1 ./internal/handler -run '^(TestOpenAIImages_UnifiedAuditRunsLegacyOnce|TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease|TestOpenAIImages_SecurityAuditUsesFrozenPayloadBeforeRelease|TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$'
go test -count=1 ./internal/handler -run '^(TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot|TestOpenAIImages_UnifiedAuditRunsLegacyOnce|TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease|TestOpenAIImages_SecurityAuditUsesFrozenPayloadBeforeRelease|TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$'
```

Run from repository root:

```powershell
make test
```

Expected: 三个 hook 测试、完整八个 Images 测试、handler focused 命令和 `make test` 均 exit 0；GREEN 第三个测试的 blocking engine 只在 `Evaluate` 触发后发出 `started`，并由 cleanup-safe `release` 解除；production hook 为 nil 时仍只把 `parsed.ModerationBody` 传入原有审计路径。

3. **严格提交 P1 code/test repair。** 仅在前一子步全部 GREEN 后运行：

```powershell
git add backend/internal/handler/openai_gateway_handler.go backend/internal/handler/openai_images.go backend/internal/handler/openai_images_controls_test.go
$staged = @(git diff --cached --name-only)
$staged
$reviewRepairAllow = @('backend/internal/handler/openai_gateway_handler.go', 'backend/internal/handler/openai_images.go', 'backend/internal/handler/openai_images_controls_test.go')
$unexpected = $staged | Where-Object { $_ -notin $reviewRepairAllow }
if ($unexpected -or $staged.Count -ne $reviewRepairAllow.Count -or (Compare-Object -ReferenceObject $reviewRepairAllow -DifferenceObject $staged)) { throw "review repair allowlist mismatch: $($unexpected -join ', ')" }
git.exe commit -m "test: observe Images audit payload lifecycle"
```

Expected: commit 严格只含三个 P1 code/test 文件；ledger 留待下一子步，shared `handlerPromptEngine` 不在 diff 中。

4. **P2: 在 ledger 末尾追加逐命令 Review Repair Evidence。** 不改写旧 RED、旧 history 或旧 commit；在 `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md` 最末尾新增 `## Review Repair Evidence`。从 `.superpowers/sdd/2026-08-02-staged-merge-upstream-v0-1-169/task-4-brief.md` 逐字抄录 rows 1/2、row 3a、row 3b、row 13 non-Docker 与 row 11 的两条命令，且逐条保存 exact command、exit code 和关键输出；不得写“五组”或“全部通过”。对本轮没有重跑的每条命令，必须记录其 Fix-2 实际 evidence 的时间戳和来源，明确为引用而不是伪造的新运行。

Evidence 中还必须逐条列出下列全部九个 focused backend 命令、Vitest、`make test`、VERSION/SHELL build、repository lint、static/conflict/OID 检查，以及各自 exact command、exit 和关键输出：

```powershell
# Run from backend, retaining one exit code and key output for every command.
go test -count=1 ./internal/service -run '^(TestLayered_GroupedAccountPassesDBFreshRecheck|TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount|TestLayered_SessionStickyPreservesGrokBinding|TestLayered_SessionStickyRecheckHonorsImageCapability|TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot|TestCalculateQuotaCycleAdvance_ResetsOnlySingleExhaustedWindow|TestAdvanceQuotaCycle_RejectsTwoExhaustedWindowsBeforeUpdate|TestAdminResetQuota_UsesCommittedResetVersionForCacheInvalidation|TestCheckAndResetWindows_UsesCommittedResetVersionForCacheInvalidation)$'
go test -count=1 ./internal/handler -run '^(TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot|TestOpenAIImages_UnifiedAuditRunsLegacyOnce|TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease|TestOpenAIImages_SecurityAuditUsesFrozenPayloadBeforeRelease|TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$'
go test -count=1 ./internal/server/routes -run '^(TestEveryGatewayPOSTRouteIsClassifiedForPromptAuditCoverage|TestResponsesWebSocketHasFirstAndSubsequentTurnPromptGates)$'
go test -count=1 ./internal/service -run '^(TestLayered_FallbackWaitPlanRechecksPrivacyRequirementAgainstDB|TestLayered_PreviousResponseStickyHonorsRequirePrivacySet)$'
go test -count=1 ./internal/handler -run '^(TestOpenAIResponsesWebSocket_ReacquireSlotsOnSecondTurnWithoutDoubleRelease|TestOpenAIResponsesWebSocket_SecondTurnGroupAcquireFailureRollsBackUserSlot|TestOpenAIResponsesWebSocket_SecondTurnAccountAcquireFailureRollsBackUserAndGroupSlots)$'
go test -count=1 ./internal/service -run '^(TestRunLiveControllerClosesExpiredSession|TestLiveSidebandNormalCloseEndsCall|TestOpenAIWSPassthroughTurnLifecycle_SerializesTerminalCommitAndNextTurn)$'
go test -count=1 ./internal/service -run '^(TestGetModelPricing_Gpt54UsesStaticFallbackWhenRemoteMissing|TestOpenAIGatewayService_ForwardCountTokensAsAnthropic_OAuthFallsBackWhenPlatformEndpointUnsupported|TestGatewayServiceNewSelectionResult_ReleasesAcquiredSlotWhenHydrationFails)$'
go test -count=1 -tags unit ./internal/handler ./internal/handler/admin ./internal/service -run '^(TestGetMyPlatformQuotas_EmptyReturns200WithEmptyArray|TestGetMyPlatformQuotas_D14_LazyZeroForExpiredWindow|TestGetMyPlatformQuotas_NilRepo_Returns200Empty|TestGetMyPlatformQuotas_NoAuth_Returns401|TestLazyZeroQuotaForResponse_UserViewStripsWindowStart|TestLazyZeroQuotaForResponse_AdminViewIncludesWindowStart|TestLazyZeroQuotaForResponse_ActiveWindowPreservesUsage|TestUserHandlerBatchUpdateLimitsAcceptsPartialAndZeroValues|TestUserHandlerBatchUpdateLimitsRejectsInvalidRequests|TestUserHandlerBatchUpdateLimitsAllUsesEveryListedUser|TestDuplicateGroupHandlerReturnsAdminDTOWithoutOperationMetadata|TestDuplicateGroupHandlerRejectsInvalidID|TestDuplicateGroupHandlerReplaysSameIdempotencyKey|TestDuplicateGroupHandlerRecoversAfterMarkSucceededFailure|TestDuplicateGroupCopiesConfigurationDeeplyAndResetsRuntimeState|TestDuplicateGroupRecoversSameOperationAndScopesByAdmin|TestDuplicateGroupAdvancesNameAndTruncatesUnicodeByRunes|TestDuplicateGroupAtomicCreateFailureReturnsNoCopy)$'
go test -count=1 ./internal/service -run '^TestUserCanBindGroupRejectsBlockedPublicGroup$'
```

Also record the following commands from their indicated working directory:

```powershell
# Run from repository root.
pnpm --dir frontend exec vitest run src/utils/__tests__/subscriptionQuota.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/UsageView.spec.ts src/components/channels/__tests__/AvailableChannelsTable.spec.ts
make test
make "VERSION=0.1.165.4" "SHELL=D:/scoop/shims/bash.exe" build
git diff --check
git diff --cached --check
git diff --name-only --diff-filter=U
$conflictPattern = '^(<<<<<<< .+|\|\|\|\|\|\|\| .+|=======|>>>>>>> .+)$'
git grep -n -I -E $conflictPattern -- ':!docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md'
$conflictExit = $LASTEXITCODE
if ($conflictExit -eq 0) { throw 'tracked conflict markers found' }
if ($conflictExit -ne 1) { throw "conflict marker scan failed: $conflictExit" }
git rev-parse HEAD:backend/migrations/191_subscription_quota_advance_receipts.sql
git rev-parse HEAD:backend/migrations/192_subscription_cache_invalidation_outbox.sql
git status --short --untracked-files=all
```

```powershell
# Run from backend.
golangci-lint run ./internal/repository
```

明确记录旧 ledger line 738 的“row 13 交 Task 5”仅为当时失败的历史结论，现已被 supersede：Task 5 仅负责 row 10 Docker quota/outbox/migration integration；row 13 deploy scripts 仅由 v0.1.169 merge 后的 Task 13 执行。还要记录 controller 已读取 12 个 Temp generate 日志，保留真实 runId 前缀：失败 attempts 的 stderr 仅含 `user-mapped section open` announcement/read 与 `exit status`/`make` wrapper，stdout 记录的 11 个 `backend/ent` diff paths 均已恢复；retry 成功和其余四次均 exit 0 且 `diff_paths` 为空。不得改写任一旧日志或 evidence。

重跑最小 high-signal 命令只限三个/八个 Images 测试、default service focused、精确 unit outbox、repository lint、`make test`、`git diff --check`、`git diff --cached --check`、unmerged、conflict 和 status；其余 required evidence 只能逐条重跑或按前述 Fix-2 规则引用，不能概括为 PASS。

5. **严格提交 P2 ledger evidence，复核并闭合 Task 4。** 只有 P2 evidence 完整、所有实际重跑命令 GREEN 且 historic references 明确时，运行：

```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'review repair evidence allowlist mismatch' }
git.exe commit -m "docs: complete stage 0 review evidence"
$selectionStatus = '?? .comet/current-change.json'
$index = @(git diff --cached --name-only)
if ($index) { throw "review repair evidence left staged paths: $($index -join ', ')" }
$status = @(git status --short --untracked-files=all)
$unexpected = $status | Where-Object { $_ -ne $selectionStatus }
if ($unexpected) { throw "review repair evidence left dirty paths: $($unexpected -join '; ')" }
$status
```

Expected: P2 commit 严格只含 ledger，结束时 tracked worktree clean，status 只能为空或精确为 `?? .comet/current-change.json`。派发 fix 前，controller 必须将完整 Task 4（含 Step 9）同步到 ignored brief，并确保该 brief 不再授权 `backend/internal/handler/security_audit_helper.go`；brief stale 是 controller sync 工作，不是 Plan 内容缺陷。P1 commit 之后派发 fresh reviewer 2/2 审核 code/test 与 ledger；只有 reviewer 通过后 controller 才单独 check off Task 4/OpenSpec 1.4，implementer/reviewer 均不得修改 Plan、OpenSpec tasks 或 progress。

### Task 5: 执行阶段 0 row 10 Docker/Testcontainers 判定

- [x] Task 5: 执行阶段 0 row 10 Docker/Testcontainers 判定

**映射 OpenSpec:** 1.5

**Files:**
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: 本机 Docker daemon 与现有 repository integration fixture。
- Produces: 仅 row 10 quota/outbox/migration 的 `protected` 顶级测试证据，或带环境原因和契约影响的 `unverified` 条目；不承担 row 13 deploy scripts。

**Step 1: 运行轻量 Docker 预检并保存结果**

Run:
```powershell
$log = Join-Path $env:TEMP 'sub2api-stage0-docker-preflight.log'
$dockerCommand = Get-Command docker -ErrorAction SilentlyContinue
if ($null -eq $dockerCommand) { 'docker_command=unavailable' | Set-Content -LiteralPath $log; $preflightAvailable = $false } else { & $dockerCommand.Path version --format '{{.Server.Version}}' 2>&1 | Tee-Object -FilePath $log; $preflightExitCode = $LASTEXITCODE; "exit_code=$preflightExitCode" | Add-Content -LiteralPath $log; $preflightAvailable = ($preflightExitCode -eq 0) }
```

Expected: 输出 server version 且 `exit_code=0` 代表 Docker preflight 可用；`docker_command=unavailable` 直接代表 preflight unavailable，不读取陈旧 `$LASTEXITCODE`。命令不存在或 daemon 不可达时，将完整日志、可用的退出码和受影响契约记为 `unverified`，不尝试 SSH 或其他远程补验。

**Step 2: Docker 可用时运行基线 integration 并验证顶级 PASS**

Run from `backend` only when Step 1 succeeds:
```powershell
$targets = @('TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce', 'TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails', 'TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack', 'TestSubscriptionCacheInvalidationMigration_RawRerunIsIdempotent')
$log = Join-Path $env:TEMP 'sub2api-stage0-integration.log'
go test -tags=integration -v -count=1 -run '^(TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce|TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails|TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack|TestSubscriptionCacheInvalidationMigration_RawRerunIsIdempotent)$' ./internal/repository 2>&1 | Tee-Object -FilePath $log
$exitCode = $LASTEXITCODE
"exit_code=$exitCode" | Add-Content -LiteralPath $log
if ($exitCode -ne 0) { throw "integration failed; retain $log and classify the root cause before choosing BLOCK or unverified" }
foreach ($target in $targets) { $pattern = '^--- PASS: ' + [regex]::Escape($target) + ' \('; if (-not (Select-String -Path $log -Pattern $pattern -Quiet)) { throw "missing top-level PASS: $target" } }
```

Expected: 四个指定顶级测试都以锚定 `--- PASS: TestName (` 行出现，只证明 row 10 的 quota/outbox/migration integration。Docker preflight 成功后的非零退出先按 systematic debugging 保存/检查完整日志：断言或代码失败 BLOCK；只有日志证明 Docker/Testcontainers 环境不可用时才记为 `unverified`。`--- SKIP:`、`no tests to run` 或缺少单个 PASS 都不能通过。

**Step 3: 仅对已证明的环境阻塞记录 unverified**

预检失败时，在 ledger 记录实际预检命令、完整日志、退出码、未执行的四个 row 10 integration 目标，以及“PostgreSQL transaction/lock、receipt、outbox、migration 幂等未验证”的影响。预检成功但 integration 非零时，先依照日志执行 systematic debugging；只有 Docker/Testcontainers 不可用的证据才能设为 `unverified`，否则设为 BLOCK，不开始 v0.1.166。row 13 deploy scripts 由 Task 13 在 v0.1.169 merge 后单独执行，不能在此记录为 Docker gap。

**Step 4: 提交 Docker/阶段 0 ledger evidence**

在 ledger 写明 Docker 结论、日志位置、退出码、`protected`/`unverified` 状态和残余风险。implementer 不修改或暂存 `tasks.md`。

Run:
```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'stage 0 Docker evidence allowlist mismatch' }
git.exe commit -m "docs: close stage 0 baseline"
$selectionStatus = '?? .comet/current-change.json'
$index = @(git diff --cached --name-only)
if ($index) { throw "stage 0 evidence left staged paths: $($index -join ', ')" }
$status = @(git status --short --untracked-files=all)
$unexpected = $status | Where-Object { $_ -ne $selectionStatus }
if ($unexpected) { throw "stage 0 evidence left dirty paths: $($unexpected -join '; ')" }
$status
```

Expected: implementer 提交严格只含 row 10 Docker/ledger evidence；最终 index 为空，status 过滤实际存在的 `?? .comet/current-change.json` 后为空，也没有远程服务器操作。reviewer 通过后 controller 只勾选 Task 5/OpenSpec 1.5；此前 Task 1-4 都已各自 check off，因此此时 1.1-1.5 均完成。

### Task 6: 在未提交状态合入 v0.1.166 并完成阻塞审查

- [x] Task 6: 在未提交状态合入 v0.1.166 并完成阻塞审查

**映射 OpenSpec:** 2.1

**Files:**
- Modify: 下列 17 个预测冲突文件，以及 `v0.1.165..v0.1.166` 的实际 changed files。
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: 从 `$executionBase` 推进且干净的阶段 0 HEAD 和 `v0.1.166^{}`。
- Produces: 第二父为 `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` 的 merge commit 与独立 merge-review ledger evidence；所有语义不兼容在提交前阻塞。

**Step 1: 启动唯一允许的首段 merge**

Run:
```powershell
$selectionStatus = '?? .comet/current-change.json'
$index = @(git diff --cached --name-only)
if ($index) { throw "v0.1.166 merge blocked by staged paths: $($index -join ', ')" }
$status = @(git status --short --untracked-files=all)
$unexpected = $status | Where-Object { $_ -ne $selectionStatus }
if ($unexpected) { throw "v0.1.166 merge blocked by dirty paths: $($unexpected -join '; ')" }
git.exe merge --no-ff --no-commit v0.1.166
$conflictPaths = @(git diff --name-only --diff-filter=U)
$conflictPaths
```

Expected: merge 前 `$index` 必须为空，`$status` 只能为空或精确为 `?? .comet/current-change.json`；其它任何用户路径均阻塞并要求 Comet 重建隔离位置。merge 停留在未提交状态；`$conflictPaths` 是唯一可用于冲突暂存的实际清单，必须完整写入 ledger。预测的 17 个文本冲突仅供审查参考：

| 文件 | 冲突分类和必须保留的语义 |
| --- | --- |
| `backend/cmd/server/VERSION` | 版本/依赖；保留本地 `0.1.165.4`。 |
| `backend/go.sum` | 版本/依赖；由融合后的 `go.mod` 和 Go module 工具生成。 |
| `backend/internal/config/config_test.go` | 接口/配置演进；保留本地配置断言并加入上游字段。 |
| `backend/internal/handler/admin/setting_handler_update.go` | 接口/配置演进；保留本地设置热更新并融合上游部分更新。 |
| `backend/internal/handler/gateway_handler_responses.go` | 本地定制；保留本地 Responses 路由/审计/调度入口与上游行为。 |
| `backend/internal/handler/openai_gateway_handler.go` | 本地定制；保留 HTTP gateway、usage、failover 和审计入口。 |
| `backend/internal/handler/openai_gateway_handler_test.go` | 本地定制；同时断言本地和上游 handler 契约。 |
| `backend/internal/server/router.go` | 接口/配置演进；注册上游路由而不绕过本地 middleware/audit。 |
| `backend/internal/service/gateway_forward_as_responses.go` | 本地定制；保留 Responses 转发、prompt cache 和 body replay。 |
| `backend/internal/service/gateway_forward_as_responses_test.go` | 本地定制；保留缓存 usage 和转发字段断言。 |
| `backend/internal/service/gateway_record_usage_test.go` | 本地定制；保留最终 upstream model、failed usage 和计费上下文断言。 |
| `backend/internal/service/gemini_chat_completions_compat_service.go` | 接口/配置演进；保留 compat 映射且融合上游变更。 |
| `backend/internal/service/ollama_cloud_usage_test.go` | 本地定制；保留 Ollama usage 归属断言。 |
| `backend/internal/service/openai_ws_forwarder.go` | 本地定制；保留每轮模型、terminal ownership、failed usage 和 prompt-cache reuse。 |
| `backend/internal/service/openai_ws_v2/passthrough_relay.go` | 本地定制；保留 relay turn 生命周期和最终模型。 |
| `backend/internal/service/openai_ws_v2_passthrough_adapter.go` | 本地定制；保留 adapter usage/model 映射。 |
| `frontend/src/views/admin/__tests__/UsageView.spec.ts` | 接口/配置演进；保留本地用量页面断言并融合上游 UI 行为。 |

**Step 2: 在 merge commit 前执行阻塞式审查**

逐个解决 `$conflictPaths` 的所有实际文件，写入冲突台账的 ours/theirs/fusion/证据；预测表未列出的冲突也必须处理，不能只暂存表内 17 项。审查 `config.go`、settings DTO/handler、router、Responses 转发、OpenAI handler、WS relay/adapter、Gemini compat、usage 调用链和前端 UsageView；确认 panel API rate limit、settings 部分更新、每轮模型计费、effective composite route、account failover、terminal ownership 与本地 quota reset 不存在不可共存语义。

Run after resolving every actual conflict:
```powershell
if ($conflictPaths.Count -gt 0) { git add -- $conflictPaths }
git diff --name-only --diff-filter=U
```

Expected: 最后一个命令无输出；非冲突上游结果已由 merge 放入 index，不用宽泛 `git add` 重扫工作树。

阻塞条件：任何未决 conflict marker、未注册路由、DTO/config 字段丢失、请求绕过审计/usage、或无法同时保留上游修复与本地定制时，不得创建 merge commit，记录证据后等待用户取舍。

**Step 3: 由工具生成依赖和生成输出**

Run from `backend` after source/manifest 融合:
```powershell
go mod tidy
make generate
```

Run from repository root only when `frontend/package.json` 发生冲突或变更:
```powershell
pnpm --dir frontend install --lockfile-only --frozen-lockfile=false
```

Expected: `go.sum`、`backend/ent`、`backend/cmd/server/wire_gen.go` 和 `frontend/pnpm-lock.yaml` 仅由对应工具更新；存在变更时进入本 merge commit。

**Step 4: 创建 v0.1.166 merge commit 并核验第二父**

Run:
```powershell
git add backend/go.mod backend/go.sum backend/ent backend/cmd/server/wire_gen.go frontend/pnpm-lock.yaml
git diff --name-only --diff-filter=U
git diff --cached --check
$staged = @(git diff --cached --name-only)
$staged
$mergePlanningPaths = @('.comet/current-change.json', 'docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md', 'docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-verify.md', 'docs/superpowers/plans/2026-08-02-staged-merge-upstream-v0-1-169.md', 'docs/superpowers/specs/2026-08-02-staged-merge-upstream-v0-1-169-design.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/proposal.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/design.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/specs/upstream-release-sync/spec.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/tasks.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/subagent-progress.md')
$forbidden = $staged | Where-Object { $_ -in $mergePlanningPaths }
if ($forbidden) { throw "forbidden paths staged for merge: $($forbidden -join ', ')" }
git commit --no-edit
git rev-parse HEAD^2
```

Expected: unmerged 命令无输出，cached whitespace 检查通过；`$staged` 由上游自动暂存结果、所有实际 `$conflictPaths` 和工具产生的 `backend/go.mod`、`backend/go.sum`、Ent/Wire/lockfile 组成，且不含 runtime selection、ledger、计划、Design Doc 或任一 OpenSpec 规划产物。第二父精确为 `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8`。

**Step 5: 严格提交 v0.1.166 merge-review ledger evidence**

在 ledger 记录实际 `$conflictPaths`、六分类冲突台账、合并后的第二父和阻塞审查结论，然后运行：

```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'v0.1.166 merge-review evidence allowlist mismatch' }
git.exe commit -m "docs: record v0.1.166 merge review"
```

Expected: 此 evidence commit 严格只含 build ledger，保持 preceding merge commit 纯 merge；reviewer 通过后 controller 单独 check off Task 6/OpenSpec 2.1。

### Task 7: 对 v0.1.166 运行 merge 后行为审查并修复 RED

- [ ] Task 7: 对 v0.1.166 运行 merge 后行为审查并修复 RED

**映射 OpenSpec:** 2.2

**Files:**
- Modify when RED exists: `backend/internal/handler/admin/setting_handler_update.go`, `backend/internal/handler/openai_gateway_handler.go`, `backend/internal/service/gateway_forward_as_responses.go`, `backend/internal/service/openai_ws_forwarder.go`, `backend/internal/service/openai_ws_v2/passthrough_relay.go`, `backend/internal/service/openai_ws_v2_passthrough_adapter.go` 和对应命名测试。
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: v0.1.166 merge 后 changed-files 和阶段 0 保护测试。
- Produces: `protected` 或 `manual` 的阶段 1 行；可复现回归只能由独立兼容提交修复。

**Step 1: 先运行 v0.1.166 的行为聚焦测试，保存任何 RED**

Run from `backend`:
```powershell
go test -count=1 ./internal/server/middleware -run '^(TestPanelRateLimiterGlobalPerUser|TestPanelRateLimiterFailOpenOnRedisError)$'
go test -count=1 ./internal/handler/admin -run '^(TestUpdateSettingsPartialPayloadKeepsUnsentKeys|TestUpdateSettingsFullPayloadStillClearsSentEmptyFields)$'
go test -count=1 ./internal/service -run '^(TestCompositeRouteResolverExplicitExactRouteRewritesModel|TestOpenAIGatewayService_Forward_WSv2_ResponseDoneUsageParsed|TestRelay_OnTurnComplete_UsesCurrentResponseCreateModel|TestRelay_OnTurnComplete_PerTerminalEvent|TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot|TestGatewayService_ForwardAsResponses_PassthroughHeaderForwardCopiesFromClientRequest)$'
go test -count=1 ./internal/handler -run '^(TestGatewayChatCredentialStopDoesNotSelectAnotherAccountAndReturnsSafe503|TestOpenAIUsageRecordTaskCopiesCompositeBillingContextAfterQueueDelay)$'
```

Expected: 每个测试 PASS，且 changed-files × matrix 审查明确说明上游限流、部分设置、WS 计费、composite route 没有回归本地 scheduler/sticky/fallback/usage。

**Step 2: 对发现的 RED 用失败测试驱动最小兼容修复**

在保持失败输出的前提下，先在失败测试所在文件加入精确回归断言，再仅修改对应生产文件：settings RED 使用 `setting_handler_update.go`；Responses/body replay RED 使用 `gateway_forward_as_responses.go`；HTTP gateway/usage RED 使用 `openai_gateway_handler.go`；WS turn/terminal/model RED 使用 `openai_ws_forwarder.go`、`passthrough_relay.go` 或 `passthrough_adapter.go`。重复 Step 1 的精确命令直到 PASS。

**Step 3: 独立提交每组兼容修复**

Run for each non-empty RED fix set:
```powershell
git add backend/internal/handler/admin/setting_handler_update.go backend/internal/handler/admin/setting_handler_partial_payload_test.go backend/internal/handler/openai_gateway_handler.go backend/internal/handler/openai_gateway_handler_test.go backend/internal/service/gateway_forward_as_responses.go backend/internal/service/gateway_forward_as_responses_test.go backend/internal/service/openai_ws_forwarder.go backend/internal/service/openai_ws_forwarder_success_test.go backend/internal/service/openai_ws_v2/passthrough_relay.go backend/internal/service/openai_ws_v2/passthrough_relay_test.go backend/internal/service/openai_ws_v2_passthrough_adapter.go backend/internal/service/openai_ws_v2_passthrough_adapter_effort_test.go backend/ent backend/cmd/server/wire_gen.go
$staged = @(git diff --cached --name-only)
$staged
$compatAllow = @('backend/internal/handler/admin/setting_handler_update.go', 'backend/internal/handler/admin/setting_handler_partial_payload_test.go', 'backend/internal/handler/openai_gateway_handler.go', 'backend/internal/handler/openai_gateway_handler_test.go', 'backend/internal/service/gateway_forward_as_responses.go', 'backend/internal/service/gateway_forward_as_responses_test.go', 'backend/internal/service/openai_ws_forwarder.go', 'backend/internal/service/openai_ws_forwarder_success_test.go', 'backend/internal/service/openai_ws_v2/passthrough_relay.go', 'backend/internal/service/openai_ws_v2/passthrough_relay_test.go', 'backend/internal/service/openai_ws_v2_passthrough_adapter.go', 'backend/internal/service/openai_ws_v2_passthrough_adapter_effort_test.go', 'backend/cmd/server/wire_gen.go')
$unexpected = $staged | Where-Object { $_ -notin $compatAllow -and $_ -notlike 'backend/ent/*' }
if ($unexpected) { throw "v0.1.166 compatibility allowlist mismatch: $($unexpected -join ', ')" }
git.exe commit -m "fix: preserve local behavior after v0.1.166"
```

Expected: 每个兼容提交同时包含复现测试、最小源修复和由该修复触发的生成输出；无 RED 时不创建空兼容提交。

**Step 4: 严格提交 merge 后行为审查 ledger evidence**

在 ledger 按能力矩阵写入每个测试命令、审查的调用链、RED 和兼容提交 SHA；确认 `gap=0` 后运行：

```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'v0.1.166 behavior-review evidence allowlist mismatch' }
git.exe commit -m "docs: record v0.1.166 behavior review"
```

Expected: 兼容代码提交保持独立，无 RED 时不创建空兼容提交；此 evidence commit 严格只含 build ledger，reviewer 通过后 controller 单独 check off Task 7/OpenSpec 2.2。

### Task 8: 封闭 v0.1.166 的本机门禁与阶段证据

- [ ] Task 8: 封闭 v0.1.166 的本机门禁与阶段证据

**映射 OpenSpec:** 2.3

**Files:**
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: v0.1.166 merge 与所有兼容提交。
- Produces: 可作为 v0.1.168 前置条件的封闭阶段 1 证据。

**Step 1: 运行 v0.1.166 full gate**

Run from repository root:
```powershell
make test
make "VERSION=0.1.165.4" "SHELL=D:/scoop/shims/bash.exe" build
make -C backend generate
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git.exe diff --check
git.exe diff --cached --check
git diff --name-only --diff-filter=U
$conflictPattern = '^(<<<<<<< .+|\|\|\|\|\|\|\| .+|=======|>>>>>>> .+)$'
git grep -n -I -E $conflictPattern -- ':!docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md'
$conflictExit = $LASTEXITCODE
if ($conflictExit -eq 0) { throw 'tracked conflict markers found' }
if ($conflictExit -ne 1) { throw "conflict marker scan failed: $conflictExit" }
git rev-parse HEAD:backend/migrations/191_subscription_quota_advance_receipts.sql
git rev-parse HEAD:backend/migrations/192_subscription_cache_invalidation_outbox.sql
```

Expected: 所有命令满足全局约束；VERSION 仍为 `0.1.165.4`；conflict marker scan 的 no-match exit `1` 是 PASS，exit `0` 的真实 marker 或其它 grep 错误均 BLOCK；最后两个命令精确输出两个固定 base-ref blob OID。

**Step 2: 运行 v0.1.166 Docker 判定与 integration**

Run:
```powershell
$preflightLog = Join-Path $env:TEMP 'sub2api-v0166-docker-preflight.log'
$dockerCommand = Get-Command docker -ErrorAction SilentlyContinue
if ($null -eq $dockerCommand) { 'docker_command=unavailable' | Set-Content -LiteralPath $preflightLog; $preflightAvailable = $false } else { & $dockerCommand.Path version --format '{{.Server.Version}}' 2>&1 | Tee-Object -FilePath $preflightLog; $preflightExitCode = $LASTEXITCODE; "exit_code=$preflightExitCode" | Add-Content -LiteralPath $preflightLog; $preflightAvailable = ($preflightExitCode -eq 0) }
```

When available, run from `backend`:
```powershell
$targets = @('TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce', 'TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails', 'TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack')
$log = Join-Path $env:TEMP 'sub2api-v0166-integration.log'
go test -tags=integration -v -count=1 -run '^(TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce|TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails|TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack)$' ./internal/repository 2>&1 | Tee-Object -FilePath $log
$exitCode = $LASTEXITCODE
"exit_code=$exitCode" | Add-Content -LiteralPath $log
if ($exitCode -ne 0) { throw "integration failed; retain $log and classify the root cause before choosing BLOCK or unverified" }
foreach ($target in $targets) { $pattern = '^--- PASS: ' + [regex]::Escape($target) + ' \('; if (-not (Select-String -Path $log -Pattern $pattern -Quiet)) { throw "missing top-level PASS: $target" } }
```

Expected: 仅在 `$preflightAvailable` 为 true 的环境中运行 integration，三项逐一以锚定顶级 PASS 匹配；命令不存在或 preflight 失败直接记录 `unverified`。preflight 成功后的非零退出先按 systematic debugging 分类，代码/断言失败 BLOCK，只有 Docker/Testcontainers 环境不可用证据可记为 `unverified`，不远程补跑。

**Step 3: 提交阶段 1 证据并验证清洁度**

Run:
```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'v0.1.166 evidence allowlist mismatch' }
git.exe commit -m "docs: close v0.1.166 merge stage"
$selectionStatus = '?? .comet/current-change.json'
$index = @(git diff --cached --name-only)
if ($index) { throw "v0.1.166 evidence left staged paths: $($index -join ', ')" }
$status = @(git status --short --untracked-files=all)
$unexpected = $status | Where-Object { $_ -ne $selectionStatus }
if ($unexpected) { throw "v0.1.166 evidence left dirty paths: $($unexpected -join '; ')" }
$status
```

Expected: implementer 提交严格只含 ledger；最终 index 为空，status 过滤实际存在的 `?? .comet/current-change.json` 后为空，任何其它路径阻塞。reviewer 通过后 controller 单独 check off Task 8/OpenSpec 2.3。

### Task 9: 在未提交状态合入 v0.1.168 并完成阻塞审查

- [ ] Task 9: 在未提交状态合入 v0.1.168 并完成阻塞审查

**映射 OpenSpec:** 3.1

**Files:**
- Modify: `backend/migrations/191_passkey_credentials.sql` 与 `v0.1.166..v0.1.168` 的实际 changed files。
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: 从 `$executionBase` 经 v0.1.166 阶段推进的封闭 HEAD 和 `v0.1.168^{}`。
- Produces: 第二父为 `99c8e4bf7564823bafbab369acab6539e734c1bb` 的 merge commit 与独立 merge-review ledger evidence。

**Step 1: 启动唯一允许的第二段 merge**

Run:
```powershell
$selectionStatus = '?? .comet/current-change.json'
$index = @(git diff --cached --name-only)
if ($index) { throw "v0.1.168 merge blocked by staged paths: $($index -join ', ')" }
$status = @(git status --short --untracked-files=all)
$unexpected = $status | Where-Object { $_ -ne $selectionStatus }
if ($unexpected) { throw "v0.1.168 merge blocked by dirty paths: $($unexpected -join '; ')" }
git.exe merge --no-ff --no-commit v0.1.168
$conflictPaths = @(git diff --name-only --diff-filter=U)
$conflictPaths
```

Expected: merge 前 `$index` 必须为空，`$status` 只能为空或精确为 `?? .comet/current-change.json`；其它任何用户路径均阻塞并要求 Comet 重建隔离位置。merge 不提交；`$conflictPaths` 全量写入六分类台账。

**Step 2: 在 merge commit 前阻塞审查 Passkey、模型广场和本地数据语义**

在未提交 merge 状态审查 `backend/internal/handler/passkey_handler.go`、`backend/internal/service/passkey.go`、`backend/internal/repository/passkey_repo.go`、`backend/internal/handler/model_plaza_handler.go`、`backend/internal/repository/user_repo.go`、`backend/internal/repository/api_key_repo.go`、`backend/internal/securityaudit/prompt_config.go`、`backend/internal/service/openai_live.go`、Wire/provider、路由和前端 `frontend/src/router/index.ts`、`frontend/src/views/ModelPlazaView.vue`、`frontend/src/views/user/ProfileView.vue`。

确认 repository scoped update、DTO 演进、settings backfill、用户资源控制、隐藏菜单、session binding、step-up 与 subscription quota reset 的 receipt、事务锁、版本化 tombstone/outbox 不丢字段也不绕过。确认三份迁移文件原名共存，禁止将任一 `191_*` 重命名。逐项解决实际 `$conflictPaths` 后运行 `if ($conflictPaths.Count -gt 0) { git add -- $conflictPaths }` 和 `git diff --name-only --diff-filter=U`；后者必须无输出。

阻塞条件：Passkey/auth route 未注册、模型广场越过本地权限、scoped update 丢失本地 quota 数据、Wire/Ent 生成源与输出不一致，或 migration filename/checksum 被修改时，不得提交 merge。

**Step 3: 重新生成需要工具维护的输出**

Run from `backend`:
```powershell
go mod tidy
make generate
```

Run from repository root when frontend manifest 改变:
```powershell
pnpm --dir frontend install --lockfile-only --frozen-lockfile=false
```

Expected: Ent/Wire、Go checksum 与 lockfile 都由工具生成；发生变化的输出归入当前 merge commit。

**Step 4: 创建 v0.1.168 merge commit 并核验第二父**

Run:
```powershell
git add backend/migrations/191_passkey_credentials.sql backend/ent backend/cmd/server/wire_gen.go backend/go.mod backend/go.sum frontend/pnpm-lock.yaml
git diff --name-only --diff-filter=U
git diff --cached --check
$staged = @(git diff --cached --name-only)
$staged
$mergePlanningPaths = @('.comet/current-change.json', 'docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md', 'docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-verify.md', 'docs/superpowers/plans/2026-08-02-staged-merge-upstream-v0-1-169.md', 'docs/superpowers/specs/2026-08-02-staged-merge-upstream-v0-1-169-design.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/proposal.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/design.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/specs/upstream-release-sync/spec.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/tasks.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/subagent-progress.md')
$forbidden = $staged | Where-Object { $_ -in $mergePlanningPaths }
if ($forbidden) { throw "forbidden paths staged for merge: $($forbidden -join ', ')" }
git.exe commit --no-edit
git rev-parse HEAD^2
```

Expected: 所有实际 `$conflictPaths` 已逐项暂存，unmerged/cached whitespace 检查通过；`$staged` 不含 runtime selection、ledger、计划、Design Doc 或任一 OpenSpec 规划产物，且 Go manifest/checksum、Ent/Wire/lockfile 的工具输出均在本 merge commit。第二父精确为 `99c8e4bf7564823bafbab369acab6539e734c1bb`。

**Step 5: 严格提交 v0.1.168 merge-review ledger evidence**

在 ledger 记录实际 `$conflictPaths`、Passkey/模型广场/migration 阻塞审查和合并后的第二父，然后运行：

```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'v0.1.168 merge-review evidence allowlist mismatch' }
git.exe commit -m "docs: record v0.1.168 merge review"
```

Expected: 此 evidence commit 严格只含 build ledger，保持 preceding merge commit 纯 merge；reviewer 通过后 controller 单独 check off Task 9/OpenSpec 3.1。

### Task 10: 对 v0.1.168 审查交互并新增 migration 升级回归测试

- [ ] Task 10: 对 v0.1.168 审查交互并新增 migration 升级回归测试

**映射 OpenSpec:** 3.2

**Files:**
- Create: `backend/internal/repository/migrations_subscription_quota_passkey_upgrade_integration_test.go`
- Modify when RED exists: Passkey/model plaza/repository/prompt audit/OpenAI Live 的最小对应生产文件和测试。
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: `applyMigrationsFS(ctx context.Context, db *sql.DB, fsys fs.FS) error`、完整 embedded migration FS。
- Produces: `TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade`，它先应用排除 `191_passkey_credentials.sql` 的基线 FS，再应用完整 FS。

**Step 1: 写出 migration 升级 RED 测试**

在新测试文件中创建 test-local `fstest.MapFS`：从完整 embedded migration `fs.FS` 复制所有 `.sql`，唯独排除 `191_passkey_credentials.sql`。在 test 函数前写入注释：`filtered FS 仅模拟从本地 0.1.165.4 集合升级；历史本地 migration 不变性由 build ledger 的 base-ref Git blob OID 断言单独证明。` 测试按以下顺序执行并断言：

| 顺序 | 操作 | 必须断言 |
| --- | --- | --- |
| 1 | `applyMigrationsFS(ctx, integrationDB, baselineFS)` | 基线包含 `191_subscription_quota_advance_receipts.sql` 和 `192_subscription_cache_invalidation_outbox.sql`，不含 Passkey 迁移。 |
| 2 | `applyMigrationsFS(ctx, integrationDB, completeFS)` | `191_passkey_credentials.sql` 被新增执行，两个本地 migration 保持原 checksum。 |
| 3 | 再次 `applyMigrationsFS(ctx, integrationDB, completeFS)` | 无 checksum mismatch，`schema_migrations` 中三个完整 filename 各一行。 |
| 4 | 查询 schema | Passkey 表存在；receipt 和 invalidation outbox 表仍存在，说明升级未丢本地数据面。 |

Run from `backend` before实现任何兼容修复:
```powershell
go test -tags=integration -v -count=1 -run '^TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade$' ./internal/repository
```

Expected: Docker 可用时测试必须在基线 FS、完整 FS 和完整 FS 重跑三步都通过；Docker/Testcontainers 不可用时记录 `unverified`，不得以编译或包级成功替代该升级路径。

**Step 2: 运行 Passkey、模型广场、scoped update、prompt audit 和 Live 聚焦测试**

Run from `backend`:
```powershell
go test -count=1 ./internal/handler -run '^(TestPasskey|TestModelPlaza|TestUserHandler)'
go test -count=1 ./internal/service -run '^(TestPasskey|TestChannelPlaza|TestOpenAILive|TestSettingServiceUpdate|TestUserServiceUpdateFields)'
go test -count=1 ./internal/securityaudit -run '^(TestPromptConfig|TestPromptHandler|TestPromptService)'
go test -count=1 ./internal/repository -run '^(TestUserRepo|TestUserRepoAPIKeyGroupFilterSuite|TestUserRepository)'
```

Run from repository root:
```powershell
pnpm --dir frontend exec vitest run src/api/__tests__/passkey.spec.ts src/components/modelPlaza/__tests__/PlazaModelPricingTable.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/stores/__tests__/app.spec.ts
```

Expected: 所有已匹配的命名测试 PASS；没有匹配到测试的命令必须先用 `go test -list` 确认 target 的实际名称，再在 ledger 以实际存在的顶级测试名重跑，不能把 `no tests to run` 记为 PASS。

**Step 3: 用最小兼容提交修复可复现 RED**

对每个 RED，先在该组件的现有 `_test.go` 文件添加只覆盖丢失语义的断言，然后只修改产生该行为的 Passkey、model plaza、repository、prompt config 或 Live 文件。若修改 schema/provider 源，运行 `make -C backend generate` 并在同一提交加入 `backend/ent` 和 `backend/cmd/server/wire_gen.go`。

Run:
```powershell
git add backend/internal/repository/migrations_subscription_quota_passkey_upgrade_integration_test.go backend/internal/handler/passkey_handler.go backend/internal/handler/passkey_handler_test.go backend/internal/handler/model_plaza_handler.go backend/internal/handler/model_plaza_handler_test.go backend/internal/service/passkey.go backend/internal/service/passkey_test.go backend/internal/service/channel_plaza.go backend/internal/service/channel_plaza_test.go backend/internal/service/openai_live.go backend/internal/service/openai_live_lifecycle_test.go backend/internal/repository/user_repo.go backend/internal/repository/user_repo_integration_test.go backend/internal/repository/api_key_repo.go backend/internal/repository/api_key_repo_integration_test.go backend/internal/securityaudit/prompt_config.go backend/internal/securityaudit/prompt_config_test.go backend/ent backend/cmd/server/wire_gen.go
$staged = @(git diff --cached --name-only)
$staged
$compatAllow = @('backend/internal/repository/migrations_subscription_quota_passkey_upgrade_integration_test.go', 'backend/internal/handler/passkey_handler.go', 'backend/internal/handler/passkey_handler_test.go', 'backend/internal/handler/model_plaza_handler.go', 'backend/internal/handler/model_plaza_handler_test.go', 'backend/internal/service/passkey.go', 'backend/internal/service/passkey_test.go', 'backend/internal/service/channel_plaza.go', 'backend/internal/service/channel_plaza_test.go', 'backend/internal/service/openai_live.go', 'backend/internal/service/openai_live_lifecycle_test.go', 'backend/internal/repository/user_repo.go', 'backend/internal/repository/user_repo_integration_test.go', 'backend/internal/repository/api_key_repo.go', 'backend/internal/repository/api_key_repo_integration_test.go', 'backend/internal/securityaudit/prompt_config.go', 'backend/internal/securityaudit/prompt_config_test.go', 'backend/cmd/server/wire_gen.go')
$unexpected = $staged | Where-Object { $_ -notin $compatAllow -and $_ -notlike 'backend/ent/*' }
if ($unexpected) { throw "v0.1.168 compatibility allowlist mismatch: $($unexpected -join ', ')" }
git.exe commit -m "fix: preserve local behavior after v0.1.168"
```

Expected: migration 回归测试与任何实际兼容修复同属 v0.1.168 的兼容提交；若其它 RED 不存在，不创建空修复提交。

**Step 4: 严格提交 merge 后行为审查 ledger evidence**

在 ledger 说明 Passkey、model plaza、scoped update、prompt audit config 恢复、OpenAI Live 容错、前端权限/路由、quota reset 与 migration 的调用链结论，所有阶段 2 矩阵行必须非 `gap`，然后运行：

```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'v0.1.168 behavior-review evidence allowlist mismatch' }
git.exe commit -m "docs: record v0.1.168 behavior review"
```

Expected: migration 回归测试和任何 RED 修复仍在独立兼容提交；无 RED 时不创建空兼容提交。此 evidence commit 严格只含 build ledger，reviewer 通过后 controller 单独 check off Task 10/OpenSpec 3.2。

### Task 11: 封闭 v0.1.168 的本机门禁、migration 证据与阶段证据

- [ ] Task 11: 封闭 v0.1.168 的本机门禁、migration 证据与阶段证据

**映射 OpenSpec:** 3.3

**Files:**
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: v0.1.168 merge、升级 migration 测试与兼容提交。
- Produces: 迁移在新库/升级库上的明确通过或 `unverified` 风险，以及可进入 v0.1.169 的封闭阶段。

**Step 1: 运行 v0.1.168 full gate**

Run from repository root:
```powershell
make test
make "VERSION=0.1.165.4" "SHELL=D:/scoop/shims/bash.exe" build
make -C backend generate
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git.exe diff --check
git.exe diff --cached --check
git diff --name-only --diff-filter=U
$conflictPattern = '^(<<<<<<< .+|\|\|\|\|\|\|\| .+|=======|>>>>>>> .+)$'
git grep -n -I -E $conflictPattern -- ':!docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md'
$conflictExit = $LASTEXITCODE
if ($conflictExit -eq 0) { throw 'tracked conflict markers found' }
if ($conflictExit -ne 1) { throw "conflict marker scan failed: $conflictExit" }
git rev-parse HEAD:backend/migrations/191_subscription_quota_advance_receipts.sql
git rev-parse HEAD:backend/migrations/192_subscription_cache_invalidation_outbox.sql
```

Expected: 所有门禁通过；两个 191 文件和本地 192 文件不因 generate 或 merge 改名；conflict marker scan 的 no-match exit `1` 是 PASS，exit `0` 的真实 marker 或其它 grep 错误均 BLOCK；最后两个命令精确输出两个固定 base-ref blob OID。

**Step 2: Docker 可用时验证新库和升级库，逐项匹配 PASS**

Run:
```powershell
$preflightLog = Join-Path $env:TEMP 'sub2api-v0168-docker-preflight.log'
$dockerCommand = Get-Command docker -ErrorAction SilentlyContinue
if ($null -eq $dockerCommand) { 'docker_command=unavailable' | Set-Content -LiteralPath $preflightLog; $preflightAvailable = $false } else { & $dockerCommand.Path version --format '{{.Server.Version}}' 2>&1 | Tee-Object -FilePath $preflightLog; $preflightExitCode = $LASTEXITCODE; "exit_code=$preflightExitCode" | Add-Content -LiteralPath $preflightLog; $preflightAvailable = ($preflightExitCode -eq 0) }
```

When available, run from `backend`:
```powershell
$targets = @('TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate', 'TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade', 'TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails', 'TestUserRepoSuite', 'TestSubscriptionCacheInvalidationMigration_RawRerunIsIdempotent')
$log = Join-Path $env:TEMP 'sub2api-v0168-integration.log'
go test -tags=integration -v -count=1 -run '^(TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate|TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade|TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails|TestUserRepoSuite|TestSubscriptionCacheInvalidationMigration_RawRerunIsIdempotent)$' ./internal/repository 2>&1 | Tee-Object -FilePath $log
$exitCode = $LASTEXITCODE
"exit_code=$exitCode" | Add-Content -LiteralPath $log
if ($exitCode -ne 0) { throw "integration failed; retain $log and classify the root cause before choosing BLOCK or unverified" }
foreach ($target in $targets) { $pattern = '^--- PASS: ' + [regex]::Escape($target) + ' \('; if (-not (Select-String -Path $log -Pattern $pattern -Quiet)) { throw "missing top-level PASS: $target" } }
```

Expected: 仅在 `$preflightAvailable` 为 true 时运行 integration。新库幂等、基线升级后新增 Passkey、双方 191/local 192 checksum、quota receipt rollback 和 repository 行为均有锚定顶级 PASS。命令不存在或 preflight 失败直接记录 `unverified`；preflight 成功后的失败必须先按 systematic debugging 分类，只有 Docker/Testcontainers 环境不可用才能记录升级库路径为 `unverified`，静态/空库测试不能替代。

**Step 3: 提交阶段 2 证据并验证清洁度**

Run:
```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'v0.1.168 evidence allowlist mismatch' }
git.exe commit -m "docs: close v0.1.168 merge stage"
$selectionStatus = '?? .comet/current-change.json'
$index = @(git diff --cached --name-only)
if ($index) { throw "v0.1.168 evidence left staged paths: $($index -join ', ')" }
$status = @(git status --short --untracked-files=all)
$unexpected = $status | Where-Object { $_ -ne $selectionStatus }
if ($unexpected) { throw "v0.1.168 evidence left dirty paths: $($unexpected -join '; ')" }
$status
```

Expected: implementer 提交严格只含 ledger；最终 index 为空，status 过滤实际存在的 `?? .comet/current-change.json` 后为空，无未归属生成物或未解决索引项。reviewer 通过后 controller 单独 check off Task 11/OpenSpec 3.3。

### Task 12: 在未提交状态合入 v0.1.169 并完成阻塞审查

- [ ] Task 12: 在未提交状态合入 v0.1.169 并完成阻塞审查

**映射 OpenSpec:** 4.1

**Files:**
- Modify: `backend/internal/service/upstream_path_guard.go`、`backend/internal/service/gemini_upstream_url.go`、`backend/internal/service/openai_proxy_stream_circuit.go`、pricing/count_tokens/release/deploy 文件以及 `v0.1.168..v0.1.169` 的实际 changed files。
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: 从 `$executionBase` 经 v0.1.166/v0.1.168 阶段推进的封闭 HEAD 和 `v0.1.169^{}`。
- Produces: 第二父为 `26d894ef4f50645a4bf1030e378ac892f17d0223` 的 merge commit 与独立 merge-review ledger evidence；所有客户端可控 URL path 入口受闭集护栏保护。

**Step 1: 启动唯一允许的第三段 merge**

Run:
```powershell
$selectionStatus = '?? .comet/current-change.json'
$index = @(git diff --cached --name-only)
if ($index) { throw "v0.1.169 merge blocked by staged paths: $($index -join ', ')" }
$status = @(git status --short --untracked-files=all)
$unexpected = $status | Where-Object { $_ -ne $selectionStatus }
if ($unexpected) { throw "v0.1.169 merge blocked by dirty paths: $($unexpected -join '; ')" }
git.exe merge --no-ff --no-commit v0.1.169
$conflictPaths = @(git diff --name-only --diff-filter=U)
$conflictPaths
```

Expected: merge 前 `$index` 必须为空，`$status` 只能为空或精确为 `?? .comet/current-change.json`；其它任何用户路径均阻塞并要求 Comet 重建隔离位置。merge 停在未提交状态；实际 `$conflictPaths` 逐文件进入六分类台账。

**Step 2: 在 merge commit 前执行 GHSA、gateway 与资源阻塞审查**

审查所有客户端可控片段进入上游 URL 拼接的路径：Responses 的 `/v1/responses`、`/responses`、`/backend-api/codex/responses`，以及 Gemini native/compat 模型名。确认 `sanitizedUpstreamPathSuffix` 和 `validateUpstreamPathSegment` 在 URL 构造/请求发送前生效，gateway route、compat bridge、prompt audit、body replay、scheduler 不形成绕过。

同时审查 `openai_proxy_stream_circuit.go` 的 fail-open、Qwen3Guard 辅助字段、`gateway_count_tokens.go`、pricing 服务/资源、token refresh、release fallback 与 `no-new-privileges` 部署配置。逐项解决实际 `$conflictPaths` 后运行 `if ($conflictPaths.Count -gt 0) { git add -- $conflictPaths }` 和 `git diff --name-only --diff-filter=U`；后者必须无输出。阻塞任何漏洞输入可达上游、合法路径被改写、proxy circuit 变为 fail-closed、或本地审计/调度被绕过的状态。

**Step 3: 由工具生成依赖和生成输出**

Run from `backend`:
```powershell
go mod tidy
make generate
```

Run from repository root when manifest 改变:
```powershell
pnpm --dir frontend install --lockfile-only --frozen-lockfile=false
```

Expected: 所有生成物归入这次 merge；不能手工修改 Go checksum、Ent/Wire 或 lockfile。

**Step 4: 创建 v0.1.169 merge commit 并核验第二父**

Run:
```powershell
git add backend/internal/service/upstream_path_guard.go backend/internal/service/gemini_upstream_url.go backend/internal/service/openai_proxy_stream_circuit.go backend/resources/model-pricing/model_prices_and_context_window.json backend/ent backend/cmd/server/wire_gen.go backend/go.mod backend/go.sum frontend/pnpm-lock.yaml
git diff --name-only --diff-filter=U
git diff --cached --check
$staged = @(git diff --cached --name-only)
$staged
$mergePlanningPaths = @('.comet/current-change.json', 'docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md', 'docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-verify.md', 'docs/superpowers/plans/2026-08-02-staged-merge-upstream-v0-1-169.md', 'docs/superpowers/specs/2026-08-02-staged-merge-upstream-v0-1-169-design.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/proposal.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/design.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/specs/upstream-release-sync/spec.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/tasks.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/subagent-progress.md')
$forbidden = $staged | Where-Object { $_ -in $mergePlanningPaths }
if ($forbidden) { throw "forbidden paths staged for merge: $($forbidden -join ', ')" }
git.exe commit --no-edit
git rev-parse HEAD^2
```

Expected: 所有实际 `$conflictPaths` 已逐项暂存，unmerged/cached whitespace 检查通过；`$staged` 不含 runtime selection、ledger、计划、Design Doc 或任一 OpenSpec 规划产物，且所有工具产生的 manifest/checksum/生成输出在本 merge commit。第二父精确为 `26d894ef4f50645a4bf1030e378ac892f17d0223`；VERSION 仍为 `0.1.165.4`。

**Step 5: 严格提交 v0.1.169 merge-review ledger evidence**

在 ledger 记录实际 `$conflictPaths`、GHSA/gateway/资源阻塞审查和合并后的第二父，然后运行：

```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'v0.1.169 merge-review evidence allowlist mismatch' }
git.exe commit -m "docs: record v0.1.169 merge review"
```

Expected: 此 evidence commit 严格只含 build ledger，保持 preceding merge commit 纯 merge；reviewer 通过后 controller 单独 check off Task 12/OpenSpec 4.1。

### Task 13: 对 v0.1.169 完成 GHSA 负向矩阵与行为审查

- [ ] Task 13: 对 v0.1.169 完成 GHSA 负向矩阵与行为审查

**映射 OpenSpec:** 4.2

**Files:**
- Modify when RED exists: `backend/internal/service/upstream_path_guard.go`, `backend/internal/service/upstream_path_guard_test.go`, `backend/internal/service/gemini_upstream_url.go`, `backend/internal/service/gemini_upstream_url_test.go`, `backend/internal/server/routes/gateway_test.go` 和最小相关 gateway/compat 调用方。
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: `sanitizedUpstreamPathSuffix`、`validateUpstreamPathSegment`、`buildGeminiAIStudioModelActionURL`、三类 Responses 路由。
- Produces: GHSA 负向输入在边缘被拒绝且从不参与 upstream suffix 拼接/请求，合法输入保持原样。

**Step 1: 写入并运行精确 GHSA 负向矩阵**

在 `upstream_path_guard_test.go`、`gateway_test.go` 和 `gemini_upstream_url_test.go` 的表驱动测试中覆盖下表。编码 path traversal 和其他非法 path suffix 在三类 Responses 入口都断言 `404`、`Unsupported responses subpath`、上游请求计数为零且不被误判为 `compact`。helper 原始 `?/#` 输入与真实 HTTP query 是不同边界。

| 输入类别 | 必测 raw suffix/model 值 | 期望 |
| --- | --- | --- |
| 原始 traversal | `../models`、`../../models`、`...` | 拒绝；纯点 segment 不合法。 |
| 百分号编码 traversal | `%2e%2e/models`、`%2E%2E%2Fmodels`、`compact%2f..%2fmodels` | 解析后拒绝；不拼上游 URL。 |
| 反斜杠 | `compact\\detail`、`%5c..%5cmodels` | 拒绝。 |
| helper 原始 query/fragment 字符 | 直接传给 `sanitizedUpstreamPathSuffix` 的 `compact?next=%2fmodels`、`compact#fragment` | 拒绝；helper 的 raw suffix 不得接受 `?` 或 `#`。 |
| 真实 HTTP query | `/v1/responses/compact?next=%2fmodels`、`/responses/compact?next=%2fmodels`、`/backend-api/codex/responses/compact?next=%2fmodels` | 放行既定 `compact` 路由；query 不属于 path suffix，不参与 suffix 拼接，也不得导致 404。 |
| 空 segment | `//double`、`/compact//detail` | 拒绝。 |
| 长度/段数 | 129-byte 单 segment、9 个非空 segment | 拒绝。 |
| 非法字符 | 含 NUL/control、非 ASCII、空格、`:`、`;`、`@`、`%` 的 segment/model | 拒绝。 |
| Gemini native/compat | 含上述非法 model 的 native path 与 compat request body | 构造 URL 前返回错误，HTTP stub 的请求数为零。 |
| 合法回归 | 根路径、尾斜杠、`compact`、合法 response id、`gemini-2.5-pro` 与三个允许 action | 保持原样并走既定处理。 |

Run from `backend`:
```powershell
go test -count=1 ./internal/service -run '^(TestSanitizedUpstreamPathSuffixRejectsNonConformingSegments|TestSanitizedUpstreamPathSuffixEnforcesBounds|TestOpenAIResponsesRequestPathSuffixRejectsNonConformingSubpaths|TestAppendOpenAIResponsesRequestPathSuffixRefusesUnsafeSuffix|TestBuildGeminiAIStudioModelActionURL|TestBuildGeminiAIStudioModelActionURLRejectsNonConformingModel|TestOpenAIProxyStreamCircuitThresholdTTLAndSuccessReset|TestOpenAIProxyStreamCircuitDisabled)$'
go test -count=1 ./internal/server/routes -run '^(TestGatewayRoutesResponsesSubpathRejectsNonConformingSubpaths|TestGatewayRoutesOpenAIResponsesCompactPathIsRegistered)$'
go test -count=1 ./internal/securityaudit -run '^(TestParseQwen3GuardStrictAndPolicy|TestParseQwen3GuardIgnoresAuxiliaryResponseFields)$'
```

Expected: 所有负向项拒绝于边缘；合法项通过；proxy circuit 保持 fail-open 语义；Qwen3Guard 辅助字段不改变既定策略。

**Step 2: 验证 v0.1.169 新增 deploy 测试脚本**

Run after the v0.1.169 merge, using the bash verified in Task 1:
```powershell
$bash = 'D:/scoop/shims/bash.exe'
$scripts = @('deploy/tests/docker-compose-security-test.sh', 'deploy/tests/docker-runtime-resources-test.sh')
foreach ($script in $scripts) { if (-not (Test-Path -LiteralPath $script)) { throw "missing v0.1.169 deploy test: $script" }; & $bash $script; if ($LASTEXITCODE -ne 0) { throw "deploy test failed: $script" } }
```

Expected: 两个脚本都在 v0.1.169 merge 后存在并由已验证 bash 成功执行；它们不作为阶段 0 文件或测试目标。失败阻塞阶段，不执行部署。

**Step 3: 审查 v0.1.169 与本地能力的调用链**

将 path guard、Gemini compat、Responses route、prompt audit、OpenAI gateway、scheduler、body replay、count_tokens、pricing、token refresh 和 release fallback 逐项映射到 canonical matrix。确认安全护栏没有因本地分支绕过，pricing 与图片/usage 倍率没有退化，部署安全文件只随上游树合并且未执行部署。

**Step 4: 对 RED 创建独立兼容提交**

先使相关单元/路由测试 RED，再修改最小 path guard、Gemini URL builder 或调用方代码，重新运行 Step 1。

Run:
```powershell
git add backend/internal/service/upstream_path_guard.go backend/internal/service/upstream_path_guard_test.go backend/internal/service/gemini_upstream_url.go backend/internal/service/gemini_upstream_url_test.go backend/internal/server/routes/gateway_test.go backend/ent backend/cmd/server/wire_gen.go
$staged = @(git diff --cached --name-only)
$staged
$compatAllow = @('backend/internal/service/upstream_path_guard.go', 'backend/internal/service/upstream_path_guard_test.go', 'backend/internal/service/gemini_upstream_url.go', 'backend/internal/service/gemini_upstream_url_test.go', 'backend/internal/server/routes/gateway_test.go', 'backend/cmd/server/wire_gen.go')
$unexpected = $staged | Where-Object { $_ -notin $compatAllow -and $_ -notlike 'backend/ent/*' }
if ($unexpected) { throw "v0.1.169 compatibility allowlist mismatch: $($unexpected -join ', ')" }
git.exe commit -m "fix: preserve local behavior after v0.1.169"
```

Expected: 仅在 RED 存在时创建该兼容提交；测试和修复在同一提交，生成输出随源提交。

**Step 5: 严格提交 merge 后行为审查 ledger evidence**

在 ledger 逐项写入 GHSA 矩阵结果、路径入口/调用链、proxy circuit、Qwen3Guard、count_tokens、pricing、token refresh、release fallback 的证据，并确认阶段 3 `gap=0`，然后运行：

```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'v0.1.169 behavior-review evidence allowlist mismatch' }
git.exe commit -m "docs: record v0.1.169 behavior review"
```

Expected: 兼容代码提交保持独立，无 RED 时不创建空兼容提交；此 evidence commit 严格只含 build ledger，reviewer 通过后 controller 单独 check off Task 13/OpenSpec 4.2。

### Task 14: 封闭 v0.1.169 的本机门禁与阶段证据

- [ ] Task 14: 封闭 v0.1.169 的本机门禁与阶段证据

**映射 OpenSpec:** 4.3

**Files:**
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: v0.1.169 merge、GHSA 负向矩阵和兼容提交。
- Produces: 可进行最终版本更新的封闭阶段 3 证据。

**Step 1: 运行 v0.1.169 full gate**

Run from repository root:
```powershell
make test
make "VERSION=0.1.165.4" "SHELL=D:/scoop/shims/bash.exe" build
make -C backend generate
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git.exe diff --check
git.exe diff --cached --check
git diff --name-only --diff-filter=U
$conflictPattern = '^(<<<<<<< .+|\|\|\|\|\|\|\| .+|=======|>>>>>>> .+)$'
git grep -n -I -E $conflictPattern -- ':!docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md'
$conflictExit = $LASTEXITCODE
if ($conflictExit -eq 0) { throw 'tracked conflict markers found' }
if ($conflictExit -ne 1) { throw "conflict marker scan failed: $conflictExit" }
git rev-parse HEAD:backend/migrations/191_subscription_quota_advance_receipts.sql
git rev-parse HEAD:backend/migrations/192_subscription_cache_invalidation_outbox.sql
```

Expected: full gate 通过，VERSION 仍为 `0.1.165.4`，无生成 diff、whitespace 或 unmerged；conflict marker scan 的 no-match exit `1` 是 PASS，exit `0` 的真实 marker 或其它 grep 错误均 BLOCK；最后两个命令精确输出 `c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6` 和 `502ecec1caf9f76e022c2e83acf3707190539301`。

**Step 2: 运行 v0.1.169 Docker 判定与 integration**

Run:
```powershell
$preflightLog = Join-Path $env:TEMP 'sub2api-v0169-docker-preflight.log'
$dockerCommand = Get-Command docker -ErrorAction SilentlyContinue
if ($null -eq $dockerCommand) { 'docker_command=unavailable' | Set-Content -LiteralPath $preflightLog; $preflightAvailable = $false } else { & $dockerCommand.Path version --format '{{.Server.Version}}' 2>&1 | Tee-Object -FilePath $preflightLog; $preflightExitCode = $LASTEXITCODE; "exit_code=$preflightExitCode" | Add-Content -LiteralPath $preflightLog; $preflightAvailable = ($preflightExitCode -eq 0) }
```

When available, run from `backend`:
```powershell
$targets = @('TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate', 'TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade', 'TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails', 'TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack')
$log = Join-Path $env:TEMP 'sub2api-v0169-integration.log'
go test -tags=integration -v -count=1 -run '^(TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate|TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade|TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails|TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack)$' ./internal/repository 2>&1 | Tee-Object -FilePath $log
$exitCode = $LASTEXITCODE
"exit_code=$exitCode" | Add-Content -LiteralPath $log
if ($exitCode -ne 0) { throw "integration failed; retain $log and classify the root cause before choosing BLOCK or unverified" }
foreach ($target in $targets) { $pattern = '^--- PASS: ' + [regex]::Escape($target) + ' \('; if (-not (Select-String -Path $log -Pattern $pattern -Quiet)) { throw "missing top-level PASS: $target" } }
```

Expected: 仅在 `$preflightAvailable` 为 true 时运行 integration，每个目标以锚定顶级 PASS 匹配。命令不存在或 preflight 失败直接记录 `unverified`；preflight 成功后的失败先按 systematic debugging 分类，代码/断言失败 BLOCK，只有 Docker/Testcontainers 环境不可用时记录环境根因和未验证 migration/repository 契约。

**Step 3: 提交阶段 3 证据并验证清洁度**

Run:
```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'v0.1.169 evidence allowlist mismatch' }
git.exe commit -m "docs: close v0.1.169 merge stage"
$selectionStatus = '?? .comet/current-change.json'
$index = @(git diff --cached --name-only)
if ($index) { throw "v0.1.169 evidence left staged paths: $($index -join ', ')" }
$status = @(git status --short --untracked-files=all)
$unexpected = $status | Where-Object { $_ -ne $selectionStatus }
if ($unexpected) { throw "v0.1.169 evidence left dirty paths: $($unexpected -join '; ')" }
$status
```

Expected: implementer 提交严格只含 ledger；最终 index 为空，status 过滤实际存在的 `?? .comet/current-change.json` 后为空，阶段位置没有新增未归因路径。reviewer 通过后 controller 单独 check off Task 14/OpenSpec 4.3。

### Task 15: 一次更新最终版本

- [ ] Task 15: 一次更新最终版本

**映射 OpenSpec:** 5.1

**Files:**
- Modify: `backend/cmd/server/VERSION`

**Interfaces:**
- Consumes: 三个封闭 merge 阶段和当前 `VERSION=0.1.165.4`。
- Produces: 唯一最终版本 `0.1.169.1`。

**Step 1: 确认三个阶段已封闭**

Run:
```powershell
git show HEAD:backend/cmd/server/VERSION
git log --merges --first-parent --format='%H %P' e9a0e4aa53b5d9d5f5c84986cfadd8098dc8e4f3..HEAD
```

Expected: VERSION 仍为 `0.1.165.4`；merge log 从 immutable source base 而非 `$executionBase` 开始，first-parent 区间存在三个 merge 行且各阶段 ledger 结论为封闭。

**Step 2: 修改并提交唯一的版本变更**

将 `backend/cmd/server/VERSION` 的完整内容改为 `0.1.169.1`，然后运行：
```powershell
git add backend/cmd/server/VERSION
$staged = @(git diff --cached --name-only)
$staged
$versionAllow = @('backend/cmd/server/VERSION')
if ((Compare-Object -ReferenceObject $versionAllow -DifferenceObject $staged) -or $staged.Count -ne 1) { throw 'version commit allowlist mismatch' }
git.exe commit -m "chore: bump version to 0.1.169.1"
git show HEAD:backend/cmd/server/VERSION
```

Expected: 输出精确为 `0.1.169.1`；提交严格只包含 `backend/cmd/server/VERSION`，不包含生成物、文档或 tag 操作。

reviewer 通过后 controller 单独 check off Task 15/OpenSpec 5.1。

### Task 16: 在最终 source HEAD 重跑完整本机验证

- [ ] Task 16: 在最终 source HEAD 重跑完整本机验证

**映射 OpenSpec:** 5.2

**Files:**
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: `VERSION=0.1.169.1` 的最终 source HEAD。
- Produces: 最终自动门禁、Images 精确契约和前后端能力的可审计结果。

**Step 1: 重跑全部命中的能力聚焦测试**

Run from `backend`:
```powershell
go test -count=1 ./internal/service -run '^(TestLayered_GroupedAccountPassesDBFreshRecheck|TestLayered_SessionStickyPreservesGrokBinding|TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount|TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot|TestCalculateQuotaCycleAdvance_ResetsOnlySingleExhaustedWindow|TestAdminResetQuota_UsesCommittedResetVersionForCacheInvalidation|TestSanitizedUpstreamPathSuffixRejectsNonConformingSegments|TestBuildGeminiAIStudioModelActionURLRejectsNonConformingModel|TestOpenAIProxyStreamCircuitThresholdTTLAndSuccessReset)$'
go test -count=1 ./internal/handler -run '^(TestOpenAIImages_UnifiedAuditRunsLegacyOnce|TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease|TestOpenAIImages_SecurityAuditUsesFrozenPayloadBeforeRelease|TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce|TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot)$'
go test -count=1 ./internal/server/routes -run '^(TestEveryGatewayPOSTRouteIsClassifiedForPromptAuditCoverage|TestResponsesWebSocketHasFirstAndSubsequentTurnPromptGates|TestGatewayRoutesResponsesSubpathRejectsNonConformingSubpaths)$'
```

Run from repository root:
```powershell
pnpm --dir frontend exec vitest run src/utils/__tests__/subscriptionQuota.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/UsageView.spec.ts src/components/channels/__tests__/AvailableChannelsTable.spec.ts
```

Expected: 上述测试逐项 PASS，且 Images 六项契约都由直接测试证据支撑。

**Step 2: 重跑 final full gate 和静态检查**

Run from repository root:
```powershell
make test
make "VERSION=0.1.169.1" "SHELL=D:/scoop/shims/bash.exe" build
make -C backend generate
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git.exe diff --check
git.exe diff --cached --check
git diff --name-only --diff-filter=U
$conflictPattern = '^(<<<<<<< .+|\|\|\|\|\|\|\| .+|=======|>>>>>>> .+)$'
git grep -n -I -E $conflictPattern -- ':!docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md'
$conflictExit = $LASTEXITCODE
if ($conflictExit -eq 0) { throw 'tracked conflict markers found' }
if ($conflictExit -ne 1) { throw "conflict marker scan failed: $conflictExit" }
git rev-parse HEAD:backend/migrations/191_subscription_quota_advance_receipts.sql
git rev-parse HEAD:backend/migrations/192_subscription_cache_invalidation_outbox.sql
```

Expected: 后端默认/unit/lint、前端 ESLint/Vitest/typecheck/build 均经 `make test`/已验证的 `VERSION=0.1.169.1` Windows build 命令通过；两轮生成稳定；静态检查无输出；conflict marker scan 的 no-match exit `1` 是 PASS，exit `0` 的真实 marker 或其它 grep 错误均 BLOCK；最后两个命令精确输出 `c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6` 和 `502ecec1caf9f76e022c2e83acf3707190539301`。

**Step 3: 提交最终自动验证证据**

Run:
```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'final gate evidence allowlist mismatch' }
git.exe commit -m "docs: record final local quality gates"
$selectionStatus = '?? .comet/current-change.json'
$index = @(git diff --cached --name-only)
if ($index) { throw "final gate evidence left staged paths: $($index -join ', ')" }
$status = @(git status --short --untracked-files=all)
$unexpected = $status | Where-Object { $_ -ne $selectionStatus }
if ($unexpected) { throw "final gate evidence left dirty paths: $($unexpected -join '; ')" }
$status
```

Expected: ledger 明确关联最终 source HEAD 和命令退出码；提交后 index 为空，status 过滤实际存在的 `?? .comet/current-change.json` 后为空。reviewer 通过后 controller 单独 check off Task 16/OpenSpec 5.2。

### Task 17: 验证拓扑、migration 保留和最终 integration 状态

- [ ] Task 17: 验证拓扑、migration 保留和最终 integration 状态

**映射 OpenSpec:** 5.3

**Files:**
- Modify: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`

**Interfaces:**
- Consumes: 最终 HEAD、三个 tag peeled SHA、三个完整 migration filename。
- Produces: ancestor/second-parent/migration/integration 的最终可判定结论。

**Step 1: 验证三个 tag 祖先和 merge 第二父**

Run:
```powershell
git merge-base --is-ancestor v0.1.166 HEAD
git merge-base --is-ancestor v0.1.168 HEAD
git merge-base --is-ancestor v0.1.169 HEAD
git log --merges --first-parent --format='%H %P' e9a0e4aa53b5d9d5f5c84986cfadd8098dc8e4f3..HEAD
```

Expected: 三个 ancestor 命令退出 0；merge log 从 immutable source base 而非 `$executionBase` 开始，恰有三个 merge commit，且其第二父集合精确包含 `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8`、`99c8e4bf7564823bafbab369acab6539e734c1bb`、`26d894ef4f50645a4bf1030e378ac892f17d0223`。

**Step 2: 验证 migration filename、排序和 checksum 身份**

Run:
```powershell
git ls-files --error-unmatch backend/migrations/191_passkey_credentials.sql backend/migrations/191_subscription_quota_advance_receipts.sql backend/migrations/192_subscription_cache_invalidation_outbox.sql
git diff --name-status e9a0e4aa53b5d9d5f5c84986cfadd8098dc8e4f3..HEAD -- backend/migrations/191_passkey_credentials.sql backend/migrations/191_subscription_quota_advance_receipts.sql backend/migrations/192_subscription_cache_invalidation_outbox.sql
git rev-parse HEAD:backend/migrations/191_subscription_quota_advance_receipts.sql
git rev-parse HEAD:backend/migrations/192_subscription_cache_invalidation_outbox.sql
```

Expected: 三个完整 filename 都存在；本地两个历史文件没有 rename，Passkey 文件作为上游新增保留；最后两个命令精确输出 `c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6` 和 `502ecec1caf9f76e022c2e83acf3707190539301`。升级 integration 的 filtered FS 与这两个 blob identity 是互补证据，前者不能替代后者。Docker 已可用时，Task 11/14 的升级测试必须出现顶级 PASS；不可用时该项保持 `unverified`，不得改写为成功。

**Step 3: 执行最终 Docker 判定**

Run:
```powershell
$preflightLog = Join-Path $env:TEMP 'sub2api-final-docker-preflight.log'
$dockerCommand = Get-Command docker -ErrorAction SilentlyContinue
if ($null -eq $dockerCommand) { 'docker_command=unavailable' | Set-Content -LiteralPath $preflightLog; $preflightAvailable = $false } else { & $dockerCommand.Path version --format '{{.Server.Version}}' 2>&1 | Tee-Object -FilePath $preflightLog; $preflightExitCode = $LASTEXITCODE; "exit_code=$preflightExitCode" | Add-Content -LiteralPath $preflightLog; $preflightAvailable = ($preflightExitCode -eq 0) }
```

When available, run from `backend`:
```powershell
$targets = @('TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate', 'TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade', 'TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce', 'TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails', 'TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack')
$log = Join-Path $env:TEMP 'sub2api-final-integration.log'
go test -tags=integration -v -count=1 -run '^(TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate|TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade|TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce|TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails|TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack)$' ./internal/repository 2>&1 | Tee-Object -FilePath $log
$exitCode = $LASTEXITCODE
"exit_code=$exitCode" | Add-Content -LiteralPath $log
if ($exitCode -ne 0) { throw "integration failed; retain $log and classify the root cause before choosing BLOCK or unverified" }
foreach ($target in $targets) { $pattern = '^--- PASS: ' + [regex]::Escape($target) + ' \('; if (-not (Select-String -Path $log -Pattern $pattern -Quiet)) { throw "missing top-level PASS: $target" } }
```

Expected: 仅在 `$preflightAvailable` 为 true 时运行 integration；可用 Docker 环境中五项以锚定顶级 PASS 匹配。命令不存在或 preflight 失败直接记为 `unverified`；preflight 成功后的失败先按 systematic debugging 分类，代码/断言失败 BLOCK，只有 Docker/Testcontainers 环境不可用时才将原始错误、完整日志、退出码和“新库/升级库 PostgreSQL 路径未验证”写入 ledger。

**Step 4: 提交拓扑与 migration 证据**

Run:
```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md') -DifferenceObject $staged) { throw 'topology evidence allowlist mismatch' }
git.exe commit -m "docs: verify upstream merge topology"
$selectionStatus = '?? .comet/current-change.json'
$index = @(git diff --cached --name-only)
if ($index) { throw "topology evidence left staged paths: $($index -join ', ')" }
$status = @(git status --short --untracked-files=all)
$unexpected = $status | Where-Object { $_ -ne $selectionStatus }
if ($unexpected) { throw "topology evidence left dirty paths: $($unexpected -join '; ')" }
$status
```

Expected: 该提交只更新最终拓扑、migration 和 integration 证据；提交后 index 为空，status 过滤实际存在的 `?? .comet/current-change.json` 后为空。reviewer 通过后 controller 单独 check off Task 17/OpenSpec 5.3。

### Task 18: 完成本地能力专项 review 与最终验证报告

- [ ] Task 18: 完成本地能力专项 review 与最终验证报告

**映射 OpenSpec:** 5.4

**Files:**
- Create: `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-verify.md`

**Interfaces:**
- Consumes: build ledger、最终测试结果、tag 拓扑和 Docker 判定。
- Produces: 可供 OpenSpec/Comet Verify 使用的最终 report，明确通过项和唯一允许的 `unverified` 风险。

**Step 1: 逐项完成本地能力专项 review**

在 verify report 为 canonical 矩阵每一行写入测试、调用链或生成物证据，并显式复核 scheduler、各平台 sticky、fallback/WaitPlan、DB recheck、privacy、image capability、异步图片/对象存储、图片输入计费、上游倍率、session binding/step-up、runtime setting 热更新、gateway 透传、body replay/cleanup、用户资源控制、分组复制、批量限额、quota cycle reset、前端本地功能、版本依赖、生成代码和 migrations。

Images 段必须单列六项精确契约。GHSA 段必须列出三类 Responses 入口、所有负向类别、Gemini native/compat 的“构造上游 URL 前拒绝且请求数为零”证据。Docker 不可用时，只列出 Task 5/8/11/14/17 已记录的 `unverified` 契约和原因。

**Step 2: 执行 OpenSpec 严格验证**

Run:
```powershell
openspec validate staged-merge-upstream-v0-1-169 --strict
```

Expected: 命令退出 0，并只读取 controller 已逐项勾选的 `tasks.md`。若失败，只可修改实际需要的下列路径：`docs/openspec/changes/staged-merge-upstream-v0-1-169/proposal.md`、`docs/openspec/changes/staged-merge-upstream-v0-1-169/design.md`、`docs/openspec/changes/staged-merge-upstream-v0-1-169/specs/upstream-release-sync/spec.md`、`docs/superpowers/specs/2026-08-02-staged-merge-upstream-v0-1-169-design.md`、`docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-verify.md`；不得修改 `tasks.md` 或已验证源码来回避文档验证。

**Step 3: 严格验证失败时创建独立 docs 修复提交**

仅在 Step 2 非零且修复后 strict validate 变为 0 时执行。列出实际修改的路径，确认全部属于 Step 2 列出的五个路径，然后运行：
```powershell
$docsAllow = @('docs/openspec/changes/staged-merge-upstream-v0-1-169/proposal.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/design.md', 'docs/openspec/changes/staged-merge-upstream-v0-1-169/specs/upstream-release-sync/spec.md', 'docs/superpowers/specs/2026-08-02-staged-merge-upstream-v0-1-169-design.md', 'docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-verify.md')
$changed = @(git diff --name-only)
$changed
$unexpected = $changed | Where-Object { $_ -notin $docsAllow }
if ($unexpected -or $changed.Count -eq 0) { throw "docs repair allowlist mismatch: $($unexpected -join ', ')" }
git add -f -- $changed
$staged = @(git diff --cached --name-only)
$staged
$unexpected = $staged | Where-Object { $_ -notin $docsAllow }
if ($unexpected) { throw "staged docs repair allowlist mismatch: $($unexpected -join ', ')" }
git.exe commit -m "docs: fix staged merge spec validation"
```

Expected: strict validate 首次成功时不创建此提交；失败修复时，该提交只包含实际需要的明确 docs 路径。

**Step 4: 提交最终报告并检查清洁度**

报告明确写出“未推送、未发版、未部署、未操作服务器、未操作 Nginx，生产临时盾仍保留”。implementer 不修改或暂存 `tasks.md`。

Run:
```powershell
git add -f -- docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-verify.md
$staged = @(git diff --cached --name-only)
$staged
if (Compare-Object -ReferenceObject @('docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-verify.md') -DifferenceObject $staged) { throw 'final verify evidence allowlist mismatch' }
git.exe commit -m "docs: verify staged upstream merge"
$selectionStatus = '?? .comet/current-change.json'
$index = @(git diff --cached --name-only)
if ($index) { throw "final verify evidence left staged paths: $($index -join ', ')" }
$status = @(git status --short --untracked-files=all)
$unexpected = $status | Where-Object { $_ -ne $selectionStatus }
if ($unexpected) { throw "final verify evidence left dirty paths: $($unexpected -join '; ')" }
$status
```

Expected: implementer commit 严格只含 verify report；最终 index 为空，status 过滤实际存在的 `?? .comet/current-change.json` 后为空；不执行 push、tag、release、deploy、镜像构建、服务器或 Nginx 命令。reviewer 通过后 controller 单独 check off Task 18/OpenSpec 5.4；至此 18 个顶层 Task 和 1.1-1.5、2.1-2.3、3.1-3.3、4.1-4.3、5.1-5.4 均已通过各自 docs-only checkoff commit 完成。

## 计划自审

| 检查项 | 结论 |
| --- | --- |
| SDD coverage | 18 个 `### Task N` 均有唯一顶层 `- [ ] Task N` checkbox，Plan 中仅此 18 个 checkbox；内部 Step 为非 checkbox 编号。implementer/reviewer 不修改 Plan；仅 controller 在 reviewer 通过后将当前 Task 改为 `[x]`、同步唯一 OpenSpec 项并写入 progress。 |
| Progress/topology coverage | 通用 live-ID 平台在成功派发后持久 agent identity；本平台不存在“dispatch success 与 completion 之间”的 controller 窗口，因此在原子调用两侧持久 dispatch intent 和 returned result。恢复先查 report、Git 与宿主工具结果，不能确认即 BLOCKED 而不重派；已有 task ID 的 fix resume 保持 Comet thorough round。任何下一次派发和每次 merge clean gate 前最新 checkpoint 均已提交且 worktree clean。progress-only 与三路径 docs-only checkoff commits 均不改变最终 `git log --merges` 恰有三个 upstream merge 节点的判定。 |
| Spec coverage | OpenSpec 1.1-1.5、2.1-2.3、3.1-3.3、4.1-4.3、5.1-5.4 共 18 项均有一个同编号任务；Task 4 将 Images、outbox unit-tag、Wire、deferred rows-close lint、repaired stage-0 ledger 与 Step 9 首轮 review repair 分离提交。Step 9 的 P1 只允许 request-scoped `imagesModerationBody` hook、Images callsite 与 test-local blocking prompt engine，先以同请求 `expected/want 1, actual/got 0` RED 再 GREEN；P2 在不改写历史的前提下逐命令追加 evidence、明确 row 13 历史 supersede 和 12 个 generate 日志。Task 4 只有 fresh reviewer 2/2 与 tracked clean 后闭合；Task 5 只验证 row 10 quota/outbox/migration integration，row 13 deploy scripts 只由 v0.1.169 merge 后的 Task 13 执行。三段 no-ff/no-commit、merge 前阻塞审查、merge 后行为审查、17 冲突、能力矩阵、quota reset、迁移、Docker 和最终版本均有明确步骤；所有阶段/最终 conflict scan 均识别 diff3 marker，只有 no-match exit `1` 通过。 |
| Placeholder scan | 本计划没有未填内容、未命名测试或未定义接口占位；所有条件分支给出停止、RED、`unverified` 或 PASS 判定，且真实 HTTP query 与 helper raw 输入的 GHSA 断言明确分离。 |
| 双基线与类型/名称一致性 | frontmatter `base-ref` 始终是 immutable source base，`$executionBase` 仅为其 planning-only 后代；Task 1 使用当前 change OpenSpec 目录前缀加两个精确文档路径同时验证 source-to-execution tree diff 和全提交触及路径，且 source/execution VERSION 都为 `0.1.165.4`。`applyMigrationsFS`、quota 接口、路径护栏、Gemini URL builder、migration 文件名、两个 source-base blob OID、tag SHA、最终从 source base 开始的 merge range 和 integration 顶级测试名在任务间使用一致。 |
| 范围边界 | 计划只描述 Comet 已确认的实际隔离位置内的 merge、测试、文档和本地生成；明确排除 push、发布、部署、镜像、服务器、数据库运行态和 Nginx 操作。 |
