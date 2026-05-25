import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ConversationTimeline from '../ConversationTimeline.vue'
import type { ConversationFlow } from '@/utils/conversation/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
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
    'conversation.timelineLabel': 'Conversation timeline',
    'conversation.imageAlt': 'Conversation image',
    'conversation.role.user': 'User',
    'conversation.role.assistant': 'Assistant',
    'conversation.role.system': 'System',
    'conversation.role.developer': 'Developer',
  }

  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => messages[key] ?? key }),
  }
})

const flow: ConversationFlow = {
  source: 'client',
  format: 'openai-chat',
  warnings: [],
  nodes: [
    {
      id: 'system-1',
      type: 'system',
      role: 'system',
      title: 'System',
      summary: 'System instructions',
      defaultCollapsed: true,
      parts: [{ type: 'text', text: 'Do not reveal secrets' }],
    },
    {
      id: 'user-1',
      type: 'user',
      role: 'user',
      title: 'User',
      summary: 'Hello',
      defaultCollapsed: false,
      parts: [{ type: 'text', text: 'Hello **world**' }],
    },
    {
      id: 'tool-1',
      type: 'tool_call',
      title: 'Tool Call · web_search',
      summary: 'query timeout',
      defaultCollapsed: true,
      toolName: 'web_search',
      input: { query: 'timeout' },
    },
    {
      id: 'assistant-1',
      type: 'assistant',
      role: 'assistant',
      title: 'Assistant',
      summary: 'Done',
      defaultCollapsed: false,
      parts: [
        { type: 'text', text: 'Done\n\n```ts\nconst ok = true\n```' },
        { type: 'image', src: 'https://example.com/output.png', alt: 'output' },
      ],
    },
  ],
}

describe('ConversationTimeline', () => {
  it('renders conversation cards with the expected default collapsed states', () => {
    const wrapper = mount(ConversationTimeline, {
      props: { flow },
    })

    expect(wrapper.findAll('[data-test="conversation-node-card"]')).toHaveLength(4)
    expect(wrapper.get('ol').attributes('aria-label')).toBe('Conversation timeline')
    expect(wrapper.html()).toContain('<strong>world</strong>')
    expect(wrapper.text()).toContain('System instructions')
    expect(wrapper.text()).not.toContain('Do not reveal secrets')
    expect(wrapper.text()).toContain('query timeout')
    expect(wrapper.text()).not.toContain('"query": "timeout"')
    expect(wrapper.text()).toContain('const ok = true')

    const image = wrapper.get('[data-test="conversation-image"]')
    expect(image.attributes('src')).toBe('https://example.com/output.png')
  })

  it('expands only the clicked collapsed node', async () => {
    const wrapper = mount(ConversationTimeline, {
      props: { flow },
    })

    const toggles = wrapper.findAll('[data-test="conversation-node-toggle"]')
    await toggles[0].trigger('click')

    expect(wrapper.text()).toContain('Do not reveal secrets')
    expect(wrapper.text()).not.toContain('"query": "timeout"')
  })

  it('resets collapsed state when the node collapsed prop changes', async () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: {
          ...flow,
          nodes: [
            {
              id: 'system-reset',
              type: 'system',
              role: 'system',
              title: 'System',
              summary: 'Expanded first',
              defaultCollapsed: false,
              parts: [{ type: 'text', text: 'Initially visible' }],
            },
          ],
        },
      },
    })

    expect(wrapper.text()).toContain('Initially visible')

    await wrapper.setProps({
      flow: {
        ...flow,
        nodes: [
          {
            id: 'system-reset',
            type: 'system',
            role: 'system',
            title: 'System',
            summary: 'Collapsed next',
            defaultCollapsed: true,
            parts: [{ type: 'text', text: 'Should be hidden after prop update' }],
          },
        ],
      },
    })

    expect(wrapper.text()).toContain('Collapsed next')
    expect(wrapper.text()).not.toContain('Should be hidden after prop update')
  })

  it('filters unsafe media urls and keeps safe data images accessible', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: {
          ...flow,
          nodes: [
            {
              id: 'assistant-media-safety',
              type: 'assistant',
              role: 'assistant',
              title: 'Assistant',
              summary: 'Media safety',
              defaultCollapsed: false,
              parts: [
                { type: 'image', src: 'javascript:alert(1)' },
                { type: 'image', src: 'data:image/png;base64,AAAA' },
              ],
            },
          ],
        },
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
        flow: {
          ...flow,
          nodes: [
            {
              id: 'user-dangerous',
              type: 'user',
              role: 'user',
              title: 'User',
              summary: 'Dangerous html',
              defaultCollapsed: false,
              parts: [{ type: 'text', text: '<img src="x" onerror="alert(1)"> **safe**' }],
            },
          ],
        },
      },
    })

    expect(wrapper.html()).not.toContain('onerror')
    expect(wrapper.html()).toContain('<strong>safe</strong>')
  })

  it('does not render markdown images while still rendering explicit media parts', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: {
          ...flow,
          nodes: [
            {
              id: 'assistant-markdown-image',
              type: 'assistant',
              role: 'assistant',
              title: 'Assistant',
              summary: 'Markdown media safety',
              defaultCollapsed: false,
              parts: [
                { type: 'text', text: '![x](https://example.com/a.png) <img src="https://example.com/a.png">' },
                { type: 'image', src: 'https://example.com/output.png', alt: 'output' },
              ],
            },
          ],
        },
      },
    })

    const images = wrapper.findAll('img')
    expect(images).toHaveLength(1)
    expect(images[0].attributes('data-test')).toBe('conversation-image')
    expect(images[0].attributes('src')).toBe('https://example.com/output.png')
    expect(wrapper.html()).not.toContain('https://example.com/a.png')
  })

  it('uses localized card titles and labels instead of node titles', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: {
          ...flow,
          nodes: [
            {
              id: 'user-localized',
              type: 'user',
              role: 'user',
              title: 'Hard-coded title should not render',
              summary: 'Hello',
              defaultCollapsed: false,
              parts: [{ type: 'text', text: 'Hello' }],
            },
            {
              id: 'tool-localized',
              type: 'tool_call',
              title: 'Hard-coded tool title should not render',
              summary: 'Tool summary',
              defaultCollapsed: true,
              toolName: 'web_search',
              input: { query: 'timeout' },
            },
          ],
        },
      },
    })

    expect(wrapper.text()).toContain('User')
    expect(wrapper.text()).toContain('Tool Call · web_search')
    expect(wrapper.text()).not.toContain('Hard-coded title should not render')
    expect(wrapper.text()).not.toContain('Hard-coded tool title should not render')
  })

  it('renders the empty state when there are no nodes', () => {
    const wrapper = mount(ConversationTimeline, {
      props: {
        flow: {
          source: 'client',
          format: 'unknown',
          warnings: [],
          nodes: [],
        },
      },
    })

    expect(wrapper.text()).toContain('No conversation content found')
  })
})
