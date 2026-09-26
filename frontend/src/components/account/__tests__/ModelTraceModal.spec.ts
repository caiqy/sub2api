import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ModelTraceModal from '../ModelTraceModal.vue'
import type { ModelTraceTask } from '@/api/admin/modeltrace'
const api = vi.hoisted(() => ({ getModels: vi.fn(), list: vi.fn(), start: vi.fn() }))
vi.mock('@/api/admin/modeltrace', () => ({ default: api }))
vi.mock('@/utils/format', () => ({ formatDateTime: (value: string) => value }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, params?: unknown) => `${key}${params ? JSON.stringify(params) : ''}` }) }))
enableAutoUnmount(afterEach)
const account = (id = 1) => ({ id, name: `account-${id}`, platform: 'openai' as const })
const task = (status: ModelTraceTask['status'] = 'running', id = 1): ModelTraceTask => ({
  id, account_id: 1, status, source: 'manual', model: 'gpt-6-astra', target_model: 'gpt-6-astra', rounds: 3, completed_rounds: 2,
  result: status === 'completed' ? 'normal' : '', winner: 'gpt-6-astra', probabilities: { 'gpt-6-astra': 0.9 },
  version: 'fingerprint-v1', created_at: '2026-09-26T01:00:00Z', duration_ms: 1000, error: ''
})
const history = (active: ModelTraceTask | null = null, items: ModelTraceTask[] = [], page = 1, total = items.length) => ({ active, items, page, total, page_size: 20, pages: Math.ceil(total / 20) })
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((res, rej) => { resolve = res; reject = rej })
  return { resolve, reject, promise }
}
function open(initialTab: 'start' | 'history' = 'start') {
  return mount(ModelTraceModal, { props: { show: true, account: account(), initialTab }, global: { stubs: {
    BaseDialog: { props: ['show'], emits: ['close'], template: '<div v-if="show"><button data-testid="close" @click="$emit(\'close\')">close</button><slot /></div>' },
    Pagination: { name: 'Pagination', props: ['page', 'total'], emits: ['update:page'], template: '<button data-testid="next-page" @click="$emit(\'update:page\', 2)">next</button>' }
  } } })
}
beforeEach(() => {
  vi.useFakeTimers()
  vi.resetAllMocks()
  api.getModels.mockResolvedValue({ models: ['gpt-6-astra', 'gpt-6-sol', 'gpt-5.6-sol'], version: 'v1' })
  api.list.mockResolvedValue(history())
  api.start.mockResolvedValue(task())
})
afterEach(() => vi.useRealTimers())
describe('ModelTrace modal lifecycle', () => {
  it('defaults to one round and sends a task only after confirmation', async () => {
    const w = open(); await flushPromises()
    expect(api.start).not.toHaveBeenCalled()
    expect(w.findAll('select')[1].element.value).toBe('1')
    await w.get('form').trigger('submit')
    await flushPromises()
    expect(api.start).toHaveBeenCalledOnce()
    expect(api.start).toHaveBeenCalledWith(1, { model: 'gpt-6-astra', rounds: 1 })
    expect(w.get('[data-testid="modeltrace-start"]').attributes('disabled')).toBeDefined()
  })
  it('restores an active task, uses one serial polling loop and stops after completion', async () => {
    const poll = deferred<ReturnType<typeof history>>()
    api.list.mockResolvedValueOnce(history(task(), [task()])).mockReturnValueOnce(poll.promise).mockResolvedValue(history(null, [task('completed')]))
    const w = open(); await flushPromises()
    expect(api.start).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(2000)
    expect(api.list).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(10000)
    expect(api.list).toHaveBeenCalledTimes(2)
    poll.resolve(history(task(), [task()])); await flushPromises()
    await vi.advanceTimersByTimeAsync(2000); await flushPromises()
    expect(w.emitted('completed')).toEqual([[1]])
    await vi.advanceTimersByTimeAsync(10000)
    expect(api.list).toHaveBeenCalledTimes(3)
  })
  it('stops on close without cancelling the task and recovers on reopen', async () => {
    api.list.mockResolvedValue(history(task(), [task()]))
    const w = open(); await flushPromises()
    await w.get('[data-testid="close"]').trigger('click')
    await w.setProps({ show: false })
    await vi.advanceTimersByTimeAsync(5000)
    expect(api.list).toHaveBeenCalledTimes(1)
    await w.setProps({ show: true }); await flushPromises()
    expect(api.list).toHaveBeenCalledTimes(2)
    expect(api.start).not.toHaveBeenCalled()
    expect(w.text()).toContain('admin.modeltrace.backgroundNotice')
  })
  it('ignores late model/history responses and errors from a different account', async () => {
    const oldModels = deferred<{ models: string[]; version: string }>()
    const oldHistory = deferred<ReturnType<typeof history>>()
    api.getModels.mockReturnValueOnce(oldModels.promise)
    api.list.mockReturnValueOnce(oldHistory.promise)
    const w = open()
    await w.setProps({ account: account(2) }); await flushPromises()
    oldModels.resolve({ models: ['obsolete'], version: 'obsolete' })
    oldHistory.reject({ message: 'obsolete error' }); await flushPromises()
    expect(w.text()).not.toContain('obsolete')
    expect(w.get('[data-testid="modeltrace-start"]').attributes('disabled')).toBeUndefined()
    expect(api.getModels).toHaveBeenLastCalledWith(2)
  })
  it('does not apply a late POST to a new account or start its polling loop', async () => {
    const pending = deferred<ModelTraceTask>()
    api.start.mockReturnValueOnce(pending.promise)
    const w = open(); await flushPromises()
    await w.get('form').trigger('submit')
    await w.setProps({ account: account(2) }); await flushPromises()
    pending.resolve({ ...task(), model: 'obsolete' }); await flushPromises()
    await vi.advanceTimersByTimeAsync(5000)
    expect(w.text()).not.toContain('obsolete')
    expect(api.list).toHaveBeenCalledTimes(2)
  })
  it('blocks starts on failed recovery, shows server errors and supports retry', async () => {
    api.list.mockRejectedValueOnce({ message: 'account unavailable' })
    const w = open(); await flushPromises()
    expect(w.text()).toContain('account unavailable')
    await w.get('form').trigger('submit')
    expect(api.start).not.toHaveBeenCalled()
    await w.get('[role="alert"] button').trigger('click'); await flushPromises()
    expect(w.get('[data-testid="modeltrace-start"]').attributes('disabled')).toBeUndefined()
  })
  it('fetches paginated history and renders all task metadata and probabilities', async () => {
    const failed = { ...task('failed'), source: 'auto' as const, error: 'quota exhausted', target_model: 'gpt-6-sol', finished_at: '2026-09-26T01:01:00Z' }
    api.list.mockResolvedValueOnce(history(null, [failed], 1, 21)).mockResolvedValueOnce(history(null, [failed], 2, 21))
    const w = open('history'); await flushPromises()
    expect(w.get('[role="tab"][aria-selected="true"]').text()).toContain('admin.modeltrace.history')
    await w.get('[data-testid="next-page"]').trigger('click'); await flushPromises()
    expect(api.list).toHaveBeenLastCalledWith(1, 2, 20)
    for (const text of ['quota exhausted', 'gpt-6-sol', '90.0%', 'fingerprint-v1', 'admin.modeltrace.source.auto', 'admin.modeltrace.progress']) expect(w.text()).toContain(text)
  })
  it('does not leak timers or apply a pending poll after unmount', async () => {
    const poll = deferred<ReturnType<typeof history>>()
    api.list.mockResolvedValueOnce(history(task())).mockReturnValueOnce(poll.promise)
    const w = open(); await flushPromises()
    await vi.advanceTimersByTimeAsync(2000)
    w.unmount()
    poll.resolve(history(task())); await flushPromises()
    await vi.advanceTimersByTimeAsync(10000)
    expect(api.list).toHaveBeenCalledTimes(2)
    expect(vi.getTimerCount()).toBe(0)
  })
  it('supports keyboard navigation between tabs', async () => {
    const w = open(); await flushPromises()
    await w.findAll('[role="tab"]')[0].trigger('keydown', { key: 'ArrowRight' })
    expect(w.findAll('[role="tab"]')[1].attributes('aria-selected')).toBe('true')
    await w.findAll('[role="tab"]')[1].trigger('keydown', { key: 'Home' })
    expect(w.findAll('[role="tab"]')[0].attributes('aria-selected')).toBe('true')
  })

  it('keeps history accessible when supported models fail to load', async () => {
    api.getModels.mockRejectedValueOnce({ message: 'unsupported credentials' })
    api.list.mockResolvedValueOnce(history(null, [task('failed')]))
    const w = open('history'); await flushPromises()
    expect(w.text()).toContain('admin.modeltrace.status.failed')
    expect(w.text()).toContain('unsupported credentials')
    expect(w.get('[data-testid="modeltrace-start"]').attributes('disabled')).toBeDefined()
    await w.get('form').trigger('submit')
    expect(api.start).not.toHaveBeenCalled()
  })

  it('ignores stale successful history from the previous account', async () => {
    const old = deferred<ReturnType<typeof history>>()
    api.list.mockReturnValueOnce(old.promise)
    const w = open()
    await w.setProps({ account: account(2) }); await flushPromises()
    old.resolve(history({ ...task(), model: 'obsolete' })); await flushPromises()
    await vi.advanceTimersByTimeAsync(5000)
    expect(w.text()).not.toContain('obsolete')
    expect(api.list).toHaveBeenCalledTimes(2)
  })

  it('fetches the terminal task from page one while preserving the selected history page', async () => {
    api.list.mockResolvedValueOnce(history(task(), [task()], 1, 21))
      .mockResolvedValueOnce(history(null, [task('failed', 99)], 2, 21))
      .mockResolvedValueOnce(history(null, [task('completed')], 1, 21))
    const w = open('history'); await flushPromises()
    await w.get('[data-testid="next-page"]').trigger('click'); await flushPromises()
    expect(api.list.mock.calls).toEqual([[1, 1, 20], [1, 2, 20], [1, 1, 20]])
    expect(w.emitted('completed')).toEqual([[1]])
    expect(w.getComponent({ name: 'Pagination' }).props('page')).toBe(2)
    await vi.advanceTimersByTimeAsync(5000)
    expect(api.list).toHaveBeenCalledTimes(3)
  })

  it('submits the selected model and three rounds, and discovers a task after an ambiguous POST failure', async () => {
    api.start.mockRejectedValueOnce({ message: 'connection lost' })
    api.list.mockResolvedValueOnce(history()).mockResolvedValue(history(task()))
    const w = open(); await flushPromises()
    await w.findAll('select')[0].setValue('gpt-5.6-sol')
    await w.findAll('select')[1].setValue('3')
    await w.get('form').trigger('submit'); await flushPromises()
    expect(api.start).toHaveBeenCalledWith(1, { model: 'gpt-5.6-sol', rounds: 3 })
    expect(w.text()).toContain('connection lost')
    expect(w.text()).toContain('admin.modeltrace.backgroundNotice')
    expect(w.get('[data-testid="modeltrace-start"]').attributes('disabled')).toBeDefined()
  })
})
