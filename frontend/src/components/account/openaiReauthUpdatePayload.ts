import type { OpenAITokenInfo } from '@/composables/useOpenAIOAuth'
import type { Account, UpdateAccountRequest } from '@/types'

type AccountConfigSnapshot = Pick<Account, 'credentials' | 'extra'>

type BuildCredentials = (tokenInfo: OpenAITokenInfo) => Record<string, unknown>
type BuildExtraInfo = (tokenInfo: OpenAITokenInfo) => Record<string, string> | undefined

export function buildOpenAIReauthUpdatePayload(
  account: AccountConfigSnapshot,
  tokenInfo: OpenAITokenInfo,
  buildCredentials: BuildCredentials,
  buildExtraInfo: BuildExtraInfo
): UpdateAccountRequest {
  const currentCredentials = (account.credentials as Record<string, unknown> | undefined) ?? {}
  const currentExtra = (account.extra as Record<string, unknown> | undefined) ?? {}

  const nextCredentials = {
    ...currentCredentials,
    ...buildCredentials(tokenInfo)
  }

  const nextExtra = {
    ...currentExtra,
    ...(buildExtraInfo(tokenInfo) ?? {})
  }

  return {
    type: 'oauth',
    credentials: nextCredentials,
    extra: nextExtra
  }
}
