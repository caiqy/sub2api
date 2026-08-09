## MODIFIED Requirements

### Requirement: 按正式 release tag 分段集成
维护流程 SHALL 允许将一个最终上游 release 目标拆为具有严格祖先顺序的多个正式 tag 阶段。每个阶段 MUST 完成冲突处理、能力审查和阶段验证后，才能进入下一阶段。

#### Scenario: 顺序合入多个 tag
- **WHEN** 用户选择按 `v0.1.170`、`v0.1.171`、`v0.1.172` 分段集成，且三者形成严格祖先链
- **THEN** 维护流程 MUST 按三个正式 tag 的顺序建立独立 `--no-ff` merge 节点，不得跳过尚未完成验证的前置阶段，也不得合入 `v0.1.172` 之后的 `upstream/main` 提交

#### Scenario: 从已验证但未归档的中间 release 继续扩展
- **WHEN** 一个分段合并 change 已通过中间 release 的最终验证但尚未归档，且用户将目标扩展到后续正式 tag
- **THEN** 维护流程 MUST 保留已完成任务和验证报告作为历史证据，使旧验证结果对新增范围失效，并在追加 merge 前重新运行基线与能力映射门禁

#### Scenario: 某阶段首次出现本地能力回归
- **WHEN** 阶段验证发现阶段 0 已保护的本地能力不再成立
- **THEN** 维护流程 MUST 在当前 release 区间内保留失败证据并完成最小修复，不得继续合入下一 tag

#### Scenario: 最终版本与 merge 拓扑闭合
- **WHEN** `v0.1.170`、`v0.1.171` 和 `v0.1.172` 三个阶段均已通过验证
- **THEN** 维护流程 MUST 将结果版本更新为 `0.1.172.1`，确认三个 tag 都是结果 HEAD 的祖先，且三个 merge 节点的第二父分别精确匹配实施前固定的 peeled SHA

### Requirement: 合并后验证本地关键能力
维护流程 SHALL 在每个分段 merge 后运行所选验证配置要求的自动门禁及该阶段受影响能力的能力级审查，并在最终阶段执行完整本机自动验证和本地能力专项 review。测试通过 MUST NOT 替代能力级审查结论；用户明确选择本机验证配置时，未执行的 Docker-backed integration MUST 作为残余风险保留，不得伪装为通过。

#### Scenario: 分段自动验证通过
- **WHEN** 一个目标 tag 的 merge、冲突处理和兼容修复完成，且用户已选择本机验证配置
- **THEN** 维护流程 MUST 运行根目录 `make test` 与 `make build`、受影响能力聚焦测试、Ent/Wire 两次生成稳定性检查、冲突标记检查和能力映射审查，全部通过后才能进入下一阶段

#### Scenario: 本机 Docker-backed integration 可用
- **WHEN** 本机 Docker/Testcontainers 运行环境可用
- **THEN** 维护流程 MUST 在本机运行 integration，并验证 migration 新库与已有本地记录升级路径；要求的 migration/repository integration test 未实际通过时 MUST 阻塞当前阶段

#### Scenario: Integration 运行环境不可用或目标测试被跳过
- **WHEN** 用户已明确选择仅本机验证，且本机 Docker/Testcontainers 不可用或目标 integration 无法执行
- **THEN** 维护流程 MUST 记录未执行命令、环境原因和受影响契约，将其列入阶段及最终报告的残余风险，并 MAY 在其他本机门禁通过后继续下一 release tag；维护流程 MUST NOT 使用远程服务器补跑，也 MUST NOT 将 integration 记录为通过

#### Scenario: 最终自动验证通过
- **WHEN** 最终目标 tag 合并完成且无冲突残留
- **THEN** 维护流程 MUST 运行后端默认与 unit 测试、后端 lint、前端 ESLint、前端单测、前端类型检查和构建验证

#### Scenario: 本地关键能力专项 review
- **WHEN** 自动验证完成
- **THEN** 维护流程 MUST 逐项复核 scheduler、各平台 sticky、fallback/WaitPlan、DB recheck、privacy、image capability、异步图片任务与对象存储、图片输入计费、上游计费倍率、会话绑定与 step-up、runtime setting 热更新、网关透传字段、请求体重放与清理、用户资源控制、分组复制、用户批量限额、订阅额度周期提前重置、前端本地功能、版本依赖、生成代码和 migrations，并记录每项证据

#### Scenario: 新增上游能力与本地定制交互专项审查
- **WHEN** 目标 release 区间引入触及调度、路由、网关转发、安全审计或 repository 更新语义的新上游能力
- **THEN** 维护流程 MUST 审查该能力入口调用链与本地 advanced/layered scheduler、Grok/platform sticky、prompt cache reuse、body replay、统一 prompt audit 和 subscription quota cycle reset 的交互，并记录不被绕过或改写的证据

#### Scenario: v0.1.170 利润控制与本地调度计费共存
- **WHEN** `v0.1.170` 的分组利润控制、账号倍率自动同步和槽位后二次复核进入本地调度与计费调用链
- **THEN** 合并结果 MUST 保持本地 advanced/layered scheduler、sticky、fallback/WaitPlan、DB recheck 和 usage billing 语义，并验证不合格账号不会提前绑定粘性、释放槽位后可以重新选号、同一请求定价时刻稳定且默认关闭时行为不变

#### Scenario: v0.1.171 Codex 身份与过载策略共存
- **WHEN** `v0.1.171` 统一 Codex 出站身份、动态版本来源，并把流内过载调整为同账号有界重试后切号
- **THEN** 所有 HTTP、透传、WebSocket、探针、模型列表与 alpha-search 路径 MUST 使用一致身份来源，同时保持本地请求体 spooling/释放、最终 outbound model、账号 failover、sticky 和错误响应契约

#### Scenario: 验证码与认证设置兼容
- **WHEN** `v0.1.171` 新增腾讯天御与阿里云验证码并合并后台人机验证设置
- **THEN** 注册、登录、找回密码、OAuth 启动和 passkey 登录 MUST 按所选互斥服务商执行 fail-closed 拦截，既有 Turnstile 配置与本地 settings 热更新、CSP、自定义菜单及前端认证流程 MUST 保持兼容

#### Scenario: v0.1.172 OAuth pending 安全修复与验证码兼容
- **WHEN** `v0.1.172` 修复 pending OAuth 非终态 session 可绑定攻击者身份的账号接管漏洞，并扩展腾讯验证码 region/ticket/CSP
- **THEN** 非终态登录 session MUST NOT 绑定身份、修改目标用户或消费 session；终态登录与已登录用户主动绑定 MUST 保持可用，且 Turnstile、腾讯、阿里云三种互斥 provider 在全部认证入口继续 fail-closed

#### Scenario: v0.1.172 响应模型审计与本地网关共存
- **WHEN** 网关记录上游响应声明模型并恢复 pre-output capacity failover
- **THEN** HTTP、Responses、Anthropic/Gemini/Grok、WebSocket 和失败用量路径 MUST 保存请求模型、实际 outbound 模型与上游响应模型的可区分证据，并保持本地 body replay/release、sticky、最终账号、错误改写和单次计费契约

#### Scenario: 上游 midnight 日额度修复与本地实际时刻锚点冲突
- **WHEN** `v0.1.172` 把订阅日额度改为配置时区 0 点刷新，而本地产品决策要求实际操作时刻锚点
- **THEN** 新购、用户手动重置和管理员手动重置 MUST 以实际操作时刻作为窗口起点，后续自动日窗口 MUST 按该锚点每 24 小时推进，不得被静默改为 midnight；既有一日卡、事务锁、receipt、outbox 和 cache invalidation 语义 MUST 保持

#### Scenario: UsageLog 194/195 migration 与本地持久化共存
- **WHEN** `v0.1.172` 为 upstream response model audit 新增 UsageLog 字段、`194` schema migration 和 `195` 非事务索引 migration
- **THEN** schema、Ent、单条/批量/best-effort usage insert、查询筛选和管理端展示 MUST 一致，194/195 MUST 与既有双方 191/192 和 193 按完整文件名共存；真实 PostgreSQL integration 未执行时 MUST 保持 `unverified`

#### Scenario: 最终审查发现 Images 审计入口重复与关闭态大 payload 构造
- **WHEN** 生产依赖图中的 unified security-audit coordinator 已包含 legacy content moderation，而 OpenAI Images handler 仍直接调用 legacy moderation，或仅按依赖指针决定是否序列化 prompt/image payload
- **THEN** Images 请求 MUST 只经统一审计入口执行 legacy/prompt 审核，legacy moderation 对每个请求最多执行一次；审核 payload MUST 在线程安全 provider 中最多冻结一次，并只在有效 prompt audit 或完成运行态与范围判定后的 legacy moderation 确实需要时求值，同时保持在请求文本释放前可用

#### Scenario: 同号不同名 migration 兼容
- **WHEN** 上游 `191_passkey_credentials.sql` 与本地 `191_subscription_quota_advance_receipts.sql` 使用相同数字前缀但文件名和用途不同
- **THEN** 维护流程 MUST 保留双方完整文件名与既有校验和，并在可用的 integration 环境中验证迁移执行器在空库和已应用本地 migration 的升级库上正确执行全部文件，不得仅因数字前缀重复而重命名历史 migration

#### Scenario: 新增同号 192 migration 兼容
- **WHEN** 上游新增 `192_group_profit_control.sql`，而本地主线已发布 `192_subscription_cache_invalidation_outbox.sql`
- **THEN** 维护流程 MUST 保留两个 `192_*` 文件和上游后续 `193_group_profit_control_auth_cache_invalidation.sql` 的完整文件名与既有校验和，并在可用的 integration 环境中验证空库、已应用本地 192 的升级库、排序、幂等和 checksum；不得重命名任何已发布 migration

#### Scenario: 从未合入默认分支的已验证分支发布测试版本
- **WHEN** 用户明确授权从已通过 full verify 的隔离分支发布本地四段式 tag，并将 CI 镜像更新到指定测试服务器验收
- **THEN** Release workflow MUST 正常产出精确版本二进制、checksum 与镜像，但 MUST 在同步默认分支 VERSION 前验证 tag commit 是默认分支 HEAD 的祖先；不满足时 MUST 跳过同步，禁止产生 VERSION-only 主线提交
- **AND** 测试服务器 MUST 只拉取 CI 发布的镜像，更新前 MUST 记录旧 image digest 并备份测试数据库，更新后 MUST 验证 health、revision/migration 与关键页面；不得在服务器构建 Sub2API 镜像或把失败烟测记为通过
