---
comet_change: extend-request-body-spooling
role: technical-design
canonical_spec: openspec
---

# 扩展请求体文件化技术设计

## 背景

现有 `RequestBodyHandle` 已在 OpenAI/Grok `/responses` 路径提供 10MB spool、5MB preview、完整 SHA-256、重复 `Open()` 和清理能力。其他入口仍通过 `ReadRequestBodyWithPrealloc` 将完整请求体读入 `[]byte`，并让该切片跨越内容审计、账号选择、上游等待、retry/failover 和 usage 提交阶段。

本变更把已有机制扩展到 Anthropic、OpenAI、Grok 和 Gemini 的剩余大 body 入口。OpenSpec delta spec 是行为事实源；本文只描述实现边界、数据流、资源 ownership 和验证方式。

## 设计约束

- spool threshold 固定为 10MB，preview limit 固定为 5MB。
- 压缩 JSON 的 threshold 按解压后大小计算，并继续受 64MB 解压上限约束。
- multipart 的 raw threshold 按客户端发送的原始请求体大小计算。
- 不实现流式 JSON parser；同步校验、审计和协议转换可短暂持有完整 JSON。
- 不新增第三方依赖、配置项、数据库 schema 或协议层接口体系。
- 小请求、错误格式、计费、模型映射、流式终止、retry/failover 和错误透传语义保持不变。

## 现有基础

`service.RequestBodyHandle` 已具备本变更需要的存储原语：

- `NewRequestBodyHandleFromReader` 在读取过程中计算 size、hash 和 preview，超过 threshold 时把已缓冲内容迁移到临时文件。
- `Open` 为内存或文件后端返回新的 reader；文件打开和读取错误包装为 `ErrRequestBodySpool`。
- `ReadAll` 只用于同步解析窗口。
- `CleanupRequestBodyHandle` 负责幂等清理及删除失败重试。
- `openAIRequestBodyHandleForBytes` 已实现 size/hash 相同则复用、不同则创建 effective handle 的模式。

现有 `UsageDetailCapture` 已不再预读 request body，handler 可通过 `SetUsageRequestBody` 写入不可变 snapshot。扩展路径必须继续使用该模式，不能为观测重新打开完整 handle。

## 总体结构

### 1. 共享 coordinator

在 handler 包新增一个小型 request body coordinator。它是普通结构体和辅助函数，不定义接口、工厂或协议回调。

它只负责：

- 从 HTTP request 创建 raw inbound handle。
- 在同步解析时按需读取 raw bytes。
- 根据最终上游 bytes 或 reader 设置 effective outbound handle。
- 通过 size/hash 复用内容相同的 raw handle。
- 替换 effective handle 时清理不再使用的 owned handle。
- 对 raw、effective 和 multipart form 做去重且幂等的最终清理。

协议解析、内容审计、模型映射、账号选择、计费和错误响应仍留在各 handler。coordinator 不依赖 Gin response writer，也不决定 HTTP 状态码。

建议保持最小 API 面：

```go
type requestBodyCoordinator struct {
    raw       *service.RequestBodyHandle
    effective *service.RequestBodyHandle
    form      *multipart.Form
}

func newJSONRequestBody(req *http.Request) (*requestBodyCoordinator, error)
func newMultipartRequestBody(req *http.Request, maxMemory int64) (*requestBodyCoordinator, error)
func (b *requestBodyCoordinator) ReadRaw() ([]byte, error)
func (b *requestBodyCoordinator) SetEffectiveBytes(body []byte) error
func (b *requestBodyCoordinator) SetEffectiveReader(r io.Reader) error
func (b *requestBodyCoordinator) Effective() *service.RequestBodyHandle
func (b *requestBodyCoordinator) Cleanup()
```

具体命名可在实现时贴合现有 handler 约定，但职责不得扩展为通用请求框架。

### 2. 解码 reader

将 `httputil.ReadRequestBodyWithPrealloc` 内部的 Content-Encoding 分支拆出为共享 decoded reader 构造函数：

- identity：直接返回原始 `req.Body`。
- gzip/x-gzip、deflate、zstd：返回对应 decoder，并在外层施加 64MB 解压上限。
- 不支持或损坏的编码：保留当前 400 行为。
- 解压超过上限：保留当前 413 行为。

JSON coordinator 把 decoded reader 直接传给 `NewRequestBodyHandleFromReader`，并在完成后关闭 decoder。只有 handle 创建成功后，才删除 `Content-Encoding`、更新 `Content-Length` 并把有效大小写回 request metadata，避免失败请求留下半更新状态。

原有 `ReadRequestBodyWithPrealloc` 继续调用同一 decoded reader，避免未迁移入口出现解码语义分叉。

### 3. raw 与 effective handle

raw handle 表示 handler 实际解析的入站内容：JSON 为解压后的有效 body，multipart 为未经解析的原始 body。它提供 request preview、size 和 hash。

effective handle 表示当前 attempt 最终发送给上游的完整 body：

- 未改写时，effective 与 raw 指向同一 handle。
- 模型映射、协议转换或 multipart 重建改变内容时，根据最终内容创建新 handle。
- size 和 hash 都相同才允许复用；仅 size 相同不能复用。
- retry/failover 从 effective handle 的 `Open()` 获取新 reader，不从 raw bytes、usage snapshot 或 `req.GetBody` 的旧闭包重建。

每个上游 request 的 `Body` 和 `GetBody` 都绑定同一个 effective handle。若后续 attempt 创建新 effective handle，先确保旧 request 已关闭，再替换并清理旧的 owned handle。

现有 OpenAI service 已区分借用 handle 与 attempt 内新建的 owned handle：借用 handle 由 handler/coordinator 清理，派生 handle 通过 request context 转交给具体 HTTP request，并由 `closeOpenAIRequestBody` 清理。其他迁移路径沿用这两个 ownership 类别，不再引入第三套引用计数。

## 数据流

### JSON 入口

适用于 Anthropic `/v1/messages`、Anthropic 分组 `/v1/responses` 兼容入口、OpenAI Chat/Embeddings 及 Gemini `/v1beta/models/*`：

1. handler 创建 JSON coordinator；decoded reader 直接写入 raw handle。
2. 立即把 raw handle 的 preview snapshot 写入 usage/ops。
3. `ReadRaw` 在同步窗口 materialize JSON，执行现有合法性校验、内容审计、session 计算和协议转换。
4. handler 得到最终上游 body 后调用 `SetEffectiveBytes`；内容未变则复用 raw handle。
5. 计算 usage 指纹时使用 effective handle 的 hash，或保持现有协议明确要求的等价输入。
6. 转发 service 改为接收 handle 或 reader，而不是让 handler 的完整 `[]byte` 跨越 failover 循环。确需按账号转换时，在 attempt 构建 helper 内短暂 `ReadAll`、转换并创建派生 handle，构造 request 后立即释放局部 bytes。
7. 每次 attempt 从 effective handle 打开 reader；retry/failover 重复打开。协议转换生成的新 effective handle 按 request ownership 清理。
8. handler defer 在所有返回路径执行 coordinator cleanup。

JSON parser 产生的对象可能仍短时持有字符串和数组。实现不得把这些对象写入 Gin context、usage snapshot 或异步任务；进入上游等待前只保留路由、计费和转发必需的小字段。仅执行 `body = nil` 不作为释放保证；长生命周期循环不得再捕获该切片，materialize 与转换应放在返回 handle 的短生命周期 helper 中。

### multipart 媒体入口

适用于 OpenAI/Grok Images 与 Videos：

1. coordinator 从原始 `req.Body` 创建 raw handle，不做 Content-Encoding 解码。
2. 从 raw handle `Open()` 得到 reader，使用现有 boundary 和标准库 multipart parser。
3. 使用 `ParseMultipartForm` 的既有 memory limit；超过该 limit 的文件 part 由标准库落到临时文件。
4. coordinator 记录返回的 `multipart.Form`，最终 cleanup 必须调用 `RemoveAll()`。
5. usage/ops 只写入模型、prompt、尺寸、文件数量、是否含源图/遮罩等脱敏 metadata，不写 raw preview、base64、data URL 或文件正文。
6. 构建上游 multipart 时，由 `multipart.Writer` 写入 `io.Pipe`，`NewRequestBodyHandleFromReader` 从 pipe 消费并形成 effective handle。生产端错误通过 `CloseWithError` 传给消费端。
7. 上游 request 从 effective handle 打开 reader，并使用 writer 生成的 Content-Type boundary 与 handle size。

pipe 只用于同步构建 effective handle，不跨上游网络等待期。这样无需增加 writer 型 handle API，也不会额外构造完整 multipart `[]byte`。

### JSON 媒体入口与 inline binary

媒体 JSON 请求复用 JSON coordinator，但观测 snapshot 必须先走现有脱敏逻辑。即使整个 JSON 小于 5MB，只要包含 base64、data URL 或 inline binary，也不能把通用 JSON preview 写入 usage/ops。

## Ownership 与清理

coordinator 是 handler 生命周期内 raw handle 和基础 effective handle 的根 owner。创建成功后立即 `defer Cleanup()`，不得把 cleanup 依赖于正常转发结束。attempt 构建阶段若创建派生 effective handle，则显式转交给具体 HTTP request，沿用现有 owned-handle context 机制。

清理规则：

- raw 与 effective 是同一指针时只清理一次。
- effective 替换时，仅清理 coordinator 独占且未转交给当前 request 的旧 handle。
- 上游 request body reader 由 HTTP transport 或调用方关闭；handle 只在该 attempt 完全结束后清理。
- request context 标记的 owned handle 由 request close helper 清理；借用 coordinator handle 的 request 不执行 handle cleanup。
- multipart form 在 raw/effective handle 之前或之后清理均可，但必须在 handler 返回前执行 `RemoveAll()`。
- `CleanupRequestBodyHandle` 保持幂等并继续使用 stale sweep 作为异常退出兜底；stale sweep 不能替代正常 cleanup。
- panic recovery、客户端取消、业务拒绝、路由失败、上游 4xx/5xx 和流式结束均经过同一个 defer。

禁止把 coordinator 或 handle 交给 usage 异步 worker。若现有异步路径必须读取请求信息，只能传 immutable bounded snapshot、size 和 hash。

## 各入口接入边界

### Anthropic

- `/v1/messages`：raw bytes 继续供现有 Anthropic 校验、内容审计、session 与模型解析；最终 Anthropic body 绑定 effective handle。
- Anthropic 分组 `/v1/responses`：raw Responses body 完成兼容转换后，以转换出的 Messages body 创建 effective handle；计费与 usage 指纹沿用当前兼容路径定义。

### OpenAI

- Chat Completions：保留 `gjson` 校验、stream 解析、内容审计、session hash 与 channel mapping；映射后的 body 成为 effective body。
- Embeddings：保留模型校验和 channel mapping；每个 failover attempt 重用当前 effective handle，不再长期捕获入口 `body`。
- 复用现有 `BindOpenAIRequestBodyHandle`、`openAIRequestBodyHandleForBytes`、`openAINewRequestWithBodyHandle` 和 owned request context；只扩展适用入口，不另建平行 ownership 机制。

### Gemini

- `generateContent`、`streamGenerateContent`、`countTokens` 共用 JSON coordinator。
- 模型路径解析、Google 错误格式、failed usage、流式终止和 Antigravity 强制平台路由保持原位。
- URL path 中的模型与 body 转换结果不纳入 raw body hash；最终发送 body 仍由 effective handle 表示。

### OpenAI/Grok 媒体

- Images generate/edit、Videos create 处理 body，接入 coordinator。
- 仅查询状态或下载且没有 request body 的媒体路由不创建 handle。
- Grok/OpenAI 不同上游格式由现有 service 决定；coordinator 只接收最终 reader 和 Content-Type。

## 错误映射

错误在 handler 边界统一分类：

- `errors.Is(err, service.ErrRequestBodySpool)`：若尚未写响应，返回 503；记录现有 request/ops 错误上下文，但不包含 body 正文。
- `*http.MaxBytesError` 或等价解压超限：返回现有 413 格式。
- 不支持、损坏的 Content-Encoding：返回现有 400 格式。
- 空 body、JSON/multipart 格式错误：保持各协议当前 400 错误类型和消息。
- multipart `RemoveAll` 或最终 spool 删除失败：主响应已确定时只记录清理错误并安排现有 retry，不覆盖业务响应。
- effective handle 在发送前无法 `Open()`：返回 503；在已开始流式响应后沿用当前流式终止语义，不尝试写第二个 HTTP 错误。

不允许 spool 失败后回退到完整内存 body。

## 观测与安全

- JSON request snapshot 直接来自 raw handle 的 `PreviewString()`，包含 size/truncated 元信息。
- upstream snapshot 直接来自 effective handle 的 preview，不调用 `GetBody` 或 `ReadAll`。
- ops 单字段继续受 20KB 上限约束。
- multipart 和 inline binary 使用专用 metadata snapshot，fail-closed；无法确定是否安全时省略正文。
- hash、size、截断标记可用于定位一致性问题，但日志不得记录 spool 路径或 body 正文。

## 测试策略

### 共享原语

- identity、gzip/x-gzip、deflate、zstd 的正常输入与损坏输入。
- 10MB 以下、等于 10MB、超过 10MB 的内存/文件模式边界。
- 5MB preview 边界、完整 size/hash、重复 `Open()` 和 `ReadAll()`。
- 解压后刚好 64MB 与超过 64MB 的行为。
- create/write/close/open/read spool 失败均可被 `errors.Is` 识别，并映射到 503。
- cleanup 幂等、删除失败 retry 和 stale sweep。

### 生命周期

- 成功、业务校验拒绝、账号选择失败、上游 4xx/5xx、客户端取消和 panic recovery 后目录无残留。
- raw/effective 相同只清理一次；内容改变时两者都清理。
- retry/failover 每次读取内容、size 和 hash 一致；替换 effective handle 后旧 handle 不残留。
- 流式 request 在结束或中断后清理，但不会在 transport 仍读取时提前删除。
- multipart parser 临时文件和 raw/effective spool 均被清理。

### 协议回归矩阵

- Anthropic Responses/Messages：小、大、压缩、转换、流式、内容审计与 failover。
- OpenAI Chat/Embeddings：模型映射、stream 字段、retry/failover、usage 指纹和错误透传。
- Gemini：三个目标 action、模型路径、Google 错误、failed usage 与 Antigravity 路由。
- OpenAI/Grok Images/Videos：JSON、multipart、源图、遮罩、inline binary、生成/编辑/创建及上游失败。

每类至少保留一个可运行测试，证明进入模拟上游等待后 usage/ops 不持有完整 body，spool 文件存在且请求结束后删除。

### 全量与端侧验证

- 后端 `go test ./... -count=1`。
- 前端 `pnpm test:run` 与 `pnpm typecheck`，确认 usage detail 展示契约未回归。
- 受控环境执行 5MB、10MB、12MB identity、gzip、multipart 请求，对照客户端发送 hash、上游接收 hash、usage detail、ops 和 spool 目录。
- 在上游延迟窗口采样容器 RSS，验证大 body 不再形成长期 `[]byte` 驻留；本设计不承诺消除同步解析瞬时峰值。

## 实施顺序与回滚

1. 先增加 decoded reader 与 coordinator 的失败优先测试，再实现共享原语。
2. 迁移 Anthropic/OpenAI JSON 入口并运行定向测试。
3. 迁移 Gemini。
4. 最后迁移 multipart/媒体路径，因为其 ownership 与上游 body 构建最复杂。
5. 每批完成后运行对应测试，全部完成后执行全量和端侧矩阵。

部署不需要数据迁移。回滚为上一镜像即可；正常 cleanup 与现有 stale sweep 负责清理旧进程可能留下的 spool 文件。

## 已接受风险

- 同步 JSON 解析仍会出现短暂内存峰值；本次只消除上游等待和异步观测阶段的长期驻留。
- multipart 解析期间可能短时同时存在 raw spool、标准库 part 文件和 effective spool；通过严格 ownership 缩短重叠窗口。
- 文件系统空间或 I/O 异常会让大请求显式失败为 503；这是保护进程整体可用性的预期取舍。
- 跨协议迁移面较大；按协议分批接入和保留独立回归测试控制风险。
