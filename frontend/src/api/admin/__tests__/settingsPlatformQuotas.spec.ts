import { describe, expect, it } from 'vitest'
import {
  appendAuthSourceDefaultsToUpdateRequest,
  buildAuthSourceDefaultsState,
  normalizePlatformQuotasMap,
  sanitizePlatformQuotasMap,
  type DefaultPlatformQuotasMap,
  type UpdateSettingsRequest,
} from '../settings'

const platforms = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'kimi', 'zhipu', 'deepseek', 'minimax', 'opencode_go', 'typesafe']
const quotas: DefaultPlatformQuotasMap = {
  kimi: { daily: 0, weekly: null, monthly: 50 },
  zhipu: { daily: 12.5, weekly: 20, monthly: null },
  deepseek: { daily: null, weekly: 30, monthly: 60 },
  minimax: { daily: 40, weekly: null, monthly: null },
  opencode_go: { daily: null, weekly: 50, monthly: null },
  typesafe: { daily: 6, weekly: 12, monthly: 24 },
}

describe('settings TypeSafe and fork platform quota fusion', () => {
  it('round-trips all eleven global platforms without losing zero, null or domestic limits', () => {
    const normalized = normalizePlatformQuotasMap(quotas)
    expect(Object.keys(normalized)).toEqual(platforms)
    expect(sanitizePlatformQuotasMap(normalized)).toMatchObject(quotas)
    expect(normalized.openai).toEqual({ daily: null, weekly: null, monthly: null })
  })

  it.each(['email', 'linuxdo', 'oidc', 'wechat', 'github', 'google', 'dingtalk'] as const)(
    'round-trips TypeSafe and domestic limits for the %s signup source', (source) => {
      const key = `auth_source_default_${source}_platform_quotas` as const
      const state = buildAuthSourceDefaultsState({ [key]: quotas })
      expect(Object.keys(state[source].platform_quotas)).toEqual(platforms)
      const payload: UpdateSettingsRequest = {}
      appendAuthSourceDefaultsToUpdateRequest(payload, state)
      expect(payload[key]).toMatchObject(quotas)
    },
  )

  it('cleans invalid TypeSafe limits while retaining explicit disabled and unlimited limits', () => {
    const clean = sanitizePlatformQuotasMap({
      ...quotas,
      typesafe: { daily: NaN, weekly: -1, monthly: Infinity },
    })
    expect(clean.typesafe).toEqual({ daily: null, weekly: null, monthly: null })
    expect(clean.kimi).toEqual(quotas.kimi)
  })
})
