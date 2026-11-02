<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/PageStatus.vue'
import {
  createMCP,
  deleteMCP,
  getCatalogOptions,
  listMCP,
  testMCP,
  type CatalogItem,
  type CatalogOptions,
  type CatalogScope,
} from '@/api/catalog'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canManage = computed(() => auth.hasPermission('toolbox:manage'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const items = ref<CatalogItem[]>([])
const editorVisible = ref(false)
const saving = ref(false)
const testingID = ref('')
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
})

const scopeLabel: Record<string, string> = {
  personal: '个人',
  tenant: '租户',
  org_node: '组织',
  role: '角色',
}

const columns = [
  { colKey: 'name', title: '名称' },
  { colKey: 'transport_type', title: '传输' },
  { colKey: 'scope_type', title: '归属' },
  { colKey: 'description', title: '描述' },
  { colKey: 'creator', title: '创建人' },
  { colKey: 'actions', title: '操作', width: 180 },
]

const scopeChoices = computed(() => {
  if (form.value.scope_type === 'org_node') return options.value.organizations
  if (form.value.scope_type === 'role') return options.value.roles
  if (form.value.scope_type === 'tenant') return options.value.tenants
  return []
})

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

async function openCreate() {
  const response = await getCatalogOptions()
  options.value = response.data
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
    await createMCP({
      ...form.value,
      scope_id: form.value.scope_type === 'personal' ? undefined : form.value.scope_id,
      share_user_ids: form.value.scope_type === 'personal' ? form.value.share_user_ids : undefined,
    })
    feedback.success('MCP 已登记')
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
        <p class="hint">工具箱仅支持 MCP（SSE / HTTP Streamable）。</p>
        <t-button theme="primary" @click="openCreate">登记 MCP</t-button>
      </div>
      <t-table row-key="id" :data="items" :columns="columns">
        <template #scope_type="{ row }">{{ scopeLabel[row.scope_type] ?? row.scope_type }}</template>
        <template #actions="{ row }">
          <t-button variant="text" :loading="testingID === row.id" @click="onTest(row)">测试</t-button>
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
      header="登记 MCP"
      :confirm-btn="{ content: '创建', loading: saving }"
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
        <t-form-item v-else label="分享给">
          <t-select
            v-model="form.share_user_ids"
            multiple
            filterable
            :options="options.users.map((item) => ({ label: item.name, value: item.id }))"
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
