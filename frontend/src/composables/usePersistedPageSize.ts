import { getConfiguredTableDefaultPageSize, normalizeTablePageSize } from '@/utils/tablePreferences'

const STORAGE_KEY = 'table-page-size'
const SOURCE_KEY = 'table-page-size-source'

export function getPersistedPageSize(fallback = getConfiguredTableDefaultPageSize()): number {
  if (typeof window !== 'undefined') {
    try {
      // `table-page-size-source` belongs to the old persistence scheme and can
      // leave stale user values behind after system defaults change.
      const stored = window.localStorage.getItem(SOURCE_KEY) === null
        ? window.localStorage.getItem(STORAGE_KEY)
        : null
      if (stored !== null) {
        const parsed = Number(stored)
        if (Number.isFinite(parsed)) {
          return normalizeTablePageSize(parsed)
        }
      }
    } catch (error) {
      console.warn('Failed to read persisted page size:', error)
    }
  }
  return normalizeTablePageSize(getConfiguredTableDefaultPageSize() || fallback)
}

export function setPersistedPageSize(size: number): void {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(STORAGE_KEY, String(size))
    window.localStorage.removeItem(SOURCE_KEY)
  } catch (error) {
    console.warn('Failed to persist page size:', error)
  }
}
