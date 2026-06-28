import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import ImageGenerateForm from '../ImageGenerateForm.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

function mountForm(props = {}) {
  return mount(ImageGenerateForm, { props })
}

describe('ImageGenerateForm', () => {
  it('does not render the n selector', () => {
    const wrapper = mountForm()
    expect(wrapper.find('#image-generate-n').exists()).toBe(false)
  })

  it('submits n as 1 and omits response_format', async () => {
    const wrapper = mountForm({
      initialValues: {
        prompt: 'old prompt',
        n: 4,
      },
    })

    await wrapper.find('[data-testid="image-generate-prompt"]').setValue('new prompt')
    await wrapper.find('[data-testid="image-generate-submit"]').trigger('click')

    const submit = wrapper.emitted('submit')?.[0]?.[0] as Record<string, unknown>
    expect(submit.n).toBe(1)
    expect(submit.prompt).toBe('new prompt')
    expect(submit).not.toHaveProperty('response_format')
  })

  it('falls back to auto size when switching from gpt-image-2 large size to gpt-image-1', async () => {
    const wrapper = mountForm({
      initialValues: {
        model: 'gpt-image-2',
        size: '3840x2160',
      },
    })

    await wrapper.find('#image-generate-model').setValue('gpt-image-1')
    expect((wrapper.find('#image-generate-size').element as HTMLSelectElement).value).toBe('auto')
  })

  it('normalizes incompatible initial values on mount', () => {
    const wrapper = mountForm({
      initialValues: {
        model: 'gpt-image-1',
        size: '3840x2160',
        background: 'transparent',
        output_format: 'jpeg',
        n: 4,
      },
    })

    expect((wrapper.find('#image-generate-size').element as HTMLSelectElement).value).toBe('auto')
    expect((wrapper.find('#image-generate-output-format').element as HTMLSelectElement).value).toBe('png')
  })

  it('clears a stale custom size error after switching back to a preset size', async () => {
    const wrapper = mountForm({
      initialValues: {
        prompt: 'new prompt',
      },
    })

    await wrapper.find('#image-generate-size').setValue('custom')
    await wrapper.find('[data-testid="image-generate-custom-size"]').setValue('2050x1152')
    await wrapper.find('[data-testid="image-generate-submit"]').trigger('click')
    expect(wrapper.text()).toContain('images.forms.generate.customSizeMultipleOf16')

    await wrapper.find('#image-generate-size').setValue('auto')
    expect(wrapper.text()).not.toContain('images.forms.generate.customSizeMultipleOf16')
  })

  it('keeps custom size mode when another option changes while custom value is invalid', async () => {
    const wrapper = mountForm()

    await wrapper.find('#image-generate-size').setValue('custom')
    await wrapper.find('[data-testid="image-generate-custom-size"]').setValue('2050x1152')
    await wrapper.find('#image-generate-background').setValue('transparent')

    expect((wrapper.find('#image-generate-size').element as HTMLSelectElement).value).toBe('custom')
    expect(wrapper.find('[data-testid="image-generate-custom-size"]').exists()).toBe(true)
  })

  it('does not clear prompt errors when custom size input becomes valid', async () => {
    const wrapper = mountForm()

    await wrapper.find('[data-testid="image-generate-submit"]').trigger('click')
    expect(wrapper.text()).toContain('images.forms.generate.promptRequired')

    await wrapper.find('#image-generate-size').setValue('custom')
    await wrapper.find('[data-testid="image-generate-custom-size"]').setValue('3072x1728')
    expect(wrapper.text()).toContain('images.forms.generate.promptRequired')
  })

  it('replays a valid custom initial size as custom UI and submits the real size', async () => {
    const wrapper = mountForm({
      initialValues: {
        prompt: 'wide prompt',
        model: 'gpt-image-2',
        size: '3072x1728',
      },
    })

    expect((wrapper.find('#image-generate-size').element as HTMLSelectElement).value).toBe('custom')
    expect((wrapper.find('[data-testid="image-generate-custom-size"]').element as HTMLInputElement).value).toBe('3072x1728')

    await wrapper.find('[data-testid="image-generate-submit"]').trigger('click')
    const submit = wrapper.emitted('submit')?.[0]?.[0] as Record<string, unknown>
    expect(submit.size).toBe('3072x1728')
  })

  it('normalizes unknown initial model before deciding whether a custom size should show', async () => {
    const wrapper = mountForm({
      initialValues: {
        prompt: 'legacy custom prompt',
        model: 'legacy-image-model',
        size: ' 3072x1728 ',
      },
    })

    expect((wrapper.find('#image-generate-model').element as HTMLSelectElement).value).toBe('gpt-image-2')
    expect((wrapper.find('#image-generate-size').element as HTMLSelectElement).value).toBe('custom')
    expect((wrapper.find('[data-testid="image-generate-custom-size"]').element as HTMLInputElement).value).toBe('3072x1728')

    await wrapper.find('[data-testid="image-generate-submit"]').trigger('click')
    const submit = wrapper.emitted('submit')?.[0]?.[0] as Record<string, unknown>
    expect(submit.model).toBe('gpt-image-2')
    expect(submit.size).toBe('3072x1728')
  })

  it('normalizes whitespace-padded preset initial sizes without entering custom mode', () => {
    const wrapper = mountForm({
      initialValues: {
        model: 'gpt-image-2',
        size: ' 1024x1024 ',
      },
    })

    expect((wrapper.find('#image-generate-size').element as HTMLSelectElement).value).toBe('1024x1024')
    expect(wrapper.find('[data-testid="image-generate-custom-size"]').exists()).toBe(false)
  })

  it('shows a notice when transparent background changes jpeg output to png', async () => {
    const wrapper = mountForm({
      initialValues: {
        background: 'transparent',
        output_format: 'jpeg',
      },
    })

    expect((wrapper.find('#image-generate-output-format').element as HTMLSelectElement).value).toBe('png')
    expect(wrapper.text()).toContain('images.forms.generate.transparentFormatAdjusted')
  })
})
