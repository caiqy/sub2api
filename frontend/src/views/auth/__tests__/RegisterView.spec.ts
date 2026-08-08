import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RegisterView from '@/views/auth/RegisterView.vue'

const { getPublicSettingsMock, startOAuthLoginMock, verifyActionMock } = vi.hoisted(() => ({
  getPublicSettingsMock: vi.fn(),
  startOAuthLoginMock: vi.fn(),
  verifyActionMock: vi.fn()
}))

const locationState = { href: 'http://localhost/register' }

const publicSettings = {
  registration_enabled: true,
  email_verify_enabled: false,
  promo_code_enabled: false,
  invitation_code_enabled: false,
  affiliate_enabled: true,
  turnstile_enabled: true,
  turnstile_site_key: 'site-key',
  site_name: 'Sub2API',
  registration_email_suffix_whitelist: [],
  linuxdo_oauth_enabled: false,
  wechat_oauth_enabled: false,
  oidc_oauth_enabled: false,
  github_oauth_enabled: false,
  google_oauth_enabled: false
}

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
  useRoute: () => ({ query: {} })
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key
    }
  }),
  useI18n: () => ({
    t: (key: string) => key,
    locale: { value: 'en' }
  })
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({ register: vi.fn() }),
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn()
  })
}))

vi.mock('@/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/api/auth')>('@/api/auth')
  return {
    ...actual,
    getPublicSettings: (...args: unknown[]) => getPublicSettingsMock(...args),
    startOAuthLogin: (...args: unknown[]) => startOAuthLoginMock(...args)
  }
})

const CaptchaChallengeStub = defineComponent({
  setup(_, { expose }) {
    expose({ verifyAction: verifyActionMock, reset: vi.fn() })
    return () => h('div')
  }
})

const OAuthButtonStub = defineComponent({
  emits: ['start'],
  setup(_, { emit }) {
    return () => h('button', {
      type: 'button',
      'data-testid': 'oauth-start',
      onClick: () => emit('start', { provider: 'github', params: { aff_code: '' } })
    })
  }
})

function mountRegister() {
  return mount(RegisterView, {
    global: {
      stubs: {
        AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
        Icon: true,
        TurnstileWidget: CaptchaChallengeStub,
        LoginAgreementPrompt: true,
        EmailOAuthButtons: OAuthButtonStub,
        LinuxDoOAuthSection: true,
        WechatOAuthSection: true,
        OidcOAuthSection: true,
        RouterLink: true,
        transition: false
      }
    }
  })
}

describe('RegisterView invitation layout', () => {
  beforeEach(() => {
    getPublicSettingsMock.mockReset()
    startOAuthLoginMock.mockReset()
    verifyActionMock.mockReset()
    getPublicSettingsMock.mockResolvedValue(publicSettings)
    startOAuthLoginMock.mockResolvedValue({ authorize_url: 'https://github.example/authorize' })
    verifyActionMock.mockResolvedValue({ token: 'turnstile-token', randstr: '' })
    locationState.href = 'http://localhost/register'
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: locationState
    })
  })

  it('keeps the optional affiliate invitation field before Turnstile', async () => {
    const wrapper = mountRegister()
    await flushPromises()

    const invitationField = wrapper.get('[data-testid="affiliate-invitation-field"]')
    const turnstile = wrapper.get('[data-testid="registration-turnstile"]')

    expect(invitationField.get('input').attributes('id')).toBe('affiliate_code')
    expect(invitationField.text()).toContain('common.optional')
    expect(
      invitationField.element.compareDocumentPosition(turnstile.element) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy()
  })

  it('uses the mandatory invitation field without duplicating the affiliate field', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      invitation_code_enabled: true
    })

    const wrapper = mountRegister()
    await flushPromises()

    expect(wrapper.find('[data-testid="affiliate-invitation-field"]').exists()).toBe(false)
    expect(wrapper.get('#invitation_code').exists()).toBe(true)
  })

  it('starts OAuth through the Turnstile gate with its completed token', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      github_oauth_enabled: true
    })
    const wrapper = mountRegister()
    await flushPromises()

    await wrapper.get('[data-testid="oauth-start"]').trigger('click')
    await flushPromises()

    expect(verifyActionMock).toHaveBeenCalledOnce()
    expect(startOAuthLoginMock).toHaveBeenCalledWith(
      { provider: 'github', params: { aff_code: '' } },
      { turnstile_token: 'turnstile-token' }
    )
    expect(locationState.href).toBe('https://github.example/authorize')
  })

  it('does not start OAuth when Turnstile has no completed token', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      github_oauth_enabled: true
    })
    verifyActionMock.mockResolvedValue(null)
    const wrapper = mountRegister()
    await flushPromises()

    await wrapper.get('[data-testid="oauth-start"]').trigger('click')
    await flushPromises()

    expect(verifyActionMock).toHaveBeenCalledOnce()
    expect(startOAuthLoginMock).not.toHaveBeenCalled()
    expect(locationState.href).toBe('http://localhost/register')
  })
})
