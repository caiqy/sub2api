import { getCurrentScope, onScopeDispose, ref } from 'vue'

import { imagesAPI } from '@/api'
import type { ImageHistoryDetail, ImageHistoryListItem, ImageHistoryListParams, PaginatedResponse } from '@/types'
import { extractApiErrorCode, extractApiErrorMessage } from '@/utils/apiError'

type LoadState = 'idle' | 'loading' | 'success' | 'error'

export function useImageHistory() {
  const items = ref<ImageHistoryListItem[]>([])
  const listState = ref<LoadState>('idle')
  const listError = ref('')
  const detail = ref<ImageHistoryDetail | null>(null)
  const detailState = ref<LoadState>('idle')
  const detailError = ref('')
  const detailUnavailable = ref(false)
  const selectedHistoryId = ref<number | null>(null)
  const page = ref(1)
  const pageSize = ref(20)
  const total = ref(0)
  let latestListRequestId = 0
  let latestDetailRequestId = 0
  let lastListKey = ''
  let lastParams: ImageHistoryListParams = { page: 1, page_size: 20 }
  let cachedDetail: ImageHistoryDetail | null = null
  const listRequests = new Map<string, Promise<PaginatedResponse<ImageHistoryListItem>>>()
  const detailRequests = new Map<number, Promise<ImageHistoryDetail>>()

  function clearSelection() {
    latestDetailRequestId += 1
    selectedHistoryId.value = null
    detail.value = null
    detailState.value = 'idle'
    detailError.value = ''
    detailUnavailable.value = false
  }

  function invalidateHistory() {
    lastListKey = ''
  }

  async function loadHistory(params?: ImageHistoryListParams, options: { force?: boolean } = {}) {
    const query = { page: 1, page_size: 20, ...(params ?? lastParams) }
    const key = JSON.stringify([query.page, query.page_size, query.api_key_id, query.tab, query.status])
    if (!options.force && key === lastListKey && listState.value === 'success') return items.value
    const requestId = ++latestListRequestId
    if (JSON.stringify(lastParams) !== JSON.stringify(query)) clearSelection()
    lastParams = query
    page.value = query.page
    listState.value = 'loading'
    listError.value = ''
    let request = listRequests.get(key)
    if (!request || options.force) {
      request = imagesAPI.listHistory(query)
      listRequests.set(key, request)
    }
    try {
      const response = await request
      if (requestId !== latestListRequestId) return items.value
      items.value = response.items ?? []
      page.value = response.page ?? query.page
      pageSize.value = response.page_size ?? query.page_size
      total.value = response.total ?? items.value.length
      lastListKey = key
      listState.value = 'success'
      if (selectedHistoryId.value && !items.value.some(item => item.id === selectedHistoryId.value)) clearSelection()
      return items.value
    } catch (error) {
      if (requestId !== latestListRequestId) return items.value
      items.value = []
      total.value = 0
      listState.value = 'error'
      listError.value = extractApiErrorMessage(error, 'Failed to load image history.')
      return []
    } finally {
      if (listRequests.get(key) === request) listRequests.delete(key)
    }
  }

  async function selectHistory(id: number) {
    const requestId = ++latestDetailRequestId
    selectedHistoryId.value = id
    detail.value = null
    detailError.value = ''
    detailUnavailable.value = false
    if (cachedDetail?.id === id) {
      detail.value = cachedDetail
      detailState.value = 'success'
      return cachedDetail
    }
    detailState.value = 'loading'
    let request = detailRequests.get(id)
    if (!request) {
      request = imagesAPI.getHistoryDetail(id)
      detailRequests.set(id, request)
    }
    try {
      const response = await request
      if (requestId !== latestDetailRequestId || selectedHistoryId.value !== id) return detail.value
      cachedDetail = response
      detail.value = response
      detailState.value = 'success'
      return response
    } catch (error) {
      if (requestId !== latestDetailRequestId || selectedHistoryId.value !== id) return detail.value
      detail.value = null
      detailState.value = 'error'
      detailUnavailable.value = extractApiErrorCode(error) === 'USAGE_LOG_DETAIL_NOT_FOUND'
      detailError.value = extractApiErrorMessage(error, 'Failed to load history detail.')
      return null
    } finally {
      if (detailRequests.get(id) === request) detailRequests.delete(id)
    }
  }

  if (getCurrentScope()) onScopeDispose(() => {
    latestListRequestId += 1
    latestDetailRequestId += 1
    cachedDetail = null
    listRequests.clear()
    detailRequests.clear()
  })

  return {
    detail, detailError, detailState, detailUnavailable, items, listError, listState,
    page, pageSize, total, loadHistory, invalidateHistory, selectedHistoryId, selectHistory,
  }
}
