# Outcome

遵循历史合并惯例，将调查时最新 upstream 正式发布 v0.2.20 完整融合进 fork，保留上游能力、本地定制和可追溯历史，经检查与独立验收后供用户接受。

# Scope

从干净 main 基线 a654a78d79ede218b106fb255f9703837a17a829（VERSION 0.2.19.7，已含 v0.2.19）创建临时分支 comet/merge-upstream-v0-2-20。唯一上游目标为 v0.2.20 peeled commit dab829a4e9b7d1cc307189aea8b0319eb29e29d0（tag 对象 ec41309811cd30af310a0e4b9119b976ce65c9be），采用 no-ff merge。GitHub 最新正式 Release 已核实为 v0.2.20，发布时间 2026-11-07T06:31:24Z。该 tag 后 upstream/main 只有版本同步提交 6b4b37db43443b5e88e424fe3c69602cc3810dfb，不纳入本轮。

完整吸收此 tag 的管理 MCP/OAuth、管理员密钥权限、注册邮箱黑名单、图片尺寸路由/GPT Image 2.6、协议转换和流式回复、请求体及后台任务资源约束、鉴权/限流/计费与前端依赖修复。对本地改动交集逐项审查；除用户明确要求移除的 fork OpenAI layered 调度及健康探测子系统外，可共存的行为保留双方，真实业务语义冲突再请用户决定。移除截图中的 weighted/layered 模式选择及九项惩罚/探测参数、layered/probe 实现、账号探测专属配置和管理桥接，收敛至上游 default/weighted 调度路径；保留该路径上的其他 fork 资格、安全和计费约束。一个集成结果与同一回归边界，不拆 Supervisor 子任务。

## 目录结构

### 修改

- backend/ — 融合配置、鉴权、路由、handler、service、repository、协议库、Wire、依赖及测试，保留本地调度、计费、模型策略、透传、加密恢复和请求详情能力。
- frontend/ — 融合 MCP、账号、设置、密钥、用量、路由及多语言和依赖，保留本地界面功能与配置字段。
- backend/migrations/ — 核对新增 249_mcp_oauth.sql 与 fork 已有 249_group_exclusive_rate_models.sql、250_user_cyber_overflow_exempt.sql 的顺序、校验和和升级兼容性，必要时调整新增迁移，禁止改写已执行迁移。
- .github/、deploy/ — 融合相关 CI、依赖安全和运行配置，保留 fork 四段版本与单架构发布约定。
- README*.md、assets/、docs/ — 吸收本 tag 文档与资源及本轮必要的合并经验和流程产物。

### 明确不建

- 新发布版本、tag、独立服务或替代框架 — 本次仅集成已发布上游与解决融合回归。

# Non-goals

不吸收 v0.2.20 之后的 upstream 提交，不发布派生版本，不推送、部署或操作生产数据。保留本地 VERSION 0.2.19.7，待单独发布请求按已合入上游基准确定四段版本。验收后等待用户接受结果和选择是否本地合回 main；本次执行阶段在临时分支完成。

# Acceptance examples

1. A1：合并节点第一父精确为 a654a78d79ede218b106fb255f9703837a17a829，第二父精确为 dab829a4e9b7d1cc307189aea8b0319eb29e29d0；目标为候选祖先，不纳入 tag 后 upstream 提交。
2. A2：上游 MCP/OAuth、管理员密钥权限和邮箱黑名单功能完整融合；本地鉴权投影、账号限制、用户策略、菜单与密钥作用域不被覆盖，安全边界的相关测试通过。
3. A3：移除 fork layered 调度、健康探测及模式工厂，收敛上游 default/weighted 调度；旧模式配置不再启用旧实现，无后台探测线程。保留上游独立高级调度设置及普通候选、session/previous-response sticky、fallback/WaitPlan、DB 复核、privacy、模型/生图、guardian/Grok、图片尺寸分流和本地 Codex 客户端资格，不整文件覆盖独立 fork 规则；其他 runtime 热更新照常工作。
4. A4：保留上游协议、GPT Image 2.6、工具媒体及流式修复；fork 请求体缓存/handle/释放、生图历史、加密内容恢复、透传字段和错误分类保持正确，资源限制不引入请求体丢失、超时契约回归或重试失效。
5. A5：计费、专属倍率、利润门、订阅配额预支、用量明细与导出保留 fork 行为，同时吸收上游计费与每 turn 首 token 修复；相关回归通过。
6. A6：迁移保留已执行文件及校验和、兼容 fork 249/250 历史与新增 MCP 表；旧 layered/probe 配置键及账号 Extra 可留存但不再生效，不破坏历史迁移、不批量清理其他来源临时封禁；Wire 与依赖一致；VERSION 与四段式单架构发布约定保留，不触发发布。
7. A7：前端撤除截图模式/九项参数及探测专属账号入口，相关 settings/DTO/类型/表单/保存链路同步收回；其余 MCP/设置/账号/密钥/用量与 fork 生图工作台等功能融合且配置可往返，无重复控件；前端测试、typecheck、build 通过，对改变的关键页面做本地预览及浏览器实测。
8. A8：完成冲突和本地关键能力的专项语义 review、后端 go test ./...、相关构建与前端检查，并由新的独立只读 Verifier 覆盖全部验收项。失败、超时和未执行如实记录，不能视为通过；没有用户接受前不归档或交付主线。

# Constraints and invariants

沿用 memory/context/upstream-merge-workflow.md 的合并流程作为实现参考。SDK 由 vfox 管理。测试临时 PostgreSQL 必须结束后停止并确认端口关闭，不停止既有用户实例；确实不可用时记录未运行。实际冲突依据双方源码与历史处理，不机械选择 ours/theirs。新增复杂融合修复留下可运行回归测试；全量测试相反断言追溯来源并按业务语义处理，不为通过门禁删断言。

# Verification expectations

后端执行 go test ./... 并按仓库既有方式补齐 unit/integration 标签测试和构建，前端执行 pnpm test:run、pnpm typecheck、pnpm build，发布工具运行现有脚本测试。真实数据库可用时验证新增迁移与鉴权/MCP 相关集成测试；不能启动时明确证据缺口。独立 Verifier 复核候选和 Runtime 证据，失败回 Build 修复。
