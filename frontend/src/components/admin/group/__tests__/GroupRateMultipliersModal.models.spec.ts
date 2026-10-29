import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import type { AdminGroup, ExclusiveRateModels } from '@/types'
import GroupRateMultipliersModal from '../GroupRateMultipliersModal.vue'

const mocks = vi.hoisted(() => ({ getById: vi.fn(), getGroupRateMultipliers: vi.fn(), getModelAllowlistCandidates: vi.fn(), batchSetGroupRateMultipliers: vi.fn() }))
vi.mock('@/api/admin', () => ({ adminAPI: { groups: mocks } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
enableAutoUnmount(afterEach)
let saved: ExclusiveRateModels
beforeEach(() => {
  vi.resetAllMocks()
  saved = { enabled: false, models: [] }
  mocks.getById.mockImplementation(async () => ({ exclusive_rate_models: structuredClone(saved) }))
  mocks.getGroupRateMultipliers.mockResolvedValue([])
  mocks.getModelAllowlistCandidates.mockResolvedValue(['client-A', 'client-B'])
  mocks.batchSetGroupRateMultipliers.mockImplementation(async (_id, _entries, cfg) => { saved = structuredClone(cfg) })
})
async function open() {
  const wrapper = mount(GroupRateMultipliersModal, {
    props: { show: true, group: { id: 1, platform: 'openai', rate_multiplier: 0.225 } as AdminGroup },
    global: { stubs: { BaseDialog: { template: '<div><slot /></div>' }, Icon: true, PlatformIcon: true, Pagination: true } }
  })
  await flushPromises()
  return wrapper
}

describe('shared exclusive rate model policy', () => {
  it('defaults off, rejects an empty enabled selection and saves exact IDs only explicitly', async () => {
    const wrapper = await open()
    expect((wrapper.get('[data-testid="exclusive-rate-enabled"]').element as HTMLInputElement).checked).toBe(false)
    await wrapper.get('[data-testid="exclusive-rate-enabled"]').setValue(true)
    const save = () => wrapper.findAll('button').find(b => b.text() === 'common.save')!
    expect(save().attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('admin.groups.exclusiveRateModelsRequired')
    await save().trigger('click')
    expect(mocks.batchSetGroupRateMultipliers).not.toHaveBeenCalled()
    await wrapper.get('input[value="client-A"]').setValue(true)
    expect(mocks.batchSetGroupRateMultipliers).not.toHaveBeenCalled()
    await save().trigger('click')
    await flushPromises()
    expect(saved).toEqual({ enabled: true, models: ['client-A'] })
    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect((wrapper.get('input[value="client-A"]').element as HTMLInputElement).checked).toBe(true)
  })

  it('keeps selection while off and reverts cancelled edits', async () => {
    saved = { enabled: true, models: ['client-A'] }
    const wrapper = await open()
    await wrapper.get('[data-testid="exclusive-rate-enabled"]').setValue(false)
    await wrapper.get('[data-testid="exclusive-rate-enabled"]').setValue(true)
    expect((wrapper.get('input[value="client-A"]').element as HTMLInputElement).checked).toBe(true)
    await wrapper.get('input[value="client-B"]').setValue(true)
    await wrapper.findAll('button').find(b => b.text() === 'admin.groups.revertChanges')!.trigger('click')
    expect((wrapper.get('input[value="client-B"]').element as HTMLInputElement).checked).toBe(false)
    expect(mocks.batchSetGroupRateMultipliers).not.toHaveBeenCalled()
    await wrapper.get('[data-testid="exclusive-rate-enabled"]').setValue(false)
    await wrapper.findAll('button').find(b => b.text() === 'common.save')!.trigger('click')
    await flushPromises()
    expect(saved).toEqual({ enabled: false, models: ['client-A'] })
  })

  it('searches candidates and retries failures without losing saved or draft selection', async () => {
    saved = { enabled: true, models: ['removed-from-upstream'] }
    mocks.getModelAllowlistCandidates.mockRejectedValueOnce(new Error('offline'))
    const wrapper = await open()
    expect(wrapper.text()).toContain('admin.groups.exclusiveRateModelsLoadFailed')
    expect((wrapper.get('input[value="removed-from-upstream"]').element as HTMLInputElement).checked).toBe(true)
    await wrapper.findAll('button').find(b => b.text() === 'common.retry')!.trigger('click')
    await flushPromises()
    await wrapper.get('input[value="client-B"]').setValue(true)
    await wrapper.get('input[type="search"]').setValue('client-B')
    expect(wrapper.find('input[value="client-A"]').exists()).toBe(false)
    expect((wrapper.get('input[value="client-B"]').element as HTMLInputElement).checked).toBe(true)
    await wrapper.get('input[type="search"]').setValue('')
    expect((wrapper.get('input[value="removed-from-upstream"]').element as HTMLInputElement).checked).toBe(true)
  })
})
