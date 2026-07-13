## 1. Gateway 配置与运行时设置契约

- [x] 1.1 为生文和明确生图首 Token 超时补充配置加载测试，覆盖 30/600 秒默认值、环境变量、零值和负数拒绝
- [x] 1.2 在 `GatewayConfig`、运行时设置 view/update DTO 与持久化逻辑中增加两个超时字段，并保证旧 JSON 缺失字段时保留配置默认值
- [x] 1.3 补充运行时设置 API 测试，验证读取、更新和负数校验

## 2. 共享分类、事件判定与超时错误

- [x] 2.1 以表驱动测试定义严格请求分类：仅 `tool_choice.type=image_generation` 使用图片档，常驻图片工具仍使用文本档
- [x] 2.2 实现共享 Responses 事件判定，区分前导、首业务和终态事件，并覆盖图片 `output_item.added`、文本及工具输出
- [x] 2.3 增加专用首 Token 超时错误、watchdog 与结构化阶段信息，验证并发竞争只产生一个终态且错误不可 failover

## 3. HTTP SSE 首 Token 超时

- [ ] 3.1 为 passthrough 与协议转换 SSE 路径补充响应头前超时、前导事件后超时、业务事件停止计时和零值关闭测试
- [ ] 3.2 在上游请求 context 中接入共享 watchdog，缓冲前导事件，并在超时时返回 HTTP 504 `first_token_timeout`
- [ ] 3.3 验证超时写入失败 usage、Ops 错误和阶段日志，且不重试、不换号、不封禁账号

## 4. 池化 Responses WebSocket ingress 超时

- [ ] 4.1 补充 per-response timeout 测试，覆盖前导事件、业务事件、cancel/drain 成功复用和清理失败废弃连接
- [ ] 4.2 为每个已发送的 `response.create` 接入独立 deadline，超时时发送 `response.cancel` 和一次下游 error
- [ ] 4.3 实现有限 drain 与连接复用判定，并验证 timeout 不触发账号 failover 或健康惩罚

## 5. Responses WebSocket V2 relay 超时

- [ ] 5.1 补充多 turn relay 测试，覆盖首个 turn 超时后成功继续、终态不明时退出及新 turn 使用最新运行时配置
- [ ] 5.2 在 V2 passthrough relay 中维护 per-turn watchdog，并复用共享事件分类与 `response.cancel`/drain 语义
- [ ] 5.3 记录 V2 timeout 的失败 usage、Ops 错误和结构化日志，确保单个 turn 只产生一次错误

## 6. 管理端配置与全量验证

- [ ] 6.1 在管理端“网关服务”设置页增加两个非负秒级输入，支持 `0` 关闭，并补充加载、保存与校验测试
- [ ] 6.2 运行后端定向测试、race 测试与前端测试和类型检查，修复本 change 引入的失败
- [ ] 6.3 按 spec 复核 SSE、两套 WebSocket、运行时配置、可观测性和账号调度隔离场景，并记录验证结果
