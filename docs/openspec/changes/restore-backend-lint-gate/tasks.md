## 1. 固定基线与回归合同

- [x] 1.1 在实施分支上复核已固定的 144 项 baseline manifest、39 个文件、分类计数和受保护文件 blob，确认相对 `b576f73a` 未漂移
- [x] 1.2 运行请求体内存保留、spool、retry/failover 聚焦测试，建立修复前的绿色行为基线与 lint RED

## 2. 修复入口层 lint

- [x] 2.1 修复 handler 与 routes 中的无效局部 body 清零，保留可观察的 ownership/结构体字段清理，并通过 scoped lint 与 handler 测试
- [x] 2.2 将生产与测试文件中的 3 项 QF1003 改为语义等价 tagged switch，并通过相关 handler 测试

## 3. 修复服务层 lint

- [ ] 3.1 修复通用 Gateway、Anthropic、Bedrock 与 Antigravity 路径及其测试源中的无效局部 body 清零，并通过 manifest 闭包、package 测试和内存保留矩阵
- [ ] 3.2 修复 OpenAI、Gemini、Grok 适配路径中的无效局部 body 清零，并通过 scoped lint、package 测试和 retry/failover 聚焦测试
- [ ] 3.3 删除确认无调用方的 `sendCCUpstreamRequest` 私有方法，并完成三批 changed-file allowlist 与 issue manifest 交叉检查

## 4. 恢复全仓绿色门禁

- [ ] 4.1 运行 gofmt、`git diff --check` 与 uncapped golangci-lint，确认 144 项全部关闭且结果为 0 issues
- [ ] 4.2 重跑 backend 默认/unit 测试及请求体内存保留矩阵，确认请求体 ownership、spool、retry/failover 和上游等待语义未回退
- [ ] 4.3 运行仓库级 `make test` 并记录 exit 0，形成供后续 upstream merge change 更新 source base 的验证证据
