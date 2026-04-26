<template>
  <form class="grid gap-5" data-testid="image-edit-form" @submit.prevent="handleSubmit">
    <div class="grid gap-5 lg:grid-cols-2">
      <div class="lg:col-span-2">
        <label class="input-label mb-1.5 block" for="image-edit-prompt">{{ t('images.forms.generate.prompt') }}</label>
        <textarea
          id="image-edit-prompt"
          v-model="form.prompt"
          class="input min-h-[120px] w-full resize-y"
          :placeholder="t('images.forms.generate.promptPlaceholder')"
          data-testid="image-edit-prompt"
          required
          @input="handlePromptInput"
        ></textarea>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-edit-source-input">{{ t('images.forms.edit.sourceImage') }}</label>
        <input
          id="image-edit-source-input"
          accept="image/png,image/jpeg,image/webp"
          class="input w-full file:mr-3 file:rounded-lg file:border-0 file:bg-primary-50 file:px-3 file:py-2 file:text-sm file:font-medium file:text-primary-700 dark:file:bg-primary-900/30 dark:file:text-primary-300"
          data-testid="image-edit-source-input"
          type="file"
          @change="handleSourceChange"
        />
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('images.forms.edit.sourceImageHint') }}</p>
        <p v-if="sourceImageName" class="mt-1 text-xs text-gray-600 dark:text-gray-300">{{ sourceImageName }}</p>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-edit-mask-input">{{ t('images.forms.edit.maskImage') }}</label>
        <input
          id="image-edit-mask-input"
          accept="image/png,image/jpeg,image/webp"
          class="input w-full file:mr-3 file:rounded-lg file:border-0 file:bg-gray-100 file:px-3 file:py-2 file:text-sm file:font-medium file:text-gray-700 dark:file:bg-dark-700 dark:file:text-gray-200"
          data-testid="image-edit-mask-input"
          type="file"
          @change="handleMaskChange"
        />
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('images.forms.edit.maskImageHint') }}</p>
        <p v-if="maskImageName" class="mt-1 text-xs text-gray-600 dark:text-gray-300">{{ maskImageName }}</p>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-edit-model">{{ t('images.forms.generate.model') }}</label>
        <select id="image-edit-model" v-model="form.model" :class="selectClass">
          <option v-for="option in modelOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-edit-size">{{ t('images.forms.generate.size') }}</label>
        <select id="image-edit-size" v-model="form.size" :class="selectClass">
          <option v-for="option in sizeOptions" :key="option.value" :value="option.value">
            {{ option.value === CUSTOM_IMAGE_SIZE_OPTION_VALUE ? t('images.forms.generate.customSize') : option.label }}
          </option>
        </select>
        <div v-if="form.size === CUSTOM_IMAGE_SIZE_OPTION_VALUE" class="mt-3">
          <label class="input-label mb-1.5 block" for="image-edit-custom-size">{{ t('images.forms.generate.customSize') }}</label>
          <input
            id="image-edit-custom-size"
            v-model.trim="customSize"
            class="input w-full"
            :placeholder="t('images.forms.generate.customSizePlaceholder')"
            data-testid="image-edit-custom-size"
            type="text"
            @input="handleCustomSizeInput"
          />
          <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400" data-testid="image-edit-custom-size-requirements">
            {{ t('images.forms.generate.customSizeRequirements') }}
          </p>
        </div>
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400" data-testid="image-edit-size-hint">{{ t('images.forms.generate.sizeHint') }}</p>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-edit-quality">{{ t('images.forms.generate.quality') }}</label>
        <select id="image-edit-quality" v-model="form.quality" :class="selectClass">
          <option v-for="option in qualityOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-edit-background">{{ t('images.forms.generate.background') }}</label>
        <select id="image-edit-background" v-model="form.background" :class="selectClass">
          <option v-for="option in backgroundOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-edit-output-format">{{ t('images.forms.generate.outputFormat') }}</label>
        <select id="image-edit-output-format" v-model="form.output_format" :class="selectClass">
          <option v-for="option in outputFormatOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-edit-moderation">{{ t('images.forms.generate.moderation') }}</label>
        <select id="image-edit-moderation" v-model="form.moderation" :class="selectClass">
          <option v-for="option in moderationOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>

    </div>

    <p v-if="showApiKeyRequiredMessage" class="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-900/40 dark:bg-amber-900/10 dark:text-amber-300">
      {{ t('images.forms.generate.apiKeyRequired') }}
    </p>
    <p v-if="validationErrorKey" class="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-900/10 dark:text-red-300">
      {{ t(validationErrorKey) }}
    </p>
    <p v-if="noticeKey" class="rounded-2xl border border-sky-200 bg-sky-50 px-4 py-3 text-sm text-sky-700 dark:border-sky-900/40 dark:bg-sky-900/10 dark:text-sky-300">
      {{ t(noticeKey) }}
    </p>

    <div class="flex justify-end">
      <button class="btn btn-primary" :disabled="disabled || loading" data-testid="image-edit-submit" type="button" @click="handleSubmit">
        {{ loading ? t('images.forms.edit.submittingWithSeconds', { seconds: loadingSeconds }) : t('images.forms.edit.submit') }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import {
  CUSTOM_IMAGE_SIZE_OPTION_VALUE,
  createDefaultImageFormValues,
  getImageSizeOptions,
  isPresetImageSize,
  normalizeImageFormValues,
  sanitizeImageGenerationPayload,
  useImageFormOptions,
  validateCustomImageSize,
} from '@/composables/useImageFormOptions'
import type { ImageCommonFormValues } from '@/composables/useImageFormOptions'

interface Props {
  disabled?: boolean
  initialValues?: Partial<ImageCommonFormValues>
  loading?: boolean
  loadingSeconds?: number
  showApiKeyRequiredMessage?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  initialValues: () => ({}),
  loading: false,
  loadingSeconds: 0,
  showApiKeyRequiredMessage: false,
})

const emit = defineEmits<{
  (e: 'submit', payload: FormData): void
}>()

const { t } = useI18n()
const defaultValues = createDefaultImageFormValues()
const normalizedInitialValues = normalizeImageFormValues({
  ...defaultValues,
  ...props.initialValues,
  model: props.initialValues.model ?? defaultValues.model,
  size: props.initialValues.size ?? defaultValues.size,
})
const usesCustomInitialSize = normalizedInitialValues.model === 'gpt-image-2' && !isPresetImageSize(normalizedInitialValues.size, normalizedInitialValues.model) && validateCustomImageSize(normalizedInitialValues.size) === null
const form = reactive({
  ...normalizedInitialValues,
  size: usesCustomInitialSize ? CUSTOM_IMAGE_SIZE_OPTION_VALUE : normalizedInitialValues.size,
})
const customSize = ref(usesCustomInitialSize ? normalizedInitialValues.size : '')
const sourceImage = ref<File | null>(null)
const maskImage = ref<File | null>(null)
const hasInvalidSourceImage = ref(false)
const validationErrorKey = ref('')
const noticeKey = ref('')
if (props.initialValues.background === 'transparent' && props.initialValues.output_format === 'jpeg') {
  noticeKey.value = 'images.forms.generate.transparentFormatAdjusted'
}
const { backgroundOptions, modelOptions, moderationOptions, outputFormatOptions, qualityOptions } = useImageFormOptions()
const sizeOptions = computed(() => getImageSizeOptions(form.model))

const selectClass = 'input w-full'
const customSizeValidationKeys = [
  'images.forms.generate.customSizeRequired',
  'images.forms.generate.customSizeFormat',
  'images.forms.generate.customSizeMultipleOf16',
  'images.forms.generate.customSizeMaxEdge',
  'images.forms.generate.customSizeAspectRatio',
  'images.forms.generate.customSizePixelRange',
]

const sourceImageName = computed(() => sourceImage.value?.name ?? '')
const maskImageName = computed(() => maskImage.value?.name ?? '')

function clearCustomSizeValidationError() {
  if (customSizeValidationKeys.includes(validationErrorKey.value)) {
    validationErrorKey.value = ''
  }
}

function clearSourceImageValidationError() {
  if (validationErrorKey.value === 'images.forms.edit.sourceImageRequired' || validationErrorKey.value === 'images.forms.edit.sourceImageInvalid') {
    validationErrorKey.value = ''
  }
}

watch(
  () => [form.model, form.background, form.output_format] as const,
  () => {
    const size = form.size === CUSTOM_IMAGE_SIZE_OPTION_VALUE ? customSize.value.trim() : form.size
    const shouldShowTransparentFormatNotice = form.background === 'transparent' && form.output_format === 'jpeg'
    const normalized = normalizeImageFormValues({ ...form, size })
    // Preserve the UI sentinel while editing custom sizes; payload sanitization validates the real size on submit.
    const keepCustomSize = form.model === 'gpt-image-2' && form.size === CUSTOM_IMAGE_SIZE_OPTION_VALUE
    Object.assign(form, normalized)
    if (keepCustomSize) {
      form.size = CUSTOM_IMAGE_SIZE_OPTION_VALUE
    }
    if (shouldShowTransparentFormatNotice) {
      noticeKey.value = 'images.forms.generate.transparentFormatAdjusted'
    } else if (form.background !== 'transparent' || form.output_format !== 'png') {
      noticeKey.value = ''
    }
  },
  { immediate: true }
)

watch(
  () => form.size,
  (size) => {
    if (size !== CUSTOM_IMAGE_SIZE_OPTION_VALUE) {
      clearCustomSizeValidationError()
    }
  }
)

function isImageFile(file: File | null): file is File {
  return !!file && file.type.startsWith('image/')
}

function handlePromptInput() {
  if (form.prompt.trim() && validationErrorKey.value === 'images.forms.generate.promptRequired') {
    validationErrorKey.value = ''
  }
}

function handleCustomSizeInput() {
  const size = customSize.value.trim()
  if (!size) {
    return
  }

  if (form.size === CUSTOM_IMAGE_SIZE_OPTION_VALUE && validateCustomImageSize(size) === null) {
    clearCustomSizeValidationError()
  }
}

function handleSourceChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] ?? null

  if (file && !isImageFile(file)) {
    sourceImage.value = null
    hasInvalidSourceImage.value = true
    validationErrorKey.value = 'images.forms.edit.sourceImageInvalid'
    input.value = ''
    return
  }

  sourceImage.value = file
  hasInvalidSourceImage.value = false
  clearSourceImageValidationError()
}

function handleMaskChange(event: Event) {
  const input = event.target as HTMLInputElement
  maskImage.value = input.files?.[0] ?? null
}

function handleSubmit() {
  if (props.disabled || props.loading) {
    return
  }

  const prompt = form.prompt.trim()
  if (!prompt) {
    validationErrorKey.value = 'images.forms.generate.promptRequired'
    return
  }

  const size = form.size === CUSTOM_IMAGE_SIZE_OPTION_VALUE ? customSize.value.trim() : form.size
  if (!size) {
    validationErrorKey.value = 'images.forms.generate.customSizeRequired'
    return
  }

  if (form.size === CUSTOM_IMAGE_SIZE_OPTION_VALUE) {
    const sizeValidationKey = validateCustomImageSize(size)
    if (sizeValidationKey) {
      validationErrorKey.value = sizeValidationKey
      return
    }
  }

  if (hasInvalidSourceImage.value) {
    validationErrorKey.value = 'images.forms.edit.sourceImageInvalid'
    return
  }

  if (!sourceImage.value) {
    validationErrorKey.value = 'images.forms.edit.sourceImageRequired'
    return
  }

  if (!isImageFile(sourceImage.value)) {
    sourceImage.value = null
    hasInvalidSourceImage.value = true
    validationErrorKey.value = 'images.forms.edit.sourceImageInvalid'
    return
  }

  validationErrorKey.value = ''

  const sanitized = sanitizeImageGenerationPayload({
    prompt,
    model: form.model,
    size,
    quality: form.quality,
    background: form.background,
    output_format: form.output_format,
    moderation: form.moderation,
    n: 1,
  })

  const payload = new FormData()
  payload.append('prompt', String(sanitized.prompt))
  payload.append('image', sourceImage.value)
  payload.append('model', String(sanitized.model))
  payload.append('size', String(sanitized.size))
  payload.append('quality', String(sanitized.quality))
  payload.append('background', String(sanitized.background))
  payload.append('output_format', String(sanitized.output_format))
  payload.append('moderation', String(sanitized.moderation))
  payload.append('n', '1')

  if (maskImage.value) {
    payload.append('mask', maskImage.value)
  }

  emit('submit', payload)
}
</script>
