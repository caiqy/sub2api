import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import UserRemainingQuotaSummary from '../UserRemainingQuotaSummary.vue'

const { getRemainingQuota, saveQuotaExclusions } = vi.hoisted(() => ({ getRemainingQuota: vi.fn(), saveQuotaExclusions: vi.fn() }))
vi.mock('@/api/admin/users', () => ({ getRemainingQuota, saveQuotaExclusions }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key })
}))
vi.mock('@/views/admin/settings/OpenAIFastPolicyUserSelector.vue', () => ({ default: { template: '<div />' } }))
const options = {
  props: { refreshKey: 0 },
  global: { stubs: {
    BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
    OpenAIFastPolicyUserSelector: {
      props: ['modelValue'],
      template: '<div data-test="selected">{{ modelValue }}<button data-test="select-user" @click="$emit(\'update:modelValue\', [7, 9])">Select</button><button data-test="clear" @click="$emit(\'update:modelValue\', [])">Clear</button></div>'
    }
  } }
}
const quota = { total: 60, balance: 10, subscription: 50, unlimited_subscriptions: 1, excluded_user_ids: [3] }

beforeEach(() => {
  localStorage.clear()
  getRemainingQuota.mockReset().mockResolvedValue(quota)
  saveQuotaExclusions.mockReset().mockResolvedValue({ excluded_user_ids: [7, 9] })
})

describe('UserRemainingQuotaSummary', () => {
  it('loads the server list, saves changes and restores them in a new page without browser storage', async () => {
    const wrapper = mount(UserRemainingQuotaSummary, options)
    await flushPromises()
    expect(wrapper.text()).toContain('60.00$')
    expect(wrapper.text()).toContain('10.00$')
    expect(wrapper.text()).toContain('50.00$')
    expect(wrapper.text()).toContain('admin.users.remainingQuota.unlimited')
    await wrapper.find('button').trigger('click')
    expect(wrapper.get('[data-test="selected"]').text()).toContain('3')
    await wrapper.get('[data-test="select-user"]').trigger('click')
    getRemainingQuota.mockResolvedValue({ ...quota, excluded_user_ids: [7, 9] })
    await wrapper.get('.btn-primary').trigger('click')
    await flushPromises()
    expect(saveQuotaExclusions).toHaveBeenCalledWith([7, 9])
    expect(getRemainingQuota).toHaveBeenLastCalledWith(expect.any(AbortSignal))
    wrapper.unmount()
    localStorage.clear()
    const freshPage = mount(UserRemainingQuotaSummary, options)
    await flushPromises()
    await freshPage.find('button').trigger('click')
    expect(freshPage.get('[data-test="selected"]').text()).toContain('7')
    expect(freshPage.get('[data-test="selected"]').text()).toContain('9')
    freshPage.unmount()
  })

  it('keeps edits and the dialog open when saving fails, prevents duplicate writes, and retries', async () => {
    let rejectSave!: (error: Error) => void
    saveQuotaExclusions.mockImplementationOnce(() => new Promise((_, reject) => { rejectSave = reject }))
    const wrapper = mount(UserRemainingQuotaSummary, options)
    await flushPromises()
    await wrapper.find('button').trigger('click')
    await wrapper.get('[data-test="select-user"]').trigger('click')
    await wrapper.get('.btn-primary').trigger('click')
    expect(wrapper.get('.btn-primary').attributes('disabled')).toBeDefined()
    await wrapper.get('.btn-primary').trigger('click')
    expect(saveQuotaExclusions).toHaveBeenCalledTimes(1)
    rejectSave(new Error('offline'))
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('admin.users.remainingQuota.storageFailed')
    expect(wrapper.get('[data-test="selected"]').text()).toContain('7')
    expect(getRemainingQuota).toHaveBeenCalledTimes(1)
    await wrapper.get('.btn-primary').trigger('click')
    await flushPromises()
    expect(saveQuotaExclusions).toHaveBeenCalledTimes(2)
    expect(wrapper.find('.btn-primary').exists()).toBe(false)
    wrapper.unmount()
  })

  it('saves an empty list to clear exclusions', async () => {
    const wrapper = mount(UserRemainingQuotaSummary, options)
    await flushPromises()
    await wrapper.find('button').trigger('click')
    await wrapper.get('[data-test="clear"]').trigger('click')
    await wrapper.get('.btn-primary').trigger('click')
    await flushPromises()
    expect(saveQuotaExclusions).toHaveBeenCalledWith([])
    wrapper.unmount()
  })

  it('does not turn failed reads into zero totals or allow overwriting an unknown server list', async () => {
    getRemainingQuota.mockRejectedValueOnce(new Error('offline'))
    const wrapper = mount(UserRemainingQuotaSummary, options)
    await flushPromises()
    expect(wrapper.text()).toContain('admin.users.remainingQuota.failed')
    expect(wrapper.text()).not.toContain('0.00$')
    expect(wrapper.find('button').attributes('disabled')).toBeDefined()
    await wrapper.findAll('button')[1].trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('60.00$')
    wrapper.unmount()
  })

  it('ignores stale totals and exclusion lists after refresh', async () => {
    let resolveOld!: (value: typeof quota) => void
    getRemainingQuota.mockImplementationOnce(() => new Promise((resolve) => { resolveOld = resolve }))
    const wrapper = mount(UserRemainingQuotaSummary, options)
    const oldSignal = getRemainingQuota.mock.calls[0][0] as AbortSignal
    await wrapper.setProps({ refreshKey: 1 })
    await flushPromises()
    resolveOld({ ...quota, total: 999, excluded_user_ids: [99] })
    await flushPromises()
    expect(oldSignal.aborted).toBe(true)
    expect(wrapper.text()).toContain('60.00$')
    expect(wrapper.text()).not.toContain('999.00$')
    await wrapper.find('button').trigger('click')
    expect(wrapper.get('[data-test="selected"]').text()).toContain('3')
    expect(wrapper.get('[data-test="selected"]').text()).not.toContain('99')
    wrapper.unmount()
  })
})
