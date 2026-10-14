# Outcome

从干净 main 基线 5fa3d16a5c7453df25096e6c0faa9c3b47ff526a（已包含 upstream v0.2.15，fork VERSION 0.2.15.5）开始，沿用历史 change 惯例，在独立分支融合调查时最新稳定上游 tag v0.2.16，保留 fork 定制，完成真实检查及新的独立只读验收。

# Scope

单个 Native change merge-upstream-tags-20261013；branch 隔离，分支 comet/merge-upstream-tags-20261013，目标 main。2026-10-13 已获取 upstream 并核对远端全部三段式稳定 tag；唯一未被基线包含的是 v0.2.16，tag 对象 c3cb8588ce6a5aef4dc29fa4f71ac91f3c0bdf3d，peeled commit 01842576b3d5c1e2545b5a99ad555d36e28ed306。采用独立 no-ff merge，第二父精确对应冻结 commit；后续新增 tag 与 upstream/main 的 tag 后提交不自动纳入。

完整融合 v0.2.15..v0.2.16 的 213 个变动文件（98 个与 fork 两侧改动重叠），包括 Gemini 显式上下文缓存及存储计费、并发租约/失败回滚/额度预留、Vertex API Key/服务账号模型同步及工具结果图片、国际 Coding 额度、模型级临时不可调度规则、Antigravity 图片模型限流隔离、Claude Haiku 5.5、GPT-6.1 Sol 500k 上下文与订阅长上下文附加费取消、Responses 结构化输入安全 Chat 回退、Codex 图片桥平台边界、分组多语言及前端运维/倒计时/IME/异步交互修复。

专项保留 fork 最终模型身份和请求校验、scheduler/sticky/freshDB、privacy/图片能力、请求体 replay/cache/handle/释放、额度预占与 mandatory billing、ModelTrace、用量详情/样式化 Excel、官方输出 TPS、Codex 点数整数显示、GPT Image 工作台及 JSON keepalive/Base64 本地解码、额度周期锚点/自助开关、配置热更新、支付安全、随机管理员凭据与 Unicode 缺省、部署资源和四段式单 linux/amd64 发布规则。

历史参考为 2026-10-07、2026-10-09 Comet 卷宗及 memory/context/upstream-merge-workflow.md，仅供实施和验收惯例，不将历史通过作为本次证据。完整目标规格位于 specs/upstream-tag-fusion-20261013/spec.md。融合共享核心区域，串行单 change，不拆 Supervisor。

## Directory structure

### Created

新增上游 Gemini 显式缓存、分组多语言等 domain/service/repository/handler、前端组件及配套回归；新增四个上游 SQL 迁移。新增本 change brief 和 specs/upstream-tag-fusion-20261013/spec.md；状态和报告由 Runtime 创建。

### Modified

融合上游涉及的 backend/、frontend/、deploy/config.example.yaml 和 Makefile；重新生成受 schema/provider 影响的 Ent/Wire；更新 backend/cmd/server/VERSION；必要的本次融合回归测试与最终项目经验文档。

### Deleted

不计划删除本地功能或已有 SQL；仅在上游确实删除且无需保留 fork 行为时同步对应删除，本次上游差异无文件删除。

### Not created

不创建 Supervisor children、额外服务/服务器、发布 tag、PR、独立发布流程、新的非必要依赖或无关重构。

# Non-goals

不推送远端、不创建 PR、不发布 tag、不触发 Release 或部署；不改写历史、已有 SQL 文件名或内容，不绕过 Hook，不修改信任配置。不扩展无关存量缺陷、UI 重设计或其他工作区集成。最终接受验收结果及分支收尾按用户明确选择执行。

# Acceptance examples

- A1：基线、冻结 tag/commit 与稳定版本清单可核对；v0.2.16 形成独立 no-ff merge，第二父精确为 01842576b3d5c1e2545b5a99ad555d36e28ed306，最终候选包含该版且无遗漏稳定版本。
- A2：Gemini 显式缓存的创建/读取/更新/删除、过期和账号失效处理、串行修改/失败回滚及存储计费完整融合；新计费/预留与 fork API Key/订阅/用户平台额度、结算及请求体生命周期同时成立，Vertex、AQ. Key、模型同步和工具结果图片保留上游修复。
- A3：模型级临时不可调度及上游模型传递、Antigravity 图片限流隔离、Haiku 5.5、Sol 500k/订阅计费、Responses 安全 Chat 回退和 OpenAI 专用图片桥正确融合，保留 fork 最终模型校验、协议目录、sticky/freshDB、privacy、图片能力与重放/释放等实际入口语义。
- A4：分组多语言、账号表单/用量、监控平台与自动刷新、运维账号名/本地时区、支付倒计时、IME 和异步表单修复完整融合；fork 图片工作台近期修复、官方 TPS、点数整数显示、用量详情/Excel、额度周期及 ModelTrace 等前端定制保留，冲突断言覆盖双方有效行为。
- A5：fork 核心调度、预占/mandatory billing、配置热更新、支付安全、安装凭据、部署及发布规则继续成立；按入口/条件/回退/生命周期专项复核。非平凡融合回归留下可运行复现并修共享根因；真实业务语义不能共存时先取得用户决定。
- A6：最终候选后端 default/unit 测试、golangci-lint、embed 测试/构建与前端 lint:check/typecheck/test:run/build（含 i18n）由 Runtime 实际执行和绑定，新的独立只读 Verifier 覆盖 A1-A8。相关隔离 PostgreSQL/Redis 和浏览器条件具备时进行实测，否则明确未执行边界，不把 skip/超时/历史成功当 pass。
- A7：融合 VERSION 为 0.2.16.1；已有 SQL 全名/内容/checksum 保持，新增 242_group_i18n.sql、243_gemini_cached_contents.sql、244_gemini_cached_contents_channel_usage.sql、245_gemini_cached_contents_mutation_lease.sql 与 fork 同数字前缀迁移安全共存；Ent/Wire 官方生成与输入一致且稳定，Go 1.27.2 和 fork 安全依赖/发布矩阵保留，不触发发布。
- A8：正式 brief/spec 由 Agent 编辑，状态/检查/报告/事务由公开 Runtime 管理；候选与验收可追溯，完整独立验收后等待用户接受和选择交付；无未经授权推送/发版/部署，真正结束时完成项目经验与学习检查记录。

# Constraints and invariants

冲突优先共存上游变更与本地定制，不机械 ours/theirs。新增迁移按完整文件名排序并以 filename 主键识别；相同数字前缀本身不是重命名已有迁移的理由，需要核对表/字段依赖与升级实际结果。融合版本 0.2.16.1 仅本地版本，本次无发版动作。

开发期只做定向检查，完整计划由 Runtime 冻结候选后运行。后端沿用 Windows scripts/test.ps1，重检查串行，避免共享 .test-tmp 被并发清理。当前 shell Go 1.27.1，backend/go.mod 要求 1.27.2；Build/Verify 使用 vfox 管理的相应 SDK，不降低项目要求或影响用户其他 SDK。当前 Node 22.23.1、pnpm 9.15.9。

检查使用本地 fake upstream 与隔离测试服务，不发送真实第三方凭据。临时 PostgreSQL 测试结束仅停止本次实例并确认测试端口关闭，不停止用户原有数据库。服务不可用时如实记录未运行。

# Verification expectations

文本自动合并和编译不足以证明语义保留；重点复核缓存新增入口是否覆盖 fork 额度/请求体守卫，模型级规则是否以最终上游模型生效，以及 Responses 回退是否守住重试安全与既有协议转换。候选绑定记录真实退出码和覆盖范围；最终独立验收通过后等待用户接受与收尾选择。
