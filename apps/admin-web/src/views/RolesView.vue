<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/PageStatus.vue'
import {
  createRole,
  dataScopeLabel,
  deleteRole,
  flattenOrganizations,
  listMembers,
  listMenus,
  listOrganizations,
  listRoleDepts,
  listRoleMenus,
  listRoles,
  listRoleUsers,
  replaceRoleDepts,
  replaceRoleMenus,
  replaceRoleUsers,
  roleTypeLabel,
  statusLabel,
  updateRole,
  type DataScope,
  type FlatOrganization,
  type Member,
  type MenuNode,
  type RecordStatus,
  type Role,
  type RoleType,
} from '@/api/identity'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canCreate = computed(() => auth.hasPermission('system:role:create'))
const canUpdate = computed(() => auth.hasPermission('system:role:update'))
const canDelete = computed(() => auth.hasPermission('system:role:delete'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const tenantId = computed(() => auth.tenantId)
const items = ref<Role[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')

const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<Role | null>(null)
const form = ref({
  name: '',
  code: '',
  sort: 0,
  type: 2 as RoleType,
  dataScope: 1 as DataScope,
  remark: '',
  status: 0 as RecordStatus,
})

const menuVisible = ref(false)
const menuSaving = ref(false)
const menuRole = ref<Role | null>(null)
const menuTree = ref<MenuNode[]>([])
const selectedMenuIds = ref<string[]>([])

const deptVisible = ref(false)
const deptSaving = ref(false)
const deptRole = ref<Role | null>(null)
const orgOptions = ref<FlatOrganization[]>([])
const selectedDeptIds = ref<string[]>([])

const userVisible = ref(false)
const userSaving = ref(false)
const userRole = ref<Role | null>(null)
const members = ref<Member[]>([])
const selectedUserIds = ref<string[]>([])

const columns = [
  { colKey: 'name', title: '名称' },
  { colKey: 'code', title: '权限字符串' },
  { colKey: 'type', title: '类型' },
  { colKey: 'data_scope', title: '数据范围' },
  { colKey: 'sort', title: '排序' },
  { colKey: 'remark', title: '备注' },
  { colKey: 'status', title: '状态' },
  { colKey: 'actions', title: '操作', width: 280 },
]

async function bootstrap() {
  if (!tenantId.value) {
    pageStatus.value = 'error'
    errorMessage.value = '当前登录未绑定租户'
    return
  }
  await load()
}

async function load() {
  if (!tenantId.value) {
    items.value = []
    total.value = 0
    pageStatus.value = 'ready'
    return
  }
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listRoles(tenantId.value, {
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    items.value = response.data.items
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载角色'
    pageStatus.value = 'error'
  }
}

function search() {
  page.value = 1
  void load()
}

function openCreate() {
  editing.value = null
  form.value = { name: '', code: '', sort: 0, type: 2, dataScope: 1, remark: '', status: 0 }
  editorVisible.value = true
}

function openEdit(row: Role) {
  editing.value = row
  form.value = {
    name: row.name,
    code: row.code,
    sort: row.sort,
    type: row.type,
    dataScope: row.data_scope || 1,
    remark: row.remark,
    status: row.status,
  }
  editorVisible.value = true
}

async function save() {
  if (!tenantId.value || !form.value.name.trim() || !form.value.code.trim()) {
    feedback.warning('请填写名称和权限字符串')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await updateRole(tenantId.value, editing.value.id, {
        name: form.value.name.trim(),
        code: form.value.code.trim(),
        sort: Number(form.value.sort) || 0,
        type: form.value.type,
        data_scope: form.value.dataScope,
        remark: form.value.remark.trim(),
        status: form.value.status,
      })
      feedback.success('角色已更新')
    } else {
      await createRole(tenantId.value, {
        name: form.value.name.trim(),
        code: form.value.code.trim(),
        sort: Number(form.value.sort) || 0,
        type: form.value.type,
        data_scope: form.value.dataScope,
        remark: form.value.remark.trim() || undefined,
      })
      feedback.success('角色已创建')
    }
    editorVisible.value = false
    await load()
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    saving.value = false
  }
}

function remove(row: Role) {
  const dialog = DialogPlugin.confirm({
    header: '删除角色',
    body: `确定删除「${row.name}」？`,
    onConfirm: async () => {
      try {
        await deleteRole(tenantId.value, row.id)
        feedback.success('角色已删除')
        dialog.hide()
        await load()
      } catch {
        dialog.hide()
      }
    },
  })
}

async function openMenus(row: Role) {
  editorVisible.value = false
  userVisible.value = false
  deptVisible.value = false
  menuRole.value = row
  menuSaving.value = false
  try {
    const [menusResponse, assignedResponse] = await Promise.all([
      listMenus(),
      listRoleMenus(tenantId.value, row.id),
    ])
    menuTree.value = labelMenuTree(menusResponse.data ?? [])
    selectedMenuIds.value = assignedResponse.data ?? []
    menuVisible.value = true
  } catch {
    // The HTTP client already shows the server message.
  }
}

async function openDepts(row: Role) {
  editorVisible.value = false
  menuVisible.value = false
  userVisible.value = false
  if (row.data_scope !== 2) {
    feedback.warning('仅自定义数据范围可配置部门')
    return
  }
  deptRole.value = row
  deptSaving.value = false
  try {
    const [orgsResponse, assignedResponse] = await Promise.all([
      listOrganizations(tenantId.value),
      listRoleDepts(tenantId.value, row.id),
    ])
    orgOptions.value = flattenOrganizations(orgsResponse.data ?? [])
    selectedDeptIds.value = assignedResponse.data ?? []
    deptVisible.value = true
  } catch {
    // The HTTP client already shows the server message.
  }
}

async function saveDepts() {
  if (!deptRole.value) {
    return
  }
  deptSaving.value = true
  try {
    await replaceRoleDepts(tenantId.value, deptRole.value.id, selectedDeptIds.value)
    feedback.success('角色部门已保存')
    deptVisible.value = false
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    deptSaving.value = false
  }
}

async function saveMenus() {
  if (!menuRole.value) {
    return
  }
  menuSaving.value = true
  try {
    await replaceRoleMenus(tenantId.value, menuRole.value.id, selectedMenuIds.value)
    feedback.success('角色菜单已保存')
    menuVisible.value = false
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    menuSaving.value = false
  }
}

async function openUsers(row: Role) {
  editorVisible.value = false
  menuVisible.value = false
  deptVisible.value = false
  userRole.value = row
  userSaving.value = false
  try {
    const [membersResponse, assignedResponse] = await Promise.all([
      listMembers(tenantId.value, { page: 1, page_size: 100 }),
      listRoleUsers(tenantId.value, row.id),
    ])
    members.value = membersResponse.data.items
    selectedUserIds.value = (assignedResponse.data ?? []).map((item) => item.user_id)
    userVisible.value = true
  } catch {
    // The HTTP client already shows the server message.
  }
}

async function saveUsers() {
  if (!userRole.value) {
    return
  }
  userSaving.value = true
  try {
    await replaceRoleUsers(tenantId.value, userRole.value.id, selectedUserIds.value)
    feedback.success('角色用户已保存')
    userVisible.value = false
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    userSaving.value = false
  }
}

function labelMenuTree(nodes: MenuNode[]): MenuNode[] {
  return nodes.map((node) => ({
    ...node,
    name: node.type === 3 && node.permission ? `${node.name}（${node.permission}）` : node.name,
    children: labelMenuTree(node.children ?? []),
  }))
}

onMounted(() => {
  void bootstrap()
})
</script>

<template>
  <div class="page-grid">
    <t-card :bordered="false">
      <div class="toolbar">
        <t-input v-model="keyword" clearable placeholder="搜索名称或权限字符串" @enter="search" />
        <t-button theme="default" @click="search">查询</t-button>
        <t-button v-if="canCreate" theme="primary" :disabled="!tenantId" @click="openCreate">新建角色</t-button>
      </div>
      <PageStatus :status="pageStatus" :error-message="errorMessage" @retry="bootstrap">
        <t-table row-key="id" :data="items" :columns="columns" hover>
          <template #type="{ row }">{{ roleTypeLabel(row.type) }}</template>
          <template #data_scope="{ row }">{{ dataScopeLabel(row.data_scope) }}</template>
          <template #remark="{ row }">{{ row.remark || '—' }}</template>
          <template #status="{ row }">
            <t-tag :theme="row.status === 0 ? 'success' : 'warning'" variant="light">
              {{ statusLabel(row.status) }}
            </t-tag>
          </template>
          <template #actions="{ row }">
            <t-button v-if="canUpdate" theme="primary" variant="text" @click="openEdit(row)">编辑</t-button>
            <t-button v-if="canUpdate" theme="primary" variant="text" @click="openMenus(row)">菜单</t-button>
            <t-button
              v-if="canUpdate"
              theme="primary"
              variant="text"
              :disabled="row.data_scope !== 2"
              @click="openDepts(row)"
            >
              部门
            </t-button>
            <t-button v-if="canUpdate" theme="primary" variant="text" @click="openUsers(row)">用户</t-button>
            <t-button v-if="canDelete" theme="danger" variant="text" @click="remove(row)">删除</t-button>
          </template>
        </t-table>
        <t-pagination v-model="page" v-model:page-size="pageSize" class="pager" :total="total" @change="load" />
      </PageStatus>
    </t-card>

    <t-dialog
      v-model:visible="editorVisible"
      :header="editing ? '编辑角色' : '新建角色'"
      :confirm-btn="{ content: '保存', loading: saving }"
      @confirm="save"
    >
      <t-form label-align="top">
        <t-form-item label="名称">
          <t-input v-model="form.name" maxlength="30" />
        </t-form-item>
        <t-form-item label="权限字符串">
          <t-input v-model="form.code" maxlength="100" />
        </t-form-item>
        <t-form-item label="类型">
          <t-radio-group v-model="form.type">
            <t-radio :value="1">系统</t-radio>
            <t-radio :value="2">自定义</t-radio>
          </t-radio-group>
        </t-form-item>
        <t-form-item label="数据范围">
          <t-select
            v-model="form.dataScope"
            :options="[
              { label: '全部数据', value: 1 },
              { label: '自定义部门', value: 2 },
              { label: '本部门', value: 3 },
              { label: '本部门及以下', value: 4 },
              { label: '仅本人', value: 5 },
            ]"
          />
        </t-form-item>
        <t-form-item label="排序">
          <t-input-number v-model="form.sort" :min="0" :max="1000000" />
        </t-form-item>
        <t-form-item label="备注">
          <t-input v-model="form.remark" maxlength="500" />
        </t-form-item>
        <t-form-item v-if="editing" label="状态">
          <t-radio-group v-model="form.status">
            <t-radio :value="0">正常</t-radio>
            <t-radio :value="1">停用</t-radio>
          </t-radio-group>
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="menuVisible"
      :header="menuRole ? `分配菜单 · ${menuRole.name}` : '分配菜单'"
      :confirm-btn="{ content: '保存', loading: menuSaving }"
      width="520px"
      @confirm="saveMenus"
    >
      <t-tree
        v-if="menuTree.length > 0"
        v-model:value="selectedMenuIds"
        :data="menuTree"
        :keys="{ value: 'id', label: 'name', children: 'children' }"
        checkable
        expand-all
        hover
        line
      />
      <p v-else class="hint">还没有菜单</p>
    </t-dialog>

    <t-dialog
      v-model:visible="deptVisible"
      :header="deptRole ? `数据权限部门 · ${deptRole.name}` : '数据权限部门'"
      :confirm-btn="{ content: '保存', loading: deptSaving }"
      width="520px"
      @confirm="saveDepts"
    >
      <t-checkbox-group v-model="selectedDeptIds" class="role-user-list">
        <t-checkbox v-for="org in orgOptions" :key="org.id" :value="org.id">
          {{ org.label }}
        </t-checkbox>
      </t-checkbox-group>
      <p v-if="orgOptions.length === 0" class="hint">该租户还没有部门</p>
    </t-dialog>

    <t-dialog
      v-model:visible="userVisible"
      :header="userRole ? `分配用户 · ${userRole.name}` : '分配用户'"
      :confirm-btn="{ content: '保存', loading: userSaving }"
      width="520px"
      @confirm="saveUsers"
    >
      <t-checkbox-group v-model="selectedUserIds" class="role-user-list">
        <t-checkbox v-for="member in members" :key="member.user_id" :value="member.user_id">
          {{ member.nickname }}（{{ member.username }}）
        </t-checkbox>
      </t-checkbox-group>
      <p v-if="members.length === 0" class="hint">该租户还没有成员</p>
    </t-dialog>
  </div>
</template>
