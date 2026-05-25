import type { ConversationContentPart, ConversationFlow, ConversationNode } from './types'

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

const copyLabelForNode = (node: ConversationNode): string => {
  if (node.type === 'tool_call' && node.toolName) return `tool_call: ${node.toolName}`
  if (node.type === 'tool_result' && node.toolName) return `tool_result: ${node.toolName}`
  if (node.type === 'reasoning') return 'reasoning'
  return node.type
}

const copyBodyForNode = (node: ConversationNode): string => {
  if ('parts' in node) return textFromParts(node.parts)
  if (node.type === 'tool_call') return formatRawValue(node.input ?? node.metadata ?? '')
  if (node.type === 'tool_result') return formatRawValue(node.output ?? node.metadata ?? '')
  if (node.type === 'raw') return node.raw
  if (node.type === 'error') return [node.error, node.raw].filter(Boolean).join('\n')
  return node.summary || ''
}

export const formatConversationAsText = (flow: ConversationFlow): string => {
  return flow.nodes.map((node) => {
    const body = copyBodyForNode(node)
    return [`[${copyLabelForNode(node)}]`, body].filter(Boolean).join('\n')
  }).join('\n\n')
}
