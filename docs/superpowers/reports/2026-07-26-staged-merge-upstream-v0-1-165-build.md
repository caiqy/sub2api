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
- 本任务仅初始化规划证据，不执行业务 TDD、本地门禁或远程 integration；这些项由后续 OpenSpec task 真实执行。
- 首次本地 full 门禁（Task 3）历史失败：`make test` 退出码 `2`。后端默认 `go test ./...`、`golangci-lint run ./...`（`0 issues.`）及 `go test -tags=unit ./...` 均完成；前端 `pnpm --dir frontend run lint:check` 在启动前失败，错误为 `process_begin: CreateProcess(NULL, pnpm --dir frontend run lint:check, ...) failed.` 和 `make (e=2): 系统找不到指定的文件。`，未产生失败测试名。
- 首次失败前置条件恢复：执行 `corepack enable pnpm` 后，自检 `Get-Command pnpm` 解析到 `C:\Users\caiqy\.version-fox\sdks\nodejs\pnpm.ps1`，`pnpm --version` 输出 `11.17.0`；未改变仓库文件。
- 完整重跑本地 full 门禁：再次执行 `make test`，退出码 `2`。后端默认测试、`golangci-lint run ./...`（`0 issues.`）和 unit 测试均通过；前端 `pnpm --dir frontend run lint:check` 触发依赖状态检查，因无 TTY 拒绝清理 modules 目录，报 `ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY`，内部 `pnpm install` 退出码 `1`，最终 `make: *** [test-frontend] Error 1`。这是新的首个未解决阻塞，未产生失败测试名。
- 第二次失败前置条件恢复：执行 `corepack install --global pnpm@9.15.9` 后，自检 `pnpm --version` 输出 `9.15.9`，`frontend/node_modules/.modules.yaml` 的 `packageManager: pnpm@9.15.9` 与其一致；未删除 `node_modules`，未修改仓库或 lockfile。
- 再次完整重跑：`make test` 退出码 `0`。后端默认测试、`golangci-lint run ./...`（`0 issues.`）、unit 测试、前端 lint 与 typecheck 均通过；前端 Vitest 为 `194 passed` 测试文件、`1493 passed` 用例。随后 `make build` 退出码 `2`：backend build 的 `./scripts/resolve-version.sh` 调用无法创建 `sh D:\Caiqy\Projects\Github\sub2api\backend\scripts\resolve-version.sh` 进程，继而 `CGO_ENABLED` 未被识别，最终 `make: *** [build-backend] Error 2`。这是新的首个未解决阻塞。
- 第三次失败前置条件恢复：每条门禁 PowerShell 进程均在执行前设置 `$env:PATH = "C:\Program Files\Git\bin;C:\Program Files\Git\usr\bin;" + $env:PATH`。自检 `Get-Command sh` 为 `C:\Program Files\Git\bin\sh.exe`、`sh --version` 为 GNU bash `5.2.37`，且 `sh backend/scripts/resolve-version.sh backend/cmd/server/VERSION` 输出 `0.1.159.6`、退出码 `0`；未修改系统 PATH 或仓库。
- 阶段 0 本地 full 门禁最终完整重跑（当前 HEAD：`aca233e82c08778e221a049d99a69aa02febaf87`）：`make test` 退出码 `0`，后端默认测试、`golangci-lint run ./...`（`0 issues.`）、unit 测试、前端 lint/typecheck 均通过，Vitest 为 `194 passed` 测试文件、`1493 passed` 用例；`make build` 退出码 `0`，后端按 `0.1.159.6` 构建，前端 Vite 处理 `987` 个模块并完成 production build；两次 `make -C backend generate` 均退出 `0`，每次后的 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 均退出 `0`、无输出，生成稳定。
- 静态和范围检查均通过：`git diff --check` 退出 `0`（仅 CRLF 自动转换提示，无空白错误）；两次 `git diff --name-only --diff-filter=U` 均退出 `0`、未合并文件集合为空；真实冲突标记扫描退出 `0`、集合为空；`git diff --name-only 'v0.1.159^{}..HEAD'` 退出 `0` 并列出本分支历史变更文件；最终 `backend/cmd/server/VERSION` 为 `0.1.159.6`。最终 `git status --short` 仅显示协调者既有 plan、OpenSpec `tasks.md`、`.comet/subagent-progress.md` 改动、本任务 ledger，以及未跟踪 `.comet/current-change.json`、`paseo.json`；后两者未触碰。
- 远程 integration：待执行（Task 4），本地门禁通过不构成远程放行。

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

- 阶段 0：待执行。
- `v0.1.160`：待执行。
- `v0.1.161`：待执行。
- `v0.1.162`：待执行。
- `v0.1.163`：待执行。
- `v0.1.164`：待执行。
- `v0.1.165`：待执行。

## 阻塞与残余风险

- 当前无隔离或工作树范围阻塞。
- 尚未执行任何 tag merge、changed-files 审查、能力矩阵填充、本地门禁或 `local-serv-ai` integration；在对应证据完成前不得进入下一阶段。
