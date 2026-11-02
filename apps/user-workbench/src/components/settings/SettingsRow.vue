<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<template>
  <div class="settings-row">
    <div class="settings-row__meta">
      <div class="settings-row__title">{{ title }}</div>
      <p v-if="description" class="settings-row__desc">{{ description }}</p>
    </div>
    <div class="settings-row__control">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.settings-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  padding: 24px 0;
  border-bottom: 1px solid var(--wb-border);
}

.settings-row:last-child {
  border-bottom: 0;
}

.settings-row__meta {
  min-width: 0;
  flex: 1;
}

.settings-row__title {
  font-size: 16px;
  font-weight: 500;
  color: var(--wb-text);
  line-height: 1.4;
}

.settings-row__desc {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--wb-text-secondary);
}

.settings-row__control {
  width: min(360px, 100%);
  flex-shrink: 0;
  display: flex;
  justify-content: flex-end;
  align-items: flex-start;
}

@media (max-width: 768px) {
  .settings-row {
    flex-direction: column;
    gap: 12px;
  }

  .settings-row__control {
    width: 100%;
    justify-content: stretch;
  }
}
</style>
