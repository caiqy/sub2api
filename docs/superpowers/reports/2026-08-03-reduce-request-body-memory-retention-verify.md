# 验证报告：reduce-request-body-memory-retention

- Change: reduce-request-body-memory-retention
- 日期: 2026-08-03
- verify_mode: full（31 任务 / 39 文件 / 2757 行新增）
- 分支: feature/20260803/reduce-request-body-memory-retention（33 commits，base 8b494e187）

## Summary

| 维度 | 状态 |
|------|------|
| Completeness | 31/31 tasks 完成；4 个 requirement 全部实现 |
| Correctness | 4/4 requirements 覆盖，spec 场景均有实现与测试 |
| Coherence | 符合 design.md 高层决策与 Design Doc；无矛盾 |

## 回归专项审查（用户要求，2026-08-03 补充）

验证通过后追加**五轮**业务回归专项审查（16 个修复 wave，b563f6066 收尾，55 commits），全部发现已解决：

| 轮次 | 方法 | 发现 | 处置 |
|------|------|------|------|
| 第 1 轮 | 重点面核查 | Chat usage hash 口径漂移（**真回归**）、Gemini transport spool、原生 Anthropic byte-backed、fallback 标量 | 6 wave |
| 第 2 轮 | 修复 wave 复核 | post-ping SSE 裸 JSON、Responses silent EOF、passthrough build 400、Antigravity 502、等待期持有 | 5 wave |
| 第 3 轮 | 二级路径 + 收尾 | Antigravity 二级路径吞 sentinel、测试假阳性、Claude 路径 byte-backed、body 未关闭 | 3 wave |
| 第 4 轮 | 穷举式（26 文件逐 hunk） | Responses 标量 clone 遗漏 | 1 wave |
| 第 5 轮（最终） | 发布前整体关 | parser spool 未统一 503、OAuth/Websearch 静默 Bytes()、transport 漏关 resp body、Gemini backoff 不响应 ctx | 1 wave（16） |

**回归审查最终结论**：**PASS（可发布）**——无未解决的业务回归；重放一致性、错误映射（未提交 503 / 已提交协议终止帧）、failover、usage/billing（含零费用 failed usage 审计语义）、streaming 协议（SSE `event: error`/`response.failed`）均与 base 一致；新增 2 个导出常量（`DefaultRequestBodySpoolThresholdBytes`/`DefaultRequestBodyPreviewLimitBytes`，有意的默认值 API），无其他 exported API 变更；WS 文件零改动；wave 14 为纯测试，wave 16/17 含生产修复。

## 1. Completeness（完整性）

- tasks.md：31 个任务全部 `[x]`（0 未完成）
- 变更文件 39 个，与 tasks.md 描述一致（handler/service 生产代码、测试、memory 文档）
- delta spec 4 个 requirement 全部有对应实现：
  - 大请求文件化（spool 阈值 1MB 导出常量、preview 256KB）✅
  - usage 指纹与业务语义保持（session/usage/cyber hash 预计算口径测试）✅
  - 上游等待期零完整副本（8 条路径阻塞 heap 矩阵）✅
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
- 8 条大 body 路径阻塞 transport heap 验证：retained growth 均 < 3MiB（2MB vs 8.9MB 增长 84KB；混合 Gemini 分支从 7.2MB 降至 1.5KB）
- WS 分支明确排除（后续 change），两个 WS 文件 git diff 零改动

### Requirement 4: preview 有界 ✅
- preview 上限 256KB，不超过 spool 阈值 1MB
- 测试覆盖普通 JSON 256KB 截断 + 序列化快照总长上限（修复轮 R1 补占位符缺口）

## 3. Coherence（一致性）

- 实现符合 design.md 高层决策（D1 阈值 1MB、D2 常量统一派生、D3 循环内物化）与 Design Doc 全链路 handle 化方案
- 公开 `Forward` 签名保持；仅新增 handle-backed 内部入口（ForwardGeminiHandle/ForwardAsResponsesHandle）
- 无新增依赖、配置、数据库 schema
- memory/context 文档已更新为 v2 结论（10MB 标注为 v2 前状态）

## 4. 代码审查

- build 阶段 final whole-branch review（thorough）：3 Critical + 2 Important，经 3 轮修复全部解决（final fix waves 1-3）
- 覆盖正确性（ownership 边界、错误映射）、安全（preview 清洗、无敏感泄漏）、边界（hash 口径、槽位释放）
- spool 失败统一 503 约束整体通过（materialization / outbound open / transport / signature retry 的 sentinel 均保留至 handler）

## 5. 验证证据

- `go build ./...`：exit 0
- `go vet ./...`：exit 0
- `go test ./... -count=1`（backend/）：exit 0，无 FAIL/panic
- blocked transport heap 矩阵：8 路径 retained growth < 3MiB
- WS 文件 `git diff`：零改动

## 结论

**PASS**。无 CRITICAL/WARNING 未解决项。可进入归档。

## 已知限制（记录，非阻断）

- WSv2 reconnect map 持有不在本 change 范围（Design Doc 明确排除，列为后续 change）
- 首次并行全量测试出现一次既有 WSv2 close/EOF 时序 flake（串行重跑通过，与本次改动无关）
