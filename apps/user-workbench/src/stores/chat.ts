import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  askChatStream,
  createChatSession,
  deleteChatSession,
  listChatKnowledgeBases,
  listChatSessions,
  loadChatMessages,
  renameChatSession,
  stopChatGeneration,
} from '@/api/chat'
import { listAgents } from '@/api/catalog'
import type { ChatAgentOption, ChatKnowledgeBase, ChatMessage, ChatReference, ChatSession } from '@/types/chat'
import { ApiError } from '@/utils/http'

export const useChatStore = defineStore('chat', () => {
  const sessions = ref<ChatSession[]>([])
  const knowledgeBases = ref<ChatKnowledgeBase[]>([])
  const agents = ref<ChatAgentOption[]>([])
  const selectedKnowledgeBaseIds = ref<string[]>([])
  const selectedAgentId = ref('')
  const activeSessionId = ref('')
  const messages = ref<ChatMessage[]>([])
  const streaming = ref(false)
  const loadingSessions = ref(false)
  const loadingMessages = ref(false)
  const bootstrapping = ref(false)
  const errorMessage = ref('')
  const sessionFilter = ref('')
  const assistantMessageId = ref('')
  let abortController: AbortController | null = null
  let currentAssistantLocalId = ''

  const activeSession = computed(
    () => sessions.value.find((item) => item.id === activeSessionId.value) || null,
  )
  const selectedAgent = computed(
    () => agents.value.find((item) => item.id === selectedAgentId.value) || null,
  )
  const filteredSessions = computed(() => {
    const keyword = sessionFilter.value.trim().toLowerCase()
    if (!keyword) return sessions.value
    return sessions.value.filter((item) => (item.title || '').toLowerCase().includes(keyword))
  })

  async function bootstrap() {
    if (bootstrapping.value) return
    bootstrapping.value = true
    errorMessage.value = ''
    try {
      await Promise.all([loadKnowledgeBases(), loadAgents(), loadSessions()])
    } finally {
      bootstrapping.value = false
    }
  }

  async function loadKnowledgeBases() {
    const response = await listChatKnowledgeBases()
    knowledgeBases.value = response.data ?? []
    if (selectedKnowledgeBaseIds.value.length === 0 && knowledgeBases.value.length > 0) {
      selectedKnowledgeBaseIds.value = [knowledgeBases.value[0].id]
    }
  }

  async function loadAgents() {
    try {
      const response = await listAgents()
      agents.value = (response.data ?? []).map((item) => ({
        id: item.id,
        name: item.name,
        description: item.description,
      }))
    } catch {
      agents.value = []
    }
  }

  async function loadSessions() {
    loadingSessions.value = true
    errorMessage.value = ''
    try {
      const response = await listChatSessions(1, 100)
      sessions.value = response.data.items ?? []
    } catch (error) {
      errorMessage.value = error instanceof ApiError ? error.message : '加载会话失败'
      throw error
    } finally {
      loadingSessions.value = false
    }
  }

  async function openSession(sessionId: string) {
    if (streaming.value) {
      await stop()
    }
    activeSessionId.value = sessionId
    const session = sessions.value.find((item) => item.id === sessionId)
    selectedAgentId.value = session?.agent_id || ''
    loadingMessages.value = true
    errorMessage.value = ''
    try {
      const response = await loadChatMessages(sessionId, 100)
      messages.value = response.data ?? []
    } catch (error) {
      errorMessage.value = error instanceof ApiError ? error.message : '加载消息失败'
      messages.value = []
      throw error
    } finally {
      loadingMessages.value = false
    }
  }

  /** Clears the composer to a blank draft so the user can pick an agent before the first ask. */
  async function startDraft() {
    if (streaming.value) {
      await stop()
    }
    activeSessionId.value = ''
    messages.value = []
    errorMessage.value = ''
    assistantMessageId.value = ''
    currentAssistantLocalId = ''
  }

  async function createSession(title?: string) {
    if (streaming.value) {
      await stop()
    }
    const response = await createChatSession(title, selectedAgentId.value || undefined)
    sessions.value = [response.data, ...sessions.value.filter((item) => item.id !== response.data.id)]
    activeSessionId.value = response.data.id
    messages.value = []
    return response.data
  }

  async function renameSession(sessionId: string, title: string) {
    const response = await renameChatSession(sessionId, title)
    sessions.value = sessions.value.map((item) => (item.id === sessionId ? response.data : item))
  }

  async function removeSession(sessionId: string) {
    await deleteChatSession(sessionId)
    sessions.value = sessions.value.filter((item) => item.id !== sessionId)
    if (activeSessionId.value === sessionId) {
      activeSessionId.value = ''
      messages.value = []
    }
  }

  async function sendQuestion(query: string) {
    const text = query.trim()
    if (!text || streaming.value) return
    const useAgent = Boolean(selectedAgentId.value)
    if (!useAgent && selectedKnowledgeBaseIds.value.length === 0) {
      errorMessage.value = '请先选择至少一个知识库，或选择一个智能体'
      return
    }
    errorMessage.value = ''
    if (!activeSessionId.value) {
      await createSession(text.slice(0, 30))
    } else if (selectedAgentId.value && activeSession.value?.agent_id !== selectedAgentId.value) {
      // Agent is bound at session create; start a new session when selection changes.
      await createSession(text.slice(0, 30))
    }
    const sessionId = activeSessionId.value
    const userMessage: ChatMessage = {
      id: `local-user-${Date.now()}`,
      session_id: sessionId,
      request_id: '',
      content: text,
      role: 'user',
      knowledge_references: [],
      is_completed: true,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    messages.value = [...messages.value, userMessage]

    const assistantLocalId = `local-assistant-${Date.now()}`
    currentAssistantLocalId = assistantLocalId
    const assistant: ChatMessage = {
      id: assistantLocalId,
      session_id: sessionId,
      request_id: '',
      content: '',
      role: 'assistant',
      knowledge_references: [],
      is_completed: false,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    messages.value = [...messages.value, assistant]
    streaming.value = true
    assistantMessageId.value = ''
    abortController = new AbortController()

    try {
      await askChatStream({
        sessionId,
        query: text,
        knowledgeBaseIds: [...selectedKnowledgeBaseIds.value],
        signal: abortController.signal,
        onEvent(event) {
          const type = event.response_type || ''
          if (event.assistant_message_id) {
            assistantMessageId.value = event.assistant_message_id
          }
          if (type === 'answer' && event.content) {
            patchAssistant(assistantLocalId, (msg) => {
              msg.content += event.content || ''
              if (event.id) {
                assistantMessageId.value = event.id
              }
            })
          } else if (type === 'references' && event.knowledge_references) {
            patchAssistant(assistantLocalId, (msg) => {
              msg.knowledge_references = event.knowledge_references || []
            })
          } else if (type === 'session_title' && event.content) {
            sessions.value = sessions.value.map((item) =>
              item.id === sessionId ? { ...item, title: event.content || item.title } : item,
            )
          } else if (type === 'error') {
            errorMessage.value = event.content || '生成失败'
          } else if (type === 'complete' || event.done) {
            patchAssistant(assistantLocalId, (msg) => {
              msg.is_completed = true
            })
          }
        },
      })
      patchAssistant(assistantLocalId, (msg) => {
        msg.is_completed = true
      })
    } catch (error) {
      if ((error as Error).name === 'AbortError') {
        patchAssistant(assistantLocalId, (msg) => {
          msg.is_completed = true
          if (!msg.content) msg.content = '（已停止生成）'
        })
      } else {
        errorMessage.value = error instanceof Error ? error.message : '发送失败'
        patchAssistant(assistantLocalId, (msg) => {
          msg.is_completed = true
          if (!msg.content) msg.content = `错误：${errorMessage.value}`
        })
      }
    } finally {
      streaming.value = false
      abortController = null
      currentAssistantLocalId = ''
    }
  }

  async function stop() {
    if (!streaming.value || !activeSessionId.value) return
    const messageId = assistantMessageId.value
    abortController?.abort()
    if (messageId) {
      try {
        await stopChatGeneration(activeSessionId.value, messageId)
      } catch {
        /* client abort already stops UI stream */
      }
    }
  }

  function patchAssistant(localId: string, mutate: (msg: ChatMessage) => void) {
    const targetId = localId || currentAssistantLocalId
    messages.value = messages.value.map((item) => {
      if (item.id !== targetId) {
        return item
      }
      const next: ChatMessage = {
        ...item,
        knowledge_references: [...(item.knowledge_references || [])] as ChatReference[],
      }
      mutate(next)
      return next
    })
  }

  /** Clears all chat UI state; call on logout / user switch so prior user data cannot linger. */
  function reset() {
    abortController?.abort()
    abortController = null
    sessions.value = []
    knowledgeBases.value = []
    agents.value = []
    selectedKnowledgeBaseIds.value = []
    selectedAgentId.value = ''
    activeSessionId.value = ''
    messages.value = []
    streaming.value = false
    loadingSessions.value = false
    loadingMessages.value = false
    bootstrapping.value = false
    errorMessage.value = ''
    sessionFilter.value = ''
    assistantMessageId.value = ''
    currentAssistantLocalId = ''
  }

  return {
    sessions,
    knowledgeBases,
    agents,
    selectedKnowledgeBaseIds,
    selectedAgentId,
    selectedAgent,
    activeSessionId,
    activeSession,
    messages,
    streaming,
    loadingSessions,
    loadingMessages,
    bootstrapping,
    errorMessage,
    sessionFilter,
    filteredSessions,
    bootstrap,
    loadKnowledgeBases,
    loadSessions,
    openSession,
    startDraft,
    createSession,
    renameSession,
    removeSession,
    sendQuestion,
    stop,
    reset,
  }
})
