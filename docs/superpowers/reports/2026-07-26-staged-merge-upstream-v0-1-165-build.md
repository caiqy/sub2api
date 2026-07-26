# 分段合入上游 v0.1.165 构建台账

## 固定对象与范围

- 分支：`feature/20260726/staged-merge-upstream-v0-1-165`。
- 固定分支创建 base：`075abc07399d6154130d2a2695fb24c785acd69c`。
- `backend/cmd/server/VERSION`：`0.1.159.6`。
- 当前工作区：`D:/Caiqy/Projects/Github/sub2api`；它是主工作树，隔离形式为独立 feature 分支，而非额外 linked worktree。另有两个与本 change 无关的 detached worktree，未触碰。
- Task 2 开始 HEAD：`f1ad4a6da432e005d904f1deb1f1ab9bd339df63`，不是 base ref。此前的 Task 1 文档提交链为：
  - `f5656d5ef6b8dd4d93b10b7779f044e14ca8f43f docs: record staged merge baseline`（父提交：`075abc07399d6154130d2a2695fb24c785acd69c`）
  - `6e18ca4270109b098940223c4a9b317f41aa4292 docs: localize staged merge baseline report`（父提交：`f5656d5ef6b8dd4d93b10b7779f044e14ca8f43f`）
  - `f1ad4a6da432e005d904f1deb1f1ab9bd339df63 docs: translate task 1 ledger title`（父提交：`6e18ca4270109b098940223c4a9b317f41aa4292`）
- Task 2 主规划提交 HEAD：`53fbd1f83dee72ddfb459b96c964508ca732a962`。

| Tag | Tag 对象 | Peeled SHA |
| --- | --- | --- |
| `v0.1.160` | `2a519c0f8878aa8d9d75918e3acd734e536cc675` | `8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62` |
| `v0.1.161` | `317df5405c0ff1c67f12dcc0c669a16fc2e21dac` | `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c` |
| `v0.1.162` | `34b7a5ad70b4b9b9bb96955562fe632ad625d783` | `27f094e0960ebd8e52de7ff7e763c6fec2ff4057` |
| `v0.1.163` | `bb752ef7776dc126ffca5df9188087d0d0aed559` | `d0bdd7e771636a8d315f542cafd39484f39bd60c` |
| `v0.1.164` | `38a46fd33795c8946a1e88d0f72597c79ca02a76` | `cd8bb98c44303b2c8f04c0da340447c992f0cb7d` |
| `v0.1.165` | `892c8fa3ab80ada8a624668808c3e575da7c04d5` | `e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29` |

- 已验证的相邻 peeled tag 祖先链：`v0.1.160 -> v0.1.161 -> v0.1.162 -> v0.1.163 -> v0.1.164 -> v0.1.165`；全部 `git merge-base --is-ancestor` 检查以退出码 `0` 通过。
- release 上界：已合入 `upstream/main` 的最新正式 tag 是 `v0.1.165`。
- 排除命令：`git log --oneline 'v0.1.165^{}..upstream/main'`。
- 唯一排除提交：`2730c1c43b29be003925b033f3f9e645e726bb8c chore: sync VERSION to 0.1.165 [skip ci]`。
- `paseo.json` 是未跟踪用户文件，继续排除在本任务之外，绝不暂存、修改、删除或移动。
- 根目录 `.comet/current-change.json` 是本地 Comet selector，继续排除在本任务提交之外，绝不暂存、修改、删除或移动。

## 阶段 0

- Task 1 的固定基线、tag 链、release 上界和排除提交证据已记录在本台账的“固定对象与范围”。
- Task 2 提交前快照：不在 `main`，分支和主工作树状态符合分支级隔离；工作树仅含根 `.comet/current-change.json`、本 change 的 OpenSpec 规划工件和 `paseo.json` 三类未跟踪项。
- Task 2 历史状态：本任务仅初始化规划证据，不执行业务 TDD、本地门禁或远程 integration；这些项由后续 OpenSpec task 真实执行。
- 首次本地 full 门禁（Task 3）历史失败：`make test` 退出码 `2`。后端默认 `go test ./...`、`golangci-lint run ./...`（`0 issues.`）及 `go test -tags=unit ./...` 均完成；前端 `pnpm --dir frontend run lint:check` 在启动前失败，错误为 `process_begin: CreateProcess(NULL, pnpm --dir frontend run lint:check, ...) failed.` 和 `make (e=2): 系统找不到指定的文件。`，未产生失败测试名。
- 首次失败前置条件恢复：执行 `corepack enable pnpm` 后，自检 `Get-Command pnpm` 解析到 `C:\Users\caiqy\.version-fox\sdks\nodejs\pnpm.ps1`，`pnpm --version` 输出 `11.17.0`；未改变仓库文件。
- 完整重跑本地 full 门禁：再次执行 `make test`，退出码 `2`。后端默认测试、`golangci-lint run ./...`（`0 issues.`）和 unit 测试均通过；前端 `pnpm --dir frontend run lint:check` 触发依赖状态检查，因无 TTY 拒绝清理 modules 目录，报 `ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY`，内部 `pnpm install` 退出码 `1`，最终 `make: *** [test-frontend] Error 1`。这是新的首个未解决阻塞，未产生失败测试名。
- 第二次失败前置条件恢复：执行 `corepack install --global pnpm@9.15.9` 后，自检 `pnpm --version` 输出 `9.15.9`，`frontend/node_modules/.modules.yaml` 的 `packageManager: pnpm@9.15.9` 与其一致；未删除 `node_modules`，未修改仓库或 lockfile。
- 再次完整重跑：`make test` 退出码 `0`。后端默认测试、`golangci-lint run ./...`（`0 issues.`）、unit 测试、前端 lint 与 typecheck 均通过；前端 Vitest 为 `194 passed` 测试文件、`1493 passed` 用例。随后 `make build` 退出码 `2`：backend build 的 `./scripts/resolve-version.sh` 调用无法创建 `sh D:\Caiqy\Projects\Github\sub2api\backend\scripts\resolve-version.sh` 进程，继而 `CGO_ENABLED` 未被识别，最终 `make: *** [build-backend] Error 2`。这是新的首个未解决阻塞。
- 第三次失败前置条件恢复：每条门禁 PowerShell 进程均在执行前设置 `$env:PATH = "C:\Program Files\Git\bin;C:\Program Files\Git\usr\bin;" + $env:PATH`。自检 `Get-Command sh` 为 `C:\Program Files\Git\bin\sh.exe`、`sh --version` 为 GNU bash `5.2.37`，且 `sh backend/scripts/resolve-version.sh backend/cmd/server/VERSION` 输出 `0.1.159.6`、退出码 `0`；未修改系统 PATH 或仓库。
- 阶段 0 本地 full 门禁最终完整重跑（当前 HEAD：`aca233e82c08778e221a049d99a69aa02febaf87`）：`make test` 退出码 `0`，后端默认测试、`golangci-lint run ./...`（`0 issues.`）、unit 测试、前端 lint/typecheck 均通过，Vitest 为 `194 passed` 测试文件、`1493 passed` 用例；`make build` 退出码 `0`，后端按 `0.1.159.6` 构建，前端 Vite 处理 `987` 个模块并完成 production build；两次 `make -C backend generate` 均退出 `0`，每次后的 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 均退出 `0`、无输出，生成稳定。
- 静态和范围检查均通过：`git diff --check` 退出 `0`（仅 CRLF 自动转换提示，无空白错误）；两次 `git diff --name-only --diff-filter=U` 均退出 `0`、未合并文件集合为空；真实冲突标记扫描退出 `0`、集合为空；`git diff --name-only 'v0.1.159^{}..HEAD'` 退出 `0` 并列出本分支历史变更文件；最终 `backend/cmd/server/VERSION` 为 `0.1.159.6`。最终 `git status --short` 仅显示协调者既有 plan、OpenSpec `tasks.md`、`.comet/subagent-progress.md` 改动、本任务 ledger，以及未跟踪 `.comet/current-change.json`、`paseo.json`；后两者未触碰。
- 远程 integration（Task 4，归档 HEAD：`c8e0110a9a2354453753db9c4acae0ed7570458d`）：BLOCKED。唯一远程目录为 `/tmp/sub2api-stage-0-4d6c9b00fce84b9bb7622e747dc12556`，以 `umask 077` 创建并已清理。`git archive --format=tar HEAD` 已成功上传；预检 `go version && make --version && docker info` 退出码 `1`，首个阻塞为 `make: command not found`，因此未执行当次 `docker info`、解包或 integration 命令。预检 stdout 同时显示远程 Go `1.26.1`，低于归档 `backend/go.mod` 的 `go 1.26.5`；即使安装 make，仍须先升级 Go。后续只读工具链发现确认 `/usr/bin/docker` 的 `docker info` 退出码 `0`、ServerVersion `29.2.1`，但 `make`、`gmake` 和要求的 Go 均未发现。授权配置前的完整依赖发现还确认 `powershell.exe` 和 `pwsh` 均不存在；`backend/Makefile` 的 `test-integration` 必须调用 `powershell.exe -NoProfile ... scripts/test.ps1`，且用户本次仅授权 Make/Go，故未执行任何配置。远程 `integration.log` 不存在，故无可下载日志；清理 JSON 为 `success=true, exit_code=0`，本地 tar 已删除。完整命令、JSON、范围自审见 `.superpowers/sdd/task-4-v0-1-165-report.md`。
- 规范澄清提交 `849f956992178e25ab2074e1e4cc596d29f8834f` 已将门禁改为 Linux 原生等价序列，明确不需要 Make 或 PowerShell；此前的 Make/PowerShell 缺失记录保留为历史。用户授权现有 vfox 安装并全局激活 `golang@1.26.5`，两个命令的 JSON 均为 `success=true, exit_code=0`；独立复核 `go version && docker info` 也为 `success=true, exit_code=0`，Go 和 directive 均为 `1.26.5`，Docker Server 为 `29.2.1`。
- 原生门禁重跑仍 BLOCKED：归档 HEAD 为 `849f956992178e25ab2074e1e4cc596d29f8834f`，新 nonce 为 `419b110da56e4b4294e48332dc4e6aec`，远程目录 `/tmp/sub2api-stage-0-419b110da56e4b4294e48332dc4e6aec` 仅以 `umask 077` 创建、解包和测试。预检 JSON 为 `success=true, exit_code=0`；重建 `backend/.test-tmp` 后执行 `CI=true GOFLAGS='-v' TMP='<remote>/src/backend/.test-tmp' TEMP='<remote>/src/backend/.test-tmp' go test -tags=integration ./...` 的 JSON 为 `success=false, exit_code=1`。日志已成功下载至 `C:/Users/caiqy/AppData/Local/Temp/sub2api-stage-0-419b110da56e4b4294e48332dc4e6aec-integration.log`，目标 `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 明确 PASS；但 `internal/handler` 与 `internal/repository` 有七个失败测试（详见 scratch report），整套 integration 未通过。新目录清理 JSON 为 `success=true, exit_code=0`，本地 tar 已删除且日志保留。
- Task 4 远程 GREEN（归档 HEAD：`99cb81de306cb0e8ea811387e362e4d601f6f4b0`）：规范 Gate、测试基线和 reviewer 修复已提交；新 nonce `35d92b78d17d4b558b808b72d043e9f5` 的唯一远程目录 `/tmp/sub2api-stage-0-35d92b78d17d4b558b808b72d043e9f5` 创建、上传、预检、六份日志下载与清理 JSON 均为 `success=true, exit_code=0`。Go `1.26.5` 满足 `backend/go.mod`，Docker Server `29.2.1`。五个聚焦命令及完整 `CI=true GOFLAGS='-v' TMPDIR='<remote>/src/backend/.test-tmp' TMP='<remote>/src/backend/.test-tmp' TEMP='<remote>/src/backend/.test-tmp' go test -tags=integration ./...` 都退出 `0`；完整日志显式 PASS `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate`（`4.91s`）。本地 tar 已删除，六份日志保留于 `C:/Users/caiqy/AppData/Local/Temp/sub2api-stage-0-35d92b78d17d4b558b808b72d043e9f5-*.log`。完整 skip 分类、命令和历史 RED 见 scratch report。
- 固定门禁 HEAD range 分类：重新执行 `git diff --name-only 'v0.1.159^{}..aca233e82c08778e221a049d99a69aa02febaf87'`，退出 `0`，共 `963` 个路径。非重叠分类为 `backend/` 393、`frontend/` 139、`docs/superpowers/` 49、其他 `docs/` 1、`openspec/` 345、`deploy/` 1、`memory/` 10、`.superpowers/` 12、`.comet/` 1、`.github/` 2、根目录 10；只包含该固定 HEAD 中从 `v0.1.159` 起的既有本地提交和 Task 1/2 规划证据，未纳入 Task 3 后续提交。
- range 排除核对：`paseo.json`、根 `.comet/current-change.json`、`bin/`、`dist/` 和 `node_modules/` 均为 `0`；唯一的 `backend/cmd/server/wire_gen.go` 是固定 HEAD 中已跟踪的历史生成源码，不是本门禁残留，且两次生成后的受检 diff 均为空。
- warning/noise 分类（完整原文见 `.superpowers/sdd/task-3-v0-1-165-{make-test,make-build,git-diff-check}.log`）：Vitest 的 `src/views/admin/__tests__/SettingsView.spec.ts` 输出 `[Vue warn]: Failed to resolve component: router-link`，网络负路径输出如 `src/stores/__tests__/subscriptions.spec.ts` 的 `Failed to fetch active subscriptions: Error: Network error` 和 `src/composables/__tests__/useTableLoader.spec.ts` 的 `Table load error: Error: Server error`，均为测试刻意覆盖的 stderr/console 输出；`SubscriptionPlanCard.spec.ts` 的 `[intlify] The message format compilation is not supported in this build` 为测试运行时 i18n advisory。`make test` 仍以 `194 passed` 文件、`1493 passed` 用例和退出 `0` 结束，故无吞错迹象。
- 工具 advisory：Browserslist 输出 `browsers data (caniuse-lite) is 7 months old`，Vite 输出 `Some chunks are larger than 500 kB after minification`（`AccountsView` 为 `670.83 kB`），分别是数据时效和 bundle size 建议；`make build` 退出 `0`。Git CRLF 提示为 `LF will be replaced by CRLF the next time Git touches it`，`git diff --check` 退出 `0`，故不是空白错误。

## 能力矩阵

| 能力 | 受影响 tag | 当前状态 | 证据 |
| --- | --- | --- | --- |
| changed-files 与本地能力交集 | `v0.1.160` 至 `v0.1.165` | `待执行` | Task 1.6 尚未运行六段实际 diff 与调用链审查。 |

## v0.1.160

- changed-files：待执行。
- 冲突台账：无（尚未合并）。
- 能力矩阵交集：待执行。
- 聚焦测试：待执行。
- 本地门禁：待执行。
- 远程门禁：待执行。
- 放行结论：待执行。

## v0.1.161

- changed-files：待执行。
- 冲突台账：无（尚未合并）。
- 能力矩阵交集：待执行。
- 聚焦测试：待执行。
- 本地门禁：待执行。
- 远程门禁：待执行。
- 放行结论：待执行。

## v0.1.162

- changed-files：待执行。
- 冲突台账：无（尚未合并）。
- 能力矩阵交集：待执行。
- 聚焦测试：待执行。
- 本地门禁：待执行。
- 远程门禁：待执行。
- 放行结论：待执行。

## v0.1.163

- changed-files：待执行。
- 冲突台账：无（尚未合并）。
- 能力矩阵交集：待执行。
- 聚焦测试：待执行。
- 本地门禁：待执行。
- 远程门禁：待执行。
- 放行结论：待执行。

## v0.1.164

- changed-files：待执行。
- 冲突台账：无（尚未合并）。
- 能力矩阵交集：待执行。
- 聚焦测试：待执行。
- 本地门禁：待执行。
- 远程门禁：待执行。
- 放行结论：待执行。

## v0.1.165

- changed-files：待执行。
- 冲突台账：无（尚未合并）。
- 能力矩阵交集：待执行。
- 聚焦测试：待执行。
- 本地门禁：待执行。
- 远程门禁：待执行。
- 放行结论：待执行。

## 远程 integration 记录

- 阶段 0：BLOCKED（Task 4）。归档来源为已提交 HEAD `c8e0110a9a2354453753db9c4acae0ed7570458d`，本地 tar `C:/Users/caiqy/AppData/Local/Temp/sub2api-stage-0-4d6c9b00fce84b9bb7622e747dc12556.tar` 成功生成、上传后删除。远程目录 `/tmp/sub2api-stage-0-4d6c9b00fce84b9bb7622e747dc12556` 创建和清理 JSON 均为 `success=true, exit_code=0`。预检 JSON 为 `success=false, exit_code=1`，stderr 是 `bash: line 1: make: command not found`；stdout 的 Go `1.26.1` 低于归档所需 `1.26.5`。随后只读发现 JSON 为 `success=true, exit_code=0`：Docker 在 `/usr/bin/docker`，`docker info` 退出 `0`、ServerVersion `29.2.1`；`make`、`gmake`、`/usr/bin/make`、`/usr/local/bin/make` 均不存在，且只发现 PATH 中 `/root/.vfox/sdks/golang/bin/go` 的 `go1.26.1`。授权配置前的 vfox/PowerShell 发现也为 `success=true, exit_code=0`：`vfox 1.0.6` 与 `dnf` 存在，但 `powershell.exe`、`pwsh` 均不存在。用户本次授权仅覆盖 Make 和 Go，因此未执行配置；必须先由用户或管理员提供 `powershell.exe`，或提供 `pwsh`（届时可在唯一任务目录创建临时 `powershell.exe` wrapper）。因预检未通过，未解包、未运行 `CI=true GOFLAGS='-v' make -C backend test-integration`，目标 `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 无 PASS，亦无 `--- SKIP:` 可分类。日志文件不存在，未生成或下载 `$env:TEMP/sub2api-stage-0-*-integration.log`。详见 `.superpowers/sdd/task-4-v0-1-165-report.md`。
- 阶段 0 原生门禁重跑（规范 HEAD `849f956992178e25ab2074e1e4cc596d29f8834f`）：vfox `golang@1.26.5` 的 `install --yes`、`use --global` 和 Go/Docker 复核 JSON 都是 `success=true, exit_code=0`；未安装 Make 或 PowerShell。新 tar 为 `C:/Users/caiqy/AppData/Local/Temp/sub2api-stage-0-419b110da56e4b4294e48332dc4e6aec.tar`，新远程目录为 `/tmp/sub2api-stage-0-419b110da56e4b4294e48332dc4e6aec`。创建、上传、预检、日志下载和清理 JSON 都是 `success=true, exit_code=0`；integration JSON 为 `success=false, exit_code=1`。`TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 在日志中 PASS，七项 `--- SKIP:` 已分类为 DingTalk sentinel、未配置 TLS capture URL、外部 TLS 服务证书时间错误的 JA3/两个 profile 子例、已知 CI concurrency TODO、未配置 `OPENAI_API_KEY`；它们不是本阶段的通过证据。失败项为 `TestOpenAIGatewayHandlerImages_MultipartSourceOpenFailureReturns503WithoutMarkingAccount`（期望 503、实际 502）、`TestOpenAIGatewayHandler_ChatAndEmbeddingsReplayMappedSpoolAcrossFailover/chat`（等待第二次 upstream 超时）、`TestRequestBodyCoordinator_CleanupRemovesRawEffectiveAndMultipartTemps`（未创建预期 multipart temp）、`TestUsageLogRepoSuite` 的三项 detail/transaction 断言，以及 `TestUserRepoSuite/TestHiddenUIResourcesRoundTrip`（资源 ID 顺序不符）。日志已下载保留，remote 目录和 local tar 已清理；阶段仍 BLOCKED。
- 阶段 0 原生门禁 GREEN（Task 4，HEAD `99cb81de306cb0e8ea811387e362e4d601f6f4b0`）：新 nonce `35d92b78d17d4b558b808b72d043e9f5`，tar `C:/Users/caiqy/AppData/Local/Temp/sub2api-stage-0-35d92b78d17d4b558b808b72d043e9f5.tar` 由 `git archive HEAD` 生成后上传并删除。预检 `go version && docker info` JSON 为 `success=true, exit_code=0`，远程 Go `1.26.5` 等于 directive，Docker Server `29.2.1`。multipart 503、coordinator cleanup、Chat/Embeddings replay、UsageLogRepoSuite 四项修复子测试、HiddenUIResources 子测试均以三变量 temp 环境在同一 archive 通过并分别下载日志；随后完整命令 JSON 为 `success=true, exit_code=0`。完整日志没有 `FAIL`，目标 migration test PASS（`4.91s`）。七项 skip：DingTalk Task 1.10 sentinel、未设 `TLSFINGERPRINT_CAPTURE_URL`、外部 `tls.peet.ws` 证书时间错误导致的 JA3 和两个 profile 子例、已知 CI concurrency TODO、未设 `OPENAI_API_KEY`；均非 migration/repository target，记录为基线风险但不阻塞本阶段。日志下载和清理 JSON 为 `success=true, exit_code=0`，远程目录已不存在，本地 tar 已删除，六份日志保留。
- Task 5 生成稳定性尝试（HEAD `6200a8105f2d8a389d10bd6e660ec8007df237c9`，`backend/cmd/server/VERSION` 为 `0.1.159.6`）：首轮 `make -C backend generate` 未通过。`go generate ./ent` 写入 `backend/ent/paymentproviderinstance.go` 时返回 `The requested operation cannot be performed on a file with a user-mapped section open.`，生成器退出 `1`，Make 失败。按 fail-fast 规则，未执行首轮生成 diff、第二轮生成/生成 diff、`git ls-tree` migration 清单和指定 `git grep`；因此没有生成稳定性或 migration 基线通过结论。失败过程留下 5 个未暂存 Ent 生成输出改动：`paymentproviderinstance.go`、`setting_query.go`、`usagecleanuptask_create.go`、`usersubscription_create.go`、`usersubscription_query.go`（`4068` 新增、`2555` 删除）；按约束原样保留，未手工修正、未暂存。完整证据见 `.superpowers/sdd/task-5-v0-1-165-report.md`。
- Task 5 最小重试：协调者确认无残留 `go`、`gopls`、`entc`、`wire` 进程且磁盘空间充足后，诊断再次确认上述 5 个文件均为普通 `Archive` 属性、相关进程为空。唯一重试的 `make -C backend generate` 不再报 user-mapped section，而在 `entc/load` 编译现存部分输出时失败：5 个文件分别有 `field redeclared`，`paymentproviderinstance.go` 另有未使用的 `context`、`errors`、`math`、`sync` 和 `predicate` 导入；生成器输出 `exit status 1`，Make 失败。未执行生成 diff、第二轮生成或 migration 静态命令，未作第三次尝试、未清理/格式化/恢复任何生成输出；Task 5 仍为 BLOCKED。
- Task 5 工作区第二次 user-mapped section 失败：在协调者仅恢复首次失败的 5 个已知 Ent 部分输出后，工作区首轮 `make -C backend generate` 与紧随的 generated-path diff 均退出 `0`；第二轮在写入 `backend/ent/usagelogdetail_query.go` 时再次报 `The requested operation cannot be performed on a file with a user-mapped section open.`。协调者恢复该失败留下的 3 个明确部分输出；生成路径随后再次为零 diff。该现象未证明 watcher 是根因。
- Task 5 隔离尝试审计：归档脚手架有两次生成前准备失败，均未触及工作区：第一次为 `New-Item -LiteralPath` 参数错误；第二次为 archive/extract 后、`git init` 前额外执行 `git show HEAD:...` 的顺序错误。唯一 nonce `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-task5-stage0-8f1d9e6b` 已确认删除。首次 detached worktree 目标因路径过长创建失败且无残留。最终以短路径 detached worktree `C:/Users/caiqy/AppData/Local/Temp/opencode/w84b854`，通过 `git -c core.longpaths=true worktree add --detach <path> HEAD` 创建：两轮各自 `make -C backend generate`、`git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 与 `git status --short` 均退出 `0`，两个 status 均为空。随后 `git -c core.longpaths=true worktree remove <path>` 成功；路径不存在且 `git worktree list --porcelain` 无该路径。
- Task 5 migration/runner 静态证据：主工作区 generated-path diff 退出 `0`。监督复核命令 `git ls-tree -r --name-only 98fbd1448ad38fbf45fb7334a0cd508576f0fd74 backend/migrations` 有 `238` 个路径，其中 `231` 个 `.sql` migration、`7` 个非 SQL 支持文件。HEAD 中 `172_video_per_second_billing_metadata.sql` 与 `181_group_duplicate_operation_id.sql` 各只有一个，且没有 `186*` 文件，故不存在预期的 172/181 同号不同名或两个 186。runner 以 `fs.Glob(fsys, "*.sql")` 后 `sort.Strings` 按完整文件名执行；`schema_migrations.filename TEXT PRIMARY KEY` 唯一标识 migration，内容先 `TrimSpace` 再算 SHA-256 hex checksum。checksum 不一致仅接受按 migration 名与数据库/文件 hash 双重匹配的受限兼容白名单（14 项），其余失败。`*_notx.sql` validator 用简单分号切分以及 `strings.ToUpper`/`strings.Contains` 作关键词限制，尝试要求幂等 `CREATE/DROP INDEX CONCURRENTLY` 并拒绝事务控制或混合普通语句；它不是 SQL parser，字符串字面量或注释中的关键词理论上可绕过该限制。reviewer 观察当前实际 `_notx.sql` 均为幂等并发索引语句。

### HEAD migration 完整路径清单（238 项）

```text
backend/migrations/001_init.sql
backend/migrations/002_account_type_migration.sql
backend/migrations/003_subscription.sql
backend/migrations/004_add_redeem_code_notes.sql
backend/migrations/005_schema_parity.sql
backend/migrations/006_add_users_allowed_groups_compat.sql
backend/migrations/006_fix_invalid_subscription_expires_at.sql
backend/migrations/006b_guard_users_allowed_groups.sql
backend/migrations/007_add_user_allowed_groups.sql
backend/migrations/008_seed_default_group.sql
backend/migrations/009_fix_usage_logs_cache_columns.sql
backend/migrations/010_add_usage_logs_aggregated_indexes.sql
backend/migrations/011_remove_duplicate_unique_indexes.sql
backend/migrations/012_add_user_subscription_soft_delete.sql
backend/migrations/013_log_orphan_allowed_groups.sql
backend/migrations/014_drop_legacy_allowed_groups.sql
backend/migrations/015_fix_settings_unique_constraint.sql
backend/migrations/016_soft_delete_partial_unique_indexes.sql
backend/migrations/018_user_attributes.sql
backend/migrations/019_migrate_wechat_to_attributes.sql
backend/migrations/020_add_temp_unschedulable.sql
backend/migrations/024_add_gemini_tier_id.sql
backend/migrations/026_ops_metrics_aggregation_tables.sql
backend/migrations/027_usage_billing_consistency.sql
backend/migrations/028_add_account_notes.sql
backend/migrations/028_add_usage_logs_user_agent.sql
backend/migrations/028_group_image_pricing.sql
backend/migrations/029_add_group_claude_code_restriction.sql
backend/migrations/029_usage_log_image_fields.sql
backend/migrations/030_add_account_expires_at.sql
backend/migrations/031_add_ip_address.sql
backend/migrations/032_add_api_key_ip_restriction.sql
backend/migrations/033_add_promo_codes.sql
backend/migrations/033_ops_monitoring_vnext.sql
backend/migrations/034_ops_upstream_error_events.sql
backend/migrations/034_usage_dashboard_aggregation_tables.sql
backend/migrations/035_usage_logs_partitioning.sql
backend/migrations/036_ops_error_logs_add_is_count_tokens.sql
backend/migrations/036_scheduler_outbox.sql
backend/migrations/037_add_account_rate_multiplier.sql
backend/migrations/037_ops_alert_silences.sql
backend/migrations/038_ops_errors_resolution_retry_results_and_standardize_classification.sql
backend/migrations/039_ops_job_heartbeats_add_last_result.sql
backend/migrations/040_add_group_model_routing.sql
backend/migrations/041_add_model_routing_enabled.sql
backend/migrations/042_add_usage_cleanup_tasks.sql
backend/migrations/042b_add_ops_system_metrics_switch_count.sql
backend/migrations/043_add_usage_cleanup_cancel_audit.sql
backend/migrations/043b_add_group_invalid_request_fallback.sql
backend/migrations/044_add_user_totp.sql
backend/migrations/044b_add_group_mcp_xml_inject.sql
backend/migrations/045_add_accounts_extra_index.sql
backend/migrations/045_add_announcements.sql
backend/migrations/045_add_api_key_quota.sql
backend/migrations/046_add_sora_accounts.sql
backend/migrations/046_add_usage_log_reasoning_effort.sql
backend/migrations/046b_add_group_supported_model_scopes.sql
backend/migrations/047_add_sora_pricing_and_media_type.sql
backend/migrations/047_add_user_group_rate_multipliers.sql
backend/migrations/048_add_error_passthrough_rules.sql
backend/migrations/049_unify_antigravity_model_mapping.sql
backend/migrations/050_map_opus46_to_opus45.sql
backend/migrations/051_migrate_opus45_to_opus46_thinking.sql
backend/migrations/052_add_group_sort_order.sql
backend/migrations/052_migrate_upstream_to_apikey.sql
backend/migrations/053_add_security_secrets.sql
backend/migrations/053_add_skip_monitoring_to_error_passthrough.sql
backend/migrations/054_drop_legacy_cache_columns.sql
backend/migrations/054_ops_system_logs.sql
backend/migrations/055_add_cache_ttl_overridden.sql
backend/migrations/056_add_api_key_last_used_at.sql
backend/migrations/057_add_idempotency_records.sql
backend/migrations/058_add_sonnet46_to_model_mapping.sql
backend/migrations/059_add_gemini31_pro_to_model_mapping.sql
backend/migrations/060_add_gemini31_flash_image_to_model_mapping.sql
backend/migrations/060_add_usage_log_openai_ws_mode.sql
backend/migrations/061_add_usage_log_request_type.sql
backend/migrations/062_add_scheduler_and_usage_composite_indexes_notx.sql
backend/migrations/063_add_sora_client_tables.sql
backend/migrations/064_add_api_key_rate_limits.sql
backend/migrations/065_add_search_trgm_indexes.sql
backend/migrations/066_add_scheduled_test_tables.sql
backend/migrations/067_add_account_load_factor.sql
backend/migrations/068_add_announcement_notify_mode.sql
backend/migrations/069_add_group_messages_dispatch.sql
backend/migrations/070_add_scheduled_test_auto_recover.sql
backend/migrations/070_add_usage_log_service_tier.sql
backend/migrations/071_add_gemini25_flash_image_to_model_mapping.sql
backend/migrations/071_add_usage_billing_dedup.sql
backend/migrations/072_add_usage_billing_dedup_created_at_brin_notx.sql
backend/migrations/073_add_usage_billing_dedup_archive.sql
backend/migrations/074_add_usage_log_endpoints.sql
backend/migrations/075_add_usage_log_upstream_model.sql
backend/migrations/075_map_haiku45_to_sonnet46.sql
backend/migrations/076_add_usage_log_upstream_model_index_notx.sql
backend/migrations/077_add_usage_log_details.sql
backend/migrations/077_add_usage_log_requested_model.sql
backend/migrations/078_add_usage_log_detail_upstream_request.sql
backend/migrations/078_add_usage_log_requested_model_index_notx.sql
backend/migrations/079_add_usage_log_detail_upstream_response.sql
backend/migrations/079_ops_error_logs_add_endpoint_fields.sql
backend/migrations/080_create_tls_fingerprint_profiles.sql
backend/migrations/081_add_group_account_filter.sql
backend/migrations/081_create_channels.sql
backend/migrations/082_refactor_channel_pricing.sql
backend/migrations/083_channel_model_mapping.sql
backend/migrations/084_channel_billing_model_source.sql
backend/migrations/085_channel_restrict_and_per_request_price.sql
backend/migrations/086_channel_platform_pricing.sql
backend/migrations/087_usage_log_billing_mode.sql
backend/migrations/088_channel_billing_model_source_channel_mapped.sql
backend/migrations/089_usage_log_image_output_tokens.sql
backend/migrations/090_drop_sora.sql
backend/migrations/091_add_group_messages_dispatch_model_config.sql
backend/migrations/092_payment_orders.sql
backend/migrations/093_payment_audit_logs.sql
backend/migrations/094_removed_payment_channels.sql
backend/migrations/095_channel_features.sql
backend/migrations/095_subscription_plans.sql
backend/migrations/096_payment_provider_instances.sql
backend/migrations/097_fix_settings_updated_at_default.sql
backend/migrations/098_migrate_purchase_subscription_to_custom_menu.sql
backend/migrations/099_fix_migrated_purchase_menu_label_icon.sql
backend/migrations/100_remove_easypay_from_enabled_payment_types.sql
backend/migrations/101_add_account_stats_pricing.sql
backend/migrations/101_add_balance_notify_fields.sql
backend/migrations/101_add_channel_features_config.sql
backend/migrations/101_add_payment_mode.sql
backend/migrations/102_add_balance_notify_threshold_type.sql
backend/migrations/102_add_out_trade_no_to_payment_orders.sql
backend/migrations/103_add_allow_user_refund.sql
backend/migrations/104_migrate_notify_emails_to_struct.sql
backend/migrations/105_migrate_websearch_emulation_to_tristate.sql
backend/migrations/106_add_account_stats_pricing_intervals.sql
backend/migrations/107_add_account_cost_to_dashboard_tables.sql
backend/migrations/108_add_group_user_concurrency.sql
backend/migrations/108_auth_identity_foundation_core.sql
backend/migrations/108a_widen_auth_identity_migration_report_type.sql
backend/migrations/109_auth_identity_compat_backfill.sql
backend/migrations/110_pending_auth_and_provider_default_grants.sql
backend/migrations/111_payment_routing_and_scheduler_flags.sql
backend/migrations/112_add_payment_order_provider_key_snapshot.sql
backend/migrations/113_normalize_legacy_wechat_provider_key.sql
backend/migrations/114_auth_identity_migration_report_resolution.sql
backend/migrations/115_auth_identity_legacy_external_backfill.sql
backend/migrations/116_auth_identity_legacy_external_safety_reports.sql
backend/migrations/117_add_payment_order_provider_snapshot.sql
backend/migrations/118_wechat_dual_mode_and_auth_source_defaults.sql
backend/migrations/119_enforce_payment_orders_out_trade_no_unique.sql
backend/migrations/120_enforce_payment_orders_out_trade_no_unique_notx.sql
backend/migrations/120a_align_payment_orders_out_trade_no_index_name.sql
backend/migrations/121_auth_identity_migration_report_type_widen.sql
backend/migrations/122_pending_auth_completion_token_cleanup.sql
backend/migrations/123_fix_legacy_auth_source_grant_on_signup_defaults.sql
backend/migrations/124_backfill_legacy_oidc_security_flags.sql
backend/migrations/125_add_channel_monitors.sql
backend/migrations/125_add_group_rpm_limit.sql
backend/migrations/126_add_channel_monitor_aggregation.sql
backend/migrations/126_add_user_rpm_limit.sql
backend/migrations/127_add_user_group_rpm_override.sql
backend/migrations/127_drop_channel_monitor_deleted_at.sql
backend/migrations/128_add_channel_monitor_request_templates.sql
backend/migrations/129_seed_claude_code_template.sql
backend/migrations/130_add_usage_log_detail_type.sql
backend/migrations/130_add_user_affiliates.sql
backend/migrations/131_affiliate_rebate_hardening.sql
backend/migrations/132_affiliate_custom_settings.sql
backend/migrations/133_affiliate_rebate_freeze.sql
backend/migrations/134_affiliate_ledger_audit_snapshots.sql
backend/migrations/134_image_generation_group_controls.sql
backend/migrations/135_allow_email_oauth_provider_types.sql
backend/migrations/135_content_moderation.sql
backend/migrations/136_add_dingtalk_provider_type.sql
backend/migrations/136_remove_ops_retry_replay.sql
backend/migrations/136_usage_log_image_size_metadata.sql
backend/migrations/137_redeem_code_expires_at.sql
backend/migrations/138_channel_monitor_openai_api_mode.sql
backend/migrations/139_seed_openai_monitor_templates.sql
backend/migrations/140_extend_user_provider_default_grants_check.sql
backend/migrations/141_subscription_expiry_notify_enabled.sql
backend/migrations/142_user_platform_quotas.sql
backend/migrations/143_group_models_list_config.sql
backend/migrations/144_add_opus48_to_model_mapping.sql
backend/migrations/145_deleted_api_key_audit.sql
backend/migrations/145_ops_metrics_ttft_sample_count.sql
backend/migrations/147_ops_error_log_api_key_prefix.sql
backend/migrations/148_add_ops_error_logs_user_time_index_notx.sql
backend/migrations/149_content_moderation_matched_keyword.sql
backend/migrations/149_proxy_expiry_fallback.sql
backend/migrations/150_account_group_scheduler_indexes_notx.sql
backend/migrations/151_account_autopause_expiry_index_notx.sql
backend/migrations/151_channel_monitor_jitter.sql
backend/migrations/152_scheduler_outbox_dedup_key.sql
backend/migrations/153_scheduler_outbox_pending_dedup_key_index_notx.sql
backend/migrations/154_account_spark_shadow.sql
backend/migrations/154_add_ops_system_logs_api_key_id.sql
backend/migrations/154a_account_spark_shadow_indexes_notx.sql
backend/migrations/155_add_ops_system_logs_api_key_id_index_notx.sql
backend/migrations/156_content_moderation_matched_keyword.sql
backend/migrations/157_user_platform_quotas_add_grok.sql
backend/migrations/158_add_group_peak_rate_multiplier.sql
backend/migrations/158_enable_grok_media_generation_groups.sql
backend/migrations/159_batch_image_foundation.sql
backend/migrations/159_user_resource_overrides.sql
backend/migrations/160_add_user_frozen_balance.sql
backend/migrations/160_batch_image_provider_refs.sql
backend/migrations/161_batch_image_pricing_snapshot.sql
backend/migrations/162_add_group_batch_image_generation_gate.sql
backend/migrations/163_batch_image_default_discount_and_hold_ratio.sql
backend/migrations/164_batch_image_download_and_user_delete.sql
backend/migrations/165_hide_pre_upstream_batch_image_failures.sql
backend/migrations/166_batch_image_task_name.sql
backend/migrations/167_clear_auto_batch_image_task_names.sql
backend/migrations/168_restore_empty_batch_image_task_names.sql
backend/migrations/169_batch_image_parent_batch.sql
backend/migrations/170_add_grok_video_pricing_controls.sql
backend/migrations/171_allow_video_usage_without_image_size.sql
backend/migrations/172_video_per_second_billing_metadata.sql
backend/migrations/173_allow_cyber_blocked_usage_request_type.sql
backend/migrations/174_add_usage_log_long_context_billing.sql
backend/migrations/174_add_usage_logs_api_key_latest_ip_index_notx.sql
backend/migrations/174_group_web_search_price_per_call.sql
backend/migrations/175_add_ops_system_logs_host.sql
backend/migrations/175_default_openai_long_context_billing.sql
backend/migrations/175a_add_ops_system_logs_host_index_notx.sql
backend/migrations/176_channel_monitor_grok_provider.sql
backend/migrations/177_add_subscription_plan_currency.sql
backend/migrations/178_channel_image_input_price.sql
backend/migrations/179_usage_log_image_input_tokens.sql
backend/migrations/180_audit_logs.sql
backend/migrations/181_group_duplicate_operation_id.sql
backend/migrations/README.md
backend/migrations/auth_identity_payment_migrations_regression_test.go
backend/migrations/channel_monitor_grok_provider_migration_test.go
backend/migrations/group_user_concurrency_migration_test.go
backend/migrations/latest_api_key_ip_index_test.go
backend/migrations/migrations.go
backend/migrations/openai_long_context_billing_migration_test.go
```
- Task 5 监督复核：`git diff-tree --no-commit-id --name-status -r 98fbd1448ad38fbf45fb7334a0cd508576f0fd74` 仅输出 `M docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`；`git diff --cached --exit-code`、generated-path diff 均退出 `0`。`git status --short` 仅余协调者 plan、OpenSpec tasks/progress 和受保护根 `.comet/current-change.json`、`paseo.json`；无 Task 5 源码、生成物或暂存项。`git worktree list --porcelain` 仅主工作区与两个既有 Task 27 detached worktree，且两个 `Test-Path`（旧 nonce 与 `w84b854`）均为 `False`。
- `v0.1.160`：待执行。
- `v0.1.161`：待执行。
- `v0.1.162`：待执行。
- `v0.1.163`：待执行。
- `v0.1.164`：待执行。
- `v0.1.165`：待执行。

## 阻塞与残余风险

- 历史阻塞（已解决，归档 HEAD `c8e0110a9a2354453753db9c4acae0ed7570458d`）：原 Make/PowerShell/Go 预检依次暴露 GNU Make 缺失、Go `1.26.1` 低于 directive、PowerShell 入口缺失；后续规范已改为 Linux 原生 gate，vfox Go `1.26.5` 和 Docker `29.2.1` 已在新 nonce 中验证。
- 历史阻塞（已解决，规范 HEAD `849f956992178e25ab2074e1e4cc596d29f8834f`）：首次原生 full RED 暴露 handler body spool/panic、repository usage-detail transaction/retention 与 hidden UI resource ordering 断言；后续测试修复及 HEAD `99cb81de306cb0e8ea811387e362e4d601f6f4b0` 的新 nonce focused/full GREEN 已解决。
- 当前状态：阶段 0 本地与远程门禁通过；七项非 migration/repository target skip 是已分类的基线风险。Task 5 为 `DONE_WITH_CONCERNS`：HEAD 生成内容经 detached worktree 双轮验证稳定，但受监视 Windows 工作区的随机文件映射风险仍未定因，隔离门禁与工作区门禁的等价性待 reviewer 判断。Task 6-7（能力矩阵与保护断言）也未完成，且六个 tag merge 尚未开始；仍不得开始、merge 或放行 `v0.1.160`。

## Task 6: 六段 changed-files 与本地能力矩阵

- Task 6 在首次 CodeGraph MCP/全局 CLI 不可用时曾 BLOCKED；协调者随后确认现有 `.codegraph/` 健康。此任务用官方一次性 CLI `npx -y @colbymchenry/codegraph` 恢复，先执行 `sync . --quiet`，再由 `status . --json` 确认 `initialized=true`、`pendingChanges` 全 0、`worktreeMismatch=null`。
- TDD 不适用：本任务仅映射现有变更、现有能力和现有测试，未改变行为。`gap` 是 Task 7 的输入，不阻塞 Task 6。

### v0.1.160

- Command: `git diff --name-only 'v0.1.159^{}..v0.1.160^{}'`
- Exact paths: 133
<!-- TASK6:v0.1.160:raw:begin -->
```text
README.md
assets/partners/logos/code0.jpg
backend/cmd/server/VERSION
backend/cmd/server/main.go
backend/cmd/server/wire.go
backend/cmd/server/wire_gen.go
backend/cmd/server/wire_gen_test.go
backend/internal/handler/batch_image_handler.go
backend/internal/handler/content_moderation_helper.go
backend/internal/handler/gateway_handler.go
backend/internal/handler/gateway_handler_chat_completions.go
backend/internal/handler/gateway_handler_responses.go
backend/internal/handler/gemini_v1beta_handler.go
backend/internal/handler/grok_media.go
backend/internal/handler/grok_media_test.go
backend/internal/handler/handler.go
backend/internal/handler/image_task_handler.go
backend/internal/handler/openai_alpha_search.go
backend/internal/handler/openai_chat_completions.go
backend/internal/handler/openai_embeddings.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/openai_grok_image_intent_gate_test.go
backend/internal/handler/openai_images.go
backend/internal/handler/security_audit_errors.go
backend/internal/handler/security_audit_errors_test.go
backend/internal/handler/security_audit_helper.go
backend/internal/handler/security_audit_helper_test.go
backend/internal/handler/security_audit_media_submit_test.go
backend/internal/handler/security_audit_order_test.go
backend/internal/handler/wire.go
backend/internal/pkg/xai/billing.go
backend/internal/repository/scheduler_cache.go
backend/internal/repository/scheduler_cache_unit_test.go
backend/internal/securityaudit/coordinator.go
backend/internal/securityaudit/coordinator_legacy.go
backend/internal/securityaudit/coordinator_test.go
backend/internal/securityaudit/prompt_config.go
backend/internal/securityaudit/prompt_config_integration_test.go
backend/internal/securityaudit/prompt_config_store.go
backend/internal/securityaudit/prompt_config_test.go
backend/internal/securityaudit/prompt_enqueue.go
backend/internal/securityaudit/prompt_event_repository.go
backend/internal/securityaudit/prompt_guard.go
backend/internal/securityaudit/prompt_guard_test.go
backend/internal/securityaudit/prompt_handler.go
backend/internal/securityaudit/prompt_handler_test.go
backend/internal/securityaudit/prompt_issue_summary.go
backend/internal/securityaudit/prompt_logging.go
backend/internal/securityaudit/prompt_logging_test.go
backend/internal/securityaudit/prompt_metrics.go
backend/internal/securityaudit/prompt_metrics_test.go
backend/internal/securityaudit/prompt_module.go
backend/internal/securityaudit/prompt_outbound_security.go
backend/internal/securityaudit/prompt_outbound_security_test.go
backend/internal/securityaudit/prompt_payload_store.go
backend/internal/securityaudit/prompt_payload_store_integration_test.go
backend/internal/securityaudit/prompt_qwen3guard.go
backend/internal/securityaudit/prompt_qwen3guard_test.go
backend/internal/securityaudit/prompt_repository.go
backend/internal/securityaudit/prompt_repository_integration_test.go
backend/internal/securityaudit/prompt_repository_test.go
backend/internal/securityaudit/prompt_scanner.go
backend/internal/securityaudit/prompt_service.go
backend/internal/securityaudit/prompt_service_test.go
backend/internal/securityaudit/prompt_snapshot.go
backend/internal/securityaudit/prompt_snapshot_test.go
backend/internal/securityaudit/prompt_types.go
backend/internal/securityaudit/prompt_worker.go
backend/internal/securityaudit/prompt_worker_test.go
backend/internal/server/middleware/audit_log.go
backend/internal/server/middleware/audit_log_test.go
backend/internal/server/routes/admin.go
backend/internal/server/routes/prompt_audit_route_coverage_test.go
backend/internal/service/account.go
backend/internal/service/account_grok_media_eligibility_test.go
backend/internal/service/admin_account.go
backend/internal/service/admin_account_upstream_billing_probe_test.go
backend/internal/service/grok_media.go
backend/internal/service/grok_quota_service.go
backend/internal/service/grok_quota_service_test.go
backend/internal/service/image_generation_intent.go
backend/internal/service/image_generation_intent_explicit_test.go
backend/internal/service/openai_account_scheduler_test.go
backend/internal/service/openai_gateway_grok_test.go
backend/internal/service/openai_gateway_scheduling.go
backend/migrations/181_prompt_audit.sql
backend/migrations/182_prompt_audit_full_prompt.sql
deploy/build_image.sh
deploy/docker-compose.yml
frontend/package.json
frontend/pnpm-lock.yaml
frontend/src/components/layout/AppSidebar.vue
frontend/src/features/prompt-audit/PromptAuditView.vue
frontend/src/features/prompt-audit/__tests__/PromptAuditView.spec.ts
frontend/src/features/prompt-audit/__tests__/api.spec.ts
frontend/src/features/prompt-audit/__tests__/components.spec.ts
frontend/src/features/prompt-audit/__tests__/integrationSurface.spec.ts
frontend/src/features/prompt-audit/__tests__/viewModel.spec.ts
frontend/src/features/prompt-audit/api.ts
frontend/src/features/prompt-audit/components/EndpointPool.vue
frontend/src/features/prompt-audit/components/EventDetailDialog.vue
frontend/src/features/prompt-audit/components/EventWorkspace.vue
frontend/src/features/prompt-audit/components/FilterDeleteDialog.vue
frontend/src/features/prompt-audit/components/PolicyPanel.vue
frontend/src/features/prompt-audit/components/RuntimeOverview.vue
frontend/src/features/prompt-audit/types.ts
frontend/src/features/prompt-audit/viewModel.ts
frontend/src/i18n/locales/en/admin/index.ts
frontend/src/i18n/locales/en/admin/promptAudit.ts
frontend/src/i18n/locales/en/common.ts
frontend/src/i18n/locales/zh/admin/index.ts
frontend/src/i18n/locales/zh/admin/promptAudit.ts
frontend/src/i18n/locales/zh/common.ts
frontend/src/router/index.ts
frontend/src/style.css
frontend/src/views/admin/BackupView.vue
openspec/changes/add-openai-compatible-prompt-audit/.openspec.yaml
openspec/changes/add-openai-compatible-prompt-audit/README.md
openspec/changes/add-openai-compatible-prompt-audit/design.md
openspec/changes/add-openai-compatible-prompt-audit/implementation-evidence.md
openspec/changes/add-openai-compatible-prompt-audit/implementation-guide.md
openspec/changes/add-openai-compatible-prompt-audit/proposal.md
openspec/changes/add-openai-compatible-prompt-audit/source-baseline.md
openspec/changes/add-openai-compatible-prompt-audit/source-feature-map.md
openspec/changes/add-openai-compatible-prompt-audit/source-freeze/MANIFEST.md
openspec/changes/add-openai-compatible-prompt-audit/source-freeze/aicodex-prompt-audit-tracked.patch
openspec/changes/add-openai-compatible-prompt-audit/source-freeze/aicodex-prompt-audit-untracked.tar.gz
openspec/changes/add-openai-compatible-prompt-audit/specs/prompt-input-audit/spec.md
openspec/changes/add-openai-compatible-prompt-audit/specs/prompt-input-guard/spec.md
openspec/changes/add-openai-compatible-prompt-audit/specs/security-audit-console/spec.md
openspec/changes/add-openai-compatible-prompt-audit/tasks.md
openspec/changes/add-openai-compatible-prompt-audit/verification.md
openspec/config.yaml
```
<!-- TASK6:v0.1.160:raw:end -->
- Local key-file intersection (matrix-derived exact local core paths):
```text
backend/cmd/server/wire_gen.go
backend/internal/handler/batch_image_handler.go
backend/internal/handler/content_moderation_helper.go
backend/internal/handler/openai_images.go
backend/internal/pkg/xai/billing.go
backend/internal/repository/scheduler_cache.go
backend/internal/service/admin_account_upstream_billing_probe_test.go
backend/internal/service/grok_media.go
backend/internal/service/openai_gateway_scheduling.go
backend/migrations/181_prompt_audit.sql
frontend/package.json
frontend/src/i18n/locales/en/common.ts
```

### v0.1.161

- Command: `git diff --name-only 'v0.1.160^{}..v0.1.161^{}'`
- Exact paths: 257
<!-- TASK6:v0.1.161:raw:begin -->
```text
Dockerfile
README.md
backend/cmd/cleanup-ingress-reject-logs/README.md
backend/cmd/cleanup-ingress-reject-logs/main.go
backend/cmd/cleanup-ingress-reject-logs/main_test.go
backend/cmd/server/VERSION
backend/cmd/server/wire.go
backend/cmd/server/wire_gen.go
backend/cmd/server/wire_gen_test.go
backend/internal/config/config.go
backend/internal/config/config_test.go
backend/internal/handler/admin/account_handler.go
backend/internal/handler/admin/account_handler_mixed_channel_test.go
backend/internal/handler/admin/admin_basic_handlers_test.go
backend/internal/handler/admin/admin_service_stub_test.go
backend/internal/handler/admin/grok_import_probe.go
backend/internal/handler/admin/grok_import_probe_test.go
backend/internal/handler/admin/grok_oauth_handler.go
backend/internal/handler/admin/grok_oauth_handler_test.go
backend/internal/handler/admin/ops_auth_cache_health_handler.go
backend/internal/handler/admin/ops_ingress_reject_handler.go
backend/internal/handler/admin/ops_ingress_reject_handler_test.go
backend/internal/handler/admin/setting_handler.go
backend/internal/handler/admin/setting_handler_audit.go
backend/internal/handler/admin/setting_handler_stepup_switch_test.go
backend/internal/handler/admin/setting_handler_update.go
backend/internal/handler/admin/user_handler.go
backend/internal/handler/admin/user_handler_activity_test.go
backend/internal/handler/admin/user_handler_batch_limits_test.go
backend/internal/handler/admin/user_handler_get_deleted_test.go
backend/internal/handler/admin/user_handler_list_apikey_group_test.go
backend/internal/handler/admin/user_handler_role_stepup_test.go
backend/internal/handler/dto/settings.go
backend/internal/handler/gateway_handler.go
backend/internal/handler/grok_media.go
backend/internal/handler/grok_media_test.go
backend/internal/handler/no_account_error.go
backend/internal/handler/no_account_error_test.go
backend/internal/handler/openai_chat_completions.go
backend/internal/handler/openai_gateway_count_tokens.go
backend/internal/handler/openai_gateway_credential_failover_loop_test.go
backend/internal/handler/openai_gateway_credential_failover_test.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/openai_gateway_handler_test.go
backend/internal/handler/ops_error_logger.go
backend/internal/handler/ops_error_logger_attribution_test.go
backend/internal/handler/ops_error_logger_test.go
backend/internal/handler/ops_ingress_reject_capture_test.go
backend/internal/handler/wire.go
backend/internal/pkg/apicompat/anthropic_to_responses_response.go
backend/internal/pkg/apicompat/anthropic_to_responses_stream_test.go
backend/internal/pkg/ip/ip.go
backend/internal/pkg/ip/ip_test.go
backend/internal/pkg/logger/logger.go
backend/internal/repository/account_repo.go
backend/internal/repository/account_repo_model_availability_test.go
backend/internal/repository/account_repo_sort_integration_test.go
backend/internal/repository/account_repo_upstream_billing_probe_update_test.go
backend/internal/repository/api_key_cache.go
backend/internal/repository/api_key_cache_subscriber_test.go
backend/internal/repository/api_key_repo.go
backend/internal/repository/api_key_repo_integration_test.go
backend/internal/repository/auth_cache_invalidation_outbox_integration_test.go
backend/internal/repository/auth_cache_invalidation_outbox_repo.go
backend/internal/repository/auth_cache_invalidation_outbox_repo_test.go
backend/internal/repository/http_upstream.go
backend/internal/repository/http_upstream_test.go
backend/internal/repository/migrations_runner.go
backend/internal/repository/migrations_runner_notx_test.go
backend/internal/repository/migrations_schema_integration_test.go
backend/internal/repository/ops_error_where_test.go
backend/internal/repository/ops_ingress_reject_repo.go
backend/internal/repository/ops_ingress_reject_repo_test.go
backend/internal/repository/ops_repo.go
backend/internal/repository/ops_repo_args_test.go
backend/internal/repository/ops_repo_get_error_log_by_id_integration_test.go
backend/internal/repository/ops_repo_lookup_deleted_key_audit_integration_test.go
backend/internal/repository/user_repo.go
backend/internal/repository/user_repo_delete_atomicity_integration_test.go
backend/internal/repository/wire.go
backend/internal/securityaudit/prompt_module.go
backend/internal/server/api_contract_test.go
backend/internal/server/http.go
backend/internal/server/http_ingress_test.go
backend/internal/server/middleware/api_key_auth.go
backend/internal/server/middleware/api_key_auth_google.go
backend/internal/server/middleware/api_key_auth_google_test.go
backend/internal/server/middleware/api_key_auth_test.go
backend/internal/server/middleware/client_request_id.go
backend/internal/server/middleware/client_request_id_test.go
backend/internal/server/middleware/ingress_reject.go
backend/internal/server/middleware/ingress_reject_access_sampler.go
backend/internal/server/middleware/ingress_reject_access_sampler_test.go
backend/internal/server/middleware/ingress_reject_test.go
backend/internal/server/middleware/invalid_auth_abuse_test.go
backend/internal/server/middleware/logger.go
backend/internal/server/middleware/middleware.go
backend/internal/server/middleware/request_access_logger_test.go
backend/internal/server/middleware/request_logger.go
backend/internal/server/middleware/request_metadata.go
backend/internal/server/middleware/session_binding.go
backend/internal/server/middleware/session_binding_test.go
backend/internal/server/middleware/step_up.go
backend/internal/server/middleware/step_up_test.go
backend/internal/server/router.go
backend/internal/server/routes/admin.go
backend/internal/server/routes/gateway.go
backend/internal/server/routes/gateway_test.go
backend/internal/server/routes/ops_ingress_reject_routes_test.go
backend/internal/service/account.go
backend/internal/service/account_grok_media_eligibility_test.go
backend/internal/service/account_service.go
backend/internal/service/account_service_delete_test.go
backend/internal/service/account_test_service.go
backend/internal/service/account_test_service_grok_test.go
backend/internal/service/admin_account.go
backend/internal/service/admin_account_upstream_billing_probe_test.go
backend/internal/service/admin_service.go
backend/internal/service/antigravity_gateway_service.go
backend/internal/service/antigravity_subscription_service.go
backend/internal/service/antigravity_subscription_test.go
backend/internal/service/api_key_auth_cache_impl.go
backend/internal/service/api_key_service.go
backend/internal/service/api_key_service_cache_test.go
backend/internal/service/auth_cache_invalidation_outbox.go
backend/internal/service/auth_cache_invalidation_outbox_test.go
backend/internal/service/channel_monitor_checker.go
backend/internal/service/channel_monitor_checker_body_test.go
backend/internal/service/domain_constants.go
backend/internal/service/error_policy_test.go
backend/internal/service/gateway_model_availability.go
backend/internal/service/gateway_model_availability_test.go
backend/internal/service/gateway_multiplatform_test.go
backend/internal/service/gateway_non_streaming_response_test.go
backend/internal/service/gateway_request.go
backend/internal/service/gateway_request_test.go
backend/internal/service/gemini_chat_completions_compat_service.go
backend/internal/service/gemini_error_policy_test.go
backend/internal/service/gemini_messages_compat_service.go
backend/internal/service/gemini_multiplatform_test.go
backend/internal/service/grok_media.go
backend/internal/service/grok_media_content_test.go
backend/internal/service/grok_quota_fetcher.go
backend/internal/service/grok_quota_fetcher_test.go
backend/internal/service/grok_quota_service.go
backend/internal/service/grok_quota_service_test.go
backend/internal/service/grok_upstream_url.go
backend/internal/service/grok_upstream_url_test.go
backend/internal/service/invalid_auth_abuse_limiter.go
backend/internal/service/invalid_auth_abuse_limiter_test.go
backend/internal/service/openai_account_runtime_block_fastpath.go
backend/internal/service/openai_account_runtime_block_fastpath_test.go
backend/internal/service/openai_alpha_search.go
backend/internal/service/openai_embeddings.go
backend/internal/service/openai_gateway_cc_pipeline.go
backend/internal/service/openai_gateway_chat_completions.go
backend/internal/service/openai_gateway_chat_completions_test.go
backend/internal/service/openai_gateway_forward.go
backend/internal/service/openai_gateway_grok.go
backend/internal/service/openai_gateway_grok_cache.go
backend/internal/service/openai_gateway_grok_cache_test.go
backend/internal/service/openai_gateway_grok_cache_tool_test.go
backend/internal/service/openai_gateway_grok_test.go
backend/internal/service/openai_gateway_model_availability.go
backend/internal/service/openai_gateway_passthrough.go
backend/internal/service/openai_gateway_upstream_errors.go
backend/internal/service/openai_images.go
backend/internal/service/openai_images_responses.go
backend/internal/service/openai_stream_read_error.go
backend/internal/service/openai_ws_client_read.go
backend/internal/service/openai_ws_v2/passthrough_relay.go
backend/internal/service/openai_ws_v2/passthrough_relay_internal_test.go
backend/internal/service/openai_ws_v2/passthrough_relay_test.go
backend/internal/service/openai_ws_v2_passthrough_adapter.go
backend/internal/service/openai_ws_v2_passthrough_lifecycle_test.go
backend/internal/service/ops_cleanup_executor.go
backend/internal/service/ops_cleanup_service.go
backend/internal/service/ops_ingress_reject.go
backend/internal/service/ops_ingress_reject_test.go
backend/internal/service/ops_log_runtime_test.go
backend/internal/service/ops_models.go
backend/internal/service/ops_port.go
backend/internal/service/ops_queue_sanitize_test.go
backend/internal/service/ops_repo_mock_test.go
backend/internal/service/ops_runtime_snapshot_test.go
backend/internal/service/ops_service.go
backend/internal/service/ops_service_user_error_test.go
backend/internal/service/ops_settings.go
backend/internal/service/ops_settings_advanced_test.go
backend/internal/service/ops_settings_models.go
backend/internal/service/ops_system_log_sink.go
backend/internal/service/ops_system_log_sink_test.go
backend/internal/service/ops_upstream_context.go
backend/internal/service/ratelimit_service.go
backend/internal/service/ratelimit_service_model_not_found_test.go
backend/internal/service/ratelimit_session_window_test.go
backend/internal/service/scheduler_snapshot_batch_query_test.go
backend/internal/service/scheduler_snapshot_full_rebuild_lifecycle_test.go
backend/internal/service/setting_features.go
backend/internal/service/setting_parse.go
backend/internal/service/setting_update.go
backend/internal/service/settings_view.go
backend/internal/service/subscription_assign_idempotency_test.go
backend/internal/service/subscription_service.go
backend/internal/service/upstream_billing_probe.go
backend/internal/service/upstream_billing_probe_test.go
backend/internal/service/user_subscription_daily_quota_test.go
backend/internal/service/wire.go
backend/internal/web/embed_on.go
backend/internal/web/embed_test.go
backend/migrations/183_ops_ingress_reject_aggregates.sql
backend/migrations/184_auth_cache_invalidation_outbox.sql
backend/scripts/finalize-ingress-reject-cleanup.sql
deploy/Caddyfile
deploy/EDGE_SECURITY.md
deploy/README.md
deploy/config.example.yaml
deploy/docker-compose.yml
frontend/src/App.vue
frontend/src/api/admin/ops.ts
frontend/src/api/admin/settings.ts
frontend/src/components/account/AccountUsageCell.vue
frontend/src/components/account/BulkEditAccountModal.vue
frontend/src/components/account/CreateAccountModal.vue
frontend/src/components/account/UpstreamBillingRateCell.vue
frontend/src/components/account/__tests__/AccountUsageCell.spec.ts
frontend/src/components/account/__tests__/BulkEditAccountModal.spec.ts
frontend/src/components/account/__tests__/CreateAccountModal.spec.ts
frontend/src/components/account/__tests__/UpstreamBillingRateCell.spec.ts
frontend/src/components/common/DataTable.vue
frontend/src/components/common/__tests__/DataTable.spec.ts
frontend/src/i18n/__tests__/wsModeLocaleDesc.spec.ts
frontend/src/i18n/locales/en/admin/accounts.ts
frontend/src/i18n/locales/en/admin/ops.ts
frontend/src/i18n/locales/en/admin/settings.ts
frontend/src/i18n/locales/en/misc.ts
frontend/src/i18n/locales/zh/admin/accounts.ts
frontend/src/i18n/locales/zh/admin/ops.ts
frontend/src/i18n/locales/zh/admin/settings.ts
frontend/src/i18n/locales/zh/misc.ts
frontend/src/main.ts
frontend/src/types/index.ts
frontend/src/utils/__tests__/branding.spec.ts
frontend/src/utils/branding.ts
frontend/src/views/admin/AccountsView.vue
frontend/src/views/admin/BackupView.vue
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts
frontend/src/views/admin/__tests__/AccountsView.usageWindowsHint.spec.ts
frontend/src/views/admin/__tests__/SettingsView.spec.ts
frontend/src/views/admin/ops/components/OpsErrorDetailModal.vue
frontend/src/views/admin/ops/components/OpsErrorLogTable.vue
frontend/src/views/admin/ops/components/OpsSettingsDialog.vue
frontend/src/views/admin/orders/AdminPaymentPlansView.vue
frontend/src/views/admin/orders/PlanEditDialog.vue
frontend/src/views/public/LegalDocumentView.vue
frontend/vite.config.ts
```
<!-- TASK6:v0.1.161:raw:end -->
- Local key-file intersection (matrix-derived exact local core paths):
```text
backend/cmd/server/wire_gen.go
backend/internal/handler/admin/user_handler_batch_limits_test.go
backend/internal/handler/admin/user_handler_list_apikey_group_test.go
backend/internal/repository/user_repo.go
backend/internal/server/middleware/session_binding.go
backend/internal/server/middleware/step_up.go
backend/internal/service/channel_monitor_checker_body_test.go
backend/internal/service/openai_gateway_grok_cache.go
backend/internal/service/openai_gateway_passthrough.go
backend/internal/service/openai_images_responses.go
backend/internal/service/openai_images.go
backend/internal/service/ops_service_user_error_test.go
backend/internal/service/scheduler_snapshot_batch_query_test.go
backend/internal/service/scheduler_snapshot_full_rebuild_lifecycle_test.go
backend/internal/service/setting_parse.go
backend/internal/service/setting_update.go
backend/internal/service/subscription_service.go
backend/internal/service/upstream_billing_probe.go
backend/internal/service/user_subscription_daily_quota_test.go
backend/migrations/183_ops_ingress_reject_aggregates.sql
frontend/src/i18n/locales/en/admin/settings.ts
frontend/vite.config.ts
```

### v0.1.162

- Command: `git diff --name-only 'v0.1.161^{}..v0.1.162^{}'`
- Exact paths: 190
<!-- TASK6:v0.1.162:raw:begin -->
```text
README.md
README_CN.md
README_JA.md
assets/logo.svg
backend/cmd/server/VERSION
backend/cmd/server/main.go
backend/cmd/server/wire_gen.go
backend/internal/config/config.go
backend/internal/config/config_test.go
backend/internal/config/env_reachability_test.go
backend/internal/config/image_storage_env_test.go
backend/internal/handler/admin/account_codex_agent_identity_import_test.go
backend/internal/handler/admin/account_codex_import.go
backend/internal/handler/admin/backup_handler.go
backend/internal/handler/admin/setting_handler.go
backend/internal/handler/admin/setting_handler_audit.go
backend/internal/handler/admin/setting_handler_stepup_switch_test.go
backend/internal/handler/admin/setting_handler_update.go
backend/internal/handler/admin/system_handler.go
backend/internal/handler/admin/system_handler_test.go
backend/internal/handler/api_key_handler.go
backend/internal/handler/dto/settings.go
backend/internal/handler/failover_loop.go
backend/internal/handler/failover_loop_test.go
backend/internal/handler/image_task_admin_toggle_test.go
backend/internal/handler/image_task_handler.go
backend/internal/handler/openai_codex_models_handler.go
backend/internal/handler/openai_gateway_count_tokens.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/openai_gateway_handler_test.go
backend/internal/handler/openai_responses_image_intent_benchmark_test.go
backend/internal/pkg/apicompat/anthropic_responses_test.go
backend/internal/pkg/apicompat/anthropic_to_responses.go
backend/internal/pkg/apicompat/anthropic_to_responses_response.go
backend/internal/pkg/apicompat/chatcompletions_anthropic_bridge.go
backend/internal/pkg/apicompat/chatcompletions_anthropic_bridge_test.go
backend/internal/pkg/apicompat/responses_anthropic_cache_creation_test.go
backend/internal/pkg/apicompat/responses_to_anthropic.go
backend/internal/pkg/apicompat/types.go
backend/internal/pkg/ip/ip.go
backend/internal/pkg/ip/ip_test.go
backend/internal/pkg/xai/quota.go
backend/internal/pkg/xai/quota_test.go
backend/internal/repository/github_release_service.go
backend/internal/repository/github_release_service_test.go
backend/internal/repository/wire.go
backend/internal/securityaudit/prompt_config.go
backend/internal/securityaudit/prompt_config_store.go
backend/internal/securityaudit/prompt_config_test.go
backend/internal/server/api_contract_test.go
backend/internal/server/http.go
backend/internal/server/http_ingress_test.go
backend/internal/server/middleware/api_key_auth.go
backend/internal/server/middleware/api_key_auth_test.go
backend/internal/server/middleware/middleware.go
backend/internal/server/middleware/session_binding.go
backend/internal/server/middleware/session_binding_test.go
backend/internal/server/router.go
backend/internal/server/routes/admin.go
backend/internal/server/routes/gateway.go
backend/internal/server/routes/gateway_test.go
backend/internal/service/account_test_service.go
backend/internal/service/account_usage_service.go
backend/internal/service/admin_service.go
backend/internal/service/admin_service_proxy_quality_test.go
backend/internal/service/api_key_service.go
backend/internal/service/backup_service.go
backend/internal/service/backup_service_test.go
backend/internal/service/domain_constants.go
backend/internal/service/gateway_forward_as_chat_completions.go
backend/internal/service/gateway_forward_as_responses.go
backend/internal/service/gateway_service.go
backend/internal/service/gemini_chat_completions_compat_service.go
backend/internal/service/grok_media.go
backend/internal/service/grok_media_content_test.go
backend/internal/service/grok_quota_fetcher.go
backend/internal/service/grok_quota_fetcher_test.go
backend/internal/service/grok_quota_service.go
backend/internal/service/grok_quota_service_test.go
backend/internal/service/grok_token_provider.go
backend/internal/service/grok_token_provider_test.go
backend/internal/service/image_storage_settings.go
backend/internal/service/image_storage_settings_test.go
backend/internal/service/image_task.go
backend/internal/service/notification_email_service.go
backend/internal/service/notification_email_service_test.go
backend/internal/service/openai_codex_models_service.go
backend/internal/service/openai_codex_models_service_test.go
backend/internal/service/openai_codex_transform.go
backend/internal/service/openai_codex_transform_test.go
backend/internal/service/openai_compat_model_test.go
backend/internal/service/openai_gateway_chat_completions.go
backend/internal/service/openai_gateway_chat_completions_test.go
backend/internal/service/openai_gateway_count_tokens.go
backend/internal/service/openai_gateway_count_tokens_test.go
backend/internal/service/openai_gateway_grok.go
backend/internal/service/openai_gateway_grok_cache.go
backend/internal/service/openai_gateway_grok_cache_test.go
backend/internal/service/openai_gateway_grok_chat_bridge.go
backend/internal/service/openai_gateway_grok_chat_bridge_test.go
backend/internal/service/openai_gateway_grok_test.go
backend/internal/service/openai_gateway_messages.go
backend/internal/service/openai_gateway_response_flush_test.go
backend/internal/service/openai_gateway_response_handling.go
backend/internal/service/openai_gateway_response_handling_type_test.go
backend/internal/service/openai_ws_http_bridge.go
backend/internal/service/openai_ws_http_bridge_test.go
backend/internal/service/ops_scheduled_report_service.go
backend/internal/service/ops_scheduled_report_service_test.go
backend/internal/service/setting_parse.go
backend/internal/service/setting_service.go
backend/internal/service/setting_service_update_test.go
backend/internal/service/setting_update.go
backend/internal/service/settings_view.go
backend/internal/service/subscription_calculate_progress_test.go
backend/internal/service/user_subscription.go
backend/internal/service/user_subscription_days_remaining_test.go
backend/internal/service/wire.go
deploy/.env.example
deploy/EDGE_SECURITY.md
deploy/README.md
deploy/config.example.yaml
deploy/docker-compose.dev.yml
deploy/docker-compose.local.yml
deploy/docker-compose.standalone.yml
deploy/docker-compose.yml
deploy/install.sh
deploy/tests/install-github-token-test.sh
docs/ASYNC_IMAGE_TASKS.md
frontend/index.html
frontend/public/logo.png
frontend/public/logo.svg
frontend/src/api/admin/backup.ts
frontend/src/api/admin/settings.ts
frontend/src/api/admin/system.ts
frontend/src/components/account/AccountUsageCell.vue
frontend/src/components/account/EditAccountModal.vue
frontend/src/components/account/OpenAIQuotaResetCell.vue
frontend/src/components/account/UpstreamBillingRateCell.vue
frontend/src/components/account/__tests__/AccountUsageCell.spec.ts
frontend/src/components/account/__tests__/EditAccountModal.grokUpstream.spec.ts
frontend/src/components/admin/usage/UsageFilters.vue
frontend/src/components/admin/user/UserBalanceModal.vue
frontend/src/components/channels/AvailableChannelsTable.vue
frontend/src/components/channels/__tests__/AvailableChannelsTable.spec.ts
frontend/src/components/charts/EndpointDistributionChart.vue
frontend/src/components/charts/GroupDistributionChart.vue
frontend/src/components/charts/ModelDistributionChart.vue
frontend/src/components/charts/UserBreakdownSubTable.vue
frontend/src/components/common/AutoRefreshButton.vue
frontend/src/components/layout/AppHeader.vue
frontend/src/components/layout/AppSidebar.vue
frontend/src/components/layout/AuthLayout.vue
frontend/src/components/payment/PaymentProviderDialog.vue
frontend/src/components/payment/SubscriptionPlanCard.vue
frontend/src/components/payment/__tests__/PaymentProviderDialog.spec.ts
frontend/src/components/payment/__tests__/SubscriptionPlanCard.spec.ts
frontend/src/components/user/dashboard/UserDashboardCharts.vue
frontend/src/i18n/locales/en/admin/accounts.ts
frontend/src/i18n/locales/en/admin/overview.ts
frontend/src/i18n/locales/en/admin/settings.ts
frontend/src/i18n/locales/en/batchImage.ts
frontend/src/i18n/locales/en/common.ts
frontend/src/i18n/locales/en/index.ts
frontend/src/i18n/locales/zh/admin/accounts.ts
frontend/src/i18n/locales/zh/admin/overview.ts
frontend/src/i18n/locales/zh/admin/settings.ts
frontend/src/i18n/locales/zh/batchImage.ts
frontend/src/i18n/locales/zh/common.ts
frontend/src/i18n/locales/zh/index.ts
frontend/src/types/index.ts
frontend/src/utils/__tests__/branding.spec.ts
frontend/src/utils/__tests__/formatDateTimeToMinute.spec.ts
frontend/src/utils/format.ts
frontend/src/views/HomeView.vue
frontend/src/views/KeyUsageView.vue
frontend/src/views/admin/AccountsView.vue
frontend/src/views/admin/BackupView.vue
frontend/src/views/admin/ProxiesView.vue
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/SubscriptionsView.vue
frontend/src/views/admin/__tests__/SettingsView.spec.ts
frontend/src/views/admin/ops/components/OpsDashboardHeader.vue
frontend/src/views/admin/orders/AdminPaymentPlansView.vue
frontend/src/views/admin/orders/__tests__/AdminPaymentPlansView.spec.ts
frontend/src/views/admin/settings/EmailTemplateEditor.vue
frontend/src/views/public/LegalDocumentView.vue
frontend/src/views/user/BatchImageGuideView.vue
frontend/src/views/user/CustomPageView.vue
frontend/src/views/user/SubscriptionsView.vue
```
<!-- TASK6:v0.1.162:raw:end -->
- Local key-file intersection (matrix-derived exact local core paths):
```text
backend/cmd/server/wire_gen.go
backend/internal/handler/openai_responses_image_intent_benchmark_test.go
backend/internal/server/middleware/session_binding.go
backend/internal/service/account_usage_service.go
backend/internal/service/image_task.go
backend/internal/service/openai_gateway_grok_cache.go
backend/internal/service/setting_parse.go
backend/internal/service/setting_update.go
backend/internal/service/user_subscription.go
frontend/src/i18n/locales/en/batchImage.ts
frontend/src/views/admin/__tests__/SettingsView.spec.ts
```

### v0.1.163

- Command: `git diff --name-only 'v0.1.162^{}..v0.1.163^{}'`
- Exact paths: 171
<!-- TASK6:v0.1.163:raw:begin -->
```text
README.md
backend/cmd/server/VERSION
backend/cmd/server/main.go
backend/ent/group.go
backend/ent/group/group.go
backend/ent/group/where.go
backend/ent/group_create.go
backend/ent/group_update.go
backend/ent/migrate/schema.go
backend/ent/mutation.go
backend/ent/runtime/runtime.go
backend/ent/schema/group.go
backend/go.mod
backend/go.sum
backend/internal/config/config.go
backend/internal/config/config_test.go
backend/internal/domain/reasoning_effort.go
backend/internal/handler/admin/group_handler.go
backend/internal/handler/admin/group_handler_reasoning_effort_test.go
backend/internal/handler/admin/usage_handler.go
backend/internal/handler/admin/usage_handler_request_type_test.go
backend/internal/handler/dto/mappers.go
backend/internal/handler/dto/types.go
backend/internal/handler/gateway_handler.go
backend/internal/handler/openai_chat_completions.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/pkg/apicompat/responses_client_tools.go
backend/internal/pkg/apicompat/responses_client_tools_test.go
backend/internal/repository/api_key_repo.go
backend/internal/repository/group_repo.go
backend/internal/repository/proxy_probe_service.go
backend/internal/repository/proxy_probe_service_test.go
backend/internal/repository/redis.go
backend/internal/repository/redis_test.go
backend/internal/repository/scheduler_cache.go
backend/internal/repository/scheduler_cache_last_used_unit_test.go
backend/internal/repository/scheduler_cache_unit_test.go
backend/internal/server/api_contract_test.go
backend/internal/service/account_test_service.go
backend/internal/service/account_test_service_openai_compact_test.go
backend/internal/service/account_test_service_openai_test.go
backend/internal/service/admin_group.go
backend/internal/service/admin_group_duplicate.go
backend/internal/service/admin_group_duplicate_test.go
backend/internal/service/admin_service.go
backend/internal/service/admin_service_group_test.go
backend/internal/service/api_key_auth_cache.go
backend/internal/service/api_key_auth_cache_impl.go
backend/internal/service/api_key_auth_cache_version_test.go
backend/internal/service/api_key_service_cache_test.go
backend/internal/service/channel_monitor_runner.go
backend/internal/service/channel_monitor_runner_test.go
backend/internal/service/gateway_anthropic_passthrough.go
backend/internal/service/gateway_forward_as_chat_completions.go
backend/internal/service/gateway_forward_as_chat_completions_test.go
backend/internal/service/gateway_forward_as_responses.go
backend/internal/service/gateway_forward_as_responses_test.go
backend/internal/service/gateway_non_streaming_response_test.go
backend/internal/service/grok_media.go
backend/internal/service/grok_upstream_errors.go
backend/internal/service/grok_upstream_errors_test.go
backend/internal/service/group.go
backend/internal/service/openai_account_runtime_block_fastpath.go
backend/internal/service/openai_account_scheduler.go
backend/internal/service/openai_account_scheduler_compact_test.go
backend/internal/service/openai_account_scheduler_test.go
backend/internal/service/openai_account_scheduler_upstream_cost_test.go
backend/internal/service/openai_apikey_responses_probe.go
backend/internal/service/openai_apikey_responses_probe_test.go
backend/internal/service/openai_codex_identity.go
backend/internal/service/openai_codex_identity_test.go
backend/internal/service/openai_codex_transform.go
backend/internal/service/openai_codex_transform_test.go
backend/internal/service/openai_gateway_cc_pipeline.go
backend/internal/service/openai_gateway_chat_completions_raw.go
backend/internal/service/openai_gateway_forward.go
backend/internal/service/openai_gateway_grok.go
backend/internal/service/openai_gateway_grok_cache.go
backend/internal/service/openai_gateway_grok_cache_test.go
backend/internal/service/openai_gateway_grok_chat_bridge.go
backend/internal/service/openai_gateway_grok_chat_bridge_test.go
backend/internal/service/openai_gateway_grok_compact.go
backend/internal/service/openai_gateway_grok_test.go
backend/internal/service/openai_gateway_grok_tool_protocol.go
backend/internal/service/openai_gateway_grok_tool_protocol_test.go
backend/internal/service/openai_gateway_response_handling.go
backend/internal/service/openai_gateway_response_handling_image_usage_test.go
backend/internal/service/openai_gateway_scheduling.go
backend/internal/service/openai_gateway_service.go
backend/internal/service/openai_gateway_service_test.go
backend/internal/service/openai_gateway_upstream_errors.go
backend/internal/service/openai_oauth_passthrough_test.go
backend/internal/service/openai_reasoning_effort_policy.go
backend/internal/service/openai_reasoning_effort_policy_test.go
backend/internal/service/openai_responses_namespace.go
backend/internal/service/openai_responses_namespace_test.go
backend/internal/service/openai_ws_forwarder.go
backend/internal/service/openai_ws_forwarder_ingress.go
backend/internal/service/openai_ws_forwarder_payload.go
backend/internal/service/openai_ws_forwarder_success_test.go
backend/internal/service/openai_ws_http_bridge.go
backend/internal/service/openai_ws_v2_passthrough_adapter.go
backend/internal/service/scheduler_snapshot_service.go
backend/internal/service/upstream_models.go
backend/internal/service/upstream_models_test.go
backend/internal/setup/handler.go
backend/internal/setup/setup.go
backend/internal/setup/setup_test.go
backend/migrations/185_group_reasoning_effort_policy.sql
deploy/.env.example
deploy/config.example.yaml
deploy/docker-compose.dev.yml
deploy/docker-compose.local.yml
deploy/docker-compose.standalone.yml
deploy/docker-compose.yml
docs/PAYMENT.md
docs/PAYMENT_CN.md
docs/screenshots/mobile-account-actions-menu.png
frontend/package.json
frontend/pnpm-lock.yaml
frontend/src/api/setup.ts
frontend/src/components/admin/group/GroupRPMOverridesModal.vue
frontend/src/components/admin/group/GroupRateMultipliersModal.vue
frontend/src/components/admin/group/ReasoningEffortPolicyFields.vue
frontend/src/components/admin/usage/UsageFilters.vue
frontend/src/components/admin/usage/__tests__/UsageFilters.spec.ts
frontend/src/components/charts/EndpointDistributionChart.vue
frontend/src/components/charts/GroupDistributionChart.vue
frontend/src/components/charts/ModelDistributionChart.vue
frontend/src/components/common/Select.vue
frontend/src/components/layout/AppHeader.vue
frontend/src/components/payment/SubscriptionPlanCard.vue
frontend/src/components/payment/__tests__/SubscriptionPlanCard.spec.ts
frontend/src/components/payment/__tests__/validity.spec.ts
frontend/src/components/payment/validity.ts
frontend/src/components/user/dashboard/UserDashboardCharts.vue
frontend/src/i18n/locales/en/admin/ops.ts
frontend/src/i18n/locales/en/admin/overview.ts
frontend/src/i18n/locales/en/landing.ts
frontend/src/i18n/locales/en/misc.ts
frontend/src/i18n/locales/zh/admin/accounts.ts
frontend/src/i18n/locales/zh/admin/ops.ts
frontend/src/i18n/locales/zh/admin/overview.ts
frontend/src/i18n/locales/zh/landing.ts
frontend/src/i18n/locales/zh/misc.ts
frontend/src/main.ts
frontend/src/types/index.ts
frontend/src/utils/__tests__/device.spec.ts
frontend/src/utils/__tests__/floatingPanel.spec.ts
frontend/src/utils/__tests__/formatMultiplier.spec.ts
frontend/src/utils/device.ts
frontend/src/utils/floatingPanel.ts
frontend/src/utils/formatters.ts
frontend/src/views/admin/AccountsView.vue
frontend/src/views/admin/GroupsView.vue
frontend/src/views/admin/PromoCodesView.vue
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/__tests__/groupsReasoningEffort.spec.ts
frontend/src/views/admin/groupsReasoningEffort.ts
frontend/src/views/admin/ops/components/OpsAlertEventsCard.vue
frontend/src/views/admin/ops/components/OpsAlertRulesCard.vue
frontend/src/views/admin/ops/components/OpsDashboardSkeleton.vue
frontend/src/views/admin/ops/components/OpsErrorDetailsModal.vue
frontend/src/views/admin/ops/components/OpsOpenAITokenStatsCard.vue
frontend/src/views/admin/ops/components/OpsRequestDetailsModal.vue
frontend/src/views/admin/ops/components/OpsSystemLogTable.vue
frontend/src/views/admin/ops/components/OpsThroughputTrendChart.vue
frontend/src/views/admin/ops/components/__tests__/OpsThroughputTrendChart.spec.ts
frontend/src/views/setup/SetupWizardView.vue
frontend/src/views/user/KeysView.vue
frontend/src/views/user/PaymentView.vue
```
<!-- TASK6:v0.1.163:raw:end -->
- Local key-file intersection (matrix-derived exact local core paths):
```text
backend/ent/schema/group.go
backend/go.mod
backend/internal/service/admin_group_duplicate_test.go
backend/internal/service/admin_group_duplicate.go
backend/internal/service/admin_group.go
backend/internal/service/admin_service.go
backend/internal/service/gateway_anthropic_passthrough.go
backend/internal/service/openai_account_scheduler_upstream_cost_test.go
backend/internal/service/openai_account_scheduler.go
backend/internal/service/openai_gateway_grok_cache.go
backend/internal/service/openai_gateway_response_handling_image_usage_test.go
backend/internal/service/openai_gateway_scheduling.go
backend/internal/service/scheduler_snapshot_service.go
backend/migrations/185_group_reasoning_effort_policy.sql
frontend/pnpm-lock.yaml
frontend/src/i18n/locales/en/admin/overview.ts
```

### v0.1.164

- Command: `git diff --name-only 'v0.1.163^{}..v0.1.164^{}'`
- Exact paths: 202
<!-- TASK6:v0.1.164:raw:begin -->
```text
README.md
README_CN.md
README_JA.md
assets/partners/logos/nagora.png
backend/cmd/server/VERSION
backend/cmd/server/wire.go
backend/cmd/server/wire_gen.go
backend/cmd/server/wire_gen_test.go
backend/ent/client.go
backend/ent/compositemodelroute.go
backend/ent/compositemodelroute/compositemodelroute.go
backend/ent/compositemodelroute/where.go
backend/ent/compositemodelroute_create.go
backend/ent/compositemodelroute_delete.go
backend/ent/compositemodelroute_query.go
backend/ent/compositemodelroute_update.go
backend/ent/ent.go
backend/ent/hook/hook.go
backend/ent/intercept/intercept.go
backend/ent/migrate/schema.go
backend/ent/mutation.go
backend/ent/predicate/predicate.go
backend/ent/runtime/runtime.go
backend/ent/schema/composite_model_route.go
backend/ent/tx.go
backend/internal/config/config.go
backend/internal/config/config_test.go
backend/internal/domain/constants.go
backend/internal/handler/admin/account_codex_import.go
backend/internal/handler/admin/account_codex_import_test.go
backend/internal/handler/admin/account_handler.go
backend/internal/handler/admin/account_handler_available_models_test.go
backend/internal/handler/admin/account_ollama_cloud_usage.go
backend/internal/handler/admin/account_ollama_cloud_usage_test.go
backend/internal/handler/admin/admin_basic_handlers_test.go
backend/internal/handler/admin/admin_service_stub_test.go
backend/internal/handler/admin/group_handler.go
backend/internal/handler/admin/payment_handler.go
backend/internal/handler/admin/payment_handler_test.go
backend/internal/handler/admin/setting_handler.go
backend/internal/handler/admin/setting_handler_update.go
backend/internal/handler/composite_platform.go
backend/internal/handler/composite_platform_test.go
backend/internal/handler/content_moderation_helper.go
backend/internal/handler/dto/account_mapper_redact_test.go
backend/internal/handler/dto/mappers.go
backend/internal/handler/dto/settings.go
backend/internal/handler/dto/types.go
backend/internal/handler/gateway_handler.go
backend/internal/handler/gateway_handler_cancellation_test.go
backend/internal/handler/gateway_handler_chat_completions.go
backend/internal/handler/gateway_handler_responses.go
backend/internal/handler/gateway_handler_warmup_intercept_unit_test.go
backend/internal/handler/gateway_key_billing_test.go
backend/internal/handler/gateway_models_test.go
backend/internal/handler/gemini_v1beta_handler.go
backend/internal/handler/grok_media.go
backend/internal/handler/no_account_error.go
backend/internal/handler/openai_chat_completions.go
backend/internal/handler/openai_embeddings.go
backend/internal/handler/openai_gateway_count_tokens.go
backend/internal/handler/openai_gateway_credential_failover_loop_test.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/openai_images.go
backend/internal/handler/ops_error_logger.go
backend/internal/handler/ops_platform_test.go
backend/internal/handler/payment_handler.go
backend/internal/handler/wire.go
backend/internal/payment/provider/alipay.go
backend/internal/payment/provider/alipay_test.go
backend/internal/payment/types.go
backend/internal/pkg/ctxkey/ctxkey.go
backend/internal/pkg/openai/constants.go
backend/internal/pkg/openai/constants_test.go
backend/internal/repository/account_repo.go
backend/internal/repository/account_repo_ollama_cloud_usage.go
backend/internal/repository/account_repo_ollama_cloud_usage_integration_test.go
backend/internal/repository/account_repo_ollama_cloud_usage_test.go
backend/internal/repository/account_repo_upstream_billing_probe_update_test.go
backend/internal/repository/auth_cache_invalidation_outbox_integration_test.go
backend/internal/repository/composite_model_route_repo.go
backend/internal/repository/group_repo.go
backend/internal/repository/proxy_repo.go
backend/internal/repository/proxy_repo_upstream_billing_probe_test.go
backend/internal/repository/simple_mode_default_groups.go
backend/internal/repository/simple_mode_default_groups_integration_test.go
backend/internal/repository/usage_log_effective_platform_test.go
backend/internal/repository/usage_log_repo.go
backend/internal/repository/wire.go
backend/internal/server/api_contract_test.go
backend/internal/server/http.go
backend/internal/server/middleware/audit_log.go
backend/internal/server/middleware/audit_log_test.go
backend/internal/server/router.go
backend/internal/server/routes/admin.go
backend/internal/server/routes/composite_platform_test.go
backend/internal/server/routes/gateway.go
backend/internal/server/routes/gateway_key_billing_test.go
backend/internal/server/routes/gateway_test.go
backend/internal/service/account_service.go
backend/internal/service/admin_account.go
backend/internal/service/admin_group.go
backend/internal/service/admin_service.go
backend/internal/service/admin_service_bulk_update_test.go
backend/internal/service/admin_service_composite_group_test.go
backend/internal/service/admin_service_group_test.go
backend/internal/service/audit_log.go
backend/internal/service/audit_log_test.go
backend/internal/service/channel_service.go
backend/internal/service/channel_service_test.go
backend/internal/service/composite_model_route.go
backend/internal/service/composite_platform.go
backend/internal/service/composite_platform_test.go
backend/internal/service/composite_route_resolver.go
backend/internal/service/composite_route_resolver_test.go
backend/internal/service/crs_sync_service.go
backend/internal/service/domain_constants.go
backend/internal/service/gateway_record_usage_test.go
backend/internal/service/gateway_scheduling.go
backend/internal/service/gateway_service.go
backend/internal/service/gateway_usage_billing.go
backend/internal/service/gateway_usage_billing_fallback_test.go
backend/internal/service/ollama_cloud_usage.go
backend/internal/service/ollama_cloud_usage_parser.go
backend/internal/service/ollama_cloud_usage_test.go
backend/internal/service/openai_account_scheduler.go
backend/internal/service/openai_account_scheduler_test.go
backend/internal/service/openai_gateway_grok.go
backend/internal/service/openai_gateway_grok_test.go
backend/internal/service/openai_gateway_passthrough.go
backend/internal/service/openai_gateway_request_body.go
backend/internal/service/openai_gateway_response_handling.go
backend/internal/service/openai_gateway_scheduling.go
backend/internal/service/openai_gateway_service.go
backend/internal/service/openai_gateway_service_test.go
backend/internal/service/openai_passthrough_normalization_test.go
backend/internal/service/openai_proxy_stream_circuit.go
backend/internal/service/openai_proxy_stream_circuit_test.go
backend/internal/service/payment_config_service.go
backend/internal/service/payment_config_service_test.go
backend/internal/service/payment_fulfillment_test.go
backend/internal/service/payment_order.go
backend/internal/service/payment_order_result_test.go
backend/internal/service/payment_service.go
backend/internal/service/payment_visible_method_instances.go
backend/internal/service/testdata/ollama_settings_usage.html
backend/internal/service/wire.go
backend/migrations/172_composite_model_routes.sql
backend/migrations/186_alipay_mobile_precreate_deep_link.sql
backend/migrations/186_group_auth_cache_image_generation.sql
deploy/.env.example
deploy/config.example.yaml
deploy/docker-compose.dev.yml
deploy/docker-compose.local.yml
deploy/docker-compose.standalone.yml
deploy/docker-compose.yml
docs/COMPOSITE_GROUPS.md
docs/PAYMENT_CN.md
frontend/src/api/__tests__/admin.accounts.ollamaCloudUsage.spec.ts
frontend/src/api/admin/accounts.ts
frontend/src/api/admin/groups.ts
frontend/src/api/admin/settings.ts
frontend/src/components/account/AccountStatusIndicator.vue
frontend/src/components/account/AccountUsageCell.vue
frontend/src/components/account/EditAccountModal.vue
frontend/src/components/account/OllamaCloudUsageCell.vue
frontend/src/components/account/OllamaCloudUsageSettings.vue
frontend/src/components/account/__tests__/AccountUsageCell.spec.ts
frontend/src/components/account/__tests__/OllamaCloudUsageCell.spec.ts
frontend/src/components/account/__tests__/OllamaCloudUsageSettings.spec.ts
frontend/src/components/common/DataTable.vue
frontend/src/components/common/GroupBadge.vue
frontend/src/components/common/GroupSelector.vue
frontend/src/components/common/PlatformIcon.vue
frontend/src/components/common/__tests__/DataTable.spec.ts
frontend/src/components/payment/PaymentStatusPanel.vue
frontend/src/components/payment/__tests__/PaymentStatusPanel.spec.ts
frontend/src/components/payment/__tests__/alipayDeepLink.spec.ts
frontend/src/components/payment/__tests__/paymentFlow.spec.ts
frontend/src/components/payment/alipayDeepLink.ts
frontend/src/components/payment/paymentFlow.ts
frontend/src/i18n/locales/en/admin/accounts.ts
frontend/src/i18n/locales/en/admin/overview.ts
frontend/src/i18n/locales/en/admin/settings.ts
frontend/src/i18n/locales/en/misc.ts
frontend/src/i18n/locales/zh/admin/accounts.ts
frontend/src/i18n/locales/zh/admin/overview.ts
frontend/src/i18n/locales/zh/admin/settings.ts
frontend/src/i18n/locales/zh/misc.ts
frontend/src/types/index.ts
frontend/src/types/payment.ts
frontend/src/utils/__tests__/ccswitchImport.spec.ts
frontend/src/utils/ccswitchImport.ts
frontend/src/utils/platformColors.ts
frontend/src/views/admin/AccountsView.vue
frontend/src/views/admin/ChannelsView.vue
frontend/src/views/admin/GroupsView.vue
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/__tests__/AccountsView.usageWindowsHint.spec.ts
frontend/src/views/admin/__tests__/SettingsView.spec.ts
frontend/src/views/admin/orders/__tests__/PlanEditDialog.spec.ts
frontend/src/views/user/PaymentView.vue
```
<!-- TASK6:v0.1.164:raw:end -->
- Local key-file intersection (matrix-derived exact local core paths):
```text
backend/cmd/server/wire_gen.go
backend/internal/handler/admin/setting_handler_update.go
backend/internal/handler/admin/setting_handler.go
backend/internal/handler/content_moderation_helper.go
backend/internal/handler/openai_images.go
backend/internal/repository/account_repo_upstream_billing_probe_update_test.go
backend/internal/repository/group_repo.go
backend/internal/repository/usage_log_repo.go
backend/internal/service/gateway_usage_billing.go
backend/internal/service/openai_account_scheduler.go
backend/internal/service/openai_gateway_grok.go
backend/internal/service/openai_gateway_passthrough.go
backend/internal/service/openai_gateway_request_body.go
backend/internal/service/openai_gateway_scheduling.go
backend/migrations/186_group_auth_cache_image_generation.sql
frontend/src/i18n/locales/en/admin/settings.ts
frontend/src/views/admin/__tests__/AccountsView.usageWindowsHint.spec.ts
```

### v0.1.165

- Command: `git diff --name-only 'v0.1.164^{}..v0.1.165^{}'`
- Exact paths: 168
<!-- TASK6:v0.1.165:raw:begin -->
```text
backend/cmd/server/VERSION
backend/ent/group.go
backend/ent/group/group.go
backend/ent/group/where.go
backend/ent/group_create.go
backend/ent/group_update.go
backend/ent/migrate/schema.go
backend/ent/mutation.go
backend/ent/runtime/runtime.go
backend/ent/schema/group.go
backend/internal/config/config.go
backend/internal/domain/constants.go
backend/internal/handler/admin/account_ollama_cloud_usage_test.go
backend/internal/handler/admin/group_handler.go
backend/internal/handler/auth_oauth_pending_flow_test.go
backend/internal/handler/batch_image_handler.go
backend/internal/handler/dto/mappers.go
backend/internal/handler/dto/types.go
backend/internal/handler/gateway_handler.go
backend/internal/handler/gateway_handler_chat_completions.go
backend/internal/handler/gateway_handler_responses.go
backend/internal/handler/gemini_v1beta_handler.go
backend/internal/handler/grok_media.go
backend/internal/handler/openai_alpha_search.go
backend/internal/handler/openai_chat_completions.go
backend/internal/handler/openai_embeddings.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/openai_images.go
backend/internal/handler/openai_images_failover_test.go
backend/internal/handler/openai_live.go
backend/internal/handler/openai_live_test.go
backend/internal/handler/user_handler_test.go
backend/internal/pkg/claude/constants.go
backend/internal/platform/liveattestation/attestation.go
backend/internal/platform/liveattestation/attestation_darwin.go
backend/internal/platform/liveattestation/attestation_unsupported.go
backend/internal/platform/liveattestation/attestation_unsupported_test.go
backend/internal/repository/account_repo_ollama_cloud_usage.go
backend/internal/repository/account_repo_ollama_cloud_usage_integration_test.go
backend/internal/repository/account_repo_ollama_cloud_usage_test.go
backend/internal/repository/api_key_repo.go
backend/internal/repository/batch_image_repo.go
backend/internal/repository/concurrency_cache.go
backend/internal/repository/concurrency_cache_integration_test.go
backend/internal/repository/concurrency_cache_live_test.go
backend/internal/repository/gateway_cache.go
backend/internal/repository/gateway_cache_live_test.go
backend/internal/repository/group_repo.go
backend/internal/repository/integration_harness_test.go
backend/internal/repository/migrations_schema_integration_test.go
backend/internal/repository/usage_log_repo_insert.go
backend/internal/repository/usage_log_repo_query.go
backend/internal/repository/usage_log_repo_request_type_test.go
backend/internal/repository/usage_log_session_id_integration_test.go
backend/internal/repository/usage_log_session_id_unit_test.go
backend/internal/repository/user_repo.go
backend/internal/repository/user_repo_email_alias_test.go
backend/internal/server/api_contract_test.go
backend/internal/server/middleware/admin_auth_test.go
backend/internal/server/routes/admin.go
backend/internal/server/routes/gateway.go
backend/internal/server/routes/prompt_audit_route_coverage_test.go
backend/internal/service/account.go
backend/internal/service/admin_group.go
backend/internal/service/admin_group_duplicate.go
backend/internal/service/admin_group_duplicate_test.go
backend/internal/service/admin_service.go
backend/internal/service/admin_service_apikey_test.go
backend/internal/service/admin_service_delete_test.go
backend/internal/service/admin_service_email_identity_sync_test.go
backend/internal/service/admin_service_group_test.go
backend/internal/service/api_key_auth_cache.go
backend/internal/service/api_key_auth_cache_impl.go
backend/internal/service/auth_oauth_email_flow.go
backend/internal/service/auth_service.go
backend/internal/service/auth_service_email_bind_test.go
backend/internal/service/auth_service_register_test.go
backend/internal/service/batch_image.go
backend/internal/service/batch_image_processor_test.go
backend/internal/service/batch_image_public.go
backend/internal/service/batch_image_public_test.go
backend/internal/service/batch_image_settlement.go
backend/internal/service/batch_image_settlement_test.go
backend/internal/service/bedrock_request.go
backend/internal/service/billing_service.go
backend/internal/service/claude_opus5_test.go
backend/internal/service/content_moderation_test.go
backend/internal/service/gateway_anthropic_apikey_passthrough_test.go
backend/internal/service/gateway_anthropic_passthrough.go
backend/internal/service/gateway_forward.go
backend/internal/service/gateway_upstream_response.go
backend/internal/service/gateway_usage_billing.go
backend/internal/service/gemini_chat_completions_compat_service.go
backend/internal/service/gemini_chat_completions_compat_service_test.go
backend/internal/service/gemini_messages_compat_service.go
backend/internal/service/group.go
backend/internal/service/ollama_cloud_usage.go
backend/internal/service/ollama_cloud_usage_test.go
backend/internal/service/openai_account_runtime_block_fastpath.go
backend/internal/service/openai_account_runtime_block_fastpath_test.go
backend/internal/service/openai_codex_transform.go
backend/internal/service/openai_gateway_apikey_item_id_test.go
backend/internal/service/openai_gateway_forward.go
backend/internal/service/openai_gateway_grok.go
backend/internal/service/openai_gateway_grok_cache.go
backend/internal/service/openai_gateway_grok_test.go
backend/internal/service/openai_gateway_scheduling.go
backend/internal/service/openai_gateway_service.go
backend/internal/service/openai_gateway_usage.go
backend/internal/service/openai_live.go
backend/internal/service/openai_live_attestation.go
backend/internal/service/openai_live_lifecycle_test.go
backend/internal/service/openai_live_test.go
backend/internal/service/openai_live_types.go
backend/internal/service/openai_oauth_passthrough_test.go
backend/internal/service/openai_responses_item_id.go
backend/internal/service/openai_responses_namespace.go
backend/internal/service/openai_responses_namespace_test.go
backend/internal/service/openai_responses_rejected_field_retry_test.go
backend/internal/service/openai_upstream_transport_error.go
backend/internal/service/openai_upstream_transport_error_handle_test.go
backend/internal/service/openai_ws_forwarder_success_test.go
backend/internal/service/pricing_service.go
backend/internal/service/pricing_service_test.go
backend/internal/service/registration_email_alias.go
backend/internal/service/registration_email_alias_test.go
backend/internal/service/session_id.go
backend/internal/service/session_id_test.go
backend/internal/service/usage_log.go
backend/internal/service/user_service.go
backend/internal/service/user_service_test.go
backend/migrations/187_add_usage_log_session_id.sql
backend/migrations/188_allow_live_usage_request_type.sql
backend/migrations/189_add_group_allow_live.sql
backend/migrations/190_add_users_email_alias_dedup_index_notx.sql
backend/resources/model-pricing/model_prices_and_context_window.json
deploy/config.example.yaml
frontend/package.json
frontend/pnpm-lock.yaml
frontend/src/api/__tests__/admin.accounts.ollamaCloudUsage.spec.ts
frontend/src/api/admin/groups.ts
frontend/src/components/account/AccountStatusIndicator.vue
frontend/src/components/admin/usage/UsageFilters.vue
frontend/src/components/admin/usage/UsageTable.vue
frontend/src/components/common/AnnouncementBell.vue
frontend/src/components/common/AnnouncementPopup.vue
frontend/src/components/common/__tests__/AnnouncementPopup.spec.ts
frontend/src/composables/useModelWhitelist.ts
frontend/src/i18n/locales/en/admin/overview.ts
frontend/src/i18n/locales/en/admin/resources.ts
frontend/src/i18n/locales/en/admin/settings.ts
frontend/src/i18n/locales/en/dashboard.ts
frontend/src/i18n/locales/zh/admin/overview.ts
frontend/src/i18n/locales/zh/admin/resources.ts
frontend/src/i18n/locales/zh/admin/settings.ts
frontend/src/i18n/locales/zh/dashboard.ts
frontend/src/styles/announcement-markdown.css
frontend/src/types/index.ts
frontend/src/utils/errorBadges.ts
frontend/src/utils/usageRequestType.ts
frontend/src/views/admin/AnnouncementsView.vue
frontend/src/views/admin/GroupsView.vue
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/UsageView.vue
frontend/src/views/admin/__tests__/SettingsView.spec.ts
frontend/src/views/user/AffiliateView.vue
frontend/src/views/user/UsageView.vue
frontend/src/views/user/__tests__/AffiliateView.spec.ts
```
<!-- TASK6:v0.1.165:raw:end -->
- Local key-file intersection (matrix-derived exact local core paths):
```text
backend/ent/schema/group.go
backend/internal/handler/openai_images.go
backend/internal/repository/group_repo.go
backend/internal/repository/user_repo.go
backend/internal/service/admin_group_duplicate_test.go
backend/internal/service/admin_group_duplicate.go
backend/internal/service/batch_image_settlement.go
backend/internal/service/batch_image.go
backend/internal/service/billing_service.go
backend/internal/service/content_moderation_test.go
backend/internal/service/gateway_anthropic_apikey_passthrough_test.go
backend/internal/service/gateway_anthropic_passthrough.go
backend/internal/service/openai_gateway_grok_cache.go
backend/internal/service/openai_gateway_scheduling.go
backend/internal/service/session_id.go
backend/internal/service/usage_log.go
backend/migrations/187_add_usage_log_session_id.sql
frontend/package.json
frontend/pnpm-lock.yaml
frontend/src/i18n/locales/en/admin/resources.ts
```

## Task 6 能力矩阵

`--` 表示该 tag 段没有命中此能力的本地核心入口；列内每个非空值均为对应段的实际 changed path。`gap` 是 Task 7 的唯一补测输入，不阻塞本 Task 的文档提交。

| 能力 | 160 | 161 | 162 | 163 | 164 | 165 | 基线保护状态与证据 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| advanced/layered scheduler | `backend/internal/repository/scheduler_cache.go`<br>`backend/internal/service/openai_gateway_scheduling.go` | `backend/internal/service/scheduler_snapshot_full_rebuild_lifecycle_test.go` | -- | `backend/internal/service/openai_account_scheduler.go`<br>`backend/internal/service/openai_gateway_scheduling.go` | `backend/internal/service/openai_account_scheduler.go`<br>`backend/internal/service/openai_gateway_scheduling.go` | `backend/internal/service/openai_gateway_scheduling.go` | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestLayered_PriorityDeterminism$'` |
| fallback/WaitPlan | `backend/internal/service/openai_gateway_scheduling.go` | -- | -- | `backend/internal/service/openai_account_scheduler.go`<br>`backend/internal/service/openai_gateway_scheduling.go` | `backend/internal/service/openai_account_scheduler.go`<br>`backend/internal/service/openai_gateway_scheduling.go` | `backend/internal/service/openai_gateway_scheduling.go` | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount$'` |
| DB recheck | `backend/internal/repository/scheduler_cache.go` | `backend/internal/service/scheduler_snapshot_batch_query_test.go` | -- | `backend/internal/service/openai_account_scheduler.go` | `backend/internal/service/openai_account_scheduler.go` | -- | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestLayered_GroupedAccountPassesDBFreshRecheck$'` |
| Grok/platform Sticky | `backend/internal/service/grok_media.go` | `backend/internal/server/middleware/session_binding.go` | `backend/internal/server/middleware/session_binding.go` | `backend/internal/service/openai_account_scheduler.go` | `backend/internal/service/openai_gateway_grok.go` | `backend/internal/service/session_id.go` | `protected`: Grok binding `go -C backend test -tags=unit ./internal/service -run '^TestLayered_SessionStickyPreservesGrokBinding$'`; platform toggle `go -C backend test -tags=unit ./internal/service -run '^TestGatewayService_SelectAccountForModelWithPlatform_StickyDisabledBypassesStickyReadAndWrite$'` |
| privacy | `backend/internal/handler/content_moderation_helper.go` | -- | -- | -- | `backend/internal/handler/content_moderation_helper.go` | `backend/internal/service/content_moderation_test.go` | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestLayered_PreviousResponseStickyHonorsRequirePrivacySet$'` |
| image capability | `backend/internal/handler/openai_images.go` | `backend/internal/service/openai_images.go` | `backend/internal/handler/openai_responses_image_intent_benchmark_test.go` | `backend/internal/service/openai_gateway_response_handling_image_usage_test.go` | `backend/internal/handler/openai_images.go` | `backend/internal/handler/openai_images.go` | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestLayered_RequiredImageCapabilityFiltersUnsupportedAccounts$'` |
| async images/object storage | `backend/internal/handler/batch_image_handler.go` | `backend/internal/service/openai_images_responses.go` | `backend/internal/service/image_task.go` | -- | -- | `backend/internal/service/batch_image.go` | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestImageTaskServiceCompleteOffloadsToStorage$'` |
| image and video billing | `backend/internal/pkg/xai/billing.go` | `backend/internal/service/upstream_billing_probe.go` | `backend/internal/service/account_usage_service.go` | `backend/internal/service/openai_gateway_response_handling_image_usage_test.go` | `backend/internal/service/gateway_usage_billing.go` | `backend/internal/service/batch_image_settlement.go`<br>`backend/internal/service/billing_service.go` | `protected`: image `go -C backend test -tags=unit ./internal/service -run '^TestCalculateImageCost$'`; video `go -C backend test -tags=unit ./internal/service -run '^TestCalculateVideoCostUsesSeparateConfig$'` |
| upstream multiplier | `backend/internal/service/admin_account_upstream_billing_probe_test.go` | `backend/internal/service/upstream_billing_probe.go` | -- | `backend/internal/service/openai_account_scheduler_upstream_cost_test.go` | `backend/internal/repository/account_repo_upstream_billing_probe_update_test.go` | -- | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestOpenAIFreshUpstreamBillingRateRecomputesPeakAtSelectionTime$'` |
| session/step-up | -- | `backend/internal/server/middleware/session_binding.go`<br>`backend/internal/server/middleware/step_up.go` | `backend/internal/server/middleware/session_binding.go` | -- | -- | `backend/internal/service/session_id.go` | `protected`: session `go -C backend test -tags=unit ./internal/service -run '^TestGatewayService_SelectAccountForModelWithPlatform_StickySession$'`; step-up `go -C backend test ./internal/server/middleware -run '^TestEnforceStepUpPassesWithGrant$'` |
| runtime hot update | -- | `backend/internal/service/setting_update.go` | `backend/internal/service/setting_update.go` | `backend/internal/service/scheduler_snapshot_service.go` | `backend/internal/handler/admin/setting_handler_update.go` | -- | `protected`: `go -C backend test -tags=unit ./internal/task3tests -run '^TestSettingService_UpdateSettings_PersistsAndHotUpdatesGatewayControls$'` |
| gateway passthrough | -- | `backend/internal/service/openai_gateway_passthrough.go` | -- | `backend/internal/service/gateway_anthropic_passthrough.go` | `backend/internal/service/openai_gateway_passthrough.go` | `backend/internal/service/gateway_anthropic_passthrough.go` | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestPassthroughFieldsV2OpenAIForward_APIKeyBodyMapCopiesFromOriginalInboundRequest$'` |
| prompt cache | -- | `backend/internal/service/openai_gateway_grok_cache.go` | `backend/internal/service/openai_gateway_grok_cache.go` | `backend/internal/service/openai_gateway_grok_cache.go` | -- | `backend/internal/service/openai_gateway_grok_cache.go` | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestForwardAsAnthropic_InjectsPromptCacheKeyForAPIKeyMessagesDispatch$'` |
| body replay/spooling | -- | `backend/internal/service/channel_monitor_checker_body_test.go` | -- | -- | `backend/internal/service/openai_gateway_request_body.go` | `backend/internal/service/gateway_anthropic_apikey_passthrough_test.go` | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestOpenAIForwardReusesBoundRequestBodyHandle$'` |
| failed usage | -- | `backend/internal/service/ops_service_user_error_test.go` | -- | -- | `backend/internal/repository/usage_log_repo.go` | `backend/internal/service/usage_log.go` | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestOpenAIForwardStreamingResponseFailedReturnsUsageWithError$'` |
| user resource control | -- | `backend/internal/repository/user_repo.go` | -- | `backend/internal/service/admin_service.go` | -- | `backend/internal/repository/user_repo.go` | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestAdminServiceUpdateUserBlockedGroups$'` |
| user bulk limits | -- | `backend/internal/handler/admin/user_handler_batch_limits_test.go` | -- | -- | -- | -- | `protected`: `go -C backend test ./internal/handler/admin -run '^TestUserHandlerBatchUpdateLimitsAcceptsPartialAndZeroValues$'` |
| public group blocking | -- | `backend/internal/handler/admin/user_handler_list_apikey_group_test.go` | -- | `backend/internal/service/admin_group.go` | `backend/internal/repository/group_repo.go` | `backend/internal/repository/group_repo.go` | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestUserCanBindGroupRejectsBlockedPublicGroup$'` |
| menu hiding | -- | `backend/internal/repository/user_repo.go` | -- | -- | -- | `backend/internal/repository/user_repo.go` | `protected`: `go -C backend test -tags=integration ./internal/repository -run '^TestUserRepoSuite$/TestHiddenUIResourcesRoundTrip$'` |
| frontend translations | `frontend/src/i18n/locales/en/common.ts` | `frontend/src/i18n/locales/en/admin/settings.ts` | `frontend/src/i18n/locales/en/batchImage.ts` | `frontend/src/i18n/locales/en/admin/overview.ts` | `frontend/src/i18n/locales/en/admin/settings.ts` | `frontend/src/i18n/locales/en/admin/resources.ts` | `protected`: `pnpm --dir frontend run test:run -- src/i18n/__tests__/localeKeysExist.spec.ts` |
| subscription quota atomic reset | -- | `backend/internal/service/subscription_service.go`<br>`backend/internal/service/user_subscription_daily_quota_test.go` | `backend/internal/service/user_subscription.go` | -- | -- | -- | `protected`: `go -C backend test ./internal/service -run '^TestAdminResetQuota_KeepsCacheAfterAtomicResetFailure$'`; the untagged `service` test asserts `resetUsageWindowsCalled`, selected daily/weekly windows, and an unchanged L1 cache (`cache.deletedKeys` empty) when the atomic reset fails. |
| settings backfill | -- | `backend/internal/service/setting_parse.go` | `backend/internal/service/setting_parse.go` | -- | `backend/internal/handler/admin/setting_handler.go` | -- | `protected`: `go -C backend test -tags=unit ./internal/task3tests -run '^TestSettingService_GetAllSettings_BackfillsGatewayControlsFromConfigAndDB$'` |
| group duplication | -- | -- | -- | `backend/internal/service/admin_group_duplicate.go`<br>`backend/internal/service/admin_group_duplicate_test.go` | -- | `backend/internal/service/admin_group_duplicate.go`<br>`backend/internal/service/admin_group_duplicate_test.go` | `protected`: `go -C backend test -tags=unit ./internal/service -run '^TestDuplicateGroupCopiesConfigurationDeeplyAndResetsRuntimeState$'` |
| Ent/Wire | `backend/cmd/server/wire_gen.go` | `backend/cmd/server/wire_gen.go` | `backend/cmd/server/wire_gen.go` | `backend/ent/schema/group.go` | `backend/cmd/server/wire_gen.go` | `backend/ent/schema/group.go` | `protected`: `go -C backend test ./cmd/server` |
| dependencies | `frontend/package.json` | `frontend/vite.config.ts` | -- | `backend/go.mod` | -- | `frontend/package.json` | `manual`: run `go -C backend mod verify` and `pnpm --dir frontend install --frozen-lockfile`; these validate declared dependency graphs without a behavior-specific assertion. |
| migration | `backend/migrations/181_prompt_audit.sql` | `backend/migrations/183_ops_ingress_reject_aggregates.sql` | -- | `backend/migrations/185_group_reasoning_effort_policy.sql` | `backend/migrations/186_group_auth_cache_image_generation.sql` | `backend/migrations/187_add_usage_log_session_id.sql` | `protected`: `go -C backend test -tags=integration ./internal/repository -run '^TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate$'` |
| local test gates | `frontend/package.json` | `frontend/vite.config.ts` | `frontend/src/views/admin/__tests__/SettingsView.spec.ts` | `frontend/pnpm-lock.yaml` | `frontend/src/views/admin/__tests__/AccountsView.usageWindowsHint.spec.ts` | `frontend/pnpm-lock.yaml` | `protected`: `make test` |
| openai-first-token-timeout | -- | -- | -- | -- | -- | -- | `approved-removal`: Task brief explicitly permits this row alone; no changed local timeout entry was found in any of the six ranges. |

### CodeGraph 影响证据

官方 CLI 查询均在 `sync` 后执行，未使用 grep 重建调用链：

- `impact 'OpenAIGatewayService::getOpenAIAccountSchedulerWithContext' --json`: `openai_account_scheduler.go:2132` 影响 `SelectAccountWithScheduler`、`SelectAccountWithSchedulerForCapability`、`SelectAccountWithSchedulerForImages` 和 `handleOpenAISchedulerSettingsUpdate`。
- `impact 'layeredOpenAIAccountScheduler::selectByLayeredFilter' --json`: `openai_account_scheduler_layered.go:264` 由 `Select` 使用，确认分层选择入口。
- `impact 'GatewayService::stickyEnabledForPlatform' --json`: `gateway_service.go:708` 到 `SelectAccountWithLoadAwareness`、HTTP `ChatCompletions`/`Responses`、`Messages` 和 sticky read/bind 方法；影响图也列出 Sticky-disabled 现有测试。
- `impact 'Account::SupportsOpenAIImageCapability' --json`: `account.go:1533` 到 normal/layered scheduler compatibility、session sticky recheck 及两项 image-capability 测试。
- `impact 'BatchImageQueue' --json`: `batch_image_queue.go:27` 到 queue/public service/worker/runtime 及 worker lifecycle tests，确认异步图片链路。
- `impact 'ContentModerationConfig' --json`: `content_moderation.go:140` 到 `UpdateConfig`、`refreshRuntimeSnapshot`、`Check`、`enqueueAsync` 与已存在的 deleted-group/privacy tests。
- `impact 'OpenAIGatewayService::bindOpenAICompatAnthropicDigestPromptCacheKey' --json`: `openai_messages_digest_session.go:75` 到 `ForwardAsAnthropic`、prompt-cache/continuation tests 和 OpenAI `Messages` handler。
- `impact 'RequestBodyRef' --json`: `gateway_request.go:61` 影响 `ReplaceBody`、OpenAI forward/WS ingress、Anthropic retry loop、spool/replay tests 和 terminal failed-usage paths。
- `impact 'OpenAIGatewayService::forwardOpenAIPassthrough' --json`: `openai_gateway_passthrough.go:30` 到 `Forward`、responses/chat fallback、bound-body failover/replay、timeout and passthrough tests。
- `impact 'upstreamBillingRateAt' --json`: `upstream_billing_probe.go:760` 到 `openAIFreshUpstreamBillingRate`、`openAISchedulingRate` 和 peak-rate/DST tests。
- `impact 'TotpHandler::StepUp' --json` 与 `impact 'stepUpAuth' --json`: `totp_handler.go:206` 到 `POST /step-up` 路由，及 `step_up.go:46` 到 `NewStepUpAuthMiddleware`。
- `impact 'userRepository::SetHiddenUIResources' --json`: `user_repo.go:1124` 到 `TestHiddenUIResourcesRoundTrip`。
- `impact 'adminServiceImpl::DuplicateGroup' --json`: `admin_group_duplicate.go:162` 到 deep-copy、idempotent recover、name and atomic-create-failure tests；该路径补齐主 OpenSpec 的分组复制能力。
- `impact 'UserHandler::BatchUpdateLimits' --json`: `handler/admin/user_handler.go:624` 是批量 concurrency/RPM 路由入口；其服务和 repository `BatchUpdateLimits` 的精确实现符号已由 `query 'BatchUpdateLimits' --json` 定位。
- `impact 'SubscriptionService::AdminResetQuota' --json`: `subscription_service.go:881` 到 `ResetUsageWindows` 单次窗口重置和 `TestAdminResetQuota_*` tests；此项替代错误的 OpenAI/Grok account quota 映射。
- `impact 'BillingService::CalculateVideoCost' --json`: `billing_service.go:1448` 到 `calculateOpenAIVideoCost` 与分辨率/每秒计费测试，补齐图片和视频双计费证据。
- `impact 'SettingService::GetAllSettings' --json`: `setting_service.go:259` 到 gateway-control backfill test 及 admin `UpdateSettings`/`GetSettings` handlers，补齐 settings backfill。
- `impact 'validateAPIKeyGroupAllowed' --json`: `server/middleware/api_key_auth.go:337` 经 `abortIfAPIKeyGroupNotAllowed` 到 API-key subscription middleware 与 Google variant；这是公开分组阻断的真实调用链，而非 definition-only query。
- `query 'openAIFirstOutputTimeout' --json`: 精确定位 `service/openai_first_output_timeout.go:232` 和现有 timeout 测试；本行按 brief 的唯一批准例外登记。

### Task 6 自审

- 矩阵有 28 行：brief 的 26 行加主 OpenSpec 明确要求、此前遗漏的 `group duplication` 与独立 `user bulk limits`。`user resource control` 保持独立，避免将两种用户管理边界混淆。
- 状态汇总：`protected=26`、`manual=1`、`gap=0`、`approved-removal=1`。Task 7 没有由本矩阵产生的补测输入。
- 所有矩阵 Go regex 都按目标文件 build tag 修正；`-list` 已实际列出 runtime hot update、user bulk limits、user resource control、group duplication、subscription quota reset、image/video billing。integration repository runner 在 Docker 缺失时列举前跳过，因此 menu/migration 的准确 suite/test name 另由带 `integration` build tag 的源码位置核验，未声称已执行 integration test。
- 原始清单的 machine check 在本次提交前逐段重取 `git diff --name-only`，抽取本 ledger `TASK6:*:raw` block，以 `Compare-Object` 比较，且同时断言计数 `133/257/190/171/202/168`。

## Task 7 阶段 0 封闭（2026-07-27）

- 固定输入：开始及归档 HEAD 均为 `d32256d4fc557f20b87a1c802e2d54c938f5226d`，分支为 `feature/20260726/staged-merge-upstream-v0-1-165`。开始和提交前暂存区均为空；协调者的 plan、OpenSpec `tasks.md`/progress，以及根 `.comet/current-change.json`、`paseo.json` 未触碰。
- Task 6 reviewer Approved 矩阵为 28 行：`protected=26`、`manual=1`、`gap=0`、`approved-removal=1`。因此没有 `gap` 行需要 RED/GREEN 或新增测试；TDD N/A，不伪造失败或通过测试。唯一 `approved-removal` 是明确许可的 `openai-first-token-timeout`，不构成补测输入。
- 本地 full gate：在当前 PowerShell 进程以 `$env:PATH = "C:\Program Files\Git\bin;C:\Program Files\Git\usr\bin;" + $env:PATH` 运行 `make test` 和 `make build`，均退出 `0`。`make test` 的 Vitest 摘要为 `194 passed` 文件、`1493 passed` 用例；`make build` 构建 backend `0.1.159.6`，Vite 为 `987 modules transformed`。Browserslist、Vue/i18n 测试 stderr、动态 import/chunk-size 信息均为已有 advisory，命令没有失败。
- 本地静态 gate：`git diff --name-only 'v0.1.159^{}..HEAD'` 退出 `0`；`git diff --name-only --diff-filter=U` 无输出；`git diff --check` 退出 `0`（仅协调者既有文件的 CRLF 提示）。精确冲突扫描实际命令为 `git grep -n -I -E '^(<<<<<<< |=======$|>>>>>>> )' HEAD; if ($LASTEXITCODE -eq 1) { exit 0 }; exit $LASTEXITCODE`：无输出时内层 `git grep` 的预期退出码为 `1`，wrapper 整体退出 `0`。初版宽松扫描曾命中 `request_transformer.go` 的 MCP 协议等号横线，精确规则只匹配完整冲突行，故不是冲突标记。`backend/cmd/server/VERSION` 为 `0.1.159.6`，根 `VERSION` 不存在。
- 远程 Task 4 重跑：从当前已提交 HEAD 执行 `git archive --format=tar HEAD`，唯一 tar 是 `C:/Users/caiqy/AppData/Local/Temp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59.tar`，大小 `49827840` bytes，SHA-256 `37DD9EED379D633FFE6ED69551B962B595AB049F3DDBED2874D667C0FFA74D16`。只经 `ssh-skill` Python 脚本创建、上传、运行和清理唯一远端目录 `/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59`；所有实际远端 JSON 均为 `success=true, exit_code=0`。
- 远程预检严格确认 `go version go1.26.5 linux/amd64` 与 `Docker ServerVersion=29.2.1`。没有安装 Make/PowerShell、构建镜像、部署、访问服务或生产数据。首次含 PowerShell `$()` 的本地 `ssh_execute.py` 参数在调用端被 argparse 拒绝，未建立 SSH 或执行远端预检；已改为不含 shell substitution 的精确等价命令，结果如上。
- 远程 full integration 在 archive 的 `backend` 重建 `.test-tmp`，以同一绝对路径设置 `TMPDIR`/`TMP`/`TEMP` 后运行 `CI=true GOFLAGS='-v' go test -tags=integration ./...`，JSON 为 `success=true, exit_code=0`。完整日志保存于 `C:/Users/caiqy/AppData/Local/Temp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59-integration.log`；其中 `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 明确 `PASS (4.20s)`，无 `FAIL`。
- 远程 skip 共 7 项，均为既有非目标风险：DingTalk disabled sentinel；未设 `TLSFINGERPRINT_CAPTURE_URL`；外部 `tls.peet.ws` 证书问题导致 JA3 和两个 profile 子例；已知 CI concurrency TODO；未设 `OPENAI_API_KEY`。日志下载 JSON 为 `success=true, exit_code=0`。远端清理以 `rm -rf ... && test ! -e ...` 返回 `success=true, exit_code=0`；本地 tar 已删除并确认不存在，完整日志按要求保留。

### Task 7 ssh-skill 调用审计

以下是原始会话中实际运行的调用，不含凭据。相应 tool record ID 依次为 `prt_fa046bdae001E2vY9V2N5ZkW8r`、`prt_fa046e161001YmXB3kUV4niA3f`、`prt_fa04700d9001wQ9zPM0tHbdwi4`、`prt_fa047587a001VQ7nB63to2dWPt`、`prt_fa047a31a0015a1DByzrCeRPgC`、`prt_fa04bbbf8001jdh0uCd33zA9bi`、`prt_fa04da039001DBwi0IvyjKM65q`、`prt_fa052bf4f00169weJlM4axitTh`。

```powershell
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "umask 077 && mkdir -p '/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59/src'"
```

create：JSON `success=true, exit_code=0`。

```powershell
$env:MSYS_NO_PATHCONV = '1'; python ~/.claude/skills/ssh-skill/scripts/ssh_upload.py local-serv-ai "C:/Users/caiqy/AppData/Local/Temp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59.tar" "/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59/source.tar" --no-progress
```

upload：JSON `success=true, exit_code=0`。

```powershell
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; go version; test \"$(go version | sed -n 's/^go version go\\([0-9.]*\\).*/\\1/p')\" = '1.26.5'; docker info --format 'ServerVersion={{.ServerVersion}}'"
```

首次 precheck：本地 `ssh_execute.py` argparse 拒绝，错误为 `unrecognized arguments: \ = '1.26.5'; docker info --format 'ServerVersion={{.ServerVersion}}'`；PowerShell 展开 `$()` 后命令没有建立 SSH、没有执行远端预检。

```powershell
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; go version | grep -Fx 'go version go1.26.5 linux/amd64'; docker info --format 'ServerVersion={{.ServerVersion}}'"
```

成功 precheck：JSON `success=true, exit_code=0`；stdout 为 `go version go1.26.5 linux/amd64` 和 `ServerVersion=29.2.1`。

```powershell
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; test -f '/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59/source.tar'; tar -xf '/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59/source.tar' -C '/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59/src'; cd '/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59/src/backend'; rm -rf .test-tmp; mkdir .test-tmp; CI=true GOFLAGS='-v' TMPDIR='/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59/src/backend/.test-tmp' TMP='/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59/src/backend/.test-tmp' TEMP='/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59/src/backend/.test-tmp' go test -tags=integration ./... > '/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59/integration.log' 2>&1" --timeout 1200
```

integration：JSON `success=true, exit_code=0`。

```powershell
$env:MSYS_NO_PATHCONV = '1'; python ~/.claude/skills/ssh-skill/scripts/ssh_download.py local-serv-ai "/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59/integration.log" "C:/Users/caiqy/AppData/Local/Temp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59-integration.log" --no-progress
```

download：JSON `success=true, exit_code=0`。

```powershell
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "rm -rf '/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59' && test ! -e '/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59'"
```

cleanup：JSON `success=true, exit_code=0`。

```powershell
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "test ! -e '/tmp/sub2api-stage-0-58619bc639bb414e947be37a6ea56b59'"
```

final absence check：JSON `success=true, exit_code=0`。
- Task 5 重跑：在 `C:/Users/caiqy/AppData/Local/Temp/opencode/wf09b22` 以 `git -c core.longpaths=true worktree add --detach <path> HEAD` 建立短路径 detached worktree。两轮均按顺序运行 `make -C backend generate`、`git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go`、`git status --short`；六项退出 `0`，每轮 diff/status 均无输出。`git -c core.longpaths=true worktree remove <path>` 退出 `0`，路径和 `git worktree list --porcelain` 注册均无残留。
- 主工作区最终静态复核：generated path diff 退出 `0`；`git ls-tree -r --name-only HEAD backend/migrations` 为 238 路径，其中 231 SQL、7 支持文件。只有 `172_video_per_second_billing_metadata.sql` 与 `181_group_duplicate_operation_id.sql`，没有 `186*`。指定 runner `git grep` 退出 `0`：`fs.Glob` 后 `sort.Strings` 按完整 filename，`schema_migrations.filename` 是主键并保存 SHA-256 checksum，`*_notx.sql` 逐语句非事务执行且限制幂等的 `CREATE/DROP INDEX CONCURRENTLY`。
- 当前残余风险：Windows 受监视主工作区的历史 user-mapped section 问题未定因；本轮仅以 reviewer 接受的 detached worktree 双轮证明当前 committed HEAD 的生成稳定。远程的七项 skip 仍是外部环境/已知 CI 风险。阶段 0 已关闭；没有开始 tag merge、push、tag、release 或 deploy。

## Task 8: v0.1.160 合入与记账

- 合并提交：`e04cb1aa2c2554a04bec55f9b4393d3efd2eb693 merge: upstream v0.1.160`。
- 父关系：第一父 `d3e0c596ebff2298d07a3f4f336c16aa653cb840`，第二父（`v0.1.160^{}`）`8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62`；tag peeled SHA 已在 merge 前复核。
- 9 个文本冲突及决议：
  - `backend/cmd/server/VERSION`：保留本地中间版本 `0.1.159.6`，未接受 tag 内的 `0.1.159`。
  - `backend/internal/handler/grok_media.go`、`grok_media_test.go`：保留本地 multipart 快照脱敏、spool 和 sticky 覆盖，同时加入上游 Grok media capability 过滤及测试。
  - `backend/internal/handler/openai_gateway_handler.go`：HTTP/WS Responses 使用上游显式 image-generation capability；保留本地 group permission 检查；删除已移除的 `openai-first-token-timeout` 分支、常量和测试。
  - `backend/internal/handler/openai_images.go`：先运行本地 content moderation，再运行上游 security audit；两者均通过后才释放 multipart/text。
  - `backend/internal/service/grok_media.go`、`openai_gateway_grok_test.go`：同时接受本地 `image_url`/`mask_image_url` 输入兼容和上游 `reference_images`/官方 `url` 出站规范；保留 multipart 上限测试。
  - `backend/internal/service/openai_account_scheduler_test.go`：同时保留本地 Grok chat/session sticky 与上游 media eligibility 过滤覆盖。
  - `openspec/config.yaml`：保留本地中文 OpenSpec 规则并保留上游模板注释。
- 无文本冲突审查：security audit 的 `181_prompt_audit.sql` 与 `182_prompt_audit_full_prompt.sql` 均保留；`FullPromptFromScanText` 仅将完整提示词写入 audit event，`RequirePrivacySet` 仍独立约束上游账号调度，未被改写。完整提示词审计是数据留存边界，Task 9 必须纳入安全/回归审查，不得据此放行下一 tag。
- `image_gen`：账号 capability 仅对显式 image-generation intent 提高到 Responses；本地平台级 group permission 检查仍覆盖 HTTP/WS 请求。Grok media 继续隔离到 generation capability，并保留本地请求/usage snapshot 脱敏。
- migration：完整 filename 均存在，`181_group_duplicate_operation_id.sql`、`181_prompt_audit.sql`、`182_prompt_audit_full_prompt.sql` 的字典序和依赖正确，后者只 ALTER 前者创建的 event 表。
- 能力矩阵交集已审查：`wire_gen`、batch image、content moderation、OpenAI images、xAI billing、scheduler cache、upstream billing probe、Grok media、OpenAI scheduling、`181_prompt_audit.sql`、frontend package/i18n；`openai-first-token-timeout` 仍为唯一 approved-removal。
- 静态事实：根 `VERSION` 不存在，服务器 VERSION 为 `0.1.159.6`，无 conflict marker/未合并路径；`git diff --cached --check` 在提交前为 0。上游 source-freeze patch 含 literal diff whitespace，`.gitattributes` 对该单一路径设 `-whitespace`，以保持其内容原样。
- Task 9 full gate、回归修复和任何下一 tag 合入均尚未运行，当前不得放行 `v0.1.161`。
