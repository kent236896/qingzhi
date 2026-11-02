<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchOwnProfile } from '@/api/auth'
import PageStatus from '@/components/common/PageStatus.vue'
import SettingsPageHeader from '@/components/settings/SettingsPageHeader.vue'
import SettingsRow from '@/components/settings/SettingsRow.vue'
import { useAuthStore } from '@/stores/auth'
import type { AuthMethod, OwnProfile } from '@/types/auth'
import { ApiError } from '@/utils/http'

const auth = useAuthStore()
const loading = ref(true)
const loadError = ref('')
const profile = ref<OwnProfile | null>(null)
const authMethod = ref<AuthMethod>('local')
const localPasswordEnabled = ref(false)

const accountTypeLabel = computed(() => {
  if (localPasswordEnabled.value && authMethod.value === 'sso') {
    return '本地账号（当前通过统一认证登录）'
  }
  if (!localPasswordEnabled.value) {
    return 'Keycloak 统一认证'
  }
  return '本地账号'
})

const loginMethodLabel = computed(() =>
  authMethod.value === 'sso' ? '统一认证（Keycloak / SSO）' : '本地用户名密码',
)

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await fetchOwnProfile()
    profile.value = response.data.user
    authMethod.value = response.data.auth_method
    localPasswordEnabled.value = response.data.local_password_enabled
    auth.applyAuthMeta({
      auth_method: response.data.auth_method,
      local_password_enabled: response.data.local_password_enabled,
      username_editable: response.data.username_editable,
      account_console_url: response.data.account_console_url,
    })
  } catch (error) {
    loadError.value = error instanceof ApiError ? error.message : '加载账号信息失败'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div>
    <SettingsPageHeader title="账号与登录方式" description="查看账号类型与当前登录方式" />

    <PageStatus v-if="loading" title="正在加载账号信息" kind="loading" />
    <PageStatus v-else-if="loadError" title="无法加载账号信息" :description="loadError" kind="error">
      <t-button theme="primary" variant="outline" @click="load">重试</t-button>
    </PageStatus>

    <template v-else-if="profile">
      <SettingsRow title="内部用户 ID">
        <span class="value">{{ profile.id }}</span>
      </SettingsRow>
      <SettingsRow title="用户名">
        <span class="value">{{ profile.username }}</span>
      </SettingsRow>
      <SettingsRow title="当前账号类型">
        <span class="value">{{ accountTypeLabel }}</span>
      </SettingsRow>
      <SettingsRow title="当前登录方式">
        <span class="value">{{ loginMethodLabel }}</span>
      </SettingsRow>
      <SettingsRow title="本地密码凭证">
        <span class="value">
          {{ localPasswordEnabled ? '已启用（可修改本地密码）' : '未启用（不可设置本地密码）' }}
        </span>
      </SettingsRow>
      <SettingsRow title="当前租户">
        <span class="value">{{ auth.tenantId || '—' }}</span>
      </SettingsRow>
      <p class="note">
        账号类型与本地密码凭证状态由后端返回。本阶段不提供本地用户名修改，也不会把平台资料同步到 Keycloak。
      </p>
    </template>
  </div>
</template>

<style scoped>
.value {
  font-size: 14px;
  color: var(--wb-text);
  line-height: 1.5;
  word-break: break-all;
  text-align: right;
}

.note {
  margin: 20px 0 0;
  max-width: 640px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--wb-text-secondary);
}
</style>
