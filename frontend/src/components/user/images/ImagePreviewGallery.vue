<template>
  <div class="grid grid-cols-1 gap-4" data-testid="image-preview-gallery">
    <figure
      v-for="(image, index) in displayImages"
      :key="`${image.source}-${index}-${image.src}`"
      class="overflow-hidden rounded-2xl border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-900/60"
    >
      <div class="relative">
        <button
          class="block w-full overflow-hidden bg-white transition hover:opacity-95 focus:outline-none focus:ring-2 focus:ring-primary-400 dark:bg-dark-950"
          :aria-label="buildIndexedActionLabel('images.results.openPreview', index)"
          :data-testid="`image-preview-open-${index}`"
          type="button"
          @click="openPreview(image, index, $event)"
        >
          <img
            :src="image.src"
            :alt="`preview-${index + 1}`"
            class="max-h-[70vh] min-h-[240px] w-full bg-white object-contain dark:bg-dark-950 sm:min-h-[360px]"
            :data-testid="`${imageTestIdPrefix}-${index}`"
          />
        </button>

        <div class="absolute right-3 top-3 flex gap-2">
          <button
            class="rounded-full bg-black/65 px-3 py-1.5 text-xs font-medium text-white shadow-sm backdrop-blur transition hover:bg-black/80"
            :aria-label="buildIndexedActionLabel('images.results.openPreview', index)"
            type="button"
            @click="openPreview(image, index, $event)"
          >
            {{ t('images.results.openPreview') }}
          </button>
          <a
            class="rounded-full bg-black/65 px-3 py-1.5 text-xs font-medium text-white shadow-sm backdrop-blur transition hover:bg-black/80"
            :aria-label="buildIndexedActionLabel('images.results.download', index)"
            :download="buildDownloadFilename(image, index)"
            :href="image.src"
            @click.stop
          >
            {{ t('images.results.download') }}
          </a>
        </div>
      </div>

      <figcaption v-if="image.revisedPrompt" class="border-t border-gray-200 px-4 py-3 text-sm text-gray-600 dark:border-dark-700 dark:text-gray-300">
        <span class="font-medium text-gray-900 dark:text-white">{{ t('images.results.revisedPrompt') }}:</span>
        {{ image.revisedPrompt }}
      </figcaption>
    </figure>
  </div>

  <div
    v-if="selectedPreview"
    class="fixed inset-0 z-50 flex min-h-screen items-center justify-center bg-black/95 p-4 backdrop-blur-sm sm:p-6"
    :aria-label="t('images.results.previewTitle')"
    aria-modal="true"
    data-testid="image-preview-modal"
    role="dialog"
    @keydown="handleModalKeydown"
    @click.self="closePreview"
  >
    <div class="absolute right-4 top-4 z-10 flex items-center gap-2 sm:right-6 sm:top-6">
      <a
        ref="previewDownloadRef"
        class="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm font-medium text-white shadow-lg backdrop-blur transition hover:border-white/30 hover:bg-white/20"
        :aria-label="buildIndexedActionLabel('images.results.download', selectedPreview.index)"
        data-testid="image-preview-modal-download"
        :download="buildDownloadFilename(selectedPreview.image, selectedPreview.index)"
        :href="selectedPreview.image.src"
        @click.stop
      >
        {{ t('images.results.download') }}
      </a>
      <button
        ref="previewCloseButtonRef"
        class="rounded-full border border-white/15 bg-white/10 px-4 py-2 text-sm font-medium text-white shadow-lg backdrop-blur transition hover:border-white/30 hover:bg-white/20"
        :aria-label="t('images.results.closePreview')"
        data-testid="image-preview-close"
        type="button"
        @click="closePreview"
      >
        {{ t('images.results.closePreview') }}
      </button>
    </div>

    <img
      :src="selectedPreview.image.src"
      :alt="t('images.results.previewTitle')"
      class="max-h-[calc(100vh-6rem)] max-w-[calc(100vw-2rem)] object-contain shadow-2xl sm:max-w-[calc(100vw-3rem)]"
      data-testid="image-preview-modal-image"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import { sanitizeUrl } from '@/utils/url'

export interface ImagePreviewGalleryItem {
  src: string
  revisedPrompt?: string
  source: 'data-url' | 'url'
}

const props = withDefaults(defineProps<{
  images: ImagePreviewGalleryItem[]
  imageTestIdPrefix?: string
}>(), {
  imageTestIdPrefix: 'image-preview-image',
})

const { t } = useI18n()
const selectedPreview = ref<{ image: ImagePreviewGalleryItem, index: number } | null>(null)
const previewCloseButtonRef = ref<HTMLButtonElement | null>(null)
const previewDownloadRef = ref<HTMLAnchorElement | null>(null)
const lastPreviewTrigger = ref<HTMLElement | null>(null)
let hasPreviewKeydownListener = false

const imageTestIdPrefix = computed(() => props.imageTestIdPrefix)

const displayImages = computed(() => props.images.flatMap((image) => {
  const safeSrc = image.source === 'data-url'
    ? sanitizeUrl(image.src, { allowDataUrl: true })
    : sanitizeUrl(image.src)

  if (!safeSrc) {
    return []
  }

  return [{
    ...image,
    src: safeSrc,
  }]
}))

function buildIndexedActionLabel(labelKey: string, index: number): string {
  return `${t(labelKey)} ${index + 1}`
}

async function openPreview(image: ImagePreviewGalleryItem, index: number, event?: MouseEvent) {
  lastPreviewTrigger.value = event?.currentTarget instanceof HTMLElement ? event.currentTarget : null
  selectedPreview.value = { image, index }
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
    // 在嵌套于 BaseDialog 等父级弹窗时，优先消费 Escape，避免父子弹窗被同一次按键一起关闭。
    event.preventDefault()
    event.stopPropagation()
    closePreview()
  }
}

function syncPreviewKeydownListener(shouldListen: boolean) {
  if (shouldListen && !hasPreviewKeydownListener) {
    window.addEventListener('keydown', handlePreviewKeydown, true)
    hasPreviewKeydownListener = true
    return
  }

  if (!shouldListen && hasPreviewKeydownListener) {
    window.removeEventListener('keydown', handlePreviewKeydown, true)
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

function buildDownloadFilename(image: ImagePreviewGalleryItem, index: number): string {
  if (image.source === 'data-url') {
    const mimeMatch = image.src.match(/^data:(image\/[a-z0-9.+-]+);/i)
    const extension = mimeMatch?.[1]?.split('/')[1]?.replace('jpeg', 'jpg') ?? 'png'
    return `sub2api-image-${index + 1}.${extension}`
  }

  const pathname = (() => {
    try {
      return new URL(image.src).pathname
    } catch {
      return ''
    }
  })()

  const extensionMatch = pathname.match(/\.([a-z0-9]+)$/i)
  const extension = extensionMatch?.[1] ?? 'png'
  return `sub2api-image-${index + 1}.${extension}`
}
</script>
