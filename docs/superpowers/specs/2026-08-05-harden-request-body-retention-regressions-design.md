# 请求体内存驻留回归加固设计

## 背景

已归档的 `reduce-request-body-memory-retention` 将非 WebSocket 网关请求体改为 `RequestBodyHandle` 驱动，并以 `1MiB` spool 阈值和 `256KiB` preview 上限限制上游等待期内存驻留。归档后新增的异步 Images 路径直接读取并把完整 `[]byte` 捕获进后台任务，绕过了该契约；Embeddings 还缺少 transport 同时返回 response 和 error 时的响应体清理。

本变更从已合并旧治理提交的 `main` 创建独立 Native change。旧 Classic 归档只作为行为基线，不修改历史产物。

## 目标

- 异步 Images 在后台等待和执行期间不持有完整大请求体 `[]byte`。
- Embeddings 在所有 transport 错误组合下释放非空 response body。
- 用关键契约矩阵锁定容易在后续上游合并中被绕开的行为。
- 保持请求内容、协议、审计、usage、错误 envelope 和 WebSocket 行为不变。

## 非目标

- 不治理 WebSocket 首帧、重连 map 或多轮消息内存。
- 不调整 spool/preview 阈值。
- 不新增依赖、配置、数据库变更或请求体抽象。
- 不为每个平台重复同一底层 handle 测试。

## 设计

### 异步 Images 所有权

提交阶段完成现有读取、校验和安全审计后，使用现有 `RequestBodyHandle` 保存请求体。任务成功交给 `ImageTaskService.Run` 后，handle 所有权转移给 worker；转移失败则由提交方清理。

worker 从 handle 打开 request reader，并把 `GetBody` 绑定到 `handle.Open`。`newAsyncImageContext` 不再接收或闭包捕获完整 `[]byte`。worker 外层统一 defer 关闭 reader 和清理 handle，因此正常完成、失败、取消和 panic 使用同一清理路径。任务接受后发生 spool open/read 错误时，任务记录为服务不可用失败；不得回退到内存重放。

### Embeddings transport 清理

`ForwardEmbeddings` 在 `HTTPUpstream.Do` 返回后先关闭 request body。若 `err != nil` 且 response/body 非空，则同时关闭 response body，再执行现有 `ErrRequestBodySpool` 传播或普通 502 映射。该修复不改变成功和 HTTP 4xx/5xx 路径。

### WebSocket 边界

WebSocket 从升级后的首个 frame 读取消息，并跨多轮与重连维护状态，不属于 HTTP `Request.Body` 生命周期。本次仅运行既有测试并确认两个 WS 生产文件零 diff；其内存治理另行立项。

## 测试策略

严格 TDD，每项生产修复先运行 RED，再做最小 GREEN：

- 异步 Images 大 JSON/multipart 在 worker 阻塞时不保留完整 body。
- 多个并发异步大请求的 retained heap 增长有界。
- worker 正常、失败、取消、panic 和拒绝启动均清理 handle/reader。
- spool 创建失败在提交前返回 503；接受后 open/read 失败使任务失败。
- 上游收到的 body、content type 和 content length 与提交请求一致。
- Embeddings `resp+err` 关闭 body，同时保留 sentinel 和 502 语义。
- multipart `1MiB` / `1MiB+1` 生产边界。
- Grok bridge `>=64KiB` 空完成流静默拒绝保护。
- 代表性并发阻塞 heap 契约，不重复完整 25 路矩阵。

## 验证门禁

最终执行聚焦测试、unit tag handler/service 测试、全量 Go 测试、build、vet、`git diff --check`、WS 测试与 WS 文件零 diff检查。完成后由独立 reviewer 对旧规格、新完整目标规格和当前路由做只读复审。

## 风险与控制

- 异步任务所有权转移可能产生双清理或泄漏：通过单一转移点和统一 worker defer 约束。
- heap 测试可能受 GC 波动影响：复用现有 harness、比较大/小 body 增量并重复运行。
- spool 失败发生在返回 202 之后无法改写提交响应：任务以服务不可用失败结束，并通过轮询返回结果。
- 新路径可能在未来上游合并中再次绕过 handle：入口级契约测试与完整规格用于合并复核门禁。
