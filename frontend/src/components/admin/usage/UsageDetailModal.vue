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
          <div class="break-all text-gray-900 dark:text-white">{{ usageLog?.user?.email || '-' }}</div>
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

      <div v-if="loading" class="rounded-lg border border-dashed border-gray-200 p-6 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>

      <div v-else-if="error" class="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-900/40 dark:bg-red-900/10">
        <div class="text-sm text-red-700 dark:text-red-300">{{ error }}</div>
        <button data-test="usage-detail-retry" type="button" class="btn btn-secondary mt-3" @click="emit('retry')">
          {{ t('common.retry') }}
        </button>
      </div>

      <pre v-else-if="activeContent" class="max-h-[60vh] overflow-auto rounded-lg bg-gray-50 p-4 text-xs text-gray-800 dark:bg-dark-900 dark:text-gray-100">{{ activeContent }}</pre>

      <div v-else class="rounded-lg border border-dashed border-gray-200 p-6 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
        {{ t('admin.usage.emptyDetailContent') }}
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { AdminUsageDetail, AdminUsageLog } from '@/types'

type DetailTabKey =
  | 'client-request-headers'
  | 'client-request-body'
  | 'upstream-request-headers'
  | 'upstream-request-body'
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

const tabs = computed(() => [
  { key: 'client-request-headers' as const, label: t('admin.usage.clientRequestHeaders') },
  { key: 'client-request-body' as const, label: t('admin.usage.clientRequestBody') },
  { key: 'upstream-request-headers' as const, label: t('admin.usage.upstreamRequestHeaders') },
  { key: 'upstream-request-body' as const, label: t('admin.usage.upstreamRequestBody') },
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

const activeContent = computed(() => {
  if (!props.detail) return ''
  if (activeTab.value === 'client-request-headers') return formatJsonLike(props.detail.request_headers)
  if (activeTab.value === 'client-request-body') return formatJsonLike(props.detail.request_body)
  if (activeTab.value === 'upstream-request-headers') return formatJsonLike(props.detail.upstream_request_headers)
  if (activeTab.value === 'upstream-request-body') return formatJsonLike(props.detail.upstream_request_body)
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
    }
  },
)
</script>
