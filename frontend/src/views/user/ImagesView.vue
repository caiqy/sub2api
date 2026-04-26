<template>
  <AppLayout>
    <div class="w-full space-y-4" data-testid="images-view">
      <section
        class="card border border-gray-200/80 bg-white/90 p-2 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-800/90"
        data-testid="images-workbench-toolbar"
      >
        <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex flex-wrap gap-2" role="tablist" :aria-label="t('images.tabs.ariaLabel')">
            <button
              v-for="tab in tabs"
              :key="tab.key"
              :id="getTabId(tab.key)"
              :aria-selected="activeTab === tab.key"
              :aria-controls="getTabPanelId(tab.key)"
              :class="[
                'rounded-xl px-4 py-2.5 text-sm font-medium transition-all',
                activeTab === tab.key
                  ? 'bg-primary-600 text-white shadow-sm dark:bg-primary-500'
                  : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white'
              ]"
              :data-testid="`images-tab-${tab.key}`"
              :ref="(element) => setTabButtonRef(tab.key, element)"
              :tabindex="activeTab === tab.key ? 0 : -1"
              role="tab"
              type="button"
              @click="selectTab(tab.key)"
              @keydown="handleTabKeydown($event, tab.key)"
            >
              {{ t(tab.labelKey) }}
            </button>
          </div>

          <ImageApiKeySelector
            v-model="selectedApiKeyId"
            class="lg:min-w-[320px]"
            :api-keys="apiKeys"
            compact
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

      <section
        v-for="tab in tabs"
        :key="tab.key"
        class="card p-2"
        :id="getTabPanelId(tab.key)"
        :aria-labelledby="getTabId(tab.key)"
        :data-testid="`images-panel-${tab.key}`"
        :hidden="tab.key !== activePanel.key"
        role="tabpanel"
      >
        <div v-if="tab.key === activePanel.key" class="space-y-5">
          <div v-if="tab.key === 'history'" class="grid gap-4 xl:grid-cols-[minmax(360px,390px)_minmax(0,1fr)]" data-testid="images-panel-layout-history">
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

          <div v-else :data-testid="`images-panel-layout-${tab.key}`" class="grid gap-4 xl:grid-cols-[minmax(360px,390px)_minmax(0,1fr)]">
            <div class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 xl:self-start">
              <div class="mb-5 space-y-2">
                <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
                  {{ t(tab.panelTitleKey) }}
                </h2>
                <p class="max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-300">
                  {{ t(tab.panelDescriptionKey) }}
                </p>
              </div>

              <p
                v-if="tab.key === 'edit' && editReplayNotice"
                class="mb-5 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-900/40 dark:bg-amber-900/10 dark:text-amber-300"
                data-testid="image-edit-replay-notice"
              >
                {{ editReplayNotice }}
              </p>

              <ImageGenerateForm
                v-if="tab.key === 'generate'"
                :key="`generate-${generateFormKey}`"
                :disabled="!canSubmitWithApiKey"
                :initial-values="generateReplayValues"
                :loading="isLoading"
                :loading-seconds="loadingSeconds"
                :show-api-key-required-message="showApiKeyRequiredMessage"
                @submit="handleGenerateSubmit"
              />
              <ImageEditForm
                v-else
                :key="`edit-${editFormKey}`"
                :disabled="!canSubmitWithApiKey"
                :initial-values="editReplayValues"
                :loading="isLoading"
                :loading-seconds="loadingSeconds"
                :show-api-key-required-message="showApiKeyRequiredMessage"
                @submit="handleEditSubmit"
              />
            </div>

            <ImageResultPanel :duration-ms="lastResultDurationMs" :error="error" :loading="isLoading" :results="results" />
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch, type ComponentPublicInstance } from 'vue'
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
const lastResultDurationMs = ref<number | null>(null)
const tabButtonRefs = ref<Record<ImagesTabKey, HTMLButtonElement | null>>({
  generate: null,
  edit: null,
  history: null
})

const activePanel = computed(() => tabs.find((tab) => tab.key === activeTab.value) ?? tabs[0])
const canSubmitWithApiKey = computed(() => apiKeyLoadState.value === 'success' && selectedApiKeyId.value.trim().length > 0)
const showApiKeyRequiredMessage = computed(() => apiKeyLoadState.value === 'success' && apiKeys.value.length > 0 && selectedApiKeyId.value.trim().length === 0)
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

  lastResultDurationMs.value = null
  const response = await submitGenerate(payload, selectedApiKeyValue)
  if (response) {
    await loadHistory()
    lastResultDurationMs.value = findLatestMatchingHistoryDuration('generate', payload)
  }
}

async function handleEditSubmit(payload: FormData) {
  if (!canSubmitWithApiKey.value) {
    return
  }

  const selectedApiKeyValue = selectedApiKey.value?.key?.trim()
  if (!selectedApiKeyValue) {
    return
  }

  lastResultDurationMs.value = null
  const response = await submitEdit(payload, selectedApiKeyValue)
  if (response) {
    await loadHistory()
    lastResultDurationMs.value = findLatestMatchingHistoryDuration('edit', payload)
  }
}

function findLatestMatchingHistoryDuration(mode: 'generate' | 'edit', payload: ImageGenerationRequest | FormData) {
  const prompt = payload instanceof FormData ? String(payload.get('prompt') ?? '').trim() : String(payload.prompt ?? '').trim()
  const model = payload instanceof FormData ? String(payload.get('model') ?? '').trim() : String(payload.model ?? '').trim()

  const match = historyItems.value.find((item) => {
    if (item.mode !== mode || item.status !== 'success') {
      return false
    }

    if (model && item.model !== model) {
      return false
    }

    if (prompt && item.prompt && item.prompt !== prompt) {
      return false
    }

    return typeof item.duration_ms === 'number'
  })

  return match?.duration_ms ?? null
}

async function handleHistoryRetry() {
  await loadHistory()
}

async function handleHistorySelect(id: number) {
  await selectHistory(id)
}

function getTabId(tabKey: ImagesTabKey): string {
  return `images-tab-${tabKey}`
}

function getTabPanelId(tabKey: ImagesTabKey): string {
  return `images-tabpanel-${tabKey}`
}

function setTabButtonRef(tabKey: ImagesTabKey, element: Element | ComponentPublicInstance | null) {
  tabButtonRefs.value[tabKey] = element instanceof HTMLButtonElement ? element : null
}

async function focusTab(tabKey: ImagesTabKey) {
  await nextTick()
  tabButtonRefs.value[tabKey]?.focus()
}

async function selectTab(tabKey: ImagesTabKey) {
  activeTab.value = tabKey
  await focusTab(tabKey)
}

async function handleTabKeydown(event: KeyboardEvent, tabKey: ImagesTabKey) {
  const currentIndex = tabs.findIndex((tab) => tab.key === tabKey)
  if (currentIndex === -1) {
    return
  }

  let targetIndex: number | null = null

  switch (event.key) {
    case 'ArrowLeft':
      targetIndex = (currentIndex - 1 + tabs.length) % tabs.length
      break
    case 'ArrowRight':
      targetIndex = (currentIndex + 1) % tabs.length
      break
    case 'Home':
      targetIndex = 0
      break
    case 'End':
      targetIndex = tabs.length - 1
      break
    default:
      return
  }

  event.preventDefault()
  await selectTab(tabs[targetIndex].key)
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

async function handleHistoryReplay(detail: ImageHistoryDetailType) {
  if (detail.replay.mode === 'edit') {
    clearGenerateReplayState()
    editReplayValues.value = toReplayValues(detail)
    editFormKey.value += 1
    editReplayNotice.value = t('images.history.replayEditNotice')
    await selectTab('edit')
    return
  }

  clearEditReplayState()
  generateReplayValues.value = toReplayValues(detail)
  generateFormKey.value += 1
  await selectTab('generate')
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
