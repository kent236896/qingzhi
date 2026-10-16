<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import {
  AddIcon,
  DeleteIcon,
  EditIcon,
  FolderIcon,
  KeyIcon,
  RefreshIcon,
  RootListIcon,
  SearchIcon,
} from 'tdesign-icons-vue-next'
import PageStatus from '@/components/PageStatus.vue'
import {
  createMenu,
  deleteMenu,
  flattenMenus,
  listMenus,
  menuTypeLabel,
  statusLabel,
  updateMenu,
  type MenuNode,
  type MenuType,
  type RecordStatus,
} from '@/api/identity'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canCreate = computed(() => auth.hasPermission('system:menu:create'))
const canUpdate = computed(() => auth.hasPermission('system:menu:update'))
const canDelete = computed(() => auth.hasPermission('system:menu:delete'))
const pageStatus = ref<'loading' | 'error' | 'ready'>('loading')
const errorMessage = ref('')
const tree = ref<MenuNode[]>([])
const selectedId = ref('')
const keyword = ref('')
/** 默认只展开目录，菜单下的按钮保持折叠。 */
const expanded = ref<string[]>([])

const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<MenuNode | null>(null)
const form = ref({
  name: '',
  type: 2 as MenuType,
  permission: '',
  parentId: '',
  sort: 0,
  path: '',
  icon: '#',
  component: '',
  componentName: '',
  visible: true,
  keepAlive: true,
  alwaysShow: true,
  status: 0 as RecordStatus,
})

const flat = computed(() => flattenMenus(tree.value))
const selected = computed(() => findNode(tree.value, selectedId.value))
const blockedParentIds = computed(() => (editing.value ? collectIds(editing.value) : []))
const parentOptions = computed(() =>
  flat.value.filter((item) => {
    if (blockedParentIds.value.includes(item.id)) {
      return false
    }
    if (form.value.type === 3) {
      return item.type === 2
    }
    return item.type !== 3
  }),
)

const stats = computed(() => {
  let directories = 0
  let menus = 0
  let buttons = 0
  const walk = (nodes: MenuNode[]) => {
    for (const node of nodes) {
      if (node.type === 1) directories += 1
      else if (node.type === 2) menus += 1
      else buttons += 1
      walk(node.children ?? [])
    }
  }
  walk(tree.value)
  return { directories, menus, buttons, total: directories + menus + buttons }
})

const treeData = computed(() => mapTreeNodes(filterTree(tree.value, keyword.value.trim())))

const selectedButtons = computed(() => {
  if (!selected.value || selected.value.type !== 2) {
    return []
  }
  return (selected.value.children ?? []).filter((child) => child.type === 3)
})

const selectedChildrenMenus = computed(() => {
  if (!selected.value || selected.value.type === 3) {
    return []
  }
  return (selected.value.children ?? []).filter((child) => child.type !== 3)
})

function filterTree(nodes: MenuNode[], query: string): MenuNode[] {
  if (!query) {
    return nodes
  }
  const lower = query.toLowerCase()
  const match = (node: MenuNode) =>
    node.name.toLowerCase().includes(lower) ||
    node.permission.toLowerCase().includes(lower) ||
    node.path.toLowerCase().includes(lower)

  return nodes.flatMap((node) => {
    const children = filterTree(node.children ?? [], query)
    if (match(node) || children.length > 0) {
      return [{ ...node, children }]
    }
    return []
  })
}

function mapTreeNodes(nodes: MenuNode[]): Array<MenuNode & { label: string; buttonCount: number }> {
  return nodes.map((node) => {
    const buttonCount = (node.children ?? []).filter((child) => child.type === 3).length
    return {
      ...node,
      label: node.name,
      buttonCount,
      children: mapTreeNodes(node.children ?? []),
    }
  })
}

function directoryIds(nodes: MenuNode[]): string[] {
  return nodes.flatMap((node) => {
    if (node.type !== 1) {
      return []
    }
    return [node.id, ...directoryIds(node.children ?? [])]
  })
}

function allExpandableIds(nodes: MenuNode[]): string[] {
  return nodes.flatMap((node) => {
    const children = node.children ?? []
    if (children.length === 0) {
      return []
    }
    return [node.id, ...allExpandableIds(children)]
  })
}

function findNode(nodes: MenuNode[], id: string): MenuNode | undefined {
  for (const node of nodes) {
    if (node.id === id) {
      return node
    }
    const child = findNode(node.children ?? [], id)
    if (child) {
      return child
    }
  }
  return undefined
}

function collectIds(node: MenuNode): string[] {
  return [node.id, ...(node.children ?? []).flatMap((child) => collectIds(child))]
}

function nodeType(data: unknown): MenuType {
  const node = data as MenuNode | undefined
  return (node?.type ?? 2) as MenuType
}

function nodeData(data: unknown): MenuNode & { buttonCount?: number } {
  return data as MenuNode & { buttonCount?: number }
}

function typeTheme(type: MenuType): 'success' | 'primary' | 'warning' {
  if (type === 1) return 'success'
  if (type === 2) return 'primary'
  return 'warning'
}

async function load() {
  pageStatus.value = 'loading'
  errorMessage.value = ''
  try {
    const response = await listMenus()
    tree.value = response.data ?? []
    expanded.value = directoryIds(tree.value)
    if (selectedId.value && !findNode(tree.value, selectedId.value)) {
      selectedId.value = ''
    }
    pageStatus.value = 'ready'
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法加载菜单'
    pageStatus.value = 'error'
  }
}

function onActive(value: Array<string | number>) {
  selectedId.value = String(value[0] ?? '')
}

function expandDirectories() {
  expanded.value = directoryIds(tree.value)
}

function expandAll() {
  expanded.value = allExpandableIds(tree.value)
}

function collapseAll() {
  expanded.value = []
}

function emptyForm(parentId = ''): typeof form.value {
  const parent = parentId ? findNode(tree.value, parentId) : undefined
  return {
    name: '',
    type: parent?.type === 2 ? 3 : 2,
    permission: '',
    parentId,
    sort: 0,
    path: '',
    icon: '#',
    component: '',
    componentName: '',
    visible: parent?.type === 2 ? false : true,
    keepAlive: true,
    alwaysShow: true,
    status: 0,
  }
}

function openCreate() {
  editing.value = null
  form.value = emptyForm(selectedId.value)
  editorVisible.value = true
}

function openEdit() {
  if (!selected.value) {
    feedback.warning('请先选择菜单')
    return
  }
  editing.value = selected.value
  form.value = {
    name: selected.value.name,
    type: selected.value.type,
    permission: selected.value.permission,
    parentId: selected.value.parent_id,
    sort: selected.value.sort,
    path: selected.value.path,
    icon: selected.value.icon || '#',
    component: selected.value.component,
    componentName: selected.value.component_name,
    visible: selected.value.visible,
    keepAlive: selected.value.keep_alive,
    alwaysShow: selected.value.always_show,
    status: selected.value.status,
  }
  editorVisible.value = true
}

async function save() {
  if (!form.value.name.trim()) {
    feedback.warning('请填写名称')
    return
  }
  saving.value = true
  try {
    const body = {
      name: form.value.name.trim(),
      type: form.value.type,
      permission: form.value.permission.trim(),
      sort: Number(form.value.sort) || 0,
      path: form.value.path.trim(),
      icon: form.value.icon.trim() || '#',
      component: form.value.component.trim(),
      component_name: form.value.componentName.trim(),
      visible: form.value.visible,
      keep_alive: form.value.keepAlive,
      always_show: form.value.alwaysShow,
    }
    if (editing.value) {
      await updateMenu(editing.value.id, {
        ...body,
        parent_id: form.value.parentId || null,
        status: form.value.status,
      })
      feedback.success('菜单已更新')
    } else {
      await createMenu({
        ...body,
        parent_id: form.value.parentId || undefined,
      })
      feedback.success('菜单已创建')
    }
    editorVisible.value = false
    await load()
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    saving.value = false
  }
}

function removeSelected() {
  if (!selected.value) {
    feedback.warning('请先选择菜单')
    return
  }
  const current = selected.value
  const dialog = DialogPlugin.confirm({
    header: '删除菜单',
    body: `确定删除「${current.name}」？`,
    onConfirm: async () => {
      try {
        await deleteMenu(current.id)
        feedback.success('菜单已删除')
        dialog.hide()
        selectedId.value = ''
        await load()
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
  <div class="menus-page">
    <div class="menus-toolbar">
      <div class="menus-toolbar__actions">
        <t-button v-if="canCreate" theme="primary" @click="openCreate">
          <template #icon><AddIcon /></template>
          新建菜单
        </t-button>
        <t-button v-if="canUpdate" theme="default" variant="outline" :disabled="!selected" @click="openEdit">
          <template #icon><EditIcon /></template>
          编辑
        </t-button>
        <t-button v-if="canDelete" theme="danger" variant="outline" :disabled="!selected" @click="removeSelected">
          <template #icon><DeleteIcon /></template>
          删除
        </t-button>
        <t-button theme="default" variant="text" @click="load">
          <template #icon><RefreshIcon /></template>
          刷新
        </t-button>
      </div>
      <div class="menus-toolbar__stats">
        <span>共 {{ stats.total }} 项</span>
        <t-tag size="small" theme="success" variant="light">目录 {{ stats.directories }}</t-tag>
        <t-tag size="small" theme="primary" variant="light">菜单 {{ stats.menus }}</t-tag>
        <t-tag size="small" theme="warning" variant="light">按钮 {{ stats.buttons }}</t-tag>
      </div>
    </div>

    <PageStatus :status="pageStatus" :error-message="errorMessage" description="还没有菜单" @retry="load">
      <div class="menus-layout">
        <t-card class="menus-panel menus-panel--tree" title="菜单树" :bordered="false">
          <template #actions>
            <div class="tree-actions">
              <t-button size="small" variant="text" @click="expandDirectories">仅目录</t-button>
              <t-button size="small" variant="text" @click="expandAll">全部展开</t-button>
              <t-button size="small" variant="text" @click="collapseAll">全部折叠</t-button>
            </div>
          </template>

          <t-input v-model="keyword" class="tree-search" clearable placeholder="搜索名称 / 权限 / 路由">
            <template #prefix-icon><SearchIcon /></template>
          </t-input>

          <t-tree
            v-if="treeData.length > 0"
            v-model:expanded="expanded"
            class="menu-tree"
            :data="treeData"
            :keys="{ value: 'id', label: 'label', children: 'children' }"
            :actived="selectedId ? [selectedId] : []"
            activable
            hover
            line
            transition
            @active="onActive"
          >
            <template #label="{ node }">
              <span class="menu-node" :class="`menu-node--type-${nodeType(node.data)}`">
                <span class="menu-node__icon" aria-hidden="true">
                  <FolderIcon v-if="nodeType(node.data) === 1" />
                  <RootListIcon v-else-if="nodeType(node.data) === 2" />
                  <KeyIcon v-else />
                </span>
                <span class="menu-node__main">
                  <span class="menu-node__name">{{ nodeData(node.data).name }}</span>
                  <span v-if="nodeType(node.data) === 2 && (nodeData(node.data).buttonCount ?? 0) > 0" class="menu-node__meta">
                    {{ nodeData(node.data).buttonCount }} 个按钮
                  </span>
                  <span v-else-if="nodeType(node.data) === 3 && nodeData(node.data).permission" class="menu-node__meta">
                    {{ nodeData(node.data).permission }}
                  </span>
                </span>
                <t-tag size="small" :theme="typeTheme(nodeType(node.data))" variant="light">
                  {{ menuTypeLabel(nodeType(node.data)) }}
                </t-tag>
              </span>
            </template>
          </t-tree>
          <div v-else class="menus-empty">
            <p>{{ keyword ? '没有匹配的菜单' : '还没有菜单' }}</p>
          </div>
        </t-card>

        <t-card class="menus-panel menus-panel--detail" title="节点详情" :bordered="false">
          <div v-if="selected" class="detail">
            <div class="detail__hero">
              <span class="detail__hero-icon" :class="`detail__hero-icon--type-${selected.type}`">
                <FolderIcon v-if="selected.type === 1" size="28px" />
                <RootListIcon v-else-if="selected.type === 2" size="28px" />
                <KeyIcon v-else size="28px" />
              </span>
              <div class="detail__hero-text">
                <h2>{{ selected.name }}</h2>
                <div class="detail__hero-tags">
                  <t-tag size="small" :theme="typeTheme(selected.type)" variant="light">
                    {{ menuTypeLabel(selected.type) }}
                  </t-tag>
                  <t-tag size="small" :theme="selected.status === 0 ? 'success' : 'danger'" variant="light">
                    {{ statusLabel(selected.status) }}
                  </t-tag>
                  <t-tag v-if="selected.type !== 3" size="small" variant="outline">
                    {{ selected.visible ? '可见' : '隐藏' }}
                  </t-tag>
                </div>
              </div>
            </div>

            <t-descriptions :column="1" size="medium" item-layout="horizontal">
              <t-descriptions-item label="排序">{{ selected.sort }}</t-descriptions-item>
              <t-descriptions-item v-if="selected.type !== 3" label="路由">
                {{ selected.path || '—' }}
              </t-descriptions-item>
              <t-descriptions-item label="权限标识">
                {{ selected.permission || '—' }}
              </t-descriptions-item>
              <t-descriptions-item v-if="selected.type !== 3" label="图标">
                {{ selected.icon || '—' }}
              </t-descriptions-item>
              <t-descriptions-item v-if="selected.type === 2" label="组件路径">
                {{ selected.component || '—' }}
              </t-descriptions-item>
              <t-descriptions-item v-if="selected.type === 2" label="组件名">
                {{ selected.component_name || '—' }}
              </t-descriptions-item>
              <t-descriptions-item v-if="selected.type === 2" label="缓存">
                {{ selected.keep_alive ? '是' : '否' }}
              </t-descriptions-item>
              <t-descriptions-item v-if="selected.type === 1" label="总是显示">
                {{ selected.always_show ? '是' : '否' }}
              </t-descriptions-item>
            </t-descriptions>

            <div v-if="selectedButtons.length > 0" class="detail__section">
              <h3>按钮权限</h3>
              <div class="detail__chips">
                <t-tag v-for="btn in selectedButtons" :key="btn.id" theme="warning" variant="light">
                  {{ btn.name }} · {{ btn.permission }}
                </t-tag>
              </div>
            </div>

            <div v-if="selectedChildrenMenus.length > 0" class="detail__section">
              <h3>下级节点</h3>
              <div class="detail__chips">
                <t-tag
                  v-for="child in selectedChildrenMenus"
                  :key="child.id"
                  :theme="typeTheme(child.type)"
                  variant="light"
                >
                  {{ child.name }}
                </t-tag>
              </div>
            </div>

            <div class="detail__actions">
              <t-button v-if="canUpdate" theme="primary" variant="outline" @click="openEdit">
                <template #icon><EditIcon /></template>
                编辑此节点
              </t-button>
              <t-button v-if="canCreate" theme="default" variant="outline" @click="openCreate">
                <template #icon><AddIcon /></template>
                在此新建
              </t-button>
            </div>
          </div>
          <div v-else class="menus-empty menus-empty--detail">
            <RootListIcon size="40px" />
            <p>从左侧选择一个节点查看详情</p>
            <span>目录默认展开，按钮权限折叠在菜单节点下</span>
          </div>
        </t-card>
      </div>
    </PageStatus>

    <t-dialog
      v-model:visible="editorVisible"
      :header="editing ? '编辑菜单' : '新建菜单'"
      :confirm-btn="{ content: '保存', loading: saving }"
      width="560px"
      @confirm="save"
    >
      <t-form label-align="top">
        <t-form-item label="名称">
          <t-input v-model="form.name" maxlength="50" />
        </t-form-item>
        <t-form-item label="类型">
          <t-radio-group v-model="form.type">
            <t-radio :value="1">目录</t-radio>
            <t-radio :value="2">菜单</t-radio>
            <t-radio :value="3">按钮</t-radio>
          </t-radio-group>
        </t-form-item>
        <t-form-item label="权限标识">
          <t-input v-model="form.permission" maxlength="100" />
        </t-form-item>
        <t-form-item label="上级菜单">
          <t-select
            v-model="form.parentId"
            clearable
            placeholder="不选则为根菜单"
            :options="parentOptions.map((item) => ({ label: item.label, value: item.id }))"
          />
        </t-form-item>
        <t-form-item label="排序">
          <t-input-number v-model="form.sort" :min="0" :max="1000000" />
        </t-form-item>
        <t-form-item v-if="form.type !== 3" label="路由地址">
          <t-input v-model="form.path" maxlength="200" />
        </t-form-item>
        <t-form-item v-if="form.type !== 3" label="图标">
          <t-input v-model="form.icon" maxlength="100" placeholder="侧边栏图标，可用 # 占位" />
        </t-form-item>
        <t-form-item v-if="form.type === 2" label="组件路径">
          <t-input v-model="form.component" maxlength="255" />
        </t-form-item>
        <t-form-item v-if="form.type === 2" label="组件名">
          <t-input v-model="form.componentName" maxlength="255" />
        </t-form-item>
        <t-form-item v-if="form.type !== 3" label="可见">
          <t-switch v-model="form.visible" />
        </t-form-item>
        <t-form-item v-if="form.type === 2" label="缓存">
          <t-switch v-model="form.keepAlive" />
        </t-form-item>
        <t-form-item v-if="form.type === 1" label="总是显示">
          <t-switch v-model="form.alwaysShow" />
        </t-form-item>
        <t-form-item v-if="editing" label="状态">
          <t-radio-group v-model="form.status">
            <t-radio :value="0">正常</t-radio>
            <t-radio :value="1">停用</t-radio>
          </t-radio-group>
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<style scoped>
.menus-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: calc(100vh - 148px);
}

.menus-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 20px;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-radius: 10px;
  background: var(--td-bg-color-container);
  box-shadow: var(--td-shadow-1);
}

.menus-toolbar__actions,
.menus-toolbar__stats {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.menus-toolbar__stats {
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.menus-layout {
  display: grid;
  grid-template-columns: minmax(420px, 1.15fr) minmax(320px, 0.85fr);
  gap: 16px;
  align-items: stretch;
  min-height: calc(100vh - 220px);
}

.menus-panel {
  display: flex;
  flex-direction: column;
  min-height: 100%;
  height: 100%;
  box-shadow: var(--td-shadow-1);
}

.menus-panel :deep(.t-card__body) {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}

.tree-actions {
  display: flex;
  gap: 2px;
}

.tree-search {
  margin-bottom: 12px;
}

.menu-tree {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding-right: 4px;
}

.menu-tree :deep(.t-tree__item) {
  padding-top: 4px;
  padding-bottom: 4px;
}

.menu-tree :deep(.t-tree__label) {
  width: 100%;
  margin-left: 4px;
  padding: 6px 10px;
  border-radius: 8px;
}

.menu-tree :deep(.t-is-active > .t-tree__label),
.menu-tree :deep(.t-tree__item--active > .t-tree__label) {
  background: var(--td-brand-color-light);
}

.menu-node {
  display: flex;
  gap: 10px;
  align-items: center;
  width: 100%;
  min-width: 0;
}

.menu-node__icon {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
  font-size: 16px;
}

.menu-node--type-1 .menu-node__icon {
  background: var(--qz-success-light);
  color: var(--qz-success);
}

.menu-node--type-2 .menu-node__icon {
  background: var(--qz-brand-light);
  color: var(--qz-brand);
}

.menu-node--type-3 .menu-node__icon {
  background: var(--qz-warning-light);
  color: var(--qz-warning);
}

.menu-node__main {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.menu-node__name {
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 500;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-node__meta {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 20px;
}

.detail__hero {
  display: flex;
  gap: 14px;
  align-items: center;
  padding: 16px;
  border-radius: var(--qz-radius);
  background: var(--qz-brand-light);
}

.detail__hero-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border-radius: 14px;
  background: var(--td-bg-color-container);
  color: var(--td-brand-color);
  box-shadow: var(--td-shadow-1);
}

.detail__hero-icon--type-1 {
  color: var(--qz-success);
}

.detail__hero-icon--type-2 {
  color: var(--qz-brand);
}

.detail__hero-icon--type-3 {
  color: var(--qz-warning);
}

.detail__hero-text h2 {
  margin: 0 0 8px;
  font-size: 20px;
  font-weight: 600;
  line-height: 1.3;
}

.detail__hero-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.detail__section h3 {
  margin: 0 0 10px;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 600;
}

.detail__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.detail__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: auto;
  padding-top: 8px;
}

.menus-empty {
  display: grid;
  gap: 8px;
  place-content: center;
  min-height: 180px;
  color: var(--td-text-color-secondary);
  text-align: center;
}

.menus-empty--detail {
  flex: 1;
  justify-items: center;
  color: var(--td-text-color-placeholder);
}

.menus-empty p {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 14px;
}

.menus-empty span {
  font-size: 12px;
}

@media (max-width: 1100px) {
  .menus-layout {
    grid-template-columns: 1fr;
    min-height: auto;
  }

  .menus-panel {
    min-height: 420px;
  }
}
</style>
