# Upstream tag fusion 20261017

## R1：冻结版本、基线与融合拓扑

基线 b3190a7a7a4f4b9d38384b8bed156807929dea55 位于 main，与 origin/main 相同且工作树干净。基线已含最高上游稳定 v0.2.16，fork VERSION 为 0.2.16.3。2026-10-17 调查 upstream 远端全部稳定三段式 tag 后，未合入的只有 v0.2.17：tag 对象 6d6c4e701a463dcf83c261e8428e93423d6c0b3e，peeled commit 060d0160417782cd962a0c7f460f6cd4565d4916。v0.2.16..v0.2.17 共 48 个提交、153 个文件，66 个文件与 fork 改动重叠。

在 comet/merge-upstream-tags-20261017 分支完整融合冻结版本，保留双方历史，独立 no-ff merge 的第二父精确为冻结 commit。upstream/main 的后续 VERSION 提交 b0a02a3304a0a78cdd003ddda1dd23ba02c03084 和调查后的新版本不自动纳入。稳定版本检查以真实 upstream 远端清单和冻结区间为准，不使用无关本地旧 tag 作为待合并清单。

### Scenario: 独立冻结版本融合

Acceptance: A1

WHEN 核对最终候选的祖先、merge 第二父、tag 对象和本次远端稳定版本清单
THEN 基线及 v0.2.17 均可达，第二父和 tag 对象精确匹配冻结值，只有确认的版本进入范围，无遗漏和误并 tag 后 main。

## R2：Gemini、Antigravity 与 Vertex 的新增行为

Gemini API、CLI 与 Antigravity 支持 gemini-4-argon 及其 low/medium/high 档位；同步模型目录时按账号真实端点获取，保存上下文窗口、图片输入、thinking 等能力。Argon 的普通 token、reasoning 和显式缓存存储费使用对应计价；Claude 5.5 tier ID 按完整模型语义匹配基础计价，保留旧 Antigravity 映射运行时默认透传，不通过模糊子串过量计费。

Gemini Chat/Messages 兼容入口把 reasoning_effort / thinking 转为 Gemini thinking 配置，流式输出思考摘要，正确区分思考与正文并保持合法事件顺序/usage。初发和 signature retry 都从正确的 canonical/stripped 请求应用配置，不恢复过时 []byte 路径或让借用 handle 的所有权转移；模型目录 capability 元数据与真实端点一致，不让自动注入的默认透传污染管理员显式目标或能力筛选。Antigravity 的 tool_result 图片、原生 Gemini functionCall/thoughtSignature 及 composite /v1beta 入口保留。

Vertex 共享容量不足 429 使用上游定义的同号退避，不对绑定显式缓存的请求错误冷却换号或直接 503；被删除的服务账号置 error 并允许换号，失败必须释放相关资源。每个上游错误只调用一次错误策略，首次 401 不在同一请求内重复累积停用。

同时保留 fork 请求体压缩/handle/spool/缓存绑定与账号固定、sticky、最终有效模型、配额平台选择、固定费用 ReserveFixedCharge、并发 pending、API Key/余额/订阅/用户平台额度及 mandatory billing/失败回滚，不截断已知固定收费预占金额。

### Scenario: 新模型与绑定缓存错误策略共存

Acceptance: A2

WHEN 实际兼容入口接收 Argon thinking/工具图片请求，或绑定 cachedContents 的 Vertex 请求遇到共享容量 429、失效 SA、首次 401
THEN 模型/能力/计价和思考事件正确；错误只判定一次并选择规定的同号退避或置错换号；fork 最终模型、平台费用预占/结算及请求体释放在成功、重试和失败路径保持。

## R3：模型与协议入口安全融合

配置 gateway.min_output_tokens 默认 128、最小允许 16，配置加载/验证、服务实例与所有上游本次改动的协议转换入口接线一致；不能因 fork 预构造或绑定 handle 而走回旧的固定阈值。其语义是协议转换中的正数 token 上限 floor，不扩大为所有原生 Responses 请求强制补 token。覆盖 Chat/Anthropic 转换及 raw Chat 回退、legacy Responses messages 的 HTTP/passthrough compatibility/WS 首轮和后续 response.create，保持省略/0 的原转换语义。

OpenAI API Key 账号在 /v1/messages 入口按账号/渠道 Web Search 模拟配置执行，OAuth 仍不进入模拟；保持 fallback、安全上游与工具结果语义。明确区分模拟与 fork 原生 Grok 搜索，沿用各自既有计费规则，不擅自新增/重复搜索费；适用的 SearchCount、有效 QuotaPlatform、固定费用预占与 mandatory usage 不能因提前返回丢失，队列拥塞/取消后仍需正确结算与释放。composite 模型归属不明确且含 OpenAI OAuth 时拒绝误入 Anthropic，明确归属及合法 Gemini/Antigravity 路由继续生效。原生 Anthropic 和 count_tokens 使用账号 Anthropic 协议 base URL，保留最终模型校验、专用错误优先级及此前 count_tokens prepared 释放后的安全错误归因。

GPT-5.5 退休限定为 ChatGPT 支撑的 OpenAI OAuth/SetupToken 和凭据影子账号：普通调度、sticky、compact、公开模型列表、远端/固定 Codex 目录和 ETag 都按最终目标及一次别名映射处理。API Key 账号不受该退休规则影响，仍需实际模型可用；映射到支持的替代模型仍允许，映射到已退休 GPT-5.5 的订阅目标拒绝。无法服务返回既有 model_not_found 和迁移提示，不静默改写请求、历史计价或管理员配置。

Codex 导入以同一用户下各 workspace 区分身份，保留原幂等边界。Responses created_at 按上游实际覆盖范围补齐：缺失/null 补值，已有非 null 字段（包括 0 或字符串）保留，不强制已有显式值相同；共享 stamper 为缺失值提供一致回退。普通 HTTP JSON 的 response 对象、SSE 转 JSON、普通 SSE 生命周期和 HTTP→WS 桥按上游实际入口覆盖，桥接在 replay collector 前 stamp，本地合成失败还包含 sequence_number（可用时复用被抑制 bare-error 的序号，否则 0）。原样透传未新增 stamper 的 HTTP/WS 事件维持原始语义，不以提交标题要求全面重写，但本地合成失败仍合法。

fork 模型目录、模型级临时冷却、sticky/WaitPlan/freshDB、privacy、图片能力、缓存到最终映射时序、handle 所有权/重放/释放与请求开始时间保持。遇到语义融合修改时，以真实入口和 fallback/重试生命周期做最小回归。

### Scenario: 最终模型与不同凭据类型的协议契约

Acceptance: A3

WHEN 通过别名、sticky、compact、模型目录或转换入口访问模型，或发生 composite 归属歧义、Anthropic base URL 路由与 Responses 回退
THEN GPT-5.5 仅拒绝最终退休的订阅目标，API Key 和有效替代映射保持；凭据不会误入另一平台，最小输出配置和时间戳按定义生效，fork 映射/handle/冷却/计时边界不退化。

## R4：支付、退款与 secret 的安全边界

完整融合 EasyPay 彩虹/ZPay 查单参数与数字字符串解析、回调通知可信查单确认后履约、订单类型允许集合校验、退款 pending 期间保持余额/订阅扣减、网关明确失败时原子回滚、支付宝退款状态查询/异常保留待定/未找到宽限，以及 Stripe client_secret 从本地授权订单快照读取、不经 URL 传递。

fork EasyPay VerifyNotification 在验签和履约之前拒绝允许集合外的所有字段，包含值为空的未知字段；合法通知继续验签并核对金额、商户与订单身份。新增查单确认入口的金额必须是有限正数，NaN、Inf、缺失、非法或与通知不符时不能确认入账。查单确认与严格字段守卫必须同时成立，不能以放宽未知参数兼容上游。回调/主动查询/resume/取消/到期的合法恢复路径保持幂等，订阅履约提交后正确使缓存失效，购买页隐藏校验不被绕过。现有周期 reconcile 只扫描 alipay/wxpay 及相应前缀，不覆盖 easypay 自定义方式；合法恢复依赖 owner 主动 verify、有效签名 resume 或到期查单，不承诺立即自动补单。匿名 VerifyOrderPublic 仍只读持久态，不放宽主动查询。保留支付状态认领/履约 lease/redeem/退款的自身幂等，不扩大成原本未接入的通用 Idempotency-Key 下单承诺。

使用本地 provider stub/fake upstream 验证资金状态转换，覆盖余额与订阅两类 held deduction、成功不重扣、失败退还及提交后缓存失效；可用数据库时补审计/提交故障的真实事务回滚检查。不能用真实支付或生产凭据。mock 成功路径不能作为真实事务回滚证据，数据库事务未实跑时明确边界。

### Scenario: 不可信回调拒绝与 pending 退款一致性

Acceptance: A4

WHEN 未知空字段通知、伪造/金额不符/商户不符通知、重复合法确认或 pending 退款的成功/失败/查询异常到来
THEN 非法通知拒绝且不入账，合法确认仅履约一次，pending 保留预扣、明确失败原子恢复且不重复恢复；Stripe URL 无 secret，授权快照仍可完成支付流程。

## R5：前端与近期 fork 定制

融合上游账号筛选恢复平铺和窄屏布局、Argon 模型白名单、Antigravity OAuth 图片连接测试与图片事件、Stripe 跳转及 GLM 5h/weekly 用量窗口纠正。平铺筛选保留五类筛选/search 的合并参数与 debounce、不清除其他条件，控件保留可辨认名称，不随旧折叠布局测试删除有效行为或无障碍基础。同步 README、GPT-5.5 迁移文档和赞助资源；不借机重设计 UI。

保留 fork 人工连接测试自定义 prompt：文本 trim 后为空使用 hi，否则传用户文本到真实出站 HTTP；Antigravity OAuth 后台 RunTestBackground 原输入 .、usage probe 固定 hi、Grok quota builder 保持各自语义。融合上游 TestConnectionWithPrompt 与图片分支时，不能让文本测试退回固定点号。管理员/普通账号两处 modal 和后端多协议路径共同核对。

保留普通用户 UsageView 的 request_body_size/response_body_size/总计与颜色，大小来自独立快照、不依赖清理后的原文，管理员与普通 DTO 共用大小但普通用户不获 has_detail/请求明细权限。保留图片工作台参数/历史及安全脱敏、JSON keepalive 后普通 JSON 解析、Base64 本地解码、官方 TPS、Codex 整数点数、Excel/ModelTrace/额度周期和分组多语言等功能。

memory/ 与 CLAUDE.md 的文件保存在本地但不再被 Git 跟踪，.gitignore 排除持续生效；不因同步上游或归档恢复这些跟踪文件，不删除实际本地记忆。

### Scenario: 近期 fork 能力跨上游改动保留

Acceptance: A5

WHEN 账号连接测试提交自定义或空白 prompt、执行后台探测/图片测试，普通用户查看用量，或核对图片历史及本地记忆
THEN 对应出站请求分别保持既有文本/后台默认并支持新增图片；用量体积与权限正确；其他 fork 展示/历史能力保留，记忆文件仍在本地且 Git 未跟踪。

## R6：候选绑定验证与独立验收

开发期只做有意义的定向检查。Builder 将完整最终计划提交给 Runtime 冻结候选后执行：后端 scripts/test.ps1 默认 ./...、-tags=unit ./...、兼容 linter、前端生产构建后的 embed 测试与 embed 构建；前端 pnpm lint:check、typecheck、test:run、build（含 check:i18n）。按故障诊断修共享根因，不删除有效断言或降低阈值。

新的只读 Verifier 独立判断全部 A1-A8，可复用仍与当前候选及输入匹配的 Runtime 通过记录，只补缺失/失效证据。失败、timeout、skip、未运行及历史通过不作为当前通过。

复核环境后优先做本地 fake upstream 和隔离集成/浏览器预览。真实 PostgreSQL/Redis、认证后端或支付凭据不可用时，在报告明确哪些测试仅编译、哪些业务未执行；不得把测试框架自动 skip、静态页面或模拟结果当真实支付/数据库/登录通过。临时 DB 只清理本次实例并确认端口关闭，不影响用户原服务。

### Scenario: 完整门禁与受限环境证据

Acceptance: A6

WHEN Runtime 在最终候选上执行完整计划并启动新的只读 Verifier
THEN 每项结果有真实退出码/覆盖范围及候选绑定，全部验收独立判断，隔离集成/浏览器的可用性和未执行部分如实披露。

## R7：版本、迁移、生成和发布规则

本次融合 VERSION 为 0.2.17.1；已核对 origin 无该基准的派生 tag，但本次仅修改本地版本，不创建 tag 或触发发布。

上游范围没有新增/修改 SQL、Ent schema、Wire provider、Go/前端依赖清单及锁、CI/Docker/发布流程；保留 fork 所有既有 SQL 全名、内容和 checksum，包括重复数字前缀的合法迁移。官方 go generate ./ent 与 go generate ./cmd/server 核对合并输入和输出稳定，不手改生成文件隐藏漂移。

项目 Go 1.27.2 和安全依赖保持，SDK 通过 vfox 使用，不降级或改变用户全局配置。显式核对 GOROOT/PATH/GOTOOLCHAIN，复用 Go 1.27.2 编译的 golangci-lint 2.14.0，避免旧工具 export data 格式不兼容。

保留 fork 四段式 tag 解析、单 linux/amd64 构建矩阵、tag 后 workflow_dispatch 唯一 Release 入口和部署资源。若未来另行授权发布，仍按当时 HEAD 包含的最高 upstream 稳定 tag 与 origin 派生 tag 计算，不因本次融合自动发布。

### Scenario: 本地版本及不可变迁移

Acceptance: A7

WHEN 检查 VERSION、全部既有 SQL、官方生成前后哈希、模块与发布脚本
THEN 版本为 0.2.17.1，迁移名称/内容不变，生成一致稳定，Go/安全依赖/四段式单架构规则保留，未产生发布动作。

## R8：Native 阶段与交付

单 Native change，不拆 Supervisor；按本地配置在独立分支推进。Shape 的完整目标和验收经用户明确确认后才实施；实际业务语义无法同时保留时记录新的决定并按 Native 规则确认。Agent 只编辑允许的正式产物，状态、检查、报告和事务由公开 Runtime 管理。

Builder 提交可审查候选，Runtime 检查后由新 Verifier 验收全部标准。全部通过后等待用户明确接受及选择本地 merge/保留分支等交付方式。归档、merge、push、PR、发布、部署和清理的授权分别处理，本次请求不包含后四者。

实际结束时记录已验证可复用的项目经验与学习检查；项目记忆仍仅本地保存，不把任务进度、测试日志或一次性输出当个人记忆。

### Scenario: 可追溯验收和用户交付选择

Acceptance: A8

WHEN 当前候选完成全部独立验收
THEN 报告与候选可追溯，等待用户接受结果及交付选择，未授权推送/发布/部署不执行，真正结束时完成学习记录且不恢复记忆文件跟踪。
