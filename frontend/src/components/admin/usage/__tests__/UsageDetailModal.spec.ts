import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'

import UsageDetailModal from '../UsageDetailModal.vue'

const messages: Record<string, string> = {
  'common.copy': 'Copy',
  'common.copied': 'Copied',
  'common.close': 'Close',
  'common.loading': 'Loading...',
  'common.retry': 'Retry',
  'admin.usage.clientRequestHeaders': 'Client Request Headers',
  'admin.usage.clientRequestBody': 'Client Request Body',
  'admin.usage.upstreamRequestHeaders': 'Upstream Request Headers',
  'admin.usage.upstreamRequestBody': 'Upstream Request Body',
  'admin.usage.responseHeaders': 'Response Headers',
  'admin.usage.responseBody': 'Response Body',
  'admin.usage.requestId': 'Request ID',
  'admin.usage.user': 'User',
  'usage.model': 'Model',
  'usage.time': 'Time',
  'admin.usage.emptyDetailContent': 'No content',
  'admin.usage.detailLoadFailed': 'Failed to load detail',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

describe('UsageDetailModal', () => {
  beforeEach(() => {
    vi.stubGlobal('navigator', {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  it('renders six top-level tabs and supports upstream detail tabs', async () => {
    const wrapper = mount(UsageDetailModal, {
      props: {
        show: true,
        usageLog: {
          request_id: 'req-123',
          user: { email: 'alice@example.com' },
          model: 'gpt-4.1',
          created_at: '2026-03-20T10:00:00Z',
        },
        detail: {
          usage_log_id: 1,
          request_headers: '{"authorization":"Bearer token"}',
          request_body: '{"foo":1}',
          upstream_request_headers: '{"x-upstream":"gateway"}',
          upstream_request_body: '{"bar":2}',
          response_headers: null,
          response_body: 'not-json',
          created_at: '2026-03-20T10:00:00Z',
        },
        loading: false,
        error: '',
      },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show', 'title'],
            template: '<div v-if="show"><slot /></div>',
          },
        },
      },
    })

    expect(wrapper.text()).toContain('req-123')
    expect(wrapper.text()).toContain('alice@example.com')
    expect(wrapper.text()).toContain('gpt-4.1')
    expect(wrapper.text()).toContain('2026-03-20T10:00:00Z')
    expect(wrapper.findAll('button[data-test^="tab-"]')).toHaveLength(6)
    const detailPanel = wrapper.find('[data-test="detail-content-panel"]')
    expect(detailPanel.exists()).toBe(true)
    expect(detailPanel.classes()).toContain('h-[60vh]')
    expect(wrapper.text()).toContain('Client Request Headers')
    expect(wrapper.text()).toContain('Client Request Body')
    expect(wrapper.text()).toContain('Upstream Request Headers')
    expect(wrapper.text()).toContain('Upstream Request Body')
    expect(wrapper.text()).toContain('Response Headers')
    expect(wrapper.text()).toContain('Response Body')
    expect(wrapper.find('pre').classes()).toContain('whitespace-pre-wrap')
    expect(wrapper.find('pre').classes()).toContain('break-words')
    expect(wrapper.text()).toContain(`{
  "authorization": "Bearer token"
}`)

    await wrapper.find('[data-test="tab-client-request-body"]').trigger('click')
    expect(wrapper.text()).toContain(`{
  "foo": 1
}`)

    await wrapper.find('[data-test="tab-upstream-request-headers"]').trigger('click')
    expect(wrapper.text()).toContain(`{
  "x-upstream": "gateway"
}`)

    await wrapper.find('[data-test="tab-upstream-request-body"]').trigger('click')
    expect(wrapper.text()).toContain(`{
  "bar": 2
}`)

    await wrapper.find('[data-test="copy-current-tab"]').trigger('click')
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(`{
  "bar": 2
}`)
    expect(wrapper.text()).toContain('Copied')

    await wrapper.find('[data-test="tab-response-headers"]').trigger('click')
    expect(wrapper.text()).toContain('No content')

    await wrapper.find('[data-test="tab-response-body"]').trigger('click')
    expect(wrapper.text()).toContain('not-json')
  })

  it('shows empty state when legacy upstream fields are missing', async () => {
    const wrapper = mount(UsageDetailModal, {
      props: {
        show: true,
        usageLog: {
          request_id: 'req-legacy',
          user: { email: 'legacy@example.com' },
          model: 'gpt-4.1',
          created_at: '2026-03-20T10:00:00Z',
        },
        detail: {
          usage_log_id: 2,
          request_headers: 'client-headers',
          request_body: 'client-body',
          upstream_request_headers: null,
          upstream_request_body: null,
          response_headers: 'response-headers',
          response_body: 'response-body',
          created_at: '2026-03-20T10:00:00Z',
        },
        loading: false,
        error: '',
      },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show', 'title'],
            template: '<div v-if="show"><slot /></div>',
          },
        },
      },
    })

    await wrapper.find('[data-test="tab-upstream-request-headers"]').trigger('click')
    expect(wrapper.text()).toContain('No content')

    await wrapper.find('[data-test="tab-upstream-request-body"]').trigger('click')
    expect(wrapper.text()).toContain('No content')
  })

  it('shows retry button when error is present', async () => {
    const wrapper = mount(UsageDetailModal, {
      props: {
        show: true,
        usageLog: null,
        detail: null,
        loading: false,
        error: 'Failed to load detail',
      },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show', 'title'],
            template: '<div v-if="show"><slot /></div>',
          },
        },
      },
    })

    expect(wrapper.text()).toContain('Failed to load detail')
    await wrapper.find('[data-test="usage-detail-retry"]').trigger('click')
    expect(wrapper.emitted('retry')).toHaveLength(1)
  })
})
