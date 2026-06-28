<template>
  <section data-test="conversation-part-injection">
    <button
      v-if="collapsed"
      type="button"
      class="inline-flex cursor-pointer items-center gap-1.5 rounded border border-dashed border-gray-300 bg-gray-50 px-2 py-1 font-mono text-[11px] text-gray-500 dark:border-dark-700 dark:bg-dark-900/30 dark:text-dark-400"
      @click="collapsed = false"
    >
      <span>[{{ part.tag }}]</span>
      <svg viewBox="0 0 20 20" fill="currentColor" class="h-3 w-3" aria-hidden="true">
        <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.168l3.71-3.938a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06Z" clip-rule="evenodd" />
      </svg>
    </button>
    <div v-else>
      <button
        type="button"
        class="mb-2 inline-flex cursor-pointer items-center gap-1.5 rounded border border-dashed border-gray-300 bg-gray-50 px-2 py-1 font-mono text-[11px] text-gray-500 dark:border-dark-700 dark:bg-dark-900/30 dark:text-dark-400"
        @click="collapsed = true"
      >
        <span>[{{ part.tag }}]</span>
        <svg viewBox="0 0 20 20" fill="currentColor" class="h-3 w-3 rotate-180" aria-hidden="true">
          <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.168l3.71-3.938a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06Z" clip-rule="evenodd" />
        </svg>
      </button>
      <pre class="max-h-72 overflow-auto whitespace-pre-wrap break-words rounded-md border border-gray-200 bg-gray-50 p-3 font-mono text-[12px] leading-5 text-gray-700 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-200">{{ part.text }}</pre>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { ConversationInjectionPart } from '@/utils/conversation/types'

const props = defineProps<{
  part: ConversationInjectionPart
}>()

const collapsed = ref<boolean>(props.part.defaultCollapsed)

watch(
  () => [props.part.id, props.part.defaultCollapsed] as const,
  () => {
    collapsed.value = props.part.defaultCollapsed
  },
)
</script>
