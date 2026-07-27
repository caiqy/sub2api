---
change: staged-merge-upstream-v0-1-165
design-doc: docs/superpowers/specs/2026-07-26-staged-merge-upstream-v0-1-165-design.md
base-ref: 075abc07399d6154130d2a2695fb24c785acd69c
---

# 分段合并上游 v0.1.165 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 从 `075abc07399d6154130d2a2695fb24c785acd69c` 开始，按严格顺序合入 `v0.1.160` 至 `v0.1.165`，保留本地定制并以 `0.1.165.1`、完整阶段证据和最终验证结束。

**架构：** 每个上游 tag 形成一个独立的 `--no-ff --no-commit` merge 节点；merge commit 只容纳上游树与冲突融合。测试或能力审查发现的回归必须在该 merge 之后通过独立的聚焦修复提交处理。阶段证据集中写入一份 build ledger，所有阶段通过后再创建一份最终 verify report。

**技术栈：** Git annotated tags、Go 1.26.5、Ent、Wire、Gin、Vue 3、Vite、pnpm、Vitest、PostgreSQL、Redis、Testcontainers、`ssh-skill` Python 脚本。

## 全局约束

- 初始基线固定为 `075abc07399d6154130d2a2695fb24c785acd69c`；实施分支必须由 Comet/执行者预先创建，实施前记录其名称和 `HEAD`。
- 目标仅为 `v0.1.160`、`v0.1.161`、`v0.1.162`、`v0.1.163`、`v0.1.164`、`v0.1.165`；不合并 `upstream/main` 在 `v0.1.165^{}` 之后的 `VERSION` 同步提交或任何新 release。
- 执行开始时使用 `git fetch upstream --tags --prune`。若发现 `v0.1.165` 之后的新正式 release tag，立即停止，更新 OpenSpec 和本计划后才可继续。
- 所有 tag merge 一律使用各任务写明的实际 tag 执行 `git merge --no-ff --no-commit`；merge commit 的第二父必须为下表对应 peeled SHA，且只包含上游树和为完成 merge 必需的冲突融合。
- 语义回归、补测、生成源修正和 migration 回归测试必须在 merge commit 后的独立聚焦提交中处理。复杂 scheduler、sticky、fallback、WaitPlan、DB recheck、runtime 或网关回放回归先保留失败测试，再写最小修复；纯 merge、台账和文档任务不伪造 TDD。
- 中间阶段始终保持 `backend/cmd/server/VERSION=0.1.159.6`；只能在全部六段闭合后一次更新为 `0.1.165.1`。
- 冲突台账类别只能是：上游修复、本地定制、接口/配置演进、版本/依赖、生成代码、migration。每个文件记录 ours 行为、theirs 行为、融合结果、验证证据；语义无法共存且不是 `openai-first-token-timeout` 已批准移除项时停止等待用户决定。
- 能力矩阵每行记录行为契约、入口/调用链、关键文件、受影响 tag、自动测试、人工审查点、阶段结论、证据，并只使用 `protected`、`gap`、`manual`、`approved-removal`。进入下一阶段前不得遗留 `gap`。
- Ent 只从 `backend/ent/schema/` 生成，Wire 只从 provider 声明和 `wire.go` 生成；不得手改生成结果。依赖只经对应 package manifest 和 Go module 工具融合，不手拼 lockfile 或 checksum。
- 每个阶段的本地门禁必须依次包含：受影响的聚焦测试、`make test`、`make build`、两次 `make -C backend generate`、生成 diff 检查、`git diff --check`、未合并文件检查与真实冲突标记扫描。任一失败、跳过或未解释 diff 都阻塞下一 tag。
- 每个阶段的远程门禁必须使用已提交 `HEAD` 的 `git archive` 和 `local-serv-ai` 的唯一 `/tmp` 临时目录；在 Linux 上按 `backend/scripts/test.ps1` 的等价语义重建 `backend/.test-tmp`、设置 `TMPDIR`/`TMP`/`TEMP`，并运行 `CI=true GOFLAGS='-v' go test -tags=integration ./...`。负责该门禁的 agent 必须先使用 Skill 工具加载 `ssh-skill`；只能通过其 `ssh_execute.py`、`ssh_upload.py`、`ssh_download.py` 操作，不得使用原生 SSH 或 SCP。
- 远程主机只允许 Testcontainers 拉取现有 PostgreSQL/Redis 测试镜像。禁止构建 Sub2API 镜像、禁止部署、禁止访问或写入 Sub2API 服务运行目录、禁止使用生产 PostgreSQL/Redis。
- 仅创建 `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md` 和 `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-verify.md` 两份实施证据，不创建逐任务报告。计划文件和现有 OpenSpec 工件不作为运行证据的替代物。
- 当前未跟踪的 `paseo.json` 是用户文件：不得 `git add`、覆盖、删除、移动或写入；所有 `git add` 必须使用明确路径，所有工作树检查都将它记录为排除项。
- 本 change 不执行 `git push`、不打 tag、不触发 Release workflow、不部署、不合回 `main`；验证通过也只提交 merge、必要修复、证据和 OpenSpec 状态。

## 固定 tag、证据与文件结构

| Tag | Peeled SHA | 上游区间 | 阶段重点 |
| --- | --- | --- | --- |
| `v0.1.160` | `8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62` | `v0.1.159..v0.1.160` | security audit、privacy、Grok media、image capability、181/182 migration。 |
| `v0.1.161` | `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c` | `v0.1.160..v0.1.161` | step-up、模型冷却、scheduler、Grok video、183/184 migration。 |
| `v0.1.162` | `27f094e0960ebd8e52de7ff7e763c6fec2ff4057` | `v0.1.161..v0.1.162` | trusted proxy、settings backfill/hot reload、Grok client tool cache、Sticky、image storage。 |
| `v0.1.163` | `d0bdd7e771636a8d315f542cafd39484f39bd60c` | `v0.1.162..v0.1.163` | reasoning policy、quota metadata、LastUsedAt、Cleanup、billing、185 migration。 |
| `v0.1.164` | `cd8bb98c44303b2c8f04c0da340447c992f0cb7d` | `v0.1.163..v0.1.164` | composite routing、advanced/layered scheduler、Grok/platform Sticky、Ollama、Alipay、172/186 migration。 |
| `v0.1.165` | `e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29` | `v0.1.164..v0.1.165` | OpenAI Live、prompt cache/body replay、Ollama usage、email alias、187-190 migration、postcss。 |

| 路径 | 执行时责任 |
| --- | --- |
| `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md` | 阶段 0、六个 tag 的固定对象、changed-files、冲突台账、能力矩阵、失败/修复、门禁与远程日志摘要。 |
| `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-verify.md` | 最终拓扑、全部迁移、能力矩阵终态、Chrome DevTools 烟测、OpenSpec 校验、残余风险和未执行项。 |
| `backend/internal/repository/migrations_schema_integration_test.go` | 复用隔离 PostgreSQL 数据库和 `requireMigrationApplied`，新增升级路径 migration 回归。 |
| `backend/internal/repository/migrations_runner.go` | 审查 `applyMigrationsFS` 的完整文件名排序、checksum 与 `*_notx.sql` 非事务规则；不是手工修改 migration runner 的目标。 |
| `backend/migrations/` | 保留本地 `172_video_per_second_billing_metadata.sql`、`181_group_duplicate_operation_id.sql` 及上游 12 个文件的完整文件名与内容。 |
| `backend/ent/schema/`、`backend/ent/`、`backend/cmd/server/wire.go`、`backend/cmd/server/wire_gen.go` | 仅在阶段变更触及对应源时，作为生成源、生成输出和稳定性检查范围。 |
| `openspec/changes/staged-merge-upstream-v0-1-165/tasks.md` | 仅在每项具备 ledger/verify report 证据后勾选对应 29 项。 |

## 通用阶段门禁脚本

每个阶段均在该阶段 merge 和必要兼容修复均已提交后执行。命令输出、退出码、commit SHA、远程临时目录和日志下载路径写入 build ledger 的对应阶段章节。

```powershell
# 本地全量与生成稳定性门禁
make test
make build
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git diff --check
git diff --name-only --diff-filter=U
$markers = @(git grep -n -E '^(<<<<<<< .+|=======|>>>>>>> .+)$' -- . ':!docs/superpowers/reports/**')
if ($LASTEXITCODE -notin @(0, 1)) { throw 'conflict marker scan failed' }
if ($markers.Count -ne 0) { $markers; throw 'conflict markers remain' }
```

预期：测试、构建、生成和 diff 命令退出码为 `0`，未合并文件与冲突标记集合为空，且 `backend/cmd/server/VERSION` 仍为 `0.1.159.6`，最终验证阶段除外。先运行能力矩阵命中的本地聚焦测试；migration 升级行为只由随后真正执行的远程 integration 计入。失败必须记录为 RED，并在独立修复提交后从聚焦测试开始完整重跑本段门禁。

```powershell
# 将已提交 HEAD 唯一归档并传至 local-serv-ai；每次只修改 $stage 的实际值。
$stage = 'stage-0'
$nonce = [guid]::NewGuid().ToString('N')
$archive = Join-Path $env:TEMP "sub2api-$stage-$nonce.tar"
$log = Join-Path $env:TEMP "sub2api-$stage-$nonce-integration.log"
$remote = "/tmp/sub2api-$stage-$nonce"
$requiredIntegrationTest = if ($stage -eq 'stage-0') { 'TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate' } else { 'TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages' }
$env:MSYS_NO_PATHCONV = '1'
$remoteCreated = $false

function Assert-SshResult($result, $label) {
    if (-not $result.success -or $result.exit_code -ne 0) {
        throw "$label failed: exit=$($result.exit_code) stderr=$($result.stderr)"
    }
}

try {
    if (-not (Test-Path -LiteralPath $env:TEMP)) { throw 'local temp directory is unavailable' }
    git archive --format=tar HEAD -o $archive
    if ($LASTEXITCODE -ne 0) { throw 'git archive failed' }

    $created = python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "umask 077 && mkdir -p '$remote/src'" | ConvertFrom-Json
    Assert-SshResult $created 'create remote temp'
    $remoteCreated = $true

    $uploaded = python ~/.claude/skills/ssh-skill/scripts/ssh_upload.py local-serv-ai "$archive" "$remote/source.tar" --no-progress | ConvertFrom-Json
    Assert-SshResult $uploaded 'upload source archive'

    $preflight = python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "go version && docker info" | ConvertFrom-Json
    Assert-SshResult $preflight 'remote preflight'
    $requiredGo = [version]((Get-Content backend/go.mod | Where-Object { $_ -match '^go\s+' } | Select-Object -First 1) -split '\s+')[1]
    if ($preflight.stdout -notmatch 'go(?<version>\d+\.\d+(?:\.\d+)?)') { throw 'cannot parse remote Go version' }
    if ([version]$Matches.version -lt $requiredGo) { throw "remote Go $($Matches.version) is older than $requiredGo" }

    $integration = python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; test -f '$remote/source.tar'; tar -xf '$remote/source.tar' -C '$remote/src'; cd '$remote/src/backend'; rm -rf .test-tmp; mkdir .test-tmp; CI=true GOFLAGS='-v' TMPDIR='$remote/src/backend/.test-tmp' TMP='$remote/src/backend/.test-tmp' TEMP='$remote/src/backend/.test-tmp' go test -tags=integration ./... > '$remote/integration.log' 2>&1" --timeout 1800 | ConvertFrom-Json
    $downloaded = python ~/.claude/skills/ssh-skill/scripts/ssh_download.py local-serv-ai "$remote/integration.log" "$log" --no-progress | ConvertFrom-Json
    Assert-SshResult $downloaded 'download integration log'
    Assert-SshResult $integration 'remote integration'
    $requiredPattern = '^--- PASS: ' + [regex]::Escape($requiredIntegrationTest) + '(?:\s|$)'
    if (-not (Select-String -LiteralPath $log -Pattern $requiredPattern)) { throw "required integration test did not pass: $requiredIntegrationTest" }
    $skippedTests = @(Select-String -LiteralPath $log -Pattern '^--- SKIP:')
    $skippedTests
}
finally {
    if ($remoteCreated) {
        $cleaned = python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "rm -rf '$remote' && test ! -e '$remote'" | ConvertFrom-Json
        Assert-SshResult $cleaned 'clean remote temp'
    }
    if (Test-Path -LiteralPath $archive) { Remove-Item -LiteralPath $archive }
}
```

预期：先把远程 `go version` 与当期 `backend/go.mod` 比较并记录，版本必须满足 go directive；`docker info` 成功。Linux 原生 integration 命令必须先重建 `backend/.test-tmp` 并将 `TMPDIR`/`TMP`/`TEMP` 都指向该目录，再以 `CI=true GOFLAGS='-v' go test -tags=integration ./...` 退出码 `0` 结束，且日志包含当前 `$requiredIntegrationTest` 的 PASS；这与当前 `backend/scripts/test.ps1` 的行为等价，不要求远程安装 Make 或 PowerShell。其他 `--- SKIP:` 逐项写入 ledger；仅当命中本 change 的受影响能力时阻塞。任何预检失败、Testcontainers 无法启动、目标测试未 PASS、日志下载失败或清理失败均为阶段阻塞；即使测试失败也必须下载日志并通过同一 Python 脚本清理。远程命令只处理 `$remote`，不得调用 Docker build 或接触服务运行目录。

## OpenSpec 任务映射

| OpenSpec task | 本计划任务 |
| --- | --- |
| 1.1 | 1 |
| 1.2 | 2 |
| 1.3 | 3 |
| 1.4 | 4 |
| 1.5 | 5 |
| 1.6 | 6 |
| 1.7 | 7 |
| 2.1 | 8 |
| 2.2 | 9 |
| 2.3 | 10 |
| 3.1 | 11 |
| 3.2 | 12 |
| 3.3 | 13 |
| 4.1 | 14 |
| 4.2 | 15 |
| 4.3 | 16 |
| 5.1 | 17 |
| 5.2 | 18 |
| 5.3 | 19 |
| 6.1 | 20 |
| 6.2 | 21 |
| 6.3 | 22 |
| 7.1 | 23 |
| 7.2 | 24 |
| 7.3 | 25 |
| 8.1 | 26 |
| 8.2 | 27 |
| 8.3 | 28 |
| 8.4 | 29 |

---

## 阶段 0

### Task 1：固定基线、tag 链与排除范围（OpenSpec 1.1）

**文件：**
- 创建：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`
- 审查：`.git/` refs、`backend/cmd/server/VERSION`、`paseo.json`

- [x] **步骤 1：验证起点、版本和用户未跟踪文件边界**

  执行：
  ```powershell
  git rev-parse HEAD
  Get-Content backend/cmd/server/VERSION
  git status --short
  git ls-files --error-unmatch paseo.json 2>$null
  ```

  预期：`HEAD` 是固定 base ref，版本是 `0.1.159.6`；`paseo.json` 显示为未跟踪且最后一条退出码非零。把其状态记入 ledger 的“排除的用户文件”，后续只使用显式 `git add`，绝不处理该文件。

- [x] **步骤 2：重新获取并验证完整正式 tag 链**

  执行：
  ```powershell
  git fetch upstream --tags --prune
  git rev-parse v0.1.160 "v0.1.160^{}"
  git rev-parse v0.1.161 "v0.1.161^{}"
  git rev-parse v0.1.162 "v0.1.162^{}"
  git rev-parse v0.1.163 "v0.1.163^{}"
  git rev-parse v0.1.164 "v0.1.164^{}"
  git rev-parse v0.1.165 "v0.1.165^{}"
  git merge-base --is-ancestor "v0.1.160^{}" "v0.1.161^{}"
  git merge-base --is-ancestor "v0.1.161^{}" "v0.1.162^{}"
  git merge-base --is-ancestor "v0.1.162^{}" "v0.1.163^{}"
  git merge-base --is-ancestor "v0.1.163^{}" "v0.1.164^{}"
  git merge-base --is-ancestor "v0.1.164^{}" "v0.1.165^{}"
  $officialTags = @(git tag --merged upstream/main | Where-Object { $_ -match '^v\d+\.\d+\.\d+$' } | Sort-Object { [version]$_.Substring(1) } -Descending)
  $officialTags
  if ($officialTags.Count -eq 0 -or $officialTags[0] -ne 'v0.1.165') { throw "latest upstream release changed: $($officialTags[0])" }
  git log --oneline 'v0.1.165^{}..upstream/main'
  ```

  预期：六个 peeled SHA 与固定表一致，五个祖先检查均为 `0`，上游没有比 `v0.1.165` 新的正式 release，且排除范围只含已知 `VERSION` 同步提交。否则停止，不创建 merge 节点。

### Task 2：创建隔离实施位置并初始化最小 ledger（OpenSpec 1.2）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`
- 审查：Comet 创建的工作分支/工作区、`.git/worktrees/`

- [x] **步骤 1：确认隔离分支或工作区由实施环境创建**

  执行：
  ```powershell
  git branch --show-current
  git worktree list --porcelain
  git status --short
  git show -s --format='%H%n%P%n%s' HEAD
  ```

  预期：当前工作区不是 `main` 的直接实施位置；分支由固定 base ref 创建，Task 2 开始 HEAD 允许仅包含已审查通过的 Task 1 文档提交，不得含业务或无关提交。除了文档性 change 文件和已知未跟踪 `paseo.json` 没有无关改动。把分支名、worktree 路径、Task 2 开始 HEAD 和排除文件写入 ledger。

- [x] **步骤 2：建立 ledger 的固定章节**

  写入以下章节：`固定对象与范围`、`阶段 0`、`能力矩阵`、`v0.1.160` 至 `v0.1.165`、`远程 integration 记录`、`阻塞与残余风险`。每个 tag 章节预置 `changed-files`、`冲突台账`、`能力矩阵交集`、`聚焦测试`、`本地门禁`、`远程门禁`、`放行结论` 字段；没有冲突必须明确写“无”，不能留空。

- [x] **步骤 3：提交仅限规划证据的起点提交**

  执行：
  ```powershell
  git add -f -- docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md
  git add -f -- docs/superpowers/plans/2026-07-26-staged-merge-upstream-v0-1-165.md
  git add -f -- docs/superpowers/specs/2026-07-26-staged-merge-upstream-v0-1-165-design.md
  git add openspec/changes/staged-merge-upstream-v0-1-165
  git commit -m "docs: add v0.1.165 staged merge plan"
  ```

  预期：提交不含 `paseo.json`、应用源码或生成物。若执行环境要求计划在实施前已提交，则仅记录该提交；不重复创建。

### Task 3：完成阶段 0 本地 full 门禁（OpenSpec 1.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`
- 审查：`Makefile`、`backend/Makefile`、`frontend/package.json`

- [x] **步骤 1：运行本地测试、构建、生成和静态冲突检查**

  执行“通用阶段门禁脚本”中的本地命令，并额外执行：
  ```powershell
  git diff --name-only 'v0.1.159^{}..HEAD'
  git diff --name-only --diff-filter=U
  ```

  预期：`make test` 覆盖后端默认/unit/lint 和前端 lint/typecheck/Vitest，`make build` 覆盖前后端构建，两次生成均稳定，静态检查无输出。将完整命令、退出码、失败测试名或通过摘要记入 ledger；失败为基线阻塞，禁止 merge。

### Task 4：完成阶段 0 local-serv-ai Docker-backed integration（OpenSpec 1.4）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`
- 下载到临时目录：`$env:TEMP/sub2api-stage-0-*-integration.log`

- [x] **步骤 1：仅使用通用远程门禁脚本运行 baseline integration**

  将通用脚本的 `$stage` 设为 `stage-0`。归档只能来自 `git archive HEAD`，上传、预检、测试、下载和清理只能通过列出的 `ssh-skill` Python 脚本。

  预期：远程日志证明 Linux 原生等价命令 `CI=true GOFLAGS='-v' go test -tags=integration ./...` 在重建 `backend/.test-tmp` 并设置 `TMPDIR`/`TMP`/`TEMP` 后真正执行并通过，且 `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 明确 PASS；其他 skip 已分类记录，远程目录已清理。任何必要条件不满足都在 ledger 中标为阻塞，禁止 `v0.1.160` merge。

### Task 5：记录阶段 0 生成、migration 与静态稳定性（OpenSpec 1.5）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`
- 审查：`backend/ent/schema/`、`backend/cmd/server/wire.go`、`backend/migrations/`、`backend/internal/repository/migrations_runner.go`

- [x] **步骤 1：记录两次生成和当前 migration runner 契约**

  执行：
  ```powershell
  make -C backend generate
  git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
  make -C backend generate
  git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
  git ls-tree -r --name-only HEAD backend/migrations
  git grep -n -E 'fs\.Glob|sort\.Strings|schema_migrations|checksum|_notx\.sql' -- backend/internal/repository/migrations_runner.go
  ```

  预期：两次生成没有 diff；ledger 记录现有 migration 文件清单、完整 filename 排序、filename 主键、checksum 和 `_notx.sql` 非事务执行规则。对 migration 只审查，不重命名、不手改已发布文件。

### Task 6：建立六段 changed-files x 能力矩阵（OpenSpec 1.6）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`
- 审查：`openspec/specs/`、`memory/context/upstream-merge-workflow.md`、`knowledge-base/reference/capabilities-index.md`

- [x] **步骤 1：固定每个上游区间的 changed-files 输入**

  执行：
  ```powershell
  git diff --name-only 'v0.1.159^{}..v0.1.160^{}'
  git diff --name-only 'v0.1.160^{}..v0.1.161^{}'
  git diff --name-only 'v0.1.161^{}..v0.1.162^{}'
  git diff --name-only 'v0.1.162^{}..v0.1.163^{}'
  git diff --name-only 'v0.1.163^{}..v0.1.164^{}'
  git diff --name-only 'v0.1.164^{}..v0.1.165^{}'
  ```

  预期：ledger 保存六段实际文件清单和每段与本地关键文件的交集，不以设计风险表替代真实 diff。

- [x] **步骤 2：填充能力矩阵并对核心入口记录调用链**

  矩阵至少覆盖 advanced/layered scheduler、fallback/WaitPlan、DB recheck、Grok/platform Sticky、privacy、image capability、异步图片/对象存储、图片和视频计费、上游倍率、session/step-up、runtime 热更新、网关透传、prompt cache、body replay/spooling、失败 usage、用户资源控制、公开分组屏蔽、菜单隐藏、前端翻译、quota 原子重置、settings backfill、Ent/Wire、依赖、migration、local test gates 和 `openai-first-token-timeout`。

  对命中入口使用 CodeGraph `context`、`trace` 或 `impact` 记录真实调用链；`openai-first-token-timeout` 是唯一允许标记 `approved-removal` 的行，其余行不得静默移除。每行列出准确的现有聚焦测试命令或 `manual` 结构证据。

### Task 7：补齐阶段 0 保护断言并关闭基线（OpenSpec 1.7）

**文件：**
- 修改：能力矩阵确定的现有 `backend/**/*_test.go` 或 `frontend/**/*.spec.ts`
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [x] **步骤 1：只对 `gap` 行先写最小失败测试**

  每个 `gap` 必须先在矩阵所列包或组件中添加一个直接断言本地行为的测试，并在 ledger 中写入含真实包名、测试名或 Vitest 文件路径的完整命令后执行，确认 RED。不得把示意占位符复制为命令。没有 `gap` 时，在 ledger 写明零项和判断依据，不创建无目的测试。

- [x] **步骤 2：令保护测试在当前基线通过并重跑阶段 0 全门禁**

  执行每条新增聚焦命令，再依次完成 Task 3、Task 4、Task 5 的全部命令。预期：所有 `gap` 转为 `protected` 或有可复现 `manual` 证据；无 `gap`、无静态冲突、远程 integration 通过后才可进入 Task 8。

- [x] **步骤 3：提交阶段 0 测试与证据并封闭阶段**

  执行：
  ```powershell
  git add docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md
  git commit -m "test: protect local behavior before v0.1.160"
  ```

  提交前，对矩阵记录的真实测试路径逐个执行 `git add --`，严禁 `git add .`。预期：仅有实际必要的测试和 ledger。无新增测试时只暂存 ledger，并使用 `git commit -m "docs: close v0.1.165 stage zero"`。`paseo.json` 永不在暂存区。

## v0.1.160

### Task 8：合入并记账 v0.1.160（OpenSpec 2.1）

**文件：**
- 修改：本次 Git merge 实际触及文件
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [x] **步骤 1：确认上一段闭合并启动唯一允许的 merge 形式**

  执行：
  ```powershell
  git status --short
  git rev-parse "v0.1.160^{}"
  git merge --no-ff --no-commit v0.1.160
  git diff --name-only --diff-filter=U
  git diff --cached --check
  ```

  预期：tag peeled SHA 为 `8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62`；merge 停在可检查状态。绝不改用 squash、cherry-pick 或机械 ours/theirs。

- [x] **步骤 2：逐文件融合并提交仅含上游树的 merge 节点**

  对冲突及无文本冲突的 security-audit full prompt/privacy、Grok media 隔离、`image_gen` 权限、`181_prompt_audit.sql`、`182_prompt_audit_full_prompt.sql`、本地 `181_group_duplicate_operation_id.sql` 逐项记录冲突台账和能力矩阵交集。确认完整文件名均保留。然后执行：
  ```powershell
  git diff --cached --check
  git commit -m "merge: upstream v0.1.160"
  git show -s --format='%H%n%P%n%s' HEAD
  ```

  预期：第二父精确为固定 peeled SHA，merge commit 不含 ledger、测试或回归修复。

### Task 9：修复 v0.1.160 回归并执行 full 门禁（OpenSpec 2.2）

**文件：**
- 修改：能力矩阵交集和 RED 失败确定的实际后端/前端文件
- 修改：`backend/internal/repository/migrations_schema_integration_test.go`
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [x] **步骤 1：建立可随后续 tag 自动扩展的 migration 升级回归**

  新增 `TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages`，复用 `newEmptyIsolatedMigrationDB`、`applyMigrationsFS`、`requireMigrationApplied` 和嵌入的 `migrations.FS`。测试维护本次 12 个上游 filename 的固定集合，并：

  1. 从当前 embedded FS 中识别已经随 tag 出现的集合；v0.1.160 阶段必须恰好包含 181/182 两个文件；
  2. 构造排除当前已出现上游文件的 `fstest.MapFS`，应用后断言本地 `172_video_per_second_billing_metadata.sql` 和 `181_group_duplicate_operation_id.sql` 已记录；
  3. 对同一数据库应用完整 FS，断言当前已出现的每个上游 filename 均记录；
  4. 保存相关 filename 的 checksum 与记录数，第二次应用完整 FS 后断言两者不变；
  5. 当 `190_*_notx.sql` 在最终阶段出现时，沿现有非事务路径验证成功。

  提交该保护测试：
  ```powershell
  git add backend/internal/repository/migrations_schema_integration_test.go
  git commit -m "test: cover staged migration upgrades"
  ```

  随后用 `$stage = 'v0.1.160'` 执行通用远程门禁。若测试直接通过，记录为对现有 filename runner 行为的保护；若失败，则保留该远程 RED，再进入步骤 2 做最小修复。禁止把本机缺少 Docker 导致的 skip 当作 GREEN。

- [x] **步骤 2：运行本段能力测试并以独立聚焦提交处理真实回归**

  执行矩阵为本段列出的 scheduler、Sticky、privacy、image capability 和 audit 聚焦命令。只在 RED 已证明回归时写最小修复和所需测试；提交前逐个暂存 ledger 记录的真实路径，严禁 `git add .`。然后执行：
  ```powershell
  git commit -m "fix: preserve local behavior after v0.1.160"
  ```

  预期：没有回归不创建空提交；修复不回写 merge commit。

- [x] **步骤 3：运行本地、生成、静态与远程全门禁**

  完成通用阶段门禁脚本，`$stage` 设为 `v0.1.160`，并运行矩阵全部聚焦命令。预期：本地 `make test`、`make build`、两次生成、静态冲突检查和 local-serv-ai integration 都通过；新增 upgrade test 证明本地 172/181 与当前上游 181/182 可升级、幂等且 checksum 稳定。失败后停在本段并从本任务重跑。

### Task 10：关闭 v0.1.160 能力矩阵与证据（OpenSpec 2.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [x] **步骤 1：记录本段 changed-files、冲突台账、能力结论和门禁证据**

  预期：矩阵中该 tag 的每个交集都有 `protected` 或 `manual` 结论、命令/调用链证据和阶段结果；无遗留 `gap`。ledger 写入 merge SHA、第二父、修复 SHA、远程日志路径与清理结果。

- [x] **步骤 2：提交证据并判定下一段入口**

  执行：
  ```powershell
  git add docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md
  git commit -m "docs: close v0.1.160 stage gate"
  ```

  预期：本段所有门禁真实通过才允许 Task 11；`VERSION` 仍为 `0.1.159.6`。

## v0.1.161

### Task 11：合入并记账 v0.1.161（OpenSpec 3.1）

**文件：**
- 修改：本次 Git merge 实际触及文件
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：创建 v0.1.161 的受审 merge 状态**

  执行：
  ```powershell
  git rev-parse "v0.1.161^{}"
  git merge --no-ff --no-commit v0.1.161
  git diff --name-only --diff-filter=U
  git diff --cached --check
  ```

  预期：peeled SHA 为 `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c`。

- [ ] **步骤 2：融合 step-up、scheduler 冷却、Grok 视频和 183/184 migration 后提交 merge**

  对 step-up 2FA 开关、模型级临时冷却与本地 scheduler、Grok 视频代理、`183_ops_ingress_reject_aggregates.sql`、`184_auth_cache_invalidation_outbox.sql` 的每项冲突或无冲突调用链更新 ledger。然后执行：
  ```powershell
  git commit -m "merge: upstream v0.1.161"
  git show -s --format='%H%n%P%n%s' HEAD
  ```

  预期：第二父为固定 SHA；只有冲突融合进入该提交。

### Task 12：修复 v0.1.161 回归并执行 full 门禁（OpenSpec 3.2）

**文件：**
- 修改：RED 失败确定的 scheduler、认证、Grok 或 migration 实际路径
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：优先运行复杂路径的失败测试**

  对模型冷却、advanced/layered scheduler、fallback/WaitPlan、DB recheck、session/step-up、Grok 视频代理运行矩阵聚焦测试。若上游触及路径但没有断言，先写一个最小失败测试，再修复；不对纯 merge 内容虚构 RED。

- [ ] **步骤 2：独立提交最小兼容修复并重跑受影响测试**

  提交前对 ledger 中 RED 证据关联的真实源码和测试路径逐个执行 `git add --`，严禁暂存其他文件。然后执行：
  ```powershell
  git commit -m "fix: preserve local behavior after v0.1.161"
  ```

  预期：仅在确有回归时提交，修复后全部聚焦测试 GREEN。

- [ ] **步骤 3：执行通用阶段门禁**

  用 `$stage = 'v0.1.161'` 执行本地全门禁与 remote integration；远程 migration 断言本地 172/181、当前已存在上游 181-184 全部按 filename 可升级、幂等且 checksum 稳定。预期：所有门禁通过，否则阻塞本段。

### Task 13：关闭 v0.1.161 能力矩阵与证据（OpenSpec 3.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：记录调用链、冲突台账和 full 门禁结论**

  预期：每项 changed-files 交集明确保留或经人工审查，阶段结果没有 `gap`。

- [ ] **步骤 2：提交封闭证据**

  执行：
  ```powershell
  git add docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md
  git commit -m "docs: close v0.1.161 stage gate"
  ```

  预期：`VERSION=0.1.159.6`，且只有证据文件进入提交。

## v0.1.162

### Task 14：合入并记账 v0.1.162（OpenSpec 4.1）

**文件：**
- 修改：本次 Git merge 实际触及文件
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：创建受审 merge 状态**

  执行：
  ```powershell
  git rev-parse "v0.1.162^{}"
  git merge --no-ff --no-commit v0.1.162
  git diff --name-only --diff-filter=U
  git diff --cached --check
  ```

  预期：peeled SHA 为 `27f094e0960ebd8e52de7ff7e763c6fec2ff4057`。

- [ ] **步骤 2：融合 trusted proxy、Grok cache/Sticky、S3/image storage**

  对客户端 IP 请求头与可信代理体系、Grok client tool cache 与 sticky、S3 备份/image storage 逐项记录 ours/theirs、入口调用链和融合结论，然后执行：
  ```powershell
  git commit -m "merge: upstream v0.1.162"
  git show -s --format='%H%n%P%n%s' HEAD
  ```

  预期：第二父为 `27f094e0960ebd8e52de7ff7e763c6fec2ff4057`，不混入修复。

### Task 15：修复 v0.1.162 回归并执行 full 门禁（OpenSpec 4.2）

**文件：**
- 修改：RED 失败确定的 proxy、settings、Grok、storage 或前端实际路径
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：以失败测试审查 proxy、runtime 与 Sticky 边界**

  运行矩阵中 trusted proxy、settings JSON backfill、热更新、Grok client tool cache、platform/session Sticky、异步图片和对象存储的聚焦测试。复杂行为发现缺口时先 RED 再最小修复。

- [ ] **步骤 2：提交必要的最小修复并执行全部门禁**

  提交前对 ledger 中 RED 证据关联的真实源码和测试路径逐个执行 `git add --`，严禁暂存其他文件。然后执行：
  ```powershell
  git commit -m "fix: preserve local behavior after v0.1.162"
  ```

  无回归则不提交。随后以 `$stage = 'v0.1.162'` 执行通用本地与远程门禁；远程 upgrade migration 目标测试必须明确 PASS，不能被 skip。

### Task 16：关闭 v0.1.162 专项审查与证据（OpenSpec 4.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：记录 settings backfill 与热更新调用链结论**

  预期：ledger 明确 settings JSON 的迁移/backfill、配置解析、运行时缓存重建/重载均未被绕过；同时记录 proxy、Sticky、S3/image storage 的自动或人工证据。

- [ ] **步骤 2：提交阶段封闭证据**

  执行：
  ```powershell
  git add docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md
  git commit -m "docs: close v0.1.162 stage gate"
  ```

  预期：无 `gap`、无冲突标记、`VERSION=0.1.159.6`。

## v0.1.163

### Task 17：合入并记账 v0.1.163（OpenSpec 5.1）

**文件：**
- 修改：本次 Git merge 实际触及文件
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：创建受审 merge 状态**

  执行：
  ```powershell
  git rev-parse "v0.1.163^{}"
  git merge --no-ff --no-commit v0.1.163
  git diff --name-only --diff-filter=U
  git diff --cached --check
  ```

  预期：peeled SHA 为 `d0bdd7e771636a8d315f542cafd39484f39bd60c`。

- [ ] **步骤 2：融合 reasoning、scheduler metadata、Cleanup、计费与依赖**

  审查 OpenAI reasoning policy、scheduler quota metadata/`LastUsedAt`、优雅关停 Cleanup、计费修复、axios 安全升级和 `185_group_reasoning_effort_policy.sql`。完成台账后执行：
  ```powershell
  git commit -m "merge: upstream v0.1.163"
  git show -s --format='%H%n%P%n%s' HEAD
  ```

  预期：第二父为固定 SHA，merge commit 只含融合结果。

### Task 18：修复 v0.1.163 回归并执行 full 门禁（OpenSpec 5.2）

**文件：**
- 修改：RED 失败确定的 reasoning、scheduler、Cleanup、billing、依赖或 migration 实际路径
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：先对 scheduler、fallback 与 Cleanup 运行失败测试**

  执行矩阵指定的 scheduler quota、`LastUsedAt`、fallback/WaitPlan、DB recheck、失败 usage、优雅关停与计费聚焦命令。行为回归必须先 RED；lockfile/manifest 融合则以已有 pnpm 与 build 结果作证据。

- [ ] **步骤 2：独立修复、重跑聚焦测试和通用门禁**

  需要修复时，先对 ledger 中 RED 证据关联的真实源码和测试路径逐个执行 `git add --`，再执行：
  ```powershell
  git commit -m "fix: preserve local behavior after v0.1.163"
  ```

  随后用 `$stage = 'v0.1.163'` 执行通用门禁。预期：本地和 remote Docker-backed integration 均通过，migration upgrade 记录本阶段 185 与既存完整文件名，且 checksum 稳定。

### Task 19：关闭 v0.1.163 能力矩阵与证据（OpenSpec 5.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：记录能力审查和阶段门禁**

  预期：reasoning policy、scheduler metadata、Cleanup、billing、axios 与 migration 的 changed-files 交集均有证据；无 `gap`。

- [ ] **步骤 2：提交阶段封闭证据**

  执行：
  ```powershell
  git add docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md
  git commit -m "docs: close v0.1.163 stage gate"
  ```

  预期：`VERSION` 没有过程版本变更。

## v0.1.164

### Task 20：合入并记账 v0.1.164（OpenSpec 6.1）

**文件：**
- 修改：本次 Git merge 实际触及文件
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：创建受审 merge 状态**

  执行：
  ```powershell
  git rev-parse "v0.1.164^{}"
  git merge --no-ff --no-commit v0.1.164
  git diff --name-only --diff-filter=U
  git diff --cached --check
  ```

  预期：peeled SHA 为 `cd8bb98c44303b2c8f04c0da340447c992f0cb7d`。

- [ ] **步骤 2：融合 composite routing、Ollama、Grok 冷却、Alipay 和 migration**

  对 composite group routing、Ollama Cloud、Grok 402 冷却、Alipay deep link、`172_composite_model_routes.sql` 与本地 `172_video_per_second_billing_metadata.sql`、两个 `186_*` 完成台账。两份 172 都必须保留。然后执行：
  ```powershell
  git commit -m "merge: upstream v0.1.164"
  git show -s --format='%H%n%P%n%s' HEAD
  ```

  预期：第二父为 `cd8bb98c44303b2c8f04c0da340447c992f0cb7d`，不把语义修复混入。

### Task 21：修复 v0.1.164 回归并执行 full 门禁（OpenSpec 6.2）

**文件：**
- 修改：RED 失败确定的 routing、scheduler、Sticky、Ollama、支付或 migration 实际路径
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：先以失败测试验证 composite routing 不绕过本地调度**

  对 composite group routing 入口使用 CodeGraph `trace` 记录到 scheduler factory、advanced/layered scheduler、Grok/platform Sticky、fallback/WaitPlan 与 DB recheck 的完整调用链。运行矩阵已有测试；少断言时先添加最小 RED，确认路由不能绕过本地调度定制。

- [ ] **步骤 2：提交最小修复并执行完整门禁**

  需要修复时，先对 ledger 中 RED 证据关联的真实源码和测试路径逐个执行 `git add --`，再执行：
  ```powershell
  git commit -m "fix: preserve local behavior after v0.1.164"
  ```

  随后以 `$stage = 'v0.1.164'` 执行通用门禁。远程 migration upgrade 必须验证本地/上游两份 172、两份 186 作为不同完整 filename 被记录，重复运行无新增记录且 checksum 不变。

### Task 22：关闭 v0.1.164 专项审查与证据（OpenSpec 6.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：记录 routing 到 scheduler/Sticky 的不绕过证据**

  预期：ledger 链接入口调用链、失败/通过测试、两份 172 与两份 186 的 migration 证据，以及本地和远程门禁结果。

- [ ] **步骤 2：提交阶段封闭证据**

  执行：
  ```powershell
  git add docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md
  git commit -m "docs: close v0.1.164 stage gate"
  ```

  预期：只有该阶段证据进入提交；失败或 `gap` 不得开始 Task 23。

## v0.1.165

### Task 23：合入并记账 v0.1.165（OpenSpec 7.1）

**文件：**
- 修改：本次 Git merge 实际触及文件
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：创建最终 tag 的受审 merge 状态**

  执行：
  ```powershell
  git rev-parse "v0.1.165^{}"
  git merge --no-ff --no-commit v0.1.165
  git diff --name-only --diff-filter=U
  git diff --cached --check
  ```

  预期：peeled SHA 为 `e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29`；不得合并 `upstream/main`。

- [ ] **步骤 2：融合 Live、body 生命周期、usage、alias、migration 与 postcss**

  对 OpenAI Live gateway、Ollama 用量刷新、email alias 注册查重、`187_add_usage_log_session_id.sql`、`188_allow_live_usage_request_type.sql`、`189_add_group_allow_live.sql`、`190_add_users_email_alias_dedup_index_notx.sql` 和 postcss 安全升级逐项记账。然后执行：
  ```powershell
  git commit -m "merge: upstream v0.1.165"
  git show -s --format='%H%n%P%n%s' HEAD
  ```

  预期：第二父为固定 SHA，merge commit 未包含 migration 测试或回归修复。

### Task 24：修复 v0.1.165 回归并执行 full 门禁（OpenSpec 7.2）

**文件：**
- 修改：RED 失败确定的 OpenAI gateway、prompt cache、request-body、Ollama、auth、frontend 或 migration 实际路径
- 修改：`backend/internal/repository/migrations_schema_integration_test.go`（收紧最终 12/12 断言）
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：将渐进式 migration 升级回归收紧为最终 12/12**

  更新 Task 9 已新增的 `TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages`：在保留过滤 FS、完整 FS、重复执行、checksum 与本地 172/181 断言的基础上，明确要求固定上游集合在当前 FS 中为 12/12，并要求两个上游 186 及 `190_*_notx.sql` 均已记录。然后执行：

  ```powershell
  git add backend/internal/repository/migrations_schema_integration_test.go
  git commit -m "test: complete staged migration upgrade coverage"
  ```

  用 `$stage = 'v0.1.165'` 运行通用远程门禁。若失败，保留远程 RED 后再做最小修复；若直接通过，记录现有 runner 已满足完整文件名升级契约。不得在本机把 integration skip 当作验证，也不得引入新的 migration runner 抽象。

- [ ] **步骤 2：先验证 OpenAI Live 与本地请求体语义，再最小修复**

  用 CodeGraph `trace` 审查 OpenAI Live gateway 入口至 prompt cache reuse、body replay/spooling、cleanup、失败 usage 的路径。对矩阵命中路径运行聚焦测试；任何回归必须先 RED，再最小修复。提交前对 ledger 中 RED 证据关联的真实路径逐个执行 `git add --`，需要提交时执行：
  ```powershell
  git commit -m "fix: preserve local behavior after v0.1.165"
  ```

  预期：本地 OpenAI 定制未被 Live 改写；无回归时不创建空修复提交，migration 测试已由步骤 1 独立提交。

- [ ] **步骤 3：完成本段全门禁**

  执行所有矩阵聚焦命令，再以 `$stage = 'v0.1.165'` 完整重跑通用本地/远程门禁。预期：远程 integration 覆盖新库与完整升级路径，未跳过；本地两次生成稳定；静态冲突检查为空。

### Task 25：关闭 v0.1.165 专项审查与证据（OpenSpec 7.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：记录 OpenAI Live 与本地 OpenAI 定制的交互结论**

  ledger 必须链接 gateway 入口调用链、prompt cache reuse、body replay/spooling、cleanup 和失败 usage 的测试或结构证据，并记录迁移回归的 RED/GREEN、12/12 filename、两个本地同号文件、两个 186、190 notx、幂等/checksum 结论。

- [ ] **步骤 2：提交最终 tag 阶段证据**

  执行：
  ```powershell
  git add docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md
  git commit -m "docs: close v0.1.165 stage gate"
  ```

  预期：六段均已关闭，但版本仍为 `0.1.159.6`；尚未执行发布相关操作。

## 最终验证

### Task 26：一次更新最终版本并检查已批准移除项（OpenSpec 8.1）

**文件：**
- 修改：`backend/cmd/server/VERSION`
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`

- [ ] **步骤 1：在六个阶段全部通过后更新唯一版本文件**

  将 `backend/cmd/server/VERSION` 的唯一内容改为：
  ```text
  0.1.165.1
  ```

  然后执行：
  ```powershell
  git diff -- backend/cmd/server/VERSION
  $legacyFirstToken = @(git grep -n -E 'openai_text_first_token_timeout|openai_image_first_token_timeout|first_token_timeout|gateway\.openai_first_token_timeout|OpenAIFirstTokenTimeout|openAIFirstTokenWatchdog' -- backend frontend deploy ':!docs/superpowers/reports/**')
  if ($LASTEXITCODE -notin @(0, 1)) { throw 'first-token removal scan failed' }
  if ($legacyFirstToken.Count -ne 0) { $legacyFirstToken; throw 'removed first-token contract returned' }
  ```

  预期：仅版本文件改变，旧本地首 Token 超时业务符号无输出。上游 `first_output_timeout` 行为不是扫描目标，必须保留。

- [ ] **步骤 2：提交最终版本元数据**

  执行：
  ```powershell
  git add backend/cmd/server/VERSION
  git add docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md
  git commit -m "chore: set version to 0.1.165.1"
  ```

  预期：该提交只含版本与其证据，不含 `paseo.json`。

### Task 27：执行最终 full verify（OpenSpec 8.2）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-verify.md`
- 审查：所有矩阵测试和 `backend/internal/repository/migrations_schema_integration_test.go`

- [ ] **步骤 1：重跑全量本地和生成门禁**

  执行通用本地门禁脚本，并再次运行能力矩阵中的所有非 integration 聚焦测试：
  ```powershell
  git diff --check
  ```

  预期：`make test`、`make build`、两次生成和静态检查均通过，版本精确为 `0.1.165.1`。Docker-backed migration 回归只在步骤 2 的远程 integration 中计为行为验证。

- [ ] **步骤 2：重跑最终 local-serv-ai integration**

  用 `$stage = 'final-verify'` 执行通用远程门禁脚本。预期：提交后的最终 `HEAD` 被 `git archive` 打包，远程预检和 Linux 原生等价命令 `CI=true GOFLAGS='-v' go test -tags=integration ./...` 成功，`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` 明确 PASS 并验证最终 12/12；其他 skip 已分类记录，日志下载且远程临时目录已删除。

- [ ] **步骤 3：记录最终自动验证结果**

  报告记录全部命令、退出码、远程日志位置、未执行项和残余风险。远程前置失败或清理异常必须标为失败，不能写为 manual pass。

### Task 28：验证拓扑、冲突、migration 与范围边界（OpenSpec 8.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-verify.md`
- 审查：`.git/` 图、`backend/migrations/`、`backend/internal/repository/migrations_runner.go`

- [ ] **步骤 1：验证六个 tag 祖先和六个 merge 第二父**

  执行：
  ```powershell
  git merge-base --is-ancestor 8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62 HEAD
  git merge-base --is-ancestor 19149ca196eeae4a4482e5299dc6fa4ba0b06c8c HEAD
  git merge-base --is-ancestor 27f094e0960ebd8e52de7ff7e763c6fec2ff4057 HEAD
  git merge-base --is-ancestor d0bdd7e771636a8d315f542cafd39484f39bd60c HEAD
  git merge-base --is-ancestor cd8bb98c44303b2c8f04c0da340447c992f0cb7d HEAD
  git merge-base --is-ancestor e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29 HEAD
  git log --first-parent --merges --format='%H %P %s' '075abc07399d6154130d2a2695fb24c785acd69c..HEAD'
  git merge-base --is-ancestor upstream/main HEAD
  ```

  预期：前六条退出码 `0`；first-parent 历史按六个 tag 顺序显示且第二父逐一等于固定表；最后一条必须非零，证明未把 release 后的 `upstream/main` 合入，并将此预期非零记录为范围证据。

- [ ] **步骤 2：验证静态冲突、版本和全部 migration 集合**

  执行：
  ```powershell
  Get-Content backend/cmd/server/VERSION
  git diff --check
  git diff --name-only --diff-filter=U
  $markers = @(git grep -n -E '^(<<<<<<< .+|=======|>>>>>>> .+)$' -- . ':!docs/superpowers/reports/**')
  if ($LASTEXITCODE -notin @(0, 1)) { throw 'conflict marker scan failed' }
  if ($markers.Count -ne 0) { $markers; throw 'conflict markers remain' }
  $migrationMatches = @(git ls-tree -r --name-only HEAD backend/migrations | Select-String -Pattern '172_composite_model_routes.sql|172_video_per_second_billing_metadata.sql|181_prompt_audit.sql|181_group_duplicate_operation_id.sql|182_prompt_audit_full_prompt.sql|183_ops_ingress_reject_aggregates.sql|184_auth_cache_invalidation_outbox.sql|185_group_reasoning_effort_policy.sql|186_alipay_mobile_precreate_deep_link.sql|186_group_auth_cache_image_generation.sql|187_add_usage_log_session_id.sql|188_allow_live_usage_request_type.sql|189_add_group_allow_live.sql|190_add_users_email_alias_dedup_index_notx.sql')
  if ($migrationMatches.Count -ne 14) { $migrationMatches; throw "expected 14 protected migrations, got $($migrationMatches.Count)" }
  ```

  预期：版本为 `0.1.165.1`，静态检查无输出，14 个列出的 migration filename 均存在。verify report 链接 Task 24 的升级测试，说明新库/升级库、同号 172/181、双 186、190 notx、幂等与 checksum 均通过。

### Task 29：完成能力级最终审查、浏览器烟测与 OpenSpec 收口（OpenSpec 8.4）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-verify.md`
- 修改：`openspec/changes/staged-merge-upstream-v0-1-165/tasks.md`

- [ ] **步骤 1：逐项关闭能力矩阵**

  对 advanced/layered scheduler、fallback/WaitPlan、DB recheck、所有平台 Sticky、privacy、image capability、异步图片/对象存储、计费/倍率、session/step-up、runtime 热更新、网关透传、prompt cache、body replay/spooling、失败 usage、用户资源控制、公开分组屏蔽、菜单隐藏、前端翻译、quota 原子重置、settings backfill、Ent/Wire、依赖、migration、local gates 逐项确认入口可达、边界成立、自动或人工证据可复现。只允许 `openai-first-token-timeout` 维持 `approved-removal`；最终不得有 `gap`。

- [ ] **步骤 2：运行前端关键后台页面烟测**

  先读取 `memory/context/frontend-debug-preview.md`，确认本地后端/API 与管理员会话可用，再启动：
  ```powershell
  pnpm --dir frontend dev --host 127.0.0.1 --port 5173
  ```

  使用 Skill 工具加载 `chrome-devtools`，打开 `http://127.0.0.1:5173` 下由 changed-files/能力矩阵确认实际存在的客户端 IP、step-up 2FA、S3/image storage、security audit、Alipay 等后台路由。记录每页加载、关键网络请求和控制台结果；不存在的路由不虚构测试，后端/API 或管理员会话不可用则记录为阻塞而非通过。浏览器验证结束后停止该开发服务。

- [ ] **步骤 3：校验 OpenSpec、勾选具备证据的 29 项并提交最终文档**

  先解析可用的 OpenSpec 1.5+ CLI 并校验 change：
  ```powershell
  $openspecCommand = Get-Command openspec -ErrorAction SilentlyContinue
  if ($openspecCommand) {
      $openspec = $openspecCommand.Source
  } else {
      $npmRoot = (npm root -g).Trim()
      $openspec = Join-Path $npmRoot '@rpamis/comet/node_modules/.bin/openspec.cmd'
  }
  if (-not (Test-Path -LiteralPath $openspec)) { throw 'OpenSpec CLI not found' }
  $openspecVersion = [version]((& $openspec --version).Trim())
  if ($openspecVersion -lt [version]'1.5.0') { throw "OpenSpec $openspecVersion is unsupported" }
  & $openspec validate staged-merge-upstream-v0-1-165
  if ($LASTEXITCODE -ne 0) { throw 'OpenSpec validation failed' }
  ```

  确认通过后只勾选有 ledger 或 verify report 证据的 `tasks.md` 项。然后执行：
  ```powershell
  git add docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md
  git add -f -- docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-verify.md
  git add openspec/changes/staged-merge-upstream-v0-1-165/tasks.md
  git commit -m "docs: verify staged upstream merge v0.1.165"
  git status --short
  ```

  预期：29 项均有直接证据时才全部勾选；最终提交只含两份证据和 tasks 状态，`paseo.json` 仍未跟踪且未触碰。到此停止：不 push、不打 tag、不触发 Release workflow、不部署、不合回 `main`。

## 计划自检

- OpenSpec 29 项映射：Task 1-29 与 `tasks.md` 的 1.1-8.4 一一对应，无漏项、无扩展范围。
- 阶段完整性：阶段 0、六个 tag 和最终验证均包含冲突台账/能力矩阵、受影响测试、`make test`、`make build`、两次 `make -C backend generate`、静态冲突检查、local-serv-ai Docker-backed integration 与证据封闭；每段失败均阻塞下一段。
- migration 覆盖：测试以过滤 embedded FS 模拟本地 `0.1.159.6`，后以完整 FS 升级；覆盖本地/上游同号 172/181、两个上游 186、190 notx、幂等和 checksum。
- 远程限制：所有远程命令均通过 `ssh-skill` Python 脚本；计划没有原生 SSH/SCP、服务器 Sub2API 镜像构建、发布、推送或部署步骤。
- 占位符检查：本文不含未完成占位标记；运行时由实际 diff 决定的文件使用明确的 RED 证据和矩阵路径约束，不允许无依据暂存。
