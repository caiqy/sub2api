# restore-platform-sticky-routing 验证报告

## 结论

本 change 验证通过。OpenSpec 任务 7/7 完成，4 个 delta spec 场景及详细设计增量均有实现和回归覆盖；最终代码审查无 Critical、Important 或 Minor 问题。

| 维度 | 结果 |
|---|---|
| Completeness | PASS：任务、需求与场景均已覆盖 |
| Correctness | PASS：平台开关、共享 resolver、fallback cleanup 与 hydration release 符合设计 |
| Coherence | PASS：OpenSpec、Design Doc、实现和测试一致 |
| Security | PASS：无新增依赖、公开 API、配置、schema、密钥或 unsafe 操作 |

## 场景映射

- OpenAI HTTP、WS V2 与 WS ingress 在 Sticky disabled 时绕过 response-account、response-connection、turn state 和 session-connection 状态。
- Gemini Messages compat 按最终 Gemini、Anthropic 或 Antigravity 平台开关实现 cache 零读写；nil/未知平台保持默认 enabled。
- `GatewayService.ResolveGatewayGroup` 统一 handler 与 scheduler 的 Claude Code fallback、循环检测和强制平台语义。
- resolved group/platform 驱动 session key、调度、绑定及 mixed Antigravity smart-retry cleanup；覆盖 initial fallback、invalid-request 二次 fallback、signature/budget rectifier 和 Anthropic Sticky enabled/disabled。
- 无 `ConcurrencyService` 时保持模型路由候选集合；OpenAI hydration 失败仅释放一次槽位。

## 验证证据

以下命令通过：

```powershell
go test ./internal/handler -count=1
go test ./internal/service -run '^(TestAntigravityGatewayService_ForwardBudgetRectifierClearsProvidedStickySession|TestAntigravityGatewayService.*Sticky|TestHandleSmartRetry_.*Sticky|TestGatewayService_.*(Claude|Fallback|Sticky)|TestGeminiMessagesCompatService|TestOpenAIGatewayService_StickyDisabled|TestGatewayServiceNewSelectionResult_ReleasesAcquiredSlotWhenHydrationFails|TestOpenAINewAcquiredSelectionResult_ReleasesSlotWhenHydrationFails)$' -count=1
go build ./...
openspec validate restore-platform-sticky-routing --strict
git diff --check
```

完整 `go test -tags=unit ./internal/service -count=1` 仍有 8 个已知失败，均属于后续批次，未命中本 change 的 Sticky、WS、compat、resolver、cleanup 或 hydration 行为：

- `TestHandleSmartRetry_QuotaExhausted_UsesCreditsAndStoresIndependentState`
- `TestResolvedPricingAsTokenMode_PreservesTokenPricingAndImageOutputPriceSecondCase`
- `TestResolvedPricingAsTokenMode_PreservesTokenPricingAndImageOutputPrice`
- `TestForwardAsRawChatCompletions_SilentRefusalTriggersFailover`
- `TestForwardAsChatCompletions_UnknownResponsesSupportFallbackUsesVersionedChatURL`
- `TestParseGrokMediaRequestBuildsMultipartModerationBody`
- `TestForwardGrokMediaImagesEditMultipartConvertsToJSON`
- `TestSettingService_GetAllSettings_OpenAIAdvancedSchedulerEffectiveValuesUseConfig`

## 已接受建议

`proposal.md` 的 Impact 文件列表未穷举 build 阶段扩展出的 handler、WS、compat 和测试文件，但不与需求或行为契约矛盾，作为非阻塞文档建议留待归档或后续整理。

## 环境限制

- Docker 不可用，未运行容器集成测试。
- GCC 不可用，未运行 Go race 测试。
