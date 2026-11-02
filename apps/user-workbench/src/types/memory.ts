export interface MemoryStatus {
  verified: boolean
  ready: boolean
  reason: string
  retested_at?: string
  workspace_enabled: boolean | null
  write_mode?: string
  max_items?: number
}

export interface MemorySettings {
  workspace_enabled: boolean
  user_enabled: boolean
  effective: boolean
  write_mode: string
  item_count: number
  max_items: number
}

export type MemoryKind = 'profile' | 'preference' | 'fact' | 'task' | 'interest'

export interface MemoryItem {
  id: string
  kind: MemoryKind | string
  content: string
  status: string
  importance: number
  origin?: string
  subject_id?: string
  created_at: string
  updated_at: string
}

export interface MemoryItemList {
  items: MemoryItem[]
  total: number
  page: number
  page_size: number
}
