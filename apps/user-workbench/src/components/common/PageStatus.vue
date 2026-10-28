<script setup lang="ts">
withDefaults(
  defineProps<{
    title?: string
    description?: string
    kind?: 'empty' | 'loading' | 'error' | 'info'
  }>(),
  {
    title: '暂无内容',
    description: '',
    kind: 'info',
  },
)
</script>

<template>
  <div class="page-status" :data-kind="kind">
    <div class="page-status__title">{{ title }}</div>
    <p v-if="description" class="page-status__desc">{{ description }}</p>
    <div v-if="$slots.default" class="page-status__extra">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.page-status {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  padding: 24px;
  border: 1px dashed var(--wb-border);
  border-radius: var(--wb-radius);
  background: var(--wb-surface);
}

.page-status__title {
  font-size: 16px;
  font-weight: 600;
  color: var(--wb-text);
}

.page-status__desc {
  margin: 0;
  color: var(--wb-text-secondary);
  font-size: 14px;
}

.page-status__extra {
  margin-top: 8px;
}

.page-status[data-kind='error'] {
  border-color: #f5c6cb;
  background: #fff8f8;
}

.page-status[data-kind='loading'] {
  border-style: solid;
}
</style>
