# Outcome

复审已归档的 `reduce-request-body-memory-retention` 行为，并修复后来新增路径绕过请求体生命周期契约的问题。完成后，所有非 WebSocket 请求入口在上游等待或异步执行期间均不得长期持有完整大请求体 `[]byte`，transport 错误路径必须释放已返回的响应体，关键行为由稳定的回归测试锁定。

# Scope

- 修复异步 Images 提交路径跨后台任务持有完整请求体的问题，使用 worker-owned `RequestBodyHandle` 承载可重放 body。
- 修复 Embeddings 在 `HTTPUpstream.Do` 同时返回 response 和 error 时未关闭 `resp.Body` 的问题。
- 补齐异步 Images 大 JSON、multipart、并发阻塞、资源清理和 spool 失败测试。
- 补齐生产 multipart 的 `1MiB` / `1MiB+1` 边界、Grok Chat-to-Responses `>=64KiB` 静默拒绝保护和 Embeddings response close 测试。
- 对旧请求体治理规格和当前非 WebSocket 路由做独立复审；WebSocket 仅检查共享代码交叉影响。

# Non-goals

- 不治理 WebSocket 连接内首帧、重连 map 或多轮消息的内存生命周期。
- 不修改 WebSocket 生产文件。
- 不调整 `1MiB` spool 阈值、`256KiB` preview 上限或现有 API 行为。
- 不新增依赖、配置项、数据库 schema 或新的请求体抽象。
- 不改写已归档的 Classic change 产物。

# Acceptance examples

- 当异步 Images 收到超过 `1MiB` 的 JSON 或 multipart body 并进入后台等待时，完整 body 由可重放文件型 handle 承载，提交闭包和 request `GetBody` 不捕获完整 `[]byte`。
- 当多个异步 Images 大请求同时阻塞时，保留堆内存不随完整 body 大小线性增长。
- 当异步 worker 正常完成、失败、取消、panic 或拒绝启动时，其拥有的 handle 和打开的 reader 均被清理。
- 当异步请求在提交前无法创建 spool 时，接口返回 503 且不创建任务；任务接受后无法打开或读取 spool 时，任务以服务不可用失败结束且不降级为内存重放。
- 当 Embeddings transport 同时返回 response 和 error 时，系统关闭非空 `resp.Body`，同时保持现有 spool sentinel 和普通 502 错误语义。
- 当 Grok Chat-to-Responses bridge 处理原请求体长度不小于 `64KiB` 的空完成流时，静默拒绝保护继续生效。
- 当 multipart 有效 body 分别为 `1MiB` 与 `1MiB+1` 时，前者保持内存模式，后者进入 spool 模式。

# Constraints and invariants

- 严格 TDD：每项生产修复必须先有可重复失败的 RED 测试，再做最小修复。
- 异步 Images worker 接受 handle 所有权后，负责所有终止路径的清理；提交方仅在所有权尚未转移时清理。
- 上游收到的 body、`Content-Type`、`Content-Length`、模型映射、审计和 usage 语义保持不变。
- `ErrRequestBodySpool` 必须保留错误链，不得静默转换为普通 transport 错误或回退到完整内存 body。
- `backend/internal/service/openai_ws_forwarder_v2.go` 与 `backend/internal/service/openai_ws_protocol_forward.go` 保持零 diff。

# Decisions

- 使用独立 Native change `harden-request-body-retention-regressions`，旧 Classic 归档仅作为行为基线。
- 两项已确认生产缺陷均纳入本次修复；测试采用关键契约矩阵，不重复所有平台的同类底层测试。
- WebSocket 只做共享代码交叉影响检查，不纳入本次内存治理。
- 异步 Images 复用现有 `RequestBodyHandle`，不新增队列 payload 抽象。
- 用户已确认范围、所有权、测试矩阵和验证门禁。

# Open questions

无。用户于 2026-08-05 确认本 brief 所述目标与边界。

# Verification expectations

- 聚焦 RED/GREEN 测试记录每项缺陷的失败与通过证据。
- 重复运行异步/并发 heap 测试，确认结果不是偶然通过。
- 运行 `go test -tags=unit ./internal/handler ./internal/service -count=1`。
- 运行 `go test ./... -count=1`、`go build ./...`、`go vet ./...` 和 `git diff --check`。
- 运行现有 WebSocket 测试，并确认两个 WebSocket 生产文件相对 change baseline 零 diff。
- 完成一轮独立只读复审，逐条核对旧规格、新完整目标规格和当前路由。
