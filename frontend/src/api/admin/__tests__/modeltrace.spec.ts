import { beforeEach, describe, expect, it, vi } from 'vitest'
import modeltraceAPI from '../modeltrace'
const client = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: client }))
beforeEach(() => { vi.resetAllMocks(); client.get.mockResolvedValue({ data: {} }); client.post.mockResolvedValue({ data: {} }); client.put.mockResolvedValue({ data: {} }) })
describe('ModelTrace API contract', () => {
  it('scopes supported models and history to the account without changing pagination', async () => {
    await modeltraceAPI.getModels()
    expect(client.get).toHaveBeenLastCalledWith('/admin/modeltrace/models', { params: undefined })
    await modeltraceAPI.getModels(42)
    expect(client.get).toHaveBeenLastCalledWith('/admin/modeltrace/models', { params: { account_id: 42 } })
    await modeltraceAPI.list(42, 2, 20)
    expect(client.get).toHaveBeenLastCalledWith('/admin/accounts/42/modeltrace', { params: { page: 2, page_size: 20 } })
  })
  it('uses independent settings and sends model/rounds for manual tasks', async () => {
    const settings = { enabled: false, model: 'gpt-6-astra', rounds: 1, interval_minutes: 60 }
    await modeltraceAPI.getSettings()
    expect(client.get).toHaveBeenLastCalledWith('/admin/modeltrace/settings')
    await modeltraceAPI.updateSettings(settings)
    expect(client.put).toHaveBeenCalledWith('/admin/modeltrace/settings', settings)
    await modeltraceAPI.start(42, { model: 'gpt-6-astra', rounds: 3 })
    expect(client.post).toHaveBeenCalledWith('/admin/accounts/42/modeltrace', { model: 'gpt-6-astra', rounds: 3 })
  })
})
