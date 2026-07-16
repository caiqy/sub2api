# 上游 v0.1.156 分段合并验证记录

## 固定对象与工作区

- 隔离方式：当前仓库中的 feature 分支；本任务未创建、切换或合并分支。
- 隔离分支：`feature/20260716/staged-merge-upstream-v0-1-156`。
- 开始 `HEAD`：`d5f8192d32d9840d63477c24d4a567abb8cb4a90`。
- `HEAD` 父提交：`d1cc02502271f54b3b7f0593a18db4f2aaab63ea`。
- `HEAD` 主题：`test: isolate Go test temporary files`。
- `d1cc02502..HEAD`：仅该已确认的测试基础设施提交；差异文件为 `.gitignore`、`backend/Makefile`、`backend/scripts/test.ps1`，没有本次业务合并变更。

| Tag | Annotated tag object | Peel commit |
| --- | --- | --- |
| `v0.1.152` | `553ab6f911247963eb368fcf6ac1dcb65d5495b1` | `b73d8c3efe01a290eaaa9326b6e40ece02c67a0e` |
| `v0.1.153` | `53717a125583e3916b751c2a5340901c4bfa2bb3` | `a2bc1337474b68b62391116835e5698ebb5526bd` |
| `v0.1.155` | `ec4a37da4f023fbaa4d46d2ee46a6e7f22e313d4` | `41cec0db059ffb82d0efdcfcf07a24ab51fbfe97` |
| `v0.1.156` | `9cc1b469a24e6f79aeec9401ad1f9534f9b98aec` | `12f991dde8a58e183d4bd16a87ef6fd0df714757` |

- `git fetch upstream --tags` 后，`upstream/main` 从 `807850769` 更新至 `09c6c6d74`。
- 排除范围：`git log --oneline "v0.1.156^{}"..upstream/main` 仅用于记录 release 后上游历史；输出起点为 `09c6c6d74 Merge pull request #4387 from yardbirds0/feat/upstream-rate-scheduling`，尾部为 `75fb3c41c fix(apicompat): responses->chat ...`。未将该范围或 `upstream/main` merge 到当前分支。

## 初始工作树

`git status --short` 输出仅为：

```text
?? .comet/current-change.json
?? openspec/changes/staged-merge-upstream-v0-1-156/
```

`.comet/current-change.json` 保持未暂存、未提交。`openspec/changes/staged-merge-upstream-v0-1-156/` 是本任务允许提交的协调产物目录。

## 执行命令与结果

| 命令 | 关键输出 | 退出状态 |
| --- | --- | --- |
| `git status --short` | 仅初始工作树章节所列两个未跟踪路径 | 0 |
| `git rev-parse HEAD` | `d5f8192d32d9840d63477c24d4a567abb8cb4a90` | 0 |
| `git merge-base --is-ancestor d1cc02502271f54b3b7f0593a18db4f2aaab63ea HEAD` | 无输出，祖先关系成立 | 0 |
| `git log --oneline d1cc02502271f54b3b7f0593a18db4f2aaab63ea..HEAD` | `d5f8192d3 test: isolate Go test temporary files` | 0 |
| `git diff --name-status d1cc02502271f54b3b7f0593a18db4f2aaab63ea..HEAD` | 仅 `.gitignore`、`backend/Makefile`、`backend/scripts/test.ps1` | 0 |
| `git fetch upstream --tags` | `upstream/main`：`807850769..09c6c6d74` | 0 |
| `git rev-parse v0.1.152 "v0.1.152^{}"` | object/peel 与固定对象表一致 | 0 |
| `git rev-parse v0.1.153 "v0.1.153^{}"` | object/peel 与固定对象表一致 | 0 |
| `git rev-parse v0.1.155 "v0.1.155^{}"` | object/peel 与固定对象表一致 | 0 |
| `git rev-parse v0.1.156 "v0.1.156^{}"` | object/peel 与固定对象表一致 | 0 |
| `git log --oneline "v0.1.156^{}"..upstream/main` | 仅记录 release 后排除范围，未 merge | 0 |
| `git branch --show-current` | `feature/20260716/staged-merge-upstream-v0-1-156` | 0 |
| `git show -s --format='%H%n%P%n%s' HEAD` | `HEAD`、父提交和主题与本报告一致 | 0 |

## 提交与自审

- 首次协调提交 SHA：`3877dc247ea58ef2194051399db3e67974d68473`，message 为 `docs: add staged upstream merge plan`。本报告更正后另行创建普通文档提交，不在本次提交中记录其自身 SHA。
- 变更文件：3 个 `docs/superpowers/{specs,plans,reports}/2026-07-16-staged-merge-upstream-v0-1-156*` 文档，以及 `openspec/changes/staged-merge-upstream-v0-1-156/` 下 19 个协调文件，共 22 个新增文件。
- 暂存自审：`git diff --cached --check` 退出 0；`git diff --cached --name-only -- .comet/current-change.json .superpowers` 无输出。
- 提交自审：首次提交的 `git show --name-status --format=fuller` 仅列出上述 22 个允许路径；根目录 `.comet/current-change.json` 保持未跟踪，未提交 `.superpowers/` 或业务代码。
- 事实自审：分支、开始 `HEAD`、父提交、四个 tag object/peel SHA 与 brief 完全一致；没有执行 `git merge`、分支切换、业务代码修改、测试、push、release、deploy 或 main 合并。未勾选计划或 OpenSpec task。

## 顾虑

- `upstream/main` 在 fetch 时前进至 `09c6c6d74`，其相对 `v0.1.156^{}` 的完整范围只作记录；后续四个 tag 分段 merge 必须继续以本报告固定的 tag peel commit 为目标。
- 本任务依用户裁决只核验 Git/工作树证据，不运行或伪造 RED/GREEN 测试。
- 协调产物为 22 文件、2300 行，超过 200 行风险阈值；均为本次既有设计、计划、OpenSpec/Comet 协调内容。
- 暂存时 Git 提示这些文档的工作副本下次被 Git 触及时可能发生 LF/CRLF 工作树转换；本次 `git diff --cached --check` 通过。

## 阶段 0 基线（OpenSpec 1.2）

**结论：PASS。** 本阶段未执行任何 tag merge、业务代码修改、测试补齐、push、release、deploy 或 main 合并；不勾选 plan/OpenSpec task。

### 执行环境与范围

- 工作目录：`D:\Caiqy\Projects\Github\sub2api`（Windows）。
- 质量门禁定义：根目录 `Makefile` 的 `test` 依次运行后端默认测试（含 `golangci-lint`）、后端 `unit` tag 测试、前端 ESLint、`vue-tsc --noEmit` 与 Vitest。
- 前端构建定义：`frontend/package.json` 的 `build` 为 `vue-tsc -b && vite build`，产物写入 `backend/internal/web/dist`。
- 生成定义：`backend/Makefile` 的 `generate` 依次执行 `go generate ./ent` 与 `go generate ./cmd/server`；检查范围严格限制为 `backend/ent` 和 `backend/cmd/server/wire_gen.go`。

### 命令与结果

| 阶段 | 命令 | 退出码 | 摘要 |
| --- | --- | --- | --- |
| 初始工作树 | `git status --short` | 0 | `docs/superpowers/plans/2026-07-16-staged-merge-upstream-v0-1-156.md`、`openspec/changes/staged-merge-upstream-v0-1-156/.comet/subagent-progress.md` 为修改；`.comet/current-change.json` 未跟踪。均为主会话协调状态，未触碰。 |
| 初始静态检查 | `git diff --check` | 0 | 无空白错误；对上述两份既有协调文件提示下次 Git 写入会将 LF 转为 CRLF。 |
| 质量门禁 | `make test` | 0 | 后端默认测试、`unit` tag 测试与 `golangci-lint` 通过；前端 ESLint、类型检查通过；Vitest 为 167 个文件、1246 个测试通过。 |
| 前端嵌入构建 | `pnpm --dir frontend run build` | 0 | `vue-tsc -b` 与 Vite 生产构建通过，966 个模块完成转换，构建耗时 40.52 秒。 |
| 第 1 轮生成 | `make -C backend generate` | 0 | Ent 与 Wire 生成完成；Wire 写入 `backend/cmd/server/wire_gen.go`。 |
| 第 1 轮生成稳定性 | `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` | 0 | 无输出；生成目标相对当前基线无 diff。 |
| 第 2 轮生成 | `make -C backend generate` | 0 | Ent 与 Wire 再次生成完成。 |
| 第 2 轮生成稳定性 | `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` | 0 | 无输出；两轮生成稳定。 |
| 最终工作树 | `git status --short` | 0 | 与初始工作树相同，未出现生成目标或业务代码变更。 |
| 最终静态检查 | `git diff --check` | 0 | 无空白错误；仅重复既有协调文件的 LF/CRLF 警告。 |

### 警告与风险信号

- `make test` 的 Vitest 输出有预期错误路径日志、`router-link` 解析警告与 `intlify` message compiler 警告；所有断言通过，命令退出 0。
- 测试和构建均提示 `caniuse-lite` 浏览器数据已 7 个月未更新；这是依赖数据维护信号，未阻塞本阶段。
- Vite 报告多个动态/静态混用导入，且 `AccountsView` 压缩后为 635.06 kB，超过 500 kB chunk 警戒线；构建成功，但后续性能工作应单独处理，不能归因于本阶段。
- 生成检查仅覆盖 brief 规定的 Ent 与 Wire 目标；构建产物由 Git 忽略，最终 `git status --short` 未显示其变更。

### 自审与提交

- 自审：未改动业务代码、测试、生成源码或主会话的 plan/`.comet/subagent-progress.md`；两轮受限 diff 均为空，所有基线命令退出 0，因此没有触发阻塞或根因调查流程。
- 仅暂存本报告：`git add -f docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md`。
- 提交命令：`git commit -m "docs: record stage zero baseline"`；提交内容只允许为本报告。`.comet/current-change.json`、`.superpowers/` 以及主会话协调文件必须保持未暂存、未提交。

## Task 3 能力至验证证据映射（OpenSpec 1.3，正式交付）

### 调查范围与判定

- 本地实现基线：`d5f8192d32d9840d63477c24d4a567abb8cb4a90`；本地独有范围由 `v0.1.151^{}` 至该基线导出。范围包含先前合并后的本地修复、OpenSpec 归档、测试基础设施和协调提交，不能仅凭提交标题将行为视为受保护。
- 目标 release：`v0.1.152`、`v0.1.153`、`v0.1.155`、`v0.1.156`；阶段 0 的 `make test`、前端构建和受限生成检查均已通过，本任务未重跑测试、未制造 RED/GREEN。
- `protected`：存在直接行为断言，且阶段 0 已通过；`manual`：生成物、migration、版本/依赖或跨层契约；`approved-removal`：仅首 Token 超时，在前三段仍保持 protected，待 `v0.1.156` 合并后按已批准移除项处理；`gap` 仅在本地独有、目标 release 触及且无关键行为断言同时成立时使用。
- 主规格已读取：`openspec/specs/{upstream-release-sync,platform-sticky-boundaries,content-moderation-config,request-body-retention-control,large-input-memory-control,user-ui-visibility-overrides,user-public-group-blocklist,openai-first-token-timeout,local-test-gates}/spec.md`；业务索引、v0.1.151 验证报告和上游合并工作流亦已读取。

### 命令与结构调查

| 类别 | 命令/查询 | 结果 |
| --- | --- | --- |
| 本地独有提交 | `git log --format='%H %s' v0.1.151^{}..d5f8192d32d9840d63477c24d4a567abb8cb4a90` | 范围已导出；高风险本地行为提交见本节后的矩阵证据。`d5f8192d3` 本身仅隔离 Go 测试临时文件。 |
| 四段 changed-files | `git diff --name-only v0.1.151..v0.1.152`、`v0.1.152..v0.1.153`、`v0.1.153..v0.1.155`、`v0.1.155..v0.1.156` | 原始清单在下节；四段均直接触及网关、协议、配置或生成契约。 |
| 目标总览 | `git diff --name-only v0.1.151..v0.1.156`、`git diff --stat v0.1.151..v0.1.156` | 503 文件、58,452 新增、2,858 删除；不能以全量通过代替能力断言。 |
| 文字定位 | `git grep -n -E 'previous_response|sticky|WaitPlan|recheck|replay|cleanup|runtime|blocked.*group|purchase|custom.*menu' -- backend frontend` | 定位到各能力的入口、设置键、前端页面与现有测试名。 |
| CodeGraph context | `context`：scheduler、Sticky、WaitPlan、recheck、协议转换、usage、privacy、image、runtime、body、资源控制 | `GatewayService` 统一处理平台 Sticky；`OpenAIGatewayService` 负责 OpenAI 调度/转发；Ent `UsageLog` 为记录出口。 |
| CodeGraph explore（唯一一次） | `explore GatewayService OpenAIGatewayService scheduler sticky WaitPlan GetGeminiCachedSessionAccountID ForwardAsMessages ForwardAsChatCompletions ForwardAsResponses contentModeration OpenAIImagesCapability settingService requestBodyHandle UserResourceOverride` | 确认 `GatewayService.stickyEnabledForPlatform -> get/bindStickySessionForPlatform`、`ForwardAs*` 协议链和 request body handle 所有权边界；未用文件名推断调用链。 |

### 四段 changed-files 高风险摘录（辅助索引）

以下摘录用于快速定位高风险文件；五份命令的无删节 stdout 见本章末尾的正式证据附录，摘录不替代该附录。

#### v0.1.151..v0.1.152

```text
README.md
backend/cmd/server/VERSION
backend/ent/group.go
backend/ent/group/group.go
backend/ent/group/where.go
backend/ent/group_create.go
backend/ent/group_update.go
backend/ent/migrate/schema.go
backend/ent/mutation.go
backend/ent/runtime/runtime.go
backend/ent/schema/group.go
backend/go.mod
backend/go.sum
backend/internal/handler/admin/group_handler.go
backend/internal/handler/dto/mappers.go
backend/internal/handler/dto/types.go
backend/internal/handler/endpoint.go
backend/internal/handler/openai_chat_completions.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/pkg/apicompat/anthropic_to_responses_response.go
backend/internal/pkg/apicompat/chatcompletions_responses_bridge.go
backend/internal/pkg/apicompat/responses_stream_event_wire.go
backend/internal/pkg/apicompat/responses_to_anthropic.go
backend/internal/pkg/apicompat/types.go
backend/internal/repository/account_repo.go
backend/internal/repository/api_key_repo.go
backend/internal/repository/group_repo.go
backend/internal/repository/http_upstream.go
backend/internal/server/routes/gateway.go
backend/internal/service/account.go
backend/internal/service/admin_group.go
backend/internal/service/api_key_auth_cache.go
backend/internal/service/api_key_auth_cache_impl.go
backend/internal/service/billing_service.go
backend/internal/service/grok_media.go
backend/internal/service/group.go
backend/internal/service/image_generation_intent.go
backend/internal/service/media_price_config.go
backend/internal/service/openai_gateway_cc_pipeline.go
backend/internal/service/openai_gateway_chat_completions.go
backend/internal/service/openai_gateway_chat_completions_raw.go
backend/internal/service/openai_gateway_forward.go
backend/internal/service/openai_gateway_messages.go
backend/internal/service/openai_gateway_messages_chat_fallback.go
backend/internal/service/openai_gateway_responses_chat_fallback.go
backend/internal/service/openai_gateway_scheduling.go
backend/internal/service/openai_gateway_service.go
backend/internal/service/openai_gateway_usage.go
backend/internal/service/openai_ws_forwarder_ingress.go
backend/internal/service/openai_ws_http_bridge.go
backend/internal/service/openai_ws_pool.go
backend/migrations/174_group_web_search_price_per_call.sql
frontend/src/components/account/CreateAccountModal.vue
frontend/src/components/account/EditAccountModal.vue
frontend/src/components/keys/UseKeyModal.vue
frontend/src/i18n/locales/en/admin/settings.ts
frontend/src/i18n/locales/zh/admin/settings.ts
frontend/src/views/admin/GroupsView.vue
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/settings/OpenAIFastPolicyUserSelector.vue
```

#### v0.1.152..v0.1.153

```text
.github/workflows/backend-ci.yml
.gitignore
README.md
README_CN.md
README_JA.md
backend/cmd/server/VERSION
backend/cmd/server/wire_gen.go
backend/internal/config/config.go
backend/internal/config/config_test.go
backend/internal/handler/failover_loop.go
backend/internal/handler/failover_loop_test.go
backend/internal/handler/gateway_handler.go
backend/internal/handler/gateway_handler_chat_completions.go
backend/internal/handler/gateway_handler_responses.go
backend/internal/handler/gateway_handler_usage_test.go
backend/internal/handler/gateway_helper.go
backend/internal/handler/gemini_v1beta_handler.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/pkg/apicompat/anthropic_responses_test.go
backend/internal/pkg/apicompat/anthropic_to_responses_response.go
backend/internal/pkg/apicompat/chatcompletions_responses_bridge.go
backend/internal/pkg/apicompat/responses_to_anthropic.go
backend/internal/pkg/apicompat/responses_to_chatcompletions.go
backend/internal/pkg/apicompat/streaming_stop_reason_test.go
backend/internal/repository/api_key_repo.go
backend/internal/repository/concurrency_cache.go
backend/internal/repository/migrations_runner.go
backend/internal/repository/scheduler_cache.go
backend/internal/server/routes/gateway.go
backend/internal/service/account.go
backend/internal/service/concurrency_service.go
backend/internal/service/grok_media.go
backend/internal/service/openai_gateway_responses_chat_fallback.go
backend/internal/service/openai_ws_client.go
backend/internal/service/openai_ws_forwarder_ingress.go
backend/internal/service/openai_ws_pool.go
backend/internal/service/upstream_models.go
backend/migrations/174_add_usage_logs_api_key_latest_ip_index_notx.sql
deploy/.env.example
deploy/config.example.yaml
frontend/src/api/payment.ts
frontend/src/components/account/EditAccountModal.vue
frontend/src/components/common/DataTable.vue
frontend/src/composables/useSwipeSelect.ts
frontend/src/i18n/locales/en/admin/accounts.ts
frontend/src/i18n/locales/zh/admin/accounts.ts
frontend/src/views/KeyUsageView.vue
frontend/src/views/admin/AccountsView.vue
frontend/src/views/user/DashboardView.vue
```

#### v0.1.153..v0.1.155

```text
backend/cmd/server/VERSION
backend/cmd/server/wire_gen.go
backend/ent/channelmonitor/channelmonitor.go
backend/ent/channelmonitorrequesttemplate/channelmonitorrequesttemplate.go
backend/ent/migrate/schema.go
backend/ent/mutation.go
backend/ent/runtime/runtime.go
backend/ent/schema/channel_monitor.go
backend/ent/schema/channel_monitor_request_template.go
backend/ent/schema/usage_log.go
backend/ent/usagelog.go
backend/ent/usagelog_create.go
backend/ent/usagelog_update.go
backend/internal/config/config.go
backend/internal/handler/admin/account_handler.go
backend/internal/handler/admin/channel_monitor_handler.go
backend/internal/handler/admin/ops_system_log_handler.go
backend/internal/handler/dto/mappers.go
backend/internal/handler/dto/types.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/openai_images.go
backend/internal/handler/wire.go
backend/internal/pkg/apicompat/responses_namespace.go
backend/internal/repository/account_repo.go
backend/internal/repository/migrations_runner.go
backend/internal/repository/scheduler_outbox_repo.go
backend/internal/repository/usage_log_repo_insert.go
backend/internal/repository/usage_log_repo_query.go
backend/internal/server/router.go
backend/internal/server/routes/admin.go
backend/internal/service/account.go
backend/internal/service/admin_account.go
backend/internal/service/channel_monitor_checker.go
backend/internal/service/content_moderation.go
backend/internal/service/gateway_usage_billing.go
backend/internal/service/image_generation_intent.go
backend/internal/service/openai_gateway_forward.go
backend/internal/service/openai_gateway_grok.go
backend/internal/service/openai_gateway_messages.go
backend/internal/service/openai_gateway_passthrough.go
backend/internal/service/openai_gateway_request_body.go
backend/internal/service/openai_gateway_response_handling.go
backend/internal/service/openai_gateway_service.go
backend/internal/service/openai_gateway_usage.go
backend/internal/service/openai_images_responses.go
backend/internal/service/openai_ws_forwarder_ingress.go
backend/internal/service/openai_ws_http_bridge.go
backend/internal/service/openai_ws_pool.go
backend/internal/service/ops_models.go
backend/internal/service/scheduler_snapshot_service.go
backend/internal/service/wire.go
backend/migrations/174_add_usage_log_long_context_billing.sql
backend/migrations/175_add_ops_system_logs_host.sql
backend/migrations/175_default_openai_long_context_billing.sql
backend/migrations/175a_add_ops_system_logs_host_index_notx.sql
backend/migrations/176_channel_monitor_grok_provider.sql
deploy/.env.example
deploy/config.example.yaml
frontend/src/api/admin/accounts.ts
frontend/src/api/admin/ops.ts
frontend/src/components/account/CreateAccountModal.vue
frontend/src/components/account/EditAccountModal.vue
frontend/src/components/admin/usage/UsageTable.vue
frontend/src/i18n/locales/en/admin/ops.ts
frontend/src/i18n/locales/zh/admin/ops.ts
frontend/src/views/admin/AccountsView.vue
frontend/src/views/admin/ops/components/OpsSystemLogTable.vue
```

#### v0.1.155..v0.1.156

```text
README.md
README_CN.md
README_JA.md
backend/cmd/server/VERSION
backend/cmd/server/wire_gen.go
backend/ent/group.go
backend/ent/group/group.go
backend/ent/group/where.go
backend/ent/group_create.go
backend/ent/group_update.go
backend/ent/migrate/schema.go
backend/ent/mutation.go
backend/ent/runtime/runtime.go
backend/ent/schema/group.go
backend/go.mod
backend/go.sum
backend/internal/config/config.go
backend/internal/handler/admin/account_handler.go
backend/internal/handler/admin/group_handler.go
backend/internal/handler/failover_loop.go
backend/internal/handler/gateway_handler.go
backend/internal/handler/gateway_handler_chat_completions.go
backend/internal/handler/gateway_handler_responses.go
backend/internal/handler/gateway_helper.go
backend/internal/handler/gemini_v1beta_handler.go
backend/internal/handler/openai_chat_completions.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/openai_images.go
backend/internal/pkg/apicompat/anthropic_to_responses_response.go
backend/internal/pkg/apicompat/chatcompletions_anthropic_bridge.go
backend/internal/pkg/apicompat/chatcompletions_responses_bridge.go
backend/internal/pkg/apicompat/responses_to_anthropic.go
backend/internal/pkg/apicompat/streaming_stop_reason_test.go
backend/internal/repository/account_repo.go
backend/internal/repository/scheduler_cache.go
backend/internal/server/routes/gateway.go
backend/internal/service/account.go
backend/internal/service/admin_account.go
backend/internal/service/content_moderation.go
backend/internal/service/gateway_service.go
backend/internal/service/grok_media.go
backend/internal/service/image_generation_intent.go
backend/internal/service/openai_account_runtime_block_fastpath.go
backend/internal/service/openai_first_output_timeout.go
backend/internal/service/openai_gateway_cc_pipeline.go
backend/internal/service/openai_gateway_chat_completions.go
backend/internal/service/openai_gateway_forward.go
backend/internal/service/openai_gateway_messages.go
backend/internal/service/openai_gateway_passthrough.go
backend/internal/service/openai_gateway_response_handling.go
backend/internal/service/openai_gateway_service.go
backend/internal/service/openai_images.go
backend/internal/service/openai_ws_forwarder_ingress.go
backend/internal/service/openai_ws_http_bridge.go
backend/internal/service/openai_ws_pool.go
backend/internal/service/scheduler_cache.go
backend/internal/service/scheduler_snapshot_service.go
backend/internal/service/wire.go
deploy/config.example.yaml
frontend/src/api/admin/accounts.ts
frontend/src/components/account/CreateAccountModal.vue
frontend/src/components/admin/account/AccountActionMenu.vue
frontend/src/components/keys/UseKeyModal.vue
frontend/src/i18n/locales/en/admin/accounts.ts
frontend/src/i18n/locales/zh/admin/accounts.ts
frontend/src/views/admin/AccountsView.vue
frontend/src/views/admin/GroupsView.vue
frontend/src/views/user/KeysView.vue
```

## Task 5 / v0.1.152 merge 决策（OpenSpec 2.1）

### 状态

- 结果：`DONE`。
- merge commit：`4ffe039a4399f8cbac1f83df32b709afda777ffe` `merge: upstream v0.1.152`。
- 第一父：`f10199795fd5fe4ef54c99553149177612179756`（任务准备提交）。
- 第二父：`b73d8c3efe01a290eaaa9326b6e40ece02c67a0e`（唯一允许的 `v0.1.152^{}`）。
- 原 scratch 台账曾被错误提交；本章节承载其完整正式记录，scratch 保留为 ignored 工作区报告。
- 用户边界裁决：保持 `4ffe039a` 不变；review 后的 Grok 默认 URL RED/GREEN、代码/测试提交 `b19c03d01` 和证据提交 `2026265cb` 均归入 Task 6 提前执行项，不作为 Task 5 实现。Task 5 仅验收 merge 拓扑、冲突融合台账及向 Task 6 的交接。

### 前置核验与命令记录

| 命令 | 结果 |
| --- | --- |
| `git status --short` | merge 前唯一未跟踪项为允许的 `.comet/current-change.json`。 |
| `git status --branch --short` | 分支为 `feature/20260716/staged-merge-upstream-v0-1-156`。 |
| `git log -1 --format='%H %s'` | `f10199795fd5fe4ef54c99553149177612179756 chore: prepare staged merge task 5`。 |
| `git rev-parse 'v0.1.152^{}'` | `b73d8c3efe01a290eaaa9326b6e40ece02c67a0e`，与任务固定 SHA 一致。 |
| `git cat-file -t v0.1.152` | `tag`，确认目标为 annotated tag。 |
| `git tag -v v0.1.152` | 本机缺失 tag 签名，校验失败；不影响 tag 对象类型与 peel SHA 的确定性核验。 |
| Task 4 阶段 0 报告和 `openspec/changes/staged-merge-upstream-v0-1-156/tasks.md` | 阶段 0 的 1.1 至 1.4 已勾选，`make test`、前端 build、Ent/Wire 生成检查和 `git diff --check` 均有通过记录。 |
| `git merge-base --is-ancestor v0.1.152 HEAD` | merge 前返回非零，确认 tag 尚未是当前分支祖先。 |
| `git diff --name-status HEAD...v0.1.152` | 确认本次仅需处理该 tag 与当前分支的差异。 |
| `git merge --no-ff v0.1.152 -m "merge: upstream v0.1.152"` | 进入冲突状态；未 abort、squash 或 cherry-pick。 |
| `git diff --name-only --diff-filter=U`、`git diff --cc -- <path>`、`git show :1:<path>`、`git show :2:<path>`、`git show :3:<path>` | 逐路径核对共同祖先、第一父本地实现和 tag 上游实现。 |
| CodeGraph `context`、`impact(openAIFirstTokenWatchdog)`、`callers(resolveOpenAIUpstreamEndpoint)` | 确认首 Token watchdog 覆盖 Responses-to-Chat fallback，真实上游端点解析由 Responses、WebSocket、cyber-policy 记录调用。 |
| `go generate ./ent` | 基于已自动合并的 `backend/ent/schema/group.go` 重生成 Ent，修复 runtime descriptor 索引。 |
| `gofmt -w <10 个冲突 Go 文件>` | 格式化必要融合路径。 |
| `git ls-files -u`、`git diff --name-only --diff-filter=U`、`rg -n '^(<<<<<<<|>>>>>>>)' backend frontend` | 输出为空。 |
| `git diff --cached --check` | 输出为空。 |
| `git commit --no-edit` | 创建上述 merge commit。 |
| `git show -s --format='%H%n%P%n%s' HEAD`、`git rev-parse HEAD^2` | merge SHA、两个父节点、subject 和第二父均符合要求。 |
| `git merge-base --is-ancestor upstream/main HEAD` | 返回非零，`upstream/main` 不是结果祖先。 |
| `git log --oneline v0.1.152..upstream/main --not HEAD` | 列出所有 tag 后 upstream/main 提交，均未进入结果。 |
| `git show --check --format=fuller --stat HEAD` | 输出无 whitespace 错误；commit 仅为 merge 树与冲突融合。 |

### 冲突台账

共 `15` 个文本冲突；无其他未合并路径。

| 路径 | 分类 | 第一父本地语义 | tag 上游语义 | 融合结论 | Task 6 验证 |
| --- | --- | --- | --- | --- | --- |
| `backend/cmd/server/VERSION` | 版本依赖 | 本地四段版本 `0.1.151.2`。 | tag 内为 `0.1.151`。 | 保留本地版本，避免在 staged merge 擅自改变本地发布编号。 | 复核最终版本与 tag/发布流程。 |
| `backend/ent/mutation.go` | 生成代码/接口演进 | `user_concurrency_*` mutation 字段。 | `web_search_price_per_call` mutation 字段。 | 按合并 schema 重生成，两个字段集合均存在，`Fields` 容量为 50。 | Ent 生成可复现检查。 |
| `backend/ent/runtime/runtime.go` | 生成代码/接口演进 | 用户并发 descriptor 偏移。 | 网页搜索价格 descriptor 偏移。 | 重生成后所有后续 descriptor 索引后移一位，未发生同位读取。 | Ent 生成可复现检查。 |
| `backend/internal/handler/openai_chat_completions.go` | 网关接口演进/本地定制 | failover 耗尽 usage、usage detail snapshot、raw Chat endpoint 推导。 | 模型不存在错误分类和 forwarding result 的真实 endpoint。 | 保留 failover 分支与 detail snapshot；无 failover 时使用上游分类；采用真实 endpoint resolver。 | Chat Completions 无账号、failover、usage detail 和 raw Chat 路径测试。 |
| `backend/internal/handler/openai_gateway_handler.go` | 网关接口演进/本地定制 | Responses/Messages failover、detail snapshot、调度保护。 | 模型不存在错误分类和真实 endpoint。 | failover 分支保留；无 failover 时使用上游错误分类；所有 usage 记录使用真实 endpoint。 | Responses、Messages、WebSocket usage 与 no-account 定向测试。 |
| `backend/internal/server/routes/gateway.go` | 路由/本地定制 | Responses 路由捕获 usage detail，Grok 拒绝 Responses WebSocket。 | 新增 `/alpha/search` 路由。 | 保留 middleware 和 Grok 拒绝，新增 alpha/search 并同样接入 usage detail。 | 路由契约及 alpha/search 计费测试。 |
| `backend/internal/service/api_key_auth_cache_impl.go` | 缓存版本依赖 | snapshot v17 已涵盖 blocked group、批量图像/视频价格。 | v15 新增网页搜索单次价格。 | 保持 v17，并明确其快照覆盖网页搜索价格。 | API Key cache 序列化、失效与网页搜索价格测试。 |
| `backend/internal/service/openai_gateway_grok.go` | 调度/配额融合 | pool-mode 可重试状态不临时下线账号。 | 将 Grok quota snapshot 写入运行时与持久限流状态。 | 先保留 pool-mode 早退，其他状态继续写入 quota snapshot。 | Grok pool-mode、429/quota snapshot 测试。 |
| `backend/internal/service/openai_gateway_responses_chat_fallback.go` | 首 Token 保护/协议桥 | SSE 首 Token watchdog、超时记录和客户端写入保护。 | custom/tool_search/namespace 工具往返还原。 | 两者共存；watchdog 仍覆盖 fallback 串流，工具元数据完整传入转换器。 | 首 Token fallback 超时、custom/tool_search/namespace 流式与非流式测试。 |
| `backend/internal/service/openai_gateway_service.go` | usage/接口演进 | 异步 usage 的深拷贝快照。 | 实际上游 endpoint context 和网页搜索计数字段。 | 保留快照函数，同时加入 endpoint context API 和 `WebSearchCalls`。 | usage 异步快照与实际 endpoint 记录测试。 |
| `backend/internal/service/openai_oauth_passthrough_test.go` | 测试融合 | 断言账户注入 header、转发 header 和 body 字段。 | 断言 `x-codex-beta-features=remote_compaction_v2`。 | 同一用例保留全部三类断言。 | 运行该文件的 passthrough 回归测试。 |
| `frontend/src/components/account/CreateAccountModal.vue` | 前端本地定制 | `getDefaultBaseUrl` 统一提供平台默认 URL。 | Grok API Key 默认 xAI URL。 | 原“已覆盖 Grok”结论错误：helper 漏掉 `grok`，会回退 Anthropic URL；Task 6 提前执行项 `b19c03d01` 已以最小分支修复。 | Task 6 仍须覆盖 CreateAccountModal Grok/API Key 表单。 |
| `frontend/src/components/account/EditAccountModal.vue` | 前端本地定制 | `getDefaultBaseUrl` 统一提供平台默认 URL。 | Grok API Key 默认 xAI URL。 | 原“完整平台覆盖”结论错误：同一 helper 漏掉 `grok`，会回退 Anthropic URL；Task 6 提前执行项 `b19c03d01` 已以最小分支修复。 | Task 6 仍须覆盖 EditAccountModal Grok/API Key 表单。 |
| `frontend/src/i18n/locales/en/admin/settings.ts` | 前端接口演进 | 手工用户 ID 输入文案。 | 邮箱模糊搜索 selector 文案。 | 采用上游 selector 键；底层仍选择同一用户 ID，配合已合入 selector 组件。 | Fast/Flex 用户 selector locale 测试。 |
| `frontend/src/i18n/locales/zh/admin/settings.ts` | 前端接口演进 | 手工用户 ID 输入中文文案。 | 邮箱模糊搜索 selector 中文文案。 | 采用上游 selector 键，与英文 locale 和组件契约一致。 | Fast/Flex 用户 selector locale 测试。 |

### Task 6 提前执行：Merge 后 reviewer Grok URL 修复

- 用户裁决：保持 Task 5 merge commit `4ffe039a` 不变；本段 RED/GREEN、代码/测试和证据均为 Task 6 提前执行，不计为 Task 5 实现。
- reviewer 发现：`getDefaultBaseUrl('grok')` 实际回退 `https://api.anthropic.com`，因此 Create/Edit 依赖 helper 时未保留 Grok API Key 的默认 URL；本台账原“已融合”结论已更正。
- 精确默认值：`v0.1.152` 的 `CreateAccountModal.grok.spec.ts` 与 `EditAccountModal.spec.ts` 均断言官方 xAI API Key URL 为 `https://api.x.ai/v1`。
- Task 6 提前执行 RED：`pnpm --dir frontend test:run src/components/account/__tests__/passthroughFieldSupport.spec.ts`，1 个测试失败，实际 `https://api.anthropic.com`、期望 `https://api.x.ai/v1`。
- Task 6 提前执行 GREEN：同一命令，1 个测试通过。
- Task 6 提前执行代码/测试提交：`b19c03d01` `fix: preserve Grok API key default URL after v0.1.152`；仅修改 `passthroughFieldSupport.ts` 和其相邻的最小单测。
- Task 6 提前执行证据提交：`2026265cb` `docs: record v0.1.152 Grok URL fix`。
- 后续：Task 6 仍须运行并覆盖 CreateAccountModal 与 EditAccountModal 的 Grok API Key 表单行为；本次未运行完整前端 suite。

### 自审与风险

- merge commit 不含验证报告、OpenSpec task、Comet progress、`.comet/current-change.json` 或其他 `.superpowers/` 文件；本台账由 merge 后独立文档提交承载。
- Task 5 merge implementer 未运行行为回归、完整后端测试或前端 build，未新增测试、未做无冲突能力审查或 merge 后语义修复。review 发现后的 Grok RED/GREEN、最小语义修复和测试由 Task 6 提前执行项完成；Task 6 仍须按 TDD 和阶段测试矩阵核对 Create/Edit 表单。
- 本地首 Token 超时仍受保护：其 watchdog 及 fallback 覆盖路径保留；未提前进行仅获批准于 v0.1.156 的移除。
- 风险信号：本 tag 触及网关、协议转换、Ent schema/生成物、Grok 配额、API Key cache、计费与前端设置；除 Task 6 提前执行的 helper 修复外，生成物、无文本冲突行为和 Create/Edit Grok API Key 表单仍需 Task 6 复核。
- 顾虑：本机无法验证 annotated tag 签名；已使用对象类型与固定 peel SHA 作为合并身份依据。`VERSION` 选择保留本地四段版本，需在最终发布阶段再次复核。

## Task 8 / v0.1.153 merge 决策（OpenSpec 3.1）

### 状态与拓扑

- 结果：`PASS`，仅完成 OpenSpec 3.1 的 `v0.1.153` merge；未合入 `v0.1.155` 或 `upstream/main`。
- annotated tag object：`53717a125583e3916b751c2a5340901c4bfa2bb3`；唯一允许的 peel：`a2bc1337474b68b62391116835e5698ebb5526bd`。
- merge commit：`9219483d7c34606e7c2cb530c00a46b764096414` `merge: upstream v0.1.153`。
- 第一父：`4e4ed09887bfbe9e8072ea60b137b85f704da185`；第二父：`a2bc1337474b68b62391116835e5698ebb5526bd`。
- `git merge-base --is-ancestor upstream/main HEAD` 返回 `not-ancestor`。原 scratch 台账曾被错误提交；本专章为完整正式记录，scratch 保留为 ignored 工作区报告。

### 命令与静态核验

| 命令 | 结果 |
| --- | --- |
| `git status --porcelain=v1` | merge 前后仅有既有未跟踪 `.comet/current-change.json`。 |
| `git tag -v v0.1.153`; `git rev-parse v0.1.153`; `git rev-parse "v0.1.153^{}"` | tag 为 annotated；object 和 peel 如上。本机没有 tag 签名材料。 |
| `git merge --no-ff v0.1.153 -m "merge: upstream v0.1.153"` | 产生 9 个内容冲突，逐项融合。 |
| `make -C backend generate` | Ent/Wire 完成；Wire 两次写入 `backend/cmd/server/wire_gen.go`。 |
| `git diff --name-only --diff-filter=U`; `git ls-files -u`; `git grep --cached -n -E "^(<<<<<<< |>>>>>>> |=======$)"` | 均无输出。 |
| `git diff --cached --check`; `git diff --check "HEAD^1" HEAD` | 均无输出；删除上游测试文件末尾空白行以满足检查。 |
| `git diff --cached --name-only -- .superpowers openspec .comet` | merge commit 前无受限路径被暂存。 |

### 冲突台账

共 `9` 个文本冲突；无其他未合并路径。

| 路径 | 分类 | 第一父本地语义 | tag 上游语义 | 融合结论 |
| --- | --- | --- | --- | --- |
| `.gitignore` | 忽略规则 | 忽略本地开发工具。 | 放行 `deploy/tests` 并保留 `CLAUDE.md`。 | 合并两侧规则。 |
| `backend/cmd/server/VERSION` | 版本裁决 | `0.1.151.2`。 | `0.1.152`。 | 保留已验证的本地四段版本裁决。 |
| `backend/cmd/server/wire_gen.go` | Wire 生成物 | `ProvidePaymentHandler` 的 channel/user 注入。 | 两参构造器。 | 按当前 `wire.go` 的四参 provider 生成。 |
| `backend/internal/handler/gateway_handler_chat_completions.go` | failover/usage | 失败账号与耗时用于失败用量。 | 账号级 pool retry limit。 | 同时记录失败信息，并传入 `account.GetPoolModeRetryCount()`。 |
| `backend/internal/handler/gateway_handler_responses.go` | failover/usage | 失败账号与耗时用于失败用量。 | 账号级 pool retry limit。 | 同时记录失败信息，并传入 `account.GetPoolModeRetryCount()`。 |
| `backend/internal/handler/gateway_helper_fastpath_test.go` | 测试融合 | 用户组并发 mock。 | OpenAI WebSocket ingress lease mock。 | 合并字段和接口方法。 |
| `backend/internal/handler/gemini_v1beta_handler.go` | failover/usage | Gemini 失败用量。 | 账号级 retry limit。 | 同时保留两侧语义。 |
| `backend/internal/handler/openai_gateway_handler_test.go` | 测试 import | 请求体/用量测试 import。 | WebSocket 并发测试所需 `sync`。 | merge 时保留 import；review 后确认该文件无 `sync` 引用，作为 Task 9 early work 删除 stale import。 |
| `backend/internal/server/routes/gateway.go` | 路由与鉴权 | usage detail capture。 | Grok video edit/extension。 | 所有端点保留 capture、认证与分组中间件，并加入 edit/extension。未分组 Key 仍由 `RequireGroupAssignment` 标记受限、写入 403 并 `Abort`，不会进入 handler。 |

### 自审、Task 9 入口与风险

- CodeGraph 复核 `HandleFailoverError` 的账号级 retry limit 调用方、`ProvidePaymentHandler` 的四参签名及 `RequireGroupAssignment` 的 `Abort` 路径；没有机械选择 ours/theirs。
- `backend/internal/service/openai_first_token_timeout.go` 在 `HEAD^1..HEAD` 无变化，本地首 Token 保护仍在。
- **Task 9 入口：**审查 `v0.1.152..v0.1.153` 的无冲突能力，运行 merge 后行为/回归验证，并处理由此发现的语义修复；Task 8 不执行这些工作。
- 本机缺少 `v0.1.153` 的签名验证材料；已核验 annotated object 与指定 peel。
- Task 8 仅执行冲突与拓扑静态核验，未运行 merge 后完整测试或 build，按 Task 9 边界留待后续阶段。
- 未 push、release、deploy 或合并 main。

### Review finding: stale OpenAI handler test import（Task 9 early work）

- reviewer finding：`backend/internal/handler/openai_gateway_handler_test.go` 导入 `sync`，但文件没有 `sync` 标识符引用；`sync/atomic` 仍被使用。
- RED：`go -C backend test ./internal/handler -run '^$' -count=1` 退出 1，包含 `openai_gateway_handler_test.go:16:2: "sync" imported and not used`；同次还报告 3 个既存 `NewPaymentHandler` 三参调用与当前两参签名不匹配。
- GREEN（目标诊断）：删除唯一的 `sync` import 后重跑同一命令，`sync` unused 诊断消失；命令仍退出 1，仅保留上述 3 个 `NewPaymentHandler` 编译错误。没有将无关 PaymentHandler 修复混入本次提交。
- OpenAI 聚焦复验：`go -C backend test ./internal/handler -run '^(TestOpenAIHandleStreamingAwareError_JSONEscaping|TestOpenAIHandleStreamingAwareError_NonStreaming|TestOpenAIEnsureForwardErrorResponse_WritesFallbackWhenNotWritten)$' -count=1` 同样不再报告 `sync`，但受相同 3 个既存 PaymentHandler 编译错误阻断，目标测试未能执行。
- 修复提交：`07eba46c6edeaff011574b6be4c12d79b7317877` `fix: remove stale OpenAI handler test import`；仅含该测试文件的一行删除。
- 归属：这是 Task 9 early work。Task 8 的 merge、九项冲突台账和 OpenSpec 3.1 边界不变；本报告的本次记录以独立文档提交保存。

<a id="task-3-capability-matrix"></a>
### 能力矩阵

| 能力 | 行为契约 | 入口/调用链 | 关键文件 | 受影响 tag | 矩阵命令/证据 ID | 人工审查点 | 状态 | 阶段结果 | 证据位置 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Scheduler | 按能力、负载、privacy、sticky 与 WaitPlan 选可调度账号，不把不可用候选发往上游。 | `OpenAIGatewayService.SelectAccountWithScheduler -> layered/default scheduler -> AccountRepository`。 | `openai_gateway_scheduling.go`、`openai_account_scheduler_layered_test.go`。 | 152, 153, 155, 156 | [M-01](#m-01-scheduler) | 合并逐段确认 scheduler 模式、候选过滤和 WaitPlan 回退仍同时存在。 | protected | 阶段 0 PASS；四段均待合并前保护。 | M-01；OpenSpec `upstream-release-sync`。 |
| OpenAI previous-response/session Sticky | OpenAI 开关关闭时不读写 response/session/WS 状态；开启时保持绑定、权重与清理语义。 | `OpenAIGatewayService.Forward -> SelectAccountWithScheduler -> sticky store`。 | `openai_account_scheduler*.go`、`openai_ws_*`。 | 152, 153, 155, 156 | [M-02](#m-02-openai-sticky) | HTTP、WS V2、ingress 均须在禁用时跳过 response-account、response-connection、turn state 和 session-connection 的读写。 | protected | 阶段 0 PASS；本轮 M-16 覆盖 4/4 直接断言（exit 0）；四段均 protected。 | `platform-sticky-boundaries` 10-14；M-02、M-16。 |
| Gemini/Anthropic Sticky | 按最终平台而非 Anthropic alias 判定开关；禁用时不读写或清理会话。 | `GatewayService.stickyEnabledForPlatform -> get/bindStickySessionForPlatform -> Gemini handler/compat`。 | `gateway_service.go`、`gateway_helper.go`、`gemini_sticky_toggle_test.go`、`antigravity_smart_retry_test.go`。 | 153, 156 | [M-03](#m-03-gemini-anthropic-sticky) | Gemini 路由按 Gemini 开关选择；Antigravity retry/模型限流 cleanup 必须在 Anthropic 开关禁用时跳过删除。 | protected | 阶段 0 PASS；本轮 M-16 覆盖 4/4 直接断言（exit 0）；153、156 protected；152、155 未直接触及该调用链。 | `platform-sticky-boundaries` 16-20、27-30；附录 C、E；M-03、M-16。 |
| fallback / WaitPlan | 首选无法取得 slot 时可等待或回退；已经写出 SSE 后不再 failover。 | handler failover loop -> service selection/`WaitPlan` -> upstream error path。 | `failover_loop.go`、`openai_gateway_service.go`、`gateway_handler_stream_failover_test.go`。 | 152, 153, 155, 156 | [M-04](#m-04-fallback-waitplan) | Responses->Chat fallback 不得覆盖首 Token 或已提交响应。 | protected | 阶段 0 PASS；四段均 protected。 | M-04；`upstream-release-sync`；附录 B-E。 |
| DB recheck | stale cache sticky 候选必须在 DB 状态、模型端点、privacy/image capability 后复核；不错误删除仍可恢复的绑定。 | `SelectAccountWithScheduler -> scheduler recheck -> AccountRepository`。 | `openai_account_scheduler*.go`、`account_repo.go`。 | 152, 155, 156 | [M-05](#m-05-db-recheck) | 复核 DB recheck 发生在 cache 命中后、上游前。 | protected | 阶段 0 PASS；152-156 protected。 | M-05；`platform-sticky-boundaries`。 |
| Messages/Responses/Chat 转换与透传 | 保留 Chat->Responses->Anthropic 映射、工具/usage 字段和原始客户端 body 的 passthrough source。 | route -> `ForwardAsChatCompletions`/`ForwardAsResponses` -> apicompat -> upstream builder。 | `gateway_forward_as_*.go`、`chatcompletions_*bridge.go`、`responses_to_anthropic.go`。 | 152, 153, 155, 156 | [M-06](#m-06-protocol-conversion) | 从转换后 body 取 passthrough source 会静默丢客户端字段。 | protected | 阶段 0 PASS；四段均 protected。 | M-06；CodeGraph explore；`upstream-release-sync`。 |
| 终止 usage | 顶层 terminal usage 只记录一次；failed、partial failover 与 passthrough 保持失败 usage。 | protocol stream -> response handling -> handler usage submission。 | `responses_stream_event_wire.go`、`openai_gateway_response_handling.go`、`terminal_failed_usage_test.go`。 | 152, 153, 155, 156 | [M-07](#m-07-terminal-usage) | 流式 terminal、client disconnect 与 fallback 都不能重复提交。 | protected | 阶段 0 PASS；四段均 protected。 | M-07；v0.1.151 验证报告 191-196。 |
| privacy/内容审计 | 审计只抽取可审计内容，受限大输入仍保留最新窗口；删除分组保存时剔除且查询错误不吞掉。 | handler -> `ContentModerationService` -> audit/setting store。 | `content_moderation.go`、`content_moderation_input_test.go`。 | 155, 156 | [M-08](#m-08-content-moderation) | 审计副本限制不能改变 pre-block、模型过滤或上游 body。 | protected | 阶段 0 PASS；152-153 不触及，155-156 protected。 | M-08；`content-moderation-config`、`large-input-memory-control`。 |
| image capability | 明确 image intent 才要求 image capability；sticky DB recheck 与 image rate limit 不误伤文本账号。 | Responses/image handler -> `IsImageGenerationIntent` -> scheduler capability filter。 | `image_generation_intent.go`、`openai_images.go`、`ratelimit_service_openai_image_test.go`。 | 152, 155, 156 | [M-09](#m-09-image-capability) | raw Images、Responses tool 与 Codex bridge 的 intent 必须一致。 | protected | 阶段 0 PASS；152/155/156 protected。 | M-09；`upstream-release-sync`；CodeGraph context。 |
| 运行时设置热更新 | 持久化的 gateway 控制项可回读、校验、原子更新并立即影响新请求。 | Settings handler -> `SettingService` -> config runtime getter -> gateway service。 | `config.go`、`setting_service_gateway_runtime_test.go`、`SettingsView.gatewayRuntime.spec.ts`。 | 152, 153, 155, 156 | [M-10](#m-10-runtime-settings) | omitted 字段、Sticky/WS scheduler 和前端保存 payload 均要保留。 | protected | 阶段 0 PASS；四段均 protected。 | M-10；`openai-first-token-timeout`；v0.1.151 报告 154-163。 |
| 请求体重放与清理 | 受支持入口大 body 采用可重开 handle；failover 保持有效 body，成功/错误/提前返回均清理 owned spool。 | handler coordinator -> parsed/effective handle -> forward/failover -> cleanup。 | `request_body_coordinator*.go`、`openai_gateway_request_body.go`、`request_body_handle_test.go`。 | 152, 153, 155, 156 | [M-11](#m-11-request-body) | raw inbound 与 effective outbound 的 owner 不能重复关闭或泄漏。 | protected | 阶段 0 PASS；四段均 protected。 | M-11；`request-body-retention-control`；CodeGraph explore。 |
| 用户资源控制 | 用户级隐藏购买/自定义菜单与 blocked public group 需同时由管理写入、服务鉴权和前端路由生效。 | admin user update -> user resources -> payment/page/API-key checks -> UI visibility helpers。 | `admin_service_blocked_groups_test.go`、`payment_handler_hidden_purchase_test.go`、`page_handler_hidden_menu_test.go`。 | 152, 156 | [M-12](#m-12-user-resource-control) | 审查 API-key bind/已有 key 鉴权、管理员豁免与前端隐藏的跨层一致性。 | manual | 阶段 0 PASS；152/156 需合并后结构复核。 | M-12；`user-ui-visibility-overrides`、`user-public-group-blocklist`。 |
| 前端本地功能 | 用户不可见购买/菜单不能经导航或直达绕过；设置页保留本地 runtime 控制项。 | router guard/App UI -> visibility helper -> page navigation。 | `frontend/src/router/__tests__/feature-access.spec.ts`、`utils/userUiVisibility.spec.ts`、`SettingsView.gatewayRuntime.spec.ts`。 | 152, 153, 155, 156 | [M-13](#m-13-frontend-local-features) | route guard 与后端支付/页面拒绝不是同一断言，需联合验收。 | manual | 阶段 0 PASS；四段 UI 变更待合并后人工联检。 | M-13；`user-ui-visibility-overrides`；阶段 0 命令。 |
| 版本/依赖 | VERSION、Go toolchain/依赖与 release 元数据应来自目标 tag，不夹带本地版本漂移。 | release metadata/config -> build tooling。 | `backend/cmd/server/VERSION`、`backend/go.mod`、`backend/go.sum`。 | 152, 153, 155, 156 | [M-14](#m-14-version-dependencies) | 对每段 target tag 做 `git show <tag>^{}:backend/cmd/server/VERSION` 和 `go.mod` diff，不将编译视为版本正确性。 | manual | 阶段 0 PASS；四段均需 tag 对照。 | M-14；本节 changed-files；上游合并工作流 34-35。 |
| Ent/Wire/migrations | schema、生成 client/Wire 及 SQL migration 顺序和幂等性必须一致。 | Ent schema -> generate -> wire providers；migration runner 按文件名执行。 | `ent/schema/*.go`、`cmd/server/wire_gen.go`、`migrations/*.sql`。 | 152, 153, 155, 156 | [M-15](#m-15-ent-wire-migrations) | 新增 174/175/175a/176 与同号前缀、Wire provider 图及生成稳定性必须逐段复核。 | manual | 阶段 0 两轮生成稳定；四段合并后复跑。 | M-15；阶段 0 85-88；v0.1.151 报告 147-152。 |
| OpenAI 首 Token 超时 | 仅流式 Responses 按明确 image intent 选择 watchdog；HTTP 超时返回 504 `first_token_timeout` 且不得 failover；WS 超时 cancel、drain、完成并清理当前 turn，不处罚账号。 | OpenAI handler -> `openai_first_output_timeout` -> SSE/WS forwarder。 | `openai_first_output_timeout.go`、`terminal_failed_usage_test.go`、`openai_ws_v2/passthrough_relay_test.go`。 | 156 | [M-16](#m-16-sticky-and-first-token-rerun) | v0.1.156 合并后必须完整移除本地实现，且不得保留兼容别名；前三段保持本地实现与保护。复核 HTTP 不选择下一账号，WS cancel/drain 后才释放下一 turn。 | approved-removal | 本轮 M-16 覆盖 2/2 直接断言（exit 0）；v0.1.152-v0.1.155 protected；仅 v0.1.156 合并后可完整移除。 | `openai-first-token-timeout` 67-114；`terminal_failed_usage_test.go:223`；`passthrough_relay_test.go:104`；M-16。 |

<a id="task-3-matrix-command-definitions"></a>
### 可直接执行的矩阵聚焦命令

矩阵的 16 行各有一个稳定命令 ID。表格只引用 ID；以下 fenced code blocks 是唯一可直接执行的命令文本。`M-10` 和 `M-15` 各含同一能力行所需的多条命令。

#### M-01 Scheduler

```bash
cd backend && go test ./internal/service -run '^(TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount|TestOpenAISelectAccountWithLoadAwareness_AllFullWaitPlan)$' -count=1
```

#### M-02 OpenAI Sticky

```bash
cd backend && go test -tags unit ./internal/service -run '^(TestOpenAIGatewayService_SelectAccountWithScheduler_PreviousResponseStickyDisabledBypassesStickyLookupAndBind|TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDisabledBypassesLookupBindAndRefresh|TestOpenAIGatewayService_StickyDisabledWSv2SkipsStateStore|TestOpenAIGatewayService_StickyDisabledIngressSkipsStateStore)$' -count=1
```

#### M-03 Gemini/Anthropic Sticky

```bash
cd backend && go test -tags unit ./internal/handler ./internal/service -run '^(TestGatewayHandler_GeminiRouteStickyLookupUsesGeminiToggleNotAnthropicToggle|TestGatewayHandler_GeminiRouteStickyBindUsesGeminiToggleNotAnthropicToggle|TestGatewayHandler_OpenAIRouteStickyUsesOpenAIToggleNotAnthropicToggle|TestAntigravityGatewayServiceClearStickySessionSkipsDisabledSticky)$' -count=1
```

#### M-04 fallback / WaitPlan

```bash
cd backend && go test ./internal/handler -run '^(TestHandleFailoverError_IntegrationScenario|TestStreamWrittenGuard_MessagesPath_AbortFailoverOnSSEContentWritten)$' -count=1
```

#### M-05 DB recheck

```bash
cd backend && go test ./internal/service -run '^(TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDBRuntimeRecheckSkipsStaleCachedAccount|TestLayered_SessionStickyDBRecheckRejectsEndpointCapabilityChange|TestOpenAIGatewayService_SelectAccountByPreviousResponseID_DBRuntimeRecheckRateLimitedMiss)$' -count=1
```

#### M-06 协议转换与透传

```bash
cd backend && go test ./internal/pkg/apicompat ./internal/service -run '^(TestChatCompletionsToResponses_ToolCalls|TestResponsesToAnthropic_CustomToolPreservesSchemaParameters|TestGatewayService_ForwardAsChatCompletions_PassthroughBodyMapCopiesFromOriginalCCBody|TestGatewayService_ForwardAsResponses_PassthroughBodyMapCopiesFromOriginalResponsesBody)$' -count=1
```

#### M-07 终止 usage

```bash
cd backend && go test ./internal/pkg/apicompat ./internal/handler -run '^(TestResponsesEventToChatChunks_TopLevelTerminalUsage|TestResponsesEventToAnthropicEvents_TopLevelTerminalUsage|TestOpenAIGatewayHandler_NativeNonPassthroughResponsesFailedIsNotDuplicated|TestOpenAIGatewayHandler_ResponsesPartialFailoverCreatesExactlyOneFailedUsage)$' -count=1
```

#### M-08 内容审计

```bash
cd backend && go test ./internal/service -run '^(TestExtractContentModerationInput_ResponsesLargeInputKeepsLatestText|TestExtractContentModerationInput_ResponsesInlineImagesLimitedDuringCollection|TestContentModerationUpdateConfig_DropsDeletedGroupIDs|TestContentModerationUpdateConfig_KeepsGroupLookupErrors)$' -count=1
```

#### M-09 image capability

```bash
cd backend && go test ./internal/service -run '^(TestLayered_RequiredImageCapabilityFiltersUnsupportedAccounts|TestLayered_SessionStickyRecheckHonorsImageCapability|TestOpenAIGatewayServiceForward_RejectsDisabledImageGenerationIntents|TestOpenAIGatewayServiceForwardImages_ImageRateLimitReturnsFailoverAndCoolsCapability)$' -count=1
```

#### M-10 运行时设置热更新

```bash
go -C backend test ./internal/service -run '^(TestSettingService_SetGatewayRuntimeSettings_PersistsUpdatesCfgAndInvalidatesOnResponseHeaderTimeoutChange|TestSettingServiceGatewayRuntimeSettings_RejectsNegativeFirstTokenTimeouts)$' -count=1
pnpm --dir frontend test:run src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts
```

#### M-11 请求体重放与清理

```bash
cd backend && go test ./internal/handler ./internal/service -run '^(TestGatewayHandler_MessagesAndResponsesReplayLargeBodiesAcrossFailover|TestOpenAIGatewayHandler_ChatReplayRawSpoolAcrossFailoverWhenResponsesUnsupported|TestRequestBodyCoordinator_CleanupRemovesRawEffectiveAndMultipartTemps|TestOpenAIForwardCleansBoundRequestBodyHandlesInParallel|TestRequestBodyHandle_CleanupRemovesSpoolFile)$' -count=1
```

#### M-12 用户资源控制

```bash
cd backend && go test ./internal/service ./internal/handler -run '^(TestAdminServiceUpdateUserSavesHiddenUIResources|TestPaymentHandlerGetCheckoutInfoRejectsHiddenPurchasePage|TestPageHandlerGetPageContentRejectsHiddenCustomMenu)$' -count=1
```

#### M-13 前端本地功能

```bash
cd frontend && pnpm test:run src/router/__tests__/feature-access.spec.ts src/utils/__tests__/userUiVisibility.spec.ts src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts
```

#### M-14 版本/依赖

```bash
cd backend && go test ./cmd/server -run '^$' -count=1
```

#### M-15 Ent/Wire/migrations

```bash
go -C backend test ./migrations -run '^(TestMigration173AllowsCyberBlockedUsageRequestType|TestMigration158BackfillsGrokMediaGenerationGroups)$' -count=1
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
```

#### M-16 Sticky and first Token rerun

```bash
cd backend && go test -v -tags unit ./internal/service ./internal/handler ./internal/service/openai_ws_v2 -run '^(TestOpenAIGatewayService_SelectAccountWithScheduler_PreviousResponseStickyDisabledBypassesStickyLookupAndBind|TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDisabledBypassesLookupBindAndRefresh|TestOpenAIGatewayService_StickyDisabledWSv2SkipsStateStore|TestOpenAIGatewayService_StickyDisabledIngressSkipsStateStore|TestGatewayHandler_GeminiRouteStickyLookupUsesGeminiToggleNotAnthropicToggle|TestGatewayHandler_GeminiRouteStickyBindUsesGeminiToggleNotAnthropicToggle|TestGatewayHandler_OpenAIRouteStickyUsesOpenAIToggleNotAnthropicToggle|TestAntigravityGatewayServiceClearStickySessionSkipsDisabledSticky|TestOpenAIGatewayHandler_FirstTokenTimeoutReturns504AndCreatesOneFailedUsage|TestRelayFirstTokenTimeoutCancelsDrainsAndCompletesTurn)$' -count=1
```

### 本轮 M-16 执行证据

- 执行目录：`backend`；退出码：`0`。M-16 的正则以未转义 `|` 连接 10 个目标顶层测试名。
- 顶层目标总数：10；实际 `=== RUN` 目标数：10；实际顶层 `--- PASS` 数：10。Gemini/OpenAI 路由测试还输出 6 个子测试 RUN/PASS，以下完整列出全部 16 个命名 RUN/PASS 条目。

```text
=== RUN   TestAntigravityGatewayServiceClearStickySessionSkipsDisabledSticky
--- PASS: TestAntigravityGatewayServiceClearStickySessionSkipsDisabledSticky
=== RUN   TestOpenAIGatewayService_SelectAccountWithScheduler_PreviousResponseStickyDisabledBypassesStickyLookupAndBind
--- PASS: TestOpenAIGatewayService_SelectAccountWithScheduler_PreviousResponseStickyDisabledBypassesStickyLookupAndBind
=== RUN   TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDisabledBypassesLookupBindAndRefresh
--- PASS: TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDisabledBypassesLookupBindAndRefresh
=== RUN   TestOpenAIGatewayService_StickyDisabledIngressSkipsStateStore
--- PASS: TestOpenAIGatewayService_StickyDisabledIngressSkipsStateStore
=== RUN   TestOpenAIGatewayService_StickyDisabledWSv2SkipsStateStore
--- PASS: TestOpenAIGatewayService_StickyDisabledWSv2SkipsStateStore
=== RUN   TestGatewayHandler_GeminiRouteStickyLookupUsesGeminiToggleNotAnthropicToggle
=== RUN   TestGatewayHandler_GeminiRouteStickyLookupUsesGeminiToggleNotAnthropicToggle/gemini_disabled_bypasses_lookup_even_when_anthropic_enabled
=== RUN   TestGatewayHandler_GeminiRouteStickyLookupUsesGeminiToggleNotAnthropicToggle/gemini_enabled_performs_lookup_even_when_anthropic_disabled
--- PASS: TestGatewayHandler_GeminiRouteStickyLookupUsesGeminiToggleNotAnthropicToggle
--- PASS: TestGatewayHandler_GeminiRouteStickyLookupUsesGeminiToggleNotAnthropicToggle/gemini_disabled_bypasses_lookup_even_when_anthropic_enabled
--- PASS: TestGatewayHandler_GeminiRouteStickyLookupUsesGeminiToggleNotAnthropicToggle/gemini_enabled_performs_lookup_even_when_anthropic_disabled
=== RUN   TestGatewayHandler_GeminiRouteStickyBindUsesGeminiToggleNotAnthropicToggle
=== RUN   TestGatewayHandler_GeminiRouteStickyBindUsesGeminiToggleNotAnthropicToggle/gemini_disabled_bypasses_bind_even_when_anthropic_enabled
=== RUN   TestGatewayHandler_GeminiRouteStickyBindUsesGeminiToggleNotAnthropicToggle/gemini_enabled_writes_bind_even_when_anthropic_disabled
--- PASS: TestGatewayHandler_GeminiRouteStickyBindUsesGeminiToggleNotAnthropicToggle
--- PASS: TestGatewayHandler_GeminiRouteStickyBindUsesGeminiToggleNotAnthropicToggle/gemini_disabled_bypasses_bind_even_when_anthropic_enabled
--- PASS: TestGatewayHandler_GeminiRouteStickyBindUsesGeminiToggleNotAnthropicToggle/gemini_enabled_writes_bind_even_when_anthropic_disabled
=== RUN   TestGatewayHandler_OpenAIRouteStickyUsesOpenAIToggleNotAnthropicToggle
=== RUN   TestGatewayHandler_OpenAIRouteStickyUsesOpenAIToggleNotAnthropicToggle/openai_disabled_bypasses_messages_lookup_and_bind_even_when_anthropic_enabled
=== RUN   TestGatewayHandler_OpenAIRouteStickyUsesOpenAIToggleNotAnthropicToggle/openai_enabled_reads_and_binds_messages_session_even_when_anthropic_disabled
--- PASS: TestGatewayHandler_OpenAIRouteStickyUsesOpenAIToggleNotAnthropicToggle
--- PASS: TestGatewayHandler_OpenAIRouteStickyUsesOpenAIToggleNotAnthropicToggle/openai_disabled_bypasses_messages_lookup_and_bind_even_when_anthropic_enabled
--- PASS: TestGatewayHandler_OpenAIRouteStickyUsesOpenAIToggleNotAnthropicToggle/openai_enabled_reads_and_binds_messages_session_even_when_anthropic_disabled
=== RUN   TestOpenAIGatewayHandler_FirstTokenTimeoutReturns504AndCreatesOneFailedUsage
--- PASS: TestOpenAIGatewayHandler_FirstTokenTimeoutReturns504AndCreatesOneFailedUsage
=== RUN   TestRelayFirstTokenTimeoutCancelsDrainsAndCompletesTurn
--- PASS: TestRelayFirstTokenTimeoutCancelsDrainsAndCompletesTurn
```

- package 结果：`internal/service` 2.269s、`internal/handler` 3.235s、`internal/service/openai_ws_v2` 0.375s，均为 `ok`。

### 矩阵统计

- 命令 ID：`M-01` 至 `M-16` 共 16 个，对应全部 16 行；code blocks 共 19 条可执行 shell 命令，其中 15 条含 Go `-run`。

| 状态 | 数量 | 结论 |
| --- | ---: | --- |
| protected | 11 | 有直接行为断言且阶段 0 基线已通过。 |
| manual | 4 | 跨层资源控制、前端联检、版本依赖、Ent/Wire/migration 不能由单一测试替代。 |
| approved-removal | 1 | 首 Token 超时仅在 v0.1.156 后执行。 |
| gap | 0 | 没有任何能力同时满足“本地独有、目标 release 触及、缺少关键行为断言”；Task 4 无补测输入。合计 16 行：brief 的 15 项能力加首 Token `approved-removal` 预登记。 |

### 自审、风险信号与顾虑

- 自审：矩阵覆盖 brief 列出的 15 类高风险行为，并保留首 Token `approved-removal` 预登记，共 16 行；每行均有唯一状态、阶段结果、调用链、关键文件、tag、精确命令、人工点和证据位置。未将“测试存在”当作完整保护，四项跨层/生成契约明确标为 `manual`。附录 A-E 持久化了五条指定 Git 命令的完整 stdout；已据此修正 fallback、OpenAI Sticky 和 Gemini/Anthropic Sticky 的受影响 tag。
- 工作树：只允许本报告变更；`docs/superpowers/plans/2026-07-16-staged-merge-upstream-v0-1-156.md`、`openspec/changes/staged-merge-upstream-v0-1-156/.comet/subagent-progress.md` 与 `.comet/current-change.json` 保持未暂存、未修改、未提交。
- 风险信号：目标范围 503 文件且覆盖网关、apicompat、Ent、Wire、配置与 UI；`upstream/main` 不属于本次 merge 目标；阶段 0 已记录的 Browserslist、动态/静态 import、chunk 大小警告仍非阻塞信号。
- 顾虑：四段清单的非高风险文件需在各阶段 merge 时仍执行原始命令复核；本报告不声称尚未合并代码已通过目标 release 测试，也不替代后续实际阶段验证。
- 阻塞结论：解除。五份原始 stdout 已完整附录，所有矩阵行字段完整且状态唯一；`gap=0` 不降低后续逐阶段验证标准。

### Task 3 reviewer evidence repair（本轮命令修复）

- 范围：只修复验证证据；本轮仅修改本报告。源码、测试、生成物、plan、OpenSpec task、Comet progress、`opencode.json`、`.comet/current-change.json` 与 `.superpowers/` 均未修改、暂存或提交。五份原始清单、16 行矩阵和前两次文档提交信息保留。
- 命令修复：矩阵的“矩阵命令/证据 ID”列只保存 M-01 至 M-16 的稳定锚点；16 个完整命令均在“可直接执行的矩阵聚焦命令”以 fenced code block 保存。所有 Go `-run` alternation 都使用原始 `|`，不使用反斜杠竖线；M-10、M-15 的非单测/生成证据也在对应 ID 中完整列出。
- 真实重跑：M-16 使用 `-v`，退出码 `0`。10 个目标顶层测试各出现一次 RUN 和 PASS；6 个路由子测试亦全部 RUN/PASS。完整命令、测试名及 package 结果见 M-16 执行证据，不能再以仅 package PASS 代替命名测试证明。
- 既有 reviewer 闭环保留：OpenAI Sticky 覆盖 ingress state-store bypass；Gemini/Anthropic Sticky 覆盖 Antigravity 禁用 cleanup；首 Token 覆盖 HTTP 504/单次上游尝试和 WS cancel-drain-turn cleanup；Gemini/Anthropic 的附录为 C、E；首 Token 仅在 v0.1.156 后完整移除且无兼容别名，前三段仍 protected。
- 矩阵复评：`protected=11`、`manual=4`、`approved-removal=1`、`gap=0`、合计 16。`gap=0` 的依据是每个能力均有矩阵命令或明确人工/生成证据，且 protected 行有直接行为断言；它不声称本轮执行了全部 M-01 至 M-15，也不表示目标 tag 已合并或验证完成。
- 提交：本报告由普通文档提交 `docs: fix capability test commands` 承载；不 amend、merge、push、release 或 deploy。
- 自审：表格没有可执行正则；所有 `-run` fenced 命令逐项扫描，均不含反斜杠竖线。文档证据修复依用户裁决豁免 TDD，未执行或伪造 RED/GREEN。
- 风险信号与顾虑：目标范围仍为 503 文件，且阶段 0 的 Browserslist、动态/静态 import 与 chunk 大小警告仍存在；首 Token 的完整移除仅可在 v0.1.156 合并后检查上游最终树，当前基线不得提前删除保护。

### Task 3 reviewer evidence repair（第四轮工作目录语义）

- 第四个提交：`docs: fix matrix command working directories`。本轮只修改本报告；plan、OpenSpec task、Comet progress、`opencode.json`、`.comet/current-change.json`、`.superpowers/`、源码、测试和生成物均未修改、暂存或提交。
- M-10 从仓库根目录依次执行：`go -C backend test ./internal/service -run '^(TestSettingService_SetGatewayRuntimeSettings_PersistsUpdatesCfgAndInvalidatesOnResponseHeaderTimeoutChange|TestSettingServiceGatewayRuntimeSettings_RejectsNegativeFirstTokenTimeouts)$' -count=1` 退出码 `0`，两个命名目标所在的 `internal/service` package `ok`（1.503s）；`pnpm --dir frontend test:run src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts` 退出码 `0`，1 个测试文件、14/14 测试通过（测试 1.211s，总计 28.62s）。
- M-15 从仓库根目录依次执行：`go -C backend test ./migrations -run '^(TestMigration173AllowsCyberBlockedUsageRequestType|TestMigration158BackfillsGrokMediaGenerationGroups)$' -count=1` 退出码 `0`，两个命名目标所在的 `migrations` package `ok`（0.461s）；`make -C backend generate` 退出码 `0`，完成 Ent 与 Wire 生成；`git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` 退出码 `0`、stdout 为空，生成 diff 为空。
- 审查闭环：M-10 和 M-15 的唯一可执行命令定义均可从仓库根目录逐行或整块执行，使用 `go -C backend`、`pnpm --dir frontend`、`make -C backend` 或根目录 `git diff`，不持续改变调用 shell 的 cwd。对两个命令 block 的 `cd` 扫描为零。
- 矩阵统计不变：`protected=11`、`manual=4`、`approved-removal=1`、`gap=0`、合计 16；本轮是纯文档证据修复，依用户裁决豁免 TDD。
- 风险信号与顾虑：前端命令仍输出既有非阻塞 Browserslist 数据陈旧警告；本次仅证明当前根目录命令与生成稳定性，后续各 tag 合并阶段仍须执行对应原始验证。

<a id="task-3-review-conclusion"></a>
### Task 3 审查结论

- final review 结论：`Approved`；`Critical`、`Important`、`Minor` 均无。
- 最终实现提交：`abc694a4d6cb1ec7c6c8ba76a49ac28c056f6e00`（`docs: fix matrix command working directories`）。

## 阶段 0 最小行为保护门禁（OpenSpec 1.4）

### 零 gap 与测试结论

- 零 gap 判定参照 [Task 3 的 16 行能力矩阵](#task-3-capability-matrix)、[M-01 至 M-16 命令定义](#task-3-matrix-command-definitions) 与 [Task 3 审查结论](#task-3-review-conclusion)：`protected=11`、`manual=4`、`approved-removal=1`、`gap=0`。
- 没有任何能力同时满足“本地独有、目标 release 触及、缺少行为断言”三项条件，因此无符合条件的补测；新增 characterization test 文件数为 `0`，新增测试数为 `0`，无新增聚焦命令。
- 本轮无行为修改，按用户裁决豁免 TDD；未执行或伪造 RED/GREEN。Task 3 已真实运行的 M-16（10 个顶层测试）及 M-10/M-15 仅为既有证据，未替代下列完整阶段 0 门禁。

### 完整门禁记录

- 下表五条门禁命令均从仓库根目录 `D:\Caiqy\Projects\Github\sub2api` 执行；`--dir frontend` 与 `-C backend` 仅由该根目录命令指定子目录。本次文档修复未重跑门禁，也未补写原始 stdout。

| 命令 | 退出码 | 摘要 |
| --- | ---: | --- |
| `make test` | 0 | 后端默认测试、`golangci-lint`（`0 issues`）、unit-tag 测试、前端 ESLint 与 `vue-tsc --noEmit` 通过；Vitest 为 `167` 个文件、`1246` 个测试通过。 |
| `pnpm --dir frontend run build` | 0 | `vue-tsc -b` 与 Vite 生产构建通过；转换 `966` 个模块，`built in 36.71s`。 |
| `make -C backend generate` | 0 | `go generate ./ent` 与 `go generate ./cmd/server` 完成；Wire 写入 `backend/cmd/server/wire_gen.go`。 |
| `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` | 0 | stdout 为空；生成 diff 为空。 |
| `git diff --check` | 0 | 无空白错误。 |

### 警告与工作树

- `make test`：Browserslist 数据已 7 个月未更新；Vitest 有已覆盖错误路径日志、`router-link` 解析警告及 intlify message compiler 警告，全部断言仍通过。
- 前端构建：重复 Browserslist 提示；动态/静态 import 混用提示；`AccountsView` 压缩后 `635.06 kB`，超过 `500 kB` chunk 警戒线。均不阻塞本阶段。
- `git diff --check`：仅提示用户既有 `docs/superpowers/plans/2026-07-16-staged-merge-upstream-v0-1-156.md` 与 `openspec/changes/staged-merge-upstream-v0-1-156/.comet/subagent-progress.md` 下次 Git 写入会发生 LF/CRLF 转换。
- 门禁开始时工作树已有 `.superpowers/sdd/task-4-report.md` 删除、上述 plan 与 Comet progress 修改、`.comet/current-change.json` 未跟踪；均未触碰。`opencode.json` 保持用户保留的未提交状态，未暂存或提交。生成后未出现 `backend/ent`、`backend/cmd/server/wire_gen.go` 或其他生成物差异。

### 自审与放行

- 本轮只修改本报告；未修改或暂存测试、业务代码、生成源码、plan、OpenSpec task、Comet progress、`opencode.json`、`.comet/current-change.json`、`.superpowers/`、`backend` 或 `frontend`。
- 暂存前后将核验 `git diff --cached --check` 与暂存文件清单；提交仅含本报告，message 为 `docs: close stage zero protection gate`。该报告的最终提交 SHA 以提交后 Git 记录为准，避免将自引用 SHA 写入提交内容而改变对象。
- 阶段 0 门禁提交：`8294bf1f827698b1b9b696d0e46a66a0d439b8a7`（`docs: close stage zero protection gate`）。
- 证据链接修复提交：`f324b7016dc22a4bf0d8000f7d93954af199652c`（`docs: link stage zero protection evidence`）。
- 所有阶段 0 门禁退出 `0`、生成 diff 为空，允许进入 Task 5；本任务未执行 merge、push、release、deploy 或 main 合并，未勾选任务。

### 风险与顾虑

- 目标范围仍覆盖网关、协议、Ent/Wire、配置与 UI；`gap=0` 仅说明当前本地独有高风险能力已有直接断言或明确人工/生成证据，不替代各 target tag 合并后的阶段验证。
- 首 Token 完整移除及无兼容别名只能在 `v0.1.156` 合并后的最终树复核；前三段不得提前删除保护。

## Task 6 / v0.1.152 能力审查（OpenSpec 2.2）

### 范围与交集

- 审查基准：merge commit `4ffe039a4399f8cbac1f83df32b709afda777ffe`，release diff `v0.1.151..v0.1.152` 为完整的 `128` 个路径；merge 相对第一父的最终树差异为 `126` 路径，系融合后树口径。`git diff --check 4ffe039a^ 4ffe039a`、当前 `git diff --check`、未解决路径和冲突标记扫描均为退出码 `0`/空输出。
- 与 Task 3 的 16 行矩阵直接交集为 `13` 行：M-01、M-02、M-04、M-05、M-06、M-07、M-09、M-10、M-11、M-12、M-13、M-14、M-15。M-03、M-08 未由 152 触及；M-16 虽在矩阵登记为 156 的 approved-removal，但 152 冲突融合实际触及 fallback watchdog，且本地首 Token 在前三段继续 protected，故作为第 `14` 项复核。
- 交集覆盖 scheduler/Sticky/WaitPlan、DB recheck、apicompat 与 usage、image capability、runtime/body cleanup、用户资源与前端、版本、Ent/Wire/migration 和首 Token；其余 changed files 依路径归入 account/Grok quota、API key cache、group web-search pricing、route/DTO、前端 account/settings/i18n 功能逐项人工复核。

### CodeGraph 与调用链

- 执行 `context`（first Token、scheduler、Sticky、WaitPlan、recheck、body、Create/Edit）和一次 `explore`（firstTokenTimeout、scheduler、sticky、WaitPlan、body replay、Group、Create/Edit、passthrough）；另执行 `impact(supportsPassthroughFields)`、`impact(ValidatePeakRateConfig)`、`impact(getDefaultBaseUrl)`、`impact(WebSearchPricePerCall)`、`impact(resolveOpenAIUpstreamEndpoint)`，以及 CreateGroup、OpenAIGatewayService、CreateAccountModal 的按需 trace。
- `CreateGroup -> ValidatePeakRateConfig -> parseMinutes` 保持单一校验入口。`WebSearchPricePerCall` 影响 Ent create/update、repository create/update、service Group、DTO 和 GroupsView；schema、migration `174`、Create/Edit 归一化及前端 null/default 往返均存在。
- OpenAI service/component trace 在动态 dispatch 处停止；源码复核确认 `Forward/handler -> SelectAccountWithScheduler -> recheck/sticky store`、route -> ForwardAsChatCompletions/Responses -> apicompat、以及 Create/Edit -> `getDefaultBaseUrl` 的静态边界。Grok helper 返回 `https://api.x.ai/v1`，Create 切换平台和 Edit 默认值均依赖该 helper。

### 能力结论

| M-ID | 状态 | 结论 |
| --- | --- | --- |
| M-01 Scheduler | PASS | WaitPlan 与受限候选过滤仍在上游前完成。 |
| M-02 OpenAI Sticky | PASS | disabled 时 HTTP/WS/ingress 均不读写状态。 |
| M-03 Gemini/Anthropic Sticky | N/A | 152 无直接 changed-file 交集。 |
| M-04 fallback/WaitPlan | PASS | SSE 已写入保护及 fallback 路径通过。 |
| M-05 DB recheck | PASS | stale sticky/cache 在调度后、上游前复核。 |
| M-06 协议转换/透传 | PASS | 原始 source body 与 effective mapped body 分离后，body map 和工具转换均通过；原矩阵命令漏 unit tag，已补跑。 |
| M-07 终止 usage | PASS | terminal/failed/partial failover 不重复记账。 |
| M-08 内容审计 | N/A | 152 无直接 changed-file 交集。 |
| M-09 image capability | PASS | image intent/capability/rate-limit 过滤保持。 |
| M-10 运行时设置 | PASS | 后端持久化校验和 SettingsView 14 项断言通过。 |
| M-11 body replay/cleanup | PASS | raw/effective handle 的 failover 与 cleanup 通过。 |
| M-12 用户资源控制 | PASS/manual | 管理写入、支付/页面拒绝、管理员豁免及路由/visibility helper 已联检。 |
| M-13 前端本地功能 | PASS/manual | 路由 guard、hidden menu helper、runtime card 联检通过。 |
| M-14 版本/依赖 | PASS/manual | tag 为 `0.1.151`，merge 保留获批准本地四段 `0.1.151.2`；Go `1.26.5` 与 `golang.org/x/mod v0.35.0` 一致。 |
| M-15 Ent/Wire/migrations | PASS/manual | schema -> generated client -> idempotent SQL 174 -> Create/Edit/UI 链路完整；生成无 diff。 |
| M-16 首 Token | PASS/protected | HTTP 504/单次 failed usage、WS cancel-drain-turn 均通过；未提前执行 156 的移除。 |

### 命令记录

| 范围 | 命令结果 | 退出码 |
| --- | --- | ---: |
| M-01 | `go test ./internal/service -run WaitPlan targets -count=1` | 0 |
| M-02 | `go test -tags unit ./internal/service -run OpenAI Sticky targets -count=1` | 0 |
| M-04 | `go test ./internal/handler -run failover/SSE-written targets -count=1` | 0 |
| M-05 | `go test ./internal/service -run DB recheck targets -count=1` | 0 |
| M-06 | 原矩阵 `go test ./internal/pkg/apicompat ./internal/service -run conversion/passthrough targets -count=1` | 0 |
| M-06 补充 | `go test -tags unit ./internal/service -run ForwardAs(ChatCompletions|Responses)_PassthroughBodyMapCopies -count=1` | 0 |
| M-07 | `go test ./internal/pkg/apicompat ./internal/handler -run terminal usage targets -count=1` | 0 |
| M-09 | `go test ./internal/service -run image capability targets -count=1` | 0 |
| M-10 | backend runtime setting targets；`pnpm --dir frontend test:run SettingsView.gatewayRuntime.spec.ts`（14/14） | 0 / 0 |
| M-11 | `go test ./internal/handler ./internal/service -run body replay/cleanup targets -count=1` | 0 |
| M-12 | 原矩阵 service/handler 命令；补充 `go test -tags unit ./internal/service ./internal/handler -run hidden-resource targets -count=1` | 0 / 0 |
| M-13 | `pnpm --dir frontend test:run feature-access userUiVisibility SettingsView.gatewayRuntime`（21/21） | 0 |
| M-14 | `go test ./cmd/server -run '^$' -count=1` | 0 |
| M-15 | migration targets；`make -C backend generate`；`git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` | 0 / 0 / 0 |
| M-16 | `go test -v -tags unit ./internal/service ./internal/handler ./internal/service/openai_ws_v2 -run StickyAndFirstTokenTargets -count=1`（10 顶层目标） | 0 |
| Task 5 冲突入口 | Alpha search route/service、endpoint/no-account/usage、API-key cache/Grok quota、OAuth passthrough、Create/Edit Grok、FastPolicy locale/selector | 全部 0 |
| Task 6 alpha-search reviewer fix | `go test ./internal/service -run '^TestForwardAlphaSearchAPIKeyMapsModelAndPassesThroughError$' -count=1`（RED 1，GREEN 0）；`-run '^TestForwardAlphaSearch'`、`-run '^TestApplyAccountPassthroughFields'`、`-run '^(TestOpenAIOAuthPassthrough|TestForwardOpenAIPassthrough|TestOpenAIForward.*Passthrough)'` | GREEN 均 0 |
| 扫描/人工 | merge/current `git diff --check`、conflict marker scan、`git ls-files -u`、tag/version/dependency 对照 | 全部 0/空 |

### RED/GREEN 与修复

- 已有 Task 6 early work 已核对：`b19c03d01` 在 `getDefaultBaseUrl('grok')` 从错误的 Anthropic 回退修复为 xAI URL；真实 RED/同命令 GREEN 为 `pnpm --dir frontend test:run src/components/account/__tests__/passthroughFieldSupport.spec.ts`。`2026265cb` 为该 RED/GREEN 证据提交，Create/Edit 表单依赖已由本轮 56/56 前端聚焦测试确认。
- 本轮真实 RED：所有受影响 service/handler 命令先稳定报错。根因一是 152 新增 alpha search 与本地 body-handle/Grok/fallback/WS 签名演进未同步；根因二是上游两参 `resolveOpenAIUpstreamEndpoint` 与本地 result-aware 三参实现同时保留。GREEN：对应 alpha-search、Grok preview、first-token fallback、WS bridge、endpoint/usage、M-04/M-07/M-11/M-16 全部通过。
- 新增普通提交 `03c45833e` `fix: restore v0.1.152 gateway call contracts`：最小同步调用、保留 result-aware endpoint 优先级，并合并 nil-account fallback。未 amend/rewrite `4ffe039a`。
- reviewer finding：`03c45833e` 中“alpha-search 将未改写 body 同时作为 source/effective body 传递”的结论错误。`ForwardAlphaSearch` 先映射 `model`，随后把映射后的 body 两次传给 passthrough builder；账号 body map/forward 因而只能读取上游 model，无法读取客户端原始字段。
- 真实 RED：`go test ./internal/service -run '^TestForwardAlphaSearchAPIKeyMapsModelAndPassesThroughError$' -count=1` 退出 `1`，期望 `client_model=gpt-5.6-sol`，实际为 `upstream-5.6`。真实 GREEN：同一命令退出 `0`；`go test ./internal/service -run '^TestForwardAlphaSearch' -count=1`、`go test ./internal/service -run '^TestApplyAccountPassthroughFields' -count=1` 和 `go test ./internal/service -run '^(TestOpenAIOAuthPassthrough|TestForwardOpenAIPassthrough|TestOpenAIForward.*Passthrough)' -count=1` 均退出 `0`。
- `874843826` `fix: preserve alpha search passthrough source body` 在模型映射前保留 `sourceBody`，仅 alpha-search 私有请求构造函数分别传入 source/effective body；通用 passthrough builder contract 与其他路径不变。
- 本轮前端 RED：Create Grok source test 仍要求内联 xAI literal；运行时已正确集中在 early-work helper。GREEN：改为断言 Create 依赖 helper，`d81633a6e` `test: align Grok account modal helper contract`，56/56 通过。

### 人工审查、风险与顾虑

- manual：15 项 Task 5 冲突逐一覆盖 VERSION、Ent descriptors、route/endpoint usage、API key cache、Grok pool/quota、fallback watchdog、usage snapshot、OAuth passthrough、Create/Edit、locale/selector；未发现不可共存语义。首 Token 保持 protected，未触及 156 removal。
- 风险信号：Task 3 原 M-06/M-12 的不带 `-tags unit` 命令对 service 目标显示 `no tests to run`；本轮原命令与显式 unit-tag 补充命令均记录为 0，应在后续矩阵维护时修正原命令。
- 顾虑：annotated tag 本机仍无签名材料，只能依据固定 peel SHA；VERSION 有意不同于 tag，最终发布阶段仍须复核。前端命令持续输出既有 Browserslist 数据陈旧警告，未阻断断言。
- Task 6 结论：`DONE`。除上述已修复的三项后合并回归和已归类 Grok early work 外，没有未解释的 v0.1.152 行为回归；未运行 Task 7 完整阶段门禁，未修改 plan、OpenSpec task、Comet progress、`.comet/current-change.json` 或 `.superpowers`。

## 正式证据附录（五份原始 stdout）

以下代码块逐行保存指定命令的 stdout；行数以命令输出的非空记录数计。附录 A 的 534 个 Git 输出记录含 8 个历史 subject 内嵌回车，因而持久化为 542 个物理行；其余附录的记录数与物理行数一致。

### 附录 A：本地独有提交（534 个 Git 输出记录，542 个物理行）

命令：git log --format='%H %s' v0.1.151^{}..d5f8192d32d9840d63477c24d4a567abb8cb4a90

```text
d5f8192d32d9840d63477c24d4a567abb8cb4a90 test: isolate Go test temporary files
d1cc02502271f54b3b7f0593a18db4f2aaab63ea chore: archive restore-local-test-gates
981c713ed08f7fa20894ec3abadc31ff8f6d7ce4 chore: complete local gate verification
15ed75d627a174d9a9b142322b9d41d756030526 chore: enter local gate verification
1c22cb5fb5d528a9c35e53b8c1b08edd4b18260c test: verify local test gate fixes
5a3166faf24dbcb45c16ccb479729b49db377bd0 chore: restore local test gate
1c25d74c58cbfe7f22e2b6ff0e04d81463afada6 fix: resolve task 6 lint diagnostics
b596ac66aabfb3a42e8b9bdc195e577234f3e59e fix: resolve task 5 lint diagnostics
5f20a851463f648d754ee327f56d99d1d6e48f5d fix: stabilize Grok media heap test
4c9e492cc107d27d9ae1dcc10a6360d7b6b2d42b fix: release Grok edit forward body
4509395c09b13be762754386cbe52f952f8481b0 test: isolate admin usage stats cache
84ee2fe046cf42fc3e44f0e6609bfe52685c69cc test: assert bound openai body handle reuse
fc752398c6040eefe8ec83ad162bfd94493fa573 fix: preserve bound openai request bodies across retries
df5aae61a352b18017e18368d869acba3efe74bb fix: clean bound openai request body handles
8f10ca847a07ec9446cb80fafa6ccfb2d1f1247b chore: record user repository stub progress
55dba42752d60b4815de69dc576cfbca6fbbf769 test: update server package contracts
f2c370d2f896ea285650eede2df8982338c601cd test: complete user repository stubs
1bce5bd8d6e4244353ab0be063c86340e5f4b09b chore: record images failover test progress
23bd1b4dab1e28d8319cc37a7c6f6a2b369d399d test: focus images failover exhaustion
dc2bad551a9caba3425c37e96d5ac34164d69c9e chore: record failed usage test progress
78ed75bfb4c697d3d2855aac044f1d4f72d39903 test: restore failed usage header assertions
a6072374b2c7f1b397a6cfbb63d398914d131a91 chore: configure local test gate build
a962aa00a560bb493b8a10f111d405e5199210b4 chore: plan local test gate recovery
ddefbbffa13569f973aee4bb2802eb2414c7d70f docs: design local test gate recovery
0c1c4bc14d282ad4b65ee41c053bf222a532bfdd chore: remove unfinished openspec changes
7bc3a02e49c98892837ab203ce533e69c609bba0 chore: archive service unit regressions
bda166b9d84a92c58c0cfd54f9536f052728ab78 docs: record service regression verification
d1316c68a1cd6ea16fb35b665053685ccc6687ef Merge branch 'hotfix/20260715/restore-atomic-subscription-quota-reset' into integration/20260715/service-unit-regressions
693e231bbd54856764dd80808ce13bacac3f1365 test: align atomic reset fixture
6d58db693b5f9fb840327cf727b97cd22a95c232 Merge branch 'hotfix/20260715/restore-atomic-subscription-quota-reset' into integration/20260715/service-unit-regressions
279335b3459a0a5edbbb412d51c0e6ac3adfa0cc Merge branch 'hotfix/20260715/restore-settings-json-default-backfill' into integration/20260715/service-unit-regressions
5f68e0e693a056adb0d519772ca3a13b5ef22a78 Merge branch 'hotfix/20260715/repair-service-unit-test-fixtures' into integration/20260715/service-unit-regressions
e313e014dfb9c5e351fd49182e807bc3a0f1e61f chore: record quota reset verification state
fdeef923489b4fadc3933a63a7c88c883488a2e3 fix: restore atomic subscription quota reset
06b19c3d9cdba0629e4ccc4f632582be54634e17 chore: record settings backfill verification state
db4d78ab37414dbb6af4cd26c531d384f288435d fix: restore settings json default backfill
b1cd0d5432ab8374f98a2f7f59a8a2e9f58a0cf0 chore: record fixture verification state
4c64ce725a6a44816ace5d137378a0bf106b0905 test: close scheduler probe leaks
6b51854c28d450873c6d0c70ee9f7973f52b059f test: stop image scheduler probe
063c1d60998d20195f3040469bb7522348a10533 test: stop layered scheduler probes
cf5e3694bf3f531637e9459e0777194c4451d591 test: repair settings service fixture
4e7da8c35b4f71476eaa91c33806792128a950aa test: align pricing resolver fixture
2235e1046672ef1ad6973da2882c1526bc39bf5a test: repair credits retry fixture
7cad1a887caafd1955a9764abc6fe8852cab26f3 chore: archive restore-openai-grok-body-replay
ca940ace70653ff875ca8705e041bba9e7c3ab82 chore: record body replay verification
8626af15e0cde470a1200002329fa7f52691d386 fix: enforce grok multipart body limits
90a66cd49fecb283b503e6c86b56e3bf44386ce7 fix: preserve grok multipart upload data
a572b9f3a28d4ba989c8e0268f10d8c611999e4c fix: restore openai raw body replay
4e385e52cd61be48b689fe34473769d57bf78a5d docs: record sticky routing verification
f8088e79129859b8c6a62b22048db28aa1e2d645 chore: archive restore-platform-sticky-routing
c598e63df948f56c574fbe70a3088205ee92c771 chore: record sticky routing verification
74a3021ac0b8fc1b13634756dad93b1943655844 chore: complete sticky cleanup task
398087a0367855f2d463494cbe5d31262f731ba7 fix: propagate sticky cleanup through rectifiers
05138969dbcabf665c56211c0f4f54f2d7b0683c fix: propagate invalid fallback sticky cleanup
44aeb06f8f8f28294e881cc57bb7ea059542104d chore: record sticky routing review
e534f0137689dc2bbc63955ee846691f7b1e30aa fix: forward resolved Gemini fallback via compat
aa684ec1e2e9d219dfd01e1075db07ae9931879c test: update gateway resolver coverage
ed4a88cd5e77de3c38334d9567ab6e244b87a107 test: cover invalid fallback sticky resolution
e3cfdc018c8e0c3b60a962c8d22d044a142764be test: cover fallback antigravity cleanup
951e0cbfc70269e129b94f74fb984c6d3f9b4d06 test: cover fallback Gemini sticky bind
3e30b60974fd6249a9ffd5def81391040d83db12 refactor: share gateway group resolution
bb70391901d4c87213f6f115ed11f4b31a08b1ff docs: centralize sticky route resolution
21f5dfa5ece9082bf70d148f5b49d7054877d47d fix: resolve sticky route fallback group
8e962fcfa72695133e1c97fb6485cecd76a57518 fix: honor gateway sticky platform toggle
17ca158d672538106639b7b75592a11ccbd70435 test: enable sticky store-false isolation fixture
c76c261cc68b8748769b9c7b39507d1bf8b1fcdb test: verify platform sticky boundaries
9883810d4decd60b6969419467b756e964730165 fix: honor platform sticky toggles in gemini compat
fc9e82fec6d209712af53eb560cd0a0fdcf01b27 test: cover ingress sticky state bypass
32e25ba9124dfbbc2b056264bbe1d4c0305acbd5 test: strengthen OpenAI websocket sticky coverage
ed857d619d57f60ebab779c4be43006107f321dd fix: gate OpenAI websocket sticky state
f4883639075434789e64b666266cd6d190a3e316 fix: restore platform sticky routing semantics
d2bb2370986c592fd398e8b3b81f10b09358b093 docs: design platform sticky boundaries
ec35ea633ddb44913189b4b4f3ef14041ee048a9 chore: archive restore-admin-user-resource-controls
6e71b2556c2fbef41a776ab4fd17fc4054f9723f chore: record admin resource verification
adf9ac59020f668016d05cd3ad189ced65fc6c01 fix: restore admin user resource controls
b65b338e7c10cf1813cc0a9d70f57686f6c7c434 fix: harden subscription cache consistency
1f00e6c558d4c70b1ec92802b010b10391aae844 Merge remote-tracking branch 'origin/main'
fa2841917468b97bfc33be591839d79611b86d2c docs: update wire verification report
5350370ad9cdc3fc5b13917d136dc0e3bab6b20a ci: simplify linux amd64 release
a7a73279243ac9f81ba219b3256f5c4108e7e052 chore: sync VERSION to 0.1.151.2 [skip ci]
92ed1918fdd434854666c6fa27f9bc49c3e775e1 chore: archive restore-missing-frontend-translations
1b71b6e717c22a39e26a0910f85f50526017b097 fix: restore missing frontend translations
82c36346efc36ff4695dc210c9dcfe7102c1a91a chore: archive add-openai-first-token-timeouts
526940cc5d02ee7716ae99c6694699d42732780c docs: record timeout fix verification
bf5bef91a2a408493fc2c27a9c870de6bd2ccf1c fix: harden OpenAI streaming timeouts
2bf4f40b4b4ab2c317647ca964fe6f032e7a7a74 chore: sync VERSION to 0.1.151.1 [skip ci]
498023191292583d26c2551359cb3d10459e070c docs: record OpenAI timeout verification
05422dbe796afadcf808af22f1bd04ded1069943 chore: complete OpenAI timeout implementation plan
2a6afb1d95072db82c1c5da0bb449a48da543be3 feat: configure OpenAI first token timeouts
279ee4ced16217127e662bf3ee7d6ac6a5dae008 feat: enforce passthrough WebSocket first token timeout
b90325b333d5eea490ae7609e4e04d0e93c7cecb feat: enforce pooled WebSocket first token timeout
68ad2b828b16e117a1ee178940f60db06025bbf7 fix: harden OpenAI first token timeout races
2d98137545921ec887a26fa39f6455e270094475 feat: enforce OpenAI SSE first token timeout
4f7629575b656ef0b32c31faddc2a476dcb8c1a3 feat: classify OpenAI first token events
5f02e76f349fbad9524ab4299b35fb4f798460fa feat: add OpenAI first token timeout settings
ec6f6e25f20be8c16864a81cbfa7689a25b69871 chore: archive merge-upstream-v0-1-151
7cc4b378ba43b49a89dbf9fb909055b741a962f6 docs: record upstream merge verification
2a694408bf207fa8ef5cf4b2e0bd741c7fb9a4ab chore: record upstream merge completion
6803b6bf1c8bdecff979692a2e724b3440e9f6a0 fix: complete upstream v0.1.151 compatibility
9df2caf1122f3a9e1d471965b5398249a8b0491c docs: 设计 OpenAI 首 Token 超时
e4ef7f66ab520552ab8f8d1c4354147f1c81e920 test: harden sticky settings response contract
c98296a514aaa22738d25ac4af5f73625c94cfcc fix: return sticky scheduler settings
eb2aad4c5de9c74b88d7fd40d81f9f6a1ad792c8 chore: 完成请求体生命周期审查
7836f74f6b0140d28f503944ff99b101d9f0f551 fix: keep large-body failover nil-safe
07890b5887e66a6af35a123139120015f36f4ca1 chore: 完成关键能力专项审查
75d81b80b3ed48d26dc7eb7eebecf549b1f46071 fix: restore sticky and runtime settings semantics
6a278434b1ea3a59a6d38dab1e97f4bd38cd6118 docs: 记录上游元数据复核
6153e0ab6c87135af1fa4a5ef87304998ad4e311 chore: 完成上游冲突语义审查
b48d6d910ce42f9d825cb5fff75637182ed261db fix: preserve merged gateway runtime semantics
12eaafb95074a3c66b1bc3d697b918a5caff6e7d fix: restore Wire service providers
d7876ba57e88c796b57c4d825b2d8330ac42f470 fix: restore OpenAI passthrough attempt telemetry
80927aa8457e519c13aa19d7abbd79394934261c docs: record v0.1.151 merge validation
2e3e92457b435d91d3c3a93cc120cecc8aa81cd4 Merge tag 'v0.1.151' into feature/20260713/merge-upstream-v0-1-151
cd9166900b8817bf52c2a407737393d7e9f17786 chore: 完成上游合并基线检查
0aa43dc9bd348858eb16d29531a61c9f2261a944 docs: 记录 v0.1.151 合并基线
46d92f1d75f9835539f2a86d92849604a79d2f44 Merge branch 'feature/20260709/optimize-large-input-memory'
d3221f61f4d5d08ad7325336432fbb1bd8382a8b chore: sync VERSION to 0.1.146.4 [skip ci]
bfa02c04eb63f7001c1f6f6d2c5f99d779819a1c chore: archive extend-request-body-spooling
bedf81a19aff26c6faf4372a8d07ee492bc3417b chore: advance reverified request spooling change
b33ac4f782c4029248966bb308735930a0404051 docs: record full request spooling reverify
6a759f71e7c60cc0cdb5bf1326253d89c482809e chore: reopen request spooling verification
6b122d9009c47eb16923964622e305fcb4543f35 chore: advance request spooling change to archive
110ffeb87ad9718432d82ddcc84545af4b2840be chore: record request spooling branch handling
d9c65e2dbf6660b1c239b3c32d65e2db7b99c62a docs: add request spooling verification report
207265e4356de9b6eabdf432d4d5bc24d8d70203 chore: complete json media affinity repair
93f6adf9e5b09dc02c573ac399ed3edc84715a94 test: align media affinity descriptions
4491ed82b994259def1c847f86e4553b8bf333af fix: preserve json media session affinity
8ef402e63e4b772f2506219f5e208be030aec3d9 chore: add json media affinity repair task
e031c8857027a25bc328490ee512ad579ea71ef6 chore: return request spooling change to verify
8fab7eb589a4dd024f832eeb3428f5e9087b944e chore: complete multipart compatibility repair
e8743c5dc6e4e0069dea7f7a8cff4eeb5c40b2a2 fix: preserve multipart scheduler affinity
3c69a5eeb4855a2811c479a94e7b038aaef30e12 chore: record multipart sticky verification blocker
989705430112a13fdfe89794c9d96eb00aa8893e fix: preserve media moderation and sticky semantics
80b90b70cf2d837ead43577d5d805c355fd7170d fix: release multipart text values before forwarding
33e89342538eaf33b7eb5dccf41ef32f67c295ce fix: preserve multipart text part limits
0df2f2b802297a2027eba629e3a8d1f521aff7fb chore: add multipart compatibility repair task
f300c50c996656b5ba46d9d62b5112c225587c4d chore: advance request spooling change to verify
7d1461388101ad58944e8c6fa18afcc60c8745b2 chore: configure change build verification
532cfebb16700a2a5365eea8ceb7ef591d11fd9a chore: complete runtime acceptance task
2c18a7da59868f54464914d79dd65c6262f1ccde chore: record runtime acceptance review
3245bab779a87f1c212484e8a99c8b352f324bc9 chore: record runtime verification blocker
a3a2463128b7ab2049bc9af4b4867e087cc970b2 chore: complete automated verification task
3dd893d75b840fbf17510c3066fab7b25aa90d46 test: harden controlled body matrix
179ce52ea4bdd72c5903ecfe137f989670c03568 test: add controlled request body size matrix
ebf4b79def6c42c35cf699caeda84d8af407a130 chore: complete cross protocol spool contract
56247ad2f862a0a790b9f330f8c805323e5fa298 fix: complete responses spool contract
a08495821f7cc2020f9f18c6c5dda27be537461d fix: enforce cross protocol spool errors
914719c58d3f721c119ee49792427902ce5a66ae test: enforce request body spool contract
84ed2255d5314daf3205b639e8e690c1c9a8cc8e chore: complete media lifecycle regression task
9ed5842f4788860960127f7d454a88d65f7895c5 test: strengthen media lifecycle matrix
d224ee1f1f3a6b510015e498b2d02c3852efd44e test: cover grok video creation
be634951bca33d7ddec5ab955f0d81bc33cac6c2 test: preserve media request semantics
3665db1e455d2cd6fdd9882fa91a06338f525fac chore: complete media multipart pipe task
de3222b8d0c8c9aa7b11a5648db493a85539041c fix: classify media spool failures
03390b659c8e5e976943cf5cbfaf39f0c65fd69f fix: harden media multipart replay
1a0821cf3081b7918238c68e8b6be67f91611417 feat: spool media multipart request bodies
78bcce698b31906abc881ae344185445067add3c chore: complete media spooling regression task
9eb6fa322ad09eae3230c5d1f9c127ed1b006f46 fix: preserve media multipart inputs
b1f19b98a3aa4219665a74a761cab646c353bed0 chore: record media spooling review blocker
8ef0328cd7e71305a4140414ed2f60853725cb9a test: complete media spooling matrix
e652f57cac5c11f61d7849507fa4431c2a4a2f0b feat: spool grok media request bodies
ccebd4cdaafaaebc1c4eb9d9bae43001ba218d60 feat: spool openai image multipart bodies
2c0d4665f954dcd9223208ad976ede036d069d75 test: cover media body spooling
c85be440ec4afe36cf44b34223fe3b84916fbe4b chore: complete gemini semantic regression task
9cb0f402ca85428530c1215035e8585a3576a878 fix: release account slot when selection hydration fails
7dd86770cc37d7c7085fc79369fe251f6891dc82 fix: honor prefetched gemini sticky account
8843c3290e1f9bb304e19236e75cad94912449a8 test: preserve gemini spooling semantics
45ed8edf3eb3cc79f9a0101c09a3470fcfdbcff1 chore: complete gemini native spooling task
13d98033e72a6ffdabdb1118ce3f1ebed903c982 fix: release gemini account slot on spool errors
eb9568256c888fe832d942190d1963c0781ab6c0 fix: close gemini request lifecycle gaps
2952beb2836c5f8685310dca729b590f22b4f103 chore: record gemini matrix handoff
a6d40e6afe7ed5cb928cf4ca93b8eeddedb63cd5 fix: complete gemini request body replay
3a786c4daed491f5cb1237a806473104dd8c09c4 feat: spool gemini native request bodies
d0990d3844afe050a8d4d3495175cb65606f04be chore: complete json regression task
32255184e9268ece93a256983e3dd07b123e78d9 test: complete anthropic body replay matrix
62f4af23483cc3ec161dc8c77de311c01ff0ebed test: cover json request body spooling
933c7fca49d5efd13b46e29644dd3d92e1d4a4a5 chore: complete openai replay task
a83a4d10d1880d08217e252a923c68346d720eef test: cover raw chat bounded snapshots
d974b709bd79da40ad531c8a92aef59967e2cf2b test: complete openai request replay coverage
fe3a1ec25f23f63ca2ab4618282dcac3032d2f9e chore: record openai replay test gaps
e9bcb7499b9913013daf027e3b73ae1d852b9c85 fix: complete openai request body replay
b9544acdf35fb5401b70d7af667f84eedb8c7b08 chore: record openai lifecycle test blocker
11e2ccdc868f3240e89c99b0f34a6f3550ddb990 feat: replay openai request bodies from handles
2eda86b94f138d8a1e683c347e38f8f26a5f39fe chore: complete anthropic spooling task
efd5a7ef624a0d90123c8958128ea00b3ba12868 fix: close anthropic request body gaps
0bb4efbe38dd18678bb332abc69f5217e2c0a831 chore: record upstream handle review findings
814f339a27bd21794308b35f0b13bc372adefd9b fix: complete anthropic request body spooling
f7117bf98d8037c51f636e60918329a2a18c937e chore: record task 4 evidence gap
188f76d9a186da8d350a6882a598971ae17a2f3d chore: record anthropic verification blocker
098eb085b234edc383a70434e9e84e94ac0ff3b1 feat: spool antigravity retry payloads
ca4551836be719f1f00619d507c19736b2cf81d4 docs: extend antigravity request body design
b6391280fe23a7807f78c47d260a7813b45e956c chore: record antigravity retry blocker
a5f4b80394777a02bd3ca3aabf05220da6260d7f chore: record anthropic spooling blocker
bc06ce133b55debdecc5c73f1789a4470f63d0df fix: preserve anthropic request attempt state
1dc7422b83ba9f9973b60685f4a1a1acd7302108 fix: propagate anthropic request body handles
0786a6a080cc68da9309ed741cb648639178c13d feat: spool anthropic gateway request bodies
9405bddce475efb4bd70b867171deb1ed2bf9dc4 chore: complete request cleanup task
0c519e8e486a7eb23164bfa4cc33d65ff6a9fa13 test: prove request body cleanup ownership
24a425e9f03852005d926c57e413f3c0d37de52e test: cover request body ownership cleanup
a10400b75a22e21c12ea7ff173ac0a9b837b74e5 chore: complete request coordinator task
5b58a9704d85f0338730c1e936755d764ffda6be feat: add request body coordinator
cc371e3d0ff7e0ef684de058258424d993629bb8 chore: complete request decoder task
7d042007699f9546c437f8ef0255f3fce1901b6d fix: wrap request body decoder read errors
175f4f01c12ef06a812c67397eec24037e2c8f01 test: cover request body decoder boundaries
ae7806874317d98d4c97328aab928930ee823c36 refactor: share request body decoder
6dd19ebbd6c27dac0770d4c6c8542183cc92086c docs: add request body spooling plan
0f389fe7ed783ca4a8444fbe6d12acb9d3e19af6 docs: add request body spooling design
ad068f9ff2ade9cb9740353aebd35d57d58188a2 docs: add request body spooling extension change
cd652db85a8d6bc0a7e9619cc9984c0ec2350bb0 chore: archive request body retention change
e04a21ed007ad515b035da8ea1e20a279bdeac27 chore: archive large input memory change
14208b9e7a2ae6c98b48e4dfd9a2df6e1ecee43e chore: sync VERSION to 0.1.146.3 [skip ci]
262c576c38e0bc4b0a796ad96a9e06f65bfe2e97 fix: harden request body retention
56e7525536b8fa12a0f116ca57dfe834ea4d9207 docs: add request body retention change
ef981d498eb159c28dc912b0051336ee3da9b16d chore: set large input verify mode
cc9931b0086723d661b6e224cf345a5d0c5c46b9 chore: record large input verify state
76690f153c197ba8f7e2e2a1e9cfb51022e2bf31 chore: advance large input change to verify
9c811182cda18bd6c72c0f03bf0fe659ad451f4d chore: fix large input build command
94cdf8a82dad2dd9deefdfcbb9c4d95a23965164 chore: set large input build check
ff3fc219b0f09db490510f2b2fe2e62fb406c2fb chore: complete large input memory plan
f5b953c89fd41b1c7d146f471ddcf755425167f2 feat: optimize large input memory handling
40b807f114d1fe2e02ccc118fc4e6bd75417e4e5 chore: sync VERSION to 0.1.146.2 [skip ci]
d329e475e5e0c44588cbf8b9f529ab3d193174ed fix: drop deleted moderation group ids
47a90f44b955d6e08f014e66ed03b07c4686f28a chore: archive merge-upstream-v0-1-146
fd5bd3e4816edb387a00ac25893b7dbbd85f6549 chore: advance upstream merge to archive
b7aba5c2cf3ddbb836d0a2b6da54209b146d5d60 docs: record upstream merge verification
e872315bb08d5d81670c0536be986fcc658c5893 chore: advance upstream merge to verify
db2cab2a0d01fd424c7c1d204d88a577c0a77721 chore: make comet merge guard reproducible
4dcf9f04492d70aec56b39e890c6544475eadec3 docs: close upstream merge plan
084dd760f79d2118ed3e7c9b05d449bd57a5184a docs: close upstream merge comet task
a9ac14a9ba566858644da64f137982490a0d681d Merge branch 'feature/20260707/merge-upstream-v0-1-146'
2e3401ecee36eda2f2fa40c10944189e20daf643 docs: record full release workflow
21125a62f5de45616f8ec6d9af6d372ce203c3d1 chore: sync VERSION to 0.1.146.1 [skip ci]
84b22feaa9112db2d5e4189029c169aa055320a9 fix(openai): align responses and layered sticky scheduling
6ee3aea94808dc7feb263d5e7c9eede16ec33fe1 merge: upstream v0.1.146
e378b33f60f2202d80cbf9b1c11cee4e4ddb9dc3 docs: archive hidden user ui controls change
ddf9a211008419539e681a3425b0bb3d306fe729 merge: hide user purchase and custom menus
f7fe1a5fa497a2a7fb2806b8e3ca59819d52b59a docs: verify hidden user ui controls
5043287c42dd5c0693f7511d605798e6d645266c docs: mark hidden ui menu plan complete
42b85436756027545e2adb5f71b473433f606064 chore: sync VERSION to 0.1.143.4 [skip ci]
c9e93f50ed2acacf72fe2624c91ca6b5e429f20c fix: hydrate hidden custom menu ids on auth user loads
3cd5f17a447a8a0a2682d69c79dbba7d6c82ec6e chore: sync VERSION to 0.1.143.3 [skip ci]
ff4243b34ad104a8e55f58fea849635534284f35 feat: hide user purchase and custom menus
f5fa926773d7ae9df7825dcc3d9e5fb77a99c96a docs: plan hidden user ui controls
b41f4fc166998198dfea8a704c01983fd5bb0884 chore: archive blocked public groups change
0287464a7b2682b55534566d96275a0680a45bf6 merge: integrate blocked public groups
70083f67e263f8950882192690e80ccf778bf432 chore: sync VERSION to 0.1.143.2 [skip ci]
ad56048612c5325f51e0a8767b23a2b1a7da9eba fix: reuse outer user transactions
14c74600b8098c15da6efe8bab8e021bcc6042b8 fix: keep blocked groups admin scoped
0b3fa21ad57e80fb8475764164cfa60abd2a9f58 fix: close admin blocked group gaps
82a23b32b399965323d12951971c8a92d11d3518 chore: clean blocked groups handoff context
7ab42dbc1bdbaff605eb99195e64ffde0ae77387 chore: configure blocked groups verification
95ace2b2a255baf52683b1ee82012b498b628096 chore: complete blocked groups implementation plan
a0be367e959fdf3f29e4cf485f22df4b9059bd25 fix: close blocked group auth gaps
5ebc781e9a7007249fb060256dfe9fb001c81094 chore: mark blocked public groups change complete
7dc4bb31bf7f88073f8b9310818eebd617677f24 feat: configure blocked public groups in admin UI
65b6d0ce7565ed59c0580b4cf674b0a72c91f861 feat: enforce user blocked public groups
a41ceec4d3cf2bcca8f4f90857d70d85dc5a0d9e feat: store user blocked public groups
dd070eeaff578c2d91468f4de3246c3a7f2a6d3e chore: sync VERSION to 0.1.143.1 [skip ci]
3e322c60dc6395fd9610102a8c759cfd2227fe34 merge: finalize upstream v0.1.143 integration
02788540680809181aba336b7245a896c8985bc8 fix(account): correct openai auto-pause i18n keys
3c877683dafbd4c5b3eb42dd2106eadc3939f5b2 merge: sync upstream v0.1.139
2006f648e0e7c1aac8e3f13210e50c2d5abddfb3 chore: sync VERSION to 0.1.139.2 [skip ci]
e37f3a3db95c16b1e78c3bb98f1751bf75a71107 chore: sync VERSION to 0.1.139.1 [skip ci]
40b34f198516edcdcc0b53095c7bf45175719c73 test(risk-control): cover codex developer context filtering
261f73700dcc8ea351962fed8dd7bea393ae2b83 chore: sync VERSION to 0.1.137.6 [skip ci]
f640b046012e4595783befc5870b0581fb15b3b0 fix(subscription): anchor quota windows to subscription start
588766be3a90d9371247926bf43779a3db6b025b chore: sync VERSION to 0.1.137.5 [skip ci]
21424f6832c4e5cf3deca938f682d24bfe2de564 fix(risk-control): store full moderation audit input
f67356b7568f45292c2e0a62e8babf7250abd263 chore: sync VERSION to 0.1.137.4 [skip ci]
7ac1456d76ef91a87d6567fded3a9fe5719a6ad0 fix(risk-control): keep latest moderation input in audit logs
2aad4ebe9204226242dd22e07e43e6f194af1583 fix(risk-control): review re-审 fixes — 测试重命名 + 补 roleless typed 正向测试
961852d476bc797103c4252152c628ec9fa208ff fix(risk-control): review fixes — 空 role 跳过、过时注释、测试重命名、去重 key 组合
2b940d4abfc5e4bc62836a0875929911bc345865 refactor(risk-control): 清理旧抽取函数，统一为 collectAll* 系列
411a76ec8da739445c195c559c7c20325e245841 feat(risk-control): Gemini 审计抽取 user+model 非工具输出
986cec2591edbc08b54dbf89c298b392bbf5948e feat(risk-control): Responses 审计抽取 user+assistant 非工具输出
d913becea83b0dfe19f77b7236f6a31423ab7a42 feat(risk-control): Anthropic 审计抽取 user+assistant 非工具输出
b0349e72aa13c2085df218912f4dc2e105d1d336 feat(risk-control): OpenAI Chat 审计抽取 user+assistant 非工具输出
2b98f28ee58ecb78e79e8847d4badf0da957935b chore: sync VERSION to 0.1.137.3 [skip ci]
25675fec87301b10593cb7fd8de5544c9a40d669 fix(risk-control): audit recent unique user inputs
44773e59666b56143326e7c70e83a5ce3d7c98be merge branch 'merge-upstream-v0.1.137-20260617'
8fe64c66ed33afc0c65fdd8ea1f7d4d707fd332b chore: sync VERSION to 0.1.137.2 [skip ci]
04195ad8b6dd5eb0759547b3994f396926497d28 fix(settings): restore upstream payload fields
92972aa6f66febfba8b33a38d79e0caa087a6bf7 chore: sync VERSION to 0.1.137.1 [skip ci]
9f23f45a4d2573cf503fb4c8b74e1682fe37ff78 merge upstream v0.1.137
615c2f61f333d25f2e2284969ea382644189bc2c chore: sync VERSION to 0.1.134.2 [skip ci]
6b13d269fc1b93277f150e92fd17fb3d96b0ac36 feat(risk-control): show matched keyword in audit detail
d3d8aaa4022766f9c0ec362165fc5b28f2253a80 更新文档：完善 CLAUDE.md 中的项目描述和术语定义，添加知识库导航
bf557a3272488fa65c82804a15ff1f96630f75b6 chore: sync VERSION to 0.1.134.1 [skip ci]
5c9cb1161024c3b45e37d4974a20c41503e5a362 merge upstream v0.1.134
694771f8d62909d79a09fa750f7fb5f299f08c90 chore: sync VERSION to 0.1.133.3 [skip ci]
e62cc2c7253420a9f0c97f1af9dda852579c175b chore: add .understand-anything to .gitignore
9c1417c9009493dde160c408dc12c5dbc91fb40b Merge branch 'main' of https://github.com/caiqy/sub2api
840d8c8637c2e3c054dc39d36030853e40a8a859 test: cover anthropic unknown content fallback
3351bb2449d77698d77e73de0a1f7dcf6b69d5ba feat: reconstruct anthropic message streams
4142270b834802a474513b7c299ec8f3924ec750 fix: avoid anthropic detection false positive
761a0c2301e2ba8f832f7be5f3225ee775b46e75 feat: parse anthropic messages requests
52633cdd397351f394642498e6564897387dc1ca docs: design anthropic conversation rendering
ef344c7516c964e1fa7117389429cdaa151f4956 chore: sync VERSION to 0.1.133.2 [skip ci]
20c913a1e28685eb492b94c504067eead877f3e8 fix(review): idiomatic bool assertion, bulk error logging, probe model delete semantics
9f23e74b3a0d00533fd4c176cfa492bc385795b4 fix(review): bulk probe toggle side effects, naming clarity, frontend ref reset, BulkEdit key semantics
5fe3bf82be82adfed7914abdebcde6d8a8b80352 feat(account-modal): probe enabled and probe model in BulkEditAccountModal
b442435a8db55712b9b07433f81f4dc19ae7617d feat(account-modal): probe enabled and probe model in EditAccountModal
d5a8fb48c3fd9be041cb366cbd8a7b7a8f23f338 feat(account-modal): probe enabled and probe model in CreateAccountModal
60e4668a2e0e9ba048ac16c663cd34df3d94627e i18n: add probe enabled and probe model account fields
ad1228f206fe42be25a6569c84bd3c7b694363fd feat(admin): probe toggle off clears layered_probe temp state and drops probe entry
3e7906d381e25f1902fdbbcc9802acaff56d5949 feat(gateway): add DropProbeEntry idempotent entry remover
c15dfde2b8b661e48d05a292a9668e22bd4208c9 fix(probe): remove immediate probe after manual recovery
b82d00faa8c4bbd1b6aefee5aa9e148255fc3299 feat(probe): defensive tick guard removes orphan entries for disabled accounts
1bb107b2b3bc24c7bceef93ae637863a631b58ee feat(probe): skip runtime reattach for probe-disabled accounts
34aed24fdecf1b766b2a856f146edc64b7ad905a feat(probe): skip bootstrap and startup rehydrate for probe-disabled accounts
f3d8ecc8927b4a24323f2e31f5e59c54b24929ae feat(scheduler): skip probe registration for accounts with probe disabled
cf0e31a073438c0263710e38a98e17c15a5a74ab feat(probe): explicit openai_probe_model overrides default selection
72f552e3cca89a56b5e2d75634323ffe53bb0158 feat(account): add IsOpenAIProbeEnabled and GetOpenAIProbeModel getters
89c4f0ee739c4072034899f4cc14b2e358f495a7 docs: add openai probe account toggle implementation plan
eaf1a013d9e6a4dd2044c73ab999723929f08b38 docs: add openai probe account toggle and manual recovery design spec
d248cbba94fb0571e188935b8fe278c43c932096 style(service): fix gofmt formatting
a86576dd9cc85dc36f3473e10427b220621cc988 merge: sync upstream v0.1.133
4783c23909f94068a00e5caeca12293373a4f5d9 chore: sync VERSION to 0.1.131.4
b1c45a942b22c1ae2fed05e1a8ea4a4bbd80f8b4 fix(openai-probe): replace 100y temp-unschedulable with bounded cooldown
2ae98db37c89e749078419b44c20602e03f3888b chore: sync VERSION to 0.1.131.3 [skip ci]
16702b5c03fa77c4bd28b65dfa5091d175abae61 feat: skill tool title + SSE response.completed merge fix
86ed2e38c7f57b59d313933917e9edfeb763562c chore: sync VERSION to 0.1.131.2 [skip ci]
09a4ab52b2379d3e7f0dd79abf31c1adee737481 feat: conversation timeline layout & parser improvements
0e0406968deee4218501bc8899eac9f420a0a894 merge: sync upstream v0.1.131
512fa3ee1add54f6eed18bbc97a0be206bfb30ba chore: sync VERSION to 0.1.131.1 [skip ci]
7c473f3d64e9ae060f985745b584aa6864f10609 merge: sync upstream v0.1.131
332457a83b2049356a723570573c773866e7d37e Merge branch 'main' of https://github.com/caiqy/sub2api
443699f7f5164b536b89e1c5fc79039b722aa904 chore: sync VERSION to 0.1.129.3 [skip ci]
2753bc1e214045f75d248cf5031f789215d0da23 feat: improve conversation timeline rendering
65bfb370209362b8c9f69c2c8c7851b251e1af58 chore: sync VERSION to 0.1.129.2 [skip ci]
3fecb260522ef2ae1331ffc0bdb5c3c16aaf34d5 feat: render conversation timeline message parts
976df75d86ff1a1834831e9d676abb09e06c6170 test: add reasoning i18n key to ConversationTimeline test mock
ebc2a46f4f532a020190e464c827eeffa093da39 refactor: introduce dedicated reasoning node type for Responses reasoning
5f6511c49d6585e500499995bb098f81eff4f659 fix: render Responses reasoning summary as text instead of raw
5290a9a480bad28fe407e6274845ff5b496fb5b0 chore: sync VERSION to 0.1.130.1 [skip ci]
9115187f5f647aea726124dc91a35e6565ce4a5e fix: restore frontend test suite stability
190175062d6eba05123912d1d24c544e9666fe7f feat: add usage conversation flow timeline
f464ea7b2d5851e41878ab9f8f3e6b125b834138 Merge branch 'chore/merge-upstream-main-2026-05-21'
1d8949f1365d062365659c892fa039d064552fc6 chore: add .opencode to .gitignore
6b6a4240142e1279a9274983ef72118087476920 chore: sync VERSION to 0.1.129.1 [skip ci]
7ae825640341db222963170ed36efdc8218529df merge: sync upstream main and preserve layered scheduler compatibility
7a711ffcd9a791e9dc59312de0d726d7a048fc49 Merge branch 'fix/usage-chart-username-display'
385b356e2a6a18307328fa72d9c47368f76b92a8 chore: sync VERSION to 0.1.127.3 [skip ci]
e243d65a727cc0c0bc1901908f5f6dd5627baea2 fix: update permissions for agent-browser-cli and chrome-devtools
0344e98c29a6094573eee1f171b145c72739e5f8 fix: prefer usernames in usage charts
cc79db76a1253bed55b332eb210b1b47644c2fbf Merge remote-tracking branch 'origin/main'
f0100cf8327afd9c27e7a09bcdc056d4b52e7539 fix: satisfy backend lint checks after upstream sync
292e99d219cca4c2119a03a73101d0e3106c99d0 chore: sync VERSION to 0.1.127.2 [skip ci]
c10a0f9e8db708774c2c03010db81b69c801a10d merge: sync upstream main and preserve layered gateway settings
6c92fa51da8c9f7d7ddf1214b9af585103b65be9 chore: sync VERSION to 0.1.127.1 [skip ci]
626a7c4805884a485f413de23a4cbfcc11d3e0fa merge: sync upstream main and preserve layered scheduler
d3684513ee2abc2c954c45e33720fcefd9f21685 merge: integrate upstream sync branch
9692d90893f5671e732d52aa7927bcfe1f879a66 fix: update gpt-image-2 permission to allow
5f041b26a9384d8e9c13c0e29367e472edbb129b docs: clarify release versioning and workflow tracking
fa316dcb34f66a93b1c906b8ba80593be3eefd12 chore: sync VERSION to 0.1.127 [skip ci]
b9e555d75905c05c37e0a2861da12acd0235a45f fix: preserve OpenAI channel mapping cache
af76856a0d1c8b38798b6eb8106247568cf92acd docs: add sub2api server update workflows
1990dbe33bee1d1a700799d974830639aae6afca chore: sync VERSION to 0.1.126 [skip ci]
2a8d23a372d0253b1a3ce74d152a45005e35d09b fix: update Go toolchain for security scan
9e86a994ea76a57932609e692a0b99f12d87e74e fix: tighten OpenAI gateway scheduling regressions
9f923b1226f38b5bd17a9860f26d82e583864778 merge: sync upstream v0.1.125 and preserve local gateway features
4a714a7e1d09b1a90ffa481a762cea3c022a7488 chore: sync VERSION to 0.1.121.1 [skip ci]
c6260eb45f81346394b93a69dd1a62a5259c0145 test: protect fast policy and layered settings round-trip
5ce5de4717b1f7b02cf5eeed24a5ba34991f1d4f merge: sync upstream v0.1.121 and preserve layered gateway features
3aeba29d676a5402f02ece9fdaa9545be9a462d1 Merge branch 'chore/merge-upstream-2026-04-26'
08fefba0c8e359db725c56f7616e252056941dfa chore: sync VERSION to 0.1.119.4 [skip ci]
d3b4989d3e37353abc1b0fde57fc1b68fb71a6cd fix: restore gateway runtime controls after merge
26a47000781fc435d54f364c3fcce1db19c63dd6 chore: sync VERSION to 0.1.119.3 [skip ci]
d8f3914d1eeafd382154d56479620feb7571a5a4 fix: respect fixed quota reset windows in account views
a9d09d5b218ba8d67478baa5eee8de66d226b01a chore: sync VERSION to 0.1.119.2 [skip ci]
492af62326921c4f8b9f0206fa970820f32c1564 fix: align admin usage image preview with ai image gallery
b923fdf91146e5affc55743fec6271cbb0cdfb2c docs: capture upstream merge workflow in memory
14b103f2645823fab40f594450fe39d9667b6adc chore: sync VERSION to 0.1.119.1 [skip ci]
1b8d6d2a9a5dae918c6056b7b73ddf1fb3793dce fix: preserve openai scheduler safeguards after merge
8c7d464720ad79330721def03606eb1163a525b0 merge: sync upstream v0.1.119 and preserve local features
8b13a27fe224d552536d897d01fe41d440fe4638 Merge branch 'main' of https://github.com/caiqy/sub2api
bafb59fcb0093555bd3daa95ba1028b0a4323c8d fix: update release workflow to clarify post-release checks
f5b0c60f9796a4367d92005457579364fd6eac7b chore: sync VERSION to 0.1.117.110 [skip ci]
0c8d1e7d5a46a2a2a5e1e6606b2db962cb73115d fix: polish ai image workbench
bc6a36af77a053e3744e798e9ec8e1b1bf886272 chore: bootstrap project memory-management skeleton
4cbf95c5913b372c6fcc6d0c984bb0b86926c659 chore: sync VERSION to 0.1.117.109 [skip ci]
da05a554080db53ec66679692673ae80d2e88244 fix: extend ai image timeout to 30 minutes
5f7757914770cbbac1de8303ab97915f5b220a0d chore: sync VERSION to 0.1.117.108 [skip ci]
7eb2f8ac752058aeb0fd5cffb4e87e7de0f4e4ab fix: extend ai image gateway timeout
e7677344aa00f1832db9eafbb7da7f158c604560 chore: sync VERSION to 0.1.117.107 [skip ci]
7c696cddc72b75950f92fa12a8d473b0195d9c3f fix: tighten ai image workbench layout
149461535be6305c892afff2b130609d37884aa3 chore: sync VERSION to 0.1.117.106 [skip ci]
1ddebe0f91a17b28f28c8427a58151ae4d7f9d86 fix: improve ai image workbench preview
e1ef9a6ef296199dd705493a411b7b13845ecec7 chore: sync VERSION to 0.1.117.105 [skip ci]
f7bcf5c081096fea0e00fba94fb38a4b79e1d111 fix: improve ai image layout and params
2cb6325c3bb7456db73c0b53802dbf5d2b48e8b0 chore: sync VERSION to 0.1.117.104 [skip ci]
4794cb582c61a8a1b4b3b89b34b42d070fe41b9d fix: split usage detail retention by image calls
86efc858e212d46728f1e79371e89e4999bdf3ba chore: sync VERSION to 0.1.117.103 [skip ci]
ea978ef1d31463c7ac1a7a63dbd4336fd5050b76 fix: preserve admin openai oauth config on reauth
d0ffe6421157e289a35029a559d4cc17ffb27a73 chore: sync VERSION to 0.1.117.102 [skip ci]
cca561d681bdd6ec89bffb0bed37044953bd66d8 fix: preserve openai oauth config on reauth
05b75080e18828c19dfbcce41960ccd6d3f4db86 Merge remote-tracking branch 'origin/main'
5b9d87877bc39e8242429187f8cb11ffb7bc0cad fix: preserve images upstream request snapshots
1204ed2fbd82b17f910cde7695e37a10729b916f chore: sync VERSION to 0.1.117.101 [skip ci]
5277dee1484c3551e047d352be634c7397d5d93a merge: sync upstream main and preserve local image gateway coverage
4edd12a82559acb6750777eb42a47f76890ee21b Merge branch 'feature/ai-images'
274225bf4c8a6125be56b1a34aaf77e3fa0ddd8d Merge remote-tracking branch 'origin/main'
8502a54b7dc2b9a20b214586067f0ce04b2e011a fix: align messages failed-usage test with compat errors
60585e5d0e300588caa3a1d6b156a5da0966fecd chore: sync VERSION to 0.1.115.106 [skip ci]
4ccdc46c374dd20a0aaff8c8810c062437e2770f feat: improve image debug visibility
dc5ce672aa47b0cc7efddb93edc2955542386ad0 chore: sync VERSION to 0.1.115.105 [skip ci]
80959147d1211df5ddbc5f9eb663466fcc004a08 feat: polish AI image generation workflow
a3eaffb89e289cf9c1054148371c5242f2f31e55 chore: sync VERSION to 0.1.115.104 [skip ci]
9cf976896119e0b3d063a92731c200e4f789c5a3 feat: improve AI image sizing and request feedback
30d321428834787532e1e5d5a804dbe01ab38517 chore: sync VERSION to 0.1.115.103 [skip ci]
e6419fdc70a59b304a16f5aa771c294a406d7d8d feat: support custom AI image sizes
a6a0f80f64561053b683d2a029bdedfabe440783 chore: sync VERSION to 0.1.115.102 [skip ci]
d6e430fd0b27ffbd2136bccde16fe73ec5f55134 chore: sync VERSION to 0.1.115.101 [skip ci]
5bc1226a8237a9842834657bf24f6425a81a3d73 feat: add user AI image workspace
b642205525fd4c99bc144fe357d91f07098f190d feat: update permissions to deny additional skills and file types
c73ff549b471bd04d8c1ca52f278911afaab802d chore: sync VERSION to 0.1.115 [skip ci]
db1463e8a82e6f60d3b12eba61f9f53c5213ec9b chore: sync VERSION to 0.1.116 [skip ci]
4f1b89428ceb399abd3c7c5ef1705dd1b52e7fb9 chore: merge origin main version sync
a0b6eb62e23d84c41d77b57bff8bfe5dd67f6a30 chore: merge upstream main while preserving local gateway controls
bb8dd06a640e377a5f5a6f811bb2662f70a40ac6 chore: sync VERSION to 0.1.114.102 [skip ci]
8a51ae5b6809fb8502d0cd1461ddae34855f8e98 fix: close group user concurrency runtime gaps
1aa4aa940aa0715e43af47e76fb3585af72dcca2 fix: persist user concurrency fields in group Create/Update repo
6711d06ee7f9d2866b57f9f1da27c62e68ab82c4 chore: sync VERSION to 0.1.114.101 [skip ci]
a3d0eb51a6e590732693debdc1b32d45adc6c690 feat: add per-group user concurrency limit
2ef284255dec9d179f2d7db048a8aa7476cd79cb Merge branch 'main' of https://github.com/caiqy/sub2api
0c46c2737f82eb5966bd87434a72a5864e845fae chore: sync VERSION to 0.1.114 [skip ci]
9c52355d932705f203a86b298c2c1c334a93737e chore: merge upstream main while preserving local gateway controls
34b99dd856559029232d801772df478317070a21 chore: sync VERSION to 0.1.105.15 [skip ci]
be7e562f77d60ce826f7cb5e91e603ce16b656f0 fix: skip quota-exceeded OpenAI account selection
27c7fc20910c33f20c5aebfe2730981c97c3c9d4 chore: sync VERSION to 0.1.105.14 [skip ci]
71aac9109c4a5701e36ebddae6d12386b5d44db5 feat: redesign OpenAI layered probe startup rehydrate with source-scoped recovery and DB truth consistency
32a17fa4f37a19355fe8295ac832ad1edc0de3c9 chore: sync VERSION to 0.1.105.13 [skip ci]
ef2b147df8adb562a67d67c903857331cc32b859 fix: add missing tools section for chrome-devtools screenshot configuration
15f89bf5f99edf7da63231006f0dad5c3e2eba84 fix: align openai probe recovery with responses endpoints
6ce1b99e330e6e5a1c672337a875d10f67afa58b chore: sync VERSION to 0.1.105.12 [skip ci]
8612e00c9d2f8836db089bf8dd2920f3f49e2c2d feat: add platform-specific sticky toggles
8d69215843c41b66ba2da81710c579f3240bb83c chore: sync VERSION to 0.1.105.11 [skip ci]
3a0bf1abfb6a7c8f2b34201bf8db6a134cf12d70 fix: restore logger shutdown semantics and unblock backend tests
f56e7812599b487c54db26a49690d0bac30a3c16 fix: serialize probe result processing and keep penalty state accurate
904b0033e94af1a0663955e895c6f8422263f9db fix: add explainability metrics to probe logs and ignore stale probe results after manual recovery
4248467814f52b7b357a717c3c30fffb8c87542b fix: add probe success logging and clarify manual recovery log context
6d483000f1c572da63f0fd6ff5b33fb9819d31bf feat: add manual full recovery semantics and reason-aware probe logging
3fab594e73e7d76fccdcb17326eab9aa14cd9e7f fix: persist penalty group context for probe reevaluation
846c9cfd052c6018b2c6a2dd163b2899b1aa799d fix: preserve account group context when probe reevaluates ttft penalty
96e7182f79d9595de545bc993b2f471a13ec2e90 feat: reevaluate error and ttft reasons independently on probe success
770c6c9fbf5614cbcc93a050402fb5dfb0035213 fix: keep probe entry when temp unschedulable clear fails
b406ab2d828857972d8a992e81e2e1eed90f6209 fix: serialize probe shutdown with worker dispatch
d5c67f038a60211953509eb46e2a7fd8e45dbe3d fix: cancel and wait in-flight probe work during shutdown
e21fee34261ce74292edcd280dc76e0e0898e65b fix: make probe reason clearing safe for in-flight probe and db state
ea3b7a8d5612ddd1df01ae8a7e75333557227c1b feat: track probe penalty reasons separately for error and ttft
07e75943689f6ec60f0a91c8ddda1e4c96d7430c test: rename shared TTFT baseline test to reflect helper scope
8abfd01c5648699540c5c4207e7bfbbc55c3a9a4 test: cover group-level TTFT baseline with excluded fast account
808bd1bc54b4384b8d2a59a8a15e0524e60ff919 fix: stabilize TTFT baseline tests and avoid repeated group scans
6189701a2dea2537a8e3c65511ad606ac5396890 refactor: share group-level TTFT penalty evaluation between scheduler and probe
b73310cea30fefb429b0380a271890ebcff3bcf2 Merge branch 'main' of https://github.com/caiqy/sub2api
f8d8b8310349f6bc76b9911385a2f55872439019 fix: resetAccount preserves TTFT EWMA, probe feeds real TTFT samples for recovery
0aa3ccb1dbb3e02991169eb474080dfe16831ce9 fix: probe measures TTFT and reports to EWMA instead of blind reset on success
bf20a1b50fc19f97d97da5bae7f1eb2d67adbc01 fix: register TTFT-penalized accounts for probe recovery
1f77f55a203f75a771be05b22257ab40b4dc264b chore: sync VERSION to 0.1.105.10 [skip ci]
f79e9b47d2bd66ba5549f64dfe1fc933ea4168f6 chore: sync VERSION to 0.1.105.9
9b30101c163535ba87df380c4667f25c34b3d4b8 chore: sync VERSION to 0.1.105.8 [skip ci]
1193355b490b87b8353ae8cd4e59207c055591b5 fix(update): handle four-segment versions in update checks
fe761c71564ca44a338062cf9e151c10a93c9f3b chore: sync VERSION to 0.1.105.7 [skip ci]
bda5806ee8b066f013b9676ab2aac41f62eaa12f fix(playwright): disable playwright feature
77a13fe780d0f29291ec491aac777956d44728fa chore: sync VERSION to 0.1.105.6 [skip ci]
6a04b5c72595ed406ec2e0f8469c8036a46d8b6a Merge branch 'main' of https://github.com/caiqy/sub2api
e538d1347d7252ef0cf670de5b8928d6cef93973 feat(permission): add skill restrictions for writing and git worktrees
4e8d2ac789bef992fc09d79f620f01fdce079e94 fix(quota): unify billing, display, and scheduling semantics across quota subsystems
1e209692837ae38f2a0bef3d4be852f8ebe0e61d fix: extract constants, add scheduler_mode typo warning, add TTFT penalty test
19897766c5ddc04e665b8e1362c30386274870e1 fix: address review issues — probe goroutine lifecycle, test cleanup, URL normalization
8cec44727c419be642d46f7906c21a22d2528ff7 test: add layered scheduler and health probe tests
cf7c74dfbadd0fdb880c3b1074037dc004235d37 feat: implement layered OpenAI scheduler with health probe and factory switch
2660711074b2f00d86cd4434cf3604197747d453 feat: add EWMA reset method for account recovery
a25896f5365b7d69de6e29b353586bcc5cdb920c docs: add layered scheduler config example
f9cf1cacd1a4b8e87824d90b65d42aad9f4e2a99 feat(config): add scheduler_mode and scheduler_layered config
9332fca447c7a2fb62c4091ae25a0fb067572485 chore: sync VERSION to 0.1.105.5 [skip ci]
de06808fccb8f1fd456659090d24eead8d79b32c fix(passthrough): preserve delete mode when editing rules
ef786b9a5df4047b961b527d5488b8b72bcd1cd8 chore: sync VERSION to 0.1.105.4 [skip ci]
8833a31f8085d9ada88fddc973755c45116ef358 fix(passthrough): align delete conflict validation with backend
f95966d21e298107b361ce062ee7967496f87e0e feat(passthrough): add delete mode to frontend UI, validation, i18n, and tests
a17c1b45a516be678ea1100ef137b4451c38edf6 feat(passthrough): implement delete mode runtime execution
303137506457f0e348a87e548503ffa71d290ff9 feat(passthrough): add delete mode to normalization and validation
61e552e01320863f710c54d06109979d172007cc chore: sync VERSION to 0.1.105.3 [skip ci]
cc16317fb73024485512debf2248227181e2c020 ci: disable auto-trigger for backend-ci workflow
882a19d412706698d1273659ec25bae596487fc5 fix(merge): add custom_base_url save logic to applyQuotaControlExtra
e5c55c15db16803fa13c706579a92097659781af Merge remote-tracking branch 'upstream/main'
cc4728e58fc60b0be5a3725fb3b9134e7c514af1 fix(update): change GitHub repo to caiqy/sub2api for release checks
29958024a73236f8d5911d2407ad16e420bf752d fix(i18n): update passthrough field description for all account types
38a1b4bff8ab6cb08667608a2c55a2fa7207c073 chore: sync VERSION to 0.1.105.2 [skip ci]
0d88c255b9460715fa4dc3724ed853d9798fe6cf feat(passthrough): support passthrough fields for all account types
e0d997294e25ae7c86b78f267808d1248aadaeaa Merge branch 'main' of https://github.com/caiqy/sub2api
20884b43f940afc048c8a91017a7ac0c31cda9d1 fix(gateway): use original client body as sourceBody for passthrough fields in ForwardAs* paths
bf916d34d32d564dd523aae5a9cdfcb3cac208ac chore: sync VERSION to 0.1.105.1 [skip ci]
9095a21c6e7df32d6b8d0fe0f5bbaf48e0980784 fix(settings): backfill defaults for legacy JSON configs
9e9807f579e36756b50fe05662899dedff425e9b merge: sync upstream/main and preserve passthrough behavior
c5643e6a51c93d4d8dd411b7737651ab8f06e2d9 chore: sync VERSION to 0.1.104.12 [skip ci]
83e2e095affc282559609761b408273bc2e55cb6 feat: add upstream response headers/body to usage log details
394a6f62fb5b256fb739959dfe6eadfd08a24aad chore: sync VERSION to 0.1.104.11 [skip ci]
e98fdaab9948544211403ba49c83db9f4f5f7557 feat: add response status code to usage detail and fix modal scroll background
b5ae00f520046499ff8858d4186f24dcb66f9945 chore: sync VERSION to 0.1.104.10 [skip ci]
80fb58ad82f8dc061c6c9ebe881285b33a132dcc feat: display username in admin usage records page
f992eeb32e044b724a1708ae15390cd124310d7c chore: sync VERSION to 0.1.104.9 [skip ci]
23ce43a1a21ab6443ab236988d24d4f4d632132a fix: fallback estimation for output_tokens when upstream reports 0
7166f85eb98f98b56ae5fb5105b500573296ab80 chore: sync VERSION to 0.1.104.8 [skip ci]
d90d7d8986a85984a38ebf46d05399a83c1cf645 Merge branch 'feat/gateway-runtime-settings'
d862428b3724469b4af98b17c04dba2ec3872edf feat: add gateway runtime settings management
b0905b313c26411aa506d0133b2348bad0d5847a chore: sync VERSION to 0.1.104.7 [skip ci]
707199d26bb9abf397edf10ada87fbaab86adef8 fix: record failed usage duration and reasoning effort
371275e170c511fe71ca69f1c08d7702814dfeee fix: store final upstream request body for usage detail and ops
3431998e0cb9898810bf8bbe33978bbeba06a3d7 fix: capture ops upstream request body in OpenAI compat gateway paths
b73975d990d96c235eb5fae8c111c7282450411d fix: capture final upstream request body in Anthropic gateway paths
0b28906b3cabc279669ef8b8e746ede262fedf90 fix: add body path prefix-conflict validation to frontend passthrough rules
05edb54b58deec550319ddf93bd1964998cdcd64 fix: bulk update explicitly rejects passthrough config keys
4a01ab355ed3487ac7459eab28ef61eb7109b3ce fix: disabled passthrough short-circuits before rule parsing
08fa87a800ab3120f03708936a89bc2fc65db0f5 chore: sync VERSION to 0.1.104.6 [skip ci]
75954419ac73cb0acc7a25472d7e205e63a65bfe Merge branch 'feature/account-passthrough-fields-v2'
42a4f3f9e7435c6bb5bbc515ba2a5d684699e57f feat: support account passthrough fields v2 for apikey accounts
09ce1fea07b2db73e8a3e42a7a1573e4585fddca fix: add request method and url to usage detail headers
8fb0cf43a5d8b57c9df929f14bc3e452902d2a16 chore: sync VERSION to 0.1.104.5 [skip ci]
b86c55d394778586b0f1e4007df3df998c668b2f fix: preserve failed usage logs and upstream snapshots
61bced175aa95b25817d78cd9ceb3c36a5b47a0b chore: sync VERSION to 0.1.104.4 [skip ci]
2629ca28336ddad01a1176d8d853d8e67a9cd9d4 fix: stabilize admin usage detail modal layout
9d2e658dbe4a7ec73f1629d459411ccc4411c650 Merge branch 'main' of https://github.com/caiqy/sub2api
6635376f9948216cf581af34f15465bbf57ad7c8 chore: add schema definition to opencode.json
d670f4b9dcdbb66ba7223f7e5ca26702f5d0d75c chore: sync VERSION to 0.1.104.3 [skip ci]
cd56579e3dc438cc3fa770e4d6360f2de6855994 chore: enable playwright mcp in opencode
bb69ba03c60c8bd14319c6f3888c8d41105f762c fix: persist best-effort usage details
4e2745a67dae857ca61291ad23a4516eceffe1bb Merge branch 'usage-detail-client-upstream'
c5e655660a900d095bfa5c2308d589a56d3dc9af feat: capture upstream requests in admin usage detail
c4a315c528d1e85534ac2fac47d6aa0e6d90cdfe chore: sync VERSION to 0.1.104.2 [skip ci]
7a6222438f8ee4918771331c7e30433681ee3a99 Merge remote-tracking branch 'origin/main'
8a9c8ddaaedc921d5f2ef2352fba712dda523f08 fix: restore admin usage list without detail table
7c527a1ab155aa22d61c6174244c1cb06c287a24 chore: sync VERSION to 0.1.104.1 [skip ci]
edd431386a639103644ddb9fa14adfc76b823e78 chore: ignore local superpowers files
2c46f8e2b7521d21765883ba7829773d1688b80d Merge branch 'feature/account-passthrough-fields'
692374535e3c6fec6e15ef459e5ecb514eb6ce4e feat: support account passthrough fields for apikey accounts
51c5a3b6daa3bca5f7914d98c4703994cd236f03 Merge branch 'admin-usage-detail'
4e90ef5a5a2ee5ce3a4aa9adb049e99fde399647 feat: add admin usage detail viewer
1a28546a4f8df6bdbbfa60845431faf8ff78856e chore: ignore local worktrees
9d57cbfcac7d41caf5a4169b773456aadd9a620e docs: add account passthrough fields design spec
aba3d034485d345a731509ff4af289ce5c550934 docs: refine usage detail design spec
c2d6518604ee5d38f69b232274533881862f1dd2 docs: add admin usage detail design spec
```

### 附录 B：v0.1.151..v0.1.152 changed-files（128 行）

命令：git diff --name-only v0.1.151..v0.1.152

```text
README.md
backend/cmd/server/VERSION
backend/ent/group.go
backend/ent/group/group.go
backend/ent/group/where.go
backend/ent/group_create.go
backend/ent/group_update.go
backend/ent/migrate/schema.go
backend/ent/mutation.go
backend/ent/runtime/runtime.go
backend/ent/schema/group.go
backend/go.mod
backend/go.sum
backend/internal/handler/admin/grok_oauth_handler_test.go
backend/internal/handler/admin/group_handler.go
backend/internal/handler/dto/mappers.go
backend/internal/handler/dto/types.go
backend/internal/handler/endpoint.go
backend/internal/handler/endpoint_test.go
backend/internal/handler/no_account_error.go
backend/internal/handler/no_account_error_test.go
backend/internal/handler/openai_alpha_search.go
backend/internal/handler/openai_chat_completions.go
backend/internal/handler/openai_gateway_compact_body_signal_test.go
backend/internal/handler/openai_gateway_count_tokens.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/ops_capture_writer_nil_test.go
backend/internal/handler/ops_error_logger.go
backend/internal/pkg/apicompat/anthropic_to_responses_response.go
backend/internal/pkg/apicompat/chatcompletions_responses_bridge.go
backend/internal/pkg/apicompat/chatcompletions_responses_bridge_custom_tools_test.go
backend/internal/pkg/apicompat/chatcompletions_responses_test.go
backend/internal/pkg/apicompat/responses_anthropic_cache_creation_test.go
backend/internal/pkg/apicompat/responses_stream_event_wire.go
backend/internal/pkg/apicompat/responses_stream_event_wire_test.go
backend/internal/pkg/apicompat/responses_to_anthropic.go
backend/internal/pkg/apicompat/types.go
backend/internal/repository/account_repo.go
backend/internal/repository/account_repo_integration_test.go
backend/internal/repository/api_key_repo.go
backend/internal/repository/group_repo.go
backend/internal/repository/http_upstream.go
backend/internal/repository/http_upstream_test.go
backend/internal/server/api_contract_test.go
backend/internal/server/routes/gateway.go
backend/internal/server/routes/gateway_test.go
backend/internal/service/account.go
backend/internal/service/account_base_url_test.go
backend/internal/service/account_test_service.go
backend/internal/service/account_test_service_grok_test.go
backend/internal/service/admin_group.go
backend/internal/service/admin_service.go
backend/internal/service/api_key_auth_cache.go
backend/internal/service/api_key_auth_cache_impl.go
backend/internal/service/billing_service.go
backend/internal/service/billing_service_test.go
backend/internal/service/grok_media.go
backend/internal/service/grok_oauth_service.go
backend/internal/service/grok_oauth_service_test.go
backend/internal/service/grok_quota_service.go
backend/internal/service/grok_quota_service_test.go
backend/internal/service/group.go
backend/internal/service/media_price_config.go
backend/internal/service/openai_alpha_search.go
backend/internal/service/openai_alpha_search_billing_test.go
backend/internal/service/openai_alpha_search_test.go
backend/internal/service/openai_codex_function_call_id_test.go
backend/internal/service/openai_codex_identity.go
backend/internal/service/openai_codex_identity_test.go
backend/internal/service/openai_codex_message_item_id_test.go
backend/internal/service/openai_codex_transform.go
backend/internal/service/openai_compact_body_signal.go
backend/internal/service/openai_compact_sse_keepalive.go
backend/internal/service/openai_compact_sse_keepalive_test.go
backend/internal/service/openai_compat_model_test.go
backend/internal/service/openai_gateway_cc_pipeline.go
backend/internal/service/openai_gateway_chat_completions.go
backend/internal/service/openai_gateway_chat_completions_raw.go
backend/internal/service/openai_gateway_forward.go
backend/internal/service/openai_gateway_grok.go
backend/internal/service/openai_gateway_grok_cache.go
backend/internal/service/openai_gateway_grok_cache_test.go
backend/internal/service/openai_gateway_grok_chat_bridge.go
backend/internal/service/openai_gateway_grok_chat_bridge_test.go
backend/internal/service/openai_gateway_grok_test.go
backend/internal/service/openai_gateway_messages.go
backend/internal/service/openai_gateway_messages_chat_fallback.go
backend/internal/service/openai_gateway_messages_chat_fallback_test.go
backend/internal/service/openai_gateway_responses_chat_fallback.go
backend/internal/service/openai_gateway_scheduling.go
backend/internal/service/openai_gateway_service.go
backend/internal/service/openai_gateway_usage.go
backend/internal/service/openai_gpt56_max_test.go
backend/internal/service/openai_oauth_passthrough_test.go
backend/internal/service/openai_ws_forwarder_ingress.go
backend/internal/service/openai_ws_forwarder_payload.go
backend/internal/service/openai_ws_forwarder_success_test.go
backend/internal/service/openai_ws_http_bridge.go
backend/internal/service/openai_ws_http_bridge_test.go
backend/internal/service/openai_ws_pool.go
backend/internal/service/openai_ws_pool_test.go
backend/migrations/174_group_web_search_price_per_call.sql
frontend/src/components/account/AccountUsageCell.vue
frontend/src/components/account/CreateAccountModal.vue
frontend/src/components/account/EditAccountModal.vue
frontend/src/components/account/UsageProgressBar.vue
frontend/src/components/account/__tests__/AccountStatusIndicator.spec.ts
frontend/src/components/account/__tests__/AccountUsageCell.spec.ts
frontend/src/components/account/__tests__/CreateAccountModal.grok.spec.ts
frontend/src/components/account/__tests__/EditAccountModal.spec.ts
frontend/src/components/account/__tests__/UsageProgressBar.spec.ts
frontend/src/components/keys/UseKeyModal.vue
frontend/src/components/keys/__tests__/UseKeyModal.spec.ts
frontend/src/composables/__tests__/useGrokOAuth.spec.ts
frontend/src/composables/useGrokOAuth.ts
frontend/src/i18n/__tests__/openaiFastPolicyLocales.spec.ts
frontend/src/i18n/__tests__/opsLocaleKeys.spec.ts
frontend/src/i18n/locales/en/admin/overview.ts
frontend/src/i18n/locales/en/admin/settings.ts
frontend/src/i18n/locales/en/dashboard.ts
frontend/src/i18n/locales/zh/admin/overview.ts
frontend/src/i18n/locales/zh/admin/settings.ts
frontend/src/i18n/locales/zh/dashboard.ts
frontend/src/types/index.ts
frontend/src/views/admin/GroupsView.vue
frontend/src/views/admin/SettingsView.vue
frontend/src/views/admin/settings/OpenAIFastPolicyUserSelector.vue
frontend/src/views/admin/settings/__tests__/OpenAIFastPolicyUserSelector.spec.ts
```

### 附录 C：v0.1.152..v0.1.153 changed-files（97 行）

命令：git diff --name-only v0.1.152..v0.1.153

```text
.github/workflows/backend-ci.yml
.gitignore
README.md
README_CN.md
README_JA.md
backend/cmd/server/VERSION
backend/cmd/server/wire_gen.go
backend/internal/config/config.go
backend/internal/config/config_test.go
backend/internal/handler/endpoint.go
backend/internal/handler/failover_loop.go
backend/internal/handler/failover_loop_test.go
backend/internal/handler/gateway_handler.go
backend/internal/handler/gateway_handler_chat_completions.go
backend/internal/handler/gateway_handler_responses.go
backend/internal/handler/gateway_handler_usage_test.go
backend/internal/handler/gateway_helper.go
backend/internal/handler/gateway_helper_fastpath_test.go
backend/internal/handler/gemini_v1beta_handler.go
backend/internal/handler/grok_media.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/openai_gateway_handler_test.go
backend/internal/handler/payment_handler.go
backend/internal/handler/payment_handler_resume_test.go
backend/internal/pkg/apicompat/anthropic_responses_test.go
backend/internal/pkg/apicompat/anthropic_to_responses_response.go
backend/internal/pkg/apicompat/chatcompletions_responses_bridge.go
backend/internal/pkg/apicompat/chatcompletions_responses_bridge_custom_tools_test.go
backend/internal/pkg/apicompat/responses_to_anthropic.go
backend/internal/pkg/apicompat/responses_to_anthropic_read_tool_test.go
backend/internal/pkg/apicompat/responses_to_chatcompletions.go
backend/internal/pkg/apicompat/streaming_stop_reason_test.go
backend/internal/pkg/pagination/pagination.go
backend/internal/pkg/pagination/pagination_test.go
backend/internal/pkg/xai/oauth.go
backend/internal/pkg/xai/oauth_test.go
backend/internal/repository/api_key_repo.go
backend/internal/repository/api_key_repo_last_used_unit_test.go
backend/internal/repository/concurrency_cache.go
backend/internal/repository/concurrency_cache_integration_test.go
backend/internal/repository/migrations_runner.go
backend/internal/repository/migrations_runner_notx_test.go
backend/internal/repository/scheduler_cache.go
backend/internal/repository/scheduler_cache_unit_test.go
backend/internal/server/routes/gateway.go
backend/internal/server/routes/gateway_test.go
backend/internal/server/routes/payment.go
backend/internal/service/account.go
backend/internal/service/account_base_url_test.go
backend/internal/service/account_test_service.go
backend/internal/service/account_test_service_openai_test.go
backend/internal/service/concurrency_service.go
backend/internal/service/concurrency_service_test.go
backend/internal/service/grok_media.go
backend/internal/service/openai_gateway_grok_test.go
backend/internal/service/openai_gateway_responses_chat_fallback.go
backend/internal/service/openai_ws_client.go
backend/internal/service/openai_ws_client_test.go
backend/internal/service/openai_ws_forwarder_ingress.go
backend/internal/service/openai_ws_forwarder_ingress_session_test.go
backend/internal/service/openai_ws_http_bridge_test.go
backend/internal/service/openai_ws_pool.go
backend/internal/service/openai_ws_pool_test.go
backend/internal/service/upstream_models.go
backend/internal/service/upstream_models_test.go
backend/internal/web/embed_on.go
backend/internal/web/embed_test.go
backend/internal/web/static_cache.go
backend/internal/web/static_cache_test.go
backend/migrations/174_add_usage_logs_api_key_latest_ip_index_notx.sql
backend/migrations/latest_api_key_ip_index_test.go
deploy/.env.example
deploy/APPLE_CONTAINER.md
deploy/README.md
deploy/apple-container.sh
deploy/config.example.yaml
deploy/tests/apple-container-test.sh
deploy/tests/fixtures/bin/container
deploy/tests/fixtures/bin/curl
frontend/src/api/payment.ts
frontend/src/components/account/EditAccountModal.vue
frontend/src/components/account/__tests__/credentialsBuilder.spec.ts
frontend/src/components/account/credentialsBuilder.ts
frontend/src/components/common/DataTable.vue
frontend/src/components/common/__tests__/DataTable.spec.ts
frontend/src/composables/__tests__/useSwipeSelect.spec.ts
frontend/src/composables/useSwipeSelect.ts
frontend/src/i18n/locales/en/admin/accounts.ts
frontend/src/i18n/locales/zh/admin/accounts.ts
frontend/src/i18n/locales/zh/admin/overview.ts
frontend/src/i18n/locales/zh/misc.ts
frontend/src/utils/__tests__/formatDateLocalInput.spec.ts
frontend/src/utils/format.ts
frontend/src/views/KeyUsageView.vue
frontend/src/views/__tests__/KeyUsageView.spec.ts
frontend/src/views/admin/AccountsView.vue
frontend/src/views/user/DashboardView.vue
```

### 附录 D：v0.1.153..v0.1.155 changed-files（238 行）

命令：git diff --name-only v0.1.153..v0.1.155

```text
backend/cmd/server/VERSION
backend/cmd/server/wire_gen.go
backend/ent/channelmonitor/channelmonitor.go
backend/ent/channelmonitorrequesttemplate/channelmonitorrequesttemplate.go
backend/ent/migrate/schema.go
backend/ent/mutation.go
backend/ent/runtime/runtime.go
backend/ent/schema/channel_monitor.go
backend/ent/schema/channel_monitor_request_template.go
backend/ent/schema/usage_log.go
backend/ent/usagelog.go
backend/ent/usagelog/usagelog.go
backend/ent/usagelog/where.go
backend/ent/usagelog_create.go
backend/ent/usagelog_update.go
backend/internal/config/config.go
backend/internal/config/config_test.go
backend/internal/handler/admin/account_codex_import.go
backend/internal/handler/admin/account_codex_import_test.go
backend/internal/handler/admin/account_data.go
backend/internal/handler/admin/account_handler.go
backend/internal/handler/admin/account_handler_long_context_billing_test.go
backend/internal/handler/admin/admin_service_stub_test.go
backend/internal/handler/admin/channel_monitor_handler.go
backend/internal/handler/admin/channel_monitor_template_handler.go
backend/internal/handler/admin/grok_import_probe.go
backend/internal/handler/admin/grok_import_probe_handler_test.go
backend/internal/handler/admin/grok_import_probe_test.go
backend/internal/handler/admin/grok_oauth_handler.go
backend/internal/handler/admin/grok_oauth_handler_test.go
backend/internal/handler/admin/openai_oauth_handler.go
backend/internal/handler/admin/ops_system_log_handler.go
backend/internal/handler/admin/ops_system_log_handler_test.go
backend/internal/handler/admin/ops_ws_handler.go
backend/internal/handler/dto/mappers.go
backend/internal/handler/dto/types.go
backend/internal/handler/openai_codex_models_handler.go
backend/internal/handler/openai_codex_models_handler_test.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/openai_gateway_handler_test.go
backend/internal/handler/openai_images.go
backend/internal/handler/wire.go
backend/internal/pkg/antigravity/client.go
backend/internal/pkg/apicompat/responses_namespace.go
backend/internal/pkg/apicompat/responses_namespace_test.go
backend/internal/pkg/httpclient/pool.go
backend/internal/pkg/servertiming/collector.go
backend/internal/pkg/servertiming/collector_test.go
backend/internal/pkg/servertiming/http.go
backend/internal/pkg/servertiming/http_test.go
backend/internal/pkg/xai/billing.go
backend/internal/pkg/xai/billing_test.go
backend/internal/pkg/xai/sso_device.go
backend/internal/pkg/xai/sso_device_test.go
backend/internal/repository/account_repo.go
backend/internal/repository/account_repo_auto_pause_test.go
backend/internal/repository/account_repo_grok_billing_test.go
backend/internal/repository/backup_s3_store.go
backend/internal/repository/claude_oauth_service.go
backend/internal/repository/ent.go
backend/internal/repository/grok_oauth_client.go
backend/internal/repository/http_upstream.go
backend/internal/repository/http_upstream_http2_keepalive_test.go
backend/internal/repository/openai_long_context_billing_migration_integration_test.go
backend/internal/repository/ops_repo.go
backend/internal/repository/ops_repo_system_logs_test.go
backend/internal/repository/proxy_expiry_integration_test.go
backend/internal/repository/proxy_expiry_test.go
backend/internal/repository/proxy_repo.go
backend/internal/repository/redis.go
backend/internal/repository/req_client_pool.go
backend/internal/repository/req_client_pool_test.go
backend/internal/repository/scheduler_outbox_repo.go
backend/internal/repository/scheduler_outbox_repo_test.go
backend/internal/repository/server_timing_redis.go
backend/internal/repository/server_timing_redis_test.go
backend/internal/repository/server_timing_sql.go
backend/internal/repository/server_timing_sql_test.go
backend/internal/repository/usage_log_repo_insert.go
backend/internal/repository/usage_log_repo_query.go
backend/internal/repository/usage_log_repo_request_type_test.go
backend/internal/server/api_contract_test.go
backend/internal/server/middleware/cors.go
backend/internal/server/middleware/cors_test.go
backend/internal/server/middleware/server_timing.go
backend/internal/server/middleware/server_timing_test.go
backend/internal/server/router.go
backend/internal/server/routes/admin.go
backend/internal/service/account.go
backend/internal/service/account_base_url_test.go
backend/internal/service/account_long_context_billing_test.go
backend/internal/service/account_test_service.go
backend/internal/service/account_test_service_grok_test.go
backend/internal/service/account_usage_service.go
backend/internal/service/admin_account.go
backend/internal/service/admin_service_spark_shadow_test.go
backend/internal/service/billing_service.go
backend/internal/service/billing_service_test.go
backend/internal/service/channel_monitor_checker.go
backend/internal/service/channel_monitor_checker_body_test.go
backend/internal/service/channel_monitor_const.go
backend/internal/service/channel_monitor_service.go
backend/internal/service/channel_monitor_service_grok_test.go
backend/internal/service/channel_monitor_template_types.go
backend/internal/service/channel_monitor_validate.go
backend/internal/service/content_moderation.go
backend/internal/service/crs_sync_long_context_billing_test.go
backend/internal/service/crs_sync_service.go
backend/internal/service/gateway_usage_billing.go
backend/internal/service/grok_media.go
backend/internal/service/grok_oauth_service.go
backend/internal/service/grok_oauth_service_test.go
backend/internal/service/grok_quota_fetcher.go
backend/internal/service/grok_quota_fetcher_test.go
backend/internal/service/grok_quota_service.go
backend/internal/service/grok_quota_service_test.go
backend/internal/service/image_generation_intent.go
backend/internal/service/model_not_found_error.go
backend/internal/service/model_not_found_error_test.go
backend/internal/service/oauth_service.go
backend/internal/service/openai_alpha_search_billing_test.go
backend/internal/service/openai_codex_models_service.go
backend/internal/service/openai_codex_models_service_test.go
backend/internal/service/openai_codex_transform.go
backend/internal/service/openai_codex_transform_test.go
backend/internal/service/openai_compat_model_test.go
backend/internal/service/openai_content_session_seed.go
backend/internal/service/openai_content_session_seed_test.go
backend/internal/service/openai_gateway_chat_completions_test.go
backend/internal/service/openai_gateway_forward.go
backend/internal/service/openai_gateway_grok.go
backend/internal/service/openai_gateway_grok_cache.go
backend/internal/service/openai_gateway_grok_cache_test.go
backend/internal/service/openai_gateway_grok_test.go
backend/internal/service/openai_gateway_passthrough.go
backend/internal/service/openai_gateway_record_usage_test.go
backend/internal/service/openai_gateway_request_body.go
backend/internal/service/openai_gateway_request_body_reasoning_test.go
backend/internal/service/openai_gateway_response_handling.go
backend/internal/service/openai_gateway_service.go
backend/internal/service/openai_gateway_service_hotpath_test.go
backend/internal/service/openai_gateway_service_test.go
backend/internal/service/openai_gateway_usage.go
backend/internal/service/openai_image_generation_controls_test.go
backend/internal/service/openai_images_json_keepalive.go
backend/internal/service/openai_images_json_keepalive_test.go
backend/internal/service/openai_images_responses.go
backend/internal/service/openai_messages_dispatch_test.go
backend/internal/service/openai_model_mapping.go
backend/internal/service/openai_model_mapping_test.go
backend/internal/service/openai_oauth_passthrough_test.go
backend/internal/service/openai_quota_reset_credits.go
backend/internal/service/openai_quota_reset_credits_test.go
backend/internal/service/openai_quota_service.go
backend/internal/service/openai_quota_spark_window_test.go
backend/internal/service/openai_responses_namespace.go
backend/internal/service/openai_responses_namespace_test.go
backend/internal/service/openai_ws_forwarder_ingress.go
backend/internal/service/openai_ws_forwarder_ingress_session_test.go
backend/internal/service/openai_ws_http_bridge.go
backend/internal/service/openai_ws_http_bridge_test.go
backend/internal/service/ops_models.go
backend/internal/service/ops_port.go
backend/internal/service/ops_system_log_service.go
backend/internal/service/ops_system_log_service_test.go
backend/internal/service/ops_system_log_sink.go
backend/internal/service/ops_system_log_sink_test.go
backend/internal/service/payment_order.go
backend/internal/service/payment_order_lifecycle.go
backend/internal/service/payment_refund.go
backend/internal/service/ratelimit_service.go
backend/internal/service/ratelimit_service_model_not_found_test.go
backend/internal/service/scheduler_outbox.go
backend/internal/service/scheduler_snapshot_full_rebuild_test.go
backend/internal/service/scheduler_snapshot_outbox_cleanup_test.go
backend/internal/service/scheduler_snapshot_service.go
backend/internal/service/usage_log.go
backend/internal/service/vertex_service_account.go
backend/internal/service/vertex_service_account_test.go
backend/internal/service/wire.go
backend/migrations/174_add_usage_log_long_context_billing.sql
backend/migrations/175_add_ops_system_logs_host.sql
backend/migrations/175_default_openai_long_context_billing.sql
backend/migrations/175a_add_ops_system_logs_host_index_notx.sql
backend/migrations/176_channel_monitor_grok_provider.sql
backend/migrations/channel_monitor_grok_provider_migration_test.go
backend/migrations/openai_long_context_billing_migration_test.go
deploy/.env.example
deploy/config.example.yaml
deploy/docker-compose.dev.yml
deploy/docker-compose.local.yml
deploy/docker-compose.standalone.yml
deploy/docker-compose.yml
frontend/src/api/__tests__/admin.grok.spec.ts
frontend/src/api/__tests__/adminUIRequest.spec.ts
frontend/src/api/__tests__/client.spec.ts
frontend/src/api/admin/channelMonitor.ts
frontend/src/api/admin/grok.ts
frontend/src/api/admin/ops.ts
frontend/src/api/adminUIRequest.ts
frontend/src/api/client.ts
frontend/src/components/account/AccountUsageCell.vue
frontend/src/components/account/CreateAccountModal.vue
frontend/src/components/account/EditAccountModal.vue
frontend/src/components/account/GrokQuotaProbeCell.vue
frontend/src/components/account/OAuthAuthorizationFlow.vue
frontend/src/components/account/__tests__/AccountUsageCell.spec.ts
frontend/src/components/account/__tests__/CreateAccountModal.spec.ts
frontend/src/components/account/__tests__/EditAccountModal.spec.ts
frontend/src/components/account/__tests__/GrokQuotaProbeCell.spec.ts
frontend/src/components/admin/account/AccountTestModal.vue
frontend/src/components/admin/monitor/MonitorAdvancedRequestConfig.vue
frontend/src/components/admin/monitor/MonitorFiltersBar.vue
frontend/src/components/admin/monitor/MonitorFormDialog.vue
frontend/src/components/admin/monitor/MonitorTemplateManagerDialog.vue
frontend/src/components/admin/usage/UsageTable.vue
frontend/src/components/admin/usage/__tests__/UsageTable.spec.ts
frontend/src/components/common/GrokFreeIcon.vue
frontend/src/components/common/PlatformTypeBadge.vue
frontend/src/components/common/__tests__/PlatformTypeBadge.grok.spec.ts
frontend/src/components/user/monitor/MonitorCard.vue
frontend/src/components/user/monitor/ProviderIcon.vue
frontend/src/composables/useAccountOAuth.ts
frontend/src/composables/useChannelMonitorFormat.ts
frontend/src/composables/useGrokOAuth.ts
frontend/src/constants/channelMonitor.ts
frontend/src/i18n/locales/en/admin/accounts.ts
frontend/src/i18n/locales/en/admin/ops.ts
frontend/src/i18n/locales/en/dashboard.ts
frontend/src/i18n/locales/zh/admin/accounts.ts
frontend/src/i18n/locales/zh/admin/ops.ts
frontend/src/i18n/locales/zh/dashboard.ts
frontend/src/types/index.ts
frontend/src/views/admin/AccountsView.vue
frontend/src/views/admin/__tests__/AccountsView.sparkShadow.spec.ts
frontend/src/views/admin/__tests__/ChannelMonitorView.grok.spec.ts
frontend/src/views/admin/ops/components/OpsSystemLogTable.vue
frontend/src/views/admin/ops/components/__tests__/OpsSystemLogTable.spec.ts
```

### 附录 E：v0.1.155..v0.1.156 changed-files（253 行）

命令：git diff --name-only v0.1.155..v0.1.156

```text
README.md
README_CN.md
README_JA.md
assets/partners/logos/aimzoon.jpg
backend/cmd/server/VERSION
backend/cmd/server/wire_gen.go
backend/internal/config/config.go
backend/internal/config/config_test.go
backend/internal/handler/admin/account_codex_agent_identity_import_test.go
backend/internal/handler/admin/account_codex_import.go
backend/internal/handler/admin/account_codex_import_test.go
backend/internal/handler/admin/account_data_handler_test.go
backend/internal/handler/admin/account_handler.go
backend/internal/handler/admin/account_handler_available_models_test.go
backend/internal/handler/admin/account_handler_duplicate_test.go
backend/internal/handler/admin/account_handler_grok_refresh_test.go
backend/internal/handler/admin/account_handler_list_test.go
backend/internal/handler/admin/account_handler_long_context_billing_test.go
backend/internal/handler/admin/account_handler_mixed_channel_test.go
backend/internal/handler/admin/account_handler_passthrough_test.go
backend/internal/handler/admin/admin_service_stub_test.go
backend/internal/handler/admin/batch_update_credentials_test.go
backend/internal/handler/admin/grok_import_probe.go
backend/internal/handler/admin/grok_import_probe_handler_test.go
backend/internal/handler/admin/grok_oauth_handler.go
backend/internal/handler/admin/grok_oauth_handler_test.go
backend/internal/handler/admin/idempotency_helper.go
backend/internal/handler/admin/ops_handler.go
backend/internal/handler/dto/credentials_redact_test.go
backend/internal/handler/failover_loop.go
backend/internal/handler/failover_loop_test.go
backend/internal/handler/gateway_handler.go
backend/internal/handler/gateway_handler_cancellation_test.go
backend/internal/handler/gateway_handler_chat_completions.go
backend/internal/handler/gateway_handler_responses.go
backend/internal/handler/gateway_handler_warmup_intercept_unit_test.go
backend/internal/handler/gateway_helper.go
backend/internal/handler/gateway_helper_test.go
backend/internal/handler/gemini_v1beta_handler.go
backend/internal/handler/grok_media.go
backend/internal/handler/openai_alpha_search.go
backend/internal/handler/openai_chat_completions.go
backend/internal/handler/openai_embeddings.go
backend/internal/handler/openai_gateway_compact_log_test.go
backend/internal/handler/openai_gateway_credential_failover_loop_test.go
backend/internal/handler/openai_gateway_credential_failover_test.go
backend/internal/handler/openai_gateway_first_output_timeout_test.go
backend/internal/handler/openai_gateway_handler.go
backend/internal/handler/openai_gateway_handler_test.go
backend/internal/handler/openai_image_intent_hint_test.go
backend/internal/handler/openai_images.go
backend/internal/handler/openai_responses_failover_cancel_test.go
backend/internal/handler/ops_error_logger.go
backend/internal/pkg/apicompat/anthropic_to_responses_response.go
backend/internal/pkg/apicompat/chatcompletions_anthropic_bridge.go
backend/internal/pkg/apicompat/chatcompletions_anthropic_bridge_test.go
backend/internal/pkg/apicompat/chatcompletions_responses_bridge.go
backend/internal/pkg/apicompat/chatcompletions_responses_bridge_custom_tools_test.go
backend/internal/pkg/apicompat/responses_to_anthropic.go
backend/internal/pkg/apicompat/responses_to_anthropic_parallel_tool_test.go
backend/internal/pkg/apicompat/responses_to_anthropic_read_tool_test.go
backend/internal/pkg/apicompat/streaming_stop_reason_test.go
backend/internal/pkg/servertiming/collector.go
backend/internal/pkg/xai/oauth.go
backend/internal/pkg/xai/oauth_test.go
backend/internal/repository/account_repo.go
backend/internal/repository/account_repo_duplicate_integration_test.go
backend/internal/repository/account_repo_integration_test.go
backend/internal/repository/account_repo_schedulable_projection_integration_test.go
backend/internal/repository/account_repo_schedulable_projection_test.go
backend/internal/repository/account_repo_temp_unsched_test.go
backend/internal/repository/grok_oauth_client.go
backend/internal/repository/grok_oauth_client_test.go
backend/internal/repository/ops_error_where_test.go
backend/internal/repository/ops_repo.go
backend/internal/repository/ops_repo_args_test.go
backend/internal/repository/ops_repo_get_error_log_by_id_integration_test.go
backend/internal/repository/scheduler_cache.go
backend/internal/repository/scheduler_cache_integration_test.go
backend/internal/repository/scheduler_cache_unit_test.go
backend/internal/repository/wire.go
backend/internal/server/api_contract_test.go
backend/internal/server/middleware/cors.go
backend/internal/server/middleware/cors_test.go
backend/internal/server/middleware/server_timing.go
backend/internal/server/middleware/server_timing_test.go
backend/internal/server/routes/admin.go
backend/internal/server/routes/gateway.go
backend/internal/server/routes/gateway_codex_models_test.go
backend/internal/service/account.go
backend/internal/service/account_base_url_test.go
backend/internal/service/account_credentials_redact.go
backend/internal/service/account_service.go
backend/internal/service/account_stats_pricing.go
backend/internal/service/account_stats_pricing_test.go
backend/internal/service/account_test_service.go
backend/internal/service/account_test_service_grok_test.go
backend/internal/service/account_usage_service.go
backend/internal/service/admin_account.go
backend/internal/service/admin_service.go
backend/internal/service/admin_service_duplicate_account_test.go
backend/internal/service/content_moderation.go
backend/internal/service/content_moderation_keyword_matcher.go
backend/internal/service/content_moderation_keyword_matcher_test.go
backend/internal/service/content_moderation_runtime_cache_test.go
backend/internal/service/gateway_service.go
backend/internal/service/grok_credential_failure.go
backend/internal/service/grok_credential_failure_test.go
backend/internal/service/grok_media.go
backend/internal/service/grok_oauth_reconciliation.go
backend/internal/service/grok_oauth_reconciliation_test.go
backend/internal/service/grok_oauth_service.go
backend/internal/service/grok_quota_service.go
backend/internal/service/grok_quota_service_test.go
backend/internal/service/grok_token_provider.go
backend/internal/service/grok_token_provider_test.go
backend/internal/service/grok_token_refresher.go
backend/internal/service/grok_upstream_url.go
backend/internal/service/grok_upstream_url_test.go
backend/internal/service/image_generation_intent.go
backend/internal/service/image_generation_intent_benchmark_test.go
backend/internal/service/image_generation_intent_test.go
backend/internal/service/oauth_refresh_api.go
backend/internal/service/oauth_refresh_api_test.go
backend/internal/service/openai_account_runtime_block_fastpath.go
backend/internal/service/openai_account_runtime_block_fastpath_test.go
backend/internal/service/openai_agent_identity.go
backend/internal/service/openai_agent_identity_compat_test.go
backend/internal/service/openai_agent_identity_test.go
backend/internal/service/openai_codex_models_service.go
backend/internal/service/openai_codex_models_service_test.go
backend/internal/service/openai_codex_transform.go
backend/internal/service/openai_codex_transform_test.go
backend/internal/service/openai_content_session_seed.go
backend/internal/service/openai_content_session_seed_benchmark_test.go
backend/internal/service/openai_content_session_seed_test.go
backend/internal/service/openai_first_output_timeout.go
backend/internal/service/openai_first_output_timeout_test.go
backend/internal/service/openai_gateway_cc_pipeline.go
backend/internal/service/openai_gateway_chat_completions.go
backend/internal/service/openai_gateway_chat_completions_raw.go
backend/internal/service/openai_gateway_count_tokens.go
backend/internal/service/openai_gateway_forward.go
backend/internal/service/openai_gateway_grok.go
backend/internal/service/openai_gateway_grok_cache.go
backend/internal/service/openai_gateway_grok_cache_test.go
backend/internal/service/openai_gateway_grok_chat_bridge.go
backend/internal/service/openai_gateway_grok_chat_bridge_test.go
backend/internal/service/openai_gateway_grok_test.go
backend/internal/service/openai_gateway_messages.go
backend/internal/service/openai_gateway_messages_chat_fallback.go
backend/internal/service/openai_gateway_messages_failed_response_test.go
backend/internal/service/openai_gateway_passthrough.go
backend/internal/service/openai_gateway_passthrough_flush_test.go
backend/internal/service/openai_gateway_passthrough_image_intent_benchmark_test.go
backend/internal/service/openai_gateway_passthrough_image_intent_test.go
backend/internal/service/openai_gateway_response_flush_test.go
backend/internal/service/openai_gateway_response_handling.go
backend/internal/service/openai_gateway_service.go
backend/internal/service/openai_gateway_service_hotpath_test.go
backend/internal/service/openai_gateway_upstream_errors.go
backend/internal/service/openai_image_dimensions.go
backend/internal/service/openai_image_generation_controls_test.go
backend/internal/service/openai_image_intent_hint.go
backend/internal/service/openai_image_intent_hint_test.go
backend/internal/service/openai_images.go
backend/internal/service/openai_images_actual_size_test.go
backend/internal/service/openai_images_json_keepalive.go
backend/internal/service/openai_images_responses.go
backend/internal/service/openai_oauth_passthrough_test.go
backend/internal/service/openai_quota_reset_credits.go
backend/internal/service/openai_quota_reset_credits_test.go
backend/internal/service/openai_quota_service.go
backend/internal/service/openai_quota_spark_window_test.go
backend/internal/service/openai_responses_lite_tools.go
backend/internal/service/openai_responses_lite_tools_test.go
backend/internal/service/openai_sse_concatenated_json_test.go
backend/internal/service/openai_sse_json_documents.go
backend/internal/service/openai_ws_client.go
backend/internal/service/openai_ws_forwarder_ingress.go
backend/internal/service/openai_ws_forwarder_ingress_session_test.go
backend/internal/service/openai_ws_forwarder_payload.go
backend/internal/service/openai_ws_forwarder_v2.go
backend/internal/service/openai_ws_http_bridge.go
backend/internal/service/openai_ws_http_bridge_test.go
backend/internal/service/openai_ws_pool.go
backend/internal/service/openai_ws_v2_passthrough_adapter.go
backend/internal/service/ops_metrics_collector.go
backend/internal/service/ops_metrics_collector_projection_test.go
backend/internal/service/ops_models.go
backend/internal/service/ops_service.go
backend/internal/service/ops_service_batch_test.go
backend/internal/service/ops_upstream_context.go
backend/internal/service/ops_user_error.go
backend/internal/service/ops_user_error_test.go
backend/internal/service/refresh_policy.go
backend/internal/service/scheduler_cache.go
backend/internal/service/scheduler_snapshot_batch_query_test.go
backend/internal/service/scheduler_snapshot_full_rebuild_lifecycle_test.go
backend/internal/service/scheduler_snapshot_full_rebuild_test.go
backend/internal/service/scheduler_snapshot_group_lifecycle_test.go
backend/internal/service/scheduler_snapshot_hydration_test.go
backend/internal/service/scheduler_snapshot_outbox_cleanup_test.go
backend/internal/service/scheduler_snapshot_retirement_test.go
backend/internal/service/scheduler_snapshot_service.go
backend/internal/service/token_refresh_pool_health_test.go
backend/internal/service/token_refresh_service.go
backend/internal/service/token_refresh_service_candidates_test.go
backend/internal/service/token_refresh_service_test.go
backend/internal/service/wire.go
backend/internal/web/embed_on.go
backend/internal/web/embed_test.go
backend/internal/web/static_cache.go
backend/internal/web/static_cache_test.go
deploy/Caddyfile
deploy/config.example.yaml
deploy/test-caddyfile-cache.sh
frontend/src/api/__tests__/admin.accounts.duplicate.spec.ts
frontend/src/api/__tests__/adminUIRequest.spec.ts
frontend/src/api/__tests__/client.spec.ts
frontend/src/api/admin/accounts.ts
frontend/src/api/adminUIRequest.ts
frontend/src/api/client.ts
frontend/src/components/account/CreateAccountModal.vue
frontend/src/components/account/OAuthAuthorizationFlow.vue
frontend/src/components/admin/account/AccountActionMenu.vue
frontend/src/components/admin/account/__tests__/AccountActionMenu.spark_shadow.spec.ts
frontend/src/components/admin/usage/UsageFilters.vue
frontend/src/components/common/DataTable.vue
frontend/src/components/common/PlatformTypeBadge.vue
frontend/src/components/common/__tests__/DataTable.spec.ts
frontend/src/components/common/__tests__/PlatformTypeBadge.grok.spec.ts
frontend/src/components/keys/UseKeyModal.vue
frontend/src/components/keys/__tests__/UseKeyModal.spec.ts
frontend/src/composables/__tests__/useAntigravityOAuth.spec.ts
frontend/src/composables/useAntigravityOAuth.ts
frontend/src/i18n/locales/en/admin/accounts.ts
frontend/src/i18n/locales/en/admin/ops.ts
frontend/src/i18n/locales/en/admin/overview.ts
frontend/src/i18n/locales/en/dashboard.ts
frontend/src/i18n/locales/zh/admin/accounts.ts
frontend/src/i18n/locales/zh/admin/ops.ts
frontend/src/i18n/locales/zh/admin/overview.ts
frontend/src/i18n/locales/zh/dashboard.ts
frontend/src/utils/errorCategory.ts
frontend/src/views/admin/AccountsView.vue
frontend/src/views/admin/GroupsView.vue
frontend/src/views/admin/__tests__/AccountsView.sparkShadow.spec.ts
frontend/src/views/admin/__tests__/GroupsView.columnSettings.spec.ts
frontend/src/views/admin/ops/components/OpsErrorDetailsModal.vue
frontend/src/views/admin/ops/components/OpsErrorLogTable.vue
frontend/src/views/user/KeysView.vue
frontend/src/views/user/__tests__/KeysView.spec.ts
```

## Task 7 / v0.1.152 阶段门禁（OpenSpec 2.3）

### 前两次 BLOCKED 与修复闭环

- 第一次完整重跑为 **BLOCKED**：`make test` 退出 `1`，Grok 429 failed-usage fixture 因未实现 `UpdateExtra`/`SetRateLimited` 持久化契约而将预期 `429` 转为 `502`；`pnpm --dir frontend run build` 退出 `1`，`OpenAIFastPolicyUserSelector.vue` 从未导出的模块导入 `SimpleUser`，产生 `TS2614` 和三个 `TS7006`。
- `4b697fa0a` `test: support Grok quota persistence in failed usage fixture`：RED 为 `TestOpenAIGatewayHandler_GrokMediaFailoverExhaustedCreatesFailedUsage`，期望 `429`、实际 `502`；GREEN 为同一目标及相邻 failed-usage 组通过。仅修改测试 fixture。
- `3ede52fc0` `fix: import SimpleUser from shared types`：RED 为前端 build/typecheck 的 `TS2614` 与三个 `TS7006`，修正后暴露 hydration 缺少必填 `username`；GREEN 为 `pnpm --dir frontend run typecheck` 和 selector Vitest `1` 文件、`3/3` 通过。仅修改该 Vue type-only import/hydration 映射。
- 第二次完整重跑为 **BLOCKED**：首条 `make test` 退出 `2`，unit service 中 `TestOpenAIWSHTTPBridgeGrok429PersistsRateLimit` 期望一次 rate-limit 持久化、实际两次；后续门禁按规则未执行。
- `2c639fa8b` `fix: persist Grok WS rate limits once`：RED 为该目标退出 `1`、`rateLimitedCalls` 期望 `1` 实际 `2`；GREEN 为同一目标及 WS bridge/Grok 相邻 `3/3`、Grok quota 服务组 `3/3` 通过。仅移除 failover 分支的重复 Grok 处理，保留首次持久化和 failover 返回语义。

### 第三次完整重跑（2026-07-16）

**结论：PASS。** 当前 `HEAD` 包含 `4b697fa0a`、`3ede52fc0`、`2c639fa8b`；本次没有修改业务代码、测试或生成物，没有 merge、push、release、deploy 或 main 合并，也没有勾选任何计划、OpenSpec task 或 Comet progress。

| 全量门禁命令 | 退出码 | 测试/结果 |
| --- | ---: | --- |
| `make test` | 0 | 后端 default、lint、unit，以及前端 ESLint/typecheck/Vitest 均完成；Vitest `171` 文件、`1265/1265` 测试通过。 |
| `pnpm --dir frontend run build` | 0 | `vue-tsc -b` 与 Vite 生产构建完成，转换 `968` modules，`built in 33.47s`。 |
| `make -C backend generate` | 0 | Ent 和 Wire 完成，Wire 两次写入 `backend/cmd/server/wire_gen.go`。 |
| `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` | 0 | stdout 为空，受限生成 diff 为空。 |

### 受影响 M-ID 稳定命令

| M-ID | 根目录命令 | 退出码 | 测试/结论 |
| --- | --- | ---: | --- |
| M-01 | `go -C backend test ./internal/service -run '^(TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount|TestOpenAISelectAccountWithLoadAwareness_AllFullWaitPlan)$' -count=1` | 0 | 2 个命名目标通过。 |
| M-02 | `go -C backend test -tags unit ./internal/service -run '^(TestOpenAIGatewayService_SelectAccountWithScheduler_PreviousResponseStickyDisabledBypassesStickyLookupAndBind|TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDisabledBypassesLookupBindAndRefresh|TestOpenAIGatewayService_StickyDisabledWSv2SkipsStateStore|TestOpenAIGatewayService_StickyDisabledIngressSkipsStateStore)$' -count=1` | 0 | 4 个命名目标通过。 |
| M-04 | `go -C backend test ./internal/handler -run '^(TestHandleFailoverError_IntegrationScenario|TestStreamWrittenGuard_MessagesPath_AbortFailoverOnSSEContentWritten)$' -count=1` | 0 | 2 个命名目标通过。 |
| M-05 | `go -C backend test ./internal/service -run '^(TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDBRuntimeRecheckSkipsStaleCachedAccount|TestLayered_SessionStickyRecheckHonorsImageCapability|TestOpenAIGatewayService_SelectAccountByPreviousResponseID_DBRuntimeRecheckRateLimitedMiss)$' -count=1` | 0 | 3 个命名目标通过。 |
| M-06 矩阵 | `go -C backend test ./internal/pkg/apicompat ./internal/service -run '^(TestChatCompletionsToResponses_ToolCalls|TestResponsesToAnthropic_CustomToolPreservesSchemaParameters|TestGatewayService_ForwardAsChatCompletions_PassthroughBodyMapCopiesFromOriginalCCBody|TestGatewayService_ForwardAsResponses_PassthroughBodyMapCopiesFromOriginalResponsesBody)$' -count=1` | 0 | apicompat 2 个目标通过；service 为 `no tests to run`。 |
| M-06 实际补充 | `go -C backend test -tags unit ./internal/service -run '^(TestGatewayService_ForwardAsChatCompletions_PassthroughBodyMapCopiesFromOriginalCCBody|TestGatewayService_ForwardAsResponses_PassthroughBodyMapCopiesFromOriginalResponsesBody)$' -count=1` | 0 | 2 个 service unit 目标通过。 |
| M-07 | `go -C backend test ./internal/pkg/apicompat ./internal/handler -run '^(TestResponsesEventToChatChunks_TopLevelTerminalUsage|TestResponsesEventToAnthropicEvents_TopLevelTerminalUsage|TestOpenAIGatewayHandler_NativeNonPassthroughResponsesFailedIsNotDuplicated|TestOpenAIGatewayHandler_ResponsesPartialFailoverCreatesExactlyOneFailedUsage)$' -count=1` | 0 | 4 个命名目标通过。 |
| M-09 | `go -C backend test ./internal/service -run '^(TestLayered_RequiredImageCapabilityFiltersUnsupportedAccounts|TestLayered_SessionStickyRecheckHonorsImageCapability|TestOpenAIGatewayServiceForward_RejectsDisabledImageGenerationIntents|TestOpenAIGatewayServiceForwardImages_ImageRateLimitReturnsFailoverAndCoolsCapability)$' -count=1` | 0 | 4 个命名目标通过。 |
| M-10 后端 | `go -C backend test ./internal/service -run '^(TestSettingService_SetGatewayRuntimeSettings_PersistsUpdatesCfgAndInvalidatesOnResponseHeaderTimeoutChange|TestSettingServiceGatewayRuntimeSettings_RejectsNegativeFirstTokenTimeouts)$' -count=1` | 0 | 2 个命名目标通过。 |
| M-10 前端 | `pnpm --dir frontend test:run src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts` | 0 | 1 个文件、`14/14` 通过。 |
| M-11 | `go -C backend test ./internal/handler ./internal/service -run '^(TestGatewayHandler_MessagesAndResponsesReplayLargeBodiesAcrossFailover|TestOpenAIGatewayHandler_ChatReplayRawSpoolAcrossFailoverWhenResponsesUnsupported|TestRequestBodyCoordinator_CleanupRemovesRawEffectiveAndMultipartTemps|TestOpenAIForwardCleansBoundRequestBodyHandlesInParallel|TestRequestBodyHandle_CleanupRemovesSpoolFile)$' -count=1` | 0 | 5 个命名目标通过。 |
| M-12 矩阵 | `go -C backend test ./internal/service ./internal/handler -run '^(TestAdminServiceUpdateUserSavesHiddenUIResources|TestPaymentHandlerGetCheckoutInfoRejectsHiddenPurchasePage|TestPageHandlerGetPageContentRejectsHiddenCustomMenu)$' -count=1` | 0 | handler 2 个目标通过；service 为 `no tests to run`。 |
| M-12 实际补充 | `go -C backend test -tags unit ./internal/service ./internal/handler -run '^(TestAdminServiceUpdateUserSavesHiddenUIResources|TestPaymentHandlerGetCheckoutInfoRejectsHiddenPurchasePage|TestPageHandlerGetPageContentRejectsHiddenCustomMenu)$' -count=1` | 0 | 3 个命名目标通过。 |
| M-13 | `pnpm --dir frontend test:run src/router/__tests__/feature-access.spec.ts src/utils/__tests__/userUiVisibility.spec.ts src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts` | 0 | 3 个文件、`21/21` 通过。 |
| M-14 | `go -C backend test ./cmd/server -run '^$' -count=1` | 0 | 编译门禁，0 测试。 |
| M-15 migration | `go -C backend test ./migrations -run '^(TestMigration173AllowsCyberBlockedUsageRequestType|TestMigration158BackfillsGrokMediaGenerationGroups)$' -count=1` | 0 | 2 个命名目标通过。 |
| M-16 | `go -C backend test -v -tags unit ./internal/service ./internal/handler ./internal/service/openai_ws_v2 -run '^(TestOpenAIGatewayService_SelectAccountWithScheduler_PreviousResponseStickyDisabledBypassesStickyLookupAndBind|TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDisabledBypassesLookupBindAndRefresh|TestOpenAIGatewayService_StickyDisabledWSv2SkipsStateStore|TestOpenAIGatewayService_StickyDisabledIngressSkipsStateStore|TestGatewayHandler_GeminiRouteStickyLookupUsesGeminiToggleNotAnthropicToggle|TestGatewayHandler_GeminiRouteStickyBindUsesGeminiToggleNotAnthropicToggle|TestGatewayHandler_OpenAIRouteStickyUsesOpenAIToggleNotAnthropicToggle|TestAntigravityGatewayServiceClearStickySessionSkipsDisabledSticky|TestOpenAIGatewayHandler_FirstTokenTimeoutReturns504AndCreatesOneFailedUsage|TestRelayFirstTokenTimeoutCancelsDrainsAndCompletesTurn)$' -count=1` | 0 | 10 个顶层目标、6 个子测试均 RUN/PASS。 |

- M-03、M-08 按 Task 6 的 v0.1.152 changed-file 交集结论为 N/A，未记为通过项；其余 `14` 个受影响 M-ID 均有上述稳定命令和 PASS 证据。

### 15 项冲突入口与 alpha-search

| 冲突项 | 验证入口 | 退出码 | 测试/结论 |
| --- | --- | ---: | --- |
| 1 VERSION | M-14 编译门禁与 Task 6 tag/version 人工对照 | 0 | 本地 `0.1.151.2` 保留裁决未变。 |
| 2-3 Ent mutation/runtime | M-15 生成及受限 diff | 0 | 生成稳定，无 diff。 |
| 4-5 Chat/handler endpoint | `go -C backend test ./internal/handler -run '^(TestResolveOpenAIUpstreamEndpointPrefersForwardResult|TestOpenAIGatewayHandler_ChatCompletionsUsageTaskUsesCapturedEndpointAndSnapshot|TestClassifyNoAccountError_ModelNotSupported_Returns404|TestClassifyOpenAICompatibleNoAccountError_GrokUsesGrokPlatform)$' -count=1` | 0 | 4 个命名目标通过。 |
| 6 routes | `go -C backend test ./internal/server/routes -run '^(TestGatewayRoutesOpenAIAlphaSearchPathsAreRegistered|TestGatewayRoutesAlphaSearchRejectsNonOpenAIGroup)$' -count=1` | 0 | 2 个命名目标通过。 |
| 7-8 API-key cache/Grok | 无标签原命令同一 5-target 正则 | 0 | service 为 `no tests to run`，不作为实际覆盖。 |
| 7-8 API-key cache/Grok 实际补充 | `go -C backend test -tags unit ./internal/service -run '^(TestAPIKeyService_GetByKey_UsesL2Cache|TestAPIKeyService_GetByKey_CacheHitPreservesGroupUserConcurrencyFields|TestAccountTestService_Grok429PersistsRateLimitReset|TestAccountTestService_Grok429WithoutQuotaHeadersUsesFallback|TestGrokQuotaServiceProbeUsageStoresHeaders)$' -count=1` | 0 | 5 个 unit 目标通过。 |
| 9 fallback | M-06 与 M-16 | 0 | 协议 2 个实际 unit 目标和首 Token/Sticky 10 顶层目标通过。 |
| 10 gateway service | endpoint/usage 4-target handler 入口 | 0 | usage snapshot 与 endpoint 分类通过。 |
| 11 OAuth passthrough | `go -C backend test ./internal/service -run '^(TestOpenAIOAuthPassthrough|TestForwardOpenAIPassthrough|TestOpenAIForward.*Passthrough)' -count=1` | 0 | 命名 passthrough 组通过。 |
| 12-15 Create/Edit/locales | `pnpm --dir frontend test:run src/components/account/__tests__/passthroughFieldSupport.spec.ts src/components/account/__tests__/CreateAccountModal.grok.spec.ts src/components/account/__tests__/EditAccountModal.spec.ts src/i18n/__tests__/openaiFastPolicyLocales.spec.ts src/views/admin/settings/__tests__/OpenAIFastPolicyUserSelector.spec.ts` | 0 | 5 个文件、`56/56` 通过。 |
| alpha-search 补充 | `go -C backend test ./internal/service -run '^(TestForwardAlphaSearchOAuthPreservesWire|TestForwardAlphaSearchAPIKeyMapsModelAndPassesThroughError|TestForwardAlphaSearchReturnsFailoverBeforeWriting)$' -count=1` | 0 | 3 个命名目标通过。 |

### 静态、人工结论与放行

| 命令 | 退出码 | 结果 |
| --- | ---: | --- |
| `git diff --check` | 0 | 无 whitespace 错误；仅用户既有 Comet progress 的 LF/CRLF 警告。 |
| `git diff --name-only --diff-filter=U` | 0 | 无输出。 |
| `rg -n '^(<<<<<<<|=======|>>>>>>>)' backend frontend docs README.md` | 0（未匹配归一化） | 无冲突标记。 |
| `git ls-files -u` | 0 | 无输出。 |

- 人工矩阵结论：14 个受影响 M-ID 均 PASS，M-03/M-08 N/A；15 项冲突融合及 alpha-search 没有未解释回归。Ent/Wire/migration、版本/依赖、用户资源/前端跨层契约沿用 Task 6 的人工结论并由本次命令复验。
- 非阻塞警告：Browserslist/caniuse-lite 数据已 7 个月过期；Vite 报告动态/静态 import 混用和 `AccountsView` chunk 超过 500 kB。Vitest 的预期错误路径日志、`router-link` 和 intlify 警告不影响全部断言通过。
- 工作树保留用户既有 `openspec/changes/staged-merge-upstream-v0-1-156/.comet/subagent-progress.md` 修改和未跟踪 `.comet/current-change.json`；未暂存它们。本报告为唯一提交文件。
- **Task 8：允许。** v0.1.152 阶段门禁已关闭；本结论不授权 merge `v0.1.153` 之外的 tag、`upstream/main`，也不授权 push/release/deploy。

## Task 9 / v0.1.153 Capability Review (OpenSpec 3.2)

### Scope and Method

- Review target: `v0.1.152..v0.1.153`; `git diff --name-only v0.1.152..v0.1.153` exited `0` and listed `97` paths.
- Structural review used CodeGraph once for context, once for exploration, and one impact query: `NewPaymentHandler` resolves to user/admin handler constructors; the user-facing constructor's impact has the Wire provider and hidden-purchase tests. No additional broad explore was used.
- The current user-facing `NewPaymentHandler` is a two-argument constructor. The merge removed its `ChannelService` argument, while `ProvidePaymentHandler` and two hidden-purchase fixtures still supplied a third argument.
- The 16-row matrix comes from `docs/superpowers/reports/2026-07-16-staged-merge-upstream-v0-1-156-validation.md` under `Task 3 capability matrix`. `0 tests to run` results are never counted as coverage.

### Matrix Intersection

| M-ID | v0.1.153 intersection | Review mode | Result |
| --- | --- | --- | --- |
| M-01 Scheduler | yes: scheduler/cache and gateway selection paths | automatic | 2 named tests PASS. |
| M-02 OpenAI Sticky | yes: OpenAI WS ingress/pool paths | automatic | 4 named unit tests PASS. |
| M-03 Gemini/Anthropic Sticky | yes: Gemini handler/helper paths | automatic | 4 named unit tests PASS. |
| M-04 fallback / WaitPlan | yes: failover loop and gateway paths | automatic | 2 named tests PASS. |
| M-05 DB recheck | no direct matrix-key-file intersection | manual N/A | no v0.1.153 DB-recheck change to validate. |
| M-06 protocol conversion | yes: apicompat bridge/converter files | automatic | 2 apicompat and 2 tagged service tests PASS. |
| M-07 terminal usage | yes: gateway usage/failover handlers | automatic | 4 named tests PASS. |
| M-08 content moderation | no direct matrix-key-file intersection | manual N/A | no v0.1.153 content-moderation change to validate. |
| M-09 image capability | no direct matrix-key-file intersection | manual N/A | no v0.1.153 image-capability change to validate. |
| M-10 runtime settings | yes: runtime config path | automatic | 2 backend tests PASS; frontend runtime settings tests also PASS under M-13. |
| M-11 request-body replay | yes: gateway forwarding/replay paths | automatic | 5 named tests PASS. |
| M-12 user resource control | no direct matrix-key-file intersection | manual N/A | no v0.1.153 user-resource-control change to validate. |
| M-13 frontend local features | yes: account/table/swipe/date/key-usage UI paths | automatic | matrix suite 21 tests and changed-file supplement 58 tests PASS. |
| M-14 version/dependencies | yes: `VERSION` and dependency metadata | manual | merge retains `0.1.151.2`; compile command has 0 tests and is not counted as coverage. |
| M-15 Ent/Wire/migrations | yes: Wire generation and migration 174 | automatic + manual | 3 named migration tests PASS; generate completed with no generated diff. |
| M-16 first Token | no: implementation unchanged by this merge | protected / N/A | implementation remains present and protected through v0.1.156; this task found no incompatible semantics requiring a user decision. |

Counts: `affected=11`, `manual N/A=4`, `protected/N/A=1`.

### Early Compile Repair

- Accepted early work: `07eba46c6` removes the stale OpenAI test import; `94c2c3fb7` records that initial repair.
- RED: `go -C backend test ./internal/handler -run '^$' -count=1` exited `1`. It reported three stale three-argument calls: `wire.go:94` and `payment_handler_hidden_purchase_test.go:19,36`.
- GREEN: after removing only the obsolete third argument from those callers, the same compile command exited `0`; `go -C backend test ./internal/handler -run '^(TestPaymentHandlerGetCheckoutInfoRejectsHiddenPurchasePage|TestPaymentHandlerHiddenPurchasePageAllowsAdmin)$' -count=1` exited `0`.
- Fix commit: `de17e7a67 fix: align payment handler construction`. Production constructor behavior was not changed. No subsequent release behavior failure was found, so no additional TDD cycle or regression-fix commit was required.

### Commands and Evidence

| Command group | Exit | Actual coverage/result |
| --- | ---: | --- |
| `git diff --name-only v0.1.152..v0.1.153` | 0 | 97 changed paths. |
| M-01 service scheduler regex | 0 | 2 RUN/PASS. |
| M-02 tagged service sticky regex | 0 | 4 RUN/PASS. |
| M-03 tagged handler/service sticky regex | 0 | 4 RUN/PASS. |
| M-04 handler failover regex | 0 | 2 RUN/PASS. |
| M-06 untagged apicompat/service regex | 0 | apicompat 2 RUN/PASS; service reported `no tests to run`. |
| M-06 tagged service supplement | 0 | 2 RUN/PASS. |
| M-07 apicompat/handler usage regex | 0 | 4 RUN/PASS. |
| M-10 service settings regex | 0 | 2 RUN/PASS. |
| M-11 handler/service request-body regex | 0 | 5 RUN/PASS. |
| M-13 frontend matrix suite | 0 | 3 files, 21 tests PASS. |
| M-13 changed-file frontend supplement | 0 | 5 files, 58 tests PASS. |
| M-14 `go -C backend test ./cmd/server -run '^$' -count=1` | 0 | compile only; `no tests to run`. |
| M-15 migration regex including `TestLatestAPIKeyIPIndexMigration` | 0 | 3 RUN/PASS. |
| M-15 `make -C backend generate` then `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` | 0 | Ent/Wire generation stable; no diff. |
| conflict: OpenAI error handling regex | 0 | 3 RUN/PASS. |
| conflict: routes alpha-search regex | 0 | 2 RUN/PASS. |
| conflict: fastpath and endpoint/usage regex | 0 | 4 RUN/PASS. |
| conflict: tagged no-account classification regex | 0 | 2 RUN/PASS. |
| conflict: tagged Gemini fallback/retry regex | 0 | 1 RUN/PASS. |
| static: `git diff --check`; conflict-index checks | 0 | no whitespace or unresolved-index output. |

Literal command ledger (all commands below exited `0` unless explicitly marked):

```text
git diff --name-only v0.1.152..v0.1.153                         # 0, 97 paths
go -C backend test ./internal/handler -run '^$' -count=1        # 1 RED, then 0 GREEN
go -C backend test ./internal/handler -run '^(TestPaymentHandlerGetCheckoutInfoRejectsHiddenPurchasePage|TestPaymentHandlerHiddenPurchasePageAllowsAdmin)$' -count=1
go -C backend test -v ./internal/service -run '^(TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount|TestOpenAISelectAccountWithLoadAwareness_AllFullWaitPlan)$' -count=1
go -C backend test -v -tags unit ./internal/service -run '^(TestOpenAIGatewayService_SelectAccountWithScheduler_PreviousResponseStickyDisabledBypassesStickyLookupAndBind|TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDisabledBypassesLookupBindAndRefresh|TestOpenAIGatewayService_StickyDisabledWSv2SkipsStateStore|TestOpenAIGatewayService_StickyDisabledIngressSkipsStateStore)$' -count=1
go -C backend test -v -tags unit ./internal/handler ./internal/service -run '^(TestGatewayHandler_GeminiRouteStickyLookupUsesGeminiToggleNotAnthropicToggle|TestGatewayHandler_GeminiRouteStickyBindUsesGeminiToggleNotAnthropicToggle|TestGatewayHandler_OpenAIRouteStickyUsesOpenAIToggleNotAnthropicToggle|TestAntigravityGatewayServiceClearStickySessionSkipsDisabledSticky)$' -count=1
go -C backend test -v ./internal/handler -run '^(TestHandleFailoverError_IntegrationScenario|TestStreamWrittenGuard_MessagesPath_AbortFailoverOnSSEContentWritten)$' -count=1
go -C backend test -v ./internal/pkg/apicompat ./internal/service -run '^(TestChatCompletionsToResponses_ToolCalls|TestResponsesToAnthropic_CustomToolPreservesSchemaParameters|TestGatewayService_ForwardAsChatCompletions_PassthroughBodyMapCopiesFromOriginalCCBody|TestGatewayService_ForwardAsResponses_PassthroughBodyMapCopiesFromOriginalResponsesBody)$' -count=1
go -C backend test -v -tags unit ./internal/service -run '^(TestGatewayService_ForwardAsChatCompletions_PassthroughBodyMapCopiesFromOriginalCCBody|TestGatewayService_ForwardAsResponses_PassthroughBodyMapCopiesFromOriginalResponsesBody)$' -count=1
go -C backend test -v ./internal/pkg/apicompat ./internal/handler -run '^(TestResponsesEventToChatChunks_TopLevelTerminalUsage|TestResponsesEventToAnthropicEvents_TopLevelTerminalUsage|TestOpenAIGatewayHandler_NativeNonPassthroughResponsesFailedIsNotDuplicated|TestOpenAIGatewayHandler_ResponsesPartialFailoverCreatesExactlyOneFailedUsage)$' -count=1
go -C backend test -v ./internal/service -run '^(TestSettingService_SetGatewayRuntimeSettings_PersistsUpdatesCfgAndInvalidatesOnResponseHeaderTimeoutChange|TestSettingServiceGatewayRuntimeSettings_RejectsNegativeFirstTokenTimeouts)$' -count=1
go -C backend test -v ./internal/handler ./internal/service -run '^(TestGatewayHandler_MessagesAndResponsesReplayLargeBodiesAcrossFailover|TestOpenAIGatewayHandler_ChatReplayRawSpoolAcrossFailoverWhenResponsesUnsupported|TestRequestBodyCoordinator_CleanupRemovesRawEffectiveAndMultipartTemps|TestOpenAIForwardCleansBoundRequestBodyHandlesInParallel|TestRequestBodyHandle_CleanupRemovesSpoolFile)$' -count=1
pnpm --dir frontend test:run src/router/__tests__/feature-access.spec.ts src/utils/__tests__/userUiVisibility.spec.ts src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts
go -C backend test ./cmd/server -run '^$' -count=1             # 0, no tests to run
go -C backend test -v ./migrations -run '^(TestMigration173AllowsCyberBlockedUsageRequestType|TestMigration158BackfillsGrokMediaGenerationGroups|TestLatestAPIKeyIPIndexMigration)$' -count=1
make -C backend generate; git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
pnpm --dir frontend test:run src/components/account/__tests__/credentialsBuilder.spec.ts src/components/common/__tests__/DataTable.spec.ts src/composables/__tests__/useSwipeSelect.spec.ts src/utils/__tests__/formatDateLocalInput.spec.ts src/views/__tests__/KeyUsageView.spec.ts
go -C backend test -v ./internal/handler -run '^(TestOpenAIHandleStreamingAwareError_JSONEscaping|TestOpenAIHandleStreamingAwareError_NonStreaming|TestOpenAIEnsureForwardErrorResponse_WritesFallbackWhenNotWritten)$' -count=1
go -C backend test -v ./internal/server/routes -run '^(TestGatewayRoutesOpenAIAlphaSearchPathsAreRegistered|TestGatewayRoutesAlphaSearchRejectsNonOpenAIGroup)$' -count=1
go -C backend test -v ./internal/handler -run '^(TestConcurrencyHelper_TryAcquireUserSlot|TestConcurrencyHelper_TryAcquireAccountSlot_NotAcquired|TestResolveOpenAIUpstreamEndpointPrefersForwardResult|TestOpenAIGatewayHandler_ChatCompletionsUsageTaskUsesCapturedEndpointAndSnapshot)$' -count=1
go -C backend test -v -tags unit ./internal/handler -run '^(TestClassifyNoAccountError_ModelNotSupported_Returns404|TestClassifyOpenAICompatibleNoAccountError_GrokUsesGrokPlatform)$' -count=1
go -C backend test -v -tags unit ./internal/handler -run '^TestGatewayHandlerMessages_ClaudeCodeFallbackMixedAntigravitySmartRetryClearsResolvedGeminiStickySession$' -count=1
git diff --check; git diff --name-only --diff-filter=U; git ls-files -u
```

The untagged no-account classification invocation initially matched no tagged tests while other handler tests ran. It is excluded from coverage; the follow-up `-tags unit` command above supplied the two actual RUN/PASS results.

### Nine Conflict Entries

| Entry | Conclusion |
| --- | --- |
| `.gitignore` | manual: merge unignores `deploy/tests/**`; fixture is not ignored. Tracked `CLAUDE.md` remains tracked. |
| `VERSION` | manual: both merge parents/current resolve to `0.1.151.2`; M-14 compile is supplementary only. |
| `wire_gen.go` | automatic: compile GREEN and M-15 regenerate/diff clean. |
| `gateway_handler_chat_completions.go` | automatic: endpoint/usage target PASS. |
| `gateway_handler_responses.go` | automatic: M-07 failed/partial usage targets PASS. |
| `gateway_helper_fastpath_test.go` | automatic: both concurrency helper targets PASS. |
| `gemini_v1beta_handler.go` | automatic: tagged Gemini fallback smart-retry target PASS. |
| `openai_gateway_handler_test.go` | automatic: stale-import compile GREEN and 3 OpenAI error targets PASS. |
| `routes/gateway.go` | automatic: 2 alpha-search route targets PASS. |

### Risk and Boundary

- First Token implementation remains unchanged and protected through v0.1.156; removal remains limited to that separately authorized stage.
- This task found no incompatible coexistence semantics requiring a user decision.
- Frontend test commands emitted the existing non-blocking Browserslist/caniuse-lite staleness warning.
- This task did not run the Task 10 full gate and did not merge v0.1.155, `upstream/main`, push, release, or deploy.

## Task 10 / v0.1.153 Stage Gate (OpenSpec 3.3)

### Status

- Result: `PASS`; all v0.1.153 gate commands and required manual conclusions passed. Task 11 is released.
- Scope: re-ran the full root gate, frontend production build, Ent/Wire generation with restricted diff, all 11 affected Task 9 M-ID commands with their required tags, the PaymentHandler early-fix checks, nine conflict entries, and static scans.
- `git diff --name-only v0.1.152..v0.1.153` exited `0` and listed `97` paths. The review set remains `affected=11`, `manual N/A=4`, and `protected/N/A=1`.

### Full Gate And Generation

| Command | Exit | Result |
| --- | ---: | --- |
| `make test` | 0 | Backend default and `unit` tests, `golangci-lint`, frontend ESLint/typecheck and Vitest all passed; Vitest: `173` files, `1288` tests. |
| `pnpm --dir frontend run build` | 0 | `vue-tsc -b` and Vite production build passed; `968` modules, `33.84s`. |
| `make -C backend generate` | 0 | Ent and Wire generation completed. |
| `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` | 0 | stdout empty; no generated diff. |

### Task 9 Affected M-ID Evidence

All named automatic targets below were observed as RUN/PASS. A `0 tests to run` compile result is recorded as manual evidence only and is not counted as coverage.

| M-ID | Command result | Actual coverage/manual conclusion |
| --- | --- | --- |
| M-01 | service scheduler regex, exit `0` | 2 named tests PASS. |
| M-02 | `-tags unit` service sticky regex, exit `0` | 4 named tests PASS. |
| M-03 | `-tags unit` handler/service sticky regex, exit `0` | 4 named tests PASS. |
| M-04 | handler failover regex, exit `0` | 2 named tests PASS. |
| M-06 | untagged apicompat/service plus `-tags unit` service supplement, exit `0`/`0` | 2 apicompat plus 2 tagged service tests PASS; untagged service package had `no tests to run` and was excluded from coverage. |
| M-07 | apicompat/handler usage regex, exit `0` | 4 named tests PASS. |
| M-10 | service runtime-settings regex, exit `0` | 2 named tests PASS. |
| M-11 | handler/service request-body regex, exit `0` | 5 named tests PASS. |
| M-13 | frontend matrix and changed-file supplements, exit `0`/`0` | `21 + 58 = 79` tests PASS. |
| M-14 | `go -C backend test ./cmd/server -run '^$' -count=1`, exit `0` | Compile-only `no tests to run`; version remains manual. |
| M-15 | migration regex, generate, restricted diff, all exit `0` | 3 named migration tests PASS; Ent/Wire generation stable. |

Literal commands executed for the 11 affected M-IDs:

```text
go -C backend test -v ./internal/service -run '^(TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount|TestOpenAISelectAccountWithLoadAwareness_AllFullWaitPlan)$' -count=1
go -C backend test -v -tags unit ./internal/service -run '^(TestOpenAIGatewayService_SelectAccountWithScheduler_PreviousResponseStickyDisabledBypassesStickyLookupAndBind|TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDisabledBypassesLookupBindAndRefresh|TestOpenAIGatewayService_StickyDisabledWSv2SkipsStateStore|TestOpenAIGatewayService_StickyDisabledIngressSkipsStateStore)$' -count=1
go -C backend test -v -tags unit ./internal/handler ./internal/service -run '^(TestGatewayHandler_GeminiRouteStickyLookupUsesGeminiToggleNotAnthropicToggle|TestGatewayHandler_GeminiRouteStickyBindUsesGeminiToggleNotAnthropicToggle|TestGatewayHandler_OpenAIRouteStickyUsesOpenAIToggleNotAnthropicToggle|TestAntigravityGatewayServiceClearStickySessionSkipsDisabledSticky)$' -count=1
go -C backend test -v ./internal/handler -run '^(TestHandleFailoverError_IntegrationScenario|TestStreamWrittenGuard_MessagesPath_AbortFailoverOnSSEContentWritten)$' -count=1
go -C backend test -v ./internal/pkg/apicompat ./internal/service -run '^(TestChatCompletionsToResponses_ToolCalls|TestResponsesToAnthropic_CustomToolPreservesSchemaParameters|TestGatewayService_ForwardAsChatCompletions_PassthroughBodyMapCopiesFromOriginalCCBody|TestGatewayService_ForwardAsResponses_PassthroughBodyMapCopiesFromOriginalResponsesBody)$' -count=1
go -C backend test -v -tags unit ./internal/service -run '^(TestGatewayService_ForwardAsChatCompletions_PassthroughBodyMapCopiesFromOriginalCCBody|TestGatewayService_ForwardAsResponses_PassthroughBodyMapCopiesFromOriginalResponsesBody)$' -count=1
go -C backend test -v ./internal/pkg/apicompat ./internal/handler -run '^(TestResponsesEventToChatChunks_TopLevelTerminalUsage|TestResponsesEventToAnthropicEvents_TopLevelTerminalUsage|TestOpenAIGatewayHandler_NativeNonPassthroughResponsesFailedIsNotDuplicated|TestOpenAIGatewayHandler_ResponsesPartialFailoverCreatesExactlyOneFailedUsage)$' -count=1
go -C backend test -v ./internal/service -run '^(TestSettingService_SetGatewayRuntimeSettings_PersistsUpdatesCfgAndInvalidatesOnResponseHeaderTimeoutChange|TestSettingServiceGatewayRuntimeSettings_RejectsNegativeFirstTokenTimeouts)$' -count=1
go -C backend test -v ./internal/handler ./internal/service -run '^(TestGatewayHandler_MessagesAndResponsesReplayLargeBodiesAcrossFailover|TestOpenAIGatewayHandler_ChatReplayRawSpoolAcrossFailoverWhenResponsesUnsupported|TestRequestBodyCoordinator_CleanupRemovesRawEffectiveAndMultipartTemps|TestOpenAIForwardCleansBoundRequestBodyHandlesInParallel|TestRequestBodyHandle_CleanupRemovesSpoolFile)$' -count=1
pnpm --dir frontend test:run src/router/__tests__/feature-access.spec.ts src/utils/__tests__/userUiVisibility.spec.ts src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts
go -C backend test ./cmd/server -run '^$' -count=1
go -C backend test -v ./migrations -run '^(TestMigration173AllowsCyberBlockedUsageRequestType|TestMigration158BackfillsGrokMediaGenerationGroups|TestLatestAPIKeyIPIndexMigration)$' -count=1
pnpm --dir frontend test:run src/components/account/__tests__/credentialsBuilder.spec.ts src/components/common/__tests__/DataTable.spec.ts src/composables/__tests__/useSwipeSelect.spec.ts src/utils/__tests__/formatDateLocalInput.spec.ts src/views/__tests__/KeyUsageView.spec.ts
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
```

### PaymentHandler And Conflict Entries

- PaymentHandler early fix: `go -C backend test ./internal/handler -run '^$' -count=1` exited `0`; the compile-only result is not coverage. `go -C backend test -v ./internal/handler -run '^(TestPaymentHandlerGetCheckoutInfoRejectsHiddenPurchasePage|TestPaymentHandlerHiddenPurchasePageAllowsAdmin)$' -count=1` exited `0` with 2 named tests PASS.
- Nine entries: 2 manual and 7 automatic all PASS. Manual `.gitignore` retains `!deploy/tests/` and `!deploy/tests/**`; `CLAUDE.md` remains tracked. Manual VERSION check showed first parent `0.1.151.2`, tag second parent `0.1.152`, and merge result `0.1.151.2`, preserving the approved local four-part version.
- Automatic entry commands all exited `0`: OpenAI error handling 3 named tests, alpha-search routes 2, fastpath/endpoint/usage 4, tagged no-account classification 2, and tagged Gemini fallback/retry 1. `wire_gen.go` is covered by the PaymentHandler compile GREEN plus M-15 generation/diff; `gateway_handler_responses.go` is covered by M-07's 4 named usage tests.

```text
go -C backend test -v ./internal/handler -run '^(TestOpenAIHandleStreamingAwareError_JSONEscaping|TestOpenAIHandleStreamingAwareError_NonStreaming|TestOpenAIEnsureForwardErrorResponse_WritesFallbackWhenNotWritten)$' -count=1
go -C backend test -v ./internal/server/routes -run '^(TestGatewayRoutesOpenAIAlphaSearchPathsAreRegistered|TestGatewayRoutesAlphaSearchRejectsNonOpenAIGroup)$' -count=1
go -C backend test -v ./internal/handler -run '^(TestConcurrencyHelper_TryAcquireUserSlot|TestConcurrencyHelper_TryAcquireAccountSlot_NotAcquired|TestResolveOpenAIUpstreamEndpointPrefersForwardResult|TestOpenAIGatewayHandler_ChatCompletionsUsageTaskUsesCapturedEndpointAndSnapshot)$' -count=1
go -C backend test -v -tags unit ./internal/handler -run '^(TestClassifyNoAccountError_ModelNotSupported_Returns404|TestClassifyOpenAICompatibleNoAccountError_GrokUsesGrokPlatform)$' -count=1
go -C backend test -v -tags unit ./internal/handler -run '^TestGatewayHandlerMessages_ClaudeCodeFallbackMixedAntigravitySmartRetryClearsResolvedGeminiStickySession$' -count=1
```

### Static And Release Decision

| Command | Exit | Result |
| --- | ---: | --- |
| `git diff --check` | 0 | stdout empty. |
| `git diff --cached --check` | 0 | stdout empty. |
| `git diff --name-only --diff-filter=U` | 0 | stdout empty. |
| `git ls-files -u` | 0 | stdout empty. |
| `git grep -n -E '^(<<<<<<< |>>>>>>> |=======$)' -- backend frontend` | 1 | stdout empty；`git grep` 以 `1` 表示无匹配，这是本检查的预期 PASS 结果。review follow-up 于同一 HEAD 重跑并打印 `GIT_GREP_EXIT=1`。 |
| `git status --short` | 0 | only pre-existing untracked `.comet/current-change.json`. |

- Warnings: Browserslist/caniuse-lite is 7 months stale; Vitest retains expected error-path logs plus `router-link` and intlify warnings; Vite reports dynamic/static import mixing and an `AccountsView` chunk of `638.19 kB` over the `500 kB` warning threshold. All are non-blocking and pre-existing.
- Boundary: no source, test, generated, plan, OpenSpec, Comet progress, current-change, merge, push, release, or deploy changes were made. Only this canonical report is to be committed with `docs: close v0.1.153 stage gate`.
- Review follow-up: reviewer 指出原报告将无匹配的 `git grep` 退出码误记为 `0`；已按原命令重跑并记录真实退出码 `1`。后端 `make build` 不在 Task 10 既定门禁内；计划步骤 1 明确要求 `make test`、`pnpm --dir frontend run build`、受影响测试和冲突验证，以上均已执行。
- Decision: corrected static-scan evidence remains PASS. Task 11 is released. No Task 10 blocker remains.

## Task 11 / v0.1.155 Merge Ledger (OpenSpec 4.1)

### Status And Topology

- Result: `DONE` for the required v0.1.155 merge and conflict fusion. The merge commit is not rewritten; the remaining builder-call compatibility work is explicitly deferred to Task 12.
- Merge commit: `347ad61301c989a6ea53ba2539022513879aceab` `merge: upstream v0.1.155`.
- First parent: `2ec17fce2f923d237336b9ebb56793553d5cf5d8`; second parent: `41cec0db059ffb82d0efdcfcf07a24ab51fbfe97`.
- Target verification before merge: annotated `v0.1.155` object `ec4a37da4f023fbaa4d46d2ee46a6e7f22e313d4`; `v0.1.155^{}` exactly `41cec0db059ffb82d0efdcfcf07a24ab51fbfe97`. The peel is a result ancestor; `upstream/main` is not.
- `git diff --name-only HEAD^1 HEAD` reports `234` changed files. This includes gateway, scheduler, account settings, frontend, Ent/Wire, migrations, monitoring and upstream tests; it is the merge tree, not a post-merge compatibility commit.

### Conflict Ledger

| Path | Ours | Theirs | Caller / entry | Merge conclusion |
| --- | --- | --- | --- | --- |
| `backend/cmd/server/VERSION` | Approved four-part local version. | Upstream release version. | Server build/version embedding and release packaging read this file. | Retained `0.1.151.2`; release reconciliation remains final-stage work. |
| `backend/cmd/server/wire_gen.go` | Local dependency injections. | Grok quota injection. | `cmd/server` bootstrap invokes the generated `initializeApplication` graph. | Regenerated from the merged provider graph. |
| `backend/ent/mutation.go` | Usage-detail generated API. | Long-context generated API. | Ent update/create builders call these generated mutation setters. | Regenerated from merged schema; both API sets exist. |
| `backend/internal/config/config_test.go` | First Token defaults/negative validation. | Image JSON keepalive environment assertion. | `go test ./internal/config` enters config environment loading/default validation. | Both test groups retained. |
| `backend/internal/handler/openai_images.go` | Exact failed-usage endpoint/write tracking. | Non-stream JSON keepalive. | OpenAI image generation/edit route handlers call this forwarding path. | Both behaviors retained. |
| `backend/internal/repository/usage_log_repo_query.go` | Usage-detail list projection. | Long-context billing projection. | Admin/user usage-log list services call repository paged/detail query methods. | Both select columns and scan values retained. |
| `backend/internal/service/admin_account.go` | Shadow/type invariants and passthrough normalization. | Long-context flag validation/defaulting. | Admin account create/update service paths normalize and validate `extra`. | Both normalizers and invariants retained. |
| `backend/internal/service/openai_gateway_chat_completions_test.go` | OAuth session isolation and model mapping assertion. | Unknown model without Messages dispatch assertion. | `go test ./internal/service` exercises chat-completions gateway dispatch. | Local broad test body retained; upstream production dispatch remains merged. |
| `backend/internal/service/openai_gateway_record_usage_test.go` | Cyber quota test stub. | Account repository test stub. | `go test ./internal/service` constructs usage-recording gateway fixtures. | Both stubs retained. |
| `backend/internal/service/openai_oauth_passthrough_test.go` | Existing passthrough regression body. | Namespace imports for upstream additions. | `go test ./internal/service` exercises OAuth passthrough request construction. | Compiling local body retained; namespace production path remains merged. |
| `backend/internal/service/scheduler_snapshot_service.go` | OpenAI account-change callback. | Coalesced full rebuild. | Startup/periodic rebuilds and account-change notifications enter snapshot rebuild scheduling. | Both retained; startup and periodic rebuilds use the coalescer. |
| `backend/internal/service/wire.go` | Existing providers. | Usage-log-aware Grok quota provider. | Wire's `service.ProviderSet` feeds the server bootstrap dependency graph. | Provider added and duplicate direct constructor removed. |
| `frontend/src/components/account/EditAccountModal.vue` | Aggregated local extra writers/probe controls. | Long-context billing toggle. | Admin account list opens this modal; submit emits normalized account update payload. | Both retained; Spark shadow neither renders nor submits the flag. |
| `frontend/src/components/account/__tests__/CreateAccountModal.spec.ts` | Broad local create-modal suite. | Codex import/long-context fixture. | Vitest mounts CreateAccountModal and drives create/import interactions. | Local suite retained; upstream Create modal production behavior remains merged. |

### Focused Review

- Gateway: Images keeps exact failed-usage accounting while enabling JSON keepalive; passthrough namespace restoration retains the required `bytes` comparison.
- Scheduler: local OpenAI account-change propagation coexists with upstream full-rebuild coalescing and its state mutexes.
- Settings: OpenAI account long-context flag now validates as boolean, defaults to false for OpenAI, preserves an omitted current value on update, and leaves non-OpenAI extra unchanged.
- Frontend: Edit modal restores/saves the same flag for OAuth, SetupToken and API-key accounts; shadow accounts are excluded. The inherited local create/edit test suites pass.
- Generated artifacts: Ent regeneration combines UsageLog detail and long-context fields; Wire regeneration combines local services with the usage-log-aware Grok quota provider.
- First Token: protected. `backend/internal/service/openai_first_token_timeout.go` is unchanged in `HEAD^1..HEAD`; text/image defaults and negative-value validation remain in `config.go` and its focused tests.

### Commands And Results

| Command | Exit | Result |
| --- | ---: | --- |
| `make -C backend generate` | 0 | Ent and Wire generation completed after schema/provider fusion. |
| Config first-Token/keepalive exact tests | 0 | 4 named tests PASS. |
| `pnpm exec vitest run ...CreateAccountModal.spec.ts ...EditAccountModal.spec.ts` | 0 | 2 files, 69 tests PASS. |
| `git diff --cached --check` before merge commit | 0 | No whitespace errors. |
| `git diff --check HEAD^1 HEAD` / `git show --check HEAD` | 0 | No whitespace errors. |
| unmerged path/index scan and marker scan | 0 / expected grep 1 | Empty output; no unresolved path or marker. |
| focused service/handler/repository packages | 1 | Blocked by the Task 12 compatibility item below. |

### Boundary And Warning

- Task 12 handoff: `backend/internal/service/openai_image_generation_controls_test.go:150` calls `buildUpstreamRequestOpenAIPassthrough` with the old five-argument contract, while v0.1.155 now requires original/effective body and metadata arguments. No argument mapping was guessed or repaired in this merge-only task.
- Review follow-up: the 14 conflict rows now record their concrete caller or execution entry. Task 12 will link the builder compatibility repair and focused backend results back to this ledger.
- Browser test output still warns that `caniuse-lite` is stale. No v0.1.156 or `upstream/main` merge, push, release, or deploy occurred.
- The detailed ignored scratch ledger is `.superpowers/sdd/staged-merge-upstream-v0-1-156-task-11-report.md`; this canonical section fully records its topology, conflict, category, verification, first-Token, and boundary evidence.

## Task 12 / v0.1.155 Capability Review (OpenSpec 4.2)

### Scope, Handoff, And Counts

- Task 11 handoff resolved: `openai_image_generation_controls_test.go:150` used the obsolete five-argument `buildUpstreamRequestOpenAIPassthrough` call. The new signature requires source body, effective body and body-handle/token arguments. The fixture has no transformation, so source/effective body are the same bytes.
- `git diff --name-only v0.1.153..v0.1.155` lists `238` release-diff paths. The Task 11 merge-tree comparison `347ad613^1..347ad613` has `234` changed paths. Both counts are retained because the 234 value is not the release-diff cardinality.
- Matrix intersection: `affected=13` (M-01, M-02, M-04, M-05, M-06, M-07, M-08, M-09, M-10, M-11, M-13, M-14, M-15); `manual/protected N/A=3` (M-03, M-12, M-16). A `0 tests to run` result is compile/manual evidence only and never test coverage.

### RED/GREEN And Compatibility Repairs

| Finding | RED | GREEN | Commit |
| --- | --- | --- | --- |
| Task 11 builder fixture | `go -C backend test ./internal/service -run '^$' -count=1` failed: old five arguments, new signature needs source/effective body plus token/handle. | Same compile command passed; `TestOpenAIBuildUpstreamRequestOpenAIPassthroughForwardsResponsesLiteHeader` passed. | `1716639f8` `fix: preserve local behavior after v0.1.155` |
| Images JSON keepalive | Handler package failed because `openAIImagesJSONKeepaliveInterval` was called but absent after merge. | Restored the existing second-parent config-to-duration helper; 0/positive interval contract and keepalive suite passed. | `305f7ad55` `fix: restore image keepalive interval` |
| Chat model fixture | Unknown-model test expected `gpt6`, but supplied a Messages-only default mapping and produced `gpt-5.4`. | Fixture now supplies no Messages dispatch mapping; named target passed. | `4e4e7e583` `test: align chat model mapping fixture` |
| Long-context fixture | Test expected long-context multipliers while omitting v0.1.155's explicit opt-in account flag. | Fixture explicitly enables the OpenAI flag; named target passed. | `fd02f8bbf` `test: enable long-context billing fixture` |
| Usage-log SQL mock | Local mock expected 53 placeholders while merged insert has `long_context_billing_applied` and 54. | Mock includes the new column and `$54`; named target and repository package passed. | `3e3c6e71e` `test: align usage log insert mock` |
| Scheduler outbox lag | Consumed old event still triggered a lag rebuild; `FirstCreatedAtAfter` test failed. | Query only pending events after watermark and reset failures when absent; scheduler targets passed. | `806df474d` `fix: preserve scheduler outbox lag semantics` |

### M-ID Results

| M-ID | Result | Actual evidence |
| --- | --- | --- |
| M-01 Scheduler | PASS | 2 WaitPlan targets and 6 snapshot coalescer/outbox targets. |
| M-02 OpenAI Sticky | PASS | 4 `unit` tagged HTTP/WS/ingress sticky targets. |
| M-03 Gemini/Anthropic Sticky | manual N/A | No Gemini/Anthropic sticky changed-file intersection. |
| M-04 fallback / WaitPlan | PASS | 2 handler failover/SSE-written targets. |
| M-05 DB recheck | PASS | 3 scheduler DB-recheck targets. |
| M-06 protocol conversion | PASS | 2 apicompat targets plus 2 `unit` source-body passthrough targets. |
| M-07 terminal usage | PASS | 4 terminal/failed usage targets. |
| M-08 content moderation | PASS | 4 moderation targets. |
| M-09 image capability | PASS | 3 untagged image-intent/capability targets plus 1 `unit` rate-limit target. |
| M-10 runtime settings | PASS | 2 backend targets and SettingsView `14/14`. |
| M-11 request-body replay | PASS | 5 raw/effective handle replay and cleanup targets. |
| M-12 user resource control | manual N/A | No direct user-resource-control changed-file intersection. |
| M-13 frontend local features | PASS | 9 changed-file suites, `136/136` tests. |
| M-14 version/dependencies | PASS/manual | Result and first parent are `0.1.151.2`; tag parent is `0.1.153`, matching the approved local four-part version decision. Its compile-only zero-test result was excluded. |
| M-15 Ent/Wire/migrations | PASS/manual | 3 v0.1.155 migration targets passed; `make -C backend generate` and restricted generated diff are clean. |
| M-16 first Token | protected N/A | Source remains unchanged; supplemental tagged rerun passed all 10 sticky/first-Token targets, including HTTP 504/single failed usage and WS cancel-drain-turn. |

### Focused Verification And Manual Review

- Gateway/handler/service/repository: `go -C backend test ./internal/service ./internal/handler ./internal/repository -count=1` passed after all repairs.
- Settings/long-context/Ent/Wire/migration: account flag validation/default/preservation, admin boundary validation, usage-log projection/insert, migration 175/176 assertions, and generated graph were reviewed. The only local SQL mock not updated by the merge is repaired above.
- Scheduler: OpenAI account-change callback and coalesced rebuild coexist; lag logic now checks only pending outbox rows after a successful watermark.
- Frontend: account, usage, Grok/monitor, ops and runtime-settings entry suites passed (`136/136`); Spark shadow remains excluded from the long-context control path.
- Named test executions: `232` total (`187` M-ID matrix executions plus `45` builder/conflict/first-Token supplemental executions). This counts intentional reruns as executions, not unique test definitions.
- Static checks passed: `git diff --check`, cached whitespace check, unmerged-path scan and `git ls-files -u` were empty. Marker `git grep` returned expected no-match exit `1`.

### Boundary, Warning, And Task 13 Entry

- Existing non-blocking warning: Browserslist/caniuse-lite data is stale. Expected test-path logging remains non-failing.
- No merge commit was rewritten. No v0.1.156 or `upstream/main` merge, push, release, deploy, plan, OpenSpec, Comet, or current-change modification occurred.
- **Task 13 entry:** run `go test ./... -count=1` from `backend`, frontend `pnpm test:run` and `pnpm typecheck`, then its controlled 5/10/12 MB request-body matrix and affected-package rerun.

### Reviewer Follow-Up: Reproducible Evidence Ledger

- Historical RED below is evidence observed during the original Task 12 run. It is not recreated after a repair, so its command, exit and observed count are recorded rather than represented as a fresh failure.

| Repair | Historical RED command and observation | GREEN command, exit and named count | v0.1.153..v0.1.155 upstream source |
| --- | --- | --- | --- |
| Builder fixture | `go -C backend test ./internal/service -run '^$' -count=1`; exit `1`, compile failed before test execution on the five-argument call. | Same compile command exit `0` (0 tests, excluded); `go -C backend test -v ./internal/service -run '^TestOpenAIBuildUpstreamRequestOpenAIPassthroughForwardsResponsesLiteHeader$' -count=1`, exit `0`, 1. | `252ef8b73` `backend/internal/service/openai_gateway_passthrough.go`; `2f715baf0` `backend/internal/service/openai_image_generation_controls_test.go`. |
| Images keepalive helper | `go -C backend test -v ./internal/handler -run '^TestOpenAIImagesJSONKeepaliveInterval$' -count=1`; exit `1`, missing helper at handler and new contract call sites. | Same command exit `0`, 1; `go -C backend test -v ./internal/service -run '^TestOpenAIImagesJSONKeepalive' -count=1`, exit `0`, 8. | `002c0b9fd` `config.go`, `openai_images.go`, `openai_images_json_keepalive.go` and tests. |
| Chat mapping fixture | Combined command `go -C backend test -v ./internal/service -run '^(TestForwardAsChatCompletions_OAuthPromptCacheKeyKeepsAPIKeyIsolatedSessionID|TestForwardAsChatCompletions_UnknownModelDoesNotUseDefaultMappedModel|TestOpenAIGatewayServiceRecordUsage_Gpt54LongContextBillsWholeSession|TestOpenAIGatewayService_OAuthPassthrough_StreamKeepsToolNameAndBodyNormalized)$' -count=1`; exit `1`, 4 named targets ran, Chat and long-context targets failed. | `go -C backend test -v ./internal/service -run '^TestForwardAsChatCompletions_UnknownModelDoesNotUseDefaultMappedModel$' -count=1`, exit `0`, 1. | `40ec74b9f` `openai_gateway_chat_completions_test.go`, `openai_model_mapping.go`. |
| Long-context fixture | Same combined historical command; exit `1`, the long-context target lacked the new account opt-in. | `go -C backend test -v ./internal/service -run '^TestOpenAIGatewayServiceRecordUsage_Gpt54LongContextBillsWholeSession$' -count=1`, exit `0`, 1. | `e9fb5983c` account default-off behavior and fixture; `92dcfb5eb` `account.go`, `openai_gateway_usage.go`, `openai_gateway_record_usage_test.go`. |
| Usage-log SQL mock | `go -C backend test ./internal/service ./internal/handler ./internal/repository -count=1`; exit `1`, named mock `TestUsageLogRepositoryCreateSingle_SkipsDetailPersistenceWhenDisabled` expected 53 rather than 54 placeholders. Package command was not verbose, so it has no named-count claim. | `go -C backend test -v ./internal/repository -run '^TestUsageLogRepositoryCreateSingle_SkipsDetailPersistenceWhenDisabled$' -count=1`, exit `0`, 1. | `92dcfb5eb` `usage_log_repo_insert.go`, `usage_log_repo_query.go` and generated UsageLog fields. |
| Scheduler lag | `go -C backend test -v ./internal/service -run '^TestSchedulerSnapshotServicePollOutboxDoesNotUseConsumedEventForLag$' -count=1`; exit `1`, 1 target used the consumed event and did not query after watermark. | Same command exit `0`, 1. Reviewer positive command below adds the pending-event threshold case. | `a0778e9a4` `scheduler_outbox.go`, `scheduler_snapshot_service.go`, `scheduler_snapshot_outbox_cleanup_test.go`, `scheduler_outbox_repo.go`. |

#### Original 232 Execution Ledger

The original Task 12 total counts final named command results, not unique test definitions. Compile-only zero-test commands are excluded. A target intentionally re-run by a later command is counted again as an execution only in that later command's row.

| Command group | Named executions |
| --- | ---: |
| M-01, M-02, M-04, M-05, M-06, M-07, M-08, M-09, M-10, M-11, M-13, M-15 | `2 + 4 + 2 + 3 + 4 + 4 + 4 + 4 + 16 + 5 + 136 + 3 = 187` |
| Builder header; image interval/keepalive; config; long-context service/admin; scheduler regression/coalescer/outbox; SQL mock; chat/usage/OAuth; M-16 | `1 + 9 + 4 + 9 + 7 + 1 + 4 + 10 = 45` |
| Original Task 12 ledger | `187 + 45 = 232` |

#### Pending-Lag Positive Test

- `901523953` adds `TestSchedulerSnapshotServicePollOutboxRebuildsForPendingLagAfterWatermark` and a one-line source reason comment. With 401 old events, first poll commits watermark `200` and records one lag failure for pending event `201`; second poll commits `400`, sees pending `401`, reaches threshold `2`, and enters exactly one full rebuild.
- Exact reviewer command: `go -C backend test -v ./internal/service -run '^(TestSchedulerSnapshotServicePollOutboxCleansConsumedRowsAfterWatermark|TestSchedulerSnapshotServicePollOutboxDoesNotUseConsumedEventForLag|TestSchedulerSnapshotServicePollOutboxRebuildsForPendingLagAfterWatermark)$' -count=1`; exit `0`, 3 named executions.
- These are command executions, not new unique definitions: the first two were already in the 45-row scheduler accounting and are rerun deliberately with the new third target. Reviewer cumulative execution count is therefore `232 + 3 = 235`.

## Task 13 / v0.1.155 Stage Gate (OpenSpec 4.3)

### Four Attempts And Repairs

- Attempt one stopped at `golangci-lint`: `347ad613` had retained `openAIRecordUsageAccountRepoStub` while omitting its only v0.1.155 consumer. `1138be1d9 test: restore shadow parent billing coverage` restored that upstream parent opt-out/opt-in test; the named test and service lint passed.
- Attempt two exposed unit-only merge omissions. `3fdedb4d1 test: align usage list query mock` synchronized the 56-column usage scan SQL mock. `a7515cbb0 fix: restore long-context account validation` restores key-present-only `UpdateAccountExtra` validation, target-aware bulk validation, and parent-effective (`false` when missing) spark-shadow persistence. Their named repository, service, non-OpenAI allowance and shadow subtests passed.
- Attempt three added reviewer-requested coverage: `f33ad07bb test: cover shadow billing opt-in inheritance` extends `TestCreateShadowInheritsParentEffectiveOpenAILongContextBillingValue` with explicit parent opt-in. The tagged target passed with all three inheritance subtests.
- This section records only attempt four, a fresh serialized full run after those three repair/coverage attempts. Earlier partial-gate outcomes are not final evidence.

### Full Gate

| Command | Exit | Result |
| --- | ---: | --- |
| `make test` | `0` | Backend default, lint and unit tests plus frontend ESLint/typecheck/Vitest passed. Vitest: `179` files, `1337` tests. |
| `pnpm --dir frontend run build` | `0` | `vue-tsc -b` and Vite production build passed; `970` modules, `built in 33.91s`. |
| `make -C backend generate` (two runs) | `0` / `0` | Ent and Wire generation completed twice. |
| `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` | `0` | Empty restricted diff after each generation run. |

### Affected Capabilities And Conflicts

| Evidence | Command | Exit | Result |
| --- | --- | ---: | --- |
| M-01/M-02/M-04/M-05 | Targeted scheduler, sticky, failover and DB-recheck regexes | `0` | `2 + 4 + 2 + 3` named tests passed. |
| M-06/M-07/M-08/M-09 | Conversion, usage, moderation and image-capability regexes | `0` | `4 + 4 + 4 + 4` named tests passed. Untagged M-06 service output `0 tests to run`; its two tagged service targets supplied coverage. |
| M-10/M-11/M-13 | Runtime settings, body replay/cleanup and frontend stable suites | `0` | `16 + 5 + 21` named tests passed; the full Vitest gate also passed. |
| M-14/M-15 | `go -C backend test ./cmd/server -run '^$' -count=1`; three migration targets; two generate/diff runs | `0` | M-14 compiled with `0 tests to run` and remains manual; M-15 ran three migration tests and generated output remained stable. |
| Request-body matrix | `go -C backend test -v ./internal/handler -run '^TestRequestBodySizeMatrix$' -count=1` | `0` | All `9` identity/gzip/multipart x `5/10/12MB` cases passed with matching client/upstream SHA-256, bounded snapshots, spool thresholds and cleanup. |
| Task11 conflict entries | Targeted endpoint/usage, alpha-route, no-account, Gemini fallback, OAuth passthrough, account modal and locale suites | `0` | All `14` ledger entries remain closed. OAuth passthrough ran `15` current named targets; Create/Edit/locales ran `75` current tests. |
| Follow-ups | Builder; image JSON interval/keepalive; scheduler outbox; usage-list mock; long-context/shadow inheritance | `0` | `1`; `1 + 8`; `3`; `2`; and `6` top-level long-context tests passed. Shadow inheritance has `3` subtests, including explicit parent opt-in. |

### Static, Manual, And Release Decision

- `git diff --check`, `git diff --cached --check`, `git diff --name-only --diff-filter=U`, and `git ls-files -u` were clean. Exact marker scan `git grep -n -E '^(<<<<<<< |=======$|>>>>>>> )' -- backend frontend docs README.md` returned its expected no-match exit `1`.
- Non-blocking warnings remain: Browserslist/caniuse-lite staleness; Vite dynamic/static import notices and `AccountsView` `649.25 kB`; expected Vitest error-path, router-link and intlify logs. `git diff --check` also printed a CRLF warning for the unrelated pre-existing `openspec/.../.comet/subagent-progress.md` modification, with no whitespace error.
- No plan, OpenSpec, Comet/current-change, merge, push, release, or deploy operation occurred. **Task 14 is authorized.**

## Task 14 v0.1.156 Merge Ledger

- Immutable merge: `94a681bbdad61f2d0bef089e14ed214c83f411da` `merge: upstream v0.1.156`; parents `d4820cd1b8952bd1ff61d61110055c614b680eba` and `12f991dde8a58e183d4bd16a87ef6fd0df714757` (`v0.1.156^{}`). The only merge command was `git merge --no-ff v0.1.156 -m "merge: upstream v0.1.156"`; `upstream/main` was not merged.
- Conflict ledger: VERSION/Wire/generated graph, Agent Identity account flows, account runtime block, WS v2 relay and scheduler lifecycle used tag-side resolution. Local handler and regression test retained first-token behavior. Config and forwarder coexist: local `openai_*_first_token_timeout` plus upstream `openai_*_first_output_timeout_seconds`, high-effort override, header/semantic first-output guard, failover, and `HandleStreamTimeout`.
- Callers: `Config.Load`/`Validate`; `OpenAIGatewayService.Forward` and `handleStreamingResponseWithReasoning`; `Responses`/`ResponsesWebSocket`; passthrough relay; scheduler outbox. The detailed ours/theirs/caller ledger is retained in ignored Task14 scratch evidence.
- First-token remains in backend settings, DTO/API/UI, watchdog, HTTP/WS paths, and tests: exact `backend frontend deploy` scan found `138` matches. It is not deleted in this merge; Task15 alone owns deletion.
- First-output remains default-disabled with high-effort override, `first_output_timeout`, failover and `HandleStreamTimeout`: exact `backend deploy frontend` scan found `22` matches. Client WebSocket first-message timeout remains present. The approved upstream WebSocket first-output watchdog removal is not treated as a regression.
- Verification: merge subject/parents correct; `git diff --name-only --diff-filter=U` and `git ls-files -u` empty; exact marker scan empty; merge changed `250` paths. `git diff --check HEAD^1..HEAD` reports one inherited tag warning: `backend/internal/service/openai_gateway_grok_test.go:2159: new blank line at EOF`. No Task15 deletion or follow-up compatibility repair was made.

## Task 17 Early Compatibility Review

- Scope: `94a681bbd` remains immutable. Task17 repairs merge-side API drift only; no merge rewrite, Task15 deletion, OpenSpec/Comet/current-change edit, push, release, or deploy occurred.
- RED: `go test -gcflags=-e ./internal/service ./internal/handler -run '^$' -count=1` exposed missing scheduler hook/passthrough contracts, handler failover helpers and OAuth-429 state, agent-identity passthrough authentication, plus stale test fixtures. The two packages now compile with `go test ./internal/service ./internal/handler -run '^$' -count=1` exit `0`.
- Repairs: `3d0c8eb24` restored config validation; `7364aba7e` restored the authoritative successful-rebuild scheduler hook; `a8eded776` restored the 10-argument passthrough image-intent contract; `da4a24d7c` restored handler failover compatibility; `773731199` restored passthrough authentication/header compatibility; `f683caec8` and `bd800bcca` aligned stale service and handler fixtures to current interfaces; `9d0000963` honors configured WebSocket ingress timeout.
- Focused GREEN: scheduler/passthrough/image-intent/first-token and fixture targets passed under `go test ./internal/service -run <target-regex> -count=1`; handler compatibility targets passed under `go test ./internal/handler -run <target-regex> -count=1`.
- WebSocket TDD: the retained real upgraded-connection test with `client_first_message_timeout_seconds=1` first RED at the client's three-second deadline while handler code used `30*time.Second`; after using `ResolveOpenAIWSClientFirstMessageTimeout(h.cfg)` for both `WithTimeout` and logging, it GREENed in about one second.
- Task19 handoff: `backend/cmd/server/VERSION` remains temporary `0.1.155`; the highest upstream tag is `v0.1.156`. Task19 owns the version decision and any change. The inherited EOF warning remains `backend/internal/service/openai_gateway_grok_test.go:2159: new blank line at EOF` for follow-up.

## Task 14 Re-review: Responses First-Output Failover

- RED: `TestOpenAIGatewayHandler_ResponsesFirstOutputTimeoutFailsOverAfterKeepalive` configured a real two-second native first-output timeout. Account 1 emitted only `:\n\n` keepalive, returned `UpstreamFailoverError{SafeToFailoverAfterWrite:true}` with `first_output_timeout`, and current `Responses` stopped at account 1 instead of replaying to account 2.
- Fix: `0502b26d1` `fix: restore first-output failover path` routes the production Responses branch through `openAIForwardMayFailover`, checks client replay eligibility, clears stale response headers after a safe written keepalive, and limits first-output account switches with `openAIFirstOutputFailoverExhausted`. Existing first-token handling, same-account retry, and OAuth-429 state remain intact.
- GREEN: the new complete handler regression replays account 2, preserves the keepalive, hides account 1's staged `response.created`, and terminates with account 2's `response.completed`. Focused handler helper/replay-cancellation tests, `TestOpenAINativeFirstOutputFailoverKeepsAttemptHeadersPrivateAfterKeepaliveCommit`, and `go test ./internal/handler -run '^$' -count=1` passed.
- Correction: the ordinary Responses first-output failover follow-up is complete; Task14's immutable merge and the Task15/VERSION/EOF handoffs remain unchanged.

## Task 15 Local First-Token Removal

- Approved removal: deleted local HTTP SSE and upstream WebSocket first-token watchdogs, text/image timeout settings, runtime persistence/DTO/API/UI, dedicated errors, failed usage, Ops/log records, specialized tests, and the obsolete verification report. No compatibility key or alias remains.
- Retained: native HTTP `first_output_timeout` stays default-off with the high-effort override, failover, and `HandleStreamTimeout`; `gateway.openai_ws.client_first_message_timeout_seconds` and existing WebSocket read/write timeouts remain unchanged.
- Baseline: `go -C backend test ./... -run 'First(Output|Token)' -count=1` was blocked before deletion by unrelated generated-Wire/API drift and existing local first-token assertions. Post-removal targeted service/handler first-output and client-first-message tests pass; frontend typecheck and gateway-runtime Vitest pass.
- Static proof: `rg -n 'openai_text_first_token_timeout|openai_image_first_token_timeout|first_token_timeout|gateway\.openai_first_token_timeout|OpenAIFirstTokenTimeout|openAIFirstTokenWatchdog|openAIFirstToken|FirstTokenTimeout|ResponsesFirstToken|ResponsesEventEndsFirstTokenWait|ResponsesEventRecordsFirstToken'` returns no match when this canonical history report and OpenSpec history are excluded.
