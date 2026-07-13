# Comet Design Handoff

- Change: optimize-large-input-memory
- Phase: design
- Mode: compact
- Context hash: b6eb848787d2d1b0a90df9964c9c8246abd81fdaaebbb7660569caafe41100a8

Generated-by: comet-handoff.sh

OpenSpec remains the canonical capability spec. This handoff is a deterministic, source-traceable context pack, not an agent-authored summary.

## openspec/changes/optimize-large-input-memory/proposal.md

- Source: openspec/changes/optimize-large-input-memory/proposal.md
- Lines: 1-24
- SHA256: 9ac26bb195d7c9350dc6de0be815676f72ee9990ade127ec0d243be695ac6cbc

```md
## Why

dmit 线上 OOM 证据显示，超大 input 请求会在网关读取、内容审计抽取、base64 图片处理和 usage 记录任务提交期间产生多份短生命周期副本，导致 `sub2api` 在小内存机器上出现高 RSS 与 swap 压力。Nginx 已负责 80MiB 请求体上限，本变更聚焦程序内部在该上限内的内存放大问题。

## What Changes

- 调整内容审计输入抽取流程，对文本和图片在收集阶段提前截断/限量，避免先收集完整大对象再裁剪。
- 降低 inline/base64 图片在审计路径中的重复字符串构造，超过审计需要的图片不进入后续审核 payload。
- 优化 usage 记录任务提交时的捕获对象，避免 worker 队列持有不必要的大对象引用。
- 保留现有业务行为：大请求是否允许通过仍由入口层/Nginx 与既有 `gateway.max_body_size` 决定。

## Capabilities

### New Capabilities
- `large-input-memory-control`: 降低大 input 请求在网关、内容审计和 usage 记录链路中的内存放大。

### Modified Capabilities
- `content-moderation-config`: 内容审计在处理大输入时必须保持审计语义，同时限制临时内存增长。

## Impact

- 影响后端 Go 网关请求处理、内容审计输入抽取、OpenAI Responses usage 记录提交路径。
- 不新增外部依赖，不改变数据库 schema，不改变公开 API。
- 需要增加针对大文本、多图/base64、usage 队列捕获对象的单元或回归测试。

```

## openspec/changes/optimize-large-input-memory/design.md

- Source: openspec/changes/optimize-large-input-memory/design.md
- Lines: 1-30
- SHA256: 0b9ef4cb9f87247a9055dd07977db34fa2025972ca3b7d5be70f3c1a0bb12476

```md
## Context

线上日志显示今天存在多个 `50MiB+` 的 `/v1/responses` 请求，最大约 `78MiB`。代码当前会一次性读取请求体，内容审计会从完整 body 中抽取文本和图片，抽取后再 normalize、hash、marshal；inline/base64 图片也会先完整进入 `Images` 后再限制数量。usage 记录任务虽然已有有界 worker 池，但闭包仍可能持有 result、account、snapshot 等较大的对象引用。

Nginx 已配置 80MiB 入口上限，因此本设计不再新增业务层大 input 拦截策略，而是减少已进入程序的合法大请求造成的堆内存峰值。

## Goals / Non-Goals

**Goals:**
- 在保持现有请求兼容性的前提下，减少内容审计路径对大文本和 base64 图片的重复分配。
- usage 记录任务只捕获记录所需的小型快照，降低队列等待期间的大对象保留。
- 为大输入内存优化留下最小可运行测试，覆盖文本截断、图片限量和 usage 任务引用释放边界。

**Non-Goals:**
- 不实现后台可配置的大 input 业务策略；入口限制继续由 Nginx 和现有 `gateway.max_body_size` 承担。
- 不重写网关请求体为流式 JSON 解析；这是更大改动，只有当前优化不足时再考虑。
- 不引入新的第三方内存池或对象池。

## Decisions

- 内容审计采用“收集时限量”而不是“收集后裁剪”。文本累积达到 `maxModerationInputRunes` 对应上限后停止追加旧内容或只保留最新片段；图片达到审核需要数量后停止构造 data URL。这样改动集中，避免全链路流式解析。
- 对 inline/base64 图片先做轻量长度判断，再决定是否构造 `data:<mime>;base64,...` 字符串。超过审核图片数量或明显超出审核价值的图片不进入审计 payload。
- usage 记录任务在提交前构造轻量输入快照，只保留 ID、计费字段、hash、必要 header 快照和小型 usage result。避免闭包直接捕获 `body`、完整 account 结构或 gin context 相关对象。
- 保持配置默认值不在本 change 中大幅调整。部署保护单独由 `document-low-memory-deployment-guards` 记录，避免代码优化与运维策略混在一起。

## Risks / Trade-offs

- 审计只处理截断后的文本和有限图片，可能减少对超大历史上下文的审计覆盖 → 保留最新/最相关内容，并通过日志/指标记录截断发生次数，后续由告警 change 观测。
- usage 记录快照化可能遗漏某些后续动态字段 → 先盘点 `RecordUsage` 真正使用的字段，测试覆盖计费、分组、渠道字段和图片计费字段。
- 不做流式 JSON 解析意味着请求体本身仍会完整读入内存 → 这次目标是消除额外放大，不解决基础读入成本。

```

## openspec/changes/optimize-large-input-memory/tasks.md

- Source: openspec/changes/optimize-large-input-memory/tasks.md
- Lines: 1-17
- SHA256: e5f2c72249ad6746e607f9c40aeed4f1bbdfdbc9a47803d617a1c008034389e1

```md
## 1. 内容审计大输入优化

- [ ] 1.1 为内容审计抽取增加大文本截断测试，覆盖超大 Responses input 只保留有限审计文本。
- [ ] 1.2 调整内容审计文本收集逻辑，在收集阶段限制累计文本规模。
- [ ] 1.3 为 inline/base64 图片增加限量测试，覆盖多图请求不会构造多余 data URL。
- [ ] 1.4 调整图片收集逻辑，在达到审核图片数量后停止构造额外图片字符串。

## 2. usage 任务内存保留优化

- [ ] 2.1 梳理 OpenAI usage 记录实际使用字段，定义轻量快照结构或提交参数。
- [ ] 2.2 调整 usage 任务提交闭包，避免捕获请求体、gin context 或完整大对象。
- [ ] 2.3 增加回归测试，确认 usage 记录计费字段、图片计费字段和渠道字段保持一致。

## 3. 验证

- [ ] 3.1 运行内容审计与 OpenAI usage 相关 Go 测试。
- [ ] 3.2 用合成大 input 测试验证临时分配和请求结果符合预期。

```

## openspec/changes/optimize-large-input-memory/specs/large-input-memory-control/spec.md

- Source: openspec/changes/optimize-large-input-memory/specs/large-input-memory-control/spec.md
- Lines: 1-32
- SHA256: dc33fbafeaa27a4d61684ec69db781e118d6ab8cb09bad949bd0d4a52837b8c1

```md
## ADDED Requirements

### Requirement: 大输入审计抽取必须限制临时内存放大
系统 MUST 在内容审计输入抽取阶段限制文本和图片的累计规模，避免为了审计构造超过审计需要的完整大对象副本。

#### Scenario: 超大文本请求进入内容审计
- **WHEN** 网关收到包含大量历史上下文的合法请求，且内容审计需要抽取文本
- **THEN** 系统只保留审计所需的有限文本片段，并继续完成请求处理

#### Scenario: 多个 inline base64 图片进入内容审计
- **WHEN** 请求包含多个 inline/base64 图片，且内容审计只需要有限图片样本
- **THEN** 系统 MUST 在收集阶段停止构造多余图片字符串

### Requirement: usage 记录任务不得保留不必要的大对象
系统 MUST 在提交异步 usage 记录任务前构造轻量快照，避免队列等待期间持有请求体、完整 gin context 或其他不必要的大对象引用。

#### Scenario: usage worker 队列存在等待任务
- **WHEN** 数据库写入变慢导致 usage 记录任务排队
- **THEN** 排队任务 MUST 只保留记录使用量所需的小型字段快照

## MODIFIED Requirements

### Requirement: 保存时清理已删除审计分组
内容审计配置保存时，系统 MUST 移除不存在的审计分组 ID，并保留仍存在的审计分组 ID。内容审计运行时处理大输入的优化不得改变该配置保存行为。

#### Scenario: 配置包含已删除分组
- **WHEN** 管理员保存内容审计配置，且配置中的指定审计分组包含已删除分组 ID
- **THEN** 系统保存配置成功，并从保存结果中移除已删除分组 ID

#### Scenario: 分组查询发生非不存在错误
- **WHEN** 管理员保存内容审计配置，且分组查询返回非不存在错误
- **THEN** 系统 MUST 返回错误且不保存配置

```
