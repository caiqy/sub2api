## Why

同一套默认配置需要服务从小内存 VPS 到高并发生产环境。dmit 这类 1.7GiB 内存机器必须有明确的低内存部署保护建议，否则即使代码优化后仍可能因大 input、非流式响应或 worker 默认值偏大而接近 OOM。

## What Changes

- 增加低内存部署保护文档，说明 Nginx 请求体上限、`GOMEMLIMIT`、`GOGC`、上游响应读取上限、SSE 行上限、usage worker 池参数的推荐值和副作用。
- 增加配置示例，覆盖 dmit 这类小内存单机部署。
- 增加验证清单，指导上线后检查 RSS、swap、OOM 日志、应用大请求日志和告警。

## Capabilities

### New Capabilities
- `low-memory-deployment-guards`: 为小内存部署提供可操作的配置保护和验证方法。

### Modified Capabilities

## Impact

- 影响部署文档、知识库和示例配置。
- 不修改运行时代码，不改变默认配置。
- 可与代码优化、告警建设并行推进。
