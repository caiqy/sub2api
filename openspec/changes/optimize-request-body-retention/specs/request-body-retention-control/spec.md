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
