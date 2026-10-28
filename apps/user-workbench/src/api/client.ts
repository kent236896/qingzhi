import { createHttpClient } from '@/utils/http'
import { feedback } from '@/utils/feedback'

const baseURL = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '')

export const http = createHttpClient({
  baseURL,
  timeout: 30_000,
  onError: (error) => {
    feedback.error(error.message)
  },
  getAccessToken: () => localStorage.getItem('qz_wb_access_token'),
  onUnauthorized: () => {
    localStorage.removeItem('qz_wb_access_token')
    localStorage.removeItem('qz_wb_tenant_id')
    const path = window.location.pathname + window.location.search
    if (!path.startsWith('/login')) {
      const redirect = encodeURIComponent(path)
      window.location.assign(`/login?redirect=${redirect}`)
    }
  },
})
