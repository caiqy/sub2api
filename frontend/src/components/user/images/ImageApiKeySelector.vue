<template>
  <div class="rounded-2xl border border-gray-200 bg-white/80 p-4 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-800/80">
    <label class="mb-2 block text-xs font-semibold uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">
      {{ label }}
    </label>
    <select
      :value="modelValue"
      :disabled="disabled"
      class="min-w-[240px] rounded-xl border border-gray-200 bg-white px-3 py-2.5 text-sm text-gray-900 outline-none transition focus:border-primary-400 focus:ring-2 focus:ring-primary-200 dark:border-dark-600 dark:bg-dark-900 dark:text-white dark:focus:border-primary-500 dark:focus:ring-primary-900"
      data-testid="image-api-key-selector"
      @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
    >
      <option :value="''">{{ placeholder }}</option>
      <option v-for="key in apiKeys" :key="key.id" :value="String(key.id)">
        {{ key.name }}
      </option>
    </select>
    <p class="mt-2 text-xs text-gray-500 dark:text-gray-400" data-testid="images-key-status">
      {{ statusMessage }}
    </p>
    <p v-if="loadState === 'success' && apiKeys.length > 0" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
      {{ pageHint }}
    </p>
    <button
      v-if="loadState === 'error'"
      class="mt-3 inline-flex items-center rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium text-gray-700 transition hover:bg-gray-50 dark:border-dark-600 dark:text-gray-200 dark:hover:bg-dark-700"
      type="button"
      @click="emit('retry')"
    >
      {{ retryLabel }}
    </button>
  </div>
</template>

<script setup lang="ts">
import type { ApiKey } from '@/types'

interface Props {
  modelValue: string
  apiKeys: ApiKey[]
  label: string
  placeholder: string
  statusMessage: string
  pageHint: string
  retryLabel: string
  disabled?: boolean
  loadState: 'loading' | 'success' | 'error'
}

withDefaults(defineProps<Props>(), {
  disabled: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'retry'): void
}>()
</script>
