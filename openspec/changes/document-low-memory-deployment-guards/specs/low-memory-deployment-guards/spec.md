## ADDED Requirements

### Requirement: 文档必须提供低内存部署保护配置
系统文档 MUST 为小内存部署提供一组可操作的内存保护配置建议，并说明每项配置的效果和副作用。

#### Scenario: 管理员部署到 2GiB 级别服务器
- **WHEN** 管理员查看低内存部署文档
- **THEN** 文档 MUST 提供 Nginx 请求体上限、Go runtime 内存参数、上游响应上限、SSE 行上限和 usage worker 池参数建议

#### Scenario: 管理员评估是否调大阈值
- **WHEN** 管理员需要支持更大的合法请求或响应
- **THEN** 文档 MUST 说明调大相关阈值的内存风险和验证步骤

### Requirement: 文档必须提供上线后验证清单
系统文档 MUST 提供低内存配置上线后的只读验证清单，帮助管理员确认配置生效并观察内存压力。

#### Scenario: 配置上线后检查内存
- **WHEN** 管理员完成低内存配置并重启服务
- **THEN** 文档 MUST 指导管理员检查容器 RSS、swap、OOM 日志和应用大请求日志聚合
