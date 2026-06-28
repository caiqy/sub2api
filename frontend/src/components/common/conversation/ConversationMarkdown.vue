<template>
  <div class="conversation-markdown" v-html="html"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import DOMPurify from 'dompurify'
import { marked } from 'marked'

const props = defineProps<{
  content: string
}>()

const html = computed(() => {
  const rendered = marked.parse(props.content || '', {
    breaks: true,
  })

  return DOMPurify.sanitize(typeof rendered === 'string' ? rendered : '', {
    FORBID_TAGS: ['img'],
  })
})
</script>

<style scoped>
.conversation-markdown {
  @apply max-w-none text-sm leading-6 text-gray-800 dark:text-dark-100;
}

.conversation-markdown :deep(p) {
  @apply my-2 first:mt-0 last:mb-0;
}

.conversation-markdown :deep(strong) {
  @apply font-semibold text-gray-950 dark:text-white;
}

.conversation-markdown :deep(a) {
  @apply rounded-sm font-medium text-primary-700 underline decoration-primary-300 decoration-2 underline-offset-4 transition-colors hover:text-primary-600 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:text-primary-300 dark:decoration-primary-700 dark:hover:text-primary-200;
}

.conversation-markdown :deep(code) {
  @apply rounded-md border border-primary-100 bg-primary-50 px-1.5 py-0.5 font-mono text-[0.85em] text-primary-800 dark:border-primary-900/50 dark:bg-primary-950/40 dark:text-primary-200;
}

.conversation-markdown :deep(pre) {
  @apply my-3 overflow-x-auto rounded-xl border border-gray-200 bg-gray-950 p-4 shadow-inner dark:border-dark-700 dark:bg-dark-950;
}

.conversation-markdown :deep(pre code) {
  @apply border-0 bg-transparent p-0 text-xs leading-5 text-emerald-100 dark:text-emerald-100;
}

.conversation-markdown :deep(blockquote) {
  @apply my-3 border-l-4 border-primary-300 bg-primary-50/70 px-4 py-2 text-gray-700 dark:border-primary-700 dark:bg-primary-950/20 dark:text-dark-200;
}

.conversation-markdown :deep(ul),
.conversation-markdown :deep(ol) {
  @apply my-2 space-y-1 pl-5;
}

.conversation-markdown :deep(ul) {
  @apply list-disc;
}

.conversation-markdown :deep(ol) {
  @apply list-decimal;
}
</style>
