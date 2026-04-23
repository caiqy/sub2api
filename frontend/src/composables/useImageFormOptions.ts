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

const sizeOptions: ImageFormOption[] = [
  { value: '1024x1024', label: '1024x1024' },
  { value: '1536x1024', label: '1536x1024' },
  { value: '1024x1536', label: '1024x1536' },
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
    size: sizeOptions[0].value,
    quality: qualityOptions[0].value,
    background: backgroundOptions[0].value,
    output_format: outputFormatOptions[0].value,
    moderation: moderationOptions[0].value,
    n: countOptions[0].value,
  }
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
