# Task 4 Report

## RED

- 隔离前：`go test -tags=unit ./internal/service -run '^TestOpenAIForwardCleansRequestBodyHandleWhenHTTPDoErrors$' -count=20` 通过。原测试按时间扫描共享 `os.TempDir()`，因此不能证明自身没有泄漏。
- 隔离后：绑定 `t.TempDir()` 中、阈值为 `1` 的 handle，并只检查该目录；定向运行稳定失败 20 次，目录均遗留一个 `sub2api-test-*` spool 文件。
- Owner 路径：`Forward` -> `buildUpstreamRequestWithSourceBody` -> `openAIRequestBodyHandleForBytes`。匹配已绑定 handle 时返回 `owned=false`，使 `Forward` 的 `closeOpenAIRequestBody` 只关闭 reader，未调用 `CleanupRequestBodyHandle`。

## GREEN

- `openAIRequestBodyHandleForBytes` 对匹配的绑定 handle 返回 `owned=true`；现有 HTTP Do 后的 `closeOpenAIRequestBody` 清理该 handle。
- 测试改为关闭上游 request body 后读取专用 `spoolDir`，断言为空；删除共享系统 temp 扫描 helper。

## Matrix

| Command | Result |
| --- | --- |
| `go test -tags=unit ./internal/service -run '^TestOpenAIForwardCleansRequestBodyHandleWhenHTTPDoErrors$' -count=20` | PASS |
| `go test -tags=unit ./internal/service -count=3` | PASS |
| `go test -tags=unit ./... -count=3` | FAIL（非目标既有失败） |

## Production Change

有。仅修改 `backend/internal/service/openai_gateway_request_body.go`：让匹配的已绑定 handle 进入现有 owned cleanup 分支。

## Risk Signals And Concerns

- 全套矩阵仍在 `internal/handler` 的 `TestMediaJSONHandlers_PreserveContentDerivedSchedulerAffinity/openai_images` 和 `internal/handler/admin` 的 `TestAdminUsageStatsRequestTypePriority` 失败；两者不在本任务文件或调用路径中。
- 绑定 handle 现在会在一次 `Forward` 的 HTTP Do 后清理，后续若需要跨独立 `Forward` 调用复用同一 handle，必须重新绑定/创建；当前服务包三轮矩阵通过。
