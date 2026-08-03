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
