import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ConversationTimeline from '../ConversationTimeline.vue'
import messageRowSource from '../ConversationMessageRow.vue?raw'
import type { ConversationFlow } from '@/utils/conversation/types'

const i18nMock = vi.hoisted(() => ({ locale: 'zh' }))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messagesByLocale: Record<string, Record<string, string>> = {
    zh: {
    'conversation.empty': 'No conversation content found',
    'conversation.expand': 'Expand',
    'conversation.collapse': 'Collapse',
    'conversation.toolCall': 'Tool Call',
    'conversation.toolResult': 'Tool Result',
    'conversation.reasoning': 'Reasoning',
    'conversation.raw': 'Raw',
    'conversation.rawRequest': 'Raw Request',
    'conversation.rawResponse': 'Raw Response',
    'conversation.error': 'Error',
    'conversation.systemPrompt.title': 'Developer instructions',
    'conversation.systemPrompt.segments': '{n} segments',
    'conversation.injection.title': 'System injection',
    'conversation.timelineLabel': 'Conversation timeline',
    'conversation.imageAlt': 'Conversation image',
    'conversation.role.you': '你',
    'conversation.role.user': 'User',
    'conversation.role.assistant': 'Assistant',
    'conversation.role.system': 'System',
    'conversation.role.developer': 'Developer',
    'conversation.tool': '工具',
    'conversation.toolInput': '输入',
    'conversation.toolOutput': '输出',
    'conversation.toolMetadata': '元数据',
    'conversation.toolLabels.bash': '执行命令',
    'conversation.toolLabels.read': '查看',
    'conversation.toolLabels.write': '写入',
    'conversation.toolLabels.edit': '编辑',
    'conversation.toolLabels.multiedit': '批量编辑',
    'conversation.toolLabels.grep': '文本查找',
    'conversation.toolLabels.glob': '路径匹配',
    'conversation.toolLabels.webfetch': '抓取网页',
      'conversation.toolLabels.task': '委派子任务',
      'conversation.reasoningMeta.collapsedLabel': 'Reasoning',
      'conversation.reasoningMeta.segments': '{n} segments',
      'conversation.toolMeta.error': 'error',
      'conversation.toolMeta.lines': '{n} lines',
      'conversation.toolMeta.sizeWithLines': '{lines} · {size}',
    },
    en: {
      'conversation.empty': 'No conversation content found',
      'conversation.expand': 'Expand',
      'conversation.collapse': 'Collapse',
      'conversation.toolCall': 'Tool Call',
      'conversation.toolResult': 'Tool Result',
      'conversation.reasoning': 'Reasoning',
      'conversation.raw': 'Raw',
      'conversation.rawRequest': 'Raw Request',
      'conversation.rawResponse': 'Raw Response',
      'conversation.error': 'Error',
      'conversation.systemPrompt.title': 'Developer instructions',
      'conversation.systemPrompt.segments': '{n} segments',
      'conversation.injection.title': 'System injection',
      'conversation.timelineLabel': 'Conversation timeline',
      'conversation.imageAlt': 'Conversation image',
      'conversation.role.you': 'You',
      'conversation.role.user': 'User',
      'conversation.role.assistant': 'Assistant',
      'conversation.role.system': 'System',
      'conversation.role.developer': 'Developer',
      'conversation.tool': 'Tool',
      'conversation.toolInput': 'Input',
      'conversation.toolOutput': 'Output',
      'conversation.toolMetadata': 'Metadata',
      'conversation.toolLabels.bash': 'Run Command',
      'conversation.reasoningMeta.collapsedLabel': 'Reasoning',
      'conversation.reasoningMeta.segments': '{n} segments',
      'conversation.toolMeta.error': 'error',
      'conversation.toolMeta.lines': '{n} lines',
      'conversation.toolMeta.sizeWithLines': '{lines} · {size}',
    },
  }

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, values?: Record<string, string | number>) => {
        const message = messagesByLocale[i18nMock.locale][key] ?? key
        return values
          ? Object.entries(values).reduce((text, [name, value]) => text.replace(`{${name}}`, String(value)), message)
          : message
      },
      te: (key: string) => Object.prototype.hasOwnProperty.call(messagesByLocale[i18nMock.locale], key),
    }),
  }
})

const createFlow = (messages: ConversationFlow['messages']): ConversationFlow => ({
  source: 'client',
  format: 'openai-chat',
  warnings: [],
  nodes: [],
  messages: messages ?? [],
})

describe('ConversationTimeline', () => {
  beforeEach(() => {
    i18nMock.locale = 'zh'
  })

  it('renders user and assistant message rows in webgui-like layout', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-user-1',
            role: 'user',
            parts: [{ id: 'part-text-user-1', type: 'text', text: 'Hello **world**' }],
          },
          {
            id: 'message-assistant-1',
            role: 'assistant',
            parts: [{ id: 'part-text-assistant-1', type: 'text', text: 'Hi' }],
          },
        ]),
      },
    })

    expect(wrapper.find('[data-test="conversation-message-row-user"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="conversation-message-row-assistant"]').exists()).toBe(true)
    expect(wrapper.get('ol').attributes('aria-label')).toBe('Conversation timeline')
    expect(wrapper.text()).toContain('你')
    expect(wrapper.text()).toContain('Hi')
    expect(wrapper.html()).toContain('<strong>world</strong>')
  })

  it('sorts assistant parts as reasoning, text, tool and renders one merged tool card', async () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-assistant-tool',
            role: 'assistant',
            parts: [
              {
                id: 'part-tool-1',
                type: 'tool',
                callId: 'call_hidden',
                tool: 'bash',
                state: {
                  status: 'completed',
                  input: { command: 'pwd', description: '显示当前目录' },
                  output: '/repo',
                  metadata: { hidden: 'metadata_hidden' },
                },
                metadata: { hidden: 'part_metadata_hidden' },
              },
              { id: 'part-text-1', type: 'text', text: 'Done' },
              { id: 'part-reasoning-1', type: 'reasoning', text: '思考内容', defaultCollapsed: true },
            ],
          },
        ]),
      },
    })

    expect(wrapper.findAll('[data-test^="conversation-part-"]').map((part) => part.attributes('data-test'))).toEqual([
      'conversation-part-reasoning',
      'conversation-part-text',
      'conversation-part-tool',
    ])
    expect(wrapper.text()).toContain('Done')
    expect(wrapper.text()).toContain('执行命令：显示当前目录')
    expect(wrapper.get('[data-test="conversation-tool-toggle"] .sr-only').text()).toBe('Completed')
    expect(wrapper.text()).not.toContain('call_hidden')

    await wrapper.get('[data-test="conversation-part-reasoning"] button').trigger('click')

    expect(wrapper.text()).toContain('思考内容')

    await wrapper.get('[data-test="conversation-tool-toggle"]').trigger('click')

    expect(wrapper.text()).toContain('$ pwd')
    expect(wrapper.text()).toContain('/repo')
    expect(wrapper.text()).not.toContain('call_hidden')
    expect(wrapper.text()).not.toContain('metadata_hidden')
    expect(wrapper.text()).not.toContain('part_metadata_hidden')
  })

  it('uses localized English tool labels in tool titles', () => {
    i18nMock.locale = 'en'

    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-assistant-tool-en',
            role: 'assistant',
            parts: [
              {
                id: 'part-tool-en',
                type: 'tool',
                tool: 'bash',
                state: {
                  status: 'completed',
                  input: { command: 'pwd', description: 'show current directory' },
                },
              },
            ],
          },
        ]),
      },
    })

    expect(wrapper.text()).toContain('Run Command：show current directory')
    expect(wrapper.text()).not.toContain('执行命令')
  })

  it('preserves original order for parts with the same priority', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-assistant-stable-sort',
            role: 'assistant',
            parts: [
              { id: 'part-text-first', type: 'text', text: 'First same-priority part' },
              { id: 'part-text-second', type: 'text', text: 'Second same-priority part' },
              { id: 'part-text-third', type: 'text', text: 'Third same-priority part' },
            ],
          },
        ]),
      },
    })

    expect(wrapper.findAll('[data-test="conversation-part-text"]').map((part) => part.text())).toEqual([
      'First same-priority part',
      'Second same-priority part',
      'Third same-priority part',
    ])
  })

  it('resets collapsed state when a reasoning part changes', async () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-assistant-reasoning',
            role: 'assistant',
            parts: [
              { id: 'part-reasoning-open', type: 'reasoning', text: 'Initially visible', defaultCollapsed: true },
            ],
          },
        ]),
      },
    })

    await wrapper.get('[data-test="conversation-part-reasoning"] button').trigger('click')

    expect(wrapper.text()).toContain('Initially visible')

    await wrapper.setProps({
      flow: createFlow([
        {
          id: 'message-assistant-reasoning',
          role: 'assistant',
          parts: [
            { id: 'part-reasoning-closed', type: 'reasoning', text: 'Should be hidden after prop update', defaultCollapsed: true },
          ],
        },
      ]),
    })

    expect(wrapper.text()).toContain('Reasoning')
    expect(wrapper.text()).not.toContain('Should be hidden after prop update')
  })

  it('filters unsafe media urls and keeps safe data images accessible', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-assistant-media-safety',
            role: 'assistant',
            parts: [
              { id: 'part-image-unsafe', type: 'image', src: 'javascript:alert(1)' },
              { id: 'part-image-safe', type: 'image', src: 'data:image/png;base64,AAAA' },
            ],
          },
        ]),
      },
    })

    const images = wrapper.findAll('[data-test="conversation-image"]')
    expect(images).toHaveLength(1)
    expect(images[0].attributes('src')).toBe('data:image/png;base64,AAAA')
    expect(images[0].attributes('alt')).toBe('Conversation image')
    expect(wrapper.html()).not.toContain('javascript:alert(1)')
  })

  it('sanitizes dangerous markdown html attributes', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-user-dangerous',
            role: 'user',
            parts: [{ id: 'part-text-dangerous', type: 'text', text: '<img src="x" onerror="alert(1)"> **safe**' }],
          },
        ]),
      },
    })

    expect(wrapper.html()).not.toContain('onerror')
    expect(wrapper.html()).toContain('<strong>safe</strong>')
  })

  it('does not render markdown images while still rendering explicit media parts', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-assistant-markdown-image',
            role: 'assistant',
            parts: [
              { id: 'part-text-markdown-image', type: 'text', text: '![x](https://example.com/a.png) <img src="https://example.com/a.png">' },
              { id: 'part-image-output', type: 'image', src: 'https://example.com/output.png', alt: 'output' },
            ],
          },
        ]),
      },
    })

    const images = wrapper.findAll('img')
    expect(images).toHaveLength(1)
    expect(images[0].attributes('data-test')).toBe('conversation-image')
    expect(images[0].attributes('src')).toBe('https://example.com/output.png')
    expect(wrapper.html()).not.toContain('https://example.com/a.png')
  })

  it('renders raw and error parts as collapsed fallback panels', async () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-assistant-raw-error',
            role: 'assistant',
            parts: [
              { id: 'part-raw-1', type: 'raw', title: 'Raw payload', raw: '{"ok":true}', defaultCollapsed: true },
              { id: 'part-error-1', type: 'error', error: 'Boom', raw: 'stack', defaultCollapsed: true },
            ],
          },
        ]),
      },
    })

    expect(wrapper.findAll('[data-test="conversation-part-raw"]')).toHaveLength(2)
    expect(wrapper.text()).toContain('Raw payload')
    expect(wrapper.text()).toContain('Error')
    expect(wrapper.text()).not.toContain('{"ok":true}')
    expect(wrapper.text()).not.toContain('Boom')

    await wrapper.findAll('[data-test="conversation-raw-toggle"]')[1].trigger('click')

    expect(wrapper.text()).toContain('Boom')
    expect(wrapper.text()).toContain('stack')
  })

  it('renders system prompt bar collapsed by default', () => {
    const flow: ConversationFlow = {
      source: 'client',
      format: 'openai-chat',
      warnings: [],
      nodes: [],
      systemPrompt: { id: 'sp-0', text: 'You are helpful.', sources: ['developer'] },
      messages: [{ id: 'msg-0', role: 'user', parts: [{ id: 'p-0', type: 'text', text: 'Hi' }] }],
    }

    const wrapper = mount(ConversationTimeline, {
      props: { flow },
    })

    expect(wrapper.find('[data-test="conversation-system-prompt-bar"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('You are helpful.')
  })

  it('system prompt bar toggles open and shows markdown content', async () => {
    const flow: ConversationFlow = {
      source: 'client',
      format: 'openai-chat',
      warnings: [],
      nodes: [],
      systemPrompt: { id: 'sp-0', text: 'You are helpful.', sources: ['developer'] },
      messages: [{ id: 'msg-0', role: 'user', parts: [{ id: 'p-0', type: 'text', text: 'Hi' }] }],
    }

    const wrapper = mount(ConversationTimeline, {
      props: { flow },
    })

    await wrapper.get('[data-test="conversation-system-prompt-bar"] button').trigger('click')

    expect(wrapper.text()).toContain('You are helpful.')
  })

  it('does not render developer/system as message rows', () => {
    const flow: ConversationFlow = {
      source: 'client',
      format: 'openai-chat',
      warnings: [],
      nodes: [],
      systemPrompt: { id: 'sp-0', text: 'Developer instructions', sources: ['developer', 'system'] },
      messages: [{ id: 'msg-0', role: 'user', parts: [{ id: 'p-0', type: 'text', text: 'Hi' }] }],
    }

    const wrapper = mount(ConversationTimeline, {
      props: { flow },
    })

    expect(wrapper.find('[data-test="conversation-system-prompt-bar"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="conversation-message-row-user"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="conversation-message-row-developer"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="conversation-message-row-system"]').exists()).toBe(false)
  })

  it('renders injection part collapsed with tag label', () => {
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
            { id: 'p-1', type: 'text', text: 'Question' },
          ],
        },
      ],
    }

    const wrapper = mount(ConversationTimeline, {
      props: { flow },
    })

    expect(wrapper.find('[data-test="conversation-part-injection"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('[EXTREMELY_IMPORTANT]')
    expect(wrapper.find('[data-test="conversation-part-injection"] pre').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Rules')
  })

  it('injection part expands to show raw text in pre', async () => {
    const injectionText = '<EXTREMELY_IMPORTANT>\nRules\n</EXTREMELY_IMPORTANT>'
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
            { id: 'p-0', type: 'injection', tag: 'EXTREMELY_IMPORTANT', text: injectionText, defaultCollapsed: true },
            { id: 'p-1', type: 'text', text: 'Question' },
          ],
        },
      ],
    }

    const wrapper = mount(ConversationTimeline, {
      props: { flow },
    })

    await wrapper.get('[data-test="conversation-part-injection"] button').trigger('click')

    expect(wrapper.get('[data-test="conversation-part-injection"] pre').text()).toContain(injectionText)
  })

  it('renders the empty state when there are no messages', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([]),
      },
    })

    expect(wrapper.text()).toContain('No conversation content found')
  })

  it('user bubble uses neutral background with primary left bar', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-user-neutral-shell',
            role: 'user',
            parts: [{ id: 'part-text-user-neutral-shell', type: 'text', text: 'Hello' }],
          },
        ]),
      },
    })

    const row = wrapper.get('[data-test="conversation-message-row-user"]')
    const shell = row.get('.conversation-message-shell')

    expect(shell.attributes('class')).toContain('conversation-message-shell')
    expect(shell.attributes('class')).not.toContain('bg-primary-50')
    expect(messageRowSource).toContain('border-l-primary-500')
    expect(messageRowSource).toContain('bg-gray-100')
    expect(messageRowSource).not.toContain('bg-primary-50')
  })

  it('reasoning part shows minimalist collapsed label with segments count', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-assistant-reasoning-segments',
            role: 'assistant',
            parts: [
              {
                id: 'part-reasoning-segments',
                type: 'reasoning',
                text: 'First thought\n\nSecond thought\n\nThird thought',
                defaultCollapsed: true,
                metadata: { segments: 3 },
              },
            ],
          },
        ]),
      },
    })

    const toggle = wrapper.get('[data-test="conversation-part-reasoning"] button')

    expect(toggle.text()).toContain('Reasoning')
    expect(toggle.text()).toContain('3 segments')
  })

  it('tool card header shows lines and size when outputSize is present', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-assistant-tool-size',
            role: 'assistant',
            parts: [
              {
                id: 'part-tool-size',
                type: 'tool',
                tool: 'bash',
                state: {
                  status: 'completed',
                  input: { command: 'ls' },
                  output: 'file.txt',
                  outputSize: { bytes: 2048, lines: 15 },
                },
              },
            ],
          },
        ]),
      },
    })

    const meta = wrapper.get('.conversation-tool-meta')

    expect(meta.text()).toContain('15 lines')
    expect(meta.text()).toContain('KB')
  })

  it('tool card header shows error label when state.error is set', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: createFlow([
          {
            id: 'message-assistant-tool-error',
            role: 'assistant',
            parts: [
              {
                id: 'part-tool-error',
                type: 'tool',
                tool: 'bash',
                state: {
                  status: 'error',
                  input: { command: 'npm test' },
                  error: 'something failed',
                },
              },
            ],
          },
        ]),
      },
    })

    expect(wrapper.get('.conversation-tool-meta').text()).toBe('error')
  })
})
