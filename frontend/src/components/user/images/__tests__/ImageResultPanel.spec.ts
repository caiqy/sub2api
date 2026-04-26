import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import ImageResultPanel from '../ImageResultPanel.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('ImageResultPanel', () => {
  it('shows backend recorded duration when provided', () => {
    const wrapper = mount(ImageResultPanel, {
      props: {
        loading: false,
        error: '',
        durationMs: 2140,
        results: [{ src: 'data:image/png;base64,QUJD', source: 'data-url' }],
      },
    })

    expect(wrapper.text()).toContain('images.results.duration')
    expect(wrapper.text()).toContain('2.1s')
  })

  it('uses a single-column large preview layout for one image', () => {
    const wrapper = mount(ImageResultPanel, {
      props: {
        loading: false,
        error: '',
        results: [{ src: 'data:image/png;base64,QUJD', source: 'data-url' }],
      },
    })

    expect(wrapper.get('[data-testid="image-result-grid"]').classes()).toContain('grid-cols-1')
    expect(wrapper.get('[data-testid="image-result-preview-0"]').classes()).toContain('max-h-[70vh]')
    expect(wrapper.get('[data-testid="image-result-preview-0"]').classes()).toContain('min-h-[240px]')
  })

  it('opens image preview in an immersive fullscreen modal', async () => {
    const wrapper = mount(ImageResultPanel, {
      props: {
        loading: false,
        error: '',
        results: [{ src: 'data:image/png;base64,QUJD', source: 'data-url' }],
      },
    })

    await wrapper.get('[data-testid="image-result-open-0"]').trigger('click')

    expect(wrapper.get('[data-testid="image-result-preview-modal"]').classes()).toContain('inset-0')
    expect(wrapper.get('[data-testid="image-result-preview-modal"]').classes()).toContain('bg-black/95')
    expect(wrapper.get('[data-testid="image-result-preview-modal-image"]').classes()).toContain('max-h-[calc(100vh-6rem)]')
    expect(wrapper.get('[data-testid="image-result-preview-modal-download"]').attributes('href')).toBe('data:image/png;base64,QUJD')
  })

  it('marks the preview modal as modal, labels it, and moves focus to close button', async () => {
    const wrapper = mount(ImageResultPanel, {
      props: {
        loading: false,
        error: '',
        results: [{ src: 'data:image/png;base64,QUJD', source: 'data-url' }],
      },
      attachTo: document.body,
    })

    await wrapper.get('[data-testid="image-result-open-0"]').trigger('click')
    await wrapper.vm.$nextTick()

    const modal = wrapper.get('[data-testid="image-result-preview-modal"]')
    const closeButton = wrapper.get('[data-testid="image-result-preview-close"]')

    expect(modal.attributes('aria-modal')).toBe('true')
    expect(modal.attributes('aria-label')).toBe('images.results.previewTitle')
    expect(document.activeElement).toBe(closeButton.element)

    wrapper.unmount()
  })

  it('restores focus to the preview trigger after closing the modal', async () => {
    const wrapper = mount(ImageResultPanel, {
      props: {
        loading: false,
        error: '',
        results: [{ src: 'data:image/png;base64,QUJD', source: 'data-url' }],
      },
      attachTo: document.body,
    })

    const openButton = wrapper.get('[data-testid="image-result-open-0"]')
    openButton.element.focus()

    await openButton.trigger('click')
    await wrapper.vm.$nextTick()
    await wrapper.get('[data-testid="image-result-preview-close"]').trigger('click')
    await wrapper.vm.$nextTick()

    expect(document.activeElement).toBe(openButton.element)

    wrapper.unmount()
  })

  it('closes the preview modal on backdrop self click and restores focus to the trigger', async () => {
    const wrapper = mount(ImageResultPanel, {
      props: {
        loading: false,
        error: '',
        results: [{ src: 'data:image/png;base64,QUJD', source: 'data-url' }],
      },
      attachTo: document.body,
    })

    const openButton = wrapper.get('[data-testid="image-result-open-0"]')
    openButton.element.focus()

    await openButton.trigger('click')
    await wrapper.vm.$nextTick()

    const modal = wrapper.get('[data-testid="image-result-preview-modal"]')
    modal.element.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="image-result-preview-modal"]').exists()).toBe(false)
    expect(document.activeElement).toBe(openButton.element)

    wrapper.unmount()
  })

  it('traps tab navigation between modal download and close controls', async () => {
    const wrapper = mount(ImageResultPanel, {
      props: {
        loading: false,
        error: '',
        results: [{ src: 'data:image/png;base64,QUJD', source: 'data-url' }],
      },
      attachTo: document.body,
    })

    await wrapper.get('[data-testid="image-result-open-0"]').trigger('click')
    await wrapper.vm.$nextTick()

    const downloadLink = wrapper.get('[data-testid="image-result-preview-modal-download"]')
    const closeButton = wrapper.get('[data-testid="image-result-preview-close"]')

    expect(document.activeElement).toBe(closeButton.element)

    await closeButton.trigger('keydown', { key: 'Tab' })
    expect(document.activeElement).toBe(downloadLink.element)

    await downloadLink.trigger('keydown', { key: 'Tab', shiftKey: true })
    expect(document.activeElement).toBe(closeButton.element)

    wrapper.unmount()
  })

  it('closes fullscreen preview with the close button and Escape key', async () => {
    const wrapper = mount(ImageResultPanel, {
      props: {
        loading: false,
        error: '',
        results: [{ src: 'data:image/png;base64,QUJD', source: 'data-url' }],
      },
      attachTo: document.body,
    })

    await wrapper.get('[data-testid="image-result-open-0"]').trigger('click')
    expect(wrapper.find('[data-testid="image-result-preview-modal"]').exists()).toBe(true)

    await wrapper.get('[data-testid="image-result-preview-close"]').trigger('click')
    expect(wrapper.find('[data-testid="image-result-preview-modal"]').exists()).toBe(false)

    await wrapper.get('[data-testid="image-result-open-0"]').trigger('click')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="image-result-preview-modal"]').exists()).toBe(false)

    wrapper.unmount()
  })

  it('registers the Escape listener only while the preview modal is open', async () => {
    const addEventListenerSpy = vi.spyOn(window, 'addEventListener')
    const removeEventListenerSpy = vi.spyOn(window, 'removeEventListener')

    try {
      const wrapper = mount(ImageResultPanel, {
        props: {
          loading: false,
          error: '',
          results: [{ src: 'data:image/png;base64,QUJD', source: 'data-url' }],
        },
      })

      const getKeydownAddCalls = () => addEventListenerSpy.mock.calls.filter(([type]) => type === 'keydown')
      const getKeydownRemoveCalls = () => removeEventListenerSpy.mock.calls.filter(([type]) => type === 'keydown')

      expect(getKeydownAddCalls()).toHaveLength(0)
      expect(getKeydownRemoveCalls()).toHaveLength(0)

      await wrapper.get('[data-testid="image-result-open-0"]').trigger('click')
      await wrapper.vm.$nextTick()

      const firstRegisteredHandler = getKeydownAddCalls()[0]?.[1]

      expect(getKeydownAddCalls()).toHaveLength(1)

      await wrapper.get('[data-testid="image-result-preview-close"]').trigger('click')
      await wrapper.vm.$nextTick()

      expect(getKeydownRemoveCalls()).toHaveLength(1)
      expect(getKeydownRemoveCalls()[0]?.[1]).toBe(firstRegisteredHandler)

      await wrapper.get('[data-testid="image-result-open-0"]').trigger('click')
      await wrapper.vm.$nextTick()

      const secondRegisteredHandler = getKeydownAddCalls()[1]?.[1]

      expect(getKeydownAddCalls()).toHaveLength(2)

      wrapper.unmount()

      expect(getKeydownRemoveCalls()).toHaveLength(2)
      expect(getKeydownRemoveCalls()[1]?.[1]).toBe(secondRegisteredHandler)
    } finally {
      addEventListenerSpy.mockRestore()
      removeEventListenerSpy.mockRestore()
    }
  })
})
