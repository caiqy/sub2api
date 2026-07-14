## ADDED Requirements

### Requirement: Platform Sticky State Boundaries

系统 MUST 依据请求最终平台的 Sticky 开关决定是否访问会话和响应绑定状态。OpenAI、Gemini 与 Anthropic/Antigravity 的开关彼此独立；缺失运行时配置时保持既有默认开启行为。

#### Scenario: OpenAI Sticky disabled bypasses HTTP and WebSocket state

- **WHEN** OpenAI Sticky 被关闭且请求经过 HTTP、WS V2 或 WS ingress 路径
- **THEN** 系统 MUST 不读取或写入 response-account、response-connection、turn state 和 session-connection
- **AND** 系统 MUST 继续完成不依赖这些状态的当前请求

#### Scenario: Compat selection honors the resolved platform toggle

- **WHEN** Gemini Messages compat 服务为请求解析出 Gemini、Anthropic 或 Antigravity 平台且该平台 Sticky 被关闭
- **THEN** 系统 MUST 不读取、写入、删除或刷新该请求的会话缓存绑定
- **AND** 系统 MUST 从正常候选账号选择路径继续处理请求

#### Scenario: Enabled Sticky preserves existing state behavior

- **WHEN** 请求平台的 Sticky 开关保持开启
- **THEN** 系统 MUST 保留既有会话、响应和连接状态的读写行为

#### Scenario: Anthropic Sticky disabled bypasses Antigravity cleanup

- **WHEN** Anthropic/Antigravity Sticky 被关闭且 Antigravity 重试或模型限流路径需要清理会话绑定
- **THEN** 系统 MUST 不删除会话缓存绑定
