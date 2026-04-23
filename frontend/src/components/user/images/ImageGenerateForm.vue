<template>
  <form class="grid gap-5" data-testid="image-generate-form" @submit.prevent="handleSubmit">
    <div class="grid gap-5 lg:grid-cols-2">
      <div class="lg:col-span-2">
        <label class="input-label mb-1.5 block" for="image-generate-prompt">{{ t('images.forms.generate.prompt') }}</label>
        <textarea
          id="image-generate-prompt"
          v-model="form.prompt"
          class="input min-h-[120px] w-full resize-y"
          :placeholder="t('images.forms.generate.promptPlaceholder')"
          data-testid="image-generate-prompt"
          required
          @input="handlePromptInput"
        ></textarea>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-generate-model">{{ t('images.forms.generate.model') }}</label>
        <select id="image-generate-model" v-model="form.model" :class="selectClass">
          <option v-for="option in modelOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-generate-size">{{ t('images.forms.generate.size') }}</label>
        <select id="image-generate-size" v-model="form.size" :class="selectClass">
          <option v-for="option in sizeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-generate-quality">{{ t('images.forms.generate.quality') }}</label>
        <select id="image-generate-quality" v-model="form.quality" :class="selectClass">
          <option v-for="option in qualityOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-generate-background">{{ t('images.forms.generate.background') }}</label>
        <select id="image-generate-background" v-model="form.background" :class="selectClass">
          <option v-for="option in backgroundOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-generate-output-format">{{ t('images.forms.generate.outputFormat') }}</label>
        <select id="image-generate-output-format" v-model="form.output_format" :class="selectClass">
          <option v-for="option in outputFormatOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-generate-moderation">{{ t('images.forms.generate.moderation') }}</label>
        <select id="image-generate-moderation" v-model="form.moderation" :class="selectClass">
          <option v-for="option in moderationOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>

      <div>
        <label class="input-label mb-1.5 block" for="image-generate-n">{{ t('images.forms.generate.n') }}</label>
        <select id="image-generate-n" v-model.number="form.n" :class="selectClass">
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
      <button
        class="btn btn-primary"
        :disabled="disabled || loading"
        data-testid="image-generate-submit"
        type="button"
        @click="handleSubmit"
      >
        {{ loading ? t('images.forms.generate.submitting') : t('images.forms.generate.submit') }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import { createDefaultImageFormValues, useImageFormOptions } from '@/composables/useImageFormOptions'
import type { ImageCommonFormValues } from '@/composables/useImageFormOptions'
import type { ImageGenerationRequest } from '@/types'

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
  (e: 'submit', payload: ImageGenerationRequest): void
}>()

const { t } = useI18n()
const form = reactive({
  ...createDefaultImageFormValues(),
  ...props.initialValues,
})
const validationError = ref('')
const { backgroundOptions, countOptions, modelOptions, moderationOptions, outputFormatOptions, qualityOptions, sizeOptions } = useImageFormOptions()

const selectClass = 'input w-full'

function handlePromptInput() {
  if (form.prompt.trim()) {
    validationError.value = ''
  }
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

  validationError.value = ''

  emit('submit', {
    background: form.background,
    model: form.model,
    moderation: form.moderation,
    n: Number(form.n),
    output_format: form.output_format,
    prompt,
    quality: form.quality,
    size: form.size,
  })
}
</script>
