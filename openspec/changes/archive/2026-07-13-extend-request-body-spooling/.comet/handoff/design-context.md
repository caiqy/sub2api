# Comet Design Handoff

- Change: extend-request-body-spooling
- Phase: design
- Mode: compact
- Context hash: f2f63222f5d48a82eed63062307adc59083d02e664f26074e2d15c5068fafa6d

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/extend-request-body-spooling/proposal.md

- Source: openspec/changes/extend-request-body-spooling/proposal.md
- Lines: 1-25
- SHA256: 83a17f9717df462780456e5f38a95ff193a734cd896efb09ebbdfd2c74d0fc92

```md
## Why

现有 request body 驻留优化只在 OpenAI/Grok `/responses` 入口启用 10MB spool，其余协议仍会在 handler 入口把完整大 body 读入内存，并在上游等待、重试或 failover 期间形成长生命周期引用。需要将同一套文件化请求体和有界观测契约扩展到所有剩余入口，避免不同协议在小内存部署中继续出现不一致的内存放大。

## What Changes

- 引入共享的入站 request body coordinator，统一处理 identity/压缩 JSON、10MB spool、5MB preview、hash、重复打开和 cleanup。
- 将 spool 覆盖扩展到 Anthropic 分组 `/v1/responses` 兼容转换、Anthropic `/v1/messages`、OpenAI `/v1/chat/completions`、OpenAI Embeddings、OpenAI/Grok Images 与 Videos、Gemini `/v1beta/models/*`。
- 压缩 JSON 按解压后的有效 body 大小判断 spool；multipart 按原始请求体大小处理，并复用标准库临时文件能力承载上传内容。
- 所有入口在成功、失败、取消、retry 和 failover 后清理临时文件；spool I/O 失败统一返回 503，不回退到长期持有完整大 body。
- 保持现有 API、内容审计、模型映射、计费、usage 指纹、上游请求、流式终止和错误透传语义不变。

## Capabilities

### New Capabilities

### Modified Capabilities

- `request-body-retention-control`: 将大请求文件化和可重放要求从 `/responses` 扩展到所有支持大 body 的网关协议及媒体入口，并明确压缩、multipart 和生命周期语义。

## Impact

- 影响 Anthropic/OpenAI/Grok/Gemini handler、协议转换服务、上游 request builder、usage detail 与 ops capture。
- 复用现有 `RequestBodyHandle` 和 Go 标准库，不新增第三方依赖、配置项或数据库 schema。
- 需要跨协议成功、失败、取消、压缩、multipart、retry/failover 和临时文件清理回归测试。

```

## openspec/changes/extend-request-body-spooling/design.md

- Source: openspec/changes/extend-request-body-spooling/design.md
- Lines: 1-73
- SHA256: d60b1314c5954626a0b4791e05d7cace3513a683cee0a38d47590a735c9b5747

```md
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

```

## openspec/changes/extend-request-body-spooling/tasks.md

- Source: openspec/changes/extend-request-body-spooling/tasks.md
- Lines: 1-28
- SHA256: 7224a088e5e59ffcf471c7f0d56a7306173c7b7a227db7e4d27572fdd5cb5759

```md
## 1. 共享 request body coordinator

- [ ] 1.1 为 identity、gzip 等压缩 JSON 编写阈值、64MB 解压上限、preview、hash、503 和 cleanup 的失败优先测试。
- [ ] 1.2 实现共享 coordinator，使解压流直接进入 `RequestBodyHandle`，并支持 raw/effective handle 复用与显式 ownership。
- [ ] 1.3 增加成功、业务拒绝、客户端取消、panic recovery、handle 替换和 stale cleanup 生命周期测试。

## 2. Anthropic 与 OpenAI JSON 入口

- [ ] 2.1 将 Anthropic 分组 `/v1/responses` 兼容转换和 `/v1/messages` 迁移到 coordinator，保持内容审计、转换、计费和流式语义。
- [ ] 2.2 将 OpenAI `/v1/chat/completions` 与 Embeddings 迁移到 coordinator，使最终 outbound body 通过 effective handle 支持 retry/failover 重放。
- [ ] 2.3 为四类入口补充小请求、大请求、压缩请求、上游 4xx/5xx、取消、retry/failover 和 usage/ops snapshot 回归测试。

## 3. Gemini 原生入口

- [ ] 3.1 将 Gemini `/v1beta/models/*` 的 generateContent、streamGenerateContent 与 countTokens 请求迁移到 coordinator。
- [ ] 3.2 验证 Gemini 模型路径解析、内容审计、Google 错误格式、流式终止、failed usage 和 Antigravity 强制平台路由保持不变。

## 4. OpenAI/Grok 媒体入口

- [ ] 4.1 为 JSON、multipart、inline binary 和源图/遮罩上传编写 spool、脱敏 metadata、完整上游请求和 cleanup 测试。
- [ ] 4.2 将 OpenAI/Grok Images 与 Videos 的 raw multipart 和 effective outbound multipart 接入 coordinator，并统一 `RemoveAll` 与 handle cleanup。
- [ ] 4.3 验证生成、编辑、视频创建、视频状态、业务拒绝、上游错误和重试路径不泄露二进制正文且不残留临时文件。

## 5. 全链路验证

- [ ] 5.1 增加跨协议契约测试，确认所有目标入口使用共享 coordinator，usage/ops 不回读完整 body，spool I/O 失败统一返回 503。
- [ ] 5.2 运行后端全量测试、前端全量测试与类型检查，并执行 5MB/10MB/12MB identity、gzip、multipart 受控端侧矩阵。
- [ ] 5.3 对照容器 RSS、spool 生命周期、usage detail 和 ops 数据，确认进入上游等待后无长期大 `[]byte` 引用且业务语义无回归。

```

## openspec/changes/extend-request-body-spooling/specs/request-body-retention-control/spec.md

- Source: openspec/changes/extend-request-body-spooling/specs/request-body-retention-control/spec.md
- Lines: 1-67
- SHA256: fc89bb3ecf756e7656bb41d877fd9b35abb8271d0fb06724f1c5c9d23d3fe9bb

```md
## MODIFIED Requirements

### Requirement: 系统必须限制 request body 在观测链路中的长生命周期副本
系统 MUST 在所有支持 request body 的网关入口中限制 request body 和 upstream request body 在 usage detail、ops context 和异步 usage 快照中的保留方式；观测链路不得继续持有完整超大 body 副本。

#### Scenario: 普通请求进入 usage detail
- **WHEN** 网关处理任一受支持协议请求并构建 usage detail
- **THEN** 系统 MUST 记录 request/upstream body 的有界 preview、完整大小和截断状态，而不是再次完整复制 body

#### Scenario: 超大请求进入 usage detail
- **WHEN** 任一受支持入口的请求体大小超过 preview 上限
- **THEN** 系统 MUST 只保存 preview 或安全省略标记，并记录 `truncated` 与原始大小

#### Scenario: multipart 或 inline binary 请求进入观测链路
- **WHEN** Images、Videos 或其他入口收到 multipart、base64、data URL 或 inline binary 内容
- **THEN** 系统 MUST 只保存脱敏 metadata 或安全省略标记，不得将二进制正文写入 usage detail 或 ops

### Requirement: 系统必须对大请求使用可重放的文件化请求体
系统 MUST 在 Anthropic 分组 `/v1/responses` 兼容转换、Anthropic `/v1/messages`、OpenAI `/v1/chat/completions`、OpenAI Embeddings、OpenAI/Grok Images 与 Videos、Gemini `/v1beta/models/*` 的有效 body 超过 `spool threshold` 时使用临时文件承载完整请求体，并支持 failover 或 retry 时重新打开完整 reader。

#### Scenario: 大 JSON 请求等待上游响应
- **WHEN** identity 编码 JSON 的有效 body 超过 `10MB`，且请求正在等待上游响应
- **THEN** 系统 MUST 让完整 body 主要驻留在临时文件，并释放同步解析阶段不再需要的完整内存副本

#### Scenario: 压缩大请求进入网关
- **WHEN** 压缩 JSON 解压后的有效 body 超过 `10MB` 且未超过解压安全上限
- **THEN** 系统 MUST 将解压流写入文件型 handle，不得先长期保留完整解压 `[]byte`

#### Scenario: multipart 大请求进入媒体入口
- **WHEN** OpenAI/Grok Images 或 Videos 收到超过 `10MB` 的 multipart 请求
- **THEN** 系统 MUST 使用文件承载大上传内容，并保持文本字段、文件 part 和上游请求语义不变

#### Scenario: failover 或 retry 需要重发请求
- **WHEN** 上游失败导致同一有效 body 需要再次发送
- **THEN** 系统 MUST 从 effective outbound handle 重新打开完整 reader，并保持发送内容一致

#### Scenario: 文件化失败
- **WHEN** 超过 `spool threshold` 的请求无法创建、写入、关闭、打开或读取临时文件
- **THEN** 系统 MUST 返回 503，并不得静默回退到继续持有完整 body 的高内存路径

### Requirement: 系统必须保持 usage 指纹和业务语义不变
系统 MUST 保持各受支持入口的内容审计、协议解析、模型映射、上游转发、流式终止和 usage 指纹语义不变；大请求优化不得改变计费、usage 去重、retry 或 failover 行为。

#### Scenario: 同步解析后创建 effective body
- **WHEN** handler 完成请求校验、内容审计或协议转换并生成最终上游 body
- **THEN** effective handle 的内容、hash 和上游发送结果 MUST 与优化前相同

#### Scenario: 小请求继续使用内存模式
- **WHEN** 有效 body 不超过 `10MB`
- **THEN** 系统 MUST 保持现有成功、错误、流式和计费行为，且无需创建 spool 文件

## ADDED Requirements

### Requirement: 系统必须在所有终止路径释放 request body 临时资源
系统 MUST 明确管理 raw inbound handle、effective outbound handle 和 multipart 临时文件的 ownership，并在资源不再使用时完成清理。

#### Scenario: 请求成功或上游返回错误
- **WHEN** 请求成功完成或以上游 4xx/5xx 结束
- **THEN** 系统 MUST 清理该请求拥有的所有 spool 和 multipart 临时文件

#### Scenario: 客户端取消或 handler 提前返回
- **WHEN** 客户端取消、业务校验失败、路由失败或 panic recovery 导致 handler 提前结束
- **THEN** 系统 MUST 清理已创建的临时资源且不得影响错误响应语义

#### Scenario: retry 替换 effective body
- **WHEN** 模型映射、协议转换或 retry 为下一 attempt 创建新的 owned effective handle
- **THEN** 系统 MUST 在旧 handle 不再被请求使用后清理旧资源，并保留新 handle 直到 attempt 完成

```
