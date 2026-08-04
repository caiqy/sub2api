# Reduce Request Body Memory Retention — Design

## Context

动机见 proposal.md。核心事实（已通过代码核实）：

1. **双份拷贝问题**：`Responses` 主路径 `coordinator.ReadRaw()` → `raw.ReadAll()` 时，`io.ReadAll` 产生与 `raw.memory` **独立的第二份** `body []byte`。spool 生效时 `raw.memory` 为 nil，只剩 body 一份；spool 未生效（≤10MB）时两份并存。8.9MB 请求 ≈ 17.8MB 峰值。
2. **持有窗口**：`body []byte` 从 ReadRaw（handler 入口）存活到 handler 结束（failover 循环 + 上游等待），期间 normalize 改写还会产生新 `[]byte`（sjson 类操作），旧引用（`sessionHashBody` 等）仍持有旧份。
3. **阈值常量三处**：`request_body_handle.go:24`（默认值）、`openai_gateway_handler.go:97`（Responses 专用）、`openai_gateway_request_body.go:40`（service 层 OpenAI 通用）。`jsonRequestBodyHandleOptions`（coordinator 用）来自 handler 常量。
4. **测试依赖**：约 90 处测试显式覆盖 options（不受默认值影响）；`openai_gateway_handler_test.go:4665` 与 `openai_gateway_request_body_retention_test.go:137` 用常量构造 body 大小，需同步。
5. **上游发送已是流式**：`openAINewRequestWithBodyHandle` → `bodyHandle.Open()`，不读回内存。

## Goals / Non-Goals

**Goals**
- 1~10MB 请求体落盘（消除 `raw.memory` 第二份拷贝）
- 缩短 handler 层早期 `body []byte` 引用的持有窗口
- 保持全部业务语义（failover/retry/usage/审计）不变

**Non-Goals**
- 不删除或改变既有 byte-based `Forward(ctx, c, account, body []byte)` 兼容入口；内部允许增加 handle-backed 入口并由 byte wrapper 委托，现有调用方无需迁移
- 不再次调整 preview limit（当前 256KB，观测契约）
- 不动 multipart 逻辑与 `multipartParseMemoryBudget`

## Decisions

### D1: 阈值统一收紧到 1MB（`1 << 20`）

选择 1MB 而非 512KB：分布显示 100KB~1MB 是主体流量（占 68%），若降到 512KB 会让大量正常小请求落盘（磁盘 I/O 暴增），收益却只多覆盖 0.5~1MB 的一小段。1MB 恰好覆盖 1~10MB 的 280 个/窗口大请求，I/O 增量可控（~1.2GB/天）。

替代方案：512KB（多覆盖一段但 I/O 翻倍）；保持 10MB（现状，不解决问题）。已排除。

### D2: 三处常量统一从单一默认值派生

`request_body_handle.go` 的 `DefaultRequestBodySpoolThresholdBytes` 改为 `1 << 20` 作为唯一事实源；handler 与 service options 均引用该导出默认值，避免未来再次分叉。

替代方案：三处各自改值（简单但埋分叉隐患）。选统一派生。

### D3: handler 预计算业务标量，等待与重试由 handle 承载

- normalize、路由和模型改写完成后，将 final body 写入 effective handle
- 在释放同步物化的 `body []byte` 前计算 session/cyber/usage hash、模型与流式标量
- 进入排队、账号选择和上游等待前清空完整切片及解析结构中的大字段
- failover/retry 从 canonical handle 重开 reader；需要改写时建立独立 derived handle，并在对应 attempt 结束后清理

### D4: 测试同步

- `openai_gateway_handler_test.go:4665`：body 大小改用新阈值常量（引用常量而非字面量，自动跟随）
- `openai_gateway_request_body_retention_test.go:137`：同上
- 其余显式覆盖 options 的测试不动

## Risks / Trade-offs

- [磁盘 I/O 增加] 280 次/窗口的写+读（~1.2GB/天）→ 生产为 SSD/云盘，量级可忽略；`/tmp` 空间需监控（spool 文件 24h stale sweep 已存在）
- [文件化失败面扩大] 阈值降低后更多请求可能命中 spool 失败 → 已有完整 503 映射与重试机制，行为不变，仅频率上升
- [handler 引用释放引入回归] 过早释放会改变 hash、路由或重放内容 → D3 在清空切片前冻结业务标量，canonical 内容由 handle 保留；由 failover 与 20 路径 retention 覆盖验证
- [preview 常驻成本] 每个 handle 仍保留最多 256KB preview → 属观测契约，接受

## Migration Plan

- 部署：常规镜像发布（GitHub Actions → 生产拉取），无需数据迁移
- 回滚：回退上一镜像即可；spool 文件由 stale sweep 清理
- 验证：`go test ./... -count=1` 全绿；受控 0.5MB/1MB/2MB/10MB identity 请求对照 spool 目录与 usage；生产观测内存峰值与 `/tmp` 使用率

## Open Questions

无（本设计不改变 specs、方案与任务拆分；阈值取值已在 D1 定案）。
