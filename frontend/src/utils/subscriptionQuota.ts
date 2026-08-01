import type { UserSubscription } from '@/types'

const ONE_DAY_MS = 24 * 60 * 60 * 1000

export type SubscriptionQuotaWindow = 'daily' | 'weekly' | 'monthly'

export interface ExhaustedQuotaWindow {
  key: SubscriptionQuotaWindow
  remainingMs: number
  resetsAt: string
}

export interface QuotaAdvancePreview {
  deductedMs: number
  newExpiresAt: string | null
  affordable: boolean
}

export interface RemainingDurationParts {
  days: number
  hours: number
  minutes: number
}

export function isOneTimeDailyQuota(
  subscription: Pick<UserSubscription, 'starts_at' | 'expires_at'>
): boolean {
  if (!subscription.starts_at || !subscription.expires_at) return false

  const startsAt = new Date(subscription.starts_at).getTime()
  const expiresAt = new Date(subscription.expires_at).getTime()

  if (!Number.isFinite(startsAt) || !Number.isFinite(expiresAt)) return false

  return expiresAt <= startsAt + ONE_DAY_MS
}

export function getRemainingDurationParts(
  targetAt: Date | string,
  now: Date = new Date()
): RemainingDurationParts | null {
  const targetTime = targetAt instanceof Date ? targetAt.getTime() : new Date(targetAt).getTime()
  const nowTime = now.getTime()

  if (!Number.isFinite(targetTime) || !Number.isFinite(nowTime)) return null

  const diffMs = targetTime - nowTime
  if (diffMs <= 0) return null

  const totalMinutes = Math.floor(diffMs / (1000 * 60))
  const days = Math.floor(totalMinutes / (24 * 60))
  const hours = Math.floor((totalMinutes % (24 * 60)) / 60)
  const minutes = totalMinutes % 60

  return { days, hours, minutes }
}

export function getExhaustedQuotaWindows(
  subscription: UserSubscription,
  now: Date = new Date(),
): ExhaustedQuotaWindow[] {
  const nowMs = now.getTime()
  const expiresMs = subscription.expires_at ? new Date(subscription.expires_at).getTime() : NaN
  if (subscription.status !== 'active' || !Number.isFinite(nowMs) || !Number.isFinite(expiresMs) || expiresMs <= nowMs) {
    return []
  }

  const group = subscription.group
  if (!group) return []
  const configs: Array<{
    key: SubscriptionQuotaWindow
    usage: number
    limit: number | null
    start: string | null
    periodMs: number
  }> = [
    { key: 'daily', usage: subscription.daily_usage_usd, limit: group.daily_limit_usd, start: subscription.daily_window_start, periodMs: ONE_DAY_MS },
    { key: 'weekly', usage: subscription.weekly_usage_usd, limit: group.weekly_limit_usd, start: subscription.weekly_window_start, periodMs: 7 * ONE_DAY_MS },
    { key: 'monthly', usage: subscription.monthly_usage_usd, limit: group.monthly_limit_usd, start: subscription.monthly_window_start, periodMs: 30 * ONE_DAY_MS },
  ]

  return configs.flatMap((config) => {
    if (config.key === 'daily' && isOneTimeDailyQuota(subscription)) return []
    if (!config.limit || config.limit <= 0 || config.usage < config.limit || !config.start) return []
    const resetMs = effectiveWindowStartMs(subscription.starts_at, config.start, config.periodMs) + config.periodMs
    const remainingMs = resetMs - nowMs
    if (!Number.isFinite(resetMs) || remainingMs <= 0) return []
    return [{ key: config.key, remainingMs, resetsAt: new Date(resetMs).toISOString() }]
  })
}

function effectiveWindowStartMs(startsAt: string, windowStart: string, periodMs: number): number {
  const startsMs = new Date(startsAt).getTime()
  const windowMs = new Date(windowStart).getTime()
  if (!Number.isFinite(startsMs) || !Number.isFinite(windowMs)) return windowMs
  if (!isMidnightTimestamp(windowStart) || isMidnightTimestamp(startsAt)) return windowMs
  const since = windowMs - startsMs
  if (since < periodMs) return startsMs
  return startsMs + Math.floor(since / periodMs) * periodMs
}

function isMidnightTimestamp(value: string): boolean {
  const match = value.match(/T(\d{2}):(\d{2}):(\d{2})(?:\.(\d+))?/)
  return !!match && match[1] === '00' && match[2] === '00' && match[3] === '00'
    && (!match[4] || /^0+$/.test(match[4]))
}

export function getQuotaAdvancePreview(
  subscription: UserSubscription,
  now: Date = new Date(),
): QuotaAdvancePreview {
  const windows = getExhaustedQuotaWindows(subscription, now)
  const deductedMs = windows.reduce((max, window) => Math.max(max, window.remainingMs), 0)
  const expiresMs = subscription.expires_at ? new Date(subscription.expires_at).getTime() : NaN
  return {
    deductedMs,
    newExpiresAt: deductedMs > 0 && Number.isFinite(expiresMs)
      ? new Date(expiresMs - deductedMs).toISOString()
      : null,
    affordable: deductedMs > 0 && Number.isFinite(expiresMs) && expiresMs - deductedMs > now.getTime(),
  }
}
