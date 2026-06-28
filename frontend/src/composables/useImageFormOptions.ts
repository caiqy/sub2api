import type { ImageGenerationRequest } from '@/types'

export interface ImageFormOption<T extends string | number = string> {
  value: T
  label: string
}

export interface ImageCommonFormValues {
  model: string
  prompt: string
  size: string
  quality: string
  background: string
  output_format: string
  moderation: string
  n: number
}

export const CUSTOM_IMAGE_SIZE_OPTION_VALUE = 'custom'

const DEFAULT_IMAGE_MODEL = 'gpt-image-2'
const DEFAULT_IMAGE_SIZE = 'auto'
const DEFAULT_IMAGE_QUALITY = 'auto'
const DEFAULT_IMAGE_COUNT = 1

const modelOptions: ImageFormOption[] = [
  { value: 'gpt-image-2', label: 'gpt-image-2' },
  { value: 'gpt-image-1.5', label: 'gpt-image-1.5' },
  { value: 'gpt-image-1', label: 'gpt-image-1' },
]

const gptImage2PresetSizeOptions: ImageFormOption[] = [
  { value: 'auto', label: 'auto' },
  { value: '1024x1024', label: '1024x1024' },
  { value: '1536x1024', label: '1536x1024' },
  { value: '1024x1536', label: '1024x1536' },
  { value: '2048x2048', label: '2048x2048' },
  { value: '2048x1152', label: '2048x1152' },
  { value: '3840x2160', label: '3840x2160' },
  { value: '2160x3840', label: '2160x3840' },
]

const officialGptImagePresetSizeOptions: ImageFormOption[] = [
  { value: 'auto', label: 'auto' },
  { value: '1024x1024', label: '1024x1024' },
  { value: '1536x1024', label: '1536x1024' },
  { value: '1024x1536', label: '1024x1536' },
]

const gptImage2SizeOptions: ImageFormOption[] = [
  ...gptImage2PresetSizeOptions,
  { value: CUSTOM_IMAGE_SIZE_OPTION_VALUE, label: CUSTOM_IMAGE_SIZE_OPTION_VALUE },
]

const qualityOptions: ImageFormOption[] = [
  { value: 'auto', label: 'auto' },
  { value: 'low', label: 'low' },
  { value: 'medium', label: 'medium' },
  { value: 'high', label: 'high' },
]

const backgroundOptions: ImageFormOption[] = [
  { value: 'auto', label: 'auto' },
  { value: 'transparent', label: 'transparent' },
  { value: 'opaque', label: 'opaque' },
]

const outputFormatOptions: ImageFormOption[] = [
  { value: 'png', label: 'png' },
  { value: 'webp', label: 'webp' },
  { value: 'jpeg', label: 'jpeg' },
]

const moderationOptions: ImageFormOption[] = [
  { value: 'auto', label: 'auto' },
  { value: 'low', label: 'low' },
]

export function getImageSizeOptions(model: string): ImageFormOption[] {
  return model === 'gpt-image-2' ? gptImage2SizeOptions : officialGptImagePresetSizeOptions
}

export function isPresetImageSize(value?: string, model = DEFAULT_IMAGE_MODEL): value is string {
  return !!value && getImageSizeOptions(model).some((option) => option.value === value && option.value !== CUSTOM_IMAGE_SIZE_OPTION_VALUE)
}

export function createDefaultImageFormValues(): ImageCommonFormValues {
  return {
    model: DEFAULT_IMAGE_MODEL,
    prompt: '',
    size: DEFAULT_IMAGE_SIZE,
    quality: DEFAULT_IMAGE_QUALITY,
    background: backgroundOptions[0].value,
    output_format: outputFormatOptions[0].value,
    moderation: moderationOptions[0].value,
    n: DEFAULT_IMAGE_COUNT,
  }
}

export function normalizeImageFormValues(values: ImageCommonFormValues): ImageCommonFormValues {
  const next = { ...values, n: DEFAULT_IMAGE_COUNT, size: values.size.trim() }
  if (!modelOptions.some((option) => option.value === next.model)) {
    next.model = DEFAULT_IMAGE_MODEL
  }
  const sizeOptions = getImageSizeOptions(next.model)
  const supportsPresetSize = next.size !== CUSTOM_IMAGE_SIZE_OPTION_VALUE && sizeOptions.some((option) => option.value === next.size)
  const supportsCustomSize = next.model === 'gpt-image-2' && next.size !== CUSTOM_IMAGE_SIZE_OPTION_VALUE && validateCustomImageSize(next.size) === null

  if (!supportsPresetSize && !supportsCustomSize) {
    next.size = DEFAULT_IMAGE_SIZE
  }

  if (!qualityOptions.some((option) => option.value === next.quality)) {
    next.quality = DEFAULT_IMAGE_QUALITY
  }

  if (!backgroundOptions.some((option) => option.value === next.background)) {
    next.background = backgroundOptions[0].value
  }

  if (!outputFormatOptions.some((option) => option.value === next.output_format)) {
    next.output_format = outputFormatOptions[0].value
  }

  if (!moderationOptions.some((option) => option.value === next.moderation)) {
    next.moderation = moderationOptions[0].value
  }

  if (next.background === 'transparent' && next.output_format === 'jpeg') {
    next.output_format = 'png'
  }

  return next
}

export function sanitizeImageGenerationPayload(payload: ImageGenerationRequest): ImageGenerationRequest {
  const normalized = normalizeImageFormValues({
    model: String(payload.model ?? DEFAULT_IMAGE_MODEL),
    prompt: String(payload.prompt ?? '').trim(),
    size: String(payload.size ?? DEFAULT_IMAGE_SIZE),
    quality: String(payload.quality ?? DEFAULT_IMAGE_QUALITY),
    background: String(payload.background ?? 'auto'),
    output_format: String(payload.output_format ?? 'png'),
    moderation: String(payload.moderation ?? 'auto'),
    n: DEFAULT_IMAGE_COUNT,
  })

  const sanitized: ImageGenerationRequest = {
    prompt: normalized.prompt,
    model: normalized.model,
    size: normalized.size,
    quality: normalized.quality,
    background: normalized.background,
    output_format: normalized.output_format,
    moderation: normalized.moderation,
    n: DEFAULT_IMAGE_COUNT,
  }

  if ((normalized.output_format === 'webp' || normalized.output_format === 'jpeg') && typeof payload.output_compression === 'number') {
    sanitized.output_compression = payload.output_compression
  }

  return sanitized
}

export function validateCustomImageSize(value: string): string | null {
  const trimmedValue = value.trim()
  const match = trimmedValue.match(/^(\d+)x(\d+)$/i)

  if (!match) {
    return 'images.forms.generate.customSizeFormat'
  }

  const width = Number(match[1])
  const height = Number(match[2])

  if (!Number.isInteger(width) || !Number.isInteger(height) || width <= 0 || height <= 0) {
    return 'images.forms.generate.customSizeFormat'
  }

  if (width % 16 !== 0 || height % 16 !== 0) {
    return 'images.forms.generate.customSizeMultipleOf16'
  }

  if (Math.max(width, height) > 3840) {
    return 'images.forms.generate.customSizeMaxEdge'
  }

  if (Math.max(width, height) / Math.min(width, height) > 3) {
    return 'images.forms.generate.customSizeAspectRatio'
  }

  const totalPixels = width * height
  if (totalPixels < 655_360 || totalPixels > 8_294_400) {
    return 'images.forms.generate.customSizePixelRange'
  }

  return null
}

export function useImageFormOptions() {
  return {
    modelOptions,
    sizeOptions: gptImage2SizeOptions,
    qualityOptions,
    backgroundOptions,
    outputFormatOptions,
    moderationOptions,
    getImageSizeOptions,
  }
}
