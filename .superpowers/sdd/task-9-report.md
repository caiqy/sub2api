# Task 9 实施报告：v0.1.160 回归保护与 full 门禁

## 状态

`DONE_WITH_CONCERNS`

任务起点为 `d130c67544b0aaf7c819abd8a5a2cc683ddda6bc`。未 push、tag、release 或 deploy。

## 提交与变更

- `3de7191e3 test: cover staged migration upgrades`
  - `backend/internal/repository/migrations_schema_integration_test.go`
- `a719a8e6c fix: preserve local behavior after v0.1.160`
  - `backend/internal/handler/content_moderation_helper.go`
  - `backend/internal/handler/openai_gateway_handler.go`
- `31b132689 fix: preserve local behavior after v0.1.160`
  - `backend/internal/securityaudit/prompt_module.go`

最终提交为 `31b132689f858f4b6e657c653829c80d2fc09738`。未触碰 plan、OpenSpec `tasks.md`、`.comet/subagent-progress.md`、`.comet/current-change.json` 或 `paseo.json`。

## Migration 保护

新增 `TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages`：固定覆盖本地 `172_video_per_second_billing_metadata.sql`、本地 `181_group_duplicate_operation_id.sql` 和上游 181-190 的 11 个完整 filename（含 `190_add_users_email_alias_dedup_index_notx.sql`）。测试在当前阶段严格识别上游仅出现 `181_prompt_audit.sql` 和 `182_prompt_audit_full_prompt.sql`；先以排除当前上游文件的 `fstest.MapFS` 应用本地集，再应用完整 embedded FS，并检查完整 filename、checksum 与记录数二次应用稳定。

首次本机命令 `go -C backend test -tags=integration ./internal/repository -run '^TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages$' -count=1 -v` 退出 `0`，但 Docker 不可用，integration harness 整体 skip，未记为 GREEN。依简报转用远程 Docker gate：首次 archive (`3de7191e3`) 的完整 integration 虽因 handler 编译 RED 整体退出 `1`，目标测试已明确 `PASS (4.32s)`，证明测试本身首次执行为既有 filename runner 行为保护，未伪造 RED。最终 archive 的同一目标测试明确 `PASS (4.76s)`，无对应 SKIP。

## 实际 RED 与最小修复

1. 远程 full integration 首次 RED：`OpenAIGatewayHandler.checkContentModeration` 与 `closeOpenAIWSFailoverExhausted` 缺失，且 v0.1.160 合入的 `handleWSFailover` closure 从未在当前本地 WS 实现调用，还引用不存在的 `releaseAccountSlot`。`go -C backend test -tags=unit ./internal/handler -run '^(TestOpenAIGatewayHandler|TestOpenAIGatewayHandlerImages|TestContentModeration)' -count=1 -v` 真实 RED。恢复 OpenAI handler 的已有 content moderation 委派，并删除无调用的错误 closure。既有 `TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease` 转 GREEN；全 handler unit package 转 GREEN。
2. 生成 RED：`make -C backend generate` 退出 `2`，Wire 报 `no provider found for ... PromptAdminService`。根因是 v0.1.160 新增 `PromptAdminHandler`，但 `securityaudit.ProviderSet` 仅绑定 `PromptEngine`，未将同一 `*PromptService` 绑定为 `PromptAdminService`。添加一个 Wire interface bind；两轮 generate 均转 GREEN，Ent/Wire diff 为空。

## GREEN 摘要

- scheduler：`TestLayered_PriorityDeterminism` PASS。
- Sticky：`TestLayered_SessionStickyPreservesGrokBinding`、`TestGatewayService_SelectAccountForModelWithPlatform_StickyDisabledBypassesStickyReadAndWrite` PASS。
- privacy：`TestLayered_PreviousResponseStickyHonorsRequirePrivacySet` PASS。
- image capability：`TestLayered_RequiredImageCapabilityFiltersUnsupportedAccounts` PASS。
- audit：handler 的 `TestSecurityAuditBlockingFailuresLeaveAllDownstreamCountersAtZero`（3 子例）与 `TestPromptAuditGatePrecedesAccountBillingAndUpstreamSideEffects`（13 gateway 路径）PASS；routes 的两项 Prompt Audit 覆盖/权限测试 PASS。
- 最终 `make test` 退出 `0`：后端默认、lint、unit 与前端 lint/typecheck 通过；Vitest `199` files、`1521` tests PASS。
- 最终 `make build` 退出 `0`：backend 版本 `0.1.159.6`，Vite `1005 modules transformed`。
- 最终连续两轮 `make -C backend generate` 及每轮 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 均退出 `0`、无 diff。未复现 Windows `user-mapped section` 锁，未启用 detached worktree 备用流程。
- `git diff --check`、`git diff --cached --check`、未合并路径检查、精确 conflict marker 扫描均退出 `0`；`backend/cmd/server/VERSION` 为 `0.1.159.6`。

## 远程 integration 与清理

全部远程操作只使用 `ssh-skill` 的 `ssh_execute.py`、`ssh_upload.py`、`ssh_download.py`，目标 `local-serv-ai`。未构建 Sub2API 镜像、未部署、未访问生产数据。

- 最终 archive HEAD：`31b132689`；本地 tar `C:/Users/caiqy/AppData/Local/Temp/sub2api-task-9-d70a45257d0f46c0911f6e17747ab6aa.tar`，`50,851,840` bytes，SHA-256 `0A6133485F5747EFC1DE77F349D3F16825CF0C0E9572BFC95580B85088A61235`。
- 远程唯一目录：`/tmp/sub2api-task-9-d70a45257d0f46c0911f6e17747ab6aa`；创建、上传、预检、integration、下载和清理 JSON 均为 `success=true, exit_code=0`。
- 预检：`go version go1.26.5 linux/amd64`，Docker `ServerVersion=29.2.1`。
- 在 archive 的 `backend` 重建 `.test-tmp`，以同一绝对路径设置 `TMPDIR`、`TMP`、`TEMP` 后运行 `CI=true GOFLAGS='-v' go test -tags=integration ./...`，退出 `0`。日志保留于 `C:/Users/caiqy/AppData/Local/Temp/sub2api-task-9-d70a45257d0f46c0911f6e17747ab6aa-integration.log`。
- 清理：`rm -rf` 后 `test ! -e` 退出 `0`；本地 tar 已删除并确认不存在。首轮失败 archive 的远程目录 `/tmp/sub2api-task-9-861b4dabecc04906981793fe89fb05a4` 与 tar 同样已清理。

## 风险信号与顾虑

- migration：当前 live PostgreSQL 已验证本地 172/181 与上游 181/182 的 staged upgrade、完整 filename、checksum 和记录数幂等；未来 183-190（含 `_notx`）由固定集合和完整 FS 路径自动纳入。完整上游 183-190 尚未合入当前 HEAD，不能声称已在 live DB 执行。
- 跨模块 full gate：真实合入回归包括 handler 的本地 moderation 委派、WS 死 closure 和 Prompt Audit 的 Wire DI；均有定向 RED/GREEN、最终本地 full gate 与远程 full integration 证据。
- 远程 integration 有既有环境型 skip：DingTalk disabled、TLS capture/JA3 外部条件、若干 Prompt Audit Redis/PostgreSQL 条件测试、未配置 OpenAI key；新增 migration 测试未 skip。
- `TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease` 通过时输出 worker panic recovery 日志；未造成断言失败，但应单独调查其测试/worker 生命周期。
- Browserslist 数据过期、Vite 动态 import/chunk-size advisory（最大 AccountsView `670.83 kB`）仍存在；均未阻断 build。
- 历史 Windows `user-mapped section` 风险本次未重现；因此无需短路径 detached worktree，但根因仍未有新证据。

## Review Fix Round 1

### 提交边界

- 受门禁代码/archive HEAD 是 `31b132689f858f4b6e657c653829c80d2fc09738`；原 docs ledger HEAD 是其后的 docs-only `516e998b07d8e81a6d1e18f7acc0c0425d6b083f`，不是 `31b132689`。
- 本轮新增 `fd909c5be test: harden staged migration coverage`，只修改 `backend/internal/repository/migrations_schema_integration_test.go` 与 `backend/internal/handler/openai_images_controls_test.go`。未 amend 或重写此前四个提交，未 push、tag、release 或 deploy。

### Reviewer Findings 修复

1. Critical 1：`upstreamMigrations` 补齐 `172_composite_model_routes.sql`，固定集合为 12 个完整上游 filename；本地 `172_video_per_second_billing_metadata.sql` 和 `181_group_duplicate_operation_id.sql` 继续独立处理。
2. Critical 2：删除永久精确等于 181/182 的全局约束，改为严格匹配累计 release 快照：`v0.1.160`=181/182、`v0.1.161-v0.1.162`=再加 183/184、`v0.1.163`=再加 185、`v0.1.164`=再加上游 172 和两个 186、`v0.1.165`=再加 187-190。当前 HEAD 仍要求恰为 v0.1.160；任意缺失、中间子集或额外组合均失败。完整 embedded FS 第二次应用继续核验 filename、checksum 和记录数；当 190 出现时，它会作为当前快照成员由同一 `applyMigrationsFS` 调用真实走 `_notx` 分支。
3. Minor：删除重复的 `requireMigrationAppliedInDB`，升级路径改复用既有 `requireMigrationApplied`。
4. Important 1：worker panic 可稳定复现。panic 值为 `runtime error: invalid memory address or nil pointer dereference`；临时 stack 证据定位到测试 mock 的 nil 嵌入接口调用：`openAIImagesModerationRepo.CountFlaggedByUserSince` -> `applyFlaggedAccountSideEffects` -> `persistContentModerationLog` -> `worker`。请求在 keyword pre-block 后同步返回并异步 `enqueueRecord`；旧测试未等待 worker，且 mock 只实现 `CreateLog`。本轮 mock 补齐计数方法，并以 `CreateLog` channel 等待异步 record drain；不改生产代码。该服务生产仓库实现完整接口，故此为测试 fixture/lifecycle 观察缺口，不是生产 worker 回归。

### RED / GREEN

- migration RED（local-serv-ai，Docker 已连接、无 SKIP）：从 `fd909c5be` archive 临时删除固定集合中的上游 172 后运行 `CI=true TMPDIR=/tmp/sub2api-task-9-red-fd909c5be/backend/.test-tmp TMP=/tmp/sub2api-task-9-red-fd909c5be/backend/.test-tmp TEMP=/tmp/sub2api-task-9-red-fd909c5be/backend/.test-tmp go test -tags=integration ./internal/repository -run TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages -count=1 -v`，退出 1，失败摘要：`should have 12 item(s), but has 11`。
- migration GREEN（已提交 `fd909c5be` archive）：同一测试命令在 `/tmp/sub2api-task-9-review-fd909c5be` 退出 0，目标 `TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS（4.31s），无 SKIP。
- worker 调查 RED：`go -C backend test -tags=unit ./internal/handler -run '^TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease$' -count=1 -v` 在加入 drain assertion后退出 1，等待异步 record 超时并输出同一 panic。栈通过临时 recovery instrumentation 获取后已还原，未进入提交。
- worker GREEN：同一命令本机 `-count=3` PASS；已提交 archive 的 `go test -tags=unit ./internal/handler -run TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease -count=3 -v` 亦 PASS 三次，输出无 `content_moderation.worker_panic`。
- 本机 migration 命令因 Docker 不可用而 skip，仅作编译检查，未记为 GREEN。

### 远端与清理

- 全部远端操作仅用 `ssh-skill` 的 Python 脚本，目标 `local-serv-ai`；只运行 Testcontainers 测试，未构建 Sub2API 镜像、未部署、未访问生产数据。
- GREEN archive：`fd909c5be`，`C:/Users/caiqy/AppData/Local/Temp/sub2api-task-9-review-fd909c5be.tar`，`50,862,080` bytes，SHA-256 `9B0DEC5D21D3413C00577DB6A8BDDC1198D4EE745E8AE229B26483980C0202FF`；远端目录 `/tmp/sub2api-task-9-review-fd909c5be` 已 `rm -rf` 后确认不存在。
- RED replay archive 和目录 `/tmp/sub2api-task-9-red-fd909c5be` 同样已删除；两个本地 tar 均确认不存在。

### 剩余顾虑

- 当前 v0.1.160 未包含 183-190，故无法声称 190 已在当前 embedded FS 的 live DB 执行；测试已将其限定为 v0.1.165 完整快照成员，届时会实际执行 `_notx` runner 路径。
- 远端 Testcontainers 出现既有 Ryuk handshake advisory，但目标 migration 测试完整执行并 PASS，未跳过。
