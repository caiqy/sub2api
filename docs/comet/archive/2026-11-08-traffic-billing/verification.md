---
generated_from_state_version: 15
---

# 验证

## 当前结果

- 结果: **已归档**
- 验证情况: **已完成检查，验证结果已确认**
- 目标周期: 1
- 迭代: 2
- 验证器尝试次数: 1
- 完成时间: 2026-11-08T16:48:48.252Z
- 摘要: 独立Native Verifier execution native-verifier-independent-20261109-0042：A1-A12恰好各一次全部passed，verdict=pass。只采第二候选db639bdaf2d3493d448660249ef3cc0a0c1b560df1e6cf509facc2d38bafc344/action traffic-billing:12正式检查，不采用旧a97/action9结论。五项真实日志逐一读取并计算SHA256匹配回执：backend-unit 27421ms/37f69d74cb8164c05d5fd6ec7bfa61db35085439dea456870bae04e1fc4a9bf6；ws-relay 5696ms/f9d1b9c4f046a126610963fefc41f89575ed35f63844963cb683d8d44381c7e2；frontend-tests 16079ms/0218c263a4233c6eb4c312717c17bfb8b377255a5aca5403e425c06b7caedea0；frontend-build 94265ms/55dd8a9bbd48e65518597350298b48259d0161b1c932c86f0422d802e86dd382；diff-check 611ms/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855，全部exit0/passed。inputSnapshot complete/diagnostics[]且候选输入gate=1789d8cddc57441b9ef2c96bd43cebc10d714bc6b1263e5d75aef4b43e77e251一致；检查前后工作树指纹不同源于build生成物，未混用旧候选。本execution额外只读重跑真实交付专项exit0；真实PG统计原始PASS和外部390px浏览器事件独立查阅作为补充，不代替正式回执。未修改普通项目实现、未启动服务、未运行next/accept/archive或Git外部交付。 VerifiedAt=2026-11-08T16:48:46.014Z

## 验收

| 编号 | 结果 | 来源 | 验收项 | 原因 |
| --- | --- | --- | --- | --- |
| A1 | passed | brief.md | A1：升级后新老用户个人授权为关，总开关关、T=10 MB、P=0；未配置正价格不能开启总开关，非法/非有限配置被服务端拒绝。 | 独立阅读正式brief/spec全篇；backend/migrations/252_traffic_billing.sql:1-7新增用户授权默认false、历史traffic_cost=0及nullable账号基准；backend/internal/service/setting_traffic_billing.go:21-79默认off/10MB/0，拒绝非有限数、非正阈值、负价、开启且零价；backend/internal/repository/setting_repo.go:137-151事务锁内合并配置再校验，部分更新不能绕过。前端SettingsView.vue:11696-11708具备相应输入校验。第二候选backend-unit及frontend-tests正式通过。 |
| A2 | passed | brief.md | A2：全局关时现有同步请求准入与扣费不新增限制；全局开且个人关时S=T放行免费、S>T返回413与授权错误且不访问上游、不扣款；已授权仍被原硬上限限制。 | backend/internal/handler/gateway_inflight_reservation.go:138-171在上游调用前取得授权/配置快照并校验，TrafficBillingSnapshot.ConsentRequired严格S>T，未授权返回413；禁用不增加限制，S=T免费，原硬请求上限仍由原中间件执行。核查messages/chat/responses/OpenAI/Gemini/图片等reserveInflightBalance调用点与routes/traffic_billing.go原始长度捕获。handler流量测试覆盖边界、未知长度与排除路径，第二候选backend-unit通过。 |
| A3 | passed | brief.md | A3：T=10 MB、P=0.01 $/MB时，输入10,000,000字节流量费0，输入12,000,000字节流量费0.24美元，响应任意大小不改变结果；小数字节按比例，无整MB进位。 | backend/internal/service/traffic_billing.go:128-176使用原始InputBytes，S<=ThresholdBytes为0，越阈值按float64(S)*2/1e6*PricePerMB全量比例计算；不使用响应长度、不向上取整MB。测试覆盖10,000,000免费及12,000,000*2/1e6*.01=.24；前端UsageTable流量用例核查准确字节与12 MB×2×$.01/MB，91例正式通过。 |
| A4 | passed | brief.md | A4：改变模型价或用户/分组/高峰倍率不改变同样输入对应流量费；模型价0或倍率0仍可收0.24美元，账号成本/额度不包含这0.24美元。 | traffic_billing.go在模型倍率之后独立加客户TotalCost/ActualCost，冻结AccountStatsCost为原模型基准；gateway_usage_billing.go:352-366账户额度仍取纯模型cost.TotalCost。traffic_billing_record_usage_test.go:54-89双管线零模型/零倍率仍收.24到订阅、账户0。repository统计查询使用COALESCE(account_stats_cost,total_cost-traffic_cost)。真实PG原始日志traffic-billing-pg-original-evidence.txt:164-201显示指定统计隔离用例PASS .07s/exit0，Postgres16/Redis8.4实际ready，无SKIP；源码断言客户1.24/2.24/traffic.24、账号倍率1.5、显式override0/.4。 |
| A5 | passed | brief.md | A5：模型费和流量费使用同一订阅/余额来源，总额、实际扣款、用户/Key消费限制、消费汇总一致，模型费0的订阅请求仍走订阅，不回落余额。 | gateway_inflight_reservation.go:218-256把预测流量加入同源估算，traffic>0走ReserveFixedCharge，即使模型0或原估算开关关闭仍校验；billing_cache_service.go:856-955校验余额或订阅日周月及Key总额/5h/1d/7d限制、平台额度和pending，失败释放预留。gateway_usage_billing.go:352-366最终选择同一订阅或余额并包含客户流量，账户额度隔离；正式backend-unit通过，双管线零模型订阅测试证实SubscriptionCost=.24、BalanceCost=0。 |
| A6 | passed | brief.md | A6：失败、超时、客户端流式断开、输入未知无流量扣款；正常成功一次请求收费一次，多次内部重试/重复结算不重复，模型原有费用保留；输入未知留有原因。 | 关键旧问题本轮自行复核并额外运行go -C backend test -tags unit ./internal/service ./internal/handler -run 'TestTrafficBillingUpstreamFailureRetainsModelBill\|TestTrafficBillingHTTPDeliveryThroughHandlers' -count=1 -v，exit0，完整日志C:/Users/caiqy/AppData/Local/Temp/traffic-verifier13-delivery.log:2-28及138-152。真实Forward/ForwardAsChatCompletions/ForwardAsResponsesHandle遇partial EOF保留模型usage且traffic0；真实三Handler×partial/write_failure/short_write/success经shared submission→RecordUsage，失败0/成功.24。gateway_inflight_reservation.go:68-99捕获Write/WriteString err和nil-error短写到共享atomic DeliveryFailed；traffic_billing.go:109-117、gateway_usage_billing.go:899及openai_gateway_usage.go:522最终成功门控制收费；partialStreamUsageResult明确TrafficDeliveryIncomplete。unknown免收；最终数据库Apply沿用request-id幂等，非Applied不再次扣费。不把私有处理方法必须返回非nil err当作新需求。 |
| A7 | passed | brief.md | A7：请求开始后调价、改阈值、切换开关不改变本单位金额；下一请求使用新配置/授权。历史依据可复算、旧记录为0且不被追扣，用户授权变更使鉴权缓存失效。 | traffic_billing.go的请求快照固定enabled/consent/threshold/price/input，日志持久化所用历史基准；raw body输入在释放/协议改写前捕获。HTTP授权鉴权缓存含用户字段并双向映射，UserService.UpdateProfile在事务和普通分支均于AllowLargeRequests更新后InvalidateAuthCacheByUserID（user_service.go:480-506），WS每turn读取最新用户并固定该turn快照。相关setting/user/auth-cache/traffic单元与前端profile用例正式通过；历史数据默认0，不依据当前配置回算。 |
| A8 | passed | brief.md | A8：同步各平台文本/图片/音频入口与别名适用同规则；批处理、异步图片、异步视频及其状态/下载结算不收流量费、不执行新授权门槛，原模型费与硬限制保持。 | 独立搜索并审查同步messages、chat、responses、embeddings、Gemini、同步图片、搜索及Grok音频等reserveInflightBalance接入；routes/traffic_billing.go显式排除GET及batch/batches/async/videos/tasks路径，image_task_handler.go显式Excluded，InflightEstimateVideo排除视频。捕获的是外部原始请求体而非内部重写或响应。backend handler/service及相关routes正式检查通过；不把Windows integration harness空执行exit0视为覆盖证据。 |
| A9 | passed | brief.md | A9：OpenAI Responses WS按每轮原始客户端payload判断准入、冻结并结算；异常终态、客户端断流drain、部分图片模型成功例外不误收流量费，升级后错误通过协议发送。 | backend/internal/service/traffic_billing_ws.go:17-22要求turnErr=nil且ClientDeliveryComplete和SucceededForScheduling；WS relay按每turn原始客户端payload长度统计，在上游发送前执行授权/额度门，协议转换和上游复用不重复计数。traffic_billing_ws_test.go及wsrelay测试覆盖终端成功、写失败、取消/断开等分支。第二候选独立ws-relay正式检查5696ms/exit0，实际log SHA256 f9d1b9c4f046a126610963fefc41f89575ed35f63844963cb683d8d44381c7e2匹配。 |
| A10 | passed | brief.md | A10：Grok实时语音按本次提案的整连接累计输入，在正常1000关闭且有既有可结算音频时收一次；异常关闭免流量费，未授权累计超阈值时协议报错并关闭，模型费保持原规则。 | Grok realtime按全连接累计客户端payload、包含control文本；grok_realtime_traffic_test.go:16-63通过真实本地WebSocket发送control和audio，越阈值在发送上游前拒绝，断言计入二者字节且拒绝audio不造模型账单。traffic_billing_ws.go中Grok成功门只接受有audio且Close1000；1001/1005/1006、超时/错误/未知均不收费。连接稳定请求ID沿用幂等。正式backend-unit及ws-relay通过。 |
| A11 | passed | brief.md | A11：simple普通路径仅记流量费不扣余额/订阅，simple Key窗口模式仅纳入窗口；标准模式同样输入按规则实际收费。 | traffic_billing_record_usage_test.go覆盖gateway和openai双管线simple模式：无限速只写含traffic的usage日志，不扣余额/订阅/Key总额度/账户；有限速只产生APIKeyRateLimitCost=.24，其余字段0。gateway_usage_billing.go:380-387限速路径不能退回无幂等legacy扣款；billing_cache_service.go:959起simple限速DB authoritative，错误fail closed。正式backend-unit通过。 |
| A12 | passed | brief.md | A12：个人资料显示当前价格与默认关闭开关；管理员功能开关表单可保存合法设置；管理员/用户账单均显示流量费、精确输入计费依据和可鼠标悬停/键盘聚焦的问号提示“包含接收和发送双边流量”；总费用与导出已有总额一致，不新增导出分项，原输入/输出/总计展示保留。 | 独立审查ProfileTrafficBillingCard的默认关闭/授权保存、SettingsView功能设置和校验、管理员与用户共享UsageTable；流量行位于缓存项后服务档位前，保留原请求体区块，显示exact bytes、阈值、正确MB×2×单价，标准/实际合计与CSV沿用含流量值，账号列只用隔离基准。正式frontend 4文件91例通过（含focus→click幂等、小屏几何/未知字节等）且i18n3例/type/production build通过。独立读取外部原始session b2e78fa3-4b2c-4ff2-8690-0cd8c3b38d5d seq1950/1955/1960：2026-11-08T16:33:38.000Z附近实际click问号成功，390px问号161.575..175.575、说明100.075..237.075均可见；快照展示原始1.24/扣费2.24与12,000,000B/.24。原生button支持focus和click，click只幂等打开不toggle，标准触屏click无需hover即可访问；未声称物理触屏验收。 |

## 检查

| 检查 | 命令 | 工作目录 | 状态 | 退出码 | 耗时 |
| --- | --- | --- | --- | ---: | ---: |
| 后端受影响链路单元回归 | -C backend test -tags unit ./internal/service ./internal/repository ./internal/handler ./internal/handler/admin ./internal/handler/dto ./internal/server/routes -run TestTrafficBilling\|Test.*RecordUsage\|TestReserveFixedCharge\|TestInflightReservation\|TestReserveInflightBalance\|TestBuildUsageBillingCommand\|TestApplyUsageBillingSimpleMode\|TestCheckBillingEligibilitySimpleMode\|TestGeminiCachedContent\|TestGrokVoice\|TestGrokRealtime\|TestResponsesWSTraffic\|TestAPIKeyAuthSnapshot\|TestUsageLog\|TestUser.*Profile\|Test.*Setting\|Test.*PlatformQuota\|TestSystemPlatformQuota\|TestAllowUserViewErrorRequests -count=1 | . | passed | 0 | 27421 ms |
| WebSocket relay整包 | -C backend test ./internal/service/openai_ws_v2 -count=1 | . | passed | 0 | 5696 ms |
| 前端流量相关回归 | -NoProfile -Command pnpm --dir frontend exec vitest run src/components/admin/usage/__tests__/UsageTable.trafficBilling.spec.ts src/components/user/profile/__tests__/ProfileTrafficBillingCard.spec.ts src/views/user/__tests__/ProfileView.spec.ts src/views/admin/__tests__/SettingsView.spec.ts | . | passed | 0 | 16079 ms |
| 前端i18n/类型/生产构建 | -NoProfile -Command pnpm --dir frontend run build | . | passed | 0 | 94265 ms |
| 补丁空白错误检查 | diff --check | . | passed | 0 | 611 ms |

### Builder 报告的证据

以下为 Builder 报告，不等同于 Runtime 检查凭据或独立验收结果。

- 后端相关包定向单元回归: passed — service/repository/handler/admin/dto/routes相关设置、授权、费用、WS、预占和simple测试通过。
- 真实PostgreSQL迁移和统计回归: passed — Linux交叉编译后WSL执行TestUsageLogRepoSuite/TestTrafficBillingStatsKeepAccountCostsSeparate PASS，客户1.24/2.24、流量.24、账号1.5和显式override0/.4均验证；非SKIP。
- 前端定向回归与类型检查: passed — 四测试文件全部用例及vue-tsc通过。
- 真实浏览器本地验收: passed — 隔离本地实例验证初始授权关闭、开关持久化、0价格拒绝开启、合法保存、价格和系统状态、管理员/用户共享费用及鼠标/键盘提示。
- 第二轮失败交付和小屏专项回归: passed — 三个真实HTTP Handler×四场景12case与三个实际Forward→RecordUsage回归通过；前端四文件91例+vue-tsc通过；390px浏览器问号实际点击及气泡边界确认。
- 已知限制: 未连接真实外部收费上游；HTTP/WS交付及失败边界使用现有测试替身与真实本地WebSocket回归。批处理和异步按已确认范围排除。
- 已知限制: 移动端使用iPhone14视口/touch能力模拟；Bsk原生click实际产生mouse pointer，focus→click合成序列由回归覆盖，未使用物理触屏设备。

## 阻塞项

_无。_

## 风险与跳过的工作

- [Suggestion] 验证边界：物理触屏设备行为无法验证。外部浏览器虽maxTouchPoints=5/hover:none，原生click实测pointerType仍为mouse；A12通过依据为真实390px click/布局、原生button事件处理及focus→click回归测试，不能宣传为物理iPhone触摸实测。
- [Suggestion] 第二候选frontend-tests实际日志有router-link/watch-source/jsdom AggregateError噪音，但最终4文件91例通过；production build有Browserslist/chunk>500k/mixed-import警告。未发现这些警告构成本次A1-A12阻断的实证，保留现状风险，不扩展本次实现。

## 之前的迭代

| 目标周期 | 迭代 | 尝试 | 结果 | 未解决项 | 摘要 | 完成时间 |
| ---: | ---: | ---: | --- | --- | --- | --- |
| 1 | 1 | 1 | fail | A6, A12 | 需整改，verdict fail。独立逐项核对A1-A12；A6实测failed，A12 blocked，其余按源码及匹配冻结记录passed。2026-11-08T16:13:30.000Z。上游部分失败/下游写失败成功状态未贯通，不能接受候选。 | 2026-11-08T16:14:25.895Z |
| 1 | 2 | 1 | pass | — | 独立Native Verifier execution native-verifier-independent-20261109-0042：A1-A12恰好各一次全部passed，verdict=pass。只采第二候选db639bdaf2d3493d448660249ef3cc0a0c1b560df1e6cf509facc2d38bafc344/action traffic-billing:12正式检查，不采用旧a97/action9结论。五项真实日志逐一读取并计算SHA256匹配回执：backend-unit 27421ms/37f69d74cb8164c05d5fd6ec7bfa61db35085439dea456870bae04e1fc4a9bf6；ws-relay 5696ms/f9d1b9c4f046a126610963fefc41f89575ed35f63844963cb683d8d44381c7e2；frontend-tests 16079ms/0218c263a4233c6eb4c312717c17bfb8b377255a5aca5403e425c06b7caedea0；frontend-build 94265ms/55dd8a9bbd48e65518597350298b48259d0161b1c932c86f0422d802e86dd382；diff-check 611ms/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855，全部exit0/passed。inputSnapshot complete/diagnostics[]且候选输入gate=1789d8cddc57441b9ef2c96bd43cebc10d714bc6b1263e5d75aef4b43e77e251一致；检查前后工作树指纹不同源于build生成物，未混用旧候选。本execution额外只读重跑真实交付专项exit0；真实PG统计原始PASS和外部390px浏览器事件独立查阅作为补充，不代替正式回执。未修改普通项目实现、未启动服务、未运行next/accept/archive或Git外部交付。 VerifiedAt=2026-11-08T16:48:46.014Z | 2026-11-08T16:48:48.252Z |



## 结论

独立Native Verifier execution native-verifier-independent-20261109-0042：A1-A12恰好各一次全部passed，verdict=pass。只采第二候选db639bdaf2d3493d448660249ef3cc0a0c1b560df1e6cf509facc2d38bafc344/action traffic-billing:12正式检查，不采用旧a97/action9结论。五项真实日志逐一读取并计算SHA256匹配回执：backend-unit 27421ms/37f69d74cb8164c05d5fd6ec7bfa61db35085439dea456870bae04e1fc4a9bf6；ws-relay 5696ms/f9d1b9c4f046a126610963fefc41f89575ed35f63844963cb683d8d44381c7e2；frontend-tests 16079ms/0218c263a4233c6eb4c312717c17bfb8b377255a5aca5403e425c06b7caedea0；frontend-build 94265ms/55dd8a9bbd48e65518597350298b48259d0161b1c932c86f0422d802e86dd382；diff-check 611ms/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855，全部exit0/passed。inputSnapshot complete/diagnostics[]且候选输入gate=1789d8cddc57441b9ef2c96bd43cebc10d714bc6b1263e5d75aef4b43e77e251一致；检查前后工作树指纹不同源于build生成物，未混用旧候选。本execution额外只读重跑真实交付专项exit0；真实PG统计原始PASS和外部390px浏览器事件独立查阅作为补充，不代替正式回执。未修改普通项目实现、未启动服务、未运行next/accept/archive或Git外部交付。 VerifiedAt=2026-11-08T16:48:46.014Z
