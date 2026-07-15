## 1. 恢复稳定 unit 测试

- [x] 1.1 将 Anthropic failed-usage 测试改为按 HTTP header 语义断言，并运行对应 handler 测试。
- [ ] 1.2 修复 Images failover 耗尽时的错误响应契约，并运行对应 handler 测试。
- [ ] 1.3 让 server 与 middleware 测试 fixture 实现当前 `UserRepository` 接口，并运行两个 package 的 unit 测试。

## 2. 确认 request body spool 生命周期

- [ ] 2.1 通过重复与并行运行复现或排除 OpenAI request body spool cleanup 失败。
- [ ] 2.2 仅在稳定复现确认后修复资源所有权边界，并运行相关 service 测试。

## 3. 修复静态检查

- [ ] 3.1 修复依赖边界、资源关闭和 context cancel 诊断，并运行受影响 package 测试与 lint。
- [ ] 3.2 修复无效赋值、静态分析和未使用符号诊断，不降低 lint 规则。

## 4. 固化并验证本地门禁

- [ ] 4.1 更新本地测试入口以覆盖后端默认测试、后端 unit、lint 和前端全量验证，不调用 integration/e2e。
- [ ] 4.2 从冷缓存运行全部本地门禁并记录结果。
