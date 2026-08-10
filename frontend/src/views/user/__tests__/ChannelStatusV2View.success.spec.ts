import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ChannelStatusV2View from '../ChannelStatusV2View.vue'

const api = vi.hoisted(() => ({
  getDimensions: vi.fn(),
  getSnapshot: vi.fn(),
  getMatrix: vi.fn(),
  getModels: vi.fn(),
  getErrors: vi.fn(),
  getUsers: vi.fn(),
}))

vi.mock('@/api/channelMonitorV2', () => api)
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ replace: vi.fn() }),
}))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ isAdmin: false }) }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: vi.fn() }) }))
vi.mock('@/utils/featureFlags', () => ({ isChannelMonitorThroughputHidden: () => false }))
vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      te: () => false,
      locale: { value: 'en-US' },
    }),
  }
})

const coverage = {
  requested_start: '2026-08-01T00:00:00Z',
  requested_end: '2026-08-01T01:00:00Z',
  coverage_start: '2026-08-01T00:00:00Z',
  data_through: '2026-08-01T01:00:00Z',
  computed_at: '2026-08-01T01:00:00Z',
  aggregation_lag_seconds: 0,
  coverage_complete: true,
  bucket_seconds: 300,
}

const metrics = {
  success_requests: 80,
  error_requests: 20,
  request_count: 100,
  token_count: 0,
  rpm: 1,
  tpm: 1,
  error_rate: 0.1,
  success_rate: 0.8,
  cache_rate: 0,
  cache_rate_numerator: 0,
  cache_rate_denominator: 0,
  ttft: { sample_count: 0, p50_ms: null, p95_ms: null, avg_ms: null },
  duration: { sample_count: 0, p50_ms: null, p95_ms: null, avg_ms: null },
}

const health = { overall: 'healthy', error_rate: 'healthy', ttft: 'healthy', minimum_sample: 1 }

describe('ChannelStatusV2View success rate', () => {
  it('shows backend success_rate rather than one minus scored error_rate', async () => {
    api.getDimensions.mockResolvedValue({ platforms: [], groups: [], models: [] })
    api.getSnapshot.mockResolvedValue({
      config: { refresh_interval_seconds: 300 },
      coverage,
      metrics,
      health,
      trend: [],
    })
    api.getMatrix.mockResolvedValue({ coverage, group_by: 'platform', items: [] })
    api.getModels.mockResolvedValue({ coverage, items: [] })
    api.getErrors.mockResolvedValue({ coverage, items: [] })
    api.getUsers.mockResolvedValue({ coverage, items: [] })

    const wrapper = mount(ChannelStatusV2View, {
      global: {
        stubs: {
          AppLayout: defineComponent({ template: '<main><slot /></main>' }),
          Icon: true,
          LoadingSpinner: true,
          Select: true,
          FilterMultiSelect: true,
          MonitorRankBadge: true,
          MonitorTrendChart: true,
          RelayPulseMatrix: true,
          MetricCell: defineComponent({
            props: { label: String, value: String },
            template: '<div data-testid="metric-cell">{{ label }} {{ value }}</div>',
          }),
        },
      },
    })
    await flushPromises()

    const successCell = wrapper.findAll('[data-testid="metric-cell"]')
      .find((cell) => cell.text().includes('channelMonitorV2.metrics.successRate'))
    expect(successCell?.text()).toContain('80.0%')
    wrapper.unmount()
  })
})
