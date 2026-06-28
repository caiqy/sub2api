import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useImageHistory } from '@/composables/useImageHistory'

const { getHistoryDetail, listHistory } = vi.hoisted(() => ({
  getHistoryDetail: vi.fn(),
  listHistory: vi.fn(),
}))

vi.mock('@/api', () => ({
  imagesAPI: {
    getHistoryDetail,
    listHistory,
  },
}))

describe('useImageHistory', () => {
  beforeEach(() => {
    getHistoryDetail.mockReset()
    listHistory.mockReset()
  })

  it('keeps the latest history list when list requests resolve out of order', async () => {
    let resolveFirst: ((value: { items: Array<{ id: number; mode: string; status: string; model: string; image_count: number; actual_cost: number; created_at: string; api_key_id: number }> }) => void) | undefined
    let resolveSecond: ((value: { items: Array<{ id: number; mode: string; status: string; model: string; image_count: number; actual_cost: number; created_at: string; api_key_id: number }> }) => void) | undefined

    listHistory
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirst = resolve }))
      .mockImplementationOnce(() => new Promise((resolve) => { resolveSecond = resolve }))

    const { items, loadHistory, listState } = useImageHistory()

    const firstRequest = loadHistory({ page: 1 })
    const secondRequest = loadHistory({ page: 2 })

    resolveSecond?.({
      items: [{ id: 202, api_key_id: 7, mode: 'edit', status: 'error', model: 'latest-model', image_count: 0, actual_cost: 0.1, created_at: '2026-04-23T12:00:00Z' }],
    })
    await secondRequest

    resolveFirst?.({
      items: [{ id: 101, api_key_id: 7, mode: 'generate', status: 'success', model: 'stale-model', image_count: 1, actual_cost: 0.2, created_at: '2026-04-23T11:00:00Z' }],
    })
    await firstRequest

    expect(listState.value).toBe('success')
    expect(items.value.map((item) => item.id)).toEqual([202])
  })

  it('keeps the latest selected detail when detail requests resolve out of order', async () => {
    let resolveFirst: ((value: { id: number; replay: { mode: string; model: string; n: number; requires_source_image_upload: boolean; requires_mask_upload: boolean }; mode: string; status: string; model: string; api_key_id: number; n: number; had_source_image: boolean; had_mask: boolean; created_at: string }) => void) | undefined
    let resolveSecond: ((value: { id: number; replay: { mode: string; model: string; n: number; requires_source_image_upload: boolean; requires_mask_upload: boolean }; mode: string; status: string; model: string; api_key_id: number; n: number; had_source_image: boolean; had_mask: boolean; created_at: string }) => void) | undefined

    getHistoryDetail
      .mockImplementationOnce(() => new Promise((resolve) => { resolveFirst = resolve }))
      .mockImplementationOnce(() => new Promise((resolve) => { resolveSecond = resolve }))

    const { detail, detailState, selectHistory, selectedHistoryId } = useImageHistory()

    const firstRequest = selectHistory(31)
    const secondRequest = selectHistory(32)

    resolveSecond?.({
      id: 32,
      api_key_id: 7,
      mode: 'edit',
      status: 'error',
      model: 'latest-model',
      n: 1,
      had_source_image: true,
      had_mask: false,
      replay: {
        mode: 'edit',
        model: 'latest-model',
        n: 1,
        requires_source_image_upload: true,
        requires_mask_upload: false,
      },
      created_at: '2026-04-23T12:00:00Z',
    })
    await secondRequest

    resolveFirst?.({
      id: 31,
      api_key_id: 7,
      mode: 'generate',
      status: 'success',
      model: 'stale-model',
      n: 1,
      had_source_image: false,
      had_mask: false,
      replay: {
        mode: 'generate',
        model: 'stale-model',
        n: 1,
        requires_source_image_upload: false,
        requires_mask_upload: false,
      },
      created_at: '2026-04-23T11:00:00Z',
    })
    await firstRequest

    expect(detailState.value).toBe('success')
    expect(selectedHistoryId.value).toBe(32)
    expect(detail.value?.id).toBe(32)
  })
})
