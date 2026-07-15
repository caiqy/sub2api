---
change: restore-local-test-gates
design-doc: docs/superpowers/specs/2026-07-15-restore-local-test-gates-design.md
base-ref: ddefbbffa13569f973aee4bb2802eb2414c7d70f
---

# 恢复本地测试门禁实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use `subagent-driven-development` (recommended) or `executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 使不依赖 Docker 的 `make test` 串行通过后端默认测试、后端 unit 测试和 lint，以及前端 ESLint、TypeScript 与全量 Vitest。

**Architecture:** 先以测试语义、测试夹具和确定性资源隔离恢复 unit 基线；Images 只修复候选账号耗尽时的最终网关错误写入。随后按已有 linter 类别逐项消除已记录的 47 个诊断，最后将根 `Makefile` 聚合既有后端和前端命令；不修改 lint 规则、Docker、integration/e2e 或公开契约。

**Tech Stack:** Go 1.25、Gin、Testify、golangci-lint v2、GNU Make、pnpm、Vue 3、Vitest。

## 全局约束

- 仅以 `openspec/changes/restore-local-test-gates/specs/local-test-gates/spec.md` 为验收标准，并遵循设计文档的范围和回退边界。
- 不修改 `backend/.golangci.yml`，不新增依赖，不添加 build tag、ignore 或缩小 lint 检查范围。
- 不添加 Docker、integration 或 e2e 命令；根 `test` 不得引用 `test-integration`、`test-e2e` 或 Docker。
- header 断言必须保留 API key、`anthropic-version` 与 `Content-Type` 的值检查，只消除名称大小写/序列化形式依赖。
- Images 只调整账号全部失败后的最终错误写入；不得改变选择顺序、重试、计费或对外成功响应路径。
- spool 先隔离后复现；隔离后无法稳定复现时，只提交测试隔离，不改生产生命周期。
- lint 修改如涉及资源所有权、错误返回或并发，先补最小回归测试；删除无效赋值、未使用代码和等价静态修复只复用现有测试。

## 文件变更图

| 文件 | 责任 |
| --- | --- |
| `backend/internal/handler/gateway_failed_usage_unit_test.go` | 以大小写无关的 header 语义断言验证 Anthropic failed-usage 快照。 |
| `backend/internal/handler/openai_images_failover_test.go` | 固化 Images 两账号 failover 耗尽后的状态、错误体和尝试次数。 |
| `backend/internal/handler/openai_images.go` | 在 Images 候选账号耗尽时使用已有最终错误写入路径。 |
| `backend/internal/server/api_contract_test.go` | 补全 server package 私有 `stubUserRepo` 的当前接口实现。 |
| `backend/internal/server/middleware/admin_auth_test.go` | 补全 middleware package 私有 `stubUserRepo` 的当前接口实现。 |
| `backend/internal/service/openai_gateway_service_test.go` | 将 OpenAI HTTP Do 错误的 spool 清理断言限制到该测试绑定的 `t.TempDir()`。 |
| `backend/internal/{handler,pkg,repository,service}/...` | 仅修复本计划记录的 47 个 golangci-lint 诊断。 |
| `Makefile` | 聚合 backend 默认测试、backend unit 与完整 frontend 本地验证。 |

### Task 1: 将 failed-usage header 断言改为协议语义

**Files:**
- Modify: `backend/internal/handler/gateway_failed_usage_unit_test.go`
- Test: `backend/internal/handler/gateway_failed_usage_unit_test.go`

**Interfaces:**
- Consumes: `middleware.UsageDetailSnapshot.UpstreamRequestHeaders string`。
- Produces: 测试本地 helper `requireSerializedHeader(t *testing.T, raw, name, want string)`；以 `strings.EqualFold` 匹配 header 名称并精确匹配值。

- [ ] **Step 1: 写入失败的语义断言**

在同一测试文件新增下列 helper，并将第 138、252、364 行的文本包含断言替换为三个语义断言：`X-Api-Key: anthropic-test-key`、`anthropic-version: 2023-06-01`、`Content-Type: application/json`。第 248、249 行改为断言已记录的 `:status: 429` 与响应体错误码，不要求快照包含未被当前捕获器保存的上游响应 header。

```go
func requireSerializedHeader(t *testing.T, raw, name, want string) {
	t.Helper()
	for _, line := range strings.Split(raw, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if ok && strings.EqualFold(strings.TrimSpace(key), name) {
			require.Equal(t, want, strings.TrimSpace(value))
			return
		}
	}
	t.Fatalf("header %q not found in %q", name, raw)
}
```

- [ ] **Step 2: 运行测试确认当前基线失败**

Run: `go test -tags=unit ./internal/handler -run 'TestGatewayHandler_Messages(ForwardErrorStillCreatesUsageLog|FailoverExhaustedStillCreatesUsageLog|SelectionExhaustedAfterFailoverStillCreatesUsageLog)' -count=1 -v`

Expected: FAIL；当前快照将 `x-api-key` 序列化为 `X-Api-Key`，并且 `ResponseHeaders` 仅包含 `:status: 429`。

- [ ] **Step 3: 实施最小测试修复**

保留现有请求、上游 stub 和 usage-log 断言；只引入 Step 1 的 helper 并替换四处字符串格式断言。不要修改 `gateway_handler.go`、上游 header 写入或快照生产逻辑。

- [ ] **Step 4: 运行聚焦测试确认通过**

Run: `go test -tags=unit ./internal/handler -run 'TestGatewayHandler_Messages(ForwardErrorStillCreatesUsageLog|FailoverExhaustedStillCreatesUsageLog|SelectionExhaustedAfterFailoverStillCreatesUsageLog)' -count=1`

Expected: PASS；每个用例仍验证 API key、Anthropic 版本、content type、状态/错误体和 failed usage 快照。

### Task 2: 修复 Images failover 耗尽的最终响应

**Files:**
- Modify: `backend/internal/handler/openai_images_failover_test.go`
- Modify: `backend/internal/handler/openai_images.go`
- Test: `backend/internal/handler/openai_images_failover_test.go`

**Interfaces:**
- Consumes: `(*OpenAIGatewayHandler).Images(*gin.Context)`、`(*OpenAIGatewayHandler).handleFailoverExhausted(*gin.Context, *service.UpstreamFailoverError, bool)`、`service.UpstreamFailoverError`。
- Produces: 当 `SelectAccountWithSchedulerForImages` 在至少一次 `UpstreamFailoverError` 后无候选账号时，writer 写入已有网关错误 JSON，且 handler 返回。

- [ ] **Step 1: 收紧耗尽情形的失败测试**

在 `TestOpenAIGatewayHandlerImages_ServerErrorFailsOverAndReturnsClearErrorWhenExhausted` 中，在现有两账号断言后保留并明确检查：`upstream.calls()` 恰有账号 `1`、`2` 各一次，`rec.Code == http.StatusBadGateway`，`error.type == "upstream_error"`，`error.message == "Upstream service temporarily unavailable"`，并且 `rec.Body` 非空。保留同一测试已有的第二次请求和 session 断言；它们覆盖独立的粘性会话行为，不为本次错误响应修复而删除。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test -tags=unit ./internal/handler -run '^TestOpenAIGatewayHandlerImages_ServerErrorFailsOverAndReturnsClearErrorWhenExhausted$' -count=1 -v`

Expected: FAIL；当前 `openai_images.go` 的 OAuth 上游失败分支在没有写入响应时直接 `return`，导致 recorder body 为空。

- [ ] **Step 3: 在最终分支写入已有网关错误**

在 `backend/internal/handler/openai_images.go` 的 `Images` 方法中，保留账号选择、`failedAccountIDs`、`switchCount`、usage 记录和 `handleFailoverExhausted` 的现有调用。仅将 OAuth `OpenAIImagesUpstreamError` 的无响应直接返回路径改为：若没有上游响应已写入，则归入现有 `UpstreamFailoverError` 耗尽处理，写入已有 `handleFailoverExhausted` 的错误响应并返回；响应已写入时保持现有行为。不得新增响应类型或修改 `handleFailoverExhausted` 的消息映射。

- [ ] **Step 4: 运行聚焦测试确认通过**

Run: `go test -tags=unit ./internal/handler -run '^TestOpenAIGatewayHandlerImages_ServerErrorFailsOverAndReturnsClearErrorWhenExhausted$' -count=1`

Expected: PASS；两次候选尝试后返回非空的 502 `upstream_error` JSON。

- [ ] **Step 5: 运行 handler unit package**

Run: `go test -tags=unit ./internal/handler -count=1`

Expected: PASS；不改变其他 Images 成功、部分输出或已提交响应分支。

### Task 3: 补齐 server 与 middleware 的 UserRepository test stub

**Files:**
- Modify: `backend/internal/server/api_contract_test.go`
- Modify: `backend/internal/server/middleware/admin_auth_test.go`
- Test: `backend/internal/server/api_contract_test.go`
- Test: `backend/internal/server/middleware/admin_auth_test.go`

**Interfaces:**
- Consumes: `service.UserRepository` 的 `GetBlockedGroups(context.Context, int64) ([]int64, error)`、`SetBlockedGroups(context.Context, int64, []int64) error`、`GetHiddenUIResources(context.Context, int64) (bool, []int64, error)`、`SetHiddenUIResources(context.Context, int64, bool, []string) error`。
- Produces: 两个 package 私有 `stubUserRepo` 均满足 `service.UserRepository`；默认空状态返回不带 error。

- [ ] **Step 1: 运行两个 package 确认编译失败**

Run: `go test -tags=unit ./internal/server ./internal/server/middleware -count=1`

Expected: FAIL；`internal/server/api_contract_test.go` 缺 `GetBlockedGroups`，`internal/server/middleware/admin_auth_test.go` 缺 `GetHiddenUIResources`。

- [ ] **Step 2: 先写出接口符合性测试约束**

在两处 `stubUserRepo` 声明附近加入下列编译期检查；`api_contract_test.go` 已有接口检查组时，将其保留在该组中。

```go
var _ service.UserRepository = (*stubUserRepo)(nil)
```

- [ ] **Step 3: 实施受控的空值 stub 方法**

在 `api_contract_test.go` 的 `stubUserRepo` 补齐四个当前缺失的方法：`GetBlockedGroups` 返回 `[]int64{}, nil`，`SetBlockedGroups` 返回 `nil`，`GetHiddenUIResources` 返回 `false, []int64{}, nil`，`SetHiddenUIResources` 返回 `nil`。在 `admin_auth_test.go` 只补齐缺失的 `GetHiddenUIResources` 与 `SetHiddenUIResources`；已有的 `GetBlockedGroups` 与 `SetBlockedGroups` 保持“unexpected call”语义。不得改变生产 `service.UserRepository`。

```go
func (r *stubUserRepo) GetBlockedGroups(context.Context, int64) ([]int64, error) {
	return []int64{}, nil
}

func (r *stubUserRepo) GetHiddenUIResources(context.Context, int64) (bool, []int64, error) {
	return false, []int64{}, nil
}

func (r *stubUserRepo) SetHiddenUIResources(context.Context, int64, bool, []string) error {
	return nil
}
```

- [ ] **Step 4: 运行两个 package 确认通过**

Run: `go test -tags=unit ./internal/server ./internal/server/middleware -count=1`

Expected: PASS；两个 stub 都能作为 `service.UserRepository` 注入。

### Task 4: 隔离并复现 OpenAI HTTP Do 错误的 request body spool 生命周期

**Files:**
- Modify: `backend/internal/service/openai_gateway_service_test.go`
- Conditional Modify: `backend/internal/service/openai_gateway_request_body.go`
- Conditional Modify: `backend/internal/service/request_body_handle.go`
- Test: `backend/internal/service/openai_gateway_service_test.go`

**Interfaces:**
- Consumes: `NewRequestBodyHandleFromBytes`、`RequestBodyHandleOptions{TempDir, FilePrefix, SpoolThresholdBytes}`、`BindOpenAIRequestBodyHandle`、`closeOpenAIRequestBody` 与 `CleanupRequestBodyHandle`。
- Produces: HTTP Do 错误用例只读取自身绑定 handle 的 `t.TempDir()`，并在 `Forward` 返回后断言该目录为空。

- [ ] **Step 1: 记录隔离前的 flaky 基线**

Run: `go test -tags=unit ./internal/service -run '^TestOpenAIForwardCleansRequestBodyHandleWhenHTTPDoErrors$' -count=20`

Expected: PASS；该单测通常单独通过，因此此命令只记录其最小复现基线，不将单次通过当作全套稳定性证据。

- [ ] **Step 2: 让测试绑定并观察专用 spool 目录**

在 `TestOpenAIForwardCleansRequestBodyHandleWhenHTTPDoErrors` 中创建 `spoolDir := t.TempDir()`，以阈值 `1`、该目录和 `sub2api-test-` prefix 创建 body handle，并在调用 `Forward` 前通过 `BindOpenAIRequestBodyHandle(c, handle)` 绑定它。删除 `startedAt`、`requireNoFreshOpenAISpoolFiles` 及其扫描 `os.TempDir()` 的 helper。`Forward` 返回错误且关闭 upstream request body 后，读取 `spoolDir` 并断言 entries 为空。

```go
spoolDir := t.TempDir()
handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{
	SpoolThresholdBytes: 1,
	TempDir:             spoolDir,
	FilePrefix:          "sub2api-test-",
})
require.NoError(t, err)
BindOpenAIRequestBodyHandle(c, handle)

_, err = svc.Forward(c.Request.Context(), c, account, body)
require.Error(t, err)
require.NoError(t, upstream.lastReq.Body.Close())
entries, err := os.ReadDir(spoolDir)
require.NoError(t, err)
require.Empty(t, entries)
```

- [ ] **Step 3: 验证隔离后的定向和全套矩阵**

Run: `go test -tags=unit ./internal/service -run '^TestOpenAIForwardCleansRequestBodyHandleWhenHTTPDoErrors$' -count=20`

Expected: PASS 20 次；每次只检查该测试自身的 spool 目录。

Run: `go test -tags=unit ./... -count=3`

Expected: PASS 3 次；并发 package 不会再被共享系统 temp 文件污染。

- [ ] **Step 4: 仅在隔离后仍稳定失败时修复生产所有权**

使用失败测试的专用 `spoolDir` 和 handle 路径确认拥有者，沿 `openAIRequestBodyHandleForBytes`、`closeOpenAIRequestBody` 与 `CleanupRequestBodyHandle` 追踪创建和关闭。只在拥有该路径的 cleanup 分支补充一次关闭或删除，并保留 Step 2 的目录断言作为回归测试。

Run: `go test -tags=unit ./internal/service -run '^TestOpenAIForwardCleansRequestBodyHandleWhenHTTPDoErrors$' -count=20`

Expected: PASS；若 Step 3 全套已稳定通过，则不修改 `openai_gateway_request_body.go` 或 `request_body_handle.go`。

### Task 5: 修复 depguard、errcheck 与 govet 的 19 项诊断

**Files:**
- Modify: `backend/internal/handler/page_handler_hidden_menu_test.go`
- Modify: `backend/internal/handler/payment_handler_hidden_purchase_test.go`
- Modify: `backend/internal/handler/gateway_request_body_spooling_test.go`
- Modify: `backend/internal/handler/openai_request_body_spooling_test.go`
- Modify: `backend/internal/pkg/openai/first_token_timeout.go`
- Modify: `backend/internal/service/antigravity_gateway_service_test.go`
- Modify: `backend/internal/service/gateway_anthropic_apikey_passthrough_test.go`
- Modify: `backend/internal/service/gateway_anthropic_passthrough.go`
- Modify: `backend/internal/service/openai_account_scheduler_layered_test.go`
- Modify: `backend/internal/service/openai_first_token_timeout.go`
- Modify: `backend/internal/handler/grok_media_test.go`
- Modify: `backend/internal/service/openai_ws_protocol_resolver_test.go`

**Interfaces:**
- Consumes: 当前 `backend/.golangci.yml` 的既有 `depguard`、`errcheck` 与 `govet` 规则。
- Produces: 该三类所有已记录诊断为零，不改 lint 配置。

- [ ] **Step 1: 记录此组的 RED 基线**

Run: `golangci-lint run ./...`

Expected: 非零退出；本组精确包含 `depguard` 2 项、`errcheck` 12 项、`govet` 5 项。

- [ ] **Step 2: 修复 2 项 depguard**

移除 `page_handler_hidden_menu_test.go:15` 与 `payment_handler_hidden_purchase_test.go:12` 对 `internal/repository` 的直接导入，改用这些 handler 测试包已存在的 service 层 stub 或本文件最小 package 私有 stub；不得将 repository import 加入 allowlist。

- [ ] **Step 3: 修复 12 项 errcheck**

逐项处理下列现有诊断：`gateway_request_body_spooling_test.go:568`、`openai_request_body_spooling_test.go:346` 的 `req.Body.Close`；`first_token_timeout.go:169,176,187` 的 `strings.Builder.WriteByte`；`antigravity_gateway_service_test.go:1649`、`gateway_anthropic_apikey_passthrough_test.go:520`、`openai_account_scheduler_layered_test.go:1023` 的类型断言；`gateway_anthropic_passthrough.go:620,621,622` 的 `WriteString`；`openai_first_token_timeout.go:166` 的 `body.Close`。将可返回错误的生产资源关闭合并回原错误优先级；`strings.Builder` 写入使用显式忽略返回值的现有仓库惯例；测试类型断言使用 `value, ok := ...` 并在 `!ok` 时 `t.Fatalf`。

- [ ] **Step 4: 修复 5 项 govet**

在 `grok_media_test.go:323,372` 保证每条返回路径都调用 `cancel`；在 `openai_ws_protocol_resolver_test.go:33,65,103` 不复制含 `sync.RWMutex` 的 `config.Config`，改为从构造函数创建独立 config 并复制各测试所需字段。

- [ ] **Step 5: 运行受影响 package 测试**

Run: `go test ./internal/handler ./internal/pkg/openai ./internal/service -count=1`

Expected: PASS。

- [ ] **Step 6: 运行 lint 确认本组三类清零**

Run: `golangci-lint run ./...`

Expected: 非零退出仅可保留 `ineffassign`、`staticcheck`、`unused`；不得出现 `depguard`、`errcheck` 或 `govet`。

### Task 6: 修复 ineffassign、staticcheck 与 unused 的 28 项诊断

**Files:**
- Modify: `backend/internal/handler/gateway_handler.go`
- Modify: `backend/internal/handler/gemini_v1beta_handler.go`
- Modify: `backend/internal/handler/openai_chat_completions.go`
- Modify: `backend/internal/handler/openai_images.go`
- Modify: `backend/internal/handler/openai_images_controls_test.go`
- Modify: `backend/internal/handler/gateway_request_body_spooling_test.go`
- Modify: `backend/internal/handler/handler_usage_detail_contract_test.go`
- Modify: `backend/internal/pkg/httputil/body_test.go`
- Modify: `backend/internal/pkg/openai/first_token_timeout.go`
- Modify: `backend/internal/repository/user_repo.go`
- Modify: `backend/internal/service/gateway_forward_as_responses.go`
- Modify: `backend/internal/service/gateway_request_body_handle_test.go`
- Modify: `backend/internal/service/gemini_messages_compat_service.go`
- Modify: `backend/internal/service/grok_media.go`
- Modify: `backend/internal/service/openai_account_scheduler.go`
- Modify: `backend/internal/service/openai_embeddings.go`
- Modify: `backend/internal/service/openai_gateway_chat_completions.go`
- Modify: `backend/internal/service/openai_ws_v2/passthrough_relay.go`
- Modify: `backend/internal/service/ops_upstream_context.go`
- Modify: `backend/internal/service/user_subscription_daily_quota_test.go`

**Interfaces:**
- Consumes: Task 5 后仍存在的 linter 输出。
- Produces: `golangci-lint run ./...` 以退出码 0 结束。

- [ ] **Step 1: 删除 16 项 ineffassign 的无效赋值**

删除以下赋值而不改变释放责任：`gateway_handler.go:971`、`gemini_v1beta_handler.go:365,384`、`openai_chat_completions.go:176`、`openai_images.go:154,158`、`openai_images_controls_test.go:513`、`gateway_forward_as_responses.go:131,132,137`、`gemini_messages_compat_service.go:1270,1276,1277`、`grok_media.go:408`、`openai_embeddings.go:98`、`openai_gateway_chat_completions.go:268`。

- [ ] **Step 2: 修复 7 项 staticcheck**

应用诊断给出的等价改写：`httputil/body_test.go:382` 和 `pkg/openai/first_token_timeout.go:132` 使用 tagged switch；删除 `repository/user_repo.go:1019,1092` 未使用的 `client`；在 `gateway_request_body_handle_test.go:46` 移除冗余函数类型；在 `grok_media.go:376` 使用 De Morgan 等价条件；在 `openai_ws_v2/passthrough_relay.go:672` 以具名跳转或控制流重组替换无效的内层 `break`，确保退出预期外层循环。

- [ ] **Step 3: 删除 5 项 unused 私有符号**

删除 `requestBodyStatus`、`readHandlerFunctionSource`、`(*defaultOpenAIAccountScheduler).lookupShadowParentAccount`、`setOpsUpstreamRequestBody` 与 `seedSubscriptionCache`，以及仅为这些符号保留的导入；不创建调用点。

- [ ] **Step 4: 运行涉及行为的回归测试**

Run: `go test ./internal/handler ./internal/pkg/httputil ./internal/pkg/openai ./internal/repository ./internal/service -count=1`

Expected: PASS。

- [ ] **Step 5: 运行完整 linter**

Run: `golangci-lint run ./...`

Expected: PASS，退出码 0；规则、阈值、ignore 与扫描范围均未变化。

### Task 7: 将根 Makefile 固化为完整本地门禁

**Files:**
- Modify: `Makefile`
- Read-only reference: `backend/Makefile`
- Read-only reference: `frontend/package.json`

**Interfaces:**
- Consumes: `make -C backend test`、`make -C backend test-unit`、`pnpm --dir frontend run lint:check`、`typecheck`、`test:run`。
- Produces: 根目标 `test-backend`、`test-backend-unit`、`test-frontend`、`test`；`test-frontend-critical` 保持快速命令。

- [ ] **Step 1: 写入 Makefile 目标依赖的失败检查**

先执行当前入口：

Run: `make test`

Expected: 当前 `test` 只依赖 `test-backend test-frontend`，`test-frontend` 调用 `test-frontend-critical`，且不存在 `test-backend-unit`。

- [ ] **Step 2: 最小化修改根 Makefile**

将 `.PHONY` 增加 `test-backend-unit`；添加：

```make
test-backend-unit:
	@$(MAKE) -C backend test-unit
```

将聚合目标更新为：

```make
test: test-backend test-backend-unit test-frontend
```

将 `test-frontend` 的最后一行替换为：

```make
	@pnpm --dir frontend run test:run
```

保留 `FRONTEND_CRITICAL_VITEST` 和 `test-frontend-critical` 原样；不把 integration/e2e 或 Docker 目标加入任何依赖链。

- [ ] **Step 3: 运行 Makefile dry-run 验证依赖图**

Run: `make -n test`

Expected: 依次显示 `make -C backend test`、`make -C backend test-unit`、前端 `lint:check`、`typecheck`、`test:run`；不显示 `test-integration`、`test-e2e`、`docker`。

### Task 8: 从冷缓存验证全部本地门禁并完成计划自检

**Files:**
- Modify: `docs/superpowers/plans/2026-07-15-restore-local-test-gates.md`（仅勾选实施期间完成的步骤并记录实际命令退出状态）

**Interfaces:**
- Consumes: Tasks 1-7 的代码与现有本地依赖。
- Produces: 可复现的 Docker-free 本地质量门禁证据。

- [ ] **Step 1: 清理 Go 缓存**

Run: `go clean -testcache -cache`

Expected: 退出码 0。

- [ ] **Step 2: 逐域从冷缓存运行后端与前端检查**

Run: `make -C backend test`

Expected: PASS，包含 `go test ./...` 与 `golangci-lint run ./...`。

Run: `make -C backend test-unit`

Expected: PASS，包含 `go test -tags=unit ./...`。

Run: `pnpm --dir frontend run lint:check`

Expected: PASS。

Run: `pnpm --dir frontend run typecheck`

Expected: PASS。

Run: `pnpm --dir frontend run test:run`

Expected: PASS，运行完整 Vitest 集合。

- [ ] **Step 3: 运行最终聚合门禁**

Run: `make test`

Expected: PASS，退出码 0；不要求 Docker，且不运行 integration/e2e。

- [ ] **Step 4: 实施前计划自检**

执行以下人工核对后再开始代码修改：

1. Spec 覆盖：Task 1 覆盖 header canonicalization；Tasks 2-3 覆盖 Images 耗尽响应与 fixture 接口；Task 4 覆盖 spool 单测重复、同 package 并行、全套 unit 与条件性所有权修复；Tasks 5-6 覆盖 47 项 lint；Task 7 覆盖根门禁；Task 8 覆盖冷缓存验证。
2. 占位符：关键词扫描无命中；每个步骤均给出文件、符号或可运行命令。
3. 符号/类型：所有 repository stub 方法与 `service.UserRepository` 第 86-127 行一致；Images 使用现有 `handleFailoverExhausted`；spool 使用现有 `RequestBodyHandleOptions`、`requestBodyCoordinator.Cleanup` 和 `CleanupRequestBodyHandle`。
