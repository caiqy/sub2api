---
change: merge-upstream-v0-1-151
design-doc: docs/superpowers/specs/2026-07-13-merge-upstream-v0-1-151-design.md
base-ref: 46d92f1d75f9835539f2a86d92849604a79d2f44
---

# 合并上游 v0.1.151 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在独立分支以一次非快进合并引入 upstream `v0.1.151`，保留本地关键能力，并留下可复核的验证记录。

**Architecture:** 以本文档 `base-ref` 为业务代码基线，先确认远端 tag 的固定对象，再在隔离分支执行一次 `--no-ff` merge。merge 第一父允许在 `base-ref` 之上包含本 change 的纯文档和 Comet 进度提交，但不得包含其他业务代码。冲突按设计文档第 3 节按语义处理；任何不可共存的业务语义必须停在用户决策门槛，不能由实施者自行选择。合并后的兼容修复与上游 merge 保持为不同提交。

**Tech Stack:** Git、Go 1.25、Gin、Ent、Wire、Vue 3、Vite、Tailwind、pnpm。

## Global Constraints

- 仅合并 `upstream` 的 annotated tag `v0.1.151`，其 peel commit 必须为 `deff3123ded1d14e51df1fd1286e3d43ed9ec9bd`；禁止合入 tag 后的 `upstream/main` 40 个提交。
- 基线必须是 `46d92f1d75f9835539f2a86d92849604a79d2f44`，且开始前 `main` 必须与 `origin/main` 相同。
- 允许创建必要的 merge、兼容修复和 Comet 进度提交；不执行发布、部署、推送、合回 `main` 或无关重构，这些操作由用户在验证完成后决定。
- 所有冲突先分类为上游演进、本地定制、接口/配置/数据模型演进、或版本/依赖/生成文件/migration 差异；可共存时最小融合。
- 发现回归时先添加稳定失败测试，再写最小兼容修复；覆盖主路径及适用的 previous response、fallback、重试和终止错误路径。
- 最终必须通过 `go test ./... -count=1`、`pnpm test:run`、`pnpm typecheck`、`pnpm build`、`git diff --check`、冲突标记扫描和 tag 祖先关系检查；新增失败不得忽略。

---

## 文件清单

- Create: `docs/superpowers/reports/2026-07-13-merge-upstream-v0-1-151-validation.md`，记录 tag、merge commit、冲突决策、能力审查、命令结果、警告和残余风险。
- Modify when merge changes them: `backend/cmd/server/VERSION`、`backend/go.mod`、`backend/go.sum`、`frontend/package.json`、前端锁文件、配置默认值、`backend/cmd/server/wire.go`、`backend/cmd/server/wire_gen.go`、`backend/ent/**`、`backend/ent/migrate/**`。
- Review and test as applicable: `backend/internal/service/openai_account_scheduler*.go`、`backend/internal/service/sticky_session_test.go`、`backend/internal/handler/gateway_handler_error_fallback_test.go`、`backend/internal/service/openai_gateway_service*.go`、`backend/internal/pkg/apicompat/*.go`、`backend/internal/service/*privacy*`、`backend/internal/service/openai_images*.go`、`backend/internal/service/setting_service.go`、`backend/internal/handler/admin/setting_handler.go`、`backend/internal/handler/gateway_request_body_spooling_test.go`、`backend/internal/handler/request_body_coordinator*.go`、`backend/internal/handler/openai_request_body_retention_test.go`、`backend/internal/service/request_body_handle*.go`。

### Task 1: 固定起点与上游目标

**Files:**
- Modify: 无。
- Create: `docs/superpowers/reports/2026-07-13-merge-upstream-v0-1-151-validation.md`（初始化记录）。

**Produces:** 经确认的 `base-ref`、tag peel commit 与工作树审计结果；后续任务只可使用该记录的对象。

- [x] **Step 1: 记录当前分支、基线和工作树状态**

Run:
```powershell
git branch --show-current
git rev-parse HEAD
git status --short --branch
git rev-parse origin/main
```

Expected: 当前分支为 `feature/20260713/merge-upstream-v0-1-151`，`HEAD` 与 `origin/main` 均为 `46d92f1d75f9835539f2a86d92849604a79d2f44`，且没有未授权的改动。

- [x] **Step 2: 处理工作树门槛**

Run:
```powershell
git status --porcelain
```

Expected: 仅允许本 change 的 OpenSpec、Comet、计划和报告文件；出现其他未跟踪或未提交内容时暂停并向用户列出路径。不得 stash、删除或覆盖用户文件。

- [x] **Step 3: 刷新并固定 release tag**

Run:
```powershell
git fetch upstream --tags
git rev-parse v0.1.151^{}
git show -s --format='%H%n%P%n%s' v0.1.151^{}
git merge-base --is-ancestor v0.1.151 upstream/main
```

Expected: peel commit 为 `deff3123ded1d14e51df1fd1286e3d43ed9ec9bd`，最后一条命令退出码为 `0`；否则停止，不创建分支，并报告实际对象。

- [x] **Step 4: 初始化验证记录**

Create `docs/superpowers/reports/2026-07-13-merge-upstream-v0-1-151-validation.md` with:
```markdown
# 上游 v0.1.151 合并验证记录

- Base ref: `46d92f1d75f9835539f2a86d92849604a79d2f44`
- Target tag: `v0.1.151`
- Target peel commit: `deff3123ded1d14e51df1fd1286e3d43ed9ec9bd`
- Merge branch: pending
- Merge commit: pending
```

Expected: 记录可追溯本次合并的固定输入，尚不声称验证通过。

### Task 2: 创建隔离分支并执行唯一上游合并

**Files:**
- Modify: 合并实际触及的文件，尚未确定。
- Create: 无。

**Consumes:** Task 1 确认的 `base-ref` 和 tag peel commit。
**Produces:** 一个包含一次 `--no-ff` 上游 merge 的隔离分支，或一个明确的冲突清单。

- [x] **Step 1: 创建带基线的隔离分支**

Run:
```powershell
git branch --show-current
git rev-parse HEAD
git status --short --branch
```

Expected: 分支为 `feature/20260713/merge-upstream-v0-1-151`，`base-ref` 是 `HEAD` 的祖先，`base-ref..HEAD` 仅包含本 change 的纯文档和 Comet 进度提交，工作树仅包含本 change 的 OpenSpec、Comet、计划和报告文件。

- [x] **Step 2: 发起一次非快进 merge**

Run:
```powershell
git merge --no-ff v0.1.151
git status --short
git diff --name-only --diff-filter=U
```

Expected: 要么产生一个 merge commit，要么最后一条命令列出全部未解决冲突文件。不得改用 cherry-pick、逐 tag 合并、`upstream/main`、`-s ours` 或 `-X ours/theirs`。

- [x] **Step 3: 记录上游变更范围**

Run:
```powershell
git diff --stat 46d92f1d75f9835539f2a86d92849604a79d2f44..v0.1.151
git log --oneline --decorate 46d92f1d75f9835539f2a86d92849604a79d2f44..v0.1.151
```

Expected: 验证记录列出变更范围、冲突文件及初始归类，供 Task 3 审核。

### Task 3: 逐项处理冲突并设置不可共存语义门槛

**Files:**
- Modify: `git diff --name-only --diff-filter=U` 返回的每个文件。
- Test: 与每个冲突文件所属能力对应的既有 `*_test.go` 或前端测试。

**Consumes:** Task 2 的未解决冲突清单。
**Produces:** 无冲突标记的 merge 结果，或等待用户选择的阻塞说明。

- [x] **Step 1: 为每个冲突建立决策表**

For each path, run:
```powershell
git diff -- "<path>"
git log -1 --format='%H %s' HEAD -- "<path>"
git log -1 --format='%H %s' v0.1.151 -- "<path>"
```

Expected: 验证记录对每个文件包含类别、上游行为、本地行为、融合结果和对应测试命令。

- [x] **Step 2: 处理语义可共存冲突**

Resolve by retaining both independently required behaviors and only the smallest shared glue. Then run:
```powershell
git add -- "<path>"
git diff --cached --check
```

Expected: 路径已暂存且无空白错误；不通过全文采用 `ours` 或 `theirs` 丢弃任一侧功能。

- [x] **Step 3: 在不可共存语义处停止并请求用户决定**

Pause before staging the path when either choice changes externally observable gateway, scheduling, privacy, billing, request-body ownership, schema/migration, or configuration-default behavior and both cannot coexist. 向用户提供：
```markdown
阻塞文件：填入冲突文件路径。
方案 A：描述保留的行为及调用链影响。
方案 B：描述另一种保留的行为及调用链影响。
数据/兼容性风险：列出具体风险。
推荐：给出推荐方案及理由。
需要的决定：选择 A 或 B，或明确新的期望行为。
```

Expected: 未收到明确选择前，不编辑该冲突为最终解、不执行 `git merge --continue`，也不进入后续任务。

- [x] **Step 4: 完成 merge 状态检查**

Run:
```powershell
git diff --name-only --diff-filter=U
git grep -n -E '^(<<<<<<<|=======|>>>>>>>)' -- . ':!docs/superpowers/reports/**'
git status --short
```

Expected: 前两条没有输出；merge 已完成且工作树仅包含本次预期改动。若仍处于 merge 状态，先完成所有冲突决策后才可继续。

### Task 4: 复核版本、依赖、配置、生成代码和 migration

**Files:**
- Modify as required: `backend/cmd/server/VERSION`、`backend/go.mod`、`backend/go.sum`、`frontend/package.json`、前端锁文件、配置默认值、`backend/cmd/server/wire.go`、`backend/cmd/server/wire_gen.go`、`backend/ent/**`、`backend/ent/migrate/**`。
- Test: 受影响 Go package 的 `go test`；受影响前端 workspace 的 `pnpm` 检查。

**Consumes:** 已无冲突标记的 merge 结果。
**Produces:** 运行时和发布元数据与 merge 结果一致，所有额外修复与 merge commit 分离。

- [x] **Step 1: 枚举高风险元数据差异**

Run:
```powershell
git diff --name-status 46d92f1d75f9835539f2a86d92849604a79d2f44..HEAD -- backend/cmd/server/VERSION backend/go.mod backend/go.sum frontend/package.json frontend/pnpm-lock.yaml backend/cmd/server/wire.go backend/cmd/server/wire_gen.go backend/ent backend/ent/migrate
git diff --check
```

Expected: 每个变更文件在验证记录中有“保留上游”、“保留本地”或“融合”结论；`git diff --check` 无输出。

- [x] **Step 2: 按项目既有生成方式校验生成产物**

Run:
```powershell
rg -n "wire gen|ent generate|go generate" Makefile backend frontend .github --glob '!**/node_modules/**'
```

Expected: 使用仓库已定义的命令重新生成受影响的 Wire/Ent 产物；生成后仅出现与源定义一致的变更。若生成命令会改动 schema 或 migration，先记录 diff 并在不可逆变更处请求用户确认。

- [x] **Step 3: 验证依赖锁与配置默认值**

Run:
```powershell
git diff 46d92f1d75f9835539f2a86d92849604a79d2f44..HEAD -- backend/go.mod backend/go.sum frontend/package.json frontend/pnpm-lock.yaml
git diff 46d92f1d75f9835539f2a86d92849604a79d2f44..HEAD -- backend/internal/config backend/internal/handler/dto backend/internal/service/settings_view.go
```

Expected: 每项版本、锁文件和默认值变更都有来源与兼容性结论；不为清理目的升级无关依赖。

### Task 5: 审查调度、网关、安全和运行时关键能力

**Files:**
- Review/Test: `backend/internal/service/openai_account_scheduler*.go`、`backend/internal/service/sticky_session_test.go`、`backend/internal/handler/gateway_handler_error_fallback_test.go`、`backend/internal/service/openai_gateway_service*.go`、`backend/internal/pkg/apicompat/*.go`、`backend/internal/service/*privacy*`、`backend/internal/service/openai_images*.go`、`backend/internal/service/setting_service.go`、`backend/internal/handler/admin/setting_handler.go`。

**Consumes:** Task 4 的一致元数据和生成代码。
**Produces:** 能力矩阵逐项具备“保持/回归/修复后保持”结论与定向验证证据。

- [x] **Step 1: 审查 scheduler、sticky、fallback 与 DB recheck**

Run:
```powershell
go test ./internal/service -run 'Test(OpenAIAccountScheduler|Sticky|Fallback|WaitPlan|PreviousResponse)' -count=1
go test ./internal/handler -run 'Test.*Fallback|Test.*Sticky' -count=1
```

Working directory: `backend/`.

Expected: 覆盖常规调度、previous response 粘性、fallback、等待计划与重试/终止边界；任何失败先归因到 merge 差异。

- [x] **Step 2: 审查协议转换与流式终止 usage**

Run:
```powershell
go test ./internal/pkg/apicompat -count=1
go test ./internal/service -run 'Test(OpenAIGateway|Gateway.*Responses|Gateway.*Chat|Gateway.*Messages|Terminal.*Usage)' -count=1
```

Working directory: `backend/`.

Expected: Messages、Responses、Chat、stream/non-stream、透传字段和终止 usage 均无回归。

- [x] **Step 3: 审查 privacy、图像能力和运行时设置更新**

Run:
```powershell
go test ./internal/service -run 'Test.*Privacy|Test.*Image|Test.*Setting|Test.*Reload|Test.*Cache' -count=1
go test ./internal/handler/admin -run 'Test.*Setting' -count=1
```

Working directory: `backend/`.

Expected: privacy 和内容审计、image capability/模型过滤、setting 热更新、缓存失效及服务重建/重载语义均有测试证据。

### Task 6: 审查大输入请求体生命周期

**Files:**
- Review/Test: `backend/internal/handler/gateway_request_body_spooling_test.go`、`backend/internal/handler/request_body_coordinator.go`、`backend/internal/handler/request_body_coordinator_test.go`、`backend/internal/handler/openai_request_body_retention_test.go`、`backend/internal/service/request_body_handle.go`、`backend/internal/service/request_body_handle_test.go`。

**Consumes:** Task 4 的最终请求处理与依赖状态。
**Produces:** 内存到磁盘切换、重放、所有权转移和成功/失败清理的专项结论。

- [x] **Step 1: 运行现有请求体生命周期测试**

Run:
```powershell
go test ./internal/handler -run 'Test.*(RequestBody|BodySpool|BodyRetention|Replay|Cleanup)' -count=1
go test ./internal/service -run 'Test.*(RequestBody|CachedBody|BodyOrder)' -count=1
```

Working directory: `backend/`.

Expected: 大输入落盘、重放、fallback/重试、取消或终止错误、panic/失败清理与内存释放均通过。

- [x] **Step 2: 对发现的回归执行测试优先修复**

Create a focused regression test next to the affected implementation before changing production code. The test must assert the exact retained behavior, for example:
```go
func TestGatewayRequestBodySpoolingCleansTemporaryFileAfterFallbackFailure(t *testing.T) {
	// Arrange an oversized body and a failing fallback attempt.
	// Assert replay receives identical bytes and the temporary file is removed.
}
```

Run:
```powershell
go test ./internal/handler -run '^TestGatewayRequestBodySpoolingCleansTemporaryFileAfterFallbackFailure$' -count=1
```

Expected: 新测试先失败且能稳定复现；实施最小修复后通过，随后重跑 Step 1。若实际回归属于其他包，测试名称、路径和命令必须对应真实所有者。

### Task 7: 完整验证与 Git 完整性检查

**Files:**
- Modify: 无（除验证报告）。
- Test: `backend/` 全部 Go 包与 `frontend/` 全部规定检查。

**Consumes:** 已完成的 merge 和任何独立兼容修复。
**Produces:** 可重复执行的全量验证结果。

- [x] **Step 1: 执行后端全量测试**

Run:
```powershell
go test ./... -count=1
```

Working directory: `backend/`.

Expected: 退出码为 `0`；既有非阻塞警告须记录，新增失败必须修复或阻塞并升级给用户。

- [x] **Step 2: 执行前端全量验证**

Run:
```powershell
pnpm test:run
pnpm typecheck
pnpm build
```

Working directory: `frontend/`.

Expected: 三条命令均退出码为 `0`，构建产物不纳入提交范围，除非仓库既有规则要求。

- [x] **Step 3: 验证 merge 祖先关系、差异和冲突残留**

Run:
```powershell
git merge-base --is-ancestor v0.1.151 HEAD
git diff --check 46d92f1d75f9835539f2a86d92849604a79d2f44..HEAD
git grep -n -E '^(<<<<<<<|=======|>>>>>>>)' -- . ':!docs/superpowers/reports/**'
git status --short --branch
```

Expected: 前三项退出码为 `0` 且无冲突/空白输出；最后一项仅包含本 change 已知文件。任何意外文件先停止并向用户报告，不删除用户改动。

### Task 8: 完成验证记录并交由用户处置分支

**Files:**
- Modify: `docs/superpowers/reports/2026-07-13-merge-upstream-v0-1-151-validation.md`。

**Consumes:** Task 1-7 的命令输出、冲突决策和能力审查结论。
**Produces:** 合并结果的审计记录，以及明确等待用户的后续操作。

- [x] **Step 1: 填写验证报告的结果表**

Append `## Merge` with the actual branch, merge commit, tag-ancestor result, each conflict decision, and every post-merge compatibility-fix path. Append `## Capability Review` with one row each for scheduler/sticky/fallback, gateway conversion/terminal usage, privacy/image capability, runtime setting/cache reload, and large request-body lifecycle; each row must state the actual result and command evidence. Append `## Full Verification` with the actual results and warnings for all four mandatory commands and Git integrity checks. Finish with `## Residual Risks`, listing concrete risks or `None identified`.

Expected: 所有 `pass` 只在实际命令退出码为 `0` 后填写；失败或未执行项目必须如实保留为失败或未执行。

- [ ] **Step 2: 停止在用户处置门槛**

Report branch name、merge commit、验证报告路径、失败/警告和残余风险。请求用户选择以下其中一项：保留隔离分支、合回 `main`、或推送；不得自行执行其中任何操作，也不得发布或部署。
