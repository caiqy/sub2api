# Task 12 Report

- Anthropic `/v1/responses` 在 user/group 等待前完成同步解析、session hash 和 request body handle 绑定，并释放 handler 本地大 body 切片；阻塞 user wait 回归确认此时仅保留 raw replay spool，派生 upstream spool 只在实际转发时创建。
- `ForwardAsResponsesHandle` 透传 transport `ErrRequestBodySpool` 的 `%w` 链，不预写 502；真实 Anthropic Responses handler 故障注入确认返回既有 Responses 格式 503，且无 usage 记录、无重试尝试。
- 已提交 writer 时 Responses spool 错误不再追加第二段 JSON。
- 移除了 `request_body_coordinator_test` 中剩余的生产源码字符串检查，保留 raw/effective/multipart 临时资源的实际 cleanup 行为断言。
- Task 5.1 入口矩阵继续覆盖 Anthropic Responses、OpenAI Chat/Embeddings、Gemini 和媒体独立行为路径；本轮仅补齐缺失的 Anthropic Responses transport 与等待生命周期代表路径。
- 验证命令使用 `D:\cache\sub2api-task12`：Task12 定向 handler/service、`go test ./internal/handler -count=1`、`go test ./internal/service -count=1` 和 `go test -tags=unit ./internal/handler -count=1`。
