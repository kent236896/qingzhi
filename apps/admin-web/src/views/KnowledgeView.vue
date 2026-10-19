<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/PageStatus.vue'
import {
  createKnowledgeBase,
  deleteKnowledgeBase,
  getKnowledgeOptions,
  listKnowledgeBases,
  type FormOptions,
  type KnowledgeBase,
} from '@/api/knowledge'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()
const canManage = computed(() => auth.hasPermission('knowledge:manage'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const items = ref<KnowledgeBase[]>([])
const total = ref(0)
const page = ref(1)
const keyword = ref('')
const scopeType = ref('')
const editorVisible = ref(false)
const saving = ref(false)
const options = ref<FormOptions>({ tenants: [], organizations: [], posts: [], roles: [], users: [], can_manage: false })
const form = ref({
  name: '',
  description: '',
  scope_type: 'personal',
  scope_id: '',
  tenant_ids: [] as string[],
  share_user_ids: [] as string[],
  wiki_enabled: false,
})

const scopeLabel: Record<string, string> = {
  personal: '个人',
  org_node: '组织',
  position: '岗位',
  role: '角色',
}

const columns = [
  { colKey: 'name', title: '名称' },
  { colKey: 'scope_type', title: '归属' },
  { colKey: 'wiki_enabled', title: 'Wiki' },
  { colKey: 'creator', title: '创建人' },
  { colKey: 'actions', title: '操作', width: 180 },
]

const scopeChoices = computed(() => {
  if (form.value.scope_type === 'org_node') {
    return options.value.organizations
  }
  if (form.value.scope_type === 'position') {
    return options.value.posts
  }
  if (form.value.scope_type === 'role') {
    return options.value.roles
  }
  return []
})

async function load() {
  pageStatus.value = items.value.length ? 'ready' : 'loading'
  try {
    const response = await listKnowledgeBases({ page: page.value, page_size: 20, keyword: keyword.value, scope_type: scopeType.value })
    items.value = response.data.items ?? []
    total.value = response.data.total
    pageStatus.value = 'ready'
  } catch {
    if (!items.value.length) {
      pageStatus.value = 'error'
    }
  }
}

async function openCreate() {
  const response = await getKnowledgeOptions()
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
    wiki_enabled: false,
  }
  editorVisible.value = true
}

async function save() {
  saving.value = true
  try {
    await createKnowledgeBase({
      ...form.value,
      scope_id: form.value.scope_type === 'personal' ? undefined : form.value.scope_id,
      share_user_ids: form.value.scope_type === 'personal' ? form.value.share_user_ids : undefined,
    })
    feedback.success('知识库已创建')
    editorVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

function remove(row: KnowledgeBase) {
  const dialog = DialogPlugin.confirm({
    header: '删除知识库',
    body: `删除「${row.name}」后，里面的文件也会删除。`,
    onConfirm: async () => {
      await deleteKnowledgeBase(row.id)
      feedback.success('知识库已删除')
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
        <t-input v-model="keyword" placeholder="搜索名称" clearable style="width: 220px" @enter="load" />
        <t-select v-model="scopeType" clearable placeholder="归属类型" style="width: 160px" @change="load">
          <t-option value="personal" label="个人" />
          <t-option value="org_node" label="组织" />
          <t-option value="position" label="岗位" />
          <t-option value="role" label="角色" />
        </t-select>
        <t-button theme="primary" @click="openCreate">新建知识库</t-button>
      </div>
      <t-table row-key="id" :data="items" :columns="columns" :pagination="{ current: page, pageSize: 20, total }" @page-change="(info: { current: number }) => { page = info.current; load() }">
        <template #scope_type="{ row }">{{ scopeLabel[row.scope_type] ?? row.scope_type }}</template>
        <template #wiki_enabled="{ row }">{{ row.wiki_enabled ? '已启用' : '未启用' }}</template>
        <template #actions="{ row }">
          <t-button theme="primary" variant="text" @click="router.push({ name: 'knowledge-detail', params: { id: row.id } })">打开</t-button>
          <t-button v-if="canManage || row.scope_type === 'personal'" theme="danger" variant="text" @click="remove(row)">删除</t-button>
        </template>
      </t-table>
    </t-card>

    <t-dialog v-model:visible="editorVisible" header="新建知识库" :confirm-btn="{ content: '创建', loading: saving }" @confirm="save">
      <t-form label-align="top">
        <t-form-item label="名称"><t-input v-model="form.name" maxlength="100" /></t-form-item>
        <t-form-item label="描述"><t-textarea v-model="form.description" maxlength="500" /></t-form-item>
        <t-form-item label="租户">
          <t-select v-model="form.tenant_ids" multiple :options="options.tenants.map((item) => ({ label: item.name, value: item.id }))" />
        </t-form-item>
        <t-form-item label="归属">
          <t-radio-group v-model="form.scope_type">
            <t-radio value="personal">个人</t-radio>
            <t-radio v-if="canManage" value="org_node">组织</t-radio>
            <t-radio v-if="canManage" value="position">岗位</t-radio>
            <t-radio v-if="canManage" value="role">角色</t-radio>
          </t-radio-group>
        </t-form-item>
        <t-form-item v-if="form.scope_type !== 'personal'" label="归属对象">
          <t-select v-model="form.scope_id" :options="scopeChoices.map((item) => ({ label: item.name, value: item.id }))" />
        </t-form-item>
        <t-form-item v-else label="分享给">
          <t-select v-model="form.share_user_ids" multiple :options="options.users.map((item) => ({ label: item.name, value: item.id }))" />
        </t-form-item>
        <t-form-item>
          <t-checkbox v-model="form.wiki_enabled">启用 Wiki 与图谱</t-checkbox>
          <p class="hint">勾选后会为文档自动生成 Wiki 页面和知识图谱，会增加大模型调用耗时，创建后不可更改。</p>
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}
.hint {
  margin: 8px 0 0;
  color: var(--qz-text-secondary);
}
</style>
