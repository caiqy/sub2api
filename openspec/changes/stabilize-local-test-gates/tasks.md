## 1. 回归测试

- [x] 1.1 固化活动 spool reader 与 cleanup 并发的确定性回归测试，并保留已观测的 first-output keepalive RED 证据

## 2. 根因修复

- [x] 2.1 实现 spool reader 延迟删除生命周期，并修正 keepalive ticker 的 downstream 空闲计时基准

## 3. 规范与验证

- [x] 3.1 补足 `local-test-gates` Purpose、稳定 Server-Timing 时序测试，并运行定向重复测试、后端与前端全量测试及 OpenSpec strict 校验
