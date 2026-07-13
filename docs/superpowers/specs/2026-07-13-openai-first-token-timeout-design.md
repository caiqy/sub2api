---
comet_change: add-openai-first-token-timeouts
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-13-add-openai-first-token-timeouts
status: final
---

# OpenAI 流式请求首 Token 超时设计

## 背景

OpenAI/Codex Responses API 偶发在返回首个有效业务事件前停滞数分钟。现有 `response_header_timeout` 只覆盖 HTTP 响应头，无法限制响应头已返回但 SSE 或 WebSocket 长时间只有前导事件、没有业务输出的情况。

图片生成通常需要 50–300 秒，不能与文本请求共用短超时。同时，Codex 可能在所有请求中常驻 `image_generation` 工具，因此仅凭工具是否存在无法判断本轮一定生图。

## 目标

- 为 OpenAI Responses 流式请求增加可运行时调整的首 Token 超时。
- 文本请求默认 30 秒，明确生图请求默认 600 秒。
- 覆盖 HTTP SSE 和 Responses WebSocket。
- 超时直接失败，不重试、不换号、不改变账号调度状态。
- 不误伤已经进入正常图片生成阶段的请求。

## 非目标

- 不覆盖 `stream=false` 请求。
- 不为图片生成增加总时长限制；首个业务事件后仍沿用现有流间隔超时。
- 不通过提示词关键词猜测图片意图。
- 不修改现有图片权限、并发或模型限流语义。

## 方案比较

### 只按请求体中的工具判断

只要 `tools` 包含 `image_generation` 就使用 600 秒。实现简单，但 Codex 会常驻该工具，导致大量文本请求绕过 30 秒保护，因此不采用。

### 只按上游事件动态判断

所有请求先使用 30 秒，收到图片事件后确认生图。该方案无法保护图片前置事件本身超过 30 秒的情况，因此不单独采用。

### 强信号预判并由事件确认

明确选择图片工具的请求从开始即使用 600 秒；仅常驻图片工具的请求使用 30 秒。收到 `image_generation_call` 输出项即视为首个业务事件并停止计时。这是本次采用的方案。

## 请求分类

仅对 OpenAI Responses `stream=true` 请求启用首 Token 超时。

- `tool_choice.type == "image_generation"`：使用图片首 Token 超时。
- 仅在 `tools` 中出现 `image_generation`，且 `tool_choice` 为 `auto` 或未指定：使用文本首 Token 超时。
- 不读取自然语言提示词进行分类。

首 Token 分类不得复用或修改 `IsImageGenerationIntent`。后者仍按现有宽松规则服务图片权限、并发和限流。首 Token 超时使用独立、严格的判定函数。

## 业务事件定义

以下事件是前导事件，不停止首 Token 计时：

- `response.created`
- `response.in_progress`

仅已知输出事件视为首个业务事件；`session.updated`、`rate_limits.updated`、`response.queued` 和未知 control 事件不结束等待。业务事件包括：

- 文本或 reasoning delta
- 函数、工具和其他输出项
- `response.output_item.added`，其中 `item.type == "image_generation_call"` 时确认上游已进入图片生成
- 图片生成状态、partial image 或完成事件

OpenAI 官方协议和 SDK 预期图片输出遵循以下生命周期：

```text
response.created
response.in_progress
response.output_item.added (image_generation_call, status=in_progress)
response.image_generation_call.in_progress
response.image_generation_call.generating
response.image_generation_call.partial_image (可选)
response.image_generation_call.completed
response.output_item.done
response.completed
```

`response.output_item.added` 到达后首 Token 保护立即结束。现有 `stream_data_interval_timeout` 从该业务事件起开始计时，覆盖协议转换、OAuth passthrough 和 Responses→Chat fallback；不得在首 Token 等待阶段抢先结束请求或惩罚账号。

## 配置

在 Gateway 配置和运行时设置中新增两个秒级字段：

| 字段 | 默认值 | 含义 |
|---|---:|---|
| `openai_text_first_token_timeout` | 30 | 普通流式请求等待首个业务事件的最长时间 |
| `openai_image_first_token_timeout` | 600 | 明确生图流式请求等待首个业务事件的最长时间 |

值为 `0` 时关闭对应超时。负数无效。配置进入：

- 配置文件与环境变量加载
- Gateway 运行时设置读取和更新 API
- 管理端“网关服务”设置页

运行时修改只影响随后创建的新上游请求。

## 计时与取消

计时从真正发起上游请求前开始，覆盖：

- 等待 HTTP 响应头
- 等待首个 SSE 业务事件
- 等待 WebSocket 当前 response 的首个业务事件

每个上游请求创建独立的可停止定时器和带 cause 的取消 context。首个业务事件到达时停止定时器；超时时以专用 `first_token_timeout` cause 取消请求。实现使用 Go 标准库，不增加依赖。

超时错误不是 `UpstreamFailoverError`，不得进入现有 failover loop，也不得触发同账号重试、换号、临时封禁或账号健康惩罚。

## HTTP SSE 行为

现有实现会缓冲 `response.created` 和 `response.in_progress`，因此超时时客户端响应尚未提交。网关取消上游请求并返回 HTTP `504`：

```json
{
  "error": {
    "type": "first_token_timeout",
    "message": "Upstream timed out before the first response event"
  }
}
```

若首个业务事件已写入客户端，则首 Token 定时器已经停止，不再产生首 Token 超时。

## WebSocket 行为

WebSocket 超时后：

1. 向上游发送 `response.cancel`。
2. 短暂 drain 当前 response，等待 canceled、failed 或 completed 终态。
3. 确认终态后允许连接复用。
4. cancel 发送失败、终态等待失败或 response 归属不明确时，丢弃该上游连接，防止迟到事件污染下一轮。
5. 向下游发送一次符合 Responses schema 的 `error` 事件并结束当前 response。

连接清理不触发账号 failover 或封禁。

V2 relay 使用单 active turn permit。后续 turn 的 `BeforeRequest`、`BeforeTurn`、metadata 切换、watchdog arm 和上游写入都在 permit 内按顺序执行；首轮完成或 timeout drain 成功后才释放 permit，避免并发槽绕过和跨 turn 元数据污染。

## 可观测性

超时写入失败 usage log 和 Ops 上游错误事件，并至少记录：

- account ID、模型和传输类型
- 使用的超时档位与配置秒数
- 实际等待毫秒数
- 是否已收到响应头
- 是否已收到 `response.created`
- 上游 request ID（如有）

新增结构化日志事件 `gateway.openai_first_token_timeout`。无需新增数据库字段；首 Token 保持为空，阶段信息写入现有错误详情和结构化日志。drain 获得终态 usage 时，失败审计保留真实 token 但保持零成本；无法取得 usage 时在详情中明确标记 `usage_state=unknown`。

取消本地请求不能保证 OpenAI 停止计算或不计费，日志和运维说明不得将超时等同于上游零消耗。

## 验证

最小测试覆盖：

1. `tools` 包含图片工具但 `tool_choice=auto` 时使用 30 秒。
2. `tool_choice=image_generation` 时使用 600 秒。
3. 只有 `response.created/in_progress` 时到期超时。
4. 文本 delta、函数调用或图片 `output_item.added` 到达时停止超时。
5. 响应头之前和之后超时均能取消上游。
6. 超时不进入 failover、不封禁账号。
7. 配置值 `0` 关闭超时，负数被拒绝。
8. 运行时设置保存后对新请求生效。
9. WebSocket cancel 成功时连接可复用；无法确认终态时连接被丢弃。
10. 超时与首业务事件竞争时只产生一个终态，不重复写响应。
11. 流间隔超时只在首业务事件后启动，并覆盖 OAuth passthrough 与 Responses→Chat fallback。
12. V2 后续 turn 静默时仍能唤醒 timeout，且 hooks/metadata 不会越过 active turn permit。
13. timeout drain 保留已知 usage，未知 usage 不伪装为上游零消耗。

## 参考

- OpenAI Image generation guide: <https://platform.openai.com/docs/guides/tools-image-generation>
- OpenAI Node ResponseAccumulator: <https://github.com/openai/openai-node/blob/master/src/lib/responses/ResponseAccumulator.ts>
- Codex Responses SSE parser: <https://github.com/openai/codex/blob/main/codex-rs/codex-api/src/sse/responses.rs>
- Codex TTFT classification: <https://github.com/openai/codex/blob/main/codex-rs/core/src/turn_timing.rs>
