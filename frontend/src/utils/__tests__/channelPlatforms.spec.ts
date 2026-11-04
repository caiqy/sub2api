import { describe, expect, it } from 'vitest'
import { resolveActiveChannelPlatforms } from '@/utils/channelPlatforms'
import type { GroupPlatform } from '@/types'

const concrete: GroupPlatform[] = ['anthropic', 'openai', 'gemini', 'grok', 'deepseek', 'opencode_go']

function groupPlatforms(entries: Record<number, GroupPlatform>): Map<number, GroupPlatform> {
  return new Map(Object.entries(entries).map(([id, platform]) => [Number(id), platform]))
}

describe('resolveActiveChannelPlatforms', () => {
  it('展开全部平台：只有组合分组且没有任何平台级配置（存量兼容）', () => {
    const active = resolveActiveChannelPlatforms(
      { group_ids: [35, 42, 47], model_pricing: [], model_mapping: {} },
      groupPlatforms({ 35: 'composite', 42: 'composite', 47: 'composite' }),
      concrete
    )
    expect([...active].sort()).toEqual([...concrete].sort())
  })

  it('只保留配置过的平台：组合分组 + 具体分组 + 定价', () => {
    const active = resolveActiveChannelPlatforms(
      {
        group_ids: [35, 41, 42, 47],
        model_pricing: [{ platform: 'opencode_go' }],
        model_mapping: {}
      },
      groupPlatforms({ 35: 'composite', 41: 'opencode_go', 42: 'composite', 47: 'composite' }),
      concrete
    )
    expect([...active]).toEqual(['opencode_go'])
  })

  it('映射与账号统计规则里的平台同样算作已配置', () => {
    const active = resolveActiveChannelPlatforms(
      {
        group_ids: [35],
        model_pricing: [{ platform: 'openai' }],
        model_mapping: { anthropic: { 'claude-sonnet-4': 'claude-sonnet-4' } },
        account_stats_pricing_rules: [{ pricing: [{ platform: 'gemini' }] }]
      },
      groupPlatforms({ 35: 'composite' }),
      concrete
    )
    expect([...active].sort()).toEqual(['anthropic', 'gemini', 'openai'])
  })

  it('没有组合分组时不会再展开全部平台', () => {
    const active = resolveActiveChannelPlatforms(
      { group_ids: [13], model_pricing: [], model_mapping: {} },
      groupPlatforms({ 13: 'openai' }),
      concrete
    )
    expect([...active]).toEqual(['openai'])
  })

  it('忽略未知平台与未知分组', () => {
    const active = resolveActiveChannelPlatforms(
      { group_ids: [999], model_pricing: [{ platform: 'unknown' }], model_mapping: { unknown: {} } },
      groupPlatforms({}),
      concrete
    )
    expect(active.has('unknown' as GroupPlatform)).toBe(false)
    expect(active.size).toBe(0)
  })
})
