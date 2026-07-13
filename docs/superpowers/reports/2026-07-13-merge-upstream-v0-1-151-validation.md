# 上游 v0.1.151 合并验证记录

- Base ref: `46d92f1d75f9835539f2a86d92849604a79d2f44`
- Target tag: `v0.1.151`
- Target peel commit: `deff3123ded1d14e51df1fd1286e3d43ed9ec9bd`
- Merge branch: `feature/20260713/merge-upstream-v0-1-151`
- Merge commit: `2e3e92457b435d91d3c3a93cc120cecc8aa81cd4`

## Merge 前第一父核验

- `ORIG_HEAD`：`cd9166900b8817bf52c2a407737393d7e9f17786`，亦为当前 `HEAD`；正在进行的 merge 尚未生成 merge commit。
- 用户已决定接受该第一父。旧计划中“merge 前 HEAD 精确等于 base-ref”的预期不适用：业务代码基线仍为 `46d92f1d75f9835539f2a86d92849604a79d2f44`。
- 祖先关系已核验：`46d92f1d75f9835539f2a86d92849604a79d2f44 -> 0aa43dc9bd348858eb16d29531a61c9f2261a944 -> cd9166900b8817bf52c2a407737393d7e9f17786`。

## base-ref..ORIG_HEAD 范围核验

`git log --format='%H %s' 46d92f1d75f9835539f2a86d92849604a79d2f44..ORIG_HEAD` 仅返回两个提交：

1. `0aa43dc9bd348858eb16d29531a61c9f2261a944 docs: 记录 v0.1.151 合并基线`
2. `cd9166900b8817bf52c2a407737393d7e9f17786 chore: 完成上游合并基线检查`

`git diff --name-status 46d92f1d75f9835539f2a86d92849604a79d2f44..ORIG_HEAD` 仅列出以下新增文件：

```text
docs/superpowers/plans/2026-07-13-merge-upstream-v0-1-151.md
docs/superpowers/reports/2026-07-13-merge-upstream-v0-1-151-validation.md
openspec/changes/merge-upstream-v0-1-151/.comet/subagent-progress.md
openspec/changes/merge-upstream-v0-1-151/tasks.md
```

逐提交 `git diff-tree --no-commit-id --name-status -r` 结果与上述范围一致：仅为本 change 的验证报告、计划、OpenSpec 任务和 Comet 进度，不含 `backend/`、`frontend/`、部署或其他业务代码路径。

## Merge 结果

- 实际分支：`feature/20260713/merge-upstream-v0-1-151`。
- merge commit：`2e3e92457b435d91d3c3a93cc120cecc8aa81cd4`。
- 第一父：`cd9166900b8817bf52c2a407737393d7e9f17786`；这是用户明确接受的例外。它相对 `base-ref` 仅含本 change 的协调文档，不含业务代码。
- 实际第二父：`deff3123ded1d14e51df1fd1286e3d43ed9ec9bd`，即 annotated tag `v0.1.151` 的正确 peel commit。`c463d0c84338548f3924ec25532c8575e16fc344` 是 annotated tag object，Git commit parent 不能指向 tag object；无需 amend 或 retry。

## 44 个冲突决策

每行均记录冲突文件、两侧行为、最终融合和直接验证。`merge 暂存区检查`表示 `git diff --cached --check`；服务测试表示后文列出的完整 `go test ./internal/service -count=1`。

| 路径 | 类别 | 本地行为 | 上游行为 | 融合结论 | 验证 |
| --- | --- | --- | --- | --- | --- |
| `backend/cmd/server/VERSION` | 发布元数据 | `0.1.146.4` | `0.1.150` | 保留目标 release tree 的 `0.1.150` | merge 暂存区检查 |
| `backend/cmd/server/wire_gen.go` | Wire 生成代码 | 本地依赖图 | 上游 Batch Image 依赖图 | 合并两侧 provider 注入并保留生成结果 | handler/repository 编译 |
| `backend/ent/client.go` | Ent 生成代码 | UsageLogDetail、UserResourceOverride 客户端 | Batch Image 客户端 | 由合并 schema 生成，保留实体集合 | 服务测试 |
| `backend/ent/group.go` | Ent 生成代码 | 用户并发字段扫描 | Batch Image/视频字段扫描 | 由合并 schema 生成 | 服务测试 |
| `backend/ent/mutation.go` | Ent 生成代码 | Group mutation 本地字段 | Group mutation 上游字段 | 由合并 schema 生成 | 服务测试 |
| `backend/ent/runtime/runtime.go` | Ent 生成代码 | Group 默认值索引 | 新增 Group 字段索引 | 由合并 schema 生成 | 服务测试 |
| `backend/internal/handler/admin/setting_handler.go` | 管理设置 | 本地设置处理 | 上游按领域拆分处理 | 保留拆分后的 handler，并接入本地设置语义 | handler 编译 |
| `backend/internal/handler/admin/user_handler.go` | 管理用户 | 本地用户资源字段 | 上游用户管理变更 | 同时保留资源字段和上游操作 | handler 编译 |
| `backend/internal/handler/dto/mappers.go` | DTO 映射 | 本地 API key/资源映射 | 上游模型映射 | 合并字段映射，不丢弃任一响应字段 | handler 编译 |
| `backend/internal/handler/dto/types.go` | DTO 类型 | 本地管理类型 | 上游 Batch Image/设置类型 | 合并类型定义和可选字段 | handler 编译 |
| `backend/internal/handler/gateway_handler.go` | 网关入口 | 本地请求协调 | 上游路由拆分 | 保留协调层并接入上游入口 | handler 编译 |
| `backend/internal/handler/gateway_handler_responses.go` | Responses handler | 本地错误/用量路径 | 上游 Responses 路由 | 合并路由及错误传递 | handler 编译 |
| `backend/internal/handler/openai_chat_completions.go` | Chat handler | 本地请求体协调 | 上游 Chat 入口 | 保留协调和上游转发入口 | handler 编译 |
| `backend/internal/handler/openai_gateway_handler.go` | OpenAI handler | 本地 body 生命周期 | 上游 compact/路由逻辑 | 保留两条路径及错误映射 | handler 编译 |
| `backend/internal/handler/request_body_limit.go` | 请求体错误映射 | spool 失败映射 503 | 宽松 JSON 归一化和限制 | 合并 imports，保留两类错误路径 | handler 编译 |
| `backend/internal/pkg/httputil/body.go` | 请求体处理 | 流式解压和解压后大小限制 | BOM/控制字节归一化 | 解压读取后归一化，避免大 body 先驻留内存 | 服务测试 |
| `backend/internal/pkg/usagestats/usage_log_types.go` | 用量类型 | 本地请求类型 | 上游图像/视频类型 | 合并枚举和统计分类 | 服务测试 |
| `backend/internal/repository/usage_log_repo.go` | 用量仓储 | 本地查询/计费统计 | 上游按职责拆分 | 保留拆分后的实现和本地统计语义 | repository 编译 |
| `backend/internal/repository/user_repo.go` | 用户仓储 | 本地资源覆盖 | 上游用户操作 | 合并查询/更新字段 | repository 编译 |
| `backend/internal/server/middleware/api_key_auth_google.go` | API Key 鉴权 | 本地鉴权缓存/资源限制 | 上游 Google 鉴权 | 合并鉴权字段和缓存读取 | 服务测试 |
| `backend/internal/service/admin_service.go` | 管理服务 | 本地管理能力 | 上游按领域拆分 | 保留拆分文件及本地能力 | 服务测试 |
| `backend/internal/service/antigravity_gateway_service.go` | Antigravity 网关 | 本地请求语义 | 上游按流水线拆分 | 保留拆分入口和本地转发语义 | 服务测试 |
| `backend/internal/service/api_key_auth_cache_impl.go` | 认证缓存 | blocked groups 快照 | Batch Image/视频定价快照 | 缓存版本升至 16，保留双方字段 | 服务测试 |
| `backend/internal/service/gateway_service.go` | 通用网关 | 本地网关能力 | 上游按职责拆分 | 保留拆分后的协作入口和本地行为 | 服务测试 |
| `backend/internal/service/grok_media.go` | Grok 媒体 | 本地图像/视频计费 | 上游媒体控制 | 合并媒体意图和计费参数 | 服务测试 |
| `backend/internal/service/image_generation_intent.go` | 图像意图 | 本地来源/模型判断 | 上游图像控制 | 合并意图解析和兼容字段 | 服务测试 |
| `backend/internal/service/model_pricing_resolver_test.go` | 定价回归测试 | 本地价格断言 | 上游模型定价断言 | 保留双方测试覆盖 | 服务测试 |
| `backend/internal/service/openai_codex_transform.go` | Codex 协议转换 | 本地转换规则 | 上游 identity 修复 | 保留 identity 配对和本地转换 | 服务测试 |
| `backend/internal/service/openai_gateway_chat_completions.go` | Chat 网关 | legacy/raw usage 采集 | 上游流水线拆分 | 从最终请求 body 采集并接入拆分 sender | 服务测试、legacy/raw 聚焦测试 |
| `backend/internal/service/openai_gateway_chat_completions_raw.go` | Raw Chat | 本地 raw 兼容 | 上游请求发送拆分 | 在共享 sender 记录 final preview/attempt | 服务测试、raw 聚焦测试 |
| `backend/internal/service/openai_gateway_messages.go` | Messages 网关 | legacy body/usage 语义 | 上游流水线拆分 | 使用 source-aware builder 和最终 body 采集 | 服务测试、legacy 聚焦测试 |
| `backend/internal/service/openai_gateway_responses_chat_fallback.go` | Responses fallback | 本地 fallback | 上游 Responses 改造 | 保留 fallback 和终止错误语义 | 服务测试、sticky 聚焦测试 |
| `backend/internal/service/openai_gateway_service.go` | OpenAI 网关 | 本地完整网关 | 上游按职责拆分 | 保留拆分后的调度/响应/usage 协作 | 服务测试 |
| `backend/internal/service/openai_ws_forwarder.go` | WebSocket 转发 | 本地 HTTP bridge 状态 | 上游拆分为 ingress/support/v2 | 保留拆分架构，每 turn 重置 bridge attempt 状态 | 服务测试、bridge 聚焦测试 |
| `backend/internal/service/setting_service.go` | 运行时设置 | 本地缓存失效 | 上游按设置领域拆分 | 保留拆分服务和 partial-reset 失效 | 服务测试 |
| `backend/internal/service/subscription_service.go` | 订阅 | 本地赋值/缓存失效 | 上游订阅演进 | 赋值后同步等待缓存失效完成 | 服务测试、subscription 聚焦测试 |
| `backend/internal/service/user.go` | 用户服务 | 本地用户资源 | 上游用户字段 | 合并接口和调用方字段 | 服务测试 |
| `backend/internal/service/wire.go` | Wire 声明 | 本地 provider | 上游 Batch Image/provider | 合并 provider set | 服务测试 |
| `frontend/src/components/layout/AppHeader.vue` | 前端布局 | 本地导航/品牌项 | 上游版本/功能项 | 保留两侧可见入口 | 暂存区检查 |
| `frontend/src/components/layout/AppSidebar.vue` | 前端导航 | 本地菜单权限 | 上游 Batch Image/运维菜单 | 合并菜单与访问控制 | 暂存区检查 |
| `frontend/src/i18n/locales/en.ts` | i18n 迁移 | 聚合英文语言包 | 域模块目录 | 删除旧入口，保留域模块和键冲突测试 | 暂存区检查 |
| `frontend/src/i18n/locales/zh.ts` | i18n 迁移 | 聚合中文语言包 | 域模块目录 | 删除旧入口，保留域模块和键冲突测试 | 暂存区检查 |
| `frontend/src/views/admin/UsageView.vue` | 用量界面 | 本地统计展示 | 上游筛选/排行展示 | 合并筛选、统计和排行视图 | 暂存区检查 |
| `frontend/src/views/admin/__tests__/UsageView.spec.ts` | 前端回归测试 | 本地用量断言 | 上游视图断言 | 保留双方断言 | 暂存区检查 |

## 系统化调试

### RED

`go test ./internal/service -count=1` 初次失败于 OpenAI legacy/raw compatibility usage-preview：

- `TestOpenAILegacyChatCompletionsCapturesUpstreamRequestPreview`
- `TestOpenAIRawCompatCallersCaptureFinalPreviewAttemptAndFailoverHeaders`
- `TestOpenAILegacyMessagesCapturesUpstreamRequestPreview`
- `TestForwardAsAnthropic_CapturesFinalSessionAndTurnStateHeaders`

测试显示 collector 取得空 body 或 headers；完整服务测试还触达 API-key passthrough 映射、final ops preview、body-path error 和请求体清理。

### Root Cause 与 GREEN

上游将 `sendCCUpstreamRequest` 拆出时，没有同时迁移调用方的 usage/ops 采集。legacy Chat Completions、Messages 和通用 `Forward` 仍使用 pre-passthrough builder，既绕过 account passthrough，也让采集快照不是最终 wire body。

修复在实际 upstream attempt 前，从拥有所有权的最终 body handle 采集 preview 并标记 attempt；legacy、generic Responses 和 OAuth image 都改走既有 source-aware builder，并在尝试后关闭 owned body。相关回归一并恢复：passthrough builder 错误映射 400、禁用时跳过 previous-response sticky 查询、每 turn 重置 HTTP bridge attempt、partial reset 缓存失效和订阅赋值后的同步缓存失效等待。

## 已通过命令

- `go test ./internal/service -count=1`：PASS。
- `go test ./internal/handler ./internal/repository -run '^$'`：PASS。
- legacy/raw/final-body/cleanup/passthrough/sticky/bridge/subscription 聚焦测试：PASS。
- `git diff --cached --check`：PASS。
- `git diff --name-only --diff-filter=U`：无输出，unmerged=0。
- `git grep -n -E '^(<<<<<<< .+|=======|>>>>>>> .+)$' -- . ':!docs/superpowers/reports/**'`：无输出，真正冲突标记=0。

## 残余风险

- 本 Task 3 收口仅重跑服务测试和 handler/repository 编译检查；计划所列完整后端、前端和全仓验证尚未在本次收口执行。
- 工作树保留用户/协调流程的未暂存 `docs/superpowers/plans/**`、`openspec/**`、`.comet/**` 和 scratch 报告；它们不属于 merge commit。

## Task 3 Review Fix（第 1/2 轮）

- passthrough 在 `httpUpstream.Do` 前记录 owned final body 的 usage/ops 快照并标记 upstream attempted；非 failover HTTP 400 现在也会触发 failed-usage gating。
- `ForwardAsChatCompletions` 的 unsupported Responses fallback 已有 replay 路径：`openAIRequestBodyBytes(c, nil)` 从 `BindOpenAIRequestBodyHandle` 读取原始 CC body，覆盖 spooled handle；注释已明确这一所有权边界。
- 普通 `Forward` 的 final preview 曾被旧入站 handle 覆盖；现在仅当最终 request 不拥有 body handle 时才回退入站 preview。绑定 spooled handle、显式 `instructions` 且仅 account rule 改 body 的回归测试证明 collector 与 ops preview 等于 wire body。
- 受冲突前端文件 `UsageView.vue` 补回 `closeUsageDetailModal` 缺失的闭合；其测试补齐 fake timer 与 `listErrorLogs` mock，`CreateUserRequest` contract 补齐既有 `role` 字段。
- 验证通过：`go test ./internal/service -count=1`、`go test ./cmd/server -run '^$'`、`go generate ./ent` 后 `git diff --exit-code -- ent`、`pnpm test:run src/views/admin/__tests__/UsageView.spec.ts src/i18n/__tests__/localesNoKeyCollision.spec.ts`、`pnpm typecheck`。
- Wire 首次生成暴露 `GatewayService` 与 `BatchImageWorkerRuntime` provider 缺失；补齐既有 provider 后又发现 quota flusher 误用不启动后台任务的 `NewUserPlatformQuotaUsageFlusher`。最终 `ProviderSet` 改用 `ProvideUserPlatformQuotaUsageFlusher`，生成代码同步调用该 provider。

## Task 3 Review Fix（第 2/2 轮）

- quota flusher provider 恢复 `Start()` 启动语义；`TestProvideUserPlatformQuotaUsageFlusher_EnabledStartsFlush` 验证启用配置会实际消费并落库 quota snapshot。
- `adminServiceImpl` 恢复 `OpenAIProbeController`，`UpdateAccount` 在 probe 从开到关时删除 probe entry；仅 `layered_probe` 来源会清理临时不可调度状态和 runtime block。
- 新增 handler 级 passthrough HTTP 400 failed-usage 回归，验证真实上游错误会写入 usage detail，而不是因 attempted marker 缺失被跳过。
- 修复 unit-tag 测试桩的仓储接口漂移后，probe toggle 与 BatchImage worker runtime 聚焦测试实际执行通过。
- `go test ./internal/service -run '^(TestProvideUserPlatformQuotaUsageFlusher_EnabledStartsFlush|TestOpenAIForwardBoundInboundHandleSnapshotsFinalWireBody|TestOpenAIPassthroughCapturesAttemptBeforeNonFailoverHTTPError)$' -count=1`：PASS。
- `go test ./internal/handler -run '^(TestOpenAIGatewayHandler_PassthroughHTTP400CreatesFailedUsage|TestOpenAIGatewayHandler_UsageDetailStoresInjectedUpstreamRequestBody)$' -count=1`：PASS。
- `go test -tags unit ./internal/service -run '^(TestAdminService_.*ProbeToggle.*|TestBatchImageWorkerRuntime_.*)$' -count=1`：PASS。
- `go generate ./cmd/server` 连续两次生成的 `wire_gen.go` SHA-256 均为 `48F0188F1CEE9CD50BDD34910517834B1977FD90A447B29637E4A3EC7334E4EB`。
- `go test ./cmd/server -run '^$' -count=1`、`go test ./internal/service -count=1`：PASS。
- `pnpm test:run -- src/views/admin/__tests__/UsageView.spec.ts src/i18n/__tests__/localesNoKeyCollision.spec.ts`：2 files / 17 tests PASS。
- `pnpm typecheck`：PASS。

## 元数据、生成代码与 Migration 复核

- `backend/cmd/server/VERSION` 为 `0.1.150`，与 `git show 'v0.1.151^{}:backend/cmd/server/VERSION'` 完全一致；未擅自改写为 tag 名。
- `backend/go.mod` 将 Go toolchain 从 1.26.4 更新到 1.26.5，并更新 AWS SDK 小版本；`go.mod`/`go.sum` 均来自目标 release 与 Wire 工具的可复现依赖解析，未升级无关依赖。前端 `package.json` 与锁文件无额外合并差异。
- `go generate ./ent` 与 `go generate ./cmd/server` 完成后没有产生未提交生成差异；Wire 连续生成哈希稳定。
- 新增 migrations `159` 至 `173` 均来自目标 release。`160_add_user_frozen_balance.sql` 与 `160_batch_image_provider_refs.sql` 虽共享数字前缀，但 migration runner 使用完整 `filename` 作为主键，按完整文件名排序执行，因此两者会独立应用；两个 SQL 均使用 `ADD COLUMN IF NOT EXISTS`。

## 关键能力专项审查

### Scheduler、Sticky 与 Runtime Settings 回归

- RED：Gemini route sticky lookup/bind 在 `Gemini.Enabled=false`、`Anthropic.Enabled=true` 时仍访问缓存；admin setting 聚焦测试显示 omitted login/OAuth/risk/backend 字段被清空，sticky/WS scheduler 字段没有持久化、回读或热更新。
- 根因：上游 service/handler 拆分时，`GetGeminiCachedSessionAccountID`/`BindGeminiStickySession` 退化为无平台开关的通用 alias；本地 `UpdateSettingsRequest` pointer 字段、`SystemSettings` 映射、`buildSystemSettingsUpdates`、`parseSettings` 和 `refreshCachedSettings` 的 sticky/WS 逻辑未完整迁移。
- 修复：Gemini wrapper 恢复独立开关；设置请求恢复 omitted=保留语义；sticky/WS scheduler 字段重新接入验证、持久化、回读、响应和运行时 config 热更新。gateway-runtime handler 单测改用无启动迁移副作用的 service constructor，避免把 provider 启动迁移计入 handler 写入断言。
- GREEN：`go test ./internal/handler -run 'TestGatewayHandler_GeminiRouteSticky(Lookup|Bind)UsesGeminiToggleNotAnthropicToggle' -count=1`、admin setting 聚焦测试和 Task 5 三组能力矩阵全部通过。
- `go test ./internal/pkg/apicompat -count=1`：PASS。
- scheduler/sticky/fallback、gateway Responses/Chat/Messages/terminal usage、privacy/image/settings/reload/cache 聚焦命令：PASS。

### 大输入请求体生命周期

- RED：`go test ./internal/handler -run 'Test.*(RequestBody|BodySpool|BodyRetention|Replay|Cleanup)' -count=1` 在 12 MiB body 首次账号返回 429 后 panic；堆栈显示请求体已成功重放到 failover 路径，panic 位于可选 `RateLimitService` 副作用的 nil 调用。
- 根因与修复：拆分后的 `GatewayService.handleFailoverSideEffects` 未沿用同文件其他错误路径的 nil guard。服务未注入时直接跳过 rate-limit 副作用，不改变 body handle、重放、所有权或清理。
- GREEN：上述 handler 命令通过；`go test ./internal/service -run 'Test.*(RequestBody|CachedBody|BodyOrder)' -count=1` 通过。
- 结论：内存到磁盘切换、同字节重放、fallback/重试与成功/失败清理语义保持。
