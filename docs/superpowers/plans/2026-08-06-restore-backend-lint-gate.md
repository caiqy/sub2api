---
change: restore-backend-lint-gate
design-doc: docs/superpowers/specs/2026-08-06-restore-backend-lint-gate-design.md
base-ref: 9dafc41f7ca0d7ea334698bf554cf7e0facb6038
---

# Restore Backend Lint Gate 实施计划

**目标：** 在不改变请求体 ownership、spool、retry、failover、审计、计费或上游 wire body 语义的前提下，关闭 baseline 的 144 个 lint issues，使 uncapped lint 与根级 `make test` 均为 exit 0。

**实施方式：** 三个 implementation commits 依次处理 handler/routes/QF1003、Gateway/Anthropic/Bedrock/Antigravity、OpenAI/Gemini/Grok/unused。每批以 uncapped golangci-lint JSON 作为 RED/GREEN 断言，关闭死局部而不删除任何可观察 owner、struct field、handle 或真实 cleanup；第四任务只提交 docs 证据与进度。

## 规范与范围

- 语义、失败处理、三提交边界与 upstream 后续边界以 [Design Doc](../specs/2026-08-06-restore-backend-lint-gate-design.md) 第 1-10 节为准；问题位置、计数和 protected blob 以 [baseline manifest](../reports/2026-08-06-restore-backend-lint-gate-baseline.md) 第 11-122 行为准。
- `9dafc41f7ca0d7ea334698bf554cf7e0facb6038` 是 plan base，不要求等于实施时 HEAD；它必须是 checkpoint HEAD 的祖先。`b576f73a22c4bf23d61727fc93950766a7e33929` 只用于 manifest identity 与 protected-input 复核。
- 不修改 `backend/.golangci.yml`、`backend/go.mod`、`backend/go.sum`、`.github/workflows/backend-ci.yml`、`Makefile`、`backend/Makefile`、Go 版本、Make target、CI 或 issue cap；禁止 suppression、假读取、反射、清零 helper、Docker、upstream merge、版本、部署、push、tag 和 release。
- 默认 backend allowlist 是 `$manifestPaths` 的 39 个文件。只有真实行为回退必须新增测试时，才可停止当前任务，先同步更新 Design Doc、baseline manifest、OpenSpec `tasks.md` 和本计划的 allowlist，并完成范围审查。
- 纯死局部只删除赋值；混合赋值只删 linter 指向的局部，保留 owner/field/handle cleanup。测试 GC 清零先删除；仅在 retained-heap 测试回退时，对失败 branch 的物化路径以内层函数缩窄大 slice 作用域。
- 本计划只有以下 5 个顶层任务标记。Comet coordinator 在验收后用 `apply_patch` 改写对应 marker 与映射 OpenSpec marker，实施者不得直接编辑这些进度 artifacts。

## 可重放常量与验证助手

每个恢复 session 在执行任何 Gate/Task 命令前，必须先完整重载本代码块；不得依赖上一 session 的变量或函数。

```powershell
$changeName = 'restore-backend-lint-gate'
$changeDir = 'docs/openspec/changes/restore-backend-lint-gate'
$planFile = 'docs/superpowers/plans/2026-08-06-restore-backend-lint-gate.md'
$tasksFile = 'docs/openspec/changes/restore-backend-lint-gate/tasks.md'
$reportFile = 'docs/superpowers/reports/2026-08-06-restore-backend-lint-gate-verification.md'
$progressFile = 'docs/openspec/changes/restore-backend-lint-gate/.comet/subagent-progress.md'
$planBase = '9dafc41f7ca0d7ea334698bf554cf7e0facb6038'
$manifestBase = 'b576f73a22c4bf23d61727fc93950766a7e33929'
$handlerQFPaths = @(
  'internal/handler/gateway_handler.go',
  'internal/handler/gateway_handler_chat_completions.go',
  'internal/handler/gateway_handler_responses.go',
  'internal/handler/gemini_v1beta_handler.go',
  'internal/handler/grok_media.go',
  'internal/handler/image_task_handler.go',
  'internal/handler/openai_alpha_search.go',
  'internal/handler/openai_chat_completions.go',
  'internal/handler/openai_gateway_count_tokens.go',
  'internal/handler/openai_gateway_handler.go',
  'internal/handler/openai_live.go',
  'internal/handler/request_body_memory_retention_test.go',
  'internal/server/routes/gateway.go'
)
$qfOnlyPaths = @(
  'internal/handler/gateway_handler_responses.go',
  'internal/handler/request_body_memory_retention_test.go'
)
$gatewayPaths = @(
  'internal/service/antigravity_gateway_claude.go',
  'internal/service/antigravity_gateway_compat.go',
  'internal/service/antigravity_gateway_gemini.go',
  'internal/service/antigravity_gateway_service_test.go',
  'internal/service/antigravity_gateway_upstream.go',
  'internal/service/gateway_anthropic_apikey_passthrough_test.go',
  'internal/service/gateway_anthropic_passthrough.go',
  'internal/service/gateway_bedrock.go',
  'internal/service/gateway_count_tokens.go',
  'internal/service/gateway_forward.go',
  'internal/service/gateway_forward_as_chat_completions.go',
  'internal/service/gateway_upstream_request.go',
  'internal/service/gemini_chat_completions_compat_service.go'
)
$openAIPaths = @(
  'internal/service/openai_alpha_search.go',
  'internal/service/openai_gateway_cc_pipeline.go',
  'internal/service/openai_gateway_chat_completions.go',
  'internal/service/openai_gateway_chat_completions_raw.go',
  'internal/service/openai_gateway_count_tokens.go',
  'internal/service/openai_gateway_forward.go',
  'internal/service/openai_gateway_grok.go',
  'internal/service/openai_gateway_grok_chat_bridge.go',
  'internal/service/openai_gateway_messages.go',
  'internal/service/openai_gateway_messages_chat_fallback.go',
  'internal/service/openai_gateway_passthrough.go',
  'internal/service/openai_gateway_responses_chat_fallback.go',
  'internal/service/openai_live.go'
)
$manifestPaths = @($handlerQFPaths + $gatewayPaths + $openAIPaths | Sort-Object -Unique)
if ($manifestPaths.Count -ne 39) { throw "manifest path count is $($manifestPaths.Count), expected 39" }
$protectedPaths = @(
  'backend/.golangci.yml',
  'backend/go.mod',
  'backend/go.sum',
  '.github/workflows/backend-ci.yml',
  'Makefile',
  'backend/Makefile'
)
$expectedBlobs = @{
  'backend/.golangci.yml' = '92ba3916948b4b859737c3c4831c7416dcd7f01e'
  'backend/go.mod' = '7d5150f4a969df8a578e5bce8e6f5a01ec856823'
  'backend/go.sum' = '72146c2305a91a48f92ac8fe2f9d888a2a1a2886'
  '.github/workflows/backend-ci.yml' = 'ee84c994ca2f1e27ae32eb02f25c3d094581b1ff'
  'Makefile' = 'da7c0c59fe67dfc8219ecfb2fbab1238fd0bbb55'
  'backend/Makefile' = '0327160ff0959575ed6a8f950d7d257a96ae3ab0'
}
$protectedDiffArgs = @('diff', '--exit-code', $manifestBase, '--') + $protectedPaths
$checkpointStateArtifacts = @(
  "$changeDir/.comet.yaml",
  "$changeDir/.comet/run-state.json",
  "$changeDir/.comet/state-events.jsonl",
  "$changeDir/.comet/trajectory.jsonl"
)
$retainedHeapPattern = '^(TestAsyncImageRequestBodyMemoryRetentionWhileWorkersBlocked|TestRequestBodyMemoryRetentionWhileUpstreamBlocked)$'
$handlerReplayPattern = '^(TestGatewayHandler_MessagesAndResponsesReplayLargeBodiesAcrossFailover|TestOpenAIGatewayHandler_ChatAndEmbeddingsReplayMappedSpoolAcrossFailover|TestOpenAIGatewayHandler_ChatReplayRawSpoolAcrossFailoverWhenResponsesUnsupported|TestAsyncImageHandlerSpoolsReplaysAndCleansOwnedBody|TestAsyncImageHandlerSpoolsReplaysAndCleansOwnedMultipartBody|TestGrokMedia_GenerateEditVideoRejectUpstreamFailoverPreserveRequestSemantics|TestGrokMedia_MultipartSpoolPreservesFilesAndOmitsSnapshots)$'
$serviceReplayPattern = '^(TestAntigravityGatewayService_ClaudeForwardHandleSignatureRetryReparsesFileBackedCanonical|TestAntigravityRetryLoopReopensGeminiPayloadHandleForRetry|TestGatewayService_AnthropicPassthroughRetryRereadsHandleAfterForwardFirstAttemptBytes|TestForwardAsResponsesHandle_SpoolTransportErrorPreservesSentinel|TestGeminiMessagesCompatSignatureRetryBuildsHandleFromCanonical|TestOpenAIGatewayService_RejectedFieldRetryReturnsSpoolError|TestOpenAIForwardReusesBoundRequestBodyHandle|TestOpenAIForwardPreservesBoundRequestBodyHandleWhenHTTPDoErrors|TestOpenAILiveCreateUpstreamHandleTransportSpoolErrorClosesBodies)$'

function Assert-ExactStringSet {
  param([string]$Label, [string[]]$Actual, [string[]]$Expected)
  $actualSet = @($Actual | Sort-Object -Unique)
  $expectedSet = @($Expected | Sort-Object -Unique)
  $missing = @($expectedSet | Where-Object { $actualSet -notcontains $_ })
  $extra = @($actualSet | Where-Object { $expectedSet -notcontains $_ })
  if ($missing.Count -ne 0 -or $extra.Count -ne 0) {
    throw "$Label mismatch; missing=[$($missing -join ', ')]; extra=[$($extra -join ', ')]"
  }
}

function Assert-StagedPathSet {
  param([string]$Label, [string[]]$ExpectedPaths)
  $actualPaths = @(& git diff --cached --name-only)
  Assert-ExactStringSet -Label "$Label staged path set" -Actual $actualPaths -Expected $ExpectedPaths
}

function Commit-ExactStaged {
  param([string]$Label, [string[]]$ExpectedPaths, [string]$Message)
  Assert-StagedPathSet -Label $Label -ExpectedPaths $ExpectedPaths
  & git commit -m $Message
  if ($LASTEXITCODE -ne 0) { throw "$Label commit failed" }
  $remainingStaged = @(& git diff --cached --name-only)
  if ($remainingStaged.Count -ne 0) { throw "$Label left staged paths: $($remainingStaged -join ', ')" }
}

function ConvertTo-NormalizedLintPath {
  param([string]$Path)
  $normalized = $Path -replace '\\', '/'
  $normalized = $normalized -replace '^\./', ''
  if ($normalized.StartsWith('backend/')) { $normalized = $normalized.Substring('backend/'.Length) }
  if ([string]::IsNullOrWhiteSpace($normalized)) { throw 'lint issue has no Pos.Filename' }
  return $normalized
}

function Invoke-UncappedLintJson {
  Push-Location backend
  try {
    $jsonText = (& golangci-lint run ./... --max-issues-per-linter 0 --max-same-issues 0 --show-stats=false --output.text.path stderr --output.json.path stdout 2>$null) -join "`n"
    $exitCode = $LASTEXITCODE
  } finally {
    Pop-Location
  }
  if ([string]::IsNullOrWhiteSpace($jsonText)) { throw 'golangci-lint produced no JSON output' }
  try {
    $document = $jsonText | ConvertFrom-Json
  } catch {
    throw "golangci-lint JSON parse failed: $($_.Exception.Message)"
  }
  $issues = @($document.Issues | Where-Object { $null -ne $_ })
  $linterCounts = @{ ineffassign = 0; staticcheck = 0; unused = 0 }
  $paths = @()
  $fingerprints = @()
  $stableFingerprints = @()
  foreach ($issue in $issues) {
    $linter = [string]$issue.FromLinter
    if ($linterCounts.ContainsKey($linter)) { $linterCounts[$linter]++ }
    $path = ConvertTo-NormalizedLintPath -Path ([string]$issue.Pos.Filename)
    $line = [int]$issue.Pos.Line
    $text = (([string]$issue.Text).Trim() -replace '\s+', ' ')
    $paths += $path
    $fingerprints += "$path|$line|$linter|$text"
    $stableFingerprints += "$path|$linter|$text"
  }
  [pscustomobject]@{
    ExitCode = $exitCode
    JsonText = $jsonText
    IssueCount = $issues.Count
    LinterCounts = $linterCounts
    Paths = @($paths | Sort-Object -Unique)
    Fingerprints = @($fingerprints | Sort-Object -Unique)
    StableFingerprints = @($stableFingerprints | Sort-Object)
  }
}

function Assert-LintState {
  param(
    [string]$Label,
    [psobject]$Summary,
    [int]$ExpectedExitCode,
    [int]$ExpectedIssueCount,
    [int]$ExpectedIneffassign,
    [int]$ExpectedStaticcheck,
    [int]$ExpectedUnused,
    [int]$ExpectedPathCount,
    [string[]]$ExpectedPaths
  )
  if ($Summary.ExitCode -ne $ExpectedExitCode) { throw "$Label lint exit=$($Summary.ExitCode), expected $ExpectedExitCode" }
  if ($Summary.IssueCount -ne $ExpectedIssueCount) { throw "$Label issues=$($Summary.IssueCount), expected $ExpectedIssueCount" }
  if ($Summary.LinterCounts['ineffassign'] -ne $ExpectedIneffassign) { throw "$Label ineffassign=$($Summary.LinterCounts['ineffassign']), expected $ExpectedIneffassign" }
  if ($Summary.LinterCounts['staticcheck'] -ne $ExpectedStaticcheck) { throw "$Label staticcheck=$($Summary.LinterCounts['staticcheck']), expected $ExpectedStaticcheck" }
  if ($Summary.LinterCounts['unused'] -ne $ExpectedUnused) { throw "$Label unused=$($Summary.LinterCounts['unused']), expected $ExpectedUnused" }
  if ($Summary.Paths.Count -ne $ExpectedPathCount) { throw "$Label distinct paths=$($Summary.Paths.Count), expected $ExpectedPathCount" }
  Assert-ExactStringSet -Label "$Label lint path set" -Actual $Summary.Paths -Expected $ExpectedPaths
}

function Get-MultisetDifference {
  param([string[]]$Left, [string[]]$Right)
  $rightCounts = @{}
  foreach ($value in $Right) {
    if (-not $rightCounts.ContainsKey($value)) { $rightCounts[$value] = 0 }
    $rightCounts[$value]++
  }
  $difference = @()
  foreach ($value in $Left) {
    if ($rightCounts.ContainsKey($value) -and $rightCounts[$value] -gt 0) {
      $rightCounts[$value]--
    } else {
      $difference += $value
    }
  }
  return @($difference)
}

function Compare-LintIdentity {
  param([psobject]$Previous, [psobject]$Current)
  [pscustomobject]@{
    Added = @(Get-MultisetDifference -Left $Current.StableFingerprints -Right $Previous.StableFingerprints)
    Removed = @(Get-MultisetDifference -Left $Previous.StableFingerprints -Right $Current.StableFingerprints)
    PreviousDetailed = $Previous.Fingerprints
    CurrentDetailed = $Current.Fingerprints
  }
}

function Assert-NoAddedLintIdentity {
  param([string]$Label, [psobject]$Diff)
  if ($Diff.Added.Count -ne 0) { throw "$Label introduced lint identities: $($Diff.Added -join '; ')" }
}

function Invoke-GoPackageTests {
  param([string]$Label, [string[]]$Packages)
  Push-Location backend
  try {
    $output = @(& go test -count=1 @Packages)
    $exitCode = $LASTEXITCODE
  } finally {
    Pop-Location
  }
  if ($exitCode -ne 0) { throw "$Label package tests failed with exit $exitCode`n$($output -join "`n")" }
}

function Invoke-VerifiedGoTests {
  param([string]$Label, [string]$Package, [string]$Pattern, [int]$ExpectedCount)
  Push-Location backend
  try {
    $listOutput = @(& go test $Package -list $Pattern)
    $listExitCode = $LASTEXITCODE
    $names = @($listOutput | ForEach-Object { ([string]$_).Trim() } | Where-Object { $_ -match '^Test' })
    if ($listExitCode -ne 0) { throw "$Label -list failed with exit $listExitCode`n$($listOutput -join "`n")" }
    if ($names.Count -ne $ExpectedCount) { throw "$Label -list matched $($names.Count), expected $ExpectedCount: $($names -join ', ')" }
    $runOutput = @(& go test -count=1 $Package -run $Pattern)
    $runExitCode = $LASTEXITCODE
  } finally {
    Pop-Location
  }
  if ($runExitCode -ne 0) { throw "$Label -run failed with exit $runExitCode`n$($runOutput -join "`n")" }
}

function Assert-NoBackendImplementationChanges {
  param([string]$Label, [string[]]$Paths)
  $backendPaths = @($Paths | Where-Object { $_ -like 'backend/*' })
  if ($backendPaths.Count -ne 0) { throw "$Label contains backend paths: $($backendPaths -join ', ')" }
}

function Get-ChangedCandidatePaths {
  param([string[]]$CandidatePaths)
  $changed = @()
  foreach ($candidate in $CandidatePaths) {
    $statusLines = @(& git status --porcelain=v1 --untracked-files=all --ignored -- $candidate)
    if ($statusLines.Count -ne 0) { $changed += $candidate }
  }
  return @($changed | Sort-Object -Unique)
}

function Get-CheckpointCommitExpectedPaths {
  return Get-ChangedCandidatePaths -CandidatePaths @($planFile + $checkpointStateArtifacts)
}

function Get-ProgressCommitExpectedPaths {
  $expected = @($planFile, $tasksFile)
  if (Test-Path -LiteralPath $progressFile) {
    $progressStatus = @(& git status --porcelain=v1 --untracked-files=all --ignored -- $progressFile)
    if ($progressStatus.Count -ne 0) { $expected += $progressFile }
  }
  return @($expected | Sort-Object -Unique)
}

function Assert-FinalRepositoryScope {
  $committedPaths = @(& git diff --name-only "$planBase..HEAD")
  $committedBackendPaths = @($committedPaths | Where-Object { $_ -like 'backend/*' } | ForEach-Object { $_.Substring('backend/'.Length) })
  Assert-ExactStringSet -Label 'committed backend allowlist' -Actual $committedBackendPaths -Expected $manifestPaths
  $allowedCommittedFiles = @(
    'docs/superpowers/specs/2026-08-06-restore-backend-lint-gate-design.md',
    $planFile,
    'docs/superpowers/reports/2026-08-06-restore-backend-lint-gate-baseline.md',
    $reportFile,
    $tasksFile,
    "$changeDir/.comet.yaml"
  )
  $unexpectedCommitted = @($committedPaths | Where-Object {
    $_ -notlike 'backend/*' -and
    $_ -notlike 'docs/openspec/changes/restore-backend-lint-gate/.comet/*' -and
    $allowedCommittedFiles -notcontains $_
  })
  if ($unexpectedCommitted.Count -ne 0) { throw "unexpected committed paths: $($unexpectedCommitted -join ', ')" }
  $stagedPaths = @(& git diff --cached --name-only)
  $unstagedPaths = @(& git diff --name-only)
  if ($stagedPaths.Count -ne 0) { throw "staged paths remain: $($stagedPaths -join ', ')" }
  if ($unstagedPaths.Count -ne 0) { throw "unstaged paths remain: $($unstagedPaths -join ', ')" }
  $untrackedPaths = @(& git ls-files --others --exclude-standard)
  $unexpectedUntracked = @($untrackedPaths | Where-Object { $_ -ne '.comet/current-change.json' })
  if ($unexpectedUntracked.Count -ne 0) { throw "unexpected untracked paths: $($unexpectedUntracked -join ', ')" }
}
```

## Coordinator durable planning checkpoint

在任何顶层任务开始前，coordinator 是唯一可以执行下列 state 与进度操作的角色。它先运行：

```powershell
comet state set $changeName plan $planFile
```

若 coordinator 选择 `plan-ready` 暂停，再运行：

```powershell
comet state set $changeName build_pause plan-ready
```

coordinator 不在本计划中选择 isolation、build、TDD 或 review 模式；恢复实施时由 coordinator 与实施者联合决定。建立 durable checkpoint 时按以下普通步骤执行：

1. 读取 `comet state get $changeName plan`，其输出必须精确等于 `$planFile`。若选择暂停，读取 `comet state get $changeName build_pause`，其输出必须精确等于 `plan-ready`。
2. 运行下列 state 验证；第二段只在 coordinator 已选择 `plan-ready` 暂停时执行：

   ```powershell
   $recordedPlan = ((& comet state get $changeName plan) -join "`n").Trim()
   if ($recordedPlan -ne $planFile) { throw "state plan is $recordedPlan, expected $planFile" }
   ```

   ```powershell
   $recordedBuildPause = ((& comet state get $changeName build_pause) -join "`n").Trim()
   if ($recordedBuildPause -ne 'plan-ready') { throw "state build_pause is $recordedBuildPause, expected plan-ready" }
   ```

3. 使用 `$checkpointExpected = Get-CheckpointCommitExpectedPaths` 从 `$planFile` 与 `$checkpointStateArtifacts` 构造实际 diff set；若集合为空则 throw。不得暂存整个 change 目录。运行 `& git add -f -- @checkpointExpected`，再以 `Commit-ExactStaged -Label 'planning checkpoint' -ExpectedPaths $checkpointExpected -Message 'docs: checkpoint backend lint gate plan'` 完成精确 staged 断言、提交退出码检查与空 index 检查。
4. 运行下列命令确认 plan、当前 phase/build state artifacts 已在 HEAD，index 为空且 checkpoint 路径无未提交状态：

   ```powershell
   $checkpointCheckPaths = @($planFile + $checkpointStateArtifacts)
   foreach ($path in $checkpointCheckPaths) {
     & git cat-file -e ("HEAD:" + $path)
     if ($LASTEXITCODE -ne 0) { throw "checkpoint artifact is not committed: $path" }
   }
   & git diff --cached --quiet
   if ($LASTEXITCODE -ne 0) { throw 'checkpoint index is not empty' }
    $checkpointHeadDiffArgs = @('diff', '--quiet', 'HEAD', '--') + $checkpointCheckPaths
    & git @checkpointHeadDiffArgs
    if ($LASTEXITCODE -ne 0) { throw 'checkpoint artifacts have uncommitted changes' }
    $remainingChangeState = @(& git status --porcelain=v1 --untracked-files=all -- $changeDir)
    if ($remainingChangeState.Count -ne 0) { throw "checkpoint left change artifacts dirty: $($remainingChangeState -join '; ')" }
   ```
5. 运行 `git merge-base --is-ancestor $planBase HEAD`，并以 `Assert-NoBackendImplementationChanges -Label 'checkpoint range' -Paths @(& git diff --name-only "$planBase..HEAD")` 断言 checkpoint 未携带 backend diff。

任何 checkpoint 验证失败都阻止 Gate 0。恢复 session 先重载整个 helper block，再复核该 checkpoint，不假设旧变量或旧进度仍有效。

## Coordinator 进度协议

每个顶层任务完成 implementation、验证和 review 后，coordinator 严格按此顺序执行：

1. 使用 `apply_patch` 同时把该顶层任务 marker 与本任务映射的 OpenSpec `tasks.md` marker 改为已勾选态；task-checkoff 命令不负责勾选。
2. 调用 `Get-ProgressCommitExpectedPaths`。它只允许 `$planFile`、`$tasksFile`，以及确实存在且实际有 diff 的 `$progressFile`；任何其他路径都不得暂存。
3. 精确暂存该集合，运行 `Commit-ExactStaged -Label 'task progress' -ExpectedPaths $progressExpected -Message 'docs: record backend lint gate task progress'`；该 helper 对 commit 非零和残留 index 都会 throw。
4. 仅在提交之后，运行该顶层任务指定的 plan 与 OpenSpec `comet state task-checkoff` 命令；每条命令后检查 `$LASTEXITCODE`，失败时停止且不得进入下一个任务。

## 执行任务

- [x] Gate 0：验证身份、保护输入、lint RED 与行为基线

  **OpenSpec 映射：**
  `1.1 在实施分支上复核已固定的 144 项 baseline manifest、39 个文件、分类计数和受保护文件 blob，确认相对 `b576f73a` 未漂移`
  `1.2 运行请求体内存保留、spool、retry/failover 聚焦测试，建立修复前的绿色行为基线与 lint RED`

  1. 确认 `git cat-file -e ($planBase + '^{commit}')` 与 `git cat-file -e ($manifestBase + '^{commit}')` 成功；运行 `git merge-base --is-ancestor $planBase HEAD` 并在非零时 throw。以 `Assert-NoBackendImplementationChanges` 分别断言 `git diff --name-only "$planBase..HEAD"`、`git diff --cached --name-only`、`git diff --name-only` 与 `git ls-files --others --exclude-standard` 均无 `backend/` 路径。此处不检查 ignored artifacts。
  2. 对 `$protectedPaths` 中每一路径，以 `git rev-parse ($manifestBase + ':' + $path)` 比对 `$expectedBlobs[$path]`。使用 `$protectedDiffArgs = @('diff', '--exit-code', $manifestBase, '--') + $protectedPaths; & git @protectedDiffArgs` 断言受保护 6 路径相对 manifest source base 无 diff。
  3. 从 `backend` 目录捕获工具版本与 JSON lint RED：

  ```powershell
  Push-Location backend
  try {
    $goVersion = ((& go version) -join "`n").Trim()
    $goVersionExit = $LASTEXITCODE
    $golangciLintVersion = ((& golangci-lint version) -join "`n").Trim()
    $golangciLintVersionExit = $LASTEXITCODE
  } finally {
    Pop-Location
  }
  if ($goVersionExit -ne 0) { throw 'go version failed' }
  if ($golangciLintVersionExit -ne 0) { throw 'golangci-lint version failed' }
  $gate0Lint = Invoke-UncappedLintJson
  Assert-LintState -Label 'Gate 0 RED' -Summary $gate0Lint -ExpectedExitCode 1 -ExpectedIssueCount 144 -ExpectedIneffassign 140 -ExpectedStaticcheck 3 -ExpectedUnused 1 -ExpectedPathCount 39 -ExpectedPaths $manifestPaths
  ```

  reviewer 对 `$gate0Lint.Fingerprints` 的 normalized `(path,line,linter,text)` 与 baseline manifest 第 24-82 行逐项比对；出现同计数/同路径但不同 identity 时阻塞。
  4. 先运行 `Invoke-GoPackageTests -Label 'Gate 0 handler/routes' -Packages @('./internal/handler', './internal/server/routes')` 与 `Invoke-GoPackageTests -Label 'Gate 0 service' -Packages @('./internal/service')`。然后运行 `Invoke-VerifiedGoTests`，分别以 `$retainedHeapPattern`、`$handlerReplayPattern`、`$serviceReplayPattern` 精确断言 2、7、9 个 `^Test` 匹配和对应 `-run` exit 0。任一失败为 baseline failure，停止而不开始修复。
  5. Gate 0 review 后，coordinator 按进度协议提交 plan/tasks 进度，再执行以下提交后定向验证：

  ```powershell
  $progressExpected = Get-ProgressCommitExpectedPaths
  & git add -f -- @progressExpected
  Commit-ExactStaged -Label 'Gate 0 progress' -ExpectedPaths $progressExpected -Message 'docs: record backend lint gate task progress'
  & comet state task-checkoff $planFile 'Gate 0：验证身份、保护输入、lint RED 与行为基线'
  if ($LASTEXITCODE -ne 0) { throw 'Gate 0 plan task-checkoff failed' }
  & comet state task-checkoff $tasksFile '1.1 在实施分支上复核已固定的 144 项 baseline manifest、39 个文件、分类计数和受保护文件 blob，确认相对 `b576f73a` 未漂移'
  if ($LASTEXITCODE -ne 0) { throw 'OpenSpec 1.1 task-checkoff failed' }
  & comet state task-checkoff $tasksFile '1.2 运行请求体内存保留、spool、retry/failover 聚焦测试，建立修复前的绿色行为基线与 lint RED'
  if ($LASTEXITCODE -ne 0) { throw 'OpenSpec 1.2 task-checkoff failed' }
  ```

- [x] Task 1：修复 handler、routes 与 QF1003

  **OpenSpec 映射：**
  `2.1 修复 handler 与 routes 中的无效局部 body 清零，保留可观察的 ownership/结构体字段清理，并通过 scoped lint 与 handler 测试`
  `2.2 将生产与测试文件中的 3 项 QF1003 改为语义等价 tagged switch，并通过相关 handler 测试`

  1. 运行 `$task1Red = Invoke-UncappedLintJson`，用 `Assert-LintState` 断言 144/140/3/1/39 与 `$manifestPaths`。此 task 触及的精确文件仅为 `$handlerQFPaths`，对应 baseline manifest 第 28-40 行。
  2. 先只删除 33 个 `ineffassign` 的死局部赋值。多重赋值必须拆成局部删除和可观察 owner 清理，保留 `input.SourceBody`、`input.Body`、request/response body、`CleanupRequestBodyHandle`、`Close`、defer、context snapshot 和 closure 捕获值。运行 `$afterTask1Local = Invoke-UncappedLintJson`，断言 exit 1、111/107/3/1/28 与 `$qfOnlyPaths + $gatewayPaths + $openAIPaths`。
  3. 计算 `$task1LocalIdentityDiff = Compare-LintIdentity -Previous $task1Red -Current $afterTask1Local`，运行 `Assert-NoAddedLintIdentity`，并执行 `if ($task1LocalIdentityDiff.Removed.Count -ne 33) { throw 'Task 1 local removal identity count mismatch' }`。Added/Removed 使用保留重复计数的 `(path,linter,text)` multiset，避免同文件删行引起的行号平移假阳性；reviewer 再用 `PreviousDetailed`/`CurrentDetailed` 检查详细行号，确认只关闭 manifest 中的 33 个 handler/routes `ineffassign`。
  4. 只在 `gateway_handler_responses.go:404` 和 `request_body_memory_retention_test.go:626,628` 将相同操作数的条件链改为保序 tagged switch。保持 case 顺序、default、提前 return、branch、错误映射、retained-heap 阈值与断言；不得合并 case 或移动副作用。
  5. 从 `backend` 目录对精确 `$handlerQFPaths` 运行 `gofmt -w @handlerQFPaths`，非零 exit 时 throw。运行 `$task1Green = Invoke-UncappedLintJson` 并断言 exit 1、108/107/0/1/26 与 `$gatewayPaths + $openAIPaths`；运行 handler/routes package tests、2 个 retained-heap tests、7 个 handler spool/replay tests。
  6. 计算 `$task1IdentityDiff = Compare-LintIdentity -Previous $afterTask1Local -Current $task1Green`，运行 `Assert-NoAddedLintIdentity`，并执行 `if ($task1IdentityDiff.Removed.Count -ne 3) { throw 'Task 1 QF1003 removal identity count mismatch' }`。reviewer 必须确认 Removed multiset 精确对应 3 个 QF1003，且请求构造、owner cleanup 和 error/status/payload 语义无变化。
  7. 创建唯一 implementation commit 前，运行以下精确暂存与断言：

  ```powershell
  $task1CommitPaths = @($handlerQFPaths | ForEach-Object { 'backend/' + $_ })
  & git add -- @task1CommitPaths
  Commit-ExactStaged -Label 'Task 1 implementation' -ExpectedPaths $task1CommitPaths -Message 'fix(backend): restore handler lint gate'
  ```
  8. Task 1 review 后，coordinator 按进度协议提交进度，再执行：

  ```powershell
  $progressExpected = Get-ProgressCommitExpectedPaths
  & git add -f -- @progressExpected
  Commit-ExactStaged -Label 'Task 1 progress' -ExpectedPaths $progressExpected -Message 'docs: record backend lint gate task progress'
  & comet state task-checkoff $planFile 'Task 1：修复 handler、routes 与 QF1003'
  if ($LASTEXITCODE -ne 0) { throw 'Task 1 plan task-checkoff failed' }
  & comet state task-checkoff $tasksFile '2.1 修复 handler 与 routes 中的无效局部 body 清零，保留可观察的 ownership/结构体字段清理，并通过 scoped lint 与 handler 测试'
  if ($LASTEXITCODE -ne 0) { throw 'OpenSpec 2.1 task-checkoff failed' }
  & comet state task-checkoff $tasksFile '2.2 将生产与测试文件中的 3 项 QF1003 改为语义等价 tagged switch，并通过相关 handler 测试'
  if ($LASTEXITCODE -ne 0) { throw 'OpenSpec 2.2 task-checkoff failed' }
  ```

- [ ] Task 2：修复 Gateway/Anthropic/Bedrock/Antigravity

  **OpenSpec 映射：**
  `3.1 修复通用 Gateway、Anthropic、Bedrock 与 Antigravity 路径及其测试源中的无效局部 body 清零，并通过 manifest 闭包、package 测试和内存保留矩阵`

  1. 运行 `$task2Red = Invoke-UncappedLintJson`，断言 exit 1、108/107/0/1/26 与 `$gatewayPaths + $openAIPaths`。精确 touched files 是 baseline manifest 第 48-60 行的 `$gatewayPaths`。
  2. 在 retry/failover 迭代的最后可观察读取后，只删除死局部清零。保留 `RequestBodyHandle`、`http.Request.Body`、input struct、defer、goroutine、closure、返回对象、response body 和 upstream owner cleanup；不得提取通用 helper。
  3. 对 `antigravity_gateway_service_test.go` 和 `gateway_anthropic_apikey_passthrough_test.go` 先删除 GC 辅助死局部。只有 retained-heap 测试实际回退时，保存失败输出，只在失败 branch 的物化路径以内层函数返回 handle、hash 或断言所需小值，再重跑全部 GREEN。
  4. 从 `backend` 目录对精确 `$gatewayPaths` 运行 `gofmt -w @gatewayPaths`，非零 exit 时 throw。运行 `$task2Green = Invoke-UncappedLintJson`，断言 exit 1、45/44/0/1/13 与 `$openAIPaths`；运行 service package tests、2 个 retained-heap tests、9 个 service spool/retry/failover tests。
  5. 计算 `$task2IdentityDiff = Compare-LintIdentity -Previous $task2Red -Current $task2Green`，运行 `Assert-NoAddedLintIdentity`，并执行 `if ($task2IdentityDiff.Removed.Count -ne 63) { throw 'Task 2 removal identity count mismatch' }`。reviewer 使用 detailed fingerprints 与 baseline 第 48-60 行逐项对比，确认仅移除 63 个目标而没有以同计数/同路径替换 lint identity。
  6. 创建唯一 implementation commit 前，运行以下精确暂存与断言：

  ```powershell
  $task2CommitPaths = @($gatewayPaths | ForEach-Object { 'backend/' + $_ })
  & git add -- @task2CommitPaths
  Commit-ExactStaged -Label 'Task 2 implementation' -ExpectedPaths $task2CommitPaths -Message 'fix(backend): remove dead gateway body clears'
  ```
  7. Task 2 review 后，coordinator 按进度协议提交进度，再执行：

  ```powershell
  $progressExpected = Get-ProgressCommitExpectedPaths
  & git add -f -- @progressExpected
  Commit-ExactStaged -Label 'Task 2 progress' -ExpectedPaths $progressExpected -Message 'docs: record backend lint gate task progress'
  & comet state task-checkoff $planFile 'Task 2：修复 Gateway/Anthropic/Bedrock/Antigravity'
  if ($LASTEXITCODE -ne 0) { throw 'Task 2 plan task-checkoff failed' }
  & comet state task-checkoff $tasksFile '3.1 修复通用 Gateway、Anthropic、Bedrock 与 Antigravity 路径及其测试源中的无效局部 body 清零，并通过 manifest 闭包、package 测试和内存保留矩阵'
  if ($LASTEXITCODE -ne 0) { throw 'OpenSpec 3.1 task-checkoff failed' }
  ```

- [ ] Task 3：修复 OpenAI/Gemini/Grok/unused

  **OpenSpec 映射：**
  `3.2 修复 OpenAI、Gemini、Grok 适配路径中的无效局部 body 清零，并通过 scoped lint、package 测试和 retry/failover 聚焦测试`
  `3.3 删除确认无调用方的 `sendCCUpstreamRequest` 私有方法，并完成三批 changed-file allowlist 与 issue manifest 交叉检查`

  1. 运行 `$task3Red = Invoke-UncappedLintJson`，断言 exit 1、45/44/0/1/13 与 `$openAIPaths`。精确 touched files 是 baseline manifest 第 68-80 行的 `$openAIPaths`。
  2. 运行以下命令并在非零 exit 时停止。CodeGraph 与随后 service package 编译必须都无静态、接口、callback 或测试调用方；任一调用方出现时停止，更新 Design Doc、manifest、OpenSpec tasks 和本计划后重新设计，不能用注释或伪调用保活。

  ```powershell
  codegraph explore "sendCCUpstreamRequest callers interface callback test OpenAIGatewayService"
  if ($LASTEXITCODE -ne 0) { throw 'CodeGraph caller check failed' }
  ```
  3. 在 manifest 第 68、70-80 行只删最后一次读取后的死局部；保留 `closeOpenAIRequestBody`、`CleanupRequestBodyHandle`、`resp.Body.Close`、defer、retry/failover owner 与 request context 中 owned handle。仅在双重无调用方确认后删除整个 `sendCCUpstreamRequest` 及只服务它的局部语句；不留下空壳或 fake use。
  4. 从 `backend` 目录对精确 `$openAIPaths` 运行 `gofmt -w @openAIPaths`，非零 exit 时 throw。运行 `$task3Green = Invoke-UncappedLintJson`，断言 exit 0、0 issues、三个 linter 均 0、0 paths；运行 service package tests 和 9 个 service spool/retry/failover tests。
  5. 计算 `$task3IdentityDiff = Compare-LintIdentity -Previous $task3Red -Current $task3Green`，运行 `Assert-NoAddedLintIdentity`，并执行 `if ($task3IdentityDiff.Removed.Count -ne 45) { throw 'Task 3 removal identity count mismatch' }`。reviewer 使用 detailed fingerprints 与 baseline 第 68-80 行比较，确认仅关闭 44 个 `ineffassign` 与 1 个 `unused`，不以相同计数/path 掩盖 replacement。
  6. 创建唯一 implementation commit 前，运行以下精确暂存与断言：

  ```powershell
  $task3CommitPaths = @($openAIPaths | ForEach-Object { 'backend/' + $_ })
  & git add -- @task3CommitPaths
  Commit-ExactStaged -Label 'Task 3 implementation' -ExpectedPaths $task3CommitPaths -Message 'fix(backend): remove dead OpenAI body clears'
  ```
  7. Task 3 review 后，coordinator 按进度协议提交进度，再执行：

  ```powershell
  $progressExpected = Get-ProgressCommitExpectedPaths
  & git add -f -- @progressExpected
  Commit-ExactStaged -Label 'Task 3 progress' -ExpectedPaths $progressExpected -Message 'docs: record backend lint gate task progress'
  & comet state task-checkoff $planFile 'Task 3：修复 OpenAI/Gemini/Grok/unused'
  if ($LASTEXITCODE -ne 0) { throw 'Task 3 plan task-checkoff failed' }
  & comet state task-checkoff $tasksFile '3.2 修复 OpenAI、Gemini、Grok 适配路径中的无效局部 body 清零，并通过 scoped lint、package 测试和 retry/failover 聚焦测试'
  if ($LASTEXITCODE -ne 0) { throw 'OpenSpec 3.2 task-checkoff failed' }
  & comet state task-checkoff $tasksFile '3.3 删除确认无调用方的 `sendCCUpstreamRequest` 私有方法，并完成三批 changed-file allowlist 与 issue manifest 交叉检查'
  if ($LASTEXITCODE -ne 0) { throw 'OpenSpec 3.3 task-checkoff failed' }
  ```

- [ ] Task 4：最终 gate、证据与范围断言

  **OpenSpec 映射：**
  `4.1 运行 gofmt、`git diff --check` 与 uncapped golangci-lint，确认 144 项全部关闭且结果为 0 issues`
  `4.2 重跑 backend 默认/unit 测试及请求体内存保留矩阵，确认请求体 ownership、spool、retry/failover 和上游等待语义未回退`
  `4.3 运行仓库级 `make test` 并记录 exit 0，形成供后续 upstream merge change 更新 source base 的验证证据`

  1. 从 `backend` 目录对精确 `$manifestPaths` 运行 `gofmt -w @manifestPaths`，非零 exit 时 throw。运行 `$finalLint = Invoke-UncappedLintJson`，断言 0 issues；运行完整 handler/routes 与 service package tests，及精确 2/7/9 个具名测试；从根目录运行 `make test` 并断言 exit 0。
  2. 执行 `git diff --check "$planBase..HEAD"`、`git diff --check`、`git diff --cached --check`，任一非零即 throw。用 `$protectedDiffArgs` 再次断言 protected inputs 未改，再运行 `Assert-FinalRepositoryScope`。该报告前断言必须通过，且其结果随后写入 report；此时 `git diff --name-only` 与 cached diff 中不得出现 backend 路径。
  3. 实施 agent 只能使用 `apply_patch` 创建或更新 `$reportFile`，不得使用 shell 文件写入。报告记录实际 HEAD、plan base、manifest source base、go/golangci-lint 版本、Gate 0 144/140/3/1/39、每批 JSON state 与 normalized identity review、2/7/9 匹配数、package tests、`make test`、protected blobs、base-range diff 和 allowlist 结果。
  4. report commit 前只运行以下精确暂存与断言。plan 已由 planning checkpoint 与此前进度 commits 提交，因此 report commit 不得重新暂存 plan、tasks 或其他 artifact。

  ```powershell
  & git add -f -- $reportFile
  Commit-ExactStaged -Label 'verification report' -ExpectedPaths @($reportFile) -Message 'docs: record backend lint gate verification'
  ```
  5. Task 4 review 后，coordinator 先按进度协议用 `apply_patch` 勾选 Task 4 与 OpenSpec 4.1/4.2/4.3，再提交唯一 progress allowlist，随后执行：

  ```powershell
  $progressExpected = Get-ProgressCommitExpectedPaths
  & git add -f -- @progressExpected
  Commit-ExactStaged -Label 'Task 4 progress' -ExpectedPaths $progressExpected -Message 'docs: record backend lint gate task progress'
  & comet state task-checkoff $planFile 'Task 4：最终 gate、证据与范围断言'
  if ($LASTEXITCODE -ne 0) { throw 'Task 4 plan task-checkoff failed' }
  & comet state task-checkoff $tasksFile '4.1 运行 gofmt、`git diff --check` 与 uncapped golangci-lint，确认 144 项全部关闭且结果为 0 issues'
  if ($LASTEXITCODE -ne 0) { throw 'OpenSpec 4.1 task-checkoff failed' }
  & comet state task-checkoff $tasksFile '4.2 重跑 backend 默认/unit 测试及请求体内存保留矩阵，确认请求体 ownership、spool、retry/failover 和上游等待语义未回退'
  if ($LASTEXITCODE -ne 0) { throw 'OpenSpec 4.2 task-checkoff failed' }
  & comet state task-checkoff $tasksFile '4.3 运行仓库级 `make test` 并记录 exit 0，形成供后续 upstream merge change 更新 source base 的验证证据'
  if ($LASTEXITCODE -ne 0) { throw 'OpenSpec 4.3 task-checkoff failed' }
  Assert-FinalRepositoryScope
  git diff --check "$planBase..HEAD"
  if ($LASTEXITCODE -ne 0) { throw 'post-progress base-range diff has whitespace errors' }
  ```

  6. 最终断言要求 committed backend set 精确等于 39 个 `$manifestPaths`；其他 committed paths 只可为当前 change 的 `.comet` artifacts、Design Doc、plan、baseline/report 与 `tasks.md`；staged/unstaged set 必须为空；untracked 只允许 `.comet/current-change.json`。任何额外路径均 throw。

## 完成边界

本计划只完成 `restore-backend-lint-gate` prerequisite change。不得自动恢复、切换、合并、rebase 或继续 `staged-merge-upstream-v0-1-171`；只有本 change 集成到 `main` 且新的 source base 经用户确认后，才可依照 Design Doc 第 9 节在独立流程恢复 upstream change。
