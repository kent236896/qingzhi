export interface ChatKnowledgeBase {
  id: string
  name: string
  description: string
  scope_type: string
}

export interface ChatSession {
  id: string
  title: string
  description: string
  created_at: string
  updated_at: string
}

export interface ChatReference {
  id: string
  content: string
  knowledge_id: string
  knowledge_title: string
  knowledge_filename: string
  score: number
  knowledge_base_id: string
  chunk_index: number
}

export interface ChatMessage {
  id: string
  session_id: string
  request_id: string
  content: string
  role: string
  knowledge_references: ChatReference[]
  is_completed: boolean
  created_at: string
  updated_at: string
}

export interface ChatSessionPage {
  items: ChatSession[]
  page: number
  page_size: number
  total: number
}

export type StreamEventType =
  | 'answer'
  | 'references'
  | 'complete'
  | 'error'
  | 'thinking'
  | 'session_title'
  | string

export interface StreamEvent {
  id?: string
  response_type?: StreamEventType
  content?: string
  done?: boolean
  knowledge_references?: ChatReference[]
  assistant_message_id?: string
  session_id?: string
}
