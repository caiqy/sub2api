## Context

当前调度管线已拆分 Sticky、负载感知和模型路由路径。历史平台级开关逻辑未完整迁入，导致关闭开关后部分路径仍读取或写入会话缓存；无 `ConcurrencyService` 的模型路由也可能绕过已筛选的路由账号集合。另一个独立问题是 OpenAI wrapper 与通用 `GatewayService.newSelectionResult` 同时拥有 hydration 失败时的 release 责任。

## Goals / Non-Goals

**Goals:**
- 平台 Sticky 开关统一决定 cache read、write、prefetch 和相关调度分支是否可用。
- 未配置并发服务时维持模型路由账号集合的选择语义。
- hydration 失败时每个已取得的槽位仅释放一次。

**Non-Goals:**
- 不调整 Sticky 默认值、配置结构、公开接口或数据库 schema。
- 不重构调度算法或新增缓存策略。

## Decisions

### 1. 在 GatewayService 集中平台开关判断

添加最小平台判断 helper：nil service/config 与未知平台保持默认启用；Gemini、OpenAI、Anthropic/Antigravity 分别读取各自配置。所有已拆分的 Sticky 缓存读取、绑定、prefetch 和负载感知入口复用该判断；关闭时记录既有结构化 bypass 日志。

### 2. 保持模型路由候选集合

当 `ConcurrencyService` 不可用时，从当前已筛选的 routing account 集合选择，而不是退回全量账号列表。该路径不引入新的调度策略。

### 3. 由通用选择结果唯一拥有 release

`GatewayService.newSelectionResult` 已在 hydration 失败时调用 release。OpenAI `newAcquiredSelectionResult` 只委托该方法，不再包装第二次 release。

## Risks / Trade-offs

平台判断遗漏入口会造成关闭 Sticky 后仍有缓存操作，因此由现有跨平台、bypass 日志和模型路由测试覆盖。删除重复 release 只改变错误路径，保留成功路径的既有计数语义。

## Migration Plan

无需迁移。代码发布后新请求立即采用恢复后的调度与释放语义。

## Open Questions

无。
