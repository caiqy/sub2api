<template>
  <section data-test="conversation-part-reasoning" class="conversation-reasoning-part">
    <button
      type="button"
      class="conversation-reasoning-toggle"
      :aria-expanded="!collapsed"
      @click="collapsed = !collapsed"
    >
      <svg
        viewBox="0 0 20 20"
        fill="currentColor"
        class="conversation-reasoning-arrow"
        :class="{ 'rotate-90': !collapsed }"
        aria-hidden="true"
      >
        <path fill-rule="evenodd" d="M7.22 4.22a.75.75 0 0 1 1.06 0l5.25 5.25a.75.75 0 0 1 0 1.06l-5.25 5.25a.75.75 0 0 1-1.06-1.06L11.94 10 7.22 5.28a.75.75 0 0 1 0-1.06Z" clip-rule="evenodd" />
      </svg>
      <span>{{ t('conversation.reasoning') }}</span>
    </button>

    <div v-if="!collapsed" class="conversation-reasoning-body">
      <ConversationMarkdown :content="part.text" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ConversationMarkdown from './ConversationMarkdown.vue'
import type { ConversationReasoningPart } from '@/utils/conversation/types'

const props = defineProps<{
  part: ConversationReasoningPart
}>()

const { t } = useI18n()
const collapsed = ref<boolean>(props.part.defaultCollapsed)

watch(
  () => [props.part.id, props.part.defaultCollapsed] as const,
  () => {
    collapsed.value = props.part.defaultCollapsed
  },
)
</script>

<style scoped>
.conversation-reasoning-part {
  @apply rounded-xl border border-dashed border-violet-200/80 bg-violet-50/50 px-3 py-2 dark:border-violet-900/50 dark:bg-violet-950/15;
}

.conversation-reasoning-toggle {
  @apply inline-flex items-center gap-1.5 text-xs font-medium text-violet-700 transition-colors hover:text-violet-900 focus:outline-none focus:ring-2 focus:ring-violet-500/25 dark:text-violet-300 dark:hover:text-violet-100;
}

.conversation-reasoning-arrow {
  @apply h-3.5 w-3.5 transition-transform duration-200;
}

.conversation-reasoning-body {
  @apply mt-2 border-l border-violet-200 pl-3 text-gray-700 dark:border-violet-900/60 dark:text-dark-200;
}
</style>
