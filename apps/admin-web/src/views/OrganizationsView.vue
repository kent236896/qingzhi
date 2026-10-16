<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import PageStatus from '@/components/PageStatus.vue'
import {
  createOrganization,
  deleteOrganization,
  flattenOrganizations,
  getTenant,
  listOrganizations,
  statusLabel,
  updateOrganization,
  type OrganizationNode,
  type RecordStatus,
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
const tenantId = computed(() => auth.tenantId)
const tenantName = ref('')
const tree = ref<OrganizationNode[]>([])
const selectedId = ref('')

const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<OrganizationNode | null>(null)
const form = ref({
  name: '',
  code: '',
  parentId: '',
  sort: 0,
  phone: '',
  email: '',
  status: 0 as RecordStatus,
})

const flat = computed(() => flattenOrganizations(tree.value))
const selected = computed(() => flatNode(tree.value, selectedId.value))
const editingRoot = computed(() => Boolean(editing.value && !editing.value.parent_id))
const parentOptions = computed(() => {
  const blocked = descendantIds(editing.value)
  return flat.value.filter((item) => !blocked.has(item.id))
})
const parentName = computed(() => {
  if (!selected.value?.parent_id) {
    return '主体组织'
  }
  return flat.value.find((item) => item.id === selected.value?.parent_id)?.label || '—'
})
const childCount = computed(() => selected.value?.children?.length ?? 0)
const selectedIsRoot = computed(() => Boolean(selected.value && !selected.value.parent_id))

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
  if (!tenantId.value) {
    pageStatus.value = 'error'
    errorMessage.value = '当前登录未绑定租户'
    return
  }
  try {
    const response = await getTenant(tenantId.value)
    tenantName.value = response.data.name
  } catch {
    tenantName.value = ''
  }
  await loadTree()
}

async function loadTree() {
  if (!tenantId.value) {
    tree.value = []
    return
  }
  pageStatus.value = 'loading'
  try {
    const response = await listOrganizations(tenantId.value)
    tree.value = response.data ?? []
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

function openCreate() {
  const parentId = selectedId.value || tree.value[0]?.id || ''
  if (!parentId) {
    feedback.warning('当前租户还没有主体组织')
    return
  }
  editing.value = null
  form.value = {
    name: '',
    code: '',
    parentId,
    sort: 0,
    phone: '',
    email: '',
    status: 0,
  }
  editorVisible.value = true
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
    sort: selected.value.sort,
    phone: selected.value.phone,
    email: selected.value.email,
    status: selected.value.status,
  }
  editorVisible.value = true
}

async function save() {
  if (!tenantId.value || !form.value.name.trim() || !form.value.code.trim()) {
    feedback.warning('请填写名称和编码')
    return
  }
  if (!editingRoot.value && !form.value.parentId) {
    feedback.warning('请选择同一个租户组织树中的上级')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await updateOrganization(tenantId.value, editing.value.id, {
        name: form.value.name.trim(),
        code: form.value.code.trim(),
        parent_id: editingRoot.value ? undefined : form.value.parentId,
        sort: Number(form.value.sort) || 0,
        phone: form.value.phone.trim(),
        email: form.value.email.trim(),
        status: form.value.status,
      })
      feedback.success('组织已更新')
    } else {
      await createOrganization(tenantId.value, {
        name: form.value.name.trim(),
        code: form.value.code.trim(),
        parent_id: form.value.parentId,
        sort: Number(form.value.sort) || 0,
        phone: form.value.phone.trim() || undefined,
        email: form.value.email.trim() || undefined,
      })
      feedback.success('下级组织已创建')
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
  if (!selected.value || !tenantId.value) {
    feedback.warning('请先选择组织')
    return
  }
  if (!selected.value.parent_id) {
    feedback.warning('主体组织不能删除')
    return
  }
  const current = selected.value
  const dialog = DialogPlugin.confirm({
    header: '删除部门',
    body: `确定删除「${current.name}」？有下级或任职时不能删除。`,
    onConfirm: async () => {
      try {
        await deleteOrganization(tenantId.value, current.id)
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
      <p class="hint">当前租户：{{ tenantName || '当前登录租户' }}。这里只加载该租户的主体组织和部门，不能挂到其他租户。</p>
      <div class="org-panel__bar">
        <h3>组织树</h3>
        <div class="toolbar">
          <t-button v-if="canCreate" theme="primary" :disabled="!tenantId || tree.length === 0" @click="openCreate">
            新建下级
          </t-button>
          <t-button v-if="canUpdate" theme="default" :disabled="!selected" @click="openEdit">编辑</t-button>
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
        :description="tenantId ? '当前租户还没有主体组织' : '请先创建租户'"
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
              <t-tag v-if="!node.data?.parent_id" size="small" theme="primary" variant="light">主体</t-tag>
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
          <t-descriptions-item label="所属租户">{{ tenantName || '当前登录租户' }}</t-descriptions-item>
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
      :header="editing ? (editingRoot ? '编辑主体组织' : '编辑部门') : '新建下级组织'"
      :confirm-btn="{ content: '保存', loading: saving, disabled: saving }"
      :cancel-btn="{ content: '取消', disabled: saving }"
      @confirm="save"
    >
      <t-form label-align="top">
        <t-form-item label="名称">
          <t-input v-model="form.name" maxlength="30" />
        </t-form-item>
        <t-form-item label="编码">
          <t-input v-model="form.code" maxlength="64" />
        </t-form-item>
        <t-form-item v-if="editingRoot" label="上级部门">
          <t-input value="主体组织固定属于当前租户" disabled />
        </t-form-item>
        <t-form-item v-else label="上级部门">
          <t-select
            v-model="form.parentId"
            placeholder="选择当前租户组织树中的上级"
            :options="parentOptions.map((item) => ({ label: item.label, value: item.id }))"
          />
        </t-form-item>
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
