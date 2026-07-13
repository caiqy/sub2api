## ADDED Requirements

### Requirement: 系统必须采集内存压力指标
系统 MUST 周期性采集 Go 进程、容器或宿主相关的内存压力指标，并在采集失败时保留可诊断状态。

#### Scenario: 采集 Go 进程内存
- **WHEN** Ops 指标采集任务运行
- **THEN** 系统 MUST 记录 heap、RSS 或可获得的等价进程内存指标

#### Scenario: cgroup 指标不可用
- **WHEN** 当前部署环境无法读取 cgroup memory 文件
- **THEN** 系统 MUST 继续采集可用的 runtime 指标，并标记 cgroup 指标不可用

### Requirement: 系统必须记录大请求风险信号
系统 MUST 记录请求体大小分桶、内容审计截断次数、SSE 行超限次数和 usage 记录池水位，不得记录请求正文或图片内容。

#### Scenario: 请求体超过大请求阈值
- **WHEN** 网关处理请求体大小超过配置阈值的请求
- **THEN** 系统 MUST 增加大请求计数，并记录 endpoint、model、stream、user_id、api_key_id、group_id 等非正文标签

#### Scenario: 内容审计发生截断
- **WHEN** 内容审计抽取因文本或图片上限丢弃部分输入
- **THEN** 系统 MUST 记录截断计数和截断类型

### Requirement: 系统必须对内存压力触发告警
系统 MUST 支持配置内存压力和大请求风险告警阈值，并在阈值满足时触发告警和恢复状态。

#### Scenario: swap 或 RSS 超过阈值
- **WHEN** 采集到的 swap 或 RSS 指标超过配置阈值并持续达到告警条件
- **THEN** 系统 MUST 产生内存压力告警并通过已配置通知渠道发送

#### Scenario: 指标恢复正常
- **WHEN** 已触发的内存压力告警对应指标恢复到恢复阈值以下
- **THEN** 系统 MUST 标记告警恢复，并按配置发送恢复通知
