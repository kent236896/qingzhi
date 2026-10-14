import type { HealthLive, HealthReady } from '@/types/api'
import { http } from './client'

export function getHealthLive() {
  return http.get<HealthLive>('/api/v1/health/live', { skipErrorToast: true })
}

export function getHealthReady() {
  return http.get<HealthReady>('/api/v1/health/ready', { skipErrorToast: true })
}
