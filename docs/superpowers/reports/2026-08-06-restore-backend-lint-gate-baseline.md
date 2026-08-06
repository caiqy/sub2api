# Restore Backend Lint Gate Baseline

## Identity

- Source base: `b576f73a22c4bf23d61727fc93950766a7e33929`
- Go: `go1.26.5 windows/amd64`
- golangci-lint: `2.9.0`
- Command from `backend`: `golangci-lint run ./... --max-issues-per-linter 0 --max-same-issues 0`
- Result: exit 1, 144 issues, 39 files (`ineffassign=140`, `staticcheck=3`, `unused=1`)

## Protected Inputs

| Path | Blob at source base |
| --- | --- |
| `backend/.golangci.yml` | `92ba3916948b4b859737c3c4831c7416dcd7f01e` |
| `backend/go.mod` | `7d5150f4a969df8a578e5bce8e6f5a01ec856823` |
| `backend/go.sum` | `72146c2305a91a48f92ac8fe2f9d888a2a1a2886` |
| `.github/workflows/backend-ci.yml` | `ee84c994ca2f1e27ae32eb02f25c3d094581b1ff` |
| `Makefile` | `da7c0c59fe67dfc8219ecfb2fbab1238fd0bbb55` |
| `backend/Makefile` | `0327160ff0959575ed6a8f950d7d257a96ae3ab0` |

这些路径在本 change 中不得修改。最终以相对 source base 的 blob/diff 复核，而不是仅检查未提交工作区。

## Batch A: Handler, Routes, QF1003

| File | Diagnostics at source base | Count |
| --- | --- | ---: |
| `internal/handler/gateway_handler.go` | `ineffassign`: 366, 406, 565, 1089, 1246, 2484, 2507 | 7 |
| `internal/handler/gateway_handler_chat_completions.go` | `ineffassign`: 181, 182 | 2 |
| `internal/handler/gateway_handler_responses.go` | `ineffassign`: 169; `staticcheck/QF1003`: 404 | 2 |
| `internal/handler/gemini_v1beta_handler.go` | `ineffassign`: 270, 384, 390, 404, 410 | 5 |
| `internal/handler/grok_media.go` | `ineffassign`: 145, 204, 205 | 3 |
| `internal/handler/image_task_handler.go` | `ineffassign`: 111 | 1 |
| `internal/handler/openai_alpha_search.go` | `ineffassign`: 104, 105 | 2 |
| `internal/handler/openai_chat_completions.go` | `ineffassign`: 152, 153 | 2 |
| `internal/handler/openai_gateway_count_tokens.go` | `ineffassign`: 143, 186, 232, 253 | 4 |
| `internal/handler/openai_gateway_handler.go` | `ineffassign`: 508, 687, 1170 | 3 |
| `internal/handler/openai_live.go` | `ineffassign`: 59, 71 | 2 |
| `internal/handler/request_body_memory_retention_test.go` | `staticcheck/QF1003`: 626, 628 | 2 |
| `internal/server/routes/gateway.go` | `ineffassign`: 481 | 1 |

Batch A total: 36 issues in 13 files (`ineffassign=33`, `staticcheck=3`).

## Batch B: Gateway, Anthropic, Bedrock, Antigravity

| File | Diagnostics at source base | Count |
| --- | --- | ---: |
| `internal/service/antigravity_gateway_claude.go` | `ineffassign`: 36, 116, 121, 213, 216, 234, 351, 354, 374 | 9 |
| `internal/service/antigravity_gateway_compat.go` | `ineffassign`: 88, 156, 275, 283 | 4 |
| `internal/service/antigravity_gateway_gemini.go` | `ineffassign`: 57, 163, 164, 165, 239, 242, 314, 316, 319 | 9 |
| `internal/service/antigravity_gateway_service_test.go` | `ineffassign`: 2122, 2132 | 2 |
| `internal/service/antigravity_gateway_upstream.go` | `ineffassign`: 66 | 1 |
| `internal/service/gateway_anthropic_apikey_passthrough_test.go` | `ineffassign`: 2460, 2470, 2496, 2506, 2545, 2563, 2564, 2604, 2614 | 9 |
| `internal/service/gateway_anthropic_passthrough.go` | `ineffassign`: 174, 199, 206, 433 | 4 |
| `internal/service/gateway_bedrock.go` | `ineffassign`: 96, 97, 223 | 3 |
| `internal/service/gateway_count_tokens.go` | `ineffassign`: 151, 200, 208, 209, 326, 327 | 6 |
| `internal/service/gateway_forward.go` | `ineffassign`: 422, 423, 430, 432, 450, 1010 | 6 |
| `internal/service/gateway_forward_as_chat_completions.go` | `ineffassign`: 153, 154, 156 | 3 |
| `internal/service/gateway_upstream_request.go` | `ineffassign`: 264, 265, 270 | 3 |
| `internal/service/gemini_chat_completions_compat_service.go` | `ineffassign`: 71, 73, 120, 140 | 4 |

Batch B total: 63 `ineffassign` issues in 13 files.

## Batch C: OpenAI, Gemini, Grok, Unused

| File | Diagnostics at source base | Count |
| --- | --- | ---: |
| `internal/service/openai_alpha_search.go` | `ineffassign`: 69, 70, 78, 79, 164, 165 | 6 |
| `internal/service/openai_gateway_cc_pipeline.go` | `unused`: 168 | 1 |
| `internal/service/openai_gateway_chat_completions.go` | `ineffassign`: 300, 301 | 2 |
| `internal/service/openai_gateway_chat_completions_raw.go` | `ineffassign`: 244, 245 | 2 |
| `internal/service/openai_gateway_count_tokens.go` | `ineffassign`: 100, 139 | 2 |
| `internal/service/openai_gateway_forward.go` | `ineffassign`: 816, 866, 867, 958, 989, 1031, 1046 | 7 |
| `internal/service/openai_gateway_grok.go` | `ineffassign`: 93, 94, 95 | 3 |
| `internal/service/openai_gateway_grok_chat_bridge.go` | `ineffassign`: 595, 596 | 2 |
| `internal/service/openai_gateway_messages.go` | `ineffassign`: 48, 322, 323, 324, 326, 415, 416, 446, 451, 460, 508 | 11 |
| `internal/service/openai_gateway_messages_chat_fallback.go` | `ineffassign`: 108, 109 | 2 |
| `internal/service/openai_gateway_passthrough.go` | `ineffassign`: 194, 195, 218 | 3 |
| `internal/service/openai_gateway_responses_chat_fallback.go` | `ineffassign`: 106, 108, 110 | 3 |
| `internal/service/openai_live.go` | `ineffassign`: 166 | 1 |

Batch C total: 45 issues in 13 files (`ineffassign=44`, `unused=1`).

## Closure Arithmetic

| Batch | Files | ineffassign | staticcheck | unused | Total |
| --- | ---: | ---: | ---: | ---: | ---: |
| A | 13 | 33 | 3 | 0 | 36 |
| B | 13 | 63 | 0 | 0 | 63 |
| C | 13 | 44 | 0 | 1 | 45 |
| **Total** | **39** | **140** | **3** | **1** | **144** |

## Verified Focused Test Inventory

以下 `-list` 命令已在 source base 上运行并确认匹配非空。

### Retained Heap

```powershell
$pattern = '^(TestAsyncImageRequestBodyMemoryRetentionWhileWorkersBlocked|TestRequestBodyMemoryRetentionWhileUpstreamBlocked)$'
go test ./internal/handler -list $pattern
```

预期 2 个 top-level tests：第一个内部运行 3 轮 async image 场景；第二个覆盖 25 条 handler/upstream 阻塞分支，每条分别比较 2MB 与 8.9MB 请求。

### Handler Spool And Replay

```powershell
$pattern = '^(TestGatewayHandler_MessagesAndResponsesReplayLargeBodiesAcrossFailover|TestOpenAIGatewayHandler_ChatAndEmbeddingsReplayMappedSpoolAcrossFailover|TestOpenAIGatewayHandler_ChatReplayRawSpoolAcrossFailoverWhenResponsesUnsupported|TestAsyncImageHandlerSpoolsReplaysAndCleansOwnedBody|TestAsyncImageHandlerSpoolsReplaysAndCleansOwnedMultipartBody|TestGrokMedia_GenerateEditVideoRejectUpstreamFailoverPreserveRequestSemantics|TestGrokMedia_MultipartSpoolPreservesFilesAndOmitsSnapshots)$'
go test ./internal/handler -list $pattern
```

预期 7 个 top-level tests。

### Service Spool, Retry, Failover

```powershell
$pattern = '^(TestAntigravityGatewayService_ClaudeForwardHandleSignatureRetryReparsesFileBackedCanonical|TestAntigravityRetryLoopReopensGeminiPayloadHandleForRetry|TestGatewayService_AnthropicPassthroughRetryRereadsHandleAfterForwardFirstAttemptBytes|TestForwardAsResponsesHandle_SpoolTransportErrorPreservesSentinel|TestGeminiMessagesCompatSignatureRetryBuildsHandleFromCanonical|TestOpenAIGatewayService_RejectedFieldRetryReturnsSpoolError|TestOpenAIForwardReusesBoundRequestBodyHandle|TestOpenAIForwardPreservesBoundRequestBodyHandleWhenHTTPDoErrors|TestOpenAILiveCreateUpstreamHandleTransportSpoolErrorClosesBodies)$'
go test ./internal/service -list $pattern
```

预期 9 个 top-level tests。
