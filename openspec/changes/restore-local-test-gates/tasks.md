## 1. 恢复稳定 unit 测试

- [x] 1.1 将 Anthropic failed-usage 测试改为按 HTTP header 语义断言，并运行对应 handler 测试。
- [x] 1.2 修复 Images failover 耗尽时的错误响应契约，并运行对应 handler 测试。
- [x] 1.3 让 server 与 middleware 测试 fixture 实现当前 `UserRepository` 接口，并运行两个 package 的 unit 测试。

## 2. 确认 request body spool 生命周期

- [x] 2.1 通过重复与并行运行复现或排除 OpenAI request body spool cleanup 失败。
- [x] 2.2 仅在稳定复现确认后修复资源所有权边界，并运行相关 service 测试。
- [x] 2.3 隔离 admin usage stats 测试的进程级缓存，并验证重复运行。
- [x] 2.4 释放 Grok edit 转发的大 body 局部引用，并确保失败断言也清理阻塞 goroutine。

## 3. 修复静态检查

- [x] 3.1 修复依赖边界、资源关闭和 context cancel 诊断，并运行受影响 package 测试与 lint。
- [x] 3.2 修复无效赋值、静态分析和未使用符号诊断，不降低 lint 规则。

## 4. 固化并验证本地门禁

- [x] 4.1 更新本地测试入口以覆盖后端默认测试、后端 unit、lint 和前端全量验证，不调用 integration/e2e。
- [x] 4.2 从冷缓存运行全部本地门禁并记录结果。
