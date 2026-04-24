<template>
  <BaseDialog :show="show" :title="t('admin.usage.viewDetail')" width="full" @close="emit('close')">
    <div class="space-y-4">
      <div class="grid gap-3 rounded-lg border border-gray-200 p-4 text-sm dark:border-dark-700 md:grid-cols-2 xl:grid-cols-4">
        <div>
          <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.requestId') }}</div>
          <div class="break-all font-mono text-gray-900 dark:text-white">{{ usageLog?.request_id || '-' }}</div>
        </div>
        <div>
          <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.usage.user') }}</div>
          <div class="break-all text-gray-900 dark:text-white">{{ usageLog?.user?.username || usageLog?.user?.email || '-' }}</div>
        </div>
        <div>
          <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('usage.model') }}</div>
          <div class="break-all text-gray-900 dark:text-white">{{ usageLog?.model || '-' }}</div>
        </div>
        <div>
          <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('usage.time') }}</div>
          <div class="text-gray-900 dark:text-white">{{ detail?.created_at || usageLog?.created_at || '-' }}</div>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          :data-test="`tab-${tab.key}`"
          type="button"
          class="rounded-lg px-3 py-2 text-sm transition-colors"
          :class="activeTab === tab.key ? 'bg-primary-600 text-white' : 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>

        <button
          data-test="copy-current-tab"
          type="button"
          class="btn btn-secondary ml-auto"
          :disabled="!activeContent"
          @click="copyCurrentTab"
        >
          {{ copied ? t('common.copied') : t('common.copy') }}
        </button>
      </div>

      <div data-test="detail-content-panel" class="h-[60vh] overflow-auto rounded-lg" :class="{ 'bg-gray-50 dark:bg-dark-900': !loading && !error && activeContent }">
        <div v-if="loading" class="h-full rounded-lg border border-dashed border-gray-200 p-6 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
          {{ t('common.loading') }}
        </div>

        <div v-else-if="error" class="h-full rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-900/40 dark:bg-red-900/10">
          <div class="text-sm text-red-700 dark:text-red-300">{{ error }}</div>
          <button data-test="usage-detail-retry" type="button" class="btn btn-secondary mt-3" @click="emit('retry')">
            {{ t('common.retry') }}
          </button>
        </div>

        <div v-else-if="activeContent && activeImagePreviews.length > 0" class="space-y-4 p-4">
          <section class="space-y-3">
            <h4 class="text-xs font-semibold uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">{{ t('admin.usage.imagePreview') }}</h4>
            <div class="grid gap-4 sm:grid-cols-2">
              <figure
                v-for="(preview, index) in activeImagePreviews"
                :key="`${preview.src}-${index}`"
                class="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800"
              >
                <button
                  type="button"
                  class="block w-full overflow-hidden bg-white transition hover:opacity-95 focus:outline-none focus:ring-2 focus:ring-primary-400 dark:bg-dark-950"
                  :aria-label="t('admin.usage.openImagePreview')"
                  :data-test="`usage-detail-image-open-${index}`"
                  @click="openPreview(preview)"
                >
                  <img
                    :src="preview.src"
                    :alt="`usage-image-preview-${index + 1}`"
                    :data-test="`usage-detail-image-preview-${index}`"
                    class="aspect-square w-full bg-white object-contain dark:bg-dark-950"
                  />
                </button>
                <figcaption v-if="preview.revisedPrompt" class="border-t border-gray-200 px-4 py-3 text-sm text-gray-600 dark:border-dark-700 dark:text-gray-300">
                  {{ preview.revisedPrompt }}
                </figcaption>
              </figure>
            </div>
          </section>

          <section class="overflow-hidden rounded-2xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
            <div class="border-b border-gray-200 px-4 py-3 text-xs font-semibold uppercase tracking-[0.18em] text-gray-500 dark:border-dark-700 dark:text-gray-400">
              {{ t('admin.usage.rawResponseBody') }}
            </div>
            <pre class="min-h-full whitespace-pre-wrap break-words p-4 text-xs text-gray-800 dark:text-gray-100">{{ activeContent }}</pre>
          </section>
        </div>

        <pre v-else-if="activeContent" class="min-h-full whitespace-pre-wrap break-words p-4 text-xs text-gray-800 dark:text-gray-100">{{ activeContent }}</pre>

        <div v-else class="h-full rounded-lg border border-dashed border-gray-200 p-6 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
          {{ t('admin.usage.emptyDetailContent') }}
        </div>
      </div>

      <div
        v-if="selectedPreview"
        class="fixed inset-0 z-[70] flex items-center justify-center bg-black/80 p-6 backdrop-blur-sm"
        data-test="usage-detail-image-preview-modal"
        role="dialog"
        @click.self="closePreview"
      >
        <div class="relative w-full max-w-5xl overflow-hidden rounded-3xl border border-white/10 bg-slate-950 shadow-2xl">
          <div class="flex items-center justify-between border-b border-white/10 px-5 py-4 text-white">
            <h4 class="text-sm font-semibold uppercase tracking-[0.18em]">{{ t('admin.usage.previewImageTitle') }}</h4>
            <button
              type="button"
              class="rounded-full border border-white/15 px-3 py-1.5 text-xs font-medium text-white transition hover:border-white/30 hover:bg-white/10"
              :aria-label="t('admin.usage.closeImagePreview')"
              data-test="usage-detail-image-preview-close"
              @click="closePreview"
            >
              {{ t('admin.usage.closeImagePreview') }}
            </button>
          </div>

          <div class="flex max-h-[80vh] items-center justify-center bg-black px-4 py-4">
            <img
              :src="selectedPreview.src"
              :alt="t('admin.usage.previewImageTitle')"
              class="max-h-[72vh] w-auto max-w-full object-contain"
              data-test="usage-detail-image-preview-modal-image"
            />
          </div>
        </div>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { AdminUsageDetail, AdminUsageLog } from '@/types'
import { sanitizeUrl } from '@/utils/url'

type DetailTabKey =
  | 'client-request-headers'
  | 'client-request-body'
  | 'upstream-request-headers'
  | 'upstream-request-body'
  | 'upstream-response-headers'
  | 'upstream-response-body'
  | 'response-headers'
  | 'response-body'

interface Props {
  show: boolean
  usageLog: AdminUsageLog | null
  detail: AdminUsageDetail | null
  loading: boolean
  error: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'retry'): void
}>()

const { t } = useI18n()
const activeTab = ref<DetailTabKey>('client-request-headers')
const copied = ref(false)
const selectedPreview = ref<UsageDetailImagePreview | null>(null)

interface UsageDetailImagePreview {
  src: string
  revisedPrompt?: string
}

const OUTPUT_FORMAT_MIME_MAP: Record<string, string> = {
  jpeg: 'image/jpeg',
  jpg: 'image/jpeg',
  png: 'image/png',
  webp: 'image/webp',
}

const tabs = computed(() => [
  { key: 'client-request-headers' as const, label: t('admin.usage.clientRequestHeaders') },
  { key: 'client-request-body' as const, label: t('admin.usage.clientRequestBody') },
  { key: 'upstream-request-headers' as const, label: t('admin.usage.upstreamRequestHeaders') },
  { key: 'upstream-request-body' as const, label: t('admin.usage.upstreamRequestBody') },
  { key: 'upstream-response-headers' as const, label: t('admin.usage.upstreamResponseHeaders') },
  { key: 'upstream-response-body' as const, label: t('admin.usage.upstreamResponseBody') },
  { key: 'response-headers' as const, label: t('admin.usage.responseHeaders') },
  { key: 'response-body' as const, label: t('admin.usage.responseBody') },
])

const formatJsonLike = (value: unknown) => {
  if (value == null) return ''
  if (typeof value === 'string') {
    try {
      return JSON.stringify(JSON.parse(value), null, 2)
    } catch {
      return value
    }
  }
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

const normalizeOutputFormat = (value: unknown): string | undefined => {
  if (typeof value !== 'string') {
    return undefined
  }

  const normalized = value.trim().toLowerCase()
  return normalized || undefined
}

const parseJsonRecord = (value: string | null | undefined): Record<string, unknown> | null => {
  if (!value || typeof value !== 'string') {
    return null
  }

  try {
    const parsed = JSON.parse(value) as unknown
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return null
    }
    return parsed as Record<string, unknown>
  } catch {
    return null
  }
}

const inferPreviewMimeType = (item: Record<string, unknown>, requestBody: string | null): string => {
  const requestPayload = parseJsonRecord(requestBody)
  const candidateValues = [
    item.mime_type,
    item.mimeType,
    item.content_type,
    item.contentType,
    item.output_format,
    item.outputFormat,
    item.format,
    requestPayload?.output_format,
  ]

  for (const candidate of candidateValues) {
    const normalized = normalizeOutputFormat(candidate)
    if (!normalized) {
      continue
    }

    if (normalized.startsWith('image/')) {
      return normalized
    }

    if (OUTPUT_FORMAT_MIME_MAP[normalized]) {
      return OUTPUT_FORMAT_MIME_MAP[normalized]
    }
  }

  return OUTPUT_FORMAT_MIME_MAP.png
}

const buildImagePreviews = (responseBody: string | null, requestBody: string | null): UsageDetailImagePreview[] => {
  const payload = parseJsonRecord(responseBody)
  const data = payload?.data
  if (!Array.isArray(data)) {
    return []
  }

  return data.flatMap((item): UsageDetailImagePreview[] => {
    if (!item || typeof item !== 'object' || Array.isArray(item)) {
      return []
    }

    const imageItem = item as Record<string, unknown>
    const revisedPrompt = typeof imageItem.revised_prompt === 'string' ? imageItem.revised_prompt.trim() || undefined : undefined

    if (typeof imageItem.b64_json === 'string' && imageItem.b64_json.trim()) {
      return [{
        src: `data:${inferPreviewMimeType(imageItem, requestBody)};base64,${imageItem.b64_json.trim()}`,
        revisedPrompt,
      }]
    }

    if (typeof imageItem.url === 'string') {
      const safeSrc = sanitizeUrl(imageItem.url)
      if (!safeSrc) {
        return []
      }
      return [{ src: safeSrc, revisedPrompt }]
    }

    return []
  })
}

const activeResponseBody = computed(() => {
  if (!props.detail) {
    return null
  }

  if (activeTab.value === 'upstream-response-body') {
    return props.detail.upstream_response_body
  }

  if (activeTab.value === 'response-body') {
    return props.detail.response_body
  }

  return null
})

const activeImagePreviews = computed(() => buildImagePreviews(activeResponseBody.value, props.detail?.request_body ?? null))

const openPreview = (preview: UsageDetailImagePreview) => {
  selectedPreview.value = preview
}

const closePreview = () => {
  selectedPreview.value = null
}

const activeContent = computed(() => {
  if (!props.detail) return ''
  if (activeTab.value === 'client-request-headers') return formatJsonLike(props.detail.request_headers)
  if (activeTab.value === 'client-request-body') return formatJsonLike(props.detail.request_body)
  if (activeTab.value === 'upstream-request-headers') return formatJsonLike(props.detail.upstream_request_headers)
  if (activeTab.value === 'upstream-request-body') return formatJsonLike(props.detail.upstream_request_body)
  if (activeTab.value === 'upstream-response-headers') return formatJsonLike(props.detail.upstream_response_headers)
  if (activeTab.value === 'upstream-response-body') return formatJsonLike(props.detail.upstream_response_body)
  if (activeTab.value === 'response-headers') return formatJsonLike(props.detail.response_headers)
  return formatJsonLike(props.detail.response_body)
})

const copyCurrentTab = async () => {
  if (!activeContent.value) return
  await navigator.clipboard.writeText(activeContent.value)
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 1500)
}

watch(
  () => props.show,
  (show) => {
    if (show) {
      activeTab.value = 'client-request-headers'
      copied.value = false
      selectedPreview.value = null
    }
  },
)
</script>
