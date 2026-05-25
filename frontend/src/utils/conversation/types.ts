export type ConversationSource = 'client' | 'upstream'

export type ConversationFormat = 'openai-chat' | 'openai-responses' | 'unknown'

export type ConversationNodeType =
  | 'user'
  | 'assistant'
  | 'system'
  | 'developer'
  | 'reasoning'
  | 'tool_call'
  | 'tool_result'
  | 'image'
  | 'file'
  | 'raw'
  | 'error'

export interface ConversationFlow {
  nodes: ConversationNode[]
  source: ConversationSource
  format: ConversationFormat
  warnings: string[]
}

export interface ConversationBaseNode {
  id: string
  type: ConversationNodeType
  title: string
  summary?: string
  defaultCollapsed: boolean
  metadata?: Record<string, unknown>
}

export interface ConversationTextPart {
  type: 'text'
  text: string
}

export interface ConversationImagePart {
  type: 'image'
  src: string
  alt?: string
}

export interface ConversationFilePart {
  type: 'file'
  filename?: string
  mimeType?: string
  url?: string
  text?: string
}

export type ConversationContentPart = ConversationTextPart | ConversationImagePart | ConversationFilePart

export interface ConversationMessageNode extends ConversationBaseNode {
  type: 'user' | 'assistant' | 'system' | 'developer'
  role: 'user' | 'assistant' | 'system' | 'developer'
  parts: ConversationContentPart[]
}

export interface ConversationToolNode extends ConversationBaseNode {
  type: 'tool_call' | 'tool_result'
  toolName?: string
  callId?: string
  input?: unknown
  output?: unknown
}

export interface ConversationReasoningNode extends ConversationBaseNode {
  type: 'reasoning'
  parts: ConversationContentPart[]
}

export interface ConversationMediaNode extends ConversationBaseNode {
  type: 'image' | 'file'
  parts: ConversationContentPart[]
}

export interface ConversationRawNode extends ConversationBaseNode {
  type: 'raw'
  raw: string
}

export interface ConversationErrorNode extends ConversationBaseNode {
  type: 'error'
  error: string
  raw?: string
}

export type ConversationNode =
  | ConversationMessageNode
  | ConversationToolNode
  | ConversationReasoningNode
  | ConversationMediaNode
  | ConversationRawNode
  | ConversationErrorNode

export interface ParseConversationPayloadInput {
  requestBody: string | null | undefined
  responseBody: string | null | undefined
  source?: ConversationSource
  formatHint?: ConversationFormat | 'auto'
}
