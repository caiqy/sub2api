# Invalid Fallback Forwarding Fix

## Root Cause

`GatewayHandler.Messages` selected its initial forwarding branch before the invalid-request fallback. When the fallback resolved to a Gemini group, the existing non-Gemini loop still called `GatewayService.Forward`, sending the Claude `POST /v1/messages` payload upstream.

## TDD Evidence

- RED: `go test ./internal/handler -run '^TestGatewayHandler_MessagesPromptTooLongFallbackResolvesClaudeCodeOnlyGroup$' -count=1` failed because the second request path was `/v1/messages`, not `/v1beta/models/claude-opus-4-6:generateContent`.
- GREEN: the same test passed after routing the selected Gemini account through `GeminiMessagesCompatService.Forward`.

## Scope

The attempt-local branch preserves Antigravity handling and sends all selected Gemini account types through the existing compat service. Channel mapping, `ParsedRequest.GroupID`, Bedrock routing, sticky route resolution, billing identity, usage recording, retry behavior, and the Antigravity cleanup toggle were not changed.

## Verification

- `go test ./internal/handler -count=1`
- `go test ./internal/service -run '^TestGeminiMessagesCompatServiceForward_' -count=1`
- `go build ./...`
- `git diff --check`
