import { describe, expect, it } from 'vitest'

import { getDefaultBaseUrl } from '../passthroughFieldSupport'

describe('getDefaultBaseUrl', () => {
  it('returns the official xAI API URL for Grok', () => {
    expect(getDefaultBaseUrl('grok')).toBe('https://api.x.ai/v1')
  })
})
