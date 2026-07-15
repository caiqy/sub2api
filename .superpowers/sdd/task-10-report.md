# Task 10 Report

## Grok edit body lifecycle

- `SetEffectiveBytes(forwardBody)` 成功后立即清空局部 `forwardBody`；有效 handle 已独立持有内容，局部引用可供 GC 回收。
- 完整 handler 三轮复现仅在 Grok edit 发生 HeapAlloc 波动（约 20 MiB）。测试在阻塞 upstream 后连续执行两次 GC，再读取 HeapAlloc；不调整阈值或公共 API。
- 阻塞 upstream 的 HeapAlloc 测试在 `t.Cleanup` 中释放 upstream 并等待 handler 结束，断言失败也不会遗留 goroutine。

## 验证

- `go test -tags=unit ./internal/handler -run '^TestGrokMedia_TextIsReleasedBeforeBlockedUpstream$' -count=10`：PASS。
- `go test -tags=unit ./internal/handler -count=3`：PASS（263.791s）。
