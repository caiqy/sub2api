## Context

现有 OpsMetricsCollector 已周期采集系统、DB、Redis、usage 统计等指标，但本次问题暴露出缺少面向 Go 进程与大 input 的内存压力信号。线上需要在接近 OOM 前看到趋势，并能定位是大请求、内容审计、usage 队列还是宿主机内存不足导致。

## Goals / Non-Goals

**Goals:**
- 采集进程和容器内存压力指标，并进入现有 ops 指标链路。
- 对大请求分桶、审计截断、usage 任务堆积建立阈值告警。
- 后台能查看最近告警和关键指标，通知渠道复用现有邮件/站内或已有告警通知能力。

**Non-Goals:**
- 不实现新的监控系统或外部 Prometheus 依赖。
- 不修改请求处理逻辑；只观测和告警。
- 不把请求正文、图片 base64 或敏感 prompt 写入指标/告警。

## Decisions

- 指标采集优先复用 OpsMetricsCollector，新增轻量采样项：runtime MemStats、RSS/cgroup memory、swap、usage worker pool stats。这样少建新后台任务。
- 大请求分桶在网关读取 body 后记录尺寸和标签，不记录正文；标签限制为 endpoint、model、stream、user_id/api_key_id/group_id 的 ID 维度，避免敏感内容。
- 告警评估复用现有 OpsAlertEvaluatorService 或同类 setting 驱动机制。新增阈值包括 RSS 占宿主/容器比例、swap 使用、body size 分桶突增、usage waiting tasks、sync fallback/drop 增长。
- 后台展示先做最小可用：指标卡片 + 最近告警列表 + 阈值配置入口。更复杂的趋势图等后续再做。

## Risks / Trade-offs

- 指标维度过多会增加存储和查询压力 → 控制 label 维度，不记录 request_id 级别指标。
- cgroup 信息在不同部署环境路径不同 → 读取失败时降级为 runtime/进程指标并记录采集状态。
- 告警太敏感会产生噪声 → 默认阈值保守，支持冷却时间和恢复通知。
