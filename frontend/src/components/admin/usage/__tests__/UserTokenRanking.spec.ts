import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

import UserTokenRanking from '../UserTokenRanking.vue'

const getUserBreakdown = vi.fn()

const { saveAsMock, aoaToSheetMock, bookNewMock, bookAppendSheetMock, xlsxWriteMock } = vi.hoisted(() => {
  return {
    saveAsMock: vi.fn(),
    aoaToSheetMock: vi.fn((data: unknown) => ({ data })),
    bookNewMock: vi.fn(() => ({})),
    bookAppendSheetMock: vi.fn(),
    xlsxWriteMock: vi.fn(() => new Uint8Array([1, 2, 3])),
  }
})

vi.mock('@/api/admin/dashboard', () => ({
  getUserBreakdown: (...args: unknown[]) => getUserBreakdown(...args),
}))

vi.mock('file-saver', () => ({
  saveAs: saveAsMock,
}))

vi.mock('xlsx', () => ({
  utils: {
    aoa_to_sheet: aoaToSheetMock,
    book_new: bookNewMock,
    book_append_sheet: bookAppendSheetMock,
  },
  write: xlsxWriteMock,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (params && 'count' in params) {
          return `${key}:${params.count}`
        }
        return key
      },
    }),
  }
})

const item = (id: number, tokens: number, username = `user_${id}`, actualCost = 0.5) => ({
  user_id: id,
  email: `u${id}@test.com`,
  username,
  requests: 10,
  input_tokens: tokens,
  output_tokens: 0,
  cache_tokens: 0,
  total_tokens: tokens,
  cost: actualCost,
  actual_cost: actualCost,
  account_cost: 0,
})

const mountRanking = (props: Record<string, unknown> = {}) =>
  mount(UserTokenRanking, {
    props: {
      startDate: '2026-07-01',
      endDate: '2026-07-08',
      filters: {},
      ...props,
    },
    global: { stubs: { Select: true, LoadingSpinner: true } },
  })

describe('UserTokenRanking', () => {
  beforeEach(() => {
    getUserBreakdown.mockReset()
    saveAsMock.mockReset()
    aoaToSheetMock.mockClear()
    bookNewMock.mockClear()
    bookAppendSheetMock.mockClear()
    xlsxWriteMock.mockClear()
    getUserBreakdown.mockResolvedValue({ users: [item(1, 100000000), item(2, 50000000)] })
  })

  it('loads on mount with shared filters and emits select-user with id + email on row click', async () => {
    const wrapper = mountRanking({ filters: { group_id: 3 }, model: 'claude-fable-5' })
    await flushPromises()

    expect(getUserBreakdown).toHaveBeenCalledWith(expect.objectContaining({
      group_id: 3,
      model: 'claude-fable-5',
      start_date: '2026-07-01',
      end_date: '2026-07-08',
      sort_by: 'total_tokens',
      limit: 50,
    }))

    const rows = wrapper.findAll('tbody tr')
    expect(rows).toHaveLength(2)

    await rows[0].trigger('click')
    expect(wrapper.emitted('select-user')![0]).toEqual([1, 'u1@test.com'])
  })

  it('reloads when shared filters change', async () => {
    const wrapper = mountRanking()
    await flushPromises()
    expect(getUserBreakdown).toHaveBeenCalledTimes(1)

    await wrapper.setProps({ filters: { user_id: 9 } })
    await flushPromises()

    expect(getUserBreakdown).toHaveBeenCalledTimes(2)
    expect(getUserBreakdown).toHaveBeenLastCalledWith(expect.objectContaining({ user_id: 9 }))
  })

  it('renders username column before user column and falls back to email or hyphen', async () => {
    getUserBreakdown.mockResolvedValue({
      users: [
        item(1, 100000000, 'alice'),
        { ...item(2, 50000000, ''), email: 'bob@test.com' },
        { ...item(3, 20000000, ''), email: '' },
      ],
    })

    const wrapper = mountRanking()
    await flushPromises()

    const headers = wrapper.findAll('thead th')
    // # -> username -> user -> ...
    expect(headers[0].text()).toBe('#')
    expect(headers[1].text()).toBe('admin.usage.tokenRanking.columns.username')
    expect(headers[2].text()).toBe('admin.usage.tokenRanking.columns.user')

    const rows = wrapper.findAll('tbody tr')
    expect(rows).toHaveLength(3)

    // Row 1: username present -> 'alice'
    const row1Cells = rows[0].findAll('td')
    expect(row1Cells[1].text()).toBe('alice')
    expect(row1Cells[2].text()).toContain('u1@test.com')

    // Row 2: username empty, email present -> fallback to 'bob@test.com'
    const row2Cells = rows[1].findAll('td')
    expect(row2Cells[1].text()).toBe('bob@test.com')
    expect(row2Cells[2].text()).toContain('bob@test.com')

    // Row 3: username empty, email empty -> '-'
    const row3Cells = rows[2].findAll('td')
    expect(row3Cells[1].text()).toBe('-')
  })

  it('controls export button disabled state based on loading and data availability', async () => {
    let resolvePromise!: (val: unknown) => void
    getUserBreakdown.mockReturnValue(new Promise((resolve) => { resolvePromise = resolve }))

    const wrapper = mountRanking()
    // In loading state
    const exportBtn = wrapper.find('button.btn-secondary')
    expect(exportBtn.exists()).toBe(true)
    expect(exportBtn.attributes('disabled')).toBeDefined()

    resolvePromise({ users: [] })
    await flushPromises()

    // No data state
    expect(exportBtn.attributes('disabled')).toBeDefined()

    // Data available
    getUserBreakdown.mockResolvedValue({ users: [item(1, 100)] })
    await (wrapper.vm as any).reload()
    await flushPromises()
    expect(exportBtn.attributes('disabled')).toBeUndefined()
  })

  it('exports excel with correct headers, 100M token scale, 4 decimal cost, and username fallback', async () => {
    getUserBreakdown.mockResolvedValue({
      users: [
        { ...item(1, 840000000, 'alice', 1778.2539), requests: 27643 },
        { ...item(2, 460000000, '', 1968.5476), email: 'wangpingmo@scqtkj.com', requests: 29315 },
      ],
    })

    const wrapper = mountRanking({
      startDate: '2026-09-01',
      endDate: '2026-09-10',
    })
    await flushPromises()

    const exportBtn = wrapper.find('button.btn-secondary')
    await exportBtn.trigger('click')
    await flushPromises()

    expect(aoaToSheetMock).toHaveBeenCalledTimes(1)
    const passedAoa = aoaToSheetMock.mock.calls[0][0] as unknown[][]

    // Check header
    expect(passedAoa[0]).toEqual([
      'admin.usage.tokenRanking.exportHeaders.username',
      'admin.usage.tokenRanking.exportHeaders.requests',
      'admin.usage.tokenRanking.exportHeaders.totalTokens',
      'admin.usage.tokenRanking.exportHeaders.billedCost',
    ])

    // Row 1: alice, requests: 27643, 8.4亿 (840,000,000 / 1e8 = 8.4), cost: 1778.2539, 4-decimal format
    expect(passedAoa[1]).toEqual([
      'alice',
      27643,
      { t: 'n', v: 8.4, z: '0.0000' },
      { t: 'n', v: 1778.2539, z: '0.0000' },
    ])

    // Row 2: username empty -> falls back to email 'wangpingmo@scqtkj.com'
    // 460,000,000 / 1e8 = 4.6
    expect(passedAoa[2]).toEqual([
      'wangpingmo@scqtkj.com',
      29315,
      { t: 'n', v: 4.6, z: '0.0000' },
      { t: 'n', v: 1968.5476, z: '0.0000' },
    ])

    expect(saveAsMock).toHaveBeenCalledTimes(1)
    const fileName = saveAsMock.mock.calls[0][1]
    expect(fileName).toBe('user_token_ranking_2026-09-01_to_2026-09-10.xlsx')
  })
})
