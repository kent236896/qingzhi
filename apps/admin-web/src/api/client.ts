import { feedback } from '@/utils/feedback'
import { createHttpClient } from '@/utils/http'

function timeoutMs(): number {
  const parsed = Number(import.meta.env.VITE_API_TIMEOUT_MS)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 10000
}

export const http = createHttpClient({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: timeoutMs(),
  getAccessToken() {
    return localStorage.getItem('qz_access_token')
  },
  onUnauthorized() {
    localStorage.removeItem('qz_access_token')
    localStorage.removeItem('qz_tenant_id')
    if (!window.location.pathname.startsWith('/login')) {
      const redirect = encodeURIComponent(window.location.pathname + window.location.search)
      window.location.assign(`/login?redirect=${redirect}`)
    }
  },
  onError(error) {
    feedback.error(error.message)
  },
})
