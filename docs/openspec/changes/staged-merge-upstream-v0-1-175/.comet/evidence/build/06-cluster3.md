# 06-cluster3 - OpenAI error and unified audit

## Status

Tests-only review-fix round 2 is GREEN. The pending commit strengthens three existing test files and this evidence only; it does not change production code. The brief-wide `-run` command selected 1189 tests, recorded 1186 `PASS` lines and 0 `SKIP` lines; both service and handler exited 0.

## RED / GREEN

- Fresh RED: five top-level handler tests failed with six `FAIL` lines, including the failed capacity subtest. The failures were stale test contracts, not production behavior.
- GREEN: `TestResponsesFinalHandleReplayAcrossFailover` compares all non-fingerprint fields and checks installation/session/thread/window fields individually, requiring at least one account-derived stable field to differ for accounts 1 and 2.
- GREEN: the OAuth capacity case keeps all four account-71 bodies equal, then verifies the account-72 body has the same non-fingerprint payload and a different account-derived stable fingerprint field.
- GREEN: the service replay regression uses an actual second OAuth account. Same account plus same body reuses IDs; second account checks installation/session/thread/window fields individually while preserving the non-fingerprint payload; same account with a different body receives a new turn ID.
- GREEN: image failed-usage tests preserve usage logs and multipart metadata redaction while deterministic upstream 400 remains `400 invalid_request_error`, including the multipart client response `error.type`.
- GREEN: cyber billing-order test explicitly scopes group 1/model `gpt-5` and sets `cyber_policy_exclude_from_ban_count`; mandatory billing completes before the blocking `CreateLog` returns.

## Invariants

- Deterministic 400 remains client-facing 400: `TestHandleErrorResponse_Deterministic400*` GREEN.
- HTML 403 avoids account penalties: `TestHandleUpstreamError_OpenAIHTML403*` GREEN.
- OAuth image streams, incomplete image completion, empty `response.completed`, and visible-output TTFT: matching service tests are GREEN.
- Capacity/pool budget: default/range coverage, zero retry, fixed 500 ms pool retry, request-scoped exponential 500 ms backoff capped at 8 s, exhaustion-before-switch, no early account penalty, and pool 401/403 retry are GREEN.
- Audit: HTTP completion caching, WS turn dedupe, failed/flagged WS rechecks, prompt-audit-before-side-effects, cyber event scope, and WS cyber mark tests are GREEN.

## Evidence Correction

Earlier evidence incorrectly attributed the cyber test to an empty moderation config being out of scope. Empty config normalizes to `all_groups=true` and model filter `all`; this round uses explicit scope and excludes ban-count work so the test isolates the intended billing-before-CreateLog ordering.

All raw list, RED, GREEN, and test logs are colocated in this directory.

## Round 2 Rerun

- Five focused tests: handler and service exit 0.
- Brief-wide `-list`: 1189 selected tests.
- Brief-wide service `-run`: exit 0 (67.360s).
- Brief-wide handler `-run`: exit 0 (74.126s).
