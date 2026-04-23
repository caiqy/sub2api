import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ImagesView from '../ImagesView.vue'

const { edit, generate, getHistoryDetail, list, listHistory, errorSpy } = vi.hoisted(() => ({
  edit: vi.fn(),
  generate: vi.fn(),
  getHistoryDetail: vi.fn(),
  list: vi.fn(),
  listHistory: vi.fn(),
  errorSpy: vi.fn()
}))

const messages: Record<string, string> = {
  'images.badge': 'AI Images',
  'images.title': 'AI Images',
  'images.description': 'Generate, edit, and review AI image work in one place.',
  'images.tabs.generate': 'Generate',
  'images.tabs.edit': 'Edit',
  'images.tabs.history': 'History',
  'images.keySelector.label': 'API Key',
  'images.keySelector.loading': 'Loading API keys...',
  'images.keySelector.placeholder': 'Select an API key',
  'images.keySelector.empty': 'No API keys available yet',
  'images.keySelector.count': '{count} API keys loaded',
  'images.keySelector.pageHint': 'Selection uses the first API key from the current page only.',
  'images.keySelector.loadFailed': 'Failed to load API keys.',
  'images.keySelector.retry': 'Retry',
  'images.summary.selectedTab': 'Selected tab',
  'images.summary.selectedKey': 'Selected key',
  'images.summary.placeholder': 'Results update after each submission.',
  'images.panels.generate.title': 'Generate',
  'images.panels.generate.description': 'Create a fresh image from a prompt and standard gateway parameters.',
  'images.panels.edit.title': 'Edit',
  'images.panels.edit.description': 'Upload a source image, optionally add a mask, and submit multipart edits.',
  'images.panels.history.title': 'History',
  'images.panels.history.description': 'Review past image requests and replay their parameters.',
  'images.forms.generate.prompt': 'Prompt',
  'images.forms.generate.promptPlaceholder': 'Describe the image you want to create',
  'images.forms.generate.model': 'Model',
  'images.forms.generate.size': 'Size',
  'images.forms.generate.sizeHint': 'Official popular presets are shown here. GPT Image 2 also supports auto and more custom sizes that satisfy OpenAI constraints.',
  'images.forms.generate.customSize': 'Custom size',
  'images.forms.generate.customSizePlaceholder': 'e.g. 2048x1152',
  'images.forms.generate.customSizeRequired': 'Custom size is required.',
  'images.forms.generate.quality': 'Quality',
  'images.forms.generate.background': 'Background',
  'images.forms.generate.outputFormat': 'Output format',
  'images.forms.generate.moderation': 'Moderation',
  'images.forms.generate.n': 'Images',
  'images.forms.generate.submit': 'Generate image',
  'images.forms.generate.submitting': 'Generating...',
  'images.forms.generate.apiKeyRequired': 'Select an API key before submitting.',
  'images.forms.generate.promptRequired': 'Prompt is required.',
  'images.forms.edit.sourceImage': 'Source image',
  'images.forms.edit.sourceImageHint': 'PNG, WEBP, or JPEG.',
  'images.forms.edit.sourceImageInvalid': 'Source image must be an image file.',
  'images.forms.edit.maskImage': 'Mask image',
  'images.forms.edit.maskImageHint': 'Transparent areas mark editable regions.',
  'images.forms.edit.sourceImageRequired': 'Source image is required.',
  'images.forms.edit.submit': 'Edit image',
  'images.forms.edit.submitting': 'Editing...',
  'images.results.title': 'Results',
  'images.results.description': 'Latest gateway response previews render here.',
  'images.results.loading': 'Loading latest result...',
  'images.results.empty': 'Submit a generate or edit request to see results.',
  'images.results.errorTitle': 'Request failed',
  'images.results.revisedPrompt': 'Revised prompt',
  'images.history.listTitle': 'Recent requests',
  'images.history.empty': 'No image history yet.',
  'images.history.loading': 'Loading image history...',
  'images.history.loadFailed': 'Failed to load image history.',
  'images.history.retry': 'Retry',
  'images.history.detailTitle': 'History detail',
  'images.history.detailEmpty': 'Select a history record to inspect it.',
  'images.history.detailLoading': 'Loading history detail...',
  'images.history.detailLoadFailed': 'Failed to load history detail.',
  'images.history.prompt': 'Prompt',
  'images.history.parameters': 'Parameters',
  'images.history.images': 'Images',
  'images.history.status': 'Status',
  'images.history.apiKey': 'API key',
  'images.history.createdAt': 'Created at',
  'images.history.count': 'Images',
  'images.history.errorMessage': 'Error message',
  'images.history.replay': 'Replay with these settings',
  'images.history.modes.generate': 'Generate',
  'images.history.modes.edit': 'Edit',
  'images.history.statuses.success': 'Success',
  'images.history.statuses.error': 'Error',
  'images.history.replayEditNotice': 'Replay restored the edit parameters. Re-upload the source image and optional mask before submitting again.',
  'images.history.booleanYes': 'Yes',
  'images.history.booleanNo': 'No',
  'images.history.hadSourceImage': 'Source image uploaded',
  'images.history.hadMask': 'Mask uploaded'
}

vi.mock('@/api', () => ({
  keysAPI: {
    list
  },
  imagesAPI: {
    edit,
    generate,
    getHistoryDetail,
    listHistory
  }
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        const message = messages[key] ?? key
        if (!params) {
          return message
        }

        return Object.entries(params).reduce(
          (result, [paramKey, value]) => result.replace(`{${paramKey}}`, String(value)),
          message
        )
      }
    })
  }
})

const historyListItems = [
  {
    id: 31,
    api_key_id: 7,
    api_key_name: 'Vision Key A',
    api_key_masked: 'sk-***A',
    mode: 'generate',
    status: 'success',
    model: 'gpt-image-2',
    image_count: 2,
    image_size: '1536x1024',
    actual_cost: 0.24,
    created_at: '2026-04-20T10:00:00Z'
  },
  {
    id: 32,
    api_key_id: 7,
    api_key_name: 'Vision Key A',
    api_key_masked: 'sk-***A',
    mode: 'edit',
    status: 'error',
    model: 'gpt-image-2',
    image_count: 0,
    image_size: '1024x1024',
    actual_cost: 0.12,
    created_at: '2026-04-20T11:00:00Z'
  }
]

const generateHistoryDetail = {
  id: 31,
  api_key_id: 7,
  api_key_name: 'Vision Key A',
  api_key_masked: 'sk-***A',
  mode: 'generate',
  status: 'success',
  model: 'gpt-image-2',
  prompt: 'Draw a paper crane over water',
  size: '1536x1024',
  quality: 'high',
  background: 'transparent',
  output_format: 'webp',
  moderation: 'low',
  n: 2,
  had_source_image: false,
  had_mask: false,
  images: [
    {
      data_url: 'data:image/webp;base64,QUJD',
      revised_prompt: 'Draw a paper crane over water at dusk'
    }
  ],
  replay: {
    mode: 'generate',
    model: 'gpt-image-2',
    prompt: 'Draw a paper crane over water',
    size: '1536x1024',
    quality: 'high',
    background: 'transparent',
    output_format: 'webp',
    moderation: 'low',
    n: 2,
    requires_source_image_upload: false,
    requires_mask_upload: false
  },
  created_at: '2026-04-20T10:00:00Z'
}

const editHistoryDetail = {
  id: 32,
  api_key_id: 7,
  api_key_name: 'Vision Key A',
  api_key_masked: 'sk-***A',
  mode: 'edit',
  status: 'error',
  model: 'gpt-image-2',
  prompt: 'Retouch the subject',
  size: '1024x1024',
  quality: 'medium',
  background: 'auto',
  output_format: 'png',
  moderation: 'auto',
  n: 1,
  had_source_image: true,
  had_mask: true,
  images: [],
  error_message: 'Upstream timeout',
  replay: {
    mode: 'edit',
    model: 'gpt-image-2',
    prompt: 'Retouch the subject',
    size: '1024x1024',
    quality: 'medium',
    background: 'auto',
    output_format: 'png',
    moderation: 'auto',
    n: 1,
    requires_source_image_upload: true,
    requires_mask_upload: true
  },
  created_at: '2026-04-20T11:00:00Z'
}

const generateHistoryDetailWithDefaults = {
  ...generateHistoryDetail,
  replay: {
    ...generateHistoryDetail.replay,
    background: undefined,
    output_format: undefined,
    quality: undefined,
    size: undefined,
  }
}

const unsafeImageHistoryDetail = {
  ...generateHistoryDetail,
  images: [
    {
      data_url: 'javascript:alert(1)',
      revised_prompt: 'unsafe history image'
    }
  ]
}

const primaryApiKey = {
  id: 7,
  key: 'sk-vision-key-a',
  name: 'Vision Key A'
}

const secondaryApiKey = {
  id: 9,
  key: 'sk-vision-key-b',
  name: 'Vision Key B'
}

describe('ImagesView', () => {
  beforeEach(() => {
    edit.mockReset()
    generate.mockReset()
    getHistoryDetail.mockReset()
    list.mockReset()
    listHistory.mockReset()
    errorSpy.mockReset()
    edit.mockResolvedValue({ created: 1, data: [] })
    generate.mockResolvedValue({ created: 1, data: [] })
    getHistoryDetail.mockResolvedValue(null)
    list.mockResolvedValue({ items: [] })
    listHistory.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    vi.spyOn(console, 'error').mockImplementation(errorSpy)
  })

  it('renders generate, edit, and history tabs', async () => {
    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Generate')
    expect(wrapper.text()).toContain('Edit')
    expect(wrapper.text()).toContain('History')
  })

  it('shows the size guidance in both generate and edit forms', async () => {
    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Official popular presets are shown here. GPT Image 2 also supports auto and more custom sizes that satisfy OpenAI constraints.')

    await wrapper.get('[data-testid="images-tab-edit"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Official popular presets are shown here. GPT Image 2 also supports auto and more custom sizes that satisfy OpenAI constraints.')
  })

  it('loads current user api keys on mount with explicit paging and stable sorting', async () => {
    mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()

    expect(list).toHaveBeenCalledTimes(1)
    expect(list).toHaveBeenCalledWith(1, 100, { sort_by: 'created_at', sort_order: 'asc' })
  })

  it('defaults to the first api key from the current page after loading', async () => {
    list.mockResolvedValue({
      items: [
        { id: 7, name: 'Vision Key A' },
        { id: 9, name: 'Vision Key B' }
      ]
    })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()

    const select = wrapper.get('select')
    expect((select.element as HTMLSelectElement).value).toBe('7')
    expect(wrapper.text()).toContain('Vision Key A')
    expect(wrapper.text()).toContain('Selection uses the first API key from the current page only.')
  })

  it('switches the active panel when a tab is clicked', async () => {
    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()

    await wrapper.get('[data-testid="images-tab-edit"]').trigger('click')

    expect(wrapper.get('[data-testid="images-panel-edit"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Upload a source image, optionally add a mask, and submit multipart edits.')
  })

  it('loads image history when the history tab is opened', async () => {
    listHistory.mockResolvedValue({ items: historyListItems, total: 2, page: 1, page_size: 20, pages: 1 })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-history"]').trigger('click')
    await flushPromises()

    expect(listHistory).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Recent requests')
    expect(wrapper.text()).toContain('gpt-image-2')
  })

  it('loads history detail after a history record is selected', async () => {
    listHistory.mockResolvedValue({ items: historyListItems, total: 2, page: 1, page_size: 20, pages: 1 })
    getHistoryDetail.mockResolvedValue(generateHistoryDetail)

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-history"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-list-item-31"]').trigger('click')
    await flushPromises()

    expect(getHistoryDetail).toHaveBeenCalledWith(31)
    expect(wrapper.text()).toContain('Draw a paper crane over water')
    expect(wrapper.find('[data-testid="image-history-detail-image-0"]').exists()).toBe(true)
  })

  it('hides the previous detail summary and replay action while a new detail is loading', async () => {
    listHistory.mockResolvedValue({ items: historyListItems, total: 2, page: 1, page_size: 20, pages: 1 })
    getHistoryDetail
      .mockResolvedValueOnce(generateHistoryDetail)
      .mockImplementationOnce(() => new Promise(() => {}))

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-history"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-list-item-31"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Generate · gpt-image-2')
    expect(wrapper.find('[data-testid="image-history-replay"]').exists()).toBe(true)

    await wrapper.get('[data-testid="image-history-list-item-32"]').trigger('click')

    expect(wrapper.text()).toContain('Loading history detail...')
    expect(wrapper.text()).not.toContain('Generate · gpt-image-2')
    expect(wrapper.find('[data-testid="image-history-replay"]').exists()).toBe(false)
  })

  it('replays generate history parameters back into the generate form', async () => {
    listHistory.mockResolvedValue({ items: historyListItems, total: 2, page: 1, page_size: 20, pages: 1 })
    getHistoryDetail.mockResolvedValue(generateHistoryDetail)

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-history"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-list-item-31"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-replay"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="images-panel-generate"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="image-generate-prompt"]').element as HTMLTextAreaElement).value).toBe('Draw a paper crane over water')
    expect((wrapper.get('#image-generate-model').element as HTMLSelectElement).value).toBe('gpt-image-2')
    expect((wrapper.get('#image-generate-size').element as HTMLSelectElement).value).toBe('1536x1024')
    expect((wrapper.get('#image-generate-output-format').element as HTMLSelectElement).value).toBe('webp')
  })

  it('replays edit history parameters back into the edit form and shows the upload reminder', async () => {
    listHistory.mockResolvedValue({ items: historyListItems, total: 2, page: 1, page_size: 20, pages: 1 })
    getHistoryDetail.mockResolvedValue(editHistoryDetail)

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-history"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-list-item-32"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-replay"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="images-panel-edit"]').exists()).toBe(true)
    expect((wrapper.get('[data-testid="image-edit-prompt"]').element as HTMLTextAreaElement).value).toBe('Retouch the subject')
    expect((wrapper.get('#image-edit-model').element as HTMLSelectElement).value).toBe('gpt-image-2')
    expect((wrapper.get('#image-edit-size').element as HTMLSelectElement).value).toBe('1024x1024')
    expect(wrapper.text()).toContain('Replay restored the edit parameters. Re-upload the source image and optional mask before submitting again.')
  })

  it('clears replay state after leaving the replayed tab and applies only the latest replay', async () => {
    listHistory.mockResolvedValue({ items: historyListItems, total: 2, page: 1, page_size: 20, pages: 1 })
    getHistoryDetail
      .mockResolvedValueOnce(editHistoryDetail)
      .mockResolvedValueOnce(generateHistoryDetail)

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-history"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-list-item-32"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-replay"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="image-edit-replay-notice"]').text()).toContain('Re-upload the source image')
    expect((wrapper.get('[data-testid="image-edit-prompt"]').element as HTMLTextAreaElement).value).toBe('Retouch the subject')

    await wrapper.get('[data-testid="images-tab-generate"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="image-edit-replay-notice"]').exists()).toBe(false)
    expect((wrapper.get('[data-testid="image-generate-prompt"]').element as HTMLTextAreaElement).value).toBe('')

    await wrapper.get('[data-testid="images-tab-history"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-list-item-31"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-replay"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="image-edit-replay-notice"]').exists()).toBe(false)
    expect((wrapper.get('[data-testid="image-generate-prompt"]').element as HTMLTextAreaElement).value).toBe('Draw a paper crane over water')
  })

  it('clears edit replay values together with the reminder after leaving the edit tab', async () => {
    listHistory.mockResolvedValue({ items: historyListItems, total: 2, page: 1, page_size: 20, pages: 1 })
    getHistoryDetail.mockResolvedValue(editHistoryDetail)

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-history"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-list-item-32"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-replay"]').trigger('click')
    await flushPromises()

    expect((wrapper.get('[data-testid="image-edit-prompt"]').element as HTMLTextAreaElement).value).toBe('Retouch the subject')
    expect(wrapper.find('[data-testid="image-edit-replay-notice"]').exists()).toBe(true)

    await wrapper.get('[data-testid="images-tab-history"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="images-tab-edit"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="image-edit-replay-notice"]').exists()).toBe(false)
    expect((wrapper.get('[data-testid="image-edit-prompt"]').element as HTMLTextAreaElement).value).toBe('')
  })

  it('keeps form defaults when replay fields are undefined', async () => {
    listHistory.mockResolvedValue({ items: historyListItems, total: 2, page: 1, page_size: 20, pages: 1 })
    getHistoryDetail.mockResolvedValue(generateHistoryDetailWithDefaults)

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-history"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-list-item-31"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-replay"]').trigger('click')
    await flushPromises()

    expect((wrapper.get('#image-generate-size').element as HTMLSelectElement).value).toBe('1024x1024')
    expect((wrapper.get('#image-generate-quality').element as HTMLSelectElement).value).toBe('medium')
    expect((wrapper.get('#image-generate-background').element as HTMLSelectElement).value).toBe('auto')
    expect((wrapper.get('#image-generate-output-format').element as HTMLSelectElement).value).toBe('png')
  })

  it('drops unsafe history data urls before rendering detail previews', async () => {
    listHistory.mockResolvedValue({ items: historyListItems, total: 2, page: 1, page_size: 20, pages: 1 })
    getHistoryDetail.mockResolvedValue(unsafeImageHistoryDetail)

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-history"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="image-history-list-item-31"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="image-history-detail-image-0"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Submit a generate or edit request to see results.')
  })

  it('shows an empty state when the current page returns no api keys', async () => {
    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('No API keys available yet')
    expect(wrapper.text()).not.toContain('Failed to load API keys.')
  })

  it('shows a loading state while api keys are being fetched', async () => {
    list.mockImplementation(() => new Promise(() => {}))

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    expect(wrapper.text()).toContain('Loading API keys...')
    expect(wrapper.text()).not.toContain('No API keys available yet')
  })

  it('shows a load failure state when api keys cannot be fetched', async () => {
    list.mockRejectedValue(new Error('network failed'))

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Failed to load API keys.')
    expect(wrapper.text()).toContain('Retry')
    expect(wrapper.text()).not.toContain('No API keys available yet')
    expect(errorSpy).toHaveBeenCalled()
  })

  it('submits image generation with the selected api key', async () => {
    list.mockResolvedValue({
      items: [primaryApiKey, secondaryApiKey]
    })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="image-generate-prompt"]').setValue('Draw a paper crane')
    await wrapper.get('[data-testid="image-generate-submit"]').trigger('click')
    await flushPromises()

    expect(generate).toHaveBeenCalledTimes(1)
    const [payload, selectedApiKey] = generate.mock.calls[0]
    expect(payload).toEqual(
      expect.objectContaining({
        prompt: 'Draw a paper crane'
      })
    )
    expect(selectedApiKey).toBe('sk-vision-key-a')
    expect(selectedApiKey).not.toBe('7')
  })

  it('submits image generation with a custom size', async () => {
    list.mockResolvedValue({
      items: [primaryApiKey]
    })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('#image-generate-size').setValue('custom')
    await wrapper.get('[data-testid="image-generate-custom-size"]').setValue('3072x1728')
    await wrapper.get('[data-testid="image-generate-prompt"]').setValue('Draw a wide editorial hero image')
    await wrapper.get('[data-testid="image-generate-submit"]').trigger('click')
    await flushPromises()

    expect(generate).toHaveBeenCalledTimes(1)
    expect(generate.mock.calls[0][0]).toEqual(
      expect.objectContaining({
        size: '3072x1728'
      })
    )
  })

  it('renders preview results for non-png base64 responses using the selected output format mime', async () => {
    list.mockResolvedValue({
      items: [primaryApiKey]
    })
    generate.mockResolvedValue({
      created: 1,
      data: [
        {
          b64_json: 'QUJD',
          revised_prompt: 'Draw a paper crane over water'
        }
      ]
    })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('#image-generate-output-format').setValue('webp')
    await wrapper.get('[data-testid="image-generate-prompt"]').setValue('Draw a paper crane')
    await wrapper.get('[data-testid="image-generate-submit"]').trigger('click')
    await flushPromises()

    const preview = wrapper.get('[data-testid="image-result-preview-0"]')
    expect(preview.attributes('src')).toBe('data:image/webp;base64,QUJD')
    expect(wrapper.text()).toContain('Draw a paper crane over water')
  })

  it('sanitizes unsafe url results before rendering previews', async () => {
    list.mockResolvedValue({
      items: [primaryApiKey]
    })
    generate.mockResolvedValue({
      created: 1,
      data: [
        {
          url: 'javascript:alert(1)',
          revised_prompt: 'unsafe result'
        }
      ]
    })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="image-generate-prompt"]').setValue('Draw a paper crane')
    await wrapper.get('[data-testid="image-generate-submit"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="image-result-preview-0"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Submit a generate or edit request to see results.')
  })

  it('shows a validation error for blank generate prompts', async () => {
    list.mockResolvedValue({
      items: [{ id: 7, name: 'Vision Key A' }]
    })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="image-generate-prompt"]').setValue('   ')
    await wrapper.get('[data-testid="image-generate-submit"]').trigger('click')
    await flushPromises()

    expect(generate).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Prompt is required.')
  })

  it('requires a source image before submitting an edit request', async () => {
    list.mockResolvedValue({
      items: [{ id: 7, name: 'Vision Key A' }]
    })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-edit"]').trigger('click')
    await wrapper.get('[data-testid="image-edit-prompt"]').setValue('Retouch the subject')
    await wrapper.get('[data-testid="image-edit-submit"]').trigger('click')
    await flushPromises()

    expect(edit).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Source image is required.')
  })

  it('rejects non-image source files before edit submission', async () => {
    list.mockResolvedValue({
      items: [{ id: 7, name: 'Vision Key A' }]
    })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-edit"]').trigger('click')
    await wrapper.get('[data-testid="image-edit-prompt"]').setValue('Retouch the subject')

    const sourceInput = wrapper.get('[data-testid="image-edit-source-input"]')
    Object.defineProperty(sourceInput.element, 'files', {
      value: [new File(['not-image'], 'notes.txt', { type: 'text/plain' })],
      configurable: true
    })

    await sourceInput.trigger('change')
    await wrapper.get('[data-testid="image-edit-submit"]').trigger('click')
    await flushPromises()

    expect(edit).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Source image must be an image file.')
  })

  it('submits image edits as FormData when a source image is provided', async () => {
    list.mockResolvedValue({
      items: [primaryApiKey]
    })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-edit"]').trigger('click')
    await wrapper.get('[data-testid="image-edit-prompt"]').setValue('Retouch the subject')

    const sourceInput = wrapper.get('[data-testid="image-edit-source-input"]')
    Object.defineProperty(sourceInput.element, 'files', {
      value: [new File(['source-bytes'], 'source.png', { type: 'image/png' })],
      configurable: true
    })

    await sourceInput.trigger('change')
    await wrapper.get('[data-testid="image-edit-submit"]').trigger('click')
    await flushPromises()

    expect(edit).toHaveBeenCalledTimes(1)
    const [payload, selectedApiKey, options] = edit.mock.calls[0]
    expect(payload).toBeInstanceOf(FormData)
    expect(payload.get('prompt')).toBe('Retouch the subject')
    expect(payload.get('image')).toBeInstanceOf(File)
    expect(selectedApiKey).toBe('sk-vision-key-a')
    expect(selectedApiKey).not.toBe('7')
    expect(options).toBeUndefined()
  })

  it('submits image edits with a custom size in FormData', async () => {
    list.mockResolvedValue({
      items: [primaryApiKey]
    })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="images-tab-edit"]').trigger('click')
    await wrapper.get('#image-edit-size').setValue('custom')
    await wrapper.get('[data-testid="image-edit-custom-size"]').setValue('3072x1728')
    await wrapper.get('[data-testid="image-edit-prompt"]').setValue('Retouch the wide composition')

    const sourceInput = wrapper.get('[data-testid="image-edit-source-input"]')
    Object.defineProperty(sourceInput.element, 'files', {
      value: [new File(['source-bytes'], 'source.png', { type: 'image/png' })],
      configurable: true
    })

    await sourceInput.trigger('change')
    await wrapper.get('[data-testid="image-edit-submit"]').trigger('click')
    await flushPromises()

    expect(edit).toHaveBeenCalledTimes(1)
    expect((edit.mock.calls[0][0] as FormData).get('size')).toBe('3072x1728')
  })

  it('does not submit generation when no api key is selected', async () => {
    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="image-generate-prompt"]').setValue('Draw a paper crane')
    await wrapper.get('[data-testid="image-generate-form"]').trigger('submit')
    await flushPromises()

    expect(generate).not.toHaveBeenCalled()
  })

  it('uses the manually selected api key for generation submissions', async () => {
    list.mockResolvedValue({
      items: [primaryApiKey, secondaryApiKey]
    })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="image-api-key-selector"]').setValue('9')
    await wrapper.get('[data-testid="image-generate-prompt"]').setValue('Draw a new skyline')
    await wrapper.get('[data-testid="image-generate-submit"]').trigger('click')
    await flushPromises()

    expect(generate).toHaveBeenCalledTimes(1)
    expect(generate.mock.calls[0][1]).toBe('sk-vision-key-b')
    expect(generate.mock.calls[0][1]).not.toBe('9')
  })

  it('shows request errors and clears previous previews after a failed submission', async () => {
    list.mockResolvedValue({
      items: [primaryApiKey]
    })
    generate
      .mockResolvedValueOnce({
        created: 1,
        data: [
          {
            url: 'https://cdn.example.com/image.png',
            revised_prompt: 'safe result'
          }
        ]
      })
      .mockRejectedValueOnce({ message: 'Gateway failed' })

    const wrapper = mount(ImagesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="image-generate-prompt"]').setValue('Draw a safe result')
    await wrapper.get('[data-testid="image-generate-submit"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="image-result-preview-0"]').exists()).toBe(true)

    await wrapper.get('[data-testid="image-generate-prompt"]').setValue('Draw a failing result')
    await wrapper.get('[data-testid="image-generate-submit"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Gateway failed')
    expect(wrapper.find('[data-testid="image-result-preview-0"]').exists()).toBe(false)
  })
})
