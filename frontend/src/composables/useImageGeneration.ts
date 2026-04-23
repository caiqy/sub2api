import { ref } from 'vue'

import { imagesAPI } from '@/api'
import type { ImageGatewayDataItem, ImageGatewayResponse, ImageGenerationRequest } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

const OUTPUT_FORMAT_MIME_MAP: Record<string, string> = {
  jpeg: 'image/jpeg',
  jpg: 'image/jpeg',
  png: 'image/png',
  webp: 'image/webp',
}

export interface ImageResultPreview {
  src: string
  revisedPrompt?: string
  source: 'data-url' | 'url'
}

function normalizeOutputFormat(value: unknown): string | undefined {
  if (typeof value !== 'string') {
    return undefined
  }

  const normalized = value.trim().toLowerCase()
  return normalized || undefined
}

function inferMimeType(item: ImageGatewayDataItem, requestedOutputFormat?: string): string {
  const metadata = item as Record<string, unknown>
  const candidateValues = [
    metadata.mime_type,
    metadata.mimeType,
    metadata.content_type,
    metadata.contentType,
    metadata.output_format,
    metadata.outputFormat,
    metadata.format,
    requestedOutputFormat,
  ]

  for (const candidate of candidateValues) {
    const normalized = normalizeOutputFormat(candidate)
    if (!normalized) {
      continue
    }

    if (normalized.startsWith('image/')) {
      return normalized
    }

    if (OUTPUT_FORMAT_MIME_MAP[normalized]) {
      return OUTPUT_FORMAT_MIME_MAP[normalized]
    }
  }

  return OUTPUT_FORMAT_MIME_MAP.png
}

function getRequestedOutputFormat(payload: ImageGenerationRequest | FormData): string | undefined {
  if (payload instanceof FormData) {
    return normalizeOutputFormat(payload.get('output_format'))
  }

  return normalizeOutputFormat(payload.output_format)
}

function normalizeGatewayResult(item: ImageGatewayDataItem, requestedOutputFormat?: string): ImageResultPreview {
  if ('b64_json' in item) {
    return {
      src: `data:${inferMimeType(item, requestedOutputFormat)};base64,${item.b64_json}`,
      revisedPrompt: item.revised_prompt,
      source: 'data-url',
    }
  }

  return {
    src: item.url,
    revisedPrompt: item.revised_prompt,
    source: 'url',
  }
}

export function useImageGeneration() {
  const isLoading = ref(false)
  const error = ref('')
  const results = ref<ImageResultPreview[]>([])
  const lastResponse = ref<ImageGatewayResponse | null>(null)

  async function submitGenerate(payload: ImageGenerationRequest, selectedApiKey: string) {
    if (isLoading.value || !selectedApiKey.trim()) {
      return null
    }

    isLoading.value = true
    error.value = ''

    try {
      const response = await imagesAPI.generate(payload, selectedApiKey)
      lastResponse.value = response
      results.value = response.data.map((item) => normalizeGatewayResult(item, getRequestedOutputFormat(payload)))
      return response
    } catch (err) {
      lastResponse.value = null
      results.value = []
      error.value = extractApiErrorMessage(err, 'Request failed')
      return null
    } finally {
      isLoading.value = false
    }
  }

  async function submitEdit(payload: FormData, selectedApiKey: string) {
    if (isLoading.value || !selectedApiKey.trim()) {
      return null
    }

    isLoading.value = true
    error.value = ''

    try {
      const response = await imagesAPI.edit(payload, selectedApiKey)
      lastResponse.value = response
      results.value = response.data.map((item) => normalizeGatewayResult(item, getRequestedOutputFormat(payload)))
      return response
    } catch (err) {
      lastResponse.value = null
      results.value = []
      error.value = extractApiErrorMessage(err, 'Request failed')
      return null
    } finally {
      isLoading.value = false
    }
  }

  function clearError() {
    error.value = ''
  }

  function clearResults() {
    results.value = []
    lastResponse.value = null
  }

  return {
    isLoading,
    error,
    results,
    lastResponse,
    submitGenerate,
    submitEdit,
    clearError,
    clearResults,
  }
}
