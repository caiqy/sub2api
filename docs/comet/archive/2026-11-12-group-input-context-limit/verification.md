---
generated_from_state_version: 20
---

# 验证

## 当前结果

- 结果: **已归档**
- 验证情况: **已完成检查，验证结果已确认**
- 目标周期: 1
- 迭代: 3
- 验证器尝试次数: 2
- 完成时间: 2026-11-12T09:48:02.410Z
- 摘要: 通过：独立只读核验全部A1-A10；原失败检查ID parser-review-independent经正式refreshChecks刷新后完整passed，六项当前Runtime收据全部passed，inputBefore=inputAfter、snapshot complete=true。VerifiedAt=2026-11-12T09:47:59.777Z

## 验收

| 编号 | 结果 | 来源 | 验收项 | 原因 |
| --- | --- | --- | --- | --- |
| A1 | passed | brief.md | A1：默认关闭；配置/关闭/开启可保存，关闭保留各组阈值，无限制快速路径不计数。 | backend/internal/service/setting_input_context_limit.go:63-86 defaults disabled; gateway_input_context_limit.go:53-69 returns before estimating for disabled/unlimited. frontend InputContextLimitSettings.vue:368-415 saves retained draft while disabled; current candidate Runtime backend and 7 component tests passed. |
| A2 | passed | brief.md | A2：组A=64000、组B=128000，同一计数为80000的请求A拒绝B放行；等于阈值放行，超过1拒绝。 | backend/internal/service/gateway_input_context_limit.go:61-78 uses minPositive and estimate.Tokens <= limit allows equality. gateway_input_context_limit_test.go:220-239 asserts equality/one-over plus original identity freeze; current candidate backend Runtime passed. 80000 vs 64000/128000 follows same numeric comparison; no exact80000 fixture executed. |
| A3 | passed | brief.md | A3：非法组/负数/小数/溢出拒绝且不污染旧配置；保存失败不发布快照；refresh不会覆盖较新的save。 | backend/internal/service/setting_input_context_limit.go:20-46,88-118 validates integer/int32/canonical existing group, saves single JSON before cloned snapshot publication. Shared mutex:113,129 orders refresh/save. admin setting_input_context_limit_test.go:23-52 asserts invalid values preserve old config; service:151-218 tests failure and refresh barrier; current Runtime passed. |
| A4 | passed | brief.md | A4：原绑定组、有效组、合成平台映射和回退按Q2工作；本地拒绝不触发宽松组回退。 | backend/internal/server/middleware/api_key_auth.go:185,294,408 freezes original group before effective route overwrite. service/gateway_input_context_limit.go:56-66 merges original/effective only. handler/gateway_handler.go:1484-1492 rechecks fallback before billing; actual fallback test handler/gateway_input_context_limit_test.go:176-196 verifies initial account only and no rejected usage. Current Runtime passed. |
| A5 | passed | brief.md | A5：文本历史、system、tools/schema、调用参数与结果计入；输出预算不计，缓存不打折；Q1选项对多模态/历史/密文/计数失败有明确且测试过的结果。 | 独立核对backend/internal/service/gateway_input_context_limit.go:83-250：SimpleFold仅协议键，gjson有序遍历全部候选，null跳过，schema/tool业务键保留Raw；已读本仓typedNested、重复/null、output schema与partial测试且当前Runtime passed。新独立Runtime parser-reviewed-corrected-input-v2完整passed exit0，logSha256=ff92bc824f035149659e28c652c1b6ec230fddf39a95cc80c713576b0c73c716；TestParserReviewCorrectedInput upper/simplefold raw=converted1002拒绝、İ/ı raw=converted1放行、scalarNull1002拒绝、两方向重复1003保守拒绝、tools:null无错误且1。脚本16-18行正常三协议精确计数/4 opaque冲突/3schema业务键循环全部执行后总PASS；18拼接JSON外部检查合法。输出预算与缓存不改变估算由service测试17-38/281-285覆盖；unknown交上游符合Q1。 正式refreshChecks刷新原失败计划parser-review-independent完整passed exit0/44421ms，logSha256=dc40be723e0aff611723bef055b12ef7e9d1ba8eac593bec4f6f3d5cb95fe4db；TestParserReviewIndependent wrapper调用已审查CorrectedInput全部断言，日志总PASS。 |
| A6 | passed | brief.md | A6：HTTP七个接点和现有别名均覆盖；count端点不拦，compact行为服从Q3；专用非对话入口范围可核对。 | handler gateway_input_context_limit_test.go:104-174 executes seven actual HTTP handlers, current Runtime passed. routes/gateway.go:218-264,389-438,538-558 wires aliases to same handlers (static inspection, route package reports no tests to run). Shared handler guard:35-36 uses true compact path service/openai_gateway_request_body.go:738-740; count routes separate. Gemini generation-only guard line439; UI locale zh/admin/settings.ts:39-44 explicitly states coverage/exemptions. |
| A7 | passed | brief.md | A7：Messages/OpenAI/Gemini协议状态码及关键字段正确；Responses流式遵从Q4，keepalive不能抢先破坏所选契约。 | handler/gateway_input_context_limit.go:48-72 formats inbound Messages400 type/error/request_id, Chat and nonstream Responses400 code/param, Gemini400 INVALID_ARGUMENT; Responses stream true200 failed SSE using stream_error_event.go:61-92 with sequence/id/status/output/error. Current Runtime protocol/SSE tests passed. openai_gateway_handler.go:695 guard precedes compact keepalive start. Actual Codex auto-compaction compatibility remains unverified. |
| A8 | passed | brief.md | A8：Responses WS首帧、第二轮、两种ingress都检查，返回可解析的终态且无上游生成；拒绝后按约定关闭连接。 | handler/openai_gateway_handler.go:3062 rejects first WS frame before generation and:3674-3679 refreshes per-turn snapshot and checks later turns. handler/gateway_input_context_limit_test.go:48-101 exercises native/passthrough x first/second-turn blocked; asserts parseable response.failed, code, sequence, policy1008 close, no rejected upstream message/usage. Current Runtime passed. |
| A9 | passed | brief.md | A9：策略拒绝不产生账号惩罚、重试、生成计费和正文日志；配置按单请求/turn快照读取。 | handler/gateway_input_context_limit.go:18-30 marks RequestScoped and local business-limit reason with metadata-only log. All rejection guards return before generation; fallback check returns before target billing. ops_error_logger.go:1827,1994 clears residual upstream attribution; handler tests:198-210 verify clearing, HTTP/WS/fallback assert no rejected usage. Current Runtime passed; real balances/subscription DB fault drill not executed. |
| A10 | passed | brief.md | A10：保存节点下一请求读取新配置，健康多节点60秒内刷新；DB故障保留last-known，未知首次值行为与告警可验证。 | service/setting_input_context_limit.go:55-61 clones maps,118 immediately publishes successful save;125-164 TTL60s bounded5s refresh, warn+disabled initial failure, retain last-known and retry5s. Shared mutex prevents old refresh overwriting save. service gateway_input_context_limit.go:48-52 freezes per-request and:259-260 resets next WS turn. Current snapshot/failure/barrier tests passed; physical multi-node propagation/outage unverified. |

## 检查

| 检查 | 命令 | 工作目录 | 状态 | 退出码 | 耗时 |
| --- | --- | --- | --- | ---: | ---: |
| 后端输入准入协议及回归 | test -tags unit ./internal/service ./internal/handler ./internal/handler/admin ./internal/server/middleware ./internal/server/routes -run TestInputContext\|TestEstimateVisible\|TestWriteResponsesFailedSSE\|TestEstimateOpenAIInputTokens\|TestGatewayHandler_MessagesPromptTooLongFallback -count=1 | backend | passed | 0 | 11713 ms |
| 前端国际化类型及生产构建 | -NoProfile -Command pnpm run build; exit $LASTEXITCODE | frontend | passed | 0 | 133654 ms |
| 设置组件交互回归 | -NoProfile -Command pnpm exec vitest run src/components/settings/__tests__/InputContextLimitSettings.spec.ts; exit $LASTEXITCODE | frontend | passed | 0 | 5304 ms |
| Independent typed binding null duplicate opaque and normal count | test -tags unit -overlay C:\Users\caiqy\AppData\Local\Temp\group-context-parser-review-overlay.json ./internal/service -run TestParserReviewIndependent -v -count=1 | backend | passed | 0 | 44421 ms |
| 独立解析补证：已审查合法JSON与typed转换 | test -tags unit -overlay C:\Users\caiqy\AppData\Local\Temp\group-context-parser-review-overlay.json ./internal/service -run ^TestParserReviewCorrectedInput$ -count=1 -v | backend | passed | 0 | 7224 ms |
| Independent typed binding null duplicate opaque and normal count | test -tags unit -overlay C:\Users\caiqy\AppData\Local\Temp\group-context-parser-review-overlay.json ./internal/service -run TestParserReviewIndependent -v -count=1 | backend | passed | 0 | 7017 ms |

### Builder 报告的证据

以下为 Builder 报告，不等同于 Runtime 检查凭据或独立验收结果。

- 后端service/handler/admin输入限制相关回归: passed — 最终helper命令exit0，service0.471s handler0.277s admin0.130s；包含新增typed转换、Unicode、重复键/null和先前协议/WS/回退回归
- git diff --check: passed — 完成后无输出，exit0
- 已知限制: 只统计可见输入；隐藏历史及多模态未知部分交上游
- 已知限制: 含重复可见字段的歧义JSON按全部候选保守累加，可能高估；正常无重复请求不重复计数
- 已知限制: 未验证真实上游客户端自动压缩、物理多节点故障及真实账务数据库副作用

## 阻塞项

_无。_

## 风险与跳过的工作

- 无法验证：真实浏览器设置流程、真实Codex各版本自动压缩、物理多节点故障及真实账务数据库演练；组件/mock/HTTP/WS证据不等于这些实测。
- 含重复歧义可见字段按用户授权保守累加，可能高估；正常无重复输入已独立精确计数验证。
- 本会话仅写Temp请求文件并调用正式Runtime，未改项目业务/测试/规格/报告/状态文件，未git提交、接受、归档或启动服务。
- Runtime refreshChecks:true实际重跑全部既有计划六项（后端、production build、7组件、三个parser ID），全部passed；Verifier只申请原失败检查，不手工修改历史收据。

## 之前的迭代

| 目标周期 | 迭代 | 尝试 | 结果 | 未解决项 | 摘要 | 完成时间 |
| ---: | ---: | ---: | --- | --- | --- | --- |
| 1 | 1 | 1 | fail | A5 | 独立只读验收需整改：A5失败，其余A1-A4/A6-A10通过。可见schema根字段遗漏使Gemini/Anthropic超限输入放行；两项Runtime外部overlay观测证实。无实现/正式产物修改，无DB启动。VerifiedAt=2026-11-12T07:25:16.228Z | 2026-11-12T07:25:17.422Z |
| 1 | 2 | 1 | pass | — | 独立只读验收通过：A1-A10逐项通过，无已确认Blocker。已读完整brief/Spec、全部相关实现及当前Runtime检查；新增12schema正向检查证实修复和边界，最后核对Builder自评。候选未变化，未改项目实现/测试/正式产物，未接受归档/提交推送部署。VerifiedAt=2026-11-12T07:38:19.867Z | 2026-11-12T07:38:23.028Z |
| 1 | 2 | 1 | recovery | — | The user rejected the Skill-coordinated Verifier pass. | 2026-11-12T08:48:02.211Z |
| 1 | 3 | 1 | blocked | A5 | 独立正式只读验收：A1-A4/A6-A10 passed，A5 blocked。原大小写绕过与tools:null反例已实测修复，未确认新业务Blocker；独立overlay末尾fixture JSON错误后纠正检查受Runtime去重及轮数上限阻断，不能伪称完整通过。完整brief/Spec与正式路径已核对；复用当前candidate匹配checks，旧candidate收据未作为新证据。submittedAt=2026-11-12T09:12:39.929Z | 2026-11-12T09:12:41.337Z |
| 1 | 3 | 1 | recovery | — | 用户明确选择继续验证：保留当前候选与已有检查，修正的独立Temp脚本补齐A5；此前失败为验证输入JSON拼写错误与追加检查限额，不是产品缺陷。 | 2026-11-12T09:18:45.476Z |
| 1 | 3 | 2 | pass | — | 通过：独立只读核验全部A1-A10；原失败检查ID parser-review-independent经正式refreshChecks刷新后完整passed，六项当前Runtime收据全部passed，inputBefore=inputAfter、snapshot complete=true。VerifiedAt=2026-11-12T09:47:59.777Z | 2026-11-12T09:48:02.410Z |



## 结论

通过：独立只读核验全部A1-A10；原失败检查ID parser-review-independent经正式refreshChecks刷新后完整passed，六项当前Runtime收据全部passed，inputBefore=inputAfter、snapshot complete=true。VerifiedAt=2026-11-12T09:47:59.777Z
