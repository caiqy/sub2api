# Outcome

从 main 基线 1171052e7f5b3b607a49b9fdc940ef2d084fc13d（已包含 upstream v0.2.13，fork VERSION 0.2.13.1）开始，使用 Comet Native 在独立分支 comet/merge-upstream-tags-20261007 逐版本融合至本次调查的最新稳定 tag v0.2.14，保留 fork 定制，完成真实检查与独立验收。

# Scope

单个 Native change，branch 隔离，目标分支 main。2026-10-07 从 upstream 获取并核对正式 tag：v0.2.14 的 tag 对象为 1400a7b482974d98db5b284a8b2afbe3eaf9aaef，peeled commit 为 0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d。本次只有这一版未合入；调查后出现的新 tag 不自动扩展范围。完整目标规格位于 specs/upstream-tag-fusion-20261007/spec.md。

参考 2026-10-06 历史 Comet 卷宗和 memory/context/upstream-merge-workflow.md 的实施与验证方式，不将历史成功当作本次证据。版本共享完整应用和安全链路，使用单个 change 串行融合，不拆 Supervisor 子任务。

采用独立 no-ff merge，第二父精确对应冻结 commit；融合全部 25 个上游变动文件，重点覆盖 EasyPay 回调签名复用防护、支付 return_url 清理、新安装随机管理员凭据及登录兼容校验、远端 Codex catalog 的 API Key discovery、Vue/source-map-js 依赖修复、xlsx 审计说明及部署文档。既有部署不重置管理员账户。

专项保留 fork 的调度/sticky/freshDB、privacy/图片能力、最终模型身份与请求校验、请求体 replay/cache/释放、额度预占及 mandatory billing、ModelTrace、日志和 Excel 导出、配置热更新、额度周期持久化锚点/按钮/分组开关，以及支付结算、部署资源、安全限制、四段式版本和单架构发布矩阵。真实业务语义无法兼容时请求用户决定，其余直接融合两边能力。

# Non-goals

不推送远端、不创建 PR、不发布 tag、不触发 Release 或部署；不改写历史、不修改既有 SQL 名称或 checksum、不更改信任与 Hook 配置。不扩大为与融合无关的存量缺陷修复或 UI 重设计。最终接受结果及保留分支/本地合回 main 按 Comet 的用户选择执行。

# Acceptance examples

- A1：main 基线和冻结 v0.2.14 的 tag/commit 可核对；形成独立 no-ff merge，第二父精确为 0363b8cdba8cec3e2ba4b2dbd49c4481143fa55d，最终 HEAD 包含该 tag，未跳过尚未合入的稳定版本。
- A2：EasyPay 防签名复用与 return_url 清理完整融合；合法回调和 QueryOrder reconcile 路径保留，未知回调字段被拒绝，充值/赠送/返利/退款等既有结算语义不被削弱，有针对性安全回归验证。
- A3：Web/CLI/环境变量新安装使用随机缺省管理员凭据，并校验登录邮箱和 bcrypt 密码长度；现有管理员/用户部署不被重新初始化，部署示例和说明一致。
- A4：远端 Codex catalog 各配置生成入口启用 API Key discovery，本地文件 catalog 保留原语义；Vue/source-map-js 更新与 fork Excel writer/vendor 及依赖锁文件兼容，审计说明与实际使用一致。
- A5：fork 核心定制、近期额度周期行为、部署资源和发布规则继续成立；按入口/条件/回退/生命周期复核，非平凡融合回归有最小可运行复现与共享根因修复，不以删断言替代业务决定。
- A6：最终候选的后端 default/unit 测试、golangci-lint、构建和前端 lint:check/typecheck/test:run/build（含 i18n）由 Runtime 实际执行并绑定；新独立只读 Verifier 覆盖 A1-A8。真实服务集成或浏览器环境具备时执行相关验证，否则明确未运行及边界，不将 skip 当 pass。
- A7：融合 VERSION 为 0.2.14.1；既有 SQL 名称及内容保持，Ent/Wire 与 schema/provider 一致；存在生成相关变动时官方生成稳定。fork 四段式版本解析和现有发布构建矩阵有效，不触发发布。
- A8：Comet 状态、检查和验收报告由公开 Runtime 管理，真实候选和独立验收可追溯；成果保留在独立本地分支供用户接受及选择交付，无未经授权的推送/发版/部署；任务结束记录合格项目经验及学习检查。

# Constraints and invariants

SDK 通过 vfox 管理，沿用已安装的适用版本，不手改 PATH、不绕过 Hook。后端 Windows 测试使用既有脚本；重型检查串行，避免共享 .test-tmp 被并发清理。开发期定向验证，最终完整计划交 Runtime 冻结候选后运行。

测试使用本地 fake upstream 和隔离服务，不使用真实第三方测试凭据。若启动临时 PostgreSQL 实例，结束必须仅停止本次实例并确认端口关闭；不得停止原有数据库。本次上游没有 SQL/schema/provider 变动，生成检查按实际融合结果选择。

# Verification expectations

重点对安全更新进行端到端调用路径核查及回归。对未变动核心定制核对相对基线差异并结合现有回归覆盖；文本无冲突和编译通过不足以证明语义保留。记录真实退出码、候选绑定和未覆盖范围，独立验收后等待用户接受及交付选择。
