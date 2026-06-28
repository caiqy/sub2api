# Anthropic Messages Conversation Rendering Design

## Goal

Support Claude / Anthropic Messages API request and streaming response payloads in the admin usage-detail conversation renderer. The parser should follow the existing webgui-style `ConversationMessage -> parts` model, avoid Raw Response fallback when Anthropic content is reconstructable, and preserve the current OpenAI Chat Completions and OpenAI Responses behavior.

## Context

The existing conversation parser supports:

- `openai-chat`: request/response payloads based on `messages[]` and `choices[]`.
- `openai-responses`: request/response payloads based on `input[]`, `output[]`, and `output_text`.
- SSE reconstruction for OpenAI Chat Completions and OpenAI Responses streams.

Claude / Anthropic Messages payloads look superficially similar to OpenAI Chat because they also contain `messages[]`, but their content blocks and streaming events differ:

- Top-level `system[]` carries system prompt blocks.
- Assistant tool calls are inline content blocks: `{ type: "tool_use", id, name, input }`.
- Tool results are user content blocks: `{ type: "tool_result", tool_use_id, content }`.
- Thinking is a native content block: `{ type: "thinking", thinking }`.
- Streaming uses block-indexed SSE events: `message_start`, `content_block_start`, `content_block_delta`, `content_block_stop`, `message_delta`, and `message_stop`.

## Format Detection

Add a new conversation format:

```ts
type ConversationFormat = 'openai-chat' | 'openai-responses' | 'anthropic-messages' | 'unknown'
```

Detection should prefer Anthropic before OpenAI Chat so Claude request bodies are not misclassified simply because they contain `messages[]`.

Anthropic request signals:

- `messages` is an array, and at least one of these is true:
  - top-level `system` exists,
  - top-level `thinking` exists,
  - top-level `max_tokens` exists,
  - any message content block has `type: "tool_use"`, `type: "tool_result"`, or `type: "thinking"`.

Anthropic response signals:

- Reconstructed response object has `type: "message"` and `content[]` with Anthropic block types.
- Raw SSE contains Anthropic event names and is reconstructed by the Anthropic SSE branch.

## SSE Reconstruction

Extend `parseSSEBody()` with an Anthropic stream branch. It should scan `data:` lines and reconstruct a message envelope plus ordered content blocks.

Event handling:

- `message_start`: store base message metadata such as `id`, `model`, `role`, `usage`, `stop_reason`, and `stop_sequence`.
- `content_block_start`: initialize `blocks[index]` from `content_block`.
- `content_block_delta`: append delta text by block index:
  - `text_delta.text` appends to a `text` block.
  - `thinking_delta.thinking` appends to a `thinking` block.
  - unknown deltas are stored as metadata/raw material for fallback.
- `content_block_stop`: mark the block as complete if needed for metadata.
- `message_delta`: merge `delta.stop_reason`, `delta.stop_sequence`, and usage updates into the message metadata.
- `message_stop`: confirms the stream ended; no extra content is required.

The returned value should be a normalized Anthropic message object:

```ts
{
  type: 'message',
  role: 'assistant',
  content: [
    { type: 'text', text: '...' },
    { type: 'thinking', thinking: '...' },
    { type: 'text', text: '...' }
  ],
  model,
  usage,
  stop_reason,
  stop_sequence
}
```

If Anthropic SSE events are present but no usable content is reconstructed, parsing should return the envelope if available; otherwise `null` so the normal Raw Response fallback remains visible.

## Anthropic Parser

Add `parseAnthropic()` beside `parseChat()` and `parseResponses()`. It should reuse the existing generic helpers instead of creating separate renderer models.

### Request Parsing

- Top-level `system`:
  - Convert text blocks to the existing `systemPrompt` output via `pushSystemPrompt('system', request.value.system)`.
  - Preserve cache-related fields in message metadata only if needed for debugging; do not render them as text.
- `messages[]`:
  - `role: "user"` and `role: "assistant"` create normal conversation messages.
  - `text` blocks become `text` parts.
  - `tool_use` blocks call `pushToolCall()` with `id` mapped to `callId`, `name` mapped to `tool`, and `input` mapped to `state.input`.
  - `tool_result` blocks call `pushToolResult()` with `tool_use_id` mapped to `callId` and `content` mapped to `state.output`.
  - `thinking` blocks call `pushReasoningPart()` and remain collapsed by default.
  - Unknown blocks become collapsed `raw` parts.

Tool result matching should work across the entire request sequence, matching Claude's alternating assistant-tool-use and user-tool-result turns.

### Response Parsing

For the reconstructed Anthropic message:

- `text` blocks become assistant `text` parts.
- `thinking` blocks become assistant `reasoning` parts, collapsed by default.
- Empty text blocks whose trimmed text is empty are skipped to avoid blank UI rows.
- Unknown blocks become collapsed `raw` parts.
- Message metadata preserves `id`, `model`, `usage`, `stop_reason`, and `stop_sequence`.

The parser should append response blocks into a single assistant message, matching the current OpenAI Responses behavior for streamed assistant output.

## Rendering Behavior

No new visual component is required. Anthropic content should use the existing part components:

- Text: existing markdown-safe text part.
- Thinking: existing reasoning part.
- Tool use/result: existing tool part with input and output merged by call ID.
- Unknown content: existing collapsed raw part.

This keeps the UI consistent across OpenAI and Claude payloads.

## Error Handling and Fallbacks

- If request detection succeeds but no request message can be parsed, show Raw Request.
- If response detection succeeds but no response content can be parsed, show Raw Response.
- If a `tool_result` has no matching `tool_use`, render an orphan completed tool part with output only, matching existing behavior.
- If unknown Anthropic content block types appear, render those blocks as collapsed raw parts rather than dropping them.
- Existing OpenAI tests must continue to pass.

## Tests

Add focused unit tests in `parseConversationPayload.spec.ts`:

1. Anthropic request with top-level `system`, user text, assistant `tool_use`, and user `tool_result` renders a system prompt, text part, and merged tool part.
2. Anthropic SSE response with `message_start`, `thinking` block, and `text` block reconstructs an assistant message with reasoning plus final text.
3. Empty Anthropic text blocks are skipped.
4. Unknown Anthropic content blocks render as collapsed raw parts.
5. OpenAI Chat and OpenAI Responses parsing remains unchanged through existing tests.

## Out of Scope

- Adding new UI components or visual styles.
- Displaying cache control fields as user-visible text.
- Supporting non-text Anthropic media blocks beyond raw fallback unless real examples are provided.
- Changing the existing OpenAI parsing or rendering semantics.

## Acceptance Criteria

- Claude `input.txt` renders structured conversation messages instead of Raw Request.
- Claude `result.txt` SSE renders assistant thinking and final text instead of Raw Response.
- Tool calls and tool results merge into the same tool part when IDs match.
- Empty leading text blocks do not create visible blank messages.
- Current OpenAI parser tests still pass.
