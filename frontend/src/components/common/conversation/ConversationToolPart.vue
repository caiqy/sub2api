<template>
  <section data-test="conversation-part-tool" class="conversation-tool-part" :class="statusBarClass">
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
      <span v-if="metaRight" class="conversation-tool-meta" :class="{ 'conversation-tool-meta--error': part.state.error }">{{ metaRight }}</span>
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
      <!-- Todo list rendering for todowrite/todoread -->
      <template v-if="isTodoTool && todoItems.length > 0">
        <ul class="conversation-todo-list">
          <li v-for="(item, idx) in todoItems" :key="idx" class="conversation-todo-item">
            <span class="conversation-todo-status" :class="`conversation-todo-status--${item.status}`">
              <svg v-if="item.status === 'completed'" viewBox="0 0 16 16" fill="currentColor" class="h-3 w-3">
                <path fill-rule="evenodd" d="M12.416 3.376a.75.75 0 0 1 .208 1.04l-5 7.5a.75.75 0 0 1-1.154.114l-3-3a.75.75 0 0 1 1.06-1.06l2.353 2.353 4.493-6.74a.75.75 0 0 1 1.04-.207Z" clip-rule="evenodd" />
              </svg>
              <svg v-else-if="item.status === 'in_progress'" viewBox="0 0 16 16" fill="currentColor" class="h-3 w-3 animate-spin">
                <path d="M8 1a7 7 0 0 0 0 14 .75.75 0 0 1 0 1.5A8.5 8.5 0 1 1 16.5 8 .75.75 0 0 1 15 8a7 7 0 0 0-7-7Z" />
              </svg>
              <span v-else class="h-2 w-2 rounded-full bg-current"></span>
            </span>
            <span class="conversation-todo-content" :class="{ 'line-through opacity-50': item.status === 'completed' }">{{ item.content }}</span>
            <span v-if="item.priority" class="conversation-todo-priority" :class="`conversation-todo-priority--${item.priority}`">{{ item.priority }}</span>
          </li>
        </ul>
      </template>

      <!-- Bash tool -->
      <template v-else-if="part.tool === 'bash'">
        <pre v-if="command" class="conversation-tool-code">$ {{ command }}</pre>
        <pre v-if="outputText" class="conversation-tool-code">{{ outputText }}</pre>
        <pre v-if="part.state.error" class="conversation-tool-code conversation-tool-code--error">{{ part.state.error }}</pre>
      </template>

      <!-- Generic tools -->
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
import { formatHumanBytes, formatRawValue } from '@/utils/conversation/format'
import { getToolDisplayName } from '@/utils/conversation/toolDisplay'
import type { ConversationToolPart } from '@/utils/conversation/types'

interface TodoItem {
  content: string
  status: string
  priority?: string
}

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

const isTodoTool = computed(() => props.part.tool === 'todowrite' || props.part.tool === 'todoread')

const todoItems = computed<TodoItem[]>(() => {
  if (!isTodoTool.value) return []
  try {
    // Try output first (todoread returns list), then input (todowrite sends list)
    const raw = props.part.state.output ?? props.part.state.input
    if (!raw) return []
    const data = typeof raw === 'string' ? JSON.parse(raw) : raw
    // todowrite input has { todos: [...] } or is directly an array
    const items = Array.isArray(data) ? data : (data?.todos ?? data?.items ?? [])
    if (!Array.isArray(items)) return []
    return items.filter((item): item is TodoItem =>
      typeof item === 'object' && item !== null && typeof item.content === 'string'
    )
  } catch {
    return []
  }
})

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

const metaRight = computed(() => {
  if (props.part.state.error) return t('conversation.toolMeta.error')
  const size = props.part.state.outputSize
  if (!size) return ''
  const lines = t('conversation.toolMeta.lines', { n: size.lines })
  const bytes = formatHumanBytes(size.bytes)
  return t('conversation.toolMeta.sizeWithLines', { lines, size: bytes })
})

const statusBarClass = computed(() => `conversation-tool-part--${props.part.state.status}`)

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
  @apply overflow-hidden rounded-lg border border-gray-200 border-l-2 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900/40;
}

.conversation-tool-part--pending {
  @apply border-l-gray-300;
}

.conversation-tool-part--running {
  @apply border-l-amber-400;
}

.conversation-tool-part--completed {
  @apply border-l-emerald-500;
}

.conversation-tool-part--error {
  @apply border-l-red-500;
}

.conversation-tool-header {
  @apply flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs font-medium text-gray-700 transition-colors hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-400/25 dark:text-dark-100 dark:hover:bg-dark-800/50;
}

.conversation-tool-status {
  @apply h-2 w-2 shrink-0 rounded-full;
}

.conversation-tool-status--pending {
  @apply bg-gray-300;
}

.conversation-tool-status--running {
  @apply bg-amber-400;
}

.conversation-tool-status--completed {
  @apply bg-emerald-500;
}

.conversation-tool-status--error {
  @apply bg-red-500;
}

.conversation-tool-title {
  @apply min-w-0 flex-1 truncate;
}

.conversation-tool-meta {
  @apply shrink-0 text-[11px] text-gray-500 dark:text-dark-400;
}

.conversation-tool-meta--error {
  @apply text-red-500 dark:text-red-400;
}

.conversation-tool-arrow {
  @apply h-4 w-4 shrink-0 text-gray-400 transition-transform duration-200 dark:text-dark-500;
}

.conversation-tool-body {
  @apply space-y-2 border-t border-gray-200 bg-gray-50/50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900/60;
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

/* Todo list styles */
.conversation-todo-list {
  @apply space-y-1;
}

.conversation-todo-item {
  @apply flex items-start gap-2 rounded px-2 py-1.5 text-xs;
}

.conversation-todo-status {
  @apply mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center;
}

.conversation-todo-status--completed {
  @apply text-emerald-500;
}

.conversation-todo-status--in_progress {
  @apply text-amber-500;
}

.conversation-todo-status--pending {
  @apply text-gray-400 dark:text-dark-500;
}

.conversation-todo-status--cancelled {
  @apply text-gray-300 dark:text-dark-600;
}

.conversation-todo-content {
  @apply flex-1 text-gray-700 dark:text-dark-200;
}

.conversation-todo-priority {
  @apply shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium uppercase;
}

.conversation-todo-priority--high {
  @apply bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300;
}

.conversation-todo-priority--medium {
  @apply bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300;
}

.conversation-todo-priority--low {
  @apply bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300;
}
</style>
