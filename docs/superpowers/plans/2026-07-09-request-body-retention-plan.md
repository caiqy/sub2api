---
change: optimize-request-body-retention
design-doc: docs/superpowers/specs/2026-07-09-request-body-retention-design.md
base-ref: 56e7525536b8fa12a0f116ca57dfe834ea4d9207
archived-with: 2026-07-10-optimize-request-body-retention
---

# 请求体驻留优化实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 `/responses` 请求在不改变审计、转发、计费和 failover 语义的前提下，减少 request/upstream body 的长生命周期副本，并让 `>10MB` 请求在等待上游期间通过临时文件承载完整 body。

**Architecture:** 新增一个小型 `RequestBodyHandle` 统一管理完整 body、preview、hash 和重复打开能力。handler 先用它承接入站请求并提前计算 `requestPayloadHash`；usage detail 和 ops 只保留 preview；OpenAI upstream request builder 在大 body 时用文件型 `GetBody` 重放，替换当前 `bytes.NewReader(body)` 的等待期内存驻留。

**Tech Stack:** Go 1.25，Gin，标准库 `os/io/crypto/sha256`，现有 `gjson`/`sjson`，`testify/require`，不新增依赖。

## Global Constraints

- 不做流式 JSON 解析。
- 不处理 response body / upstream response body 的完整捕获。
- 不扩展到 Anthropic/Gemini 等其他协议入口。
- 不新增数据库 schema。
- 默认 `spool threshold = 10MB`。
- 默认 `preview limit = 5MB`。
- 保持 `requestPayloadHash`、内容审计、上游转发和 failover 语义不变。
- 除非用户明确要求，不执行 `git commit`；执行者完成后只保留工作区改动和测试结果。
- 最小有效实现：优先改现有文件，只新增一个 request body handle 文件和对应测试文件，不引入新包层级。

## File Structure

- Create: `backend/internal/service/request_body_handle.go`：定义 `RequestBodyHandle`、内存/文件模式、preview/hash、cleanup 和 stale file sweep。
- Create: `backend/internal/service/request_body_handle_test.go`：覆盖内存模式、文件模式、重复 `Open()`、cleanup 和 spool 失败路径。
- Modify: `backend/internal/server/middleware/usage_detail_capture.go`：删除 request body 预读/replay，改为只捕获头和响应体，并接受显式 request preview setter。
- Modify: `backend/internal/server/middleware/usage_detail_capture_test.go`：把“完整 request body 自动捕获”改成“downstream 正常读取 + 显式 setter 写回 preview”。
- Modify: `backend/internal/service/usage_detail_capture.go`：新增 request body setter，移除 `SetUsageUpstreamRequest` 对 `req.GetBody()` 的 fallback 读取。
- Modify: `backend/internal/service/ops_upstream_context.go`：新增 preview setter，移除从 request 自行 `ReadAll` 完整 upstream body 的路径。
- Modify: `backend/internal/handler/openai_gateway_handler.go`：引入 raw/effective inbound handle，提前计算 `requestPayloadHash`，改用 preview 填充 usage detail。
- Create: `backend/internal/handler/openai_gateway_request_body_retention_test.go`：覆盖 `/responses` 预览写回、hash 语义不变和 detail snapshot 不再持有完整大 body。
- Modify: `backend/internal/service/openai_gateway_service.go`：在 `/responses` HTTP 和 passthrough 路径为最终 outbound body 创建 `RequestBodyHandle`，让 request builder 和 `GetBody` 在大 body 时走文件重放。
- Modify: `backend/internal/service/openai_gateway_service_test.go`：增加 build request / `GetBody` / spool 文件重放的回归测试。

archived-with: 2026-07-10-optimize-request-body-retention
---

### Task 1: 实现 RequestBodyHandle 基础能力

**Files:**
- Create: `backend/internal/service/request_body_handle.go`
- Create: `backend/internal/service/request_body_handle_test.go`

**Interfaces:**
- Produces:
  - `type RequestBodyHandleOptions struct { SpoolThresholdBytes int64; PreviewLimitBytes int64; TempDir string; FilePrefix string }`
  - `func NewRequestBodyHandleFromReader(r io.Reader, opts RequestBodyHandleOptions) (*RequestBodyHandle, error)`
  - `func NewRequestBodyHandleFromBytes(body []byte, opts RequestBodyHandleOptions) (*RequestBodyHandle, error)`
  - `func (h *RequestBodyHandle) Open() (io.ReadCloser, error)`
  - `func (h *RequestBodyHandle) ReadAll() ([]byte, error)`
  - `func (h *RequestBodyHandle) PreviewString() string`
  - `func (h *RequestBodyHandle) Hash() string`
  - `func (h *RequestBodyHandle) Size() int64`
  - `func (h *RequestBodyHandle) Cleanup() error`
  - `func CleanupStaleRequestBodySpoolFiles(dir, prefix string, olderThan time.Duration, now time.Time) error`

- [x] **Step 1: 写先失败的 handle 测试**

在 `backend/internal/service/request_body_handle_test.go` 增加：

```go
func TestRequestBodyHandle_MemoryAndFileModes(t *testing.T) {
    t.Run("memory mode keeps bytes in RAM", func(t *testing.T) {
        h, err := NewRequestBodyHandleFromBytes([]byte(`{"model":"gpt-5"}`), RequestBodyHandleOptions{
            SpoolThresholdBytes: 10 << 20,
            PreviewLimitBytes:   5 << 20,
            TempDir:             t.TempDir(),
            FilePrefix:          "sub2api-test-",
        })
        require.NoError(t, err)
        require.Equal(t, int64(len(`{"model":"gpt-5"}`)), h.Size())
        require.NotEmpty(t, h.Hash())
        require.Equal(t, `{"model":"gpt-5"}`, h.PreviewString())
    })

    t.Run("file mode reopens full body", func(t *testing.T) {
        body := []byte(strings.Repeat("x", 2048))
        h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{
            SpoolThresholdBytes: 1024,
            PreviewLimitBytes:   256,
            TempDir:             t.TempDir(),
            FilePrefix:          "sub2api-test-",
        })
        require.NoError(t, err)

        r1, err := h.Open()
        require.NoError(t, err)
        first, err := io.ReadAll(r1)
        require.NoError(t, err)
        require.NoError(t, r1.Close())

        r2, err := h.Open()
        require.NoError(t, err)
        second, err := io.ReadAll(r2)
        require.NoError(t, err)
        require.NoError(t, r2.Close())

        require.Equal(t, body, first)
        require.Equal(t, body, second)
        require.Contains(t, h.PreviewString(), strings.Repeat("x", 256))
    })
}

func TestRequestBodyHandle_CleanupRemovesSpoolFile(t *testing.T) {
    body := []byte(strings.Repeat("z", 2048))
    dir := t.TempDir()
    h, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{
        SpoolThresholdBytes: 1024,
        PreviewLimitBytes:   128,
        TempDir:             dir,
        FilePrefix:          "sub2api-test-",
    })
    require.NoError(t, err)

    require.NoError(t, h.Cleanup())
    matches, err := filepath.Glob(filepath.Join(dir, "sub2api-test-*"))
    require.NoError(t, err)
    require.Empty(t, matches)
}
```

- [x] **Step 2: 运行测试并确认失败**

Run: `go -C backend test ./internal/service -run 'TestRequestBodyHandle' -count=1`

Expected: FAIL，提示 `RequestBodyHandle` / 构造函数未定义。

- [x] **Step 3: 写最小实现**

在 `backend/internal/service/request_body_handle.go` 增加：

```go
type RequestBodyHandle struct {
    size        int64
    hash        string
    preview     string
    memory      []byte
    spoolPath   string
    spoolActive bool
}

func (h *RequestBodyHandle) Open() (io.ReadCloser, error) {
    if h == nil {
        return nil, errors.New("request body handle is nil")
    }
    if h.spoolActive {
        return os.Open(h.spoolPath)
    }
    return io.NopCloser(bytes.NewReader(h.memory)), nil
}

func (h *RequestBodyHandle) ReadAll() ([]byte, error) {
    r, err := h.Open()
    if err != nil {
        return nil, err
    }
    defer func() { _ = r.Close() }()
    return io.ReadAll(r)
}
```

并补齐：
- 创建时一次性计算 `sha256`、`size`、`preview`
- 大于阈值时用 `os.CreateTemp` 写文件
- `Cleanup()` 删除 spool 文件
- `CleanupStaleRequestBodySpoolFiles()` 只删除匹配前缀且超过时限的文件

- [x] **Step 4: 运行测试并确认通过**

Run: `go -C backend test ./internal/service -run 'TestRequestBodyHandle' -count=1`

Expected: PASS。

- [x] **Step 5: 检查点**

Run: `git diff -- backend/internal/service/request_body_handle.go backend/internal/service/request_body_handle_test.go`

Expected: 只看到 handle 基础能力和对应测试，无 handler / middleware 变更。

### Task 2: 改造 usage detail 和 ops capture，只保留 preview

**Files:**
- Modify: `backend/internal/server/middleware/usage_detail_capture.go`
- Modify: `backend/internal/server/middleware/usage_detail_capture_test.go`
- Modify: `backend/internal/service/usage_detail_capture.go`
- Modify: `backend/internal/service/ops_upstream_context.go`

**Interfaces:**
- Consumes: `(*RequestBodyHandle).PreviewString()`
- Produces:
  - `func SetUsageRequestBody(c *gin.Context, body string)`
  - `func SetOpsUpstreamRequestBodyPreview(c *gin.Context, body string)`
  - `func SetUsageUpstreamRequest(c *gin.Context, req *http.Request, body string)`（保留签名，但移除 `GetBody` fallback）

- [x] **Step 1: 写先失败的 middleware / capture 测试**

在 `backend/internal/server/middleware/usage_detail_capture_test.go` 增加或替换为：

```go
func TestUsageDetailCaptureMiddleware_DownstreamStillReadsFullBodyWithoutPreread(t *testing.T) {
    gin.SetMode(gin.TestMode)

    var (
        downstream string
        snapshot   *UsageDetailSnapshot
    )

    r := gin.New()
    r.Use(UsageDetailCapture())
    r.POST("/capture", func(c *gin.Context) {
        raw, err := io.ReadAll(c.Request.Body)
        require.NoError(t, err)
        downstream = string(raw)
        snapshot = BuildUsageDetailSnapshot(c)
        c.Status(http.StatusNoContent)
    })

    req := httptest.NewRequest(http.MethodPost, "/capture", strings.NewReader(`{"message":"hi"}`))
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    require.Equal(t, `{"message":"hi"}`, downstream)
    require.NotNil(t, snapshot)
    require.Equal(t, "", snapshot.RequestBody)
}

func TestSetUsageUpstreamRequest_DoesNotFallbackToGetBody(t *testing.T) {
    gin.SetMode(gin.TestMode)

    var snapshot *UsageDetailSnapshot
    r := gin.New()
    r.Use(UsageDetailCapture())
    r.POST("/capture", func(c *gin.Context) {
        req, err := http.NewRequest(http.MethodPost, "https://example.com/v1/responses", strings.NewReader("ignored"))
        require.NoError(t, err)
        called := 0
        req.GetBody = func() (io.ReadCloser, error) {
            called++
            return io.NopCloser(strings.NewReader("should-not-be-read")), nil
        }

        service.SetUsageUpstreamRequest(c, req, "")
        snapshot = BuildUsageDetailSnapshot(c)
        require.Equal(t, 0, called)
        c.Status(http.StatusNoContent)
    })

    req := httptest.NewRequest(http.MethodPost, "/capture", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    require.NotNil(t, snapshot)
    require.Equal(t, "", snapshot.UpstreamRequestBody)
}
```

- [x] **Step 2: 运行测试并确认失败**

Run: `go -C backend test ./internal/server/middleware -run 'TestUsageDetailCaptureMiddleware_DownstreamStillReadsFullBodyWithoutPreread|TestSetUsageUpstreamRequest_DoesNotFallbackToGetBody' -count=1`

Expected: FAIL；当前 middleware 会自动捕获完整 request body，`SetUsageUpstreamRequest` 会自行读取 `GetBody()`。

- [x] **Step 3: 写最小实现**

按下面方向修改：

```go
// backend/internal/server/middleware/usage_detail_capture.go
type usageDetailRequestBodySetter interface {
    SetUsageRequestBody(body string)
}

func UsageDetailCapture() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestHeaders := ""
        if c.Request != nil {
            requestHeaders = service.FormatUsageDetailRequestHeadersText(c.Request)
        }
        collector := &usageDetailCollector{requestHeaders: requestHeaders}
        c.Set(service.UsageDetailCaptureContextKey, collector)
        c.Writer = &usageDetailResponseWriter{ResponseWriter: c.Writer, collector: collector}
        c.Next()
        collector.responseHeaders = service.FormatUsageDetailResponseHeadersText(c.Writer.Status(), c.Writer.Header())
    }
}

func (c *usageDetailCollector) SetUsageRequestBody(body string) {
    if c != nil {
        c.requestBody = body
    }
}
```

```go
// backend/internal/service/usage_detail_capture.go
func SetUsageRequestBody(c *gin.Context, body string) {
    if c == nil {
        return
    }
    v, ok := c.Get(UsageDetailCaptureContextKey)
    if !ok {
        return
    }
    collector, ok := v.(interface{ SetUsageRequestBody(string) })
    if ok && collector != nil {
        collector.SetUsageRequestBody(body)
    }
}

func SetUsageUpstreamRequest(c *gin.Context, req *http.Request, body string) {
    if c == nil || req == nil {
        return
    }
    // 删除 req.GetBody fallback；body 由调用方显式提供 preview。
}
```

```go
// backend/internal/service/ops_upstream_context.go
func SetOpsUpstreamRequestBodyPreview(c *gin.Context, body string) {
    if c == nil || strings.TrimSpace(body) == "" {
        return
    }
    c.Set(OpsUpstreamRequestBodyKey, body)
}
```

同时删除：
- `replayRequestBody`
- `captureRequestPrefix`
- middleware 中对 request body 的预读与 replay 注入

- [x] **Step 4: 运行测试并确认通过**

Run: `go -C backend test ./internal/server/middleware ./internal/service -run 'TestUsageDetailCapture|TestSetUsageUpstreamRequest' -count=1`

Expected: PASS。

- [x] **Step 5: 检查点**

Run: `git diff -- backend/internal/server/middleware/usage_detail_capture.go backend/internal/server/middleware/usage_detail_capture_test.go backend/internal/service/usage_detail_capture.go backend/internal/service/ops_upstream_context.go`

Expected: 只看到 request/upstream preview 捕获链路变化，没有 handler / OpenAI service 逻辑改动。

### Task 3: 改造 `/responses` handler，提前算 hash 并写回 request preview

**Files:**
- Modify: `backend/internal/handler/openai_gateway_handler.go`
- Create: `backend/internal/handler/openai_gateway_request_body_retention_test.go`

**Interfaces:**
- Consumes:
  - `service.NewRequestBodyHandleFromReader`
  - `service.NewRequestBodyHandleFromBytes`
  - `service.SetUsageRequestBody`
- Produces:
  - `/responses` handler 中的 raw/effective handle 读取顺序
  - 进入转发前就稳定的 `requestPayloadHash`

- [x] **Step 1: 写先失败的 `/responses` handler 回归测试**

创建 `backend/internal/handler/openai_gateway_request_body_retention_test.go`，增加：

```go
func TestOpenAIGatewayHandler_ResponsesPassesPreviewSnapshotAndStableHash(t *testing.T) {
    gin.SetMode(gin.TestMode)

    cfg := &config.Config{RunMode: config.RunModeSimple}
    cfg.Default.RateMultiplier = 1
    cfg.Gateway.MaxAccountSwitches = 1

    usageRepo := &openAIChatCompletionsUsageLogRepoStub{}
    upstream := &openAIChatCompletionsHTTPUpstreamStub{
        response: &http.Response{
            StatusCode: http.StatusOK,
            Header: http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"req_test_123"}},
            Body: io.NopCloser(strings.NewReader(`{"id":"resp_1","model":"gpt-5","usage":{"input_tokens":1,"output_tokens":1,"input_tokens_details":{"cached_tokens":0}}}`)),
        },
    }

    // 复用现有 stub 方式创建 handler，断言 usageRepo.lastLog.DetailSnapshot.RequestBody 是 preview，
    // RequestPayloadHash 仍等于 service.HashUsageRequestPayload(compact-normalized body)。
}
```

在这个测试里：
- 临时把 `/responses` preview limit 降到 `32` 字节，避免构造 5MB fixture
- 断言 `usageRepo.lastLog.DetailSnapshot.RequestBody` 不等于完整请求体
- 断言其包含 preview 和 `truncated` 元信息
- 断言 `usageRepo.lastBillingCmd.RequestPayloadHash`（或等价 stub 字段）等于对 handler 有效 body 计算出的 hash

- [x] **Step 2: 运行测试并确认失败**

Run: `go -C backend test ./internal/handler -run 'TestOpenAIGatewayHandler_ResponsesPassesPreviewSnapshotAndStableHash' -count=1`

Expected: FAIL；当前 `/responses` detail snapshot 仍保留完整 request body，且 hash 在转发返回后才计算。

- [x] **Step 3: 写最小实现**

在 `backend/internal/handler/openai_gateway_handler.go` 里按下面顺序改造：

```go
rawHandle, err := service.NewRequestBodyHandleFromReader(c.Request.Body, service.RequestBodyHandleOptions{
    SpoolThresholdBytes: 10 << 20,
    PreviewLimitBytes:   5 << 20,
    TempDir:             "",
    FilePrefix:          "sub2api-request-body-",
})
if err != nil { /* 保持当前 bad request 处理 */ }
defer func() { _ = rawHandle.Cleanup() }()

service.SetUsageRequestBody(c, rawHandle.PreviewString())
body, err := rawHandle.ReadAll()
if err != nil { /* 保持当前 bad request 处理 */ }

effectiveHandle := rawHandle
if normalizedCompact {
    effectiveHandle, err = service.NewRequestBodyHandleFromBytes(body, sameOpts)
    if err != nil { /* 500/400 按当前路径处理 */ }
    defer func() { if effectiveHandle != rawHandle { _ = effectiveHandle.Cleanup() } }()
}

requestPayloadHash := effectiveHandle.Hash()
```

然后：
- 把 line 440 / 538 一类的 `HashUsageRequestPayload(body)` 改成复用 `requestPayloadHash`
- `detailSnapshot := middleware2.BuildUsageDetailSnapshot(c)` 保留，但 snapshot 里的 request body 已变成 preview
- failover 日志里的 `ExtractOpenAIReasoningEffortFromBody(...)` 仍使用当前 attempt `body` 局部变量，不改语义

- [x] **Step 4: 运行测试并确认通过**

Run: `go -C backend test ./internal/handler -run 'TestOpenAIGatewayHandler_ResponsesPassesPreviewSnapshotAndStableHash' -count=1`

Expected: PASS。

- [x] **Step 5: 检查点**

Run: `git diff -- backend/internal/handler/openai_gateway_handler.go backend/internal/handler/openai_gateway_request_body_retention_test.go`

Expected: 只看到 `/responses` handler 的 request body 生命周期和 snapshot/hash 路径变化，没有扩散到 messages/chat completions。

### Task 4: 改造 OpenAI upstream request builder，让大 body 走文件型 `GetBody`

**Files:**
- Modify: `backend/internal/service/openai_gateway_service.go`
- Modify: `backend/internal/service/openai_gateway_service_test.go`

**Interfaces:**
- Consumes:
  - `service.NewRequestBodyHandleFromBytes`
  - `(*RequestBodyHandle).Open`
  - `service.SetUsageUpstreamRequest`
  - `service.SetOpsUpstreamRequestBodyPreview`
- Produces:
  - 大 body request builder 的 file-backed `req.Body` / `req.GetBody`
  - usage / ops upstream preview 的显式写回

- [x] **Step 1: 写先失败的 builder 回归测试**

在 `backend/internal/service/openai_gateway_service_test.go` 增加：

```go
func TestOpenAIBuildUpstreamRequestWithSourceBody_ReplaysLargeBodyFromGetBody(t *testing.T) {
    gin.SetMode(gin.TestMode)

    rec := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(rec)
    c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader([]byte(`{"model":"gpt-5"}`)))

    svc := &OpenAIGatewayService{cfg: &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}}}
    account := &Account{Type: AccountTypeAPIKey, Platform: PlatformOpenAI, Credentials: map[string]any{"base_url": "https://example.com/v1"}}

    body := []byte(strings.Repeat("x", 2048))
    handle, err := NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 1024, PreviewLimitBytes: 64, TempDir: t.TempDir(), FilePrefix: "sub2api-test-"})
    require.NoError(t, err)
    defer func() { _ = handle.Cleanup() }()

    req, err := svc.buildUpstreamRequestWithSourceBody(c.Request.Context(), c, account, body, body, handle, "token", false, "", false)
    require.NoError(t, err)

    first, err := io.ReadAll(req.Body)
    require.NoError(t, err)
    replay, err := req.GetBody()
    require.NoError(t, err)
    second, err := io.ReadAll(replay)
    require.NoError(t, err)
    require.Equal(t, body, first)
    require.Equal(t, body, second)
}
```

再补一条 passthrough 版本同样覆盖 `buildUpstreamRequestOpenAIPassthrough(...)`。

- [x] **Step 2: 运行测试并确认失败**

Run: `go -C backend test ./internal/service -run 'TestOpenAIBuildUpstreamRequest(WithSourceBody_ReplaysLargeBodyFromGetBody|OpenAIPassthrough.*)' -count=1`

Expected: FAIL；当前 builder 仍只接受 `[]byte`，没有 file-backed handle 参数。

- [x] **Step 3: 写最小实现**

在 `backend/internal/service/openai_gateway_service.go` 中：

```go
func (s *OpenAIGatewayService) buildUpstreamRequestWithSourceBody(
    ctx context.Context,
    c *gin.Context,
    account *Account,
    sourceBody []byte,
    body []byte,
    bodyHandle *RequestBodyHandle,
    token string,
    isStream bool,
    promptCacheKey string,
    isCodexCLI bool,
) (*http.Request, error) {
    // applyAccountPassthroughFieldsWithContext(...) 之后，如果 bodyHandle 内容已不等于最终 body，重新创建 handle
    if bodyHandle == nil || bodyHandle.Size() != int64(len(body)) {
        var err error
        bodyHandle, err = NewRequestBodyHandleFromBytes(body, RequestBodyHandleOptions{SpoolThresholdBytes: 10 << 20, PreviewLimitBytes: 5 << 20, FilePrefix: "sub2api-request-body-"})
        if err != nil {
            return nil, err
        }
    }
    reader, err := bodyHandle.Open()
    if err != nil {
        return nil, err
    }
    req, err := http.NewRequestWithContext(ctx, "POST", targetURL, reader)
    if err != nil {
        _ = reader.Close()
        return nil, err
    }
    req.GetBody = bodyHandle.Open
    SetUsageUpstreamRequest(c, req, bodyHandle.PreviewString())
    SetOpsUpstreamRequestBodyPreview(c, bodyHandle.PreviewString())
    return req, nil
}
```

同样改造：
- `buildUpstreamRequestOpenAIPassthrough(...)`
- `/responses` HTTP 路径在最终 outbound body 确定后创建 `bodyHandle`
- invalid_encrypted_content retry 重新 `json.Marshal(reqBody)` 后重新创建 `bodyHandle`

并删除：
- `setOpsUpstreamRequestBodyFromRequest(c, upstreamReq)`
- `SetUsageUpstreamRequest(c, upstreamReq, "")` 这种依赖 request 自行回读的调用方式

- [x] **Step 4: 运行测试并确认通过**

Run: `go -C backend test ./internal/service ./internal/handler -run 'Test(RequestBodyHandle|UsageDetailCapture|OpenAIBuildUpstreamRequest|OpenAIGatewayHandler_ResponsesPassesPreviewSnapshotAndStableHash)' -count=1`

Expected: PASS。

- [x] **Step 5: 最终核验**

Run: `go -C backend test ./internal/service ./internal/handler -count=1`

Expected: PASS；不出现新的 request body 预读、`GetBody` 完整回读或 hash 语义回归。
