## ADDED Requirements

### Requirement: 流式请求按明确生图意图选择首 Token 超时
系统 MUST 仅对 OpenAI Responses 流式请求启用首 Token 超时，并 MUST 仅在 `tool_choice.type` 明确等于 `image_generation` 时选择图片超时档；工具列表中仅存在 `image_generation` MUST NOT 被视为明确生图。

#### Scenario: 自动工具选择仍使用生文档
- **WHEN** 流式 Responses 请求的 `tools` 包含 `image_generation` 且 `tool_choice` 为 `auto` 或未指定
- **THEN** 系统 MUST 使用生文首 Token 超时

#### Scenario: 强制图片工具使用生图档
- **WHEN** 流式 Responses 请求的 `tool_choice.type` 等于 `image_generation`
- **THEN** 系统 MUST 使用生图首 Token 超时

#### Scenario: 非流式请求不启用保护
- **WHEN** Responses 请求的 `stream` 为 `false`
- **THEN** 系统 MUST NOT 启动本能力的首 Token watchdog

### Requirement: 首 Token 等待采用业务事件边界
系统 MUST 从上游请求真正发出前开始计时，并 MUST 覆盖等待响应头及等待首个业务事件的阶段。`response.created` 与 `response.in_progress` MUST 被视为前导事件，不得结束等待。

#### Scenario: 只有前导事件时继续等待
- **WHEN** 上游已返回响应头并仅发送 `response.created` 或 `response.in_progress`
- **THEN** 系统 MUST 继续首 Token 计时直至业务事件或超时

#### Scenario: 文本或工具输出结束等待
- **WHEN** 上游发送首个文本 delta、reasoning delta、函数调用或其他输出项事件
- **THEN** 系统 MUST 停止首 Token watchdog 并记录首 Token 延迟

#### Scenario: 图片输出项结束等待
- **WHEN** 上游发送 `response.output_item.added` 且 `item.type` 为 `image_generation_call`
- **THEN** 系统 MUST 停止首 Token watchdog，并 MUST NOT 用该 watchdog 限制后续图片生成总时长

#### Scenario: 无业务输出的终态不伪造首 Token
- **WHEN** 上游在业务输出前发送 failed、completed、incomplete 或 canceled 终态
- **THEN** 系统 MUST 停止 watchdog，但 MUST NOT 将该终态记录为首 Token

#### Scenario: Control 或未知事件不结束等待
- **WHEN** 上游在业务输出前发送 `session.updated`、`rate_limits.updated` 或未知 control 事件
- **THEN** 系统 MUST 继续首 Token 计时，且 MUST NOT 记录首 Token 延迟

#### Scenario: 通用流间隔超时在首业务事件后启动
- **WHEN** 首 Token watchdog 仍在等待，包括 OAuth passthrough 或 Responses→Chat fallback 路径
- **THEN** `stream_data_interval_timeout` MUST NOT 抢先结束请求或惩罚账号；首业务事件到达后 MUST 从该时刻开始执行流间隔保护

### Requirement: 首 Token 超时可运行时配置
系统 MUST 提供 `openai_text_first_token_timeout` 和 `openai_image_first_token_timeout` 两个秒级 Gateway 运行时设置，默认值 MUST 分别为 30 和 600；值为 `0` MUST 关闭对应超时，负数 MUST 被拒绝。

#### Scenario: 默认配置生效
- **WHEN** 部署未显式配置两个首 Token 超时字段
- **THEN** 系统 MUST 对生文使用 30 秒，对明确生图使用 600 秒

#### Scenario: 零值关闭对应超时
- **WHEN** 管理员将任一首 Token 超时设置为 `0`
- **THEN** 系统 MUST 对对应类别的新请求或新 WebSocket turn 关闭首 Token 超时

#### Scenario: 旧运行时设置保持兼容
- **WHEN** 持久化的 Gateway 运行时 JSON 不包含新增字段
- **THEN** 系统 MUST 保留当前配置来源提供的默认值，而不是用零值覆盖

#### Scenario: 管理端修改配置
- **WHEN** 管理员在“网关服务”设置页保存两个非负超时值
- **THEN** 运行时设置 API MUST 持久化并返回这些值，且新请求 MUST 使用更新后的值

### Requirement: HTTP SSE 超时直接返回不可重试错误
系统 MUST 在 HTTP SSE 首 Token 超时时取消当前上游请求，并 MUST 返回 HTTP 504、错误类型 `first_token_timeout`。该错误 MUST NOT 进入账号 failover。

#### Scenario: 响应头前超时
- **WHEN** 上游在所选首 Token deadline 前未返回响应头
- **THEN** 系统 MUST 取消上游请求并向客户端返回 HTTP 504 `first_token_timeout`

#### Scenario: 前导事件后超时
- **WHEN** 上游已返回响应头和前导事件，但在 deadline 前没有业务事件
- **THEN** 系统 MUST 取消流读取并向客户端返回一次 HTTP 504 `first_token_timeout`

#### Scenario: 业务事件与超时竞争
- **WHEN** 首个业务事件与 timeout deadline 并发到达
- **THEN** 系统 MUST 只提交一种终态，不得同时写入 SSE 业务输出和 504 错误

### Requirement: WebSocket 超时取消并清理当前 response
系统 MUST 为每个 Responses WebSocket `response.create` 独立计时。超时时系统 MUST 向上游发送 `response.cancel`，向下游发送一次符合 Responses schema 的 `first_token_timeout` error，并清理当前 response。

#### Scenario: cancel 后确认终态并复用连接
- **WHEN** 首 Token 超时后 `response.cancel` 发送成功，且上游在 drain 窗口内返回 canceled、failed、completed 或 incomplete 终态
- **THEN** 系统 MUST 结束当前 turn，并 MUST 允许健康的上游连接用于后续 turn

#### Scenario: cancel 或 drain 失败时废弃连接
- **WHEN** `response.cancel` 写入失败、drain 窗口内没有终态或 response 归属无法确认
- **THEN** 系统 MUST 将上游连接标记为不可复用并关闭

#### Scenario: 多 turn relay 在清理成功后继续
- **WHEN** V2 passthrough relay 的一个 turn 首 Token 超时且 cancel/drain 成功
- **THEN** relay MUST 清除该 turn 的 watchdog 状态，并 MUST 能处理下一条 `response.create`

#### Scenario: 后续 turn 静默时仍能超时
- **WHEN** V2 passthrough relay 已完成一个 turn，随后发送新的 `response.create` 且上游不再发送任何事件
- **THEN** 新 turn 的首 Token deadline MUST 唤醒 relay 并执行 cancel，不得因 reader 已阻塞而失效

#### Scenario: 异 response 事件不得归属当前 turn
- **WHEN** 当前 turn 尚未收到自己的 `response.created`，但收到携带其他 response ID 的迟到事件
- **THEN** relay MUST NOT 将该 ID 绑定到当前 turn、停止当前 watchdog或释放下一 turn

### Requirement: 首 Token 超时不影响账号调度状态
系统 MUST 将首 Token 超时作为不可 failover 的请求级错误处理，不得同账号重试、切换账号、临时封禁账号或提交账号调度失败结果。

#### Scenario: SSE 超时不切换账号
- **WHEN** HTTP SSE 请求发生首 Token 超时
- **THEN** handler MUST 结束当前请求且 MUST NOT 选择另一个账号

#### Scenario: WebSocket 超时不惩罚账号
- **WHEN** WebSocket turn 发生首 Token 超时
- **THEN** 系统 MUST NOT 增加账号失败惩罚或临时不可调度状态

### Requirement: 超时诊断信息可观测
系统 MUST 记录失败 usage、Ops 上游错误和结构化日志 `gateway.openai_first_token_timeout`，并 MUST 包含账号、模型、传输类型、超时档位、配置时长、实际等待时长及可获得的上游阶段信息。

#### Scenario: 响应头后超时记录阶段
- **WHEN** 请求收到响应头和 `response.created` 后发生首 Token 超时
- **THEN** 日志 MUST 标记已收到响应头和 created，并在可用时记录上游 request ID

#### Scenario: 超时用量不伪造首 Token
- **WHEN** 系统记录首 Token 超时失败 usage
- **THEN** 首 Token 字段 MUST 保持为空，且记录 MUST NOT 声明上游零消耗

#### Scenario: 超时日志包含可用模型和阶段
- **WHEN** HTTP 或 WebSocket 已知请求模型，或 V2 已收到 `response.created` 后超时
- **THEN** 结构化日志 MUST 记录请求模型，并正确标记 `created_received`
