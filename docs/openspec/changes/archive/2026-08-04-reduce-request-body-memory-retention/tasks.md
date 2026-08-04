# Tasks — reduce-request-body-memory-retention

## 1. Spool 阈值与 preview 收紧

- [x] 1.1 将 `defaultRequestBodySpoolThresholdBytes`（`backend/internal/service/request_body_handle.go:24`）从 `10 << 20` 改为 `1 << 20`，并**导出**（`DefaultRequestBodySpoolThresholdBytes`）供跨包引用
- [x] 1.2 将 `defaultRequestBodyPreviewLimitBytes`（`request_body_handle.go:25`）从 `5 << 20` 改为 `256 << 10`（256KB），并导出
- [x] 1.3 `openai_gateway_handler.go:97` `openAIResponsesRequestBodySpoolThresholdBytes` 与 `openAIResponsesRequestBodyPreviewLimitBytes`（line 103）改为引用导出的默认值，删除重复字面量
- [x] 1.4 `openai_gateway_request_body.go:40` `SpoolThresholdBytes: 10 << 20` 改为引用默认值
- [x] 1.5 同步依赖阈值常量的测试：`openai_gateway_handler_test.go:4665`、`openai_gateway_request_body_retention_test.go:137` 改用常量引用而非字面量
- [x] 1.6 `request_body_size_matrix_test.go` 新增 0.5MB/1MB/1MB+1/2MB/10MB 默认阈值矩阵（现固定 10MB）
- [x] 1.7 `go build ./...` 与 `go vet ./...` 通过

## 2. Responses 入口循环内按需物化（含 final handle 重建）

- [x] 2.1 在 normalize 前从 raw body 预计算 session hash（保持 normalize 前语义）；body 尚存时预计算完整 cyber key（`explicitOpenAISessionID` 口径，含 Grok header 边界）
- [x] 2.2 全部请求级改写（route model line 419 / reasoning policy line 429）完成后**重建 final handle 并重绑**（修复 CRITICAL-1：现有 handle 建于 line 367，之后仍有改写）
- [x] 2.3 循环外 `body = nil`，failover 循环内每轮从 final handle `ReadAll()` 物化 canonical body
- [x] 2.4 循环内 `deriveOpenAIForwardAttemptBody` 改用物化结果派生，`passthroughFailoverState` 状态机语义不变
- [x] 2.5 循环内物化失败时释放已获取账号槽位（MINOR 修复）
- [x] 2.6 保留 `requestPayloadHash` 现有口径（基于改写后 final body）
- [x] 2.7 运行 Responses 定向测试（retention / spooling / failover 系列）确认无回归

## 3. service 层 Forward retry 循环 handle 化

- [x] 3.1 `openai_gateway_forward.go` retry 循环：`invalid_encrypted_content`（line 870-885）与 rejected field（line 887-895）改写路径改为"物化→改写→重建 handle→重发→释放"
- [x] 3.2 响应到达后 `handleErrorResponse`（计费/usage 提取）与 `extractOpenAIServiceTierFromBody`（line 935）改为按需物化提取小字段后释放
- [x] 3.3 首轮 attempt 复用 handler 已物化的 body，不重复 ReadAll（无 failover 时与现状性能等价）
- [x] 3.4 运行 service 层 failover / retry 定向测试确认无回归

## 4. ChatCompletions 入口对齐

- [x] 4.1 循环外 `body = nil`，循环内 `forwardBody` 改为从 handle 物化
- [x] 4.2 `HashUsageRequestPayload` 在 channel mapping 前对原始 body 预计算（保持计费去重口径）
- [x] 4.3 `CyberSessionBlockKey` 在 body 尚存时预计算完整 key
- [x] 4.4 运行 ChatCompletions 定向测试确认无回归

## 5. 其他入口对齐

- [x] 5.1 Anthropic Messages（`gateway_handler.go`）：对齐循环内物化；修复 `ParsedRequest.Body.Replace(normalizedBody)`（gateway_request.go:237）byte-owned 跨等待期问题
- [x] 5.2 Anthropic Responses 兼容（`gateway_handler_responses.go`）：Antigravity 分支（line 295）不再传全量 body
- [x] 5.3 Gemini（`gemini_v1beta_handler.go`）：主转发已用 handle，仅清残留 byte 引用
- [x] 5.4 multipart 媒体（`openai_images.go` / `grok_media.go`）：Images 验证现有 nil 模式；Grok 处理 session/multipart 二次转换
- [x] 5.5 各入口定向测试通过（WS 分支明确不改，记录为后续）

## 6. 验证

- [x] 6.1 `go test ./... -count=1` 全绿
- [x] 6.2 阻塞 transport 测试：transport 读完 body 后阻塞，2MB/8.9MB 请求执行 GC/heap 检查（2MB 用例验证 preview 不再保留完整副本）
- [x] 6.3 受控 0.5MB / 1MB / 2MB / 10MB identity 请求验证：阈值内不落盘、超阈值落盘、spool 目录清理正常、上游等待期内存零驻留
- [x] 6.4 更新 memory/context 文档：修正方案 3 的表述，记录设计复盘结论（canonical 不可变、全链路 handle 化、preview 256KB、WS 分支排除）
