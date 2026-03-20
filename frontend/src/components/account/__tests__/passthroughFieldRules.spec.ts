import { describe, expect, it } from 'vitest'

import {
  normalizePassthroughFieldRule,
  validatePassthroughFieldRules
} from '../passthroughFieldRules'

describe('passthroughFieldRules', () => {
  it('normalizes key whitespace but preserves original value', () => {
    const result = normalizePassthroughFieldRule({
      id: 'rule-1',
      target: 'body',
      mode: 'inject',
      key: '  metadata.user_id  ',
      value: '  123  '
    })

    expect(result).toEqual({
      id: 'rule-1',
      target: 'body',
      mode: 'inject',
      key: 'metadata.user_id',
      value: '  123  '
    })
  })

  it('treats header keys as case-insensitive duplicates', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'forward', key: 'X-Test', value: '' },
      { id: 'rule-2', target: 'header', mode: 'inject', key: 'x-test', value: '1' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[1]?.key).toBe('duplicate_key')
  })

  it('rejects body paths with numeric path segments', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'body', mode: 'forward', key: 'messages.0.role', value: '' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      key: 'invalid_body_path'
    })
  })

  it('rejects body paths with non-identifier segments', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'body', mode: 'forward', key: 'metadata.user-id', value: '' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      key: 'invalid_body_path'
    })
  })

  it('requires non-blank value for inject mode', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'body', mode: 'inject', key: 'metadata.user_id', value: '   ' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      value: 'value_required'
    })
  })

  it('uses stable error code for missing key', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'forward', key: '   ', value: '' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      key: 'key_required'
    })
  })

  it('rejects reserved header keys before submit', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'forward', key: ' Authorization ', value: '' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      key: 'reserved_key'
    })
  })

  it('rejects gemini upstream auth header before submit', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'inject', key: ' X-Goog-Api-Key ', value: 'evil-key' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      key: 'reserved_key'
    })
  })

  it('rejects cookie header before submit', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'forward', key: ' Cookie ', value: '' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      key: 'reserved_key'
    })
  })

  it('rejects reserved body keys before submit', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'body', mode: 'inject', key: 'model', value: 'gpt-4.1' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      key: 'reserved_key'
    })
  })

  it('accepts valid passthrough rules without errors', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'forward', key: ' X-Test ', value: '' },
      { id: 'rule-2', target: 'body', mode: 'inject', key: 'metadata.user_id', value: ' 123 ' }
    ])

    expect(result.ok).toBe(true)
    expect(result.errors).toEqual({})
  })
})
