## Why

本次 OOM 调查依赖 SSH 手工查看 RSS、swap、Docker stats、OOM 日志和应用日志聚合，定位成本高且滞后。平台需要把内存压力、大请求、审计截断和 usage 队列堆积转成可观测、可告警的运行时信号。

## What Changes

- 增加内存压力采集：进程 RSS/heap、cgroup memory、swap、Go GC 指标。
- 增加大请求与大响应相关指标：请求体大小分桶、超过阈值计数、内容审计截断计数、SSE 行超限计数。
- 增加 usage 记录池水位告警：等待任务、worker 数、同步 fallback、drop 计数。
- 接入完整告警：复用现有 ops/通知能力，支持阈值配置、告警触发、恢复和后台查看。

## Capabilities

### New Capabilities
- `memory-pressure-alerting`: 采集并告警 sub2api 内存压力和大请求相关运行时风险。

### Modified Capabilities

## Impact

- 影响后端 ops 指标采集、告警评估、通知发送和管理端展示。
- 可能需要新增配置项或 setting key；尽量复用现有 ops 表和通知服务。
- 不改变网关请求转发行为，不改变计费逻辑。
