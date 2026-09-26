<template>
  <section data-testid="modeltrace-settings" class="card" :aria-labelledby="`${id}-title`">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 :id="`${id}-title`" class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.modeltrace.settingsTitle') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.modeltrace.settingsDescription') }}</p>
    </div>
    <div class="space-y-5 p-6">
      <p v-if="loading" role="status" class="text-sm text-gray-500">{{ t('common.loading') }}</p>
      <div v-if="loadError" role="alert" class="flex flex-wrap items-center gap-3 text-sm text-red-600 dark:text-red-400">
        {{ loadError }}
        <button type="button" class="btn btn-secondary btn-sm" :disabled="loading || saving" @click="load">{{ t('admin.modeltrace.retry') }}</button>
      </div>
      <fieldset :disabled="loading || saving || !loaded || !!loadError" class="space-y-5">
        <div class="flex items-center justify-between gap-4">
          <label :for="`${id}-enabled`" class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.modeltrace.enabled') }}</label>
          <Toggle :id="`${id}-enabled`" v-model="form.enabled" data-testid="modeltrace-global-enabled" :aria-label="t('admin.modeltrace.enabled')" />
        </div>
        <div class="grid gap-4 sm:grid-cols-3">
          <div class="min-w-0">
            <label :for="`${id}-model`" class="input-label">{{ t('admin.modeltrace.model') }}</label>
            <select :id="`${id}-model`" v-model="form.model" data-testid="modeltrace-global-model" class="input w-full">
              <option v-if="!models.includes(form.model)" :value="form.model">{{ form.model || t('admin.modeltrace.noModels') }}</option>
              <option v-for="name in models" :key="name" :value="name">{{ name }}</option>
            </select>
          </div>
          <div>
            <label :for="`${id}-rounds`" class="input-label">{{ t('admin.modeltrace.rounds') }}</label>
            <select :id="`${id}-rounds`" v-model.number="form.rounds" data-testid="modeltrace-global-rounds" class="input w-full">
              <option v-for="n in 3" :key="n" :value="n">{{ n }}</option>
            </select>
          </div>
          <div>
            <label :for="`${id}-interval`" class="input-label">{{ t('admin.modeltrace.interval') }}</label>
            <input :id="`${id}-interval`" v-model.number="form.interval_minutes" data-testid="modeltrace-global-interval" type="number" min="5" max="10080" step="1" class="input w-full" />
          </div>
        </div>
      </fieldset>
      <p v-if="saveError" role="alert" class="text-sm text-red-600 dark:text-red-400">{{ saveError }}</p>
      <div class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700">
        <button type="button" data-testid="modeltrace-settings-save" class="btn btn-primary btn-sm" :disabled="loading || saving || !loaded || !!loadError" @click="save">
          <Icon name="check" size="sm" />{{ t(saving ? 'common.loading' : 'admin.modeltrace.save') }}
        </button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { getCurrentInstance, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import modeltraceAPI, { type ModelTraceSettings } from '@/api/admin/modeltrace'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
const { t } = useI18n()
const appStore = useAppStore()
const id = `modeltrace-settings-${getCurrentInstance()?.uid}`
const form = reactive<ModelTraceSettings>({ enabled: false, model: 'gpt-6-astra', rounds: 1, interval_minutes: 60 })
const models = ref<string[]>([])
const loading = ref(true)
const saving = ref(false)
const loaded = ref(false)
const loadError = ref('')
const saveError = ref('')
let generation = 0
async function load() {
  const g = ++generation
  loading.value = true
  loaded.value = false
  loadError.value = ''
  try {
    const [settings, supported] = await Promise.all([modeltraceAPI.getSettings(), modeltraceAPI.getModels()])
    if (g !== generation) return
    Object.assign(form, settings)
    models.value = supported.models
    loaded.value = true
  } catch (error) {
    if (g !== generation) return
    loadError.value = `${t('admin.modeltrace.settingsLoadFailed')} ${extractApiErrorMessage(error, t('common.unknownError'))}`
  } finally {
    if (g === generation) loading.value = false
  }
}
async function save() {
  if (!loaded.value || loadError.value || loading.value || saving.value) return
  if (!Number.isInteger(form.rounds) || form.rounds < 1 || form.rounds > 3 || !Number.isInteger(form.interval_minutes) || form.interval_minutes < 5 || form.interval_minutes > 10080 || (form.enabled && !models.value.includes(form.model))) {
    saveError.value = t('admin.modeltrace.invalidSettings')
    return
  }
  const g = generation
  saving.value = true
  saveError.value = ''
  try {
    const updated = await modeltraceAPI.updateSettings({ ...form })
    if (g !== generation) return
    Object.assign(form, updated)
    appStore.showSuccess(t('admin.modeltrace.settingsSaved'))
  } catch (error) {
    if (g === generation) saveError.value = `${t('admin.modeltrace.settingsSaveFailed')} ${extractApiErrorMessage(error, t('common.unknownError'))}`
  } finally {
    if (g === generation) saving.value = false
  }
}
onMounted(load)
onBeforeUnmount(() => { generation++ })
</script>
