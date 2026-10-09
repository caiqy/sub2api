<template>
  <section class="min-w-0 rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800" data-testid="image-result-panel" :aria-busy="loading">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('images.results.title') }}</h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('images.results.description') }}</p>
      </div>
      <p class="rounded-full bg-gray-100 px-3 py-1 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300" role="status" aria-live="polite">{{ t(`images.results.states.${status ?? (loading ? 'generating' : results.length ? 'success' : 'idle')}`) }}</p>
    </div>
    <p v-if="formattedDuration" class="mt-3 text-xs tabular-nums text-gray-500 dark:text-gray-400" data-testid="image-result-duration">{{ t('images.results.duration') }}: {{ formattedDuration }}</p>
    <div v-if="loading" class="mt-4 flex items-center justify-between gap-3 text-sm">
      <p class="flex items-center gap-2 text-gray-600 dark:text-gray-300"><span class="h-4 w-4 rounded-full border-2 border-gray-200 border-t-primary-600 motion-safe:animate-spin" aria-hidden="true" />{{ t('images.results.loading') }}</p>
      <button type="button" class="btn btn-secondary shrink-0" data-testid="image-stop-waiting" @click="emit('stop')">{{ t('images.results.stop') }}</button>
    </div>
    <p v-if="loading || status === 'stopped'" class="mt-3 text-xs leading-5 text-amber-800 dark:text-amber-300">{{ t('images.results.stopNotice') }}</p>
    <div v-if="error" class="mt-4 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-900/10 dark:text-red-300" role="alert"><p class="font-medium">{{ t('images.results.errorTitle') }}</p><p class="mt-1 break-words">{{ error }}</p></div>
    <figure v-if="partialSrc && results.length === 0" class="mt-4 overflow-hidden rounded-xl border border-gray-200 dark:border-dark-700" data-testid="image-partial-preview">
      <img :src="partialSrc" :alt="t('images.results.draft')" class="max-h-[70vh] min-h-[240px] w-full object-contain" decoding="async" />
      <figcaption class="bg-gray-50 px-4 py-2 text-xs text-gray-600 dark:bg-dark-900 dark:text-gray-300">{{ t('images.results.draft') }}</figcaption>
    </figure>
    <ImagePreviewGallery v-if="results.length" class="mt-4" :images="results" image-test-id-prefix="image-result-preview" allow-edit :edit-disabled="loading || editDisabled" data-testid="image-result-grid" @edit="emit('edit', $event)" />
    <div v-else-if="!partialSrc && !error" class="mt-5 flex min-h-[340px] flex-col items-center justify-center rounded-xl border border-dashed border-gray-300 bg-gray-50 px-6 text-center dark:border-dark-600 dark:bg-dark-900/50">
      <svg class="mb-4 h-12 w-12 text-gray-300 dark:text-dark-600" viewBox="0 0 48 48" fill="none" aria-hidden="true"><rect x="5" y="5" width="38" height="38" rx="7" stroke="currentColor" stroke-width="2"/><circle cx="17" cy="17" r="4" stroke="currentColor" stroke-width="2"/><path d="m8 38 11-12 8 7 6-8 8 13" stroke="currentColor" stroke-width="2" stroke-linejoin="round"/></svg>
      <p class="max-w-sm text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('images.results.empty') }}</p>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ImagePreviewGallery from './ImagePreviewGallery.vue'
import type { ImageResultPreview } from '@/composables/useImageGeneration'
import { formatImageDuration } from '@/utils/imageDuration'
import { sanitizeUrl } from '@/utils/url'
const props = defineProps<{
  loading: boolean
  error: string
  results: ImageResultPreview[]
  durationMs?: number | null
  status?: string
  partial?: ImageResultPreview | null
  editDisabled?: boolean
}>()
const emit = defineEmits<{ (e: 'stop'): void; (e: 'edit', image: ImageResultPreview): void }>()
const { t } = useI18n()
const formattedDuration = computed(() => formatImageDuration(props.durationMs))
const partialSrc = computed(() => props.partial ? sanitizeUrl(props.partial.src, { allowDataUrl: props.partial.source === 'data-url' }) : '')
</script>
