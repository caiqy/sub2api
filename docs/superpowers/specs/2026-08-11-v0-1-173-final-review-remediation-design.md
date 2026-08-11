# v0.1.173 Final Review Remediation Design

## Context

Task 35 thorough review initially reported four Important findings in the v0.1.173 extension; the STT item was later scope-corrected by the user as inherited upstream behavior:

1. Voice/Realtime and native web search do not pass a session hash into their existing schedulers.
2. STT fallback ordering was initially flagged, but the user later confirmed that this staged merge must preserve upstream behavior rather than repair an inherited upstream billing policy.
3. Grok Realtime forwards prompt-bearing client events without unified security audit.
4. Native web search returns failover errors without invoking existing Grok cooldown reconciliation.

`custom-voices` is an upstream v0.1.173 capability. This change has no local persistent voice-ownership feature. The remediation therefore preserves upstream resource semantics and does not add a voice-ID ownership table, migration, or cache key.

## Chosen Approach

Use the existing scheduler, sticky cache, audit coordinator, rate-limit service and tests. Make two owning remediation commits:

- Task 28: Voice/search session sticky, preservation of upstream STT duration ordering and Realtime prompt audit.
- Task 29: native search upstream-error reconciliation and cooldown.

No dependency, schema, migration, generated, VERSION or frontend change is needed.

## Task 28 Design

### Standard Session Sticky

- `GrokVoice` derives the normal OpenAI/Grok session hash from the current request and already-buffered body, then passes it to account selection and slot admission.
- `GrokRealtime` derives the same hash from explicit client session signals and passes it to selection and slot admission.
- `GatewayHandler.WebSearch` derives a hash from its normalized audited query body through the already-wired OpenAI gateway service and passes it to `SelectAccountWithLoadAwareness`.
- Existing scheduler/cache code remains responsible for lookup, eligibility recheck, failover exclusion, cooldown filtering and binding.
- No custom voice ID is parsed or bound. Requests without a usable session signal retain ordinary upstream-style account scheduling.

### STT Duration

For STT billing seconds:

1. Read a positive upstream response duration when present.
2. Otherwise use request elapsed time.
3. Otherwise use positive client `duration_seconds`, retaining upstream's under-report safeguard: when that client value is less than half of the larger request-size/elapsed estimate, use the estimate instead.
4. Otherwise use the inherited request-size estimate.

This is the exact upstream v0.1.173 ordering. The staged merge does not add a WAV parser or otherwise change the inherited missing-duration billing policy. In particular, client/body estimates must never override an authoritative upstream duration.

### Realtime Audit

- Add one optional event-guard callback to the existing `ProxyGrokRealtime` client-to-upstream relay.
- Before each JSON event is written upstream, extract only prompt-bearing `instructions`, text, transcript or input fields from known Realtime event shapes.
- Normalize extracted text into the existing OpenAI audit body format and call `checkSecurityAuditStage` with a non-HTTP Realtime stage so each event is audited independently.
- Audio/base64 payloads and events with no prompt text produce no audit body and continue unchanged.
- A blocking decision stops the event before upstream write and closes the client WebSocket with policy-violation status. Invalid JSON keeps the existing protocol-error behavior.

## Task 29 Design

`GatewayService.DoGrokNativeResponsesJSON` will call the existing `RateLimitService.HandleUpstreamError` after reading an upstream error response and before returning its existing failover error. It passes the selected account, response status/headers/body and final mapped upstream model.

This reuses current account state, Grok team+model 429 cooldown, reset parsing and scheduler exclusion. No new cooldown state or timeout is introduced. Existing failover response behavior remains unchanged.

## Error Boundaries

- Sticky cache read/write failure remains best effort and cannot block a request.
- Audit block prevents only the current prompt-bearing Realtime event and terminates that socket; it does not persist account failure state.
- Search cooldown reconciliation never replaces the current `UpstreamFailoverError`; the current request may still select another account.
- STT keeps the upstream fallback and nil behavior unchanged.

## TDD And Verification

Add focused RED tests before implementation:

- Voice and web search pass a non-empty standard session hash into selection and reuse an existing sticky account; no voice-ID ownership assertion.
- STT preserves upstream response/elapsed/client/body precedence and never inflates an upstream duration with request size.
- Realtime text/instruction events are audited before upstream write, blocked text is not forwarded, and audio-only events bypass prompt extraction.
- Native search 429 invokes existing reconciliation and makes a subsequent scheduler selection exclude the cooled team/model account.

After each owning commit, run its focused Task 28 or Task 29 canonical bundle and related lint. Then rerun the final Task 34 full gates, two stable generates, 24 migration identities and four-stage topology from the remediation HEAD. Resume the same Task 35 reviewer; only an approved `gap=0` result permits the final two-report commit.

## Explicit Non-Goals

- Persistent or TTL voice-ID ownership mapping.
- New sticky/cache abstraction.
- New cooldown policy or storage.
- Fixing inherited upstream STT missing-duration billing heuristics.
- Realtime audio transcription or audio-content moderation.
- Pricing, schema, migration, generated, dependency, frontend or VERSION changes.
