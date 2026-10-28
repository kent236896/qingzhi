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
  <!--
    Do not wrap ready content in t-loading: its absolute overlay can remain and block
    all clicks even after data has rendered (z-index 3500 full-size mask).
  -->
  <div v-if="status === 'loading'" class="page-status">
    <t-loading :loading="true" size="small" />
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
</template>
