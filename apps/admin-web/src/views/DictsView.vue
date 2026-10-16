<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import { AddIcon, DeleteIcon, EditIcon, SearchIcon } from 'tdesign-icons-vue-next'
import PageStatus from '@/components/PageStatus.vue'
import { statusLabel, type RecordStatus } from '@/api/identity'
import {
  colorTypeOptions,
  createDictData,
  createDictType,
  deleteDictData,
  deleteDictType,
  listDictData,
  listDictTypes,
  updateDictData,
  updateDictType,
  type DictData,
  type DictType,
} from '@/api/dict'
import { ApiError } from '@/utils/http'
import { feedback } from '@/utils/feedback'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const canCreate = computed(() => auth.hasPermission('system:dict:create'))
const canUpdate = computed(() => auth.hasPermission('system:dict:update'))
const canDelete = computed(() => auth.hasPermission('system:dict:delete'))

const typeStatus = ref<'loading' | 'error' | 'ready'>('loading')
const typeError = ref('')
const types = ref<DictType[]>([])
const typePage = ref(1)
const typePageSize = ref(20)
const typeTotal = ref(0)
const typeKeyword = ref('')
const selectedId = ref('')

const dataStatus = ref<'loading' | 'error' | 'ready'>('ready')
const dataError = ref('')
const items = ref<DictData[]>([])
const dataPage = ref(1)
const dataPageSize = ref(20)
const dataTotal = ref(0)
const dataKeyword = ref('')

const typeEditorVisible = ref(false)
const typeSaving = ref(false)
const editingType = ref<DictType | null>(null)
const typeForm = ref({ name: '', type: '', remark: '', status: 0 as RecordStatus })

const dataEditorVisible = ref(false)
const dataSaving = ref(false)
const editingData = ref<DictData | null>(null)
const dataForm = ref({
  label: '',
  value: '',
  sort: 0,
  colorType: '',
  cssClass: '',
  remark: '',
  status: 0 as RecordStatus,
})

const selected = computed(() => types.value.find((item) => item.id === selectedId.value))

const typeColumns = [
  { colKey: 'name', title: '名称', ellipsis: true },
  { colKey: 'type', title: '类型', ellipsis: true },
  { colKey: 'status', title: '状态', width: 88 },
  { colKey: 'actions', title: '操作', width: 140 },
]

const dataColumns = [
  { colKey: 'label', title: '标签', ellipsis: true },
  { colKey: 'value', title: '键值', width: 120 },
  { colKey: 'sort', title: '排序', width: 72 },
  { colKey: 'color_type', title: '颜色', width: 100 },
  { colKey: 'status', title: '状态', width: 88 },
  { colKey: 'actions', title: '操作', width: 140 },
]

function tagTheme(color: string): 'default' | 'primary' | 'success' | 'warning' | 'danger' {
  if (color === 'primary' || color === 'success' || color === 'warning' || color === 'danger') {
    return color
  }
  return 'default'
}

function colorLabel(color: string) {
  return colorTypeOptions.find((item) => item.value === color)?.label || color || '—'
}

async function loadTypes() {
  typeStatus.value = 'loading'
  typeError.value = ''
  try {
    const response = await listDictTypes({
      page: typePage.value,
      page_size: typePageSize.value,
      keyword: typeKeyword.value.trim() || undefined,
    })
    types.value = response.data.items
    typeTotal.value = response.data.total
    if (selectedId.value && !types.value.some((item) => item.id === selectedId.value)) {
      selectedId.value = types.value[0]?.id ?? ''
    } else if (!selectedId.value && types.value.length > 0) {
      selectedId.value = types.value[0].id
    }
    typeStatus.value = 'ready'
    if (selected.value) {
      await loadData()
    } else {
      items.value = []
      dataTotal.value = 0
      dataStatus.value = 'ready'
    }
  } catch (error) {
    typeError.value = error instanceof ApiError ? error.message : '无法加载字典类型'
    typeStatus.value = 'error'
  }
}

async function loadData() {
  if (!selected.value) {
    items.value = []
    dataTotal.value = 0
    dataStatus.value = 'ready'
    return
  }
  dataStatus.value = 'loading'
  dataError.value = ''
  try {
    const response = await listDictData(selected.value.type, {
      page: dataPage.value,
      page_size: dataPageSize.value,
      keyword: dataKeyword.value.trim() || undefined,
    })
    items.value = response.data.items
    dataTotal.value = response.data.total
    dataStatus.value = 'ready'
  } catch (error) {
    dataError.value = error instanceof ApiError ? error.message : '无法加载字典数据'
    dataStatus.value = 'error'
  }
}

function searchTypes() {
  typePage.value = 1
  void loadTypes()
}

function searchData() {
  dataPage.value = 1
  void loadData()
}

function typeRowClass(params: { row: DictType }) {
  return params.row.id === selectedId.value ? 'dict-row--active' : ''
}

function onTypeRow(context: { row: DictType }) {
  selectType(context.row)
}

function selectType(row: DictType) {
  if (selectedId.value === row.id) {
    return
  }
  selectedId.value = row.id
  dataPage.value = 1
  dataKeyword.value = ''
  void loadData()
}

function openCreateType() {
  editingType.value = null
  typeForm.value = { name: '', type: '', remark: '', status: 0 }
  typeEditorVisible.value = true
}

function openEditType(row: DictType) {
  editingType.value = row
  typeForm.value = { name: row.name, type: row.type, remark: row.remark, status: row.status }
  typeEditorVisible.value = true
}

async function saveType() {
  if (!typeForm.value.name.trim() || !typeForm.value.type.trim()) {
    feedback.warning('请填写名称和类型')
    return
  }
  typeSaving.value = true
  try {
    if (editingType.value) {
      await updateDictType(editingType.value.id, {
        name: typeForm.value.name.trim(),
        type: typeForm.value.type.trim(),
        remark: typeForm.value.remark.trim(),
        status: typeForm.value.status,
      })
      feedback.success('字典类型已更新')
    } else {
      const response = await createDictType({
        name: typeForm.value.name.trim(),
        type: typeForm.value.type.trim(),
        remark: typeForm.value.remark.trim() || undefined,
        status: typeForm.value.status,
      })
      selectedId.value = response.data.id
      feedback.success('字典类型已创建')
    }
    typeEditorVisible.value = false
    await loadTypes()
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    typeSaving.value = false
  }
}

function removeType(row: DictType) {
  const dialog = DialogPlugin.confirm({
    header: '删除字典类型',
    body: `确定删除「${row.name}」？类型下仍有数据时不能删除。`,
    onConfirm: async () => {
      try {
        await deleteDictType(row.id)
        feedback.success('字典类型已删除')
        dialog.hide()
        if (selectedId.value === row.id) {
          selectedId.value = ''
        }
        await loadTypes()
      } catch {
        dialog.hide()
      }
    },
  })
}

function openCreateData() {
  if (!selected.value) {
    feedback.warning('请先选择字典类型')
    return
  }
  editingData.value = null
  dataForm.value = { label: '', value: '', sort: 0, colorType: '', cssClass: '', remark: '', status: 0 }
  dataEditorVisible.value = true
}

function openEditData(row: DictData) {
  editingData.value = row
  dataForm.value = {
    label: row.label,
    value: row.value,
    sort: row.sort,
    colorType: row.color_type,
    cssClass: row.css_class,
    remark: row.remark,
    status: row.status,
  }
  dataEditorVisible.value = true
}

async function saveData() {
  if (!selected.value || !dataForm.value.label.trim() || !dataForm.value.value.trim()) {
    feedback.warning('请填写标签和键值')
    return
  }
  dataSaving.value = true
  try {
    const body = {
      label: dataForm.value.label.trim(),
      value: dataForm.value.value.trim(),
      sort: Number(dataForm.value.sort) || 0,
      color_type: dataForm.value.colorType || '',
      css_class: dataForm.value.cssClass.trim(),
      remark: dataForm.value.remark.trim(),
      status: dataForm.value.status,
    }
    if (editingData.value) {
      await updateDictData(editingData.value.id, body)
      feedback.success('字典数据已更新')
    } else {
      await createDictData({ ...body, dict_type: selected.value.type })
      feedback.success('字典数据已创建')
    }
    dataEditorVisible.value = false
    await loadData()
  } catch {
    // The HTTP client already shows the server message.
  } finally {
    dataSaving.value = false
  }
}

function removeData(row: DictData) {
  const dialog = DialogPlugin.confirm({
    header: '删除字典数据',
    body: `确定删除「${row.label}」？`,
    onConfirm: async () => {
      try {
        await deleteDictData(row.id)
        feedback.success('字典数据已删除')
        dialog.hide()
        await loadData()
      } catch {
        dialog.hide()
      }
    },
  })
}

onMounted(() => {
  void loadTypes()
})
</script>

<template>
  <div class="dicts-page">
    <div class="dicts-layout">
      <t-card class="dicts-panel" title="字典类型" :bordered="false">
        <div class="toolbar">
          <t-input v-model="typeKeyword" clearable placeholder="搜索名称或类型" @enter="searchTypes">
            <template #prefix-icon><SearchIcon /></template>
          </t-input>
          <t-button theme="default" @click="searchTypes">查询</t-button>
          <t-button v-if="canCreate" theme="primary" @click="openCreateType">
            <template #icon><AddIcon /></template>
            新建类型
          </t-button>
        </div>
        <PageStatus :status="typeStatus" :error-message="typeError" description="还没有字典类型" @retry="loadTypes">
          <t-table
            row-key="id"
            :data="types"
            :columns="typeColumns"
            hover
            :row-class-name="typeRowClass"
            @row-click="onTypeRow"
          >
            <template #status="{ row }">
              <t-tag :theme="row.status === 0 ? 'success' : 'warning'" variant="light" size="small">
                {{ statusLabel(row.status) }}
              </t-tag>
            </template>
            <template #actions="{ row }">
              <t-button v-if="canUpdate" theme="primary" variant="text" @click.stop="openEditType(row)">
                <template #icon><EditIcon /></template>
                编辑
              </t-button>
              <t-button v-if="canDelete" theme="danger" variant="text" @click.stop="removeType(row)">
                <template #icon><DeleteIcon /></template>
                删除
              </t-button>
            </template>
          </t-table>
          <t-pagination
            v-model="typePage"
            v-model:page-size="typePageSize"
            class="pager"
            :total="typeTotal"
            @change="loadTypes"
          />
        </PageStatus>
      </t-card>

      <t-card class="dicts-panel" :title="selected ? `${selected.name} · ${selected.type}` : '字典数据'" :bordered="false">
        <div class="toolbar">
          <t-input
            v-model="dataKeyword"
            clearable
            :disabled="!selected"
            placeholder="搜索标签或键值"
            @enter="searchData"
          >
            <template #prefix-icon><SearchIcon /></template>
          </t-input>
          <t-button theme="default" :disabled="!selected" @click="searchData">查询</t-button>
          <t-button v-if="canCreate" theme="primary" :disabled="!selected" @click="openCreateData">
            <template #icon><AddIcon /></template>
            新建数据
          </t-button>
        </div>
        <div v-if="!selected" class="dicts-empty">
          <p>从左侧选择一个字典类型</p>
          <span>同一类型下的键值不能重复</span>
        </div>
        <PageStatus
          v-else
          :status="dataStatus"
          :error-message="dataError"
          description="这个类型还没有数据"
          @retry="loadData"
        >
          <t-table row-key="id" :data="items" :columns="dataColumns" hover>
            <template #color_type="{ row }">
              <t-tag v-if="row.color_type" :theme="tagTheme(row.color_type)" variant="light" size="small">
                {{ colorLabel(row.color_type) }}
              </t-tag>
              <span v-else>—</span>
            </template>
            <template #status="{ row }">
              <t-tag :theme="row.status === 0 ? 'success' : 'warning'" variant="light" size="small">
                {{ statusLabel(row.status) }}
              </t-tag>
            </template>
            <template #actions="{ row }">
              <t-button v-if="canUpdate" theme="primary" variant="text" @click="openEditData(row)">
                <template #icon><EditIcon /></template>
                编辑
              </t-button>
              <t-button v-if="canDelete" theme="danger" variant="text" @click="removeData(row)">
                <template #icon><DeleteIcon /></template>
                删除
              </t-button>
            </template>
          </t-table>
          <t-pagination
            v-model="dataPage"
            v-model:page-size="dataPageSize"
            class="pager"
            :total="dataTotal"
            @change="loadData"
          />
        </PageStatus>
      </t-card>
    </div>

    <t-dialog
      v-model:visible="typeEditorVisible"
      :header="editingType ? '编辑字典类型' : '新建字典类型'"
      :confirm-btn="{ content: '保存', loading: typeSaving }"
      width="480px"
      @confirm="saveType"
    >
      <t-form label-align="top">
        <t-form-item label="名称">
          <t-input v-model="typeForm.name" maxlength="100" />
        </t-form-item>
        <t-form-item label="类型">
          <t-input v-model="typeForm.type" maxlength="100" placeholder="字母、数字或下划线，如 common_status" />
        </t-form-item>
        <t-form-item label="备注">
          <t-textarea v-model="typeForm.remark" maxlength="500" />
        </t-form-item>
        <t-form-item label="状态">
          <t-radio-group v-model="typeForm.status">
            <t-radio :value="0">正常</t-radio>
            <t-radio :value="1">停用</t-radio>
          </t-radio-group>
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog
      v-model:visible="dataEditorVisible"
      :header="editingData ? '编辑字典数据' : '新建字典数据'"
      :confirm-btn="{ content: '保存', loading: dataSaving }"
      width="480px"
      @confirm="saveData"
    >
      <t-form label-align="top">
        <t-form-item label="标签">
          <t-input v-model="dataForm.label" maxlength="100" />
        </t-form-item>
        <t-form-item label="键值">
          <t-input v-model="dataForm.value" maxlength="100" placeholder="字母、数字、下划线、点或中划线" />
        </t-form-item>
        <t-form-item label="排序">
          <t-input-number v-model="dataForm.sort" :min="0" :max="1000000" />
        </t-form-item>
        <t-form-item label="颜色类型">
          <t-select v-model="dataForm.colorType" clearable :options="colorTypeOptions" placeholder="可不选" />
        </t-form-item>
        <t-form-item label="CSS 样式">
          <t-input v-model="dataForm.cssClass" maxlength="100" />
        </t-form-item>
        <t-form-item label="备注">
          <t-textarea v-model="dataForm.remark" maxlength="500" />
        </t-form-item>
        <t-form-item label="状态">
          <t-radio-group v-model="dataForm.status">
            <t-radio :value="0">正常</t-radio>
            <t-radio :value="1">停用</t-radio>
          </t-radio-group>
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<style scoped>
.dicts-page {
  min-height: calc(100vh - 148px);
}

.dicts-layout {
  display: grid;
  grid-template-columns: minmax(420px, 0.9fr) minmax(480px, 1.1fr);
  gap: 16px;
  align-items: stretch;
  min-height: calc(100vh - 148px);
}

.dicts-panel {
  height: 100%;
  box-shadow: var(--td-shadow-1);
}

.dicts-panel :deep(.t-card__body) {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.dicts-panel :deep(.dict-row--active) {
  background: var(--td-brand-color-light);
}

.dicts-empty {
  display: grid;
  flex: 1;
  gap: 8px;
  place-content: center;
  min-height: 240px;
  color: var(--td-text-color-secondary);
  text-align: center;
}

.dicts-empty p {
  margin: 0;
  font-size: 14px;
}

.dicts-empty span {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

@media (max-width: 1100px) {
  .dicts-layout {
    grid-template-columns: 1fr;
    min-height: auto;
  }
}
</style>
