<template>
  <BaseDialog :show="show" :title="t('admin.modeltrace.title')" width="wide" @close="close">
    <p class="mb-4 break-words font-medium text-gray-900 dark:text-white">{{ account?.name }}</p>
    <div role="tablist" :aria-label="t('admin.modeltrace.title')" class="mb-5 flex border-b border-gray-200 dark:border-dark-600">
      <button v-for="key in tabs" :id="`${id}-${key}-tab`" :key="key" type="button" role="tab"
        :aria-selected="tab === key" :aria-controls="`${id}-${key}-panel`" :tabindex="tab === key ? 0 : -1"
        class="border-b-2 px-4 py-3 text-sm font-medium focus-visible:outline focus-visible:outline-primary-500"
        :class="tab === key ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 dark:text-gray-400'"
        @click="tab = key" @keydown="tabKeydown($event, key)">{{ t(`admin.modeltrace.${key}`) }}</button>
    </div>
    <div v-show="tab === 'start'" :id="`${id}-start-panel`" role="tabpanel" :aria-labelledby="`${id}-start-tab`" tabindex="0" class="space-y-4">
      <p class="text-sm text-amber-700 dark:text-amber-300">{{ t('admin.modeltrace.costNotice') }}</p>
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.modeltrace.observationNotice') }}</p>
      <div v-if="modelsError" role="alert" class="text-sm text-red-600 dark:text-red-400">
        {{ modelsError }} <button type="button" class="btn btn-secondary btn-sm" @click="loadModels">{{ t('admin.modeltrace.retry') }}</button>
      </div>
      <form class="flex flex-wrap items-end gap-4" @submit.prevent="startTask">
        <div class="w-full min-w-0 flex-auto sm:w-auto sm:flex-1">
          <label :for="`${id}-model`" class="input-label">{{ t('admin.modeltrace.model') }}</label>
          <select :id="`${id}-model`" v-model="model" class="input w-full" :disabled="modelsLoading || !models.length || isActive || starting">
            <option v-if="!models.length" value="">{{ t(modelsLoading ? 'common.loading' : 'admin.modeltrace.noModels') }}</option>
            <option v-for="name in models" :key="name" :value="name">{{ name }}</option>
          </select>
        </div>
        <div class="w-28">
          <label :for="`${id}-rounds`" class="input-label">{{ t('admin.modeltrace.rounds') }}</label>
          <select :id="`${id}-rounds`" v-model.number="rounds" class="input w-full" :disabled="isActive || starting">
            <option v-for="n in 3" :key="n" :value="n">{{ n }}</option>
          </select>
        </div>
        <button type="submit" data-testid="modeltrace-start" class="btn btn-primary" :disabled="!canStart">
          <Icon name="play" size="sm" />{{ t(starting ? 'common.loading' : 'admin.modeltrace.start') }}
        </button>
      </form>
      <p v-if="version" class="text-xs text-gray-500">{{ t('admin.modeltrace.version') }}: {{ version }}</p>
      <p v-if="startError" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ startError }}</p>
      <p v-if="isActive" class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modeltrace.backgroundNotice') }}</p>
      <ModelTraceTaskDetails v-if="task" :task="task" class="border-t border-gray-200 pt-4 dark:border-dark-600" />
    </div>
    <div v-show="tab === 'history'" :id="`${id}-history-panel`" role="tabpanel" :aria-labelledby="`${id}-history-tab`" tabindex="0" class="space-y-4">
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.modeltrace.retentionNotice') }}</p>
      <div class="flex justify-end"><button type="button" class="btn btn-secondary btn-sm" :disabled="historyLoading || starting" @click="loadHistory"><Icon name="refresh" size="sm" />{{ t('admin.modeltrace.refresh') }}</button></div>
      <p v-if="!historyLoading && historyReady && !items.length" class="py-6 text-center text-sm text-gray-500">{{ t('admin.modeltrace.noHistory') }}</p>
      <div v-for="item in items" :key="item.id" class="border-b border-gray-200 py-4 last:border-0 dark:border-dark-600">
        <ModelTraceTaskDetails :task="item" />
      </div>
      <Pagination v-if="total > 0" :page="page" :total="total" :page-size="20" :show-page-size-selector="false" @update:page="changePage" />
    </div>
    <p v-if="historyLoading && !historyReady" role="status" class="mt-4 text-sm text-gray-500">{{ t('common.loading') }}</p>
    <div v-if="historyError" role="alert" class="mt-4 text-sm text-red-600 dark:text-red-400">
      {{ historyError }} <button type="button" class="btn btn-secondary btn-sm" :disabled="historyLoading || starting" @click="loadHistory">{{ t('admin.modeltrace.retry') }}</button>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, getCurrentInstance, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import modeltraceAPI, { type ModelTraceTask } from '@/api/admin/modeltrace'
import type { AccountListItem } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelTraceTaskDetails from './ModelTraceTaskDetails.vue'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = withDefaults(defineProps<{ show: boolean; account: Pick<AccountListItem, 'id' | 'name' | 'platform'> | null; initialTab?: 'start' | 'history' }>(), { initialTab: 'start' })
const emit = defineEmits<{ close: []; completed: [accountId: number] }>()
const { t } = useI18n()
const id = `modeltrace-${getCurrentInstance()?.uid}`
const tabs = ['start', 'history'] as const
const tab = ref<'start' | 'history'>('start')
const models = ref<string[]>([])
const model = ref('')
const rounds = ref(1)
const version = ref('')
const modelsLoading = ref(false)
const modelsError = ref('')
const historyLoading = ref(false)
const historyReady = ref(false)
const historyError = ref('')
const startError = ref('')
const starting = ref(false)
const items = ref<ModelTraceTask[]>([])
const task = ref<ModelTraceTask | null>(null)
const page = ref(1)
const total = ref(0)
let generation = 0
let historyRequest = 0
let modelsRequest = 0
let timer: ReturnType<typeof setTimeout> | undefined
const active = (value: ModelTraceTask | null) => value?.status === 'queued' || value?.status === 'running'
const isActive = computed(() => active(task.value))
const canStart = computed(() => historyReady.value && !historyError.value && !historyLoading.value && !starting.value && !isActive.value && !modelsLoading.value && !modelsError.value && models.value.includes(model.value) && Number.isInteger(rounds.value) && rounds.value >= 1 && rounds.value <= 3)
const current = (g: number, accountId: number) => g === generation && props.show && props.account?.id === accountId
const errorMessage = (error: unknown, key: string) => `${t(key)}: ${extractApiErrorMessage(error, t('common.unknownError'))}`
function stopPolling() { clearTimeout(timer); timer = undefined }
function invalidate() { generation++; historyRequest++; modelsRequest++; stopPolling() }
function close() { invalidate(); emit('close') }

async function loadModels() {
  const accountId = props.account?.id
  if (!props.show || accountId == null) return
  const g = generation
  const request = ++modelsRequest
  modelsLoading.value = true
  modelsError.value = ''
  try {
    const data = await modeltraceAPI.getModels(accountId)
    if (!current(g, accountId) || request !== modelsRequest) return
    models.value = data.models
    version.value = data.version
    if (!models.value.includes(model.value)) model.value = models.value.includes('gpt-6-astra') ? 'gpt-6-astra' : (models.value[0] || '')
  } catch (error) {
    if (current(g, accountId) && request === modelsRequest) modelsError.value = errorMessage(error, 'admin.modeltrace.modelsLoadFailed')
  } finally {
    if (current(g, accountId) && request === modelsRequest) modelsLoading.value = false
  }
}

async function loadHistory() {
  const accountId = props.account?.id
  if (!props.show || accountId == null || starting.value) return
  stopPolling()
  const g = generation
  const request = ++historyRequest
  const requestedPage = page.value
  const previous = task.value
  historyLoading.value = true
  historyError.value = ''
  try {
    const data = await modeltraceAPI.list(accountId, requestedPage, 20)
    if (!current(g, accountId) || request !== historyRequest) return
    let nextTask = data.active || data.items.find(item => item.id === previous?.id) || (!previous && requestedPage === 1 ? data.items[0] : previous) || null
    // A terminal task may no longer appear on the history page being viewed.
    if (active(previous) && !data.active && active(nextTask) && requestedPage !== 1) {
      const latest = await modeltraceAPI.list(accountId, 1, 20)
      if (!current(g, accountId) || request !== historyRequest) return
      nextTask = latest.active || latest.items.find(item => item.id === previous?.id) || nextTask
    }
    items.value = data.items
    total.value = data.total
    page.value = data.page
    historyReady.value = true
    task.value = nextTask
    if (active(previous) && nextTask && !active(nextTask)) emit('completed', accountId)
    if (active(nextTask)) timer = setTimeout(() => void loadHistory(), 2000)
  } catch (error) {
    if (current(g, accountId) && request === historyRequest) historyError.value = errorMessage(error, 'admin.modeltrace.historyLoadFailed')
  } finally {
    if (current(g, accountId) && request === historyRequest) historyLoading.value = false
  }
}

async function startTask() {
  if (!canStart.value || !props.account) return
  const accountId = props.account.id
  const g = generation
  stopPolling()
  historyRequest++
  starting.value = true
  startError.value = ''
  try {
    const created = await modeltraceAPI.start(accountId, { model: model.value, rounds: rounds.value })
    if (!current(g, accountId)) return
    task.value = created
    page.value = 1
    if (!active(created)) emit('completed', accountId)
  } catch (error) {
    if (current(g, accountId)) startError.value = errorMessage(error, 'admin.modeltrace.startFailed')
  } finally {
    if (current(g, accountId)) {
      starting.value = false
      // Refresh even after an ambiguous network error: the backend may have created the task.
      void loadHistory()
    }
  }
}

function changePage(value: number) { page.value = value; void loadHistory() }
function tabKeydown(event: KeyboardEvent, key: 'start' | 'history') {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  tab.value = event.key === 'Home' ? 'start' : event.key === 'End' ? 'history' : key === 'start' ? 'history' : 'start'
  void nextTick(() => document.getElementById(`${id}-${tab.value}-tab`)?.focus())
}
watch(() => [props.show, props.account?.id, props.account?.platform], () => {
  invalidate()
  task.value = null
  items.value = []
  total.value = 0
  page.value = 1
  model.value = ''
  models.value = []
  rounds.value = 1
  version.value = ''
  starting.value = false
  modelsLoading.value = false
  historyLoading.value = false
  historyReady.value = false
  startError.value = historyError.value = modelsError.value = ''
  tab.value = props.initialTab
  if (props.show && props.account?.platform === 'openai') {
    void loadModels()
    void loadHistory()
  }
}, { immediate: true, flush: 'sync' })
onBeforeUnmount(invalidate)
</script>
