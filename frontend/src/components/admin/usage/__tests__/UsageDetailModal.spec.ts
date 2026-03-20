import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'

import UsageDetailModal from '../UsageDetailModal.vue'

const messages: Record<string, string> = {
  'common.copy': 'Copy',
  'common.copied': 'Copied',
  'common.close': 'Close',
  'common.loading': 'Loading...',
  'common.retry': 'Retry',
  'admin.usage.requestHeaders': 'Request Headers',
  'admin.usage.requestBody': 'Request Body',
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

  it('renders header info, formats JSON, switches tabs, shows empty state and copied state', async () => {
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
    expect(wrapper.text()).toContain(`{
  "authorization": "Bearer token"
}`)

    await wrapper.find('[data-test="tab-request-body"]').trigger('click')
    expect(wrapper.text()).toContain(`{
  "foo": 1
}`)

    await wrapper.find('[data-test="copy-current-tab"]').trigger('click')
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(`{
  "foo": 1
}`)
    expect(wrapper.text()).toContain('Copied')

    await wrapper.find('[data-test="tab-response-headers"]').trigger('click')
    expect(wrapper.text()).toContain('No content')

    await wrapper.find('[data-test="tab-response-body"]').trigger('click')
    expect(wrapper.text()).toContain('not-json')
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
