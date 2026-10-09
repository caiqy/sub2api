import { sanitizeUrl } from './url'

const IMAGE_EXTENSIONS: Record<string, string> = {
  'image/png': 'png', 'image/jpeg': 'jpg', 'image/webp': 'webp',
}

export function imageDownloadFilename(src: string, index = 0): string {
  const mime = src.match(/^data:(image\/[a-z0-9.+-]+);/i)?.[1]?.toLowerCase()
  let extension = mime ? IMAGE_EXTENSIONS[mime] : undefined
  if (!extension) {
    try {
      extension = new URL(src).pathname.match(/\.(png|jpe?g|webp)$/i)?.[1]?.toLowerCase().replace('jpeg', 'jpg')
    } catch { /* The source is validated before use. */ }
  }
  return `sub2api-image-${index + 1}.${extension ?? 'png'}`
}

/** Decode inline images locally; never proxy image URLs or send platform credentials. */
export async function imageToFile(src: string, signal?: AbortSignal): Promise<File> {
  signal?.throwIfAborted()
  const safeSrc = sanitizeUrl(src, { allowDataUrl: true })
  if (!safeSrc) throw new Error('Invalid image URL.')
  let blob: Blob
  if (safeSrc.startsWith('data:')) {
    const header = safeSrc.match(/^data:(image\/(?:png|jpeg|webp));base64,/i)
    if (!header) throw new Error('Invalid Base64 image URL.')
    const binary = atob(safeSrc.slice(header[0].length))
    blob = new Blob([Uint8Array.from(binary, (character) => character.charCodeAt(0))], { type: header[1].toLowerCase() })
  } else {
    const response = await fetch(safeSrc, { signal, credentials: 'omit' })
    if (!response.ok) throw new Error(`Image could not be read (${response.status}).`)
    blob = await response.blob()
  }
  const mime = blob.type.split(';')[0].toLowerCase()
  const extension = IMAGE_EXTENSIONS[mime]
  if (!extension || blob.size === 0) throw new Error('The response is not a PNG, JPEG or WebP image.')
  return new File([blob], `sub2api-image.${extension}`, { type: mime })
}
