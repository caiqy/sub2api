import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import InputContextLimitSettings from '../InputContextLimitSettings.vue'
import type { AdminGroup } from '@/types'

const api = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
}))

const groupsApi = vi.hoisted(() => ({
  getAllIncludingInactive: vi.fn(),
}))

vi.mock('@/api/admin/settings', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/admin/settings')>()
  return {
    ...actual,
    inputContextLimitAPI: api,
    getInputContextLimitSettings: api.get,
    updateInputContextLimitSettings: api.put,
  }
})

vi.mock('@/api/admin/groups', () => ({
  getAllIncludingInactive: groupsApi.getAllIncludingInactive,
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (params) {
          return `${key}:${JSON.stringify(params)}`
        }
        return key
      },
      tm: (key: string) => {
        if (key === 'admin.settings.inputContextLimit.noticeItems') {
          return ['rule1', 'rule2', 'rule3', 'rule4', 'rule5', 'rule6']
        }
        return []
      },
    }),
  }
})

enableAutoUnmount(afterEach)

const mockGroups: AdminGroup[] = [
  {
    id: 1,
    name: 'Default Group',
    platform: 'openai',
    description: '',
    rate_multiplier: 1,
    is_active: true,
    is_default: true,
  } as AdminGroup,
  {
    id: 2,
    name: 'Claude Special',
    platform: 'anthropic',
    description: '',
    rate_multiplier: 1.5,
    is_active: true,
    is_default: false,
  } as AdminGroup,
  {
    id: 3,
    name: 'Gemini Pro',
    platform: 'gemini',
    description: '',
    rate_multiplier: 1,
    is_active: false,
    is_default: false,
  } as AdminGroup,
]

const initialSettings = {
  enabled: true,
  group_limits: {
    '1': 64000,
    '2': 128000,
  },
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.resetAllMocks()
  api.get.mockResolvedValue(initialSettings)
  api.put.mockImplementation(async (payload) => payload)
  groupsApi.getAllIncludingInactive.mockResolvedValue(mockGroups)
})

describe('InputContextLimitSettings.vue', () => {
  it('loads and displays settings and groups correctly', async () => {
    const wrapper = mount(InputContextLimitSettings)
    await flushPromises()

    expect(api.get).toHaveBeenCalledOnce()
    expect(groupsApi.getAllIncludingInactive).toHaveBeenCalledOnce()

    const toggle = wrapper.find('[data-testid="input-context-limit-toggle"]')
    expect(toggle.exists()).toBe(true)

    const input1 = wrapper.find('[data-testid="input-context-limit-input-1"]')
    const input2 = wrapper.find('[data-testid="input-context-limit-input-2"]')
    const input3 = wrapper.find('[data-testid="input-context-limit-input-3"]')

    expect((input1.element as HTMLInputElement).value).toBe('64000')
    expect((input2.element as HTMLInputElement).value).toBe('128000')
    expect((input3.element as HTMLInputElement).value).toBe('')
  })

  it('prohibits save and shows retry button when load fails', async () => {
    api.get.mockRejectedValueOnce(new Error('Network error'))
    const wrapper = mount(InputContextLimitSettings)
    await flushPromises()

    expect(wrapper.text()).toContain('admin.settings.inputContextLimit.loadFailed')
    expect(wrapper.find('[data-testid="input-context-limit-save"]').attributes('disabled')).toBeDefined()

    // 点击保存不应触发 put
    await wrapper.find('[data-testid="input-context-limit-save"]').trigger('click')
    expect(api.put).not.toHaveBeenCalled()

    // 点击重试恢复
    api.get.mockResolvedValueOnce(initialSettings)
    const retryBtn = wrapper.find('button.btn-secondary')
    await retryBtn.trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="input-context-limit-save"]').attributes('disabled')).toBeUndefined()
  })

  it('rejects invalid integer values (negative, decimal, NaN, >2147483647) without making a request', async () => {
    const wrapper = mount(InputContextLimitSettings)
    await flushPromises()

    const input1 = wrapper.find('[data-testid="input-context-limit-input-1"]')
    const saveBtn = wrapper.find('[data-testid="input-context-limit-save"]')

    const invalidCases = ['-1', '64.5', 'abc', '2147483648', '1e5']
    for (const val of invalidCases) {
      await input1.setValue(val)
      await saveBtn.trigger('click')
      await flushPromises()

      expect(api.put).not.toHaveBeenCalled()
      expect(wrapper.text()).toContain('admin.settings.inputContextLimit.invalidValue')
    }
  })

  it('normalizes empty string and 0 as unlimited without failing', async () => {
    const wrapper = mount(InputContextLimitSettings)
    await flushPromises()

    const input1 = wrapper.find('[data-testid="input-context-limit-input-1"]')
    const input2 = wrapper.find('[data-testid="input-context-limit-input-2"]')
    const saveBtn = wrapper.find('[data-testid="input-context-limit-save"]')

    // 留空与设为 0
    await input1.setValue('')
    await input2.setValue('0')

    await saveBtn.trigger('click')
    await flushPromises()

    expect(api.put).toHaveBeenCalledWith({
      enabled: true,
      group_limits: {},
    })
    expect(wrapper.text()).toContain('admin.settings.inputContextLimit.saved')
  })

  it('preserves configuration and limits when toggle is disabled', async () => {
    const wrapper = mount(InputContextLimitSettings)
    await flushPromises()

    const toggle = wrapper.find('[data-testid="input-context-limit-toggle"]')
    await toggle.trigger('click')

    const saveBtn = wrapper.find('[data-testid="input-context-limit-save"]')
    await saveBtn.trigger('click')
    await flushPromises()

    expect(api.put).toHaveBeenCalledWith({
      enabled: false,
      group_limits: {
        '1': 64000,
        '2': 128000,
      },
    })
  })

  it('does not lose unfiltered group limits when saving with an active search/filter', async () => {
    const wrapper = mount(InputContextLimitSettings)
    await flushPromises()

    // 搜索只匹配 Group 2
    const searchInput = wrapper.find('input[type="search"]')
    await searchInput.setValue('Claude')

    // 此时仅 Group 2 渲染可见
    expect(wrapper.find('[data-testid="input-context-limit-input-1"]').exists()).toBe(false)
    const input2 = wrapper.find('[data-testid="input-context-limit-input-2"]')
    expect(input2.exists()).toBe(true)

    // 修改 Group 2
    await input2.setValue('200000')

    const saveBtn = wrapper.find('[data-testid="input-context-limit-save"]')
    await saveBtn.trigger('click')
    await flushPromises()

    // 验证未在当前视图显示的 Group 1（64000）仍然完整保留并提交
    expect(api.put).toHaveBeenCalledWith({
      enabled: true,
      group_limits: {
        '1': 64000,
        '2': 200000,
      },
    })
  })

  it('detects orphaned group limits and provides cleanup action', async () => {
    // 包含已删除的 group 999
    api.get.mockResolvedValueOnce({
      enabled: true,
      group_limits: {
        '1': 64000,
        '999': 50000,
      },
    })

    const wrapper = mount(InputContextLimitSettings)
    await flushPromises()

    expect(wrapper.text()).toContain('999')
    const cleanBtn = wrapper.find('button.btn-xs')
    expect(cleanBtn.exists()).toBe(true)

    await cleanBtn.trigger('click')
    await flushPromises()

    // 孤儿警告消除
    expect(wrapper.find('button.btn-xs').exists()).toBe(false)

    // 保存后不再包含 999
    const saveBtn = wrapper.find('[data-testid="input-context-limit-save"]')
    await saveBtn.trigger('click')
    await flushPromises()

    expect(api.put).toHaveBeenCalledWith({
      enabled: true,
      group_limits: {
        '1': 64000,
      },
    })
  })
})
