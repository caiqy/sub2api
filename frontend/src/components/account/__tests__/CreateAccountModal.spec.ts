import { computed, defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const {
  createAccountMock,
  probeUpstreamBillingMock,
  checkMixedChannelRiskMock,
  getSettingsMock,
  getWebSearchEmulationConfigMock,
  showErrorMock,
  showInfoMock,
  syncUpstreamModelsMock,
  showWarningMock,
  importCodexSessionMock,
  createOpenAICodexPATMock,
  authIsSimpleMode
} = vi.hoisted(() => ({
  createAccountMock: vi.fn(),
  probeUpstreamBillingMock: vi.fn(),
  checkMixedChannelRiskMock: vi.fn(),
  getSettingsMock: vi.fn(),
  getWebSearchEmulationConfigMock: vi.fn(),
  showErrorMock: vi.fn(),
  syncUpstreamModelsMock: vi.fn(),
  showWarningMock: vi.fn(),
  importCodexSessionMock: vi.fn(),
  createOpenAICodexPATMock: vi.fn(),
  showInfoMock: vi.fn(),
  authIsSimpleMode: { value: true },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: vi.fn(),
    showInfo: showInfoMock,
    showWarning: showWarningMock,
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    get isSimpleMode() {
      return authIsSimpleMode.value
    }
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      create: createAccountMock,
      probeUpstreamBilling: probeUpstreamBillingMock,
      checkMixedChannelRisk: checkMixedChannelRiskMock,
      syncUpstreamModels: syncUpstreamModelsMock,
      importCodexSession: importCodexSessionMock,
      createOpenAICodexPAT: createOpenAICodexPATMock
    },
    settings: {
      getSettings: getSettingsMock,
      getWebSearchEmulationConfig: getWebSearchEmulationConfigMock
    },
    tlsFingerprintProfiles: {
      list: vi.fn().mockResolvedValue([])
    }
  }
}))

vi.mock('@/composables/useModelWhitelist', () => ({
  claudeModels: [],
  getPresetMappingsByPlatform: vi.fn(() => []),
  getModelsByPlatform: vi.fn(() => []),
  commonErrorCodes: [],
  buildModelMappingObject: vi.fn(
    (
      mode: 'whitelist' | 'mapping',
      models: string[],
      mappings: Array<{ from: string; to: string }>
    ) => {
      if (mode === 'mapping') {
        return Object.fromEntries(
          mappings
            .filter((mapping) => mapping.from && mapping.to)
            .map((mapping) => [mapping.from, mapping.to])
        )
      }
      return models.length ? Object.fromEntries(models.map((model) => [model, model])) : undefined
    }
  ),
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

const OAuthAuthorizationFlowStub = defineComponent({
  name: 'OAuthAuthorizationFlow',
  props: {
    showManualOption: Boolean,
    showCodexSessionImportOption: Boolean,
    showAgentIdentityOption: Boolean,
    showCodexPatOption: Boolean,
    initialInputMethod: String
  },
  data: () => ({ inputMethod: 'manual' }),
  emits: ['import-codex-session', 'import-codex-pat'],
  template: `
    <div>
      <button data-testid="import-codex-session" @click="$emit('import-codex-session', 'session-json')">session</button>
      <button data-testid="import-codex-pat" @click="$emit('import-codex-pat', 'pat-token')">pat</button>
    </div>
  `
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
  name: 'SelectStub',
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

const GroupSelectorStub = defineComponent({
  name: 'GroupSelector',
  props: {
    modelValue: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue'],
  template: `
    <button
      type="button"
      data-testid="select-pricing-groups"
      @click="$emit('update:modelValue', [1, 2])"
    >
      groups
    </button>
  `,
})

const ModelWhitelistSelectorStub = defineComponent({
  name: 'ModelWhitelistSelector',
  props: {
    modelValue: {
      type: Array,
      default: () => [],
    },
    platform: String,
    syncCredentials: Object,
  },
  emits: ['update:modelValue', 'upstream-synced'],
  template: `<button
    type="button"
    data-testid="model-whitelist-selector"
    @click="$emit('update:modelValue', ['public-glm']); $emit('upstream-synced')"
  >models</button>`,
})

function mountModal(groups: any[] = []) {
  return mount(CreateAccountModal, {
    props: { show: true, proxies: [], groups },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        Select: SelectStub,
        Icon: true,
        PlatformIcon: true,
        ProxySelector: ProxySelectorStub,
        ProxyAdBanner: true,
        GroupSelector: GroupSelectorStub,
        ModelWhitelistSelector: ModelWhitelistSelectorStub,
        QuotaLimitCard: QuotaLimitCardStub,
        OAuthAuthorizationFlow: OAuthAuthorizationFlowStub
      },
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

async function selectButtonByText(wrapper: ReturnType<typeof mountModal>, text: string) {
  const button = wrapper.findAll('button').find((candidate) => candidate.text().includes(text))
  expect(button).toBeDefined()
  await button?.trigger('click')
}

async function submitApiKeyAccount(
  platform: 'openai' | 'anthropic',
  enableLongContextBilling = false,
  disableUpstreamBillingProbe = false
) {
  const wrapper = mountModal()
  await selectButtonByText(wrapper, platform === 'openai' ? 'OpenAI' : 'admin.accounts.claudeConsole')
  if (platform === 'openai') {
    await selectButtonByText(wrapper, 'API Key')
  }
  await wrapper.get('form#create-account-form input[type="text"]').setValue(`${platform} account`)
  await wrapper.get('form#create-account-form input[type="password"]').setValue('test-api-key')
  if (enableLongContextBilling) {
    await wrapper.get('[data-testid="openai-long-context-billing-toggle"]').trigger('click')
  }
  if (disableUpstreamBillingProbe) {
    await wrapper.get('[data-testid="upstream-billing-auto-probe"]').trigger('click')
  }
  await wrapper.get('form#create-account-form').trigger('submit.prevent')
  await flushPromises()
  return wrapper
}

async function openCodexImportStep(toggleClicks = 0) {
  const wrapper = mountModal()
  await selectButtonByText(wrapper, 'OpenAI')
  for (let click = 0; click < toggleClicks; click += 1) {
    await wrapper.get('[data-testid="openai-long-context-billing-toggle"]').trigger('click')
  }
  await wrapper.get('form#create-account-form input[type="text"]').setValue('Codex import')
  await wrapper.get('form#create-account-form').trigger('submit.prevent')
  return wrapper
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
    authIsSimpleMode.value = true
    createAccountMock.mockReset().mockResolvedValue({ id: 42, platform: 'openai', type: 'apikey' })
    probeUpstreamBillingMock.mockReset().mockResolvedValue({})
    syncUpstreamModelsMock.mockReset().mockResolvedValue({ models: [], metadata: {} })
    showWarningMock.mockReset()
    importCodexSessionMock.mockReset().mockResolvedValue({
      created: 1,
      updated: 0,
      skipped: 0,
      failed: 0,
      errors: [],
      warnings: []
    })
    createOpenAICodexPATMock.mockReset().mockResolvedValue({})
    checkMixedChannelRiskMock.mockReset().mockResolvedValue({ has_risk: false })
    getSettingsMock.mockReset().mockResolvedValue({ account_quota_notify_enabled: true })
    getWebSearchEmulationConfigMock.mockReset().mockResolvedValue({ enabled: true, providers: ['brave'] })
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

  it('hides only the redundant account toggle when every selected group enables tier pricing', async () => {
    authIsSimpleMode.value = false
    const wrapper = mountModal([
      { id: 1, long_context_pricing_enabled: true },
      { id: 2, long_context_pricing_enabled: true },
    ])

    await selectButtonByText(wrapper, 'OpenAI')
    await wrapper.get('[data-testid="select-pricing-groups"]').trigger('click')

    expect(wrapper.find('[data-testid="openai-long-context-billing-toggle"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="create-openai-ws-mode"]').exists()).toBe(true)
  })

  it('keeps the account toggle when any selected group disables tier pricing', async () => {
    authIsSimpleMode.value = false
    const wrapper = mountModal([
      { id: 1, long_context_pricing_enabled: true },
      { id: 2, long_context_pricing_enabled: false },
    ])

    await selectButtonByText(wrapper, 'OpenAI')
    await wrapper.get('[data-testid="select-pricing-groups"]').trigger('click')

    expect(wrapper.find('[data-testid="openai-long-context-billing-toggle"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="create-openai-ws-mode"]').exists()).toBe(true)
  })

  it('sends false explicitly for normal OpenAI account creation by default', async () => {
    await submitApiKeyAccount('openai')

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]?.extra?.openai_long_context_billing_enabled).toBe(false)
  })

  it('persists upstream model metadata after creating an account from preview', async () => {
    const wrapper = mountModal()
    await selectButtonByText(wrapper, 'OpenAI')
    await selectButtonByText(wrapper, 'API Key')
    await wrapper.get('form#create-account-form input[type="text"]').setValue('OpenCode account')
    await wrapper.get('form#create-account-form input[type="password"]').setValue('test-api-key')
    await wrapper.get('[data-testid="model-whitelist-selector"]').trigger('click')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledOnce()
    expect(syncUpstreamModelsMock).toHaveBeenCalledWith(42)
  })

  it('includes the current concrete model mapping in preview credentials', async () => {
    const wrapper = mountModal()
    await selectButtonByText(wrapper, 'OpenAI')
    await selectButtonByText(wrapper, 'API Key')
    await wrapper.get('form#create-account-form input[type="password"]').setValue('test-api-key')
    await wrapper.get('[data-testid="model-whitelist-selector"]').trigger('click')
    await flushPromises()

    expect(wrapper.getComponent(ModelWhitelistSelectorStub).props('syncCredentials')).toMatchObject({
      model_mapping: { 'public-glm': 'public-glm' }
    })
  })

  it('runs formal capability sync after creating an account with explicit mappings', async () => {
    const wrapper = mountModal()
    await selectButtonByText(wrapper, 'OpenAI')
    await selectButtonByText(wrapper, 'API Key')
    await wrapper.get('form#create-account-form input[type="text"]').setValue('Mapped account')
    await wrapper.get('form#create-account-form input[type="password"]').setValue('test-api-key')
    await selectButtonByText(wrapper, 'admin.accounts.modelMapping')
    await selectButtonByText(wrapper, 'admin.accounts.addMapping')
    await wrapper.get('input[placeholder="admin.accounts.requestModel"]').setValue('public-glm')
    await wrapper.get('input[placeholder="admin.accounts.actualModel"]').setValue('glm-5.3')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock.mock.calls[0]?.[0]?.credentials?.model_mapping).toEqual({
      'public-glm': 'glm-5.3'
    })
    expect(syncUpstreamModelsMock).toHaveBeenCalledWith(42)
  })

  it('warns when post-create capability metadata remains incomplete', async () => {
    syncUpstreamModelsMock.mockResolvedValue({
      models: ['x-preview-f-free'],
      warnings: [{ code: 'upstream_model_metadata_incomplete', message: 'metadata incomplete' }],
    })
    const wrapper = mountModal()
    await selectButtonByText(wrapper, 'OpenAI')
    await selectButtonByText(wrapper, 'API Key')
    await wrapper.get('form#create-account-form input[type="text"]').setValue('OpenCode account')
    await wrapper.get('form#create-account-form input[type="password"]').setValue('test-api-key')
    await wrapper.get('[data-testid="model-whitelist-selector"]').trigger('click')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(showWarningMock).toHaveBeenCalledWith(
      'admin.accounts.syncUpstreamModelsMetadataIncomplete'
    )
  })

  // namespace 摊平是仅 OAuth 的兼容开关：API Key 走 chat completions 回退桥时由桥自行摊平
  it('shows the Codex namespace flatten toggle only for OpenAI OAuth accounts', async () => {
    const wrapper = mountModal()
    await selectButtonByText(wrapper, 'OpenAI')

    expect(wrapper.find('[data-testid="create-openai-flatten-namespaces-toggle"]').exists()).toBe(
      true
    )

    await selectButtonByText(wrapper, 'API Key')
    expect(wrapper.find('[data-testid="create-openai-flatten-namespaces-toggle"]').exists()).toBe(
      false
    )
  })

  it('sends an explicit disabled state when the create toggle is turned off', async () => {
    await submitApiKeyAccount('openai', false, true)

    expect(createAccountMock.mock.calls[0]?.[0]?.upstream_billing_probe_enabled).toBe(false)
    expect(probeUpstreamBillingMock).not.toHaveBeenCalled()
  })

  it('submits adaptive Kimi protocol endpoints', async () => {
    const wrapper = mountModal()
    await selectButtonByText(wrapper, 'Kimi')
    await wrapper.get('form#create-account-form input[type="text"]').setValue('Kimi adaptive')
    await wrapper.get('form#create-account-form input[type="password"]').setValue('sk-kimi')

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]?.credentials).toMatchObject({
      account_mode: 'payg',
      api_protocol: 'adaptive',
      base_url: 'https://api.moonshot.cn/v1',
      api_base_urls: {
        chat_completions: 'https://api.moonshot.cn/v1',
        anthropic: 'https://api.moonshot.cn/anthropic'
      }
    })
  })

  it('uses the edited adaptive Chat endpoint when previewing upstream models', async () => {
    const wrapper = mountModal()
    await selectButtonByText(wrapper, 'Kimi')
    await wrapper
      .get('[data-testid="cn-adaptive-base-url-chat_completions"]')
      .setValue('https://relay.example.com/v1')
    await wrapper.get('form#create-account-form input[type="password"]').setValue('sk-relay')

    expect(wrapper.getComponent(ModelWhitelistSelectorStub).props('syncCredentials')).toMatchObject({
      platform: 'kimi',
      type: 'apikey',
      base_url: 'https://relay.example.com/v1',
      api_key: 'sk-relay'
    })
  })

  it('exposes Agent Identity in the OpenAI authorization methods', async () => {
    const wrapper = mountModal()

    await selectButtonByText(wrapper, 'OpenAI')
    await wrapper.get('form#create-account-form input[type="text"]').setValue('OpenAI account')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')

    const flow = wrapper.getComponent(OAuthAuthorizationFlowStub)
    expect(flow.props('showManualOption')).toBe(true)
    expect(flow.props('showCodexSessionImportOption')).toBe(true)
    expect(flow.props('showAgentIdentityOption')).toBe(true)
    expect(flow.props('showCodexPatOption')).toBe(true)
    expect(flow.props('initialInputMethod')).toBe('manual')
  })

  it.each([
    ['camelCase', { authMode: 'agentIdentity', agentIdentity: { agentRuntimeId: 'runtime' } }],
    ['nested identity without auth_mode', { agent_identity: { agent_runtime_id: 'runtime' } }]
  ])('accepts backend-compatible %s Agent Identity imports', async (_name, content) => {
    const wrapper = await openCodexImportStep()
    const flow = wrapper.getComponent(OAuthAuthorizationFlowStub)
    flow.vm.inputMethod = 'agent_identity'

    flow.vm.$emit('import-codex-session', JSON.stringify(content))
    await flushPromises()

    expect(importCodexSessionMock).toHaveBeenCalledTimes(1)
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

  it('enables upstream billing probes for antigravity upstream accounts by default', async () => {
    const wrapper = mountModal()

    await switchToAntigravityUpstream(wrapper)
    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledWith(expect.objectContaining({
      upstream_billing_probe_enabled: true
    }))
    expect(probeUpstreamBillingMock).toHaveBeenCalledWith(42)
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
