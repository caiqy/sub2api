# v0.1.171 Build Ledger

## 阶段 0：基线

- 执行位置：`D:/Caiqy/Projects/Github/sub2api`
- 执行分支：`feature/20260806/staged-merge-upstream-v0-1-171`
- immutable source base：`b576f73a22c4bf23d61727fc93950766a7e33929`
- execution base：`b576f73a22c4bf23d61727fc93950766a7e33929`
- source `VERSION`：`0.1.169.3`
- execution `VERSION`：`0.1.169.3`
- source-to-execution 路径：无
- runtime selection 状态：`git status --short --untracked-files=all` 输出为空；未出现 `?? .comet/current-change.json`，且无其他 dirty path。

### 禁止操作

- 不切换分支、merge、push、tag、release、GitHub Actions、镜像构建或发布。
- 不部署，不操作服务器、数据库、Redis 或 Nginx。
- 不修改应用源码、plan、OpenSpec tasks、`.comet/**` 或 `.comet/current-change.json`。

### Task 1 命令与退出码

```text
comet classic root show                                      exit 0
layout.schema=comet.classic-layout.v1

git rev-parse --show-toplevel                               exit 0
D:/Caiqy/Projects/Github/sub2api

git branch --show-current                                   exit 0
feature/20260806/staged-merge-upstream-v0-1-171

git merge-base --is-ancestor b576f73a22c4bf23d61727fc93950766a7e33929 HEAD
exit 0

git rev-parse HEAD                                          exit 0
b576f73a22c4bf23d61727fc93950766a7e33929

git show b576f73a22c4bf23d61727fc93950766a7e33929:backend/cmd/server/VERSION
exit 0
0.1.169.3

git show b576f73a22c4bf23d61727fc93950766a7e33929:backend/cmd/server/VERSION
exit 0
0.1.169.3

git log -m --format= --name-only b576f73a22c4bf23d61727fc93950766a7e33929..b576f73a22c4bf23d61727fc93950766a7e33929
exit 0
(no paths)

Assert-CleanGate                                            exit 0
staged paths: (none)
status: (empty)
unexpected ignored change artifacts: (none)
```

### TDD

不适用。本任务只创建基线证据文档，不包含生产代码或行为变更；未伪造 RED/GREEN。

### 阶段结论

基线身份和隔离状态均通过；可由协调者决定是否推进后续 Task。

## Task 2：上游 tag manifest

- refs 刷新：`git fetch upstream --tags --prune` 成功；`upstream/main` 从 `00b859617` 前进到 `c123caddd`。
- `v0.1.170^{}`：`c043c24774228ba891ddf90d783aa6dc7d0855b5`，与固定 peeled SHA 一致。
- `v0.1.171^{}`：`f0e7a9c7a23a7d02fb159b62fa809621eb0475a6`，与固定 peeled SHA 一致。
- 严格祖先链：`git merge-base --is-ancestor v0.1.170 v0.1.171` exit 0。
- merged `upstream/main` 的正式 tag 筛选为 `^v0\.1\.\d+$`；降序首项仍为 `v0.1.171`，未发现更高正式 tag。
- 冻结范围预期规模：`v0.1.169..v0.1.170` 为 `62/242`；`v0.1.170..v0.1.171` 为 `49/206`。

### Task 2 命令与退出码

```text
comet classic root show                                      exit 0
layout.schema=comet.classic-layout.v1

git rev-parse --show-toplevel                               exit 0
D:/Caiqy/Projects/Github/sub2api

git branch --show-current                                   exit 0
feature/20260806/staged-merge-upstream-v0-1-171

Assert-CleanGate                                            exit 0
git diff --cached --name-only                               exit 0
git status --short --untracked-files=all                    exit 0
git ls-files --others --ignored --exclude-standard -- docs/openspec/changes/staged-merge-upstream-v0-1-171 docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md docs/superpowers/specs/2026-08-06-staged-merge-upstream-v0-1-171-design.md docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-build.md docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-verify.md
exit 0

git fetch upstream --tags --prune                           exit 0
upstream/main: 00b859617 -> c123caddd

git rev-parse v0.1.170^{}                                   exit 0
c043c24774228ba891ddf90d783aa6dc7d0855b5

git rev-parse v0.1.171^{}                                   exit 0
f0e7a9c7a23a7d02fb159b62fa809621eb0475a6

git merge-base --is-ancestor v0.1.170 v0.1.171              exit 0

git for-each-ref refs/tags --merged=upstream/main --format='%(refname:short)' | Where-Object { $_ -match '^v0\.1\.\d+$' } | Sort-Object { [version]$_.Substring(1) } -Descending
exit 0
highest formal v0.1.* tag: v0.1.171

git log --oneline v0.1.171..upstream/main                   exit 0
```

### Task 2 范围外提交

以下提交均在 `v0.1.171..upstream/main`，只记录为范围外，不在当前变更中 merge：

```text
c123caddd chore: update sponsors
e08aee49e Merge pull request #5266 from shentry/fix/transient-streak-rate-dependence
c9e60d1f2 Merge pull request #5031 from keaipiao/fix/easypay-error-utf8
47c03c75d Merge pull request #5232 from fengshao1227/fix/billing-quantize-monetary-scale
00b859617 chore: update sponsors
c5e046b7d chore: update sponsors
aac53afe0 chore: sync VERSION to 0.1.171 [skip ci]
7d38e6712 fix(openai): keep transient failure streak from resetting on sparse traffic
e2652eb85 fix(billing): quantize usage billing amounts to the NUMERIC(20,8) scale
e3e033bb3 fix(payment): preserve UTF-8 in EasyPay errors
```

### Task 2 TDD 与风险

- TDD 不适用。本任务只刷新 Git refs 并更新证据文档，不包含生产代码或行为变更；未伪造 RED/GREEN，也未运行应用测试。
- 风险信号：`upstream/main` 已有 10 个 tag 后提交；它们已完整列为范围外，且没有更高正式 tag 改变当前范围。
- 顾虑：正式 tag 与祖先链在本次 fetch 后稳定；后续 Task 仍必须只通过固定的两个 tag merge 入口引入上游内容。

## Task 3：changed-files 能力矩阵和冲突台账

- Task 3 执行 base：`e95ef3f021f5b1ffe6b4a351dc00ce87a710b1b7`。
- 矩阵输入只使用下列两个 Git tag 区间的完整 changed-files；未以包级测试成功替代范围审查。
- 初始状态均为 `gap`：本阶段只完成范围、入口和后续审查点登记，尚未直接证明任何行为契约。

### changed-files 命令与计数

```text
git diff --name-only v0.1.169..v0.1.170    exit 0    count 242
git diff --name-only v0.1.170..v0.1.171    exit 0    count 206
```

### `v0.1.169..v0.1.170`（242 files）

```text
README.md
README_CN.md
README_JA.md
assets/partners/logos/666api.jpg
assets/partners/logos/fennoai.jpg
assets/partners/logos/lanox.jpg
assets/partners/logos/novada.png
assets/partners/logos/qiniu.jpg
backend/cmd/profit-preview/main.go
backend/cmd/profit-preview/main_test.go
backend/cmd/server/VERSION
backend/cmd/server/wire_gen.go
backend/ent/client.go
backend/ent/group.go
backend/ent/group/group.go
backend/ent/group/where.go
backend/ent/group_create.go
backend/ent/group_update.go
backend/ent/migrate/schema.go
backend/ent/mutation.go
backend/ent/runtime/runtime.go
backend/ent/schema/group.go
backend/internal/handler/admin/account_handler.go
backend/internal/handler/admin/account_handler_batch_delete_test.go
backend/internal/handler/admin/account_handler_mixed_channel_test.go
backend/internal/handler/admin/admin_service_stub_test.go
backend/internal/handler/admin/content_moderation_handler.go
backend/internal/handler/admin/group_handler.go
backend/internal/handler/admin/setting_handler.go
backend/internal/handler/admin/setting_handler_audit.go
backend/internal/handler/admin/setting_handler_platform_quota_test.go
backend/internal/handler/admin/setting_handler_update.go
backend/internal/handler/dto/group_mapper_profit_visibility_test.go
backend/internal/handler/dto/mappers.go
backend/internal/handler/dto/settings.go
backend/internal/handler/dto/types.go
backend/internal/handler/failover_loop.go
backend/internal/handler/failover_loop_profit_veto_test.go
backend/internal/handler/gateway_handler.go
backend/internal/handler/gateway_handler_chat_completions.go
backend/internal/handler/gateway_handler_responses.go
backend/internal/handler/gemini_v1beta_handler.go
backend/internal/handler/grok_media.go
backend/internal/handler/openai_alpha_search.go
backend/internal/handler/openai_chat_completions.go
backend/internal/handler/openai_embeddings.go
backend/internal/handler/openai_gateway_count_tokens.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/openai_gateway_handler_test.go
backend/internal/handler/openai_images.go
backend/internal/handler/openai_profit_slot_recheck_test.go
backend/internal/handler/openai_profit_veto_budget_test.go
backend/internal/handler/openai_ws_turn_pricing_test.go
backend/internal/handler/setting_handler.go
backend/internal/handler/usage_record_task_fallback_test.go
backend/internal/pkg/apicompat/chatcompletions_responses_bridge.go
backend/internal/pkg/apicompat/chatcompletions_responses_tool_output_media_test.go
backend/internal/repository/account_repo.go
backend/internal/repository/account_repo_ollama_cloud_usage_integration_test.go
backend/internal/repository/account_repo_upstream_billing_probe_cas_test.go
backend/internal/repository/account_repo_upstream_billing_probe_due_integration_test.go
backend/internal/repository/account_repo_upstream_billing_probe_due_test.go
backend/internal/repository/account_repo_upstream_billing_probe_update_test.go
backend/internal/repository/api_key_repo.go
backend/internal/repository/api_key_repo_profit_projection_integration_test.go
backend/internal/repository/auth_cache_invalidation_profit_integration_test.go
backend/internal/repository/fixtures_integration_test.go
backend/internal/repository/gateway_cache.go
backend/internal/repository/gateway_cache_integration_test.go
backend/internal/repository/group_repo.go
backend/internal/repository/proxy_repo.go
backend/internal/repository/proxy_repo_upstream_billing_probe_test.go
backend/internal/repository/scheduler_cache_rate_multiplier_unit_test.go
backend/internal/repository/upstream_billing_probe_persistence_integration_test.go
backend/internal/repository/user_subscription_repo.go
backend/internal/repository/user_subscription_repo_integration_test.go
backend/internal/securityaudit/prompt_config.go
backend/internal/securityaudit/prompt_config_store.go
backend/internal/securityaudit/prompt_config_test.go
backend/internal/securityaudit/prompt_handler.go
backend/internal/securityaudit/prompt_service.go
backend/internal/securityaudit/prompt_service_test.go
backend/internal/securityaudit/prompt_snapshot.go
backend/internal/securityaudit/prompt_snapshot_test.go
backend/internal/server/api_contract_test.go
backend/internal/server/routes/admin.go
backend/internal/service/account.go
backend/internal/service/account_service.go
backend/internal/service/admin_account.go
backend/internal/service/admin_account_upstream_billing_probe_test.go
backend/internal/service/admin_group.go
backend/internal/service/admin_group_duplicate.go
backend/internal/service/admin_service.go
backend/internal/service/admin_service_bulk_update_test.go
backend/internal/service/admin_service_duplicate_account_test.go
backend/internal/service/api_key_auth_cache.go
backend/internal/service/api_key_auth_cache_impl.go
backend/internal/service/api_key_auth_cache_profit_test.go
backend/internal/service/claude_code_validator.go
backend/internal/service/claude_code_validator_test.go
backend/internal/service/content_moderation.go
backend/internal/service/content_moderation_cyber_test.go
backend/internal/service/content_moderation_proxy_test.go
backend/internal/service/content_moderation_test.go
backend/internal/service/crs_sync_helpers_test.go
backend/internal/service/crs_sync_service.go
backend/internal/service/domain_constants.go
backend/internal/service/email_service.go
backend/internal/service/email_service_smtp_test.go
backend/internal/service/gateway_anthropic_passthrough.go
backend/internal/service/gateway_forward.go
backend/internal/service/gateway_forward_partial_usage_test.go
backend/internal/service/gateway_profit_control.go
backend/internal/service/gateway_profit_control_v2_test.go
backend/internal/service/gateway_record_usage_test.go
backend/internal/service/gateway_request_pricing.go
backend/internal/service/gateway_request_pricing_test.go
backend/internal/service/gateway_scheduling.go
backend/internal/service/gateway_service.go
backend/internal/service/gateway_upstream_response.go
backend/internal/service/gateway_usage_billing.go
backend/internal/service/grok_upstream_errors_test.go
backend/internal/service/group.go
backend/internal/service/group_profit_platform_test.go
backend/internal/service/image_storage.go
backend/internal/service/image_storage_test.go
backend/internal/service/openai_account_scheduler.go
backend/internal/service/openai_account_scheduler_test.go
backend/internal/service/openai_account_scheduler_upstream_cost_test.go
backend/internal/service/openai_compact_sse_keepalive_test.go
backend/internal/service/openai_gateway_chat_completions.go
backend/internal/service/openai_gateway_forward.go
backend/internal/service/openai_gateway_grok.go
backend/internal/service/openai_gateway_grok_sse_filter.go
backend/internal/service/openai_gateway_grok_sse_filter_test.go
backend/internal/service/openai_gateway_grok_test.go
backend/internal/service/openai_gateway_messages.go
backend/internal/service/openai_gateway_passthrough.go
backend/internal/service/openai_gateway_request_body.go
backend/internal/service/openai_gateway_request_body_reasoning_test.go
backend/internal/service/openai_gateway_response_handling.go
backend/internal/service/openai_gateway_scheduling.go
backend/internal/service/openai_gateway_service_hotpath_test.go
backend/internal/service/openai_gateway_service_test.go
backend/internal/service/openai_gateway_usage.go
backend/internal/service/openai_live.go
backend/internal/service/openai_oauth_passthrough_test.go
backend/internal/service/openai_passthrough_normalization_test.go
backend/internal/service/openai_profit_control.go
backend/internal/service/openai_profit_control_paths_test.go
backend/internal/service/openai_profit_control_pricing_test.go
backend/internal/service/openai_profit_control_test.go
backend/internal/service/openai_responses_namespace.go
backend/internal/service/openai_responses_namespace_forward_test.go
backend/internal/service/openai_responses_namespace_test.go
backend/internal/service/openai_ws_forwarder_ingress.go
backend/internal/service/openai_ws_forwarder_support.go
backend/internal/service/openai_ws_passthrough_turn_pricing_test.go
backend/internal/service/openai_ws_protocol_forward_test.go
backend/internal/service/openai_ws_v2/passthrough_relay.go
backend/internal/service/payment_config_service.go
backend/internal/service/payment_config_service_test.go
backend/internal/service/pricing_service_test.go
backend/internal/service/profit_control_threshold_helpers_test.go
backend/internal/service/profit_preview.go
backend/internal/service/profit_preview_test.go
backend/internal/service/scheduler_snapshot_service.go
backend/internal/service/setting_parse.go
backend/internal/service/setting_public.go
backend/internal/service/setting_service_public_test.go
backend/internal/service/setting_service_update_test.go
backend/internal/service/setting_update.go
backend/internal/service/settings_view.go
backend/internal/service/subscription_assign_idempotency_test.go
backend/internal/service/subscription_monthly_window_test.go
backend/internal/service/subscription_reset_quota_test.go
backend/internal/service/subscription_service.go
backend/internal/service/upstream_billing_probe.go
backend/internal/service/upstream_billing_probe_multiplatform_test.go
backend/internal/service/upstream_billing_probe_test.go
backend/internal/service/usage_record_worker_pool.go
backend/internal/service/usage_record_worker_pool_test.go
backend/internal/service/user_subscription.go
backend/internal/service/user_subscription_daily_quota_test.go
backend/migrations/192_group_profit_control.sql
backend/migrations/193_group_profit_control_auth_cache_invalidation.sql
backend/resources/model-pricing/model_prices_and_context_window.json
frontend/src/api/admin/accounts.ts
frontend/src/api/admin/riskControl.ts
frontend/src/api/admin/settings.ts
frontend/src/components/account/BulkEditAccountModal.vue
frontend/src/components/account/CreateAccountModal.vue
frontend/src/components/account/EditAccountModal.vue
frontend/src/components/account/UpstreamBillingRateCell.vue
frontend/src/components/account/__tests__/BulkEditAccountModal.spec.ts
frontend/src/components/account/__tests__/CreateAccountModal.spec.ts
frontend/src/components/account/__tests__/EditAccountModal.spec.ts
frontend/src/components/account/__tests__/UpstreamBillingRateCell.spec.ts
frontend/src/components/admin/account/AccountBulkActionsBar.vue
frontend/src/components/admin/account/__tests__/AccountBulkActionsBar.spec.ts
frontend/src/components/auth/__tests__/WechatOAuthSection.spec.ts
frontend/src/components/modelPlaza/PlazaFilterBar.vue
frontend/src/components/modelPlaza/PlazaGroupSection.vue
frontend/src/components/modelPlaza/PlazaModelPricingTable.vue
frontend/src/components/modelPlaza/__tests__/PlazaModelPricingTable.spec.ts
frontend/src/components/payment/PaymentMethodSelector.vue
frontend/src/components/payment/__tests__/PaymentMethodSelector.spec.ts
frontend/src/components/user/profile/__tests__/ProfileIdentityBindingsSection.spec.ts
frontend/src/features/prompt-audit/PromptAuditView.vue
frontend/src/features/prompt-audit/__tests__/PromptAuditView.spec.ts
frontend/src/features/prompt-audit/__tests__/components.spec.ts
frontend/src/features/prompt-audit/__tests__/viewModel.spec.ts
frontend/src/features/prompt-audit/types.ts
frontend/src/features/prompt-audit/viewModel.ts
frontend/src/i18n/locales/en/admin/accounts.ts
frontend/src/i18n/locales/en/admin/channels.ts
frontend/src/i18n/locales/en/admin/overview.ts
frontend/src/i18n/locales/en/admin/promptAudit.ts
frontend/src/i18n/locales/en/admin/settings.ts
frontend/src/i18n/locales/zh/admin/accounts.ts
frontend/src/i18n/locales/zh/admin/channels.ts
frontend/src/i18n/locales/zh/admin/overview.ts
frontend/src/i18n/locales/zh/admin/promptAudit.ts
frontend/src/i18n/locales/zh/admin/settings.ts
frontend/src/stores/__tests__/app.spec.ts
frontend/src/stores/app.ts
frontend/src/types/index.ts
frontend/src/utils/__tests__/accountSelection.spec.ts
frontend/src/utils/accountSelection.ts
frontend/src/views/HomeView.vue
frontend/src/views/__tests__/HomeView.compact.spec.ts
frontend/src/views/admin/AccountsView.vue
frontend/src/views/admin/GroupsView.vue
frontend/src/views/admin/RiskControlView.vue
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts
frontend/src/views/admin/__tests__/AccountsView.selectAllResults.spec.ts
frontend/src/views/admin/__tests__/AccountsView.usageWindowsHint.spec.ts
frontend/src/views/admin/__tests__/RiskControlView.spec.ts
frontend/src/views/admin/__tests__/SettingsView.spec.ts
frontend/src/views/admin/__tests__/groupsProfitControl.spec.ts
frontend/src/views/admin/groupsProfitControl.ts
```

### `v0.1.170..v0.1.171`（206 files）

```text
Makefile
README.md
backend/cmd/server/VERSION
backend/cmd/server/wire.go
backend/cmd/server/wire_gen.go
backend/cmd/server/wire_gen_test.go
backend/go.mod
backend/go.sum
backend/internal/config/config.go
backend/internal/handler/admin/account_handler_long_context_billing_test.go
backend/internal/handler/admin/openai_oauth_handler.go
backend/internal/handler/admin/openai_oauth_handler_reset_quota_test.go
backend/internal/handler/admin/openai_oauth_handler_spark_shadow_test.go
backend/internal/handler/admin/setting_handler.go
backend/internal/handler/admin/setting_handler_audit.go
backend/internal/handler/admin/setting_handler_partial_payload_test.go
backend/internal/handler/admin/setting_handler_platform_quota_test.go
backend/internal/handler/admin/setting_handler_update.go
backend/internal/handler/auth_captcha_request_test.go
backend/internal/handler/auth_dingtalk_oauth.go
backend/internal/handler/auth_email_oauth.go
backend/internal/handler/auth_handler.go
backend/internal/handler/auth_linuxdo_oauth.go
backend/internal/handler/auth_oauth_captcha_start.go
backend/internal/handler/auth_oauth_captcha_start_test.go
backend/internal/handler/auth_oauth_pending_flow.go
backend/internal/handler/auth_oidc_oauth.go
backend/internal/handler/auth_wechat_oauth.go
backend/internal/handler/composite_platform.go
backend/internal/handler/composite_platform_test.go
backend/internal/handler/dto/settings.go
backend/internal/handler/model_plaza_handler.go
backend/internal/handler/model_plaza_handler_test.go
backend/internal/handler/openai_chat_completions.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/passkey_handler.go
backend/internal/handler/passkey_handler_test.go
backend/internal/handler/setting_handler.go
backend/internal/handler/setting_handler_public_test.go
backend/internal/handler/wire.go
backend/internal/payment/provider/stripe.go
backend/internal/payment/provider/stripe_test.go
backend/internal/pkg/oauth/oauth.go
backend/internal/pkg/oauth/oauth_test.go
backend/internal/pkg/openai/request.go
backend/internal/pkg/openai/request_codex_version_test.go
backend/internal/pkg/usagestats/usage_log_types.go
backend/internal/pkg/xai/billing.go
backend/internal/repository/account_repo.go
backend/internal/repository/aliyun_captcha_verifier.go
backend/internal/repository/aliyun_captcha_verifier_test.go
backend/internal/repository/http_upstream.go
backend/internal/repository/http_upstream_test.go
backend/internal/repository/tencent_captcha_service.go
backend/internal/repository/tencent_captcha_service_test.go
backend/internal/repository/usage_log_repo_request_type_test.go
backend/internal/repository/usage_log_repo_trend.go
backend/internal/repository/user_repo.go
backend/internal/repository/user_repo_integration_test.go
backend/internal/repository/user_subscription_lock_test.go
backend/internal/repository/user_subscription_repo.go
backend/internal/repository/wire.go
backend/internal/securityaudit/prompt_snapshot.go
backend/internal/securityaudit/prompt_snapshot_test.go
backend/internal/server/api_contract_test.go
backend/internal/server/middleware/api_key_auth_google_test.go
backend/internal/server/middleware/api_key_auth_test.go
backend/internal/server/middleware/security_headers.go
backend/internal/server/middleware/security_headers_test.go
backend/internal/server/routes/admin.go
backend/internal/server/routes/auth.go
backend/internal/service/account_test_service.go
backend/internal/service/account_usage_service.go
backend/internal/service/admin_service_composite_group_test.go
backend/internal/service/aliyun_captcha_service.go
backend/internal/service/aliyun_captcha_service_test.go
backend/internal/service/api_key_service_cache_test.go
backend/internal/service/auth_service.go
backend/internal/service/auth_service_captcha_test.go
backend/internal/service/auth_service_register_test.go
backend/internal/service/channel_plaza.go
backend/internal/service/channel_plaza_test.go
backend/internal/service/domain_constants.go
backend/internal/service/gateway_record_usage_test.go
backend/internal/service/gateway_service.go
backend/internal/service/gateway_usage_billing.go
backend/internal/service/grok_upstream_errors_test.go
backend/internal/service/openai_alpha_search.go
backend/internal/service/openai_alpha_search_test.go
backend/internal/service/openai_capacity_shed_test.go
backend/internal/service/openai_codex_identity.go
backend/internal/service/openai_codex_identity_test.go
backend/internal/service/openai_codex_version_sync_service.go
backend/internal/service/openai_codex_version_sync_service_test.go
backend/internal/service/openai_compat_model_test.go
backend/internal/service/openai_gateway_cc_pipeline.go
backend/internal/service/openai_gateway_forward.go
backend/internal/service/openai_gateway_grok.go
backend/internal/service/openai_gateway_messages.go
backend/internal/service/openai_gateway_messages_chat_fallback.go
backend/internal/service/openai_gateway_messages_chat_fallback_test.go
backend/internal/service/openai_gateway_passthrough.go
backend/internal/service/openai_gateway_record_usage_test.go
backend/internal/service/openai_gateway_service.go
backend/internal/service/openai_gateway_service_test.go
backend/internal/service/openai_gateway_usage.go
backend/internal/service/openai_oauth_passthrough_test.go
backend/internal/service/openai_quota_service.go
backend/internal/service/openai_quota_spark_window_test.go
backend/internal/service/openai_reasoning_effort_policy.go
backend/internal/service/openai_reasoning_effort_policy_test.go
backend/internal/service/openai_ws_forwarder.go
backend/internal/service/openai_ws_forwarder_ingress.go
backend/internal/service/openai_ws_forwarder_ingress_session_test.go
backend/internal/service/openai_ws_forwarder_payload.go
backend/internal/service/openai_ws_forwarder_success_test.go
backend/internal/service/payment_refund.go
backend/internal/service/payment_refund_test.go
backend/internal/service/scheduler_snapshot_cancellation_test.go
backend/internal/service/scheduler_snapshot_service.go
backend/internal/service/setting_features.go
backend/internal/service/setting_gateway_runtime.go
backend/internal/service/setting_parse.go
backend/internal/service/setting_public.go
backend/internal/service/setting_service.go
backend/internal/service/setting_service_public_test.go
backend/internal/service/setting_update.go
backend/internal/service/settings_view.go
backend/internal/service/subscription_assign_idempotency_test.go
backend/internal/service/subscription_expiry_service_test.go
backend/internal/service/subscription_renewal_lock_test.go
backend/internal/service/subscription_service.go
backend/internal/service/tencent_captcha_service.go
backend/internal/service/tencent_captcha_service_test.go
backend/internal/service/tencent_captcha_settings_test.go
backend/internal/service/turnstile_service.go
backend/internal/service/user_service_test.go
backend/internal/service/user_subscription_port.go
backend/internal/service/wire.go
deploy/config.example.yaml
frontend/src/api/__tests__/auth-captcha-oauth-start.spec.ts
frontend/src/api/__tests__/client.spec.ts
frontend/src/api/__tests__/passkey.spec.ts
frontend/src/api/__tests__/tokenRefresh.spec.ts
frontend/src/api/admin/accounts.ts
frontend/src/api/admin/settings.ts
frontend/src/api/auth.ts
frontend/src/api/client.ts
frontend/src/api/modelPlaza.ts
frontend/src/api/passkey.ts
frontend/src/api/tokenRefresh.ts
frontend/src/components/AliyunCaptchaWidget.vue
frontend/src/components/CaptchaChallenge.vue
frontend/src/components/TencentCaptchaGate.vue
frontend/src/components/__tests__/AliyunCaptchaWidget.spec.ts
frontend/src/components/__tests__/TencentCaptchaGate.spec.ts
frontend/src/components/account/AccountUsageCell.vue
frontend/src/components/account/OpenAIQuotaResetCell.vue
frontend/src/components/account/__tests__/AccountUsageCell.spec.ts
frontend/src/components/account/__tests__/OpenAIQuotaResetCell.spark_shadow.spec.ts
frontend/src/components/auth/DingTalkOAuthSection.vue
frontend/src/components/auth/EmailOAuthButtons.vue
frontend/src/components/auth/LinuxDoOAuthSection.vue
frontend/src/components/auth/OidcOAuthSection.vue
frontend/src/components/auth/PendingOAuthCreateAccountForm.vue
frontend/src/components/auth/WechatOAuthSection.vue
frontend/src/components/auth/__tests__/EmailOAuthButtons.spec.ts
frontend/src/components/auth/__tests__/OAuthLoginSections.spec.ts
frontend/src/components/auth/__tests__/PendingOAuthCreateAccountForm.spec.ts
frontend/src/components/auth/__tests__/WechatOAuthSection.spec.ts
frontend/src/components/charts/ModelDistributionChart.vue
frontend/src/components/charts/__tests__/ModelDistributionChart.spec.ts
frontend/src/components/modelPlaza/PlazaGroupSection.vue
frontend/src/components/modelPlaza/PlazaModelPricingTable.vue
frontend/src/components/modelPlaza/__tests__/PlazaModelPricingTable.spec.ts
frontend/src/i18n/locales/en/admin/accounts.ts
frontend/src/i18n/locales/en/admin/overview.ts
frontend/src/i18n/locales/en/admin/settings.ts
frontend/src/i18n/locales/en/common.ts
frontend/src/i18n/locales/zh/admin/accounts.ts
frontend/src/i18n/locales/zh/admin/overview.ts
frontend/src/i18n/locales/zh/admin/settings.ts
frontend/src/i18n/locales/zh/common.ts
frontend/src/stores/app.ts
frontend/src/stores/auth.ts
frontend/src/types/index.ts
frontend/src/utils/tencentCaptcha.ts
frontend/src/views/admin/AccountsView.vue
frontend/src/views/admin/GroupsView.vue
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/__tests__/SettingsView.spec.ts
frontend/src/views/admin/__tests__/groupsReasoningEffort.spec.ts
frontend/src/views/admin/groupsReasoningEffort.ts
frontend/src/views/admin/orders/AdminOrdersView.vue
frontend/src/views/auth/DingTalkCallbackView.vue
frontend/src/views/auth/DingTalkEmailCompletionView.vue
frontend/src/views/auth/EmailVerifyView.vue
frontend/src/views/auth/ForgotPasswordView.vue
frontend/src/views/auth/LinuxDoCallbackView.vue
frontend/src/views/auth/LoginView.vue
frontend/src/views/auth/OidcCallbackView.vue
frontend/src/views/auth/RegisterView.vue
frontend/src/views/auth/WechatCallbackView.vue
frontend/src/views/auth/__tests__/EmailVerifyView.spec.ts
frontend/src/views/auth/__tests__/TencentCaptchaActionGate.spec.ts
frontend/src/views/auth/__tests__/WechatCallbackView.spec.ts
```

### 能力矩阵（初始）

| # | 行为契约 | 入口/调用链 | 关键文件 | 受影响 tag | 聚焦命令或人工审查点 | 初始状态 | 证据位置 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | advanced/layered scheduler、pool mode、DB recheck、WaitPlan fallback、同账号重试 | `OpenAIGatewayHandler` -> `OpenAIGatewayService` -> `OpenAIAccountScheduler.Select` -> sticky/session -> layered filter -> slot/`WaitPlan` | `backend/internal/service/openai_account_scheduler.go`、`backend/internal/service/openai_account_scheduler_layered.go`、`backend/internal/service/openai_gateway_scheduling.go`、`backend/internal/service/openai_account_scheduler_test.go` | `v0.1.170` | 人工审查：候选过滤、粘性绑定、槽位释放后 DB recheck 与重新选号 | `gap` | `docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md:L243`；`backend/internal/service/openai_account_scheduler_layered.go: layeredOpenAIAccountScheduler.Select`；`backend/internal/service/openai_account_scheduler_test.go: TestLayered_GroupedAccountPassesDBFreshRecheck` |
| 2 | Grok/platform/session/previous-response sticky、privacy、image capability | scheduler/WS 入口 -> platform 归一化与 sticky lookup -> capability/privacy/transport 过滤 | `backend/internal/service/openai_account_scheduler_layered.go`、`backend/internal/service/openai_gateway_grok.go`、`backend/internal/service/openai_ws_forwarder_ingress.go` | `v0.1.170`、`v0.1.171` | 人工审查：跨轮绑定、privacy 和 image 条件不会被上游选择短路 | `gap` | `docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md:L244`；`backend/internal/service/openai_account_scheduler_layered.go: selectBySessionHash`；`backend/internal/service/openai_account_scheduler_test.go: TestLayered_SessionStickyPreservesGrokBinding` |
| 3 | OpenAI HTTP/WS、turn ownership、最终 outbound model、failed usage、prompt-cache reuse、proxy circuit | HTTP `OpenAIGatewayHandler` 或 WS forwarder -> `OpenAIGatewayService` -> forward/usage billing -> usage worker | `backend/internal/handler/openai_gateway_handler.go`、`backend/internal/service/openai_ws_forwarder_ingress.go`、`backend/internal/service/openai_gateway_usage.go`、`backend/internal/service/gateway_usage_billing.go` | `v0.1.170`、`v0.1.171` | 人工审查：流式失败、WS 终止和切号均不重复计费 | `gap` | `docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md:L245`；`backend/internal/service/openai_gateway_passthrough.go: forwardOpenAIPassthrough`；`backend/internal/service/openai_ws_forwarder_ingress_session_test.go` |
| 4 | alpha-search、Responses fallback、PAT 401 副作用、WebSearchCalls、body handle | alpha-search handler/service -> upstream request builder -> matched `RequestBodyHandle` -> fallback/retry | `backend/internal/handler/openai_alpha_search.go`、`backend/internal/service/openai_alpha_search.go`、`backend/internal/service/request_body_handle.go` | `v0.1.170`、`v0.1.171` | 人工审查：fallback/retry 后 handle 可重放且最终释放 | `gap` | `docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md:L246`；`backend/internal/service/openai_alpha_search.go: ForwardAlphaSearch, buildOpenAIAlphaSearchRequest`；`backend/internal/service/openai_alpha_search_test.go: TestForwardAlphaSearchPATResponsesFallbackUnauthorizedDoesNotMarkAccountError` |
| 5 | 请求体 spooling/replay/cleanup、异步图片、对象存储、图片输入计费 | request-body coordinator -> Images handler -> `OpenAIGatewayService.ForwardImages` -> async `ImageTaskStore.Save/Get` -> `ImageStorage.Save` -> usage billing | `backend/internal/handler/request_body_coordinator.go`、`backend/internal/handler/openai_images.go`、`backend/internal/service/openai_images.go`、`backend/internal/service/image_task.go`、`backend/internal/repository/image_task_store.go`、`backend/internal/service/image_storage.go` | `v0.1.170` | 人工审查：body 不泄漏、不提前关闭，图片计费仅一次 | `gap` | `docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md:L247`；`backend/internal/service/openai_images.go: OpenAIGatewayService.ForwardImages`；`backend/internal/repository/image_task_store.go: imageTaskStore.Save/Get` |
| 6 | 统一 prompt/security audit 与 Images | gateway/images handler -> request-body coordinator -> `runContentModerationLazy` -> prompt snapshot/coordinator -> legacy moderation -> `OpenAIGatewayService.ForwardImages` | `backend/internal/handler/content_moderation_helper.go`、`backend/internal/handler/request_body_coordinator.go`、`backend/internal/securityaudit/prompt_service.go`、`backend/internal/service/openai_images.go` | `v0.1.170`、`v0.1.171` | 人工审查：每请求最多一次 moderation/payload freeze；关闭态不构造大 payload；`ReleaseText` 前仍可审核 | `gap` | `docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md:L248`；`backend/internal/handler/content_moderation_helper.go: runContentModerationLazy`；`backend/internal/service/openai_images_test.go: TestOpenAIImagesRequestModerationBody_JSONEditIncludesInputImageURLs` |
| 7 | settings 热更新、repository scoped update、API Key auth cache、session binding/step-up | setting handler -> `SettingService` -> repository scoped update/callback；auth middleware -> `APIKeyService.GetByKey` -> L1/L2 auth cache；`SessionBindingContext` -> `WithSessionBinding` -> `enforceSessionBinding`；`TotpHandler.StepUp` -> `StepUpSessionKey` | `backend/internal/repository/setting_repo.go`、`backend/internal/repository/user_repo.go`、`backend/internal/service/api_key_service.go`、`backend/internal/server/middleware/session_binding.go`、`backend/internal/server/middleware/step_up.go`、`backend/internal/handler/totp_handler.go` | `v0.1.170`、`v0.1.171` | 人工审查：部分更新不清空本地字段，失效事件和 session/step-up 不遗漏 | `gap` | `docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md:L249`；`backend/internal/repository/user_repo.go: NewUserRepository`；`backend/internal/server/middleware/session_binding.go: SessionBindingContext, enforceSessionBinding`；`backend/internal/handler/totp_handler.go: TotpHandler.StepUp` |
| 8 | subscription quota reset、续期、退款/余额、receipt、tombstone、outbox | payment/refund handler -> `SubscriptionService` -> `UserSubscriptionRepository` -> transaction/receipt -> cache invalidation outbox | `backend/internal/service/subscription_service.go`、`backend/internal/service/payment_refund.go`、`backend/internal/repository/user_subscription_repo.go`、`backend/migrations/` | `v0.1.170`、`v0.1.171` | 人工审查：单窗口资格、事务锁、提交后 invalidation 与失败回滚 | `gap` | `docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md:L250`；`backend/internal/service/subscription_service.go: AdvanceQuotaCycleWithReceipt`；`backend/internal/service/subscription_advance_quota_test.go` |
| 9 | 用户资源控制、分组复制/批量限额、account shadow、前端本地定制 | admin handler/service -> account/group repository；`AccountsView`/`GroupsView`/Settings/Usage/Subscription/Channels/mobile UI -> admin APIs | `backend/internal/handler/admin/account_handler.go`、`backend/internal/repository/group_repo.go`、`frontend/src/views/admin/AccountsView.vue`、`frontend/src/views/admin/GroupsView.vue`、`frontend/src/api/admin/accounts.ts` | `v0.1.170`、`v0.1.171` | 人工审查：变更页面保持本地入口和资源控制语义 | `gap` | `docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md:L251`；`backend/internal/handler/admin/account_handler.go: AccountHandler`；`frontend/src/views/admin/AccountsView.vue` |
| 10 | Codex identity、动态版本、UA、过载重试 | HTTP/alpha-search `ForwardAlphaSearch`、passthrough `forwardOpenAIPassthrough`、WS 和 `newOpenAIAccountProbe` -> identity/version resolver -> upstream request/retry -> final account/model/error | `backend/internal/service/openai_alpha_search.go`、`backend/internal/service/openai_gateway_passthrough.go`、`backend/internal/service/openai_account_probe.go`、`backend/internal/service/openai_codex_identity.go`、`backend/internal/service/openai_codex_version_sync_service.go`、`backend/internal/pkg/openai/request.go` | `v0.1.170`、`v0.1.171` | 人工审查：模型列表、账号测试和 alpha-search 共用身份来源，重试保留最终语义 | `gap` | `docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md:L252`；`backend/internal/service/openai_gateway_passthrough.go: forwardOpenAIPassthrough`；`backend/internal/service/openai_codex_identity.go: enforceCodexIdentityHeaders`；`backend/internal/service/openai_account_probe.go: newOpenAIAccountProbe`；`backend/internal/service/openai_alpha_search_test.go: TestForwardAlphaSearchPATResponsesFallbackUnauthorizedDoesNotMarkAccountError` |
| 11 | Ent/Wire、Go/pnpm 依赖、CSP/deploy 配置、migrations | schema/provider/manifest source -> `make -C backend generate` -> Ent/Wire output；migration runner -> ordered filename/checksum；frontend manifest -> pnpm lock；`SecurityHeaders` -> CSP policy/nonce | `backend/ent/schema/`、`backend/cmd/server/wire.go`、`backend/cmd/server/wire_gen.go`、`backend/go.mod`、`backend/go.sum`、`frontend/package.json`、`frontend/pnpm-lock.yaml`、`backend/internal/server/middleware/security_headers.go`、`deploy/config.example.yaml`、`backend/migrations/` | `v0.1.170`、`v0.1.171` | 人工审查：source-driven generation、两轮无 diff、完整 migration filename/排序/checksum 和本机 integration | `gap` | `docs/superpowers/plans/2026-08-06-staged-merge-upstream-v0-1-171.md:L253`；`backend/internal/server/middleware/security_headers.go: SecurityHeaders`；`backend/internal/server/middleware/security_headers_test.go: TestSecurityHeaders`；`backend/ent/enttest/enttest.go: NewClient` |

### 冲突台账模板

| 阶段 | 文件 | 分类 | ours 行为 | theirs 行为 | 最小融合 | 验证证据 | 状态 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 待发生 | 待填 | 上游修复 | 待填 | 待填 | 待填 | 待填 | 待填 |
| 待发生 | 待填 | 本地定制 | 待填 | 待填 | 待填 | 待填 | 待填 |
| 待发生 | 待填 | 接口/配置演进 | 待填 | 待填 | 待填 | 待填 | 待填 |
| 待发生 | 待填 | 版本/依赖 | 待填 | 待填 | 待填 | 待填 | 待填 |
| 待发生 | 待填 | 生成代码 | 待填 | 待填 | 待填 | 待填 | 待填 |
| 待发生 | 待填 | migration | 待填 | 待填 | 待填 | 待填 | 待填 |

### Task 3 TDD 与风险

- TDD 不适用。本 Task 只记录 Git 区间事实、能力审查入口和冲突台账模板，不包含生产代码或行为变更；未伪造 RED/GREEN。
- 风险信号：两个 release 区间共 448 个 changed files，横跨 scheduler、gateway/body、audit/auth、subscription/migration、frontend 和生成物。
- 顾虑：矩阵 11 行在阶段 0 均为 `gap`，后续阶段必须以对应的聚焦验证或人工审查证据更新，不能据此预判为 `protected`。
