import type { KnowledgeBase, KnowledgeScopeType } from '@/types/knowledge'

export const SCOPE_LABEL: Record<KnowledgeScopeType, string> = {
  personal: '个人',
  org_node: '组织',
  position: '岗位',
  role: '角色',
}

export function isPersonalKnowledge(item: Pick<KnowledgeBase, 'scope_type'>) {
  return item.scope_type === 'personal'
}

/** Personal KB owned by the current user may be edited / shared / deleted in the workbench. */
export function canManagePersonalKnowledge(
  item: Pick<KnowledgeBase, 'scope_type' | 'scope_id' | 'creator_id'>,
  userId: string | undefined,
) {
  if (!userId || !isPersonalKnowledge(item)) return false
  return item.scope_id === userId || item.creator_id === userId
}

export function formatFileSize(bytes: number) {
  if (!bytes || bytes < 0) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function parseStatusLabel(status: string) {
  switch (status) {
    case 'completed':
      return '已完成'
    case 'failed':
      return '失败'
    case 'processing':
    case 'pending':
      return '处理中'
    case 'duplicate':
      return '重复'
    case 'skipped':
      return '已跳过'
    default:
      return status || '未知'
  }
}
