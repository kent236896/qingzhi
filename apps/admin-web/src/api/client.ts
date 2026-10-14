import { feedback } from '@/utils/feedback'
import { createHttpClient } from '@/utils/http'

function timeoutMs(): number {
  const parsed = Number(import.meta.env.VITE_API_TIMEOUT_MS)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 10000
}

export const http = createHttpClient({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: timeoutMs(),
  onError(error) {
    feedback.error(error.message)
  },
})
