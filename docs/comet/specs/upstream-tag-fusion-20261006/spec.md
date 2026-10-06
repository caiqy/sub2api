# 逐版本上游融合完整目标规格

## R1：基线、版本与拓扑

在独立分支 comet/merge-upstream-tags-20261006，从 main 的 9e3106ddb645017938de7a4aa82f56d4837e82e9 / VERSION 0.2.11.3 开始。顺序合并 v0.2.12（5106065716e494204fc0e8db16f68f6e9d576be0）和 v0.2.13（3040209f205472038c1ba745a1bedd2edd9053b1），每版独立 no-ff merge，第二父必须精确对应冻结 commit。调查后的新增版本不自动进入本次范围。

### Scenario: 冻结上游版本按序成为祖先

Acceptance: A1

WHEN 查看最终 Git 历史和两个 merge 的父提交
THEN 两版均为最终 HEAD 的祖先，v0.2.12 先于 v0.2.13 融合，main 基线可恢复。

## R2：每版闭环

每版先处理全部冲突、核对能力交互、运行完整后端/前端门禁并完成独立只读审查，再创建可恢复提交，随后进入下一版。提交保留上游原历史及 fork 历史，不以 ours/theirs 批量选择替代语义融合。两版共享核心区域，单个 change 串行实现。

### Scenario: 第一版失败时停止进入下一版

Acceptance: A2

WHEN v0.2.12 检查失败或独立审查存在未解决融合回归
THEN 先修复并补齐实际验证，不提前合并 v0.2.13；环境阻塞、未运行、超时保持真实状态。

## R3：上游能力

保留 TypeSafe/Jev System One 原生协议、独立端点及模型列表隔离、平台迁移、调度快照、内容审核和审计；保留充值优惠/赠送档位和推广返利基数；保留账号优先级快捷操作、API Key 分组分页排序、邮件验证码原子尝试计数与 hashed single-use reset token、公开订单验证限流、Antigravity 错误净化、Grok CLI 身份头及 Axios 更新。v0.2.13 保留删除 API Key 后的 usage 结算修复和 TypeSafe API Key 账号计费探针。

### Scenario: TypeSafe 和计费改动完整融合

Acceptance: A3

WHEN 检查新增平台链路及删除 API Key 后的结算路径
THEN 上游端点隔离、安全校验、审计、计费探针、结算行为保持，并可与 fork 定制同时工作。

## R4：fork 不变量

保留 public/route/channel/account 模型身份与最终模型/请求校验；保留调度、sticky、freshDB、privacy、图片能力过滤和热更新；保留 replay/body handle 以及缓存协议在最终映射后选择、发送前归一化、原始媒体大 body 的释放时序；保留额度预占、释放与 mandatory billing；保留 ModelTrace、使用详情日志和 Excel 导出。额度周期提前功能使用持久化锚点且幂等，重置按钮依据有效期展示，分组开关可关闭自助提前周期。复核所有入口、回退和边界路径。

### Scenario: 新平台合入后已有 fork 路径仍成立

Acceptance: A4

WHEN 运行相关回归并检查新增分支与 fork 共享调度/计费/请求体链路
THEN 已有语义保持；近期额度周期锚点、按钮与分组开关未被丢失。

## R5：回归修复边界

仅修复本次合并或上下游交互导致的回归。非平凡问题先用最小可运行回归复现失败，再在共享根因路径修复并验证所有调用方；使用现有工具、依赖及项目模式。未经批准的真实产品语义冲突须请求用户决定，不随意删除相反断言换取测试通过。

### Scenario: 发现共享网关回归

Acceptance: A5

WHEN 同一逻辑有多个调用入口，融合造成语义错配
THEN 留下能失败的实际链路测试，在共同路径修复并确认相关入口，保留失败到通过的证据。

## R6：验证和独立验收

每版运行后端 default/unit 测试、golangci-lint、构建及前端 lint:check/typecheck/test:run/build，build 自带 i18n 完整性检查。后端使用既有 Windows scripts/test.ps1 并串行执行，避免共享 .test-tmp 被并发删除。最终候选的完整检查计划由 Runtime 执行并绑定，新只读 Verifier 独立覆盖 A1-A8。缺少 PostgreSQL/Redis/Docker 等隔离服务时记录 integration 缺口，不把未运行标为通过。测试不得向第三方发送未知凭证。

### Scenario: 候选冻结后验收

Acceptance: A6

WHEN Builder 完成全部条目并交接候选
THEN Runtime 执行真实完整检查，新的独立 Verifier 核对全部行为、证据版本和边界，明确服务集成验证的实际覆盖。

## R7：版本、迁移及生成代码

每版 fork VERSION 分别设为 0.2.12.1 和 0.2.13.1，仅作为本地融合版本，不触发发布。保留已发布迁移的名称和内容，新增 upstream SQL 与 fork SQL 可按完整文件名同时执行，核对同 241 前缀的 SQL 依赖与 checksum 不变量。Ent/Wire 按官方 generate 命令生成并核对稳定性，不通过手改生成代码隐藏 schema/provider 不一致。保留四段式版本解析和现有发布构建矩阵。

### Scenario: 新 SQL 与 ModelTrace 迁移共存

Acceptance: A7

WHEN 合入两个上游 241 SQL 并生成后端代码
THEN 已有 SQL 名称及 checksum 不变，新增迁移分别记录且顺序依赖安全，生成代码与 fork schema/provider 一致，最终 VERSION 为 0.2.13.1。

## R8：工作流和交付

正式 brief/spec 由 Agent 填写，状态、检查和报告由 Comet Runtime 管理。历史验收仅作经验，不算本次证据。最终独立验收通过后等待用户明确接受，并按用户选择保留分支或本地合回 main；推送、PR、发版和部署不在本任务范围。任务结束记录验证过的可复用项目经验及学习检查，不把命令输出/进展写成个人偏好。

### Scenario: 独立验收结束等待交付

Acceptance: A8

WHEN 全部验收项有真实结论
THEN 展示可 review 的分支与结果，由用户接受并选择 Comet 交付方式，未经授权不推送、不发版、不部署。
