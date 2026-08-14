# Task 11 Stage 2 Cluster A Evidence

## Scope

- Branch: `feature/20260813/staged-merge-upstream-v0-1-175`.
- Review HEAD: `710c70a1a18a753fa1609c6479692b5b0206d682`; v0.1.176 merge base retained: `d9213769232a081c98354a048857d2e33a491fa1`.
- `backend/cmd/server/VERSION`, plan/tasks/spec/design/.comet.yaml, existing commits, remotes, tags, billing core, and subscription schema were not modified.

## Required References

- Read the complete Task 11 Cluster A brief.
- Read `design.md` decisions 5/6. The current design has no `8.2` heading; this matches the prior Cluster D evidence.
- Read the `Grok JWT tier 与定价体系保留本地语义` Scenario in `specs/upstream-release-sync/spec.md`.

## Upstream Commit Review

- `bb9e74285`: numeric access-token JWT tiers map to stable values; refresh only retains a stored tier when the new access token has no tier. `applyGrokResolvedSubscriptionTier` and `isKnownGrokFreeAccount` keep the live access token ahead of stale billing, credentials, and snapshots.
- `0fa577a19`: merge commit has no independent combined diff. Its Grok behavior is covered by the included `bb9e74285`, `a04ce4901`, `363cc4994`, `69648476d`, `0ae151a23`, and `b61e4bcc4` changes.
- `a04ce4901`: registers `grok-4.6` and its alias, routes cache/image Chat-to-Responses bridging through the compatible 4.6 path, and installs the official fallback card. The default text model remains 4.5.
- `363cc4994`: resolves ambiguous `SuperGrokPro` only from fresh grok-4.5 Responses quota windows; JWT remains authoritative. Capacity jitter installs a bounded per-model block rather than excluding the whole account.
- `b830bc14d`: makes the group long-context switch default true and backfills existing rows. Resolver handling preserves official long-context ladders while the OpenAI account switch remains an additional AND gate.
- `8c4c3c09c`: unknown Grok text-family IDs fall back to the 4.5 card. Image, voice, web-search, speech, and other non-text families cannot enter that fallback.
- `69648476d`: the usage cell and badge prefer the current credential tier over lagging billing/quota data, while retaining paid Lite behavior and free-tier rendering.
- `0ae151a23`: gateway quota observations persist at `grok_usage_snapshot`; the account list reads that snapshot before the legacy quota key and includes canonical snapshots in its incremental refresh key.
- `f3d949107`: resolver priority is Group -> Channel -> LiteLLM -> fallback. A group without a matching override retains local channel pricing; explicit group pricing is intentionally higher priority. Unified cost calculation still applies one resolved price per request.
- `678eb22a4`: Realtime billing only occurs after observed audio and positive duration. Cluster D already covered the gateway path; this audit found no additional pricing-side interaction.
- `b61e4bcc4`: gofmt-only JWT constant alignment plus the `long_context_pricing_enabled` available-groups contract field; no identity or price behavior changed.

## Local Contract Cross-Check

- Identity: `RefreshAccountToken` persists a new access-token tier and preserves the old credential tier only for an opaque/missing claim. Runtime usage and free-account routing independently re-read the live access token.
- Pricing: channel entries remain platform-isolated and take effect when no higher-priority group price matches. Group long-context opt-out gates both built-in and configured token ladders; the persisted default/backfill is true.
- Fallback: exact 4.6 pricing wins before the unknown-text fallback, and only unknown `grok-*` numeric/build/composer text IDs inherit 4.5.
- Single billing and persistence: `recordUsageCore` reaches `applyUsageBilling`, whose `UsageBillingRepository.Apply` command is keyed by `request_id`; a non-applied duplicate does not re-run finalization. Subscription usage, receipt/outbox, and cache invalidation flow were not changed.
- Snapshot/UI: `updateGrokUsageSnapshot` writes the canonical snapshot and stamps the 4.5-only plan signal; the frontend serializes billing plus canonical usage state so a tier-only refresh replaces the row.

## RED To GREEN

- RED: `pnpm --dir frontend exec vitest run src/components/account/__tests__/AccountUsageCell.spec.ts src/components/common/__tests__/PlatformTypeBadge.grok.spec.ts src/utils/__tests__/accountUsageRefresh.spec.ts src/views/admin/__tests__/AccountsView.sparkShadow.spec.ts` initially had four `AccountsView.sparkShadow.spec.ts` failures: `Cannot read properties of undefined (reading 'matches')` at `AccountsView.vue:702`.
- Root cause: `7c62382d04`, already an ancestor of `d92137692`, introduced `window.matchMedia` in `AccountsView`; the related view spec did not define the jsdom API. Its untracked per-test upstream-billing-probe mock was also cleared by `vi.restoreAllMocks`, producing console errors after the first test.
- Fix: the existing view spec now stubs a complete `MediaQueryList` and resets the existing upstream-billing-probe mock before each test. No production behavior changed.
- GREEN: `pnpm --dir frontend exec vitest run src/views/admin/__tests__/AccountsView.sparkShadow.spec.ts` passed 10/10 without application console errors.

## Verification

- `go -C backend test -tags=unit -count=1 -run '^(TestMapJWTSubscriptionTierNumber|TestSubscriptionTierFromJWTUsesNumericClaim|TestCanonicalGrokPlanUsesOnlyGrok45ResponsesWindow|TestGrokQuotaFetcher.*|TestGrokOAuthServiceRefreshAccountToken.*|TestGetIntervalPricing.*|TestResolve.*LongContext.*|TestUpdateGrokUsageSnapshot.*|TestGatewayServiceRecordUsage.*)$' ./internal/pkg/xai/ ./internal/service/ ./internal/handler/`: PASS. The handler package had no matching tests.
- `pnpm --dir frontend exec vitest run src/components/account/__tests__/AccountUsageCell.spec.ts src/components/common/__tests__/PlatformTypeBadge.grok.spec.ts src/utils/__tests__/accountUsageRefresh.spec.ts src/views/admin/__tests__/AccountsView.sparkShadow.spec.ts`: PASS, 4 files / 51 tests.
- `go -C backend build ./...`: PASS.
- `git diff --check`: PASS.

## Expected Paths

- `frontend/src/views/admin/__tests__/AccountsView.sparkShadow.spec.ts`
- `docs/openspec/changes/staged-merge-upstream-v0-1-175/.comet/evidence/build/15-cluster-A.md`

## Residual Risk

- The focused tests do not call a live xAI upstream or PostgreSQL/Redis-backed usage persistence. Existing request-id idempotency, subscription receipt/outbox, and cache invalidation behavior is structurally unchanged.
- Vitest reports the repository's existing Browserslist `caniuse-lite` age notice. Dependency metadata is outside this cluster.
