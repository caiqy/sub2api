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
- CodeGraph `context`/`impact` 复核当前基线的共享入口：`SelectAccountWithScheduler`、`IsImageGenerationIntent`、`BillingRateMultiplier`、`Responses`、`ResponsesWebSocket`、`ForwardAlphaSearch`、`GetClientIP`、`BatchUpdateConcurrency`、`DuplicateAccount` 与 `applyUsageBilling`。`Responses`/`ResponsesWebSocket` 共同经过 `OpenAIGatewayHandler` 的并发、usage 和 failover 边界；scheduler 的 capability 和 image 入口分别由 `SelectAccountWithScheduler*` 与 `IsImageGenerationIntent` 收敛；`ForwardAlphaSearch` 仅在 2xx 返回 `WebSearchCalls=1`。
- session binding、操作审计、step-up、group duplicate 和 Grok cache 是目标 tag 新增符号，尚未存在于当前基线索引；已从固定 tag source 核对入口，不能据此把它们标成无影响。它们在合并后按 `manual` 逐段验收。
- 状态含义沿用首轮：`protected` 为当前基线存在直接行为断言；`manual` 为新 schema/生成物、新入口或跨层契约，要求合并后人工验收；`approved-removal` 仅用于已批准移除的本地首 Token 超时；`gap` 仅适用于本地独有、目标触及且缺少直接断言的既有行为。

### 扩展能力矩阵

| 风险面 | Tag | 入口及结构化调用链 | 现有测试 | 人工审查点 | 状态 |
| --- | --- | --- | --- | --- | --- |
| 异步图片任务和对象存储 | 157 | `ImageTaskHandler` -> `ImageTaskService` -> `ImageTaskStore`/S3 store。 | 目标 tag 的 `image_task_handler_test.go`、`image_task_store_test.go`；当前基线无同名本地独有契约。 | worker 状态迁移、对象 key、失败清理和 API-key image-task 鉴权。 | manual |
| 图片输入 token 和费用 | 157 | image request -> `gateway_key_billing`/usage repository -> usage log。 | 目标 tag 的 `gateway_key_billing_test.go`、`usage_log_repo_request_type_test.go`。 | image input tokens、group price 和 usage detail 同步写入。 | manual |
| 上游计费倍率和 scheduler | 157, 159 | `OpenAIGatewayService.SelectAccountWithScheduler*` -> account eligibility; `Account.BillingRateMultiplier` -> usage billing。 | `account_billing_rate_multiplier_test.go`、`openai_account_scheduler_test.go`、`openai_account_scheduler_upstream_cost_test.go`。 | cache snapshot 缺失倍率按 1.0、0 倍率合法，调度不得绕过 capability/价格限制。 | protected |
| 操作审计 | 157, 159 | `NewAuditLogMiddleware` -> `AuditLogService.Record`；session mismatch 也写入 audit。 | 目标 tag 的 `audit_log_test.go`、`audit_log_test.go`（middleware）。 | 脱敏、body 回填、认证后挂载和失败登录/refresh 例外。 | manual |
| 会话 IP/UA 绑定 | 157, 159 | `SessionBindingContext` -> `service.WithSessionBinding`; `enforceSessionBinding` -> revoke session family + audit。 | 目标 tag 的 `session_binding_test.go`、`jwt_auth_test.go`。 | 仅信任代理链 IP；旧无 binding claim 放行；不匹配撤销 family 并返回 401。 | manual |
| step-up 2FA | 157 | `/user/totp/step-up` -> TOTP service -> audit action。 | 目标 tag 的 `step_up_test.go`、`totp_verification_method_test.go`。 | step-up scope、过期/失败路径和审计动作。 | manual |
| OpenAI Responses 和 WebSocket | 157, 158, 159 | `Responses`/`ResponsesWebSocket` -> `OpenAIGatewayHandler` -> scheduler/usage/failover; WS ingress/HTTP bridge 共享 account transport。 | `openai_gateway_service_test.go`、`openai_ws_forwarder_ingress_session_test.go`、`openai_ws_forwarder_success_test.go`、`openai_ws_http_bridge_test.go`。 | Responses compact、WS V2 ingress 绑定、terminal usage 和 lease release。 | protected |
| Grok 端点和 prompt cache | 157, 158, 159 | Grok request -> `resolveGrokCacheIdentity` -> Responses body/header cache routing；endpoint 由 account type/base URL 解析。 | `grok_upstream_url_test.go`、`openai_gateway_grok_test.go`、目标 tag 的 `openai_images_test.go`。 | API key tenant isolation、Free tier tools、compact path 排除和 OAuth/API-key endpoint。 | protected |
| 分组复制 | 158 | admin group duplicate route -> `DuplicateGroup` -> group repository/outbox transaction。 | 目标 tag 的 `admin_group_duplicate_test.go`、`group_repo_duplicate_integration_test.go`、`GroupsView.duplicate.spec.ts`。 | inactive copy、深拷贝配置、account priority、idempotency key 和 name collision recovery。 | manual |
| 用户批量限额 | 158 | admin user handler `BatchUpdateConcurrency` -> admin user service -> user repository。 | 目标 tag 的 `user_handler_batch_limits_test.go`、`admin_service_batch_limits_test.go`、`BulkEditUserModal.spec.ts`。 | 仅允许的 mode、部分失败和管理 UI payload。 | manual |
| 客户端 IP 解析 | 159 | `GetClientIP`/`GetTrustedClientIP` -> middleware ACL/session/audit consumers。 | 目标 tag 的 `ip_test.go` 和 `session_binding_test.go`。 | trusted proxies 只影响安全路径；端口、XFF 和私网顺序不能回归。 | manual |
| alpha/search API-key 调度 | 157, 159 | route `POST /alpha/search` -> handler `AlphaSearch` -> `ForwardAlphaSearch` -> usage record。 | `openai_alpha_search_test.go`、`openai_alpha_search_billing_test.go`。 | 仅 2xx 产生一次 web-search 计费，OAuth/API-key endpoint 与 failover 保持一致。 | protected |
| 账号上游链接和账单探测 | 157, 158 | admin account handler -> upstream billing probe repository/CAS -> account UI。 | 目标 tag 的 `account_upstream_billing_probe_test.go`、`account_repo_upstream_billing_probe_*_test.go`、`admin.accounts.upstreamBillingProbe.spec.ts`。 | CAS due-time、持久化、链接/默认 base URL 和前端展示。 | manual |
| Stripe 惰性加载 | 159 | `StripePaymentView`/popup -> lazy Stripe import -> `StripePaymentInline`。 | 目标 tag 的 `StripePaymentView.spec.ts`、`stripeLazyLoading.spec.ts`。 | 首次加载、失败提示、popup/inline 路径及 Vite chunk 输出。 | manual |
| Wire、Ent 和 migrations 177-181 | 157, 158 | Ent schema -> generated client/migrate schema; `wire.go` -> `wire_gen.go`; migration runner 按文件名。 | `wire_gen_test.go`、migration/Ent 定向生成检查。 | 177 currency、178 image input price、179 usage tokens、180 audit、181 group duplicate 的顺序、幂等性和两次生成无 diff。 | manual |
| 本地首 Token 超时 | 157-159 | 已批准移除；不恢复 watchdog、config、logging、API 或 UI。 | 首轮报告 Task 15 记录的 stream interval timeout 仍是独立保护。 | 合并时确认只保留 stream data interval timeout，未重新引入 first-token 控制。 | approved-removal |

### 本轮保护结论
- `gap`：无。每项当前基线本地独有共享行为均有直接断言，新增 tag 自身入口均正确标为 `manual`，未将未合并测试伪称为当前保护。
- 因无 gap，未添加 characterization test；也未修改 backend 或 frontend。
- 现有保护命令：`go -C backend test ./internal/service ./internal/handler/... ./internal/repository ./internal/server/... -count=1` 退出 0；`pnpm --dir frontend exec vitest run` 退出 0（181 files, 1405 tests）。Vitest 保留既有 router-link、预期错误路径和 intlify 警告，未出现失败。

## v0.1.157
未开始合并。

## v0.1.158
未开始合并。

## v0.1.159
未开始合并。

## 最终验证
- 扩展前完整基线全部退出码为 0。
- 本次提交只包含本报告；首轮 v0.1.156 验证报告保持只读。

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
- 提交命令：`git add -f docs/superpowers/reports/2026-07-17-staged-merge-upstream-v0-1-159-validation.md`，随后执行 `git commit -m "docs: record v0.1.159 extension baseline"`。
- 提交范围：仅本报告。

## 自审
- 固定对象、提交计数、变更文件计数和最终边界与任务 brief 一致。
- 未合并 v0.1.157、v0.1.158 或 v0.1.159；未 push、release 或 deploy。
- 首轮 v0.1.156 验证报告未修改；`.comet/current-change.json` 未加入提交。
- 已复核首轮报告无 diff；暂存区和提交后将复核仅包含本报告，工作区仅保留忽略的 `.comet/current-change.json`。

## 残余风险与未执行事项
- 本任务仅固定 refs 和建立扩展前基线；v0.1.157、v0.1.158、v0.1.159 的合并和能力验证均未执行。
- 构建保留现有 Browserslist 数据过期、动态导入与 chunk 大小告警；本次命令均成功，未将其作为本任务范围内的修复项。
