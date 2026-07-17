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
- 31 个文本冲突均已按第一父、tag 与调用方核对后消除并记录。30 个冲突完成两侧融合；`backend/internal/server/routes/gateway.go` 的同步路由链与 handler 注入已在 merge 中融合，但 async 公共 route 注册遗漏已转交用户授权的 Task 27 early work。
- 原生 HTTP Responses `openai_first_output_timeout.go` 及 `openai_first_output_timeout_seconds` 保留。它不是已批准删除的旧本地 first-token watchdog；后者的 `openai_text_first_token_timeout`、`openai_image_first_token_timeout`、`first_token_timeout` 错误/日志和本地 HTTP/WS watchdog 均未重新引入。
- 生成：两次 `make -C backend generate` 成功；第二次生成后 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 成功。
- 编译：`go -C backend test ./internal/... -run '^$'` 成功。Wire 缺失 provider 经 `service/wire.go` 的 `ProvideImageTaskService`、`ProvideUpstreamBillingProbeService`、`ProvideAuditLogService` 声明修复，未手改生成结果。
- Task 26 边界仅包括上述 merge topology、31-entry 冲突融合台账以及编译/生成检查；不将 merge 后行为修复计入该任务。

### v0.1.157 冲突台账

| 路径 | 类别 | ours 语义 | theirs 语义 | 最终融合 | 验证命令 |
| --- | --- | --- | --- | --- | --- |
| `backend/cmd/server/VERSION` | version | 本地四段开发号 | tag 版本号 | tag 版本号 | `git diff --check` |
| `backend/cmd/server/wire_gen.go` | generated Wire | startup recovery、image history | audit、probe、async image 注入 | 由 Wire 声明重新生成 | `make -C backend generate` |
| `backend/internal/config/config.go` | config | layered scheduler | upstream cost、image storage | 两套配置和校验并存 | `go -C backend test ./internal/config` |
| `backend/internal/handler/admin/setting_handler.go` | settings read | tri-state agreement、sticky/layered | session binding、audit、cost settings | DTO 返回两侧字段 | `go -C backend test ./internal/handler/admin` |
| `backend/internal/handler/admin/setting_handler_update.go` | settings write | tri-state agreement、runtime controls | session/audit/cost 更新 | 保留两侧写入语义 | `go -C backend test ./internal/handler/admin` |
| `backend/internal/handler/grok_media.go` | Grok handler | usage request detail | account header overrides | 记录并覆写最终请求 | `go -C backend test ./internal/handler` |
| `backend/internal/handler/openai_embeddings.go` | scheduler caller | platform-aware selection | model-scoped selection result | 两种调度参数兼容 | `go -C backend test ./internal/handler` |
| `backend/internal/handler/openai_gateway_handler.go` | gateway handler | usage snapshot、local guards | model terminal scheduling | 两侧结果记录 | `go -C backend test ./internal/handler` |
| `backend/internal/handler/openai_gateway_handler_test.go` | handler tests | local Responses/WS assertions | upstream async/lease assertions | 保留测试覆盖 | `go -C backend test ./internal/handler` |
| `backend/internal/repository/usage_log_repo_query.go` | usage query | detail-presence columns | image input token columns | select 列并集 | `go -C backend test ./internal/repository` |
| `backend/internal/server/http.go` | router DI | user service | audit/step-up middleware | 注入全部依赖 | `go -C backend test ./internal/server` |
| `backend/internal/server/router.go` | route middleware | embedded frontend/user pages | session binding/audit/step-up | 中间件与路由调用并集 | `go -C backend test ./internal/server` |
| `backend/internal/server/routes/gateway.go` | gateway routes | usage-detail chain | async image endpoint 代码路径 | merge 融合保留公共同步路由链；遗漏的公共 async route 注册移交 Task 27 early | `go -C backend test ./internal/server/routes -run '^TestGatewayRoutesAsyncImagesPathsAreRegistered$' -count=1` |
| `backend/internal/service/admin_service.go` | admin DI | runtime blocker/probe control | affiliate accrual | 构造函数保留两侧依赖 | `go -C backend test ./internal/service` |
| `backend/internal/service/grok_media.go` | Grok forward | multipart usage snapshot | account header overrides | 两侧请求处理顺序 | `go -C backend test ./internal/service` |
| `backend/internal/service/openai_account_scheduler.go` | scheduler | layered/privacy/sticky recovery | upstream-cost/model transient | 选择、重检和 cost 支撑并集 | `go -C backend test ./internal/service` |
| `backend/internal/service/openai_alpha_search.go` | alpha search | request snapshots | PAT Responses fallback | 构建与 fallback 并存 | `go -C backend test ./internal/service` |
| `backend/internal/service/openai_gateway_forward.go` | forward errors | response snapshots | normalized failover error | 保留 failover 与 detail | `go -C backend test ./internal/service` |
| `backend/internal/service/openai_gateway_grok.go` | Grok request | usage snapshot | final header overrides | 两者均执行 | `go -C backend test ./internal/service` |
| `backend/internal/service/openai_gateway_passthrough.go` | passthrough error | request-builder compatibility | body-too-large/model transient | 两侧 error policy | `go -C backend test ./internal/service` |
| `backend/internal/service/openai_gateway_scheduling.go` | scheduling support | layered recovery hooks | group-aware recheck | 两侧 account recheck | `go -C backend test ./internal/service` |
| `backend/internal/service/openai_gateway_service.go` | result/service state | usage snapshot cloning | WS terminal/model state | result fields并集 | `go -C backend test ./internal/service` |
| `backend/internal/service/openai_ws_http_bridge.go` | WS bridge | bridge detail handling | terminal event tracking | 两侧 terminal state | `go -C backend test ./internal/service` |
| `backend/internal/service/wire.go` | provider declarations | startup recovery/probe bind | image/audit/probe providers | 声明后生成 Wire | `make -C backend generate` |
| `deploy/config.example.yaml` | sample config | layered scheduler | upstream cost | 配置示例并集 | `git diff --check` |
| `frontend/src/components/account/EditAccountModal.vue` | account edit | passthrough/quota extras | upstream billing probe extras | 合并 extra 写入 | `pnpm --dir frontend run build` |
| `frontend/src/components/account/__tests__/CreateAccountModal.spec.ts` | frontend tests | passthrough cases | OAuth/import cases | 两侧测试 helpers/cases | `pnpm --dir frontend exec vitest run src/components/account/__tests__/CreateAccountModal.spec.ts` |
| `frontend/src/i18n/locales/en/admin/accounts.ts` | i18n | passthrough labels | custom Grok URL labels | message keys并集 | `pnpm --dir frontend run build` |
| `frontend/src/i18n/locales/zh/admin/accounts.ts` | i18n | passthrough labels | custom Grok URL labels | message keys并集 | `pnpm --dir frontend run build` |
| `frontend/src/types/index.ts` | frontend types | passthrough/quota extras | upstream billing probe types | Account extra 类型并集 | `pnpm --dir frontend run build` |
| `frontend/src/views/admin/SettingsView.vue` | settings UI | sticky/layered settings | upstream rate settings | form defaults并集 | `pnpm --dir frontend run build` |

### Task 27 early work
- 用户明确保留未推送历史、不重建 `fa656646d` merge。review 在 merge 后发现失败的 async route 测试，故按全局设计采用独立普通修复提交，而非改写 merge 历史。
- `e3b0c15b1 fix: register async image gateway routes` 是用户授权的 Task 27 early merge-after semantic fix，不属于 Task 26 merge 或台账提交。
- 它恢复六条 `/v1` 和 alias async image/task routes，并沿用同步 Images 的 `bodyLimit`、request ID、usage detail、ops、endpoint、API-key 和 group middleware chain。
- RED: `go -C backend test ./internal/server/routes -run 'AsyncImage|ImageTask|images.*async' -count=1` failed because `POST /v1/images/generations/async` was absent.
- GREEN: `go -C backend test ./internal/server/routes -run '^TestGatewayRoutesAsyncImagesPathsAreRegistered$' -count=1` and `go -C backend test ./internal/handler -run '^TestAsyncImageHandler' -count=1` passed.

### Task 27 regression review and repair
- 比较 `fa656646d^1`、`fa656646d^2` 与当前分支后，确认以下失败来自 merge 后本地融合路径，而非 v0.1.157 tag 基线：Anthropic OAuth mimic 默认 system block、OpenAI scheduler top-K/已满溢出、Grok Responses image declaration gate、key billing route、settings 读写和 SettingsView 绑定。
- 最小修复保留默认 OAuth mimic 的 billing + Claude Code 两块 system 形态；scheduler 在 DB recheck 未加载 `GroupIDs` 时不把关系未知误判为组外，同时继续拒绝已加载但不匹配的关系；Grok 仅把明确图片请求送入图片权限门；恢复 `/v1/sub2api/billing`；补齐 scheduler setting 的 handler DTO/持久化和 SettingsView payload/控件。
- `localesMessageCompile.spec.ts` 原始失败为 `Failed to resolve import "@intlify/message-compiler"`。该包已由 `vue-i18n` 锁定为 transitive `9.14.5`，现作为相同版本的直接 dev dependency 声明，供 pnpm isolated layout 下的测试解析。

#### Task 27 定向 RED/GREEN
- RED 后 GREEN：`go -C backend test ./internal/service -run '^(TestGatewayService_AnthropicOAuthMimic_RewritesSystemWithBillingBlock|TestOpenAIGatewayService_SelectAccountWithScheduler_LoadBalanceTopKFallback|TestAdvancedCostSchedulerUsesTopKOverflowWhenPreferredAccountIsKnownFull|TestAdvancedCostSchedulerKeepsCompactSupportedOverflowAheadOfUnknown)$' -count=1` 退出 0。
- RED 后 GREEN：`pnpm --dir frontend exec vitest run src/views/admin/__tests__/SettingsView.spec.ts -t 'submits the admin recharge affiliate rebate setting|places and explains rate controls for both scheduling modes'` 退出 0。

#### Task 27 结构审查
- 异步图片：route 位于 API-key/group middleware 之后；`AsyncImageHandler` 以 `UserID + APIKeyID` 限制任务 owner，并以对象存储启用作为总开关，避免 base64 写入 Redis。Wire 生成物注册 Redis task store、image storage、task service 和 handler。
- 图片计费：migration `178` 增加 `channel_model_pricing.image_input_price`（缺省回退文本输入价），`179` 增加 usage log 图片 token/cost 字段；billing、usage DTO 和 repository 高风险测试通过。
- 安全路径：audit middleware 捕获受限大小 body 后脱敏并回填；router 安装 session binding context；admin JWT 在权限判断前执行 session binding；step-up 对管理员 API key、未启用 TOTP、授权查询错误均 fail-closed。
- Responses/Grok/Agent Identity：Responses HTTP/WS 统一使用 platform-aware image intent；Grok 的 namespace/Responses Lite declarations 本身不再触发图片权限，明确图片 model/native tool/choice 仍触发。first-output spool 先 unlink，再在 cleanup 失败时不把已提交流改为失败。Wire providers 将 Agent Identity websocket service 注入 quota、usage 和 account-test service。
- migrations `177` 至 `180` 均存在且顺序连续，覆盖 subscription currency、图片输入定价、图片 usage 账单和 append-only audit logs。

#### Task 27 高风险回归命令
- `go -C backend test ./internal/service -run 'Image|Billing|Scheduler|Audit|StepUp|Session|FirstOutput|WebSocket' -count=1` 退出 0。
- `go -C backend test ./internal/handler/... -run 'Image|Billing|Audit|StepUp|Account|Setting' -count=1` 退出 0。
- `go -C backend test ./internal/repository -run 'Image|Billing|Scheduler|Audit|Probe|ChannelMonitor' -count=1` 退出 0。
- `go -C backend test ./internal/server/... -run 'Audit|Session|StepUp|APIKey|ImageTask' -count=1` 退出 0。
- `pnpm --dir frontend exec vitest run src/components/account/__tests__/UpstreamBillingRateCell.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/ChannelMonitorView.duplicate.spec.ts src/i18n/__tests__/localesMessageCompile.spec.ts` 退出 0，4 files / 33 tests 通过。
- 前端测试仍输出既有 `router-link` stub、jsdom XHR 和 Browserslist data 过期警告；无失败断言，未扩大本任务范围修复测试基建告警。
- 前端类型契约补充：`pnpm --dir frontend run typecheck` 曾退出 2。`git diff d77bea9c9..a2779cd5f` 证明 upstream billing probe、图片输入 usage 和 session/audit settings 是 v0.1.157 引入的契约；`d77bea9c9` 则保留 EditAccountModal 仍调用的 header override platform/template helpers。当前 merge 将两侧契约遗漏，故不是 pre-existing failure。恢复 probe/image types、local helpers/templates 以及 session/audit UI/default/payload 后，`pnpm --dir frontend run typecheck` 退出 0；`UpstreamBillingRateCell.spec.ts` 与 `SettingsView.spec.ts` 共 26 tests 通过。

#### Task 27 follow-up validation
- Fresh `go -C backend test ./internal/service -run 'Image|Billing|Scheduler|Audit|StepUp|Session|FirstOutput|WebSocket' -count=1` 退出 0。
- Fresh `go -C backend test ./internal/handler/... -run 'Image|Billing|Audit|StepUp|Account|Setting' -count=1` 退出 0。
- Fresh `go -C backend test ./internal/repository -run 'Image|Billing|Scheduler|Audit|Probe|ChannelMonitor' -count=1` 退出 0。
- Fresh `go -C backend test ./internal/server/... -run 'Audit|Session|StepUp|APIKey|ImageTask' -count=1` 退出 0；`internal/server` 无测试文件，`internal/server/routes` 无匹配测试。
- Fresh `pnpm --dir frontend exec vitest run src/components/account/__tests__/UpstreamBillingRateCell.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/ChannelMonitorView.duplicate.spec.ts src/i18n/__tests__/localesMessageCompile.spec.ts` 退出 0，4 files / 33 tests 通过。
- Fresh `pnpm --dir frontend run typecheck` 退出 0。

#### Task 27 final repair and validation
- `15e5ff41b fix: preserve local behavior after v0.1.157` 恢复 admin settings DTO 中被 merge 遗漏的 session binding、audit retention、OpenAI scheduler rate 和 upstream-cost 字段；`go test -tags=unit ./internal/server -run '^TestAPIContract' -count=1` 由 RED 转 GREEN。
- 同一提交恢复 OpenAI API Key 上游倍率自动探测开关、持久化和中英文 locale keys；未配置的 false 默认值不会在无关账号保存时新增 `extra` 字段。`localeKeysExist.spec.ts` 与 `EditAccountModal.spec.ts` 共 57 个测试由 RED 转 GREEN。
- 最终 `make test` 退出 0：后端测试和 lint、前端 lint/typecheck，以及 189 个 Vitest 文件 / 1454 个测试全部通过；`pnpm --dir frontend run typecheck` 退出 0。
- 未合并 v0.1.158/v0.1.159，未 push、release 或 deploy；`.comet/current-change.json` 已在收尾前删除且未提交。

#### Task 27 OAuth scheduler zero follow-up
- `5e5afd7f6 fix: preserve zero OAuth scheduler rate` 将 SettingsView 的 OAuth scheduler reference multiplier 序列化从 truthy fallback 改为有限数值检查：合法 `0` 原样提交，仅 `NaN` 或无穷值回退为 `1`。
- RED：输入 `0` 后，`pnpm exec vitest run src/views/admin/__tests__/SettingsView.spec.ts -t "preserves a zero OAuth scheduling reference multiplier"` 观察到 payload 错误提交 `1`。GREEN：同一命令通过；完整 `SettingsView.spec.ts` 为 23/23，`pnpm run typecheck` 退出 0。
- 本次两文件 follow-up 未重新运行 `make test`；此前 Task 27 full gate 为 `make test` 退出 0（189 个 Vitest 文件 / 1454 个测试）。自审确认未扩大功能范围、未合并 v0.1.158/v0.1.159，`.comet/current-change.json` 未提交。

#### Task 27 OAuth scheduler empty-value follow-up
- `c2b6a9c05 fix: default empty OAuth scheduler rate` 将序列化收紧为仅保留有限的 `number`。因此合法 `0` 保持不变，而 `v-model.number` 清空输入产生的 `""`、`NaN` 和无穷值均回退为 `1`。
- RED：`pnpm exec vitest run src/views/admin/__tests__/SettingsView.spec.ts -t "defaults an empty OAuth scheduling reference multiplier"` 在清空输入后观察到 payload 错误提交 `0`。GREEN：`pnpm exec vitest run src/views/admin/__tests__/SettingsView.spec.ts -t "OAuth scheduling reference multiplier"` 同时通过零值和空值两项；完整 `SettingsView.spec.ts` 为 24/24，`pnpm run typecheck` 退出 0。
- 本次两文件 follow-up 未重新运行 `make test`；此前 Task 27 full gate 仍为 `make test` 退出 0（189 个 Vitest 文件 / 1454 个测试）。未合并 v0.1.158/v0.1.159，未 push、release 或 deploy。
- 自审：`.comet/current-change.json` 当前为未跟踪文件，按最新用户指令保留；未加入任何提交。

### Task 28 v0.1.157 阶段门禁
- 起始提交 `e41ae036f21cde0fff0ec4086da4ddd2c1b49849`；仅关闭 v0.1.157 门禁，未开始 v0.1.158/v0.1.159 合并，未 push、release 或 deploy。
- `make test` 退出 0：后端 Go 测试与 `golangci-lint` 通过；前端 lint/typecheck 通过；Vitest 为 189 个测试文件、1456 个测试通过。测试输出保留既有 `router-link`、预期错误路径、intlify 和 Browserslist 数据过期警告，无失败断言。
- `pnpm --dir frontend run build` 退出 0：983 个模块完成生产构建（32.74 秒）。保留既有 Browserslist 数据过期、动态导入 chunk 和大于 500 kB chunk 告警。
- `make -C backend generate` 退出 0；`git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 无输出、退出 0，Ent/Wire 无生成 diff。
- `git diff --check` 和 `git diff --name-only --diff-filter=U` 均无输出、退出 0；冲突标记 `git grep` 无匹配输出（无匹配时 `git grep` 返回 1），未发现未合并文件或真实冲突标记。
- 能力矩阵无 `gap`：当前行为已由 `protected` 断言覆盖，目标 tag 新入口仍按既定 `manual` 合并后验收，已批准移除项维持 `approved-removal`。Task 29 获准开始 v0.1.158 阶段。

## v0.1.158
- Merge commit：`be00309dd72cb42c2c1ab1769dd949a35c625f29`，第一父 `0cb9e52bdbfb9045dd0aacc77028ad04aed12a78`，第二父 `26abd19a2812edba02bbef93c3e2a620141cc257`。
- 未合并 `upstream/main`、v0.1.159；未 push、release 或 deploy。`.comet/current-change.json` 保持未跟踪，未加入 merge 或本文档提交。
- merge 前的 9 个文本冲突均核对第一父、tag 和相关调用路径后最小融合；无 BLOCKED 项。原生 HTTP `first-output` 实现继续保留，旧本地 first-token watchdog 未重新引入。

### v0.1.158 冲突台账

| 路径 | 类别 | ours 语义 | theirs 语义 | 最终融合 | 验证 |
| --- | --- | --- | --- | --- | --- |
| `backend/cmd/server/VERSION` | version | 本地四段开发号 | tag 版本号 | 使用 tag `0.1.157` | `git diff --cached --check` |
| `backend/cmd/server/wire_gen.go` | generated Wire | 现有 runtime/probe 依赖 | `AdminGroupRepository` 注入 | 以 `wire.go` 声明重生，保留两类依赖 | 连续两次 `make -C backend generate` |
| `backend/ent/mutation.go` | generated Ent | 用户并发字段容量 | group duplicate operation ID 字段 | 以合并 schema 重生，容量 51 | 连续两次 `make -C backend generate` |
| `backend/ent/runtime/runtime.go` | generated Ent | 用户并发字段索引 | duplicate operation ID 插入字段索引 | 以合并 schema 重生，全部后续索引右移 | 连续两次 `make -C backend generate` |
| `backend/internal/service/grok_media.go` | Grok media forward | multipart edit body 转 JSON | 仅 CLI proxy target 追加 Grok CLI headers | 两者保留；非 CLI upstream 不注入 CLI headers | service 定向测试、`make test` |
| `backend/internal/service/openai_ws_forwarder_ingress.go` | Responses WS ingress | 首个 response ID 绑定并过滤非归属事件 | 图片生成 completed status 归一化 | 在事件 envelope 解析前归一化，并保留 response 归属过滤 | WS 定向测试、`make test` |
| `backend/internal/service/openai_ws_forwarder_ingress_session_test.go` | WS regression | `store:false` 与创建事件断言 | image-generation terminal status 断言 | 同时保留 create/store 语义和 normalized completed image 断言 | service 定向测试、`make test` |
| `frontend/src/api/__tests__/admin.users.spec.ts` | admin users API tests | 用户创建 username/notes 请求断言 | bind identity/批量限额类型与请求断言 | 三组契约测试并存 | 定向 Vitest、`make test` |
| `frontend/src/components/account/EditAccountModal.vue` | account editor | local passthrough、quota、header editor 和默认 URL | Grok OAuth custom upstream URL/header overrides | 保留 local 编辑能力；Grok OAuth 控件置于 API-key 容器外，复用 header editor | Grok editor 定向 Vitest、typecheck、build、`make test` |

### v0.1.158 验证
- `go -C backend test ./internal/service ./internal/handler/admin ./internal/repository -run 'DuplicateGroup|BatchUpdateConcurrency|BatchLimits|Grok|ProxyResponses' -count=1` 通过。
- `pnpm --dir frontend exec vitest run src/api/__tests__/admin.users.spec.ts src/components/account/__tests__/EditAccountModal.grokUpstream.spec.ts src/components/account/__tests__/EditAccountModal.spec.ts src/components/admin/user/__tests__/BulkEditUserModal.spec.ts` 通过：4 files / 70 tests。
- `go -C backend test ./internal/... -run '^$'`、`pnpm --dir frontend run typecheck`、`pnpm --dir frontend run build` 和 `make test` 全部通过；`make test` 为 193 个 Vitest 文件 / 1488 个测试。
- `make -C backend generate` 连续两次成功，第二次后 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 通过；`git diff --cached --check` 和未合并路径检查均无输出。
- 上游新增 `BatchUpdateLimits` 接口后，第一父的 `adminAPIKeyBlockedUserRepo` fake 缺少该方法，补入仅 panic 的 stub 以恢复接口编译；这不是 Task 30 行为修复。

### Task 30 v0.1.158 能力复审
- `cce846c5e7f7782e8f019b84ccab6baa2691f2ce` 是业务代码复审快照和 Task 29 ledger；`a60b02c8a` 是初始 Task 30 报告提交。本次为后续 docs correction，不在本文自引用完整提交列表。未合并 v0.1.159，未 push、release 或 deploy；未跟踪的 `.comet/current-change.json` 未修改、未提交。
- brief 所列 broad service、handler、repository 和七文件 Vitest 命令均已执行，七文件 Vitest 为 7 files / 38 tests；但 handler broad 正则的 `GroupDuplicate|BatchLimits` 不匹配实际 `TestDuplicateGroupHandler...` / `TestUserHandlerBatchUpdateLimits...`，不作为这两组 handler 证据。
- 精确 handler 覆盖：`go -C backend test -v -tags unit ./internal/handler/admin -run '^(TestDuplicateGroupHandler(ReturnsAdminDTOWithoutOperationMetadata|RejectsInvalidID|ReplaysSameIdempotencyKey|RecoversAfterMarkSucceededFailure)|TestUserHandlerBatchUpdateLimits(AcceptsPartialAndZeroValues|RejectsInvalidRequests|AllUsesEveryListedUser))$' -count=1` 通过，7 个顶层测试及 10 个命名子测试均通过。补充的 `-tags unit` service group duplicate/batch limits 7 tests，以及既有 Grok/Codex/WS 和四文件前端专项均通过。
- repository/migration 实际证据仅为默认 tag 的 runner 测试：`go -C backend test -v ./internal/repository -run '^(TestLatestMigrationBaseline|TestValidateMigrationExecutionMode|TestApplyMigrations_NilDB|TestApplyMigrations_DelegatesToApplyMigrationsFS|TestApplyMigrationsFS_(NonTransactionalMigration|TransactionalMigration))$' -count=1` 通过，6 个顶层测试及 9 个命名子测试均通过。它们不执行 live PostgreSQL migration。`TestCreateGroupFromSourceRollsBackWhenOutboxInsertFails`、`TestUserRepoSuite` 的 batch-limit cases、`TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 均为未执行的 `integration` tag，保留为 manual。
- 静态无冲突语义审查：migration `181` 的 nullable operation ID 与 partial unique index 同 Ent schema 一致；复制操作按 actor/source/idempotency key 隔离，`CreateFromSource` 的 group、bindings 和 outbox 事务边界经静态审查。批量限额构造单条参数化 `UPDATE`，成功后才失效认证缓存。Grok OAuth 保留官方凭据生命周期端点、使用配置的转发端点，并保证 CLI/cache/header override 顺序。WS v2/ingress/HTTP bridge/passthrough 均在解析、计费和下发前仅把带结果的终态 image item 归一化为 `completed`。Codex manifest 的账号切换、上游校验、ETag 和原样模型能力发现未受影响。DataTable 保留跨页受控选择、可见行 select-all 和换页缓存清理。
- 无业务代码修复提交。Task 29 的 `make test` 结果未作为本项能力审查证据。

## v0.1.159
### Task 32 merge
- 起始提交：`436e1f1cd14e650d460f6b1ceb431b055d467d5a`。
- merge commit：`00517cf860cbb328ecaaae0d56bb59f1848d13ec`，第一父为起始提交，第二父为固定 peeled tag commit `2a75d7d2387587d86ca3c5e5cd8ca96cf3d104c6`。
- 仅执行 `git merge --no-ff v0.1.159 -m "merge: upstream v0.1.159"`；未合并 `upstream/main` 或 tag 后提交，未 push、release 或 deploy。
- 4 个文本冲突均逐项核对第一父、tag 和调用方后融合；原生 HTTP first-output 实现保留，旧本地 first-token watchdog 未恢复。

### v0.1.159 冲突台账

| 路径 | ours 语义 | tag 语义 | 最终融合 | 调用方/验证 |
| --- | --- | --- | --- | --- |
| `backend/internal/server/router.go` | 会话绑定上下文已在全局中间件链安装，并保留本地 embedded frontend 与 user service 路由依赖。 | `SessionBindingContext` 需要配置以读取可信反代 IP 开关。 | 以 `SessionBindingContext(cfg)` 安装，保留其余第一父路由装配。 | `SetupRouter`；`go -C backend test -tags unit ./internal/pkg/ip ./internal/server/middleware -run 'IP|SessionBinding' -count=1`。 |
| `backend/internal/service/openai_alpha_search.go` | 保留原请求体供 passthrough/header 兼容，以及既有 PAT fallback。 | API-key 上游的 `404/405` 作为 alpha/search 不支持而换号，且 `401/404/405` 不写账号错误状态。 | 保留 `sourceBody` 调用契约，加入 endpoint-unavailable 判定及 tag 的无副作用状态集。 | `ForwardAlphaSearch` gateway 调度入口；`go -C backend test ./internal/service -run 'AlphaSearch|Grok.*Cache|WSHTTPBridge' -count=1`。 |
| `frontend/src/i18n/locales/en/admin/accounts.ts` | Codex 导入文案覆盖 OAuth 与 Agent Identity，并解释动态签名。 | 新增 Mobile RT/AT 的手动输入标签。 | 保留 Agent Identity 文案，采用 tag 的两项手动输入标签，移除重复键。 | `OAuthAuthorizationFlow.vue`；前端 typecheck 与完整 Vitest。 |
| `frontend/src/i18n/locales/zh/admin/accounts.ts` | 同英文 locale 的 Agent Identity 文案与提示。 | 同英文 locale 的 Mobile RT/AT 手动输入标签。 | 同英文 locale 的融合，移除重复键。 | `OAuthAuthorizationFlow.vue`；前端 typecheck 与完整 Vitest。 |

### v0.1.159 验证
- `git diff --name-only --diff-filter=U`、`git diff --check` 均无输出；merge 二父为 `2a75d7d2387587d86ca3c5e5cd8ca96cf3d104c6`。
- `git diff --name-only v0.1.156^{}..HEAD -- backend/internal/service/openai_first_output_timeout.go` 无输出；未恢复旧本地 first-token watchdog。
- `make test` 退出 0：后端测试与 lint、前端 lint/typecheck、Vitest 194 files / 1493 tests 通过。
- `pnpm --dir frontend run build` 退出 0，987 个模块完成构建。保留既有 Browserslist、dynamic import 与大 chunk 警告。

### Task 33 v0.1.159 能力复审
- 起始提交：`9d2d4a8e4`。仅审查 v0.1.159 合并后的高风险能力；未修改业务或测试代码，未开始 Task 34/full gate，未合并 `upstream/main` 或 tag 后提交，未 push、release 或 deploy。
- trusted proxy/IP：`GetSecurityClientIP` 以 `TrustForwardedIPForAPIKeyACL` 为唯一开关；`SessionBindingContext(cfg)` 注入的结果供会话哈希、session mismatch 审计、常规 audit 和 API-key ACL 共用。关闭开关时使用 Gin `trusted_proxies` 链，开启时使用转发头；缺失注入时保持历史 trusted-proxies 回退。
- alpha/search：OpenAI API-key 账号可参与 `alpha_search` 调度，显式 `chat_completions` 能力同样允许；Grok 仍排除。API-key 上游 404/405 触发 failover 且不写账号错误状态，OAuth 404 保持透传；仅 2xx 返回 `WebSearchCalls: 1`，非 2xx 不计费。
- Grok/图片/前端：Responses 对已知 Free OAuth 的合格 function tools 在 tenant/model 隔离 cache identity 后补全 `web_search`/`x_search`；非合格工具、付费/未知/API-key 不变。图片意图、API-key `base_url` 的安全 origin 链接、三个 Stripe `@stripe/stripe-js/pure` 动态 import 和 `vendor-stripe` chunk 均保持预期行为。
- 结论：未发现能力回归，无业务修复提交。

#### Task 33 精确测试

```text
go -C backend test ./internal/pkg/ip ./internal/server/middleware -count=1
PASS (ip package has no default-tag tests; middleware passed)

go -C backend test ./internal/service -run 'AlphaSearch|Scheduler|Grok|Image|Account' -count=1
PASS

pnpm --dir frontend exec vitest run src/views/admin/__tests__/AccountsView.sparkShadow.spec.ts src/views/user/__tests__/StripePaymentView.spec.ts src/views/user/__tests__/stripeLazyLoading.spec.ts
PASS (3 files, 13 tests)
```

- 补充 `-tags unit` IP/session/API-key/audit、alpha/search side-effect、Grok cache identity/function-tool 和图片定向测试均通过。

#### Task 33 manual/残余项
- 生产反代拓扑下的可信代理配置、第三方 API-key relay 对 alpha/search 404/405 的实际行为，以及 Grok Free OAuth 的真实上游 cache 命中仍需运行环境人工验收。
- Task 34/full gate 未执行；本项复审不能替代完整 Go lint、前端 typecheck/build 或全量 Vitest 门禁。
- 前端定向测试保留既有 Browserslist data 过期警告，无失败断言。

### Task 34 v0.1.159 阶段门禁
- 起始提交：`80750db5de771eebf3cb6f1a8b2438f21adfbed9`。仅关闭 v0.1.159 阶段门禁；未开始 Task 35 版本规范化，未提交 `.comet/current-change.json`，未合并 `upstream/main`、tag 后提交，未 push、release 或 deploy。
- `make test` 退出 0：后端 Go 测试和 `golangci-lint` 通过；前端 lint/typecheck 通过；Vitest 为 194 个测试文件、1493 个测试通过。输出中的 `router-link`、预期错误路径、intlify、jsdom XHR 和 Browserslist 提示均无失败断言。
- `pnpm --dir frontend run build` 退出 0：`vue-tsc -b` 和 Vite production build 通过，完成 987 个模块（35.18 秒）。保留既有 Browserslist 数据过期、动态导入 chunk 与大于 500 kB chunk 警告。
- `make -C backend generate` 退出 0；随后 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 无输出，Ent/Wire 无生成 diff。
- `git diff --check` 和 `git diff --name-only --diff-filter=U` 均无输出；冲突标记扫描无真实匹配。旧本地首 Token 超时符号扫描无匹配，未恢复已批准删除的 watchdog。
- 能力矩阵无 `gap`，既有 `protected` 证据保持有效；目标 tag 新入口继续保留为合并后 `manual` 验收。残余 PostgreSQL integration manual：live PostgreSQL 未运行，`TestCreateGroupFromSourceRollsBackWhenOutboxInsertFails`、`TestUserRepoSuite` 的 batch-limit cases 和 `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 仍受 `integration` tag/数据库环境约束，需在具备 PostgreSQL 的环境执行。

### Task 31 v0.1.158 阶段门禁
- 起始提交：`1f51f4a382afb2422beae1ef4ad2bd7b5df488ee`；仅关闭 v0.1.158 阶段门禁，未合并 v0.1.159，未 push、release 或 deploy。
- `make test` 退出 0：后端 Go 测试与 lint 通过；前端 lint/typecheck 通过；Vitest 为 193 个测试文件、1488 个测试通过。
- `pnpm --dir frontend run build` 退出 0：987 个模块完成生产构建。保留既有 Browserslist 数据过期、动态导入 chunk 和大于 500 kB chunk 警告，无构建失败。
- `make -C backend generate` 退出 0；`git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 无输出、退出 0，Ent/Wire 无生成 diff。
- `git diff --check` 和 `git diff --name-only --diff-filter=U` 均无输出、退出 0；未发现空白错误或未合并路径。
- 能力矩阵无 `gap`：既有 `protected` 覆盖保持有效，目标 tag 的新增入口继续保留为 `manual` 合并后验收，已批准移除项保持 `approved-removal`。Task 32 获准开始 v0.1.159 阶段。
- `.comet/current-change.json` 保持未跟踪、未修改、未提交；本次仅提交本活动报告。

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
- Task 24 当时仅固定 refs 和建立扩展前基线；后续已完成 v0.1.157、v0.1.158、v0.1.159 合并与分段验证。Task 33 已关闭 v0.1.159 能力复审，Task 34/full gate 仍未执行。
- 构建保留现有 Browserslist 数据过期、动态导入与 chunk 大小告警；本次命令均成功，未将其作为本任务范围内的修复项。

## Task 35 版本规范化、生成物与 migration 复核
- 起始提交：`4f97cdf44d6052c2b8e43e38f72ac3e10f1816de`。仅执行本节所列版本、生成物、依赖/配置和 migration 复核；未开始 Task 36 full capability matrix，未提交 `.comet/current-change.json`，未合并 `upstream/main` 或 tag 后提交，未 push、release 或 deploy。
- `backend/cmd/server/VERSION` 是嵌入到 server 二进制的 runtime 版本源，已精确规范为 `0.1.159.1`；`go -C backend test ./cmd/server -count=1` 退出 0。版本提交为 `fa03e00a5 chore: set version to 0.1.159.1`。

### 生成物
- 连续两次 `make -C backend generate` 均退出 0；每次后 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 均退出 0。Ent 和 Wire 生成物无漂移，未手工修改生成输出。

### 依赖与配置
- 相对 `v0.1.156^{}` 的审查范围中，`backend/go.mod` 和 `pnpm-lock.yaml` 无变更；`backend/go.sum` 仅新增既有间接解析的 `github.com/google/subcommands v1.2.0` 与 `github.com/inconshreveable/mousetrap v1.1.0` 校验和。
- `frontend/package.json` 直接声明 `@intlify/message-compiler@9.14.5`（与 `vue-i18n@^9.14.5` 对齐）和 `vue-eslint-parser@^9.4.3`；lockfile 的现有解析已覆盖前者，故没有 lockfile 差异。
- `deploy/config.example.yaml` 的可见默认值经过复核：非流式上游响应读取上限为 128 MiB；各平台 sticky 默认启用；weighted scheduler 和 reset/quota-headroom 默认保持关闭；layered 参数有确定默认值；旧 `prefer_soonest_reset` 与 billing minimum reserve 示例项被移除。`git diff --check -- backend/migrations deploy/config.example.yaml` 退出 0。

### Migrations 177-181
- `git ls-tree -r --name-only HEAD backend/migrations` 列出 231 个 SQL 文件，完整文件名无重复；最后五个按字典序为 `177_add_subscription_plan_currency.sql`、`178_channel_image_input_price.sql`、`179_usage_log_image_input_tokens.sql`、`180_audit_logs.sql`、`181_group_duplicate_operation_id.sql`。`migrations.FS` 嵌入 `*.sql`，runner 对完整文件名 `sort.Strings` 后执行，并用 SHA-256 checksum 保护已应用文件。
- 五个 DDL 文件均使用 `IF NOT EXISTS`，均为事务迁移且不含 `CONCURRENTLY`：177 与 `SubscriptionPlan.currency`（`VARCHAR(3)`、默认空字符串）一致；178 由 channel pricing raw-SQL repository 的 `image_input_price` 读写承载；179 由 usage-log raw-SQL repository/DTO 的图片输入 token 和 cost 字段承载；180 由 append-only `auditLogRepository` 的 `audit_logs` 列集承载；181 与 Ent `Group.duplicate_operation_id`（`VARCHAR(64)`、nullable）及同名 partial unique index/predicate 一致。
- `go -C backend test ./migrations -count=1` 退出 0。`go -C backend test -v ./internal/repository -run '^(TestLatestMigrationBaseline|TestValidateMigrationExecutionMode|TestApplyMigrations_NilDB|TestApplyMigrations_DelegatesToApplyMigrationsFS|TestApplyMigrationsFS_(NonTransactionalMigration|TransactionalMigration))$' -count=1` 退出 0（6 个顶层测试、9 个命名子测试）。该 runner 单测覆盖排序基线、执行模式、nil DB、事务/非事务分支，但不连接 live PostgreSQL。

### 前端与残余手工项
- `pnpm --dir frontend run typecheck` 退出 0；`pnpm --dir frontend run build` 退出 0，完成 987 个模块。构建只保留既有 Browserslist 数据过期、动态导入和大于 500 kB chunk 警告。
- Docker-backed PostgreSQL migration integration 未运行，保持 manual；尤其 `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 及带 `integration` tag 的 repository migration/transaction 测试需要具备 live PostgreSQL 的环境后执行，不能由本节 unit 结果替代。
