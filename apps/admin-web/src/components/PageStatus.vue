<script setup lang="ts">
defineProps<{
  status: 'loading' | 'empty' | 'error' | 'ready'
  description?: string
  errorMessage?: string
}>()

defineEmits<{
  retry: []
}>()
</script>

<template>
  <t-loading :loading="status === 'loading'" size="small">
    <div v-if="status === 'loading'" class="page-status">
      <p>正在加载</p>
    </div>
    <div v-else-if="status === 'empty'" class="page-status">
      <t-empty :description="description || '暂无数据'" />
    </div>
    <div v-else-if="status === 'error'" class="page-status">
      <p>{{ errorMessage || '页面加载失败' }}</p>
      <t-button theme="primary" variant="outline" @click="$emit('retry')">重试</t-button>
    </div>
    <slot v-else />
  </t-loading>
</template>
