# Brainstorm Summary

- Change: restore-platform-sticky-routing
- Date: 2026-07-14

## 确认的技术方案

- change 已从 Hotfix 升级为 full workflow，以统一跨 HTTP、WebSocket、Ingress 与 compat 服务的 Sticky 状态边界。
- 关闭 OpenAI Sticky 时，OpenAI WS 的 response-account、response-connection、turn state 和 session-connection 全部 bypass：不读取、不写入。
- `GeminiMessagesCompatService` 按最终解析的平台（Gemini、Anthropic、Antigravity）遵守对应 Sticky 开关，关闭时不读取、写入或刷新会话缓存。
- 采用各 service 的最小平台守卫：Gateway 保持现有 helper；OpenAI WS 通过受控 sticky state-store accessor；compat 在解析平台后一次确定 Sticky 是否启用。不新增共享服务、依赖或公共接口。
- 架构已确认：HTTP、WS V2、WS ingress、调度、compat 与 Antigravity 清理分别在入口处受对应平台开关控制；默认开启语义和现有状态存储实现保持不变。

## 关键取舍与风险

- 将全部 WS 状态纳入 Sticky 边界会禁用跨请求连接复用，但符合用户对关闭 Sticky 的完整隔离要求。
- 现有未提交的调度、HTTP response-id 与 Antigravity 清理修复保留为设计的实施基础。
- 不在底层缓存全局拦截，避免跨平台配置与非 Sticky 状态相互影响。

## 测试策略

- 为 HTTP、WS V2 与 Ingress 分别验证关闭时状态存储零读写，开启时保留既有状态行为。
- 验证 compat 在 Gemini、Anthropic、Antigravity 三个平台关闭时的会话缓存零读写。
- 继续保留模型路由优先级、默认开启和 hydration 单次 release 回归测试。

## Spec Patch

- 新增 `platform-sticky-boundaries` delta spec，定义跨 HTTP、WebSocket、Ingress 与 compat 服务的关闭时零读写和默认开启行为。
