# Task 11 实施报告：v0.1.161 合入与记账

## 状态与对象

- 状态：`DONE_WITH_CONCERNS`。
- 起点：`3fc60752acc459ecc37cd50b40df4a1f84ce3b62`（`docs: complete v0.1.160 stage review`）。
- tag：`v0.1.161^{}` = `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c`。
- merge：`f2158292c7ff3de4caa7ec22f9b7148400948f08`（`merge: upstream v0.1.161`），父提交为第一父 `3fc60752acc459ecc37cd50b40df4a1f84ce3b62` 和第二父/tag `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c`。
- ledger：`4c37dfccb2d6dd38fd4b727c00b0bf332d79b7e5`（`docs: record v0.1.161 merge`），唯一父是 merge；该独立文档提交只修改 `docs/superpowers/reports/2026-07-26-staged-merge-upstream-v0-1-165-build.md`。
- `backend/cmd/server/VERSION` 保持 `0.1.159.6`。已审批移除的 `openai-first-token-timeout` 未恢复，后端、前端和 deploy 的旧符号静态扫描无匹配。

merge commit 是合入后的代码树，不是 ledger 或协调工件提交；`plan`、OpenSpec、progress、`.comet` selector 和 `paseo.json` 均未进入 merge 或 ledger 提交。

## 原始 Unmerged 文件及决议

merge 前 `git diff --name-only --diff-filter=U` 的原始文本冲突集合为以下 26 个文件；`git show --remerge-diff f2158292c` 重建结果与之相符。每项均为冲突融合，而非选择单侧版本。

- `backend/cmd/server/VERSION`：保留本地中间版本 `0.1.159.6`，不降回 tag 的版本号。
- `backend/cmd/server/wire_gen.go`：合并生成图的上游 auth-cache/outbox 依赖，同时保持本地既有服务图；Task 9 的 `PromptAdminService` bind 仍由 `securityaudit.ProviderSet` 提供。
- `backend/internal/config/config_test.go`：保留本地配置/校验覆盖，并纳入上游配置默认值与环境读取测试准备。
- `backend/internal/handler/admin/setting_handler.go`：同时保留本地 gateway runtime/session 设置读取与上游 `step_up_enabled`、审计保留、管理员充值返利字段。
- `backend/internal/handler/admin/setting_handler_update.go`：同样并存本地 scheduler/runtime 热更新字段和上游 step-up 设置更新字段。
- `backend/internal/handler/grok_media.go`：保留 multipart spooling、快照脱敏、usage 和 sticky 约束，吸收 generation capability、视频状态/content 代理入口。
- `backend/internal/handler/grok_media_test.go`：保留本地 spool/脱敏行为断言，并合并上游媒体 capability/视频覆盖。
- `backend/internal/handler/openai_gateway_handler_test.go`：保留本地 Responses/WS failover、usage 语义测试，吸收上游所需并发测试支撑；未把旧 first-token 测试带回。
- `backend/internal/repository/migrations_schema_integration_test.go`：保留语义空字符串默认值与 Task 9 的 `TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages`，并吸收上游并发 runner session-lock 覆盖。
- `backend/internal/server/routes/gateway.go`：Responses/images/videos 保留 `usageDetailCapture` 和 Grok Responses WebSocket 拒绝；同时采用 `textBodyLimit`、视频 content 路由。
- `backend/internal/service/account_service.go`：`AccountRepository` 同时保留本地 `ListTempUnschedulableByPlatform` 与上游 `ListModelAvailabilityCandidates`。
- `backend/internal/service/admin_account.go`：预取条件同时保留本地 probe 关闭逻辑和上游 `ProbeEnabled` 更新场景。
- `backend/internal/service/gateway_multiplatform_test.go`：测试桩同时实现临时不可调度查询与 model-availability 查询。
- `backend/internal/service/gemini_multiplatform_test.go`：同上，Gemini 测试桩完整实现两个 repository 契约。
- `backend/internal/service/grok_media.go`：保留本地严格解析、form 转换、模型映射和 CLI 代理约束；吸收视频 owner binding、状态查询、官方签名 content URL/redirect 限制。
- `backend/internal/service/openai_alpha_search.go`：保留本地 failover 副作用入口，采用上游 PAT/工具端点 401 不永久置错的限制。
- `backend/internal/service/openai_embeddings.go`：保留响应头；仅在账号未被永久禁用时允许 pool-mode 同账号重试。
- `backend/internal/service/openai_gateway_grok.go`：保留本地 upstream-attempt/延迟记录，吸收一次 `invalid encrypted_content` 清理重试。
- `backend/internal/service/openai_gateway_passthrough.go`：使用 canonical model 处理账号上游错误，保留本地 passthrough failover/ops 记录。
- `backend/internal/service/openai_images.go`：保留响应头，并以未永久禁用为前提决定同账号 pool retry。
- `backend/internal/service/openai_images_responses.go`：保留响应头和 OAuth 条件重试；Responses image 错误保持不可同账号重试。
- `deploy/config.example.yaml`：保留 128 MiB 本地读取上限说明，同时纳入 32 MiB `text_max_body_size` 和上游 8 MiB 非流响应默认值。
- `frontend/src/App.vue`：同时保留自定义菜单可见性与 favicon 更新。
- `frontend/src/components/account/CreateAccountModal.vue`：采用上游创建 payload 中的 billing probe/expiry 字段，同时保持本地表单能力。
- `frontend/src/components/account/__tests__/CreateAccountModal.spec.ts`：保留本地 passthrough/mixed-channel fixture；不以冲突覆盖替换为不兼容的旧建号辅助函数。
- `frontend/src/views/admin/SettingsView.vue`：保留 gateway runtime/scheduler 设置与 session binding，纳入 `step_up_enabled` 开关；本地前端 step-up 重试 dialog 未被误报为仍保留的契约。

## 无文本冲突审查

以下是自动合入、但因本地契约而明确审查的关键文件；它们不属于上述 U 集合。

- `backend/cmd/server/wire.go`、`backend/internal/securityaudit/prompt_module.go`：保留 Task 9 修复的 `wire.Bind(new(PromptAdminService), new(*PromptService))`，因此没有恢复 generate 时的缺 provider 状态。
- `backend/internal/handler/content_moderation_helper.go`、`backend/internal/handler/openai_gateway_handler.go`：Task 9 恢复的 `checkContentModeration` 委派仍在 Responses/Messages/WS 入口之前；已有 failover、failed-usage 和 WS failover 路径未被替换。
- `backend/internal/service/openai_ws_forwarder_ingress.go`、`backend/internal/service/openai_ws_v2_passthrough_adapter.go`：仅作无文本冲突审查，既有 WS sticky/failover 生命周期保留；本任务未执行其测试。
- `backend/internal/service/admin_service_passthrough_fields_test.go`：自动合入 model-availability repository 测试桩，未破坏本地 passthrough 契约。
- `backend/migrations/183_ops_ingress_reject_aggregates.sql`、`backend/migrations/184_auth_cache_invalidation_outbox.sql`：作为完整文件名新增并吸收。既有 `172_video_per_second_billing_metadata.sql`、`181_group_duplicate_operation_id.sql` 保留；Task 9 staged migration 测试仍枚举 183/184，migration runner 的并发覆盖也保留。

## 关键合入结论

- step-up/session：上游 `step_up_enabled` 设置开关与本地 session binding、`SetStepUpDeps` 注入和 scheduler/runtime 设置并存；管理员侧不因 tag 合入丢失本地会话/step-up 边界。
- scheduler：模型级冷却在 canonical model 下处理；advanced/layered scheduler、fallback/WaitPlan、DB fresh recheck 和 platform/Grok sticky 保持为本地保护链路。repository 与各测试桩已同时满足 temp-unschedulable 和 model-availability 契约。
- Grok media：本地 request spooling、redaction、sticky、usage snapshot 与 tag 的 generation eligibility、视频 owner binding/content proxy、官方 URL 限制并存；Grok Responses 仍记录 upstream attempt，并仅作一次 encrypted-content 清理重试。
- Task 9 既有修复：staged migration 测试、content moderation 委派、WS failover 路径、`PromptAdminService` Wire bind 均保留在 merge tree。本 Task 未重新执行这些既有测试，不能把它们写成当前轮 GREEN。

## 实际执行与范围

implementer 实际执行的命令及结果如下，均来自本次 merge/ledger 记录：

- `git rev-parse "v0.1.161^{}"`：输出 `19149ca196eeae4a4482e5299dc6fa4ba0b06c8c`。
- `git merge --no-ff --no-commit v0.1.161`：进入受审 merge 状态；随后原始 U 集合如上。
- `git diff --name-only --diff-filter=U`、精确 conflict-marker 扫描、`git ls-files -u`：均为空（完成冲突解决后）。
- `git diff --cached --check`：通过。
- `go test -run '^$' ./cmd/server ./internal/handler/admin ./internal/handler ./internal/service ./internal/server/routes`：通过；这是编译/定向包检查，不是运行测试用例。
- `pnpm exec vue-tsc --noEmit`：通过。

本 Task 没有执行 tests、build、generate 或 integration；尤其 **Task 12 的 full test/build/generate/integration gates 未运行**。旧 Task 11 媒体 TDD、RED/GREEN 和 handler/service 测试命令均不属于本报告，也未带入。

## 风险与 Task 12

- 风险信号：v0.1.161 的 26 个冲突虽已静态和定向编译核对，但尚未经过本 tag 的 full gate；Task 9 的远程 migration GREEN 是前序证据，不替代当前 merge 验证。
- 风险信号：既有 Windows `user-mapped section` 生成文件锁根因未定；环境型 DingTalk、TLS/JA3、Prompt Audit Redis/PostgreSQL、无 OpenAI key skip 仍需分类观察。
- Task 12 应验证：full test/build、两轮 generate 及 generated diff、migration/integration、step-up/session、advanced/layered scheduler/fallback/DB recheck/sticky、Grok media spooling/redaction/video URL、content moderation 和 HTTP/WS failover。

## 第 1 轮复审修复

- 修复提交：`1b80f95c9 fix: correct v0.1.161 conflict resolutions`。merge `f2158292c`、ledger `4c37dfccb` 和本报告原始提交 `89d5cb794` 均未改写。
- Finding 1 step-up RED：`go test ./internal/handler/admin -run '^TestUpdateSettings(EnableStepUp|DisableStepUp)' -count=1` 的五项转换测试均期望拒绝、实际 `200`。GREEN：恢复启用前管理员 session/TOTP 校验、关闭时 `EnforceStepUpAlways` 与 SettingsView TOTP 重试后，同一聚焦命令及 Ops snapshot 测试通过；未恢复 `openai-first-token-timeout`。
- Finding 2 Grok RED：新增 `TestGrokVideoStatus_RejectsSchedulerAccountOtherThanOwnerBinding` 期望 `404`、实际 `200`。GREEN：改为写入外层 `boundLookupAccountID` 后，owner 不同的调度选择不会转发；同组无请求体 status 测试也注入 owner binding 后通过，spooling 生命周期保持原断言。
- Finding 3 API Key 创建 RED：`pnpm exec vitest run src/components/account/__tests__/CreateAccountModal.spec.ts --reporter=dot` 缺失 quota/notify 字段。GREEN：API Key 路径回到 `createAccountAndFinish`，保留 `upstream_billing_probe_enabled`、`expires_at` 与暂停 payload，15 项通过。
- Finding 4 billing probe UI RED：`SettingsView.spec.ts` 找不到 `upstream-billing-probe-settings`。GREEN：恢复加载、保存、状态和 gateway 卡，25 项通过。
- Finding 5 Ops runtime RED：新增 `TestUpdateSettingsPublishesOpsMonitoringSnapshot` 在持久化后仍读取到 enabled。GREEN：持久化成功后调用 `SetMonitoringEnabled`，聚焦 Go 测试通过。
- Finding 6 示例 YAML RED：新增 `TestExampleConfigGatewayBodyLimits` 报 `yaml: line 178: found a tab character that violates indentation`。GREEN：使用空格缩进，保留 `text_max_body_size=33554432`，并将 `upstream_response_read_max_bytes=134217728` 与 128 MiB 契约对齐；解析及数值断言通过。
- 本轮精确 GREEN：上述三组 Go 命令、两个 Vitest 文件和 `pnpm exec vue-tsc --noEmit` 均退出 `0`；`git diff --check` 通过。Vitest 仍有既有 `router-link` / jsdom 网络 stderr advisory，退出码为 `0`。
- 残余 Task 12 风险：未运行 full test/build/generate/integration；仍需执行全量门禁，以及 step-up/session、scheduler、Grok media、migration/integration 与 HTTP/WS failover 的端到端验证。
