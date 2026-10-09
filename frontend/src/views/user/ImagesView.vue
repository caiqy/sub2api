<template>
  <AppLayout>
    <div class="w-full space-y-5" data-testid="images-view">
      <header class="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between" data-testid="images-workbench-toolbar">
        <div><h1 class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">{{ t('images.title') }}</h1><p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('images.description') }}</p></div>
        <div class="min-w-0 space-y-2 xl:max-w-lg">
          <ImageApiKeySelector v-model="selectedApiKeyId" :api-keys="apiKeys" compact :disabled="apiKeyLoadState !== 'success' || isLoading" :label="t('images.keySelector.label')" :load-state="apiKeyLoadState" :page-hint="t('images.keySelector.pageHint')" :placeholder="keySelectorPlaceholder" :retry-label="t('images.keySelector.retry')" :status-message="keyStatusMessage" @retry="loadApiKeys(keyPage)" />
          <div v-if="keyPages > 1" class="flex items-center justify-end gap-3 text-xs text-gray-500 dark:text-gray-400">
            <button class="rounded px-2 py-1 focus-visible:ring-2 focus-visible:ring-primary-500 disabled:opacity-40" type="button" :disabled="keyPage <= 1 || apiKeyLoadState === 'loading' || isLoading" data-testid="image-keys-previous" @click="loadApiKeys(keyPage - 1)">{{ t('pagination.previous') }}</button>
            <span>{{ t('pagination.pageOf', { page: keyPage, total: keyPages }) }}</span>
            <button class="rounded px-2 py-1 focus-visible:ring-2 focus-visible:ring-primary-500 disabled:opacity-40" type="button" :disabled="keyPage >= keyPages || apiKeyLoadState === 'loading' || isLoading" data-testid="image-keys-next" @click="loadApiKeys(keyPage + 1)">{{ t('pagination.next') }}</button>
          </div>
        </div>
      </header>
      <div class="grid items-start gap-5 xl:grid-cols-[376px_minmax(0,1fr)]" data-testid="images-workbench-layout">
        <section class="min-w-0 rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <div class="mb-5 flex gap-1 rounded-xl bg-gray-100 p-1 dark:bg-dark-900" role="tablist" :aria-label="t('images.tabs.ariaLabel')">
            <button v-for="tab in tabs" :id="`images-tab-${tab}`" :key="tab" :ref="element => setTabRef(tab, element)" :aria-selected="activeTab === tab" :aria-controls="`images-tabpanel-${tab}`" :tabindex="activeTab === tab ? 0 : -1" :class="['flex-1 rounded-lg px-3 py-2.5 text-sm font-medium transition focus-visible:ring-2 focus-visible:ring-primary-500', activeTab === tab ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300' : 'text-gray-500 dark:text-gray-400']" :data-testid="`images-tab-${tab}`" role="tab" type="button" @click="selectTab(tab)" @keydown="handleTabKeydown($event, tab)">{{ t(`images.tabs.${tab}`) }}</button>
          </div>
          <div class="mb-4 text-xs leading-5" role="status" aria-live="polite">
            <p v-if="modelState === 'loading'" class="text-gray-500 dark:text-gray-400">{{ t('images.models.loading') }}</p>
            <div v-else-if="modelState === 'error'" class="rounded-xl bg-amber-50 p-3 text-amber-800 dark:bg-amber-900/20 dark:text-amber-300"><p>{{ t('images.models.failed') }} {{ modelError }}</p><button class="mt-2 underline" type="button" data-testid="image-models-retry" @click="loadModels">{{ t('images.keySelector.retry') }}</button></div>
            <p v-else-if="modelState === 'success' && models.length === 0" class="rounded-xl bg-gray-50 p-3 text-gray-600 dark:bg-dark-900 dark:text-gray-400">{{ t('images.models.empty') }}</p>
            <p v-else-if="models.length" class="text-gray-500 dark:text-gray-400">{{ t('images.models.visibilityNotice') }}</p>
          </div>
          <div id="images-tabpanel-generate" :hidden="activeTab !== 'generate'" role="tabpanel" aria-labelledby="images-tab-generate" data-testid="images-panel-generate">
            <ImageGenerateForm :key="`generate-${generateFormKey}`" :models="models" :disabled="!canSubmitWithApiKey" :initial-values="generateReplayValues" :loading="isLoading" :loading-seconds="loadingSeconds" :show-api-key-required-message="showApiKeyRequiredMessage" @submit="handleGenerateSubmit" />
          </div>
          <div id="images-tabpanel-edit" :hidden="activeTab !== 'edit'" role="tabpanel" aria-labelledby="images-tab-edit" data-testid="images-panel-edit">
            <p v-if="editReplayNotice" class="mb-4 rounded-xl bg-primary-50 px-3 py-2 text-xs leading-5 text-primary-800 dark:bg-primary-900/20 dark:text-primary-200" data-testid="image-edit-replay-notice">{{ editReplayNotice }}</p>
            <ImageEditForm :key="`edit-${editFormKey}`" :models="models" :disabled="!canSubmitWithApiKey" :initial-values="editReplayValues" :initial-files="editInitialFiles" :loading="isLoading" :loading-seconds="loadingSeconds" :show-api-key-required-message="showApiKeyRequiredMessage" @submit="handleEditSubmit" />
          </div>
        </section>
        <div class="min-w-0 space-y-4">
          <ImageResultPanel :duration-ms="durationMs" :error="error" :loading="isLoading" :results="results" :partial="partialResult" :status="status" :edit-disabled="reusePending" @stop="stop" @edit="sendResultToEdit($event)" />
          <p v-if="reuseError" class="rounded-xl bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-900/20 dark:text-amber-300" role="alert">{{ reuseError }}</p>
          <p v-if="reusePending" class="text-sm text-gray-500 dark:text-gray-400" role="status">{{ t('images.results.reading') }}</p>
        </div>
      </div>
      <section class="overflow-hidden rounded-2xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <button class="flex w-full items-center justify-between gap-3 px-5 py-4 text-left focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500" :aria-expanded="historyOpen" aria-controls="images-history-region" data-testid="images-history-toggle" type="button" @click="toggleHistory"><span class="font-semibold text-gray-900 dark:text-white">{{ t('images.tabs.history') }}</span><span class="text-xs text-gray-500 dark:text-gray-400">{{ t(historyOpen ? 'images.history.collapse' : 'images.history.expand') }}</span></button>
        <div v-if="historyOpen" id="images-history-region" class="border-t border-gray-200 p-4 dark:border-dark-700 sm:p-5">
          <div class="mb-4 flex flex-wrap items-end gap-3">
            <label class="min-w-0 flex-1 text-xs text-gray-500 dark:text-gray-400">{{ t('images.history.apiKey') }}<select v-model="historyFilters.apiKey" class="input mt-1 w-full" data-testid="image-history-key-filter"><option value="">{{ t('images.history.allKeys') }}</option><option v-if="historyFilters.apiKey && !apiKeys.some(key => String(key.id) === historyFilters.apiKey)" :value="historyFilters.apiKey">{{ t('images.history.keyId', { id: historyFilters.apiKey }) }}</option><option v-for="key in apiKeys" :key="key.id" :value="String(key.id)">{{ key.name }}</option></select></label>
            <label class="min-w-[120px] flex-1 text-xs text-gray-500 dark:text-gray-400">{{ t('images.history.mode') }}<select v-model="historyFilters.mode" class="input mt-1 w-full" data-testid="image-history-mode-filter"><option value="">{{ t('images.history.allModes') }}</option><option value="generate">{{ t('images.history.modes.generate') }}</option><option value="edit">{{ t('images.history.modes.edit') }}</option></select></label>
            <label class="min-w-[120px] flex-1 text-xs text-gray-500 dark:text-gray-400">{{ t('images.history.status') }}<select v-model="historyFilters.status" class="input mt-1 w-full" data-testid="image-history-status-filter"><option value="">{{ t('images.history.allStatuses') }}</option><option value="success">{{ t('images.history.statuses.success') }}</option><option value="error">{{ t('images.history.statuses.error') }}</option></select></label>
            <button class="btn btn-secondary" :disabled="isHistoryListLoading" type="button" data-testid="image-history-refresh" @click="loadHistory(historyQuery(historyPage), { force: true })">{{ t('images.history.refresh') }}</button>
          </div>
          <p class="mb-4 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('images.history.retentionNotice') }}</p>
          <div class="grid items-start gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
            <ImageHistoryList :error="historyListError" :items="historyItems" :loading="isHistoryListLoading" :selected-id="selectedHistoryId" @retry="loadHistory(historyQuery(historyPage), { force: true })" @select="selectHistory" />
            <ImageHistoryDetail :detail="historyDetail" :error="historyDetailError" :unavailable="detailUnavailable" :loading="isHistoryDetailLoading" :edit-disabled="isLoading || reusePending" @replay="handleHistoryReplay" @edit="sendHistoryResultToEdit" />
          </div>
          <Pagination v-if="historyTotal > 0" class="mt-4" :total="historyTotal" :page="historyPage" :page-size="historyPageSize" :show-page-size-selector="false" @update:page="loadHistory(historyQuery($event))" />
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { imagesAPI, keysAPI } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import ImageApiKeySelector from '@/components/user/images/ImageApiKeySelector.vue'
import ImageEditForm from '@/components/user/images/ImageEditForm.vue'
import ImageGenerateForm from '@/components/user/images/ImageGenerateForm.vue'
import ImageHistoryDetail from '@/components/user/images/ImageHistoryDetail.vue'
import ImageHistoryList from '@/components/user/images/ImageHistoryList.vue'
import ImageResultPanel from '@/components/user/images/ImageResultPanel.vue'
import type { ImageCommonFormValues } from '@/composables/useImageFormOptions'
import { useImageGeneration, type ImageResultPreview } from '@/composables/useImageGeneration'
import { useImageHistory } from '@/composables/useImageHistory'
import type { ApiKey, ImageGenerationRequest, ImageHistoryDetail as HistoryDetail, ImageHistoryListParams, ImageHistoryMode, ImageHistoryStatus } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import { imageToFile } from '@/utils/imageResult'

type Mode = 'generate' | 'edit'
type LoadState = 'idle' | 'loading' | 'success' | 'error'
const { t } = useI18n()
const tabs: Mode[] = ['generate', 'edit']
const activeTab = ref<Mode>('generate')
const tabRefs: Partial<Record<Mode, HTMLButtonElement>> = {}
const apiKeys = ref<ApiKey[]>([])
const apiKeyLoadState = ref<'loading' | 'success' | 'error'>('loading')
const selectedApiKeyId = ref('')
const keyPage = ref(1)
const keyPages = ref(1)
const selectedApiKey = computed(() => apiKeys.value.find(key => String(key.id) === selectedApiKeyId.value))
const models = ref<string[]>([])
const modelState = ref<LoadState>('idle')
const modelError = ref('')
let keyRequestId = 0
let modelRequestId = 0
let modelController: AbortController | null = null
let keyController: AbortController | null = null
const { error, isLoading, loadingSeconds, results, partialResult, status, durationMs, stop, submitEdit, submitGenerate } = useImageGeneration()
const {
  detail: historyDetail, detailError: historyDetailError, detailState: historyDetailState, detailUnavailable,
  items: historyItems, listError: historyListError, listState: historyListState,
  page: historyPage, pageSize: historyPageSize, total: historyTotal,
  loadHistory, invalidateHistory, selectedHistoryId, selectHistory,
} = useImageHistory()
const historyOpen = ref(false)
const historyFilters = reactive<{ apiKey: string; mode: ImageHistoryMode | ''; status: ImageHistoryStatus | '' }>({ apiKey: '', mode: '', status: '' })
const isHistoryListLoading = computed(() => historyListState.value === 'loading')
const isHistoryDetailLoading = computed(() => historyDetailState.value === 'loading')
const generateFormKey = ref(0)
const editFormKey = ref(0)
const generateReplayValues = ref<Partial<ImageCommonFormValues>>({})
const editReplayValues = ref<Partial<ImageCommonFormValues>>({})
const editInitialFiles = ref<File[]>([])
const editReplayNotice = ref('')
const reusePending = ref(false)
const reuseError = ref('')
let reuseController: AbortController | null = null
let lastSubmittedValues: Partial<ImageCommonFormValues> = {}
const canSubmitWithApiKey = computed(() => apiKeyLoadState.value === 'success' && !!selectedApiKey.value?.key?.trim() && modelState.value === 'success' && models.value.length > 0 && !reusePending.value)
const showApiKeyRequiredMessage = computed(() => apiKeyLoadState.value === 'success' && apiKeys.value.length > 0 && !selectedApiKeyId.value)
const keySelectorPlaceholder = computed(() => t(apiKeyLoadState.value === 'loading' ? 'images.keySelector.loading' : apiKeyLoadState.value === 'error' ? 'images.keySelector.loadFailed' : 'images.keySelector.placeholder'))
const keyStatusMessage = computed(() => {
  if (apiKeyLoadState.value === 'loading') return t('images.keySelector.loading')
  if (apiKeyLoadState.value === 'error') return t('images.keySelector.loadFailed')
  return apiKeys.value.length ? t('images.keySelector.count', { count: apiKeys.value.length }) : t('images.keySelector.empty')
})
async function loadApiKeys(page = 1) {
  const requestId = ++keyRequestId
  keyController?.abort()
  keyController = new AbortController()
  apiKeyLoadState.value = 'loading'
  try {
    const response = await keysAPI.list(page, 100, { sort_by: 'created_at', sort_order: 'asc' }, { signal: keyController.signal })
    if (requestId !== keyRequestId) return
    apiKeys.value = response.items ?? []
    keyPage.value = response.page ?? page
    keyPages.value = response.pages ?? Math.max(1, Math.ceil((response.total ?? apiKeys.value.length) / 100))
    apiKeyLoadState.value = 'success'
    if (!apiKeys.value.some(key => String(key.id) === selectedApiKeyId.value)) selectedApiKeyId.value = apiKeys.value.length ? String(apiKeys.value[0].id) : ''
  } catch {
    if (requestId !== keyRequestId) return
    apiKeys.value = []
    selectedApiKeyId.value = ''
    apiKeyLoadState.value = 'error'
  }
}
async function loadModels() {
  const requestId = ++modelRequestId
  modelController?.abort()
  const key = selectedApiKey.value?.key?.trim()
  models.value = []
  modelError.value = ''
  modelState.value = key ? 'loading' : 'idle'
  if (!key) return
  modelController = new AbortController()
  try {
    const candidates = await imagesAPI.listModels(key, { signal: modelController.signal })
    if (requestId !== modelRequestId) return
    models.value = candidates
    modelState.value = 'success'
  } catch (err) {
    if (requestId !== modelRequestId) return
    modelState.value = 'error'
    modelError.value = extractApiErrorMessage(err, '')
  }
}
function payloadValues(payload: Partial<ImageCommonFormValues> | FormData): Partial<ImageCommonFormValues> {
  const values: Partial<ImageCommonFormValues> = {}
  for (const key of ['prompt', 'model', 'size', 'quality', 'background', 'output_format', 'moderation'] as const) {
    const value = payload instanceof FormData ? payload.get(key) : payload[key]
    if (typeof value === 'string') values[key] = value
  }
  return values
}
async function handleGenerateSubmit(payload: ImageGenerationRequest) {
  if (activeTab.value !== 'generate' || !canSubmitWithApiKey.value || isLoading.value) return
  lastSubmittedValues = payloadValues(payload)
  const response = await submitGenerate(payload, selectedApiKey.value!.key)
  invalidateHistory()
  if (response && historyOpen.value) await loadHistory(historyQuery(historyPage.value), { force: true })
}
async function handleEditSubmit(payload: FormData) {
  if (activeTab.value !== 'edit' || !canSubmitWithApiKey.value || isLoading.value) return
  lastSubmittedValues = payloadValues(payload)
  const response = await submitEdit(payload, selectedApiKey.value!.key)
  invalidateHistory()
  if (response && historyOpen.value) await loadHistory(historyQuery(historyPage.value), { force: true })
}
function historyQuery(page = 1): ImageHistoryListParams {
  return { page, page_size: 20, api_key_id: historyFilters.apiKey ? Number(historyFilters.apiKey) : undefined, tab: historyFilters.mode || undefined, status: historyFilters.status || undefined }
}
async function toggleHistory() {
  historyOpen.value = !historyOpen.value
  if (historyOpen.value) await loadHistory(historyQuery(historyPage.value))
}
function setTabRef(tab: Mode, element: Element | ComponentPublicInstance | null) {
  if (element instanceof HTMLButtonElement) tabRefs[tab] = element
}
async function selectTab(tab: Mode) {
  activeTab.value = tab
  await nextTick()
  tabRefs[tab]?.focus()
}
function handleTabKeydown(event: KeyboardEvent, tab: Mode) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  selectTab(event.key === 'Home' ? 'generate' : event.key === 'End' ? 'edit' : tab === 'generate' ? 'edit' : 'generate')
}
function toReplayValues(detail: HistoryDetail): Partial<ImageCommonFormValues> {
  return payloadValues(detail.replay)
}
async function handleHistoryReplay(detail: HistoryDetail) {
  if (isLoading.value || reusePending.value) return
  if (detail.replay.mode === 'edit') {
    editReplayValues.value = toReplayValues(detail)
    editInitialFiles.value = []
    editFormKey.value += 1
    editReplayNotice.value = t('images.history.replayEditNotice')
  } else {
    generateReplayValues.value = toReplayValues(detail)
    generateFormKey.value += 1
  }
  await selectTab(detail.replay.mode)
}
async function sendResultToEdit(image: ImageResultPreview, values = lastSubmittedValues) {
  if (isLoading.value || reusePending.value) return
  reuseController?.abort()
  const controller = new AbortController()
  reuseController = controller
  reusePending.value = true
  reuseError.value = ''
  try {
    const file = await imageToFile(image.src, controller.signal)
    if (controller.signal.aborted) return
    if (file.size > 20 * 1024 * 1024) throw new Error('image-too-large')
    editReplayValues.value = { ...values, prompt: '', size: 'auto' }
    editInitialFiles.value = [file]
    editFormKey.value += 1
    editReplayNotice.value = t('images.results.editNotice')
    await selectTab('edit')
    await nextTick()
    document.getElementById('image-edit-prompt')?.focus()
  } catch (err) {
    if (!controller.signal.aborted) reuseError.value = t(err instanceof Error && err.message === 'image-too-large' ? 'images.forms.edit.sourceImageTooLarge' : 'images.results.readFailed')
  } finally {
    if (reuseController === controller) reusePending.value = false
  }
}
function sendHistoryResultToEdit(image: ImageResultPreview) {
  if (historyDetail.value) sendResultToEdit(image, toReplayValues(historyDetail.value))
}
watch(() => selectedApiKey.value?.key, loadModels)
watch(historyFilters, () => {
  historyPage.value = 1
  if (historyOpen.value) loadHistory(historyQuery())
})
onMounted(() => loadApiKeys())
onBeforeUnmount(() => {
  keyRequestId += 1
  modelRequestId += 1
  keyController?.abort()
  modelController?.abort()
  reuseController?.abort()
})
</script>
