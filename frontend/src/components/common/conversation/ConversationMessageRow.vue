<template>
  <article
    :data-test="`conversation-message-row-${message.role}`"
    class="conversation-message-row"
    :class="`conversation-message-row--${message.role}`"
  >
    <div class="conversation-message-shell">
      <header class="conversation-message-header">
        <span class="conversation-message-role">{{ roleLabel }}</span>
      </header>

      <div class="conversation-message-parts">
        <template v-for="part in sortedParts" :key="part.id">
          <ConversationReasoningPart v-if="part.type === 'reasoning'" :part="part" />
          <ConversationTextPart v-else-if="part.type === 'text'" :part="part" />
          <ConversationToolPart v-else-if="part.type === 'tool'" :part="part" />
          <ConversationMediaPreview v-else-if="part.type === 'image' || part.type === 'file'" :parts="[part]" />
          <ConversationRawPart v-else-if="part.type === 'raw' || part.type === 'error'" :part="part" />
        </template>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ConversationMediaPreview from './ConversationMediaPreview.vue'
import ConversationRawPart from './ConversationRawPart.vue'
import ConversationReasoningPart from './ConversationReasoningPart.vue'
import ConversationTextPart from './ConversationTextPart.vue'
import ConversationToolPart from './ConversationToolPart.vue'
import { partPriority } from '@/utils/conversation/toolDisplay'
import type { ConversationMessage } from '@/utils/conversation/types'

const props = defineProps<{
  message: ConversationMessage
}>()

const { t } = useI18n()

const roleLabel = computed(() => {
  if (props.message.role === 'user') return t('conversation.role.you')
  return t(`conversation.role.${props.message.role}`)
})

const sortedParts = computed(() => {
  return [...props.message.parts].sort((a, b) => partPriority(a.type) - partPriority(b.type))
})
</script>

<style scoped>
.conversation-message-row {
  @apply flex w-full;
}

.conversation-message-row--user {
  @apply justify-end;
}

.conversation-message-row--assistant,
.conversation-message-row--system,
.conversation-message-row--developer {
  @apply justify-start;
}

.conversation-message-shell {
  @apply max-w-[min(100%,52rem)] rounded-2xl border border-transparent px-4 py-3;
}

.conversation-message-row--user .conversation-message-shell {
  @apply max-w-[min(100%,44rem)] border-primary-100 bg-primary-50/70 shadow-sm dark:border-primary-900/50 dark:bg-primary-950/20;
}

.conversation-message-row--assistant .conversation-message-shell {
  @apply px-0 py-1;
}

.conversation-message-row--system .conversation-message-shell,
.conversation-message-row--developer .conversation-message-shell {
  @apply border-amber-200/70 bg-amber-50/60 shadow-sm dark:border-amber-900/50 dark:bg-amber-950/15;
}

.conversation-message-header {
  @apply mb-2 flex items-center gap-2;
}

.conversation-message-row--assistant .conversation-message-header {
  @apply sr-only;
}

.conversation-message-role {
  @apply text-[11px] font-semibold uppercase tracking-[0.18em] text-gray-500 dark:text-dark-400;
}

.conversation-message-row--user .conversation-message-role {
  @apply text-primary-700 dark:text-primary-300;
}

.conversation-message-parts {
  @apply min-w-0 space-y-3;
}
</style>
