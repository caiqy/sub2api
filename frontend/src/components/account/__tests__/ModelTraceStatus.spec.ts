import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ModelTraceStatus from '../ModelTraceStatus.vue'
vi.mock('@/utils/format', () => ({ formatDateTime: (value: string) => value }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string, params?: unknown) => `${key}${params ? JSON.stringify(params) : ''}` }) }))
describe('account IQ summary', () => {
  it('uses only the latest valid summary and opens history', async () => {
    const account = { platform: 'openai' as const, extra: { modeltrace_latest: { result: 'normal', model: 'gpt-6-sol', target_model: 'gpt-6-sol', finished_at: '2026-09-26T01:00:00Z' } } }
    const w = mount(ModelTraceStatus, { props: { account } })
    expect(w.text()).toContain('admin.modeltrace.result.normal')
    expect(w.get('button').attributes('title')).toContain('gpt-6-sol')
    expect(w.get('button').attributes('title')).toContain('2026')
    await w.get('button').trigger('click')
    expect(w.emitted('history')).toHaveLength(1)
    await w.setProps({ account: { ...account, extra: { modeltrace_latest: { ...account.extra.modeltrace_latest, result: 'degraded' } } } })
    expect(w.text()).toContain('admin.modeltrace.result.degraded')
  })
  it('shows untested when there is no valid result, and hides non-OpenAI accounts', async () => {
    const w = mount(ModelTraceStatus, { props: { account: { platform: 'openai', extra: {} } } })
    expect(w.text()).toContain('admin.modeltrace.result.untested')
    await w.setProps({ account: { platform: 'anthropic', extra: {} } })
    expect(w.find('button').exists()).toBe(false)
  })
})
