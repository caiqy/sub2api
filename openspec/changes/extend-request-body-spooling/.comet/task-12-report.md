# Task 12 Report

- OpenAI Chat and Embeddings return OpenAI-format 503 for request-body spool sentinels before account failover or scheduling results are recorded.
- Embeddings preserves `ErrRequestBodySpool` through transport body-read failures; the handler does not write a fallback 502 first.
- Anthropic-backed Responses clears its handler-local request bytes after synchronous parsing and retains only the effective borrowed handle across attempts.
- Committed Responses streams do not append a second HTTP JSON error after a stream or spool failure.
- Replaced source-string coordinator checks with real Chat and Embeddings route fault injection, raw-read sentinel coverage, handle cleanup assertions, and committed-writer behavior checks. Grok media coverage remains in its dedicated behavior tests.
- Verification with `D:\cache\sub2api-task12` passed: Task12-focused handler/service tests, `go test ./internal/handler -count=1`, `go test ./internal/service -count=1`, and `go test -tags=unit ./internal/handler -count=1`.
