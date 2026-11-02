<script setup lang="ts">
import { onMounted } from 'vue'
import { useSettingsStore } from '@/stores/settings'

const settings = useSettingsStore()

onMounted(() => {
  void settings.openSettings('profile')
})
</script>

<template>
  <div class="profile-redirect" />
</template>
