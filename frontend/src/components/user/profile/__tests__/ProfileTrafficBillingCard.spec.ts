import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import ProfileTrafficBillingCard from '../ProfileTrafficBillingCard.vue'

const { updateProfile } = vi.hoisted(() => ({
  updateProfile: vi.fn()
}))

const mockAuthUser = {
  id: 1,
  username: 'testuser',
  email: 'test@example.com',
  allow_large_requests: false
}

vi.mock('@/api', () => ({
  userAPI: { updateProfile }
}))
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user: mockAuthUser })
}))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: vi.fn(),
    showError: vi.fn(),
    cachedPublicSettings: {
      traffic_billing_enabled: true,
      traffic_billing_threshold_mb: 10,
      traffic_billing_price_per_mb: 0.05
    }
  })
}))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === 'profile.trafficBilling.label') return `启用超大请求体并接受流量费（${params?.price} $/MB）`
      if (key === 'profile.trafficBilling.thresholdInfo') return `起步包免费门槛：${params?.threshold} MB`
      return key
    }
  })
}))

enableAutoUnmount(afterEach)

describe('ProfileTrafficBillingCard', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    mockAuthUser.allow_large_requests = false
    updateProfile.mockResolvedValue({ ...mockAuthUser, allow_large_requests: true })
  })

  it('renders default false state and correct price label', () => {
    const wrapper = mount(ProfileTrafficBillingCard, {
      props: {
        user: mockAuthUser,
        trafficBillingEnabled: true,
        trafficBillingPricePerMB: 0.05,
        trafficBillingThresholdMB: 10
      }
    })

    const toggle = wrapper.find('[data-testid="allow-large-requests-toggle"]')
    expect(toggle.exists()).toBe(true)
    expect(toggle.attributes('aria-checked')).toBe('false')
    expect(toggle.attributes('aria-label')).toBe('启用超大请求体并接受流量费（0.05 $/MB）')
    expect(wrapper.text()).toContain('启用超大请求体并接受流量费（0.05 $/MB）')
    expect(wrapper.text()).toContain('起步包免费门槛：10 MB')
  })

  it('updates profile when toggle is clicked', async () => {
    const wrapper = mount(ProfileTrafficBillingCard, {
      props: {
        user: mockAuthUser,
        trafficBillingEnabled: true,
        trafficBillingPricePerMB: 0.05,
        trafficBillingThresholdMB: 10
      }
    })

    const toggleBtn = wrapper.find('[data-testid="allow-large-requests-toggle"]')
    await toggleBtn.trigger('click')
    await flushPromises()

    expect(updateProfile).toHaveBeenCalledWith({
      allow_large_requests: true
    })
  })

  it('hides disabledNotice when trafficBillingEnabled is true and shows it when false', async () => {
    // Enabled via prop
    const wrapperEnabled = mount(ProfileTrafficBillingCard, {
      props: {
        user: mockAuthUser,
        trafficBillingEnabled: true,
        trafficBillingThresholdMB: 10,
        trafficBillingPricePerMB: 0.05
      }
    })
    expect(wrapperEnabled.text()).not.toContain('profile.trafficBilling.disabledNotice')

    // Disabled via prop
    const wrapperDisabled = mount(ProfileTrafficBillingCard, {
      props: {
        user: mockAuthUser,
        trafficBillingEnabled: false,
        trafficBillingThresholdMB: 10,
        trafficBillingPricePerMB: 0.05
      }
    })
    expect(wrapperDisabled.text()).toContain('profile.trafficBilling.disabledNotice')
  })
})
