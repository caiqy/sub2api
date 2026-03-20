<template>
  <section class="space-y-3">
    <div class="space-y-1">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
        {{ t('admin.accounts.passthroughFields.title') }}
      </h3>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.passthroughFields.description') }}
      </p>
      <p
        v-if="showDisabledHint && (disabled || !enabled)"
        class="text-xs text-amber-600 dark:text-amber-400"
      >
        {{ t('admin.accounts.passthroughFields.disabledHint') }}
      </p>
    </div>

    <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
      <input
        :checked="enabled"
        :disabled="disabled"
        data-testid="passthrough-enabled-toggle"
        type="checkbox"
        @change="emit('update:enabled', ($event.target as HTMLInputElement).checked)"
      />
      <span>{{ t('admin.accounts.passthroughFields.title') }}</span>
    </label>

    <div
      :class="[
        'space-y-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600',
        !enabled ? 'opacity-50' : ''
      ]"
      data-testid="passthrough-rules-section"
    >
      <div
        v-for="(rule, index) in localRules"
        :key="rule.id"
        :data-testid="`passthrough-rule-row-${index}`"
        class="space-y-2 rounded-lg border border-gray-100 p-3 dark:border-dark-700"
      >
        <div class="flex flex-wrap items-start gap-2">
          <select
            :value="rule.target"
            :disabled="disabled"
            :data-testid="`passthrough-rule-target-${index}`"
            class="input min-w-[120px]"
            @change="updateRule(index, 'target', ($event.target as HTMLSelectElement).value as PassthroughFieldTarget)"
          >
            <option value="header">{{ t('admin.accounts.passthroughFields.targetHeader') }}</option>
            <option value="body">{{ t('admin.accounts.passthroughFields.targetBody') }}</option>
          </select>

          <select
            :value="rule.mode"
            :disabled="disabled"
            :data-testid="`passthrough-rule-mode-${index}`"
            class="input min-w-[120px]"
            @change="updateRule(index, 'mode', ($event.target as HTMLSelectElement).value as PassthroughFieldMode)"
          >
            <option value="forward">{{ t('admin.accounts.passthroughFields.modeForward') }}</option>
            <option value="inject">{{ t('admin.accounts.passthroughFields.modeInject') }}</option>
          </select>

          <div class="min-w-[220px] flex-1">
            <input
              :value="rule.key"
              :disabled="disabled"
              :data-testid="`passthrough-rule-key-${index}`"
              class="input w-full"
              type="text"
              @input="updateRule(index, 'key', ($event.target as HTMLInputElement).value)"
            />
            <p v-if="validationErrors[index]?.key" class="mt-1 text-xs text-red-500">
              {{ formatValidationError(validationErrors[index]?.key) }}
            </p>
          </div>

          <div v-if="rule.mode === 'inject'" class="min-w-[180px] flex-1">
            <input
              :value="rule.value"
              :disabled="disabled"
              :data-testid="`passthrough-rule-value-${index}`"
              class="input w-full"
              type="text"
              @input="updateRule(index, 'value', ($event.target as HTMLInputElement).value)"
            />
            <p v-if="validationErrors[index]?.value" class="mt-1 text-xs text-red-500">
              {{ formatValidationError(validationErrors[index]?.value) }}
            </p>
          </div>

          <button
            type="button"
            :disabled="disabled"
            :data-testid="`passthrough-rule-delete-${index}`"
            class="rounded border border-red-200 px-3 py-2 text-sm text-red-600 disabled:cursor-not-allowed disabled:opacity-60"
            @click="removeRule(index)"
          >
            {{ t('common.delete') }}
          </button>
        </div>

        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ rule.target === 'header'
            ? t('admin.accounts.passthroughFields.headerHint')
            : t('admin.accounts.passthroughFields.bodyHint') }}
        </p>
        <p
          v-if="rule.mode === 'inject'"
          class="text-xs text-blue-600 dark:text-blue-400"
        >
          {{ t('admin.accounts.passthroughFields.injectHint') }}
        </p>
      </div>

      <button
        type="button"
        :disabled="disabled"
        data-testid="passthrough-add-rule"
        class="rounded border border-primary-200 px-3 py-2 text-sm text-primary-600 disabled:cursor-not-allowed disabled:opacity-60"
        @click="addRule"
      >
        {{ t('admin.accounts.passthroughFields.addRule') }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  createPassthroughFieldRuleDraft,
  type PassthroughFieldRuleErrorCode,
  type PassthroughFieldMode,
  type PassthroughFieldRuleDraft,
  type PassthroughFieldTarget,
  validatePassthroughFieldRules
} from './passthroughFieldRules'

const props = withDefaults(defineProps<{
  enabled: boolean
  rules: PassthroughFieldRuleDraft[]
  disabled?: boolean
  showDisabledHint?: boolean
}>(), {
  disabled: false,
  showDisabledHint: false
})

const emit = defineEmits<{
  'update:enabled': [value: boolean]
  'update:rules': [value: PassthroughFieldRuleDraft[]]
}>()

const { t } = useI18n()

const localRules = ref<PassthroughFieldRuleDraft[]>([])

watch(
  () => props.rules,
  (rules) => {
    if (rules.length === 0) {
      syncRules([createPassthroughFieldRuleDraft()])
      return
    }

    localRules.value = cloneRules(rules)
  },
  { deep: true, immediate: true }
)

const validationErrors = computed(() => validatePassthroughFieldRules(localRules.value).errors)

function cloneRules(rules: PassthroughFieldRuleDraft[]): PassthroughFieldRuleDraft[] {
  return rules.map(rule => ({ ...rule }))
}

function syncRules(rules: PassthroughFieldRuleDraft[]) {
  localRules.value = rules
  emit('update:rules', cloneRules(rules))
}

function updateRule<K extends keyof PassthroughFieldRuleDraft>(
  index: number,
  field: K,
  value: PassthroughFieldRuleDraft[K]
) {
  const nextRules = cloneRules(localRules.value)
  const currentRule = nextRules[index]

  if (!currentRule) {
    return
  }

  currentRule[field] = value
  syncRules(nextRules)
}

function addRule() {
  syncRules([...cloneRules(localRules.value), createPassthroughFieldRuleDraft()])
}

function removeRule(index: number) {
  syncRules(cloneRules(localRules.value).filter((_, currentIndex) => currentIndex !== index))
}

function formatValidationError(error?: PassthroughFieldRuleErrorCode) {
  switch (error) {
    case 'key_required':
      return t('admin.accounts.passthroughFields.errors.keyRequired')
    case 'invalid_body_path':
      return t('admin.accounts.passthroughFields.errors.bodyPath')
    case 'value_required':
      return t('admin.accounts.passthroughFields.errors.valueRequired')
    case 'duplicate_key':
      return t('admin.accounts.passthroughFields.errors.duplicateKey')
    case 'reserved_key':
      return t('admin.accounts.passthroughFields.errors.reservedKey')
    default:
      return error ?? ''
  }
}
</script>
