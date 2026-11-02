<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { fetchOwnProfile, updateOwnProfile } from '@/api/auth'
import PageStatus from '@/components/common/PageStatus.vue'
import SettingsPageHeader from '@/components/settings/SettingsPageHeader.vue'
import SettingsRow from '@/components/settings/SettingsRow.vue'
import { useAuthStore } from '@/stores/auth'
import type { OwnProfile } from '@/types/auth'
import { ApiError } from '@/utils/http'

const auth = useAuthStore()

const loading = ref(true)
const saving = ref(false)
const loadError = ref('')
const profile = ref<OwnProfile | null>(null)

const nickname = ref('')
const email = ref('')
const mobile = ref('')
const remark = ref('')
const avatar = ref('')
const sex = ref(0)

const avatarLetter = computed(() => {
  const name = nickname.value.trim() || profile.value?.username || '?'
  return name.slice(0, 1)
})

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
  if (saving.value || !profile.value) return
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

onMounted(() => {
  void loadProfile()
})
</script>

<template>
  <div>
    <SettingsPageHeader title="用户信息" description="管理头像、昵称与联系方式" />

    <PageStatus v-if="loading" title="正在加载个人资料" kind="loading" />
    <PageStatus v-else-if="loadError" title="无法加载个人资料" :description="loadError" kind="error">
      <t-button theme="primary" variant="outline" @click="loadProfile">重试</t-button>
    </PageStatus>

    <template v-else-if="profile">
      <SettingsRow title="头像" description="填写头像地址后可预览；未设置时显示用户名首字">
        <div class="avatar-control">
          <span class="avatar-preview">
            <img v-if="avatar" :src="avatar" alt="" />
            <template v-else>{{ avatarLetter }}</template>
          </span>
          <t-input v-model="avatar" placeholder="https://" clearable style="width: 100%" />
        </div>
      </SettingsRow>

      <SettingsRow title="用户名" description="用户名由平台管理，不可自行修改">
        <span class="readonly">@{{ profile?.username }}</span>
      </SettingsRow>

      <SettingsRow title="显示名称" description="对外展示的昵称">
        <t-input v-model="nickname" placeholder="显示名称" maxlength="30" clearable style="width: 100%" />
      </SettingsRow>

      <SettingsRow title="性别">
        <t-radio-group v-model="sex" variant="default-filled">
          <t-radio-button :value="0">未知</t-radio-button>
          <t-radio-button :value="1">男</t-radio-button>
          <t-radio-button :value="2">女</t-radio-button>
        </t-radio-group>
      </SettingsRow>

      <SettingsRow title="邮箱" description="用于联系与通知（不自动同步到统一认证中心）">
        <t-input v-model="email" placeholder="name@example.com" clearable style="width: 100%" />
      </SettingsRow>

      <SettingsRow title="手机号">
        <t-input v-model="mobile" placeholder="请输入手机号" clearable style="width: 100%" />
      </SettingsRow>

      <SettingsRow title="备注">
        <t-textarea
          v-model="remark"
          placeholder="备注"
          :maxcharacter="200"
          :autosize="{ minRows: 3, maxRows: 5 }"
          style="width: 100%"
        />
      </SettingsRow>

      <div class="profile-actions">
        <t-button theme="primary" :loading="saving" style="width: 120px" @click="saveProfile">
          保存
        </t-button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.avatar-control {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 10px;
}

.avatar-preview {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--wb-primary);
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 600;
  overflow: hidden;
}

.avatar-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.readonly {
  font-size: 14px;
  color: var(--wb-text);
  line-height: 32px;
}

.profile-actions {
  margin-top: 28px;
  padding-top: 8px;
}
</style>
