# 分段合并上游 v0.1.159 验证记录

## 执行内容
- 固定 v0.1.157、v0.1.158、v0.1.159 的 annotated tag refs 和 peeled commits。
- 在尚未合并 v0.1.157 前运行扩展前完整基线。
- 未合并任何新 tag，未 push、release 或 deploy。

## 扩展起点与固定对象
- 首轮已验证提交：`315617bdec0e21fe8aeb119a986bde960c4864b3`
- 扩展实施分支：`feature/20260717/staged-merge-upstream-v0-1-159`
- 最终边界：`v0.1.159^{}` (`2a75d7d2387587d86ca3c5e5cd8ca96cf3d104c6`)
- 排除范围：`v0.1.159^{}` 之后的 `upstream/main`
- v0.1.157：tag object `a44e63f9fab426ec181bafcf4e4c1a002bbcb8e0`，peeled commit `a2779cd5f30d6d3904a9d59088aed09507678dfe`。
- v0.1.158：tag object `c6ece7d092843c19a2d14d1264669c6416969f6d`，peeled commit `26abd19a2812edba02bbef93c3e2a620141cc257`。
- v0.1.159：tag object `2a2b58263cdf20adf049f3ad8f9e23b4401698c9`，peeled commit `2a75d7d2387587d86ca3c5e5cd8ca96cf3d104c6`。

## 扩展基线
- `make test` 通过：Go 测试和 lint、前端 lint/typecheck、Vitest 181 个测试文件和 1405 个测试全部通过。
- `pnpm --dir frontend run build` 通过：970 个模块完成生产构建。
- 连续两次 `make -C backend generate` 均通过，之后的 Ent/Wire 定向 diff 均为空。
- `git diff --check` 通过。

## 能力矩阵增量

### 判定和调查方法
- 三段交集：以 `git diff --name-only v0.1.156^{}..v0.1.157^{}`、`v0.1.157^{}..v0.1.158^{}`、`v0.1.158^{}..v0.1.159^{}` 与首轮能力矩阵相交。changed-files 只用于归属 tag，不用于推导调用链。
- CodeGraph `context`/`impact` 复核当前基线的共享入口：`SelectAccountWithScheduler`、`IsImageGenerationIntent`、`BillingRateMultiplier`、`Responses`、`ResponsesWebSocket`、`ForwardAlphaSearch`、`GetClientIP`、`BatchUpdateConcurrency`、`DuplicateAccount` 与 `applyUsageBilling`。随后从调用点核对：`RecordUsage -> BillingRateMultiplier -> applyUsageBilling -> buildUsageBillingCommand`，以及 `openai_gateway_grok.go`、Messages、Chat、WS bridge 对 `resolveGrokCacheIdentity` 的调用。changed-files 从未作为调用链判断依据。
- 更正：`resolveGrokCacheIdentity` 和 `openai_gateway_grok_cache_test.go` 已存在于当前基线；测试文件带 `//go:build unit`，所以必须使用 `-tags unit`。session binding、操作审计、step-up 和 group duplicate 仍是目标 tag 新符号，按 `manual` 逐段验收。
- 状态含义沿用首轮：`protected` 为当前基线存在直接行为断言；`manual` 为新 schema/生成物、新入口或跨层契约，要求合并后人工验收；`approved-removal` 仅用于已批准移除的本地首 Token 超时；`gap` 仅适用于本地独有、目标触及且缺少直接断言的既有行为。

### 扩展能力矩阵

| 风险面 / tag | 入口 <- 调用方 | 目标 tag 修改文件 | 现有测试和验证命令 | 人工审查点 | 状态 |
| --- | --- | --- | --- | --- | --- |
| 异步图片任务和对象存储 / 157 | `ImageTaskHandler` <- gateway route registration; -> `ImageTaskService` -> `ImageTaskStore`/S3 store。 | `handler/image_task_handler.go`、`service/image_task.go`、`repository/{image_task_store,image_storage_s3}.go`、`server/routes/gateway.go`。 | tag tests: `image_task_handler_test.go`、`image_task_store_test.go`; 合并后 `go -C backend test ./internal/handler ./internal/repository ./internal/server/middleware -run 'ImageTask' -count=1`。 | worker 状态、对象 key、失败清理、API-key 鉴权。 | manual |
| 图片输入 token 和费用 / 157 | `GatewayKeyBilling` <- gateway route; -> `RecordUsage` -> `calculateRecordUsageCost`/usage repository。 | `handler/gateway_key_billing.go`、`service/{gateway_usage_billing,billing_service}.go`、`repository/usage_log_repo_*.go`、migrations `178`、`179`。 | tag tests: `gateway_key_billing_test.go`、`usage_log_repo_request_type_test.go`; 合并后 `go -C backend test ./internal/handler ./internal/service ./internal/repository -run 'Image.*(Billing|Token)|RequestType' -count=1`。 | token 字段、group price、usage detail 同步。 | manual |
| 上游计费倍率、scheduler 和 image intent / 157,159 | `SelectAccountWithScheduler*` <- OpenAI handler; `IsImageGenerationIntent` <- image/Responses ingress; `RecordUsage` <- gateway handler -> `BillingRateMultiplier` -> `applyUsageBilling` -> `buildUsageBillingCommand`。 | 157: `service/{account,openai_account_scheduler,openai_gateway_usage,gateway_usage_billing,billing_service}.go` 及 target-only `openai_account_scheduler_upstream_cost_test.go`; 159: `openai_account_scheduler_test.go`。 | 当前：`account_billing_rate_multiplier_test.go`、`openai_account_scheduler_test.go`、本轮 `openai_gateway_record_usage_test.go`; 已执行 `go -C backend test ./internal/service -run '^TestOpenAIGatewayServiceRecordUsage_AccountRateMultiplierFeedsUsageBilling$' -count=1` (0)。 | nil/0/negative multiplier、capability filter、账户 quota cost。 | protected |
| 操作审计 / 157,159 | `NewAuditLogMiddleware` <- auth/admin/user route groups; -> `AuditLogService.Record` -> audit repo。 | `server/middleware/audit_log.go`、`service/{audit_log,audit_log_service}.go`、`repository/audit_log_repo.go`、migration `180`、`handler/admin/audit_log_handler.go`。 | tag tests: `server/middleware/audit_log_test.go`、`service/audit_log_test.go`; 合并后 `go -C backend test ./internal/server/middleware ./internal/service -run 'AuditLog' -count=1`。 | 脱敏、body 回填、认证后挂载、refresh 例外。 | manual |
| 会话 IP/UA 绑定 / 157,159 | `SessionBindingContext` <- router; `enforceSessionBinding` <- JWT auth; -> `RevokeSessionFamily` + audit。 | `server/middleware/session_binding.go`、`service/session_binding.go`、`server/router.go`。 | tag tests: `session_binding_test.go`、`jwt_auth_test.go`; 合并后 `go -C backend test ./internal/server/middleware -run 'SessionBinding' -count=1`。 | trusted-proxy IP、旧 claim 放行、不匹配撤销/401。 | manual |
| step-up 2FA / 157 | `StepUp` middleware <- protected user/admin routes; `TotpHandler` -> TOTP service -> audit action。 | `server/middleware/{step_up,wire}.go`、`handler/totp_handler.go`、`service/{totp_service,totp_verification_method}.go`。 | tag tests: `step_up_test.go`、`totp_verification_method_test.go`; 合并后 `go -C backend test ./internal/server/middleware ./internal/service -run 'StepUp|Totp' -count=1`。 | scope、过期/失败、审计 action。 | manual |
| OpenAI Responses 和 WebSocket / 157,158,159 | `Responses`/`ResponsesWebSocket` <- gateway routes; -> handler concurrency/failover/usage -> WS ingress/HTTP bridge。 | 157: `handler/{gateway_handler_responses,openai_gateway_handler}.go`、`service/openai_gateway_usage.go`; 158-159: `service/openai_ws_{forwarder_ingress,forwarder_v2,http_bridge}.go`。 | 当前：`openai_gateway_service_test.go`、`openai_ws_forwarder_ingress_session_test.go`、`openai_ws_forwarder_success_test.go`、`openai_ws_http_bridge_test.go`; baseline command `go -C backend test ./internal/service ./internal/handler/... ./internal/repository ./internal/server/... -count=1` (0)。 | compact、WS V2 binding、terminal usage、lease release。 | protected |
| Grok endpoint / 157,158,159 | Grok gateway/media forwarders <- gateway routes; -> account `GetGrokBaseURL`/media URL resolution。 | 157: `service/{grok_media,grok_upstream_url}.go`; 158: `handler/admin/grok_oauth_handler.go`; 159: `service/openai_gateway_grok.go`。 | 当前：`grok_upstream_url_test.go`、`account_base_url_test.go`; 已执行 `go -C backend test ./internal/service -run '^(TestGetGrokBaseURL|TestOpenAIGatewayService_SelectAccountWithScheduler_DefaultDisabledUsesLegacyLoadAwareness|TestOpenAIAlphaSearch)' -count=1` (0)。 | OAuth/API-key endpoint、media endpoint、scheduler platform route。 | protected |
| Grok prompt cache helper / 159 | `resolveGrokCacheIdentity` <- Grok/Responses request builders; -> deterministic tenant/model cache identity and body/header helpers。 | `service/openai_gateway_grok_cache.go`。 | 当前直接 unit tests：`openai_gateway_grok_cache_test.go`; 已执行 `go -C backend test -tags unit ./internal/service -run '^(TestResolveGrokCacheIdentity.*|TestApplyGrokCacheIdentity.*|TestGrokFreeMessagesFunctionToolCacheRoute.*|TestGrokCompactRequestSkipsCacheIdentityAndNativeTools)$' -count=1` (0，19 tests)。 | API-key/model isolation、Responses body/header、Free-tier tools、compact exclusion。 | protected |
| Grok prompt cache WS/HTTP cross-entry integration / 159 | `openai_gateway_grok.go`、Messages/Chat bridge、`openai_ws_http_bridge.go` <- their protocol ingress; -> shared cache helper and upstream request transport。 | `service/{openai_gateway_grok,openai_ws_http_bridge}.go`。 | 合并后 `go -C backend test ./internal/service -run 'Grok.*Cache|WSHTTPBridge' -count=1`，并联检 Responses/Messages/Chat/WS ingress。 | 跨入口 identity、header/body 注入与 upstream transport 一致性。 | manual |
| 分组复制 / 158 | `GroupHandler.Duplicate` <- admin route; -> `DuplicateGroup` -> group repo/outbox transaction。 | `handler/admin/group_handler.go`、`service/admin_group_duplicate.go`、`repository/group_repo.go`、migration `181`。 | tag tests: `admin_group_duplicate_test.go`、`group_repo_duplicate_integration_test.go`、`admin.groups.duplicate.spec.ts`; 合并后 `go -C backend test ./internal/handler/admin ./internal/service ./internal/repository -run 'DuplicateGroup' -count=1`。 | inactive copy、deep copy、priority、idempotency、name collision。 | manual |
| 用户批量限额 / 158 | `UserHandler.BatchUpdateConcurrency` <- admin route; -> `AdminUser.BatchUpdateConcurrency` -> user repo。 | `handler/admin/user_handler.go`、`service/admin_user.go`、`repository/user_repo.go`。 | tag tests: `user_handler_batch_limits_test.go`、`admin_service_batch_limits_test.go`、`BulkEditUserModal.spec.ts`; 合并后 `go -C backend test ./internal/handler/admin ./internal/service -run 'BatchUpdateConcurrency|BatchLimits' -count=1`。 | mode、partial failure、UI payload。 | manual |
| 客户端 IP 解析 / 159 | `GetClientIP`/`GetTrustedClientIP` <- audit/session/ACL middleware。 | `pkg/ip/ip.go`、`server/{router,middleware/api_key_auth,middleware/audit_log,middleware/session_binding}.go`。 | tag tests: `ip_test.go`、`session_binding_test.go`; 合并后 `go -C backend test ./internal/pkg/ip ./internal/server/middleware -run 'IP|SessionBinding' -count=1`。 | proxy trust、port、XFF、private range order。 | manual |
| alpha/search API-key 调度 / 157,159 | `POST /alpha/search` <- gateway routes; -> handler `AlphaSearch` -> `ForwardAlphaSearch` -> usage record。 | `handler/openai_alpha_search.go`、`service/openai_alpha_search.go`、`service/openai_gateway_usage.go`。 | 当前：`openai_alpha_search_test.go`、`openai_alpha_search_billing_test.go`; baseline command above (0)。 | 仅 2xx 计一次、OAuth/API-key endpoint、failover。 | protected |
| 账号上游链接和账单探测 / 157,158 | account admin handler <- admin route; -> upstream billing probe -> account repo CAS -> account UI。 | `handler/admin/{account_handler,account_upstream_billing_probe}.go`、`service/upstream_billing_probe.go`、`repository/account_repo_upstream_billing_probe_*.go`。 | tag tests: `account_upstream_billing_probe_test.go`、`account_repo_upstream_billing_probe_*_test.go`、`admin.accounts.upstreamBillingProbe.spec.ts`; 合并后 `go -C backend test ./internal/handler/admin ./internal/service ./internal/repository -run 'UpstreamBillingProbe' -count=1`。 | CAS due-time、persistence、base URL、UI display。 | manual |
| Stripe 惰性加载 / 159 | user payment route <- `StripePaymentView`/popup; -> lazy Stripe import -> `StripePaymentInline`。 | `frontend/src/{views/user/StripePaymentView.vue,views/user/StripePopupView.vue,components/payment/StripePaymentInline.vue,views/user/__tests__/stripeLazyLoading.spec.ts,vite.config.ts}`。 | tag tests: `StripePaymentView.spec.ts`、`stripeLazyLoading.spec.ts`; 合并后 `pnpm --dir frontend exec vitest run src/views/user/__tests__/StripePaymentView.spec.ts src/views/user/__tests__/stripeLazyLoading.spec.ts`。 | first load、failure UI、popup/inline、chunk output。 | manual |
| Wire、Ent 和 migrations 177-181 / 157,158 | `make -C backend generate` <- Ent schema/Wire providers; migration runner <- numbered SQL files。 | `cmd/server/{wire.go,wire_gen.go,wire_gen_test.go}`、Ent generated schema、migrations `177`-`181`。 | current `wire_gen_test.go`; 合并后 `make -C backend generate` twice then `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go`。 | order、idempotence、provider graph。 | manual |
| 本地首 Token 超时 / 157-159 | approved removal; no reintroduced caller or watchdog. | 157-159 no `openai_first_output_timeout.go` modification. | `git diff --name-only v0.1.156^{}..v0.1.159^{} -- backend/internal/service/openai_first_output_timeout.go` must stay empty. | retain independent stream-data-interval timeout only. | approved-removal |

### 本轮保护结论
- 本轮复审前的 gap：`BillingRateMultiplier` 仅有 accessor 单测，没有 `RecordUsage -> applyUsageBilling` 的直接断言；`openai_account_scheduler_upstream_cost_test.go` 只存在于目标 v0.1.157，不存在于当前基线，不能作为当前证据。
- 关闭：在现有 `openai_gateway_record_usage_test.go` 添加 `TestOpenAIGatewayServiceRecordUsage_AccountRateMultiplierFeedsUsageBilling`。当前基线执行通过，断言 usage log snapshot 与 `UsageBillingCommand.AccountQuotaCost == TotalCost * account multiplier`；该行现为 `protected`。
- Grok cache 拆为两行：helper 是当前基线 `protected`，已实际执行 19-test 正则 `go -C backend test -tags unit ./internal/service -run '^(TestResolveGrokCacheIdentity.*|TestApplyGrokCacheIdentity.*|TestGrokFreeMessagesFunctionToolCacheRoute.*|TestGrokCompactRequestSkipsCacheIdentityAndNativeTools)$' -count=1` 并退出 0；WS/HTTP 跨入口集成是合并后的 `manual` 审查。不带 tag 的默认命令显示 `no tests to run`，不作为证据。
- 现有扩展基线：`go -C backend test ./internal/service ./internal/handler/... ./internal/repository ./internal/server/... -count=1` 退出 0；`pnpm --dir frontend exec vitest run` 退出 0（181 files, 1405 tests）。Vitest 保留既有 router-link、预期错误路径和 intlify 警告，未出现失败。

## v0.1.157
- Merge commit：`fa656646d09ce0a6207a21f9bf1149cb3bafac73`，第一父 `d77bea9c93a77f662c6218c3a83aaab05092c7f0`，第二父 `a2779cd5f30d6d3904a9d59088aed09507678dfe`。
- 31 个文本冲突均按第一父、tag 与调用方核对后融合：版本/生成物；配置与 settings；审计、step-up、session binding 路由；异步图片与对象存储；usage detail；Grok 请求头；OpenAI scheduler、alpha/search、forward、WS bridge；以及账户编辑 UI、类型、i18n 与 settings UI。
- 原生 HTTP Responses `openai_first_output_timeout.go` 及 `openai_first_output_timeout_seconds` 保留。它不是已批准删除的旧本地 first-token watchdog；后者的 `openai_text_first_token_timeout`、`openai_image_first_token_timeout`、`first_token_timeout` 错误/日志和本地 HTTP/WS watchdog 均未重新引入。
- 生成：两次 `make -C backend generate` 成功；第二次生成后 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 成功。
- 编译：`go -C backend test ./internal/... -run '^$'` 成功。Wire 缺失 provider 经 `service/wire.go` 的 `ProvideImageTaskService`、`ProvideUpstreamBillingProbeService`、`ProvideAuditLogService` 声明修复，未手改生成结果。

## v0.1.158
未开始合并。

## v0.1.159
未开始合并。

## 最终验证
- 扩展前完整基线全部退出码为 0。
- 首轮 v0.1.156 验证报告保持只读；Task 25 的报告和 characterization test 提交见下文。

## 完整命令与结果摘要
### 工作区、分支和首轮结果
```bash
git branch --show-current
git status --short
git rev-parse HEAD
git merge-base --is-ancestor 12f991dde8a58e183d4bd16a87ef6fd0df714757 HEAD
git merge-base --is-ancestor a2779cd5f30d6d3904a9d59088aed09507678dfe HEAD
```

- 分支为 `feature/20260717/staged-merge-upstream-v0-1-159`。
- HEAD 为 `df422122df428b130cb0f9b114a23bb71b59c9a8`。
- 工作区仅有忽略的 `.comet/current-change.json`。
- `v0.1.156^{}` 祖先检查退出码为 0；`v0.1.157^{}` 祖先检查退出码为非 0。

### 获取和固定 tag
```bash
git fetch upstream --tags --prune
git rev-parse v0.1.157 "v0.1.157^{}"
git rev-parse v0.1.158 "v0.1.158^{}"
git rev-parse v0.1.159 "v0.1.159^{}"
git rev-list --count "v0.1.156^{}..v0.1.157^{}"
git rev-list --count "v0.1.157^{}..v0.1.158^{}"
git rev-list --count "v0.1.158^{}..v0.1.159^{}"
git diff --name-only "v0.1.156^{}..v0.1.157^{}"
git diff --name-only "v0.1.157^{}..v0.1.158^{}"
git diff --name-only "v0.1.158^{}..v0.1.159^{}"
git log --oneline "v0.1.159^{}"..upstream/main
```

- `git fetch upstream --tags --prune` 成功。
- tag object/peeled SHA 与固定对象一致。
- 三个提交区间的提交数依次为 82、20、12；变更文件数依次为 331、105、30。
- `git log --oneline "v0.1.159^{}"..upstream/main` 已记录为排除范围，未纳入本次扩展边界。

### 扩展前完整基线
```bash
make test
pnpm --dir frontend run build
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git diff --check
```

- 初次 `make test` 在 Go 测试通过后受执行器 120 秒时限终止，未产生命令失败输出或退出结果；以 600 秒执行窗口原样重跑后退出码为 0。
- `make test` 的重跑结果：Go 测试和 `golangci-lint` 通过；前端 lint/typecheck 通过；Vitest 为 181 个测试文件、1405 个测试通过。
- `pnpm --dir frontend run build` 退出码为 0，完成 970 个模块构建。
- 两次 `make -C backend generate` 均退出码为 0；两次 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 均退出码为 0，无生成 diff。
- `git diff --check` 退出码为 0。

## 文件
- 创建：`docs/superpowers/reports/2026-07-17-staged-merge-upstream-v0-1-159-validation.md`
- 未修改：`docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`
- 未提交：`.comet/current-change.json`（忽略的本地选择文件）。

## 提交
- Task 24 baseline：`b1bd6028e docs: record v0.1.159 extension baseline`，仅活动报告。
- Task 25 matrix：`fe4037449 docs: extend capability matrix to v0.1.159`，仅活动报告。
- Task 25 gap closure：`bdeb1d1af test: protect local behavior before v0.1.157`，包含活动报告及 `backend/internal/service/openai_gateway_record_usage_test.go` 的 characterization test。
- Task 25 Grok cache evidence：`668997d77 docs: clarify Task 25 Grok cache evidence`，仅活动报告。
- 后续文档收敛提交不要求在本报告内自引用 SHA；以 `git log --oneline b1bd6028e..HEAD` 为权威范围，不修改首轮报告。

## 自审
- 固定对象、提交计数、变更文件计数和最终边界与任务 brief 一致。
- 未合并 v0.1.157、v0.1.158 或 v0.1.159；未 push、release 或 deploy。
- 首轮 v0.1.156 验证报告未修改；`.comet/current-change.json` 未加入提交。
- 已复核首轮报告无 diff；已知 Task 25 提交为 `fe4037449`、`bdeb1d1af`、`668997d77`，其中 `bdeb1d1af` 含最小 characterization test。后续文档收敛以指定 Git range 核验，不含 `.comet/current-change.json`。

## 残余风险与未执行事项
- 本任务仅固定 refs 和建立扩展前基线；v0.1.157、v0.1.158、v0.1.159 的合并和能力验证均未执行。
- 构建保留现有 Browserslist 数据过期、动态导入与 chunk 大小告警；本次命令均成功，未将其作为本任务范围内的修复项。
