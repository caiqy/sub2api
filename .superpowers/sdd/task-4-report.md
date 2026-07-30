# Task 4 Report

## RED
- 命令：`go test ./internal/handler -run '^TestClientRequestedUsageFieldsPreservesConcreteCompositeRouteWithoutChannelMapping$' -count=1`
- 结果：失败，`ChannelMappedModel` 实际为 `public-alias`，与期望的 `gpt-5` 不一致。

## GREEN
- 修复：删除 `clientRequestedUsageFields` 里 `!mapping.Mapped` 时把 `ChannelMappedModel` 覆盖成 `requestedModel` 的逻辑。
- 复测：同一条定向测试通过。

## Package
- `go test ./internal/handler -count=1`：通过。
- `gofmt -w internal/handler/content_moderation_helper.go internal/handler/composite_platform_test.go`
- `git diff --check`：仅提示既有工作区文件的 LF/CRLF 警告，无本次补丁格式错误。

## 变更
- `backend/internal/handler/composite_platform_test.go`：拆出新测试 `TestClientRequestedUsageFieldsPreservesConcreteCompositeRouteWithoutChannelMapping`，断言 `public-alias → gpt-5 → gpt-5.2`。
- `backend/internal/handler/content_moderation_helper.go`：只保留 `ToUsageFields` 的原始路由结果，不再在未映射分支覆盖 `ChannelMappedModel`。

## Concerns
- 工作区里还有既有未提交变更和未跟踪文件，不在本次范围内。

## Fresh Fix Attempt
- 按 reviewer 备注尝试在 `TestClientRequestedUsageFieldsPreservesConcreteCompositeRouteWithoutChannelMapping` 中补 `fields.UpstreamModel == "gpt-5.2"` 的直接断言。
- 结果：`service.ChannelUsageFields` 当前没有 `UpstreamModel` 字段，`go test` 会在该断言处编译失败；在“不改生产代码”的约束下无法真实补上这条断言。
- 继续保留现有可编译断言：`OriginalModel`、`ChannelMappedModel`、`ModelMappingChain`。

## Fresh Fixer Round 2

### Root Cause 与用户裁决
- `ChannelMappedModel` 同时用于审计和默认计费；前一轮为保留 concrete route model 修复审计后，无 channel mapping 的 composite public alias 仍带默认 `BillingModelSource=channel_mapped`，使默认计费直接读取 concrete model，跳过 public alias 显式价。
- 按用户裁决不新增 `ChannelUsageFields` 字段：仅当 composite public alias 覆盖 `OriginalModel`、`!mapping.Mapped` 且原 source 为 `channel_mapped` 时，将 source 改为既有 `requested`。真实 channel mapping、显式 `requested/upstream` 和普通非 composite identity 均保持原行为。

### RED / GREEN
- Handler RED：`go test ./internal/handler -run '^TestClientRequestedUsageFieldsPreservesConcreteCompositeRouteWithoutChannelMapping$' -count=1` 按预期失败，source 实际为 `channel_mapped`、期望为 `requested`。
- 最小 GREEN：在 `clientRequestedUsageFields` 的 public alias 覆盖分支中加入 `!mapping.Mapped && fields.BillingModelSource == BillingModelSourceChannelMapped` 条件改写；同一测试及 Task 4 focused handler tests 通过。
- 保护证据：mapped composite 保持 `channel_mapped`；无映射的显式 `requested/upstream` source 不变；非 composite identity 与 `mapping.ToUsageFields` 相同。

### Pricing 证据
- OpenAI：将 `TestOpenAIGatewayServiceRecordUsage_CompositePublicAliasPricing` 的两个 requested-source 场景改为真实形态 `OriginalModel=team/gpt-5.2`、`ChannelMappedModel=gpt-5.4`。测试无需生产计费改动即 GREEN：alias 有显式 channel price 时按 alias；alias 无显式价时回退 concrete。
- Generic Gateway：现有测试只有 helper 级 composite fallback，没有 `RecordUsage` runtime 覆盖；新增最小 unit runtime 测试，验证 requested-source + alias/concrete 字段形态按 concrete fallback 计费，`go test -tags=unit ./internal/service -run '^TestGatewayServiceRecordUsage_CompositeRequestedAliasFallsBackToConcretePricing$' -count=1` 通过。

### 验证
- Task 4 focused handler tests：通过。
- OpenAI composite pricing focused test：通过。
- Generic Gateway composite pricing focused unit test：通过。
- `go test ./internal/handler -count=1`：通过。
- `go test ./internal/service -count=1`：通过。
- `gofmt`：已运行于 5 个实际变更 Go 文件。
- `git diff --check`：退出成功，仅提示两个既有未提交文档的 LF/CRLF 警告。

### 变更与自审
- 变更文件：`backend/internal/handler/content_moderation_helper.go`、`backend/internal/handler/composite_platform_test.go`、`backend/internal/service/channel.go`、`backend/internal/service/openai_gateway_record_usage_test.go`、`backend/internal/service/gateway_record_usage_test.go`。
- 未改 `ChannelUsageFields` 结构、审计字段、mapping chain、service billing production、依赖或工作流状态。
- 工作树中的既有 OpenSpec/Comet/Paseo 文件与本报告均不纳入提交。

## Fresh Fixer Round 3

### Generic Gateway Pricing Test Discriminability
- 模型选择：将 concrete model 从 `claude-sonnet-4` 改为仓库已有定价模型 `claude-opus-4-6`。`billing_service.go` 将其解析为 Claude Opus 4.6 fallback（输入 `$5/MTok`、输出 `$25/MTok`）；`team/claude` 保持 Claude/Sonnet family fallback（输入 `$3/MTok`、输出 `$15/MTok`）。
- 成本差异：对 20 input / 10 output tokens，测试先独立计算 alias cost `$0.00021` 与 concrete cost `$0.00035`，并以 `require.NotEqual` 确认两者不同，随后断言 `RecordUsage` 的 usage log 和用户扣费均等于 concrete cost。

### Mutation RED / Restore / GREEN
- RED：临时将 `selectCompositeBillableModel` 的 fallback 返回值从 `concreteBillingModel` 改为 `billingModel`，使 generic composite billing 保留 alias。运行 `go test -tags=unit ./internal/service -run '^TestGatewayServiceRecordUsage_CompositeRequestedAliasFallsBackToConcretePricing$' -count=1` 失败：期望 `$0.00035`，实际 `$0.00021`，差额 `$0.00014`。
- Restore：立即恢复 `backend/internal/service/gateway_usage_billing.go`；`git diff --exit-code -- backend/internal/service/gateway_usage_billing.go` 退出成功，确认 production 文件零 diff。
- GREEN：恢复后同一 Generic focused test 通过；`go test ./internal/service -run '^TestOpenAIGatewayServiceRecordUsage_CompositePublicAliasPricing$' -count=1` 通过。
