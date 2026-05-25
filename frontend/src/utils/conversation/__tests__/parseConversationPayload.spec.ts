import { describe, expect, it } from 'vitest'
import { formatConversationAsText, formatRawValue, parseJsonValue, summarizeText } from '../format'
import { parseConversationPayload } from '../parseConversationPayload'
import { getToolDisplayName, getToolLabel } from '../toolDisplay'
import type { ConversationPart, ConversationToolPart } from '../types'

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

  it('maps common tool labels like webgui', () => {
    expect(getToolLabel('bash')).toBe('执行命令')
    expect(getToolLabel('read')).toBe('查看')
    expect(getToolLabel('grep')).toBe('文本查找')
    expect(getToolLabel('webfetch')).toBe('抓取网页')
    expect(getToolLabel('unknown_tool')).toBe('unknown_tool')
  })

  it('builds webgui-style tool display names without call ids', () => {
    expect(getToolDisplayName({ tool: 'bash', input: { description: '列出文件', command: 'ls' } })).toBe('执行命令：列出文件')
    expect(getToolDisplayName({ tool: 'read', input: { filePath: 'src/app.ts' } })).toBe('查看：src/app.ts')
    expect(getToolDisplayName({ tool: 'grep', input: { pattern: 'timeout', include: '*.ts' } })).toBe('文本查找：timeout (*.ts)')
    expect(getToolDisplayName({ tool: 'webfetch', input: { url: 'https://example.com' } })).toBe('抓取网页：https://example.com')
    expect(getToolDisplayName({ tool: 'bash', callId: 'call_123', input: { command: 'pwd' } } satisfies Partial<ConversationToolPart> & { tool: string })).toBe('执行命令')
  })

  it('builds todo tool display names from JSON output arrays', () => {
    expect(getToolDisplayName({ tool: 'todowrite', output: '[{"status":"completed"},{"status":"pending"}]' })).toBe('更新任务列表：已完成 1/2')
    expect(getToolDisplayName({ tool: 'todowrite', output: [{ status: 'completed' }, { status: 'pending' }] })).toBe('更新任务列表：已完成 1/2')
    expect(getToolDisplayName({ tool: 'todoread', output: '[{"status":"pending"},{"status":"in_progress"}]' })).toBe('读取任务列表：共 2 项')
    expect(getToolDisplayName({ tool: 'todowrite', output: 'not-json' })).toBe('更新任务列表')
  })

  it('formats conversation messages and merged tool parts as plain text for copying', () => {
    const text = formatConversationAsText({
      source: 'client',
      format: 'openai-chat',
      warnings: [],
      nodes: [],
      messages: [
        {
          id: 'message-user-0',
          role: 'user',
          parts: [{ id: 'part-text-0', type: 'text', text: 'Hello' }],
        },
        {
          id: 'message-assistant-1',
          role: 'assistant',
          parts: [
            { id: 'part-reasoning-1', type: 'reasoning', text: 'Thinking', defaultCollapsed: true },
            {
              id: 'part-tool-2',
              type: 'tool',
              tool: 'bash',
              callId: 'call_ignored_in_title',
              state: { status: 'completed', input: { command: 'pwd' }, output: '/repo' },
            },
            { id: 'part-text-3', type: 'text', text: 'Done' },
          ],
        },
      ],
    })

    expect(text).toContain('[user]')
    expect(text).toContain('Hello')
    expect(text).toContain('[assistant]')
    expect(text).toContain('[reasoning]')
    expect(text).toContain('Thinking')
    expect(text).toContain('[tool: bash]')
    expect(text).toContain('$ pwd')
    expect(text).toContain('/repo')
    expect(text).not.toContain('call_ignored_in_title')
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
    expect(flow.messages?.map((message) => message.role)).toEqual(['system', 'user', 'assistant', 'user', 'assistant'])
    expect(flow.messages?.[0].parts).toMatchObject([{ type: 'text', text: 'Be concise.' }])
    expect(flow.messages?.[3].parts.some((part) => part.type === 'image')).toBe(true)
    expect(flow.messages?.[4].parts).toMatchObject([{ type: 'text', text: '**Done**\n\n```ts\nconst ok = true\n```' }])
  })

  it('merges OpenAI chat tool calls and tool results into one tool part', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [
          {
            role: 'assistant',
            content: 'Searching',
            tool_calls: [
              { id: 'call_1', type: 'function', function: { name: 'web_search', arguments: '{"query":"timeout"}' } },
            ],
          },
          { role: 'tool', tool_call_id: 'call_1', name: 'web_search', content: '{"ok":true}' },
        ],
      }),
      responseBody: null,
    })

    expect(flow.messages).toHaveLength(1)
    expect(flow.messages?.[0].role).toBe('assistant')
    expect(flow.messages?.[0].parts.map((part) => part.type)).toEqual(['text', 'tool'])
    const tool = flow.messages?.[0].parts[1] as ConversationToolPart
    expect(tool).toMatchObject({ type: 'tool', tool: 'web_search', callId: 'call_1' })
    expect(tool.state.input).toEqual({ query: 'timeout' })
    expect(tool.state.output).toEqual({ ok: true })
    expect(tool.state.status).toBe('completed')
  })

  it('does not let generic OpenAI chat tool titles override input descriptions', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [
          {
            role: 'assistant',
            content: null,
            tool_calls: [
              { id: 'call_1', type: 'function', function: { name: 'bash', arguments: '{"command":"pwd","description":"显示当前目录"}' } },
            ],
          },
          { role: 'tool', tool_call_id: 'call_1', name: 'bash', content: '/repo' },
        ],
      }),
      responseBody: null,
    })

    const tool = flow.messages?.[0].parts[0] as ConversationToolPart
    expect(tool.state.title).toBeUndefined()
    expect(getToolDisplayName({
      tool: tool.tool,
      input: tool.state.input,
      title: tool.state.title,
      output: tool.state.output,
    })).toBe('执行命令：显示当前目录')
  })

  it('preserves explicit OpenAI chat tool titles for display names', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [
          {
            role: 'assistant',
            content: null,
            tool_calls: [
              {
                id: 'call_1',
                type: 'function',
                title: '自定义标题',
                function: { name: 'bash', arguments: '{"command":"pwd","description":"显示当前目录"}' },
              },
            ],
          },
          { role: 'tool', tool_call_id: 'call_1', name: 'bash', content: '/repo' },
        ],
      }),
      responseBody: null,
    })

    const tool = flow.messages?.[0].parts[0] as ConversationToolPart
    expect(tool.state.title).toBe('自定义标题')
    expect(getToolDisplayName({
      tool: tool.tool,
      input: tool.state.input,
      title: tool.state.title,
      output: tool.state.output,
    })).toBe('执行命令：自定义标题')
  })

  it('pairs multiple tool results by call id regardless of result order', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [
          {
            role: 'assistant',
            content: null,
            tool_calls: [
              { id: 'call_a', type: 'function', function: { name: 'read', arguments: '{"filePath":"a.ts"}' } },
              { id: 'call_b', type: 'function', function: { name: 'grep', arguments: '{"pattern":"x","include":"*.ts"}' } },
            ],
          },
          { role: 'tool', tool_call_id: 'call_b', name: 'grep', content: 'matched' },
          { role: 'tool', tool_call_id: 'call_a', name: 'read', content: 'file contents' },
        ],
      }),
      responseBody: null,
    })

    const tools = flow.messages?.[0].parts.filter((part): part is ConversationToolPart => part.type === 'tool') ?? []
    expect(tools).toHaveLength(2)
    expect(tools.find((tool) => tool.callId === 'call_a')?.state.output).toBe('file contents')
    expect(tools.find((tool) => tool.callId === 'call_b')?.state.output).toBe('matched')
  })

  it('keeps unmatched tool results as orphan tool parts', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({ messages: [{ role: 'tool', tool_call_id: 'missing', name: 'lookup', content: '{"value":1}' }] }),
      responseBody: null,
    })

    expect(flow.messages).toHaveLength(1)
    expect(flow.messages?.[0].role).toBe('assistant')
    const tool = flow.messages?.[0].parts[0] as ConversationToolPart
    expect(tool.type).toBe('tool')
    expect(tool.tool).toBe('lookup')
    expect(tool.callId).toBe('missing')
    expect(tool.state.input).toBeUndefined()
    expect(tool.state.output).toEqual({ value: 1 })
  })

  it('keeps blank assistant messages when they contain tool parts', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [
          { role: 'assistant', content: null, tool_calls: [{ id: 'call_empty', type: 'function', function: { name: 'lookup', arguments: '{"id":1}' } }] },
        ],
      }),
      responseBody: null,
    })

    expect(flow.messages?.map((message) => message.role)).toEqual(['assistant'])
    expect(flow.messages?.[0].parts).toMatchObject([{ type: 'tool', tool: 'lookup', callId: 'call_empty' }])
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
    expect(flow.messages?.map((message) => message.role)).toEqual(['user', 'assistant'])
    expect(flow.messages?.[1].parts).toMatchObject([{ type: 'raw', title: 'Raw Response', defaultCollapsed: true, raw: 'not-json' }])
  })

  it('keeps unrecognized chat messages as collapsed raw parts', () => {
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
    expect(flow.messages?.map((message) => message.role)).toEqual(['user', 'assistant', 'assistant', 'assistant'])
    expect(flow.messages?.slice(1).map((message) => message.parts[0])).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ type: 'raw', defaultCollapsed: true, raw: 'orphan message' }),
        expect.objectContaining({ type: 'raw', defaultCollapsed: true, raw: expect.stringContaining('visible unknown role') }),
        expect.objectContaining({ type: 'raw', defaultCollapsed: true, raw: expect.stringContaining('assistant') }),
      ])
    )
  })

  it('keeps unrecognized chat response choices as collapsed raw parts', () => {
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
    expect(flow.messages?.map((message) => message.role)).toEqual(['user', 'assistant', 'assistant', 'assistant'])
    expect(flow.messages?.[1].parts).toMatchObject([{ type: 'text', text: 'Known' }])
    expect(flow.messages?.[2].parts[0]).toMatchObject({
      type: 'raw',
      defaultCollapsed: true,
      raw: expect.stringContaining('custom_choice'),
      metadata: expect.objectContaining({ rawSource: 'response', nestedSource: 'choices' }),
    })
    expect(flow.messages?.[3].parts[0]).toMatchObject({
      type: 'raw',
      defaultCollapsed: true,
      raw: 'legacy-choice',
      metadata: expect.objectContaining({ rawSource: 'response', nestedSource: 'choices' }),
    })
  })

  it('keeps null chat response choices as non-empty collapsed raw parts', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [{ role: 'user', content: 'Hello' }],
      }),
      responseBody: JSON.stringify({
        choices: [null],
      }),
    })

    expect(flow.format).toBe('openai-chat')
    expect(flow.messages?.map((message) => message.role)).toEqual(['user', 'assistant'])
    expect(flow.messages?.[1].parts[0]).toMatchObject({
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
    expect(flow.messages?.map((message) => message.role)).toEqual(['user', 'assistant'])
    expect(flow.messages?.[1].parts.map((part) => part.type)).toEqual(['text', 'tool'])
    expect(flow.messages?.[1].parts[1]).toMatchObject({ type: 'tool', tool: 'read_log', callId: 'call_2' })
  })

  it('uses Responses output_text only as fallback when output has no assistant text', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({ input: [{ role: 'user', content: [{ type: 'input_text', text: 'Say hi' }] }] }),
      responseBody: JSON.stringify({
        output: [
          { type: 'message', role: 'assistant', content: [{ type: 'output_text', text: 'Hi' }] },
        ],
        output_text: 'Hi',
      }),
    })

    expect(flow.format).toBe('openai-responses')
    expect(flow.messages?.map((message) => message.role)).toEqual(['user', 'assistant'])
    expect(flow.messages?.[1].parts).toMatchObject([{ type: 'text', text: 'Hi' }])
    expect(flow.messages?.[1].parts).toHaveLength(1)
  })

  it('keeps Responses output_text fallback when output has only non-answer assistant parts', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({ input: [{ role: 'user', content: [{ type: 'input_text', text: 'Summarize logs' }] }] }),
      responseBody: JSON.stringify({
        output: [
          { type: 'reasoning', summary: [{ type: 'summary_text', text: '检查日志。' }] },
          { type: 'function_call', call_id: 'call_2', name: 'read', arguments: '{"filePath":"app.log"}' },
        ],
        output_text: 'Summary complete.',
      }),
    })

    expect(flow.messages?.[1].parts.map((part) => part.type)).toEqual(['reasoning', 'tool', 'text'])
    expect(flow.messages?.[1].parts[2]).toMatchObject({ type: 'text', text: 'Summary complete.' })
  })

  it('keeps unrecognized Responses input and output items as collapsed raw parts', () => {
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
    expect(flow.messages?.map((message) => message.role)).toEqual(['user', 'user', 'assistant', 'assistant'])
    expect(flow.messages?.[1].parts[0]).toMatchObject({ type: 'raw', defaultCollapsed: true, raw: expect.stringContaining('custom_input') })
    expect(flow.messages?.[3].parts[0]).toMatchObject({ type: 'raw', defaultCollapsed: true, raw: expect.stringContaining('custom_output') })
  })

  it('parses Responses reasoning and paired function call output as parts', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({ input: [{ role: 'user', content: [{ type: 'input_text', text: 'Summarize logs' }] }] }),
      responseBody: JSON.stringify({
        output: [
          { type: 'reasoning', summary: [{ type: 'summary_text', text: '检查日志。' }], encrypted_content: 'secret' },
          { type: 'function_call', call_id: 'call_2', name: 'read', arguments: '{"filePath":"app.log"}' },
          { type: 'function_call_output', call_id: 'call_2', output: 'done' },
          { type: 'message', role: 'assistant', content: [{ type: 'output_text', text: 'Summary complete.' }] },
        ],
      }),
    })

    expect(flow.messages?.map((message) => message.role)).toEqual(['user', 'assistant'])
    expect(flow.messages?.[1].parts.map((part) => part.type)).toEqual(['reasoning', 'tool', 'text'])
    expect((flow.messages?.[1].parts[0] as { text: string }).text).toBe('检查日志。')
    expect(JSON.stringify(flow.messages?.[1])).not.toContain('secret')
    const tool = flow.messages?.[1].parts[1] as ConversationToolPart
    expect(tool.state.input).toEqual({ filePath: 'app.log' })
    expect(tool.state.output).toBe('done')
  })

  it('keeps Responses response parts separate from assistant messages in request history', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        input: [
          { role: 'user', content: [{ type: 'input_text', text: 'Continue' }] },
          { role: 'assistant', content: [{ type: 'output_text', text: 'Previous assistant answer.' }] },
        ],
      }),
      responseBody: JSON.stringify({
        output: [
          { type: 'reasoning', summary: [{ type: 'summary_text', text: 'Need fresh answer.' }] },
          { type: 'function_call', call_id: 'response_call', name: 'read', arguments: '{"filePath":"new.log"}' },
          { type: 'function_call_output', call_id: 'response_call', output: 'new contents' },
          { type: 'message', role: 'assistant', content: [{ type: 'output_text', text: 'New assistant answer.' }] },
        ],
      }),
    })

    expect(flow.messages?.map((message) => message.role)).toEqual(['user', 'assistant', 'assistant'])
    expect(flow.messages?.[1].parts.map((part) => part.type)).toEqual(['text'])
    expect(flow.messages?.[1].parts[0]).toMatchObject({ type: 'text', text: 'Previous assistant answer.' })
    expect(flow.messages?.[2].parts.map((part) => part.type)).toEqual(['reasoning', 'tool', 'text'])
  })

  it('does not pair Responses response tool outputs with request-history tool calls that reuse a call id', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        input: [
          { type: 'function_call', call_id: 'reused_call', name: 'read', arguments: '{"filePath":"old.log"}' },
          { type: 'function_call_output', call_id: 'reused_call', output: 'old contents' },
        ],
      }),
      responseBody: JSON.stringify({
        output: [
          { type: 'function_call_output', call_id: 'reused_call', output: 'new contents' },
        ],
      }),
    })

    expect(flow.messages).toHaveLength(2)
    const requestTool = flow.messages?.[0].parts[0] as ConversationToolPart
    const responseTool = flow.messages?.[1].parts[0] as ConversationToolPart
    expect(requestTool.state.output).toBe('old contents')
    expect(responseTool).toMatchObject({ type: 'tool', callId: 'reused_call' })
    expect(responseTool.state.input).toBeUndefined()
    expect(responseTool.state.output).toBe('new contents')
  })

  it('redacts nested encrypted content from raw Responses items and tool metadata', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({ input: [{ role: 'user', content: [{ type: 'input_text', text: 'Check secrets' }] }] }),
      responseBody: JSON.stringify({
        output: [
          { type: 'custom_output', payload: { nested: { encrypted_content: 'raw-secret', keep: true } } },
          { type: 'function_call', call_id: 'secret_call', name: 'lookup', arguments: '{}', extra: { encrypted_content: 'tool-secret' } },
          { type: 'function_call_output', call_id: 'secret_call', output: { value: 1, nested: { encrypted_content: 'result-secret' } } },
        ],
      }),
    })

    const serialized = JSON.stringify(flow.messages)
    expect(serialized).not.toContain('raw-secret')
    expect(serialized).not.toContain('tool-secret')
    expect(serialized).not.toContain('result-secret')
  })

  it('redacts encrypted content from unknown-format raw fallback JSON strings', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({ payload: { encrypted_content: 'unknown-request-secret', keep: true } }),
      responseBody: JSON.stringify({ encrypted_content: 'unknown-response-secret', ok: true }),
    })

    const serialized = JSON.stringify(flow.messages)
    expect(serialized).not.toContain('encrypted_content')
    expect(serialized).not.toContain('unknown-request-secret')
    expect(serialized).not.toContain('unknown-response-secret')
  })

  it('redacts encrypted content from known-format raw response strings', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({ messages: [{ role: 'user', content: 'Hello' }] }),
      responseBody: JSON.stringify({ encrypted_content: 'known-response-secret', ok: true }),
    })

    const serialized = JSON.stringify(flow.messages)
    expect(serialized).not.toContain('encrypted_content')
    expect(serialized).not.toContain('known-response-secret')
  })

  it('redacts encrypted content from JSON strings in tool metadata', () => {
    const flow = parseConversationPayload({
      requestBody: JSON.stringify({
        messages: [
          {
            role: 'assistant',
            content: null,
            tool_calls: [
              { id: 'call_secret', type: 'function', function: { name: 'lookup', arguments: '{"encrypted_content":"argument-secret","keep":true}' } },
            ],
          },
          { role: 'tool', tool_call_id: 'call_secret', name: 'lookup', content: '{"encrypted_content":"content-secret","ok":true}' },
        ],
      }),
      responseBody: null,
    })

    const serialized = JSON.stringify(flow.messages)
    expect(serialized).not.toContain('encrypted_content')
    expect(serialized).not.toContain('argument-secret')
    expect(serialized).not.toContain('content-secret')
  })

  it('keeps reasoning item as raw part when summary has no displayable text', () => {
    const flow = parseConversationPayload({
      requestBody: null,
      responseBody: JSON.stringify({
        output: [
          { type: 'reasoning', id: 'rs-2', summary: [], encrypted_content: '...' },
        ],
      }),
    })

    expect(flow.format).toBe('openai-responses')
    expect(flow.messages?.map((message) => message.role)).toEqual(['assistant'])
    expect(flow.messages?.[0].parts.map((part) => part.type)).toEqual(['raw'])
  })

  it('falls back to raw parts for unrecognized payloads and empty messages for empty input', () => {
    const rawFlow = parseConversationPayload({ requestBody: '{"foo":1}', responseBody: 'not-json' })
    expect(rawFlow.format).toBe('unknown')
    const rawParts = rawFlow.messages?.flatMap((message) => message.parts) as ConversationPart[]
    expect(rawParts.map((part) => part.type)).toEqual(['raw', 'raw'])
    expect(rawParts.every((part) => part.type === 'raw' && part.defaultCollapsed)).toBe(true)

    const emptyFlow = parseConversationPayload({ requestBody: null, responseBody: null })
    expect(emptyFlow.messages).toEqual([])
  })
})
