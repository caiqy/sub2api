<template>
  <div class="grid gap-4">
    <div>
      <label class="input-label mb-1.5 block" :for="`${id}-prompt`">{{ t('images.forms.generate.prompt') }}</label>
      <textarea :id="`${id}-prompt`" v-model="form.prompt" class="input min-h-[140px] w-full resize-y" :placeholder="t('images.forms.generate.promptPlaceholder')" :data-testid="`${id}-prompt`" required />
    </div>
    <div>
      <label class="input-label mb-1.5 block" :for="`${id}-model`">{{ t('images.forms.generate.model') }}</label>
      <select :id="`${id}-model`" v-model="form.model" class="input w-full">
        <option value="" disabled>{{ t('images.forms.generate.modelRequired') }}</option>
        <option v-for="model in models" :key="model" :value="model">{{ model }}</option>
      </select>
    </div>
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <div>
        <label class="input-label mb-1.5 block" :for="`${id}-size`">{{ t('images.forms.generate.size') }}</label>
        <select :id="`${id}-size`" v-model="form.size" class="input w-full">
          <option v-for="option in sizes" :key="option.value" :value="option.value">{{ sizeLabel(option.value) }}</option>
        </select>
        <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('images.forms.generate.sizeHint') }}</p>
      </div>
      <div>
        <label class="input-label mb-1.5 block" :for="`${id}-quality`">{{ t('images.forms.generate.quality') }}</label>
        <select :id="`${id}-quality`" v-model="form.quality" class="input w-full">
          <option v-for="option in qualities" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>
    </div>
    <div v-if="form.size === CUSTOM_IMAGE_SIZE_OPTION_VALUE">
      <label class="input-label mb-1.5 block" :for="`${id}-custom-size`">{{ t('images.forms.generate.customSize') }}</label>
      <input :id="`${id}-custom-size`" v-model.trim="customSize" class="input w-full" :data-testid="`${id}-custom-size`" :placeholder="t('images.forms.generate.customSizePlaceholder')" />
      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('images.forms.generate.customSizeRequirements') }}</p>
    </div>
    <details class="rounded-xl border border-gray-200 p-3 dark:border-dark-600">
      <summary class="cursor-pointer text-sm font-medium">{{ t('images.forms.generate.advanced') }}</summary>
      <div class="mt-4 grid gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label mb-1.5 block" :for="`${id}-background`">{{ t('images.forms.generate.background') }}</label>
          <select :id="`${id}-background`" v-model="form.background" class="input w-full"><option v-for="option in backgroundOptions" :key="option.value" :value="option.value">{{ option.label }}</option></select>
        </div>
        <div>
          <label class="input-label mb-1.5 block" :for="`${id}-output-format`">{{ t('images.forms.generate.outputFormat') }}</label>
          <select :id="`${id}-output-format`" v-model="form.output_format" class="input w-full"><option v-for="option in outputFormatOptions" :key="option.value" :value="option.value">{{ option.label }}</option></select>
        </div>
        <div v-if="form.output_format !== 'png'">
          <label class="input-label mb-1.5 block" :for="`${id}-compression`">{{ t('images.forms.generate.outputCompression') }}</label>
          <input :id="`${id}-compression`" v-model.number="compression" class="input w-full" type="number" min="0" max="100" step="1" />
        </div>
        <div>
          <label class="input-label mb-1.5 block" :for="`${id}-moderation`">{{ t('images.forms.generate.moderation') }}</label>
          <select :id="`${id}-moderation`" v-model="form.moderation" class="input w-full"><option v-for="option in moderationOptions" :key="option.value" :value="option.value">{{ option.label }}</option></select>
        </div>
      </div>
    </details>
    <p v-if="noticeKey" role="status" class="text-sm text-sky-700 dark:text-sky-300">{{ t(noticeKey) }}</p>
    <p v-if="errorKey" role="alert" class="text-sm text-red-700 dark:text-red-300">{{ t(errorKey) }}</p>
  </div>
</template>
<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { CUSTOM_IMAGE_SIZE_OPTION_VALUE, createDefaultImageFormValues, getImageQualityOptions, getImageSizeOptions, isPresetImageSize, normalizeImageFormValues, sanitizeImageGenerationPayload, supportsCustomImageSize, useImageFormOptions, validateCustomImageSize } from '@/composables/useImageFormOptions'
import type { ImageCommonFormValues } from '@/composables/useImageFormOptions'

const props = withDefaults(defineProps<{ id: string; initialValues?: Partial<ImageCommonFormValues>; models?: string[] }>(), { initialValues: () => ({}), models: () => [] })
const { t } = useI18n()
const form = reactive(createDefaultImageFormValues())
const customSize = ref('')
const compression = ref<number | string>('')
const errorKey = ref('')
const noticeKey = ref('')
const sizes = computed(() => getImageSizeOptions(form.model))
const qualities = computed(() => getImageQualityOptions(form.model))
const { backgroundOptions, moderationOptions, outputFormatOptions } = useImageFormOptions()
function sizeLabel(size: string) {
  const labels: Record<string, string> = { auto: 'autoSize', '1024x1024': 'squareSize', '1536x1024': 'landscapeSize', '1024x1536': 'portraitSize', custom: 'customSize' }
  return labels[size] ? t(`images.forms.generate.${labels[size]}`) : size
}
function reset(values: Partial<ImageCommonFormValues> = {}) {
  const input = { ...createDefaultImageFormValues(), ...values }
  const original = { ...input }
  if (props.models.length && !props.models.includes(input.model)) input.model = props.models[0]
  const normalized = normalizeImageFormValues(input)
  noticeKey.value = JSON.stringify(original) !== JSON.stringify(normalized) ? 'images.forms.generate.parametersAdjusted' : ''
  if (input.background === 'transparent' && input.output_format === 'jpeg') noticeKey.value = 'images.forms.generate.transparentFormatAdjusted'
  Object.assign(form, normalized)
  compression.value = values.output_compression ?? ''
  customSize.value = ''
  if (supportsCustomImageSize(form.model) && !isPresetImageSize(form.size, form.model)) {
    customSize.value = form.size
    form.size = CUSTOM_IMAGE_SIZE_OPTION_VALUE
  }
  errorKey.value = ''
}
watch(() => props.initialValues, values => reset(values), { immediate: true, deep: true })
watch(() => props.models, models => {
  if (models.length && !models.includes(form.model)) form.model = models[0]
}, { deep: true })
watch(() => [form.model, form.background, form.output_format], () => {
  const keepCustom = form.size === CUSTOM_IMAGE_SIZE_OPTION_VALUE && supportsCustomImageSize(form.model)
  const transparentAdjustment = form.background === 'transparent' && form.output_format === 'jpeg'
  const normalized = normalizeImageFormValues({ ...form, size: keepCustom ? customSize.value : form.size })
  if ((!keepCustom && normalized.size !== form.size) || normalized.quality !== form.quality) noticeKey.value = 'images.forms.generate.parametersAdjusted'
  Object.assign(form, normalized)
  if (keepCustom) form.size = CUSTOM_IMAGE_SIZE_OPTION_VALUE
  if (transparentAdjustment) noticeKey.value = 'images.forms.generate.transparentFormatAdjusted'
  if (form.output_format === 'png') compression.value = ''
})
watch(() => form.prompt, prompt => {
  if (prompt.trim() && errorKey.value === 'images.forms.generate.promptRequired') errorKey.value = ''
})
watch(() => [form.size, customSize.value], () => {
  if (errorKey.value.startsWith('images.forms.generate.customSize') && (form.size !== CUSTOM_IMAGE_SIZE_OPTION_VALUE || validateCustomImageSize(customSize.value) === null)) errorKey.value = ''
})
function getPayload() {
  errorKey.value = ''
  if (!form.model || !props.models.includes(form.model)) errorKey.value = 'images.forms.generate.modelRequired'
  else if (!form.prompt.trim()) errorKey.value = 'images.forms.generate.promptRequired'
  const size = form.size === CUSTOM_IMAGE_SIZE_OPTION_VALUE ? customSize.value : form.size
  if (!errorKey.value && form.size === CUSTOM_IMAGE_SIZE_OPTION_VALUE) errorKey.value = size ? validateCustomImageSize(size) || '' : 'images.forms.generate.customSizeRequired'
  if (!errorKey.value && form.output_format !== 'png' && compression.value !== '' && (!Number.isInteger(compression.value) || Number(compression.value) < 0 || Number(compression.value) > 100)) errorKey.value = 'images.forms.generate.compressionInvalid'
  if (errorKey.value) return null
  return sanitizeImageGenerationPayload({ ...form, size, output_compression: compression.value === '' ? undefined : Number(compression.value) })
}
defineExpose({ getPayload, reset })
</script>
