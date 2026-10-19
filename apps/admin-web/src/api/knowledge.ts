import { http } from '@/api/client'

export interface KnowledgeBase {
  id: string
  name: string
  description: string
  scope_type: 'personal' | 'org_node' | 'position' | 'role'
  scope_id: string
  scope_name?: string
  wiki_enabled: boolean
  status: number
  creator_id: string
  tenant_ids: string[]
  share_user_ids?: string[]
  creator: string
  create_time: string
}

export interface KnowledgeFile {
  id: string
  knowledge_base_id: string
  file_name: string
  folder_path: string
  file_size: number
  parse_status: string
  error_msg: string
  create_time: string
}

export interface UploadResult {
  file_name: string
  status: 'success' | 'duplicate' | 'failed' | 'skipped'
  file_id?: string
  error?: string
}

export interface KnowledgeLimits {
  max_file_mb: number
  max_files: number
  zip_max_total_mb: number
  zip_max_entries: number
  allowed_exts: string[]
}

export interface WikiStatus {
  status: 'disabled' | 'empty' | 'generating' | 'ready' | 'timeout'
  page_count: number
  doc_total: number
  doc_terminal: number
  updated_at?: string
}

export interface WikiNode {
  slug: string
  title: string
  page_type: string
  link_count: number
}

export interface WikiGraph {
  nodes: WikiNode[]
  edges: { source: string; target: string }[]
  meta: { mode: string; total: number; returned: number; truncated: boolean }
}

export interface WikiPage {
  id: string
  slug: string
  title: string
  page_type: string
  summary: string
  content: string
}

export interface FormOptions {
  tenants: { id: string; name: string; tenant_id?: string }[]
  organizations: { id: string; name: string; tenant_id?: string }[]
  posts: { id: string; name: string; tenant_id?: string }[]
  roles: { id: string; name: string; tenant_id?: string }[]
  users: { id: string; name: string }[]
  can_manage: boolean
}

export function listKnowledgeBases(params: { page?: number; page_size?: number; keyword?: string; scope_type?: string } = {}) {
  return http.get<{ items: KnowledgeBase[]; page: number; page_size: number; total: number }>('/api/v1/knowledge-bases', { params })
}

export function getKnowledgeOptions() {
  return http.get<FormOptions>('/api/v1/knowledge-bases/options')
}

export function getKnowledgeLimits() {
  return http.get<KnowledgeLimits>('/api/v1/knowledge-bases/limits')
}

export function createKnowledgeBase(body: {
  name: string
  description?: string
  scope_type: string
  scope_id?: string
  tenant_ids: string[]
  share_user_ids?: string[]
  wiki_enabled?: boolean
}) {
  return http.post<KnowledgeBase>('/api/v1/knowledge-bases', body)
}

export function updateKnowledgeBase(
  id: string,
  body: { name?: string; description?: string; tenant_ids?: string[]; share_user_ids?: string[] },
) {
  return http.patch<KnowledgeBase>(`/api/v1/knowledge-bases/${id}`, body)
}

export function deleteKnowledgeBase(id: string) {
  return http.delete<null>(`/api/v1/knowledge-bases/${id}`)
}

export function getKnowledgeBase(id: string) {
  return http.get<KnowledgeBase>(`/api/v1/knowledge-bases/${id}`)
}

export function listKnowledgeFiles(id: string, sync = false) {
  return http.get<KnowledgeFile[]>(`/api/v1/knowledge-bases/${id}/files`, { params: sync ? { sync: 1 } : {} })
}

export function uploadKnowledgeFiles(id: string, files: File[]) {
  const body = new FormData()
  for (const file of files) {
    body.append('file', file)
  }
  return http.post<UploadResult[]>(`/api/v1/knowledge-bases/${id}/files`, body, { timeout: 120000 })
}

export function uploadKnowledgeZip(id: string, file: File) {
  const body = new FormData()
  body.append('file', file)
  return http.post<UploadResult[]>(`/api/v1/knowledge-bases/${id}/files/zip`, body, { timeout: 120000 })
}

export function deleteKnowledgeFiles(id: string, ids: string[]) {
  return http.post<null>(`/api/v1/knowledge-bases/${id}/files/delete`, { ids })
}

export function reparseKnowledgeFiles(id: string, ids: string[]) {
  return http.post<null>(`/api/v1/knowledge-bases/${id}/files/reparse`, { ids })
}

export function getWikiStatus(id: string) {
  return http.get<WikiStatus>(`/api/v1/knowledge-bases/${id}/wiki/status`)
}

export function getWikiGraph(id: string, params: { mode?: string; center?: string; depth?: number; limit?: number } = {}) {
  return http.get<WikiGraph>(`/api/v1/knowledge-bases/${id}/wiki/graph`, { params })
}

export function getWikiPage(id: string, slug: string) {
  return http.get<WikiPage>(`/api/v1/knowledge-bases/${id}/wiki/pages/${slug}`)
}

export function searchWiki(id: string, q: string) {
  return http.get<{ pages: WikiPage[] }>(`/api/v1/knowledge-bases/${id}/wiki/search`, { params: { q } })
}
