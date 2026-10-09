import { effectScope } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { normalizeGatewayResult, useImageGeneration } from '../useImageGeneration'
import type { ImageFetchOptions } from '@/api/images'
import type { ImageGatewayResponse } from '@/types'
const { edit, generate } = vi.hoisted(() => ({ edit: vi.fn(), generate: vi.fn() }))
vi.mock('@/api', () => ({ imagesAPI: { edit, generate } }))
beforeEach(() => { generate.mockReset(); edit.mockReset() })
afterEach(() => vi.useRealTimers())
const response = (url = 'https://cdn.example/final.webp'): ImageGatewayResponse => ({ created: 1, data: [{ url }] })
describe('image generation state', () => {
  it('prevents duplicate requests, preserves elapsed and uses streaming', async () => {
    vi.useFakeTimers()
    let resolve!: (value: ImageGatewayResponse) => void
    generate.mockImplementation(() => new Promise(done => { resolve = done }))
    const state = useImageGeneration()
    const pending = state.submitGenerate({ prompt: 'x' }, 'k')
    expect(await state.submitGenerate({ prompt: 'second' }, 'k')).toBeNull()
    await vi.advanceTimersByTimeAsync(3100)
    resolve(response()); await pending
    expect(generate).toHaveBeenCalledTimes(1)
    expect(generate.mock.calls[0][0]).toMatchObject({ stream: true, partial_images: 1 })
    expect(state.loadingSeconds.value).toBe(3)
    expect(state.durationMs.value).toBe(3100)
    expect(state.status.value).toBe('success')
  })
  it('shows partial immediately, retains finals on failure, never auto retries', async () => {
    generate.mockImplementation(async (_payload, _key, options: ImageFetchOptions) => {
      options.onPartial?.({ b64_json: 'draft', output_format: 'webp' })
      options.onCompleted?.({ b64_json: 'final', output_format: 'jpeg' })
      throw new Error('stream lost')
    })
    const state = useImageGeneration()
    const pending = state.submitGenerate({ prompt: 'x' }, 'k')
    expect(state.partialResult.value?.src).toBe('data:image/webp;base64,draft')
    await pending
    expect(state.status.value).toBe('error')
    expect(state.results.value[0]?.mimeType).toBe('image/jpeg')
    expect(state.error.value).toBe('stream lost')
    expect(generate).toHaveBeenCalledTimes(1)
  })
  it('ignores stopped stale callbacks and completion after a new request starts', async () => {
    let resolveOld!: (value: ImageGatewayResponse) => void
    let old!: ImageFetchOptions
    generate.mockImplementationOnce((_payload, _key, options) => { old = options; return new Promise(done => { resolveOld = done }) })
    generate.mockResolvedValueOnce(response('https://cdn.example/new.png'))
    const state = useImageGeneration()
    const first = state.submitGenerate({ prompt: 'old' }, 'k')
    state.stop()
    expect(old.signal?.aborted).toBe(true)
    expect(state.status.value).toBe('stopped')
    await state.submitGenerate({ prompt: 'new' }, 'k')
    old.onPartial?.({ b64_json: 'stale' }); old.onCompleted?.({ url: 'https://cdn.example/stale' })
    resolveOld(response('https://cdn.example/old')); await first
    expect(state.status.value).toBe('success')
    expect(state.results.value.map(item => item.src)).toEqual(['https://cdn.example/new.png'])
    expect(state.partialResult.value).toBeNull()
  })
  it('aborts on scope disposal and stops the timer', async () => {
    vi.useFakeTimers()
    let resolve!: (value: ImageGatewayResponse) => void
    generate.mockImplementation(() => new Promise(done => { resolve = done }))
    const scope = effectScope()
    const state = scope.run(useImageGeneration)!
    const pending = state.submitGenerate({ prompt: 'x' }, 'k')
    await vi.advanceTimersByTimeAsync(2000); scope.stop(); await vi.advanceTimersByTimeAsync(3000)
    expect(state.loadingSeconds.value).toBe(2)
    expect(generate.mock.calls[0][2].signal.aborted).toBe(true)
    resolve(response()); await pending
    expect(state.status.value).toBe('stopped')
  })
  it('clones multipart, sends streaming flags and uses requested MIME', async () => {
    edit.mockResolvedValue({ created: 1, data: [{ b64_json: 'abc' }] })
    const payload = new FormData(); payload.set('output_format', 'jpeg')
    const state = useImageGeneration(); await state.submitEdit(payload, 'k')
    expect(payload.has('stream')).toBe(false)
    expect(edit.mock.calls[0][0].get('stream')).toBe('true')
    expect(state.results.value[0]?.src).toBe('data:image/jpeg;base64,abc')
  })
  it('uses actual metadata and magic instead of a PNG fallback', () => {
    expect(normalizeGatewayResult({ b64_json: 'abc', output_format: 'jpeg' }, 'png').mimeType).toBe('image/jpeg')
    expect(normalizeGatewayResult({ b64_json: '/9j/test' }, 'png').mimeType).toBe('image/jpeg')
    expect(() => normalizeGatewayResult({ url: '/relative.png' })).toThrow('Invalid image result URL')
    expect(() => normalizeGatewayResult({ url: 'javascript:alert(1)' })).toThrow('Invalid image result URL')
    expect(() => normalizeGatewayResult({ url: undefined } as never)).toThrow('Invalid image result URL')
    expect(() => normalizeGatewayResult({ url: 42 } as never)).toThrow('Invalid image result URL')
    expect(() => normalizeGatewayResult({ b64_json: 'unknown' })).toThrow('Unknown image output format')
  })
})
