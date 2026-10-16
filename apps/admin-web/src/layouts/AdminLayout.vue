<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MenuFoldIcon, MenuUnfoldIcon } from 'tdesign-icons-vue-next'
import UserAvatar from '@/components/UserAvatar.vue'
import { fetchAuthMenus } from '@/api/auth'
import type { MenuNode } from '@/api/identity'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const auth = useAuthStore()

const menus = ref<MenuNode[]>([])
const collapsed = ref(false)
const expanded = ref<string[]>([])
const menuStatus = ref<'loading' | 'error' | 'ready'>('loading')
const breadcrumbs = computed(() => route.meta.breadcrumb ?? [])
const showBreadcrumb = computed(() => breadcrumbs.value.length > 1)
const pageTitle = computed(() => (typeof route.meta.title === 'string' ? route.meta.title : ''))
const activeMenu = computed(() => route.path || '/')

function visibleTree(nodes: MenuNode[]): MenuNode[] {
  return nodes
    .filter((node) => node.status === 0 && node.visible && node.type !== 3)
    .map((node) => ({ ...node, children: visibleTree(node.children ?? []) }))
}

function directoryIds(nodes: MenuNode[]): string[] {
  return nodes.flatMap((node) => (node.type === 1 ? [node.id, ...directoryIds(node.children ?? [])] : []))
}

function routePath(path: string): string {
  if (!path || path === '/') {
    return '/'
  }
  return path.startsWith('/') ? path : `/${path}`
}

function pagesOf(nodes: MenuNode[] | undefined): MenuNode[] {
  return (nodes ?? []).filter((node) => node.type === 2 && node.path)
}

function directoriesOf(nodes: MenuNode[] | undefined): MenuNode[] {
  return (nodes ?? []).filter((node) => node.type === 1)
}

async function loadMenus() {
  if (menuStatus.value !== 'ready') {
    menuStatus.value = 'loading'
  }
  try {
    const response = await fetchAuthMenus()
    menus.value = visibleTree(response.data ?? [])
    if (expanded.value.length === 0) {
      expanded.value = directoryIds(menus.value)
    }
    menuStatus.value = 'ready'
  } catch {
    if (menuStatus.value !== 'ready') {
      menuStatus.value = 'error'
    }
  }
}

function logout() {
  auth.logout()
  void router.replace({ name: 'login' })
}

function onMenuChange(value: string | number) {
  const path = String(value)
  if (path.startsWith('/')) {
    void router.push(path)
  }
}

onMounted(() => {
  void loadMenus()
})

watch(
  () => route.name,
  (_name, previous) => {
    if (previous === 'menus') {
      void loadMenus()
    }
  },
)
</script>

<template>
  <t-layout class="admin-shell">
    <t-aside
      class="admin-aside"
      :class="{ 'is-collapsed': collapsed }"
      :width="collapsed ? 'var(--qz-aside-collapsed)' : 'var(--qz-aside-width)'"
    >
      <div class="brand">
        <img src="/favicon.svg" width="32" height="32" alt="" />
        <div v-show="!collapsed">
          <h1>{{ appStore.title }}</h1>
          <p>管理控制台</p>
        </div>
      </div>
      <p v-if="menuStatus === 'loading'" class="menu-state">正在加载菜单</p>
      <p v-else-if="menuStatus === 'error'" class="menu-state">
        菜单加载失败
        <t-button theme="primary" variant="text" @click="loadMenus">重试</t-button>
      </p>
      <t-menu
        v-else
        v-model:expanded="expanded"
        class="admin-menu"
        theme="light"
        :collapsed="collapsed"
        :value="activeMenu"
        @change="onMenuChange"
      >
        <template v-for="level1 in menus" :key="level1.id">
          <t-submenu v-if="level1.type === 1" :value="level1.id" :title="level1.name">
            <template #icon>
              <span v-if="collapsed" class="menu-mark">{{ level1.name.slice(0, 1) }}</span>
            </template>
            <template v-for="level2 in directoriesOf(level1.children)" :key="level2.id">
              <t-submenu :value="level2.id" :title="level2.name">
                <t-menu-item v-for="level3 in pagesOf(level2.children)" :key="level3.id" :value="routePath(level3.path)">
                  {{ level3.name }}
                </t-menu-item>
                <t-menu-item v-if="pagesOf(level2.children).length === 0" :value="`empty:${level2.id}`" disabled>
                  暂无功能
                </t-menu-item>
              </t-submenu>
            </template>
            <t-menu-item v-for="level2 in pagesOf(level1.children)" :key="level2.id" :value="routePath(level2.path)">
              {{ level2.name }}
            </t-menu-item>
            <t-menu-item
              v-if="directoriesOf(level1.children).length === 0 && pagesOf(level1.children).length === 0"
              :value="`empty:${level1.id}`"
              disabled
            >
              暂无功能
            </t-menu-item>
          </t-submenu>
          <t-menu-item v-else-if="level1.type === 2 && level1.path" :value="routePath(level1.path)">
            <template #icon>
              <span v-if="collapsed" class="menu-mark">{{ level1.name.slice(0, 1) }}</span>
            </template>
            {{ level1.name }}
          </t-menu-item>
        </template>
      </t-menu>
      <button class="aside-toggle" type="button" :aria-label="collapsed ? '展开导航' : '收起导航'" @click="collapsed = !collapsed">
        <MenuUnfoldIcon v-if="collapsed" />
        <MenuFoldIcon v-else />
        <span v-show="!collapsed">收起导航</span>
      </button>
    </t-aside>
    <t-layout>
      <t-header class="admin-header" :height="'var(--qz-header-height)'">
        <t-breadcrumb v-if="showBreadcrumb">
          <t-breadcrumb-item v-for="item in breadcrumbs" :key="item">{{ item }}</t-breadcrumb-item>
        </t-breadcrumb>
        <span v-else class="header-placeholder"></span>
        <div class="user-slot">
          <UserAvatar size="small" />
          <span>{{ auth.displayName }}</span>
          <t-button theme="default" variant="text" size="small" @click="logout">退出</t-button>
        </div>
      </t-header>
      <t-content class="admin-content">
        <div class="page-shell">
          <header v-if="pageTitle" class="page-shell__head">
            <h2>{{ pageTitle }}</h2>
          </header>
          <router-view />
        </div>
      </t-content>
    </t-layout>
  </t-layout>
</template>
