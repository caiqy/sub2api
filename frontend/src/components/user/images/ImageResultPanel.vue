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

    <div v-else class="mt-4 grid grid-cols-1 gap-4" data-testid="image-result-grid">
      <figure v-for="(result, index) in displayResults" :key="`${result.source}-${index}-${result.src}`" class="overflow-hidden rounded-2xl border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-900/60">
        <div class="relative">
          <button
            class="block w-full overflow-hidden bg-white transition hover:opacity-95 focus:outline-none focus:ring-2 focus:ring-primary-400 dark:bg-dark-950"
            :aria-label="t('images.results.openPreview')"
            :data-testid="`image-result-open-${index}`"
            type="button"
            @click="openPreview(result)"
          >
            <img
              :src="result.src"
              :alt="`preview-${index + 1}`"
              class="max-h-[70vh] min-h-[240px] w-full bg-white object-contain dark:bg-dark-950 sm:min-h-[360px]"
              :data-testid="`image-result-preview-${index}`"
            />
          </button>

          <div class="absolute right-3 top-3 flex gap-2">
            <button
              class="rounded-full bg-black/65 px-3 py-1.5 text-xs font-medium text-white shadow-sm backdrop-blur transition hover:bg-black/80"
              :aria-label="t('images.results.openPreview')"
              type="button"
              @click="openPreview(result)"
            >
              {{ t('images.results.openPreview') }}
            </button>
            <a
              class="rounded-full bg-black/65 px-3 py-1.5 text-xs font-medium text-white shadow-sm backdrop-blur transition hover:bg-black/80"
              :aria-label="t('images.results.download')"
              :data-testid="`image-result-download-${index}`"
              :download="buildDownloadFilename(result, index)"
              :href="result.src"
              @click.stop
            >
              {{ t('images.results.download') }}
            </a>
          </div>
        </div>

        <figcaption v-if="result.revisedPrompt" class="border-t border-gray-200 px-4 py-3 text-sm text-gray-600 dark:border-dark-700 dark:text-gray-300">
          <span class="font-medium text-gray-900 dark:text-white">{{ t('images.results.revisedPrompt') }}:</span>
          {{ result.revisedPrompt }}
        </figcaption>
      </figure>
    </div>

    <div
      v-if="selectedPreview"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-6 backdrop-blur-sm"
      data-testid="image-result-preview-modal"
      role="dialog"
      @click.self="closePreview"
    >
      <div class="relative w-full max-w-5xl overflow-hidden rounded-3xl border border-white/10 bg-slate-950 shadow-2xl">
        <div class="flex items-center justify-between border-b border-white/10 px-5 py-4 text-white">
          <h4 class="text-sm font-semibold uppercase tracking-[0.18em]">{{ t('images.results.previewTitle') }}</h4>
          <button
            class="rounded-full border border-white/15 px-3 py-1.5 text-xs font-medium text-white transition hover:border-white/30 hover:bg-white/10"
            :aria-label="t('images.results.closePreview')"
            data-testid="image-result-preview-close"
            type="button"
            @click="closePreview"
          >
            {{ t('images.results.closePreview') }}
          </button>
        </div>

        <div class="flex max-h-[80vh] items-center justify-center bg-black px-4 py-4">
          <img
            :src="selectedPreview.src"
            :alt="t('images.results.previewTitle')"
            class="max-h-[72vh] w-auto max-w-full object-contain"
            data-testid="image-result-preview-modal-image"
          />
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import type { ImageResultPreview } from '@/composables/useImageGeneration'
import { sanitizeUrl } from '@/utils/url'

const props = defineProps<{
  loading: boolean
  error: string
  results: ImageResultPreview[]
}>()

const { t } = useI18n()
const selectedPreview = ref<ImageResultPreview | null>(null)

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

function openPreview(result: ImageResultPreview) {
  selectedPreview.value = result
}

function closePreview() {
  selectedPreview.value = null
}

function buildDownloadFilename(result: ImageResultPreview, index: number): string {
  if (result.source === 'data-url') {
    const mimeMatch = result.src.match(/^data:(image\/[a-z0-9.+-]+);/i)
    const extension = mimeMatch?.[1]?.split('/')[1]?.replace('jpeg', 'jpg') ?? 'png'
    return `sub2api-image-${index + 1}.${extension}`
  }

  const pathname = (() => {
    try {
      return new URL(result.src).pathname
    } catch {
      return ''
    }
  })()

  const extensionMatch = pathname.match(/\.([a-z0-9]+)$/i)
  const extension = extensionMatch?.[1] ?? 'png'
  return `sub2api-image-${index + 1}.${extension}`
}
</script>
