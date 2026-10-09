import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ImageGenerateForm from '../ImageGenerateForm.vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const models = ['gpt-image-2.5-flare', 'gpt-image-2', 'gpt-image-1']
const mountForm = (props = {}) => mount(ImageGenerateForm, { props: { models, ...props } })
describe('generation form', () => {
  it('uses collapsed advanced controls and submits defaults with fixed n', async () => {
    const wrapper = mountForm({ initialValues: { prompt: 'fox', n: 4 } })
    expect(wrapper.find('details').attributes('open')).toBeUndefined()
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('submit')?.[0]?.[0]).toMatchObject({ prompt: 'fox', n: 1, size: 'auto' })
    expect(wrapper.find('#image-generate-n').exists()).toBe(false)
    wrapper.unmount()
  })
  it('revalidates historical parameters when switching models', async () => {
    const wrapper = mountForm({ initialValues: { prompt: 'x', model: models[0], quality: 'max', size: '3072x1728' } })
    expect(wrapper.find('#image-generate-custom-size').exists()).toBe(true)
    await wrapper.find('#image-generate-model').setValue('gpt-image-1')
    expect((wrapper.find('#image-generate-quality').element as HTMLSelectElement).value).toBe('auto')
    expect((wrapper.find('#image-generate-size').element as HTMLSelectElement).value).toBe('auto')
    wrapper.unmount()
  })
  it('rejects invalid custom dimensions, preserves sentinel and normalizes transparency', async () => {
    const wrapper = mountForm({ initialValues: { prompt: 'x', model: 'gpt-image-2' } })
    await wrapper.find('#image-generate-size').setValue('custom')
    await wrapper.find('#image-generate-custom-size').setValue('2050x1152')
    await wrapper.find('#image-generate-background').setValue('transparent')
    await wrapper.find('#image-generate-output-format').setValue('jpeg')
    expect(wrapper.find('#image-generate-custom-size').exists()).toBe(true)
    expect((wrapper.find('#image-generate-output-format').element as HTMLSelectElement).value).toBe('png')
    await wrapper.find('form').trigger('submit')
    expect(wrapper.text()).toContain('images.forms.generate.customSizeMultipleOf16')
    expect(wrapper.emitted('submit')).toBeUndefined()
    await wrapper.find('#image-generate-size').setValue('auto')
    expect(wrapper.text()).not.toContain('images.forms.generate.customSizeMultipleOf16')
    wrapper.unmount()
  })
  it('blocks empty candidates and reset clears prompt', async () => {
    const wrapper = mountForm({ models: [], initialValues: { prompt: 'x' } })
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('submit')).toBeUndefined()
    expect(wrapper.find('button').attributes('disabled')).toBeDefined()
    wrapper.vm.reset()
    await wrapper.vm.$nextTick()
    expect((wrapper.find('textarea').element as HTMLTextAreaElement).value).toBe('')
    wrapper.unmount()
  })
})
