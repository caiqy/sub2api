export interface ImageDimensions { width: number; height: number }
const PREFIX = 'images.forms.edit.'

export function readImageBytes(file: File): Promise<Uint8Array> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(new Error(PREFIX + 'sourceImageDecode'))
    reader.onload = () => resolve(new Uint8Array(reader.result as ArrayBuffer))
    reader.readAsArrayBuffer(file)
  })
}

export function sniffImageMime(bytes: Uint8Array): string | null {
  if ([137, 80, 78, 71, 13, 10, 26, 10].every((byte, index) => bytes[index] === byte)) return 'image/png'
  if (bytes[0] === 255 && bytes[1] === 216 && bytes[2] === 255) return 'image/jpeg'
  if (String.fromCharCode(...bytes.slice(0, 4)) === 'RIFF' && String.fromCharCode(...bytes.slice(8, 12)) === 'WEBP') return 'image/webp'
  return null
}

// Palette transparency is an alpha channel too, but only a structurally valid tRNS counts.
export function pngHasAlpha(bytes: Uint8Array): boolean {
  if (sniffImageMime(bytes) !== 'image/png' || bytes.length < 33) return false
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
  let colorType = -1
  let paletteEntries = 0
  let alpha = false
  let imageData = false
  let transparency = false
  for (let offset = 8; offset + 12 <= bytes.length;) {
    const length = view.getUint32(offset)
    if (length > bytes.length - offset - 12) return false
    const type = String.fromCharCode(...bytes.slice(offset + 4, offset + 8))
    const start = offset + 8
    if (offset === 8 && (type !== 'IHDR' || length !== 13)) return false
    if (type === 'IHDR') {
      if (offset !== 8 || length !== 13) return false
      colorType = bytes[start + 9]
      alpha = colorType === 4 || colorType === 6
    } else if (type === 'PLTE') {
      if (imageData || !length || length % 3 || length > 768) return false
      paletteEntries = length / 3
    } else if (type === 'tRNS') {
      if (imageData || transparency) return false
      transparency = true
      if (!((colorType === 0 && length === 2) || (colorType === 2 && length === 6) || (colorType === 3 && length > 0 && length <= paletteEntries))) return false
      alpha = true
    } else if (type === 'IDAT') imageData = true
    else if (type === 'IEND') return length === 0 && imageData && alpha && offset + 12 === bytes.length
    offset += length + 12
  }
  return false
}

function decodeImage(file: File, signal?: AbortSignal): Promise<ImageDimensions> {
  const url = URL.createObjectURL(file)
  const image = new Image()
  let abort: () => void = () => {}
  let timeout: ReturnType<typeof setTimeout>
  return new Promise<ImageDimensions>((resolve, reject) => {
    abort = () => reject(new DOMException('Aborted', 'AbortError'))
    signal?.addEventListener('abort', abort, { once: true })
    timeout = setTimeout(() => reject(new Error(PREFIX + 'sourceImageDecode')), 15000)
    image.onload = () => image.naturalWidth && image.naturalHeight
      ? resolve({ width: image.naturalWidth, height: image.naturalHeight })
      : reject(new Error(PREFIX + 'sourceImageDecode'))
    image.onerror = () => reject(new Error(PREFIX + 'sourceImageDecode'))
    if (signal?.aborted) abort()
    else image.src = url
  }).finally(() => {
    clearTimeout(timeout)
    signal?.removeEventListener('abort', abort)
    image.onload = null
    image.onerror = null
    image.src = ''
    URL.revokeObjectURL(url)
  })
}

export async function validateSourceImage(file: File, signal?: AbortSignal): Promise<ImageDimensions> {
  if (file.size > 20 * 1024 * 1024) throw new Error(PREFIX + 'sourceImageTooLarge')
  const bytes = await readImageBytes(file)
  const mime = sniffImageMime(bytes)
  if (!mime || file.type.toLowerCase() !== mime) throw new Error(PREFIX + 'sourceImageInvalid')
  return decodeImage(file, signal)
}

export async function validateMaskImage(file: File, first?: ImageDimensions, signal?: AbortSignal): Promise<ImageDimensions> {
  if (file.size >= 4_000_000) throw new Error(PREFIX + 'maskTooLarge')
  const bytes = await readImageBytes(file)
  if (file.type.toLowerCase() !== 'image/png' || sniffImageMime(bytes) !== 'image/png') throw new Error(PREFIX + 'maskPngRequired')
  if (!pngHasAlpha(bytes)) throw new Error(PREFIX + 'maskAlphaRequired')
  const dimensions = await decodeImage(file, signal)
  if (!first || dimensions.width !== first.width || dimensions.height !== first.height) throw new Error(PREFIX + 'maskDimensions')
  return dimensions
}
