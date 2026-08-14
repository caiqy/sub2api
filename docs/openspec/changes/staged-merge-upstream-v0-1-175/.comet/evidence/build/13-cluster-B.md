# Task 11 Stage 2 Cluster B Evidence

## Scope

- Branch: `feature/20260813/staged-merge-upstream-v0-1-175`.
- Base merge retained: `d9213769232a081c98354a048857d2e33a491fa1`.
- `backend/cmd/server/VERSION` was not modified.

## RED To GREEN

### Duplicate group pricing configuration

- RED command: `go -C backend test -tags=unit -count=1 -run '^TestDuplicateGroupCopiesConfigurationDeeplyAndResetsRuntimeState$' ./internal/service`.
- The first execution was blocked before the test body by `ent/runtime/runtime.go:1195`: `panic: interface conversion: interface {} is domain.OpenAIMessagesDispatchModelConfig, not bool`. `f3d949107` added the two group pricing fields but left later generated Ent descriptor indexes stale. `go -C backend generate ./ent` corrected the generated indexes and associated generated metadata only.
- RED after the generated Ent fix: `admin_group_duplicate_test.go:214: Error: Should be true`. The duplicate lost `LongContextPricingEnabled`; it also had no `ModelPricing` copy.
- Fix: `cloneGroupForDuplicate` now preserves the long-context switch and deep-clones per-model pricing.
- GREEN: the same service command passed.

### Create API long-context default

- RED command: `go -C backend test -tags=unit -count=1 -run '^TestGroupHandlerDefaultsLongContextPricingEnabled$' ./internal/handler/admin`.
- Observed failure: `group_handler_pricing_test.go:116: Error: Should be true`.
- Fix: `CreateGroupRequest.LongContextPricingEnabled` is a pointer. `GroupHandler.Create` defaults an omitted value to `true` while an explicit `false` remains distinguishable and is forwarded unchanged.
- GREEN: the same handler command passed.

## Upstream Review

- `814ecfba7`: `UpdateGroup` records the previous platform and invalidates the channel cache only when the platform changes. Group routes remain under the parent `/admin` router with admin authentication, audit logging, and compliance middleware. Existing platform-cache tests passed; no cross-group cache or permission-boundary gap found.
- `bd404c16f`: pricing conflict detection uses `normalizeChannelPricingModelName`, matching the pricing cache key. Mapping conflict detection remains on the distinct lower-case-only mapping key, so local mapping semantics are not narrowed. Existing conflict tests passed; no new gap found.
- Grok JWT tier and local pricing semantics were not changed. The duplicate path retains configured local per-model prices and the long-context pricing switch.

## Verification

- `go -C backend test -tags=unit -count=1 -run '^(TestDuplicateGroupCopiesConfigurationDeeplyAndResetsRuntimeState|TestGroupHandlerDefaultsLongContextPricingEnabled|TestUpdateGroupInvalidatesChannelCacheOnPlatformChange|TestUpdateGroupWithoutChannelCacheInvalidator|TestValidateNoConflictingModels|TestValidateNoConflictingMappings)$' ./internal/service/ ./internal/handler/admin/`: PASS.
- `go -C backend build ./...`: PASS.
- `git diff --check`: PASS.

## Expected Paths

- `backend/ent/group.go`
- `backend/ent/mutation.go`
- `backend/ent/runtime/runtime.go`
- `backend/internal/handler/admin/group_handler.go`
- `backend/internal/handler/admin/group_handler_pricing_test.go`
- `backend/internal/service/admin_group_duplicate.go`
- `backend/internal/service/admin_group_duplicate_test.go`
- `docs/openspec/changes/staged-merge-upstream-v0-1-175/.comet/evidence/build/13-cluster-B.md`

## Residual Risk

- The focused unit tests and full Go build do not exercise PostgreSQL/Redis-backed cache persistence. The reviewed invalidation remains a process-local cache clear and is covered by its unit spy; distributed cache behavior is unchanged.

## Round 1 Follow-up

### RED To GREEN

- Review found that `ChannelModelPricing.Clone` copies slices but intentionally shares price and interval pointers. The group-only `cloneGroupModelPricing` must not reuse those pointer values.
- RED command: `go -C backend test -tags=unit -count=1 -run '^TestDuplicateGroupCopiesConfigurationDeeplyAndResetsRuntimeState$' ./internal/service`.
- Observed failure: `admin_group_duplicate_test.go:281: Expected and actual point to the same object` for `InputPrice`.
- Fix: the group duplicate helper now copies all `ChannelModelPricing` price pointers and every interval `MaxTokens` and price pointer. The global `ChannelModelPricing.Clone` contract remains unchanged.
- GREEN: the duplicate test passed. The handler now also covers JSON `long_context_pricing_enabled:false`, confirming explicit false remains distinct from an omitted field.

### Ent Generate Determinism

- Round 1: `go -C backend generate ./...` exited 0 and printed two Wire writes for `backend/cmd/server/wire_gen.go`.
- Round 2: the same command exited 0 with the same two Wire writes. A before/after `git status --porcelain` comparison produced no difference, so the second generation made no diff.

### Round 1 Verification

- `go -C backend test -tags=unit -count=1 -run '^(TestDuplicateGroupCopiesConfigurationDeeplyAndResetsRuntimeState|TestGroupHandlerDefaultsLongContextPricingEnabled|TestGroupHandlerPreservesExplicitFalseLongContextPricingEnabled|TestUpdateGroupInvalidatesChannelCacheOnPlatformChange|TestUpdateGroupWithoutChannelCacheInvalidator|TestValidateNoConflictingModels|TestValidateNoConflictingMappings)$' ./internal/service/ ./internal/handler/admin/`: PASS.
- `go -C backend build ./...`: PASS.
- `git diff --check`: PASS.

### Round 1 Expected Paths

- `backend/internal/handler/admin/group_handler_pricing_test.go`
- `backend/internal/service/admin_group_duplicate.go`
- `backend/internal/service/admin_group_duplicate_test.go`
- `docs/openspec/changes/staged-merge-upstream-v0-1-175/.comet/evidence/build/13-cluster-B.md`
