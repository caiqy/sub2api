# 本地门禁新增最小修复项

- admin `TestAdminUsageStatsRequestTypePriority`：每次测试重置进程级 `usageStatsCache`，避免 `-count=3` 的同秒缓存命中跳过 fixture。
- Grok `TestGrokMedia_TextIsReleasedBeforeBlockedUpstream/edit`：`SetEffectiveBytes` 成功后释放 `forwardBody`，并以 `t.Cleanup` 解锁/等待阻塞 upstream goroutine。

两项均由 `go test -tags=unit ./... -count=3` 阻塞发现，不改变产品缓存语义、Grok API 或 lint 配置。
