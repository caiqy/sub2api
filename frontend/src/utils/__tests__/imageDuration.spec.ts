import { describe, expect, it } from 'vitest'

import { formatImageDuration } from '../imageDuration'

describe('formatImageDuration', () => {
  it('returns an empty string for missing or invalid durations', () => {
    expect(formatImageDuration(undefined)).toBe('')
    expect(formatImageDuration(null)).toBe('')
    expect(formatImageDuration(-1)).toBe('')
    expect(formatImageDuration(Number.NaN)).toBe('')
  })

  it('formats sub-second durations in milliseconds', () => {
    expect(formatImageDuration(0)).toBe('0ms')
    expect(formatImageDuration(850)).toBe('850ms')
  })

  it('formats second durations with one decimal place', () => {
    expect(formatImageDuration(1000)).toBe('1.0s')
    expect(formatImageDuration(2140)).toBe('2.1s')
    expect(formatImageDuration(63420)).toBe('63.4s')
  })
})
