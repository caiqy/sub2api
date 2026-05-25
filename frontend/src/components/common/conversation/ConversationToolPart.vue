<template>
  <section data-test="conversation-part-tool" class="conversation-tool-part">
    <button
      data-test="conversation-tool-toggle"
      type="button"
      class="conversation-tool-header"
      :aria-expanded="!collapsed"
      @click="collapsed = !collapsed"
    >
      <span class="conversation-tool-status" :class="statusClass" aria-hidden="true"></span>
      <span class="sr-only">{{ statusText }}</span>
      <span class="conversation-tool-title">{{ displayName }}</span>
      <svg
        viewBox="0 0 20 20"
        fill="currentColor"
        class="conversation-tool-arrow"
        :class="{ 'rotate-180': !collapsed }"
        aria-hidden="true"
      >
        <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.168l3.71-3.938a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06Z" clip-rule="evenodd" />
      </svg>
    </button>

    <div v-if="!collapsed" class="conversation-tool-body">
      <template v-if="part.tool === 'bash'">
        <pre v-if="command" class="conversation-tool-code">$ {{ command }}</pre>
        <pre v-if="outputText" class="conversation-tool-code">{{ outputText }}</pre>
        <pre v-if="part.state.error" class="conversation-tool-code conversation-tool-code--error">{{ part.state.error }}</pre>
      </template>

      <template v-else>
        <div v-if="part.state.error" class="conversation-tool-section">
          <p class="conversation-tool-section-label">{{ t('conversation.error') }}</p>
          <pre class="conversation-tool-code conversation-tool-code--error">{{ part.state.error }}</pre>
        </div>
        <div v-else-if="outputText" class="conversation-tool-section">
          <p class="conversation-tool-section-label">{{ t('conversation.toolOutput') }}</p>
          <pre class="conversation-tool-code">{{ outputText }}</pre>
        </div>
        <div v-else-if="inputText" class="conversation-tool-section">
          <p class="conversation-tool-section-label">{{ t('conversation.toolInput') }}</p>
          <pre class="conversation-tool-code">{{ inputText }}</pre>
        </div>
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatRawValue } from '@/utils/conversation/format'
import { getToolDisplayName } from '@/utils/conversation/toolDisplay'
import type { ConversationToolPart } from '@/utils/conversation/types'

const props = defineProps<{
  part: ConversationToolPart
}>()

const { t, te } = useI18n()
const collapsed = ref(true)

watch(
  () => props.part.id,
  () => {
    collapsed.value = true
  },
)

const command = computed(() => {
  const input = props.part.state.input
  if (typeof input !== 'object' || input === null || Array.isArray(input)) return ''
  const value = (input as Record<string, unknown>).command
  return typeof value === 'string' ? value : ''
})

const inputText = computed(() => formatRawValue(props.part.state.input))
const outputText = computed(() => formatRawValue(props.part.state.output))
const toolLabel = computed(() => {
  const key = `conversation.toolLabels.${props.part.tool}`
  return te(key) ? t(key) : undefined
})

const displayName = computed(() => getToolDisplayName({
  tool: props.part.tool,
  input: props.part.state.input,
  title: props.part.state.title,
  output: props.part.state.output,
  label: toolLabel.value,
}))

const statusClass = computed(() => ({
  'conversation-tool-status--pending': props.part.state.status === 'pending',
  'conversation-tool-status--running': props.part.state.status === 'running',
  'conversation-tool-status--completed': props.part.state.status === 'completed',
  'conversation-tool-status--error': props.part.state.status === 'error',
}))

const statusText = computed(() => {
  if (props.part.state.status === 'pending') return 'Pending'
  if (props.part.state.status === 'running') return 'Running'
  if (props.part.state.status === 'completed') return 'Completed'
  return 'Error'
})
</script>

<style scoped>
.conversation-tool-part {
  @apply overflow-hidden rounded-xl border border-cyan-200/80 bg-cyan-50/50 shadow-sm dark:border-cyan-900/50 dark:bg-cyan-950/15;
}

.conversation-tool-header {
  @apply flex w-full items-center gap-2 px-3 py-2 text-left text-xs font-semibold text-cyan-900 transition-colors hover:bg-cyan-100/70 focus:outline-none focus:ring-2 focus:ring-cyan-500/25 dark:text-cyan-100 dark:hover:bg-cyan-950/35;
}

.conversation-tool-status {
  @apply h-2 w-2 shrink-0 rounded-full bg-gray-400;
}

.conversation-tool-status--pending {
  @apply bg-gray-400;
}

.conversation-tool-status--running {
  @apply bg-amber-400 shadow-[0_0_0_3px_rgba(251,191,36,0.18)];
}

.conversation-tool-status--completed {
  @apply bg-emerald-500 shadow-[0_0_0_3px_rgba(16,185,129,0.16)];
}

.conversation-tool-status--error {
  @apply bg-red-500 shadow-[0_0_0_3px_rgba(239,68,68,0.16)];
}

.conversation-tool-title {
  @apply min-w-0 flex-1 truncate;
}

.conversation-tool-arrow {
  @apply h-4 w-4 shrink-0 transition-transform duration-200;
}

.conversation-tool-body {
  @apply space-y-2 border-t border-cyan-200/70 bg-white/75 p-3 dark:border-cyan-900/50 dark:bg-dark-950/45;
}

.conversation-tool-section {
  @apply space-y-1.5;
}

.conversation-tool-section-label {
  @apply text-[11px] font-semibold uppercase tracking-[0.16em] text-gray-500 dark:text-dark-400;
}

.conversation-tool-code {
  @apply max-h-72 overflow-auto rounded-lg border border-gray-200 bg-gray-950 p-3 font-mono text-xs leading-5 text-emerald-100 shadow-inner dark:border-dark-700 dark:bg-dark-950;
}

.conversation-tool-code--error {
  @apply text-red-100;
}
</style>
