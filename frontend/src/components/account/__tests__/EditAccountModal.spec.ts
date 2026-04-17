import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const {
  updateAccountMock,
  checkMixedChannelRiskMock,
  getSettingsMock,
  getWebSearchEmulationConfigMock,
  listTLSFingerprintProfilesMock,
  showErrorMock,
  showSuccessMock,
  showInfoMock
} = vi.hoisted(() => ({
  updateAccountMock: vi.fn(),
  checkMixedChannelRiskMock: vi.fn(),
  getSettingsMock: vi.fn(),
  getWebSearchEmulationConfigMock: vi.fn(),
  listTLSFingerprintProfilesMock: vi.fn(),
  showErrorMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showInfoMock: vi.fn()
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: showSuccessMock,
    showInfo: showInfoMock
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
      update: updateAccountMock,
      checkMixedChannelRisk: checkMixedChannelRiskMock
    },
    settings: {
      getSettings: getSettingsMock,
      getWebSearchEmulationConfig: getWebSearchEmulationConfigMock
    },
    tlsFingerprintProfiles: {
      list: listTLSFingerprintProfilesMock
    }
  }
}))

vi.mock('@/api/admin/accounts', () => ({
  getAntigravityDefaultModelMapping: vi.fn()
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

import EditAccountModal from '../EditAccountModal.vue'
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

const ModelWhitelistSelectorStub = defineComponent({
  name: 'ModelWhitelistSelector',
  props: {
    modelValue: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: `
    <div>
      <button
        type="button"
        data-testid="rewrite-to-snapshot"
        @click="$emit('update:modelValue', ['gpt-5.2-2025-12-11'])"
      >
        rewrite
      </button>
      <span data-testid="model-whitelist-value">
        {{ Array.isArray(modelValue) ? modelValue.join(',') : '' }}
      </span>
    </div>
  `
})

const QuotaLimitCardStub = defineComponent({
  name: 'QuotaLimitCard',
  emits: [
    'update:totalLimit',
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
  template: '<button type="button" data-testid="quota-limit-set" @click="$emit(\'update:totalLimit\', 99)">quota</button>'
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

function buildAccount(overrides: Record<string, any> = {}) {
  const { credentials: credentialOverrides, extra: extraOverrides, ...restOverrides } = overrides
  const baseAccount = {
    id: 1,
    name: 'OpenAI Key',
    notes: '',
    platform: 'openai',
    type: 'apikey',
    credentials: {
      api_key: 'sk-test',
      base_url: 'https://api.openai.com',
      model_mapping: {
        'gpt-5.2': 'gpt-5.2'
      }
    },
    extra: {},
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    rate_multiplier: 1,
    status: 'active',
    group_ids: [],
    expires_at: null,
    auto_pause_on_expired: false
  }

  return {
    ...baseAccount,
    ...restOverrides,
    credentials: {
      ...baseAccount.credentials,
      ...(credentialOverrides || {})
    },
    extra: {
      ...baseAccount.extra,
      ...(extraOverrides || {})
    }
  } as any
}

function mountModal(account = buildAccount()) {
  return mount(EditAccountModal, {
    props: {
      show: true,
      account,
      proxies: [],
      groups: []
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        Select: true,
        Icon: true,
        ProxySelector: true,
        GroupSelector: true,
        ModelWhitelistSelector: ModelWhitelistSelectorStub,
        QuotaLimitCard: QuotaLimitCardStub
      }
    }
  })
}

describe('EditAccountModal', () => {
  beforeEach(() => {
    updateAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    getSettingsMock.mockReset()
    getSettingsMock.mockResolvedValue({ account_quota_notify_enabled: true })
    getWebSearchEmulationConfigMock.mockReset()
    getWebSearchEmulationConfigMock.mockResolvedValue({ enabled: true, providers: ['brave'] })
    showErrorMock.mockReset()
    showSuccessMock.mockReset()
    showInfoMock.mockReset()
    listTLSFingerprintProfilesMock.mockReset()
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
    listTLSFingerprintProfilesMock.mockResolvedValue([{ id: 101, name: 'Chrome 136' }])
    updateAccountMock.mockResolvedValue(buildAccount())
  })

  it('keeps random TLS fingerprint profile selected and submits -1 for anthropic oauth accounts', async () => {
	  const wrapper = mountModal(buildAccount({
	    platform: 'anthropic',
	    type: 'oauth',
	    enable_tls_fingerprint: true,
	    tls_fingerprint_profile_id: -1,
	    extra: {
	      enable_tls_fingerprint: true,
	      tls_fingerprint_profile_id: -1
	    }
	  }))

	  await flushPromises()
	  expect(listTLSFingerprintProfilesMock).toHaveBeenCalledTimes(1)

	  await wrapper.get('form#edit-account-form').trigger('submit.prevent')
	  await flushPromises()

	  expect(updateAccountMock).toHaveBeenCalledTimes(1)
	  expect(updateAccountMock.mock.calls[0]?.[1]?.extra?.tls_fingerprint_profile_id).toBe(-1)
  })

  it('reopening the same account rehydrates the OpenAI whitelist from props', async () => {
    const account = buildAccount()
    updateAccountMock.mockResolvedValue(account)

    const wrapper = mountModal(account)

    expect(wrapper.get('[data-testid="model-whitelist-value"]').text()).toBe('gpt-5.2')

    await wrapper.get('[data-testid="rewrite-to-snapshot"]').trigger('click')
    expect(wrapper.get('[data-testid="model-whitelist-value"]').text()).toBe('gpt-5.2-2025-12-11')

    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })

    expect(wrapper.get('[data-testid="model-whitelist-value"]').text()).toBe('gpt-5.2')

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.model_mapping).toEqual({
      'gpt-5.2': 'gpt-5.2'
    })
  })

  it('rehydrates passthrough field rules from account.extra', async () => {
    const wrapper = mountModal(buildAccount({
      extra: {
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'header', mode: 'inject', key: 'X-Env', value: 'prod' }
        ]
      }
    }))

    expect((wrapper.get('[data-testid="passthrough-enabled-toggle"]').element as HTMLInputElement).checked).toBe(true)
    expect((wrapper.get('[data-testid="passthrough-rule-key-0"]').element as HTMLInputElement).value).toBe('X-Env')
    expect((wrapper.get('[data-testid="passthrough-rule-value-0"]').element as HTMLInputElement).value).toBe('prod')
  })

  it('rehydrates map passthrough rules with source and target fields from account.extra', async () => {
    const wrapper = mountModal(buildAccount({
      extra: {
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'body', mode: 'map', key: 'metadata.target', source_key: 'metadata.source' }
        ]
      }
    }))

    expect((wrapper.get('[data-testid="passthrough-enabled-toggle"]').element as HTMLInputElement).checked).toBe(true)
    expect((wrapper.get('[data-testid="passthrough-rule-target-0"]').element as HTMLSelectElement).value).toBe('body')
    expect((wrapper.get('[data-testid="passthrough-rule-mode-0"]').element as HTMLSelectElement).value).toBe('map')
    expect((wrapper.get('[data-testid="passthrough-rule-key-0"]').element as HTMLInputElement).value).toBe('metadata.target')
    expect((wrapper.get('[data-testid="passthrough-rule-source-key-0"]').element as HTMLInputElement).value).toBe('metadata.source')
  })

  it('rehydrates delete passthrough rules without downgrading mode to forward', async () => {
    const wrapper = mountModal(buildAccount({
      extra: {
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'body', mode: 'delete', key: 'metadata.internal' }
        ]
      }
    }))

    expect((wrapper.get('[data-testid="passthrough-enabled-toggle"]').element as HTMLInputElement).checked).toBe(true)
    expect((wrapper.get('[data-testid="passthrough-rule-target-0"]').element as HTMLSelectElement).value).toBe('body')
    expect((wrapper.get('[data-testid="passthrough-rule-mode-0"]').element as HTMLSelectElement).value).toBe('delete')
    expect((wrapper.get('[data-testid="passthrough-rule-key-0"]').element as HTMLInputElement).value).toBe('metadata.internal')
  })

  it('blocks submit when header keys differ only by case', async () => {
    const wrapper = mountModal(buildAccount())

    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(true)
    await wrapper.get('[data-testid="passthrough-rule-mode-0"]').setValue('inject')
    await wrapper.get('[data-testid="passthrough-rule-key-0"]').setValue('X-Test')
    await wrapper.get('[data-testid="passthrough-rule-value-0"]').setValue('one')
    await wrapper.get('[data-testid="passthrough-add-rule"]').trigger('click')
    await wrapper.get('[data-testid="passthrough-rule-mode-1"]').setValue('inject')
    await wrapper.get('[data-testid="passthrough-rule-key-1"]').setValue('x-test')
    await wrapper.get('[data-testid="passthrough-rule-value-1"]').setValue('two')

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).not.toHaveBeenCalled()
  })

  it('blocks submit with hidden invalid rules and shows only toggle error until reopened', async () => {
    const wrapper = mountModal(buildAccount())

    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(true)
    await wrapper.get('[data-testid="passthrough-rule-mode-0"]').setValue('map')
    await wrapper.get('[data-testid="passthrough-rule-key-0"]').setValue('metadata.target')
    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(false)

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="passthrough-rules-section"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('admin.accounts.passthroughFields.hiddenRulesError')
    expect(wrapper.text()).not.toContain('admin.accounts.passthroughFields.errors.sourceKeyRequired')

    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(true)

    expect(wrapper.find('[data-testid="passthrough-rules-section"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('admin.accounts.passthroughFields.errors.sourceKeyRequired')

    await wrapper.get('[data-testid="passthrough-rule-source-key-0"]').setValue('metadata.source')
    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
  })

  it('submits full passthrough extra for apikey accounts', async () => {
    const wrapper = mountModal(buildAccount({
      extra: {
        openai_passthrough: true,
        existing_flag: 'keep-me'
      }
    }))

    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(true)
    await wrapper.get('[data-testid="passthrough-rule-mode-0"]').setValue('inject')
    await wrapper.get('[data-testid="passthrough-rule-key-0"]').setValue('X-Env')
    await wrapper.get('[data-testid="passthrough-rule-value-0"]').setValue('prod')

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra).toEqual(expect.objectContaining({
      existing_flag: 'keep-me',
      openai_passthrough: true,
      openai_apikey_responses_websockets_v2_mode: 'off',
      openai_apikey_responses_websockets_v2_enabled: false,
      passthrough_fields_enabled: true,
      passthrough_field_rules: [
        { target: 'header', mode: 'inject', key: 'X-Env', value: 'prod' }
      ]
    }))
  })

  it('submits map passthrough rules with source_key for edit payload', async () => {
    const wrapper = mountModal(buildAccount())

    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(true)
    await wrapper.get('[data-testid="passthrough-rule-target-0"]').setValue('body')
    await wrapper.get('[data-testid="passthrough-rule-mode-0"]').setValue('map')
    await wrapper.get('[data-testid="passthrough-rule-key-0"]').setValue('metadata.target')
    await wrapper.get('[data-testid="passthrough-rule-source-key-0"]').setValue('metadata.source')

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra).toEqual(expect.objectContaining({
      passthrough_fields_enabled: true,
      passthrough_field_rules: [
        {
          target: 'body',
          mode: 'map',
          key: 'metadata.target',
          source_key: 'metadata.source'
        }
      ]
    }))
  })

  it('keeps openai passthrough extra and model mapping when passthrough fields are submitted', async () => {
    const wrapper = mountModal(buildAccount({
      credentials: {
        model_mapping: {
          'gpt-5.2': 'gpt-5.2',
          'gpt-4.1-mini': 'gpt-4.1-mini'
        }
      },
      extra: {
        openai_passthrough: true,
        openai_apikey_responses_websockets_v2_mode: 'off',
        openai_apikey_responses_websockets_v2_enabled: false
      }
    }))

    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(true)
    await wrapper.get('[data-testid="passthrough-rule-key-0"]').setValue('X-Org')

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.model_mapping).toEqual({
      'gpt-5.2': 'gpt-5.2',
      'gpt-4.1-mini': 'gpt-4.1-mini'
    })
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra).toEqual(expect.objectContaining({
      openai_passthrough: true,
      openai_apikey_responses_websockets_v2_mode: 'off',
      openai_apikey_responses_websockets_v2_enabled: false,
      passthrough_fields_enabled: true,
      passthrough_field_rules: [
        { target: 'header', mode: 'forward', key: 'X-Org' }
      ]
    }))
  })

  it('merges passthrough fields with anthropic passthrough and quota extra without overwriting', async () => {
    const wrapper = mountModal(buildAccount({
      platform: 'anthropic',
      extra: {
        anthropic_passthrough: true
      }
    }))

    await wrapper.get('[data-testid="quota-limit-set"]').trigger('click')
    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(true)
    await wrapper.get('[data-testid="passthrough-rule-key-0"]').setValue('X-Tenant')

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra).toEqual(expect.objectContaining({
      anthropic_passthrough: true,
      quota_limit: 99,
      passthrough_fields_enabled: true,
      passthrough_field_rules: [
        { target: 'header', mode: 'forward', key: 'X-Tenant' }
      ]
    }))
  })

  it('merges passthrough fields with anthropic web search and quota notify settings on edit', async () => {
    const wrapper = mountModal(buildAccount({
      platform: 'anthropic',
      extra: {
        anthropic_passthrough: true,
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'header', mode: 'forward', key: 'X-Test' }
        ],
        quota_notify_total_enabled: true,
        quota_notify_total_threshold: 60,
        quota_notify_total_threshold_type: 'fixed'
      }
    }))

    await flushPromises()
    await findWebSearchSelect(wrapper).setValue('disabled')
    await wrapper.get('[data-testid="passthrough-rule-key-0"]').setValue('X-Tenant')

    const quotaCard = wrapper.getComponent({ name: 'QuotaLimitCard' })
    quotaCard.vm.$emit('update:totalLimit', 120)
    quotaCard.vm.$emit('update:quotaNotifyTotalEnabled', true)
    quotaCard.vm.$emit('update:quotaNotifyTotalThreshold', 70)
    quotaCard.vm.$emit('update:quotaNotifyTotalThresholdType', 'percentage')
    await flushPromises()

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra).toEqual(expect.objectContaining({
      anthropic_passthrough: true,
      passthrough_fields_enabled: true,
      passthrough_field_rules: [
        { target: 'header', mode: 'forward', key: 'X-Tenant' }
      ],
      web_search_emulation: 'disabled',
      quota_limit: 120,
      quota_notify_total_enabled: true,
      quota_notify_total_threshold: 70,
      quota_notify_total_threshold_type: 'percentage'
    }))
  })

  it('renders passthrough section for oauth accounts', () => {
    const wrapper = mountModal(buildAccount({
      type: 'oauth'
    }))

    expect(wrapper.find('[data-testid="passthrough-fields-section"]').exists()).toBe(true)
  })

  it('renders passthrough section for antigravity apikey edit accounts', () => {
    const wrapper = mountModal(buildAccount({
      platform: 'antigravity',
      type: 'apikey',
      credentials: {
        api_key: 'sk-antigravity',
        base_url: 'https://cloudcode-pa.googleapis.com'
      },
      extra: {
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'header', mode: 'forward', key: 'X-Test' }
        ]
      }
    }))

    expect(wrapper.find('[data-testid="passthrough-fields-section"]').exists()).toBe(true)
  })

  it('submits passthrough extra for antigravity apikey edit accounts', async () => {
    const wrapper = mountModal(buildAccount({
      platform: 'antigravity',
      type: 'apikey',
      credentials: {
        api_key: 'sk-antigravity',
        base_url: 'https://cloudcode-pa.googleapis.com'
      },
      extra: {
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'header', mode: 'forward', key: 'X-Test' }
        ],
        mixed_scheduling: true
      }
    }))

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra).toEqual(expect.objectContaining({
      passthrough_fields_enabled: true,
      passthrough_field_rules: [
        { target: 'header', mode: 'forward', key: 'X-Test' }
      ],
      mixed_scheduling: true
    }))
  })

  it('uses antigravity default base_url when cleared before submit', async () => {
    const wrapper = mountModal(buildAccount({
      platform: 'antigravity',
      type: 'apikey',
      credentials: {
        api_key: 'sk-antigravity',
        base_url: 'https://cloudcode-pa.googleapis.com'
      }
    }))

    await wrapper.get('input[placeholder="https://cloudcode-pa.googleapis.com"]').setValue('')
    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.base_url).toBe('https://cloudcode-pa.googleapis.com')
  })

  it('initial anthropic base_url state is sourced from helper', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/components/account/EditAccountModal.vue'), 'utf8')

    expect(source).toContain("const editBaseUrl = ref(getDefaultBaseUrl('anthropic'))")
  })

  it('uses antigravity default base_url on open-and-save when history base_url is missing', async () => {
    const wrapper = mountModal(buildAccount({
      platform: 'antigravity',
      type: 'apikey',
      credentials: {
        api_key: 'sk-antigravity',
        base_url: ''
      }
    }))

    const antigravityBaseUrlInput = wrapper.get(`input[placeholder="${getDefaultBaseUrl('antigravity')}"]`)
    expect((antigravityBaseUrlInput.element as HTMLInputElement).value).toBe(getDefaultBaseUrl('antigravity'))

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.base_url).toBe('https://cloudcode-pa.googleapis.com')
  })

  it('keeps legacy upstream branch on antigravity default base_url path', async () => {
    const wrapper = mountModal(buildAccount({
      platform: 'antigravity',
      type: 'upstream',
      credentials: {
        api_key: 'sk-antigravity',
        base_url: ''
      }
    }))

    const upstreamBaseUrlInput = wrapper.get(`input[placeholder="${getDefaultBaseUrl('antigravity')}"]`)
    expect((upstreamBaseUrlInput.element as HTMLInputElement).value).toBe(getDefaultBaseUrl('antigravity'))

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.credentials?.base_url).toBe(getDefaultBaseUrl('antigravity'))
  })

  it('keeps passthrough section and config when switched from apikey to oauth with passthrough config', async () => {
    const apikeyAccount = buildAccount({
      extra: {
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'header', mode: 'forward', key: 'X-Test' }
        ]
      }
    })
    const wrapper = mountModal(apikeyAccount)

    await wrapper.setProps({
      account: buildAccount({
        id: apikeyAccount.id,
        type: 'oauth',
        extra: apikeyAccount.extra
      })
    })

    expect(wrapper.find('[data-testid="passthrough-fields-section"]').exists()).toBe(true)
    expect(showInfoMock).not.toHaveBeenCalledWith(expect.stringContaining('移除透传字段规则配置'))

    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(updateAccountMock).toHaveBeenCalledTimes(1)
    expect(updateAccountMock.mock.calls[0]?.[1]?.extra).toEqual(expect.objectContaining({
      passthrough_fields_enabled: true,
      passthrough_field_rules: [
        { target: 'header', mode: 'forward', key: 'X-Test' }
      ]
    }))
  })

  it('keeps passthrough support after switching from supported apikey to antigravity apikey', async () => {
    const apikeyAccount = buildAccount({
      platform: 'openai',
      type: 'apikey',
      extra: {
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'header', mode: 'forward', key: 'X-Test' }
        ]
      }
    })
    const wrapper = mountModal(apikeyAccount)

    await wrapper.setProps({
      account: buildAccount({
        id: apikeyAccount.id,
        platform: 'antigravity',
        type: 'apikey',
        credentials: {
          api_key: 'sk-antigravity',
          base_url: 'https://cloudcode-pa.googleapis.com'
        },
        extra: apikeyAccount.extra
      })
    })

    expect(wrapper.find('[data-testid="passthrough-fields-section"]').exists()).toBe(true)
    expect(showInfoMock).not.toHaveBeenCalledWith(expect.stringContaining('移除透传字段规则配置'))
  })
})
