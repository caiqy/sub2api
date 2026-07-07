---
change: merge-upstream-v0-1-146
design-doc: docs/superpowers/specs/2026-07-07-merge-upstream-v0-1-146-design.md
base-ref: e378b33f60f2202d80cbf9b1c11cee4e4ddb9dc3
---

# Merge Upstream v0.1.146 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在临时分支上合并 upstream release tag `v0.1.146`，保留本地定制能力并完成自动验证与专项 review。

**Architecture:** 先刷新 upstream refs/tags 并确认目标 tag，再从当前 `main` 的 `e378b33f60f2202d80cbf9b1c11cee4e4ddb9dc3` 创建隔离分支执行 merge。合并收敛后只修复本次 upstream 合并引入或暴露的问题，最后用后端测试、前端 typecheck/build 和本地关键能力 review 判断是否可进入收尾决策。

**Tech Stack:** Git, Go 1.26.4, Gin, Ent, Wire, Vue 3, TypeScript, pnpm, Vite

## Global Constraints

- 合并目标必须是 upstream release tag `v0.1.146`，不是 `upstream/main`。
- 基线必须是 `e378b33f60f2202d80cbf9b1c11cee4e4ddb9dc3`。
- 只在隔离分支或 worktree 执行合并，不直接合回 `main`，不推送远端。
- 不新增业务 capability、public API 或数据库 schema。
- 不做无关重构，不修复与本次 upstream 合并无关的历史问题。
- 冲突处理优先保留 upstream 修复和本地定制；不可共存语义必须暂停让用户确认。
- 可以按 Comet build 规则在隔离分支提交本 change 的计划、合并结果和必要修复；不合回 `main`，不推送远端。

---

## 文件结构与复核边界

- `backend/cmd/server/VERSION`：版本号复核，预期对齐 upstream `v0.1.146`。
- `backend/go.mod`、`backend/go.sum`：后端依赖与 Go 版本复核。
- `backend/ent/**`、`backend/migrations/**`、`backend/cmd/server/wire_gen.go`：生成文件与 migration 复核；能重生成的生成物以源码重生成结果为准。
- `backend/internal/service/**`、`backend/internal/handler/**`、`backend/internal/repository/**`：网关、调度、sticky、privacy、image capability、runtime setting 热更新和透传字段专项 review 入口。
- `frontend/package.json`、`frontend/pnpm-lock.yaml`、`frontend/src/**`：前端依赖、类型、构建和 UI 接线复核入口。
- `docs/superpowers/plans/2026-07-07-merge-upstream-v0-1-146.md`：执行过程中只更新 checklist 和结果记录，不写业务逻辑。

### Task 1: 获取 upstream refs/tags 并确认目标

**Files:**
- Modify: `docs/superpowers/plans/2026-07-07-merge-upstream-v0-1-146.md`

**Interfaces:**
- Consumes: 本地 git remote `upstream=https://github.com/Wei-Shaw/sub2api`
- Produces: 可用于 merge 的本地 tag/ref `refs/tags/upstream-v0.1.146` 或已确认安全的 `refs/tags/v0.1.146`

- [x] **Step 1: 确认当前基线和工作区状态**

Run:

```bash
git branch --show-current
git status --short --branch
git rev-parse HEAD
git remote -v
```

Expected: 当前分支是 `main`；`HEAD` 是 `e378b33f60f2202d80cbf9b1c11cee4e4ddb9dc3`；没有业务代码脏改；`upstream` 指向 `https://github.com/Wei-Shaw/sub2api`。

- [x] **Step 2: 刷新 upstream 分支和目标 tag**

Run:

```bash
git fetch upstream --prune
git fetch upstream refs/tags/v0.1.146:refs/tags/upstream-v0.1.146 --no-tags
```

Expected: `upstream/main` 刷新成功；本地出现 `refs/tags/upstream-v0.1.146`。如果 tag 已存在且对象相同，记录为已存在；如果对象不同，暂停让用户确认，不覆盖本地 tag。

- [x] **Step 3: 确认 `v0.1.146` 指向真实 upstream release**

Run:

```bash
git rev-parse upstream-v0.1.146^{commit}
git ls-remote --tags upstream refs/tags/v0.1.146
git log --oneline -1 upstream-v0.1.146
```

Expected: 本地 `upstream-v0.1.146` 与 `git ls-remote` 返回的 upstream tag commit 一致；记录 commit hash 和标题。

### Task 2: 确认隔离合并分支

**Files:**
- Modify: `docs/superpowers/plans/2026-07-07-merge-upstream-v0-1-146.md`

**Interfaces:**
- Consumes: Task 1 确认的 `main` 基线、用户在 Comet build 决策点确认的隔离方式和 `upstream-v0.1.146`
- Produces: 用户确认的隔离分支或 worktree 工作区

- [x] **Step 1: 再次确认隔离基线**

Run:

```bash
git rev-parse HEAD
git status --short --branch
```

Expected: 当前工作区已处于用户在 Comet build 决策点确认的隔离分支或 worktree；`HEAD` 仍是 `e378b33f60f2202d80cbf9b1c11cee4e4ddb9dc3`；没有未预期业务代码脏改。若仍在 `main`，先按 Comet build 的隔离分支确认结果创建分支；若存在用户未提交业务改动，暂停确认，不覆盖。

- [x] **Step 2: 记录隔离分支现场**

Run:

```bash
git branch --show-current
git log --oneline --decorate -1
```

Expected: 当前分支或 worktree 是用户确认的隔离执行位置，且基于本计划 base-ref。

- [x] **Step 3: 记录分支现场**

Run:

```bash
git status --short --branch
git log --oneline --decorate -1
```

Expected: 当前位于用户确认的隔离执行位置；HEAD 仍指向 `e378b33f60f2202d80cbf9b1c11cee4e4ddb9dc3`。

### Task 3: 合并 upstream `v0.1.146`

**Files:**
- Modify: `backend/**`
- Modify: `frontend/**`
- Modify: `deploy/**`
- Modify: `.github/**`
- Modify: repository root metadata files changed by upstream release
- Modify: `docs/superpowers/plans/2026-07-07-merge-upstream-v0-1-146.md`

**Interfaces:**
- Consumes: 用户确认的隔离分支或 worktree，以及 tag `upstream-v0.1.146`
- Produces: 一个未提交或已收敛的 merge 工作区，无冲突标记

- [x] **Step 1: 执行不自动提交的 merge**

Run:

```bash
git merge --no-commit --no-ff upstream-v0.1.146
```

Expected: 进入 merge 现场；如果无冲突，仍保持未提交状态用于复核；如果有冲突，进入下一步。

- [x] **Step 2: 列出冲突和自动合并文件**

Run:

```bash
git diff --name-only --diff-filter=U
git diff --name-only
git status --short
```

Expected: 得到硬冲突清单和所有变更清单；把关键冲突文件记录到本计划的执行记录区。

- [x] **Step 3: 按三类规则处理冲突**

Action:

```text
可共存：手工融合 upstream 更新和本地定制。
需要小修：只修复本次合并引入或暴露的编译、测试、类型或语义问题。
不可共存：停止编辑，向用户列出选项和影响，等待确认。
```

Expected: 没有 `<<<<<<<`、`=======`、`>>>>>>>` 冲突标记；没有为了绕过冲突而删除本地 scheduler、sticky、privacy、image、runtime setting 或 passthrough 行为。

- [x] **Step 4: 合并后做格式和冲突残留检查**

Run:

```bash
git diff --check
git diff --name-only --diff-filter=U
rg "<<<<<<<|=======|>>>>>>>" -n .
```

Expected: `git diff --check` exit 0；无未解决冲突文件；`rg` 不命中冲突标记。

### Task 4: 复核版本、生成文件、配置和 migration

**Files:**
- Modify: `backend/cmd/server/VERSION`
- Modify: `backend/go.mod`
- Modify: `backend/go.sum`
- Modify: `backend/cmd/server/wire_gen.go`
- Modify: `backend/ent/**`
- Modify: `backend/migrations/**`
- Modify: `frontend/package.json`
- Modify: `frontend/pnpm-lock.yaml`
- Modify: `deploy/*.yml`
- Modify: `deploy/*.yaml`

**Interfaces:**
- Consumes: Task 3 的合并工作区
- Produces: 明确的版本、生成文件、配置和 migration 处理结论

- [x] **Step 1: 复核版本文件**

Run:

```bash
git diff -- backend/cmd/server/VERSION
```

Expected: `backend/cmd/server/VERSION` 对齐 upstream `v0.1.146`。如果本地四段式版本和 upstream 三段式版本冲突，按本次 merge 目标先保留 upstream `v0.1.146`，本地发版四段式版本留到后续 release change 决定。

- [x] **Step 2: 复核后端依赖和生成物**

Run:

```bash
git diff -- backend/go.mod backend/go.sum backend/cmd/server/wire_gen.go backend/ent
```

Expected: `go.mod`/`go.sum` 与合并后的源码一致；`wire_gen.go` 与 `backend/ent/**` 没有明显丢失本地字段或 provider 接线。若源码接线变化要求重生成，先记录需要运行的生成命令，再执行项目既有生成流程。

- [x] **Step 3: 复核 migration 编号和语义**

Run:

```bash
git diff -- backend/migrations
```

Expected: 没有同编号不同语义 migration 被静默覆盖；新增 migration 顺序可解释；如果 upstream 与本地定制对同一 schema 约束不可共存，暂停让用户确认。

- [x] **Step 4: 复核前端依赖和部署配置**

Run:

```bash
git diff -- frontend/package.json frontend/pnpm-lock.yaml deploy
```

Expected: 前端 lockfile 与 `package.json` 一致；部署配置没有覆盖本地环境约定；仅记录 upstream 引入但本次不使用的新配置。

### Task 5: 运行后端测试

**Files:**
- Test: `backend/**`
- Modify: 仅限修复本次合并引入或暴露问题所需的后端文件
- Modify: `docs/superpowers/plans/2026-07-07-merge-upstream-v0-1-146.md`

**Interfaces:**
- Consumes: Task 4 复核后的后端工作区
- Produces: `go test ./...` 的结果和必要修复

- [x] **Step 1: 运行后端全量测试**

Run:

```bash
go test ./...
```

Run from: `backend`

Expected: exit 0。若失败，先判断是否由本次 merge 引入；只修复 merge 引入或暴露的问题。

- [x] **Step 2: 对失败做最小修复并复跑**

Run:

```bash
go test ./...
```

Run from: `backend`

Expected: exit 0；若失败属于历史问题，记录失败包、测试名和错误摘要，不扩大修复范围。

### Task 6: 运行前端 typecheck、build 和单测

**Files:**
- Test: `frontend/src/**`
- Test: `frontend/package.json`
- Test: `frontend/pnpm-lock.yaml`
- Modify: 仅限修复本次合并引入或暴露问题所需的前端文件
- Modify: `docs/superpowers/plans/2026-07-07-merge-upstream-v0-1-146.md`

**Interfaces:**
- Consumes: Task 4 复核后的前端工作区
- Produces: 前端 typecheck/build/test 结果和必要修复

- [x] **Step 1: 运行 TypeScript 类型检查**

Run:

```bash
pnpm typecheck
```

Run from: `frontend`

Expected: exit 0。若失败，只修复 upstream 合并造成的类型或接口接线问题。

- [x] **Step 2: 运行前端生产构建**

Run:

```bash
pnpm build
```

Run from: `frontend`

Expected: exit 0；构建 warning 可记录，但不能隐藏错误。

- [x] **Step 3: 运行前端单测**

Run:

```bash
pnpm test:run
```

Run from: `frontend`

Expected: exit 0；warning 可记录，失败只修复 upstream 合并引入或暴露的问题。

### Task 7: 本地关键能力专项 review

**Files:**
- Reference: `backend/internal/service/openai_account_scheduler*.go`
- Reference: `backend/internal/service/openai_gateway*.go`
- Reference: `backend/internal/service/gateway_service*.go`
- Reference: `backend/internal/service/setting_service.go`
- Reference: `backend/internal/service/settings_view.go`
- Reference: `backend/internal/handler/admin/setting_handler.go`
- Reference: `backend/internal/repository/**usage**.go`
- Reference: `frontend/src/types/index.ts`
- Reference: `frontend/src/views/admin/**`
- Reference: `frontend/src/views/user/**`
- Modify: `docs/superpowers/plans/2026-07-07-merge-upstream-v0-1-146.md`

**Interfaces:**
- Consumes: 通过后端测试与前端 typecheck/build 的合并结果
- Produces: 六项本地关键能力的 review 结论

- [x] **Step 1: Review scheduler 和 sticky**

Run:

```bash
rg "layered|Layered|sticky|Sticky|previous_response|session" backend/internal/service backend/internal/handler
```

Expected: 分层调度、账号恢复、sticky/session/previous response 相关本地语义仍存在；upstream 新调度逻辑没有绕开本地调度入口。

- [x] **Step 2: Review privacy**

Run:

```bash
rg "privacy|Privacy|RequirePrivacy|privacy_mode" backend/internal frontend/src
```

Expected: OpenAI/Antigravity privacy 设置、强制重设、列表过滤和前端字段仍接通；上游账号字段变化没有让隐私状态丢失或误显示。

- [x] **Step 3: Review image capability**

Run:

```bash
rg "image_generation|AllowImageGeneration|ImageRate|ImagePrice|ImagesView|image_output" backend/internal frontend/src
```

Expected: AI Images 路由、渠道价格、图片计费、图片历史和 usage detail 中的 image metadata 仍保留；上游模型能力变化没有关闭本地图片入口。

- [x] **Step 4: Review runtime setting 热更新**

Run:

```bash
rg "runtime|Runtime|settings_view|SettingService|UpdateSettings|Reload" backend/internal/service backend/internal/handler/admin frontend/src/views/admin
```

Expected: 设置保存后仍能热更新运行时行为；Wire/constructor 接线没有丢依赖；前端设置项与后端 DTO 字段一致。

- [x] **Step 5: Review 网关透传字段**

Run:

```bash
rg "passthrough|PassThrough|pass-through|extra_body|metadata" backend/internal frontend/src
```

Expected: 账号透传字段配置、规则保存、网关请求透传和 usage 记录快照仍然成立；upstream 请求清洗逻辑没有吞掉本地允许的字段。

- [x] **Step 6: 记录 review 结论**

Action:

```text
scheduler: 通过 / 需修复 / upstream 原生问题
sticky: 通过 / 需修复 / upstream 原生问题
privacy: 通过 / 需修复 / upstream 原生问题
image capability: 通过 / 需修复 / upstream 原生问题
runtime setting 热更新: 通过 / 需修复 / upstream 原生问题
网关透传字段: 通过 / 需修复 / upstream 原生问题
```

Expected: 每项都有结论；`需修复` 项必须只包含本次 merge 相关问题；`upstream 原生问题` 只记录，不并入当前 change。

### Task 8: 收尾汇总并等待用户决策

**Files:**
- Modify: `docs/superpowers/plans/2026-07-07-merge-upstream-v0-1-146.md`

**Interfaces:**
- Consumes: Task 1-7 的执行记录
- Produces: 用户可据此决定是否合回 `main`、是否推送远端的汇总

- [x] **Step 1: 做最终状态检查**

Run:

```bash
git status --short --branch
git diff --check
git diff --stat
```

Expected: 当前仍在临时分支；无冲突残留；diff 只来自 upstream merge 和必要修复。

- [x] **Step 2: 汇总执行结果**

Action:

```text
目标 tag：upstream-v0.1.146 -> <commit>
隔离分支或 worktree：<用户确认的隔离执行位置>
冲突处理：<冲突文件和处理策略摘要>
版本/生成文件/配置/migration：<复核结论>
后端测试：go test ./... -> PASS / FAIL <摘要>
前端 typecheck：pnpm typecheck -> PASS / FAIL <摘要>
前端 build：pnpm build -> PASS / FAIL <摘要>
前端单测：pnpm test:run -> PASS / FAIL <摘要>
专项 review：scheduler/sticky/privacy/image/runtime setting/passthrough -> <逐项结论>
遗留问题：<只列无关旧问题或 upstream 原生问题>
```

Expected: 汇总足够让用户判断是否继续合回 `main` 或推送远端。

- [x] **Step 3: 按 Comet build 规则提交隔离分支结果**

Run:

```bash
git status --short
git add -A
git add -f docs/superpowers/specs/2026-07-07-merge-upstream-v0-1-146-design.md docs/superpowers/plans/2026-07-07-merge-upstream-v0-1-146.md
git commit -m "merge: upstream v0.1.146"
```

Expected: 提交只包含当前 change 的 OpenSpec/Comet 产物、Design Doc、计划、upstream merge 结果和必要修复；不包含无关旧问题修复。

- [x] **Step 4: 暂停等待用户确认下一步**

Action:

```text
不合回 main。
不推送远端。
询问用户：是否合回 main，是否推送远端，是否进入后续 release change。
```

Expected: 本 change 的 build 阶段停在可审查、已提交的隔离分支结果上。

## 执行记录

- 目标 tag：`upstream-v0.1.146` -> `d7a6a4513a58b082922dfb8bd80f36cbe6b8a4c4`。
- 隔离分支：`feature/20260707/merge-upstream-v0-1-146`。
- 冲突处理：已融合冲突文件，保留 upstream 更新和本地 scheduler、sticky、privacy、image capability、runtime setting、passthrough 定制；未遇到需要暂停确认的不可共存语义。
- 版本/生成/配置/migration：`backend/cmd/server/VERSION` 对齐 upstream tag 内的 `0.1.145`；无 `go.mod/go.sum`、Ent、migration 差异；`wire_gen.go` 为构造依赖顺序调整；部署配置新增 `SETUP_MIGRATION_TIMEOUT_SECONDS` 并更新 URL allowlist 默认说明。
- 后端测试：`go test ./internal/handler -run "TestOpenAIGatewayHandler_Responses(RequiresChatCompletionsCapability|UsesGrokRequestPlatform)"` PASS；`go test ./internal/service -run "TestLayered_StickyWeighted(SessionPrefersStickyWithinTopK|PreviousCanMovePrefersStickyWithinTopK)"` PASS；`go test ./internal/handler ./internal/service` PASS；`go test ./...` PASS。
- 前端验证：`pnpm typecheck` PASS；`pnpm build` PASS；`pnpm test:run` PASS（157 files / 1175 tests）。构建仅输出 Vite chunk 和 Browserslist 数据过期警告；单测保留既有 stderr warning/log 输出但无失败。
- 专项 review：scheduler/sticky/privacy/image capability/runtime setting 热更新/网关透传字段通过；代码审查发现的 HTTP Responses capability 过滤、Grok `/v1/responses` platform 透传测试缺口和 layered sticky-weighted TopK 偏好语义差异已修复并补测试。
- 遗留问题：未发现需要并入当前 change 的旧问题；Vite chunk/Browserslist 警告保持记录，不在本次 upstream merge 扩大处理。
- 收尾：用户已确认提交并合并到 `main`，临时分支已删除；发版后 `backend/cmd/server/VERSION` 保持 `0.1.146.1`。

## 自检

- Spec coverage: 已覆盖获取 upstream refs/tags、确认 `v0.1.146`、确认隔离分支、合并、冲突/版本/生成文件复核、后端测试、前端 typecheck/build、本地关键能力专项 review、收尾汇总。
- Placeholder scan: 未发现待填占位语句或未定义接口名。
- Scope check: 计划不包含直接合回 `main`、推送、发版、无关重构、新 capability 或 schema 设计。
