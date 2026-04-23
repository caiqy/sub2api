import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useImageGeneration } from '@/composables/useImageGeneration'

const { edit, generate } = vi.hoisted(() => ({
  edit: vi.fn(),
  generate: vi.fn(),
}))

vi.mock('@/api', () => ({
  imagesAPI: {
    edit,
    generate,
  },
}))

describe('useImageGeneration', () => {
  beforeEach(() => {
    edit.mockReset()
    generate.mockReset()
  })

  it('prevents duplicate generate submissions while loading', async () => {
    let resolveGenerate: ((value: { created: number; data: Array<{ url: string }> }) => void) | undefined
    generate.mockImplementationOnce(
      () => new Promise((resolve) => { resolveGenerate = resolve })
    )
    generate.mockResolvedValueOnce({ created: 1, data: [{ url: 'https://cdn.example.com/duplicate.png' }] })

    const { isLoading, submitGenerate } = useImageGeneration()

    const firstSubmission = submitGenerate({ prompt: 'first request' }, 'sk-primary')
    expect(isLoading.value).toBe(true)

    const secondSubmission = await submitGenerate({ prompt: 'second request' }, 'sk-primary')

    expect(generate).toHaveBeenCalledTimes(1)
    expect(secondSubmission).toBeNull()

    resolveGenerate?.({ created: 1, data: [{ url: 'https://cdn.example.com/image.png' }] })
    await firstSubmission
    expect(isLoading.value).toBe(false)
  })

  it('uses form-data output_format to build the correct base64 preview mime', async () => {
    const formData = new FormData()
    formData.append('prompt', 'repair this image')
    formData.append('output_format', 'jpeg')
    formData.append('image', new File(['image-bytes'], 'source.png', { type: 'image/png' }))
    edit.mockResolvedValue({
      created: 1,
      data: [
        {
          b64_json: 'QUJD',
          revised_prompt: 'repair this image',
        },
      ],
    })

    const { results, submitEdit } = useImageGeneration()

    await submitEdit(formData, 'sk-edit')

    expect(results.value).toEqual([
      expect.objectContaining({
        src: 'data:image/jpeg;base64,QUJD',
      }),
    ])
  })
})
