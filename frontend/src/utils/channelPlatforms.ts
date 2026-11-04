import type { GroupPlatform } from '@/types'

/** 渠道编辑表单里参与平台分区推导的字段（Channel 的超集，便于测试传入子集）。 */
export interface ChannelPlatformConfig {
  group_ids?: number[]
  model_pricing?: { platform?: string }[]
  model_mapping?: Record<string, unknown>
  account_stats_pricing_rules?: { pricing?: { platform?: string }[] }[]
}

/**
 * 推导渠道表单中应当启用的平台分区。
 *
 * 组合分组（composite）不属于任何具体平台，因此：
 * - 渠道已有平台级配置（定价 / 模型映射 / 账号统计规则 / 具体平台分组）时，只启用这些平台，
 *   否则只要渠道挂了组合分组就会把所有平台都勾上，用户无法只保留实际在用的平台；
 * - 完全没有平台级配置时才展开全部具体平台，避免打开存量渠道时一个分区都没有、
 *   保存后 group_ids 被清空导致分组脱离渠道。
 */
export function resolveActiveChannelPlatforms(
  channel: ChannelPlatformConfig,
  groupPlatforms: ReadonlyMap<number, GroupPlatform>,
  concretePlatforms: readonly GroupPlatform[]
): Set<GroupPlatform> {
  const active = new Set<GroupPlatform>()

  for (const pricing of channel.model_pricing || []) {
    const platform = pricing.platform as GroupPlatform | undefined
    if (platform && concretePlatforms.includes(platform)) active.add(platform)
  }
  for (const rule of channel.account_stats_pricing_rules || []) {
    for (const pricing of rule.pricing || []) {
      const platform = pricing.platform as GroupPlatform | undefined
      if (platform && concretePlatforms.includes(platform)) active.add(platform)
    }
  }
  for (const platform of Object.keys(channel.model_mapping || {})) {
    if (concretePlatforms.includes(platform as GroupPlatform)) active.add(platform as GroupPlatform)
  }

  let hasCompositeGroup = false
  for (const groupID of channel.group_ids || []) {
    const platform = groupPlatforms.get(groupID)
    if (platform === 'composite') {
      hasCompositeGroup = true
    } else if (platform) {
      active.add(platform)
    }
  }

  if (hasCompositeGroup && active.size === 0) {
    for (const platform of concretePlatforms) active.add(platform)
  }

  return active
}
