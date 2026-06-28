import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import ImagePreviewGallery from '../ImagePreviewGallery.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('ImagePreviewGallery', () => {
  it('renders full-width adaptive images', () => {
    const wrapper = mount(ImagePreviewGallery, {
      props: {
        images: [{ src: 'data:image/png;base64,QUJD', source: 'data-url' }],
        imageTestIdPrefix: 'gallery-image',
      },
    })

    expect(wrapper.get('[data-testid="image-preview-gallery"]').classes()).toContain('grid-cols-1')
    expect(wrapper.get('[data-testid="gallery-image-0"]').classes()).toContain('w-full')
    expect(wrapper.get('[data-testid="gallery-image-0"]').classes()).toContain('object-contain')
  })

  it('opens fullscreen preview and exposes download', async () => {
    const wrapper = mount(ImagePreviewGallery, {
      props: {
        images: [{ src: 'data:image/png;base64,QUJD', source: 'data-url' }],
      },
      attachTo: document.body,
    })

    await wrapper.get('[data-testid="image-preview-open-0"]').trigger('click')

    expect(wrapper.get('[data-testid="image-preview-modal"]').attributes('aria-modal')).toBe('true')
    expect(wrapper.get('[data-testid="image-preview-modal-download"]').attributes('href')).toBe('data:image/png;base64,QUJD')

    await wrapper.get('[data-testid="image-preview-close"]').trigger('click')
    expect(wrapper.find('[data-testid="image-preview-modal"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('filters unsafe url images while allowing image data urls', () => {
    const wrapper = mount(ImagePreviewGallery, {
      props: {
        images: [
          { src: 'javascript:alert(1)', source: 'url' },
          { src: 'data:image/webp;base64,QUJD', source: 'data-url' },
        ],
      },
    })

    expect(wrapper.find('[data-testid="image-preview-open-0"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="image-preview-open-1"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="image-preview-image-0"]').attributes('src')).toBe('data:image/webp;base64,QUJD')
  })
})
