<script setup lang="ts">
import { computed } from 'vue'
import type { AuthMethod } from '@/types/auth'

const props = defineProps<{
  username: string
  authMethod: AuthMethod
  localPasswordEnabled: boolean
  tenantId: string
}>()

const accountType = computed(() => {
  if (props.localPasswordEnabled && props.authMethod === 'sso') {
    return '本地账号（当前通过统一认证登录）'
  }
  if (props.localPasswordEnabled) {
    return '本地账号'
  }
  return 'Keycloak 统一认证'
})

const loginMethodLabel = computed(() =>
  props.authMethod === 'sso' ? '统一认证中心（SSO）' : '本地用户名密码',
)
</script>

<template>
  <div class="account-info">
    <dl>
      <div>
        <dt>用户名</dt>
        <dd>{{ username || '—' }}</dd>
      </div>
      <div>
        <dt>账号类型</dt>
        <dd>{{ accountType }}</dd>
      </div>
      <div>
        <dt>当前登录方式</dt>
        <dd>{{ loginMethodLabel }}</dd>
      </div>
      <div>
        <dt>本地密码凭证</dt>
        <dd>{{ localPasswordEnabled ? '已启用（可修改本地密码）' : '未启用（不可通过本平台改密）' }}</dd>
      </div>
      <div>
        <dt>当前租户</dt>
        <dd>{{ tenantId || '—' }}</dd>
      </div>
    </dl>
    <p class="account-info__note">
      账号类型与本地密码凭证状态由后端返回，前端不会根据输入猜测。修改本地密码不会同步到 Keycloak。
    </p>
  </div>
</template>

<style scoped>
.account-info dl {
  margin: 0;
  display: grid;
  gap: 12px;
}

.account-info dl > div {
  display: grid;
  grid-template-columns: 140px 1fr;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid var(--wb-border);
}

.account-info dt {
  color: var(--wb-text-secondary);
  font-size: 13px;
}

.account-info dd {
  margin: 0;
  font-size: 14px;
}

.account-info__note {
  margin: 16px 0 0;
  color: var(--wb-text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

@media (max-width: 640px) {
  .account-info dl > div {
    grid-template-columns: 1fr;
    gap: 4px;
  }
}
</style>
