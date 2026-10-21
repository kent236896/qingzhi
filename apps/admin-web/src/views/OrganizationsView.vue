<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/PageStatus.vue'
import {
  createOrganization,
  deleteOrganization,
  flattenOrganizations,
  listOrganizations,
  listTenants,
  statusLabel,
  updateOrganization,
  type OrganizationNode,
  type RecordStatus,
  type Tenant,
} from '@/api/identity'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canCreate = computed(() => auth.hasPermission('system:organization:create'))
const canUpdate = computed(() => auth.hasPermission('system:organization:update'))
const canDelete = computed(() => auth.hasPermission('system:organization:delete'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const tenants = ref<Tenant[]>([])
const tree = ref<OrganizationNode[]>([])
const selectedId = ref('')

const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<OrganizationNode | null>(null)
const form = ref({
  name: '',
  code: '',
  parentId: '',
  tenantId: '',
  sort: 0,
  phone: '',
  email: '',
  status: 0 as RecordStatus,
})

const flat = computed(() => flattenOrganizations(tree.value))
const selected = computed(() => flatNode(tree.value, selectedId.value))
const editingRoot = computed(() => Boolean(editing.value && isCompanyRoot(editing.value)))
const selectedIsRoot = computed(() => Boolean(selected.value && isCompanyRoot(selected.value)))
const tenantNameByID = computed(() => new Map(tenants.value.map((item) => [item.id, item.name])))
const parentName = computed(() => {
  if (!selected.value?.parent_id) {
    return '无上级，与其他主体同级'
  }
  return flat.value.find((item) => item.id === selected.value?.parent_id)?.label || '—'
})
const childCount = computed(() => selected.value?.children?.length ?? 0)
const selectedTenantName = computed(() => tenantName(selected.value?.tenant_id || ''))
const rootParentOptions = computed(() => {
  const blocked = descendantIds(editing.value)
  return flatNodes().filter((item) => isCompanyRoot(item) && !blocked.has(item.id) && item.id !== editing.value?.id)
})
const departmentParentOptions = computed(() => {
  const blocked = descendantIds(editing.value)
  const tenantId = editing.value?.tenant_id || ''
  return flatNodes().filter((item) => item.tenant_id === tenantId && !blocked.has(item.id))
})
const placementHint = computed(() => {
  const parent = nodeByID(form.value.parentId)
  if (!form.value.tenantId) {
    return '先选择这个节点隶属的租户。集团和每家子公司各用自己的租户。'
  }
  if (!parent) {
    return '不选上级时，所选租户的主体组织会和其他公司主体处于同级。'
  }
  if (parent.tenant_id === form.value.tenantId) {
    return '租户与上级相同，将在该公司下新建部门。部门不能改到别的租户。'
  }
  return '租户与上级不同，会把所选租户的主体组织挂到这里，作为子公司。'
})

function flatNode(nodes: OrganizationNode[], id: string): OrganizationNode | undefined {
  for (const node of nodes) {
    if (node.id === id) {
      return node
    }
    const child = flatNode(node.children ?? [], id)
    if (child) {
      return child
    }
  }
  return undefined
}

function isCompanyRoot(node: OrganizationNode | undefined | null): boolean {
  return Boolean(node && (node.is_tenant_root || !node.parent_id))
}

function tenantName(id: string): string {
  return tenantNameByID.value.get(id) || '—'
}

function flatNodes(): OrganizationNode[] {
  const items: OrganizationNode[] = []
  const walk = (nodes: OrganizationNode[]) => {
    for (const node of nodes) {
      items.push(node)
      walk(node.children ?? [])
    }
  }
  walk(tree.value)
  return items
}

function nodeByID(id: string): OrganizationNode | undefined {
  return flatNodes().find((item) => item.id === id)
}

function parentNameOf(id: string): string {
  if (!id) {
    return '无上级，作为同级主体'
  }
  const node = nodeByID(id)
  if (!node) {
    return '—'
  }
  return `${node.name}（${tenantName(node.tenant_id)}）`
}

function companyRootOf(tenantId: string): OrganizationNode | undefined {
  return flatNodes().find((item) => item.tenant_id === tenantId && isCompanyRoot(item))
}

function flattenTree(nodes: OrganizationNode[]): OrganizationNode[] {
  const result: OrganizationNode[] = []
  for (const node of nodes) {
    result.push(node)
    result.push(...flattenTree(node.children ?? []))
  }
  return result
}

function buildForest(nodes: OrganizationNode[]): OrganizationNode[] {
  const map = new Map<string, OrganizationNode>()
  for (const node of nodes) {
    map.set(node.id, { ...node, children: [] })
  }
  const roots: OrganizationNode[] = []
  for (const node of map.values()) {
    const parent = node.parent_id ? map.get(node.parent_id) : undefined
    if (!parent) {
      roots.push(node)
      continue
    }
    parent.children.push(node)
  }
  return roots
}

function descendantIds(node: OrganizationNode | null): Set<string> {
  const blocked = new Set<string>()
  const walk = (current: OrganizationNode) => {
    blocked.add(current.id)
    for (const child of current.children ?? []) {
      walk(child)
    }
  }
  if (node) {
    walk(node)
  }
  return blocked
}

async function bootstrap() {
  await loadTree()
}

async function loadTree() {
  pageStatus.value = 'loading'
  try {
    const tenantResponse = await listTenants({ page: 1, page_size: 100 })
    tenants.value = tenantResponse.data.items
    const lists = await Promise.all(
      tenants.value.map(async (tenant) => {
        const response = await listOrganizations(tenant.id)
        return response.data ?? []
      }),
    )
    tree.value = buildForest(lists.flatMap((nodes) => flattenTree(nodes)))
    if (selectedId.value && !flatNode(tree.value, selectedId.value)) {
      selectedId.value = ''
    }
    if (!selectedId.value && tree.value[0]) {
      selectedId.value = tree.value[0].id
    }
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载组织'
    pageStatus.value = 'error'
  }
}

function onActive(value: Array<string | number>) {
  selectedId.value = String(value[0] ?? '')
}

function openCreate(kind: 'sibling' | 'child') {
  if (kind === 'child' && !selected.value) {
    feedback.warning('请先选择上级组织')
    return
  }
  const parentId = kind === 'child' ? selected.value?.id || '' : selected.value?.parent_id || ''
  const tenantId = selected.value?.tenant_id || auth.tenantId
  editing.value = null
  form.value = {
    name: '',
    code: '',
    parentId,
    tenantId,
    sort: 0,
    phone: '',
    email: '',
    status: 0,
  }
  applyExistingRoot(tenantId)
  editorVisible.value = true
}

function applyExistingRoot(tenantId: string) {
  const parent = nodeByID(form.value.parentId)
  const root = companyRootOf(tenantId)
  if (!root || (parent && parent.tenant_id === tenantId)) {
    return
  }
  form.value.name = root.name
  form.value.code = root.code
}

function onFormTenantChange() {
  applyExistingRoot(form.value.tenantId)
}

function openEdit() {
  if (!selected.value) {
    feedback.warning('请先选择部门')
    return
  }
  editing.value = selected.value
  form.value = {
    name: selected.value.name,
    code: selected.value.code,
    parentId: selected.value.parent_id,
    tenantId: selected.value.tenant_id,
    sort: selected.value.sort,
    phone: selected.value.phone,
    email: selected.value.email,
    status: selected.value.status,
  }
  editorVisible.value = true
}

async function save() {
  if (!form.value.tenantId || !form.value.name.trim() || !form.value.code.trim()) {
    feedback.warning('请填写租户、名称和编码')
    return
  }
  const parent = nodeByID(form.value.parentId)
  if (parent && !isCompanyRoot(parent) && parent.tenant_id !== form.value.tenantId) {
    feedback.warning('子公司只能挂在主体组织下面')
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.value.name.trim(),
      code: form.value.code.trim(),
      sort: Number(form.value.sort) || 0,
      phone: form.value.phone.trim(),
      email: form.value.email.trim(),
    }
    if (editing.value) {
      await updateOrganization(editing.value.tenant_id, editing.value.id, {
        ...payload,
        parent_id: form.value.parentId || null,
        status: form.value.status,
      })
      feedback.success('组织已更新')
    } else if (parent && parent.tenant_id === form.value.tenantId) {
      await createOrganization(form.value.tenantId, {
        ...payload,
        parent_id: form.value.parentId,
        phone: payload.phone || undefined,
        email: payload.email || undefined,
      })
      feedback.success('部门已创建')
    } else {
      const root = companyRootOf(form.value.tenantId)
      if (!parent && root && !root.parent_id) {
        feedback.warning('该租户的主体组织已经是同级节点。新建部门请使用新建下级；调整其他公司请更换租户。')
        saving.value = false
        return
      }
      if (root) {
        await updateOrganization(form.value.tenantId, root.id, {
          ...payload,
          parent_id: form.value.parentId || null,
        })
        feedback.success(form.value.parentId ? '子公司已挂到所选组织下' : '主体组织已调整为同级')
      } else {
        await createOrganization(form.value.tenantId, {
          ...payload,
          parent_id: form.value.parentId || undefined,
          phone: payload.phone || undefined,
          email: payload.email || undefined,
        })
        feedback.success('主体组织已创建')
      }
    }
    editorVisible.value = false
    await loadTree()
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    saving.value = false
  }
}

function removeSelected() {
  if (!selected.value) {
    feedback.warning('请先选择组织')
    return
  }
  if (isCompanyRoot(selected.value)) {
    feedback.warning('主体组织不能删除')
    return
  }
  const current = selected.value
  const dialog = DialogPlugin.confirm({
    header: '删除部门',
    body: `确定删除「${current.name}」？有下级或任职时不能删除。`,
    onConfirm: async () => {
      try {
        await deleteOrganization(current.tenant_id, current.id)
        feedback.success('部门已删除')
        selectedId.value = ''
        dialog.hide()
        await loadTree()
      } catch {
        dialog.hide()
      }
    },
  })
}

onMounted(() => {
  void bootstrap()
})
</script>

<template>
  <div class="org-page">
    <section class="org-panel">
      <p class="hint">集团和每家子公司各有一个主体组织，并隶属各自的租户。主体之间可以同级，也可以把子公司挂在集团下面。部门跟随本公司租户，不能改到别的租户。</p>
      <div class="org-panel__bar">
        <h3>组织树</h3>
        <div class="toolbar">
          <t-button v-if="canCreate" theme="primary" variant="outline" @click="openCreate('sibling')">新建同级</t-button>
          <t-button v-if="canCreate" theme="primary" variant="base" :disabled="!selected" @click="openCreate('child')">
            新建下级
          </t-button>
          <t-button v-if="canUpdate" theme="default" variant="outline" :disabled="!selected" @click="openEdit">编辑</t-button>
          <t-divider v-if="canDelete" layout="vertical" />
          <t-button
            v-if="canDelete"
            theme="danger"
            variant="outline"
            :disabled="!selected || selectedIsRoot"
            @click="removeSelected"
          >
            删除
          </t-button>
        </div>
      </div>
      <PageStatus
        :status="pageStatus === 'ready' && tree.length === 0 ? 'empty' : pageStatus"
        :error-message="errorMessage"
        description="还没有主体组织，请先创建租户"
        @retry="bootstrap"
      >
        <t-tree
          :data="tree"
          :keys="{ value: 'id', label: 'name', children: 'children' }"
          :actived="selectedId ? [selectedId] : []"
          activable
          hover
          expand-all
          line
          @active="onActive"
        >
          <template #label="{ node }">
            <span class="org-node">
              <span>{{ node.label }}</span>
              <t-tag v-if="node.data?.is_tenant_root || !node.data?.parent_id" size="small" theme="primary" variant="light">
                主体
              </t-tag>
              <t-tag v-if="node.data?.tenant_id" size="small" variant="light">
                {{ tenantName(node.data.tenant_id) }}
              </t-tag>
              <t-tag v-if="node.data?.status === 1" size="small" theme="warning" variant="light">停用</t-tag>
            </span>
          </template>
        </t-tree>
      </PageStatus>
    </section>

    <section class="org-panel">
      <div class="org-panel__bar">
        <h3>组织详情</h3>
      </div>
      <div v-if="selected" class="org-detail">
        <div class="org-detail__title">
          <strong>{{ selected.name }}</strong>
          <t-tag theme="primary" variant="light">{{ selectedIsRoot ? '主体组织' : '部门' }}</t-tag>
          <t-tag :theme="selected.status === 0 ? 'success' : 'warning'" variant="light">
            {{ statusLabel(selected.status) }}
          </t-tag>
        </div>
        <t-descriptions :column="1" item-layout="horizontal" size="medium">
          <t-descriptions-item label="所属租户">{{ selectedTenantName }}</t-descriptions-item>
          <t-descriptions-item label="编码">{{ selected.code || '—' }}</t-descriptions-item>
          <t-descriptions-item label="上级部门">{{ parentName }}</t-descriptions-item>
          <t-descriptions-item label="下级数量">{{ childCount }}</t-descriptions-item>
          <t-descriptions-item label="排序">{{ selected.sort }}</t-descriptions-item>
          <t-descriptions-item label="联系电话">{{ selected.phone || '—' }}</t-descriptions-item>
          <t-descriptions-item label="邮箱">{{ selected.email || '—' }}</t-descriptions-item>
        </t-descriptions>
      </div>
      <div v-else class="page-status">
        <t-empty description="从左侧选择一个组织" />
      </div>
    </section>

    <t-dialog
      v-model:visible="editorVisible"
      width="520px"
      :header="editing ? (editingRoot ? '编辑主体组织' : '编辑部门') : '新建组织'"
      :confirm-btn="{ content: '保存', loading: saving, disabled: saving }"
      :cancel-btn="{ content: '取消', disabled: saving }"
      @confirm="save"
    >
      <t-form label-align="top">
        <t-form-item label="租户">
          <t-select
            v-if="!editing"
            v-model="form.tenantId"
            placeholder="选择集团或子公司租户"
            :options="tenants.map((item) => ({ label: `${item.name}（${item.code}）`, value: item.id }))"
            @change="onFormTenantChange"
          />
          <t-input v-else :value="tenantName(editing?.tenant_id || '')" disabled />
        </t-form-item>
        <t-form-item label="名称">
          <t-input v-model="form.name" maxlength="30" />
        </t-form-item>
        <t-form-item label="编码">
          <t-input v-model="form.code" maxlength="64" />
        </t-form-item>
        <t-form-item v-if="editing && editingRoot" label="上级主体">
          <t-select
            v-model="form.parentId"
            clearable
            placeholder="不选则与其他主体同级"
            :options="rootParentOptions.map((item) => ({ label: `${item.name}（${tenantName(item.tenant_id)}）`, value: item.id }))"
          />
        </t-form-item>
        <t-form-item v-else-if="editing" label="上级部门">
          <t-select
            v-model="form.parentId"
            placeholder="选择本公司内的上级"
            :options="departmentParentOptions.map((item) => ({ label: item.name, value: item.id }))"
          />
        </t-form-item>
        <t-form-item v-else label="上级组织">
          <t-input :value="parentNameOf(form.parentId)" disabled />
        </t-form-item>
        <p v-if="!editing" class="hint">{{ placementHint }}</p>
        <t-form-item label="排序">
          <t-input-number v-model="form.sort" :min="0" :max="1000000" />
        </t-form-item>
        <t-form-item label="联系电话">
          <t-input v-model="form.phone" maxlength="11" />
        </t-form-item>
        <t-form-item label="邮箱">
          <t-input v-model="form.email" maxlength="50" />
        </t-form-item>
        <t-form-item v-if="editing" label="状态">
          <t-radio-group v-model="form.status">
            <t-radio :value="0">正常</t-radio>
            <t-radio :value="1">停用</t-radio>
          </t-radio-group>
        </t-form-item>
        <p v-if="editing" class="hint">当前状态：{{ statusLabel(editing.status) }}</p>
      </t-form>
    </t-dialog>
  </div>
</template>

<style scoped>
.org-page {
  display: grid;
  grid-template-columns: minmax(320px, 420px) minmax(0, 1fr);
  gap: var(--qz-space-4);
  align-items: start;
}

.org-panel {
  min-height: 420px;
  padding: var(--qz-card-padding);
  border: 1px solid var(--qz-border);
  border-radius: var(--qz-radius);
  background: var(--qz-surface);
  box-shadow: none;
}

.org-panel__bar {
  display: flex;
  flex-wrap: wrap;
  gap: var(--qz-space-3);
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--qz-space-3);
}

.org-panel__bar h3 {
  margin: 0;
  font-size: var(--qz-font-lg);
  font-weight: 600;
}

.org-panel__bar .toolbar {
  margin-bottom: 0;
}

.org-node {
  display: inline-flex;
  gap: var(--qz-space-2);
  align-items: center;
}

.org-detail__title {
  display: flex;
  gap: var(--qz-space-2);
  align-items: center;
  margin-bottom: var(--qz-space-4);
  font-size: var(--qz-font-lg);
}

.org-panel :deep(.t-tree__label:hover) {
  background: var(--qz-hover);
}

.org-panel :deep(.t-is-active > .t-tree__label),
.org-panel :deep(.t-tree__label.t-is-active) {
  color: var(--qz-brand);
  font-weight: 600;
  background: var(--qz-brand-light);
}

@media (max-width: 960px) {
  .org-page {
    grid-template-columns: 1fr;
  }
}
</style>
