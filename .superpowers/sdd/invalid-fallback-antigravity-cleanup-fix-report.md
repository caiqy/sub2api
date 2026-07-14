# Invalid Fallback Antigravity Cleanup Fix

## Scope

- Added real `/v1/messages` regressions for PromptTooLong invalid fallback through a ClaudeCodeOnly group to a mixed Antigravity account in a resolved Gemini group.
- Covered 429 and 503 long-delay smart retry paths with Anthropic Sticky enabled and disabled.
- Propagated the existing `ForwardGeminiOption` session data through Claude `Forward` and `ForwardHandle`.
- Propagated the same session data through the signature and budget rectifier retry loops.

## Evidence

- RED: restoring `groupID: 0` and `sessionHash: ""` caused both enabled regressions to fail with zero `DeleteSessionAccountID` calls.
- GREEN: the restored implementation deletes `groupID=62`, `sessionKey=gemini:invalid-fallback-session` when Anthropic Sticky is enabled and makes zero deletes when disabled.
- RED/GREEN: the budget rectifier's follow-up 429 deletes `groupID=79`, `sessionHash=gemini:budget-rectifier` only after its retry loop receives the option data.

## Risk

- The option remains variadic, so callers that do not supply session data retain zero-value behavior.
- `ForwardGeminiOption` intentionally remains named for its original protocol path: renaming it is a broader Minor cleanup and would expand call-site churn without changing this fix.
