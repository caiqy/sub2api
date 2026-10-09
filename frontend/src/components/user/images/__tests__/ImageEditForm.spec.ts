import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ImageEditForm from '../ImageEditForm.vue'
const { validateSourceImage, validateMaskImage } = vi.hoisted(() => ({ validateSourceImage: vi.fn(), validateMaskImage: vi.fn() }))
vi.mock('@/utils/imageFileValidation', () => ({ validateSourceImage, validateMaskImage }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
const file = (name = 'source.png') => new File(['image'], name, { type: 'image/png' })
const revoke = vi.fn()
beforeEach(() => {
  validateSourceImage.mockReset().mockResolvedValue({ width: 1024, height: 1024 })
  validateMaskImage.mockReset().mockResolvedValue({ width: 1024, height: 1024 })
  let id = 0
  vi.stubGlobal('URL', { createObjectURL: vi.fn(() => `blob:preview-${++id}`), revokeObjectURL: revoke })
  revoke.mockReset()
})
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals() })
async function choose(wrapper: ReturnType<typeof mount>, selector: string, files: File[]) {
  const input = wrapper.find(selector)
  Object.defineProperty(input.element, 'files', { value: files, configurable: true })
  await input.trigger('change'); await flushPromises()
}
describe('edit reference files', () => {
  it('validates initial reused files, sends multiple image[] and revokes previews', async () => {
    const first = file(), second = file('other.png')
    const wrapper = mount(ImageEditForm, { props: { models: ['gpt-image-2'], initialValues: { prompt: 'edit', n: 4 }, initialFiles: [first, second] } })
    await flushPromises()
    expect(validateSourceImage).toHaveBeenCalledTimes(2)
    await wrapper.find('form').trigger('submit')
    const payload = wrapper.emitted('submit')?.[0]?.[0] as FormData
    expect(payload.getAll('image[]')).toEqual([first, second])
    expect(payload.get('n')).toBe('1')
    wrapper.unmount(); expect(revoke).toHaveBeenCalledTimes(2)
  })
  it('rejects invalid initial files and the 17th reference', async () => {
    validateSourceImage.mockRejectedValueOnce(new Error('images.forms.edit.sourceImageInvalid'))
    const wrapper = mount(ImageEditForm, { props: { models: ['gpt-image-2'], initialValues: { prompt: 'x' }, initialFiles: [file()] } })
    await flushPromises()
    expect(wrapper.text()).toContain('images.forms.edit.sourceImageInvalid')
    await choose(wrapper, '#image-edit-source-input', Array.from({ length: 17 }, () => file()))
    expect(wrapper.text()).toContain('images.forms.edit.sourceImageLimit')
    expect(wrapper.emitted('submit')).toBeUndefined()
    wrapper.unmount()
  })
  it('rechecks mask against the new first image and blocks dimension mismatch', async () => {
    const wrapper = mount(ImageEditForm, { props: { models: ['gpt-image-2'], initialValues: { prompt: 'x' }, initialFiles: [file(), file('second.png')] } })
    await flushPromises()
    await choose(wrapper, '#image-edit-mask-input', [file('mask.png')])
    expect(validateMaskImage).toHaveBeenCalledWith(expect.any(File), { width: 1024, height: 1024 }, expect.any(AbortSignal))
    validateMaskImage.mockRejectedValueOnce(new Error('images.forms.edit.maskDimensions'))
    await wrapper.find('li button').trigger('click'); await flushPromises()
    expect(validateMaskImage).toHaveBeenCalledTimes(2)
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('submit')).toBeUndefined()
    expect(wrapper.text()).toContain('images.forms.edit.maskDimensions')
    wrapper.unmount()
  })
})
