# Task 12 报告：v0.1.161 回归修复与 full 门禁

## 结论

- 状态：`DONE_WITH_CONCERNS`。Task 12 的源码、测试、本地 full gate、生成稳定性、静态/migration 核验和 `local-serv-ai` integration 均已通过；环境型 skip 与 Windows 工具链提示已分类。
- Task 13 在 Sol review 前保持封闭；本任务未生成 review package，未修改 OpenSpec task/progress、selector、plan 或其他协调文件，也未 push、tag、release、deploy 或合并 `main`。

## 提交与真实时间线

| 提交 | 角色与结论 |
| --- | --- |
| `0775a6063 fix: preserve local behavior after v0.1.161` | 初始兼容修复集合；其中 WS V2 将空 active turn 的认领放宽至 terminal，违反 created-only ownership，不能作为该契约的最终修复。其余 fixture/测试桩对上游接口和输出的适配经聚焦包通过。 |
| `f1cde1b52 test: cover terminal WS turn binding` | 无效的反向测试适配：重命名保护测试并断言 terminal 可直接绑定空 turn。它不是 GREEN 证据，后续已恢复原测试名和语义。 |
| `c3bfb765f fix: restore created-only WS turn ownership` | follow-up 源码/测试修复：空 active turn 仅允许首个同 ID `response.created` 绑定；带 ID 的外来 delta/terminal 被忽略，不计 usage、不完成 turn、不释放 permit。 |
| `47a6c031e test: align websocket lifecycle fixtures` | follow-up fixture 修复：为正常 terminal/delta 生命周期在同 ID 后续事件前显式发送并消费 `response.created`，不放宽 ownership。 |
| `07029cc45 fix: remove stale Grok session hash assignments` | follow-up 源码修复：删除 `grok_media.go` 中被统一 session hash 计算覆盖的两项死赋值，消除 full gate 的 `ineffassign` RED。 |

### 真实 RED/GREEN

- created-only RED：先恢复 `TestObserveUpstreamMessage_BindsOnlyResponseCreated`，在 `0775a6063` 的 terminal 放宽行为上执行 `go test ./internal/service/openai_ws_v2 -run '^TestObserveUpstreamMessage_BindsOnlyResponseCreated$' -count=1`，失败于外来 `response.completed` 被标记 terminal。该 RED 证明外来 terminal 会错误认领、计费并完成 turn。
- created-only GREEN：`c3bfb765f` 删除 terminal 放宽后，原保护测试及 `go test ./internal/service/openai_ws_v2 -count=1` 均通过。
- lifecycle fixture RED/GREEN：严格 ownership 后，`TestPassthroughLifecycle_TerminalSwitchesToInterTurnIdleTimeout` 与 `TestPassthroughLifecycle_SecondTurnTimeoutIsNotFailoverSafe` 暴露 terminal-first fixture；`47a6c031e` 只补同 ID `response.created`，完整 `go test ./internal/service -run '^TestPassthroughLifecycle_' -count=1` 通过。
- Grok lint RED/GREEN：首次 `make test` 的 `golangci-lint` 报 `grok_media.go:178,196` 两项 `ineffassign`；追踪确认两次赋值均被第 231/235 行覆盖。`07029cc45` 删除死赋值后，`golangci-lint run ./internal/handler` 与 `go test ./internal/handler -run '^TestGrok' -count=1` 通过。
- 未观察到其他可归因于 v0.1.161 的真实生产 RED；没有把 `f1cde1b52` 的反向断言或环境/fixture 事件写成生产修复证据。

## 聚焦矩阵

- 模型冷却、advanced/layered scheduler、fallback/WaitPlan、DB fresh recheck、session/sticky、Grok URL/owner/spooling/redaction、content moderation：`go test ./internal/service -count=1` 通过（98 秒）。
- step-up、HTTP failover、Grok 视频和 Ops snapshot：`go test ./internal/handler ./internal/handler/admin ./internal/server/middleware -count=1` 通过。
- migration 181-184、runner checksum/幂等、repository：`go test ./migrations ./internal/repository -count=1` 及 migration 定向命令通过。
- YAML、PromptAdminService Wire bind：`go test ./internal/config ./internal/securityaudit ./cmd/server -count=1` 通过。
- API Key helper、billing probe UI、content moderation UI：`pnpm exec vitest run src/components/account/__tests__/CreateAccountModal.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/RiskControlView.spec.ts --reporter=dot` 为 3 文件、45 用例通过。

## 本地 full gate 与静态核验

- 最终 `make test` 通过：后端默认/unit/lint 与前端 lint/typecheck/Vitest 均完成；前端为 201 文件、1537 用例通过。首次同命令的 Grok lint RED 已由 `07029cc45` 闭合。
- `make build` 在未注入 Git `sh` 的 Windows 进程中首次因 `resolve-version.sh` 启动失败而退出；仅对该进程补 `C:\Program Files\Git\bin` 和 `usr\bin` 后重跑通过，后端按 `0.1.159.6` 构建，前端 Vite 处理 1006 模块。
- 连续两轮 `make -C backend generate` 均通过；每轮后的 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 均无 diff，未复现 Windows 文件锁。
- `git diff --check` 通过；`git ls-files -u` 为空；精确 conflict marker 扫描无残留。`backend/cmd/server/VERSION` 为 `0.1.159.6`。旧本地 `openai-first-token-timeout` 配置/错误/watchdog 在业务根目录无残留；保留的是上游原生 `openai_first_output_timeout`。
- migration 文件集合包含本地 `172_video_per_second_billing_metadata.sql`、本地 `181_group_duplicate_operation_id.sql` 及上游 `181_prompt_audit.sql`、`182_prompt_audit_full_prompt.sql`、`183_ops_ingress_reject_aggregates.sql`、`184_auth_cache_invalidation_outbox.sql`；本地 migration/repository 聚焦测试通过。

## Remote Integration

- archive HEAD：`07029cc45cdfc8530f81351b38516401d202c837`，通过 `git archive` 上传至 `local-serv-ai` 的唯一目录 `/tmp/sub2api-task12-07029cc45`；未构建 Sub2API 镜像、未部署、未访问生产数据。
- 预检：Go `1.26.5`、Docker Server `29.2.1`。重建 `backend/.test-tmp` 后执行 `CI=true GOFLAGS='-v' TMPDIR=... TMP=... TEMP=... go test -tags=integration ./...`，退出 `0`。
- 日志 `C:\Users\caiqy\AppData\Local\Temp\sub2api-task12-07029cc45-integration.log` 无 `FAIL` 或 panic，所有包以 `ok` 或无测试包收束。`TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` PASS（5.20 秒），`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS（4.75 秒），验证当前上游 181-184 与本地 172/181 按 filename 升级、幂等和 checksum 稳定。
- 16 个 `--- SKIP:` 均为阶段 0/v0.1.160 已接受的环境基线：DingTalk disabled sentinel、TLS capture/JA3/两个 profile 的外部条件、已知 CI concurrency TODO、Prompt Audit Redis/PostgreSQL 未配置、无 OpenAI key；未命中 v0.1.161 受影响能力，不阻塞本阶段。
- 远端临时目录和本地 archive 已删除；下载日志保留为证据。

## 残余风险

- Windows 环境仍依赖可用的 Git `sh` 才能运行 `make build`；本次使用仅进程级 PATH 注入通过。
- Windows `user-mapped section` 生成文件锁未复现但历史根因未定；本轮双 generate 已稳定。
- 前端存在既有 Browserslist、动态 import/chunk-size、router-link/jsdom 负路径及 i18n stderr advisory；full gate 退出码为 0。
