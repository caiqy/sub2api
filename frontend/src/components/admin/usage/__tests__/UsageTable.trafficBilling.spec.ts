import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import UsageTable from '../UsageTable.vue'
import { computeTooltipPosition } from '@/utils/tooltipPosition'
import type { AdminUsageLog } from '@/types'

vi.mock('@/utils/ipGeoLookup', () => ({
  getEntry: vi.fn(() => ({ status: 'idle' as const })),
  fetchOne: vi.fn(),
  fetchBatch: vi.fn()
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: vi.fn(),
    showError: vi.fn()
  })
}))

const messages: Record<string, string> = {
  'usage.costDetails': '费用明细',
  'usage.trafficCost': '流量费',
  'usage.trafficTooltip': '包含接收和发送双边流量',
  'usage.trafficReasonDeliveryIncomplete': '请求未完整交付（免计费）',
  'usage.trafficReasonConsentRequired': '未开启超大请求授权',
  'usage.trafficReasonBelowThreshold': '未超出免费门槛',
  'usage.trafficReasonDisabled': '流量计费功能未启用',
  'usage.accountBilled': '账号计费',
  'usage.serviceTier': '服务档位',
  'usage.rate': '倍率',
  'usage.original': '原价',
  'usage.userBilled': '用户扣费'
}

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] || key
    })
  }
})

const baseLog: AdminUsageLog = {
  id: 1,
  user_id: 1,
  api_key_id: 1,
  account_id: 1,
  request_id: 'req_123',
  model: 'gpt-4o',
  input_tokens: 100,
  output_tokens: 100,
  cache_creation_tokens: 0,
  cache_read_tokens: 0,
  cache_creation_5m_tokens: 0,
  cache_creation_1h_tokens: 0,
  input_cost: 0.001,
  output_cost: 0.002,
  cache_creation_cost: 0,
  cache_read_cost: 0,
  total_cost: 0.243,
  actual_cost: 0.243,
  rate_multiplier: 1,
  long_context_billing_applied: false,
  billing_type: 1,
  stream: false,
  native_compaction_v2: false,
  duration_ms: 500,
  first_token_ms: 100,
  image_count: 0,
  image_size: null,
  image_input_size: null,
  image_output_size: null,
  image_size_source: null,
  image_size_breakdown: null,
  image_input_tokens: 0,
  image_input_cost: 0,
  image_output_tokens: 0,
  image_output_cost: 0,
  user_agent: 'test',
  cache_ttl_overridden: false,
  created_at: '2026-11-08T12:00:00Z',
  has_detail: true,
  account_rate_multiplier: 1.5,
  // Traffic billing fields
  traffic_cost: 0.24,
  traffic_input_bytes: 12000000,
  traffic_threshold_bytes: 10000000,
  traffic_price_per_mb: 0.01,
  traffic_billing_reason: ''
}

const DataTableStub = {
  props: ['data', 'columns'],
  template: `
    <div>
      <div v-for="row in data" :key="row.request_id">
        <slot name="cell-cost" :row="row" />
      </div>
    </div>
  `
}

describe('UsageTable Traffic Billing and Account Isolation', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('isolates traffic_cost from account billed cost in table cell and tooltip', async () => {
    // total_cost is 0.243, traffic_cost is 0.24.
    // Pure model cost = 0.243 - 0.24 = 0.003
    // account_rate_multiplier = 1.5 => 0.003 * 1.5 = 0.004500
    const wrapper = mount(UsageTable, {
      props: {
        data: [baseLog],
        columns: [{ key: 'cost', label: 'Cost' }],
        showAccountBilling: true
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    const costCell = wrapper.find('[data-testid="cost-detail-trigger"]')
    expect(costCell.exists()).toBe(true)

    // Cell should show account billed = 0.004500, NOT (0.243 * 1.5 = 0.364500)
    expect(wrapper.text()).toContain('A $0.004500')

    // Open tooltip
    await costCell.trigger('mouseenter')
    await nextTick()

    // Tooltip should contain traffic cost line
    expect(wrapper.text()).toContain('流量费')
    expect(wrapper.text()).toContain('$0.24000000')
    expect(wrapper.text()).toContain('12,000,000 B (12 MB)')
    expect(wrapper.text()).toContain('12 MB × 2 × $0.01/MB')
    // Normal charged reason should not be shown as raw text
    expect(wrapper.text()).not.toContain('charged')

    // Account billed in tooltip should also be isolated (0.00450000)
    expect(wrapper.text()).toContain('$0.00450000')
  })

  it('renders question mark with accessible hover, focus and exact tooltip text', async () => {
    const wrapper = mount(UsageTable, {
      props: {
        data: [baseLog],
        columns: [{ key: 'cost', label: 'Cost' }],
        showAccountBilling: true
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    const costCell = wrapper.find('[data-testid="cost-detail-trigger"]')
    await costCell.trigger('mouseenter')
    await nextTick()

    const trigger = wrapper.find('[data-testid="traffic-tooltip-trigger"]')
    expect(trigger.exists()).toBe(true)
    expect(trigger.attributes('aria-label')).toBe('包含接收和发送双边流量')
    expect(trigger.attributes('title')).toBe('包含接收和发送双边流量')

    // Focus on the question mark button
    await trigger.trigger('focus')
    await nextTick()

    const bubble = wrapper.find('[data-testid="traffic-question-tooltip"]')
    expect(bubble.exists()).toBe(true)
    expect(bubble.text()).toBe('包含接收和发送双边流量')
  })

  it('keeps tooltip open when focus transitions from cost trigger to traffic question mark and past 180ms', async () => {
    vi.useFakeTimers()
    const wrapper = mount(UsageTable, {
      props: {
        data: [baseLog],
        columns: [{ key: 'cost', label: 'Cost' }],
        showAccountBilling: true
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    const costCell = wrapper.find('[data-testid="cost-detail-trigger"]')
    // 1. Focus cost trigger
    await costCell.trigger('focus')
    await nextTick()

    const trigger = wrapper.find('[data-testid="traffic-tooltip-trigger"]')
    expect(trigger.exists()).toBe(true)

    // 2. Blur cost trigger (which schedules hide after 180ms)
    await costCell.trigger('blur')
    // 3. Focus traffic question button in portal
    await trigger.trigger('focus')
    await nextTick()

    // 4. Advance timers past 180ms
    vi.advanceTimersByTime(250)
    await nextTick()

    // Tooltip and question mark tooltip must remain visible!
    const bubble = wrapper.find('[data-testid="traffic-question-tooltip"]')
    expect(bubble.exists()).toBe(true)
    expect(bubble.text()).toBe('包含接收和发送双边流量')
    expect(wrapper.text()).toContain('流量费')

    // 5. Blur traffic question mark button and trigger focusout on portal
    await trigger.trigger('blur')
    const portal = wrapper.find('.pointer-events-auto')
    if (portal.exists()) {
      await portal.trigger('focusout')
    }
    vi.advanceTimersByTime(250)
    await nextTick()

    expect(wrapper.find('[data-testid="traffic-tooltip-trigger"]').exists()).toBe(false)
    vi.useRealTimers()
  })

  it('account_stats_cost takes precedence over total_cost - traffic_cost fallback', async () => {
    const customLog: AdminUsageLog = {
      ...baseLog,
      account_stats_cost: 0.005
    }
    // With account_stats_cost = 0.005, result should be 0.005 * 1.5 = 0.007500
    const wrapper = mount(UsageTable, {
      props: {
        data: [customLog],
        columns: [{ key: 'cost', label: 'Cost' }],
        showAccountBilling: true
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    expect(wrapper.text()).toContain('A $0.007500')
  })

  it('correctly displays localized reason for delivery_incomplete without misleading free label', async () => {
    const incompleteLog: AdminUsageLog = {
      ...baseLog,
      traffic_cost: 0,
      traffic_input_bytes: 15_000_000,
      traffic_threshold_bytes: 10_000_000,
      traffic_billing_reason: 'delivery_incomplete'
    }
    const wrapper = mount(UsageTable, {
      props: {
        data: [incompleteLog],
        columns: [{ key: 'cost', label: 'Cost' }],
        showAccountBilling: true
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    const costCell = wrapper.find('[data-testid="cost-detail-trigger"]')
    await costCell.trigger('mouseenter')
    await nextTick()

    // Must show localized reason
    expect(wrapper.text()).toContain('请求未完整交付（免计费）')
    // Must NOT display "免费" since S > T and it was waived due to delivery incomplete
    expect(wrapper.text()).not.toContain('(免费)')
  })

  it('displays free threshold label when reason is below_threshold', async () => {
    const belowThresholdLog: AdminUsageLog = {
      ...baseLog,
      traffic_cost: 0,
      traffic_input_bytes: 5_000_000,
      traffic_threshold_bytes: 10_000_000,
      traffic_billing_reason: 'below_threshold'
    }
    const wrapper = mount(UsageTable, {
      props: {
        data: [belowThresholdLog],
        columns: [{ key: 'cost', label: 'Cost' }],
        showAccountBilling: true
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    const costCell = wrapper.find('[data-testid="cost-detail-trigger"]')
    await costCell.trigger('mouseenter')
    await nextTick()

    expect(wrapper.text()).toContain('≤ 10 MB (免费)')
  })

  it('touch/click synthetic event sequence: cost focus->click opens AND question focus->click still shows tooltip', async () => {
    vi.useFakeTimers()
    const wrapper = mount(UsageTable, {
      props: {
        data: [baseLog],
        columns: [{ key: 'cost', label: 'Cost' }],
        showAccountBilling: true
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    const costCell = wrapper.find('[data-testid="cost-detail-trigger"]')

    // 1. Native touch on cost button fires focus then click
    await costCell.trigger('focus')
    await costCell.trigger('click')
    await nextTick()

    const questionBtn = wrapper.find('[data-testid="traffic-tooltip-trigger"]')
    expect(questionBtn.exists()).toBe(true)

    // 2. Native touch on question mark button fires focus then click
    await questionBtn.trigger('focus')
    await questionBtn.trigger('click')
    await nextTick()

    // Must still show tooltip bubble (not toggled closed by click after focus!)
    const bubble = wrapper.find('[data-testid="traffic-question-tooltip"]')
    expect(bubble.exists()).toBe(true)
    expect(bubble.text()).toBe('包含接收和发送双边流量')

    // 3. Escape key immediately closes the tooltip
    await questionBtn.trigger('keydown', { key: 'Escape' })
    await nextTick()
    expect(wrapper.find('[data-testid="traffic-tooltip-trigger"]').exists()).toBe(false)

    // 4. Mouseleave scheduleHide exit behavior
    await costCell.trigger('mouseenter')
    await nextTick()
    expect(wrapper.find('[data-testid="traffic-tooltip-trigger"]').exists()).toBe(true)

    await costCell.trigger('mouseleave')
    vi.advanceTimersByTime(250)
    await nextTick()
    expect(wrapper.find('[data-testid="traffic-tooltip-trigger"]').exists()).toBe(false)

    // 5. Blur scheduleHide exit behavior
    await costCell.trigger('focus')
    await nextTick()
    expect(wrapper.find('[data-testid="traffic-tooltip-trigger"]').exists()).toBe(true)

    await costCell.trigger('blur')
    vi.advanceTimersByTime(250)
    await nextTick()
    expect(wrapper.find('[data-testid="traffic-tooltip-trigger"]').exists()).toBe(false)

    vi.useRealTimers()
  })

  describe('small screen viewport boundary positioning (A12)', () => {
    it('computeTooltipPosition preserves desktop right-placement when sufficient width is available', () => {
      const target = { left: 740, right: 756, top: 300, bottom: 316, width: 16, height: 16 }
      const panel = { width: 224, height: 260 }
      const viewport = { width: 1920, height: 1080 }

      const pos = computeTooltipPosition(target, panel, viewport)
      expect(pos.x).toBe(756 + 8) // 764
      expect(pos.y).toBe(300 + 8) // 308
    })

    it('computeTooltipPosition clamps to left on small screen (iPhone 14 390x844) so panel and question button remain on-screen', () => {
      // Button at x=340.8, w=16 (right=356.8), viewport=390
      const target = { left: 340.8, right: 356.8, top: 542.2, bottom: 558.2, width: 16, height: 16 }
      const panel = { width: 224, height: 260 }
      const viewport = { width: 390, height: 844 }

      const pos = computeTooltipPosition(target, panel, viewport)
      // Default right placement would be 356.8 + 8 = 364.8, which with width 224 overflows to 588.8 > 390
      // Switched to left placement: 340.8 - 224 - 8 = 108.8
      expect(pos.x).toBeCloseTo(108.8, 1)
      // Panel right edge must be within viewport width with margin (390 - 12 = 378)
      expect(pos.x + panel.width).toBeLessThanOrEqual(390 - 12)
      // Question mark (approx offset ~52.8px inside panel) must be on-screen
      const questionMarkX = pos.x + 52.8
      expect(questionMarkX).toBeGreaterThan(12)
      expect(questionMarkX).toBeLessThan(390 - 12)
    })

    it('computeTooltipPosition clamps within viewport margins when neither left nor right placement fits', () => {
      // Extremely narrow 300px screen, target centered at 140..156
      const target = { left: 140, right: 156, top: 200, bottom: 216, width: 16, height: 16 }
      const panel = { width: 220, height: 200 }
      const viewport = { width: 300, height: 600 }

      const pos = computeTooltipPosition(target, panel, viewport)
      // Must clamp within [12, 300 - 220 - 12 = 68]
      expect(pos.x).toBeGreaterThanOrEqual(12)
      expect(pos.x + panel.width).toBeLessThanOrEqual(300 - 12)
    })

    it('computeTooltipPosition clamps vertically when target is near top or bottom', () => {
      const panel = { width: 224, height: 260 }
      const viewport = { width: 390, height: 800 }

      // Near top: target top = 10
      const topTarget = { left: 100, right: 116, top: 10, bottom: 26, width: 16, height: 16 }
      const topPos = computeTooltipPosition(topTarget, panel, viewport)
      // Panel actual top is pos.y - panel.height / 2, must be >= 12
      expect(topPos.y - panel.height / 2).toBeGreaterThanOrEqual(12)

      // Near bottom: target top = 780
      const bottomTarget = { left: 100, right: 116, top: 780, bottom: 796, width: 16, height: 16 }
      const bottomPos = computeTooltipPosition(bottomTarget, panel, viewport)
      // Panel actual bottom is pos.y + panel.height / 2, must be <= 800 - 12 = 788
      expect(bottomPos.y + panel.height / 2).toBeLessThanOrEqual(800 - 12)
    })

    it('UsageTable on iPhone 14 viewport positions portal so question trigger is not overflowed off-screen', async () => {
      const originalInnerWidth = window.innerWidth
      const originalInnerHeight = window.innerHeight
      Object.defineProperty(window, 'innerWidth', { writable: true, configurable: true, value: 390 })
      Object.defineProperty(window, 'innerHeight', { writable: true, configurable: true, value: 844 })

      const wrapper = mount(UsageTable, {
        props: {
          data: [baseLog],
          columns: [{ key: 'cost', label: 'Cost' }],
          showAccountBilling: true
        },
        global: {
          stubs: {
            DataTable: DataTableStub,
            EmptyState: true,
            Icon: true,
            Teleport: true
          }
        }
      })

      const costBtn = wrapper.find('[data-testid="cost-detail-trigger"]')
      // Mock cost button bounding rect matching iPhone 14 measurements: x=340.8, w=16
      costBtn.element.getBoundingClientRect = () => ({
        x: 340.8,
        y: 542.2,
        width: 16,
        height: 16,
        top: 542.2,
        right: 356.8,
        bottom: 558.2,
        left: 340.8,
        toJSON: () => {}
      })

      await costBtn.trigger('click')
      await nextTick()

      // The portal container must exist and have style left <= 390 - 224 - 12
      const portal = wrapper.find('.fixed.z-\\[9999\\]')
      expect(portal.exists()).toBe(true)
      const leftStyle = portal.attributes('style')
      const leftMatch = leftStyle?.match(/left:\s*([0-9.]+)px/)
      expect(leftMatch).toBeTruthy()
      const leftVal = parseFloat(leftMatch![1])
      // In iPhone 14 with button at right, left must be clamped to the left of the button (~108.8px)
      // and NOT placed at 364.8px which would overflow 390px
      expect(leftVal).toBeLessThan(200)
      expect(leftVal).toBeGreaterThanOrEqual(12)

      // Ensure question mark trigger exists within the clamped portal
      const questionTrigger = wrapper.find('[data-testid="traffic-tooltip-trigger"]')
      expect(questionTrigger.exists()).toBe(true)

      Object.defineProperty(window, 'innerWidth', { writable: true, configurable: true, value: originalInnerWidth })
      Object.defineProperty(window, 'innerHeight', { writable: true, configurable: true, value: originalInnerHeight })
    })
  })
})
