import { describe, expect, it } from 'vitest'
import type { UserSubscription } from '@/types'
import {
  getExhaustedQuotaWindows,
  getQuotaAdvancePreview,
} from '@/utils/subscriptionQuota'

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
    ])
  })

  it('uses the longest selected remaining time and reports unselected exhausted windows', () => {
    const preview = getQuotaAdvancePreview(makeSubscription(), ['daily', 'weekly'], now)

    expect(preview).toEqual({
      deductedMs: 5 * 24 * 60 * 60 * 1000,
      newExpiresAt: '2026-09-04T12:00:00.000Z',
      unselectedExhausted: ['monthly'],
    })
  })

  it('aligns legacy midnight windows to the original subscription anchor', () => {
    const subscription = makeSubscription({
      daily_window_start: '2026-07-31T00:00:00.000Z',
    })

    const windows = getExhaustedQuotaWindows(subscription, new Date('2026-07-31T08:00:00.000Z'))

    expect(windows.find((window) => window.key === 'daily')?.remainingMs).toBe(4 * 60 * 60 * 1000)
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
    daily_usage_usd: 10,
    weekly_usage_usd: 70,
    monthly_usage_usd: 300,
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
