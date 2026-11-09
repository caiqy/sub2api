<template>
  <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-sm">
    <p role="status" :aria-busy="loading" class="text-gray-600 dark:text-gray-300">
      <template v-if="loading">{{ t('common.loading') }}</template>
      <template v-else-if="summary">
        {{ t('admin.users.remainingQuota.label') }}：
        <strong class="font-semibold tabular-nums text-gray-900 dark:text-white">{{ money(summary.total) }}</strong>
        <span class="tabular-nums">（{{ t('admin.users.remainingQuota.balance') }} {{ money(summary.balance) }}，{{ t('admin.users.remainingQuota.subscription') }} {{ money(summary.subscription) }}）</span>
        <span v-if="summary.unlimited_subscriptions" class="ml-2 text-amber-700 dark:text-amber-400">
          {{ t('admin.users.remainingQuota.unlimited', { count: summary.unlimited_subscriptions }) }}
        </span>
      </template>
      <template v-else>{{ t('admin.users.remainingQuota.failed') }}</template>
    </p>
    <button type="button" class="text-primary-600 hover:underline disabled:cursor-not-allowed disabled:opacity-50 dark:text-primary-400" :disabled="loading || saving || !summary" @click="openSettings">
      {{ t('admin.users.remainingQuota.exclude', { count: excludedIds.length }) }}
    </button>
    <button v-if="!loading && !summary" type="button" class="text-primary-600 hover:underline dark:text-primary-400" @click="loadSummary">
      {{ t('common.refresh') }}
    </button>
    <BaseDialog :show="showSettings" :title="t('admin.users.remainingQuota.settings')" @close="!saving && (showSettings = false)">
      <p class="mb-4 text-sm text-gray-600 dark:text-gray-300">{{ t('admin.users.remainingQuota.scope') }}</p>
      <fieldset :disabled="saving">
        <legend class="mb-2 text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.users.remainingQuota.settings') }}</legend>
        <OpenAIFastPolicyUserSelector v-if="showSettings" v-model="draftIds" />
      </fieldset>
      <p class="mt-4 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.users.remainingQuota.subscriptionHint') }}</p>
      <p v-if="storageError" role="alert" class="mt-3 text-sm text-red-600 dark:text-red-400">{{ t('admin.users.remainingQuota.storageFailed') }}</p>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="saving" @click="showSettings = false">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="saving" @click="saveSettings">{{ t(saving ? 'common.saving' : 'common.save') }}</button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getRemainingQuota, saveQuotaExclusions, type RemainingQuotaSummary } from '@/api/admin/users'
import BaseDialog from '@/components/common/BaseDialog.vue'
import OpenAIFastPolicyUserSelector from '@/views/admin/settings/OpenAIFastPolicyUserSelector.vue'

const props = defineProps<{ refreshKey: number }>()
const { t } = useI18n()
const excludedIds = ref<number[]>([])
const saving = ref(false)
const draftIds = ref<number[]>([])
const showSettings = ref(false)
const storageError = ref(false)
const summary = ref<RemainingQuotaSummary | null>(null)
const loading = ref(false)
let controller: AbortController | undefined
const money = (value: number) => `${value.toFixed(2)}$`

function openSettings() {
  draftIds.value = [...excludedIds.value]
  storageError.value = false
  showSettings.value = true
}
async function saveSettings() {
  if (saving.value) return
  saving.value = true
  storageError.value = false
  controller?.abort()
  try {
    const saved = await saveQuotaExclusions(draftIds.value)
    excludedIds.value = saved.excluded_user_ids
    showSettings.value = false
  } catch {
    storageError.value = true
  } finally {
    saving.value = false
  }
  if (!storageError.value) await loadSummary()
}
async function loadSummary() {
  if (saving.value) return
  controller?.abort()
  const current = new AbortController()
  controller = current
  loading.value = true
  summary.value = null
  try {
    const result = await getRemainingQuota(current.signal)
    if (!current.signal.aborted) {
      summary.value = result
      excludedIds.value = result.excluded_user_ids
    }
  } catch {
    // Keep failure separate from a legitimate zero balance.
  } finally {
    if (controller === current) loading.value = false
  }
}
watch(() => props.refreshKey, loadSummary, { immediate: true })
onUnmounted(() => controller?.abort())
</script>
