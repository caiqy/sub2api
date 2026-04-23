import { describe, expect, it } from 'vitest'

import { createDefaultImageFormValues, useImageFormOptions } from '../useImageFormOptions'

describe('useImageFormOptions', () => {
  it('includes OpenAI recommended gpt-image-2 size presets plus auto', () => {
    const { sizeOptions } = useImageFormOptions()

    expect(sizeOptions.map((option) => option.value)).toEqual([
      'auto',
      '1024x1024',
      '1536x1024',
      '1024x1536',
      '2048x2048',
      '2048x1152',
      '3840x2160',
      '2160x3840',
      'custom',
    ])
  })

  it('keeps the existing square size as default form value', () => {
    expect(createDefaultImageFormValues().size).toBe('1024x1024')
  })
})
