import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))

vi.mock('@/api/client', () => ({ apiClient: { post } }))

import { advanceQuotaCycle } from '@/api/subscriptions'

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
    await expect(advanceQuotaCycle(9, { daily: true, weekly: true, monthly: false })).rejects.toThrow('network timeout')

    post.mockResolvedValueOnce({ data: { subscription: { id: 9 }, deducted_seconds: 432000 } })
    await advanceQuotaCycle(9, { daily: true, weekly: true, monthly: false })

    expect(post.mock.calls[1][2].headers).toEqual(post.mock.calls[0][2].headers)
  })
})
