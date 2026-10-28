/**
 * Allow only same-origin relative paths to prevent open redirects.
 */
export function safeRedirectPath(value: unknown, fallback = '/'): string {
  if (typeof value !== 'string') {
    return fallback
  }
  const path = value.trim()
  if (!path.startsWith('/') || path.startsWith('//') || path.includes('://') || path.includes('\\')) {
    return fallback
  }
  return path
}
