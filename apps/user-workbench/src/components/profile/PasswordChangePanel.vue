<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { changeOwnPassword, fetchPasswordPolicy } from '@/api/auth'
import type { PasswordPolicy } from '@/types/auth'
import { ApiError } from '@/utils/http'
import { useAuthStore } from '@/stores/auth'

const LOCAL_PASSWORD_NOTICE =
  '注意：此处修改的密码仅用于本平台本地账号登录，不会修改统一认证中心（Keycloak）的密码。如果你使用统一认证中心登录，请前往统一认证中心修改密码。'

const SSO_ONLY_NOTICE =
  '当前账号使用统一认证中心登录。本页面不会修改统一认证中心密码，请前往统一认证中心修改密码。'

const auth = useAuthStore()

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const submitting = ref(false)
const formError = ref('')
const policy = ref<PasswordPolicy | null>(null)
const policyLoaded = ref(false)

const canChangeLocalPassword = computed(() => auth.localPasswordEnabled)
const accountConsoleUrl = computed(() => auth.accountConsoleUrl.trim())

const policyHint = computed(() => {
  if (!policy.value) {
    return '新密码需满足平台安全策略（以服务端校验为准）。'
  }
  const parts = [
    `长度 ${policy.value.password_min_length}-${policy.value.password_max_length} 个字符`,
  ]
  if (policy.value.password_require_letter) parts.push('需包含字母')
  if (policy.value.password_require_number) parts.push('需包含数字')
  if (policy.value.password_require_symbol) parts.push('需包含符号')
  return `密码要求：${parts.join('；')}。`
})

function clearForm() {
  currentPassword.value = ''
  newPassword.value = ''
  confirmPassword.value = ''
  formError.value = ''
}

function validateClient(): string {
  if (!currentPassword.value || !newPassword.value || !confirmPassword.value) {
    return '请填写当前密码、新密码和确认密码'
  }
  if (newPassword.value !== confirmPassword.value) {
    return '新密码与确认密码不一致'
  }
  if (newPassword.value === currentPassword.value) {
    return '新密码不能与当前密码相同'
  }
  if (policy.value) {
    const len = newPassword.value.length
    if (len < policy.value.password_min_length || len > policy.value.password_max_length) {
      return `新密码长度必须在 ${policy.value.password_min_length} 到 ${policy.value.password_max_length} 个字符之间`
    }
    if (policy.value.password_require_letter && !/[A-Za-z]/.test(newPassword.value)) {
      return '新密码需包含字母'
    }
    if (policy.value.password_require_number && !/\d/.test(newPassword.value)) {
      return '新密码需包含数字'
    }
    if (
      policy.value.password_require_symbol &&
      !/[^A-Za-z0-9]/.test(newPassword.value)
    ) {
      return '新密码需包含符号'
    }
  }
  return ''
}

async function submit() {
  if (submitting.value || !canChangeLocalPassword.value) {
    return
  }
  const clientError = validateClient()
  if (clientError) {
    formError.value = clientError
    return
  }
  submitting.value = true
  formError.value = ''
  try {
    await changeOwnPassword({
      current_password: currentPassword.value,
      new_password: newPassword.value,
    })
    clearForm()
    void MessagePlugin.success('本地密码已修改成功。当前登录会话仍然有效。')
  } catch (error) {
    formError.value = error instanceof ApiError ? error.message : '修改密码失败'
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  if (!canChangeLocalPassword.value) {
    policyLoaded.value = true
    return
  }
  try {
    const response = await fetchPasswordPolicy()
    policy.value = response.data
  } catch {
    policy.value = null
  } finally {
    policyLoaded.value = true
  }
})
</script>

<template>
  <div class="password-panel">
    <t-alert v-if="canChangeLocalPassword" theme="warning" :message="LOCAL_PASSWORD_NOTICE" />
    <t-alert v-else theme="warning" :message="SSO_ONLY_NOTICE" />

    <template v-if="!canChangeLocalPassword">
      <div class="password-panel__sso">
        <a
          v-if="accountConsoleUrl"
          class="password-panel__link"
          :href="accountConsoleUrl"
          target="_blank"
          rel="noopener noreferrer"
        >
          前往统一认证中心
        </a>
        <p v-else class="password-panel__muted">
          统一认证中心账号管理入口尚未配置，请联系管理员获取改密方式。不要在本平台尝试设置本地密码。
        </p>
      </div>
    </template>

    <template v-else>
      <p class="password-panel__hint">{{ policyHint }}</p>
      <t-form label-align="top" class="password-panel__form" @submit.prevent="submit">
        <t-form-item label="当前密码" required>
          <t-input
            v-model="currentPassword"
            type="password"
            autocomplete="current-password"
            placeholder="请输入当前本地密码"
          />
        </t-form-item>
        <t-form-item label="新密码" required>
          <t-input
            v-model="newPassword"
            type="password"
            autocomplete="new-password"
            placeholder="请输入新本地密码"
          />
        </t-form-item>
        <t-form-item label="确认新密码" required>
          <t-input
            v-model="confirmPassword"
            type="password"
            autocomplete="new-password"
            placeholder="请再次输入新密码"
            @enter="submit"
          />
        </t-form-item>
        <p v-if="formError" class="password-panel__error">{{ formError }}</p>
        <t-button
          theme="primary"
          type="submit"
          :loading="submitting"
          :disabled="!policyLoaded"
          @click.prevent="submit"
        >
          修改本地密码
        </t-button>
      </t-form>
    </template>
  </div>
</template>

<style scoped>
.password-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 520px;
}

.password-panel__form {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.password-panel__hint,
.password-panel__muted {
  margin: 0;
  color: var(--wb-text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.password-panel__error {
  margin: 0 0 8px;
  color: #d54941;
  font-size: 13px;
}

.password-panel__sso {
  padding: 12px 0;
}

.password-panel__link {
  color: var(--wb-primary);
  font-size: 14px;
}
</style>
