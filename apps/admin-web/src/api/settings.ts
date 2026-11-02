import { http } from '@/api/client'

export interface SystemSettings {
  base_url: string
  api_key_configured: boolean
  weknora_tenant_id?: string
  admin_email?: string
  admin_password_configured?: boolean
  reachable: boolean
  version?: string
  updated_at: string
  updater: string
}

export function getSystemSettings() {
  return http.get<SystemSettings>('/api/v1/system/settings')
}

export function updateSystemSettings(body: {
  base_url: string
  api_key?: string
  weknora_tenant_id?: string
  admin_email?: string
  admin_password?: string
}) {
  return http.put<SystemSettings>('/api/v1/system/settings', body)
}

export function ensureWeKnoraPrincipal() {
  return http.post<{ mode: string }>('/api/v1/system/settings/ensure-principal')
}
