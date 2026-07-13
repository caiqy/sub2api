# Task 15 Report

## Result

- Multipart text values are released after synchronous parsing, moderation, hashing, and replay-handle preparation, before image, user, group, and upstream waits.
- File headers remain available for cleanup and source validation. API-key model remapping rewrites the replayable effective handle without restoring `form.Value`.
- OpenAI OAuth receives a prebuilt response JSON handle; Grok releases parsed text before forwarding. Multipart snapshots omit prompt text.
- OpenAI Images freezes its moderation payload before release, so synchronous moderation still receives the complete prompt and URL images; that payload is cleared before any slot or upstream wait.
- OpenAI Images freezes an opaque sticky-session seed before release. OAuth retry reuses it and refuses to rebuild from released parsed text when no prebuilt body handle is bound.
- Blocked-upstream GC coverage now includes OpenAI OAuth plus table-driven Grok generate, edit, and video paths. Upstreams retain only hash and size; each case proves no 20MB text is retained and cleanup removes spools.
- Frozen multipart seeds now use `GenerateSessionHashWithFallback`: explicit header and JSON `prompt_cache_key` still win, while a raw SHA seed becomes a stable non-empty scheduler hash without JSON parsing. OpenAI OAuth failover observes the actual scheduler sticky lookup: every retry keeps one hash and different prompts produce different hashes. Grok multipart uses its effective-body hash through the same path; JSON and video request-ID behavior remain unchanged.

## Verification

- `go test ./internal/handler -count=1`
- `go test ./internal/service -count=1`
- The 10MB/20MB multipart text matrix succeeds, 20MB+1 returns 413 with cleanup, and a blocked upstream GC regression confirms no retained 20MB request text.
- `go test ./internal/service ./internal/handler -run 'TestOpenAILegacyBuilderCallersCloseRequestBodyWhenHTTPDoErrors|TestOpenAIGatewayServiceForwardImages_OAuth|TestOpenAIImagesRequest_FreezeStickySessionSeedSurvivesRelease|TestOpenAIImages_ContentModerationUsesFrozenPayloadBeforeRelease|TestOpenAIImages_OAuthTextIsReleasedBeforeBlockedUpstream|TestGrokMedia_TextIsReleasedBeforeBlockedUpstream|TestOpenAIGatewayHandlerImages_ServerErrorFailsOverAndReturnsClearErrorWhenExhausted' -count=1`
- `go test ./internal/handler ./internal/service -count=1` (using a temporary `D:\cache\go-build` cache).
- `go test -tags unit ./internal/handler -run TestOpenAIGatewayHandlerImages_ServerErrorFailsOverAndReturnsClearErrorWhenExhausted -count=1` (scheduler-boundary multipart OAuth regression).
