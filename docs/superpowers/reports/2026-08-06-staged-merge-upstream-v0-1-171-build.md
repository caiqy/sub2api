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
- Task 5 已按阶段 0 当前证据关闭下列各行：`protected` 仅表示有精确命令证据；`manual` 表示已完成下文记录的当前 source base 结构复核，不能视为自动证明；`unverified` 仅表示本机 Docker/Testcontainers 边界。

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

### 能力矩阵（阶段 0 证据）

| # | 行为契约 | 入口/调用链 | 关键文件 | 受影响 tag | 聚焦命令或人工审查点 | 阶段 0 状态 | 证据位置 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | advanced/layered scheduler、pool mode、DB recheck、WaitPlan fallback、同账号重试 | `OpenAIGatewayHandler` -> `OpenAIGatewayService` -> `OpenAIAccountScheduler.Select` -> sticky/session -> layered filter -> slot/`WaitPlan` | `backend/internal/service/openai_account_scheduler.go`、`backend/internal/service/openai_account_scheduler_layered.go`、`backend/internal/service/openai_gateway_scheduling.go`、`backend/internal/service/openai_account_scheduler_test.go` | `v0.1.170` | 人工审查：候选过滤、粘性绑定、槽位释放后 DB recheck 与重新选号 | `manual` | 49-test 正则只匹配 `TestLayered_`、`TestOpenAISelectAccountWithLoadAwareness_`、`TestGatewayServiceRecordUsage_`，不覆盖 `account_pool_mode_test.go` 或 `account_pool_retry_status_codes_test.go`；M1 复核入口：`backend/internal/service/openai_account_scheduler.go:(*OpenAIGatewayService).getOpenAIAccountSchedulerWithContext`、`backend/internal/service/openai_account_scheduler_layered.go:(*layeredOpenAIAccountScheduler).Select`/`selectBySessionHash`、`backend/internal/service/account.go:(*Account).IsPoolMode`/`GetPoolModeRetryCount`/`IsPoolModeRetryableStatus`、`backend/internal/handler/openai_gateway_handler.go:ensureOpenAIPoolModeSessionHash`。 |
| 2 | Grok/platform/session/previous-response sticky、privacy、image capability | scheduler/WS 入口 -> platform 归一化与 sticky lookup -> capability/privacy/transport 过滤 | `backend/internal/service/openai_account_scheduler_layered.go`、`backend/internal/service/openai_gateway_grok.go`、`backend/internal/handler/openai_gateway_handler.go` | `v0.1.170`、`v0.1.171` | 人工审查：跨轮绑定、privacy 和 image 条件不会被上游选择短路 | `manual` | 已完成 M2 结构复核：`backend/internal/service/openai_account_scheduler_layered.go:(*layeredOpenAIAccountScheduler).selectBySessionHash` 与 `backend/internal/handler/openai_gateway_handler.go:ResponsesWebSocket`；sticky 链在分类和 DB recheck 后才取槽位。Task 4 sticky 测试不自动证明 privacy/image 条件。 |
| 3 | OpenAI HTTP/WS、turn ownership、最终 outbound model、failed usage、prompt-cache reuse、proxy circuit | HTTP/WS `OpenAIGatewayHandler` -> `OpenAIGatewayService` -> forward/usage billing -> usage worker | `backend/internal/handler/openai_gateway_handler.go`、`backend/internal/service/openai_gateway_usage.go`、`backend/internal/service/gateway_usage_billing.go` | `v0.1.170`、`v0.1.171` | 人工审查：流式失败、WS 终止和切号均不重复计费 | `manual` | Task 4 仅自动覆盖 usage、Responses WebSocket 和路由分类子集；M3 已复核 `backend/internal/handler/openai_gateway_handler.go:ResponsesWebSocket` 及 `backend/internal/service/openai_proxy_stream_circuit.go:(*openAIProxyStreamCircuit).recordFailure`/`isBlocked`/`activeBlockCount`、`(*OpenAIGatewayService).isOpenAIProxyStreamQuarantined`，prompt-cache reuse/circuit 语义仍非精确自动证明。 |
| 4 | alpha-search、Responses fallback、PAT 401 副作用、WebSearchCalls、body handle | alpha-search handler/service -> upstream request builder -> matched `RequestBodyHandle` -> fallback/retry | `backend/internal/handler/openai_alpha_search.go`、`backend/internal/service/openai_alpha_search.go`、`backend/internal/service/request_body_handle.go` | `v0.1.170`、`v0.1.171` | 人工审查：fallback/retry 后 handle 可重放且最终释放 | `manual` | 已完成 M4 结构复核：`backend/internal/handler/openai_alpha_search.go:AlphaSearch` 绑定 effective handle；`backend/internal/service/openai_alpha_search.go:ForwardAlphaSearch` 通过 matched handle 建请求并在 owned 情况 cleanup；Task 4 未自动证明整条 fallback/PAT 链。 |
| 5 | 请求体 spooling/replay/cleanup、异步图片、对象存储、图片输入计费 | request-body coordinator -> Images handler -> `OpenAIGatewayService.ForwardImages` -> async `ImageTaskStore.Save/Get` -> `ImageStorage.Save` -> usage billing | `backend/internal/handler/request_body_coordinator.go`、`backend/internal/handler/openai_images.go`、`backend/internal/service/openai_images.go`、`backend/internal/service/image_task.go`、`backend/internal/repository/image_task_store.go`、`backend/internal/service/image_storage.go` | `v0.1.170` | 人工审查：body 不泄漏、不提前关闭，图片计费仅一次 | `manual` | Task 4 的 multipart replay/spool-cleanup 测试仅保护该子契约；M5 已复核 `backend/internal/handler/request_body_coordinator.go:requestBodyCoordinator`、`backend/internal/service/request_body_handle.go:RequestBodyHandle`、`backend/internal/service/image_task.go:ImageTaskService` 与 `backend/internal/service/image_storage.go:ImageStorage`，未把异步图片、对象存储或图片计费提升为自动证明。 |
| 6 | 统一 prompt/security audit 与 Images | gateway/images handler -> request-body coordinator -> `runContentModerationLazy` -> prompt snapshot/coordinator -> legacy moderation -> `OpenAIGatewayService.ForwardImages` | `backend/internal/handler/content_moderation_helper.go`、`backend/internal/handler/request_body_coordinator.go`、`backend/internal/securityaudit/prompt_service.go`、`backend/internal/service/openai_images.go` | `v0.1.170`、`v0.1.171` | 人工审查：每请求最多一次 moderation/payload freeze；关闭态不构造大 payload；`ReleaseText` 前仍可审核 | `protected` | Task 4 的 `TestOpenAIImages_` 集合 exit 0；报告明确记录 audit、legacy-once、frozen payload、lazy evaluation、single freeze 和 text release ordering。 |
| 7 | settings 热更新、repository scoped update、API Key auth cache、session binding/step-up | setting handler -> `SettingService` -> repository scoped update/callback；auth middleware -> `APIKeyService.GetByKey` -> L1/L2 auth cache；`SessionBindingContext` -> `WithSessionBinding` -> `enforceSessionBinding`；`TotpHandler.StepUp` 与 middleware `StepUpSessionKey` -> step-up grant | `backend/internal/repository/setting_repo.go`、`backend/internal/repository/user_repo.go`、`backend/internal/service/api_key_service.go`、`backend/internal/server/middleware/session_binding.go`、`backend/internal/server/middleware/step_up.go`、`backend/internal/handler/totp_handler.go` | `v0.1.170`、`v0.1.171` | 人工审查：部分更新不清空本地字段，失效事件和 session/step-up 不遗漏 | `manual` | 已完成 M7 结构复核：`backend/internal/service/setting_update.go:(*SettingService).UpdateSettingsOmitting`/`refreshCachedSettingsAfterWrite` 保持 omitted key 并写后刷新缓存；`backend/internal/service/api_key_service.go:(*APIKeyService).GetByKey`、`backend/internal/server/middleware/session_binding.go:enforceSessionBinding`、`backend/internal/handler/totp_handler.go:StepUp` 与 `backend/internal/server/middleware/step_up.go:StepUpSessionKey` 保持既有安全链。 |
| 8 | subscription quota reset、续期、退款/余额、receipt、tombstone、outbox | payment/refund handler -> `SubscriptionService` -> `UserSubscriptionRepository` -> transaction/receipt -> cache invalidation outbox | `backend/internal/service/subscription_quota_advance_receipt.go`、`backend/internal/service/payment_refund.go`、`backend/internal/repository/user_subscription_repo.go`、`backend/migrations/` | `v0.1.170`、`v0.1.171` | 人工审查：单窗口资格、事务锁、提交后 invalidation 与失败回滚 | `manual` | 已完成 M8 结构复核：`backend/internal/service/subscription_quota_advance_receipt.go:AdvanceQuotaCycleWithReceipt` 与 `backend/internal/service/subscription_cache_invalidation_outbox.go:(*SubscriptionCacheInvalidationWorker).processEvent`；后者只在 tombstone/publish 成功后安排二次投递或 ack，失败走 retry；`backend/internal/repository/subscription_cache_invalidation_outbox_repo.go:ScheduleSecondPass`/`DeleteClaimed`/`RetryClaimed` 保留 claim ownership。 |
| 9 | 用户资源控制、分组复制/批量限额、account shadow、前端本地定制 | admin handler/service -> account/group repository；`AccountsView`/`GroupsView`/Settings/Usage/Subscription/Channels/mobile UI -> admin APIs | `backend/internal/handler/admin/account_handler.go`、`backend/internal/repository/group_repo.go`、`frontend/src/views/admin/AccountsView.vue`、`frontend/src/views/admin/GroupsView.vue`、`frontend/src/api/admin/index.ts` | `v0.1.170`、`v0.1.171` | 人工审查：变更页面保持本地入口和资源控制语义 | `manual` | 已完成 M9 结构复核：`frontend/src/views/admin/AccountsView.vue:AccountsView`、`frontend/src/views/admin/GroupsView.vue:GroupsView` 继续经 `frontend/src/api/admin/index.ts:adminAPI`；`backend/internal/handler/admin/account_handler.go:AccountHandler` 进入 `AdminService`，`backend/internal/repository/group_repo.go:groupRepository` 使用事务资源路径。未把 67 个 frontend 测试扩展为跨模块全量证明。 |
| 10 | Codex identity、动态版本、UA、过载重试 | HTTP/alpha-search `ForwardAlphaSearch`、passthrough `forwardOpenAIPassthrough`、WS 和 `newOpenAIAccountProbe` -> current identity/version headers -> upstream request/retry -> final account/model/error | `backend/internal/service/openai_alpha_search.go`、`backend/internal/service/openai_gateway_passthrough.go`、`backend/internal/service/openai_account_probe.go`、`backend/internal/service/openai_codex_identity.go`、`backend/internal/service/openai_gateway_service.go`、`tag-source v0.1.171:backend/internal/service/openai_codex_version_sync_service.go:OpenAICodexVersionSyncService`、`backend/internal/pkg/openai/request.go` | `v0.1.170`、`v0.1.171` | 人工审查：模型列表、账号测试和 alpha-search 共用身份来源，重试保留最终语义 | `manual` | 已完成 M10 结构复核：当前 source 的 `backend/internal/service/openai_gateway_passthrough.go:forwardOpenAIPassthrough`、`backend/internal/service/openai_codex_identity.go:ensureCodexIdentityHeaders`/`enforceCodexIdentityHeaders`、`backend/internal/service/openai_gateway_service.go:codexCLIUserAgent`/`codexCLIVersion`、`backend/internal/service/openai_account_probe.go:newOpenAIAccountProbe` 与 `backend/internal/service/openai_alpha_search.go:ForwardAlphaSearch` 维持既有链路；version sync service 仅为 `tag-source v0.1.171:backend/internal/service/openai_codex_version_sync_service.go:OpenAICodexVersionSyncService`，不是当前 source base 文件。未提升为组合级自动证明。 |
| 11 | Ent/Wire、Go/pnpm 依赖、CSP/deploy 配置、migrations | schema/provider/manifest source -> `make -C backend generate` -> Ent/Wire output；migration runner -> ordered filename/checksum；frontend manifest -> pnpm lock；`SecurityHeaders` -> CSP policy/nonce | `backend/ent/schema/`、`backend/cmd/server/wire.go`、`backend/cmd/server/wire_gen.go`、`backend/go.mod`、`backend/go.sum`、`frontend/package.json`、`frontend/pnpm-lock.yaml`、`backend/internal/server/middleware/security_headers.go`、`deploy/config.example.yaml`、`backend/migrations/` | `v0.1.170`、`v0.1.171` | 人工审查：source-driven generation、两轮无 diff、完整 migration filename/排序/checksum 和本机 integration | `unverified` | Task 4 的 refresh、stable 1、stable 2 generate 均 exit 0 且生成 diff 为空；仅完整 migration 空库/升级/幂等/checksum integration 因本机 Docker CLI 缺失而未验证。 |

### Task 6 最终冲突台账（取代模板）

以下是 `v0.1.170` 的实际 28 个文本冲突。`CC` 是 `git show --cc --format= --unified=0 98c7b048... -- <下表 28 路径>`，exit `0`，显示两亲本与最终 hunk；`TP` 是 merge parent 检查，`WC` 是 `git diff --check 30528a82... 98c7b048...`，exit `0`。它们证明 pure merge 拓扑与最终文本，不是 merge 后行为 PASS。

| # | 文件 | 分类 | ours 行为 | theirs 行为 | 最终融合 | 验证证据 | 状态 |
| ---: | --- | --- | --- | --- | --- | --- | --- |
| 1 | `backend/cmd/server/VERSION` | 版本/依赖 | `0.1.169.3` | `0.1.169` | 保持 `0.1.169.3` | `CC`、对象读取 | 已融合 |
| 2 | `backend/cmd/server/wire_gen.go` | 生成代码 | 本地 Wire 输出 | 上游 provider 输出 | 从融合 Wire source 的最终输出 | `CC`、生成映射 | 已融合 |
| 3 | `backend/ent/client.go` | 生成代码 | 本地 Ent entities | 上游 Group profit schema | 完整 schema 的 final client | `CC`、生成映射 | 已融合 |
| 4 | `backend/ent/group.go` | 生成代码 | 本地 Group fields | profit fields | 两侧 Group fields | `CC`、生成映射 | 已融合 |
| 5 | `backend/ent/mutation.go` | 生成代码 | 本地 mutation fields | profit mutation fields | 两侧 mutation API | `CC`、生成映射 | 已融合 |
| 6 | `backend/internal/handler/admin/setting_handler.go` | 接口/配置演进 | 本地 settings DTO | 上游 settings/admin fields | 两组 response fields | `CC` | 已融合 |
| 7 | `backend/internal/handler/admin/setting_handler_update.go` | 接口/配置演进 | 本地 runtime update | 上游 settings update | 两侧 partial update fields | `CC` | 已融合 |
| 8 | `backend/internal/handler/gateway_handler.go` | 上游修复 | body/detail snapshot | profit/pricing/partial usage | 两侧 route 与 usage 数据 | `CC` | 已融合 |
| 9 | `backend/internal/handler/gateway_handler_chat_completions.go` | 接口/配置演进 | local usage fields | `PricingAt` | usage input 带两侧字段 | `CC` | 已融合 |
| 10 | `backend/internal/handler/gemini_v1beta_handler.go` | 上游修复 | detail snapshot/slot | profit admission/`PricingAt` | 两侧 usage/slot 路径 | `CC` | 已融合 |
| 11 | `backend/internal/handler/openai_alpha_search.go` | 上游修复 | effective request body | profit selection | 两侧入口语义 | `CC` | 已融合 |
| 12 | `backend/internal/handler/openai_chat_completions.go` | 接口/配置演进 | local usage fields | `PricingAt` | usage input 带两侧字段 | `CC` | 已融合 |
| 13 | `backend/internal/handler/openai_embeddings.go` | 接口/配置演进 | local usage fields | `PricingAt` | usage input 带两侧字段 | `CC` | 已融合 |
| 14 | `backend/internal/handler/openai_gateway_count_tokens.go` | 上游修复 | effective body/count path | profit gate | count tokens 显式抑制 profit gate | `CC` | 已融合 |
| 15 | `backend/internal/handler/openai_gateway_handler.go` | 上游修复 | concurrency/body/usage | profit veto/turn pricing | 两侧 slot/body/usage 语义 | `CC` | 已融合 |
| 16 | `backend/internal/service/admin_account.go` | 接口/配置演进 | `probeToggleOff` prefetch | `RateMultiplier` prefetch | prefetch 条件取并集 | `CC` | 已融合 |
| 17 | `backend/internal/service/api_key_auth_cache_impl.go` | 接口/配置演进 | snapshot v19 local controls | snapshot v18 profit fields | v20 覆盖两侧 fields | `CC` | 已融合 |
| 18 | `backend/internal/service/content_moderation.go` | 上游修复 | `buildLog` matched-keyword contract | moderation proxy client/cache | helpers 和 log contract 共存 | `CC` | 已融合 |
| 19 | `backend/internal/service/gateway_scheduling.go` | 上游修复 | composite route/platform | group profit gate | route decision 后装 gate | `CC` | 已融合 |
| 20 | `backend/internal/service/gateway_usage_billing.go` | 接口/配置演进 | `DetailSnapshot` | `PricingAt` | usage structs 传递两侧 fields | `CC` | 已融合 |
| 21 | `backend/internal/service/openai_account_scheduler.go` | 上游修复 | layered sticky/DB recheck | upstream-cost ordering | 两侧 scheduler behavior | `CC` | 已融合 |
| 22 | `backend/internal/service/openai_gateway_scheduling.go` | 上游修复 | quota auto-pause | legacy profit gate | 两者先于 selection | `CC` | 已融合 |
| 23 | `backend/internal/service/openai_oauth_passthrough_test.go` | 本地定制 | missing instructions reject | default instructions forward | 按用户裁决保留 upstream default/forward test | `CC` | 已融合 |
| 24 | `backend/internal/service/subscription_service.go` | 本地定制 | `StartsAt`/day-start reset | `s.now()` window reset | `StartsAt` 优先，`startOfDay(s.now())` | `CC` | 已融合；Task 8 RED handoff |
| 25 | `backend/internal/service/user_subscription.go` | 本地定制 | strict `<` 和 local windows | inclusive `<=`/upstream windows | `Check*Limit` 为 `<=`，保留 local window correction | `CC` | 已融合；Task 7 RED handoff |
| 26 | `frontend/src/components/account/CreateAccountModal.vue` | 本地定制 | passthrough `extra` | billing probe payload | passthrough 加 API-key probe | `CC` | 已融合 |
| 27 | `frontend/src/components/account/EditAccountModal.vue` | 本地定制 | passthrough/quota editor | probe/rate-sync controls | 两侧 `extra` fields | `CC` | 已融合 |
| 28 | `frontend/src/components/account/__tests__/CreateAccountModal.spec.ts` | 本地定制 | local create harness | upstream probe/import harness | 两组 mock/assertions | `CC` | 已融合 |

### Task 3 TDD 与风险

- TDD 不适用。本 Task 只记录 Git 区间事实、能力审查入口和冲突台账模板，不包含生产代码或行为变更；未伪造 RED/GREEN。
- 风险信号：两个 release 区间共 448 个 changed files，横跨 scheduler、gateway/body、audit/auth、subscription/migration、frontend 和生成物。
- 顾虑：阶段 0 的状态已关闭为 1 个 `protected`、9 个 `manual` 和 1 个仅 Docker 边界的 `unverified`；后续 merge 阶段仍必须以对应的聚焦验证或结构审查更新，不能将 `manual` 推断为自动证明。

## Task 4 恢复：更新 immutable source base

- 原 source base `b576f73a22c4bf23d61727fc93950766a7e33929` 的 Task 4 lint 阻塞保留为历史证据，不改写原始结果。
- 独立前置 change `restore-backend-lint-gate` 已完成、验证并归档到 `main@16c07d8064b0b4604e9f47ef782e7d29534402d3`。
- 用户于 2026-08-07 明确确认用该 `main` 提交替代原 source base。
- 新 immutable source base：`16c07d8064b0b4604e9f47ef782e7d29534402d3`。
- 恢复时 execution base：`fd109296b5f41398350070dd8df826846d9adb1b`，即当前 change checkpoint 与新 `main` 的 merge commit。
- `git merge-base --is-ancestor 16c07d8064b0b4604e9f47ef782e7d29534402d3 fd109296b5f41398350070dd8df826846d9adb1b` exit 0；相对新 source base 的 backend 差异仅为已提交并审查的 `backend/internal/handler/openai_images_controls_test.go` 基线保护断言，无未归属生产差异。
- source 与 execution `VERSION` 均为 `0.1.169.3`。Task 4 必须在新基线上重新执行全部聚焦测试、修正后的 quota unit 集合、`make test`、`make build`、三轮 generate 稳定性和静态冲突检查；不得沿用旧阻塞前的通过结果替代重跑。

## Task 5：基线 Docker 条件门禁与阶段 0 证据封存

- TDD 不适用。本任务只运行既有 integration 条件门禁并记录证据，不修改生产行为；未伪造 RED/GREEN。
- Task 4 当前证据：`make test` exit 0；canonical Bash-shell `make VERSION=0.1.169.3 build` exit 0；`make -C backend generate` 的 refresh、stable 1、stable 2 均 exit 0，且每轮 `backend/ent` 与 `backend/cmd/server/wire_gen.go` diff 均为空；静态冲突检查 exit 0。

### Task 4 聚焦命令的持久化转录

以下均为 `task-4-report.md` 的已有结果，本 Task 未重跑：

| 完整命令 | Exit | 测试发现/摘要 |
| --- | ---: | --- |
| `go test -count=1 ./internal/service -run '^(TestLayered_|TestOpenAISelectAccountWithLoadAwareness_|TestGatewayServiceRecordUsage_)'` | `0` | `ok github.com/Wei-Shaw/sub2api/internal/service 1.662s`；独立 `-list` 发现 `49` 个测试。 |
| `go test -tags=unit -count=1 ./internal/service -run '^(TestCalculateQuotaCycleAdvance_.*|TestAdvanceQuotaCycle_.*|TestAdminResetQuota_(UsesCommittedResetVersionForCacheInvalidation|OuterTransactionInvalidatesAfterCommit)|TestCheckAndResetWindows_UsesCommittedResetVersionForCacheInvalidation)$'` | `0` | `ok github.com/Wei-Shaw/sub2api/internal/service 1.845s`；独立 `-list` 发现 `13` 个 quota 测试。 |
| `go test -count=1 ./internal/handler -run '^(TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot|TestOpenAIImages_|TestOpenAIResponsesWebSocket_)'` | `0` | `ok github.com/Wei-Shaw/sub2api/internal/handler 10.149s`；独立 `-list` 发现 `39` 个 handler 测试。 |
| `go test -count=1 ./internal/server/routes -run '^(TestEveryGatewayPOSTRouteIsClassifiedForPromptAuditCoverage|TestResponsesWebSocketHasFirstAndSubsequentTurnPromptGates)$'` | `0` | `ok github.com/Wei-Shaw/sub2api/internal/server/routes 1.651s`；独立 `-list` 发现 `2` 个 routes 测试。 |
| `pnpm --dir frontend exec vitest run src/utils/__tests__/subscriptionQuota.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/UsageView.spec.ts src/components/channels/__tests__/AvailableChannelsTable.spec.ts` | `0` | `4` files、`67` tests passed in `19.43s`（`15 + 30 + 16 + 6`）。 |
| `go test -count=1 ./internal/handler -run '^TestOpenAIGatewayHandlerImages_MultipartReplayUsesMappedEffectiveBody$'` | `0` | `ok github.com/Wei-Shaw/sub2api/internal/handler 2.660s`；单个具名测试验证 mapped replay 与 forced-spool cleanup。 |

### Task 5 Reviewer Evidence Repair

- 结构复核 source base：`16c07d8064b0b4604e9f47ef782e7d29534402d3`；本次复核 HEAD `d3c010a67dbfcca411d8b255f25b2a66b9ab2744` 是其后代。相对该 source base 的 `backend`/`frontend` 差异只有既有 `backend/internal/handler/openai_images_controls_test.go`，没有本 Task 生产代码变更。

#### M1：advanced/layered scheduler、pool mode、DB recheck、WaitPlan fallback、同账号重试

- 已复核：`backend/internal/service/openai_account_scheduler.go:(*OpenAIGatewayService).getOpenAIAccountSchedulerWithContext`、`backend/internal/service/openai_account_scheduler_layered.go:(*layeredOpenAIAccountScheduler).Select`/`selectBySessionHash`、`backend/internal/service/account.go:(*Account).IsPoolMode`/`GetPoolModeRetryCount`/`IsPoolModeRetryableStatus`、`backend/internal/handler/openai_gateway_handler.go:ensureOpenAIPoolModeSessionHash`，以及 `backend/internal/service/openai_account_runtime_block_fastpath.go:(*OpenAIGatewayService).handleOpenAIAccountUpstreamError`。
- 调用链/不变量：runtime setting 只在 `layered` 时构造 layered scheduler，否则回退 weighted；layered `Select` 依序处理 previous-response、session sticky 和分层过滤。session sticky 在占槽前检查 excluded、可调度性、platform/privacy/request/transport 兼容性并 DB recheck，取槽失败才形成 `WaitPlan`。pool mode 仅允许 API Key/Bedrock credentials 开关；同账号 retry 次数默认 3、下界 0、上界 10；未配置状态码时为 `401/403/429`，显式空数组关闭该触发；未提供 session hash 的 pool request 生成一次性 sticky key。pool-mode retryable 错误不进入通用 account+model transient cooldown。
- 当前 source base 结论：Task 4 的 49-test 正则只匹配 `TestLayered_`、`TestOpenAISelectAccountWithLoadAwareness_`、`TestGatewayServiceRecordUsage_`，没有匹配 `account_pool_mode_test.go`、`account_pool_retry_status_codes_test.go` 等 pool-mode 契约；整行按最弱证据保持 `manual`，而非扩大 `protected`。
- tag merge 后重新验证：pool mode 开关、retry count 边界和默认/覆盖状态码；一次性 sticky key 的同账号 retry；retryable 错误不抢先冷却账号；scheduler 的 sticky/DB recheck/slot/`WaitPlan` 组合。

#### M2：Grok/platform/session/previous-response sticky、privacy、image capability

- 已复核：`backend/internal/service/openai_account_scheduler_layered.go:(*layeredOpenAIAccountScheduler).selectBySessionHash` 与 `backend/internal/handler/openai_gateway_handler.go:ResponsesWebSocket` 的首轮入口。
- 调用链/不变量：sticky ID 经过 schedulable 获取、分类及 DB recheck 后才取槽位；失效时清 sticky 或置 `SkipStickyBind`，仅成功取槽位刷新 TTL，繁忙账号以真实 `Concurrency` 形成 `WaitPlan`。
- 当前 source base 结论：跨轮 sticky 链路存在且先复核再占槽；privacy/image capability 的跨轮组合没有精确自动执行证据，因此保持 `manual`。
- tag merge 后重新验证：previous-response 与 session 两类绑定，及 platform/privacy/image/transport 过滤不被任何快捷路径绕过。

#### M3：OpenAI HTTP/WS、usage、prompt-cache reuse、proxy circuit

- 已复核：`backend/internal/handler/openai_gateway_handler.go:Responses`、`backend/internal/handler/openai_gateway_handler.go:ResponsesWebSocket`、session hash、`backend/internal/service/openai_gateway_passthrough.go:forwardOpenAIPassthrough`，以及 `backend/internal/service/openai_proxy_stream_circuit.go:(*openAIProxyStreamCircuit).recordFailure`/`isBlocked`/`activeBlockCount`、`(*OpenAIGatewayService).recordOpenAIProxyStreamDisconnect`/`isOpenAIProxyStreamQuarantined`。
- 调用链/不变量：HTTP 在 route/model mapping 后固定 effective body；WS 校验首个 `response.create`、映射模型并用 defer 释放 turn 槽位；session hash 优先 metadata/cacheable content 后再回退内容摘要；proxy circuit 以 OpenAI proxy ID 有界、TTL 隔离，只在无可用账号且确有隔离时 fail-open 重选。
- 当前 source base 结论：HTTP/WS 路由、body、槽位和 circuit 的结构链存在；Task 4 仅自动证明 usage、WS 和路由分类子集，未精确证明 prompt-cache reuse 或 circuit 语义，故整行必须是 `manual`。
- tag merge 后重新验证：prompt-cache 命中/重绑、proxy trip/TTL/fail-open、最终 outbound model 与 failover/WS 终止下 failed usage 的一次性结算。

#### M4：alpha-search、Responses fallback、PAT 401、WebSearchCalls、body handle

- 已复核：`backend/internal/handler/openai_alpha_search.go:AlphaSearch`、`backend/internal/service/openai_alpha_search.go:ForwardAlphaSearch`/`buildOpenAIAlphaSearchRequest` 和 `backend/internal/service/request_body_handle.go:RequestBodyHandle`。
- 调用链/不变量：handler 将 effective handle bind 到 context；服务复用匹配 handle，owned handle 由 defer cleanup；request 通过 `Open`/`GetBody` 重放，关闭 reader 后才删除 spool，删除失败安排有限重试；PAT 路径分流到 Responses WebSearch，单次 401 不按通用路径认定全局凭据失效。
- 当前 source base 结论：fallback/replay/cleanup 与 PAT 分流的结构路径可追溯，但没有一条 Task 4 精确 PASS 覆盖整个组合，保持 `manual`。
- tag merge 后重新验证：PAT 401 副作用、Responses fallback、WebSearchCalls 计费和 retry 后 handle 的重放/释放。

#### M5：spooling/replay/cleanup、异步图片、对象存储、图片计费

- 已复核：`backend/internal/handler/request_body_coordinator.go:requestBodyCoordinator`、`backend/internal/service/request_body_handle.go:RequestBodyHandle`、`backend/internal/service/image_task.go:ImageTaskService`、`backend/internal/repository/image_task_store.go:ImageTaskStore` 和 `backend/internal/service/image_storage.go:ImageStorage`。
- 调用链/不变量：coordinator 替换 effective handle 时清理旧 owned handle；spool reader 未关闭时只标记 cleanup，最后 reader 关闭后删除；任务 `Create` 以 owner 和 TTL 写 Redis，`Get` 对 owner/API key 不匹配返回 not found；对象存储未启用时 async task 整体禁用，避免写 Redis。
- 当前 source base 结论：请求体、任务归属和存储启用门的结构明确；multipart replay/spool-cleanup 的单测没有覆盖 async task、对象存储或图片输入/输出计费，整行保持 `manual`。
- tag merge 后重新验证：async 图片从 `ForwardImages` 到 uploader/store 的成功和失败路径，以及每次图片输入/输出只结算一次。

#### M7：settings、auth cache、session binding、step-up

- 已复核：`backend/internal/service/setting_update.go:(*SettingService).UpdateSettingsOmitting`/`refreshCachedSettingsAfterWrite`、`backend/internal/repository/setting_repo.go:(*settingRepository).SetMultiple`、`backend/internal/service/api_key_service.go:(*APIKeyService).GetByKey`、`backend/internal/server/middleware/session_binding.go:SessionBindingContext`/`enforceSessionBinding`、`backend/internal/handler/totp_handler.go:StepUp` 与 `backend/internal/server/middleware/step_up.go:StepUpSessionKey`。
- 调用链/不变量：omitted key 在写入前从 map 删除，部分更新从存储重建缓存并调用 update callback；认证先查 L1/L2 并以 singleflight 汇合；binding mismatch 撤销 session family、审计并 401；step-up 优先 session ID，未授权或检查失败均拒绝。
- 当前 source base 结论：局部更新和安全门控的实际入口、失效与 fail-closed 路径均存在；跨实例 cache invalidation 与所有敏感路由组合没有完整自动证明，保持 `manual`。
- tag merge 后重新验证：partial settings 不覆盖未携带字段、L1/L2 失效广播、session mismatch 和 step-up 路由门控。

#### M8：subscription quota、receipt、tombstone、outbox

- 已复核：`backend/internal/service/subscription_quota_advance_receipt.go:AdvanceQuotaCycleWithReceipt`、`backend/internal/repository/user_subscription_repo.go:(*userSubscriptionRepository).GetByIDForUpdate`，以及 `backend/internal/service/subscription_cache_invalidation_outbox.go:(*SubscriptionCacheInvalidationWorker).processEvent`/`retry`。
- 调用链/不变量：同一事务内先检查 receipt、`GetByIDForUpdate`、计算并更新订阅、写 receipt；仅 commit 成功后才 defer invalidation；outbox 先 tombstone 和 publish，成功后安排第二次安全投递或最终 ack，失败则 retry。
- 当前 source base 结论：事务、receipt 和 post-commit invalidation 顺序由源码落实；Task 4 的 13 个 quota 测试只自动保护其中子集，支付/退款全链仍为 `manual`。
- tag merge 后重新验证：续期、退款/余额回滚、单窗口资格、行锁和 outbox 重试/二次投递。

#### M9：用户资源控制、分组复制/批量限额、account shadow、前端本地定制

- 已复核：`frontend/src/views/admin/AccountsView.vue:AccountsView`、`frontend/src/views/admin/GroupsView.vue:GroupsView`、`frontend/src/api/admin/index.ts:adminAPI`、`backend/internal/handler/admin/account_handler.go:AccountHandler`、`backend/internal/service/admin_service.go:AdminService` 的 duplicate/batch/shadow 入口和 `backend/internal/repository/group_repo.go:groupRepository` 事务资源路径。
- 调用链/不变量：前端列表与 duplicate 操作继续经 `adminAPI`；账户创建由 `executeAdminIdempotent` 进入 `AdminService`，仅新建结果调度探测；服务接口保留 batch limit、group duplicate 和 shadow 入口；group 删除先事务锁定并原子处理关联资源。
- 当前 source base 结论：本地 UI 到 admin service 的入口及资源控制边界未丢失；67 个 frontend 测试不能证明全部 clone/batch/shadow 语义，保持 `manual`。
- tag merge 后重新验证：复制后的账号绑定/优先级、批量限额和 shadow 资源归属，以及 Accounts/Groups 页面仍调用本地 API 语义。

#### M10：Codex identity、动态版本、UA、过载重试

- 已复核：当前 source 的 `backend/internal/service/openai_gateway_passthrough.go:forwardOpenAIPassthrough`、`backend/internal/service/openai_codex_identity.go:ensureCodexIdentityHeaders`/`enforceCodexIdentityHeaders`、`backend/internal/service/openai_gateway_service.go:codexCLIUserAgent`/`codexCLIVersion`、`backend/internal/service/openai_account_probe.go:newOpenAIAccountProbe` 和 `backend/internal/service/openai_alpha_search.go:ForwardAlphaSearch`；`tag-source v0.1.171:backend/internal/service/openai_codex_version_sync_service.go:OpenAICodexVersionSyncService` 仅属于待合并 tag。
- 调用链/不变量：passthrough 每次从 immutable handle 读 body，构造请求后发布 final upstream model，并关闭 request body；current identity path 以 `codexCLIUserAgent`/`codexCLIVersion` 填充缺失头，再由 `enforceCodexIdentityHeaders` 配对 UA/originator 并提升过低 version；probe 只 rehydrate active、schedulable、probe-enabled 的 OpenAI 账号。
- 当前 source base 结论：身份收敛、final model 发布、探活资格和重试入口可追溯；`openai_codex_version_sync_service.go` 不在当前 source base，动态版本同步只按上述 tag-source 记录。所有 overload/retry 终态仍缺少组合级精确 PASS，保持 `manual`。
- tag merge 后重新验证：models、probe、alpha-search 和 passthrough 使用同一身份/version 规则，并保留最终模型与错误语义。

- 阶段 0 矩阵汇总：`protected=1`，`manual=9`，`unverified=1`（仅 Docker/Testcontainers migration 边界），`gap=0`。`gap=0` 仅因每个 `manual` 行已有上述实际结构复核结论，不表示获得自动保护。
- 固定 `Invoke-MigrationUpgradeIntegration -Stage 'baseline'` 已运行。外层 PowerShell exit 0，返回 `Status=unverified`，原因为本机未解析到 `docker` 可执行文件；目标 Go test 未启动，未出现精确 PASS 或 SKIP。
- helper 返回的 Docker preflight 路径：`C:/Users/caiqy/AppData/Local/Temp/sub2api-v0.1.171-baseline-docker-preflight.log`。后续 `Test-Path` 为 `False`：PowerShell 的命令解析错误未进入 helper 的 `2>&1` 管道，空管道未创建该文件。目标 test log `C:/Users/caiqy/AppData/Local/Temp/sub2api-v0.1.171-baseline-migration-upgrade.log` 同样不存在，因为 integration 未启动。
- 风险信号：跨模块、并发和公共 API 为上游后续 merge 审查范围；matrix 行 1-5、7-10 均为已复核但未自动证明的 `manual`；migration 为唯一 `unverified`；结论 `DONE_WITH_CONCERNS`；本次受跟踪的 ledger diff 未超过 200 行。

### 2026-08-07 Reviewer Citation Supersession

- 此节替代此前 M2/M3/M7/M8/M9/M10 的错误定位：`ResponsesWebSocket` 为 `backend/internal/handler/openai_gateway_handler.go:ResponsesWebSocket`；`StepUpSessionKey` 为 `backend/internal/server/middleware/step_up.go:StepUpSessionKey`；`AdvanceQuotaCycleWithReceipt` 为 `backend/internal/service/subscription_quota_advance_receipt.go:AdvanceQuotaCycleWithReceipt`；`adminAPI` 为 `frontend/src/api/admin/index.ts:adminAPI`。
- 当前 source 的 Codex identity/version 入口为 `backend/internal/service/openai_codex_identity.go:ensureCodexIdentityHeaders`/`enforceCodexIdentityHeaders` 和 `backend/internal/service/openai_gateway_service.go:codexCLIUserAgent`/`codexCLIVersion`。`OpenAICodexVersionSyncService` 仅为 `tag-source v0.1.171:backend/internal/service/openai_codex_version_sync_service.go:OpenAICodexVersionSyncService`，不作为当前 source base 文件引用。
- Superseding conclusion: `protected=1`，`manual=9`，`unverified=1`，`gap=0`；Docker/Testcontainers migration 仍是唯一 `unverified` 边界。

## Task 6：v0.1.170 merge 最终证据与 handoff

- 唯一可信状态：`DONE`，仅表示 pure merge 拓扑和上述 28 项融合边界已完成；不表示所有 merge 后行为已验证。
- merge commit：`98c7b04874361a1cf95b8dea90ed1c4db2f05d4d`（`merge: upstream v0.1.170`）。
- parents：第一父 `30528a82e32bfedc011d741e870964beb5743aa4`；第二父 `c043c24774228ba891ddf90d783aa6dc7d0855b5`（固定 `v0.1.170^{}`）。`TP` exit `0` 证实该拓扑。
- `backend/cmd/server/VERSION` 经对象读取仍为 `0.1.169.3`；没有将中间版本升级混入 merge。
- 全部 28 个 final hunk 已以 `CC` 复核；`WC` 未报告空白错误。未在本轮重跑 implementer 已运行的测试。

### Generated source mapping

| 输出 | 最终 source mapping | 复核结论 |
| --- | --- | --- |
| `backend/ent/client.go` | 完整 `backend/ent/schema/**`；本冲突的关键 source 为 `backend/ent/schema/group.go` | 最终 client 包含 Group profit 字段和既有本地 schema。 |
| `backend/ent/group.go`、`backend/ent/mutation.go` | `backend/ent/schema/group.go` | 最终 output 同时暴露本地 Group API 和 profit-control 字段。 |
| `backend/cmd/server/wire_gen.go` | `backend/cmd/server/wire.go`、`backend/internal/handler/wire.go`、`backend/internal/repository/wire.go`、`backend/internal/service/wire.go` | `wire_gen.go` 仍标记为 `Code generated by Wire. DO NOT EDIT.`；没有把它作为手写融合源。 |

### 已批准的 compile adaptations

- `backend/internal/handler/openai_gateway_cyber_test.go`：1 处 `NewContentModerationService(..., nil)` 仅追加最后一个 `nil`。
- `backend/internal/handler/openai_images_controls_test.go`：4 处 `NewContentModerationService(..., nil)` 仅各自追加最后一个 `nil`。
- 合计 5 处。追加的末尾 `nil` 对应 merge 中 `backend/internal/service/content_moderation.go:NewContentModerationService` 构造器签名的 `emailService *EmailService` 形参；该构造器 source 与两个适配路径同在 `98c7b048...` merge。
- 这两条路径是用户批准的 merge-induced compile adaptations，不再作为 unexpected/non-upstream violation；没有修改任何断言或测试行为。

- 按 canonical Task 6 的当前定义重建：`$generated170Exceptions` 从非上游路径中排除上述两个 `$compileAdaptationPaths`。`git diff --name-only 30528a82... 98c7b048...` 相对 `git diff --name-only v0.1.169..v0.1.170` 的仅有差异正是这两条 compile-adaptation 路径，因此 `$generated170Exceptions` 为空。它们既不是生成输出，也不属于 generated exceptions；Ent/Wire/go.sum/pnpm-lock 的真实 generated-source mapping 仍仅为上表所列 source，且没有未映射的 generated exception。

### Known RED handoff

- **Task 7，未闭合且不得写为 PASS：**`user_subscription.go` 已把领域 `CheckDailyLimit`、`CheckWeeklyLimit`、`CheckMonthlyLimit` 改为 inclusive `<=`；但 `BillingCacheService.CheckBillingEligibility` 仍以缓存 usage `>= limit` 拒绝。canonical Task 7 要求先建立端到端 exact-limit 放行、超限拒绝的 RED/GREEN 回归，并做最小修复。
- **Task 8，未闭合且不得写为 PASS：**带 `unit` tag 的 `TestDelayedFirstUseAnchorsMonthlyWindowAtActivation` 仍断言首次使用激活窗口，`TestAdminResetQuota_ResetBoth` 仍断言操作精确时刻；两者与用户裁决的 `StartsAt` 和当天零点语义相反。canonical Task 8 要求以 `go test -tags=unit` gate 先记录真实 RED，再修正测试/实现到用户裁决。
- Task 6 不修改这两类生产代码或测试；它们必须由 Task 7/8 的 TDD 闭合。
