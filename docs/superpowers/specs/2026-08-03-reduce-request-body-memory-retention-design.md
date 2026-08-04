---
comet_change: reduce-request-body-memory-retention
role: technical-design
canonical_spec: openspec
archived-with: 2026-08-04-reduce-request-body-memory-retention
status: final
---

# 请求体内存驻留治理技术设计（Design Doc）

> open 阶段 design.md 提供高层框架；本文是深度技术细化。背景与动机见 proposal.md，行为契约见 delta spec。

## 1. 现状核实（代码事实）

### 1.1 双份拷贝问题

`Responses`/`ChatCompletions` 主路径：

```
newJSONRequestBody(req) → raw handle（≤阈值存 memory，>阈值落盘）
  → coordinator.ReadRaw() → raw.ReadAll() → io.ReadAll
    → body []byte（与 raw.memory 独立的第二份拷贝）
  → SetEffectiveBytes(body) → effective handle（hash 不同则再建一份）
  → failover 循环持有 body []byte 常驻
```

8.9MB 请求峰值 ≈ 2~3 份 body（17.8~26.7MB）：`raw.memory` + `body` 拷贝 + effective handle 副本。

### 1.2 循环持有与不可释放性

- `body []byte` 是 failover 循环的 canonical 数据源：`deriveOpenAIForwardAttemptBody`（Responses）、`forwardBody := effectiveBody`（ChatCompletions）每轮读取。**canonical 从不被改写**（注释明确："The canonical slice is never mutated"）。
- `sessionHashBody`（Responses）在循环内被 `CyberSessionBlockKey` 消费（只读 `prompt_cache_key`，且仅当 cyber policy 开启）。
- `GenerateSessionHash` 在循环外消费 sessionHashBody（Responses）或 body（ChatCompletions）。

### 1.3 已存在的模板：Embeddings

`openai_embeddings.go:120` 已实现目标模式：

```
handler:  SetEffectiveBytes(body) → BindOpenAIRequestBodyHandle(c, effective) → body = nil
service:  ForwardEmbeddings(ctx, c, account, body=nil, ...)
            → openAIRequestBodyBytes(c, body)   // body 为空时从 context handle ReadAll 物化
            → 按账号改写 model → openAIRequestBodyHandleForContext → 流式发送
```

**Embeddings 已证明该模式可行**：handler 不持有 body，service 每轮从 handle 物化、改写、发送，函数返回即释放。

## 2. 目标模式：循环内按需物化

### 2.1 通用模式（v2：全链路 handle 化）

```
handler 层:
  1. coordinator.ReadRaw() → body（同步解析窗口）
  2. 校验 / 内容审计 / session hash（基于 normalize 前 raw body 预计算）/ model 解析
  3. SetEffectiveBytes(body) → effective handle
  4. 全部请求级改写完成（route model / reasoning policy / channel mapping）后:
       SetEffectiveBytes(finalBody) → 重建 final handle（若内容变化）
       BindOpenAIRequestBodyHandle(c, finalHandle)
       body = nil                                    ← 循环外释放
  5. failover 循环内每轮:
       canonical ← finalHandle.ReadAll()             ← 短暂物化（ms 级）
       attemptBody ← 账号级派生(canonical)
       Forward(attemptBody)                          ← 返回后 canonical 可 GC
  6. 上游等待期间: 0 驻留（handle 文件/内存 + 流式 reader）

service 层（Forward 内部 retry 循环）:
  每轮 attempt:
    canonical ← 当前 handle.ReadAll()                ← 短暂物化
    （invalid_encrypted_content / rejected field 改写时）
    改写 → 重建 attempt handle → 重发 → 释放
    响应到达后按需物化提取小字段（计费/usage/service tier）→ 提取完释放
```

### 2.2 service 层物化入口

复用 `openAIRequestBodyBytes(c, body)` 语义：body 为空时从 gin context 绑定的 handle `ReadAll()`。所有 `Forward*` service 函数入口统一该模式（Embeddings 已如此）。

## 3. 各入口改造点

### 3.1 OpenAI Responses（openai_gateway_handler.go）

| 位置 | 现状 | 改造 |
|---|---|---|
| line 336-358 | `ReadRaw()` → body → normalize（可能产生新切片） | 保留（同步窗口） |
| line 357 | `sessionHashBody := body`（normalize 前） | **预计算 session hash（基于 normalize 前 raw body）**，循环外消费后释放 |
| line 367-373 | `SetEffectiveBytes` + `BindOpenAIRequestBodyHandle` | 保留（初次绑定） |
| line 419/429 | route model 改写 + reasoning policy 改写（在初次 handle 之后） | **全部请求级改写完成后重建 final handle 并重绑**（CRITICAL-1 修复） |
| line 548 | `GenerateSessionHash(c, sessionHashBody)` | 循环外，消费后可释放 |
| line 656-680 | `forwardBody := body` → 循环内 `deriveOpenAIForwardAttemptBody(forwardBody)` | **改为循环内 `finalHandle.ReadAll()` 物化后派生**；`body = nil` |
| line 683-686 | 循环内 `CyberSessionBlockKey(apiKey.ID, c, sessionHashBody)` | **body 尚存时预计算完整 cyber key**（`explicitOpenAISessionID` 口径，含 Grok header 边界） |
| — | 循环内物化失败 | **释放已获取账号槽位**（MINOR 修复） |

**关键语义保持**：
- `deriveOpenAIForwardAttemptBody` 的 `passthroughFailoverState` 状态机不变（每轮派生时机不变，只是 canonical 来源从常驻变量变为每轮物化）
- session hash 必须基于 normalize 前 body（session 标识语义）：在 normalize 前先物化一次计算，或从 raw handle 物化（raw 就是 normalize 前内容）
- `requestPayloadHash`（line 372）保持现有口径：基于改写后 final body

### 3.1b service 层 OpenAI Forward retry 循环（openai_gateway_forward.go）

| 位置 | 现状 | 改造 |
|---|---|---|
| line 819-932 | `Do()` 后直接读 `body` 做错误恢复/改写/计费 | **改为从 attempt handle 按需物化**：响应到达后物化→提取小字段（`handleErrorResponse` 计费、`extractOpenAIServiceTierFromBody`）→释放 |
| line 870-885 | `invalid_encrypted_content` 400 → `trimOpenAIEncryptedReasoningItems(decoded)` → `body = marshal(...)` → continue | **物化→改写→重建 attempt handle→重发→释放**（CRITICAL-2 修复） |
| line 887-895 | rejected field 400 → `normalizeOpenAIResponsesRejectedFieldRetryBody` → `body = retryBody` → continue | 同上：重建 handle 后重发 |
| line 935-937 | 成功响应 `extractOpenAIServiceTierFromBody(body)` | 物化提取后立即释放；`reqBody = nil` 已存在，保持 |

### 3.1c OpenAI Chat Completions（openai_chat_completions.go）

| 位置 | 现状 | 改造 |
|---|---|---|
| line 176-177 | 循环外 `GenerateSessionHash(c, body)` + `ExtractSessionID(c, body)` | 保留（循环外消费） |
| line 256 | 循环内 `forwardBody := effectiveBody` | **改为循环内从 handle 物化**；`body = nil` |
| line 269 | 循环内 `CyberSessionBlockKey(apiKey.ID, c, body)` | **body 尚存时预计算完整 cyber key** |
| line 271 | `HashUsageRequestPayload(body)` | **channel mapping 前预计算 usage hash**（保持计费去重口径，MAJOR 修复） |

### 3.2 OpenAI Embeddings（openai_embeddings.go）

**无需改动**——已是目标模式（line 120 `body = nil`）。

### 3.3 Anthropic Messages（gateway_handler.go）

同模式：line 246 `ReadRaw()` → line 260/325 `SetEffectiveBytes` → 循环内 `body` 消费。对齐循环内物化。**注意 `ParsedRequest.Body.Replace(normalizedBody)`（gateway_request.go:237，模型归一化）会 byte-owned 跨等待期——需改为 handle 承载**（MAJOR 修复）。

### 3.4 Anthropic Responses 兼容（gateway_handler_responses.go）

同模式：line 67 `ReadRaw()` → line 81/127 `SetEffectiveBytes`。**仅 Antigravity 分支仍传 `body`（line 295）——对齐**。

### 3.5 Gemini（gemini_v1beta_handler.go）

line 211 `ReadRaw()` → line 224 `SetEffectiveBytes`；line 377-401 有二次 ReadRaw（CleanGeminiNativeThoughtSignatures 路径）。**主转发分支已直接用 handle（line 475）——仅清残留引用**。

### 3.6 multipart 媒体（openai_images.go / grok_media.go）

- 大文件 part 已由标准库落盘（`ParseMultipartForm` 超限写临时文件）
- **Images 已向 service 传 nil（line 349）——验证现有模式即可**
- **Grok 需处理 session/multipart 二次转换**（MAJOR）

## 4. WS 分支（明确排除，列为后续）

`forwardOpenAIWSV2` 持有 `wsReqBody map[string]any` 跨重连循环（`recoverPrevResponseNotFound`/`recoverInvalidEncryptedContent` 原地改 map 重试）。map 内存 3-5× []byte。

**排除理由**：
1. WS 需账号显式启用（`responses_websockets_v2`），流量占比小
2. 恢复逻辑依赖 map 原地改写（删 previous_response_id / reasoning items），改 handle 化需重写恢复语义
3. 本次改动面已足够大，WS 单列后续 change

## 4b. Preview 与观测契约（CRITICAL-3 处理）

**问题**：preview limit 5MB > 新 spool 阈值 1MB，1MB-5MB 请求的 preview 字符串就是完整 body 副本，直接违反"零完整副本"约束。

**决策**：**降低 preview limit 至 256KB**（`defaultRequestBodyPreviewLimitBytes` 与 `openAIResponsesRequestBodyPreviewLimitBytes` 同步）。理由：
- preview 用途是 usage detail 展示请求摘要，256KB 足够（现有 ops 单字段 20KB 上限，preview 已远超）
- 5MB→256KB 使 256KB-5MB 请求不再产生完整副本
- 观测契约变化需同步 delta spec（preview 上限从 5MB 调整为 256KB）

**风险**：usage detail 展示的请求预览变短；`usage_log_details` 中已存的历史 preview 不受影响。

## 5. 风险与缓解

| 风险 | 缓解 |
|---|---|
| failover 派生时机改变导致状态机回归 | `deriveOpenAIForwardAttemptBody` 的派生逻辑不动，仅 canonical 来源变化；现有 failover 测试矩阵（credential_failover / passthrough / reasoning failover）覆盖 |
| session hash 语义漂移（normalize 前后 body 不同） | hash 必须基于 normalize 前内容 → 从 raw handle 物化计算，保持现状语义；有 `openai_gateway_request_body_retention_test.go` 验证 |
| usage hash 口径变化（计费去重） | Chat 的 `HashUsageRequestPayload` 必须在 channel mapping 前对原始 body 预计算 |
| CyberSessionBlockKey 语义变化（Grok header） | 用 `explicitOpenAISessionID` 口径预计算完整 key，body 尚存时完成 |
| 每轮 ReadAll 增加 CPU（8.9MB 文件读 1-2ms） | 仅在 failover/retry 发生时多读；无 failover 单次请求与现状等价（首轮物化后直接复用，service 层首轮不重复物化） |
| 误释放仍被引用的 body | 逐入口核实引用链，循环内物化的局部变量作用域严格限定在单轮 attempt |
| service 层 retry 改写后重建 handle 失败 | 沿用现有 spool 错误映射（503）；重建失败不得回退 byte 路径 |
| preview 降低影响观测 | delta spec 同步 preview 上限；usage 展示变短可接受 |

## 6. 测试策略

1. **定向矩阵**（现有测试文件扩展）：
   - `gateway_request_body_spooling_test.go`：阈值边界（1MB 上下）、落盘/清理
   - `openai_gateway_request_body_retention_test.go`：session hash / usage hash 语义不变、body 释放时机
   - failover 系列（`openai_gateway_credential_failover_loop_test.go` 等）：派生一致性、service retry 重建 handle 正确性
   - `request_body_size_matrix_test.go`：**新增 0.5MB/1MB/1MB+1/2MB/10MB 默认阈值矩阵**（现固定 10MB）
   - **阻塞 transport 测试**：transport 读完 body 后阻塞，2MB/8.9MB 请求执行 GC/heap 检查（2MB 用例暴露 preview 完整副本问题）
2. **全量**：`go test ./... -count=1`
3. **受控验证**：生产或 staging 用 5MB/10MB/12MB 请求，对照上游接收 hash、usage、spool 目录、容器 RSS 采样

## 7. 实施顺序

1. ① spool 阈值 1MB + preview 256KB（常量导出 + 测试同步）——独立可验证，先落地
2. ②a Responses handler 循环内物化 + final handle 重建（最复杂，先做）
3. ②b service 层 Forward retry 循环 handle 化（invalid_encrypted_content / rejected field / 计费提取）
4. ②c ChatCompletions 对齐（含 usage hash / cyber key 预计算）
5. ②d Anthropic Messages / Responses 兼容对齐（含 ParsedRequest byte-owned 修复）
6. ②e Gemini 对齐（清残留引用）
7. ②f multipart 媒体（Images 验证 / Grok session 处理）
8. 全量测试 + 阻塞 transport 内存验证 + 受控验证
