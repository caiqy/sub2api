# 逐版本上游融合完整目标规格

## R1：冻结版本和 Git 拓扑

在独立分支 comet/merge-upstream-tags-20261007，从 main 基线 1171052e7f5b3b607a49b9fdc940ef2d084fc13d / VERSION 0.2.13.1 开始。已核实 v0.2.13 是基线祖先；2026-10-07 调查的最新稳定 tag 为 v0.2.14，tag 对象 1400a7b482974d98db5b284a8b2afbe3eaf9aaef，commit 0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d。本次只有这一版待融合，采用 no-ff merge，第二父为该 commit，保持两侧历史。新出现的 tag 不自动加入范围。

### Scenario: 上游版本成为可核对的祖先

Acceptance: A1

WHEN 查看冻结 tag、最终历史及 merge 父提交
THEN 基线保留、第二父精确对应 v0.2.14、最终 HEAD 包含该版且没有遗漏中间稳定版本。

## R2：支付安全和结算

融合 EasyPay 通知允许字段集合，拒绝创建订单专属及其他未知回调字段，阻断订单签名复用为支付成功通知。CanonicalizeReturnURL 清除用户 query 和 fragment，由既有构造器加入受控返回参数。保留合法签名验证、真实支付 QueryOrder reconcile、fork 充值优惠/赠送/返利/退款/履约和幂等路径。使用本地测试验证合法回调、伪造回调与 URL 清理，不访问真实支付服务。

### Scenario: 订单签名不能复用为成功回调

Acceptance: A2

WHEN 回调带 notify_url、return_url 或其他未知参数，或用户 return_url 嵌入 trade_status 等 query
THEN 非法回调被拒绝且查询参数被清除；合法通知和支付对账回退仍可工作，既有结算语义保持。

## R3：新安装管理员安全

缺省邮箱和密码仅在确需创建管理员时随机生成，并提供上游一次性保存提示；Web/CLI/环境变量入口使用一致邮箱登录兼容校验及 bcrypt 72 字节上限。保留明确有效的用户配置。既有管理员或已有用户的部署遵循既有初始化判定，不重置凭据或意外创建新管理员。部署示例及中英日文档移除可猜测默认凭据且保持 fork 部署约束。

### Scenario: 新安装安全且旧部署不重置

Acceptance: A3

WHEN 缺省配置首次初始化，或既有用户数据库再次启动
THEN 首次创建使用不可猜测的随机凭据且可登录；旧部署不重新初始化，非法邮箱和密码在创建前被拒绝。

## R4：客户端目录和依赖

在 OpenAI HTTP/WS、Grok 及其他现有 Codex 配置生成分支中，远端 catalog 配置包含 api_key_model_discovery = true，本地文件模式维持原有行为。更新 Vue 与 source-map-js 及锁文件，保留 fork 样式化 Excel writer/vendor 和导出能力。审计例外必须与实际只写不读的使用路径相符，不机械套用不适用于 fork 的描述。

### Scenario: 远端目录发现与文件目录兼容

Acceptance: A4

WHEN 生成各平台的远端或本地文件 Codex 配置，并检查依赖、导出及审计使用路径
THEN 远端启用 API Key discovery，本地文件语义保持；更新后的依赖可构建、测试，fork Excel 样式和数据行为保留。

## R5：fork 不变量与回归修复

保留最终模型身份/请求校验、调度/sticky/freshDB、privacy/图片过滤、请求体 replay/cache/释放时序、额度预占和 mandatory billing、ModelTrace、日志和 Excel 导出、配置热更新、额度周期持久化锚点/有效期按钮/分组自助提前开关、部署资源限制及发布规则。核查真实入口和所有相关调用路径；未变动区域可结合相对基线 diff 与现有测试证明保留。非平凡融合回归先留下最小失败复现，再修共享根因。无法共存的产品语义须用户决定，不删除相反断言换取通过。

### Scenario: 安全更新不削弱 fork 功能

Acceptance: A5

WHEN 核对上游改动和本地定制的交互
THEN 核心定制保留；发现融合回归时存在可运行复现和最小根因修复，真实语义冲突未未经确认选择一侧。

## R6：真实检查和独立验收

开发期定向测试；最终候选交 Runtime 实际运行完整后端 default/unit、golangci-lint、构建及前端 lint:check/typecheck/test:run/build，build 含 i18n。后端使用 Windows 既有测试脚本，重检查串行。新的只读 Verifier 独立核对 A1-A8、源码和绑定记录，历史通过只作参考。真实 PostgreSQL/Redis/浏览器等环境具备时进行相关验证，缺失或跳过的检查记录为未运行，不当通过。临时数据库只清理本次创建实例并验证端口关闭，不停止现有实例。

### Scenario: 候选冻结后真实验收

Acceptance: A6

WHEN Builder 交接完成候选
THEN Runtime 执行绑定检查，新的独立 Verifier 覆盖全部验收项，并明确服务集成和浏览器实际覆盖边界。

## R7：版本、迁移、生成和发布兼容

VERSION 为 0.2.14.1，仅为融合版本，不创建 tag 或发布。保留既有 SQL 文件名与内容，完整文件名主键和 checksum 不变。上游无 schema/provider 变动；若实际融合涉及生成输入，使用官方 Ent/Wire 生成并核对稳定，否则复核生成文件及输入未改变。保留 fork 四段式版本解析和既有单架构发布矩阵。

### Scenario: 版本一致且已有迁移稳定

Acceptance: A7

WHEN 核查 VERSION、SQL、生成输入输出和发布配置
THEN 版本为 0.2.14.1，已发布 SQL 不变，生成与输入一致，四段式和构建矩阵保留且没有触发发布。

## R8：Comet 工作流和交付

正式 brief/spec 由 Agent 编辑，状态、检查和报告由公开 Runtime 管理。全部验收后等待用户明确接受及选择工作区收尾；本次在本地分支形成可复核成果。未包含推送、PR、发版、部署或清理其他工作区授权。结束记录已验证且可复用的项目经验，并完成个人记忆学习检查，不将进展和日志当用户偏好。

### Scenario: 独立验收完成后选择交付

Acceptance: A8

WHEN 全部验收项完成并有真实候选和检查记录
THEN 展示成果并等待接受与交付选择，未经授权不推送、发版或部署，按 Runtime 完成后续记录。
