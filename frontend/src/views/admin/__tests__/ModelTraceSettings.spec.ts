import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ModelTraceSettings from '../settings/ModelTraceSettings.vue'
const api = vi.hoisted(() => ({ getSettings: vi.fn(), getModels: vi.fn(), updateSettings: vi.fn(), showSuccess: vi.fn() }))
vi.mock('@/api/admin/modeltrace', () => ({ default: api }))
vi.mock('@/stores/app', () => ({ useAppStore: () => api }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
enableAutoUnmount(afterEach)
const settings = { enabled: false, model: 'gpt-6-astra', rounds: 1, interval_minutes: 60, probe_timeout_seconds: 90, task_timeout_seconds: 300, accepted_models: [] }
beforeEach(() => {
  vi.resetAllMocks()
  api.getSettings.mockResolvedValue(settings)
  api.getModels.mockResolvedValue({ models: ['gpt-6-astra', 'gpt-6-sol', 'gpt-5.6-sol'], version: 'v1' })
  api.updateSettings.mockImplementation(async value => value)
})
describe('independent ModelTrace gateway settings', () => {
  it('loads and saves only through the independent API', async () => {
    const w = mount(ModelTraceSettings); await flushPromises()
    await w.get('[data-testid="modeltrace-global-enabled"]').trigger('click')
    await w.get('[data-testid="modeltrace-global-model"]').setValue('gpt-5.6-sol')
    await w.get('[data-testid="modeltrace-global-rounds"]').setValue('3')
    await w.get('[data-testid="modeltrace-global-interval"]').setValue('10080')
    await w.get('[data-testid="modeltrace-global-probe-timeout"]').setValue('180')
    await w.get('[data-testid="modeltrace-global-task-timeout"]').setValue('600')
    await w.get('[data-testid="modeltrace-settings-save"]').trigger('click'); await flushPromises()
    expect(api.updateSettings).toHaveBeenCalledWith({ enabled: true, model: 'gpt-5.6-sol', rounds: 3, interval_minutes: 10080, probe_timeout_seconds: 180, task_timeout_seconds: 600, accepted_models: [] })
    expect(api.showSuccess).toHaveBeenCalledOnce()
  })
  it('does not silently replace failed loading with default settings', async () => {
    api.getSettings.mockRejectedValueOnce({ message: 'server down' })
    const w = mount(ModelTraceSettings); await flushPromises()
    expect(w.text()).toContain('server down')
    expect(w.get('[data-testid="modeltrace-settings-save"]').attributes('disabled')).toBeDefined()
    expect(api.updateSettings).not.toHaveBeenCalled()
    await w.get('[role="alert"] button').trigger('click'); await flushPromises()
    expect(w.get('[data-testid="modeltrace-settings-save"]').attributes('disabled')).toBeUndefined()
  })
  it.each(['', '4', '10081', '5.5'])('rejects invalid interval %s', async value => {
    const w = mount(ModelTraceSettings); await flushPromises()
    await w.get('[data-testid="modeltrace-global-interval"]').setValue(value)
    await w.get('[data-testid="modeltrace-settings-save"]').trigger('click')
    expect(api.updateSettings).not.toHaveBeenCalled()
    expect(w.text()).toContain('admin.modeltrace.invalidSettings')
  })
  it.each([
    ['modeltrace-global-probe-timeout', '9'],
    ['modeltrace-global-probe-timeout', '1801'],
    ['modeltrace-global-probe-timeout', '30.5'],
    ['modeltrace-global-task-timeout', '29'],
    ['modeltrace-global-task-timeout', '7201'],
    ['modeltrace-global-task-timeout', '89'],
  ])('rejects invalid timeout %s = %s', async (field, value) => {
    const w = mount(ModelTraceSettings); await flushPromises()
    await w.get(`[data-testid="${field}"]`).setValue(value)
    await w.get('[data-testid="modeltrace-settings-save"]').trigger('click')
    expect(api.updateSettings).not.toHaveBeenCalled()
    expect(w.text()).toContain('admin.modeltrace.invalidSettings')
  })
  it('preserves configured unavailable models and rejects enabling them', async () => {
    api.getSettings.mockResolvedValueOnce({ ...settings, enabled: true, model: 'removed-model' })
    const w = mount(ModelTraceSettings); await flushPromises()
    expect(w.get<HTMLSelectElement>('[data-testid="modeltrace-global-model"]').element.value).toBe('removed-model')
    await w.get('[data-testid="modeltrace-settings-save"]').trigger('click')
    expect(api.updateSettings).not.toHaveBeenCalled()
  })
  it('preserves edits and displays server failures on save', async () => {
    api.updateSettings.mockRejectedValueOnce({ message: 'invalid fingerprint model' })
    const w = mount(ModelTraceSettings); await flushPromises()
    await w.get('[data-testid="modeltrace-global-interval"]').setValue('5')
    await w.get('[data-testid="modeltrace-settings-save"]').trigger('click'); await flushPromises()
    expect(w.text()).toContain('invalid fingerprint model')
    expect(w.get<HTMLInputElement>('[data-testid="modeltrace-global-interval"]').element.value).toBe('5')
    expect(api.showSuccess).not.toHaveBeenCalled()
  })
  it('selects, toggles, and saves additional accepted attribution models', async () => {
    const w = mount(ModelTraceSettings); await flushPromises()
    const checkbox = w.get<HTMLInputElement>('[data-testid="modeltrace-accepted-option-gpt-6-sol"]')
    expect(checkbox.element.checked).toBe(false)
    await checkbox.setValue(true)
    expect(checkbox.element.checked).toBe(true)
    await w.get('[data-testid="modeltrace-settings-save"]').trigger('click'); await flushPromises()
    expect(api.updateSettings).toHaveBeenCalledWith(expect.objectContaining({
      accepted_models: ['gpt-6-sol']
    }))
  })
  it('preserves stale saved accepted models and allows removing them', async () => {
    api.getSettings.mockResolvedValueOnce({
      ...settings,
      accepted_models: ['stale-v1-model', 'gpt-6-sol']
    })
    const w = mount(ModelTraceSettings); await flushPromises()
    const staleCheckbox = w.get<HTMLInputElement>('[data-testid="modeltrace-accepted-option-stale-v1-model"]')
    const validCheckbox = w.get<HTMLInputElement>('[data-testid="modeltrace-accepted-option-gpt-6-sol"]')
    expect(staleCheckbox.element.checked).toBe(true)
    expect(validCheckbox.element.checked).toBe(true)
    expect(w.text()).toContain('admin.modeltrace.staleModel')

    await staleCheckbox.setValue(false)
    expect(staleCheckbox.element.checked).toBe(false)

    await w.get('[data-testid="modeltrace-settings-save"]').trigger('click'); await flushPromises()
    expect(api.updateSettings).toHaveBeenCalledWith(expect.objectContaining({
      accepted_models: ['gpt-6-sol']
    }))
  })
})
