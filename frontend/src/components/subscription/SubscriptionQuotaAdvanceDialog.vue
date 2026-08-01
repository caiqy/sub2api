<template>
  <BaseDialog
    :show="show"
    :title="t('userSubscriptions.quotaAdvance.title')"
    width="normal"
    :close-on-escape="!submitting"
    :show-close-button="!submitting"
    @close="close"
  >
    <form class="space-y-5" @submit.prevent="submit">
      <p class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('userSubscriptions.quotaAdvance.description', { name: subscription?.group?.name || '' }) }}
      </p>

      <fieldset class="divide-y divide-gray-200 border-y border-gray-200 dark:divide-dark-700 dark:border-dark-700">
        <legend class="sr-only">{{ t('userSubscriptions.quotaAdvance.selectWindows') }}</legend>
        <label
          v-for="window in windows"
          :key="window.key"
          class="flex cursor-pointer items-center justify-between gap-4 py-3"
        >
          <span class="flex items-center gap-3">
            <input
              v-model="selected"
              type="checkbox"
              :value="window.key"
              :disabled="submitting"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-800"
            />
            <span class="text-sm font-medium text-gray-800 dark:text-gray-200">
              {{ windowLabel(window.key) }}
            </span>
          </span>
          <span class="text-xs text-gray-500 dark:text-dark-400">
            {{ t('userSubscriptions.quotaAdvance.normalResetIn', { time: formatDuration(window.remainingMs) }) }}
          </span>
        </label>
      </fieldset>

      <dl v-if="preview.deductedMs > 0" class="space-y-2 border-b border-gray-200 pb-4 text-sm dark:border-dark-700">
        <div class="flex items-center justify-between gap-4">
          <dt class="text-gray-500 dark:text-dark-400">{{ t('userSubscriptions.quotaAdvance.deducted') }}</dt>
          <dd data-test="deducted-duration" class="font-semibold text-gray-900 dark:text-white">
            {{ formatDuration(preview.deductedMs) }}
          </dd>
        </div>
        <div class="flex items-center justify-between gap-4">
          <dt class="text-gray-500 dark:text-dark-400">{{ t('userSubscriptions.quotaAdvance.newExpiry') }}</dt>
          <dd class="font-medium text-gray-900 dark:text-white">
            {{ preview.newExpiresAt ? formatDateTimeToMinute(preview.newExpiresAt) : '-' }}
          </dd>
        </div>
      </dl>

      <div
        v-if="selected.length > 0 && preview.unselectedExhausted.length > 0"
        data-test="partial-warning"
        class="flex gap-2 text-sm text-amber-700 dark:text-amber-300"
      >
        <Icon name="exclamationTriangle" size="sm" class="mt-0.5 shrink-0" />
        <span>{{ t('userSubscriptions.quotaAdvance.partialWarning', { windows: unselectedWindowNames }) }}</span>
      </div>

      <div class="flex gap-2 text-sm text-red-700 dark:text-red-300">
        <Icon name="exclamationCircle" size="sm" class="mt-0.5 shrink-0" />
        <span>{{ t('userSubscriptions.quotaAdvance.irreversible') }}</span>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button data-test="cancel-advance" type="button" class="btn btn-secondary" :disabled="submitting" @click="close">
          {{ t('common.cancel') }}
        </button>
        <button
          data-test="confirm-advance"
          type="button"
          class="btn btn-primary"
          :disabled="!canSubmit"
          @click="submit"
        >
          {{ submitting ? t('userSubscriptions.quotaAdvance.submitting') : t('userSubscriptions.quotaAdvance.confirm') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserSubscription } from '@/types'
import { advanceQuotaCycle } from '@/api/subscriptions'
import { useAppStore } from '@/stores/app'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTimeToMinute } from '@/utils/format'
import {
  getExhaustedQuotaWindows,
  getQuotaAdvancePreview,
  type SubscriptionQuotaWindow,
} from '@/utils/subscriptionQuota'

const props = defineProps<{
  show: boolean
  subscription: UserSubscription | null
}>()

const emit = defineEmits<{
  close: []
  success: [subscription: UserSubscription]
}>()

const { t } = useI18n()
const appStore = useAppStore()
const selected = ref<SubscriptionQuotaWindow[]>([])
const submitting = ref(false)

const windows = computed(() => props.subscription ? getExhaustedQuotaWindows(props.subscription) : [])
const preview = computed(() => props.subscription
  ? getQuotaAdvancePreview(props.subscription, selected.value)
  : { deductedMs: 0, newExpiresAt: null, unselectedExhausted: [] })
const canSubmit = computed(() => selected.value.length > 0 && preview.value.deductedMs > 0 && !submitting.value)
const unselectedWindowNames = computed(() => preview.value.unselectedExhausted.map(windowLabel).join(', '))

watch(() => [props.show, props.subscription?.id], () => {
  selected.value = []
  submitting.value = false
})

function windowLabel(window: SubscriptionQuotaWindow): string {
  return t(`userSubscriptions.${window}`)
}

function formatDuration(ms: number): string {
  const totalMinutes = Math.max(1, Math.floor(ms / 60000))
  const days = Math.floor(totalMinutes / 1440)
  const hours = Math.floor((totalMinutes % 1440) / 60)
  const minutes = totalMinutes % 60
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

function close() {
  if (!submitting.value) emit('close')
}

async function submit() {
  if (!props.subscription || !canSubmit.value) return
  submitting.value = true
  try {
    const result = await advanceQuotaCycle(props.subscription.id, {
      daily: selected.value.includes('daily'),
      weekly: selected.value.includes('weekly'),
      monthly: selected.value.includes('monthly'),
    })
    emit('success', result.subscription)
    emit('close')
  } catch (error: any) {
    appStore.showError(
      extractApiErrorMessage(error, t('userSubscriptions.quotaAdvance.failed')),
    )
  } finally {
    submitting.value = false
  }
}
</script>
