<template>
  <div class="card" data-testid="profile-traffic-billing-card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.trafficBilling.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.trafficBilling.description') }}
      </p>
    </div>

    <div class="space-y-5 px-6 py-6">
      <!-- Toggle Row -->
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div class="space-y-1">
          <label class="text-sm font-medium text-gray-900 dark:text-white">
            {{ toggleLabel }}
          </label>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('profile.trafficBilling.thresholdInfo', { threshold: displayThreshold }) }}
          </p>
        </div>
        <div class="shrink-0">
          <Toggle
            data-testid="allow-large-requests-toggle"
            :model-value="allowLargeRequests"
            :aria-label="toggleLabel"
            :disabled="loading"
            @update:model-value="handleToggle"
          />
        </div>
      </div>

      <!-- Explanatory Details Box -->
      <div class="rounded-xl border border-gray-100 bg-gray-50/80 p-4 text-xs text-gray-600 dark:border-dark-700 dark:bg-dark-900/30 dark:text-gray-300 space-y-2">
        <div class="flex items-start gap-2">
          <span class="font-semibold text-primary-600 dark:text-primary-400 shrink-0">•</span>
          <p>{{ t('profile.trafficBilling.formula') }}</p>
        </div>
        <div class="flex items-start gap-2">
          <span class="font-semibold text-primary-600 dark:text-primary-400 shrink-0">•</span>
          <p>{{ t('profile.trafficBilling.futurePriceNotice') }}</p>
        </div>
        <div v-if="!systemTrafficBillingEnabled" class="flex items-start gap-2 text-amber-600 dark:text-amber-400">
          <span class="font-semibold shrink-0">!</span>
          <p>{{ t('profile.trafficBilling.disabledNotice') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import { userAPI } from '@/api'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { User } from '@/types'

const props = defineProps<{
  user?: User | null
  trafficBillingEnabled: boolean
  trafficBillingThresholdMB: number
  trafficBillingPricePerMB: number
}>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const loading = ref(false)

const currentUser = computed(() => props.user ?? authStore.user)
const allowLargeRequests = computed(() => Boolean(currentUser.value?.allow_large_requests))

const systemTrafficBillingEnabled = computed(() => props.trafficBillingEnabled)
const currentPrice = computed(() => props.trafficBillingPricePerMB)
const displayThreshold = computed(() => props.trafficBillingThresholdMB)

const toggleLabel = computed(() => {
  const price = currentPrice.value
  return t('profile.trafficBilling.label', { price })
})

const handleToggle = async (value: boolean) => {
  loading.value = true
  try {
    const updatedUser = await userAPI.updateProfile({
      allow_large_requests: value
    })
    authStore.user = updatedUser
    appStore.showSuccess(t('profile.trafficBilling.updateSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('profile.trafficBilling.updateFailed')))
  } finally {
    loading.value = false
  }
}
</script>
