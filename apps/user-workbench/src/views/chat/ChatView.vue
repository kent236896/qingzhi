<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { AddIcon, DeleteIcon, EditIcon, SendIcon, StopCircleIcon } from 'tdesign-icons-vue-next'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/common/PageStatus.vue'
import { useChatStore } from '@/stores/chat'
import type { ChatReference } from '@/types/chat'
import { renderSafeMarkdown } from '@/utils/markdown'

const chat = useChatStore()
const draft = ref('')
const messagePane = ref<HTMLElement | null>(null)
const activeReferences = ref<ChatReference[]>([])
const showReferences = ref(false)

const knowledgeOptions = computed(() =>
  chat.knowledgeBases.map((item) => ({ label: item.name, value: item.id })),
)

const emptyKnowledge = computed(() => !chat.bootstrapping && chat.knowledgeBases.length === 0)

async function refresh() {
  try {
    await chat.bootstrap()
  } catch {
    /* store already keeps errorMessage */
  }
}

async function onNewSession() {
  try {
    await chat.createSession()
    draft.value = ''
    activeReferences.value = []
    showReferences.value = false
  } catch (error) {
    void MessagePlugin.error(error instanceof Error ? error.message : '创建会话失败')
  }
}

async function onOpenSession(id: string) {
  try {
    await chat.openSession(id)
    activeReferences.value = []
    showReferences.value = false
    await scrollToBottom()
  } catch {
    /* surfaced via errorMessage */
  }
}

async function onRename(id: string, current: string) {
  const next = window.prompt('请输入新的会话标题', current || '新对话')
  if (next === null) return
  const title = next.trim()
  if (!title) {
    void MessagePlugin.warning('标题不能为空')
    return
  }
  try {
    await chat.renameSession(id, title)
    void MessagePlugin.success('已重命名')
  } catch (error) {
    void MessagePlugin.error(error instanceof Error ? error.message : '重命名失败')
  }
}

function onDelete(id: string) {
  const dialog = DialogPlugin.confirm({
    header: '删除会话',
    body: '删除后无法恢复，确定继续吗？',
    onConfirm: async () => {
      try {
        await chat.removeSession(id)
        void MessagePlugin.success('会话已删除')
      } catch (error) {
        void MessagePlugin.error(error instanceof Error ? error.message : '删除失败')
      } finally {
        dialog.destroy()
      }
    },
  })
}

async function onSend() {
  const text = draft.value
  if (!text.trim() || chat.streaming) return
  draft.value = ''
  await chat.sendQuestion(text)
  await scrollToBottom()
}

async function onStop() {
  await chat.stop()
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

onMounted(() => {
  void refresh()
})
</script>

<template>
  <div class="chat-page">
    <aside class="chat-sidebar">
      <div class="chat-sidebar__toolbar">
        <t-button theme="primary" block :disabled="chat.streaming" @click="onNewSession">
          <template #icon><AddIcon /></template>
          新建会话
        </t-button>
        <t-input v-model="chat.sessionFilter" clearable placeholder="搜索会话标题" />
      </div>
      <div class="chat-sidebar__list">
        <PageStatus
          v-if="chat.loadingSessions && chat.sessions.length === 0"
          title="正在加载会话"
          kind="loading"
        />
        <PageStatus
          v-else-if="chat.filteredSessions.length === 0"
          title="暂无会话"
          description="点击「新建会话」开始提问。"
          kind="empty"
        />
        <button
          v-for="item in chat.filteredSessions"
          :key="item.id"
          type="button"
          class="session-item"
          :class="{ 'session-item--active': item.id === chat.activeSessionId }"
          @click="onOpenSession(item.id)"
        >
          <div class="session-item__title">{{ item.title || '未命名会话' }}</div>
          <div class="session-item__actions" @click.stop>
            <t-button
              size="small"
              variant="text"
              shape="square"
              :title="'重命名'"
              @click="onRename(item.id, item.title)"
            >
              <EditIcon />
            </t-button>
            <t-button
              size="small"
              variant="text"
              shape="square"
              theme="danger"
              :title="'删除'"
              @click="onDelete(item.id)"
            >
              <DeleteIcon />
            </t-button>
          </div>
        </button>
      </div>
    </aside>

    <section class="chat-main">
      <header class="chat-main__header">
        <div class="chat-main__heading">
          <h2>{{ chat.activeSession?.title || 'AI 对话' }}</h2>
          <p>选择授权知识库后提问；回答与引用来自 WeKnora 知识检索。</p>
        </div>
        <div class="chat-main__kb">
          <span class="chat-main__kb-label">检索知识库</span>
          <t-select
            v-model="chat.selectedKnowledgeBaseIds"
            multiple
            clearable
            filterable
            placeholder="选择知识库"
            :options="knowledgeOptions"
            :disabled="chat.streaming || emptyKnowledge"
            style="min-width: 280px"
          />
        </div>
      </header>

      <PageStatus
        v-if="emptyKnowledge"
        title="没有可用知识库"
        description="当前账号没有可检索的知识库。请联系管理员授权后重试，不会使用模拟数据。"
        kind="empty"
      >
        <t-button theme="primary" variant="outline" @click="refresh">重新加载</t-button>
      </PageStatus>

      <template v-else>
        <div ref="messagePane" class="chat-messages">
          <PageStatus
            v-if="chat.loadingMessages"
            title="正在加载消息"
            kind="loading"
          />
          <PageStatus
            v-else-if="!chat.activeSessionId"
            title="开始一段对话"
            description="新建会话或从左侧选择历史会话。"
            kind="info"
          />
          <PageStatus
            v-else-if="chat.messages.length === 0"
            title="还没有消息"
            description="在下方输入问题，系统将在所选知识库中检索并回答。"
            kind="empty"
          />
          <article
            v-for="item in chat.messages"
            :key="item.id"
            class="bubble"
            :class="item.role === 'user' ? 'bubble--user' : 'bubble--assistant'"
          >
            <div class="bubble__meta">
              <span>{{ item.role === 'user' ? '我' : '助手' }}</span>
              <div class="bubble__tools">
                <t-button
                  v-if="item.role === 'assistant' && item.content"
                  size="small"
                  variant="text"
                  @click="copyText(item.content)"
                >
                  复制
                </t-button>
                <t-button
                  v-if="item.role === 'assistant' && item.knowledge_references?.length"
                  size="small"
                  variant="text"
                  @click="openReferences(item.knowledge_references)"
                >
                  查看来源（{{ item.knowledge_references.length }}）
                </t-button>
              </div>
            </div>
            <div
              v-if="item.role === 'assistant'"
              class="bubble__content markdown-body"
              v-html="renderSafeMarkdown(item.content || (item.is_completed ? '' : '…'))"
            />
            <div v-else class="bubble__content">{{ item.content }}</div>
          </article>
        </div>

        <p v-if="chat.errorMessage" class="chat-error">{{ chat.errorMessage }}</p>

        <footer class="chat-composer">
          <t-textarea
            v-model="draft"
            :autosize="{ minRows: 2, maxRows: 6 }"
            placeholder="输入问题，Enter 发送（Shift+Enter 换行）"
            :disabled="chat.streaming || emptyKnowledge"
            @keydown.enter.exact.prevent="onSend"
          />
          <div class="chat-composer__actions">
            <t-button
              v-if="chat.streaming"
              theme="default"
              variant="outline"
              @click="onStop"
            >
              <template #icon><StopCircleIcon /></template>
              停止生成
            </t-button>
            <t-button
              theme="primary"
              :loading="chat.streaming"
              :disabled="!draft.trim() || emptyKnowledge"
              @click="onSend"
            >
              <template #icon><SendIcon /></template>
              发送
            </t-button>
          </div>
        </footer>
      </template>
    </section>

    <aside v-if="showReferences" class="chat-refs">
      <div class="chat-refs__header">
        <h3>引用来源</h3>
        <t-button size="small" variant="text" @click="showReferences = false">关闭</t-button>
      </div>
      <p class="chat-refs__hint">
        本阶段展示检索片段与元数据。未提供安全原文代理时，不会生成永久公开链接。
      </p>
      <article v-for="(refItem, index) in activeReferences" :key="refItem.id || index" class="ref-card">
        <h4>{{ refItem.knowledge_title || refItem.knowledge_filename || '未命名文档' }}</h4>
        <p class="ref-card__meta">
          相关度 {{ typeof refItem.score === 'number' ? refItem.score.toFixed(4) : '—' }}
          <span v-if="refItem.knowledge_id"> · ID {{ refItem.knowledge_id }}</span>
        </p>
        <p class="ref-card__content">{{ refItem.content || '无片段内容' }}</p>
      </article>
    </aside>
  </div>
</template>

<style scoped>
.chat-page {
  display: flex;
  gap: 16px;
  min-height: calc(100vh - var(--wb-header-height) - 52px);
  max-height: calc(100vh - var(--wb-header-height) - 52px);
}

.chat-sidebar {
  width: 280px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  padding: 12px;
  overflow: hidden;
}

.chat-sidebar__toolbar {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.chat-sidebar__list {
  flex: 1;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  border: 0;
  background: transparent;
  border-radius: var(--td-radius-medium);
  padding: 10px 8px;
  cursor: pointer;
  text-align: left;
  color: var(--td-text-color-primary);
}

.session-item:hover {
  background: var(--td-bg-color-container-hover);
}

.session-item--active {
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
}

.session-item__title {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.session-item__actions {
  display: flex;
  gap: 0;
  opacity: 0.75;
}

.chat-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  overflow: hidden;
}

.chat-main__header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  padding: 16px 20px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.chat-main__heading h2 {
  margin: 0 0 4px;
  font-size: 18px;
}

.chat-main__heading p {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.chat-main__kb {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: flex-end;
}

.chat-main__kb-label {
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.chat-messages {
  flex: 1;
  overflow: auto;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.bubble {
  max-width: min(720px, 100%);
  padding: 12px 14px;
  border-radius: var(--td-radius-medium);
  background: var(--td-bg-color-secondarycontainer);
}

.bubble--user {
  align-self: flex-end;
  background: var(--td-brand-color-light);
}

.bubble--assistant {
  align-self: flex-start;
}

.bubble__meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.bubble__tools {
  display: flex;
  gap: 4px;
}

.bubble__content {
  font-size: 14px;
  line-height: 1.6;
  color: var(--td-text-color-primary);
  white-space: pre-wrap;
  word-break: break-word;
}

.markdown-body {
  white-space: normal;
}

.markdown-body :deep(p) {
  margin: 0 0 0.6em;
}

.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(pre) {
  overflow: auto;
  padding: 10px;
  border-radius: var(--td-radius-small);
  background: var(--td-bg-color-secondarycontainer);
}

.chat-error {
  margin: 0;
  padding: 0 20px 8px;
  color: var(--td-error-color);
  font-size: 13px;
}

.chat-composer {
  border-top: 1px solid var(--td-component-stroke);
  padding: 12px 16px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.chat-composer__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.chat-refs {
  width: 300px;
  flex-shrink: 0;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  padding: 12px;
  overflow: auto;
}

.chat-refs__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.chat-refs__header h3 {
  margin: 0;
  font-size: 15px;
}

.chat-refs__hint {
  margin: 8px 0 12px;
  font-size: 12px;
  color: var(--td-text-color-secondary);
  line-height: 1.5;
}

.ref-card {
  padding: 10px 0;
  border-top: 1px solid var(--td-component-stroke);
}

.ref-card h4 {
  margin: 0 0 4px;
  font-size: 13px;
}

.ref-card__meta {
  margin: 0 0 6px;
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.ref-card__content {
  margin: 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--td-text-color-primary);
  white-space: pre-wrap;
  word-break: break-word;
}

@media (max-width: 960px) {
  .chat-page {
    flex-direction: column;
    max-height: none;
  }

  .chat-sidebar,
  .chat-refs {
    width: 100%;
    max-height: 240px;
  }

  .chat-main__header {
    flex-direction: column;
  }

  .chat-main__kb {
    align-items: stretch;
    width: 100%;
  }
}
</style>
