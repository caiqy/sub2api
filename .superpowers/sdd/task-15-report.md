# Task 15：v0.1.162 回归修复与 full 门禁

## 范围与结论

- 起点为 `98fa814d2`；保留 Task 14 cleanup `46eb292c04f14630dcfb31b28f6e83f541029d93` 和所有既有未提交协调文件。
- `backend/cmd/server/VERSION` 保持 `0.1.159.6`。
- 结论：`PASS`。v0.1.162 的 proxy/settings/Grok/sticky/storage 聚焦回归、完整本地门禁、生成稳定性和远程 Testcontainers integration 均通过。

## RED 与最小修复

- `57b5bf758052fa5753ce1f9688f2603c9cad34bd`：HTTP failover 在后续转发前回填已读错误 body；Grok client-tool cache 写入最终提交 payload；更新 v0.1.162 failover close-code 测试。
- `1d191894a319cc99f7a80dc3f2b6ce8b409816b9`：后续 WebSocket turn transport error 写入协议 error event，不重放已完成 turn。
- `84d9a4f4f33250bd814fe033177cdfecbbcaf772`：注册 `minimum_balance_reserve`、`upstream_cost`、`prefer_soonest_reset` 的 env 可达默认值，并使 forwarded-IP 设置 fixture 覆盖启动加载路径。
- `69ac6209fcb4928251142a3368ffdbd750919076`：对齐前端 rollback 请求的 900000 ms timeout 测试契约。

## 本地门禁

- 聚焦矩阵：trusted proxy/client IP、settings、Grok cache/sticky/WebSocket、storage/image、step-up/API Key/billing/Ops/scheduler 和 migrations 均 GREEN。
- `make test`：退出 `0`；Vitest `204` 个文件、`1549` 个用例通过。
- `make build`：退出 `0`；后端和前端生产构建完成。
- 连续两轮 `make -C backend generate`：均退出 `0`；每轮后的 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 均为零差异。
- 静态检查：`git ls-files -u`、`git diff --name-only --diff-filter=U`、精确 conflict marker 扫描和 `git diff --check` 均通过。迁移集合含 `181_prompt_audit.sql`、`181_group_duplicate_operation_id.sql`、`182_prompt_audit_full_prompt.sql`、`183_ops_ingress_reject_aggregates.sql`、`184_auth_cache_invalidation_outbox.sql`。
- advisory：Browserslist 数据时效、Vite dynamic/static import 与大 chunk 提示、Vitest `router-link` warning；均不影响退出码。

## 远程 Integration

- archive HEAD：`69ac6209fcb4928251142a3368ffdbd750919076`。
- 目标：`local-serv-ai`；唯一远程目录：`/tmp/sub2api-task15-b369bbfa849c4b8fadad749d4c66f2b9`。
- 预检：Go `1.26.5`、Docker Server `29.2.1`。
- 命令：`CI=true GOFLAGS='-v' TMPDIR='<remote>/src/backend/.test-tmp' TMP='<remote>/src/backend/.test-tmp' TEMP='<remote>/src/backend/.test-tmp' go test -tags=integration ./...`，退出 `0`。
- 目标 `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` PASS（`4.80s`），`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS（`4.58s`）。日志保存于 `C:/Users/caiqy/AppData/Local/Temp/sub2api-task15-b369bbfa849c4b8fadad749d4c66f2b9-integration.log`；本地 archive 与远程目录已删除。
- 日志中保留环境/显式配置型 skip：DingTalk disabled、TLS capture/JA3 profile、并发 cache TODO、prompt-audit/Redis 未配置、OpenAI API comparison；没有 `FAIL`。
