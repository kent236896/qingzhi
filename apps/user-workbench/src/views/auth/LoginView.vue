<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError } from '@/utils/http'
import { safeRedirectPath } from '@/utils/redirect'
import { useAuthStore } from '@/stores/auth'
import {
  getLoginOptions,
  oidcStartURL,
  protocolLabel,
  samlStartURL,
  type LoginOption,
} from '@/api/sso'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const tenantId = ref('')
const showTenant = ref(false)
const submitting = ref(false)
const errorMessage = ref('')
const localLoginEnabled = ref(true)
const providers = ref<LoginOption[]>([])
const optionsLoaded = ref(false)

async function loadProviders() {
  try {
    const response = await getLoginOptions()
    localLoginEnabled.value = response.data.local_login_enabled
    providers.value = response.data.providers ?? []
    const auto = providers.value.find((item) => item.auto_login)
    if (auto && !route.query.sso_error && !route.query.local) {
      startProvider(auto)
    }
  } catch {
    localLoginEnabled.value = true
    providers.value = []
  } finally {
    optionsLoaded.value = true
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
    await auth.login(
      username.value.trim(),
      password.value,
      showTenant.value ? tenantId.value.trim() : undefined,
    )
    password.value = ''
    await router.replace(safeRedirectPath(route.query.redirect, '/chat'))
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
  const redirect = safeRedirectPath(route.query.redirect, '/chat')
  const tenant = showTenant.value ? tenantId.value.trim() : undefined
  if (item.protocol === 'saml') {
    window.location.assign(samlStartURL(item.id, redirect, tenant))
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
        <span class="brand__mark">知</span>
        <div>
          <h1>用户工作台</h1>
          <p>使用企业账号登录</p>
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
        <t-button
          theme="primary"
          variant="base"
          type="submit"
          block
          :loading="submitting"
          @click.prevent="submit"
        >
          登录
        </t-button>
      </t-form>
      <template v-else-if="optionsLoaded">
        <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
        <p class="hint">本地密码登录已关闭，请使用上方统一认证入口。</p>
      </template>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 24px;
  background: var(--wb-bg);
}

.login-panel {
  width: min(420px, 100%);
  padding: 28px;
  border: 1px solid var(--wb-border);
  border-radius: var(--wb-radius);
  background: var(--wb-surface);
}

.brand {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 24px;
}

.brand__mark {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background: var(--wb-primary);
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-weight: 650;
}

.brand h1 {
  margin: 0;
  font-size: 22px;
  line-height: 1.2;
}

.brand p {
  margin: 4px 0 0;
  color: var(--wb-text-secondary);
  font-size: 13px;
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
  color: #d54941;
  font-size: 13px;
}

.hint {
  margin: 16px 0 0;
  color: var(--wb-text-secondary);
  font-size: 13px;
  line-height: 1.5;
}
</style>
