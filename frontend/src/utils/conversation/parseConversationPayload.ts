import { createConversationNodeId, formatRawValue, parseJsonValue, summarizeText, textFromParts } from './format'
import type {
  ConversationContentPart,
  ConversationFlow,
  ConversationFormat,
  ConversationMessageNode,
  ConversationNode,
  ConversationToolNode,
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

const isMessageRole = (value: unknown): value is ConversationMessageNode['role'] => {
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

const titleForRole = (role: ConversationMessageNode['role']): string => {
  if (role === 'user') return 'User'
  if (role === 'assistant') return 'Assistant'
  if (role === 'system') return 'System'
  return 'Developer'
}

const defaultCollapsedForRole = (role: ConversationMessageNode['role']): boolean => {
  return role === 'system' || role === 'developer'
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
      if (!isRecord(part)) return [{ type: 'text', text: formatRawValue(part) }]

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
      return [{ type: 'text', text: formatRawValue(part) }]
    })
  }

  if (isRecord(content)) {
    const text = stringValue(content.text)
    if (text) return [{ type: 'text', text }]
    const src = imageUrlFromPart(content)
    if (src) return [{ type: 'image', src, alt: stringValue(content.alt) }]
  }

  return [{ type: 'text', text: formatRawValue(content) }]
}

const isDuplicateAssistant = (previous: ConversationNode | undefined, next: ConversationMessageNode): boolean => {
  // Only collapse adjacent assistant messages when their renderable parts are exactly identical.
  return previous?.type === 'assistant'
    && next.type === 'assistant'
    && textFromParts(previous.parts) === textFromParts(next.parts)
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

  const nodes: ConversationNode[] = []
  const toolNamesByCallId = new Map<string, string>()
  let nodeIndex = 0

  const nextId = (prefix: string) => createConversationNodeId(prefix, nodeIndex++)

  const pushMessage = (role: ConversationMessageNode['role'], content: unknown, metadata?: Record<string, unknown>): boolean => {
    const parts = partsFromContent(content)
    if (parts.length === 0) return false

    const node: ConversationMessageNode = {
      id: nextId(role),
      type: role,
      role,
      title: titleForRole(role),
      summary: summarizeText(textFromParts(parts)),
      defaultCollapsed: defaultCollapsedForRole(role),
      parts,
      metadata,
    }

    if (!isDuplicateAssistant(nodes[nodes.length - 1], node)) nodes.push(node)
    return true
  }

  const formatNestedRawValue = (raw: unknown): string => {
    if (raw === null) return 'null'
    if (raw === undefined) return 'undefined'
    return formatRawValue(raw)
  }

  const pushRawNode = (raw: unknown, metadata: Record<string, unknown>, prefix = 'raw') => {
    nodes.push({
      id: nextId(prefix),
      type: 'raw',
      title: 'Raw',
      defaultCollapsed: true,
      raw: formatNestedRawValue(raw),
      metadata,
    })
  }

  const pushToolCall = (toolCall: Record<string, unknown>) => {
    const fn = isRecord(toolCall.function) ? toolCall.function : undefined
    const toolName = stringValue(fn?.name) ?? stringValue(toolCall.name) ?? stringValue(toolCall.tool_name)
    const callId = stringValue(toolCall.id) ?? stringValue(toolCall.call_id) ?? stringValue(toolCall.callId)
    const rawArguments = fn?.arguments ?? toolCall.arguments ?? toolCall.input
    const parsedArguments = parsePossiblyJson(rawArguments)

    if (callId && toolName) toolNamesByCallId.set(callId, toolName)

    const node: ConversationToolNode = {
      id: nextId('tool-call'),
      type: 'tool_call',
      title: `Tool Call${toolName ? ` · ${toolName}` : ''}`,
      summary: toolName,
      defaultCollapsed: true,
      toolName,
      callId,
      input: parsedArguments,
      metadata: toolCall,
    }
    nodes.push(node)
  }

  const pushToolResult = (toolResult: Record<string, unknown>) => {
    const callId = stringValue(toolResult.tool_call_id) ?? stringValue(toolResult.call_id) ?? stringValue(toolResult.callId)
    const toolName = stringValue(toolResult.name) ?? stringValue(toolResult.tool_name) ?? (callId ? toolNamesByCallId.get(callId) : undefined)
    const output = parsePossiblyJson(toolResult.content ?? toolResult.output ?? toolResult.result)

    const node: ConversationToolNode = {
      id: nextId('tool-result'),
      type: 'tool_result',
      title: `Tool Result${toolName ? ` · ${toolName}` : ''}`,
      summary: toolName,
      defaultCollapsed: true,
      toolName,
      callId,
      output,
      metadata: toolResult,
    }
    nodes.push(node)
  }

  const parseChatMessage = (message: unknown, rawSource: 'request' | 'response', nestedSource: string) => {
    const nodeCount = nodes.length
    let handled = false

    if (!isRecord(message)) {
      pushRawNode(message, { rawSource, nestedSource }, `raw-${rawSource}`)
      return
    }

    if (message.role === 'tool') {
      pushToolResult(message)
      return
    }

    if (isMessageRole(message.role)) handled = pushMessage(message.role, message.content, message)

    if (Array.isArray(message.tool_calls)) {
      message.tool_calls.forEach((toolCall) => {
        if (isRecord(toolCall)) {
          pushToolCall(toolCall)
          handled = true
        }
      })
    }

    if (!handled && nodes.length === nodeCount) pushRawNode(message, { rawSource, nestedSource }, `raw-${rawSource}`)
  }

  const parseChat = () => {
    const requestNodeCount = nodes.length
    if (isRecord(request.value) && Array.isArray(request.value.messages)) {
      request.value.messages.forEach((message) => parseChatMessage(message, 'request', 'messages'))
    }
    if (request.hasBody && nodes.length === requestNodeCount) pushRawRequest()

    const responseNodeCount = nodes.length
    if (isRecord(response.value) && Array.isArray(response.value.choices)) {
      response.value.choices.forEach((choice) => {
        if (!isRecord(choice)) {
          pushRawNode(choice, { rawSource: 'response', nestedSource: 'choices' }, 'raw-response')
          return
        }

        const hasMessage = Object.prototype.hasOwnProperty.call(choice, 'message')
        const hasDelta = Object.prototype.hasOwnProperty.call(choice, 'delta')
        if (!hasMessage && !hasDelta) {
          pushRawNode(choice, { rawSource: 'response', nestedSource: 'choices' }, 'raw-response')
          return
        }

        const message = choice.message ?? choice.delta
        if (message == null) {
          pushRawNode(choice, { rawSource: 'response', nestedSource: 'choices' }, 'raw-response')
          return
        }

        parseChatMessage(message, 'response', 'choices')
      })
    }
    if (response.hasBody && nodes.length === responseNodeCount) pushRawResponse()
  }

  const parseResponsesItem = (item: unknown, fallbackRole: ConversationMessageNode['role'], rawSource: 'request' | 'response', nestedSource: string) => {
    if (typeof item === 'string') {
      pushMessage(fallbackRole, item)
      return
    }

    if (!isRecord(item)) {
      pushRawNode(item, { rawSource, nestedSource }, `raw-${rawSource}`)
      return
    }

    const type = item.type

    if (type === 'message' || isMessageRole(item.role)) {
      const pushed = pushMessage(isMessageRole(item.role) ? item.role : fallbackRole, item.content ?? item.output_text ?? item.text, item)
      if (!pushed) pushRawNode(item, { rawSource, nestedSource }, `raw-${rawSource}`)
      return
    }

    if (type === 'function_call' || type === 'tool_call') {
      pushToolCall(item)
      return
    }

    if (type === 'function_call_output' || type === 'tool_result') {
      pushToolResult(item)
      return
    }

    if (type === 'output_text' || type === 'input_text') {
      const pushed = pushMessage(fallbackRole, [item])
      if (!pushed) pushRawNode(item, { rawSource, nestedSource }, `raw-${rawSource}`)
      return
    }

    pushRawNode(item, { rawSource, nestedSource }, `raw-${rawSource}`)
  }

  const parseResponsesInput = (value: unknown) => {
    if (Array.isArray(value)) {
      value.forEach((item) => parseResponsesItem(item, 'user', 'request', 'input'))
      return
    }

    if (typeof value === 'string') pushMessage('user', value)
    else if (value != null) parseResponsesItem(value, 'user', 'request', 'input')
  }

  const parseResponses = () => {
    const requestNodeCount = nodes.length
    if (isRecord(request.value) && Object.prototype.hasOwnProperty.call(request.value, 'input')) {
      parseResponsesInput(request.value.input)
    }
    if (request.hasBody && nodes.length === requestNodeCount) pushRawRequest()

    const responseNodeCount = nodes.length
    if (isRecord(response.value)) {
      if (Array.isArray(response.value.output)) {
        response.value.output.forEach((item) => parseResponsesItem(item, 'assistant', 'response', 'output'))
      }

      if (typeof response.value.output_text === 'string') pushMessage('assistant', response.value.output_text)
    }
    if (response.hasBody && nodes.length === responseNodeCount) pushRawResponse()
  }

  const pushRawRequest = () => {
    nodes.push({
      id: nextId('raw-request'),
      type: 'raw',
      title: 'Raw Request',
      defaultCollapsed: true,
      raw: formatRawValue(request.raw),
      metadata: { rawSource: 'request' },
    })
  }

  const pushRawResponse = () => {
    nodes.push({
      id: nextId('raw-response'),
      type: 'raw',
      title: 'Raw Response',
      defaultCollapsed: true,
      raw: formatRawValue(response.raw),
      metadata: { rawSource: 'response' },
    })
  }

  const pushRawFallback = () => {
    if (request.hasBody) pushRawRequest()
    if (response.hasBody) pushRawResponse()
  }

  const format = detectFormat(request.value, response.value, input.formatHint)

  try {
    if (format === 'openai-chat') parseChat()
    else if (format === 'openai-responses') parseResponses()

    if (format === 'unknown') pushRawFallback()
  } catch (error) {
    warnings.push(error instanceof Error ? error.message : 'Failed to parse conversation payload.')
    nodes.length = 0
    pushRawFallback()
  }

  return {
    source: input.source ?? 'client',
    format,
    warnings,
    nodes,
  }
}
