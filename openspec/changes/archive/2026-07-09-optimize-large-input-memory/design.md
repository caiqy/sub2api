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
