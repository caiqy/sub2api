import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import ImageResultPanel from '../ImageResultPanel.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('ImageResultPanel', () => {
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
})
