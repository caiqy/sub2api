<template>
  <div data-test="conversation-part-text" class="conversation-text-part">
    <ConversationMarkdown :content="part.text" />
  </div>
</template>

<script setup lang="ts">
import ConversationMarkdown from './ConversationMarkdown.vue'
import type { ConversationTextPart } from '@/utils/conversation/types'

defineProps<{
  part: ConversationTextPart
}>()
</script>

<style scoped>
.conversation-text-part {
  @apply min-w-0 text-sm leading-6 text-gray-800 dark:text-dark-100;
}
</style>
