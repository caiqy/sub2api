---
comet_change: optimize-request-body-retention
role: technical-design
canonical_spec: openspec
---

# 请求体驻留优化技术设计

## 背景

dmit 线上 OOM 排查表明，超大 `/responses` 请求的问题不只在内容审计和 usage 队列的额外副本，还在于完整 request body 会从入口读入后一直活到上游响应返回、usage detail 快照构建和异步 usage 提交完成。上游响应经常需要 10 秒以上，这让大 body 在小内存机器上形成长时间驻留。

当前主要问题：

- `UsageDetailCapture` 在 handler 前完整读取 request body，并同时保存 `string` 和 replay `[]byte`。
- handler 层会再次持有完整 `body []byte`，直到上游返回后才计算 `requestPayloadHash`。
- 上游 request capture 和 ops upstream capture 会再次从 `req.GetBody()` 读出完整 body。
- 大请求在等待上游期间，完整 body 继续以 request body、usage detail、ops context 等多种形态存活。
- `requestPayloadHash` 和 usage detail 构建过晚，迫使 handler 长时间保留完整 body。

## 目标

- 所有 `/responses` 请求都减少 request body 在观测链路中的重复副本和长生命周期引用。
- `>10MB` 请求在等待上游期间，将完整 body 从 RAM 转为临时文件承载。
- `requestPayloadHash` 仍基于当前有效转发 body 计算，保持现有计费与去重语义。
- usage detail / ops 只保留有界 preview，默认预览上限 `5MB`。
- 不改变请求校验、内容审计、模型映射、上游转发、计费与 failover 的业务语义。

## 非目标

- 不做流式 JSON 解析。
- 不改 response body / upstream response body 的捕获策略。
- 不改 Anthropic、Gemini 等其他协议入口。
- 不新增数据库 schema；超大 body 的元信息继续编码在现有 detail 文本里。

## 默认值

- `spool threshold = 10MB`
- `preview limit = 5MB`

两者独立：`10MB` 用于决定是否进入临时文件模式；`5MB` 用于约束 request/upstream body 的观测预览大小。

## 方案

### 1. 引入 RequestBodyHandle

新增一个小型 `RequestBodyHandle`，统一表示请求体来源。它不是抽象体系，只是一个集中管理 body 生命周期的小结构体。

职责：

- 保存完整 body 的来源：内存或临时文件。
- 在创建时一次性生成：完整字节数、完整 `sha256`、有界 preview。
- 提供重新打开完整 reader 的能力，供上游 request body 和 `GetBody` 使用。
- 提供 `Cleanup()`，负责临时文件删除。

行为：

- `<=10MB`：使用内存后端。
- `>10MB`：使用临时文件后端。
- preview 一律限制在 `5MB` 内，附带 `truncated` 和 `total_bytes` 元信息。

### 2. handler 分成 raw/effective 两阶段

`OpenAIGatewayHandler.Responses` 不再把完整 `body []byte` 当作整个请求生命周期的主载体，而是分成两个阶段：

#### 2.1 raw request handle

从原始 `Request.Body` 创建 `rawHandle`，同时得到：

- request preview
- request size
- raw request hash（仅供诊断，不直接用于 usage）

随后 handler 在短生命周期内读取完整 bytes，完成：

- JSON 合法性校验
- 内容审计
- compact normalize
- session/cyber 相关输入计算

#### 2.2 effective forward handle

如果前置处理未改 body，则 `effectiveHandle = rawHandle`；
如果前置处理改了 body，则根据改写后的 bytes 再创建 `effectiveHandle`。

约束：

- `requestPayloadHash` 必须在进入转发前基于 `effectiveHandle` 计算。
- 这样可保持当前计费、usage 去重和日志定位语义不变。
- 一旦 `effectiveHandle` 就绪，handler 不再依赖完整 `body []byte` 穿过整个上游等待期。

### 3. 观测链路只保留 preview

#### 3.1 request body capture

`UsageDetailCapture` 不再在 middleware 中预读完整 request body。它只负责：

- request headers
- response headers / response body

request body preview 改由 handler 在创建 `rawHandle` 后显式写回 collector。

这样可删除现在的双副本模式：

- `collector.requestBody string`
- replay `[]byte`

#### 3.2 upstream request capture

`SetUsageUpstreamRequest` 和 ops upstream request capture 不再自行 `ReadAll(req.GetBody())`。

上游 request preview 在构造 request 前从 `effectiveHandle` 直接生成，并显式传入：

- usage detail 只保存 upstream preview
- ops context 只保存 upstream preview

这样可消除“为了观测再次读取完整上游 body”的重复副本。

### 4. `>10MB` 进入文件化发送模式

对 `effectiveHandle` 大于 `10MB` 的请求：

- 完整 body 写入受管临时文件。
- 构造上游 request 时，body reader 改为 `handle.Open()`。
- `req.GetBody` 也改为重新 `Open()`，支持 failover / retry。

收益：

- 大请求在等待上游那 10 多秒期间，完整 body 主要驻留在临时文件，而不是 RAM。
- 比依赖 swap 更可控；body 是已知冷数据，可以主动下沉到磁盘。

### 5. 清理与失败策略

#### 5.1 清理

- `rawHandle` / `effectiveHandle` 都由 handler `defer Cleanup()`。
- 文件型 handle 的 `Cleanup()` 负责关闭和删除临时文件。
- 文件生命周期只覆盖本次请求及其同步重试窗口，不依赖 async usage worker。
- 增加 best-effort stale file sweep，清理明显过期的残留 spool 文件。

#### 5.2 失败

对 `>10MB` 请求，如果临时文件创建或写入失败：

- 不回退到“继续把完整 body 长时间留在 RAM”。
- 直接返回 `503 api_error`。
- 记录 ops 事件，如 `large_request_spool_failed`。

这样可避免线上在最危险的压力场景中悄悄回退回 OOM 路径。

## 受影响模块

- `backend/internal/handler/openai_gateway_handler.go`
- `backend/internal/server/middleware/usage_detail_capture.go`
- `backend/internal/service/usage_detail_capture.go`
- `backend/internal/service/ops_upstream_context.go`
- `backend/internal/service/openai_gateway_service.go`
- 新增 request body handle 所在文件

## 风险与取舍

- `preview limit=5MB` 仍然不小，但已明显低于完整大 body；接受这部分内存与排障可读性的折中。
- `>10MB` 请求在磁盘慢或空间紧张时可能更容易失败；这是显式 trade-off，优先保护进程整体可用性。
- 非透传路径中 `map[string]any` 仍可能在同步转发窗口内存在；本设计先收掉长生命周期完整 body，不做流式 JSON 改造。
- usage detail 不再保留完整超大 request/upstream body；管理员看到的是 preview + 元信息，而不是完整正文。

## 测试策略

- `RequestBodyHandle`：内存模式、文件模式、hash、preview、size、重复 `Open()`、cleanup。
- `UsageDetailCapture`：不再预读完整 request body，downstream 仍能读取完整请求。
- `OpenAIGatewayHandler.Responses`：`requestPayloadHash` 与现状一致；usage / ops 只保留 preview。
- 上游 request 构造：文件型 body 的 `GetBody` 可重复打开，failover / retry 内容一致。
- 错误路径：spool 创建失败返回 `503`，并有清理和 ops 记录。
