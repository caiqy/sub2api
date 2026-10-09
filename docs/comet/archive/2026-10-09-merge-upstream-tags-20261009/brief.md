# Outcome

从 main 基线 ce10e1e777653310629d651e4b867e3b713781bb（已包含 upstream v0.2.14，fork VERSION 0.2.14.2）开始，按历史 change 惯例在独立分支逐版本融合至 2026-10-09 调查的最新稳定 tag v0.2.15，保留 fork 定制，完成真实检查与独立验收。

# Scope

单个 Native change merge-upstream-tags-20261009，branch 隔离，分支 comet/merge-upstream-tags-20261009，目标 main。已获取并逐一检查 upstream 全部三段式稳定 tag，唯一尚未成为基线祖先的是 v0.2.15：tag 对象 86a80c13dcba86f52f9ca815b5a471cecc236227，peeled commit f2669c8cf62555cd92389b3f55920e9e6e7c6ff2。调查后新增 tag 不自动扩展范围。采用独立 no-ff merge，第二父精确对应冻结 commit，保留双方历史。

历史参考为 2026-10-06、2026-10-07 Comet 卷宗及 memory/context/upstream-merge-workflow.md，仅用于实施和验收惯例，不复用历史通过结论作为本次证据。完整目标规格位于 specs/upstream-tag-fusion-20261009/spec.md。版本共享核心区域，单 change 串行处理，不拆 Supervisor。另一 active change 位于独立 worktree，不修改其目录、分支或状态。

完整融合 v0.2.14..v0.2.15 的 278 个变动文件，重点为 provider profile 与平台清单、Cline/Command Code、应用层平台校验及新增迁移、三入口上游协议分流、模型目录隔离及同步、Claude 计费/钱包冷却、OpenCode/Grok 修复、Responses Lite web_search 历史、Anthropic thinking/tool/cache TTL、Chat/Responses 转换、WS 图片与分组定价、请求样本健康评分、输出 TPS 和前端异步生命周期修复。冲突保留上游和 fork 两侧能力；实际产品语义无法共存时先提出明确选项。

专项保留最终模型身份和请求校验、调度/sticky/freshDB、privacy/图片能力、请求体 replay/cache/handle/释放、额度预占与 mandatory billing、ModelTrace、日志和样式化 Excel 导出、配置热更新、额度周期持久化锚点/按钮/分组开关、支付安全与结算、新安装管理员随机凭据及 Unicode 缺省、远端 Codex catalog discovery、Codex 点数去除小数末尾多余零、部署资源及四段式单架构发布规则。

# Non-goals

不推送远端、不创建 PR、不发布 tag、不触发 Release 或部署；不改写历史、已有 SQL 文件名或内容，不绕过 Hook，不修改信任配置。不扩展为无关存量缺陷修复、UI 重设计或其他工作区集成。最终接受验收结果及工作区收尾按用户明确选择执行。

# Acceptance examples

- A1：基线、冻结 tag/commit 和逐版本清单可核对；v0.2.15 形成独立 no-ff merge，第二父精确为 f2669c8cf62555cd92389b3f55920e9e6e7c6ff2，最终 HEAD 包含该版且无遗漏中间稳定版本。
- A2：provider profile/平台清单及 Cline、Command Code 完整融合；账号/分组/配额/设置/模型候选及同步能力使用一致平台边界；新增 SQL 移除平台 CHECK 后接口、repository 和 Ent 仍拒绝未登记或不合法具体平台，新平台可用且既有平台行为保留。
- A3：三入口协议分流与模型目录隔离正确，Claude 计费/钱包冷却、OpenCode/Grok、web_search 历史、Anthropic thinking/tool/cache TTL、Chat/Responses 转换、WS 图片与分组定价等上游修复与 fork 请求模型、调度、请求体及结算语义同时成立，有相关实际调用路径回归证据。
- A4：前端平台表单、筛选、输出 TPS、请求样本健康评分和异步生命周期修复完整融合；fork 用量详情/Excel/额度周期/平台额度/设置及 Codex 点数显示保留，冲突测试覆盖两侧有效行为，不随意删除断言换取通过。
- A5：fork 核心定制、近期支付安全、首装随机凭据与 Unicode 缺省、部署资源和发布规则继续成立；按入口/条件/回退/生命周期专项复核。非平凡融合回归留下可运行失败复现并修共享根因；真正语义冲突不未经确认选择一侧。
- A6：最终候选的后端 default/unit 测试、golangci-lint、嵌入前端测试/构建和前端 lint:check/typecheck/test:run/build（含 i18n）由 Runtime 实际执行并绑定；新的独立只读 Verifier 覆盖 A1-A8。隔离 PostgreSQL/Redis 和浏览器条件具备时进行相关实测，否则明确未运行与覆盖边界，不把 skip、超时或历史成功当 pass。
- A7：融合 VERSION 为 0.2.15.1；保留已有 SQL 名称及内容/checksum，新增 242 迁移与 fork 迁移安全共存；按官方命令验证 Ent/Wire 与输入一致和生成稳定。Go 1.27.2 及 x/net 等安全依赖、CI/Docker/构建要求一致，SDK 通过 vfox 管理；四段式版本解析和既有单 linux/amd64 发布矩阵继续有效，不触发发布。
- A8：正式 brief/spec 由 Agent 编辑，状态、检查和验收报告由公开 Runtime 管理；本地分支成果、候选及独立验收可追溯，完成后等待用户接受并选择交付；无未经授权推送/发版/部署，结束记录已验证可复用项目经验与学习检查。

# Constraints and invariants

开发期定向检查；最终完整计划由 Runtime 冻结候选后运行。后端 Windows 测试使用既有 scripts/test.ps1，重检查串行，避免共享 .test-tmp 被并发清理。当前 Go 1.27.1、Node 22.23.1、pnpm 9.15.9；Build 中按 vfox 安装/执行 Go 1.27.2，核对 lint 工具兼容，不手改 PATH 或全局切换来影响其他工作。

测试使用本地 fake upstream 和隔离服务，不发送真实第三方凭据。若启动临时 PostgreSQL，结束只停止本次测试实例并确认端口关闭，不停止用户原有数据库。已有 SQL 不改名或改内容；新增 SQL 的完整文件名、约束校验和迁移顺序单独检查。

# Verification expectations

文本自动合并和编译通过不足以证明语义保留，重点复核新 profile/路由与 fork 最终模型、请求体缓存和计费的交互，平台枚举迁移后的全部校验入口，以及输出 TPS/健康评分与既有表格定制。检查记录真实退出码、候选绑定和未覆盖范围，独立验收通过后等待用户接受及交付选择。
