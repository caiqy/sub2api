# Task 27 Execution Report: final full verify

## Source and scope

- Branch: `feature/20260726/staged-merge-upstream-v0-1-165`.
- Task start HEAD: `8c3b281f7f9e08a9f2d776f4241a922f7a85bff8`.
- Final source/test HEAD: `417bbcc6a44c35b3e3ed16efb0bb86a4717401c9`.
- Formal verify report commits: `35dddd7ae` (`docs: record final automated verification`); `0884b595c` (`docs: record final make test pass`); `ac3d1b833` (`docs: record unified image audit verification`); `763db6ad4` (`docs: record blocking audit race verification`).
- VERSION: `0.1.165.1`.
- Source/test commits: `aff04f9cd style: format websocket ownership changes`; `1cc41c72c fix: avoid unused image moderation payload`; `6c88f1891 fix: avoid unused image audit payload`; `90b008901 fix: unify image security audit payload`; `417bbcc6a fix: isolate blocking audit requests`.
- Preserved throughout and prohibited from subagent staging: `.superpowers/sdd/task-27-report.md`、`.superpowers/sdd/task-4-report.md`、`openspec/changes/staged-merge-upstream-v0-1-165/.comet/subagent-progress.md`、`opencode.json`、`.comet/current-change.json`、`paseo.json`。主协调器可按 Comet 协议持久化 checkpoint；progress 仅在 `aa5b576fe` 中由主协调器与 plan 一起合法提交，不是 implementer 夹带。
- Task 28 ancestry and Task 29 browser/OpenSpec closure were not run.

## Focused evidence

- All non-integration matrix commands recorded in the formal report executed real named tests and exited `0` after excluding one zero-test Grok command; the corrected `-tags=unit` Grok 402 command matched and passed.
- Task 25 handler matrix covered Live parsing/gates, HTTP body replay/spooling/cleanup, failed usage, ordinary/Grok final models, account failover reset, OAuth normalization and passthrough turn failures.
- Task 25 service matrix covered Live selection/finalization, prompt-cache injection, passthrough first-write/drain/mapping, HTTP bridge publication boundaries and model normalization.
- `openai_ws_v2` matrix covered terminal downstream ownership, follow-up ordering, turn permits, metrics, mismatched terminal and drain terminal handling.
- Scheduler matrix covered advanced/layered, both sticky paths, fallback/WaitPlan, DB fresh recheck and composite routing.
- Grok/Ollama/Alipay/email alias, migration package/runner, RequestType/Live lease and admin group guards all passed.
- Frontend Groups/UsageFilters focused run: 3 files / 17 tests PASS; vue-tsc exit `0`; PostCSS resolved to `8.5.23`.
- Exact commands and test names are self-contained in `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-verify.md`.

## `make test` and repairs

- First `make test` at `8c3b281f7`: exit `2`; default Go tests passed, then lint found exactly two committed gofmt violations. The nonzero result is preserved and is not reported as PASS.
- `aff04f9cd` applies only gofmt output. Affected WebSocket tests passed and full lint returned `0 issues`.
- A diagnostic post-format `make test` reached unit tests and exited `2` only on `TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream`: HeapAlloc `85,544,152` exceeded `77,155,608`.
- Root cause: `parsed.ModerationBody()` allocated a prompt-sized JSON payload even when moderation was disabled. `1cc41c72c` guards that allocation and makes the fixture retry-safe.
- GREEN: exact OAuth test, `-count=10`, moderation-enabled sibling, full unit handler package and handler lint all pass.
- Final source HEAD expanded full gate: backend default `./...`, `golangci-lint ./...`, backend unit `./...`, frontend lint, typecheck and full Vitest all exit `0`; Vitest is 215 files / 1626 tests.

## Build, generate and static

- `make "VERSION=0.1.165.1" "SHELL=D:/scoop/shims/bash.exe" build`: exit `0`; VERSION file and binary both contain `0.1.165.1`; binary metadata points to `1cc41c72c`.
- Detached `D:\w27` at the same source HEAD: two `make -C backend generate` rounds exit `0`; both Ent/Wire diff checks exit `0`; worktree removed and `Test-Path=False`.
- `git diff --check` exit `0`; unmerged index/diff counts `0`; exact marker count `0`; legacy first-token count `0`; VERSION assertion passes.
- Fourteen protected migration filenames exist, including local/upstream 172 and 181, both 186 files and `190_*_notx.sql`.

## Remote integration

- ssh-skill only; alias `local-serv-ai`; stage `final-verify`; nonce `cbfc15a98f5e418e92f2944efeecb676`.
- Remote path: `/tmp/sub2api-final-verify-cbfc15a98f5e418e92f2944efeecb676`.
- Go `1.26.5`; Docker Server `29.2.1`; preflight success.
- `CI=true GOFLAGS='-v'` with rebuilt `.test-tmp` and `TMPDIR`/`TMP`/`TEMP` ran `go test -tags=integration ./...`: execution JSON `success=true, exit_code=0`; no FAIL.
- Migration new-schema target PASS `5.50s`; local-v0.1.159.6 upgrade target PASS `4.29s`. Committed fixed assertion plus target PASS proves 12/12, dual 186, 190 notx, local same-number files, reapply count and checksum.
- 13 skips classified: DingTalk sentinel, TLS capture, known concurrency TODO, prompt-audit external Redis/PostgreSQL fixtures and external OpenAI token comparison. None hits Task 27 capability or migration targets.
- Log: `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-cbfc15a98f5e418e92f2944efeecb676-integration.log`.
- Download and cleanup JSON both `success=true, exit_code=0`; remote directory absent; local tar `Test-Path=False`; log retained.

## Concerns and status

- Residuals: Grok attempted is marked before request build; drain lifecycle test uses fixed `50ms`; OAuth HeapAlloc history is retained although the fresh root cause was fixed and final full gate is green; Windows lacks CGO/gcc for `-race`.
- Process concern: the first `make test` nonzero is preserved; one diagnostic rerun exposed the OAuth RED, and final verification used the command's expanded constituents rather than relabeling the first run.
- No push, tag, release, deploy, Sub2API image build, Task 28 or Task 29 action occurred.
- Status: `DONE_WITH_CONCERNS`.

## 2026-07-30 原样 `make test` 最终证据补跑

### 起始状态与约束

- 已先加载 `test-driven-development`；本轮仅补验证证据，未伪造新 RED，未修改生产或测试代码。
- `comet state get staged-merge-upstream-v0-1-165 language`：exit `0`，输出 `zh-CN`。
- `git rev-parse HEAD`：exit `0`，输出 `35dddd7aeafc9d1db1267241757cb0c93c35ff67`；source/test 基础仍为其父提交 `1cc41c72c1def83113263d9b631f9856dbff030d`。
- 起始 `git status --short`：exit `0`，为 `M .superpowers/sdd/task-27-report.md`、`M .superpowers/sdd/task-4-report.md`、`M openspec/changes/staged-merge-upstream-v0-1-165/.comet/subagent-progress.md`、`?? .comet/current-change.json`、`?? paseo.json`。这些既有 dirty 均保留且未暂存；本轮只继续追加 Task 27 report。

### 唯一一次原样执行

- 在仓库根目录原样执行 `make test` 一次，未设置额外参数，未自动重跑；exit `2`，不得记为 PASS。
- backend default tests 全部通过，其中 handler `65.789s`、service `105.348s`；随后 `golangci-lint run ./...` exit `0`，输出 `0 issues.`。
- backend unit-tag 阶段仅 `github.com/Wei-Shaw/sub2api/internal/handler` 失败（`103.333s`）；日志中其余 package 无其他 `--- FAIL:`，service 通过（`158.427s`）。`make[1]` 以 `test-unit Error 1` 退出，顶层 `make` 以 `test-backend-unit Error 2` 退出。
- 唯一失败测试为 `TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream`（`2.75s`），断言位置 `backend/internal/handler/openai_images_controls_test.go:660`：actual HeapAlloc `85,602,768`，ceiling `77,228,584`，消息 `blocked OAuth request retained 20MB text`。
- 因依赖的 backend unit target 失败，frontend lint、typecheck 和 Vitest 未启动。本轮完整输出保存在 `C:/Users/caiqy/.local/share/opencode/tool-output/tool_faed161c20010vxcv0o8WbZLdV`。

### 只读根因调查

- 已按失败分支加载 `systematic-debugging`，没有再次运行任何测试，也没有修改生产/测试代码。
- 由阈值反推 GC 前 baseline 为 `64,645,672` bytes；actual 相对 baseline 增加 `20,957,096` bytes，几乎等于 fixture 的 20 MiB prompt（`20,971,520` bytes），并超过 12 MiB 容许增量 `8,374,184` bytes。失败信号仍是一个完整 prompt 量级的存活堆对象，不是小幅阈值噪声。
- `git diff --name-only 1cc41c72c..HEAD` 和 `git diff --stat 1cc41c72c..HEAD` 均 exit `0`，只列 formal verify report；`1cc41c72c` 后没有 source/test 变更可解释新回归。
- `git show 1cc41c72c -- backend/internal/handler/openai_images.go backend/internal/handler/openai_images_controls_test.go` 显示该提交只在 `contentModerationService != nil` 时构造第一份 `parsed.ModerationBody()`，并用 `startedOnce` 保护 fixture 重试。当前 `Images` 仍在 security-audit 调用参数中无条件求值 `parsed.ModerationBody()`；本测试环境的 security-audit coordinator 与 content moderation service 均为 nil。此外 OAuth 转换会先构造完整 Responses body，再由阈值为 1 byte 的 `SetOAuthBytes` 写入 spool。这两处均是只读调查确认的完整 prompt 量级分配路径，但在禁止重跑/profile 的约束下，不能把具体存活对象进一步归因到其中一处。
- 近期相关历史并非首次波动：`e98ce7a78` 将请求 fixture 改为流式生成，`196ee1488` 增加完整 prompt 下界及失败 cleanup；Task 17/21/25 报告记录同一测试多次在 full package 中超限、focused 随后通过。`3cfdaffa1` 曾尝试 `body = nil`，lint 证明为 ineffectual assignment，`d8022d582` 已回退。当前失败数值又与修复前 Task 27 诊断的 `85,544,152 > 77,155,608` 几乎一致，因此 `1cc41c72c` 不能提供稳定的原样 full-gate 绿色证据；process-wide `HeapAlloc` 断言本身仍有已知不稳定风险。
- 只读 git 调查命令 `git log --oneline --decorate -12`、相关文件 `git log --oneline --all`、`git show`（`1cc41c72c`、`e98ce7a78`、`196ee1488`、`3cfdaffa1`、`d8022d582`、`35dddd7ae`）、`git diff`/`git diff --stat`/`git diff --name-only` 均 exit `0`。`git diff --check` exit `0`，只输出既有 Task 27/Task 4/progress 的 LF/CRLF 提示；`git diff --cached --name-only` exit `0` 且无输出。

### 结果、变更与风险

- 状态：`BLOCKED`。缺少修复后原样 `make test` 的零退出证据，且本轮首次、唯一结果为 exit `2`。
- 本轮变更文件只有 `.superpowers/sdd/task-27-report.md`（未提交工作报告）；formal verify report、OpenSpec tasks/plan/progress、业务源码和测试均未修改。
- 提交：无；未暂存任何文件，未 amend。
- 顾虑：formal verify report 仍保留此前两次非零和展开式绿色证据，但本轮不能追加 PASS；既有“fresh root cause 已修复”结论受到同型复现反证，需要后续单独处理测试稳定性或残余分配后再取得一次原样 full-gate 证据。
- 风险信号：同一 20 MiB HeapAlloc failure 跨任务反复出现；完整 prompt 量级分配路径仍存在；该断言读取 process-wide HeapAlloc；frontend 本轮未执行。未运行远程操作、Paseo、push/tag/release/deploy 或 Sub2API 镜像构建。

## 2026-07-30 security-audit payload 修复与最终门禁

### 入口、RED 与根因

- 按要求先加载 `systematic-debugging` 与 `test-driven-development`；`comet state get staged-merge-upstream-v0-1-165 language` exit `0`，输出 `zh-CN`。
- 起始 HEAD 为 `35dddd7aeafc9d1db1267241757cb0c93c35ff67`，既有 source/test 为 `1cc41c72c1def83113263d9b631f9856dbff030d`。起始 dirty 为 `M .superpowers/sdd/task-27-report.md`、`M .superpowers/sdd/task-4-report.md`、`M openspec/changes/staged-merge-upstream-v0-1-165/.comet/subagent-progress.md`、`?? .comet/current-change.json`、`?? paseo.json`；除本报告外均未修改或暂存。
- 本轮 RED 沿用本报告第 59-91 行记录的唯一一次原样 `make test` 真实失败，不另造 RED、不重跑碰绿：exit `2`，唯一失败 `TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream`，HeapAlloc `85,602,768 > 77,228,584`，相对 baseline 存活增量 `20,957,096` bytes，约为一个 20 MiB prompt。
- 数据流确认：`1cc41c72c` 只把 content moderation 的 `parsed.ModerationBody()` 放到 `contentModerationService != nil` 守卫内；`OpenAIGatewayHandler.Images` 的 security-audit 实参仍直接调用 `parsed.ModerationBody()`。Go 在进入 `checkSecurityAudit` 前求值该参数，而 `runSecurityAudit` 到 coordinator 判空后才返回 nil，因此 moderation/audit 双 nil 路径仍构造并保留完整 prompt payload。
- 最小修复只改 `backend/internal/handler/openai_images.go` 两处：仅当 `contentModerationService != nil || securityAuditCoordinator != nil` 时构造一次 frozen `moderationBody`，content moderation 与 security audit 复用该 slice。未改测试、HeapAlloc ceiling、配置或抽象；启用任一检查时仍构造完整 payload。

### GREEN 与 source commit

- `go test -tags=unit ./internal/handler -run '^TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream$' -count=1 -v`：exit `0`，目标明确 PASS（`2.41s`）。
- `go test -tags=unit ./internal/handler -run '^TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream$' -count=10 -v`：exit `0`，10/10 PASS。
- `go test -tags=unit ./internal/handler -run '^TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease$' -count=1 -v`：exit `0`，目标 PASS，确认 moderation 启用时完整 prompt 仍可检查并阻断。
- `go test -tags=unit ./internal/handler -run '^(TestAsyncImagePromptGuardRunsBeforeTaskCreation|TestAsyncImageSuccessfulPrecheckIsNotRepeatedByDetachedExecution|TestBatchImagePromptGuardRunsBeforePersistenceOrBilling|TestSecurityAuditBlockingFailuresLeaveAllDownstreamCountersAtZero)$' -count=1 -v`：exit `0`，4 个 top-level 及 3 个 decision 子例均 PASS；实际匹配 security-audit media tests。
- `go test -tags=unit ./internal/handler -count=1`：exit `0`，package PASS（`94.890s`）。`golangci-lint run ./internal/handler/...`：exit `0`，`0 issues.`。
- `gofmt -d internal/handler/openai_images.go internal/handler/openai_images_controls_test.go` 与局部 `git diff --check` 均 exit `0`、无输出。提交前暂存区只有 `backend/internal/handler/openai_images.go`；提交 `6c88f1891650e0ef18b0b5ae105b8f44a069a5a4 fix: avoid unused image audit payload`，未 amend。

### 新 source commit 本地最终门禁

- 在 source commit `6c88f1891650e0ef18b0b5ae105b8f44a069a5a4` 上原样执行唯一一次 `make test`：exit `0`，未自动重跑。backend default handler `61.833s`、service `105.188s`；`golangci-lint run ./...` 为 `0 issues.`；unit handler `98.673s`、service `156.695s`；frontend lint/typecheck 成功，Vitest `215 passed` files / `1626 passed` tests。完整输出：`C:/Users/caiqy/.local/share/opencode/tool-output/tool_faee7c633001EnQID0RLCtGhrW`。
- `make "VERSION=0.1.165.1" "SHELL=D:/scoop/shims/bash.exe" build`：exit `0`；`backend/cmd/server/VERSION` 与二进制字符串均为 `0.1.165.1`，`go version -m backend/bin/server` 为 Go `1.26.5`、`vcs.revision=6c88f1891650e0ef18b0b5ae105b8f44a069a5a4`。Vite `1019` modules，既有 Browserslist/dynamic import/chunk-size advisory 非失败。
- detached worktree `D:\w27-final` 精确指向 source commit；两轮 `make -C backend generate` 均 exit `0`，每轮后的 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 均 exit `0`、无输出；worktree status 为空。`git worktree remove "D:/w27-final"` exit `0`，最终 `Test-Path=False`。
- 静态检查：`git diff --check` exit `0`（仅既有 dirty report/progress 的 LF/CRLF warning）；`git diff --name-only --diff-filter=U` 与 `git ls-files -u` 均 exit `0`、无输出；精确 tracked conflict marker count `0`；backend/frontend legacy first-token count `0`；VERSION assertion 为 `0.1.165.1`；`git diff -- opencode.json` 无输出；14 个 protected migration 全部存在、missing count `0`。

### Remote 前置失败、对象与清理

- 已先加载 `ssh-skill`，未使用 raw SSH/SCP。stage `final-verify`；nonce `eecce42b226d4e03a8cb2d875070f12e`；计划唯一 remote 目录 `/tmp/sub2api-final-verify-eecce42b226d4e03a8cb2d875070f12e`。
- `git archive --format=tar --output=C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-eecce42b226d4e03a8cb2d875070f12e.tar 6c88f1891650e0ef18b0b5ae105b8f44a069a5a4`：exit `0`；archive size `53,739,520` bytes，SHA-256 `BE2DCE002DD26E3BA1FAAA79E83052A3AA8B2A70DDCD76BD76C8F56142E960D3`。
- 首次且唯一 remote create/preflight 调用为 `python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai 'set -eu; test ! -e /tmp/sub2api-final-verify-eecce42b226d4e03a8cb2d875070f12e; mkdir -m 700 /tmp/sub2api-final-verify-eecce42b226d4e03a8cb2d875070f12e; go version; docker version --format "{{.Server.Version}}"; docker info >/dev/null; printf "preflight=ok\\n"' --timeout 120`。返回 JSON `success=false, exit_code=1`、stdout 为空；stderr 为 Windows native fallback 调用 OpenSSH 时 PowerShell 将 `BatchMode=yes` 误作 `outputFormat` 参数，`method=native_ssh_windows`，fallback reason 为检测到密钥需要 passphrase。该 required gate 非零后按约束停止，未重跑 preflight、未上传 archive、未解包、未运行 integration、未下载日志。
- 必要清理调用仍仅经 `ssh_execute.py`：`rm -rf /tmp/sub2api-final-verify-eecce42b226d4e03a8cb2d875070f12e; test ! -e ...` 返回 `success=true, exit_code=0`；remote 目录最终不存在。本地 archive 已删除，`archive_exists=False`；integration 未执行且本地 log 不存在，`log_exists=False`。

### 状态与风险

- 状态：`BLOCKED`。本地 source 修复、原样 `make test`、build、generate 与 static 均已通过，但 required remote preflight 首次结果非零，缺少新 source commit 上的 Linux full integration、两个 migration target、12/12 与 skip 分类证据；不得更新 formal report 为最终 PASS。
- 本轮提交仅 `6c88f1891650e0ef18b0b5ae105b8f44a069a5a4`；未修改 formal verify report，因此未创建 `docs: record final make test pass` 提交。Task 27 report 保持未提交，暂存区为空。
- 顾虑：OAuth HeapAlloc 使用 process-wide 指标，虽本次根因修复后精确 10/10、完整 unit package 及唯一原样 full gate 均绿色，历史敏感性仍应保留。
- 风险信号：`ssh-skill` 的 Windows native fallback 在 create/preflight 组合命令上发生参数解析失败，而后续同 method cleanup 成功，说明远程执行入口存在命令形状相关不一致；本轮不能据此声称 local-serv-ai integration PASS。未执行 Paseo、push、tag、release、deploy、merge main 或 Sub2API 镜像构建。

## 2026-07-30 新 nonce 受控 remote 验证

### 入口、对象与 archive

- 已加载 `ssh-skill` 与 `systematic-debugging`；`comet state get staged-merge-upstream-v0-1-165 language` exit `0`，输出 `zh-CN`。起始 `git rev-parse HEAD` 为 `6c88f1891650e0ef18b0b5ae105b8f44a069a5a4`；起始 dirty 为 `M .superpowers/sdd/task-27-report.md`、`M .superpowers/sdd/task-4-report.md`、`M openspec/changes/staged-merge-upstream-v0-1-165/.comet/subagent-progress.md`、`?? .comet/current-change.json`、`?? paseo.json`，均保留，除本报告外未修改或暂存。
- 未重跑 focused、原样 `make test`、显式版本 build、detached 双 generate 或 static；继续采用第 93-118 行在 source/test commit `6c88f1891650e0ef18b0b5ae105b8f44a069a5a4` 上的既有绿色证据，并完整保留第 99 行以及 formal verify report 中此前三次 `make test` 非零历史。
- stage `final-verify`；新 nonce `a9bada9b52f244e8bfc39ba41e9f092d`；唯一 remote 目录 `/tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d`；local archive `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d.tar`；local log `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d-integration.log`。
- `git archive --format=tar --output="C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d.tar" 6c88f1891650e0ef18b0b5ae105b8f44a069a5a4`：exit `0`；size `53,739,520` bytes；local SHA-256 `BE2DCE002DD26E3BA1FAAA79E83052A3AA8B2A70DDCD76BD76C8F56142E960D3`。

### 唯一受控 preflight、上传与执行

- 单一假设：上一 nonce 的本机解析失败由 Docker `--format "{{...}}"` brace/template 与命令引用形状触发；本轮只移除该 template 并按 `ssh-skill` 标准双引号形状执行一次最小 create/preflight，不修改脚本、不尝试第三种形状。
- 唯一 preflight 精确命令：`python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; test ! -e /tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d; mkdir -m 700 /tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d; go version; docker version; docker info >/dev/null; echo preflight=ok" --timeout 120`。JSON 为 `success=true, exit_code=0, stderr="", method=native_ssh_windows`；Go `1.26.5 linux/amd64`，Docker client/server `29.2.1`，`docker info` 成功。fallback reason 仍为检测到密钥需要 passphrase，但本次未发生参数解析非零；第 119-124 行 nonce `eecce42b226d4e03a8cb2d875070f12e` 的首次 preflight 本机解析失败历史保持不变。
- 上传精确命令：`$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_upload.py local-serv-ai "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d.tar" "/tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d/source.tar" --no-progress`。JSON `success=true, exit_code=0, stderr=""`。
- 解包/setup 精确命令：`python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d; sha256sum source.tar; mkdir src; tar -xf source.tar -C src; rm -f source.tar; rm -rf src/backend/.test-tmp; mkdir -p src/backend/.test-tmp; test -d src/backend/.test-tmp; echo setup=ok" --timeout 300`。JSON `success=true, exit_code=0, stderr=""`；remote SHA-256 `be2dce002dd26e3ba1faaa79e83052a3aa8b2a70ddcd76bd76c8f56142e960d3` 与 local 一致；`.test-tmp` 已先删除后重建。
- integration 精确命令：`python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d/src/backend; CI=true GOFLAGS='-v' TMPDIR='/tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d/src/backend/.test-tmp' TMP='/tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d/src/backend/.test-tmp' TEMP='/tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d/src/backend/.test-tmp' go test -tags=integration ./... > '/tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d/integration.log' 2>&1" --timeout 1800`。execution JSON `success=false, exit_code=1, stdout="", stderr="", method=native_ssh_windows`；按契约未重跑。

### 日志、migration targets 与 skip

- download 精确命令：`$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_download.py local-serv-ai "/tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d/integration.log" "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d-integration.log" --no-progress`。JSON `success=true, exit_code=0, stderr=""`；log size `1,454,761` bytes，SHA-256 `6A518272B4F359D2A84C2088F68C126DF1D399037430CA5977C492E05752566F`。
- 日志无 `--- FAIL:`，但有两个 package failure 和最终 `FAIL`：`internal/server/routes.test` 与 `internal/service.test` 均在 linker 阶段报 `/root/.vfox/sdks/golang/pkg/tool/linux_amd64/link: mapping output file failed: no space left on device`，随后分别为 `FAIL ... [build failed]`。根因是 local-serv-ai 可用磁盘不足，不是测试 assertion failure；required execution 仍为真实非零并阻断 PASS。
- `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 明确 PASS（`4.93s`）；`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` 明确 PASS（`4.20s`）。committed test 仍以两个 `require.Len(..., 12)` 约束 12/12，覆盖本地 `172_video_per_second_billing_metadata.sql`、`181_group_duplicate_operation_id.sql`，上游同号 `172_composite_model_routes.sql`、`181_prompt_audit.sql`，双 186 与 `190_add_users_email_alias_dedup_index_notx.sql`，并断言重复 apply 不增加 record count 且 checksum 不变；但这不能覆盖未链接成功的两个 package。
- 日志共有 12 个 `--- SKIP:`（11 top-level + 1 nested）：`TestDingTalkOAuthStart_Disabled` 为 Task 1.10 sentinel；`TestDialerAgainstCaptureServer` 因未设置 `TLSFINGERPRINT_CAPTURE_URL`；nested `TestConcurrencyCacheSuite/TestGetAccountsLoadBatch` 为既有 CI CurrentConcurrency TODO；`TestPromptAuditConfigCASSecretRoundTripInvalidationAndTTL`、`TestRedisPayloadStoreRoundTripTTLNamespaceAndDelete`、`TestPromptRuntimeAggregatesConfigWorkersQueueRedisEndpointsAndGuardMetrics` 因未设置 `PROMPT_AUDIT_TEST_REDIS_ADDR`；六个 `TestPromptAuditMigrationSchemaAndLeakageGate`、`TestPromptAuditDatabasePersistsFullPromptOnEventsOnly`、`TestPromptAuditRepositoryAdmissionClaimFencingAndEventTransaction`、`TestPromptAuditRepositoryForeignKeysFiltersAndStableIdentitySnapshots`、`TestPromptAuditRepositoryHighWaterAndSafeDeletion`、`TestPromptAuditServiceConfirmationKeepsPostPreviewEventsAndConcurrentDeletesAreSafe` 因未设置 `PROMPT_AUDIT_TEST_POSTGRES_DSN`。上一成功日志中的外部 OpenAI API skip 本次未出现，因为 `internal/service.test` 未完成链接，不能视为已执行或已分类的成功覆盖。

### finally cleanup、状态与风险

- remote cleanup 精确命令：`python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "rm -rf /tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d; test ! -e /tmp/sub2api-final-verify-a9bada9b52f244e8bfc39ba41e9f092d" --timeout 120`。JSON `success=true, exit_code=0, stdout="", stderr=""`；唯一 remote 目录最终不存在。
- local tar 已在 finally 删除，`archive_exists=false`；下载日志 `log_exists=true` 并保留。未构建 Sub2API 镜像、未部署、未接触服务运行目录或生产数据；Testcontainers 仅使用既有 PostgreSQL/Redis 测试镜像。未执行 Paseo、push、tag、release、merge main。
- 状态：`BLOCKED`。受控 preflight、上传、setup、下载和 cleanup 均成功，但 required integration execution exit `1`；因此 formal verify report 未修改，`docs: record final make test pass` 提交未创建，Task 27 report 保持未提交。
- 顾虑：migration targets 与 committed 12/12 assertions 已通过，但 `internal/server/routes` 和 `internal/service` 未形成可执行测试二进制；无法声称 source `6c88f1891650e0ef18b0b5ae105b8f44a069a5a4` 的完整 Linux integration PASS。
- 风险信号：local-serv-ai 在 Go linker 映射输出文件时磁盘空间耗尽；在清理服务器空间并重新启动一次全新 nonce 的完整 remote gate 前，routes/service integration 覆盖及完整 skip 集均缺证据。

## 2026-07-30 local-serv-ai ENOSPC 只读诊断

### 边界与查询记录

- 本轮为 Task 27 全新只读诊断；source/test 仍为 `6c88f1891650e0ef18b0b5ae105b8f44a069a5a4`，失败 integration nonce 仍为 `a9bada9b52f244e8bfc39ba41e9f092d`。已先加载 `ssh-skill` 与 `systematic-debugging`，先读取本报告第 133 行之后再查询；未重跑 integration，未创建或删除远程文件，未清 Go cache，未执行 Docker prune，未 inspect 生产容器内容，未递归读取 Sub2API 服务运行目录或生产数据。
- 首次合并查询的精确本机命令如下。PowerShell 参数层提前展开了原拟交给远端 shell 的变量，导致 `GO_ENV`/`GO_AND_TMP_USAGE` 段失真（`stderr` 含 `GOPATH entry is relative` 与 7 次 `stat: missing operand`）；该段不得作为路径或容量证据。命令中不依赖这些变量的 `docker system df`、容器/volume 列表可采信，并已由后续无变量查询补齐其余证据。

```powershell
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -u; GOCACHE=`$(go env GOCACHE); GOTMPDIR=`$(go env GOTMPDIR); GOPATH=`$(go env GOPATH); EFFECTIVE_GOTMPDIR=`${GOTMPDIR:-/tmp}; echo '=== GO_ENV ==='; go env GOCACHE GOTMPDIR GOPATH; echo effective_gotmpdir=`$EFFECTIVE_GOTMPDIR; echo '=== DF_HT ==='; df -hT / /tmp /var/tmp `"`$GOCACHE`" `"`$EFFECTIVE_GOTMPDIR`" `"`$GOPATH`"; echo '=== DF_INODES ==='; df -i / /tmp /var/tmp `"`$GOCACHE`" `"`$EFFECTIVE_GOTMPDIR`" `"`$GOPATH`"; echo '=== GO_AND_TMP_USAGE ==='; for p in `"`$GOCACHE`" `"`$GOPATH/pkg/mod`" `"`$GOPATH/pkg/mod/cache`" `"`$GOPATH/pkg/cache`" /tmp/go-build* /var/tmp/go-build* /tmp/sub2api-* /tmp/sub2api-*/src/backend/.test-tmp; do [ -e `"`$p`" ] || continue; du -shx `"`$p`" 2>&1; stat -c '%n|bytes=%s|mtime=%y' `"`$p`"; done; echo '=== DOCKER_SYSTEM_DF ==='; docker system df; echo '=== TESTCONTAINERS_CONTAINERS ==='; docker ps -a --filter label=org.testcontainers=true --no-trunc; echo '=== TESTCONTAINERS_VOLUMES ==='; docker volume ls --filter label=org.testcontainers=true; echo '=== ANONYMOUS_VOLUMES ==='; docker volume ls --filter label=com.docker.volume.anonymous; echo '=== DANGLING_ANONYMOUS_VOLUME_COUNT ==='; docker volume ls -q --filter dangling=true --filter label=com.docker.volume.anonymous | wc -l" --timeout 300
```

- 上述 execution JSON：`success=true, exit_code=0, method=native_ssh_windows`；`stderr` 非空，含 `go: GOPATH entry is relative; must be absolute path: "C".` 及 `stat: missing operand`。因为命令未启用 `set -e` 且末尾 `wc -l` 成功，整体 exit `0` 不代表失真段有效。
- 第二次查询完全移除远端 shell 变量，精确命令如下：

```powershell
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "echo '=== GO_ENV_JSON ==='; go env -json GOCACHE GOTMPDIR GOPATH GOENV; echo '=== TEMP_ENV ==='; env | grep -E '^(TMPDIR|TMP|TEMP)=' || true; echo '=== BASE_FS_HT ==='; df -hT / /tmp /var/tmp; echo '=== BASE_FS_INODES ==='; df -i / /tmp /var/tmp" --timeout 120
```

- 第二次 execution JSON：`success=true, exit_code=0, stderr="", method=native_ssh_windows`。
- 第三次按第二次返回的绝对路径执行定向统计，精确命令如下；只对 Go cache/module 路径及指定 `/tmp` glob 执行 `du/stat`，没有扫描服务运行目录或生产数据。`du` 同时接收 `pkg/mod` 与其子目录 `pkg/mod/cache` 时进行了重复项去重，因此随后用第四次单路径查询单列 download cache。

```powershell
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "echo '=== GO_PATH_FS_HT ==='; df -hT /root/.cache/go-build /root/.vfox/sdks/golang/packages /tmp; echo '=== GO_PATH_FS_INODES ==='; df -i /root/.cache/go-build /root/.vfox/sdks/golang/packages /tmp; echo '=== GO_CACHE_USAGE ==='; du -shx /root/.cache/go-build /root/.vfox/sdks/golang/packages/pkg/mod /root/.vfox/sdks/golang/packages/pkg/mod/cache 2>&1; stat -c '%n|bytes=%s|mtime=%y' /root/.cache/go-build /root/.vfox/sdks/golang/packages/pkg/mod /root/.vfox/sdks/golang/packages/pkg/mod/cache 2>&1; echo '=== TMP_GO_BUILD_DIRS ==='; du -shx /tmp/go-build* /var/tmp/go-build* 2>/dev/null || true; stat -c '%n|bytes=%s|mtime=%y' /tmp/go-build* /var/tmp/go-build* 2>/dev/null || true; echo '=== TMP_SUB2API_DIRS ==='; du -shx /tmp/sub2api-* 2>/dev/null || true; stat -c '%n|bytes=%s|mtime=%y' /tmp/sub2api-* 2>/dev/null || true; echo '=== TMP_SUB2API_TEST_TMP_DIRS ==='; du -shx /tmp/sub2api-*/src/backend/.test-tmp 2>/dev/null || true; stat -c '%n|bytes=%s|mtime=%y' /tmp/sub2api-*/src/backend/.test-tmp 2>/dev/null || true; echo '=== ANONYMOUS_VOLUME_USAGE ==='; docker system df -v | sed -n '/Local Volumes space usage:/,/Build cache usage:/p'; echo '=== ANONYMOUS_VOLUME_ATTACHMENTS ==='; docker ps -a --filter volume=65554aab098ee8307f208815bafb0ba23ec20f24ee7c45081b86de176817a157 --filter volume=db585d3830f4dd0a87623cb9df6d1093d576cbb1026d76b5c7d332647135bc05 --filter volume=dca5680ef653bed62ba9bebabf4376bdf8d76817bed8ebe2fa65d9df9f7c77f8 --filter volume=f01830a6ff38c219a9eb10a7bd8ff53184b99548a6af10dc4b67cd325eda82ce --no-trunc" --timeout 300
```

- 第三次 execution JSON：`success=true, exit_code=0, stderr="", method=native_ssh_windows`。
- 第四次精确命令：

```powershell
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "echo '=== GOPATH_MODULE_DOWNLOAD_CACHE_USAGE ==='; du -shx /root/.vfox/sdks/golang/packages/pkg/mod/cache; stat -c '%n|bytes=%s|mtime=%y' /root/.vfox/sdks/golang/packages/pkg/mod/cache" --timeout 120
```

- 第四次 execution JSON：`success=true, exit_code=0, stderr="", method=native_ssh_windows`。四次调用均由 `ssh_execute.py` 返回 `fallback_reason=检测到密钥需要 passphrase（建议使用 ssh-agent）`，但实际 `method=native_ssh_windows`，有效查询均 exit `0`。

### 容量、cache 与 Testcontainers 证据

- `go env -json`：`GOCACHE=/root/.cache/go-build`，`GOPATH=/root/.vfox/sdks/golang/packages`，`GOTMPDIR=""`，且进程环境没有 `TMPDIR`/`TMP`/`TEMP`，因此 Go 默认 temp 为 `/tmp`。失败 integration 还显式把 `TMPDIR`/`TMP`/`TEMP` 指向 nonce 下的 `/tmp/.../src/backend/.test-tmp`。
- `/`、`/tmp`、`/var/tmp`、`GOCACHE`、`GOPATH` 与有效 Go temp 全部位于同一 `/dev/mapper/rl-root` XFS filesystem：总量 `35G`，已用 `34G`，可用 `1.3G`，使用率 `97%`。inode 约总计 `2,899,928`、已用 `235,017`、可用 `2,664,911`、使用率 `9%`；故失败是 block capacity 耗尽，不是 inode 耗尽，也没有独立 `/tmp` filesystem 可隔离 linker 峰值。
- `GOCACHE` 占 `14G`，目录 mtime `2026-03-16 15:41:32.570707847 +0800`；`GOPATH/pkg/mod` 占 `1017M`，mtime `2026-07-27 01:28:35.906635560 +0800`，其中 `pkg/mod/cache` 单独占 `215M`，mtime `2026-07-27 01:28:19.700722706 +0800`。module download cache 已包含于约 `1017M` 的 module 总量，不应重复相加；当前最显著且可再生的占用是 `14G` Go build cache。
- 当前没有 `/tmp/go-build*`、`/var/tmp/go-build*` 或 `/tmp/sub2api-*/src/backend/.test-tmp`；失败 nonce 目录确已不在。唯一匹配 `/tmp/sub2api-*` 的是 `/tmp/sub2api-request-body-2906762940`，占 `21M`，mtime `2026-07-27 01:32:42.847155995 +0800`；它早于本 nonce且无法归属本 change，本轮未触碰，也不建议删除。
- `docker system df`：images `18`/active `5`、`4.668GB`、reclaimable `1.193GB`；containers `5` 且全部 active、`225MB`、reclaimable `0B`；local volumes `4`/active `1`、size/reclaimable 均 `0B`；build cache `0B`。当前 `org.testcontainers=true` 容器和 volume 均为空。
- 四个 anonymous volume 分别为 `65554aab...a157`、`db585d38...bc05`、`dca5680e...77f8`、`f01830a6...82ce`，Docker verbose 统计均为 `0B`；其中 3 个 dangling。唯一 links=`1` 的 `db585d38...bc05` 挂载于健康运行的生产容器 `sub2api-postgres`，绝不能触碰；其余三个没有 Testcontainers label，无法安全归属本次 change，且报告 `0B`，也不是本次 ENOSPC 的有效处置对象。

### 根因、恢复判断与最小安全处置

- 根因判断：失败日志已证明 filesystem 在 `internal/server/routes.test` 与 `internal/service.test` linker 映射输出的峰值达到 ENOSPC；本次快照证明 linker temp、Go build cache、module cache 和根卷共享同一仅 `35G` 的 filesystem，其中持久 `GOCACHE` 已占 `14G`。容量压力是直接根因，inode、Docker build cache、活动 Testcontainers 容器或 Testcontainers volume 均不是根因。
- 清理失败 nonce 后已从 linker 的不可分配状态恢复到 `1.3G` 可用，但仅有约 `3%` block headroom。因为 nonce 已删除且失败前/峰值没有 `df` 采样，无法精确量化该 nonce 的 transient high-water；然而下一次仍需在同一根卷重建 source/test temp 并链接此前未生成的两个测试二进制。Go cache 可能因失败运行而变暖，但现有证据不足以证明 `1.3G` 能容纳下一峰值，仍有很高概率再次 ENOSPC，不应直接重跑。
- 建议的精确最小处置：后续经用户明确授权、并确认没有并发 Go build/test 后，仅执行 `python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "go clean -cache" --timeout 300`，清理可再生的 `14G` Go build cache；随后只读复查 `df -hT / /tmp /var/tmp` 与 `df -i / /tmp /var/tmp`，确认实际回收后再由全新 nonce 执行一次完整 remote gate。当前 change 无遗留目录可先清，且 `1017M` module cache 不是达到足够余量所必需，故不建议 `go clean -modcache`。
- 安全边界：不得 Docker prune，不得删除 image/container/volume，不得删除三个无法归属的 dangling anonymous volumes，不得删除 `/tmp/sub2api-request-body-2906762940` 或他人临时目录，不得触碰 `sub2api-postgres`、服务运行目录或生产数据。以上均未执行。
- 状态仍为 `BLOCKED`：只读诊断已闭环 ENOSPC 根因与最小处置，但 required integration 仍为真实 exit `1`；只有获授权清理可再生 build cache并用全新 nonce 取得完整 Linux integration PASS 后才能解除。
- 顾虑：无失败前/峰值精确容量采样，故不能给出 transient bytes；`GOCACHE` 可能由同一账号的其他构建共享，清理前需排除并发构建。风险信号是根卷 `97%`、仅余 `1.3G`，而两个目标测试二进制尚未链接成功。

## 2026-07-30 ENOSPC 最小恢复与最终 remote PASS

### 并发排除、cache 清理与容量复查

- 本轮为 Task 27 全新远程恢复与验证代理；已先加载 `ssh-skill` 与 `systematic-debugging`，先读取本报告第 133 行之后，未重跑任何本地门禁。固定 source/test commit 为 `6c88f1891650e0ef18b0b5ae105b8f44a069a5a4`。
- 不自匹配进程查询精确命令：`python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "echo '=== GO_TOOLCHAIN_COMM ==='; ps -C go,compile,link,asm,cgo,vet -o pid=,ppid=,etimes=,comm=,args= || true; echo '=== GO_TEST_BINARIES ==='; ps -eo pid=,ppid=,etimes=,comm=,args= | grep -E '[[:space:]][^[:space:]]+[.]test([[:space:]]|$)' || true; echo '=== END_GO_PROCESS_CHECK ==='" --timeout 120`。JSON `success=true, exit_code=0, stderr=""`；`GO_TOOLCHAIN_COMM` 与 `GO_TEST_BINARIES` 均无进程行，明确排除 `go`、`compile`、`link`、`asm`、`cgo`、`vet` 与 Go test binaries 并发。
- 无并发后仅执行一次 `python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "go clean -cache" --timeout 300`；JSON `success=true, exit_code=0, stdout="", stderr=""`。未清 modcache/testcache，未 Docker prune，未删除 image/container/volume、未知 `/tmp`、服务目录或生产数据。
- 复查 JSON `success=true, exit_code=0, stderr=""`：`/root/.cache/go-build` 为 `20K`；`df -hT / /tmp /var/tmp` 均为 `/dev/mapper/rl-root` XFS、总量 `35G`、已用 `20G`、可用 `15G`、使用率 `58%`；`df -i` 均为 inode 使用率 `2%`。相对诊断时 `14G` GOCACHE/仅余 `1.3G`，已实际回收出足够门禁余量。

### 新 nonce、archive 与执行

- stage `final-verify`；新 nonce `860617d8c7d8427a944f30c0a915c894`；唯一 remote 目录 `/tmp/sub2api-final-verify-860617d8c7d8427a944f30c0a915c894`；local archive `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-860617d8c7d8427a944f30c0a915c894.tar`；local log `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-860617d8c7d8427a944f30c0a915c894-integration.log`。
- archive 只由 `git archive --format=tar --output="C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-860617d8c7d8427a944f30c0a915c894.tar" 6c88f1891650e0ef18b0b5ae105b8f44a069a5a4` 创建，exit `0`；size `53,739,520` bytes；local SHA-256 `BE2DCE002DD26E3BA1FAAA79E83052A3AA8B2A70DDCD76BD76C8F56142E960D3`。
- 沿用不含 Docker brace/template 的简单 preflight：`set -eu; test ! -e <remote>; mkdir -m 700 <remote>; go version; docker version; docker info >/dev/null; echo preflight=ok`。JSON `success=true, exit_code=0, stderr=""`；Go `1.26.5 linux/amd64`，Docker client/server `29.2.1`，`docker info` 成功。
- upload JSON `success=true, exit_code=0, stderr=""`。setup JSON `success=true, exit_code=0, stderr=""`；remote SHA-256 `be2dce002dd26e3ba1faaa79e83052a3aa8b2a70ddcd76bd76c8f56142e960d3` 与 local 一致；解包后删除并重建 `src/backend/.test-tmp`。
- integration 精确命令：`CI=true GOFLAGS='-v' TMPDIR='/tmp/sub2api-final-verify-860617d8c7d8427a944f30c0a915c894/src/backend/.test-tmp' TMP='/tmp/sub2api-final-verify-860617d8c7d8427a944f30c0a915c894/src/backend/.test-tmp' TEMP='/tmp/sub2api-final-verify-860617d8c7d8427a944f30c0a915c894/src/backend/.test-tmp' go test -tags=integration ./... > '/tmp/sub2api-final-verify-860617d8c7d8427a944f30c0a915c894/integration.log' 2>&1`，timeout `1800`；未加 `-p`、未重试或改变测试语义。execution JSON `success=true, exit_code=0, stdout="", stderr=""`。

### 日志、migration、skip 与 finally

- download JSON `success=true, exit_code=0, stderr=""`。本地日志 size `4,264,020` bytes，SHA-256 `26B347735ED1D244775A835A86C4E709D29FD5DC498E92981F55DFAC09BCA47D`；`--- FAIL:`/package `FAIL` count `0`。
- `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 明确 PASS（`5.34s`）；`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` 明确 PASS（`4.80s`）。committed test 的 upstream list 与 embedded FS 两个 `require.Len(..., 12)` 均由通过 target 执行，覆盖本地/上游 172/181、双 186、`190_*_notx.sql`、full filename identity、重复 apply record count 与 checksum 不变，形成 12/12 证据。
- 日志共有 13 个 `--- SKIP:`（12 top-level + 1 nested）：`TestDingTalkOAuthStart_Disabled` 为 Task 1.10 sentinel；`TestDialerAgainstCaptureServer` 未设置 `TLSFINGERPRINT_CAPTURE_URL`；nested `TestConcurrencyCacheSuite/TestGetAccountsLoadBatch` 为既有 CI CurrentConcurrency TODO；3 个 prompt-audit Redis tests 未设置 `PROMPT_AUDIT_TEST_REDIS_ADDR`；6 个 prompt-audit PostgreSQL tests 未设置 `PROMPT_AUDIT_TEST_POSTGRES_DSN`；`TestEstimateOpenAIInputTokens_CompareWithOpenAIAPI` 未设置 `OPENAI_API_KEY`。所有 skip 已分类，均不命中 migration targets 或 Task 27 required affected surface。
- 测试后只读 `df` JSON `success=true, exit_code=0, stderr=""`：`/`、`/tmp`、`/var/tmp` 均为总量 `35G`、已用 `22G`、可用 `14G`、使用率 `62%`，inode 使用率 `2%`；未做额外 cache 或 Docker 清理。
- finally remote cleanup JSON `success=true, exit_code=0, stderr=""`；唯一 remote 目录不存在。local tar cleanup exit `0`，`archive_exists=False`；下载日志 `log_exists=True` 并保留。preflight/upload/test/download/cleanup 全部零退出。
- 所有远程操作仅使用 ssh-skill Python scripts；未使用 raw SSH/SCP。未执行 Paseo、push、tag、release、deploy、merge main 或 Sub2API 镜像构建，未修改源码/测试、OpenSpec plan/tasks/progress、Task 4 report、`.comet/current-change.json`、`paseo.json` 或其他文件。

### 最终状态

- 最终 source/test SHA：`6c88f1891650e0ef18b0b5ae105b8f44a069a5a4`。既有 focused、最终唯一原样 `make test` exit `0`、版本化 build、两轮 generate 与 static 证据沿用第 103-117 行，不在本轮重跑。
- 全部失败历史保持原样：三次 `make test` exit `2`、首次含 Docker brace/template 的 SSH preflight 本机解析 exit `1`、第二次 remote integration linker ENOSPC exit `1`，以及第 165-215 行只读诊断均未改写为 PASS。
- 状态：`DONE_WITH_CONCERNS`。required final remote integration 已 exit `0`，两个 migration targets、12/12、完整 13 skip 分类、下载和 cleanup 均闭环。
- 最终只暂存 formal verify report 并创建提交 `0884b595c docs: record final make test pass`；该提交只有 formal report。Task 27 report 保持未提交，未 amend。
- 顾虑：OAuth HeapAlloc 使用 process-wide 指标，虽最终原样 gate 与 remote integration 已通过，历史敏感性仍保留；本次曾需清理共享 GOCACHE，后续并发门禁仍需在清理前排除共享构建。
- 风险信号：local-serv-ai 的 GOCACHE 曾增长至 `14G` 并将根卷推到 `97%`；本次测试后尚余 `14G`，但没有长期容量配额或自动告警证据。

## 2026-07-30 thorough review 第 1 轮 fixer

### 起点与生产形态 RED

- 起始 HEAD：`aa5b576fef3d1dfcf397463274ae67c08229cfdd`；设计提交 `9fd9976d7`、计划增量提交 `aa5b576fe` 均已在 HEAD。起始 dirty 为 `M .superpowers/sdd/task-27-report.md`、`M .superpowers/sdd/task-4-report.md`、`M opencode.json`、`?? .comet/current-change.json`、`?? paseo.json`；禁止暂存项均保留。
- `go -C backend test -tags=unit ./internal/service -run '^TestContentModerationCheckLazy(SkipsBodyWhenDisabled|DefersBodyUntilRequestIsInScope)$' -count=1 -v`：exit `1`，按计划因 `ContentModerationService.CheckLazy undefined` 编译 RED。
- `go -C backend test ./internal/securityaudit -run '^TestCoordinatorCheckLazy(EvaluatesBodyOnceAcrossPromptAndLegacy|SkipsBodyWhenAllEnginesAreOff)$' -count=1 -v`：exit `1`，按计划因 `Coordinator.CheckLazy undefined` 编译 RED。
- `go -C backend test -tags=unit ./internal/handler -run '^TestOpenAIImages_(UnifiedAuditRunsLegacyOnce|SecurityAuditUsesFrozenPayloadBeforeRelease)$' -count=1 -v`：exit `1`；audit-only frozen prompt/image 测试 PASS，生产形态同步 Images 测试真实执行 direct moderation 与 coordinator 内同一 legacy adapter，断言明确为 `expected 1, actual 2`。

### GREEN 与 source commit

- 步骤 4 三条原样命令重跑均 exit `0`：service 覆盖全局/config/mode 关闭和 group/model scope；coordinator 覆盖 Blocking prompt + legacy 共享一次 provider 及双关闭零求值；Images 覆盖生产注入 legacy 单次与 audit-only 完整 prompt/image。
- `go -C backend test -tags=unit ./internal/handler -run '^TestOpenAIImages_(OAuthTextIsReleasedBeforeBlockedUpstream|ContentModerationUsesFrozenPayloadBeforeRelease|UnifiedAuditRunsLegacyOnce|SecurityAuditUsesFrozenPayloadBeforeRelease)$' -count=10 -v`：exit `0`，40/40 PASS，未提高 12 MiB HeapAlloc ceiling。
- `go -C backend test ./internal/securityaudit -count=1`、`go -C backend test -tags=unit ./internal/service -count=1`、`go -C backend test -tags=unit ./internal/handler -count=1` 均 exit `0`，其中 service `154.376s`、handler `96.423s`；`golangci-lint run ./internal/handler/... ./internal/securityaudit/... ./internal/service/...` exit `0`、`0 issues.`。
- `gofmt -d` 与 `git diff --check` exit `0`。只暂存计划步骤 4 的 9 个 Go 源码/测试；source commit：`90b008901 fix: unify image security audit payload`，未 amend。

### 新 source HEAD 本地 full gate 首次结果

- 在 `90b0089010c77d0fef1790e2e8cf3b675d4994cd` 上原样执行 `make test` 一次：exit `2`，不得记为 PASS。default Go tests、`golangci-lint run ./...`（`0 issues.`）和 unit handler 均通过；unit service 唯一失败为既有 `TestPassthroughLifecycle_LeaseLossSendsRetryClose`，随后 frontend targets 未执行。完整输出：`C:/Users/caiqy/.local/share/opencode/tool-output/tool_fb0dc8a92001b12CrNYUOz3RYV`。
- 失败日志中服务端明确得到 `ErrOpenAIWSIngressLeaseLost` 并映射为 1013，但测试客户端断言处只读到 `failed to read frame header: EOF`，未得到 `websocket.CloseError`。该测试及 `Close` 后紧接 `CloseNow` 的路径不在本轮允许修改范围，且本轮 source commit 前完整 unit service 已 exit `0`；先做精确目标重复性诊断，不改 WebSocket 源码/测试。
- `go -C backend test -tags=unit ./internal/service -run '^TestPassthroughLifecycle_LeaseLossSendsRetryClose$' -count=10 -v`：exit `1`，9 次 PASS、1 次同型 EOF；每次服务端 trace 都已产生 1013 lease-loss close error。由此确认既有 close-frame 竞态可重复但非必现；在不扩展允许源码范围的前提下仅做一次受控原样 full-gate 复验，首次 exit `2` 永久保留。

## 2026-07-30 第 1 轮 fixer continuation 最终记录

### 精确断点与本地门禁

- continuation 起点：`HEAD=90b0089010c77d0fef1790e2e8cf3b675d4994cd`；源码/测试无未提交 diff，暂存区为空。dirty 仅为六个明确禁止暂存项。无遗留 `make`、`go`、`pnpm`；现有 Node 进程仅为 Chrome DevTools MCP。
- `git worktree list --porcelain` 发现两个目录已不存在的历史 `sub2api-task27-*` prunable metadata；没有中断中的 generate worktree。随后只清理这些 stale metadata，并创建全新 detached worktree。
- 未重跑原样 `make test`。保留第 268-272 行在 source `90b008901` 上的 exit `2` 及精确目标 9 PASS/1 EOF；用户接受客户端 EOF/服务端 1013 的既有上游基线例外，但两条非零命令均未记为 PASS。
- 首次 `make test` 未启动的 frontend constituents：`pnpm --dir frontend run lint:check`、`pnpm --dir frontend run typecheck`、`pnpm --dir frontend run test:run` 均 exit `0`；Vitest `215/215 files`、`1626/1626 tests`，duration `71.55s`。
- 同 source 已有完整 backend 组成证据继续有效：default constituent、全量 lint（`0 issues.`）、unit handler package、独立完整 unit service package及 securityaudit package均 exit `0`。未以这些零退出改写原样 `make test` 的 exit `2`。
- `make "VERSION=0.1.165.1" "SHELL=D:/scoop/shims/bash.exe" build`：exit `0`；Vite `1019` modules；VERSION 文件/二进制字符串均为 `0.1.165.1`，binary metadata 为 Go `1.26.5`、`vcs.revision=90b0089010c77d0fef1790e2e8cf3b675d4994cd`。

generate 精确命令：

```text
git worktree prune && git worktree add --detach "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-task27-generate-90b008901" 90b0089010c77d0fef1790e2e8cf3b675d4994cd
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git status --short
git worktree remove "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-task27-generate-90b008901"
```

- 全部 exit `0`；两轮 Ent/Wire diff 无输出，worktree clean；移除后 `Test-Path=False`，worktree 列表只剩主树。
- static：`git diff --check` exit `0`（仅禁止暂存文件的 LF/CRLF warning）；unmerged diff/index、tracked conflict marker、legacy first-token count 均为 `0`；VERSION 为 `0.1.165.1`；14 个 protected migration 全部存在。
- `git diff -- opencode.json` exit `0` 但有用户新增 CodeGraph/ddg-search 配置；按契约原样保留，不暂存且不写成无 diff。

### 新 nonce remote final gate

- 已先加载 `ssh-skill`，未使用 raw SSH/SCP。stage `final-verify`；nonce `547fff8e23a24a92ad566202331ba360`；remote `/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360`；local tar/log 分别为 `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-547fff8e23a24a92ad566202331ba360.tar` 与 `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-547fff8e23a24a92ad566202331ba360-integration.log`。

archive 精确命令：

```text
git archive --format=tar --output="C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-547fff8e23a24a92ad566202331ba360.tar" 90b0089010c77d0fef1790e2e8cf3b675d4994cd
```

- exit `0`；size `53,770,240`；SHA-256 `F7121A186EAFD3F80E86F52EA02F264BF7B945559D24757EA9DF0BE87E0743B8`。

preflight 精确命令：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; test ! -e /tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360; mkdir -m 700 /tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360; go version; docker version; docker info >/dev/null; df -hT / /tmp /var/tmp; df -i / /tmp /var/tmp; echo preflight=ok" --timeout 120
```

- JSON `success=true, exit_code=0`；Go `1.26.5 linux/amd64`、Docker client/server `29.2.1`；根卷可用 `14G`、使用率 `60%`，inode `2%`。

upload 精确命令：

```text
$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_upload.py local-serv-ai "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-547fff8e23a24a92ad566202331ba360.tar" "/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/source.tar" --no-progress
```

- JSON `success=true, exit_code=0`。

setup 精确命令：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360; sha256sum source.tar; mkdir src; tar -xf source.tar -C src; rm -f source.tar; rm -rf src/backend/.test-tmp; mkdir -p src/backend/.test-tmp; test -d src/backend/.test-tmp; echo setup=ok" --timeout 300
```

- JSON `success=true, exit_code=0`；remote hash `f7121a186eafd3f80e86f52ea02f264bf7b945559d24757ea9df0be87e0743b8` 一致；`.test-tmp` 重建成功。

integration 精确命令：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/src/backend; CI=true GOFLAGS='-v' TMPDIR='/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/src/backend/.test-tmp' TMP='/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/src/backend/.test-tmp' TEMP='/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/src/backend/.test-tmp' go test -tags=integration ./... > '/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/integration.log' 2>&1" --timeout 1800
```

- JSON `success=true, exit_code=0`；未加 `-p`、未重试。

download 精确命令：

```text
$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_download.py local-serv-ai "/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/integration.log" "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-547fff8e23a24a92ad566202331ba360-integration.log" --no-progress
```

- JSON `success=true, exit_code=0`；log size `4,279,146`，SHA-256 `D5BBE067DBBAB72529073539AEA60EF96F567E807B9A4C0591F374F685A021D6`；FAIL count `0`。

cleanup 精确命令：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; df -hT / /tmp /var/tmp; df -i / /tmp /var/tmp; rm -rf /tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360; test ! -e /tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360; echo cleanup=ok" --timeout 120
```

- JSON `success=true, exit_code=0`；测试后根卷可用 `14G`、使用率 `62%`、inode `2%`。remote 目录不存在；local tar 删除 exit `0`、`archive_exists=False`；log 保留、`log_exists=True`。

### Remote 结果、顾虑与风险信号

- 两个 migration targets 明确 PASS：新库 `5.23s`、local v0.1.159.6 升级 `4.98s`；后者执行两条 12/12 `require.Len`，并覆盖双方 172/181、双 186、190 notx、重放 count/checksum。
- 13 skips（12 top-level + 1 nested）完整分类：DingTalk sentinel、外部 TLS capture、既有 CurrentConcurrency TODO、3 个 prompt-audit Redis fixture、6 个 prompt-audit PostgreSQL fixture、外部 OpenAI token comparison；均不命中 required targets/affected surface。
- 未 Docker prune、未构建 Sub2API 镜像、未部署、未触碰服务目录或生产数据；未 Paseo、push/tag/release、merge main。
- 顾虑：最终 source 原样 `make test` 仍是 exit `2`，只能依据用户接受的精确 EOF/1013 基线例外与同 source 完整 constituent/package 证据给出 concerns 状态；Windows 仍未跑 `-race`。
- 风险信号：lease-loss 客户端 close frame 存在 9/10 可见、1/10 EOF 的既有竞态；OAuth HeapAlloc 仍是 process-wide 指标；local-serv-ai GOCACHE 历史曾将根卷推到 97%，本次前后均保有 `14G` 但没有长期配额/告警证据。

### 提交与最终状态

- 首次 `git add -- "docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-verify.md"` 因 `.gitignore:138` 的 `docs/*` 规则打印 ignored path 提示并返回非零，故其后的 `&&` 检查未执行；诊断确认该文件本来就是 tracked，且本次内容实际已完整进入 index，其他文件未暂存。没有重复修改 ignore 配置或扩大 pathspec。
- 随后 `git diff --cached --name-only` 只输出 formal report；`git diff --cached --check` exit `0`；cached stat 为 1 file、126 insertions、5 deletions。`git commit -m "docs: record unified image audit verification"` exit `0`，提交 `ac3d1b833`，只包含 formal report，未 amend。
- 最终状态：`DONE_WITH_CONCERNS`。required remote final gate、build、双 generate、static、backend/frontend constituent/package 均有零退出证据；最终 source 原样 `make test` 保持 exit `2`，仅按用户批准的 EOF/1013 上游基线例外接受，未称 PASS。

## 2026-07-30 thorough review 第 2/2 轮 fixer

### 起点与 Linux race RED 前置阻塞

- 起始 `HEAD=ac3d1b833fc9591353d4e71427f027183bea4a5e`，待验证 source/test commit 为 `90b0089010c77d0fef1790e2e8cf3b675d4994cd`；暂存区为空。既有 dirty 六项原样保留且未暂存。
- reviewer 已确认上一轮 4 项均 CLOSED、Spec compliance PASS；本轮唯一 Important 是 `checkBlockingLazy` 的 legacy/prompt goroutine 捕获同一 `req`，prompt 写 `req.Body` 可与 legacy 复制 `req` 并发；Minor 是 lazy provider owning API 缺同步生命周期契约。代码核对确认 `sync.Once` 只保护 provider，不保护共享 `Request`。
- 本节同时修正本报告第 11 行的历史角色措辞：禁止的是 subagent 暂存；主协调器可按 Comet 协议持久化 checkpoint。`aa5b576fe` 中与 plan 一起提交的 `.comet/subagent-progress.md` 是合法协调器提交，无需回退或前向删除。

固定 archive 精确命令：

```text
git archive --format=tar --output="C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-race-red-01a3c2b3c9544c17bc996808a02d0b4c.tar" 90b0089010c77d0fef1790e2e8cf3b675d4994cd
```

- exit `0`；size `53,770,240` bytes；SHA-256 `f7121a186eafd3f80e86f52ea02f264bf7b945559d24757ea9df0be87e0743b8`。stage `race-red`，nonce `01a3c2b3c9544c17bc996808a02d0b4c`，唯一 remote 目录 `/tmp/sub2api-race-red-01a3c2b3c9544c17bc996808a02d0b4c`。

按要求先确认 gcc/CGO 的 preflight 精确命令：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; test ! -e /tmp/sub2api-race-red-01a3c2b3c9544c17bc996808a02d0b4c; mkdir -m 700 /tmp/sub2api-race-red-01a3c2b3c9544c17bc996808a02d0b4c; go version; printf 'go_env_cgo='; go env CGO_ENABLED; gcc --version; printf 'df_before\n'; df -hT / /tmp; echo preflight=ok" --timeout 120
```

- JSON `success=false, exit_code=1`；stdout 为 Go `1.26.5 linux/amd64`、`go_env_cgo=0`；stderr 为 `bash: line 1: gcc: command not found`。前置不满足，未上传或解包 archive，未运行指定 `go test -race`，因此没有 race log，也没有伪造 `DATA RACE` RED。
- 一次包含 shell loop 的只读定位调用先被本机 `ssh_execute.py` 参数解析拒绝，exit `2`，未执行远端命令；缩短后的 loop 形状又在 native Windows fallback 报 `The filename or extension is too long`，exit `1`，也未形成远端证据。随后使用下列简单只读命令完成确认：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -u; command -v gcc || true; command -v cc || true; command -v clang || true; rpm -q gcc || true; find /opt/rh /usr/bin /usr/local/bin -maxdepth 4 -type f -name gcc -perm /111 -print 2>/dev/null || true" --timeout 120
```

- JSON `success=true, exit_code=0`；`gcc`、`cc`、`clang` 与常见 toolset 路径均无结果，唯一 stdout 为 `package gcc is not installed`。未安装编译器，也未变更服务器 package、Go 全局环境或源码。

cleanup 精确命令：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; rm -rf /tmp/sub2api-race-red-01a3c2b3c9544c17bc996808a02d0b4c; test ! -e /tmp/sub2api-race-red-01a3c2b3c9544c17bc996808a02d0b4c; echo cleanup=ok" --timeout 120
```

- JSON `success=true, exit_code=0`，stdout `cleanup=ok`；local archive 删除后 `archive_exists=False`。全程只使用 ssh-skill scripts，未使用 raw SSH/SCP，未 Docker prune、构建镜像、部署或访问服务/生产数据。

### 状态、顾虑与风险信号

- 状态：`BLOCKED`。用户明确要求未复现 race 时不得猜改；因此未修改 `coordinator.go`/测试/formal report，未创建 source/docs commit，也未运行后续 focused、lint、`make test`、build、generate、static 或 integration 门禁。
- 顾虑：`local-serv-ai` 当前缺少 Linux race detector 的必要 C toolchain；需要用户提供具备 gcc/CGO 的允许环境，或明确授权最小安装后，才能重新从固定 source commit 取得 `DATA RACE` RED。
- 风险信号：reviewer 指出的共享 `req` 并发访问仍存在于 source `90b008901`；在 RED 前置恢复前保持未修复状态，不能声明 race-clean。

## 2026-07-30 thorough review 第 2 轮 race 环境只读诊断

### 边界与仓库状态

- 本轮为全新只读环境诊断；已先加载 `systematic-debugging`。固定待验证 source/test 为 `90b0089010c77d0fef1790e2e8cf3b675d4994cd`，当前 `HEAD=ac3d1b833fc9591353d4e71427f027183bea4a5e`。
- 起始 `git status --short` 精确输出为 `M .superpowers/sdd/task-27-report.md`、`M .superpowers/sdd/task-4-report.md`、`M opencode.json`、`M openspec/changes/staged-merge-upstream-v0-1-165/.comet/subagent-progress.md`、`?? .comet/current-change.json`、`?? paseo.json`；`git diff --cached --name-only` exit `0` 且无输出。以上用户 dirty 均未修改、删除或暂存，本轮只追加本报告。
- 未运行任何 `go test`，未安装软件、修改仓库源码/测试/配置或系统环境，未联网下载、操作服务器、创建提交或暂存文件。第 375-418 行记录的 Linux race RED 缺口保持不变。

### Windows Go 与 PATH

精确命令与输出：

```powershell
go version
```

```text
go version go1.26.3 windows/amd64
```

```powershell
go env GOOS GOARCH CGO_ENABLED CC CXX
```

```text
windows
amd64
0
gcc
g++
```

- 五行依次对应 `GOOS=windows`、`GOARCH=amd64`、`CGO_ENABLED=0`、`CC=gcc`、`CXX=g++`。`CC`/`CXX` 是 Go 默认名称，不证明可执行文件存在。

```powershell
'gcc','clang','clang-cl','zig','cc','x86_64-w64-mingw32-gcc' | ForEach-Object { $name = $_; $commands = @(Get-Command -Name $name -All -ErrorAction SilentlyContinue); if ($commands.Count -eq 0) { "${name}: NOT FOUND" } else { $commands | ForEach-Object { "${name}: $($_.Source) [$($_.CommandType)]" } } }
```

```text
gcc: NOT FOUND
clang: NOT FOUND
clang-cl: NOT FOUND
zig: NOT FOUND
cc: NOT FOUND
x86_64-w64-mingw32-gcc: NOT FOUND
```

- 命令 exit `0`；PATH 中没有可供 Windows `cgo`/race 使用的 C 编译器。当前直接执行 `go -C backend test -race ./internal/securityaudit` 会受 `CGO_ENABLED=0` 阻断；即使仅把 CGO 改为 `1`，默认 `CC=gcc` 也无法解析。

### Scoop、MSYS2 与 Visual Studio LLVM

- Scoop 实际根目录 `D:\scoop\apps` 存在，一级已安装应用共 62 项，其中没有 `gcc`、`mingw`、`llvm`、`zig` 或 `msys2`；`C:\Users\caiqy\scoop\apps` 不存在。
- 对 `C:\Users\caiqy\scoop`、`D:\scoop` 的 `apps\gcc\current\bin\gcc.exe`、`apps\mingw\current\bin\gcc.exe`、`apps\llvm\current\bin\clang.exe`、`apps\llvm\current\bin\clang-cl.exe`、`apps\zig\current\zig.exe`、`apps\msys2\current\ucrt64\bin\gcc.exe`、`apps\msys2\current\mingw64\bin\gcc.exe`、`apps\msys2\current\clang64\bin\clang.exe`、`apps\msys2\current\usr\bin\gcc.exe` 定向执行 `Test-Path -LiteralPath`；18 项输出全部为 `absent`。
- 对 `C:\msys64\{ucrt64,mingw64}\bin\gcc.exe`、`C:\msys64\clang64\bin\clang.exe`、`C:\msys64\usr\bin\gcc.exe`、`C:\tools\msys64\{ucrt64,mingw64}\bin\gcc.exe`、`D:\msys64\{ucrt64,mingw64}\bin\gcc.exe`、`C:\Program Files\LLVM\bin\{clang,clang-cl}.exe` 定向执行 `Test-Path -LiteralPath`；10 项输出全部为 `absent`。

Visual Studio 发现命令与输出：

```powershell
& 'C:\Program Files (x86)\Microsoft Visual Studio\Installer\vswhere.exe' -all -products '*' -property installationPath
```

```text
C:\Program Files (x86)\Microsoft Visual Studio\2017\BuildTools
C:\BuildTools
```

```powershell
& 'C:\Program Files (x86)\Microsoft Visual Studio\Installer\vswhere.exe' -all -products '*' -requires Microsoft.VisualStudio.Component.VC.Llvm.Clang -property installationPath
```

```text
<无输出>
```

- LLVM component 查询 exit `0` 但无匹配。另对两个 installation root 下的 `VC\Tools\Llvm\bin\{clang,clang-cl}.exe`、`VC\Tools\Llvm\x64\bin\{clang,clang-cl}.exe`、`Common7\IDE\CommonExtensions\Microsoft\CMake\LLVM\bin\{clang,clang-cl}.exe` 共 12 个常见入口定向执行 `Test-Path -LiteralPath`，输出全部为 `absent`。未做全盘递归搜索。

### Git Bash/MSYS

发现命令与输出：

```powershell
Get-Command git,bash -All -ErrorAction SilentlyContinue | Select-Object Name,CommandType,Source | Format-Table -AutoSize
git --exec-path
```

```text
Name     CommandType Source
----     ----------- ------
git.exe  Application C:\Program Files\Git\cmd\git.exe
bash.exe Application D:\scoop\shims\bash.exe
bash.exe Application C:\Users\caiqy\AppData\Local\Microsoft\WindowsApps\bash.exe

C:/Program Files/Git/mingw64/libexec/git-core
```

```powershell
& 'C:\Program Files\Git\bin\bash.exe' --noprofile --norc --version
```

```text
GNU bash, version 5.2.37(1)-release (x86_64-pc-msys)
```

```powershell
& 'C:\Program Files\Git\bin\bash.exe' --noprofile --norc -lc 'if command -v gcc >/dev/null 2>&1; then command -v gcc; gcc --version; else printf "gcc: NOT FOUND\n"; fi'
```

```text
gcc: NOT FOUND
```

- Git for Windows 的 `C:\Program Files\Git\bin\bash.exe` 存在；`C:\Program Files\Git\mingw64\bin\gcc.exe` 与 `C:\Program Files\Git\usr\bin\gcc.exe` 均不存在。用户级 Git 及 `D:\scoop\apps\git\current` 的同类入口也不存在。因此现有 Git Bash/MSYS 只能提供 shell，不能为 Go Windows race 提供 GCC。

### WSL

精确命令 `wsl.exe --status` exit `0`。该 Windows 本地化命令输出 UTF-16LE，工具终端直接显示时乱码；只读内存捕获的原始 stdout 十六进制为：

```text
D89EA48B0652D1533A0020005500620075006E00740075000D000A00D89EA48B48722C673A00200032000D000A00535F4D52A18B977B3A674D916E7F0D4E2F6501632000570053004C00310002300D000A00E58281897F4F28752000570053004C0031000CFFF78B2F5428751C20570069006E0064006F00770073002000530075006200730079007300740065006D00200066006F00720020004C0069006E00750078001D20EF530990C47EF64E02300D000A00
```

按 UTF-16LE 精确解码为：

```text
默认分发: Ubuntu
默认版本: 2
当前计算机配置不支持 WSL1。
若要使用 WSL1，请启用“Windows Subsystem for Linux”可选组件。
```

```powershell
wsl.exe -l -q
```

```text
Ubuntu
```

- `wsl.exe -d Ubuntu -- true` exit `0`、无输出，确认该 WSL2 发行版可启动。随后三条只读命令的精确结果如下，三者均 exit `127`：

```powershell
wsl.exe -d Ubuntu -- go version
# /bin/bash: line 1: go: command not found

wsl.exe -d Ubuntu -- gcc --version
# /bin/bash: line 1: gcc: command not found

wsl.exe -d Ubuntu -- go env CGO_ENABLED
# /bin/bash: line 1: go: command not found
```

- 因此 Ubuntu 虽可启动，但没有 Linux Go toolchain，也没有 race detector 所需 GCC；`go env CGO_ENABLED` 无法取得，不是值为 `0`。本轮未安装包或复用 Windows Go 二进制冒充 Linux toolchain。

### 候选结论、推荐命令与顾虑

| 候选 | 当前能否原样运行 | 判定 |
|---|---:|---|
| Windows Go 1.26.3 | 否 | `CGO_ENABLED=0`，PATH 及定向常见入口均无可用 GCC/Clang/zig/cc |
| Git Bash/MSYS | 否 | Bash 可用，但不包含 GCC |
| Scoop/MSYS2 | 否 | 对应应用未安装，常见入口不存在 |
| Visual Studio LLVM | 否 | 两个 Build Tools installation 均无 LLVM component/入口 |
| WSL2 Ubuntu | 否 | 发行版可启动，但 `go` 与 `gcc` 均不存在 |
| local-serv-ai | 否 | 第 391-404 行已有 `CGO_ENABLED=0`、`gcc/cc/clang` 均不存在的只读证据；本轮按禁止操作服务器约束未重查 |

- 当前可用候选：无。状态：`BLOCKED`；没有在不改仓库/系统、不卡在缺失工具链前置条件的现成环境，故未执行 race RED，也不能声明 source `90b008901` race-clean。
- 若后续由用户提供一个**已具备** Linux Go 与 GCC 的 WSL 发行版，且依赖已缓存、仍禁止联网，推荐从仓库根目录执行：

```powershell
wsl.exe -d Ubuntu --cd /mnt/d/Caiqy/Projects/Github/sub2api/backend -- env CGO_ENABLED=1 GOPROXY=off go test -race ./internal/securityaudit
```

- 若后续由用户提供一个**已在 PATH 中**的 Windows amd64 MinGW-w64 GCC，推荐从仓库根目录执行；环境变量只作用于当前 PowerShell 进程：

```powershell
$env:CGO_ENABLED='1'; $env:CC='gcc'; go -C backend test -race ./internal/securityaudit
```

- 以上两条是前置恢复后的精确命令，不是当前可执行候选；当前 WSL 命令会在 `go` 缺失处 exit `127`，Windows 命令会在 `gcc` 缺失处失败。顾虑仍是 reviewer 确认的共享 `req` 数据竞争没有 race detector RED；不得据静态确认猜改，也不得把缺失环境写成测试通过。

## 2026-07-30 thorough review 第 2/2 轮授权 continuation

### gcc 安装恢复前置

- 用户明确授权在 `local-serv-ai` 安装 gcc；继续使用 `ssh-skill`，未重做第 420-597 行本地候选诊断。

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cat /etc/os-release; echo package_managers; command -v dnf || true; command -v yum || true; command -v apt-get || true; echo gcc_status; rpm -q gcc || true; command -v gcc || true; echo package_processes; ps -C dnf,yum,rpm,apt,apt-get,dpkg -o pid=,comm=,args= || true; echo lock_holders; if command -v fuser >/dev/null 2>&1; then fuser /var/run/dnf.pid /var/cache/dnf/metadata_lock.pid /var/lib/rpm/.rpm.lock /var/lib/dpkg/lock /var/lib/dpkg/lock-frontend 2>/dev/null || true; else echo fuser=missing; fi; echo readonly_preflight=ok" --timeout 120
```

- JSON `success=true, exit_code=0`；Rocky Linux `9.7`，`dnf`/`yum` 可用，gcc 未安装；没有并发包管理进程或 lock holder。

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; dnf install -y gcc" --timeout 600
```

- JSON `success=true, exit_code=0`。只请求 gcc；dnf 自动安装 12 个包（gcc、cpp、glibc-devel/headers、kernel-headers、libmpc、libpkgconf、libxcrypt-devel、make、pkgconf 三项）并升级 6 个依赖包（glibc 四项、libgcc、libgomp）。未安装开发工具组，未改 Go/Docker/服务/生产数据，未自动卸载 gcc。

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; gcc --version; rpm -q gcc; CGO_ENABLED=1 CC=gcc go env CGO_ENABLED CC; echo gcc_ready=ok" --timeout 120
```

- JSON `success=true, exit_code=0`；gcc `11.5.0-14.el9.x86_64`，显式环境为 `CGO_ENABLED=1`、`CC=gcc`。

### Linux race RED

- 固定 source `90b0089010c77d0fef1790e2e8cf3b675d4994cd`；stage `race-red`；nonce `714d12f8d8004666ac19f8e95fea6b1e`；remote `/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e`。

```text
git archive --format=tar --output="C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e.tar" 90b0089010c77d0fef1790e2e8cf3b675d4994cd
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; test ! -e /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e; mkdir -m 700 /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e; go version; gcc --version | head -n 1; df -hT / /tmp; echo preflight=ok" --timeout 120
$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_upload.py local-serv-ai "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e.tar" "/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/source.tar" --no-progress
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e; sha256sum source.tar; mkdir src; tar -xf source.tar -C src; rm -f source.tar; rm -rf src/backend/.test-tmp; mkdir -p src/backend/.test-tmp; echo setup=ok" --timeout 300
```

- 全部 exit `0`；archive size `53,770,240`，local/remote SHA-256 `f7121a186eafd3f80e86f52ea02f264bf7b945559d24757ea9df0be87e0743b8`。

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend; CGO_ENABLED=1 CC=gcc TMPDIR='/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend/.test-tmp' TMP='/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend/.test-tmp' TEMP='/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend/.test-tmp' go test -race ./internal/securityaudit -run '^TestCoordinatorCheckLazyEvaluatesBodyOnceAcrossPromptAndLegacy$' -count=1 > '/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/race-red.log' 2>&1" --timeout 900
```

- 首次 `-count=1` JSON `success=true, exit_code=0`，未命中调度 race，不作为 RED；下载日志 size `63`、SHA-256 `a74e7876cb44f4b5478de2b4f2b8fa2e5fcb5be1e3f2fada0dfd06483efe2cbd`。

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend; CGO_ENABLED=1 CC=gcc TMPDIR='/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend/.test-tmp' TMP='/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend/.test-tmp' TEMP='/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend/.test-tmp' go test -race ./internal/securityaudit -run '^TestCoordinatorCheckLazyEvaluatesBodyOnceAcrossPromptAndLegacy$' -count=100 > '/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/race-red-count100.log' 2>&1" --timeout 900
```

- 调度放大后 JSON `success=false, exit_code=1`；下载 RED log size `2,528`、SHA-256 `9ae589906c64c1656a386940e88420dc27ea92e6ccc3b993c594a6797b51a698`。`WARNING: DATA RACE` 精确指向 legacy goroutine `coordinator.go:82` 读与 prompt goroutine `coordinator.go:90` 写，随后 test/package `FAIL`。

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; rm -rf /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e; test ! -e /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e; echo cleanup=ok" --timeout 120
```

- cleanup exit `0`；remote 目录/local archive 均不存在，两份下载日志保留。

### 最小修复、source commit 与 GREEN

- 只修改 `backend/internal/securityaudit/coordinator.go`：两个 goroutine 以值参数各自复制 `Request`；在 `lazyLegacyEngine`/`Coordinator.CheckLazy` 附近约束 provider 仅能在当前同步调用返回前求值、不得保留，Async 先冻结再 clone。未改测试、未新增抽象。
- `go test ./internal/securityaudit -run '^TestCoordinatorCheckLazyEvaluatesBodyOnceAcrossPromptAndLegacy$' -count=1 -v`、`go test ./internal/securityaudit -count=1`、`golangci-lint run ./internal/securityaudit/...` 均 exit `0`；lint `0 issues.`。
- source commit：`417bbcc6a44c35b3e3ed16efb0bb86a4717401c9 fix: isolate blocking audit requests`；只含 `coordinator.go`，未 amend。
- GREEN stage `race-green`；nonce `3fef3c36ca644830b8d76e740b6a3e8e`；remote `/tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e`。

```text
git archive --format=tar --output="C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e.tar" 417bbcc6a44c35b3e3ed16efb0bb86a4717401c9
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; test ! -e /tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e; mkdir -m 700 /tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e; go version; gcc --version | head -n 1; df -hT / /tmp; echo preflight=ok" --timeout 120
$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_upload.py local-serv-ai "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e.tar" "/tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e/source.tar" --no-progress
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e; sha256sum source.tar; mkdir src; tar -xf source.tar -C src; rm -f source.tar; rm -rf src/backend/.test-tmp; mkdir -p src/backend/.test-tmp; echo setup=ok" --timeout 300
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e/src/backend; CGO_ENABLED=1 CC=gcc TMPDIR='/tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e/src/backend/.test-tmp' TMP='/tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e/src/backend/.test-tmp' TEMP='/tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e/src/backend/.test-tmp' go test -race ./internal/securityaudit -run '^TestCoordinatorCheckLazyEvaluatesBodyOnceAcrossPromptAndLegacy$' -count=10 > '/tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e/race-green.log' 2>&1" --timeout 900
```

- 全部 exit `0`；archive size `53,780,480`、hash `3fc7a336e443034f458356131f732b17fe687f6dc7bc2b5e0201250655996f57`。下载 GREEN log size `63`、hash `7a84320668e3fab9a53ddf4d3d5b74ac1caf4b7813aff5512b4f775383a9b50f`，package `ok`、`DATA RACE` count `0`。

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; rm -rf /tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e; test ! -e /tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e; echo cleanup=ok" --timeout 120
```

- cleanup exit `0`；remote 目录/local archive 不存在，GREEN log 保留。

### 新 source 本地 full gate

- 原样 `make test` 在 `417bbcc6a` 上只执行一次，exit `0`，未重跑且无需 EOF/1013 例外。default/unit backend、全量 lint、frontend lint/typecheck 全部通过；Vitest `215/215 files`、`1626/1626 tests`、`73.51s`。完整输出：`C:/Users/caiqy/.local/share/opencode/tool-output/tool_fb15b8185001iS8b5UJLS60oFN`。
- `make "VERSION=0.1.165.1" "SHELL=D:/scoop/shims/bash.exe" build` exit `0`；Vite `1019` modules；VERSION/二进制均为 `0.1.165.1`，binary metadata `vcs.revision=417bbcc6a44c35b3e3ed16efb0bb86a4717401c9`。

```text
git worktree add --detach "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-task27-generate-417bbcc6a" 417bbcc6a44c35b3e3ed16efb0bb86a4717401c9
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git status --short
git worktree remove "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-task27-generate-417bbcc6a"
```

- 全部 exit `0`；两轮 Ent/Wire zero diff，worktree clean 且移除后不存在。
- static：`git diff --check` exit `0`（仅三个保留文件 LF/CRLF warning）；unmerged diff/index、tracked conflict marker、legacy first-token count 均 `0`；VERSION `0.1.165.1`；protected migrations `14/14`、missing `0`。用户 `opencode.json` diff 原样保留未暂存。

### 新 nonce remote integration

- stage `final-verify`；nonce `936ea72e3c4140ca930f88d07e0f34a5`；remote `/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5`。

```text
git archive --format=tar --output="C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5.tar" 417bbcc6a44c35b3e3ed16efb0bb86a4717401c9
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; test ! -e /tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5; mkdir -m 700 /tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5; go version; gcc --version | head -n 1; docker version; docker info >/dev/null; df -hT / /tmp /var/tmp; df -i / /tmp /var/tmp; echo preflight=ok" --timeout 120
$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_upload.py local-serv-ai "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5.tar" "/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/source.tar" --no-progress
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5; sha256sum source.tar; mkdir src; tar -xf source.tar -C src; rm -f source.tar; rm -rf src/backend/.test-tmp; mkdir -p src/backend/.test-tmp; test -d src/backend/.test-tmp; echo setup=ok" --timeout 300
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/src/backend; CI=true GOFLAGS='-v' TMPDIR='/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/src/backend/.test-tmp' TMP='/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/src/backend/.test-tmp' TEMP='/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/src/backend/.test-tmp' go test -tags=integration ./... > '/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/integration.log' 2>&1" --timeout 1800
$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_download.py local-serv-ai "/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/integration.log" "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5-integration.log" --no-progress
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; df -hT / /tmp /var/tmp; df -i / /tmp /var/tmp; rm -rf /tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5; test ! -e /tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5; echo cleanup=ok" --timeout 120
```

- 全部 exit `0`；archive size `53,780,480`、local/remote hash `3fc7a336e443034f458356131f732b17fe687f6dc7bc2b5e0201250655996f57`。preflight Go `1.26.5`、gcc `11.5.0`、Docker client/server `29.2.1`、起始可用 `12G`/inode `2%`。
- integration 未加 `-p`、未重跑；log size `4,295,249`、SHA-256 `28458502f274803aadd1e427fb7dffbd42eabf4dd61317b4e4bcb0f6cc97f3e9`；test/package FAIL `0`。
- migration `2/2` PASS：新库 `4.50s`、local v0.1.159.6 升级 `4.21s`；源码两个 12/12 `require.Len` 仍存在。13 skips（12 top-level + 1 nested）分类与上一轮一致且均不命中 required targets。
- cleanup exit `0`；测试后可用 `11G`、使用率 `70%`、inode `2%`；remote 目录/local tar 不存在，integration log 保留。未 raw SSH/SCP、Docker prune、构建 Sub2API 镜像、部署或访问服务/生产数据。

### 最终状态、顾虑与风险信号

- 状态：`DONE_WITH_CONCERNS`。source/race/local/remote 所有要求均有真实通过证据；formal report commit 为 `763db6ad47f2c39dad46af70c8da9ce72abb45fd docs: record blocking audit race verification`，仅含 formal report。
- 顾虑：gcc 授权安装触发 dnf 自动升级 6 个 glibc/libgcc/libgomp 包，按授权不自动卸载；远端可用空间降至 `11G`。
- 风险信号：race RED 的首次 `-count=1` 未命中、`-count=100` 才复现，说明该调度竞争具有概率性；历史 lease-loss EOF/1013 与 HeapAlloc 波动仍是既有非阻断风险。
