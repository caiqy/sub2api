import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { UserSubscription } from '@/types'

const getMySubscriptions = vi.hoisted(() => vi.fn())
const advanceQuotaCycle = vi.hoisted(() => vi.fn())
const updateAutoQuotaAdvance = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())
const showSuccess = vi.hoisted(() => vi.fn())

vi.mock('@/api/subscriptions', () => ({
  default: { getMySubscriptions, updateAutoQuotaAdvance },
  advanceQuotaCycle,
  updateAutoQuotaAdvance,
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

  it.each([
    ['daily', { daily_usage_usd: 8.9999999999 }],
    ['unused monthly', {
      daily_usage_usd: 0,
      monthly_usage_usd: 0,
      monthly_window_start: '2026-07-31T11:00:00.000Z',
      expires_at: '2026-10-29T11:00:00.000Z',
      group: {
        ...makeSubscription().group!,
        allow_quota_advance: true,
        daily_limit_usd: null,
        weekly_limit_usd: null,
        monthly_limit_usd: 90,
      },
    }],
  ] as const)('keeps the %s action below the threshold and explains why reset is unavailable', async (_, overrides) => {
    getMySubscriptions.mockResolvedValue([makeSubscription(overrides)])

    const wrapper = mountView(false)
    await flushPromises()

    await wrapper.get('[data-test="advance-quota-1"]').trigger('click')

    expect(showError).toHaveBeenCalledWith('userSubscriptions.quotaAdvance.usageNotEligible')
    expect(wrapper.find('[data-test="confirm-advance"]').exists()).toBe(false)
    expect(advanceQuotaCycle).not.toHaveBeenCalled()
  })

  it('shows the action and opens dialog with insufficient validity warning when validity cannot cover the reset', async () => {
    getMySubscriptions.mockResolvedValue([makeSubscription({
      daily_usage_usd: 0,
      weekly_usage_usd: 70,
      weekly_window_start: '2026-07-29T12:00:00.000Z',
      expires_at: '2026-08-01T08:00:00.000Z',
    })])

    const wrapper = mountView(false)
    await flushPromises()

    expect(wrapper.find('[data-test="advance-quota-1"]').exists()).toBe(true)
    await wrapper.get('[data-test="advance-quota-1"]').trigger('click')

    expect(wrapper.find('[data-test="confirm-advance"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="confirm-advance"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-test="insufficient-validity"]').text()).toContain('userSubscriptions.quotaAdvance.insufficientValidity')
    expect(showError).not.toHaveBeenCalled()
  })

  it('hides the action and explains when multiple windows are exhausted', async () => {
    getMySubscriptions.mockResolvedValue([makeSubscription({
      weekly_usage_usd: 70,
      weekly_window_start: '2026-07-29T12:00:00.000Z',
    })])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="advance-quota-1"]').exists()).toBe(false)
    expect(wrapper.get('[data-test="advance-quota-multiple-1"]').text()).toContain(
      'userSubscriptions.quotaAdvance.multipleWindowsUnavailable',
    )
  })

  it('rejects unexpected openQuotaAdvance calls when quota windows are not eligible', async () => {
    getMySubscriptions.mockResolvedValue([makeSubscription()])
    const wrapper = mountView(false)
    await flushPromises()

    const zeroWindowSub = makeSubscription({ daily_usage_usd: 0 })
    ;(wrapper.vm as any).openQuotaAdvance(zeroWindowSub)
    expect(showError).toHaveBeenCalledWith('userSubscriptions.quotaAdvance.usageNotEligible')

    showError.mockReset()
    const multiWindowSub = makeSubscription({ weekly_usage_usd: 70, weekly_window_start: '2026-07-29T12:00:00.000Z' })
    ;(wrapper.vm as any).openQuotaAdvance(multiWindowSub)
    expect(showError).toHaveBeenCalledWith('userSubscriptions.quotaAdvance.multipleWindowsUnavailable')
  })

  it('opens the reset dialog with the persisted midnight window deduction', async () => {
    getMySubscriptions.mockResolvedValue([makeSubscription({ daily_window_start: '2026-07-31T00:00:00.000Z' })])
    const wrapper = mountView(false)
    await flushPromises()

    await wrapper.get('[data-test="advance-quota-1"]').trigger('click')

    expect(wrapper.get('[data-test="deducted-duration"]').text()).toBe('12h 0m')
    expect(wrapper.get('[data-test="confirm-advance"]').attributes('disabled')).toBeUndefined()
    expect(showError).not.toHaveBeenCalled()
    expect(advanceQuotaCycle).not.toHaveBeenCalled()
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
    expect(showSuccess).toHaveBeenCalledWith('userSubscriptions.quotaAdvance.success')
    expect(wrapper.find('[data-test="confirm-advance"]').exists()).toBe(false)
    await wrapper.get('[data-test="advance-quota-1"]').trigger('click')
    expect(showError).toHaveBeenCalledWith('userSubscriptions.quotaAdvance.usageNotEligible')
    expect(advanceQuotaCycle).toHaveBeenCalledOnce()
  })
})

describe('SubscriptionsView auto quota advance toggle', () => {
  beforeEach(() => {
    getMySubscriptions.mockReset()
    updateAutoQuotaAdvance.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('shows toggle when group has allow_quota_advance and auto_quota_advance enabled', async () => {
    getMySubscriptions.mockResolvedValue([makeSubscription({
      auto_quota_advance: false,
      group: {
        id: 20,
        name: 'Pro',
        allow_quota_advance: true,
        auto_quota_advance: true,
      } as any,
    })])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="auto-quota-advance-1"]').exists()).toBe(true)
  })

  it('hides toggle when group has auto_quota_advance disabled', async () => {
    getMySubscriptions.mockResolvedValue([makeSubscription({
      auto_quota_advance: false,
      group: {
        id: 20,
        name: 'Pro',
        allow_quota_advance: true,
        auto_quota_advance: false,
      } as any,
    })])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="auto-quota-advance-1"]').exists()).toBe(false)
  })

  it('hides toggle when group has allow_quota_advance disabled', async () => {
    getMySubscriptions.mockResolvedValue([makeSubscription({
      auto_quota_advance: false,
      group: {
        id: 20,
        name: 'Pro',
        allow_quota_advance: false,
        auto_quota_advance: true,
      } as any,
    })])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="auto-quota-advance-1"]').exists()).toBe(false)
  })

  it('toggling switch calls updateAutoQuotaAdvance API and shows success', async () => {
    const sub = makeSubscription({
      auto_quota_advance: false,
      group: {
        id: 20,
        name: 'Pro',
        allow_quota_advance: true,
        auto_quota_advance: true,
      } as any,
    })
    getMySubscriptions.mockResolvedValue([sub])
    updateAutoQuotaAdvance.mockResolvedValue({ ...sub, auto_quota_advance: true })

    const wrapper = mountView()
    await flushPromises()

    const toggle = wrapper.findComponent({ name: 'Toggle' })
    expect(toggle.exists()).toBe(true)
    await toggle.vm.$emit('update:modelValue', true)
    await flushPromises()

    expect(updateAutoQuotaAdvance).toHaveBeenCalledWith(1, true)
    expect(showSuccess).toHaveBeenCalledWith('userSubscriptions.autoQuotaAdvance.enabledSuccess')
  })

  it('keeps each subscription disabled until its own save finishes', async () => {
    const first = makeSubscription({ auto_quota_advance: false })
    first.group = { ...first.group!, allow_quota_advance: true, auto_quota_advance: true }
    const second = { ...first, id: 2 }
    getMySubscriptions.mockResolvedValue([first, second])
    let finishFirst!: (value: UserSubscription) => void
    let finishSecond!: (value: UserSubscription) => void
    updateAutoQuotaAdvance
      .mockReturnValueOnce(new Promise<UserSubscription>((resolve) => { finishFirst = resolve }))
      .mockReturnValueOnce(new Promise<UserSubscription>((resolve) => { finishSecond = resolve }))
    const wrapper = mountView()
    await flushPromises()

    const firstSwitch = wrapper.get('[data-test="auto-quota-advance-1"] [role="switch"]')
    const secondSwitch = wrapper.get('[data-test="auto-quota-advance-2"] [role="switch"]')
    await firstSwitch.trigger('click')
    await secondSwitch.trigger('click')
    expect(firstSwitch.attributes('disabled')).toBeDefined()
    expect(secondSwitch.attributes('disabled')).toBeDefined()

    finishFirst({ ...first, auto_quota_advance: true })
    await flushPromises()
    expect(firstSwitch.attributes('disabled')).toBeUndefined()
    expect(firstSwitch.attributes('aria-checked')).toBe('true')
    expect(secondSwitch.attributes('disabled')).toBeDefined()

    finishSecond({ ...second, auto_quota_advance: true })
    await flushPromises()
    expect(secondSwitch.attributes('disabled')).toBeUndefined()
    expect(secondSwitch.attributes('aria-checked')).toBe('true')
  })

  it('names the switch and preserves the preference when saving fails', async () => {
    const subscription = makeSubscription({ auto_quota_advance: false })
    subscription.group = { ...subscription.group!, allow_quota_advance: true, auto_quota_advance: true }
    getMySubscriptions.mockResolvedValue([subscription])
    updateAutoQuotaAdvance.mockRejectedValue({ response: { data: { message: 'Save failed' } } })
    const wrapper = mountView()
    await flushPromises()

    const toggle = wrapper.get('[data-test="auto-quota-advance-1"] [role="switch"]')
    expect(toggle.attributes('aria-label')).toBe('userSubscriptions.autoQuotaAdvance.title')
    await toggle.trigger('click')
    await flushPromises()
    expect(toggle.attributes('aria-checked')).toBe('false')
    expect(toggle.attributes('disabled')).toBeUndefined()
    expect(showError).toHaveBeenCalledWith('Save failed')
    expect(showSuccess).not.toHaveBeenCalled()
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
