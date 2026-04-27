import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, onMounted, onUnmounted } from 'vue'

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
  'admin.usage.upstreamResponseHeaders': 'Upstream Response Headers',
  'admin.usage.upstreamResponseBody': 'Upstream Response Body',
  'admin.usage.responseHeaders': 'Response Headers',
  'admin.usage.responseBody': 'Response Body',
  'admin.usage.imagePreview': 'Image Preview',
  'admin.usage.rawResponseBody': 'Raw Response JSON',
  'admin.usage.openImagePreview': 'Open image preview',
  'admin.usage.previewImageTitle': 'Image Preview Modal',
  'admin.usage.closeImagePreview': 'Close preview',
  'images.results.openPreview': 'Open preview',
  'images.results.download': 'Download',
  'images.results.closePreview': 'Close preview',
  'images.results.previewTitle': 'Preview image',
  'images.results.revisedPrompt': 'Revised prompt',
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

const EscapeClosingBaseDialogStub = defineComponent({
  props: ['show', 'title'],
  emits: ['close'],
  setup(_props, { emit }) {
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        emit('close')
      }
    }

    onMounted(() => {
      document.addEventListener('keydown', handleEscape)
    })

    onUnmounted(() => {
      document.removeEventListener('keydown', handleEscape)
    })

    return {}
  },
  template: '<div v-if="show"><slot /></div>',
})

describe('UsageDetailModal', () => {
  beforeEach(() => {
    vi.stubGlobal('navigator', {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    })
  })

  it('renders eight top-level tabs and supports upstream detail tabs', async () => {
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
          request_headers: ':method: POST\n:url: https://api.example.com/v1/chat/completions\nauthorization: Bearer token\nx-trace-id: trace-client',
          request_body: '{"foo":1}',
          upstream_request_headers: ':method: POST\n:url: https://upstream.example.com/v1/chat/completions\nx-upstream: gateway\nx-upstream-trace-id: trace-upstream',
          upstream_request_body: '{"bar":2}',
          upstream_response_headers: ':status: 200\nContent-Type: application/json',
          upstream_response_body: '{"upstream_result":"ok"}',
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
    expect(wrapper.findAll('button[data-test^="tab-"]')).toHaveLength(8)
    const detailPanel = wrapper.find('[data-test="detail-content-panel"]')
    expect(detailPanel.exists()).toBe(true)
    expect(detailPanel.classes()).toContain('h-[60vh]')
    expect(wrapper.text()).toContain('Client Request Headers')
    expect(wrapper.text()).toContain('Client Request Body')
    expect(wrapper.text()).toContain('Upstream Request Headers')
    expect(wrapper.text()).toContain('Upstream Request Body')
    expect(wrapper.text()).toContain('Upstream Response Headers')
    expect(wrapper.text()).toContain('Upstream Response Body')
    expect(wrapper.text()).toContain('Response Headers')
    expect(wrapper.text()).toContain('Response Body')
    expect(wrapper.find('pre').classes()).toContain('whitespace-pre-wrap')
    expect(wrapper.find('pre').classes()).toContain('break-words')
    expect(wrapper.text()).toContain(`:method: POST
:url: https://api.example.com/v1/chat/completions
authorization: Bearer token
x-trace-id: trace-client`)

    await wrapper.find('[data-test="copy-current-tab"]').trigger('click')
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(`:method: POST
:url: https://api.example.com/v1/chat/completions
authorization: Bearer token
x-trace-id: trace-client`)
    expect(wrapper.text()).toContain('Copied')

    await wrapper.find('[data-test="tab-client-request-body"]').trigger('click')
    expect(wrapper.text()).toContain(`{
  "foo": 1
}`)

    await wrapper.find('[data-test="tab-upstream-request-headers"]').trigger('click')
    expect(wrapper.text()).toContain(`:method: POST
:url: https://upstream.example.com/v1/chat/completions
x-upstream: gateway
x-upstream-trace-id: trace-upstream`)

    await wrapper.find('[data-test="copy-current-tab"]').trigger('click')
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(`:method: POST
:url: https://upstream.example.com/v1/chat/completions
x-upstream: gateway
x-upstream-trace-id: trace-upstream`)
    expect(wrapper.text()).toContain('Copied')

    await wrapper.find('[data-test="tab-upstream-request-body"]').trigger('click')
    expect(wrapper.text()).toContain(`{
  "bar": 2
}`)

    await wrapper.find('[data-test="copy-current-tab"]').trigger('click')
    expect(navigator.clipboard.writeText).toHaveBeenLastCalledWith(`{
  "bar": 2
}`)

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
          upstream_response_headers: null,
          upstream_response_body: null,
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

  it('renders gpt-image previews through the shared gallery while keeping raw json visible', async () => {
    const wrapper = mount(UsageDetailModal, {
      props: {
        show: true,
        usageLog: {
          request_id: 'req-image-123',
          user: { email: 'image@example.com' },
          model: 'gpt-image-2',
          created_at: '2026-03-20T10:00:00Z',
        },
        detail: {
          usage_log_id: 3,
          request_headers: null,
          request_body: '{"model":"gpt-image-2","output_format":"webp"}',
          upstream_request_headers: null,
          upstream_request_body: null,
          upstream_response_headers: null,
          upstream_response_body: null,
          response_headers: ':status: 200\nContent-Type: application/json',
          response_body: '{"created":1776989094,"data":[{"b64_json":"QUJD","revised_prompt":"draw a neon fox"}]}',
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

    await wrapper.find('[data-test="tab-response-body"]').trigger('click')

    expect(wrapper.get('[data-testid="image-preview-gallery"]').classes()).toContain('grid-cols-1')
    expect(wrapper.get('[data-testid="usage-detail-image-preview-0"]').attributes('src')).toBe('data:image/webp;base64,QUJD')
    expect(wrapper.text()).toContain('Image Preview')
    expect(wrapper.text()).toContain('Revised prompt')
    expect(wrapper.text()).toContain('draw a neon fox')
    expect(wrapper.text()).toContain('Raw Response JSON')
    expect(wrapper.text()).toContain('"b64_json": "QUJD"')
  })

  it('opens the shared fullscreen gallery preview and exposes download', async () => {
    const wrapper = mount(UsageDetailModal, {
      props: {
        show: true,
        usageLog: {
          request_id: 'req-image-zoom',
          user: { email: 'image@example.com' },
          model: 'gpt-image-2',
          created_at: '2026-03-20T10:00:00Z',
        },
        detail: {
          usage_log_id: 4,
          request_headers: null,
          request_body: '{"model":"gpt-image-2"}',
          upstream_request_headers: null,
          upstream_request_body: null,
          upstream_response_headers: null,
          upstream_response_body: null,
          response_headers: ':status: 200\nContent-Type: application/json',
          response_body: '{"created":1776989094,"data":[{"b64_json":"QUJD"}]}',
          created_at: '2026-03-20T10:00:00Z',
        },
        loading: false,
        error: '',
      },
      attachTo: document.body,
      global: {
        stubs: {
          BaseDialog: {
            props: ['show', 'title'],
            template: '<div v-if="show"><slot /></div>',
          },
        },
      },
    })

    await wrapper.find('[data-test="tab-response-body"]').trigger('click')
    await wrapper.get('[data-testid="image-preview-open-0"]').trigger('click')

    expect(wrapper.get('[data-testid="image-preview-modal"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="image-preview-modal-image"]').attributes('src')).toBe('data:image/png;base64,QUJD')
    expect(wrapper.get('[data-testid="image-preview-modal-download"]').attributes('href')).toBe('data:image/png;base64,QUJD')

    await wrapper.get('[data-testid="image-preview-close"]').trigger('click')
    expect(wrapper.find('[data-testid="image-preview-modal"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('passes upstream image urls through the shared gallery and keeps raw json visible', async () => {
    const wrapper = mount(UsageDetailModal, {
      props: {
        show: true,
        usageLog: {
          request_id: 'req-upstream-image',
          user: { email: 'image@example.com' },
          model: 'gpt-image-2',
          created_at: '2026-03-20T10:00:00Z',
        },
        detail: {
          usage_log_id: 5,
          request_headers: null,
          request_body: '{"model":"gpt-image-2"}',
          upstream_request_headers: null,
          upstream_request_body: null,
          upstream_response_headers: ':status: 200\nContent-Type: application/json',
          upstream_response_body: '{"data":[{"url":"https://cdn.example.com/output.png","revised_prompt":"draw a teal fox"}]}',
          response_headers: null,
          response_body: null,
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

    await wrapper.find('[data-test="tab-upstream-response-body"]').trigger('click')

    expect(wrapper.get('[data-testid="usage-detail-image-preview-0"]').attributes('src')).toBe('https://cdn.example.com/output.png')
    expect(wrapper.text()).toContain('draw a teal fox')
    expect(wrapper.text()).toContain('"url": "https://cdn.example.com/output.png"')
  })

  it('pressing Escape closes only the shared image preview, not the parent usage dialog', async () => {
    const wrapper = mount(UsageDetailModal, {
      props: {
        show: true,
        usageLog: {
          request_id: 'req-image-escape',
          user: { email: 'image@example.com' },
          model: 'gpt-image-2',
          created_at: '2026-03-20T10:00:00Z',
        },
        detail: {
          usage_log_id: 6,
          request_headers: null,
          request_body: '{"model":"gpt-image-2"}',
          upstream_request_headers: null,
          upstream_request_body: null,
          upstream_response_headers: null,
          upstream_response_body: null,
          response_headers: ':status: 200\nContent-Type: application/json',
          response_body: '{"created":1776989094,"data":[{"b64_json":"QUJD"}]}',
          created_at: '2026-03-20T10:00:00Z',
        },
        loading: false,
        error: '',
      },
      attachTo: document.body,
      global: {
        stubs: {
          BaseDialog: EscapeClosingBaseDialogStub,
        },
      },
    })

    await wrapper.find('[data-test="tab-response-body"]').trigger('click')
    await wrapper.get('[data-testid="image-preview-open-0"]').trigger('click')
    expect(wrapper.get('[data-testid="image-preview-modal"]').exists()).toBe(true)

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="image-preview-modal"]').exists()).toBe(false)
    expect(wrapper.emitted('close')).toBeUndefined()

    wrapper.unmount()
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
