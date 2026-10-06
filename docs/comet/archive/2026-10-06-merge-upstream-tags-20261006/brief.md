# Outcome

从 main 基线 9e3106ddb645017938de7a4aa82f56d4837e82e9（VERSION 0.2.11.3，已包含 upstream v0.2.11）开始，使用 Comet Native 在独立分支 comet/merge-upstream-tags-20261006 逐版本融合 upstream v0.2.12、v0.2.13，保留 fork 定制并完成可复核的验证和独立验收。最新版本以 2026-10-06 本次调查冻结的正式 tag 为准，最终 VERSION 为 0.2.13.1。

# Scope

单个 Native change，branch 隔离，目标分支 main。两版共享网关、调度、计费、生成代码及前端区域且严格串行，不拆 Supervisor 子任务。完整目标规格位于 specs/upstream-tag-fusion-20261006/spec.md。

冻结范围：

- v0.2.12：tag 对象 cbe9966432317e6757b47f598a103d2132ec27db；commit 5106065716e494204fc0e8db16f68f6e9d576be0。
- v0.2.13：tag 对象 7d0c0067f406c380f0a94cfc3879cdae7049b467；commit 3040209f205472038c1ba745a1bedd2edd9053b1。

每版独立 no-ff merge，第二父对应冻结 commit。先完成该版冲突融合、回归验证、独立只读审查及可恢复提交，再开始下一版。上游改动包括 TypeSafe/System One 接入、充值促销、账号优先级操作、API Key 分组排序、邮件验证码与重置 token 原子处理、公开订单验证限流、Antigravity 错误净化、Grok 身份头、Axios 更新，以及删除 API Key 后的计费结算和 TypeSafe 计费探针。

专项保留 fork 的安全准入和最终请求校验、public/route/channel/account 模型身份、调度/sticky/freshDB、图片能力/privacy、请求体缓存及可重放/释放时序、额度预占和计费、ModelTrace、配置热更新、使用日志及 Excel 导出，以及最新的额度周期持久化锚点、重置按钮展示和分组自助提前周期开关。真实业务语义无法兼容时记录冲突并请求用户决定；已能同时保留的能力直接融合。

# Non-goals

不发布 tag、推送远端、创建 PR 或部署；不扩大到调查后新出现的上游版本。不修未经合并影响的上游原样缺陷或无关 fork 存量问题。不改写 Git 历史、不修改已发布 SQL 的文件名或 checksum、不更改信任、安全及 Hook 配置。最终结果接受与本地合回 main 或保留分支按 Comet 最终交付选择执行。

# Acceptance examples

- A1：Git 基线及两个冻结 tag 可核对；存在顺序独立 no-ff merge，第二父分别精确等于冻结 commit，最终 HEAD 包含两版。
- A2：v0.2.12 完成版本检查、后端/前端门禁、独立只读语义审查及可恢复提交后才合并 v0.2.13；失败、未运行和环境阻塞如实记录。
- A3：完整吸收两版上游发布能力，重点核对 TypeSafe 端点隔离/模型列表/调度/审核/计费，充值优惠和赠送基数，原子邮件校验、公开订单限流、错误净化、API Key 排序及删除后计费。
- A4：fork 核心定制及近期额度周期行为保持有效；复核入口、分支、回退、缓存、生命周期和运行时热更新，不能仅依据文本无冲突或编译通过判断。
- A5：合并造成或上下游交互造成的非平凡回归有可运行失败复现和最小根因修复，检查所有相关调用路径；真实语义冲突未经用户决定不擅自二选一。
- A6：每版后端 default/unit 测试、golangci-lint 和构建，前端 lint:check/typecheck/test:run/build（含 i18n）通过；最终候选由 Runtime 执行并绑定完整计划，新的独立只读 Verifier 覆盖 A1-A8。服务依赖的 integration 检查仅在实际具备隔离环境时执行，缺口明确列出。
- A7：版本依次为 0.2.12.1、0.2.13.1；上游新增 SQL 与 fork ModelTrace/额度周期迁移共存且已发布 SQL 不变；Ent/Wire 生成与 schema/provider 一致，生成稳定；fork 四段式版本和发布构建矩阵保持有效。
- A8：Comet 状态和报告由公开 Runtime 管理；候选、检查与独立验收有真实记录；成果在独立本地分支供用户接受和选择交付，未执行发布/推送/部署；仅验证过的可复用经验写入项目知识。

# Constraints and invariants

使用当前 DSH 会话和原生文件工具；SDK 沿用 vfox 管理的 Go 1.27.1、Node 22.23.1，pnpm 按项目现状；不手改 PATH 或绕过 Hook。开发期跑定向检查，最终完整计划交 Runtime 执行。Go 测试临时目录脚本会删除共享 .test-tmp，后端检查必须串行，避免同目录并发互相删除。

迁移执行器使用完整文件名主键与逐文件 checksum，按文件名排序；241_add_payment_order_bonus_amount.sql、241_add_typesafe_platform.sql 与 241_modeltrace_tasks.sql 前缀相同本身不构成重复迁移，核对 SQL 依赖后保留原名及已发布内容。

# Verification expectations

参考 memory/context/upstream-merge-workflow.md 和前次正式卷宗进行能力审查；历史命令和成功记录仅作为参考。只用本地 fake upstream 及隔离服务做回归测试，不向真实第三方发送测试凭证。限制重 Go 检查并发，记录真实退出码和证据。最终仍等待用户明确接受结果并选择交付方式。
