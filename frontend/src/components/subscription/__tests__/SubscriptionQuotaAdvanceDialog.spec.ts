import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { UserSubscription } from '@/types'

const advanceQuotaCycle = vi.hoisted(() => vi.fn())
const showError = vi.hoisted(() => vi.fn())

vi.mock('@/api/subscriptions', () => ({ advanceQuotaCycle }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => params?.windows ? `${key}:${params.windows}` : key,
    }),
  }
})

import SubscriptionQuotaAdvanceDialog from '../SubscriptionQuotaAdvanceDialog.vue'

const BaseDialogStub = {
  props: ['show', 'title'],
  emits: ['close'],
  template: '<div v-if="show"><h2>{{ title }}</h2><slot/><slot name="footer"/></div>',
}

describe('SubscriptionQuotaAdvanceDialog', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-07-31T12:00:00.000Z'))
    advanceQuotaCycle.mockReset()
    showError.mockReset()
  })

  it('lists all exhausted windows as one mandatory reset and confirms the full set', async () => {
    const wrapper = mountDialog()

    expect(wrapper.findAll('input[type="checkbox"]')).toHaveLength(0)
    expect(wrapper.text()).toContain('userSubscriptions.daily')
    expect(wrapper.text()).toContain('userSubscriptions.weekly')
    expect(wrapper.text()).toContain('userSubscriptions.monthly')
    expect(wrapper.get('[data-test="confirm-advance"]').attributes('disabled')).toBeUndefined()
    expect(wrapper.get('[data-test="deducted-duration"]').text()).toContain('20d')
  })

  it('cancels without sending a request', async () => {
    const wrapper = mountDialog()

    await wrapper.get('[data-test="cancel-advance"]').trigger('click')

    expect(advanceQuotaCycle).not.toHaveBeenCalled()
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('submits all displayed exhausted windows once and emits the updated subscription', async () => {
    const updated = makeSubscription({ daily_usage_usd: 0, expires_at: '2026-09-08T16:00:00.000Z' })
    advanceQuotaCycle.mockResolvedValue({ subscription: updated, deducted_seconds: 72000 })
    const wrapper = mountDialog()

    await wrapper.get('[data-test="confirm-advance"]').trigger('click')
    await flushPromises()

    expect(advanceQuotaCycle).toHaveBeenCalledWith(1, { daily: true, weekly: true, monthly: true })
    expect(wrapper.emitted('success')?.[0]).toEqual([updated])
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('shows the message from a flat API interceptor error', async () => {
    advanceQuotaCycle.mockRejectedValue({
      status: 409,
      code: 'QUOTA_ADVANCE_STATE_CHANGED',
      message: 'quota window state changed; reopen the confirmation dialog',
    })
    const wrapper = mountDialog()

    await wrapper.get('[data-test="confirm-advance"]').trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('quota window state changed; reopen the confirmation dialog')
  })

  it('falls back to the generic quota advance error when the API error has no message', async () => {
    advanceQuotaCycle.mockRejectedValue({})
    const wrapper = mountDialog()

    await wrapper.get('[data-test="confirm-advance"]').trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('userSubscriptions.quotaAdvance.failed')
  })

  it('shows all exhausted windows but disables confirmation when the validity cannot cover them', () => {
    const wrapper = mountDialog(makeSubscription({
      expires_at: '2026-08-03T12:00:00.000Z',
      monthly_usage_usd: 0,
    }))

    expect(wrapper.text()).toContain('userSubscriptions.daily')
    expect(wrapper.text()).toContain('userSubscriptions.weekly')
    expect(wrapper.get('[data-test="insufficient-validity"]').text()).toContain('userSubscriptions.quotaAdvance.insufficientValidity')
    expect(wrapper.get('[data-test="confirm-advance"]').attributes('disabled')).toBeDefined()
  })
})

function mountDialog(subscription = makeSubscription()) {
  return mount(SubscriptionQuotaAdvanceDialog, {
    props: { show: true, subscription },
    global: { stubs: { BaseDialog: BaseDialogStub, Icon: true } },
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
      daily_limit_usd: 10,
      weekly_limit_usd: 70,
      monthly_limit_usd: 300,
    } as UserSubscription['group'],
    ...overrides,
  }
}
