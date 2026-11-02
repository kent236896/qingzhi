import { http } from '@/api/client'

export type CatalogScope = 'personal' | 'tenant' | 'org_node' | 'role'

export interface CatalogItem {
  id: string
  name: string
  description: string
  scope_type: CatalogScope
  scope_id: string
  status: number
  creator_id: string
  tenant_ids: string[]
  share_user_ids?: string[]
  transport_type?: string
  creator: string
  updater: string
  create_time: string
  update_time: string
  writable?: boolean
}

export interface CatalogOptions {
  tenants: { id: string; name: string }[]
  organizations: { id: string; name: string; tenant_id?: string }[]
  roles: { id: string; name: string; tenant_id?: string }[]
  users: { id: string; name: string }[]
  can_manage_agents: boolean
  can_manage_toolbox: boolean
}

export interface ArtifactItem {
  session_id: string
  session_title?: string
  message_id: string
  index: number
  file_name: string
  file_type?: string
  file_size?: number
  created_at?: string
  version_count?: number
}

export interface ArtifactPage {
  items: ArtifactItem[]
  total: number
  page: number
  page_size: number
}

export function getCatalogOptions() {
  return http.get<CatalogOptions>('/api/v1/catalog/options')
}

export function listAgents() {
  return http.get<CatalogItem[]>('/api/v1/agents')
}

export function createAgent(body: {
  name: string
  description?: string
  scope_type: CatalogScope
  scope_id?: string
  tenant_ids: string[]
  share_user_ids?: string[]
}) {
  return http.post<CatalogItem>('/api/v1/agents', body)
}

export function deleteAgent(id: string) {
  return http.delete<{ deleted: boolean }>(`/api/v1/agents/${id}`)
}

export function listMCP() {
  return http.get<CatalogItem[]>('/api/v1/toolbox/mcp')
}

export function createMCP(body: {
  name: string
  description?: string
  scope_type: CatalogScope
  scope_id?: string
  tenant_ids: string[]
  share_user_ids?: string[]
  transport_type: 'sse' | 'http-streamable'
  url: string
}) {
  return http.post<CatalogItem>('/api/v1/toolbox/mcp', body)
}

export function deleteMCP(id: string) {
  return http.delete<{ deleted: boolean }>(`/api/v1/toolbox/mcp/${id}`)
}

export function testMCP(id: string) {
  return http.post<Record<string, unknown>>(`/api/v1/toolbox/mcp/${id}/test`)
}

export function listArtifacts(params: { keyword?: string; page?: number; page_size?: number } = {}) {
  return http.get<ArtifactPage>('/api/v1/artifacts', { params })
}

export function deleteArtifact(params: { session_id: string; message_id: string; index: number }) {
  return http.delete<{ deleted: boolean }>('/api/v1/artifacts', { params })
}
