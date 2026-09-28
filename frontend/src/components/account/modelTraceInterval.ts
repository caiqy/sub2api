export function parseModelTraceInterval(value: string): number | null | undefined {
  const input = value.trim()
  if (!input) return null
  const minutes = Number(input)
  return /^\d+$/.test(input) && Number.isInteger(minutes) && minutes >= 5 && minutes <= 10080
    ? minutes
    : undefined
}
