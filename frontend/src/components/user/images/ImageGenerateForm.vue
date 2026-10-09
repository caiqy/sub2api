<template>
  <form class="grid gap-5" data-testid="image-generate-form" @submit.prevent="handleSubmit">
    <ImageFormControls ref="controls" id="image-generate" :initial-values="initialValues" :models="models" />
    <p v-if="showApiKeyRequiredMessage" role="status" class="text-sm text-amber-700 dark:text-amber-300">{{ t('images.forms.generate.apiKeyRequired') }}</p>
    <div class="flex justify-end">
      <button class="btn bg-primary-700 text-white hover:bg-primary-800" :disabled="disabled || loading || !models.length" data-testid="image-generate-submit" type="submit">
        {{ loading ? t('images.forms.generate.submittingWithSeconds', { seconds: loadingSeconds }) : t('images.forms.generate.submit') }}
      </button>
    </div>
  </form>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ImageFormControls from './ImageFormControls.vue'
import type { ImageCommonFormValues } from '@/composables/useImageFormOptions'
import type { ImageGenerationRequest } from '@/types'
const props = withDefaults(defineProps<{ disabled?: boolean; initialValues?: Partial<ImageCommonFormValues>; models?: string[]; loading?: boolean; loadingSeconds?: number; showApiKeyRequiredMessage?: boolean }>(), { disabled: false, initialValues: () => ({}), models: () => [], loading: false, loadingSeconds: 0, showApiKeyRequiredMessage: false })
const emit = defineEmits<{ (e: 'submit', payload: ImageGenerationRequest): void; (e: 'reset'): void }>()
const { t } = useI18n()
const controls = ref<InstanceType<typeof ImageFormControls>>()
function handleSubmit() {
  if (props.disabled || props.loading || !props.models.length) return
  const payload = controls.value?.getPayload()
  if (payload) emit('submit', payload)
}
function reset() { controls.value?.reset(); emit('reset') }
defineExpose({ reset })
</script>
