<template>
  <button v-if="account.platform === 'openai'" type="button" class="block max-w-full rounded text-left text-xs focus-visible:outline focus-visible:outline-primary-500"
    :class="latest?.result === 'degraded' ? 'text-red-600 dark:text-red-400' : latest?.result === 'normal' ? 'text-green-600 dark:text-green-400' : 'text-gray-500 dark:text-gray-400'"
    :title="tooltip" :aria-label="`${label}; ${t('admin.modeltrace.history')}; ${tooltip}`" @click="$emit('history')">{{ label }}</button>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountListItem } from '@/types'
import type { ModelTraceLatest } from '@/api/admin/modeltrace'
import { formatDateTime } from '@/utils/format'
const props = defineProps<{ account: Pick<AccountListItem, 'platform' | 'extra'> }>()
defineEmits<{ history: [] }>()
const { t } = useI18n()
const latest = computed(() => {
  const value = props.account.extra?.modeltrace_latest as ModelTraceLatest | undefined
  return value && (value.result === 'normal' || value.result === 'degraded') ? value : null
})
const label = computed(() => t('admin.modeltrace.iq', { result: t(`admin.modeltrace.result.${latest.value?.result || 'untested'}`) }))
const tooltip = computed(() => latest.value ? t('admin.modeltrace.latestTooltip', {
  model: latest.value.model,
  target: latest.value.target_model || latest.value.model,
  time: latest.value.finished_at ? formatDateTime(latest.value.finished_at) : '-'
}) : t('admin.modeltrace.result.untested'))
</script>
