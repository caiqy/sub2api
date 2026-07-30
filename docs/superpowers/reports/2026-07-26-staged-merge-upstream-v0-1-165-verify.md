# 分段合入上游 v0.1.165 最终自动验证报告

## 结论与边界

- 状态：`DONE_WITH_CONCERNS`。
- 分支：`feature/20260726/staged-merge-upstream-v0-1-165`。
- Task 27 起始 HEAD：`8c3b281f7f9e08a9f2d776f4241a922f7a85bff8`。
- 最终 source/test HEAD：`417bbcc6a44c35b3e3ed16efb0bb86a4717401c9`。
- 最终版本：`0.1.165.1`。
- 本轮发现并提交五个修复：`aff04f9cd` 仅格式化两个 WebSocket ownership 路径；`1cc41c72c` 避免 content moderation 未配置时构造完整 Images moderation payload，并使 OAuth retention fixture 的 started 信号可重入；`6c88f1891` 避免 content moderation 与 security audit 均未配置时仍构造完整 Images audit payload；`90b008901` 以一个 memoized lazy body 统一 Images 的 prompt/legacy security audit，消除生产形态重复 legacy 调用；`417bbcc6a` 为 blocking prompt/legacy goroutine 分别复制 `Request`，消除共享 `req.Body` 数据竞争并补充 lazy provider 生命周期契约。
- 最终 source/test HEAD 的 Linux race GREEN、完整原样 `make test`、版本化 Windows build、两轮 Ent/Wire generate、静态检查和全新 nonce 远程 integration 均 exit `0`；最终 Vitest 为 215 files / 1626 tests，integration 为 FAIL `0`、migration `2/2`、13 skips。此前 source `90b008901` 的原样 `make test` EOF/1013 非零、首次 SSH preflight 本机解析失败、remote ENOSPC、race `-count=1` 未命中及放大后 DATA RACE RED 均继续保留，不改写为 PASS。
- Task 28 已在验证基线 `de3fffdd76f79831b4503ebfc204b0dc4cd156e7` 完成完整 ancestry、六个 merge 第二父、范围边界和 migration 静态复核，证据见文末 Task 28 专节；Task 29 浏览器烟测与 OpenSpec 8.4 收口仍未执行，本报告不声称这些后续任务完成。

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
- Task 28 后续已完成完整 tag ancestry、六个 merge 第二父和范围边界检查，见文末 Task 28 专节；本段仍只记录 Task 27 当时的静态证据。
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

## 统一审计 source `90b008901` 最终补验

### 生产形态 TDD 与 source commit

- 最终 source/test SHA：`90b0089010c77d0fef1790e2e8cf3b675d4994cd`，提交为 `90b008901 fix: unify image security audit payload`。
- 三组生产形态 RED 均真实执行：service/coordinator 先因 lazy 接口不存在编译失败；Images 生产注入测试真实得到 legacy 调用次数 `expected 1, actual 2`。未用空断言或仅编译失败替代 Images 行为 RED。
- 步骤 4 三组 GREEN 均 exit `0`。Images 四 targets `-count=10` 为 40/40 PASS；`go -C backend test ./internal/securityaudit -count=1`、`go -C backend test -tags=unit ./internal/service -count=1`、`go -C backend test -tags=unit ./internal/handler -count=1` 和聚焦 lint 均 exit `0`。
- 实现保持 eager `Check` wrapper，只新增窄 lazy 入口；coordinator 以 `sync.Once` 让 prompt 与 legacy 共享 frozen body；Images 删除 direct moderation 调用并只走一次统一 security-audit helper。未提高 HeapAlloc ceiling、未增加配置或通用缓存框架。

### 原样 `make test` 非零与 constituent 证据

```text
make test
```

- 在 `90b0089010c77d0fef1790e2e8cf3b675d4994cd` 上仅原样执行一次，exit `2`，不得记为 PASS。default Go tests、`golangci-lint run ./...`（`0 issues.`）和 unit handler package 均通过；frontend targets 因 unit service 非零而未启动。
- 唯一失败为 `TestPassthroughLifecycle_LeaseLossSendsRetryClose`：服务端 trace 明确将 `ErrOpenAIWSIngressLeaseLost` 映射为 close code 1013，客户端断言偶发只读到 `failed to read frame header: EOF`。精确 `-count=10` 诊断 exit `1`，9 PASS/1 同型 EOF，且每次服务端均产生 1013。
- 用户已明确接受该精确 EOF/服务端 1013 上游基线例外；接受例外不改变两条命令的非零事实，也不把它们写为 PASS。完整输出保存在 `C:/Users/caiqy/.local/share/opencode/tool-output/tool_fb0dc8a92001b12CrNYUOz3RYV`。
- 同一 source 内容另有完整 unit service package exit `0`、unit handler package exit `0`、default constituent exit `0`、全量 lint exit `0` 的独立证据；因此保留原样 gate 非零，同时不丢失各组成门禁的实际绿色证据。

首次 `make test` 未启动的 frontend constituents 在同一 source HEAD 补执行：

```text
pnpm --dir frontend run lint:check
pnpm --dir frontend run typecheck
pnpm --dir frontend run test:run
```

- 三条均 exit `0`；Vitest 为 `215 passed` files / `1626 passed` tests，duration `71.55s`。router-link、预期负路径和 intlify 输出均为既有非失败 stderr。

### Windows build、generate 与静态检查

```text
make "VERSION=0.1.165.1" "SHELL=D:/scoop/shims/bash.exe" build
```

- exit `0`；backend ldflags 包含 `-X main.Version=0.1.165.1`，Vite 处理 `1019` modules。`backend/cmd/server/VERSION` 和二进制字符串均为 `0.1.165.1`；`go version -m backend/bin/server` 显示 Go `1.26.5`、`vcs.revision=90b0089010c77d0fef1790e2e8cf3b675d4994cd`。

detached worktree 精确命令与结果：

```text
git worktree prune && git worktree add --detach "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-task27-generate-90b008901" 90b0089010c77d0fef1790e2e8cf3b675d4994cd
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git status --short
git worktree remove "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-task27-generate-90b008901"
```

- 全部 exit `0`；两轮 Ent/Wire diff 均无输出，worktree status 为空。`Test-Path` 最终为 `False`，`git worktree list --porcelain` 只剩主树；两个目录已不存在的历史 Task 27 stale metadata 也由 `git worktree prune` 清理。
- `git diff --check` exit `0`，仅对禁止暂存的 Task 27/Task 4/progress 文件输出 LF/CRLF warning；`git diff --name-only --diff-filter=U` 与 `git ls-files -u` 均 exit `0`、无 unmerged 文件。
- 精确 tracked conflict marker scan exit `0`、count `0`；Task 26 原样 legacy first-token scan exit `0`、count `0`；VERSION 精确为 `0.1.165.1`。
- protected migration presence 检查 exit `0`、count `14`，双方 172/181、双 186 和 182-190（含 `190_*_notx.sql`）全部存在。
- `git diff -- opencode.json` exit `0` 但有输出：用户新增 CodeGraph MCP 与 disabled ddg-search 配置。该文件从起点即 dirty，按契约保留且禁止暂存；未把它误写成无 diff。

### Remote final integration 精确命令

- 已先加载 `ssh-skill`；所有服务器操作仅使用其 Python scripts，alias `local-serv-ai`。stage `final-verify`，新 nonce `547fff8e23a24a92ad566202331ba360`，唯一 remote 目录 `/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360`。
- local archive `C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-547fff8e23a24a92ad566202331ba360.tar` 只由下列固定 SHA 命令创建：

```text
git archive --format=tar --output="C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-547fff8e23a24a92ad566202331ba360.tar" 90b0089010c77d0fef1790e2e8cf3b675d4994cd
```

- exit `0`；size `53,770,240` bytes；local SHA-256 `f7121a186eafd3f80e86f52ea02f264bf7b945559d24757ea9df0be87e0743b8`。

preflight 精确命令：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; test ! -e /tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360; mkdir -m 700 /tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360; go version; docker version; docker info >/dev/null; df -hT / /tmp /var/tmp; df -i / /tmp /var/tmp; echo preflight=ok" --timeout 120
```

- JSON `success=true, exit_code=0, stderr=""`；Go `1.26.5 linux/amd64`，Docker client/server `29.2.1`，`docker info` 成功。根卷总量 `35G`、已用 `21G`、可用 `14G`、使用率 `60%`，inode 使用率 `2%`。

upload 精确命令：

```text
$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_upload.py local-serv-ai "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-547fff8e23a24a92ad566202331ba360.tar" "/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/source.tar" --no-progress
```

- JSON `success=true, exit_code=0, stderr=""`。

setup 精确命令：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360; sha256sum source.tar; mkdir src; tar -xf source.tar -C src; rm -f source.tar; rm -rf src/backend/.test-tmp; mkdir -p src/backend/.test-tmp; test -d src/backend/.test-tmp; echo setup=ok" --timeout 300
```

- JSON `success=true, exit_code=0, stderr=""`；remote SHA-256 `f7121a186eafd3f80e86f52ea02f264bf7b945559d24757ea9df0be87e0743b8` 与 local 一致，`.test-tmp` 已删除重建。

integration 精确命令：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/src/backend; CI=true GOFLAGS='-v' TMPDIR='/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/src/backend/.test-tmp' TMP='/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/src/backend/.test-tmp' TEMP='/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/src/backend/.test-tmp' go test -tags=integration ./... > '/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/integration.log' 2>&1" --timeout 1800
```

- JSON `success=true, exit_code=0, stdout="", stderr=""`；未加 `-p`、未重试或改变测试语义。

download 精确命令：

```text
$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_download.py local-serv-ai "/tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360/integration.log" "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-547fff8e23a24a92ad566202331ba360-integration.log" --no-progress
```

- JSON `success=true, exit_code=0, stderr=""`。local log size `4,279,146` bytes，SHA-256 `d5bbe067dbbab72529073539aea60ef96f567e807b9a4c0591f374f685a021d6`；`--- FAIL:`/package `FAIL` count `0`。

cleanup 精确命令：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; df -hT / /tmp /var/tmp; df -i / /tmp /var/tmp; rm -rf /tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360; test ! -e /tmp/sub2api-final-verify-547fff8e23a24a92ad566202331ba360; echo cleanup=ok" --timeout 120
```

- JSON `success=true, exit_code=0, stderr=""`；测试后根卷总量 `35G`、已用 `22G`、可用 `14G`、使用率 `62%`，inode 使用率 `2%`。唯一 remote 目录不存在；local tar cleanup exit `0`、`archive_exists=False`，下载日志保留且 `log_exists=True`。

### Migration 与 SKIP 分类

- `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 明确 PASS（`5.23s`）；`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` 明确 PASS（`4.98s`）。后者实际执行 `require.Len(t, upstreamMigrations, 12)` 和 `require.Len(t, currentUpstream, 12)`，并覆盖本地/上游 172/181、双 186、`190_*_notx.sql`、完整 filename identity、重复 apply record count 与 checksum 不变，形成 12/12 证据。
- 日志共有 13 个 `--- SKIP:`（12 top-level + 1 nested）：DingTalk Task 1.10 sentinel；未设置 `TLSFINGERPRINT_CAPTURE_URL`；既有 CI `CurrentConcurrency` TODO；3 个未设置 `PROMPT_AUDIT_TEST_REDIS_ADDR` 的 Redis fixture；6 个未设置 `PROMPT_AUDIT_TEST_POSTGRES_DSN` 的 PostgreSQL fixture；未设置外部 `OPENAI_API_KEY` 的 token comparison。均未命中 required migration targets 或 Task 27 受影响面。
- 未 Docker prune、未构建 Sub2API 镜像、未部署，未接触服务运行目录、生产数据库或 Redis；Testcontainers 只使用既有测试镜像。

## Blocking audit race 修复与最终 source `417bbcc6a`

### gcc 授权安装

用户明确授权在 `local-serv-ai` 安装 gcc，仅允许 gcc 及包管理器自动解析的必要依赖。安装前只读命令：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cat /etc/os-release; echo package_managers; command -v dnf || true; command -v yum || true; command -v apt-get || true; echo gcc_status; rpm -q gcc || true; command -v gcc || true; echo package_processes; ps -C dnf,yum,rpm,apt,apt-get,dpkg -o pid=,comm=,args= || true; echo lock_holders; if command -v fuser >/dev/null 2>&1; then fuser /var/run/dnf.pid /var/cache/dnf/metadata_lock.pid /var/lib/rpm/.rpm.lock /var/lib/dpkg/lock /var/lib/dpkg/lock-frontend 2>/dev/null || true; else echo fuser=missing; fi; echo readonly_preflight=ok" --timeout 120
```

- JSON `success=true, exit_code=0`；系统为 Rocky Linux `9.7 (Blue Onyx)`，`dnf`/`yum` 可用，gcc 未安装；没有并发 dnf/yum/rpm/apt/dpkg 进程或 lock holder。

安装精确命令：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; dnf install -y gcc" --timeout 600
```

- JSON `success=true, exit_code=0`。dnf 安装 12 个包：`gcc`、`cpp`、`glibc-devel`、`glibc-headers`、`kernel-headers`、`libmpc`、`libpkgconf`、`libxcrypt-devel`、`make`、`pkgconf`、`pkgconf-m4`、`pkgconf-pkg-config`；并按依赖求解升级 6 个既有包：`glibc`、`glibc-common`、`glibc-gconv-extra`、`glibc-langpack-en`、`libgcc`、`libgomp`。未安装开发工具组，未改 Go/Docker/服务/生产数据，完成后按授权保留 gcc。

安装后验证：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; gcc --version; rpm -q gcc; CGO_ENABLED=1 CC=gcc go env CGO_ENABLED CC; echo gcc_ready=ok" --timeout 120
```

- JSON `success=true, exit_code=0`；gcc 为 `11.5.0-14.el9.x86_64`，显式 Go 环境输出 `CGO_ENABLED=1`、`CC=gcc`。

### 固定 source `90b008901` race RED

- stage `race-red`；nonce `714d12f8d8004666ac19f8e95fea6b1e`；唯一 remote 目录 `/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e`。

```text
git archive --format=tar --output="C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e.tar" 90b0089010c77d0fef1790e2e8cf3b675d4994cd
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; test ! -e /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e; mkdir -m 700 /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e; go version; gcc --version | head -n 1; df -hT / /tmp; echo preflight=ok" --timeout 120
$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_upload.py local-serv-ai "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e.tar" "/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/source.tar" --no-progress
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e; sha256sum source.tar; mkdir src; tar -xf source.tar -C src; rm -f source.tar; rm -rf src/backend/.test-tmp; mkdir -p src/backend/.test-tmp; echo setup=ok" --timeout 300
```

- 四条均 exit `0`；archive size `53,770,240` bytes，local/remote SHA-256 均为 `f7121a186eafd3f80e86f52ea02f264bf7b945559d24757ea9df0be87e0743b8`；Go `1.26.5 linux/amd64`、gcc `11.5.0`。

首次按指定 count 执行：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend; CGO_ENABLED=1 CC=gcc TMPDIR='/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend/.test-tmp' TMP='/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend/.test-tmp' TEMP='/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend/.test-tmp' go test -race ./internal/securityaudit -run '^TestCoordinatorCheckLazyEvaluatesBodyOnceAcrossPromptAndLegacy$' -count=1 > '/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/race-red.log' 2>&1" --timeout 900
```

- JSON `success=true, exit_code=0`；该次调度未命中 race，日志为 package `ok`。日志下载成功，size `63` bytes，SHA-256 `a74e7876cb44f4b5478de2b4f2b8fa2e5fcb5be1e3f2fada0dfd06483efe2cbd`；不作为 RED。
- 按原任务允许的调度放大只提高 count，不改源码或测试：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend; CGO_ENABLED=1 CC=gcc TMPDIR='/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend/.test-tmp' TMP='/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend/.test-tmp' TEMP='/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/src/backend/.test-tmp' go test -race ./internal/securityaudit -run '^TestCoordinatorCheckLazyEvaluatesBodyOnceAcrossPromptAndLegacy$' -count=100 > '/tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e/race-red-count100.log' 2>&1" --timeout 900
```

- JSON `success=false, exit_code=1`；下载的 RED log size `2,528` bytes，SHA-256 `9ae589906c64c1656a386940e88420dc27ea92e6ccc3b993c594a6797b51a698`。日志明确包含 `WARNING: DATA RACE`：goroutine 35 在 `coordinator.go:82` 读取共享 `req`，goroutine 36 在 `coordinator.go:90` 写 `req.Body`，随后 `race detected during execution of test` 与 package `FAIL`。这证明 `sync.Once` 只保护 provider，未保护闭包共享的 `Request`。
- 两份日志均通过 `ssh_download.py --no-progress` 下载。cleanup 使用：

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; rm -rf /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e; test ! -e /tmp/sub2api-race-red-714d12f8d8004666ac19f8e95fea6b1e; echo cleanup=ok" --timeout 120
```

- JSON `success=true, exit_code=0`；local archive 删除后不存在，下载日志保留。

### 最小修复、source commit 与 race GREEN

- `backend/internal/securityaudit/coordinator.go` 是唯一源码改动：两个 blocking goroutine 通过值参数各自取得独立 `Request`；`lazyLegacyEngine` 与 `Coordinator.CheckLazy` 注释明确 provider 只能在当前同步调用返回前求值且不得保留，Async 在 clone 入队前冻结。未修改测试或新增抽象。

```text
go test ./internal/securityaudit -run '^TestCoordinatorCheckLazyEvaluatesBodyOnceAcrossPromptAndLegacy$' -count=1 -v
go test ./internal/securityaudit -count=1
golangci-lint run ./internal/securityaudit/...
```

- 在 `backend/` 执行，三条均 exit `0`；lint 为 `0 issues.`。source commit 为 `417bbcc6a44c35b3e3ed16efb0bb86a4717401c9`（`fix: isolate blocking audit requests`），提交仅包含 `coordinator.go`，未 amend。
- stage `race-green`；nonce `3fef3c36ca644830b8d76e740b6a3e8e`；唯一 remote 目录 `/tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e`。

```text
git archive --format=tar --output="C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e.tar" 417bbcc6a44c35b3e3ed16efb0bb86a4717401c9
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; test ! -e /tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e; mkdir -m 700 /tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e; go version; gcc --version | head -n 1; df -hT / /tmp; echo preflight=ok" --timeout 120
$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_upload.py local-serv-ai "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e.tar" "/tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e/source.tar" --no-progress
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e; sha256sum source.tar; mkdir src; tar -xf source.tar -C src; rm -f source.tar; rm -rf src/backend/.test-tmp; mkdir -p src/backend/.test-tmp; echo setup=ok" --timeout 300
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e/src/backend; CGO_ENABLED=1 CC=gcc TMPDIR='/tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e/src/backend/.test-tmp' TMP='/tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e/src/backend/.test-tmp' TEMP='/tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e/src/backend/.test-tmp' go test -race ./internal/securityaudit -run '^TestCoordinatorCheckLazyEvaluatesBodyOnceAcrossPromptAndLegacy$' -count=10 > '/tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e/race-green.log' 2>&1" --timeout 900
```

- 五条均 exit `0`；archive size `53,780,480` bytes，local/remote SHA-256 均为 `3fc7a336e443034f458356131f732b17fe687f6dc7bc2b5e0201250655996f57`。GREEN log 通过 `ssh_download.py` 下载，size `63` bytes，SHA-256 `7a84320668e3fab9a53ddf4d3d5b74ac1caf4b7813aff5512b4f775383a9b50f`；package `ok`，`DATA RACE` count `0`。

```text
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; rm -rf /tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e; test ! -e /tmp/sub2api-race-green-3fef3c36ca644830b8d76e740b6a3e8e; echo cleanup=ok" --timeout 120
```

- cleanup exit `0`；local archive 不存在，GREEN log 保留。

### 最终本地门禁

```text
make test
```

- 在 source `417bbcc6a44c35b3e3ed16efb0bb86a4717401c9` 上原样只执行一次，exit `0`，未重跑。default handler/service 为 `61.929s`/`105.348s`，unit handler/service 为 `96.644s`/`158.958s`，两轮 securityaudit package 均通过；`golangci-lint run ./...` 为 `0 issues.`；frontend lint/typecheck 通过，Vitest 为 `215/215 files`、`1626/1626 tests`、duration `73.51s`。完整输出位于 `C:/Users/caiqy/.local/share/opencode/tool-output/tool_fb15b8185001iS8b5UJLS60oFN`。本次无需使用获准的 EOF/1013 例外。

```text
make "VERSION=0.1.165.1" "SHELL=D:/scoop/shims/bash.exe" build
```

- exit `0`；backend ldflags 含 `-X main.Version=0.1.165.1`，frontend Vite 处理 `1019` modules。VERSION 文件/二进制字符串均为 `0.1.165.1`；`go version -m backend/bin/server` 为 Go `1.26.5`、`vcs.revision=417bbcc6a44c35b3e3ed16efb0bb86a4717401c9`。`vcs.modified=true` 来自明确排除的 dirty reports/config。

detached generate 精确步骤：

```text
git worktree add --detach "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-task27-generate-417bbcc6a" 417bbcc6a44c35b3e3ed16efb0bb86a4717401c9
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
make -C backend generate
git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go
git status --short
git worktree remove "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-task27-generate-417bbcc6a"
```

- 全部 exit `0`；两轮 Ent/Wire diff 与 worktree status 均为空，移除后 `worktree_exists=False`，worktree 列表只剩主树。
- static：`git diff --check` exit `0`，仅有三个保留文件的 LF/CRLF warning；unmerged diff/index、tracked conflict marker、legacy first-token count 均为 `0`；VERSION `0.1.165.1`；14 个 protected migrations 全存在、missing `0`。`opencode.json` 的用户 CodeGraph/ddg-search diff原样保留且未暂存。

### 最终 remote integration

- stage `final-verify`；nonce `936ea72e3c4140ca930f88d07e0f34a5`；唯一 remote 目录 `/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5`。

```text
git archive --format=tar --output="C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5.tar" 417bbcc6a44c35b3e3ed16efb0bb86a4717401c9
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; test ! -e /tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5; mkdir -m 700 /tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5; go version; gcc --version | head -n 1; docker version; docker info >/dev/null; df -hT / /tmp /var/tmp; df -i / /tmp /var/tmp; echo preflight=ok" --timeout 120
$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_upload.py local-serv-ai "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5.tar" "/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/source.tar" --no-progress
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5; sha256sum source.tar; mkdir src; tar -xf source.tar -C src; rm -f source.tar; rm -rf src/backend/.test-tmp; mkdir -p src/backend/.test-tmp; test -d src/backend/.test-tmp; echo setup=ok" --timeout 300
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; cd /tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/src/backend; CI=true GOFLAGS='-v' TMPDIR='/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/src/backend/.test-tmp' TMP='/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/src/backend/.test-tmp' TEMP='/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/src/backend/.test-tmp' go test -tags=integration ./... > '/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/integration.log' 2>&1" --timeout 1800
$env:MSYS_NO_PATHCONV = "1"; python ~/.claude/skills/ssh-skill/scripts/ssh_download.py local-serv-ai "/tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5/integration.log" "C:/Users/caiqy/AppData/Local/Temp/opencode/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5-integration.log" --no-progress
python ~/.claude/skills/ssh-skill/scripts/ssh_execute.py local-serv-ai "set -eu; df -hT / /tmp /var/tmp; df -i / /tmp /var/tmp; rm -rf /tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5; test ! -e /tmp/sub2api-final-verify-936ea72e3c4140ca930f88d07e0f34a5; echo cleanup=ok" --timeout 120
```

- 七条均 exit `0`；archive size `53,780,480` bytes，local/remote SHA-256 均为 `3fc7a336e443034f458356131f732b17fe687f6dc7bc2b5e0201250655996f57`。preflight 为 Go `1.26.5`、gcc `11.5.0`、Docker client/server `29.2.1`、`docker info` 成功；根卷起始可用 `12G`、使用率 `66%`、inode `2%`。
- integration 原样命令未加 `-p`、未重试，JSON `success=true, exit_code=0`。下载日志 size `4,295,249` bytes，SHA-256 `28458502f274803aadd1e427fb7dffbd42eabf4dd61317b4e4bcb0f6cc97f3e9`；`--- FAIL:` count `0`、package `FAIL` count `0`。
- `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` PASS（`4.50s`）；`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` PASS（`4.21s`）。源码仍以两个 `require.Len(..., 12)` 固定 upstream list/embedded FS，并覆盖双方 172/181、双 186、190 notx、filename identity、重复 apply count/checksum，形成 migration `2/2` 与 `12/12` 证据。
- 13 skips（12 top-level + 1 nested）分类不变：DingTalk sentinel、外部 TLS capture、既有 CurrentConcurrency TODO、3 个 prompt-audit Redis fixture、6 个 prompt-audit PostgreSQL fixture、外部 OpenAI token comparison；均不命中 required targets/本轮受影响面。
- cleanup JSON `success=true, exit_code=0`；测试后根卷可用 `11G`、使用率 `70%`、inode `2%`。remote 目录和 local tar 均不存在，integration log 保留。未 Docker prune、未构建 Sub2API 镜像、未部署或访问服务/生产数据。

## 未执行项与残余风险

- Task 28 后续已完成完整 ancestry、merge parent、范围边界与 migration 静态复核，见文末 Task 28 专节。
- 未执行 Task 29 Chrome DevTools 前端烟测、能力终审、OpenSpec validate/tasks/progress/comet 收口。
- 未 push、tag、release、deploy、构建 Sub2API 镜像或触发 workflow。
- Task 25 遗留：Grok 在 request build 前设置 attempted 标志；client-disconnect drain lifecycle regression 使用固定 `50ms` 排序延迟。当前命名矩阵和完整门禁均通过，但这两项仍是非阻断测试/诊断风险。
- OAuth Images HeapAlloc 有历史波动；后续统一 lazy audit 修复在生产形态 RED/GREEN、Images 40/40 和完整 unit handler 中均通过。该测试仍使用 process-wide HeapAlloc，因此保留历史敏感性说明，不把先前失败抹除。
- Windows 当前仍为 `CGO_ENABLED=0` 且无 gcc，但本轮已在 Linux Go `1.26.5` + gcc `11.5.0` 上取得 source `90b008901` 的 DATA RACE RED，并在最终 source `417bbcc6a` 上完成同一 target `-count=10` race GREEN。
- 流程 concern：所有历史 `make test` 结果均已保留；source `90b008901` 的原样 exit `2` 仍只按用户批准的 EOF/1013 基线例外接受且未称 PASS。最终 source `417bbcc6a` 的原样 `make test` 是另一次唯一执行并真实 exit `0`；首次 SSH preflight 本机解析失败、remote ENOSPC、race `-count=1` 未命中和 `-count=100` DATA RACE exit `1` 均保持真实历史。
- gcc 安装由用户明确授权，但 dnf 自动依赖求解除新增 12 个包外还升级了 6 个 glibc/libgcc/libgomp 包；未自动卸载。远程共享根卷历史曾仅余 `1.3G`，本轮 integration cleanup 时可用 `11G`、使用率 `70%`；长期仍有 GOCACHE 增长风险，后续门禁应先观察容量。

## Task 28 拓扑、范围边界与 migration 静态复核（OpenSpec 8.3）

### 结论与范围

- 状态：`DONE_WITH_CONCERNS`。验证基线及 Task 28 起始 HEAD 为 `de3fffdd76f79831b4503ebfc204b0dc4cd156e7`，分支为 `feature/20260726/staged-merge-upstream-v0-1-165`。
- 六个固定 tag commit 均为 HEAD 祖先；指定 first-parent 范围恰有六个 merge，顺序、merge SHA 与第二父全部匹配 v0.1.160 至 v0.1.165 固定表。
- `upstream/main` 不是 HEAD 祖先，命令真实 exit `1`；这是未合入 release 后上游主线的预期范围边界 PASS，不是命令成功或 exit `0`。
- VERSION 为 `0.1.165.1`，unmerged 路径为 0，真实 tracked conflict marker 为 0，14 个 protected migration 全部存在。
- 本节仅完成 Task 28/OpenSpec 8.3 的只读拓扑与静态复核；没有重跑 remote integration，没有修改源码、migration、OpenSpec tasks/progress 或 plan。Task 29 与 OpenSpec 8.4 仍未完成。

### 步骤 1：祖先、first-parent merge 与范围边界

原样执行：

```powershell
git merge-base --is-ancestor 8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62 HEAD
git merge-base --is-ancestor 19149ca196eeae4a4482e5299dc6fa4ba0b06c8c HEAD
git merge-base --is-ancestor 27f094e0960ebd8e52de7ff7e763c6fec2ff4057 HEAD
git merge-base --is-ancestor d0bdd7e771636a8d315f542cafd39484f39bd60c HEAD
git merge-base --is-ancestor cd8bb98c44303b2c8f04c0da340447c992f0cb7d HEAD
git merge-base --is-ancestor e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29 HEAD
git log --first-parent --merges --format='%H %P %s' '075abc07399d6154130d2a2695fb24c785acd69c..HEAD'
git merge-base --is-ancestor upstream/main HEAD
```

- 前六条 ancestry 命令依次均 exit `0`。
- first-parent 命令 exit `0`，输出由新到旧精确为六条；反向即实际 v0.1.160 到 v0.1.165 的合入顺序：

```text
dc3df2d573f3e0601226075caed8c6e7ba85718a 34702ad029eaa1a11a2145efbd7fcf3485ab991a e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29 merge: upstream v0.1.165
6994599211d3714e30b67cc61ef0834a94c34610 07167bbfa44ecd702cf32268ad98eabb0dbb6c65 cd8bb98c44303b2c8f04c0da340447c992f0cb7d merge: upstream v0.1.164
02abe1574bf8044a1b180e62b002f58f9928d88f b7b7bba6952460bb7cc38f1d41a0de95c449bcb8 d0bdd7e771636a8d315f542cafd39484f39bd60c merge: upstream v0.1.163
8bda73544d6e26a323f101e5c68981634f0375ab 940c5cfcf390ecbfd2e041fb2b46c99846e6ea3e 27f094e0960ebd8e52de7ff7e763c6fec2ff4057 merge: upstream v0.1.162
f2158292c7ff3de4caa7ec22f9b7148400948f08 3fc60752acc459ecc37cd50b40df4a1f84ce3b62 19149ca196eeae4a4482e5299dc6fa4ba0b06c8c merge: upstream v0.1.161
e04cb1aa2c2554a04bec55f9b4393d3efd2eb693 d3e0c596ebff2298d07a3f4f336c16aa653cb840 8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62 merge: upstream v0.1.160
```

- 因而六个固定对应关系逐项成立：`e04cb1aa...` -> `8bfbc5ca...`、`f2158292...` -> `19149ca1...`、`8bda7354...` -> `27f094e0...`、`02abe157...` -> `d0bdd7e7...`、`69945992...` -> `cd8bb98c...`、`dc3df2d5...` -> `e9a58c1c...`。
- 最后一条 `git merge-base --is-ancestor upstream/main HEAD` 真实 exit `1`、无输出，按 brief 作为范围边界 PASS 保留。

### 步骤 2：版本、冲突与 protected migrations

原样执行：

```powershell
Get-Content backend/cmd/server/VERSION
git diff --check
git diff --name-only --diff-filter=U
$markers = @(git grep -n -E '^(<<<<<<< .+|=======|>>>>>>> .+)$' -- . ':!docs/superpowers/reports/**')
if ($LASTEXITCODE -notin @(0, 1)) { throw 'conflict marker scan failed' }
if ($markers.Count -ne 0) { $markers; throw 'conflict markers remain' }
$migrationMatches = @(git ls-tree -r --name-only HEAD backend/migrations | Select-String -Pattern '172_composite_model_routes.sql|172_video_per_second_billing_metadata.sql|181_prompt_audit.sql|181_group_duplicate_operation_id.sql|182_prompt_audit_full_prompt.sql|183_ops_ingress_reject_aggregates.sql|184_auth_cache_invalidation_outbox.sql|185_group_reasoning_effort_policy.sql|186_alipay_mobile_precreate_deep_link.sql|186_group_auth_cache_image_generation.sql|187_add_usage_log_session_id.sql|188_allow_live_usage_request_type.sql|189_add_group_allow_live.sql|190_add_users_email_alias_dedup_index_notx.sql')
if ($migrationMatches.Count -ne 14) { $migrationMatches; throw "expected 14 protected migrations, got $($migrationMatches.Count)" }
```

- `Get-Content` exit `0`，唯一版本内容为 `0.1.165.1`。
- `git diff --check` exit `0`，无 whitespace error；它只为禁止修改/暂存的 `.superpowers/sdd/task-27-report.md` 与 `.superpowers/sdd/task-4-report.md` 打印工作树 LF/CRLF warning。
- `git diff --name-only --diff-filter=U` exit `0`，没有 unmerged 路径；它打印相同的两个保留文件 LF/CRLF warning。
- marker 扫描中的原始 `git grep` exit `1`，表示零匹配；允许退出码 guard 与 count guard 均正常完成、未 throw，`$markers.Count=0`，整个 marker block exit `0`。
- protected migration 管道中的 `git ls-tree` exit `0`，count guard 正常完成、未 throw，`$migrationMatches.Count=14`，整个 migration block exit `0`。完整集合为：

```text
172_composite_model_routes.sql
172_video_per_second_billing_metadata.sql
181_group_duplicate_operation_id.sql
181_prompt_audit.sql
182_prompt_audit_full_prompt.sql
183_ops_ingress_reject_aggregates.sql
184_auth_cache_invalidation_outbox.sql
185_group_reasoning_effort_policy.sql
186_alipay_mobile_precreate_deep_link.sql
186_group_auth_cache_image_generation.sql
187_add_usage_log_session_id.sql
188_allow_live_usage_request_type.sql
189_add_group_allow_live.sql
190_add_users_email_alias_dedup_index_notx.sql
```

### Migration 执行语义与既有 integration 证据

- [`migrations_runner.go`](../../../backend/internal/repository/migrations_runner.go) 按完整 filename 排序，并以完整 filename 作为 `schema_migrations` 主键；已执行项按完整 filename 查询并校验 SHA-256 checksum。因此同号的本地/上游 172、本地/上游 181 和两个 186 都是独立 migration，不按数字前缀折叠。
- runner 对普通 migration 使用事务；对 `*_notx.sql` 先校验只包含幂等的 `CREATE/DROP INDEX CONCURRENTLY`，再逐语句非事务执行并记录。因此 `190_add_users_email_alias_dedup_index_notx.sql` 走明确的 notx 路径。
- Task 24 提交的 [`migrations_schema_integration_test.go`](../../../backend/internal/repository/migrations_schema_integration_test.go) 提供两条互补路径：`TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate` 创建全新隔离库、应用全量 migration 后再次 apply；`TestMigrationsRunner_UpgradesLocalV01596AcrossUpstreamStages` 先用过滤 FS 建立仅含本地历史的升级库，再应用完整 embedded FS。
- 升级测试固定列出 12 个上游完整 filename，以两个 `require.Len(..., 12)` 约束列表和 embedded FS，明确断言双方 172/181、双 186、`190_*_notx.sql` 已记录；第二次 apply 后 record count 不增加且 checksum map 不变，覆盖完整 filename 隔离、幂等与 checksum 保持。
- [既有最终 remote integration 证据](#最终-remote-integration)记录两条 target 分别 PASS（`4.50s`、`4.21s`），migration `2/2`、上游集合 `12/12`；Task 28 只核读 committed test、runner 和既有 formal evidence，没有重跑远程 integration。

### Concern 与后续边界

- 唯一 Task 28 concern 是两个协调器保留 report 的 LF/CRLF warning；相关命令均 exit `0`，无 whitespace error、unmerged 路径或 conflict marker。所有保留改动均未修改或暂存。
- Task 29 浏览器烟测、能力终审以及 OpenSpec 8.4/tasks/progress/comet 收口不属于 Task 28，仍未执行。
