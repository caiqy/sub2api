import { describe, expect, expectTypeOf, it, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import { apiClient } from '@/api/client'
import type { AdminUsageDetail, AdminUsageLog } from '@/types'
import UsageView from '../UsageView.vue'

const createDeferred = <T>() => {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

const { list, getStats, getDetail, getSnapshotV2, getModelStats, getById, showError } = vi.hoisted(() => {
  vi.stubGlobal('localStorage', {
    getItem: vi.fn(() => null),
    setItem: vi.fn(),
    removeItem: vi.fn(),
  })

  return {
    list: vi.fn(),
    getStats: vi.fn(),
    getDetail: vi.fn(),
    getSnapshotV2: vi.fn(),
    getModelStats: vi.fn(),
    getById: vi.fn(),
    showError: vi.fn(),
  }
})

const messages: Record<string, string> = {
  'admin.dashboard.timeRange': 'Time Range',
  'admin.dashboard.day': 'Day',
  'admin.dashboard.hour': 'Hour',
  'admin.usage.failedToLoadUser': 'Failed to load user',
  'admin.usage.detailNotFound': 'Detail not found',
}

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: {
      list,
      getStats,
      getDetail,
    },
    dashboard: {
      getSnapshotV2,
      getModelStats,
    },
    users: {
      getById,
    },
  },
}))

vi.mock('@/api/admin/usage', () => ({
  adminUsageAPI: {
    list: vi.fn(),
    getDetail: vi.fn(),
  },
  default: {
    list: vi.fn(),
    getDetail: vi.fn(),
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showWarning: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn(),
  }),
}))

vi.mock('@/utils/format', () => ({
  formatReasoningEffort: (value: string | null | undefined) => value ?? '-',
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: {}
  })
}))

const AppLayoutStub = { template: '<div><slot /></div>' }
const UsageFiltersStub = { template: '<div><slot name="after-reset" /></div>' }
const UsageFiltersBillingModeStub = {
  props: ['modelValue'],
  emits: ['update:modelValue', 'change'],
  template: `
    <div>
      <slot name="after-reset" />
      <button
        data-test="apply-billing-mode-filter"
        @click="$emit('update:modelValue', { ...modelValue, billing_mode: 'image' }); $emit('change')"
      >billing mode</button>
    </div>
  `,
}
const ModelDistributionChartStub = {
  props: ['metric'],
  emits: ['update:metric'],
  template: `
    <div data-test="model-chart">
      <span class="metric">{{ metric }}</span>
      <button class="switch-metric" @click="$emit('update:metric', 'actual_cost')">switch</button>
    </div>
  `,
}
const GroupDistributionChartStub = {
  props: ['metric'],
  emits: ['update:metric'],
  template: `
    <div data-test="group-chart">
      <span class="metric">{{ metric }}</span>
      <button class="switch-metric" @click="$emit('update:metric', 'actual_cost')">switch</button>
    </div>
  `,
}

const UsageTableStub = {
  emits: ['detail', 'userClick'],
  template: '<button data-test="open-detail" @click="$emit(\'detail\', { id: 42, request_id: \'req-42\', user: { email: \'alice@example.com\' }, model: \'gpt-4.1\', created_at: \'2026-03-20T10:00:00Z\', has_detail: true })">detail</button>',
}

const UsageTableMultipleRowsStub = {
  emits: ['detail', 'userClick'],
  template: `
    <div>
      <button
        data-test="open-detail-1"
        @click="$emit('detail', { id: 1, request_id: 'req-1', user: { email: 'alice@example.com' }, model: 'gpt-4.1', created_at: '2026-03-20T10:00:00Z', has_detail: true })"
      >detail 1</button>
      <button
        data-test="open-detail-2"
        @click="$emit('detail', { id: 2, request_id: 'req-2', user: { email: 'bob@example.com' }, model: 'gpt-4.1-mini', created_at: '2026-03-20T10:01:00Z', has_detail: true })"
      >detail 2</button>
    </div>
  `,
}

const UsageTableSortAndDetailStub = {
  props: ['serverSideSort', 'defaultSortKey', 'defaultSortOrder'],
  emits: ['detail', 'sort', 'userClick'],
  template: `
    <div>
      <span data-test="server-side-sort">{{ serverSideSort }}</span>
      <span data-test="default-sort-key">{{ defaultSortKey }}</span>
      <span data-test="default-sort-order">{{ defaultSortOrder }}</span>
      <button
        data-test="emit-sort"
        @click="$emit('sort', 'model', 'asc')"
      >sort</button>
      <button
        data-test="emit-detail"
        @click="$emit('detail', { id: 7, request_id: 'req-7', user: { email: 'sort@example.com' }, model: 'gpt-4.1', created_at: '2026-03-20T10:00:00Z', has_detail: true })"
      >detail</button>
    </div>
  `,
}

const UsageDetailModalStub = {
  props: ['show', 'usageLog', 'detail', 'loading', 'error'],
  emits: ['close', 'retry'],
  template: `
    <div v-if="show" data-test="usage-detail-modal">
      <span class="request-id">{{ usageLog?.request_id }}</span>
      <span class="detail-id">{{ detail?.usage_log_id }}</span>
      <span class="error">{{ error }}</span>
      <button data-test="retry-detail" @click="$emit('retry')">retry</button>
    </div>
  `,
}

describe('admin UsageView distribution metric toggles', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    list.mockReset()
    getStats.mockReset()
    getDetail.mockReset()
    getSnapshotV2.mockReset()
    getModelStats.mockReset()
    getById.mockReset()

    list.mockResolvedValue({
      items: [],
      total: 0,
      pages: 0,
    })
    getStats.mockResolvedValue({
      total_requests: 0,
      total_input_tokens: 0,
      total_output_tokens: 0,
      total_cache_tokens: 0,
      total_tokens: 0,
      total_cost: 0,
      total_actual_cost: 0,
      average_duration_ms: 0,
    })
    getSnapshotV2.mockResolvedValue({
      trend: [],
      models: [],
      groups: [],
    })
    getModelStats.mockResolvedValue([])
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('keeps model and group metric toggles independent without refetching chart data', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          UsageStatsCards: true,
          UsageFilters: UsageFiltersStub,
          UsageTable: true,
          UsageExportProgress: true,
          UsageCleanupDialog: true,
          UserBalanceHistoryModal: true,
          Pagination: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenUsageTrend: true,
          ModelDistributionChart: ModelDistributionChartStub,
          GroupDistributionChart: GroupDistributionChartStub,
        },
      },
    })

    vi.advanceTimersByTime(120)
    await flushPromises()

    expect(getSnapshotV2).toHaveBeenCalledTimes(1)
    const now = new Date()
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000)
    expect(getSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({
      start_date: formatLocalDate(yesterday),
      end_date: formatLocalDate(now),
      granularity: 'hour'
    }))

    const modelChart = wrapper.find('[data-test="model-chart"]')
    const groupChart = wrapper.find('[data-test="group-chart"]')

    expect(modelChart.find('.metric').text()).toBe('tokens')
    expect(groupChart.find('.metric').text()).toBe('tokens')

    await modelChart.find('.switch-metric').trigger('click')
    await flushPromises()

    expect(modelChart.find('.metric').text()).toBe('actual_cost')
    expect(groupChart.find('.metric').text()).toBe('tokens')
    expect(getSnapshotV2).toHaveBeenCalledTimes(1)

    await groupChart.find('.switch-metric').trigger('click')
    await flushPromises()

    expect(modelChart.find('.metric').text()).toBe('actual_cost')
    expect(groupChart.find('.metric').text()).toBe('actual_cost')
    expect(getSnapshotV2).toHaveBeenCalledTimes(1)
  })

  it('does not refetch unsupported chart or model endpoints when billing_mode filter is active', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          UsageStatsCards: true,
          UsageFilters: UsageFiltersBillingModeStub,
          UsageTable: true,
          UsageExportProgress: true,
          UsageCleanupDialog: true,
          UserBalanceHistoryModal: true,
          Pagination: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenUsageTrend: true,
          ModelDistributionChart: ModelDistributionChartStub,
          GroupDistributionChart: GroupDistributionChartStub,
          EndpointDistributionChart: true,
        },
      },
    })

    vi.advanceTimersByTime(120)
    await flushPromises()

    expect(getModelStats).toHaveBeenCalledTimes(1)
    expect(getSnapshotV2).toHaveBeenCalledTimes(1)

    getModelStats.mockClear()
    getSnapshotV2.mockClear()

    await wrapper.find('[data-test="apply-billing-mode-filter"]').trigger('click')
    await flushPromises()

    expect(getModelStats).not.toHaveBeenCalled()
    expect(getSnapshotV2).not.toHaveBeenCalled()
  })
})

describe('admin usage detail API contract', () => {
  it('calls the usage detail endpoint from admin usage APIs', async () => {
    const getSpy = vi.spyOn(apiClient, 'get').mockResolvedValue({
      data: {
        usage_log_id: 42,
        request_headers: null,
        request_body: null,
        response_headers: null,
        response_body: null,
        created_at: '2026-03-20T00:00:00Z',
      },
    })

    const { adminAPI } = await import('@/api/admin')
    const { adminUsageAPI } = await vi.importActual<typeof import('@/api/admin/usage')>('@/api/admin/usage')

    expect(typeof adminAPI.usage.getDetail).toBe('function')

    await adminUsageAPI.getDetail(42)

    expect(getSpy).toHaveBeenCalledWith('/admin/usage/42/detail')

    getSpy.mockRestore()
  })

  it('includes detail-related fields in admin usage types', () => {
    expectTypeOf<AdminUsageLog>().toMatchTypeOf<{ has_detail: boolean }>()
    expectTypeOf<AdminUsageDetail>().toMatchTypeOf<{
      usage_log_id: number
      request_headers: string | null
      request_body: string | null
      upstream_request_headers: string | null
      upstream_request_body: string | null
      response_headers: string | null
      response_body: string | null
      created_at: string
    }>()
  })
})

describe('admin UsageView detail modal', () => {
  beforeEach(() => {
    showError.mockReset()
    getDetail.mockReset()
    list.mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockResolvedValue({
      total_requests: 0,
      total_input_tokens: 0,
      total_output_tokens: 0,
      total_cache_tokens: 0,
      total_tokens: 0,
      total_cost: 0,
      total_actual_cost: 0,
      average_duration_ms: 0,
    })
    getSnapshotV2.mockResolvedValue({ trend: [], models: [], groups: [] })
    getModelStats.mockResolvedValue([])
  })

  it('requests detail and opens modal when detail action is clicked', async () => {
    getDetail.mockResolvedValue({
      usage_log_id: 42,
      request_headers: '{"foo":"bar"}',
      request_body: '{}',
      upstream_request_headers: '{"x-upstream":"gateway"}',
      upstream_request_body: '{}',
      response_headers: null,
      response_body: null,
      created_at: '2026-03-20T10:00:00Z',
    })

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          UsageStatsCards: true,
          UsageFilters: UsageFiltersStub,
          UsageTable: UsageTableStub,
          UsageDetailModal: UsageDetailModalStub,
          UsageExportProgress: true,
          UsageCleanupDialog: true,
          UserBalanceHistoryModal: true,
          Pagination: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenUsageTrend: true,
          ModelDistributionChart: ModelDistributionChartStub,
          GroupDistributionChart: GroupDistributionChartStub,
          EndpointDistributionChart: true,
        },
      },
    })

    await wrapper.find('[data-test="open-detail"]').trigger('click')
    await flushPromises()

    expect(getDetail).toHaveBeenCalledWith(42)
    expect(wrapper.find('[data-test="usage-detail-modal"]').exists()).toBe(true)
    expect(wrapper.find('.request-id').text()).toBe('req-42')
    expect(wrapper.find('.detail-id').text()).toBe('42')
  })

  it('keeps server-side sort while preserving detail entrypoint', async () => {
    getDetail.mockResolvedValue({
      usage_log_id: 7,
      request_headers: null,
      request_body: null,
      upstream_request_headers: null,
      upstream_request_body: null,
      response_headers: null,
      response_body: null,
      created_at: '2026-03-20T10:00:00Z',
    })

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          UsageStatsCards: true,
          UsageFilters: UsageFiltersStub,
          UsageTable: UsageTableSortAndDetailStub,
          UsageDetailModal: UsageDetailModalStub,
          UsageExportProgress: true,
          UsageCleanupDialog: true,
          UserBalanceHistoryModal: true,
          Pagination: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenUsageTrend: true,
          ModelDistributionChart: ModelDistributionChartStub,
          GroupDistributionChart: GroupDistributionChartStub,
          EndpointDistributionChart: true,
        },
      },
    })

    expect(wrapper.find('[data-test="server-side-sort"]').text()).toBe('true')
    expect(wrapper.find('[data-test="default-sort-key"]').text()).toBe('created_at')
    expect(wrapper.find('[data-test="default-sort-order"]').text()).toBe('desc')

    await wrapper.find('[data-test="emit-sort"]').trigger('click')
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ sort_by: 'model', sort_order: 'asc' }), expect.anything())

    await wrapper.find('[data-test="emit-detail"]').trigger('click')
    await flushPromises()
    expect(getDetail).toHaveBeenCalledWith(7)
    expect(wrapper.find('.detail-id').text()).toBe('7')
  })

  it('shows not found error and can retry detail loading', async () => {
    getDetail.mockRejectedValueOnce({ response: { status: 404 } })
    getDetail.mockResolvedValueOnce({
      usage_log_id: 42,
      request_headers: null,
      request_body: null,
      upstream_request_headers: null,
      upstream_request_body: null,
      response_headers: null,
      response_body: null,
      created_at: '2026-03-20T10:00:00Z',
    })

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          UsageStatsCards: true,
          UsageFilters: UsageFiltersStub,
          UsageTable: UsageTableStub,
          UsageDetailModal: UsageDetailModalStub,
          UsageExportProgress: true,
          UsageCleanupDialog: true,
          UserBalanceHistoryModal: true,
          Pagination: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenUsageTrend: true,
          ModelDistributionChart: ModelDistributionChartStub,
          GroupDistributionChart: GroupDistributionChartStub,
          EndpointDistributionChart: true,
        },
      },
    })

    await wrapper.find('[data-test="open-detail"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="usage-detail-modal"]').exists()).toBe(true)
    expect(wrapper.find('.error').text()).toBe('Detail not found')
    expect(showError).toHaveBeenCalledWith('Detail not found')

    await wrapper.find('[data-test="retry-detail"]').trigger('click')
    await flushPromises()

    expect(getDetail).toHaveBeenCalledTimes(2)
    expect(wrapper.find('.detail-id').text()).toBe('42')
  })

  it('keeps the latest detail when earlier request resolves later', async () => {
    const firstRequest = createDeferred<AdminUsageDetail>()
    const secondRequest = createDeferred<AdminUsageDetail>()

    getDetail.mockImplementation((id: number) => {
      if (id === 1) return firstRequest.promise
      if (id === 2) return secondRequest.promise
      return Promise.reject(new Error(`unexpected id ${id}`))
    })

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          UsageStatsCards: true,
          UsageFilters: UsageFiltersStub,
          UsageTable: UsageTableMultipleRowsStub,
          UsageDetailModal: UsageDetailModalStub,
          UsageExportProgress: true,
          UsageCleanupDialog: true,
          UserBalanceHistoryModal: true,
          Pagination: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenUsageTrend: true,
          ModelDistributionChart: ModelDistributionChartStub,
          GroupDistributionChart: GroupDistributionChartStub,
          EndpointDistributionChart: true,
        },
      },
    })

    await wrapper.find('[data-test="open-detail-1"]').trigger('click')
    await wrapper.find('[data-test="open-detail-2"]').trigger('click')
    await flushPromises()

    secondRequest.resolve({
      usage_log_id: 2,
      request_headers: null,
      request_body: null,
      upstream_request_headers: null,
      upstream_request_body: null,
      response_headers: null,
      response_body: null,
      created_at: '2026-03-20T10:01:00Z',
    })
    await flushPromises()

    expect(wrapper.find('.request-id').text()).toBe('req-2')
    expect(wrapper.find('.detail-id').text()).toBe('2')

    firstRequest.resolve({
      usage_log_id: 1,
      request_headers: null,
      request_body: null,
      upstream_request_headers: null,
      upstream_request_body: null,
      response_headers: null,
      response_body: null,
      created_at: '2026-03-20T10:00:00Z',
    })
    await flushPromises()

    expect(wrapper.find('.request-id').text()).toBe('req-2')
    expect(wrapper.find('.detail-id').text()).toBe('2')
  })

  it('does not write detail state after the modal is closed', async () => {
    const delayedRequest = createDeferred<AdminUsageDetail>()
    getDetail.mockReturnValue(delayedRequest.promise)

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          UsageStatsCards: true,
          UsageFilters: UsageFiltersStub,
          UsageTable: UsageTableStub,
          UsageDetailModal: UsageDetailModalStub,
          UsageExportProgress: true,
          UsageCleanupDialog: true,
          UserBalanceHistoryModal: true,
          Pagination: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenUsageTrend: true,
          ModelDistributionChart: ModelDistributionChartStub,
          GroupDistributionChart: GroupDistributionChartStub,
          EndpointDistributionChart: true,
        },
      },
    })

    await wrapper.find('[data-test="open-detail"]').trigger('click')
    await flushPromises()

    await wrapper.findComponent(UsageDetailModalStub).vm.$emit('close')
    await flushPromises()

    expect(wrapper.find('[data-test="usage-detail-modal"]').exists()).toBe(false)
    expect(wrapper.findComponent(UsageDetailModalStub).props('detail')).toBe(null)
    expect(wrapper.findComponent(UsageDetailModalStub).props('error')).toBe('')
    expect(wrapper.findComponent(UsageDetailModalStub).props('loading')).toBe(false)

    delayedRequest.resolve({
      usage_log_id: 42,
      request_headers: null,
      request_body: null,
      upstream_request_headers: null,
      upstream_request_body: null,
      response_headers: null,
      response_body: null,
      created_at: '2026-03-20T10:00:00Z',
    })
    await flushPromises()

    expect(wrapper.find('[data-test="usage-detail-modal"]').exists()).toBe(false)
    expect(wrapper.findComponent(UsageDetailModalStub).props('detail')).toBe(null)
    expect(wrapper.findComponent(UsageDetailModalStub).props('error')).toBe('')
    expect(wrapper.findComponent(UsageDetailModalStub).props('loading')).toBe(false)
  })
})
