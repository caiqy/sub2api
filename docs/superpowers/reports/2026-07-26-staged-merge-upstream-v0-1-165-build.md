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
| changed-files 与本地能力交集 | `v0.1.160` | 证据已补齐，等待独立复审 | Task 10：133 paths、12 个精确交集路径、14 项矩阵结论及 Task 9 门禁证据见第 1845 行。 |
| changed-files 与本地能力交集 | `v0.1.161` | 证据修正待 Sol 复审 | Task 13：257 paths、22 个精确交集路径、`protected=21`、`manual=1`、`gap=0`；跨路径补充回归另列，见 `Task 13`。 |
| changed-files 与本地能力交集 | `v0.1.162` | 已闭合 | Task 16 review round 1：190 个 raw changed files；4 个指定能力行引用 41 条选定核心证据路径（`9+9+9+14`），不是穷举 file-level intersection。`protected=4`、`manual=0`、`gap=0` 只定义为四个 capability row 均有自动测试和当前调用链证据；见 v0.1.162 Task 16 closure。 |
| changed-files 与本地能力交集 | `v0.1.163` 至 `v0.1.165` | 待执行 | 尚未开始实际 diff 与调用链审查。 |

## v0.1.160

- changed-files：`git diff --name-only 'v0.1.159^{}..v0.1.160^{}'` 已确认 133 paths，完整清单见 `TASK6:v0.1.160:raw`。
- 冲突台账：9 个冲突文件及逐项融合结论见第 1818-1825 行；merge 为 `e04cb1aa2c2554a04bec55f9b4393d3efd2eb693`。
- 能力矩阵交集：12 个精确路径，14 项计数为 `protected=13`、`manual=1`、`gap=0`，见 Task 10。
- 聚焦测试：Task 9 的 scheduler、sticky/privacy/image、migration、audit 证据均为 GREEN，见第 1833-1842 行。
- 本地门禁：`make test`、`make build`、两轮 generate/generated diff 和静态门禁均为 GREEN。
- 远程门禁：archive `31b132689` 的 full integration GREEN；日志、remote 目录及本地 tar 清理证据见第 1842、1875 行。
- Task 11 状态：证据已补齐，等待独立复审。

## v0.1.161

- changed-files/冲突：已合并 `f2158292c`，26 个冲突与 Task 11 的六项 review finding 已闭合；Task 12 follow-up 不改写 merge 历史。
- 精确交集：22 条路径表的 Ent/Wire、用户资源、sticky/session、step-up、spooling、prompt cache、passthrough、images、failed usage、DB fresh recheck、scheduler、settings、quota、billing、migration、i18n 与 dependency gate 计为 `protected=21`、`manual=1`、`gap=0`。
- 跨路径补充聚焦回归（不属于上述 22 条路径或计数）：模型冷却、fallback/WaitPlan、Grok 视频/owner/redaction、content moderation、HTTP/WS failover、YAML、API Key/billing probe UI 与 migration/repository 均 GREEN。
- WS V2：`0775a6063` 的 terminal ownership 放宽与 `f1cde1b52` 的反向测试适配均非最终 GREEN；`c3bfb765f` 恢复 created-only ownership，`47a6c031e` 仅补齐正常 terminal-first fixture 的同 ID `response.created`。
- 本地门禁：最终 `make test`、进程级 Git `sh` PATH 下的 `make build`、两轮 `make -C backend generate` 和 Ent/Wire 零 diff、静态/冲突/VERSION/旧 first-token timeout/migration 集合检查均 GREEN；VERSION 保持 `0.1.159.6`。
- 远程门禁：archive `07029cc45` 在 `local-serv-ai`（Go `1.26.5`、Docker `29.2.1`）以隔离 `.test-tmp` 完整 integration GREEN；migration 新库幂等与从本地 v0.1.159.6 升级均 PASS，16 个已接受环境型 skip 未命中本阶段能力。
- 阶段结论：`DONE_WITH_CONCERNS`。v0.1.161 实现门禁以精确 `gap=0` 闭合，Task 13 docs/evidence 修正待 Sol 复审；Task 14/v0.1.162 在该复审 PASS 前保持封闭。风险为 Windows Git `sh`/历史文件锁及既有前端 advisory。

## v0.1.162

- changed-files：`v0.1.161^{}` 到 `v0.1.162^{}` 共 190 条，原始清单见 `TASK6:v0.1.162:raw:begin/end`。
- merge：`8bda73544d6e26a323f101e5c68981634f0375ab merge: upstream v0.1.162`；第一父 `940c5cfcf390ecbfd2e041fb2b46c99846e6ea3e`，第二父及 peeled tag 为 `27f094e0960ebd8e52de7ff7e763c6fec2ff4057`。
- 原始 U 文件：`VERSION`、`config.go`、`config_test.go`、两个 admin setting handler 文件、`openai_gateway_handler_test.go`、`openai_gateway_messages.go`、`openai_ws_http_bridge.go`、`setting_service.go`、`setting_update.go`、`service/wire.go`、`EditAccountModal.vue`、两份 locale index；逐项决议见 Task 14。
- 冲突融合：VERSION 保持 `0.1.159.6`；可信代理/forwarded header 保留本地安全入口与 runtime scheduler/sticky 热更新，接入上游 header JSON 回填与配置；Grok 保留 usage/ops/request body 处理和 created-only WS 契约，接入 cache 与单次 encrypted-content retry；Wire 保留 startup recovery 并接入 settings-resolved S3/image storage；前端同时保留 images/batch-image locale、header override 与 Grok cache 开关。
- 高风险无文本冲突审查：`pkg/ip -> middleware -> server/http` 保持 Gin trusted-proxy fail-closed 路径；`grok cache -> session binding -> HTTP/WS bridge` 保持 platform sticky、owner/spool/redaction/usage 入口；`backup/image storage -> image task -> admin handler` 保持异步任务、对象存储、moderation/audit 和权限边界。Task 15 负责行为回归。
- 静态检查：`git ls-files -u`、真实 conflict marker 扫描和 `git diff --cached --check` 均为空/通过；限定 Go 编译和 `pnpm exec vue-tsc --noEmit` 通过。未运行 Task 15 full test/build/generate/integration。
- 迁移与遗留边界：本地 `172_video_per_second_billing_metadata.sql`、`181_group_duplicate_operation_id.sql` 与上游 `181_prompt_audit.sql`、182、183、184 均保留；`VERSION=0.1.159.6`，`openai-first-token-timeout` 未恢复。
- 放行结论：`DONE_WITH_CONCERNS`，等待 Task 15 回归与 full gates。
- Sol reviewer 第 1 轮 cleanup：`46eb292c04f14630dcfb31b28f6e83f541029d93 chore: clean v0.1.162 conflict remnants` 仅清除两个 admin setting handler 中三段冲突遗留的 `/* ... */` 死副本，并恢复 step-up 启用防自锁、关闭强制 step-up 的安全理由注释；未改变实际 JSON/setting 映射、trusted proxy、scheduler/sticky、audit 或 step-up 代码。
- cleanup 验证：`gofmt -w backend/internal/handler/admin/setting_handler.go backend/internal/handler/admin/setting_handler_update.go`、`go -C backend test ./internal/handler/admin -run '^TestUpdateSettings(EnableStepUp|DisableStepUp)' -count=1`、`go -C backend test ./internal/handler/admin -run '^$'` 均退出 `0`；`git diff --check` 退出 `0`。TDD N/A：仅注释清理，不伪造 RED；Task 15 full gates、远程操作均未运行。
- Task 15 回归与 full gate：以 `98fa814d2` 为起点，四个真实 RED 修复提交为 `57b5bf758`、`1d191894a`、`84d9a4f4f`、`69ac6209f`；分别闭合 HTTP/WS failover body 与 terminal event、Grok client-tool cache 提交、settings env 可达默认值/forwarded-IP fixture、rollback timeout 契约。聚焦矩阵均 GREEN；`make test`（Vitest `204` 文件、`1549` 用例）、`make build`、两轮 `make -C backend generate` 加 Ent/Wire 零 diff、unmerged/精确 marker/diff-check/migration 集合检查均通过，VERSION 保持 `0.1.159.6`。
- Task 15 远程 integration：archive `69ac6209fcb4928251142a3368ffdbd750919076` 在 `local-serv-ai` 的 `/tmp/sub2api-task15-b369bbfa849c4b8fadad749d4c66f2b9` 完整通过。Go `1.26.5`、Docker Server `29.2.1`；`CI=true GOFLAGS='-v'` 和隔离三变量 temp 的 `go test -tags=integration ./...` 退出 `0`，`TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` PASS（`4.80s`），`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS（`4.58s`）。日志保存于 `C:/Users/caiqy/AppData/Local/Temp/sub2api-task15-b369bbfa849c4b8fadad749d4c66f2b9-integration.log`，archive 与远程目录均已清理；配置型/环境型 skip 已记录于 `.superpowers/sdd/task-15-report.md`，无 `FAIL`。
- Task 15 Sol review round 1：`be5b98314` 将 transport 的 Ops/durable-account side effects 与首 turn failover 返回值最小拆分；后续 turn 或已写 downstream 的 `Do`/scanner/terminal transport failure 记录健康副作用、返回普通 error 并向仍连接客户端发送固定 `Upstream request failed` event，不回显内部地址且不触发 replay。context cancellation 与已识别 client disconnect 保持无摘除、无强写；`8807d9a34` 补齐并对齐相关 bridge/truncation 契约。RED 是 turn 2 `connection refused` 的地址回显/副作用丢失、200 后 reader error 的无 event/副作用丢失，以及 client disconnect 的重复 error event；聚焦 bridge 与 unit-tag transport helper 测试均 GREEN。
- Task 15 Sol review round 1 full gate：最终 archive `8807d9a34` 在 `local-serv-ai` 的 `/tmp/sub2api-task15-r1-134c34a4b1ca4251b119b864ab79d38e` 完整 integration GREEN。根 `make test`（Vitest `204` 文件、`1549` 用例）、注入 Git `sh` PATH 后的 `make build`、两轮 generate/Ent-Wire 零 diff、unmerged/精确 marker/diff-check/migration 集合均通过；Go `1.26.5`、Docker `29.2.1`。远程 `CI=true GOFLAGS='-v'` 和三变量 temp 的完整 integration 退出 `0`，`TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` PASS（`4.92s`），`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS（`4.17s`）。日志为 `C:/Users/caiqy/AppData/Local/Temp/sub2api-task15-r1-134c34a4b1ca4251b119b864ab79d38e-integration.log`；archive/远程目录已清理，合法 skip 同 Task 15 report，无 `FAIL`。
- Task 15 Sol review round 2：`8b2c969dc` 闭合 Backup S3 凭证轮换后复用型 async image uploader 继续使用旧 endpoint/credentials 的实际行为 gap。RED 为实际 `BackupHandler.UpdateS3Config` 成功 mutation 后公开 `Resolver()` 未重建 uploader；修复仅调用既有 `imageStorage.Invalidate()`，使下一次 resolve 用新 endpoint/access key/secret 构建，nil receiver 安全。handler 聚焦测试及 image-storage/backup unit 聚焦测试均 GREEN。
- Task 15 Sol review round 2 full gate：archive `8b2c969dc` 在 `local-serv-ai` 的 `/tmp/sub2api-task15-r2-f831df6d0d98496996f5d9798349f714` 完整 integration GREEN。根 `make test`（Vitest `204` 文件、`1549` 用例）、注入 Git `sh` PATH 后的 `make build`、两轮 generate/Ent-Wire 零 diff、unmerged/精确 marker/diff-check/migration 集合均通过；VERSION `0.1.159.6`，Go `1.26.5`、Docker `29.2.1`。完整 integration 退出 `0`，`TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` PASS（`4.76s`），`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS（`4.42s`）。日志为 `C:/Users/caiqy/AppData/Local/Temp/sub2api-task15-r2-f831df6d0d98496996f5d9798349f714-integration.log`；archive/远程目录已清理，合法 skip 同 Task 15 report，无 `FAIL`。

### Task 16 独立 capability closure（OpenSpec 4.3）

- 初版审查起点为 `11889c61c`；review round 1 以 Task 15 round 2 证据记录 `9ba81a5b1` 为当前 HEAD。当前 agent 未暴露 CodeGraph 工具，改用精确 Git range、知识库能力入口和源码定义/调用点读取；未读取全量 plan，未执行远程操作、Task 17 或任何源码修复。TDD N/A：本任务只补充文档审查证据，不改变行为。
- 计数口径修正：`git diff --name-only 'v0.1.161^{}..v0.1.162^{}'` 为 `190` 条 raw changed files。下表 `41` 条（settings/runtime `9`、trusted proxy/client-IP `9`、Grok cache/sticky `9`、S3/image `14`）是按四个 brief capability row 选定、且经该 190 条集合核对的核心实现/测试证据路径；它不是预声明或机器完备的 file-level intersection universe，不能作为 path-level completeness/gap 证明。`protected=4`、`manual=0`、`gap=0` 仅定义在四个 capability row：每行都有自动测试和当前调用链证据；不由 41 条路径的数量推导。

| 能力 | 上游触及与选定核心证据路径 | 实际入口/调用链 | 自动或人工证据 | 结论 |
| --- | --- | --- | --- | --- |
| settings JSON migration/backfill、解析默认值、启动与热更新 | 9 条：`config.go`、`config_test.go`、`env_reachability_test.go`、`cmd/server/main.go`、`setting_service.go`、`setting_parse.go`、`setting_update.go`、`setting_service_update_test.go`、`admin/setting_handler_update.go` | `main.runMainServer -> config.LoadForBootstrap` 解析 env/default 并发布初始 snapshot；admin `UpdateSettingsRequest` 接收 trust/header 字段并映射给 service；`ProvideSettingService -> LoadForwardedClientIPSettings` 读取三键、缺 header JSON 或 mode-v2 时 `SetMultiple` 回填、再 `SetForwardedClientIPSettings`；`UpdateSettings -> refreshCachedSettings -> SetGatewayControlRuntime/SetForwardedClientIPSettings` 同步重建 scheduler/sticky 与 IP runtime snapshot。 | 本任务 `-tags=unit` 聚焦 GREEN：config 的 header snapshot、trusted proxy、image-storage env；service 的 forwarded migration/backfill/read-failure、settings refresh。Task 14 仅 merge/cleanup 的限定编译证据；Task 15 round 2 full gate/integration 的最终证据另见下。 | closed；不存在“只写 DB 不生效”的路径。 |
| trusted proxy / forwarded IP | 9 条：`pkg/ip/ip.go`、`pkg/ip/ip_test.go`、`server/http.go`、`server/http_ingress_test.go`、`middleware/api_key_auth.go`、`api_key_auth_test.go`、`session_binding.go`、`session_binding_test.go`、`server/router.go` | `ProvideRouter -> configureTrustedProxies -> Gin.SetTrustedProxies`：未配置或非法配置 fail-closed；`SetupRouter -> SessionBindingContext` 每请求 snapshot 设置；`APIKeyAuth -> GetSecurityClientIP` 在开关关闭时仅走 Gin trusted-proxy chain，开启时才走兼容 forwarded header。 | 本任务 `-tags=unit` 聚焦 GREEN：`TestConfigureTrustedProxies`、三项 API Key ACL trusted/forwarded 测试、两项 session snapshot/switch 测试。当前源码还显示 malformed DB header/read failure 会关闭 trust。Task 15 是 final-HEAD full regression/integration 证据，非 Task 14 静态证据。 | closed；ACL、会话绑定与审计共享同一请求 snapshot。 |
| Grok client-tool cache 与 session/platform Sticky | 9 条：`openai_gateway_grok.go`、`openai_gateway_grok_test.go`、`openai_gateway_grok_cache.go`、`openai_gateway_grok_cache_test.go`、`openai_gateway_grok_chat_bridge.go`、`openai_gateway_grok_chat_bridge_test.go`、`openai_gateway_messages.go`、`openai_ws_http_bridge.go`、`openai_ws_http_bridge_test.go` | HTTP `forwardGrokResponses -> resolveGrokCacheIdentity -> applyGrokResponsesCacheIdentity -> applyGrokFreeRequestToolCacheRoute`；WS bridge 走同一 identity/tool-route 后构建 Grok request。identity 由 API key、mapped model、stable session/content seed 隔离；runtime platform Sticky 由 settings snapshot 被 `OpenAIGatewayService`/scheduler 读取。 | 本任务 `-tags=unit` 聚焦 GREEN：stable append-only identity、known-free client-tool cache default、request opt-in override，及 `TestForwardGrokResponsesCodexAdditionalToolsUsesMixedCacheIntent` 的真实 forward body/cache identity/header 行为。Task 15 覆盖最终 HEAD 的 cache 提交、HTTP/WS bridge/terminal 及 failover 回归；Task 14 只确认冲突融合时没有丢失 sticky/owner/spool/redaction/usage 契约。 | closed；HTTP 与 WS 采用同一 tenant-isolated cache identity，且保留 platform sticky。 |
| S3 backup / image storage / async image path | 14 条：`image_storage_env_test.go`、`backup_service.go`、`backup_service_test.go`、`image_storage_settings.go`、`image_storage_settings_test.go`、`image_task.go`、`admin/backup_handler.go`、`image_task_handler.go`、`image_task_admin_toggle_test.go`、`routes/admin.go`、`routes/gateway.go`、`routes/gateway_test.go`、`service/wire.go`、`cmd/server/wire_gen.go` | Wire `ProvideBackupService.Start` 启动备份调度；`ProvideImageStorageSettingService` 将 config fallback 与 S3 factory 绑定；`ProvideImageTaskService -> Resolver -> resolve` 缓存有效 uploader，`ImageStorageSettings.Update -> Invalidate` 后下次请求重建；`AsyncImageHandler -> ImageTaskService.Complete -> ImageResultUploader.Rewrite` 再落 Redis。Task 15 round 2 后，`BackupHandler.UpdateS3Config -> BackupService.UpdateS3Config -> ImageStorageSettingService.Invalidate -> next Resolver` 也会重建复用 Backup S3 凭证的 uploader。admin S3/image-storage 与 gateway async routes 均已注册。 | Task 16 round 2 的 `-tags=unit` 聚焦命令实际执行 `TestImageTaskServiceCompleteOffloadsToStorage` 并 PASS；同命令还覆盖 env active、image setting 无重启 toggle/reuse backup credential，另有 `TestAsyncImageEnablesWithoutRestart` 的实际 admin toggle HTTP path。Task 15 round 2 的 `TestBackupHandlerUpdateS3ConfigRefreshesReusedImageStorage` 先缓存旧 uploader、经成功 handler mutation 轮换 endpoint/access key/secret、断言 factory 用新值第二次构建；`8b2c969dc` 修复后由 `9ba81a5b1` 记录 full gate/integration。 | closed as of `8b2c969dc`/`9ba81a5b1`；初版审查时此 shared Backup S3 rotation 路径尚有实际 gap，不能倒推为当时已闭合。 |

- Task 14 与 Task 15 的证据边界：Task 14 merge `8bda73544d6e26a323f101e5c68981634f0375ab`（及 cleanup `46eb292c04f14630dcfb31b28f6e83f541029d93`）只证明冲突融合、无 unmerged/marker、限定 Go 编译与 vue-tsc，明确未跑 full gate。Task 15 round 1 archive `8807d9a34` 证明当时的 full gate；随后 review 发现 shared Backup S3 rotation gap，只有 source fix `8b2c969dc` 和记录其 full gate/integration 的 `9ba81a5b1` 才构成当前最终 runtime 证据。Task 16 的上列源码/聚焦测试独立补足 capability-row closure，不把 Task 14 静态结论或 41 条 selected-path 数量误写为完整性证明。
- Task 16 review round 1 额外验证：config、server/middleware、service 聚焦命令都带 `-tags=unit` 并实际匹配；新增 async toggle 和 BackupHandler rotation 测试均实际匹配且通过。`git ls-files -u` 为 0、精确 tracked conflict marker 为 0、`git diff --check` exit `0`、`backend/cmd/server/VERSION` 精确为 `0.1.159.6`。结论：`DONE`，四个 capability row 的 `gap=0`；无 path-level completeness 声明。

## v0.1.163

- changed-files：Task 19 以 `v0.1.162^{}..v0.1.163^{}` 取上游 raw `171` 条，以 `02abe1574^1..02abe1574` 取 merge first-parent raw `171` 条；严格交集为 `170` 条。上游独有 `backend/cmd/server/VERSION`，merge 独有 `backend/internal/service/openai_account_scheduler_layered.go`；两者不混入交集。
- 冲突台账：14 个文本冲突的融合结论见 Task 17；Task 19 只复核当前能力证据，不重做 merge。
- 能力矩阵交集：reasoning policy、scheduler quota metadata/LastUsedAt/sticky/fallback/WaitPlan/DB recheck、shutdown/Cleanup/background drain、usage/billing/output estimate/cache/image、axios manifest/lockfile、migration 185/schema/runner 均有交集或明确标注的 stage/context evidence；`gap=0`。
- 聚焦测试：Task 19 的定向 service/repository/cmd/server/handler/admin Go 测试均 PASS；完整命令、路径集合、调用链和行级结论保留在未提交 `.superpowers/sdd/task-19-report.md`。
- 本地门禁：复用 Task 18 的 `make test`、`make build`、双 `make -C backend generate` 与 clean diff 证据；本轮 clean archive 从 `dbb18b7059a5378411dc8c51fde570003dd072ba` 执行 `pnpm --dir frontend install --frozen-lockfile` PASS，`pnpm list axios` 与实际 `require('axios/package.json').version` 均断言 `1.18.1`，同一安装态 `pnpm --dir frontend run build` PASS。Task 19 未重跑 full gate，也不将其替代能力审查。
- 远程门禁：复用 Task 18 remote integration 的两个 migration target PASS、唯一 migration 185/checksum 和 `SKIP=13` 分类；未执行远程操作。
- 放行结论：`DONE_WITH_CONCERNS`；六行业务能力矩阵 `protected=6/manual=0/gap=0/approved-removal=0`；reviewer provenance 是矩阵外已满足前置。`backend/cmd/server/VERSION` 保持 `0.1.159.6`。

## v0.1.164

- changed-files：`U=diff(v0.1.163^{}, v0.1.164^{})=202`，`M=diff(699459921^1, 699459921)=205`。`U-M={backend/cmd/server/VERSION}`，按约束保留 `0.1.159.6`；`M-U` 是四个本地回归测试的新增依赖参数适配，未改变运行时语义。
- 冲突台账：21 个文本冲突均人工融合；下表是已提交 ledger 的逐项最终决议，替代 ignored scratch 的引用。

| 路径 | 最终决议 |
| --- | --- |
| `backend/cmd/server/VERSION` | 保留本地 `0.1.159.6`，不随上游 tag 改写。 |
| `backend/cmd/server/wire_gen.go` | 从已融合的 Wire source 确定性生成，保留本地生命周期依赖和 composite/Ollama providers。 |
| `backend/ent/client.go` | 从已融合 schema 生成，保留既有 Ent clients 和 `CompositeModelRoute` client。 |
| `backend/internal/handler/admin/setting_handler.go` | 保留原 settings/payment 字段，并返回 Alipay mobile deep-link 字段。 |
| `backend/internal/handler/admin/setting_handler_update.go` | 保留原有更新校验，贯通 Alipay mobile deep-link 请求、更新和响应。 |
| `backend/internal/handler/content_moderation_helper.go` | moderation 使用 client public model、resolved provider；Round 1 同时让 usage fields 保留 client、concrete mapped、upstream 的完整链。 |
| `backend/internal/handler/gateway_handler.go` | 保留 sticky/fallback 和异步 usage；Round 1 以 effective group 重算 composite decision，并在失败 usage 中快照三段模型。 |
| `backend/internal/handler/gateway_handler_chat_completions.go` | 保留 session/sticky 语义，按 resolved target 选择与计费。 |
| `backend/internal/handler/gateway_handler_responses.go` | 保留 image/usage 行为，先解析 target platform 再分类和调度。 |
| `backend/internal/handler/openai_embeddings.go` | 保留 usage detail，并拒绝非 OpenAI composite target。 |
| `backend/internal/handler/openai_gateway_handler.go` | 保留异步 usage context；Round 1 修复 Messages 授权、WS 首帧 resolver/改写，并向 Ops 同步精确 upstream model。 |
| `backend/internal/handler/openai_images.go` | 保留 multipart/detail；Round 1 以 concrete upstream model 重建 multipart，并在失败 usage 快照模型和请求详情。 |
| `backend/internal/server/api_contract_test.go` | 保留原 contract fixtures，加入 Alipay mobile deep-link 字段断言。 |
| `backend/internal/server/http.go` | 传递 composite resolver，不移除既有 server setup。 |
| `backend/internal/server/router.go` | 在所有 router 层传递 composite resolver。 |
| `backend/internal/server/routes/gateway.go` | 保留 `UsageDetailCapture` 和 Grok guards；加入 model-aware composite routing，Round 1 在 body 改写前固定原始 client request snapshot。 |
| `backend/internal/service/gateway_scheduling.go` | 保留 advanced/layered sticky、fallback/WaitPlan、DB fresh recheck，并在其前解析 concrete composite platform/model。 |
| `backend/internal/service/openai_gateway_service.go` | 同时保留 scheduler state 和 proxy stream quarantine state。 |
| `frontend/src/components/account/EditAccountModal.vue` | 保留既有账号控件；Round 1 仅在弹窗打开或 account ID 改变时回填表单，避免 Ollama runtime 刷新覆盖未保存编辑。 |
| `frontend/src/types/index.ts` | 保留既有 contracts，补齐 composite/Ollama Cloud 类型并兼容 Alipay 字段。 |
| `frontend/src/views/admin/SettingsView.vue` | 保留本地 scheduler/runtime settings，加入 Ollama global settings 和 Alipay deep-link switch。 |

- Round 1 额外修复：group copy 在 transaction 内完成 source 校验、绑定替换和 outbox 写入；Create/Update account response 解析共享 Ollama 状态；copy source 独立加载全部 active/inactive groups；所有 failed usage input 在 `Submit` 前快照，避免 delayed worker 读取复用的 Gin request；Ops request drilldown 返回 `requested_model`、`model`、`upstream_model` 三段审计字段。
- 能力矩阵交集：composite 入口为 `ProvideRouter -> SetupRouter -> RegisterGatewayRoutes -> composite target middleware -> GatewayService.resolvePlatform`，未绕过本地调度与粘性路径；Ollama Cloud 由 account/settings admin API、service/repository 与 Wire cleanup 接入；Grok 402 cooldown 位于 `openai_gateway_grok.go` 调度错误处理；Alipay mobile deep link 贯通 settings、provider、支付 UI。
- migration：完整保留 `172_video_per_second_billing_metadata.sql`、`172_composite_model_routes.sql`、`186_alipay_mobile_precreate_deep_link.sql`、`186_group_auth_cache_image_generation.sql`；未重命名或改写发布 migration。
- 聚焦验证：Round 1 实际运行并通过三段模型 handler/route/repository 测试、失败 usage unit 测试、Ollama shutdown ordering 测试，以及 proxy quarantine 正反向/threshold/TTL 命名测试。先前的 `go test ./internal/service -run '^$' -count=1` 仅编译并执行零测试，不作为行为证据；`^TestOpenAIStreaming.*QuarantineProxy$` 只命中否定用例，也不再使用。完整命令与测试边界记录在本次 repair report。
- 本地门禁：本任务按边界未运行 Task 21 full gate；`VERSION=0.1.159.6`，`openai-first-token-timeout` 在 `backend`、`frontend`、`deploy` 中为 0 命中。
- 远程门禁：未执行远程操作。
- 放行结论：`DONE_WITH_CONCERNS`。merge 为 `6994599211d3714e30b67cc61ef0834a94c34610`，第一父 `07167bbfa44ecd702cf32268ad98eabb0dbb6c65`，第二父 `cd8bb98c44303b2c8f04c0da340447c992f0cb7d`；Round 1 的聚焦修复已记录，剩余边界是 Task 21 full gate 未执行。

### Task 20 Round 2 Review Closure

- 复审输入：`task-20-review-2.md` 的 6 个 blocker 与 4 个 non-blocking concern；本轮不改变 merge commit、migration、VERSION、Task 21/22、Docker、远程或发布边界。
- repair 提交链：`48e2d4a0b`（Round 1 源码/测试）、`88aeed4b0`（Round 1 ledger）、`96455c43b965a1736e105ff934e16681689ddeb7`（Round 1 composite mapping recovery）、`a9292253faca30a3977d6a68c40c9785ad2ddb09`（Round 2 源码/测试）。
- 范围口径：`699459921..48e2d4a0b` 为 53 个 `backend/`、`frontend/` 源码/测试路径与 1 个正式 ledger；加入 recovery 和 Round 2 后，`699459921..a9292253f` 为 61 个源码/测试路径与同一 1 个 ledger。`git diff --check 699459921..a9292253f` 通过。
- blocker 闭合：
  - Messages 与 fallback 统一采用 effective group、`messages` endpoint decision、重写后的 body/model、channel mapping 与 group concurrency；不再用原 composite group 或 `endpoint=any`。
  - `channel_mapped` 计费始终先检查 channel alias 显式价格，缺失时才回退 concrete model。
  - Ollama Cloud Create/Update commit 后的 usage enrichment 改为 warning telemetry，响应仍保持 mutation 成功。
  - Responses WebSocket 为每个后续 `response.create` 重用 composite resolver、重写 payload model，并把 decision 传入 ingress 与 passthrough 路径。
  - composite route 的 public/upstream model 都限制为 100 字符，和 usage/Ops 存储契约一致；未改写已发布 migration。
  - 正式台账现包含 Round 1/2 repair SHA、路径范围、验证和 advisory，不将 Task 21 门禁写成已完成。
- concern 闭合：failed usage 传播 channel usage fields，Images 成功记录基于 `routingModel` 构造三段链，Ops 列表/详情按 client、mapped、upstream 顺序展示；非 multipart 原始 body snapshot 限制为 5 MiB，multipart 保留 handler 的 metadata-only snapshot；copy source loader 以 generation/in-flight 拒绝 stale response 并报告当前失败；group-copy service 以 `-tags=unit` fresh 运行。
- Round 2 聚焦验证通过：composite 长度/alias pricing/failed usage unit tests、Ollama mutation test、multipart snapshot tests、每帧 WS route test、group-copy unit tests、GroupsView 与 OpsErrorLogTable Vitest、`vue-tsc --noEmit`、`pnpm build` 和相关 Go package compile。前端 build 仅保留既有 Browserslist、dynamic-import 和 bundle-size warning。
- advisory：Docker integration 的两个 group-copy rollback case 因 `docker is not available` 为 0 cases；`-race` 因 `CGO_ENABLED=0` 无法启动。两者均非 PASS 结论。完整命令和 RED/GREEN 原文位于 ignored `.superpowers/sdd/task-20-report.md`。
- 本轮结论：`DONE_WITH_CONCERNS`。Task 21 full gate、CGO race、Docker integration、Task 22 fresh review、`v0.1.165`、remote/push/tag/release/deploy 继续封闭，未执行。

### Task 20 Extra Repair 1/1 Closure

- 输入：fresh Sol final review 的 6 个 Important 与 1 个 Minor；base 为 `09db65607052778852cc44c0d7252e9bf1102fb8`。
- 可验证 repair 链：`48e2d4a0b`（Round 1 源码/测试）、`88aeed4b0`（Round 1 ledger）、`96455c43b`（composite mapping recovery）、`a9292253f`（Round 2 源码/测试）、`09db65607`（Round 2 review ledger）、`1e7b8af75`（Extra Repair 1/1 源码/测试）。最终 ledger 是包含本文件本段的紧随其后的 `docs: record v0.1.164 extra review repair` 提交；该提交不能在自身内容中引用其 SHA。
- 精确范围：`git diff --name-only 699459921..1e7b8af75` 为 `66` 个路径，其中 `backend/` 或 `frontend/` 源码/测试路径 `65` 个、正式 ledger 路径 `1` 个；`git diff --name-only 09db65607..1e7b8af75` 为 `17` 个路径，全部是源码/测试，正式 ledger 为 `0` 个。包含本段 ledger 提交后，累计 unique-path 计数仍为 `66 = 65` 个源码/测试路径 + `1` 个正式 ledger 路径。
- 边界：`backend/cmd/server/VERSION` 保持 `0.1.159.6`；未改 migration，未恢复 `openai-first-token-timeout`；未提交 `.comet/current-change.json`、`paseo.json`、`.superpowers/sdd/progress.md` 或 `openspec/changes/staged-merge-upstream-v0-1-165/.comet/subagent-progress.md`。

#### 原始 10 个 Blocker 最终状态

| ID | 最终状态 | 最终证据或修复提交 |
| --- | --- | --- |
| 1. Messages/count_tokens effective group | CLOSED | `1e7b8af75`：Messages、两条 count_tokens 和 Gateway 路由均使用最终 effective group、精确 endpoint、重建 body 与 channel mapping。 |
| 2. Responses WS 首帧 composite alias | CLOSED | `48e2d4a0b` 已建立首帧 route；`1e7b8af75` 复核 first/later frame。 |
| 3. multipart image concrete upstream model | CLOSED | `48e2d4a0b`，既有 multipart 回归保持在本轮限定矩阵边界内。 |
| 4. async image composite context | CLOSED | `48e2d4a0b`，未重设计已关闭调用链。 |
| 5. usage worker context ownership | CLOSED | `48e2d4a0b`，未重设计已关闭快照路径。 |
| 6. fallback 后 stale composite decision | CLOSED | `1e7b8af75`：G1 body 恢复 public model 后按 G2 的 `messages` route 重算 decision。 |
| 7. composite alias pricing | CLOSED | `1e7b8af75`：非 composite 未映射请求优先 concrete account model，composite explicit alias price 保持优先。 |
| 8. group-copy transactional validation | CLOSED | `a9292253f`；Docker rollback 仍为 0-case advisory。 |
| 9. Ollama form/runtime state | CLOSED | `a9292253f`，未改动已关闭前端路径。 |
| 10. formal ledger completeness | CLOSED | 本段在 tracked ledger 中写入完整状态、SHA、范围与证据；最终 ledger 由包含本文件的本次 docs 提交识别。 |

#### 原始 4 个 Concern 最终状态

| ID | 最终状态 | 最终证据或修复提交 |
| --- | --- | --- |
| 1. client/mapped/upstream 三段模型 | CLOSED | `1e7b8af75`：普通 failed-usage 分支传递 `ChannelUsageFields`；Images Ops 使用 concrete channel/routing model。 |
| 2. Ollama create/update shared state | CLOSED | `a9292253f`。 |
| 3. copy-source request ownership | CLOSED | `a9292253f`。 |
| 4. 证据命令语义与 producer order | CLOSED | `88aeed4b0`、`09db65607` 与本段明确 compile-only、Docker 0 cases、CGO race unavailable 和 Task 21 closed boundary。 |

#### Extra Repair 1/1 Finding 最终状态

| Finding | 最终状态 | 修复 |
| --- | --- | --- |
| Important 1: effective-group authority | CLOSED | `ResolveEffectiveGatewayGroup` 与 handler effective route 使授权、channel、并发、billing 和 scheduler 共享最终 group。 |
| Important 2: G1 stale body/secondary fallback billing | CLOSED | secondary fallback 清除旧 decision、还原 public model 后按 G2 解析；billing 移到最终 fallback route 之后；G2 接受 concrete 或 composite。 |
| Important 3: non-composite `channel_mapped` pricing | CLOSED | 未发生 channel mapping 时不让 `OriginalModel` 抢占 concrete price；composite explicit alias price 不回归。 |
| Important 4: later Responses WS frames | CLOSED | later frame 路由后再次 account-map；cross-provider/no-route/resolver-error 都关闭连接；passthrough 的 `RewriteRequest` 不再依赖 `BeforeRequest`。 |
| Important 5: runtime 100-character boundary | CLOSED | resolver 在 detector/client 和 legacy route public/upstream 值进入 billing/audit 前拒绝超过 100 字符的模型。 |
| Important 6: self-contained formal ledger | CLOSED | 本段。 |
| Minor: three-stage model audit | CLOSED | 两个普通 Gateway failed-usage 分支传递 channel fields；Images Ops model 使用 concrete channel/routing stage。 |

#### 本轮实际 RED 与 GREEN

- RED：`go test ./internal/service -run '^TestCompositeRouteResolverRejectsRuntimeModelsBeyondStorageContract$' -count=1 -v` 失败，因为 101 字符 detector model 返回 nil error。
- RED：`go test ./internal/service -run '^TestOpenAIGatewayServiceRecordUsage_ChannelMappedDoesNotOverrideBillingModelWhenUnmapped$' -count=1 -v` 失败，因为 original price 覆盖 concrete account-mapped model。
- RED：`go test ./internal/handler -run '^(TestOpenAIResponsesWebSocketAppliesAccountMappingAfterLaterCompositeRoute|TestOpenAIResponsesWebSocketRejectsLaterCrossProviderCompositeRoute)$' -count=1 -v` 在 later-frame ordering/mapping/provider-affinity 修复前失败。
- RED：`go test ./internal/handler -run '^TestOpenAIGatewayHandler_MessagesUsesEffectiveClaudeCodeFallbackGroup$' -count=1 -v` 返回 `403`，证明 `Messages` 在回退前仍检查原 ClaudeCode-only group。
- RED：`go test ./internal/service -run '^TestPassthroughLifecycle_AppliesAccountMappingAfterLaterRequestRewrite$' -count=1 -v` 初始把 `public-model` 直通，暴露 `RewriteRequest` 被 `BeforeRequest` 隐式门控。

```powershell
go test ./internal/handler -run '^(TestGatewayHandlerResolveStickyRouteRecomputesCompositeFallbackDecision|TestGatewayHandlerResolveEffectiveGatewayRouteUsesConcreteClaudeCodeFallback|TestOpenAIGatewayHandler_MessagesUsesEffectiveClaudeCodeFallbackGroup|TestOpenAIGatewayHandler_CountTokensUsesEffectiveClaudeCodeFallbackGroup|TestOpenAIResponsesWebSocketResolvesCompositeExplicitAliasOnFirstFrame|TestOpenAIResponsesWebSocketResolvesCompositeExplicitAliasOnEveryFrame|TestOpenAIResponsesWebSocketAppliesAccountMappingAfterLaterCompositeRoute|TestOpenAIResponsesWebSocketRejectsLaterCrossProviderCompositeRoute|TestOpenAIResponsesWebSocketRejectsLaterCompositeNoRoute|TestOpenAIResponsesWebSocketClosesOnCompositeResolverError)$' -count=1 -v
go test ./internal/service -run '^(TestCompositeRouteResolverRejectsRuntimeModelsBeyondStorageContract|TestOpenAIGatewayServiceRecordUsage_ChannelMappedDoesNotOverrideBillingModelWhenUnmapped|TestOpenAIGatewayServiceRecordUsage_CompositePublicAliasPricing|TestPassthroughLifecycle_AppliesAccountMappingAfterLaterRequestRewrite)$' -count=1 -v
go test ./internal/server/routes -run '^TestCompositeTargetPlatformMiddlewareRejectsOversizedRuntimeRouteModels$' -count=1 -v
go test ./internal/handler ./internal/service ./internal/server/routes -run '^$' -count=1
git diff --check 09db65607..1e7b8af75
```

- GREEN：上述全部命令退出 `0`。最后一条 compile-only 命令只证明包可编译，不替代前三条行为证据。
- Docker rollback integration：Docker 不可用，执行 `0` cases，非 PASS。CGO race：`CGO_ENABLED=0`，`-race` 无法启动，非 PASS。
- Task 21 full gate、Task 22、`v0.1.165`、Docker/远程 integration、push、tag、release、deploy 仍 CLOSED，均未执行。

### Task 21 Full Gate Closure

- 起始 branch/HEAD 已按要求核验为 `feature/20260726/staged-merge-upstream-v0-1-165` / `489fa1025de715d85c6368cc96680d8de1801eec`；CodeGraph MCP 在本 agent 不可用，改以 `SetupRouter -> RegisterGatewayRoutes -> composite middleware -> effective group/route -> resolvePlatform -> scheduler Select` 的精确源码链与命名测试记录 composite、advanced/layered、Grok sticky、fallback/WaitPlan、DB fresh recheck 证据。
- 最终 focused matrix 均 GREEN：handler 9、routes 9、service scheduler 15、Ollama/admin/Alipay 8、migration package 20；本机 migration integration 因 Docker unavailable guard 未执行 cases，不作为 PASS。
- 最终 `make test` exit 0：后端 default/unit、`golangci-lint` `0 issues` 和前端 lint/typecheck/Vitest `213` files、`1613` tests 全部通过。用户接受的 `TestPassthroughLifecycle_LeaseLossSendsRetryClose` EOF/1013 例外未发生，未以例外掩盖失败。
- 初始 `make SHELL=D:/scoop/shims/bash.exe build` 的 `CreateProcess(... sh ...) failed` 已由 controller 定位为 PATH 中 Gow GNU Make 3.81 对 `$(shell ./scripts/resolve-version.sh)` 的错误继续，不是未闭合 build concern。最终有效命令 `make "VERSION=0.1.159.6" "SHELL=D:/scoop/shims/bash.exe" build` exit 0；backend 实际含 `-X main.Version=0.1.159.6`，frontend Vite transformed `1019` modules，且 `rg -a -o -m 1 "0\.1\.159\.6" backend/bin/server` 输出 `0.1.159.6`。主树 generate 命中 `user-mapped section` 后，短 detached worktree `D:\w21` 在同一提交上连续两次 generate/Ent-Wire diff 均为零并已清理；这是唯一残余风险。静态检查无 unmerged/精确 marker，VERSION `0.1.159.6`，禁用 timeout marker 为 0。
- 远程 archive HEAD 为 `aa7b67369bb0b0ffa4f101b6b2b831916288464b`；`local-serv-ai` 唯一目录 nonce `8875a92987b243ff886c82498c00e0ca` 预检 Go `1.26.5`/Docker 后，以隔离三临时变量运行 `CI=true GOFLAGS=-v go test -tags=integration ./...`，exit 0。最终日志为 `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-v0.1.164-8875a92987b243ff886c82498c00e0ca-integration-final.log`；remote 目录和本地 tar 均清理。
- 远程 `TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS（4.62s）。该测试分别验证 `172_video_per_second_billing_metadata.sql`、`172_composite_model_routes.sql`、`186_alipay_mobile_precreate_deep_link.sql`、`186_group_auth_cache_image_generation.sql`，并断言重复 apply 无新增 record、checksum 不变。
- 远程完整命令 exit 0、无 `FAIL`；`--- SKIP:` 精确为 13（12 top-level + 嵌套 `TestConcurrencyCacheSuite/TestGetAccountsLoadBatch` 1）。分类为 DingTalk disabled sentinel `TestDingTalkOAuthStart_Disabled` 1、未配置 external TLS capture `TestDialerAgainstCaptureServer` 1、既有 CI TODO concurrency cache 1、未设置 `PROMPT_AUDIT_TEST_REDIS_ADDR` 的 config CAS/payload store/runtime aggregate 3、未设置 `PROMPT_AUDIT_TEST_POSTGRES_DSN` 的 prompt-audit migration/database/repository/service 六项 6、未设置 `OPENAI_API_KEY` 的 external token comparison 1；均非本阶段能力失败。
- 临时 `6489a88b6 fix: preserve local behavior after v0.1.164` 的无效 `body = nil` 被 lint 识别为 ineffectual，已由 `aa7b67369 Revert "fix: preserve local behavior after v0.1.164"` 撤回；最终无该源码净变更。完整命令、失败根因、skip 分类和清理证据见 ignored `.superpowers/sdd/task-21-report.md`。

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
- 以下为 Task 5 当时状态（历史快照；不代表当前阶段状态）。
- `v0.1.160`：证据已补齐，等待独立复审；133 paths、9 个冲突、12 个交集路径及 Task 9 门禁证据见 Task 10。
- `v0.1.161`：实现门禁已闭合，docs/evidence 修正待 Sol 复审。257 paths、22 个精确交集路径，`protected=21`、`manual=1`、`gap=0`；跨路径补充回归不计入该统计。Task 14/v0.1.162 在复审 PASS 前封闭。
- `v0.1.162`：待执行。
- `v0.1.163`：待执行。
- `v0.1.164`：待执行。
- `v0.1.165`：待执行。

## 阻塞与残余风险

- 历史阻塞（已解决，归档 HEAD `c8e0110a9a2354453753db9c4acae0ed7570458d`）：原 Make/PowerShell/Go 预检依次暴露 GNU Make 缺失、Go `1.26.1` 低于 directive、PowerShell 入口缺失；后续规范已改为 Linux 原生 gate，vfox Go `1.26.5` 和 Docker `29.2.1` 已在新 nonce 中验证。
- 历史阻塞（已解决，规范 HEAD `849f956992178e25ab2074e1e4cc596d29f8834f`）：首次原生 full RED 暴露 handler body spool/panic、repository usage-detail transaction/retention 与 hidden UI resource ordering 断言；后续测试修复及 HEAD `99cb81de306cb0e8ea811387e362e4d601f6f4b0` 的新 nonce focused/full GREEN 已解决。
- Task 5 当时状态（历史快照）：阶段 0 本地与远程门禁通过；七项非 migration/repository target skip 是已分类的基线风险。Task 5 为 `DONE_WITH_CONCERNS`：HEAD 生成内容经 detached worktree 双轮验证稳定，但受监视 Windows 工作区的随机文件映射风险仍未定因，隔离门禁与工作区门禁的等价性待 reviewer 判断。`v0.1.160` 已闭合；`v0.1.161` 实现门禁已闭合但 docs/evidence 修正待 Sol 复审；`v0.1.162` 至 `v0.1.165` 尚未开始，且 Task 14/v0.1.162 须先获该复审 PASS。

## Task 11 第 1 轮复审修复

- 修复提交：`1b80f95c9 fix: correct v0.1.161 conflict resolutions`；只包含 reviewer 指出的六项冲突融合回归及其聚焦测试，未改写 merge `f2158292c` 或先前 ledger。
- RED：`go test ./internal/handler/admin -run '^TestUpdateSettings(EnableStepUp|DisableStepUp)' -count=1` 显示五项 step-up 转换错误放行；新增 Grok owner-binding 测试显示预期 `404`、实际 `200`；新增 Ops 测试显示持久化后 snapshot 仍为启用；`CreateAccountModal.spec.ts` 缺失 quota 字段；`SettingsView.spec.ts` 找不到 billing probe 卡；新增示例 YAML 解析测试报第 178 行 tab 缩进错误。
- GREEN：step-up/Ops 聚焦 Go 测试、Grok status/owner-binding 聚焦 Go 测试、`TestExampleConfigGatewayBodyLimits`、`CreateAccountModal.spec.ts`、`SettingsView.spec.ts` 和 `pnpm exec vue-tsc --noEmit` 均通过。
- 修复内容：恢复 step-up 启用前管理员会话/TOTP 校验与关闭时 `EnforceStepUpAlways`，并恢复 SettingsView 的 TOTP 重试；修正 Grok owner account 赋值作用域；API Key 创建统一进入配额/临时不可调度 helper 并保留 billing probe、expiry；恢复 billing probe 设置卡；持久化后发布 Ops monitoring 原子 snapshot；YAML 改为空格并把非流响应读取上限恢复为 128 MiB，同时保留 `text_max_body_size`。
- Task 12 风险：本轮刻意未运行 `make test`、`make build`、generate 或 integration；全量门禁、迁移/集成、跨端 step-up 会话和完整媒体/调度路径仍须在 Task 12 执行。

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
| API Key group ID 解析 | -- | `backend/internal/handler/admin/user_handler_list_apikey_group_test.go` | -- | `backend/internal/service/admin_group.go` | `backend/internal/repository/group_repo.go` | `backend/internal/repository/group_repo.go` | `protected`: `go -C backend test ./internal/handler/admin -run '^TestAdminUserList_ParsesAPIKeyGroupID$' -count=1`（exit `0`，PASS） |
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
- Task 9 full gate、回归修复及其复核证据已在第 1833-1875 行归档；`v0.1.161` 仍未开始合入或放行。

## Task 9：v0.1.160 回归保护与 full 门禁（2026-07-27）

- 状态：`DONE_WITH_CONCERNS`。起点 `d130c6754`；提交 `3de7191e3 test: cover staged migration upgrades`、`a719a8e6c fix: preserve local behavior after v0.1.160`、`31b132689 fix: preserve local behavior after v0.1.160`。未 push、tag、release 或 deploy；未触碰协调者 plan、tasks、progress 或根临时文件。
- Migration：新增 `TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages`，固定完整 filename 集合覆盖本地 172/181、当前上游 181/182 和未来 183-190（含 `190_add_users_email_alias_dedup_index_notx.sql`）。本机 Docker 不可用导致 integration harness skip，不记 GREEN。首个远程 archive (`3de7191e3`) 的整套 integration 因 handler 编译 RED 退出 `1`，但目标新测试已 `PASS (4.32s)`；最终 archive 则 `PASS (4.76s)` 且未 skip，验证 staged upgrade、记录数/checksum 幂等和 filename runner 行为。
- 真实 RED 1：远程 full integration 的 handler build 缺少 `OpenAIGatewayHandler.checkContentModeration`，并带入未调用且引用不存在 `releaseAccountSlot`/`closeOpenAIWSFailoverExhausted` 的 WS closure。恢复 OpenAI moderation 委派并删除死 closure；既有 image moderation 定向测试和 `go -C backend test -tags=unit ./internal/handler -count=1` 均 GREEN。
- 真实 RED 2：`make -C backend generate` 退出 `2`，Wire 报 `PromptAdminService` 无 provider。`securityaudit.ProviderSet` 增加 `wire.Bind(new(PromptAdminService), new(*PromptService))`；随后每轮 generate 和生成 diff 均退出 `0`。
- 能力矩阵聚焦均退出 `0`：scheduler priority、Grok/platform Sticky 两项、RequirePrivacySet、required image capability；Prompt/Security Audit 还覆盖 13 个 gateway 路径、3 个 blocking 子例和两项 admin route 断言。
- 最终本地 full gate：在 PowerShell 进程补 Git `sh` PATH 后，`make test`、`make build` 均退出 `0`。Vitest 为 `199 passed` files / `1521 passed` tests；backend VERSION `0.1.159.6`，Vite `1005 modules transformed`。最终两轮 `make -C backend generate` 与每轮 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 均退出 `0`、无 diff；本次未触发 Windows `user-mapped section` 锁，无需 detached worktree 备用流程。
- 静态：`git diff --check`、cached diff check、未合并路径检查、精确 conflict marker 扫描均退出 `0`；无未合并路径或 conflict marker。
- 远程 full integration：最终 archive 为已提交 HEAD `31b132689`；tar `C:/Users/caiqy/AppData/Local/Temp/sub2api-task-9-d70a45257d0f46c0911f6e17747ab6aa.tar`（`50,851,840` bytes，SHA-256 `0A6133485F5747EFC1DE77F349D3F16825CF0C0E9572BFC95580B85088A61235`）。唯一远程目录 `/tmp/sub2api-task-9-d70a45257d0f46c0911f6e17747ab6aa` 的创建、上传、Go `1.26.5`/Docker `29.2.1` 预检、带 `.test-tmp` 及 `TMPDIR`/`TMP`/`TEMP` 的 `CI=true GOFLAGS='-v' go test -tags=integration ./...`、下载、清理均为 `success=true, exit_code=0`。日志保存于 `C:/Users/caiqy/AppData/Local/Temp/sub2api-task-9-d70a45257d0f46c0911f6e17747ab6aa-integration.log`；remote `rm -rf && test ! -e` 和 local tar 删除均完成。
- 顾虑：远程仍有 DingTalk、TLS/JA3、Prompt Audit Redis/PostgreSQL 条件和无 OpenAI key 的既有环境型 skip；新 migration 目标没有 skip。image moderation 的 worker panic 已由 `fd909c5be` 修复为测试 fixture/lifecycle 问题，替代本行原有顾虑；Browserslist、动态 import 与 `670.83 kB` chunk advisory 未消除。历史 Windows 文件锁根因仍未确定。

## Task 10：v0.1.160 能力矩阵与证据封闭（2026-07-27）

- 状态：`DONE_WITH_CONCERNS`。本段是已有实现、测试和远程门禁证据的文档封闭；不改行为代码、不构造 TDD RED，且未重跑 Task 9 的 full test/build/generate/integration。Task 11 状态为：证据已补齐，等待独立复审。
- 只读 Git 复核：merge 为 `e04cb1aa2c2554a04bec55f9b4393d3efd2eb693`（`merge: upstream v0.1.160`），第一父 `d3e0c596ebff2298d07a3f4f336c16aa653cb840`，精确第二父及 `v0.1.160^{}` 为 `8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62`。`backend/cmd/server/VERSION` 仍为 `0.1.159.6`。
- changed-files：`git diff --name-only 'v0.1.159^{}..v0.1.160^{}'` 为 133 个精确路径，完整逐项原文保留在本台账 `TASK6:v0.1.160:raw`（第 395-531 行），未以 merge commit 的较小文件集替代 tag diff。
- 阶段 0 能力矩阵实际交集是 12 个精确路径：`backend/cmd/server/wire_gen.go`、`backend/internal/handler/batch_image_handler.go`、`backend/internal/handler/content_moderation_helper.go`、`backend/internal/handler/openai_images.go`、`backend/internal/pkg/xai/billing.go`、`backend/internal/repository/scheduler_cache.go`、`backend/internal/service/admin_account_upstream_billing_probe_test.go`、`backend/internal/service/grok_media.go`、`backend/internal/service/openai_gateway_scheduling.go`、`backend/migrations/181_prompt_audit.sql`、`frontend/package.json`、`frontend/src/i18n/locales/en/common.ts`。这 12 个路径映射为 14 个矩阵能力结论：`protected=13`、`manual=1`、`gap=0`。

| 能力交集 | 变更路径 | 结论与证据 |
| --- | --- | --- |
| advanced/layered scheduler | `scheduler_cache.go`、`openai_gateway_scheduling.go` | `protected`：`TestLayered_PriorityDeterminism`。 |
| fallback/WaitPlan | `scheduler_cache.go`、`openai_gateway_scheduling.go` | `protected`：`TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount`。 |
| DB recheck | `scheduler_cache.go` | `protected`：`TestLayered_GroupedAccountPassesDBFreshRecheck`。 |
| Grok/platform Sticky 与 media | `grok_media.go` | `protected`：`TestLayered_SessionStickyPreservesGrokBinding`、`TestGatewayService_SelectAccountForModelWithPlatform_StickyDisabledBypassesStickyReadAndWrite`；合并静态审查确认 media eligibility、spool/snapshot 脱敏、sticky 和官方 `url` 输出同存。 |
| privacy | `content_moderation_helper.go` | `protected`：`TestLayered_PreviousResponseStickyHonorsRequirePrivacySet`；`RequirePrivacySet` 仍是调度约束，未被 full prompt audit 存储语义替代。 |
| image capability | `openai_images.go` | `protected`：`TestLayered_RequiredImageCapabilityFiltersUnsupportedAccounts`。 |
| async images/object storage | `batch_image_handler.go` | `protected`：`TestImageTaskServiceCompleteOffloadsToStorage`。 |
| image/video billing | `billing.go` | `protected`：`TestCalculateImageCost`、`TestCalculateVideoCostUsesSeparateConfig`。 |
| upstream multiplier | `admin_account_upstream_billing_probe_test.go` | `protected`：`TestOpenAIFreshUpstreamBillingRateRecomputesPeakAtSelectionTime`。 |
| English i18n | `en/common.ts` | `protected`：`pnpm --dir frontend run test:run -- src/i18n/__tests__/localeKeysExist.spec.ts`，并由 Task 9 `make test` 的 199 files / 1521 tests 通过。 |
| Ent/Wire | `wire_gen.go` | `protected`：`go -C backend test ./cmd/server`；Task 9 两轮 `make -C backend generate` 及各轮 generated diff 均为 0。 |
| dependencies | `frontend/package.json` | `manual`：`go -C backend mod verify` 退出 `0`（`all modules verified`）；`pnpm --dir frontend install --frozen-lockfile` 退出 `0`（`Lockfile is up to date, resolution step is skipped`、`Already up to date`）。命令前后 lockfile 和其他 tracked 文件无变更；`git diff --exit-code -- backend/go.mod backend/go.sum frontend/package.json frontend/pnpm-lock.yaml` 退出 `0`。 |
| migrations | `181_prompt_audit.sql` | `protected`：远程 `TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS、无 SKIP；full integration 的 `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` PASS。 |
| local test gates | `frontend/package.json` | `protected`：Task 9 `make test` GREEN。 |

- 9 个冲突文件与融合结论已在本台账第 1818-1825 行逐项记录并复核：`VERSION` 保持 `0.1.159.6`；Grok handler/test 保留 spooling、脱敏、sticky 与 media eligibility；OpenAI gateway 保留 group permission 并采用显式 image intent；OpenAI images 先 moderation 后 audit；Grok service/test 兼容 `image_url`/`mask_image_url` 和 `reference_images`，规范化官方 `url`；scheduler tests 同时覆盖 sticky 与 media eligibility；OpenSpec config 同存中文规则和上游注释。
- security audit/full prompt：`181_prompt_audit.sql` 创建 redacted metadata，`182_prompt_audit_full_prompt.sql` 仅为 event 增加 `full_prompt`；`FullPromptFromScanText` 只写 audit event。该留存边界与 `RequirePrivacySet` 调度隐私开关独立，已由上述 privacy/audit 证据保护。
- `openai-first-token-timeout` 是唯一已审批移除项，不属于 12 路径实际交集：`manual` 静态证据为 merge 后 runtime 与测试中该 branch、constant、test 均不存在，且第 1821 行记录了保留该删除的冲突决议；它不计为 `gap`。
- 补充证据（不计入阶段 0 14 项统计）：image moderation/security audit 由 `TestSecurityAuditBlockingFailuresLeaveAllDownstreamCountersAtZero`（3 子例）和 `TestPromptAuditGatePrecedesAccountBillingAndUpstreamSideEffects`（13 gateway 路径）保护；调用顺序为 moderation 后 audit，二者通过才转发。
- migrations：本地 `181_group_duplicate_operation_id.sql`、上游 `181_prompt_audit.sql`、`182_prompt_audit_full_prompt.sql` 三个完整 filename 均存在，按字典序先建 event 再 ALTER。目标 staged-upgrade 测试在 review-fix archive `fd909c5be1eb628a551d11128a4329ade57070f5` PASS（`4.31s`、无 SKIP）；mutation RED 发生在修复后，临时删除的是上游 `172_composite_model_routes.sql`，使固定集合由 12 变 11、退出 `1`，与本地 `172_video_per_second_billing_metadata.sql` 无关，仅证明测试敏感性，不冒充实现前 RED。
- Task 9 完整提交链与边界：
  - `3de7191e391c0fc3edffb4de11d31a3eee9c87f5 test: cover staged migration upgrades`：仅新增 staged migration upgrade 覆盖。
  - `a719a8e6c0d49871b8cb31eb3a8fe08ec645f3ee fix: preserve local behavior after v0.1.160`：恢复缺失的 OpenAI moderation 委派，并删除引用不存在符号的死 WS failover closure。
  - `31b132689f858f4b6e657c653829c80d2fc09738 fix: preserve local behavior after v0.1.160`：仅为 `PromptAdminService` 添加 Wire 绑定，解除 generate 的 provider 缺失。
  - `516e998b07d8e81a6d1e18f7acc0c0425d6b083f docs: record v0.1.160 full gate`：仅记录 Task 9 full gate 和 ledger 证据。
  - `fd909c5be1eb628a551d11128a4329ade57070f5 test: harden staged migration coverage`：补齐上游 migration 快照/断言，并修正 image moderation 测试 mock 与异步 drain。
  - `0186949e0e6810d5e4eb49345280a3c3a7b8267a docs: record task 9 review fixes`：仅将上述 review-fix 的 RED/GREEN、远端清理和剩余顾虑记入 Task 9 报告。
- worker panic：根因是测试 mock 的 nil 嵌入接口在异步 `enqueueRecord -> worker -> persistContentModerationLog -> applyFlaggedAccountSideEffects -> CountFlaggedByUserSince` 调用链触发；生产 repository 实现完整接口。review-fix 补齐 mock 计数方法并等待 `CreateLog` drain，不改生产代码。修复后本机及已提交 archive 的 `TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease -count=3 -v` 均 PASS，输出无 `content_moderation.worker_panic`；第 1843 行的旧 panic 顾虑由本条替代。
- Gate 复核：Task 9 本地 `make test`、`make build`、两轮 generate/generated diff 和静态门禁均 GREEN；远程 archive `31b132689` 的 full integration GREEN。远程日志为 `C:/Users/caiqy/AppData/Local/Temp/sub2api-task-9-d70a45257d0f46c0911f6e17747ab6aa-integration.log`；remote `/tmp/sub2api-task-9-d70a45257d0f46c0911f6e17747ab6aa` 已删除并确认不存在，本地 tar 已删除。环境型 skip 仅为 DingTalk、TLS/JA3 外部条件、Prompt Audit Redis/PostgreSQL 条件和无 OpenAI key；已分类，目标 migration 未 skip。
- Task 11 状态：证据已补齐，等待独立复审。能力交集 `gap=0`、目标 migration 未 skip、VERSION 不变；残余风险为既有环境型 skip、Browserslist/Vite advisory、Ryuk handshake advisory，以及 Windows `user-mapped section` 历史根因未定。

## Task 11：v0.1.161 合入与记账（2026-07-27）

- 合并提交：`f2158292c7ff3de4caa7ec22f9b7148400948f08 merge: upstream v0.1.161`。第一父为 `3fc60752acc459ecc37cd50b40df4a1f84ce3b62`，第二父及 `v0.1.161^{}` 为 `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c`。
- `backend/cmd/server/VERSION` 保持本地中间版本 `0.1.159.6`；后端和前端 runtime 路径未恢复已审批移除的 `openai-first-token-timeout`。
- step-up/session：设置 DTO、handler、Wire 和 SettingsView 同时保留会话绑定、本地 scheduler 设置与上游 step-up 开关、审计保留及管理员充值返利字段；`SetStepUpDeps` 保持注入路径。
- scheduler：`AccountRepository` 同时具有本地临时不可调度读取和上游 model availability 查询；测试桩补齐新接口。OpenAI failover 在 canonical model 下处理冷却，images/embeddings 保留响应头并只在不永久禁用时允许同账号 pool retry。
- Grok：媒体 handler 同时保留本地 multipart spool、快照脱敏、usage 记录和上游 generation eligibility、视频 owner binding/content proxy；服务层同时保留 multipart 转换、映射模型和官方视频 URL/redirect 约束。Responses 保留一次 invalid encrypted-content 清理重试及 upstream-attempt 记录。
- gateway/frontend：Responses、images 和 video 路由保留 usage detail capture 与 Grok WebSocket 拒绝，同时纳入 text body limit 和视频 content 代理；账号创建表单纳入上游 billing probe/expiry 字段，App 同时保留自定义菜单可见性与 favicon 更新。
- migration：本地 `172_video_per_second_billing_metadata.sql`、`181_group_duplicate_operation_id.sql` 与上游 181/182 快照继续存在，并纳入 `183_ops_ingress_reject_aggregates.sql`、`184_auth_cache_invalidation_outbox.sql`；staged upgrade 和并发 migration runner 覆盖均保留。
- 静态检查：精确 conflict marker 扫描和 `git ls-files -u` 均为空，提交前 `git diff --cached --check` 通过。`go test -run '^$' ./cmd/server ./internal/handler/admin ./internal/handler ./internal/service ./internal/server/routes` 通过；`pnpm exec vue-tsc --noEmit` 通过。未运行 Task 12 的 full test/build/generate/integration gates。

## Task 12：v0.1.161 full gate 与回归收口（2026-07-27）

- 历史保留：`0775a6063 fix: preserve local behavior after v0.1.161` 包含无效的 WS V2 terminal ownership 放宽；`f1cde1b52 test: cover terminal WS turn binding` 是把保护测试改为反向断言的无效适配，均不改写且不作为 GREEN。
- follow-up：`c3bfb765f fix: restore created-only WS turn ownership` 恢复空 active turn 仅由同 ID `response.created` 绑定；`47a6c031e test: align websocket lifecycle fixtures` 为正常 terminal/delta 序列补同 ID created；`07029cc45 fix: remove stale Grok session hash assignments` 删除 full gate lint 发现的两项死赋值。
- 真实证据：恢复 `TestObserveUpstreamMessage_BindsOnlyResponseCreated` 后，在 `0775a6063` 行为上外来 `response.completed` 错误成为 terminal 的 RED；`c3bfb765f` 后原保护测试和 `go test ./internal/service/openai_ws_v2 -count=1` GREEN。terminal-first lifecycle fixture 随严格契约暴露并由 `47a6c031e` 对齐，不是放宽 ownership 的理由。首次 `make test` 的唯一生产 RED 是 `grok_media.go` 两项 `ineffassign`，`07029cc45` 后 handler lint/聚焦测试及完整门禁 GREEN。
- 聚焦与本地门禁：service 覆盖模型冷却、advanced/layered scheduler、fallback/WaitPlan、DB recheck、Grok/spooling/redaction 和 moderation；handler/admin/middleware 覆盖 session/step-up、Grok 视频、HTTP failover 和 Ops；migrations/repository/config/securityaudit/cmd server 及 API Key/billing probe/RiskControl 前端用例均通过。最终 `make test` 为后端全测试/lint 和前端 201 文件、1537 用例 GREEN；`make build` 在仅进程级补 Git `sh` PATH 后按 VERSION `0.1.159.6` GREEN；两轮 generate 的 Ent/Wire diff 均为空。
- 静态/migration：`git diff --check`、`git ls-files -u`、精确 conflict marker 和旧本地 first-token timeout 业务符号检查均通过；本地 172/181 与上游 `181_prompt_audit.sql`、182、183、184 全部保留。migration/repository 聚焦测试通过。
- remote integration：已提交 archive `07029cc45` 在 `local-serv-ai`（Go `1.26.5`、Docker `29.2.1`）重建 `.test-tmp` 后执行 `CI=true GOFLAGS='-v' go test -tags=integration ./...`，退出 `0`。日志 `C:/Users/caiqy/AppData/Local/Temp/sub2api-task12-07029cc45-integration.log` 无 FAIL/panic；`TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` PASS 5.20 秒，`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS 4.75 秒。16 个 skip 均为已接受 DingTalk、TLS/JA3/profile、CI concurrency、Prompt Audit Redis/PostgreSQL、无 OpenAI key 环境基线，未命中本阶段能力。远端 `/tmp/sub2api-task12-07029cc45` 与本地 archive 已清理，日志保留。
- Task 12 结论：`DONE_WITH_CONCERNS`。残余风险为 Windows Git `sh`/历史 generate 文件锁和既有前端 advisory；其门禁证据由 Task 13 只读汇总，未重跑。

## Task 13：v0.1.161 能力矩阵与证据封闭（2026-07-27）

- 状态：`DONE_WITH_CONCERNS`。起点为 `0595aa671daa90d46d3e030c84a6b096adc019af`；本节只读核对既有提交、报告和日志，不修改行为代码、不构造 TDD RED，且不重跑 Task 12 heavy gates。
- tag changed-files：`git diff --name-only 'v0.1.160^{}..v0.1.161^{}'` 为 257 个路径，逐字原始清单以稳定 marker `TASK6:v0.1.161:raw:begin/end` 为准（当前快照第 562-822 行）。与阶段 0 矩阵的实际交集为下表 22 个精确路径，均有既有测试、调用链或人工证据：`protected=21`、`manual=1`、`gap=0`。

| 交集路径 | 能力与结论 | 精确既有证据 |
| --- | --- | --- |
| `backend/cmd/server/wire_gen.go` | Ent/Wire，`protected` | `go -C backend test ./cmd/server`；Task 12 两轮 `make -C backend generate` 后 `backend/ent` 与 `wire_gen.go` 均零 diff。 |
| `backend/internal/handler/admin/user_handler_batch_limits_test.go` | user bulk limits，`protected` | `TestUserHandlerBatchUpdateLimitsAcceptsPartialAndZeroValues`。 |
| `backend/internal/handler/admin/user_handler_list_apikey_group_test.go` | API Key group ID 解析，`protected` | `TestAdminUserList_ParsesAPIKeyGroupID`；本轮 `go -C backend test ./internal/handler/admin -run '^TestAdminUserList_ParsesAPIKeyGroupID$' -count=1` exit `0`、PASS。 |
| `backend/internal/repository/user_repo.go` | user resource control/menu hiding，`protected` | `TestAdminServiceUpdateUserBlockedGroups` 与 integration `TestUserRepoSuite/TestHiddenUIResourcesRoundTrip`。 |
| `backend/internal/server/middleware/session_binding.go` | Grok/platform sticky、session，`protected` | `TestLayered_SessionStickyPreservesGrokBinding`、`TestGatewayService_SelectAccountForModelWithPlatform_StickyDisabledBypassesStickyReadAndWrite`、`TestGatewayService_SelectAccountForModelWithPlatform_StickySession`。 |
| `backend/internal/server/middleware/step_up.go` | step-up，`protected` | `TestEnforceStepUpPassesWithGrant`；Task 11 的 enable/disable 转换修复聚焦测试通过。 |
| `backend/internal/service/channel_monitor_checker_body_test.go` | body replay/spooling，`protected` | `TestOpenAIForwardReusesBoundRequestBodyHandle`。 |
| `backend/internal/service/openai_gateway_grok_cache.go` | prompt cache，`protected` | `TestForwardAsAnthropic_InjectsPromptCacheKeyForAPIKeyMessagesDispatch`。 |
| `backend/internal/service/openai_gateway_passthrough.go` | gateway passthrough，`protected` | `TestPassthroughFieldsV2OpenAIForward_APIKeyBodyMapCopiesFromOriginalInboundRequest`。 |
| `backend/internal/service/openai_images_responses.go` | async images/object storage，`protected` | `TestImageTaskServiceCompleteOffloadsToStorage`。 |
| `backend/internal/service/openai_images.go` | image capability，`protected` | `TestLayered_RequiredImageCapabilityFiltersUnsupportedAccounts`。 |
| `backend/internal/service/ops_service_user_error_test.go` | failed usage，`protected` | `TestOpenAIForwardStreamingResponseFailedReturnsUsageWithError`。 |
| `backend/internal/service/scheduler_snapshot_batch_query_test.go` | DB fresh recheck，`protected` | `TestLayered_GroupedAccountPassesDBFreshRecheck`。 |
| `backend/internal/service/scheduler_snapshot_full_rebuild_lifecycle_test.go` | advanced/layered scheduler，`protected` | `TestLayered_PriorityDeterminism`。 |
| `backend/internal/service/setting_parse.go` | settings backfill，`protected` | `TestSettingService_GetAllSettings_BackfillsGatewayControlsFromConfigAndDB`。 |
| `backend/internal/service/setting_update.go` | runtime hot update，`protected` | `TestSettingService_UpdateSettings_PersistsAndHotUpdatesGatewayControls`。 |
| `backend/internal/service/subscription_service.go` | subscription quota atomic reset，`protected` | `TestAdminResetQuota_KeepsCacheAfterAtomicResetFailure`。 |
| `backend/internal/service/upstream_billing_probe.go` | image/video billing、upstream multiplier，`protected` | `TestCalculateImageCost`、`TestCalculateVideoCostUsesSeparateConfig`、`TestOpenAIFreshUpstreamBillingRateRecomputesPeakAtSelectionTime`。 |
| `backend/internal/service/user_subscription_daily_quota_test.go` | subscription quota atomic reset fixture，`protected` | atomic reset failure 断言选定 daily/weekly window 且 L1 cache 不删。 |
| `backend/migrations/183_ops_ingress_reject_aggregates.sql` | migration 181-184，`protected` | remote `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` PASS 5.20s，`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS 4.75s。 |
| `frontend/src/i18n/locales/en/admin/settings.ts` | frontend translations，`protected` | `pnpm --dir frontend run test:run -- src/i18n/__tests__/localeKeysExist.spec.ts`；Task 12 `make test` 亦通过。 |
| `frontend/vite.config.ts` | dependencies/local test gates，`manual` | Task 6 `go -C backend mod verify` 与 `pnpm --dir frontend install --frozen-lockfile` 均通过且无 tracked diff；Task 12 `make test` 通过。 |

- 跨路径补充聚焦回归（不计入 22 条精确路径或 `protected=21`、`manual=1`、`gap=0`）：模型冷却、fallback/WaitPlan、Grok 视频/owner/redaction、content moderation、HTTP/WS failover、YAML、API Key/billing probe UI 与 migration/repository 均已有 Task 12 GREEN 证据。
- merge 与冲突台账：`f2158292c7ff3de4caa7ec22f9b7148400948f08` 的第二父及 `v0.1.161^{}` 为 `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c`。26 个原始 U 文件均逐项人工决议，`git show --remerge-diff f2158292c` 与该集合相符：
  1. `backend/cmd/server/VERSION`：按中间版本策略只保留本地 `0.1.159.6`。
  2. `backend/cmd/server/wire_gen.go`：融合本地 PromptAdminService bind 与上游 auth-cache/outbox 图。
  3. `backend/internal/config/config_test.go`：融合本地配置覆盖与上游默认值/环境读取。
  4. `backend/internal/handler/admin/setting_handler.go`：融合本地 runtime/session 与上游 step-up、审计、充值返利字段。
  5. `backend/internal/handler/admin/setting_handler_update.go`：融合本地热更新与上游 step-up 更新。
  6. `backend/internal/handler/grok_media.go`：融合 spooling/redaction/usage/sticky 与 generation/video 入口。
  7. `backend/internal/handler/grok_media_test.go`：融合 spool/redaction 与 capability/video 覆盖。
  8. `backend/internal/handler/openai_gateway_handler_test.go`：融合 Responses/WS failover/usage 与上游并发支撑，不恢复旧 timeout。
  9. `backend/internal/repository/migrations_schema_integration_test.go`：融合语义空默认值、本地 staged upgrade 和上游 runner lock 覆盖。
  10. `backend/internal/server/routes/gateway.go`：融合 usage detail/Grok WS 拒绝与 body limit/video content。
  11. `backend/internal/service/account_service.go`：融合 temp-unschedulable 与 model-availability 查询。
  12. `backend/internal/service/admin_account.go`：融合 probe 关闭和上游 ProbeEnabled 更新。
  13. `backend/internal/service/gateway_multiplatform_test.go`：采用补全测试桩以实现两个 repository 契约。
  14. `backend/internal/service/gemini_multiplatform_test.go`：采用补全测试桩以实现两个 repository 契约。
  15. `backend/internal/service/grok_media.go`：融合解析/映射/CLI 约束与 owner/status/官方 URL 限制。
  16. `backend/internal/service/openai_alpha_search.go`：融合 failover side effect 与 PAT/tool 401 永久置错限制。
  17. `backend/internal/service/openai_embeddings.go`：融合响应头与非永久禁用前提下的 pool retry。
  18. `backend/internal/service/openai_gateway_grok.go`：融合 attempt/延迟记录与一次 encrypted-content 清理重试。
  19. `backend/internal/service/openai_gateway_passthrough.go`：融合 canonical model 错误处理与本地 failover/Ops。
  20. `backend/internal/service/openai_images.go`：融合响应头和条件 pool retry。
  21. `backend/internal/service/openai_images_responses.go`：融合响应头/OAuth 条件重试与 Responses image 不同账号重试。
  22. `deploy/config.example.yaml`：融合 128 MiB 读取上限、32 MiB text body 和 8 MiB 非流默认值。
  23. `frontend/src/App.vue`：融合自定义菜单可见性与 favicon。
  24. `frontend/src/components/account/CreateAccountModal.vue`：融合本地能力与 billing probe/expiry payload。
  25. `frontend/src/components/account/__tests__/CreateAccountModal.spec.ts`：采用保留 passthrough/mixed-channel fixture 的测试策略。
  26. `frontend/src/views/admin/SettingsView.vue`：融合 gateway runtime/scheduler/session 与 step-up 开关。
- Task 11 follow-up：`1b80f95c9` 闭合 6 项 finding：step-up 会话/TOTP、Grok video owner binding、API Key helper quota/notify、billing probe UI、Ops monitoring snapshot、YAML 空格与 body limits；`a534148f3` 记录修复，`2fce42855` 修正状态。Sol 复审 PASS（`ses_05e81d823ffe1nVy2t46bqPWUk`）。
- Task 12 完整历史：`0775a6063` 是初始兼容集合；`f1cde1b52` 是无效反向测试，绝非 GREEN；`c3bfb765f` 恢复 created-only；`47a6c031e` 对齐 fixture；`07029cc45` 清除 Grok lint 死赋值；`81aa202ba` 记录 docs；`0595aa671` 协调 checkoff。Sol thorough review PASS（`ses_05e2803fbffeeoKJpSwmoE2aGm`）。
- created-only 契约：仅同 ID `response.created` 可绑定空 active turn；外来 delta/terminal 不计 usage、不完成 turn、不释放 permit。正常 fixture 必须先发送同 ID created，随后才发送 delta/terminal。`TestObserveUpstreamMessage_BindsOnlyResponseCreated` 在 `0775a6063` 的 terminal 放宽行为上真实 RED，`c3bfb765f` 后与 `go test ./internal/service/openai_ws_v2 -count=1` GREEN；`47a6c031e` 只补 fixture，不放宽契约。
- Task 12 聚焦覆盖已包含模型冷却、advanced/layered scheduler、fallback/WaitPlan、DB recheck、sticky、step-up、Grok owner/spooling/redaction/video URL、API Key helper、billing probe、Ops snapshot、YAML、moderation、HTTP/WS failover、Wire bind 和 migration 181-184。最终 `make test` 为 201 files/1537 tests，`make build` 通过；双轮 generate/diff、静态/VERSION/timeout/migration 核验及 remote integration 均通过。`VERSION=0.1.159.6`，已移除的 `openai-first-token-timeout` 未恢复。remote `local-serv-ai` 为 Go 1.26.5/Docker 29.2.1，日志无 FAIL/panic，archive 和远端目录均已清理并保留日志；16 个 skip 是已接受环境基线，不命中本阶段能力。
- 放行边界：v0.1.161 实现门禁的精确矩阵为 `gap=0`，本轮 docs/evidence 修正待 Sol 复审。Task 14/v0.1.162 在该复审 PASS 前保持封闭；不因此开始下一 tag、push、tag、release、deploy 或合并 main。

## Task 17：v0.1.163 合入与记账（2026-07-27）

- 状态：`DONE_WITH_CONCERNS`。任务起点为 `b7b7bba6952460bb7cc38f1d41a0de95c449bcb8`；`v0.1.163^{}` 和 merge 时的 `MERGE_HEAD^{}` 均为 `d0bdd7e771636a8d315f542cafd39484f39bd60c`。merge commit 为 `02abe1574bf8044a1b180e62b002f58f9928d88f`，第一父为任务起点，第二父为该精确 peeled SHA；未触碰 `v0.1.164`、Task 18/19、远程、tag、release、deploy 或 push。
- 本 merge 的 first-parent diff 为 171 个文件、7,412 行新增、613 行删除。完整、可复现的 changed-files 清单由 `git diff --name-only 02abe1574bf8044a1b180e62b002f58f9928d88f^1 02abe1574bf8044a1b180e62b002f58f9928d88f` 记录；范围包括 root/dependency/deploy、Ent、migration 185、backend handler/repository/service/setup、frontend reasoning/usage/payment/mobile/i18n/UI 与 docs/screenshot。
- 14 个文本冲突均人工融合，未使用整文件 `ours/theirs`。`git show --remerge-diff --name-only 02abe1574b` 还会列出 `backend/cmd/server/VERSION` 及两份 Ent 生成输出；前者是版本元数据保留，后两者是 post-merge regeneration，均不计入 14 个文本冲突。

| 文本冲突 | ours | theirs | 最终融合 |
| --- | --- | --- | --- |
| `backend/internal/handler/admin/usage_handler_request_type_test.go` | 本地 usage 筛选覆盖 | request-type 筛选覆盖 | 保留双方断言。 |
| `backend/internal/handler/openai_gateway_handler.go` | 本地网关处理 | reasoning policy 入口调整 | 保留网关处理并传递 policy 所需信息。 |
| `backend/internal/server/api_contract_test.go` | 本地 contract | 上游响应字段 | 两侧断言共存。 |
| `backend/internal/service/api_key_auth_cache_impl.go` | 本地缓存/并发字段 | group reasoning 字段 | `apiKeyAuthSnapshotVersion = 18`，两侧字段均进入 snapshot。 |
| `backend/internal/service/api_key_auth_cache_version_test.go` | 本地版本断言 | 上游字段断言 | 覆盖 v18。 |
| `backend/internal/service/api_key_service_cache_test.go` | 本地失效语义 | group policy 缓存语义 | 同时验证失效和 policy 变更。 |
| `backend/internal/service/gateway_anthropic_passthrough.go` | 本地 cache/透传计费 | 输出估算和 forced-cache 分类 | 两类计费逻辑均保留。 |
| `backend/internal/service/gateway_forward_as_responses_test.go` | 本地 Responses 覆盖 | compact SSE 覆盖 | 两套用例均保留。 |
| `backend/internal/service/openai_account_scheduler.go` | advanced/layered、sticky、DB recheck | quota metadata、runtime 过滤/诊断 | 四段预算、粘性和 DB 复核与上游 metadata 共存。 |
| `backend/internal/service/openai_account_scheduler_layered.go` | layered 候选逻辑 | 不可用原因详情 | layered 选择保留，并适配三参无可用账号错误。 |
| `backend/internal/service/openai_gateway_grok.go` | Grok sticky/cache | pool retryable/policy rejection | 两种失败条件合并。 |
| `backend/internal/service/openai_gateway_scheduling.go` | 本地调度失败语义 | 上游诊断详情 | `noAvailableOpenAISelectionError` 统一接受详情。 |
| `backend/internal/service/openai_gateway_service_test.go` | 本地网关回归 | Codex identity 覆盖 | 两套测试共存。 |
| `frontend/src/components/admin/usage/UsageFilters.vue` | username-first 展示 | 搜索取消/竞态处理 | 按用户名优先展示并取消过期请求。 |

- `backend/cmd/server/VERSION` 保持 `0.1.159.6`。reasoning policy 的持久化、管理和强制执行由 `backend/internal/domain/reasoning_effort.go`、`backend/ent/schema/group.go`、`backend/internal/repository/group_repo.go`、`backend/internal/service/openai_reasoning_effort_policy.go`、`frontend/src/components/admin/group/ReasoningEffortPolicyFields.vue` 与 `backend/migrations/185_group_reasoning_effort_policy.sql` 共同覆盖。
- Scheduler quota metadata/LastUsedAt 证据位于 `backend/internal/repository/scheduler_cache.go`、`backend/internal/repository/scheduler_cache_last_used_unit_test.go`、`backend/internal/service/openai_account_scheduler.go` 和 `backend/internal/service/openai_account_scheduler_layered.go`；融合保留 runtime 过滤、粘性、DB recheck，并隔离 LastUsedAt 写入以免污染调度键。
- Cleanup 初始证据位于 `backend/cmd/server/main.go`：超时路径仍会进入 cleanup；该结论不能证明 active handler 已完成或零丢失，已在 Review Round 1 标为 concern 并修复。Billing 证据位于 `backend/internal/service/gateway_anthropic_passthrough.go`、`backend/internal/service/openai_gateway_response_handling.go` 及 image usage/Responses 测试，保留 output estimate、cache 分类与 image usage 修复。
- 依赖与 migration：`frontend/package.json` 为 `axios ^1.18.0`，`frontend/pnpm-lock.yaml` 为解析的 `1.18.1`；新增 `backend/migrations/185_group_reasoning_effort_policy.sql`，未重命名历史 migration。
- Ent：首次 focused service 测试在 `ent/runtime.init.0` 以 `interface {} is int, not string` 失败。根因是本地并发字段令 `Group` schema 的上游 reasoning 字段索引后移。执行标准 `go generate ./ent` 后，生成器仅修改 `backend/ent/mutation.go` 和 `backend/ent/runtime/runtime.go`：`GroupMutation.Fields()` 容量 51 -> 53，`max_reasoning_effort` 索引 46 -> 48，`reasoning_effort_mappings` 索引 47 -> 49；focused service 测试随后通过。
- 限定验证：`go generate ./ent` 通过；`go test -tags=unit ./internal/service -run '^(TestAPIKey|TestParseAnthropicSSEField|TestHandleResponses.*CompactSSEFormat|TestOpenAI)' -count=1` 通过（58.789s）；本 session 既有 focused Go handler/server 成功检查保留、未重跑；`pnpm --dir frontend exec vitest run src/components/admin/usage/__tests__/UsageFilters.spec.ts` 通过（1 file、4 tests）；`pnpm --dir frontend run build` 通过。提交前 `unmerged=0`、cached diff-check、精确 conflict marker 检查、protected staged 检查和 `VERSION=0.1.159.6` 均通过。
- Task 18 边界：broad backend `./...`、完整 Vitest、lint/integration 等 full gate 不计为本 Task PASS。收尾前误启动的 broad Go 仅输出依赖下载，完整 Vitest 仅输出部分测试结果；均没有完成摘要、均未重跑。部分 Vitest 输出含既有 `router-link` warning 与 jsdom `AggregateError`，Task 18 必须完整复核。

### Task 17 Review Round 1 修复（2026-07-27）

- 输入：协调者核验 Sol reviewer `ses_05ce435e2ffeKaG1tASC6Efg88` 的三项 finding。代码提交为 `c411927ece1e83a19e4d9d70434acfcc7eb97316 fix: preserve v0.1.163 scheduler and shutdown behavior`；该提交不改写 merge commit。
- Sticky RED：新增 `TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyRuntimeBlockedAccountFallsBackWithoutRebinding` 后，global runtime cooldown 和 model-specific transient cooldown 都在 `selectBySessionHash` 错误选中并 acquire sticky 账号，测试分别以“Expected nil, but got AccountSelectionResult”失败。
- Sticky GREEN：default 与 layered `classifySessionStickyAccount` 均调用既有 `isOpenAIAccountRequestRuntimeBlocked(account, req.RequestedModel)`；runtime block 返回 `(nil, false)`，因此 `SkipStickyBind` 保留 temporary binding。两种 cooldown 均回退到 backup、未 acquire blocked account、未删除或改写 binding。
- Shutdown/Cleanup RED：先加入可控 lifecycle 与 phase 测试；当前 HEAD 缺少 `activeHandlerTracker`、`shutdownServerWithDrain`、`cleanupPhase` 和 `runCleanupPhases`，`cmd/server` 定向测试以未定义符号编译失败，证明所需行为尚未存在。
- Shutdown GREEN：`main.go` 在 server handler 外包 tracker；先给 `Server.Shutdown` 5 秒 graceful budget，超时后调用 `Server.Close`，再在总计 15 秒 hard context 内等待 active handler drain，之后才调用 `app.Cleanup()`。测试让 handler 跨过首次短 timeout、提交 side effect 后结束，并断言 helper 在该 side effect 前不返回且 `close -> drain` 顺序成立；另一个测试证明 hard deadline 真实返回。
- Cleanup GREEN：新增 package-private phase runner。独立 producer stop 仍并行；随后严格执行 `usage-record-drain -> quota-final-flush -> billing-cache-drain -> Redis -> Ent`。每个 phase 受同一 10 秒 context 约束；某 phase 阻塞到 deadline 时立即返回且不启动 downstream teardown。顺序和 deadline 测试直接断言该行为，不再只是 no-panic。
- Wire 与注释：修改 `wire.go` 后执行 `go generate ./ent` 与两次 `go generate ./cmd/server`；`wire_gen.go` 稳定哈希为 `9b71b96ace6fa4309d4cf427ae8be31f22e69588`。移除 `gateway_anthropic_passthrough.go` 已过期的“无任何行为变更”注释；scheduler fallback 增加 overflow 才启用 64 probe 及四 pass attempted/full 顺序说明。
- 限定 GREEN：`go test -tags=unit ./internal/service -run '^(TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyRuntimeBlockedAccountFallsBackWithoutRebinding|TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyRateLimitedAccountFallsBackToFreshCandidate|TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDBRuntimeRecheckSkipsStaleCachedAccount|TestOpenAIAccountScheduler_SkipsAccountBlockedForRequestedModel|TestUsageRecordWorkerPool_SubmitEnqueued|TestUsageRecordWorkerPool_SubmitAfterStop|TestBillingCacheServiceQueueHighLoad|TestBillingCacheServiceEnqueueAfterStopReturnsFalse|TestHandleNonStreamingResponseAnthropicAPIKeyPassthrough_ForceCacheBillingResponse|TestApplyOutputTokenEstimation_ZeroOutputWithText)$' -count=1` 通过；`go test ./cmd/server` 的 lifecycle/phase/cleanup 定向测试和 compile-only 检查通过；generated output stable、unmerged/marker/diff-check 和 VERSION 核验留在本 Task 限定范围。
- 真实保证与剩余边界：修复保证首次 graceful timeout 后会 force-close，并在进入 app cleanup 前有界等待已跟踪 handler；它不声称无法证明的零丢失。若 handler 超过 15 秒 hard deadline，helper 记录并返回，cleanup 仍继续以避免无限停机。若 cleanup phase 自身超时，后续可能使用的 quota/billing/infra 不会关闭。Task 18 full gate 仍未启动。

### Task 17 Review Round 2 修复（2026-07-27，待协调者复审）

- 代码提交：`73d25ba105fe83043fe3497490fa7ce1e56edd19 fix: close v0.1.163 background lifecycle gaps`。它不改写 merge commit、未触碰 `backend/cmd/server/VERSION`、Task 18、远程、tag、release、deploy 或 push。
- RED：新增 admission late-entry、parallel cleanup error、实际 `producers -> usage-record-drain -> deferred-last-used-flush` wiring、ImageTaskService cancel/wait/reject、Deferred in-flight flush、cyber worker routing、billing detached goroutine，以及 layered sticky global/model cooldown 覆盖。首轮 server/service/handler 命令因 `CloseAdmission`、`ImageTaskService.Run/Shutdown` 与 `DeferredService.Stop(context.Context)` 尚不存在而按预期编译失败；这证明 lifecycle API 缺口。layered cooldown 是 Round 1 已有实现的覆盖补齐，新增后在 GREEN 命令中验证。
- GREEN：handler admission 在 `Server.Shutdown` 前关闭并对 late entry 返回 `503`；`runCleanupParallel` 以 `errors.Join` 返回子错误，phase 错误停止 downstream teardown。ImageTaskService 统一拥有任务 goroutine、取消和等待，AsyncImageHandler 使用该 lifecycle，且 Wire 将其置于 producer phase、usage drain 之前。
- GREEN：cyber policy 后处理改投既有 `UsageRecordWorkerPool`；platform quota DB 直写与余额/账号阈值通知在同一 usage task 内执行，不再从 `finalizePostUsageBilling` 派生 detached goroutine。DeferredService 的 final flush 接受 cleanup context、串行等待并接入 usage 后、quota 前 phase；失败或 deadline 会保留 downstream Redis/Ent。
- 限定验证实际通过：`go test -tags=unit ./cmd/server -count=1`；`go test -tags=unit ./internal/service -run '^(TestImageTaskService|TestDeferredServiceStopWaitsForInFlightFlush|TestFinalizePostUsageBillingDoesNotStartDetachedGoroutines|TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyRuntimeBlockedAccountFallsBackWithoutRebinding|TestLayeredOpenAIAccountSchedulerSessionStickyRuntimeBlockedAccountFallsBackWithoutRebinding)$' -count=1`；`go test -tags=unit ./internal/handler -run '^(TestAsyncImageHandler|TestRecordCyberPolicyIfMarked|TestOpenAIGatewayHandler_ResponsesCyberPolicyCreatesSingleUsageLog|TestOpenAIGatewayHandlerSubmitMandatoryUsageRecordTask)' -count=1`。`go generate ./ent`、`go generate ./cmd/server`、`git diff --check`、`git ls-files -u` 与 VERSION 零 diff 均通过。
- 剩余边界：不响应取消的 handler 或 image upstream 仍可能耗尽 hard deadline；此时 cleanup 不会关闭其下游资源。未运行 Task 18 full gate；Sol 最终复审由协调者恢复既有 `ses_05ce435e2ffeKaG1tASC6Efg88` 会话后决定，不在本轮声明 PASS。

### Task 17 预算外最终修复（2026-07-28）

- 基线是 `0b265313e2d17a38e996444a35894cdde1e9a835`。代码提交为 `0e69a1b2cc67dece06d244783c75a04390d23d7f fix: finish v0.1.163 background drains`，父提交为该基线；未改写既有提交，`backend/cmd/server/VERSION` 保持 `0.1.159.6`。
- RED：`go test ./internal/service -run TestFinalizePostUsageBillingWaitsForNotificationDelivery -count=1` 观察到余额和账号配额的 usage task 在 SMTP 交付前返回；`go test ./internal/handler -run TestRecordCyberPolicyIfMarkedBillsBeforeBlockingModeration -count=1` 观察到阻塞 moderation 先耗尽 worker deadline；`go test ./cmd/server -run TestProvideCleanupDrainsOpsErrorsBeforeEntTeardown -count=1` 观察到 Ent teardown 先于已入队 ops error drain。
- GREEN：余额和账号配额通知改为在 usage task 内完成，保留 panic recovery；Ops error worker 在 `usage-record-drain` 后、deferred/Redis/Ent 前 drain，Stop API 接受 cleanup context 并返回错误，worker 固定持有启动队列以避免关闭后读取 nil queue；`forwardErrored && gwSvc != nil` 的 cyber mandatory billing 先于 moderation、session block 和 Ops 辅助工作。`gateway_usage_billing.go` 的 DB 持久化注释同步为实际同步路径。
- 验证：三项精确 GREEN 命令通过；`go test ./internal/service -count=1`、`go test ./internal/handler -count=1`、`go test ./cmd/server -count=1` 通过；`go generate ./cmd/server` 后 `wire_gen.go` 已更新并经零工作树 diff 核验。
- 真实边界：未运行 Task 18/full gate，未触及 `v0.1.164`、远程、push/tag/release/deploy。cleanup phase 超时时会跳过下游 teardown，但 `main.go` 在 handler hard deadline 后仍无条件调用 `app.Cleanup()`；因此只记录有界停机取舍，不能声称下游资源必然持续保留。
- 台账纠正：`4ba0c9f235095b1123a76cca784b92da11bb508d docs: record v0.1.163 extra review fix` 误把 `.superpowers/sdd/task-17-report.md` 加入历史。后续正式 ledger 修正提交只从 Git 索引移除该 scratch，保留其本地内容并恢复 ignore 保护；该错误及净修正会在最终报告中显式列出。
- 净修正：首次 ledger 修正提交 `4e1c83ad2a6726aed16d7fc3c87fa7c14b3e26e0` 仅记录了正式台账，未实际删除 scratch，故不将其误记为完成。后续提交同时更新本段并从索引删除 scratch；不 reset、amend、rebase 或删除本地 scratch 内容。

## Task 18：v0.1.163 回归修复与 full gate（2026-07-28）

- 状态：`DONE_WITH_CONCERNS`。起点与 Task 17 最终 review HEAD 为 `2eb7ba771eb6d2cb195f267fba0af84056a83f39`；未开始 Task 19、未合入 `v0.1.164`、未改最终版本、未 push/tag/release/deploy。`backend/cmd/server/VERSION` 始终为 `0.1.159.6`，已移除的 `openai-first-token-timeout` 未恢复。
- 先执行 v0.1.163 命中矩阵：scheduler quota/LastUsedAt、sticky、fallback/WaitPlan、DB recheck、reasoning、failed usage、billing、producer/deferred lifecycle 与 server shutdown/Cleanup 的所有定向 Go 命令均 exit `0` 且命名目标实际 PASS。迁移定向命令在本机因 Docker 不可用退出 `0` 但实际 SKIP，不计本地 migration PASS，登记为 concern。
- 首次根 `make test` exit `2`，在 `golangci-lint` 发现 `deferred_service_test.go` 丢弃 `flushLastUsed` 错误，以及 `openai_account_scheduler.go` 的无调用方 `acquireExhausted`。两项均经复现、blame/调用方核验后最小修复：测试将 goroutine error 传回主协程断言，删除无调用方方法。独立提交为 `f7a14121dd75525e55a4530321382863e20da2a0 fix: preserve local behavior after v0.1.163`；修复后目标 lifecycle/scheduler 测试 PASS、`golangci-lint run ./internal/service` 为 `0 issues`。
- 修复提交后的 fresh 本地 full gate：根 `make test` exit `0`（前端 Vitest `209` files、`1576` tests PASS）；`make build` exit `0`（backend `0.1.159.6`、Vite `1013` modules）；两轮 `make -C backend generate` 均 exit `0`，每轮后的 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 均 exit `0`；`git diff --check` exit `0`、`git ls-files -u` 为空、精确 conflict-marker 扫描 exit `0`、VERSION 精确匹配。前端测试 stderr、Browserslist、dynamic-import/chunk-size 和协调者既有 CRLF 提示均为 advisory。
- remote full gate：archive 为上述已提交 HEAD，唯一目录 `/tmp/sub2api-v0.1.163-368822c706f84dee88147b896e6bed00`。预检 exit `0`（Go `1.26.5`、Docker `29.2.1`）；在 archive 内重建 `backend/.test-tmp` 后，`CI=true GOFLAGS=-v TMPDIR/TMP/TEMP=<isolated .test-tmp> go test -tags=integration ./...` exit `0`。`TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` PASS（`4.79s`），`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS（`4.42s`）；`185_group_reasoning_effort_policy.sql` 是唯一 `185_*.sql`，SHA-256 为 `940c1132c5a01f824a83439cce41e6a182d51c1a3c6357a96c76dba950fc307e`，且升级测试断言重放后 migration record count/checksum 不变。
- remote 首次解包命令因 ssh-skill Windows native fallback 在本机提前展开 `$d`，以 `/source.tar` 失败，未执行测试；改为绝对路径命令后完整 gate 通过。完整远端日志有 `13` 条 SKIP、`0` 条 FAIL：11 条为外部环境、凭证或可选服务缺失，分别是 `TestDialerAgainstCaptureServer`（未设置 `TLSFINGERPRINT_CAPTURE_URL`）、`TestPromptAuditConfigCASSecretRoundTripInvalidationAndTTL`、`TestRedisPayloadStoreRoundTripTTLNamespaceAndDelete`、`TestPromptRuntimeAggregatesConfigWorkersQueueRedisEndpointsAndGuardMetrics`（均未设置 `PROMPT_AUDIT_TEST_REDIS_ADDR`）、6 条 `TestPromptAuditMigrationSchemaAndLeakageGate`/`TestPromptAuditDatabasePersistsFullPromptOnEventsOnly`/`TestPromptAuditRepositoryAdmissionClaimFencingAndEventTransaction`/`TestPromptAuditRepositoryForeignKeysFiltersAndStableIdentitySnapshots`/`TestPromptAuditRepositoryHighWaterAndSafeDeletion`/`TestPromptAuditServiceConfirmationKeepsPostPreviewEventsAndConcurrentDeletesAreSafe`（均未设置 `PROMPT_AUDIT_TEST_POSTGRES_DSN`），以及 `TestEstimateOpenAIInputTokens_CompareWithOpenAIAPI`（未设置 `OPENAI_API_KEY`）；`TestDingTalkOAuthStart_Disabled` 是既有 disabled sentinel，不归入环境型；第 8427 行的 `TestConcurrencyCacheSuite/TestGetAccountsLoadBatch` 是无条件 TODO skip（`backend/internal/repository/concurrency_cache_integration_test.go:488-489`，`6d01be0c30` 引入，且未在 `2eb7ba771eb6d2cb195f267fba0af84056a83f39..HEAD` 改动）。后者的生产 `GetAccountsLoadBatch` 位于 default/layered scheduler 调用链（`openai_account_scheduler.go`、`openai_account_scheduler_layered.go`），所以是 scheduler 邻接的既有基线 TODO，不能称为环境型或“不命中 scheduler”；本阶段 scheduler quota/LastUsedAt、sticky、fallback/WaitPlan、DB recheck 聚焦目标均已 PASS。SKIP 与首次命令插值失败均保留为 concern。archive 已删除、远端唯一目录经 `test ! -e` 核验不存在；日志保留在 `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-v0.1.163-368822c706f84dee88147b896e6bed00-integration.log`。

## Task 19：关闭 v0.1.163 能力矩阵与证据（2026-07-28）

- 状态：`DONE_WITH_CONCERNS`。审查输入 HEAD 为 `a452e3fdd396e62d94766730e64868adc02be1dd`；只审查 OpenSpec 5.3，未开始 Task 20、未合入 `v0.1.164`，未改 `backend/cmd/server/VERSION`、未 push/tag/release/deploy。
- 结构化集合：`U=diff(v0.1.162^{}, v0.1.163^{})=171`，`M=diff(02abe1574^1, 02abe1574)=171`，`I=U intersect M=170`。`U-I={backend/cmd/server/VERSION}`，`M-I={backend/internal/service/openai_account_scheduler_layered.go}`；stage fix/evidence 范围 `02abe1574..a452e3fdd` 有 14 个 first-parent commits、29 条历史路径并集。精选 evidence paths 是 33 条单独标注的路径，绝不等同于 170 条 `I`。

| 能力 | 行为契约与实际入口/调用链 | 交集与补充证据 | 自动证据 | 人工审查点 | 结论 |
| --- | --- | --- | --- | --- | --- |
| reasoning policy | admin group 请求映射到 group 字段；OpenAI 请求在转发前执行精确 mapping 后上限裁剪。 | `I`：group schema、admin handler、gateway handler、policy/service test、migration 185。 | Task 19 的 `TestUpdateGroupRequestReasoningEffortMappingsTriState`、`TestApplyOpenAIReasoningEffortPolicy` PASS。 | 未识别值保持原样，符合 policy 的 omitted/unknown 契约。 | `protected` |
| scheduler quota metadata/LastUsedAt/sticky/fallback/WaitPlan/DB recheck | gateway handler -> `SelectAccountWithSchedulerForCapability` -> default/layered `Select` -> sticky classify/recheck -> acquire/fallback WaitPlan；cache 的 metadata 与 side-key 保留 quota/LastUsedAt。 | `I`：scheduler/cache/gateway sources and tests；`M-I`：`openai_account_scheduler_layered.go`；stage repairs 覆盖 default/layered sticky。 | Task 19 的 service/repository targets PASS。 | 64-probe ceiling 与 hard fallback 边界保持 Task 17 说明，不把它们写成无限候选保证。 | `protected` |
| shutdown/Cleanup/background drain | signal -> CloseAdmission -> graceful shutdown/force close -> handler drain -> `app.Cleanup()` -> producers -> usage -> ops/deferred -> quota -> billing -> Redis -> Ent。handler 在 15 秒内完成时先 drain 后 cleanup；hard deadline 后仍调用 `app.Cleanup()`。 | `I`：`main.go`；stage-only：`shutdown.go`、`wire.go`、lifecycle tests 和 deferred drain。 | Task 19 的 cmd/server targets 与 deferred target PASS。 | 仅 cleanup phase 失败或超时时跳过后续 cleanup phase；这是真实的有界停机取舍，不声称零丢失。 | `protected` |
| usage/billing/output estimate/cache/image | gateway handler -> Forward -> response usage parsing -> usage/billing apply; Anthropic fallback estimate 与 force-cache 分类、OpenAI hosted-image usage 合并和同步通知路径均可达。 | `I`：Anthropic/response handling/image tests；stage-only：usage billing lifecycle、cyber worker routing。 | Task 19 的 service/handler targets PASS。 | detached billing context 有 timeout；通知与 quota 更新在 usage task 中完成，不扩张为 delivery exactly-once 声明。 | `protected` |
| axios manifest/lockfile | release source declares `axios ^1.18.0`; frozen resolution installs and actually loads exact `1.18.1`. | `I`：`frontend/package.json`、`frontend/pnpm-lock.yaml`。 | Clean `git archive` of `dbb18b7059a5378411dc8c51fde570003dd072ba` started without `frontend/node_modules`; `pnpm --dir frontend install --frozen-lockfile` PASS, `pnpm --dir frontend list axios --depth 0` shows `axios 1.18.1`, `require('axios/package.json').version` is `1.18.1`, and the same install state passes `pnpm --dir frontend run build`. | Evidence is limited to that clean archive; full command log is retained in the Task 19 scratch report. | `protected` |
| migration 185/schema/runner | embedded migration -> sorted runner -> SHA-256/checksum record -> idempotent upgrade test. | `I`：migration 185; runner and integration test are context-only, not claimed as intersection. | Task 18 remote targets `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` and `TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS; Task 19 reconfirmed one 185 file and SHA-256 `940c1132c5a01f824a83439cce41e6a182d51c1a3c6357a96c76dba950fc307e`. | Task 18 local migration command was Docker-SKIP and is not reported as a local migration PASS. | `protected` |
- 前置已满足（矩阵外，不计能力行）：Task 17 final Sol `ses_05b7cc12dffev47d9cJ8EyBhba` PASS；Task 18 final Sol `ses_05b2d6856ffeslt1jo0WjqIoQj` PASS。
- Task 18 prerequisite recheck：remote migration two-target PASS；`185_group_reasoning_effort_policy.sql` is the unique `185_*.sql` and matches the recorded SHA-256; full gate reports `SKIP=13` as 11 environment/credential/optional-service skips, 1 disabled sentinel and 1 existing scheduler-adjacent TODO skip, with `0` FAIL. Local full gate recorded two stable generates, zero generated diff and `VERSION=0.1.159.6`.
- Task 19 did not use the full gate as capability proof: each `protected` row above has a current call-chain review plus named, narrow test evidence. Complete commands, raw path set, stage range, membership table and concerns are in uncommitted `.superpowers/sdd/task-19-report.md`.
- 汇总：`rows=6/protected=6/manual=0/gap=0/approved-removal=0`。clean archive build 仅有 Vite dynamic-import/chunk-size warnings，exit `0`；Task 18 local migration 仍为 Docker-SKIP，迁移执行证据来自其 remote two-target PASS。
