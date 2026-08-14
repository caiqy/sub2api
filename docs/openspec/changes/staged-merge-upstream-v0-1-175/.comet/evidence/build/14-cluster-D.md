# Task 11 Stage 2 Cluster D Evidence

## Scope

- Branch: `feature/20260813/staged-merge-upstream-v0-1-175`.
- Base HEAD: `4f2a66bbf210d60787432291fb6aa29cc26b7034`.
- `backend/cmd/server/VERSION`, plan/tasks/spec/design/.comet.yaml, existing commits, remotes, and tags were not modified.

## Required References

- Read Task 11 Cluster D brief, `design.md` decision 5/6 (the current design has no `8.2` heading), and the `x_search 与 Responses 探测不破坏本地网关契约` Scenario.
- Compared `service.SetUsageRequestBody` with `middleware.UsageDetailCapture`, the existing root `/web_search` registration, and `BuildUsageDetailSnapshot` response-capture timing.

## RED To GREEN

### Root `/x_search` usage detail capture

- RED command: `go -C backend test -tags=unit -count=1 -run '^TestGatewayRoutesBareVoiceAndSearchPathsInstallUsageDetailCapture$' ./internal/server/routes`.
- Observed failure: `gateway_test.go:567: path=/x_search should install usageDetailCapture before apiKeyAuth`.
- Root cause: the bare `/x_search` registration omitted `usageDetailCapture`; therefore `service.SetUsageRequestBody` found no collector and returned without retaining a detail snapshot.
- Fix: insert the same `usageDetailCapture` middleware used by bare `/web_search`, before authentication.
- GREEN: the same command passed.

### Search usage snapshot preservation

- Existing focused handler check first exposed the upstream-expanded standalone request struct leaking empty x_search fields into the existing `/web_search` normalized request snapshot. Optional `input` and x_search-only fields now use `omitempty`, preserving the prior `{query,max_results}` shape while retaining supplied x_search fields.
- After that correction, RED command `go -C backend test -tags=unit -count=1 -run '^TestGatewayHandler_WebSearchFailoverRecordsFinalMappedUpstreamModel$' ./internal/handler` failed at `gateway_web_search_test.go:190`: the recorded `ResponseBody` was empty.
- Root cause: `BuildUsageDetailSnapshot` ran before `c.JSON`; its returned copy could not see the later response writer capture.
- Fix: write the successful JSON response first, then build the detail snapshot and submit the existing mandatory one-per-search usage task. Account selection, audit, request ID, `SearchCount: 1`, and billing calls are unchanged.
- GREEN: the same handler command passed.

## Upstream Review

- `c4d883b8d`: Chat/Responses conversions retain `x_search`, its filters, and a valid x_search tool choice while leaving unsupported service tools dropped. This is conversion-only and does not bypass the local body coordinator, audit, replay/release, or accounting paths. Focused apicompat x_search tests passed.
- `0de6d7e9b`: standalone `/x_search` delegates to the existing audited failover path and emits exactly one `SearchCount: 1` usage record with a unique `x_search:` request ID. This cluster fixes its omitted root capture and response-snapshot ordering; no duplicate billing path was found.
- `fd9ce5328`: probe responses with `status=failed` or `status=incomplete` because of `max_output_tokens` remain unknown and skip `UpdateExtra`; conclusive responses still persist their result. This preserves the local Responses route instead of a durable false downgrade. Focused service tests passed.
- `678eb22a4`: Realtime waits for both relay pumps, retains the audit guard exit, and bills only after observed audio and positive duration with a per-session ID. Focused service and handler tests passed; no gateway-side interaction requiring a Cluster D change was found.

## Verification

- `go -C backend test -tags=unit -count=1 -run '(XSearch|WebSearch|ResponsesProbe|GrokRealtime|BareVoiceAndSearchPathsInstallUsageDetailCapture)' ./internal/pkg/apicompat/ ./internal/service/ ./internal/handler/ ./internal/server/ ./internal/server/routes/`: PASS.
- `go -C backend build ./...`: PASS.
- `git diff --check`: PASS.

## Expected Paths

- `backend/internal/handler/gateway_web_search.go`
- `backend/internal/handler/openai_x_search.go`
- `backend/internal/server/routes/gateway.go`
- `backend/internal/server/routes/gateway_test.go`
- `docs/openspec/changes/staged-merge-upstream-v0-1-175/.comet/evidence/build/14-cluster-D.md`

## Local Handoff

- Updated the ignored `.superpowers/sdd/2026-08-12-staged-merge-upstream-v0-1-175/subagent-progress.md`; it is intentionally outside this commit.

## Residual Risk

- Focused unit tests and `go build` do not call a live xAI upstream or run PostgreSQL/Redis-backed usage persistence. The local audit, failover, request lifecycle, and single-record paths are exercised with in-process fixtures.

## Round 1 Follow-up

### x_search behavior coverage

- Added `TestGatewayHandler_XSearchRecordsSourcesUsageAndSnapshot`, which invokes `GatewayHandler.XSearch` through Gin with `UsageDetailCapture` installed. It verifies `x_search_call.action.sources` reaches the client result, a native x_search upstream body is sent once, the recorded usage has an `x_search:` request ID, one configured search-price unit is charged, and the preview request plus response-body snapshots retain x_search fields and the extracted source.
- Added `TestGatewayHandler_XSearchAuditRejectsBeforeDispatch`, which invokes the same handler with a blocking audit coordinator and verifies the 403 block plus the audited query body before scheduling or dispatch.
- Mutation RED command: temporarily change the standalone search result from `SearchCount: 1` to `SearchCount: 0`, then run `go -C backend test -tags=unit -count=1 -run '^TestGatewayHandler_XSearchRecordsSourcesUsageAndSnapshot$' ./internal/handler`.
- Observed RED: `gateway_web_search_test.go:286` reported an expected total cost of `1` and actual `0`, proving the test observes the single-search surcharge rather than only response formatting.
- GREEN: restored the existing `SearchCount: 1`; `go -C backend test -tags=unit -count=1 -run '^(TestGatewayHandler_XSearchRecordsSourcesUsageAndSnapshot|TestGatewayHandler_XSearchAuditRejectsBeforeDispatch|TestGatewayHandler_WebSearchFailoverRecordsFinalMappedUpstreamModel)$' ./internal/handler` passed.

### Realtime production entry and comment

- Replaced the unused `awaitGrokRealtimeAudioObserved` helper test with `TestProxyGrokRealtimeReadsAudioObservedAfterRelayExit`. Its controlled relay holds the upstream pump until the proxy is already waiting, sends an audio event, and verifies the direct `ProxyGrokRealtime` return reports `audioObserved=true` only after relay exit.
- Removed the now-unused helper. No relay production behavior was changed.
- Updated `DoGrokNativeResponsesJSON` documentation to name both `/v1/web_search` and `/x_search` callers.

### Probe Decision Boundary

- Per the main-window ruling, `responsesProbeVerdictIsConclusive` and its behavior were not modified. The implementation remains limited to `status=failed` and `status=incomplete` with `incomplete_details.reason=max_output_tokens` retaining unknown; all other statuses retain their existing behavior.

### Round 1 Verification

- `go -C backend test -tags=unit -count=1 -run '^(TestGatewayHandler_XSearchRecordsSourcesUsageAndSnapshot|TestGatewayHandler_XSearchAuditRejectsBeforeDispatch|TestGatewayHandler_WebSearchFailoverRecordsFinalMappedUpstreamModel)$' ./internal/handler`: PASS.
- `go -C backend test -tags=unit -count=1 -run '^(TestProxyGrokRealtimeReadsAudioObservedAfterRelayExit|TestProxyGrokRealtimeUsesMappedModelInUpstreamURL|TestProxyGrokRealtimeGuardBlocksBeforeUpstreamWriteAndWaitsForPumps|TestProxyGrokRealtimePrefersGuardAfterOrdinaryPumpExit|TestDoGrokNativeResponsesJSON)' ./internal/service`: PASS.
- `go -C backend build ./...`: PASS.
- `git diff --check`: PASS.

### Round 1 Expected Paths

- `backend/internal/handler/gateway_web_search_test.go`
- `backend/internal/service/gateway_service.go`
- `backend/internal/service/grok_audio.go`
- `backend/internal/service/grok_audio_test.go`
- `docs/openspec/changes/staged-merge-upstream-v0-1-175/.comet/evidence/build/14-cluster-D.md`
