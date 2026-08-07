# v0.1.171 Final Verification Report

## Outcome

`DONE_WITH_CONCERNS`

All non-Docker final gates passed. The sole integration residual is Docker-only migration execution; `go test -race` remains unavailable because the local environment lacks cgo. No product, test, plan, OpenSpec task, checkpoint, VERSION, runtime selector, remote, server, release, or deployment operation was changed or performed by this verifier.

## Provenance

| Binding | Value |
| --- | --- |
| Immutable source base | `16c07d8064b0b4604e9f47ef782e7d29534402d3` (`VERSION=0.1.169.3`) |
| Execution base | `fd109296b5f41398350070dd8df826846d9adb1b` (`VERSION=0.1.169.3`) |
| Tested source HEAD | `73df7248383b9f534df64956efe3c0d321f0e3bc` (`chore: bump version to 0.1.171.1`) |
| Reporting HEAD | `436ebf66676aabee02e44a974e76cbb671b4e163` (`docs: advance to final verification report`) |
| Final VERSION | `0.1.171.1` |
| Worktree before report staging | Only `?? .comet/current-change.json`; staged index empty; no unmerged paths or index entries. |

`16c07d806...` is an ancestor of both `fd109296...` and reporting HEAD. The Task 16 source gate is bound to `73df724...`. Its only descendants before reporting HEAD are documentation checkoffs and Comet checkpoint commits: plan/OpenSpec task checkoffs at `440ba3f`, `5149a93`, and `75c234c`; `.comet/subagent-progress.md` checkpoints at `7175134`, `b182aab`, and `436ebf6`. `git diff --name-only 73df724..436ebf6` lists only those plan, task, and checkpoint paths. No long gate is claimed as rerun on a later documentation-only HEAD.

## Tags And Topology

| Item | Object / parent list | Result |
| --- | --- | --- |
| `v0.1.170` | Tag object `60286d35e4b6dc6851ab69f890c2d1b7b7a3bcb8`; peeled `c043c24774228ba891ddf90d783aa6dc7d0855b5` | Annotated tag; ancestor of reporting HEAD. |
| `v0.1.171` | Tag object `afd154b92aac36c6dafb1fa8e181ca827c78c465`; peeled `f0e7a9c7a23a7d02fb159b62fa809621eb0475a6` | Annotated tag; ancestor of reporting HEAD. |
| v0.1.170 merge | `98c7b04874361a1cf95b8dea90ed1c4db2f05d4d 30528a82e32bfedc011d741e870964beb5743aa4 c043c24774228ba891ddf90d783aa6dc7d0855b5` | Exact second parent. |
| v0.1.171 merge | `cca37e01eb719d65ce81dc7569b190fe9550ae5d 5f505520ded16114e3f2850f7b856a0650a82755 f0e7a9c7a23a7d02fb159b62fa809621eb0475a6` | Exact second parent; follows v0.1.170 on first parent. |

## Task 16 Focused Gates

These are the complete inherited final-source command outcomes from the Task 16 temporary report. All Go commands ran from `backend`; Vitest commands ran from repository root.

| ID | Command | Exit | Result |
| --- | --- | ---: | --- |
| F1 | `go test -count=1 ./internal/service -run '^(Test.*Profit.*|Test.*Pricing.*|Test.*Layered.*|Test.*Sticky.*|Test.*WaitPlan.*|TestGatewayServiceRecordUsage.*|TestGatewayBillingEligibility.*)$'` | 0 | PASS |
| F2 | `go test -tags=unit -count=1 ./internal/service -run '^(Test.*Profit.*|Test.*Pricing.*|Test.*Layered.*|Test.*Sticky.*|Test.*WaitPlan.*|TestGatewayServiceRecordUsage.*|TestGatewayBillingEligibility.*)$'` | 0 | PASS |
| F3 | `go test -count=1 ./internal/handler -run '^(Test.*Profit.*|TestOpenAI.*Pricing.*|TestGatewayHandler.*Sticky.*|Test.*Sticky.*|Test.*WaitPlan.*|Test.*Usage.*)$'` | 0 | PASS |
| F4 | `go test -tags=unit -count=1 ./internal/handler -run '^(Test.*Profit.*|TestOpenAI.*Pricing.*|TestGatewayHandler.*Sticky.*|Test.*WaitPlan.*|Test.*Usage.*)$'` | 0 | PASS |
| F5 | `go test -count=1 ./internal/service -run '^(TestGateway.*PartialUsage.*|TestOpenAI.*(WS|WebSocket).*|Test.*Anthropic.*|Test.*ContentModeration.*|Test.*Subscription.*|Test.*Setting.*|Test.*RequestBody.*)$'` | 0 | PASS |
| F6 | `go test -tags=unit -count=1 ./internal/service -run '^(TestDelayedFirstUseAnchorsMonthlyWindowAtStartsAt|TestAdminResetQuota_.*)$'` | 0 | PASS |
| F7 | `go test -count=1 ./internal/handler -run '^(TestOpenAI.*(WS|WebSocket|Images|Responses).*|TestGatewayHandler.*(Usage|Body|Settings).*)$'` | 0 | PASS |
| F8 | `go test -tags=unit -count=1 ./internal/handler -run '^TestGatewayHandler_Messages(ForwardErrorStillCreatesUsageLog|FailoverExhaustedStillCreatesUsageLog|SelectionExhaustedAfterFailoverStillCreatesUsageLog|StreamingPartialWriteFailureStillCreatesUsageLog)$'` | 0 | PASS, 4 named cases |
| F9 | `go test -tags=unit -count=1 ./internal/handler -run '^TestGatewayHandler_GeminiV1BetaModels_(ForwardErrorStillCreatesUsageLog|FailoverExhaustedStillCreatesUsageLog|SelectionExhaustedAfterFailoverStillCreatesUsageLog)$'` | 0 | PASS, 3 named cases |
| F10 | `pnpm --dir frontend exec vitest run src/components/account/__tests__/CreateAccountModal.spec.ts src/components/account/__tests__/EditAccountModal.spec.ts src/components/account/__tests__/BulkEditAccountModal.spec.ts src/components/account/__tests__/UpstreamBillingRateCell.spec.ts src/components/payment/__tests__/PaymentMethodSelector.spec.ts src/features/prompt-audit/__tests__/viewModel.spec.ts src/features/prompt-audit/__tests__/PromptAuditView.spec.ts src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts src/views/admin/__tests__/SettingsView.spec.ts` | 0 | PASS, 174 tests |
| F11 | Ordered five-file `git cat-file` / expected and HEAD `git rev-parse` migration identity loop | 0 | PASS, 5 exact blob matches |
| F12 | `go test -tags=integration -count=1 -v ./internal/repository -run '^TestMigrationsRunner_PreservesPasskeyAndSubscriptionQuotaMigrationsAcrossUpgrade$'` | 0 | `docker is not available; skipping integration tests`; Docker-only `unverified` |
| F13 | `go test -tags=unit -count=1 -v ./internal/service -run 'Test(AdminResetQuota|AutomaticWindow|CheckAndResetWindowsResetsPartialFinalMonthlySubscriptions|AutomaticWindowsAllowPartialFinalDailyAndWeeklyPeriods)'` | 0 | PASS, 21 top-level tests and 4 subtests |
| F14 | `go test -tags=unit -count=1 ./internal/service` | 0 | PASS |
| F15 | `go test -count=1 ./internal/pkg/openai -run '^Test.*Codex.*'` | 0 | PASS |
| F16 | `go test -count=1 ./internal/service -run '^(Test.*Codex.*|Test.*Capacity.*|Test.*AlphaSearch.*|TestOpenAI.*(Forward|Passthrough|WS).*)$'` | 0 | PASS |
| F17 | `go test -count=1 ./internal/handler -run '^(Test.*Gateway.*(Body|Failover|Usage).*|TestOpenAI.*WebSocket.*)$'` | 0 | PASS |
| F18 | `go test -count=1 ./internal/service -run '^(Test.*(Tencent|Aliyun|Turnstile|Captcha|Auth|Refund|Renewal|Reasoning|WebSocket|Prompt|Usage).*)$'` | 0 | PASS |
| F19 | `go test -count=1 ./internal/handler -run '^(Test.*(Captcha|Auth|Passkey|Setting|Prompt|Usage).*)$'` | 0 | PASS |
| F20 | `go test -count=1 ./internal/server/middleware -run '^(Test.*(Auth|SecurityHeaders|CSP).*)$'` | 0 | PASS |
| F21 | `pnpm --dir frontend exec vitest run src/components/__tests__/AliyunCaptchaWidget.spec.ts src/components/__tests__/TencentCaptchaGate.spec.ts src/components/auth/__tests__/PendingOAuthCreateAccountForm.spec.ts src/views/auth/__tests__/TencentCaptchaActionGate.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/groupsReasoningEffort.spec.ts` | 0 | PASS, 73 tests |
| F22 | `go test -tags=unit -count=1 ./internal/service -run '^(Test(VerifyCaptcha.*|VerifyActionCaptcha.*|AuthServiceVerifyActionCaptcha.*|SettingService_.*Captcha.*|PrepDeductBalanceRequiresForceWhenBalanceIsInsufficient|WriteFailedUsageLogBestEffort_CreatesZeroCostUsageLog|AssignOrExtendSubscription.*|OpenAIWSIngressLease.*|OpenAIGatewayService_ProxyResponsesWebSocketFromClient_(LeaseLossSendsRetryClose|IdleTimeoutReleasesStoreDisabledSession)))$'` | 0 | PASS |
| F23 | `go test -tags=unit -count=1 ./internal/handler -run '^(Test(PasskeyBeginLogin.*|OpenAIReasoningEffortPolicyForCompositeTarget|GatewayHandlerPreCancelledCompatibleRequestsDoNotSelectAccount|OpenAIGatewayHandler_.*FailedUsage.*|OpenAIResponsesWebSocket.*FailedUsage.*))$'` | 0 | PASS |
| F24 | `go test -tags=unit -count=1 ./internal/payment/provider -run '^TestStripeRefundUsesStableAmountSpecificIdempotencyKey$'` | 0 | PASS |
| F25 | `go test -count=1 ./internal/securityaudit -run '^(Test(PromptSnapshot.*|ResponsesWebSocketOnlyAuditsResponseCreateAndPreservesStage))$'` | 0 | PASS |
| F26 | `go test -count=1 ./internal/server/routes -run '^(TestEveryGatewayPOSTRouteIsClassifiedForPromptAuditCoverage|TestResponsesWebSocketHasFirstAndSubsequentTurnPromptGates)$'` | 0 | PASS, 2 named tests |
| F27 | `go test -count=1 ./internal/server/middleware` | 0 | PASS |
| F28 | `pnpm --dir frontend exec vitest run src/views/admin/__tests__/SettingsView.spec.ts` | 0 | PASS, 36 tests |

## Full, Build, Generate, And Static Gates

| Command | Exit | Result |
| --- | ---: | --- |
| `make test` | 0 | Backend default/unit/lint and frontend ESLint/typecheck/Vitest PASS; Vitest 236 files / 1806 tests. |
| `make VERSION=0.1.171.1 SHELL=D:/scoop/shims/bash.exe build` | 0 | Backend build, `vue-tsc -b`, Vite production build; 1051 modules transformed. |
| `make -C backend generate` (round 1) | 0 | Ent/Wire completed. |
| `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` (round 1) | 0 | No generated diff. |
| `make -C backend generate` (round 2) | 0 | Ent/Wire completed. |
| `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go` (round 2) | 0 | No generated diff. |
| `git diff --check` | 0 | No worktree whitespace errors. |
| `git diff --cached --check` | 0 | No staged whitespace errors. |
| `git diff --cached --name-only` | 0 | Empty before report staging. |
| `git diff --name-only --diff-filter=U` | 0 | No unmerged worktree paths. |
| `git ls-files -u` | 0 | No unmerged index entries. |
| `git grep -n -I -e '^<<<<<<< ' -e '^=======$' -e '^>>>>>>> ' -- .` | 1 expected | No tracked conflict markers. |

Existing non-blocking warnings were not failures: Browserslist data, Vue `router-link` and jsdom stderr, expected mocked-error/i18n stderr, and Vite dynamic-import / 699 kB chunk advisories.

## Migration Identity And Docker Residual

| Filename | Authority | Blob OID | Current status |
| --- | --- | --- | --- |
| `191_passkey_credentials.sql` | `16c07d8064b0b4604e9f47ef782e7d29534402d3` | `522b16b5bba12aedb9c4198d2d4ef082c8ea718f` | exact HEAD match |
| `191_subscription_quota_advance_receipts.sql` | `16c07d8064b0b4604e9f47ef782e7d29534402d3` | `c22d47d79cbbaf4bc40524d42ef52e6cc8ac3af6` | exact HEAD match |
| `192_subscription_cache_invalidation_outbox.sql` | `16c07d8064b0b4604e9f47ef782e7d29534402d3` | `502ecec1caf9f76e022c2e83acf3707190539301` | exact HEAD match |
| `192_group_profit_control.sql` | `c043c24774228ba891ddf90d783aa6dc7d0855b5` | `072b3c5db17accfd5197ea72f9a49fd6bdf446b4` | exact HEAD match |
| `193_group_profit_control_auth_cache_invalidation.sql` | `c043c24774228ba891ddf90d783aa6dc7d0855b5` | `f32f6e6f8b6d026b2e8620c90954336e30550c41` | exact HEAD match |

The canonical `Invoke-MigrationUpgradeIntegration -Stage 'final'` helper was inherited from Task 17 with the actual status `unverified`. The local Docker CLI is absent, so no native Docker process, Testcontainers target test, remote endpoint, or server operation occurred. Empty database execution, local 191/192 upgrade, ordering, idempotency, relations, and checksum remain unverified. This is the sole `unverified` matrix row and is not reported as PASS.

## Capability Review

CodeGraph was used before supplemental read-only source confirmation. It confirmed these current paths: layered scheduling is `OpenAIGatewayHandler -> OpenAIGatewayService -> getOpenAIAccountSchedulerWithContext -> layeredOpenAIAccountScheduler.Select`; `selectBySessionHash` checks schedulable/platform/privacy/request/transport compatibility, does DB recheck, then acquires a slot or returns `WaitPlan`. OpenAI Images calls `checkSecurityAuditLazy` before `ReleaseText`. `AdminResetQuota` uses `s.now()` for manual reset anchoring.

| # | Capability | Call-path evidence | Test evidence | Status |
| ---: | --- | --- | --- | --- |
| 1 | Advanced/layered scheduler, pool admission, DB recheck, sticky, WaitPlan/fallback | Gateway handler -> scheduler service -> layered `Select` -> sticky DB recheck -> slot/`WaitPlan`; pool retry remains in account selection. | F1-F4 | `protected` |
| 2 | Grok/platform/session sticky, privacy/image capability, prompt-cache reuse | Responses/Images -> capability scheduler -> previous-response/session sticky -> platform/privacy/image/transport filters. | F1-F10 and full gate support this path, but not every cross-product. | `manual` |
| 3 | OpenAI HTTP/WS/turn ownership, final outbound account/model, usage/circuit, replay/release | HTTP/WS ingress -> admission -> forward/failover -> final account/model usage; turn and request-body cleanup are terminal-path owned. | F5-F10, F15-F17 | `manual` |
| 4 | Alpha-search, Responses/PAT/composite reasoning and routing | AlphaSearch -> ForwardAlphaSearch -> matched body handle -> fallback/retry/cleanup; composite resolver/reasoning precedes forward. | F15-F18 | `manual` |
| 5 | Unified security audit, latest-input/proxy/Images duplicate moderation and snapshot stage | Gateway/Images -> audit coordinator -> snapshot/legacy moderation; Images lazy audit precedes text release. | F5, F25, F26 | `protected` |
| 6 | Runtime settings, auth cache/session/step-up, captcha providers/CSP and fail-closed auth | Setting update -> runtime refresh; auth cache -> session binding/step-up; captcha provider selection -> auth gate/CSP. | F18-F23, F27-F28 | `protected` |
| 7 | Exact subscription windows, renewal lock, quota reset/receipt/outbox, refund and failed usage | Refund/subscription -> locked repository -> receipt/reset -> post-commit outbox; failed usage preserves final identity. | F13-F14, F18, F22, F24 | `protected` |
| 8 | User resource controls, group duplication, account shadow and admin bulk limits | Admin handlers -> AdminService -> repository transaction/resource paths. | Root full gate plus F10 | `protected` |
| 9 | Local frontend account/settings/payment/prompt-audit/captcha/reasoning behavior | Local views/components -> local admin/auth APIs -> backend contracts. | F10, F21, F28 and full gate | `protected` |
| 10 | Dependencies, generated Ent/Wire, VERSION and merge topology | Manifest/schema/provider -> source-driven outputs; VERSION/tag/merge checks bind release topology. | Full/build/two-generate/static/topology gates | `protected` |
| 11 | Migration filenames, blobs and integration | Migration runner -> full filenames/checksums -> PostgreSQL upgrade paths. | F11 passed; F12 target was skipped without Docker. | Docker-only `unverified` |

Matrix counts: `protected=7`, `manual=3`, Docker-only `unverified=1`, `gap=0`. `manual` rows have both call-path review and passing supporting gates, but are not promoted on call-path evidence alone.

## Strict Validation

```text
COMMAND: comet classic openspec -- validate staged-merge-upstream-v0-1-171 --strict
OUTPUT: Change 'staged-merge-upstream-v0-1-171' is valid
EXIT: 0
```

## Non-Operations

This change was not pushed, tagged, released, deployed, or used to operate any server. Docker integration was not rerun. `go test -race` was not run because cgo is unavailable.
