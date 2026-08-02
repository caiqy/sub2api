# v0.1.169 Staged Merge Final Verification

## Verdict And Scope

**Verdict: PASS with two manual boundaries and one Docker/Testcontainers residual.** This is an independent final review of the immutable source base, staged topology, final local gates, capability evidence, and operational boundary. It consumes the existing build ledger and does not represent a fresh test run.

- Immutable source base: `e9a0e4aa53b5d9d5f5c84986cfadd8098dc8e4f3`.
- Final version authority: `backend/cmd/server/VERSION = 0.1.169.1`.
- Current pre-report `HEAD`: `9f3ed146dcdcf83dfea9d47d50ed8b3684141bc5`; it retains exactly the three required first-parent merge commits below.
- Final full-gate source anchor: `b54cd46a45ccf934885a2ab66597e386ebecbf99`. The Task 16 service-target correction was run at `8bbaafb0473afe2693b6763dd760cd9860de21d0`; Task 17 topology evidence was recorded from `8b9934393c9c503c3aabf7517f0bf3b69f7db2b3`.

| Merge commit | Tag | First parent | Exact second parent |
| --- | --- | --- | --- |
| `c7ae76df77755b5b84b26b91606d37efc13b5deb` | `v0.1.166` | `e9d2ce48e23391f12a255ca9430d3f16bfd7fea3` | `dc893dd0b8eab41df5be595ae9fcd1aa74a062b8` |
| `de4264ba5d15ca1024da51846f43bf48b02a9882` | `v0.1.168` | `cd78fa1d5406a9ab468e47806c5ce94ac69c79f1` | `99c8e4bf7564823bafbab369acab6539e734c1bb` |
| `827369f76f8f301320759a0dc85b11ab05a7a1d6` | `v0.1.169` | `e6b163fcb62e608b7af0ca0f872e25de9e2fb516` | `26d894ef4f50645a4bf1030e378ac892f17d0223` |

The final Task 16 focused suite selected all nine service, nine handler, three route, and four frontend-file targets after the service correction; all exited `0`. The final full gate recorded `make test`, `make "VERSION=0.1.169.1" "SHELL=D:/scoop/shims/bash.exe" build`, two `make -C backend generate` runs, generated-path diffs, whitespace/index/unmerged/conflict checks, migration OID checks, and final VERSION check as exit `0`/PASS. `make test` reported default and `unit` backend suites, `golangci-lint` `0 issues.`, and frontend `225` files / `1698` tests. The two generator passes left no `backend/ent` or `backend/cmd/server/wire_gen.go` diff.

Residual boundaries are deliberate: row 9 remains manual for the Passkey joined-login path; row 12 remains manual for the `/model-plaza` embedded/authentication route boundary; row 10 remains unverified because Docker/Testcontainers was unavailable. Existing Browserslist, Vue/router-link, jsdom expected-error, intlify missing-key, Vite import, and chunk-size warnings were non-failing ledger warnings.

## Canonical 14-Row Matrix

| # | Canonical capability | Status | Auditable evidence / call chain / generated artifact |
| --- | --- | --- | --- |
| 1 | layered scheduler, DB recheck, WaitPlan fallback | `protected` | `OpenAIGatewayHandler.Messages -> EffectiveGatewayRouteResolver -> layered scheduler -> DB fresh recheck -> WaitPlan`; direct `TestLayered_GroupedAccountPassesDBFreshRecheck`, `TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount`, `TestLayered_FallbackWaitPlanRechecksPrivacyRequirementAgainstDB`, and `TestGatewayHandlerMessagesUsesEffectiveRouteSnapshot` evidence. |
| 2 | Grok/platform/session/previous-response sticky, privacy, image capability | `protected` | OpenAI selection reaches the layered scheduler's session/previous-response sticky checks before privacy/image capability recheck; direct `TestLayered_SessionStickyPreservesGrokBinding`, `TestLayered_SessionStickyRecheckHonorsImageCapability`, `TestLayered_PreviousResponseStickyEnabled`, and `TestLayered_PreviousResponseStickyHonorsRequirePrivacySet`. |
| 3 | OpenAI HTTP/WS/Live, turn ownership, final outbound model, failed usage, prompt-cache reuse, passthrough fields | `protected` | HTTP/WS forwarding reaches response-bound relay turn completion and `RecordUsage`; direct evidence includes `TestOpenAIGatewayService_Forward_WSv2_ResponseDoneUsageParsed`, `TestRelay_OnTurnComplete_UsesCurrentResponseCreateModel`, `TestRelay_OnTurnComplete_PerTerminalEvent`, `TestGatewayServiceRecordUsage_AttachesFinalUpstreamRequestSnapshot`, `TestDeriveCompatPromptCacheKey_StableAcrossLaterTurns`, `TestGatewayService_ForwardAsResponses_PassthroughHeaderForwardCopiesFromClientRequest`, and focused WS ownership/Live lifecycle gates. |
| 4 | prompt/security audit | `protected` | Gateway POST routes classify into audit before selection; WS first and subsequent turns pass the same gate: `TestEveryGatewayPOSTRouteIsClassifiedForPromptAuditCoverage` and `TestResponsesWebSocketHasFirstAndSubsequentTurnPromptGates`. Current Qwen3Guard strict/auxiliary-field focused gate is also green. |
| 5 | Images exact audit and text lifecycle | `protected` | `Images -> checkSecurityAuditLazy -> runSecurityAuditLazy -> Coordinator.CheckLazy`; the coordinator shares the frozen payload with legacy moderation and release follows audit. All eight direct Images lifecycle tests are recorded below. |
| 6 | request-body replay/spooling/cleanup | `protected` | Images body ingress spools/reopens the mapped effective body, then cleans it up after send; `TestOpenAIImages_InlineSpoolKeepsRawBodyAndOmitsSnapshots` and `TestOpenAIGatewayHandlerImages_MultipartReplayUsesMappedEffectiveBody` passed under the protected local gate. |
| 7 | async images, object storage, image input/output billing and upstream multiplier | `protected` | Prompt guard precedes async task creation; task completion can offload to object storage; usage then applies resolved image multiplier. Direct evidence: `TestAsyncImagePromptGuardRunsBeforeTaskCreation`, `TestAsyncImageSuccessfulPrecheckIsNotRepeatedByDetachedExecution`, `TestImageTaskServiceCompleteOffloadsToStorage`, `TestGatewayServiceRecordUsage_EmptyImageSizeDefaultsBeforeBillingAndPersistence`, `TestGatewayServiceRecordUsage_PeakRateAffectsTokenModeImageOutputTokens`, and `TestOpenAIFreshUpstreamBillingRateRecomputesPeakAtSelectionTime`. |
| 8 | settings hot/partial update | `protected` | `SettingsView.saveSettings -> SettingHandler.UpdateSettings -> omittedSettingKeys -> setting runtime reload`; direct tagged `TestUpdateSettingsPartialPayloadKeepsUnsentKeys` and `TestUpdateSettingsFullPayloadStillClearsSentEmptyFields` passed. |
| 9 | repository scoped updates, user/API key, Passkey/session step-up | `manual` | Scoped user/API-key update and Passkey setting coverage are protected, but `PasskeyService.FinishLogin -> session consume -> active/backend-mode checks -> audit -> token issuance` has no direct joined-login top-level test. This row remains manual specifically for that joined-login boundary. |
| 10 | subscription quota reset, receipt/outbox/migration integration | `unverified` | Unit/static quota evidence and frontend quota tests are protected, but PostgreSQL transaction/lock, receipt rollback, outbox rollback, migration-runner and upgrade integration require unavailable Docker/Testcontainers. The complete residual union is listed below. |
| 11 | user resources, group copy, batch limits | `protected` | User/group handlers and repositories retain field masks, batch validation, group-copy idempotency/recovery and default group binding. The 18 unit-tag user/group/batch tests plus `TestUserCanBindGroupRejectsBlockedPublicGroup` passed. |
| 12 | frontend local capability | `manual` | Settings, usage, quota, and available-channel Vitest targets passed. `ModelPlazaView`'s public `/model-plaza` route and `embedded=1` authenticated-layout predicate were source-reviewed only, with no direct route/auth test; this row remains manual specifically for that boundary. |
| 13 | pricing, count_tokens, release fallback, deploy security | `protected` | `GatewayService.ForwardCountTokens` retains mapping/retry/accounting; image token and generation pricing apply independent/effective rate multipliers. Pricing/count-token/release tests passed; runtime-resource and repaired no-stderr compose security scripts passed without a deployment. |
| 14 | Ent/Wire, dependencies, migrations | `protected` | Final full gate ran two clean `make -C backend generate` passes with no `backend/ent` or `backend/cmd/server/wire_gen.go` diff. Receipt/outbox blobs remain `c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6` and `502ecec1caf9f76e022c2e83acf3707190539301`; `191_passkey_credentials.sql` is the sole added migration. |

Final counts: `protected=11, manual=2, gap=0, unverified=1`.

## Images Contracts

The direct Task 16 handler command selected and passed all eight tests below. These are direct lifecycle assertions, not a package-level PASS substitute.

| Contract | Direct test name(s) |
| --- | --- |
| Unified audit entry; legacy moderation runs once | `TestOpenAIImages_UnifiedAuditRunsLegacyOnce` |
| Moderation and security consumers see the frozen payload before release | `TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease`; `TestOpenAIImages_SecurityAuditUsesFrozenPayloadBeforeRelease` |
| Disabled security audit makes zero freeze and provider calls | `TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload` |
| Runtime scope is evaluated before legacy moderation freezes the payload | `TestOpenAIImages_LegacyModerationDefersPayloadUntilRuntimeScope` |
| Security audit freezes at most once | `TestOpenAIImages_SecurityAuditFreezesPayloadAtMostOnce` |
| Multipart and OAuth text remains available through audit/moderation and is released before a blocked upstream return | `TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream`; `TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream` |

`TestOpenAIImages_DisabledSecurityAuditDoesNotFreezePayload` directly proves zero frozen payload and zero provider calls for the disabled boundary; it does not by itself use a 20 MiB payload. `TestOpenAIImages_MultipartTextIsReleasedBeforeBlockedUpstream` and `TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream` each supply the 20 MiB payload lifecycle evidence: text remains available through audit/moderation and is released before the blocked upstream return. `OpenAIImagesRequest.ReleaseText` clears prompt/image text only after the lazy audit/moderation consumers complete.

## GHSA-vrxq-qm4h-6hgg Review

All three Responses route families are guarded: `/v1/responses`, `/responses`, and `/backend-api/codex/responses`. `guardResponsesSubpath -> IsForwardableOpenAIResponsesRequestPath -> sanitizedUpstreamPathSuffix` rejects malformed client suffixes before the registered handler. `openAIResponsesRequestPathSuffix` and `appendOpenAIResponsesRequestPathSuffix` independently revalidate before normal or passthrough URL construction; replay retains body bytes only and does not construct a client-controlled path. Prompt audit is before scheduling, and the scheduler has no Responses suffix assembly.

- Raw traversal `../models`, `../../models`, and `...`; encoded traversal `%2e%2e/models`, `%2E%2E%2Fmodels`, and `compact%2f..%2fmodels`; and backslash `compact\detail` and `%5c..%5cmodels` are rejected.
- Direct-helper `compact?next=%2fmodels` and `compact#fragment` are rejected, while real HTTP query boundaries on all three compact routes remain legal and do not enter the suffix.
- Empty segments `//double` and `/compact//detail`; a 129-byte segment; nine nonempty segments; NUL/control/non-ASCII/space; and punctuation `:`, `;`, `@`, `%` are rejected.
- `TestGatewayRoutesResponsesSubpathRejectsNonConformingSubpaths` proves malformed inputs receive HTTP `404` with `Unsupported responses subpath`; its direct guard path uses a counting downstream handler and proves downstream calls are zero. `TestOpenAIResponsesRequestPathSuffixRejectsNonConformingSubpaths` proves rejection is neither compact classification nor an appended upstream suffix.
- Legal root, trailing slash, `compact`, response-ID, `gemini-2.5-pro`, and all three AI Studio actions (`generateContent`, `streamGenerateContent`, `countTokens`) retain their established behavior.
- Native Gemini and chat/messages compatibility paths call `buildGeminiAIStudioModelActionURL` before URL/request construction. `TestGeminiAIStudioInvalidModelsDoNotSendRequests` exercises native URL-model and compat body-model failures and records HTTP stub count zero.
- `TestOpenAIProxyStreamCircuitThresholdTTLAndSuccessReset` and `TestOpenAIProxyStreamCircuitDisabled` cover the proxy circuit. Normal selection honors quarantine; only a no-capacity OpenAI second pass is quarantine-blind, preserving the required fail-open boundary. `TestParseQwen3GuardStrictAndPolicy` and `TestParseQwen3GuardIgnoresAuxiliaryResponseFields` confirm auxiliary fields do not change Qwen3Guard policy.

The focused service, route, and security-audit commands for this matrix all exited `0`; the test-only path-guard commit was `140f9fb7b`, later testability/evidence closure was `46bda30bb`, and no production GHSA compatibility commit was required.

## Docker/Testcontainers Residual

Task 5, Task 8, Task 11, Task 14, and Task 17 each recorded `docker_command=unavailable`; no integration command was invoked after any failed preflight. The exact union of still-unverified Docker/Testcontainers contracts is:

| Contract / test target | Why it remains unverified |
| --- | --- |
| `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` | Final Task 17 Docker preflight was unavailable. |
| `TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade` | Final Task 17 Docker preflight was unavailable. |
| `TestAdvanceQuotaCycle_ConcurrentRequestsDeductOnce` | Docker was unavailable in Tasks 5, 8, and final Task 17. |
| `TestAdvanceQuotaCycleReceipt_RollsBackSubscriptionWhenReceiptWriteFails` | Docker was unavailable in all applicable stage/final preflights. |
| `TestSubscriptionCacheInvalidationOutbox_TriggersSemanticChangesAndRollsBack` | Docker was unavailable in all applicable stage/final preflights. |
| `TestSubscriptionCacheInvalidationMigration_RawRerunIsIdempotent` | Docker was unavailable in Task 5 and Task 11; no later Testcontainers run superseded it. |
| `TestUserRepoSuite` | Docker was unavailable in Task 11; no later Testcontainers run superseded it. |
| `TestAPIKeyRepoSuite` | Docker was unavailable in Task 10; no later Testcontainers run superseded it. |
| New and upgrade PostgreSQL migration paths | The Testcontainers migration lifecycle could not start without Docker. |

Task 10 additionally leaves the User/API-key PostgreSQL lost-update category unverified, including the independent `TestAPIKeyRepoSuite` and the retained `TestUserRepoSuite`; compile-only or field-mask unit evidence does not cover their transactional concurrency behavior. The final Task 17 five-target set is exactly the two migration-runner targets, concurrent quota deduction, receipt rollback, and outbox semantic-change/rollback target shown above. It and the new/upgrade PostgreSQL paths remain `unverified`. Filtered-FS upgrade tests, static inspection, compilation, historical output, and blob identity are complementary evidence only; none is reported here as an integration PASS.

## OpenSpec Strict Validation

Initial required invocation from repository root:

```powershell
openspec validate staged-merge-upstream-v0-1-169 --strict
```

Exact output and exit:

```text
Unknown item 'staged-merge-upstream-v0-1-169'.
exit_code=1
```

Diagnosis: OpenSpec CLI `1.7.0` resolves a root only when an ancestor contains an `openspec/` directory. This repository stores its OpenSpec root at `docs/openspec`, so repository-root invocation treats the repository as an implicit root and cannot see the change. The document itself is not invalid.

The same strict command was rerun from `docs`, the parent of the actual OpenSpec root. Exact output and exit:

```text
Change 'staged-merge-upstream-v0-1-169' is valid
exit_code=0
```

No OpenSpec document repair was needed. Repair commit: none. `docs/openspec/changes/staged-merge-upstream-v0-1-169/tasks.md` was not changed, staged, or checked off.

## Operational Boundary And Final Status

未推送、未发版、未部署、未操作服务器、未操作 Nginx，生产临时盾仍保留。

No tag, release, image build/push/run, database runtime operation, or remote operation occurred. No fetch, deploy, server, Redis, or Nginx operation occurred. No runtime selection was read or modified. This report is the sole Task 18 tracked deliverable; after its report-only commit, the required final status is exactly `?? .comet/current-change.json`.
