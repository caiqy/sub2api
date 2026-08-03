# Final Fix Report

Status: `PASS`

Base commit: `0aa2153a392710d562a0abaf0dc72acc0e2b377c`

## Scope

The final wave removes full request-body retention while HTTP upstream calls are blocked for:

- OpenAI Responses passthrough and agent-identity recovery.
- Grok Responses and encrypted-content retry.
- Responses-to-Chat-Completions fallback.
- Raw and converted Chat Completions.
- Anthropic Messages routed through Gemini compatibility, including signature retry.

The public byte-based Gemini `Forward` method remains available. No dependency, configuration, schema, or WebSocket implementation changed.

## Implementation

- Upstream requests now open their bodies from `RequestBodyHandle` instances. Large byte slices and parsed conversion structures are released before `HTTPUpstream.Do` waits.
- Retry paths rebuild or replace owned handles from the canonical source and clean superseded handles.
- Error handlers materialize request bytes only after the upstream response returns.
- `GatewayHandler.Messages` calls Gemini `ForwardHandle` directly and maps spool/materialization failures to HTTP 503 after releasing the account slot.
- Parsed model, metadata user ID, and output effort strings are cloned before the full parsed body becomes unreachable, preventing short strings from retaining its backing array.
- Gemini signature-retry spool failures retain `ErrRequestBodySpool` instead of falling through to the original upstream 400.

## TDD Evidence

The final review added `TestGeminiMessagesCompatSignatureRetryPropagatesCanonicalSpoolFailure`. With the canonical spool removed after the first signature-related 400, the test failed before the fix because the service returned the original upstream error:

```text
expected: request body spool failed
in chain: upstream error: 400 message=invalid thought_signature
```

After propagating the spool sentinel, the focused test passed:

```text
go test ./internal/service -run 'TestGeminiMessagesCompatSignatureRetry(BuildsHandleFromCanonical|PropagatesCanonicalSpoolFailure)' -count=1
ok github.com/Wei-Shaw/sub2api/internal/service
```

Other regression coverage includes spool-backed Chat Completions replay, Gemini signature retry from the canonical handle, Gemini materialization failure returning 503 with zero upstream calls and a released account slot, and the seven-path blocked-upstream heap matrix.

## Heap Evidence

Command:

```text
go test ./internal/handler -run TestRequestBodyMemoryRetentionWhileUpstreamBlocked -count=1 -v
```

Fresh retained growth from 2 MiB to 8.9 MiB requests:

| Path | Retained growth |
| --- | ---: |
| Responses | 8,152 B |
| OpenAI passthrough | 1,424 B |
| Grok Responses | 6,312 B |
| Responses chat fallback | 36,232 B |
| Chat raw | 6,416 B |
| Chat converted | 19,088 B |
| Messages to Gemini | 528 B |

All paths passed the `< 3 MiB` ceiling. Large-body previews remained 31 bytes, serialized snapshots remained 107 bytes, and the ordinary preview boundary remained 262,065 bytes with a 262,143-byte serialized snapshot.

## Verification

The following commands passed from `backend/` unless noted:

```text
go test ./internal/service -run '(Passthrough|GrokResponses|ChatCompletions.*Raw|ChatFallback|GeminiMessagesCompat|ForwardAsResponses)' -count=1
go test -tags=unit ./internal/service ./internal/handler -run '(Passthrough|GrokResponses|ChatCompletions|ChatFallback|GeminiMessagesCompat|ForwardAsResponses|RequestBodyMemoryRetention|MessagesGeminiMaterialization)' -count=1
go test ./internal/handler -run '(GatewayHandler_(Messages|Responses)|ChatCompletions|OpenAIImages|GrokMedia)' -count=1
go test ./internal/handler -run TestRequestBodyMemoryRetentionWhileUpstreamBlocked -count=1 -v
go build ./...
go vet ./...
go test ./... -count=1
git diff --check
```

The full test run passed, including `internal/handler` in 76.284s and `internal/service` in 111.863s.

## Boundary Review

- `backend/internal/service/openai_ws_forwarder_v2.go`: zero diff.
- `backend/internal/service/openai_ws_protocol_forward.go`: zero diff.
- `.comet/current-change.json`: pre-existing untracked file, unchanged and excluded from the commit.
- Self-review found no remaining Critical or Important issue in handle ownership, retry replacement, response status mapping, or public compatibility.

## Wave 2 Re-review Fixes

状态：`PASS`

修复提交：`c570826c9fadbcc3e4a11723196d79ff3633b196 fix: preserve gemini request body semantics`

### CRITICAL 3：混合调度 Messages 到 Gemini

- 根因：invalid-request fallback 已进入通用调度循环后切换到 Gemini，仍会 `ReadAll()` 后调用 byte wrapper；`CloneForHandle` 刷新的三个 gjson 标量还会重新引用完整 body backing array。
- 修复：通用 Gemini 分支直接调用 `ForwardHandle(..., attemptHandle)`；attempt clone 后对 `Model`、`MetadataUserID`、`OutputEffort` 执行 `strings.Clone`。
- RED：`messages-gemini-mixed` 从 2 MiB 到 8.9 MiB 的 retained growth 为 `7,247,824 B`，超过 `< 3 MiB` 上限。
- GREEN：完整 heap 矩阵中该路径 retained growth 为 `1,584 B`；其余七条路径同时通过。

### IMPORTANT：Outbound Handle Open 错误链

- 根因：`outboundHandle.Open()` 的 `ErrRequestBodySpool` 经 `buildReq` 返回后被 `writeClaudeError(502)` 转换为普通字符串错误，handler 无法识别并映射 503。
- 修复：构建错误命中 `ErrRequestBodySpool` 时直接 `%w` 包装返回，不提交 service 层 502；handler 保持既有 503 映射。
- RED：`TestGeminiMessagesCompatOutboundOpenFailurePreservesSpoolError` 观察到错误链只剩 `forced outbound open failure: request body spool failed` 的普通字符串，`errors.Is` 失败且响应已被提交。
- GREEN：同一测试确认 `errors.Is(err, ErrRequestBodySpool)` 为 true、service 未提交响应、上游零调用；handler 的 spool-open/materialization 503 测试同时通过。

### IMPORTANT：Signature Retry Passthrough Source

- 契约：signature retry 只使用 stripped body 做 Gemini 协议转换；账号 passthrough map/forward 规则始终从客户端 canonical original body 提取字段，与 wave 1 前语义一致。
- 修复：`prepareMessagesCompatSignatureRetryHandle` 将 `canonicalBody` 作为 passthrough source，将 `strippedBody` 仅用于转换。
- RED：测试捕获的 passthrough source 已删除顶层 thinking，并把 thinking block 降级为 text。
- GREEN：测试确认 passthrough source 与 canonical original JSON 等价，同时 retry outbound body 仍移除 stale signature。

### Wave 2 Verification

以下命令均通过：

```text
go test ./internal/service -run '(GeminiMessagesCompat|ForwardGeminiHandle)' -count=1
go test ./internal/handler -run '(GatewayHandler_(Messages|Responses)|GeminiV1Beta)' -count=1
go test ./internal/handler -run TestRequestBodyMemoryRetentionWhileUpstreamBlocked -count=1 -v
go build ./...
go vet ./...
go test ./... -count=1
git diff --check
git diff -- backend/internal/service/openai_ws_forwarder_v2.go backend/internal/service/openai_ws_protocol_forward.go
```

- 完整测试：PASS；`internal/handler` 78.536s，`internal/service` 112.121s。
- WS diff 命令无输出。
- 未修改依赖、配置、schema 或其他生产文件；`.comet/current-change.json` 保持未跟踪且未提交。

## Wave 3 Final Closure

状态：`PASS`

修复提交：`e30fdabb5 fix: preserve gemini transport spool errors`

### IMPORTANT：Gemini Messages Transport Spool Error

- 根因：`HTTPUpstream.Do` 可从重定向 `GetBody` 或 spool reader 返回 wrapped `ErrRequestBodySpool`，但 transport error 分支把它当作普通网络错误重试，最终经 `writeClaudeError(502)` 丢失 sentinel。
- 修复：关闭当前 request body 后，优先用 `errors.Is` 识别 `ErrRequestBodySpool` 并 `%w` 返回；不重试、不提交 service 层 502，由 handler 映射 503。
- RED：`TestGatewayHandler_MessagesGeminiTransportSpoolFailureReturns503AndReleasesAccountSlot` 观察到 5 次上游调用，最终状态 502，响应为 `Upstream request failed after retries`，耗时 14.42s。
- GREEN：同一真实 handler 测试仅调用 transport 1 次，返回 503，未出现 service 502 文案，账号槽位释放 1 次，耗时 1.508s。

### Clone Comment

- 在 attempt `strings.Clone` 前补充注释：gjson scalar 可能引用完整 materialized body backing array，必须在上游等待前复制切断引用。

### Wave 3 Verification

以下命令均通过：

```text
go test ./internal/service -run '(GeminiMessagesCompat|TransportSpool|ErrRequestBodySpool)' -count=1
go test ./internal/handler -run '(GatewayHandler_Messages|RequestBodySpool)' -count=1
go build ./...
go vet ./...
go test ./... -count=1
git diff --check
git diff -- backend/internal/service/openai_ws_forwarder_v2.go backend/internal/service/openai_ws_protocol_forward.go
```

- 完整测试：PASS；`internal/handler` 81.126s，`internal/service` 113.606s。
- WS diff 命令无输出。
- 未修改依赖、配置、schema 或其他生产文件；`.comet/current-change.json` 保持未跟踪且未提交。
