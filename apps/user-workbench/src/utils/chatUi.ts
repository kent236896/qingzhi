import type { ChatKnowledgeBase, ChatSession } from '@/types/chat'

export interface SessionGroup {
  key: string
  label: string
  items: ChatSession[]
}

/** Group sessions for sidebar history labels (UI only). */
export function groupSessionsByRecency(sessions: ChatSession[]): SessionGroup[] {
  const now = Date.now()
  const dayMs = 24 * 60 * 60 * 1000
  const startOfToday = new Date()
  startOfToday.setHours(0, 0, 0, 0)
  const todayStart = startOfToday.getTime()

  const buckets: SessionGroup[] = [
    { key: 'today', label: '今天', items: [] },
    { key: '7d', label: '近7天', items: [] },
    { key: '30d', label: '近30天', items: [] },
    { key: 'older', label: '更早', items: [] },
  ]

  for (const item of sessions) {
    const raw = item.updated_at || item.created_at
    const ts = raw ? new Date(raw).getTime() : NaN
    if (!Number.isFinite(ts) || ts >= todayStart) {
      buckets[0].items.push(item)
      continue
    }
    const age = now - ts
    if (age < 7 * dayMs) buckets[1].items.push(item)
    else if (age < 30 * dayMs) buckets[2].items.push(item)
    else buckets[3].items.push(item)
  }

  return buckets.filter((group) => group.items.length > 0)
}

/**
 * Build suggestion chips only from real knowledge-base fields.
 * No invented FAQ content: empty list when no KB name/description.
 */
export function buildSuggestionPool(knowledgeBases: ChatKnowledgeBase[]): string[] {
  const pool: string[] = []
  for (const kb of knowledgeBases) {
    const name = (kb.name || '').trim()
    const desc = (kb.description || '').trim()
    if (!name && !desc) continue
    if (name) {
      pool.push(`请介绍知识库「${name}」的主要内容`)
    }
    if (desc) {
      const snippet = desc.length > 48 ? `${desc.slice(0, 48)}…` : desc
      pool.push(`关于「${name || '该知识库'}」：${snippet}`)
    }
  }
  return pool
}

export function pickSuggestions(pool: string[], seed: number, limit = 4): string[] {
  if (pool.length === 0) return []
  const copy = [...pool]
  let s = seed || 1
  for (let i = copy.length - 1; i > 0; i -= 1) {
    s = (s * 9301 + 49297) % 233280
    const j = s % (i + 1)
    ;[copy[i], copy[j]] = [copy[j], copy[i]]
  }
  return copy.slice(0, Math.min(limit, copy.length))
}
