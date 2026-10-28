<script setup lang="ts">
import zhConfig from 'tdesign-vue-next/es/locale/zh_CN'
</script>

<template>
  <t-config-provider :global-config="zhConfig">
    <RouterView />
  </t-config-provider>
</template>
