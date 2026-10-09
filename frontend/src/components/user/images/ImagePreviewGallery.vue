<template>
  <div class="grid grid-cols-1 gap-4" data-testid="image-preview-gallery" v-bind="$attrs">
    <figure v-for="(image, index) in displayImages" :key="index" class="overflow-hidden rounded-2xl border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-900/60">
      <div class="relative">
        <button class="block w-full overflow-hidden bg-white transition hover:opacity-95 focus:outline-none focus:ring-2 focus:ring-primary-400 dark:bg-dark-950" :aria-label="actionLabel('images.results.openPreview', index)" :data-testid="`image-preview-open-${index}`" type="button" @click="openPreview(image, index, $event)">
          <img :src="image.src" :alt="t('images.results.previewTitle')" class="max-h-[70vh] min-h-[240px] w-full bg-white object-contain dark:bg-dark-950 sm:min-h-[360px]" :data-testid="`${imageTestIdPrefix}-${index}`" loading="lazy" decoding="async" @error="failedImages.add(index)" />
        </button>
        <div class="absolute right-3 top-3 flex flex-wrap justify-end gap-2">
          <button class="image-action" :aria-label="actionLabel('images.results.openPreview', index)" type="button" @click="openPreview(image, index, $event)">{{ t('images.results.openPreview') }}</button>
          <a class="image-action" :aria-label="actionLabel('images.results.download', index)" :data-testid="`image-preview-download-${index}`" :download="imageDownloadFilename(image.src, index)" :href="image.src" @click="downloadImage($event, image, index)">{{ t('images.results.download') }}</a>
          <button v-if="allowEdit" class="image-action" :data-testid="`image-preview-edit-${index}`" :disabled="editDisabled || failedImages.has(index)" type="button" @click="emit('edit', image)">{{ t('images.results.sendToEdit') }}</button>
        </div>
      </div>
      <p v-if="failedImages.has(index)" class="px-4 py-3 text-sm text-amber-800 dark:text-amber-300" role="status">{{ t('images.results.imageUnavailable') }}</p>
      <figcaption v-if="image.revisedPrompt" class="border-t border-gray-200 px-4 py-3 text-sm text-gray-600 dark:border-dark-700 dark:text-gray-300">
        <span class="font-medium text-gray-900 dark:text-white">{{ t('images.results.revisedPrompt') }}:</span> {{ image.revisedPrompt }}
      </figcaption>
    </figure>
    <div v-if="actionError" class="rounded-xl bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-900/20 dark:text-amber-300" role="alert">
      <p>{{ actionError }}</p>
      <a v-if="failedUrl" class="mt-2 inline-block underline" :href="failedUrl" target="_blank" rel="noopener noreferrer">{{ t('images.results.openOriginal') }}</a>
    </div>
  </div>
  <div v-if="selectedPreview" class="fixed inset-0 z-50 flex min-h-screen items-center justify-center bg-black/95 p-4 backdrop-blur-sm sm:p-6" :aria-label="t('images.results.previewTitle')" aria-modal="true" data-testid="image-preview-modal" role="dialog" @keydown="handleModalKeydown" @click.self="closePreview">
    <div class="absolute right-4 top-4 z-10 flex items-center gap-2 sm:right-6 sm:top-6">
      <a ref="previewDownloadRef" class="image-action" :aria-label="actionLabel('images.results.download', selectedPreview.index)" data-testid="image-preview-modal-download" :download="imageDownloadFilename(selectedPreview.image.src, selectedPreview.index)" :href="selectedPreview.image.src" @click="downloadImage($event, selectedPreview.image, selectedPreview.index)">{{ t('images.results.download') }}</a>
      <button ref="previewCloseButtonRef" class="image-action" :aria-label="t('images.results.closePreview')" data-testid="image-preview-close" type="button" @click="closePreview">{{ t('images.results.closePreview') }}</button>
    </div>
    <p v-if="actionError" class="absolute bottom-5 max-w-lg rounded-xl bg-amber-100 p-3 text-sm text-amber-900" role="alert">{{ actionError }}</p>
    <img :src="selectedPreview.image.src" :alt="t('images.results.previewTitle')" class="max-h-[calc(100vh-6rem)] max-w-[calc(100vw-2rem)] object-contain shadow-2xl sm:max-w-[calc(100vw-3rem)]" data-testid="image-preview-modal-image" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { imageDownloadFilename, imageToFile } from '@/utils/imageResult'
import { sanitizeUrl } from '@/utils/url'

export interface ImagePreviewGalleryItem {
  src: string
  revisedPrompt?: string
  source: 'data-url' | 'url'
}
defineOptions({ inheritAttrs: false })
const props = withDefaults(defineProps<{
  images: ImagePreviewGalleryItem[]
  imageTestIdPrefix?: string
  allowEdit?: boolean
  editDisabled?: boolean
}>(), { imageTestIdPrefix: 'image-preview-image', allowEdit: false, editDisabled: false })
const emit = defineEmits<{ (e: 'edit', image: ImagePreviewGalleryItem): void }>()
const { t } = useI18n()
const selectedPreview = ref<{ image: ImagePreviewGalleryItem, index: number } | null>(null)
const previewCloseButtonRef = ref<HTMLButtonElement | null>(null)
const previewDownloadRef = ref<HTMLAnchorElement | null>(null)
const failedImages = ref(new Set<number>())
const actionError = ref('')
const failedUrl = ref('')
let lastPreviewTrigger: HTMLElement | null = null
let downloadController: AbortController | null = null
const downloadUrls = new Set<string>()
const displayImages = computed(() => props.images.flatMap(image => {
  const src = sanitizeUrl(image.src, { allowDataUrl: image.source === 'data-url' })
  return src ? [{ ...image, src }] : []
}))
function actionLabel(key: string, index: number) { return `${t(key)} ${index + 1}` }
async function openPreview(image: ImagePreviewGalleryItem, index: number, event?: MouseEvent) {
  lastPreviewTrigger = event?.currentTarget instanceof HTMLElement ? event.currentTarget : null
  selectedPreview.value = { image, index }
  await nextTick()
  previewCloseButtonRef.value?.focus()
}
async function closePreview() {
  selectedPreview.value = null
  await nextTick()
  if (lastPreviewTrigger?.isConnected) lastPreviewTrigger.focus()
}
function handlePreviewKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  event.preventDefault()
  event.stopPropagation()
  closePreview()
}
function handleModalKeydown(event: KeyboardEvent) {
  if (event.key !== 'Tab') return
  const controls = [previewDownloadRef.value, previewCloseButtonRef.value].filter((control): control is HTMLAnchorElement | HTMLButtonElement => control !== null)
  if (!controls.length) return
  event.preventDefault()
  const index = controls.findIndex(control => control === document.activeElement)
  controls[(index + (event.shiftKey ? controls.length - 1 : 1)) % controls.length]?.focus()
}
async function downloadImage(event: MouseEvent, image: ImagePreviewGalleryItem, index: number) {
  if (image.source === 'data-url') return
  event.preventDefault()
  downloadController?.abort()
  const controller = new AbortController()
  downloadController = controller
  actionError.value = ''
  failedUrl.value = ''
  try {
    const file = await imageToFile(image.src, controller.signal)
    if (controller.signal.aborted) return
    const url = URL.createObjectURL(file)
    downloadUrls.add(url)
    const link = document.createElement('a')
    link.href = url
    link.download = file.name.replace('sub2api-image.', `sub2api-image-${index + 1}.`)
    document.body.appendChild(link)
    link.click()
    link.remove()
    setTimeout(() => { URL.revokeObjectURL(url); downloadUrls.delete(url) }, 1000)
  } catch {
    if (controller.signal.aborted) return
    actionError.value = t('images.results.readFailed')
    failedUrl.value = image.src
  }
}
watch(selectedPreview, (preview, previous) => {
  if (preview && !previous) window.addEventListener('keydown', handlePreviewKeydown, true)
  if (!preview && previous) window.removeEventListener('keydown', handlePreviewKeydown, true)
})
watch(() => props.images, () => { failedImages.value = new Set(); actionError.value = ''; failedUrl.value = '' })
onBeforeUnmount(() => {
  window.removeEventListener('keydown', handlePreviewKeydown, true)
  downloadController?.abort()
  downloadUrls.forEach(url => URL.revokeObjectURL(url))
})
</script>

<style scoped>
.image-action { @apply rounded-full bg-black/70 px-3 py-2 text-xs font-medium text-white shadow-sm transition hover:bg-black/90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-400 disabled:cursor-not-allowed disabled:opacity-50; }
</style>
