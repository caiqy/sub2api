---
change: add-openai-first-token-timeouts
design-doc: docs/superpowers/specs/2026-07-13-openai-first-token-timeout-design.md
base-ref: ec6f6e25f20be8c16864a81cbfa7689a25b69871
---

# OpenAI 流式请求首 Token 超时实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 OpenAI Responses 的 HTTP SSE 和 WebSocket 流式请求增加可运行时配置、不可 failover 的首 Token 超时保护。

**Architecture:** 请求发送前按 `tool_choice.type == "image_generation"` 选择 600 秒图片档，否则选择 30 秒文本档；`response.created` 与 `response.in_progress` 不结束等待，首个业务事件结束等待。HTTP 使用可取消 request context 中断响应头或流读取；WebSocket 对当前 response 发送 `response.cancel` 并 drain 终态，无法确认终态时废弃连接。

**Tech Stack:** Go 1.25、Gin、coder/websocket、gjson、Vue 3、TypeScript、Vitest、pnpm

## Global Constraints

- 仅覆盖 OpenAI Responses `stream=true`，不改变 `stream=false` 行为。
- 默认 `openai_text_first_token_timeout=30`、`openai_image_first_token_timeout=600`，单位秒；`0` 关闭，负数拒绝。
- 仅 `tool_choice.type == "image_generation"` 使用图片档；工具列表中存在 `image_generation` 不足以判定生图。
- 超时直接失败，不重试、不换号、不临时封禁、不调用失败调度上报。
- `response.created`、`response.in_progress` 是前导事件；图片 `response.output_item.added` 是首个业务事件。
- 首个业务事件之后继续使用现有 `stream_data_interval_timeout`，不新增图片总时长限制。
- 不修改 `IsImageGenerationIntent`，不新增数据库字段，不新增第三方依赖。

---

### Task 1: Gateway 配置与运行时设置契约

**Files:**
- Modify: `backend/internal/config/config.go:739-750,1947-1952,2665-2670`
- Modify: `backend/internal/config/config_test.go:310-325`
- Modify: `backend/internal/service/settings_view.go:507-513`
- Modify: `backend/internal/service/setting_gateway_runtime.go:132-215`
- Modify: `backend/internal/service/setting_service_gateway_runtime_test.go`
- Modify: `backend/internal/handler/dto/settings.go:397-403`
- Modify: `backend/internal/handler/admin/setting_handler_runtime.go:13-64`
- Modify: `backend/internal/handler/admin/setting_handler_gateway_runtime_test.go`

**Interfaces:**
- Produces: `config.GatewayConfig.OpenAITextFirstTokenTimeout int`
- Produces: `config.GatewayConfig.OpenAIImageFirstTokenTimeout int`
- Produces: JSON fields `openai_text_first_token_timeout` and `openai_image_first_token_timeout`

- [x] **Step 1: 写配置加载失败测试**

在 `config_test.go` 增加默认值、环境变量和负数校验：

```go
func TestLoadOpenAIFirstTokenTimeoutDefaults(t *testing.T) {
	resetViperWithJWTSecret(t)
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, 30, cfg.Gateway.OpenAITextFirstTokenTimeout)
	require.Equal(t, 600, cfg.Gateway.OpenAIImageFirstTokenTimeout)
}

func TestLoadOpenAIFirstTokenTimeoutsFromEnv(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("GATEWAY_OPENAI_TEXT_FIRST_TOKEN_TIMEOUT", "45")
	t.Setenv("GATEWAY_OPENAI_IMAGE_FIRST_TOKEN_TIMEOUT", "720")
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, 45, cfg.Gateway.OpenAITextFirstTokenTimeout)
	require.Equal(t, 720, cfg.Gateway.OpenAIImageFirstTokenTimeout)
}
```

把两个负数字段加入现有配置校验表，期望错误分别包含：

```text
gateway.openai_text_first_token_timeout must be non-negative
gateway.openai_image_first_token_timeout must be non-negative
```

- [x] **Step 2: 运行配置测试并确认失败**

Run: `cd backend && go test ./internal/config -run 'FirstTokenTimeout' -count=1`

Expected: FAIL，提示 `GatewayConfig` 尚无对应字段或默认值不匹配。

- [x] **Step 3: 增加配置字段、默认值和校验**

在 `GatewayConfig` 增加：

```go
// OpenAITextFirstTokenTimeout 普通 Responses 流式请求等待首个业务事件的秒数，0 表示关闭。
OpenAITextFirstTokenTimeout int `mapstructure:"openai_text_first_token_timeout"`
// OpenAIImageFirstTokenTimeout 明确生图 Responses 流式请求等待首个业务事件的秒数，0 表示关闭。
OpenAIImageFirstTokenTimeout int `mapstructure:"openai_image_first_token_timeout"`
```

设置默认值：

```go
viper.SetDefault("gateway.openai_text_first_token_timeout", 30)
viper.SetDefault("gateway.openai_image_first_token_timeout", 600)
```

在 `Config.Validate` 中拒绝负数。

- [x] **Step 4: 写运行时设置失败测试**

扩展 service 和 handler 测试中的配置与 payload，断言：

```go
OpenAITextFirstTokenTimeout:  30,
OpenAIImageFirstTokenTimeout: 600,
```

增加三类测试：保存 `0` 成功、负数返回 `INVALID_GATEWAY_RUNTIME_SETTINGS`、旧 JSON 缺少新字段时保留 `cfg` 中的 `30/600`。

- [x] **Step 5: 运行运行时设置测试并确认失败**

Run: `cd backend && go test ./internal/service ./internal/handler/admin -run 'GatewayRuntime' -count=1`

Expected: FAIL，新字段未出现在返回值、持久化 JSON 或配置更新中。

- [x] **Step 6: 扩展运行时设置实现**

在 service/DTO struct 增加：

```go
OpenAITextFirstTokenTimeout  int `json:"openai_text_first_token_timeout"`
OpenAIImageFirstTokenTimeout int `json:"openai_image_first_token_timeout"`
```

handler 更新请求使用指针以保留旧客户端的省略语义：

```go
OpenAITextFirstTokenTimeout  *int `json:"openai_text_first_token_timeout"`
OpenAIImageFirstTokenTimeout *int `json:"openai_image_first_token_timeout"`
```

`SetGatewayRuntimeSettings` 对两个值执行 `>= 0` 校验并立即写入 `s.cfg.Gateway`。`loadGatewayRuntimeSettingsFromDB` 仅在原始 JSON 存在对应 key 且值非负时覆盖配置默认值。

- [x] **Step 7: 运行测试并提交**

Run: `cd backend && go test ./internal/config ./internal/service ./internal/handler/admin -run 'FirstTokenTimeout|GatewayRuntime' -count=1`

Expected: PASS。

```bash
git add backend/internal/config/config.go backend/internal/config/config_test.go backend/internal/service/settings_view.go backend/internal/service/setting_gateway_runtime.go backend/internal/service/setting_service_gateway_runtime_test.go backend/internal/handler/dto/settings.go backend/internal/handler/admin/setting_handler_runtime.go backend/internal/handler/admin/setting_handler_gateway_runtime_test.go
git commit -m "feat: add OpenAI first token timeout settings"
```

### Task 2: 共享请求分类、事件判定与超时错误

**Files:**
- Create: `backend/internal/pkg/openai/first_token_timeout.go`
- Create: `backend/internal/pkg/openai/first_token_timeout_test.go`
- Create: `backend/internal/service/openai_first_token_timeout.go`
- Create: `backend/internal/service/openai_first_token_timeout_test.go`

**Interfaces:**
- Consumes: `config.GatewayConfig.OpenAITextFirstTokenTimeout`
- Consumes: `config.GatewayConfig.OpenAIImageFirstTokenTimeout`
- Produces: `openai.ResponsesFirstTokenClass(payload []byte) FirstTokenClass`
- Produces: `openai.ResponsesEventEndsFirstTokenWait(payload []byte) bool`
- Produces: `openai.ResponsesEventRecordsFirstToken(payload []byte) bool`
- Produces: `service.OpenAIFirstTokenTimeoutError`
- Produces: HTTP helper context lifecycle used by Task 3

- [x] **Step 1: 写纯判定失败测试**

覆盖以下表格：

```go
func TestResponsesFirstTokenClass(t *testing.T) {
	tests := []struct {
		body string
		want FirstTokenClass
	}{
		{`{"stream":true,"tools":[{"type":"image_generation"}],"tool_choice":"auto"}`, FirstTokenClassText},
		{`{"stream":true,"tool_choice":{"type":"image_generation"}}`, FirstTokenClassImage},
		{`{"type":"response.create","tool_choice":{"type":"image_generation"}}`, FirstTokenClassImage},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, ResponsesFirstTokenClass([]byte(tt.body)))
	}
}
```

事件测试必须断言：

```go
require.False(t, ResponsesEventEndsFirstTokenWait([]byte(`{"type":"response.created"}`)))
require.False(t, ResponsesEventEndsFirstTokenWait([]byte(`{"type":"response.in_progress"}`)))
require.True(t, ResponsesEventEndsFirstTokenWait([]byte(`{"type":"response.output_item.added","item":{"type":"image_generation_call","status":"in_progress"}}`)))
require.True(t, ResponsesEventRecordsFirstToken([]byte(`{"type":"response.output_text.delta","delta":"x"}`)))
require.False(t, ResponsesEventRecordsFirstToken([]byte(`{"type":"response.failed"}`)))
```

- [x] **Step 2: 运行纯判定测试并确认失败**

Run: `cd backend && go test ./internal/pkg/openai -run 'FirstToken' -count=1`

Expected: FAIL，函数和类型尚不存在。

- [x] **Step 3: 实现纯判定函数**

使用 `gjson`，不解析提示词：

```go
type FirstTokenClass string

const (
	FirstTokenClassText  FirstTokenClass = "text"
	FirstTokenClassImage FirstTokenClass = "image"
)

func ResponsesFirstTokenClass(payload []byte) FirstTokenClass {
	if strings.EqualFold(strings.TrimSpace(gjson.GetBytes(payload, "tool_choice.type").String()), "image_generation") {
		return FirstTokenClassImage
	}
	return FirstTokenClassText
}

func ResponsesEventEndsFirstTokenWait(payload []byte) bool {
	t := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
	return t != "" && t != "response.created" && t != "response.in_progress"
}

func ResponsesEventRecordsFirstToken(payload []byte) bool {
	t := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
	if t == "" || t == "error" || strings.HasPrefix(t, "response.completed") ||
		strings.HasPrefix(t, "response.done") || strings.HasPrefix(t, "response.failed") ||
		strings.HasPrefix(t, "response.incomplete") || strings.HasPrefix(t, "response.cancel") {
		return false
	}
	return t != "response.created" && t != "response.in_progress"
}
```

- [x] **Step 4: 写 watchdog 与错误失败测试**

测试 20ms 超时产生 `OpenAIFirstTokenTimeoutError`，业务事件先到时不会超时，`0` 不创建定时器；错误字段包含 class、timeout、transport、headers received、created received 和 elapsed。

- [x] **Step 5: 运行 service 测试并确认失败**

Run: `cd backend && go test ./internal/service -run 'OpenAIFirstTokenTimeout' -count=1`

Expected: FAIL，watchdog 和错误类型尚不存在。

- [x] **Step 6: 实现不可 failover 的错误与 HTTP watchdog**

实现以下最小接口：

```go
type OpenAIFirstTokenTimeoutError struct {
	Class           openai.FirstTokenClass
	Timeout         time.Duration
	Elapsed         time.Duration
	Transport       string
	HeadersReceived bool
	CreatedReceived bool
	UpstreamRequestID string
}

func (e *OpenAIFirstTokenTimeoutError) Error() string {
	return fmt.Sprintf("OpenAI %s stream timed out before first output after %s", e.Class, e.Timeout)
}
```

watchdog 使用 `context.WithCancelCause` 与 `time.AfterFunc`，提供：

```go
func (s *OpenAIGatewayService) withOpenAIFirstTokenTimeout(
	ctx context.Context, payload []byte, transport string,
) (context.Context, *openAIFirstTokenWatchdog)

func (w *openAIFirstTokenWatchdog) MarkHeaders(requestID string)
func (w *openAIFirstTokenWatchdog) Observe(payload []byte)
func (w *openAIFirstTokenWatchdog) Stop()
func (w *openAIFirstTokenWatchdog) TimeoutError() *OpenAIFirstTokenTimeoutError
```

`Observe` 在 `response.created` 时只记录状态，在 `ResponsesEventEndsFirstTokenWait` 为 true 时停止定时器。超时 cause 使用包内 sentinel，转换为公开错误时不得包装成 `UpstreamFailoverError`。

- [x] **Step 7: 运行测试并提交**

Run: `cd backend && go test ./internal/pkg/openai ./internal/service -run 'FirstToken' -count=1`

Expected: PASS。

```bash
git add backend/internal/pkg/openai/first_token_timeout.go backend/internal/pkg/openai/first_token_timeout_test.go backend/internal/service/openai_first_token_timeout.go backend/internal/service/openai_first_token_timeout_test.go
git commit -m "feat: classify OpenAI first token events"
```

### Task 3: HTTP SSE 首 Token 超时

**Files:**
- Modify: `backend/internal/service/openai_gateway_passthrough.go:157-223,867-1092`
- Modify: `backend/internal/service/openai_gateway_forward.go:670-815`
- Modify: `backend/internal/service/openai_gateway_response_handling.go:52-220`
- Modify: `backend/internal/service/openai_gateway_service_test.go`
- Modify: `backend/internal/handler/openai_gateway_handler.go:478-588`
- Modify: `backend/internal/handler/terminal_failed_usage_test.go`

**Interfaces:**
- Consumes: Task 2 的 `withOpenAIFirstTokenTimeout`、`Observe`、`TimeoutError`
- Produces: SSE 超时 HTTP `504` / `first_token_timeout`

- [x] **Step 1: 写 passthrough SSE 失败测试**

使用阻塞 `RoundTripper`/`io.Pipe` 和 20ms 测试配置，覆盖：响应头前超时、只收到 `response.created` 后超时、图片 `output_item.added` 后等待最终图片超过 20ms 仍不超时、`0` 禁用。

超时断言：

```go
var timeoutErr *OpenAIFirstTokenTimeoutError
require.ErrorAs(t, err, &timeoutErr)
require.Equal(t, openai.FirstTokenClassText, timeoutErr.Class)
require.False(t, timeoutErr.HeadersReceived) // 响应头前用例
```

- [x] **Step 2: 写 handler 不 failover 失败测试**

构造 gateway service 返回 `OpenAIFirstTokenTimeoutError`，断言只调用一次 forward、没有账号切换/调度失败上报、HTTP 状态为 504、错误 type 为 `first_token_timeout`、失败 usage 被提交一次。

- [x] **Step 3: 运行目标测试并确认失败**

Run: `cd backend && go test ./internal/service ./internal/handler -run 'FirstTokenTimeout' -count=1`

Expected: FAIL，HTTP request 尚未挂载 watchdog，handler 尚未识别专用错误。

- [x] **Step 4: 在最终 wire body 上启动 watchdog**

在 passthrough 和非 passthrough 分支完成请求体归一化后，先 detach 原 context，再挂载 watchdog：

```go
upstreamBaseCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
upstreamCtx, firstTokenWatchdog := s.withOpenAIFirstTokenTimeout(upstreamBaseCtx, body, "sse")
defer firstTokenWatchdog.Stop()
upstreamReq, err := s.buildUpstreamRequestOpenAIPassthrough(upstreamCtx, c, account, body, body, token)
releaseUpstreamCtx()
```

`Do` 返回后调用 `MarkHeaders(resp.Header.Get("x-request-id"))`。`Do` 或流读取报错时先检查 `TimeoutError()`；命中后直接返回该错误，禁止进入 `handleOpenAIUpstreamTransportError`。

- [x] **Step 5: 在两套 SSE 处理器观察事件**

passthrough 在解析 `data:` JSON 后调用：

```go
firstTokenWatchdogFromContext(ctx).Observe(dataBytes)
```

转换流在事件 reader/select 收到上游 JSON 后执行同样调用。必须在写客户端之前停止定时器，避免边界竞争产生 504 与 SSE 双写。

- [x] **Step 6: handler 返回专用 504 且不惩罚账号**

在 `UpstreamFailoverError` 分支之前增加：

```go
var firstTokenTimeout *service.OpenAIFirstTokenTimeoutError
if errors.As(err, &firstTokenTimeout) {
	h.submitFailedUsageLog(c, apiKey, account, reqModel, true, 0, nil, nil, forwardDuration, attemptReasoningEffort, "handler.openai_gateway.responses")
	h.handleStreamingAwareError(c, http.StatusGatewayTimeout, "first_token_timeout", "Upstream timed out before the first response event", false)
	return
}
```

记录 `gateway.openai_first_token_timeout`，字段包含 error struct 中的 class、timeout、elapsed、transport、headers/created flags 和 request ID。该分支不得调用 `ReportOpenAIAccountScheduleResult(account.ID, false, nil)`。

- [x] **Step 7: 运行 SSE 与 handler 测试并提交**

Run: `cd backend && go test ./internal/service ./internal/handler -run 'FirstTokenTimeout|StreamingPassthrough' -count=1`

Expected: PASS。

```bash
git add backend/internal/service/openai_gateway_passthrough.go backend/internal/service/openai_gateway_forward.go backend/internal/service/openai_gateway_response_handling.go backend/internal/service/openai_gateway_service_test.go backend/internal/handler/openai_gateway_handler.go backend/internal/handler/terminal_failed_usage_test.go
git commit -m "feat: enforce OpenAI SSE first token timeout"
```

### Task 4: 池化 Responses WebSocket ingress 超时

**Files:**
- Modify: `backend/internal/service/openai_ws_forwarder_support.go:207-229`
- Modify: `backend/internal/service/openai_ws_forwarder_ingress.go:680-945`
- Modify: `backend/internal/service/openai_ws_forwarder_ingress_session_test.go`
- Modify: `backend/internal/service/openai_ws_forwarder_test.go`
- Modify: `backend/internal/handler/openai_gateway_handler.go:1771-1904`

**Interfaces:**
- Consumes: Task 2 的 request class/event helpers 和 `OpenAIFirstTokenTimeoutError`
- Produces: pooled lease 的 cancel/drain/reuse 行为

- [x] **Step 1: 修正 WS 首业务事件测试预期**

将事件判定改为读取完整 payload，而不是只读 type。新增断言：图片和函数 `output_item.added` 计作首 Token，`response.created/in_progress/completed/failed` 不计作首 Token。

```go
require.True(t, openAIWSMessageRecordsFirstToken([]byte(`{"type":"response.output_item.added","item":{"type":"image_generation_call"}}`)))
require.True(t, openAIWSMessageRecordsFirstToken([]byte(`{"type":"response.output_item.added","item":{"type":"function_call"}}`)))
require.False(t, openAIWSMessageRecordsFirstToken([]byte(`{"type":"response.completed"}`)))
```

- [x] **Step 2: 写池化连接超时场景失败测试**

在 session fake upstream 中覆盖：

1. `response.created` 后不再发事件，网关发送 `response.cancel`。
2. 上游随后发 `response.canceled`，lease 未 `MarkBroken`，下一 turn 可复用同一 conn ID。
3. cancel 写失败或 drain 期限内无终态，lease 被 `MarkBroken`。
4. 下游收到且只收到一个 `first_token_timeout` error。
5. `AfterTurn` 收到专用 timeout error，handler 不进入 failover。

- [x] **Step 3: 运行目标测试并确认失败**

Run: `cd backend && go test ./internal/service -run 'WS.*FirstTokenTimeout|FirstToken.*WS' -count=1`

Expected: FAIL，当前 read 只使用通用 WS timeout，也不会发送 `response.cancel`。

- [x] **Step 4: 按 turn 建立绝对首 Token deadline**

每次发送 `response.create` 后，根据该 payload 计算 timeout；读取上游时使用通用 read timeout 与 deadline 剩余时间的较小值。前导事件不得重置 deadline：

```go
firstTokenTimeout := s.openAIFirstTokenTimeout(payload)
firstTokenDeadline := time.Time{}
if firstTokenTimeout > 0 {
	firstTokenDeadline = turnStart.Add(firstTokenTimeout)
}
firstTokenDone := false
```

完整 payload 满足 `openai.ResponsesEventEndsFirstTokenWait` 时令 `firstTokenDone=true`；仅 `ResponsesEventRecordsFirstToken` 为 true 时填写 `FirstTokenMs`。

- [x] **Step 5: 实现 cancel/drain**

到达 deadline 时：

```go
cancelPayload := map[string]any{"type": "response.cancel"}
if err := lease.WriteJSONWithContextTimeout(ctx, cancelPayload, s.openAIWSWriteTimeout()); err != nil {
	lease.MarkBroken()
	return nil, firstTokenTimeoutErr
}
```

使用固定 2 秒 drain 窗口读取到 `response.canceled/cancelled/failed/completed/incomplete`。成功时保留 lease；失败时 `MarkBroken`。向下游写一次：

```json
{"type":"error","error":{"type":"first_token_timeout","code":"first_token_timeout","message":"Upstream timed out before the first response event"}}
```

随后以 `OpenAIFirstTokenTimeoutError` 完成当前 turn。

- [x] **Step 6: handler 跳过 failover 与调度惩罚**

`AfterTurn` 继续提交失败 usage；外层 `ProxyResponsesWebSocketFromClient` 错误处理在 `UpstreamFailoverError` 之前识别 timeout error，直接结束，不增加 `failedAccountIDs`、`switchCount`，也不调用调度失败上报。

- [x] **Step 7: 运行测试并提交**

Run: `cd backend && go test ./internal/service ./internal/handler -run 'WS.*FirstToken|FirstToken.*WS|ResponsesWebSocket' -count=1`

Expected: PASS。

```bash
git add backend/internal/service/openai_ws_forwarder_support.go backend/internal/service/openai_ws_forwarder_ingress.go backend/internal/service/openai_ws_forwarder_ingress_session_test.go backend/internal/service/openai_ws_forwarder_test.go backend/internal/handler/openai_gateway_handler.go
git commit -m "feat: enforce pooled WebSocket first token timeout"
```

### Task 5: Responses WebSocket V2 passthrough relay 超时

**Files:**
- Modify: `backend/internal/service/openai_ws_v2/passthrough_relay.go`
- Modify: `backend/internal/service/openai_ws_v2/passthrough_relay_internal_test.go`
- Modify: `backend/internal/service/openai_ws_v2/passthrough_relay_test.go`
- Modify: `backend/internal/service/openai_ws_v2_passthrough_adapter.go:504-667`

**Interfaces:**
- Consumes: Task 2 的 request/event helpers
- Extends: `openai_ws_v2.RelayOptions` with text/image first-token durations
- Extends: `RelayTurnResult` with `TurnError error`

- [x] **Step 1: 写 relay 失败测试**

使用 fake frame conn 覆盖同一连接两 turn：第一 turn 只收到 `response.created` 后超时，relay 写 `response.cancel`，收到 `response.canceled` 后回调 `TurnError=OpenAIFirstTokenTimeoutError`；第二 turn 正常输出并完成，证明连接继续工作。

再覆盖 cancel 写失败和 drain 超时，期望 `RelayExit.Stage == "first_token_timeout"` 且连接关闭。

- [x] **Step 2: 运行 relay 测试并确认失败**

Run: `cd backend && go test ./internal/service/openai_ws_v2 -run 'FirstTokenTimeout' -count=1`

Expected: FAIL，`RelayOptions`、per-turn deadline 和 cancel 状态尚不存在。

- [x] **Step 3: 增加 per-turn watchdog 状态**

`RelayOptions` 增加：

```go
TextFirstTokenTimeout  time.Duration
ImageFirstTokenTimeout time.Duration
```

在每个成功写入上游的 `response.create` 上 arm deadline；`tool_choice.type=image_generation` 选择图片档。状态至少包含：

```go
type relayTurnTiming struct {
	startAt          time.Time
	firstTokenMs     *int
	firstTokenClass  openai.FirstTokenClass
	firstTokenDueAt  time.Time
	firstTokenDone   bool
	firstTokenTimedOut bool
}
```

用 relay 内部 control channel 传递 deadline 到期，不以普通 idle activity 重置。

- [x] **Step 4: 在 relay 内执行 cancel、drain 和继续**

deadline 到期后由 relay 的单写者路径依次执行：写 `response.cancel`、向客户端写 timeout error、暂停转发新的 `response.create`。上游终态到达后：

- 设置当前 `RelayTurnResult.TurnError` 为 timeout error；
- 调用 `OnTurnComplete`；
- 清除当前 turn watchdog；
- 恢复接收下一条 `response.create`，不退出 relay。

cancel 写失败或 2 秒未收到终态时发送 `RelayExit{Stage:"first_token_timeout"}` 并关闭连接。所有状态切换保持在 relay goroutine/control channel 内，避免为共享 state 增加数据竞争。

- [x] **Step 5: adapter 接入配置和失败 usage**

设置 options：

```go
TextFirstTokenTimeout:  time.Duration(s.cfg.Gateway.OpenAITextFirstTokenTimeout) * time.Second,
ImageFirstTokenTimeout: time.Duration(s.cfg.Gateway.OpenAIImageFirstTokenTimeout) * time.Second,
```

`OnTurnComplete` 将 `turn.TurnError` 传给 `hooks.AfterTurn`；timeout turn 不按成功调用 `ReportOpenAIAccountScheduleResult`。relay 整体因 cancel/drain 失败退出时返回专用 timeout error，不转换为 `UpstreamFailoverError`。

- [x] **Step 6: 运行 race 与 relay 测试并提交**

Run: `cd backend && go test -race ./internal/service/openai_ws_v2 ./internal/service -run 'FirstTokenTimeout|Relay' -count=1`

Expected: PASS 且 race detector 无报告。

```bash
git add backend/internal/service/openai_ws_v2/passthrough_relay.go backend/internal/service/openai_ws_v2/passthrough_relay_internal_test.go backend/internal/service/openai_ws_v2/passthrough_relay_test.go backend/internal/service/openai_ws_v2_passthrough_adapter.go
git commit -m "feat: enforce passthrough WebSocket first token timeout"
```

### Task 6: 管理端配置与全量验证

**Files:**
- Modify: `frontend/src/api/admin/settings.ts:1088-1093`
- Modify: `frontend/src/views/admin/SettingsView.vue:237-297,7851-7860,10295-10340`
- Modify: `frontend/src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts`
- Modify: `frontend/src/i18n/locales/zh/admin/settings.ts`
- Modify: `frontend/src/i18n/locales/en/admin/settings.ts`

**Interfaces:**
- Consumes: Task 1 的两个 runtime JSON 字段
- Produces: 网关服务页两个数值输入框，允许 `0` 或正整数

- [x] **Step 1: 扩展前端失败测试**

mock API 返回：

```ts
openai_text_first_token_timeout: 30,
openai_image_first_token_timeout: 600,
```

断言两个输入存在、加载值正确、保存 payload 包含修改值；参数化校验测试增加两个负数用例，`0` 必须允许提交。

- [x] **Step 2: 运行前端测试并确认失败**

Run: `cd frontend && pnpm vitest run src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts`

Expected: FAIL，新字段和输入框尚不存在。

- [x] **Step 3: 扩展 API 类型、表单和校验**

API interface 与 reactive form 增加：

```ts
openai_text_first_token_timeout: number
openai_image_first_token_timeout: number
```

默认值分别为 `30`、`600`。`buildGatewayRuntimePayload` 要求两者非空且 `>= 0`，并原样提交。

- [x] **Step 4: 增加两个原生 number input**

在响应头超时和流间隔超时之间加入两个 `type="number" min="0"` 输入：

```html
<input v-model.number="gatewayRuntimeForm.openai_text_first_token_timeout"
       data-testid="gateway-runtime-openai-text-first-token-timeout"
       type="number" min="0" class="input w-40" />
<input v-model.number="gatewayRuntimeForm.openai_image_first_token_timeout"
       data-testid="gateway-runtime-openai-image-first-token-timeout"
       type="number" min="0" class="input w-40" />
```

中文标签分别为“生文请求首 Token 超时时间（秒）”“生图请求首 Token 超时时间（秒）”；提示明确 `0` 关闭、仅影响 OpenAI Responses 流式请求。英文提供等价文本。

- [x] **Step 5: 运行前端测试与类型检查**

Run: `cd frontend && pnpm vitest run src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts`

Expected: PASS。

Run: `cd frontend && pnpm typecheck`

Expected: PASS。

- [x] **Step 6: 运行后端目标包和 race 测试**

Run: `cd backend && go test ./internal/config ./internal/pkg/openai ./internal/handler/admin ./internal/handler ./internal/service -count=1`

Expected: PASS。

Run: `cd backend && go test -race ./internal/service/openai_ws_v2 ./internal/service -run 'FirstTokenTimeout|Relay' -count=1`

Expected: PASS 且 race detector 无报告。

- [x] **Step 7: 检查格式并提交**

Run: `cd backend && gofmt -w internal/config/config.go internal/config/config_test.go internal/pkg/openai/first_token_timeout.go internal/pkg/openai/first_token_timeout_test.go internal/service/openai_first_token_timeout.go internal/service/openai_first_token_timeout_test.go internal/service/openai_gateway_passthrough.go internal/service/openai_gateway_forward.go internal/service/openai_gateway_response_handling.go internal/service/openai_ws_forwarder_support.go internal/service/openai_ws_forwarder_ingress.go internal/service/openai_ws_v2/passthrough_relay.go internal/service/openai_ws_v2/passthrough_relay_internal_test.go internal/service/openai_ws_v2/passthrough_relay_test.go internal/service/openai_ws_v2_passthrough_adapter.go internal/service/settings_view.go internal/service/setting_gateway_runtime.go internal/service/setting_service_gateway_runtime_test.go internal/handler/dto/settings.go internal/handler/admin/setting_handler_runtime.go internal/handler/admin/setting_handler_gateway_runtime_test.go internal/handler/openai_gateway_handler.go`

Expected: 命令成功且只格式化本计划涉及的 Go 文件。

```bash
git add frontend/src/api/admin/settings.ts frontend/src/views/admin/SettingsView.vue frontend/src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts frontend/src/i18n/locales/zh/admin/settings.ts frontend/src/i18n/locales/en/admin/settings.ts
git commit -m "feat: configure OpenAI first token timeouts"
```

- [x] **Step 8: 最终变更核验**

Run: `git diff --check HEAD~6..HEAD`

Expected: 无输出。

Run: `git status --short`

Expected: 本功能涉及文件无未提交改动；保留实施前已经存在的其他工作区改动。
