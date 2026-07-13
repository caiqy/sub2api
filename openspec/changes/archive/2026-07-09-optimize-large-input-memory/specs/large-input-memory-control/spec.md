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
