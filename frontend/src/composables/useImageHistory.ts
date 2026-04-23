import { ref } from 'vue'

import { imagesAPI } from '@/api'
import type { ImageHistoryDetail, ImageHistoryListItem, ImageHistoryListParams } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

type LoadState = 'idle' | 'loading' | 'success' | 'error'

export function useImageHistory() {
  const items = ref<ImageHistoryListItem[]>([])
  const listState = ref<LoadState>('idle')
  const listError = ref('')
  const detail = ref<ImageHistoryDetail | null>(null)
  const detailState = ref<LoadState>('idle')
  const detailError = ref('')
  const selectedHistoryId = ref<number | null>(null)
  let latestListRequestId = 0
  let latestDetailRequestId = 0

  async function loadHistory(params?: ImageHistoryListParams) {
    const requestId = ++latestListRequestId
    listState.value = 'loading'
    listError.value = ''

    try {
      const response = await imagesAPI.listHistory({ page: 1, page_size: 20, ...params })
      if (requestId !== latestListRequestId) {
        return items.value
      }

      items.value = response.items ?? []
      listState.value = 'success'

      if (selectedHistoryId.value && !items.value.some((item) => item.id === selectedHistoryId.value)) {
        selectedHistoryId.value = null
        detail.value = null
        detailState.value = 'idle'
        detailError.value = ''
      }

      return items.value
    } catch (error) {
      if (requestId !== latestListRequestId) {
        return items.value
      }

      items.value = []
      listState.value = 'error'
      listError.value = extractApiErrorMessage(error, 'Failed to load image history.')
      return []
    }
  }

  async function selectHistory(id: number) {
    const requestId = ++latestDetailRequestId
    selectedHistoryId.value = id
    detail.value = null
    detailState.value = 'loading'
    detailError.value = ''

    try {
      const response = await imagesAPI.getHistoryDetail(id)
      if (requestId !== latestDetailRequestId || selectedHistoryId.value !== id) {
        return detail.value
      }

      detail.value = response
      detailState.value = 'success'
      return response
    } catch (error) {
      if (requestId !== latestDetailRequestId || selectedHistoryId.value !== id) {
        return detail.value
      }

      detail.value = null
      detailState.value = 'error'
      detailError.value = extractApiErrorMessage(error, 'Failed to load history detail.')
      return null
    }
  }

  return {
    detail,
    detailError,
    detailState,
    items,
    listError,
    listState,
    loadHistory,
    selectedHistoryId,
    selectHistory,
  }
}
