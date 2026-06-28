import axios from 'axios'

import { apiClient } from './client'
import type {
  ApiResponse,
  FetchOptions,
  ImageGatewayResponse,
  ImageGenerationRequest,
  ImageHistoryDetail,
  ImageHistoryListItem,
  ImageHistoryListParams,
  PaginatedResponse,
} from '@/types'

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '/api/v1'
const IMAGE_GATEWAY_TIMEOUT_MS = 1800000

const imageGatewayClient = axios.create({
  timeout: IMAGE_GATEWAY_TIMEOUT_MS,
  // Keep image gateway requests aligned with the shared API client's cookie/credential policy.
  withCredentials: apiClient.defaults?.withCredentials ?? false,
})

function buildGatewayImageURL(endpoint: '/v1/images/generations' | '/v1/images/edits'): string {
  const normalizedBase = API_BASE_URL.trim().replace(/\/+$/, '')

  if (/^https?:\/\//i.test(normalizedBase)) {
    return `${normalizedBase.replace(/\/api\/v1$/, '')}${endpoint}`
  }

  return endpoint
}

function isWrappedApiResponse<T>(value: T | ApiResponse<T>): value is ApiResponse<T> {
  return typeof value === 'object' && value !== null && 'code' in value
}

function unwrapGatewayResponse<T>(payload: T | ApiResponse<T>): T {
  if (!isWrappedApiResponse(payload)) {
    return payload
  }

  if (payload.code === 0) {
    return payload.data
  }

  const resp = payload as unknown as Record<string, unknown>

  throw {
    status: 200,
    code: payload.code,
    message: payload.message || 'Unknown error',
    reason: resp.reason,
    metadata: resp.metadata,
  }
}

function normalizeGatewayError(error: unknown): never {
  if (axios.isCancel(error)) {
    throw error
  }

  const err = error as {
    response?: {
      status: number
      data?: unknown
    }
    message?: string
    status?: number
    code?: unknown
  }

  if (err.response) {
    const apiData =
      typeof err.response.data === 'object' && err.response.data !== null
        ? (err.response.data as Record<string, unknown>)
        : {}

    throw {
      status: err.response.status,
      code: apiData.code,
      reason: apiData.reason,
      error: apiData.error,
      message:
        (typeof apiData.message === 'string' && apiData.message) ||
        (typeof apiData.detail === 'string' && apiData.detail) ||
        err.message ||
        'Unknown error',
      metadata: apiData.metadata,
    }
  }

  if (typeof err.status === 'number' && typeof err.message === 'string') {
    throw err
  }

  throw {
    status: 0,
    message: 'Network error. Please check your connection.',
  }
}

function buildGatewayRequestConfig(selectedApiKey: string, options?: FetchOptions) {
  const config: {
    headers: {
      Authorization: string
    }
    signal?: AbortSignal
  } = {
    headers: {
      Authorization: `Bearer ${selectedApiKey}`,
    },
  }

  if (options?.signal) {
    config.signal = options.signal
  }

  return config
}

function buildHistoryRequestConfig(options?: FetchOptions) {
  if (!options?.signal) {
    return undefined
  }

  return {
    signal: options.signal,
  }
}

async function postToImageGateway<T>(
  endpoint: '/v1/images/generations' | '/v1/images/edits',
  payload: ImageGenerationRequest | FormData,
  selectedApiKey: string,
  options?: FetchOptions
): Promise<T> {
  try {
    const response = await imageGatewayClient.post<T | ApiResponse<T>>(
      buildGatewayImageURL(endpoint),
      payload,
      buildGatewayRequestConfig(selectedApiKey, options)
    )

    return unwrapGatewayResponse(response.data)
  } catch (error) {
    return normalizeGatewayError(error)
  }
}

export async function listHistory(
  params?: ImageHistoryListParams,
  options?: FetchOptions
): Promise<PaginatedResponse<ImageHistoryListItem>> {
  const { data } = await apiClient.get<PaginatedResponse<ImageHistoryListItem>>('/images/history', {
    params,
    ...buildHistoryRequestConfig(options),
  })
  return data
}

export async function getHistoryDetail(id: number, options?: FetchOptions): Promise<ImageHistoryDetail> {
  const requestConfig = buildHistoryRequestConfig(options)

  if (requestConfig) {
    const { data } = await apiClient.get<ImageHistoryDetail>(`/images/history/${id}`, requestConfig)
    return data
  }

  const { data } = await apiClient.get<ImageHistoryDetail>(`/images/history/${id}`)
  return data
}

export async function generate(
  payload: ImageGenerationRequest,
  selectedApiKey: string,
  options?: FetchOptions
): Promise<ImageGatewayResponse> {
  return postToImageGateway<ImageGatewayResponse>('/v1/images/generations', payload, selectedApiKey, options)
}

export async function edit(
  payload: FormData,
  selectedApiKey: string,
  options?: FetchOptions
): Promise<ImageGatewayResponse> {
  return postToImageGateway<ImageGatewayResponse>('/v1/images/edits', payload, selectedApiKey, options)
}

export const imagesAPI = {
  listHistory,
  getHistoryDetail,
  generate,
  edit,
}

export default imagesAPI
