export type PassthroughFieldTarget = 'header' | 'body'
export type PassthroughFieldMode = 'forward' | 'inject' | 'map'

export interface PassthroughFieldRuleDraft {
  id: string
  target: PassthroughFieldTarget
  mode: PassthroughFieldMode
  key: string
  source_key: string
  value: string
}

export interface PassthroughFieldRuleErrors {
  key?: PassthroughFieldRuleErrorCode
  source_key?: PassthroughFieldRuleErrorCode
  value?: PassthroughFieldRuleErrorCode
}

export type PassthroughFieldRuleErrorCode =
  | 'key_required'
  | 'source_key_required'
  | 'invalid_body_path'
  | 'value_required'
  | 'duplicate_key'
  | 'same_source_and_target'

const BODY_PATH_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$/

let passthroughFieldRuleDraftId = 0

export function createPassthroughFieldRuleDraft(): PassthroughFieldRuleDraft {
  passthroughFieldRuleDraftId += 1

  return {
    id: `passthrough-field-rule-${passthroughFieldRuleDraftId}`,
    target: 'header',
    mode: 'forward',
    key: '',
    source_key: '',
    value: ''
  }
}

export function normalizePassthroughFieldRule(
  rule: PassthroughFieldRuleDraft
): PassthroughFieldRuleDraft {
  const key = rule.key.trim()
  const sourceKey = (rule.source_key ?? '').trim()

  return {
    ...rule,
    key,
    source_key: rule.mode === 'map' ? sourceKey : '',
    value: rule.mode === 'inject' ? rule.value : ''
  }
}

export function validatePassthroughFieldRules(rules: PassthroughFieldRuleDraft[]): {
  ok: boolean
  errors: Record<number, PassthroughFieldRuleErrors>
} {
  const errors: Record<number, PassthroughFieldRuleErrors> = {}
  const seenKeys = new Map<string, number>()

  rules.forEach((rule, index) => {
    const normalizedRule = normalizePassthroughFieldRule(rule)
    const rowErrors: PassthroughFieldRuleErrors = {}

    if (!normalizedRule.key) {
      rowErrors.key = 'key_required'
    } else if (!isValidTargetKey(normalizedRule.target, normalizedRule.key)) {
      rowErrors.key = 'invalid_body_path'
    }

    if (normalizedRule.mode === 'inject' && !normalizedRule.value.trim()) {
      rowErrors.value = 'value_required'
    }

    if (normalizedRule.mode === 'map') {
      if (!normalizedRule.source_key) {
        rowErrors.source_key = 'source_key_required'
      } else if (!isValidTargetKey(normalizedRule.target, normalizedRule.source_key)) {
        rowErrors.source_key = 'invalid_body_path'
      } else if (isSameSourceAndTarget(normalizedRule)) {
        rowErrors.source_key = 'same_source_and_target'
      }
    }

    if (!rowErrors.key) {
      const comparableKey = getComparableKey(normalizedRule)
      const existingIndex = seenKeys.get(comparableKey)

      if (existingIndex !== undefined) {
        rowErrors.key = 'duplicate_key'
      } else {
        seenKeys.set(comparableKey, index)
      }
    }

    if (rowErrors.key || rowErrors.source_key || rowErrors.value) {
      errors[index] = rowErrors
    }
  })

  return {
    ok: Object.keys(errors).length === 0,
    errors
  }
}

function getComparableKey(rule: PassthroughFieldRuleDraft): string {
  const normalizedKey = rule.target === 'header' ? rule.key.toLowerCase() : rule.key
  return `${rule.target}:${normalizedKey}`
}

function isSameSourceAndTarget(rule: PassthroughFieldRuleDraft): boolean {
  if (rule.target === 'header') {
    return rule.source_key.toLowerCase() === rule.key.toLowerCase()
  }

  return rule.source_key === rule.key
}

function isValidTargetKey(target: PassthroughFieldTarget, key: string): boolean {
  if (target === 'header') {
    return true
  }

  return isValidBodyPath(key)
}

function isValidBodyPath(path: string): boolean {
  return BODY_PATH_PATTERN.test(path)
}
