import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { create, directPost, get, isCancel, post } = vi.hoisted(() => ({
  create: vi.fn(),
  directPost: vi.fn(),
  get: vi.fn(),
  isCancel: vi.fn(() => false),
  post: vi.fn(),
}))

vi.mock('axios', () => ({
  default: {
    create,
    isCancel,
    post,
  },
  create,
  isCancel,
  post,
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    defaults: {
      timeout: 30000,
      withCredentials: true,
    },
    get,
  },
}))

describe('images api', () => {
  beforeEach(() => {
    vi.resetModules()
    create.mockReset()
    directPost.mockReset()
    get.mockReset()
    isCancel.mockReset()
    post.mockReset()
    isCancel.mockReturnValue(false)
    get.mockResolvedValue({ data: {} })
    directPost.mockResolvedValue({ data: { created: 1, data: [] } })
    post.mockResolvedValue({ data: { created: 1, data: [] } })
    create.mockReturnValue({
      post: directPost,
    })
  })

  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('lists image history through the shared api client', async () => {
    const { imagesAPI } = await import('@/api/images')
    const signal = new AbortController().signal

    await imagesAPI.listHistory(
      {
        tab: 'edit',
        status: 'success',
        api_key_id: 9,
        page: 2,
        page_size: 10,
      },
      { signal }
    )

    expect(get).toHaveBeenCalledWith('/images/history', {
      params: {
        tab: 'edit',
        status: 'success',
        api_key_id: 9,
        page: 2,
        page_size: 10,
      },
      signal,
    })
  })

  it('loads image history detail through the shared api client', async () => {
    const { imagesAPI } = await import('@/api/images')

    await imagesAPI.getHistoryDetail(31)

    expect(get).toHaveBeenCalledWith('/images/history/31')
  })

  it('uses a dedicated 30 minute timeout for direct image gateway requests', async () => {
    await import('@/api/images')

    expect(create).toHaveBeenCalledWith({
      timeout: 1800000,
      withCredentials: true,
    })
  })

  it('targets absolute gateway urls for image generation and sends the selected api key', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://gateway.example.com/api/v1')
    const { imagesAPI } = await import('@/api/images')
    const signal = new AbortController().signal
    const payload = {
      model: 'gpt-image-2',
      prompt: 'draw a neon fox',
      size: '1024x1024',
    }

    await imagesAPI.generate(payload, 'sk-selected-key', { signal })

    expect(directPost).toHaveBeenCalledWith('https://gateway.example.com/v1/images/generations', payload, {
      headers: {
        Authorization: 'Bearer sk-selected-key',
      },
      signal,
    })
  })

  it('normalizes absolute gateway urls with a trailing slash', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://gateway.example.com/api/v1/')
    const { imagesAPI } = await import('@/api/images')

    await imagesAPI.generate({ prompt: 'draw a lantern' }, 'sk-absolute')

    expect(directPost).toHaveBeenCalledWith(
      'https://gateway.example.com/v1/images/generations',
      { prompt: 'draw a lantern' },
      expect.objectContaining({
        headers: {
          Authorization: 'Bearer sk-absolute',
        },
      })
    )
  })

  it('keeps relative gateway urls relative for image generation', async () => {
    vi.stubEnv('VITE_API_BASE_URL', '/api/v1')
    const { imagesAPI } = await import('@/api/images')

    await imagesAPI.generate({ prompt: 'draw a skyline' }, 'sk-relative')

    expect(directPost).toHaveBeenCalledWith(
      '/v1/images/generations',
      { prompt: 'draw a skyline' },
      expect.objectContaining({
        headers: {
          Authorization: 'Bearer sk-relative',
        },
      })
    )
  })

  it('returns gateway url items when response_format=url is requested', async () => {
    vi.stubEnv('VITE_API_BASE_URL', '/api/v1')
    directPost.mockResolvedValue({
      data: {
        created: 1,
        data: [
          {
            url: 'https://cdn.example.com/image.png',
            revised_prompt: 'draw a skyline',
          },
        ],
      },
    })
    const { imagesAPI } = await import('@/api/images')

    const result = await imagesAPI.generate(
      { prompt: 'draw a skyline', response_format: 'url' },
      'sk-url'
    )

    expect(result.data).toEqual([
      {
        url: 'https://cdn.example.com/image.png',
        revised_prompt: 'draw a skyline',
      },
    ])
  })

  it('uses FormData for edits without forcing multipart content-type', async () => {
    vi.stubEnv('VITE_API_BASE_URL', '/api/v1')
    const { imagesAPI } = await import('@/api/images')
    const formData = new FormData()
    formData.append('prompt', 'repair this image')
    formData.append('image', new Blob(['png-bytes'], { type: 'image/png' }), 'source.png')

    await imagesAPI.edit(formData, 'sk-edit-key')

    expect(directPost).toHaveBeenCalledWith('/v1/images/edits', formData, {
      headers: {
        Authorization: 'Bearer sk-edit-key',
      },
    })

    expect(directPost.mock.calls[0][2]?.headers).not.toHaveProperty('Content-Type')
  })

  it('propagates direct image request failures using the shared api error shape', async () => {
    vi.stubEnv('VITE_API_BASE_URL', '/api/v1')
    directPost.mockRejectedValue({
      response: {
        status: 502,
        data: {
          code: 'UPSTREAM_ERROR',
          message: 'Gateway failed',
          reason: 'bad upstream',
        },
      },
      config: {
        url: '/v1/images/generations',
      },
      message: 'Request failed',
    })
    const { imagesAPI } = await import('@/api/images')

    await expect(imagesAPI.generate({ prompt: 'draw a fox' }, 'sk-fail')).rejects.toEqual(
      expect.objectContaining({
        status: 502,
        code: 'UPSTREAM_ERROR',
        message: 'Gateway failed',
        reason: 'bad upstream',
      })
    )
  })
})
