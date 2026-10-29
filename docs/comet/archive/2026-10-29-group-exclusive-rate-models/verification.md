---
generated_from_state_version: 11
---

# 验证

## 当前结果

- 结果: **已归档**
- 验证情况: **已完成检查，验证结果已确认**
- 目标周期: 1
- 迭代: 2
- 验证器尝试次数: 1
- 完成时间: 2026-10-29T17:27:46.937Z
- 摘要: 通过：全新只读Verifier会话独立领取Action11，补查后同会话领取Action13，全面核验A1-A7并判定全部通过。上一轮A6映射预占及WS旧分组快照问题有生产链路与真实入口回归支持，未改旧Verifier反例经本轮Runtime复测通过。Builder自评仅为线索，结论基于独立源码检查、候选绑定Runtime日志、原始PG与独立浏览器证据。只提交验收结果，等待用户接受；未accept/archive/push。学习检查已applied，personal no-observation，project submitted memo-fccf6cbf，submissionVerified=true且无待评审。

## 验收

| 编号 | 结果 | 来源 | 验收项 | 原因 |
| --- | --- | --- | --- | --- |
| A1 | passed | brief.md | A1：关闭模型限制时，保持原有专属倍率行为，旧配置默认不受限制。 | 独立核对backend/migrations/249_group_exclusive_rate_models.sql:1为NOT NULL JSONB且默认enabled=false；backend/internal/domain/group_exclusive_rate_models.go:10-22关闭时AppliesTo返回true；backend/internal/service/admin_group.go:1293-1297旧API不传models仍走原Sync不覆盖策略。Runtime exclusive-rate-backend通过，TestExclusiveRateModelsExactMatchBeforeCache覆盖关闭恢复且保留models；Modal默认关闭测试与既有浏览器操作员b143d9cc-00da-4b51-bdc5-f3fcadb3fa56 seq952原始报告一致。 |
| A2 | passed | brief.md | A2：开启模型限制并选中模型后，命中模型应用对应用户专属倍率，未命中模型使用分组默认倍率。 | backend/internal/service/group_exclusive_rate_models.go:30-47先精确判定clientModel，不命中直接返回Group.RateMultiplier，命中才访问用户组倍率；backend/internal/service/gateway_usage_billing.go:817/1019及backend/internal/service/openai_gateway_usage.go:212从OriginalModel选择倍率。Runtime exclusive-rate-backend中的backend/internal/service/group_exclusive_rate_models_billing_test.go:14-62实际调用QuoteUsageCost及Gateway/OpenAI RecordUsage，校验client-A=0.12、client-B/CLIENT-A=0.225、Quote=usage实际费用=余额扣减及Gemini缓存存储费用，成功。 |
| A3 | passed | brief.md | A3：保存后重新打开，开关与模型选择正确回显；取消未保存修改不改变服务端配置。 | frontend/src/components/admin/group/GroupRateMultipliersModal.vue:370-387从服务端重读并复制快照，:531-535取消恢复副本，:539-558仅显式保存且失败保留弹窗/草稿；frontend/src/api/admin/groups.ts:384-393同一PUT发送倍率与配置。backend/internal/repository/user_group_rate_repo.go:25-72同事务保存策略、倍率及outbox。Runtime Modal18项和SQL commit/rollback回归通过；独立读取原浏览器seq952核对保存失败不发success、重试保存和重新打开、取消不改mock状态；另读取backend/internal/repository/group_exclusive_rate_models_postgres_test.go:52-82及先前真实PG原始日志C:/Users/caiqy/AppData/Local/Temp/exclusive-rate-builder-b8-events.txt:1149-1152的PASS10.791s，复用其真实数据库回滚/保留证据。 |
| A4 | passed | brief.md | A4：开启且没有有效模型选择时，前后端均拒绝保存并提示至少选择一个模型，不改变已存配置。 | frontend/src/components/admin/group/GroupRateMultipliersModal.vue:315/:264/:540检查开启且无非空模型并禁保存，:42显示提示；backend/internal/service/group_exclusive_rate_models.go:13-27去空/去重后拒空，backend/internal/handler/admin/group_handler.go:1090-1099在服务/存储前校验。Runtime通过backend/internal/handler/admin/group_handler_exclusive_rate_models_test.go:26-40的HTTP400、提示及零存储调用断言，backend/internal/repository/group_exclusive_rate_models_test.go:45-47证明非法输入零SQL。浏览器原seq952也记录必选提示与保存禁用。 |
| A5 | passed | brief.md | A5：同一模型在各用户间采用各自专属倍率；未设置专属倍率的用户仍按分组默认倍率；分组间配置互不影响。 | 共享策略属于Group，专属数值依旧由backend/internal/service/user_group_rate_resolver.go:49/:74-84以user:group缓存并按对应用户/组查询；未命中模型在缓存前返回默认值。Runtime TestExclusiveRateModelsUserAndGroupIsolation（backend/internal/service/group_exclusive_rate_models_test.go:42-56）覆盖两用户同组0.12/0.18、另一组0.3、无专属0.225及命中/未命中交错，成功。 |
| A6 | passed | brief.md | A6：以客户端请求模型 ID 精确匹配，上游映射不会扩大适用范围；各网关计费及预估路径遵守相同规则，缓存不得使未选模型误用专属倍率。 | 独立追踪修复后的真实调用顺序：Gemini native在backend/internal/handler/gemini_v1beta_handler.go:418映射前保存客户端模型，cache在backend/internal/handler/gemini_cached_contents_handler.go:130同理；backend/internal/service/billing_inflight_reservation.go:471-472按RequestedModel、原始pricing/public上下文、Model依次选身份，:585在专属缓存前筛选；backend/internal/handler/gateway_inflight_reservation.go:110首次冻结身份，:74-80账号重估复用。WS后续turn在backend/internal/service/openai_ws_v2_passthrough_adapter.go:1215-1234及backend/internal/service/openai_ws_forwarder_ingress.go:403-404/:622/:1538先映射/BeforeTurn再outbound；backend/internal/handler/openai_gateway_handler.go:3671刷新并保存turnKey、:3673读取当轮原模型，:3611预占与:3852/:3867最终计费用同一个forTurn；首轮透传无begin时两者都回退连接Key。当前Runtime backend通过真实handler回归backend/internal/handler/group_exclusive_rate_models_inflight_test.go:44（Gemini native/cache双向模型选择）、:97（HTTP重估冻结）、:124（OpenAI/Grok复合图像别名）、:160（三种WS transport策略刷新后预占与实扣0.12→0.225）；非仅helper断言。本人通过Action11 request-checks由Runtime Action12重跑上一Verifier未修改overlay，verifier2-mapped-inflight exit0，日志f1e0792c3acba016ba95be0530676e7bf98eae0d0786ed9a6e666f8e0661b582-verifier2-mapped-inflight.log:4/:6：只选client-A时inflight=Quote=0.001350000，只选上游gemini-2.5-pro时两者=0.002531250，SHA256=6ff0ff2b52da11942582cada65e9ee89b42c3b7c9a57bd35546e3f416678c9d4。profit gate、batch image和auth roundtrip由backend/internal/service/group_exclusive_rate_models_billing_test.go:65/:102及backend/internal/service/group_exclusive_rate_models_test.go:59覆盖。另核对backend/internal/repository/api_key_repo.go:260投影和:1036转换、backend/internal/service/api_key_auth_cache_impl.go:17版本28/:328拒旧/:434与:545双向字段，以及migration:15-18持久化失效触发器、admin_group.go:1314-1316显式失效；旧PG原始PASS覆盖auth投影/outbox。五份Runtime日志SHA全部与回执一致，基础与补查各自输入before/after相同。 |
| A7 | passed | brief.md | A7：模型候选读取失败显示错误并允许重试，不静默清除已保存的选择；关闭开关保留选择，重新开启可恢复。 | frontend/src/components/admin/group/GroupRateMultipliersModal.vue:316-317以已选与候选并集展示，:399-411失败只置错误不清选择且可重试，:430-431/:549复制保存完整models并保留关闭列表。Runtime models.spec.ts:50-79通过关闭/开启、取消、搜索、候选失败重试及候选移除后仍保留。本人读取原浏览器操作员b143d9cc-00da-4b51-bdc5-f3fcadb3fa56 seq952完整原始报告及selected-model-desktop.png，确认键盘、搜索、两项选择、失败重试、关闭保存重开再开启恢复；报告375x667无水平溢出且滚动可达。 |

## 检查

| 检查 | 命令 | 工作目录 | 状态 | 退出码 | 耗时 |
| --- | --- | --- | --- | ---: | ---: |
| Exclusive rate policy billing persistence and handlers | test -tags=unit ./internal/service ./internal/repository ./internal/handler/... -run TestExclusiveRateModels\|TestNormalizeExclusiveRateModels\|TestAdminService_BatchSetGroupRateMultipliers\|TestUserGroupRateResolver\|TestGatewayServiceRecordUsage\|TestOpenAIGatewayServiceRecordUsage\|TestProfitControl\|TestGeminiCachedContent\|Test.*Inflight\|TestOpenAIResponsesWebSocket\|TestRefreshOpenAIWSTurnBillingAPIKey\|TestOpenAIWSTurnBillingAPIKeys -count=1 | backend | passed | 0 | 39449 ms |
| Modal save cancel model search and retry regressions | -NoProfile -Command pnpm exec vitest run src/components/admin/group/__tests__/GroupRateMultipliersModal; exit $LASTEXITCODE | frontend | passed | 0 | 4671 ms |
| Frontend typecheck locale completeness and production build | -NoProfile -Command pnpm run build; exit $LASTEXITCODE | frontend | passed | 0 | 93051 ms |
| Diff whitespace check | diff --check | . | passed | 0 | 546 ms |
| Independent prior counterexample against repaired candidate using unchanged temporary overlay | test -tags=unit -overlay C:\Users\caiqy\AppData\Local\Temp\comet-verifier-9fabd63c\overlay.json ./internal/service -run ^TestVerifierMappedInflightUsesOriginalModel$ -v -count=1 | backend | passed | 0 | 9536 ms |

### Builder 报告的证据

以下为 Builder 报告，不等同于 Runtime 检查凭据或独立验收结果。

- Backend targeted service/repository/handler regressions: passed — Repair2: go test -tags=unit ./internal/service ./internal/repository ./internal/handler/... -run feature plus Test.*Inflight, TestOpenAIResponsesWebSocket, TestRefreshOpenAIWSTurnBillingAPIKey, TestOpenAIWSTurnBillingAPIKeys -count=1 passed: service7.134s repository6.801s handler27.812s admin2.090s. New real-entry tests GREEN after observed RED; all3 WS transports refreshed current policy.
- Real PostgreSQL persistence and rollback: passed — Prior Build actual PostgreSQL run passed full migrations/save/auth/outbox/rollback/off-retain on own postgres:16 port55439, cleaned container/volume and confirmed closed. Repair2 changes only handler/inflight snapshots, no persistence changes; no new DB started.
- Modal focused Vitest: passed — Prior Build focused modal18 tests passed and Runtime check passed; repair2 no frontend changes. Runtime plan retains modal check.
- Vue typecheck and production frontend build: passed — Prior Build typecheck/frontend build and Runtime build passed; repair2 no frontend changes. Runtime plan retains production build.
- Browser actual component functional scenarios: passed — Parent-coordinated automation at http://127.0.0.1:5175/ passed keyboard enable/select, search, empty selection blocks save, save failure retains draft without success/close, successful save closes and reopen shows enabled=true models=[gpt-4o], candidate first-load failure retains selection then retry recovers, switch-off save/reopen disabled and re-enable retains gpt-4o, cancel empty draft/reopen leaves server unchanged. Evidence C:\Users\caiqy\AppData\Local\Temp\sub2api-group-rate-preview\selected-model-desktop.png and related screenshots. Final parent browser report also passed selecting two models and iPhoneSE375x667: save footer reachable by scrolling, document/body scrollWidth375 with no horizontal overflow. Desktop CSS2048x961 DPR1.25.
- Ent schema generation and diff whitespace: passed — go generate ./ent succeeded; git diff --check passed.
- Unchanged independent verifier mapped inflight overlay: passed — go test -tags=unit -overlay C:\Users\caiqy\AppData\Local\Temp\comet-verifier-9fabd63c\overlay.json ./internal/service -run ^TestVerifierMappedInflightUsesOriginalModel$ -v -count=1 passed; actual preserved output C:\Users\caiqy\AppData\Local\Temp\exclusive-rate-repair2-overlay-0d719c.log
- Repair2 VSCode diagnostics: passed — vscodeOperator_readProblems on project returned total0 after repair.
- 已知限制: PostgreSQL integration test requires EXCLUSIVE_RATE_TEST_DSN with CREATE DATABASE permission. Real prior Build run passed; optional test skips without env; repair2 no persistence changes and no new DB started.
- 已知限制: Prior Build real browser preview used mock API; actual persistence/auth and rollback independently verified with PostgreSQL. Repair2 no UI changes. This Builder self-check is not independent acceptance.

## 阻塞项

_无。_

## 风险与跳过的工作

- 本次复用与候选匹配的Runtime四项检查，另由本人请求Runtime执行一项旧独立反例overlay；未修改项目实现、测试或规格。Backend定向命令未提供EXCLUSIVE_RATE_TEST_DSN，当前Runtime未执行真实PG；PG证据仅为本人已核对源码和原始日志的先前开发期实际执行，数据库层本轮无变。
- UI证据是实际Vue组件加mockAPI的原独立浏览器记录和当前Runtime18项Modal测试、类型检查及production build。本会话未重跑浏览器，全栈UI到真实数据库E2E及真实收费上游调用无法验证，未将其宣称为已测。
- 保留现有缓存失效和降级行为；分布式Redis失效延迟、故障注入与生产负载性能未单独实测。本轮检查范围内未发现阻断A1-A7的剩余缺陷。

## 之前的迭代

| 目标周期 | 迭代 | 尝试 | 结果 | 未解决项 | 摘要 | 完成时间 |
| ---: | ---: | ---: | --- | --- | --- | --- |
| 1 | 1 | 1 | fail | A6 | 需整改：A1-A5、A7通过，A6失败。独立只读审查全部7项并复用候选匹配Runtime四项成功证据，通过SDK追加检查实测证伪映射场景inflight与最终报价一致性。未修改项目实现、测试或规格，仅在TEMP生成Go overlay复现。候选e01ab948e5b367684eb1f3108c6a6858238f1f0a88b3693bdfe137f43dfef5c5不得接受或归档，应回Build修复全部相关入口及当轮快照传递后重新验收。 | 2026-10-29T16:56:32.124Z |
| 1 | 2 | 1 | pass | — | 通过：全新只读Verifier会话独立领取Action11，补查后同会话领取Action13，全面核验A1-A7并判定全部通过。上一轮A6映射预占及WS旧分组快照问题有生产链路与真实入口回归支持，未改旧Verifier反例经本轮Runtime复测通过。Builder自评仅为线索，结论基于独立源码检查、候选绑定Runtime日志、原始PG与独立浏览器证据。只提交验收结果，等待用户接受；未accept/archive/push。学习检查已applied，personal no-observation，project submitted memo-fccf6cbf，submissionVerified=true且无待评审。 | 2026-10-29T17:27:46.937Z |



## 结论

通过：全新只读Verifier会话独立领取Action11，补查后同会话领取Action13，全面核验A1-A7并判定全部通过。上一轮A6映射预占及WS旧分组快照问题有生产链路与真实入口回归支持，未改旧Verifier反例经本轮Runtime复测通过。Builder自评仅为线索，结论基于独立源码检查、候选绑定Runtime日志、原始PG与独立浏览器证据。只提交验收结果，等待用户接受；未accept/archive/push。学习检查已applied，personal no-observation，project submitted memo-fccf6cbf，submissionVerified=true且无待评审。
