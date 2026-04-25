import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import ImageEditForm from '../ImageEditForm.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

function mountForm(props = {}) {
  return mount(ImageEditForm, { props })
}

function makeFile(name = 'source.png', type = 'image/png') {
  return new File(['image'], name, { type })
}

describe('ImageEditForm', () => {
  it('does not render the n selector', () => {
    const wrapper = mountForm()
    expect(wrapper.find('#image-edit-n').exists()).toBe(false)
  })

  it('submits FormData with n fixed to 1', async () => {
    const wrapper = mountForm({
      initialValues: {
        prompt: 'edit prompt',
        n: 4,
      },
    })

    const input = wrapper.find('[data-testid="image-edit-source-input"]')
    Object.defineProperty(input.element, 'files', {
      value: [makeFile()],
      configurable: true,
    })
    await input.trigger('change')
    await wrapper.find('[data-testid="image-edit-submit"]').trigger('click')

    const payload = wrapper.emitted('submit')?.[0]?.[0] as FormData
    expect(payload.get('n')).toBe('1')
  })

  it('normalizes incompatible initial values on mount', () => {
    const wrapper = mountForm({
      initialValues: {
        model: 'gpt-image-1.5',
        size: '3840x2160',
        background: 'transparent',
        output_format: 'jpeg',
        n: 3,
      },
    })

    expect((wrapper.find('#image-edit-size').element as HTMLSelectElement).value).toBe('auto')
    expect((wrapper.find('#image-edit-output-format').element as HTMLSelectElement).value).toBe('png')
  })

  it('clears a stale custom size error after switching back to a preset size', async () => {
    const wrapper = mountForm({
      initialValues: {
        prompt: 'edit prompt',
      },
    })

    await wrapper.find('#image-edit-size').setValue('custom')
    await wrapper.find('[data-testid="image-edit-custom-size"]').setValue('2050x1152')
    await wrapper.find('[data-testid="image-edit-submit"]').trigger('click')
    expect(wrapper.text()).toContain('images.forms.generate.customSizeMultipleOf16')

    await wrapper.find('#image-edit-size').setValue('auto')
    expect(wrapper.text()).not.toContain('images.forms.generate.customSizeMultipleOf16')
  })

  it('keeps custom size mode when another option changes while custom value is invalid', async () => {
    const wrapper = mountForm()

    await wrapper.find('#image-edit-size').setValue('custom')
    await wrapper.find('[data-testid="image-edit-custom-size"]').setValue('2050x1152')
    await wrapper.find('#image-edit-background').setValue('transparent')

    expect((wrapper.find('#image-edit-size').element as HTMLSelectElement).value).toBe('custom')
    expect(wrapper.find('[data-testid="image-edit-custom-size"]').exists()).toBe(true)
  })

  it('does not clear prompt errors when custom size input or source image changes', async () => {
    const wrapper = mountForm()

    await wrapper.find('[data-testid="image-edit-submit"]').trigger('click')
    expect(wrapper.text()).toContain('images.forms.generate.promptRequired')

    await wrapper.find('#image-edit-size').setValue('custom')
    await wrapper.find('[data-testid="image-edit-custom-size"]').setValue('3072x1728')
    expect(wrapper.text()).toContain('images.forms.generate.promptRequired')

    const input = wrapper.find('[data-testid="image-edit-source-input"]')
    Object.defineProperty(input.element, 'files', {
      value: [makeFile()],
      configurable: true,
    })
    await input.trigger('change')
    expect(wrapper.text()).toContain('images.forms.generate.promptRequired')
  })

  it('replays a valid custom initial size as custom UI and submits the real size', async () => {
    const wrapper = mountForm({
      initialValues: {
        prompt: 'edit wide prompt',
        model: 'gpt-image-2',
        size: '3072x1728',
      },
    })

    expect((wrapper.find('#image-edit-size').element as HTMLSelectElement).value).toBe('custom')
    expect((wrapper.find('[data-testid="image-edit-custom-size"]').element as HTMLInputElement).value).toBe('3072x1728')

    const input = wrapper.find('[data-testid="image-edit-source-input"]')
    Object.defineProperty(input.element, 'files', {
      value: [makeFile()],
      configurable: true,
    })
    await input.trigger('change')
    await wrapper.find('[data-testid="image-edit-submit"]').trigger('click')

    const payload = wrapper.emitted('submit')?.[0]?.[0] as FormData
    expect(payload.get('size')).toBe('3072x1728')
  })

  it('normalizes unknown initial model before deciding whether a custom size should show', async () => {
    const wrapper = mountForm({
      initialValues: {
        prompt: 'legacy edit prompt',
        model: 'legacy-image-model',
        size: ' 3072x1728 ',
      },
    })

    expect((wrapper.find('#image-edit-model').element as HTMLSelectElement).value).toBe('gpt-image-2')
    expect((wrapper.find('#image-edit-size').element as HTMLSelectElement).value).toBe('custom')
    expect((wrapper.find('[data-testid="image-edit-custom-size"]').element as HTMLInputElement).value).toBe('3072x1728')

    const input = wrapper.find('[data-testid="image-edit-source-input"]')
    Object.defineProperty(input.element, 'files', {
      value: [makeFile()],
      configurable: true,
    })
    await input.trigger('change')
    await wrapper.find('[data-testid="image-edit-submit"]').trigger('click')

    const payload = wrapper.emitted('submit')?.[0]?.[0] as FormData
    expect(payload.get('model')).toBe('gpt-image-2')
    expect(payload.get('size')).toBe('3072x1728')
  })

  it('normalizes whitespace-padded preset initial sizes without entering custom mode', () => {
    const wrapper = mountForm({
      initialValues: {
        model: 'gpt-image-2',
        size: ' 1024x1024 ',
      },
    })

    expect((wrapper.find('#image-edit-size').element as HTMLSelectElement).value).toBe('1024x1024')
    expect(wrapper.find('[data-testid="image-edit-custom-size"]').exists()).toBe(false)
  })

  it('shows a notice when transparent background changes jpeg output to png', () => {
    const wrapper = mountForm({
      initialValues: {
        background: 'transparent',
        output_format: 'jpeg',
      },
    })

    expect((wrapper.find('#image-edit-output-format').element as HTMLSelectElement).value).toBe('png')
    expect(wrapper.text()).toContain('images.forms.generate.transparentFormatAdjusted')
  })
})
