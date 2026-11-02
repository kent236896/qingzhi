<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from '@/components/layout/AppHeader.vue'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import SettingsModal from '@/components/settings/SettingsModal.vue'
import { useAppStore } from '@/stores/app'
import { useSettingsStore } from '@/stores/settings'

const route = useRoute()
const appStore = useAppStore()
useSettingsStore()

const isChatShell = computed(() => route.name === 'chat')
let wasNarrow = typeof window !== 'undefined' ? window.innerWidth < 900 : false

function syncResponsiveCollapse(force = false) {
  if (typeof window === 'undefined') return
  const narrow = window.innerWidth < 900
  if (narrow && (force || !wasNarrow)) {
    appStore.setCollapsed(true)
  }
  wasNarrow = narrow
}

function onResize() {
  syncResponsiveCollapse()
}

onMounted(() => {
  syncResponsiveCollapse(true)
  window.addEventListener('resize', onResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', onResize)
})

watch(
  () => route.name,
  (name) => {
    if (name === 'chat') syncResponsiveCollapse(true)
  },
)
</script>

<template>
  <div class="workbench" :class="{ 'workbench--chat': isChatShell }">
    <AppSidebar />
    <div class="workbench__main">
      <AppHeader v-if="!isChatShell" />
      <main class="workbench__content">
        <RouterView />
      </main>
    </div>
    <SettingsModal />
  </div>
</template>

<style scoped>
.workbench {
  display: flex;
  min-height: 100vh;
  background: var(--wb-bg);
}

.workbench--chat {
  height: 100vh;
  max-height: 100vh;
  overflow: hidden;
  background: var(--wb-surface);
}

.workbench__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.workbench--chat .workbench__main {
  background: var(--wb-surface);
}

.workbench__content {
  flex: 1;
  padding: 20px 24px 32px;
  overflow: auto;
}

.workbench--chat .workbench__content {
  padding: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

@media (max-width: 768px) {
  .workbench__content {
    padding: 16px;
  }

  .workbench--chat .workbench__content {
    padding: 0;
  }
}
</style>
