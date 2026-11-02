<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ChevronLeftIcon, CloseIcon } from 'tdesign-icons-vue-next'
import {
  SETTINGS_NAV_GROUPS,
  findSettingsItem,
  type SettingsSectionId,
} from '@/settings/registry'
import { useSettingsStore } from '@/stores/settings'

const settings = useSettingsStore()
const isNarrow = ref(false)
const mobileShowContent = ref(false)

const activeItem = computed(() => findSettingsItem(settings.section))

function syncNarrow() {
  isNarrow.value = window.innerWidth < 768
  if (!isNarrow.value) mobileShowContent.value = true
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && settings.open) {
    void settings.closeSettings()
  }
}

function selectSection(id: SettingsSectionId) {
  void settings.setSection(id)
  if (isNarrow.value) mobileShowContent.value = true
}

function onBackdropClick(event: MouseEvent) {
  if (event.target === event.currentTarget) {
    void settings.closeSettings()
  }
}

watch(
  () => settings.open,
  (open) => {
    if (open) {
      document.body.style.overflow = 'hidden'
      if (isNarrow.value) mobileShowContent.value = false
    } else {
      document.body.style.overflow = ''
    }
  },
)

onMounted(() => {
  syncNarrow()
  window.addEventListener('resize', syncNarrow)
  window.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  window.removeEventListener('resize', syncNarrow)
  window.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})
</script>

<template>
  <Teleport to="body">
    <Transition name="settings-fade">
      <div
        v-if="settings.open"
        class="settings-overlay"
        :class="{ 'settings-overlay--narrow': isNarrow }"
        @click="onBackdropClick"
      >
        <div
          class="settings-modal"
          role="dialog"
          aria-modal="true"
          aria-label="设置"
          @click.stop
        >
          <button type="button" class="settings-modal__close" title="关闭" @click="settings.closeSettings()">
            <CloseIcon />
          </button>

          <aside
            v-show="!isNarrow || !mobileShowContent"
            class="settings-modal__nav"
          >
            <h2 class="settings-modal__nav-title">设置</h2>
            <div v-for="group in SETTINGS_NAV_GROUPS" :key="group.id" class="settings-modal__group">
              <div class="settings-modal__group-title">{{ group.title }}</div>
              <button
                v-for="item in group.items"
                :key="item.id"
                type="button"
                class="settings-modal__nav-item"
                :class="{ 'settings-modal__nav-item--active': settings.section === item.id }"
                @click="selectSection(item.id)"
              >
                <component :is="item.icon" class="settings-modal__nav-icon" />
                <span>{{ item.title }}</span>
              </button>
            </div>
          </aside>

          <section
            v-show="!isNarrow || mobileShowContent"
            class="settings-modal__content"
          >
            <button
              v-if="isNarrow"
              type="button"
              class="settings-modal__back"
              @click="mobileShowContent = false"
            >
              <ChevronLeftIcon />
              返回设置
            </button>
            <component :is="activeItem?.component" v-if="activeItem" :key="activeItem.id" />
          </section>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.settings-overlay {
  position: fixed;
  inset: 0;
  z-index: 1200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: color-mix(in srgb, var(--wb-text) 45%, transparent);
}

.settings-modal {
  position: relative;
  width: min(1000px, 90vw);
  height: min(720px, 85vh);
  display: flex;
  border-radius: 16px;
  overflow: hidden;
  background: var(--wb-surface);
  border: 1px solid var(--wb-border);
  box-shadow: 0 16px 48px color-mix(in srgb, var(--wb-text) 18%, transparent);
}

.settings-modal__close {
  position: absolute;
  top: 14px;
  right: 14px;
  z-index: 2;
  width: 32px;
  height: 32px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--wb-text-secondary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.settings-modal__close:hover {
  background: var(--wb-hover);
  color: var(--wb-text);
}

.settings-modal__nav {
  width: 240px;
  flex-shrink: 0;
  background: var(--wb-sidebar-bg);
  border-right: 1px solid var(--wb-border);
  padding: 20px 12px;
  overflow: auto;
}

.settings-modal__nav-title {
  margin: 0 8px 16px;
  font-size: 18px;
  font-weight: 700;
  color: var(--wb-text);
}

.settings-modal__group + .settings-modal__group {
  margin-top: 16px;
}

.settings-modal__group-title {
  margin: 0 8px 6px;
  font-size: 12px;
  color: var(--wb-text-secondary);
}

.settings-modal__nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 40px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--wb-text);
  padding: 8px 10px;
  font-size: 14px;
  cursor: pointer;
  text-align: left;
}

.settings-modal__nav-item:hover {
  background: var(--wb-hover);
}

.settings-modal__nav-item--active {
  background: var(--wb-primary-light);
  color: var(--wb-primary);
  font-weight: 600;
}

.settings-modal__nav-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.settings-modal__content {
  flex: 1;
  min-width: 0;
  overflow: auto;
  padding: 32px 40px;
  background: var(--wb-surface);
}

.settings-modal__back {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: 0;
  background: transparent;
  color: var(--wb-primary);
  font-size: 13px;
  cursor: pointer;
  margin: 0 0 16px;
  padding: 0;
}

.settings-overlay--narrow {
  padding: 0;
}

.settings-overlay--narrow .settings-modal {
  width: 100vw;
  height: 100vh;
  border-radius: 0;
  border: 0;
}

.settings-overlay--narrow .settings-modal__nav,
.settings-overlay--narrow .settings-modal__content {
  width: 100%;
  border-right: 0;
}

.settings-fade-enter-active,
.settings-fade-leave-active {
  transition: opacity 0.18s ease;
}

.settings-fade-enter-active .settings-modal,
.settings-fade-leave-active .settings-modal {
  transition:
    transform 0.18s ease,
    opacity 0.18s ease;
}

.settings-fade-enter-from,
.settings-fade-leave-to {
  opacity: 0;
}

.settings-fade-enter-from .settings-modal,
.settings-fade-leave-to .settings-modal {
  opacity: 0;
  transform: translateY(8px) scale(0.98);
}
</style>
