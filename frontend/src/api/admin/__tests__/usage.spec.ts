import { readFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { expectTypeOf } from 'vitest'

import { apiClient } from '@/api/client'
import type { AdminUsageDetail, AdminUsageQueryParams } from '@/types'

const currentDir = dirname(fileURLToPath(import.meta.url))
const usageModulePath = resolve(currentDir, '../usage.ts')

describe('admin usage module', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('uses shared admin usage types from @/types instead of redefining cleanup types locally', async () => {
    const source = await readFile(usageModulePath, 'utf8')

    expect(source).not.toMatch(/export interface UsageCleanupFilters/)
    expect(source).not.toMatch(/export interface UsageCleanupTask/)
    expect(source).not.toMatch(/export interface AdminUsageQueryParams/)
    expect(source).toMatch(/import type\s*\{[^}]*AdminUsageQueryParams[^}]*UsageCleanupTask[^}]*\}\s*from\s*'@\/types'/s)
  })

  it('exposes usage.getDetail from the real admin barrel export', async () => {
    const getSpy = vi.spyOn(apiClient, 'get').mockResolvedValue({
      data: {
        usage_log_id: 42,
        request_headers: null,
        request_body: null,
        upstream_request_headers: null,
        upstream_request_body: null,
        response_headers: null,
        response_body: null,
        created_at: '2026-03-20T00:00:00Z'
      }
    })

    const { adminAPI, usageAPI } = await import('@/api/admin')

    expect(adminAPI.usage.getDetail).toBe(usageAPI.getDetail)

    await adminAPI.usage.getDetail(42)

    expect(getSpy).toHaveBeenCalledWith('/admin/usage/42/detail')
  })

  it('includes upstream request detail fields in the shared admin usage detail type', () => {
    expectTypeOf<AdminUsageDetail>().toMatchTypeOf<{
      upstream_request_headers: string | null
      upstream_request_body: string | null
    }>()
  })

  it('keeps upstream usage list filter fields in shared query types', () => {
    expectTypeOf<AdminUsageQueryParams>().toMatchTypeOf<{
      billing_mode?: string
      exact_total?: boolean
      sort_by?: string
      sort_order?: 'asc' | 'desc'
    }>()
  })
})
