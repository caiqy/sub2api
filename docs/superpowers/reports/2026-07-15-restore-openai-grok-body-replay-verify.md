# restore-openai-grok-body-replay 验证报告

## 结论

本 Hotfix 验证通过。OpenSpec 任务 3/3 完成，`request-body-retention-control` 的 7 个场景与实现一致；最终 OpenSpec/code reviewer 无 Critical、Warning 或 Suggestion。

| 维度 | 结果 |
|---|---|
| Completeness | PASS：任务、requirement 与场景均有实现证据 |
| Correctness | PASS：OpenAI raw fallback 与 Grok multipart body 内容保持一致 |
| Coherence | PASS：proposal、design、delta spec、实现和测试一致 |
| Security | PASS：无公开 API、配置、schema、依赖或密钥变更 |

## 修复证据

- OpenAI raw chat 在解析 model 前通过 bound `RequestBodyHandle` 恢复 body；Responses unsupported fallback 的 URL、model 与发送内容保持一致。
- silent-refusal 仅修正测试 fixture 的 canonical HTTP header，不改变生产行为。
- Grok raw multipart 上传 bytes 可用于 moderation 与 images edit data URL；测试解码并精确比较原始 bytes。
- raw multipart 使用 `limit+1` 检测并对 `20MB+1` 返回 `MaxBytesError`。
- metadata 提取后清空 uploads/mask 的 `Data`，不让二进制副本跨越上游网络等待；usage metadata 保持不变。
- `FileHeader` 存在时继续优先读取文件，标准 handler multipart 路径未回归。

## 验证命令

以下命令通过：

```powershell
go test -tags=unit ./internal/service -run '^(TestForwardAsRawChatCompletions_SilentRefusalTriggersFailover|TestForwardAsChatCompletions_UnknownResponsesSupportFallbackUsesVersionedChatURL)$' -count=1
go test -tags=unit ./internal/service -run '^(Test.*GrokMedia|TestForwardGrokMedia.*|TestOpenAIImagesRequestModerationBody_MultipartEditIncludesUploadsFromForm|TestOpenAIGatewayServiceParseOpenAIImagesRequest_MultipartEdit)$' -count=1
go test ./internal/handler -count=1
go build ./...
openspec validate restore-openai-grok-body-replay --strict
git diff --check
```

完整 `go test -tags=unit ./internal/service -count=1` 中，本 change 修复前的 OpenAI raw chat 2 项与 Grok multipart 2 项失败均已消失；仍有 4 个后续批次失败：

- `TestHandleSmartRetry_QuotaExhausted_UsesCreditsAndStoresIndependentState`
- `TestResolvedPricingAsTokenMode_PreservesTokenPricingAndImageOutputPriceSecondCase`
- `TestResolvedPricingAsTokenMode_PreservesTokenPricingAndImageOutputPrice`
- `TestSettingService_GetAllSettings_OpenAIAdvancedSchedulerEffectiveValuesUseConfig`

## 环境限制

- Docker 不可用，未运行容器集成测试。
- GCC 不可用，未运行 Go race 测试。
