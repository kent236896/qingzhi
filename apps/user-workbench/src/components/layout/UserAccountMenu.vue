<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  ChevronUpIcon,
  LogoutIcon,
  SecuredIcon,
  UserCircleIcon,
  UserIcon,
} from 'tdesign-icons-vue-next'
import { MessagePlugin } from 'tdesign-vue-next'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import type { SettingsSectionId } from '@/settings/registry'

const props = defineProps<{
  collapsed?: boolean
}>()

const router = useRouter()
const appStore = useAppStore()
const auth = useAuthStore()
const settings = useSettingsStore()

const menuOpen = ref(false)
const isNarrow = ref(false)
const rootEl = ref<HTMLElement | null>(null)

const displayName = computed(() => auth.displayName)
const accountLine = computed(() => auth.user?.username || auth.user?.id || '—')
const avatarLetter = computed(() => (displayName.value || '?').slice(0, 1))
const roleLabel = computed(() => {
  const permissions = auth.permissions || []
  if (permissions.some((item) => item.startsWith('system:') || item === 'knowledge:manage')) {
    return '管理员'
  }
  return '成员'
})

function syncNarrow() {
  isNarrow.value = window.innerWidth < 768
}

function toggleMenu() {
  if (props.collapsed) {
    appStore.setCollapsed(false)
  }
  menuOpen.value = !menuOpen.value
}

function closeMenu() {
  menuOpen.value = false
}

async function openSettings(section: SettingsSectionId) {
  closeMenu()
  await settings.openSettings(section)
}

async function goMemory() {
  closeMenu()
  await router.push({ name: 'memory' })
}

function onLogout() {
  closeMenu()
  auth.logout()
  void MessagePlugin.success('已退出登录')
  void router.replace({ name: 'login' })
}

function onDocPointer(event: MouseEvent) {
  if (!menuOpen.value || !rootEl.value) return
  if (!rootEl.value.contains(event.target as Node)) {
    closeMenu()
  }
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') closeMenu()
}

onMounted(() => {
  syncNarrow()
  window.addEventListener('resize', syncNarrow)
  document.addEventListener('mousedown', onDocPointer)
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  window.removeEventListener('resize', syncNarrow)
  document.removeEventListener('mousedown', onDocPointer)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div ref="rootEl" class="user-menu" :class="{ 'user-menu--collapsed': collapsed }">
    <Transition :name="isNarrow ? 'sheet' : 'menu'">
      <div
        v-if="menuOpen"
        class="user-menu__panel"
        :class="{ 'user-menu__panel--sheet': isNarrow }"
        role="menu"
      >
        <div class="user-menu__head">
          <span class="user-menu__avatar user-menu__avatar--lg">
            <img v-if="auth.user?.avatar" :src="auth.user.avatar" alt="" />
            <template v-else>{{ avatarLetter }}</template>
          </span>
          <div class="user-menu__identity">
            <div class="user-menu__name">{{ displayName }}</div>
            <div class="user-menu__email">{{ accountLine }}</div>
          </div>
        </div>

        <div class="user-menu__divider" />

        <div class="user-menu__role">
          <SecuredIcon class="user-menu__role-icon" />
          <span>{{ roleLabel }}</span>
        </div>

        <div class="user-menu__divider" />

        <button type="button" class="user-menu__item" role="menuitem" @click="openSettings('profile')">
          <UserIcon />
          <span>个人设置</span>
        </button>
        <button type="button" class="user-menu__item" role="menuitem" @click="goMemory">
          <UserCircleIcon />
          <span>我的记忆</span>
        </button>

        <div class="user-menu__divider" />

        <button
          type="button"
          class="user-menu__item user-menu__item--danger"
          role="menuitem"
          @click="onLogout"
        >
          <LogoutIcon />
          <span>退出</span>
        </button>
      </div>
    </Transition>

    <div v-if="menuOpen && isNarrow" class="user-menu__sheet-mask" @click="closeMenu" />

    <button
      type="button"
      class="user-menu__card"
      :class="{ 'user-menu__card--active': menuOpen }"
      :title="displayName"
      @click="toggleMenu"
    >
      <span class="user-menu__avatar">
        <img v-if="auth.user?.avatar" :src="auth.user.avatar" alt="" />
        <template v-else>{{ avatarLetter }}</template>
      </span>
      <span v-if="!collapsed" class="user-menu__meta">
        <span class="user-menu__name">{{ displayName }}</span>
        <span class="user-menu__email">{{ accountLine }}</span>
      </span>
      <ChevronUpIcon
        v-if="!collapsed"
        class="user-menu__chevron"
        :class="{ 'user-menu__chevron--open': menuOpen }"
      />
    </button>
  </div>
</template>

<style scoped>
.user-menu {
  position: relative;
  flex-shrink: 0;
  padding: 10px 12px;
  border-top: 1px solid var(--wb-border);
}

.user-menu__card {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  padding: 8px;
  cursor: pointer;
  text-align: left;
  color: var(--wb-text);
}

.user-menu__card:hover,
.user-menu__card--active {
  background: var(--wb-hover);
}

.user-menu--collapsed .user-menu__card {
  justify-content: center;
  padding: 8px 0;
}

.user-menu__avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--wb-primary);
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
  flex-shrink: 0;
  overflow: hidden;
}

.user-menu__avatar--lg {
  width: 40px;
  height: 40px;
  font-size: 16px;
}

.user-menu__avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.user-menu__meta {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.user-menu__name {
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-menu__email {
  font-size: 12px;
  color: var(--wb-text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-menu__chevron {
  width: 16px;
  height: 16px;
  color: var(--wb-text-secondary);
  flex-shrink: 0;
  transition: transform 0.15s ease;
}

.user-menu__chevron--open {
  transform: rotate(180deg);
  color: var(--wb-primary);
}

.user-menu__panel {
  position: absolute;
  left: 12px;
  right: 12px;
  bottom: calc(100% - 4px);
  z-index: 40;
  width: auto;
  min-width: 0;
  max-width: 280px;
  margin: 0 auto;
  background: var(--wb-surface);
  border: 1px solid var(--wb-border);
  border-radius: 12px;
  box-shadow: 0 10px 28px color-mix(in srgb, var(--wb-text) 12%, transparent);
  padding: 8px;
}

.user-menu--collapsed .user-menu__panel {
  left: calc(100% + 8px);
  right: auto;
  bottom: 12px;
  width: 280px;
}

.user-menu__head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px;
}

.user-menu__identity {
  min-width: 0;
}

.user-menu__divider {
  height: 1px;
  background: var(--wb-border);
  margin: 4px 0;
}

.user-menu__role {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  font-size: 13px;
  color: var(--wb-text);
}

.user-menu__role-icon {
  width: 16px;
  height: 16px;
  color: var(--wb-text-secondary);
}

.user-menu__item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 40px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  padding: 8px 10px;
  font-size: 14px;
  color: var(--wb-text);
  cursor: pointer;
  text-align: left;
}

.user-menu__item:hover {
  background: var(--wb-hover);
}

.user-menu__item--danger {
  color: var(--td-error-color, #d54941);
}

.user-menu__sheet-mask {
  position: fixed;
  inset: 0;
  z-index: 35;
  background: color-mix(in srgb, var(--wb-text) 35%, transparent);
}

.user-menu__panel--sheet {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 40;
  max-width: none;
  width: 100%;
  margin: 0;
  border-radius: 16px 16px 0 0;
  padding: 12px 12px calc(12px + env(safe-area-inset-bottom));
}

.menu-enter-active,
.menu-leave-active {
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}

.menu-enter-from,
.menu-leave-to {
  opacity: 0;
  transform: translateY(6px);
}

.sheet-enter-active,
.sheet-leave-active {
  transition:
    opacity 0.18s ease,
    transform 0.18s ease;
}

.sheet-enter-from,
.sheet-leave-to {
  opacity: 0;
  transform: translateY(16px);
}
</style>
