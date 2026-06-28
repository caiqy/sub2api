import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UserBreakdownSubTable from '../UserBreakdownSubTable.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => ({
        'admin.dashboard.noDataAvailable': 'No data available',
        'admin.redeem.userPrefix': `User #${params?.id}`,
      }[key] ?? key),
    }),
  }
})

describe('UserBreakdownSubTable', () => {
  it('displays username before email and falls back when username is empty', () => {
    const wrapper = mount(UserBreakdownSubTable, {
      props: {
        items: [
          {
            user_id: 1,
            email: 'alice@example.com',
            username: '  Alice  ',
            requests: 10,
            total_tokens: 1000,
            cost: 1.5,
            actual_cost: 1.2,
            account_cost: 1.1,
          },
          {
            user_id: 2,
            email: 'bob@example.com',
            username: '   ',
            requests: 5,
            total_tokens: 500,
            cost: 0.5,
            actual_cost: 0.4,
            account_cost: 0.3,
          },
          {
            user_id: 3,
            email: '   ',
            username: '',
            requests: 1,
            total_tokens: 50,
            cost: 0.05,
            actual_cost: 0.04,
            account_cost: 0.03,
          },
        ],
      },
      global: {
        stubs: {
          LoadingSpinner: true,
        },
      },
    })

    const rows = wrapper.findAll('tbody tr')
    expect(rows[0].text()).toContain('Alice')
    expect(rows[0].text()).not.toContain('alice@example.com')
    expect(rows[1].text()).toContain('bob@example.com')
    expect(rows[2].text()).toContain('User #3')

    expect(rows[0].find('td').attributes('title')).toBe('Alice')
    expect(rows[1].find('td').attributes('title')).toBe('bob@example.com')
    expect(rows[2].find('td').attributes('title')).toBe('User #3')
  })
})
