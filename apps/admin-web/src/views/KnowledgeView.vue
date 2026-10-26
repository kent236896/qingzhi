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
  shareKnowledgeBase,
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
const shareVisible = ref(false)
const saving = ref(false)
const sharing = ref(false)
const shareTarget = ref<KnowledgeBase | null>(null)
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
const shareForm = ref({
  share_user_ids: [] as string[],
  share_org_ids: [] as string[],
  share_role_ids: [] as string[],
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
  { colKey: 'actions', title: '操作', width: 240 },
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

const shareIsPersonal = computed(() => shareTarget.value?.scope_type === 'personal')
const shareIsScope = computed(
  () => shareTarget.value?.scope_type === 'org_node' || shareTarget.value?.scope_type === 'role',
)

const shareOrgOptions = computed(() => {
  const exclude = shareTarget.value?.scope_type === 'org_node' ? shareTarget.value.scope_id : ''
  return options.value.organizations
    .filter((item) => item.id !== exclude)
    .map((item) => ({ label: item.name, value: item.id }))
})

const shareRoleOptions = computed(() => {
  const exclude = shareTarget.value?.scope_type === 'role' ? shareTarget.value.scope_id : ''
  return options.value.roles
    .filter((item) => item.id !== exclude)
    .map((item) => ({ label: item.name, value: item.id }))
})

function canShareRow(row: KnowledgeBase) {
  if (row.scope_type === 'personal') {
    return row.scope_id === auth.user?.id || row.creator_id === auth.user?.id
  }
  if (row.scope_type === 'org_node' || row.scope_type === 'role') {
    return canManage.value
  }
  return false
}

async function load() {
  pageStatus.value = items.value.length ? 'ready' : 'loading'
  try {
    const response = await listKnowledgeBases({
      page: page.value,
      page_size: 20,
      keyword: keyword.value,
      scope_type: scopeType.value,
    })
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

async function openShare(row: KnowledgeBase) {
  const response = await getKnowledgeOptions()
  options.value = response.data
  shareTarget.value = row
  shareForm.value = {
    share_user_ids: [...(row.share_user_ids ?? [])],
    share_org_ids: [...(row.share_org_ids ?? [])],
    share_role_ids: [...(row.share_role_ids ?? [])],
  }
  shareVisible.value = true
}

function modelName(model?: { id: string; name: string; display_name: string }) {
  if (!model?.id) return '未绑定'
  return model.display_name || model.name || model.id
}

async function save() {
  if (form.value.wiki_enabled && !options.value.models?.wiki_synthesis?.id) {
    feedback.warning('尚未绑定 Wiki 生成模型')
    return
  }
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

async function saveShare() {
  if (!shareTarget.value) {
    return
  }
  sharing.value = true
  try {
    const updated = await shareKnowledgeBase(
      shareTarget.value.id,
      shareIsPersonal.value
        ? { share_user_ids: shareForm.value.share_user_ids }
        : {
            share_org_ids: shareForm.value.share_org_ids,
            share_role_ids: shareForm.value.share_role_ids,
          },
    )
    const index = items.value.findIndex((item) => item.id === updated.data.id)
    if (index >= 0) {
      items.value[index] = updated.data
    }
    feedback.success('分享设置已保存')
    shareVisible.value = false
  } finally {
    sharing.value = false
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
        <t-button theme="primary" variant="base" @click="openCreate">新建知识库</t-button>
      </div>
      <t-table
        row-key="id"
        :data="items"
        :columns="columns"
        :pagination="{ current: page, pageSize: 20, total }"
        @page-change="(info: { current: number }) => { page = info.current; load() }"
      >
        <template #scope_type="{ row }">{{ scopeLabel[row.scope_type] ?? row.scope_type }}</template>
        <template #wiki_enabled="{ row }">{{ row.wiki_enabled ? '已启用' : '未启用' }}</template>
        <template #actions="{ row }">
          <t-button variant="text" @click="router.push({ name: 'knowledge-detail', params: { id: row.id } })">打开</t-button>
          <t-button v-if="canShareRow(row)" variant="text" @click="openShare(row)">分享</t-button>
          <t-button v-if="canManage || row.scope_type === 'personal'" theme="danger" variant="text" @click="remove(row)">
            删除
          </t-button>
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
          <t-select
            v-model="form.share_user_ids"
            multiple
            filterable
            placeholder="可选，创建后也可再改"
            :options="options.users.map((item) => ({ label: item.name, value: item.id }))"
          />
        </t-form-item>
        <t-form-item>
          <t-checkbox v-model="form.wiki_enabled">启用 Wiki 与图谱</t-checkbox>
          <p class="hint">勾选后会为文档自动生成 Wiki 页面和知识图谱，会增加大模型调用耗时，创建后不可更改。</p>
          <p class="hint">将使用默认模型：向量 {{ modelName(options.models?.embedding) }}，摘要 {{ modelName(options.models?.summary) }}。</p>
          <p v-if="form.wiki_enabled && !options.models?.wiki_synthesis?.id" class="hint">
            尚未绑定 Wiki 生成模型，不能提交。
            <router-link :to="{ name: 'settings' }">前往系统配置</router-link>
          </p>
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="shareVisible"
      :header="shareTarget ? `分享「${shareTarget.name}」` : '分享知识库'"
      :confirm-btn="{ content: '保存', loading: sharing }"
      @confirm="saveShare"
    >
      <t-form v-if="shareTarget" label-align="top">
        <template v-if="shareIsPersonal">
          <t-form-item label="分享给用户">
            <t-select
              v-model="shareForm.share_user_ids"
              multiple
              filterable
              placeholder="选择可查看的用户"
              :options="options.users.filter((item) => item.id !== auth.user?.id).map((item) => ({ label: item.name, value: item.id }))"
            />
          </t-form-item>
          <p class="hint">个人知识库只能分享给用户；被分享者可查看和检索，不能改归属。</p>
        </template>
        <template v-else-if="shareIsScope">
          <t-form-item label="分享给组织">
            <t-select
              v-model="shareForm.share_org_ids"
              multiple
              filterable
              placeholder="选择可查看的组织"
              :options="shareOrgOptions"
            />
          </t-form-item>
          <t-form-item label="分享给角色">
            <t-select
              v-model="shareForm.share_role_ids"
              multiple
              filterable
              placeholder="选择可查看的角色"
              :options="shareRoleOptions"
            />
          </t-form-item>
          <p class="hint">组织与角色知识库可互相分享给其他组织或角色；本归属对象不会出现在列表中。</p>
        </template>
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
