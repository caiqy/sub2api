# Task 4 Report

## RED

- 隔离前：`go test -tags=unit ./internal/service -run '^TestOpenAIForwardCleansRequestBodyHandleWhenHTTPDoErrors$' -count=20` 通过。原测试按时间扫描共享 `os.TempDir()`，因此不能证明自身没有泄漏。
- 隔离后：绑定 `t.TempDir()` 中、阈值为 `1` 的 handle，并只检查该目录；定向运行稳定失败 20 次，目录均遗留一个 `sub2api-test-*` spool 文件。
- 生命周期回归：`TestOpenAIForwardPreservesBoundRequestBodyHandleWhenHTTPDoErrors` 首次 `Forward` 返回 HTTP Do 错误后读取 bound handle，修复前稳定失败：`request body handle has been cleaned up`。
- Owner 路径：handler 的 `requestBodyCoordinator` 在 handler 返回时统一 `Cleanup`；`Forward` -> `buildUpstreamRequestWithSourceBody` -> `openAIRequestBodyHandleForBytes` 却把匹配的 bound handle 标记为 `owned=true`，使每次 `Forward` 的 `closeOpenAIRequestBody` 提前清理 coordinator 持有的 handle。

## GREEN

- `openAIRequestBodyHandleForBytes` 对匹配的绑定 handle 返回 `owned=false`，恢复 handler coordinator 的跨 failover 所有权；仅本次 `Forward` 新建的 handle 仍在请求关闭时清理。
- 回归只使用其 `t.TempDir()`：首次错误后读取 bound handle，第二次 `Forward` 的上游实际读取同一 body；调用方 `CleanupRequestBodyHandle(handle)` 后才断言目录为空。

## Matrix

| Command | Result |
| --- | --- |
| `go test -tags=unit ./internal/service -run '^TestOpenAIForwardPreservesBoundRequestBodyHandleWhenHTTPDoErrors$' -count=20` | PASS |
| `go test -tags=unit ./internal/service -count=3` | PASS |
| `go test -tags=unit ./internal/handler -run '^TestMediaJSONHandlers_PreserveContentDerivedSchedulerAffinity/openai_images$' -count=1` | PASS |
| `go test -tags=unit ./... -count=3` | FAIL；见下方准确归因 |

## Production Change

有。仅修改 `backend/internal/service/openai_gateway_request_body.go`：匹配的已绑定 handle 保持 borrowed，不进入单次 upstream request 的 cleanup 分支。

## Risk Signals And Concerns

- `openai_images` 审查复现当前通过，先前报告的该失败无法在当前工作区复现；未将其归因为本任务。
- 绑定 handle 必须由 handler coordinator（或直接调用方）清理；直接调用 `Forward` 的测试已显式 cleanup。若未来出现不经 coordinator 绑定且长期存活的 handle，调用方仍须负责释放。
- 全套三轮仅失败于 `internal/handler` 的 `TestGrokMedia_TextIsReleasedBeforeBlockedUpstream/edit`（20MB 文本保留阈值，174150984 > 165839216）和 `internal/handler/admin` 的 `TestAdminUsageStatsRequestTypePriority`（`usage_handler_request_type_test.go:154` 期望非 nil）。两者各自 `-count=1` 均通过，说明失败只在全套并发包矩阵中出现；未发现其经过本任务修改的 `OpenAIGatewayService.Forward` body-handle 路径，故未扩大修复范围。
