<template>
  <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800" data-testid="image-result-panel">
    <div class="space-y-1">
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('images.results.title') }}</h3>
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('images.results.description') }}</p>
    </div>

    <p v-if="formattedDuration" class="mt-2 text-xs font-medium text-gray-500 dark:text-gray-400" data-testid="image-result-duration">
      {{ t('images.results.duration') }}: {{ formattedDuration }}
    </p>

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
            :aria-label="buildIndexedActionLabel('images.results.openPreview', index)"
            :data-testid="`image-result-open-${index}`"
            type="button"
            @click="openPreview(result, index, $event)"
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
              :aria-label="buildIndexedActionLabel('images.results.openPreview', index)"
              type="button"
              @click="openPreview(result, index, $event)"
            >
              {{ t('images.results.openPreview') }}
            </button>
            <a
              class="rounded-full bg-black/65 px-3 py-1.5 text-xs font-medium text-white shadow-sm backdrop-blur transition hover:bg-black/80"
              :aria-label="buildIndexedActionLabel('images.results.download', index)"
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
      class="fixed inset-0 z-50 flex min-h-screen items-center justify-center bg-black/95 p-4 backdrop-blur-sm sm:p-6"
      :aria-label="t('images.results.previewTitle')"
      aria-modal="true"
      data-testid="image-result-preview-modal"
      role="dialog"
      @keydown="handleModalKeydown"
      @click.self="closePreview"
    >
      <div class="absolute right-4 top-4 z-10 flex items-center gap-2 sm:right-6 sm:top-6">
        <a
          ref="previewDownloadRef"
          class="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm font-medium text-white shadow-lg backdrop-blur transition hover:border-white/30 hover:bg-white/20"
          :aria-label="buildIndexedActionLabel('images.results.download', selectedPreview.index)"
          data-testid="image-result-preview-modal-download"
          :download="buildDownloadFilename(selectedPreview.result, selectedPreview.index)"
          :href="selectedPreview.result.src"
          @click.stop
        >
          {{ t('images.results.download') }}
        </a>
        <button
          ref="previewCloseButtonRef"
          class="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm font-medium text-white shadow-lg backdrop-blur transition hover:border-white/30 hover:bg-white/20"
          :aria-label="t('images.results.closePreview')"
          data-testid="image-result-preview-close"
          type="button"
          @click="closePreview"
        >
          {{ t('images.results.closePreview') }}
        </button>
      </div>

      <img
        :src="selectedPreview.result.src"
        :alt="t('images.results.previewTitle')"
        class="max-h-[calc(100vh-6rem)] max-w-[calc(100vw-2rem)] object-contain shadow-2xl sm:max-w-[calc(100vw-3rem)]"
        data-testid="image-result-preview-modal-image"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import type { ImageResultPreview } from '@/composables/useImageGeneration'
import { formatImageDuration } from '@/utils/imageDuration'
import { sanitizeUrl } from '@/utils/url'

const props = defineProps<{
  loading: boolean
  error: string
  results: ImageResultPreview[]
  durationMs?: number | null
}>()

const { t } = useI18n()
const selectedPreview = ref<{ result: ImageResultPreview, index: number } | null>(null)
const previewCloseButtonRef = ref<HTMLButtonElement | null>(null)
const previewDownloadRef = ref<HTMLAnchorElement | null>(null)
const lastPreviewTrigger = ref<HTMLElement | null>(null)
let hasPreviewKeydownListener = false

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

const formattedDuration = computed(() => formatImageDuration(props.durationMs))

function buildIndexedActionLabel(labelKey: string, index: number): string {
  return `${t(labelKey)} ${index + 1}`
}

async function openPreview(result: ImageResultPreview, index: number, event?: MouseEvent) {
  lastPreviewTrigger.value = event?.currentTarget instanceof HTMLElement ? event.currentTarget : null
  selectedPreview.value = { result, index }
  await nextTick()
  previewCloseButtonRef.value?.focus()
}

async function closePreview() {
  const returnFocusTarget = lastPreviewTrigger.value
  selectedPreview.value = null
  await nextTick()
  if (returnFocusTarget?.isConnected) {
    returnFocusTarget.focus()
  }
}

function handlePreviewKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && selectedPreview.value) {
    closePreview()
  }
}

function syncPreviewKeydownListener(shouldListen: boolean) {
  if (shouldListen && !hasPreviewKeydownListener) {
    window.addEventListener('keydown', handlePreviewKeydown)
    hasPreviewKeydownListener = true
    return
  }

  if (!shouldListen && hasPreviewKeydownListener) {
    window.removeEventListener('keydown', handlePreviewKeydown)
    hasPreviewKeydownListener = false
  }
}

function handleModalKeydown(event: KeyboardEvent) {
  if (event.key !== 'Tab') {
    return
  }

  const focusableControls = [previewDownloadRef.value, previewCloseButtonRef.value].filter(
    (element): element is HTMLAnchorElement | HTMLButtonElement => element !== null
  )

  if (focusableControls.length === 0) {
    return
  }

  event.preventDefault()

  const activeElement = document.activeElement as HTMLElement | null
  const currentIndex = focusableControls.findIndex((element) => element === activeElement)
  const nextIndex = event.shiftKey
    ? (currentIndex <= 0 ? focusableControls.length - 1 : currentIndex - 1)
    : (currentIndex + 1) % focusableControls.length

  focusableControls[nextIndex]?.focus()
}

watch(selectedPreview, (preview) => {
  syncPreviewKeydownListener(preview !== null)
})

onBeforeUnmount(() => {
  syncPreviewKeydownListener(false)
})

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
