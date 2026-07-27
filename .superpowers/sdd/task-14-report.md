# Task 14 实施报告：合入并记账 v0.1.162

## 状态与对象

- 状态：`DONE_WITH_CONCERNS`。TDD N/A：本任务限定为既有上游 tag 的受审 merge、冲突语义融合与静态检查；未改独立产品行为，不伪造 RED。
- 起点：`940c5cfcf390ecbfd2e041fb2b46c99846e6ea3e`。
- tag：`git rev-parse "v0.1.162^{}"` 为 `27f094e0960ebd8e52de7ff7e763c6fec2ff4057`。
- merge：`8bda73544d6e26a323f101e5c68981634f0375ab merge: upstream v0.1.162`；父为 `940c5cfcf390ecbfd2e041fb2b46c99846e6ea3e 27f094e0960ebd8e52de7ff7e763c6fec2ff4057`，只含上游合入树和冲突决议。
- ledger：`bcfe364020d05df209ae95bf86cb1a41b4ddd7f9 docs: record v0.1.162 merge`，只含 `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`。

## 原始冲突与决议

- 原始 unmerged 清单（执行任何决议前）：`backend/cmd/server/VERSION`、`backend/internal/config/config.go`、`backend/internal/config/config_test.go`、`backend/internal/handler/admin/setting_handler.go`、`backend/internal/handler/admin/setting_handler_update.go`、`backend/internal/handler/openai_gateway_handler_test.go`、`backend/internal/service/openai_gateway_messages.go`、`backend/internal/service/openai_ws_http_bridge.go`、`backend/internal/service/setting_service.go`、`backend/internal/service/setting_update.go`、`backend/internal/service/wire.go`、`frontend/src/components/account/EditAccountModal.vue`、`frontend/src/i18n/locales/en/index.ts`、`frontend/src/i18n/locales/zh/index.ts`。
- `VERSION`：保持本地中间版本 `0.1.159.6`。
- 配置/setting：保留本地 step-up 转换门控、settings JSON backfill、sticky/scheduler runtime 热更新和审计入口；加入上游 trusted proxy、forwarded client-IP header 优先级与 JSON 回填。安全路径仍由 Gin `SetTrustedProxies` 解析，未授权外部 header 不作为 ACL/风控来源。
- Grok HTTP/WS：保留 platform sticky、session hash、usage/ops、spool/redaction、created-only WS ownership；加入客户端 tool cache、Grok encrypted-content 单次剥离重试及上游 bridge failover 分支。HTTP bridge 按本地 `handleFailoverErrorResponsePassthrough` 契约融合，并保留上游 error frame helper。
- S3/image：Wire 同时保留本地 `ProvideOpenAIGatewayServiceWithStartupRecovery`，接入上游 `ImageStorageSettingService`、runtime resolver 和 S3/backup 回落；异步图片任务、image capability、moderation/audit 与权限 handler 均保留。
- 前端：Grok OAuth cache 开关与 header override 同时存在；locale index 同时导出 images 与 batch-image。

## 高风险无冲突审查

- 客户端 IP 调用链：`config -> SettingService.LoadForwardedClientIPSettings -> middleware -> pkg/ip -> server/http`。可信代理未配置时禁用 forwarded chain；自定义 header 仅在显式 trust 开关下进入兼容路径。
- Grok 调用链：`openai_gateway_grok_cache -> session_binding -> openai_gateway_messages/openai_ws_http_bridge`。保留 Grok/platform sticky、owner binding、body spool/redaction、usage/ops 与 created-only session ownership。
- 存储调用链：`BackupService/ImageStorageSettingService -> ImageTaskService -> admin backup/image handler`。保留对象存储加密、异步图片任务、capability/moderation/audit 和后台权限边界。
- v0.1.161 闭合能力：step-up、API Key helper、billing probe UI、Ops snapshot、YAML 128 MiB、advanced/layered scheduler/fallback/DB recheck、PromptAdminService Wire bind 均仍在合入树；不恢复 `openai-first-token-timeout`。
- migration：`172_video_per_second_billing_metadata.sql`、`181_group_duplicate_operation_id.sql`、`181_prompt_audit.sql`、182、183、184 与 staged migration test/后续快照逻辑均保留。

## 静态与交接

- 通过：`git ls-files -u` 为空；真实 marker 扫描无结果；`git diff --cached --check` 通过；`go test -run '^$' ./cmd/server ./internal/config ./internal/handler/admin ./internal/handler ./internal/service ./internal/server ./internal/server/middleware ./internal/server/routes` 通过；`pnpm exec vue-tsc --noEmit` 通过。
- 未运行：Task 15 的 full test、build、generate、integration 与回归门禁。
- Task 15 待验证：可信代理/header fail-closed、Grok cache/sticky/owner/spool/redaction/usage/WS created-only、S3/backup/image task/moderation/audit、step-up/API Key/billing probe/Ops/YAML、scheduler/fallback/recheck、migration staged upgrade 和前端账户/设置流。
- 风险：本轮仅做静态与限定编译；运行时行为尚未由 Task 15 覆盖。Windows Git `sh`/历史 generate 文件锁和既有 Browserslist/Vite advisory 仍是基线顾虑。
- 顾虑：v0.1.162 的 forwarded-IP、Grok retry/cache 与 runtime image storage 跨安全/账号/存储边界，必须以 Task 15 回归结果决定后续放行。

## Sol reviewer 第 1 轮 cleanup

- TDD N/A：本轮只删除冲突遗留的注释化死副本并恢复安全理由注释，不改变可执行行为，不伪造 RED。
- reviewer finding：Important 为两个 admin setting handler 内遗留的三段 `/* ... */` 重复字段/结构体块；Minor 为 step-up 转换门控缺少自锁与强制验证的安全理由说明。
- 源码提交：`46eb292c04f14630dcfb31b28f6e83f541029d93 chore: clean v0.1.162 conflict remnants`，仅含 `setting_handler.go` 与 `setting_handler_update.go`；删除三段死副本，保留实际 JSON/setting 映射、trusted proxy、scheduler/sticky、audit 与 step-up 行为。
- 注释：启用前校验当前管理员会话/TOTP 以防自锁；关闭时继续由 `EnforceStepUpAlways` 强制 step-up，以防被劫持会话降级防护。
- 通过：`gofmt -w backend/internal/handler/admin/setting_handler.go backend/internal/handler/admin/setting_handler_update.go`；`go -C backend test ./internal/handler/admin -run '^TestUpdateSettings(EnableStepUp|DisableStepUp)' -count=1`；`go -C backend test ./internal/handler/admin -run '^$'`；源码 diff 仅含注释删除/新增及 gofmt 空白对齐。
- 未运行：Task 15 full test、build、generate、integration 与回归门禁；无需远程操作。
