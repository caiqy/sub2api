# Outcome

沿用历史上游融合 change 惯例，从干净 main 基线 b3190a7a7a4f4b9d38384b8bed156807929dea55（包含 upstream v0.2.16，fork VERSION 0.2.16.3）出发，在独立分支完整融合调查时最新稳定上游 v0.2.17，同时保留 fork 定制，完成 Runtime 检查及新的独立只读验收。

# Scope

单个 Native change merge-upstream-tags-20261017，branch 隔离，工作分支 comet/merge-upstream-tags-20261017，目标 main。2026-10-17 获取并核对 upstream 远端稳定三段式 tag；唯一未包含版本为 v0.2.17，tag 对象 6d6c4e701a463dcf83c261e8428e93423d6c0b3e，peeled commit 060d0160417782cd962a0c7f460f6cd4565d4916。独立 no-ff merge 的第二父必须为该冻结 commit；tag 内 VERSION 为 0.2.16，身份按 tag 对象及 commit 而非该滞后文件判断。upstream/main 的 tag 后 VERSION 提交 b0a02a3304a0a78cdd003ddda1dd23ba02c03084、调查后的新增版本和未发布分支不自动进入范围。

融合 v0.2.16..v0.2.17 的全部 153 个变动文件，48 个提交，其中 66 个文件与 fork 改动重叠。内容包括 Gemini 4 Argon / 思考档位与流式摘要 / 计价和缓存存储费、Antigravity 模型目录与能力信息及工具图片/原生调用签名、Vertex 同号退避/服务账号失效/单次错误策略、可配置最小输出 token、OpenAI API Key 的 Messages Web Search 模拟、GPT-5.5 订阅账号退休、composite 路由凭据隔离、Anthropic 协议 base URL、Codex workspace 隔离、Responses created_at 补齐、支付退款/查单/回调/Stripe 安全、账号筛选及 GLM 用量窗口纠正，并同步上游文档与赞助资源。

专项保留 fork 最终模型校验与模型级临时冷却、路由目录、scheduler/sticky/freshDB/privacy/图片能力、请求体 cache/replay/handle/spool/释放和原始计时、用户平台额度与固定费用预占/mandatory billing、ModelTrace、账号连接测试自定义提示词、普通用户用量体积显示及权限边界、图片工作台与历史参数/JSON keepalive/Base64 本地解码、官方 TPS、Codex 点数整数显示、Excel/用量明细、额度周期及分组多语言等增强。

历史资料 docs/comet/archive/2026-10-13-merge-upstream-tags-20261013/ 和本地 memory/context/upstream-merge-workflow.md 仅作实施和验收惯例参考，历史通过不作本次证据。完整目标规格位于 specs/upstream-tag-fusion-20261017/spec.md。多个子域共享网关和账号核心，使用单 change，不拆 Supervisor。

## Directory structure

### Created

上游新增的 Gemini thinking、Vertex 错误、模型退休、Responses 时间戳、支付验证辅助文件及回归测试，docs/OPENAI_GPT55_RETIREMENT.md 和新增赞助图片；本 change brief 与完整目标 Spec。Runtime 创建状态、检查和验收报告；必要时新增最小融合回归。

### Modified

上游涉及的 backend/、frontend/、deploy/config.example.yaml、README 系列和赞助资源；本次融合根因修复及必要测试，backend/cmd/server/VERSION 更新为 0.2.17.1。若 schema/provider 因融合确有变化，按官方流程生成 Ent/Wire；其余核对生成稳定性。保留 .gitignore 对本地记忆的排除。

### Deleted

仅同步上游已删除且不承载 fork 需求的两个赞助图片及旧折叠筛选组件测试；删除测试前核对 fork 仍有效的断言应迁入现有合适回归，不能以删测试掩盖退化。不删除本地功能、SQL 或实际本地记忆文件。

### Not created

不创建 Supervisor children、新迁移、发布 tag、PR、额外持久服务或新的非必要依赖；不把 memory/ 与 CLAUDE.md 重新纳入 Git。

# Non-goals

本次不推送远端、不创建 PR、不打 tag、不触发 Release 或部署，不重写历史、不修改既有 SQL 名称/内容/checksum，不绕过 Hook 或修改信任配置。不做无关重构、UI 重设计或扩大无关上游存量问题的修复范围；本次新增支付确认入口必须维持金额等信任边界，不自动迁移管理员模型映射，不把原样透传扩大成全部事件重写。最终接受验收、归档和分支交付由用户明确选择。

# Acceptance examples

- A1：当前基线、远端稳定 tag 清单及冻结对象可核对；v0.2.17 有独立 no-ff merge，第二父精确为 060d0160417782cd962a0c7f460f6cd4565d4916，最终候选包含全部冻结版本且不误并 tag 后 upstream/main。
- A2：Gemini 4 Argon 的模型/档位/目录能力/计价、兼容入口 thinking 与流式摘要、Antigravity 工具图片和原生签名完整融合；Vertex 共享容量 429 对绑定缓存同号退避、失效服务账号置错换号及首次 401 单次判定保留正确语义，同时保留 fork 缓存绑定、固定费用/平台额度预留、结算和请求体生命周期。
- A3：最小输出 token 默认 128、最低 16 的配置和转换接线、OpenAI API Key Messages Web Search 模拟、composite OAuth 凭据隔离、Anthropic base URL、Codex workspace 导入和 Responses 时间戳按实际入口完整融合；保留最终模型身份、模型级冷却、协议回退/handle 重放/释放/计时。GPT-5.5 对 ChatGPT OAuth/SetupToken 及其影子账号在普通/sticky/compact/目录路径退休，API Key 不受该退休规则影响，显式替代映射有效且不自动改写存量配置或客户端请求。
- A4：支付回调经可信查单确认、EasyPay 数字字符串解析、订单类型校验、退款 pending 保持余额/订阅扣减和明确失败时原子回滚、支付宝查询异常与未找到宽限、Stripe 本地快照 secret 读取完整融合；fork 严格回调字段、验签/金额/商户身份、履约及退款幂等与订阅提交后缓存失效保留，未知字段包括空值仍拒绝；新增查单确认金额须为有限正数，NaN/Inf/缺失/非法或不符均不得确认，未确认支付不得入账。
- A5：上游账号筛选/Argon 白名单、Antigravity 图片连接测试、Stripe 跳转和 GLM 用量窗口修复生效，平铺/窄屏筛选保持功能和可辨认控件名称；fork 连接测试手动文本自定义 prompt 与空白默认 hi、后台原输入 . 和 usage probe 语义保持，普通用户用量体积/明细权限、图片历史与编辑、官方 TPS/整数点数/Excel/额度周期/ModelTrace 等定制不退化，原有记忆文件留在本地且 Git 不跟踪。
- A6：当前候选后端 default/unit 测试、兼容 Go 1.27.2 的 golangci-lint、embed 测试/构建及前端 lint:check/typecheck/test:run/build（含 i18n）由 Runtime 真实执行并绑定候选，新的独立只读 Verifier 覆盖 A1-A8。按实际条件开展隔离 PostgreSQL/Redis、支付本地 fake upstream 与前端浏览器实测；缺少服务/凭据时明确未执行范围，不把 skip/编译成功/历史结果当业务通过。
- A7：融合 VERSION 为 0.2.17.1；既有 SQL、schema、依赖安全版本、工具链及 fork 四段式单 linux/amd64 发布规则保留，本次上游无新增 SQL 或依赖升级。Ent/Wire 通过官方命令核对一致与稳定，Go 使用 vfox 管理的 1.27.2，不降低要求，不触发发布。
- A8：Agent 仅编辑正式 brief/Spec 等允许产物，阶段、检查、报告和事务由公开 Runtime 管理；候选及每项验收可追溯，真实语义冲突先取得决定，完整验收后等待用户接受及交付选择，实际结束时完成可复用项目经验与学习检查，不将进度或测试输出写作个人记忆。

# Constraints and invariants

冲突优先共存上游修复与 fork 定制，不机械选 ours/theirs。涉及新分支、循环、转换或金额路径的融合修复必须留下能复现根因的最小实际入口回归；不通过降低阈值、跳过断言或删功能通过检查。若双方业务要求不能共存，先在 brief 记录决定并按 Native 规则确认。

Responses 时间戳以实际代码边界为准：转换/非透传路径按上游新增 stamper 对缺失/null 补值，已有非 null 值（包括 0、字符串）保留，共享 stamper 为缺失值提供一致回退；本地合成错误附带 created_at 和 sequence_number。上游保持原样的透传事件不引入额外重写，透传中的本地合成错误仍需合法时间戳。不能由提交标题推定所有原样 HTTP/WS 事件均已覆盖。

人工连接测试自定义文本与上游新增图片 prompt 同时保留。对出站 HTTP payload、空白默认值和后台 probe 分别验证，不能让上游固定点号重置人工文本输入。

memory/ 和 CLAUDE.md 仅保存在本地，.gitignore 的排除继续生效，不将记忆内容写回版本库。已有 SQL 按完整文件名保持，不因历史重复数字前缀重命名。

# Verification expectations

开发期执行定向检查；完整最终计划由 Runtime 冻结候选后执行。后端沿用 scripts/test.ps1，重检查串行，避免 .test-tmp 被并发清理。shell 默认 Go 1.27.1，而项目需要 1.27.2；显式使用 vfox 已安装 Go 1.27.2 并核对 GOROOT/PATH/GOTOOLCHAIN，复用已有 Go 1.27.2 构建的 golangci-lint 2.14.0；Node 22.23.1、pnpm 9.15.9 保持现有环境。

本轮调查未发现 Docker/PostgreSQL/Redis 命令及对应服务、已运行的项目后端或前端预览。Build/Verify 复核环境后，优先使用本地 stub/fake upstream；有隔离服务时执行实际集成，缺少则明确未执行。默认进行浏览器预览；依赖后端登录/真实支付才能验证的场景不可用时列明缺口，不以静态页面或 fake 结果宣称真实资金/账户验证。不得使用生产凭据或真实付款。临时数据库仅停止本次创建的实例并确认测试端口关闭，不停止用户原有数据库。
