import { describe, expect, it } from 'vitest'
import { formatConversationAsText, formatHumanBytes } from '../format'
import type { ConversationFlow } from '../types'

describe('formatHumanBytes', () => {
  it('formats bytes as human-readable B, KB, and MB values', () => {
    expect(formatHumanBytes(0)).toBe('0 B')
    expect(formatHumanBytes(512)).toBe('512 B')
    expect(formatHumanBytes(1024)).toBe('1.0 KB')
    expect(formatHumanBytes(1536)).toBe('1.5 KB')
    expect(formatHumanBytes(12288)).toBe('12 KB')
    expect(formatHumanBytes(1048576)).toBe('1.0 MB')
  })
})

describe('formatConversationAsText', () => {
  it('formatConversationAsText prepends system prompt block', () => {
    const flow: ConversationFlow = {
      source: 'client',
      format: 'openai-chat',
      warnings: [],
      nodes: [],
      systemPrompt: {
        id: 'system-prompt-0',
        text: 'You are a helpful assistant.',
        sources: ['developer'],
      },
      messages: [
        {
          id: 'msg-0',
          role: 'user',
          parts: [{ id: 'p-0', type: 'text', text: 'Hello' }],
        },
      ],
    }
    const result = formatConversationAsText(flow)
    expect(result).toContain('[system prompt]')
    expect(result).toContain('You are a helpful assistant.')
    expect(result.indexOf('[system prompt]')).toBeLessThan(result.indexOf('[user]'))
  })

  it('formatConversationAsText includes injection label in user message', () => {
    const flow: ConversationFlow = {
      source: 'client',
      format: 'openai-chat',
      warnings: [],
      nodes: [],
      messages: [
        {
          id: 'msg-0',
          role: 'user',
          parts: [
            { id: 'p-0', type: 'injection', tag: 'EXTREMELY_IMPORTANT', text: '<EXTREMELY_IMPORTANT>\nRules\n</EXTREMELY_IMPORTANT>', defaultCollapsed: true },
            { id: 'p-1', type: 'text', text: 'What is 2+2?' },
          ],
        },
      ],
    }
    const result = formatConversationAsText(flow)
    expect(result).toContain('[injection: EXTREMELY_IMPORTANT]')
    expect(result).toContain('<EXTREMELY_IMPORTANT>')
    expect(result).toContain('[text]')
    expect(result).toContain('What is 2+2?')
  })

  it('formatConversationAsText omits system prompt block when not present', () => {
    const flow: ConversationFlow = {
      source: 'client',
      format: 'openai-chat',
      warnings: [],
      nodes: [],
      messages: [
        {
          id: 'msg-0',
          role: 'user',
          parts: [{ id: 'p-0', type: 'text', text: 'Hello' }],
        },
      ],
    }
    const result = formatConversationAsText(flow)
    expect(result).not.toContain('[system prompt]')
    expect(result).toContain('[user]')
  })
})
