import { http } from '@/api/client'
import type { MemoryItem, MemoryItemList, MemorySettings, MemoryStatus } from '@/types/memory'

export function fetchMemoryStatus() {
  return http.get<MemoryStatus>('/api/v1/memory/status')
}

export function fetchMemorySettings() {
  return http.get<MemorySettings>('/api/v1/memory/settings')
}

export function updateMemorySettings(userEnabled: boolean) {
  return http.put<MemorySettings>('/api/v1/memory/settings', { enabled: userEnabled, user_enabled: userEnabled })
}

export function listMemoryItems(params?: { page?: number; page_size?: number; type?: string; status?: string }) {
  return http.get<MemoryItemList>('/api/v1/memory/items', { params })
}

export function createMemoryItem(body: { type: string; content: string; importance?: number }) {
  return http.post<MemoryItem>('/api/v1/memory/items', body)
}

export function updateMemoryItem(id: string, body: { type: string; content: string; importance?: number }) {
  return http.put<MemoryItem>(`/api/v1/memory/items/${id}`, body)
}

export function deleteMemoryItem(id: string) {
  return http.delete<{ deleted: boolean }>(`/api/v1/memory/items/${id}`)
}

export function confirmMemoryItem(id: string) {
  return http.post<{ confirmed: boolean }>(`/api/v1/memory/items/${id}/confirm`)
}

export function rejectMemoryItem(id: string) {
  return http.post<{ rejected: boolean }>(`/api/v1/memory/items/${id}/reject`)
}
