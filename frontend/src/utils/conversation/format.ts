import type { ConversationContentPart, ConversationFlow, ConversationPart, ConversationToolPart } from './types'

// Treat JSON null, empty input, and parse failures as no usable JSON value.
export const parseJsonValue = (value: string | null | undefined): unknown | null => {
  if (typeof value !== 'string' || !value.trim()) return null

  try {
    return JSON.parse(value)
  } catch {
    return null
  }
}

export const formatRawValue = (value: unknown): string => {
  if (value == null) return ''

  if (typeof value === 'string') {
    const parsed = parseJsonValue(value)
    if (parsed !== null) {
      return JSON.stringify(parsed, null, 2)
    }
    return value
  }

  try {
    const formatted = JSON.stringify(value, null, 2)
    return typeof formatted === 'string' ? formatted : String(value)
  } catch {
    return String(value)
  }
}

export const summarizeText = (value: string, maxLength = 120): string => {
  const compact = value.replace(/\s+/g, ' ').trim()
  if (compact.length <= maxLength) return compact
  if (maxLength <= 3) return compact.slice(0, Math.max(0, maxLength))
  return `${compact.slice(0, Math.max(0, maxLength - 3))}...`
}

export const formatHumanBytes = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`

  const unit = bytes < 1024 * 1024 ? 'KB' : 'MB'
  const value = bytes / (unit === 'KB' ? 1024 : 1024 * 1024)
  const formatted = value < 10 ? value.toFixed(1) : Math.round(value).toString()
  return `${formatted} ${unit}`
}

export const createConversationNodeId = (prefix: string, index: number): string => `${prefix}-${index}`

export const textFromParts = (parts: ConversationContentPart[]): string => {
  return parts.map((part) => {
    switch (part.type) {
      case 'text':
        return part.text
      case 'image':
        return `[image: ${part.alt || part.src}]`
      case 'file':
        return `[file: ${part.filename || part.url || part.mimeType || 'attachment'}]`
      default: {
        const exhaustive: never = part
        return exhaustive
      }
    }
  }).filter(Boolean).join('\n')
}

const formatToolPartForCopy = (part: ConversationToolPart): string => {
  const lines: string[] = []
  const command = part.state.input && typeof part.state.input === 'object' && !Array.isArray(part.state.input)
    ? (part.state.input as Record<string, unknown>).command
    : undefined

  if (part.tool === 'bash' && typeof command === 'string' && command.length > 0) lines.push(`$ ${command}`)
  else if (part.state.input !== undefined && part.state.output === undefined) lines.push(formatRawValue(part.state.input))

  if (part.state.output !== undefined) lines.push(formatRawValue(part.state.output))
  if (part.state.error) lines.push(part.state.error)

  return lines.filter(Boolean).join('\n')
}

const copyLabelForPart = (part: ConversationPart): string => {
  if (part.type === 'tool') return `tool: ${part.tool}`
  if (part.type === 'injection') return `injection: ${part.tag}`
  return part.type
}

const copyBodyForPart = (part: ConversationPart): string => {
  if (part.type === 'text') return part.text
  if (part.type === 'reasoning') return part.text
  if (part.type === 'image') return `[image: ${part.alt || part.src}]`
  if (part.type === 'file') return `[file: ${part.filename || part.url || part.mimeType || 'attachment'}]`
  if (part.type === 'tool') return formatToolPartForCopy(part)
  if (part.type === 'raw') return part.raw
  if (part.type === 'error') return [part.error, part.raw].filter(Boolean).join('\n')
  if (part.type === 'injection') return part.text
  return ''
}

export const formatConversationAsText = (flow: ConversationFlow): string => {
  const blocks: string[] = []

  if (flow.systemPrompt) {
    blocks.push(`[system prompt]\n${flow.systemPrompt.text}`)
  }

  const messageBlocks = (flow.messages ?? []).map((message) => {
    const partText = message.parts.map((part) => {
      const body = copyBodyForPart(part)
      return [`[${copyLabelForPart(part)}]`, body].filter(Boolean).join('\n')
    }).filter(Boolean).join('\n\n')
    return [`[${message.role}]`, partText].filter(Boolean).join('\n')
  }).filter(Boolean)

  blocks.push(...messageBlocks)
  return blocks.join('\n\n')
}
