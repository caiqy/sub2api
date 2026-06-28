export type ConversationSource = 'client' | 'upstream'

export type ConversationFormat = 'openai-chat' | 'openai-responses' | 'anthropic-messages' | 'unknown'

export type ConversationMessageRole = 'user' | 'assistant' | 'system' | 'developer'

export type ConversationPartType = 'text' | 'reasoning' | 'tool' | 'image' | 'file' | 'raw' | 'error' | 'injection'

export interface ConversationFlow {
  messages?: ConversationMessage[]
  nodes: ConversationNode[]
  source: ConversationSource
  format: ConversationFormat
  warnings: string[]
  systemPrompt?: ConversationSystemPrompt
}

export interface ConversationSystemPrompt {
  id: string
  text: string
  sources: ('developer' | 'system')[]
}

export interface ConversationMessage {
  id: string
  role: ConversationMessageRole
  parts: ConversationPart[]
  metadata?: Record<string, unknown>
}

export interface ConversationBasePart {
  id: string
  type: ConversationPartType
  metadata?: Record<string, unknown>
}

export interface ConversationTextPart extends ConversationBasePart {
  type: 'text'
  text: string
}

export interface ConversationReasoningPart extends ConversationBasePart {
  type: 'reasoning'
  text: string
  defaultCollapsed: true
}

export interface ConversationInjectionPart extends ConversationBasePart {
  type: 'injection'
  tag: string
  text: string
  defaultCollapsed: true
}

export interface ConversationImagePart extends ConversationBasePart {
  type: 'image'
  src: string
  alt?: string
}

export interface ConversationFilePart extends ConversationBasePart {
  type: 'file'
  filename?: string
  mimeType?: string
  url?: string
  text?: string
}

export type ConversationLegacyTextPart = Omit<ConversationTextPart, 'id'> & { id?: string }

export type ConversationLegacyImagePart = Omit<ConversationImagePart, 'id'> & { id?: string }

export type ConversationLegacyFilePart = Omit<ConversationFilePart, 'id'> & { id?: string }

export type ConversationContentPart = ConversationLegacyTextPart | ConversationLegacyImagePart | ConversationLegacyFilePart

export type ConversationToolStatus = 'pending' | 'running' | 'completed' | 'error'

export interface ConversationToolPart extends ConversationBasePart {
  type: 'tool'
  callId?: string
  tool: string
  state: {
    status: ConversationToolStatus
    input?: unknown
    output?: unknown
    title?: string
    error?: string
    outputSize?: { bytes: number; lines: number }
    metadata?: Record<string, unknown>
  }
}

export interface ConversationRawPart extends ConversationBasePart {
  type: 'raw'
  title?: string
  raw: string
  defaultCollapsed: true
}

export interface ConversationErrorPart extends ConversationBasePart {
  type: 'error'
  error: string
  raw?: string
  defaultCollapsed: true
}

export type ConversationPart =
  | ConversationTextPart
  | ConversationReasoningPart
  | ConversationInjectionPart
  | ConversationToolPart
  | ConversationImagePart
  | ConversationFilePart
  | ConversationRawPart
  | ConversationErrorPart

export interface ParseConversationPayloadInput {
  requestBody: string | null | undefined
  responseBody: string | null | undefined
  source?: ConversationSource
  formatHint?: ConversationFormat | 'auto'
}

export type ConversationNodeType =
  | ConversationMessageRole
  | 'reasoning'
  | 'tool_call'
  | 'tool_result'
  | 'image'
  | 'file'
  | 'raw'
  | 'error'

export interface ConversationBaseNode {
  id: string
  type: ConversationNodeType
  title: string
  summary?: string
  defaultCollapsed: boolean
  metadata?: Record<string, unknown>
}

export interface ConversationMessageNode extends ConversationBaseNode {
  type: ConversationMessageRole
  role: ConversationMessageRole
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
