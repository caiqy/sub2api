<template>
  <form class="grid gap-5" data-testid="image-edit-form" @submit.prevent="handleSubmit">
    <div>
      <label class="input-label mb-1.5 block" for="image-edit-source-input">{{ t('images.forms.edit.sourceImage') }}</label>
      <input id="image-edit-source-input" accept="image/png,image/jpeg,image/webp" class="input w-full" data-testid="image-edit-source-input" type="file" multiple :disabled="validating || loading" @change="handleSourceChange" />
      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('images.forms.edit.sourceImageHint') }}</p>
      <ul v-if="sources.length" class="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3">
        <li v-for="(source, index) in sources" :key="source.url" class="min-w-0 rounded-xl border border-gray-200 p-2 dark:border-dark-600">
          <img :src="source.url" :alt="source.file.name" class="h-20 w-full rounded-lg object-contain" />
          <p class="mt-1 truncate text-xs">{{ index + 1 }}. {{ source.file.name }}</p>
          <button type="button" class="btn btn-secondary mt-2 w-full text-xs" :disabled="loading || validating" @click="removeSource(index)">{{ t('images.forms.edit.removeImage') }}</button>
        </li>
      </ul>
    </div>
    <div>
      <label class="input-label mb-1.5 block" for="image-edit-mask-input">{{ t('images.forms.edit.maskImage') }}</label>
      <input id="image-edit-mask-input" accept="image/png" class="input w-full" data-testid="image-edit-mask-input" type="file" :disabled="validating || loading" @change="handleMaskChange" />
      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('images.forms.edit.maskImageHint') }}</p>
      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">{{ t('images.forms.edit.maskScopeHint') }}</p>
      <div v-if="mask" class="mt-3 flex items-center gap-3">
        <img :src="mask.url" :alt="mask.file.name" class="h-20 w-20 rounded-lg object-contain" />
        <span class="min-w-0 truncate text-xs">{{ mask.file.name }}</span>
        <button type="button" class="btn btn-secondary" :disabled="loading || validating" @click="removeMask">{{ t('images.forms.edit.removeMask') }}</button>
      </div>
    </div>
    <ImageFormControls ref="controls" id="image-edit" :initial-values="initialValues" :models="models" />
    <p v-if="showApiKeyRequiredMessage" role="status" class="text-sm text-amber-700 dark:text-amber-300">{{ t('images.forms.generate.apiKeyRequired') }}</p>
    <p v-if="validationErrorKey || maskErrorKey" role="alert" class="text-sm text-red-700 dark:text-red-300">{{ t(validationErrorKey || maskErrorKey) }}</p>
    <p v-if="validating" role="status" class="text-sm text-gray-500">{{ t('images.forms.edit.validating') }}</p>
    <div class="flex justify-end">
      <button class="btn bg-primary-700 text-white hover:bg-primary-800" :disabled="disabled || loading || validating || !models.length" data-testid="image-edit-submit" type="submit">
        {{ loading ? t('images.forms.edit.submittingWithSeconds', { seconds: loadingSeconds }) : t('images.forms.edit.submit') }}
      </button>
    </div>
  </form>
</template>
<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ImageFormControls from './ImageFormControls.vue'
import type { ImageCommonFormValues } from '@/composables/useImageFormOptions'
import { validateMaskImage, validateSourceImage } from '@/utils/imageFileValidation'
import type { ImageDimensions } from '@/utils/imageFileValidation'
const props = withDefaults(defineProps<{ disabled?: boolean; initialValues?: Partial<ImageCommonFormValues>; initialFiles?: File[]; models?: string[]; loading?: boolean; loadingSeconds?: number; showApiKeyRequiredMessage?: boolean }>(), { disabled: false, initialValues: () => ({}), initialFiles: () => [], models: () => [], loading: false, loadingSeconds: 0, showApiKeyRequiredMessage: false })
const emit = defineEmits<{ (e: 'submit', payload: FormData): void; (e: 'reset'): void }>()
const { t } = useI18n()
const controls = ref<InstanceType<typeof ImageFormControls>>()
const sources = ref<{ file: File; url: string; dimensions: ImageDimensions }[]>([])
const mask = ref<{ file: File; url: string } | null>(null)
const validating = ref(false)
const validationErrorKey = ref('')
const maskErrorKey = ref('')
let validationId = 0
let validationController = new AbortController()
function errorKey(error: unknown) { return error instanceof Error && error.message.startsWith('images.') ? error.message : 'images.forms.edit.sourceImageDecode' }
function clearSources() { sources.value.forEach(source => URL.revokeObjectURL(source.url)); sources.value = [] }
function removeMask() { if (mask.value) URL.revokeObjectURL(mask.value.url); mask.value = null; maskErrorKey.value = '' }
async function checkMask(id: number) {
  maskErrorKey.value = ''
  if (!mask.value) return
  try { await validateMaskImage(mask.value.file, sources.value[0]?.dimensions, validationController.signal) }
  catch (error) { if (id === validationId) maskErrorKey.value = errorKey(error) }
}
async function addSources(files: File[], replace = false) {
  const id = ++validationId
  validationController.abort()
  validationController = new AbortController()
  validating.value = true
  validationErrorKey.value = ''
  if (replace) clearSources()
  try {
    if (files.length + sources.value.length > 16) throw new Error('images.forms.edit.sourceImageLimit')
    const checked = []
    for (const file of files) checked.push({ file, dimensions: await validateSourceImage(file, validationController.signal) })
    if (id !== validationId) return
    sources.value.push(...checked.map(item => ({ ...item, url: URL.createObjectURL(item.file) })))
    await checkMask(id)
  } catch (error) { if (id === validationId) validationErrorKey.value = errorKey(error) }
  finally { if (id === validationId) validating.value = false }
}
function handleSourceChange(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (files.length) void addSources(files)
}
async function handleMaskChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  removeMask()
  mask.value = { file, url: URL.createObjectURL(file) }
  const id = ++validationId
  validating.value = true
  await checkMask(id)
  if (id === validationId) validating.value = false
}
async function removeSource(index: number) {
  URL.revokeObjectURL(sources.value[index].url)
  sources.value.splice(index, 1)
  validationErrorKey.value = ''
  const id = ++validationId
  validating.value = true
  await checkMask(id)
  if (id === validationId) validating.value = false
}
watch(() => props.initialFiles, files => { void addSources(files, true) }, { immediate: true })
function handleSubmit() {
  if (props.disabled || props.loading || validating.value || !props.models.length) return
  const fields = controls.value?.getPayload()
  if (!fields) return
  if (!sources.value.length) { validationErrorKey.value = 'images.forms.edit.sourceImageRequired'; return }
  if (validationErrorKey.value || maskErrorKey.value) return
  const payload = new FormData()
  Object.entries(fields).forEach(([key, value]) => { if (value !== undefined) payload.append(key, String(value)) })
  sources.value.forEach(source => payload.append('image[]', source.file))
  if (mask.value) payload.append('mask', mask.value.file)
  emit('submit', payload)
}
function reset() {
  validationId++
  validationController.abort()
  validationController = new AbortController()
  validating.value = false
  clearSources()
  removeMask()
  validationErrorKey.value = ''
  controls.value?.reset()
  emit('reset')
}
onBeforeUnmount(() => { validationId++; validationController.abort(); clearSources(); removeMask() })
defineExpose({ reset })
</script>
