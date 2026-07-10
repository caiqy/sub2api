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
