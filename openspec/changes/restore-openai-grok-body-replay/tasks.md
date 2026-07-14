## 1. OpenAI raw fallback

- [x] 1.1 在 raw chat 入口恢复 bound `RequestBodyHandle` body，并修正 silent-refusal header fixture，运行 replay/failover 聚焦测试。

## 2. Grok raw multipart

- [x] 2.1 保留 raw multipart 上传 bytes，并让 moderation/images edit data URL 转换支持该表示，运行 Grok multipart 与 handler 聚焦测试。
