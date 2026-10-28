<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { changeOwnPassword, fetchPasswordPolicy, type PasswordPolicy } from '@/api/auth'
import { ApiError } from '@/utils/http'
import {
  LOCAL_PASSWORD_NOTICE,
  SSO_PASSWORD_NOTICE,
  describePasswordPolicy,
  validateNewPassword,
} from '@/utils/password'

const props = defineProps<{
  localPasswordEnabled: boolean
  accountConsoleUrl: string
}>()

const policy = ref<PasswordPolicy | null>(null)
const submitting = ref(false)
const formError = ref('')
const form = reactive({
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const policyHint = computed(() => describePasswordPolicy(policy.value))

onMounted(async () => {
  if (!props.localPasswordEnabled) {
    return
  }
  try {
    const response = await fetchPasswordPolicy()
    policy.value = response.data
  } catch {
    policy.value = null
  }
})

function clearForm() {
  form.currentPassword = ''
  form.newPassword = ''
  form.confirmPassword = ''
  formError.value = ''
}

async function submit() {
  if (submitting.value || !props.localPasswordEnabled) {
    return
  }
  formError.value = ''
  if (!form.currentPassword || !form.newPassword || !form.confirmPassword) {
    formError.value = '请填写当前密码、新密码和确认密码'
    return
  }
  if (form.newPassword !== form.confirmPassword) {
    formError.value = '新密码与确认密码不一致'
    return
  }
  if (form.newPassword === form.currentPassword) {
    formError.value = '新密码不能与当前密码相同'
    return
  }
  const policyError = validateNewPassword(form.newPassword, policy.value)
  if (policyError) {
    formError.value = policyError
    return
  }
  submitting.value = true
  try {
    await changeOwnPassword({
      current_password: form.currentPassword,
      new_password: form.newPassword,
    })
    clearForm()
    void MessagePlugin.success('本地密码已修改')
  } catch (error) {
    formError.value = error instanceof ApiError ? error.message : '修改失败'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="password-panel">
    <t-alert theme="warning" :message="LOCAL_PASSWORD_NOTICE" class="password-panel__alert" />

    <template v-if="localPasswordEnabled">
      <p class="password-panel__hint">{{ policyHint }}</p>
      <t-form label-align="top" class="password-panel__form" @submit.prevent="submit">
        <t-form-item label="当前密码" required>
          <t-input
            v-model="form.currentPassword"
            type="password"
            autocomplete="current-password"
            placeholder="当前本地密码"
          />
        </t-form-item>
        <t-form-item label="新密码" required>
          <t-input
            v-model="form.newPassword"
            type="password"
            autocomplete="new-password"
            placeholder="新本地密码"
          />
        </t-form-item>
        <t-form-item label="确认新密码" required>
          <t-input
            v-model="form.confirmPassword"
            type="password"
            autocomplete="new-password"
            placeholder="再次输入新密码"
          />
        </t-form-item>
        <p v-if="formError" class="password-panel__error">{{ formError }}</p>
        <t-button theme="primary" type="submit" :loading="submitting" @click.prevent="submit">
          修改本地密码
        </t-button>
      </t-form>
    </template>

    <template v-else>
      <t-alert theme="info" :message="SSO_PASSWORD_NOTICE" />
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
          统一认证中心账号管理入口尚未配置，请联系管理员，不要使用未确认的地址。
        </p>
      </div>
    </template>
  </div>
</template>

<style scoped>
.password-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 560px;
}

.password-panel__alert {
  white-space: normal;
}

.password-panel__hint,
.password-panel__muted {
  margin: 0;
  color: var(--wb-text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.password-panel__form {
  max-width: 420px;
}

.password-panel__error {
  margin: 0 0 12px;
  color: #d54941;
  font-size: 13px;
}

.password-panel__sso {
  margin-top: 4px;
}

.password-panel__link {
  color: var(--wb-primary);
  font-size: 14px;
}
</style>
