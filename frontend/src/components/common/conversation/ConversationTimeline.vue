<template>
  <div v-if="!flow.systemPrompt && !flow.messages?.length" class="conversation-empty-state">
    <div class="conversation-empty-icon" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6">
        <path stroke-linecap="round" stroke-linejoin="round" d="M7 8h10M7 12h6m-8 8 3.2-2.4c.35-.26.78-.4 1.22-.4H17a4 4 0 0 0 4-4V7a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4v6.2c0 1.2.53 2.32 1.45 3.08" />
      </svg>
    </div>
    <p>{{ t('conversation.empty') }}</p>
  </div>

  <div v-else class="conversation-timeline-container">
    <div class="conversation-timeline-shell">
      <ConversationSystemPromptBar v-if="flow.systemPrompt" :prompt="flow.systemPrompt" />
      <ol class="conversation-timeline" :aria-label="t('conversation.timelineLabel')">
        <li v-for="message in flow.messages ?? []" :key="message.id" class="conversation-timeline-item">
          <ConversationMessageRow :message="message" />
        </li>
      </ol>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ConversationMessageRow from './ConversationMessageRow.vue'
import ConversationSystemPromptBar from './ConversationSystemPromptBar.vue'
import type { ConversationFlow, ConversationSystemPrompt } from '@/utils/conversation/types'

type ConversationTimelineFlow = ConversationFlow & {
  systemPrompt?: ConversationSystemPrompt
}

defineProps<{
  flow: ConversationTimelineFlow
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

.conversation-timeline-container {
  @apply mx-auto h-full max-w-[45rem] overflow-y-scroll rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800/50;
  scrollbar-gutter: stable;
}

/* Scrollbar styles - unscoped to handle .dark ancestor */
</style>

<style>
.conversation-timeline-container {
  scrollbar-width: thin;
  scrollbar-color: rgba(156, 163, 175, 0.6) rgba(0, 0, 0, 0.05);
}

.dark .conversation-timeline-container {
  scrollbar-color: rgba(148, 163, 184, 0.4) rgba(255, 255, 255, 0.05);
}

.conversation-timeline-container::-webkit-scrollbar {
  width: 8px;
}

.conversation-timeline-container::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.05);
  border-radius: 4px;
}

.conversation-timeline-container::-webkit-scrollbar-thumb {
  background-color: rgba(156, 163, 175, 0.6);
  border-radius: 4px;
  border: 1px solid transparent;
}

.conversation-timeline-container::-webkit-scrollbar-thumb:hover {
  background-color: rgba(156, 163, 175, 0.85);
}

.dark .conversation-timeline-container::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.05);
}

.dark .conversation-timeline-container::-webkit-scrollbar-thumb {
  background-color: rgba(148, 163, 184, 0.4);
}

.dark .conversation-timeline-container::-webkit-scrollbar-thumb:hover {
  background-color: rgba(148, 163, 184, 0.6);
}

.conversation-timeline {
  @apply space-y-6;
}

.conversation-timeline-shell {
  @apply space-y-4 px-5 py-5;
}

.conversation-timeline-item {
  @apply list-none;
}
</style>
