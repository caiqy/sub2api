<template>
  <section data-test="conversation-part-reasoning" class="conversation-reasoning-part">
    <button
      type="button"
      class="conversation-reasoning-toggle"
      :aria-expanded="!collapsed"
      @click="collapsed = !collapsed"
    >
      <span class="conversation-reasoning-label">
        · {{ t('conversation.reasoningMeta.collapsedLabel') }}
        <template v-if="segments > 1"> · {{ t('conversation.reasoningMeta.segments', { n: segments }) }}</template>
      </span>
    </button>

    <div v-if="!collapsed" class="conversation-reasoning-body">
      <template v-if="segments > 1">
        <template v-for="(segment, idx) in textSegments" :key="idx">
          <hr v-if="idx > 0" class="my-2 border-gray-200 dark:border-dark-700" />
          <ConversationMarkdown :content="segment" />
        </template>
      </template>
      <ConversationMarkdown v-else :content="part.text" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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

const segments = computed(() => {
  const meta = props.part.metadata
  return typeof meta?.segments === 'number' ? meta.segments : 1
})

const textSegments = computed(() => {
  if (segments.value <= 1) return [props.part.text]
  return props.part.text.split('\n\n')
})
</script>

<style scoped>
.conversation-reasoning-part {
  /* No border or background when collapsed; keep reasoning visually minimal. */
}

.conversation-reasoning-toggle {
  @apply inline-flex items-center text-[11px] text-gray-400 transition-colors hover:text-gray-600 focus:outline-none dark:text-dark-500 dark:hover:text-dark-300;
}

.conversation-reasoning-label {
  @apply select-none;
}

.conversation-reasoning-body {
  @apply mt-2 rounded-md border-l border-gray-200 bg-gray-50 p-3 pl-3 text-gray-700 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-200;
}
</style>
