---
generated_from_state_version: 17
---

# 验证

## 当前结果

- 结果: **已归档**
- 验证情况: **已完成检查，验证结果已确认**
- 目标周期: 1
- 迭代: 4
- 验证器尝试次数: 1
- 完成时间: 2026-10-27T17:39:38.868Z
- 摘要: 通过：独立只读Verifier session-verifier-15-7e70ac8b-8fe9-47a4-bc44-a4d3eb7e6a91 对iteration4/attempt1全部A1–A7逐项核查后均passed，无Blocker。当前候选07162f5954fc932626cf11c5096a624ab9fba6d541ea47fba53636634c1159a0的三项正式Runtime检查均passed/exit0，三份日志SHA实测与任务包一致，inputBefore=inputAfter。A7特别满足唯一窗口不足有效期仍可进入禁确认弹窗、多窗口隐藏并说明等待自然重置、自动开关不影响手动能力。等待用户接受；不代accept-result、归档、commit或push。

## 验收

| 编号 | 结果 | 来源 | 验收项 | 原因 |
| --- | --- | --- | --- | --- |
| A1 | passed | brief.md | A1: 管理员可以在创建或更新订阅分组时配置「额度用尽自动重置」（`auto_quota_advance`），新分组默认关闭（false）。 | 独立核对 backend/migrations/247_add_auto_quota_advance.sql:3,6 两表 NOT NULL DEFAULT false；backend/ent/schema/group.go:333-335 与 user_subscription.go:83-85 一致。backend/internal/handler/admin/group_handler.go:199,280,695,846 接收并传递创建/更新指针布尔；backend/internal/service/admin_group.go:603-605,632 创建默认false并接收显式值，:872-873仅非nil更新。repository/group_repo.go:133,400写入；DTO mappers.go:199,941和types.go:116,822对外暴露。frontend/src/views/admin/GroupsView.vue:779-795,2493-2509提供创建/编辑开关，:5111,5482默认false，:6135-6137,6488-6490提交布尔，且allow关闭时提交false。正式Runtime backend-unit-tests/frontend-build passed/exit0且日志SHA实测吻合；真实PostgreSQL迁移及管理端浏览器交互无法验证。 |
| A2 | passed | brief.md | A2: 当分组未开启 `auto_quota_advance` 或未开启 `allow_quota_advance` 时，用户端不可开启该订阅的自动重置，后端更新接口拒绝变更。 | frontend/src/views/user/SubscriptionsView.vue:350-354只在active且分组allow_quota_advance、auto_quota_advance同时true时展示用户开关；页面测试SubscriptionsView.spec.ts:163-195分别验证任一关闭隐藏。backend/internal/service/subscription_service.go:216-228在同一事务内锁读当前订阅，核对所有者，开启时以数据库关联Group的两个开关做权限校验；repository/user_subscription_repo.go:87-99 FOR UPDATE并WithGroup回源。subscription_auto_advance_test.go:49-160覆盖两个开关拒绝、允许开启、异用户拒绝。关闭已有偏好允许false，符合不可开启的权限边界并避免锁住已有个人偏好；禁止组不会因此取得自动能力。正式Runtime后端/前端检查通过。 |
| A3 | passed | brief.md | A3: 当分组开启了 `allow_quota_advance` 和 `auto_quota_advance` 时，用户可以在订阅卡片切换开启/关闭其订阅的 `auto_quota_advance`，新订阅默认关闭（false）。 | backend/internal/handler/subscription_handler.go:275-306要求显式*bool binding required并使用认证userID；handler/subscription_auto_quota_advance_test.go:12-24拒绝缺失/null/字符串，接受true/false。routes/user.go:143与frontend/src/api/subscriptions.ts:107-115 PATCH路径一致。subscription_service.go:212-238复用事务和行锁，repository/user_subscription_repo.go:172-225整行写入使用锁后当前状态并接收数据库版本；service:533-590提交后清L1、版本墓碑并发布失效。subscription_auto_advance_test.go:292-343覆盖并发新状态不回滚与外层提交后失效。SubscriptionsView.vue:244-266,356-380按订阅分别禁用进行中开关，成功更新个人值、失败保留；页面测试:197-270覆盖保存及双卡并行。新订阅service:827-840零值false，仓储:58持久化且数据库默认false。正式Runtime全量通过；真实数据库竞争和Redis跨实例通信无法验证。 |
| A4 | passed | brief.md | A4: 用户订阅续费、充值延期或天数调整时，该订阅原有的 `auto_quota_advance` 开关状态保持不变。 | backend/internal/service/subscription_service.go:652-685续费使用user+group事务行锁；:725-758有效订阅只改到期/状态/备注，:786-798 expired续费从renewed := *existingSub复制后重置周期，保留AutoQuotaAdvance。ExtendSubscription:1095-1159同事务锁读，正常延长/扣减只写expires_at，过期续费复用复制函数。payment_fulfillment.go:645与redeem_service.go:528调用assignOrExtendSubscription，充值/支付延期进入同一路径。subscription_auto_advance_test.go:424-444验证延期保留true，subscription_term_locking_test.go:282-332核查续费与调整行锁/事务协议。正式Runtime backend-unit-tests包含service包通过；真实支付/兑换及数据库续费联调无法验证。 |
| A5 | passed | brief.md | A5: 当分组和用户订阅均开启了自动重置，且用户请求因唯一达到门槛的窗口超额时，网关原子执行扣减剩余时间并重置该窗口，请求成功放行，并记录审计日志。 | 正常鉴权链独立核实：repository/api_key_repo.go:219-220显式投影两个分组开关，service/api_key_auth_cache.go:71-72定义JSON字段，api_key_auth_cache_impl.go:393-394,502-503双向保存，:17快照v26且:328-329拒绝v25回源。api_key_repo_auto_quota_advance_unit_test.go:11-52使用真实Ent/进程内SQLite四组合验证投影，helper api_key_repo_last_used_unit_test.go:21-35不跳过；api_key_auth_cache_quota_advance_test.go:17-97覆盖冷加载/L2 JSON四组合到自动扣20h清零及v25拒绝。middleware/api_key_auth.go:237-268和api_key_auth_google.go:180-207仅在日/周/月限额预检错误时尝试自动重置，用新订阅重验后进入Next。subscription_quota_advance_receipt.go:100-155在事务行锁后复查当前有效状态、两个分组授权/用户授权、唯一门槛集合、实际usage>=limit和原窗口身份；:156-190同事务更新并commit后失效。subscription_service.go:127-202仅清所选窗口、扣距自然重置剩余时间，:310-319提交后slog结构化审计含user/sub/group/窗口/扣期/新到期。subscription_auto_advance_test.go:246-274,345-422覆盖成功、锁内变化、0.5/1小限额旧快照最多扣一次。正式Runtime backend-unit-tests通过；实际HTTP自动放行、真实PostgreSQL/Redis跨实例竞争与生产审计采集持久化无法验证。 |
| A6 | passed | brief.md | A6: 当存在多个窗口同时超额，或剩余订阅时间不足以扣除本次重置所需时长时，不执行自动重置，请求按正常的 429 限额超限拦截。 | backend/internal/service/subscription_service.go:263-288拒绝多个达到提前重置门槛的可用窗口，:138-155锁后再次核对唯一集合/确认窗口；:198-200严格要求扣减后expires_at>now，等于now也拒绝。subscription_quota_advance_receipt.go:105-109事务回滚，任何校验或Update/receipt写入/commit失败不会返回成功；自动路径不写receipt但使用同一事务。两个middleware分别api_key_auth.go:249-268与api_key_auth_google.go:190-207在自动失败时保留原限额错误并返回429。subscription_auto_advance_test.go:199-244覆盖多窗口与仅剩10h却需扣20h，:345-379覆盖锁内第二窗口变化/自然过期/授权撤销而权益不变；subscription_advance_quota_test.go:193-287覆盖一次性日额度、无效状态及不足保护。正式Runtime后端检查通过；真实HTTP429和PostgreSQL失败回滚联调无法验证。 |
| A7 | passed | brief.md | A7: 手动「提前重置额度」入口与弹窗功能保持正常可用，不受自动重置开关开启与否的破坏。 | 已完整核查Spec:21-23三个边界。frontend/src/utils/subscriptionQuota.ts:80-85按唯一实际合格窗口展示入口，:87-124共享门槛/活跃/正常重置在未来/非一次性日规则，独立于auto开关；SubscriptionsView.vue:268-285唯一窗口展示、多窗口隐藏并显示等待自然重置文案（zh/misc.ts:150，en/misc.ts:153），:383-393直接调用仍校验。唯一周窗口需扣5天但只剩20h时仍可打开弹窗，页面测试SubscriptionsView.spec.ts:55-73真实挂载页面与弹窗核查不足说明/禁确认，:75-103覆盖多窗口隐藏与直接调用防御。SubscriptionQuotaAdvanceDialog.vue:18-48只读展示窗口/正常重置/扣期/预计到期，:59-69取消和确认，:106-110,130-139即使直接传多窗口也不允许确认或发请求；组件测试:36-111覆盖取消不请求、成功、状态变化消息、不足与多窗口禁确认。手动后端receipt:51-66传autoExpected=nil，只校验AllowQuotaAdvance，不受AutoQuotaAdvance影响；handler:230-272强制Idempotency-Key并携带窗口指纹、同事务receipt，原子计算测试subscription_advance_quota_test.go:15-87,137-287覆盖单窗口/原状态保留/安全拒绝。正式Runtime日志包含utils 25项、view 13项、dialog 7项通过；产品浏览器交互未运行。 |

## 检查

| 检查 | 命令 | 工作目录 | 状态 | 退出码 | 耗时 |
| --- | --- | --- | --- | ---: | ---: |
| backend unit tests | test -tags=unit ./... | backend | passed | 0 | 11067 ms |
| frontend unit tests | run test:run | frontend | passed | 0 | 104177 ms |
| frontend build | run build | frontend | passed | 0 | 91097 ms |

### Builder 报告的证据

以下为 Builder 报告，不等同于 Runtime 检查凭据或独立验收结果。

- A7定向工具、页面与弹窗回归: passed — pnpm test:run src/utils/__tests__/subscriptionQuota.spec.ts src/views/user/__tests__/SubscriptionsView.spec.ts src/components/subscription/__tests__/SubscriptionQuotaAdvanceDialog.spec.ts；RED 7 failed（3 utils+4 view），修复后GREEN 3 files/45 tests passed，3.62s。父已读取子会话seq258实际结果核实。
- 前端类型检查: passed — pnpm typecheck（vue-tsc --noEmit）退出0。
- 国际化完整性: passed — pnpm run check:i18n退出0，3 passed；复用中英现有多窗口等待自然重置文案，无新增文案键。
- 变更与格式核对: passed — 父读取当前冻结diff确认唯一生产逻辑变化为共用入口判断及删除旧分支；git diff --check退出0。实现协作者已停止所有写入，未修改后端、模板或弹窗。
- 已知限制: 仓储新增投影检查使用真实Ent和进程内SQLite，事务/缓存并发回归复用内存仓库、测试事务客户端及sqlmock；未运行真实PostgreSQL/Redis跨实例集成或产品浏览器交互测试。

## 阻塞项

_无。_

## 风险与跳过的工作

- [Suggestion] 验证边界：backend/internal/service/api_key_auth_cache_quota_advance_test.go:70-74使用内存订阅仓库/测试事务客户端，backend/internal/repository/user_subscription_lock_test.go:40-58验证sqlmock的FOR UPDATE；真实PostgreSQL迁移和竞争、Redis跨实例失效、HTTP自动放行/429、产品浏览器与审计采集持久化均无法验证。现有通过结论基于完整源码审查和匹配当前候选的正式Runtime检查，不把这些未运行场景描述为已实测。
- 未发现Blocker。未修改普通项目文件、brief、完整Spec、测试或Runtime文件；仅公开Comet claim/context/status/show/outcome命令及系统临时JSON。未启动数据库、Redis、服务、浏览器或后台job。
- 程序说明：claim响应已保存系统临时JSON，但一次提取action误带input中的Builder/历史摘要；未将其中自评或前轮passed用于任何验收结论。全部A1–A7均依据本次独立读取的正式需求、源码、测试和当前Runtime日志判定。
- checksEvidence: {"verifiedAt":"2026-10-27T17:39:10.519Z","candidateId":"07162f5954fc932626cf11c5096a624ab9fba6d541ea47fba53636634c1159a0","checksActionId":"auto-quota-advance-switch:14","checksActionAttempt":1,"inputBefore":"e488d5155a93b046c3f9667476fe01c441f548261928524124588c9aa36d4f8e","inputAfter":"e488d5155a93b046c3f9667476fe01c441f548261928524124588c9aa36d4f8e","candidateInputFingerprintGate":"979568c53234b6772ae400a41719464d2f1c1c0d5a299584464d67a8bb8946ef","completedAt":"2026-10-27T17:31:36.076Z","checks":[{"argvDisplay":["test","-tags=unit","./..."],"cwdRef":"backend","durationMs":11067,"exitCode":0,"id":"backend-unit-tests","logRef":"native/sdk-checks/3855a2eee92641d74f47ec101f840b4e61f0617dcef493f79bb42c5af6a4716b/logs/checks/ee557ad01b9e836f71980c6e2dd8898282d838b9aca3a026c73198d63cacbc76-backend-unit-tests.log","logSha256":"05a9f93e498162b1d349269da8b6ebd6010d2e1cd0fb64242e797c9338e429b2","name":"backend unit tests","status":"passed"},{"argvDisplay":["run","test:run"],"cwdRef":"frontend","durationMs":104177,"exitCode":0,"id":"frontend-unit-tests","logRef":"native/sdk-checks/3855a2eee92641d74f47ec101f840b4e61f0617dcef493f79bb42c5af6a4716b/logs/checks/ee557ad01b9e836f71980c6e2dd8898282d838b9aca3a026c73198d63cacbc76-frontend-unit-tests.log","logSha256":"92301f3c8c0f6b40814e737018bf36aa35759eb4b65402f01e61fc4825f92696","name":"frontend unit tests","status":"passed"},{"argvDisplay":["run","build"],"cwdRef":"frontend","durationMs":91097,"exitCode":0,"id":"frontend-build","logRef":"native/sdk-checks/3855a2eee92641d74f47ec101f840b4e61f0617dcef493f79bb42c5af6a4716b/logs/checks/ee557ad01b9e836f71980c6e2dd8898282d838b9aca3a026c73198d63cacbc76-frontend-build.log","logSha256":"7ccaf332442810dda29641e7fdafd736b6d3d11db7ab2fd50bbfd066eb4a88ce","name":"frontend build","status":"passed"}]}

## 之前的迭代

| 目标周期 | 迭代 | 尝试 | 结果 | 未解决项 | 摘要 | 完成时间 |
| ---: | ---: | ---: | --- | --- | --- | --- |
| 1 | 1 | 1 | fail | A3, A5 | 需整改：独立只读验收A1/A2/A4/A6/A7通过，A3/A5失败。Runtime三项自动检查通过不覆盖发现的并发及缓存授权缺陷。未修改项目实现文件。 | 2026-10-27T16:12:04.255Z |
| 1 | 2 | 1 | fail | A5 | 需整改：独立只读Verifier session 9b98ed85-e23e-4a2c-9e34-5d8f488e8de5对iteration2/attempt1全部A1-A7完成核查；A1/A2/A3/A4/A6/A7 passed，A5 failed。三项当前候选Runtime检查passed且日志摘要匹配，但真实认证链路漏传分组开关使自动重置不可达。只提交原Action真实失败验收，不接受、归档、提交或发布。 | 2026-10-27T16:48:03.394Z |
| 1 | 3 | 1 | fail | A7 | 需整改：全新独立只读Verifier verifier-iteration3-12-ef4718b7-7465-45d3-bcef-4e8b87af151a对iteration3/attempt1全部A1-A7完成验收；A1-A6 passed，A7 failed。当前候选三项正式Runtime检查均passed/exit0且日志SHA吻合；A5认证投影/缓存修复已核实，但完整目标Spec要求的手动入口在不足有效期与多窗口场景仍不符合。请求Runtime按真实失败退回Build，不接受结果、归档、commit或push。 | 2026-10-27T17:10:38.867Z |
| 1 | 4 | 1 | pass | — | 通过：独立只读Verifier session-verifier-15-7e70ac8b-8fe9-47a4-bc44-a4d3eb7e6a91 对iteration4/attempt1全部A1–A7逐项核查后均passed，无Blocker。当前候选07162f5954fc932626cf11c5096a624ab9fba6d541ea47fba53636634c1159a0的三项正式Runtime检查均passed/exit0，三份日志SHA实测与任务包一致，inputBefore=inputAfter。A7特别满足唯一窗口不足有效期仍可进入禁确认弹窗、多窗口隐藏并说明等待自然重置、自动开关不影响手动能力。等待用户接受；不代accept-result、归档、commit或push。 | 2026-10-27T17:39:38.868Z |



## 结论

通过：独立只读Verifier session-verifier-15-7e70ac8b-8fe9-47a4-bc44-a4d3eb7e6a91 对iteration4/attempt1全部A1–A7逐项核查后均passed，无Blocker。当前候选07162f5954fc932626cf11c5096a624ab9fba6d541ea47fba53636634c1159a0的三项正式Runtime检查均passed/exit0，三份日志SHA实测与任务包一致，inputBefore=inputAfter。A7特别满足唯一窗口不足有效期仍可进入禁确认弹窗、多窗口隐藏并说明等待自然重置、自动开关不影响手动能力。等待用户接受；不代accept-result、归档、commit或push。
