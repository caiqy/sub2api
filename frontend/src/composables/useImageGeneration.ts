import { getCurrentScope, onScopeDispose, ref } from 'vue'
import { imagesAPI } from '@/api'
import type { ImageFetchOptions } from '@/api/images'
import type { ImageGatewayDataItem, ImageGatewayResponse, ImageGenerationRequest } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import { sanitizeUrl } from '@/utils/url'

export interface ImageResultPreview {
  src: string
  revisedPrompt?: string
  source: 'data-url' | 'url'
  mimeType?: string
}

const MIME: Record<string, string> = { png: 'image/png', jpeg: 'image/jpeg', jpg: 'image/jpeg', webp: 'image/webp' }
export function normalizeGatewayResult(item: ImageGatewayDataItem, requestedFormat?: string): ImageResultPreview {
  if (!item || typeof item !== 'object') throw new Error('Invalid image result')
  const metadata = item as Record<string, unknown>
  if (item.b64_json !== undefined && (typeof item.b64_json !== 'string' || !item.b64_json.trim())) throw new Error('Invalid image result')
  let mimeType: string | undefined
  const magicFormat = item.b64_json?.startsWith('iVBORw0KGgo') ? 'png' : item.b64_json?.startsWith('/9j/') ? 'jpeg' : item.b64_json?.startsWith('UklGR') ? 'webp' : undefined
  for (const value of [metadata.mime_type, metadata.mimeType, metadata.content_type, metadata.contentType, item.output_format, metadata.outputFormat, metadata.format, magicFormat, requestedFormat]) {
    if (typeof value !== 'string') continue
    const normalized = value.toLowerCase().trim()
    mimeType = Object.values(MIME).includes(normalized) ? normalized : MIME[normalized]
    if (mimeType) break
  }
  if (item.b64_json) {
    if (!mimeType) throw new Error('Unknown image output format')
    return { src: `data:${mimeType};base64,${item.b64_json}`, source: 'data-url', revisedPrompt: item.revised_prompt, mimeType }
  }
  const src = typeof item.url === 'string' ? sanitizeUrl(item.url) : ''
  if (!src) throw new Error('Invalid image result URL')
  return { src, source: 'url', revisedPrompt: item.revised_prompt, mimeType }
}

export function useImageGeneration() {
  const isLoading = ref(false)
  const loadingSeconds = ref(0)
  const durationMs = ref<number | null>(null)
  const status = ref<'idle' | 'generating' | 'success' | 'error' | 'stopped'>('idle')
  const error = ref('')
  const results = ref<ImageResultPreview[]>([])
  const partialResult = ref<ImageResultPreview | null>(null)
  const lastResponse = ref<ImageGatewayResponse | null>(null)
  let timer: ReturnType<typeof setInterval> | undefined
  let controller: AbortController | undefined
  let generationId = 0
  let started = 0
  const finish = () => {
    clearInterval(timer)
    timer = undefined
    durationMs.value = Math.max(0, Date.now() - started)
    loadingSeconds.value = Math.floor(durationMs.value / 1000)
    isLoading.value = false
  }
  function stop() {
    if (!isLoading.value) return
    generationId++
    controller?.abort()
    controller = undefined
    status.value = 'stopped'
    finish()
  }
  if (getCurrentScope()) onScopeDispose(stop)

  async function submit(payload: ImageGenerationRequest | FormData, key: string) {
    if (isLoading.value || !key.trim()) return null
    const id = ++generationId
    controller = new AbortController()
    isLoading.value = true
    status.value = 'generating'
    started = Date.now()
    loadingSeconds.value = 0
    durationMs.value = null
    error.value = ''
    partialResult.value = null
    results.value = []
    lastResponse.value = null
    timer = setInterval(() => { loadingSeconds.value = Math.floor((Date.now() - started) / 1000) }, 1000)
    const format = String(payload instanceof FormData ? payload.get('output_format') || 'png' : payload.output_format || 'png')
    const current = () => generationId === id && !controller?.signal.aborted
    const addFinal = (item: ImageGatewayDataItem, outputFormat = format) => {
      if (!current()) return
      const preview = normalizeGatewayResult(item, outputFormat)
      if (!results.value.some(result => result.src === preview.src)) results.value.push(preview)
    }
    const options: ImageFetchOptions = {
      signal: controller.signal,
      onPartial: item => { if (current()) partialResult.value = normalizeGatewayResult(item, format) },
      onCompleted: item => addFinal(item),
    }
    try {
      let response: ImageGatewayResponse
      if (payload instanceof FormData) {
        const request = new FormData()
        payload.forEach((value, name) => request.append(name, value))
        request.set('stream', 'true')
        request.set('partial_images', '1')
        response = await imagesAPI.edit(request, key, options)
      } else {
        response = await imagesAPI.generate({ ...payload, stream: true, partial_images: 1 }, key, options)
      }
      if (!current()) return null
      response.data.forEach(item => addFinal(item, response.output_format || format))
      if (!results.value.length) throw new Error('No final image received')
      lastResponse.value = response
      partialResult.value = null
      status.value = 'success'
      return response
    } catch (err) {
      if (!current()) return null
      status.value = 'error'
      error.value = extractApiErrorMessage(err, 'Request failed')
      return null
    } finally {
      if (generationId === id) { finish(); controller = undefined }
    }
  }
  function clearError() { error.value = '' }
  function clearResults() { results.value = []; partialResult.value = null; lastResponse.value = null }
  return {
    isLoading, loadingSeconds, durationMs, status, error, results, partialResult, lastResponse, stop,
    submitGenerate: (payload: ImageGenerationRequest, key: string) => submit(payload, key),
    submitEdit: (payload: FormData, key: string) => submit(payload, key),
    clearError, clearResults,
  }
}
