import { http } from '@/api/client'
import type {
  ChatKnowledgeBase,
  ChatMessage,
  ChatSession,
  ChatSessionPage,
  StreamEvent,
} from '@/types/chat'

export function listChatKnowledgeBases(keyword = '') {
  return http.get<ChatKnowledgeBase[]>('/api/v1/chat/knowledge-bases', {
    params: keyword ? { keyword } : undefined,
  })
}

export function listChatSessions(page = 1, pageSize = 50) {
  return http.get<ChatSessionPage>('/api/v1/chat/sessions', {
    params: { page, page_size: pageSize },
  })
}

export function createChatSession(title?: string) {
  return http.post<ChatSession>('/api/v1/chat/sessions', { title: title || '' })
}

export function renameChatSession(id: string, title: string) {
  return http.put<ChatSession>(`/api/v1/chat/sessions/${id}`, { title })
}

export function deleteChatSession(id: string) {
  return http.delete<{ deleted: boolean }>(`/api/v1/chat/sessions/${id}`)
}

export function loadChatMessages(sessionId: string, limit = 50) {
  return http.get<ChatMessage[]>(`/api/v1/chat/sessions/${sessionId}/messages`, {
    params: { limit },
  })
}

export function stopChatGeneration(sessionId: string, messageId: string) {
  return http.post<{ stopped: boolean }>(`/api/v1/chat/sessions/${sessionId}/stop`, {
    message_id: messageId,
  })
}

export interface AskChatOptions {
  sessionId: string
  query: string
  knowledgeBaseIds: string[]
  signal?: AbortSignal
  onEvent: (event: StreamEvent) => void
}

/**
 * POST SSE knowledge Q&A through the platform proxy.
 * Uses fetch + ReadableStream (EventSource cannot POST with auth body).
 */
export async function askChatStream(options: AskChatOptions): Promise<void> {
  const base = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '')
  const token = localStorage.getItem('qz_wb_access_token')
  const response = await fetch(`${base}/api/v1/chat/sessions/${options.sessionId}/ask`, {
    method: 'POST',
    headers: {
      Accept: 'text/event-stream',
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({
      query: options.query,
      knowledge_base_ids: options.knowledgeBaseIds,
    }),
    signal: options.signal,
  })

  if (!response.ok) {
    let message = `请求失败（${response.status}）`
    try {
      const body = (await response.json()) as { message?: string }
      if (body.message) message = body.message
    } catch {
      /* ignore */
    }
    if (response.status === 401) {
      localStorage.removeItem('qz_wb_access_token')
      localStorage.removeItem('qz_wb_tenant_id')
      const path = window.location.pathname + window.location.search
      if (!path.startsWith('/login')) {
        window.location.assign(`/login?redirect=${encodeURIComponent(path)}`)
      }
    }
    throw new Error(message)
  }

  if (!response.body) {
    throw new Error('浏览器不支持流式响应')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder('utf-8')
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    buffer = consumeSSE(buffer, options.onEvent)
  }
  if (buffer.trim()) {
    consumeSSE(buffer + '\n\n', options.onEvent)
  }
}

function consumeSSE(chunk: string, onEvent: (event: StreamEvent) => void): string {
  const parts = chunk.split('\n\n')
  const rest = parts.pop() ?? ''
  for (const block of parts) {
    const dataLines: string[] = []
    for (const line of block.split('\n')) {
      if (line.startsWith('data:')) {
        dataLines.push(line.slice(5).trimStart())
      }
    }
    const raw = dataLines.join('\n').trim()
    if (!raw) continue
    try {
      onEvent(JSON.parse(raw) as StreamEvent)
    } catch {
      /* ignore malformed frames */
    }
  }
  return rest
}
