import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'

import ImageGenerateForm from '@/components/user/images/ImageGenerateForm.vue'
import ImageHistoryDetail from '@/components/user/images/ImageHistoryDetail.vue'
import type { ImageHistoryDetail as HistoryDetail } from '@/types'

import en from '../locales/en'
import zh from '../locales/zh'

describe('images locale keys', () => {
  it('exposes zh images labels at the top level', () => {
    expect(zh.images?.title).toBe('AI 生图')
    expect(zh.images?.tabs?.generate).toBe('生成')
    expect(zh.admin?.images).toBeUndefined()
  })

  it('exposes en images labels at the top level', () => {
    expect(en.images?.title).toBe('AI Images')
    expect(en.images?.tabs?.generate).toBe('Generate')
    expect(en.admin?.images).toBeUndefined()
  })

  it('resolves literal image keys including indirect notice and error keys', () => {
    const sources = import.meta.glob<string>('../../components/user/images/*.vue', { eager: true, query: '?raw', import: 'default' })
    const keys = new Set(Object.values(sources).flatMap(source => [...source.matchAll(/['"](images\.[\w.]+)['"]/g)].map(match => match[1])))
    for (const dictionary of [en, zh]) {
      for (const key of keys) {
        const value = key.split('.').reduce<unknown>((node, part) => node && typeof node === 'object' ? (node as Record<string, unknown>)[part] : undefined, dictionary)
        expect(typeof value, key).toBe('string')
      }
    }
  })

  it('shows the automatic size guidance and translated transparent JPEG correction', async () => {
    const wrapper = mount(ImageGenerateForm, { props: { models: ['gpt-image-2.5-flare'] }, global: { plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh }, messageCompiler: message => () => typeof message === 'string' ? message : '' })] } })
    expect(wrapper.text()).toContain(zh.images.forms.generate.sizeHint)
    await wrapper.get('#image-generate-background').setValue('transparent')
    await wrapper.get('#image-generate-output-format').setValue('jpeg')
    expect((wrapper.get('#image-generate-output-format').element as HTMLSelectElement).value).toBe('png')
    expect(wrapper.text()).toContain(zh.images.forms.generate.transparentFormatAdjusted)
    wrapper.unmount()
  })

  it('explains missing original prompts and unavailable final images in history', () => {
    const detail: HistoryDetail = {
      id: 1, api_key_id: 1, mode: 'generate', status: 'success', model: 'gpt-image-2.5-flare', prompt: '', n: 1,
      had_source_image: false, had_mask: false, images: [], created_at: '2026-10-09T00:00:00Z',
      replay: { mode: 'generate', model: 'gpt-image-2.5-flare', n: 1, requires_source_image_upload: false, requires_mask_upload: false }
    }
    const wrapper = mount(ImageHistoryDetail, { props: { detail, error: '', loading: false }, global: { plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh }, messageCompiler: message => () => typeof message === 'string' ? message : '' })] } })
    expect(wrapper.text()).toContain(zh.images.history.noPrompt)
    expect(wrapper.text()).toContain(zh.images.history.noImages)
    expect(wrapper.text()).not.toContain(zh.images.results.empty)
    wrapper.unmount()
  })
})
