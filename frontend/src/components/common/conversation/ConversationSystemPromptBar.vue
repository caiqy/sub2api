<template>
  <section data-test="conversation-system-prompt-bar" class="border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900/40 rounded-md">
    <button
      type="button"
      class="flex w-full items-center gap-2 text-left"
      :aria-expanded="!collapsed"
      @click="collapsed = !collapsed"
    >
      <span class="text-[11px] uppercase tracking-[0.16em] text-gray-500 dark:text-dark-400">{{ t('conversation.systemPrompt.title') }}</span>
      <span v-if="prompt.sources.length > 1" class="text-[11px] text-gray-400 dark:text-dark-500">
        {{ t('conversation.systemPrompt.segments', { n: prompt.sources.length }) }}
      </span>
      <svg
        viewBox="0 0 20 20"
        fill="currentColor"
        class="ml-auto h-4 w-4 text-gray-400 transition-transform duration-200"
        :class="{ 'rotate-180': !collapsed }"
        aria-hidden="true"
      >
        <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.168l3.71-3.938a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06Z" clip-rule="evenodd" />
      </svg>
    </button>

    <div v-if="!collapsed" class="mt-2 rounded-md bg-gray-100 p-3 dark:bg-dark-900/60">
      <ConversationMarkdown :content="prompt.text" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ConversationMarkdown from './ConversationMarkdown.vue'
import type { ConversationSystemPrompt } from '@/utils/conversation/types'

defineProps<{
  prompt: ConversationSystemPrompt
}>()

const { t } = useI18n()
const collapsed = ref(true)
</script>
