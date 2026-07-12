## 1. 共享 request body coordinator

- [x] 1.1 为 identity、gzip 等压缩 JSON 编写阈值、64MB 解压上限、preview、hash、503 和 cleanup 的失败优先测试。
- [x] 1.2 实现共享 coordinator，使解压流直接进入 `RequestBodyHandle`，并支持 raw/effective handle 复用与显式 ownership。
- [x] 1.3 增加成功、业务拒绝、客户端取消、panic recovery、handle 替换和 stale cleanup 生命周期测试。

## 2. Anthropic 与 OpenAI JSON 入口

- [x] 2.1 将 Anthropic 分组 `/v1/responses` 兼容转换和 `/v1/messages` 迁移到 coordinator，保持内容审计、转换、计费和流式语义。
- [x] 2.2 将 OpenAI `/v1/chat/completions` 与 Embeddings 迁移到 coordinator，使最终 outbound body 通过 effective handle 支持 retry/failover 重放。
- [x] 2.3 为四类入口补充小请求、大请求、压缩请求、上游 4xx/5xx、取消、retry/failover 和 usage/ops snapshot 回归测试。

## 3. Gemini 原生入口

- [x] 3.1 将 Gemini `/v1beta/models/*` 的 generateContent、streamGenerateContent 与 countTokens 请求迁移到 coordinator。
- [x] 3.2 验证 Gemini 模型路径解析、内容审计、Google 错误格式、流式终止、failed usage 和 Antigravity 强制平台路由保持不变。

## 4. OpenAI/Grok 媒体入口

- [x] 4.1 为 JSON、multipart、inline binary 和源图/遮罩上传编写 spool、脱敏 metadata、完整上游请求和 cleanup 测试。
- [x] 4.2 将 OpenAI/Grok Images 与 Videos 的 raw multipart 和 effective outbound multipart 接入 coordinator，并统一 `RemoveAll` 与 handle cleanup。
- [x] 4.3 验证生成、编辑、视频创建、视频状态、业务拒绝、上游错误和重试路径不泄露二进制正文且不残留临时文件。

## 5. 全链路验证

- [ ] 5.1 增加跨协议契约测试，确认所有目标入口使用共享 coordinator，usage/ops 不回读完整 body，spool I/O 失败统一返回 503。
- [ ] 5.2 运行后端全量测试、前端全量测试与类型检查，并执行 5MB/10MB/12MB identity、gzip、multipart 受控端侧矩阵。
- [ ] 5.3 对照容器 RSS、spool 生命周期、usage detail 和 ops 数据，确认进入上游等待后无长期大 `[]byte` 引用且业务语义无回归。
