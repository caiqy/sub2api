# Brainstorm Summary

- Change: reduce-request-body-memory-retention
- Date: 2026-08-03
- Status: **已确认 v2**（2026-08-03 reviewer 审查后，用户选择全链路 handle 化）

## 确认的技术方案 v2（含 service retry）

**① spool 阈值 10MB→1MB**（三处常量，导出默认值供跨包引用；preview limit 同步评估降级）

**② 全链路 handle 化（含 service 层 retry 循环）**：
- handler 层：全部请求级改写（route model / reasoning policy / channel mapping）完成后**重建并重绑 final effective handle**，再 `body = nil`；failover 循环内每轮从 final handle 物化→派生→Forward→释放
- service 层：OpenAI `Forward` retry 循环（invalid_encrypted_content 改写、rejected field 改写、错误计费提取、service tier 提取）改为"每轮重试前从 handle 物化→改写→重建 handle"；响应到达后按需物化提取小字段（计费/usage/service tier），提取完释放
- WS 分支（forwardOpenAIWSV2 持有 wsReqBody map）**明确不改**，列为后续
- 模板：openai_embeddings.go:120 已实现 handler 层模式；service 层需扩展

## 关键取舍与风险（v2 补充）

- **session hash / usage hash 必须保持原口径**：基于 normalize 前 raw body 预计算（session hash）、channel mapping 前原始 body 预计算（usage hash），不得改用 effective body
- **CyberSessionBlockKey**：body 尚存时预计算完整 cyber key（用 `explicitOpenAISessionID` 口径，含 Grok header 边界），不得用语义不同的 `ExtractSessionID` 替代
- **preview 5MB 与"零完整副本"冲突**：1MB-5MB 请求 preview 即完整副本 → 需决策（降 preview limit 或 spec 明确"有界 preview 豁免"）
- **final handle 重建**：Responses 现有 handle 建于 line 367，之后 line 419/429 仍有改写 → 必须全部改写后重建重绑
- **性能**：无 failover 时多一次 ReadAll（8.9MB 文件读 1-2ms）——接受，或 service 层复用已物化的首轮 body
- 槽位泄漏：循环内物化失败需释放已获取账号槽位

## 测试策略（v2）

- handler 层 + service 层 failover/retention/spooling 定向矩阵
- 阻塞 transport 测试：transport 读完 body 后阻塞，2MB/8.9MB 请求 GC/heap 检查（覆盖 preview 字符串）
- `request_body_size_matrix_test.go` 增加 0.5MB/1MB/1MB+1/2MB/10MB 默认阈值矩阵
- Responses 集成测试：route model + reasoning policy + channel mapping 同时生效，failover 两轮 body 正确，session/usage hash 原口径
- 全量 `go test ./... -count=1`

## Spec Patch

- delta spec 保持"上游等待期零完整副本"强约束（v2 确认坚持）
- 需明确"有界 preview 豁免"或降 preview limit
- 需补充 service 层 requirement：retry 循环内不跨 attempt 持有完整 body
