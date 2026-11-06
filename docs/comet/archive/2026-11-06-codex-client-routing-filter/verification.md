---
generated_from_state_version: 14
---

# 验证

## 当前结果

- 结果: **已归档**
- 验证情况: **已完成检查，验证结果已确认**
- 目标周期: 1
- 迭代: 3
- 验证器尝试次数: 1
- 完成时间: 2026-11-06T22:32:02.658Z
- 摘要: 通过：当前候选A1-A8全部通过。两个真实token-count入口已在原始body转换前准备入站判定，选号阶段排除双开不合格账号，6个handler场景证明全排除无403/无上游、混合仅B、子关保留旧计数行为；4个service场景证明新增防御仅双开且不污染账号健康。其余调度/粘性/previous/shadow/DB/抢槽/WS、detector复用、UI及配置保留已独立重新核对。四项当前候选正式收据及日志哈希匹配，全部通过；无额外检查请求。等待用户决定，不接受或归档。

## 验收

| 编号 | 结果 | 来源 | 验收项 | 原因 |
| --- | --- | --- | --- | --- |
| A1 | passed | brief.md | A1：新字段缺失、false 或非布尔真值时保持原行为；父开关关闭、非 OpenAI OAuth 时子开关无效；只有双开才参与路由筛选。 | 独立核对 backend/internal/service/account.go:2493-2518：父函数nil安全且限定OpenAI OAuth，父与子均使用(bool)断言和ok&&enabled；缺失、false、字符串及数字不会启用。backend/internal/service/openai_codex_routing_test.go:24-35遍历平台/类型/父/子组合。当前候选正式backend收据passed。 |
| A2 | passed | brief.md | A2：账号 A 双开、账号 B 允许该请求，非合格客户端请求到达同一授权池时，跳过 A 并选择 B，A 不发上游、不占用最终并发槽、不成为新的粘性目标；子开关关闭后恢复原选后拦截。 | backend/internal/service/openai_codex_routing.go:40-53将双开不匹配作为候选资格；默认共享兼容openai_account_scheduler.go:1888-1894、分层兼容、legacy openai_gateway_scheduling.go:417-424均接入。openai_profit_control.go:358-362推迟新粘性绑定，handler/openai_gateway_handler.go:2655-2764终检成功才绑定，被否决三条抢槽路径释放槽位并回候选排除重选，不进入上游切换计数分支。service/openai_codex_routing_test.go:118-178三调度混合/粘性/排除/关闭兼容验证；handler/openai_codex_routing_test.go:85-125新增两计数入口各3场景，混合实际upstream.accountIDs仅[2]，关闭子仅[1]，全过滤为空且非403。未将service防御性403当作选号筛选证据。 |
| A3 | passed | brief.md | A3：官方、黑白名单、App Server 全局/账号放行、版本上下界、不可解析版本、指纹缺失/命中和 ForceCodexCLI 等代表场景，筛选结果与现有 detector 一致；使用入站头和 body 指纹，不被出站身份改写影响。 | backend/internal/service/openai_codex_routing.go:15-24使用原入站c和body调用同一个detector，保存仅两种账号AppServer策略的判定结果，不保存body/header。openai_gateway_service.go:1188-1208在双开转发时复用快照，子关闭保持旧detector路径；openai_client_restriction_detector.go:90-171顺序为父开关、ForceCLI、黑名单、官方/白名单/AppServer、版本、指纹，识别标准无diff。service/openai_codex_routing_test.go:37-80覆盖UA/未知版本/上下界/黑白名单/AppServer/缺与有body指纹/ForceCLI，并改UA后验证快照不变及取消上下文仍有效。正式backend收据通过；全局设置真实数据库并发更新未实测。 |
| A4 | passed | brief.md | A4：默认、分层和旧调度的普通候选、会话粘性、previous_response_id、影子账号、等待后/数据库刷新复核及故障重选均不能选中不合格的双开账号；可迁移时选合格账号，不能安全迁移时沿用既有失败规则；不会因单次请求不匹配把账号标记为健康故障或全局暂停。 | 独立核对普通共享兼容、default sticky classifier openai_account_scheduler.go:638与layered:693、legacy eligibility openai_gateway_scheduling.go:420、DB刷新:1664-1719、previous_response resolver openai_ws_forwarder_support.go:583,619及shadow调用共享兼容。openai_profit_control.go:335-351取不落后快照后终检，不写账号健康/暂停。HTTP acquire三路径handler/openai_gateway_handler.go:2655-2764和WS:3232-3295否决释放且用独立profitVetoCount重选，不消耗switchCount。严格WS仍由openai_ws_forwarder_ingress.go:1350-1355,1734强制原连接，openai_ws_pool.go:1155-1181按账号池验证连接，缺失本地失败；handler:3904仅可重建上下文剥previous_response_id，未新增破坏性迁移。service新增调度/previous/DB/shadow及handler acquired/fast/wait测试和正式扩展scheduler回归通过。强绑定全部模式组合和真实缓存竞态无法实测穷尽。 |
| A5 | passed | brief.md | A5：Responses（含 compact）、Chat Completions、Messages 兼容、WebSocket 首次选号/重选及其他可选 OpenAI OAuth 的业务入口都能正确传递客户端判定信息；WS 使用入站握手头与首条生成消息；筛选不得消费掉后续转发所需请求体或破坏请求体句柄的释放。 | 完整读取brief58行及spec99行，并从backend/internal/server/routes/gateway.go:77-84,195-197,215-227真实路由反查：Responses/compact、Chat、Messages、Embeddings、Images、AlphaSearch、WS与两个计数均在选号前Prepare。本轮handler/openai_gateway_count_tokens.go:65在model替换前、:222在coordinator/parse/映射前提供原body；后续WithContext链保留快照，:104和393 SelectAccountForTokenCount共用legacy资格。CountTokens:228既有defer Cleanup保留，service计数:86-91上游句柄cleanup保留；新快照仅结果不额外持有完整body。WS handler:2879-2880用握手头和首条消息复制判定到原租约取消ctx；重选沿用。其他grok媒体强制Grok；Seedance按account.go:1970-1973仅APIKey，模型目录属明确非目标。handler七HTTP/WS及本轮6计数场景均在当前backend正式收据中通过。 |
| A6 | passed | brief.md | A6：全部候选被过滤时使用当前端点无可用账号/强绑定失败响应，无不合格上游请求；不越组、不越权、不放松模型/能力约束；未启用新子开关的账号仍保持原有拦截语义。 | handler/openai_codex_routing_test.go:41-72及127-150证明七HTTP与WS无匹配时不发上游、不产生账号runtime健康项；:108-121两个计数全过滤明确无403且upstream.accountIDs为空。新规则是原平台/模型/能力/分组资格函数中的附加拒绝，没有扩大候选范围；DB复核openai_gateway_scheduling.go:1695-1711保留资格及组检查。service/openai_gateway_count_tokens.go:59-66,292-299防御仅双开触发；新增openai_gateway_count_tokens_test.go:62-96四场景证明双开无上游403，子关保持旧计数200且上游实际调用，SetError/TempUnsched均0。生成转发旧detector仍在，Messages新增防线按已提交状态发SSE错误。复用当前backend正式收据通过；真实上游未调用。 |
| A7 | passed | brief.md | A7：创建、编辑和批量编辑提供从属父开关的新子开关及中英文说明，默认关闭；保存、回显、重置、父开关关闭和批量未选更新行为正确；账号复制及重新授权不丢失该配置。 | 独立审查完整实际frontend diff：CreateAccountModal.vue:3547,4873,5379,5420,5879,5990-6002父从属/默认/平台类别重置/保存清键；EditAccountModal.vue:2709,4928,4979,6149读取/切账号重置/保存清键；BulkEditAccountModal.vue:976-1022及2189-2201仅同次显式开启父并选择更新才写子，2539附近重开复位。中文/英文在仓库既有frontend/src/i18n/locales/{zh,en}/admin/accounts.ts:827/717，brief旧locales描述按用户明确的实际目录核对。三表单新增payload回归及locale正式检查4文件264测试通过，typecheck通过。复制admin_account.go:158-166深拷贝且未丢该键；实际重新授权frontend/src/components/account/openaiReauthUpdatePayload.ts:15-31合并currentExtra，后端admin_account.go:684-764亦保留nil extra语义。浏览器手工操作及真实OAuth重新授权未实测。 |
| A8 | passed | brief.md | A8：保留可运行的针对性回归检查，覆盖账号配置、混合账号池、全部排除、粘性与故障重选、入站指纹以及前端 payload；完成相关后端检查、前端类型检查和定向测试，独立 Verifier 核对 A1–A8 与完整规格；未运行检查如实记录。 | 全新独立只读会话首个Runtime动作claim Action12成功running，verifierExecutionRef与领取session一致；独立完整读取A1-A8/完整Spec，审查实际tracked diff及3个untracked新增源码/测试，未采用Builder或前轮通过结论。核对本候选77c2b9…的verify.checks：build91012ms、backend17459ms、frontend27475ms、types29044ms全部exit0；inputBefore=inputAfter=641a5e52fd2fc57511f20c7ac15dc9a51357887ffadaeaece5c21f4280373c4e，snapshot.complete=true且diagnostics=[]。读取并SHA256核对4份准确logRef均匹配；frontend264/264，backend service7.594s/handler4.642s。检查已覆盖新10计数场景、配置/策略/混合/全过滤/sticky/previous/slot及前端payload，复用正式收据未重复执行build或检查。 |

## 检查

| 检查 | 命令 | 工作目录 | 状态 | 退出码 | 耗时 |
| --- | --- | --- | --- | ---: | ---: |
| Frontend production build | build | frontend | passed | 0 | 91012 ms |
| OpenAI routing and scheduler regressions | test -tags unit ./internal/service ./internal/handler -run TestCodexRouting\|Test.*Codex.*Restriction\|TestAccount_IsCodexCLIOnly\|Test.*(Scheduler\|Layered_\|PreviousResponse\|ProfitControl\|SlotProfit\|CountTokens\|ResponsesInputTokens)\|TestDuplicateAccount -count=1 | backend | passed | 0 | 17459 ms |
| Account form and locale regressions | exec vitest run src/components/account/__tests__/CreateAccountModal.spec.ts src/components/account/__tests__/EditAccountModal.spec.ts src/components/account/__tests__/BulkEditAccountModal.spec.ts src/i18n/__tests__/localeKeyCompleteness.spec.ts | frontend | passed | 0 | 27475 ms |
| Vue typecheck | typecheck | frontend | passed | 0 | 29044 ms |

### Builder 报告的证据

以下为 Builder 报告，不等同于 Runtime 检查凭据或独立验收结果。

- token count omitted routing RED reproduction: passed — 修复前执行 go test -tags unit ./internal/handler -run '^TestCodexRoutingTokenCountEntries$' -count=1 实际退出1，按预期抓到两入口all-filtered返回200及mixed错选账号1。该条通过指复现成功，不把失败命令称测试通过。
- current backend routing and count regression GREEN: passed — 本轮 go test -tags unit ./internal/service ./internal/handler -run 'TestCodexRouting|Test.*(CountTokens|ResponsesInputTokens)' -count=1 退出0，service5.006s handler4.581s。包含全部现有TestCodexRouting和计数功能回归及新增10场景。
- current whitespace: passed — 本轮git diff --check退出0，仅既有前端CRLF换行提示。
- prior candidate broader Runtime checks: passed — 父会话报告Action8四命令build93670ms/backend17727ms/frontend26962ms/typecheck29523ms均exit0且complete=true。此为修复前候选的历史记录，不能替代新候选Runtime。
- current frontend build tests typecheck and formal Runtime: not-run — 依父会话协调本轮仅后端开发检查，前端未改；生产build FIRST、后端SECOND、前端THIRD、类型FOURTH计划交父会话实际执行。
- full repository suites browser real upstream: not-run — 未运行全仓suite、浏览器人工验收或真实上游；无临时PG启动且未操作现有数据库。
- 已知限制: Builder自检不是独立验收；新的Runtime检查及新Verifier由父会话后续执行。
- 已知限制: 本轮无前端改动，未重跑前端build/test/typecheck；未跑全仓suite、真实上游或浏览器人工检查。

## 阻塞项

_无。_

## 风险与跳过的工作

- 没有发现有证据支持的新增Blocker。结论基于当前源码/实际diff、可运行测试内容和当前候选正式Runtime通过收据；不宣称穷尽所有运行环境。
- 未运行全仓完整suite、浏览器手工交互、真实OAuth重新授权、真实上游或真实Redis/DB并发竞态；这些场景的部署实效无法验证。无临时PostgreSQL启动、未动现有数据库。
- WS强绑定保护的静态控制流及现有定向收据已核对；未另外实测所有账号类型、WS模式、store策略与工具续链的笛卡尔组合。Images multipart的body路径保持既有detector含义，不把协议转换生成的JSON当入站指纹。
- build日志有大于500kB chunk警告，非失败，现有全局bundle拆分性能未做基准；不归因为本次路由开关回归。
- 没有修改业务/测试/brief/spec，未commit/push/archive或接受验收结果；仅正式Runtime领取/回报和临时JSON。个人学习no-observation已verified；项目学习not-run，如实披露，不声称已完成项目学习。

## 之前的迭代

| 目标周期 | 迭代 | 尝试 | 结果 | 未解决项 | 摘要 | 完成时间 |
| ---: | ---: | ---: | --- | --- | --- | --- |
| 1 | 1 | 0 | recovery | — | Runtime checks completed, but their input changed before acceptance. Preserve the original receipts and approve a new Builder candidate; do not replay the old checks. | 2026-11-06T18:02:45.107Z |
| 1 | 1 | 0 | recovery | — | The user chose to repair the implementation after Runtime checks stopped. | 2026-11-06T22:05:28.234Z |
| 1 | 2 | 1 | fail | A2, A4, A5, A6 | 需整改：一个阻断根因——两个真实token-count业务入口缺少入站Codex路由快照，使无快照放行分支可选中不合格双开OAuth并发上游，违反A2/A4/A5/A6。A1/A3/A7/A8通过；本候选四项正式Runtime检查收据有效且全部通过，但现有七入口测试遗漏计数端点。请返回Build修复并交新候选，由新独立Verifier完整复核；未自改源码。 | 2026-11-06T22:16:12.065Z |
| 1 | 3 | 1 | pass | — | 通过：当前候选A1-A8全部通过。两个真实token-count入口已在原始body转换前准备入站判定，选号阶段排除双开不合格账号，6个handler场景证明全排除无403/无上游、混合仅B、子关保留旧计数行为；4个service场景证明新增防御仅双开且不污染账号健康。其余调度/粘性/previous/shadow/DB/抢槽/WS、detector复用、UI及配置保留已独立重新核对。四项当前候选正式收据及日志哈希匹配，全部通过；无额外检查请求。等待用户决定，不接受或归档。 | 2026-11-06T22:32:02.658Z |



## 结论

通过：当前候选A1-A8全部通过。两个真实token-count入口已在原始body转换前准备入站判定，选号阶段排除双开不合格账号，6个handler场景证明全排除无403/无上游、混合仅B、子关保留旧计数行为；4个service场景证明新增防御仅双开且不污染账号健康。其余调度/粘性/previous/shadow/DB/抢槽/WS、detector复用、UI及配置保留已独立重新核对。四项当前候选正式收据及日志哈希匹配，全部通过；无额外检查请求。等待用户决定，不接受或归档。
