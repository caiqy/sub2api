import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ModelTraceTaskDetails from '../ModelTraceTaskDetails.vue'
import type { ModelTraceTask } from '@/api/admin/modeltrace'

vi.mock('@/utils/format', () => ({ formatDateTime: (v: string) => v }))
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (k: string) => k })
}))

const baseTask: ModelTraceTask = {
  id: 1,
  account_id: 10,
  source: 'auto',
  status: 'completed',
  model: 'gpt-6-astra',
  target_model: 'gpt-6-astra',
  rounds: 1,
  completed_rounds: 1,
  result: 'normal',
  winner: 'gpt-6-astra',
  probabilities: { 'gpt-6-astra': 0.95 },
  version: 'v1',
  created_at: '2026-10-23T00:00:00Z',
  started_at: '2026-10-23T00:00:01Z',
  finished_at: '2026-10-23T00:00:05Z',
  duration_ms: 4000,
  error: ''
}

describe('ModelTraceTaskDetails', () => {
  it('does not show accepted extra hit badge when winner equals target model', () => {
    const w = mount(ModelTraceTaskDetails, {
      props: {
        task: {
          ...baseTask,
          target_model: 'gpt-6-astra',
          winner: 'gpt-6-astra',
          accepted_models: ['gpt-6-sol']
        }
      }
    })
    expect(w.find('[data-testid="modeltrace-accepted-extra-hit"]').exists()).toBe(false)
    expect(w.text()).not.toContain('admin.modeltrace.acceptedExtraHit')
    expect(w.text()).toContain('admin.modeltrace.acceptedModelsSnapshot')
    expect(w.text()).toContain('gpt-6-sol')
  })

  it('shows accepted extra hit badge when completed normal task winner != target and accepted_models contains winner', () => {
    const w = mount(ModelTraceTaskDetails, {
      props: {
        task: {
          ...baseTask,
          target_model: 'gpt-6-astra',
          winner: 'gpt-6-sol',
          accepted_models: ['gpt-6-sol', 'gpt-5.6-sol']
        }
      }
    })
    expect(w.find('[data-testid="modeltrace-accepted-extra-hit"]').exists()).toBe(true)
    expect(w.text()).toContain('admin.modeltrace.acceptedExtraHit')
  })

  it('does not show accepted extra hit badge when task is degraded even if accepted_models contains winner', () => {
    const w = mount(ModelTraceTaskDetails, {
      props: {
        task: {
          ...baseTask,
          result: 'degraded',
          target_model: 'gpt-6-astra',
          winner: 'gpt-6-sol',
          accepted_models: ['gpt-6-sol']
        }
      }
    })
    expect(w.find('[data-testid="modeltrace-accepted-extra-hit"]').exists()).toBe(false)
  })

  it('does not show accepted extra hit badge when winner is not in accepted_models', () => {
    const w = mount(ModelTraceTaskDetails, {
      props: {
        task: {
          ...baseTask,
          target_model: 'gpt-6-astra',
          winner: 'gpt-6-sol',
          accepted_models: ['gpt-5.6-sol']
        }
      }
    })
    expect(w.find('[data-testid="modeltrace-accepted-extra-hit"]').exists()).toBe(false)
  })

  it('handles legacy tasks without accepted_models gracefully', () => {
    const w = mount(ModelTraceTaskDetails, {
      props: {
        task: {
          ...baseTask,
          target_model: 'gpt-6-astra',
          winner: 'gpt-6-sol',
          accepted_models: undefined
        }
      }
    })
    expect(w.find('[data-testid="modeltrace-accepted-extra-hit"]').exists()).toBe(false)
    expect(w.text()).not.toContain('admin.modeltrace.acceptedModelsSnapshot')
  })
})
