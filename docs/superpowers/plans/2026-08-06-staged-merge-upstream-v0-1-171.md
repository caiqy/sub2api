---
change: staged-merge-upstream-v0-1-171
design-doc: docs/superpowers/specs/2026-08-06-staged-merge-upstream-v0-1-171-design.md
base-ref: 16c07d8064b0b4604e9f47ef782e7d29534402d3
---

# 分段合并上游 v0.1.173 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: 使用 `subagent-driven-development`（推荐）或 `executing-plans` 逐任务执行；所有任务使用 checkbox 跟踪。既有 Comet 配置为 branch + subagent-driven-development + TDD + thorough review。

**目标：**保留已完成的 `v0.1.170`/`v0.1.171` 历史证据，继续把 `v0.1.172`、`v0.1.173` 合为第三、第四个可审计纯 merge 节点，保留本地能力和 migration identity，最终将版本更新为 `0.1.173.1`。

**实施结构：**Tasks 1-18 是已完成的 170/171 历史阶段，不重复执行。Tasks 19-25 固定 172/173 manifest，创建并封闭第三个纯 merge；Tasks 26-32 创建第四个纯 merge，再按 Grok auth/mapping、Grok gateway/scheduler、Channel Monitor V2、pricing/schema/frontend 做 TDD 兼容审查；Tasks 33-35 用最终全门禁、四段拓扑、migration identity 和 thorough review 闭合。

**技术栈：**Git merge、Go、Ent、Wire、PostgreSQL/Testcontainers、pnpm/Vitest、PowerShell、OpenSpec。

## 全局约束

- immutable source base 固定为 `16c07d8064b0b4604e9f47ef782e7d29534402d3`，其运行版本必须为 `0.1.169.3`。该提交是已归档 lint remediation 合入后的 `main`；执行位置的 `$executionBase` 可是仅含当前 change 产物和已审查基线保护测试的后代，source base 必须是其祖先。
- 四段依次为 `v0.1.170@c043c24774228ba891ddf90d783aa6dc7d0855b5`、`v0.1.171@f0e7a9c7a23a7d02fb159b62fa809621eb0475a6`、`v0.1.172@155c494964c3ea6ecc31f52679525c1034bf0f16`、`v0.1.173@29009f0b2ea14edf3b11ae2564fb617ff91a03b4`。不合入 `v0.1.173` 之后的 `upstream/main`，发现更高正式 tag 时停止并更新 OpenSpec。
- 第三、四段唯一 merge 入口分别是 `git merge --no-ff --no-commit v0.1.172`、`git merge --no-ff --no-commit v0.1.173`。每个 merge commit 只承载目标上游树和完成冲突融合必需的路径，第二父必须是对应固定 peeled SHA；后续语义修复不得混入 merge commit。
- 172/173 merge 与兼容阶段保持 `backend/cmd/server/VERSION=0.1.171.1`；第四段全部闭合后才在独立提交中改为 `0.1.173.1`。
- 所有暂存必须使用显式路径，禁止 `git add .`。`schema`/provider/manifest 源与相应 Ent/Wire、`go.sum` 或前端 lockfile 输出同一提交；不得手工编辑生成输出或 lockfile。
- migration identity 以完整文件名为准。最终必须保留双方 191、双方 192、193、UsageLog 194/195、Channel Monitor 194-206 和 Grok pricing 217-220；同号不同名文件共存，不得重命名或改写已发布 migration。
- 用户最终裁决：新购、用户手动重置、管理员手动重置均以实际操作时刻作为日窗口锚点，后续按该锚点每 24 小时推进；不得采用 172 的配置时区 midnight 滚动语义。
- 用户最终裁决：`grok_cross_client_model_map_enabled` 默认关闭；设置缺失/空值不得加入 `gpt-*`/`codex-*`/`o*`/`claude-*` wildcard，显式 true 与账号显式 `model_mapping` 仍有效。Grok 密码授权硬禁用；Channel Monitor 默认 V1，V2 显式切换且互斥，普通用户吞吐默认隐藏。
- 任何冲突若会改变用户可见语义而不能同时保留上游和本地契约，记录 ours/theirs/影响后立即停在当前未提交阶段，等待用户决定；不得预先选择 ours、theirs 或静默删除一方行为。
- Docker/Testcontainers 只在本机条件可用时执行。不可用或目标测试明确 `SKIP` 时只能记录为 `unverified`，列出原因和受影响契约；不得访问远程服务器补验。`exit 0`、包级 `ok`、`no tests to run` 均不构成 integration PASS。
- 不执行 push、tag、release、GitHub Actions、镜像构建/发布、deploy、服务器、数据库、Redis 或 Nginx 操作。
- 每个 Task 都可能在新的 PowerShell/subagent 会话中执行；不得依赖上一 Task 的变量、函数、当前目录或临时文件。每个 Task 开始时必须重新执行下方“统一检查命令”中的 layout/context/helper 初始化，再执行本 Task 使用的命令。
- 每个 Task 的实现、验证和所选 review gate 通过后，协调者同时把 plan 中唯一 Task checkbox 和映射的 OpenSpec checkbox 从 `[ ]` 改为 `[x]`，以 `git add -f` 显式暂存这两个文件并用 `docs: record SDD progress` 独立提交，然后分别运行 `comet state task-checkoff` 精确验证两段唯一任务文本。plan/tasks 不得混入 merge、能力簇、版本或证据提交。
- `.comet/current-change.json`、change 内 `.comet.yaml`、handoff 和 `subagent-progress.md` 属于 Comet runtime/归档状态，不得混入业务、merge 或进度提交；由 Comet 持续维护并在最终归档边界处理。
- 所有原生命令（Git、Go、make、pnpm、Docker、Comet/OpenSpec adapter）都必须立即检查退出码。除明确允许 merge 冲突、`Invoke-ExpectedRed` 要求并留存的测试 RED、`git grep` 无匹配、Docker 不可用或目标 integration SKIP 的分支外，任何非零退出码立即停止；不得让后续成功命令覆盖失败码。
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
| `backend/cmd/server/VERSION` | Task 15 的 `0.1.171.1` 是中间 fork 版本；仅 Task 33 改为最终 `0.1.173.1`。 |

## 统一检查命令

以下 PowerShell 片段在每个 Task 的独立执行会话中重新运行。它们不创建分支/worktree，也不修改 Comet runtime state。

```powershell
$sourceBase = '16c07d8064b0b4604e9f47ef782e7d29534402d3'
$extensionPlanningBase = '77a9a548b26ff3290339fefce4c7ac48a7d9fbe8'
$tag170 = 'c043c24774228ba891ddf90d783aa6dc7d0855b5'
$tag170Object = '60286d35e4b6dc6851ab69f890c2d1b7b7a3bcb8'
$tag171 = 'f0e7a9c7a23a7d02fb159b62fa809621eb0475a6'
$tag171Object = 'afd154b92aac36c6dafb1fa8e181ca827c78c465'
$tag172 = '155c494964c3ea6ecc31f52679525c1034bf0f16'
$tag172Object = '61ba94d2e85a00ba639fc870b91946b1bd2f990d'
$tag173 = '29009f0b2ea14edf3b11ae2564fb617ff91a03b4'
$tag173Object = '9e2a27ad39201a14074982bae331c4610161586a'
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
$requiredMigrationSources171 = [ordered]@{
    '191_passkey_credentials.sql' = $sourceBase
    '191_subscription_quota_advance_receipts.sql' = $sourceBase
    '192_subscription_cache_invalidation_outbox.sql' = $sourceBase
    '192_group_profit_control.sql' = $tag170
    '193_group_profit_control_auth_cache_invalidation.sql' = $tag170
}
$requiredMigrationSources172 = [ordered]@{}
foreach ($entry in $requiredMigrationSources171.GetEnumerator()) {
    $requiredMigrationSources172[$entry.Key] = $entry.Value
}
$requiredMigrationSources172['194_add_usage_log_upstream_response_model.sql'] = $tag172
$requiredMigrationSources172['195_add_usage_log_upstream_model_mismatch_index_notx.sql'] = $tag172
$requiredMigrationSourcesFinal = [ordered]@{}
foreach ($entry in $requiredMigrationSources172.GetEnumerator()) {
    $requiredMigrationSourcesFinal[$entry.Key] = $entry.Value
}
$requiredMigrationSourcesFinal['194_channel_monitor_v2.sql'] = $tag173
$requiredMigrationSourcesFinal['195_channel_monitor_mode.sql'] = $tag173
$requiredMigrationSourcesFinal['196_channel_monitor_v2_ignored_error_categories.sql'] = $tag173
$requiredMigrationSourcesFinal['197_channel_monitor_v2_seed_popular_models.sql'] = $tag173
$requiredMigrationSourcesFinal['198_channel_monitor_v2_health_thresholds.sql'] = $tag173
$requiredMigrationSourcesFinal['199_channel_monitor_v2_fixed_rollups.sql'] = $tag173
$requiredMigrationSourcesFinal['200_channel_monitor_v2_rollup_permissions.sql'] = $tag173
$requiredMigrationSourcesFinal['201_channel_monitor_v2_refresh_5m.sql'] = $tag173
$requiredMigrationSourcesFinal['202_channel_monitor_v2_full_table_permissions.sql'] = $tag173
$requiredMigrationSourcesFinal['203_channel_monitor_v2_default_ignore_and_cache.sql'] = $tag173
$requiredMigrationSourcesFinal['204_channel_monitor_hide_throughput.sql'] = $tag173
$requiredMigrationSourcesFinal['205_channel_monitor_v2_reset_factory_cache_thresholds.sql'] = $tag173
$requiredMigrationSourcesFinal['206_channel_monitor_v2_privacy_defaults.sql'] = $tag173
$requiredMigrationSourcesFinal['217_group_video_model_prices.sql'] = $tag173
$requiredMigrationSourcesFinal['218_group_audio_voice_pricing.sql'] = $tag173
$requiredMigrationSourcesFinal['219_group_search_price_per_1k.sql'] = $tag173
$requiredMigrationSourcesFinal['220_clear_non_grok_video_generation_config.sql'] = $tag173

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
        throw "staged allowlist mismatch for ${Message}: $($staged -join ', ')"
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

function Invoke-ExpectedRed {
    param(
        [Parameter(Mandatory)][string]$Label,
        [Parameter(Mandatory)][scriptblock]$Command,
        [Parameter(Mandatory)][string]$LogPath,
        [Parameter(Mandatory)][string]$ExpectedFailPattern
    )
    $output = @(& $Command 2>&1)
    $exitCode = $LASTEXITCODE
    $output | Set-Content -LiteralPath $LogPath
    if ($exitCode -eq 0) { throw "$Label unexpectedly passed; expected RED" }
    if (@($output | Where-Object { $_ -match $ExpectedFailPattern }).Count -eq 0) {
        throw "$Label failed without the expected target test failure; inspect $LogPath"
    }
    return @{ Label = $Label; ExitCode = $exitCode; Log = $LogPath }
}

function Assert-AnnotatedTagMergeHead {
    param(
        [Parameter(Mandatory)][string]$TagName,
        [Parameter(Mandatory)][string]$TagObject,
        [Parameter(Mandatory)][string]$PeeledCommit
    )
    # Git keeps the annotated tag object in MERGE_HEAD; the resulting commit peels it for parent 2.
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
| Grok SSO/refresh-token、默认模型、跨客户端 mapping 和密码授权 | `grok_oauth_*`、`pkg/xai/models.go`、setting parse/update、account mapping；默认无 wildcard，显式开启即时生效，密码 API 固定拒绝。 |
| Grok media/Voice/search、free gate、team+model 冷却和调度阈值 | `grok_media*`、`grok_audio*`、`gateway_web_search.go`、`grok_free_quota_gate.go`、scheduler/ratelimit；保持 sticky/failover、body、usage 和恢复语义。 |
| Channel Monitor V2 与用户隐私 | V1/V2 runner、aggregation/repository/API、`features/channel-monitor-v2`；默认 V1、runner 互斥、admin 完整指标、普通用户吞吐脱敏。 |
| Grok pricing 与 173 migrations | group video/audio/search schema、Ent、frontend price matrix、194-206/217-220；同号 migration 共存，220 先备份且只清理非 Grok/非 composite 视频残值。 |

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
foreach ($entry in $requiredMigrationSources171.GetEnumerator()) {
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

**Verify remediation 1/3：**最终审查确认 OAuth 启动和 passkey 登录的 action gate 在 Turnstile-only 配置下错误放行，违反本任务及 OpenSpec 3.3 的互斥 provider fail-closed 契约。以 TDD 补 backend/frontend Turnstile proof 流转并复用既有 provider 分派；不得改写用户已裁决的管理员精确重置时间语义。

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

- [x] Task 15: 一次更新最终 VERSION

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

- [x] Task 16: 在最终 source HEAD 重跑完整本机门禁

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

- [x] Task 17: 验证拓扑、migration identity 和最终 Docker 结果

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
foreach ($entry in $requiredMigrationSources171.GetEnumerator()) {
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

- [x] Task 18: 完成能力专项 review 和最终验证报告

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

## v0.1.172 扩展任务

### Task 19: 固定 v0.1.172/v0.1.173 manifest、重叠面和新基线

- [x] Task 19: 固定 v0.1.172/v0.1.173 manifest、重叠面和新基线

**映射 OpenSpec：**5.1

**文件：**修改 build ledger；不修改产品代码。Comet checkpoint 由协调者使用独立 runtime 提交维护。

**接口：**
- Consumes: 已验证 171 HEAD、`$tag171`、固定远端 `upstream=https://github.com/Wei-Shaw/sub2api`。
- Produces: `$tag172`/`$tag172Object`、`$tag173`/`$tag173Object`、四 tag 严格祖先链、172 的 208/113、173 的 352/初步 138 与 GitHub 300-file 截断说明、第三/四阶段能力矩阵和可复现 171 baseline。

**步骤：**

1. 重新运行统一检查命令与 `Assert-CleanGate`，固定扩展 execution base，拒绝 extension range 中的 merge commit，并确认 `77a9a548b26ff3290339fefce4c7ac48a7d9fbe8..HEAD` 逐父只含本次 173 规划文档和独立 Comet runtime checkpoint：

```powershell
Assert-CleanGate
git merge-base --is-ancestor $extensionPlanningBase HEAD
if ($LASTEXITCODE -ne 0) { throw '173 planning HEAD does not descend from fixed 77a9a548b extension base' }
$extensionExecutionBase = (git rev-parse HEAD).Trim()
$extensionMerges = @(git rev-list --merges "$extensionPlanningBase..$extensionExecutionBase")
if ($LASTEXITCODE -ne 0) { throw 'cannot inspect 173 planning merge history' }
if ($extensionMerges.Count -ne 0) { throw "merge commits are not allowed in 173 planning range: $($extensionMerges -join ', ')" }
$allowed173PlanningPaths = @(
    "$changeDir/proposal.md",
    "$changeDir/design.md",
    "$changeDir/specs/upstream-release-sync/spec.md",
    "$changeDir/tasks.md",
    "$changeDir/.comet.yaml",
    "$changeDir/.comet/run-state.json",
    "$changeDir/.comet/state-events.jsonl",
    "$changeDir/.comet/subagent-progress.md",
    "$changeDir/.comet/trajectory.jsonl",
    $designDoc,
    $planFile
)
$extensionHistoryPaths = @(git log -m --format= --name-only "$extensionPlanningBase..$extensionExecutionBase" | Where-Object { $_.Trim() } | Sort-Object -Unique)
if ($LASTEXITCODE -ne 0) { throw 'cannot enumerate 173 planning history paths' }
$unexpectedExtensionPaths = @($extensionHistoryPaths | Where-Object { $_ -notin $allowed173PlanningPaths })
if ($unexpectedExtensionPaths.Count -ne 0) { throw "173 planning history touched unexpected paths: $($unexpectedExtensionPaths -join ', ')" }
if ((Get-Content -Raw backend/cmd/server/VERSION).Trim() -ne '0.1.171.1') { throw 'unexpected extension baseline VERSION' }
```

2. fetch 并固定两个新增 tag 身份、祖先链和 latest release：

```powershell
Invoke-CheckedNative 'fetch upstream tags for v0.1.172' { git fetch upstream --prune --tags }
$expectedTags = [ordered]@{
    'v0.1.172' = @{ Object = $tag172Object; Commit = $tag172 }
    'v0.1.173' = @{ Object = $tag173Object; Commit = $tag173 }
}
foreach ($entry in $expectedTags.GetEnumerator()) {
    $actualObject = (git rev-parse "refs/tags/$($entry.Key)").Trim()
    $actualCommit = (git rev-parse "$($entry.Key)^{}").Trim()
    if ($actualObject -ne $entry.Value.Object -or $actualCommit -ne $entry.Value.Commit) {
        throw "$($entry.Key) identity mismatch: object=$actualObject commit=$actualCommit"
    }
}
git merge-base --is-ancestor $tag171 $tag172
if ($LASTEXITCODE -ne 0) { throw 'v0.1.172 is not a strict descendant of v0.1.171' }
git merge-base --is-ancestor $tag172 $tag173
if ($LASTEXITCODE -ne 0) { throw 'v0.1.173 is not a strict descendant of v0.1.172' }
$latestRelease = (gh api repos/Wei-Shaw/sub2api/releases/latest --jq '.tag_name').Trim()
if ($LASTEXITCODE -ne 0 -or $latestRelease -ne 'v0.1.173') { throw "latest official release is $latestRelease" }
$formalTags = @(git for-each-ref refs/tags --merged=upstream/main --format='%(refname:short)' |
    Where-Object { $_ -match '^v0\.1\.\d+$' } |
    Sort-Object { [version]$_.Substring(1) } -Descending)
if ($LASTEXITCODE -ne 0 -or $formalTags.Count -eq 0 -or $formalTags[0] -ne 'v0.1.173') {
    throw "highest formal upstream tag is $(if ($formalTags.Count -gt 0) { $formalTags[0] } else { '<none>' })"
}
```

3. 重算文件面并写入 ledger；release changed-file 数量变化必须停止。173 overlap 在 172 尚未合入时只记录 discovery snapshot，不作为硬门禁；Task 25 在 172 成为祖先后重算精确值：

```powershell
$files172 = @(git diff --name-only "$tag171..$tag172")
if ($LASTEXITCODE -ne 0 -or $files172.Count -ne 208) { throw "unexpected v0.1.172 file count: $($files172.Count)" }
$localAfter171 = @(git diff --name-only "$tag171...HEAD")
if ($LASTEXITCODE -ne 0) { throw 'cannot enumerate local delta after v0.1.171' }
$overlap172 = @($files172 | Where-Object { $localAfter171 -contains $_ } | Sort-Object -Unique)
if ($overlap172.Count -ne 113) { throw "unexpected v0.1.172 overlap count: $($overlap172.Count)" }
$files172 | Set-Variable -Name files172Manifest
$overlap172 | Set-Variable -Name overlap172Manifest
$files173 = @(git diff --name-only "$tag172..$tag173")
if ($LASTEXITCODE -ne 0 -or $files173.Count -ne 352) { throw "unexpected v0.1.173 file count: $($files173.Count)" }
$preMergeLocalAgainst172 = @(git diff --name-only "$tag172...HEAD")
if ($LASTEXITCODE -ne 0) { throw 'cannot enumerate preliminary local delta against v0.1.172' }
$preliminaryOverlap173 = @($files173 | Where-Object { $preMergeLocalAgainst172 -contains $_ } | Sort-Object -Unique)
$files173 | Set-Variable -Name files173Manifest
$preliminaryOverlap173 | Set-Variable -Name preliminaryOverlap173Manifest
```

4. 在 merge 前重跑 171 baseline，证明新增失败可归属 172：

```powershell
Invoke-CheckedNative 'pre-172 baseline make test' { make test }
Invoke-CheckedNative 'pre-172 baseline build' { make 'VERSION=0.1.171.1' 'SHELL=D:/scoop/shims/bash.exe' build }
Invoke-CheckedNative 'pre-172 backend lint' { Push-Location backend; try { golangci-lint run ./... } finally { Pop-Location } }
Assert-NoConflictArtifacts
```

5. ledger 记录旧 Verify 只绑定至 171、`$extensionPlanningBase`/`$extensionExecutionBase` 与规划/runtime checkpoint 路径清单、Docker/cgo 现状、两个新增 tag manifest、172 的 208/113、173 的 352/初步 138 完整清单、GitHub Compare API 的 300-file 截断来源和八个能力簇；仅提交 ledger：

```powershell
Commit-NamedPaths -Message 'docs: record v0.1.173 merge baseline' -Paths @($buildLedger)
```

**提交边界：**`docs: record v0.1.173 merge baseline` 只含 build ledger。**检查点：**tag、latest release、祖先链、172 的 208/113、173 的 352 或 baseline 任一不符均阻塞 merge。

### Task 20: 创建纯 v0.1.172 merge 节点

- [x] Task 20: 创建纯 v0.1.172 merge 节点

**映射 OpenSpec：**5.2

**文件：**由真实 merge 冲突决定；只允许 172 tag 树和完成冲突融合必需的路径。禁止混入 ledger、plan/tasks、兼容修复或 VERSION bump。

**接口：**
- Consumes: Task 19 固定的 `$tag172`、第三阶段冲突台账与 `VERSION=0.1.171.1`。
- Produces: 唯一 first-parent merge commit，第二父精确为 `$tag172`。

**步骤：**

1. 记录 pre-merge HEAD 和预测冲突后启动唯一 merge：

```powershell
$pre172MergeHead = (git rev-parse HEAD).Trim()
git merge --no-ff --no-commit v0.1.172
$mergeExit = $LASTEXITCODE
$unmerged172 = @(git diff --name-only --diff-filter=U)
$mergeHead = (@(git rev-parse -q --verify MERGE_HEAD 2>$null) -join '').Trim()
if ($mergeExit -ne 0 -and $mergeHead -eq '') { throw "v0.1.172 merge failed before conflict state: $mergeExit" }
Assert-AnnotatedTagMergeHead -TagName 'v0.1.172' -TagObject $tag172Object -PeeledCommit $tag172
if ($mergeExit -notin @(0,1)) { throw "v0.1.172 merge failed fatally: $mergeExit" }
if ($mergeExit -eq 1 -and $unmerged172.Count -eq 0) { throw 'v0.1.172 merge failed without resolvable conflicts' }
```

2. 对 `$unmerged172` 每个路径记录 ours/theirs/融合结果。适用硬规则：VERSION 保留 `0.1.171.1`；OAuth pending 安全 guard 必须存在；订阅日窗口保留实际时刻锚点；194/195 与既有 migrations 共存；schema/provider 源先融合再生成。无法同时保留且未被设计裁决的用户可见语义立即暂停。

3. 逐个显式 `git add -- <resolved-path>`，不得 `git add .`。若 schema/Wire 源有冲突，运行生成并把源与输出留在同一 merge：

```powershell
Invoke-CheckedNative 'v0.1.172 merge generate' { make -C backend generate }
git add -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'failed to stage v0.1.172 Ent/Wire outputs' }
$stagedGoMod172 = @(git diff --cached --name-only -- backend/go.mod)
if ($stagedGoMod172.Count -gt 0) {
    Push-Location backend; try { Invoke-CheckedNative 'v0.1.172 go mod tidy' { go mod tidy } } finally { Pop-Location }
    git add -- backend/go.mod backend/go.sum
    if ($LASTEXITCODE -ne 0) { throw 'failed to stage v0.1.172 Go module outputs' }
}
$stagedFrontendManifest172 = @(git diff --cached --name-only -- frontend/package.json)
if ($stagedFrontendManifest172.Count -gt 0) {
    Invoke-CheckedNative 'v0.1.172 pnpm lock refresh' { pnpm --dir frontend install --lockfile-only }
    git add -- frontend/package.json frontend/pnpm-lock.yaml
    if ($LASTEXITCODE -ne 0) { throw 'failed to stage v0.1.172 frontend manifest outputs' }
}
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go backend/go.mod backend/go.sum frontend/package.json frontend/pnpm-lock.yaml
if ($LASTEXITCODE -ne 0) { throw 'v0.1.172 generated/tool outputs remain unstaged' }
if ((Get-Content -Raw backend/cmd/server/VERSION).Trim() -ne '0.1.171.1') { throw 'merge changed intermediate VERSION' }
Assert-NoConflictArtifacts
Invoke-CheckedNative 'v0.1.172 merge compile' { Push-Location backend; try { go test ./internal/handler/... ./internal/service/... ./internal/repository/... -run '^$' } finally { Pop-Location } }
```

4. 创建纯 merge 并验证父节点：

```powershell
git commit -m 'merge: integrate upstream v0.1.172'
if ($LASTEXITCODE -ne 0) { throw 'v0.1.172 merge commit failed' }
$merge172Commit = (git rev-parse HEAD).Trim()
if ((git rev-parse 'HEAD^1').Trim() -ne $pre172MergeHead) { throw 'v0.1.172 first parent mismatch' }
if ((git rev-parse 'HEAD^2').Trim() -ne $tag172) { throw 'v0.1.172 second parent mismatch' }
```

**提交边界：**唯一提交 `merge: integrate upstream v0.1.172`。**检查点：**merge review 必须确认无后续能力修复、报告或 final VERSION 混入。

### Task 21: 关闭 OAuth pending、captcha 和认证兼容面

- [x] Task 21: 关闭 OAuth pending、captcha 和认证兼容面

**映射 OpenSpec：**5.3

**文件：**重点审查 `backend/internal/handler/auth_oauth_pending_flow.go`、对应测试、Tencent captcha repository/service/settings/CSP，以及 `frontend/src/components/CaptchaChallenge.vue`、`PendingOAuthCreateAccountForm.vue` 和 auth views/tests。

**接口：**
- Consumes: `ExchangePendingOAuthCompletion`、`pendingOAuthCompletionCanIssueTokenPair`、`CaptchaChallenge.verifyAction(): Promise<ActionCaptchaResult | null>`。
- Produces: 非终态 pending session 不绑定身份；三 provider 互斥 fail-closed；Tencent `region` 全认证入口透传；Turnstile token 生命周期不回退。

**步骤：**

1. 运行 0day 与本地 auth/captcha tests。任何失败先保存 RED，不得先改生产代码：

```powershell
Invoke-CheckedNative 'OAuth takeover RED/protection' { Push-Location backend; try { go test ./internal/handler -run 'TestExchangePendingOAuthCompletion(ChoiceStateDoesNotBindIdentity|.*)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'captcha backend protection' { Push-Location backend; try { go test ./internal/service ./internal/repository ./internal/server/middleware -run '(Captcha|Turnstile|Tencent|Aliyun|OAuth|Passkey)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'captcha frontend protection' { pnpm --dir frontend exec vitest run src/components/__tests__/CaptchaChallenge.spec.ts src/components/__tests__/TencentCaptchaGate.spec.ts src/components/auth/__tests__/PendingOAuthCreateAccountForm.spec.ts src/views/auth/__tests__/TencentCaptchaActionGate.spec.ts src/views/auth/__tests__/TencentCaptchaForgotPassword.spec.ts src/views/auth/__tests__/EmailVerifyView.spec.ts }
```

2. 若 0day test 失败，最小 guard 必须位于 adoption decision/apply 前，保持邀请、补邮箱和 bind-login 早返回：

```go
if !canIssueTokenPair && !strings.EqualFold(strings.TrimSpace(session.Intent), oauthIntentBindCurrentUser) {
	response.Success(c, payload)
	return
}
```

3. 若 captcha compatibility 失败，保留 `CaptchaChallenge` 的 `else-if` 互斥与 Turnstile cache reset，只增加 Tencent region 输入：

```ts
tencentRegion?: string
```

```vue
<TencentCaptchaGate
  v-else-if="tencentEnabled && tencentAppId"
  ref="tencentRef"
  :app-id="tencentAppId"
  :region="tencentRegion === 'intl' ? 'intl' : 'cn'"
/>
```

4. 重跑上述 tests、backend handler/service/repository focused gate、frontend typecheck/ESLint。只有真实修复时提交：

```powershell
Invoke-CheckedNative 'auth compatibility GREEN' { Push-Location backend; try { go test ./internal/handler ./internal/service ./internal/repository -run '(PendingOAuth|Captcha|Turnstile|Tencent|Aliyun)' -count=1 } finally { Pop-Location } }
$task21ChangedPaths = @(git diff --name-only -- backend/internal/handler/auth_oauth_pending_flow.go backend/internal/handler/auth_oauth_pending_flow_test.go backend/internal/repository backend/internal/server/middleware backend/internal/service frontend/src/components/CaptchaChallenge.vue frontend/src/components/TencentCaptchaGate.vue frontend/src/components/auth frontend/src/views/auth frontend/src/api/auth.ts frontend/src/types/index.ts)
if ($task21ChangedPaths.Count -gt 0) { Commit-NamedPaths -Message 'fix: preserve auth security after v0.1.172' -Paths $task21ChangedPaths }
```

**提交边界：**OAuth/captcha/backend/frontend 直接测试可同提交；无真实 diff 不创建空提交。**检查点：**账号接管测试与三 provider auth matrix 必须全部通过。

### Task 22: 保留实际时刻额度窗口并融合 billing 修复

- [x] Task 22: 保留实际时刻额度窗口并融合 billing 修复

**映射 OpenSpec：**5.4

**文件：**`backend/internal/service/subscription_service.go`、`user_subscription.go`、`subscription_reset_quota_test.go`、`user_subscription_daily_quota_test.go`、`subscription_monthly_window_test.go`、usage billing quantize 源/tests；repository receipt/outbox/cache 仅在真实回归时修改。

**接口：**
- Consumes: `renewedSubscriptionTerm`、`AdminResetQuota`、`automaticWindowStartAt`、`CheckAndResetWindows`、`DailyResetTime`。
- Produces: 所有手动/新购日窗口用实际时刻，自动日窗口每 24 小时推进；一日卡不重复发放；金额按 `NUMERIC(20,8)` 量化。

**步骤：**

1. merge 后先运行精确锚点 tests，预期上游 midnight 实现导致 RED；必须保存具体失败断言：

```powershell
$subscriptionRedLog = Join-Path $env:TEMP 'sub2api-v0.1.172-subscription-anchor-red.log'
$subscriptionRed = Invoke-ExpectedRed -Label 'subscription exact-anchor RED' -LogPath $subscriptionRedLog -ExpectedFailPattern '^--- FAIL: (TestAdminResetQuota_ResetBoth|TestAssignOrExtendSubscription_ExpiredDailyCardStartsNewOneTimeQuota|TestUserSubscriptionNeedsDailyReset_MultiDaySubscriptionStillRefreshes|TestAutomaticWindowPreservesPersistedManualAnchor)' -Command { Push-Location backend; try { go test -tags unit ./internal/service -run '(TestAdminResetQuota_ResetBoth|TestAssignOrExtendSubscription_ExpiredDailyCardStartsNewOneTimeQuota|TestUserSubscriptionNeedsDailyReset_MultiDaySubscriptionStillRefreshes|TestAutomaticWindowPreservesPersistedManualAnchor)' -count=1 } finally { Pop-Location } }
$subscriptionRed
```

2. 确认测试断言固定非 midnight 时刻，禁止把期望改成 `StartOfDay`：

```go
resetAt := time.Date(2026, 7, 1, 10, 37, 42, 123, time.UTC)
require.Equal(t, resetAt, stub.windowStart)
require.Equal(t, resetAt, *result.DailyWindowStart)
```

3. 最小恢复本地窗口实现；不得删除 172 的 billing quantization、锁、receipt/outbox/cache 修复：

```go
renewed.DailyWindowStart = &startsAt
windowStart := s.now()
version, err := s.resetUsageWindowsWithVersion(ctx, sub.ID, resetDaily, resetWeekly, resetMonthly, windowStart)
```

```go
if windowStart, ok := sub.automaticWindowStartAt(sub.DailyWindowStart, 24*time.Hour, now); !sub.HasOneTimeDailyQuota() && ok {
	resetDaily = true
	dailyWindowStart = &windowStart
}
```

4. 验证 GREEN 和 172 billing 修复：

```powershell
Invoke-CheckedNative 'subscription/billing GREEN' { Push-Location backend; try { go test -tags unit ./internal/service -run '(Subscription|Quota|UsageBilling|Quantize|Refund)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'subscription repository contracts' { Push-Location backend; try { go test ./internal/repository -run '(Subscription|Quota|Receipt|Outbox|UsageLog)' -count=1 } finally { Pop-Location } }
```

5. 提交实际修复和直接 tests：

```powershell
$task22ChangedPaths = @(git diff --name-only -- backend/internal/service/subscription_service.go backend/internal/service/user_subscription.go backend/internal/service/subscription_reset_quota_test.go backend/internal/service/user_subscription_daily_quota_test.go backend/internal/service/subscription_monthly_window_test.go backend/internal/service/usage_billing.go backend/internal/service/usage_billing_quantize_test.go backend/internal/repository)
if ($task22ChangedPaths.Count -gt 0) { Commit-NamedPaths -Message 'fix: preserve subscription billing after v0.1.172' -Paths $task22ChangedPaths }
```

**提交边界：**只含 subscription/billing/persistence 能力簇。**检查点：**精确锚点、一日卡、量化、事务锁和缓存失效任一失败都阻塞。

### Task 23: 融合 gateway、transport 和 protocol 修复

- [x] Task 23: 融合 gateway、transport 和 protocol 修复

**映射 OpenSpec：**5.5

**文件：**重点审查 OpenAI/Codex gateway、WS、`gateway_upstream_response.go`、`http_upstream.go`、`pkg/proxyutil`、Responses→Anthropic、count_tokens、Grok、rate-limit/image cooldown 及对应 tests。

**接口：**
- Consumes: `ForwardResult`、`UpstreamFailoverError`、request-body handle、sticky/final account/model、usage record inputs。
- Produces: pre-output capacity 可切号；post-output 错误仅改写为客户端可重试；每次 attempt 重置 response-model observer；dial/TLS/SOCKS5 有界；body/usage 只处理一次。

**步骤：**

1. 运行 172 upstream tests 与本地保护集，保留任何 RED：

```powershell
Invoke-CheckedNative 'gateway 172 focused RED/protection' { Push-Location backend; try { go test -tags unit ./internal/service ./internal/repository ./internal/pkg/apicompat ./internal/pkg/proxyutil -run '(Capacity|Overload|Codex|Prewarm|CountTokens|Grok|Cooldown|Response|Anthropic|Dial|Timeout|Body|Sticky|Failover)' -count=1 } finally { Pop-Location } }
```

2. 审查必须满足的分支，不以注释替代行为：pre-output `server_is_overloaded`/`slow_down` 返回 failover error 且 result=nil；已输出响应不切号、不重复计费；observer 在每次 attempt 重新创建；timeout 同时覆盖 direct、TLS、SOCKS5；count_tokens HTML 403 回退本地估算且不冷却 OAuth 账号。

3. 用现有本地 AST/行为 tests 验证所有 `OpenAIRecordUsageInput` 仍携带 `QuotaPlatform`，body handle 在最终成功/失败路径恰好释放一次：

```powershell
Invoke-CheckedNative 'gateway local contract GREEN' { Push-Location backend; try { go test -tags unit ./internal/handler ./internal/service -run '(OpenAIRecordUsageInputsCarryQuotaPlatform|RequestBody|BodyRetention|Sticky|Failover|RecordUsage|Capacity|Prewarm)' -count=1 } finally { Pop-Location } }
```

4. 只对真实兼容回归做最小修改并重跑两个 focused gate、backend lint。若有 diff：

```powershell
$task23ChangedPaths = @(git diff --name-only -- backend/internal/handler backend/internal/repository/http_upstream.go backend/internal/repository/http_upstream_dial_timeout_test.go backend/internal/pkg/apicompat backend/internal/pkg/proxyutil backend/internal/service)
if ($task23ChangedPaths.Count -gt 0) { Commit-NamedPaths -Message 'fix: preserve gateway transport after v0.1.172' -Paths $task23ChangedPaths }
```

**提交边界：**gateway/transport/protocol 及直接 tests；不得混入 UsageLog schema/UI（归 Task 24）。**检查点：**最终账号/模型、body 生命周期、sticky、错误边界或单次计费不明确时不得通过 review。

### Task 24: 闭合 response-model audit、194/195 和前端展示

- [x] Task 24: 闭合 response-model audit、194/195 和前端展示

**映射 OpenSpec：**5.6

**文件：**UsageLog service/schema/Ent、单条/批量/best-effort repository insert/query、admin DTO/handler、migrations 194/195、migration integration test、frontend admin usage API/types/filters/table/view/i18n/tests，以及模型广场与错误时间范围冲突路径。

**接口：**
- Produces: `UsageLog.UpstreamResponseModel *string`、`UsageLog.UpstreamModelMismatch *bool`；NULL 表示未观察，false/true 表示已比较；UI 支持 only-mismatch 筛选。
- Consumes: Task 23 在协议转换前冻结的 upstream response model 与 conflict 标记。

**步骤：**

1. 运行 observer/persistence/UI tests，任何本地融合缺口先 RED：

```powershell
Invoke-CheckedNative 'response model backend RED/protection' { Push-Location backend; try { go test -tags unit ./internal/service ./internal/repository ./internal/handler -run '(UpstreamResponseModel|UpstreamModelMismatch|PreservesRequestedAndUpstreamModels|UsageLog|UsageHandler)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'response model frontend RED/protection' { pnpm --dir frontend exec vitest run src/components/admin/usage/__tests__/UsageFilters.spec.ts src/components/admin/usage/__tests__/UsageTable.spec.ts src/views/admin/__tests__/UsageView.spec.ts }
```

2. 保持三模型字段语义，不用字符串拼接替代结构字段：

```go
type UsageLog struct {
	Model                 string
	RequestedModel        string
	UpstreamModel         *string
	UpstreamResponseModel *string
	UpstreamModelMismatch *bool
}
```

3. 从 `backend/ent/schema/usage_log.go` 生成 Ent；确认所有 SQL insert 形态同时增加 2 列/2 参数，并在提交前对当前工作树校验 172 阶段全部 7 个 migration blob：

```powershell
Invoke-CheckedNative 'UsageLog Ent generation' { make -C backend generate }
if ($requiredMigrationSources172.Count -ne 7) { throw 'unexpected v0.1.172 migration set size' }
foreach ($entry in $requiredMigrationSources172.GetEnumerator()) {
    $name = $entry.Key
    $expectedOutput = @(git rev-parse "$($entry.Value):backend/migrations/$name" 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "cannot resolve v0.1.172 authoritative migration: $name" }
    $actualOutput = @(git hash-object -- "backend/migrations/$name" 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "cannot hash v0.1.172 migration worktree file: $name" }
    $expected = ($expectedOutput -join '').Trim()
    $actual = ($actualOutput -join '').Trim()
    if ($expected -ne $actual) { throw "v0.1.172 migration identity mismatch: $name" }
}
```

4. 扩展 migration upgrade integration 的 required set 到 195；Docker 可用时要求目标 top-level test 真实 PASS，不可用时列明 194 schema、195 NOTX index、升级、幂等和 checksum 为 `unverified`。

5. 重跑 backend repository/service/handler focused tests、frontend 3 suites/typecheck/ESLint、两轮 generate 零 diff。仅有真实兼容 diff 时提交，schema 与生成输出必须同提交：

```powershell
$task24ChangedPaths = @(git diff --name-only -- backend/ent backend/internal/handler/admin backend/internal/handler/dto backend/internal/repository backend/internal/service/usage_log.go backend/internal/service/upstream_response_model.go backend/internal/service/upstream_response_model_test.go frontend/src/api/admin/usage.ts frontend/src/components/admin/usage frontend/src/i18n frontend/src/types/index.ts frontend/src/views/admin)
if ($task24ChangedPaths.Count -gt 0) { Commit-NamedPaths -Message 'fix: preserve usage audit after v0.1.172' -Paths $task24ChangedPaths }
```

**提交边界：**response-model audit 的 source/generated/frontend 与 migration integration test 闭合在一个能力提交；172 SQL blob 只来自 merge，不得手工改 Ent 或 migration。**检查点：**三态语义、59-column insert 口径、query/filter/UI 或 migration identity 任一不一致均阻塞。

### Task 25: 运行 v0.1.172 全量门禁并冻结 v0.1.173 精确矩阵

- [x] Task 25: 运行 v0.1.172 全量门禁并冻结 v0.1.173 精确矩阵

**映射 OpenSpec：**5.7、5.8

**文件：**默认只更新 build ledger；门禁发现回归时按首次能力簇派发 remediation，Task 25 保持未勾选。

**接口：**Consumes Tasks 20-24 final source HEAD；Produces 非 Docker full PASS、两轮 generate 零 diff、`gap=0` 和 Docker residual。

**步骤：**

1. 先运行四能力簇 canonical focused bundle：

```powershell
Invoke-CheckedNative 'gate auth/captcha' { Push-Location backend; try { go test ./internal/handler ./internal/service ./internal/repository ./internal/server/middleware -run '(PendingOAuth|Captcha|Turnstile|Tencent|Aliyun|Passkey)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'gate subscription/billing' { Push-Location backend; try { go test -tags unit ./internal/service ./internal/repository -run '(Subscription|Quota|UsageBilling|Quantize|Receipt|Outbox)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'gate gateway/transport' { Push-Location backend; try { go test -tags unit ./internal/handler ./internal/service ./internal/repository ./internal/pkg/apicompat ./internal/pkg/proxyutil -run '(Capacity|Codex|Prewarm|CountTokens|Grok|Cooldown|Anthropic|Dial|Timeout|RequestBody|Sticky|Failover)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'gate response model audit' { Push-Location backend; try { go test -tags unit ./internal/service ./internal/repository ./internal/handler -run '(UpstreamResponseModel|UpstreamModelMismatch|PreservesRequestedAndUpstreamModels|UsageLog|UsageHandler)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'gate frontend 172' { pnpm --dir frontend exec vitest run src/components/__tests__/CaptchaChallenge.spec.ts src/components/__tests__/TencentCaptchaGate.spec.ts src/components/auth/__tests__/PendingOAuthCreateAccountForm.spec.ts src/components/admin/usage/__tests__/UsageFilters.spec.ts src/components/admin/usage/__tests__/UsageTable.spec.ts src/views/admin/__tests__/UsageView.spec.ts src/views/auth/__tests__/TencentCaptchaActionGate.spec.ts src/views/auth/__tests__/TencentCaptchaForgotPassword.spec.ts src/views/auth/__tests__/EmailVerifyView.spec.ts }
```

2. 然后执行完整门禁：

```powershell
Invoke-CheckedNative 'v0.1.172 make test' { make test }
Invoke-CheckedNative 'v0.1.172 build' { make 'VERSION=0.1.171.1' 'SHELL=D:/scoop/shims/bash.exe' build }
Invoke-CheckedNative 'v0.1.172 backend lint' { Push-Location backend; try { golangci-lint run ./... } finally { Pop-Location } }
Invoke-CheckedNative 'v0.1.172 frontend lint' { pnpm --dir frontend run lint:check }
Invoke-CheckedNative 'v0.1.172 frontend typecheck' { pnpm --dir frontend run typecheck }
Invoke-CheckedNative 'v0.1.172 first generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'first v0.1.172 generate changed output' }
Invoke-CheckedNative 'v0.1.172 second generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'second v0.1.172 generate changed output' }
Assert-NoConflictArtifacts
```

3. Docker preflight 按既有 helper 执行；不可用只记 `unverified`，真实目标 integration failure 必须修复。逐行关闭第三阶段能力矩阵，`gap` 必须为 0。

4. 确认 172 已成为 HEAD 祖先且 VERSION 仍为 `0.1.171.1`，再重算第四阶段精确 overlap 和能力矩阵；不为 overlap 预设数量，但完整列表必须写入 ledger：

```powershell
git merge-base --is-ancestor $tag172 HEAD
if ($LASTEXITCODE -ne 0) { throw 'v0.1.172 is not an ancestor after stage gate' }
if ((Get-Content -Raw backend/cmd/server/VERSION).Trim() -ne '0.1.171.1') { throw '172 stage changed intermediate VERSION' }
$files173 = @(git diff --name-only "$tag172..$tag173")
if ($LASTEXITCODE -ne 0 -or $files173.Count -ne 352) { throw 'v0.1.173 manifest drifted' }
$localAfter172 = @(git diff --name-only "$tag172...HEAD")
if ($LASTEXITCODE -ne 0) { throw 'cannot enumerate local delta after v0.1.172' }
$overlap173 = @($files173 | Where-Object { $localAfter172 -contains $_ } | Sort-Object -Unique)
$overlap173
```

5. 仅提交 ledger：

```powershell
Commit-NamedPaths -Message 'docs: record v0.1.172 gates and v0.1.173 matrix' -Paths @($buildLedger)
```

**提交边界：**报告提交与 remediation 分离。**检查点：**全部非 Docker gate PASS、generate stable、第三阶段 gap=0、第四阶段精确 overlap/矩阵已记录后才能 merge 173；不得 bump VERSION。

### Task 26: 创建纯 v0.1.173 merge 节点

- [x] Task 26: 创建纯 v0.1.173 merge 节点

**映射 OpenSpec：**6.1

**文件：**由真实 merge 冲突决定；只允许 173 tag 树和完成冲突融合必需的路径。禁止混入 ledger、plan/tasks、兼容修复或 VERSION bump。

**接口：**
- Consumes: Task 25 固定的 `$tag173`、精确第四阶段冲突台账与 `VERSION=0.1.171.1`。
- Produces: 唯一 first-parent merge commit，第二父精确为 `$tag173`。

**步骤：**

1. 重新运行统一检查命令与 `Assert-CleanGate`，确认 `$tag172` 是 HEAD 祖先、`$tag173` 尚不是 HEAD 祖先，记录 pre-merge HEAD：

```powershell
git merge-base --is-ancestor $tag172 HEAD
if ($LASTEXITCODE -ne 0) { throw 'v0.1.172 stage is not closed' }
git merge-base --is-ancestor $tag173 HEAD
if ($LASTEXITCODE -eq 0) { throw 'v0.1.173 was already merged outside this task' }
$pre173MergeHead = (git rev-parse HEAD).Trim()
```

2. 启动唯一 merge 并固定 annotated tag 身份：

```powershell
git merge --no-ff --no-commit v0.1.173
$merge173Exit = $LASTEXITCODE
$unmerged173 = @(git diff --name-only --diff-filter=U)
Assert-AnnotatedTagMergeHead -TagName 'v0.1.173' -TagObject $tag173Object -PeeledCommit $tag173
if ($merge173Exit -notin @(0,1)) { throw "v0.1.173 merge failed fatally: $merge173Exit" }
if ($merge173Exit -eq 1 -and $unmerged173.Count -eq 0) { throw 'merge failed without resolvable conflicts' }
```

3. 对每个冲突记录 ours/theirs/最小融合。VERSION 保留 `0.1.171.1`；本地 subscription 锚点、scheduler/sticky/body/audit/settings/frontend 不得被删除；同号 migration 按完整文件名共存。Grok mapping/password 的 release/tag 偏差留给 Task 27 独立 RED/GREEN 修复，不把无冲突语义修复混入 merge。

4. 显式暂存每个 resolved path；融合 schema/Wire/manifest 源后用现有工具再生输出，运行 compile gate：

```powershell
Invoke-CheckedNative 'v0.1.173 merge generate' { make -C backend generate }
git add -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'failed to stage v0.1.173 Ent/Wire outputs' }
$stagedGoMod173 = @(git diff --cached --name-only -- backend/go.mod)
if ($stagedGoMod173.Count -gt 0) {
    Push-Location backend; try { Invoke-CheckedNative 'v0.1.173 go mod tidy' { go mod tidy } } finally { Pop-Location }
    git add -- backend/go.mod backend/go.sum
    if ($LASTEXITCODE -ne 0) { throw 'failed to stage v0.1.173 Go module outputs' }
}
$stagedFrontendManifest173 = @(git diff --cached --name-only -- frontend/package.json)
if ($stagedFrontendManifest173.Count -gt 0) {
    Invoke-CheckedNative 'v0.1.173 pnpm lock refresh' { pnpm --dir frontend install --lockfile-only }
    git add -- frontend/package.json frontend/pnpm-lock.yaml
    if ($LASTEXITCODE -ne 0) { throw 'failed to stage v0.1.173 frontend manifest outputs' }
}
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go backend/go.mod backend/go.sum frontend/package.json frontend/pnpm-lock.yaml
if ($LASTEXITCODE -ne 0) { throw 'v0.1.173 generated/tool outputs remain unstaged' }
if ((Get-Content -Raw backend/cmd/server/VERSION).Trim() -ne '0.1.171.1') { throw 'v0.1.173 merge changed intermediate VERSION' }
Assert-NoConflictArtifacts
Invoke-CheckedNative 'v0.1.173 merge compile' { Push-Location backend; try { go test ./internal/handler/... ./internal/service/... ./internal/repository/... -run '^$' } finally { Pop-Location } }
```

5. 审查暂存 diff 无报告、任务、runtime 和兼容修复后提交并验证父节点：

```powershell
git commit -m 'merge: integrate upstream v0.1.173'
if ($LASTEXITCODE -ne 0) { throw 'v0.1.173 merge commit failed' }
if ((git rev-parse 'HEAD^1').Trim() -ne $pre173MergeHead) { throw 'v0.1.173 first parent mismatch' }
if ((git rev-parse 'HEAD^2').Trim() -ne $tag173) { throw 'v0.1.173 second parent mismatch' }
```

**提交边界：**唯一提交 `merge: integrate upstream v0.1.173`。**检查点：**merge review 必须确认第二父、生成源归属和纯 merge 边界。

### Task 27: 固定 Grok 授权与模型映射安全默认

- [x] Task 27: 固定 Grok 授权与模型映射安全默认

**映射 OpenSpec：**6.2

**文件：**`backend/internal/pkg/xai/models.go`、`models_test.go`、`backend/internal/service/setting_parse.go`、新建 `backend/internal/service/setting_grok_mapping_test.go`、`grok_oauth_service.go`、`grok_oauth_service_test.go`、`backend/internal/handler/admin/grok_oauth_handler_test.go`、account mapping tests、`frontend/src/views/admin/SettingsView.vue`、对应 Vitest。

**接口：**
- Consumes: `xai.ModelMappingOptions`、`xai.SetRuntimeModelMappingOptions`、`SettingKeyGrokCrossClientModelMapEnabled`、`GrokOAuthService.GetCapabilities/AuthorizePassword`。
- Produces: 设置缺失/空值/false 时无跨客户端 wildcard；显式 true 时映射至配置默认模型；账号显式 mapping 优先；密码 capability 固定 false且 API 固定 403。

**步骤：**

1. 在 `setting_grok_mapping_test.go` 先写默认关闭与显式开启测试，并把 handler 中“配置 true 成功登录”改为“配置 true 仍拒绝且 OAuth client 零调用”；`grokOAuthHandlerClient.LoginWithPassword` 增加调用计数，403 响应不得回显密码：

```go
func TestParseSettingsGrokCrossClientMappingDefaultsOff(t *testing.T) {
	original := xai.RuntimeModelMappingOptions()
	t.Cleanup(func() { xai.SetRuntimeModelMappingOptions(original) })
	s := &SettingService{}
	got := s.parseSettings(map[string]string{})
	require.False(t, got.GrokCrossClientModelMapEnabled)
	_, hasGPT := xai.DefaultModelMapping()["gpt-*"]
	require.False(t, hasGPT)
}

func TestParseSettingsGrokCrossClientMappingExplicitTrue(t *testing.T) {
	original := xai.RuntimeModelMappingOptions()
	t.Cleanup(func() { xai.SetRuntimeModelMappingOptions(original) })
	s := &SettingService{}
	got := s.parseSettings(map[string]string{SettingKeyGrokCrossClientModelMapEnabled: "true"})
	require.True(t, got.GrokCrossClientModelMapEnabled)
	require.Equal(t, "grok-4.5", xai.DefaultModelMapping()["claude-*"])
}

type grokOAuthHandlerClient struct {
	passwordLoginCalls int
}

func (c *grokOAuthHandlerClient) LoginWithPassword(_ context.Context, email, _ string, _ string) (*service.GrokPasswordLoginResult, error) {
	c.passwordLoginCalls++
	return &service.GrokPasswordLoginResult{Email: email, SSOToken: "sso-from-password"}, nil
}

func TestGrokOAuthHandlerAuthorizePasswordRemainsDisabledWhenConfigTrue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oauthClient := &grokOAuthHandlerClient{}
	cfg := &config.Config{}
	cfg.Gateway.Grok.PasswordAuthEnabled = true
	oauthService := service.NewGrokOAuthService(nil, oauthClient, cfg)
	defer oauthService.Stop()
	handler := NewGrokOAuthHandler(oauthService, nil, nil, nil)
	router := gin.New()
	router.POST("/api/v1/admin/grok/oauth/password", handler.AuthorizePassword)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/grok/oauth/password", strings.NewReader(`{"email":"user@example.com","password":"super-secret"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Zero(t, oauthClient.passwordLoginCalls)
	require.NotContains(t, rec.Body.String(), "super-secret")
}
```

```powershell
$grokAuthRedLog = Join-Path $env:TEMP 'sub2api-v0.1.173-grok-auth-red.log'
$grokAuthRed = Invoke-ExpectedRed -Label 'Grok mapping/password RED' -LogPath $grokAuthRedLog -ExpectedFailPattern '^--- FAIL: (TestParseSettingsGrokCrossClientMappingDefaultsOff|TestGrokOAuthHandlerAuthorizePasswordRemainsDisabledWhenConfigTrue)' -Command { Push-Location backend; try { go test ./internal/pkg/xai ./internal/service ./internal/handler/admin -run '(GrokCrossClient|DefaultModelMapping|GrokOAuthHandlerAuthorizePassword|PasswordCapability|MessagesDispatchModel)' -count=1 } finally { Pop-Location } }
$grokAuthRed
```

预期：default-off 与 config-true-hard-disable 至少一项在原 173 tag 行为上 FAIL；不得放宽断言。

2. 最小修复初始化与解析默认，并让 password capability 忽略兼容配置：

```go
SettingKeyGrokCrossClientModelMapEnabled: "false",
```

```go
result.GrokCrossClientModelMapEnabled = settings[SettingKeyGrokCrossClientModelMapEnabled] == "true"
```

```go
func (s *GrokOAuthService) passwordAuthEnabled() bool {
	return false
}
```

3. 重跑 focused tests，并验证账号显式 mapping、设置 API round-trip、前端开关 payload 和密码入口隐藏：

```powershell
Invoke-CheckedNative 'Grok mapping/password GREEN' { Push-Location backend; try { go test ./internal/pkg/xai ./internal/service ./internal/handler/admin ./internal/server -run '(GrokCrossClient|DefaultModelMapping|ExplicitModelMapping|GrokOAuthHandlerAuthorizePassword|PasswordCapability|MessagesDispatchModel|Settings)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'Grok settings frontend GREEN' { pnpm --dir frontend exec vitest run src/views/admin/__tests__/SettingsView.spec.ts src/components/account/__tests__/CreateAccountModal.grok.spec.ts src/components/admin/account/__tests__/ReAuthAccountModal.grok.spec.ts src/composables/__tests__/useGrokOAuth.spec.ts }
```

4. backend lint、frontend typecheck/ESLint 通过后只提交实际 auth/mapping diff：

```powershell
$task27Paths = @(git diff --name-only -- backend/internal/pkg/xai backend/internal/service/setting_parse.go backend/internal/service/setting_grok_mapping_test.go backend/internal/service/grok_oauth_service.go backend/internal/service/grok_oauth_service_test.go backend/internal/handler/admin/grok_oauth_handler_test.go backend/internal/server frontend/src/views/admin/SettingsView.vue frontend/src/views/admin/__tests__/SettingsView.spec.ts frontend/src/components/account frontend/src/components/admin/account frontend/src/composables)
Commit-NamedPaths -Message 'fix: enforce Grok authorization defaults after v0.1.173' -Paths $task27Paths
```

**提交边界：**只含 Grok auth/model mapping 和直接测试。**检查点：**默认关闭、显式开启、显式账号 mapping、密码硬禁用四项都必须有行为证据。

### Task 28: 融合 Grok 媒体、Voice、搜索与网关契约

- [x] Task 28: 融合 Grok 媒体、Voice、搜索与网关契约

**映射 OpenSpec：**6.3

**文件：**`backend/internal/handler/grok_media.go`、`grok_audio.go`、`gateway_web_search.go`、`backend/internal/service/grok_media*.go`、`grok_audio*.go`、`grok_search_count.go`、`openai_gateway_grok*.go`、`openai_images.go`、`backend/internal/pkg/xai/billing.go` 及对应 tests。

**接口：**
- Consumes: Grok account model mapping、gateway request-body handle、`OpenAIRecordUsageInput`、统一 audit、sticky/failover context。
- Produces: 图片/视频/Voice/search 路由使用最终账号和模型；异步视频只在成功完成时计费；search 只计一次；body/stream 恰好释放一次。

**步骤：**

1. 运行上游新增与本地保护测试，保存任何 RED：

```powershell
Invoke-CheckedNative 'Grok media/voice/search RED/protection' { Push-Location backend; try { go test -tags unit ./internal/handler ./internal/service ./internal/pkg/xai -run '(GrokMedia|GrokAudio|Voice|Video|WebSearch|SearchCount|SearchSurcharge|ImageGeneration|RequestBody|Sticky|Failover|RecordUsage)' -count=1 } finally { Pop-Location } }
```

2. 调用链审查必须确认：multipart/JSON/media status routes 经过 SSRF/base URL 约束；异步 pending/failed 不扣费且成功不重复扣费；Voice TTS/STT/Realtime 和 custom voices 不落敏感 payload；JSON/SSE search count 去重；每个 usage 输入携带最终 model/account、media/search 维度与 `QuotaPlatform`。

3. 仅对真实兼容 RED 做最小修复，重跑上述命令及本地 Images/audit/body tests；无 RED 不创建生产 diff：

```powershell
Invoke-CheckedNative 'Grok gateway local contracts' { Push-Location backend; try { go test -tags unit ./internal/handler ./internal/service -run '(OpenAIImages|SecurityAudit|PromptAudit|RequestBody|BodyRetention|Grok.*Billing|Search.*Billing|OpenAIRecordUsageInputsCarryQuotaPlatform)' -count=1 } finally { Pop-Location } }
$task28Paths = @(git diff --name-only -- backend/internal/handler backend/internal/service backend/internal/pkg/xai)
if ($task28Paths.Count -gt 0) { Commit-NamedPaths -Message 'fix: preserve Grok gateway contracts after v0.1.173' -Paths $task28Paths }
```

**提交边界：**Grok media/Voice/search gateway 与直接测试；pricing schema/UI 留给 Task 31。**检查点：**最终账号/模型、审计、body 生命周期或单次计费不明确时不得通过。

### Task 29: 融合 Grok free gate、冷却与调度阈值

- [x] Task 29: 融合 Grok free gate、冷却与调度阈值

**映射 OpenSpec：**6.4

**文件：**`backend/internal/service/grok_free_quota_gate.go`、`grok_team_rate_limit.go`、`grok_stream_idle.go`、`grok_model_quota_block.go`、`openai_account_scheduler.go`、`ratelimit_service*.go`、`setting_*`、token refresh/quota files 及 tests；`frontend/src/views/admin/SettingsView.vue` 与 `GrokQuotaProbeCell.vue`。

**接口：**
- Consumes: 本地 scheduler/sticky/pool/WaitPlan、Grok quota snapshots、`AccountSchedulingThresholds`、订阅 quota windows。
- Produces: free 24h gate 只作用于明确 free 账号；team+model 冷却不牵连兄弟模型；阈值默认 100=禁用；临时下线可恢复且不改变订阅锚点。

**步骤：**

1. 运行调度和设置聚焦测试：

```powershell
Invoke-CheckedNative 'Grok scheduler RED/protection' { Push-Location backend; try { go test -tags unit ./internal/service -run '(GrokFreeQuota|GrokTeamRateLimit|GrokStreamIdle|GrokModelQuota|SchedulingThreshold|SelectAccountWithScheduler|Layered|Sticky|WaitPlan|TokenRefresh|GrokQuota)' -count=1 } finally { Pop-Location } }
```

2. 新增或保留一条跨域保护：Grok free rolling 24h 判断不得调用 subscription `automaticWindowStartAt`，订阅 exact-anchor tests 必须继续 GREEN：

```powershell
Invoke-CheckedNative 'subscription anchor isolation' { Push-Location backend; try { go test -tags unit ./internal/service -run '(TestAdminResetQuota_ResetBoth|TestAssignOrExtendSubscription_ExpiredDailyCardStartsNewOneTimeQuota|TestAutomaticWindowPreservesPersistedManualAnchor|GrokFreeQuota)' -count=1 } finally { Pop-Location } }
```

3. 审查默认值与恢复边界：free gate miss/failure fail-open 且异步刷新；明确耗尽产生可恢复 temporary reason；pool mode 保留既有不变异账号规则；7d/30d 阈值只支持 OpenAI/Anthropic/Grok，默认 100 不停调；设置更新即时刷新 cache。

4. 只修复真实 RED，重跑 backend focused/full package、Settings/GrokQuota 前端 tests；有 diff 时提交：

```powershell
Invoke-CheckedNative 'Grok scheduler frontend' { pnpm --dir frontend exec vitest run src/views/admin/__tests__/SettingsView.spec.ts src/components/account/__tests__/GrokQuotaProbeCell.spec.ts }
$task29Paths = @(git diff --name-only -- backend/internal/service frontend/src/views/admin/SettingsView.vue frontend/src/views/admin/__tests__/SettingsView.spec.ts frontend/src/components/account/GrokQuotaProbeCell.vue)
if ($task29Paths.Count -gt 0) { Commit-NamedPaths -Message 'fix: preserve Grok scheduling after v0.1.173' -Paths $task29Paths }
```

**提交边界：**Grok scheduling/quota/settings 与直接测试。**检查点：**sticky/pool/WaitPlan、可恢复状态和订阅锚点必须保持。

### Task 30: 融合 Channel Monitor V2 与隐私默认

- [x] Task 30: 融合 Channel Monitor V2 与隐私默认

**映射 OpenSpec：**6.5

**文件：**`backend/internal/service/channel_monitor_*`、`backend/internal/repository/channel_monitor_v2_*`、`backend/internal/handler/channel_monitor_*`、routes guard、setting parse/public/update、`frontend/src/features/channel-monitor-v2/`、`api/channelMonitorV2.ts`、`views/admin/ChannelMonitorView.vue` 及 tests。

**接口：**
- Consumes: `channel_monitor_mode`、V1 runner、gateway usage logs、user/admin scope。
- Produces: 缺失设置默认为 V1；V1/V2 runner 互斥；V2 rollup 幂等；admin 指标完整，普通用户 RPM/TPM 默认归零/隐藏。

**步骤：**

1. 运行 backend V1/V2、feature guard、repository 和 privacy tests：

```powershell
Invoke-CheckedNative 'Channel Monitor V2 backend RED/protection' { Push-Location backend; try { go test ./internal/service ./internal/repository ./internal/handler ./internal/server/routes -run '(ChannelMonitor|MonitorMode|RedactChannelMonitorV2|HideThroughput)' -count=1 } finally { Pop-Location } }
```

2. 审查并用测试锁定：`normalizeChannelMonitorMode("")==v1`；V1 只在 mode=v1 发主动探测，V2 aggregator 只在 mode=v2 运行；用户 scope 不泄露其它用户身份；rate 使用覆盖窗口；ignored categories、histogram merge 和 retention 幂等。

3. 运行前端 API、format、matrix、view tests 与 typecheck：

```powershell
Invoke-CheckedNative 'Channel Monitor V2 frontend' { pnpm --dir frontend exec vitest run src/api/__tests__/channelMonitorV2.spec.ts src/features/channel-monitor-v2/__tests__ src/views/admin/__tests__/ChannelMonitorView.duplicate.spec.ts src/views/admin/__tests__/ChannelMonitorView.grok.spec.ts }
Invoke-CheckedNative 'Channel Monitor V2 typecheck' { pnpm --dir frontend run typecheck }
```

4. 只对真实本地兼容回归做最小修复并提交；migration 文件留给 Task 31 统一 identity/integration：

```powershell
$task30Paths = @(git diff --name-only -- backend/internal/service backend/internal/repository backend/internal/handler backend/internal/server/routes frontend/src/api/channelMonitorV2.ts frontend/src/api/__tests__/channelMonitorV2.spec.ts frontend/src/features/channel-monitor-v2 frontend/src/views/admin/ChannelMonitorView.vue frontend/src/views/admin/__tests__)
if ($task30Paths.Count -gt 0) { Commit-NamedPaths -Message 'fix: preserve channel monitoring after v0.1.173' -Paths $task30Paths }
```

**提交边界：**Channel Monitor backend/frontend 与直接 tests。**检查点：**默认 V1、runner 互斥和用户吞吐脱敏缺一不可。

### Task 31: 闭合 Grok 定价、生成物与 173 migrations

- [x] Task 31: 闭合 Grok 定价、生成物与 173 migrations

**映射 OpenSpec：**6.6

**文件：**`backend/ent/schema/group.go`、生成 Ent、group service/handler/DTO、`video_billing*.go`、`billing_search_audio_cost_test.go`、auth cache snapshot、`backend/migrations/194_channel_monitor_v2.sql` 至 `206_*`、`217_*` 至 `220_*`、migration integration test、`frontend/src/views/admin/groupsVideoModelPricing.ts`、Groups/Settings UI/tests。

**接口：**
- Produces: model family × resolution video pricing、audio/Voice/search pricing、auth snapshot v19 fields、24 个受保护 migration blobs。
- Consumes: Task 28 的 usage dimensions、Task 30 的 Channel Monitor schema、既有完整文件名 migration runner。

**步骤：**

1. 运行定价、auth snapshot、API contract 和前端矩阵 tests：

```powershell
Invoke-CheckedNative 'Grok pricing backend RED/protection' { Push-Location backend; try { go test -tags unit ./internal/service ./internal/handler/admin ./internal/server -run '(VideoBilling|VideoModelPrice|Audio|Voice|Search.*Cost|Search.*Price|AuthCache.*Profit|APIContract)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'Grok pricing frontend' { pnpm --dir frontend exec vitest run src/views/admin/__tests__/groupsVideoModelPricing.spec.ts src/views/admin/__tests__/GroupsView.columnSettings.spec.ts src/utils/__tests__/billingMode.spec.ts }
```

2. 融合 schema/provider 源后运行 generate；不手工编辑 Ent。确认 group DTO、cache snapshot、repository insert/update 和前端 payload 同时携带 `video_model_prices`、audio/Voice、`search_price_per_1k`。

```powershell
Invoke-CheckedNative 'v0.1.173 pricing generate' { make -C backend generate }
```

3. 对 `$requiredMigrationSourcesFinal` 全部 24 个文件比较当前工作树 blob 与权威 ref；同号 194/195 必须同时存在，任何 SQL 工作树修改立即阻塞提交：

```powershell
if ($requiredMigrationSourcesFinal.Count -ne 24) { throw "expected 24 protected migrations, got $($requiredMigrationSourcesFinal.Count)" }
foreach ($entry in $requiredMigrationSourcesFinal.GetEnumerator()) {
    $name = $entry.Key
    $sourceRef = $entry.Value
    $expectedOutput = @(git rev-parse "${sourceRef}:backend/migrations/$name" 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "cannot resolve authoritative migration: $name" }
    $actualOutput = @(git hash-object -- "backend/migrations/$name" 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "cannot hash migration worktree file: $name" }
    $expected = ($expectedOutput -join '').Trim()
    $actual = ($actualOutput -join '').Trim()
    if ($expected -ne $actual) { throw "migration identity mismatch: $name" }
}
```

4. 扩展固定 migration integration：种子包含 Grok、composite、OpenAI 三类 group，并给 OpenAI 行同时写入 `audio_realtime_price_per_min`、`audio_tts_price_per_million_chars`、`audio_stt_price_per_hour`、`search_price_per_1k`。执行 220 后 Grok/composite 视频价格保持，OpenAI 视频价格清空且 `groups_video_price_backup_220` 保留原值；四个 audio/search 字段逐值保持。Docker 可用时必须看到目标 top-level test 真实 PASS；不可用时把 schema、backup、清理边界、其它定价不变、幂等/checksum 记为 `unverified`。

5. 两轮 generate 零 diff、focused tests 和 frontend lint/typecheck 通过后，只提交真实兼容 diff；tag migration 文件不得修改：

```powershell
$task31Paths = @(git diff --name-only -- backend/ent backend/internal/handler/admin backend/internal/handler/dto backend/internal/repository backend/internal/service frontend/src/views/admin frontend/src/utils frontend/src/types)
if ($task31Paths.Count -gt 0) { Commit-NamedPaths -Message 'fix: preserve pricing and migrations after v0.1.173' -Paths $task31Paths }
```

**提交边界：**pricing schema/source/generated/frontend 与 migration integration test；17 个 173 SQL blob 只能来自 merge。**检查点：**24 个 identity、生成归属和 220 backup/平台边界必须闭合。

### Task 32: 运行 v0.1.173 全量门禁并关闭矩阵

- [x] Task 32: 运行 v0.1.173 全量门禁并关闭矩阵

**映射 OpenSpec：**6.7

**文件：**默认只更新 build ledger；门禁发现回归时按 Tasks 27-31 所属能力簇派发 remediation，Task 32 保持未勾选。

**步骤：**

1. 重跑第四阶段五个 canonical focused bundles：

```powershell
Invoke-CheckedNative 'gate Grok auth/mapping' { Push-Location backend; try { go test ./internal/pkg/xai ./internal/service ./internal/handler/admin -run '(GrokCrossClient|DefaultModelMapping|GrokOAuth|PasswordCapability|MessagesDispatchModel)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'gate Grok gateway' { Push-Location backend; try { go test -tags unit ./internal/handler ./internal/service ./internal/pkg/xai -run '(GrokMedia|GrokAudio|Voice|Video|WebSearch|SearchCount|RequestBody|Sticky|Failover|RecordUsage)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'gate Grok scheduler' { Push-Location backend; try { go test -tags unit ./internal/service -run '(GrokFreeQuota|GrokTeamRateLimit|GrokStreamIdle|SchedulingThreshold|Layered|Sticky|WaitPlan|Subscription.*Anchor)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'gate Channel Monitor V2' { Push-Location backend; try { go test ./internal/service ./internal/repository ./internal/handler ./internal/server/routes -run '(ChannelMonitor|HideThroughput)' -count=1 } finally { Pop-Location } }
Invoke-CheckedNative 'gate 173 frontend' { pnpm --dir frontend exec vitest run src/views/admin/__tests__/SettingsView.spec.ts src/components/account/__tests__/CreateAccountModal.grok.spec.ts src/components/admin/account/__tests__/ReAuthAccountModal.grok.spec.ts src/api/__tests__/channelMonitorV2.spec.ts src/features/channel-monitor-v2/__tests__ src/views/admin/__tests__/ChannelMonitorView.grok.spec.ts src/views/admin/__tests__/groupsVideoModelPricing.spec.ts }
```

2. 在中间 VERSION 上运行完整门禁和两轮生成：

```powershell
Invoke-CheckedNative 'v0.1.173 make test' { make test }
Invoke-CheckedNative 'v0.1.173 intermediate build' { make 'VERSION=0.1.171.1' 'SHELL=D:/scoop/shims/bash.exe' build }
Invoke-CheckedNative 'v0.1.173 backend lint' { Push-Location backend; try { golangci-lint run ./... } finally { Pop-Location } }
Invoke-CheckedNative 'v0.1.173 frontend lint' { pnpm --dir frontend run lint:check }
Invoke-CheckedNative 'v0.1.173 frontend typecheck' { pnpm --dir frontend run typecheck }
Invoke-CheckedNative 'v0.1.173 first generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'first v0.1.173 generate changed output' }
Invoke-CheckedNative 'v0.1.173 second generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'second v0.1.173 generate changed output' }
Assert-NoConflictArtifacts
```

3. 执行 Docker 条件 integration；不可用只记 `unverified`，目标真实失败阻塞。逐行关闭第四阶段能力矩阵，`gap=0` 后提交 ledger：

```powershell
Commit-NamedPaths -Message 'docs: record v0.1.173 gates' -Paths @($buildLedger)
```

**提交边界：**报告提交与 remediation 分离。**检查点：**全部非 Docker gate PASS、generate stable、gap=0 后才能 bump VERSION。

### Task 33: 将最终版本更新为 0.1.173.1

- [ ] Task 33: 将最终版本更新为 0.1.173.1

**映射 OpenSpec：**6.8

**文件：**仅 `backend/cmd/server/VERSION`。

**步骤：**

1. 确认当前值为中间版本并写入唯一新值：

```powershell
if ((Get-Content -Raw backend/cmd/server/VERSION).Trim() -ne '0.1.171.1') { throw 'unexpected pre-173 VERSION' }
```

```diff
-0.1.171.1
+0.1.173.1
```

2. 提交并复核：

```powershell
if ((Get-Content -Raw backend/cmd/server/VERSION).Trim() -ne '0.1.173.1') { throw 'VERSION write failed' }
Commit-NamedPaths -Message 'chore: bump version to 0.1.173.1' -Paths @('backend/cmd/server/VERSION')
```

**提交边界：**版本提交只有一个文件。**检查点：**不创建 `0.1.172.1` 或其他过程版本。

### Task 34: 在最终 173 HEAD 重跑门禁并验证四段拓扑

- [ ] Task 34: 在最终 173 HEAD 重跑门禁并验证四段拓扑

**映射 OpenSpec：**6.8

**文件：**无产品修改；结果由 Task 35 记录。

**步骤：**

1. 在 VERSION commit 后重新运行完整命令，不继承 Task 32 结论：

```powershell
Invoke-CheckedNative 'final 173 make test' { make test }
Invoke-CheckedNative 'final 173 build' { make 'VERSION=0.1.173.1' 'SHELL=D:/scoop/shims/bash.exe' build }
Invoke-CheckedNative 'final 173 backend lint' { Push-Location backend; try { golangci-lint run ./... } finally { Pop-Location } }
Invoke-CheckedNative 'final 173 frontend lint' { pnpm --dir frontend run lint:check }
Invoke-CheckedNative 'final 173 frontend typecheck' { pnpm --dir frontend run typecheck }
Invoke-CheckedNative 'final 173 first generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'final first generate changed output' }
Invoke-CheckedNative 'final 173 second generate' { make -C backend generate }
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
if ($LASTEXITCODE -ne 0) { throw 'final second generate changed output' }
Assert-NoConflictArtifacts
```

2. 验证四个 tag 和唯一 first-parent merge 第二父：

```powershell
$targetTags = @($tag170,$tag171,$tag172,$tag173)
foreach ($tag in $targetTags) {
    git merge-base --is-ancestor $tag HEAD
    if ($LASTEXITCODE -ne 0) { throw "target tag is not ancestor: $tag" }
}
$firstParentMerges = @(git rev-list --first-parent --merges "$sourceBase..HEAD")
foreach ($tag in $targetTags) {
    $nodes = @($firstParentMerges | Where-Object { (git rev-parse "$_^2").Trim() -eq $tag })
    if ($nodes.Count -ne 1) { throw "expected one merge node for $tag, got $($nodes.Count)" }
}
if ((Get-Content -Raw backend/cmd/server/VERSION).Trim() -ne '0.1.173.1') { throw 'final VERSION mismatch' }
```

3. 重跑 `$requiredMigrationSourcesFinal` 24 个 committed blob identity 和 Docker 条件 integration；任何 filename/blob mismatch 或真实 integration failure 阻塞：

```powershell
if ($requiredMigrationSourcesFinal.Count -ne 24) { throw 'final protected migration count mismatch' }
foreach ($entry in $requiredMigrationSourcesFinal.GetEnumerator()) {
    $name = $entry.Key
    $expectedOutput = @(git rev-parse "$($entry.Value):backend/migrations/$name" 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "cannot resolve final authoritative migration: $name" }
    $actualOutput = @(git rev-parse "HEAD:backend/migrations/$name" 2>&1)
    if ($LASTEXITCODE -ne 0) { throw "cannot resolve final committed migration: $name" }
    if (($expectedOutput -join '').Trim() -ne ($actualOutput -join '').Trim()) { throw "final migration identity mismatch: $name" }
}
$finalIntegration = Invoke-MigrationUpgradeIntegration -Stage 'v0.1.173-final'
$finalIntegration
```

**提交边界：**无提交。**检查点：**最终 gates、四 merge、VERSION、24 migration identities 全部闭合。

### Task 35: 完成 v0.1.173 thorough review 和最终 Verify 报告

- [ ] Task 35: 完成 v0.1.173 thorough review 和最终 Verify 报告

**映射 OpenSpec：**6.8

**文件：**修改 build ledger 和现有 verify report；旧 171 Verify 章节保留为历史证据。

**步骤：**

1. 逐行复核四阶段能力矩阵，特别记录 172 OAuth pending/实际时刻 anchor/response-model audit，以及 173 Grok mapping/password、media/Voice/search、free gate/threshold、Channel Monitor V2/privacy、pricing 和 24 migrations；`gap=0`。

2. 运行 strict validation：

```powershell
Invoke-CheckedNative 'v0.1.173 OpenSpec strict validation' { comet classic openspec -- validate staged-merge-upstream-v0-1-171 --strict }
```

3. 派发 fresh thorough reviewer，范围覆盖 Task 19 execution base 至 final HEAD；CRITICAL/IMPORTANT 回到所属能力簇 remediation。Docker/cgo 仅作为环境 residual，不伪装 PASS。

4. verify report 记录四个 tag object/SHA、172/173 文件面、四个 merge parents、VERSION、focused/full gates、生成稳定性、24 migration blobs/integration、review verdict和无远程操作，提交两份报告：

```powershell
Commit-NamedPaths -Message 'docs: record v0.1.173 verification' -Paths @($buildLedger,$verifyReport)
Assert-CleanGate
```

**提交边界：**只含两份报告。**完成检查点：**reviewer APPROVED、所有非 Docker gate 新鲜 PASS、四段拓扑精确、`gap=0`，然后进入 Comet Verify；仍不 push/tag/release/deploy。

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
| 5.1 | Task 19 |
| 5.2 | Task 20 |
| 5.3 | Task 21 |
| 5.4 | Task 22 |
| 5.5 | Task 23 |
| 5.6 | Task 24 |
| 5.7 | Task 25 |
| 5.8 | Task 25 |
| 6.1 | Task 26 |
| 6.2 | Task 27 |
| 6.3 | Task 28 |
| 6.4 | Task 29 |
| 6.5 | Task 30 |
| 6.6 | Task 31 |
| 6.7 | Task 32 |
| 6.8 | Tasks 33-35 |
