## Context

`RequestBodyHandle` 已为 OpenAI/Grok `/responses` 提供 10MB spool、5MB preview、hash、重复 `Open()` 和 cleanup，但其余入口仍直接调用 `ReadRequestBodyWithPrealloc`，完整 body 会跨越协议解析、上游请求构建、网络等待和 retry 生命周期。各入口还分别处理压缩、multipart、内容审计和 request capture，直接复制 `/responses` 代码会形成六套生命周期分支。

本变更涉及 Anthropic/OpenAI/Grok/Gemini handler、协议转换和媒体上传，必须保持现有 API、计费、内容审计、模型映射、错误透传和流式语义不变。继续使用标准库和现有 `RequestBodyHandle`，默认阈值保持 10MB spool 与 5MB preview。

## Goals / Non-Goals

**Goals:**

- 为所有剩余大 body 入口提供统一、可重放、可清理的文件化请求体。
- 让 identity 与压缩 JSON 按解压后的有效大小执行相同阈值；让 multipart 上传使用磁盘承载大文件内容。
- 将 raw inbound body 和转换后的 effective outbound body 分开管理，确保 retry/failover 重放最终上游内容。
- 在进入上游等待后释放同步解析阶段的完整 `[]byte`，usage/ops 只持有 bounded snapshot。
- 为成功、失败、取消、流式、retry/failover 和 spool I/O 失败建立跨协议回归矩阵。

**Non-Goals:**

- 不实现流式 JSON parser；同步校验和协议转换阶段仍可短暂 materialize 完整 JSON。
- 不改变 10MB/5MB 默认值，不增加运行时配置。
- 不改变 response body capture、数据库 schema、API 契约或上游协议。
- 不以本变更解决 Nginx 与应用 body limit 配置不一致问题。

## Decisions

### 1. 共享 coordinator，而不是复制 handler 分支或全局 middleware

新增小型共享入口 coordinator，组合现有解压上限、`RequestBodyHandle`、preview snapshot 和 cleanup。它返回 raw handle、必要的同步读取能力及明确 ownership；各协议 handler 负责解析和转换。

选择该方案是因为全局 middleware 不理解 effective outbound body、multipart 和协议转换，而逐 handler 复制会重复错误处理与 cleanup。coordinator 只管理 body 生命周期，不引入协议接口或工厂。

### 2. raw inbound 与 effective outbound 使用独立 handle

raw handle 表示客户端提交的解压后 JSON 或原始 multipart body，用于审计、hash 和入站 snapshot。协议转换或模型映射产生最终上游 body 后，创建 effective handle 并绑定到具体 attempt；HTTP request body 与 `GetBody` 都从该 handle 打开 reader。

如果 raw/effective 内容相同，可通过 size/hash 复用 handle；不同则分别清理。retry/failover 只能重放 effective handle，不能回读 usage/ops snapshot。

### 3. 压缩 JSON 流式解压进入 handle

gzip 等已支持编码通过带 64MB 安全上限的解压 reader 直接写入 handle，spool 阈值按解压后的有效 body 计算，避免先完整解压到 `[]byte` 再 spool。超出解压上限继续返回 413；spool I/O 失败返回 503。

### 4. multipart 复用标准库临时文件能力

媒体入口先用 raw handle 承接原始 multipart 请求，再从 handle 创建 reader 交给标准库 multipart parser。文件 part 使用标准库磁盘临时文件，文本字段只保留协议需要的有限内容；任何路径都调用 `RemoveAll` 和 handle cleanup。

媒体服务构建新的上游 multipart body 时使用 effective handle，使大输出 body 同样可重放。usage/ops 继续只保存模型、prompt、尺寸和是否含源图等脱敏 metadata。

### 5. handler 显式持有 ownership

coordinator 不把 cleanup 隐藏在全局 middleware。handler 在成功创建 handle 后立即注册 defer；转交给异步或 retry 组件时使用显式 ownership 标记，沿用现有 owned-handle context 模式。成功、业务拒绝、上游错误、客户端取消和 panic recovery 都必须释放资源。

### 6. 保持现有观测与失败契约

所有入口调用同一个 immutable snapshot setter；5MB 内保留可审计 preview，超限或敏感/不完整 JSON fail-closed。ops 单字段继续受 20KB 限制。创建、写入、关闭、打开或读取 spool 文件失败均包装为 `ErrRequestBodySpool`，入口在尚未写响应时返回 503。

## Risks / Trade-offs

- [同步 JSON 解析仍有瞬时峰值] → 明确在进入上游等待前释放完整切片；用运行时测试验证长期驻留而非承诺零峰值。
- [multipart 可能同时存在 raw spool 和标准库 part 临时文件] → 严格限定 ownership，测试 `Cleanup` 与 `RemoveAll`，并保留 stale sweep 兜底。
- [压缩炸弹消耗磁盘或 CPU] → 继续执行 64MB 解压上限和全局 body limit，不允许无限解压。
- [协议转换产生多个 effective body] → 每个 attempt 只持有当前 effective handle，替换时先清理旧 owned handle。
- [跨入口改动面较大] → 按协议分批接入共享 coordinator，每批保留独立回归测试，并运行后端、前端和端侧矩阵。

## Migration Plan

1. 先引入共享 coordinator 和跨编码测试，不改变 handler。
2. 依次迁移 Anthropic/兼容 Responses、OpenAI Chat/Embeddings、Gemini、媒体入口。
3. 每批迁移后运行协议定向测试，最后运行全量测试和受控大 body 端侧验证。
4. 部署无需数据迁移；回滚为上一镜像即可，stale sweep 会清理遗留 spool 文件。

## Open Questions

无。阈值、覆盖入口、压缩与 multipart 口径、失败语义均已确认。
