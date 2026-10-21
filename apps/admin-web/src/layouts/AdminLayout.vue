<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MenuFoldIcon, MenuUnfoldIcon } from 'tdesign-icons-vue-next'
import UserAvatar from '@/components/UserAvatar.vue'
import { changeOwnPassword, fetchAuthMenus, fetchOwnProfile, updateOwnProfile } from '@/api/auth'
import type { MenuNode } from '@/api/identity'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { feedback } from '@/utils/feedback'

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

const profileVisible = ref(false)
const passwordVisible = ref(false)
const profileSaving = ref(false)
const passwordSaving = ref(false)
const profileForm = ref({ nickname: '', email: '', mobile: '', sex: 0, avatar: '', remark: '' })
const passwordForm = ref({ current: '', next: '', confirm: '' })

async function openProfile() {
  profileVisible.value = true
  const response = await fetchOwnProfile()
  const user = response.data.user
  if (response.data.version) {
    auth.version = response.data.version
  }
  profileForm.value = {
    nickname: user.nickname,
    email: user.email,
    mobile: user.mobile,
    sex: user.sex,
    avatar: user.avatar,
    remark: user.remark,
  }
}

async function saveProfile() {
  profileSaving.value = true
  try {
    const response = await updateOwnProfile(profileForm.value)
    auth.applyProfile(response.data)
    feedback.success('个人信息已保存')
    profileVisible.value = false
  } finally {
    profileSaving.value = false
  }
}

function openPassword() {
  passwordForm.value = { current: '', next: '', confirm: '' }
  passwordVisible.value = true
}

async function savePassword() {
  if (!passwordForm.value.current || !passwordForm.value.next) {
    feedback.warning('请填写当前密码和新密码')
    return
  }
  if (passwordForm.value.next !== passwordForm.value.confirm) {
    feedback.warning('两次输入的新密码不一致')
    return
  }
  passwordSaving.value = true
  try {
    await changeOwnPassword({
      current_password: passwordForm.value.current,
      new_password: passwordForm.value.next,
    })
    feedback.success('密码已修改')
    passwordVisible.value = false
  } finally {
    passwordSaving.value = false
  }
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
        <t-button variant="text" @click="loadMenus">重试</t-button>
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
        <t-dropdown trigger="click" placement="bottom-right" :min-column-width="168">
          <button class="user-trigger" type="button" aria-label="账号菜单">
            <UserAvatar :image="auth.user?.avatar" size="small" />
            <span>{{ auth.displayName }}</span>
          </button>
          <t-dropdown-menu>
            <t-dropdown-item disabled>系统版本 {{ auth.version || '0.1.0' }}</t-dropdown-item>
            <t-dropdown-item @click="openProfile">修改个人信息</t-dropdown-item>
            <t-dropdown-item @click="openPassword">修改密码</t-dropdown-item>
            <t-dropdown-item @click="logout">退出</t-dropdown-item>
          </t-dropdown-menu>
        </t-dropdown>
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

  <t-dialog v-model:visible="profileVisible" header="修改个人信息" :confirm-btn="{ loading: profileSaving, content: '保存' }" @confirm="saveProfile">
    <t-form label-align="top">
      <t-form-item label="用户名"><t-input :value="auth.user?.username" disabled /></t-form-item>
      <t-form-item label="昵称"><t-input v-model="profileForm.nickname" /></t-form-item>
      <t-form-item label="邮箱"><t-input v-model="profileForm.email" /></t-form-item>
      <t-form-item label="手机"><t-input v-model="profileForm.mobile" /></t-form-item>
      <t-form-item label="性别">
        <t-radio-group v-model="profileForm.sex">
          <t-radio :value="0">未知</t-radio>
          <t-radio :value="1">男</t-radio>
          <t-radio :value="2">女</t-radio>
        </t-radio-group>
      </t-form-item>
      <t-form-item label="头像地址"><t-input v-model="profileForm.avatar" /></t-form-item>
      <t-form-item label="备注"><t-textarea v-model="profileForm.remark" /></t-form-item>
    </t-form>
  </t-dialog>

  <t-dialog v-model:visible="passwordVisible" header="修改密码" :confirm-btn="{ loading: passwordSaving, content: '保存' }" @confirm="savePassword">
    <t-form label-align="top">
      <t-form-item label="当前密码"><t-input v-model="passwordForm.current" type="password" /></t-form-item>
      <t-form-item label="新密码"><t-input v-model="passwordForm.next" type="password" /></t-form-item>
      <t-form-item label="确认新密码"><t-input v-model="passwordForm.confirm" type="password" /></t-form-item>
    </t-form>
  </t-dialog>
</template>
