import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')

  const messages: Record<string, string> = {
    'admin.accounts.passthroughFields.title': '透传字段规则',
    'admin.accounts.passthroughFields.description': '仅对 API Key 类型账号生效；与自动透传能力独立',
    'admin.accounts.passthroughFields.disabledHint': '已配置规则会保留，但当前不会生效',
    'admin.accounts.passthroughFields.addRule': '新增规则',
    'admin.accounts.passthroughFields.targetHeader': 'Header',
    'admin.accounts.passthroughFields.targetBody': 'Body',
    'admin.accounts.passthroughFields.modeForward': '放行透传',
    'admin.accounts.passthroughFields.modeInject': '固定注入',
    'admin.accounts.passthroughFields.headerHint': 'Header 比较时不区分大小写',
    'admin.accounts.passthroughFields.bodyHint': '仅支持 xx.xx 形式的对象层级路径',
    'admin.accounts.passthroughFields.injectHint': '固定注入将在转发前写入上游请求',
    'admin.accounts.passthroughFields.errors.reservedKey': '保留字段不能透传',
    'common.delete': '删除'
  }

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key
    })
  }
})

import PassthroughFieldRulesEditor from '../PassthroughFieldRulesEditor.vue'

describe('PassthroughFieldRulesEditor', () => {
  it('初始空规则会同步给父层', () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: []
      }
    })

    expect(wrapper.find('[data-testid="passthrough-rule-row-0"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="passthrough-rule-key-0"]').element as HTMLInputElement).value).toBe('')
    expect(wrapper.emitted('update:rules')?.[0]?.[0]).toEqual([
      expect.objectContaining({
        target: 'header',
        mode: 'forward',
        key: '',
        value: ''
      })
    ])
  })

  it('mode=inject 时显示 value 输入框与固定注入提示', () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'inject', key: 'X-Test', value: '1' }]
      }
    })

    expect(wrapper.text()).toContain('固定注入将在转发前写入上游请求')
    expect(wrapper.find('[data-testid="passthrough-rule-value-0"]').exists()).toBe(true)
  })

  it('mode=forward 时隐藏 value 输入框', () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'forward', key: 'X-Test', value: '1' }]
      }
    })

    expect(wrapper.find('[data-testid="passthrough-rule-value-0"]').exists()).toBe(false)
  })

  it('切换 target 后显示对应 hint', async () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'forward', key: '', value: '' }]
      }
    })

    expect(wrapper.text()).toContain('Header 比较时不区分大小写')
    expect(wrapper.text()).not.toContain('仅支持 xx.xx 形式的对象层级路径')

    await wrapper.get('[data-testid="passthrough-rule-target-0"]').setValue('body')

    expect(wrapper.emitted('update:rules')?.[0]?.[0]).toEqual([
      expect.objectContaining({
        id: '1',
        target: 'body',
        mode: 'forward',
        key: '',
        value: ''
      })
    ])
    expect(wrapper.text()).toContain('仅支持 xx.xx 形式的对象层级路径')
    expect(wrapper.text()).not.toContain('Header 比较时不区分大小写')
  })

  it('切换总开关时会发出 update:enabled', async () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'forward', key: 'X-Test', value: '' }]
      }
    })

    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(false)

    expect(wrapper.emitted('update:enabled')).toEqual([[false]])
  })

  it('删除最后一条规则后仍可重新新增', async () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'forward', key: 'X-Test', value: '' }]
      }
    })

    await wrapper.get('[data-testid="passthrough-rule-delete-0"]').trigger('click')

    expect(wrapper.find('[data-testid="passthrough-rule-row-0"]').exists()).toBe(false)

    await wrapper.get('[data-testid="passthrough-add-rule"]').trigger('click')

    expect(wrapper.find('[data-testid="passthrough-rule-row-0"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="passthrough-rule-key-0"]').element as HTMLInputElement).value).toBe('')
  })

  it('总开关关闭时仍显示列表并展示 disabledHint', () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: false,
        showDisabledHint: true,
        rules: [{ id: '1', target: 'header', mode: 'forward', key: 'X-Test', value: '' }]
      }
    })

    expect(wrapper.text()).toContain('已配置规则会保留，但当前不会生效')
    expect(wrapper.find('[data-testid="passthrough-rules-section"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="passthrough-rule-row-0"]').exists()).toBe(true)
  })

  it('父层接收初始空规则后能保持父子同步', async () => {
    const Host = defineComponent({
      components: { PassthroughFieldRulesEditor },
      data() {
        return {
          enabled: true,
          rules: [] as Array<{ id: string; target: 'header' | 'body'; mode: 'forward' | 'inject'; key: string; value: string }>
        }
      },
      template: `
        <PassthroughFieldRulesEditor
          v-model:enabled="enabled"
          v-model:rules="rules"
        />
      `
    })

    const wrapper = mount(Host)

    expect((wrapper.vm as { rules: Array<unknown> }).rules).toHaveLength(1)

    await wrapper.get('[data-testid="passthrough-rule-key-0"]').setValue('X-From-Parent')

    expect((wrapper.vm as { rules: Array<{ key: string }> }).rules[0]?.key).toBe('X-From-Parent')
  })

  it('展示保留键校验错误文案', () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'forward', key: 'authorization', value: '' }]
      }
    })

    expect(wrapper.text()).toContain('保留字段不能透传')
  })
})
