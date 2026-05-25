import { describe, expect, it } from 'vitest'
import { formatConversationAsText, formatRawValue, parseJsonValue, summarizeText } from '../format'
import { parseConversationPayload } from '../parseConversationPayload'
import type { ConversationNode } from '../types'

describe('conversation format helpers', () => {
  it('parses valid JSON and returns null for invalid JSON', () => {
    expect(parseJsonValue('{"messages":[]}')).toEqual({ messages: [] })
    expect(parseJsonValue('null')).toBeNull()
    expect(parseJsonValue('not-json')).toBeNull()
    expect(parseJsonValue(null)).toBeNull()
  })

  it('formats raw values predictably', () => {
    function sampleFunction() {
      return true
    }

    expect(formatRawValue('{"b":2,"a":1}')).toBe(`{\n  "b": 2,\n  "a": 1\n}`)
    expect(formatRawValue('plain text')).toBe('plain text')
    expect(formatRawValue({ ok: true })).toBe(`{\n  "ok": true\n}`)
    expect(formatRawValue(Symbol('value'))).toBe('Symbol(value)')
    expect(formatRawValue(sampleFunction)).toBe(String(sampleFunction))
  })

  it('summarizes long text without dropping short text', () => {
    expect(summarizeText('short text', 20)).toBe('short text')
    expect(summarizeText('abcdefghijklmnopqrstuvwxyz', 10)).toBe('abcdefg...')
    expect(summarizeText('abcdefghijklmnopqrstuvwxyz', 0)).toBe('')
    expect(summarizeText('abcdefghijklmnopqrstuvwxyz', 1)).toBe('a')
    expect(summarizeText('abcdefghijklmnopqrstuvwxyz', 2)).toBe('ab')
  })

  it('formats conversation nodes as plain text for copying', () => {
    const text = formatConversationAsText({
      source: 'client',
      format: 'openai-chat',
      warnings: [],
      nodes: [
        {
          id: 'n1',
          type: 'user',
          role: 'user',
          title: 'User',
          defaultCollapsed: false,
          parts: [{ type: 'text', text: 'Hello' }],
        },
        {
          id: 'n2',
          type: 'tool_call',
          title: 'Tool Call · web_search',
          defaultCollapsed: true,
          toolName: 'web_search',
          input: { query: 'timeout' },
        },
      ],
    })

    expect(text).toContain('[user]')
    expect(text).toContain('Hello')
    expect(text).toContain('[tool_call: web_search]')
    expect(text).toContain('"query": "timeout"')
  })
})

describe('parseConversationPayload', () => {
  it('parses OpenAI chat history and appends response assistant message', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        model: 'gpt-4.1',
        messages: [
          { role: 'system', content: 'Be concise.' },
          { role: 'user', content: 'Hello' },
          { role: 'assistant', content: 'Hi. How can I help?' },
          { role: 'user', content: [{ type: 'text', text: 'Analyze this image' }, { type: 'image_url', image_url: { url: 'https://example.com/a.png' } }] },
        ],
      }),
      responseBody: JSON.stringify({
        choices: [{ message: { role: 'assistant', content: '**Done**\n\n```ts\nconst ok = true\n```' } }],
      }),
    })

    expect(flow.format).toBe('openai-chat')
    expect(flow.nodes.map((node) => node.type)).toEqual(['system', 'user', 'assistant', 'user', 'assistant'])
    expect(flow.nodes[0].defaultCollapsed).toBe(true)
    expect(flow.nodes[1].defaultCollapsed).toBe(false)
    expect(flow.nodes[3]).toMatchObject({ type: 'user' })
    expect('parts' in flow.nodes[3] ? flow.nodes[3].parts.some((part) => part.type === 'image') : false).toBe(true)
  })

  it('creates independent tool call and tool result cards', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [
          { role: 'assistant', content: 'Searching', tool_calls: [{ id: 'call_1', type: 'function', function: { name: 'web_search', arguments: '{"query":"timeout"}' } }] },
          { role: 'tool', tool_call_id: 'call_1', name: 'web_search', content: '{"ok":true}' },
        ],
      }),
      responseBody: null,
    })

    expect(flow.nodes.map((node) => node.type)).toEqual(['assistant', 'tool_call', 'tool_result'])
    expect(flow.nodes[1]).toMatchObject({ type: 'tool_call', toolName: 'web_search', defaultCollapsed: true })
    expect(flow.nodes[2]).toMatchObject({ type: 'tool_result', toolName: 'web_search', defaultCollapsed: true })
  })

  it('skips blank assistant messages while preserving tool call cards', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [
          { role: 'assistant', content: null, tool_calls: [{ id: 'call_empty', type: 'function', function: { name: 'lookup', arguments: '{"id":1}' } }] },
        ],
      }),
      responseBody: null,
    })

    expect(flow.nodes.map((node) => node.type)).toEqual(['tool_call'])
    expect(flow.nodes[0]).toMatchObject({ type: 'tool_call', toolName: 'lookup' })
  })

  it('keeps raw response when known format response JSON is invalid', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [{ role: 'user', content: 'Hello' }],
      }),
      responseBody: 'not-json',
    })

    expect(flow.format).toBe('openai-chat')
    expect(flow.warnings).toContain('Response body is not valid JSON.')
    expect(flow.nodes.map((node) => node.type)).toEqual(['user', 'raw'])
    expect(flow.nodes[1]).toMatchObject({ type: 'raw', title: 'Raw Response', defaultCollapsed: true, raw: 'not-json' })
  })

  it('keeps unrecognized chat messages as collapsed raw nodes', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [
          { role: 'user', content: 'Hello' },
          'orphan message',
          { role: 'observer', content: 'visible unknown role' },
          { role: 'assistant', content: null },
        ],
      }),
      responseBody: null,
    })

    expect(flow.format).toBe('openai-chat')
    expect(flow.nodes.map((node) => node.type)).toEqual(['user', 'raw', 'raw', 'raw'])
    expect(flow.nodes.slice(1)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ type: 'raw', defaultCollapsed: true, raw: 'orphan message' }),
        expect.objectContaining({ type: 'raw', defaultCollapsed: true, raw: expect.stringContaining('visible unknown role') }),
        expect.objectContaining({ type: 'raw', defaultCollapsed: true, raw: expect.stringContaining('assistant') }),
      ])
    )
  })

  it('keeps unrecognized chat response choices as collapsed raw nodes', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [{ role: 'user', content: 'Hello' }],
      }),
      responseBody: JSON.stringify({
        choices: [
          { message: { role: 'assistant', content: 'Known' } },
          { custom_choice: { value: 1 } },
          'legacy-choice',
        ],
      }),
    })

    expect(flow.format).toBe('openai-chat')
    expect(flow.nodes.map((node) => node.type)).toEqual(['user', 'assistant', 'raw', 'raw'])
    expect(flow.nodes[1]).toMatchObject({ type: 'assistant', summary: 'Known' })
    expect(flow.nodes[2]).toMatchObject({
      type: 'raw',
      defaultCollapsed: true,
      raw: expect.stringContaining('custom_choice'),
      metadata: expect.objectContaining({ rawSource: 'response', nestedSource: 'choices' }),
    })
    expect(flow.nodes[3]).toMatchObject({
      type: 'raw',
      defaultCollapsed: true,
      raw: 'legacy-choice',
      metadata: expect.objectContaining({ rawSource: 'response', nestedSource: 'choices' }),
    })
  })

  it('keeps null chat response choices as non-empty collapsed raw nodes', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [{ role: 'user', content: 'Hello' }],
      }),
      responseBody: JSON.stringify({
        choices: [null],
      }),
    })

    expect(flow.format).toBe('openai-chat')
    expect(flow.nodes.map((node) => node.type)).toEqual(['user', 'raw'])
    expect(flow.nodes[1]).toMatchObject({
      type: 'raw',
      defaultCollapsed: true,
      raw: 'null',
      metadata: expect.objectContaining({ rawSource: 'response', nestedSource: 'choices' }),
    })
  })

  it('parses OpenAI Responses input and output', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        input: [
          { role: 'user', content: [{ type: 'input_text', text: 'Summarize logs' }] },
        ],
      }),
      responseBody: JSON.stringify({
        output: [
          { type: 'message', role: 'assistant', content: [{ type: 'output_text', text: 'Summary complete.' }] },
          { type: 'function_call', call_id: 'call_2', name: 'read_log', arguments: '{"file":"app.log"}' },
        ],
      }),
    })

    expect(flow.format).toBe('openai-responses')
    expect(flow.nodes.map((node) => node.type)).toEqual(['user', 'assistant', 'tool_call'])
    expect(flow.nodes[2]).toMatchObject({ type: 'tool_call', toolName: 'read_log' })
  })

  it('keeps unrecognized Responses input and output items as collapsed raw nodes', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        input: [
          { role: 'user', content: [{ type: 'input_text', text: 'Summarize logs' }] },
          { type: 'custom_input', payload: { id: 1 } },
        ],
      }),
      responseBody: JSON.stringify({
        output: [
          { type: 'message', role: 'assistant', content: [{ type: 'output_text', text: 'Summary complete.' }] },
          { type: 'custom_output', payload: { id: 2 } },
        ],
      }),
    })

    expect(flow.format).toBe('openai-responses')
    expect(flow.nodes.map((node) => node.type)).toEqual(['user', 'raw', 'assistant', 'raw'])
    expect(flow.nodes[1]).toMatchObject({ type: 'raw', defaultCollapsed: true, raw: expect.stringContaining('custom_input') })
    expect(flow.nodes[3]).toMatchObject({ type: 'raw', defaultCollapsed: true, raw: expect.stringContaining('custom_output') })
  })

  it('renders Responses reasoning summary text as collapsed assistant text card and hides encrypted content', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        input: [{ role: 'user', content: [{ type: 'input_text', text: 'Translate' }] }],
      }),
      responseBody: JSON.stringify({
        output: [
          {
            type: 'reasoning',
            id: 'rs-1',
            summary: [
              { type: 'summary_text', text: '考虑到这是一个翻译任务' },
              { type: 'summary_text', text: '需要保持原文语气' },
            ],
            encrypted_content: '...',
          },
          { type: 'message', role: 'assistant', content: [{ type: 'output_text', text: 'Hello' }] },
        ],
      }),
    })

    expect(flow.format).toBe('openai-responses')
    expect(flow.nodes.map((node) => node.type)).toEqual(['user', 'assistant', 'assistant'])
    const reasoningNode = flow.nodes[1] as Extract<ConversationNode, { type: 'assistant' }>
    expect(reasoningNode.type).toBe('assistant')
    expect(reasoningNode.defaultCollapsed).toBe(true)
    expect(reasoningNode.parts.every((p) => p.type === 'text')).toBe(true)
    expect(reasoningNode.parts.map((p) => (p as { text: string }).text).join('\n')).toContain('翻译任务')
  })

  it('keeps reasoning item as raw node when summary has no displayable text', () => {
    const flow = parseConversationPayload({
      requestBody: null,
      responseBody: JSON.stringify({
        output: [
          { type: 'reasoning', id: 'rs-2', summary: [], encrypted_content: '...' },
        ],
      }),
    })

    expect(flow.format).toBe('openai-responses')
    expect(flow.nodes.map((node) => node.type)).toEqual(['raw'])
  })

  it('falls back to raw nodes for unrecognized payloads and empty nodes for empty input', () => {
    const rawFlow = parseConversationPayload({ requestBody: '{"foo":1}', responseBody: 'not-json' })
    expect(rawFlow.format).toBe('unknown')
    expect(rawFlow.nodes.map((node) => node.type)).toEqual(['raw', 'raw'])
    expect(rawFlow.nodes.every((node) => node.defaultCollapsed)).toBe(true)

    const emptyFlow = parseConversationPayload({ requestBody: null, responseBody: null })
    expect(emptyFlow.nodes).toEqual([])
  })
})
