## 1. 有效路由核心状态

- [x] 1.1 为最终 group、request-owned API key、显式 billing source、最终组 subscription 与 composite route model 编写 service 层 RED 测试
- [x] 1.2 实现 `EffectiveGatewayRoute`、共享 resolver、context helpers 和模型阶段 identity 规则
- [x] 1.3 为订阅分组缺少 subscription 时禁止退化为余额模式编写 RED 测试并收紧 `CheckBillingEligibility`
- [x] 1.4 将共享 resolver 接入 Wire，重新生成并确认生成结果稳定

## 2. 协议分发与 Handler 消费

- [x] 2.1 为 ClaudeCodeOnly fallback 后的最终平台分发、Gin API key/subscription、request group 与失败原子性编写 route-local RED 测试
- [x] 2.2 扩展现有 composite route middleware，在 endpoint protocol switch 前 resolve/validate/apply 有效路由
- [x] 2.3 让 `GatewayHandler`、`OpenAIGatewayHandler` 和 count_tokens 消费共享 snapshot，并移除重复的局部 effective group 推导
- [x] 2.4 增加非 composite identity、Messages/Responses/count_tokens 最终组一致性回归

## 3. Runtime fallback

- [x] 3.1 为 prompt-too-long 经过中间 ClaudeCodeOnly 组到最终订阅组编写 RED 测试，断言最终 subscription、无余额退化和无失败侧效应
- [x] 3.2 使用共享 resolver/Apply 替换 prompt-too-long 的中间组校验、局部 clone 和 `subscription=nil` 分支

## 4. 模型审计与 WebSocket bridge

- [x] 4.1 为无 channel mapping 的 public alias 到 concrete route model 编写 RED 测试，并修复 client/routing/upstream 三阶段审计
- [x] 4.2 为 HTTP bridge later-turn route rewrite 后的 account mapping 编写 RED 测试，并复用现有账号映射 helper 完成最小修复
- [x] 4.3 重跑普通 WebSocket later-turn account mapping、跨 provider 拒绝和 composite pricing/模型长度保护测试

## 5. 验证与关闭证据

- [x] 5.1 运行全部聚焦测试、`git diff --check` 与 strict OpenSpec validation
- [x] 5.2 连续运行两次 `make -C backend generate` 并确认第二次无 diff，再运行本地 `make test` 与 `make build`（`make test` 唯一失败已在原始上游 `v0.1.165` 复现，用户明确接受该基线例外；未记为测试通过）
- [x] 5.3 由 fresh reviewer 逐项核对原 `3 Important + 1 Minor`，确认无新的 Critical/Important，并记录供归档和恢复原 Task 21 使用的证据
