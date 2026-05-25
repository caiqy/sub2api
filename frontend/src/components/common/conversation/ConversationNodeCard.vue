<template>
  <article
    data-test="conversation-node-card"
    class="conversation-card group"
    :class="[`conversation-card--${tone}`]"
  >
    <header class="conversation-card-header">
      <div class="conversation-card-identity">
        <span class="conversation-role-chip">
          <span class="conversation-role-dot" aria-hidden="true"></span>
          {{ roleLabel }}
        </span>
        <div class="min-w-0">
          <h3 class="conversation-title">
            {{ displayTitle }}
          </h3>
          <p v-if="node.summary" class="conversation-summary">
            {{ node.summary }}
          </p>
        </div>
      </div>

      <button
        data-test="conversation-node-toggle"
        type="button"
        class="conversation-toggle"
        :aria-expanded="!collapsed"
        @click="collapsed = !collapsed"
      >
        <span>{{ collapsed ? t('conversation.expand') : t('conversation.collapse') }}</span>
        <svg
          viewBox="0 0 20 20"
          fill="currentColor"
          class="h-4 w-4 transition-transform duration-200"
          :class="{ 'rotate-180': !collapsed }"
          aria-hidden="true"
        >
          <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.168l3.71-3.938a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06Z" clip-rule="evenodd" />
        </svg>
      </button>
    </header>

    <div v-if="!collapsed" class="conversation-card-body">
      <template v-if="'parts' in node">
        <div class="space-y-3">
          <ConversationMarkdown
            v-for="(textPart, index) in textParts"
            :key="index"
            :content="textPart.text"
          />
        </div>
        <ConversationMediaPreview :parts="node.parts" />
      </template>

      <pre v-else-if="rawContent" class="conversation-raw-panel">{{ rawContent }}</pre>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ConversationMarkdown from './ConversationMarkdown.vue'
import ConversationMediaPreview from './ConversationMediaPreview.vue'
import type { ConversationNode } from '@/utils/conversation/types'
import { formatRawValue } from '@/utils/conversation/format'

const props = defineProps<{
  node: ConversationNode
}>()

const { t } = useI18n()
const collapsed = ref(props.node.defaultCollapsed)

watch(
  () => [props.node.id, props.node.defaultCollapsed] as const,
  () => {
    collapsed.value = props.node.defaultCollapsed
  }
)

const textParts = computed(() => {
  if (!('parts' in props.node)) return []
  return props.node.parts.filter((part) => part.type === 'text')
})

const rawContent = computed(() => {
  const node = props.node

  if (node.type === 'tool_call') return formatRawValue(node.input ?? node.metadata ?? '')
  if (node.type === 'tool_result') return formatRawValue(node.output ?? node.metadata ?? '')
  if (node.type === 'raw') return node.raw
  if (node.type === 'error') return [node.error, node.raw].filter(Boolean).join('\n\n')
  return ''
})

const tone = computed(() => {
  if (props.node.type === 'tool_call' || props.node.type === 'tool_result') return 'tool'
  if (props.node.type === 'system' || props.node.type === 'developer') return 'system'
  if (props.node.type === 'error') return 'error'
  return props.node.type
})

const roleTitle = (role: 'user' | 'assistant' | 'system' | 'developer') => t(`conversation.role.${role}`)

const rawTitle = () => {
  if (props.node.metadata?.rawSource === 'request') return t('conversation.rawRequest')
  if (props.node.metadata?.rawSource === 'response') return t('conversation.rawResponse')
  return t('conversation.raw')
}

const displayTitle = computed(() => {
  const node = props.node

  if (node.type === 'user' || node.type === 'assistant' || node.type === 'system' || node.type === 'developer') return roleTitle(node.role)
  if (node.type === 'tool_call') return `${t('conversation.toolCall')}${node.toolName ? ` · ${node.toolName}` : ''}`
  if (node.type === 'tool_result') return `${t('conversation.toolResult')}${node.toolName ? ` · ${node.toolName}` : ''}`
  if (node.type === 'raw') return rawTitle()
  if (node.type === 'error') return t('conversation.error')
  return node.title
})

const roleLabel = computed(() => {
  const node = props.node

  if (node.type === 'user' || node.type === 'assistant' || node.type === 'system' || node.type === 'developer') return roleTitle(node.role)
  if (node.type === 'tool_call') return t('conversation.toolCall')
  if (node.type === 'tool_result') return t('conversation.toolResult')
  if (node.type === 'raw') return rawTitle()
  if (node.type === 'error') return t('conversation.error')
  return displayTitle.value
})
</script>

<style scoped>
.conversation-card {
  @apply relative overflow-hidden rounded-2xl border border-gray-200/80 bg-white p-4 shadow-card transition-all duration-200 hover:-translate-y-0.5 hover:border-primary-200 hover:shadow-card-hover focus-within:ring-2 focus-within:ring-primary-500/25 dark:border-dark-700/80 dark:bg-dark-900/80 dark:hover:border-primary-800/70;
}

.conversation-card::before {
  content: '';
  @apply absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-primary-400 via-cyan-400 to-transparent opacity-80;
}

.conversation-card-header {
  @apply flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between;
}

.conversation-card-identity {
  @apply flex min-w-0 flex-1 items-start gap-3;
}

.conversation-role-chip {
  @apply inline-flex shrink-0 items-center gap-1.5 rounded-full border border-gray-200 bg-gray-50 px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300;
}

.conversation-role-dot {
  @apply h-2 w-2 rounded-full bg-primary-500 shadow-glow;
}

.conversation-title {
  @apply truncate text-sm font-semibold text-gray-950 dark:text-white;
}

.conversation-summary {
  @apply mt-1 line-clamp-2 text-sm leading-5 text-gray-500 dark:text-dark-400;
}

.conversation-toggle {
  @apply inline-flex shrink-0 items-center justify-center gap-1.5 rounded-full border border-gray-200 bg-white px-3 py-1.5 text-xs font-medium text-gray-600 shadow-sm transition-all duration-200 hover:border-primary-200 hover:bg-primary-50 hover:text-primary-700 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300 dark:hover:border-primary-800 dark:hover:bg-primary-950/30 dark:hover:text-primary-300;
}

.conversation-card-body {
  @apply mt-4 border-t border-gray-100 pt-4 dark:border-dark-800;
}

.conversation-raw-panel {
  @apply max-h-96 overflow-auto rounded-2xl border border-dark-800 bg-dark-950 p-4 font-mono text-xs leading-5 text-emerald-100 shadow-inner;
}

.conversation-card--tool {
  @apply border-cyan-200/70 bg-gradient-to-br from-cyan-50/70 to-white dark:border-cyan-900/50 dark:from-cyan-950/20 dark:to-dark-900;
}

.conversation-card--tool::before {
  @apply from-cyan-400 via-primary-400;
}

.conversation-card--tool .conversation-role-chip {
  @apply border-cyan-200 bg-cyan-50 text-cyan-700 dark:border-cyan-900/60 dark:bg-cyan-950/30 dark:text-cyan-300;
}

.conversation-card--system {
  @apply border-amber-200/80 bg-gradient-to-br from-amber-50/70 to-white dark:border-amber-900/50 dark:from-amber-950/20 dark:to-dark-900;
}

.conversation-card--system::before {
  @apply from-amber-400 via-primary-400;
}

.conversation-card--system .conversation-role-chip {
  @apply border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-300;
}

.conversation-card--assistant .conversation-role-dot,
.conversation-card--user .conversation-role-dot {
  @apply bg-primary-500;
}

.conversation-card--error {
  @apply border-red-200 bg-red-50/70 dark:border-red-900/60 dark:bg-red-950/20;
}

.conversation-card--error::before {
  @apply from-red-500 via-amber-400;
}
</style>
