<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { updateOwnProfile, type OwnProfile } from '@/api/auth'
import { ApiError } from '@/utils/http'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  profile: OwnProfile
}>()

const emit = defineEmits<{
  saved: [profile: OwnProfile]
}>()

const auth = useAuthStore()
const submitting = ref(false)
const form = reactive({
  nickname: '',
  email: '',
  mobile: '',
  remark: '',
  avatar: '',
  sex: 0,
})

watch(
  () => props.profile,
  (value) => {
    form.nickname = value.nickname
    form.email = value.email
    form.mobile = value.mobile
    form.remark = value.remark
    form.avatar = value.avatar
    form.sex = value.sex
  },
  { immediate: true },
)

async function submit() {
  if (submitting.value) {
    return
  }
  if (!form.nickname.trim()) {
    void MessagePlugin.warning('请填写显示名称')
    return
  }
  submitting.value = true
  try {
    const response = await updateOwnProfile({
      nickname: form.nickname.trim(),
      email: form.email.trim(),
      mobile: form.mobile.trim(),
      remark: form.remark.trim(),
      avatar: form.avatar.trim(),
      sex: form.sex,
    })
    auth.applyProfile(response.data)
    emit('saved', response.data)
    void MessagePlugin.success('资料已保存')
  } catch (error) {
    void MessagePlugin.error(error instanceof ApiError ? error.message : '保存失败')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <t-form label-align="top" class="profile-form" @submit.prevent="submit">
    <t-form-item label="用户名">
      <t-input :model-value="profile.username" disabled />
      <template #help>平台用户名由管理员维护，本阶段不支持自行修改，也不会同步到 Keycloak。</template>
    </t-form-item>
    <t-form-item label="显示名称" required>
      <t-input v-model="form.nickname" maxlength="30" placeholder="显示名称" />
    </t-form-item>
    <t-form-item label="邮箱">
      <t-input v-model="form.email" placeholder="邮箱" />
    </t-form-item>
    <t-form-item label="手机">
      <t-input v-model="form.mobile" placeholder="手机号" />
    </t-form-item>
    <t-form-item label="头像 URL">
      <t-input v-model="form.avatar" placeholder="可选，填写图片地址" />
    </t-form-item>
    <t-form-item label="性别">
      <t-radio-group v-model="form.sex">
        <t-radio :value="0">未知</t-radio>
        <t-radio :value="1">男</t-radio>
        <t-radio :value="2">女</t-radio>
      </t-radio-group>
    </t-form-item>
    <t-form-item label="备注">
      <t-textarea v-model="form.remark" :maxlength="200" placeholder="可选" />
    </t-form-item>
    <t-button theme="primary" type="submit" :loading="submitting" @click.prevent="submit">
      保存资料
    </t-button>
  </t-form>
</template>

<style scoped>
.profile-form {
  max-width: 480px;
}
</style>
