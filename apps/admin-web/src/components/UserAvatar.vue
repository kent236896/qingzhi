<script setup lang="ts">
defineProps<{
  image?: string
  size?: 'small' | 'medium' | 'large'
}>()
</script>

<template>
  <t-avatar :image="image || undefined" :size="size ?? 'medium'">
    <template #icon>
      <svg viewBox="0 0 24 24" width="1em" height="1em" fill="currentColor" aria-hidden="true">
        <path
          d="M16.5 7.5C16.5 9.98528 14.4853 12 12 12C9.51472 12 7.5 9.98528 7.5 7.5C7.5 5.01472 9.51472 3 12 3C14.4853 3 16.5 5.01472 16.5 7.5Z"
        />
        <path d="M20 21V19C20 16.7909 18.2091 15 16 15H8C5.79086 15 4 16.7909 4 19V21H20Z" />
      </svg>
    </template>
  </t-avatar>
</template>
