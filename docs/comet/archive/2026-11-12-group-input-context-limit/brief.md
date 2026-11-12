# Outcome

在“系统设置 → 网关服务”增加“最大输入上下文限制”总开关，并允许管理员分别设置分组的最大输入 token 数。保存后，受限分组的对话模型请求在生成前接受准入检查；探测到超限时，以客户端入口协议对应的上下文超限错误返回，不进入生成、账号故障处理或计费。

状态：调研已完成，Q1—Q4 已由用户明确选择，待最终完整 Shape 确认；未修改普通实现。用户本轮要求充分调研和设计实现方案。本文候选比较仅为决策依据，发生歧义时以 Decisions 和完整目标 Spec 中的已选行为为准。

# Scope

## 已调查事实

1. 路由同时有 Messages、Responses、Chat Completions、Gemini 原生生成、Responses WebSocket，以及根路径、Codex、Antigravity 别名。七个 HTTP 逻辑 handler 和 WS 首帧/后续轮次均需覆盖；现有路由表未发现独立 legacy `/v1/completions`。
2. 现有 OpenAI 本地计数器只完整处理部分文本/工具字段，图片分支只计算 URL 或 data URL 头部；Gemini 本地估算只统计部分 text 字段。两者不能直接宣称是完整输入上下文计数。
3. `previous_response_id`、conversation、cachedContent、文件引用、加密 compaction 内容可能包含请求体外的输入。只计本轮新增文本会漏掉这些内容。缓存命中仍占上下文，不能按优惠计费 token 扣除。
4. OpenAI Responses compact 的 keepalive 在常规安全审查前启动；若要求 HTTP 400，检查必须前置到首次提交响应之前。WS 首帧与后续 response.create 是不同路径，只有 HTTP 中间件会漏检。
5. 有效路由会改写 API Key 上的 GroupID。需要在认证后、合成路由前冻结原绑定组身份。Composite 的 target platform/model 不等于另一个供给组。
6. Antigravity 已有上游 prompt-too-long 分组回退；本地策略拒绝不得复用 PromptTooLongError，以免限制被当成上游故障后绕过。

## 配置与管理接口

建议复用 settings 表单 key JSON 模式，不新增 Group 字段或迁移。独立 `GET/PUT /api/v1/admin/settings/input-context-limit`（相对于管理员路由的 `/settings/input-context-limit`）。

```json
{
  "enabled": false,
  "group_limits": {"123": 64000, "456": 128000}
}
```

- 总开关默认关闭；关闭时不读体、不 token 化、不调用计数服务。
- 每组阈值为整数 token，0 或未配置表示不受该组自定义限制；不另外引入全局默认数值。
- 管理页独立区块：总开关、分组搜索/平台筛选、分组名/平台/最大输入 tokens；清空表示不限。关闭开关保留已保存配置。
- 后端验证管理员权限、组 ID 正整数且存在、阈值为非负 int32 整数；拒绝负数、小数、溢出、无效组。PUT 整文档替换，保存失败不更新运行时快照，多管理员按最后成功保存覆盖。
- 开关与所有阈值以一个设置值原子保存；复用现有 SettingService 实例级原子快照、singleflight、限时回源及 last-known-good 模式，但避免旧 refresh 覆盖新 save 的竞态。Map 构造后只读，不向调用者暴露可变引用。
- 保存节点下一请求生效；其他节点以 60 秒 TTL 最终一致，DB 不可用时不能承诺最大传播时间。已有快照保留旧值，首次无配置按关闭并告警。删除组后的孤儿项运行时忽略、管理页提示清理，不复活已删除组。
- 每个请求/WS turn 冻结一次配置，后续 turn 重新读取。关闭开关不撤回已经作出的拒绝。

## 分组语义（Q2 待确认）

推荐 `limit = minPositive(原绑定组限制, 当前有效组限制)`；0/缺失不参与最小值。终点供给组之外的 ClaudeCodeOnly 中间跳板不另计入。原始身份快照不可随回退被覆盖。若上游错误触发候选组回退，进入新组的计费/生成前再次校验目标策略；模型/转换变化时计数必须重新取得，不能盲用旧模型结果。本地拒绝终止，不尝试宽松组。

## 计数与判定（Q1 待确认）

共用无副作用的计数结果与阈值判定，handler 只负责提取输入、提供路由/协议身份和发射错误。不得把所有协议强转 Responses 后声称完全保真。结果带计数来源及 coverage（可见字段完整/partial/unknown），请求内复用结果而不长期持有大请求体。

两种可执行方案，必须由用户确定：

- **A 可见输入估算限制（推荐兼容起步）**：本次请求提交的 system/developer/user/assistant 文本历史、instructions、tools/schema、工具调用参数和工具结果都参与本地估算；复用已安装 tokenizer。OpenAI 按已知编码器，Anthropic/Gemini 只能近似。对于多模态、文件、历史引用、密文，标记 partial，可见部分超限仍拒绝，其余交给上游。不增加每次请求的网络往返，但不能承诺完整上下文硬上限，管理页必须明确“按可见输入估算”。
- **B 官方计数优先，无法完整计数则拒绝生成**：按实际目标模型、账号及协议使用对应计数能力；本地估算仅作非权威辅助，不能以安全余量证明一定未超限。计数必须与将要生成的路由/输入一致，遵循账号资格/代理/超时策略；账号选择可以发生，但拒绝前不发送生成请求、不结算生成费用。不支持、超时、429、历史引用无法解析时返回独立“无法检查输入长度”的协议错误，不伪造 context_length_exceeded。每个有效请求通常增加一次计数往返；Anthropic 官方计数本身明确是估计值，因此该方案也不能保证与最终 usage 数学上逐 token 完全一致。

两种方案共同规则：只比较输入，不加 max_tokens/max_output_tokens；`count > limit` 拒绝，等于允许；不得按原始 JSON 字节数或 base64 字符数代替输入 tokens。已知输入字段损坏沿用协议参数错误，不得 fallback 为 0/1。计数服务的结果不得混入生成计费。无需新增通用插件框架、历史账本或跨请求计数缓存。

## 入口与生命周期

HTTP：鉴权/组资格/请求合法性确认后，尽量在调度、并发槽和计费预留前判定；方案 B 必须选定账号后进行官方计数，生成发送前判定并正确释放资源。Responses compact 若纳入限制，检查必须早于 keepalive。

WS：首帧与每个 response.create 都检查，两种 ingress native/passthrough 都覆盖；拒绝发送 Responses response.failed 终态，带 response.error.code=context_length_exceeded、正确序号及适用的 stream_id，再按当前 policy close 路径关闭连接。不默认承诺拒绝后原连接还能安全续用。

本地拒绝复用运维业务限制标记，独立原因 input_context_limit_exceeded；记录协议、组、模型、计数来源、coverage、阈值和配置版本，不记录正文/工具参数。不得记录成上游账号异常，不换账号、不冷却、不重试、不扣余额/订阅/生成流量费。

## 错误契约

**按客户端入站协议选择格式，而非上游账号平台**；Antigravity 复用 Messages 或 Gemini 格式。以下是官方协议结构兼容与常见超限文案；厂商未承诺所有模型逐字相同，不能声称逐字官方固定错误。

| 表面 | 建议契约 |
| --- | --- |
| OpenAI Chat Completions | HTTP 400；error.type=invalid_request_error，error.code=context_length_exceeded，param=messages |
| OpenAI Responses 非流式 | HTTP 400；error.type=invalid_request_error，error.code=context_length_exceeded，param=input |
| Anthropic Messages | HTTP 400；顶层 type=error，error.type=invalid_request_error，message=`prompt is too long: N tokens > M maximum`，使用本地真实 request_id |
| Gemini generateContent/streamGenerateContent | HTTP 400；error.code=400，status=INVALID_ARGUMENT，message=`The input token count (N) exceeds the maximum number of tokens allowed (M).` |
| Responses WS | response.failed；response.error.code=context_length_exceeded；事件结构官方，WS 中此 code 的组合需客户端兼容验证 |

OpenAI 文案推荐 `Your input exceeds the configured maximum input context length of M tokens (estimated input: N tokens). Please reduce your input and try again.`，避免把网关配置值冒充模型物理窗口；若为官方计数则去掉 estimated。Anthropic/Gemini 使用兼容前缀并在必要时注明网关配置限制。不开新自定义超限 code。

**Responses SSE（Q4 待确认）**：A 统一准入 HTTP 400（推荐遵循原需求与 API 参数错误语义），所有检测完成前禁止 keepalive 提交 200；B stream=true 返回 HTTP 200 + response.failed，供依赖该终态执行压缩的客户端识别。不能把 B 描述成原生 HTTP400。已建立流之后的兜底只能发协议终态，不能改回400。Codex 不同版本有错误识别差异，不能承诺任一格式必然触发自动压缩。

## 压缩自救与覆盖范围（Q3 待确认）

推荐 count_tokens、input_tokens、Gemini countTokens 始终不受此准入门限制；它们不生成对话。compact/压缩接口建议作为独立豁免，保留超长后的自救路径，并在管理页说明豁免输入可能超过阈值；不能把普通生成仅凭可伪造 header/参数变成压缩豁免。若用户要求所有生成类操作一律限额，则 compact 一并校验，接受超长后可能无法压缩。

首版覆盖 Messages、Chat、Responses、Gemini generateContent/streamGenerateContent、Responses WS，包含工具与多模态输入（计数完整程度按 Q1）。所有现有别名同样生效。SystemOne、独立搜索、embedding、图像/音视频生成、Realtime/Live、缓存创建等非对话专用入口不属于首版；UI 明示覆盖范围。用户要求扩展时先补各自协议和计数契约。

## 目录结构

以下是待实施路径，Shape 阶段只写正式方案文档。

### 新建

- `backend/internal/service/setting_input_context_limit.go`：独立 JSON 设置、校验、快照读取与原子保存。
- `backend/internal/service/gateway_input_context_limit.go`：共享计数覆盖结果、组阈值合并与无副作用判定；同目录聚焦测试覆盖计数/阈值和策略副作用。
- `frontend/src/components/settings/InputContextLimitSettings.vue`：网关页独立配置区块与保存反馈。
- `docs/comet/changes/group-input-context-limit/specs/group-input-context-limit/spec.md`：完整目标与验收编号。

### 修改

- `backend/internal/service/setting_service.go`：接入设置缓存字段；以实施时实际 SettingService 定义路径核对。
- `backend/internal/handler/admin/setting_handler_runtime.go`、`backend/internal/server/routes/admin.go`：管理员 GET/PUT。
- `backend/internal/server/middleware/api_key_auth.go`：冻结原绑定分组身份，供全入口一致读取。
- `backend/internal/handler/gateway_handler.go`、`backend/internal/handler/openai_gateway_handler.go`、`backend/internal/handler/gateway_handler_responses.go`、`backend/internal/handler/gateway_handler_chat_completions.go`、`backend/internal/handler/openai_chat_completions.go`、`backend/internal/handler/gemini_v1beta_handler.go`：七个 HTTP 接点、WS 两段与 Antigravity 回退重查。
- `backend/internal/service/openai_gateway_count_tokens.go`、`backend/internal/service/gemini_messages_compat_service.go`：只提取可复用计数能力，不改变既有公共计数端点的兼容语义。
- `frontend/src/api/admin/settings.ts`、`frontend/src/views/admin/SettingsView.vue`、`frontend/src/i18n/locales/` 下对应设置文案模块：设置 API、组件挂载和中英文文案。
- 对应 handler/service/frontend 测试：以现有测试方式新增最小必要回归，不搭建新测试框架。

### 删除

- 无。现有计数接口、上游超限透传与客户端压缩路径继续保留。

### 明确不建

- 不建 Ent group 字段、数据库迁移和 API Key 鉴权缓存字段链。
- 不建历史全文存储、token 历史账本、新 tokenizer 依赖、计数插件系统和强一致配置广播。
- 不因本次网关阈值新增模型物理窗口数据库或重写模型能力目录。

## 实施顺序与验证计划

1. 确认 Q1—Q4 与完整 Shape；本轮不进入 Build。
2. 设置存储/API/组件：原子保存、校验、缓存竞态和管理员访问检查。
3. 共享计数与分组判定：工具结果、Unicode、边界、partial/unknown；与选定 Q1 语义一致。
4. 七个 HTTP 接点与别名、WS 首帧/后续、回退目标重查；确保 Responses keepalive 顺序。
5. 按三协议构造超限契约，使用现有 handler 测试 mock 验证无生成调用/重试/费用/健康度副作用；WS/SSE 客户端版本兼容验证单列真实结果。
6. 前端类型检查、相关 Go 测试、设置页浏览器流程；多节点热更新、故障回源与 refresh/save 并发测试。仅在需要真实仓储时启动临时数据库，完成后停止任务创建实例。

单一 Change：设置和各入口共同定义同一准入行为，拆成多个可独立批准目标会增加协调成本，暂不拆 Supervisor。

# Non-goals

自动裁剪/丢弃历史、修改 max_tokens、替用户压缩、改变上游官方模型窗口、保证客户端一定自动压缩、将字节请求上限或429当作 token 超限、覆盖未列出的专用非对话入口、发布或部署。

# Acceptance examples

- A1：默认关闭；配置/关闭/开启可保存，关闭保留各组阈值，无限制快速路径不计数。
- A2：组A=64000、组B=128000，同一计数为80000的请求A拒绝B放行；等于阈值放行，超过1拒绝。
- A3：非法组/负数/小数/溢出拒绝且不污染旧配置；保存失败不发布快照；refresh不会覆盖较新的save。
- A4：原绑定组、有效组、合成平台映射和回退按Q2工作；本地拒绝不触发宽松组回退。
- A5：文本历史、system、tools/schema、调用参数与结果计入；输出预算不计，缓存不打折；Q1选项对多模态/历史/密文/计数失败有明确且测试过的结果。
- A6：HTTP七个接点和现有别名均覆盖；count端点不拦，compact行为服从Q3；专用非对话入口范围可核对。
- A7：Messages/OpenAI/Gemini协议状态码及关键字段正确；Responses流式遵从Q4，keepalive不能抢先破坏所选契约。
- A8：Responses WS首帧、第二轮、两种ingress都检查，返回可解析的终态且无上游生成；拒绝后按约定关闭连接。
- A9：策略拒绝不产生账号惩罚、重试、生成计费和正文日志；配置按单请求/turn快照读取。
- A10：保存节点下一请求读取新配置，健康多节点60秒内刷新；DB故障保留last-known，未知首次值行为与告警可验证。

# Research evidence

检索日期：2026-11-12。官方结构证据高于社区具体文案；故障 issue 只说明特定版本风险，不视为全部客户端行为。

- [Anthropic errors](https://platform.claude.com/docs/en/api/errors.md)：400 invalid_request_error；413是请求字节大小，429是速率，不适合作为输入token超限。
- [Anthropic context windows](https://platform.claude.com/docs/en/build-with-claude/context-windows)：输入单独超窗与输出触及窗口的行为应区分。
- [Anthropic token counting](https://platform.claude.com/docs/en/build-with-claude/token-counting.md)：官方明确estimate，工具/URL/file来源有限制，不能承诺完全精确。
- [OpenAI input token API](https://developers.openai.com/api/reference/resources/responses/subresources/input_tokens/methods/count.md)：模型相关文本、图片、文件、tools、conversation等输入；previous_response_id可用性不得凭推断保证。
- [OpenAI Responses WS](https://developers.openai.com/api/reference/resources/responses/websocket-events)：response.failed结构；该帧搭配context_length_exceeded需客户端实测。
- [OpenAI error codes](https://developers.openai.com/api/docs/guides/error-codes.md)：通用错误页不保证每种超限精确文案。
- [Google AIP-193](https://google.aip.dev/193)：Google错误标准结构；Gemini具体超限句式为实证常见形式。
- [Google官方计数说明](https://firebase.google.com/docs/ai-logic/count-tokens)：模型计数能力参考，不把特定平台配额硬编码为所有Google后端通用值。
- 仓库证据：`backend/internal/service/openai_gateway_count_tokens.go:632,770,795`；`backend/internal/service/gemini_messages_compat_service.go:2988`；`backend/internal/handler/openai_gateway_handler.go:570,3055,3653`；`backend/internal/server/middleware/api_key_auth.go:401`；`backend/internal/service/setting_panel_rate_limit.go:107,133`。

# Decisions

用户于本轮结构化澄清中明确选择：

- Q1 = A：可见输入本地估算。未知部分交上游，不承诺完整上下文硬上限；不增加官方计数预检。
- Q2 = A：原绑定组与实际有效组取非零最小值，不计中间路由跳板。
- Q3 = A：真实 compact 压缩接口豁免，count 端点豁免；普通生成不得冒充压缩。
- Q4 = B：Responses stream=true 超限返回 HTTP200 + response.failed，response.error.code=context_length_exceeded；非流式400，WS同类终态后关闭。

# Open questions

无未决行为问题。等待用户对目标、范围、全部验收和非目标的最终 Shape 确认；本轮不执行 Build。
