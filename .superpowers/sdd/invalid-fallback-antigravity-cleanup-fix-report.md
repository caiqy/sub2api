# Invalid Fallback Antigravity Cleanup Fix

## Scope

- Added real `/v1/messages` regressions for PromptTooLong invalid fallback through a ClaudeCodeOnly group to a mixed Antigravity account in a resolved Gemini group.
- Covered 429 and 503 long-delay smart retry paths with Anthropic Sticky enabled and disabled.
- Propagated the existing `ForwardGeminiOption` session data through Claude `Forward` and `ForwardHandle`.

## Evidence

- RED: restoring `groupID: 0` and `sessionHash: ""` caused both enabled regressions to fail with zero `DeleteSessionAccountID` calls.
- GREEN: the restored implementation deletes `groupID=62`, `sessionKey=gemini:invalid-fallback-session` when Anthropic Sticky is enabled and makes zero deletes when disabled.

## Risk

- The option remains variadic, so callers that do not supply session data retain zero-value behavior.
