<template>
  <div class="space-y-3 text-sm">
    <div class="flex flex-wrap items-center gap-3" aria-live="polite">
      <span class="font-medium" :class="task.result === 'degraded' || task.status === 'failed' ? 'text-red-600 dark:text-red-400' : 'text-gray-900 dark:text-white'">
        {{ t(`admin.modeltrace.status.${task.status}`) }}
        <template v-if="task.status === 'completed' && task.result"> · {{ t(`admin.modeltrace.result.${task.result}`) }}</template>
      </span>
      <span>{{ t(`admin.modeltrace.source.${task.source}`) }}</span>
      <span>{{ t('admin.modeltrace.progress', { completed: task.completed_rounds, rounds: task.rounds }) }}</span>
    </div>
    <dl class="grid grid-cols-1 gap-x-6 gap-y-2 text-gray-600 dark:text-gray-300 sm:grid-cols-2">
      <div class="min-w-0"><dt class="text-xs text-gray-500">{{ t('admin.modeltrace.model') }}</dt><dd class="break-all">{{ task.model }}</dd></div>
      <div class="min-w-0"><dt class="text-xs text-gray-500">{{ t('admin.modeltrace.targetModel') }}</dt><dd class="break-all">{{ task.target_model || '-' }}</dd></div>
      <div class="min-w-0"><dt class="text-xs text-gray-500">{{ t('admin.modeltrace.winner') }}</dt><dd class="break-all">{{ task.winner || '-' }}</dd></div>
      <div><dt class="text-xs text-gray-500">{{ t('admin.modeltrace.duration') }}</dt><dd>{{ task.duration_ms == null ? '-' : `${(task.duration_ms / 1000).toFixed(1)} s` }}</dd></div>
      <div><dt class="text-xs text-gray-500">{{ t('admin.modeltrace.createdAt') }}</dt><dd>{{ formatDateTime(task.created_at) }}</dd></div>
      <div><dt class="text-xs text-gray-500">{{ t('admin.modeltrace.startedAt') }}</dt><dd>{{ task.started_at ? formatDateTime(task.started_at) : '-' }}</dd></div>
      <div><dt class="text-xs text-gray-500">{{ t('admin.modeltrace.finishedAt') }}</dt><dd>{{ task.finished_at ? formatDateTime(task.finished_at) : '-' }}</dd></div>
      <div class="min-w-0"><dt class="text-xs text-gray-500">{{ t('admin.modeltrace.version') }}</dt><dd class="break-all">{{ task.version || '-' }}</dd></div>
    </dl>
    <div v-if="Object.keys(task.probabilities || {}).length" class="flex flex-wrap gap-x-4 gap-y-1">
      <span class="text-gray-500">{{ t('admin.modeltrace.probabilities') }}:</span>
      <span v-for="[name, probability] in probabilities" :key="name" class="break-all text-gray-700 dark:text-gray-300">{{ name }}: {{ (probability * 100).toFixed(1) }}%</span>
    </div>
    <p v-if="task.error" class="whitespace-pre-wrap break-words text-red-600 dark:text-red-400">{{ t('admin.modeltrace.failure') }}: {{ task.error }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelTraceTask } from '@/api/admin/modeltrace'
import { formatDateTime } from '@/utils/format'
const props = defineProps<{ task: ModelTraceTask }>()
const { t } = useI18n()
const probabilities = computed(() => Object.entries(props.task.probabilities || {}).sort((a, b) => b[1] - a[1]))
</script>
