# Brainstorm Summary

- Change: extend-request-body-spooling
- Date: 2026-07-10

## 确认的技术方案

- 使用共享 request body coordinator 组合解压、`RequestBodyHandle`、preview snapshot 和 cleanup，不复制六套 handler 分支，也不放入协议无感的全局 middleware。
- 分离 raw inbound handle 与 effective outbound handle；前者负责客户端 body、审计和入站 snapshot，后者负责协议转换后的最终上游 body 与 retry/failover 重放。
- identity 与压缩 JSON 均按解压后有效大小应用 10MB spool；压缩流直接写入 handle，并继续执行 64MB 解压上限。
- multipart 使用 raw handle 承接原始请求，文件 part 复用标准库临时文件，重建的上游 multipart 使用 effective handle。
- handler 显式持有 ownership，在成功、失败、取消、panic recovery、retry 和 handle 替换路径清理资源。

## 关键取舍与风险

- 不做流式 JSON parser，同步解析阶段允许短暂 materialize 完整 JSON；目标是消除上游等待期和观测链路的长期大副本。
- multipart 可能短时同时存在 raw spool、part 临时文件和 outbound spool；通过显式 ownership、`RemoveAll`、handle cleanup 和 stale sweep 控制。
- spool I/O 失败统一返回 503；压缩超限继续返回 413，不回退到高内存路径。
- 保持 10MB spool、5MB preview、20KB ops 单字段边界，不新增依赖、配置或数据库 schema。

## 测试策略

- 共享 coordinator：identity/gzip、阈值边界、64MB 解压上限、hash、preview、503、取消和 cleanup。
- 协议矩阵：Anthropic Responses/Messages、OpenAI Chat/Embeddings、Gemini generate/stream/count、OpenAI/Grok Images/Videos。
- 生命周期：成功、4xx/5xx、业务拒绝、panic、取消、retry/failover、effective handle 替换和 multipart `RemoveAll`。
- 最终验证：后端/前端全量测试、类型检查，以及 5MB/10MB/12MB identity、gzip、multipart 端侧与服务器数据对照。

## Spec Patch

无。现有 delta spec 已覆盖压缩、multipart、失败语义和所有终止路径清理。
