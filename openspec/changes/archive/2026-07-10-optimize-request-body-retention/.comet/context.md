# Comet Design Handoff

- Change: optimize-request-body-retention
- Phase: design
- Mode: compact
- Context hash: 4d7d82dcf958044afb4abdadf0fdbe32f3b63378736851c4ed9f05ac40a822d1

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/optimize-request-body-retention/proposal.md

- Source: openspec/changes/optimize-request-body-retention/proposal.md
- Lines: 1-23
- SHA256: 10da90b28caacf9fa1ad13de07184f64f614a11a22c53000ed6b7c7206b803ee

```md
## Why

大 `/responses` 请求目前会在 request capture、handler、upstream request capture、ops context 和 usage snapshot 之间产生多份完整 body 副本。上游响应常常需要 10 秒以上，导致完整 request body 在等待期长时间驻留内存，并继续以 usage detail / ops 形式保存在数据库中。对于 50-80MiB 请求，这已经成为独立于内容审计之外的内存和存储放大来源。

## What Changes

- 引入统一的 request body handle，集中管理完整 body、preview、hash 和生命周期。
- 所有 `/responses` 请求都改为只在观测链路保留有界 preview，而不保留完整 request/upstream body 副本。
- 对 `>10MB` 的有效转发 body 启用临时文件模式，让上游发送和重试从文件重开 reader。
- 将 `requestPayloadHash` 提前到转发前计算，避免为了 usage 记录让 handler 长时间持有完整 body。

## Capabilities

### New Capabilities
- `request-body-retention-control`: 控制 `/responses` 请求体在网关、观测和上游转发链路中的驻留方式，降低长时间内存占用。

### Modified Capabilities

## Impact

- 影响 OpenAI `/responses` handler、usage detail capture、ops upstream request capture 和上游 request builder。
- 不改变内容审计语义、不改变上游请求语义、不改变计费与 usage 指纹语义。
- 不新增数据库 schema；超大 body 的完整正文不再进入 usage detail。

```

## openspec/changes/optimize-request-body-retention/design.md

- Source: openspec/changes/optimize-request-body-retention/design.md
- Lines: 1-28
- SHA256: 033c2cf26b7a67a2fe067e8335bc2722978d0dd69726be392ff0534dd297d13f

```md
## Context

当前 `/responses` 大请求的内存问题已经不只来自内容审计和 usage worker。完整 request body 会在 middleware、handler、upstream capture、ops context 和 async usage snapshot 中形成长生命周期副本，尤其在上游响应耗时 10 秒以上时，对小内存机器压力很大。

## Goals / Non-Goals

**Goals:**
- 所有 `/responses` 请求都减少 request/upstream body 的观测副本和长生命周期引用。
- `>10MB` 请求在等待上游期间，把完整 body 从 RAM 下沉到临时文件。
- 保持 `requestPayloadHash`、内容审计、上游转发和 failover 语义不变。

**Non-Goals:**
- 不做流式 JSON 解析。
- 不处理 response body / upstream response body 的完整捕获。
- 不扩展到 Anthropic/Gemini 等其他协议入口。

## Decisions

- 使用一个小型 `RequestBodyHandle` 统一承载 body 来源、preview、hash 和重复打开能力。
- request capture 和 upstream capture 只保留 preview，不再各自完整 `ReadAll` body。
- `spool threshold=10MB`，`preview limit=5MB`；两者分离。
- 大 body 文件化失败时返回 `503`，不静默回退到高内存路径。

## Risks / Trade-offs

- usage detail 不再保留完整超大 body，管理员排障信息减少，但换来可控内存与存储成本。
- 文件化路径引入磁盘依赖和清理逻辑，但比依赖 swap 更可控。
- 非透传路径的 `map[string]any` 仍会在同步处理窗口内存在；首版先解决长生命周期完整 body。

```

## openspec/changes/optimize-request-body-retention/tasks.md

- Source: openspec/changes/optimize-request-body-retention/tasks.md
- Lines: 1-18
- SHA256: 12ca9443faea3f0c1c7369110282563e17aa074de5e833870c5027fe9dd92c88

```md
## 1. Request body handle

- [x] 1.1 实现 `RequestBodyHandle`，支持内存模式、文件模式、preview、hash、size、重复 `Open()` 和 `Cleanup()`。
- [x] 1.2 增加 stale spool 文件清理与文件化失败测试。

## 2. `/responses` 请求链路改造

- [x] 2.1 改造 `OpenAIGatewayHandler.Responses`，引入 raw/effective handle，提前计算 `requestPayloadHash`。
- [x] 2.2 改造 usage detail request body capture，不再在 middleware 预读完整 request body。
- [x] 2.3 改造 upstream request capture 和 ops upstream capture，只保留 preview。
- [x] 2.4 改造 OpenAI upstream request builder，使 `>10MB` body 通过文件型 `GetBody` 重放。

## 3. 验证

- [x] 3.1 增加 `RequestBodyHandle` 单元测试，覆盖内存/文件模式和 cleanup。
- [x] 3.2 增加 `/responses` 回归测试，确认 `requestPayloadHash` 与现状一致。
- [x] 3.3 增加大 body 路径测试，确认 usage/ops 只保留 preview，failover / retry 内容不变。
- [x] 3.4 增加 spool 创建失败测试，确认返回 `503` 且不回退到高内存路径。

```

## openspec/changes/optimize-request-body-retention/specs/request-body-retention-control/spec.md

- Source: openspec/changes/optimize-request-body-retention/specs/request-body-retention-control/spec.md
- Lines: 1-34
- SHA256: 0c7d9ea22551a156a16281cc1b0158a74f085b08b9cbac11c2e367f675790114

```md
## ADDED Requirements

### Requirement: 系统必须限制 request body 在观测链路中的长生命周期副本
系统 MUST 在 `/responses` 请求处理中限制 request body 和 upstream request body 在 usage detail、ops context 和异步 usage 快照中的保留方式；观测链路不得继续持有完整超大 body 副本。

#### Scenario: 普通请求进入 usage detail
- **WHEN** 网关处理 `/responses` 请求并构建 usage detail
- **THEN** 系统 MUST 记录 request/upstream body 的有界 preview、完整大小和截断状态，而不是再次完整复制 body

#### Scenario: 超大请求进入 usage detail
- **WHEN** `/responses` 请求体大小超过 preview 上限
- **THEN** 系统 MUST 只保存 preview，并标记 `truncated` 与原始大小

### Requirement: 系统必须对大请求使用可重放的文件化请求体
系统 MUST 在有效转发 body 超过 `spool threshold` 时使用临时文件承载完整请求体，并支持 failover 或 retry 时重新打开完整 reader。

#### Scenario: 大请求等待上游响应
- **WHEN** 有效转发 body 超过 `10MB`，且请求正在等待上游响应
- **THEN** 系统 MUST 让完整 body 主要驻留在临时文件，而不是继续依赖 RAM 中的完整副本

#### Scenario: failover 或 retry 需要重发请求
- **WHEN** 上游失败导致同一有效 body 需要再次发送
- **THEN** 系统 MUST 能从文件型请求体重新打开完整 reader，并保持发送内容一致

### Requirement: 系统必须保持 usage 指纹和业务语义不变
系统 MUST 保持 `/responses` 请求的内容审计、上游转发和 `requestPayloadHash` 语义不变；大请求优化不得改变计费、usage 去重或 failover 行为。

#### Scenario: 提前计算 requestPayloadHash
- **WHEN** handler 在转发前为有效 body 计算 `requestPayloadHash`
- **THEN** 计算结果 MUST 与优化前对同一有效转发 body 的 hash 语义一致

#### Scenario: 文件化失败
- **WHEN** 超过 `spool threshold` 的请求无法创建或写入临时文件
- **THEN** 系统 MUST 返回服务端错误，并不得静默回退到继续持有完整 body 的高内存路径

```
