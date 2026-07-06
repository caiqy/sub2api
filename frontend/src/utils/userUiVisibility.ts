export function customMenuResourceID(id: string): number {
  let hash = 14695981039346656037n
  for (const byte of new TextEncoder().encode(id)) {
    hash ^= BigInt(byte)
    hash = (hash * 1099511628211n) & 0xffffffffffffffffn
  }
  const resourceID = hash & 0x7fffffffffffffffn
  return Number(resourceID <= 1n ? 2n : resourceID)
}

export function isCustomMenuHidden(id: string, hiddenIDs?: string[] | null): boolean {
  return (hiddenIDs ?? []).includes(id)
}
