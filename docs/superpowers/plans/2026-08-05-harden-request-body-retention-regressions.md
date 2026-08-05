# 请求体内存驻留回归加固实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复异步 Images 和 Embeddings 绕过请求体生命周期契约的问题，并用关键回归测试阻止后续上游合并重新引入完整 body 驻留或资源泄漏。

**Architecture:** 异步 Images 在同步校验后把请求体写入现有 `RequestBodyHandle`，由后台 worker 独占并在统一终止路径清理；`newAsyncImageContext` 只从 handle 打开 reader，不再捕获 `[]byte`。Embeddings 保持现有发送和错误映射，仅在 transport 错误分支关闭非空 response body。阈值、Grok 静默拒绝和并发驻留通过入口级契约测试锁定。

**Tech Stack:** Go 1.x、Gin、`net/http`、`httptest`、`testify/require`、现有 `RequestBodyHandle` 与 Comet Native workflow。

## Global Constraints

- 严格 TDD：生产代码修改前必须先运行对应 RED 测试并记录预期失败。
- `DefaultRequestBodySpoolThresholdBytes` 保持 `1MiB`，`DefaultRequestBodyPreviewLimitBytes` 保持 `256KiB`。
- 不新增依赖、配置、数据库 schema 或新的请求体抽象。
- 保持请求 body、`Content-Type`、`Content-Length`、审计、usage、错误 envelope 与 failover 语义不变。
- `ErrRequestBodySpool` 必须保留错误链，不得回退到完整内存 body。
- `backend/internal/service/openai_ws_forwarder_v2.go` 和 `backend/internal/service/openai_ws_protocol_forward.go` 零改动。
- 不提交 `.comet/current-change.json` 或手工编辑 Runtime 管理的状态文件。

## File Map

- `backend/internal/service/openai_embeddings.go`：关闭 Embeddings transport 错误携带的 response body。
- `backend/internal/service/openai_transport_response_close_test.go`：复用共享 `resp+err` stub 锁定 Embeddings 关闭契约。
- `backend/internal/handler/image_task_handler.go`：异步 Images handle 创建、所有权转移、worker 上下文和清理。
- `backend/internal/handler/image_task_handler_test.go`：异步 body 重放、spool 失败、终止路径清理测试。
- `backend/internal/handler/request_body_size_matrix_test.go`：multipart 生产 `1MiB` 边界。
- `backend/internal/service/openai_gateway_grok_chat_bridge_test.go`：Grok 大请求静默拒绝回归测试。
- `backend/internal/handler/request_body_memory_retention_test.go`：并发异步 worker 阻塞时的 heap 契约。

---

### Task 1: Embeddings transport response body 关闭

**Files:**
- Modify: `backend/internal/service/openai_transport_response_close_test.go:38`
- Modify: `backend/internal/service/openai_embeddings.go:103`

**Interfaces:**
- Consumes: `transportResponseCloseUpstream.Do(*http.Request, string, int64, int) (*http.Response, error)`。
- Produces: `ForwardEmbeddings` 在 `err != nil` 时保证非空 `resp.Body` 已关闭；返回错误类型和响应 envelope 不变。

- [ ] **Step 1: 在共享 transport close 表中加入 Embeddings RED 用例**

在 `TestOpenAIChatCompletionsOpenAIPassthroughGrokMediaTransportErrorClosesResponseBody` 的 table 中加入：

```go
{
	name: "Embeddings",
	run: func(service *OpenAIGatewayService) error {
		body := []byte(`{"model":"text-embedding-3-small","input":"hello"}`)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewReader(body))
		account := &Account{
			ID: 3, Name: "embeddings", Platform: PlatformOpenAI,
			Type: AccountTypeAPIKey, Concurrency: 1,
			Credentials: map[string]any{"api_key": "key", "base_url": "https://api.openai.com"},
		}
		_, err := service.ForwardEmbeddings(context.Background(), c, account, body, "")
		return err
	},
},
```

共享测试现有断言 `require.True(t, responseBody.closed)` 即为目标契约。

- [ ] **Step 2: 运行 RED 测试**

Run: `go test ./internal/service -run TestOpenAIChatCompletionsOpenAIPassthroughGrokMediaTransportErrorClosesResponseBody/Embeddings -count=1 -v`

Expected: FAIL，`responseBody.closed` 为 `false`。

- [ ] **Step 3: 做最小生产修复**

在 `ForwardEmbeddings` 的 `if err != nil` 首行加入：

```go
if resp != nil && resp.Body != nil {
	_ = resp.Body.Close()
}
```

关闭逻辑必须位于 `errors.Is(err, ErrRequestBodySpool)` 判断之前，确保 sentinel 和普通 transport error 都释放 response。

- [ ] **Step 4: 运行 GREEN 与 Embeddings 回归测试**

Run: `go test ./internal/service -run '(TestOpenAIChatCompletionsOpenAIPassthroughGrokMediaTransportErrorClosesResponseBody|TestForwardEmbeddings)' -count=1 -v`

Expected: PASS。

- [ ] **Step 5: 提交 Task 1**

```bash
git add backend/internal/service/openai_embeddings.go backend/internal/service/openai_transport_response_close_test.go
git commit -m "fix: close embeddings transport error responses"
```

---

### Task 2: 异步 Images worker-owned 请求体

**Files:**
- Modify: `backend/internal/handler/image_task_handler.go:83-135,232-296`
- Modify: `backend/internal/handler/image_task_handler_test.go:47-245`

**Interfaces:**
- Consumes: `service.NewRequestBodyHandleFromBytes([]byte, service.RequestBodyHandleOptions)`, `(*service.RequestBodyHandle).Open`, `Size`, `CleanupRequestBodyHandle` 和 handler 包现有 `jsonRequestBodyHandleOptions`。
- Produces: `newAsyncImageContext(c *gin.Context, bodyHandle *service.RequestBodyHandle, lifecycleCtx context.Context, timeoutDuration time.Duration) (*gin.Context, *httptest.ResponseRecorder, context.CancelFunc, error)`。
- Produces: `(*AsyncImageHandler).runWithBodyHandle(taskID, platform string, taskBase *gin.Context, bodyHandle *service.RequestBodyHandle, lifecycleCtx context.Context)`，该方法接收并最终清理 handle 所有权。

- [ ] **Step 1: 添加大 body 重放与清理 RED 测试**

在 `image_task_handler_test.go` 添加一个测试 helper，将既有全局 options 指向测试目录并在 cleanup 恢复：

```go
func useAsyncImageSpoolDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := jsonRequestBodyHandleOptions
	jsonRequestBodyHandleOptions = service.RequestBodyHandleOptions{
		SpoolThresholdBytes: service.DefaultRequestBodySpoolThresholdBytes,
		PreviewLimitBytes:   service.DefaultRequestBodyPreviewLimitBytes,
		TempDir:             dir,
	}
	t.Cleanup(func() { jsonRequestBodyHandleOptions = old })
	return dir
}
```

添加 `TestAsyncImageHandlerSpoolsReplaysAndCleansOwnedBody`：构造尺寸大于 `1MiB` 的合法 JSON，`h.execute` 中分别读取 `c.Request.Body` 和 `c.Request.GetBody()`，把两份 hash 发到 channel 后阻塞。主测试确认：

```go
require.Equal(t, http.StatusAccepted, recorder.Code)
require.Equal(t, wantHash, got.bodyHash)
require.Equal(t, wantHash, got.replayHash)
require.Equal(t, "application/json", got.contentType)
require.Equal(t, int64(len(body)), got.contentLength)
assertMatrixTempFiles(t, spoolDir, "sub2api-request-body-", true)
close(release)
require.Eventually(t, func() bool {
	entries, err := os.ReadDir(spoolDir)
	return err == nil && len(entries) == 0
}, time.Second, 10*time.Millisecond)
```

- [ ] **Step 2: 添加 spool 创建失败 RED 测试**

添加 `TestAsyncImageHandlerSpoolCreateFailureReturns503WithoutTask`。把 `jsonRequestBodyHandleOptions.TempDir` 指向 `filepath.Join(t.TempDir(), "missing")`，发送大于 `1MiB` 的合法 JSON，并断言：

```go
require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
require.Contains(t, recorder.Body.String(), "Failed to spool request body")
require.Empty(t, store.tasks)
```

- [ ] **Step 3: 添加 worker 终止路径 RED 测试**

添加 `TestAsyncImageHandlerOwnedBodyCleanupOnTerminalPaths`，table 至少包含：

```go
tests := []struct {
	name    string
	execute func(*gin.Context)
}{
	{"success", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"data": []gin.H{{"url": "https://example.test/image.png"}}}) }},
	{"failure", func(c *gin.Context) { c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"message": "failed"}}) }},
	{"panic", func(*gin.Context) { panic("image task panic") }},
}
```

每个子测试发送大于 `1MiB` 的 body，等待 task 离开 `processing`，再断言 spool 目录为空。扩展既有 `TestAsyncImageHandlerShutdownCancelsExecution` 使用同一 spool options，并在 `Shutdown` 后断言目录为空。

另加 `TestAsyncImageHandlerRunRejectedCleansOwnedBody`：先调用 `tasks.Shutdown(context.Background())`，再提交请求，断言任务为 failed 且 spool 目录为空。

- [ ] **Step 4: 添加 spool open 失败 RED 测试**

直接测试新的上下文契约：创建 spooled handle，使用 `os.ReadDir` 找到并删除 `sub2api-request-body-` 文件，再调用 `newAsyncImageContext`。断言：

```go
_, _, _, err = newAsyncImageContext(c, handle, context.Background(), time.Minute)
require.ErrorIs(t, err, service.ErrRequestBodySpool)
```

该测试在签名仍接收 `[]byte` 时编译失败，属于预期 RED。

- [ ] **Step 5: 运行 Task 2 RED 测试**

Run: `go test ./internal/handler -run 'TestAsyncImageHandler(Spools|SpoolCreate|OwnedBody|RunRejected|Shutdown)|TestNewAsyncImageContext' -count=1 -v`

Expected: FAIL；至少观察到旧实现跨阻塞持有 body、spool 目录契约不成立或新签名尚不存在。

- [ ] **Step 6: 实现 handle 创建和所有权转移**

在 `Submit` 完成校验与安全审计后、`tasks.Create` 之前创建 handle：

```go
bodyHandle, err := service.NewRequestBodyHandleFromBytes(body, jsonRequestBodyHandleOptions)
if err != nil {
	imageTaskJSONError(c, http.StatusServiceUnavailable, "api_error", "Failed to spool request body")
	return
}
body = nil
if c.Request.Body != nil {
	_ = c.Request.Body.Close()
}
c.Request.Body = http.NoBody
c.Request.GetBody = nil
```

`tasks.Create` 失败时立即 `service.CleanupRequestBodyHandle(bodyHandle)`。任务创建成功后调用：

```go
if !h.tasks.Run(func(lifecycleCtx context.Context) {
	h.runWithBodyHandle(task.ID, platform, taskBase, bodyHandle, lifecycleCtx)
}) {
	service.CleanupRequestBodyHandle(bodyHandle)
	h.failTask(task.ID, http.StatusServiceUnavailable, imageTaskErrorPayload("api_error", "image task service is shutting down"))
}
```

- [ ] **Step 7: 实现 worker 单一清理路径**

增加：

```go
func (h *AsyncImageHandler) runWithBodyHandle(taskID, platform string, taskBase *gin.Context, bodyHandle *service.RequestBodyHandle, lifecycleCtx context.Context) {
	defer service.CleanupRequestBodyHandle(bodyHandle)
	taskCtx, recorder, cancel, err := newAsyncImageContext(taskBase, bodyHandle, lifecycleCtx, h.tasks.ExecutionTimeout())
	if err != nil {
		h.failTask(taskID, http.StatusServiceUnavailable, imageTaskErrorPayload("api_error", "failed to open image task request body"))
		return
	}
	h.run(taskID, platform, taskCtx, recorder, cancel)
}
```

修改 `newAsyncImageContext`：

```go
func newAsyncImageContext(c *gin.Context, bodyHandle *service.RequestBodyHandle, lifecycleCtx context.Context, timeoutDuration time.Duration) (*gin.Context, *httptest.ResponseRecorder, context.CancelFunc, error) {
	reader, err := bodyHandle.Open()
	if err != nil {
		return nil, nil, nil, err
	}
	base := context.WithoutCancel(c.Request.Context())
	if lifecycleCtx == nil {
		lifecycleCtx = context.Background()
	}
	executionBase, cancelExecution := context.WithCancel(base)
	stopLifecycle := context.AfterFunc(lifecycleCtx, cancelExecution)
	executionCtx, cancelTimeout := context.WithTimeout(executionBase, timeoutDuration)
	request := c.Request.Clone(executionCtx)
	request.Body = reader
	request.GetBody = bodyHandle.Open
	request.ContentLength = bodyHandle.Size()
	request.URL.Path = strings.TrimSuffix(request.URL.Path, "/async")

	taskCtx := c.Copy()
	recorder := httptest.NewRecorder()
	recorderCtx, _ := gin.CreateTestContext(recorder)
	taskCtx.Writer = recorderCtx.Writer
	taskCtx.Request = request
	cancel := func() {
		_ = request.Body.Close()
		stopLifecycle()
		cancelTimeout()
		cancelExecution()
	}
	return taskCtx, recorder, cancel, nil
}
```

- [ ] **Step 8: 运行 Task 2 GREEN 与现有异步 Images 测试**

Run: `go test ./internal/handler -run 'TestAsyncImageHandler|TestNewAsyncImageContext|TestSecurityAudit.*Media' -count=1 -v`

Expected: PASS。

- [ ] **Step 9: 提交 Task 2**

```bash
git add backend/internal/handler/image_task_handler.go backend/internal/handler/image_task_handler_test.go
git commit -m "fix: spool async image request bodies"
```

---

### Task 3: 锁定 multipart 阈值与 Grok 静默拒绝语义

**Files:**
- Modify: `backend/internal/handler/request_body_size_matrix_test.go:25-193`
- Modify: `backend/internal/service/openai_gateway_grok_chat_bridge_test.go:539-687`

**Interfaces:**
- Consumes: `service.DefaultRequestBodySpoolThresholdBytes`、`testRequestBodySizeMatrixMultipart`、`openAISilentRefusalMinRequestBodyBytes` 和 `ForwardAsChatCompletions`。
- Produces: 生产 multipart 精确边界测试；Grok bridge 原 body 长度驱动的 silent refusal failover 契约。

- [ ] **Step 1: 添加 multipart 生产边界测试**

新增：

```go
func TestMultipartRequestBodyDefaultSpoolThresholdBoundary(t *testing.T) {
	for _, size := range []int{1 << 20, (1 << 20) + 1} {
		t.Run(fmt.Sprintf("%d", size), func(t *testing.T) {
			testRequestBodySizeMatrixMultipart(t, size)
		})
	}
}
```

将 `testRequestBodySizeMatrixMultipart` 中 hardcoded `10 << 20` options 改为生产默认常量，并把等待期断言改为：

```go
wantSpool := int64(size) > service.DefaultRequestBodySpoolThresholdBytes
assertMatrixTempFiles(t, rawDir, "sub2api-request-body-", wantSpool)
```

- [ ] **Step 2: 运行 multipart 契约测试**

Run: `go test ./internal/handler -run 'TestMultipartRequestBodyDefaultSpoolThresholdBoundary|TestRequestBodySizeMatrix' -count=1 -v`

Expected: PASS；这是对现有正确生产行为的补测，不要求制造生产 RED。

- [ ] **Step 3: 添加 Grok 大请求 silent refusal 测试**

在 `openai_gateway_grok_chat_bridge_test.go` 添加 `TestForwardGrokChatViaResponsesLargeBodyPreservesSilentRefusalDetection`：

```go
content := strings.Repeat("x", openAISilentRefusalMinRequestBodyBytes)
body := []byte(`{"model":"grok","stream":true,"messages":[{"role":"user","content":"` + content + `"}]}`)
upstream := &httpUpstreamRecorder{resp: &http.Response{
	StatusCode: http.StatusOK,
	Header: http.Header{"Content-Type": []string{"text/event-stream"}, "Xai-Request-Id": []string{"silent-refusal"}},
	Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_empty\",\"status\":\"completed\",\"output\":[]}}\n\n")),
}}
```

使用 `grokChatBridgeTestAccount` 和现有 `grokQuotaAccountRepo` 构造 service，调用 `ForwardAsChatCompletions`，断言：

```go
require.Nil(t, result)
var failoverErr *UpstreamFailoverError
require.ErrorAs(t, err, &failoverErr)
require.True(t, IsOpenAISilentRefusalErrorBody(failoverErr.ResponseBody))
require.Empty(t, recorder.Body.String(), "silent refusal must not release pending empty chunks")
```

- [ ] **Step 4: 验证测试能捕获 bodyLength 回归**

临时把 `handleChatStreamingResponse(..., bodyLength)` 的最后参数替换为 `0`，运行：

Run: `go test ./internal/service -run TestForwardGrokChatViaResponsesLargeBodyPreservesSilentRefusalDetection -count=1 -v`

Expected: FAIL，未返回 `UpstreamFailoverError`。立即恢复生产代码，再运行同一命令，Expected: PASS。

- [ ] **Step 5: 提交 Task 3**

```bash
git add backend/internal/handler/request_body_size_matrix_test.go backend/internal/service/openai_gateway_grok_chat_bridge_test.go
git commit -m "test: lock request body retention boundaries"
```

---

### Task 4: 并发驻留、WS 边界与完整验证

**Files:**
- Modify: `backend/internal/handler/request_body_memory_retention_test.go:216-613`
- Verify only: `backend/internal/service/openai_ws_forwarder_v2.go`
- Verify only: `backend/internal/service/openai_ws_protocol_forward.go`
- Verify/update during Verify phase: `docs/comet/changes/harden-request-body-retention-regressions/verification.md`

**Interfaces:**
- Consumes: `retainedHeapAfterGC`, `retentionJSONBody`, `asyncImageMemoryStore`, `ImageTaskService.Run` 和 `AsyncImageHandler.Submit`。
- Produces: `TestAsyncImageRequestBodyMemoryRetentionWhileWorkersBlocked`，证明并发异步大请求不保留完整 body。

- [ ] **Step 1: 添加并发异步 worker heap 测试**

新增 helper：

```go
func measureBlockedAsyncImageHeap(t *testing.T, size int64, workers int, spoolDir string) uint64 {
	store := &asyncImageMemoryStore{tasks: make(map[string]*service.ImageTaskRecord)}
	tasks := service.NewImageTaskServiceWithUploader(store, nil, time.Hour, time.Minute)
	started := make(chan struct{}, workers)
	release := make(chan struct{})
	h := &AsyncImageHandler{tasks: tasks}
	h.execute = func(_ string, c *gin.Context) {
		started <- struct{}{}
		<-release
		c.JSON(http.StatusOK, gin.H{"data": []gin.H{{"url": "https://example.test/image.png"}}})
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		groupID := int64(1501)
		c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
			ID: 1501, UserID: 1501, GroupID: &groupID,
			Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
		})
		c.Next()
	})
	router.POST("/v1/images/generations/async", h.Submit)
	for i := 0; i < workers; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/images/generations/async", retentionAsyncImageJSONBody(size))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, req)
		require.Equal(t, http.StatusAccepted, recorder.Code, recorder.Body.String())
	}
	for i := 0; i < workers; i++ {
		waitMatrixSignal(t, started, "async image worker")
	}
	heap := retainedHeapAfterGC()
	close(release)
	require.NoError(t, tasks.Shutdown(context.Background()))
	assertMatrixTempFiles(t, spoolDir, "sub2api-request-body-", false)
	return heap
}
```

为避免 helper 中出现完整测试 body，新增 `retentionAsyncImageJSONBody(size int64) io.Reader`，使用 `io.MultiReader(strings.NewReader(prefix), io.LimitReader(retentionPaddingReader{}, padding), strings.NewReader(suffix))`。

新增测试：

```go
func TestAsyncImageRequestBodyMemoryRetentionWhileWorkersBlocked(t *testing.T) {
	for run := 0; run < 3; run++ {
		t.Run(fmt.Sprintf("run-%d", run), func(t *testing.T) {
			spoolDir := useAsyncImageSpoolDir(t)
			heap2MB := measureBlockedAsyncImageHeap(t, 2<<20, 4, spoolDir)
			heap89MB := measureBlockedAsyncImageHeap(t, 89<<20/10, 4, spoolDir)
			var growth uint64
			if heap89MB > heap2MB { growth = heap89MB - heap2MB }
			require.Less(t, growth, uint64(6<<20), "four workers must not retain four complete request bodies")
		})
	}
}
```

- [ ] **Step 2: 验证并发测试能捕获旧实现**

在 Task 2 修复提交的父提交上运行该测试，或临时恢复 `newAsyncImageContext(..., body []byte, ...)` 的闭包捕获后运行：

Run: `go test ./internal/handler -run TestAsyncImageRequestBodyMemoryRetentionWhileWorkersBlocked -count=1 -v`

Expected: FAIL，8.9MiB 与 2MiB 的 retained growth 超过上限。恢复 Task 2 实现后连续运行三次：

Run: `go test ./internal/handler -run TestAsyncImageRequestBodyMemoryRetentionWhileWorkersBlocked -count=3 -v`

Expected: PASS。

- [ ] **Step 3: 提交并发驻留测试**

```bash
git add backend/internal/handler/request_body_memory_retention_test.go
git commit -m "test: cover async image body retention"
```

- [ ] **Step 4: 运行聚焦与 unit-tag 验证**

Run: `go test ./internal/handler ./internal/service -run '(AsyncImage|RequestBody|Embeddings|GrokChatViaResponsesLargeBody)' -count=1`

Run: `go test -tags=unit ./internal/handler ./internal/service -count=1`

Expected: 两条命令均 PASS。

- [ ] **Step 5: 运行 WS 边界验证**

Run: `go test ./internal/service/openai_ws_v2 ./internal/service -run 'OpenAIWS|WebSocket' -count=1`

Run: `git diff 6e25e848d -- backend/internal/service/openai_ws_forwarder_v2.go backend/internal/service/openai_ws_protocol_forward.go`

Expected: 测试 PASS；diff 无输出。

- [ ] **Step 6: 运行完整项目验证**

从 `backend/` 运行：

```bash
go test ./... -count=1
go build ./...
go vet ./...
```

从仓库根目录运行：

```bash
git diff --check
git status --short
```

Expected: 所有 Go 命令 exit 0，`git diff --check` 无输出，status 仅允许 Runtime 管理的 `.comet/current-change.json` 未跟踪。

- [ ] **Step 7: 独立只读复审**

让 reviewer 对以下内容逐项核对：

- `docs/openspec/specs/request-body-retention-control/spec.md` 旧基线；
- `docs/comet/changes/harden-request-body-retention-regressions/specs/request-body-retention-control/spec.md` 完整目标；
- 当前非 WS body-carrying 路由；
- handle ownership、spool error、`resp+err` close、Grok bodyLength 和并发 heap 测试。

只有 reviewer 返回零 Critical/High，或所有真实发现已完成新一轮 RED/GREEN 修复后，才进入 Comet Verify。

- [ ] **Step 8: 推进 Comet Build**

在所有提交和验证完成后运行：

```bash
comet native next harden-request-body-retention-regressions --summary "修复异步 Images 请求体驻留和 Embeddings transport 响应泄漏，补齐 multipart、Grok 静默拒绝与并发 heap 契约测试。" --artifact backend/internal/handler/image_task_handler.go
```

按 Runtime continuation 进入 Verify；不要手工修改 `comet-state.yaml`、receipt 或 evidence 文件。
