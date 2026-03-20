export type PassthroughFieldTarget = 'header' | 'body'
export type PassthroughFieldMode = 'forward' | 'inject'

export interface PassthroughFieldRuleDraft {
  id: string
  target: PassthroughFieldTarget
  mode: PassthroughFieldMode
  key: string
  value: string
}

export interface PassthroughFieldRuleErrors {
  key?: PassthroughFieldRuleErrorCode
  value?: PassthroughFieldRuleErrorCode
}

export type PassthroughFieldRuleErrorCode =
  | 'key_required'
  | 'invalid_body_path'
  | 'value_required'
  | 'duplicate_key'
  | 'reserved_key'

const RESERVED_HEADER_KEYS = new Set([
  'authorization',
  'cookie',
  'x-goog-api-key',
  'x-api-key',
  'api-key',
  'host',
  'content-length',
  'transfer-encoding',
  'connection'
])

const RESERVED_BODY_KEYS = new Set([
  'model',
  'stream'
])

const BODY_PATH_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$/

let passthroughFieldRuleDraftId = 0

export function createPassthroughFieldRuleDraft(): PassthroughFieldRuleDraft {
  passthroughFieldRuleDraftId += 1

  return {
    id: `passthrough-field-rule-${passthroughFieldRuleDraftId}`,
    target: 'header',
    mode: 'forward',
    key: '',
    value: ''
  }
}

export function normalizePassthroughFieldRule(
  rule: PassthroughFieldRuleDraft
): PassthroughFieldRuleDraft {
  return {
    ...rule,
    key: rule.key.trim()
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
    } else if (normalizedRule.target === 'body' && !isValidBodyPath(normalizedRule.key)) {
      rowErrors.key = 'invalid_body_path'
    } else if (isReservedPassthroughKey(normalizedRule)) {
      rowErrors.key = 'reserved_key'
    }

    if (normalizedRule.mode === 'inject' && !normalizedRule.value.trim()) {
      rowErrors.value = 'value_required'
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

    if (rowErrors.key || rowErrors.value) {
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

function isValidBodyPath(path: string): boolean {
  return BODY_PATH_PATTERN.test(path)
}

function isReservedPassthroughKey(rule: PassthroughFieldRuleDraft): boolean {
  if (rule.target === 'header') {
    return RESERVED_HEADER_KEYS.has(rule.key.toLowerCase())
  }

  return RESERVED_BODY_KEYS.has(rule.key)
}
