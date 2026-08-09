import { describe, expect, expectTypeOf, it, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'

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

const { list, exportList, getStats, getDetail, getSnapshotV2, getModelStats, getById, listErrorLogs, showError, routeQuery, aoaToSheet, sheetAddAoa, saveAs, xlsxWrite } = vi.hoisted(() => {
  vi.stubGlobal('localStorage', {
    getItem: vi.fn(() => null),
    setItem: vi.fn(),
    removeItem: vi.fn(),
  })

  return {
    list: vi.fn(),
    exportList: vi.fn(),
    getStats: vi.fn(),
    getDetail: vi.fn(),
    getSnapshotV2: vi.fn(),
    getModelStats: vi.fn(),
    getById: vi.fn(),
    listErrorLogs: vi.fn(),
    showError: vi.fn(),
    routeQuery: {} as Record<string, string>,
    aoaToSheet: vi.fn(() => ({})),
    sheetAddAoa: vi.fn(),
    saveAs: vi.fn(),
    xlsxWrite: vi.fn(() => new Uint8Array([1, 2, 3])),
  }
})

const messages: Record<string, string> = {
  'admin.dashboard.timeRange': 'Time Range',
  'admin.dashboard.day': 'Day',
  'admin.dashboard.hour': 'Hour',
  'admin.usage.failedToLoadUser': 'Failed to load user',
  'admin.usage.detailNotFound': 'Detail not found',
  'usage.requestedModel': 'Requested model',
  'usage.sentUpstreamModel': 'Sent upstream model',
  'usage.upstreamResponseModel': 'Upstream response model',
  'usage.upstreamModelMismatch': 'Upstream model mismatch',
  'common.yes': 'Yes',
  'common.no': 'No',
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
    list: exportList,
    getDetail: vi.fn(),
  },
  default: {
    list: exportList,
    getDetail: vi.fn(),
  },
}))

vi.mock('@/api/admin/ops', () => ({ listErrorLogs }))

vi.mock('file-saver', () => ({ saveAs }))

vi.mock('xlsx', () => ({
  utils: {
    aoa_to_sheet: aoaToSheet,
    sheet_add_aoa: sheetAddAoa,
    book_new: vi.fn(() => ({})),
    book_append_sheet: vi.fn(),
  },
  write: xlsxWrite,
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
    query: routeQuery
  })
}))

const AppLayoutStub = { template: '<div><slot /></div>' }
const UsageFiltersStub = defineComponent({
  props: ['modelValue'],
  emits: ['update:modelValue', 'change'],
  setup(_, { expose }) {
    const userKeyword = ref('')
    let userSearchRevision = 0
    const setUserKeyword = (email: string) => {
      userSearchRevision += 1
      userKeyword.value = email
    }
    expose({
      getUserSearchRevision: () => userSearchRevision,
      setUserKeyword,
      simulateUserInput: setUserKeyword,
    })
    return { userKeyword }
  },
  template: '<div><span data-test="user-filter-label">{{ userKeyword }}</span><slot name="after-reset" /></div>',
})
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
const UserTokenRankingStub = {
  emits: ['select-user'],
  template: '<div data-test="ranking"><button class="pick-user" @click="$emit(\'select-user\', 5, \'rank@test.com\')">pick</button></div>',
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

const mountRouteFilteredUsageView = () => mount(UsageView, {
  global: { stubs: {
    AppLayout: AppLayoutStub, UsageStatsCards: true, UsageFilters: UsageFiltersStub,
    UsageTable: true, UsageExportProgress: true, UsageCleanupDialog: true,
    UserBalanceHistoryModal: true, Pagination: true, Select: true,
    DateRangePicker: true, Icon: true, TokenUsageTrend: true,
    ModelDistributionChart: true, GroupDistributionChart: true,
    EndpointDistributionChart: true, UserTokenRanking: true,
  } },
})

describe('admin UsageView route filters', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    Object.keys(routeQuery).forEach((key) => delete routeQuery[key])
    list.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockReset().mockResolvedValue({
      total_requests: 0,
      total_input_tokens: 0,
      total_output_tokens: 0,
      total_cache_tokens: 0,
      total_tokens: 0,
      total_cost: 0,
      total_actual_cost: 0,
      average_duration_ms: 0,
    })
    getSnapshotV2.mockReset().mockResolvedValue({ trend: [], models: [], groups: [] })
    getModelStats.mockReset().mockResolvedValue({ models: [] })
    getById.mockReset()
  })

  afterEach(() => {
    Object.keys(routeQuery).forEach((key) => delete routeQuery[key])
    vi.useRealTimers()
  })

  it('shows the routed user while applying user_id to usage requests', async () => {
    routeQuery.user_id = '42'
    getById.mockResolvedValue({ id: 42, email: 'route-user@test.com' })

    const wrapper = mountRouteFilteredUsageView()
    await flushPromises()

    expect(getById).toHaveBeenCalledWith(42, true)
    expect(list).toHaveBeenCalledWith(expect.objectContaining({ user_id: 42 }), expect.anything())
    expect(wrapper.find('[data-test="user-filter-label"]').text()).toBe('route-user@test.com')
  })

  it('does not apply a stale routed user label after user_id changes', async () => {
    routeQuery.user_id = '42'
    let resolveLookup!: (user: { id: number; email: string }) => void
    getById.mockReturnValue(new Promise((resolve) => { resolveLookup = resolve }))

    const wrapper = mountRouteFilteredUsageView()
    await wrapper.vm.$nextTick()
    ;(wrapper.vm as any).filters.user_id = 84
    ;(wrapper.findComponent(UsageFiltersStub).vm as any).setUserKeyword('current-user@test.com')

    resolveLookup({ id: 42, email: 'stale-user@test.com' })
    await flushPromises()

    expect(wrapper.find('[data-test="user-filter-label"]').text()).toBe('current-user@test.com')
  })

  it('does not overwrite newer user input when the routed user lookup succeeds', async () => {
    routeQuery.user_id = '42'
    let resolveLookup!: (user: { id: number; email: string }) => void
    getById.mockReturnValue(new Promise((resolve) => { resolveLookup = resolve }))

    const wrapper = mountRouteFilteredUsageView()
    await wrapper.vm.$nextTick()
    ;(wrapper.findComponent(UsageFiltersStub).vm as any).simulateUserInput('new-search@test.com')

    resolveLookup({ id: 42, email: 'route-user@test.com' })
    await flushPromises()

    expect((wrapper.vm as any).filters.user_id).toBe(42)
    expect(wrapper.find('[data-test="user-filter-label"]').text()).toBe('new-search@test.com')
  })

  it('does not overwrite newer user input when the routed user lookup fails', async () => {
    routeQuery.user_id = '42'
    let rejectLookup!: (error: Error) => void
    getById.mockReturnValue(new Promise((_, reject) => { rejectLookup = reject }))

    const wrapper = mountRouteFilteredUsageView()
    await wrapper.vm.$nextTick()
    ;(wrapper.findComponent(UsageFiltersStub).vm as any).simulateUserInput('new-search@test.com')

    rejectLookup(new Error('lookup failed'))
    await flushPromises()

    expect((wrapper.vm as any).filters.user_id).toBe(42)
    expect(wrapper.find('[data-test="user-filter-label"]').text()).toBe('new-search@test.com')
  })

  it('shows the routed user ID when its label lookup fails', async () => {
    routeQuery.user_id = '42'
    getById.mockRejectedValue(new Error('lookup failed'))

    const wrapper = mountRouteFilteredUsageView()
    await flushPromises()

    expect(list).toHaveBeenCalledWith(expect.objectContaining({ user_id: 42 }), expect.anything())
    expect(wrapper.find('[data-test="user-filter-label"]').text()).toBe('42')
  })
})

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

  it('keeps previous model stats visible during refresh until new data arrives', async () => {
    // 首次加载返回 A
    getModelStats.mockResolvedValueOnce({ models: [{ model: 'A', total_tokens: 10 }] })

    const wrapper = mount(UsageView, {
      global: { stubs: {
        AppLayout: AppLayoutStub, UsageStatsCards: true, UsageFilters: UsageFiltersStub,
        UsageTable: true, UsageExportProgress: true, UsageCleanupDialog: true,
        UserBalanceHistoryModal: true, AuditLogModal: true, Pagination: true, Select: true,
        DateRangePicker: true, Icon: true, TokenUsageTrend: true,
        ModelDistributionChart: ModelDistributionChartStub, GroupDistributionChart: GroupDistributionChartStub,
        EndpointDistributionChart: true, UserTokenRanking: true,
      } },
    })
    vi.advanceTimersByTime(120)
    await flushPromises()
    expect((wrapper.vm as any).requestedModelStats).toEqual([{ model: 'A', total_tokens: 10 }])

    // 刷新:让第二次 getModelStats 处于 pending,断言旧数据 A 仍在(不被清空成 [])
    let resolveSecond: (v: any) => void = () => {}
    getModelStats.mockReturnValueOnce(new Promise((res) => { resolveSecond = res }))
    ;(wrapper.vm as any).refreshData()
    await flushPromises()
    expect((wrapper.vm as any).requestedModelStats).toEqual([{ model: 'A', total_tokens: 10 }])

    // 新数据到达后替换为 B
    resolveSecond({ models: [{ model: 'B', total_tokens: 20 }] })
    await flushPromises()
    expect((wrapper.vm as any).requestedModelStats).toEqual([{ model: 'B', total_tokens: 20 }])
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
          UserTokenRanking: true,
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
          UserTokenRanking: true,
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
    listErrorLogs.mockResolvedValue({ items: [], total: 0 })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('forwards model/account_id/group_id to listErrorLogs on the errors tab', async () => {
    vi.useFakeTimers()
    const wrapper = mount(UsageView, {
      global: { stubs: {
        AppLayout: AppLayoutStub, UsageStatsCards: true, UsageFilters: UsageFiltersStub,
        UsageTable: true, UsageExportProgress: true, UsageCleanupDialog: true,
        UserBalanceHistoryModal: true, AuditLogModal: true, Pagination: true, Select: true,
        DateRangePicker: true, Icon: true, TokenUsageTrend: true,
        ModelDistributionChart: true, GroupDistributionChart: true, EndpointDistributionChart: true,
        UserTokenRanking: true, OpsErrorLogTable: true, OpsErrorDetailModal: true,
      } },
    })

    vi.advanceTimersByTime(120)
    await flushPromises()
    const vm = wrapper.vm as any
    vm.filters.model = 'gpt-5.3-codex'
    vm.filters.account_id = 7
    vm.filters.group_id = 3
    const tabs = wrapper.findAll('[data-testid="usage-detail-tab"]')
    await tabs[1].trigger('click')
    await flushPromises()

    expect(listErrorLogs).toHaveBeenCalledWith(expect.objectContaining({
      view: 'all',
      model: 'gpt-5.3-codex',
      account_id: 7,
      group_id: 3,
    }))
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

describe('admin UsageView ranking tab', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    list.mockReset()
    getStats.mockReset()
    getSnapshotV2.mockReset()
    getModelStats.mockReset()

    list.mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockResolvedValue({
      total_requests: 0, total_input_tokens: 0, total_output_tokens: 0,
      total_cache_tokens: 0, total_tokens: 0, total_cost: 0, total_actual_cost: 0, average_duration_ms: 0,
    })
    getSnapshotV2.mockResolvedValue({ trend: [], models: [], groups: [] })
    getModelStats.mockResolvedValue({ models: [] })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('mounts ranking lazily and drill-down sets user filter then jumps back to usage tab', async () => {
    const wrapper = mount(UsageView, {
      global: { stubs: {
        AppLayout: AppLayoutStub, UsageStatsCards: true, UsageFilters: UsageFiltersStub,
        UsageTable: true, UsageExportProgress: true, UsageCleanupDialog: true,
        UserBalanceHistoryModal: true, Pagination: true, Select: true,
        DateRangePicker: true, Icon: true, TokenUsageTrend: true,
        ModelDistributionChart: true, GroupDistributionChart: true, EndpointDistributionChart: true,
        UserTokenRanking: UserTokenRankingStub, OpsErrorLogTable: true, OpsErrorDetailModal: true,
      } },
    })
    vi.advanceTimersByTime(120)
    await flushPromises()

    // 懒挂载:切到排行 tab 前不渲染
    expect(wrapper.find('[data-test="ranking"]').exists()).toBe(false)

    const tabs = wrapper.findAll('[data-testid="usage-detail-tab"]')
    expect(tabs).toHaveLength(3)
    await tabs[2].trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-test="ranking"]').exists()).toBe(true)

    // 下钻:设置 user_id、切回用量明细 tab 并按新筛选重新拉取列表
    list.mockClear()
    await wrapper.find('[data-test="ranking"] .pick-user').trigger('click')
    await flushPromises()

    expect((wrapper.vm as any).activeTab).toBe('usage')
    expect((wrapper.vm as any).filters.user_id).toBe(5)
    expect(list).toHaveBeenCalledWith(expect.objectContaining({ user_id: 5 }), expect.anything())
  })
})

describe('admin UsageView model audit export', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    list.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    exportList.mockReset().mockResolvedValue({
      items: [{
        id: 1,
        created_at: '2026-08-04T00:00:00Z',
        model: 'gpt-5.6-sol',
        upstream_model: 'gpt-5.5',
        upstream_response_model: 'gpt-5.4',
        upstream_model_mismatch: true,
        request_type: 'sync',
        input_tokens: 1,
        output_tokens: 1,
        cache_read_tokens: 0,
        cache_creation_tokens: 0,
        duration_ms: 10,
      }],
      total: 1,
      pages: 1,
    })
    getStats.mockReset().mockResolvedValue({
      total_requests: 0, total_input_tokens: 0, total_output_tokens: 0,
      total_cache_tokens: 0, total_tokens: 0, total_cost: 0, total_actual_cost: 0, average_duration_ms: 0,
    })
    getSnapshotV2.mockReset().mockResolvedValue({ trend: [], models: [], groups: [] })
    getModelStats.mockReset().mockResolvedValue({ models: [] })
    aoaToSheet.mockClear()
    sheetAddAoa.mockClear()
    saveAs.mockClear()
    xlsxWrite.mockClear()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('exports requested, sent, response, and mismatch as separate admin columns', async () => {
    const wrapper = mountRouteFilteredUsageView()
    vi.advanceTimersByTime(120)
    await flushPromises()

    await (wrapper.vm as any).exportToExcel()
    await flushPromises()

    const headers = aoaToSheet.mock.calls[0][0][0]
    expect(headers.slice(4, 8)).toEqual([
      'Requested model',
      'Sent upstream model',
      'Upstream response model',
      'Upstream model mismatch',
    ])
    const row = sheetAddAoa.mock.calls[0][1][0]
    expect(row.slice(4, 8)).toEqual(['gpt-5.6-sol', 'gpt-5.5', 'gpt-5.4', 'Yes'])
    expect(saveAs).toHaveBeenCalledTimes(1)
  })
})
