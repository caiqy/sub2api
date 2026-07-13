# Comet Design Handoff

- Change: add-openai-first-token-timeouts
- Phase: design
- Mode: compact
- Context hash: f560c07500e6ec6a50f1eba91be0ead45cc200e40369e75dd81543da68097048

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/add-openai-first-token-timeouts/proposal.md

- Source: openspec/changes/add-openai-first-token-timeouts/proposal.md
- Lines: 1-31
- SHA256: 3df1ba0ed34955375a24a98c564ec3422815974324c50df733080a02e8dd75d8

```md
## Why

OpenAI/Codex Responses API 偶发在返回首个业务事件前停滞数分钟，现有响应头超时无法覆盖“已返回响应头但 SSE/WebSocket 仍无有效输出”的场景。图片生成又可能正常等待 50–300 秒，因此需要按明确生图意图区分超时档位，避免用统一短超时误杀正常生图。

## What Changes

- 为 OpenAI Responses `stream=true` 请求新增首 Token 超时保护，同时覆盖 HTTP SSE 和 Responses WebSocket。
- 新增 Gateway 运行时配置：生文请求默认 30 秒、明确生图请求默认 600 秒；值为 `0` 时关闭对应超时。
- 仅当 `tool_choice.type == "image_generation"` 时使用生图档；仅在工具列表中携带 `image_generation` 仍按生文档处理。
- 将 `response.created` 和 `response.in_progress` 视为前导事件；首个有效业务事件（包括图片 `response.output_item.added`）结束首 Token 等待。
- 超时后直接失败，不重试、不换号、不临时封禁或惩罚账号；SSE 返回 HTTP 504，WebSocket 发送 `response.cancel` 并清理当前 response。
- 在管理端“网关服务”设置页暴露两个超时字段，并记录超时阶段、传输类型和上游请求标识等诊断信息。
- 不覆盖非流式请求，不增加图片生成总时长限制，不通过提示词关键词猜测生图意图。

## Capabilities

### New Capabilities

- `openai-first-token-timeout`: 定义 OpenAI Responses 流式请求的生文/生图首 Token 分类、超时、协议错误、WebSocket 取消清理、运行时配置和可观测性要求。

### Modified Capabilities

无。

## Impact

- 后端配置与运行时设置 API：`backend/internal/config`、`backend/internal/service/setting_gateway_runtime.go`、管理端 setting handler/DTO。
- OpenAI Responses 转发：passthrough SSE、转换 SSE、池化 WebSocket ingress、WebSocket V2 passthrough relay。
- 管理端前端：Gateway runtime API 类型、SettingsView 表单、中文和英文文案。
- 使用记录与 Ops：复用现有失败 usage 和上游错误记录，不新增数据库 schema。
- 依赖：仅使用 Go 标准库和现有 `gjson`/WebSocket 设施，不新增第三方依赖。

```

## openspec/changes/add-openai-first-token-timeouts/design.md

- Source: openspec/changes/add-openai-first-token-timeouts/design.md
- Lines: 1-85
- SHA256: e22771d42ae859641583e7b79272af9aa5f4b47fc9c785118cde2113ca75c1ef

[TRUNCATED]

```md
## Context

OpenAI Responses 流式链路目前只有响应头超时和流数据间隔超时。前者在收到 HTTP 响应头后失效，后者面向流中断场景，无法准确表达“从上游请求发出到首个业务事件”的等待上限。现有首 Token 指标也会排除 `response.created`、`response.in_progress` 等前导事件。

图片生成经常需要 50–300 秒，但 OpenAI 官方协议通常会在最终图片前发送 `response.output_item.added`，其中 item 为 `image_generation_call`。Codex 可能在所有请求中常驻 `image_generation` 工具，因此工具存在不等于本轮一定生图。

该变更同时影响配置、HTTP SSE、两套 WebSocket 转发路径、失败用量记录和管理端设置。各链路必须共享同一请求分类和事件语义，避免不同传输对同一请求采用不同超时。

## Goals / Non-Goals

**Goals:**

- 为 OpenAI Responses `stream=true` 请求提供生文和明确生图两档首 Token 超时。
- 让超时覆盖等待响应头和等待首个业务事件两个阶段。
- 在 SSE 与 WebSocket 上提供协议正确、不可 failover 的失败行为。
- 保证图片调用一旦通过前置事件暴露，后续正常长耗时生成不受首 Token 超时影响。
- 允许管理员运行时调整或关闭两个超时，并提供足够的阶段遥测。

**Non-Goals:**

- 不覆盖 `stream=false` 请求。
- 不限制首个业务事件之后的图片生成总时长。
- 不通过自然语言提示词判断生图意图。
- 不改变 `IsImageGenerationIntent` 的图片权限、并发或限流语义。
- 不增加数据库字段或第三方依赖。

## Decisions

### 1. 使用强信号预判，而不是工具存在性或提示词分类

仅当 `tool_choice.type == "image_generation"` 时使用图片首 Token 超时；其余流式请求使用文本超时。仅在 `tools` 中出现 `image_generation` 不构成明确生图，因为 Codex 可将该工具常驻在所有请求中。

未采用“只要有图片工具就放宽”的方案，因为它会让普通文本请求绕过短超时；未采用提示词关键词方案，因为多语言、编辑语义和间接指令会导致不可控误判。

### 2. 请求入口决定是否启用，公共分类函数只区分 text/image

HTTP 入口仅在 `stream=true` 时启动 watchdog；Responses WebSocket 天然是流式，每个 `response.create` 独立启动。公共分类函数不依赖 `stream` 字段，因为 WebSocket frame 可以省略该字段。

### 3. 前导事件不结束等待，首个业务事件结束等待

`response.created` 与 `response.in_progress` 只记录阶段，不停止 watchdog。第一个非前导事件结束等待；首 Token 指标仅由非终态业务输出填写，`response.failed/completed/canceled` 等终态只停止 watchdog，不伪造首 Token。

图片 `response.output_item.added` 是明确的业务输出和生图确认事件。到达后立即停止 watchdog，后续图片耗时继续由现有 `stream_data_interval_timeout` 管理。

### 4. HTTP 使用可取消上游 context

在最终 wire body 完成归一化后选择超时档位，并在 detached upstream context 之上创建带 cancel cause 的 watchdog context。这样既保留现有客户端断开后 drain/计费语义，也能在响应头前或 `Scanner.Scan()` 阻塞时主动关闭上游读取。

超时使用专用 `OpenAIFirstTokenTimeoutError`，不得包装成 `UpstreamFailoverError`。SSE 前导事件继续缓冲，因此超时时可返回 HTTP 504 和 `first_token_timeout` JSON，而不会与已提交 SSE 混写。

### 5. WebSocket 每个 response 独立计时并执行 cancel/drain

每个成功发送的 `response.create` 使用绝对 deadline；收到前导事件不会重置。超时后网关向上游发送 `response.cancel`，向下游发送一次 `first_token_timeout` error，并在短窗口内 drain 当前 response 的终态。

- 收到 canceled、failed、completed 或 incomplete 终态：当前 response 清理完成，池化连接可复用。
- cancel 写入失败、终态等待超时或 response 归属不明确：标记连接不可复用并关闭，避免迟到事件污染下一 turn。

V2 passthrough relay 在同一 relay 内维护 per-turn watchdog。成功 cancel/drain 后清除当前 turn 状态并允许下一条 `response.create`；清理失败才退出整个 relay。

### 6. 超时与账号健康解耦

首 Token 超时直接结束当前请求/turn，不同账号重试、不换号、不临时封禁，也不调用账号调度失败上报。失败 usage 和 Ops 仍需记录，以便运营识别上游卡顿，但不得把该错误用于账号健康惩罚。

### 7. 运行时配置向后兼容

新增 `openai_text_first_token_timeout` 和 `openai_image_first_token_timeout`，默认值分别为 30 和 600 秒。`0` 表示关闭，负数拒绝。加载旧版运行时 JSON 时，缺失字段保留配置默认值；运行时更新只影响新创建的请求或 WebSocket turn。

## Risks / Trade-offs

- [OpenAI 未承诺图片前置事件的时间上限] → 明确生图使用 600 秒档，并记录 headers/created/image-added 阶段耗时，后续以线上数据调整默认值。
- [取消本地等待不保证 OpenAI 停止计费] → 日志明确记录本地取消语义，不将 timeout 解释为零上游消耗。
- [超时与首业务事件同时发生] → watchdog 停止和取消使用一次性状态转换，测试确保只产生一个终态和一次客户端错误。
- [WebSocket 迟到事件污染下一 turn] → 只有确认当前 response 终态后才复用连接，否则强制废弃。
- [新增短超时误伤高 reasoning 文本请求] → 管理端允许动态调整或设为 0，默认值通过配置而非硬编码散落在各转发器中。

## Migration Plan

1. 部署包含默认 30/600 秒配置的新版本；旧运行时设置 JSON 自动继承默认值。
2. 通过管理端确认两个字段可读取和保存，先观察 `gateway.openai_first_token_timeout` 日志和失败 usage。
3. 如线上出现误杀，可将对应值调高或设为 0，无需回滚数据库。

```

Full source: openspec/changes/add-openai-first-token-timeouts/design.md

## openspec/changes/add-openai-first-token-timeouts/tasks.md

- Source: openspec/changes/add-openai-first-token-timeouts/tasks.md
- Lines: 1-35
- SHA256: b3812b59fa27956849edd86fec5ee1801a7ef6e4617d1a7a2d68d88cae4bebbc

```md
## 1. Gateway 配置与运行时设置契约

- [ ] 1.1 为生文和明确生图首 Token 超时补充配置加载测试，覆盖 30/600 秒默认值、环境变量、零值和负数拒绝
- [ ] 1.2 在 `GatewayConfig`、运行时设置 view/update DTO 与持久化逻辑中增加两个超时字段，并保证旧 JSON 缺失字段时保留配置默认值
- [ ] 1.3 补充运行时设置 API 测试，验证读取、更新和负数校验

## 2. 共享分类、事件判定与超时错误

- [ ] 2.1 以表驱动测试定义严格请求分类：仅 `tool_choice.type=image_generation` 使用图片档，常驻图片工具仍使用文本档
- [ ] 2.2 实现共享 Responses 事件判定，区分前导、首业务和终态事件，并覆盖图片 `output_item.added`、文本及工具输出
- [ ] 2.3 增加专用首 Token 超时错误、watchdog 与结构化阶段信息，验证并发竞争只产生一个终态且错误不可 failover

## 3. HTTP SSE 首 Token 超时

- [ ] 3.1 为 passthrough 与协议转换 SSE 路径补充响应头前超时、前导事件后超时、业务事件停止计时和零值关闭测试
- [ ] 3.2 在上游请求 context 中接入共享 watchdog，缓冲前导事件，并在超时时返回 HTTP 504 `first_token_timeout`
- [ ] 3.3 验证超时写入失败 usage、Ops 错误和阶段日志，且不重试、不换号、不封禁账号

## 4. 池化 Responses WebSocket ingress 超时

- [ ] 4.1 补充 per-response timeout 测试，覆盖前导事件、业务事件、cancel/drain 成功复用和清理失败废弃连接
- [ ] 4.2 为每个已发送的 `response.create` 接入独立 deadline，超时时发送 `response.cancel` 和一次下游 error
- [ ] 4.3 实现有限 drain 与连接复用判定，并验证 timeout 不触发账号 failover 或健康惩罚

## 5. Responses WebSocket V2 relay 超时

- [ ] 5.1 补充多 turn relay 测试，覆盖首个 turn 超时后成功继续、终态不明时退出及新 turn 使用最新运行时配置
- [ ] 5.2 在 V2 passthrough relay 中维护 per-turn watchdog，并复用共享事件分类与 `response.cancel`/drain 语义
- [ ] 5.3 记录 V2 timeout 的失败 usage、Ops 错误和结构化日志，确保单个 turn 只产生一次错误

## 6. 管理端配置与全量验证

- [ ] 6.1 在管理端“网关服务”设置页增加两个非负秒级输入，支持 `0` 关闭，并补充加载、保存与校验测试
- [ ] 6.2 运行后端定向测试、race 测试与前端测试和类型检查，修复本 change 引入的失败
- [ ] 6.3 按 spec 复核 SSE、两套 WebSocket、运行时配置、可观测性和账号调度隔离场景，并记录验证结果

```

## openspec/changes/add-openai-first-token-timeouts/specs/openai-first-token-timeout/spec.md

- Source: openspec/changes/add-openai-first-token-timeouts/specs/openai-first-token-timeout/spec.md
- Lines: 1-106
- SHA256: 6368f6064c2555cb1ba08645a1cc60bb82a46782e577bc82e1318e9ae91d8fca

[TRUNCATED]

```md
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

```

Full source: openspec/changes/add-openai-first-token-timeouts/specs/openai-first-token-timeout/spec.md
