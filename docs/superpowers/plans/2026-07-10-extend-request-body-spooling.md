---
change: extend-request-body-spooling
design-doc: docs/superpowers/specs/2026-07-10-extend-request-body-spooling-design.md
base-ref: 0f389fe7ed783ca4a8444fbe6d12acb9d3e19af6
---

# 扩展请求体文件化实施计划

> **供执行型 Agent 使用：** 必须逐任务执行；推荐使用 `superpowers:subagent-driven-development`，也可使用 `superpowers:executing-plans`。步骤使用复选框追踪。

**目标：** 将目标 JSON 与媒体请求入口接入可重放的 `RequestBodyHandle`，在不改变协议与计费语义的前提下消除上游等待期间的完整请求体长期内存驻留。

**架构：** 依照 [技术设计](../specs/2026-07-10-extend-request-body-spooling-design.md)，在 `handler` 包增加一个仅管理 raw/effective/form ownership 的 coordinator；JSON raw handle 消费共享 decoded reader，multipart raw handle 消费原始请求流。各 handler 仅在同步解析、审计或转换的小作用域内 materialize `[]byte`；服务的 retry/failover 从 effective handle 的 `Open()` 获得新 reader，并继续使用现有 OpenAI owned-handle context。

**技术栈：** Go 1.25、Gin、标准库 `io`/`mime/multipart`/`net/http`、已有 `github.com/klauspost/compress/zstd`、Testify。

## 全局约束

- 以 `0f389fe7ed783ca4a8444fbe6d12acb9d3e19af6` 为实现与测试基线；计划创建阶段不预先修改 `tasks.md`，实施时每完成一个对应任务再勾选并随该任务提交。
- 固定 spool threshold 为 `10MB`，preview limit 为 `5MB`；不新增配置、依赖、数据库 schema、协议接口、工厂或引用计数体系。
- JSON 的 threshold 按有效 body 大小计算；`gzip`/`x-gzip`、`deflate`、`zstd` 受 `64MB` 解压上限约束，identity 继续只受现有应用 body limit 约束。
- multipart 的 raw threshold 按客户端发送原始字节计算，不解压 `Content-Encoding`。
- raw 表示入站有效 JSON 或原始 multipart；effective 表示某次上游最终发送内容。仅当 `Size()` 和 `Hash()` 都相同才复用同一 handle。
- coordinator 为 raw 和基础 effective 的根 owner，handler 创建成功后立即 `defer Cleanup()`；OpenAI attempt 派生 handle 继续由 `openAIOwnedBodyHandleContextKey` 和 `closeOpenAIRequestBody` 清理。
- `ErrRequestBodySpool` 在响应尚未写出时映射 503；`*http.MaxBytesError` 映射既有 413；损坏或不支持的编码映射既有 400；不得因 spool 失败回退到完整内存路径。
- usage/ops 只能保存 immutable bounded snapshot、size、hash 或媒体脱敏 metadata；不得将 coordinator、handle、preview 源 `[]byte` 或 multipart 文件交给异步 worker。
- `multipart.Form.RemoveAll()`、raw handle、effective handle 必须覆盖成功、业务拒绝、路由失败、客户端取消、panic、上游 4xx/5xx 与流式结束路径；清理错误不得覆盖已确定的业务响应。

## 文件结构

- 新建 `backend/internal/handler/request_body_coordinator.go`：JSON decoded-reader 入站、multipart 原始入站、同步 `ReadRaw`、effective 替换、去重幂等 cleanup。
- 新建 `backend/internal/handler/request_body_coordinator_test.go`：coordinator 的阈值、hash、ownership、错误映射与终止路径测试。
- 修改 `backend/internal/pkg/httputil/body.go`：提取 decoded reader 构造函数，保持 `ReadRequestBodyWithPrealloc` 的兼容调用方式。
- 修改 `backend/internal/pkg/httputil/body_test.go`：覆盖四种编码、损坏输入、64MB 边界及 metadata 原子更新。
- 修改 `backend/internal/handler/gateway_handler.go`、`backend/internal/handler/gateway_handler_responses.go`、`backend/internal/service/gateway_request.go`、`backend/internal/service/gateway_service.go` 与 `backend/internal/service/gateway_forward_as_responses.go`：迁移 Anthropic Messages 与分组 Responses，并让内部 retry 从 handle 重开 body。
- 修改 `backend/internal/handler/openai_chat_completions.go`、`openai_embeddings.go`、`openai_gateway_handler.go`、`openai_gateway_count_tokens.go` 与 `backend/internal/service/openai_gateway_service.go`：迁移 OpenAI JSON 入口及 attempt body ownership。
- 修改 `backend/internal/handler/gemini_v1beta_handler.go`：迁移三种 Gemini action。
- 修改 `backend/internal/handler/openai_images.go`、`grok_media.go` 及其现有 service multipart 构建 helper：迁移 OpenAI/Grok Images、Videos 与 multipart 管线。
- 扩展现有 `*_test.go`，优先放在被测 handler 同目录；仅当测试跨 handler 共享 coordinator fixture 时放在 `request_body_coordinator_test.go`。

### Task 1: 共享解码 reader 的失败优先测试

**对应 OpenSpec：** 1.1（identity、压缩 JSON 的阈值、64MB、preview、hash、503、cleanup 的共享基础）。

**文件：**
- 修改：`backend/internal/pkg/httputil/body_test.go`
- 修改：`backend/internal/pkg/httputil/body.go`

**接口：**
- 产生：`NewDecodedRequestBodyReader(req *http.Request) (io.ReadCloser, error)`；压缩编码成功时返回已施加 `http.MaxBytesReader(nil, ..., maxDecompressedBodySize)` 的 decoded reader，identity 原样返回现有受应用 body limit 保护的 reader。
- 保持：`ReadRequestBodyWithPrealloc(req *http.Request) ([]byte, error)` 的调用方和错误类型不变。

- [x] **步骤 1：写失败测试。** 在 `body_test.go` 增加表驱动测试，使用 identity、`gzip`、`x-gzip`、`deflate`、`zstd` 读取同一 JSON，断言得到原始 JSON；对 `br` 与损坏 gzip/zstd 断言返回错误；对压缩内容解压后正好 `64 << 20` 与 `(64 << 20) + 1` 断言分别成功和 `errors.As(err, *http.MaxBytesError)`，并保留 identity 不受该解压上限影响的断言。

- [x] **步骤 2：验证测试失败。** 运行：`go test ./internal/pkg/httputil -run 'TestNewDecodedRequestBodyReader' -count=1`（工作目录 `backend`）。预期：因 `NewDecodedRequestBodyReader` 未定义而失败。

- [x] **步骤 3：最小实现。** 在 `body.go` 从现有 `Content-Encoding` 分支提取 `NewDecodedRequestBodyReader`：identity 返回 `req.Body` 的不重复关闭包装；gzip/x-gzip、deflate、zstd 创建 decoder 并将 decoder 的关闭纳入返回 `io.ReadCloser`，仅压缩 decoder 外层使用 `http.MaxBytesReader(nil, ..., maxDecompressedBodySize)`。让 `ReadRequestBodyWithPrealloc` 调用该函数并 `io.ReadAll`，仅在读取成功后删除 `Content-Encoding`/`Content-Length`、写入 decoded `ContentLength`，从而保持失败时 request metadata 不变。

- [x] **步骤 4：验证通过。** 运行：`go test ./internal/pkg/httputil -count=1`（工作目录 `backend`）。预期：通过，旧 `ReadRequestBodyWithPrealloc` 测试仍覆盖兼容入口。

- [x] **步骤 5：提交。** `git add backend/internal/pkg/httputil/body.go backend/internal/pkg/httputil/body_test.go && git commit -m "refactor: share request body decoder"`

### Task 2: JSON coordinator 与 spool 错误映射

**对应 OpenSpec：** 1.1、1.2。

**文件：**
- 新建：`backend/internal/handler/request_body_coordinator.go`
- 新建：`backend/internal/handler/request_body_coordinator_test.go`
- 修改：`backend/internal/handler/request_body_limit.go`

**接口：**
- 产生：`newJSONRequestBody(req *http.Request) (*requestBodyCoordinator, error)`、`(*requestBodyCoordinator).ReadRaw() ([]byte, error)`、`SetEffectiveBytes([]byte) error`、`SetEffectiveReader(io.Reader) error`、`Effective() *service.RequestBodyHandle`、`Cleanup()`。
- 约束：各 handler 直接以 `errors.Is(err, service.ErrRequestBodySpool)` 返回其既有 503 格式，以现有 `extractMaxBytesError` 返回其既有 413 格式，其余读取错误继续返回既有 400 格式。

- [x] **步骤 1：写失败测试。** 在 `request_body_coordinator_test.go` 用临时目录构造 10MB 以下、恰好 10MB、10MB+1 的 identity 与 gzip JSON；断言 `raw.Size()`、`raw.Hash()`、5MB preview 截断、两次 `Open()` 内容一致，且大请求有 spool、小请求无 spool。令临时目录不存在，断言 `errors.Is(err, service.ErrRequestBodySpool)`；用 `httptest` handler 断言该错误在尚未写响应时为 503，而解压上限仍为 413。

- [x] **步骤 2：验证测试失败。** 运行：`go test ./internal/handler -run 'TestRequestBodyCoordinator_(JSON|Spool)' -count=1`（工作目录 `backend`）。预期：因 coordinator 未定义而失败。

- [x] **步骤 3：最小实现。** 新增非导出的 `requestBodyCoordinator{raw, effective *service.RequestBodyHandle; form *multipart.Form}`。`newJSONRequestBody` 调用 `httputil.NewDecodedRequestBodyReader`，将 reader 直接交给 `service.NewRequestBodyHandleFromReader`，成功后再清除编码 metadata 并写入 raw 的有效 size；`ReadRaw` 只调用 `raw.ReadAll`；`SetEffectiveBytes`/`SetEffectiveReader` 创建 handle 并仅在 size 和 hash 都相等时保留 raw；`Effective` 返回 effective；错误分类 helper 只识别 spool 与 MaxBytes。

- [x] **步骤 4：验证通过。** 运行：`go test ./internal/handler -run 'TestRequestBodyCoordinator_(JSON|Spool)' -count=1`（工作目录 `backend`）。预期：通过。

- [x] **步骤 5：提交。** `git add backend/internal/handler/request_body_coordinator.go backend/internal/handler/request_body_coordinator_test.go backend/internal/handler/request_body_limit.go && git commit -m "feat: add request body coordinator"`

### Task 3: coordinator ownership 与全终止路径

**对应 OpenSpec：** 1.3。

**文件：**
- 修改：`backend/internal/handler/request_body_coordinator.go`
- 修改：`backend/internal/handler/request_body_coordinator_test.go`
- 修改：`backend/internal/service/request_body_handle_test.go`

**接口：**
- 保持：`Cleanup()` 为幂等；raw/effective 相同仅清理一次；替换 coordinator 独占 effective 时清理旧 handle；登记的 `multipart.Form` 由 cleanup 调用 `RemoveAll()`。

- [x] **步骤 1：写失败测试。** 用专用 tempdir 和 `t.Cleanup` 创建 raw/effective 相同、内容变更的 effective、连续两次 effective 替换三种情况；断言每个应删除的 spool 文件在 `Cleanup()` 后不存在，第二次 `Cleanup()` 不报错。为成功、业务拒绝、`context.CancelFunc()`、panic recovery 和模拟上游 4xx/5xx 各建一个 `gin.New()` 路由，断言 handler 返回后目录为空；在 `request_body_handle_test.go` 保留并运行删除失败 retry/stale sweep 测试。

- [x] **步骤 2：验证测试失败。** 运行：`go test ./internal/handler ./internal/service -run 'Test(RequestBodyCoordinator_Cleanup|RequestBodyHandle_(Cleanup|Stale))' -count=1`（工作目录 `backend`）。预期：替换和 multipart 清理断言失败。

- [x] **步骤 3：最小实现。** `Cleanup` 收集 raw、effective 的唯一非 nil 指针，以 `service.CleanupRequestBodyHandle` 清理；若存在 form，调用 `form.RemoveAll()`，仅记录错误且不写 response。`SetEffectiveBytes` 与 `SetEffectiveReader` 在替换前确认旧 effective 既非 raw、也未由 request owned context 接管，然后清理旧 handle；不增加引用计数或全局 registry。

- [x] **步骤 4：验证通过。** 运行：`go test ./internal/handler ./internal/service -run 'Test(RequestBodyCoordinator_Cleanup|RequestBodyHandle_(Cleanup|Stale))' -count=1`（工作目录 `backend`）。预期：通过。

- [x] **步骤 5：提交。** `git add backend/internal/handler/request_body_coordinator.go backend/internal/handler/request_body_coordinator_test.go backend/internal/service/request_body_handle_test.go && git commit -m "test: cover request body ownership cleanup"`

### Task 4: Anthropic Messages 与分组 Responses 接入

**对应 OpenSpec：** 2.1。

**文件：**
- 修改：`backend/internal/handler/gateway_handler.go` 的 `(*GatewayHandler).Messages`
- 修改：`backend/internal/handler/gateway_handler_responses.go` 的 `(*GatewayHandler).Responses`
- 修改：`backend/internal/service/gateway_forward_as_responses.go` 的 `ForwardAsResponses`
- 修改：`backend/internal/service/gateway_request.go` 的 `RequestBodyRef` / `ParsedRequest.CloneForBody`
- 修改：`backend/internal/service/gateway_service.go` 的 `GatewayService.Forward` 与 Anthropic upstream request builder
- 修改：`backend/internal/service/antigravity_gateway_service.go` 的 Messages forwarding 边界，使其借用 handle 并在网络等待前释放 materialized bytes
- 修改：`backend/internal/service/antigravity_gateway_service_test.go`
- 修改：`backend/internal/service/gateway_request_test.go`
- 新建：`backend/internal/service/gateway_request_body_handle_test.go`（不带 unit build tag，确保 handle 兼容测试可独立取得 GREEN）
- 修改：`backend/internal/service/gateway_forward_as_responses_test.go`
- 修改：`backend/internal/service/ops_upstream_context_test.go`（移除被本任务触发的脆弱源码字面量计数，改为行为契约）
- 新建：`backend/internal/handler/gateway_request_body_spooling_test.go`
- 修改：`backend/internal/handler/terminal_failed_usage_test.go`（仅在生产 handler 生命周期 fixture 需要复用时）

**接口：**
- 消费：`newJSONRequestBody`、`ReadRaw`、`SetEffectiveBytes`、`Effective`、`Cleanup`。
- 产生：handler 在 `defer coordinator.Cleanup()` 后向 service 传递 effective handle；`RequestBodyRef`/upstream builder 可从 handle 为每个 attempt 重开 reader，service 不让 handler 的 `body []byte` 跨越上游等待。

- [ ] **步骤 1：写失败测试。** 给真实 `GatewayHandler.Messages` 与 `.Responses` 各增加一个超过 10MB 的压缩请求和一个模型映射请求：模拟上游阻塞期间断言 raw spool 存在、usage/ops snapshot 是 bounded preview、上游收到的 hash 等于 effective hash；释放上游后断言 spool 删除。Responses 测试必须断言 Responses-to-Messages 转换后 effective 内容被发送；Messages 测试必须覆盖 Antigravity Claude→Gemini payload、same-account retry replacement，并断言内容审计、session 与计费 hash 未变。

- [ ] **步骤 2：验证测试失败。** 运行：`go test ./internal/handler -run 'TestGatewayHandler_(Messages|Responses).*RequestBody' -count=1`（工作目录 `backend`）。预期：现有路径仍将完整 `body` 传入转发，生命周期断言失败。

- [ ] **步骤 3：最小实现。** 两个 handler 在认证后创建 JSON coordinator 并立即 defer cleanup；使用 `ReadRaw` 在局部 helper 执行既有 JSON 校验、`ParseGatewayRequest`/`gjson`、审计、session、模型映射和转换，随后调用 `SetEffectiveBytes`。扩展 `RequestBodyRef`/`ParsedRequest` 以借用 effective handle，并让 `GatewayService.Forward`、`ForwardAsResponses` 及 Anthropic upstream builder 在每个 attempt 从 handle 重开 reader。Antigravity 借用 Claude handle，在同步转换 helper 中创建 service-owned Gemini payload handle；其 retry loop 只保存 handle/metadata，每次 request 通过 `Open`/`GetBody` 重开，payload 改写时替换并清理旧 owned handle。账号级改写可短暂 materialize，但完整 `[]byte` 不得跨上游等待；转换对象不得以完整 bytes 形式放入 Gin context 或 usage async 数据。所有 Forward 阶段的 wrapped spool error 必须回到 handler 的 503 分类。

- [ ] **步骤 4：验证通过。** 运行：`go test ./internal/handler -run 'TestGatewayHandler_(Messages|Responses)' -count=1`（工作目录 `backend`）。预期：通过，包含流式与错误透传既有测试。

- [ ] **步骤 5：提交。** `git add backend/internal/handler/gateway_handler.go backend/internal/handler/gateway_handler_responses.go backend/internal/handler/gateway_request_body_spooling_test.go backend/internal/handler/terminal_failed_usage_test.go backend/internal/service/gateway_request.go backend/internal/service/gateway_request_test.go backend/internal/service/gateway_request_body_handle_test.go backend/internal/service/gateway_service.go backend/internal/service/gateway_forward_as_responses.go backend/internal/service/gateway_forward_as_responses_test.go backend/internal/service/antigravity_gateway_service.go backend/internal/service/antigravity_gateway_service_test.go backend/internal/service/ops_upstream_context_test.go && git commit -m "feat: spool anthropic gateway request bodies"`

### Task 5: OpenAI Chat Completions 与 Embeddings 接入

**对应 OpenSpec：** 2.2。

**文件：**
- 修改：`backend/internal/handler/openai_chat_completions.go` 的 `ChatCompletions`
- 修改：`backend/internal/handler/openai_embeddings.go` 的 `Embeddings`
- 修改：`backend/internal/service/openai_gateway_service.go` 的 `Forward` 及 request-body helpers
- 修改：`backend/internal/service/openai_embeddings.go` 的 `ForwardEmbeddings`
- 新建：`backend/internal/handler/openai_request_body_spooling_test.go`
- 修改：`backend/internal/handler/openai_gateway_request_body_retention_test.go`

**接口：**
- 复用：`service.BindOpenAIRequestBodyHandle`、`openAIRequestBodyHandleForBytes`、`openAINewRequestWithBodyHandle`、`closeOpenAIRequestBody`。
- 约束：借用 coordinator effective handle 时 `owned=false`；attempt 内模型转换创建的派生 handle 时 `owned=true`。

- [ ] **步骤 1：写失败测试。** 为 Chat 与 Embeddings 建立可失败一次后成功的 `httpUpstream` stub；输入 12MB JSON 和模型映射，记录每个 attempt 的 bytes/hash，断言两次相同且 request 的 `GetBody` 可重开。另断言上游等待期间 usage/ops snapshot 不等于完整 body，并在请求完成后没有 spool 文件。

- [ ] **步骤 2：验证测试失败。** 运行：`go test ./internal/handler ./internal/service -run 'TestOpenAI(GatewayHandler|GatewayService).*(Chat|Embeddings).*RequestBody' -count=1`（工作目录 `backend`）。预期：现有 failover 循环捕获 `body`/`forwardBody`，重放与驻留断言失败。

- [ ] **步骤 3：最小实现。** handlers 用 coordinator 替代 `ReadRequestBodyWithPrealloc`，仅在同步校验、stream 解析、审计、session hash 与映射时持有 raw bytes；映射完成后 `SetEffectiveBytes`，循环只捕获 handle。service 扩展现有 OpenAI request helper 的适用调用点：每个 attempt 调用 `openAINewRequestWithBodyHandle`，设置 `GetBody`、`ContentLength`；按账号转换需要 bytes 时把 `ReadAll`、转换和 `openAIRequestBodyHandleForBytes` 放进 attempt helper，构造 request 后立即结束该局部作用域。

- [ ] **步骤 4：验证通过。** 运行：`go test ./internal/handler ./internal/service -run 'TestOpenAI(GatewayHandler|GatewayService).*(Chat|Embeddings)' -count=1`（工作目录 `backend`）。预期：通过。

- [ ] **步骤 5：提交。** `git add backend/internal/handler/openai_chat_completions.go backend/internal/handler/openai_embeddings.go backend/internal/handler/openai_request_body_spooling_test.go backend/internal/handler/openai_gateway_request_body_retention_test.go backend/internal/service/openai_gateway_service.go backend/internal/service/openai_embeddings.go && git commit -m "feat: replay openai request bodies from handles"`

### Task 6: Anthropic/OpenAI JSON 回归矩阵

**对应 OpenSpec：** 2.3。

**文件：**
- 修改：`backend/internal/handler/gateway_request_body_spooling_test.go`
- 修改：`backend/internal/handler/openai_request_body_spooling_test.go`
- 修改：`backend/internal/handler/openai_gateway_usage_context_test.go`
- 修改：`backend/internal/handler/openai_gateway_request_body_retention_test.go`

- [ ] **步骤 1：写失败测试。** 对 Messages、Responses、Chat、Embeddings 分别覆盖小请求、大请求、gzip、上游 4xx、上游 5xx、请求 context 取消、retry/failover；每例断言旧 HTTP status/错误格式、model mapping、usage 指纹与流式语义不变。至少一例在模拟上游等待时检查 `UsageDetailCapture` 与 ops context 中没有完整 12MB 字符串。

- [ ] **步骤 2：验证测试失败。** 运行：`go test ./internal/handler -run 'Test(GatewayHandler|OpenAIGatewayHandler).*(Messages|Responses|Chat|Embeddings).*(Compressed|Failover|Canceled|Upstream)' -count=1`（工作目录 `backend`）。预期：新增生命周期或 snapshot 断言失败。

- [ ] **步骤 3：补齐最小回归 fixture。** 复用现有 account、billing、usage 与 HTTP upstream stubs；只新增能记录 body hash、阻塞/释放发送及列出 tempdir 文件的字段。不要建立通用 mock 框架或测试专用配置。

- [ ] **步骤 4：验证通过。** 运行：`go test ./internal/handler -run 'Test(GatewayHandler|OpenAIGatewayHandler).*(Messages|Responses|Chat|Embeddings)' -count=1`（工作目录 `backend`）。预期：通过。

- [ ] **步骤 5：提交。** `git add backend/internal/handler/gateway_request_body_spooling_test.go backend/internal/handler/openai_request_body_spooling_test.go backend/internal/handler/openai_gateway_usage_context_test.go backend/internal/handler/openai_gateway_request_body_retention_test.go && git commit -m "test: cover json request body spooling"`

### Task 7: Gemini 三种 action 接入

**对应 OpenSpec：** 3.1。

**文件：**
- 修改：`backend/internal/handler/gemini_v1beta_handler.go` 的 `GeminiV1BetaModels`
- 修改：`backend/internal/service/gemini_messages_compat_service.go` 的 `ForwardNative`
- 修改：`backend/internal/service/antigravity_gateway_service.go` 的 `ForwardGemini`
- 修改：`backend/internal/handler/gemini_v1beta_handler_test.go`
- 修改：`backend/internal/handler/gemini_v1beta_failed_usage_unit_test.go`

**接口：**
- 消费：JSON coordinator effective handle；`generateContent`、`streamGenerateContent`、`countTokens` 走同一入口。
- 约束：模型 URL path 不计入 raw hash；thoughtSignature 清理后的最终发送 bytes 成为 effective。

- [ ] **步骤 1：写失败测试。** 对三个 action 使用 12MB identity 与 gzip body，模拟 Gemini upstream 接收 hash；对 failover 断言每次从 effective handle 重开。构造 thoughtSignature 在账号切换时被清理的请求，断言 raw hash 未变化、effective body 是清理后版本且没有循环捕获 raw `[]byte`。

- [ ] **步骤 2：验证测试失败。** 运行：`go test ./internal/handler -run 'TestGeminiV1Beta.*(GenerateContent|StreamGenerateContent|CountTokens).*RequestBody' -count=1`（工作目录 `backend`）。预期：现有 `body` 在 failover 循环内重写，effective ownership 断言失败。

- [ ] **步骤 3：最小实现。** 在完成 action 和模型路径解析后创建 coordinator 并 defer cleanup；在局部同步作用域读取 raw bytes，保留现有内容审计、sticky session 计算和 failed usage 输入。每次 thoughtSignature 清理后调用 `SetEffectiveBytes`，并将 upstream forwarding 改为从 `Effective().Open()` 读取；不将路径模型拼入 body hash。

- [ ] **步骤 4：验证通过。** 运行：`go test ./internal/handler -run 'TestGeminiV1Beta' -count=1`（工作目录 `backend`）。预期：通过。

- [ ] **步骤 5：提交。** `git add backend/internal/handler/gemini_v1beta_handler.go backend/internal/handler/gemini_v1beta_handler_test.go backend/internal/handler/gemini_v1beta_failed_usage_unit_test.go backend/internal/service/gemini_messages_compat_service.go backend/internal/service/antigravity_gateway_service.go && git commit -m "feat: spool gemini native request bodies"`

### Task 8: Gemini 语义回归

**对应 OpenSpec：** 3.2。

**文件：**
- 修改：`backend/internal/handler/gemini_v1beta_handler_test.go`
- 修改：`backend/internal/handler/gemini_v1beta_failed_usage_unit_test.go`
- 修改：`backend/internal/handler/gemini_cli_session_test.go`

- [ ] **步骤 1：写失败测试。** 对 model path 解析、内容审计拒绝、Google JSON 错误格式、已启动 SSE 的终止、failed usage 写入、`/antigravity` 强制平台各保留一例；每例在大 body 进入上游等待时断言 spool 生命周期正确且原先状态码/消息不变。

- [ ] **步骤 2：验证测试失败。** 运行：`go test ./internal/handler -run 'TestGemini(V1Beta|CLI).*(Error|FailedUsage|Antigravity|Stream|Model)' -count=1`（工作目录 `backend`）。预期：新增 spool 生命周期断言失败。

- [ ] **步骤 3：补齐定向 fixture。** 在现有 Gemini test stubs 中增加可阻塞 upstream 与 body-hash 捕获；复用既有 Google error assertion，禁止为测试创建第二套响应格式化函数。

- [ ] **步骤 4：验证通过。** 运行：`go test ./internal/handler -run 'TestGemini(V1Beta|CLI)' -count=1`（工作目录 `backend`）。预期：通过。

- [ ] **步骤 5：提交。** `git add backend/internal/handler/gemini_v1beta_handler_test.go backend/internal/handler/gemini_v1beta_failed_usage_unit_test.go backend/internal/handler/gemini_cli_session_test.go && git commit -m "test: preserve gemini request body semantics"`

### Task 9: 媒体 JSON、multipart 与脱敏测试

**对应 OpenSpec：** 4.1。

**文件：**
- 修改：`backend/internal/handler/openai_images_controls_test.go`
- 修改：`backend/internal/handler/openai_images_failover_test.go`
- 修改：`backend/internal/handler/openai_images_group_user_concurrency_test.go`
- 修改：`backend/internal/handler/grok_media_test.go`
- 修改：`backend/internal/handler/request_body_coordinator_test.go`

- [ ] **步骤 1：写失败测试。** 覆盖 Images generate/edit、Grok Images、Grok Videos create 的 JSON、multipart、inline base64/data URL、源图与 mask：断言 raw 超过 10MB 时 spool、multipart preview 从 usage/ops 省略正文并只含模型/prompt/尺寸/文件数/源图或遮罩标记、上游完整接收所有文本与文件 bytes。对 multipart parser 的临时文件、raw spool、effective spool 分别断言 request 返回后删除。

- [ ] **步骤 2：验证测试失败。** 运行：`go test ./internal/handler -run 'Test(OpenAIImages|GrokMedia).*(Multipart|Inline|Mask|Source|Spool)' -count=1`（工作目录 `backend`）。预期：当前入口预读完整 `[]byte` 或写入通用 preview，断言失败。

- [ ] **步骤 3：补齐 fixture。** 用标准库 `multipart.Writer` 构造请求与服务端接收断言；使用专用 tempdir，记录 `filepath.Glob` 结果。数据 URL 与 base64 用非敏感短前缀加重复内容构造，断言 snapshot 仅是 omission/metadata，不匹配任何正文片段。

- [ ] **步骤 4：验证通过。** 运行：`go test ./internal/handler -run 'Test(OpenAIImages|GrokMedia)' -count=1`（工作目录 `backend`）。预期：通过。

- [ ] **步骤 5：提交。** `git add backend/internal/handler/openai_images_*_test.go backend/internal/handler/grok_media_test.go backend/internal/handler/request_body_coordinator_test.go && git commit -m "test: cover media body spooling"`

### Task 10: OpenAI/Grok 媒体入口与 pipe 重建

**对应 OpenSpec：** 4.2。

**文件：**
- 修改：`backend/internal/handler/openai_images.go` 的 `Images`
- 修改：`backend/internal/handler/grok_media.go` 的 `handleGrokMedia`
- 修改：`backend/internal/service/openai_gateway_service.go` 中 Images/Videos forwarding 与 multipart request builder
- 修改：`backend/internal/service/grok_media.go` 的 `ForwardGrokMedia`
- 修改：`backend/internal/handler/request_body_coordinator.go`

**接口：**
- 产生：`newMultipartRequestBody(req *http.Request, maxMemory int64) (*requestBodyCoordinator, error)`，raw 读取原始请求流；从 `raw.Open()` 创建解析 request，记录其 `MultipartForm`。
- 约束：effective multipart 由 `io.Pipe` + `multipart.Writer` 写入，并由 `NewRequestBodyHandleFromReader` 同步消费；写端失败必须 `CloseWithError(err)`。

- [ ] **步骤 1：写失败测试。** 对一次 multipart edit 建立 producer 中途返回错误的 service stub，断言 pipe consumer 返回原始错误、没有发送半截上游请求且所有临时文件删除；成功 case 断言 effective `Content-Type` 含 writer boundary、`ContentLength` 等于 effective handle size、每次 retry 得到完整相同 multipart。

- [ ] **步骤 2：验证测试失败。** 运行：`go test ./internal/handler ./internal/service -run 'Test(OpenAIImages|GrokMedia|OpenAIGatewayService).*(Multipart|Pipe|Retry)' -count=1`（工作目录 `backend`）。预期：尚无 pipe 生成/handle ownership，测试失败。

- [ ] **步骤 3：最小实现。** Images/Grok 的有 body 路由按 Content-Type 选择 JSON 或 multipart coordinator；状态查询路由不创建 handle。multipart coordinator 从 raw handle open 的 reader 使用现有 `ParseMultipartForm(maxMemory)`，登记 form。需要重建上游 body 时启动 producer goroutine，`multipart.Writer` 写入 `io.PipeWriter`，消费端用 `SetEffectiveReader(pipeReader)` 完成后等待 producer；producer 以 `CloseWithError` 传递错误。仅将最终 content type 与 effective handle 交给 service；不创建完整 multipart `[]byte`。

- [ ] **步骤 4：验证通过。** 运行：`go test ./internal/handler ./internal/service -run 'Test(OpenAIImages|GrokMedia|OpenAIGatewayService).*(Multipart|Pipe|Retry)' -count=1`（工作目录 `backend`）。预期：通过。

- [ ] **步骤 5：提交。** `git add backend/internal/handler/openai_images.go backend/internal/handler/grok_media.go backend/internal/handler/request_body_coordinator.go backend/internal/service/openai_images.go backend/internal/service/grok_media.go && git commit -m "feat: spool media multipart request bodies"`

### Task 11: 媒体业务与资源回归

**对应 OpenSpec：** 4.3。

**文件：**
- 修改：`backend/internal/handler/openai_images_failover_test.go`
- 修改：`backend/internal/handler/grok_media_test.go`
- 修改：`backend/internal/handler/openai_gateway_usage_context_test.go`

- [ ] **步骤 1：写失败测试。** 为 generate、edit、video create、video status、权限或模型校验拒绝、上游 4xx/5xx、same-account retry、account failover 各保留一例。断言 video status 无 request body 时不创建 spool；所有媒体 case 的 usage/ops 不含原始 multipart、base64、data URL 或文件正文；所有失败路径均清除 raw/effective/form 文件。

- [ ] **步骤 2：验证测试失败。** 运行：`go test ./internal/handler -run 'Test(OpenAIImages|GrokMedia).*(Generate|Edit|Video|Reject|Upstream|Failover)' -count=1`（工作目录 `backend`）。预期：新增不泄露正文或无残留断言失败。

- [ ] **步骤 3：最小测试调整。** 为现有媒体 upstream recorder 增加 request hash、content type 和阻塞开关；复用 `service.grokMediaRequestBodyPreview`/现有脱敏 helper，不新增另一套 redactor。

- [ ] **步骤 4：验证通过。** 运行：`go test ./internal/handler -run 'Test(OpenAIImages|GrokMedia)' -count=1`（工作目录 `backend`）。预期：通过。

- [ ] **步骤 5：提交。** `git add backend/internal/handler/openai_images_failover_test.go backend/internal/handler/grok_media_test.go backend/internal/handler/openai_gateway_usage_context_test.go && git commit -m "test: verify media request cleanup"`

### Task 12: 跨协议 coordinator 与 503 契约

**对应 OpenSpec：** 5.1。

**文件：**
- 修改：`backend/internal/handler/request_body_coordinator_test.go`
- 修改：`backend/internal/handler/handler_usage_detail_contract_test.go`
- 修改：`backend/internal/handler/openai_gateway_request_body_retention_test.go`

- [ ] **步骤 1：写失败测试。** 建立表驱动契约，列出 Anthropic Messages/Responses、OpenAI Chat/Embeddings/Images、Grok Images/Videos、Gemini 三个 action；每项断言进入 shared coordinator、usage/ops 只读取 preview/metadata、模拟 effective `Open` spool 错误返回 503。流式响应已开始的 case 断言继续使用既有流式终止，不写第二个 HTTP 错误。

- [ ] **步骤 2：验证测试失败。** 运行：`go test ./internal/handler -run 'TestRequestBodySpoolingCrossProtocolContract' -count=1`（工作目录 `backend`）。预期：某个尚未覆盖的入口或错误映射断言失败。

- [ ] **步骤 3：最小实现修正。** 根据失败入口，仅接入现有 coordinator/error helper，不新增协议分支或 per-protocol ownership 机制；确保所有 handler 在写响应前识别 `errors.Is(err, service.ErrRequestBodySpool)` 并返回该协议既有 503 格式。

- [ ] **步骤 4：验证通过。** 运行：`go test ./internal/handler -run 'TestRequestBodySpoolingCrossProtocolContract|TestHandlerUsageDetailContract' -count=1`（工作目录 `backend`）。预期：通过。

- [ ] **步骤 5：提交。** `git add backend/internal/handler/request_body_coordinator_test.go backend/internal/handler/handler_usage_detail_contract_test.go backend/internal/handler/openai_gateway_request_body_retention_test.go backend/internal/handler/gateway_handler.go backend/internal/handler/gateway_handler_responses.go backend/internal/handler/openai_chat_completions.go backend/internal/handler/openai_embeddings.go backend/internal/handler/openai_images.go backend/internal/handler/grok_media.go backend/internal/handler/gemini_v1beta_handler.go && git commit -m "test: enforce request body spool contract"`

### Task 13: 全量自动化验证

**对应 OpenSpec：** 5.2。

**文件：**
- 修改：仅修复 Task 1-12 定向测试发现的现有实现或测试文件；不得新增功能范围。

- [ ] **步骤 1：运行后端全量测试。** 运行：`go test ./... -count=1`（工作目录 `backend`）。预期：全部通过。

- [ ] **步骤 2：运行前端契约验证。** 运行：`pnpm test:run`（工作目录 `frontend`），随后运行 `pnpm typecheck`（工作目录 `frontend`）。预期：全部通过，usage detail 展示契约未变。

- [ ] **步骤 3：执行受控端侧矩阵。** 在测试环境分别发送 `5MB`、`10MB`、`12MB` 的 identity JSON、gzip JSON、multipart；对每个请求记录客户端 SHA-256、模拟上游 SHA-256、usage detail、ops snapshot 和 `sub2api-request-body-*` 文件列表。断言 5MB/10MB 不创建 spool、12MB 创建并在 response 后删除，客户端 hash 与上游 hash 相同，gzip 以解压后大小判定。

- [ ] **步骤 4：复跑受影响测试。** 运行：`go test ./internal/pkg/httputil ./internal/handler ./internal/service -count=1`（工作目录 `backend`）。预期：通过。

- [ ] **步骤 5：提交。** 若步骤 1-4 没有修正，不创建空提交；若修正了测试，执行 `git add backend/internal/pkg/httputil/body_test.go backend/internal/handler/request_body_coordinator_test.go backend/internal/handler/gateway_request_body_spooling_test.go backend/internal/handler/openai_request_body_spooling_test.go backend/internal/handler/openai_gateway_usage_context_test.go backend/internal/handler/gemini_v1beta_handler_test.go backend/internal/handler/grok_media_test.go && git commit -m "test: stabilize request body spooling coverage"`。

### Task 14: RSS、生命周期与业务语义端侧验收

**对应 OpenSpec：** 5.3。

**文件：**
- 不修改代码；保存受控环境的命令输出、hash 对照与 RSS 采样到变更验证记录，不更新 OpenSpec tasks。

- [ ] **步骤 1：准备延迟上游采样。** 将受控测试上游设置为在读取请求后阻塞，分别发起 12MB identity、gzip 与 multipart 请求；在阻塞窗口记录容器 RSS、spool 文件数量和每个文件大小。

- [ ] **步骤 2：验证长期引用释放。** 在同步解析完成且上游仍阻塞时检查 Go heap profile 或容器 RSS 变化，并检查 usage detail/ops：完整请求体只能出现在 spool，不得出现在 usage/ops 字段；允许同步解析阶段短暂峰值，不将其作为失败。

- [ ] **步骤 3：释放上游并检查清理。** 放行上游成功、4xx、5xx、取消与流式中断五种结局；逐例等待 handler 返回和 cleanup retry，断言目录无本请求 spool，multipart `RemoveAll` 的文件也不存在，响应及计费/usage 指纹与基线一致。

- [ ] **步骤 4：最终验证。** 再次运行：`go test ./... -count=1`（工作目录 `backend`）以及 `pnpm test:run && pnpm typecheck`（工作目录 `frontend`）。预期：全部通过。

- [ ] **步骤 5：提交。** 本任务只产生验证证据，不提交代码；如 Task 13 后仍有修正，回到对应最小任务补充测试与单独提交，禁止将验收结果包装为代码提交。

## 覆盖核对

- `1.1` 至 `1.3`：Task 1、2、3 覆盖 decoder、阈值/preview/hash、503/413、raw/effective ownership、取消/panic/stale cleanup。
- `2.1` 至 `2.3`：Task 4、5、6 覆盖 Anthropic Messages/Responses、OpenAI Chat/Embeddings、压缩/小大请求/4xx/5xx/cancel/retry/failover/usage-ops。
- `3.1` 至 `3.2`：Task 7、8 覆盖 Gemini 三 action、模型路径、审计、Google errors、流式、failed usage、Antigravity。
- `4.1` 至 `4.3`：Task 9、10、11 覆盖 JSON/multipart/inline binary、源图/遮罩、`io.Pipe`、`CloseWithError`、`RemoveAll`、媒体所有业务终止路径。
- `5.1` 至 `5.3`：Task 12、13、14 覆盖跨协议契约、503、全量自动化、5/10/12MB 端侧矩阵、RSS 与 spool 生命周期。

## 实施前自审

- 已逐项覆盖 `tasks.md` 的 5 组 14 项，未添加范围外配置、依赖、schema、接口、工厂或 OpenSpec 勾选修改。
- 所有代码任务给出准确文件、目标符号、失败测试、定向命令、预期结果与提交建议；验证任务明确无代码提交条件。
- 不含 `TBD`、`TODO`、"类似任务"、未指定的错误处理或未指定的测试步骤；所有路径均以当前基线实际文件为准。
