<template>
  <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800" data-testid="image-history-detail">
    <div class="flex items-start justify-between gap-4">
      <div class="space-y-1">
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('images.history.detailTitle') }}</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400">{{ detailSummary }}</p>
      </div>
      <button
        v-if="detail"
        class="btn btn-secondary"
        data-testid="image-history-replay"
        type="button"
        @click="$emit('replay', detail)"
      >
        {{ t('images.history.replay') }}
      </button>
    </div>

    <p v-if="loading" class="mt-4 text-sm text-gray-500 dark:text-gray-400">{{ t('images.history.detailLoading') }}</p>

    <div v-else-if="error" class="mt-4 rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-900/10 dark:text-red-300">
      {{ error }}
    </div>

    <p v-else-if="!detail" class="mt-4 rounded-2xl border border-dashed border-gray-300 bg-gray-50 px-4 py-6 text-sm text-gray-500 dark:border-dark-600 dark:bg-dark-900/60 dark:text-gray-400">
      {{ t('images.history.detailEmpty') }}
    </p>

    <div v-else class="mt-4 space-y-5">
      <div class="grid gap-3 rounded-2xl border border-gray-200 bg-gray-50/80 p-4 text-sm dark:border-dark-700 dark:bg-dark-900/60">
        <div class="flex items-center justify-between gap-3">
          <span class="text-gray-500 dark:text-gray-400">{{ t('images.history.status') }}</span>
          <span :class="detail.status === 'success' ? successPillClass : errorPillClass">{{ t(`images.history.statuses.${detail.status}`) }}</span>
        </div>
        <div class="flex items-center justify-between gap-3">
          <span class="text-gray-500 dark:text-gray-400">{{ t('images.history.apiKey') }}</span>
          <span class="text-right font-medium text-gray-900 dark:text-white">{{ detail.api_key_name || detail.api_key_masked || '-' }}</span>
        </div>
        <div class="flex items-center justify-between gap-3">
          <span class="text-gray-500 dark:text-gray-400">{{ t('images.history.createdAt') }}</span>
          <span class="text-right font-medium text-gray-900 dark:text-white">{{ formatCreatedAt(detail.created_at) }}</span>
        </div>
      </div>

      <div class="space-y-2">
        <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('images.history.prompt') }}</h4>
        <p class="rounded-2xl border border-gray-200 bg-gray-50/80 px-4 py-3 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-900/60 dark:text-gray-200">
          {{ detail.prompt || '-' }}
        </p>
      </div>

      <div class="space-y-2">
        <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('images.history.parameters') }}</h4>
        <dl class="grid gap-3 rounded-2xl border border-gray-200 bg-gray-50/80 p-4 text-sm sm:grid-cols-2 dark:border-dark-700 dark:bg-dark-900/60">
          <div v-for="parameter in parameters" :key="parameter.label" class="space-y-1">
            <dt class="text-gray-500 dark:text-gray-400">{{ parameter.label }}</dt>
            <dd class="font-medium text-gray-900 dark:text-white">{{ parameter.value }}</dd>
          </div>
        </dl>
      </div>

      <div v-if="detail.error_message" class="space-y-2">
        <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('images.history.errorMessage') }}</h4>
        <p class="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-900/10 dark:text-red-300">
          {{ detail.error_message }}
        </p>
      </div>

      <div class="space-y-2">
        <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('images.history.images') }}</h4>
        <p v-if="displayImages.length === 0" class="rounded-2xl border border-dashed border-gray-300 bg-gray-50 px-4 py-6 text-sm text-gray-500 dark:border-dark-600 dark:bg-dark-900/60 dark:text-gray-400">
          {{ t('images.results.empty') }}
        </p>
        <div v-else class="grid gap-4 sm:grid-cols-2">
          <figure v-for="(image, index) in displayImages" :key="`${index}-${image.src}`" class="overflow-hidden rounded-2xl border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-900/60">
            <img :src="image.src" :alt="`history-preview-${index + 1}`" class="aspect-square w-full bg-white object-contain dark:bg-dark-950" :data-testid="`image-history-detail-image-${index}`" />
            <figcaption v-if="image.revisedPrompt" class="border-t border-gray-200 px-4 py-3 text-sm text-gray-600 dark:border-dark-700 dark:text-gray-300">
              <span class="font-medium text-gray-900 dark:text-white">{{ t('images.results.revisedPrompt') }}:</span>
              {{ image.revisedPrompt }}
            </figcaption>
          </figure>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

import type { ImageHistoryDetail as ImageHistoryDetailType } from '@/types'
import { sanitizeUrl } from '@/utils/url'

const props = defineProps<{
  detail: ImageHistoryDetailType | null
  error: string
  loading: boolean
}>()

defineEmits<{
  (e: 'replay', detail: ImageHistoryDetailType): void
}>()

const { t } = useI18n()

const successPillClass = 'rounded-full bg-emerald-100 px-2.5 py-1 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
const errorPillClass = 'rounded-full bg-red-100 px-2.5 py-1 text-red-700 dark:bg-red-900/30 dark:text-red-300'

const detailSummary = computed(() => {
  if (!props.detail) {
    return t('images.history.detailEmpty')
  }

  return `${t(`images.history.modes.${props.detail.mode}`)} · ${props.detail.model}`
})

const parameters = computed(() => {
  if (!props.detail) {
    return []
  }

  return [
    { label: t('images.forms.generate.model'), value: props.detail.model || '-' },
    { label: t('images.forms.generate.size'), value: props.detail.size || '-' },
    { label: t('images.forms.generate.quality'), value: props.detail.quality || '-' },
    { label: t('images.forms.generate.background'), value: props.detail.background || '-' },
    { label: t('images.forms.generate.outputFormat'), value: props.detail.output_format || '-' },
    { label: t('images.forms.generate.moderation'), value: props.detail.moderation || '-' },
    { label: t('images.forms.generate.n'), value: String(props.detail.n ?? '-') },
    { label: t('images.history.hadSourceImage'), value: props.detail.had_source_image ? t('images.history.booleanYes') : t('images.history.booleanNo') },
    { label: t('images.history.hadMask'), value: props.detail.had_mask ? t('images.history.booleanYes') : t('images.history.booleanNo') },
  ]
})

const displayImages = computed(() => (props.detail?.images ?? []).flatMap((image) => {
  const safeSrc = sanitizeUrl(image.data_url, { allowDataUrl: true })
  if (!safeSrc) {
    return []
  }

  return [{
    src: safeSrc,
    revisedPrompt: image.revised_prompt,
  }]
}))

function formatCreatedAt(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}
</script>
