# Fallback Antigravity Cleanup

## Scope

- Added handler-level coverage for a ClaudeCode-only Anthropic group that falls back to a Gemini group and selects a mixed-scheduling Antigravity account.
- The upstream returns a 429 with a 15-second Gemini rate-limit delay. The test verifies smart-retry deletes the Gemini fallback sticky entry, never the original Anthropic group entry.
- Extended the test-only Gemini sticky cache stub to record delete calls.

## TDD Evidence

- GREEN with the current handler: `go test ./internal/handler -run TestGatewayHandlerMessages_ClaudeCodeFallbackMixedAntigravitySmartRetryClearsResolvedGeminiStickySession -count=1`
- RED after temporarily replacing `stickyGroupID` with `apiKey.GroupID` in `WithForwardGeminiSession`: assertion failed with `expected: 2`, `actual: 1`.
- Restored `stickyGroupID`; the focused regression passed again.

## Production Change

None. The existing handler already passes the resolved fallback group ID and Gemini-prefixed session key to `WithForwardGeminiSession`.
