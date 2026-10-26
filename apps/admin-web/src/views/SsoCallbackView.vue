<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { exchangeSsoTicket } from '@/api/sso'
import { useAuthStore } from '@/stores/auth'
import { ApiError } from '@/utils/http'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const message = ref('正在完成外部登录…')

onMounted(async () => {
  const error = typeof route.query.error === 'string' ? route.query.error : ''
  if (error) {
    message.value = error
    window.setTimeout(() => {
      void router.replace({ name: 'login', query: { sso_error: error } })
    }, 1200)
    return
  }
  const ticket = typeof route.query.ticket === 'string' ? route.query.ticket : ''
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
  if (!ticket) {
    message.value = '缺少登录票据'
    return
  }
  try {
    const response = await exchangeSsoTicket(ticket)
    auth.applyExternalLogin(response.data)
    await router.replace(redirect || '/')
  } catch (err) {
    message.value = err instanceof ApiError ? err.message : '外部登录失败'
    window.setTimeout(() => {
      void router.replace({ name: 'login', query: { sso_error: message.value } })
    }, 1500)
  }
})
</script>

<template>
  <div class="login-page">
    <div class="login-panel">
      <p>{{ message }}</p>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: var(--qz-content-padding);
  background: var(--qz-bg);
}
.login-panel {
  width: min(420px, 100%);
  padding: var(--qz-space-6);
  border: 1px solid var(--qz-border);
  border-radius: var(--qz-radius);
  background: var(--qz-surface);
  text-align: center;
  color: var(--qz-text-secondary);
}
</style>
