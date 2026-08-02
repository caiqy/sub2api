import { describe, expect, it } from 'vitest'
import type { UserSubscription } from '@/types'
import {
  getExhaustedQuotaWindows,
  getQuotaAdvancePreview,
} from '@/utils/subscriptionQuota'
import { getExpirationDateRelation, getRemainingExpiryDuration } from '../subscriptionQuota'

const now = new Date('2026-07-31T12:00:00.000Z')

describe('subscription quota advance', () => {
  it('finds active exhausted windows with future reset boundaries', () => {
    expect(getExhaustedQuotaWindows(makeSubscription(), now).map((window) => window.key)).toEqual([
      'daily',
      'weekly',
      'monthly',
    ])
  })

  it('excludes one-time daily quota because it has no next daily cycle', () => {
    const subscription = makeSubscription({
      starts_at: '2026-07-31T11:00:00.000Z',
      expires_at: '2026-08-01T11:00:00.000Z',
      weekly_window_start: '2026-07-24T13:00:00.000Z',
    })

    expect(getExhaustedQuotaWindows(subscription, now).map((window) => window.key)).toEqual([
      'weekly',
      'monthly',
    ])
  })

  it('does not create an advance preview for multiple exhausted windows', () => {
    const preview = getQuotaAdvancePreview(makeSubscription(), now)

    expect(preview).toEqual({
      deductedMs: 0,
      newExpiresAt: null,
      affordable: false,
    })
  })

  it('marks a single exhausted window unaffordable when its deduction exceeds remaining validity', () => {
    const subscription = makeSubscription({
      expires_at: '2026-08-03T12:00:00.000Z',
      daily_usage_usd: 0,
      monthly_usage_usd: 0,
    })

    expect(getExhaustedQuotaWindows(subscription, now).map((window) => window.key)).toEqual([
      'weekly',
    ])
    expect(getQuotaAdvancePreview(subscription, now)).toMatchObject({
      deductedMs: 5 * 24 * 60 * 60 * 1000,
      affordable: false,
    })
  })

  it('aligns legacy midnight windows to the original subscription anchor', () => {
    const subscription = makeSubscription({
      daily_window_start: '2026-07-31T00:00:00.000Z',
    })

    const windows = getExhaustedQuotaWindows(subscription, new Date('2026-07-31T08:00:00.000Z'))


    expect(windows.find((window) => window.key === 'daily')?.remainingMs).toBe(4 * 60 * 60 * 1000)
  })

  it('excludes a window more than one dollar below its limit', () => {
    const subscription = makeSubscription({
      daily_usage_usd: 0,
      weekly_usage_usd: 0,
      monthly_usage_usd: 298.9999999999,
    })

    expect(getExhaustedQuotaWindows(subscription, now)).toEqual([])
  })

  it.each([1, 0.5])('includes a zero-usage window whose positive limit is %s dollar', (limit) => {
    const subscription = makeSubscription({
      daily_usage_usd: 0,
      weekly_usage_usd: 0,
      monthly_usage_usd: 0,
    })
    subscription.group!.daily_limit_usd = 0
    subscription.group!.weekly_limit_usd = 0
    subscription.group!.monthly_limit_usd = limit

    expect(getExhaustedQuotaWindows(subscription, now).map((window) => window.key)).toEqual([
      'monthly',
    ])
  })

  it('includes an independently represented decimal value exactly one dollar below the limit', () => {
    const subscription = makeSubscription({
      daily_usage_usd: 0.1,
      weekly_usage_usd: 0,
      monthly_usage_usd: 0,
    })
    subscription.group!.daily_limit_usd = 1.1
    subscription.group!.weekly_limit_usd = 0
    subscription.group!.monthly_limit_usd = 0

    expect(getExhaustedQuotaWindows(subscription, now).map((window) => window.key)).toEqual(['daily'])
  })

  it('includes the shared large-magnitude machine-epsilon boundary', () => {
    const subscription = makeSubscription({
      daily_usage_usd: 999_998.9999999994,
      weekly_usage_usd: 0,
      monthly_usage_usd: 0,
    })
    subscription.group!.daily_limit_usd = 1_000_000
    subscription.group!.weekly_limit_usd = 0
    subscription.group!.monthly_limit_usd = 0

    expect(getExhaustedQuotaWindows(subscription, now).map((window) => window.key)).toEqual(['daily'])
  })
})

function makeSubscription(overrides: Partial<UserSubscription> = {}): UserSubscription {
  return {
    id: 1,
    user_id: 10,
    group_id: 20,
    status: 'active',
    starts_at: '2026-07-21T12:00:00.000Z',
    expires_at: '2026-09-09T12:00:00.000Z',
    daily_usage_usd: 9,
    weekly_usage_usd: 69,
    monthly_usage_usd: 299,
    daily_window_start: '2026-07-31T08:00:00.000Z',
    weekly_window_start: '2026-07-29T12:00:00.000Z',
    monthly_window_start: '2026-07-21T12:00:00.000Z',
    created_at: '2026-07-21T12:00:00.000Z',
    updated_at: '2026-07-31T11:00:00.000Z',
    group: {
      id: 20,
      name: 'Pro',
      description: null,
      platform: 'openai',
      rate_multiplier: 1,
      is_exclusive: false,
      status: 'active',
      subscription_type: 'subscription',
      daily_limit_usd: 10,
      weekly_limit_usd: 70,
      monthly_limit_usd: 300,
      allow_image_generation: false,
      allow_batch_image_generation: false,
      image_rate_independent: false,
      image_rate_multiplier: 1,
      batch_image_discount_multiplier: 1,
      batch_image_hold_multiplier: 1,
      image_price_1k: null,
      image_price_2k: null,
      image_price_4k: null,
      video_rate_independent: false,
      video_rate_multiplier: 1,
      video_price_480p: null,
      video_price_720p: null,
      video_price_1080p: null,
      web_search_price_per_call: null,
      peak_rate_enabled: false,
      peak_start: '',
      peak_end: '',
      peak_rate_multiplier: 1,
      claude_code_only: false,
      fallback_group_id: null,
      fallback_group_id_on_invalid_request: null,
      user_concurrency_enabled: false,
      user_concurrency_limit: 0,
      allow_live: false,
      require_oauth_only: false,
      require_privacy_set: false,
      created_at: '2026-07-21T12:00:00.000Z',
      updated_at: '2026-07-21T12:00:00.000Z',
    },
    ...overrides,
  }
}
describe('subscription expiry timing', () => {
  it('uses local calendar dates for today and tomorrow', () => {
    const now = new Date(2026, 2, 7, 23, 30)

    expect(getExpirationDateRelation(new Date(2026, 2, 7, 23, 45), now)).toBe('today')
    expect(getExpirationDateRelation(new Date(2026, 2, 8, 3, 30), now)).toBe('tomorrow')
  })

  it('treats the exact expiry instant and elapsed expiries as expired', () => {
    const now = new Date(2026, 6, 30, 9, 0)

    expect(getExpirationDateRelation(now, now)).toBe('expired')
    expect(getRemainingExpiryDuration(now, now)).toBeNull()
    expect(getExpirationDateRelation(new Date(2026, 6, 30, 8, 59), now)).toBe('expired')
    expect(getRemainingExpiryDuration(new Date(2026, 6, 30, 8, 59), now)).toBeNull()
  })

  it('rejects invalid target and current dates', () => {
    const invalid = new Date('invalid')
    const valid = new Date(2026, 6, 30, 9, 0)

    expect(getExpirationDateRelation(invalid, valid)).toBeNull()
    expect(getExpirationDateRelation(valid, invalid)).toBeNull()
    expect(getRemainingExpiryDuration(invalid, valid)).toBeNull()
    expect(getRemainingExpiryDuration(valid, invalid)).toBeNull()
  })

  it('returns rounded-up hours and minutes for an expiry under 24 hours away', () => {
    const now = new Date(2026, 6, 30, 9, 0)

    expect(getRemainingExpiryDuration(new Date(2026, 6, 31, 8, 30), now)).toEqual({
      unit: 'hoursMinutes',
      hours: 23,
      minutes: 30
    })
    expect(getRemainingExpiryDuration(new Date(now.getTime() + 1), now)).toEqual({
      unit: 'hoursMinutes',
      hours: 0,
      minutes: 1
    })
    expect(getRemainingExpiryDuration(new Date(now.getTime() + 23 * 60 * 60 * 1000 + 1), now)).toEqual({
      unit: 'hoursMinutes',
      hours: 23,
      minutes: 1
    })
  })

  it('preserves rounded-up day display from 24 hours onward', () => {
    const now = new Date(2026, 6, 30, 9, 0)

    expect(getRemainingExpiryDuration(new Date(now.getTime() + 24 * 60 * 60 * 1000), now)).toEqual({
      unit: 'days',
      days: 1
    })
    expect(getRemainingExpiryDuration(new Date(now.getTime() + 24 * 60 * 60 * 1000 + 1), now)).toEqual({
      unit: 'days',
      days: 2
    })
  })
})
