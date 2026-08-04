# 验证报告：reduce-request-body-memory-retention

- Change: reduce-request-body-memory-retention
- 日期: 2026-08-03
- verify_mode: full（31 任务；Wave 26 复验）
- 分支: feature/20260803/reduce-request-body-memory-retention（84 commits，代码审查 HEAD 9c2e41a7b，base 8b494e187）

## Summary

| 维度 | 状态 |
|------|------|
| Completeness | 31/31 tasks 完成；4 个 requirement 全部实现 |
| Correctness | 4/4 requirements 覆盖，spec 场景均有实现与测试 |
| Coherence | 符合 design.md 高层决策与 Design Doc；无矛盾 |

## 回归专项审查（用户要求，2026-08-03 补充）

验证通过后追加**十轮**业务回归专项审查（26 个修复 wave，9c2e41a7b 收尾，84 commits），全部发现已解决：

| 轮次 | 方法 | 发现 | 处置 |
|------|------|------|------|
| 第 1 轮 | 重点面核查 | Chat usage hash 口径漂移（**真回归**）、Gemini transport spool、原生 Anthropic byte-backed、fallback 标量 | 6 wave |
| 第 2 轮 | 修复 wave 复核 | post-ping SSE 裸 JSON、Responses silent EOF、passthrough build 400、Antigravity 502、等待期持有 | 5 wave |
| 第 3 轮 | 二级路径 + 收尾 | Antigravity 二级路径吞 sentinel、测试假阳性、Claude 路径 byte-backed、body 未关闭 | 3 wave |
| 第 4 轮 | 穷举式（26 文件逐 hunk） | Responses 标量 clone 遗漏 | 1 wave |
| 第 5 轮 | 全链路错误语义复核 | parser spool 未统一 503、OAuth/Websearch ReadAll 错误未传播、transport spool error 漏关 response body、Gemini backoff 不响应 ctx | 1 wave（16） |
| 第 6 轮 | Gateway Chat 收尾 | Gateway Chat 未 handle 化、Chat retry backoff 不响应 ctx、验证报告范围失真 | 1 wave（17） |
| 第 7 轮（最终） | 发布前整体关 | Wave 17/18 Gateway Chat 复审：post-ping SSE spool error 追加裸 JSON、lenient JSON/解压后限长契约回归、未提交 spool error envelope 漂移 | Wave 18 修复 post-ping/lenient，Wave 19 修复 envelope |
| 第 8 轮（复核） | Wave 20 驻留与错误语义 | Composite/ClaudeCodeOnly 解压 body、OpenAI `/v1/messages` 转换 body 跨等待驻留；Grok media transport spool 落为 502；设计/API 清单失真 | Wave 20 修复并复验 |
| 第 9 轮（最终） | 剩余入口与契约收尾 | OpenAI `/v1/messages` raw-chat fallback、Grok retry snapshot、CountTokens/AlphaSearch/Live、Gateway Chat 与 Composite 契约、CC sender 关闭缺口 | Wave 21-24 修复并复验 |
| 第 10 轮（最终复核） | CountTokens / Live / AlphaSearch / 共享发送器收尾 | OpenAI CountTokens、Live multipart、Alpha Search、`sendCCUpstreamRequestHandle`、Composite/Gateway Chat 复核 | Wave 25-26 修复并复验 |

**回归审查最终结论**：**PASS（可发布）**——无未解决的业务回归；重放一致性、错误映射（未提交 503 / 已提交协议终止帧）、failover、usage/billing（含零费用 failed usage 审计语义）、streaming 协议（SSE `event: error`/`response.failed`）均与 base 一致；既有 exported 签名未改变，新增导出表面为 `DefaultRequestBodySpoolThresholdBytes`、`DefaultRequestBodyPreviewLimitBytes`、`GeminiMessagesCompatService.ForwardHandle`、`AntigravityGatewayService.ForwardAsResponsesHandle` 和 `ParsedRequest.RequestPayloadHash`；WS 文件零改动；wave 14 为纯测试，wave 16/17/18/19/20 含生产修复。

## 1. Completeness（完整性）

- tasks.md：31 个任务全部 `[x]`（0 未完成）
- 变更范围与 tasks.md 及后续 26 个回归修复 wave 一致（handler/service 生产代码、测试、memory 文档）
- delta spec 4 个 requirement 全部有对应实现：
  - 大请求文件化（spool 阈值 1MB 导出常量、preview 256KB）✅
  - usage 指纹与业务语义保持（session/usage/cyber hash 预计算口径测试）✅
  - 上游等待期零完整副本（20 条路径阻塞 heap 覆盖）✅
  - preview 有界（256KB 上限 + 序列化快照断言）✅

## 2. Correctness（正确性）

### Requirement 1: 大请求使用可重放的文件化请求体 ✅
- `DefaultRequestBodySpoolThresholdBytes = 1 << 20`、`DefaultRequestBodyPreviewLimitBytes = 256 << 10`（request_body_handle.go:24-25，导出）
- 阈值边界测试：1MB 不落盘、1MB+1 落盘（request_body_size_matrix_test.go）
- gzip/multipart 矩阵保留；spool 失败 503 映射（含 transport 阶段 sentinel 保留，final fix wave 3 修复）

### Requirement 2: usage 指纹与业务语义不变 ✅
- session hash 基于 normalize 前 raw body（Responses）；usage hash 基于 channel mapping 前原始 body（Chat，修复轮 R1 前置）；cyber key 用 `CyberSessionBlockKey` 口径 body 尚存时预计算（含 Grok header 边界测试）
- `requestPayloadHash` 基于改写后 final handle hash
- `deriveOpenAIForwardAttemptBody` 与 `passthroughFailoverState` 状态机未变

### Requirement 3: 上游等待期间不持有完整请求体内存副本 ✅
- handler 循环内按需物化（Responses/ChatCompletions/Anthropic/Gemini/Grok）
- service 层 Forward retry handle 化（invalid_encrypted_content/rejected field 改写重建 handle）
- 22 条大 body 路径阻塞 transport heap 验证：retained growth 均 < 3MiB；最终复验中 Composite 6,392B、ClaudeCodeOnly 1,008B、OpenAI `/v1/messages` 13,464B、Alpha Search 24,360B、Live 13,312B、Gateway Chat Anthropic 21,352B、Gateway Chat Gemini 1,312B
- WS 分支明确排除（后续 change），两个 WS 文件 git diff 零改动

### Requirement 4: preview 有界 ✅
- preview 上限 256KB，不超过 spool 阈值 1MB
- 测试覆盖普通 JSON 256KB 截断 + 序列化快照总长上限（修复轮 R1 补占位符缺口）

## 3. Coherence（一致性）

- 实现符合 design.md 高层决策（D1 阈值 1MB、D2 常量统一派生、D3 循环内物化）与 Design Doc 全链路 handle 化方案
- 既有公开 `Forward` 签名保持；导出新增项完整清单为两个默认常量、两个 handle-backed 方法及 `ParsedRequest.RequestPayloadHash` 字段
- 无新增依赖、配置、数据库 schema
- memory/context 文档已更新为 v2 结论（10MB 标注为 v2 前状态）

## 4. 代码审查

- build 阶段 final whole-branch review（thorough）：3 Critical + 2 Important，经 3 轮修复全部解决（final fix waves 1-3）
- 覆盖正确性（ownership 边界、错误映射）、安全（preview 清洗、无敏感泄漏）、边界（hash 口径、槽位释放）
- spool 失败统一 503 约束整体通过（materialization / outbound open / transport / signature retry 的 sentinel 均保留至 handler，含 Grok media）

## 5. 验证证据

- `go build ./...`：exit 0
- `go vet ./...`：exit 0
- `go test ./... -count=1`（backend/）：exit 0；handler 80.665s，service 115.353s，无 FAIL/panic
- blocked transport heap 覆盖：20 路径 retained growth < 3MiB；Wave 20 三条新增路径最终增长分别为 6,392B / 1,008B / 13,464B
- WS 文件 `git diff`：零改动

## 结论

**PASS**。无 CRITICAL/WARNING 未解决项。可进入归档。

## 已知限制（记录，非阻断）

- WSv2 reconnect map 持有不在本 change 范围（Design Doc 明确排除，列为后续 change）
- 首次并行全量测试出现一次既有 WSv2 close/EOF 时序 flake（串行重跑通过，与本次改动无关）
