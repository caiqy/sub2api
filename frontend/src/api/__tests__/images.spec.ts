import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { defaults: { withCredentials: true }, get } }))
const fetchMock = vi.fn()
const final = (b64_json = 'final') => ({ type: 'image_generation.completed', b64_json, output_format: 'webp' })
function stream(text: string, width = 1) {
  const bytes = new TextEncoder().encode(text)
  const cancel = vi.fn()
  const body = new ReadableStream<Uint8Array>({
    start(controller) { for (let i = 0; i < bytes.length; i += width) controller.enqueue(bytes.slice(i, i + width)); controller.close() }, cancel,
  })
  fetchMock.mockResolvedValue({ ok: true, headers: new Headers({ 'content-type': 'text/event-stream' }), body })
  return cancel
}
beforeEach(() => { vi.resetModules(); fetchMock.mockReset(); get.mockReset(); vi.stubGlobal('fetch', fetchMock) })
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs() })
describe('image transport', () => {
  it('uses fetch auth, absolute endpoint and JSON fallback format metadata', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://gateway.example/api/v1/')
    fetchMock.mockResolvedValue({ ok: true, headers: new Headers({ 'content-type': 'application/json' }), json: async () => ({ created: 1, output_format: 'jpeg', data: [{ b64_json: 'abc' }] }) })
    const { imagesAPI } = await import('../images')
    const completed = vi.fn()
    await imagesAPI.generate({ prompt: 'fox' }, 'selected', { onCompleted: completed })
    expect(fetchMock).toHaveBeenCalledWith('https://gateway.example/v1/images/generations', expect.objectContaining({ method: 'POST', credentials: 'include', body: '{"prompt":"fox"}', headers: { Authorization: 'Bearer selected', 'Content-Type': 'application/json' } }))
    expect(completed).toHaveBeenCalledWith({ b64_json: 'abc', output_format: 'jpeg' })
  })
  it('does not force multipart content-type', async () => {
    stream(`data: ${JSON.stringify({ ...final(), type: 'image_edit.completed' })}\n\n`)
    const { imagesAPI } = await import('../images')
    const body = new FormData()
    body.append('image[]', new File(['x'], 'x.png', { type: 'image/png' }))
    await imagesAPI.edit(body, 'selected')
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ body, headers: { Authorization: 'Bearer selected' } })
    expect(fetchMock.mock.calls[0][1].headers).not.toHaveProperty('Content-Type')
  })
  it('parses split UTF8, CRLF, comments, multiline, finals and optional DONE without duplication', async () => {
    const partial = { type: 'image_edit.partial_image', b64_json: 'draft', revised_prompt: '狐狸' }
    stream(`: heartbeat\r\n\r\nevent: image_edit.partial_image\r\ndata: ${JSON.stringify(partial)}\r\n\r\ndata: {"type":"image_generation.completed",\r\ndata: "b64_json":"one","output_format":"webp"}\r\n\r\ndata: ${JSON.stringify(final('one'))}\r\n\r\ndata: ${JSON.stringify(final('two'))}\r\n\r\ndata: [DONE]\r\n\r\n`)
    const { imagesAPI } = await import('../images')
    const onPartial = vi.fn(), onCompleted = vi.fn()
    const result = await imagesAPI.generate({ prompt: 'x' }, 'k', { onPartial, onCompleted })
    expect(onPartial).toHaveBeenCalledWith(expect.objectContaining({ revised_prompt: '狐狸', b64_json: 'draft' }))
    expect(result.data.map(item => item.b64_json)).toEqual(['one', 'two'])
    expect(onCompleted).toHaveBeenCalledTimes(2)
  })
  it('fails partial-only EOF and does not retry', async () => {
    stream('data: {"type":"image_generation.partial_image","b64_json":"draft"}\n\n')
    const { imagesAPI } = await import('../images')
    await expect(imagesAPI.generate({ prompt: 'x' }, 'k')).rejects.toThrow('No final image received')
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })
  it('keeps delivered finals before a nested error envelope', async () => {
    stream(`data: ${JSON.stringify(final())}\n\ndata: {"error":{"message":"quota exceeded"}}\n\n`)
    const { imagesAPI } = await import('../images')
    const onCompleted = vi.fn()
    await expect(imagesAPI.generate({ prompt: 'x' }, 'k', { onCompleted })).rejects.toThrow('quota exceeded')
    expect(onCompleted).toHaveBeenCalledTimes(1)
  })
  it('cancels an open reader on abort', async () => {
    const cancel = vi.fn()
    fetchMock.mockResolvedValue({ ok: true, headers: new Headers({ 'content-type': 'text/event-stream' }), body: new ReadableStream({ cancel }) })
    const { imagesAPI } = await import('../images')
    const controller = new AbortController()
    const pending = imagesAPI.generate({ prompt: 'x' }, 'k', { signal: controller.signal })
    const check = expect(pending).rejects.toMatchObject({ name: 'AbortError' })
    await Promise.resolve(); controller.abort(); await check
    expect(cancel).toHaveBeenCalled()
  })
  it('lists only visible image candidates and keeps empty distinct from errors', async () => {
    const { imagesAPI } = await import('../images')
    fetchMock.mockResolvedValue({ ok: true, headers: new Headers(), json: async () => ({ data: [{ id: 'gpt-image-2.5-flare' }, { id: 'gpt-5' }, { id: 'gpt-image-2.5-flare' }] }) })
    expect(await imagesAPI.listModels('k')).toEqual(['gpt-image-2.5-flare'])
    fetchMock.mockResolvedValue({ ok: true, headers: new Headers(), json: async () => ({ data: [] }) })
    expect(await imagesAPI.listModels('k')).toEqual([])
    fetchMock.mockResolvedValue({ ok: false, status: 403, json: async () => ({ error: { message: 'denied' } }) })
    await expect(imagesAPI.listModels('k')).rejects.toMatchObject({ status: 403, message: 'denied' })
  })
  it('rejects malformed JSON final entries', async () => {
    const { imagesAPI } = await import('../images')
    for (const item of [{}, { url: 42 }, { b64_json: '' }, null]) {
      fetchMock.mockResolvedValue({ ok: true, headers: new Headers(), json: async () => ({ data: [item] }) })
      await expect(imagesAPI.generate({ prompt: 'x' }, 'k')).rejects.toThrow('Invalid image result')
    }
  })
  it('keeps authenticated history on shared client', async () => {
    get.mockResolvedValue({ data: {} })
    const { imagesAPI } = await import('../images')
    const signal = new AbortController().signal
    await imagesAPI.listHistory({ page: 2 }, { signal }); await imagesAPI.getHistoryDetail(3)
    expect(get).toHaveBeenCalledWith('/images/history', { params: { page: 2 }, signal })
    expect(get).toHaveBeenCalledWith('/images/history/3')
  })
})
