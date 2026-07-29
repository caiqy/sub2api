# 分段合入上游 v0.1.165 最终自动验证报告

## 结论与边界

- 状态：`DONE_WITH_CONCERNS`。
- 分支：`feature/20260726/staged-merge-upstream-v0-1-165`。
- Task 27 起始 HEAD：`8c3b281f7f9e08a9f2d776f4241a922f7a85bff8`。
- 最终 source/test HEAD：`1cc41c72c1def83113263d9b631f9856dbff030d`。
- 最终版本：`0.1.165.1`。
- 本轮发现并提交两个修复：`aff04f9cd` 仅格式化两个 WebSocket ownership 路径；`1cc41c72c` 避免 content moderation 未配置时构造完整 Images moderation payload，并使 OAuth retention fixture 的 started 信号可重入。
- 聚焦矩阵、最终展开式本地测试门禁、版本化 Windows build、两轮 Ent/Wire generate、静态检查和远程 integration 在最终 source/test HEAD 上均有成功证据。首次 `make test` 的真实非零结果仍保留，不改写为 PASS。
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

## Ent/Wire 两轮生成

- 既有 ledger 已记录主树 `user-mapped section` 风险，因此未在主树手改生成物。
- 创建短 detached worktree `D:\w27`，HEAD 精确为 `1cc41c72c1def83113263d9b631f9856dbff030d`。
- 第一轮 `make -C backend generate`：exit `0`；随后 `git diff --exit-code -- backend/ent backend/cmd/server/wire_gen.go`：exit `0`、无输出。
- 第二轮相同两条命令：均 exit `0`、generated diff 无输出。
- detached worktree `git status --short` 无输出；`git worktree remove D:/w27` exit `0`；最终 `Test-Path D:\w27` 为 `False`。
- 未手改任何 Ent/Wire 生成物。

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

## 未执行项与残余风险

- 未执行 Task 28 完整 ancestry、merge parent 和范围边界验证；本轮静态 migration presence 不能替代 Task 28。
- 未执行 Task 29 Chrome DevTools 前端烟测、能力终审、OpenSpec validate/tasks/progress/comet 收口。
- 未 push、tag、release、deploy、构建 Sub2API 镜像或触发 workflow。
- Task 25 遗留：Grok 在 request build 前设置 attempted 标志；client-disconnect drain lifecycle regression 使用固定 `50ms` 排序延迟。当前命名矩阵和完整门禁均通过，但这两项仍是非阻断测试/诊断风险。
- OAuth Images HeapAlloc 有历史波动，本轮在完整 unit gate 再次真实复现并完成根因修复；修复后精确 10/10、moderation-enabled sibling、完整 unit handler 和最终展开式 gate 均通过。该测试仍使用 process-wide HeapAlloc，因此保留历史敏感性说明，不把先前失败抹除。
- Windows 当前 `CGO_ENABLED=0` 且无 `gcc`，未运行 `-race`；不声称 race-clean。
- 流程 concern：首次 `make test` exit `2` 已保留；formatter 修复后为定位新 OAuth RED 又执行了一次 `make test`，最终改用逐项组成命令完成完整验证。结论依赖展开式 final-HEAD gate，而不是将首次 `make test` 改写为成功。
