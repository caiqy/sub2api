import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ImagesView from '../ImagesView.vue'
import ImageEditForm from '@/components/user/images/ImageEditForm.vue'
import Pagination from '@/components/common/Pagination.vue'
import type { ApiKey, ImageGatewayResponse, ImageHistoryDetail, ImageHistoryListItem } from '@/types'
import type { ImageFetchOptions } from '@/api/images'

const { listKeys, listModels, generate, edit, listHistory, getHistoryDetail, readImage } = vi.hoisted(() => ({
  listKeys: vi.fn(), listModels: vi.fn(), generate: vi.fn(), edit: vi.fn(), listHistory: vi.fn(), getHistoryDetail: vi.fn(), readImage: vi.fn(),
}))
vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { template: '<main><slot /></main>' } }))
vi.mock('@/api', () => ({ keysAPI: { list: listKeys }, imagesAPI: { listModels, generate, edit, listHistory, getHistoryDetail } }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/utils/imageResult', async importOriginal => ({ ...await importOriginal<typeof import('@/utils/imageResult')>(), imageToFile: readImage }))
// File decoding has its own checks; these page tests exercise the handoff and request contract.
vi.mock('@/utils/imageFileValidation', () => ({ validateSourceImage: async () => ({ width: 1, height: 1 }), validateMaskImage: async () => {} }))

const model = 'gpt-image-2.5-flare'
const png = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+j6AAAAABJRU5ErkJggg=='
const response: ImageGatewayResponse = { data: [{ b64_json: png }] }
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((r, j) => { resolve = r; reject = j })
  return { promise, resolve, reject }
}
const key = (id: number) => ({ id, key: `platform-key-${id}`, name: `Key ${id}`, status: 'active', created_at: '', updated_at: '', user_id: 1 }) as ApiKey
const historyItem: ImageHistoryListItem = { id: 7, api_key_id: 1, mode: 'edit', status: 'success', model, image_count: 1, actual_cost: 0.1, prompt: 'edit old photo', created_at: '2026-10-09T12:00:00Z', summary_available: true }
const historyDetail: ImageHistoryDetail = { ...historyItem, n: 1, had_source_image: true, had_mask: true, images: [{ index: 0, data_url: `data:image/png;base64,${png}` }], replay: { mode: 'edit', model, prompt: 'edit old photo', n: 1, size: '1024x1024', requires_source_image_upload: true, requires_mask_upload: true } }
let wrapper: VueWrapper
async function openPage() {
  wrapper = mount(ImagesView, { attachTo: document.body, global: { stubs: { AppLayout: { template: '<main><slot /></main>' } } } })
  await flushPromises()
  return wrapper
}
async function submitPrompt(prompt = 'make a poster') {
  await wrapper.get('#image-generate-prompt').setValue(prompt)
  await wrapper.get('[data-testid="image-generate-form"]').trigger('submit')
  await flushPromises()
}

beforeEach(() => {
  vi.resetAllMocks()
  listKeys.mockResolvedValue({ items: [key(1), key(2)], total: 2, page: 1, pages: 1, page_size: 100 })
  listModels.mockResolvedValue([model, 'gpt-image-1.5'])
  generate.mockResolvedValue(response)
  edit.mockResolvedValue(response)
  listHistory.mockResolvedValue({ items: [historyItem], total: 41, page: 1, page_size: 20, pages: 3 })
  getHistoryDetail.mockResolvedValue(historyDetail)
  readImage.mockResolvedValue(new File(['checked image'], 'result.png', { type: 'image/png' }))
  Object.defineProperty(URL, 'createObjectURL', { configurable: true, value: vi.fn(() => 'blob:preview') })
  Object.defineProperty(URL, 'revokeObjectURL', { configurable: true, value: vi.fn() })
})
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = '' })

describe('ImagesView workbench', () => {
  it('loads current-key models, defaults size to auto, and keeps both drafts across keyboard tabs', async () => {
    await openPage()
    expect(listModels).toHaveBeenCalledWith('platform-key-1', expect.objectContaining({ signal: expect.any(AbortSignal) }))
    expect(wrapper.get<HTMLSelectElement>('#image-generate-size').element.value).toBe('auto')
    expect(wrapper.get<HTMLSelectElement>('#image-generate-model').element.value).toBe(model)
    expect(wrapper.text()).not.toContain('images.forms.generate.parametersAdjusted')
    expect(listHistory).not.toHaveBeenCalled()
    await wrapper.get('#image-generate-prompt').setValue('generation draft')
    await wrapper.get('[data-testid="images-tab-generate"]').trigger('keydown', { key: 'ArrowRight' })
    expect(wrapper.get('[data-testid="images-tab-edit"]').attributes('aria-selected')).toBe('true')
    await wrapper.get('#image-edit-prompt').setValue('editing draft')
    await wrapper.get('[data-testid="images-tab-edit"]').trigger('keydown', { key: 'Home' })
    expect(wrapper.get<HTMLTextAreaElement>('#image-generate-prompt').element.value).toBe('generation draft')
    await wrapper.get('[data-testid="images-tab-edit"]').trigger('click')
    expect(wrapper.get<HTMLTextAreaElement>('#image-edit-prompt').element.value).toBe('editing draft')
  })

  it('pages platform keys instead of silently stopping at the first 100', async () => {
    listKeys.mockResolvedValueOnce({ items: [key(1)], total: 101, page: 1, pages: 2 })
      .mockResolvedValueOnce({ items: [key(101)], total: 101, page: 2, pages: 2 })
    await openPage()
    await wrapper.get('[data-testid="image-keys-next"]').trigger('click')
    await flushPromises()
    expect(listKeys).toHaveBeenLastCalledWith(2, 100, { sort_by: 'created_at', sort_order: 'asc' }, expect.objectContaining({ signal: expect.any(AbortSignal) }))
    expect(listModels).toHaveBeenLastCalledWith('platform-key-101', expect.anything())
  })

  it('discards stale models when the selected key changes', async () => {
    const old = deferred<string[]>()
    listModels.mockReturnValueOnce(old.promise).mockResolvedValueOnce(['gpt-image-1.5'])
    await openPage()
    const oldSignal = listModels.mock.calls[0][1].signal as AbortSignal
    await wrapper.get('#image-api-key').setValue('2')
    await flushPromises()
    old.resolve([model])
    await flushPromises()
    expect(oldSignal.aborted).toBe(true)
    expect(wrapper.get('#image-generate-model').text()).toContain('gpt-image-1.5')
    expect(wrapper.get('#image-generate-model').text()).not.toContain(model)
  })

  it('blocks submission for no visible models and supports model-load retry', async () => {
    listModels.mockResolvedValueOnce([])
    await openPage()
    expect(wrapper.text()).toContain('images.models.empty')
    await submitPrompt()
    expect(generate).not.toHaveBeenCalled()
    wrapper.unmount()
    listModels.mockRejectedValueOnce(new Error('models unavailable')).mockResolvedValueOnce([model])
    await openPage()
    expect(wrapper.get('[data-testid="image-generate-submit"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-testid="image-models-retry"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="image-generate-submit"]').attributes('disabled')).toBeUndefined()
  })

  it('shows a draft, stops only waiting, and ignores late callbacks without losing input', async () => {
    const pending = deferred<ImageGatewayResponse>()
    generate.mockReturnValueOnce(pending.promise)
    await openPage()
    await submitPrompt('retained input')
    const options = generate.mock.calls[0][2] as ImageFetchOptions
    options.onPartial?.({ b64_json: png })
    await flushPromises()
    expect(wrapper.find('[data-testid="image-partial-preview"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="image-preview-edit-0"]').exists()).toBe(false)
    await wrapper.get('[data-testid="image-stop-waiting"]').trigger('click')
    expect(options.signal?.aborted).toBe(true)
    options.onCompleted?.({ b64_json: png })
    pending.resolve(response)
    await flushPromises()
    expect(wrapper.text()).toContain('images.results.states.stopped')
    expect(wrapper.find('[data-testid="image-result-grid"]').exists()).toBe(false)
    expect(wrapper.get<HTMLTextAreaElement>('#image-generate-prompt').element.value).toBe('retained input')
    expect(generate).toHaveBeenCalledTimes(1)
  })

  it('retains a completed image when a stream later fails', async () => {
    const pending = deferred<ImageGatewayResponse>()
    generate.mockReturnValueOnce(pending.promise)
    await openPage()
    await submitPrompt()
    ;(generate.mock.calls[0][2] as ImageFetchOptions).onCompleted?.({ b64_json: png })
    pending.reject(new Error('connection interrupted'))
    await flushPromises()
    expect(wrapper.find('[data-testid="image-result-preview-0"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('connection interrupted')
    expect(wrapper.text()).toContain('images.results.states.error')
  })

  it('reads a final image into editing without submitting it, with size remaining auto', async () => {
    await openPage()
    await submitPrompt()
    await wrapper.get('[data-testid="image-preview-edit-0"]').trigger('click')
    await flushPromises()
    expect(readImage).toHaveBeenCalledWith(`data:image/png;base64,${png}`, expect.any(AbortSignal))
    expect(wrapper.get('[data-testid="images-tab-edit"]').attributes('aria-selected')).toBe('true')
    expect(wrapper.get<HTMLSelectElement>('#image-edit-size').element.value).toBe('auto')
    expect(wrapper.findComponent(ImageEditForm).props('initialFiles')).toHaveLength(1)
    expect(edit).not.toHaveBeenCalled()
    await wrapper.get('#image-edit-prompt').setValue('make the background pale green')
    await wrapper.get('[data-testid="image-edit-form"]').trigger('submit')
    await flushPromises()
    const payload = edit.mock.calls[0][0] as FormData
    expect(payload.getAll('image[]')).toHaveLength(1)
    expect(payload.get('prompt')).toBe('make the background pale green')
    expect(payload.get('size')).toBe('auto')
    expect(payload.get('stream')).toBe('true')
  })

  it('explains an unreadable URL and does not invent an edit source or submit', async () => {
    readImage.mockRejectedValueOnce(new TypeError('CORS'))
    await openPage()
    await submitPrompt()
    await wrapper.get('[data-testid="image-preview-edit-0"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('images.results.readFailed')
    expect(wrapper.get('[data-testid="images-tab-generate"]').attributes('aria-selected')).toBe('true')
    expect(edit).not.toHaveBeenCalled()
  })

  it('loads history only when expanded, caches reopening, and pages or filters without fetching details', async () => {
    await openPage()
    await submitPrompt()
    expect(listHistory).not.toHaveBeenCalled()
    await wrapper.get('[data-testid="images-history-toggle"]').trigger('click')
    await flushPromises()
    expect(listHistory).toHaveBeenCalledTimes(1)
    expect(getHistoryDetail).not.toHaveBeenCalled()
    await wrapper.get('[data-testid="images-history-toggle"]').trigger('click')
    await wrapper.get('[data-testid="images-history-toggle"]').trigger('click')
    await flushPromises()
    expect(listHistory).toHaveBeenCalledTimes(1)
    wrapper.findComponent(Pagination).vm.$emit('update:page', 2)
    await flushPromises()
    expect(listHistory).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2, page_size: 20 }))
    await wrapper.get('[data-testid="image-history-mode-filter"]').setValue('edit')
    await wrapper.get('[data-testid="image-history-status-filter"]').setValue('error')
    await flushPromises()
    expect(listHistory).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, tab: 'edit', status: 'error' }))
    expect(getHistoryDetail).not.toHaveBeenCalled()
  })

  it('replays edit parameters with an explicit reupload notice and no hidden source file', async () => {
    await openPage()
    await wrapper.get('[data-testid="images-history-toggle"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-list-item-7"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-replay"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('#image-edit-prompt').element).toHaveProperty('value', 'edit old photo')
    expect(wrapper.text()).toContain('images.history.replayEditNotice')
    expect(wrapper.findComponent(ImageEditForm).props('initialFiles')).toEqual([])
    expect(edit).not.toHaveBeenCalled()
  })
})
