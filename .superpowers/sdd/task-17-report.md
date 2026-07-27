# Task 17 Scratch Report

## 状态

已完成 Task 17 的 `v0.1.163` merge、Review Round 1/2 修复与限定收尾；完整门禁明确留给 Task 18，Sol 最终复审待协调者恢复既有会话。

## 起点与上游

- 起点：`b7b7bba6952460bb7cc38f1d41a0de95c449bcb8`。
- tag：`v0.1.163^{}` = `d0bdd7e771636a8d315f542cafd39484f39bd60c`。
- merge commit：`02abe1574bf8044a1b180e62b002f58f9928d88f`。
- merge parents：`b7b7bba6952460bb7cc38f1d41a0de95c449bcb8`、`d0bdd7e771636a8d315f542cafd39484f39bd60c`。

## 冲突与融合

- 14 个文本冲突均手工融合，未使用整文件 `ours/theirs`。
- 关键融合：reasoning policy 与 auth-cache v18；quota/runtime metadata 与 advanced/layered/sticky/DB scheduler；Grok retryable/policy rejection；Anthropic/Responses 计费与 compact SSE；UsageFilters 请求取消和 username-first 展示。
- `VERSION` 保持 `0.1.159.6`。

## 生成修正

- `go generate ./ent` 修正了由本地并发字段导致的 Ent 运行时索引偏移。
- `max_reasoning_effort` 和 `reasoning_effort_mappings` 分别使用 group field 48 与 49；`GroupMutation.Fields()` 容量为 53。

## 验证

- focused service Go 测试通过：`go test -tags=unit ./internal/service -run '^(TestAPIKey|TestParseAnthropicSSEField|TestHandleResponses.*CompactSSEFormat|TestOpenAI)' -count=1`。
- 已保留本 session 的 focused handler/server 成功检查，未重跑。
- focused `UsageFilters` Vitest 通过（4 tests）；前端 build 通过。
- 提交前 tag/MERGE_HEAD peeled SHA、`unmerged=0`、cached diff-check、冲突 marker、VERSION 与生成 Ent 输出均已核验。

## 提交

- merge：`02abe1574bf8044a1b180e62b002f58f9928d88f`，`merge: upstream v0.1.163`。
- docs：`ed81151346e5d8577f2b09c939c107986829239c`，`docs: record v0.1.163 merge`；仅包含 build ledger。
- review fix：`c411927ece1e83a19e4d9d70434acfcc7eb97316`，`fix: preserve v0.1.163 scheduler and shutdown behavior`。
- review docs：`f974fb9410c5835a446d426a539ce32db1cce1dc`，`docs: record v0.1.163 merge review fix`；仅包含累计 build ledger 的 Review Round 1 证据。
- review round 2 fix：`73d25ba105fe83043fe3497490fa7ce1e56edd19`，`fix: close v0.1.163 background lifecycle gaps`。

## Review Round 1

- RED：global 与 model-specific runtime cooldown 都会让 default session sticky acquire blocked account；新增测试按预期失败。shutdown/cleanup 生命周期和 phase runner 测试先以缺少 helper 的编译失败 RED。
- Fix：default/layered sticky classifier 复用 `isOpenAIAccountRequestRuntimeBlocked` 并以 `(nil, false)` 保留 binding；server 在 graceful timeout 后 force-close、以 15 秒 hard deadline drain active handler 后才调用 cleanup；cleanup 按 producers -> usage -> quota -> billing -> Redis/Ent phases 执行，deadline 不再继续关闭 downstream infra。
- GREEN：新增 sticky、phase order/deadline、active-handler lifecycle 测试通过；受影响 scheduler/usage/billing 测试和 `cmd/server` compile 通过；Ent/Wire 生成通过，Wire 连续两次输出哈希一致。
- 限制：15 秒 hard deadline 后仍可能有不响应取消的 handler；此时记录并返回，不能声称零丢失。cleanup phase timeout 会保留 downstream infra，避免正在运行的 upstream phase 使用已关闭资源。
- 生成核验：主工作树执行 `go generate ./ent` 遇到 Windows `user-mapped section` 文件锁，已还原其三份部分写入的 Ent 文件；在短路径 detached worktree 对当前 `f974fb941` 连续两轮执行 Ent 与 Wire 生成，`git diff --exit-code -- ent cmd/server/wire_gen.go` 均为零。主工作树 `go generate ./cmd/server` 亦通过且无 diff。

## Review Round 2

- RED：新增 server admission late-entry、parallel cleanup error、实际 cleanup phase 顺序、ImageTaskService cancel/wait/reject、Deferred in-flight flush、cyber usage-worker routing、billing 无 detached goroutine，以及 layered sticky global/model cooldown 测试。首轮 server/service/handler 命令分别因 `CloseAdmission`、Image `Run/Shutdown`、Deferred context-aware stop 缺失而编译失败；layered 是 Round 1 已有行为的回归补齐。
- Fix：handler tracker 在 shutdown 前关闭 admission；parallel cleanup 使用 `errors.Join` 并在 phase error 时跳过 downstream。ImageTaskService 管理 goroutine/cancel/wait，Wire 在 producer phase 停止 ImageTaskService；AsyncImageHandler 透传 lifecycle context。cyber 后处理进入 UsageRecordWorkerPool；quota DB/notify 留在 usage task；Deferred final flush 在 usage 后、quota 前串行执行并接受 cleanup context。
- GREEN：`go test -tags=unit ./cmd/server -count=1`、限定 service 命令、限定 handler 命令均通过；`go generate ./ent`、`go generate ./cmd/server` 通过；`git diff --check`、unmerged=0、VERSION 零 diff 通过。
- 边界：未运行 Task 18 full gate、远程、push/tag/release/deploy；硬 deadline 内无法取消的 handler/image upstream 仍可能阻塞到 deadline，后续 infra 因此保留。最终 Sol review 由协调者恢复既有 `ses_05ce435e2ffeKaG1tASC6Efg88`，本 scratch 不宣称 PASS。

## 自审与 Concerns

- 171 个 merge 文件均在 merge commit；ledger/checkpoint/selectors 未混入该 commit。
- 受保护的 `.comet/subagent-progress.md`、`.comet/current-change.json`、`paseo.json` 未由本任务修改或暂存。
- broad Go 和完整 Vitest 被误启动但均未取得完成摘要，不计 PASS、未重跑。Task 18 必须运行完整门禁并复核其部分输出中的 warning/`AggregateError`。

## 预算外最终修复

- 基线：`0b265313e2d17a38e996444a35894cdde1e9a835`；代码提交：`0e69a1b2cc67dece06d244783c75a04390d23d7f`，`fix: finish v0.1.163 background drains`。
- RED：`go test ./internal/service -run TestFinalizePostUsageBillingWaitsForNotificationDelivery -count=1` 观察到 balance 与 quota usage task 在 SMTP 交付完成前返回；`go test ./internal/handler -run TestRecordCyberPolicyIfMarkedBillsBeforeBlockingModeration -count=1` 观察到阻塞 moderation 在 mandatory billing 前耗尽 worker deadline；`go test ./cmd/server -run TestProvideCleanupDrainsOpsErrorsBeforeEntTeardown -count=1` 观察到 Ent teardown 先于已入队 ops error drain。
- GREEN：余额和账号配额通知改为在 usage task 内完成并保留 panic recovery；Ops error worker 在 `usage-record-drain` 后、deferred/Redis/Ent 前 drain，Stop API 接受 cleanup context 并返回错误，worker 固定持有启动队列避免关闭后读 nil channel；cyber 的 forward-error mandatory billing 移至 moderation、session block、Ops 辅助工作之前；DB 持久化注释同步为实际同步路径。
- 验证：三项精确 GREEN 命令均通过；`go test ./internal/service -count=1`、`go test ./internal/handler -count=1`、`go test ./cmd/server -count=1` 通过；`go generate ./cmd/server` 通过并提交 `wire_gen.go`；`VERSION` 保持 `0.1.159.6`。
- 边界：未运行 Task 18/full gate，未触及 `v0.1.164`、远程、push/tag/release/deploy。cleanup phase 超时会跳过后续 teardown，但 `main.go` 在 handler hard deadline 后仍无条件调用 `app.Cleanup()`；因此只能保证有界停机取舍，不能声称下游资源必然持续保留。
