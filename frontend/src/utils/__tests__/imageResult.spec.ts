import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { imageDownloadFilename, imageToFile } from '../imageResult'

describe('image result reading', () => {
  beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
  afterEach(() => vi.unstubAllGlobals())

  it('names PNG, JPEG and WebP from MIME or a signed URL path', () => {
    expect(imageDownloadFilename('data:image/jpeg;base64,abc')).toBe('sub2api-image-1.jpg')
    expect(imageDownloadFilename('data:image/webp;base64,abc', 2)).toBe('sub2api-image-3.webp')
    expect(imageDownloadFilename('https://example.com/a.PNG?signature=x')).toBe('sub2api-image-1.png')
  })

  it('reads the actual response MIME without platform credentials', async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: true, blob: async () => new Blob(['image'], { type: 'image/webp' }) } as Response)
    const controller = new AbortController()
    const file = await imageToFile('https://example.com/no-extension?signature=x', controller.signal)
    expect(fetch).toHaveBeenCalledWith('https://example.com/no-extension?signature=x', { signal: controller.signal, credentials: 'omit' })
    expect(file.type).toBe('image/webp')
    expect(file.name).toBe('sub2api-image.webp')
    expect(file.size).toBe(5)
  })

  it('rejects unsafe URLs before fetching and rejects expired or non-image responses', async () => {
    await expect(imageToFile('javascript:alert(1)')).rejects.toThrow('Invalid image URL')
    expect(fetch).not.toHaveBeenCalled()
    vi.mocked(fetch).mockResolvedValueOnce({ ok: false, status: 403 } as Response)
    await expect(imageToFile('https://example.com/expired')).rejects.toThrow('403')
    vi.mocked(fetch).mockResolvedValueOnce({ ok: true, blob: async () => new Blob(['html'], { type: 'text/html' }) } as Response)
    await expect(imageToFile('https://example.com/login')).rejects.toThrow('not a PNG')
  })
})
