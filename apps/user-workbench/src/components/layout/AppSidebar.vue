<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ChatIcon,
  HomeIcon,
  UserIcon,
  UserCircleIcon,
  ViewModuleIcon,
} from 'tdesign-icons-vue-next'
import { useAppStore } from '@/stores/app'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const items = [
  { path: '/chat', label: 'AI 对话', icon: ChatIcon },
  { path: '/home', label: '首页', icon: HomeIcon },
  { path: '/agents', label: '智能体', icon: ViewModuleIcon },
  { path: '/memory', label: '我的记忆', icon: UserCircleIcon },
  { path: '/profile', label: '个人中心', icon: UserIcon },
]

const activePath = computed(() => {
  if (route.path === '/' || route.path.startsWith('/chat')) return '/chat'
  const match = items.find((item) => route.path.startsWith(item.path))
  return match?.path ?? route.path
})

function go(path: string) {
  if (path !== route.path) {
    void router.push(path)
  }
}
</script>

<template>
  <aside class="sidebar" :class="{ 'sidebar--collapsed': appStore.collapsed }">
    <div class="sidebar__brand">
      <span class="sidebar__mark">知</span>
      <span v-if="!appStore.collapsed" class="sidebar__name">用户工作台</span>
    </div>
    <nav class="sidebar__nav" aria-label="主导航">
      <button
        v-for="item in items"
        :key="item.path"
        type="button"
        class="sidebar__item"
        :class="{ 'sidebar__item--active': activePath === item.path }"
        :title="item.label"
        @click="go(item.path)"
      >
        <component :is="item.icon" class="sidebar__icon" />
        <span v-if="!appStore.collapsed">{{ item.label }}</span>
      </button>
    </nav>
  </aside>
</template>

<style scoped>
.sidebar {
  width: var(--wb-sidebar-width);
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--wb-surface);
  border-right: 1px solid var(--wb-border);
  transition: width 0.15s ease;
}

.sidebar--collapsed {
  width: 64px;
}

.sidebar__brand {
  height: var(--wb-header-height);
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 16px;
  border-bottom: 1px solid var(--wb-border);
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
  font-size: 15px;
  font-weight: 600;
  white-space: nowrap;
}

.sidebar__nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 8px;
}

.sidebar__item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  border: 0;
  background: transparent;
  color: var(--wb-text);
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  text-align: left;
  font-size: 14px;
}

.sidebar__item:hover {
  background: #f3f3f3;
}

.sidebar__item--active {
  background: #ecf2fe;
  color: var(--wb-primary);
  font-weight: 600;
}

.sidebar__icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

@media (max-width: 768px) {
  .sidebar:not(.sidebar--collapsed) {
    position: absolute;
    z-index: 20;
    height: 100%;
    box-shadow: 0 0 0 1px var(--wb-border);
  }
}
</style>
