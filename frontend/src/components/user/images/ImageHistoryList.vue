<template>
  <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800" data-testid="image-history-list">
    <div class="space-y-1">
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('images.history.listTitle') }}</h3>
      <p v-if="loading" class="text-sm text-gray-500 dark:text-gray-400">{{ t('images.history.loading') }}</p>
    </div>

    <div v-if="error" class="mt-4 rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-900/10 dark:text-red-300">
      <p>{{ error }}</p>
      <button class="mt-3 btn btn-secondary" type="button" @click="$emit('retry')">{{ t('images.history.retry') }}</button>
    </div>

    <p v-else-if="!loading && items.length === 0" class="mt-4 rounded-2xl border border-dashed border-gray-300 bg-gray-50 px-4 py-6 text-sm text-gray-500 dark:border-dark-600 dark:bg-dark-900/60 dark:text-gray-400">
      {{ t('images.history.empty') }}
    </p>

    <div v-else class="mt-4 space-y-3">
      <button
        v-for="item in items"
        :key="item.id"
        :class="[
          'w-full rounded-2xl border px-4 py-3 text-left transition',
          item.id === selectedId
            ? 'border-primary-400 bg-primary-50/70 dark:border-primary-500 dark:bg-primary-900/20'
            : 'border-gray-200 bg-gray-50/70 hover:border-primary-200 hover:bg-white dark:border-dark-700 dark:bg-dark-900/50 dark:hover:border-primary-800'
        ]"
        :data-testid="`image-history-list-item-${item.id}`"
        type="button"
        @click="$emit('select', item.id)"
      >
        <div class="flex items-start justify-between gap-4">
          <div class="space-y-1">
            <p class="text-sm font-medium text-gray-900 dark:text-white">{{ item.model }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ formatCreatedAt(item.created_at) }}</p>
          </div>
          <div class="flex flex-wrap justify-end gap-2 text-xs font-medium">
            <span class="rounded-full bg-gray-100 px-2.5 py-1 text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ t(`images.history.modes.${item.mode}`) }}</span>
            <span :class="item.status === 'success' ? successPillClass : errorPillClass">{{ t(`images.history.statuses.${item.status}`) }}</span>
          </div>
        </div>

        <div class="mt-3 flex flex-wrap gap-x-4 gap-y-2 text-xs text-gray-500 dark:text-gray-400">
          <span>{{ t('images.history.count') }}: {{ item.image_count }}</span>
          <span v-if="item.image_size">{{ item.image_size }}</span>
          <span v-if="item.api_key_name">{{ t('images.history.apiKey') }}: {{ item.api_key_name }}</span>
        </div>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import type { ImageHistoryListItem } from '@/types'

defineProps<{
  error: string
  items: ImageHistoryListItem[]
  loading: boolean
  selectedId: number | null
}>()

defineEmits<{
  (e: 'retry'): void
  (e: 'select', id: number): void
}>()

const { t } = useI18n()

const successPillClass = 'rounded-full bg-emerald-100 px-2.5 py-1 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
const errorPillClass = 'rounded-full bg-red-100 px-2.5 py-1 text-red-700 dark:bg-red-900/30 dark:text-red-300'

function formatCreatedAt(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}
</script>
