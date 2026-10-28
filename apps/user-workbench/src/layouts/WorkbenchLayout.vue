<script setup lang="ts">
import AppHeader from '@/components/layout/AppHeader.vue'
import AppSidebar from '@/components/layout/AppSidebar.vue'
</script>

<template>
  <div class="workbench">
    <AppSidebar />
    <div class="workbench__main">
      <AppHeader />
      <main class="workbench__content">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<style scoped>
.workbench {
  display: flex;
  min-height: 100vh;
  background: var(--wb-bg);
}

.workbench__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.workbench__content {
  flex: 1;
  padding: 20px 24px 32px;
  overflow: auto;
}

@media (max-width: 768px) {
  .workbench__content {
    padding: 16px;
  }
}
</style>
