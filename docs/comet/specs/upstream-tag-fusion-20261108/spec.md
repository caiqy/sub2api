# Upstream Tag Fusion 20261108

## Purpose

将 upstream 正式发布 v0.2.20 融合进本地 fork，保持双方功能和数据兼容性，留下可复查的合并历史与验收证据。

## R1：冻结版本与历史

从 main 的 a654a78d79ede218b106fb255f9703837a17a829 / VERSION 0.2.19.7 出发，在 comet/merge-upstream-v0-2-20 建立 no-ff merge。第一父为该基线，第二父只能是 v0.2.20 的 dab829a4e9b7d1cc307189aea8b0319eb29e29d0。不包含随后 upstream/main 的 6b4b37db43443b5e88e424fe3c69602cc3810dfb 或以后新 tag。发布、推送、部署及合回 main 不属于实施阶段授权，验收后等待用户接受与交付选择。

## R2：管理与安全

完整吸收管理 MCP 的 OAuth 授权、客户端注册、用户连接管理、配置开关及后端路由、CORS、权限和限流。吸收管理员密钥新权限和确认式全权兼容模式、审计、注册邮箱精确黑名单及批量禁用流程。保留 fork 的用户鉴权投影、缓存失效、账号限制、作用域、菜单可见性、验证码及管理端能力；不使新增接口绕过已有安全约束。

## R3：调度

普通候选、session sticky、previous_response_id sticky、fallback、WaitPlan、DB 复核与抢槽后复核均维持本地资格约束；privacy、模型能力、生图、ModelTrace 策略、Codex 客户端资格和运行时配置热更新不被改写。融合上游 1K/2K/4K 图片尺寸路由与会话绑定行为，不令请求级资格仅作用于普通候选而漏掉粘性路径。

### R3a：撤除 fork layered/probe，保留共同调度边界

用户在本轮 Build 补充要求：确认截图功能为本人 fork 自增后，撤除并恢复上游。已核实 cf7c74dfbadd0fdb880c3b1074037dc004235d37（caiqy，2026-04-02）引入 layered 调度与健康探测，属于 fork 历史且不属于冻结的上游目标；上游仍有 default 加权调度及独立高级调度设置。

移除 layered 调度实现、后台健康探测和其生命周期/管理桥接，取消模式工厂选择；移除截图的模式及九项惩罚/探测参数的配置声明、默认/校验、settings 持久化读写、DTO、前端类型、表单/提交和翻译。账号中仅服务此探测器的开关/模型入口及其调用同步撤除。更新 Wire、相关文档与测试；只删除已撤除行为的测试，其他回归保留。

default/weighted 采用上游工厂和调度路径，保留普通/previous-response/session sticky/fallback/WaitPlan/DB 复核、Codex 客户端资格、privacy、模型/图片能力与尺寸路由、guardian/Grok、利润/计费及平台 sticky 开关。不得整文件恢复上游而覆盖这些独立定制；保留上游高级调度开关、TopK/打分权重、错误/TTFT 指标和普通限流/临时不可调度/手动恢复能力，不因名字含 scheduler/probe/TTFT 而删除共享能力。

旧配置键和账号 Extra 键停止生效但不为此删除生产数据或改写历史迁移；遗留临时不可调度记录仍遵循自身过期/既有手动恢复，不批量清理其他来源的封禁。其他 runtime 热更新字段仍可往返并生效。检查覆盖旧 layered 配置不会启用旧实现、无后台探测线程、保留资格与粘性等边界及页面撤除。原 A3/A6/A7 的相关检查按新候选重做，已收束的无关融合改动保留，旧测试结果不视为新需求的正式验收。

## R4：网关与资源

吸收 GPT Image 2.6 原生图片链路、非流式 Responses 完整内容、buffered reasoning、Gemini 工具媒体、Antigravity schema/stream/身份和配额修复、Cloudflare Messages endpoint、WS HTTP bridge 指纹与每 turn 首 token 等上游修复。保留 fork 请求体 spooling、可重放 handle、最终模型映射后协议选择、提前释放、历史参数捕获、透传字段及失效密文恢复。资源预算与读写超时须与上述时序兼容，专用超时与 client-close 错误处理先于通用传输错误归一化；重试仍可重放且不重复结算预算。

## R5：计费与用量

融合上游模型计费、RPM 缓存与后台任务边界，保留 fork 按客户端模型选择专属倍率、利润门、在途估算、WS 每 turn 快照、批量图片、订阅配额预支、用量详情和导出；用实际入口的回归证明，不以单一纯函数断言代替链路核对。

## R6：数据与构建发布

新增 MCP OAuth 迁移必须兼容 fork 已存在的 249_group_exclusive_rate_models.sql 与 250_user_cyber_overflow_exempt.sql，不能改写已执行迁移或破坏校验和。保留模型和用户字段的投影、生成代码与 Wire 依赖。依赖安全升级同步 lockfile。保留 VERSION 0.2.19.7，本次不生成四段派生版本；保持 fork 单 linux/amd64 构建矩阵及 tag 后手动触发一次 release.yml 的发布规则。

## R7：前端

MCP 授权与连接页、设置黑名单、管理员密钥权限、账号图片尺寸与配额、密钥导出和用量修复完整接入。保留 fork 账号 Codex 限制、ModelTrace、用量 Excel 导出、生图工作台及其他配置字段；避免重复控件，编辑保存字段能完整往返。多语言与 vue-i18n 升级保持兼容，执行前端测试、typecheck、build，并对关键变化页面做本地预览和浏览器验证。

## R8：验收与交付

专项审查文本冲突和无冲突交集，逐项追踪本地关键能力；遇不可共存的业务语义先取得用户决定。执行 go test ./...、既有 unit/integration 标签检查与构建、前端测试与类型构建、现有发布脚本测试。临时数据库结束后停止且确认端口关闭，不停止用户数据库；不可用、失败、超时和未运行必须如实报告。新的独立只读 Verifier 对 brief A1–A8 逐项验收，失败回 Build；全部通过后待用户接受结果及选择交付方式。
