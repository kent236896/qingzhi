<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <header class="settings-page-header">
    <h2>{{ title }}</h2>
    <p v-if="description">{{ description }}</p>
  </header>
</template>

<style scoped>
.settings-page-header {
  margin-bottom: 32px;
}

.settings-page-header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  line-height: 1.3;
  color: var(--wb-text);
}

.settings-page-header p {
  margin: 8px 0 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--wb-text-secondary);
}
</style>
