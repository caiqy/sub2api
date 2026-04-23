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

      <div>
        <label class="input-label mb-1.5 block" for="image-edit-n">{{ t('images.forms.generate.n') }}</label>
        <select id="image-edit-n" v-model.number="form.n" :class="selectClass">
          <option v-for="option in countOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>
    </div>

    <p v-if="disabled" class="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-900/40 dark:bg-amber-900/10 dark:text-amber-300">
      {{ t('images.forms.generate.apiKeyRequired') }}
    </p>
    <p v-if="validationError" class="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-900/10 dark:text-red-300">
      {{ validationError }}
    </p>

    <div class="flex justify-end">
      <button class="btn btn-primary" :disabled="disabled || loading" data-testid="image-edit-submit" type="button" @click="handleSubmit">
        {{ loading ? t('images.forms.edit.submitting') : t('images.forms.edit.submit') }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import { CUSTOM_IMAGE_SIZE_OPTION_VALUE, createDefaultImageFormValues, isPresetImageSize, useImageFormOptions } from '@/composables/useImageFormOptions'
import type { ImageCommonFormValues } from '@/composables/useImageFormOptions'

interface Props {
  disabled?: boolean
  initialValues?: Partial<ImageCommonFormValues>
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  initialValues: () => ({}),
  loading: false,
})

const emit = defineEmits<{
  (e: 'submit', payload: FormData): void
}>()

const { t } = useI18n()
const defaultValues = createDefaultImageFormValues()
const initialSize = props.initialValues.size
const usesPresetInitialSize = !initialSize || isPresetImageSize(initialSize)
const form = reactive({
  ...defaultValues,
  ...props.initialValues,
  size: usesPresetInitialSize ? (initialSize ?? defaultValues.size) : CUSTOM_IMAGE_SIZE_OPTION_VALUE,
})
const customSize = ref(usesPresetInitialSize ? '' : initialSize ?? '')
const sourceImage = ref<File | null>(null)
const maskImage = ref<File | null>(null)
const hasInvalidSourceImage = ref(false)
const validationError = ref('')
const { backgroundOptions, countOptions, modelOptions, moderationOptions, outputFormatOptions, qualityOptions, sizeOptions } = useImageFormOptions()

const selectClass = 'input w-full'

const sourceImageName = computed(() => sourceImage.value?.name ?? '')
const maskImageName = computed(() => maskImage.value?.name ?? '')

function isImageFile(file: File | null): file is File {
  return !!file && file.type.startsWith('image/')
}

function handlePromptInput() {
  if (form.prompt.trim() && validationError.value === t('images.forms.generate.promptRequired')) {
    validationError.value = ''
  }
}

function handleCustomSizeInput() {
  if (customSize.value.trim()) {
    validationError.value = ''
  }
}

function handleSourceChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] ?? null

  if (file && !isImageFile(file)) {
    sourceImage.value = null
    hasInvalidSourceImage.value = true
    validationError.value = t('images.forms.edit.sourceImageInvalid')
    input.value = ''
    return
  }

  sourceImage.value = file
  hasInvalidSourceImage.value = false
  validationError.value = ''
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
    validationError.value = t('images.forms.generate.promptRequired')
    return
  }

  const size = form.size === CUSTOM_IMAGE_SIZE_OPTION_VALUE ? customSize.value.trim() : form.size
  if (!size) {
    validationError.value = t('images.forms.generate.customSizeRequired')
    return
  }

  if (hasInvalidSourceImage.value) {
    validationError.value = t('images.forms.edit.sourceImageInvalid')
    return
  }

  if (!sourceImage.value) {
    validationError.value = t('images.forms.edit.sourceImageRequired')
    return
  }

  if (!isImageFile(sourceImage.value)) {
    sourceImage.value = null
    hasInvalidSourceImage.value = true
    validationError.value = t('images.forms.edit.sourceImageInvalid')
    return
  }

  validationError.value = ''

  const payload = new FormData()
  payload.append('prompt', prompt)
  payload.append('image', sourceImage.value)
  payload.append('model', form.model)
  payload.append('size', size)
  payload.append('quality', form.quality)
  payload.append('background', form.background)
  payload.append('output_format', form.output_format)
  payload.append('moderation', form.moderation)
  payload.append('n', String(form.n))

  if (maskImage.value) {
    payload.append('mask', maskImage.value)
  }

  emit('submit', payload)
}
</script>
