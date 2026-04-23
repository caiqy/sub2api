<template>
  <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800" data-testid="image-result-panel">
    <div class="space-y-1">
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('images.results.title') }}</h3>
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('images.results.description') }}</p>
    </div>

    <p v-if="loading" class="mt-4 text-sm text-gray-600 dark:text-gray-300">{{ t('images.results.loading') }}</p>

    <div v-else-if="error" class="mt-4 rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-900/10 dark:text-red-300" role="alert">
      <p class="font-medium">{{ t('images.results.errorTitle') }}</p>
      <p class="mt-1">{{ error }}</p>
    </div>

    <p v-else-if="displayResults.length === 0" class="mt-4 rounded-2xl border border-dashed border-gray-300 bg-gray-50 px-4 py-6 text-sm text-gray-500 dark:border-dark-600 dark:bg-dark-900/60 dark:text-gray-400">
      {{ t('images.results.empty') }}
    </p>

    <div v-else class="mt-4 grid gap-4 sm:grid-cols-2">
      <figure v-for="(result, index) in displayResults" :key="`${result.source}-${index}-${result.src}`" class="overflow-hidden rounded-2xl border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-900/60">
        <img
          :src="result.src"
          :alt="`preview-${index + 1}`"
          class="aspect-square w-full bg-white object-contain dark:bg-dark-950"
          :data-testid="`image-result-preview-${index}`"
        />
        <figcaption v-if="result.revisedPrompt" class="border-t border-gray-200 px-4 py-3 text-sm text-gray-600 dark:border-dark-700 dark:text-gray-300">
          <span class="font-medium text-gray-900 dark:text-white">{{ t('images.results.revisedPrompt') }}:</span>
          {{ result.revisedPrompt }}
        </figcaption>
      </figure>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import type { ImageResultPreview } from '@/composables/useImageGeneration'
import { sanitizeUrl } from '@/utils/url'

const props = defineProps<{
  loading: boolean
  error: string
  results: ImageResultPreview[]
}>()

const { t } = useI18n()

const displayResults = computed(() => props.results.flatMap((result) => {
  if (result.source !== 'url') {
    return [result]
  }

  const safeSrc = sanitizeUrl(result.src)
  if (!safeSrc) {
    return []
  }

  return [{
    ...result,
    src: safeSrc,
  }]
}))
</script>
