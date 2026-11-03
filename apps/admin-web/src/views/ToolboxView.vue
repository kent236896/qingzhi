<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/PageStatus.vue'
import {
  createMCP,
  deleteMCP,
  getCatalogOptions,
  listMCP,
  putMCPCredentials,
  refreshMCPMetadata,
  testMCP,
  updateMCP,
  type CatalogItem,
  type CatalogOptions,
  type CatalogScope,
  type MCPAuthMode,
} from '@/api/catalog'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canManage = computed(() => auth.hasPermission('toolbox:manage'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const items = ref<CatalogItem[]>([])
const editorVisible = ref(false)
const credVisible = ref(false)
const editingID = ref('')
const credTarget = ref<CatalogItem | null>(null)
const saving = ref(false)
const testingID = ref('')
const apiKeyInput = ref('')
const options = ref<CatalogOptions>({
  tenants: [],
  organizations: [],
  roles: [],
  users: [],
  can_manage_agents: false,
  can_manage_toolbox: false,
})
const form = ref({
  name: '',
  description: '',
  scope_type: 'personal' as CatalogScope,
  scope_id: '',
  tenant_ids: [] as string[],
  share_user_ids: [] as string[],
  transport_type: 'sse' as 'sse' | 'http-streamable',
  url: '',
  auth_mode: 'shared' as MCPAuthMode,
})

const scopeLabel: Record<string, string> = {
  personal: '个人',
  tenant: '租户',
  org_node: '组织',
  role: '角色',
}

const authLabel: Record<string, string> = {
  shared: '共享凭证',
  user_oauth: '用户 OAuth',
}

const columns = [
  { colKey: 'name', title: '名称' },
  { colKey: 'transport_type', title: '传输' },
  { colKey: 'auth_mode', title: '鉴权' },
  { colKey: 'scope_type', title: '归属' },
  { colKey: 'description', title: '描述' },
  { colKey: 'creator', title: '创建人' },
  { colKey: 'actions', title: '操作', width: 320 },
]

const scopeChoices = computed(() => {
  if (form.value.scope_type === 'org_node') return options.value.organizations
  if (form.value.scope_type === 'role') return options.value.roles
  if (form.value.scope_type === 'tenant') return options.value.tenants
  return []
})

const dialogHeader = computed(() => (editingID.value ? '编辑 MCP' : '登记 MCP'))

async function load() {
  pageStatus.value = items.value.length ? 'ready' : 'loading'
  try {
    const response = await listMCP()
    items.value = response.data ?? []
    pageStatus.value = 'ready'
  } catch {
    if (!items.value.length) pageStatus.value = 'error'
  }
}

async function ensureOptions() {
  const response = await getCatalogOptions()
  options.value = response.data
}

async function openCreate() {
  await ensureOptions()
  editingID.value = ''
  form.value = {
    name: '',
    description: '',
    scope_type: 'personal',
    scope_id: '',
    tenant_ids: options.value.tenants.some((item) => item.id === auth.tenantId)
      ? [auth.tenantId]
      : options.value.tenants[0]
        ? [options.value.tenants[0].id]
        : [],
    share_user_ids: [],
    transport_type: 'sse',
    url: '',
    auth_mode: 'shared',
  }
  editorVisible.value = true
}

async function openEdit(row: CatalogItem) {
  await ensureOptions()
  editingID.value = row.id
  form.value = {
    name: row.name,
    description: row.description,
    scope_type: row.scope_type,
    scope_id: row.scope_id,
    tenant_ids: [...(row.tenant_ids ?? [])],
    share_user_ids: [...(row.share_user_ids ?? [])],
    transport_type: (row.transport_type as 'sse' | 'http-streamable') || 'sse',
    url: row.url || '',
    auth_mode: (row.auth_mode as MCPAuthMode) || 'shared',
  }
  editorVisible.value = true
}

async function save() {
  if (!form.value.name.trim() || !form.value.url.trim()) {
    feedback.warning('请填写名称与 MCP 地址')
    return
  }
  saving.value = true
  try {
    if (editingID.value) {
      await updateMCP(editingID.value, {
        name: form.value.name,
        description: form.value.description,
        share_user_ids: form.value.scope_type === 'personal' ? form.value.share_user_ids : undefined,
        transport_type: form.value.transport_type,
        url: form.value.url,
        auth_mode: form.value.auth_mode,
      })
      feedback.success('MCP 已更新')
    } else {
      await createMCP({
        ...form.value,
        scope_id: form.value.scope_type === 'personal' ? undefined : form.value.scope_id,
        share_user_ids: form.value.scope_type === 'personal' ? form.value.share_user_ids : undefined,
      })
      feedback.success('MCP 已登记')
    }
    editorVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onTest(row: CatalogItem) {
  testingID.value = row.id
  try {
    await testMCP(row.id)
    feedback.success('连通性测试完成')
  } finally {
    testingID.value = ''
  }
}

async function onRefreshMeta(row: CatalogItem) {
  testingID.value = row.id
  try {
    await refreshMCPMetadata(row.id)
    feedback.success('元数据已刷新')
  } finally {
    testingID.value = ''
  }
}

function openCreds(row: CatalogItem) {
  credTarget.value = row
  apiKeyInput.value = ''
  credVisible.value = true
}

async function saveCreds() {
  if (!credTarget.value) return
  if (!apiKeyInput.value.trim()) {
    feedback.warning('请填写 API Key')
    return
  }
  saving.value = true
  try {
    await putMCPCredentials(credTarget.value.id, { api_key: apiKeyInput.value.trim() })
    feedback.success('共享凭证已更新（密钥不下发前端回显）')
    credVisible.value = false
    apiKeyInput.value = ''
  } finally {
    saving.value = false
  }
}

function remove(row: CatalogItem) {
  const dialog = DialogPlugin.confirm({
    header: '删除 MCP',
    body: `确定删除「${row.name}」吗？`,
    onConfirm: async () => {
      await deleteMCP(row.id)
      feedback.success('已删除')
      dialog.hide()
      await load()
    },
  })
}

onMounted(load)
</script>

<template>
  <div>
    <PageStatus v-if="pageStatus !== 'ready'" :status="pageStatus" @retry="load" />
    <t-card v-else :bordered="false">
      <div class="toolbar">
        <p class="hint">工具箱仅支持 MCP（SSE / HTTP Streamable）。共享凭证仅管理面可写；用户 OAuth 下阶段接通。</p>
        <t-button theme="primary" @click="openCreate">登记 MCP</t-button>
      </div>
      <t-table row-key="id" :data="items" :columns="columns">
        <template #scope_type="{ row }">{{ scopeLabel[row.scope_type] ?? row.scope_type }}</template>
        <template #auth_mode="{ row }">{{ authLabel[row.auth_mode || 'shared'] }}</template>
        <template #actions="{ row }">
          <t-button variant="text" :loading="testingID === row.id" @click="onTest(row)">测试</t-button>
          <t-button
            v-if="row.writable || canManage"
            variant="text"
            :loading="testingID === row.id"
            @click="onRefreshMeta(row)"
          >
            同步工具
          </t-button>
          <t-button v-if="row.writable || canManage" variant="text" @click="openEdit(row)">编辑</t-button>
          <t-button
            v-if="(row.writable || canManage) && (row.auth_mode || 'shared') === 'shared'"
            variant="text"
            @click="openCreds(row)"
          >
            凭证
          </t-button>
          <t-button
            v-if="row.writable || canManage"
            theme="danger"
            variant="text"
            @click="remove(row)"
          >
            删除
          </t-button>
        </template>
      </t-table>
    </t-card>

    <t-dialog
      v-model:visible="editorVisible"
      :header="dialogHeader"
      :confirm-btn="{ content: editingID ? '保存' : '创建', loading: saving }"
      @confirm="save"
    >
      <t-form label-align="top">
        <t-form-item label="名称"><t-input v-model="form.name" maxlength="100" /></t-form-item>
        <t-form-item label="描述"><t-textarea v-model="form.description" maxlength="500" /></t-form-item>
        <t-form-item label="传输协议">
          <t-radio-group v-model="form.transport_type">
            <t-radio value="sse">SSE</t-radio>
            <t-radio value="http-streamable">HTTP Streamable</t-radio>
          </t-radio-group>
        </t-form-item>
        <t-form-item label="服务地址"><t-input v-model="form.url" placeholder="https://..." /></t-form-item>
        <t-form-item label="鉴权模式">
          <t-radio-group v-model="form.auth_mode">
            <t-radio value="shared">共享静态凭证</t-radio>
            <t-radio value="user_oauth">用户 OAuth</t-radio>
          </t-radio-group>
        </t-form-item>
        <template v-if="!editingID">
          <t-form-item label="租户">
            <t-select
              v-model="form.tenant_ids"
              multiple
              :options="options.tenants.map((item) => ({ label: item.name, value: item.id }))"
            />
          </t-form-item>
          <t-form-item label="归属">
            <t-radio-group v-model="form.scope_type">
              <t-radio value="personal">个人</t-radio>
              <t-radio v-if="canManage" value="tenant">租户</t-radio>
              <t-radio v-if="canManage" value="org_node">组织</t-radio>
              <t-radio v-if="canManage" value="role">角色</t-radio>
            </t-radio-group>
          </t-form-item>
          <t-form-item v-if="form.scope_type !== 'personal'" label="归属对象">
            <t-select
              v-model="form.scope_id"
              :options="scopeChoices.map((item) => ({ label: item.name, value: item.id }))"
            />
          </t-form-item>
        </template>
        <t-form-item v-if="form.scope_type === 'personal'" label="分享给">
          <t-select
            v-model="form.share_user_ids"
            multiple
            filterable
            :options="options.users.map((item) => ({ label: item.name, value: item.id }))"
          />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="credVisible"
      header="写入共享凭证"
      :confirm-btn="{ content: '保存', loading: saving }"
      @confirm="saveCreds"
    >
      <t-form label-align="top">
        <t-form-item :label="`MCP：${credTarget?.name || ''}`">
          <t-input
            v-model="apiKeyInput"
            type="password"
            placeholder="api_key（仅本次提交，不会回显）"
          />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}
.hint {
  margin: 0;
  color: var(--qz-text-secondary);
}
</style>
