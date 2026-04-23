import { describe, expect, it } from 'vitest'

import { createDefaultImageFormValues, validateCustomImageSize, useImageFormOptions } from '../useImageFormOptions'

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

  it('rejects custom sizes that violate OpenAI constraints', () => {
    expect(validateCustomImageSize('2050x1152')).toBe('images.forms.generate.customSizeMultipleOf16')
    expect(validateCustomImageSize('4096x1024')).toBe('images.forms.generate.customSizeMaxEdge')
    expect(validateCustomImageSize('3072x1008')).toBe('images.forms.generate.customSizeAspectRatio')
    expect(validateCustomImageSize('512x512')).toBe('images.forms.generate.customSizePixelRange')
  })

  it('accepts custom sizes that satisfy OpenAI constraints', () => {
    expect(validateCustomImageSize('3072x1728')).toBeNull()
    expect(validateCustomImageSize('2048x1152')).toBeNull()
  })
})
