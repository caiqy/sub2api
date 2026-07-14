## 1. 回归证据与平台 Sticky 修复

- [x] 1.1 运行既有平台 Sticky、模型路由和 hydration 失败聚焦用例，记录预期失败。
- [x] 1.2 在当前 GatewayService 调度路径恢复平台 Sticky 守卫、bypass 日志和无并发服务时的 routing account 选择。

## 2. 槽位释放修复与验证

- [x] 2.1 删除 OpenAI hydration wrapper 的重复 release，保留通用选择结果的单一所有权。
- [x] 2.2 运行聚焦服务测试和受影响 package 测试，确认 Sticky 默认启用、平台隔离、模型路由与单次 release。

## 3. 提交

- [x] 3.1 提交该独立 Hotfix 修复。
