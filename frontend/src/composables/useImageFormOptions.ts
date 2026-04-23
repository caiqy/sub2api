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

const modelOptions: ImageFormOption[] = [
  { value: 'gpt-image-2', label: 'gpt-image-2' },
  { value: 'gpt-image-1.5', label: 'gpt-image-1.5' },
  { value: 'gpt-image-1', label: 'gpt-image-1' },
]

export const CUSTOM_IMAGE_SIZE_OPTION_VALUE = 'custom'
const DEFAULT_IMAGE_SIZE = 'auto'
const DEFAULT_IMAGE_QUALITY = 'high'

const presetSizeOptions: ImageFormOption[] = [
  { value: 'auto', label: 'auto' },
  { value: '1024x1024', label: '1024x1024' },
  { value: '1536x1024', label: '1536x1024' },
  { value: '1024x1536', label: '1024x1536' },
  { value: '2048x2048', label: '2048x2048' },
  { value: '2048x1152', label: '2048x1152' },
  { value: '3840x2160', label: '3840x2160' },
  { value: '2160x3840', label: '2160x3840' },
]

const sizeOptions: ImageFormOption[] = [
  ...presetSizeOptions,
  { value: CUSTOM_IMAGE_SIZE_OPTION_VALUE, label: CUSTOM_IMAGE_SIZE_OPTION_VALUE },
]

const qualityOptions: ImageFormOption[] = [
  { value: 'medium', label: 'medium' },
  { value: 'high', label: 'high' },
  { value: 'low', label: 'low' },
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

const countOptions: ImageFormOption<number>[] = [
  { value: 1, label: '1' },
  { value: 2, label: '2' },
  { value: 3, label: '3' },
  { value: 4, label: '4' },
]

export function createDefaultImageFormValues(): ImageCommonFormValues {
  return {
    model: modelOptions[0].value,
    prompt: '',
    size: DEFAULT_IMAGE_SIZE,
    quality: DEFAULT_IMAGE_QUALITY,
    background: backgroundOptions[0].value,
    output_format: outputFormatOptions[0].value,
    moderation: moderationOptions[0].value,
    n: countOptions[0].value,
  }
}

export function isPresetImageSize(value?: string): value is string {
  return !!value && presetSizeOptions.some((option) => option.value === value)
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
    sizeOptions,
    qualityOptions,
    backgroundOptions,
    outputFormatOptions,
    moderationOptions,
    countOptions,
  }
}
