import { describe, expect, it, vi } from 'vitest'

import { buildOpenAIReauthUpdatePayload } from '../openaiReauthUpdatePayload'

describe('buildOpenAIReauthUpdatePayload', () => {
  it('preserves existing credentials and extra while overwriting fresh oauth fields', () => {
    const buildCredentials = vi.fn(() => ({
      access_token: 'new-at',
      refresh_token: 'new-rt',
      expires_at: 1730000000,
      client_id: 'app_new'
    }))
    const buildExtraInfo = vi.fn(() => ({
      email: 'new@example.com',
      name: 'New Name',
      privacy_mode: 'strict'
    }))

    const payload = buildOpenAIReauthUpdatePayload(
      {
        credentials: {
          access_token: 'old-at',
          refresh_token: 'old-rt',
          model_mapping: { 'gpt-5': 'gpt-5' },
          custom_flag: true
        },
        extra: {
          passthrough_fields_enabled: true,
          passthrough_field_rules: [
            { target: 'header', mode: 'forward', key: 'x-trace-id' }
          ],
          openai_passthrough: true,
          codex_cli_only: true,
          email: 'old@example.com'
        }
      },
      {
        access_token: 'ignored-by-helper-input-shape'
      },
      buildCredentials,
      buildExtraInfo
    )

    expect(payload).toEqual({
      type: 'oauth',
      credentials: {
        access_token: 'new-at',
        refresh_token: 'new-rt',
        expires_at: 1730000000,
        client_id: 'app_new',
        model_mapping: { 'gpt-5': 'gpt-5' },
        custom_flag: true
      },
      extra: {
        passthrough_fields_enabled: true,
        passthrough_field_rules: [
          { target: 'header', mode: 'forward', key: 'x-trace-id' }
        ],
        openai_passthrough: true,
        codex_cli_only: true,
        email: 'new@example.com',
        name: 'New Name',
        privacy_mode: 'strict'
      }
    })
  })

  it('treats missing account extra and undefined oauth extra as empty objects', () => {
    const payload = buildOpenAIReauthUpdatePayload(
      {
        credentials: undefined,
        extra: undefined
      },
      {
        access_token: 'at',
        expires_at: 1730000000
      },
      () => ({ access_token: 'at', expires_at: 1730000000 }),
      () => undefined
    )

    expect(payload).toEqual({
      type: 'oauth',
      credentials: {
        access_token: 'at',
        expires_at: 1730000000
      },
      extra: {}
    })
  })

  it('keeps existing refresh token when fresh credentials omit it', () => {
    const payload = buildOpenAIReauthUpdatePayload(
      {
        credentials: {
          access_token: 'old-at',
          refresh_token: 'old-rt',
          id_token: 'old-id-token'
        },
        extra: {}
      },
      {
        access_token: 'new-at',
        expires_at: 1730000000
      },
      () => ({ access_token: 'new-at', expires_at: 1730000000 }),
      () => undefined
    )

    expect(payload.credentials).toEqual({
      access_token: 'new-at',
      refresh_token: 'old-rt',
      id_token: 'old-id-token',
      expires_at: 1730000000
    })
  })
})
