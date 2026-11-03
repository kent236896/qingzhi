<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { CloseIcon, RefreshIcon, SendIcon, StopCircleIcon } from 'tdesign-icons-vue-next'
import { MessagePlugin } from 'tdesign-vue-next'
import { useChatStore } from '@/stores/chat'
import type { ChatReference } from '@/types/chat'
import { buildSuggestionPool, pickSuggestions } from '@/utils/chatUi'
import { renderSafeMarkdown } from '@/utils/markdown'

const route = useRoute()
const chat = useChatStore()
const draft = ref('')
const messagePane = ref<HTMLElement | null>(null)
const inputEl = ref<HTMLTextAreaElement | null>(null)
const activeReferences = ref<ChatReference[]>([])
const showReferences = ref(false)
const kbPopupVisible = ref(false)
const suggestionSeed = ref(Date.now())

const productName = '知助手'

const emptyKnowledge = computed(() => !chat.bootstrapping && chat.knowledgeBases.length === 0)
const hasAgent = computed(() => Boolean(chat.selectedAgentId))
const canStart = computed(() => hasAgent.value || !emptyKnowledge.value)

const isWelcome = computed(
  () => !chat.loadingMessages && chat.messages.length === 0 && !chat.streaming,
)

const selectedKnowledgeBases = computed(() =>
  chat.knowledgeBases.filter((item) => chat.selectedKnowledgeBaseIds.includes(item.id)),
)

const suggestionPool = computed(() => {
  const selected = selectedKnowledgeBases.value
  const source = selected.length > 0 ? selected : chat.knowledgeBases
  return buildSuggestionPool(source)
})

const suggestions = computed(() => pickSuggestions(suggestionPool.value, suggestionSeed.value, 4))

const canSend = computed(
  () => Boolean(draft.value.trim()) && !chat.streaming && canStart.value,
)

const agentOptions = computed(() => [
  { label: '不使用智能体（知识库问答）', value: '' },
  ...chat.agents.map((item) => ({ label: item.name, value: item.id })),
])

async function refresh() {
  try {
    await chat.bootstrap()
  } catch {
    /* store keeps errorMessage */
  }
}

function refreshSuggestions() {
  suggestionSeed.value = Date.now()
}

function toggleKnowledge(id: string) {
  const set = new Set(chat.selectedKnowledgeBaseIds)
  if (set.has(id)) set.delete(id)
  else set.add(id)
  chat.selectedKnowledgeBaseIds = [...set]
}

function removeKnowledge(id: string) {
  chat.selectedKnowledgeBaseIds = chat.selectedKnowledgeBaseIds.filter((item) => item !== id)
}

async function onSend() {
  const text = draft.value
  if (!canSend.value) return
  draft.value = ''
  showReferences.value = false
  await chat.sendQuestion(text)
  await scrollToBottom()
}

async function onStop() {
  await chat.stop()
}

async function askSuggestion(text: string) {
  if (!canStart.value || chat.streaming) return
  draft.value = text
  await onSend()
}

function openReferences(refs: ChatReference[]) {
  activeReferences.value = refs || []
  showReferences.value = true
}

async function copyText(content: string) {
  try {
    await navigator.clipboard.writeText(content)
    void MessagePlugin.success('已复制')
  } catch {
    void MessagePlugin.error('复制失败')
  }
}

async function scrollToBottom() {
  await nextTick()
  if (messagePane.value) {
    messagePane.value.scrollTop = messagePane.value.scrollHeight
  }
}

function onComposerKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    void onSend()
  }
}

function autosizeInput() {
  const el = inputEl.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(el.scrollHeight, 144)}px`
}

watch(draft, () => {
  void nextTick(() => autosizeInput())
})

watch(
  () => chat.messages.length,
  () => {
    void scrollToBottom()
  },
)

watch(
  () => chat.messages.map((item) => item.content).join(''),
  () => {
    if (chat.streaming) void scrollToBottom()
  },
)

onMounted(async () => {
  await refresh()
  const kb = typeof route.query.kb === 'string' ? route.query.kb : ''
  if (kb && chat.knowledgeBases.some((item) => item.id === kb)) {
    chat.selectedKnowledgeBaseIds = [kb]
  }
})

watch(
  () => route.query.kb,
  (kb) => {
    if (typeof kb !== 'string' || !kb) return
    if (chat.knowledgeBases.some((item) => item.id === kb)) {
      chat.selectedKnowledgeBaseIds = [kb]
    }
  },
)
</script>

<template>
  <div class="chat">
    <section class="chat__stage">
      <div ref="messagePane" class="chat__scroll">
        <div v-if="isWelcome" class="welcome">
          <h1 class="welcome__title">Hi，我是 {{ productName }}，让你的知识触手可及</h1>

          <template v-if="canStart && suggestions.length > 0">
            <div class="welcome__prompt-row">
              <span>你可以这样问我</span>
              <button type="button" class="welcome__refresh" title="换一批" @click="refreshSuggestions">
                <RefreshIcon />
              </button>
            </div>
            <div class="welcome__chips">
              <button
                v-for="(item, index) in suggestions"
                :key="`${suggestionSeed}-${index}`"
                type="button"
                class="welcome__chip"
                @click="askSuggestion(item)"
              >
                {{ item }}
              </button>
            </div>
          </template>
        </div>

        <div v-else class="messages">
          <p v-if="chat.loadingMessages" class="chat__soft">正在加载消息…</p>
          <article
            v-for="item in chat.messages"
            :key="item.id"
            class="msg"
            :class="item.role === 'user' ? 'msg--user' : 'msg--assistant'"
          >
            <div v-if="item.role === 'assistant'" class="msg__toolbar">
              <button
                v-if="item.content"
                type="button"
                class="msg__tool"
                @click="copyText(item.content)"
              >
                复制
              </button>
              <button
                v-if="item.knowledge_references?.length"
                type="button"
                class="msg__tool"
                @click="openReferences(item.knowledge_references)"
              >
                查看来源（{{ item.knowledge_references.length }}）
              </button>
            </div>
            <div
              v-if="item.role === 'assistant' && !item.content && !item.is_completed"
              class="msg__waiting"
              aria-live="polite"
              aria-label="正在生成回答"
            >
              <span class="msg__waiting-dots" aria-hidden="true">
                <i /><i /><i />
              </span>
              <span class="msg__waiting-text">正在检索并生成回答…</span>
            </div>
            <div
              v-else-if="item.role === 'assistant'"
              class="msg__body markdown-body"
              v-html="renderSafeMarkdown(item.content)"
            />
            <div v-else class="msg__bubble">{{ item.content }}</div>
          </article>
        </div>
      </div>

      <div class="chat__bottom">
        <div v-if="!canStart" class="chat__kb-tip">
          <span>
            当前账号没有可检索的知识库或可用智能体，请联系管理员授权后重试，不会使用模拟数据。
          </span>
          <button type="button" class="chat__kb-reload" @click="refresh">重新加载</button>
        </div>
        <p v-else-if="chat.errorMessage" class="chat__error">{{ chat.errorMessage }}</p>

        <div class="composer" :class="{ 'composer--disabled': !canStart }">
          <div class="composer__agent">
            <span class="composer__agent-label">智能体</span>
            <t-select
              v-model="chat.selectedAgentId"
              :options="agentOptions"
              :disabled="chat.streaming || Boolean(chat.activeSessionId)"
              placeholder="不使用智能体"
              clearable
              style="width: 220px"
            />
            <span v-if="chat.activeSessionId && chat.activeSession?.agent_id" class="composer__agent-bound">
              本会话已绑定智能体
            </span>
          </div>

          <div v-if="selectedKnowledgeBases.length" class="composer__chips">
            <span v-for="kb in selectedKnowledgeBases" :key="kb.id" class="composer__chip">
              @{{ kb.name }}
              <button
                type="button"
                class="composer__chip-remove"
                :disabled="chat.streaming || !canStart"
                @click="removeKnowledge(kb.id)"
              >
                <CloseIcon />
              </button>
            </span>
          </div>

          <textarea
            ref="inputEl"
            v-model="draft"
            class="composer__input"
            rows="2"
            placeholder="输入问题或描述任务..."
            :disabled="chat.streaming || !canStart"
            @keydown="onComposerKeydown"
            @input="autosizeInput"
          />

          <div class="composer__toolbar">
            <div class="composer__tools">
              <span class="composer__mode">{{ hasAgent ? '智能体对话' : '快速问答' }}</span>

              <t-popup
                v-model:visible="kbPopupVisible"
                trigger="click"
                placement="top-left"
                :disabled="emptyKnowledge || chat.streaming"
              >
                <button
                  type="button"
                  class="composer__at"
                  :disabled="emptyKnowledge || chat.streaming"
                  title="选择授权知识库"
                >
                  @
                </button>
                <template #content>
                  <div class="kb-popup">
                    <p class="kb-popup__title">授权知识库</p>
                    <label
                      v-for="kb in chat.knowledgeBases"
                      :key="kb.id"
                      class="kb-popup__item"
                    >
                      <input
                        type="checkbox"
                        :checked="chat.selectedKnowledgeBaseIds.includes(kb.id)"
                        @change="toggleKnowledge(kb.id)"
                      />
                      <span>{{ kb.name }}</span>
                    </label>
                    <p v-if="chat.knowledgeBases.length === 0" class="kb-popup__empty">暂无知识库</p>
                  </div>
                </template>
              </t-popup>
            </div>

            <div class="composer__actions">
              <button
                v-if="chat.streaming"
                type="button"
                class="composer__stop"
                title="停止生成"
                @click="onStop"
              >
                <StopCircleIcon />
              </button>
              <button
                type="button"
                class="composer__send"
                :disabled="!canSend"
                title="发送"
                @click="onSend"
              >
                <SendIcon />
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <aside v-if="showReferences" class="refs">
      <div class="refs__header">
        <h3>引用来源</h3>
        <button type="button" class="refs__close" @click="showReferences = false">
          <CloseIcon />
        </button>
      </div>
      <p class="refs__hint">
        本阶段展示检索片段与元数据。未提供安全原文代理时，不会生成永久公开链接。
      </p>
      <article v-for="(refItem, index) in activeReferences" :key="refItem.id || index" class="refs__card">
        <h4>{{ refItem.knowledge_title || refItem.knowledge_filename || '未命名文档' }}</h4>
        <p class="refs__meta">
          相关度 {{ typeof refItem.score === 'number' ? refItem.score.toFixed(4) : '—' }}
          <span v-if="refItem.knowledge_id"> · ID {{ refItem.knowledge_id }}</span>
        </p>
        <p class="refs__content">{{ refItem.content || '无片段内容' }}</p>
      </article>
    </aside>
  </div>
</template>

<style scoped>
.chat {
  flex: 1;
  min-height: 0;
  display: flex;
  background: var(--wb-surface);
}

.chat__stage {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  position: relative;
}

.chat__scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 24px 24px 8px;
}

.welcome {
  max-width: 800px;
  margin: 12vh auto 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 0 12px 24px;
}

.welcome__title {
  margin: 0;
  font-size: 30px;
  font-weight: 700;
  line-height: 1.35;
  color: var(--wb-text);
  letter-spacing: 0.01em;
}

.welcome__prompt-row {
  margin-top: 24px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--wb-text-secondary);
}

.welcome__refresh {
  border: 0;
  background: transparent;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--wb-text-secondary);
}

.welcome__refresh:hover {
  background: var(--wb-hover);
  color: var(--wb-primary);
}

.welcome__chips {
  margin-top: 16px;
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
  max-width: 100%;
}

.welcome__chip {
  border: 1px solid var(--wb-border);
  background: var(--wb-surface);
  color: var(--wb-text);
  border-radius: 10px;
  padding: 8px 14px;
  font-size: 14px;
  line-height: 1.4;
  cursor: pointer;
  max-width: 100%;
  text-align: left;
}

.welcome__chip:hover {
  border-color: var(--wb-primary);
  color: var(--wb-primary);
}

.messages {
  max-width: 800px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding-bottom: 16px;
}

.chat__soft {
  margin: 24px 0;
  text-align: center;
  color: var(--wb-text-secondary);
  font-size: 13px;
}

.msg {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.msg--user {
  align-items: flex-end;
}

.msg--assistant {
  align-items: flex-start;
}

.msg__toolbar {
  display: flex;
  gap: 8px;
}

.msg__tool {
  border: 0;
  background: transparent;
  color: var(--wb-text-secondary);
  font-size: 12px;
  cursor: pointer;
  padding: 0;
}

.msg__tool:hover {
  color: var(--wb-primary);
}

.msg__bubble {
  max-width: min(680px, 100%);
  padding: 10px 14px;
  border-radius: 12px;
  background: var(--wb-primary-light);
  color: var(--wb-text);
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.msg__body {
  max-width: min(720px, 100%);
  font-size: 14px;
  line-height: 1.7;
  color: var(--wb-text);
  word-break: break-word;
}

.msg__waiting {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  max-width: min(420px, 100%);
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid var(--wb-border);
  background: var(--wb-surface, #fff);
  color: var(--wb-text-secondary);
  font-size: 13px;
  line-height: 1.4;
  animation: msg-waiting-in 0.28s ease-out;
}

.msg__waiting-dots {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.msg__waiting-dots i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--wb-primary);
  opacity: 0.35;
  animation: msg-waiting-bounce 1.05s ease-in-out infinite;
}

.msg__waiting-dots i:nth-child(2) {
  animation-delay: 0.15s;
}

.msg__waiting-dots i:nth-child(3) {
  animation-delay: 0.3s;
}

.msg__waiting-text {
  color: var(--wb-text-secondary);
}

@keyframes msg-waiting-in {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes msg-waiting-bounce {
  0%,
  80%,
  100% {
    opacity: 0.3;
    transform: translateY(0);
  }
  40% {
    opacity: 1;
    transform: translateY(-3px);
  }
}

.markdown-body :deep(p) {
  margin: 0 0 0.65em;
}

.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(pre) {
  overflow: auto;
  padding: 10px;
  border-radius: 8px;
  background: var(--wb-sidebar-bg);
}

.chat__bottom {
  flex-shrink: 0;
  width: min(800px, calc(100% - 48px));
  margin: 0 auto 20px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.chat__kb-tip {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  background: var(--wb-sidebar-bg);
  color: var(--wb-text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.chat__kb-reload {
  border: 0;
  background: transparent;
  color: var(--wb-primary);
  font-size: 13px;
  cursor: pointer;
  white-space: nowrap;
  padding: 0;
}

.chat__error {
  margin: 0;
  color: var(--td-error-color, #d54941);
  font-size: 13px;
}

.composer {
  border: 1px solid var(--wb-border);
  border-radius: 16px;
  background: var(--wb-surface);
  box-shadow: 0 4px 18px color-mix(in srgb, var(--wb-text) 6%, transparent);
  padding: 12px 14px 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.composer--disabled {
  opacity: 0.72;
}

.composer__agent {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.composer__agent-label {
  font-size: 12px;
  color: var(--wb-text-secondary);
}

.composer__agent-bound {
  font-size: 12px;
  color: var(--wb-text-secondary);
}

.composer__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.composer__chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--wb-primary-light);
  color: var(--wb-primary);
  font-size: 12px;
}

.composer__chip-remove {
  border: 0;
  background: transparent;
  width: 16px;
  height: 16px;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: inherit;
}

.composer__input {
  width: 100%;
  border: 0;
  outline: none;
  resize: none;
  background: transparent;
  color: var(--wb-text);
  font-size: 14px;
  line-height: 1.6;
  min-height: 48px;
  max-height: 144px;
  overflow-y: auto;
  font-family: inherit;
}

.composer__input:disabled {
  cursor: not-allowed;
}

.composer__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.composer__tools {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.composer__mode {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 10px;
  border-radius: 8px;
  border: 1px solid var(--wb-border);
  font-size: 12px;
  color: var(--wb-text-secondary);
  background: var(--wb-surface);
}

.composer__at {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  border: 1px solid var(--wb-border);
  background: var(--wb-surface);
  color: var(--wb-text);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.composer__at:hover:not(:disabled) {
  border-color: var(--wb-primary);
  color: var(--wb-primary);
}

.composer__at:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.composer__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.composer__send,
.composer__stop {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.composer__send {
  background: var(--wb-primary);
  color: #fff;
}

.composer__send:disabled {
  background: var(--wb-hover);
  color: var(--wb-text-secondary);
  cursor: not-allowed;
}

.composer__stop {
  background: var(--wb-sidebar-bg);
  color: var(--wb-text);
}

.kb-popup {
  min-width: 220px;
  max-width: 280px;
  max-height: 260px;
  overflow: auto;
  padding: 8px;
}

.kb-popup__title {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--wb-text-secondary);
}

.kb-popup__item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 6px 4px;
  font-size: 13px;
  cursor: pointer;
  border-radius: 6px;
}

.kb-popup__item:hover {
  background: var(--wb-hover);
}

.kb-popup__empty {
  margin: 0;
  font-size: 12px;
  color: var(--wb-text-secondary);
}

.refs {
  width: 300px;
  flex-shrink: 0;
  border-left: 1px solid var(--wb-border);
  background: var(--wb-surface);
  padding: 14px;
  overflow: auto;
}

.refs__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.refs__header h3 {
  margin: 0;
  font-size: 15px;
}

.refs__close {
  border: 0;
  background: transparent;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--wb-text-secondary);
}

.refs__close:hover {
  background: var(--wb-hover);
}

.refs__hint {
  margin: 8px 0 12px;
  font-size: 12px;
  color: var(--wb-text-secondary);
  line-height: 1.5;
}

.refs__card {
  padding: 10px 0;
  border-top: 1px solid var(--wb-border);
}

.refs__card h4 {
  margin: 0 0 4px;
  font-size: 13px;
}

.refs__meta {
  margin: 0 0 6px;
  font-size: 12px;
  color: var(--wb-text-secondary);
}

.refs__content {
  margin: 0;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}

@media (max-width: 900px) {
  .welcome__title {
    font-size: 24px;
  }

  .chat__scroll {
    padding: 16px 16px 8px;
  }

  .chat__bottom {
    width: calc(100% - 32px);
    margin-bottom: 12px;
  }

  .refs {
    position: absolute;
    right: 0;
    top: 0;
    bottom: 0;
    z-index: 20;
    box-shadow: -8px 0 24px color-mix(in srgb, var(--wb-text) 8%, transparent);
  }
}
</style>
