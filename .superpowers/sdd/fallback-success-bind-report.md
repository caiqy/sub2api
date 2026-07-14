# Fallback Success Sticky Bind

- RED: temporary use of the original API key group/platform made the regression test write an Anthropic key for group `1`; with Gemini Sticky disabled it still touched cache.
- GREEN: restored `ResolveGatewayGroup` routing. A successful non-Claude-Code `POST /v1/messages` through the Gemini fallback binds the selected account under the fallback group and a `gemini:` session key.
- Disabled case: Gemini Sticky disabled with Anthropic Sticky enabled performs zero cache reads and writes.

Verification:

```text
go test ./internal/handler -run '^(TestGatewayHandlerMessages_ClaudeCodeFallbackUsesResolvedGeminiStickyBoundary|TestGatewayHandlerMessages_ClaudeCodeFallbackSuccessBindsResolvedGeminiStickySession|TestGatewayHandlerResolveStickyRoute_PreservesClaudeCodeRestrictionErrors)$' -count=1
ok github.com/Wei-Shaw/sub2api/internal/handler
```
