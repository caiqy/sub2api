# Upstream Tag Fusion 20261029

## Purpose

将 upstream 正式发布 v0.2.18、v0.2.19 按冻结边界逐版本完整集成到 fork，保留现有业务增强并以当前候选证据验收。

## R1：版本与合并拓扑

基线为 dd3829d66cfe69719b20103c488531ad5c4b70a7 / VERSION 0.2.17.5。独立分支 comet/merge-upstream-tags-20261029 以 main 为目标；依序对 v0.2.18 的 ac1b8f1271be0d674fa6f074403c1410f30a3a81、v0.2.19 的 dece26153621959ccd7635814f4b2d08304e681a 建立各自 no-ff merge。每版记录冲突与检查，合并前一版后才开始下一版；tag 后主线与后续版本不进入范围。

### Scenario: 按冻结版本逐次融合

Acceptance: A1

WHEN 检查候选提交图与远端 tag 对象，THEN 两个 merge 按序存在，其第二父分别为上述冻结 commit，二者均为候选祖先且来源可追溯。

## R2：调度与账号快照

完整吸收调度快照内存缓存、模型映射签名、模型限流写时复制、按需单 Antigravity 探测和 Vertex 配额。保留 fork 的模型身份校验/临时冷却、调度策略、会话/previous-response sticky、freshDB、privacy、图片能力、fallback/WaitPlan、运行时重载以及分组 retire/reopen epoch 与生命周期锁；优化不得让共享对象被单请求修改或绕过最新权威状态。

### Scenario: 优化后仍满足选号限制

Acceptance: A2

WHEN 在普通、sticky、fallback、热更新或分组删除重建路径选号，THEN 所有有效限制继续生效，快照读写不污染其他请求，淘汰后的旧写入不能恢复已退休分组。

## R3：协议、流式、图片与请求体

吸收兼容流上游异常收尾与断连后的用量结算、Gemini 已收集用量结算、OpenAI overload 元数据后的安全重试、Gemini 生图开关/显式缓存同号重试/模型目录、DeepSeek 角色和图片归一化、图片转存与上传大小校验。保留 fork 最终模型映射后协议构造，handle/cache/spool 重放与原文释放时序，超时类型、状态码、关闭码与计时，禁止通用错误转换覆盖专用错误。

### Scenario: 实际入口满足重试和结算契约

Acceptance: A3

WHEN 客户端断连、空输出、上游异常、缓存重试、跨模型协议映射或图片超限，THEN 对应上游修复生效，保留本地错误契约和请求体生命周期，计费只按应结算结果执行且不重复。

## R4：管理员密钥与权限

按操作与资源范围授权的管理员 API 密钥覆盖鉴权中间件、服务、查询范围、事务、幂等、返回裁剪和限流。既有管理员密钥兼容按上游定义保留；未知或未授权操作、跨资源访问与敏感字段泄露受到约束。fork 管理员操作及普通用户明细权限继续成立。订阅提前额度周期的手动入口/opt-in 自动行为、窗口选择、行锁、幂等回执及提交后版本化缓存失效继续生效。

### Scenario: 资源授权与本地订阅增强共存

Acceptance: A4

WHEN scoped key、旧管理员或普通用户调用相关接口，THEN 只能获得授权范围的数据与操作，返回遵守字段裁剪，fork 订阅操作在并发和重试时不回滚其他更新、不重复生效。

## R5：模型目录与计费

融合 Grok 目录同步/手工编辑模式隔离/诊断/并发 CAS/过期结果保护/响应模型审计，分组 image_count_billing 对实际返回图片与渠道每请求计价的正确判定，DeepSeek prompt_cache_hit_tokens 归类，GPT-6.1 Terra 与 Gemini 目录、渠道 web_search_emulation 维度更新。保留 fork 响应模型计费的可采纳边界、平台额度与固定费用预占/mandatory billing、ModelTrace 额外认可模型任务快照、官方定价与缓存存储费。

### Scenario: 新模型与计费配置按边界生效

Acceptance: A5

WHEN 同步模型、切换目录模式、出现响应模型冲突、多图结果、按请求渠道定价或缓存命中，THEN 新功能完整接线，按实际合法结果计费和审计，固定费用预占覆盖用户平台限制，ModelTrace 以创建任务时快照判定。

## R6：前端与本地能力

吸收账号筛选和 Vertex 配额展示、Grok 诊断/白名单编辑、管理员密钥设置、分组多图计费配置与文案。保留连接测试手动提示词与空白默认 hi/后台原输入、OpenAI OAuth 自动暂停、用量体积与权限、官方 TPS/整数点数/Excel、图片工作台与历史参数/SSE 心跳后 JSON/Base64 解码。保持窄屏可用与可辨认控件名称。

### Scenario: 前端可用且本地增强不退化

Acceptance: A6

WHEN 使用新增配置和原有管理/用户入口，THEN 双方功能可达且保存/读取一致；浏览器只对实际可执行的预览和服务场景提供证据，未运行部分明确记录。

## R7：迁移、生成代码与发布约束

VERSION 更新为 0.2.19.1，但不创建发布 tag。新增上游 246/247/248 迁移与本地迁移共同保留，既有 SQL 文件名和内容不修改。Ent/Wire 按官方流程生成并核对全部本地字段/provider，安全依赖不回退，Go 环境满足最终 go.mod/CI 并经 vfox 管理。fork 四段式版本解析、单 linux/amd64 归档/镜像矩阵与手动 workflow_dispatch 发布入口继续成立。

### Scenario: 可构建且迁移历史安全

Acceptance: A7

WHEN 核对 VERSION、迁移历史、生成代码与 CI，THEN 版本与范围一致，既有迁移无改写，双方字段接线保留，不触发 Release/部署。

## R8：当前候选验证

开发期仅定向检查。最终后端 default/unit/lint/embed 测试与构建，前端 lint:check/typecheck/test:run/build/i18n 由 Runtime 绑定候选执行，独立只读 Verifier 覆盖全部验收项。按实际环境执行隔离 PostgreSQL/Redis/fake upstream 与浏览器检查，未运行和 skip 不作通过。

### Scenario: 检查与验收真实绑定当前候选

Acceptance: A8

WHEN 提交候选验收，THEN 必要检查均有真实结果，新的 Verifier 独立核对范围、实现和证据；缺失服务与业务验证缺口明确报告。

## R9：流程和交付

单 change 顺序融合，不拆子任务。Agent 只编辑允许的正式产物，Runtime 管理状态、报告和检查。无法兼容的用户可见语义先取得决定；完成验收后等待用户接受结果与选择归档交付方式。不推送、不创建 PR、不发版、不部署，不重新跟踪本地记忆。实际结束完成项目经验和学习检查。

### Scenario: 交付范围与授权一致

Acceptance: A9

WHEN 从 Shape 推进到交付，THEN 按最新 continuation 和已确认范围执行；新决定和结果接受由用户确认，Git 和外部动作不越过授权，记录可复用经验而不将任务日志写为个人偏好。
