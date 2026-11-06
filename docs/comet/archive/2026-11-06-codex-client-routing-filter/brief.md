# Outcome

为 OpenAI OAuth 账号的“仅允许 Codex 官方客户端”增加“用于路由筛选”子开关。开启后，将现有客户端准入判定提前用于账号选择：不符合条件的请求跳过该账号，继续寻找同一请求原本有权使用的其他账号。

# Scope

账号级新增布尔配置 `extra.codex_cli_only_route_only`（界面名称“用于路由筛选”），默认关闭，仅在父开关 `codex_cli_only` 开启且账号为 OpenAI OAuth 时生效。这里的 route_only 表示不合格账号在路由中退出候选；保留转发前原有安全校验。

复用现有 detector 和策略，完整保留 ForceCodexCLI 旁路、黑白名单、全局/账号 App Server 放行、版本上下界及引擎指纹规则。使用客户端入站身份和当前请求/WS 首条生成消息的指纹信息，不能用网关改写后的官方身份判定。

覆盖网关实际业务请求中所有可能选择 OpenAI OAuth 账号的调度入口（Responses、Chat Completions、Messages 协议兼容、WebSocket，以及可选择此类账号的其他业务端点）；覆盖默认/分层/旧调度、普通候选、会话粘性、previous_response_id、凭据影子账号、抢槽后复核和故障重选。强绑定上下文不能安全迁移时沿用现有失败规则，禁止为了绕过筛选而向不合格账号转发。

没有合格账号时沿用当前端点“无可用账号”或已有强绑定失败响应；不额外制造该账号的 403，不越过分组、授权、模型或协议边界寻找账号。关闭新开关的其他账号继续保持原来的选后拦截行为。

新增、编辑和批量编辑账号入口支持配置，沿用现有组件和中英文本。批量编辑沿用现有子开关约束：同次明确批量开启父开关后才写子开关；未勾选更新的字段不得覆盖。账号复制、重新授权等既有 extra 保留流程必须保持新字段。

## 目录结构

### 修改

- backend/internal/service/account.go — 新子开关的适用范围与默认值。
- backend/internal/service/openai_client_restriction_detector.go、openai_gateway_service.go — 复用准入判定、构造最小请求级筛选信息。
- backend/internal/service/openai_account_scheduler.go、openai_account_scheduler_layered.go、openai_gateway_scheduling.go 及同模块选号辅助文件 — 所有调度层和最终复核接入统一条件，保留现有资源释放与粘性语义。
- backend/internal/handler/ 下 OpenAI 及协议兼容业务入口 — 在选号前提供入站请求身份/指纹，保持 failover 上下文一致。
- frontend/src/components/account/CreateAccountModal.vue、EditAccountModal.vue、BulkEditAccountModal.vue — 配置展示、读取、保存、重置与批量更新。
- frontend/src/locales/ — 新子开关中英文本。
- backend/internal/service/、backend/internal/handler/、frontend/src/components/account/__tests__/ 中相关测试 — 验证判定一致性、路由排除、兼容和保存链路。

### 明确不建

- 数据库列与迁移 — 沿用 accounts.extra。
- 第二套客户端识别器、全局新策略和独立路由框架 — 复用既有 detector 与调度机制。

# Non-goals

不调整官方客户端识别标准、既有放行规则和全局配置；不提供可绕过原检测的新授权；不改其他平台/API Key 账号的客户端限制；不重设计账号界面；不把管理测试、模型目录读取等非业务选号操作扩展为新的客户端鉴权产品；不做发布、部署或推送。

# Acceptance examples

- A1：新字段缺失、false 或非布尔真值时保持原行为；父开关关闭、非 OpenAI OAuth 时子开关无效；只有双开才参与路由筛选。
- A2：账号 A 双开、账号 B 允许该请求，非合格客户端请求到达同一授权池时，跳过 A 并选择 B，A 不发上游、不占用最终并发槽、不成为新的粘性目标；子开关关闭后恢复原选后拦截。
- A3：官方、黑白名单、App Server 全局/账号放行、版本上下界、不可解析版本、指纹缺失/命中和 ForceCodexCLI 等代表场景，筛选结果与现有 detector 一致；使用入站头和 body 指纹，不被出站身份改写影响。
- A4：默认、分层和旧调度的普通候选、会话粘性、previous_response_id、影子账号、等待后/数据库刷新复核及故障重选均不能选中不合格的双开账号；可迁移时选合格账号，不能安全迁移时沿用既有失败规则；不会因单次请求不匹配把账号标记为健康故障或全局暂停。
- A5：Responses（含 compact）、Chat Completions、Messages 兼容、WebSocket 首次选号/重选及其他可选 OpenAI OAuth 的业务入口都能正确传递客户端判定信息；WS 使用入站握手头与首条生成消息；筛选不得消费掉后续转发所需请求体或破坏请求体句柄的释放。
- A6：全部候选被过滤时使用当前端点无可用账号/强绑定失败响应，无不合格上游请求；不越组、不越权、不放松模型/能力约束；未启用新子开关的账号仍保持原有拦截语义。
- A7：创建、编辑和批量编辑提供从属父开关的新子开关及中英文说明，默认关闭；保存、回显、重置、父开关关闭和批量未选更新行为正确；账号复制及重新授权不丢失该配置。
- A8：保留可运行的针对性回归检查，覆盖账号配置、混合账号池、全部排除、粘性与故障重选、入站指纹以及前端 payload；完成相关后端检查、前端类型检查和定向测试，独立 Verifier 核对 A1–A8 与完整规格；未运行检查如实记录。

# Decisions

- 新开关为账号级子开关，缺省关闭，避免已有账号升级后静默改变行为。
- 路由不匹配只是本次请求的候选资格问题，不作为上游失败，不消耗故障切换预算。
- 转发前现有校验继续作为防御性检查，路由判断不能削弱账号限制。
- 当前工作区干净且无其他 active change，使用 Runtime 默认 current 工作区/main；本需求单个 change 管理。

# Verification expectations

宿主具备可跟踪的独立子代理能力；Build 完成后由新的独立只读 Verifier 领取 SDK Action 验收。使用现有 Go/Vue 测试工具，SDK 经 vfox 管理。不启动生产服务或改动线上账号。
