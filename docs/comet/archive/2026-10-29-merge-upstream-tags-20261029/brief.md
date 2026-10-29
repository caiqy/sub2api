# Outcome

从干净 main 基线 dd3829d66cfe69719b20103c488531ad5c4b70a7（包含 upstream v0.2.17，fork VERSION 0.2.17.5）出发，在独立分支按顺序完整融合调查时最新稳定上游 v0.2.18、v0.2.19，保留 fork 定制，完成 Runtime 检查及新的独立只读验收。

# Scope

单个 Native change merge-upstream-tags-20261029，branch 隔离，工作分支 comet/merge-upstream-tags-20261029，目标 main。2026-10-29 获取 upstream 远端稳定三段式 tag，按数值版本与祖先关系确认待合并版本如下；每版一个 no-ff merge，第二父须精确匹配冻结 commit。

| 顺序 | Tag | Tag 对象 | Peeled commit |
| --- | --- | --- | --- |
| 1 | v0.2.18 | de86961b627e48db75d61fff8c05535d44a00cbe | ac1b8f1271be0d674fa6f074403c1410f30a3a81 |
| 2 | v0.2.19 | e60cefeaf78f2fbfa95aa076029e28b40d0a5527 | dece26153621959ccd7635814f4b2d08304e681a |

融合 v0.2.17..v0.2.19 的全部 353 个变动文件。v0.2.18 包括调度快照内存缓存、模型映射签名、限流快照写时复制、单 Antigravity 探测、Vertex 配额、兼容流断连计费和错误收尾、Gemini 分组生图门禁、图片转存和账号筛选；v0.2.19 包括按操作与资源授权的管理员 API 密钥、Grok 模型目录同步/诊断/响应模型审计、分组多图计费、GPT-6.1 Terra、Gemini 模型与显式缓存同号重试、DeepSeek 缓存计费/角色兼容、分页与上传边界、渠道 web search 更新与依赖安全升级。

专项保留 fork scheduler/sticky/freshDB/privacy/图片能力/分组生命周期栅栏、最终模型身份与模型冷却、请求体 cache/replay/handle/spool/释放/计时、用户平台额度与固定费用预占/mandatory billing、响应模型计费、ModelTrace 与额外认可模型快照、连接测试自定义提示词、图片工作台/历史参数/SSE JSON/Base64 解码、普通用户用量体积与明细权限、官方 TPS/整数点数/Excel、订阅手动及自动额度周期提前与行锁/幂等回执/版本化缓存失效、OpenAI OAuth 自动暂停入口等本地增强。

既有 memory/context/upstream-merge-workflow.md 和历史归档仅作实施与验收参考，历史通过不能替代本轮证据。完整目标规格位于 specs/upstream-tag-fusion-20261029/spec.md。两个版本顺序依赖，且共同修改网关、账号和计费核心，使用单 change，不拆 Supervisor。

## Directory structure

### Created

同步上游新增源文件、测试、管理员密钥说明、迁移 246/247/248 及赞助资源；正式 brief/Spec 和必要的最小融合回归。Runtime 管理状态、检查与报告。

### Modified

上游涉及的 backend/、frontend/、CI、Dockerfile、README 与资源；冲突融合及必要测试。融合 VERSION 设为 0.2.19.1（目前 origin 无此派生 tag）；只更新文件，不发布。按官方流程核对 Ent/Wire，保留 fork 新增字段与接线。工具链按合并后 go.mod 与 CI 要求使用 vfox 管理，不降级。

### Deleted

仅同步上游实际删除且不承载本地有效能力的文件。相反测试须追溯来源与业务意图，不能以删除断言通过门禁。不删除本地记忆或既有迁移。

### Not created

不创建 Supervisor children、发布 tag、PR、额外持久服务或非必要依赖，不将 memory/ 和 CLAUDE.md 重新纳入 Git。

# Non-goals

不推送远端、不创建 PR、不打 tag、不触发 Release 或部署，不重写历史、不修改既有 SQL 内容/文件名/checksum，不绕过 Hook 或修改信任配置，不做无关重构或 UI 重设计。调查后的新增版本、tag 后 upstream/main 不自动纳入范围。最终结果接受、归档和交付方式由用户明确选择。

# Acceptance examples

- A1：基线、远端正式版本及冻结对象可核对；v0.2.18、v0.2.19 各有顺序独立 no-ff merge，第二父精确匹配冻结 commit，最终候选包含全部目标版本，不误并 tag 后 upstream/main。
- A2：v0.2.18 调度缓存/模型签名/限流快照优化、单 Antigravity 探测和 Vertex 配额完整融合；fork scheduler/sticky/freshDB/privacy/图片能力、模型冷却、分组生命周期栅栏及运行时热更新保持语义正确，不共享可变快照造成跨请求污染。
- A3：兼容流与 Gemini 断连计费、流式错误及 overload 重试、Gemini 生图门禁/缓存重试/模型列表、DeepSeek 兼容和图片上传/转存修复从实际入口生效；保留 fork 最终模型映射、协议回退和请求体 handle/cache/spool 重放释放时序、专用超时及客户端关闭语义。
- A4：新增 scoped admin API keys 的操作和资源范围、事务与幂等、返回数据裁剪、限流和存量管理员鉴权完整融合，未授权资源不可访问且不得泄露敏感字段；保留 fork 管理员增强、普通用户权限与用量明细边界、订阅自动及手动额度周期功能和并发安全。
- A5：Grok 目录同步/诊断与响应模型审计、分组多图计费、DeepSeek 缓存命中计费、GPT-6.1 Terra 和渠道 web search 更新完整融合；保留 fork 响应模型计费边界、用户平台额度/固定费用预占/mandatory billing、ModelTrace 认可快照和现有定价语义，不重复或漏结算。
- A6：前端账号筛选/配额/Grok 诊断/管理员密钥/分组图片计费等更新正常接线；fork 连接测试自定义提示词、OAuth 自动暂停、用量体积/官方 TPS/整数点数/Excel、图片工作台/历史参数/SSE JSON/Base64 等定制保留。完成可行的本地预览与浏览器验证，缺少登录或服务时如实列明未覆盖范围。
- A7：VERSION 为 0.2.19.1；合并新增迁移与 fork SQL 无覆盖，所有既有迁移内容及名称保持；Ent/Wire 保留双方字段并按官方命令核对；安全依赖和 Go 版本要求不降低，fork 四段式版本与单 linux/amd64 发布矩阵保留且未触发发布。
- A8：最终候选后端 default/unit、lint、embed 测试/构建和前端 lint:check/typecheck/test:run/build（含 i18n）由 Runtime 真实执行并绑定候选，新的独立只读 Verifier 覆盖 A1-A9。数据库/Redis 及 fake upstream 定向验证按实际环境执行；skip、超时、未运行和历史结果不能算业务通过。
- A9：只编辑允许的正式 brief/Spec 等产物，阶段/检查/报告/事务由公开 Runtime 管理；真实不可共存的语义冲突先确认，完整验收后等待用户接受与交付选择；结束时完成项目经验和学习检查，不将进度/测试输出写作个人记忆。

# Constraints and invariants

冲突保留上游变更与 fork 定制，禁止机械选 ours/theirs。复杂融合修复先用失败用例复现并保留最小实际入口回归；不得降低断言或删除有效测试掩盖退化。订阅与计费等资金/权限路径不得简化信任边界或事务语义。

v0.2.19 tag 内 VERSION 为 0.2.18，tag 身份按远端对象与 peeled commit 判断。最终 VERSION 按 fork 惯例显式改为 0.2.19.1，发布仍需另行授权。

# Verification expectations

开发期仅执行定向检查；完整最终计划交由 Runtime 冻结候选后运行，串行安排重型 Go 检查避免测试临时目录竞争。独立只读 Verifier 由宿主子代理提供。临时数据库/服务仅操作本次启动实例，结束确认端口关闭，不停止用户原有数据库。缺少本地 PostgreSQL 时明确跳过相应集成，不使用生产凭据或真实付款。工具链按 vfox 与项目现有脚本管理；若最终检查发现融合问题，回 Build 修复后再提交。
