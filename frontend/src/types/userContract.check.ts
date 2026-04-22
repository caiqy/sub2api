import type {
  AdminBindAuthIdentityRequest,
  AdminBoundAuthIdentity,
  create,
} from '@/api/admin/users'
import type { AdminUser, User } from '@/types'

type Assert<T extends true> = T
type IsExact<T, U> = (
  (<G>() => G extends T ? 1 : 2) extends (<G>() => G extends U ? 1 : 2)
    ? ((<G>() => G extends U ? 1 : 2) extends (<G>() => G extends T ? 1 : 2) ? true : false)
    : false
)

type CreateUserRequest = Parameters<typeof create>[0]

type ExpectedAdminBindAuthIdentityRequest = {
  provider_type: string
  provider_key: string
  provider_subject: string
  issuer?: string | null
  metadata?: Record<string, unknown> | null
  channel?: {
    channel: string
    channel_app_id: string
    channel_subject: string
    metadata?: Record<string, unknown> | null
  }
}

type ExpectedAdminBoundAuthIdentity = {
  user_id: number
  provider_type: string
  provider_key: string
  provider_subject: string
  verified_at?: string | null
  issuer?: string | null
  metadata: Record<string, unknown> | null
  created_at: string
  updated_at: string
  channel?: {
    channel: string
    channel_app_id: string
    channel_subject: string
    metadata: Record<string, unknown> | null
    created_at: string
    updated_at: string
  } | null
}

type ExpectedCreateUserRequest = {
  email: string
  password: string
  username?: string
  notes?: string
  balance?: number
  concurrency?: number
  allowed_groups?: number[] | null
}

const userBalanceNotifyThresholdTypeExact: Assert<
  IsExact<User['balance_notify_threshold_type'], string>
> = true

const userTotalRechargedExact: Assert<
  IsExact<User['total_recharged'], number>
> = true

const adminUserBalanceNotifyThresholdTypeExact: Assert<
  IsExact<AdminUser['balance_notify_threshold_type'], string>
> = true

const adminUserTotalRechargedExact: Assert<
  IsExact<AdminUser['total_recharged'], number>
> = true

const bindUserAuthIdentityRequestExact: Assert<
  IsExact<AdminBindAuthIdentityRequest, ExpectedAdminBindAuthIdentityRequest>
> = true

const bindUserAuthIdentityResponseExact: Assert<
  IsExact<AdminBoundAuthIdentity, ExpectedAdminBoundAuthIdentity>
> = true

const createUserRequestExact: Assert<
  IsExact<CreateUserRequest, ExpectedCreateUserRequest>
> = true

export const userContractChecks = {
  userBalanceNotifyThresholdTypeExact,
  userTotalRechargedExact,
  adminUserBalanceNotifyThresholdTypeExact,
  adminUserTotalRechargedExact,
  bindUserAuthIdentityRequestExact,
  bindUserAuthIdentityResponseExact,
  createUserRequestExact,
}
