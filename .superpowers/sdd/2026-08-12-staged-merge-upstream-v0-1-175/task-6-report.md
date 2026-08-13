# Task 6 Report - 能力簇 3：OpenAI 错误与统一审计

## 状态

**GREEN / tests-only review-fix round 2**。生产修复仍为 `58d67b73a919bcec3eb71400bf5589f4943b0a49`；本轮只加强 handler/service 测试断言与证据表述，不改生产代码。brief 精确全筛选的 service 和 handler 包均为 exit 0。

## RED / GREEN

- RED：`TestOpenAIGatewayService_OAuthPassthrough_ReusesFingerprintForSameAccountRetry` 首次失败，唯一语义差异为同一账号重试时随机 `client_metadata.turn_id` 漂移。
- RED：初版仅按 account ID 缓存，新增不同 body 断言失败，证明会跨不同请求复用 turn ID。
- GREEN：缓存键为 account ID + post-normalization request-body SHA-256；同 body 重试完整 JSON 一致，不同 body 生成新 turn ID。对应单测 GREEN。
- 非行为 RED：handler 全包先在 `openai_gateway_handler_test.go:2239` 缺第 10 参编译失败；补 `nil` resolver 后 handler 相关定向测试可运行。
- 本轮 RED：重新运行五个 handler 目标测试，得到 **5 个顶层失败、6 条 `FAIL` 行**（capacity 测试含失败子测试）。原因为跨最终账号比较完整 OAuth body、确定性 400 期待 502/`upstream_error`，以及 cyber 顺序测试的夹具未显式固定其 scope/ban-count 前置条件。
- 本轮 GREEN：切号后只比较非 fingerprint 字段，并断言 `client_metadata` 不同；OAuth capacity 测试仍要求账号 71 的四次 body 完全相同。真实 service 覆盖使用第二个账号 ID 124，确认同 body 在账号 123 重用 ID、切到 124 后仅 fingerprint 改变，且不同 body 不复用 turn ID。
- 本轮 GREEN：images 的两个确定性上游 400 都验证 client-facing `400 invalid_request_error`，并继续验证 failed usage log；multipart edit 继续验证 binary 脱敏 metadata snapshot。
- 本轮 GREEN：cyber 顺序测试显式 scope 到 group 1/model `gpt-5`，并启用 `cyber_policy_exclude_from_ban_count` 以隔离无关的 violation-count 依赖，验证 billing 在阻塞 `CreateLog` 前完成。
- 本轮 GREEN：跨账号 OAuth replay 不再以随机 `turn_id` 可造成误判的完整 `client_metadata` 不等作为证据；逐字段检查 installation/session/thread/window 均存在且至少一项账号派生稳定字段不同，同时保持非 fingerprint payload 相同断言。
- 本轮 GREEN：multipart image edit 除 400 状态码外，也验证 client-facing `error.type=invalid_request_error`。

## Commit / Paths

- Commit：`58d67b73a919bcec3eb71400bf5589f4943b0a49`，消息 `fix: preserve fork error classification and audit semantics after v0.1.175 merge`。
- 本轮提交：`6e2e09028`，`test: align cluster 3 handler assertions with merged contracts`。
- 本轮提交：`test: strengthen cross-account fingerprint isolation assertions`。
- 本轮 ExpectedPaths：
  - `backend/internal/handler/openai_gateway_credential_failover_loop_test.go`
  - `backend/internal/handler/openai_gateway_request_body_retention_test.go`
  - `backend/internal/handler/openai_gateway_failed_usage_test.go`
  - `backend/internal/service/openai_oauth_passthrough_test.go`
  - `.superpowers/sdd/2026-08-12-staged-merge-upstream-v0-1-175/task-6-report.md`
  - `docs/openspec/changes/staged-merge-upstream-v0-1-175/.comet/evidence/build/06-cluster3.md`
- Evidence（gitignored）：`06-cluster3.md`、`06-cluster3-list.txt`、`06-cluster3-*.log`。

## 不变量证据

| 不变量 | 证据 | 结论 |
|---|---|---|
| 确定性 400 不改写 502 | `TestHandleErrorResponse_Deterministic400*` GREEN；images 的两个旧期望 502 handler 测试因此失败 | PASS |
| HTML 403 不处罚账号 | `TestHandleUpstreamError_OpenAIHTML403*` GREEN | PASS |
| OAuth image stream / 空 completed / TTFT | `TestOpenAIGatewayServiceForwardImages_OAuthStreaming*`、`TestImagesOAuthNonStreaming*`、`TestOpenAIResponsesCompletedEventIsEmpty`、`TestOpenAIResponsesTTFT*` GREEN | PASS |
| capacity/pool 有界重试 | `TestSameAccountRetryDelayFor` 证明 500ms 指数且 <=8s；`TestHandleFailoverError_SameAccountRetry` 覆盖默认 3、0、不提前切号、耗尽临时排除；pool 401/403/5xx/SSE 定向测试 GREEN | PASS |
| 同账号共享 replay 身份 | 新回归测试 RED->GREEN：同一账号同 body 不漂移；真实第二账号切换后非 fingerprint payload 相同，且逐字段确认至少一个账号派生稳定 fingerprint 字段隔离；不同 body 不复用 | PASS |
| 模型级阻断不扩大 | `TestOpenAIPoolModeTempRule_StopsSameAccountRetryAndIsolatesBlockToModel` 已在 brief 全筛选 service GREEN | PASS |
| 统一 security/prompt audit 去重 | `TestCachesSecurityAuditCompletionSkipsWebSocketStages`、`TestRunSecurityAudit*`、`TestPromptAuditGatePrecedesAccountBillingAndUpstreamSideEffects` GREEN | PASS |
| cyber scope / risk-control | `TestRecordCyberPolicyEvent_*` 与 `TestOpenAIWSCyberPolicy*` GREEN；事件按 group/model scope 过滤 | PASS |
| 请求体生命周期和 client contract | OAuth same-account replay RED->GREEN；spool/replay tests and deterministic 400 client tests GREEN | PASS |

## 精确测试

```text
go -C backend test -tags=unit -list 'Retry|Failover|Audit|Moderation|TTFT|Capacity|Pool|403|400|Cyber|Image|WsTurn' ./internal/service/ ./internal/handler/
selected=1189

go -C backend test -tags=unit -v -count=1 -run 'Retry|Failover|Audit|Moderation|TTFT|Capacity|Pool|403|400|Cyber|Image|WsTurn' ./internal/service/ ./internal/handler/
selected=1189
PASS lines=1186
SKIP lines=0
service: exit 0
handler: exit 0

review-fix round 2 rerun:
five focused tests: service and handler exit 0
brief-wide service: exit 0 (67.360s)
brief-wide handler: exit 0 (74.126s)

go -C backend build ./...
exit 0
```

## 风险信号 / gaps

1. 先前证据中“空 moderation config 无 scope，所以不创建 event”的归因错误：默认 config 是 `all_groups=true` 且 model filter 为 `all`。本轮以显式 scope 和 exclude-from-ban-count 固定顺序测试前置条件，不把默认值作为隐式夹具行为。
2. Docker-backed integration 未运行；本任务未执行 merge/amend/reset/push/tag/release/deploy，未修改 plan/tasks/spec/.comet state/progress。
