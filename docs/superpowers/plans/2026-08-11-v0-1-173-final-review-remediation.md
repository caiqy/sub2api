# v0.1.173 Final Review Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the four Task 35 Important findings without adding custom voice ownership or changing the final version, then restore `gap=0` and final-review approval.

**Architecture:** Reuse the current session sticky scheduler, unified audit coordinator and rate-limit service. Task 28 owns Voice/search sticky, STT duration and Realtime audit; Task 29 owns native search cooldown reconciliation. Product remediation ends before a fresh final-gate rerun and the existing Task 35 reviewer is resumed.

**Tech Stack:** Go, Gin, coder/websocket, gjson, testify, existing GatewayCache/securityaudit/RateLimitService, PowerShell 7 command shell.

## Global Constraints

- Do not add persistent or TTL custom voice-ID ownership mapping.
- Do not add dependencies, schema, migrations, generated output, configuration or frontend changes.
- Keep `backend/cmd/server/VERSION` exactly `0.1.173.1`.
- Do not edit or commit `.comet/current-change.json`.
- Do not push, tag, release, deploy, operate a server or use remote validation.
- Docker-only migration 220 and Task 30 race remain `UNVERIFIED`, never PASS.
- Every product change starts with a focused RED and ends with a fresh reviewer gate.

---

### Task 1: Close Task 28 Voice/Search Routing, Billing And Audit Findings

**Files:**
- Modify: `backend/internal/handler/grok_audio.go`
- Modify: `backend/internal/handler/gateway_web_search.go`
- Modify: `backend/internal/service/grok_audio.go`
- Test: `backend/internal/handler/grok_audio_billing_test.go`
- Test: `backend/internal/handler/gateway_web_search_test.go`
- Test: `backend/internal/service/grok_audio_test.go`
- Test helper only if needed: `backend/internal/handler/terminal_failed_usage_test.go`

**Interfaces:**
- Consumes: `(*OpenAIGatewayService).GenerateSessionHash(*gin.Context, []byte) string`, `SelectAccountWithSchedulerForCapability(..., sessionHash string, ...)`, `SelectAccountWithLoadAwareness(..., sessionHash string, ...)`, `checkSecurityAuditStage(...)`.
- Produces: `ProxyGrokRealtime(..., eventGuard func([]byte) error) (string, error)` and `grokRealtimeAuditBody([]byte) []byte`.
- Does not produce a voice-ID key, repository or migration.

- [ ] **Step 1: Add focused sticky RED tests**

Extend the existing handler fixtures with in-memory `GatewayCache` state. Add one Voice test and one native search test that issue two requests with the same explicit session header, change ordinary account priority between requests, and require the second request to keep the first selected account. Assert the cache observed a non-empty session key; do not assert any custom voice ID binding.

```go
func TestGrokVoice_ReusesStandardSessionStickyAccount(t *testing.T) {
    // First TTS request selects account A and binds the standard session hash.
    // Reverse A/B priority, repeat X-Session-ID, and require account A again.
}

func TestGatewayHandler_WebSearchReusesStandardSessionStickyAccount(t *testing.T) {
    // Use the same cache for GatewayService and OpenAIGatewayService hash generation.
    // Repeat the same session header/query after reversing priority; require account A.
}
```

- [ ] **Step 2: Add focused STT and Realtime audit RED tests**

Add a table test for the STT estimator and relay/audit tests for Realtime:

```go
func TestEstimateGrokVoiceAudioUsage_STTPreservesUpstreamDurationOverRequestSize(t *testing.T) {
    got := estimateGrokVoiceAudioUsage("stt", bytes.Repeat([]byte("a"), 160000), "audio/wav", []byte(`{"duration":2}`), 500*time.Millisecond)
    require.InDelta(t, 2.0/3600.0, got.DurationOrUnits, 1e-9)
}

func TestGrokRealtimeAuditBodyExtractsPromptTextButNotAudio(t *testing.T) {
    require.Contains(t, string(grokRealtimeAuditBody([]byte(`{"type":"session.update","session":{"instructions":"safe text"}}`))), "safe text")
    require.Nil(t, grokRealtimeAuditBody([]byte(`{"type":"input_audio_buffer.append","audio":"base64-secret"}`)))
}

func TestProxyGrokRealtimeGuardBlocksBeforeUpstreamWrite(t *testing.T) {
    // Client sends one text event; guard returns sentinel; fake upstream WriteJSON count stays zero.
}

func TestGrokRealtimeAuditUsesIndependentEventStages(t *testing.T) {
    // Call the handler audit seam twice with one Context and require two coordinator evaluations/enqueues.
}
```

- [ ] **Step 3: Run RED commands**

Run from `backend`:

```powershell
go test -tags unit ./internal/handler -run '^(TestGrokVoice_ReusesStandardSessionStickyAccount|TestGatewayHandler_WebSearchReusesStandardSessionStickyAccount|TestGrokRealtimeAuditBodyExtractsPromptTextButNotAudio|TestGrokRealtimeAuditUsesIndependentEventStages)$' -count=1
go test ./internal/service -run '^(TestEstimateGrokVoiceAudioUsage_STTPreservesUpstreamDurationOverRequestSize|TestProxyGrokRealtimeGuardBlocksBeforeUpstreamWrite)$' -count=1
```

Expected: failures caused by empty session hashes, request-size inflation of upstream STT duration, absent Realtime prompt extraction/guard, or missing signatures. Preserve the exact RED output in the ignored Task 35 remediation report.

- [ ] **Step 4: Pass standard session hashes into existing schedulers**

In `GrokVoice`, compute once after reading the body and pass it to selection and slot admission:

```go
sessionHash := h.gatewayService.GenerateSessionHash(c, body)
// SelectAccountWithSchedulerForCapability(..., "", sessionHash, ...)
// acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, ...)
```

In `GrokRealtime`, use `GenerateSessionHash(c, nil)` so only existing explicit session signals apply. In `WebSearch`, use the already-wired `openAIGatewayService` to hash the normalized audited query body and pass that hash to `SelectAccountWithLoadAwareness`; nil service falls back to an empty hash.

- [ ] **Step 5: Preserve upstream STT duration precedence**

Restore the exact upstream v0.1.173 order after the local remediation overrode it:

```go
if secs <= 0 {
    secs = elapsed.Seconds()
}
if secs <= 0 {
    secs = clientSecs
}
if secs <= 0 {
    secs = sizeFloor
}
```

Do not add an audio parser or otherwise repair inherited upstream fallback heuristics. Keep nil usage when every value is non-positive and restore the upstream comment.

- [ ] **Step 6: Guard prompt-bearing Realtime events before forwarding**

Implement `grokRealtimeAuditBody` with an event-type switch:

```go
switch eventType {
case "session.update":
    // session.instructions
case "conversation.item.create":
    // item.content input_text/text and transcript fields; exclude audio bytes
case "response.create":
    // response.instructions and textual response.input
default:
    return nil
}
```

Normalize non-empty extracted text to an OpenAI Responses audit body. Add one optional `eventGuard func([]byte) error` argument to `ProxyGrokRealtime`; invoke it after JSON validation and before `upstream.WriteJSON`. The handler closure calls `checkSecurityAuditStage(..., "openai_responses", model, auditBody, "grok_realtime_turn")`. A block returns a request-local sentinel, writes no upstream event and closes the socket with `coderws.StatusPolicyViolation`.

- [ ] **Step 7: Run GREEN and Task 28 canonical gates**

Run from `backend`:

```powershell
go test -tags unit ./internal/handler -run '^(TestGrokVoice_|TestGatewayHandler_WebSearch|TestGrokRealtime|TestIsExpectedGrokRealtimeClose)' -count=1
go test ./internal/service -run '^(TestBuildGrokVoiceURL_|TestForwardGrokVoice_|TestEstimateGrokVoiceAudioUsage_|TestProxyGrokRealtime)' -count=1
go test -tags unit ./internal/handler ./internal/service ./internal/pkg/xai -run '(GrokMedia|GrokAudio|Voice|Video|WebSearch|SearchCount|SearchSurcharge|ImageGeneration|RequestBody|Sticky|Failover|RecordUsage)' -count=1
go test -tags unit ./internal/handler ./internal/service -run '(OpenAIImages|SecurityAudit|PromptAudit|RequestBody|BodyRetention|Grok.*Billing|Search.*Billing|OpenAIRecordUsageInputsCarryQuotaPlatform)' -count=1
golangci-lint run ./internal/handler/... ./internal/service/...
```

Expected: PASS and `0 issues`; VERSION and protected paths unchanged.

- [ ] **Step 8: Commit Task 28 remediation**

```powershell
git add -- backend/internal/handler/grok_audio.go backend/internal/handler/gateway_web_search.go backend/internal/service/grok_audio.go backend/internal/handler/grok_audio_billing_test.go backend/internal/handler/gateway_web_search_test.go backend/internal/service/grok_audio_test.go backend/internal/handler/terminal_failed_usage_test.go
git diff --cached --check
git commit -m "fix: preserve Grok voice and search gateway contracts"
```

Stage only files that actually changed; omit the optional test-helper path when unchanged.

---

### Task 2: Close Task 29 Native Search Cooldown Finding

**Files:**
- Modify: `backend/internal/service/gateway_service.go`
- Test: `backend/internal/service/openai_gateway_grok_search_billing_test.go`

**Interfaces:**
- Consumes: `(*RateLimitService).HandleUpstreamError(context.Context, *Account, int, http.Header, []byte, ...string) bool` and existing `UpstreamFailoverError`.
- Produces: no new exported API; search errors update the same account/team/model cooldown state as other Grok paths before failover.

- [ ] **Step 1: Write the 429 cooldown RED**

```go
func TestDoGrokNativeResponsesJSON_ReconcilesRateLimitBeforeFailover(t *testing.T) {
    // OAuth account has team_id and grok-4.5 -> mapped-search-model.
    // Upstream returns 429 with retry/reset headers.
    // Require UpstreamFailoverError and require the mapped team/model to be cooled.
}
```

Use a unique team ID and clear only that test key on cleanup. Keep the existing scheduler exclusion tests as the second half of the contract.

- [ ] **Step 2: Run RED**

Run from `backend`:

```powershell
go test -tags unit ./internal/service -run '^TestDoGrokNativeResponsesJSON_ReconcilesRateLimitBeforeFailover$' -count=1
```

Expected: fail because `DoGrokNativeResponsesJSON` returns failover without marking the mapped team/model cooldown.

- [ ] **Step 3: Reuse RateLimitService before returning errors**

After reading `respBytes` and before the existing `resp.StatusCode >= 400` return logic:

```go
if s.rateLimitService != nil {
    s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBytes, upstreamModel)
}
```

Do not change the existing client/failover status mapping. Do not call `markGrokTeamModelRateLimit` directly from the gateway method.

- [ ] **Step 4: Run GREEN and Task 29 focused gates**

Run from `backend`:

```powershell
go test -tags unit ./internal/service -run '^(TestDoGrokNativeResponsesJSON|TestGrokWebSearchSelectionSkipsCooledStickyAccount|TestGrokWebSearchLoadAwareSelectionSkipsCooledStickyAccount|TestGrokTeamModelRateLimit_)' -count=1
go test -tags unit ./internal/service ./internal/handler -run '(Grok.*Free|TeamModel|ModelRateLimit|StreamIdle|WebSearch|Scheduler|Sticky|WaitPlan)' -count=1
golangci-lint run ./internal/service/...
```

Expected: PASS and `0 issues`; VERSION and protected paths unchanged.

- [ ] **Step 5: Commit Task 29 remediation**

```powershell
git add -- backend/internal/service/gateway_service.go backend/internal/service/openai_gateway_grok_search_billing_test.go
git diff --cached --check
git commit -m "fix: reconcile Grok search cooldowns"
```

---

### Task 3: Review Both Owning Remediations

**Files:**
- Create ignored evidence only: `.superpowers/sdd/2026-08-06-staged-merge-upstream-v0-1-171/task-35-remediation-report.md`
- Do not modify product files in this task.

- [ ] **Step 1: Dispatch a fresh Task 28 reviewer**

Review the Task 28 remediation commit against design sections Standard Session Sticky, STT Duration and Realtime Audit. Require findings-first output, `Spec: PASS` and `Quality: APPROVED`. Any finding returns to Task 1 with a focused RED.

- [ ] **Step 2: Dispatch a fresh Task 29 reviewer**

Review the Task 29 remediation commit against the native search cooldown design. Require proof that response status/headers/body and final mapped model reach existing reconciliation while failover semantics remain unchanged. Any finding returns to Task 2 with a focused RED.

- [ ] **Step 3: Record accepted SHAs, RED/GREEN and review sessions**

Write only the ignored remediation report. Include commit scopes and confirm no voice-ID ownership, dependency, migration, generated, VERSION or selector changes.

---

### Task 4: Re-run Final Gates From The Remediation HEAD

**Files:**
- Update ignored evidence: `.superpowers/sdd/2026-08-06-staged-merge-upstream-v0-1-171/task-34-report.md`
- Do not commit in this task.

- [ ] **Step 1: Run the full local gate sequence**

```powershell
make test
make 'VERSION=0.1.173.1' 'SHELL=D:/scoop/shims/bash.exe' build
Push-Location backend; try { golangci-lint run ./... } finally { Pop-Location }
pnpm --dir frontend run lint:check
pnpm --dir frontend run typecheck
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
```

- [ ] **Step 2: Re-run immutable identity and topology checks**

Require committed VERSION `0.1.173.1`, 24/24 authoritative migration blob matches, all four peeled tags as ancestors, and exactly one first-parent merge second-parent match per tag after source base `b576f73a22c4bf23d61727fc93950766a7e33929`.

- [ ] **Step 3: Re-run local Docker-conditional integration and static boundaries**

Run the migration 220 command from `backend`; retain `UNVERIFIED` unless the target test body prints a real PASS. Run unmerged/index/conflict/whitespace/full tracked/protected-path checks. Final status may contain only `?? .comet/current-change.json`.

---

### Task 5: Restore Task 35 Approval And Close The Reports

**Files:**
- Modify: `docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-build.md`
- Modify: `docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-verify.md`
- Modify after report commit: plan, OpenSpec tasks and Comet progress as a separate completion checkpoint.

- [ ] **Step 1: Re-run strict validation**

```powershell
comet classic openspec -- validate staged-merge-upstream-v0-1-171 --strict
```

Expected: `Change 'staged-merge-upstream-v0-1-171' is valid`.

- [ ] **Step 2: Resume the Task 35 thorough reviewer**

Resume session `ses_00fbde930ffe5kGRbFCuWCmQxP` with the two remediation commits, fresh Task 34 evidence and focused reviews. Require all four prior Important findings closed, matrix `gap=0`, `Spec: PASS`, `Quality: APPROVED`, and `Ready for final report: YES`.

- [ ] **Step 3: Update exactly the two final reports**

Preserve the old v0.1.171 Verify content as history. Append current v0.1.173 provenance, four tag objects/peeled SHAs/merge parents, VERSION, 172/173 file surface, focused/full gates, two stable generates, 24 identities, Docker/race residuals, capability matrix `gap=0`, strict validation, final review verdict and no-remote/no-release/no-deploy statement.

- [ ] **Step 4: Commit the report boundary**

```powershell
git add -f -- docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-build.md docs/superpowers/reports/2026-08-06-staged-merge-upstream-v0-1-171-verify.md
git diff --cached --name-only
git diff --cached --check
git commit -m "docs: record v0.1.173 verification"
```

Require exactly two staged/committed files.

- [ ] **Step 5: Complete workflow metadata in a separate checkpoint**

Check Task 35 and OpenSpec 6.8, record final reviewer/gates/report commit in Comet progress, verify the selector is not staged, and commit only workflow docs. Do not push, tag, release, deploy or archive without the next Comet routing decision.
