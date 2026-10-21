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

export interface WikiGraphMeta {
  mode: string
  total: number
  returned: number
  truncated: boolean
  center?: string
  depth?: number
}

export interface WikiGraph {
  nodes: WikiNode[]
  edges: { source: string; target: string }[]
  meta: WikiGraphMeta
}

export interface WikiPage {
  id: string
  slug: string
  title: string
  page_type: string
  status?: string
  summary: string
  content: string
  aliases?: string[] | null
  folder_id?: string
  category_path?: string[]
  wiki_path?: string
  depth?: number
  in_links?: string[] | null
  out_links?: string[] | null
  version?: number
  last_edit_source?: string
  created_at?: string
  updated_at?: string
}

export interface WikiFolder {
  id: string
  parent_id: string
  name: string
  path: string
  depth: number
  sort_order: number
  page_count: number
  has_children: boolean
}

export interface WikiFolderList {
  parent_id: string
  folders: WikiFolder[]
}

export interface WikiPageList {
  pages: WikiPage[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface WikiIndexItem {
  slug: string
  title: string
  summary: string
  category_path?: string[]
  wiki_path?: string
  depth?: number
}

export interface WikiIndexGroup {
  type: string
  total: number
  items: WikiIndexItem[]
  next_cursor?: string
}

export interface WikiIndex {
  intro: string
  version: number
  groups: WikiIndexGroup[]
}

export interface WikiStats {
  total_pages: number
  pages_by_type: Record<string, number>
  total_links: number
  orphan_count: number
  pending_tasks?: number
  pending_issues?: number
  is_active?: boolean
}

export interface WikiTitle {
  slug: string
  title: string
  page_type?: string
}

export const WIKI_KNOWLEDGE_TYPES = ['entity', 'concept', 'category'] as const
export const WIKI_SUMMARY_TYPES = ['summary', 'synthesis', 'comparison'] as const

export function wikiGroupCounts(stats?: WikiStats | null) {
  const byType = stats?.pages_by_type ?? {}
  const sum = (types: readonly string[]) => types.reduce((total, type) => total + (byType[type] ?? 0), 0)
  return { knowledge: sum(WIKI_KNOWLEDGE_TYPES), summary: sum(WIKI_SUMMARY_TYPES) }
}

export interface FormOptions {
  tenants: { id: string; name: string; tenant_id?: string }[]
  organizations: { id: string; name: string; tenant_id?: string }[]
  posts: { id: string; name: string; tenant_id?: string }[]
  roles: { id: string; name: string; tenant_id?: string }[]
  users: { id: string; name: string }[]
  can_manage: boolean
  models?: {
    embedding: { id: string; name: string; display_name: string }
    summary: { id: string; name: string; display_name: string }
    wiki_synthesis: { id: string; name: string; display_name: string }
  }
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

export function getWikiStats(id: string) {
  return http.get<WikiStats>(`/api/v1/knowledge-bases/${id}/wiki/stats`)
}

export function listWikiFolders(id: string, params: { parent_id?: string; page_types?: string } = {}) {
  return http.get<WikiFolderList>(`/api/v1/knowledge-bases/${id}/wiki/folders`, { params })
}

export function listWikiPages(
  id: string,
  params: {
    folder_id?: string
    page_type?: string
    query?: string
    page?: number
    page_size?: number
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  } = {},
) {
  return http.get<WikiPageList>(`/api/v1/knowledge-bases/${id}/wiki/pages`, { params })
}

export function getWikiPage(id: string, slug: string) {
  return http.get<WikiPage>(`/api/v1/knowledge-bases/${id}/wiki/pages/${encodeWikiSlug(slug)}`)
}

export function getWikiIndex(id: string, params: { types?: string; limit?: number; cursor?: string } = {}) {
  return http.get<WikiIndex>(`/api/v1/knowledge-bases/${id}/wiki/index`, { params })
}

export function searchWiki(id: string, q: string, limit = 10) {
  return http.get<WikiPageList>(`/api/v1/knowledge-bases/${id}/wiki/search`, { params: { q, limit } })
}

export function getWikiTitles(id: string) {
  return http.get<WikiTitle[]>(`/api/v1/knowledge-bases/${id}/wiki/titles`)
}

function encodeWikiSlug(slug: string) {
  return slug
    .split('/')
    .map((part) => encodeURIComponent(part))
    .join('/')
}
