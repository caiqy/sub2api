import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useImageHistory } from '@/composables/useImageHistory'
import type { ImageHistoryDetail, ImageHistoryListItem } from '@/types'

const { getHistoryDetail, listHistory } = vi.hoisted(() => ({ getHistoryDetail: vi.fn(), listHistory: vi.fn() }))
vi.mock('@/api', () => ({ imagesAPI: { getHistoryDetail, listHistory } }))

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>(r => { resolve = r })
  return { promise, resolve }
}
function item(id: number): ImageHistoryListItem {
  return { id, api_key_id: 7, mode: 'generate', status: 'success', model: 'gpt-image-2.5-flare', image_count: 1, actual_cost: 0.1, created_at: '2026-10-09T10:00:00Z' }
}
function detail(id: number): ImageHistoryDetail {
  return { ...item(id), n: 1, had_source_image: false, had_mask: false, replay: { mode: 'generate', model: 'gpt-image-2.5-flare', n: 1, requires_source_image_upload: false, requires_mask_upload: false } }
}
const page = (id: number) => ({ items: [item(id)], total: 45, page: id, page_size: 20 })

describe('useImageHistory', () => {
  beforeEach(() => vi.clearAllMocks())

  it('deduplicates an in-flight list, caches it, and explicitly refreshes it', async () => {
    const response = deferred<ReturnType<typeof page>>()
    listHistory.mockReturnValueOnce(response.promise).mockResolvedValue(page(1))
    const history = useImageHistory()
    const first = history.loadHistory({ page: 1 })
    const same = history.loadHistory({ page: 1 })
    expect(listHistory).toHaveBeenCalledTimes(1)
    response.resolve(page(1))
    await Promise.all([first, same])
    await history.loadHistory({ page: 1 })
    expect(listHistory).toHaveBeenCalledTimes(1)
    expect(history.total.value).toBe(45)
    expect(history.pageSize.value).toBe(20)
    expect(getHistoryDetail).not.toHaveBeenCalled()
    await history.loadHistory({ page: 1 }, { force: true })
    expect(listHistory).toHaveBeenCalledTimes(2)
    history.invalidateHistory()
    await history.loadHistory({ page: 1 })
    expect(listHistory).toHaveBeenCalledTimes(3)
  })

  it('ignores stale list and selected-detail responses after a page change', async () => {
    const first = deferred<ReturnType<typeof page>>()
    const second = deferred<ReturnType<typeof page>>()
    const oldDetail = deferred<ImageHistoryDetail>()
    listHistory.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise)
    getHistoryDetail.mockReturnValueOnce(oldDetail.promise)
    const history = useImageHistory()
    const request1 = history.loadHistory({ page: 1 })
    const selection = history.selectHistory(31)
    const request2 = history.loadHistory({ page: 2 })
    second.resolve(page(2))
    await request2
    first.resolve(page(1))
    oldDetail.resolve(detail(31))
    await Promise.all([request1, selection])
    expect(history.items.value.map(row => row.id)).toEqual([2])
    expect(history.page.value).toBe(2)
    expect(history.selectedHistoryId.value).toBeNull()
    expect(history.detail.value).toBeNull()
  })

  it('ignores old selection, deduplicates detail, and reuses only the latest detail', async () => {
    const first = deferred<ImageHistoryDetail>()
    const second = deferred<ImageHistoryDetail>()
    getHistoryDetail.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise)
    const history = useImageHistory()
    const request1 = history.selectHistory(31)
    const request2 = history.selectHistory(32)
    const duplicate = history.selectHistory(32)
    expect(getHistoryDetail).toHaveBeenCalledTimes(2)
    second.resolve(detail(32))
    await Promise.all([request2, duplicate])
    first.resolve(detail(31))
    await request1
    await history.selectHistory(32)
    expect(getHistoryDetail).toHaveBeenCalledTimes(2)
    expect(history.detail.value?.id).toBe(32)
    expect(history.selectedHistoryId.value).toBe(32)
  })

  it('marks a pruned detail unavailable instead of inventing image or replay data', async () => {
    getHistoryDetail.mockRejectedValueOnce({ status: 404, code: 'USAGE_LOG_DETAIL_NOT_FOUND', message: 'gone' })
    const history = useImageHistory()
    await history.selectHistory(88)
    expect(history.detailState.value).toBe('error')
    expect(history.detailUnavailable.value).toBe(true)
    expect(history.detail.value).toBeNull()
  })
})
