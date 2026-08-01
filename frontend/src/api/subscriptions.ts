/**
 * User Subscription API
 * API for regular users to view their own subscriptions and progress
 */

import { apiClient } from './client'
import type { UserSubscription, SubscriptionProgress } from '@/types'

export interface AdvanceQuotaCycleRequest {
  daily: boolean
  weekly: boolean
  monthly: boolean
}

export interface AdvanceQuotaCycleResponse {
  subscription: UserSubscription
  deducted_seconds: number
}

const quotaAdvanceOperationKeys = new Map<string, string>()

function quotaAdvanceScope(id: number, request: AdvanceQuotaCycleRequest): string {
  return `${id}-${request.daily ? 'd' : '-'}${request.weekly ? 'w' : '-'}${request.monthly ? 'm' : '-'}`
}

/**
 * Subscription summary for user dashboard
 */
export interface SubscriptionSummary {
  active_count: number
  subscriptions: Array<{
    id: number
    group_name: string
    status: string
    daily_progress: number | null
    weekly_progress: number | null
    monthly_progress: number | null
    expires_at: string | null
    days_remaining: number | null
  }>
}

/**
 * Get list of current user's subscriptions
 */
export async function getMySubscriptions(): Promise<UserSubscription[]> {
  const response = await apiClient.get<UserSubscription[]>('/subscriptions')
  return response.data
}

/**
 * Get current user's active subscriptions
 */
export async function getActiveSubscriptions(): Promise<UserSubscription[]> {
  const response = await apiClient.get<UserSubscription[]>('/subscriptions/active')
  return response.data
}

/**
 * Get progress for all user's active subscriptions
 */
export async function getSubscriptionsProgress(): Promise<SubscriptionProgress[]> {
  const response = await apiClient.get<SubscriptionProgress[]>('/subscriptions/progress')
  return response.data
}

/**
 * Get subscription summary for dashboard display
 */
export async function getSubscriptionSummary(): Promise<SubscriptionSummary> {
  const response = await apiClient.get<SubscriptionSummary>('/subscriptions/summary')
  return response.data
}

/**
 * Get progress for a specific subscription
 */
export async function getSubscriptionProgress(
  subscriptionId: number
): Promise<SubscriptionProgress> {
  const response = await apiClient.get<SubscriptionProgress>(
    `/subscriptions/${subscriptionId}/progress`
  )
  return response.data
}

export async function advanceQuotaCycle(
  subscriptionId: number,
  request: AdvanceQuotaCycleRequest,
): Promise<AdvanceQuotaCycleResponse> {
  const scope = quotaAdvanceScope(subscriptionId, request)
  let key = quotaAdvanceOperationKeys.get(scope)
  if (!key) {
    const requestId = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
    key = `subscription-quota-advance-${scope}-${requestId}`
    quotaAdvanceOperationKeys.set(scope, key)
  }
  const response = await apiClient.post<AdvanceQuotaCycleResponse>(
    `/subscriptions/${subscriptionId}/advance-quota-cycle`,
    request,
    { headers: { 'Idempotency-Key': key } },
  )
  quotaAdvanceOperationKeys.delete(scope)
  return response.data
}

export default {
  getMySubscriptions,
  getActiveSubscriptions,
  getSubscriptionsProgress,
  getSubscriptionSummary,
  getSubscriptionProgress,
  advanceQuotaCycle,
}
