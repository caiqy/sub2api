# request-body-retention-control Specification

## Purpose
TBD - created by archiving change optimize-request-body-retention. Update Purpose after archive.
## Requirements
### Requirement: 系统必须限制 request body 在观测链路中的长生命周期副本
系统 MUST 在所有支持 request body 的网关入口中限制 request body 和 upstream request body 在 usage detail、ops context 和异步 usage 快照中的保留方式；观测链路不得继续持有完整超大 body 副本。

#### Scenario: 普通请求进入 usage detail
- **WHEN** 网关处理任一受支持协议请求并构建 usage detail
- **THEN** 系统 MUST 记录 request/upstream body 的有界 preview、完整大小和截断状态，而不是再次完整复制 body

#### Scenario: 超大请求进入 usage detail
- **WHEN** 任一受支持入口的请求体大小超过 preview 上限
- **THEN** 系统 MUST 只保存 preview 或安全省略标记，并记录 `truncated` 与原始大小

#### Scenario: multipart 或 inline binary 请求进入观测链路
- **WHEN** Images、Videos 或其他入口收到 multipart、base64、data URL 或 inline binary 内容
- **THEN** 系统 MUST 只保存脱敏 metadata 或安全省略标记，不得将二进制正文写入 usage detail 或 ops

### Requirement: 系统必须对大请求使用可重放的文件化请求体
系统 MUST 在 Anthropic 分组 `/v1/responses` 兼容转换、Anthropic `/v1/messages`、OpenAI `/v1/chat/completions`、OpenAI Embeddings、OpenAI/Grok Images 与 Videos、Gemini `/v1beta/models/*` 的有效 body 超过 `spool threshold` 时使用临时文件承载完整请求体，并支持 failover 或 retry 时重新打开完整 reader。`spool threshold` 收紧为 `1MB`。

#### Scenario: 大 JSON 请求等待上游响应
- **WHEN** identity 编码 JSON 的有效 body 超过 `1MB`，且请求正在等待上游响应
- **THEN** 系统 MUST 让完整 body 主要驻留在临时文件，并释放同步解析阶段不再需要的完整内存副本

#### Scenario: 压缩大请求进入网关
- **WHEN** 压缩 JSON 解压后的有效 body 超过 `1MB` 且未超过解压安全上限
- **THEN** 系统 MUST 将解压流写入文件型 handle，不得先长期保留完整解压 `[]byte`

#### Scenario: multipart 大请求进入媒体入口
- **WHEN** OpenAI/Grok Images 或 Videos 收到超过 `1MB` 的 multipart 请求
- **THEN** 系统 MUST 使用文件承载大上传内容，并保持文本字段、文件 part 和上游请求语义不变

#### Scenario: failover 或 retry 需要重发请求
- **WHEN** 上游失败导致同一有效 body 需要再次发送
- **THEN** 系统 MUST 从 effective outbound handle 重新打开完整 reader，并保持发送内容一致

#### Scenario: OpenAI Responses fallback 到 raw chat
- **WHEN** Responses 上游不受支持且 raw chat fallback 需要重新发送原始 Chat Completions 请求
- **THEN** 系统 MUST 从 bound request body handle 恢复完整 body，并保持模型、URL 与发送内容一致

#### Scenario: Grok raw multipart 转换
- **WHEN** Grok moderation 或 images edit 从 raw multipart body 构造 JSON 上游请求
- **THEN** 系统 MUST 保留上传文件 bytes，并生成内容一致的 data URL

#### Scenario: 文件化失败
- **WHEN** 超过 `spool threshold` 的请求无法创建、写入、关闭、打开或读取临时文件
- **THEN** 系统 MUST 返回 503，并不得静默回退到继续持有完整 body 的高内存路径

### Requirement: 系统必须保持 usage 指纹和业务语义不变
系统 MUST 保持各受支持入口的内容审计、协议解析、模型映射、上游转发、流式终止和 usage 指纹语义不变；大请求优化不得改变计费、usage 去重、retry 或 failover 行为。

#### Scenario: 同步解析后创建 effective body
- **WHEN** handler 完成请求校验、内容审计或协议转换并生成最终上游 body
- **THEN** effective handle 的内容、hash 和上游发送结果 MUST 与优化前相同

#### Scenario: 小请求继续使用内存模式
- **WHEN** 有效 body 不超过 `1MB`
- **THEN** 系统 MUST 保持现有成功、错误、流式和计费行为，且无需创建 spool 文件

#### Scenario: 大请求落盘后同步解析
- **WHEN** 有效 body 超过 `1MB` 且 handler 的同步解析阶段需要完整 body
- **THEN** 系统 MUST 允许在同步解析窗口内短暂物化完整 body，并在解析完成后尽快释放该内存副本，不得跨越 failover 或上游等待循环长期持有

### Requirement: 系统必须在所有终止路径释放 request body 临时资源
系统 MUST 明确管理 raw inbound handle、effective outbound handle 和 multipart 临时文件的 ownership，并在资源不再使用时完成清理。

#### Scenario: 请求成功或上游返回错误
- **WHEN** 请求成功完成或以上游 4xx/5xx 结束
- **THEN** 系统 MUST 清理该请求拥有的所有 spool 和 multipart 临时文件

#### Scenario: 客户端取消或 handler 提前返回
- **WHEN** 客户端取消、业务校验失败、路由失败或 panic recovery 导致 handler 提前结束
- **THEN** 系统 MUST 清理已创建的临时资源且不得影响错误响应语义

#### Scenario: retry 替换 effective body
- **WHEN** 模型映射、协议转换或 retry 为下一 attempt 创建新的 owned effective handle
- **THEN** 系统 MUST 在旧 handle 不再被请求使用后清理旧资源，并保留新 handle 直到 attempt 完成

### Requirement: 系统必须在上游等待期间不持有完整请求体内存副本
系统 MUST 在请求发送到上游至收到响应（或流式终止）期间，不持有完整请求体或改写后请求体的 `[]byte` 内存副本；该窗口内完整 body 只允许存在于 spool 临时文件或以流式 reader 形式被上游 transport 消费。

#### Scenario: 请求进入上游等待
- **WHEN** 任一支持 request body 的网关入口完成同步解析、校验、审计与模型映射，并将请求发送给上游后进入等待响应阶段
- **THEN** 系统 MUST 释放 handler 持有的完整 body 内存副本，仅保留可重放的 handle（内存或 spool 文件）与流式上游 reader

#### Scenario: failover 重试需要重新发送
- **WHEN** 上游失败且 failover 循环需要为下一个账号重新派生请求体
- **THEN** 系统 MUST 从 effective outbound handle 按需物化完整 body，完成账号级改写后立即释放该内存副本，不得跨 attempt 长期持有

#### Scenario: 上游等待期间并发大请求
- **WHEN** 多个大请求同时处于上游等待阶段
- **THEN** 系统 MUST 不因请求体驻留使内存随并发大请求数线性增长，body 内存副本不得跨上游等待期存活

### Requirement: 系统必须保持请求预览有界
系统 MUST 将请求预览（preview）限制为有界摘要，任何请求的 preview 内存副本不得超过 `preview limit`，且 `preview limit` 不得大于 `spool threshold`。

#### Scenario: 请求预览截断
- **WHEN** 请求体超过 `preview limit`（256KB）
- **THEN** 系统 MUST 只保留截断后的预览摘要，并记录 `truncated` 标记与原始大小

#### Scenario: preview 与 spool 边界一致
- **WHEN** 请求体大小位于 `spool threshold`（1MB）与旧 `preview limit`（5MB）之间
- **THEN** 系统 MUST 不因 preview 保留完整请求体副本；preview 上限不得超过 spool 阈值

