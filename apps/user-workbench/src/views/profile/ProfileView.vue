<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ImageIcon, MailIcon, MobileIcon } from 'tdesign-icons-vue-next'
import { MessagePlugin } from 'tdesign-vue-next'
import { fetchOwnProfile, updateOwnProfile } from '@/api/auth'
import PasswordChangePanel from '@/components/profile/PasswordChangePanel.vue'
import PageStatus from '@/components/common/PageStatus.vue'
import { useAuthStore } from '@/stores/auth'
import type { AuthMethod, OwnProfile } from '@/types/auth'
import { ApiError } from '@/utils/http'

const auth = useAuthStore()

const activeTab = ref('profile')
const loading = ref(true)
const saving = ref(false)
const loadError = ref('')
const profile = ref<OwnProfile | null>(null)
const authMethod = ref<AuthMethod>('local')
const localPasswordEnabled = ref(false)
const usernameEditable = ref(false)

const nickname = ref('')
const email = ref('')
const mobile = ref('')
const remark = ref('')
const avatar = ref('')
const sex = ref(0)

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

const avatarLetter = computed(() => {
  const name = nickname.value.trim() || profile.value?.username || '?'
  return name.slice(0, 1)
})

const usernameHint = computed(() =>
  usernameEditable.value
    ? '可修改用户名'
    : '用户名由平台管理，不可自行修改，也不会同步到统一认证中心',
)

function applyFormFromUser(user: OwnProfile) {
  nickname.value = user.nickname
  email.value = user.email
  mobile.value = user.mobile
  remark.value = user.remark
  avatar.value = user.avatar
  sex.value = user.sex
}

async function loadProfile() {
  loading.value = true
  loadError.value = ''
  try {
    const response = await fetchOwnProfile()
    profile.value = response.data.user
    authMethod.value = response.data.auth_method
    localPasswordEnabled.value = response.data.local_password_enabled
    usernameEditable.value = response.data.username_editable
    auth.applyAuthMeta({
      auth_method: response.data.auth_method,
      local_password_enabled: response.data.local_password_enabled,
      username_editable: response.data.username_editable,
      account_console_url: response.data.account_console_url,
    })
    applyFormFromUser(response.data.user)
  } catch (error) {
    loadError.value = error instanceof ApiError ? error.message : '加载个人资料失败'
  } finally {
    loading.value = false
  }
}

async function saveProfile() {
  if (saving.value || !profile.value) {
    return
  }
  saving.value = true
  try {
    const response = await updateOwnProfile({
      nickname: nickname.value.trim(),
      email: email.value.trim(),
      mobile: mobile.value.trim(),
      remark: remark.value.trim(),
      avatar: avatar.value.trim(),
      sex: sex.value,
    })
    profile.value = response.data
    applyFormFromUser(response.data)
    auth.applyProfile(response.data)
    void MessagePlugin.success('资料已保存')
  } catch (error) {
    void MessagePlugin.error(error instanceof ApiError ? error.message : '保存失败')
  } finally {
    saving.value = false
  }
}

function resetProfile() {
  if (!profile.value) {
    return
  }
  applyFormFromUser(profile.value)
}

onMounted(() => {
  void loadProfile()
})
</script>

<template>
  <div class="profile-page">
    <PageStatus v-if="loading" title="正在加载个人资料" kind="loading" />
    <PageStatus v-else-if="loadError" title="无法加载个人资料" :description="loadError" kind="error">
      <t-button theme="primary" variant="outline" @click="loadProfile">重试</t-button>
    </PageStatus>

    <t-card v-else-if="profile" class="profile-card" :bordered="false">
      <t-tabs v-model="activeTab" class="profile-tabs">
        <t-tab-panel value="profile" label="个人资料">
          <div class="profile-section">
            <div class="identity-card">
              <t-avatar class="identity-card__avatar" size="72px" :image="avatar || undefined">
                {{ avatarLetter }}
              </t-avatar>
              <div class="identity-card__meta">
                <div class="identity-card__name">{{ nickname.trim() || '未设置显示名称' }}</div>
                <div class="identity-card__row">
                  <t-tag theme="primary" variant="light">@{{ profile?.username }}</t-tag>
                  <span class="identity-card__hint">{{ usernameHint }}</span>
                </div>
              </div>
            </div>

            <t-form label-align="top" class="profile-form" @submit.prevent="saveProfile">
              <h3 class="section-title">基本信息</h3>
              <div class="form-grid">
                <t-form-item label="显示名称">
                  <t-input v-model="nickname" placeholder="显示名称" maxlength="30" clearable />
                </t-form-item>
                <t-form-item label="性别">
                  <t-radio-group v-model="sex" variant="default-filled">
                    <t-radio-button :value="0">未知</t-radio-button>
                    <t-radio-button :value="1">男</t-radio-button>
                    <t-radio-button :value="2">女</t-radio-button>
                  </t-radio-group>
                </t-form-item>
              </div>

              <h3 class="section-title">联系方式</h3>
              <div class="form-grid">
                <t-form-item label="邮箱">
                  <t-input v-model="email" placeholder="name@example.com" clearable>
                    <template #prefix-icon>
                      <MailIcon />
                    </template>
                  </t-input>
                </t-form-item>
                <t-form-item label="手机号">
                  <t-input v-model="mobile" placeholder="请输入手机号" clearable>
                    <template #prefix-icon>
                      <MobileIcon />
                    </template>
                  </t-input>
                </t-form-item>
              </div>

              <h3 class="section-title">其他</h3>
              <t-form-item label="头像地址">
                <t-input v-model="avatar" placeholder="https://" clearable>
                  <template #prefix-icon>
                    <ImageIcon />
                  </template>
                </t-input>
                <template #help>
                  <span class="field-help">选填，填写后上方头像会实时预览</span>
                </template>
              </t-form-item>
              <t-form-item label="备注">
                <t-textarea
                  v-model="remark"
                  placeholder="备注"
                  :maxcharacter="200"
                  :autosize="{ minRows: 3, maxRows: 6 }"
                />
              </t-form-item>

              <div class="form-actions">
                <t-button theme="primary" type="submit" :loading="saving" @click.prevent="saveProfile">
                  保存资料
                </t-button>
                <t-button theme="default" variant="outline" :disabled="saving" @click="resetProfile">
                  重置
                </t-button>
              </div>
            </t-form>
          </div>
        </t-tab-panel>

        <t-tab-panel value="account" label="账号与登录方式">
          <div class="profile-section account-card">
            <dl>
              <div>
                <dt>内部用户 ID</dt>
                <dd>{{ profile?.id }}</dd>
              </div>
              <div>
                <dt>用户名</dt>
                <dd>{{ profile?.username }}</dd>
              </div>
              <div>
                <dt>当前账号类型</dt>
                <dd>{{ accountTypeLabel }}</dd>
              </div>
              <div>
                <dt>当前登录方式</dt>
                <dd>{{ loginMethodLabel }}</dd>
              </div>
              <div>
                <dt>本地密码凭证</dt>
                <dd>{{ localPasswordEnabled ? '已启用（可修改本地密码）' : '未启用（不可设置本地密码）' }}</dd>
              </div>
              <div>
                <dt>当前租户</dt>
                <dd>{{ auth.tenantId || '—' }}</dd>
              </div>
            </dl>
            <p class="field-help account-note">
              账号类型与本地密码凭证状态由后端返回。本阶段不提供本地用户名修改，也不会把平台资料同步到 Keycloak。
            </p>
          </div>
        </t-tab-panel>

        <t-tab-panel value="password" label="密码管理">
          <div class="profile-section">
            <PasswordChangePanel />
          </div>
        </t-tab-panel>
      </t-tabs>
    </t-card>
  </div>
</template>

<style scoped>
.profile-page {
  max-width: 960px;
}

.profile-card {
  border-radius: var(--td-radius-large);
  box-shadow: var(--td-shadow-1);
  background: var(--td-bg-color-container);
}

.profile-tabs :deep(.t-tabs__nav-item) {
  margin-right: 36px;
  font-size: 14px;
}

.profile-tabs :deep(.t-tabs__nav-item.t-is-active) {
  font-weight: 600;
  color: var(--td-brand-color);
}

.profile-section {
  padding: 8px 0 4px;
}

.identity-card {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 22px 24px;
  border-radius: var(--td-radius-medium);
  background: var(--td-brand-color-light);
}

.identity-card__avatar {
  flex-shrink: 0;
  color: var(--td-text-color-anti);
  background: var(--td-brand-color);
  border: 3px solid var(--td-bg-color-container);
  font-size: 28px;
  font-weight: 600;
}

.identity-card__meta {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.identity-card__name {
  font-size: 20px;
  font-weight: 600;
  line-height: 1.3;
  color: var(--td-text-color-primary);
  word-break: break-word;
}

.identity-card__row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.identity-card__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--td-text-color-secondary);
}

.section-title {
  margin: 28px 0 16px;
  padding-left: 10px;
  border-left: 3px solid var(--td-brand-color);
  font-size: 14px;
  font-weight: 600;
  line-height: 1.4;
  color: var(--td-text-color-primary);
}

.profile-form :deep(.t-form__item) {
  margin-bottom: 20px;
}

.profile-form :deep(.t-form__label) {
  font-size: 14px;
  color: var(--td-text-color-primary);
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  column-gap: 32px;
  row-gap: 20px;
}

.form-grid :deep(.t-form__item) {
  margin-bottom: 0;
}

.field-help {
  font-size: 12px;
  line-height: 1.5;
  color: var(--td-text-color-secondary);
}

.form-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 8px;
  padding-top: 20px;
  border-top: 1px solid var(--td-component-stroke);
}

.account-card dl {
  margin: 0;
  display: grid;
  gap: 20px;
}

.account-card dl > div {
  display: grid;
  gap: 4px;
}

.account-card dt {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.account-card dd {
  margin: 0;
  font-size: 14px;
  color: var(--td-text-color-primary);
  word-break: break-all;
}

.account-note {
  display: block;
  margin-top: 20px;
}

@media (max-width: 720px) {
  .identity-card {
    flex-direction: column;
    align-items: flex-start;
  }

  .form-grid {
    grid-template-columns: 1fr;
    column-gap: 0;
  }

  .profile-tabs :deep(.t-tabs__nav-item) {
    margin-right: 20px;
  }
}
</style>
