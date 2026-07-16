---
change: staged-merge-upstream-v0-1-156
design-doc: docs/superpowers/specs/2026-07-16-staged-merge-upstream-v0-1-156-design.md
base-ref: d5f8192d32d9840d63477c24d4a567abb8cb4a90
---

# 分段合并上游 v0.1.156 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 从固定本地基线按顺序将上游 `v0.1.152`、`v0.1.153`、`v0.1.155`、`v0.1.156` 建立为四个独立的 `--no-ff` merge 节点，并用阶段 0 能力保护和逐段审查保留本地定制，唯一批准移除为最终阶段的本地首 Token 超时。

**架构：** 在隔离分支上先建立能力矩阵、冲突台账和可复现验证报告，再将每个正式 tag 合入。merge commit 只包含该 tag 的上游树和必要冲突融合；任何由测试或能力审查发现的语义修复均作为后续普通提交。每段完成保护测试、受影响能力测试和矩阵结论后才允许开始下一段。

**技术栈：** Git annotated tags、Go 1.25、Ent、Wire、Gin、Vue 3、Vite、pnpm、Vitest、golangci-lint。

## 全局约束

- Comet 原始起点为 `d1cc02502271f54b3b7f0593a18db4f2aaab63ea`；实施前实际基线为 `d5f8192d32d9840d63477c24d4a567abb8cb4a90`，两者之间仅包含已确认的本地测试临时目录隔离提交。最终上游祖先只能到 `v0.1.156^{}` 即 `12f991dde8a58e183d4bd16a87ef6fd0df714757`，绝不合入其后的 `upstream/main`。
- merge 顺序严格为 `v0.1.152`、`v0.1.153`、`v0.1.155`、`v0.1.156`；每个 tag 必须创建独立 `git merge --no-ff` 节点。
- 阶段 0 的基线、能力映射和所有必要补测未通过前，不得执行 `v0.1.152` merge；任何阶段门禁不通过，不得进入下一 tag。
- 冲突解决只进入对应 merge commit；merge 后发现的语义回归必须先保留失败证据，再用独立普通提交作最小修复。禁止机械 `--ours`、`--theirs` 或整文件覆盖。
- 除本地首 Token 超时外，本地能力不得静默丢失。无法与上游共存且未获批准时，停止在当前阶段并请求用户决定，不创建下一 merge 节点。
- 本地首 Token 超时在 `v0.1.152`、`v0.1.153`、`v0.1.155` 均为 `protected`；仅在 `v0.1.156` 后完整删除，不保留配置、错误、日志、watchdog、测试、文档或兼容别名。
- 不 push、release、deploy，不决定是否合回 `main`，不改写历史，不新增通用 merge 工具、测试框架或无关抽象。
- `tdd_mode=tdd` 仅约束真实行为修改：补测后需要生产代码修复、merge 后语义回归修复及超时能力移除等任务必须提供 RED/GREEN；纯文档、基线检查、能力映射和无行为修改的 tag merge 不伪造 RED，以对应命令证据代替。
- 执行中的证据统一写入 `docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`；该报告必须明确标出未执行命令，不能将未执行记为通过。

## 文件与证据结构

| 路径 | 执行时责任 |
| --- | --- |
| `docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md` | 固定 SHA、阶段 0 基线、能力矩阵、各段 changed-files 清单、冲突台账、失败证据、修复提交、门禁输出摘要、残余风险和未执行事项的唯一验证记录。 |
| `backend/ent/schema/`、`backend/ent/` | 仅当阶段 diff 或冲突触及 Ent 时，以 schema 为源执行生成并验证无丢失/稳定。 |
| `backend/cmd/server/wire.go`、`backend/cmd/server/wire_gen.go`、`backend/internal/**/wire.go` | 仅当阶段 diff 或冲突触及注入图时，以 Wire 声明为源执行生成并验证稳定，禁止手改 `wire_gen.go`。 |
| `backend/migrations/` | 仅当阶段 diff 或冲突触及 migration 时，复核文件名排序、同号文件、幂等 DDL 和 runner 语义。 |
| 阶段 0 映射发现的现有或新增 `backend/**/*_test.go`、`frontend/**/*.spec.ts` | 只为同时满足“本地独有、目标 release 触及、缺少行为断言”的能力补最小回归测试；具体路径必须由任务 1.3 的矩阵记录后确定，不能在映射完成前臆定。 |

## 固定对象与门禁命令

| 对象 | 固定值或命令 | 通过判定 |
| --- | --- | --- |
| Comet base | `d1cc02502271f54b3b7f0593a18db4f2aaab63ea` | 原始设计与 OpenSpec 事实源的固定起点。 |
| implementation base | `d5f8192d32d9840d63477c24d4a567abb8cb4a90` | 隔离工作区从此提交开始；相对 Comet base 仅多一个已确认的测试基础设施提交。 |
| `v0.1.152^{}` | `b73d8c3efe01a290eaaa9326b6e40ece02c67a0e` | 作为第一个 merge commit 的第二父。 |
| `v0.1.153^{}` | `a2bc1337474b68b62391116835e5698ebb5526bd` | 作为第二个 merge commit 的第二父。 |
| `v0.1.155^{}` | `41cec0db059ffb82d0efdcfcf07a24ab51fbfe97` | 作为第三个 merge commit 的第二父。 |
| `v0.1.156^{}` | `12f991dde8a58e183d4bd16a87ef6fd0df714757` | 作为第四个 merge commit 的第二父和最终上游边界。 |
| 后端/前端保护门禁 | `make test` | 根 `Makefile` 依次执行后端默认测试、unit-tag 测试、golangci-lint、前端 ESLint、typecheck、Vitest。 |
| 前端构建 | `pnpm --dir frontend run build` | `vue-tsc -b` 与 Vite build 退出码为 0。 |
| Ent/Wire 可复现性 | `make -C backend generate` 后重跑同一命令 | 第一次生成后无非预期 diff；第二次生成不再改变 `backend/ent/` 或 `backend/cmd/server/wire_gen.go`。 |

## 任务追踪

| OpenSpec task | 本计划任务 |
| --- | --- |
| 1.1 | 1 |
| 1.2 | 2 |
| 1.3 | 3 |
| 1.4 | 4 |
| 2.1-2.3 | 5-7 |
| 3.1-3.3 | 8-10 |
| 4.1-4.3 | 11-13 |
| 5.1-5.5 | 14-18 |
| 6.1-6.5 | 19-23 |

---

### Task 1：固定基线、tag 与隔离工作区（OpenSpec 1.1）

**文件：**
- 创建：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`
- 审查：`.git/` 引用、`docs/superpowers/reports/2026-07-13-merge-upstream-v0-1-151-validation.md`

**产物：** 报告的“固定对象与工作区”章节，记录隔离分支名、开始 `HEAD`、四个 tag object/peel SHA、`upstream/main` 相对 `v0.1.156^{}` 的 release 后范围，以及初始工作树状态。

- [x] **步骤 1：在用户确认的隔离 worktree/feature 分支中验证起点与工作树**

  执行：
  ```bash
  git status --short
  git rev-parse HEAD
  git merge-base --is-ancestor d1cc02502271f54b3b7f0593a18db4f2aaab63ea HEAD
  git log --oneline d1cc02502271f54b3b7f0593a18db4f2aaab63ea..HEAD
  ```
  预期：`HEAD` 为 `d5f8192d32d9840d63477c24d4a567abb8cb4a90`，或其后仅有本 change 的计划/报告/OpenSpec 协调提交；`d1cc02502..d5f8192d3` 仅有已确认的测试基础设施提交。出现其他业务文件或未授权改动时停止并由用户决定，不开始 merge。

- [x] **步骤 2：获取引用并固定 annotated tag 的 peel commit**

  执行：
  ```bash
  git fetch upstream --tags
  git rev-parse v0.1.152 "v0.1.152^{}"
  git rev-parse v0.1.153 "v0.1.153^{}"
  git rev-parse v0.1.155 "v0.1.155^{}"
  git rev-parse v0.1.156 "v0.1.156^{}"
  git log --oneline "v0.1.156^{}"..upstream/main
  ```
  预期：四个 peel SHA 分别等于全局表中的固定值；最后一条命令可列出 release 后提交，但这些提交只记录为排除范围，绝不作为 merge 目标。

- [x] **步骤 3：验证 Comet 已创建的隔离工作区并记录不可变证据**

  执行：
  ```bash
  git branch --show-current
  git status --short
  git rev-parse HEAD
  git show -s --format='%H%n%P%n%s' HEAD
  ```
  预期：当前分支或 worktree 已由 Comet 按用户选择创建并重新绑定 change，起点为 implementation base；报告记录隔离方式、分支名、命令输出、tag object 与 peel SHA。本任务不创建或切换分支。

- [x] **步骤 4：提交当前 change 的设计、计划与文档性起点记录**

  执行：
  ```bash
  git add openspec/changes/staged-merge-upstream-v0-1-156
  git add -f docs/superpowers/specs/2026-07-16-staged-merge-upstream-v0-1-156-design.md
  git add -f docs/superpowers/plans/2026-07-16-staged-merge-upstream-v0-1-156.md
  git add -f docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md
  git commit -m "docs: add staged upstream merge plan"
  ```
  预期：仅 Design Doc、实施计划、验证报告和 `openspec/changes/staged-merge-upstream-v0-1-156/` 协调文件进入此普通提交；不得包含 `.comet/current-change.json` 或业务代码。

### Task 2：运行阶段 0 基线与生成检查（OpenSpec 1.2）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`
- 审查：`Makefile`、`backend/Makefile`、`frontend/package.json`

**产物：** 报告的“阶段 0 基线”章节，按命令记录退出码、失败测试名或通过摘要、环境限制和生成 diff。

- [x] **步骤 1：执行当前本地质量门禁**

  执行：
  ```bash
  make test
  pnpm --dir frontend run build
  ```
  预期：两条命令均退出码 0。`make test` 覆盖 `go test ./...`、`golangci-lint run ./...`、`go test -tags=unit ./...`、前端 lint/typecheck/Vitest；build 单独覆盖前端嵌入产物。

- [x] **步骤 2：执行 Ent 与 Wire 两轮生成稳定性检查**

  执行：
  ```bash
  make -C backend generate
  git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
  make -C backend generate
  git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
  ```
  预期：两次检查均无 diff；若生成变更来自当前基线，先回到 schema/provider 源确定并提交可复现结果，再从步骤 1 重跑。

- [x] **步骤 3：处理阶段 0 阻塞**

  执行：
  ```bash
  git status --short
  git diff --check
  ```
  预期：报告精确记录全部失败命令和复现命令。任一基线测试、构建或生成检查失败时停止，禁止执行任务 5 的 merge；不得将既有失败归因于上游。

- [x] **步骤 4：提交阶段 0 基线证据**

  执行：
  ```bash
  git add -f docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md
  git commit -m "docs: record stage zero baseline"
  ```
  预期：仅提交验证报告；测试、构建或生成检查失败时仍可提交准确的阻塞证据，但任务保持未完成并停止后续 tag merge。

### Task 3：建立本地能力至验证证据的映射（OpenSpec 1.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`
- 审查：`openspec/specs/`、`knowledge-base/reference/capabilities-index.md`、`memory/context/upstream-merge-workflow.md`、`docs/superpowers/reports/2026-07-13-merge-upstream-v0-1-151-validation.md`

**产物：** 报告的“能力矩阵”，每一行具备能力名称、本地行为契约、入口与调用链、关键文件、受影响 tag、现有自动测试、人工审查点、状态、阶段结果和证据链接。

- [x] **步骤 1：导出本地独有提交和四段上游变更清单**

  执行：
  ```bash
  git log --format='%H %s' v0.1.151^{}..d5f8192d32d9840d63477c24d4a567abb8cb4a90
  git diff --name-only v0.1.151..v0.1.152
  git diff --name-only v0.1.152..v0.1.153
  git diff --name-only v0.1.153..v0.1.155
  git diff --name-only v0.1.155..v0.1.156
  ```
  预期：将每段输出原样附到报告的对应阶段 changed-files 小节；这是后续查找无文本冲突语义变化的唯一输入清单。

- [x] **步骤 2：用能力矩阵覆盖高风险本地行为**

  逐项记录以下能力，且为每项写出当前入口、调用链和现有测试命令：scheduler、OpenAI/Gemini/Anthropic Sticky、previous-response/session Sticky、fallback/WaitPlan、DB recheck、Messages/Responses/Chat 转换与透传字段、终止 usage、privacy/内容审计、image capability、运行时设置热更新、请求体重放与清理、用户资源控制、前端本地功能、版本/依赖、Ent/Wire 与 migrations。

  使用：
  ```bash
  git diff --name-only v0.1.151..v0.1.156
  git grep -n -E 'previous_response|sticky|WaitPlan|recheck|replay|cleanup|runtime|blocked.*group|purchase|custom.*menu' -- backend frontend
  ```
  并对变更入口使用 CodeGraph `context`、`impact` 或 `trace` 记录调用链；不以文件名匹配代替调用链审查。

- [x] **步骤 3：为每行分配唯一状态**

  判定：直接行为测试且阶段 0 通过为 `protected`；目标 release 触及但缺少关键行为断言为 `gap`；生成物、migration、版本依赖或跨层契约需结构审查为 `manual`；仅本地首 Token 超时预登记为 `approved-removal`，且阶段结果标为“仅 v0.1.156 后可执行”。

  预期：没有空状态或未归类高风险能力；`gap` 行成为任务 4 的唯一补测输入。

- [x] **步骤 4：提交能力矩阵与 changed-files 证据**

  执行：
  ```bash
  git add -f docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md
  git commit -m "docs: map local capability coverage"
  ```
  预期：仅提交验证报告；矩阵每行字段完整、状态唯一，所有 `gap` 均可直接作为 Task 4 输入。

### Task 4：补齐阶段 0 的最小行为保护测试（OpenSpec 1.4）

**文件：**
- 修改：任务 3 映射确定的准确 `backend/**/*_test.go` 或 `frontend/**/*.spec.ts`
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

**接口：** 消费矩阵中状态为 `gap` 的行及其已记录的入口、契约和目标 tag；产出可直接运行的测试命令和状态变更为 `protected` 的矩阵行。

- [x] **步骤 1：逐个 gap 写出基线行为断言**

  对每个 `gap`，将测试放在矩阵已记录的被测包/组件的既有测试目录；测试必须只断言该行已写明的本地行为，不扩展功能。报告为每行写入准确的既有 Go package 与测试名，或准确的 Vitest 文件路径；后端使用该行的 `go test` 聚焦命令，前端使用该行的 `pnpm --dir frontend exec vitest run` 聚焦命令。

  预期：报告列出每个准确测试文件、测试名、首个失败输出和它保护的能力；映射为零个 `gap` 时记录“无符合三条件的补测”，不创建无目的测试。

- [x] **步骤 2：运行每个新增测试，确认它保护当前本地基线**

  执行每条由矩阵生成的聚焦命令。预期：新增 characterization test 在当前基线通过，并在后续 tag 首次破坏该行为时提供失败证据；若当前基线失败，先诊断为既有阻塞，不得开始 merge。后续回归修复是否强制 Red-Green-Refactor 由用户在 Comet build 决策点选择的 `tdd_mode` 决定。

- [x] **步骤 3：重跑新增测试与阶段 0 门禁**

  执行：
  ```bash
  make test
  pnpm --dir frontend run build
  make -C backend generate
  git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
  ```
  预期：所有新增测试及阶段 0 门禁通过，矩阵行改为 `protected` 并链接命令输出；仍有 `gap`、失败或未解释生成 diff 时停止，禁止执行任务 5。

- [x] **步骤 4：提交阶段 0 测试与能力映射**

  执行：
  ```bash
  git add backend frontend
  git add -f docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md
  git commit -m "test: protect local behavior before upstream merge"
  ```
  若矩阵为零个 `gap`，不得暂存 `backend`/`frontend`，只暂存验证报告并使用 `git commit -m "docs: close stage zero protection gate"`。预期：提交只包含报告和实际必要的测试/最小修复；报告记录提交 SHA。

### Task 5：合入 v0.1.152 并记录冲突融合（OpenSpec 2.1）

**文件：**
- 修改：Git merge 涉及的实际文件；冲突文件在 merge 后由 Git 确定
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

**产物：** 第 1 阶段冲突台账，逐行包含路径、类别、ours 行为/调用链、theirs 行为、融合语义和验证命令。

- [x] **步骤 1：再次确认阶段 0 保护条件**

  执行：
  ```bash
  git status --short
  git log -1 --format='%H %s'
  git rev-parse "v0.1.152^{}"
  ```
  预期：工作树除本 change 预期文档外干净，阶段 0 全部通过且 tag peel SHA 为 `b73d8c3efe01a290eaaa9326b6e40ece02c67a0e`。

- [x] **步骤 2：创建第一个独立 merge 节点**

  执行：
  ```bash
  git merge --no-ff v0.1.152 -m "merge: upstream v0.1.152"
  ```
  预期：无冲突时直接创建 merge commit；有冲突时停留在 merge 状态，只能进入下一步骤，不得执行 `git merge --abort` 后改用 squash/cherry-pick。

- [x] **步骤 3：发现、分类并融合每个冲突**

  执行：
  ```bash
  git diff --name-only --diff-filter=U
  git diff --cached --check
  git diff --name-only v0.1.151..v0.1.152
  ```
  对每个未合并文件，分别阅读 merge 前第一父与 `v0.1.152^{}` 的行为和调用方，分类为上游修复、本地定制、接口/配置演进、版本依赖、生成代码或 migration；可共存时作最小融合。不可共存且不是首 Token 例外时停止并请求用户选择。

- [x] **步骤 4：完成 merge commit 并验证父节点**

  执行：
  ```bash
  git status --short
  git diff --cached --check
  git commit
  git show -s --format='%H%n%P%n%s' HEAD
  ```
  在 `git commit` 前，根据前一步 `git diff --name-only --diff-filter=U` 的精确冲突路径逐个执行 `git add -- 路径`，但不得暂存验证报告。预期：merge commit 的第二父为 `b73d8c3efe01a290eaaa9326b6e40ece02c67a0e`；该节点只包含上游树和必要冲突融合，不包含后续语义修复或台账文档。

- [x] **步骤 5：提交 v0.1.152 冲突台账与父节点证据**

  在 merge commit 完成后，将精确冲突路径（无冲突则明确写“无”）、类别、ours/theirs 行为、融合结论、验证方式、merge SHA 与两个父节点写入验证报告，然后执行：
  ```bash
  git add -f docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md
  git commit -m "docs: record v0.1.152 merge decisions"
  ```
  预期：文档提交紧随 merge commit，且不包含业务代码；merge 后语义回归仍留给 Task 6 的独立普通修复提交。

  若 Task 5 reviewer 在 merge 完成后发现语义回归，不改写 merge commit；将失败证据、TDD 修复和普通提交明确归入 Task 6。用户已确认 `b19c03d01` 与 `2026265cb` 的 Grok 默认 URL 修复按此规则作为 Task 6 提前执行项。

### Task 6：审查 v0.1.152 受影响能力并修复回归（OpenSpec 2.2）

**文件：**
- 修改：由 `git diff --name-only v0.1.151..v0.1.152` 与矩阵交集确定的实际文件
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：检查无文本冲突的能力交集和调用链**

  执行：
  ```bash
  git diff --name-only v0.1.151..v0.1.152
  git diff --check HEAD^1..HEAD
  git grep -n -E '^(<<<<<<< .+|=======|>>>>>>> .+)$' -- . ':!docs/superpowers/reports/**'
  ```
  预期：changed files 与矩阵关键文件相交的每项均有审查结论；入口、条件、DTO、配置解析、缓存、provider、schema 或生成结果被修改时，即使无冲突也必须记录调用链影响。

- [ ] **步骤 2：对首次回归采用失败测试驱动最小修复**

  先接收 Task 5 review 已发现并完成的 Grok 默认 URL TDD 修复（`b19c03d01`、`2026265cb`），核对其 RED/GREEN 与表单依赖，不重复实现。再运行对应 `protected` 测试以保留其他失败输出；缺少直接断言时按任务 4 的规则新增一个最小测试。仅修复当前 release 区间破坏的语义，不在 merge commit 内追加修复。

- [ ] **步骤 3：创建独立兼容修复提交（仅在存在修复时）**

  执行：
  ```bash
  git status --short
  git commit -m "fix: preserve local behavior after v0.1.152"
  ```
  在提交前，按报告中失败证据关联的精确修复/测试路径逐个执行 `git add -- 路径`，再暂存验证报告。预期：报告关联失败命令、修复测试和普通提交 SHA；无回归时不创建空提交。

- [ ] **步骤 4：独立提交 v0.1.152 能力审查结论**

  将 changed-files 与矩阵交集、调用链结论、Task 6 提前执行的 Grok 修复、全部聚焦命令/结果、修复提交和残余风险写入正式验证报告，然后执行：
  ```bash
  git add -f docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md
  git commit -m "docs: record v0.1.152 capability review"
  ```
  预期：文档提交不含业务代码；所有回归修复已在前置独立普通提交中完成，无未解释能力变化。

### Task 7：执行 v0.1.152 阶段门禁（OpenSpec 2.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：运行保护门禁、聚焦测试和冲突验证**

  执行：
  ```bash
  make test
  pnpm --dir frontend run build
  git diff --check
  git diff --name-only --diff-filter=U
  git grep -n -E '^(<<<<<<< .+|=======|>>>>>>> .+)$' -- . ':!docs/superpowers/reports/**'
  ```
  另运行矩阵中 `v0.1.152` changed-files 对应的聚焦测试和冲突台账逐行验证命令。

- [ ] **步骤 2：关闭第 1 阶段门禁**

  预期：所有保护测试、受影响能力测试和人工审查均有通过证据；`--diff-filter=U` 与冲突标记扫描无输出。失败时停留在 `v0.1.152`，修复后从本任务重跑，不得开始任务 8。

### Task 8：合入 v0.1.153 并复核已有融合（OpenSpec 3.1）

**文件：**
- 修改：Git merge 涉及的实际文件
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：确认第 1 阶段闭合并合入 tag**

  执行：
  ```bash
  git status --short
  git rev-parse "v0.1.153^{}"
  git merge --no-ff v0.1.153 -m "merge: upstream v0.1.153"
  ```
  预期：仅在任务 7 门禁闭合后执行；tag peel SHA 为 `a2bc1337474b68b62391116835e5698ebb5526bd`。

- [ ] **步骤 2：更新冲突台账并复核已确认融合**

  执行：
  ```bash
  git diff --name-only --diff-filter=U
  git diff --name-only v0.1.152..v0.1.153
  git diff --cached --check
  ```
  预期：重复冲突只可复用已验证的业务决策；必须重新检查当前 tag 的调用方和最终语义，禁止机械采用上次的 ours/theirs。

- [ ] **步骤 3：完成 v0.1.153 merge commit**

  执行：
  ```bash
  git status --short
  git commit
  git show -s --format='%H%n%P%n%s' HEAD
  ```
  在提交前，根据本段冲突台账中已发现的精确路径逐个执行 `git add -- 路径`。预期：第二父为 `a2bc1337474b68b62391116835e5698ebb5526bd`，且台账完整记录本段冲突与证据。

### Task 9：审查 v0.1.153 受影响能力并修复回归（OpenSpec 3.2）

**文件：**
- 修改：由 `git diff --name-only v0.1.152..v0.1.153` 与矩阵交集确定的实际文件
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：审查 changed files、入口和下游调用方**

  执行：
  ```bash
  git diff --name-only v0.1.152..v0.1.153
  git diff --check HEAD^1..HEAD
  ```
  对矩阵中被触及的能力执行 CodeGraph 调用链/影响分析，特别检查 scheduler、各平台 Sticky、fallback/WaitPlan、DB recheck、协议转换、runtime settings、body replay/cleanup 和用户资源控制。

- [ ] **步骤 2：保留失败证据后完成最小兼容修复**

  预期：回归先由现有保护测试或最小新增测试复现；普通修复提交不得夹带未被失败证据支持的重构。语义无法共存时停止并请求用户选择。

- [ ] **步骤 3：提交本段语义修复（仅在需要时）**

  执行：
  ```bash
  git status --short
  git commit -m "fix: preserve local behavior after v0.1.153"
  ```
  在提交前，按报告关联的精确修复/测试路径逐个执行 `git add -- 路径`，再暂存验证报告。预期：merge 节点与修复节点清晰可区分；无修复时仅更新报告。

### Task 10：执行 v0.1.153 阶段门禁（OpenSpec 3.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：运行完整保护门禁与 v0.1.153 定向验证**

  执行：
  ```bash
  make test
  pnpm --dir frontend run build
  git diff --check
  git diff --name-only --diff-filter=U
  ```
  另运行矩阵中受 `v0.1.152..v0.1.153` 影响的测试和本段冲突台账验证命令。

- [ ] **步骤 2：判定下一段入口**

  预期：所有命令通过、能力矩阵无未解释回归、冲突台账完整。失败、未执行或人工审查未完成均为阻塞，禁止执行任务 11。

### Task 11：合入 v0.1.155 并专项复核高风险区域（OpenSpec 4.1）

**文件：**
- 修改：Git merge 涉及的实际文件
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：合入 tag 并发现冲突**

  执行：
  ```bash
  git rev-parse "v0.1.155^{}"
  git merge --no-ff v0.1.155 -m "merge: upstream v0.1.155"
  git diff --name-only --diff-filter=U
  ```
  预期：tag peel SHA 为 `41cec0db059ffb82d0efdcfcf07a24ab51fbfe97`；任何冲突保持在此 merge 中等待逐文件融合。

- [ ] **步骤 2：融合冲突并专项检查网关、调度、设置、前端、生成物**

  执行：
  ```bash
  git diff --name-only v0.1.153..v0.1.155
  git diff --cached --check
  ```
  对上述五类区域逐项记录 ours/theirs 的行为、可共存结论、调用方、生成源或 migration runner。不可共存的未批准能力必须暂停；不以“编译通过”替代语义结论。

- [ ] **步骤 3：完成 v0.1.155 merge commit**

  执行：
  ```bash
  git status --short
  git commit
  git show -s --format='%H%n%P%n%s' HEAD
  ```
  在提交前，根据本段冲突台账中已发现的精确路径逐个执行 `git add -- 路径`。预期：第二父为 `41cec0db059ffb82d0efdcfcf07a24ab51fbfe97`，冲突解决未混入后续兼容修复。

### Task 12：审查 v0.1.155 受影响能力并修复回归（OpenSpec 4.2）

**文件：**
- 修改：由 `git diff --name-only v0.1.153..v0.1.155` 与矩阵交集确定的实际文件
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：执行无冲突语义审查和保护测试**

  执行：
  ```bash
  git diff --name-only v0.1.153..v0.1.155
  git diff --check HEAD^1..HEAD
  ```
  预期：每个 changed-file 与矩阵关键文件交集都有结论，尤其核查 gateway routes/handlers、scheduler/cache、settings DTO/cache、Vue 管理入口、Ent schema/provider 和 migration。

- [ ] **步骤 2：先失败后修复，并隔离普通提交**

  使用矩阵现有聚焦命令复现；无直接断言时新增最小行为测试。修复只处理这一段首次出现的回归，并记录修复前/后输出。

- [ ] **步骤 3：提交 v0.1.155 兼容修复（仅在需要时）**

  执行：
  ```bash
  git status --short
  git commit -m "fix: preserve local behavior after v0.1.155"
  ```
  在提交前，按报告关联的精确修复/测试路径逐个执行 `git add -- 路径`，再暂存验证报告。预期：没有回归时不制造空提交；报告仍完成该段审查结论。

### Task 13：执行 v0.1.155 阶段门禁（OpenSpec 4.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：运行保护门禁和高风险定向验证**

  执行：
  ```bash
  make test
  pnpm --dir frontend run build
  git diff --check
  git diff --name-only --diff-filter=U
  git grep -n -E '^(<<<<<<< .+|=======|>>>>>>> .+)$' -- . ':!docs/superpowers/reports/**'
  ```
  另运行第 3 阶段矩阵与冲突台账的全部聚焦命令。

- [ ] **步骤 2：关闭第 3 阶段**

  预期：报告中每个受影响能力有自动或人工证据，且无 unmerged/冲突标记。任何失败必须在 v0.1.155 后通过普通修复提交解决并从本任务重跑，禁止开始任务 14。

### Task 14：合入 v0.1.156 并审查上游超时语义（OpenSpec 5.1）

**文件：**
- 修改：Git merge 涉及的实际文件
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：创建最终 tag merge 节点**

  执行：
  ```bash
  git rev-parse "v0.1.156^{}"
  git merge --no-ff v0.1.156 -m "merge: upstream v0.1.156"
  git diff --name-only --diff-filter=U
  ```
  预期：tag peel SHA 为 `12f991dde8a58e183d4bd16a87ef6fd0df714757`；不得 merge `upstream/main`。

- [ ] **步骤 2：融合冲突并核实上游 HTTP/客户端 WebSocket 首输出语义**

  执行：
  ```bash
  git diff --name-only v0.1.155..v0.1.156
  git grep -n -E 'openai_first_output_timeout_seconds|openai_high_effort_first_output_timeout_seconds|first_output_timeout|HandleStreamTimeout' -- backend deploy frontend
  ```
  预期：台账记录 native HTTP Responses 的默认关闭、高 reasoning effort 覆盖、`first_output_timeout`、failover 与 `HandleStreamTimeout`，以及客户端 WebSocket 首消息超时；不得把已批准移除的“上游 WebSocket 首输出 watchdog”作为缺失回归。

- [ ] **步骤 3：完成 v0.1.156 merge commit**

  执行：
  ```bash
  git status --short
  git diff --cached --check
  git commit
  git show -s --format='%H%n%P%n%s' HEAD
  ```
  在提交前，根据本段冲突台账中已发现的精确路径逐个执行 `git add -- 路径`。预期：第二父为 `12f991dde8a58e183d4bd16a87ef6fd0df714757`；首 Token 本地能力的删除不夹在冲突融合中，而进入任务 15 的普通提交。

### Task 15：完整移除本地首 Token 超时并保留上游替代语义（OpenSpec 5.2）

**文件：**
- 修改：由任务 16 扫描确认的本地首 Token 超时后端、配置、运行时 DTO/API、管理端 UI、测试和文档实际路径
- 审查：`backend/internal/service/openai_first_output_timeout.go`、`backend/internal/handler/openai_gateway_first_output_timeout_test.go`、`backend/internal/service/openai_first_output_timeout_test.go`
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：建立删除清单与上游保留清单**

  执行：
  ```bash
  git grep -n -E 'openai_text_first_token_timeout|openai_image_first_token_timeout|first_token_timeout|openai_first_token_timeout|OpenAIFirstTokenTimeout|openAIFirstTokenWatchdog' -- backend frontend deploy docs openspec
  git grep -n -E 'openai_first_output_timeout_seconds|openai_high_effort_first_output_timeout_seconds|first_output_timeout|HandleStreamTimeout' -- backend deploy frontend
  ```
  预期：报告分别列出待删本地符号与必须保留/验证的上游符号，二者不得混淆。

- [ ] **步骤 2：先运行上游首输出测试并记录当前本地 watchdog 保护测试**

  执行：
  ```bash
  go -C backend test ./internal/service -run 'First(Output|Token)Timeout' -count=1
  go -C backend test ./internal/handler -run 'First(Output|Token)Timeout' -count=1
  ```
  预期：上游 `first_output` 覆盖能独立运行；本地 `first_token` 测试被登记为即将删除的 `approved-removal`，不能在前三阶段删除。

- [ ] **步骤 3：删除本地实现与全部暴露面**

  删除 HTTP SSE 与 WebSocket 上游首输出 watchdog、文本 30 秒/明确生图 600 秒分档、两项本地 runtime setting、持久化/DTO/API/UI、`first_token_timeout` 错误、失败 usage、Ops/结构化日志、本地专用测试及已失效文档。保留上游 native HTTP 首输出超时、failover/账号超时处理、客户端 WebSocket 首消息超时和既有读写超时。

- [ ] **步骤 4：提交完整移除作为独立普通提交**

  执行：
  ```bash
  git status --short
  git commit -m "refactor: remove local first token timeout"
  ```
  在提交前，根据任务 15 的删除/保留清单逐个执行 `git add -- 路径`，再暂存验证报告。预期：提交位于 v0.1.156 merge commit 之后；报告把矩阵中的该能力改为 `approved-removal`，并记录用户已批准的 WebSocket 上游 watchdog 缺口。

### Task 16：扫描旧首 Token 符号并消除依赖残骸（OpenSpec 5.3）

**文件：**
- 修改：任务 15 删除后仍引用旧符号的实际编译、契约或文档文件
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：执行残留符号和冲突扫描**

  执行：
  ```bash
  git grep -n -E 'openai_text_first_token_timeout|openai_image_first_token_timeout|first_token_timeout|gateway\.openai_first_token_timeout|OpenAIFirstTokenTimeout|openAIFirstTokenWatchdog' -- . ':!docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md'
  git grep -n -E '^(<<<<<<< .+|=======|>>>>>>> .+)$' -- . ':!docs/superpowers/reports/**'
  ```
  预期：第一条无输出；第二条无真实冲突标记。若 OpenSpec 历史规格保留旧术语，只能在报告中作为历史证据，不得成为业务代码兼容别名。

- [ ] **步骤 2：修复删除造成的编译或契约引用**

  只删除/改写依赖旧本地符号的调用以指向上游已存在的语义；不得重新引入旧键、旧错误、旧日志事件或 watchdog。若调用方要求的行为上游无法提供且不属于已批准删除范围，停止请求用户决定。

- [ ] **步骤 3：运行定向编译与上游超时测试**

  执行：
  ```bash
  go -C backend test ./internal/service -run 'FirstOutputTimeout|HandleStreamTimeout' -count=1
  go -C backend test ./internal/handler -run 'FirstOutputTimeout|HandleStreamTimeout' -count=1
  pnpm --dir frontend run typecheck
  ```
  预期：上游超时测试和前端契约通过；报告保留扫描输出和命令结果。

### Task 17：审查 v0.1.156 的其余本地能力并修复回归（OpenSpec 5.4）

**文件：**
- 修改：由 `git diff --name-only v0.1.155..v0.1.156` 与矩阵交集确定的实际文件，排除已批准删除的首 Token 本地能力
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：区分已批准移除与必须保留的受影响能力**

  执行：
  ```bash
  git diff --name-only v0.1.155..v0.1.156
  git diff --check HEAD^1..HEAD
  ```
  预期：能力矩阵中只有本地首 Token 超时可标为 `approved-removal`；其余每个 changed-file 交集必须保持 `protected` 或 `manual` 并具备证据。

- [ ] **步骤 2：先保留失败证据，再作最小兼容修复**

  运行受影响能力的保护测试和调用链审查。任何 scheduler、Sticky、fallback、DB recheck、转换/透传、privacy、image、热更新、body 生命周期、用户资源控制或前端本地功能回归都须先失败后修复；不可共存时停止等待用户。

- [ ] **步骤 3：提交本段非超时语义修复（仅在需要时）**

  执行：
  ```bash
  git status --short
  git commit -m "fix: preserve local behavior after v0.1.156"
  ```
  在提交前，按报告关联的精确修复/测试路径逐个执行 `git add -- 路径`，再暂存验证报告。预期：与任务 15 的删除提交分离；无修复时不创建空提交。

### Task 18：执行 v0.1.156 阶段门禁（OpenSpec 5.5）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：运行保护门禁、上游超时测试和本段定向验证**

  执行：
  ```bash
  make test
  pnpm --dir frontend run build
  go -C backend test ./internal/service -run 'FirstOutputTimeout|HandleStreamTimeout' -count=1
  go -C backend test ./internal/handler -run 'FirstOutputTimeout|HandleStreamTimeout' -count=1
  ```
  另运行矩阵中 `v0.1.155..v0.1.156` 对应能力的聚焦测试。

- [ ] **步骤 2：关闭最终 tag 阶段**

  预期：所有本地保护测试和上游超时测试通过，旧首 Token 扫描为空，能力矩阵没有未解释回归。任何失败均阻塞任务 19-23 的最终收口。

### Task 19：复核生成物、元数据、依赖与 migrations（OpenSpec 6.1）

**文件：**
- 审查：`backend/cmd/server/VERSION`、`backend/go.mod`、`backend/go.sum`、`frontend/package.json`、`pnpm-lock.yaml`、`deploy/config.example.yaml`、`backend/ent/schema/`、`backend/cmd/server/wire.go`、`backend/migrations/`
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：验证生成源与输出稳定**

  执行：
  ```bash
  make -C backend generate
  git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
  make -C backend generate
  git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
  ```
  预期：两轮无非预期 diff。若有差异，回到 schema/provider/Wire 声明修复，绝不手工编辑生成物。

- [ ] **步骤 2：审查版本、依赖、默认配置和 migration runner**

  执行：
  ```bash
  git diff --name-status v0.1.151..HEAD -- backend/cmd/server/VERSION backend/go.mod backend/go.sum frontend/package.json pnpm-lock.yaml deploy/config.example.yaml backend/migrations
  git ls-tree -r --name-only HEAD backend/migrations
  git grep -n -E 'Sort|sort\.String|filename' -- backend/internal/repository backend/migrations
  ```
  预期：报告逐项说明版本和依赖来自目标 release 或必要兼容修复；migration 以完整文件名排序/去重，所有同号文件、幂等性和 runner 顺序都有人工结论。

### Task 20：逐项完成完整能力矩阵审查（OpenSpec 6.2）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`
- 审查：任务 3 中每个能力行记录的实际入口、调用链和测试文件

- [ ] **步骤 1：完成关键能力的最终结论**

  对 scheduler、各平台 Sticky、fallback/WaitPlan、DB recheck、网关转换与透传、终止 usage、privacy、image capability、运行时热更新、请求体重放/清理、用户资源控制和前端本地功能，逐项核验：入口仍可到达、边界条件仍成立、自动或人工证据可复现、每个 release 阶段结果明确。

- [ ] **步骤 2：完成生成与发布相关结论**

  对版本、Go/前端依赖、配置默认值、Ent、Wire、migration 逐项链接任务 19 的命令和人工审查。将本地首 Token 超时写为唯一 `approved-removal`，写明上游 HTTP/客户端 WebSocket 替代行为和已批准不保留的上游 WebSocket 首输出 watchdog。

- [ ] **步骤 3：处理未批准的能力冲突**

  预期：没有“未解释”或“默认接受”结论。发现不可共存的未批准语义时，停止并向用户提交行为差异和可选保留策略；不得以完成最终审查为由继续。

### Task 21：执行最终自动验证与工作树静态检查（OpenSpec 6.3）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：运行完整门禁和生成复验**

  执行：
  ```bash
  make test
  pnpm --dir frontend run build
  make -C backend generate
  git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
  ```
  预期：后端默认/unit/lint、前端 lint/typecheck/Vitest/build 与生成复验全部通过。

- [ ] **步骤 2：检查 diff、未合并文件和冲突标记**

  执行：
  ```bash
  git diff --check
  git diff --name-only --diff-filter=U
  git grep -n -E '^(<<<<<<< .+|=======|>>>>>>> .+)$' -- . ':!docs/superpowers/reports/**'
  git status --short
  ```
  预期：无 whitespace 错误、无 unmerged、无真实冲突标记；工作树只保留报告和本 change 任务允许的预期文档变更。环境限制必须逐条记录为未执行，不得标记通过。

### Task 22：验证 Git 拓扑、阶段顺序和最终审查边界（OpenSpec 6.4）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`

- [ ] **步骤 1：验证四个 target tag 均为结果祖先**

  执行：
  ```bash
  git merge-base --is-ancestor b73d8c3efe01a290eaaa9326b6e40ece02c67a0e HEAD
  git merge-base --is-ancestor a2bc1337474b68b62391116835e5698ebb5526bd HEAD
  git merge-base --is-ancestor 41cec0db059ffb82d0efdcfcf07a24ab51fbfe97 HEAD
  git merge-base --is-ancestor 12f991dde8a58e183d4bd16a87ef6fd0df714757 HEAD
  git log --first-parent --merges --format='%H %P %s' d1cc02502271f54b3b7f0593a18db4f2aaab63ea..HEAD
  ```
  预期：四条祖先检查均退出码 0；first-parent merge 历史依次显示 `v0.1.152`、`v0.1.153`、`v0.1.155`、`v0.1.156`，每个节点的第二父为对应固定 peel SHA。

- [ ] **步骤 2：确认最终范围没有 release 后上游提交**

  执行：
  ```bash
  git log --oneline v0.1.156^{}..HEAD --ancestry-path
  git merge-base --is-ancestor upstream/main HEAD
  ```
  预期：第一条只列出 v0.1.156 merge 节点及之后的本地兼容/报告提交；第二条退出码必须为非零，确认完整 `upstream/main` 不是结果祖先，并将该预期非零结果作为范围证据记录，而不是测试失败。

- [ ] **步骤 3：完成 thorough review 的提交边界核对**

  执行：
  ```bash
  git log --first-parent --format='%H %P %s' d1cc02502271f54b3b7f0593a18db4f2aaab63ea..HEAD
  git status --short
  ```
  预期：每段 merge 与其后普通兼容修复可区分，未发生 rebase、squash 或机械策略覆盖；不进行 merge-to-main、push、release、deploy 决策。

### Task 23：完成验证报告与 OpenSpec 任务收口（OpenSpec 6.5）

**文件：**
- 修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`
- 修改：`openspec/changes/staged-merge-upstream-v0-1-156/tasks.md`

- [ ] **步骤 1：完成验证报告的可追溯记录**

  报告必须包含：固定 base/tag SHA、每段 changed-files、冲突台账、每个 merge/普通修复提交 SHA、阶段 0 与四段门禁命令/结果、能力矩阵、首 Token 的明确移除范围、上游保留语义、未执行事项和残余风险。不得将发布、部署、推送或合回 main 写成已完成。

- [ ] **步骤 2：逐项勾选 OpenSpec 的 23 项任务**

  仅当报告已链接对应证据时，将 `tasks.md` 的 1.1 至 6.5 全部标记完成。预期：任务数量为 23，任一缺证据条目保持未完成并回到对应任务处理。

- [ ] **步骤 3：提交最终验证文档，不进入发布流程**

  执行：
  ```bash
  git add openspec/changes/staged-merge-upstream-v0-1-156/tasks.md
  git add -f docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md
  git commit -m "docs: verify staged upstream merge v0.1.156"
  git status --short
  ```
  预期：最终文档提交与业务 merge/修复节点分离；不执行 `git push`、release、deploy 或合入 `main`。

## 实施前后复核

- 阶段 0 是硬门禁：任何 `make test`、`pnpm --dir frontend run build`、Ent/Wire 稳定性或能力矩阵状态失败都阻止首次 merge。
- 对 merge 后才出现的冲突，使用 `git diff --name-only --diff-filter=U` 发现并写入固定验证报告；计划不预设冲突文件，也不允许省略类别、双方语义、融合结论和验证证据。
- 对无冲突变化，使用每段前一 tag 到当前 tag 的 `git diff --name-only` 输出与能力矩阵关键文件的交集，加上 CodeGraph 调用链审查；无交集不能替代对共享入口/配置/生成物的人工检查。
- 所有后续普通提交必须引用当前 tag 阶段的失败证据；没有回归则不创建空的兼容修复提交。
- 最终只在任务 23 更新 OpenSpec 完成状态；该任务不归档 change、不执行发布或部署。
