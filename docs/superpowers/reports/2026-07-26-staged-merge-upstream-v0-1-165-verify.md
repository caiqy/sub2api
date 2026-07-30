# 分段合入上游 v0.1.165 最终自动验证报告

## 结论与边界

- 状态：`DONE_WITH_CONCERNS`。
- 分支：`feature/20260726/staged-merge-upstream-v0-1-165`。
- Task 27 起始 HEAD：`8c3b281f7f9e08a9f2d776f4241a922f7a85bff8`。
- 最终 source/test HEAD：`6c88f1891650e0ef18b0b5ae105b8f44a069a5a4`。
- 最终版本：`0.1.165.1`。
- 本轮发现并提交三个修复：`aff04f9cd` 仅格式化两个 WebSocket ownership 路径；`1cc41c72c` 避免 content moderation 未配置时构造完整 Images moderation payload，并使 OAuth retention fixture 的 started 信号可重入；`6c88f1891` 避免 content moderation 与 security audit 均未配置时仍构造完整 Images audit payload。
- 聚焦矩阵、最终原样 `make test`、版本化 Windows build、两轮 Ent/Wire generate、静态检查和远程 integration 在最终 source/test HEAD 上均有成功证据。此前三次 `make test` 的真实非零结果、首次 SSH preflight 本机解析失败及一次 remote ENOSPC 均保留，不改写为 PASS。
- Task 28 完整 ancestry/六个 merge 第二父验证、Task 29 浏览器烟测/OpenSpec 收口均未执行，本报告不声称这些任务完成。

## 起始状态

- `git branch --show-current`：exit `0`，输出 `feature/20260726/staged-merge-upstream-v0-1-165`。
- `git rev-parse HEAD`：exit `0`，输出 `8c3b281f7f9e08a9f2d776f4241a922f7a85bff8`。
- `git merge-base --is-ancestor 8c3b281f7 HEAD`：exit `0`。
- `git diff --cached --name-only`：exit `0`，无输出。
- `git diff -- opencode.json`：exit `0`，无输出。
- 起始 `git status --short --untracked-files=all` 仅为：`M .superpowers/sdd/task-4-report.md`、`?? .comet/current-change.json`、`?? paseo.json`。这些用户文件在本任务中未修改、未暂存。

## 非 integration 聚焦矩阵

下列命令均在 `backend/` 执行，除非另有说明。每条有效证据均实际匹配命名测试；唯一零测试命令单独列为排除项。

### Settings、proxy、storage 与 runtime

```text
go test -tags=unit ./internal/config -run '^(TestLoadForwardedClientIPHeadersNormalizesAndSnapshots|TestLoadExplicitTrustedProxiesEnablesConfiguredMode|TestLoadImageStorageFromEnv)$' -count=1 -v
```

exit `0`；3 个 top-level 测试 PASS。

```text
go test -tags=unit ./internal/server ./internal/server/middleware -run '^(TestConfigureTrustedProxies|TestAPIKeyAuthIPRestrictionUsesTrustedPathWhenSwitchDisabled|TestAPIKeyAuthIPRestrictionUsesConfiguredTrustedProxy|TestAPIKeyAuthIPRestrictionUsesForwardedClientIPInDenialWhenTrusted|TestSessionBindingContextFollowsForwardedIPSwitch|TestSessionBindingContextSnapshotsForwardedModeAndHeaders)$' -count=1 -v
```

exit `0`；6 个 top-level 测试及其子例 PASS。

```text
go test -tags=unit ./internal/service -run '^(TestSettingService_(UpdateSettings_APIKeyACLTrustForwardedIPRefreshesConfig|LoadForwardedClientIPSettingsMigration|LoadForwardedClientIPSettingsBackfillsConfigHeaders|LoadForwardedClientIPSettingsReadFailureFailsClosed)|TestResolveGrokCacheIdentityStableAcrossAppendOnlyTurns|TestGrokFreeMessagesClientToolCacheDefaultsOnForKnownFree|TestGrokFreeClientToolCacheRequestOptInOverridesAccountOptOut|TestImageStorageSettingsToggleTakesEffectWithoutRestart|TestImageStorageSettingsReuseBackupCredentials|TestImageTaskServiceCompleteOffloadsToStorage|TestForwardGrokResponsesCodexAdditionalToolsUsesMixedCacheIntent)$' -count=1 -v
go test -tags=unit ./internal/handler -run '^TestAsyncImageEnablesWithoutRestart$' -count=1 -v
go test ./internal/handler/admin -run '^TestBackupHandlerUpdateS3ConfigRefreshesReusedImageStorage$' -count=1 -v
```

三条均 exit `0`；service 11 个 top-level、handler 1 个、handler/admin 1 个测试 PASS。

### Reasoning、quota、Cleanup 与 billing

```text
go test -tags=unit ./internal/service -run '^(TestApplyOpenAIReasoningEffortPolicy|TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyRuntimeBlockedAccountFallsBackWithoutRebinding|TestLayeredOpenAIAccountSchedulerSessionStickyRuntimeBlockedAccountFallsBackWithoutRebinding|TestOpenAIGatewayService_SelectAccountWithScheduler_SessionStickyDBRuntimeRecheckSkipsStaleCachedAccount|TestLayered_WaitPlanFallbackSkipsUpstreamRestrictedAccount|TestHandleNonStreamingResponseAnthropicAPIKeyPassthrough_ForceCacheBillingResponse|TestApplyOutputTokenEstimation_ZeroOutputWithText|TestExtractOpenAIUsageFromJSONBytes_MergesHostedImageGenToolUsage|TestExtractOpenAIUsageFromJSONBytes_NonStreamingMergesImageGen|TestFinalizePostUsageBillingWaitsForNotificationDelivery|TestDeferredServiceStopWaitsForInFlightFlush)$' -count=1 -v
go test -tags=unit ./internal/repository -run '^(TestSchedulerCacheUpdateLastUsedUsesSideKeyWithoutRewritingPayloads|TestSchedulerCacheLastUsedSideKeySurvivesStaleAccountAndSnapshotWrites|TestBuildSchedulerMetadataAccount_KeepsQuotaStateForCachedAccounts)$' -count=1 -v
go test -tags=unit ./cmd/server -run '^(TestRunCleanupPhasesOrdersDependentDrains|TestActiveHandlerTrackerRejectsLateAdmissionAfterClose|TestRunCleanupPhasesStopsAfterProducerFailure|TestProvideCleanupOrdersProducerUsageAndDeferredDrains|TestShutdownServerWithDrainForceClosesBeforeCleanupAfterActiveHandlerFinishes|TestShutdownServerWithDrainReturnsAtHardDeadline)$' -count=1 -v
go test -tags=unit ./cmd/server -run '^(TestProvideCleanupOrdersProducersAndDependentDrains|TestProvideCleanupDrainsOpsErrorsBeforeEntTeardown)$' -count=1 -v
go test -tags=unit ./internal/handler -run '^(TestRecordCyberPolicyIfMarkedUsesUsageRecordWorkerPool|TestRecordCyberPolicyIfMarkedBillsBeforeBlockingModeration)$' -count=1 -v
go test -tags=unit ./internal/handler/admin -run '^TestUpdateGroupRequestReasoningEffortMappingsTriState$' -count=1 -v
```

六条均 exit `0`。历史 cmd/server regex 当前匹配 5 个测试；旧名 `TestProvideCleanupOrdersProducerUsageAndDeferredDrains` 已不存在，随后使用当前两个精确测试名执行并均 PASS，未把不存在的名字计为证据。

### Composite、scheduler、Sticky、fallback/WaitPlan 与 DB recheck

```text
go test ./internal/handler -run '^(TestGatewayHandlerResolveStickyRouteRecomputesCompositeFallbackDecision|TestGatewayHandlerResolveEffectiveGatewayRouteUsesConcreteClaudeCodeFallback|TestOpenAIGatewayHandler_MessagesUsesEffectiveClaudeCodeFallbackGroup|TestOpenAIGatewayHandler_CountTokensUsesEffectiveClaudeCodeFallbackGroup|TestOpenAIResponsesWebSocketResolvesCompositeExplicitAliasOnFirstFrame|TestOpenAIResponsesWebSocketResolvesCompositeExplicitAliasOnEveryFrame|TestOpenAIResponsesWebSocketAppliesAccountMappingAfterLaterCompositeRoute|TestOpenAIResponsesWebSocketRejectsLaterCrossProviderCompositeRoute|TestOpenAIResponsesWebSocketRejectsLaterCompositeNoRoute|TestOpenAIResponsesWebSocketClosesOnCompositeResolverError|TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream)$' -count=1 -v
go test ./internal/server/routes -run '^(TestCompositeTargetPlatformMiddleware(AppliesFallbackBeforeProtocolDispatch|LoadsFinalSubscription|ResolvesModelAndRestoresBody|UsesExplicitRouteAndRewritesBody|UsesExplicitRouteForMultipartImages|RejectsOversizedRuntimeRouteModels)|TestGatewayRoutesComposite(MessagesWithGrokModelUsesOpenAIGateway|ChatCompletionsWithGrokModelUsesOpenAIGateway|OpenAIOnlyEndpointsRequireOpenAITarget))$' -count=1 -v
go test ./internal/service -run '^(TestResolveCompositeRouteDecisionRecomputesForDifferentEffectiveGroup|TestResolvePlatformUsesEffectiveGroupCompositeDecision|TestCompositeGroupSchedulerHasAllCanonicalPlatformBuckets|TestCompositeRouteResolverExplicitRoutesCoverBucketTwoProviders|TestOpenAIGatewayService_SelectAccountWithScheduler_(Weighted_PreservesGrokSessionSticky|SessionStickyDBRuntimeRecheckSkipsStaleCachedAccount|DBFreshGroupRecheckReleasesMovedAccount|LayeredRequirePrivacySet)|TestLayered_(GroupedAccountPassesDBFreshRecheck|SessionStickyPreservesGrokBinding|WaitPlanFallbackSkipsUpstreamRestrictedAccount|FallbackWaitPlanRechecksPrivacyRequirementAgainstDB)|TestAdvancedScheduler(SharesProbeBudgetWithFallbackDBRechecks|ReleasesSlotWhenDBDisablesCandidate|ReacquiresOnceWhenDBConcurrencyChanges))$' -count=1 -v
go test ./internal/service -run '^(TestResolvePlatformUsesEffectiveGroupCompositeDecision|TestGatewayService_SelectAccountWithLoadAwareness|TestSelectAccountWithLoadAwareness_UsesFallbackGroupForChannelRestriction|TestSelectAccountWithLoadAwareness_StickyReadReuse|TestSelectAccountWithLoadAwareness_LoadBatchDisabledUsesPrefetchedGeminiStickyAccount|TestSelectAccountWithLoadAwareness_StickyDisabledBypassesPrefetchAndCacheRead|TestOpenAIGatewayService_SelectAccountWithLoadAwareness_DBFreshGroupRecheckWaitsOnValidAccount)$' -count=1 -v
go test -tags=unit ./internal/service -run '^(TestGatewayService_SelectAccountWithLoadAwareness|TestGatewaySelectAccountWithLoadAwareness_HydratesSelectedAccountFromSchedulerSnapshot)$' -count=1 -v
```

五条均 exit `0`；分别匹配当前 9 个 handler、9 个 routes、15 个 Path A service、5 个 Path B top-level 加 3 个 sticky 子例、2 个 unit top-level 加 28 个 load-aware 子例。

### Grok、Ollama、Alipay、email alias 与 migration compile

```text
go test ./internal/service -run '^(TestOllamaCloudUsage.*|TestShouldUseAlipayMobilePrecreate)$' -count=1 -v
go test -tags=unit ./internal/handler -run '^TestResponsesGrok402FailoverCooldown$' -count=1 -v
go test ./internal/handler/admin -run '^(TestOllamaCloudUsageSharedStateMatchesListDetailAndSpecialEndpointWithoutListNPlusOne|TestOllamaCloudUsageSharedStateCreateAndUpdateResponses|TestPaymentConfigService_UpdatePaymentConfig_PersistsAlipayForceQRCode|TestSettingHandler_UpdateSettings_PersistsPaymentAlipayForceQRCode)$' -count=1 -v
go test ./internal/repository -run '^(TestUserRepositoryExistsByEmailAlias|TestUserRepositoryExistsByEmailAliasIgnoresMalformedInput|TestUserRepositoryCreateWithEmailAliasGuard|TestUserRepositoryEmailAliasGuardFailsClosedWhenDotStrippedCandidatesSaturate)$' -count=1 -v
go test ./migrations -count=1 -v
go test ./internal/repository -run '^(TestApplyMigrations|TestLatestMigrationBaseline|TestIsMigrationChecksumCompatible|TestMigrationChecksumCompatibilityRules|TestValidateMigrationExecutionMode)' -count=1 -v
```

六条有效命令均 exit `0`；Ollama/Alipay 19 个 top-level、Grok 402 1 个、admin 4 个、alias 4 个、migration package 20 个、repository migration runner 18 个 top-level 测试 PASS。首次未带 `-tags=unit` 的 `TestResponsesGrok402FailoverCooldown` 命令 exit `0` 但输出 `no tests to run`，已排除，不作为证据；读取文件头确认其 `//go:build unit` 后执行了上列有效命令。

### Task 25 handler/service/openai_ws_v2 最终矩阵

```text
go test ./internal/handler -run '^(TestParseLiveCallRequestMultipartPreservesSession|TestParseLiveCallRequestJSONPreservesSessionWithoutDelegation|TestParseLiveCallRequestRejectsInvalidJSONShape|TestLiveEnabledForAPIKey|TestLiveAttestationErrorIsExplicit|TestOpenAIGatewayHandler_ChatAndEmbeddingsReplayMappedSpoolAcrossFailover|TestRequestBodyCoordinator_(JSON|Spool|Multipart|MultipartPipe|MultipartPipeRecoversProducerPanic|Cleanup|CleanupRemovesRawEffectiveAndMultipartTemps|CleanupGinTerminationPaths)|TestGatewayHandler_(MessagesCompressedRequestBodySpoolsUntilBlockedUpstreamCompletes|ResponsesCompressedRequestBodySpoolsEffectiveBodyUntilBlockedUpstreamCompletes|MessagesAndResponsesCanceledRequestsCleanSpools|MessagesAndResponsesReplayLargeBodiesAcrossFailover|RequestBodySpoolOpenFailureMapsTo503|ResponsesSpoolTransportFailureReturns503WithoutUsage)|TestOpenAIGatewayHandler_SubmitFailedUsageLogSnapshotsCompositeModelsBeforeQueue)$' -count=1 -v
go test ./internal/handler -run '^(TestOpenAIGatewayHandler_ResponsesFailedUsageUsesFinalOutboundModel|TestOpenAIGatewayHandler_GrokResponsesFailedUsageUsesFinalOutboundModel|TestOpenAIGatewayHandler_ResponsesPartialFailoverCreatesExactlyOneFailedUsage|TestOpenAIGatewayHandler_WSHTTPBridgeOrdinaryErrorUsage|TestOpenAIResponsesWebSocketFailedUsageFreezesChannelThenAccountMappedModel|TestOpenAIResponsesWebSocketFailoverResetsExactModelBeforeSecondAccountDialFailure|TestOpenAIResponsesWebSocketOAuthFailedUsageUsesNormalizedOutboundModel|TestOpenAIResponsesWebSocketPassthroughFailedUsageUsesActualFirstFrameModel|TestOpenAIResponsesWebSocketPassthroughNonRateLimitErrorRecordsFailedUsage|TestOpenAIResponsesWebSocketPassthroughPartialFirstErrorRecordsFailedUsage|TestOpenAIResponsesWebSocketPassthroughFailedUsageUsesSessionUpdatedModel|TestOpenAIResponsesWebSocketPassthroughSecondTurnAdmissionRejectsBeforeUpstream)$' -count=1 -v
go test ./internal/service -run '^(TestLiveCapabilityOnlyAllowsOpenAIOAuth|TestValidateLiveCallRequestDoesNotRequireDelegation|TestCreateUpstreamLiveCallPreservesSession|TestLiveCreateFailoverUsesExistingOpenAIPolicy|TestRequestTypeLive|TestFinalizeLiveCallIsIdempotentAndWritesZeroUsage|TestLiveSessionEndedTreatsLeaseLossAsTerminal|TestWaitForLiveObserverRetryLeavesExpiryToLoopFinalize|TestForwardAsAnthropic_InjectsPromptCacheKeyForAPIKeyMessagesDispatch|TestForwardAsAnthropic_AutoDerivesPromptCacheKeyWhenMessagesDispatchHasNoSessionID)$' -count=1 -v
go test ./internal/service -run '^(TestPassthroughLifecycle_(FirstWriteFailureCallsAfterTurnOnce|ClientDisconnectDrainsSecondTurnCompletionOnce|AppliesAccountMappingAfterLaterRequestRewrite)|TestOpenAIGatewayService_ProxyResponsesWebSocketFromClient_PassthroughHeadersUsePromptCacheAndTurnState|TestProxyOpenAIWSHTTPBridgeTurnLocalBuildFailureDoesNotPublishOutbound|TestProxyResponsesWebSocketFromClient_HTTPBridgeResetsAttemptBeforeFollowupLocalFailure|TestForwardAsChatCompletions_APIKeyAutoInjectsPromptCacheKey|TestNormalizeOpenAIModelForUpstream)$' -count=1 -v
go test ./internal/service/openai_ws_v2 -run '^(TestRelayOnTurnCompleteWaitsForTerminalDownstreamWrite|TestRelayCommitsTerminalBeforeFollowupAdmission|TestRelayBeforeWriteUpstreamWaitsForTurnPermit|TestRelay_OnTurnComplete_ProvidesTurnMetrics|TestRelay_ClientDisconnect_DrainCapturesLateUsage)$' -count=1 -v
go test ./internal/service/openai_ws_v2 -run '^(TestRelayMismatchedTerminalDoesNotCompleteActiveTurn|TestRunUpstreamToClient_ErrorAndDropPaths)$' -count=1 -v
go test ./internal/repository -run 'RequestType' -count=1 -v
go test ./internal/repository -run '^(TestLiveLeaseReplacesRegularSlotsAndCountsTowardLimits|TestLiveLeaseExpiresWithoutRefresh|TestGatewayCacheLiveCallIdentityAndController)$' -count=1 -v
go test -tags=unit ./internal/service -run '^(TestAdminService_CreateGroup_ClearsMessagesDispatchFieldsForNonOpenAIPlatform|TestAdminService_UpdateGroup_ClearsMessagesDispatchFieldsWhenPlatformChangesAwayFromOpenAI)$' -count=1 -v
```

九条均 exit `0`。覆盖 Live handler/service/lease/finalize/request type，prompt-cache 自动注入，HTTP/WS body replay、spooling、cleanup，failed usage/final outbound model，OAuth normalize，Grok final model，relay terminal ordering、turn permit、client-disconnect drain 和 group platform guard。

### Frontend 与 PostCSS 受影响面

```text
pnpm exec vitest run src/views/admin/__tests__/GroupsView.duplicate.spec.ts src/views/admin/__tests__/GroupsView.columnSettings.spec.ts src/components/admin/usage/__tests__/UsageFilters.spec.ts
pnpm exec vue-tsc --noEmit
pnpm exec node -e "console.log(require('postcss/package.json').version)"
```

在 `frontend/` 执行；三条均 exit `0`。Vitest 为 3 files / 17 tests PASS；typecheck 无输出；PostCSS 实际解析版本为 `8.5.23`。Browserslist stale-data advisory 非失败。

## `make test` 首次结果与修复循环

### 首次结果

- 命令：`make test`。
- source HEAD：`8c3b281f7f9e08a9f2d776f4241a922f7a85bff8`。
- 首次 exit：`2`，保留为非零，未写成 PASS。
- 所有默认 Go tests 已通过；`golangci-lint` 随后报告两个 `gofmt` finding：`backend/internal/handler/openai_gateway_handler_test.go:2138` 与 `backend/internal/service/openai_ws_v2_passthrough_adapter.go:1110`。
- `gofmt -d` 证明仅为已提交缩进漂移，无 token/行为变化。精确 formatter diff、三个受影响 WebSocket tests和 `golangci-lint run ./...` 均通过后，提交 `aff04f9cd style: format websocket ownership changes`。

### 修复后展开门禁前的第二次诊断

- 为确认 formatter 修复，曾再次执行 `make test`；exit `2`。默认 tests 与 lint 已通过，unit-tag 阶段唯一失败为 `TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream`：actual HeapAlloc `85,544,152`，ceiling `77,155,608`，差值相对 baseline 为 `20,971,456` bytes。
- 同一日志中的 `close of closed channel` 是失败 cleanup 触发 retry 后 fixture 第二次关闭 `started` 的次生 panic。
- 根因：Images handler 在 `contentModerationService == nil` 时仍无条件执行 `parsed.ModerationBody()`，为 20 MiB prompt 构造另一份完整 JSON payload；这是默认禁用 moderation 路径不需要的分配。
- 最小修复 `1cc41c72c fix: avoid unused image moderation payload`：只在 moderation service 存在时构造 payload，并用 `sync.Once` 保护 fixture 的 `started` close。
- RED 为上述完整 unit-tag run 的真实 HeapAlloc failure；GREEN：精确 OAuth test exit `0`，随后 `-count=10` 为 10/10，moderation-enabled sibling test exit `0`，`go test -tags=unit ./internal/handler -count=1` exit `0`（`113.787s`），handler lint `0 issues`。

### 最终 source/test HEAD 的展开式 full test gate

为保留 `make test` 的首次非零记录，最终 source/test HEAD `1cc41c72c1def83113263d9b631f9856dbff030d` 逐项执行其组成命令：

```text
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/test.ps1 ./...
golangci-lint run ./...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/test.ps1 -tags=unit ./...
pnpm --dir frontend run lint:check
pnpm --dir frontend run typecheck
pnpm --dir frontend run test:run
```

- 六条均 exit `0`。
- 默认 handler package `74.953s`，service `108.182s`；unit handler package `113.657s`，service `162.956s`；OAuth retention target 在完整 unit package 中通过。
- `golangci-lint` 为 `0 issues`。
- 前端 lint/typecheck 成功；Vitest 为 `215 passed` files / `1626 passed` tests。
- Vue router-link、预期负路径 stderr、intlify message-compiler、Browserslist advisory 均为既有非失败输出。

### 2026-07-30 第三次原样非零与最终原样 PASS

- 在 formal report commit `35dddd7aeafc9d1db1267241757cb0c93c35ff67` 上仅原样执行一次 `make test`；source/test 仍为其父提交 `1cc41c72c1def83113263d9b631f9856dbff030d`。exit `2`，不得记为 PASS。
- 该次 default Go tests 与 `golangci-lint run ./...` 均通过；unit-tag 阶段唯一失败为 `TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream`，HeapAlloc `85,602,768 > 77,228,584`，相对 baseline 增量 `20,957,096` bytes，约为一个 20 MiB prompt。依赖的 frontend targets 未启动。
- 只读调查确认 `1cc41c72c` 已守卫 content moderation payload，但 security-audit 实参仍无条件求值 `parsed.ModerationBody()`；moderation/audit 双 nil 路径因此仍构造完整 prompt payload。`6c88f1891650e0ef18b0b5ae105b8f44a069a5a4 fix: avoid unused image audit payload` 仅在任一检查已配置时构造一次 frozen payload，并由两条检查复用。
- 修复后 focused target、`-count=10`、moderation-enabled sibling、4 个 security-audit media targets、完整 unit handler package 与 handler lint 均 exit `0`。
- 在最终 source/test commit `6c88f1891650e0ef18b0b5ae105b8f44a069a5a4` 上原样执行唯一一次 `make test`，exit `0`，未自动重跑。default handler `61.833s`、service `105.188s`；lint `0 issues.`；unit handler `98.673s`、service `156.695s`；frontend lint/typecheck 成功，Vitest `215 passed` files / `1626 passed` tests。
- 历史口径保持不变：前三次原样 `make test` 均为真实 exit `2`，只有最终 source/test commit 上的第四次原样执行为 exit `0`；此前 `1cc41c72c` 的六条展开式绿色证据仍是历史证据，不替代本次原样 PASS。

## Windows build 与版本证据

```text
make "VERSION=0.1.165.1" "SHELL=D:/scoop/shims/bash.exe" build
```

- exit `0`。
- backend 实际命令包含 `-X main.Version=0.1.165.1`；frontend Vite 处理 `1019` modules 并完成 production build。
- `backend/cmd/server/VERSION` 精确为 `0.1.165.1`。
- `rg -a -o -m 1 "0\.1\.165\.1" backend/bin/server`：exit `0`，输出 `0.1.165.1`。
- `go version -m backend/bin/server` 显示 Go `1.26.5`、`vcs.revision=1cc41c72c1def83113263d9b631f9856dbff030d`；`vcs.modified=true` 来自明确排除的用户 dirty report，不影响已注入版本字符串。
- PostCSS 解析版本 `8.5.23`；Vite dynamic-import/chunk-size 与 Browserslist 输出为既有 advisory。
- 在最终 source/test commit `6c88f1891650e0ef18b0b5ae105b8f44a069a5a4` 上再次执行同一版本化 build，exit `0`；VERSION 文件和二进制字符串均为 `0.1.165.1`，`go version -m backend/bin/server` 显示 Go `1.26.5`、`vcs.revision=6c88f1891650e0ef18b0b5ae105b8f44a069a5a4`，Vite 处理 `1019` modules。

## Ent/Wire 两轮生成

- 既有 ledger 已记录主树 `user-mapped section` 风险，因此未在主树手改生成物。
- 创建短 detached worktree `D:\w27`，HEAD 精确为 `1cc41c72c1def83113263d9b631f9856dbff030d`。
- 第一轮 `make -C backend generate`：exit `0`；随后 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go`：exit `0`、无输出。
- 第二轮相同两条命令：均 exit `0`、generated diff 无输出。
- detached worktree `git status --short` 无输出；`git worktree remove D:/w27` exit `0`；最终 `Test-Path D:\w27` 为 `False`。
- 未手改任何 Ent/Wire 生成物。
- 最终 source/test commit 另使用 detached worktree `D:\w27-final` 重跑两轮 `make -C backend generate`；两轮均 exit `0`，每轮 Ent/Wire diff check 均 exit `0` 且无输出，worktree status 为空。移除 worktree exit `0`，最终 `Test-Path=False`。

## 静态检查

- `git diff --check`：exit `0`；仅对排除的 `.superpowers/sdd/task-4-report.md` 打印 LF/CRLF 提示，无 whitespace error。
- `git diff --name-only --diff-filter=U`：exit `0`，count `0`。
- `git ls-files -u`：exit `0`，count `0`。
- 精确 tracked conflict marker 扫描 `^(<<<<<<< .+|=======|>>>>>>> .+)$`（排除 formal reports）：exit `0`，count `0`。
- legacy first-token scan：exit `0`，count `0`；上游 `first_output_timeout` 不在移除扫描范围。
- VERSION assertion：exit `0`，`version=0.1.165.1`。
- `git diff -- opencode.json`：exit `0`，无输出。
- protected migration presence：exit `0`，count `14`。存在本地 `172_video_per_second_billing_metadata.sql`、`181_group_duplicate_operation_id.sql`，上游同号 `172_composite_model_routes.sql`、`181_prompt_audit.sql`，以及 182、183、184、185、双 186、187、188、189、`190_add_users_email_alias_dedup_index_notx.sql`。
- 此处未执行 Task 28 的完整 tag ancestry/六个 merge 第二父检查，不作相关声明。
- 最终 source/test commit 的同组静态检查再次通过：无 unmerged index/diff、tracked conflict marker count `0`、backend/frontend legacy first-token count `0`、VERSION 为 `0.1.165.1`、`opencode.json` 无 diff，14 个 protected migration 全部存在且 missing count `0`。

## Remote final integration

### 固定对象与执行

- 已先加载 `ssh-skill`；所有远程命令、上传和下载分别只使用 `ssh_execute.py`、`ssh_upload.py`、`ssh_download.py`，alias 为 `local-serv-ai`；未使用 raw SSH/SCP。
- stage：`final-verify`。
- source SHA：`1cc41c72c1def83113263d9b631f9856dbff030d`。
- nonce：`cbfc15a98f5e418e92f2944efeecb676`。
- 唯一 remote 目录：`/tmp/sub2api-final-verify-cbfc15a98f5e418e92f2944efeecb676`。
- local archive：`C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-cbfc15a98f5e418e92f2944efeecb676.tar`，只由 `git archive --format=tar 1cc41c72c1def83113263d9b631f9856dbff030d` 创建。
- remote archive SHA-256：`45143bf25d809063ad14fbf753649c0dcd21fb281a216730d152796feebbf651`。
- create JSON：`success=true, exit_code=0`；upload JSON：`success=true, exit_code=0`。
- preflight JSON：`success=true, exit_code=0`；Go `1.26.5 linux/amd64`，满足 `backend/go.mod`；Docker Server `29.2.1` 且 `docker info` 成功。

远程解包后删除并重建 `src/backend/.test-tmp`，实际命令为：

```text
CI=true GOFLAGS='-v' TMPDIR='/tmp/sub2api-final-verify-cbfc15a98f5e418e92f2944efeecb676/src/backend/.test-tmp' TMP='/tmp/sub2api-final-verify-cbfc15a98f5e418e92f2944efeecb676/src/backend/.test-tmp' TEMP='/tmp/sub2api-final-verify-cbfc15a98f5e418e92f2944efeecb676/src/backend/.test-tmp' go test -tags=integration ./... > '/tmp/sub2api-final-verify-cbfc15a98f5e418e92f2944efeecb676/integration.log' 2>&1
```

- integration execution JSON：`success=true, exit_code=0`。
- download JSON：`success=true, exit_code=0`。
- 本地日志：`C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-cbfc15a98f5e418e92f2944efeecb676-integration.log`，`4,264,520` bytes。
- 日志中 `--- FAIL:`/package `FAIL` count `0`。

### Migration targets

- `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate`：明确 PASS（`5.50s`）。
- `TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages`：明确 PASS（`4.29s`）。
- 12/12 证据不是伪造的日志输出：committed test 在 `upstreamMigrations` 中固定列出 12 个上游完整 filename，并以 `require.Len(..., 12)` 同时约束列表和 embedded FS；通过 target 还明确检查双 186、`190_*_notx.sql`、本地 172/181、full filename identity、重复 apply record count 不增加和 checksum 不变。

### SKIP 分类

日志共有 13 个 `--- SKIP:`（12 top-level + 1 nested）：

- `TestDingTalkOAuthStart_Disabled`：既有 sentinel，日志注明 helper 尚待 Task 1.10；非本 change 能力。
- `TestDialerAgainstCaptureServer`：未设置外部 `TLSFINGERPRINT_CAPTURE_URL`。
- `TestConcurrencyCacheSuite/TestGetAccountsLoadBatch`：源码显式既有 TODO，CI 中 CurrentConcurrency 期望待修；同 suite 的 Live lease、WS ingress、其他 concurrency tests 均执行。
- `TestPromptAuditConfigCASSecretRoundTripInvalidationAndTTL`、`TestRedisPayloadStoreRoundTripTTLNamespaceAndDelete`、`TestPromptRuntimeAggregatesConfigWorkersQueueRedisEndpointsAndGuardMetrics`：未设置 `PROMPT_AUDIT_TEST_REDIS_ADDR`。
- `TestPromptAuditMigrationSchemaAndLeakageGate`、`TestPromptAuditDatabasePersistsFullPromptOnEventsOnly`、`TestPromptAuditRepositoryAdmissionClaimFencingAndEventTransaction`、`TestPromptAuditRepositoryForeignKeysFiltersAndStableIdentitySnapshots`、`TestPromptAuditRepositoryHighWaterAndSafeDeletion`、`TestPromptAuditServiceConfirmationKeepsPostPreviewEventsAndConcurrentDeletesAreSafe`：未设置外部 `PROMPT_AUDIT_TEST_POSTGRES_DSN`。
- `TestEstimateOpenAIInputTokens_CompareWithOpenAIAPI`：未设置外部 `OPENAI_API_KEY`。

这些 skip 均未命中 Task 27 要求的 migration targets、Live lifecycle、prompt-cache/body replay、failed usage/final model、Grok/Ollama/email alias 或 frontend 受影响面，因此不阻断本轮 integration。

### Cleanup 与范围

- remote cleanup JSON：`success=true, exit_code=0`；执行 `rm -rf` 后 `test ! -e` 成功。
- local archive cleanup 后 `Test-Path` 为 `False`；下载日志 `Test-Path` 为 `True` 并保留。
- 只允许 Testcontainers 使用现有 PostgreSQL/Redis 测试镜像；未构建 Sub2API 镜像，未部署，未接触服务运行目录或生产数据库/Redis。

### 最终 source 的两次受控失败

- 第一次 nonce `eecce42b226d4e03a8cb2d875070f12e` 使用固定 commit `6c88f1891650e0ef18b0b5ae105b8f44a069a5a4` 的 `git archive`；本地 archive size `53,739,520` bytes，SHA-256 `be2dce002dd26e3ba1faaa79e83052a3aa8b2a70ddcd76bd76c8f56142e960d3`。
- 第一次 remote create/preflight 因含 Docker brace/template 的命令形状在 Windows native fallback 本机参数解析阶段失败：JSON `success=false, exit_code=1`。未上传、解包、运行 integration 或下载日志；remote cleanup exit `0`，唯一目录不存在，本地 tar 已删除。该失败不改写为远程 PASS。
- 第二次 nonce `a9bada9b52f244e8bfc39ba41e9f092d` 改用不含 Docker brace/template 的简单 preflight；preflight、upload、SHA 校验与 setup 均 exit `0`，remote SHA 与上述 archive 一致。原样 integration execution exit `1`，未重跑。
- 第二次日志没有 `--- FAIL:`，但 `internal/server/routes.test` 与 `internal/service.test` 均在 linker 阶段报 `mapping output file failed: no space left on device` 并形成 package `FAIL ... [build failed]`。两个 migration targets 已 PASS（`4.93s`、`4.20s`），日志当时只有 12 个 skip；外部 OpenAI API skip 因 service package 未链接完成而未出现，不能当作完整覆盖。
- 第二次 download 与 remote cleanup 均 exit `0`，唯一目录不存在，本地 tar 已删除，失败日志保留。required integration exit `1` 的历史保持为 BLOCKED 证据。

### ENOSPC 诊断与最小恢复

- 只读诊断确认 `/`、`/tmp`、`/var/tmp`、`GOCACHE`、`GOPATH` 与 Go temp 全部位于同一 `/dev/mapper/rl-root` XFS：总量 `35G`，已用 `34G`，仅余 `1.3G`、使用率 `97%`；inode 使用率仅 `9%`。`GOCACHE=/root/.cache/go-build` 占 `14G`，module cache 约 `1017M`，没有遗留 Go temp 或失败 nonce 目录。
- Docker build cache 为 `0B`，Testcontainers containers/volumes 为空；活动生产容器及无法归属的 `/tmp`/anonymous volumes 均未触碰。证据将根因限定为 linker temp 与持久 Go build cache 共享根卷时的 block capacity 峰值，不是 inode 或 Docker/Testcontainers。
- 恢复前使用不自匹配的精确 comm 查询 `go`、`compile`、`link`、`asm`、`cgo`、`vet`，并单独查询 `.test` binaries；JSON `success=true, exit_code=0`，两段输出均为空，确认没有并发 Go build/test/toolchain 进程。
- 随后仅执行一次 `go clean -cache`；JSON `success=true, exit_code=0, stderr=""`。未清 modcache/testcache，未 Docker prune，未删除任何 image/container/volume、未知 `/tmp` 或服务目录。
- 清理后 `/root/.cache/go-build` 为 `20K`；`df -hT / /tmp /var/tmp` 均为总量 `35G`、已用 `20G`、可用 `15G`、使用率 `58%`，inode 使用率 `2%`，确认实际回收出足够余量。

### 最终 source remote PASS

- stage `final-verify`；nonce `860617d8c7d8427a944f30c0a915c894`；唯一 remote 目录 `/tmp/sub2api-final-verify-860617d8c7d8427a944f30c0a915c894`。local archive 只由 `git archive --format=tar 6c88f1891650e0ef18b0b5ae105b8f44a069a5a4` 创建，size `53,739,520` bytes，local/remote SHA-256 均为 `be2dce002dd26e3ba1faaa79e83052a3aa8b2a70ddcd76bd76c8f56142e960d3`。
- 简单 preflight、upload、SHA 校验、解包及 `.test-tmp` 重建均 `success=true, exit_code=0`；Go `1.26.5 linux/amd64`，Docker client/server `29.2.1`，`docker info` 成功。
- 在 `src/backend` 设置 `CI=true GOFLAGS='-v'`，并将 `TMPDIR`/`TMP`/`TEMP` 指向重建后的 nonce `.test-tmp`；原样执行 `go test -tags=integration ./...` 到远程日志，timeout `1800`，未加 `-p`、未重试或改变测试语义。execution JSON `success=true, exit_code=0`。
- download JSON `success=true, exit_code=0`。本地日志 `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-860617d8c7d8427a944f30c0a915c894-integration.log`，`4,264,020` bytes，SHA-256 `26b347735ed1d244775a835a86c4e709d29fd5dc498e92981f55dfac09bca47d`；`--- FAIL:`/package `FAIL` count `0`。
- `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 明确 PASS（`5.34s`）；`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` 明确 PASS（`4.80s`）。committed test 以两个 `require.Len(..., 12)` 约束 upstream list 与 embedded FS，且覆盖本地/上游同号 172/181、双 186、`190_*_notx.sql`、full filename identity、重复 apply record count 与 checksum 不变；target PASS 因而提供 12/12 证据。
- 日志共有 13 个 `--- SKIP:`（12 top-level + 1 nested）：DingTalk Task 1.10 sentinel、未配置外部 TLS capture、既有 CI CurrentConcurrency TODO、3 个未配置 prompt-audit Redis fixture、6 个未配置 prompt-audit PostgreSQL fixture，以及未配置外部 `OPENAI_API_KEY` 的 token comparison。均未命中 required migration targets 或 Task 27 受影响面。
- 测试后 `/`、`/tmp`、`/var/tmp` 均为总量 `35G`、已用 `22G`、可用 `14G`、使用率 `62%`，inode 使用率 `2%`；未做额外 Go/Docker 清理。
- remote cleanup 与 local archive cleanup 均 exit `0`：唯一 remote 目录不存在，local tar 不存在，下载日志保留。所有远程操作仅使用 ssh-skill Python scripts；未使用 raw SSH/SCP。

## 未执行项与残余风险

- 未执行 Task 28 完整 ancestry、merge parent 和范围边界验证；本轮静态 migration presence 不能替代 Task 28。
- 未执行 Task 29 Chrome DevTools 前端烟测、能力终审、OpenSpec validate/tasks/progress/comet 收口。
- 未 push、tag、release、deploy、构建 Sub2API 镜像或触发 workflow。
- Task 25 遗留：Grok 在 request build 前设置 attempted 标志；client-disconnect drain lifecycle regression 使用固定 `50ms` 排序延迟。当前命名矩阵和完整门禁均通过，但这两项仍是非阻断测试/诊断风险。
- OAuth Images HeapAlloc 有历史波动，本轮在完整 unit gate 再次真实复现并完成根因修复；修复后精确 10/10、moderation-enabled sibling、完整 unit handler 和最终展开式 gate 均通过。该测试仍使用 process-wide HeapAlloc，因此保留历史敏感性说明，不把先前失败抹除。
- Windows 当前 `CGO_ENABLED=0` 且无 `gcc`，未运行 `-race`；不声称 race-clean。
- 流程 concern：三次历史 `make test` exit `2` 均已保留；最终 source/test commit `6c88f1891650e0ef18b0b5ae105b8f44a069a5a4` 上另有唯一一次原样 `make test` exit `0`，结论不再只依赖展开式 gate。首次 SSH preflight 本机解析失败与第二次 remote ENOSPC 也保持为真实失败历史。
- 远程环境曾因共享根卷仅余 `1.3G` 在 linker 阶段 ENOSPC；最小 cache 恢复后本次测试峰值结束仍余 `14G`。长期风险是同一 GOCACHE 可再次增长，后续门禁仍应先观察容量，但本轮不追加周期清理或 Docker 处置。
