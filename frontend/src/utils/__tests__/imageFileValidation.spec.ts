import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { pngHasAlpha, sniffImageMime, validateMaskImage, validateSourceImage } from '../imageFileValidation'
function png(color: number, transparency?: number[], palette?: number[]) {
  const chunk = (name: string, body: number[]) => {
    const length = body.length
    return [length >>> 24, length >>> 16 & 255, length >>> 8 & 255, length & 255, ...Array.from(name, char => char.charCodeAt(0)), ...body, 0, 0, 0, 0]
  }
  return new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10, ...chunk('IHDR', [0, 0, 0, 1, 0, 0, 0, 1, 8, color, 0, 0, 0]), ...(palette ? chunk('PLTE', palette) : []), ...(transparency ? chunk('tRNS', transparency) : []), ...chunk('IDAT', [0]), ...chunk('IEND', [])])
}
const revoke = vi.fn()
beforeEach(() => {
  vi.stubGlobal('URL', { createObjectURL: vi.fn(() => 'blob:decode'), revokeObjectURL: revoke })
  revoke.mockReset()
  class DecodedImage {
    naturalWidth = 1024; naturalHeight = 1024
    onload: (() => void) | null = null; onerror: (() => void) | null = null
    set src(value: string) { if (value) queueMicrotask(() => this.onload?.()) }
  }
  vi.stubGlobal('Image', DecodedImage)
})
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals() })
describe('real image file boundaries', () => {
  it('sniffs PNG/JPEG/WebP and rejects declared MIME spoofing', async () => {
    expect(sniffImageMime(png(6))).toBe('image/png')
    expect(sniffImageMime(new Uint8Array([255, 216, 255, 0]))).toBe('image/jpeg')
    expect(sniffImageMime(new TextEncoder().encode('RIFFxxxxWEBP'))).toBe('image/webp')
    await expect(validateSourceImage(new File(['not png'], 'fake.png', { type: 'image/png' }))).rejects.toThrow('sourceImageInvalid')
    await expect(validateSourceImage(new File([png(6)], 'fake.jpg', { type: 'image/jpeg' }))).rejects.toThrow('sourceImageInvalid')
  })
  it('enforces file caps before reading and decodes dimensions', async () => {
    const oversized = new File([], 'large.png', { type: 'image/png' }); Object.defineProperty(oversized, 'size', { value: 20 * 1024 * 1024 + 1 })
    await expect(validateSourceImage(oversized)).rejects.toThrow('sourceImageTooLarge')
    await expect(validateSourceImage(new File([png(6)], 'good.png', { type: 'image/png' }))).resolves.toEqual({ width: 1024, height: 1024 })
    expect(revoke).toHaveBeenCalledWith('blob:decode')
  })
  it('recognizes alpha and palette tRNS but rejects malformed transparency', () => {
    expect(pngHasAlpha(png(6))).toBe(true)
    expect(pngHasAlpha(png(2))).toBe(false)
    expect(pngHasAlpha(png(3, [0, 255], [255, 0, 0, 0, 255, 0]))).toBe(true)
    expect(pngHasAlpha(png(3, [0, 255]))).toBe(false)
    expect(pngHasAlpha(png(3, [0, 255], [255, 0, 0]))).toBe(false)
    expect(pngHasAlpha(png(2, [0]))).toBe(false)
    expect(pngHasAlpha(png(6).slice(0, -1))).toBe(false)
  })
  it('requires PNG alpha, strict mask cap and first-image dimensions', async () => {
    const mask = new File([png(6)], 'mask.png', { type: 'image/png' })
    await expect(validateMaskImage(mask, { width: 512, height: 512 })).rejects.toThrow('maskDimensions')
    await expect(validateMaskImage(new File([png(2)], 'opaque.png', { type: 'image/png' }), { width: 1024, height: 1024 })).rejects.toThrow('maskAlphaRequired')
    Object.defineProperty(mask, 'size', { value: 4_000_000 })
    await expect(validateMaskImage(mask)).rejects.toThrow('maskTooLarge')
  })
  it('rejects decode errors and cleans temporary URLs', async () => {
    class BrokenImage { onerror: (() => void) | null = null; onload: (() => void) | null = null; set src(value: string) { if (value) queueMicrotask(() => this.onerror?.()) } }
    vi.stubGlobal('Image', BrokenImage)
    await expect(validateSourceImage(new File([png(6)], 'bad.png', { type: 'image/png' }))).rejects.toThrow('sourceImageDecode')
    expect(revoke).toHaveBeenCalledWith('blob:decode')
  })
})
