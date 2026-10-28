import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post, patch } = vi.hoisted(() => ({ post: vi.fn(), patch: vi.fn() }))

vi.mock('@/api/client', () => ({ apiClient: { post, patch } }))

import { advanceQuotaCycle, updateAutoQuotaAdvance } from '@/api/subscriptions'

describe('subscription quota advance API', () => {
  beforeEach(() => {
    post.mockReset()
    vi.spyOn(globalThis.crypto, 'randomUUID').mockReturnValue('11111111-1111-4111-8111-111111111111')
  })

  it('sends the selected windows and returns the server result', async () => {
    post.mockResolvedValue({ data: { subscription: { id: 7 }, deducted_seconds: 72000 } })

    const result = await advanceQuotaCycle(7, { daily: true, weekly: false, monthly: false })

    expect(post).toHaveBeenCalledWith(
      '/subscriptions/7/advance-quota-cycle',
      { daily: true, weekly: false, monthly: false },
      { headers: { 'Idempotency-Key': 'subscription-quota-advance-7-d---11111111-1111-4111-8111-111111111111' } },
    )
    expect(result.deducted_seconds).toBe(72000)
  })

  it('reuses the operation key after an ambiguous failed request', async () => {
    post.mockRejectedValueOnce(new Error('network timeout'))
    await expect(advanceQuotaCycle(9, { daily: true, weekly: false, monthly: false })).rejects.toThrow('network timeout')

    post.mockResolvedValueOnce({ data: { subscription: { id: 9 }, deducted_seconds: 72000 } })
    await advanceQuotaCycle(9, { daily: true, weekly: false, monthly: false })

    expect(post.mock.calls[1][2].headers).toEqual(post.mock.calls[0][2].headers)
  })

  it('updates auto quota advance setting via patch request', async () => {
    patch.mockResolvedValueOnce({ data: { id: 7, auto_quota_advance: true } })

    const result = await updateAutoQuotaAdvance(7, true)

    expect(patch).toHaveBeenCalledWith(
      '/subscriptions/7/auto-quota-advance',
      { auto_quota_advance: true },
    )
    expect(result.auto_quota_advance).toBe(true)
  })
})
