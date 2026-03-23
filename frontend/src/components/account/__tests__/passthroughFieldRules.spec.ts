import { describe, expect, it } from 'vitest'

import {
  createPassthroughFieldRuleDraft,
  normalizePassthroughFieldRule,
  validatePassthroughFieldRules
} from '../passthroughFieldRules'

describe('passthroughFieldRules', () => {
  it('normalizes key and source_key whitespace for map rules and clears value', () => {
    const result = normalizePassthroughFieldRule({
      id: 'rule-1',
      target: 'body',
      mode: 'map',
      key: '  metadata.user_id  ',
      source_key: '  input.user_id  ',
      value: '  123  '
    })

    expect(result).toEqual({
      id: 'rule-1',
      target: 'body',
      mode: 'map',
      key: 'metadata.user_id',
      source_key: 'input.user_id',
      value: ''
    })
  })

  it('clears value when mode switches away from inject', () => {
    const result = normalizePassthroughFieldRule({
      id: 'rule-1',
      target: 'header',
      mode: 'forward',
      key: ' X-Test ',
      source_key: ' source-header ',
      value: ' 123 '
    })

    expect(result).toEqual({
      id: 'rule-1',
      target: 'header',
      mode: 'forward',
      key: 'X-Test',
      source_key: '',
      value: ''
    })
  })

  it('clears source_key when mode is inject', () => {
    const result = normalizePassthroughFieldRule({
      id: 'rule-1',
      target: 'body',
      mode: 'inject',
      key: ' metadata.user_id ',
      source_key: ' source.user_id ',
      value: ' 123 '
    })

    expect(result).toEqual({
      id: 'rule-1',
      target: 'body',
      mode: 'inject',
      key: 'metadata.user_id',
      source_key: '',
      value: ' 123 '
    })
  })

  it('treats header keys as case-insensitive duplicates', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'forward', key: 'X-Test', source_key: '', value: '' },
      { id: 'rule-2', target: 'header', mode: 'inject', key: 'x-test', source_key: '', value: '1' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[1]?.key).toBe('duplicate_key')
  })

  it('rejects body paths with numeric path segments', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'body', mode: 'forward', key: 'messages.0.role', source_key: '', value: '' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      key: 'invalid_body_path'
    })
  })

  it('rejects body paths with non-identifier segments', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'body', mode: 'forward', key: 'metadata.user-id', source_key: '', value: '' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      key: 'invalid_body_path'
    })
  })

  it('requires non-blank value for inject mode', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'body', mode: 'inject', key: 'metadata.user_id', source_key: '', value: '   ' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      value: 'value_required'
    })
  })

  it('uses stable error code for missing key', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'forward', key: '   ', source_key: '', value: '' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      key: 'key_required'
    })
  })

  it('allows header keys that used to be reserved', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'forward', key: ' Authorization ', source_key: '', value: '' }
    ])

    expect(result.ok).toBe(true)
    expect(result.errors).toEqual({})
  })

  it('allows body keys that used to be reserved', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'body', mode: 'inject', key: 'model', source_key: '', value: 'gpt-4.1' }
    ])

    expect(result.ok).toBe(true)
    expect(result.errors).toEqual({})
  })

  it('requires source_key for map mode', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'map', key: 'X-Target', source_key: '   ', value: 'ignored' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      source_key: 'source_key_required'
    })
  })

  it('rejects semantically identical source and target header keys in map mode', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'map', key: 'X-Trace-Id', source_key: 'x-trace-id', value: '' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      source_key: 'same_source_and_target'
    })
  })

  it('rejects invalid body source_key in map mode', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'body', mode: 'map', key: 'metadata.user_id', source_key: 'messages.0.role', value: '' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[0]).toEqual({
      source_key: 'invalid_body_path'
    })
  })

  it('treats conflicting header target keys as case-insensitive duplicates even in map mode', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'map', key: 'X-Test', source_key: 'X-Source-One', value: '' },
      { id: 'rule-2', target: 'header', mode: 'map', key: 'x-test', source_key: 'X-Source-Two', value: '' }
    ])

    expect(result.ok).toBe(false)
    expect(result.errors[1]?.key).toBe('duplicate_key')
  })

  it('creates new rules with header target and forward mode defaults', () => {
    const result = createPassthroughFieldRuleDraft()

    expect(result.target).toBe('header')
    expect(result.mode).toBe('forward')
    expect(result.source_key).toBe('')
  })

  it('accepts valid passthrough rules without errors', () => {
    const result = validatePassthroughFieldRules([
      { id: 'rule-1', target: 'header', mode: 'forward', key: ' X-Test ', source_key: '', value: '' },
      { id: 'rule-2', target: 'body', mode: 'inject', key: 'metadata.user_id', source_key: '', value: ' 123 ' },
      { id: 'rule-3', target: 'body', mode: 'map', key: 'metadata.mapped_user_id', source_key: 'payload.user_id', value: 'ignored' }
    ])

    expect(result.ok).toBe(true)
    expect(result.errors).toEqual({})
  })
})
