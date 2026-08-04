# Reduce Request Body Memory Retention

## Why

dmit-serv-ai 生产峰值内存贴边（980MB 撞 cgroup 980MiB 限制，峰值距限制仅 217KB）。实测请求体分布显示 1~10MB 的请求体是常态（约 280 个/10 小时窗口，含 8~9MB 聚集带 40 个），而 `RequestBodyHandle` spool 阈值 10MB 使这些请求体全部驻留内存，且 `ReadRaw()` 后 `raw.memory` 与解析用 `body []byte` 构成双份拷贝。降低阈值 + 缩短 handler 层 body 引用生命周期，可直接削减"瞬时持有 × 并发"的乘积项。

## What Changes

- `RequestBodyHandle` spool 阈值从 10MB 降至 1MB（三处常量统一引用导出的默认值），使 1~10MB 请求体落盘，消除 `raw.memory` 双份拷贝。
- preview limit 从 5MB 降至 256KB，消除 1MB-5MB 请求的 preview 完整副本（preview 不得超过 spool 阈值）。
- 全链路 handle 化：handler 层全部请求级改写完成后重建 final handle 并置 nil body，failover 循环内按需物化；service 层 Forward retry 循环（invalid_encrypted_content / rejected field / 计费提取）同样按需物化并重建 handle。
- 保持语义不变：session hash（normalize 前 raw body）、usage hash（channel mapping 前原始 body）、cyber key（body 尚存时预计算）、`deriveOpenAIForwardAttemptBody` 状态机、`requestPayloadHash` 口径。
- 同步调整依赖 spool 阈值常量构造请求体的测试；`request_body_size_matrix_test.go` 新增默认阈值矩阵。
- WS 分支（`forwardOpenAIWSV2` 持有 map 跨重连循环）**不改**，列为后续 change。
- multipart 大文件逻辑不变；`Forward` 对外签名不变（内部实现 handle 化）。

## Capabilities

### New Capabilities

- (无)

### Modified Capabilities

- `request-body-retention-control`: 系统对 JSON 入口的 spool 阈值从 10MB 收紧为 1MB；preview limit 从 5MB 收紧为 256KB 且不得大于 spool 阈值；上游等待期间不持有完整请求体内存副本（全链路 handle 化）；观测 preview 契约同步调整。

## Impact

- 代码：`backend/internal/service/request_body_handle.go`、`backend/internal/handler/openai_gateway_handler.go`、`backend/internal/service/openai_gateway_request_body.go`、`backend/internal/service/openai_gateway_forward.go`、`backend/internal/handler/openai_chat_completions.go`、`backend/internal/handler/gateway_handler.go`、`backend/internal/handler/gateway_handler_responses.go`、`backend/internal/handler/gemini_v1beta_handler.go`、`backend/internal/handler/openai_images.go`、`backend/internal/handler/grok_media.go`、`backend/internal/handler/request_body_coordinator.go`（间接）、相关测试文件。
- 行为：1~10MB 请求体从内存驻留转为临时文件落盘（磁盘 I/O 增加，约 1.2GB/天）；上游等待期间不再持有完整 body 副本；usage detail 请求预览从 5MB 缩短至 256KB；spool 失败映射 503 语义不变。
- 运维：临时 spool 文件量级增大（stale sweep 已存在，24h 清理），需关注 `/tmp` 磁盘空间。
- 无 API、无数据库 schema、无配置项变更。
