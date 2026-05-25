<template>
  <div v-if="safeMediaParts.length" class="conversation-media-grid">
    <template v-for="(part, index) in safeMediaParts" :key="index">
      <figure v-if="part.type === 'image'" class="conversation-image-frame">
        <img
          data-test="conversation-image"
          :src="part.safeSrc"
          :alt="part.alt || t('conversation.imageAlt')"
          class="conversation-image"
          loading="lazy"
        />
        <figcaption v-if="part.alt" class="conversation-media-caption">
          {{ part.alt }}
        </figcaption>
      </figure>

      <article v-else class="conversation-file-card">
        <div class="conversation-file-icon" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
            <path stroke-linecap="round" stroke-linejoin="round" d="M14 2H7a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7z" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M14 2v5h5" />
          </svg>
        </div>
        <div class="min-w-0 flex-1">
          <a
            v-if="part.safeUrl"
            :href="part.safeUrl"
            target="_blank"
            rel="noreferrer"
            class="conversation-file-name"
          >
            {{ part.filename || part.safeUrl }}
          </a>
          <p v-else class="conversation-file-name">
            {{ part.filename || 'Attachment' }}
          </p>
          <p v-if="part.mimeType" class="conversation-file-meta">
            {{ part.mimeType }}
          </p>
          <pre v-if="part.text" class="conversation-file-text">{{ part.text }}</pre>
        </div>
      </article>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ConversationContentPart } from '@/utils/conversation/types'

const props = defineProps<{
  parts: ConversationContentPart[]
}>()

const { t } = useI18n()

type SafeImagePart = Extract<ConversationContentPart, { type: 'image' }> & { safeSrc: string }
type SafeFilePart = Extract<ConversationContentPart, { type: 'file' }> & { safeUrl?: string }
type SafeMediaPart = SafeImagePart | SafeFilePart

const safeHttpUrl = (value: string | undefined): string | undefined => {
  if (!value) return undefined

  const trimmed = value.trim()
  try {
    const url = new URL(trimmed)
    return url.protocol === 'http:' || url.protocol === 'https:' ? trimmed : undefined
  } catch {
    return undefined
  }
}

const safeImageUrl = (value: string): string | undefined => {
  const trimmed = value.trim()
  if (/^data:image\/(?:png|jpe?g|webp|gif);base64,[a-z0-9+/=]+$/i.test(trimmed)) return trimmed
  return safeHttpUrl(trimmed)
}

const safeMediaParts = computed<SafeMediaPart[]>(() => {
  return props.parts.reduce<SafeMediaPart[]>((acc, part) => {
    if (part.type === 'image') {
      const safeSrc = safeImageUrl(part.src)
      if (safeSrc) acc.push({ ...part, safeSrc })
      return acc
    }

    if (part.type === 'file') {
      acc.push({ ...part, safeUrl: safeHttpUrl(part.url) })
    }

    return acc
  }, [])
})
</script>

<style scoped>
.conversation-media-grid {
  @apply mt-4 grid gap-3 sm:grid-cols-2;
}

.conversation-image-frame {
  @apply overflow-hidden rounded-2xl border border-gray-200 bg-gray-50 shadow-sm transition-all duration-200 hover:-translate-y-0.5 hover:shadow-card-hover dark:border-dark-700 dark:bg-dark-900/70;
}

.conversation-image {
  @apply max-h-72 w-full object-contain;
}

.conversation-media-caption {
  @apply border-t border-gray-200 px-3 py-2 text-xs text-gray-500 dark:border-dark-700 dark:text-dark-400;
}

.conversation-file-card {
  @apply flex gap-3 rounded-2xl border border-gray-200 bg-white/80 p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900/70;
}

.conversation-file-icon {
  @apply flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300;
}

.conversation-file-icon svg {
  @apply h-5 w-5;
}

.conversation-file-name {
  @apply block truncate text-sm font-semibold text-gray-900 transition-colors hover:text-primary-700 dark:text-white dark:hover:text-primary-300;
}

.conversation-file-meta {
  @apply mt-0.5 text-xs text-gray-500 dark:text-dark-400;
}

.conversation-file-text {
  @apply mt-3 max-h-40 overflow-auto rounded-xl border border-gray-200 bg-gray-50 p-3 font-mono text-xs leading-5 text-gray-700 dark:border-dark-700 dark:bg-dark-950 dark:text-dark-200;
}
</style>
