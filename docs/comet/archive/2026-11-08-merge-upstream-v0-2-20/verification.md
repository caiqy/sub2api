---
generated_from_state_version: 23
---

# 验证

## 当前结果

- 结果: **已归档**
- 验证情况: **已完成检查，验证结果已确认**
- 目标周期: 2
- 迭代: 3
- 验证器尝试次数: 1
- 完成时间: 2026-11-08T06:56:49.505Z
- 摘要: 通过。仅绑定冻结HEAD6b74bb9f93b1207c4218aeda50fccdab23fd353f与当前candidate/execution；八项各一次，未发现本轮Blocker。独立核实当前7项Runtime、可复用旧12项、真实PG11项、关键源码与真实页面。结论不扩大为未执行的外部OAuth/真实付费模型/全部SQL浏览器端到端证明；用户接受与正式提交交由父级。

## 验收

| 编号 | 结果 | 来源 | 验收项 | 原因 |
| --- | --- | --- | --- | --- |
| A1 | passed | brief.md | A1：合并节点第一父精确为 a654a78d79ede218b106fb255f9703837a17a829，第二父精确为 dab829a4e9b7d1cc307189aea8b0319eb29e29d0；目标为候选祖先，不纳入 tag 后 upstream 提交。 | 独立git核查HEAD=6b74bb9f93b1207c4218aeda50fccdab23fd353f，正好两个父节点a654a78d79ede218b106fb255f9703837a17a829、dab829a4e9b7d1cc307189aea8b0319eb29e29d0；上游目标由第二父关系直接为祖先，tag后6b4b37db43443b5e88e424fe3c69602cc3810dfb的is-ancestor退出1。当前Runtime merge-invariants exit0复核相同不变量。末次git status/diff --check为空。 |
| A2 | passed | brief.md | A2：上游 MCP/OAuth、管理员密钥权限和邮箱黑名单功能完整融合；本地鉴权投影、账号限制、用户策略、菜单与密钥作用域不被覆盖，安全边界的相关测试通过。 | 独立读取managementmcp/auth.go:30-104：只接受资源绑定OAuth，私有dispatch以不可由HTTP头伪造的上下文匹配method/path/tool，管理员角色、scope及readonly门保留，MCP使用独立step-up会话。mcp_oauth_service.go:372-405,592-648校验精确redirect/resource/S256及用户实时状态/tokenVersion；admin_auth.go:233-263每次检查scoped key创建者有效管理员且默认拒绝未登记路由，custom_admin_keys.go:11-21仅JWT管理员可管理密钥。api_key_repo_profit_projection_integration_test.go:21-86实DB断言fork利润/额度预支/并发/图片倍率及新增privacy投影。routes/mcp.go:19-56复用原user/admin路由及backend guard。正式Go全包、unit、integration及真实WSL PostgreSQL11项PASS支持鉴权投影、MCP重放/并发/撤销、多实例ID和批量禁用；黑名单handler保留上限及管理员跳过。正常UI账号与scoped key往返执行证据和源码相符。 |
| A3 | passed | brief.md | A3：移除 fork layered 调度、健康探测及模式工厂，收敛上游 default/weighted 调度；旧模式配置不再启用旧实现，无后台探测线程。保留上游独立高级调度设置及普通候选、session/previous-response sticky、fallback/WaitPlan、DB 复核、privacy、模型/生图、guardian/Grok、图片尺寸分流和本地 Codex 客户端资格，不整文件覆盖独立 fork 规则；其他 runtime 热更新照常工作。 | 独立全Go及前端obsolete identifier搜索和openai_default_scheduler_regression_test.go:14-41、setting_handler_retired_scheduler_test.go:17-73确认旧mode及九键即使保留也不激活旧工厂，PUT/GET不接收旧参数且advanced/sticky/TopK/TTFT仍更新。openai_account_scheduler.go:2231-2252仅独立advanced开关/default工厂；634-673、1333-1480、1760-1827、1885-1961覆盖session、普通候选、fallback/WaitPlan和DB复核的Codex、privacy、模型/能力/尺寸、guardian、Grok、quota/profit边界。openai_ws_forwarder_support.go:537-659复核previous_response资格及最新DB状态。旧layered专属生命周期不存在；保留的ModelTrace、倍率和能力探测不属于撤除范围。真实网关页保留advanced、sticky、TTFT及其余runtime配置，与正式测试一致。 |
| A4 | passed | brief.md | A4：保留上游协议、GPT Image 2.6、工具媒体及流式修复；fork 请求体缓存/handle/释放、生图历史、加密内容恢复、透传字段和错误分类保持正确，资源限制不引入请求体丢失、超时契约回归或重试失效。 | 独立读取request_body_handle.go:105-219及request_body_coordinator.go:47-69,137-224,332-347：流式预算、spool转换、raw/effective句柄去重及释放保留。openai_chat_merge_handle_regression_test.go:19-83通过真实服务转发测试验证大于1MiB spool在Responses400/404回退时字段完整和start time保持；openai_merge_config_handle_regression_test.go:21-140及upstream_protocol_fusion_test.go:16-88验证跨协议token字段和borrowed handle不丢失。httputil/body_stream_fusion_test.go:24-59验证超时诊断和解码/归一化预算。openai_upstream_transport_error.go:116-173保留spool错误身份、客户端取消与timeout/nonfailover及RequestSent契约；openai_encrypted_carriers.go:5-66,90-164遇未知加密结构拒绝不安全恢复。openai_images26_test.go:21-105覆盖2.6两模型OAuth/setup/generation/json/multipart参数以及400/404/405/429不重复回退。生图历史useImageHistory.ts:42-121保留分页、失效和过期detail处理。正式Go检查通过；真实上游成功生成未验证，见risks。 |
| A5 | passed | brief.md | A5：计费、专属倍率、利润门、订阅配额预支、用量明细与导出保留 fork 行为，同时吸收上游计费与每 turn 首 token 修复；相关回归通过。 | 独立读取group_exclusive_rate_models_billing_test.go:14-120：真实Quote/RecordUsage对原始客户端model的专属倍率、大小写边界、inflight/profit、WS turn及batch snapshot/hold作数值断言；DB投影回归确认AllowQuotaAdvance/AutoQuotaAdvance及fork计费字段。openai_ws_v2/passthrough_relay_test.go:1765-1843按独立时钟给两turn各700ms FirstTokenMs和2s Duration断言；openai_ws_passthrough_turn_number_regression_test.go:17-89覆盖提交并发及turn1/2/3编号。用量明细/导出相关代码与既有正式Go/前端回归保持，当前相对a688只改SettingsView及其测试，计费/导出后端树未变。Windows integration通过不视为全SQL实测，实DB补充范围及空用量UI限制见risks。 |
| A6 | passed | brief.md | A6：迁移保留已执行文件及校验和、兼容 fork 249/250 历史与新增 MCP 表；旧 layered/probe 配置键及账号 Extra 可留存但不再生效，不破坏历史迁移、不批量清理其他来源临时封禁；Wire 与依赖一致；VERSION 与四段式单架构发布约定保留，不触发发布。 | 独立Git比对及Runtime不变量核查345个既有迁移不变；fork249_group_exclusive_rate_models.sql/250_user_cyber_overflow_exempt.sql和VERSION=0.2.19.7未改，仅新增249_mcp_oauth.sql。migrations_runner.go:20-27,145-181,200-280按完整filename主键、advisory锁和事务应用；同数字前缀不冲突。真实PG串行/升级/幂等测试PASS。旧键/Extra只存不生效，无批量解除其他来源封禁的撤除逻辑；服务传输错误仍按特定故障处理。正式build和go mod tidy -diff退出0，Wire构建依赖一致；release.yml:3-19为workflow_dispatch入口，既有release matrix12测试通过，四段式单架构约定保留。没有发布、main交付或archive操作。 |
| A7 | passed | brief.md | A7：前端撤除截图模式/九项参数及探测专属账号入口，相关 settings/DTO/类型/表单/保存链路同步收回；其余 MCP/设置/账号/密钥/用量与 fork 生图工作台等功能融合且配置可往返，无重复控件；前端测试、typecheck、build 通过，对改变的关键页面做本地预览及浏览器实测。 | 独立搜索Vue/后端DTO与保存链路确认旧mode/九参数/专属probe撤除。当前与旧候选差异精确为SettingsView.vue和SettingsView.gatewayRuntime.spec.ts两文件；删除重复step-up块并新增唯一性/保存绑定回归。本人通过bsk Agent Window访问真实隔离后端已登录页面，亲见安全区唯一step-up、scoped key入口/黑名单、网关advanced/sticky/TTFT及其余设置，MCP已启用接入页和授权空态；未注入token。独立操作员report-settings-mcp.md及report-account-roundtrip.md的正常保存刷新/编辑回显证据与源码对照，包含账号Codex、ModelTrace、quarantine、1K/4K及scoped key3权限、撤销删除；keys/images/usage仅入口/空态。当前Runtime typecheck/build-types/build全部exit0，427文件3722测试及i18n3项通过。未用未执行的外部OAuth或付费生图充当通过依据。 |
| A8 | passed | brief.md | A8：完成冲突和本地关键能力的专项语义 review、后端 go test ./...、相关构建与前端检查，并由新的独立只读 Verifier 覆盖全部验收项。失败、超时和未执行如实记录，不能视为通过；没有用户接受前不归档或交付主线。 | 本独立Verifier先启动正式receipt取得state19并用自身稳定session获取原任务Verify context，完整读取brief46行及Spec47行，源码/检查先读、BuilderSummary最后读；逐条覆盖A1-A8。独立读取当前Runtime state.json checks七项passed/exitCode0；读取父会话原始tool/result事件2119确认旧12项正式全包Go/unit/integration/build/tidy/前端/release均exit0，逐日志核对，并由Git两文件差异证明后端/发布证据适用。真实WSL Docker PostgreSQL16/Redis补充11 PASS无SKIP日志单独核查，Windows跳过不冒充实DB；历史未引号参数调用失败不误报产品故障。浏览器会话已停止；cleanup.md记录本次API/Vite/PG/Redis关闭，本人另以Get-NetTCPConnection确认14173/18080/15432/16379均无监听。末次HEAD仍冻结值且工作树干净。仅项目外独立报告，不提交正式最终结果、不接受交付、不归档或改main。 |

## 检查

| 检查 | 命令 | 工作目录 | 状态 | 退出码 | 耗时 |
| --- | --- | --- | --- | ---: | ---: |
| merge-invariants | -c import subprocess,pathlib base='a654a78d79ede218b106fb255f9703837a17a829' target='dab829a4e9b7d1cc307189aea8b0319eb29e29d0' def git(*args): return subprocess.check_output(['git',*args]) assert git('show','-s','--format=%P','HEAD').decode().split()==[base,target] assert git('rev-parse','HEAD').decode().strip()=='6b74bb9f93b1207c4218aeda50fccdab23fd353f' assert subprocess.run(['git','merge-base','--is-ancestor','6b4b37db43443b5e88e424fe3c69602cc3810dfb','HEAD']).returncode==1 assert pathlib.Path('backend/cmd/server/VERSION').read_text().strip()=='0.2.19.7' paths=git('ls-tree','-r','--name-only',base,'backend/migrations').decode().splitlines() for name in paths: expected=git('show',base+':'+name).replace(bytes([13,10]),bytes([10])) assert pathlib.Path(name).read_bytes().replace(bytes([13,10]),bytes([10]))==expected,name print('exact merge parents, frozen version, excluded post-tag commit, unchanged existing migrations:',len(paths)) assert git('diff','--name-only','a6882b3c47a8da69a23b4f36aaaded1d1b6a90e1','HEAD').decode().splitlines()==['frontend/src/views/admin/SettingsView.vue','frontend/src/views/admin/__tests__/SettingsView.gatewayRuntime.spec.ts'] assert not git('status','--porcelain').strip() print('only verified two-file frontend repair; backend, migrations, CI and release tools unchanged') | . | passed | 0 | 22686 ms |
| frontend-typecheck | node_modules/vue-tsc/bin/vue-tsc.js --noEmit | frontend | passed | 0 | 33346 ms |
| frontend-tests | node_modules/vitest/vitest.mjs run | frontend | passed | 0 | 122215 ms |
| frontend-i18n | node_modules/vitest/vitest.mjs run src/i18n/__tests__/localeKeyCompleteness.spec.ts | frontend | passed | 0 | 26312 ms |
| frontend-build-types | node_modules/vue-tsc/bin/vue-tsc.js -b | frontend | passed | 0 | 31526 ms |
| frontend-build | node_modules/vite/bin/vite.js build | frontend | passed | 0 | 39049 ms |
| backend-build | -NoProfile -Command Invoke-Expression (vfox env --shell pwsh \| Out-String); go build ./...; exit $LASTEXITCODE | backend | passed | 0 | 33585 ms |

### Builder 报告的证据

以下为 Builder 报告，不等同于 Runtime 检查凭据或独立验收结果。

- 旧候选正式完整检查: passed — operation a6d858d2-7510-4b33-8c9f-2f996b0d8ebc 12/12；当前backend及发布工具不变。
- 真实数据库补充: passed — repository-real-postgres-wsl-v2 17363ms，真实Testcontainers PostgreSQL16/Redis，11项PASS，无SKIP。
- 独立A7原候选: failed — 两个同模型step-up控件，已正式回Build16修复，无历史豁免。
- 修复开发回归: passed — 新唯一性断言先红(1 vs 2)；删除14行重复块后SettingsView.gatewayRuntime12项全绿，含已存true读取及false保存。
- 真实浏览器配置往返: passed — 正常setup/admin登录，网关TTFT/advanced、黑名单、账号Codex/ModelTrace/尺寸、scoped key保存刷新编辑撤销，MCP及keys/images/usage；见项目外报告。
- 已知限制: 未调用真实付费模型/图片上游；工作台无可用生产Key，真实UI入口/空态覆盖。
- 已知限制: 未连接外部OAuth客户端执行完整浏览器授权，但实际MCP页面及真实数据库授权/重放/并发/撤销已有覆盖。
- 已知限制: 隔离测试PostgreSQL/Redis及本地API/Vite待验收后停止并确认端口关闭，只处理本次测试资源。
- 已知限制: 无main/push/release/deploy；最终通过仍需用户接受并选择交付。

## 阻塞项

_无。_

## 风险与跳过的工作

- [Suggestion][无法验证] 未完成真实付费模型/图片上游成功调用、真实生成后的历史复用及带真实数据的用量导出浏览器端到端验收；当前证据为源码、自动回归及真实隔离页面/配置往返。没有把空态当成成功推理证明。
- [Suggestion][无法验证] 未连接外部OAuth客户端执行完整浏览器登录/同意/回跳；MCP真实接入页已亲见，PKCE/resource/scope/重放/并发/撤销由服务/路由测试及真实PG仓储补充覆盖，不能扩张为外部客户端互操作实测。
- [Suggestion] Windows integration harness部分Docker用例跳过；额外真实WSL PostgreSQL11项无SKIP补足指定鉴权/MCP/迁移/禁用场景，但并非全部SQL用例均在真实DB执行。
- [Suggestion] cleanup.md:14修正执行覆盖描述：假凭证测试账号触发保留的Responses能力探测向api.openai.com返回401，模型同步502，且曾触发billing-probe入口；不能声称完全没有上游请求，也不能将其视为真实付费模型调用通过。该独立能力探针不等于已撤除layered健康探测。
- [Suggestion] 本地未配置TOTP/Passkey，真实step-up挑战未执行；本轮核实唯一控件，已存值读取及保存绑定由新增组件回归覆盖。隔离数据及复现脚本保留项目外，服务和四端口已关闭。

## 之前的迭代

| 目标周期 | 迭代 | 尝试 | 结果 | 未解决项 | 摘要 | 完成时间 |
| ---: | ---: | ---: | --- | --- | --- | --- |
| 1 | 1 | 0 | recovery | — | Native confirmed acceptance criteria changed; restore the approved documents, or reconcile any running Builder with its original Action and claim before using the latest continuation to revise-requirements on this change and confirm the current Shape. A legacy flattened acceptance snapshot cannot safely establish Markdown boundaries after an edit. | 2026-11-08T01:47:42.473Z |
| 2 | 1 | 0 | recovery | — | Builder handoff Runtime checks failed: merge-invariants | 2026-11-08T03:07:25.516Z |
| 2 | 2 | 1 | blocked | A7 | 需整改（验收阻塞）：当前candidate七项passed，A7 blocked，总verdict=blocked。Git精确双父、冻结版本、345既有迁移和12正式命令检查已核实；不能把全部exit0提升为真实数据库通过，亦不能把公共登录页提升为关键页面浏览器实测。无合法本地后端/管理员会话使A7无法完成，未证实代码缺陷。最终JSON交父代理提交，未接受结果、归档、修改main、推送、发布或部署。 | 2026-11-08T03:28:14.492Z |
| 2 | 2 | 1 | recovery | — | 用户明确选择搭建隔离本地环境，不调整A7。当前候选a6882b3c保持不变，WSL Ubuntu Docker的PostgreSQL16及Redis8.4独立容器仅回环15432/16379；当前候选API18080健康检查200，正常setup完成、临时管理员创建并由正常UI登录，前端14173实际已进入管理设置页。隔离data位于D:\Temp\sub2api-verify-v020\data，无生产操作或tracked代码修改。缺失的合法后端和管理会话现已补齐，继续补做关键页面浏览器操作及真实PostgreSQL MCP/鉴权/迁移检查；尚未宣称这些新增验收完成。原12项Runtime证据与候选一致可复用，需新增真实DB检查。测试结束精确停止本次服务/容器并确认端口关闭。 | 2026-11-08T06:09:10.753Z |
| 2 | 2 | 2 | fail | A7 | 需整改：仅A7失败。当前安全设置页存在两处绑定同一字段的step-up 2FA开关，违反已确认的无重复控件要求；准确归因为既有fork遗留，不改变验收标准。项目保持只读，未修改实现或正式状态；请由Runtime回Build后处理。独立证据记录：D:/Temp/sub2api-verify-v020/browser-evidence/independent-A7-source-review.md。 | 2026-11-08T06:27:28.154Z |
| 2 | 3 | 1 | pass | — | 通过。仅绑定冻结HEAD6b74bb9f93b1207c4218aeda50fccdab23fd353f与当前candidate/execution；八项各一次，未发现本轮Blocker。独立核实当前7项Runtime、可复用旧12项、真实PG11项、关键源码与真实页面。结论不扩大为未执行的外部OAuth/真实付费模型/全部SQL浏览器端到端证明；用户接受与正式提交交由父级。 | 2026-11-08T06:56:49.505Z |



## 结论

通过。仅绑定冻结HEAD6b74bb9f93b1207c4218aeda50fccdab23fd353f与当前candidate/execution；八项各一次，未发现本轮Blocker。独立核实当前7项Runtime、可复用旧12项、真实PG11项、关键源码与真实页面。结论不扩大为未执行的外部OAuth/真实付费模型/全部SQL浏览器端到端证明；用户接受与正式提交交由父级。
