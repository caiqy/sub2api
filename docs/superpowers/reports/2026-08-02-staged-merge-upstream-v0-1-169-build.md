# Stage 0 基线身份账本

- 实施时间：2026-08-02T09:23:14+08:00
- Comet 确认的隔离位置：`D:/Caiqy/Projects/Github/sub2api`
- 绑定分支：`feature/20260802/staged-merge-upstream-v0-1-169`
- immutable source base：`e9a0e4aa53b5d9d5f5c84986cfadd8098dc8e4f3`
- execution base：`10ee678a49c389958315bfdb1466796dc715f2e5`

## Source-to-execution 规划路径

`git merge-base --is-ancestor` 已确认 execution base 是 immutable source base 的后代。tree diff 与 commit path 均仅包含以下 planning-only 路径：

- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet.yaml`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/artifacts.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/checkpoint.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/context.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/handoff/brainstorm-summary.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/handoff/spec-context.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/handoff/spec-context.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/run-state.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/skill-snapshots/9bd4ffab011ae18aef91dc0db336ffc12d454b513229178c15b8b75d50930ba1/package.json`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/skill-snapshots/9bd4ffab011ae18aef91dc0db336ffc12d454b513229178c15b8b75d50930ba1/sha256`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/state-events.jsonl`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/subagent-progress.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.comet/trajectory.jsonl`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/.openspec.yaml`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/design.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/proposal.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/specs/upstream-release-sync/spec.md`
- `docs/openspec/changes/staged-merge-upstream-v0-1-169/tasks.md`
- `docs/superpowers/plans/2026-08-02-staged-merge-upstream-v0-1-169.md`
- `docs/superpowers/specs/2026-08-02-staged-merge-upstream-v0-1-169-design.md`

## 已验证身份

- source `VERSION`：`0.1.165.4`
- execution `VERSION`：`0.1.165.4`
- source receipt migration blob：`c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6`
- source outbox migration blob：`502ecec1caf9f76e022c2e83acf3707190539301`
- runtime selection：存在，唯一允许的运行时未跟踪状态为 `?? .comet/current-change.json`
- 目标 tag：`v0.1.166`、`v0.1.168`、`v0.1.169`
- Windows build shell：`D:/scoop/shims/bash.exe` 存在且可由 `Get-Command` 解析。

## TDD 与边界

- RED：N/A（证据型 docs-only 任务）
- GREEN：N/A（证据型 docs-only 任务）
- 不创建生产代码或行为变更，因此不伪造失败测试。
- 禁止 push、tag、release、deploy、镜像、SSH/服务器/数据库/Redis/Nginx 操作。

## Task 2 upstream tag manifest

- 状态：通过
- 实施时间：2026-08-02
- `git fetch upstream --tags --prune` 已完成；`upstream/main` 从 `7ceabb3fd` 更新到 `b74024c78`。

### 正式 tag manifest

| Tag | Peeled SHA | 预期 commit/file 数 |
| --- | --- | --- |
| `v0.1.166` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` | `62/142` |
| `v0.1.168` | `99c8e4bf7564823bafbab369acab6539e734c1bb` | `36/170` |
| `v0.1.169` | `26d894ef4f50645a4bf1030e378ac892f17d0223` | `38/72` |

### 命令证据

```powershell
git fetch upstream --tags --prune
```

- exit code：`0`
- 关键输出：`fcb185abf..357215105  cla-signatures -> upstream/cla-signatures`、`7ceabb3fd..b74024c78  main -> upstream/main`；新增 `upstream/feat/moderation-proxy-and-smtp-starttls`、`upstream/fix/issue-5148-stream-partial-usage-billing`、`upstream/fix/issue-5152-classifier-multi-system-entries`、`upstream/fix/openai-ws-passthrough-close-frame-race`。

```powershell
git rev-parse 'v0.1.166^{}'
git rev-parse 'v0.1.168^{}'
git rev-parse 'v0.1.169^{}'
```

- 各命令 exit code：`0`
- 输出依次为：`dc893dd0b8eab41df5be595ae9fcd1aa74a062b8`、`99c8e4bf7564823bafbab369acab6539e734c1bb`、`26d894ef4f50645a4bf1030e378ac892f17d0223`。

```powershell
git merge-base --is-ancestor v0.1.166 v0.1.168
git merge-base --is-ancestor v0.1.168 v0.1.169
```

- 两个命令均无标准输出且 exit code 为 `0`：`v0.1.166` 是 `v0.1.168` 的祖先，`v0.1.168` 是 `v0.1.169` 的祖先。

```powershell
git for-each-ref refs/tags --merged=upstream/main --format='%(refname:short)' --sort=-v:refname
```

- exit code：`0`
- 关键输出（按版本降序）：`v0.1.169`、`v0.1.168`、`v0.1.166`、`v0.1.165`；最新正式 `v0.1.*` tag 是 `v0.1.169`，未发现更高正式 tag。

```powershell
git log --oneline v0.1.169..upstream/main
```

- exit code：`0`
- 以下提交明确在 release 范围外，未合并：

```text
b74024c78 Merge pull request #5167 from Wei-Shaw/fix/openai-ws-passthrough-close-frame-race
21aacde0b fix(openai-ws): keep downstream writes off the relay cancellation context
b22f73e72 Merge pull request #5154 from Wei-Shaw/fix/issue-5148-stream-partial-usage-billing
d6d53052f chore: update sponsors
bd52e5d77 fix(gateway): record observed usage when anthropic stream is interrupted
d4cada3b6 Merge pull request #5089 from feeeei/fix/openai_sse_rate_limit
8f5caef78 Merge pull request #5153 from Wei-Shaw/fix/issue-5152-classifier-multi-system-entries
2ef124629 fix(anthropic): recognize classifier requests with extra system entries
dd9a177a6 chore: update sponsors
85a27fae3 fix(openai): retry SSE rate limits as HTTP 429
eb1c5c7ee Merge pull request #5146 from tudoujunha/codex/fix-responses-tool-output-media
d9fba8fe7 Merge pull request #5101 from Tongzai123/feat/admin-select-all-filtered-results
15b3c0c5a Merge pull request #5145 from zvensmoluya/codex/update-auto-review-pricing
2e338af82 Merge pull request #5085 from feeeei/main
682c4fe0e Merge pull request #5147 from Wei-Shaw/feat/moderation-proxy-and-smtp-starttls
948b63c9c feat(moderation): route content moderation through configurable proxy server
4c80d160d fix(email): unify SMTP connection path between send and test-connection
570ea74d1 Merge pull request #5117 from gaoren002/feat/prompt-audit-blocking-latest-input
2980ff385 Merge pull request #5094 from wucm667/feat/issue-5065-compact-homepage
04c96a201 Merge pull request #4981 from INKCR0W/fix/openai-preserve-codex-namespaces
07f980b99 Merge pull request #5084 from apple-ouyang/codex/fix-openai-compaction-encrypted-retry
fe2172586 fix(openai): recover stale encrypted compaction
698547418 fix(pricing): keep Auto-review rates evidence-based
f54e9827a fix(pricing): update Codex Auto-review rates
d29acc29a Merge pull request #5066 from wucm667/fix/issue-5051-subscription-quota-window
66998918b Merge pull request #5143 from wucm667/fix/issue-5138-codex-instructions
da6194c1c Merge pull request #5112 from chenty2333/fix/openai-stream-capacity-pool-retry
132d446ca Merge pull request #5133 from dawnx/fix/payment-visible-method-wipe
0eac363e6 Merge pull request #5120 from Vibeone/fix/grok-pool-mode-cooldown-bypass
796313e99 Merge pull request #5131 from wucm667/fix/issue-5125-image-data-url-offload
c772d1866 Merge pull request #5130 from moonfunjohn/codex/fix-epay-method-selector-overflow
2bf9c6d56 修复工具输出图片桥接
94df1fffc Merge pull request #5124 from wucm667/fix/issue-5105-filter-grok-billing-ping
30967d5d9 fix(grok): ping 帧统一改写为 SSE 注释并限制过滤缓冲
dfdbc2770 fix(openai): default missing passthrough instructions
beeb2f989 test(settings): include compact home in API contracts
77d4df954 test(grok): check filter body close error
7ceabb3fd chore: sync VERSION to 0.1.169 [skip ci]
d6467f6eb fix(images): decode data URLs during task offload
8ed9f754c fix(payment): prevent method selector overflow
3deb2f17d fix(payment): 保存系统设置时不再清空可见支付方式配置
baaae8e12 fix(grok): filter billing ping response events
5c9629ddb fix(grok): unify pool mode bypass for all default cooldown paths
4d13925c9 fix(grok): skip entitlement 403 cooldown for pool mode accounts
d74e669a2 feat(security-audit): add narrow blocking audit scope
7d3bf86e5 fix(openai): retry streamed capacity errors in pool mode
2a871ec85 feat(admin): 支持按筛选结果全选账号
a35ff9613 feat(admin): 为账号批量删除增加并发限制
739c0ff9c feat(home): add compact home page preset to avoid abuse classification
3d99acb0a 优化模型广场UI:筛选栏换行对齐、模型排序与表格留白
7b6111f2f fix(subscription): align quota windows with subscription term
272735b0a fix(openai): preserve Codex namespace tools on OAuth Responses forwarding
```

### TDD 与自审

- RED：N/A（证据型 docs-only 任务）
- GREEN：N/A（证据型 docs-only 任务）
- 不创建生产代码或行为变更，因此不伪造失败测试。
- 未修改 Plan、OpenSpec `tasks.md`、`.comet/subagent-progress.md`、selection、应用源码或配置。
- 未执行 merge、push、tag、release、deploy、镜像、SSH/服务器/数据库/Redis/Nginx 操作。

### 风险信号与顾虑

- 风险信号：fetch 时 `upstream/main` 从 `7ceabb3fd` 前进到 `b74024c78`，其全部 `v0.1.169..upstream/main` 提交已在本账本中明确排除。
- 顾虑：无阻塞顾虑；release 上界和祖先链均符合范围。

## Task 3 能力矩阵与冲突台账

- 实施时间：2026-08-02
- 执行基线：`ba860106150aa296699778e9a1af644a0995b4f9`
- Comet 配置语言：`zh-CN`
- 本节是 Stage 0 的保护面证据，不表示 Task 4 聚焦测试已执行或通过。

### Changed-files 输入

以下三个命令均在上述执行基线上运行；清单为后续行为审查输入，不能以测试全绿替代矩阵结论。

```powershell
git diff --name-only v0.1.165..v0.1.166
```

- 清单计数：`142`

```text
.github/workflows/backend-ci.yml
README.md
README_CN.md
README_JA.md
assets/partners/logos/apikl.png
assets/partners/logos/byteplus.png
assets/partners/logos/huoshan.png
assets/partners/logos/miyaip.png
assets/partners/logos/tokeneum.png
backend/cmd/server/VERSION
backend/go.mod
backend/go.sum
backend/internal/config/config.go
backend/internal/config/config_test.go
backend/internal/handler/admin/setting_handler_partial_payload_test.go
backend/internal/handler/admin/setting_handler_runtime.go
backend/internal/handler/admin/setting_handler_update.go
backend/internal/handler/admin/usage_handler.go
backend/internal/handler/admin/usage_handler_request_type_test.go
backend/internal/handler/dto/settings.go
backend/internal/handler/endpoint.go
backend/internal/handler/endpoint_test.go
backend/internal/handler/gateway_handler_chat_completions.go
backend/internal/handler/gateway_handler_error_fallback_test.go
backend/internal/handler/gateway_handler_responses.go
backend/internal/handler/openai_gateway_credential_failover_test.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/openai_gateway_handler_test.go
backend/internal/handler/openai_gateway_reasoning_failover.go
backend/internal/handler/openai_gateway_reasoning_failover_test.go
backend/internal/middleware/rate_limiter.go
backend/internal/middleware/rate_limiter_test.go
backend/internal/pkg/apicompat/responses_client_tools.go
backend/internal/pkg/apicompat/responses_client_tools_test.go
backend/internal/pkg/apicompat/responses_to_anthropic_request.go
backend/internal/pkg/apicompat/responses_to_anthropic_tool_pairing_test.go
backend/internal/pkg/apicompat/responses_to_anthropic_tools_test.go
backend/internal/pkg/apicompat/types.go
backend/internal/pkg/claude/constants.go
backend/internal/pkg/usagestats/usage_log_types.go
backend/internal/repository/account_repo.go
backend/internal/repository/account_repo_upstream_billing_probe_due_integration_test.go
backend/internal/repository/account_repo_upstream_billing_probe_due_test.go
backend/internal/repository/usage_log_repo_breakdown_test.go
backend/internal/repository/usage_log_repo_query.go
backend/internal/repository/usage_log_repo_request_type_test.go
backend/internal/repository/usage_log_repo_trend.go
backend/internal/securityaudit/prompt_config.go
backend/internal/securityaudit/prompt_config_store.go
backend/internal/securityaudit/prompt_config_test.go
backend/internal/securityaudit/prompt_guard_test.go
backend/internal/securityaudit/prompt_handler.go
backend/internal/securityaudit/prompt_handler_test.go
backend/internal/securityaudit/prompt_service.go
backend/internal/securityaudit/prompt_types.go
backend/internal/securityaudit/prompt_worker_test.go
backend/internal/server/middleware/panel_rate_limit.go
backend/internal/server/middleware/panel_rate_limit_test.go
backend/internal/server/router.go
backend/internal/server/routes/admin.go
backend/internal/server/routes/auth.go
backend/internal/server/routes/auth_rate_limit_test.go
backend/internal/server/routes/ops_ingress_reject_routes_test.go
backend/internal/server/routes/payment.go
backend/internal/server/routes/prompt_audit_route_coverage_test.go
backend/internal/server/routes/user.go
backend/internal/service/account_stats_pricing_test.go
backend/internal/service/account_test_service.go
backend/internal/service/admin_service_group_test.go
backend/internal/service/antigravity_gateway_compat.go
backend/internal/service/antigravity_gateway_compat_stream.go
backend/internal/service/antigravity_gateway_compat_test.go
backend/internal/service/antigravity_gateway_streaming.go
backend/internal/service/billing_service.go
backend/internal/service/composite_model_route.go
backend/internal/service/composite_route_resolver_test.go
backend/internal/service/domain_constants.go
backend/internal/service/gateway_claude_oauth_body.go
backend/internal/service/gateway_forward.go
backend/internal/service/gateway_forward_as_responses.go
backend/internal/service/gateway_forward_as_responses_test.go
backend/internal/service/gateway_hotpath_optimization_test.go
backend/internal/service/gateway_record_usage_test.go
backend/internal/service/gateway_usage_billing.go
backend/internal/service/gemini_chat_completions_compat_service.go
backend/internal/service/gemini_error_policy_test.go
backend/internal/service/gemini_messages_compat_service.go
backend/internal/service/gemini_messages_compat_service_test.go
backend/internal/service/ollama_cloud_usage_test.go
backend/internal/service/openai_gateway_grok_test.go
backend/internal/service/openai_gateway_record_usage_test.go
backend/internal/service/openai_gateway_request_body.go
backend/internal/service/openai_gateway_request_body_reasoning_test.go
backend/internal/service/openai_gateway_usage.go
backend/internal/service/openai_ws_forwarder.go
backend/internal/service/openai_ws_forwarder_ingress.go
backend/internal/service/openai_ws_http_bridge.go
backend/internal/service/openai_ws_http_bridge_test.go
backend/internal/service/openai_ws_v2/passthrough_relay.go
backend/internal/service/openai_ws_v2/passthrough_relay_test.go
backend/internal/service/openai_ws_v2_passthrough_adapter.go
backend/internal/service/payment_service.go
backend/internal/service/payment_stats.go
backend/internal/service/payment_stats_test.go
backend/internal/service/pricing_service.go
backend/internal/service/pricing_service_test.go
backend/internal/service/setting_panel_rate_limit.go
backend/internal/service/setting_panel_rate_limit_test.go
backend/internal/service/setting_service.go
backend/internal/service/setting_update.go
backend/internal/service/token_refresh_pool_health_test.go
backend/internal/service/usage_log_helpers.go
backend/resources/model-pricing/model_prices_and_context_window.json
deploy/Caddyfile
deploy/EDGE_SECURITY.md
deploy/test-caddyfile-cache.sh
frontend/src/api/admin/settings.ts
frontend/src/components/admin/payment/DailyRevenueChart.vue
frontend/src/components/admin/payment/OrderStatsCards.vue
frontend/src/components/admin/payment/PaymentMethodChart.vue
frontend/src/components/admin/payment/TopUsersLeaderboard.vue
frontend/src/components/admin/usage/UsageFilters.vue
frontend/src/components/channels/AvailableChannelsTable.vue
frontend/src/components/channels/__tests__/AvailableChannelsTable.spec.ts
frontend/src/components/common/GroupOptionItem.vue
frontend/src/components/common/Select.vue
frontend/src/components/common/__tests__/GroupOptionItem.spec.ts
frontend/src/components/common/__tests__/Select.spec.ts
frontend/src/components/user/monitor/MonitorTimeline.vue
frontend/src/i18n/locales/en/admin/overview.ts
frontend/src/i18n/locales/en/admin/settings.ts
frontend/src/i18n/locales/zh/admin/overview.ts
frontend/src/i18n/locales/zh/admin/settings.ts
frontend/src/types/payment.ts
frontend/src/views/admin/GroupsView.vue
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/UsageView.vue
frontend/src/views/admin/__tests__/SettingsView.spec.ts
frontend/src/views/admin/__tests__/UsageView.spec.ts
frontend/src/views/admin/orders/AdminPaymentDashboardView.vue
frontend/src/views/auth/RegisterView.vue
frontend/src/views/auth/__tests__/RegisterView.spec.ts
```

```powershell
git diff --name-only v0.1.166..v0.1.168
```

- 清单计数：`170`

```text
backend/cmd/server/VERSION
backend/cmd/server/wire_gen.go
backend/go.mod
backend/go.sum
backend/internal/config/config.go
backend/internal/config/config_test.go
backend/internal/config/webauthn_test.go
backend/internal/handler/admin/setting_handler.go
backend/internal/handler/admin/setting_handler_audit.go
backend/internal/handler/admin/setting_handler_update.go
backend/internal/handler/auth_handler.go
backend/internal/handler/auth_oauth_pending_flow_test.go
backend/internal/handler/auth_session_revocation_test.go
backend/internal/handler/dto/settings.go
backend/internal/handler/gateway_handler.go
backend/internal/handler/gateway_handler_intercept_test.go
backend/internal/handler/gateway_handler_warmup_intercept_unit_test.go
backend/internal/handler/handler.go
backend/internal/handler/model_plaza_handler.go
backend/internal/handler/model_plaza_handler_test.go
backend/internal/handler/openai_codex_models_handler.go
backend/internal/handler/passkey_handler.go
backend/internal/handler/passkey_handler_test.go
backend/internal/handler/setting_handler.go
backend/internal/handler/user_handler_test.go
backend/internal/handler/wire.go
backend/internal/pkg/antigravity/response_transformer.go
backend/internal/pkg/antigravity/stream_transformer.go
backend/internal/repository/api_key_repo.go
backend/internal/repository/api_key_repo_integration_test.go
backend/internal/repository/api_key_repo_lost_update_integration_test.go
backend/internal/repository/auth_cache_invalidation_outbox_integration_test.go
backend/internal/repository/passkey_repo.go
backend/internal/repository/passkey_session_store.go
backend/internal/repository/promo_code_repo.go
backend/internal/repository/user_profile_identity_repo_contract_test.go
backend/internal/repository/user_repo.go
backend/internal/repository/user_repo_email_identity_integration_test.go
backend/internal/repository/user_repo_email_lookup_unit_test.go
backend/internal/repository/user_repo_integration_test.go
backend/internal/repository/user_repo_lost_update_integration_test.go
backend/internal/repository/user_repo_sort_integration_test.go
backend/internal/repository/wire.go
backend/internal/securityaudit/prompt_config.go
backend/internal/securityaudit/prompt_config_integration_test.go
backend/internal/securityaudit/prompt_config_store.go
backend/internal/securityaudit/prompt_config_test.go
backend/internal/securityaudit/prompt_handler_test.go
backend/internal/securityaudit/prompt_logging.go
backend/internal/securityaudit/prompt_logging_test.go
backend/internal/securityaudit/prompt_service_test.go
backend/internal/securityaudit/prompt_types.go
backend/internal/server/api_contract_test.go
backend/internal/server/http.go
backend/internal/server/middleware/admin_auth_test.go
backend/internal/server/middleware/api_key_auth_google_test.go
backend/internal/server/middleware/api_key_auth_test.go
backend/internal/server/middleware/audit_log.go
backend/internal/server/middleware/audit_log_test.go
backend/internal/server/middleware/backend_mode_guard.go
backend/internal/server/middleware/optional_jwt_auth.go
backend/internal/server/middleware/optional_jwt_auth_test.go
backend/internal/server/middleware/wire.go
backend/internal/server/router.go
backend/internal/server/routes/auth.go
backend/internal/server/routes/model_plaza.go
backend/internal/server/routes/user.go
backend/internal/service/account.go
backend/internal/service/account_test_service.go
backend/internal/service/account_usage_service.go
backend/internal/service/admin_group.go
backend/internal/service/admin_service_apikey_test.go
backend/internal/service/admin_service_delete_test.go
backend/internal/service/admin_service_email_identity_sync_test.go
backend/internal/service/admin_service_update_balance_test.go
backend/internal/service/admin_service_update_user_rpm_test.go
backend/internal/service/admin_user.go
backend/internal/service/api_key_service.go
backend/internal/service/api_key_service_cache_test.go
backend/internal/service/api_key_service_delete_test.go
backend/internal/service/api_key_service_quota_test.go
backend/internal/service/api_key_service_update_fields_test.go
backend/internal/service/audit_log.go
backend/internal/service/auth_email_binding.go
backend/internal/service/auth_email_oauth_auto.go
backend/internal/service/auth_service.go
backend/internal/service/auth_service_email_bind_test.go
backend/internal/service/billing_service.go
backend/internal/service/billing_service_test.go
backend/internal/service/channel_plaza.go
backend/internal/service/channel_plaza_test.go
backend/internal/service/content_moderation.go
backend/internal/service/content_moderation_test.go
backend/internal/service/domain_constants.go
backend/internal/service/gateway_anthropic_apikey_passthrough_test.go
backend/internal/service/gateway_claude_oauth_body.go
backend/internal/service/gateway_prompt_test.go
backend/internal/service/gemini_chat_completions_compat_service.go
backend/internal/service/gemini_messages_compat_service.go
backend/internal/service/openai_codex_models_service.go
backend/internal/service/openai_codex_models_service_test.go
backend/internal/service/openai_compat_model.go
backend/internal/service/openai_compat_model_test.go
backend/internal/service/openai_gateway_messages.go
backend/internal/service/openai_gateway_messages_chat_fallback.go
backend/internal/service/openai_gateway_messages_chat_fallback_test.go
backend/internal/service/openai_gateway_record_usage_test.go
backend/internal/service/openai_live.go
backend/internal/service/openai_live_lifecycle_test.go
backend/internal/service/openai_model_mapping.go
backend/internal/service/openai_oauth_model_support_test.go
backend/internal/service/passkey.go
backend/internal/service/passkey_test.go
backend/internal/service/setting_features.go
backend/internal/service/setting_parse.go
backend/internal/service/setting_public.go
backend/internal/service/setting_service_update_test.go
backend/internal/service/setting_update.go
backend/internal/service/settings_view.go
backend/internal/service/thinking_protocol.go
backend/internal/service/thinking_protocol_test.go
backend/internal/service/user_service.go
backend/internal/service/user_service_test.go
backend/internal/service/user_service_update_fields_test.go
backend/internal/service/wire.go
backend/internal/setup/setup.go
backend/internal/setup/setup_test.go
backend/migrations/191_passkey_credentials.sql
deploy/config.example.yaml
frontend/src/api/__tests__/passkey.spec.ts
frontend/src/api/admin/settings.ts
frontend/src/api/index.ts
frontend/src/api/modelPlaza.ts
frontend/src/api/passkey.ts
frontend/src/components/account/AccountStatusIndicator.vue
frontend/src/components/account/ModelWhitelistSelector.vue
frontend/src/components/account/__tests__/AccountStatusIndicator.spec.ts
frontend/src/components/account/__tests__/ModelWhitelistSelector.spec.ts
frontend/src/components/layout/AppHeader.vue
frontend/src/components/modelPlaza/ModelPlazaContent.vue
frontend/src/components/modelPlaza/PlazaFilterBar.vue
frontend/src/components/modelPlaza/PlazaGroupSection.vue
frontend/src/components/modelPlaza/PlazaModelPricingTable.vue
frontend/src/components/modelPlaza/PlazaNavBar.vue
frontend/src/components/modelPlaza/__tests__/PlazaModelPricingTable.spec.ts
frontend/src/components/user/profile/ProfilePasskeyCard.vue
frontend/src/features/prompt-audit/__tests__/components.spec.ts
frontend/src/features/prompt-audit/components/EndpointPool.vue
frontend/src/features/prompt-audit/types.ts
frontend/src/i18n/locales/en/admin/promptAudit.ts
frontend/src/i18n/locales/en/admin/settings.ts
frontend/src/i18n/locales/en/common.ts
frontend/src/i18n/locales/en/dashboard.ts
frontend/src/i18n/locales/zh/admin/promptAudit.ts
frontend/src/i18n/locales/zh/admin/settings.ts
frontend/src/i18n/locales/zh/common.ts
frontend/src/i18n/locales/zh/dashboard.ts
frontend/src/router/index.ts
frontend/src/stores/__tests__/app.spec.ts
frontend/src/stores/app.ts
frontend/src/stores/auth.ts
frontend/src/types/index.ts
frontend/src/utils/featureFlags.ts
frontend/src/utils/platformColors.ts
frontend/src/utils/pricing.ts
frontend/src/views/ModelPlazaView.vue
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/__tests__/SettingsView.spec.ts
frontend/src/views/auth/LoginView.vue
frontend/src/views/user/ProfileView.vue
```

```powershell
git diff --name-only v0.1.168..v0.1.169
```

- 清单计数：`72`

```text
.github/workflows/backend-ci.yml
.goreleaser.simple.yaml
.goreleaser.yaml
Dockerfile.goreleaser
backend/cmd/server/VERSION
backend/internal/config/config.go
backend/internal/config/config_test.go
backend/internal/handler/available_channel_handler.go
backend/internal/handler/available_channel_handler_test.go
backend/internal/handler/gemini_v1beta_handler.go
backend/internal/handler/openai_gateway_handler_test.go
backend/internal/pkg/xai/oauth.go
backend/internal/repository/account_repo.go
backend/internal/repository/account_repo_integration_test.go
backend/internal/repository/account_repo_temp_unsched_test.go
backend/internal/securityaudit/prompt_qwen3guard.go
backend/internal/securityaudit/prompt_qwen3guard_test.go
backend/internal/server/routes/gateway.go
backend/internal/server/routes/gateway_test.go
backend/internal/service/account_test_service.go
backend/internal/service/billing_service.go
backend/internal/service/billing_service_test.go
backend/internal/service/claude_code_validator.go
backend/internal/service/claude_code_validator_test.go
backend/internal/service/email_message.go
backend/internal/service/email_message_test.go
backend/internal/service/email_service.go
backend/internal/service/gateway_anthropic_apikey_passthrough_test.go
backend/internal/service/gateway_context_management_test.go
backend/internal/service/gateway_count_tokens.go
backend/internal/service/gemini_chat_completions_compat_service.go
backend/internal/service/gemini_messages_compat_service.go
backend/internal/service/gemini_upstream_url.go
backend/internal/service/gemini_upstream_url_test.go
backend/internal/service/notification_email_service_test.go
backend/internal/service/openai_account_scheduler.go
backend/internal/service/openai_account_scheduler_test.go
backend/internal/service/openai_gateway_request_body.go
backend/internal/service/openai_gateway_scheduling.go
backend/internal/service/openai_gateway_service.go
backend/internal/service/openai_gateway_service_test.go
backend/internal/service/openai_proxy_stream_circuit.go
backend/internal/service/openai_proxy_stream_circuit_test.go
backend/internal/service/ops_cleanup_service.go
backend/internal/service/ops_scheduled_report_service_test.go
backend/internal/service/pricing_service.go
backend/internal/service/pricing_service_test.go
backend/internal/service/token_refresh_service_candidates_test.go
backend/internal/service/upstream_path_guard.go
backend/internal/service/upstream_path_guard_test.go
backend/resources/model-pricing/model_prices_and_context_window.json
deploy/Dockerfile
deploy/config.example.yaml
deploy/docker-compose.dev.yml
deploy/docker-compose.local.yml
deploy/docker-compose.standalone.yml
deploy/docker-compose.yml
deploy/tests/docker-compose-security-test.sh
deploy/tests/docker-runtime-resources-test.sh
frontend/src/components/payment/SubscriptionPlanCard.vue
frontend/src/components/payment/__tests__/SubscriptionPlanCard.spec.ts
frontend/src/i18n/locales/en/admin/channels.ts
frontend/src/i18n/locales/en/admin/settings.ts
frontend/src/i18n/locales/zh/admin/channels.ts
frontend/src/i18n/locales/zh/admin/settings.ts
frontend/src/utils/__tests__/subscriptionQuota.spec.ts
frontend/src/utils/subscriptionQuota.ts
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/SubscriptionsView.vue
frontend/src/views/admin/__tests__/SettingsView.spec.ts
frontend/src/views/user/SubscriptionsView.vue
frontend/src/views/user/__tests__/PaymentView.spec.ts
```

### Canonical 14 行能力矩阵

状态含义：`protected` 是 brief 提供的既有测试/静态保护面，不等同于本任务测试 PASS；`manual` 是仅有 changed-files 与人工审查证据；`gap` 是必须由 Task 4 执行的自动契约。未出现 `unverified`，因为本任务未以 Docker/Testcontainers 环境为由跳过验证。

| # | 能力契约 | 入口/调用链 | 关键文件 | 受影响 tag | 聚焦测试 | 人工审查点 | 当前状态 | 证据/待验证项 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | advanced/layered scheduler、DB recheck、WaitPlan fallback | scheduler -> DB fresh recheck -> WaitPlan -> gateway sticky fallback | `backend/internal/service/openai_account_scheduler_layered_test.go`; `backend/internal/handler/gateway_handler_sticky_fallback_test.go` | `v0.1.166`、`v0.1.168`、`v0.1.169` 范围 | `TestLayered_GroupedAccountPassesDBFreshRecheck`; `TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount`; `TestLayered_FallbackWaitPlanRechecksPrivacyRequirementAgainstDB`; `TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot` | 逐一将调度、DB 重查和 fallback 关联至三段清单，不把路由快照误当作账号重查 | `protected` | brief 给出目标测试与文件；本任务未执行，Task 4 运行聚焦测试确认行为。 |
| 2 | Grok/platform/session/previous-response sticky、privacy、image capability | OpenAI WS 入口 -> layered scheduler -> session/previous-response sticky 与 privacy/image 再校验 | `backend/internal/service/openai_account_scheduler_layered_test.go`; OpenAI WS 入口 | `v0.1.166`、`v0.1.168`、`v0.1.169` 范围 | `TestLayered_SessionStickyPreservesGrokBinding`; `TestLayered_PreviousResponseStickyEnabled`; `TestLayered_SessionStickyRecheckHonorsImageCapability`; `TestLayered_PreviousResponseStickyHonorsRequirePrivacySet` | 检查 sticky 不绕过 privacy 或 image capability，且 Grok 绑定不丢失 | `protected` | brief 的既有目标测试是静态保护面；Task 4 尚未执行它们。 |
| 3 | OpenAI HTTP/WS、Live、turn ownership、最终 outbound model、failed usage、prompt-cache reuse、透传字段 | OpenAI HTTP/WS -> Live/relay -> terminal event -> usage snapshot | `backend/internal/handler/openai_gateway_handler.go`; `backend/internal/service/openai_ws_forwarder.go`; `backend/internal/service/openai_ws_v2/passthrough_relay.go` | `v0.1.166`、`v0.1.168`、`v0.1.169` 范围 | `TestOpenAIGatewayService_Forward_WSv2_ResponseDoneUsageParsed`; `TestRelay_OnTurnComplete_UsesCurrentResponseCreateModel`; `TestRelay_OnTurnComplete_PerTerminalEvent`; `TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot` | 复核终端事件只归属当前 turn，失败 usage 与最终 outbound model 不相互污染 | `protected` | brief 列出 relay/usage 测试证据；Task 4 未运行，不能标记 PASS。 |
| 4 | prompt/security audit | gateway POST route -> prompt audit classification -> WS first/subsequent turn gates | `backend/internal/handler/security_audit_helper.go`; `backend/internal/server/routes/prompt_audit_route_coverage_test.go` | `v0.1.168`、`v0.1.169` 范围 | `TestEveryGatewayPOSTRouteIsClassifiedForPromptAuditCoverage`; `TestResponsesWebSocketHasFirstAndSubsequentTurnPromptGates` | 审查所有 gateway POST 与 Responses WS 两个 gate 都不遗漏 | `protected` | canonical 路由覆盖测试为既有保护面；Task 4 负责执行。 |
| 5 | Images 精确审计与文本生命周期 | OpenAI Images 统一入口 -> audit/moderation controls -> runtime scope -> text release -> upstream | `backend/internal/handler/openai_images.go`; `backend/internal/handler/openai_images_controls_test.go` | `v0.1.169` 范围 | `TestOpenAIImages_UnifiedAuditRunsLegacyOnce`; `TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease`; `TestOpenAIImages_SecurityAuditUsesFrozenPayloadBeforeRelease`; `TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream`; `TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream`; `TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload`; `TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope`; `TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce` | 六项精确契约：统一入口；legacy 单次；单次冻结；关闭态零大 payload；运行态/范围后求值；文本在 `ReleaseText` 前可用，且在阻断上游前释放 multipart/OAuth 文本 | `protected` | brief 提供八个 Images 目标测试与六项精确契约；Task 4 未执行。 |
| 6 | request-body replay/spooling/cleanup | Images body ingress -> inline spool -> mapped effective body replay -> cleanup | `backend/internal/handler/openai_images_controls_test.go`; `backend/internal/service/openai_gateway_request_body.go` | `v0.1.169` 范围 | `TestOpenAIImages_InlineSpoolKeepsRawBodyAndOmitsSnapshots`; `TestOpenAIGatewayHandlerImages_MultipartReplayUsesMappedEffectiveBody` | 审查 raw body、snapshot 省略、mapped body 和 cleanup 的顺序 | `protected` | brief 的两项聚焦测试为静态保护面；Task 4 待运行。 |
| 7 | 异步图片任务、对象存储、图片输入计费、上游计费倍率 | prompt guard -> async image task -> object storage -> usage record -> upstream peak-rate billing | `backend/internal/service/image_task.go`; `backend/internal/service/image_storage.go`; `backend/internal/service/gateway_record_usage_test.go` | `v0.1.169` 范围 | `TestAsyncImagePromptGuardRunsBeforeTaskCreation`; `TestAsyncImageSuccessfulPrecheckIsNotRepeatedByDetachedExecution`; `TestGatewayServiceRecordUsage_EmptyImageSizeDefaultsBeforeBillingAndPersistence`; `TestGatewayServiceRecordUsage_PeakRateAffectsTokenModeImageOutputTokens` | 核对 guard 先于建任务、预检不重复、默认 image size 与倍率落在计费持久化之前 | `protected` | canonical 测试名称已记录；没有执行结果，Task 4 关闭。 |
| 8 | settings 热更新和部分更新 | admin settings request -> update handler -> setting update service -> runtime reload | `backend/internal/handler/admin/setting_handler_update.go`; `backend/internal/service/setting_update.go` | `v0.1.166`、`v0.1.168` 范围 | `TestUpdateSettingsPartialPayloadKeepsUnsentKeys`; `TestUpdateSettingsFullPayloadStillClearsSentEmptyFields` | 区分 partial 未发送 key 保留与 full payload 显式空字段清除 | `protected` | 两阶段清单包含相关 handler/service；Task 4 尚未运行对应测试。 |
| 9 | repository scoped updates、用户/API Key 更新、会话绑定与 step-up | user/API key repository -> service -> Passkey/auth route -> session revocation | `backend/internal/repository/user_repo.go`; `backend/internal/repository/api_key_repo.go`; Passkey/auth 路由 | `v0.1.168` 范围 | `TestUserRepoSuite`; `TestUserRepoAPIKeyGroupFilterSuite`; Passkey handler/service 测试和 auth session revocation 测试 | 核对 scope filter、并发更新、身份绑定、step-up 与会话撤销边界 | `protected` | v0.1.168 清单和 brief 指定的 repository/auth 测试是现有证据；Task 4 待执行。 |
| 10 | subscription quota cycle reset | subscription service -> receipt/outbox repository -> quota cache invalidation -> frontend quota view | `backend/internal/service/subscription_service.go`; receipt/outbox repositories; `frontend/src/utils/subscriptionQuota.ts` | `v0.1.169` 范围 | `TestCalculateQuotaCycleAdvance_ResetsOnlySingleExhaustedWindow`; `TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce`; `TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails`; `TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack` | 审查单窗口 reset、并发一次扣减、receipt 回滚、outbox 语义变更与回滚 | `protected` | canonical 测试与 v0.1.169 frontend 清单已登记；Task 4 待运行。 |
| 11 | 用户资源控制、分组复制、批量限额 | user/group handler -> repository -> frontend group/account pages | user/group handler、repository、frontend group/account 页面 | `v0.1.166`、`v0.1.168` 范围 | 当前相关 handler/repository/Vitest 目标与 changed-files 审查；记录每条入口、测试和人工结论 | 为每个资源控制、复制和批量限额入口补充逐项结论，避免仅凭聚合清单放行 | `manual` | brief 明确要求以 changed-files 审查记录入口和结论；无 Task 4 执行结果。 |
| 12 | 前端本地能力 | 菜单/设置/用量/订阅/渠道展示/移动端 -> 对应 views、components、utils | 菜单、设置、用量、订阅、渠道展示、移动端以及上游触及页面 | `v0.1.166`、`v0.1.168`、`v0.1.169` 范围 | `frontend/src/views/admin/__tests__/SettingsView.spec.ts`; `frontend/src/views/admin/__tests__/UsageView.spec.ts`; `frontend/src/utils/__tests__/subscriptionQuota.spec.ts`; `frontend/src/components/channels/__tests__/AvailableChannelsTable.spec.ts` | 核对本地菜单与各页面兼容性，尤其移动端和上游触及页面 | `protected` | 三段清单均含前端变更，brief 列出四个 Vitest 目标；Task 4 尚未执行。 |
| 13 | pricing、count_tokens、release fallback、部署安全 | pricing service/resources -> count_tokens gateway route -> release/compose/Caddy paths | pricing service/resources、gateway route、release/compose/Caddy 文件 | `v0.1.166`、`v0.1.169` 范围 | pricing/count_tokens/route 聚焦测试、资源 diff 审查；两个 `deploy/tests/*.sh` 仅在 v0.1.169 merge 后存在时由已验证 bash 执行 | 审查 pricing 资源、fallback、Caddy/compose；确认 merge 后才判定两个 deploy 脚本是否可运行 | `manual` | changed-files 显示 pricing、count tokens、release/compose/Caddy 与两个 deploy 脚本；Task 4 前不声称脚本或自动测试通过。 |
| 14 | Ent/Wire、Go/pnpm 依赖、migrations | Ent schema/Wire -> generated server -> Go/pnpm dependency resolution -> migration runner | `backend/ent/schema/`; `backend/internal/**/wire.go`; `backend/cmd/server/wire_gen.go`; `backend/go.mod`; `frontend/pnpm-lock.yaml`; `backend/migrations/` | `v0.1.166`、`v0.1.168` 范围 | 两轮 generate 无 diff、依赖工具生成结果、migration runner unit/integration、前后端 full gate | 审查 generated output、锁文件、migration 顺序及全量 gate，防止生成物漂移 | `gap` | 两轮 generate、依赖工具、migration 与 full gate 均尚未在 Task 4 执行；Task 4 关闭此阻塞项。 |

初始状态统计：`protected=11`，`manual=2`，`gap=1`，`unverified=0`。任何 `protected` 仅表示本 ledger 有既有的静态/历史保护面，不是本任务的测试 PASS 断言。

### 六分类冲突台账

截至 Task 3 尚无实际冲突。下表保留实际冲突的可追踪字段；记录只能使用闭集分类：`上游修复`、`本地定制`、`接口/配置演进`、`版本/依赖`、`生成代码`、`migration`。

| 文件名 | 分类 | ours 行为 | theirs 行为 | 融合结果 | 验证证据 |
| --- | --- | --- | --- | --- |

### TDD、文档审计与提交边界

- RED：N/A（证据型 docs-only 任务；无生产代码或行为变更，未伪造 RED）。
- GREEN：N/A（证据型 docs-only 任务；Task 4 尚未运行任何自动契约）。
- 文档审计证据：三段 `git diff --name-only` 命令及其完整清单、计数 `142/170/72`、14 行矩阵和空冲突台账均在本节；提交前运行 `git diff --check` 与 staged allowlist 检查。
- 提交 allowlist：仅 `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`；不勾选 Plan/OpenSpec。
- 禁止操作遵守情况：未执行 fetch、merge、push、tag、release、deploy、镜像或服务器操作。

### Task 3 风险信号与顾虑

- 风险信号：三个区间均有单文件变更超过 200 行，尤其 `v0.1.165..v0.1.166` 的 gateway/Antigravity/panel-rate-limit 测试与服务文件、`v0.1.166..v0.1.168` 的 passkey/model-plaza/frontend 页面、`v0.1.168..v0.1.169` 的 upstream path guard；必须由 Task 4 聚焦验证，不能仅凭 changed-files 放行。
- 顾虑：Task 4 前，所有未运行的自动契约均不能声称 PASS；Ent/Wire、依赖和 migration 的 `gap` 阻塞下一阶段。
