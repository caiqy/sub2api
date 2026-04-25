<template>
  <AppLayout>
    <div class="mx-auto max-w-[1600px] space-y-5 px-3 sm:px-4 xl:px-6 2xl:px-8" data-testid="images-view">
      <section class="card overflow-hidden border border-primary-100 bg-gradient-to-br from-white via-white to-primary-50/70 p-5 dark:border-primary-900/40 dark:from-dark-900 dark:via-dark-900 dark:to-primary-950/20">
        <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div class="space-y-2">
            <p class="text-xs font-semibold uppercase tracking-[0.22em] text-primary-600 dark:text-primary-400">
              {{ t('images.badge') }}
            </p>
            <div>
              <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
                {{ t('images.title') }}
              </h1>
              <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-300">
                {{ t('images.description') }}
              </p>
            </div>
          </div>

          <ImageApiKeySelector
            v-model="selectedApiKeyId"
            :api-keys="apiKeys"
            :disabled="apiKeyLoadState !== 'success' || apiKeys.length === 0"
            :label="t('images.keySelector.label')"
            :load-state="apiKeyLoadState"
            :page-hint="t('images.keySelector.pageHint')"
            :placeholder="keySelectorPlaceholder"
            :retry-label="t('images.keySelector.retry')"
            :status-message="keyStatusMessage"
            @retry="loadApiKeys"
          />
        </div>
      </section>

      <section class="card p-2">
        <div class="flex flex-wrap gap-2" role="tablist" :aria-label="t('images.tabs.ariaLabel')">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            :aria-selected="activeTab === tab.key"
            :class="[
              'rounded-xl px-4 py-2.5 text-sm font-medium transition-all',
              activeTab === tab.key
                ? 'bg-primary-600 text-white shadow-sm dark:bg-primary-500'
                : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white'
            ]"
            :data-testid="`images-tab-${tab.key}`"
            role="tab"
            type="button"
            @click="activeTab = tab.key"
          >
            {{ t(tab.labelKey) }}
          </button>
        </div>
      </section>

      <section class="card p-6" :data-testid="`images-panel-${activePanel.key}`">
        <div class="space-y-6">
          <div class="space-y-2">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t(activePanel.panelTitleKey) }}
            </h2>
            <p class="max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-300">
              {{ t(activePanel.panelDescriptionKey) }}
            </p>
          </div>

          <div v-if="activeTab === 'history'" class="grid gap-6 xl:grid-cols-[minmax(320px,0.9fr)_minmax(0,1.1fr)]">
            <ImageHistoryList
              :error="historyListError"
              :items="historyItems"
              :loading="isHistoryListLoading"
              :selected-id="selectedHistoryId"
              @retry="handleHistoryRetry"
              @select="handleHistorySelect"
            />

            <ImageHistoryDetail
              :detail="historyDetail"
              :error="historyDetailError"
              :loading="isHistoryDetailLoading"
              @replay="handleHistoryReplay"
            />
          </div>

          <div v-else class="grid gap-6 xl:grid-cols-[minmax(390px,0.85fr)_minmax(0,1.15fr)] 2xl:grid-cols-[minmax(420px,0.78fr)_minmax(0,1.22fr)]">
            <div class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 xl:self-start">
              <p
                v-if="activeTab === 'edit' && editReplayNotice"
                class="mb-5 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-900/40 dark:bg-amber-900/10 dark:text-amber-300"
                data-testid="image-edit-replay-notice"
              >
                {{ editReplayNotice }}
              </p>

              <ImageGenerateForm
                v-if="activeTab === 'generate'"
                :key="`generate-${generateFormKey}`"
                :disabled="!canSubmitWithApiKey"
                :initial-values="generateReplayValues"
                :loading="isLoading"
                :loading-seconds="loadingSeconds"
                @submit="handleGenerateSubmit"
              />
              <ImageEditForm
                v-else
                :key="`edit-${editFormKey}`"
                :disabled="!canSubmitWithApiKey"
                :initial-values="editReplayValues"
                :loading="isLoading"
                :loading-seconds="loadingSeconds"
                @submit="handleEditSubmit"
              />
            </div>

            <div class="space-y-6">
              <div class="grid gap-3 rounded-2xl border border-dashed border-gray-300 bg-gray-50/80 p-4 text-sm text-gray-500 dark:border-dark-600 dark:bg-dark-900/60 dark:text-gray-400">
                <div class="flex items-center justify-between">
                  <span>{{ t('images.summary.selectedTab') }}</span>
                  <span class="font-medium text-gray-900 dark:text-white">{{ t(activePanel.labelKey) }}</span>
                </div>
                <div class="flex items-center justify-between">
                  <span>{{ t('images.summary.selectedKey') }}</span>
                  <span class="font-medium text-gray-900 dark:text-white">{{ selectedApiKeyLabel }}</span>
                </div>
                <div class="h-px bg-gray-200 dark:bg-dark-700"></div>
                <p>
                  {{ t('images.summary.placeholder') }}
                </p>
              </div>

              <ImageResultPanel :error="error" :loading="isLoading" :results="results" />
            </div>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import { keysAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import ImageApiKeySelector from '@/components/user/images/ImageApiKeySelector.vue'
import ImageEditForm from '@/components/user/images/ImageEditForm.vue'
import ImageGenerateForm from '@/components/user/images/ImageGenerateForm.vue'
import ImageHistoryDetail from '@/components/user/images/ImageHistoryDetail.vue'
import ImageHistoryList from '@/components/user/images/ImageHistoryList.vue'
import ImageResultPanel from '@/components/user/images/ImageResultPanel.vue'
import type { ImageCommonFormValues } from '@/composables/useImageFormOptions'
import { useImageGeneration } from '@/composables/useImageGeneration'
import { useImageHistory } from '@/composables/useImageHistory'
import type { ApiKey } from '@/types'
import type { ImageGenerationRequest, ImageHistoryDetail as ImageHistoryDetailType } from '@/types'

type ImagesTabKey = 'generate' | 'edit' | 'history'
type ApiKeyLoadState = 'loading' | 'success' | 'error'

interface ImagesTabConfig {
  key: ImagesTabKey
  labelKey: string
  panelTitleKey: string
  panelDescriptionKey: string
}

const { t } = useI18n()

const tabs: ImagesTabConfig[] = [
  {
    key: 'generate',
    labelKey: 'images.tabs.generate',
    panelTitleKey: 'images.panels.generate.title',
    panelDescriptionKey: 'images.panels.generate.description'
  },
  {
    key: 'edit',
    labelKey: 'images.tabs.edit',
    panelTitleKey: 'images.panels.edit.title',
    panelDescriptionKey: 'images.panels.edit.description'
  },
  {
    key: 'history',
    labelKey: 'images.tabs.history',
    panelTitleKey: 'images.panels.history.title',
    panelDescriptionKey: 'images.panels.history.description'
  }
]

const apiKeys = ref<ApiKey[]>([])
const apiKeyLoadState = ref<ApiKeyLoadState>('loading')
const selectedApiKeyId = ref('')
const activeTab = ref<ImagesTabKey>('generate')
const { error, isLoading, loadingSeconds, results, submitEdit, submitGenerate } = useImageGeneration()
const {
  detail: historyDetail,
  detailError: historyDetailError,
  detailState: historyDetailState,
  items: historyItems,
  listError: historyListError,
  listState: historyListState,
  loadHistory,
  selectedHistoryId,
  selectHistory,
} = useImageHistory()
const generateFormKey = ref(0)
const editFormKey = ref(0)
const generateReplayValues = ref<Partial<ImageCommonFormValues>>({})
const editReplayValues = ref<Partial<ImageCommonFormValues>>({})
const editReplayNotice = ref('')

const activePanel = computed(() => tabs.find((tab) => tab.key === activeTab.value) ?? tabs[0])
const canSubmitWithApiKey = computed(() => apiKeyLoadState.value === 'success' && selectedApiKeyId.value.trim().length > 0)
const isHistoryListLoading = computed(() => historyListState.value === 'loading')
const isHistoryDetailLoading = computed(() => historyDetailState.value === 'loading')
const selectedApiKey = computed(() => apiKeys.value.find((key) => String(key.id) === selectedApiKeyId.value) ?? null)

const keySelectorPlaceholder = computed(() => {
  if (apiKeyLoadState.value === 'loading') {
    return t('images.keySelector.loading')
  }

  if (apiKeyLoadState.value === 'error') {
    return t('images.keySelector.loadFailed')
  }

  return t('images.keySelector.placeholder')
})

const keyStatusMessage = computed(() => {
  if (apiKeyLoadState.value === 'loading') {
    return t('images.keySelector.loading')
  }

  if (apiKeyLoadState.value === 'error') {
    return t('images.keySelector.loadFailed')
  }

  if (apiKeys.value.length === 0) {
    return t('images.keySelector.empty')
  }

  return t('images.keySelector.count', { count: apiKeys.value.length })
})

const selectedApiKeyLabel = computed(() => {
  if (apiKeyLoadState.value === 'loading') {
    return t('images.keySelector.loading')
  }

  if (apiKeyLoadState.value === 'error') {
    return t('images.keySelector.loadFailed')
  }

  if (!selectedApiKeyId.value) {
    return t('images.keySelector.placeholder')
  }

  return apiKeys.value.find((key) => String(key.id) === selectedApiKeyId.value)?.name ?? t('images.keySelector.placeholder')
})

async function loadApiKeys() {
  apiKeyLoadState.value = 'loading'

  try {
    const response = await keysAPI.list(1, 100, { sort_by: 'created_at', sort_order: 'asc' })
    apiKeys.value = response.items ?? []
    apiKeyLoadState.value = 'success'

    if (apiKeys.value.length === 0) {
      selectedApiKeyId.value = ''
      return
    }

    const selectedKeyExistsOnCurrentPage = apiKeys.value.some((key) => String(key.id) === selectedApiKeyId.value)
    if (!selectedKeyExistsOnCurrentPage) {
      selectedApiKeyId.value = String(apiKeys.value[0].id)
    }
  } catch (error) {
    apiKeys.value = []
    selectedApiKeyId.value = ''
    apiKeyLoadState.value = 'error'
    console.error('Failed to load image api keys:', error)
  }
}

async function handleGenerateSubmit(payload: ImageGenerationRequest) {
  if (!canSubmitWithApiKey.value) {
    return
  }

  const selectedApiKeyValue = selectedApiKey.value?.key?.trim()
  if (!selectedApiKeyValue) {
    return
  }

  await submitGenerate(payload, selectedApiKeyValue)
}

async function handleEditSubmit(payload: FormData) {
  if (!canSubmitWithApiKey.value) {
    return
  }

  const selectedApiKeyValue = selectedApiKey.value?.key?.trim()
  if (!selectedApiKeyValue) {
    return
  }

  await submitEdit(payload, selectedApiKeyValue)
}

async function handleHistoryRetry() {
  await loadHistory()
}

async function handleHistorySelect(id: number) {
  await selectHistory(id)
}

function toReplayValues(detail: ImageHistoryDetailType): Partial<ImageCommonFormValues> {
  const replayValues: Partial<ImageCommonFormValues> = {}

  function setReplayValue<K extends keyof ImageCommonFormValues>(key: K, value: ImageCommonFormValues[K] | undefined) {
    if (value !== undefined) {
      replayValues[key] = value
    }
  }

  setReplayValue('background', detail.replay.background)
  setReplayValue('model', detail.replay.model)
  setReplayValue('moderation', detail.replay.moderation)
  setReplayValue('output_format', detail.replay.output_format)
  setReplayValue('prompt', detail.replay.prompt)
  setReplayValue('quality', detail.replay.quality)
  setReplayValue('size', detail.replay.size)

  return replayValues
}

function clearGenerateReplayState() {
  if (Object.keys(generateReplayValues.value).length === 0) {
    return
  }

  generateReplayValues.value = {}
  generateFormKey.value += 1
}

function clearEditReplayState() {
  const hadReplayValues = Object.keys(editReplayValues.value).length > 0
  const hadReplayNotice = editReplayNotice.value.length > 0

  editReplayValues.value = {}
  editReplayNotice.value = ''

  if (hadReplayValues || hadReplayNotice) {
    editFormKey.value += 1
  }
}

function handleHistoryReplay(detail: ImageHistoryDetailType) {
  if (detail.replay.mode === 'edit') {
    clearGenerateReplayState()
    editReplayValues.value = toReplayValues(detail)
    editFormKey.value += 1
    editReplayNotice.value = t('images.history.replayEditNotice')
    activeTab.value = 'edit'
    return
  }

  clearEditReplayState()
  generateReplayValues.value = toReplayValues(detail)
  generateFormKey.value += 1
  activeTab.value = 'generate'
}

watch(activeTab, async (tab, previousTab) => {
  if (previousTab === 'generate' && tab !== 'generate') {
    clearGenerateReplayState()
  }

  if (previousTab === 'edit' && tab !== 'edit') {
    clearEditReplayState()
  }

  if (tab === 'history') {
    await loadHistory()
  }
})

onMounted(async () => {
  await loadApiKeys()
})
</script>
