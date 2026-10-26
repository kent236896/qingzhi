<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError } from '@/utils/http'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { getLoginOptions, oidcStartURL, protocolLabel, type LoginOption } from '@/api/sso'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const appStore = useAppStore()

const username = ref('admin')
const password = ref('')
const tenantId = ref('')
const showTenant = ref(false)
const submitting = ref(false)
const errorMessage = ref('')
const localLoginEnabled = ref(true)
const providers = ref<LoginOption[]>([])

async function loadProviders() {
  try {
    const response = await getLoginOptions()
    localLoginEnabled.value = response.data.local_login_enabled
    providers.value = response.data.providers ?? []
    const auto = providers.value.find((item) => item.auto_login)
    if (auto && !route.query.sso_error && !route.query.local) {
      window.location.assign(oidcStartURL(auto.id, typeof route.query.redirect === 'string' ? route.query.redirect : '/'))
    }
  } catch {
    localLoginEnabled.value = true
    providers.value = []
  }
}

async function submit() {
  if (submitting.value) {
    return
  }
  if (!username.value.trim() || !password.value) {
    errorMessage.value = '请输入用户名和密码'
    return
  }
  submitting.value = true
  errorMessage.value = ''
  try {
    await auth.login(username.value.trim(), password.value, showTenant.value ? tenantId.value.trim() : undefined)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.replace(redirect || '/')
  } catch (error) {
    if (error instanceof ApiError && error.message.includes('选择租户')) {
      showTenant.value = true
      errorMessage.value = error.message
    } else {
      errorMessage.value = error instanceof ApiError ? error.message : '登录失败'
    }
  } finally {
    submitting.value = false
  }
}

function startProvider(item: LoginOption) {
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
  const tenant = showTenant.value ? tenantId.value.trim() : undefined
  if (item.protocol === 'oauth2') {
    const base = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '')
    const query = new URLSearchParams({ provider_id: item.id, redirect })
    if (tenant) query.set('tenant_id', tenant)
    window.location.assign(`${base}/api/v1/sso/oauth2/start?${query.toString()}`)
    return
  }
  window.location.assign(oidcStartURL(item.id, redirect, tenant))
}

onMounted(() => {
  if (typeof route.query.sso_error === 'string' && route.query.sso_error) {
    errorMessage.value = route.query.sso_error
  }
  void loadProviders()
})
</script>

<template>
  <div class="login-page">
    <div class="login-panel">
      <div class="brand">
        <img src="/favicon.svg" width="40" height="40" alt="" />
        <div>
          <h1>{{ appStore.title }}</h1>
          <p>账号登录</p>
        </div>
      </div>

      <div v-if="providers.length" class="sso-block">
        <t-button
          v-for="item in providers"
          :key="item.id"
          theme="primary"
          variant="outline"
          block
          class="sso-btn"
          @click="startProvider(item)"
        >
          使用 {{ item.name }}（{{ protocolLabel(item.protocol) }}）登录
        </t-button>
      </div>

      <t-form v-if="localLoginEnabled" label-align="top" @submit.prevent="submit">
        <t-form-item v-if="providers.length" label="或使用本地账号">
          <span />
        </t-form-item>
        <t-form-item label="用户名">
          <t-input v-model="username" placeholder="用户名" autocomplete="username" />
        </t-form-item>
        <t-form-item label="密码">
          <t-input
            v-model="password"
            type="password"
            placeholder="密码"
            autocomplete="current-password"
            @enter="submit"
          />
        </t-form-item>
        <t-form-item v-if="showTenant" label="租户编号">
          <t-input v-model="tenantId" placeholder="多租户时填写 tenant_id" @enter="submit" />
        </t-form-item>
        <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
        <t-button theme="primary" variant="base" type="submit" block :loading="submitting" @click.prevent="submit">
          登录
        </t-button>
      </t-form>
      <template v-else>
        <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
        <p class="hint">本地密码登录已关闭，请使用上方外部身份提供方。</p>
      </template>
      <p v-if="localLoginEnabled" class="hint">默认管理员：admin / Admin@123456（需已执行迁移并存在租户）</p>
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
  box-shadow: none;
}

.brand {
  display: flex;
  gap: var(--qz-space-3);
  align-items: center;
  margin-bottom: 24px;
}

.brand h1 {
  margin: 0;
  font-size: 22px;
  line-height: 1.2;
}

.brand p {
  margin: 4px 0 0;
  color: var(--qz-text-secondary);
}

.sso-block {
  display: grid;
  gap: 10px;
  margin-bottom: 20px;
}

.sso-btn {
  justify-content: center;
}

.error {
  margin: 0 0 12px;
  color: var(--qz-danger);
  font-size: var(--qz-font-sm);
}

.hint {
  margin: 16px 0 0;
  color: var(--qz-text-placeholder);
  font-size: var(--qz-font-sm);
  line-height: 1.5;
}
</style>
