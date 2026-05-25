import { createConversationNodeId, formatRawValue, parseJsonValue } from './format'
import type {
  ConversationContentPart,
  ConversationFlow,
  ConversationFormat,
  ConversationMessage,
  ConversationMessageRole,
  ConversationPart,
  ConversationToolPart,
  ParseConversationPayloadInput,
} from './types'

type JsonParseResult = {
  value: unknown | null
  raw: string | null
  hasBody: boolean
  invalid: boolean
}

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

const isMessageRole = (value: unknown): value is ConversationMessageRole => {
  return value === 'user' || value === 'assistant' || value === 'system' || value === 'developer'
}

const parseBody = (body: string | null | undefined): JsonParseResult => {
  const hasBody = typeof body === 'string' && body.trim().length > 0
  if (!hasBody) return { value: null, raw: null, hasBody: false, invalid: false }

  const value = parseJsonValue(body)
  return {
    value,
    raw: body,
    hasBody: true,
    invalid: value === null && body.trim() !== 'null',
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
  if (hint === 'openai-chat' || hint === 'openai-responses' || hint === 'unknown') return hint

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
  const toolPartsByCallId = new Map<string, ConversationToolPart>()
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

  const assignContentPartIds = (parts: ConversationContentPart[]): ConversationPart[] => {
    return parts.map((part) => ({ ...part, id: nextPartId(part.type) })) as ConversationPart[]
  }

  const pushContentParts = (
    role: ConversationMessageRole,
    content: unknown,
    metadata?: Record<string, unknown>,
    appendToLast = false,
    target?: ConversationMessage,
  ): ConversationMessage | undefined => {
    const parts = assignContentPartIds(partsFromContent(content))
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
    if (request.hasBody && messages.length === requestMessageCount) pushRawRequest()

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

      pushRawPart(sanitizeRecord(item), { rawSource, nestedSource }, fallbackRole)
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
    if (request.hasBody && messages.length === requestMessageCount) pushRawRequest()

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

    if (format === 'unknown') pushRawFallback()
  } catch (error) {
    warnings.push(error instanceof Error ? error.message : 'Failed to parse conversation payload.')
    messages.length = 0
    toolPartsByCallId.clear()
    pushRawFallback()
  }

  return {
    source: input.source ?? 'client',
    format,
    warnings,
    messages: messages.filter((message) => message.parts.length > 0),
    nodes: [],
  }
}
