import { http } from '@/api/client'
import type {
  KnowledgeBase,
  KnowledgeFile,
  KnowledgeFormOptions,
  KnowledgeLimits,
  UploadResult,
  WikiPageList,
} from '@/types/knowledge'

export function listKnowledgeBases(params: {
  page?: number
  page_size?: number
  keyword?: string
  scope_type?: string
} = {}) {
  return http.get<{ items: KnowledgeBase[]; page: number; page_size: number; total: number }>(
    '/api/v1/knowledge-bases',
    { params },
  )
}

export function getKnowledgeOptions() {
  return http.get<KnowledgeFormOptions>('/api/v1/knowledge-bases/options')
}

export function getKnowledgeLimits() {
  return http.get<KnowledgeLimits>('/api/v1/knowledge-bases/limits')
}

export function createKnowledgeBase(body: {
  name: string
  description?: string
  scope_type: 'personal'
  tenant_ids: string[]
  share_user_ids?: string[]
  wiki_enabled?: boolean
}) {
  return http.post<KnowledgeBase>('/api/v1/knowledge-bases', body)
}

export function updateKnowledgeBase(
  id: string,
  body: { name?: string; description?: string; share_user_ids?: string[] },
) {
  return http.patch<KnowledgeBase>(`/api/v1/knowledge-bases/${id}`, body)
}

export function shareKnowledgeBase(id: string, body: { share_user_ids?: string[] }) {
  return http.put<KnowledgeBase>(`/api/v1/knowledge-bases/${id}/shares`, body)
}

export function deleteKnowledgeBase(id: string) {
  return http.delete<null>(`/api/v1/knowledge-bases/${id}`)
}

export function getKnowledgeBase(id: string) {
  return http.get<KnowledgeBase>(`/api/v1/knowledge-bases/${id}`)
}

export function listKnowledgeFiles(id: string, sync = false) {
  return http.get<KnowledgeFile[]>(`/api/v1/knowledge-bases/${id}/files`, {
    params: sync ? { sync: 1 } : {},
  })
}

export function uploadKnowledgeFiles(id: string, files: File[]) {
  const body = new FormData()
  for (const file of files) {
    body.append('file', file)
  }
  return http.post<UploadResult[]>(`/api/v1/knowledge-bases/${id}/files`, body, {
    timeout: 120_000,
  })
}

export function deleteKnowledgeFiles(id: string, ids: string[]) {
  return http.post<null>(`/api/v1/knowledge-bases/${id}/files/delete`, { ids })
}

export function searchWiki(id: string, q: string, limit = 20) {
  return http.get<WikiPageList>(`/api/v1/knowledge-bases/${id}/wiki/search`, {
    params: { q, limit },
  })
}
