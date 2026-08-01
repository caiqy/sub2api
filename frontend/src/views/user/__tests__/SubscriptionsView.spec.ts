import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { UserSubscription } from '@/types'

const getMySubscriptions = vi.hoisted(() => vi.fn())
const advanceQuotaCycle = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())

vi.mock('@/api/subscriptions', () => ({
  default: { getMySubscriptions },
  advanceQuotaCycle,
}))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ cachedPublicSettings: null, showError, showSuccess }),
}))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

import SubscriptionsView from '../SubscriptionsView.vue'

describe('SubscriptionsView quota advance action', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-07-31T12:00:00.000Z'))
    getMySubscriptions.mockReset()
    advanceQuotaCycle.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('shows the action when the subscription has an advanceable exhausted window', async () => {
    getMySubscriptions.mockResolvedValue([makeSubscription()])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="advance-quota-1"]').exists()).toBe(true)
  })

  it('hides the action while quota remains', async () => {
    getMySubscriptions.mockResolvedValue([makeSubscription({ daily_usage_usd: 9 })])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="advance-quota-1"]').exists()).toBe(false)
  })

  it('keeps the action visible when an exhausted window needs more validity than remains', async () => {
    getMySubscriptions.mockResolvedValue([makeSubscription({
      daily_usage_usd: 0,
      weekly_usage_usd: 70,
      weekly_window_start: '2026-07-29T12:00:00.000Z',
      expires_at: '2026-08-03T12:00:00.000Z',
    })])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="advance-quota-1"]').exists()).toBe(true)
  })

  it('replaces the subscription with the successful reset response', async () => {
    const subscription = makeSubscription()
    getMySubscriptions.mockResolvedValue([subscription])
    advanceQuotaCycle.mockResolvedValue({
      subscription: makeSubscription({ daily_usage_usd: 0, expires_at: '2026-09-08T16:00:00.000Z' }),
      deducted_seconds: 72000,
    })
    const wrapper = mountView(false)
    await flushPromises()
    await wrapper.get('[data-test="advance-quota-1"]').trigger('click')

    await wrapper.get('[data-test="confirm-advance"]').trigger('click')
    await flushPromises()

    expect(advanceQuotaCycle).toHaveBeenCalledOnce()
    expect(wrapper.find('[data-test="advance-quota-1"]').exists()).toBe(false)
  })
})

function mountView(stubDialog = true) {
  return mount(SubscriptionsView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot/></main>' },
        Icon: true,
        SubscriptionQuotaAdvanceDialog: stubDialog,
        BaseDialog: {
          props: ['show', 'title'],
          emits: ['close'],
          template: '<div v-if="show"><slot/><slot name="footer"/></div>',
        },
      },
    },
  })
}

function makeSubscription(overrides: Partial<UserSubscription> = {}): UserSubscription {
  return {
    id: 1,
    user_id: 10,
    group_id: 20,
    status: 'active',
    starts_at: '2026-07-21T12:00:00.000Z',
    expires_at: '2026-09-09T12:00:00.000Z',
    daily_usage_usd: 10,
    weekly_usage_usd: 0,
    monthly_usage_usd: 0,
    daily_window_start: '2026-07-31T08:00:00.000Z',
    weekly_window_start: null,
    monthly_window_start: null,
    created_at: '2026-07-21T12:00:00.000Z',
    updated_at: '2026-07-31T11:00:00.000Z',
    group: {
      id: 20,
      name: 'Pro',
      description: null,
      platform: 'openai',
      rate_multiplier: 1,
      daily_limit_usd: 10,
      weekly_limit_usd: 70,
      monthly_limit_usd: null,
      peak_rate_enabled: false,
    } as UserSubscription['group'],
    ...overrides,
  }
}
