import { computed, defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const {
  createAccountMock,
  checkMixedChannelRiskMock,
  getSettingsMock,
  getWebSearchEmulationConfigMock,
  showErrorMock,
  showInfoMock
} = vi.hoisted(() => ({
  createAccountMock: vi.fn(),
  checkMixedChannelRiskMock: vi.fn(),
  getSettingsMock: vi.fn(),
  getWebSearchEmulationConfigMock: vi.fn(),
  showErrorMock: vi.fn(),
  showInfoMock: vi.fn()
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: vi.fn(),
    showInfo: showInfoMock,
    showWarning: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isSimpleMode: true
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      create: createAccountMock,
      checkMixedChannelRisk: checkMixedChannelRiskMock
    },
    settings: {
      getSettings: getSettingsMock,
      getWebSearchEmulationConfig: getWebSearchEmulationConfigMock
    }
  }
}))

vi.mock('@/composables/useModelWhitelist', () => ({
  claudeModels: [],
  getPresetMappingsByPlatform: vi.fn(() => []),
  getModelsByPlatform: vi.fn(() => []),
  commonErrorCodes: [],
  buildModelMappingObject: vi.fn(() => undefined),
  fetchAntigravityDefaultMappings: vi.fn(async () => []),
  isValidWildcardPattern: vi.fn(() => true)
}))

function createOAuthMock() {
  return {
    authUrl: ref(''),
    sessionId: ref(''),
    loading: ref(false),
    error: ref(''),
    oauthState: ref(''),
    resetState: vi.fn(),
    generateAuthUrl: vi.fn(),
    exchangeAuthCode: vi.fn(),
    validateRefreshToken: vi.fn(),
    validateSessionToken: vi.fn(),
    buildCredentials: vi.fn(() => ({})),
    buildExtraInfo: vi.fn(() => ({}))
  }
}

vi.mock('@/composables/useAccountOAuth', () => ({
  useAccountOAuth: () => createOAuthMock()
}))

vi.mock('@/composables/useOpenAIOAuth', () => ({
  useOpenAIOAuth: () => createOAuthMock()
}))

vi.mock('@/composables/useGeminiOAuth', () => ({
  useGeminiOAuth: () => ({
    ...createOAuthMock(),
    getCapabilities: vi.fn(async () => ({ ai_studio_oauth_enabled: false }))
  })
}))

vi.mock('@/composables/useAntigravityOAuth', () => ({
  useAntigravityOAuth: () => createOAuthMock()
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

import CreateAccountModal from '../CreateAccountModal.vue'
import PassthroughFieldRulesEditor from '../PassthroughFieldRulesEditor.vue'
import { getDefaultBaseUrl } from '../passthroughFieldSupport'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const ConfirmDialogStub = defineComponent({
  name: 'ConfirmDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const ModelWhitelistSelectorStub = defineComponent({
  name: 'ModelWhitelistSelector',
  props: {
    modelValue: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: '<div />'
})

const ProxySelectorStub = defineComponent({
  name: 'ProxySelector',
  props: {
    modelValue: {
      type: Number,
      default: null
    }
  },
  emits: ['update:modelValue'],
  template: '<div />'
})

const GroupSelectorStub = defineComponent({
  name: 'GroupSelector',
  props: {
    modelValue: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: '<div />'
})

const QuotaLimitCardStub = defineComponent({
  name: 'QuotaLimitCard',
  props: {
    totalLimit: { default: null },
    dailyLimit: { default: null },
    weeklyLimit: { default: null },
    dailyResetMode: { default: null },
    dailyResetHour: { default: null },
    weeklyResetMode: { default: null },
    weeklyResetDay: { default: null },
    weeklyResetHour: { default: null },
    resetTimezone: { default: null }
  },
  emits: [
    'update:totalLimit',
    'update:dailyLimit',
    'update:weeklyLimit',
    'update:dailyResetMode',
    'update:dailyResetHour',
    'update:weeklyResetMode',
    'update:weeklyResetDay',
    'update:weeklyResetHour',
    'update:resetTimezone',
    'update:quotaNotifyDailyEnabled',
    'update:quotaNotifyDailyThreshold',
    'update:quotaNotifyDailyThresholdType',
    'update:quotaNotifyWeeklyEnabled',
    'update:quotaNotifyWeeklyThreshold',
    'update:quotaNotifyWeeklyThresholdType',
    'update:quotaNotifyTotalEnabled',
    'update:quotaNotifyTotalThreshold',
    'update:quotaNotifyTotalThresholdType'
  ],
  template: '<div />'
})

function findWebSearchSelect(wrapper: ReturnType<typeof mountModal>) {
  const select = wrapper.findAll('select').find((candidate) => {
    const html = candidate.html()
    return html.includes('value="default"') && html.includes('value="enabled"') && html.includes('value="disabled"')
  })

  if (!select) {
    throw new Error('web search select not found')
  }

  return select
}

const SelectStub = defineComponent({
  name: 'Select',
  props: {
    modelValue: {
      type: [String, Number, Boolean, Object, Array, null],
      default: undefined
    },
    options: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    const normalizedValue = computed(() => String(props.modelValue ?? ''))
    const handleChange = (event: Event) => {
      emit('update:modelValue', (event.target as HTMLSelectElement).value)
    }
    return { normalizedValue, handleChange }
  },
  template: `
    <select
      :value="normalizedValue"
      @change="handleChange"
    >
      <option
        v-for="option in options"
        :key="String(option.value)"
        :value="String(option.value)"
      >
        {{ option.label }}
      </option>
    </select>
  `
})

function mountModal() {
  return mount(CreateAccountModal, {
    props: {
      show: true,
      proxies: [],
      groups: []
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        Select: SelectStub,
        Icon: true,
        ProxySelector: ProxySelectorStub,
        GroupSelector: GroupSelectorStub,
        ModelWhitelistSelector: ModelWhitelistSelectorStub,
        QuotaLimitCard: QuotaLimitCardStub,
        OAuthAuthorizationFlow: true
      }
    }
  })
}

async function switchToOpenAIApiKey(wrapper: ReturnType<typeof mountModal>) {
  await wrapper.get('[data-testid="platform-openai"]').trigger('click')
  await wrapper.get('[data-testid="account-type-apikey"]').trigger('click')
  await wrapper.get('[data-testid="create-account-apikey-input"]').setValue('sk-test')
}

async function switchToAntigravityUpstream(wrapper: ReturnType<typeof mountModal>) {
  await wrapper.get('[data-testid="platform-antigravity"]').trigger('click')
  await wrapper.get('[data-testid="account-type-antigravity-upstream"]').trigger('click')
  await wrapper.get('[data-tour="account-form-name"]').setValue('Antigravity upstream')
  await wrapper.get('input[placeholder="https://cloudcode-pa.googleapis.com"]').setValue('https://cloudcode-pa.googleapis.com')
  await wrapper.get('input[placeholder="sk-..."]').setValue('sk-antigravity-test')
}

async function setPassthroughState(
  wrapper: ReturnType<typeof mountModal>,
  payload: {
    enabled?: boolean
    rules?: Array<{ id: string; target: 'header' | 'body'; mode: 'forward' | 'inject' | 'map'; key: string; source_key?: string; value: string }>
  }
) {
  const editor = wrapper.getComponent(PassthroughFieldRulesEditor)

  if (payload.enabled !== undefined) {
    editor.vm.$emit('update:enabled', payload.enabled)
  }

  if (payload.rules !== undefined) {
    editor.vm.$emit('update:rules', payload.rules)
  }

  await flushPromises()
}

describe('CreateAccountModal', () => {
  beforeEach(() => {
    createAccountMock.mockReset()
    createAccountMock.mockResolvedValue({ id: 1 })
    checkMixedChannelRiskMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    getSettingsMock.mockReset()
    getSettingsMock.mockResolvedValue({ account_quota_notify_enabled: true })
    getWebSearchEmulationConfigMock.mockReset()
    getWebSearchEmulationConfigMock.mockResolvedValue({ enabled: true, providers: ['brave'] })
    showErrorMock.mockReset()
    showInfoMock.mockReset()
  })

  it('merges passthrough fields with anthropic web search and quota notify settings on create', async () => {
    const wrapper = mountModal()

    await wrapper.get('[data-tour="account-form-name"]').setValue('Anthropic API Key')
    await wrapper.get('[data-testid="platform-anthropic"]').trigger('click')
    await wrapper.get('[data-testid="account-type-apikey"]').trigger('click')
    await wrapper.get('[data-testid="create-account-apikey-input"]').setValue('sk-ant-test')
    await flushPromises()

    await setPassthroughState(wrapper, {
      enabled: true,
      rules: [
        { id: 'rule-1', target: 'header', mode: 'inject', key: 'X-Env', value: 'prod' }
      ]
    })
    await findWebSearchSelect(wrapper).setValue('disabled')

    const quotaCard = wrapper.getComponent({ name: 'QuotaLimitCard' })
    quotaCard.vm.$emit('update:totalLimit', 50)
    quotaCard.vm.$emit('update:quotaNotifyTotalEnabled', true)
    quotaCard.vm.$emit('update:quotaNotifyTotalThreshold', 80)
    quotaCard.vm.$emit('update:quotaNotifyTotalThresholdType', 'percentage')
    await flushPromises()

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledWith(expect.objectContaining({
      extra: expect.objectContaining({
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'header', mode: 'inject', key: 'X-Env', value: 'prod' }
        ],
        web_search_emulation: 'disabled',
        quota_limit: 50,
        quota_notify_total_enabled: true,
        quota_notify_total_threshold: 80,
        quota_notify_total_threshold_type: 'percentage'
      })
    }))
  })

  it('submits passthrough field rules for API key accounts only', async () => {
    const wrapper = mountModal()

    await switchToOpenAIApiKey(wrapper)
    await setPassthroughState(wrapper, {
      enabled: true,
      rules: [
        { id: 'rule-1', target: 'header', mode: 'inject', key: 'X-Env', value: 'prod' },
        { id: 'rule-2', target: 'body', mode: 'forward', key: 'metadata.user_id', value: '' }
      ]
    })

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledWith(expect.objectContaining({
      extra: expect.objectContaining({
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'header', mode: 'inject', key: 'X-Env', value: 'prod' },
          { target: 'body', mode: 'forward', key: 'metadata.user_id' }
        ]
      })
    }))
  })

  it('uses helper default base_url for initial anthropic apikey flow when left empty', async () => {
    const wrapper = mountModal()

    await wrapper.get('[data-tour="account-form-name"]').setValue('Anthropic API Key')
    await wrapper.get('[data-testid="platform-anthropic"]').trigger('click')
    await wrapper.get('[data-testid="account-type-apikey"]').trigger('click')
    await wrapper.get('[data-testid="create-account-apikey-input"]').setValue('sk-ant-test')
    await wrapper.get(`input[placeholder="${getDefaultBaseUrl('anthropic')}"]`).setValue('')

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledWith(expect.objectContaining({
      platform: 'anthropic',
      type: 'apikey',
      credentials: expect.objectContaining({
        base_url: getDefaultBaseUrl('anthropic')
      })
    }))
  })

  it('initial anthropic base_url state is sourced from helper', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/components/account/CreateAccountModal.vue'), 'utf8')

    expect(source).toContain("const apiKeyBaseUrl = ref(getDefaultBaseUrl('anthropic'))")
  })

  it('submits map passthrough rules with source_key in create payload', async () => {
    const wrapper = mountModal()

    await switchToOpenAIApiKey(wrapper)
    await setPassthroughState(wrapper, {
      enabled: true,
      rules: [
        {
          id: 'rule-1',
          target: 'body',
          mode: 'map',
          key: 'metadata.target',
          source_key: 'metadata.source',
          value: ''
        }
      ]
    })

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledWith(expect.objectContaining({
      extra: expect.objectContaining({
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          {
            target: 'body',
            mode: 'map',
            key: 'metadata.target',
            source_key: 'metadata.source'
          }
        ]
      })
    }))
  })

  it('clears passthrough field payload after switching away from apikey flow', async () => {
    const wrapper = mountModal()

    await wrapper.get('[data-tour="account-form-name"]').setValue('Bedrock account')
    await switchToOpenAIApiKey(wrapper)
    await setPassthroughState(wrapper, {
      enabled: true,
      rules: [
        { id: 'rule-1', target: 'header', mode: 'forward', key: 'X-Test', value: '' }
      ]
    })
    await wrapper.get('[data-testid="platform-anthropic"]').trigger('click')
    await wrapper.get('[data-testid="account-type-bedrock"]').trigger('click')
    await wrapper.get('[data-testid="bedrock-access-key-id-input"]').setValue('AKIA_TEST')
    await wrapper.get('[data-testid="bedrock-secret-access-key-input"]').setValue('secret')

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledWith(expect.objectContaining({
      extra: expect.not.objectContaining({
        passthrough_fields_enabled: expect.anything(),
        passthrough_field_rules: expect.anything()
      })
    }))
  })

  it('does not add passthrough payload when disabled and rules are empty', async () => {
    const wrapper = mountModal()

    await switchToOpenAIApiKey(wrapper)
    await setPassthroughState(wrapper, {
      enabled: false,
      rules: []
    })

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledWith(expect.objectContaining({
      extra: expect.not.objectContaining({
        passthrough_fields_enabled: expect.anything(),
        passthrough_field_rules: expect.anything()
      })
    }))
  })

  it('preserves existing extra fields when passthrough rules are submitted', async () => {
    const wrapper = mountModal()

    await switchToOpenAIApiKey(wrapper)
    await setPassthroughState(wrapper, {
      enabled: false,
      rules: [
        { id: 'rule-1', target: 'header', mode: 'forward', key: 'X-Test', value: '' }
      ]
    })

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledWith(expect.objectContaining({
      extra: expect.objectContaining({
        openai_apikey_responses_websockets_v2_mode: 'off',
        openai_apikey_responses_websockets_v2_enabled: false,
        passthrough_fields_enabled: false,
        passthrough_field_rules: [
          { target: 'header', mode: 'forward', key: 'X-Test' }
        ]
      })
    }))
  })

  it('blocks create submit with hidden invalid rules and shows only toggle error until reopened', async () => {
    const wrapper = mountModal()

    await switchToOpenAIApiKey(wrapper)
    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(true)
    await wrapper.get('[data-testid="passthrough-rule-mode-0"]').setValue('map')
    await wrapper.get('[data-testid="passthrough-rule-key-0"]').setValue('metadata.target')
    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(false)

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).not.toHaveBeenCalled()
    expect(showErrorMock).not.toHaveBeenCalledWith('admin.accounts.pleaseEnterApiKey')
    expect(wrapper.find('[data-testid="passthrough-rules-section"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('admin.accounts.passthroughFields.hiddenRulesError')
    expect(wrapper.text()).not.toContain('admin.accounts.passthroughFields.errors.sourceKeyRequired')

    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(true)

    expect(wrapper.find('[data-testid="passthrough-rules-section"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('admin.accounts.passthroughFields.errors.sourceKeyRequired')

    await wrapper.get('[data-testid="passthrough-rule-source-key-0"]').setValue('metadata.source')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledTimes(1)
  })

  it('renders passthrough field section for oauth account types', async () => {
    const wrapper = mountModal()

    await wrapper.get('[data-testid="platform-openai"]').trigger('click')
    await wrapper.get('[data-testid="account-type-oauth"]').trigger('click')

    expect(wrapper.find('[data-testid="passthrough-fields-section"]').exists()).toBe(true)
  })

  it('renders passthrough field section for antigravity upstream create flow and submits rules', async () => {
    const wrapper = mountModal()

    await switchToAntigravityUpstream(wrapper)

    expect(wrapper.find('[data-testid="passthrough-fields-section"]').exists()).toBe(true)

    await setPassthroughState(wrapper, {
      enabled: true,
      rules: [
        { id: 'rule-1', target: 'header', mode: 'forward', key: 'X-Antigravity', value: '' }
      ]
    })

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledWith(expect.objectContaining({
      platform: 'antigravity',
      type: 'apikey',
      extra: expect.objectContaining({
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'header', mode: 'forward', key: 'X-Antigravity' }
        ]
      })
    }))
  })

  it('uses helper default base_url for create apikey fallback and antigravity upstream input', async () => {
    const wrapper = mountModal()

    await switchToOpenAIApiKey(wrapper)

    const openaiBaseUrlInput = wrapper.get('input[placeholder="https://api.openai.com"]')
    await openaiBaseUrlInput.setValue('')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledWith(expect.objectContaining({
      credentials: expect.objectContaining({
        base_url: getDefaultBaseUrl('openai')
      })
    }))

    await wrapper.get('[data-testid="platform-antigravity"]').trigger('click')
    await wrapper.get('[data-testid="account-type-antigravity-upstream"]').trigger('click')

    const upstreamBaseUrlInput = wrapper.get(`input[placeholder="${getDefaultBaseUrl('antigravity')}"]`)
    expect((upstreamBaseUrlInput.element as HTMLInputElement).value).toBe(getDefaultBaseUrl('antigravity'))
  })

  it('submits antigravity upstream with helper default base_url when cleared', async () => {
    const wrapper = mountModal()

    await switchToAntigravityUpstream(wrapper)
    await wrapper.get(`input[placeholder="${getDefaultBaseUrl('antigravity')}"]`).setValue('')

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledWith(expect.objectContaining({
      platform: 'antigravity',
      type: 'apikey',
      credentials: expect.objectContaining({
        base_url: getDefaultBaseUrl('antigravity'),
        api_key: 'sk-antigravity-test'
      })
    }))
  })

  it('clears hidden upstream credentials after switching away and back', async () => {
    const wrapper = mountModal()

    await switchToAntigravityUpstream(wrapper)
    await wrapper.get('input[placeholder="sk-..."]').setValue('sk-hidden-upstream')
    await wrapper.get('[data-testid="platform-openai"]').trigger('click')
    await wrapper.get('[data-testid="platform-antigravity"]').trigger('click')
    await wrapper.get('[data-testid="account-type-antigravity-upstream"]').trigger('click')

    expect((wrapper.get('input[placeholder="sk-..."]').element as HTMLInputElement).value).toBe('')
  })

  it('keeps passthrough support after switching from supported apikey to antigravity upstream', async () => {
    const wrapper = mountModal()

    await switchToOpenAIApiKey(wrapper)
    await setPassthroughState(wrapper, {
      enabled: true,
      rules: [
        { id: 'rule-1', target: 'header', mode: 'forward', key: 'X-Test', value: '' }
      ]
    })

    await switchToAntigravityUpstream(wrapper)

    expect(wrapper.find('[data-testid="passthrough-fields-section"]').exists()).toBe(true)
    expect(showInfoMock).not.toHaveBeenCalledWith(expect.stringContaining('移除透传字段规则配置'))
  })
})
