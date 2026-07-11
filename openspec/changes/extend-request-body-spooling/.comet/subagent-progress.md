# Subagent Progress

- Change: extend-request-body-spooling
- Review mode: thorough
- Current task: Task 5: OpenAI Chat Completions 与 Embeddings 接入
- OpenSpec mapping: 2.2 将 OpenAI `/v1/chat/completions` 与 Embeddings 迁移到 coordinator，使最终 outbound body 通过 effective handle 支持 retry/failover 重放。
- Stage: blocked
- Base commit: 2eda86b94f138d8a1e683c347e38f8f26a5f39fe
- Implementation commit: 11e2ccdc
- Changed files: backend/internal/handler/openai_chat_completions.go; backend/internal/handler/openai_embeddings.go; backend/internal/handler/openai_request_body_spooling_test.go; backend/internal/service/openai_embeddings.go
- RED evidence: no behavioral RED; initial compile failure was only unused imports and is not accepted as TDD behavior evidence.
- GREEN evidence: focused Chat/Embeddings request-body and broader Chat/Embeddings handler/service tests pass.
- Risk signals: DONE_WITH_CONCERNS; external input and cross-module change; handlers reportedly retain full `body` for post-forward audit/hash; tests do not exercise real failover or blocked-upstream snapshot retention.
- Review round: 2/2
- Review status: blocked after 2/2 fix rounds
- Unresolved findings: preserved uncommitted implementation removes handler body capture, restores raw session/hash semantics, adds pre-create dedupe, makes Chat transformed/raw and Embeddings network waits handle-backed, and passes `go test ./internal/handler ./internal/service -count=1` plus focused tests. Remaining evidence gap: existing production Chat terminal fixture forces the WS HTTP bridge and cannot exercise the `httpUpstream.Do` 12MB mapped first-failover-then-success blocked lifecycle or bounded usage/ops assertions. User must choose compositional evidence or authorize a dedicated HTTP transport fixture; preserved Task 5 source changes must not be discarded.
