<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  AddIcon,
  ChatIcon,
  DeleteIcon,
  EditIcon,
  FileIcon,
  Folder1Icon,
  HomeIcon,
  MenuFoldIcon,
  MenuUnfoldIcon,
  SearchIcon,
  ToolsIcon,
  ViewModuleIcon,
} from 'tdesign-icons-vue-next'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import UserAccountMenu from '@/components/layout/UserAccountMenu.vue'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { groupSessionsByRecency } from '@/utils/chatUi'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const chat = useChatStore()

const searchOpen = ref(false)
const searchInput = ref<HTMLInputElement | null>(null)

const isChatRoute = computed(() => route.name === 'chat')

const navItems = [
  { key: 'new-chat', path: '/chat', label: '新对话', icon: AddIcon, action: 'new-chat' as const },
  { key: 'home', path: '/home', label: '首页', icon: HomeIcon, action: 'nav' as const },
  { key: 'knowledge', path: '/knowledge', label: '知识库', icon: Folder1Icon, action: 'nav' as const },
  { key: 'artifacts', path: '/artifacts', label: '产物', icon: FileIcon, action: 'nav' as const },
  { key: 'agents', path: '/agents', label: '智能体', icon: ViewModuleIcon, action: 'nav' as const },
  { key: 'toolbox', path: '/toolbox', label: '工具箱', icon: ToolsIcon, action: 'nav' as const },
]

const sessionGroups = computed(() => groupSessionsByRecency(chat.filteredSessions))

const newChatActive = computed(
  () => isChatRoute.value && (!chat.activeSessionId || chat.messages.length === 0),
)

function isNavActive(item: (typeof navItems)[number]) {
  if (item.action === 'new-chat') return newChatActive.value
  return route.path.startsWith(item.path)
}

async function onNav(item: (typeof navItems)[number]) {
  if (item.action === 'new-chat') {
    await onNewChat()
    return
  }
  if (route.path !== item.path) {
    void router.push(item.path)
  }
}

async function onNewChat() {
  try {
    if (route.name !== 'chat') {
      await router.push({ name: 'chat' })
    }
    // Draft only: bind agent_id on first ask via createSession.
    await chat.startDraft()
    searchOpen.value = false
  } catch (error) {
    void MessagePlugin.error(error instanceof Error ? error.message : '创建会话失败')
  }
}

async function onOpenSession(id: string) {
  try {
    if (route.name !== 'chat') {
      await router.push({ name: 'chat' })
    }
    await chat.openSession(id)
  } catch {
    /* chat store keeps errorMessage */
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

function toggleSearch() {
  if (appStore.collapsed) {
    appStore.setCollapsed(false)
  }
  searchOpen.value = !searchOpen.value
  if (searchOpen.value) {
    requestAnimationFrame(() => searchInput.value?.focus())
  } else {
    chat.sessionFilter = ''
  }
}

watch(isChatRoute, (onChat) => {
  if (onChat) {
    void chat.bootstrap().catch(() => undefined)
  }
})

onMounted(() => {
  if (isChatRoute.value) {
    void chat.bootstrap().catch(() => undefined)
  }
})
</script>

<template>
  <aside class="sidebar" :class="{ 'sidebar--collapsed': appStore.collapsed }">
    <div class="sidebar__brand">
      <span class="sidebar__mark" title="用户工作台">知</span>
      <span v-if="!appStore.collapsed" class="sidebar__name">用户工作台</span>
      <div v-if="!appStore.collapsed" class="sidebar__brand-actions">
        <button
          type="button"
          class="sidebar__icon-btn"
          title="搜索会话"
          :aria-pressed="searchOpen"
          @click="toggleSearch"
        >
          <SearchIcon />
        </button>
        <button
          type="button"
          class="sidebar__icon-btn"
          title="收起侧边栏"
          @click="appStore.setCollapsed(true)"
        >
          <MenuFoldIcon />
        </button>
      </div>
      <button
        v-else
        type="button"
        class="sidebar__icon-btn"
        title="展开侧边栏"
        @click="appStore.setCollapsed(false)"
      >
        <MenuUnfoldIcon />
      </button>
    </div>

    <div v-if="searchOpen && !appStore.collapsed && isChatRoute" class="sidebar__search">
      <input
        ref="searchInput"
        v-model="chat.sessionFilter"
        type="search"
        class="sidebar__search-input"
        placeholder="搜索会话标题"
        @keydown.escape="toggleSearch"
      />
    </div>

    <nav class="sidebar__nav" aria-label="主导航">
      <button
        v-for="item in navItems"
        :key="item.key"
        type="button"
        class="sidebar__item"
        :class="{ 'sidebar__item--active': isNavActive(item) }"
        :title="item.label"
        @click="onNav(item)"
      >
        <component :is="item.icon" class="sidebar__icon" />
        <span v-if="!appStore.collapsed">{{ item.label }}</span>
      </button>
    </nav>

    <div class="sidebar__history">
      <template v-if="isChatRoute && !appStore.collapsed">
        <p v-if="chat.loadingSessions && chat.sessions.length === 0" class="sidebar__hint">
          加载会话中…
        </p>
        <p v-else-if="chat.filteredSessions.length === 0" class="sidebar__hint">暂无会话</p>
        <div v-else class="sidebar__history-scroll">
          <section v-for="group in sessionGroups" :key="group.key" class="sidebar__group">
            <h3 class="sidebar__group-title">{{ group.label }}</h3>
            <button
              v-for="item in group.items"
              :key="item.id"
              type="button"
              class="session-row"
              :class="{ 'session-row--active': item.id === chat.activeSessionId && chat.messages.length > 0 }"
              :title="item.title || '未命名会话'"
              @click="onOpenSession(item.id)"
            >
              <span class="session-row__title">
                <ChatIcon class="session-row__icon" />
                {{ item.title || '未命名会话' }}
              </span>
              <span class="session-row__actions" @click.stop>
                <button
                  type="button"
                  class="session-row__action"
                  title="重命名"
                  @click="onRename(item.id, item.title)"
                >
                  <EditIcon />
                </button>
                <button
                  type="button"
                  class="session-row__action session-row__action--danger"
                  title="删除"
                  @click="onDelete(item.id)"
                >
                  <DeleteIcon />
                </button>
              </span>
            </button>
          </section>
        </div>
      </template>
    </div>

    <UserAccountMenu :collapsed="appStore.collapsed" />
  </aside>
</template>

<style scoped>
.sidebar {
  width: var(--wb-sidebar-width);
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--wb-sidebar-bg);
  border-right: 1px solid var(--wb-border);
  transition: width 0.15s ease;
  min-height: 0;
}

.sidebar--collapsed {
  width: 64px;
}

.sidebar__brand {
  height: var(--wb-header-height);
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  flex-shrink: 0;
}

.sidebar__mark {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  background: var(--wb-primary);
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
  flex-shrink: 0;
}

.sidebar__name {
  flex: 1;
  min-width: 0;
  font-size: 15px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sidebar__brand-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-left: auto;
}

.sidebar__icon-btn {
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
  flex-shrink: 0;
}

.sidebar__icon-btn:hover {
  background: var(--wb-hover);
  color: var(--wb-text);
}

.sidebar__search {
  padding: 0 12px 8px;
}

.sidebar__search-input {
  width: 100%;
  height: 32px;
  border: 1px solid var(--wb-border);
  border-radius: 8px;
  padding: 0 10px;
  background: var(--wb-surface);
  color: var(--wb-text);
  font-size: 13px;
  outline: none;
}

.sidebar__search-input:focus {
  border-color: var(--wb-primary);
}

.sidebar__nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 4px 12px 12px;
  flex-shrink: 0;
}

.sidebar__item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 40px;
  border: 0;
  background: transparent;
  color: var(--wb-text);
  padding: 8px 10px;
  border-radius: 8px;
  cursor: pointer;
  text-align: left;
  font-size: 14px;
}

.sidebar__item:hover {
  background: var(--wb-hover);
}

.sidebar__item--active {
  background: var(--wb-primary-light);
  color: var(--wb-primary);
  font-weight: 600;
}

.sidebar__icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.sidebar--collapsed .sidebar__item {
  justify-content: center;
  padding: 8px;
}

.sidebar__history {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0 8px 8px;
}

.sidebar__hint {
  margin: 4px 8px;
  font-size: 12px;
  color: var(--wb-text-secondary);
}

.sidebar__history-scroll {
  flex: 1;
  overflow: auto;
  padding: 0 4px;
}

.sidebar__group {
  margin-bottom: 12px;
}

.sidebar__group-title {
  margin: 0 8px 4px;
  font-size: 12px;
  font-weight: 500;
  color: var(--wb-text-secondary);
}

.session-row {
  position: relative;
  display: flex;
  align-items: center;
  width: 100%;
  border: 0;
  background: transparent;
  border-radius: 8px;
  padding: 8px 10px;
  cursor: pointer;
  text-align: left;
  color: var(--wb-text);
  font-size: 13px;
}

.session-row:hover {
  background: var(--wb-hover);
}

.session-row--active {
  background: var(--wb-primary-light);
  color: var(--wb-primary);
}

.session-row__title {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.session-row__icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  opacity: 0.65;
}

.session-row__actions {
  display: none;
  align-items: center;
  gap: 0;
  flex-shrink: 0;
}

.session-row:hover .session-row__actions {
  display: inline-flex;
}

.session-row__action {
  border: 0;
  background: transparent;
  width: 24px;
  height: 24px;
  border-radius: 4px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--wb-text-secondary);
}

.session-row__action:hover {
  background: var(--wb-surface);
  color: var(--wb-text);
}

.session-row__action--danger:hover {
  color: var(--td-error-color, #d54941);
}

@media (max-width: 900px) {
  .sidebar:not(.sidebar--collapsed) {
    position: absolute;
    z-index: 30;
    height: 100%;
  }
}
</style>
