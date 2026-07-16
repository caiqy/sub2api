# 上游 v0.1.156 分段合并验证记录

## 固定对象与工作区

- 隔离方式：当前仓库中的 feature 分支；本任务未创建、切换或合并分支。
- 隔离分支：`feature/20260716/staged-merge-upstream-v0-1-156`。
- 开始 `HEAD`：`d5f8192d32d9840d63477c24d4a567abb8cb4a90`。
- `HEAD` 父提交：`d1cc02502271f54b3b7f0593a18db4f2aaab63ea`。
- `HEAD` 主题：`test: isolate Go test temporary files`。
- `d1cc02502..HEAD`：仅该已确认的测试基础设施提交；差异文件为 `.gitignore`、`backend/Makefile`、`backend/scripts/test.ps1`，没有本次业务合并变更。

| Tag | Annotated tag object | Peel commit |
| --- | --- | --- |
| `v0.1.152` | `553ab6f911247963eb368fcf6ac1dcb65d5495b1` | `b73d8c3efe01a290eaaa9326b6e40ece02c67a0e` |
| `v0.1.153` | `53717a125583e3916b751c2a5340901c4bfa2bb3` | `a2bc1337474b68b62391116835e5698ebb5526bd` |
| `v0.1.155` | `ec4a37da4f023fbaa4d46d2ee46a6e7f22e313d4` | `41cec0db059ffb82d0efdcfcf07a24ab51fbfe97` |
| `v0.1.156` | `9cc1b469a24e6f79aeec9401ad1f9534f9b98aec` | `12f991dde8a58e183d4bd16a87ef6fd0df714757` |

- `git fetch upstream --tags` 后，`upstream/main` 从 `807850769` 更新至 `09c6c6d74`。
- 排除范围：`git log --oneline "v0.1.156^{}"..upstream/main` 仅用于记录 release 后上游历史；输出起点为 `09c6c6d74 Merge pull request #4387 from yardbirds0/feat/upstream-rate-scheduling`，尾部为 `75fb3c41c fix(apicompat): responses->chat ...`。未将该范围或 `upstream/main` merge 到当前分支。

## 初始工作树

`git status --short` 输出仅为：

```text
?? .comet/current-change.json
?? openspec/changes/staged-merge-upstream-v0-1-156/
```

`.comet/current-change.json` 保持未暂存、未提交。`openspec/changes/staged-merge-upstream-v0-1-156/` 是本任务允许提交的协调产物目录。

## 执行命令与结果

| 命令 | 关键输出 | 退出状态 |
| --- | --- | --- |
| `git status --short` | 仅初始工作树章节所列两个未跟踪路径 | 0 |
| `git rev-parse HEAD` | `d5f8192d32d9840d63477c24d4a567abb8cb4a90` | 0 |
| `git merge-base --is-ancestor d1cc02502271f54b3b7f0593a18db4f2aaab63ea HEAD` | 无输出，祖先关系成立 | 0 |
| `git log --oneline d1cc02502271f54b3b7f0593a18db4f2aaab63ea..HEAD` | `d5f8192d3 test: isolate Go test temporary files` | 0 |
| `git diff --name-status d1cc02502271f54b3b7f0593a18db4f2aaab63ea..HEAD` | 仅 `.gitignore`、`backend/Makefile`、`backend/scripts/test.ps1` | 0 |
| `git fetch upstream --tags` | `upstream/main`：`807850769..09c6c6d74` | 0 |
| `git rev-parse v0.1.152 "v0.1.152^{}"` | object/peel 与固定对象表一致 | 0 |
| `git rev-parse v0.1.153 "v0.1.153^{}"` | object/peel 与固定对象表一致 | 0 |
| `git rev-parse v0.1.155 "v0.1.155^{}"` | object/peel 与固定对象表一致 | 0 |
| `git rev-parse v0.1.156 "v0.1.156^{}"` | object/peel 与固定对象表一致 | 0 |
| `git log --oneline "v0.1.156^{}"..upstream/main` | 仅记录 release 后排除范围，未 merge | 0 |
| `git branch --show-current` | `feature/20260716/staged-merge-upstream-v0-1-156` | 0 |
| `git show -s --format='%H%n%P%n%s' HEAD` | `HEAD`、父提交和主题与本报告一致 | 0 |

## 提交与自审

- 首次协调提交 SHA：`3877dc247ea58ef2194051399db3e67974d68473`，message 为 `docs: add staged upstream merge plan`。本报告更正后另行创建普通文档提交，不在本次提交中记录其自身 SHA。
- 变更文件：3 个 `docs/superpowers/{specs,plans,reports}/2026-07-16-staged-merge-upstream-v0-1-156*` 文档，以及 `openspec/changes/staged-merge-upstream-v0-1-156/` 下 19 个协调文件，共 22 个新增文件。
- 暂存自审：`git diff --cached --check` 退出 0；`git diff --cached --name-only -- .comet/current-change.json .superpowers` 无输出。
- 提交自审：首次提交的 `git show --name-status --format=fuller` 仅列出上述 22 个允许路径；根目录 `.comet/current-change.json` 保持未跟踪，未提交 `.superpowers/` 或业务代码。
- 事实自审：分支、开始 `HEAD`、父提交、四个 tag object/peel SHA 与 brief 完全一致；没有执行 `git merge`、分支切换、业务代码修改、测试、push、release、deploy 或 main 合并。未勾选计划或 OpenSpec task。

## 顾虑

- `upstream/main` 在 fetch 时前进至 `09c6c6d74`，其相对 `v0.1.156^{}` 的完整范围只作记录；后续四个 tag 分段 merge 必须继续以本报告固定的 tag peel commit 为目标。
- 本任务依用户裁决只核验 Git/工作树证据，不运行或伪造 RED/GREEN 测试。
- 协调产物为 22 文件、2300 行，超过 200 行风险阈值；均为本次既有设计、计划、OpenSpec/Comet 协调内容。
- 暂存时 Git 提示这些文档的工作副本下次被 Git 触及时可能发生 LF/CRLF 工作树转换；本次 `git diff --cached --check` 通过。

## 阶段 0 基线（OpenSpec 1.2）

**结论：PASS。** 本阶段未执行任何 tag merge、业务代码修改、测试补齐、push、release、deploy 或 main 合并；不勾选 plan/OpenSpec task。

### 执行环境与范围

- 工作目录：`D:\Caiqy\Projects\Github\sub2api`（Windows）。
- 质量门禁定义：根目录 `Makefile` 的 `test` 依次运行后端默认测试（含 `golangci-lint`）、后端 `unit` tag 测试、前端 ESLint、`vue-tsc --noEmit` 与 Vitest。
- 前端构建定义：`frontend/package.json` 的 `build` 为 `vue-tsc -b && vite build`，产物写入 `backend/internal/web/dist`。
- 生成定义：`backend/Makefile` 的 `generate` 依次执行 `go generate ./ent` 与 `go generate ./cmd/server`；检查范围严格限制为 `backend/ent` 和 `backend/cmd/server/wire_gen.go`。

### 命令与结果

| 阶段 | 命令 | 退出码 | 摘要 |
| --- | --- | --- | --- |
| 初始工作树 | `git status --short` | 0 | `docs/superpowers/plans/2026-07-16-staged-merge-upstream-v0-1-156.md`、`openspec/changes/staged-merge-upstream-v0-1-156/.comet/subagent-progress.md` 为修改；`.comet/current-change.json` 未跟踪。均为主会话协调状态，未触碰。 |
| 初始静态检查 | `git diff --check` | 0 | 无空白错误；对上述两份既有协调文件提示下次 Git 写入会将 LF 转为 CRLF。 |
| 质量门禁 | `make test` | 0 | 后端默认测试、`unit` tag 测试与 `golangci-lint` 通过；前端 ESLint、类型检查通过；Vitest 为 167 个文件、1246 个测试通过。 |
| 前端嵌入构建 | `pnpm --dir frontend run build` | 0 | `vue-tsc -b` 与 Vite 生产构建通过，966 个模块完成转换，构建耗时 40.52 秒。 |
| 第 1 轮生成 | `make -C backend generate` | 0 | Ent 与 Wire 生成完成；Wire 写入 `backend/cmd/server/wire_gen.go`。 |
| 第 1 轮生成稳定性 | `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` | 0 | 无输出；生成目标相对当前基线无 diff。 |
| 第 2 轮生成 | `make -C backend generate` | 0 | Ent 与 Wire 再次生成完成。 |
| 第 2 轮生成稳定性 | `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` | 0 | 无输出；两轮生成稳定。 |
| 最终工作树 | `git status --short` | 0 | 与初始工作树相同，未出现生成目标或业务代码变更。 |
| 最终静态检查 | `git diff --check` | 0 | 无空白错误；仅重复既有协调文件的 LF/CRLF 警告。 |

### 警告与风险信号

- `make test` 的 Vitest 输出有预期错误路径日志、`router-link` 解析警告与 `intlify` message compiler 警告；所有断言通过，命令退出 0。
- 测试和构建均提示 `caniuse-lite` 浏览器数据已 7 个月未更新；这是依赖数据维护信号，未阻塞本阶段。
- Vite 报告多个动态/静态混用导入，且 `AccountsView` 压缩后为 635.06 kB，超过 500 kB chunk 警戒线；构建成功，但后续性能工作应单独处理，不能归因于本阶段。
- 生成检查仅覆盖 brief 规定的 Ent 与 Wire 目标；构建产物由 Git 忽略，最终 `git status --short` 未显示其变更。

### 自审与提交

- 自审：未改动业务代码、测试、生成源码或主会话的 plan/`.comet/subagent-progress.md`；两轮受限 diff 均为空，所有基线命令退出 0，因此没有触发阻塞或根因调查流程。
- 仅暂存本报告：`git add -f docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`。
- 提交命令：`git commit -m "docs: record stage zero baseline"`；提交内容只允许为本报告。`.comet/current-change.json`、`.superpowers/` 以及主会话协调文件必须保持未暂存、未提交。
