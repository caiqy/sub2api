---
change: reduce-request-body-memory-retention
design-doc: docs/superpowers/specs/2026-08-03-reduce-request-body-memory-retention-design.md
base-ref: 8b494e1871f90b1a8559797f96f1099380a3fd4e
archived-with: 2026-08-04-reduce-request-body-memory-retention
---

# 请求体内存驻留治理实施计划

> **供执行代理使用：** 必须使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 逐任务实施；使用本文的 checkbox 跟踪进度。

**目标：** 将默认请求体 spool 阈值从 10MB 收紧到 1MB、preview 上限从 5MB 收紧到 256KB，并让所有非 WS 网关入口只在同步改写窗口或单次 attempt 内物化完整 body，消除上游等待期的完整 `[]byte` 驻留。

**架构：** 以现有 `RequestBodyHandle` 作为唯一可重放 canonical body；handler 完成请求级改写后重建 final handle，failover 每轮从 handle 短暂物化，service 内部 retry 每次改写后替换 attempt handle。session、usage、cyber 与 payload hash 在其原有语义边界上预计算为小标量，转发签名和状态机保持不变。

**技术栈：** Go、Gin、标准库 `io`/`net/http`/`runtime`、现有 `testify/require` 测试工具。

**设计依据：** [请求体内存驻留治理技术设计](../specs/2026-08-03-reduce-request-body-memory-retention-design.md)；行为契约见 `docs/openspec/changes/reduce-request-body-memory-retention/specs/request-body-retention-control/spec.md`；任务边界见 `docs/openspec/changes/reduce-request-body-memory-retention/tasks.md`。

## 全局约束

- `DefaultRequestBodySpoolThresholdBytes` 必须是导出的 `int64` 常量，值为 `1 << 20`。
- `DefaultRequestBodyPreviewLimitBytes` 必须是导出的 `int64` 常量，值为 `256 << 10`，且不得大于 spool 阈值。
- `Forward(ctx, c, account, body []byte)` 等现有对外签名保持不变；不新增依赖、配置项、数据库 schema 或兼容层。
- spool 创建、写入、关闭、打开或读取失败继续映射为 503，不得回退到 byte-owned 长驻路径。
- session hash 使用 normalize 前 raw body；Chat usage hash 使用 channel mapping 前 body；cyber key 使用 `CyberSessionBlockKey`/`explicitOpenAISessionID` 现有口径；Responses `requestPayloadHash` 使用全部请求级改写后的 final body。
- `deriveOpenAIForwardAttemptBody` 与 `passthroughFailoverState` 的调用时机和状态机不变，只替换 canonical body 来源。
- `forwardOpenAIWSV2`、WS retry/reconnect map 及其恢复语义明确不改，另开 change 处理。
- 大 body 只允许在同步解析、改写或单次 attempt 中短暂物化；调用上游 transport 后，handler/service 不再保留完整 `[]byte`。
- 每个改动任务先提交失败测试，再提交最小实现；所有命令均从 `backend/` 目录运行，除非步骤另有说明。

## 文件职责与改动地图

- `backend/internal/service/request_body_handle.go`：导出并统一 1MB/256KB 默认值，继续负责 memory/spool、preview 与清理。
- `backend/internal/service/openai_gateway_request_body.go`：OpenAI handle 默认 options 与 bytes/handle 互转入口。
- `backend/internal/handler/openai_gateway_handler.go`：Responses final handle、hash 预计算、failover 单轮物化与槽位释放。
- `backend/internal/service/openai_gateway_forward.go`：HTTP Forward 内部 retry 的 attempt handle 生命周期；WS 分支不动。
- `backend/internal/handler/openai_chat_completions.go`：Chat final handle、usage/cyber 标量预计算和 failover 单轮物化。
- `backend/internal/handler/gateway_handler.go`、`backend/internal/service/gateway_request.go`：Anthropic Messages handle-backed `ParsedRequest` 与账号级 attempt。
- `backend/internal/handler/gateway_handler_responses.go`、`backend/internal/service/antigravity_gateway_compat.go`：Anthropic Responses 兼容入口的 Antigravity body 生命周期。
- `backend/internal/handler/gemini_v1beta_handler.go`：清除 sticky/hash 计算后的 raw body 引用，保持现有 handle forward。
- `backend/internal/handler/openai_images.go`、`backend/internal/handler/grok_media.go`：验证 Images nil-forward，并补齐 Grok session/multipart 转换后的引用释放。
- 现有 `*_request_body_*_test.go`、failover/retry 测试：锁定阈值、hash、重放、503 与清理语义。
- `backend/internal/handler/request_body_memory_retention_test.go`：新增阻塞 transport + GC 的上游等待期内存验证。
- `memory/context/dmit-sub2api-memory-analysis.md`：把原“方案 3”更新为 v2 的全链路 handle 结论、preview 上限与 WS 排除项。

---

### Task 1: Spool 阈值与 preview 收紧（对应 tasks.md 第 1 组）

**目标：** 建立单一导出默认值，使 identity 请求在 `size > 1MB` 时落盘、preview 在 `size > 256KB` 时截断，并保持边界 `size == 1MB` 仍为内存模式。

**改动文件：**
- Modify: `backend/internal/service/request_body_handle.go:23-31,192-206,292-304,490-501`
- Modify: `backend/internal/service/openai_gateway_request_body.go:38-43`
- Modify: `backend/internal/handler/openai_gateway_handler.go:97-110`
- Modify: `backend/internal/handler/request_body_size_matrix_test.go:25-115`
- Modify: `backend/internal/handler/openai_gateway_handler_test.go:4655-4674`
- Modify: `backend/internal/handler/openai_gateway_request_body_retention_test.go:128-146`
- Modify: service package tests that directly reference the renamed private defaults, using the exported names without changing their assertions

**接口：**
- Produces: `service.DefaultRequestBodySpoolThresholdBytes int64`
- Produces: `service.DefaultRequestBodyPreviewLimitBytes int64`
- Consumes: existing `RequestBodyHandleOptions`; `0` 值仍由 `normalizeRequestBodyHandleOptions` 填入上述默认值。

**关键实现要点：**
- 将私有默认常量直接重命名为导出常量，不保留第二组 alias。
- `openAIResponsesRequestBodySpoolThresholdBytes` 可继续作为 handler 测试注入边界，但其值必须引用 `service.DefaultRequestBodySpoolThresholdBytes`；preview 变量引用导出默认值，以保留现有测试临时覆盖能力。
- `openAIRequestBodyHandleOptions()` 不再包含 10MB/5MB 字面量。
- 矩阵单独覆盖 `512KB`、`1MB`、`1MB+1`、`2MB`、`10MB` identity body；gzip/multipart 原有覆盖不删。

**验证方式：** 运行 request-body/preview 定向测试，并执行 `go build ./...` 与 `go vet ./...`；边界断言必须证明 `1MB` 不落盘、`1MB+1` 落盘、preview 不超过 256KB。

- [x] **Step 1: 先把默认阈值矩阵改成预期的新边界**

在 `request_body_size_matrix_test.go` 增加 production-default identity 表，并让期望只由导出常量计算：

```go
func TestRequestBodyDefaultSpoolThresholdMatrix(t *testing.T) {
	for _, tt := range []struct {
		name string
		size int
	}{
		{"identity/0.5MB", 512 << 10},
		{"identity/1MB", 1 << 20},
		{"identity/1MB+1", (1 << 20) + 1},
		{"identity/2MB", 2 << 20},
		{"identity/10MB", 10 << 20},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// 复用 testRequestBodySizeMatrixJSON；options 使用导出的 production default。
			testRequestBodySizeMatrixJSON(t, tt.size, false)
		})
	}
}
```

将 helper 的 options 和 spool 断言改为：

```go
jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{
	SpoolThresholdBytes: service.DefaultRequestBodySpoolThresholdBytes,
	PreviewLimitBytes:   service.DefaultRequestBodyPreviewLimitBytes,
	TempDir:             rawDir,
}
wantSpool := int64(size) > service.DefaultRequestBodySpoolThresholdBytes
assertMatrixTempFiles(t, rawDir, "sub2api-request-body-", wantSpool)
```

- [x] **Step 2: 运行矩阵并确认它因导出常量不存在或旧阈值而失败**

Run: `go test ./internal/handler -run 'TestRequestBody(DefaultSpoolThresholdMatrix|SizeMatrix)$' -count=1`

Expected: FAIL，首先报告 `undefined: service.DefaultRequestBodySpoolThresholdBytes`；常量补齐但值未改时，`identity/1MB+1` 的 spool 断言仍失败。

- [x] **Step 3: 导出默认常量并替换三处事实源**

`request_body_handle.go` 的常量与默认填充统一为：

```go
const (
	DefaultRequestBodySpoolThresholdBytes int64 = 1 << 20
	DefaultRequestBodyPreviewLimitBytes   int64 = 256 << 10
	defaultRequestBodySpoolPrefix               = "sub2api-request-body-"
)

if opts.SpoolThresholdBytes <= 0 {
	opts.SpoolThresholdBytes = DefaultRequestBodySpoolThresholdBytes
}
if opts.PreviewLimitBytes <= 0 {
	opts.PreviewLimitBytes = DefaultRequestBodyPreviewLimitBytes
}
```

同文件的 `RequestBodyPreviewString`、snapshot 截断与测试改用导出名称。handler/service 引用写成：

```go
const openAIResponsesRequestBodySpoolThresholdBytes int64 = service.DefaultRequestBodySpoolThresholdBytes
var openAIResponsesRequestBodyPreviewLimitBytes int64 = service.DefaultRequestBodyPreviewLimitBytes

func openAIRequestBodyHandleOptions() RequestBodyHandleOptions {
	return RequestBodyHandleOptions{
		SpoolThresholdBytes: DefaultRequestBodySpoolThresholdBytes,
		PreviewLimitBytes:   DefaultRequestBodyPreviewLimitBytes,
		FilePrefix:          defaultRequestBodySpoolPrefix,
	}
}
```

- [x] **Step 4: 同步阈值依赖测试并验证 preview 契约**

`openai_gateway_handler_test.go` 与 retention 测试使用 `service.DefaultRequestBodySpoolThresholdBytes` 构造超阈值 body；service 包测试使用 `DefaultRequestBodyPreviewLimitBytes`。新增断言：

```go
require.Equal(t, int64(1<<20), service.DefaultRequestBodySpoolThresholdBytes)
require.Equal(t, int64(256<<10), service.DefaultRequestBodyPreviewLimitBytes)
require.LessOrEqual(t, service.DefaultRequestBodyPreviewLimitBytes, service.DefaultRequestBodySpoolThresholdBytes)
```

Run: `go test ./internal/service ./internal/handler -run '(RequestBody|ResponsesSpool|Preview|SizeMatrix)' -count=1`

Expected: PASS；`1MB` 不落盘，`1MB+1`、`2MB`、`10MB` 落盘，preview 最大 256KB，handler 结束后临时文件清空。

- [x] **Step 5: 构建和静态检查**

Run: `go build ./...`

Expected: PASS，无输出。

Run: `go vet ./...`

Expected: PASS，无输出。

- [x] **Step 6: 提交本组**

```bash
git add internal/service/request_body_handle.go internal/service/openai_gateway_request_body.go internal/handler/openai_gateway_handler.go internal/handler/request_body_size_matrix_test.go internal/handler/openai_gateway_handler_test.go internal/handler/openai_gateway_request_body_retention_test.go internal/service/*_test.go
git commit -m "perf: lower request body spool threshold"
```

### Task 2: Responses 入口循环内按需物化（对应 tasks.md 第 2 组）

**目标：** 在 route model 与 reasoning policy 完成后绑定 final handle；循环外只保留 hash/key/handle，failover 每轮短暂物化 canonical body，并保证物化失败释放已获取账号槽位。

**改动文件：**
- Modify: `backend/internal/handler/openai_gateway_handler.go:336-373,413-432,547-572,648-687`
- Modify: `backend/internal/handler/openai_gateway_request_body_retention_test.go`
- Modify: `backend/internal/handler/openai_gateway_credential_failover_loop_test.go`
- Modify: `backend/internal/handler/openai_gateway_reasoning_failover_test.go`

**接口：**
- Consumes: `(*jsonRequestBodyCoordinator).SetEffectiveBytes([]byte) error`
- Consumes: `(*service.RequestBodyHandle).ReadAll() ([]byte, error)`
- Consumes: `service.BindOpenAIRequestBodyHandle(*gin.Context, *service.RequestBodyHandle)`
- Produces: handler 内局部 `finalHandle *service.RequestBodyHandle`、`sessionHash string`、`cyberBlockKeyHTTP string`、`requestPayloadHash string`；不新增跨包 API。

**关键实现要点：**
- `sessionHash` 在 `normalizeOpenAIResponsesCompactRequest` 前基于 raw `body` 计算；header 优先规则由现有 `GenerateSessionHash` 保持。
- `cyberBlockKeyHTTP` 在 raw body 尚存时无条件预计算，Forward 后仅在 cyber policy 标记存在时使用；这样不会因 policy 在 service 返回时才写入而丢 key。
- 新增 handler 内部 `rejectIfCyberSessionKeyBlocked`，接受预计算 key；保留现有 `rejectIfCyberSessionBlocked` 作为 body 到 key 的薄封装，避免改动其他调用方和测试。
- 删除 line 367 的“初次绑定即 final”假设；route model 和 reasoning policy 后再次 `SetEffectiveBytes(body)`，以该 handle 的 `Hash()` 作为 `requestPayloadHash`。
- 在进入账号选择循环前 `body = nil`；每轮账号槽位获取后读取 `finalHandle`。读取失败先调用 `accountReleaseFunc`，再按 `ErrRequestBodySpool` 映射 503。
- `deriveOpenAIForwardAttemptBody` 函数本身不改；只把新物化的 `canonicalBody` 传入。

**验证方式：** 运行 Responses retention、spooling、credential failover、reasoning failover 与 cyber 定向测试；额外验证 final handle body/hash 包含 route/reasoning 改写，物化失败恰好释放一次账号槽位。

- [x] **Step 1: 增加 final handle、hash 语义和槽位释放回归测试**

在现有 handler harness 中增加三个测试：

```go
func TestOpenAIGatewayHandler_ResponsesFinalHandleIncludesRequestLevelRewrites(t *testing.T) {
	// route 将 client-model 改为 routed-model，reasoning policy 将 xhigh 限制为 high。
	// upstream recorder 阻塞前读取 request body。
	require.Equal(t, "routed-model", gjson.GetBytes(upstreamBody, "model").String())
	require.Equal(t, "high", gjson.GetBytes(upstreamBody, "reasoning.effort").String())
	require.Equal(t, service.HashUsageRequestPayload(upstreamBody), billingCommand.RequestPayloadHash)
}

func TestOpenAIGatewayHandler_ResponsesSessionAndCyberKeysUseRawBody(t *testing.T) {
	// raw prompt_cache_key 在 compact normalize/route mapping 前存在。
	require.Equal(t, wantSessionHash, selectedSessionHash)
	require.Equal(t, service.CyberSessionBlockKey(apiKey.ID, requestContext, rawBody), recordedCyberKey)
}

func TestOpenAIGatewayHandler_ResponsesReadFailureReleasesAccountSlot(t *testing.T) {
	// 账号槽位获取后删除 spool 文件，使 finalHandle.ReadAll 返回 ErrRequestBodySpool。
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, int32(1), releaseCount.Load())
}
```

- [x] **Step 2: 运行新增测试确认旧实现失败**

Run: `go test ./internal/handler -run 'TestOpenAIGatewayHandler_Responses(FinalHandle|SessionAndCyber|ReadFailure)' -count=1`

Expected: FAIL；至少出现 final handle hash/body 仍为 request-level rewrite 前内容，或 spool read failure 后 `releaseCount == 0`。

- [x] **Step 3: 预计算 raw 语义标量并在全部请求级改写后重绑**

核心顺序固定为：

```go
sessionHash := h.gatewayService.GenerateSessionHash(c, body)
cyberBlockKeyHTTP := service.CyberSessionBlockKey(apiKey.ID, c, body)

body, ok = h.normalizeOpenAIResponsesCompactRequest(c, reqLog, body)
if !ok {
	return
}
// JSON 校验、route model、reasoning policy、审计等同步 body 消费保持原顺序。

if err := coordinator.SetEffectiveBytes(body); err != nil {
	h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
	return
}
finalHandle := coordinator.Effective()
requestPayloadHash := finalHandle.Hash()
service.BindOpenAIRequestBodyHandle(c, finalHandle)
body = nil
```

原 line 548 不再重新计算 session hash；line 549 改为 `h.rejectIfCyberSessionKeyBlocked(c, apiKey, cyberBlockKeyHTTP, reqModel, cyberBlockFormatResponses)`，不得再从 nil body 推导 key。新 helper 复用现有 runtime 开关、store 查询、错误 envelope 和 ops enqueue 代码；原 `rejectIfCyberSessionBlocked` 仅计算 `CyberSessionBlockKey` 后委托它。

- [x] **Step 4: 将 failover canonical 来源改为 final handle，并覆盖错误释放**

在账号槽位获取后、调用 `deriveOpenAIForwardAttemptBody` 前：

```go
canonicalBody, readErr := finalHandle.ReadAll()
if readErr != nil {
	if accountReleaseFunc != nil {
		accountReleaseFunc()
	}
	if errors.Is(readErr, service.ErrRequestBodySpool) {
		h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body", streamStarted)
		return
	}
	h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body", streamStarted)
	return
}
attemptBody := h.deriveOpenAIForwardAttemptBody(reqLog, canonicalBody, account, &passthroughFailoverState)
canonicalBody = nil
```

Forward 后记录 cyber policy 时直接使用预计算的 `cyberBlockKeyHTTP`；不再访问 `sessionHashBody`。

- [x] **Step 5: 运行 Responses retention、spooling 与 failover 定向测试**

Run: `go test ./internal/handler -run '(Responses.*(RequestBody|Spool|Failover|Cyber|Session|FinalHandle)|DeriveOpenAIForwardAttemptBody)' -count=1`

Expected: PASS；route/reasoning 改写进入 final handle，session/cyber/hash 口径不变，spool 失败为 503，credential/passthrough/reasoning failover 重放一致。

- [x] **Step 6: 提交本组**

```bash
git add internal/handler/openai_gateway_handler.go internal/handler/openai_gateway_request_body_retention_test.go internal/handler/openai_gateway_credential_failover_loop_test.go internal/handler/openai_gateway_reasoning_failover_test.go
git commit -m "perf: materialize responses bodies per attempt"
```

### Task 3: Service Forward retry 循环 handle 化（对应 tasks.md 第 3 组）

**目标：** HTTP Forward 的首次 attempt 直接复用 handler 传入 body；请求构造完成后只保留 current attempt handle，错误恢复、计费错误处理和 service tier 提取时按需读取，retry body 改写后立即替换 handle。

**改动文件：**
- Modify: `backend/internal/service/openai_gateway_forward.go:764-939`
- Modify: `backend/internal/service/openai_responses_rejected_field_retry_test.go`
- Modify: `backend/internal/service/openai_ws_protocol_forward_test.go`
- Modify: `backend/internal/service/error_passthrough_runtime_test.go`
- Modify: `backend/internal/service/openai_gateway_record_usage_test.go`

**接口：**
- Consumes: `openAIRequestBodyHandleForContext(c, body) (*RequestBodyHandle, bool, error)`
- Consumes: `openAIRequestBodyHandleForBytes(handle, body) (*RequestBodyHandle, bool, error)`
- Consumes: `CleanupRequestBodyHandle(handle)`，只清理由当前 Forward 新建/替换且拥有的 handle。
- Keeps: `handleErrorResponse(..., requestBody []byte, ...)` 与 `extractOpenAIServiceTierFromBody([]byte)` 签名不变，调用点按需物化。

**关键实现要点：**
- WS transport 分支在 HTTP retry loop 之前返回；不得修改 `forwardOpenAIWSV2` 或 reconnect recovery。
- 进入 HTTP loop 前用当前 request-level/account-level body 建立 `attemptHandle`；若与 gin context 已绑定 handle 内容一致则借用，否则 owns=true 并 defer cleanup。
- 首轮 loop 使用已有 `body` 构造 request，不先 `ReadAll`；`Do` 前清掉循环变量中的 body。后续 retry 才从 current attempt handle 读取。
- `buildUpstreamRequestWithSourceBody` 每轮显式接收 current `attemptHandle`，不得继续固定读取 gin context 中最初绑定的 handle。
- `invalid_encrypted_content` 和 rejected field 都从 current handle 读取 canonical，改写后用新 body 重建 handle；新 handle 成功后再清理旧 owned handle，避免重建失败导致无可清理状态。
- 仅在调用 `handleErrorResponse` 或提取 service tier 前读取；调用后立即设 nil。重建/读取失败原样返回 `ErrRequestBodySpool`。

**验证方式：** 运行 rejected-field、invalid-encrypted、error passthrough、service-tier 与 cached-body 定向测试；删除 spool 文件的 retry 用例必须返回 `ErrRequestBodySpool`，且首轮不得额外 `ReadAll`。

- [x] **Step $1: 扩展 HTTP retry 测试，要求第二次发送来自新 handle**

在 rejected-field 与 invalid-encrypted 测试中绑定一个强制落盘 handle，并断言两次 payload：

```go
handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{
	SpoolThresholdBytes: 1,
	TempDir:             t.TempDir(),
})
require.NoError(t, err)
t.Cleanup(func() { CleanupRequestBodyHandle(handle) })
BindOpenAIRequestBodyHandle(c, handle)

result, err := svc.Forward(context.Background(), c, account, body)
require.NoError(t, err)
require.Len(t, upstream.bodies, 2)
require.Equal(t, expectedFirstBody, upstream.bodies[0])
require.Equal(t, expectedRetryBody, upstream.bodies[1])
```

再让 temp file 在第一响应后被删除，断言 retry materialization 返回 `ErrRequestBodySpool`，而不是回退使用旧 `body`。

- [x] **Step 2: 运行 retry 测试确认旧实现仍可绕过 handle**

Run: `go test ./internal/service -run '(RetriesExplicitMaxOutputTokensRejection|HTTPIngressRetriesInvalidEncryptedContent|RejectedFieldRetry.*Spool)' -count=1`

Expected: 新增“删除 spool 后 retry”用例 FAIL；旧实现会直接使用常驻 `body` 继续第二次请求，未返回 `ErrRequestBodySpool`。

- [x] **Step 3: 建立并管理 current attempt handle**

在 HTTP retry loop 前使用现有 helper，保留首轮 bytes：

```go
attemptHandle, attemptHandleOwned, err := openAIRequestBodyHandleForContext(c, body)
if err != nil {
	return nil, err
}
defer func() {
	if attemptHandleOwned {
		CleanupRequestBodyHandle(attemptHandle)
	}
}()

firstAttemptBody := body
body = nil
```

每轮开始：

```go
attemptBody := firstAttemptBody
firstAttemptBody = nil
if len(attemptBody) == 0 {
	attemptBody, err = attemptHandle.ReadAll()
	if err != nil {
		return nil, err
	}
}
```

所有当前 loop 的解析、request build 和 preview 使用 `attemptBody`；`httpUpstream.Do` 前不再把它存回跨轮 `body`。

- [x] **Step 4: 把两条改写 retry 路径改为替换 handle**

两个分支统一执行以下无抽象的短序列：

```go
retryHandle, retryHandleOwned, err := openAIRequestBodyHandleForBytes(attemptHandle, retryBody)
if err != nil {
	return nil, err
}
if attemptHandleOwned && retryHandle != attemptHandle {
	CleanupRequestBodyHandle(attemptHandle)
}
if retryHandle != attemptHandle {
	attemptHandle = retryHandle
	attemptHandleOwned = retryHandleOwned
}
requestView = newOpenAIRequestView(retryBody)
reqBody = nil
retryBody = nil
continue
```

`invalid_encrypted_content` 在 marshal 后先 `rejectedFieldRetryState.remember(retryBody)` 再置 nil；rejected-field 继续先调用 `Allow(retryBody)`，状态机上限不变。

- [x] **Step 5: 错误处理与 service tier 按需读取**

错误终止路径：

```go
requestBody, readErr := attemptHandle.ReadAll()
if readErr != nil {
	return nil, readErr
}
result, err := s.handleErrorResponse(ctx, resp, c, account, requestBody, billingModel)
requestBody = nil
```

成功路径：

```go
requestBody, readErr := attemptHandle.ReadAll()
if readErr != nil {
	return nil, readErr
}
serviceTier := extractOpenAIServiceTierFromBody(requestBody)
requestBody = nil
reqBody = nil
```

如果 `handleErrorResponse` 只在 debug/cyber 分支需要 request body，可在后续性能证据明确后再下沉 lazy callback；本 change 保持签名，避免扩大改动面。

- [x] **Step 6: 运行 service 定向回归**

Run: `go test ./internal/service -run '(OpenAIResponsesRejectedField|HTTPIngressRetriesInvalidEncryptedContent|OpenAIHandleErrorResponse|ExtractOpenAIServiceTier|Forward_FailoverReparsesCachedBody)' -count=1`

Expected: PASS；retry payload 正确、次数上限不变、service tier 与错误透传不变、spool read failure 保留 sentinel。

- [x] **Step 7: 提交本组**

```bash
git add internal/service/openai_gateway_forward.go internal/service/openai_responses_rejected_field_retry_test.go internal/service/openai_ws_protocol_forward_test.go internal/service/error_passthrough_runtime_test.go internal/service/openai_gateway_record_usage_test.go
git commit -m "perf: spool openai retry bodies between attempts"
```

### Task 4: ChatCompletions 入口对齐（对应 tasks.md 第 4 组）

**目标：** Chat 在 channel mapping 前固定 usage hash 与 cyber key，绑定 effective handle 后释放 handler body，failover 每轮从 handle 读取，且读取失败可靠释放账号槽位。

**改动文件：**
- Modify: `backend/internal/handler/openai_chat_completions.go:126-177,247-271`
- Modify: `backend/internal/handler/openai_gateway_request_body_retention_test.go`
- Modify: `backend/internal/handler/openai_gateway_credential_failover_loop_test.go`
- Modify: `backend/internal/handler/openai_gateway_failed_usage_test.go`

**接口：**
- Consumes: Task 1 的导出默认值和 Task 3 保持不变的 `ForwardAsChatCompletions` 接口。
- Produces: 局部 `effectiveHandle`、`usageRequestPayloadHash`、`cyberBlockKeyChat`；不新增 API。

**关键实现要点：**
- 在 `ResolveChannelMappingAndRestrict` 之前计算 `HashUsageRequestPayload(body)`；不能对 mapped body 计算。
- `GenerateSessionHash`、`ExtractSessionID` 与 `CyberSessionBlockKey` 都在 body 尚存时完成。
- line 126 的本地 block 检查改用 Task 2 的 `rejectIfCyberSessionKeyBlocked`，后续 usage 记录复用同一个 key。
- `SetEffectiveBytes(effectiveBody)` 后绑定 `effectiveHandle` 并将 `body`、`effectiveBody` 置 nil。
- 每轮账号槽位获取后 `effectiveHandle.ReadAll()`；失败路径与 Responses 相同，先 release 后返回。

**验证方式：** 运行 Chat usage、cyber、credential failover 与 failed-usage 测试；upstream 收到 mapped model，但 usage hash 和 cyber key 必须继续对应 mapping 前 client body。

- [x] **Step 1: 增加 usage/cyber 口径和 failover replay 测试**

```go
func TestOpenAIGatewayHandler_ChatCompletionsHashesBeforeChannelMapping(t *testing.T) {
	rawBody := []byte(`{"model":"client-model","prompt_cache_key":"chat-session","messages":[{"role":"user","content":"hello"}]}`)
	wantUsageHash := service.HashUsageRequestPayload(rawBody)
	wantCyberKey := service.CyberSessionBlockKey(apiKey.ID, requestContext, rawBody)

	require.Equal(t, "mapped-model", gjson.GetBytes(upstreamBody, "model").String())
	require.Equal(t, wantUsageHash, recordedUsageHash)
	require.Equal(t, wantCyberKey, recordedCyberKey)
}
```

另加两个账号 failover 用例，断言两轮都收到 mapped effective body，且 forced spool read failure 后 release count 为 1。

- [x] **Step 2: 运行新增 Chat 测试确认 hash 或释放断言失败**

Run: `go test ./internal/handler -run 'TestOpenAIGatewayHandler_ChatCompletions(HashesBeforeChannelMapping|.*Handle|.*Release)' -count=1`

Expected: FAIL；旧实现循环内仍引用 `body/effectiveBody`，且 usage hash 在 Forward 后现场计算。

- [x] **Step 3: 预计算小标量、绑定 handle 并释放 bytes**

```go
sessionHash := h.gatewayService.GenerateSessionHash(c, body)
promptCacheKey := h.gatewayService.ExtractSessionID(c, body)
usageRequestPayloadHash := service.HashUsageRequestPayload(body)
cyberBlockKeyChat := service.CyberSessionBlockKey(apiKey.ID, c, body)
if h.rejectIfCyberSessionKeyBlocked(c, apiKey, cyberBlockKeyChat, reqModel, cyberBlockFormatChat) {
	return
}

// channel mapping 后：
if err := coordinator.SetEffectiveBytes(effectiveBody); err != nil {
	// 保留现有错误映射。
	return
}
effectiveHandle := coordinator.Effective()
service.BindOpenAIRequestBodyHandle(c, effectiveHandle)
body = nil
effectiveBody = nil
```

删除后续重复的 session/hash/key 计算。

- [x] **Step 4: failover 每轮物化并处理槽位释放**

```go
forwardBody, readErr := effectiveHandle.ReadAll()
if readErr != nil {
	if accountReleaseFunc != nil {
		accountReleaseFunc()
	}
	status := http.StatusBadRequest
	if errors.Is(readErr, service.ErrRequestBodySpool) {
		status = http.StatusServiceUnavailable
	}
	h.handleStreamingAwareError(c, status, "api_error", "Failed to spool request body", streamStarted)
	return
}
```

Forward 返回后 `recordCyberPolicyIfMarked` 使用预计算 `cyberBlockKeyChat` 与 `usageRequestPayloadHash`。

- [x] **Step 5: 运行 Chat 定向回归**

Run: `go test ./internal/handler -run '(ChatCompletions|GatewayChatCredential|Chat.*Usage|Chat.*Failover)' -count=1`

Expected: PASS；mapped upstream body 不变，usage 去重 hash 仍对应 client body，cyber session 含 header/body 原语义，失败 usage 与槽位释放正常。

- [x] **Step 6: 提交本组**

```bash
git add internal/handler/openai_chat_completions.go internal/handler/openai_gateway_request_body_retention_test.go internal/handler/openai_gateway_credential_failover_loop_test.go internal/handler/openai_gateway_failed_usage_test.go
git commit -m "perf: materialize chat bodies per attempt"
```

### Task 5: 其他入口对齐（对应 tasks.md 第 5 组）

**目标：** 清除 Anthropic Messages/Responses、Gemini 与 Grok 媒体入口剩余的跨等待期 byte-owned 引用；验证 Images 已符合 nil-forward；保持 multipart 文件 part 与 WS 行为不变。

**改动文件：**
- Modify: `backend/internal/handler/gateway_handler.go:246-336,830-1078`
- Modify: `backend/internal/handler/gateway_handler_responses.go:67-163,280-303`
- Modify: `backend/internal/service/antigravity_gateway_compat.go:99-138,165-249`
- Modify: `backend/internal/handler/gemini_v1beta_handler.go:211-269,369-405`
- Verify only unless test exposes regression: `backend/internal/handler/openai_images.go:122,344-349`
- Modify: `backend/internal/handler/grok_media.go:100-235`
- Modify: `backend/internal/handler/gateway_request_body_spooling_test.go`
- Modify: `backend/internal/handler/gemini_v1beta_handler_test.go`
- Modify: `backend/internal/handler/grok_media_test.go`
- Modify: `backend/internal/handler/openai_images_failover_test.go`
- Modify: `backend/internal/service/gateway_request_body_handle_test.go`
- Modify: `backend/internal/service/antigravity_gateway_service_test.go`

**接口：**
- Consumes: `NewRequestBodyRefFromHandle`, `ParsedRequest.CloneForHandle`, `ForwardAsResponsesHandle`, `ForwardGeminiHandle`, `ForwardNativeHandle`。
- Keeps: `RequestBodyRef.Replace([]byte)` 签名；normalized model bytes 必须在进入并发等待前被 coordinator 吸收到 handle，并重新解析为 handle-backed `ParsedRequest`。
- Produces: `func (s *AntigravityGatewayService) ForwardAsResponsesHandle(ctx context.Context, c *gin.Context, account *Account, bodyHandle *RequestBodyHandle, parsed *ParsedRequest) (*ForwardResult, error)`；旧 bytes wrapper 仅供已有同步调用兼容，handler 改用 handle 版本。

**关键实现要点：**
- Anthropic Messages：`ParseGatewayRequest` 因长上下文 model normalize 调用 `Body.Replace(normalizedBody)` 后，handler 立即 `SetEffectiveBytes(parsedReq.Body.Bytes())` 并用 handle 重新 parse；进入等待前 `body=nil`。账号级 rewrite 继续每轮建立 owned attempt handle并在 forward/panic 后清理。
- Anthropic Responses：session hash/审计完成后 `body=nil`；普通分支保持 `ForwardAsResponsesHandle`，Antigravity 分支改为 handle-backed wrapper，转换完成即清除 original body，仅保留 model/reasoning/hash 标量与转换后的 Gemini handle。
- Gemini：sticky、usage hash、thought-signature bool 计算完立即 `body=nil`；切号清理从 `effectiveBody.ReadAll()` 而非 `coordinator.ReadRaw()` 获取当前 canonical，避免恢复原始签名。
- Images：只运行现有 `ForwardImages(..., nil, ...)`、multipart failover 与 spool 清理测试，不为已满足的路径制造改动。
- Grok：在 `GenerateSessionHash`、payload hash 与 multipart JSON 二次转换完成后清掉 `body`/文本字段；继续以 bound effective handle 调用 `ForwardGrokMedia(..., nil, contentType)`。

**验证方式：** 分别运行 Messages/Responses、Gemini、Images、Grok handler 测试和 Antigravity/Gateway handle service 测试；最后以 `git diff` 明确确认 WS 文件无改动。

- [x] **Step $1: 为各入口补充“阻塞上游时只持有 handle”的回归断言**

复用现有 recorder/harness，增加或强化以下断言：

```go
require.NotNil(t, parsedReq.Body.Handle())
require.Equal(t, expectedBody, upstreamBody)
require.Empty(t, tempFilesAfterHandler)
```

不为测试新增生产观测 API；handler 包使用既有 `Handle()`、context 与 transport 断言，service 同包测试直接检查 handle。测试名称：

```text
TestGatewayHandler_MessagesNormalizedModelStaysHandleBacked
TestGatewayHandler_ResponsesAntigravityWaitsWithHandle
TestGeminiV1BetaModels_ThoughtSignatureCleanupUsesEffectiveHandle
TestGrokMedia_SessionSeedReleasedBeforeBlockedUpstream
```

- [x] **Step $1: 运行新增入口测试确认剩余 byte 路径失败**

Run: `go test ./internal/handler ./internal/service -run '(MessagesNormalizedModel|ResponsesAntigravityWaitsWithHandle|ThoughtSignatureCleanupUsesEffectiveHandle|GrokMedia_SessionSeedReleased)' -count=1`

Expected: FAIL；至少 Anthropic normalized `RequestBodyRef`、Antigravity Responses 或 Grok session seed 仍可观察到 byte-owned 生命周期。

- [x] **Step $1: 修复 Anthropic Messages 的 normalized body 所有权**

紧接首次 `ParseGatewayRequest` 后吸收 parser 产生的 owned bytes：

```go
if parsedReq.Body.Handle() == nil {
	normalizedBody := parsedReq.Body.Bytes()
	if err := coordinator.SetEffectiveBytes(normalizedBody); err != nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
		return
	}
	effectiveBody = coordinator.Effective()
	parsedReq, err = service.ParseGatewayRequest(service.NewRequestBodyRefFromHandle(effectiveBody), domain.PlatformAnthropic)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
}
body = nil
```

route model rewrite 需要 bytes 时只在同步点 `effectiveBody.ReadAll()`，改写后再次 `SetEffectiveBytes`。每轮 materialization 或 attempt handle 创建失败时，在返回前释放 `selection.ReleaseFunc/accountReleaseFunc`、queue release 与 wait counter。

- [x] **Step $1: 修复 Anthropic Responses Antigravity 分支**

在 handler 完成 session hash 后释放 `body`，两个分支都只传 handle。`AntigravityGatewayService.ForwardAsResponsesHandle` 读取、转换并建立 transformed handle 后清掉原始 bytes；`antigravityCompatRequest` 不再跨 `antigravityRetryLoop` 保存 `originalBody`，只保存已提取的 `originalModel`、`reasoningEffort` 和 usage hash 标量。`forwardAntigravityCompat` 将 transformed handle 放入现有 `antigravityRetryLoopParams.payloadHandle` 并设置 `ownedPayload: true`，不改 retry loop 参数结构。

```go
result, err = h.antigravityGatewayService.ForwardAsResponsesHandle(
	requestCtx, c, account, effectiveBody, parsedReq,
)
```

若 transformed handle 创建/打开失败，返回 `ErrRequestBodySpool`，由 handler 的现有 `writeResponsesForwardRequestBodyError` 映射 503。

- [x] **Step $1: 清理 Gemini 残留 raw 引用**

```go
requestPayloadHash := service.HashUsageRequestPayload(body)
hasThoughtSignature := bytes.Contains(body, []byte(`"thoughtSignature"`))
body = nil
```

两个 thought-signature 清理分支均改为：

```go
canonicalBody, readErr := effectiveBody.ReadAll()
if readErr != nil {
	// 保留现有 Google 503 映射，并释放已获取槽位。
	return
}
cleanedBody := service.CleanGeminiNativeThoughtSignatures(canonicalBody)
canonicalBody = nil
if err := coordinator.SetEffectiveBytes(cleanedBody); err != nil {
	googleError(c, http.StatusServiceUnavailable, "Failed to spool request body")
	return
}
effectiveBody = coordinator.Effective()
cleanedBody = nil
```

- [x] **Step $1: 验证 Images 并修复 Grok session/multipart 生命周期**

Images 不改生产代码，确认 line 349 始终传 nil。Grok 在所有 session/hash 消费完成后执行：

```go
sessionHash = h.gatewayService.GenerateSessionHash(c, sessionSeed)
sessionSeed = nil
body = nil
requestInfo.ReleaseText()
```

multipart edit 的 `PrepareGrokMediaFormForwardBody` 结果必须先 `SetEffectiveBytes`，成功后才释放 form text/value；文件 part 仍由标准库 multipart 临时文件管理。

- [x] **Step $1: 运行其他入口完整定向测试**

Run: `go test ./internal/handler -run '(GatewayHandler_(Messages|Responses)|GatewayHandler_Messages|GeminiV1Beta|OpenAIImages|GrokMedia)' -count=1`

Expected: PASS；Messages/Responses 重放与流式错误契约不变，Gemini sticky/signature 清理不变，Images 继续 nil-forward，Grok multipart 文件和 data URL 内容一致。

Run: `go test ./internal/service -run '(GatewayRequestBody|ForwardAsResponses|AntigravityGatewayService_Forward|GrokMedia)' -count=1`

Expected: PASS；所有 owned attempt/transformed handle 在成功、错误、panic 后清理。

- [x] **Step $1: 明确检查 WS 无改动并提交本组**

Run: `git diff -- internal/service/openai_ws_forwarder_v2.go internal/service/openai_ws_protocol_forward.go`

Expected: 无输出。

```bash
git add internal/handler/gateway_handler.go internal/handler/gateway_handler_responses.go internal/service/antigravity_gateway_compat.go internal/handler/gemini_v1beta_handler.go internal/handler/grok_media.go internal/handler/gateway_request_body_spooling_test.go internal/handler/gemini_v1beta_handler_test.go internal/handler/grok_media_test.go internal/handler/openai_images_failover_test.go internal/service/gateway_request_body_handle_test.go internal/service/antigravity_gateway_service_test.go
git commit -m "perf: release gateway request bytes before upstream waits"
```

### Task 6: 阻塞 transport、全量验证与文档收尾（对应 tasks.md 第 6 组）

**目标：** 用可重复测试证明 2MB/8.9MB body 在 transport 读完并阻塞时不会按 body 大小线性驻留；完成全量测试、受控边界验证和 memory/context 复盘更新。

**改动文件：**
- Create: `backend/internal/handler/request_body_memory_retention_test.go`
- Modify: `backend/internal/handler/request_body_size_matrix_test.go`（仅在 Task 1 矩阵不足以复用时补充 helper）
- Modify: `memory/context/dmit-sub2api-memory-analysis.md:39-55,65-75`

**接口：**
- Test-only transport: `Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error)`；读取并关闭 `req.Body` 后通过 channel 阻塞。
- 不新增生产接口、debug endpoint 或运行时内存开关。

**关键实现要点：**
- 请求源使用流式生成器，测试本身不得先构造并保留 2MB/8.9MB `[]byte`，否则会污染 heap 结论。
- transport 使用 `io.Copy(io.Discard, req.Body)`，不把 upstream body 存入字段；关闭 reader 后通知测试执行两次 GC。
- 先测 2MB，再测 8.9MB；比较两者阻塞期 `HeapAlloc`，允许固定框架噪声，但增长不得接近 6.9MB body 差值。preview 另以 snapshot 长度 `<=256KB` 做确定性断言。
- Windows/Linux 都使用 `t.TempDir()` 与现有 handle 清理 API，不依赖 `/tmp` 固定路径。

**验证方式：** 单独运行阻塞 transport 测试和默认阈值矩阵，再执行 `go build ./...`、`go vet ./...`、`go test ./... -count=1` 与 `git diff --check`。

- [x] **Step $1: 编写阻塞 transport 内存测试**

测试骨架使用现有 OpenAI handler env，核心 transport 与测量逻辑如下：

```go
type retentionBlockingTransport struct {
	started chan struct{}
	release chan struct{}
}

func (u *retentionBlockingTransport) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	if _, err := io.Copy(io.Discard, req.Body); err != nil {
		return nil, err
	}
	if err := req.Body.Close(); err != nil {
		return nil, err
	}
	close(u.started)
	<-u.release
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_retention","output":[],"usage":{"input_tokens":1,"output_tokens":1}}`)),
	}, nil
}

func retainedHeapAfterGC() uint64 {
	runtime.GC()
	runtime.GC()
	debug.FreeOSMemory()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.HeapAlloc
}
```

请求 payload 使用 `io.MultiReader` + `io.LimitReader` 生成固定大小合法 JSON；测试记录 2MB 与 8.9MB 两个阻塞点，断言：

```go
require.Less(t, heapAt89MB-heapAt2MB, uint64(3<<20), "retained heap must not scale with full request body size")
require.LessOrEqual(t, previewSize, int(service.DefaultRequestBodyPreviewLimitBytes))
```

测量顺序固定、每个 handler 在下一次测量前 release 并等待退出；若 `heapAt89MB < heapAt2MB`，按零增长处理，避免 uint64 下溢。

- [x] **Step $1: 运行阻塞测试确认旧链路超出增长预算**

Run: `go test ./internal/handler -run TestRequestBodyMemoryRetentionWhileUpstreamBlocked -count=1 -v`

Expected: 在全链路改造前 FAIL，8.9MB 相对 2MB 的 retained heap 增长接近完整 body 差值；改造后 PASS。

- [x] **Step $1: 运行阈值、清理和受控 identity 矩阵**

Run: `go test ./internal/handler -run '(TestRequestBody(DefaultSpoolThresholdMatrix|SizeMatrix)|TestRequestBodyMemoryRetentionWhileUpstreamBlocked)' -count=1 -v`

Expected: PASS；0.5MB/1MB 不落盘，1MB+1/2MB/10MB 落盘；upstream hash 与 client hash 一致；handler 返回后 spool 目录为空。

- [x] **Step $1: 更新 memory/context 复盘**

将 `memory/context/dmit-sub2api-memory-analysis.md` 的代码方案更新为已选 v2：

```markdown
**3. 请求体全链路 handle 化（本 change 的落地方案）**
- 默认 spool 阈值为 1MB，preview 上限为 256KB。
- canonical body 在请求级改写完成后进入 final handle；failover/retry 每轮按需物化并重建 attempt handle。
- session hash 使用 normalize 前 raw body，usage hash 使用 channel mapping 前 body，cyber key在 body 尚存时预计算。
- WSv2 reconnect map 不在本 change 范围，后续独立治理。
```

同时把“方案 3 pre_block semaphore”标为独立候选，不再误写成当前请求体治理的核心方案；保留生产数据与历史分析，不改动无关运维结论。

- [x] **Step $1: 全量 build、vet、test**

Run: `go build ./...`

Expected: PASS，无输出。

Run: `go vet ./...`

Expected: PASS，无输出。

Run: `go test ./... -count=1`

Expected: PASS，所有 package 输出 `ok` 或 `[no test files]`，无 FAIL、panic、临时文件清理超时或 race-like channel close。

- [x] **Step $1: 核对任务边界与最终 diff**

从仓库根目录运行：

Run: `git diff --check`

Expected: PASS，无尾随空格或 conflict marker。

Run: `git diff --name-only 8b494e1871f90b1a8559797f96f1099380a3fd4e -- backend docs memory`

Expected: 只包含本计划列出的生产代码、测试、OpenSpec/计划状态（若执行 workflow 自动更新）和 memory/context 文件；不得包含 WS forwarder、前端、数据库 migration 或部署配置。

- [x] **Step $1: 提交验证与文档**

```bash
git add internal/handler/request_body_memory_retention_test.go internal/handler/request_body_size_matrix_test.go ../memory/context/dmit-sub2api-memory-analysis.md
git commit -m "test: verify request bodies are released while waiting"
```

## 完成判据

- `tasks.md` 1.1-1.7：导出 1MB/256KB 默认值，三处引用统一，默认边界矩阵、build、vet 通过。
- `tasks.md` 2.1-2.7：Responses raw session/cyber 语义、final handle、循环内物化、槽位释放、payload hash 与 failover 测试全部覆盖。
- `tasks.md` 3.1-3.4：service 两条 retry 改写、错误/计费提取、首轮复用与 retry 测试全部覆盖。
- `tasks.md` 4.1-4.4：Chat handle、usage hash、cyber key 与定向测试全部覆盖。
- `tasks.md` 5.1-5.5：Anthropic、Gemini、Images/Grok 对齐，WS 明确无 diff。
- `tasks.md` 6.1-6.4：全量测试、阻塞 transport、默认大小矩阵和 memory/context 文档完成。

## 执行交接

计划实施时优先使用 `superpowers:subagent-driven-development`，每个 Task 使用独立执行上下文并在提交前复核；若在当前会话串行执行，则使用 `superpowers:executing-plans` 按 Task 1-6 顺序推进。Task 2-5 共享 body ownership 约束，不应并行修改同一 handler/service 文件。
