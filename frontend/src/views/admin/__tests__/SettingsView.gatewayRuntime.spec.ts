import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import SettingsView from '../SettingsView.vue'

const {
  getSettings,
  getAllGroups,
  getAdminApiKey,
  getOverloadCooldownSettings,
  getStreamTimeoutSettings,
  getRectifierSettings,
  getBetaPolicySettings,
  getGatewayRuntimeSettings,
  updateGatewayRuntimeSettings,
  updateSettings,
  showSuccess,
  showError,
  fetchAdminSettings
} = vi.hoisted(() => ({
  getSettings: vi.fn(),
  getAllGroups: vi.fn(),
  getAdminApiKey: vi.fn(),
  getOverloadCooldownSettings: vi.fn(),
  getStreamTimeoutSettings: vi.fn(),
  getRectifierSettings: vi.fn(),
  getBetaPolicySettings: vi.fn(),
  getGatewayRuntimeSettings: vi.fn(),
  updateGatewayRuntimeSettings: vi.fn(),
  updateSettings: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
  fetchAdminSettings: vi.fn()
}))

const messages: Record<string, string> = {
  'admin.settings.tabs.gateway': 'Gateway',
  'admin.settings.gatewayRuntime.title': 'Gateway Runtime',
  'gateway.response_header_timeout': 'Response header timeout',
  'gateway.stream_data_interval_timeout': 'Stream data interval timeout',
  'admin.settings.gatewayRuntime.usageLogDetailRetentionLimit': 'Usage log detail retention limit',
  'admin.settings.gatewayRuntime.imageUsageLogDetailRetentionLimit': 'Image usage log detail retention limit',
  'admin.settings.gatewayRuntime.loadFailed': 'Failed to load gateway runtime settings',
  'admin.settings.gatewayRuntime.loadFailedInline': 'Gateway runtime settings failed to load. Saving is disabled until data loads successfully.',
  'admin.settings.gatewayRuntime.saved': 'Gateway runtime settings saved',
  'admin.settings.gatewayRuntime.saveFailed': 'Failed to save gateway runtime settings',
  'admin.settings.gatewayRuntime.validationFailed': 'Gateway runtime settings contain invalid values',
  'common.save': 'Save',
  'common.saving': 'Saving...',
  'common.loading': 'Loading...',
  'common.unknownError': 'Unknown error'
}

vi.mock('@/api', () => ({
  adminAPI: {
    settings: {
      getSettings,
      getAdminApiKey,
      getOverloadCooldownSettings,
      getStreamTimeoutSettings,
      getRectifierSettings,
      getBetaPolicySettings,
      getGatewayRuntimeSettings,
      updateGatewayRuntimeSettings,
      updateSettings
    },
    groups: {
      getAll: getAllGroups
    }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showSuccess,
    showError,
    fetchPublicSettings: vi.fn()
  })
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    fetch: fetchAdminSettings
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'en' },
      t: (key: string) => messages[key] ?? key
    })
  }
})

function createWrapper() {
  return mount(SettingsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: true,
        RouterLink: true,
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template:
            '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>'
        },
        GroupBadge: true,
        GroupOptionItem: true,
        Toggle: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<input type="checkbox" :checked="modelValue" @change="$emit(\'update:modelValue\', $event.target.checked)" />'
        },
        ImageUpload: true,
        BackupSettings: true,
        DataManagementSettings: true,
      }
    }
  })
}

function findGatewayRuntimeField(wrapper: ReturnType<typeof createWrapper>, testId: string) {
  return wrapper.find(`[data-testid="${testId}"]`)
}

function findGatewayRuntimeSaveButton(wrapper: ReturnType<typeof createWrapper>) {
  return wrapper.find('[data-testid="gateway-runtime-save"]')
}

function getOpenAIFastPolicyState(wrapper: ReturnType<typeof createWrapper>) {
  const vm = wrapper.vm as any
  const setupState = vm.$?.setupState
  const openaiFastPolicyForm = setupState?.openaiFastPolicyForm ?? vm.openaiFastPolicyForm
  const openaiFastPolicyLoadedState =
    setupState?.openaiFastPolicyLoaded ?? vm.openaiFastPolicyLoaded

  return {
    openaiFastPolicyForm,
    openaiFastPolicyLoaded:
      openaiFastPolicyLoadedState && typeof openaiFastPolicyLoadedState === 'object' && 'value' in openaiFastPolicyLoadedState
        ? openaiFastPolicyLoadedState.value
        : openaiFastPolicyLoadedState
  }
}

describe('admin SettingsView gateway runtime card', () => {
  beforeEach(() => {
    getSettings.mockReset()
    getAllGroups.mockReset()
    getAdminApiKey.mockReset()
    getOverloadCooldownSettings.mockReset()
    getStreamTimeoutSettings.mockReset()
    getRectifierSettings.mockReset()
    getBetaPolicySettings.mockReset()
    getGatewayRuntimeSettings.mockReset()
    updateGatewayRuntimeSettings.mockReset()
    updateSettings.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    fetchAdminSettings.mockReset()

    getSettings.mockResolvedValue({
      backend_mode_enabled: false,
      default_subscriptions: [],
      registration_email_suffix_whitelist: [],
      gateway_sticky_openai_enabled: false,
      gateway_sticky_gemini_enabled: true,
      gateway_sticky_anthropic_enabled: false,
      gateway_openai_ws_scheduler_mode: 'layered',
      gateway_openai_ws_scheduler_layered_error_penalty_threshold: 0.6,
      gateway_openai_ws_scheduler_layered_error_penalty_value: 100,
      gateway_openai_ws_scheduler_layered_ttft_penalty_multiplier: 12,
      gateway_openai_ws_scheduler_layered_ttft_penalty_value: 50,
      gateway_openai_ws_scheduler_layered_probe_cooldown_seconds: 20,
      gateway_openai_ws_scheduler_layered_probe_interval_seconds: 20,
      gateway_openai_ws_scheduler_layered_probe_max_failures: 3,
      gateway_openai_ws_scheduler_layered_probe_timeout_seconds: 15
    } as any)
    getAllGroups.mockResolvedValue([])
    getAdminApiKey.mockResolvedValue({ exists: false, masked_key: '' })
    getOverloadCooldownSettings.mockResolvedValue({ enabled: true, cooldown_minutes: 10 })
    getStreamTimeoutSettings.mockResolvedValue({
      enabled: true,
      action: 'temp_unsched',
      temp_unsched_minutes: 5,
      threshold_count: 3,
      threshold_window_minutes: 10
    })
    getRectifierSettings.mockResolvedValue({
      enabled: true,
      thinking_signature_enabled: true,
      thinking_budget_enabled: true
    })
    getBetaPolicySettings.mockResolvedValue({ rules: [] })
    getGatewayRuntimeSettings.mockResolvedValue({
      response_header_timeout: 120,
      stream_data_interval_timeout: 45,
      usage_log_detail_retention_limit: 320,
      image_usage_log_detail_retention_limit: 80
    })
    updateGatewayRuntimeSettings.mockResolvedValue({
      response_header_timeout: 240,
      stream_data_interval_timeout: 90,
      usage_log_detail_retention_limit: 500,
      image_usage_log_detail_retention_limit: 120
    })
    updateSettings.mockResolvedValue({
      backend_mode_enabled: false,
      default_subscriptions: [],
      registration_email_suffix_whitelist: [],
      gateway_sticky_openai_enabled: false,
      gateway_sticky_gemini_enabled: true,
      gateway_sticky_anthropic_enabled: false,
      gateway_openai_ws_scheduler_mode: 'layered',
      gateway_openai_ws_scheduler_layered_error_penalty_threshold: 0.6,
      gateway_openai_ws_scheduler_layered_error_penalty_value: 100,
      gateway_openai_ws_scheduler_layered_ttft_penalty_multiplier: 12,
      gateway_openai_ws_scheduler_layered_ttft_penalty_value: 50,
      gateway_openai_ws_scheduler_layered_probe_cooldown_seconds: 20,
      gateway_openai_ws_scheduler_layered_probe_interval_seconds: 20,
      gateway_openai_ws_scheduler_layered_probe_max_failures: 3,
      gateway_openai_ws_scheduler_layered_probe_timeout_seconds: 15
    })
  })

  it('loads gateway runtime settings on mount and renders the new card', async () => {
    const wrapper = createWrapper()

    await flushPromises()

    expect(getGatewayRuntimeSettings).toHaveBeenCalledTimes(1)
    expect(findGatewayRuntimeField(wrapper, 'gateway-runtime-response-header-timeout').exists()).toBe(true)
    expect(findGatewayRuntimeField(wrapper, 'gateway-runtime-stream-interval-timeout').exists()).toBe(true)
    expect(findGatewayRuntimeField(wrapper, 'gateway-runtime-usage-log-detail-retention-limit').exists()).toBe(true)
    expect(findGatewayRuntimeField(wrapper, 'gateway-runtime-image-usage-log-detail-retention-limit').exists()).toBe(true)

    expect(
      (
        findGatewayRuntimeField(wrapper, 'gateway-runtime-response-header-timeout')
          .element as HTMLInputElement
      ).value
    ).toBe('120')
    expect(
      (
        findGatewayRuntimeField(wrapper, 'gateway-runtime-stream-interval-timeout')
          .element as HTMLInputElement
      ).value
    ).toBe('45')
    expect(
      (
        findGatewayRuntimeField(wrapper, 'gateway-runtime-usage-log-detail-retention-limit')
          .element as HTMLInputElement
      ).value
    ).toBe('320')
    expect(
      (
        findGatewayRuntimeField(wrapper, 'gateway-runtime-image-usage-log-detail-retention-limit')
          .element as HTMLInputElement
      ).value
    ).toBe('80')
  })

  it('loads sticky and OpenAI WS scheduler settings and includes them when saving system settings', async () => {
    const wrapper = createWrapper()

    await flushPromises()

    expect((wrapper.find('[data-testid="gateway-sticky-openai-enabled"]').element as HTMLInputElement).checked).toBe(false)
    expect((wrapper.find('[data-testid="gateway-sticky-gemini-enabled"]').element as HTMLInputElement).checked).toBe(true)
    expect((wrapper.find('[data-testid="gateway-sticky-anthropic-enabled"]').element as HTMLInputElement).checked).toBe(false)
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-mode"]').element as HTMLSelectElement).value).toBe('layered')
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-layered-error-penalty-threshold"]').element as HTMLInputElement).value).toBe('0.6')
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-layered-ttft-penalty-multiplier"]').element as HTMLInputElement).value).toBe('12')

    await wrapper.find('[data-testid="settings-form"]').trigger('submit')
    await flushPromises()

    expect(updateSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        gateway_sticky_openai_enabled: false,
        gateway_sticky_gemini_enabled: true,
        gateway_sticky_anthropic_enabled: false,
        gateway_openai_ws_scheduler_mode: 'layered',
        gateway_openai_ws_scheduler_layered_error_penalty_threshold: 0.6,
        gateway_openai_ws_scheduler_layered_error_penalty_value: 100,
        gateway_openai_ws_scheduler_layered_ttft_penalty_multiplier: 12,
        gateway_openai_ws_scheduler_layered_ttft_penalty_value: 50,
        gateway_openai_ws_scheduler_layered_probe_cooldown_seconds: 20,
        gateway_openai_ws_scheduler_layered_probe_interval_seconds: 20,
        gateway_openai_ws_scheduler_layered_probe_max_failures: 3,
        gateway_openai_ws_scheduler_layered_probe_timeout_seconds: 15
      })
    )
  })

  it('round-trips OpenAI fast policy together with sticky and layered scheduler settings', async () => {
    const initialCoexistSettings = {
      backend_mode_enabled: false,
      default_subscriptions: [],
      registration_email_suffix_whitelist: [],
      gateway_sticky_openai_enabled: false,
      gateway_sticky_gemini_enabled: true,
      gateway_sticky_anthropic_enabled: false,
      gateway_openai_ws_scheduler_mode: 'layered',
      gateway_openai_ws_scheduler_layered_error_penalty_threshold: 0.55,
      gateway_openai_ws_scheduler_layered_error_penalty_value: 77,
      gateway_openai_ws_scheduler_layered_ttft_penalty_multiplier: 6,
      gateway_openai_ws_scheduler_layered_ttft_penalty_value: 33,
      gateway_openai_ws_scheduler_layered_probe_cooldown_seconds: 44,
      gateway_openai_ws_scheduler_layered_probe_interval_seconds: 22,
      gateway_openai_ws_scheduler_layered_probe_max_failures: 5,
      gateway_openai_ws_scheduler_layered_probe_timeout_seconds: 11,
      openai_fast_policy_settings: {
        rules: [
          {
            service_tier: 'priority',
            action: 'filter',
            scope: 'oauth',
            model_whitelist: ['gpt-4.1', 'gpt-4o'],
            fallback_action: 'block',
            fallback_error_message: 'tier blocked'
          }
        ]
      }
    }
    const updatedCoexistSettings = {
      ...initialCoexistSettings,
      gateway_sticky_openai_enabled: true,
      gateway_openai_ws_scheduler_layered_error_penalty_threshold: 0.85,
      gateway_openai_ws_scheduler_layered_ttft_penalty_multiplier: 9,
      gateway_openai_ws_scheduler_layered_probe_interval_seconds: 31,
      openai_fast_policy_settings: {
        rules: [
          {
            service_tier: 'priority',
            action: 'filter',
            scope: 'oauth',
            model_whitelist: ['gpt-4.1-mini'],
            fallback_action: 'block',
            fallback_error_message: 'updated tier blocked'
          }
        ]
      }
    }
    getSettings.mockResolvedValueOnce(initialCoexistSettings as any)
    updateSettings.mockResolvedValueOnce(updatedCoexistSettings as any)

    const wrapper = createWrapper()

    await flushPromises()

    const { openaiFastPolicyForm, openaiFastPolicyLoaded } = getOpenAIFastPolicyState(wrapper)

    expect((wrapper.find('[data-testid="gateway-sticky-openai-enabled"]').element as HTMLInputElement).checked).toBe(initialCoexistSettings.gateway_sticky_openai_enabled)
    expect((wrapper.find('[data-testid="gateway-sticky-gemini-enabled"]').element as HTMLInputElement).checked).toBe(initialCoexistSettings.gateway_sticky_gemini_enabled)
    expect((wrapper.find('[data-testid="gateway-sticky-anthropic-enabled"]').element as HTMLInputElement).checked).toBe(initialCoexistSettings.gateway_sticky_anthropic_enabled)
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-mode"]').element as HTMLSelectElement).value).toBe(initialCoexistSettings.gateway_openai_ws_scheduler_mode)
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-layered-error-penalty-threshold"]').element as HTMLInputElement).value).toBe(String(initialCoexistSettings.gateway_openai_ws_scheduler_layered_error_penalty_threshold))
    expect(openaiFastPolicyLoaded).toBe(true)
    expect(openaiFastPolicyForm.rules).toEqual([
      expect.objectContaining({
        service_tier: 'priority',
        action: 'filter',
        scope: 'oauth',
         model_whitelist: initialCoexistSettings.openai_fast_policy_settings.rules[0].model_whitelist,
         fallback_action: 'block',
         fallback_error_message: initialCoexistSettings.openai_fast_policy_settings.rules[0].fallback_error_message
       })
     ])

    await wrapper.find('[data-testid="settings-form"]').trigger('submit')
    await flushPromises()

    expect(updateSettings).toHaveBeenCalledTimes(1)

    const payload = updateSettings.mock.calls[0][0]

    expect(payload).toEqual(
      expect.objectContaining({
        gateway_sticky_openai_enabled: false,
        gateway_sticky_gemini_enabled: true,
        gateway_sticky_anthropic_enabled: false,
        gateway_openai_ws_scheduler_mode: 'layered',
        gateway_openai_ws_scheduler_layered_error_penalty_threshold: 0.55,
        gateway_openai_ws_scheduler_layered_error_penalty_value: 77,
        gateway_openai_ws_scheduler_layered_ttft_penalty_multiplier: 6,
        gateway_openai_ws_scheduler_layered_ttft_penalty_value: 33,
        gateway_openai_ws_scheduler_layered_probe_cooldown_seconds: 44,
        gateway_openai_ws_scheduler_layered_probe_interval_seconds: 22,
        gateway_openai_ws_scheduler_layered_probe_max_failures: 5,
        gateway_openai_ws_scheduler_layered_probe_timeout_seconds: 11
      })
    )
    expect(payload.openai_fast_policy_settings.rules[0]).toEqual(
      expect.objectContaining({
        service_tier: 'priority',
        action: 'filter',
        scope: 'oauth',
        model_whitelist: ['gpt-4.1', 'gpt-4o'],
        fallback_action: 'block',
        fallback_error_message: 'tier blocked'
      })
    )
    expect((wrapper.find('[data-testid="gateway-sticky-openai-enabled"]').element as HTMLInputElement).checked).toBe(updatedCoexistSettings.gateway_sticky_openai_enabled)
    expect((wrapper.find('[data-testid="gateway-sticky-openai-enabled"]').element as HTMLInputElement).checked).not.toBe(initialCoexistSettings.gateway_sticky_openai_enabled)
    expect((wrapper.find('[data-testid="gateway-sticky-gemini-enabled"]').element as HTMLInputElement).checked).toBe(updatedCoexistSettings.gateway_sticky_gemini_enabled)
    expect((wrapper.find('[data-testid="gateway-sticky-anthropic-enabled"]').element as HTMLInputElement).checked).toBe(updatedCoexistSettings.gateway_sticky_anthropic_enabled)
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-mode"]').element as HTMLSelectElement).value).toBe(updatedCoexistSettings.gateway_openai_ws_scheduler_mode)
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-layered-error-penalty-threshold"]').element as HTMLInputElement).value).toBe(String(updatedCoexistSettings.gateway_openai_ws_scheduler_layered_error_penalty_threshold))
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-layered-error-penalty-threshold"]').element as HTMLInputElement).value).not.toBe(String(initialCoexistSettings.gateway_openai_ws_scheduler_layered_error_penalty_threshold))
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-layered-error-penalty-value"]').element as HTMLInputElement).value).toBe(String(updatedCoexistSettings.gateway_openai_ws_scheduler_layered_error_penalty_value))
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-layered-ttft-penalty-multiplier"]').element as HTMLInputElement).value).toBe(String(updatedCoexistSettings.gateway_openai_ws_scheduler_layered_ttft_penalty_multiplier))
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-layered-ttft-penalty-multiplier"]').element as HTMLInputElement).value).not.toBe(String(initialCoexistSettings.gateway_openai_ws_scheduler_layered_ttft_penalty_multiplier))
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-layered-probe-interval-seconds"]').element as HTMLInputElement).value).toBe(String(updatedCoexistSettings.gateway_openai_ws_scheduler_layered_probe_interval_seconds))
    expect((wrapper.find('[data-testid="gateway-openai-ws-scheduler-layered-probe-interval-seconds"]').element as HTMLInputElement).value).not.toBe(String(initialCoexistSettings.gateway_openai_ws_scheduler_layered_probe_interval_seconds))
    const refreshedOpenAIFastPolicyState = getOpenAIFastPolicyState(wrapper)

    expect(refreshedOpenAIFastPolicyState.openaiFastPolicyLoaded).toBe(true)
    expect(refreshedOpenAIFastPolicyState.openaiFastPolicyForm.rules).toEqual([
      expect.objectContaining({
        service_tier: 'priority',
        action: 'filter',
        scope: 'oauth',
        model_whitelist: updatedCoexistSettings.openai_fast_policy_settings.rules[0].model_whitelist,
        fallback_action: 'block',
        fallback_error_message: updatedCoexistSettings.openai_fast_policy_settings.rules[0].fallback_error_message
      })
    ])
    expect(refreshedOpenAIFastPolicyState.openaiFastPolicyForm.rules).not.toEqual(initialCoexistSettings.openai_fast_policy_settings.rules)
  })

  it('updates gateway runtime settings and shows success feedback', async () => {
    const wrapper = createWrapper()

    await flushPromises()

    await findGatewayRuntimeField(wrapper, 'gateway-runtime-response-header-timeout')
      .setValue('240')
    await findGatewayRuntimeField(wrapper, 'gateway-runtime-stream-interval-timeout')
      .setValue('90')
    await findGatewayRuntimeField(wrapper, 'gateway-runtime-usage-log-detail-retention-limit')
      .setValue('500')
    await findGatewayRuntimeField(wrapper, 'gateway-runtime-image-usage-log-detail-retention-limit')
      .setValue('120')
    await findGatewayRuntimeSaveButton(wrapper).trigger('click')
    await flushPromises()

    expect(updateGatewayRuntimeSettings).toHaveBeenCalledWith({
      response_header_timeout: 240,
      stream_data_interval_timeout: 90,
      usage_log_detail_retention_limit: 500,
      image_usage_log_detail_retention_limit: 120
    })
    expect(showSuccess).toHaveBeenCalledWith('Gateway runtime settings saved')
  })

  it.each([
    {
      name: 'empty response header timeout',
      testId: 'gateway-runtime-response-header-timeout',
      value: ''
    },
    {
      name: 'stream interval outside the allowed non-zero range',
      testId: 'gateway-runtime-stream-interval-timeout',
      value: '20'
    },
    {
      name: 'negative normal detail retention limit',
      testId: 'gateway-runtime-usage-log-detail-retention-limit',
      value: '-1'
    },
    {
      name: 'negative image detail retention limit',
      testId: 'gateway-runtime-image-usage-log-detail-retention-limit',
      value: '-1'
    }
  ])('does not submit gateway runtime settings with $name', async ({ testId, value }) => {
    const wrapper = createWrapper()

    await flushPromises()

    await findGatewayRuntimeField(wrapper, testId).setValue(value)
    await findGatewayRuntimeSaveButton(wrapper).trigger('click')
    await flushPromises()

    expect(updateGatewayRuntimeSettings).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('Gateway runtime settings contain invalid values')
  })

  it('shows error feedback when saving gateway runtime settings fails', async () => {
    updateGatewayRuntimeSettings.mockRejectedValueOnce(new Error('boom'))

    const wrapper = createWrapper()

    await flushPromises()

    await findGatewayRuntimeSaveButton(wrapper).trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('Failed to save gateway runtime settings: boom')
  })

  it('shows load failure and disables saving when gateway runtime settings fail to load', async () => {
    getGatewayRuntimeSettings.mockRejectedValueOnce(new Error('load boom'))

    const wrapper = createWrapper()

    await flushPromises()

    const saveButton = findGatewayRuntimeSaveButton(wrapper)

    expect(showError).toHaveBeenCalledWith('Failed to load gateway runtime settings: load boom')
    expect(wrapper.text()).toContain(
      'Gateway runtime settings failed to load. Saving is disabled until data loads successfully.'
    )
    expect((saveButton.element as HTMLButtonElement).disabled).toBe(true)

    await saveButton.trigger('click')

    expect(updateGatewayRuntimeSettings).not.toHaveBeenCalled()
  })
})
