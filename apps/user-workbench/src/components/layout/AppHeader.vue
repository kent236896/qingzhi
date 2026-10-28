<script setup lang="ts">
import { MenuFoldIcon, MenuUnfoldIcon } from 'tdesign-icons-vue-next'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const appStore = useAppStore()
const auth = useAuthStore()

function goProfile() {
  void router.push('/profile')
}

function onLogout() {
  // Backend has no logout API for JWT; clear browser session only.
  auth.logout()
  void MessagePlugin.success('已退出登录')
  void router.replace({ name: 'login' })
}
</script>

<template>
  <header class="header">
    <div class="header__left">
      <button
        type="button"
        class="header__icon-btn"
        :aria-label="appStore.collapsed ? '展开导航' : '收起导航'"
        @click="appStore.toggleCollapsed()"
      >
        <MenuUnfoldIcon v-if="appStore.collapsed" />
        <MenuFoldIcon v-else />
      </button>
      <div class="header__titles">
        <span class="header__crumb">用户工作台</span>
        <span class="header__sep">/</span>
        <h1 class="header__title">{{ appStore.pageTitle }}</h1>
      </div>
    </div>
    <div class="header__right">
      <button type="button" class="header__user" title="个人中心" @click="goProfile">
        <span class="header__avatar">
          <img v-if="auth.user?.avatar" :src="auth.user.avatar" alt="" />
          <template v-else>{{ (auth.displayName || '?').slice(0, 1) }}</template>
        </span>
        <span class="header__name">{{ auth.displayName }}</span>
      </button>
      <button type="button" class="header__logout" @click="onLogout">退出登录</button>
    </div>
  </header>
</template>

<style scoped>
.header {
  height: var(--wb-header-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 20px;
  background: var(--wb-surface);
  border-bottom: 1px solid var(--wb-border);
}

.header__left,
.header__right {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.header__icon-btn {
  border: 0;
  background: transparent;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--wb-text);
}

.header__icon-btn:hover {
  background: #f3f3f3;
}

.header__titles {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.header__crumb,
.header__sep {
  color: var(--wb-text-secondary);
  font-size: 13px;
}

.header__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.header__user {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 6px;
}

.header__user:hover {
  background: #f3f3f3;
}

.header__avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #e7e7e7;
  color: #666;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  flex-shrink: 0;
  overflow: hidden;
}

.header__avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.header__name {
  font-size: 13px;
  color: var(--wb-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 180px;
}

.header__logout {
  border: 1px solid var(--wb-border);
  background: #fff;
  color: var(--wb-text);
  border-radius: 6px;
  padding: 6px 12px;
  font-size: 13px;
  cursor: pointer;
}

.header__logout:hover {
  border-color: var(--wb-primary);
  color: var(--wb-primary);
}

@media (max-width: 640px) {
  .header__crumb,
  .header__sep {
    display: none;
  }

  .header__name {
    display: none;
  }
}
</style>
