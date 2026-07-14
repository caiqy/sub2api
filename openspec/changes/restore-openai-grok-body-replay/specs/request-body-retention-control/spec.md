## MODIFIED Requirements

### Requirement: 系统必须对大请求使用可重放的文件化请求体
系统 MUST 在 Anthropic 分组 `/v1/responses` 兼容转换、Anthropic `/v1/messages`、OpenAI `/v1/chat/completions`、OpenAI Embeddings、OpenAI/Grok Images 与 Videos、Gemini `/v1beta/models/*` 的有效 body 超过 `spool threshold` 时使用临时文件承载完整请求体，并支持 failover 或 retry 时重新打开完整 reader。

#### Scenario: 大 JSON 请求等待上游响应
- **WHEN** identity 编码 JSON 的有效 body 超过 `10MB`，且请求正在等待上游响应
- **THEN** 系统 MUST 让完整 body 主要驻留在临时文件，并释放同步解析阶段不再需要的完整内存副本

#### Scenario: 压缩大请求进入网关
- **WHEN** 压缩 JSON 解压后的有效 body 超过 `10MB` 且未超过解压安全上限
- **THEN** 系统 MUST 将解压流写入文件型 handle，不得先长期保留完整解压 `[]byte`

#### Scenario: multipart 大请求进入媒体入口
- **WHEN** OpenAI/Grok Images 或 Videos 收到超过 `10MB` 的 multipart 请求
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
