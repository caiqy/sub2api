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

  it('decodes Base64 images without fetching under a restrictive CSP', async () => {
    vi.mocked(fetch).mockRejectedValue(new TypeError('Blocked by CSP'))
    for (const [mime, extension] of [['image/png', 'png'], ['image/jpeg', 'jpg'], ['image/webp', 'webp']]) {
      const file = await imageToFile(`data:${mime};base64,AAH/`)
      expect(file.type).toBe(mime)
      expect(file.name).toBe(`sub2api-image.${extension}`)
      expect(file.size).toBe(3)
      const bytes = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader()
        reader.onload = () => resolve(reader.result as string)
        reader.onerror = () => reject(reader.error)
        reader.readAsDataURL(file)
      })
      expect(bytes).toBe(`data:${mime};base64,AAH/`)
    }
    expect(fetch).not.toHaveBeenCalled()
  })

  it('rejects malformed, unsupported or empty data images without fetching', async () => {
    for (const src of ['data:image/png;base64,%%%', 'data:image/png;base64,', 'data:image/gif;base64,AAH/', 'data:image/png,abc']) {
      await expect(imageToFile(src)).rejects.toThrow()
    }
    expect(fetch).not.toHaveBeenCalled()
  })

  it('honors cancellation before decoding a data image', async () => {
    const controller = new AbortController()
    controller.abort()
    await expect(imageToFile('data:image/png;base64,AAH/', controller.signal)).rejects.toMatchObject({ name: 'AbortError' })
    expect(fetch).not.toHaveBeenCalled()
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
