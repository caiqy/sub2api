---
change: optimize-large-input-memory
design-doc: docs/superpowers/specs/2026-07-09-large-input-memory-design.md
base-ref: 40b807f114d1fe2e02ccc118fc4e6bd75417e4e5
---

# 大输入内存优化实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 降低合法大 `/v1/responses` 请求在内容审计抽取和 OpenAI usage 异步记录中的额外内存放大。

**架构：** 内容审计只新增一个包内 collector，用收集期上限替换无界 `[]string`/`[]string` 累积。OpenAI usage 不重写计费链路，只在提交 worker 前构造轻量快照，让闭包捕获快照而不是请求上下文、请求体或可携带大对象的 `result`。

**技术栈：** Go 1.25，Gin，现有 `gjson`、`testify/require`，不新增依赖。

## 全局约束

- 不做流式 JSON 请求体解析。
- 不引入对象池或新第三方依赖。
- 不新增后台大 input 策略配置。
- 不调整部署默认值。
- 不修改 `openspec/changes/optimize-large-input-memory/tasks.md`。
- 不提交 git；执行者完成后只保留工作区改动和测试结果。
- 最小有效实现：优先改现有文件，不拆新包，不新增接口层抽象。

## 文件结构

- Modify: `backend/internal/service/content_moderation_input.go`：新增包内 collector，替换审计文本和图片收集路径。
- Modify: `backend/internal/service/content_moderation_input_test.go`：新增大文本保留最新内容、inline/base64 图片限量测试。
- Modify: `backend/internal/handler/openai_gateway_handler.go`：新增 OpenAI usage 快照构造函数，并在 Responses/WebSocket usage 提交前使用。
- Modify: `backend/internal/handler/openai_gateway_usage_context_test.go`：新增快照不保留原 `OpenAIForwardResult`/detail snapshot 指针的单测。
- Modify: `backend/internal/service/openai_gateway_record_usage_test.go`：补一条 RecordUsage 回归，确认快照输入的计费、图片、渠道字段仍落库。

---

### Task 1: 内容审计大输入测试

**Files:**
- Modify: `backend/internal/service/content_moderation_input_test.go`

**Interfaces:**
- Consumes: `ExtractContentModerationInput(protocol string, body []byte) ContentModerationInput`
- Produces: 两个先失败的回归测试，约束 Task 2 的 collector 行为。

- [x] **Step 1: 增加测试 imports**

在 `backend/internal/service/content_moderation_input_test.go` 的 import 中加入：

```go
import (
    "fmt"
    "strings"
    "testing"

    "github.com/stretchr/testify/require"
)
```

- [x] **Step 2: 写大文本只保留最新内容测试**

在文件末尾增加：

```go
func TestExtractContentModerationInput_ResponsesLargeInputKeepsLatestText(t *testing.T) {
    oldText := strings.Repeat("旧内容 ", maxModerationInputRunes+100)
    latest := "最新风险片段"
    body := []byte(fmt.Sprintf(`{
        "input":[
            {"type":"message","role":"user","content":[{"type":"input_text","text":%q}]},
            {"type":"message","role":"user","content":[{"type":"input_text","text":%q}]}
        ]
    }`, oldText, latest))

    input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)

    require.LessOrEqual(t, len([]rune(input.Text)), maxModerationInputRunes)
    require.Contains(t, input.Text, latest)
}
```

- [x] **Step 3: 写图片收集期限量测试**

在同文件末尾增加：

```go
func TestExtractContentModerationInput_ResponsesInlineImagesLimitedDuringCollection(t *testing.T) {
    first := strings.Repeat("a", 1024)
    second := strings.Repeat("b", 1024)
    body := []byte(fmt.Sprintf(`{
        "input":[{"type":"message","role":"user","content":[
            {"type":"input_image","mime_type":"image/png","data":%q},
            {"type":"input_image","mime_type":"image/png","data":%q}
        ]}]
    }`, first, second))

    input := ExtractContentModerationInput(ContentModerationProtocolOpenAIResponses, body)

    require.Len(t, input.Images, maxContentModerationInputImages)
    require.Equal(t, "data:image/png;base64,"+first, input.Images[0])
    require.NotContains(t, strings.Join(input.Images, "\n"), second)
}
```

- [x] **Step 4: 运行测试并确认失败**

Run: `cd backend && go test ./internal/service -run 'TestExtractContentModerationInput_Responses(LargeInputKeepsLatestText|InlineImagesLimitedDuringCollection)' -count=1`

Expected: FAIL；第一条因文本超过 `maxModerationInputRunes` 或不稳定保留最新片段失败，第二条因当前收集阶段会保留多张图片失败。

### Task 2: 内容审计 collector 实现

**Files:**
- Modify: `backend/internal/service/content_moderation_input.go`

**Interfaces:**
- Consumes: Task 1 的两个测试。
- Produces: `contentModerationInputCollector`，方法 `AddText(string)`、`AddImageURL(string)`、`AddImageData(string, string)`、`Result() ContentModerationInput`。

- [x] **Step 1: 新增 collector 类型**

在 `moderationInputCandidate` 位置附近替换为包内 collector：

```go
type contentModerationInputCollector struct {
    text   string
    images []string
}

func (c *contentModerationInputCollector) AddText(text string) {
    text = strings.TrimSpace(text)
    if text == "" || strings.Contains(text, "<system-reminder>") {
        return
    }
    if c.text != "" {
        text = c.text + "\n" + text
    }
    c.text = trimLatestRunes(normalizeContentModerationText(text), maxModerationInputRunes)
}

func (c *contentModerationInputCollector) AddImageURL(image string) {
    if len(c.images) >= maxContentModerationInputImages {
        return
    }
    image = strings.TrimSpace(image)
    if image == "" {
        return
    }
    if strings.HasPrefix(image, "data:") || strings.HasPrefix(image, "http://") || strings.HasPrefix(image, "https://") {
        c.images = append(c.images, image)
    }
}

func (c *contentModerationInputCollector) AddImageData(mimeType, data string) {
    if len(c.images) >= maxContentModerationInputImages {
        return
    }
    mimeType = strings.TrimSpace(mimeType)
    data = strings.TrimSpace(data)
    if mimeType == "" || data == "" {
        return
    }
    c.AddImageURL(fmt.Sprintf("data:%s;base64,%s", mimeType, data))
}

func (c *contentModerationInputCollector) Result() ContentModerationInput {
    return ContentModerationInput{Text: c.text, Images: normalizeModerationImages(c.images)}
}
```

- [x] **Step 2: 替换入口收集变量**

把 `ExtractContentModerationInput` 中的 `parts/images` 改为单个 collector，并让所有分支调用 collector 版本：

```go
var collector contentModerationInputCollector
collectAllResponsesInput(gjson.GetBytes(body, "input"), &collector)
return collector.Result()
```

- [x] **Step 3: 改收集函数签名**

把下列函数的 `parts *[]string, images *[]string` 参数改为 `collector *contentModerationInputCollector`：

```go
collectAllOpenAIChatMessages
collectAnthropicUserContentValue
collectAllAnthropicMessages
collectAllResponsesInput
collectAllGeminiContents
collectContentValue
addGeminiModerationImage
```

把只收文本的 `collectAnthropicAssistantTextOnly(value gjson.Result, parts *[]string)` 改为 `collectAnthropicAssistantTextOnly(value gjson.Result, collector *contentModerationInputCollector)`。

- [x] **Step 4: 删除候选数组去重路径**

删除 `moderationInputCandidate` 和 `appendUniqueModerationCandidates`。各协议数组分支直接按原顺序调用 collector；重复文本交给最终窗口处理，避免 `strings.Join` 对超大候选再次制造副本。

- [x] **Step 5: 替换 add 函数调用**

把 `addModerationText(parts, value)` 改为 `collector.AddText(value)`，把 `addModerationImage(images, value)` 改为 `collector.AddImageURL(value)`，把 `addModerationImageData(images, mime, data)` 改为 `collector.AddImageData(mime, data)`。

- [x] **Step 6: 保留现有公开行为防线**

保留 `normalizeModerationImages` 和 `limitContentModerationImages`，让 `ContentModerationInput.Normalize()` 与 `ModerationInput()` 的既有防线不变。

- [x] **Step 7: 运行内容审计测试**

Run: `cd backend && go test ./internal/service -run 'TestExtractContentModerationInput|TestContentModerationInput' -count=1`

Expected: PASS。

### Task 3: OpenAI usage 快照测试

**Files:**
- Modify: `backend/internal/handler/openai_gateway_usage_context_test.go`
- Modify: `backend/internal/service/openai_gateway_record_usage_test.go`

**Interfaces:**
- Consumes: 现有 `service.OpenAIRecordUsageInput` 和 `service.OpenAIForwardResult`。
- Produces: 先失败测试，约束 Task 4 的快照函数不复用原 result/detail snapshot 指针且不改变计费字段。

- [x] **Step 1: 在 handler 测试中增加 service/http imports**

`backend/internal/handler/openai_gateway_usage_context_test.go` import 调整为：

```go
import (
    "context"
    "net/http"
    "testing"
    "time"

    "github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
    "github.com/Wei-Shaw/sub2api/internal/service"
    "github.com/stretchr/testify/require"
)
```

- [x] **Step 2: 写快照不保留原 result/detail 指针测试**

在 handler 测试文件末尾增加：

```go
func TestOpenAIUsageRecordInputSnapshotDropsLargeResultFields(t *testing.T) {
    tier := "priority"
    effort := "high"
    firstTokenMs := 123
    result := &service.OpenAIForwardResult{
        RequestID:       "req-1",
        ResponseID:      "resp-1",
        Model:           "gpt-5.1",
        BillingModel:    "gpt-5.1-billing",
        UpstreamModel:   "gpt-5.1-upstream",
        ServiceTier:     &tier,
        ReasoningEffort: &effort,
        Stream:          true,
        OpenAIWSMode:    true,
        ResponseHeaders: http.Header{"X-Large": []string{strings.Repeat("x", 1024)}},
        Duration:        2 * time.Second,
        FirstTokenMs:    &firstTokenMs,
        Usage:           service.OpenAIUsage{InputTokens: 10, OutputTokens: 5, ImageOutputTokens: 7},
        ImageCount:      1,
        ImageSize:       "1024x1024",
        ImageOutputSize: "1024x1024",
        ImageSizeSource: "request",
    }
    detail := &service.UsageLogDetailSnapshot{RequestBody: strings.Repeat("body", 128), ResponseBody: "ok"}

    snapshot := snapshotOpenAIUsageRecordInput(service.OpenAIRecordUsageInput{
        Result:          result,
        DetailSnapshot:  detail,
        ChannelUsageFields: service.ChannelUsageFields{ChannelID: 9, OriginalModel: "gpt-5", ChannelMappedModel: "gpt-5.1", BillingModelSource: service.BillingModelSourceChannelMapped, ModelMappingChain: "gpt-5->gpt-5.1"},
    })

    require.NotSame(t, result, snapshot.Result)
    require.Nil(t, snapshot.Result.ResponseHeaders)
    require.Equal(t, result.Usage, snapshot.Result.Usage)
    require.Equal(t, result.ImageOutputSize, snapshot.Result.ImageOutputSize)
    require.NotSame(t, detail, snapshot.DetailSnapshot)
    require.Equal(t, detail.ResponseBody, snapshot.DetailSnapshot.ResponseBody)
    require.Equal(t, int64(9), snapshot.ChannelID)
}
```

- [x] **Step 3: 在 service usage 测试中补渠道和图片回归**

在 `backend/internal/service/openai_gateway_record_usage_test.go` 增加一条测试，直接调用 `RecordUsage`，输入使用 `OpenAIRecordUsageInput` 的快照形态：

```go
func TestOpenAIGatewayServiceRecordUsage_PreservesSnapshotBillingFields(t *testing.T) {
    usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
    svc := newOpenAIRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)

    err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
        Result: &OpenAIForwardResult{
            RequestID:       "rid-snapshot",
            Model:           "gpt-5.1",
            UpstreamModel:   "gpt-5.1-upstream",
            Usage:           OpenAIUsage{InputTokens: 10, OutputTokens: 5, ImageOutputTokens: 7},
            ImageCount:      1,
            ImageOutputSize: "1024x1024",
            ImageSizeSource: "request",
        },
        APIKey: &APIKey{ID: 2, User: &User{ID: 1}},
        User:   &User{ID: 1},
        Account: &Account{ID: 3},
        ChannelUsageFields: ChannelUsageFields{ChannelID: 9, OriginalModel: "gpt-5", ChannelMappedModel: "gpt-5.1", BillingModelSource: BillingModelSourceChannelMapped, ModelMappingChain: "gpt-5->gpt-5.1"},
    })

    require.NoError(t, err)
    require.Equal(t, int64(9), *usageRepo.lastLog.ChannelID)
    require.Equal(t, "gpt-5", usageRepo.lastLog.RequestedModel)
    require.Equal(t, "gpt-5.1-upstream", *usageRepo.lastLog.UpstreamModel)
    require.Equal(t, 1, usageRepo.lastLog.ImageCount)
    require.Equal(t, 7, usageRepo.lastLog.ImageOutputTokens)
}
```

- [x] **Step 4: 运行测试并确认失败**

Run: `cd backend && go test ./internal/handler ./internal/service -run 'TestOpenAIUsageRecordInputSnapshotDropsLargeResultFields|TestOpenAIGatewayServiceRecordUsage_PreservesSnapshotBillingFields' -count=1`

Expected: FAIL；handler 测试因 `snapshotOpenAIUsageRecordInput` 未定义失败，service 回归应通过或暴露字段缺口。

### Task 4: OpenAI usage 快照实现

**Files:**
- Modify: `backend/internal/handler/openai_gateway_handler.go`

**Interfaces:**
- Consumes: Task 3 的 handler 测试。
- Produces: `snapshotOpenAIUsageRecordInput(input service.OpenAIRecordUsageInput) *service.OpenAIRecordUsageInput`，供 OpenAI Responses/WS usage submit 使用。

- [x] **Step 1: 新增快照 helper**

在 `submitOpenAIUsageRecordTask` 附近新增包内函数：

```go
func snapshotOpenAIUsageRecordInput(input service.OpenAIRecordUsageInput) *service.OpenAIRecordUsageInput {
    snapshot := input
    if input.Result != nil {
        result := *input.Result
        result.ResponseHeaders = nil
        result.ImageOutputSizes = append([]string(nil), input.Result.ImageOutputSizes...)
        if input.Result.ImageSizeBreakdown != nil {
            result.ImageSizeBreakdown = make(map[string]int, len(input.Result.ImageSizeBreakdown))
            for k, v := range input.Result.ImageSizeBreakdown {
                result.ImageSizeBreakdown[k] = v
            }
        }
        snapshot.Result = &result
    }
    snapshot.DetailSnapshot = input.DetailSnapshot.Normalize()
    return &snapshot
}
```

- [x] **Step 2: 在 Responses HTTP 成功路径使用快照**

查找 `openai_gateway_handler.go` 中调用 `h.submitOpenAIUsageRecordTask` 且闭包内直接构造 `&service.OpenAIRecordUsageInput{...}` 的 Responses HTTP 路径。把闭包前的局部变量改成：

```go
usageInput := snapshotOpenAIUsageRecordInput(service.OpenAIRecordUsageInput{
    Result:             result,
    APIKey:             apiKey,
    User:               apiKey.User,
    Account:            account,
    Subscription:       subscription,
    DetailSnapshot:     detailSnapshot,
    InboundEndpoint:    inboundEndpoint,
    UpstreamEndpoint:   upstreamEndpoint,
    UserAgent:          userAgent,
    IPAddress:          clientIP,
    RequestPayloadHash: requestPayloadHash,
    APIKeyService:      h.apiKeyService,
    QuotaPlatform:      quotaPlatform,
    ChannelUsageFields: channelMapping.ToUsageFields(reqModel, result.UpstreamModel),
    CyberBlocked:       cyberBlocked,
})
```

闭包中只捕获 `usageInput`、`reqLog` 和必要日志字段：

```go
requestID := usageInput.Result.RequestID
accountID := usageInput.Account.ID
h.submitOpenAIUsageRecordTask(ctx, usageInput.Result, func(taskCtx context.Context) {
    if err := h.gatewayService.RecordUsage(taskCtx, usageInput); err != nil {
        reqLog.Error("openai.responses_record_usage_failed", zap.Int64("account_id", accountID), zap.String("request_id", requestID), zap.Error(err))
    }
})
```

- [x] **Step 3: 在 WebSocket AfterTurn 使用快照**

将 `AfterTurn` 中现有 `h.submitOpenAIUsageRecordTask(ctx, result, func(...){ RecordUsage(&service.OpenAIRecordUsageInput{...}) })` 改为先构造 `usageInput := snapshotOpenAIUsageRecordInput(...)`，闭包只捕获 `usageInput` 和日志小字段。保留现有 `cyberBlocked`、`quotaPlatform`、`channelMappingWS.ToUsageFields(reqModel, result.UpstreamModel)` 语义。

- [x] **Step 4: 保持非目标路径不动**

不要改 `submitFailedUsageLog`、image/embeddings/chat completions 的 `RecordUsage` 调用；本 change 的风险来源是 `/v1/responses` 大 input 和 Responses WS turn。

- [x] **Step 5: 运行 OpenAI usage 相关测试**

Run: `cd backend && go test ./internal/handler ./internal/service -run 'TestOpenAI.*Usage|TestOpenAIGatewayServiceRecordUsage|TestSubmitUsageRecordTask' -count=1`

Expected: PASS。

### Task 5: 合成大 input 与最终验证

**Files:**
- No code change.

**Interfaces:**
- Consumes: Task 1-4 的实现。
- Produces: 可复查的测试输出和一条合成大 input 验证记录。

- [x] **Step 1: 运行内容审计定向测试**

Run: `cd backend && go test ./internal/service -run 'TestExtractContentModerationInput|TestContentModerationInput' -count=1`

Expected: PASS。

- [x] **Step 2: 运行 OpenAI usage 定向测试**

Run: `cd backend && go test ./internal/handler ./internal/service -run 'TestOpenAI.*Usage|TestOpenAIGatewayServiceRecordUsage|TestSubmitUsageRecordTask' -count=1`

Expected: PASS。

- [x] **Step 3: 运行合成大 input 测试入口**

Run: `cd backend && go test ./internal/service -run 'TestExtractContentModerationInput_ResponsesLargeInputKeepsLatestText|TestExtractContentModerationInput_ResponsesInlineImagesLimitedDuringCollection' -count=1 -v`

Expected: PASS，输出中两条测试均通过；这覆盖超大文本窗口和图片收集期限量。

- [x] **Step 4: 手工检查闭包捕获**

在 `backend/internal/handler/openai_gateway_handler.go` 搜索 `submitOpenAIUsageRecordTask`，确认 Responses HTTP 与 WS 成功路径的闭包内不再直接引用 `result`、`detailSnapshot`、`c`、请求 body 或 `gin.Context`，只引用 `usageInput` 和日志小字段。

- [x] **Step 5: 更新 tasks 状态由 Comet 流程处理**

本计划执行时不修改 `openspec/changes/optimize-large-input-memory/tasks.md`；完成状态由后续 Comet verify/archive 阶段统一处理。

## 自检

- 需求 1.1/1.2：Task 1 和 Task 2 覆盖大文本收集期截断，保留最新内容。
- 需求 1.3/1.4：Task 1 和 Task 2 覆盖 inline/base64 图片达到上限后不再构造额外 data URL。
- 需求 2.1/2.2：Task 3 和 Task 4 覆盖 usage 快照构造与闭包捕获收缩。
- 需求 2.3：Task 3 的 service 回归和 Task 4 的定向测试覆盖计费、图片、渠道字段。
- 需求 3.1/3.2：Task 5 给出定向 Go 测试和合成大 input 验证命令。
- 跳过项：不做流式解析、不加配置、不加依赖；这些是设计文档明确非目标。
