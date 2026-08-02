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

状态含义：本轮未运行 Task 4 自动契约，也未完成入口级/调用链/资源 diff 人工审查；因此 14 行均为 `gap`。下表的测试名称仅是 Task 4 待运行目标，历史测试存在、静态 changed-files 命中和当前 PASS 是三个不同证据单元，不能互相替代。`unverified` 仅保留给 Docker/Testcontainers 不可用导致的 integration 契约，本轮不提前使用。

| # | 能力契约 | 入口/调用链 | 关键文件 | 受影响 tag | 聚焦测试 | 人工审查点 | 当前状态 | 证据/待验证项 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | advanced/layered scheduler、DB recheck、WaitPlan fallback | scheduler -> DB fresh recheck -> WaitPlan -> gateway sticky fallback | `backend/internal/service/openai_account_scheduler_layered_test.go`; `backend/internal/handler/gateway_handler_sticky_fallback_test.go` | `v0.1.166`、`v0.1.168`、`v0.1.169` | `TestLayered_GroupedAccountPassesDBFreshRecheck`; `TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount`; `TestLayered_FallbackWaitPlanRechecksPrivacyRequirementAgainstDB`; `TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot` | 尚未完成人工入口/调用链审查 | `gap` | tag 映射：`v0.1.166` -> `backend/internal/handler/gateway_handler_error_fallback_test.go`、`backend/internal/handler/openai_gateway_handler.go`; `v0.1.168` -> `backend/internal/handler/gateway_handler.go`、`backend/internal/service/openai_gateway_messages_chat_fallback.go`; `v0.1.169` -> `backend/internal/service/openai_account_scheduler.go`、`backend/internal/service/openai_gateway_scheduling.go`。上述测试仅为待运行目标。 |
| 2 | Grok/platform/session/previous-response sticky、privacy、image capability | OpenAI WS 入口 -> layered scheduler -> session/previous-response sticky 与 privacy/image 再校验 | `backend/internal/service/openai_account_scheduler_layered_test.go`; OpenAI WS 入口 | `v0.1.166`、`v0.1.168`、`v0.1.169` | `TestLayered_SessionStickyPreservesGrokBinding`; `TestLayered_PreviousResponseStickyEnabled`; `TestLayered_SessionStickyRecheckHonorsImageCapability`; `TestLayered_PreviousResponseStickyHonorsRequirePrivacySet` | 尚未完成人工入口/调用链审查 | `gap` | tag 映射：`v0.1.166` -> `backend/internal/service/openai_gateway_grok_test.go`、`backend/internal/service/openai_ws_forwarder.go`; `v0.1.168` -> `backend/internal/service/openai_live.go`、`backend/internal/service/openai_gateway_messages.go`; `v0.1.169` -> `backend/internal/service/openai_account_scheduler.go`、`backend/internal/service/openai_gateway_scheduling.go`。上述测试仅为待运行目标。 |
| 3 | OpenAI HTTP/WS、Live、turn ownership、最终 outbound model、failed usage、prompt-cache reuse、透传字段 | OpenAI HTTP/WS -> Live/relay -> terminal event -> usage snapshot | `backend/internal/handler/openai_gateway_handler.go`; `backend/internal/service/openai_ws_forwarder.go`; `backend/internal/service/openai_ws_v2/passthrough_relay.go` | `v0.1.166`、`v0.1.168`、`v0.1.169` | `TestOpenAIGatewayService_Forward_WSv2_ResponseDoneUsageParsed`; `TestRelay_OnTurnComplete_UsesCurrentResponseCreateModel`; `TestRelay_OnTurnComplete_PerTerminalEvent`; `TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot` | 尚未完成人工入口/调用链审查 | `gap` | tag 映射：`v0.1.166` -> `backend/internal/handler/openai_gateway_handler.go`、`backend/internal/service/openai_ws_forwarder.go`、`backend/internal/service/openai_ws_v2/passthrough_relay.go`; `v0.1.168` -> `backend/internal/service/openai_live.go`、`backend/internal/service/openai_gateway_messages.go`; `v0.1.169` -> `backend/internal/service/openai_gateway_service.go`、`backend/internal/service/openai_gateway_scheduling.go`。上述测试仅为待运行目标。 |
| 4 | prompt/security audit | gateway POST route -> prompt audit classification -> WS first/subsequent turn gates | `backend/internal/handler/security_audit_helper.go`; `backend/internal/server/routes/prompt_audit_route_coverage_test.go` | `v0.1.166`、`v0.1.168`、`v0.1.169` | `TestEveryGatewayPOSTRouteIsClassifiedForPromptAuditCoverage`; `TestResponsesWebSocketHasFirstAndSubsequentTurnPromptGates` | 尚未完成人工入口/调用链审查 | `gap` | tag 映射：`v0.1.166` -> `backend/internal/securityaudit/prompt_guard_test.go`、`backend/internal/securityaudit/prompt_handler.go`、`backend/internal/server/routes/prompt_audit_route_coverage_test.go`; `v0.1.168` -> `backend/internal/securityaudit/prompt_config_store.go`、`backend/internal/securityaudit/prompt_logging.go`; `v0.1.169` -> `backend/internal/securityaudit/prompt_qwen3guard.go`。上述测试仅为待运行目标。 |
| 5 | Images 精确审计与文本生命周期 | OpenAI Images 统一入口 -> audit/moderation controls -> runtime scope -> text release -> upstream | `backend/internal/handler/openai_images.go`; `backend/internal/handler/openai_images_controls_test.go` | `v0.1.169` | `TestOpenAIImages_UnifiedAuditRunsLegacyOnce`; `TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease`; `TestOpenAIImages_SecurityAuditUsesFrozenPayloadBeforeRelease`; `TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream`; `TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream`; `TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload`; `TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope`; `TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce` | 六项待审契约：统一入口；legacy 单次；单次冻结；关闭态零大 payload；运行态/范围后求值；文本在 `ReleaseText` 前可用。尚未完成人工入口/调用链审查。 | `gap` | tag 映射：`v0.1.169` -> `backend/internal/service/openai_gateway_request_body.go`（Images body/control 调用链映射）；三段清单均无 `backend/internal/handler/openai_images.go` 或 controls 测试的直接变更。八个测试与六项契约均为待运行目标。 |
| 6 | request-body replay/spooling/cleanup | Images body ingress -> inline spool -> mapped effective body replay -> cleanup | `backend/internal/handler/openai_images_controls_test.go`; `backend/internal/service/openai_gateway_request_body.go` | `v0.1.166`、`v0.1.169` | `TestOpenAIImages_InlineSpoolKeepsRawBodyAndOmitsSnapshots`; `TestOpenAIGatewayHandlerImages_MultipartReplayUsesMappedEffectiveBody` | 尚未完成人工入口/调用链审查 | `gap` | tag 映射：`v0.1.166` -> `backend/internal/service/openai_gateway_request_body.go`; `v0.1.169` -> `backend/internal/service/openai_gateway_request_body.go`。两项测试仅为待运行目标。 |
| 7 | 异步图片任务、对象存储、图片输入计费、上游计费倍率 | prompt guard -> async image task -> object storage -> usage record -> upstream peak-rate billing | `backend/internal/service/image_task.go`; `backend/internal/service/image_storage.go`; `backend/internal/service/gateway_record_usage_test.go` | `v0.1.166`、`v0.1.168`、`v0.1.169` | `TestAsyncImagePromptGuardRunsBeforeTaskCreation`; `TestAsyncImageSuccessfulPrecheckIsNotRepeatedByDetachedExecution`; `TestGatewayServiceRecordUsage_EmptyImageSizeDefaultsBeforeBillingAndPersistence`; `TestGatewayServiceRecordUsage_PeakRateAffectsTokenModeImageOutputTokens` | 尚未完成人工入口/调用链审查 | `gap` | tag 映射：`v0.1.166` -> `backend/internal/service/gateway_record_usage_test.go`、`backend/internal/service/billing_service.go`; `v0.1.168` -> `backend/internal/service/content_moderation.go`、`backend/internal/service/billing_service.go`; `v0.1.169` -> `backend/internal/service/billing_service.go`、`backend/internal/service/openai_gateway_service.go`。三段清单均无 `image_task.go` 或 `image_storage.go` 的直接变更；测试仅为待运行目标。 |
| 8 | settings 热更新和部分更新 | admin settings request -> update handler -> setting update service -> runtime reload | `backend/internal/handler/admin/setting_handler_update.go`; `backend/internal/service/setting_update.go` | `v0.1.166`、`v0.1.168` | `TestUpdateSettingsPartialPayloadKeepsUnsentKeys`; `TestUpdateSettingsFullPayloadStillClearsSentEmptyFields` | 尚未完成人工入口/调用链审查 | `gap` | tag 映射：`v0.1.166` -> `backend/internal/handler/admin/setting_handler_update.go`、`backend/internal/service/setting_update.go`; `v0.1.168` -> `backend/internal/handler/admin/setting_handler_update.go`、`backend/internal/service/setting_update.go`。两项测试仅为待运行目标。 |
| 9 | repository scoped updates、用户/API Key 更新、会话绑定与 step-up | user/API key repository -> service -> Passkey/auth route -> session revocation | `backend/internal/repository/user_repo.go`; `backend/internal/repository/api_key_repo.go`; Passkey/auth 路由 | `v0.1.168` | `TestUserRepoSuite`; `TestUserRepoAPIKeyGroupFilterSuite`; Passkey handler/service 测试和 auth session revocation 测试 | 尚未完成人工入口/调用链审查 | `gap` | tag 映射：`v0.1.168` -> `backend/internal/repository/user_repo.go`、`backend/internal/repository/api_key_repo.go`、`backend/internal/handler/passkey_handler.go`、`backend/internal/handler/auth_handler.go`、`backend/internal/handler/auth_session_revocation_test.go`。测试仅为待运行目标。 |
| 10 | subscription quota cycle reset | subscription service -> receipt/outbox repository -> quota cache invalidation -> frontend quota view | `backend/internal/service/subscription_service.go`; receipt/outbox repositories; `frontend/src/utils/subscriptionQuota.ts` | `v0.1.169` | `TestCalculateQuotaCycleAdvance_ResetsOnlySingleExhaustedWindow`; `TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce`; `TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails`; `TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack` | 尚未完成人工入口/调用链审查 | `gap` | tag 映射：`v0.1.169` -> `frontend/src/utils/subscriptionQuota.ts`、`frontend/src/utils/__tests__/subscriptionQuota.spec.ts`、`frontend/src/views/admin/SubscriptionsView.vue`、`frontend/src/views/user/SubscriptionsView.vue`。backend subscription/receipt/outbox 路径未直接出现在三段清单；四项测试仅为待运行目标。 |
| 11 | 用户资源控制、分组复制、批量限额 | user/group handler -> repository -> frontend group/account pages | user/group handler、repository、frontend group/account 页面 | `v0.1.166`、`v0.1.168` | 当前相关 handler/repository/Vitest 目标与 changed-files 审查；记录每条入口、测试和人工结论 | 未完成用户资源、分组复制、批量限额的逐入口审查结论 | `gap` | tag 映射：`v0.1.166` -> `backend/internal/service/admin_service_group_test.go`、`backend/internal/repository/account_repo.go`、`frontend/src/views/admin/GroupsView.vue`; `v0.1.168` -> `backend/internal/service/admin_user.go`、`backend/internal/service/user_service.go`、`backend/internal/repository/user_repo.go`、`frontend/src/components/account/ModelWhitelistSelector.vue`。无当前测试或人工审查结论。 |
| 12 | 前端本地能力 | 菜单/设置/用量/订阅/渠道展示/移动端 -> 对应 views、components、utils | 菜单、设置、用量、订阅、渠道展示、移动端以及上游触及页面 | `v0.1.166`、`v0.1.168`、`v0.1.169` | `frontend/src/views/admin/__tests__/SettingsView.spec.ts`; `frontend/src/views/admin/__tests__/UsageView.spec.ts`; `frontend/src/utils/__tests__/subscriptionQuota.spec.ts`; `frontend/src/components/channels/__tests__/AvailableChannelsTable.spec.ts` | 尚未完成人工入口/调用链审查 | `gap` | tag 映射：`v0.1.166` -> `frontend/src/views/admin/SettingsView.vue`、`frontend/src/views/admin/UsageView.vue`、`frontend/src/components/channels/AvailableChannelsTable.vue`; `v0.1.168` -> `frontend/src/views/ModelPlazaView.vue`、`frontend/src/views/user/ProfileView.vue`、`frontend/src/router/index.ts`; `v0.1.169` -> `frontend/src/views/admin/SubscriptionsView.vue`、`frontend/src/views/user/SubscriptionsView.vue`、`frontend/src/views/admin/SettingsView.vue`。四个 Vitest 文件仅为待运行目标。 |
| 13 | pricing、count_tokens、release fallback、部署安全 | pricing service/resources -> count_tokens gateway route -> release/compose/Caddy paths | pricing service/resources、gateway route、release/compose/Caddy 文件 | `v0.1.166`、`v0.1.169` | pricing/count_tokens/route 聚焦测试、资源 diff 审查；两个 `deploy/tests/*.sh` 仅在 v0.1.169 merge 后存在时由已验证 bash 执行 | 未完成资源 diff、fallback 与部署路径人工审查 | `gap` | tag 映射：`v0.1.166` -> `backend/internal/service/pricing_service.go`、`backend/resources/model-pricing/model_prices_and_context_window.json`、`deploy/Caddyfile`; `v0.1.169` -> `backend/internal/service/pricing_service.go`、`backend/internal/service/gateway_count_tokens.go`、`.goreleaser.yaml`、`deploy/docker-compose.yml`、`deploy/tests/docker-compose-security-test.sh`、`deploy/tests/docker-runtime-resources-test.sh`。测试和 bash 执行均待 Task 4。 |
| 14 | Ent/Wire、Go/pnpm 依赖、migrations | Ent schema/Wire -> generated server -> Go/pnpm dependency resolution -> migration runner | `backend/ent/schema/`; `backend/internal/**/wire.go`; `backend/cmd/server/wire_gen.go`; `backend/go.mod`; `frontend/pnpm-lock.yaml`; `backend/migrations/` | `v0.1.166`、`v0.1.168` | 两轮 generate 无 diff、依赖工具生成结果、migration runner unit/integration、前后端 full gate | 尚未完成 generated output、依赖、migration 和 full gate 审查 | `gap` | tag 映射：`v0.1.166` -> `backend/go.mod`、`backend/go.sum`; `v0.1.168` -> `backend/cmd/server/wire_gen.go`、`backend/internal/handler/wire.go`、`backend/internal/repository/wire.go`、`backend/internal/service/wire.go`、`backend/go.mod`、`backend/go.sum`、`backend/migrations/191_passkey_credentials.sql`。三段清单无 `backend/ent/schema/` 或 `frontend/pnpm-lock.yaml` 的直接变更；所有 gate 待 Task 4。 |

修复后状态统计：`protected=0`，`manual=0`，`gap=14`，`unverified=0`。测试名称存在仅说明 Task 4 有待运行目标；在没有当前直接行为测试 PASS 前，不能构成 `protected`，在没有已完成入口级/调用链/资源 diff 结论前，不能构成 `manual`。

### 六分类冲突台账

截至 Task 3 尚无实际冲突。下表保留实际冲突的可追踪字段；记录只能使用闭集分类：`上游修复`、`本地定制`、`接口/配置演进`、`版本/依赖`、`生成代码`、`migration`。

| 文件名 | 分类 | ours 行为 | theirs 行为 | 融合结果 | 验证证据 |
| :--- | :--- | :--- | :--- | :--- | :--- |

### TDD、文档审计与提交边界

- RED：N/A（证据型 docs-only 任务；无生产代码或行为变更，未伪造 RED）。
- GREEN：N/A（证据型 docs-only 任务；Task 4 尚未运行任何自动契约）。
- 文档审计证据：已只读复跑三段 `git diff --name-only`；完整清单与计数 `142/170/72` 保留在本节。14 行逐 tag 映射仅引用清单中的实际路径，未将静态命中或历史测试名称表达为当前 PASS。
- 测试名称存在只是待运行目标，不是 `protected` 证据；`gap=14` 由 Task 4 逐项关闭。
- 提交前运行 `git diff --check`、矩阵 14 行/状态统计/六列冲突台账格式检查与 staged allowlist 检查。
- 提交 allowlist：仅 `docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md`；不勾选 Plan/OpenSpec。
- 禁止操作遵守情况：未执行 fetch、merge、push、tag、release、deploy、镜像或服务器操作。

### Task 3 风险信号与顾虑

- 风险信号：三个区间均有单文件变更超过 200 行，尤其 `v0.1.165..v0.1.166` 的 gateway/Antigravity/panel-rate-limit 测试与服务文件、`v0.1.166..v0.1.168` 的 passkey/model-plaza/frontend 页面、`v0.1.168..v0.1.169` 的 upstream path guard；必须由 Task 4 聚焦验证，不能仅凭 changed-files 放行。
- 顾虑：Task 4 前，14 项均为 `gap`，所有未运行的自动契约和未完成的人工审查均不能声称 PASS 或已完成；这些 `gap` 阻塞下一阶段。

## Task 4 阶段 0 本地保护测试与生成稳定性

- 实施时间：2026-08-02
- 实施基线：`22ede455341978f0403926b42639c3d05f5f417a`
- Images 保护提交：`746c0ccdef0b4536adc9867b0cfc89357b1b9787`（仅 `backend/internal/handler/openai_images_controls_test.go`）。
- RED：N/A。三个新增保护测试首次运行即通过，证明的是既有行为；没有为了流程制造 RED，也未修改 `openai_images.go` 或 `security_audit_helper.go`。

### Images TDD 与 CodeGraph 审查

- 破坏点：关闭审计时提前求值 payload、legacy runtime/group scope 检查前求值、coordinator 的 prompt/legacy 双消费者重复求值。
- 新增 `TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload`、`TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope`、`TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce`；前两者以原子计数验证零次 provider 调用和继续上游路径，第三者以原子计数验证单次冻结、legacy 单次调用和审计读取释放前文本。
- CodeGraph 调用链结论：`Images` -> `checkSecurityAuditLazy` -> `runSecurityAuditLazy` -> `Coordinator.CheckLazy`。后者通过 `sync.Once` 的 `bodyOnce` 在 blocking prompt 与 legacy moderation 间共享冻结 payload；`ContentModerationService.CheckLazy` 先检查 runtime、enabled/mode 和 group/model scope，之后才调用 provider；返回后 `Images` 才执行 `ReleaseMultipartValues` 与 `parsed.ReleaseText`。

### 命令证据

```powershell
go test -count=1 ./internal/handler -run '^(TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$'
```

- exit code：`0`；关键输出：`ok github.com/Wei-Shaw/sub2api/internal/handler 1.584s`。

```powershell
go test -count=1 ./internal/service -run '^(TestLayered_GroupedAccountPassesDBFreshRecheck|TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount|TestLayered_SessionStickyPreservesGrokBinding|TestLayered_SessionStickyRecheckHonorsImageCapability|TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot|TestCalculateQuotaCycleAdvance_ResetsOnlySingleExhaustedWindow|TestAdvanceQuotaCycle_RejectsTwoExhaustedWindowsBeforeUpdate|TestAdminResetQuota_UsesCommittedResetVersionForCacheInvalidation|TestCheckAndResetWindows_UsesCommittedResetVersionForCacheInvalidation)$'
```

- exit code：`1`；未进入命名测试。`subscription_cache_invalidation_outbox_test.go` 缺少 `newPaymentOrderLifecycleTestClient`、`redeemSubscriptionCacheStub`、`newTermLockingUserSubRepo`、`exhaustedQuotaSubscription`。
- 根因审查：四个 helper 均存在于当前仓库，但定义文件带 `//go:build unit`；无 tag 的 brief 命令与 `make test` 均未包含这些定义。诊断命令 `go test -tags unit -count=1 ./internal/service -run '^(TestLayered_GroupedAccountPassesDBFreshRecheck|TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount|TestLayered_SessionStickyPreservesGrokBinding|TestLayered_SessionStickyRecheckHonorsImageCapability|TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot|TestCalculateQuotaCycleAdvance_ResetsOnlySingleExhaustedWindow|TestAdvanceQuotaCycle_RejectsTwoExhaustedWindowsBeforeUpdate|TestAdminResetQuota_UsesCommittedResetVersionForCacheInvalidation|TestCheckAndResetWindows_UsesCommittedResetVersionForCacheInvalidation)$'` 的 exit code 为 `0`，关键输出为 `ok github.com/Wei-Shaw/sub2api/internal/service 1.763s`。此诊断不改变 brief 原命令的失败记录，也未越界修复 service 测试。

```powershell
go test -count=1 ./internal/handler -run '^(TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot|TestOpenAIImages_UnifiedAuditRunsLegacyOnce|TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease|TestOpenAIImages_SecurityAuditUsesFrozenPayloadBeforeRelease|TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$'
```

- exit code：`0`；关键输出：`ok github.com/Wei-Shaw/sub2api/internal/handler 4.694s`。

```powershell
go test -count=1 ./internal/server/routes -run '^(TestEveryGatewayPOSTRouteIsClassifiedForPromptAuditCoverage|TestResponsesWebSocketHasFirstAndSubsequentTurnPromptGates)$'
```

- exit code：`0`；关键输出：`ok github.com/Wei-Shaw/sub2api/internal/server/routes 1.902s`。

```powershell
pnpm --dir frontend exec vitest run src/utils/__tests__/subscriptionQuota.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/UsageView.spec.ts src/components/channels/__tests__/AvailableChannelsTable.spec.ts
```

- exit code：`0`；`4` files、`50` tests passed。输出含已有 Browserslist、`router-link` 和 jsdom `AggregateError` 警告，未导致测试失败。

```powershell
make test
```

- exit code：`2`；根因与无 tag service 聚焦命令相同。其余已运行包包含 `internal/handler`、`internal/handler/admin`、`internal/repository`、`internal/securityaudit`、`internal/server/routes` 均为 `ok`；不得将该 full gate 标为通过。

```powershell
make "VERSION=0.1.165.4" "SHELL=D:/scoop/shims/bash.exe" build
```

- exit code：`0`；backend `CGO_ENABLED=0 go build` 与 frontend `vue-tsc -b && vite build` 均成功。Vite 输出既有动态/静态 import 和 chunk-size 警告。

```powershell
make -C backend generate
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git.exe diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
```

- 两次 generate exit code 均为 `0`；两次 diff exit code 均为 `1`，且仅稳定重现 `backend/cmd/server/wire_gen.go` 的 Wire provider 声明顺序漂移：`SubscriptionQuotaAdvanceReceiptRepository`、`SubscriptionCacheInvalidationOutboxRepository` 和 worker 的位置被重排，变量名从 `subscriptionQuotaAdvanceReceiptRepository` 变为 `quotaAdvanceReceiptRepository`。
- 该生成物不在 Task 4 allowlist，已在记录证据后恢复至 `HEAD`，未暂存、未提交；`backend/ent` 无 diff。此项阻塞 generated-output 契约。

```powershell
git.exe diff --check
git.exe diff --cached --check
git diff --name-only --diff-filter=U
git grep -n -I -E '^(<<<<<<<|=======|>>>>>>>)' -- ':!docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md'
git rev-parse HEAD:backend/migrations/191_subscription_quota_advance_receipts.sql
git rev-parse HEAD:backend/migrations/192_subscription_cache_invalidation_outbox.sql
```

- 前三个命令 exit code 均为 `0` 且无输出。
- conflict-marker scan exit code 为 `0`，但输出 `backend/internal/pkg/antigravity/request_transformer.go:267:===========================================`；经上下文审查，这是 `mcpXMLProtocol` 的合法多行字符串分隔符，不是未合并标记。原命令的“无输出”期望未满足，未修改非 allowlist 源码。
- 两个 OID 命令 exit code 均为 `0`，输出分别精确为 `c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6`、`502ecec1caf9f76e022c2e83acf3707190539301`。

### Task 4 当前 14 行能力矩阵

状态规则：只有本轮直接行为测试覆盖完整契约时为 `protected`；未完成的 Docker 专项契约继续为 `gap`，不提前标记 `unverified`。

| # | 能力契约 | 当前状态 | 本轮真实证据或 gap 原因 |
| --- | --- | --- | --- |
| 1 | layered scheduler、DB recheck、WaitPlan fallback | `gap` | `-tags unit` 的本轮聚焦子集通过，但 matrix 所列 privacy fallback recheck 未运行；原 brief 命令编译失败。 |
| 2 | Grok/platform/session/previous-response sticky | `gap` | `-tags unit` 的 session/image 子集通过；previous-response sticky 两项未运行。 |
| 3 | HTTP/WS/Live turn ownership、usage snapshot | `gap` | `TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot` 在 unit-tag 诊断通过；WS/Live/terminal 其余指定契约未运行。 |
| 4 | prompt/security audit route 与 WS gate | `protected` | 两个 routes 直接测试本轮通过；CodeGraph 已核对 helper -> coordinator 的审计入口。 |
| 5 | Images 审计与文本生命周期 | `protected` | 八个 Images 直接测试本轮通过；新增三条覆盖关闭态、runtime scope 惰性和 coordinator/legacy 单次冻结，并完成调用链审查。 |
| 6 | Images request-body replay/spooling/cleanup | `protected` | `make test` 本轮实际完成 `internal/handler` 包并返回 `ok`，含两个 matrix 指定 handler 测试。 |
| 7 | async image task、对象存储、图片计费 | `gap` | handler 子包通过，但 service 中图片计费目标受同一编译门禁阻断，未完成完整能力契约。 |
| 8 | settings 热更新和部分更新 | `protected` | `make test` 本轮实际完成 `internal/handler/admin` 包并返回 `ok`，含 matrix 指定 settings 测试。 |
| 9 | repository scoped update、session/passkey step-up | `protected` | `make test` 本轮实际完成 `internal/repository` 和相关 handler 包并返回 `ok`；无该能力的本轮失败。 |
| 10 | subscription quota cycle reset | `gap` | 前端 quota 测试通过，但 receipt/outbox/concurrency backend 测试受 service 编译门禁阻断。 |
| 11 | 用户资源控制、分组复制、批量限额 | `gap` | 未执行该行完整的指定入口测试或逐入口人工审查。 |
| 12 | 前端本地能力 | `protected` | 指定 Vitest 4 files/50 tests 本轮通过。 |
| 13 | pricing、count_tokens、release fallback、部署安全 | `gap` | 未运行 pricing/count_tokens 定向测试；Docker 专项契约明确留给 Task 5。 |
| 14 | Ent/Wire、依赖、migrations | `gap` | VERSION/SHELL build 和 migration OID 通过，但 full gate 失败且两轮 generate 稳定产生 Wire diff。 |

状态统计：`protected=6`，`manual=0`，`gap=8`，`unverified=0`。

### Task 4 风险信号与顾虑

- 风险信号：无 tag 的 service test 及 `make test` 都因测试 helper 被 `unit` build tag 排除而失败；该问题阻断 scheduler、usage、quota 和部分图片计费证据。
- 风险信号：两轮 generate 都重现受跟踪 `wire_gen.go` diff；其内容已恢复且未进入提交，但 generated-output 不能放行。
- 风险信号：冲突扫描正则把合法字符串分隔符作为 marker 命中；仓库没有 unmerged 文件，仍需在 controller 层决定是否修正检测模式。
- 顾虑：Docker integration 未执行，保持 `gap` 交 Task 5；本轮未执行 fetch、merge、push、tag、release、deploy、镜像、SSH、服务器、数据库、Redis 或 Nginx 操作。

## Task 4 Repaired Stage 0 Gates (Fix-2)

- 修复基线：`2c108ca1dbe3db1918da136517e31adba5fb34dd`。
- 已保留的隔离提交：Images `746c0ccdef0b4536adc9867b0cfc89357b1b9787`、outbox build tag `663955ae8`、Wire `f01473818`。
- lint 修复提交：`03d522fea7b7292730799cc2178dc072f29cee66`（仅 `backend/internal/repository/user_subscription_repo.go`）。

### Lint TDD

- RED：`golangci-lint run ./internal/repository` exit `1`，唯一产品 finding 为 `internal/repository/user_subscription_repo.go:578:18` 的 `errcheck`，即 `defer rows.Close()` 未检查返回值。brief 的行级后置正则要求同一行同时带 `errcheck` 标签，因 linter 将标签输出在独立汇总行而额外抛出；原始 finding 仍为预期的精确 RED。
- GREEN：同一 linter 命令从 `backend` 目录运行，exit `0`，输出 `0 issues.`；最小修复为 `defer func() { _ = rows.Close() }()`，与同包既有模式一致且不改变行为。

### Direct Protection Evidence

- 五组新增 non-Docker gap 测试均 exit `0`：rows 1/2 privacy and previous-response sticky；row 3 WebSocket slot ownership；row 3 Live/terminal lifecycle；row 13 pricing/count_tokens/release fallback；row 11 的 unit-tag user resources/batch/group copy 与 default user-group binding。
- 完整 brief 聚焦集全部 exit `0`：9 个 backend Go test 命令和指定 Vitest 命令；Vitest 为 `4` files、`50` tests passed。
- `make test` exit `0`；`make "VERSION=0.1.165.4" "SHELL=D:/scoop/shims/bash.exe" build` exit `0`。
- 前端测试与 build 仍输出既有 Browserslist、Vue `router-link`、jsdom、Vite dynamic-import/chunk-size 警告，但全部相关命令退出 `0`。

### Generation Evidence

所有 Task 4 generation 均由 brief 的同一 bounded-retry helper 执行；每次完整 stdout/stderr、exit code 与 diff paths 已保留在以下日志中。

| run ID | attempts | result | stdout/stderr logs |
| --- | --- | --- | --- |
| `fix2-wire-refresh` | `1` | exit `0`，无 generated diff | `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-refresh-attempt-1.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-refresh-attempt-1.stderr.log` |
| `fix2-wire-stability-1` | `1` | exit `0`，无 generated diff | `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-stability-1-attempt-1.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-stability-1-attempt-1.stderr.log` |
| `fix2-wire-stability-2` | `2` | attempt 1 exit `2`，仅 `backend/ent/announcementread/announcementread.go` 的 `user-mapped section open`，helper 恢复 Ent/Wire 后等待 2 秒；attempt 2 exit `0`，无 generated diff | `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-stability-2-attempt-1.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-stability-2-attempt-1.stderr.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-stability-2-attempt-2.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-stability-2-attempt-2.stderr.log` |
| `fix2-full-stability-1` | `1` | exit `0`，无 generated diff | `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-full-stability-1-attempt-1.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-full-stability-1-attempt-1.stderr.log` |
| `fix2-full-stability-2` | `1` | exit `0`，无 generated diff | `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-full-stability-2-attempt-1.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-full-stability-2-attempt-1.stderr.log` |

- 总 retry 次数：`1`。失败 stderr 除该 generated-path exact signature、`exit status` 和 make wrapper 外无 `error`、`fail`、`panic` 或 `fatal`；未杀进程、未调整 antivirus 或 CodeGraph。
- 每次成功后 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 均 exit `0` 且无输出。

### Static And OID Gates

- `git diff --check`、`git diff --cached --check`、`git diff --name-only --diff-filter=U` 均 exit `0` 且无输出。
- 完整行 conflict scan exit `1` 且无匹配，为指定 PASS 条件。
- migration OID 均 exit `0`：receipt `c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6`；outbox `502ecec1caf9f76e022c2e83acf3707190539301`。

### Repaired 14-Line Matrix

| # | 状态 | 当前直接证据 |
| --- | --- | --- |
| 1 | `protected` | scheduler/WaitPlan 及 privacy DB recheck 命名测试通过。 |
| 2 | `protected` | session/image 与 previous-response privacy sticky 命名测试通过。 |
| 3 | `protected` | usage snapshot、WebSocket ownership、Live/terminal lifecycle 命名测试通过。 |
| 4 | `protected` | prompt-audit route 与 subsequent-turn gate 命名测试通过。 |
| 5 | `protected` | 八个 Images 审计/文本生命周期直接测试通过。 |
| 6 | `protected` | full `make test` 通过，包含 request-body replay/spooling 覆盖。 |
| 7 | `protected` | full `make test` 通过，包含 async image、object storage 和图片计费覆盖。 |
| 8 | `protected` | full `make test` 通过，包含 settings hot/partial update 覆盖。 |
| 9 | `protected` | full `make test` 通过，包含 scoped update、session/passkey 覆盖。 |
| 10 | `gap` | Docker quota/outbox/migration integration 仅交 Task 5。 |
| 11 | `protected` | user resources、batch limits、group duplication 的 18 个 unit-tag 测试和 group binding 测试通过。 |
| 12 | `protected` | 指定 Vitest 四文件 `50` tests 通过。 |
| 13 | `protected` | pricing/count_tokens/release fallback 命名测试通过；deploy scripts 是 v0.1.169 merge 后的未来阶段证据，不是当前 gap。 |
| 14 | `protected` | full test、VERSION/SHELL build、五轮受控 generate 与 migration OID 全部通过。 |

统计：`protected=13`，`manual=0`，`gap=1`，`unverified=0`；唯一 gap 是 row 10，归属 Task 5。

### Risk Signals And Concerns

- 一次 Windows `user-mapped section open` 在精确、允许的重试条件下恢复；所有后续 generation 稳定、无 diff，但其外部环境属性应继续保留为风险信号。
- Docker quota/outbox/migration integration 未执行，严格保留给 Task 5。
- 未执行 fetch、merge、push、tag、release、deploy、服务器或镜像操作；ledger 提交前无 tracked 工作树 diff。

## Review Repair Evidence

- Review repair P1 commit：`d750199d0`（`test: observe Images audit payload lifecycle`），严格只含 `backend/internal/handler/openai_gateway_handler.go`、`backend/internal/handler/openai_images.go`、`backend/internal/handler/openai_images_controls_test.go`。
- P1 仅增加 request-scoped test hook。hook 为 nil 时仍传入原始 `parsed.ModerationBody`；没有修改 shared `handlerPromptEngine`、constructor、interface 或默认生产路径。
- 本轮没有执行 `make -C backend generate`、Docker、remote、fetch、merge、push、tag、release 或 deploy。

### P1 RED And GREEN

RED command, run from `backend`:

```powershell
$imagesRedLog = Join-Path $env:TEMP "sub2api-stage0-images-hook-red-$([guid]::NewGuid().ToString('N')).log"
go test -count=1 ./internal/handler -run '^(TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$' 2>&1 | Tee-Object -FilePath $imagesRedLog
$imagesRedExit = $LASTEXITCODE
```

- exit code：`1`；日志：`C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-images-hook-red-0a87b6d6705b4b3e86079c8459fc7910.log`。
- 关键输出仅为 `TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce` 的 provider assertion：`expected: 1`、`actual : 0`。前两个同请求 zero-count 测试完成，未出现 `build failed`、`undefined:`、Docker 或 Testcontainers 签名。
- RED 的第三个测试使用 `ModeAsync` 的非阻塞 engine，不等待 `started`；旧 callsite 仍使用真实 `parsed.ModerationBody`，所以请求 bounded completion 后测试 hook 尚未被调用。

GREEN commands, run from `backend` unless stated otherwise:

```powershell
go test -count=1 ./internal/handler -run '^(TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$'
```

- exit code：`0`；关键输出：`ok github.com/Wei-Shaw/sub2api/internal/handler 1.517s`。
- 第三个测试改用 test-local `blockingImagesPromptEngine`，并按 `started`、provider count、captured parsed pointer、`release`、`done` 顺序验证。`sync.Once` 的 `t.Cleanup(release)` 在超时或断言失败时仍解除阻塞；释放前 `Prompt` 可见，`done` 后已清空。

```powershell
go test -count=1 ./internal/handler -run '^(TestOpenAIImages_UnifiedAuditRunsLegacyOnce|TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease|TestOpenAIImages_SecurityAuditUsesFrozenPayloadBeforeRelease|TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$'
```

- exit code：`0`；关键输出：`ok github.com/Wei-Shaw/sub2api/internal/handler 5.082s`。

```powershell
go test -count=1 ./internal/handler -run '^(TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot|TestOpenAIImages_UnifiedAuditRunsLegacyOnce|TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease|TestOpenAIImages_SecurityAuditUsesFrozenPayloadBeforeRelease|TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream|TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload|TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope|TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce)$'
```

- exit code：`0`；关键输出：`ok github.com/Wei-Shaw/sub2api/internal/handler 4.380s`。

```powershell
make test
```

- exit code：`0`；关键输出：frontend `219` test files passed、`1650` tests passed。输出仍包含既有 Vue/router-link、jsdom 与 intentional error-path test warnings，但没有失败。

### Current Actual Reruns

The following commands were actually rerun in this review repair. They are not Fix-2 citations.

```powershell
go test -count=1 ./internal/service -run '^(TestLayered_GroupedAccountPassesDBFreshRecheck|TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount|TestLayered_SessionStickyPreservesGrokBinding|TestLayered_SessionStickyRecheckHonorsImageCapability|TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot|TestCalculateQuotaCycleAdvance_ResetsOnlySingleExhaustedWindow|TestAdvanceQuotaCycle_RejectsTwoExhaustedWindowsBeforeUpdate|TestAdminResetQuota_UsesCommittedResetVersionForCacheInvalidation|TestCheckAndResetWindows_UsesCommittedResetVersionForCacheInvalidation)$'
```

- exit code：`0`；关键输出：`ok github.com/Wei-Shaw/sub2api/internal/service 1.592s`。

```powershell
go test -count=1 -tags unit ./internal/service -run '^(TestSubscriptionCacheInvalidationWorker_(RequiresTombstoneAndPublishBeforeAck|UsesSafetySecondPass|PublishGetsIndependentTimeout)|TestSubscriptionCacheInvalidationFastPath_(WaitsForOuterCommit|NilCacheStillClearsLocalL1|UnknownVersionOnlyClearsLocalL1)|TestAdvanceQuotaCycle_UsesVersionedPostCommitInvalidation)$'
```

- exit code：`0`；关键输出：`ok github.com/Wei-Shaw/sub2api/internal/service 3.546s`。

```powershell
golangci-lint run ./internal/repository
```

- exit code：`0`；关键输出：`0 issues.`。

```powershell
git diff --check
git diff --cached --check
git diff --name-only --diff-filter=U
$conflictPattern = '^(<<<<<<< .+|\|\|\|\|\|\|\| .+|=======|>>>>>>> .+)$'
git grep -n -I -E $conflictPattern -- ':!docs/superpowers/reports/2026-08-02-staged-merge-upstream-v0-1-169-build.md'
$conflictExit = $LASTEXITCODE
if ($conflictExit -eq 0) { throw 'tracked conflict markers found' }
if ($conflictExit -ne 1) { throw "conflict marker scan failed: $conflictExit" }
```

- `git diff --check` exit code：`0`，无输出。
- `git diff --cached --check` exit code：`0`，无输出。
- `git diff --name-only --diff-filter=U` exit code：`0`，无输出。
- conflict scan exit code：`1`，无匹配；这是指定的 PASS 条件。

```powershell
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git status --short --untracked-files=all
```

- generated-path diff exit code：`0`，无输出。
- status exit code：`0`；关键输出仅为 `?? .comet/current-change.json`。

### Fix-2 Historical References Only

The following entries were not rerun in this review repair. They are exact command and result references to Fix-2 evidence recorded at `2026-08-02T12:00:51+08:00` in commit `31f239703220b6a0765834d37fbc5981dc8c25fd`, not claims of new execution.

```powershell
go test -count=1 ./internal/server/routes -run '^(TestEveryGatewayPOSTRouteIsClassifiedForPromptAuditCoverage|TestResponsesWebSocketHasFirstAndSubsequentTurnPromptGates)$'
```

- Fix-2 exit code：`0`；关键输出：`ok github.com/Wei-Shaw/sub2api/internal/server/routes 2.112s`。

```powershell
go test -count=1 ./internal/service -run '^(TestLayered_FallbackWaitPlanRechecksPrivacyRequirementAgainstDB|TestLayered_PreviousResponseStickyHonorsRequirePrivacySet)$'
```

- Fix-2 exit code：`0`；关键输出：两个 rows 1/2 privacy and previous-response sticky 测试通过。

```powershell
go test -count=1 ./internal/handler -run '^(TestOpenAIResponsesWebSocket_ReacquireSlotsOnSecondTurnWithoutDoubleRelease|TestOpenAIResponsesWebSocket_SecondTurnGroupAcquireFailureRollsBackUserSlot|TestOpenAIResponsesWebSocket_SecondTurnAccountAcquireFailureRollsBackUserAndGroupSlots)$'
```

- Fix-2 exit code：`0`；关键输出：三个 row 3a Responses WebSocket slot ownership/rollback 测试通过。

```powershell
go test -count=1 ./internal/service -run '^(TestRunLiveControllerClosesExpiredSession|TestLiveSidebandNormalCloseEndsCall|TestOpenAIWSPassthroughTurnLifecycle_SerializesTerminalCommitAndNextTurn)$'
```

- Fix-2 exit code：`0`；关键输出：三个 row 3b Live/sideband/terminal lifecycle 测试通过。

```powershell
go test -count=1 ./internal/service -run '^(TestGetModelPricing_Gpt54UsesStaticFallbackWhenRemoteMissing|TestOpenAIGatewayService_ForwardCountTokensAsAnthropic_OAuthFallsBackWhenPlatformEndpointUnsupported|TestGatewayServiceNewSelectionResult_ReleasesAcquiredSlotWhenHydrationFails)$'
```

- Fix-2 exit code：`0`；关键输出：三个 row 13 non-Docker pricing/count_tokens/selection-release fallback 测试通过。

```powershell
go test -count=1 -tags unit ./internal/handler ./internal/handler/admin ./internal/service -run '^(TestGetMyPlatformQuotas_EmptyReturns200WithEmptyArray|TestGetMyPlatformQuotas_D14_LazyZeroForExpiredWindow|TestGetMyPlatformQuotas_NilRepo_Returns200Empty|TestGetMyPlatformQuotas_NoAuth_Returns401|TestLazyZeroQuotaForResponse_UserViewStripsWindowStart|TestLazyZeroQuotaForResponse_AdminViewIncludesWindowStart|TestLazyZeroQuotaForResponse_ActiveWindowPreservesUsage|TestUserHandlerBatchUpdateLimitsAcceptsPartialAndZeroValues|TestUserHandlerBatchUpdateLimitsRejectsInvalidRequests|TestUserHandlerBatchUpdateLimitsAllUsesEveryListedUser|TestDuplicateGroupHandlerReturnsAdminDTOWithoutOperationMetadata|TestDuplicateGroupHandlerRejectsInvalidID|TestDuplicateGroupHandlerReplaysSameIdempotencyKey|TestDuplicateGroupHandlerRecoversAfterMarkSucceededFailure|TestDuplicateGroupCopiesConfigurationDeeplyAndResetsRuntimeState|TestDuplicateGroupRecoversSameOperationAndScopesByAdmin|TestDuplicateGroupAdvancesNameAndTruncatesUnicodeByRunes|TestDuplicateGroupAtomicCreateFailureReturnsNoCopy)$'
```

- Fix-2 exit code：`0`；关键输出：18 个 row 11 user resources、batch limits 和 group duplication unit-tag 测试通过。

```powershell
go test -count=1 ./internal/service -run '^TestUserCanBindGroupRejectsBlockedPublicGroup$'
```

- Fix-2 exit code：`0`；关键输出：row 11 default user-group binding 测试通过。

```powershell
pnpm --dir frontend exec vitest run src/utils/__tests__/subscriptionQuota.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/UsageView.spec.ts src/components/channels/__tests__/AvailableChannelsTable.spec.ts
```

- Fix-2 exit code：`0`；关键输出：`4` files、`50` tests passed。

```powershell
make "VERSION=0.1.165.4" "SHELL=D:/scoop/shims/bash.exe" build
```

- Fix-2 exit code：`0`；关键输出：`CGO_ENABLED=0 go build` 与 `vue-tsc -b && vite build` 成功。

```powershell
git rev-parse HEAD:backend/migrations/191_subscription_quota_advance_receipts.sql
git rev-parse HEAD:backend/migrations/192_subscription_cache_invalidation_outbox.sql
```

- Fix-2 exit codes：`0`、`0`；关键输出依次为 `c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6`、`502ecec1caf9f76e022c2e83acf3707190539301`。

### Fix-2 Generation Log Inspection

- Controller independently read all `12` retained Fix-2 Temp logs. The real run IDs are `fix2-wire-refresh`、`fix2-wire-stability-1`、`fix2-wire-stability-2`、`fix2-full-stability-1` and `fix2-full-stability-2`.
- `fix2-wire-refresh` attempt 1 stdout/stderr: `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-refresh-attempt-1.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-refresh-attempt-1.stderr.log`.
- `fix2-wire-stability-1` attempt 1 stdout/stderr: `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-stability-1-attempt-1.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-stability-1-attempt-1.stderr.log`.
- `fix2-wire-stability-2` attempt 1 stdout/stderr: `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-stability-2-attempt-1.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-stability-2-attempt-1.stderr.log`.
- `fix2-wire-stability-2` attempt 2 stdout/stderr: `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-stability-2-attempt-2.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-wire-stability-2-attempt-2.stderr.log`.
- `fix2-full-stability-1` attempt 1 stdout/stderr: `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-full-stability-1-attempt-1.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-full-stability-1-attempt-1.stderr.log`.
- `fix2-full-stability-2` attempt 1 stdout/stderr: `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-full-stability-2-attempt-1.log`; `C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-fix2-full-stability-2-attempt-1.stderr.log`.
- Four success attempts and the recovered retry each record `exit_code=0` and `diff_paths=`. The sole failed stdout records exit `2` and these 11 temporary `backend/ent` paths: `backend/ent/announcementread/announcementread.go`, `backend/ent/idempotencyrecord_create.go`, `backend/ent/identityadoptiondecision_update.go`, `backend/ent/promocode/promocode.go`, `backend/ent/proxy_delete.go`, `backend/ent/securitysecret/securitysecret.go`, `backend/ent/securitysecret_query.go`, `backend/ent/usagecleanuptask.go`, `backend/ent/usagelogdetail/usagelogdetail.go`, `backend/ent/userattributedefinition_query.go`, `backend/ent/userattributevalue/userattributevalue.go`.
- The failed stderr contains the generated path `announcementread.go` with the exact `user-mapped section open` signature, followed only by `exit status 1`, `ent\generate.go:6: running "go": exit status 1`, and `make: *** [generate] Error 1` wrappers. Fix-2 restored the generated paths, waited two seconds, and retry attempt 2 completed with an empty diff path list; the current generated-path diff check above is also empty.

### Superseded Historical Conclusion And Scope

- The old statement at ledger line 738 that assigned row 13 to Task 5 is superseded. Task 5 only owns row 10 Docker quota/outbox/migration integration.
- Row 13 deploy scripts are exclusively Task 13 work after the v0.1.169 merge. The current row 13 non-Docker test evidence is the explicit Fix-2 historical reference above.
- Current remaining risk signal: the historic Windows user-mapped-section failure was transient and recovered under the bounded Fix-2 rules. Existing frontend warning output remains non-failing. Docker row 10 remains outside Task 4 and is not represented as complete here.

## Task 5 Docker/Testcontainers Row 10 Determination

- 实施时间：2026-08-02。
- 执行基线：`63da8b61addc8717f0436e36a8e76ad7406ac6b4`。
- 结论：row 10 为 `unverified`，不是 `protected` 或 PASS。Docker CLI 在本机不可用，因此未启动 Testcontainers integration；没有执行 SSH、远程验证、镜像构建或部署。

### Docker Preflight Evidence

```powershell
$log = Join-Path $env:TEMP 'sub2api-stage0-docker-preflight.log'
$dockerCommand = Get-Command docker -ErrorAction SilentlyContinue
if ($null -eq $dockerCommand) { 'docker_command=unavailable' | Set-Content -LiteralPath $log; $preflightAvailable = $false } else { & $dockerCommand.Path version --format '{{.Server.Version}}' 2>&1 | Tee-Object -FilePath $log; $preflightExitCode = $LASTEXITCODE; "exit_code=$preflightExitCode" | Add-Content -LiteralPath $log; $preflightAvailable = ($preflightExitCode -eq 0) }
```

- 日志：`C:\Users\caiqy\AppData\Local\Temp\sub2api-stage0-docker-preflight.log`。
- 完整日志：`docker_command=unavailable`。
- Docker process exit code：N/A；命令不存在，未读取陈旧 `$LASTEXITCODE`。预检结论：`preflightAvailable=false`。

### Row 10 Integration Status

| Target | 状态 | 证据 |
| --- | --- | --- |
| `TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce` | `unverified`（未执行） | Docker preflight unavailable；未启动 integration。 |
| `TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails` | `unverified`（未执行） | Docker preflight unavailable；未启动 integration。 |
| `TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack` | `unverified`（未执行） | Docker preflight unavailable；未启动 integration。 |
| `TestSubscriptionCacheInvalidationMigration_RawRerunIsIdempotent` | `unverified`（未执行） | Docker preflight unavailable；未启动 integration。 |

- 未执行的精确 integration 命令：`go test -tags=integration -v -count=1 -run '^(TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce|TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails|TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack|TestSubscriptionCacheInvalidationMigration_RawRerunIsIdempotent)$' ./internal/repository`（应从 `backend` 运行）。
- 未验证契约：PostgreSQL transaction/lock 的并发单次扣减、receipt 写入失败时 subscription rollback、outbox semantic change/rollback、migration raw rerun idempotency。
- 残余风险：本机尚无直接 Docker/Testcontainers 证据，row 10 不能作为 release PASS；Docker 可用的环境必须重跑上述四个精确目标，并验证每一个锚定的顶级 `--- PASS: TestName (` 输出。

### Matrix Closure And Self-review

| # | 状态 | 当前直接证据 |
| --- | --- | --- |
| 10 | `unverified` | Docker CLI unavailable；四个 quota/receipt/outbox/migration Testcontainers targets 未执行。 |

统计：`protected=13`，`manual=0`，`gap=0`，`unverified=1`。

- 自审：本次 tracked diff 仅为本 ledger；未修改 Plan、OpenSpec、`tasks.md`、progress、应用源码或配置。
- 自审：未执行 fetch、merge、push、tag、release、deploy、server/SSH、镜像构建；row 13 deploy scripts 未纳入本任务。

## Task 6 v0.1.166 Merge Review

- Review date: `2026-08-02`.
- Merge commit: `c7ae76df77755b5b84b26b91606d37efc13b5deb` (`Merge tag 'v0.1.166' into feature/20260802/staged-merge-upstream-v0-1-169`).
- Parents: first `e9d2ce48e23391f12a255ca9430d3f16bfd7fea3`; second `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` (`v0.1.166^{}`).
- Preflight permitted only `?? .comet/current-change.json`; the index was empty. The sole `git merge --no-ff --no-commit v0.1.166` produced exactly the 17 actual conflicts below, with no extras.

### Six-Class Conflict Ledger

| Class | Path | Ours retained | Theirs retained | Fusion and evidence |
| --- | --- | --- | --- | --- |
| Version/dependency | `backend/cmd/server/VERSION` | `0.1.165.4` release value | Upstream merge content | Result remains `0.1.165.4`; version is checked in the merge tree. |
| Version/dependency | `backend/go.sum` | Merged module graph | Updated upstream checksums | Regenerated by `go mod tidy` (exit `0`). |
| Config/API evolution | `backend/internal/config/config_test.go` | Local config assertions | New upstream config fields | Additive field/assertion coverage is retained. |
| Config/API evolution | `backend/internal/handler/admin/setting_handler_update.go` | Hot settings update | Partial-update semantics | Retains both, preventing omitted fields from being cleared. |
| HTTP gateway/Responses | `backend/internal/handler/gateway_handler_responses.go` | Local Responses route, audit, scheduler | Upstream Responses behavior | Merged request path keeps local entry controls. |
| HTTP gateway/Responses | `backend/internal/handler/openai_gateway_handler.go` | HTTP gateway, usage, failover, audit | Upstream handler changes | Local accounting/audit path remains before forwarding. |
| Gateway test coverage | `backend/internal/handler/openai_gateway_handler_test.go` | WS regression helpers/tests | Multi-turn usage/failover tests | Rebuilt as additive top-level Go declarations; handler/service compile check passed. |
| Config/API evolution | `backend/internal/server/router.go` | Middleware/audit registration | Panel API rate-limit routes | Upstream routes register without bypassing local middleware/audit. |
| HTTP gateway/Responses | `backend/internal/service/gateway_forward_as_responses.go` | Prompt cache and request-body replay | Upstream Responses forwarding | Both forwarding and replay behavior remain in one path. |
| Gateway test coverage | `backend/internal/service/gateway_forward_as_responses_test.go` | Cache usage assertions | Upstream forwarding-field assertions | Additive regression coverage retained. |
| Gateway test coverage | `backend/internal/service/gateway_record_usage_test.go` | Final model, failed usage, billing context | Upstream usage fixes | Targeted failed-usage upstream-model unit test passed. |
| Compatibility/WS lifecycle | `backend/internal/service/gemini_chat_completions_compat_service.go` | Compatibility mappings | Upstream compatibility fixes | Mapping contract and upstream additions coexist. |
| Gateway test coverage | `backend/internal/service/ollama_cloud_usage_test.go` | Ollama usage ownership | Upstream test updates | Ownership assertions remain. |
| Compatibility/WS lifecycle | `backend/internal/service/openai_ws_forwarder.go` | Per-turn model, terminal ownership, failed usage, prompt-cache reuse | Upstream WS fixes | Per-turn accounting and lifecycle remain preserved. |
| Compatibility/WS lifecycle | `backend/internal/service/openai_ws_v2/passthrough_relay.go` | Relay turn lifecycle/final model | Upstream relay changes | Terminal lifecycle and final-model mapping remain. |
| Compatibility/WS lifecycle | `backend/internal/service/openai_ws_v2_passthrough_adapter.go` | Adapter usage/model mapping | Upstream adapter changes | Usage attribution remains mapped to the final upstream model. |
| Frontend contract | `frontend/src/views/admin/__tests__/UsageView.spec.ts` | Local usage-page assertions | Upstream UI behavior | Assertions are additive; lockfile regenerated by frontend package-manager command (exit `0`). |

### Blocking Review And Verification

- Blocking conclusion: no unresolved semantic incompatibility was found among panel API rate limiting, settings partial update, effective composite route, account failover, audit/usage entry points, per-turn model billing, terminal ownership, or local quota reset behavior.
- `git diff --name-only --diff-filter=U` was empty before the merge commit; `git diff --cached --check` passed.
- `go mod tidy` exit `0`; `GOFLAGS=-mod=mod make generate` exit `0`; `pnpm --dir frontend install --lockfile-only --frozen-lockfile=false` exit `0`.
- `go test ./internal/handler ./internal/service -run '^$'` exit `0`.
- `go test -tags unit ./internal/service -run '^TestWriteFailedUsageLogBestEffort_OmitsMatchingNonWSUpstreamModel$'` exit `0`.
- The broad conflict-marker scan's only `=======` result was the established literal delimiter at `backend/internal/pkg/antigravity/request_transformer.go:267`; it was reviewed and is not a merge marker.
- Task 7's full behavior suite was not run under Task 6 constraints. Focused compile and usage checks reduce merge risk but do not replace that planned suite.
