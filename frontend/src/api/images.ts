import { apiClient } from './client'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { ApiResponse, FetchOptions, ImageGatewayDataItem, ImageGatewayResponse, ImageGenerationRequest, ImageHistoryDetail, ImageHistoryListItem, ImageHistoryListParams, PaginatedResponse } from '@/types'

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '/api/v1'
const IMAGE_GATEWAY_TIMEOUT_MS = 1800000

export interface ImageFetchOptions extends FetchOptions {
  onPartial?: (item: ImageGatewayDataItem) => void
  onCompleted?: (item: ImageGatewayDataItem) => void
}

function buildGatewayURL(endpoint: string): string {
  const base = API_BASE_URL.trim().replace(/\/+$/, '')
  return /^https?:\/\//i.test(base) ? `${base.replace(/\/api\/v1$/, '')}${endpoint}` : endpoint
}

function gatewayError(payload: unknown, fallback: string): string {
  const error = payload && typeof payload === 'object' && 'error' in payload ? payload.error : undefined
  return extractApiErrorMessage(error && typeof error === 'object' ? error : payload, fallback)
}

function validateImageItem(value: unknown): ImageGatewayDataItem {
  if (!value || typeof value !== 'object') throw new Error('Invalid image result')
  const item = value as ImageGatewayDataItem
  if (typeof item.b64_json === 'string' && item.b64_json.trim()) return item
  if (typeof item.url === 'string' && item.url.trim()) return item
  throw new Error('Invalid image result')
}

function unwrap<T>(payload: T | ApiResponse<T>): T {
  if (typeof payload !== 'object' || payload === null || !('code' in payload)) return payload as T
  if (payload.code === 0) return payload.data
  throw { ...payload, status: 200, message: gatewayError(payload, 'Request failed') }
}

// One request only: retrying a generation can incur another charge.
async function requestGateway<T>(endpoint: string, key: string, body?: ImageGenerationRequest | FormData, options?: ImageFetchOptions): Promise<T> {
  const controller = new AbortController()
  let reader: ReadableStreamDefaultReader<Uint8Array> | undefined
  const abort = () => {
    controller.abort()
    void reader?.cancel().catch(() => {})
  }
  options?.signal?.addEventListener('abort', abort, { once: true })
  if (options?.signal?.aborted) abort()
  const timeout = setTimeout(abort, IMAGE_GATEWAY_TIMEOUT_MS)
  try {
    const multipart = body instanceof FormData
    const response = await fetch(buildGatewayURL(endpoint), {
      method: body ? 'POST' : 'GET',
      credentials: apiClient.defaults?.withCredentials ? 'include' : 'same-origin',
      headers: {
        Authorization: `Bearer ${key}`,
        ...(body && !multipart ? { 'Content-Type': 'application/json' } : {}),
      },
      body: body ? multipart ? body : JSON.stringify(body) : undefined,
      signal: controller.signal,
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => ({}))
      throw { ...payload, status: response.status, message: gatewayError(payload, `Request failed (${response.status})`) }
    }
    if (!response.headers.get('content-type')?.toLowerCase().includes('text/event-stream')) {
      const payload = unwrap<T>(await response.json())
      if (controller.signal.aborted) throw new DOMException('Aborted', 'AbortError')
      if (body) {
        const result = payload as ImageGatewayResponse & { error?: unknown }
        if (result.error) throw new Error(gatewayError(payload, 'Image request failed'))
        if (!Array.isArray(result.data) || !result.data.length) throw new Error('No final image received')
        result.data = result.data.map(validateImageItem)
        result.data.forEach(item => options?.onCompleted?.({ ...item, output_format: item.output_format || result.output_format }))
      }
      return payload
    }
    if (!response.body) throw new Error('Empty image stream')
    reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    let data: string[] = []
    let eventName = ''
    let done = false
    const result: ImageGatewayResponse = { created: Math.floor(Date.now() / 1000), data: [] }
    const seen = new Set<string>()
    const dispatch = () => {
      if (controller.signal.aborted) throw new DOMException('Aborted', 'AbortError')
      if (!data.length) { eventName = ''; return }
      const text = data.join('\n')
      data = []
      if (text.trim() === '[DONE]') { done = true; return }
      const payload = JSON.parse(text)
      const type = payload.type || eventName
      eventName = ''
      if (type === 'error' || payload.error || (typeof payload.code === 'number' && payload.code !== 0)) {
        throw new Error(gatewayError(payload, 'Image stream failed'))
      }
      if (!/^image_(generation|edit)\.(partial_image|completed)$/.test(type)) return
      const item = validateImageItem({ ...payload, b64_json: payload.b64_json || payload.partial_image_b64 })
      if (type.endsWith('.partial_image')) { options?.onPartial?.(item); return }
      const identity = item.b64_json || item.url!
      if (seen.has(identity)) return
      seen.add(identity)
      result.data.push(item)
      if (typeof payload.created === 'number') result.created = payload.created
      options?.onCompleted?.(item)
    }
    const line = (value: string) => {
      if (!value) { dispatch(); return }
      if (value.startsWith(':')) return
      const colon = value.indexOf(':')
      const field = colon === -1 ? value : value.slice(0, colon)
      const content = colon === -1 ? '' : value.slice(colon + 1).replace(/^ /, '')
      if (field === 'data') data.push(content)
      if (field === 'event') eventName = content
    }
    while (!done) {
      const chunk = await reader.read()
      if (controller.signal.aborted) throw new DOMException('Aborted', 'AbortError')
      buffer += decoder.decode(chunk.value, { stream: !chunk.done })
      let end: number
      while (!done && (end = buffer.indexOf('\n')) !== -1) {
        line(buffer.slice(0, end).replace(/\r$/, ''))
        buffer = buffer.slice(end + 1)
      }
      if (chunk.done) {
        if (!done && buffer) line(buffer.replace(/\r$/, ''))
        if (!done) dispatch()
        break
      }
    }
    if (!result.data.length) throw new Error('No final image received')
    return result as T
  } finally {
    clearTimeout(timeout)
    options?.signal?.removeEventListener('abort', abort)
    await reader?.cancel().catch(() => {})
    reader?.releaseLock()
  }
}

export async function listModels(key: string, options?: FetchOptions): Promise<string[]> {
  const result = await requestGateway<{ data: { id: string }[] }>('/v1/models', key, undefined, options)
  return [...new Set((result.data || []).map(model => model.id).filter(id => /^gpt-image-/i.test(id)))].sort()
}

export async function listHistory(params?: ImageHistoryListParams, options?: FetchOptions): Promise<PaginatedResponse<ImageHistoryListItem>> {
  const { data } = await apiClient.get<PaginatedResponse<ImageHistoryListItem>>('/images/history', { params, ...(options?.signal ? { signal: options.signal } : {}) })
  return data
}

export async function getHistoryDetail(id: number, options?: FetchOptions): Promise<ImageHistoryDetail> {
  const { data } = options?.signal
    ? await apiClient.get<ImageHistoryDetail>(`/images/history/${id}`, { signal: options.signal })
    : await apiClient.get<ImageHistoryDetail>(`/images/history/${id}`)
  return data
}

export function generate(payload: ImageGenerationRequest, key: string, options?: ImageFetchOptions): Promise<ImageGatewayResponse> {
  return requestGateway('/v1/images/generations', key, payload, options)
}

export function edit(payload: FormData, key: string, options?: ImageFetchOptions): Promise<ImageGatewayResponse> {
  return requestGateway('/v1/images/edits', key, payload, options)
}

export const imagesAPI = { listModels, listHistory, getHistoryDetail, generate, edit }
export default imagesAPI
