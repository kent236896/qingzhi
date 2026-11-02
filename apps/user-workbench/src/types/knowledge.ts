export type KnowledgeScopeType = 'personal' | 'org_node' | 'position' | 'role'

export interface KnowledgeBase {
  id: string
  name: string
  description: string
  scope_type: KnowledgeScopeType
  scope_id: string
  scope_name?: string
  wiki_enabled: boolean
  status: number
  creator_id: string
  tenant_ids: string[]
  share_user_ids?: string[]
  share_org_ids?: string[]
  share_role_ids?: string[]
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

export interface KnowledgeFormOptions {
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

export interface WikiPage {
  id: string
  slug: string
  title: string
  page_type: string
  summary: string
  content: string
}

export interface WikiPageList {
  pages: WikiPage[]
  total: number
  page: number
  page_size: number
  total_pages: number
}
