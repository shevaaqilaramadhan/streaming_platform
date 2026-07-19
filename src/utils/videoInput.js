/**
 * Validates whether a string is a plausible video input.
 * Accepts: full URLs, YouTube IDs (11 chars), or domain-like strings.
 * @param {string} value
 * @returns {boolean}
 */
export function isValidVideoInput(value) {
  if (!value || typeof value !== 'string') return false
  const v = value.trim()
  if (!v) return false
  try { new URL(v); return true } catch {}
  if (/^[a-zA-Z0-9_-]{11}$/.test(v)) return true
  if (/^[\w.-]+\.\w+/.test(v)) return true
  return false
}
