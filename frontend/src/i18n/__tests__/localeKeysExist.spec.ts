import { readdirSync, readFileSync } from 'node:fs'
import { extname, join } from 'node:path'
import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const srcDir = join(process.cwd(), 'src')

function sourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name)
    if (entry.isDirectory()) {
      return entry.name === '__tests__' || path.includes(join('i18n', 'locales')) ? [] : sourceFiles(path)
    }
    return ['.ts', '.vue'].includes(extname(entry.name)) ? [path] : []
  })
}

function staticLocaleKeys(): string[] {
  const keys = new Set<string>()
  const callPattern = /(?:^|[^\w$])\$?t\(\s*(['"])([^'"]+)\1/g

  for (const file of sourceFiles(srcDir)) {
    const source = readFileSync(file, 'utf8')
    for (const match of source.matchAll(callPattern)) keys.add(match[2])
  }

  return [...keys].filter((key) => !key.endsWith('.')).sort()
}

function hasKey(messages: Record<string, unknown>, key: string): boolean {
  let value: unknown = messages
  for (const segment of key.split('.')) {
    if (!value || typeof value !== 'object' || !(segment in value)) return false
    value = (value as Record<string, unknown>)[segment]
  }
  return true
}

describe.each([['en', en], ['zh', zh]] as const)('%s locale', (_locale, messages) => {
  it('defines every statically referenced translation key', () => {
    expect(staticLocaleKeys().filter((key) => !hasKey(messages, key))).toEqual([])
  })
})
