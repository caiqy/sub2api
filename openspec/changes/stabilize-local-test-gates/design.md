## Context

`RequestBodyHandle.Open` 在持锁状态下打开 spool 文件，但返回 reader 后立即释放锁；并发 `Cleanup` 随后直接删除路径，在 Windows 上会因活动文件句柄触发 sharing violation。OpenAI streaming keepalive ticker 又早于 `lastDownstreamWriteAt` 初始化，导致首个 tick 略早于完整空闲间隔而被跳过，下一次 tick 与 first-output timeout 竞争。Server-Timing rows 测试还以 `30ms` app gap 对比多次 `2ms` driver sleep，全量压力会放大短 sleep 并打破脆弱比例。三处问题均只在全量测试的并发或时序压力下间歇暴露。

## Goals / Non-Goals

**Goals:**

- cleanup 调用立即使 handle 不可再次打开，并在最后一个活动 spool reader 关闭后可靠删除文件。
- 首个 keepalive 在配置间隔到达时稳定写出，不与 first-output timeout 竞争。
- 所有本地门禁及 OpenSpec strict 校验稳定通过。

**Non-Goals:**

- 不改变 request body 的落盘阈值、预览、hash 或过期文件清扫策略。
- 不改变 SSE 事件、first-output timeout 或 failover 协议语义。
- 不增加重试包装、平台特判或新依赖。

## Decisions

- `RequestBodyHandle` 记录活动 spool reader 数量。cleanup 遇到活动 reader 时先标记 handle 已清理并延迟物理删除；最后一个 reader 的幂等 `Close` 完成删除。相比容忍 Windows 错误或整段读取到内存，该方案保留跨平台语义和大请求内存上限。
- 在创建 keepalive ticker 前记录 `lastDownstreamWriteAt`，让 ticker 的首个触发点不早于完整空闲间隔。相比放宽测试时间或增加 sleep，该方案直接修正生产计时基准。
- 将 Server-Timing 测试的 app gap 设为 `100 * fakeDriverDelay`。保持 `app > db` 语义断言和生产 instrumentation 不变，仅为调度抖动提供明确裕量。
- 扩写 `local-test-gates` 的 Purpose 以满足 strict 文档校验，并在 delta spec 中明确活动 reader 的 cleanup 生命周期；其他 requirement 与 scenario 保持不变。

## Risks / Trade-offs

- [reader 未关闭会延迟 spool 删除] → 保留启动时 stale sweep 作为最终兜底，行为不劣于当前删除失败路径。
- [并发 Close 重复触发删除] → reader wrapper 使用一次性关闭语义，引用计数和删除均受 handle mutex 保护。
- [keepalive 提前提交稳定 SSE headers] → 仍沿用既有 guard，只提交稳定 keepalive，不泄露 attempt-local headers 或 staged events。
- [Server-Timing 测试增加约 170ms 时长] → 仅影响单个 repository 测试，换取全量压力下稳定判定。
