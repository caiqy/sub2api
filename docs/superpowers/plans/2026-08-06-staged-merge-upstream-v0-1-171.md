---
change: staged-merge-upstream-v0-1-171
design-doc: docs/superpowers/specs/2026-08-06-staged-merge-upstream-v0-1-171-design.md
base-ref: 16c07d8064b0b4604e9f47ef782e7d29534402d3
---

# 分段合并上游 v0.1.171 实施计划

> **执行约束：**由 Comet 在开始实施时选择隔离位置、分支/worktree、build_mode、tdd_mode 和 review_mode。本计划不创建或预设它们；每个任务完成后由执行编排层决定审查与勾选方式。

**目标：**按严格祖先顺序把 `v0.1.170` 和 `v0.1.171` 合为两个可审计的纯 merge 节点，保留本地能力和 migration identity，最终将版本一次更新为 `0.1.171.1`。

**实施结构：**先固定双基线、范围、能力矩阵和本机基线证据；每个 tag 先在未提交 merge 中完成必须的冲突融合和源驱动生成，再用独立能力簇提交修复可复现回归。每段必须关闭 `gap`、完成本机门禁和 Docker 条件门禁后，才能开始下一段；最终用拓扑、migration identity 和能力专项 review 闭合。

**技术栈：**Git merge、Go、Ent、Wire、PostgreSQL/Testcontainers、pnpm/Vitest、PowerShell、OpenSpec。

## 全局约束

- immutable source base 固定为 `16c07d8064b0b4604e9f47ef782e7d29534402d3`，其运行版本必须为 `0.1.169.3`。该提交是已归档 lint remediation 合入后的 `main`；执行位置的 `$executionBase` 可是仅含当前 change 产物和已审查基线保护测试的后代，source base 必须是其祖先。
- 两段只能依次合入 `v0.1.170@c043c24774228ba891ddf90d783aa6dc7d0855b5`、`v0.1.171@f0e7a9c7a23a7d02fb159b62fa809621eb0475a6`。不合入 `v0.1.171` 之后的 `upstream/main` 提交，发现更高正式 tag 时停止并回到 OpenSpec 更新范围。
- 每段唯一 merge 入口分别是 `git merge --no-ff --no-commit v0.1.170` 和 `git merge --no-ff --no-commit v0.1.171`。merge commit 只承载目标上游树和完成 merge 必需的冲突融合，第二父必须是该段固定 peeled SHA；后续语义修复不得混入 merge commit。
- 中间阶段的 `backend/cmd/server/VERSION` 固定为 `0.1.169.3`；两段均闭合后才在一个提交中改为 `0.1.171.1`。
- 所有暂存必须使用显式路径，禁止 `git add .`。`schema`/provider/manifest 源与相应 Ent/Wire、`go.sum` 或前端 lockfile 输出同一提交；不得手工编辑生成输出或 lockfile。
- migration identity 以完整文件名为准。最终必须保留 `191_passkey_credentials.sql`、`191_subscription_quota_advance_receipts.sql`、`192_subscription_cache_invalidation_outbox.sql`、`192_group_profit_control.sql`、`193_group_profit_control_auth_cache_invalidation.sql`，不得重命名或改写已发布 migration。
- 任何冲突若会改变用户可见语义而不能同时保留上游和本地契约，记录 ours/theirs/影响后立即停在当前未提交阶段，等待用户决定；不得预先选择 ours、theirs 或静默删除一方行为。
- Docker/Testcontainers 只在本机条件可用时执行。不可用或目标测试明确 `SKIP` 时只能记录为 `unverified`，列出原因和受影响契约；不得访问远程服务器补验。`exit 0`、包级 `ok`、`no tests to run` 均不构成 integration PASS。
- 不执行 push、tag、release、GitHub Actions、镜像构建/发布、deploy、服务器、数据库、Redis 或 Nginx 操作。
- 每个 Task 都可能在新的 PowerShell/subagent 会话中执行；不得依赖上一 Task 的变量、函数、当前目录或临时文件。每个 Task 开始时必须重新执行下方“统一检查命令”中的 layout/context/helper 初始化，再执行本 Task 使用的命令。
- 每个 Task 的实现、验证和所选 review gate 通过后，协调者同时把 plan 中唯一 Task checkbox 和映射的 OpenSpec checkbox 从 `[ ]` 改为 `[x]`，以 `git add -f` 显式暂存这两个文件并用 `docs: record SDD progress` 独立提交，然后分别运行 `comet state task-checkoff` 精确验证两段唯一任务文本。plan/tasks 不得混入 merge、能力簇、版本或证据提交。
- `.comet/current-change.json`、change 内 `.comet.yaml`、handoff 和 `subagent-progress.md` 属于 Comet runtime/归档状态，不得混入业务、merge 或进度提交；由 Comet 持续维护并在最终归档边界处理。
- 所有原生命令（Git、Go、make、pnpm、Docker、Comet/OpenSpec adapter）都必须立即检查退出码。除明确允许 merge 冲突、`git grep` 无匹配、Docker 不可用或目标 integration SKIP 的分支外，任何非零退出码立即停止；不得让后续成功命令覆盖失败码。
- 后续门禁发现已完成 Task 所属能力簇的遗漏时，不回退或取消已勾选任务。当前尚未勾选的 gate Task 保持打开，由执行编排层创建一个明确命名的 remediation 子步骤/agent，按原能力簇提交消息完成 RED/修复/review，并在当前 Task 证据中关联该提交后重跑整个当前门禁。若修复改变 spec/设计或新增任务超过阈值，按 Comet Spec 增量/范围扩张协议暂停，不能用 remediation 绕过。
- 上述 remediation 不适用于“源已提交但对应生成输出遗漏”的历史所有权错误。该情况必须硬阻塞，不得创建 generated-only 提交；只能在未共享的隔离历史中重建 source-owning 提交，使源与输出同提交，再重新执行该提交的 review 和当前 gate。任何历史重写都必须先停下按用户确认的分支恢复策略处理。

## 文件结构与证据

| 路径 | 责任 |
| --- | --- |
| `docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-build.md` | 创建并持续记录 source/execution base、tag manifest、dirty-path 判定、changed-files、冲突台账、能力矩阵、命令/exit code、RED/GREEN、Docker 判定和阶段结论。 |
| `docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-verify.md` | 创建并记录最终门禁、拓扑、migration identity/integration 结果、专项 review、残余风险和非目标确认。 |
| `backend/internal/repository/migrations_subscription_quota_passkey_upgrade_integration_test.go` | 在 `v0.1.170` 阶段扩展升级路径测试，覆盖双方 191、双方 192 和上游 193 的空库、已应用本地 migration 升级、幂等与 checksum。 |
| `backend/migrations/*.sql` | 仅由 tag merge 引入或保留。完整 filename、历史内容和 checksum 不可改写。 |
| `backend/ent/schema/`、`backend/internal/**/wire.go`、`backend/cmd/server/wire.go` | Ent/Wire 的源；先融合这些源，再运行 `make -C backend generate`。 |
| `backend/ent/`、`backend/cmd/server/wire_gen.go` | Ent/Wire 生成输出；只由生成命令更新。 |
| `backend/go.mod`、`backend/go.sum`、`frontend/package.json`、`frontend/pnpm-lock.yaml` | 依赖源和工具生成输出；只在实际 upstream manifest 冲突或能力修复触及依赖时按已有工具更新。 |
| `backend/cmd/server/VERSION` | 仅 Task 15 改为 `0.1.171.1`。 |

## 统一检查命令

以下 PowerShell 片段在每个 Task 的独立执行会话中重新运行。它们不创建分支/worktree，也不修改 Comet runtime state。

```powershell
$sourceBase = '16c07d8064b0b4604e9f47ef782e7d29534402d3'
$tag170 = 'c043c24774228ba891ddf90d783aa6dc7d0855b5'
$tag170Object = '60286d35e4b6dc6851ab69f890c2d1b7b7a3bcb8'
$tag171 = 'f0e7a9c7a23a7d02fb159b62fa809621eb0475a6'
$tag171Object = 'afd154b92aac36c6dafb1fa8e181ca827c78c465'
$layoutJson = @(comet classic root show 2>&1)
if ($LASTEXITCODE -ne 0) { throw "comet classic root show failed: $($layoutJson -join [Environment]::NewLine)" }
$layout = ($layoutJson -join [Environment]::NewLine) | ConvertFrom-Json
if ($layout.schema -ne 'comet.classic-layout.v1') { throw "unexpected Classic layout schema: $($layout.schema)" }
$repoRootOutput = @(git rev-parse --show-toplevel 2>&1)
if ($LASTEXITCODE -ne 0) { throw 'cannot normalize Classic paths without Git top-level' }
$repoRoot = ($repoRootOutput -join '').Trim()
function Convert-ToGitPath {
    param([Parameter(Mandatory)][string]$Path)
    $relative = if ([IO.Path]::IsPathRooted($Path)) { [IO.Path]::GetRelativePath($repoRoot, $Path) } else { $Path }
    return (($relative -replace '\\', '/') -replace '^\./', '')
}
$changesRootGit = Convert-ToGitPath $layout.changesRoot
$superpowersRootGit = Convert-ToGitPath $layout.superpowersRoot
$changeDir = "$changesRootGit/staged-merge-upstream-v0-1-171"
$openSpecTasks = "$changeDir/tasks.md"
$planFile = "$superpowersRootGit/plans/2026-08-06-staged-merge-upstream-v0-1-171.md"
$designDoc = "$superpowersRootGit/specs/2026-08-06-staged-merge-upstream-v0-1-171-design.md"
$buildLedger = "$superpowersRootGit/reports/2026-08-06-staged-merge-upstream-v0-1-171-build.md"
$verifyReport = "$superpowersRootGit/reports/2026-08-06-staged-merge-upstream-v0-1-171-verify.md"
$runtimeSelection = '?? .comet/current-change.json'
$requiredMigrationSources = [ordered]@{
    '191_passkey_credentials.sql' = $sourceBase
    '191_subscription_quota_advance_receipts.sql' = $sourceBase
    '192_subscription_cache_invalidation_outbox.sql' = $sourceBase
    '192_group_profit_control.sql' = $tag170
    '193_group_profit_control_auth_cache_invalidation.sql' = $tag170
}

function Invoke-CheckedNative {
    param(
        [Parameter(Mandatory)][string]$Label,
        [Parameter(Mandatory)][scriptblock]$Command
    )
    & $Command
    $exitCode = $LASTEXITCODE
    if ($exitCode -ne 0) { throw "$Label failed (exit $exitCode)" }
}

function Assert-CleanGate {
    $index = @(git diff --cached --name-only)
    if ($LASTEXITCODE -ne 0) { throw 'git diff --cached failed' }
    if ($index.Count -ne 0) { throw "staged paths block next step: $($index -join ', ')" }
    $status = @(git status --short --untracked-files=all)
    if ($LASTEXITCODE -ne 0) { throw 'git status failed' }
    $unexpected = @($status | Where-Object { $_ -ne $runtimeSelection })
    if ($unexpected.Count -ne 0) { throw "unexpected dirty paths: $($unexpected -join '; ')" }

    # docs/* 被 ignore；只显式枚举当前 change 的允许产物，防止临时文件混入归档范围。
    $ignoredChangeFiles = @(git ls-files --others --ignored --exclude-standard -- $changeDir $planFile $designDoc $buildLedger $verifyReport)
    if ($LASTEXITCODE -ne 0) { throw 'scoped ignored-file enumeration failed' }
    $allowedIgnoredExact = [System.Collections.Generic.HashSet[string]]::new([string[]]@(
        "$changeDir/.comet.yaml",
        "$changeDir/.openspec.yaml",
        "$changeDir/proposal.md",
        "$changeDir/design.md",
        "$changeDir/tasks.md",
        "$changeDir/specs/upstream-release-sync/spec.md",
        "$changeDir/.comet/artifacts.json",
        "$changeDir/.comet/checkpoint.json",
        "$changeDir/.comet/context.md",
        "$changeDir/.comet/run-state.json",
        "$changeDir/.comet/state-events.jsonl",
        "$changeDir/.comet/trajectory.jsonl",
        "$changeDir/.comet/subagent-progress.md",
        "$changeDir/.comet/handoff/brainstorm-summary.md",
        "$changeDir/.comet/handoff/spec-context.json",
        "$changeDir/.comet/handoff/spec-context.md",
        $planFile,
        $designDoc,
        $buildLedger,
        $verifyReport
    ))
    $unexpectedIgnored = @($ignoredChangeFiles | ForEach-Object { $_ -replace '\\', '/' } | Where-Object {
        -not $allowedIgnoredExact.Contains($_) -and
        $_ -notmatch ('^' + [regex]::Escape("$changeDir/.comet/skill-snapshots/") + '[0-9a-f]{64}/(package\.json|sha256)$')
    })
    if ($unexpectedIgnored.Count -ne 0) { throw "unexpected ignored change artifacts: $($unexpectedIgnored -join ', ')" }
}

function Commit-NamedPaths {
    param([Parameter(Mandatory)][string]$Message, [Parameter(Mandatory)][string[]]$Paths)
    # docs/* 被项目 .gitignore 忽略；显式 allowlist 配合 -f 同时适用于文档和源码。
    $normalizedPaths = @($Paths | ForEach-Object { Convert-ToGitPath $_ } | Sort-Object)
    git add -f -- $normalizedPaths
    if ($LASTEXITCODE -ne 0) { throw "git add failed for $Message" }
    $staged = @(git diff --cached --name-only | Sort-Object)
    $expected = $normalizedPaths
    if (Compare-Object -ReferenceObject $expected -DifferenceObject $staged) {
        throw "staged allowlist mismatch for $Message: $($staged -join ', ')"
    }
    git diff --cached --check
    if ($LASTEXITCODE -ne 0) { throw "whitespace check failed for $Message" }
    git commit -m $Message
    if ($LASTEXITCODE -ne 0) { throw "git commit failed for $Message" }
}

function Complete-TrackedTask {
    param(
        [Parameter(Mandatory)][string]$PlanTaskText,
        [Parameter(Mandatory)][string]$OpenSpecTaskText
    )
    # 协调者先用文件工具把两个唯一 checkbox 改为 [x]，再调用本函数。
    git add -f -- $planFile $openSpecTasks
    if ($LASTEXITCODE -ne 0) { throw 'failed to stage plan/OpenSpec checkoff' }
    $staged = @(git diff --cached --name-only | Sort-Object)
    $expected = @($openSpecTasks, $planFile) | Sort-Object
    if (Compare-Object -ReferenceObject $expected -DifferenceObject $staged) {
        throw "checkoff staged allowlist mismatch: $($staged -join ', ')"
    }
    git diff --cached --check
    if ($LASTEXITCODE -ne 0) { throw 'checkoff whitespace validation failed' }
    git commit -m 'docs: record SDD progress'
    if ($LASTEXITCODE -ne 0) { throw 'checkoff commit failed' }
    comet state task-checkoff $planFile $PlanTaskText
    if ($LASTEXITCODE -ne 0) { throw 'plan task-checkoff failed' }
    comet state task-checkoff $openSpecTasks $OpenSpecTaskText
    if ($LASTEXITCODE -ne 0) { throw 'OpenSpec task-checkoff failed' }
}

function Assert-NoConflictArtifacts {
    $unmerged = @(git diff --name-only --diff-filter=U)
    if ($LASTEXITCODE -ne 0) { throw 'failed to inspect unmerged worktree files' }
    if ($unmerged.Count -ne 0) { throw "unmerged files remain: $($unmerged -join ', ')" }
    $indexUnmerged = @(git ls-files -u)
    if ($LASTEXITCODE -ne 0) { throw 'failed to inspect unmerged index entries' }
    if ($indexUnmerged.Count -ne 0) { throw 'unmerged index entries remain' }
    git diff --check
    if ($LASTEXITCODE -ne 0) { throw 'worktree whitespace check failed' }
    git diff --cached --check
    if ($LASTEXITCODE -ne 0) { throw 'index whitespace check failed' }
    $markers = @(git grep -n -I -e '^<<<<<<< ' -e '^=======$' -e '^>>>>>>> ' -- .)
    if ($LASTEXITCODE -gt 1) { throw 'tracked conflict-marker scan failed' }
    if ($markers.Count -ne 0) { throw "tracked conflict markers remain: $($markers -join '; ')" }
}

function Assert-AnnotatedTagMergeHead {
    param(
        [Parameter(Mandatory)][string]$TagName,
        [Parameter(Mandatory)][string]$TagObject,
        [Parameter(Mandatory)][string]$PeeledCommit
    )
    $tagRef = (@(git rev-parse -q --verify "refs/tags/$TagName" 2>$null) -join '').Trim()
    if ($LASTEXITCODE -ne 0 -or $tagRef -ne $TagObject) { throw "$TagName tag object drifted" }
    $tagType = (@(git cat-file -t $tagRef 2>$null) -join '').Trim()
    if ($LASTEXITCODE -ne 0 -or $tagType -ne 'tag') { throw "$TagName is not the required annotated tag object" }
    $tagPeel = (@(git rev-parse -q --verify "refs/tags/$TagName^{}" 2>$null) -join '').Trim()
    if ($LASTEXITCODE -ne 0 -or $tagPeel -ne $PeeledCommit) { throw "$TagName peeled commit drifted" }
    $mergeHeadObject = (@(git rev-parse -q --verify MERGE_HEAD 2>$null) -join '').Trim()
    if ($LASTEXITCODE -ne 0 -or $mergeHeadObject -ne $TagObject) { throw "$TagName MERGE_HEAD object mismatch" }
    $mergeHeadPeel = (@(git rev-parse -q --verify 'MERGE_HEAD^{}' 2>$null) -join '').Trim()
    if ($LASTEXITCODE -ne 0 -or $mergeHeadPeel -ne $PeeledCommit) { throw "$TagName MERGE_HEAD peeled commit mismatch" }
}
```

下列函数固定目标 upgrade integration 名称和真实 PASS 判定。Task 5 在基线、Task 9 扩展测试、Task 10/14/17 重跑时都使用同一个名称；不得换成包级成功或宽泛的 `-run`。

```powershell
function Invoke-MigrationUpgradeIntegration {
    param([Parameter(Mandatory)][string]$Stage)

    $target = 'TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade'
    $dockerLog = Join-Path $env:TEMP "sub2api-v0.1.171-$Stage-docker-preflight.log"
    $testLog = Join-Path $env:TEMP "sub2api-v0.1.171-$Stage-migration-upgrade.log"
    $dockerOutput = @(& docker version --format '{{json .Server}}' 2>&1)
    $dockerExit = $LASTEXITCODE
    $dockerOutput | Set-Content -LiteralPath $dockerLog
    if ($dockerExit -ne 0 -or $dockerOutput.Count -eq 0) {
        return @{ Status = 'unverified'; Reason = "Docker daemon unavailable; preflight=$dockerLog"; Log = $dockerLog }
    }

    Push-Location backend
    try {
        $output = @(& go test -tags=integration -count=1 -v ./internal/repository -run "^$target$" 2>&1 | Tee-Object -FilePath $testLog)
        $exitCode = $LASTEXITCODE
    } finally {
        Pop-Location
    }

    $passPattern = '^--- PASS: ' + [regex]::Escape($target) + ' \('
    $skipPattern = '^--- SKIP: ' + [regex]::Escape($target) + ' \('
    $hasPass = @($output | Where-Object { $_ -match $passPattern })
    $hasSkip = @($output | Where-Object { $_ -match $skipPattern })
    if ($exitCode -eq 0 -and $hasPass.Count -ge 1) {
        return @{ Status = 'protected'; Reason = 'target test emitted real PASS'; Log = $testLog }
    }
    if ($hasSkip.Count -ge 1) {
        return @{ Status = 'unverified'; Reason = "target test skipped; log=$testLog"; Log = $testLog }
    }
    if ($exitCode -ne 0) { throw "migration integration failed; inspect $testLog" }
    throw "target integration did not emit PASS; inspect $testLog"
}
```

## 能力矩阵

Task 3 将以下每行写入 build ledger，并在每个阶段更新为 `protected`、`manual`、`unverified` 或 `gap`。`unverified` 只用于 Docker/Testcontainers 边界；任何 `gap` 都阻塞下一 tag。

| 能力契约 | 入口和证据重点 |
| --- | --- |
| advanced/layered scheduler、pool mode、DB recheck、WaitPlan fallback、同账号重试 | `openai_account_scheduler*`、`openai_gateway_scheduling.go`、`gateway_handler_sticky_fallback_test.go`；候选过滤、粘性绑定、槽位释放后重新选号。 |
| Grok/platform/session/previous-response sticky、privacy、image capability | scheduler/WS 入口；跨轮绑定、privacy 与 image 条件不能被上游选择逻辑短路。 |
| OpenAI HTTP/WS、turn ownership、最终 outbound model、failed usage、prompt-cache reuse、proxy circuit | `openai_gateway_handler.go`、`openai_ws_*`、`gateway_*usage*`；流式失败、WS 终止和切号不能重复计费。 |
| alpha-search、Responses fallback、PAT 401 副作用、WebSearchCalls、body handle | `openai_alpha_search.go`、`openai_gateway_request_body.go`；fallback/retry 后 handle 仍可重放且最终释放。 |
| 请求体 spooling/replay/cleanup、异步图片、对象存储、图片输入计费 | request-body、Images 和 image task 调用链；body 不泄漏、不提前关闭，计费仅一次。 |
| 统一 prompt/security audit 与 Images | coordinator、legacy moderation、`openai_images.go`；每请求最多一次 legacy moderation，payload 最多冻结一次，关闭态不构造大 payload，`ReleaseText` 前仍可审核。 |
| settings 热更新、repository scoped update、API Key auth cache、session binding/step-up | setting handler/service、user/api-key repository 和 auth middleware；部分更新不能清空本地字段，失效事件不得遗漏。 |
| subscription quota reset、续期、退款/余额、receipt、tombstone、outbox | `subscription_service.go`、payment/refund、repository；单窗口资格、事务锁、提交后 invalidation 和失败回滚。 |
| 用户资源控制、分组复制/批量限额、account shadow、前端本地定制 | backend handler/repository 和 Accounts/Groups/Settings/Usage/Subscription/Channels/mobile UI；变更页面保持本地入口。 |
| Codex identity、动态版本、UA、过载重试 | HTTP、透传、WS、探针、模型列表、账号测试和 alpha-search 共用身份来源；重试边界保留 final account/model 和错误语义。 |
| Ent/Wire、Go/pnpm 依赖、CSP/deploy 配置、migrations | source-driven generation、依赖工具、两轮无 diff、完整 filename、排序/checksum 和本机 integration。 |

### Task 1: 固定 source/execution 双基线和隔离状态

- [x] Task 1: 固定 source/execution 双基线和隔离状态

**映射 OpenSpec：**1.1

**文件：**创建 `docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-build.md`。

**步骤：**

1. 验证执行位置从 immutable source base 派生，`VERSION` 未提前升级，且 source-to-execution 之间只含当前 change 的规划文件：

```powershell
$repoRoot = @(git rev-parse --show-toplevel 2>&1)
if ($LASTEXITCODE -ne 0) { throw 'cannot resolve Git top-level' }
$currentBranch = @(git branch --show-current 2>&1)
if ($LASTEXITCODE -ne 0 -or ($currentBranch -join '').Trim() -eq '') { throw 'execution position must be on a named branch' }
$repoRoot
$currentBranch
git merge-base --is-ancestor $sourceBase HEAD
if ($LASTEXITCODE -ne 0) { throw "execution HEAD does not descend from $sourceBase" }
$executionBaseOutput = @(git rev-parse HEAD 2>&1)
if ($LASTEXITCODE -ne 0) { throw 'cannot resolve execution HEAD' }
$executionBase = ($executionBaseOutput -join '').Trim()
$sourceVersionOutput = @(git show "${sourceBase}:backend/cmd/server/VERSION" 2>&1)
if ($LASTEXITCODE -ne 0) { throw 'cannot read source VERSION' }
$sourceVersion = ($sourceVersionOutput -join '').Trim()
$executionVersionOutput = @(git show "${executionBase}:backend/cmd/server/VERSION" 2>&1)
if ($LASTEXITCODE -ne 0) { throw 'cannot read execution VERSION' }
$executionVersion = ($executionVersionOutput -join '').Trim()
if ($sourceVersion -ne '0.1.169.3' -or $executionVersion -ne '0.1.169.3') {
    throw "source/execution VERSION must both be 0.1.169.3: source=$sourceVersion execution=$executionVersion"
}
$allowedPlanning = @(
    'docs/openspec/changes/staged-merge-upstream-v0-1-171/',
    'docs/superpowers/specs/2026-08-06-staged-merge-upstream-v0-1-171-design.md',
    'docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md'
)
$baselinePaths = @(git log -m --format= --name-only "${sourceBase}..${executionBase}" | Where-Object { $_.Trim() } | Sort-Object -Unique)
if ($LASTEXITCODE -ne 0) { throw 'cannot enumerate source-to-execution paths' }
$unexpectedBaselinePaths = @($baselinePaths | Where-Object {
    $path = $_
    -not ($path -like 'docs/openspec/changes/staged-merge-upstream-v0-1-171/*' -or $path -in $allowedPlanning)
})
if ($unexpectedBaselinePaths.Count -ne 0) { throw "execution history touched non-planning paths: $($unexpectedBaselinePaths -join ', ')" }
$executionBase
```

预期：source/execution `VERSION` 均为 `0.1.169.3`，`$sourceBase` 是 `$executionBase` 祖先，任何 `backend/`、`frontend/`、`deploy/`、`Makefile` 或其他非规划路径都阻塞。

2. 执行 `Assert-CleanGate`。预期：index 为空，`git status --short --untracked-files=all` 只可为空或精确为 `?? .comet/current-change.json`；任何其他 dirty path 停止，不暂存、不覆盖。

3. 新建 build ledger 的“阶段 0/基线”段，写入执行位置、分支名、source/execution base、两者版本、source-to-execution 路径列表、runtime selection 状态、禁止操作清单和本任务命令输出。只提交 ledger：

```powershell
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $buildLedger) | Out-Null
Commit-NamedPaths -Message 'docs: record v0.1.171 baseline identity' -Paths @($buildLedger)
```

**提交边界：**`docs: record v0.1.171 baseline identity` 只能包含 build ledger。**检查点：**提交后再次运行 `Assert-CleanGate`；失败时不进入 Task 2。

### Task 2: 获取 refs 并冻结两个正式 tag 范围

- [x] Task 2: 获取 refs 并冻结两个正式 tag 范围

**映射 OpenSpec：**1.2

**文件：**修改 build ledger。初次 full gate 已证明 `backend/internal/handler/gateway_handler.go` 与 `backend/internal/service/gateway_usage_billing.go` 存在纯 `gofmt` RED；允许只对这两个精确文件运行 `gofmt` 并独立提交 `style: format v0.1.170 gateway files`。第二次 full gate 进一步复现两个行为簇 RED：允许在 `backend/internal/service/openai_gateway_grok.go` 中让 pool mode 在保留 quota snapshot、显式 403 temporary rule 后再跳过默认调度状态；订阅窗口按用户确认改为精确时间语义，删除 legacy midnight 纠偏，管理员重置与用户手动重置一样使用实际操作时间，并把 `backend/internal/service/subscription_monthly_window_test.go` fixture 更新为 versioned monthly reset interface。每次修复后都必须从 Tasks 7-9 focused gates 起完整重跑 Task 10，不复用失败运行的半段结果。

**步骤：**

1. 更新上游 refs 并核对两个 peeled SHA：

```powershell
Invoke-CheckedNative 'git fetch upstream refs/tags' { git fetch upstream --tags --prune }
$actual170Output = @(git rev-parse 'v0.1.170^{}' 2>&1)
if ($LASTEXITCODE -ne 0) { throw 'cannot resolve v0.1.170 peeled SHA' }
$actual170 = ($actual170Output -join '').Trim()
$actual171Output = @(git rev-parse 'v0.1.171^{}' 2>&1)
if ($LASTEXITCODE -ne 0) { throw 'cannot resolve v0.1.171 peeled SHA' }
$actual171 = ($actual171Output -join '').Trim()
if ($actual170 -ne $tag170) { throw "v0.1.170 peeled SHA mismatch: $actual170" }
if ($actual171 -ne $tag171) { throw "v0.1.171 peeled SHA mismatch: $actual171" }
git merge-base --is-ancestor v0.1.170 v0.1.171
if ($LASTEXITCODE -ne 0) { throw 'v0.1.170 is not an ancestor of v0.1.171' }
```

预期：两个 SHA 精确匹配，祖先检查 exit 0。

2. 确认 `v0.1.171` 仍为 merged `upstream/main` 中最高正式 `v0.1.*` tag，并保存范围外提交：

```powershell
$formalTags = @(git for-each-ref refs/tags --merged=upstream/main --format='%(refname:short)' |
    Where-Object { $_ -match '^v0\.1\.\d+$' } |
    Sort-Object { [version]$_.Substring(1) } -Descending)
if ($LASTEXITCODE -ne 0) { throw 'cannot enumerate formal upstream tags' }
if ($formalTags.Count -eq 0 -or $formalTags[0] -ne 'v0.1.171') {
    throw "newer formal tag changes scope: $($formalTags -join ', ')"
}
$excludedCommits = @(git log --oneline v0.1.171..upstream/main)
if ($LASTEXITCODE -ne 0) { throw 'cannot enumerate v0.1.171-excluded upstream commits' }
$excludedCommits
```

预期：最新正式 tag 是 `v0.1.171`；`$excludedCommits` 全部仅记录为范围外，不合并。出现更高正式 tag 时停止并更新 OpenSpec，不继续当前计划。

3. 在 ledger 记录 tags、peeled SHA、严格祖先关系、两段区间 `62/242` 与 `49/206` 的预期规模，以及 `$excludedCommits`。然后：

```powershell
Commit-NamedPaths -Message 'docs: record v0.1.171 upstream manifest' -Paths @($buildLedger)
```

**提交边界：**仅 build ledger。**检查点：**`Assert-CleanGate` 通过后才能建立矩阵。

### Task 3: 建立 changed-files 能力矩阵和六类冲突台账

- [x] Task 3: 建立 changed-files 能力矩阵和六类冲突台账

**映射 OpenSpec：**1.3

**文件：**修改 build ledger。

**步骤：**

1. 获取两个 release 区间的 changed-files，作为矩阵输入，而不是以全绿测试代替审查：

```powershell
$changed170 = @(git diff --name-only v0.1.169..v0.1.170)
if ($LASTEXITCODE -ne 0) { throw 'failed to enumerate v0.1.170 changed files' }
$changed171 = @(git diff --name-only v0.1.170..v0.1.171)
if ($LASTEXITCODE -ne 0) { throw 'failed to enumerate v0.1.171 changed files' }
if ($changed170.Count -ne 242) { throw "unexpected v0.1.170 changed-file count: $($changed170.Count)" }
if ($changed171.Count -ne 206) { throw "unexpected v0.1.171 changed-file count: $($changed171.Count)" }
$changed170
$changed171
```

预期：计数分别为 `242`、`206`，完整清单写入 ledger。

2. 将上文“能力矩阵”的全部 11 行写入 ledger。每行必须有行为契约、入口/调用链、关键文件、受影响 tag、聚焦命令或明确人工审查点、状态和证据位置。阶段 0 未直接证明的行初始为 `gap`，不能因为推断改为 `protected`。

3. 在 ledger 创建实际冲突台账表，字段固定为“阶段、文件、分类、ours 行为、theirs 行为、最小融合、验证证据、状态”。分类只能使用“上游修复”“本地定制”“接口/配置演进”“版本/依赖”“生成代码”“migration”。

4. 提交矩阵和台账模板：

```powershell
Commit-NamedPaths -Message 'docs: record v0.1.171 capability matrix' -Paths @($buildLedger)
```

**提交边界：**仅 build ledger。**检查点：**矩阵覆盖 scheduler、gateway/body、audit/auth、subscription/migration、frontend 和生成物；遗漏一项即不进入 Task 4。

### Task 4: 保护基线能力并完成非 Docker 门禁

- [x] Task 4: 保护基线能力并完成非 Docker 门禁

**映射 OpenSpec：**1.4

**文件：**只允许修改现有聚焦测试；阶段 0 不修改生产行为。生成稳定性检查若发现当前 source base 已有漂移，作为基线阻塞处理，不在本任务吸收未知生成输出。

**步骤：**

1. 先运行现有本地保护面。下列命令为基线聚焦集合；失败时将完整 RED 输出写入 ledger，不能在尚未引入 tag 的阶段用产品代码绕过：

```powershell
Push-Location backend
try {
    Invoke-CheckedNative 'baseline service focus tests' { go test -count=1 ./internal/service -run '^(TestLayered_|TestOpenAISelectAccountWithLoadAwareness_|TestGatewayServiceRecordUsage_)' }
    Invoke-CheckedNative 'baseline quota focus tests' { go test -tags=unit -count=1 ./internal/service -run '^(TestCalculateQuotaCycleAdvance_.*|TestAdvanceQuotaCycle_.*|TestAdminResetQuota_(UsesCommittedResetVersionForCacheInvalidation|OuterTransactionInvalidatesAfterCommit)|TestCheckAndResetWindows_UsesCommittedResetVersionForCacheInvalidation)$' }
    Invoke-CheckedNative 'baseline handler focus tests' { go test -count=1 ./internal/handler -run '^(TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot|TestOpenAIImages_|TestOpenAIResponsesWebSocket_)' }
    Invoke-CheckedNative 'baseline route audit tests' { go test -count=1 ./internal/server/routes -run '^(TestEveryGatewayPOSTRouteIsClassifiedForPromptAuditCoverage|TestResponsesWebSocketHasFirstAndSubsequentTurnPromptGates)$' }
} finally {
    Pop-Location
}
Invoke-CheckedNative 'baseline frontend focus tests' { pnpm --dir frontend exec vitest run src/utils/__tests__/subscriptionQuota.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/UsageView.spec.ts src/components/channels/__tests__/AvailableChannelsTable.spec.ts }
```

预期：所有被发现的测试都 PASS。若输出没有覆盖某个矩阵高风险契约，继续下一步补最小直接断言，而不是将该行标为 `protected`。

2. 对缺少断言的能力补最小保护测试。测试必须分别证明：scheduler 的 DB recheck/WaitPlan/sticky 不被短路；body 可 replay 后清理；Images 的统一 audit、legacy 单次、payload 单次冻结、关闭态惰性求值和 `ReleaseText` 顺序；quota reset/outbox 的提交后 tombstone。新测试若在当前基线上直接 GREEN，记录为现有行为保护，不伪造 RED；若真实失败，说明 source base 已不满足确认契约，停止并更新范围或拆分修复，不在阶段 0 修改生产代码。

```go
require.False(t, boundBeforeEligibleSelection)
require.True(t, releasedSlotWasRetried)
require.Equal(t, 1, payloadProviderCalls.Load())
require.NoError(t, err)
```

预期：保护测试证明当前契约并直接 GREEN。任何真实 RED 都是阶段 0 阻塞，不得通过放宽断言或提交生产修复继续。

3. 只提交实际新增或加强的基线保护测试，不包含生产源或生成输出：

```powershell
$baselineProtectionPaths = @(
    'backend/internal/service/openai_account_scheduler_layered_test.go',
    'backend/internal/handler/openai_images_controls_test.go',
    'backend/internal/service/subscription_reset_quota_test.go'
)
Commit-NamedPaths -Message 'test: protect v0.1.171 merge baseline' -Paths $baselineProtectionPaths
```

若 `$baselineProtectionPaths` 没有实际 diff，不创建空提交；将已运行命令和证明状态写入 ledger，留待 Task 5 统一提交证据。

4. 执行基线 full gate 和两轮源驱动生成：

```powershell
Invoke-CheckedNative 'baseline make test' { make test }
Invoke-CheckedNative 'baseline make build' { make "VERSION=0.1.169.3" "SHELL=D:/scoop/shims/bash.exe" build }
Invoke-CheckedNative 'baseline first generate refresh' { make -C backend generate }
$refreshGenerateDiff = @(git diff --name-only -- backend/ent backend/cmd/server/wire_gen.go)
if ($LASTEXITCODE -ne 0) { throw 'failed to inspect baseline generate refresh diff' }
if ($refreshGenerateDiff.Count -ne 0) {
    throw "immutable source base has stale generated output: $($refreshGenerateDiff -join ', ')"
}
Invoke-CheckedNative 'baseline first stable generate' { make -C backend generate }
$firstStableGenerateDiff = @(git diff --name-only -- backend/ent backend/cmd/server/wire_gen.go)
if ($LASTEXITCODE -ne 0) { throw 'failed to inspect baseline first stable generate diff' }
if ($firstStableGenerateDiff.Count -ne 0) { throw 'first stable generate changed tracked output' }
Invoke-CheckedNative 'baseline second stable generate' { make -C backend generate }
$secondStableGenerateDiff = @(git diff --name-only -- backend/ent backend/cmd/server/wire_gen.go)
if ($LASTEXITCODE -ne 0) { throw 'failed to inspect baseline second stable generate diff' }
if ($secondStableGenerateDiff.Count -ne 0) { throw 'second stable generate changed tracked output' }
Assert-NoConflictArtifacts
```

预期：`make test`、`make build` exit 0，三次 generate 均不得产生 diff。当前 immutable source base 若需要刷新生成物即视为基线阻塞，不在本 change 的上游 merge 前创建生成提交。

**提交边界：**只允许 `test: protect v0.1.171 merge baseline`（有实际测试新增时）及之后的 ledger evidence 提交；阶段 0 不提交生产修复或生成刷新。**检查点：**所有非 Docker 基线行必须为 `protected` 或 `manual`，无未解释 RED，才进入 Task 5。

### Task 5: 执行基线 Docker 条件门禁并封存阶段 0 证据

- [x] Task 5: 执行基线 Docker 条件门禁并封存阶段 0 证据

**映射 OpenSpec：**1.5

**文件：**修改 build ledger。

**步骤：**

1. 运行固定的 upgrade integration：

```powershell
$baselineIntegration = Invoke-MigrationUpgradeIntegration -Stage 'baseline'
$baselineIntegration
```

预期：仅当日志包含 `--- PASS: TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade (` 时，该 migration/repository 契约可标为 `protected`。Docker daemon 不可用或目标明确 `SKIP` 时将 status、日志路径、环境原因和受影响的空库/升级/幂等/checksum 契约记录为 `unverified`；非环境断言失败阻塞。

2. 在 ledger 汇总 Task 4 命令、生成稳定性、Task 5 Docker 结果、所有 `gap` 的处理。基线只允许存在由本机 Docker 边界导致的 `unverified`，不得有 `gap`。

3. 严格提交阶段 0 证据：

```powershell
Commit-NamedPaths -Message 'docs: record v0.1.171 stage 0 evidence' -Paths @($buildLedger)
Assert-CleanGate
```

**提交边界：**仅 build ledger。**检查点：**index/worktree 清洁，`gap=0`，才允许开始 `v0.1.170` merge。

### Task 6: 创建纯净的 v0.1.170 merge 节点

- [x] Task 6: 创建纯净的 v0.1.170 merge 节点

**映射 OpenSpec：**2.1

**文件：**所有由 `v0.1.169..v0.1.170` 引入的上游文件和必要冲突融合文件；可能包括 `backend/ent/`、`backend/cmd/server/wire_gen.go`、`backend/go.sum`、`frontend/pnpm-lock.yaml`。不包含 build ledger、verify report、任务文件或 Comet runtime selection。

**步骤：**

1. 重新运行 `Assert-CleanGate`，然后使用唯一允许的入口：

```powershell
git merge --no-ff --no-commit v0.1.170
$merge170Exit = $LASTEXITCODE
$merge170Conflicts = @(git diff --name-only --diff-filter=U)
$conflictListExit = $LASTEXITCODE
if ($conflictListExit -ne 0) { throw 'failed to enumerate v0.1.170 conflicts' }
Assert-AnnotatedTagMergeHead -TagName 'v0.1.170' -TagObject $tag170Object -PeeledCommit $tag170
if ($merge170Exit -notin @(0, 1)) { throw "v0.1.170 merge failed fatally (exit $merge170Exit)" }
if ($merge170Exit -eq 1 -and $merge170Conflicts.Count -eq 0) { throw 'v0.1.170 merge failed without resolvable conflicts' }
$merge170Conflicts
```

预期：Git 进入未提交 merge 状态，实际冲突清单与设计阶段预测的 28 个文本冲突可不同；将其完整写入台账。

2. 逐文件以 Task 3 的六类分类融合。审查至少覆盖 settings/auth cache、gateway handler、scheduler/usage、audit/subscription、frontend、版本/依赖和生成代码。对每个冲突记录 ours 行为、theirs 行为、最终融合与证据；可共存时实施最小融合。

**阻塞检查点：**若任一冲突要求在本地 UI、认证、计费、路由、请求体或调度的用户可见语义之间取舍，只记录未解决状态并停止，保持未提交 merge，不运行 `git commit`、不预先选择一方、也不使用破坏性回退命令。

3. 在 merge commit 前保持中间版本、从源再生生成物和依赖输出：

```powershell
$versionAfter170Merge = (Get-Content -Raw backend/cmd/server/VERSION).Trim()
if ($versionAfter170Merge -ne '0.1.169.3') { throw "intermediate VERSION changed: $versionAfter170Merge" }
Invoke-CheckedNative 'v0.1.170 merge generate' { make -C backend generate }
$stagedGoMod = @(git diff --cached --name-only -- backend/go.mod)
if ($LASTEXITCODE -ne 0) { throw 'failed to inspect staged v0.1.170 go.mod' }
if ($stagedGoMod.Count -gt 0) {
    Push-Location backend
    try { Invoke-CheckedNative 'v0.1.170 go mod tidy' { go mod tidy } } finally { Pop-Location }
}
$stagedFrontendManifest = @(git diff --cached --name-only -- frontend/package.json)
if ($LASTEXITCODE -ne 0) { throw 'failed to inspect staged v0.1.170 package.json' }
if ($stagedFrontendManifest.Count -gt 0) {
    Invoke-CheckedNative 'v0.1.170 pnpm lock refresh' { pnpm --dir frontend install --lockfile-only }
}
$mergeToolOutputs = @(git diff --name-only -- backend/ent backend/cmd/server/wire_gen.go backend/go.mod backend/go.sum frontend/package.json frontend/pnpm-lock.yaml)
if ($LASTEXITCODE -ne 0) { throw 'failed to enumerate v0.1.170 generated/tool output' }
if ($mergeToolOutputs.Count -ne 0) {
    git add -- $mergeToolOutputs
    if ($LASTEXITCODE -ne 0) { throw 'failed to stage v0.1.170 generated/tool output' }
}
Assert-NoConflictArtifacts
```

预期：Ent/Wire 只由 source 生成；Go/pnpm 输出只在对应 manifest 实际变化时由现有工具产生。生成物与 manifest/源一起进入本 merge，不手拼语义。

4. 检查暂存内容只属于上游树和必须的冲突融合，随后创建 merge commit：

```powershell
$forbiddenMergePaths = @(
    '.comet/current-change.json',
    $buildLedger,
    $verifyReport,
    'docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md',
    'docs/openspec/changes/staged-merge-upstream-v0-1-171/tasks.md'
)
$mergeStaged = @(git diff --cached --name-only)
$stagedListExit = $LASTEXITCODE
if ($stagedListExit -ne 0) { throw 'failed to enumerate v0.1.170 staged paths' }
$forbiddenStaged = @($mergeStaged | Where-Object { $_ -in $forbiddenMergePaths })
if ($forbiddenStaged.Count -ne 0) { throw "non-merge files staged: $($forbiddenStaged -join ', ')" }
$upstream170Paths = @(git diff --name-only v0.1.169..v0.1.170)
if ($LASTEXITCODE -ne 0) { throw 'failed to build v0.1.170 positive path allowlist' }
$upstream170Set = [System.Collections.Generic.HashSet[string]]::new([string[]]$upstream170Paths)
$compileAdaptationPaths = @(
    'backend/internal/handler/openai_gateway_cyber_test.go',
    'backend/internal/handler/openai_images_controls_test.go'
)
$unexpectedMergePaths = @($mergeStaged | Where-Object {
    -not $upstream170Set.Contains($_) -and
    $_ -notin $compileAdaptationPaths -and
    $_ -ne 'backend/cmd/server/wire_gen.go' -and
    $_ -ne 'backend/go.sum' -and
    $_ -ne 'frontend/pnpm-lock.yaml' -and
    -not $_.StartsWith('backend/ent/')
})
if ($unexpectedMergePaths.Count -ne 0) { throw "v0.1.170 staged paths outside upstream/generated/compile-adaptation allowlist: $($unexpectedMergePaths -join ', ')" }
$stagedCompileAdaptations = @($mergeStaged | Where-Object { $_ -in $compileAdaptationPaths })
if ($stagedCompileAdaptations.Count -gt 0 -and 'backend/internal/service/content_moderation.go' -notin $mergeStaged) {
    throw 'v0.1.170 content-moderation test signature adaptation lacks staged constructor source'
}
$generated170Exceptions = @($mergeStaged | Where-Object { -not $upstream170Set.Contains($_) -and $_ -notin $compileAdaptationPaths })
if ('backend/go.sum' -in $generated170Exceptions -and 'backend/go.mod' -notin $mergeStaged) { throw 'v0.1.170 go.sum exception lacks staged go.mod source' }
if ('frontend/pnpm-lock.yaml' -in $generated170Exceptions -and 'frontend/package.json' -notin $mergeStaged) { throw 'v0.1.170 lockfile exception lacks staged package.json source' }
if (@($generated170Exceptions | Where-Object { $_.StartsWith('backend/ent/') }).Count -gt 0 -and
    @($mergeStaged | Where-Object { $_.StartsWith('backend/ent/schema/') }).Count -eq 0) {
    throw 'v0.1.170 Ent output exception lacks staged schema source'
}
if ('backend/cmd/server/wire_gen.go' -in $generated170Exceptions -and
    @($mergeStaged | Where-Object { $_.StartsWith('backend/') -and $_.EndsWith('.go') -and $_ -ne 'backend/cmd/server/wire_gen.go' }).Count -eq 0) {
    throw 'v0.1.170 Wire output exception lacks staged Go source'
}
# 对完整 `git diff --cached` 做人工审查，并在冲突台账逐项记录 compile adaptation 和 generated exception 的源映射；无法证明源驱动时停止。
Assert-AnnotatedTagMergeHead -TagName 'v0.1.170' -TagObject $tag170Object -PeeledCommit $tag170
git diff --cached --check
if ($LASTEXITCODE -ne 0) { throw 'v0.1.170 merge staging has whitespace errors' }
git commit -m 'merge: upstream v0.1.170'
if ($LASTEXITCODE -ne 0) { throw 'v0.1.170 merge commit failed' }
$merge170Commit = (git rev-parse HEAD).Trim()
$merge170Parent2 = (git rev-parse "$merge170Commit^2").Trim()
if ($merge170Parent2 -ne $tag170) { throw "v0.1.170 merge second parent mismatch: $merge170Parent2" }
```

**提交边界：**`merge: upstream v0.1.170` 只含上游树、冲突融合、从已融合源生成的必要输出，以及用户明确批准的两个 content-moderation 测试构造器签名适配；没有 scheduler/usage、gateway/body、audit/auth、subscription/migration 或 frontend 的 merge 后行为回归修复。**检查点：**第二父精确是 `$tag170`，否则此阶段不闭合。

### Task 7: 以 TDD 修复 v0.1.170 scheduler/usage 回归

- [x] Task 7: 以 TDD 修复 v0.1.170 scheduler/usage 回归

**映射 OpenSpec：**2.2

**文件：**优先审查并按实际 RED 修改 `backend/internal/service/openai_profit_control*.go`、`backend/internal/service/gateway_profit_control*.go`、`backend/internal/service/gateway_request_pricing*.go`、`backend/internal/service/gateway_usage_billing.go`、`backend/internal/service/billing_cache_service*.go`、`backend/internal/service/openai_account_scheduler_layered*.go`、`backend/internal/handler/gateway_handler.go`、`backend/internal/handler/gemini_v1beta_handler.go`、`backend/internal/handler/gemini_sticky_toggle_test.go`、`backend/internal/handler/openai_profit_*_test.go`、`backend/internal/handler/openai_ws_turn_pricing_test.go`、`backend/internal/handler/failover_loop_profit_veto_test.go`、gateway billing eligibility 聚焦测试及同路径测试。`backend/internal/handler/gemini_v1beta_handler_test.go` 可保留 `NewContentModerationService` 的末尾 `emailService=nil` compile-only prerequisite，并可承载本 Task 的 Gemini WaitPlan/profit/sticky 回归；不得吸收 Task 8 行为修复。

**步骤：**

1. 先运行该能力簇的聚焦测试：

```powershell
Push-Location backend
try {
    Invoke-CheckedNative 'v0.1.170 scheduler/usage service tests' { go test -count=1 ./internal/service -run '^(Test.*Profit.*|Test.*Pricing.*|Test.*Layered.*|Test.*Sticky.*|Test.*WaitPlan.*|TestGatewayServiceRecordUsage.*|TestGatewayBillingEligibility.*)$' }
    Invoke-CheckedNative 'v0.1.170 scheduler/usage unit service tests' { go test -tags=unit -count=1 ./internal/service -run '^(Test.*Profit.*|Test.*Pricing.*|Test.*Layered.*|Test.*Sticky.*|Test.*WaitPlan.*|TestGatewayServiceRecordUsage.*|TestGatewayBillingEligibility.*)$' }
    Invoke-CheckedNative 'v0.1.170 scheduler/usage handler tests' { go test -count=1 ./internal/handler -run '^(Test.*Profit.*|TestOpenAI.*Pricing.*|TestGatewayHandler.*Sticky.*|Test.*Sticky.*|Test.*WaitPlan.*|Test.*Usage.*)$' }
    Invoke-CheckedNative 'v0.1.170 scheduler/usage unit handler tests' { go test -tags=unit -count=1 ./internal/handler -run '^(Test.*Profit.*|TestOpenAI.*Pricing.*|TestGatewayHandler.*Sticky.*|Test.*Sticky.*|Test.*WaitPlan.*|Test.*Usage.*)$' }
} finally {
    Pop-Location
}
```

预期：聚焦命令中的测试全部 PASS；每个失败先写入 ledger，归属为 `v0.1.170`。

2. 对缺失契约先补测试并实际运行：若直接 GREEN，记录上游实现已满足契约且不创建生产修复；若失败，保存真实 RED 后再做最小实现。断言必须覆盖默认利润控制关闭时选择不变、不合格账号不进入排序且不提前绑定 sticky、取得槽位后倍率变化释放槽位并重新选号、同一请求在等待/重试/切号中使用固定定价时刻、组合 group 不被误直接门禁，以及 advanced/layered scheduler、WaitPlan、DB recheck 和 usage billing 仍被调用。另补一条表驱动跨层 quota 生命周期回归，覆盖 daily/weekly/monthly：领域 `Check*Limit` 对“已有用量 + 本次正成本 = limit”放行；结算后 billing cache 快照 `usage = limit` 的 `CheckBillingEligibility` 继续放行；`usage > limit` 拒绝。该测试验证真实 API 契约序列，不伪造 HTTP 预检中不存在的正成本参数。

3. 只为已复现 RED 写最小兼容实现；对每个测试重跑相同 `go test` 命令至 GREEN。涉及 schema/provider 时先改源、运行 `make -C backend generate`，并将生成输出归于本提交。

4. 有实际修复时，用一个专属能力簇提交，路径严格等于实际 production/test/generated diff：

```powershell
$schedulerUsagePaths = @(
    'backend/internal/service/openai_profit_control.go',
    'backend/internal/service/gateway_profit_control.go',
    'backend/internal/service/gateway_request_pricing.go',
    'backend/internal/service/gateway_usage_billing.go',
    'backend/internal/service/openai_account_scheduler_layered.go',
    'backend/internal/service/openai_account_scheduler_layered_test.go',
    'backend/internal/service/gateway_profit_control_v2_test.go',
    'backend/internal/service/billing_cache_service.go',
    'backend/internal/service/billing_cache_service_test.go',
    'backend/internal/service/openai_profit_control_test.go',
    'backend/internal/handler/openai_profit_slot_recheck_test.go',
    'backend/internal/handler/openai_ws_turn_pricing_test.go',
    'backend/internal/handler/gateway_handler.go',
    'backend/internal/handler/gateway_handler_sticky_fallback_test.go',
    'backend/internal/handler/gemini_v1beta_handler.go',
    'backend/internal/handler/gemini_sticky_toggle_test.go',
    'backend/internal/handler/gemini_v1beta_handler_test.go'
)
Commit-NamedPaths -Message 'fix: preserve scheduler and usage after v0.1.170' -Paths $schedulerUsagePaths
```

先从列表移除未实际修改的路径，最终 `$schedulerUsagePaths` 必须等于 staged paths；没有 RED 或 diff 不创建空提交。将 RED/GREEN 和调用链结论更新到 ledger，暂不提交 ledger。

**提交边界：**`fix: preserve scheduler and usage after v0.1.170` 不包含 gateway/body、audit/auth、subscription/migration 或 frontend 文件。**检查点：**任何无法兼容的候选集、sticky 或计费语义停止并请求用户决定。

### Task 8: 审查 v0.1.170 gateway/body、audit/auth、subscription/migration 和 frontend 交互

- [x] Task 8: 审查 v0.1.170 gateway/body、audit/auth、subscription/migration 和 frontend 交互

**映射 OpenSpec：**2.3

**文件：**按实际 RED 审查 `gateway_anthropic_passthrough.go`、`openai_gateway_*`、`openai_alpha_search.go`、`openai_images.go`、`content_moderation*`、`subscription_*`、`setting_*`、相应 handler/repository 测试，以及 v0.1.170 触及的 account/settings/payment/prompt-audit 前端组件与 Vitest。

**步骤：**

1. 运行聚焦命令，覆盖 Anthropic 流中断 usage、OpenAI WS/流内错误、Responses 工具图片、审核代理/最新输入、订阅窗口、settings 部分更新、body spooling 和前端定制：

```powershell
Push-Location backend
try {
    Invoke-CheckedNative 'v0.1.170 gateway/audit/subscription service tests' { go test -count=1 ./internal/service -run '^(TestGateway.*PartialUsage.*|TestOpenAI.*(WS|WebSocket).*|Test.*Anthropic.*|Test.*ContentModeration.*|Test.*Subscription.*|Test.*Setting.*|Test.*RequestBody.*)$' }
    Invoke-CheckedNative 'v0.1.170 subscription unit contract tests' { go test -tags=unit -count=1 ./internal/service -run '^(TestDelayedFirstUseAnchorsMonthlyWindowAtStartsAt|TestAdminResetQuota_.*)$' }
    Invoke-CheckedNative 'v0.1.170 gateway/body handler tests' { go test -count=1 ./internal/handler -run '^(TestOpenAI.*(WS|WebSocket|Images|Responses).*|TestGatewayHandler.*(Usage|Body|Settings).*)$' }
} finally {
    Pop-Location
}
Invoke-CheckedNative 'v0.1.170 frontend focus tests' { pnpm --dir frontend exec vitest run src/components/account/__tests__/CreateAccountModal.spec.ts src/components/account/__tests__/EditAccountModal.spec.ts src/components/account/__tests__/BulkEditAccountModal.spec.ts src/components/account/__tests__/UpstreamBillingRateCell.spec.ts src/components/payment/__tests__/PaymentMethodSelector.spec.ts src/features/prompt-audit/__tests__/viewModel.spec.ts src/features/prompt-audit/__tests__/PromptAuditView.spec.ts src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts src/views/admin/__tests__/SettingsView.spec.ts }
```

2. 每个失败先写最小 RED，随后修复而不跨簇。必须证明请求体在 retry/failover 后不重用已关闭 handle、成功/失败 usage 不重复或遗漏、统一 audit 不绕开 latest-input/代理条件、subscription window/reset/outbox 仍保持锁和提交后失效、settings 未发送字段不丢失、本地前端入口未被上游页面改写。Task 6 已知的 tagged-unit RED 必须按用户最终裁决修正：首次订阅窗口锚定 entitlement `StartsAt`；所有持久化窗口直接使用精确起点，`AdminResetQuota` 与用户手动重置一样锚定实际操作时刻，不保留 legacy midnight 纠偏。

3. 按实际涉及面分别提交，不将四类修复合并：

```powershell
Commit-NamedPaths -Message 'fix: preserve gateway body after v0.1.170' -Paths $gatewayBodyPaths
Commit-NamedPaths -Message 'fix: preserve audit and auth after v0.1.170' -Paths $auditAuthPaths
Commit-NamedPaths -Message 'fix: preserve subscription migrations after v0.1.170' -Paths $subscriptionMigrationPaths
Commit-NamedPaths -Message 'fix: preserve frontend customization after v0.1.170' -Paths $frontendPaths
```

在调用前将每个变量显式赋为该簇实际变动路径，且每个列表不得为空。测试与必要生成输出跟随其主能力簇；没有该簇 RED 时不调用相应命令。

**提交边界：**四个消息各自只包含所属簇，严禁空提交。**检查点：**所有矩阵行有行为测试或人工调用链证据；涉及用户可见取舍时立即停止。

### Task 9: 固化 v0.1.170 migration identity 并按源生成 Ent/Wire

- [x] Task 9: 固化 v0.1.170 migration identity 并按源生成 Ent/Wire

**映射 OpenSpec：**2.4

**文件：**`backend/migrations/192_group_profit_control.sql`、`backend/migrations/193_group_profit_control_auth_cache_invalidation.sql`、`backend/migrations/192_subscription_cache_invalidation_outbox.sql`、`backend/internal/repository/migrations_subscription_quota_passkey_upgrade_integration_test.go`，以及实际更改的 Ent/Wire 源和生成输出。

**步骤：**

1. 核对五个完整 migration filename，确认旧的本地 191/192 blob identity 没有被 merge 改写：

```powershell
foreach ($entry in $requiredMigrationSources.GetEnumerator()) {
    $name = $entry.Key
    $sourceRef = $entry.Value
    git cat-file -e "HEAD:backend/migrations/$name"
    if ($LASTEXITCODE -ne 0) { throw "required migration missing from HEAD: $name" }
    $expectedOutput = @(git rev-parse "${sourceRef}:backend/migrations/$name" 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "cannot resolve authoritative migration $name from $sourceRef" }
    $actualOutput = @(git rev-parse "HEAD:backend/migrations/$name" 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "cannot resolve HEAD migration blob: $name" }
    $expectedBlob = ($expectedOutput -join '').Trim()
    $actualBlob = ($actualOutput -join '').Trim()
    if ($actualBlob -ne $expectedBlob) { throw "published migration identity changed: $name expected=$expectedBlob actual=$actualBlob" }
}
```

预期：五个文件均存在；passkey 191 和两个本地 migration 与 immutable source base blob 一致，profit-control 192/193 与固定 `v0.1.170` commit blob 一致。

2. 扩展固定 integration 测试 `TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade`。在 `filenames` 中明确列出五个 filename；构造 baseline FS 时保留双方 `191_*`、本地 `192_subscription_cache_invalidation_outbox.sql`，只排除上游 `192_group_profit_control.sql` 与 `193_group_profit_control_auth_cache_invalidation.sql`。测试必须使用两个相互隔离的 PostgreSQL 数据库：

- **升级库路径**：按完整 filename 排序运行 baseline FS、完整 FS、完整 FS；
- **空库路径**：在全新数据库直接运行完整 FS、完整 FS。

两条路径都断言每个 filename 的 checksum 与 `schema_migrations` 单行记录稳定，并对新增 migration 所创建的关系补 `to_regclass` 断言。由于 profit 192 只给既有 `groups` 增列、193 只替换既有 trigger function，还必须在 baseline 后证明三个 profit 列及函数中的 profit 字段引用缺席，在两条 complete 路径中验证三个列的类型/nullability/default，并通过 `pg_get_functiondef` 验证函数引用全部三个字段。同步更新测试中仍描述 `0.1.165.4` 或“唯一升级路径”的旧注释，使其准确描述 `main@16c07d806` / `0.1.169.3` 本地 migration 集。

测试结构固定保持下列顺序：

```go
require.NoError(t, applyMigrationsFS(ctx, db, baselineFS))
require.NoError(t, applyMigrationsFS(ctx, db, completeFS))
require.NoError(t, applyMigrationsFS(ctx, db, completeFS))

require.NoError(t, applyMigrationsFS(ctx, emptyDB, completeFS))
require.NoError(t, applyMigrationsFS(ctx, emptyDB, completeFS))
```

先扩展测试并运行目标 integration；若新增断言暴露产品或 runner 缺口，保存真实 RED 后做最小修复。若现有完整文件名执行器已满足新增断言，记录测试直接通过，不伪造 RED。随后用 `Invoke-MigrationUpgradeIntegration -Stage 'v0.1.170'` 取得真实 PASS 或明确 `unverified`。

3. 从已融合 schema/provider 源重新生成并验证连续两轮无 diff：

```powershell
Invoke-CheckedNative 'v0.1.170 migration first generate' { make -C backend generate }
$generatedPaths = @(git diff --name-only -- backend/ent backend/cmd/server/wire_gen.go)
if ($LASTEXITCODE -ne 0) { throw 'failed to inspect migration-stage generated output' }
if ($generatedPaths.Count -ne 0) {
    throw "BLOCKED: Task 9 found generated output omitted from an earlier source-owning commit. Do not create generated-only remediation; reconstruct the unshared source-owning commit under the confirmed branch recovery strategy, repeat its review, then rerun Task 9: $($generatedPaths -join ', ')"
}
Invoke-CheckedNative 'v0.1.170 migration second generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'Ent/Wire generation is not stable' }
Invoke-CheckedNative 'v0.1.170 migration third generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'second Ent/Wire generation is not stable' }
```

4. 仅在 Task 9 有实际测试、migration runner 源或生成输出 diff 时提交：

```powershell
$migrationPaths = @('backend/internal/repository/migrations_subscription_quota_passkey_upgrade_integration_test.go')
Commit-NamedPaths -Message 'test: cover v0.1.170 migration identity' -Paths $migrationPaths
```

若 Task 9 的真实 RED 证明 runner 源需要修复，将精确 repository 源路径加入同一 `$migrationPaths`；Task 9 不吸收任何 Ent/Wire 生成 diff。若测试源没有变化且没有 runner 修复，不调用提交命令。已在 Task 6 merge 的上游 migration 文件不得重新提交。

**提交边界：**`test: cover v0.1.170 migration identity` 只包含 subscription/migration 测试和真实 RED 证明必要的 runner 源。**检查点：**不满足完整 filename、五文件 blob identity、空库/升级库排序、幂等或 checksum 任一项即阻塞下一阶段。

### Task 10: 关闭 v0.1.170 阶段并记录证据

- [x] Task 10: 关闭 v0.1.170 阶段并记录证据

**映射 OpenSpec：**2.5

**文件：**修改 build ledger、technical design、`backend/internal/service/user_subscription.go`、`backend/internal/service/subscription_service.go`、`backend/internal/service/subscription_monthly_window_test.go`、`backend/internal/service/subscription_reset_quota_test.go`、`backend/internal/service/subscription_calculate_progress_test.go`、`backend/internal/service/user_subscription_daily_quota_test.go`。

**步骤：**

0. 先保留现有手动锚点 RED，并增加管理员精确时间断言：

```powershell
Push-Location backend
try {
    go test -tags=unit ./internal/service -run 'Test(AdminResetQuota_ResetBoth|AutomaticWindowPreservesPeriodAlignedLaterMidnightManualAnchor|AutomaticWindowPreservesExactMidnightManualAnchor)$' -count=1 -v
    if ($LASTEXITCODE -eq 0) { throw 'expected exact manual-anchor RED before implementation' }
} finally { Pop-Location }
```

预期：管理员重置仍返回当天 `00:00`，两个自动窗口测试仍把合法手动锚点改写到 `StartsAt` 相位，因此 RED。测试中把 `TestAdminResetQuota_ResetBoth` 的预期从 `startOfDay(resetAt)` 改为 `resetAt`；删除只验证 legacy midnight 纠偏的用例。仍验证末尾不足完整周期的用例把窗口起点改为精确 `StartsAt` 后保留。

最小生产实现固定为：

```go
func (s *UserSubscription) effectiveDailyWindowStart() *time.Time { return s.DailyWindowStart }
func (s *UserSubscription) effectiveWeeklyWindowStart() *time.Time { return s.WeeklyWindowStart }
func (s *UserSubscription) effectiveMonthlyWindowStart() *time.Time { return s.MonthlyWindowStart }

// automaticWindowStartAt 直接使用 *previous 作为 anchor。
anchor := *previous

// AdminResetQuota 与用户 AdvanceQuotaCycle 一样使用实际操作时间。
windowStart := s.now()
```

删除 `isMidnight`、`fixLegacyMidnightAnchor` 及其专属测试，不新增字段或 migration。随后执行：

```powershell
gofmt -w backend/internal/service/user_subscription.go backend/internal/service/subscription_service.go backend/internal/service/subscription_monthly_window_test.go backend/internal/service/subscription_reset_quota_test.go backend/internal/service/subscription_calculate_progress_test.go backend/internal/service/user_subscription_daily_quota_test.go
Push-Location backend
try {
    go test -tags=unit ./internal/service -run 'Test(AdminResetQuota|AutomaticWindow|CheckAndResetWindowsResetsPartialFinalMonthlySubscriptions|AutomaticWindowsAllowPartialFinalDailyAndWeeklyPeriods)' -count=1 -v
    if ($LASTEXITCODE -ne 0) { throw 'subscription exact-window focused tests failed' }
    go test -tags=unit ./internal/service -count=1
    if ($LASTEXITCODE -ne 0) { throw 'subscription unit package failed' }
} finally { Pop-Location }
```

预期：管理员和用户手动重置均保留实际操作时间，自动窗口始终从持久化起点推进，subscription unit package PASS。

fresh task review 对 spec 与 quality 均通过后，提交精确窗口 capability；technical design 与直接代码/测试同提交，保证该用户裁决有合法归属且不会遗留 dirty path：

```powershell
$subscriptionExactWindowPaths = @(
    'backend/internal/service/user_subscription.go',
    'backend/internal/service/subscription_service.go',
    'backend/internal/service/subscription_monthly_window_test.go',
    'backend/internal/service/subscription_reset_quota_test.go',
    'backend/internal/service/subscription_calculate_progress_test.go',
    'backend/internal/service/user_subscription_daily_quota_test.go',
    $designDoc
)
Commit-NamedPaths -Message 'fix: use exact subscription window anchors' -Paths $subscriptionExactWindowPaths
```

提交边界：只包含精确窗口生产代码、直接测试与该行为的 technical design 决策，不包含 plan、ledger、OpenSpec tasks、checkpoint 或 `.comet/current-change.json`。

1. 重跑 Tasks 7-9 的所有聚焦命令，然后执行阶段 full gate：

```powershell
Invoke-CheckedNative 'v0.1.170 stage make test' { make test }
Invoke-CheckedNative 'v0.1.170 stage make build' { make "VERSION=0.1.169.3" "SHELL=D:/scoop/shims/bash.exe" build }
Invoke-CheckedNative 'v0.1.170 stage first generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'first stage generate changed tracked output' }
Invoke-CheckedNative 'v0.1.170 stage second generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'second stage generate changed tracked output' }
Assert-NoConflictArtifacts
```

预期：所有命令 exit 0、两轮 generate 无 diff、无 whitespace/unmerged/tracked conflict marker。

2. 再运行 `Invoke-MigrationUpgradeIntegration -Stage 'v0.1.170-final'`。只有真实 PASS 才将该 integration 行标 `protected`；Docker/Testcontainers 不可用或明确 SKIP 时保留 `unverified` 和日志证据。

3. 在 ledger 写入实际冲突、每个兼容提交（含精确窗口 capability SHA）、逐行最终矩阵状态、聚焦/full 命令、integration 结果和 `v0.1.170` 结论。确认 `gap=0`，然后：

```powershell
Commit-NamedPaths -Message 'docs: record v0.1.170 stage evidence' -Paths @($buildLedger)
```

4. evidence review 通过后，用文件工具把本 Task 与 OpenSpec 2.5 的唯一 checkbox 改为 `[x]`，仅提交 plan/tasks，并更新两个持久状态：

```powershell
git add -f -- $planFile $openSpecTasks
if ($LASTEXITCODE -ne 0) { throw 'failed to stage Task 10 checkoff' }
$checkoffPaths = @(git diff --cached --name-only | Sort-Object)
$expectedCheckoffPaths = @($openSpecTasks, $planFile) | Sort-Object
if (Compare-Object -ReferenceObject $expectedCheckoffPaths -DifferenceObject $checkoffPaths) { throw 'Task 10 checkoff path mismatch' }
git diff --cached --check
if ($LASTEXITCODE -ne 0) { throw 'Task 10 checkoff whitespace validation failed' }
git commit -m 'docs: close v0.1.170 stage'
if ($LASTEXITCODE -ne 0) { throw 'Task 10 checkoff commit failed' }
comet state task-checkoff $planFile 'Task 10: 关闭 v0.1.170 阶段并记录证据'
if ($LASTEXITCODE -ne 0) { throw 'plan Task 10 state checkoff failed' }
comet state task-checkoff $openSpecTasks '2.5 运行 v0.1.170 聚焦测试、本机 full 门禁及适用的本机 integration，关闭能力矩阵 gap 并记录阶段证据后再进入下一 tag'
if ($LASTEXITCODE -ne 0) { throw 'OpenSpec 2.5 state checkoff failed' }
```

随后用文件工具把 checkpoint 更新到 Task 11，只提交 `$changeDir/.comet/subagent-progress.md`，消息为 `docs: advance to v0.1.171 merge`。确认其他 tracked dirty path 为空后再执行 `Assert-CleanGate`。

**提交边界：**evidence commit 仅 build ledger；checkoff commit 仅 plan/tasks；checkpoint commit 仅 subagent progress。**检查点：**`gap` 非零、未解释 RED、full gate 失败、错误地把 integration 写为 PASS，或最终 clean gate 失败，都阻塞 Task 11。

### Task 11: 创建纯净的 v0.1.171 merge 节点

- [x] Task 11: 创建纯净的 v0.1.171 merge 节点

**映射 OpenSpec：**3.1

**文件：**所有由 `v0.1.170..v0.1.171` 引入的上游文件和必要冲突融合文件；可能包括 Codex、captcha、auth、payment、subscription、WS、Ent/Wire、Go/pnpm 依赖及前端文件。排除所有本 change 报告、计划、tasks 和 runtime selection。

**步骤：**

1. 执行 clean gate 和第二段唯一入口：

```powershell
Assert-CleanGate
git merge --no-ff --no-commit v0.1.171
$merge171Exit = $LASTEXITCODE
$merge171Conflicts = @(git diff --name-only --diff-filter=U)
$conflictListExit = $LASTEXITCODE
if ($conflictListExit -ne 0) { throw 'failed to enumerate v0.1.171 conflicts' }
Assert-AnnotatedTagMergeHead -TagName 'v0.1.171' -TagObject $tag171Object -PeeledCommit $tag171
if ($merge171Exit -notin @(0, 1)) { throw "v0.1.171 merge failed fatally (exit $merge171Exit)" }
if ($merge171Exit -eq 1 -and $merge171Conflicts.Count -eq 0) { throw 'v0.1.171 merge failed without resolvable conflicts' }
$merge171Conflicts
```

预期：实际冲突逐项录入台账，并按六分类审查。

2. 融合冲突时逐项审查 Codex identity、captcha/auth/settings/CSP、退款与 usage、composite reasoning、WS lease、prompt audit、account/group/settings 前端和生成源。先从 schema/provider/manifest 源融合，再执行：

```powershell
$versionAfter171Merge = (Get-Content -Raw backend/cmd/server/VERSION).Trim()
if ($versionAfter171Merge -ne '0.1.169.3') { throw "intermediate VERSION changed: $versionAfter171Merge" }
Invoke-CheckedNative 'v0.1.171 merge generate' { make -C backend generate }
$stagedGoMod = @(git diff --cached --name-only -- backend/go.mod)
if ($LASTEXITCODE -ne 0) { throw 'failed to inspect staged v0.1.171 go.mod' }
if ($stagedGoMod.Count -gt 0) {
    Push-Location backend
    try { Invoke-CheckedNative 'v0.1.171 go mod tidy' { go mod tidy } } finally { Pop-Location }
}
$stagedFrontendManifest = @(git diff --cached --name-only -- frontend/package.json)
if ($LASTEXITCODE -ne 0) { throw 'failed to inspect staged v0.1.171 package.json' }
if ($stagedFrontendManifest.Count -gt 0) {
    Invoke-CheckedNative 'v0.1.171 pnpm lock refresh' { pnpm --dir frontend install --lockfile-only }
}
$mergeToolOutputs = @(git diff --name-only -- backend/ent backend/cmd/server/wire_gen.go backend/go.mod backend/go.sum frontend/package.json frontend/pnpm-lock.yaml)
if ($LASTEXITCODE -ne 0) { throw 'failed to enumerate v0.1.171 generated/tool output' }
if ($mergeToolOutputs.Count -ne 0) {
    git add -- $mergeToolOutputs
    if ($LASTEXITCODE -ne 0) { throw 'failed to stage v0.1.171 generated/tool output' }
}
Assert-NoConflictArtifacts
```

若出现不能共存的用户可见语义，记录并停止在未提交 merge；不创建 merge commit。

3. 提交纯 merge 并验证第二父：

```powershell
$mergeStaged = @(git diff --cached --name-only)
$stagedListExit = $LASTEXITCODE
if ($stagedListExit -ne 0) { throw 'failed to enumerate v0.1.171 staged paths' }
$forbiddenStaged = @($mergeStaged | Where-Object {
    $_ -in @('.comet/current-change.json', $buildLedger, $verifyReport,
        'docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md',
        'docs/openspec/changes/staged-merge-upstream-v0-1-171/tasks.md')
})
if ($forbiddenStaged.Count -ne 0) { throw "non-merge files staged: $($forbiddenStaged -join ', ')" }
$upstream171Paths = @(git diff --name-only v0.1.170..v0.1.171)
if ($LASTEXITCODE -ne 0) { throw 'failed to build v0.1.171 positive path allowlist' }
$upstream171Set = [System.Collections.Generic.HashSet[string]]::new([string[]]$upstream171Paths)
$unexpectedMergePaths = @($mergeStaged | Where-Object {
    -not $upstream171Set.Contains($_) -and
    $_ -ne 'backend/cmd/server/wire_gen.go' -and
    $_ -ne 'backend/go.sum' -and
    $_ -ne 'frontend/pnpm-lock.yaml' -and
    -not $_.StartsWith('backend/ent/')
})
if ($unexpectedMergePaths.Count -ne 0) { throw "v0.1.171 staged paths outside upstream/generated allowlist: $($unexpectedMergePaths -join ', ')" }
$generated171Exceptions = @($mergeStaged | Where-Object { -not $upstream171Set.Contains($_) })
if ('backend/go.sum' -in $generated171Exceptions -and 'backend/go.mod' -notin $mergeStaged) { throw 'v0.1.171 go.sum exception lacks staged go.mod source' }
if ('frontend/pnpm-lock.yaml' -in $generated171Exceptions -and 'frontend/package.json' -notin $mergeStaged) { throw 'v0.1.171 lockfile exception lacks staged package.json source' }
if (@($generated171Exceptions | Where-Object { $_.StartsWith('backend/ent/') }).Count -gt 0 -and
    @($mergeStaged | Where-Object { $_.StartsWith('backend/ent/schema/') }).Count -eq 0) {
    throw 'v0.1.171 Ent output exception lacks staged schema source'
}
if ('backend/cmd/server/wire_gen.go' -in $generated171Exceptions -and
    @($mergeStaged | Where-Object { $_.StartsWith('backend/') -and $_.EndsWith('.go') -and $_ -ne 'backend/cmd/server/wire_gen.go' }).Count -eq 0) {
    throw 'v0.1.171 Wire output exception lacks staged Go source'
}
# 对完整 `git diff --cached` 做人工审查，并在冲突台账逐项记录所有 generated exception 的源映射；无法证明源驱动时停止。
Assert-AnnotatedTagMergeHead -TagName 'v0.1.171' -TagObject $tag171Object -PeeledCommit $tag171
git diff --cached --check
if ($LASTEXITCODE -ne 0) { throw 'v0.1.171 merge staging has whitespace errors' }
git commit -m 'merge: upstream v0.1.171'
if ($LASTEXITCODE -ne 0) { throw 'v0.1.171 merge commit failed' }
$merge171Commit = (git rev-parse HEAD).Trim()
$merge171Parent2 = (git rev-parse "$merge171Commit^2").Trim()
if ($merge171Parent2 -ne $tag171) { throw "v0.1.171 merge second parent mismatch: $merge171Parent2" }
```

**提交边界：**`merge: upstream v0.1.171` 只含上游树、冲突融合及由其源驱动的输出。**检查点：**第二父精确是 `$tag171`；任何 merge 后语义回归留到 Task 12/13 的独立能力簇提交。

### Task 12: 以 TDD 修复 v0.1.171 Codex、过载重试和 gateway/body 回归

- [x] Task 12: 以 TDD 修复 v0.1.171 Codex、过载重试和 gateway/body 回归

**映射 OpenSpec：**3.2

**文件：**优先审查 `backend/internal/pkg/openai/request.go`、`request_codex_version_test.go`、`openai_codex_identity*.go`、`openai_codex_version_sync_service*.go`、`openai_capacity_shed_test.go`、`openai_alpha_search*.go`、`openai_gateway_*`、`openai_ws_*`、`gateway_request_body_spooling_test.go` 及直接依赖测试。

**步骤：**

1. 先运行 Codex/gateway/body 聚焦集：

```powershell
Push-Location backend
try {
    Invoke-CheckedNative 'v0.1.171 Codex identity tests' { go test -count=1 ./internal/pkg/openai -run '^Test.*Codex.*' }
    Invoke-CheckedNative 'v0.1.171 Codex/capacity/alpha service tests' { go test -count=1 ./internal/service -run '^(Test.*Codex.*|Test.*Capacity.*|Test.*AlphaSearch.*|TestOpenAI.*(Forward|Passthrough|WS).*)$' }
    Invoke-CheckedNative 'v0.1.171 gateway/body handler tests' { go test -count=1 ./internal/handler -run '^(Test.*Gateway.*(Body|Failover|Usage).*|TestOpenAI.*WebSocket.*)$' }
} finally {
    Pop-Location
}
```

2. 对缺失契约先补测试并运行；直接 GREEN 时记录证据且不制造生产 diff，真实失败时保存 RED 后再修复。必须覆盖 HTTP、透传、WS 握手、探针、模型列表、账号测试和 alpha-search 全部从动态版本来源得到一致 Codex identity；账号自定义 UA 只能保留允许的客户端/环境信息，不能恢复陈旧版本。还必须证明 `server_is_overloaded`/`slow_down` 仅在同账号有界重试后切号、不冷却账号、保持 pool/sticky/failover/error 语义、body 可重放且在最终完成时释放、failed usage 不重复计费并记录最终 outbound account/model。

3. 对每个 RED 写最小 gateway/body 兼容修复，重跑相同聚焦测试到 GREEN。若修复触及请求体或 routing 语义且发现用户可见冲突，停止等待用户决定。

4. 有实际 diff 时将所有 production/test/generated 路径提交为一个 gateway/body 能力簇：

```powershell
Commit-NamedPaths -Message 'fix: preserve gateway body after v0.1.171' -Paths $gatewayBodyPaths
```

调用前将 `$gatewayBodyPaths` 显式设为本任务实际 diff 路径；其中不得含 audit/auth、subscription/migration 或 frontend 路径，也不得为空。将 RED/GREEN、身份入口清单、重试次数和最终资源释放证据写入 ledger，暂不提交 ledger。

**提交边界：**仅 `fix: preserve gateway body after v0.1.171`。**检查点：**所有出站入口的身份来源一致、重试有界、body 生命周期与计费均有直接行为证据。

### Task 13: 以 TDD 修复 v0.1.171 audit/auth、subscription/migration 和 frontend 回归

- [x] Task 13: 以 TDD 修复 v0.1.171 audit/auth、subscription/migration 和 frontend 回归

**映射 OpenSpec：**3.3

**文件：**按实际 RED 审查 `auth_*`、`passkey_*`、`tencent_captcha_*`、`aliyun_captcha_*`、`setting_*`、`security_headers*`、`payment_refund*`、`gateway_usage_billing.go`、`subscription_renewal_lock_test.go`、`openai_reasoning_effort_policy*`、`openai_ws_forwarder_ingress*`、`securityaudit/prompt_snapshot*`，以及 `frontend/src/components/{AliyunCaptchaWidget,TencentCaptchaGate,CaptchaChallenge}.vue`、auth/admin settings/group/account/order 视图和对应 Vitest。

**步骤：**

1. 运行聚焦 test 集：

```powershell
Push-Location backend
try {
    Invoke-CheckedNative 'v0.1.171 auth/refund/reasoning service tests' { go test -count=1 ./internal/service -run '^(Test.*(Tencent|Aliyun|Turnstile|Captcha|Auth|Refund|Renewal|Reasoning|WebSocket|Prompt|Usage).*)$' }
    Invoke-CheckedNative 'v0.1.171 auth/settings handler tests' { go test -count=1 ./internal/handler -run '^(Test.*(Captcha|Auth|Passkey|Setting|Prompt|Usage).*)$' }
    Invoke-CheckedNative 'v0.1.171 auth/CSP middleware tests' { go test -count=1 ./internal/server/middleware -run '^(Test.*(Auth|SecurityHeaders|CSP).*)$' }
} finally {
    Pop-Location
}
Invoke-CheckedNative 'v0.1.171 frontend focus tests' { pnpm --dir frontend exec vitest run src/components/__tests__/AliyunCaptchaWidget.spec.ts src/components/__tests__/TencentCaptchaGate.spec.ts src/components/auth/__tests__/PendingOAuthCreateAccountForm.spec.ts src/views/auth/__tests__/TencentCaptchaActionGate.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/groupsReasoningEffort.spec.ts }
```

2. 对每个缺口先补测试并运行；直接 GREEN 时记录证据且不创建修复，真实失败时保存 RED 后实现最小兼容修复。断言验证码 provider 互斥、注册/登录/找回密码/OAuth 启动/passkey fail-closed、既有 Turnstile 与热更新/CSP/本地认证 UI 仍可用；断言退款 force、Stripe 幂等、失败仍落 usage 且实收为零、订阅并发续期/额度缓存保持事务锁与 outbox；断言 composite reasoning、WS lease 终止和取消后 snapshot/prompt audit 仍符合本地契约。

3. 按主能力簇分开提交，只调用有实际 diff 的命令：

```powershell
Commit-NamedPaths -Message 'fix: preserve audit and auth after v0.1.171' -Paths $auditAuthPaths
Commit-NamedPaths -Message 'fix: preserve subscription migrations after v0.1.171' -Paths $subscriptionMigrationPaths
Commit-NamedPaths -Message 'fix: preserve frontend customization after v0.1.171' -Paths $frontendPaths
```

每个变量在调用前显式列出本次实际 production/test/generated 路径；测试文件跟随主能力，不跨簇混入。无 RED、无 diff 的簇不创建提交。

**提交边界：**三个能力簇独立，分别只包含 audit/auth、subscription/migration、frontend。**检查点：**provider/认证、金额/usage、订阅事务、WS/prompt audit 和前端路径如有不可兼容的用户可见结论，停止等待用户决定。

### Task 14: 关闭 v0.1.171 阶段并记录证据

- [x] Task 14: 关闭 v0.1.171 阶段并记录证据

**映射 OpenSpec：**3.4

**文件：**修改 build ledger。

**步骤：**

1. 重跑 Tasks 12/13 全部聚焦命令，再执行阶段 full gate：

```powershell
Invoke-CheckedNative 'v0.1.171 stage make test' { make test }
Invoke-CheckedNative 'v0.1.171 stage make build' { make "VERSION=0.1.169.3" "SHELL=D:/scoop/shims/bash.exe" build }
Invoke-CheckedNative 'v0.1.171 stage first generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'v0.1.171 first generate changed tracked output' }
Invoke-CheckedNative 'v0.1.171 stage second generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'v0.1.171 second generate changed tracked output' }
Assert-NoConflictArtifacts
```

2. 运行 `Invoke-MigrationUpgradeIntegration -Stage 'v0.1.171-final'`，严格按真实 PASS/明确 `unverified` 更新矩阵。

3. 在 ledger 写入 v0.1.171 实际冲突、能力簇提交、每个入口调用链、所有测试和 Docker 结果；确认 `gap=0` 后提交：

```powershell
Commit-NamedPaths -Message 'docs: record v0.1.171 stage evidence' -Paths @($buildLedger)
Assert-CleanGate
```

**提交边界：**仅 build ledger。**检查点：**此任务完成前版本仍为 `0.1.169.3`；full gate、矩阵或 Docker 记录任何一项不实都不能进入 Task 15。

### Task 15: 一次更新最终 VERSION

- [ ] Task 15: 一次更新最终 VERSION

**映射 OpenSpec：**4.1

**文件：**修改 `backend/cmd/server/VERSION`。

**步骤：**

1. 确认两个阶段已关闭且当前版本仍为中间值：

```powershell
Assert-CleanGate
$beforeFinalVersion = (Get-Content -Raw backend/cmd/server/VERSION).Trim()
if ($beforeFinalVersion -ne '0.1.169.3') { throw "expected intermediate VERSION 0.1.169.3, got $beforeFinalVersion" }
git merge-base --is-ancestor $tag170 HEAD
if ($LASTEXITCODE -ne 0) { throw 'v0.1.170 is not an ancestor of final candidate' }
git merge-base --is-ancestor $tag171 HEAD
if ($LASTEXITCODE -ne 0) { throw 'v0.1.171 is not an ancestor of final candidate' }
```

2. 将文件的唯一内容改为：

```text
0.1.171.1
```

3. 确认只有该路径改变并提交：

```powershell
if ((Get-Content -Raw backend/cmd/server/VERSION).Trim() -ne '0.1.171.1') { throw 'VERSION write did not persist exact value' }
Commit-NamedPaths -Message 'chore: bump version to 0.1.171.1' -Paths @('backend/cmd/server/VERSION')
```

**提交边界：**`chore: bump version to 0.1.171.1` 仅一个文件。**检查点：**不创建任何 `0.1.170.1` 或其他过程版本。

### Task 16: 在最终 source HEAD 重跑完整本机门禁

- [ ] Task 16: 在最终 source HEAD 重跑完整本机门禁

**映射 OpenSpec：**4.2

**文件：**默认无产品文件修改；若 full gate 暴露回归，Task 16 保持未勾选，由执行编排层按首次引入 tag 和能力簇派发 remediation 子步骤，先 RED 后最小修复并使用原能力簇提交消息，然后完整重跑 Task 16。

**步骤：**

1. 重跑所有矩阵聚焦命令及 root full gate：

```powershell
Invoke-CheckedNative 'final make test' { make test }
Invoke-CheckedNative 'final make build' { make "VERSION=0.1.171.1" "SHELL=D:/scoop/shims/bash.exe" build }
Invoke-CheckedNative 'final first generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'final first generate changed tracked output' }
Invoke-CheckedNative 'final second generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'final second generate changed tracked output' }
Assert-NoConflictArtifacts
```

预期：`make test` 覆盖后端默认/unit/lint、前端 ESLint/Vitest/typecheck；`make build` 覆盖前后端构建；两轮生成无 diff。

2. 若失败，ledger 保存命令、exit code、RED 和首次出现 release 区间；保持 Task 16 未勾选，按全局 remediation 协议在当前 Task 内派发相应能力簇修复并重新运行全部最终门禁。不可用 Docker 不能通过修改产品代码绕过；用户可见语义取舍仍须停下。

**提交边界：**此任务不创建提交；修复使用其所属能力簇消息，门禁证据由 Task 18 统一提交。**检查点：**全部非 Docker 自动门禁通过才进入 Task 17。

### Task 17: 验证拓扑、migration identity 和最终 Docker 结果

- [ ] Task 17: 验证拓扑、migration identity 和最终 Docker 结果

**映射 OpenSpec：**4.3

**文件：**无产品文件修改；Task 18 将结果写入 build ledger 和 verify report。

**步骤：**

1. 验证 tags 是最终 HEAD 祖先，且从 immutable source base 到 HEAD 恰有一个第二父匹配各 peeled SHA 的 merge 节点：

```powershell
git merge-base --is-ancestor $tag170 HEAD
if ($LASTEXITCODE -ne 0) { throw 'v0.1.170 is not an ancestor of HEAD' }
git merge-base --is-ancestor $tag171 HEAD
if ($LASTEXITCODE -ne 0) { throw 'v0.1.171 is not an ancestor of HEAD' }
$firstParentMerges = @(git rev-list --first-parent --merges "${sourceBase}..HEAD")
if ($LASTEXITCODE -ne 0) { throw 'failed to enumerate final first-parent merge chain' }
$merge170Nodes = @($firstParentMerges | ForEach-Object {
    $candidate = $_.Trim()
    if ((git rev-parse "$candidate^2").Trim() -eq $tag170) { $candidate }
})
$merge171Nodes = @($firstParentMerges | ForEach-Object {
    $candidate = $_.Trim()
    if ((git rev-parse "$candidate^2").Trim() -eq $tag171) { $candidate }
})
if ($merge170Nodes.Count -ne 1 -or $merge171Nodes.Count -ne 1) { throw 'expected exactly one merge node per target tag' }
$merge170Commit = $merge170Nodes[0]
$merge171Commit = $merge171Nodes[0]
$index170 = [Array]::IndexOf([object[]]$firstParentMerges, $merge170Commit)
$index171 = [Array]::IndexOf([object[]]$firstParentMerges, $merge171Commit)
if ($index170 -lt 0 -or $index171 -lt 0 -or $index171 -ge $index170) {
    throw "target merges are not ordered v0.1.170 then v0.1.171 on first-parent: 170=$index170 171=$index171"
}
$merge170Commit
$merge171Commit
git rev-list --parents -n 1 $merge170Commit
if ($LASTEXITCODE -ne 0) { throw 'failed to read v0.1.170 merge parents' }
git rev-list --parents -n 1 $merge171Commit
if ($LASTEXITCODE -ne 0) { throw 'failed to read v0.1.171 merge parents' }
```

预期：两个 ancestor 检查均 exit 0；两个目标 merge 都唯一存在于最终 first-parent 链，`v0.1.171` 节点在 `v0.1.170` 之后，且与阶段 ledger 记录的 commit 一致。将 commit、父列表和 `$executionBase` 写入验证报告。

2. 再次核对最终版本与 migration identity：

```powershell
if ((Get-Content -Raw backend/cmd/server/VERSION).Trim() -ne '0.1.171.1') { throw 'final VERSION mismatch' }
foreach ($entry in $requiredMigrationSources.GetEnumerator()) {
    $name = $entry.Key
    $sourceRef = $entry.Value
    git cat-file -e "HEAD:backend/migrations/$name"
    if ($LASTEXITCODE -ne 0) { throw "required final migration missing: $name" }
    $expectedOutput = @(git rev-parse "${sourceRef}:backend/migrations/$name" 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "cannot resolve authoritative final migration $name from $sourceRef" }
    $actualOutput = @(git rev-parse "HEAD:backend/migrations/$name" 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "cannot resolve final migration blob: $name" }
    $expectedBlob = ($expectedOutput -join '').Trim()
    $actualBlob = ($actualOutput -join '').Trim()
    if ($actualBlob -ne $expectedBlob) { throw "final migration identity mismatch: $name expected=$expectedBlob actual=$actualBlob" }
}
git ls-tree -r --name-only HEAD backend/migrations | Where-Object { $_ -match '/19[123]_.*\.sql$' }
if ($LASTEXITCODE -ne 0) { throw 'failed to list final migration set' }
```

预期：五个完整文件名存在，且每个 blob OID 与其权威 source ref 精确一致；不能仅按数字前缀或测试动态计算的 checksum 推断身份。

3. 执行最终 migration integration：

```powershell
$finalIntegration = Invoke-MigrationUpgradeIntegration -Stage 'final'
$finalIntegration
```

预期：真实 `--- PASS:` 才写为通过；Docker 不可用或明确 SKIP 时写 `unverified`，并在报告中明确空库、从本地 191/192 集升级、排序、幂等和 checksum 未验证的范围。

**提交边界：**此任务不创建提交。**检查点：**拓扑、版本、完整 filename 或本地 migration blob identity 任一失败均阻塞最终报告。

### Task 18: 完成能力专项 review 和最终验证报告

- [ ] Task 18: 完成能力专项 review 和最终验证报告

**映射 OpenSpec：**4.4

**文件：**修改 build ledger；创建 `docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-verify.md`。

**步骤：**

1. 逐行关闭能力矩阵，专项 review 必须明确 scheduler、platform/session sticky、fallback/WaitPlan、DB recheck、privacy/image capability、HTTP/WS/turn ownership、body replay/cleanup、alpha-search/composite、统一 audit、settings/auth cache/step-up、quota reset/outbox、退款/usage、资源控制/分组复制/批量限额、frontend 本地能力、依赖/生成物和 migrations 的入口、证据和结论。`gap` 必须为零；`unverified` 必须只含 Task 5/10/14/17 的可追溯本机 Docker 原因。

2. 运行 OpenSpec 严格校验：

```powershell
Invoke-CheckedNative 'OpenSpec strict validation' { comet classic openspec -- validate staged-merge-upstream-v0-1-171 --strict }
```

预期：exit 0。失败时修正规划/规范一致性，不得以应用源码改动掩盖规范问题。

3. 在 verify report 记录：source/execution base、两个 tag/merge 节点及其第二父、最终 `VERSION=0.1.171.1`、每个 full/聚焦命令、两轮生成、静态检查、migration filename/blob/integration 结果、能力专项 review、残余风险，并明确本 change 未 push、未 tag、未 release、未 deploy、未操作服务器。

4. 只提交证据文档：

```powershell
Commit-NamedPaths -Message 'docs: record v0.1.171 verification' -Paths @($buildLedger, $verifyReport)
Assert-CleanGate
```

**提交边界：**`docs: record v0.1.171 verification` 只能包含两份报告。**完成检查点：**两个 merge 节点的第二父精确、最终版本精确、`gap=0`、所有非 Docker 门禁真实通过，且任何未执行 integration 只作为明确残余风险记录。

## 任务覆盖核对

| OpenSpec 任务 | 本计划任务 |
| --- | --- |
| 1.1 | Task 1 |
| 1.2 | Task 2 |
| 1.3 | Task 3 |
| 1.4 | Task 4 |
| 1.5 | Task 5 |
| 2.1 | Task 6 |
| 2.2 | Task 7 |
| 2.3 | Task 8 |
| 2.4 | Task 9 |
| 2.5 | Task 10 |
| 3.1 | Task 11 |
| 3.2 | Task 12 |
| 3.3 | Task 13 |
| 3.4 | Task 14 |
| 4.1 | Task 15 |
| 4.2 | Task 16 |
| 4.3 | Task 17 |
| 4.4 | Task 18 |
