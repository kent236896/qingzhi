import { http } from '@/api/client'

export interface MemoryConfig {
  enabled: boolean
  write_mode: 'explicit_only' | 'auto' | string
  extract_model_id: string
  max_items: number
  extract_delay_seconds: number
  extract_min_interval_seconds: number
  extract_instructions: string
  interest_threshold: number
  embedding_model_id: string
  vector_recall: boolean
  retrieval_conditioning: boolean
}

export interface PersonalMemoryStatus {
  verified: boolean
  ready: boolean
  reason: string
}

export function getMemoryConfig() {
  return http.get<MemoryConfig>('/api/v1/weknora/memory-config')
}

export function updateMemoryConfig(body: MemoryConfig) {
  return http.put<MemoryConfig>('/api/v1/weknora/memory-config', body)
}

export function getPersonalMemoryStatus() {
  return http.get<PersonalMemoryStatus>('/api/v1/weknora/memory/personal-status')
}
