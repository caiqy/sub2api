import { describe, expect, it } from 'vitest'

import { durationSeverity, firstTokenSeverity, requestBodySizeSeverity } from '../latencyHealth'

describe('latencyHealth', () => {
  it('classifies first-token latency at 10s/30s/60s boundaries', () => {
    expect(firstTokenSeverity(0)).toBe('good')
    expect(firstTokenSeverity(9_999)).toBe('good')
    expect(firstTokenSeverity(10_000)).toBe('warn')
    expect(firstTokenSeverity(29_999)).toBe('warn')
    expect(firstTokenSeverity(30_000)).toBe('slow')
    expect(firstTokenSeverity(59_999)).toBe('slow')
    expect(firstTokenSeverity(60_000)).toBe('critical')
  })

  it('classifies total duration at 1min/3min/5min boundaries', () => {
    expect(durationSeverity(0)).toBe('good')
    expect(durationSeverity(59_999)).toBe('good')
    expect(durationSeverity(60_000)).toBe('warn')
    expect(durationSeverity(179_999)).toBe('warn')
    expect(durationSeverity(180_000)).toBe('slow')
    expect(durationSeverity(299_999)).toBe('slow')
    expect(durationSeverity(300_000)).toBe('critical')
  })

  it('classifies request body size with 1MiB green boundary', () => {
    const mib = 1024 * 1024

    expect(requestBodySizeSeverity(null)).toBeNull()
    expect(requestBodySizeSeverity(undefined)).toBeNull()
    expect(requestBodySizeSeverity(-1)).toBeNull()
    expect(requestBodySizeSeverity(0)).toBe('good')
    expect(requestBodySizeSeverity(mib)).toBe('good')
    expect(requestBodySizeSeverity(mib + 1)).toBe('warn')
    expect(requestBodySizeSeverity(2 * mib)).toBe('warn')
    expect(requestBodySizeSeverity(2 * mib + 1)).toBe('slow')
    expect(requestBodySizeSeverity(4 * mib)).toBe('slow')
    expect(requestBodySizeSeverity(4 * mib + 1)).toBe('critical')
  })
})
