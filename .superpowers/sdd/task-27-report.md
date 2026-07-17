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
