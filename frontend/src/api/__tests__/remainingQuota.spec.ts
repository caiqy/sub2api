import { expect, it, vi } from 'vitest'
const { get, put } = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { get, put } }))
import { getRemainingQuota, saveQuotaExclusions } from '@/api/admin/users'

it('reads server-owned exclusions without a query override and saves them using PUT', async () => {
  const summary = { balance: 1, subscription: 2, total: 3, unlimited_subscriptions: 0, excluded_user_ids: [7] }
  get.mockResolvedValue({ data: summary })
  const signal = new AbortController().signal
  expect(await getRemainingQuota(signal)).toEqual(summary)
  expect(get).toHaveBeenCalledWith('/admin/users/remaining-quota', { signal })
  put.mockResolvedValue({ data: { excluded_user_ids: [] } })
  expect(await saveQuotaExclusions([])).toEqual({ excluded_user_ids: [] })
  expect(put).toHaveBeenCalledWith('/admin/users/remaining-quota/exclusions', { excluded_user_ids: [] })
})
