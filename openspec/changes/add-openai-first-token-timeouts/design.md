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

`response.created` 与 `response.in_progress` 只记录阶段，不停止 watchdog。已知业务输出事件结束等待；`session.updated`、`rate_limits.updated` 和未知 control 事件不得结束等待。首 Token 指标仅由非终态业务输出填写，`response.failed/completed/canceled` 等终态只停止 watchdog，不伪造首 Token。

图片 `response.output_item.added` 是明确的业务输出和生图确认事件。到达后立即停止 watchdog，随后才启动现有 `stream_data_interval_timeout`。该时序同时适用于协议转换、OAuth passthrough 和 Responses→Chat fallback，避免通用 idle timeout 抢先生图首 Token timeout。

### 4. HTTP 使用可取消上游 context

在最终 wire body 完成归一化后选择超时档位，并在 detached upstream context 之上创建带 cancel cause 的 watchdog context。这样既保留现有客户端断开后 drain/计费语义，也能在响应头前或 `Scanner.Scan()` 阻塞时主动关闭上游读取。

超时使用专用 `OpenAIFirstTokenTimeoutError`，不得包装成 `UpstreamFailoverError`。SSE 前导事件继续缓冲，因此超时时可返回 HTTP 504 和 `first_token_timeout` JSON，而不会与已提交 SSE 混写。

### 5. WebSocket 每个 response 独立计时并执行 cancel/drain

每个成功发送的 `response.create` 使用绝对 deadline；收到前导事件不会重置。超时后网关向上游发送 `response.cancel`，向下游发送一次 `first_token_timeout` error，并在短窗口内 drain 当前 response 的终态。

- 收到 canceled、failed、completed 或 incomplete 终态：当前 response 清理完成，池化连接可复用。
- cancel 写入失败、终态等待超时或 response 归属不明确：标记连接不可复用并关闭，避免迟到事件污染下一 turn。

V2 passthrough relay 在同一 relay 内维护 per-turn watchdog。成功 cancel/drain 后清除当前 turn 状态并允许下一条 `response.create`；清理失败才退出整个 relay。

### 6. 超时与账号健康解耦

首 Token 超时直接结束当前请求/turn，不同账号重试、不换号、不临时封禁，也不调用账号调度失败上报。失败 usage 和 Ops 仍需记录，以便运营识别上游卡顿，但不得把该错误用于账号健康惩罚。drain 获得终态 usage 时保留真实值；无法获得时记录未知状态，不得伪造零消耗。

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
4. 代码回滚时旧版本会忽略持久化 JSON 中的新增字段，不需要数据迁移。

## Open Questions

无。OpenAI 图片前置事件的实际延迟分布作为上线后的观测项，不阻塞实施。
