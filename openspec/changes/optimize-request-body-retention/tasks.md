## 1. Request body handle

- [ ] 1.1 实现 `RequestBodyHandle`，支持内存模式、文件模式、preview、hash、size、重复 `Open()` 和 `Cleanup()`。
- [ ] 1.2 增加 stale spool 文件清理与文件化失败测试。

## 2. `/responses` 请求链路改造

- [ ] 2.1 改造 `OpenAIGatewayHandler.Responses`，引入 raw/effective handle，提前计算 `requestPayloadHash`。
- [ ] 2.2 改造 usage detail request body capture，不再在 middleware 预读完整 request body。
- [ ] 2.3 改造 upstream request capture 和 ops upstream capture，只保留 preview。
- [ ] 2.4 改造 OpenAI upstream request builder，使 `>10MB` body 通过文件型 `GetBody` 重放。

## 3. 验证

- [ ] 3.1 增加 `RequestBodyHandle` 单元测试，覆盖内存/文件模式和 cleanup。
- [ ] 3.2 增加 `/responses` 回归测试，确认 `requestPayloadHash` 与现状一致。
- [ ] 3.3 增加大 body 路径测试，确认 usage/ops 只保留 preview，failover / retry 内容不变。
- [ ] 3.4 增加 spool 创建失败测试，确认返回 `503` 且不回退到高内存路径。
