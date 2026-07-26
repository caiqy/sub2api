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
- `v0.1.160`：待执行。
- `v0.1.161`：待执行。
- `v0.1.162`：待执行。
- `v0.1.163`：待执行。
- `v0.1.164`：待执行。
- `v0.1.165`：待执行。

## 阻塞与残余风险

- 历史阻塞（已解决，归档 HEAD `c8e0110a9a2354453753db9c4acae0ed7570458d`）：原 Make/PowerShell/Go 预检依次暴露 GNU Make 缺失、Go `1.26.1` 低于 directive、PowerShell 入口缺失；后续规范已改为 Linux 原生 gate，vfox Go `1.26.5` 和 Docker `29.2.1` 已在新 nonce 中验证。
- 历史阻塞（已解决，规范 HEAD `849f956992178e25ab2074e1e4cc596d29f8834f`）：首次原生 full RED 暴露 handler body spool/panic、repository usage-detail transaction/retention 与 hidden UI resource ordering 断言；后续测试修复及 HEAD `99cb81de306cb0e8ea811387e362e4d601f6f4b0` 的新 nonce focused/full GREEN 已解决。
- 当前状态：阶段 0 本地与远程门禁通过；七项非 migration/repository target skip 是已分类的基线风险。Task 5-7（生成/migration 静态稳定性、能力矩阵与保护断言）及六个 tag merge 仍未完成，因此仍不得开始或放行 `v0.1.160`。
