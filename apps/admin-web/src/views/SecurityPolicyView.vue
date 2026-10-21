<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import PageStatus from '@/components/PageStatus.vue'
import { formatTime, getSecurityPolicy, updateSecurityPolicy, type SecurityPolicy } from '@/api/security'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canUpdate = computed(() => auth.hasPermission('system:security-policy:update'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const saving = ref(false)
const updatedAt = ref('')
const updater = ref('')
const form = ref({
  password_min_length: 6,
  password_max_length: 32,
  password_require_letter: false,
  password_require_number: false,
  password_require_symbol: false,
  login_fail_limit: 0,
  login_lock_minutes: 15,
})

function apply(policy: SecurityPolicy) {
  form.value = {
    password_min_length: policy.password_min_length,
    password_max_length: policy.password_max_length,
    password_require_letter: policy.password_require_letter,
    password_require_number: policy.password_require_number,
    password_require_symbol: policy.password_require_symbol,
    login_fail_limit: policy.login_fail_limit,
    login_lock_minutes: policy.login_lock_minutes,
  }
  updatedAt.value = policy.update_time
  updater.value = policy.updater
}

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await getSecurityPolicy()
    apply(response.data)
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载安全策略'
    pageStatus.value = 'error'
  }
}

async function save() {
  saving.value = true
  try {
    const response = await updateSecurityPolicy({ ...form.value })
    apply(response.data)
    feedback.success('安全策略已更新')
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="page-grid">
    <t-card :bordered="false">
      <PageStatus :status="pageStatus" :error-message="errorMessage" @retry="load">
        <t-form class="policy-form" label-align="top">
          <t-form-item label="密码最小长度">
            <t-input-number v-model="form.password_min_length" :min="6" :max="32" :disabled="!canUpdate" />
          </t-form-item>
          <t-form-item label="密码最大长度">
            <t-input-number v-model="form.password_max_length" :min="6" :max="32" :disabled="!canUpdate" />
          </t-form-item>
          <t-form-item label="密码必须包含字母">
            <t-switch v-model="form.password_require_letter" :disabled="!canUpdate" />
          </t-form-item>
          <t-form-item label="密码必须包含数字">
            <t-switch v-model="form.password_require_number" :disabled="!canUpdate" />
          </t-form-item>
          <t-form-item label="密码必须包含符号">
            <t-switch v-model="form.password_require_symbol" :disabled="!canUpdate" />
          </t-form-item>
          <t-form-item label="连续失败锁定次数" help="0 表示不锁定">
            <t-input-number v-model="form.login_fail_limit" :min="0" :max="20" :disabled="!canUpdate" />
          </t-form-item>
          <t-form-item label="锁定统计窗口（分钟）">
            <t-input-number v-model="form.login_lock_minutes" :min="1" :max="1440" :disabled="!canUpdate" />
          </t-form-item>
          <p class="hint">最近更新：{{ updater || '系统' }} · {{ formatTime(updatedAt) }}</p>
          <t-button v-if="canUpdate" theme="primary" variant="base" :loading="saving" @click="save">保存</t-button>
        </t-form>
      </PageStatus>
    </t-card>
  </div>
</template>

<style scoped>
.policy-form {
  max-width: 480px;
}
</style>
