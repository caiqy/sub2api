<template>
  <section data-test="conversation-part-raw" class="conversation-raw-part" :class="{ 'conversation-raw-part--error': part.type === 'error' }">
    <button
      data-test="conversation-raw-toggle"
      type="button"
      class="conversation-raw-toggle"
      :aria-expanded="!collapsed"
      @click="collapsed = !collapsed"
    >
      <span>{{ title }}</span>
      <svg
        viewBox="0 0 20 20"
        fill="currentColor"
        class="conversation-raw-arrow"
        :class="{ 'rotate-180': !collapsed }"
        aria-hidden="true"
      >
        <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.168l3.71-3.938a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06Z" clip-rule="evenodd" />
      </svg>
    </button>

    <pre v-if="!collapsed" class="conversation-raw-body">{{ body }}</pre>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ConversationErrorPart, ConversationRawPart } from '@/utils/conversation/types'

const props = defineProps<{
  part: ConversationRawPart | ConversationErrorPart
}>()

const { t } = useI18n()
const collapsed = ref<boolean>(props.part.defaultCollapsed)

watch(
  () => [props.part.id, props.part.defaultCollapsed] as const,
  () => {
    collapsed.value = props.part.defaultCollapsed
  },
)

const title = computed(() => {
  if (props.part.type === 'error') return t('conversation.error')
  return props.part.title || t('conversation.raw')
})

const body = computed(() => {
  if (props.part.type === 'error') return [props.part.error, props.part.raw].filter(Boolean).join('\n\n')
  return props.part.raw
})
</script>

<style scoped>
.conversation-raw-part {
  @apply overflow-hidden rounded-xl border border-gray-200 bg-gray-50/80 dark:border-dark-700 dark:bg-dark-900/60;
}

.conversation-raw-part--error {
  @apply border-red-200 bg-red-50/70 dark:border-red-900/60 dark:bg-red-950/20;
}

.conversation-raw-toggle {
  @apply flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-xs font-semibold text-gray-700 transition-colors hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-primary-500/25 dark:text-dark-200 dark:hover:bg-dark-800;
}

.conversation-raw-arrow {
  @apply h-4 w-4 shrink-0 transition-transform duration-200;
}

.conversation-raw-body {
  @apply max-h-80 overflow-auto border-t border-gray-200 bg-gray-950 p-3 font-mono text-xs leading-5 text-emerald-100 dark:border-dark-700 dark:bg-dark-950;
}
</style>
