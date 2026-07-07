import { describe, expect, it } from 'vitest'
import { customMenuResourceID, isCustomMenuHidden } from '../userUiVisibility'

describe('userUiVisibility', () => {
  it('matches backend custom menu resource ids', () => {
    expect(customMenuResourceID('docs')).toBe(6649561995302739000)
    expect(customMenuResourceID('billing-help')).toBe(7082254467848891000)
  })

  it('checks hidden custom menus', () => {
    expect(isCustomMenuHidden('docs', ['docs'])).toBe(true)
    expect(isCustomMenuHidden('docs', ['billing-help'])).toBe(false)
  })
})
