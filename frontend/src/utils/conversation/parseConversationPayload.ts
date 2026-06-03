import { createConversationNodeId, formatRawValue, parseJsonValue } from './format'
import type {
  ConversationContentPart,
  ConversationFlow,
  ConversationFormat,
  ConversationInjectionPart,
  ConversationMessage,
  ConversationMessageRole,
  ConversationPart,
  ConversationReasoningPart,
  ConversationToolPart,
  ParseConversationPayloadInput,
} from './types'

type JsonParseResult = {
  value: unknown | null
  raw: string | null
  hasBody: boolean
  invalid: boolean
}

export const INJECTION_TAG_WHITELIST = ['EXTREMELY_IMPORTANT', 'EXTREMELY-IMPORTANT', 'SUBAGENT-STOP', 'system-reminder', 'reminder', 'important']

type SplitInjectionPart = { type: 'text' | 'injection'; text: string; tag?: string }
type UserContentPart = ConversationContentPart | Omit<ConversationInjectionPart, 'id'>

const MAX_INJECTION_EXTRACTIONS = 32

const escapeRegExp = (value: string): string => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

const splitTextIntoInjectionParts = (text: string): SplitInjectionPart[] => {
  const parts: SplitInjectionPart[] = []
  const tagPattern = INJECTION_TAG_WHITELIST.map(escapeRegExp).join('|')
  const openTagRegex = new RegExp(`<(${tagPattern})\\s*>`, 'i')
  let cursor = 0
  let searchStart = 0
  let extractions = 0

  while (extractions < MAX_INJECTION_EXTRACTIONS && searchStart < text.length) {
    const openMatch = openTagRegex.exec(text.slice(searchStart))
    if (!openMatch || openMatch.index === undefined) break

    const openStart = searchStart + openMatch.index
    const openEnd = openStart + openMatch[0].length
    const tag = openMatch[1]
    const closeTagRegex = new RegExp(`</${escapeRegExp(tag)}\\s*>`, 'i')
    const closeMatch = closeTagRegex.exec(text.slice(openEnd))

    if (!closeMatch || closeMatch.index === undefined) {
      searchStart = openEnd
      continue
    }

    const precedingText = text.slice(cursor, openStart)
    if (precedingText.trim()) parts.push({ type: 'text', text: precedingText })

    const closeEnd = openEnd + closeMatch.index + closeMatch[0].length
    parts.push({ type: 'injection', text: text.slice(openStart, closeEnd), tag })
    cursor = closeEnd
    searchStart = closeEnd
    extractions++
  }

  const rest = text.slice(cursor)
  if (rest.trim()) parts.push({ type: 'text', text: rest })
  return parts
}

const splitUserContentParts = (contentParts: ConversationContentPart[]): UserContentPart[] => {
  const expandedParts: UserContentPart[] = []

  for (const part of contentParts) {
    if (part.type !== 'text') {
      expandedParts.push(part)
      continue
    }

    const splits = splitTextIntoInjectionParts(part.text)
    if (splits.length === 0) {
      expandedParts.push(part)
      continue
    }

    if (splits.length === 1 && splits[0].type === 'text') {
      expandedParts.push(part)
      continue
    }

    for (const split of splits) {
      if (split.type === 'injection') {
        expandedParts.push({ type: 'injection', tag: split.tag!, text: split.text, defaultCollapsed: true })
        continue
      }

      expandedParts.push({ type: 'text', text: split.text })
    }
  }

  return expandedParts
}

const mergeConsecutiveReasoning = (message: ConversationMessage): void => {
  const mergedParts: ConversationPart[] = []
  let reasoningRun: ConversationReasoningPart[] = []

  const flushReasoningRun = () => {
    if (reasoningRun.length === 0) return

    const lastPart = reasoningRun[reasoningRun.length - 1]
    const metadata = { ...lastPart.metadata, segments: reasoningRun.length }

    if (reasoningRun.length === 1) {
      reasoningRun[0].metadata = metadata
      mergedParts.push(reasoningRun[0])
    } else {
      mergedParts.push({
        id: reasoningRun[0].id,
        type: 'reasoning',
        text: reasoningRun.map((part) => part.text).join('\n\n'),
        defaultCollapsed: true,
        metadata,
      })
    }

    reasoningRun = []
  }

  for (const part of message.parts) {
    if (part.type === 'reasoning') {
      reasoningRun.push(part)
      continue
    }

    flushReasoningRun()
    mergedParts.push(part)
  }

  flushReasoningRun()
  message.parts.splice(0, message.parts.length, ...mergedParts)
}

const attachToolOutputSize = (part: ConversationToolPart): void => {
  if (part.state.error) return
  if (part.state.output === undefined) return

  const formatted = formatRawValue(part.state.output)
  if (!formatted) return

  part.state.outputSize = {
    bytes: new TextEncoder().encode(formatted).length,
    lines: formatted.split('\n').length,
  }
}

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

const isMessageRole = (value: unknown): value is ConversationMessageRole => {
  return value === 'user' || value === 'assistant' || value === 'system' || value === 'developer'
}

const hasAnthropicContentBlock = (content: unknown): boolean => {
  if (!Array.isArray(content)) return false
  return content.some((part) => {
    if (!isRecord(part)) return false
    return part.type === 'tool_use' || part.type === 'tool_result' || part.type === 'thinking'
  })
}

const isAnthropicMessageObject = (value: unknown): value is Record<string, unknown> => {
  if (!isRecord(value)) return false
  if (value.type !== 'message') return false
  if (!Array.isArray(value.content)) return false
  return value.content.some((part) => {
    if (!isRecord(part)) return false
    return part.type === 'text' || part.type === 'tool_use' || part.type === 'tool_result' || part.type === 'thinking'
  })
}

const isAnthropicMessagesRequest = (value: unknown): value is Record<string, unknown> => {
  if (!isRecord(value) || !Array.isArray(value.messages)) return false
  if (Object.prototype.hasOwnProperty.call(value, 'input')) return false

  const hasTopLevelSignal = Object.prototype.hasOwnProperty.call(value, 'system')
    || Object.prototype.hasOwnProperty.call(value, 'thinking')
    || Object.prototype.hasOwnProperty.call(value, 'max_tokens')
  const hasBlockSignal = value.messages.some((message) => isRecord(message) && hasAnthropicContentBlock(message.content))

  return hasTopLevelSignal || hasBlockSignal
}

/**
 * Detect and parse SSE (Server-Sent Events) streaming response body.
 *
 * Strategy for OpenAI Responses API streams:
 * - Collect the terminal `response.done` / `response.completed` envelope (carries id/model/usage/status)
 * - Collect every `response.output_item.done` item (these carry the actual message/tool content)
 * - If the terminal envelope already has a non-empty `output` array, return it as-is
 * - Otherwise merge the collected items into the envelope's `output` (some servers omit `output`
 *   on the terminal event when streaming is enabled, e.g. `store: false`)
 * - If there's no terminal envelope at all, fall back to `{ output: items }`
 *
 * For Chat Completions streams, concatenate delta content from choices.
 */
const parseSSEBody = (body: string): unknown | null => {
  // Quick check: SSE bodies have lines starting with "event:" or "data:"
  if (!body.includes('data:')) return null

  const lines = body.split('\n')
  let hasData = false

  // Single pass: collect terminal envelope + output items
  let terminalResponse: Record<string, unknown> | null = null
  const outputItems: unknown[] = []

  for (const line of lines) {
    if (!line.startsWith('data:')) continue
    const data = line.slice(5).trim()
    if (!data || data === '[DONE]') continue
    hasData = true
    try {
      const parsed = JSON.parse(data)
      if (!isRecord(parsed)) continue
      if (
        (parsed.type === 'response.done' || parsed.type === 'response.completed') &&
        isRecord(parsed.response)
      ) {
        // Last terminal event wins (Responses API only emits one, but be defensive).
        terminalResponse = parsed.response
      }
      if (parsed.type === 'response.output_item.done' && parsed.item !== undefined) {
        outputItems.push(parsed.item)
      }
    } catch { /* skip unparseable lines */ }
  }

  if (!hasData) return null

  if (terminalResponse) {
    const terminalOutput = terminalResponse.output
    const hasTerminalOutput = Array.isArray(terminalOutput) && terminalOutput.length > 0
    if (hasTerminalOutput) return terminalResponse
    if (outputItems.length > 0) {
      // Preserve envelope metadata, fill missing output from collected items.
      return { ...terminalResponse, output: outputItems }
    }
    return terminalResponse
  }

  if (outputItems.length > 0) {
    return { output: outputItems }
  }

  // Strategy 3: Chat completions streaming — reconstruct from deltas
  let chatContent = ''
  let chatRole: string | undefined
  const chatToolCalls: Record<string, { name: string; arguments: string }> = {}

  for (const line of lines) {
    if (!line.startsWith('data:')) continue
    const data = line.slice(5).trim()
    if (!data || data === '[DONE]') continue
    try {
      const parsed = JSON.parse(data)
      if (!isRecord(parsed)) continue

      // Standard chat completions streaming format
      const choices = Array.isArray(parsed.choices) ? parsed.choices : []
      for (const choice of choices) {
        if (!isRecord(choice)) continue
        const delta = isRecord(choice.delta) ? choice.delta : undefined
        if (!delta) continue
        if (typeof delta.role === 'string') chatRole = delta.role
        if (typeof delta.content === 'string') chatContent += delta.content
        // Tool call deltas
        const toolCallDeltas = Array.isArray(delta.tool_calls) ? delta.tool_calls : []
        for (const tc of toolCallDeltas) {
          if (!isRecord(tc)) continue
          const idx = String(tc.index ?? '0')
          if (!chatToolCalls[idx]) chatToolCalls[idx] = { name: '', arguments: '' }
          const fn = isRecord(tc.function) ? tc.function : undefined
          if (fn) {
            if (typeof fn.name === 'string') chatToolCalls[idx].name += fn.name
            if (typeof fn.arguments === 'string') chatToolCalls[idx].arguments += fn.arguments
          }
        }
      }

      // Responses API content deltas
      if (parsed.type === 'response.output_text.delta' && typeof parsed.delta === 'string') {
        chatContent += parsed.delta
      }
      if (parsed.type === 'response.function_call_arguments.delta' && typeof parsed.delta === 'string') {
        const itemId = String(parsed.item_id ?? '0')
        if (!chatToolCalls[itemId]) chatToolCalls[itemId] = { name: '', arguments: '' }
        chatToolCalls[itemId].arguments += parsed.delta
      }
      if (parsed.type === 'response.function_call_arguments.done') {
        const itemId = String(parsed.item_id ?? '0')
        if (!chatToolCalls[itemId]) chatToolCalls[itemId] = { name: '', arguments: '' }
        if (typeof parsed.arguments === 'string') chatToolCalls[itemId].arguments = parsed.arguments
        if (typeof parsed.name === 'string') chatToolCalls[itemId].name = parsed.name
      }
    } catch { /* skip */ }
  }

  // Build reconstructed response
  const hasContent = chatContent.length > 0
  const toolCallEntries = Object.values(chatToolCalls).filter(tc => tc.name || tc.arguments)
  const hasToolCalls = toolCallEntries.length > 0

  if (hasContent || hasToolCalls) {
    const message: Record<string, unknown> = { role: chatRole ?? 'assistant' }
    if (hasContent) message.content = chatContent
    if (hasToolCalls) {
      message.tool_calls = toolCallEntries.map((tc, i) => ({
        id: `reconstructed_${i}`,
        type: 'function',
        function: { name: tc.name, arguments: tc.arguments },
      }))
    }
    return { choices: [{ message }] }
  }

  return null
}

const parseBody = (body: string | null | undefined): JsonParseResult => {
  const hasBody = typeof body === 'string' && body.trim().length > 0
  if (!hasBody) return { value: null, raw: null, hasBody: false, invalid: false }

  // Try direct JSON parse first
  const value = parseJsonValue(body)
  if (value !== null) {
    return { value, raw: body, hasBody: true, invalid: false }
  }

  // Try SSE streaming format
  const sseValue = parseSSEBody(body!)
  if (sseValue !== null) {
    return { value: sseValue, raw: body, hasBody: true, invalid: false }
  }

  return {
    value: null,
    raw: body,
    hasBody: true,
    invalid: body!.trim() !== 'null',
  }
}

const parsePossiblyJson = (value: unknown): unknown => {
  if (typeof value !== 'string') return value
  return parseJsonValue(value) ?? value
}

const stringValue = (value: unknown): string | undefined => {
  return typeof value === 'string' && value.length > 0 ? value : undefined
}

const normalizeToolName = (value: unknown): string => {
  return stringValue(value) ?? 'tool'
}

const sanitizeString = (value: string): string => {
  const parsed = parseJsonValue(value)
  if (parsed !== null) {
    const sanitized = sanitizeMetadata(parsed)
    return typeof sanitized === 'string' ? sanitizeString(sanitized) : JSON.stringify(sanitized)
  }

  return value.includes('encrypted_content') ? '[redacted]' : value
}

const sanitizeMetadata = (value: unknown): unknown => {
  if (Array.isArray(value)) return value.map(sanitizeMetadata)
  if (typeof value === 'string') return sanitizeString(value)
  if (!isRecord(value)) return value

  return Object.fromEntries(
    Object.entries(value)
      .filter(([key]) => key !== 'encrypted_content')
      .map(([key, nestedValue]) => [key, sanitizeMetadata(nestedValue)]),
  )
}

const sanitizeRecord = (value: Record<string, unknown>): Record<string, unknown> => {
  return sanitizeMetadata(value) as Record<string, unknown>
}

const imageUrlFromPart = (part: Record<string, unknown>): string | undefined => {
  const imageUrl = part.image_url
  if (typeof imageUrl === 'string') return imageUrl
  if (isRecord(imageUrl)) return stringValue(imageUrl.url)
  return stringValue(part.url) ?? stringValue(part.image_url_url)
}

const filePartFromRecord = (part: Record<string, unknown>): ConversationContentPart | null => {
  const file = isRecord(part.file) ? part.file : part
  const url = stringValue(file.url)
  const filename = stringValue(file.filename) ?? stringValue(file.name)
  const text = stringValue(file.text)
  const mimeType = stringValue(file.mime_type) ?? stringValue(file.mimeType)

  if (!url && !filename && !text && !mimeType) return null
  return { type: 'file', url, filename, text, mimeType }
}

const partsFromContent = (content: unknown): ConversationContentPart[] => {
  if (typeof content === 'string') return content ? [{ type: 'text', text: content }] : []
  if (content == null) return []

  if (Array.isArray(content)) {
    return content.flatMap((part): ConversationContentPart[] => {
      if (typeof part === 'string') return part ? [{ type: 'text', text: part }] : []
      if (!isRecord(part)) return [{ type: 'text', text: formatRawValue(sanitizeMetadata(part)) }]

      const type = part.type
      if (type === 'text' || type === 'input_text' || type === 'output_text') {
        const text = stringValue(part.text)
        return text ? [{ type: 'text', text }] : []
      }

      if (type === 'image_url' || type === 'input_image' || type === 'image') {
        const src = imageUrlFromPart(part)
        return src ? [{ type: 'image', src, alt: stringValue(part.alt) }] : []
      }

      if (type === 'file' || type === 'input_file') {
        const filePart = filePartFromRecord(part)
        return filePart ? [filePart] : []
      }

      const text = stringValue(part.text)
      if (text) return [{ type: 'text', text }]
      return [{ type: 'text', text: formatRawValue(sanitizeMetadata(part)) }]
    })
  }

  if (isRecord(content)) {
    const text = stringValue(content.text)
    if (text) return [{ type: 'text', text }]
    const src = imageUrlFromPart(content)
    if (src) return [{ type: 'image', src, alt: stringValue(content.alt) }]
  }

  return [{ type: 'text', text: formatRawValue(sanitizeMetadata(content)) }]
}

const detectFormat = (
  request: unknown,
  response: unknown,
  hint: ParseConversationPayloadInput['formatHint'],
): ConversationFormat => {
  if (hint === 'openai-chat' || hint === 'openai-responses' || hint === 'anthropic-messages' || hint === 'unknown') return hint

  if (isAnthropicMessagesRequest(request)) return 'anthropic-messages'
  if (isAnthropicMessageObject(response)) return 'anthropic-messages'

  if (isRecord(request) && Array.isArray(request.messages)) return 'openai-chat'
  if (isRecord(response) && Array.isArray(response.choices)) return 'openai-chat'

  if (isRecord(request) && Object.prototype.hasOwnProperty.call(request, 'input')) return 'openai-responses'
  if (isRecord(response) && (Array.isArray(response.output) || Object.prototype.hasOwnProperty.call(response, 'output_text'))) {
    return 'openai-responses'
  }

  return 'unknown'
}

export const parseConversationPayload = (input: ParseConversationPayloadInput): ConversationFlow => {
  const warnings: string[] = []
  const request = parseBody(input.requestBody)
  const response = parseBody(input.responseBody)

  if (request.invalid) warnings.push('Request body is not valid JSON.')
  if (response.invalid) warnings.push('Response body is not valid JSON.')

  const messages: ConversationMessage[] = []
  const systemPromptParts: string[] = []
  const systemPromptSources: ('developer' | 'system')[] = []
  const toolPartsByCallId = new Map<string, ConversationToolPart>()
  let systemPromptHandled = false
  let messageIndex = 0
  let partIndex = 0

  const nextMessageId = (role: ConversationMessageRole): string => createConversationNodeId(`message-${role}`, messageIndex++)
  const nextPartId = (type: ConversationPart['type']): string => createConversationNodeId(`part-${type}`, partIndex++)

  const pushMessage = (role: ConversationMessageRole, metadata?: Record<string, unknown>): ConversationMessage => {
    const message: ConversationMessage = {
      id: nextMessageId(role),
      role,
      parts: [],
      metadata: metadata ? sanitizeRecord(metadata) : undefined,
    }
    messages.push(message)
    return message
  }

  const getLastMessage = (role?: ConversationMessageRole): ConversationMessage | undefined => {
    const message = messages[messages.length - 1]
    if (!message) return undefined
    if (role && message.role !== role) return undefined
    return message
  }

  const mergeMessageMetadata = (message: ConversationMessage, metadata?: Record<string, unknown>) => {
    if (!metadata) return
    message.metadata = { ...message.metadata, ...sanitizeRecord(metadata) }
  }

  const pushPartMessage = (
    role: ConversationMessageRole,
    parts: ConversationPart[],
    metadata?: Record<string, unknown>,
    target?: ConversationMessage,
  ): ConversationMessage => {
    const message = target ?? getLastMessage(role) ?? pushMessage(role, metadata)
    if (message !== target) mergeMessageMetadata(message, metadata)
    message.parts.push(...parts)
    return message
  }

  const assignContentPartIds = (parts: UserContentPart[]): ConversationPart[] => {
    return parts.map((part) => ({ ...part, id: nextPartId(part.type) })) as ConversationPart[]
  }

  const isSystemPromptSource = (value: unknown): value is 'developer' | 'system' => {
    return value === 'developer' || value === 'system'
  }

  const pushSystemPrompt = (source: 'developer' | 'system', content: unknown) => {
    systemPromptHandled = true
    const text = partsFromContent(content)
      .filter((part): part is { type: 'text'; text: string } => part.type === 'text')
      .map((part) => part.text)
      .join('\n')

    if (text.trim()) {
      systemPromptParts.push(text)
      systemPromptSources.push(source)
    }
  }

  const pushContentParts = (
    role: ConversationMessageRole,
    content: unknown,
    metadata?: Record<string, unknown>,
    appendToLast = false,
    target?: ConversationMessage,
  ): ConversationMessage | undefined => {
    const rawParts = partsFromContent(content)
    const expandedParts = role === 'user' ? splitUserContentParts(rawParts) : rawParts
    const parts = assignContentPartIds(expandedParts)
    if (parts.length === 0) return undefined

    if (target) return pushPartMessage(role, parts, metadata, target)
    if (appendToLast) return pushPartMessage(role, parts, metadata)

    const message = pushMessage(role, metadata)
    message.parts.push(...parts)
    return message
  }

  const formatNestedRawValue = (raw: unknown): string => {
    const safeRaw = sanitizeMetadata(raw)
    if (safeRaw === null) return 'null'
    if (safeRaw === undefined) return 'undefined'
    return formatRawValue(safeRaw)
  }

  const rawTitle = (metadata: Record<string, unknown>): string => {
    if (metadata.rawSource === 'request') return 'Raw Request'
    if (metadata.rawSource === 'response') return 'Raw Response'
    return 'Raw'
  }

  const pushRawPart = (raw: unknown, metadata: Record<string, unknown>, role: ConversationMessageRole = 'assistant') => {
    const safeMetadata = sanitizeRecord(metadata)
    const part: ConversationPart = {
      id: nextPartId('raw'),
      type: 'raw',
      title: rawTitle(safeMetadata),
      raw: formatNestedRawValue(raw),
      defaultCollapsed: true,
      metadata: safeMetadata,
    }
    const message = pushMessage(role, safeMetadata)
    message.parts.push(part)
  }

  const pushRawRequest = () => {
    if (request.raw !== null) pushRawPart(request.raw, { rawSource: 'request' }, 'assistant')
  }

  const pushRawResponse = () => {
    if (response.raw !== null) pushRawPart(response.raw, { rawSource: 'response' }, 'assistant')
  }

  const pushRawFallback = () => {
    if (request.hasBody) pushRawRequest()
    if (response.hasBody) pushRawResponse()
  }

  const mergeToolResultMetadata = (part: ConversationToolPart, result: Record<string, unknown>) => {
    const safeResult = sanitizeRecord(result)
    part.state.metadata = { ...part.state.metadata, result: safeResult }
    part.metadata = { ...part.metadata, result: safeResult }
  }

  const pushToolCall = (toolCall: Record<string, unknown>, target?: ConversationMessage) => {
    const fn = isRecord(toolCall.function) ? toolCall.function : undefined
    const tool = normalizeToolName(fn?.name ?? toolCall.name ?? toolCall.tool_name)
    const callId = stringValue(toolCall.id) ?? stringValue(toolCall.call_id) ?? stringValue(toolCall.callId)
    const rawArguments = fn?.arguments ?? toolCall.arguments ?? toolCall.input
    const parsedArguments = sanitizeMetadata(parsePossiblyJson(rawArguments))
    const metadata = { call: sanitizeRecord(toolCall) }
    const title = stringValue(toolCall.title)
    const part: ConversationToolPart = {
      id: nextPartId('tool'),
      type: 'tool',
      callId,
      tool,
      state: {
        status: 'running',
        input: parsedArguments,
        ...(title ? { title } : {}),
        metadata,
      },
      metadata,
    }

    pushPartMessage('assistant', [part], undefined, target)
    if (callId) toolPartsByCallId.set(callId, part)
  }

  const toolOutputFromResult = (toolResult: Record<string, unknown>): unknown => {
    if (Object.prototype.hasOwnProperty.call(toolResult, 'content')) return toolResult.content
    if (Object.prototype.hasOwnProperty.call(toolResult, 'output')) return toolResult.output
    return toolResult.result
  }

  const pushToolResult = (toolResult: Record<string, unknown>, target?: ConversationMessage) => {
    const callId = stringValue(toolResult.tool_call_id) ?? stringValue(toolResult.call_id) ?? stringValue(toolResult.callId)
    const matched = callId ? toolPartsByCallId.get(callId) : undefined
    const tool = normalizeToolName(toolResult.name ?? toolResult.tool_name ?? matched?.tool)
    const output = sanitizeMetadata(parsePossiblyJson(toolOutputFromResult(toolResult)))

    if (matched) {
      matched.tool = tool
      matched.state.status = 'completed'
      matched.state.output = output
      attachToolOutputSize(matched)
      mergeToolResultMetadata(matched, toolResult)
      return
    }

    const metadata = { result: sanitizeRecord(toolResult) }
    const part: ConversationToolPart = {
      id: nextPartId('tool'),
      type: 'tool',
      callId,
      tool,
      state: {
        status: 'completed',
        output,
        metadata,
      },
      metadata,
    }

    const message = target ?? pushMessage('assistant')
    message.parts.push(part)
    attachToolOutputSize(part)
    if (callId) toolPartsByCallId.set(callId, part)
  }

  const pushReasoningPart = (text: string, metadata?: Record<string, unknown>, target?: ConversationMessage) => {
    const part: ConversationPart = {
      id: nextPartId('reasoning'),
      type: 'reasoning',
      text,
      defaultCollapsed: true,
      metadata: metadata ? sanitizeRecord(metadata) : undefined,
    }
    pushPartMessage('assistant', [part], metadata, target)
  }

  const parseChatMessage = (message: unknown, rawSource: 'request' | 'response', nestedSource: string) => {
    const messageCount = messages.length
    let handled = false
    let target: ConversationMessage | undefined

    if (!isRecord(message)) {
      pushRawPart(message, { rawSource, nestedSource })
      return
    }

    if (isSystemPromptSource(message.role)) {
      pushSystemPrompt(message.role, message.content)
      return
    }

    if (message.role === 'tool') {
      pushToolResult(message)
      return
    }

    const toolCalls = Array.isArray(message.tool_calls) ? message.tool_calls : []
    const hasToolCalls = toolCalls.length > 0
    if (isMessageRole(message.role)) {
      target = pushContentParts(message.role, message.content, message)
      if (!target && hasToolCalls && message.role === 'assistant') target = pushMessage('assistant', message)
      handled = Boolean(target)
    }

    if (hasToolCalls) {
      toolCalls.forEach((toolCall) => {
        if (isRecord(toolCall)) {
          pushToolCall(toolCall, target)
          handled = true
        }
      })
    }

    if (!handled && messages.length === messageCount) pushRawPart(message, { rawSource, nestedSource })
  }

  const parseChat = () => {
    const requestMessageCount = messages.length
    if (isRecord(request.value) && Array.isArray(request.value.messages)) {
      request.value.messages.forEach((message) => parseChatMessage(message, 'request', 'messages'))
    }
    if (request.hasBody && messages.length === requestMessageCount && !systemPromptHandled) pushRawRequest()

    toolPartsByCallId.clear()

    const responseMessageCount = messages.length
    if (isRecord(response.value) && Array.isArray(response.value.choices)) {
      response.value.choices.forEach((choice) => {
        if (!isRecord(choice)) {
          pushRawPart(choice, { rawSource: 'response', nestedSource: 'choices' })
          return
        }

        const hasMessage = Object.prototype.hasOwnProperty.call(choice, 'message')
        const hasDelta = Object.prototype.hasOwnProperty.call(choice, 'delta')
        if (!hasMessage && !hasDelta) {
          pushRawPart(choice, { rawSource: 'response', nestedSource: 'choices' })
          return
        }

        const message = choice.message ?? choice.delta
        if (message == null) {
          pushRawPart(choice, { rawSource: 'response', nestedSource: 'choices' })
          return
        }

        parseChatMessage(message, 'response', 'choices')
      })
    }
    if (response.hasBody && messages.length === responseMessageCount) pushRawResponse()
  }

  const parseAnthropicContentBlock = (
    block: unknown,
    role: ConversationMessageRole,
    rawSource: 'request' | 'response',
    nestedSource: string,
    target?: ConversationMessage,
  ): boolean => {
    if (typeof block === 'string') {
      const text = block.trim().length > 0 || role === 'user' ? block : ''
      if (!text) return false
      pushContentParts(role, text, undefined, true, target)
      return true
    }

    if (!isRecord(block)) {
      pushRawPart(block, { rawSource, nestedSource }, role)
      return true
    }

    if (block.type === 'text') {
      const text = stringValue(block.text)
      if (!text || (rawSource === 'response' && text.trim().length === 0)) return false
      pushContentParts(role, [{ type: 'text', text }], sanitizeRecord(block), true, target)
      return true
    }

    if (block.type === 'thinking') {
      const thinking = stringValue(block.thinking)
      if (!thinking || thinking.trim().length === 0) return false
      pushReasoningPart(thinking, sanitizeRecord(block), role === 'assistant' ? target : undefined)
      return true
    }

    if (block.type === 'tool_use') {
      const normalizedToolCall = {
        ...block,
        id: stringValue(block.id),
        name: stringValue(block.name),
        input: block.input,
      }
      pushToolCall(normalizedToolCall, role === 'assistant' ? target : undefined)
      return true
    }

    if (block.type === 'tool_result') {
      const normalizedToolResult = {
        ...block,
        call_id: stringValue(block.tool_use_id),
        content: block.content,
      }
      pushToolResult(normalizedToolResult, role === 'assistant' ? target : undefined)
      return true
    }

    pushRawPart(block, { rawSource, nestedSource }, role)
    return true
  }

  const parseAnthropicMessage = (message: unknown, rawSource: 'request' | 'response', nestedSource: string, target?: ConversationMessage) => {
    const before = messages.length

    if (!isRecord(message)) {
      pushRawPart(message, { rawSource, nestedSource })
      return
    }

    const role = isMessageRole(message.role) ? message.role : 'assistant'
    if (role === 'system' || role === 'developer') {
      pushSystemPrompt(role, message.content)
      return
    }

    const messageTarget = target ?? pushMessage(role, message)
    const content = Array.isArray(message.content) ? message.content : [message.content]
    let handled = false

    for (const block of content) {
      handled = parseAnthropicContentBlock(block, role, rawSource, nestedSource, messageTarget) || handled
    }

    if (!handled && messages.length === before + 1) {
      messages.pop()
      pushRawPart(message, { rawSource, nestedSource }, role)
    }
  }

  const parseAnthropic = () => {
    const requestMessageCount = messages.length
    if (isRecord(request.value)) {
      if (Object.prototype.hasOwnProperty.call(request.value, 'system')) {
        pushSystemPrompt('system', request.value.system)
      }
      if (Array.isArray(request.value.messages)) {
        request.value.messages.forEach((message) => parseAnthropicMessage(message, 'request', 'messages'))
      }
    }
    if (request.hasBody && messages.length === requestMessageCount && !systemPromptHandled) pushRawRequest()

    toolPartsByCallId.clear()

    const responseMessageCount = messages.length
    if (isAnthropicMessageObject(response.value)) {
      parseAnthropicMessage(response.value, 'response', 'message')
    }
    if (response.hasBody && messages.length === responseMessageCount) pushRawResponse()
  }

  const parseResponsesItem = (
    item: unknown,
    fallbackRole: ConversationMessageRole,
    rawSource: 'request' | 'response',
    nestedSource: string,
    getResponseTarget?: () => ConversationMessage,
  ) => {
    const appendToLast = rawSource === 'response' && !getResponseTarget

    if (typeof item === 'string') {
      const target = fallbackRole === 'assistant' ? getResponseTarget?.() : undefined
      pushContentParts(fallbackRole, item, undefined, appendToLast, target)
      return
    }

    if (!isRecord(item)) {
      pushRawPart(item, { rawSource, nestedSource }, fallbackRole)
      return
    }

    const type = item.type
    const systemPromptSource = isSystemPromptSource(item.role)
      ? item.role
      : isSystemPromptSource(type)
        ? type
        : undefined

    if (systemPromptSource) {
      pushSystemPrompt(systemPromptSource, item.content ?? item.output_text ?? item.text)
      return
    }

    if (type === 'message' || isMessageRole(item.role)) {
      const role = isMessageRole(item.role) ? item.role : fallbackRole
      const target = role === 'assistant' ? getResponseTarget?.() : undefined
      const pushed = pushContentParts(role, item.content ?? item.output_text ?? item.text, item, appendToLast, target)
      if (!pushed) pushRawPart(item, { rawSource, nestedSource }, role)
      return
    }

    if (type === 'function_call' || type === 'tool_call') {
      pushToolCall(item, fallbackRole === 'assistant' ? getResponseTarget?.() : undefined)
      return
    }

    if (type === 'function_call_output' || type === 'tool_result') {
      pushToolResult(item, fallbackRole === 'assistant' ? getResponseTarget?.() : undefined)
      return
    }

    if (type === 'output_text' || type === 'input_text') {
      const target = fallbackRole === 'assistant' ? getResponseTarget?.() : undefined
      const pushed = pushContentParts(fallbackRole, [item], item, appendToLast, target)
      if (!pushed) pushRawPart(item, { rawSource, nestedSource }, fallbackRole)
      return
    }

    if (type === 'reasoning') {
      const summary = Array.isArray(item.summary) ? item.summary : undefined
      if (summary && summary.length > 0) {
        const texts = summary
          .filter((summaryItem) => isRecord(summaryItem) && summaryItem.type === 'summary_text')
          .map((summaryItem) => stringValue(summaryItem.text))
          .filter(Boolean) as string[]
        if (texts.length > 0) {
          pushReasoningPart(texts.join('\n'), sanitizeRecord(item), fallbackRole === 'assistant' ? getResponseTarget?.() : undefined)
          return
        }
      }

      // Empty reasoning (e.g. summary: []) — skip silently, no raw fallback
      return
    }

    pushRawPart(item, { rawSource, nestedSource }, fallbackRole)
  }

  const parseResponsesInput = (value: unknown) => {
    if (Array.isArray(value)) {
      value.forEach((item) => parseResponsesItem(item, 'user', 'request', 'input'))
      return
    }

    if (typeof value === 'string') pushContentParts('user', value)
    else if (value != null) parseResponsesItem(value, 'user', 'request', 'input')
  }

  const parseResponses = () => {
    const requestMessageCount = messages.length
    if (isRecord(request.value) && Object.prototype.hasOwnProperty.call(request.value, 'input')) {
      parseResponsesInput(request.value.input)
    }
    if (request.hasBody && messages.length === requestMessageCount && !systemPromptHandled) pushRawRequest()

    toolPartsByCallId.clear()

    const responseMessageCount = messages.length
    let responseAssistantTarget: ConversationMessage | undefined
    const getResponseAssistantTarget = (): ConversationMessage => {
      responseAssistantTarget ??= pushMessage('assistant')
      return responseAssistantTarget
    }

    if (isRecord(response.value)) {
      if (Array.isArray(response.value.output)) {
        response.value.output.forEach((item) => parseResponsesItem(item, 'assistant', 'response', 'output', getResponseAssistantTarget))
      }

      const hasStructuredAssistantText = responseAssistantTarget?.parts.some((part) => part.type === 'text') ?? false
      if (typeof response.value.output_text === 'string' && !hasStructuredAssistantText) {
        pushContentParts('assistant', response.value.output_text, undefined, false, getResponseAssistantTarget())
      }
    }
    if (response.hasBody && messages.length === responseMessageCount) pushRawResponse()
  }

  const format = detectFormat(request.value, response.value, input.formatHint)

  try {
    if (format === 'openai-chat') parseChat()
    else if (format === 'openai-responses') parseResponses()
    else if (format === 'anthropic-messages') parseAnthropic()

    if (format === 'unknown') pushRawFallback()
  } catch (error) {
    warnings.push(error instanceof Error ? error.message : 'Failed to parse conversation payload.')
    messages.length = 0
    toolPartsByCallId.clear()
    pushRawFallback()
  }

  messages.forEach(mergeConsecutiveReasoning)

  const systemPrompt = systemPromptParts.length > 0
    ? {
        id: createConversationNodeId('system-prompt', 0),
        text: systemPromptParts.join('\n\n'),
        sources: [...new Set(systemPromptSources)],
      }
    : undefined

  return {
    source: input.source ?? 'client',
    format,
    warnings,
    messages: messages.filter((message) => message.parts.length > 0),
    nodes: [],
    ...(systemPrompt ? { systemPrompt } : {}),
  }
}
