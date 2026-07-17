# Task 27 Report: v0.1.157 Capability Regression Review

## Scope

- Review the v0.1.157 merge at `fa656646d` against first parent `d77bea9c9` and upstream parent `a2779cd5f`.
- Preserve local behavior and v0.1.157 capabilities without merging v0.1.158 or v0.1.159.
- Do not rewrite the merge commit, push, release, deploy, or commit `.comet/current-change.json`.

## Repairs

- `e3b0c15b1` restores six async image task routes with the existing API-key and group middleware chain.
- `6869df0c4` restores local behavior for Grok image gating, `/v1/sub2api/billing`, scheduler settings and recheck behavior, Anthropic OAuth mimic, and SettingsView scheduler controls.
- This follow-up restores the merge-dropped frontend type contract:
  - upstream billing probe DTOs and account snapshot fields;
  - image input token/cost fields on `UsageLog`;
  - d77 header override platform/template exports used by `EditAccountModal`, alongside v0.1.157 capability detection;
  - session binding and audit retention settings controls, defaults, and update payload fields.

## Typecheck Causality

- The initial `vue-tsc` failure was not pre-existing. The v0.1.157 side adds the probe, image usage, and session/audit contracts; the d77 side supplies the still-used EditAccountModal helper exports.
- The merge retained consumers while dropping those provider contracts. The direct failing `pnpm --dir frontend run typecheck` was the regression check; after the minimal union restoration it exits 0.

## Validation

- `go -C backend test ./internal/service -run 'Image|Billing|Scheduler|Audit|StepUp|Session|FirstOutput|WebSocket' -count=1`: exit 0.
- `go -C backend test ./internal/handler/... -run 'Image|Billing|Audit|StepUp|Account|Setting' -count=1`: exit 0.
- `go -C backend test ./internal/repository -run 'Image|Billing|Scheduler|Audit|Probe|ChannelMonitor' -count=1`: exit 0.
- `go -C backend test ./internal/server/... -run 'Audit|Session|StepUp|APIKey|ImageTask' -count=1`: exit 0. `internal/server` has no test files and `internal/server/routes` has no matching tests.
- `pnpm --dir frontend exec vitest run src/components/account/__tests__/UpstreamBillingRateCell.spec.ts src/views/admin/__tests__/SettingsView.spec.ts src/views/admin/__tests__/ChannelMonitorView.duplicate.spec.ts src/i18n/__tests__/localesMessageCompile.spec.ts`: 4 files / 33 tests pass.
- `pnpm --dir frontend run typecheck`: exit 0.

## Remaining Scope

- v0.1.158 and v0.1.159 remain unmerged.
- Existing Vitest `router-link`, jsdom XHR, and Browserslist warnings remain non-failing test-environment warnings and were not changed.

## Follow-Up Repair

- `15e5ff41b fix: preserve local behavior after v0.1.157` restores the admin settings DTO fields dropped by the merge: session binding, audit retention, OpenAI scheduler rate controls, and upstream-cost scheduler values.
- The same commit restores the OpenAI API-key upstream billing auto-probe control and its persisted `extra` value. An absent false default stays absent, so unrelated account saves do not create a new setting.
- It also restores the referenced English and Chinese account locale keys for upstream billing, Grok custom URL, header JSON import, and Agent Identity flows.

## Follow-Up RED/GREEN

- RED: `go test -tags=unit ./internal/server -run '^TestAPIContract' -count=1` reported zero-valued session/audit/scheduler fields because `SettingHandler.GetSettings` omitted DTO assignments. GREEN: the same command exits 0 after restoring the six-field mapping.
- RED: full frontend Vitest reported 36 missing account locale keys and the absent `[data-testid="upstream-billing-auto-probe"]` control. GREEN: `pnpm exec vitest run src/i18n/__tests__/localeKeysExist.spec.ts src/components/account/__tests__/EditAccountModal.spec.ts` exits 0 with 57 tests.

## Final Validation And Self-Review

- `make test` exits 0: backend tests and lint, frontend lint/typecheck, and 189 Vitest files / 1454 tests pass.
- `pnpm --dir frontend run typecheck` exits 0.
- No v0.1.158/v0.1.159 commits were merged; no push, release, or deploy was performed. `.comet/current-change.json` was removed before completion and excluded from commits.

## OAuth Scheduler Zero Follow-Up

- `5e5afd7f6 fix: preserve zero OAuth scheduler rate` changes the SettingsView payload serializer to retain finite values, including `0`, and to fall back to `1` only for `NaN` or infinite values.
- RED: `pnpm exec vitest run src/views/admin/__tests__/SettingsView.spec.ts -t "preserves a zero OAuth scheduling reference multiplier"` submitted `openai_oauth_scheduling_rate_multiplier: 1` after entering `0`.
- GREEN: the same command passes after the finite-value check. `pnpm exec vitest run src/views/admin/__tests__/SettingsView.spec.ts` passes 23/23 and `pnpm run typecheck` exits 0.
- This two-file follow-up did not rerun `make test`; the preceding Task 27 full gate remains `make test` exit 0 with 189 Vitest files / 1454 tests. The change is limited to one serializer expression and one regression test.
- Self-review: no unrelated feature changes, no v0.1.158/v0.1.159 merge, and `.comet/current-change.json` is excluded from commits.

## OAuth Scheduler Empty-Value Follow-Up

- `c2b6a9c05 fix: default empty OAuth scheduler rate` narrows the serializer to finite values that are already numbers. This keeps `0`, while the empty string produced by `v-model.number`, `NaN`, and infinite values fall back to `1`.
- RED: `pnpm exec vitest run src/views/admin/__tests__/SettingsView.spec.ts -t "defaults an empty OAuth scheduling reference multiplier"` submitted `openai_oauth_scheduling_rate_multiplier: 0` after clearing the input.
- GREEN: `pnpm exec vitest run src/views/admin/__tests__/SettingsView.spec.ts -t "OAuth scheduling reference multiplier"` passes both zero and empty-input cases. The complete `SettingsView.spec.ts` passes 24/24, and `pnpm run typecheck` exits 0.
- This two-file follow-up did not rerun `make test`; the preceding Task 27 full gate remains `make test` exit 0 with 189 Vitest files / 1454 tests. No v0.1.158/v0.1.159 commits were merged and no push, release, or deploy was performed.
- Self-review: `.comet/current-change.json` is currently untracked and intentionally retained under the latest user instruction; it remains excluded from every commit.
