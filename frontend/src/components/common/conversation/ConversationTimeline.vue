<template>
  <div v-if="!flow.messages?.length" class="conversation-empty-state">
    <div class="conversation-empty-icon" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6">
        <path stroke-linecap="round" stroke-linejoin="round" d="M7 8h10M7 12h6m-8 8 3.2-2.4c.35-.26.78-.4 1.22-.4H17a4 4 0 0 0 4-4V7a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4v6.2c0 1.2.53 2.32 1.45 3.08" />
      </svg>
    </div>
    <p>{{ t('conversation.empty') }}</p>
  </div>

  <ol v-else class="conversation-timeline" :aria-label="t('conversation.timelineLabel')">
    <li v-for="message in flow.messages" :key="message.id" class="conversation-timeline-item">
      <ConversationMessageRow :message="message" />
    </li>
  </ol>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ConversationMessageRow from './ConversationMessageRow.vue'
import type { ConversationFlow } from '@/utils/conversation/types'

defineProps<{
  flow: ConversationFlow
}>()

const { t } = useI18n()
</script>

<style scoped>
.conversation-empty-state {
  @apply flex flex-col items-center justify-center rounded-3xl border border-dashed border-gray-300 bg-white/70 px-6 py-12 text-center text-sm font-medium text-gray-500 shadow-sm dark:border-dark-700 dark:bg-dark-900/60 dark:text-dark-400;
}

.conversation-empty-icon {
  @apply mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary-50 text-primary-600 dark:bg-primary-950/30 dark:text-primary-300;
}

.conversation-empty-icon svg {
  @apply h-7 w-7;
}

.conversation-timeline {
  @apply space-y-5;
}

.conversation-timeline-item {
  @apply list-none;
}
</style>
