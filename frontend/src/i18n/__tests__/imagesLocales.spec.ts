import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('images locale keys', () => {
  it('exposes zh images labels at the top level', () => {
    expect(zh.images?.title).toBe('AI生图')
    expect(zh.images?.tabs?.generate).toBe('生成')
    expect(zh.admin?.images).toBeUndefined()
  })

  it('exposes en images labels at the top level', () => {
    expect(en.images?.title).toBe('AI Images')
    expect(en.images?.tabs?.generate).toBe('Generate')
    expect(en.admin?.images).toBeUndefined()
  })
})
