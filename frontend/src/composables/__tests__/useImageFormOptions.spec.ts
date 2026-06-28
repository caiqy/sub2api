import { describe, expect, it } from 'vitest'

import {
  CUSTOM_IMAGE_SIZE_OPTION_VALUE,
  createDefaultImageFormValues,
  getImageSizeOptions,
  normalizeImageFormValues,
  sanitizeImageGenerationPayload,
  validateCustomImageSize,
  useImageFormOptions,
} from '../useImageFormOptions'

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

  it('defaults image size to auto', () => {
    expect(createDefaultImageFormValues().size).toBe('auto')
  })

  it('defaults image quality to auto', () => {
    expect(createDefaultImageFormValues().quality).toBe('auto')
  })

  it('defaults image count to 1', () => {
    expect(createDefaultImageFormValues().n).toBe(1)
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

describe('image model capabilities', () => {
  it('keeps custom and large sizes for gpt-image-2', () => {
    const values = getImageSizeOptions('gpt-image-2').map((option) => option.value)
    expect(values).toContain('3840x2160')
    expect(values).toContain(CUSTOM_IMAGE_SIZE_OPTION_VALUE)
  })

  it('restricts gpt-image-1.5 sizes to official common values', () => {
    const values = getImageSizeOptions('gpt-image-1.5').map((option) => option.value)
    expect(values).toEqual(['auto', '1024x1024', '1536x1024', '1024x1536'])
  })

  it('falls back to auto when current size is unsupported by selected model', () => {
    const values = normalizeImageFormValues({
      model: 'gpt-image-1',
      prompt: '',
      size: '3840x2160',
      quality: 'high',
      background: 'auto',
      output_format: 'png',
      moderation: 'auto',
      n: 4,
    })

    expect(values.size).toBe('auto')
    expect(values.n).toBe(1)
  })

  it('normalizes transparent jpeg output to png', () => {
    const values = normalizeImageFormValues({
      model: 'gpt-image-2',
      prompt: '',
      size: 'auto',
      quality: 'auto',
      background: 'transparent',
      output_format: 'jpeg',
      moderation: 'auto',
      n: 1,
    })

    expect(values.output_format).toBe('png')
  })

  it('sanitizes generation payload for GPT image models', () => {
    const payload = sanitizeImageGenerationPayload({
      prompt: 'a robot',
      model: 'gpt-image-2',
      size: 'auto',
      quality: 'auto',
      background: 'transparent',
      output_format: 'jpeg',
      moderation: 'auto',
      n: 4,
      response_format: 'url',
      output_compression: 80,
    })

    expect(payload).toMatchObject({
      prompt: 'a robot',
      model: 'gpt-image-2',
      size: 'auto',
      quality: 'auto',
      background: 'transparent',
      output_format: 'png',
      moderation: 'auto',
      n: 1,
    })
    expect(payload).not.toHaveProperty('response_format')
    expect(payload).not.toHaveProperty('output_compression')
  })

  it('keeps valid gpt-image-2 custom sizes during normalization and sanitization', () => {
    const normalized = normalizeImageFormValues({
      model: 'gpt-image-2',
      prompt: '',
      size: '3072x1728',
      quality: 'auto',
      background: 'auto',
      output_format: 'png',
      moderation: 'auto',
      n: 1,
    })
    const payload = sanitizeImageGenerationPayload({
      prompt: 'wide image',
      model: 'gpt-image-2',
      size: '3072x1728',
      quality: 'auto',
      background: 'auto',
      output_format: 'png',
      moderation: 'auto',
      n: 1,
    })

    expect(normalized.size).toBe('3072x1728')
    expect(payload.size).toBe('3072x1728')
  })

  it('does not allow the UI custom sentinel into normalized or sanitized API payloads', () => {
    const normalized = normalizeImageFormValues({
      model: 'gpt-image-2',
      prompt: '',
      size: CUSTOM_IMAGE_SIZE_OPTION_VALUE,
      quality: 'auto',
      background: 'auto',
      output_format: 'png',
      moderation: 'auto',
      n: 1,
    })
    const payload = sanitizeImageGenerationPayload({
      prompt: 'custom sentinel',
      model: 'gpt-image-2',
      size: CUSTOM_IMAGE_SIZE_OPTION_VALUE,
      quality: 'auto',
      background: 'auto',
      output_format: 'png',
      moderation: 'auto',
      n: 1,
    })

    expect(normalized.size).toBe('auto')
    expect(payload.size).toBe('auto')
  })

  it('falls back unknown models to the default supported model', () => {
    const normalized = normalizeImageFormValues({
      model: 'legacy-image-model',
      prompt: '',
      size: '3840x2160',
      quality: 'auto',
      background: 'auto',
      output_format: 'png',
      moderation: 'auto',
      n: 1,
    })
    const payload = sanitizeImageGenerationPayload({
      prompt: 'legacy model',
      model: 'legacy-image-model',
      size: '3840x2160',
      quality: 'auto',
      background: 'auto',
      output_format: 'png',
      moderation: 'auto',
      n: 1,
    })

    expect(normalized.model).toBe('gpt-image-2')
    expect(normalized.size).toBe('3840x2160')
    expect(payload.model).toBe('gpt-image-2')
    expect(payload.size).toBe('3840x2160')
  })

  it('trims valid custom sizes before returning normalized and sanitized values', () => {
    const normalized = normalizeImageFormValues({
      model: 'gpt-image-2',
      prompt: '',
      size: ' 3072x1728 ',
      quality: 'auto',
      background: 'auto',
      output_format: 'png',
      moderation: 'auto',
      n: 1,
    })
    const payload = sanitizeImageGenerationPayload({
      prompt: 'trim size',
      model: 'gpt-image-2',
      size: ' 3072x1728 ',
      quality: 'auto',
      background: 'auto',
      output_format: 'png',
      moderation: 'auto',
      n: 1,
    })

    expect(normalized.size).toBe('3072x1728')
    expect(payload.size).toBe('3072x1728')
  })
})
