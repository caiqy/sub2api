## ADDED Requirements

### Requirement: 最终有效路由是请求策略的单一权威状态

系统 MUST 在 composite routing 或 client-specific fallback 后，以最终有效路由统一决定 group、API key、subscription、计费来源、目标平台、endpoint 协议分发、调度输入和 Ops 上下文。系统 MUST 使用 request-owned API key clone，且不得原地修改共享认证对象。

#### Scenario: Client fallback 切换到最终分组

- **WHEN** API Key 原分组因客户端类型规则 fallback 到另一个可用分组
- **THEN** 系统 MUST 在 endpoint 协议分发、并发、调度和计费前把 Gin API key、request group、subscription、目标平台和 Ops 同步为最终分组
- **AND** 后续阶段 MUST 不再从原分组重新推导策略

#### Scenario: 最终分组不允许当前用户访问

- **WHEN** fallback 或 composite routing 解析出的最终分组不允许当前用户访问
- **THEN** 系统 MUST 在协议分发和账号调度前拒绝请求
- **AND** 系统 MUST 不应用部分有效路由状态

#### Scenario: 非 composite 请求保持 identity

- **WHEN** 请求无需 client fallback 且所属分组不是 composite
- **THEN** 系统 MUST 保持既有 group、平台、endpoint 和模型行为

### Requirement: 有效路由显式决定计费来源

系统 MUST 明确区分 SimpleMode 跳过、余额和订阅三种计费来源。订阅来源 MUST 包含最终分组的有效 subscription；系统不得用空 subscription 把订阅请求隐式转换为余额请求。

#### Scenario: 最终分组使用订阅计费

- **WHEN** 有效路由的最终分组是订阅分组
- **THEN** 系统 MUST 为该最终分组加载并校验有效 subscription
- **AND** billing MUST 使用订阅限额且不执行余额或 user-platform quota 检查

#### Scenario: 最终订阅不可用

- **WHEN** 有效路由的最终订阅分组不存在有效 subscription 或订阅限额不可用
- **THEN** 系统 MUST 按现有订阅错误契约拒绝请求
- **AND** 系统 MUST 不调度账号、不计费且不回退到余额模式

#### Scenario: SimpleMode 保持跳过计费

- **WHEN** 系统运行于 SimpleMode
- **THEN** 有效路由 MUST 保持现有跳过计费语义，且不得要求加载 subscription

### Requirement: Runtime fallback 原子替换有效路由

系统 MUST 让 prompt-too-long 等 runtime fallback 调用与初始请求相同的有效路由 resolver，并在候选结果完整通过授权、subscription、余额、模型和平台校验后原子应用。

#### Scenario: Prompt-too-long 切换到有效候选组

- **WHEN** 上游返回 prompt-too-long 且配置的二级 fallback 最终解析到可用分组
- **THEN** 系统 MUST 使用最终候选组重新建立 API key、subscription、平台、模型和调度状态后再重试

#### Scenario: Runtime fallback 校验失败

- **WHEN** runtime fallback 的最终候选组未通过任一必要校验
- **THEN** 系统 MUST 终止请求并保留应用前状态
- **AND** 系统 MUST 不使用中间组继续调度、写入上游或计费

### Requirement: 模型映射阶段保持确定顺序

系统 MUST 按 client model、composite route、可选 channel mapping、可选 account mapping 的顺序解析模型，并 MUST 分别保留 client、routing 和 upstream 模型供请求改写与 usage 审计使用。

#### Scenario: 无 channel mapping 时保留 concrete route model

- **WHEN** public client alias 经过 composite route 得到 concrete model，且 channel mapping 未命中
- **THEN** `routingModel` MUST 保持该 concrete model
- **AND** 系统 MUST 不使用 public client alias 覆盖它

#### Scenario: Channel 与 account mapping 同时命中

- **WHEN** channel mapping 和已选账号的 account mapping 都命中
- **THEN** 系统 MUST 先生成 channel-mapped `routingModel`，再以它执行 account mapping 得到 `upstreamModel`
- **AND** usage MUST 记录正确的 client、routing 和 upstream 阶段

#### Scenario: 所有映射均未命中

- **WHEN** composite route、channel mapping 和 account mapping 均未改变模型
- **THEN** 三个模型阶段 MUST 保持 identity，且请求体不得发生无意义改写

### Requirement: WebSocket 后续帧保持账号与 provider 一致

系统 MUST 让 HTTP bridge 的后续 `response.create` 在有效路由与 channel mapping 后继续执行 provider affinity 和 account mapping；已建立会话连接时不得把不同 provider 的请求写入原上游连接。普通 WebSocket 的既有后续帧语义 MUST 保持不变。

#### Scenario: 后续帧应用账号模型映射

- **WHEN** 普通 WebSocket 或 HTTP bridge 的后续 `response.create` 解析出与会话账号同 provider 的 routing model
- **THEN** 系统 MUST 复用会话账号并在发送前应用该账号的 model mapping

#### Scenario: 后续帧试图切换 provider

- **WHEN** 后续 `response.create` 的有效路由目标 provider 与已绑定账号 provider 不同
- **THEN** 系统 MUST 在修改请求体、会话路由状态或写入上游前拒绝该帧
- **AND** 已建立会话的账号与 provider 状态 MUST 保持不变
