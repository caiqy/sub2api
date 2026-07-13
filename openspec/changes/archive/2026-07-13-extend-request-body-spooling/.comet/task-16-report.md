# Task 16 Report

## Result

- OpenAI Images、Grok Images 和 Grok Videos 的 JSON 请求恢复使用 `GenerateSessionHash`，因此 prompt-only 请求获得内容派生的非空 scheduler session hash。
- 内容派生补充顶层媒体 `prompt`，不同 prompt 不再共享同一 session hash。
- `session_id` 与 `prompt_cache_key` 仍优先于内容派生；同一请求在 retry/failover 的全部 scheduler lookup 使用同一 hash。
- Multipart 继续使用已冻结 fallback seed 和 `GenerateSessionHashWithFallback`，未改变 multipart affinity。
- `GenerateSessionHashWithFallback` 注释明确其只处理显式信号与 caller 提供的 fallback，不执行内容派生。

## Verification

- RED: `go test ./internal/handler -run TestMediaJSONHandlers_PreserveContentDerivedSchedulerAffinity -count=1` observed empty Grok JSON scheduler sessions before the handler switch; after the switch it exposed the missing top-level `prompt` content seed.
- GREEN: `go test ./internal/handler -run TestMediaJSONHandlers_PreserveContentDerivedSchedulerAffinity -count=1`
- `go test ./internal/handler ./internal/service -count=1` using temporary `D:\cache\sub2api-task16-*` caches.
