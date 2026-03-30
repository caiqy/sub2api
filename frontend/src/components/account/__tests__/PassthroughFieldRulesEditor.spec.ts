import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import zh from '@/i18n/locales/zh'
import en from '@/i18n/locales/en'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')

  const messages: Record<string, string> = {
    'admin.accounts.passthroughFields.title': '透传字段规则',
    'admin.accounts.passthroughFields.description': '适用于所有账号类型；与自动透传能力独立',
    'admin.accounts.passthroughFields.disabledHint': '已配置规则会保留，但当前不会生效',
    'admin.accounts.passthroughFields.addRule': '新增规则',
    'admin.accounts.passthroughFields.targetHeader': 'Header',
    'admin.accounts.passthroughFields.targetBody': 'Body',
    'admin.accounts.passthroughFields.modeForward': '放行透传',
    'admin.accounts.passthroughFields.modeInject': '固定注入',
    'admin.accounts.passthroughFields.modeMap': '映射透传',
    'admin.accounts.passthroughFields.modeDelete': '删除字段',
    'admin.accounts.passthroughFields.sourceKey': '来源字段',
    'admin.accounts.passthroughFields.targetKey': '目标字段',
    'admin.accounts.passthroughFields.headerHint': 'Header 比较时不区分大小写',
    'admin.accounts.passthroughFields.bodyHint': '仅支持 xx.xx 形式的对象层级路径',
    'admin.accounts.passthroughFields.injectHint': '固定注入将在转发前写入上游请求',
    'admin.accounts.passthroughFields.mapHint': '映射透传会复制来源字段到目标字段，且不会修改原字段',
    'admin.accounts.passthroughFields.deleteHint': '删除模式会在转发前从上游请求中移除指定字段',
    'admin.accounts.passthroughFields.hiddenRulesError': '规则列表已隐藏，请重新开启后处理错误',
    'admin.accounts.passthroughFields.errors.duplicateKey': '同一目标下的字段名或路径不能重复',
    'admin.accounts.passthroughFields.errors.sourceKeyRequired': '来源字段或路径不能为空',
    'admin.accounts.passthroughFields.errors.sameSourceAndTarget': '来源字段与目标字段不能相同',
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

  it('开关关闭时隐藏规则列表与新增按钮', () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: false,
        showDisabledHint: true,
        rules: [{ id: '1', target: 'header', mode: 'forward', key: 'X-Test', source_key: '', value: '' }]
      }
    })

    expect(wrapper.text()).toContain('已配置规则会保留，但当前不会生效')
    expect(wrapper.find('[data-testid="passthrough-rules-section"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="passthrough-add-rule"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="passthrough-rule-row-0"]').exists()).toBe(false)
  })

  it('rules 为空且关闭时不显示隐藏态错误提示', () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: false,
        rules: []
      }
    })

    expect(wrapper.text()).not.toContain('规则列表已隐藏，请重新开启后处理错误')
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

  it('mode=map 时显示 source_key 输入框与 map 提示', () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'map', key: 'X-Target', source_key: 'X-Source', value: '' }]
      }
    })

    expect(wrapper.find('[data-testid="passthrough-rule-source-key-0"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="passthrough-rule-value-0"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('映射透传会复制来源字段到目标字段，且不会修改原字段')
  })

  it('关闭再开启后规则仍存在', async () => {
    const Host = defineComponent({
      components: { PassthroughFieldRulesEditor },
      data() {
        return {
          enabled: true,
          rules: [{ id: '1', target: 'header', mode: 'map', key: 'X-Target', source_key: 'X-Source', value: '' }]
        }
      },
      template: `
        <PassthroughFieldRulesEditor
          v-model:enabled="enabled"
          v-model:rules="rules"
          :show-disabled-hint="true"
        />
      `
    })

    const wrapper = mount(Host)

    await wrapper.get('[data-testid="passthrough-enabled-toggle"]').setValue(false)
    expect(wrapper.find('[data-testid="passthrough-rule-row-0"]').exists()).toBe(false)

    await wrapper.setData({ enabled: true })

    expect(wrapper.find('[data-testid="passthrough-rule-row-0"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="passthrough-rule-key-0"]').element as HTMLInputElement).value).toBe('X-Target')
    expect((wrapper.get('[data-testid="passthrough-rule-source-key-0"]').element as HTMLInputElement).value).toBe('X-Source')
  })

  it('map 下 target=body 时 source_key 与 key 都显示 body hint', async () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'map', key: 'target.path', source_key: 'source.path', value: '' }]
      }
    })

    await wrapper.get('[data-testid="passthrough-rule-target-0"]').setValue('body')

    expect(wrapper.text()).toContain('仅支持 xx.xx 形式的对象层级路径')
    expect(wrapper.text()).not.toContain('Header 比较时不区分大小写')
  })

  it('update:rules payload 会带上 source_key', async () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'map', key: 'X-Target', source_key: '', value: '' }]
      }
    })

    await wrapper.get('[data-testid="passthrough-rule-source-key-0"]').setValue('X-Source')

    expect(wrapper.emitted('update:rules')?.at(-1)?.[0]).toEqual([
      expect.objectContaining({
        id: '1',
        target: 'header',
        mode: 'map',
        key: 'X-Target',
        source_key: 'X-Source',
        value: ''
      })
    ])
  })

  it('mode=inject 与 target header/body 时仍显示既有 hint', async () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'inject', key: 'X-Test', source_key: '', value: '1' }]
      }
    })

    expect(wrapper.text()).toContain('固定注入将在转发前写入上游请求')
    expect(wrapper.text()).toContain('Header 比较时不区分大小写')

    await wrapper.get('[data-testid="passthrough-rule-target-0"]').setValue('body')

    expect(wrapper.text()).toContain('固定注入将在转发前写入上游请求')
    expect(wrapper.text()).toContain('仅支持 xx.xx 形式的对象层级路径')
  })

  it('基于真实校验结果展示 source_key_required 错误文案', () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'map', key: 'X-Target', source_key: '   ', value: '' }]
      }
    })

    expect(wrapper.text()).toContain('来源字段或路径不能为空')
  })

  it('基于真实校验结果展示 same_source_and_target 错误文案', () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'map', key: 'X-Target', source_key: 'x-target', value: '' }]
      }
    })

    expect(wrapper.text()).toContain('来源字段与目标字段不能相同')
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

  it('中英文文案包含 map 与 source_key 相关新增字段', () => {
    expect(zh.admin.accounts.passthroughFields.modeMap).toBeTruthy()
    expect(zh.admin.accounts.passthroughFields.sourceKey).toBeTruthy()
    expect(zh.admin.accounts.passthroughFields.targetKey).toBeTruthy()
    expect(zh.admin.accounts.passthroughFields.mapHint).toBe('映射透传会复制来源字段到目标字段，且不会修改原字段')
    expect(zh.admin.accounts.passthroughFields.modeDelete).toBe('删除字段')
    expect(zh.admin.accounts.passthroughFields.deleteHint).toBe('删除模式会在转发前从上游请求中移除指定字段')
    expect(zh.admin.accounts.passthroughFields.hiddenRulesError).toBeTruthy()
    expect(zh.admin.accounts.passthroughFields.errors.sourceKeyRequired).toBeTruthy()
    expect(zh.admin.accounts.passthroughFields.errors.sameSourceAndTarget).toBeTruthy()
    expect(zh.admin.accounts.passthroughFields.errors.reservedKey).toBeUndefined()

    expect(en.admin.accounts.passthroughFields.modeMap).toBeTruthy()
    expect(en.admin.accounts.passthroughFields.sourceKey).toBeTruthy()
    expect(en.admin.accounts.passthroughFields.targetKey).toBeTruthy()
    expect(en.admin.accounts.passthroughFields.mapHint).toBe('Map mode copies the source field to the target field without changing the original field')
    expect(en.admin.accounts.passthroughFields.modeDelete).toBe('Delete')
    expect(en.admin.accounts.passthroughFields.deleteHint).toBe('Delete mode removes the specified field from the upstream request before forwarding')
    expect(en.admin.accounts.passthroughFields.hiddenRulesError).toBeTruthy()
    expect(en.admin.accounts.passthroughFields.errors.sourceKeyRequired).toBeTruthy()
    expect(en.admin.accounts.passthroughFields.errors.sameSourceAndTarget).toBeTruthy()
    expect(en.admin.accounts.passthroughFields.errors.reservedKey).toBeUndefined()
  })

  it('mode=delete 时隐藏 value 和 source_key 输入框并显示 deleteHint', () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'delete', key: 'X-Remove', source_key: '', value: '' }]
      }
    })

    expect(wrapper.find('[data-testid="passthrough-rule-value-0"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="passthrough-rule-source-key-0"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('删除模式会在转发前从上游请求中移除指定字段')
  })

  it('mode=delete 下拉选项存在', () => {
    const wrapper = mount(PassthroughFieldRulesEditor, {
      props: {
        enabled: true,
        rules: [{ id: '1', target: 'header', mode: 'forward', key: '', source_key: '', value: '' }]
      }
    })

    const options = wrapper.get('[data-testid="passthrough-rule-mode-0"]').findAll('option')
    const values = options.map(o => o.attributes('value'))
    expect(values).toContain('delete')
  })
})
