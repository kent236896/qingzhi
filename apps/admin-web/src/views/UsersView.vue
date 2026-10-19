<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/PageStatus.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import {
  createMember,
  createUser,
  flattenOrganizations,
  getTenant,
  getUser,
  listOrganizations,
  listPosts,
  listUserPosts,
  listUsers,
  removeMember,
  replaceMemberAssignments,
  replaceUserPosts,
  sexLabel,
  statusLabel,
  updateUser,
  type Assignment,
  type FlatOrganization,
  type RecordStatus,
  type UserAccount,
  type UserDetail,
  type UserMembership,
} from '@/api/identity'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canCreate = computed(() => auth.hasPermission('system:user:create'))
const canUpdate = computed(() => auth.hasPermission('system:user:update'))
const canDelete = computed(() => auth.hasPermission('system:user:delete'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const items = ref<UserAccount[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')

const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<UserAccount | null>(null)
const form = ref({
  username: '',
  password: '',
  nickname: '',
  remark: '',
  email: '',
  mobile: '',
  sex: 0,
  avatar: '',
  status: 0 as RecordStatus,
})

const detailVisible = ref(false)
const detail = ref<UserDetail | null>(null)
const sessionTenantName = ref('')
const membershipVisible = ref(false)
const membershipSaving = ref(false)
const editingMembership = ref<UserMembership | null>(null)
const orgOptions = ref<FlatOrganization[]>([])
const membershipForm = ref({
  tenantId: '',
  status: 0 as RecordStatus,
  organizationIds: [] as string[],
  primaryOrganizationId: '',
})
const postVisible = ref(false)
const postSaving = ref(false)
const postTenantId = ref('')
const postUserId = ref('')
const postIds = ref<string[]>([])
const postOptions = ref<{ label: string; value: string }[]>([])
const assignedPosts = ref<Record<string, string[]>>({})

const columns = [
  { colKey: 'avatar', title: '头像', width: 72 },
  { colKey: 'username', title: '用户名' },
  { colKey: 'nickname', title: '昵称' },
  { colKey: 'sex', title: '性别' },
  { colKey: 'email', title: '邮箱' },
  { colKey: 'mobile', title: '手机号' },
  { colKey: 'status', title: '状态' },
  { colKey: 'actions', title: '操作', width: 180 },
]

const primaryOptions = computed(() =>
  orgOptions.value.filter((item) => membershipForm.value.organizationIds.includes(item.id)),
)

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listUsers({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    items.value = response.data.items
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载账号'
    pageStatus.value = 'error'
  }
}

function search() {
  page.value = 1
  void load()
}

function openCreate() {
  editing.value = null
  form.value = { username: '', password: '', nickname: '', remark: '', email: '', mobile: '', sex: 0, avatar: '', status: 0 }
  editorVisible.value = true
}

function openEdit(row: UserAccount) {
  editing.value = row
  form.value = {
    username: row.username,
    password: '',
    nickname: row.nickname,
    remark: row.remark,
    email: row.email,
    mobile: row.mobile,
    sex: row.sex,
    avatar: row.avatar,
    status: row.status,
  }
  editorVisible.value = true
}

async function saveUser() {
  if (!form.value.nickname.trim() || (!editing.value && !form.value.username.trim())) {
    feedback.warning('请填写用户名和昵称')
    return
  }
  if (!editing.value && !form.value.password) {
    feedback.warning('请填写初始密码')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await updateUser(editing.value.id, {
        nickname: form.value.nickname.trim(),
        password: form.value.password || undefined,
        remark: form.value.remark.trim(),
        email: form.value.email.trim(),
        mobile: form.value.mobile.trim(),
        sex: form.value.sex,
        avatar: form.value.avatar.trim(),
        status: form.value.status,
      })
      feedback.success('账号已更新')
    } else {
      await createUser({
        username: form.value.username.trim(),
        password: form.value.password,
        nickname: form.value.nickname.trim(),
        remark: form.value.remark.trim() || undefined,
        email: form.value.email.trim() || undefined,
        mobile: form.value.mobile.trim() || undefined,
        sex: form.value.sex,
        avatar: form.value.avatar.trim() || undefined,
      })
      feedback.success('账号已创建')
    }
    editorVisible.value = false
    await load()
    if (detail.value && editing.value && detail.value.id === editing.value.id) {
      await openDetail(editing.value)
    }
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    saving.value = false
  }
}

async function openDetail(row: UserAccount) {
  try {
    const userResponse = await getUser(row.id)
    detail.value = userResponse.data
    detailVisible.value = true
    assignedPosts.value = {}
    await Promise.all(
      (userResponse.data.memberships ?? []).map(async (membership) => {
        if (membership.tenant_id !== auth.tenantId) {
          return
        }
        const postResponse = await listUserPosts(membership.tenant_id, userResponse.data.id)
        assignedPosts.value = {
          ...assignedPosts.value,
          [membership.tenant_id]: postResponse.data.map((item) => item.name),
        }
      }),
    )
    sessionTenantName.value = ''
    if (auth.tenantId) {
      try {
        const tenantResponse = await getTenant(auth.tenantId)
        sessionTenantName.value = tenantResponse.data.name
      } catch {
        sessionTenantName.value = ''
      }
    }
  } catch {
    // The HTTP client already shows the server message.
  }
}

async function refreshDetail() {
  if (!detail.value) {
    return
  }
  const response = await getUser(detail.value.id)
  detail.value = response.data
}

async function loadOrgOptions(tenantId: string) {
  orgOptions.value = []
  if (!tenantId) {
    return
  }
  const response = await listOrganizations(tenantId)
  orgOptions.value = flattenOrganizations(response.data ?? [])
}

function currentMembership(): UserMembership | null {
  return detail.value?.memberships.find((item) => item.tenant_id === auth.tenantId) ?? null
}

function canManageMembership(membership: UserMembership): boolean {
  return membership.tenant_id === auth.tenantId
}

async function openJoin() {
  if (!auth.tenantId) {
    feedback.warning('当前登录未绑定租户')
    return
  }
  const current = currentMembership()
  editingMembership.value = null
  membershipForm.value = {
    tenantId: auth.tenantId,
    status: current?.status ?? 0,
    organizationIds: current?.organizations.map((item) => item.id) ?? [],
    primaryOrganizationId: current?.organizations.find((item) => item.is_primary)?.id ?? '',
  }
  membershipVisible.value = true
  try {
    await loadOrgOptions(auth.tenantId)
  } catch {
    orgOptions.value = []
  }
}

async function openAdjust(membership: UserMembership) {
  editingMembership.value = membership
  membershipForm.value = {
    tenantId: membership.tenant_id,
    status: membership.status,
    organizationIds: membership.organizations.map((item) => item.id),
    primaryOrganizationId: membership.organizations.find((item) => item.is_primary)?.id ?? '',
  }
  membershipVisible.value = true
  try {
    await loadOrgOptions(membership.tenant_id)
  } catch {
    // The HTTP client already shows the server message.
  }
}

async function openPosts(membership: UserMembership) {
  if (!detail.value) {
    return
  }
  postTenantId.value = membership.tenant_id
  postUserId.value = detail.value.id
  postVisible.value = true
  const [posts, assigned] = await Promise.all([
    listPosts(membership.tenant_id, { page: 1, page_size: 100 }),
    listUserPosts(membership.tenant_id, detail.value.id),
  ])
  postOptions.value = posts.data.items.map((item) => ({ label: item.name, value: item.id }))
  postIds.value = assigned.data.map((item) => item.id)
}

async function savePosts() {
  postSaving.value = true
  try {
    const response = await replaceUserPosts(postTenantId.value, postUserId.value, postIds.value)
    assignedPosts.value = { ...assignedPosts.value, [postTenantId.value]: response.data.map((item) => item.name) }
    feedback.success('岗位任职已保存')
    postVisible.value = false
  } finally {
    postSaving.value = false
  }
}

function onOrganizationsChange() {
  if (!membershipForm.value.organizationIds.includes(membershipForm.value.primaryOrganizationId)) {
    membershipForm.value.primaryOrganizationId = membershipForm.value.organizationIds[0] ?? ''
  }
}

async function saveMembership() {
  if (!detail.value) {
    return
  }
  if (!membershipForm.value.tenantId) {
    feedback.warning('当前登录未绑定租户')
    return
  }
  if (membershipForm.value.organizationIds.length === 0 || !membershipForm.value.primaryOrganizationId) {
    feedback.warning('请选择任职组织，并指定一个主部门')
    return
  }
  membershipSaving.value = true
  try {
    const body = {
      organization_ids: membershipForm.value.organizationIds,
      primary_organization_id: membershipForm.value.primaryOrganizationId,
    }
    const existing = detail.value.memberships.find((item) => item.tenant_id === membershipForm.value.tenantId)
    if (existing) {
      await replaceMemberAssignments(existing.tenant_id, detail.value.id, body)
      feedback.success(editingMembership.value ? '任职组织已调整' : '组织任职已添加')
    } else {
      await createMember(membershipForm.value.tenantId, {
        user_id: detail.value.id,
        status: membershipForm.value.status,
        ...body,
      })
      feedback.success('组织任职已添加')
    }
    membershipVisible.value = false
    await refreshDetail()
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    membershipSaving.value = false
  }
}

function removeAssignment(membership: UserMembership, organization: Assignment) {
  if (!detail.value || !canManageMembership(membership)) {
    return
  }
  const userId = detail.value.id
  const remaining = membership.organizations.filter((item) => item.id !== organization.id)
  const dialog = DialogPlugin.confirm({
    header: '移除任职',
    body:
      remaining.length === 0
        ? `确定移除「${organization.name}」？这是「${membership.tenant_name}」下的最后一条组织任职，移除后不再保留该租户归属，账号本身不会删除。`
        : `确定移除「${organization.name}」的任职？该人员在其他组织和其他租户的任职保持不变。`,
    onConfirm: async () => {
      try {
        if (remaining.length === 0) {
          await removeMember(membership.tenant_id, userId)
        } else {
          const primary = remaining.find((item) => item.is_primary)?.id ?? remaining[0].id
          await replaceMemberAssignments(membership.tenant_id, userId, {
            organization_ids: remaining.map((item) => item.id),
            primary_organization_id: primary,
          })
        }
        feedback.success('已移除任职')
        dialog.hide()
        await refreshDetail()
      } catch {
        dialog.hide()
      }
    },
  })
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="page-grid">
    <t-card :bordered="false">
      <div class="toolbar">
        <t-input v-model="keyword" clearable placeholder="搜索用户名或显示名" @enter="search" />
        <t-button theme="default" @click="search">查询</t-button>
        <t-button v-if="canCreate" theme="primary" @click="openCreate">新建账号</t-button>
      </div>
      <PageStatus :status="pageStatus" :error-message="errorMessage" @retry="load">
        <t-table row-key="id" :data="items" :columns="columns" hover>
          <template #avatar="{ row }">
            <UserAvatar :image="row.avatar" size="small" />
          </template>
          <template #email="{ row }">{{ row.email || '—' }}</template>
          <template #mobile="{ row }">{{ row.mobile || '—' }}</template>
          <template #sex="{ row }">{{ sexLabel(row.sex) }}</template>
          <template #status="{ row }">
            <t-tag :theme="row.status === 0 ? 'success' : 'warning'" variant="light">
              {{ statusLabel(row.status) }}
            </t-tag>
          </template>
          <template #actions="{ row }">
            <t-button v-if="canUpdate" theme="primary" variant="text" @click="openEdit(row)">编辑</t-button>
            <t-button theme="primary" variant="text" @click="openDetail(row)">组织任职</t-button>
          </template>
        </t-table>
        <t-pagination
          v-model="page"
          v-model:page-size="pageSize"
          class="pager"
          :total="total"
          @change="load"
        />
      </PageStatus>
    </t-card>

    <t-dialog
      v-model:visible="editorVisible"
      :header="editing ? '编辑账号' : '新建账号'"
      :confirm-btn="{ content: '保存', loading: saving }"
      @confirm="saveUser"
    >
      <t-form label-align="top">
        <t-form-item label="用户名">
          <t-input v-model="form.username" :disabled="Boolean(editing)" maxlength="64" />
        </t-form-item>
        <t-form-item :label="editing ? '新密码' : '密码'">
          <t-input
            v-model="form.password"
            type="password"
            :placeholder="editing ? '留空表示不修改' : '6-32 位密码'"
            maxlength="32"
          />
        </t-form-item>
        <t-form-item label="昵称">
          <t-input v-model="form.nickname" maxlength="30" />
        </t-form-item>
        <t-form-item label="备注">
          <t-input v-model="form.remark" maxlength="500" />
        </t-form-item>
        <t-form-item label="邮箱">
          <t-input v-model="form.email" maxlength="50" />
        </t-form-item>
        <t-form-item label="手机号">
          <t-input v-model="form.mobile" maxlength="11" />
        </t-form-item>
        <t-form-item label="性别">
          <t-radio-group v-model="form.sex">
            <t-radio :value="0">未知</t-radio>
            <t-radio :value="1">男</t-radio>
            <t-radio :value="2">女</t-radio>
          </t-radio-group>
        </t-form-item>
        <t-form-item label="头像">
          <div class="avatar-field">
            <UserAvatar :image="form.avatar.trim()" />
            <t-input v-model="form.avatar" maxlength="512" placeholder="图片地址" />
          </div>
        </t-form-item>
        <t-form-item v-if="editing" label="状态">
          <t-radio-group v-model="form.status">
            <t-radio :value="0">正常</t-radio>
            <t-radio :value="1">停用</t-radio>
          </t-radio-group>
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-drawer v-model:visible="detailVisible" header="组织任职" size="640px" :footer="false">
      <template v-if="detail">
        <p class="hint user-heading">
          <UserAvatar :image="detail.avatar" size="small" />
          <span>{{ detail.nickname }}（{{ detail.username }}）</span>
        </p>
        <div class="toolbar">
          <t-button v-if="canCreate" theme="primary" @click="openJoin">添加组织任职</t-button>
        </div>
        <p class="hint">任职保存在当前登录租户的组织树中。其他租户已有任职会保留，但不能在本次会话里修改。</p>
        <div v-if="detail.memberships.length === 0" class="page-status">
          <p>还没有组织任职</p>
        </div>
        <t-card v-for="membership in detail.memberships" :key="membership.tenant_id" class="membership-card">
          <div class="membership-head">
            <strong>{{ membership.tenant_name }}</strong>
            <t-tag variant="light">{{ membership.tenant_code }}</t-tag>
          </div>
          <p>主体组织：{{ membership.root_organization_name || '—' }}</p>
          <div v-if="membership.organizations.length === 0" class="page-status">
            <p>该租户下还没有组织任职</p>
          </div>
          <div v-for="organization in membership.organizations" :key="organization.id" class="assignment-row">
            <span>{{ organization.name }}<template v-if="organization.is_primary">（主部门）</template></span>
            <t-tag :theme="membership.status === 0 ? 'success' : 'warning'" variant="light">
              {{ statusLabel(membership.status) }}
            </t-tag>
            <t-button
              v-if="canDelete && canManageMembership(membership)"
              theme="danger"
              variant="text"
              @click="removeAssignment(membership, organization)"
            >
              移除任职
            </t-button>
          </div>
          <div class="toolbar">
            <t-button
              v-if="canUpdate && canManageMembership(membership)"
              theme="primary"
              variant="text"
              @click="openAdjust(membership)"
            >
              调整组织
            </t-button>
            <span v-if="!canManageMembership(membership)" class="hint">需登录该租户后调整</span>
          </div>
          <p class="hint">岗位：{{ (assignedPosts[membership.tenant_id] ?? []).join('、') || '未任职' }}</p>
          <t-button
            v-if="canUpdate && canManageMembership(membership)"
            theme="primary"
            variant="text"
            @click="openPosts(membership)"
          >
            调整岗位
          </t-button>
        </t-card>
      </template>
    </t-drawer>

    <t-dialog
      v-model:visible="membershipVisible"
      :header="editingMembership ? '调整组织' : '添加组织任职'"
      :confirm-btn="{ content: '保存', loading: membershipSaving }"
      @confirm="saveMembership"
    >
      <t-form label-align="top">
        <t-form-item label="租户">
          <t-input
            :value="editingMembership ? editingMembership.tenant_name : sessionTenantName || '当前登录租户'"
            disabled
          />
        </t-form-item>
        <t-form-item v-if="!editingMembership && !currentMembership()" label="任职状态">
          <t-radio-group v-model="membershipForm.status">
            <t-radio :value="0">正常</t-radio>
            <t-radio :value="1">停用</t-radio>
          </t-radio-group>
        </t-form-item>
        <t-form-item label="任职组织">
          <t-select
            v-model="membershipForm.organizationIds"
            multiple
            placeholder="选择一个或多个组织或部门"
            :options="orgOptions.map((item) => ({ label: item.label, value: item.id }))"
            @change="onOrganizationsChange"
          />
        </t-form-item>
        <t-form-item label="主部门">
          <t-select
            v-model="membershipForm.primaryOrganizationId"
            placeholder="选择一个主部门"
            :options="primaryOptions.map((item) => ({ label: item.label, value: item.id }))"
          />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog v-model:visible="postVisible" header="调整岗位" :confirm-btn="{ content: '保存', loading: postSaving }" @confirm="savePosts">
      <t-form label-align="top">
        <t-form-item label="岗位">
          <t-select v-model="postIds" multiple placeholder="选择一个或多个岗位" :options="postOptions" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<style scoped>
.assignment-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--qz-space-2);
  align-items: center;
  margin: var(--qz-space-2) 0;
}
</style>
