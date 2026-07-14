# Invalid Fallback Sticky Report

- Added `TestGatewayHandler_MessagesPromptTooLongFallbackResolvesClaudeCodeOnlyGroup`.
- Non-Claude Code: invalid-request fallback resolves through the intermediate ClaudeCodeOnly group to final Gemini; scheduler, account, sticky cache reads/writes, `gemini:` session key, and binding use the final group/account.
- Claude Code parity: the same intermediate group remains selected and does not continue to final Gemini.
- Controlled RED: replacing the post-invalid-fallback `resolveStickyRoute` call with the intermediate group made the test fail because the sticky key was no longer Gemini-prefixed. The existing shared resolver was restored; no production change was required.

Verification:

```text
go test ./internal/handler -run 'TestGatewayHandler_MessagesPromptTooLong|TestGatewayHandlerMessages_ClaudeCodeFallback|TestGatewayHandlerResolveStickyRoute_PreservesClaudeCodeRestrictionErrors' -count=1
ok github.com/Wei-Shaw/sub2api/internal/handler
```
