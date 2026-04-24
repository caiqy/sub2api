import { computed, defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  updateAccountMock,
  clearErrorMock,
  showSuccessMock,
  showErrorMock,
  exchangeAuthCodeMock,
  buildCredentialsMock,
  buildExtraInfoMock
} = vi.hoisted(() => ({
  updateAccountMock: vi.fn(),
  clearErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
  exchangeAuthCodeMock: vi.fn(),
  buildCredentialsMock: vi.fn(),
  buildExtraInfoMock: vi.fn()
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: showSuccessMock,
    showError: showErrorMock
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      update: updateAccountMock,
      clearError: clearErrorMock
    }
  }
}))

vi.mock('@/composables/useAccountOAuth', () => ({
  useAccountOAuth: () => ({
    authUrl: ref(''),
    sessionId: ref(''),
    loading: ref(false),
    error: ref(''),
    resetState: vi.fn()
  })
}))

vi.mock('@/composables/useGeminiOAuth', () => ({
  useGeminiOAuth: () => ({
    authUrl: ref(''),
    sessionId: ref(''),
    state: ref(''),
    loading: ref(false),
    error: ref(''),
    resetState: vi.fn()
  })
}))

vi.mock('@/composables/useAntigravityOAuth', () => ({
  useAntigravityOAuth: () => ({
    authUrl: ref(''),
    sessionId: ref(''),
    state: ref(''),
    loading: ref(false),
    error: ref(''),
    resetState: vi.fn()
  })
}))

vi.mock('@/composables/useOpenAIOAuth', () => ({
  useOpenAIOAuth: () => ({
    authUrl: computed(() => 'https://auth.openai.test'),
    sessionId: ref('session-1'),
    oauthState: ref('state-1'),
    loading: ref(false),
    error: ref(''),
    resetState: vi.fn(),
    generateAuthUrl: vi.fn(),
    exchangeAuthCode: exchangeAuthCodeMock,
    validateRefreshToken: vi.fn(),
    buildCredentials: buildCredentialsMock,
    buildExtraInfo: buildExtraInfoMock
  })
}))

import ReAuthAccountModal from '../ReAuthAccountModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: { type: Boolean, default: false } },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const OAuthAuthorizationFlowStub = defineComponent({
  name: 'OAuthAuthorizationFlow',
  setup(_, { expose }) {
    expose({
      authCode: 'code-1',
      oauthState: 'state-1',
      projectId: '',
      sessionKey: '',
      inputMethod: 'manual',
      reset: vi.fn()
    })
    return {}
  },
  template: '<div data-testid="oauth-flow-stub" />'
})

function buildAccount() {
  return {
    id: 99,
    name: 'OpenAI OAuth',
    platform: 'openai',
    type: 'oauth',
    credentials: {
      access_token: 'old-at',
      refresh_token: 'old-rt',
      model_mapping: { 'gpt-5': 'gpt-5' }
    },
    extra: {
      passthrough_fields_enabled: true,
      passthrough_field_rules: [
        { target: 'header', mode: 'forward', key: 'x-trace-id' }
      ],
      openai_passthrough: true,
      openai_oauth_responses_websockets_v2_mode: 'full',
      codex_cli_only: true
    },
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    rate_multiplier: 1,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: false,
    created_at: '',
    updated_at: ''
  }
}

describe('admin ReAuthAccountModal', () => {
  beforeEach(() => {
    updateAccountMock.mockReset()
    clearErrorMock.mockReset()
    showSuccessMock.mockReset()
    showErrorMock.mockReset()
    exchangeAuthCodeMock.mockReset()
    buildCredentialsMock.mockReset()
    buildExtraInfoMock.mockReset()

    exchangeAuthCodeMock.mockResolvedValue({ access_token: 'new-at', refresh_token: 'new-rt' })
    buildCredentialsMock.mockReturnValue({
      access_token: 'new-at',
      refresh_token: 'new-rt',
      expires_at: 1730000000
    })
    buildExtraInfoMock.mockReturnValue({
      email: 'new@example.com',
      name: 'New Name',
      privacy_mode: 'strict'
    })
    updateAccountMock.mockResolvedValue(buildAccount())
    clearErrorMock.mockResolvedValue(buildAccount())
  })

  it('merges existing account config into the OpenAI reauth update payload', async () => {
    const wrapper = mount(ReAuthAccountModal, {
      props: {
        show: true,
        account: buildAccount()
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          OAuthAuthorizationFlow: OAuthAuthorizationFlowStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    await wrapper.get('button.btn.btn-primary').trigger('click')
    await flushPromises()

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock).toHaveBeenCalledWith(99, {
      type: 'oauth',
      credentials: {
        access_token: 'new-at',
        refresh_token: 'new-rt',
        expires_at: 1730000000,
        model_mapping: { 'gpt-5': 'gpt-5' }
      },
      extra: {
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'header', mode: 'forward', key: 'x-trace-id' }
        ],
        openai_passthrough: true,
        openai_oauth_responses_websockets_v2_mode: 'full',
        codex_cli_only: true,
        email: 'new@example.com',
        name: 'New Name',
        privacy_mode: 'strict'
      }
    })
    expect(clearErrorMock).toHaveBeenCalledWith(99)
  })
})
