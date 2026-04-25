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
  showSuccess: vi.fn(),
  showError: vi.fn(),
  fetchAdminSettings: vi.fn()
}))

const messages: Record<string, string> = {
  'admin.settings.tabs.gateway': 'Gateway',
  'admin.settings.gatewayRuntime.title': 'Gateway Runtime',
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
      updateGatewayRuntimeSettings
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
        Select: true,
        GroupBadge: true,
        GroupOptionItem: true,
        Toggle: true,
        ImageUpload: true,
        BackupSettings: true,
        DataManagementSettings: true,
      }
    }
  })
}

function createGatewayRuntimeCard(wrapper: ReturnType<typeof createWrapper>) {
  return wrapper.findAll('.card').find((card) => card.text().includes('Gateway Runtime'))
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
    showSuccess.mockReset()
    showError.mockReset()
    fetchAdminSettings.mockReset()

    getSettings.mockResolvedValue({
      backend_mode_enabled: false,
      default_subscriptions: [],
      registration_email_suffix_whitelist: []
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
  })

  it('loads gateway runtime settings on mount and renders the new card', async () => {
    const wrapper = createWrapper()

    await flushPromises()

    expect(getGatewayRuntimeSettings).toHaveBeenCalledTimes(1)

    const runtimeCard = createGatewayRuntimeCard(wrapper)
    expect(runtimeCard?.exists()).toBe(true)
    expect(runtimeCard?.text()).toContain('gateway.response_header_timeout')
    expect(runtimeCard?.text()).toContain('gateway.stream_data_interval_timeout')
    expect(runtimeCard?.text()).toContain('Usage log detail retention limit')
    expect(runtimeCard?.text()).toContain('Image usage log detail retention limit')

    expect(
      (
        runtimeCard!.find('[data-testid="gateway-runtime-response-header-timeout"]')
          .element as HTMLInputElement
      ).value
    ).toBe('120')
    expect(
      (
        runtimeCard!.find('[data-testid="gateway-runtime-stream-interval-timeout"]')
          .element as HTMLInputElement
      ).value
    ).toBe('45')
    expect(
      (
        runtimeCard!.find('[data-testid="gateway-runtime-usage-log-detail-retention-limit"]')
          .element as HTMLInputElement
      ).value
    ).toBe('320')
    expect(
      (
        runtimeCard!.find('[data-testid="gateway-runtime-image-usage-log-detail-retention-limit"]')
          .element as HTMLInputElement
      ).value
    ).toBe('80')
  })

  it('updates gateway runtime settings and shows success feedback', async () => {
    const wrapper = createWrapper()

    await flushPromises()

    const runtimeCard = createGatewayRuntimeCard(wrapper)!

    await runtimeCard
      .find('[data-testid="gateway-runtime-response-header-timeout"]')
      .setValue('240')
    await runtimeCard
      .find('[data-testid="gateway-runtime-stream-interval-timeout"]')
      .setValue('90')
    await runtimeCard
      .find('[data-testid="gateway-runtime-usage-log-detail-retention-limit"]')
      .setValue('500')
    await runtimeCard
      .find('[data-testid="gateway-runtime-image-usage-log-detail-retention-limit"]')
      .setValue('120')
    await runtimeCard.find('[data-testid="gateway-runtime-save"]').trigger('click')
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

    const runtimeCard = createGatewayRuntimeCard(wrapper)!

    await runtimeCard.find(`[data-testid="${testId}"]`).setValue(value)
    await runtimeCard.find('[data-testid="gateway-runtime-save"]').trigger('click')
    await flushPromises()

    expect(updateGatewayRuntimeSettings).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('Gateway runtime settings contain invalid values')
  })

  it('shows error feedback when saving gateway runtime settings fails', async () => {
    updateGatewayRuntimeSettings.mockRejectedValueOnce(new Error('boom'))

    const wrapper = createWrapper()

    await flushPromises()

    const runtimeCard = createGatewayRuntimeCard(wrapper)!
    await runtimeCard.find('[data-testid="gateway-runtime-save"]').trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('Failed to save gateway runtime settings: boom')
  })

  it('shows load failure and disables saving when gateway runtime settings fail to load', async () => {
    getGatewayRuntimeSettings.mockRejectedValueOnce(new Error('load boom'))

    const wrapper = createWrapper()

    await flushPromises()

    const runtimeCard = createGatewayRuntimeCard(wrapper)!
    const saveButton = runtimeCard.find('[data-testid="gateway-runtime-save"]')

    expect(showError).toHaveBeenCalledWith('Failed to load gateway runtime settings: load boom')
    expect(runtimeCard.text()).toContain(
      'Gateway runtime settings failed to load. Saving is disabled until data loads successfully.'
    )
    expect((saveButton.element as HTMLButtonElement).disabled).toBe(true)

    await saveButton.trigger('click')

    expect(updateGatewayRuntimeSettings).not.toHaveBeenCalled()
  })
})
