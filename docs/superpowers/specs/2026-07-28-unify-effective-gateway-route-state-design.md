---
comet_change: unify-effective-gateway-route-state
role: technical-design
canonical_spec: openspec
archived-with: 2026-07-29-unify-effective-gateway-route-state
status: final
---

# 统一网关有效路由状态技术设计

## 1. 背景

`staged-merge-upstream-v0-1-165` 在合入上游 `v0.1.164` 后完成了多轮兼容修复，但最终审查仍保留 `3 Important + 1 Minor`：

1. 最终 group 未同时成为 middleware subscription、protocol dispatch、Gin API key 与 Ops 的权威状态；
2. prompt-too-long 二级 fallback 仍校验中间组，billing 可能因 `subscription=nil` 退化为余额模式；
3. HTTP bridge 的后续 WebSocket 帧只完成 route rewrite，缺少 account mapping 与 provider affinity；
4. 无 channel mapping 时，public alias 可能覆盖 concrete route model。

这些问题的共同根因是 group、API key、subscription、endpoint、模型和请求体由不同层分别推进。局部补丁可以关闭单条路径，但不能防止下一条入口再次产生状态漂移。

本 change 在当前 staged merge 分支上完成修复。它通过独立 OpenSpec change 提供新的实施与审查预算，完成后归档，并把审查证据写回原 change 后自动恢复其 Task 21。

## 2. 目标与非目标

### 2.1 目标

- 建立一个 request-scoped 的 effective route 结果，作为鉴权、subscription、协议分发、调度、计费、请求体改写和 Ops 的共同输入；
- 让初始解析、prompt-too-long fallback、普通 WebSocket 与 HTTP bridge 使用同一解析语义；
- 明确 `client -> route -> channel -> account` 的模型阶段，缺少某层映射时保持上一阶段结果；
- 用聚焦测试和本地全门禁关闭原 `3 Important + 1 Minor`。

### 2.2 非目标

- 不修改公开 API、数据库 schema、migration、配置格式或前端；
- 不重构无关 Anthropic、Gemini 或非 composite 路径；
- 不重复原 staged merge 已关闭的 pricing、body snapshot、Ollama、Grok multipart、copy-source 或 ledger 修复；
- 不在本 change 中运行远程 Docker integration，不 push、tag、release 或 deploy。

## 3. 方案选择

采用统一请求态方案。逐路径补丁继续保留多个状态来源；把全部解析塞入 API Key 鉴权 middleware 则会扩大到所有协议。由于实际 endpoint protocol switch 位于 `server/routes/gateway.go` 且早于 handler，设计把现有 handler `effectiveGatewayRoute` 提升为 service 层 request snapshot，并由 route-local composite middleware、两个 gateway handler 和 runtime fallback 共用同一 resolver。

不新增 interface、factory 或持久化模型。内部结果表达为等价于下列字段的 request-owned 值对象：

```go
type EffectiveGatewayRoute struct {
	APIKey        *APIKey
	Group         *Group
	GroupID       *int64
	Subscription  *UserSubscription
	BillingSource EffectiveGatewayBillingSource // simple-skip / balance / subscription
	Endpoint      string
	ClientModel   string
	RoutingModel  string
	UpstreamModel string
	Platform      string
	Decision      *CompositeRouteDecision
	Channel       ChannelMappingResult
}
```

`APIKey` 必须是 request-owned clone。`RoutingModel` 是 composite route 与可选 channel mapping 后的结果；`UpstreamModel` 默认等于 `RoutingModel`，绑定账号后才由 account mapping 覆盖。`BillingSource=subscription` 时 `Subscription` 必须非空；余额模式和 SimpleMode 跳过不再依赖空指针推断。

## 4. 架构与数据流

```text
APIKeyAuth
  -> body capture
  -> client detection
  -> route-local effective route resolve + validate + apply
  -> endpoint protocol dispatch
  -> handler / scheduler / billing / Ops
                |
                +-> runtime fallback: resolve + validate + atomic apply
                +-> account selected: provider check + account mapping
```

### 4.1 初始请求

扩展现有 `compositeTargetPlatformMiddleware`：在 API Key 鉴权后读取既有受限 body、完成客户端识别，并在 endpoint protocol switch 前调用共享 resolver。它先解析最终 group、精确 endpoint、composite route、API key clone、billing source 和最终组 subscription snapshot；handler 随后在相同 snapshot 上补充 channel/account mapping。不新增全局 middleware，也不改变无关协议的链路。

只有所有校验通过后，`Apply` 才一次更新 middleware 维护的 Gin API key/subscription 键、request context group/route decision、Ops 上下文和现有 body handle。protocol dispatch、scheduler 与 billing 不再重新推导 group。

### 4.2 Runtime fallback

prompt-too-long 等运行时 fallback 调用同一 resolver，并以当前 effective route 作为输入。候选结果必须先通过最终组授权、subscription、余额和模型校验；成功后才原子替换上下文与请求体。

解析失败时保留原 route/body 状态，但请求本身按现有错误契约终止，不继续使用中间组调度或计费。

### 4.3 账号绑定与 WebSocket

账号选定后，现有账号映射 helper 校验账号 provider 与 effective provider 一致，并形成 account mapping 后的结果。普通 WebSocket 已正确处理后续帧；本 change 只让 HTTP bridge 的后续 `response.create` 补齐相同顺序：

```text
effective route -> optional channel mapping -> provider affinity -> account mapping
```

provider 不一致时，在改写 body 或写入上游连接前拒绝当前请求/帧。provider 一致时复用会话账号，并把 account mapping 后的模型作为 `UpstreamModel`。

## 5. 模型与审计语义

模型阶段固定为：

```text
ClientModel
  -> composite route
  -> channel mapping（存在时）
  = RoutingModel
  -> account mapping（存在时）
  = UpstreamModel
```

无 composite route 或 channel/account mapping 时，该阶段是 identity，不得回填 `ClientModel` 覆盖已有 concrete model。失败 usage 与正常 usage 都从 effective route 读取三阶段值，不从已改写 body 反推。

## 6. 错误与兼容语义

- resolver 只计算和校验，不修改 Gin context、body handle 或 WebSocket session；`Apply` 只接收完整有效结果；
- 最终 group 一旦选定，不再静默回退到原 group；
- 最终组未授权、subscription 不可用、余额不足、模型不可用或 provider 不一致时，沿用现有错误类型和 HTTP/WebSocket 错误映射；
- 校验失败不得调度账号、写入上游或产生计费；
- billing 必须收到 effective route 的显式 `billingSource`；订阅模式缺少 subscription 是解析错误，禁止用含义不明的 `nil` 触发余额模式；
- 不原地修改共享 Ent API key/group/account 实体。

## 7. 测试设计

测试优先扩展现有 handler、gateway service 和 WebSocket 测试文件，仅在没有合适归属时新增一个聚焦测试文件。

1. **有效 group 一致性**：构造 composite/fallback 到不同最终组的请求，断言授权、subscription、protocol dispatch、Gin API key、scheduler 输入与 Ops 均使用最终组；无权访问最终组时在 dispatch 前失败。
2. **二级 fallback 计费**：构造 prompt-too-long 后切换到订阅组，断言重新取得最终组 subscription 且 billing 不走余额；最终组校验失败时不调度、不计费。
3. **WebSocket 后续帧**：新增 HTTP bridge 在 route rewrite 后继续执行 account mapping 的回归，并保留普通转发映射与后续帧跨 provider 拒绝测试。
4. **无 channel mapping**：public alias 映射到 concrete route model 且 channel 无配置时，断言 `RoutingModel` 保持 concrete model，`ClientModel` 与 `UpstreamModel` 各自正确。
5. **回归保护**：覆盖非 composite identity 路径，证明现有协议和模型不发生额外改写。

实现完成后执行：

```text
聚焦 Go 测试
make test
make build
git diff --check
openspec validate unify-effective-gateway-route-state --strict
```

远程 Docker integration 留给恢复后的原 Task 21，避免对同一 commit 范围重复执行。

## 8. Review、归档与恢复

fresh reviewer 必须逐项核对原 `3 Important + 1 Minor`，并给出对应代码、测试和命令证据。完成条件为：

- 原四项 finding 全部关闭；
- 没有新的 Critical 或 Important finding；
- 聚焦测试和本地全门禁通过；
- OpenSpec artifacts、实现与证据一致。

通过后归档 `unify-effective-gateway-route-state`。随后把该 review 作为跨 change 证据写入 `staged-merge-upstream-v0-1-165` 的 progress 与 canonical ledger，勾选原 `6.1`，但不重开其已耗尽的 review 预算。最后恢复原 change 选择并进入 Task 21 完整门禁；任何 publish 或部署动作仍需单独授权。

## 9. Implementation Divergence

本 change 的聚焦测试、生成稳定性、构建、前端完整门禁、strict OpenSpec validation 和 fresh review 均通过，但顶层 `make test` 未取得 exit 0。唯一剩余失败 `TestPassthroughLifecycle_LeaseLossSendsRetryClose` 已在未含本地改动的上游 `v0.1.165` tag 上复现，且相关实现与测试不在本 change diff 中。用户明确接受该上游基线例外并要求归档；验证报告不得把 `make test` 记为通过。
