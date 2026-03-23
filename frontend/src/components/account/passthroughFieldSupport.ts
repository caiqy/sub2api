export function supportsPassthroughFields(params: {
  platform?: string | null
  type?: string | null
}) {
  return params.type === 'apikey'
}

export function getDefaultBaseUrl(platform?: string | null) {
  if (platform === 'openai' || platform === 'sora') return 'https://api.openai.com'
  if (platform === 'gemini') return 'https://generativelanguage.googleapis.com'
  if (platform === 'antigravity') return 'https://cloudcode-pa.googleapis.com'
  return 'https://api.anthropic.com'
}
